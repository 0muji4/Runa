package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
	"github.com/0muji4/Runa/apps/go/internal/storage/memobject"
	"github.com/google/go-cmp/cmp"
)

func createTestUser(t *testing.T, users *memauth.Store) repository.User {
	t.Helper()
	email := "u@example.com"
	u, err := users.CreateUser(context.Background(), repository.CreateUserParams{
		Email: &email, AuthProvider: "email", DisplayName: "U",
	})
	if err != nil {
		t.Fatalf("CreateUser(%q) error = %v, want nil", email, err)
	}
	return u
}

func TestAccountService_UpdateDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		createUser bool
		userID     string
		input      string
		wantErr    error
		wantName   string
	}{
		{
			name:       "有効な表示名は前後空白を除いて保存する",
			createUser: true,
			userID:     "",
			input:      "  月子  ",
			wantErr:    nil,
			wantName:   "月子",
		},
		{
			name:       "空の表示名はErrDisplayNameRequired",
			createUser: true,
			userID:     "",
			input:      "   ",
			wantErr:    service.ErrDisplayNameRequired,
			wantName:   "",
		},
		{
			name:       "長すぎる表示名はErrDisplayNameTooLong",
			createUser: true,
			userID:     "",
			input:      strings.Repeat("あ", service.MaxDisplayNameLength+1),
			wantErr:    service.ErrDisplayNameTooLong,
			wantName:   "",
		},
		{
			name:       "存在しないユーザーはErrUserNotFound",
			createUser: false,
			userID:     "missing",
			input:      "x",
			wantErr:    service.ErrUserNotFound,
			wantName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, users, _, _ := newAccountService(nil)
			ctx := context.Background()
			userID := tt.userID
			if tt.createUser {
				userID = createTestUser(t, users).ID
			}

			got, err := svc.UpdateDisplayName(ctx, userID, tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdateDisplayName(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateDisplayName(%q) error = %v, want nil", tt.input, err)
			}
			if got.DisplayName != tt.wantName {
				t.Errorf("UpdateDisplayName(%q) display_name = %q, want %q",
					tt.input, got.DisplayName, tt.wantName)
			}
		})
	}
}

