package repository

import (
	"context"
	"time"
)

// Reminder claim statuses (devices.reminder_status).
const (
	ReminderPending = "pending"
	ReminderSent    = "sent"
	ReminderSkipped = "skipped"
	ReminderFailed  = "failed"
)

// Device is the persistence model for the devices table: one app install's push
// token, its reminder preference, and the state of the reminder delivery chain.
type Device struct {
	ID         string
	UserID     string
	InstallID  string
	PushToken  string
	Platform   string // "ios" | "android"
	NotifyTime string // local reminder time "HH:MM"
	TimeZone   string // IANA zone name
	Enabled    bool
	// TokenInvalidAt is set once the provider rejects the token; nil means usable.
	TokenInvalidAt *time.Time
	// NextTaskName / NextFireAt describe the scheduled callback for the next reminder.
	NextTaskName *string
	NextFireAt   *time.Time
	// Reminder* is the per-local-date claim ("YYYY-MM-DD"); nil until the first attempt.
	ReminderDate      *string
	ReminderStatus    *string
	ReminderAttempts  int
	ReminderClaimedAt *time.Time
	ReminderError     *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// UpsertDeviceParams carries the fields for an idempotent registration keyed by (UserID, InstallID).
type UpsertDeviceParams struct {
	UserID     string
	InstallID  string
	PushToken  string
	Platform   string
	NotifyTime string
	TimeZone   string
	Enabled    bool
}

// ClaimOutcome is the result of ClaimReminder.
type ClaimOutcome int

const (
	// ClaimWon means this call holds the attempt and must finish it.
	ClaimWon ClaimOutcome = iota
	// ClaimHeld means a pending attempt for that date is still live (not stale); retry later.
	ClaimHeld
	// ClaimDone means that date is settled: sent, skipped, or failed past MaxAttempts.
	ClaimDone
)

// ClaimReminderParams identifies one reminder attempt: the device and the local
// date it is for. A claim wins when no attempt exists for that date yet, when the
// previous attempt failed and MaxAttempts is not reached, or when a pending
// attempt is older than StalePendingAfter (the process died mid-send).
type ClaimReminderParams struct {
	DeviceID          string
	LocalDate         string // "YYYY-MM-DD" in the device's zone
	Now               time.Time
	StalePendingAfter time.Duration
	MaxAttempts       int
}

// DeviceStore is the data-access boundary for the devices feature.
type DeviceStore interface {
	// UpsertDevice registers a device or updates it in place when (user_id,
	// install_id) exists. Any other row holding the same push_token is deleted
	// first: a token belongs to whoever registered it last. Clears token_invalid_at.
	UpsertDevice(ctx context.Context, p UpsertDeviceParams) (Device, error)

	// GetDevice returns a device by id, or ErrNotFound.
	GetDevice(ctx context.Context, id string) (Device, error)

	// DeleteDevice removes the caller's device by install id and returns the
	// deleted row (so its scheduled task can be cancelled), or ErrNotFound.
	DeleteDevice(ctx context.Context, userID, installID string) (Device, error)

	// DeleteDevicesByUser removes every device of a user and returns them.
	DeleteDevicesByUser(ctx context.Context, userID string) ([]Device, error)

	// SetNextTask records the scheduled callback for the next reminder; it does
	// not bump updated_at, which tracks client registrations only.
	SetNextTask(ctx context.Context, id, taskName string, fireAt time.Time) error

	// ClaimReminder atomically takes the attempt described by p, or reports why not.
	ClaimReminder(ctx context.Context, p ClaimReminderParams) (ClaimOutcome, error)

	// FinishReminder records the outcome of the pending attempt on the device;
	// errMsg is stored only when non-empty. A device with no pending attempt is a no-op.
	FinishReminder(ctx context.Context, id, status, errMsg string) error

	// MarkTokenInvalid flags the token as rejected by the provider from at on.
	MarkTokenInvalid(ctx context.Context, id string, at time.Time) error
}
