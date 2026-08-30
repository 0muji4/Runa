package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

func TestAccountFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	ctx := context.Background()

	res := do(t, env.r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"account@example.com","password":"password123","display_name":"Account"}`)
	checkStatus(t, res, http.StatusCreated)
	var signed tokens
	decode(t, res, &signed)
	token, userID, refresh := signed.AccessToken, signed.User.ID, signed.RefreshToken

	_, _, err := env.diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: userID, ClientID: "client-1", BodyText: "月がきれい", CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seeding a diary entry: UpsertEntry() error = %v, want nil", err)
	}
	objectKey := "gallery/" + userID + "/img-1"
	_, err = env.gallery.InsertImage(ctx, repository.InsertGalleryParams{
		UserID: userID, ObjectKey: objectKey, Width: 100, Height: 200, Theme: "pink",
	})
	if err != nil {
		t.Fatalf("seeding a gallery image: InsertImage() error = %v, want nil", err)
	}
	env.objects.Put(objectKey, storage.ObjectInfo{Size: 10, ContentType: "image/png"})

	res = do(t, env.r, http.MethodPatch, "/api/v1/me", token, `{"display_name":"新しい名前"}`)
	checkStatus(t, res, http.StatusOK)
	var updated struct {
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &updated)
	if got, want := updated.DisplayName, "新しい名前"; got != want {
		t.Errorf("PATCH /api/v1/me display_name = %q, want %q", got, want)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/me", token, "")
	var me struct {
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &me)
	if got, want := me.DisplayName, "新しい名前"; got != want {
		t.Errorf("GET /api/v1/me display_name = %q, want %q (the update must persist)", got, want)
	}

	res = do(t, env.r, http.MethodPatch, "/api/v1/me", token, `{"display_name":"   "}`)
	checkStatus(t, res, http.StatusBadRequest)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/me/export", token, "")
	checkStatus(t, res, http.StatusOK)
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
	if export.SchemaVersion != 1 {
		t.Errorf("export schema_version = %d, want 1", export.SchemaVersion)
	}
	if got, want := export.User.DisplayName, "新しい名前"; got != want {
		t.Errorf("export user.display_name = %q, want %q", got, want)
	}
	if len(export.Diaries) != 1 {
		t.Fatalf("export returned %d diaries, want 1", len(export.Diaries))
	}
	if got, want := export.Diaries[0].BodyText, "月がきれい"; got != want {
		t.Errorf("export diary body_text = %q, want %q", got, want)
	}
	if len(export.Images) != 1 {
		t.Fatalf("export returned %d images, want 1", len(export.Images))
	}
	if export.Images[0].URL == nil {
		t.Error("export image url = nil, want a presigned URL")
	}

	// 削除はストアのオブジェクトもpurgeする。
	res = do(t, env.r, http.MethodDelete, "/api/v1/me", token, "")
	checkStatus(t, res, http.StatusNoContent)
	res.Body.Close()
	if !env.objects.Removed(objectKey) {
		t.Errorf("object %q was not purged after account deletion", objectKey)
	}

	// 削除後は保持していたアクセストークンも解決できない。
	res = do(t, env.r, http.MethodGet, "/api/v1/me", token, "")
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()

	// refreshトークンもcascadeで失効している。
	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+refresh+`"}`)
	checkStatus(t, res, http.StatusUnauthorized)
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
			checkStatus(t, res, http.StatusUnauthorized)
			res.Body.Close()
		})
	}
}
