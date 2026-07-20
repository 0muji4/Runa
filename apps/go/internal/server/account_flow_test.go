package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

func TestAccountFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	ctx := context.Background()

	res := do(t, env.r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"account@example.com","password":"password123","display_name":"Account"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	var signed tokens
	decode(t, res, &signed)
	token, userID, refresh := signed.AccessToken, signed.User.ID, signed.RefreshToken

	_, _, err := env.diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: userID, ClientID: "client-1", BodyText: "月がきれい", CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	objectKey := "gallery/" + userID + "/img-1"
	_, err = env.gallery.InsertImage(ctx, repository.InsertGalleryParams{
		UserID: userID, ObjectKey: objectKey, Width: 100, Height: 200, Theme: "pink",
	})
	require.NoError(t, err)
	env.objects.Put(objectKey, storage.ObjectInfo{Size: 10, ContentType: "image/png"})

	res = do(t, env.r, http.MethodPatch, "/api/v1/me", token, `{"display_name":"新しい名前"}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var updated struct {
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &updated)
	assert.Equal(t, "新しい名前", updated.DisplayName)

	res = do(t, env.r, http.MethodGet, "/api/v1/me", token, "")
	var me struct {
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &me)
	assert.Equal(t, "新しい名前", me.DisplayName)

	res = do(t, env.r, http.MethodPatch, "/api/v1/me", token, `{"display_name":"   "}`)
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/me/export", token, "")
	require.Equal(t, http.StatusOK, res.StatusCode)
	var export struct {
		SchemaVersion int `json:"schema_version"`
		User          struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
		Diaries []struct {
			BodyText string `json:"body_text"`
		} `json:"diaries"`
		Images []struct {
			ObjectKey string  `json:"object_key"`
			URL       *string `json:"url"`
		} `json:"images"`
	}
	decode(t, res, &export)
	assert.Equal(t, 1, export.SchemaVersion)
	assert.Equal(t, "新しい名前", export.User.DisplayName)
	require.Len(t, export.Diaries, 1)
	assert.Equal(t, "月がきれい", export.Diaries[0].BodyText)
	require.Len(t, export.Images, 1)
	assert.NotNil(t, export.Images[0].URL)

	// 削除はストアのオブジェクトもpurgeする。
	res = do(t, env.r, http.MethodDelete, "/api/v1/me", token, "")
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res.Body.Close()
	assert.True(t, env.objects.Removed(objectKey))

	// 削除後は保持していたアクセストークンも解決できない。
	res = do(t, env.r, http.MethodGet, "/api/v1/me", token, "")
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()

	// refreshトークンもcascadeで失効している。
	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+refresh+`"}`)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()
}

func TestAccountRequiresAuth(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	tests := []struct {
		name, method, path, body string
	}{
		{
			name:   "patch me",
			method: http.MethodPatch,
			path:   "/api/v1/me",
			body:   `{"display_name":"x"}`,
		},
		{
			name:   "export",
			method: http.MethodGet,
			path:   "/api/v1/me/export",
			body:   "",
		},
		{
			name:   "delete me",
			method: http.MethodDelete,
			path:   "/api/v1/me",
			body:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := do(t, env.r, tt.method, tt.path, "", tt.body)
			require.Equal(t, http.StatusUnauthorized, res.StatusCode)
			res.Body.Close()
		})
	}
}
