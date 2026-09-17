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

func storedObject() storage.ObjectInfo {
	return storage.ObjectInfo{Size: 512, ContentType: "image/jpeg"}
}

// These build the services without the inline background runner; synctest.Wait
// blocks until the default `go f()` goroutine finishes.

func TestGalleryDeletePurgesObjectInBackground(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		objects := memobject.New()
		svc := service.NewGalleryService(memgallery.New(), objects, galleryConfig(), fixedNow)

		ctx := t.Context()
		key := "gallery/" + userA + "/purge-me"
		objects.Put(key, storedObject())

		img, err := svc.RegisterImage(ctx, userA, key, 800, 600)
		if err != nil {
			t.Fatalf("RegisterImage() error = %v, want nil", err)
		}

		if err := svc.Delete(ctx, userA, img.Image.ID); err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
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
			UserID: user.ID, ObjectKey: key, Width: 1, Height: 1,
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
