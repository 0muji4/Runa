package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/push"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/schedule"
)

// Reminder copy; both OSes display what the payload carries.
const (
	ReminderKind  = "diary_reminder"
	ReminderTitle = "月が出ました"
	ReminderBody  = "今日を、そっと綴りませんか。"
)

// ReminderCallbackPath is where the scheduler calls back; the handler mounts it.
const ReminderCallbackPath = "/api/v1/hooks/push/reminder"

// ErrPushTransient asks the handler to answer 503 so the scheduler retries.
var ErrPushTransient = errors.New("service: push delivery failed transiently")

// PushConfig tunes the reminder chain.
type PushConfig struct {
	Enabled bool // kill switch
	// RetryWindow is the provider TTL and the cut-off for a late callback.
	RetryWindow       time.Duration
	StalePendingAfter time.Duration
	MaxAttempts       int
	// DeviceMaxAge ages out devices a forced sign-out could not unregister.
	DeviceMaxAge time.Duration
}

// DefaultPushConfig is the production tuning.
func DefaultPushConfig() PushConfig {
	return PushConfig{
		Enabled:           true,
		RetryWindow:       time.Hour,
		StalePendingAfter: 5 * time.Minute,
		MaxAttempts:       3,
		DeviceMaxAge:      45 * 24 * time.Hour,
	}
}

// ReminderTask is the callback body; Deliver compares it with the current row.
type ReminderTask struct {
	DeviceID   string `json:"device_id"`
	LocalDate  string `json:"local_date"`
	NotifyTime string `json:"notify_time"`
	TimeZone   string `json:"time_zone"`
}

// Outcome says what Deliver did; every value answers the scheduler with 200.
type Outcome string

const (
	OutcomeSent           Outcome = "sent"
	OutcomeSkippedWritten Outcome = "skipped_written"
	OutcomeStale          Outcome = "stale"
	OutcomeExpired        Outcome = "expired"
	OutcomeAlreadyClaimed Outcome = "already_claimed"
	OutcomeInvalidToken   Outcome = "invalid_token"
	OutcomeDisabled       Outcome = "disabled"
	OutcomeFailed         Outcome = "failed"
)

// PushService owns the nightly-reminder chain: schedule the next callback,
// then decide on arrival whether the device gets a reminder.
type PushService struct {
	devices   repository.DeviceStore
	diaries   repository.DiaryStore
	senders   map[string]push.Sender // by platform; a missing platform is disabled
	scheduler schedule.Scheduler     // nil disables scheduling
	cfg       PushConfig
	now       func() time.Time
	logger    *slog.Logger
}

// PushOption customises NewPushService.
type PushOption func(*PushService)

// WithPushLogger sets the logger for delivery outcomes.
func WithPushLogger(logger *slog.Logger) PushOption {
	return func(s *PushService) { s.logger = logger }
}

