// Package memdevices is an in-memory implementation of repository.DeviceStore.
package memdevices

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Store is a goroutine-safe, in-memory DeviceStore.
type Store struct {
	mu      sync.Mutex
	devices map[string]repository.Device // keyed by device id
	now     func() time.Time
	lastNow time.Time
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		devices: make(map[string]repository.Device),
		now:     time.Now,
	}
}

var _ repository.DeviceStore = (*Store)(nil)

func (s *Store) UpsertDevice(_ context.Context, p repository.UpsertDeviceParams) (repository.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Mirror the DB: the token moves to this (user, install), evicting any other holder.
	for id, d := range s.devices {
		if d.PushToken == p.PushToken && !(d.UserID == p.UserID && d.InstallID == p.InstallID) {
			delete(s.devices, id)
		}
	}

	if existing, ok := s.findByInstall(p.UserID, p.InstallID); ok {
		existing.PushToken = p.PushToken
		existing.Platform = p.Platform
		existing.NotifyTime = p.NotifyTime
		existing.TimeZone = p.TimeZone
		existing.Enabled = p.Enabled
		existing.TokenInvalidAt = nil
		existing.UpdatedAt = s.tick()
		s.devices[existing.ID] = existing
		return existing, nil
	}

	now := s.tick()
	d := repository.Device{
		ID:         newID(),
		UserID:     p.UserID,
		InstallID:  p.InstallID,
		PushToken:  p.PushToken,
		Platform:   p.Platform,
		NotifyTime: p.NotifyTime,
		TimeZone:   p.TimeZone,
		Enabled:    p.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.devices[d.ID] = d
	return d, nil
}

func (s *Store) GetDevice(_ context.Context, id string) (repository.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	if !ok {
		return repository.Device{}, repository.ErrNotFound
	}
	return d, nil
}

func (s *Store) DeleteDevice(_ context.Context, userID, installID string) (repository.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.findByInstall(userID, installID)
	if !ok {
		return repository.Device{}, repository.ErrNotFound
	}
	delete(s.devices, d.ID)
	return d, nil
}

func (s *Store) DeleteDevicesByUser(_ context.Context, userID string) ([]repository.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]repository.Device, 0)
	for id, d := range s.devices {
		if d.UserID == userID {
			out = append(out, d)
			delete(s.devices, id)
		}
	}
	return out, nil
}

func (s *Store) SetNextTask(_ context.Context, id, taskName string, fireAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	if !ok {
		return nil
	}
	d.NextTaskName = &taskName
	d.NextFireAt = &fireAt
	s.devices[id] = d
	return nil
}

func (s *Store) ClaimReminder(_ context.Context, p repository.ClaimReminderParams) (repository.ClaimOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[p.DeviceID]
	if !ok {
		return repository.ClaimDone, nil
	}
	sameDate := d.ReminderDate != nil && *d.ReminderDate == p.LocalDate
	status := ""
	if d.ReminderStatus != nil {
		status = *d.ReminderStatus
	}
	staleBefore := p.Now.Add(-p.StalePendingAfter)
	livePending := status == repository.ReminderPending && d.ReminderClaimedAt != nil && !d.ReminderClaimedAt.Before(staleBefore)
	claimable := !sameDate ||
		(status == repository.ReminderFailed && d.ReminderAttempts < p.MaxAttempts) ||
		(status == repository.ReminderPending && !livePending)
	if !claimable {
		if sameDate && livePending {
			return repository.ClaimHeld, nil
		}
		return repository.ClaimDone, nil
	}
	if sameDate {
		d.ReminderAttempts++
	} else {
		d.ReminderAttempts = 1
	}
	date, pending, claimedAt := p.LocalDate, repository.ReminderPending, p.Now
	d.ReminderDate = &date
	d.ReminderStatus = &pending
	d.ReminderClaimedAt = &claimedAt
	d.ReminderError = nil
	s.devices[p.DeviceID] = d
	return repository.ClaimWon, nil
}

func (s *Store) FinishReminder(_ context.Context, id, status, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	if !ok || d.ReminderStatus == nil || *d.ReminderStatus != repository.ReminderPending {
		return nil
	}
	d.ReminderStatus = &status
	d.ReminderError = nil
	if errMsg != "" {
		d.ReminderError = &errMsg
	}
	s.devices[id] = d
	return nil
}

func (s *Store) MarkTokenInvalid(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	if !ok {
		return nil
	}
	d.TokenInvalidAt = &at
	s.devices[id] = d
	return nil
}

// findByInstall locates a device by (userID, installID); caller holds the lock.
func (s *Store) findByInstall(userID, installID string) (repository.Device, bool) {
	for _, d := range s.devices {
		if d.UserID == userID && d.InstallID == installID {
			return d, true
		}
	}
	return repository.Device{}, false
}

// tick returns a strictly increasing timestamp so updated_at values never tie; caller holds the lock.
func (s *Store) tick() time.Time {
	t := s.now()
	if !t.After(s.lastNow) {
		t = s.lastNow.Add(time.Nanosecond)
	}
	s.lastNow = t
	return t
}

// newID returns a random v4-style UUID string.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
