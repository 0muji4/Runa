package service

import (
	"context"
	"errors"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// RegisterDeviceInput is the device-registration payload, validated by the handler.
type RegisterDeviceInput struct {
	InstallID  string
	PushToken  string
	Platform   string
	NotifyTime string
	TimeZone   string
	Enabled    bool
}

// DeviceService implements the devices use case and keeps the reminder schedule in step.
type DeviceService struct {
	store repository.DeviceStore
	push  *PushService // nil when push is not wired (tests, or push disabled)
	now   func() time.Time
}

// NewDeviceService constructs the service, defaulting now to time.Now.
func NewDeviceService(store repository.DeviceStore, push *PushService, now func() time.Time) *DeviceService {
	if now == nil {
		now = time.Now
	}
	return &DeviceService{store: store, push: push, now: now}
}

// Register upserts the caller's device and schedules its next reminder; a
// scheduling failure fails the request so the client retries the whole upsert.
func (s *DeviceService) Register(ctx context.Context, userID string, in RegisterDeviceInput) (repository.Device, error) {
	d, err := s.store.UpsertDevice(ctx, repository.UpsertDeviceParams{
		UserID:     userID,
		InstallID:  in.InstallID,
		PushToken:  in.PushToken,
		Platform:   in.Platform,
		NotifyTime: in.NotifyTime,
		TimeZone:   in.TimeZone,
		Enabled:    in.Enabled,
	})
	if err != nil {
		return repository.Device{}, err
	}
	if s.push != nil {
		if err := s.push.ScheduleNext(ctx, d); err != nil {
			return repository.Device{}, err
		}
	}
	return d, nil
}

// Unregister removes one install's device and its pending reminder; unknown installs are a no-op.
func (s *DeviceService) Unregister(ctx context.Context, userID, installID string) error {
	d, err := s.store.DeleteDevice(ctx, userID, installID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	if s.push != nil {
		s.push.CancelScheduled(ctx, d)
	}
	return nil
}

// UnregisterAll removes every device of a user (account deletion).
func (s *DeviceService) UnregisterAll(ctx context.Context, userID string) error {
	devices, err := s.store.DeleteDevicesByUser(ctx, userID)
	if err != nil {
		return err
	}
	if s.push != nil {
		for _, d := range devices {
			s.push.CancelScheduled(ctx, d)
		}
	}
	return nil
}
