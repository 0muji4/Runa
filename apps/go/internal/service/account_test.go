package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

// fakeObjects is a minimal storage.ObjectStore for the account service tests. It
// signs deterministic URLs and records removed keys.
type fakeObjects struct {
	removed []string
}

func (f *fakeObjects) EnsureBucket(context.Context) error { return nil }
func (f *fakeObjects) PresignPut(_ context.Context, key string, _ time.Duration) (string, error) {
	return "put:" + key, nil
}
func (f *fakeObjects) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "get:" + key, nil
}
func (f *fakeObjects) Stat(context.Context, string) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{}, nil
}
func (f *fakeObjects) Remove(_ context.Context, key string) error {
	f.removed = append(f.removed, key)
	return nil
}

func newAccountService(objects storage.ObjectStore) (*service.AccountService, *memauth.Store, *memdiary.Store, *memgallery.Store) {
	users := memauth.New()
	diaries := memdiary.New()
	gallery := memgallery.New()
	svc := service.NewAccountService(users, diaries, gallery, objects, service.AccountConfig{ExportURLTTL: time.Hour}, nil,
		service.WithAccountBackgroundRunner(func(f func()) { f() }))
	return svc, users, diaries, gallery
}

func createTestUser(t *testing.T, users *memauth.Store) repository.User {
	t.Helper()
	email := "u@example.com"
	u, err := users.CreateUser(context.Background(), repository.CreateUserParams{
		Email: &email, AuthProvider: "email", DisplayName: "U",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func TestAccountUpdateDisplayNameValidation(t *testing.T) {
	svc, users, _, _ := newAccountService(nil)
	u := createTestUser(t, users)
	ctx := context.Background()

	got, err := svc.UpdateDisplayName(ctx, u.ID, "  月子  ")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.DisplayName != "月子" {
		t.Errorf("display_name = %q, want 月子 (trimmed)", got.DisplayName)
	}

	if _, err := svc.UpdateDisplayName(ctx, u.ID, "   "); !errors.Is(err, service.ErrDisplayNameRequired) {
		t.Errorf("empty name err = %v, want ErrDisplayNameRequired", err)
	}

	long := strings.Repeat("あ", service.MaxDisplayNameLength+1)
	if _, err := svc.UpdateDisplayName(ctx, u.ID, long); !errors.Is(err, service.ErrDisplayNameTooLong) {
		t.Errorf("long name err = %v, want ErrDisplayNameTooLong", err)
	}

	if _, err := svc.UpdateDisplayName(ctx, "missing", "x"); !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("missing user err = %v, want ErrUserNotFound", err)
	}
}

func TestAccountExportExcludesTombstones(t *testing.T) {
	svc, users, diaries, gallery := newAccountService(&fakeObjects{})
	u := createTestUser(t, users)
	ctx := context.Background()

	if _, _, err := diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: u.ID, ClientID: "c1", BodyText: "live", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed live entry: %v", err)
	}
	dead, _, err := diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: u.ID, ClientID: "c2", BodyText: "dead", CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed dead entry: %v", err)
	}
	if err := diaries.SoftDeleteEntry(ctx, u.ID, dead.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err := gallery.InsertImage(ctx, repository.InsertGalleryParams{
		UserID: u.ID, ObjectKey: "gallery/" + u.ID + "/k1", Width: 1, Height: 1, Theme: "pink",
	}); err != nil {
		t.Fatalf("seed image: %v", err)
	}

	export, err := svc.Export(ctx, u.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(export.Diaries) != 1 || export.Diaries[0].BodyText != "live" {
		t.Errorf("diaries = %+v, want only the live entry", export.Diaries)
	}
	if len(export.Images) != 1 || export.Images[0].URL == "" {
		t.Errorf("images = %+v, want one image with a URL", export.Images)
	}
}

func TestAccountDeletePurgesObjectsAndUser(t *testing.T) {
	objects := &fakeObjects{}
	svc, users, _, gallery := newAccountService(objects)
	u := createTestUser(t, users)
	ctx := context.Background()

	key := "gallery/" + u.ID + "/k1"
	if _, err := gallery.InsertImage(ctx, repository.InsertGalleryParams{
		UserID: u.ID, ObjectKey: key, Width: 1, Height: 1, Theme: "pink",
	}); err != nil {
		t.Fatalf("seed image: %v", err)
	}

	if err := svc.DeleteAccount(ctx, u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := users.GetUserByID(ctx, u.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("user lookup after delete err = %v, want ErrNotFound", err)
	}
	if len(objects.removed) != 1 || objects.removed[0] != key {
		t.Errorf("removed = %v, want [%s]", objects.removed, key)
	}
	if err := svc.DeleteAccount(ctx, u.ID); !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("re-delete err = %v, want ErrUserNotFound", err)
	}
}

// TestAccountDeleteWithoutStorage confirms deletion still succeeds when object
// storage is unconfigured (nil ObjectStore): the row goes, there is nothing to purge.
func TestAccountDeleteWithoutStorage(t *testing.T) {
	svc, users, _, _ := newAccountService(nil)
	u := createTestUser(t, users)
	if err := svc.DeleteAccount(context.Background(), u.ID); err != nil {
		t.Fatalf("delete without storage: %v", err)
	}
}
