package repository

import (
	"context"
	"time"
)

// Device is the persistence model for the devices table (migration 0006). It
// records a client's push token plus the user's reminder preference, so a future
// server-initiated notification path (FCM/APNs) knows where, when and whether to
// push. The nightly reminder itself is a local, on-device notification this slice.
type Device struct {
	ID         string
	UserID     string
	PushToken  string
	Platform   string // "ios" | "android"
	NotifyTime string // local reminder time "HH:MM"
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// UpsertDeviceParams carries the fields for an idempotent registration keyed by
// (UserID, PushToken). A repeated PUT for the same token updates the row in place.
type UpsertDeviceParams struct {
	UserID     string
	PushToken  string
	Platform   string
	NotifyTime string
	Enabled    bool
}

// DeviceStore is the data-access boundary for the devices feature. The service
// depends on this interface so tests can substitute an in-memory fake. It is
// scoped by user id: the unique key is (user_id, push_token).
type DeviceStore interface {
	// UpsertDevice inserts a new device registration or, when (user_id,
	// push_token) already exists, updates its platform/notify_time/enabled in
	// place. This makes PUT /devices idempotent.
	UpsertDevice(ctx context.Context, p UpsertDeviceParams) (Device, error)
}