func TestAccountService_Export(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		objects         storage.ObjectStore
		seed            func(t *testing.T, ctx context.Context, users *memauth.Store, diaries *memdiary.Store, gallery *memgallery.Store) string
		wantErr         error
		wantDiaries     int
		wantDiaryBody   string
		wantImages      int
		wantURLNonEmpty bool
	}{
		{
			name:    "論理削除済み日記を除外し画像URLを付与する",
			objects: memobject.New(),
			seed: func(t *testing.T, ctx context.Context, users *memauth.Store, diaries *memdiary.Store, gallery *memgallery.Store) string {
				u := createTestUser(t, users)
				_, _, err := diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
					UserID: u.ID, ClientID: "c1", BodyText: "live", CreatedAt: testNow,
				})
				if err != nil {
					t.Fatalf("UpsertEntry(live) error = %v, want nil", err)
				}
				dead, _, err := diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
					UserID: u.ID, ClientID: "c2", BodyText: "dead", CreatedAt: testNow,
				})
				if err != nil {
					t.Fatalf("UpsertEntry(dead) error = %v, want nil", err)
				}
				if err := diaries.SoftDeleteEntry(ctx, u.ID, dead.ID); err != nil {
					t.Fatalf("SoftDeleteEntry(%q) error = %v, want nil", dead.ID, err)
				}
				_, err = gallery.InsertImage(ctx, repository.InsertGalleryParams{
					UserID: u.ID, ObjectKey: "gallery/" + u.ID + "/k1", Width: 1, Height: 1, Theme: "pink",
				})
				if err != nil {
					t.Fatalf("InsertImage() error = %v, want nil", err)
				}
				return u.ID
			},
			wantErr:         nil,
			wantDiaries:     1,
			wantDiaryBody:   "live",
			wantImages:      1,
			wantURLNonEmpty: true,
		},
		{
			name:    "オブジェクトストレージ未設定なら画像はメタデータのみ",
			objects: nil,
			seed: func(t *testing.T, ctx context.Context, users *memauth.Store, diaries *memdiary.Store, gallery *memgallery.Store) string {
				u := createTestUser(t, users)
				_, err := gallery.InsertImage(ctx, repository.InsertGalleryParams{
					UserID: u.ID, ObjectKey: "gallery/" + u.ID + "/k1", Width: 1, Height: 1, Theme: "pink",
				})
				if err != nil {
					t.Fatalf("InsertImage() error = %v, want nil", err)
				}
				return u.ID
			},
			wantErr:         nil,
			wantDiaries:     0,
			wantDiaryBody:   "",
			wantImages:      1,
			wantURLNonEmpty: false,
		},
		{
			name:    "存在しないユーザーはErrUserNotFound",
			objects: memobject.New(),
			seed: func(_ *testing.T, _ context.Context, _ *memauth.Store, _ *memdiary.Store, _ *memgallery.Store) string {
				return "missing"
			},
			wantErr:         service.ErrUserNotFound,
			wantDiaries:     0,
			wantDiaryBody:   "",
			wantImages:      0,
			wantURLNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, users, diaries, gallery := newAccountService(tt.objects)
			ctx := context.Background()
			userID := tt.seed(t, ctx, users, diaries, gallery)

			export, err := svc.Export(ctx, userID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Export(%q) error = %v, want %v", userID, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Export(%q) error = %v, want nil", userID, err)
			}
			if len(export.Diaries) != tt.wantDiaries {
				t.Errorf("Export() returned %d diaries, want %d (tombstones must be excluded)",
					len(export.Diaries), tt.wantDiaries)
			}
			if tt.wantDiaries > 0 && export.Diaries[0].BodyText != tt.wantDiaryBody {
				t.Errorf("Export() first diary body_text = %q, want %q",
					export.Diaries[0].BodyText, tt.wantDiaryBody)
			}
			if len(export.Images) != tt.wantImages {
				t.Errorf("Export() returned %d images, want %d", len(export.Images), tt.wantImages)
			}
			if tt.wantImages > 0 {
				if got := export.Images[0].URL != ""; got != tt.wantURLNonEmpty {
					t.Errorf("Export() first image has a presigned URL = %t, want %t (url: %q)",
						got, tt.wantURLNonEmpty, export.Images[0].URL)
				}
			}
			if !export.ExportedAt.Equal(testNow.UTC()) {
				t.Errorf("Export() exported_at = %s, want %s", export.ExportedAt, testNow.UTC())
			}
		})
	}
}

func TestAccountService_DeleteAccount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		withStorage bool
	}{
		{
			name:        "オブジェクトを削除しユーザーを消す",
			withStorage: true,
		},
		{
			name:        "ストレージ未設定でも成功する",
			withStorage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var objects *memobject.Store
			var store storage.ObjectStore
			if tt.withStorage {
				objects = memobject.New()
				store = objects
			}
			svc, users, _, gallery := newAccountService(store)
			ctx := context.Background()
			u := createTestUser(t, users)

			var key string
			if tt.withStorage {
				key = "gallery/" + u.ID + "/k1"
				_, err := gallery.InsertImage(ctx, repository.InsertGalleryParams{
					UserID: u.ID, ObjectKey: key, Width: 1, Height: 1, Theme: "pink",
				})
				if err != nil {
					t.Fatalf("InsertImage(%q) error = %v, want nil", key, err)
				}
			}

			if err := svc.DeleteAccount(ctx, u.ID); err != nil {
				t.Fatalf("DeleteAccount(%q) error = %v, want nil", u.ID, err)
			}

			if _, err := users.GetUserByID(ctx, u.ID); !errors.Is(err, repository.ErrNotFound) {
				t.Errorf("GetUserByID(%q) after delete error = %v, want %v",
					u.ID, err, repository.ErrNotFound)
			}
			if tt.withStorage {
				if diff := cmp.Diff([]string{key}, objects.RemovedKeys()); diff != "" {
					t.Errorf("removed object keys mismatch (-want +got):\n%s", diff)
				}
			}

			// The row is gone, so a re-delete is not-found.
			if err := svc.DeleteAccount(ctx, u.ID); !errors.Is(err, service.ErrUserNotFound) {
				t.Errorf("second DeleteAccount(%q) error = %v, want %v",
					u.ID, err, service.ErrUserNotFound)
			}
		})
	}
}
