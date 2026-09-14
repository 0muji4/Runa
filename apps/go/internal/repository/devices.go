package repository

import (
	"context"
	"time"
)

// Device is the persistence model for the devices table: a push token plus the
// user's reminder preference.
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

// UpsertDeviceParams carries the fields for an idempotent registration keyed by (UserID, PushToken).
type UpsertDeviceParams struct {
	UserID     string
	PushToken  string
	Platform   string
	NotifyTime string
	Enabled    bool
}

// DeviceStore is the data-access boundary for the devices feature.
type DeviceStore interface {
	// UpsertDevice inserts a device registration or, when (user_id, push_token)
	// already exists, updates it in place.
	UpsertDevice(ctx context.Context, p UpsertDeviceParams) (Device, error)
}
