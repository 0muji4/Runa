package service

import (
	"context"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// RegisterDeviceInput is the device-registration payload. All fields are
// client-supplied; the handler validates their shape before this is called.
type RegisterDeviceInput struct {
	PushToken  string
	Platform   string
	NotifyTime string
	Enabled    bool
}

// DeviceService implements the devices use case over a DeviceStore. Every method
// is scoped by userID so a caller can only ever register their own device.
type DeviceService struct {
	store repository.DeviceStore
	now   func() time.Time
}

// NewDeviceService constructs the service, defaulting now to time.Now.
func NewDeviceService(store repository.DeviceStore, now func() time.Time) *DeviceService {
	if now == nil {
		now = time.Now
	}
	return &DeviceService{store: store, now: now}
}

// Register idempotently registers (or, on a repeated push_token, updates) the
// caller's device. Registration carries the user's reminder preference so a
// future server-initiated notification path knows when/whether to push.
func (s *DeviceService) Register(ctx context.Context, userID string, in RegisterDeviceInput) (repository.Device, error) {
	return s.store.UpsertDevice(ctx, repository.UpsertDeviceParams{
		UserID:     userID,
		PushToken:  in.PushToken,
		Platform:   in.Platform,
		NotifyTime: in.NotifyTime,
		Enabled:    in.Enabled,
	})
}
