package repotest

import (
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// RunDeviceStoreSuite exercises the DeviceStore contract: registration upserts on
// (user_id, push_token), which is what makes a retried PUT /devices idempotent.
func RunDeviceStoreSuite(t *testing.T, newFixture NewFixture) {
	t.Run("UpsertIsIdempotentPerPushToken", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		first, err := f.Devices.UpsertDevice(ctx, repository.UpsertDeviceParams{
			UserID: user, PushToken: "token-abc", Platform: "ios",
			NotifyTime: "22:00", Enabled: true,
		})
		if err != nil {
			t.Fatalf("first UpsertDevice() error = %v, want nil", err)
		}
		if first.ID == "" {
			t.Error("UpsertDevice() id is empty, want a generated id")
		}

		second, err := f.Devices.UpsertDevice(ctx, repository.UpsertDeviceParams{
			UserID: user, PushToken: "token-abc", Platform: "ios",
			NotifyTime: "23:00", Enabled: false,
		})
		if err != nil {
			t.Fatalf("second UpsertDevice() error = %v, want nil", err)
		}
		if second.ID != first.ID {
			t.Errorf("re-registering the same push token created id %q, want the existing %q",
				second.ID, first.ID)
		}
		if second.NotifyTime != "23:00" {
			t.Errorf("notify_time = %q, want %q", second.NotifyTime, "23:00")
		}
		if second.Enabled {
			t.Error("enabled = true, want false")
		}
		if !second.UpdatedAt.After(first.UpdatedAt) && !second.UpdatedAt.Equal(first.UpdatedAt) {
			t.Errorf("updated_at moved backwards: %s then %s",
				first.UpdatedAt.UTC(), second.UpdatedAt.UTC())
		}
	})

	t.Run("DistinctTokensAreDistinctDevices", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		phone, err := f.Devices.UpsertDevice(ctx, repository.UpsertDeviceParams{
			UserID: user, PushToken: "token-ios", Platform: "ios",
			NotifyTime: "22:00", Enabled: true,
		})
		if err != nil {
			t.Fatalf("UpsertDevice(ios) error = %v, want nil", err)
		}
		tablet, err := f.Devices.UpsertDevice(ctx, repository.UpsertDeviceParams{
			UserID: user, PushToken: "token-android", Platform: "android",
			NotifyTime: "21:00", Enabled: true,
		})
		if err != nil {
			t.Fatalf("UpsertDevice(android) error = %v, want nil", err)
		}
		if tablet.ID == phone.ID {
			t.Errorf("a second push token reused device id %q, want a distinct device", phone.ID)
		}
		if tablet.Platform != "android" {
			t.Errorf("second device platform = %q, want %q", tablet.Platform, "android")
		}
	})

	t.Run("TokensAreScopedPerUser", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		userA, userB := f.NewUser(t), f.NewUser(t)

		a, err := f.Devices.UpsertDevice(ctx, repository.UpsertDeviceParams{
			UserID: userA, PushToken: "shared-token", Platform: "ios",
			NotifyTime: "22:00", Enabled: true,
		})
		if err != nil {
			t.Fatalf("UpsertDevice(userA) error = %v, want nil", err)
		}
		// The unique key is (user_id, push_token), not the token alone.
		b, err := f.Devices.UpsertDevice(ctx, repository.UpsertDeviceParams{
			UserID: userB, PushToken: "shared-token", Platform: "android",
			NotifyTime: "21:00", Enabled: true,
		})
		if err != nil {
			t.Fatalf("UpsertDevice(userB) error = %v, want nil", err)
		}
		if a.ID == b.ID {
			t.Errorf("two users sharing a push token collided on id %q, want distinct rows", a.ID)
		}
	})
}
