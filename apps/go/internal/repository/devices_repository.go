package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// deviceColumns is the shared SELECT/RETURNING list; keep its order in sync with the Scan below.
const deviceColumns = `id, user_id, push_token, platform, notify_time, enabled,
	created_at, updated_at`

// DeviceRepository is the pgx-backed implementation of DeviceStore.
type DeviceRepository struct {
	pool *pgxpool.Pool
}

// NewDeviceRepository wraps a pgx pool. A nil pool (DB unreachable at boot) makes
// every method return ErrNoDatabase instead of panicking, so liveness still serves.
func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

var _ DeviceStore = (*DeviceRepository)(nil)

func (r *DeviceRepository) UpsertDevice(ctx context.Context, p UpsertDeviceParams) (Device, error) {
	if r.pool == nil {
		return Device{}, ErrNoDatabase
	}

	// ON CONFLICT on the (user_id, push_token) unique index makes a re-registration
	// idempotent (keeps id/created_at, takes the latest platform/notify_time/enabled).
	const q = `
		INSERT INTO devices (user_id, push_token, platform, notify_time, enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, push_token) DO UPDATE
			SET platform    = EXCLUDED.platform,
			    notify_time = EXCLUDED.notify_time,
			    enabled     = EXCLUDED.enabled,
			    updated_at  = now()
		RETURNING ` + deviceColumns

	var d Device
	err := r.pool.QueryRow(ctx, q, p.UserID, p.PushToken, p.Platform, p.NotifyTime, p.Enabled).Scan(
		&d.ID, &d.UserID, &d.PushToken, &d.Platform, &d.NotifyTime, &d.Enabled,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return Device{}, fmt.Errorf("upsert device: %w", err)
	}
	return d, nil
}
