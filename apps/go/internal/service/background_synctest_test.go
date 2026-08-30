package service_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
	"github.com/0muji4/Runa/apps/go/internal/storage/memobject"
)

// storedObject is the metadata a client's direct upload would have left behind.
func storedObject() storage.ObjectInfo {
	return storage.ObjectInfo{Size: 512, ContentType: "image/jpeg"}
}

// Everywhere else in this package the background runner is replaced with an
// inline one (syncBackground in helpers_test.go), so the `go f()` the
// constructors default to never runs. These build the services without that
// override; synctest.Wait blocks until the goroutine finishes, which needs no
// polling or a WaitGroup the production code does not have.

func TestGalleryDeletePurgesObjectInBackground(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		objects := memobject.New()
		// No WithBackgroundRunner: this is the wiring cmd/api builds.
		svc := service.NewGalleryService(memgallery.New(), objects, galleryConfig(), fixedNow)

		ctx := t.Context()
		key := "gallery/" + userA + "/purge-me"
		objects.Put(key, storedObject())

		img, err := svc.RegisterImage(ctx, userA, key, 800, 600, "pink")
		if err != nil {
			t.Fatalf("RegisterImage() error = %v, want nil", err)
		}

		if err := svc.Delete(ctx, userA, img.Image.ID); err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
		// Delete returns once the row is soft-deleted; the removal is still in
		// flight on its own goroutine.
		synctest.Wait()

		if !objects.Removed(key) {
			t.Errorf("object %q was not purged by the background goroutine", key)
		}
	})
}

func TestDeleteAccountPurgesObjectsInBackground(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		objects := memobject.New()
		users, diaries, gallery := memauth.New(), memdiary.New(), memgallery.New()
		svc := service.NewAccountService(users, diaries, gallery, objects,
			service.AccountConfig{ExportURLTTL: time.Hour}, fixedNow)

		ctx := t.Context()
		email := "purge@example.com"
		user, err := users.CreateUser(ctx, repository.CreateUserParams{
			Email: &email, AuthProvider: "email", DisplayName: "Purge",
		})
		if err != nil {
			t.Fatalf("CreateUser() error = %v, want nil", err)
		}
		key := "gallery/" + user.ID + "/k1"
		if _, err := gallery.InsertImage(ctx, repository.InsertGalleryParams{
			UserID: user.ID, ObjectKey: key, Width: 1, Height: 1, Theme: "pink",
		}); err != nil {
			t.Fatalf("InsertImage() error = %v, want nil", err)
		}
		objects.Put(key, storedObject())

		if err := svc.DeleteAccount(ctx, user.ID); err != nil {
			t.Fatalf("DeleteAccount() error = %v, want nil", err)
		}
		synctest.Wait()

		if !objects.Removed(key) {
			t.Errorf("object %q was not purged by the background goroutine", key)
		}
	})
}
