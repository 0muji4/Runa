package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// deviceColumns is the shared SELECT/RETURNING list; keep its order in sync with scanDevice.
// reminder_date is cast to text so the model carries the local date verbatim.
const deviceColumns = `id, user_id, install_id, push_token, platform, notify_time, time_zone, enabled,
	token_invalid_at, next_task_name, next_fire_at,
	reminder_date::text, reminder_status, reminder_attempts, reminder_claimed_at, reminder_error,
	created_at, updated_at`

// DeviceRepository is the pgx-backed implementation of DeviceStore.
type DeviceRepository struct {
	pool *pgxpool.Pool
}

// NewDeviceRepository wraps a pgx pool; a nil pool makes every method return ErrNoDatabase.
func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

var _ DeviceStore = (*DeviceRepository)(nil)

func scanDevice(row pgx.Row) (Device, error) {
	var d Device
	err := row.Scan(
		&d.ID, &d.UserID, &d.InstallID, &d.PushToken, &d.Platform, &d.NotifyTime, &d.TimeZone, &d.Enabled,
		&d.TokenInvalidAt, &d.NextTaskName, &d.NextFireAt,
		&d.ReminderDate, &d.ReminderStatus, &d.ReminderAttempts, &d.ReminderClaimedAt, &d.ReminderError,
		&d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

func (r *DeviceRepository) UpsertDevice(ctx context.Context, p UpsertDeviceParams) (Device, error) {
	if r.pool == nil {
		return Device{}, ErrNoDatabase
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Device{}, fmt.Errorf("upsert device: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	// The token moves to this (user, install); the previous holder's row goes away.
	const evict = `
		DELETE FROM devices
		WHERE push_token = $1 AND NOT (user_id = $2 AND install_id = $3)`
	if _, err := tx.Exec(ctx, evict, p.PushToken, p.UserID, p.InstallID); err != nil {
		return Device{}, fmt.Errorf("upsert device: evict token holder: %w", err)
	}

	const q = `
		INSERT INTO devices (user_id, install_id, push_token, platform, notify_time, time_zone, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, install_id) DO UPDATE
			SET push_token       = EXCLUDED.push_token,
			    platform         = EXCLUDED.platform,
			    notify_time      = EXCLUDED.notify_time,
			    time_zone        = EXCLUDED.time_zone,
			    enabled          = EXCLUDED.enabled,
			    token_invalid_at = NULL,
			    updated_at       = now()
		RETURNING ` + deviceColumns
	d, err := scanDevice(tx.QueryRow(ctx, q, p.UserID, p.InstallID, p.PushToken, p.Platform, p.NotifyTime, p.TimeZone, p.Enabled))
	if err != nil {
		return Device{}, fmt.Errorf("upsert device: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Device{}, fmt.Errorf("upsert device: commit: %w", err)
	}
	return d, nil
}

func (r *DeviceRepository) GetDevice(ctx context.Context, id string) (Device, error) {
	if r.pool == nil {
		return Device{}, ErrNoDatabase
	}
	d, err := scanDevice(r.pool.QueryRow(ctx, `SELECT `+deviceColumns+` FROM devices WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Device{}, ErrNotFound
		}
		return Device{}, fmt.Errorf("get device: %w", err)
	}
	return d, nil
}

func (r *DeviceRepository) DeleteDevice(ctx context.Context, userID, installID string) (Device, error) {
	if r.pool == nil {
		return Device{}, ErrNoDatabase
	}
	const q = `DELETE FROM devices WHERE user_id = $1 AND install_id = $2 RETURNING ` + deviceColumns
	d, err := scanDevice(r.pool.QueryRow(ctx, q, userID, installID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Device{}, ErrNotFound
		}
		return Device{}, fmt.Errorf("delete device: %w", err)
	}
	return d, nil
}

func (r *DeviceRepository) DeleteDevicesByUser(ctx context.Context, userID string) ([]Device, error) {
	if r.pool == nil {
		return nil, ErrNoDatabase
	}
	rows, err := r.pool.Query(ctx, `DELETE FROM devices WHERE user_id = $1 RETURNING `+deviceColumns, userID)
	if err != nil {
		return nil, fmt.Errorf("delete devices by user: %w", err)
	}
	defer rows.Close()
	out := make([]Device, 0)
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, fmt.Errorf("delete devices by user: scan: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("delete devices by user: rows: %w", err)
	}
	return out, nil
}

func (r *DeviceRepository) SetNextTask(ctx context.Context, id, taskName string, fireAt time.Time) error {
	if r.pool == nil {
		return ErrNoDatabase
	}
	const q = `UPDATE devices SET next_task_name = $2, next_fire_at = $3 WHERE id = $1`
	if _, err := r.pool.Exec(ctx, q, id, taskName, fireAt); err != nil {
		return fmt.Errorf("set next task: %w", err)
	}
	return nil
}

func (r *DeviceRepository) ClaimReminder(ctx context.Context, p ClaimReminderParams) (ClaimOutcome, error) {
	if r.pool == nil {
		return ClaimDone, ErrNoDatabase
	}
	// One statement decides and records the claim so concurrent callbacks cannot both win.
	const q = `
		UPDATE devices
		SET reminder_date       = $2::date,
		    reminder_status     = 'pending',
		    reminder_attempts   = CASE WHEN reminder_date IS DISTINCT FROM $2::date THEN 1 ELSE reminder_attempts + 1 END,
		    reminder_claimed_at = $3,
		    reminder_error      = NULL
		WHERE id = $1
		  AND (reminder_date IS DISTINCT FROM $2::date
		       OR (reminder_status = 'failed' AND reminder_attempts < $5)
		       OR (reminder_status = 'pending' AND reminder_claimed_at < $4))`
	staleBefore := p.Now.Add(-p.StalePendingAfter)
	tag, err := r.pool.Exec(ctx, q, p.DeviceID, p.LocalDate, p.Now, staleBefore, p.MaxAttempts)
	if err != nil {
		return ClaimDone, fmt.Errorf("claim reminder: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return ClaimWon, nil
	}
	const held = `
		SELECT EXISTS (
			SELECT 1 FROM devices
			WHERE id = $1 AND reminder_date = $2::date
			  AND reminder_status = 'pending' AND reminder_claimed_at >= $3)`
	var live bool
	if err := r.pool.QueryRow(ctx, held, p.DeviceID, p.LocalDate, staleBefore).Scan(&live); err != nil {
		return ClaimDone, fmt.Errorf("claim reminder: %w", err)
	}
	if live {
		return ClaimHeld, nil
	}
	return ClaimDone, nil
}

func (r *DeviceRepository) FinishReminder(ctx context.Context, id, status, errMsg string) error {
	if r.pool == nil {
		return ErrNoDatabase
	}
	const q = `
		UPDATE devices SET reminder_status = $2, reminder_error = NULLIF($3::text, '')
		WHERE id = $1 AND reminder_status = 'pending'`
	if _, err := r.pool.Exec(ctx, q, id, status, errMsg); err != nil {
		return fmt.Errorf("finish reminder: %w", err)
	}
	return nil
}

func (r *DeviceRepository) MarkTokenInvalid(ctx context.Context, id string, at time.Time) error {
	if r.pool == nil {
		return ErrNoDatabase
	}
	if _, err := r.pool.Exec(ctx, `UPDATE devices SET token_invalid_at = $2 WHERE id = $1`, id, at); err != nil {
		return fmt.Errorf("mark token invalid: %w", err)
	}
	return nil
}
