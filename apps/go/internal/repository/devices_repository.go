package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// deviceColumns is the SELECT/RETURNING list shared by every device query, kept
// in one place so the scan order stays in sync.
const deviceColumns = `id, user_id, push_token, platform, notify_time, enabled,
	created_at, updated_at`

// DeviceRepository is the pgx-backed implementation of DeviceStore.
type DeviceRepository struct {
	pool *pgxpool.Pool
}

// NewDeviceRepository wraps a pgx pool. As with the other repositories the pool
// may be nil when the DB is unreachable at boot; every method then returns
// ErrNoDatabase instead of panicking, so the process still serves liveness.
func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

var _ DeviceStore = (*DeviceRepository)(nil)

func (r *DeviceRepository) UpsertDevice(ctx context.Context, p UpsertDeviceParams) (Device, error) {
	if r.pool == nil {
		return Device{}, ErrNoDatabase
	}

	// ON CONFLICT upserts onto the (user_id, push_token) unique index, so a
	// re-registration of the same token never duplicates. A conflicting row keeps
	// its id/created_at and takes the latest platform/notify_time/enabled.
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
