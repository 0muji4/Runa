package repotest

import (
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

const (
	installA = "11111111-1111-4111-8111-000000000001"
	installB = "11111111-1111-4111-8111-000000000002"
)

func deviceParams(user, install, token string) repository.UpsertDeviceParams {
	return repository.UpsertDeviceParams{
		UserID: user, InstallID: install, PushToken: token, Platform: "ios",
		NotifyTime: "22:00", TimeZone: "Asia/Tokyo", Enabled: true,
	}
}

func mustUpsert(t *testing.T, f Fixture, p repository.UpsertDeviceParams) repository.Device {
	t.Helper()
	d, err := f.Devices.UpsertDevice(t.Context(), p)
	if err != nil {
		t.Fatalf("UpsertDevice(%s) error = %v, want nil", p.InstallID, err)
	}
	return d
}

func claim(t *testing.T, f Fixture, id, date string, now time.Time) repository.ClaimOutcome {
	t.Helper()
	got, err := f.Devices.ClaimReminder(t.Context(), repository.ClaimReminderParams{
		DeviceID: id, LocalDate: date, Now: now, StalePendingAfter: 5 * time.Minute, MaxAttempts: 3,
	})
	if err != nil {
		t.Fatalf("ClaimReminder(%s) error = %v, want nil", date, err)
	}
	return got
}

// RunDeviceStoreSuite exercises the DeviceStore contract.
func RunDeviceStoreSuite(t *testing.T, newFixture NewFixture) {
	t.Run("UpsertIsIdempotentPerInstall", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		user := f.NewUser(t)

		first := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		if first.ID == "" {
			t.Error("UpsertDevice() id is empty, want a generated id")
		}

		// The same install re-registers with a rotated token and new settings.
		p := deviceParams(user, installA, "token-2")
		p.NotifyTime, p.TimeZone, p.Enabled = "23:00", "Europe/London", false
		second := mustUpsert(t, f, p)
		if second.ID != first.ID {
			t.Errorf("re-registering the same install created id %q, want the existing %q", second.ID, first.ID)
		}
		if second.PushToken != "token-2" || second.NotifyTime != "23:00" || second.TimeZone != "Europe/London" || second.Enabled {
			t.Errorf("updated device = %+v, want token-2/23:00/Europe/London/disabled", second)
		}
		if second.UpdatedAt.Before(first.UpdatedAt) {
			t.Errorf("updated_at moved backwards: %s then %s", first.UpdatedAt.UTC(), second.UpdatedAt.UTC())
		}
	})

	t.Run("DistinctInstallsAreDistinctDevices", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		user := f.NewUser(t)

		phone := mustUpsert(t, f, deviceParams(user, installA, "token-ios"))
		p := deviceParams(user, installB, "token-android")
		p.Platform = "android"
		tablet := mustUpsert(t, f, p)
		if tablet.ID == phone.ID {
			t.Errorf("a second install reused device id %q, want a distinct device", phone.ID)
		}
		if tablet.Platform != "android" {
			t.Errorf("second device platform = %q, want %q", tablet.Platform, "android")
		}
	})

	t.Run("TokenFollowsLatestRegistrant", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		userA, userB := f.NewUser(t), f.NewUser(t)

		a := mustUpsert(t, f, deviceParams(userA, installA, "shared-token"))
		b := mustUpsert(t, f, deviceParams(userB, installB, "shared-token"))
		if b.ID == a.ID {
			t.Fatalf("second registrant reused device id %q, want a new row", a.ID)
		}
		if _, err := f.Devices.GetDevice(ctx, a.ID); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("previous holder of the token: GetDevice error = %v, want ErrNotFound", err)
		}
		if got, err := f.Devices.GetDevice(ctx, b.ID); err != nil || got.UserID != userB {
			t.Errorf("GetDevice(new holder) = (%+v, %v), want userB's row", got, err)
		}
	})

	t.Run("UpsertClearsTokenInvalid", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		d := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		if err := f.Devices.MarkTokenInvalid(ctx, d.ID, time.Now()); err != nil {
			t.Fatalf("MarkTokenInvalid() error = %v, want nil", err)
		}
		got, err := f.Devices.GetDevice(ctx, d.ID)
		if err != nil || got.TokenInvalidAt == nil {
			t.Fatalf("after MarkTokenInvalid: GetDevice = (%+v, %v), want token_invalid_at set", got, err)
		}
		again := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		if again.TokenInvalidAt != nil {
			t.Errorf("re-registration kept token_invalid_at = %v, want nil", again.TokenInvalidAt)
		}
	})

	t.Run("GetDeviceUnknownIsNotFound", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		if _, err := f.Devices.GetDevice(t.Context(), "11111111-1111-4111-8111-0000000000ff"); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("GetDevice(unknown) error = %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteDeviceReturnsRowThenNotFound", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user, other := f.NewUser(t), f.NewUser(t)

		d := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		if _, err := f.Devices.DeleteDevice(ctx, other, installA); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("DeleteDevice(other user) error = %v, want ErrNotFound", err)
		}
		deleted, err := f.Devices.DeleteDevice(ctx, user, installA)
		if err != nil || deleted.ID != d.ID {
			t.Fatalf("DeleteDevice() = (%+v, %v), want the registered row", deleted, err)
		}
		if _, err := f.Devices.DeleteDevice(ctx, user, installA); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("second DeleteDevice() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteDevicesByUserRemovesOnlyTheirs", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user, other := f.NewUser(t), f.NewUser(t)

		mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		mustUpsert(t, f, deviceParams(user, installB, "token-2"))
		keep := mustUpsert(t, f, deviceParams(other, installA, "token-3"))

		deleted, err := f.Devices.DeleteDevicesByUser(ctx, user)
		if err != nil || len(deleted) != 2 {
			t.Fatalf("DeleteDevicesByUser() = (%d rows, %v), want 2 rows", len(deleted), err)
		}
		if _, err := f.Devices.GetDevice(ctx, keep.ID); err != nil {
			t.Errorf("another user's device was removed: GetDevice error = %v", err)
		}
		if again, err := f.Devices.DeleteDevicesByUser(ctx, user); err != nil || len(again) != 0 {
			t.Errorf("second DeleteDevicesByUser() = (%d rows, %v), want 0 rows", len(again), err)
		}
	})

	t.Run("SetNextTaskDoesNotBumpUpdatedAt", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		d := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		fireAt := time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC)
		if err := f.Devices.SetNextTask(ctx, d.ID, "r-task", fireAt); err != nil {
			t.Fatalf("SetNextTask() error = %v, want nil", err)
		}
		got, err := f.Devices.GetDevice(ctx, d.ID)
		if err != nil {
			t.Fatalf("GetDevice() error = %v", err)
		}
		if got.NextTaskName == nil || *got.NextTaskName != "r-task" || got.NextFireAt == nil || !got.NextFireAt.Equal(fireAt) {
			t.Errorf("next task = (%v, %v), want (r-task, %s)", got.NextTaskName, got.NextFireAt, fireAt)
		}
		if !got.UpdatedAt.Equal(d.UpdatedAt) {
			t.Errorf("updated_at changed from %s to %s, want unchanged", d.UpdatedAt.UTC(), got.UpdatedAt.UTC())
		}
	})

	t.Run("ClaimIsExclusivePerLocalDate", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		d := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		now := time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC)

		if got := claim(t, f, d.ID, "2026-09-20", now); got != repository.ClaimWon {
			t.Fatalf("first claim = %v, want ClaimWon", got)
		}
		if got := claim(t, f, d.ID, "2026-09-20", now.Add(time.Minute)); got != repository.ClaimHeld {
			t.Errorf("second claim while pending = %v, want ClaimHeld", got)
		}
		if err := f.Devices.FinishReminder(ctx, d.ID, repository.ReminderSent, ""); err != nil {
			t.Fatalf("FinishReminder() error = %v", err)
		}
		if got := claim(t, f, d.ID, "2026-09-20", now.Add(time.Hour)); got != repository.ClaimDone {
			t.Errorf("claim after sent = %v, want ClaimDone", got)
		}
		if got := claim(t, f, d.ID, "2026-09-21", now.Add(24*time.Hour)); got != repository.ClaimWon {
			t.Errorf("claim for the next date = %v, want ClaimWon", got)
		}
		got, _ := f.Devices.GetDevice(ctx, d.ID)
		if got.ReminderAttempts != 1 || got.ReminderDate == nil || *got.ReminderDate != "2026-09-21" {
			t.Errorf("after a new date: attempts = %d, date = %v, want 1 / 2026-09-21", got.ReminderAttempts, got.ReminderDate)
		}
	})

	t.Run("ClaimRetriesFailedUpToMaxAttempts", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		d := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		now := time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC)

		for attempt := 1; attempt <= 3; attempt++ {
			if got := claim(t, f, d.ID, "2026-09-20", now); got != repository.ClaimWon {
				t.Fatalf("attempt %d: claim = %v, want ClaimWon", attempt, got)
			}
			got, _ := f.Devices.GetDevice(ctx, d.ID)
			if got.ReminderAttempts != attempt {
				t.Errorf("attempt %d: attempts = %d", attempt, got.ReminderAttempts)
			}
			if err := f.Devices.FinishReminder(ctx, d.ID, repository.ReminderFailed, "apns: 503"); err != nil {
				t.Fatalf("FinishReminder() error = %v", err)
			}
		}
		if got := claim(t, f, d.ID, "2026-09-20", now); got != repository.ClaimDone {
			t.Errorf("claim after max attempts = %v, want ClaimDone", got)
		}
		got, _ := f.Devices.GetDevice(ctx, d.ID)
		if got.ReminderError == nil || *got.ReminderError != "apns: 503" {
			t.Errorf("reminder_error = %v, want the last failure", got.ReminderError)
		}
	})

	t.Run("ClaimReclaimsStalePending", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		user := f.NewUser(t)
		d := mustUpsert(t, f, deviceParams(user, installA, "token-1"))
		now := time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC)

		if got := claim(t, f, d.ID, "2026-09-20", now); got != repository.ClaimWon {
			t.Fatalf("first claim = %v, want ClaimWon", got)
		}
		if got := claim(t, f, d.ID, "2026-09-20", now.Add(time.Minute)); got != repository.ClaimHeld {
			t.Errorf("claim 1 minute later = %v, want ClaimHeld (still pending)", got)
		}
		if got := claim(t, f, d.ID, "2026-09-20", now.Add(10*time.Minute)); got != repository.ClaimWon {
			t.Errorf("claim 10 minutes later = %v, want ClaimWon (pending went stale)", got)
		}
	})

	t.Run("FinishReminderIgnoresNonPending", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		d := mustUpsert(t, f, deviceParams(user, installA, "token-1"))

		if err := f.Devices.FinishReminder(ctx, d.ID, repository.ReminderSent, ""); err != nil {
			t.Fatalf("FinishReminder(no claim) error = %v, want nil", err)
		}
		got, _ := f.Devices.GetDevice(ctx, d.ID)
		if got.ReminderStatus != nil {
			t.Errorf("reminder_status = %q after finishing without a claim, want nil", *got.ReminderStatus)
		}
	})
}
