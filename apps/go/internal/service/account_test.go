package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
	"github.com/0muji4/Runa/apps/go/internal/storage/memobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T, users *memauth.Store) repository.User {
	t.Helper()
	email := "u@example.com"
	u, err := users.CreateUser(context.Background(), repository.CreateUserParams{
		Email: &email, AuthProvider: "email", DisplayName: "U",
	})
	require.NoError(t, err)
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
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, got.DisplayName)
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
				require.NoError(t, err)
				dead, _, err := diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
					UserID: u.ID, ClientID: "c2", BodyText: "dead", CreatedAt: testNow,
				})
				require.NoError(t, err)
				require.NoError(t, diaries.SoftDeleteEntry(ctx, u.ID, dead.ID))
				_, err = gallery.InsertImage(ctx, repository.InsertGalleryParams{
					UserID: u.ID, ObjectKey: "gallery/" + u.ID + "/k1", Width: 1, Height: 1, Theme: "pink",
				})
				require.NoError(t, err)
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
				require.NoError(t, err)
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
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, export.Diaries, tt.wantDiaries)
			if tt.wantDiaries > 0 {
				assert.Equal(t, tt.wantDiaryBody, export.Diaries[0].BodyText)
			}
			assert.Len(t, export.Images, tt.wantImages)
			if tt.wantImages > 0 {
				assert.Equal(t, tt.wantURLNonEmpty, export.Images[0].URL != "")
			}
			assert.True(t, export.ExportedAt.Equal(testNow.UTC()))
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
				require.NoError(t, err)
			}

			require.NoError(t, svc.DeleteAccount(ctx, u.ID))

			_, err := users.GetUserByID(ctx, u.ID)
			assert.ErrorIs(t, err, repository.ErrNotFound)
			if tt.withStorage {
				assert.Equal(t, []string{key}, objects.RemovedKeys())
			}

			// The row is gone, so a re-delete is not-found.
			assert.ErrorIs(t, svc.DeleteAccount(ctx, u.ID), service.ErrUserNotFound)
		})
	}
}