// NewPushService constructs the service, defaulting now to time.Now.
func NewPushService(devices repository.DeviceStore, diaries repository.DiaryStore, senders map[string]push.Sender,
	scheduler schedule.Scheduler, cfg PushConfig, now func() time.Time, opts ...PushOption) *PushService {
	if now == nil {
		now = time.Now
	}
	if senders == nil {
		senders = map[string]push.Sender{}
	}
	s := &PushService{devices: devices, diaries: diaries, senders: senders, scheduler: scheduler, cfg: cfg, now: now, logger: slog.Default()}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ScheduleNext (re)schedules the device's next reminder callback, or cancels
// the pending one when the device no longer qualifies.
func (s *PushService) ScheduleNext(ctx context.Context, d repository.Device) error {
	return s.scheduleFrom(ctx, d, s.now(), "")
}

// scheduleFrom schedules the first occurrence after from. executing is the
// task currently calling back (if any): it is never cancelled, and the new task
// is created before the old one is dropped so a failure here leaves a task behind.
func (s *PushService) scheduleFrom(ctx context.Context, d repository.Device, from time.Time, executing string) error {
	if s.scheduler == nil {
		return nil
	}
	if _, hasSender := s.senders[d.Platform]; !s.cfg.Enabled || !d.Enabled || d.TokenInvalidAt != nil || !hasSender {
		s.cancelPending(ctx, d, executing)
		return nil
	}
	loc, err := time.LoadLocation(d.TimeZone)
	if err != nil {
		return fmt.Errorf("schedule reminder: zone %q: %w", d.TimeZone, err)
	}
	fireAt, localDate, err := nextFireAt(from, d.NotifyTime, loc)
	if err != nil {
		return fmt.Errorf("schedule reminder: %w", err)
	}
	name := reminderTaskName(d, localDate)
	body, err := json.Marshal(ReminderTask{DeviceID: d.ID, LocalDate: localDate, NotifyTime: d.NotifyTime, TimeZone: d.TimeZone})
	if err != nil {
		return fmt.Errorf("schedule reminder: encode: %w", err)
	}
	if err := s.scheduler.Schedule(ctx, schedule.Task{Name: name, RunAt: fireAt, Path: ReminderCallbackPath, Body: body}); err != nil {
		return fmt.Errorf("schedule reminder: %w", err)
	}
	if err := s.devices.SetNextTask(ctx, d.ID, name, fireAt); err != nil {
		return fmt.Errorf("schedule reminder: %w", err)
	}
	if d.NextTaskName != nil && *d.NextTaskName != name {
		s.cancelPending(ctx, d, executing)
	}
	return nil
}

// CancelScheduled drops the pending callback; a failure is only logged since
// Deliver ignores a callback whose row is gone.
func (s *PushService) CancelScheduled(ctx context.Context, d repository.Device) {
	if s.scheduler == nil {
		return
	}
	s.cancelPending(ctx, d, "")
}

func (s *PushService) cancelPending(ctx context.Context, d repository.Device, executing string) {
	if d.NextTaskName == nil || *d.NextTaskName == executing {
		return
	}
	if err := s.scheduler.Cancel(ctx, *d.NextTaskName); err != nil {
		s.logger.WarnContext(ctx, "reminder task cancel failed", slog.String("device_id", d.ID), slog.String("task", *d.NextTaskName), slog.Any("error", err))
	}
}

// Deliver handles one callback; only ErrPushTransient asks the scheduler to retry.
func (s *PushService) Deliver(ctx context.Context, task ReminderTask) (Outcome, error) {
	if !s.cfg.Enabled {
		return OutcomeDisabled, nil
	}
	d, err := s.devices.GetDevice(ctx, task.DeviceID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return OutcomeStale, nil
		}
		return "", fmt.Errorf("%w: load device: %w", ErrPushTransient, err)
	}
	now := s.now()
	switch {
	case !d.Enabled, d.NotifyTime != task.NotifyTime, d.TimeZone != task.TimeZone:
		// The row moved on since the task was scheduled; the newer task carries the truth.
		return OutcomeStale, nil
	case d.TokenInvalidAt != nil:
		return OutcomeInvalidToken, nil
	case s.cfg.DeviceMaxAge > 0 && now.Sub(d.UpdatedAt) > s.cfg.DeviceMaxAge:
		return OutcomeStale, nil
	}
	loc, err := time.LoadLocation(d.TimeZone)
	if err != nil {
		return OutcomeStale, nil
	}
	fireAt, err := localInstant(task.LocalDate, d.NotifyTime, loc)
	if err != nil {
		return OutcomeStale, nil
	}
	executing := reminderTaskName(d, task.LocalDate)
	if now.After(fireAt.Add(s.cfg.RetryWindow)) {
		// The queue should have given up by now; never deliver a stale night's reminder.
		return OutcomeExpired, s.scheduleAfter(ctx, d, fireAt, executing)
	}
	sender, ok := s.senders[d.Platform]
	if !ok {
		return OutcomeDisabled, nil
	}

	claim, err := s.devices.ClaimReminder(ctx, repository.ClaimReminderParams{
		DeviceID: d.ID, LocalDate: task.LocalDate, Now: now,
		StalePendingAfter: s.cfg.StalePendingAfter, MaxAttempts: s.cfg.MaxAttempts,
	})
	if err != nil {
		return "", fmt.Errorf("%w: claim: %w", ErrPushTransient, err)
	}
	switch claim {
	case repository.ClaimHeld:
		// Another attempt may still be mid-send; a 503 brings the scheduler back after it went stale.
		return "", fmt.Errorf("%w: attempt for %s still pending", ErrPushTransient, task.LocalDate)
	case repository.ClaimDone:
		return OutcomeAlreadyClaimed, s.scheduleAfter(ctx, d, fireAt, executing)
	}

	dayStart, _ := time.ParseInLocation("2006-01-02", task.LocalDate, loc)
	counts, err := s.diaries.CountByLocalDate(ctx, d.UserID, dayStart, dayStart.AddDate(0, 0, 1), loc)
	if err != nil {
		s.finish(ctx, d, repository.ReminderFailed, err)
		return "", fmt.Errorf("%w: count entries: %w", ErrPushTransient, err)
	}
	if counts[task.LocalDate] > 0 {
		s.finish(ctx, d, repository.ReminderSkipped, nil)
		s.log(ctx, d, task, OutcomeSkippedWritten, "")
		return OutcomeSkippedWritten, s.scheduleAfter(ctx, d, fireAt, executing)
	}

	res, err := sender.Send(ctx, d.PushToken, push.Notification{
		Title: ReminderTitle,
		Body:  ReminderBody,
		Data: map[string]string{
			"kind": ReminderKind, "date": task.LocalDate,
			"title": ReminderTitle, "body": ReminderBody,
		},
		CollapseID: ReminderKind,
		Expiry:     fireAt.Add(s.cfg.RetryWindow),
	})
	switch {
	case err == nil:
		s.finish(ctx, d, repository.ReminderSent, nil)
		s.log(ctx, d, task, OutcomeSent, res.ProviderMessageID)
		return OutcomeSent, s.scheduleAfter(ctx, d, fireAt, executing)
	case errors.Is(err, push.ErrInvalidToken):
		s.finish(ctx, d, repository.ReminderFailed, err)
		if mErr := s.devices.MarkTokenInvalid(ctx, d.ID, now); mErr != nil {
			s.logger.ErrorContext(ctx, "mark token invalid failed", slog.String("device_id", d.ID), slog.Any("error", mErr))
		}
		s.log(ctx, d, task, OutcomeInvalidToken, err.Error())
		return OutcomeInvalidToken, nil
	case errors.Is(err, push.ErrTransient):
		s.finish(ctx, d, repository.ReminderFailed, err)
		return "", fmt.Errorf("%w: send: %w", ErrPushTransient, err)
	default:
		s.finish(ctx, d, repository.ReminderFailed, err)
		s.log(ctx, d, task, OutcomeFailed, err.Error())
		return OutcomeFailed, s.scheduleAfter(ctx, d, fireAt, executing)
	}
}

// scheduleAfter keeps the chain alive; on failure the scheduler retries, and
// the retry finds the claim settled and only reschedules. Scheduling starts
// just past fireAt rather than at now, so a server clock behind the scheduler
// cannot re-pick tonight.
func (s *PushService) scheduleAfter(ctx context.Context, d repository.Device, fireAt time.Time, executing string) error {
	from := fireAt.Add(time.Second)
	if now := s.now(); now.After(from) {
		from = now
	}
	if err := s.scheduleFrom(ctx, d, from, executing); err != nil {
		return fmt.Errorf("%w: %w", ErrPushTransient, err)
	}
	return nil
}

func (s *PushService) finish(ctx context.Context, d repository.Device, status string, cause error) {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if err := s.devices.FinishReminder(ctx, d.ID, status, msg); err != nil {
		s.logger.ErrorContext(ctx, "finish reminder failed", slog.String("device_id", d.ID), slog.Any("error", err))
	}
}

func (s *PushService) log(ctx context.Context, d repository.Device, task ReminderTask, outcome Outcome, detail string) {
	s.logger.InfoContext(ctx, "reminder delivered",
		slog.String("device_id", d.ID), slog.String("platform", d.Platform),
		slog.String("local_date", task.LocalDate), slog.String("outcome", string(outcome)),
		slog.String("detail", detail))
}

// reminderTaskName is the scheduler's idempotency key. Cloud Tasks refuses a
// name for hours after it ran or was deleted, so the registration's updated_at
// is hashed in: every re-registration (time change, off→on) gets a fresh name,
// while the self-continuing chain keeps the same one per date.
func reminderTaskName(d repository.Device, localDate string) string {
	sum := sha256.Sum256([]byte(d.NotifyTime + "|" + d.TimeZone + "|" + strconv.FormatInt(d.UpdatedAt.UnixNano(), 10)))
	return "r-" + d.ID + "-" + localDate + "-" + hex.EncodeToString(sum[:4])
}

// nextFireAt is the next occurrence of notifyTime ("HH:MM") in loc strictly
// after now, with its local calendar date.
func nextFireAt(now time.Time, notifyTime string, loc *time.Location) (time.Time, string, error) {
	hour, minute, err := parseNotifyTime(notifyTime)
	if err != nil {
		return time.Time{}, "", err
	}
	local := now.In(loc)
	candidate := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, loc)
	if !candidate.After(now) {
		candidate = time.Date(local.Year(), local.Month(), local.Day()+1, hour, minute, 0, 0, loc)
	}
	return candidate, candidate.In(loc).Format("2006-01-02"), nil
}

// localInstant is localDate ("YYYY-MM-DD") at notifyTime in loc.
func localInstant(localDate, notifyTime string, loc *time.Location) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", localDate, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("local date %q: %w", localDate, err)
	}
	hour, minute, err := parseNotifyTime(notifyTime)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc), nil
}

func parseNotifyTime(s string) (hour, minute int, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("notify time %q: want HH:MM", s)
	}
	hour, err1 := strconv.Atoi(parts[0])
	minute, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("notify time %q: want HH:MM", s)
	}
	return hour, minute, nil
}
