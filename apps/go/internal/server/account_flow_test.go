package server_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/handler"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/server"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

// newAccountFlowRouter wires auth + account over shared in-memory stores. Object
// purge runs synchronously (WithAccountBackgroundRunner) so the deletion assertion
// is deterministic. It returns the fake object store and the diary/gallery stores
// so a test can seed data the account endpoints then aggregate/purge.
func newAccountFlowRouter() (http.Handler, *fakeObjectStore, *memdiary.Store, *memgallery.Store) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	issuer := auth.NewTokenIssuer("test-secret", time.Minute)
	users := memauth.New()
	diaries := memdiary.New()
	gallery := memgallery.New()
	objects := newFakeObjectStore()

	authSvc := service.NewAuthService(service.AuthConfig{
		Store:          users,
		Issuer:         issuer,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	ah := handler.NewAuth(authSvc, logger)

	accountSvc := service.NewAccountService(users, diaries, gallery, objects, service.AccountConfig{
		ExportURLTTL: time.Hour,
	}, nil, service.WithAccountBackgroundRunner(func(f func()) { f() }))
	acc := handler.NewAccount(accountSvc, logger)

	r := server.New(server.Deps{
		Health:         handler.NewHealth(service.NewHealth(), logger),
		Auth:           ah,
		Account:        acc,
		RequireAuth:    auth.RequireAuth(issuer, ah.Unauthorized),
		AuthRateLimit:  auth.NewRateLimiter(100, time.Minute).Middleware(ah.RateLimited),
		AllowedOrigins: []string{"*"},
		Logger:         logger,
	})
	return r, objects, diaries, gallery
}

// TestAccountFlow exercises the full account lifecycle through the real router:
// signup -> PATCH display name -> export (profile + diary + image) -> delete, and
// verifies deletion purges the stored object, revokes the refresh token (cascade)
// and locks out the still-held access token.
func TestAccountFlow(t *testing.T) {
	r, objects, diaries, gallery := newAccountFlowRouter()
	ctx := context.Background()

	// 1. Signup.
	res := do(t, r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"account@example.com","password":"password123","display_name":"Account"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("signup status = %d, want 201", res.StatusCode)
	}
	var signed tokens
	decode(t, res, &signed)
	token, userID, refresh := signed.AccessToken, signed.User.ID, signed.RefreshToken

	// Seed one diary entry and one gallery image (+ stored object) to export/purge.
	if _, _, err := diaries.UpsertEntry(ctx, repository.UpsertDiaryParams{
		UserID: userID, ClientID: "client-1", BodyText: "月がきれい", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed diary: %v", err)
	}
	objectKey := "gallery/" + userID + "/img-1"
	if _, err := gallery.InsertImage(ctx, repository.InsertGalleryParams{
		UserID: userID, ObjectKey: objectKey, Width: 100, Height: 200, Theme: "pink",
	}); err != nil {
		t.Fatalf("seed image: %v", err)
	}
	objects.put(objectKey, storage.ObjectInfo{Size: 10, ContentType: "image/png"})

	// 2. PATCH /me updates the display name.
	res = do(t, r, http.MethodPatch, "/api/v1/me", token, `{"display_name":"新しい名前"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, want 200", res.StatusCode)
	}
	var updated struct {
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &updated)
	if updated.DisplayName != "新しい名前" {
		t.Errorf("patched display_name = %q, want 新しい名前", updated.DisplayName)
	}

	// GET /me reflects the update.
	res = do(t, r, http.MethodGet, "/api/v1/me", token, "")
	var me struct {
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &me)
	if me.DisplayName != "新しい名前" {
		t.Errorf("/me display_name = %q, want 新しい名前", me.DisplayName)
	}

	// 3. An empty display name is rejected.
	res = do(t, r, http.MethodPatch, "/api/v1/me", token, `{"display_name":"   "}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("patch empty status = %d, want 400", res.StatusCode)
	}

	// 4. Export returns the profile, the live diary and the image with a URL.
	res = do(t, r, http.MethodGet, "/api/v1/me/export", token, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("export status = %d, want 200", res.StatusCode)
	}
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
		t.Errorf("schema_version = %d, want 1", export.SchemaVersion)
	}
	if export.User.DisplayName != "新しい名前" {
		t.Errorf("export user = %q, want 新しい名前", export.User.DisplayName)
	}
	if len(export.Diaries) != 1 || export.Diaries[0].BodyText != "月がきれい" {
		t.Errorf("export diaries = %+v, want one live entry", export.Diaries)
	}
	if len(export.Images) != 1 || export.Images[0].URL == nil {
		t.Errorf("export images = %+v, want one image with a URL", export.Images)
	}

	// 5. Delete the account.
	res = do(t, r, http.MethodDelete, "/api/v1/me", token, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", res.StatusCode)
	}
	if !objects.wasRemoved(objectKey) {
		t.Fatalf("object %q was not purged on account deletion", objectKey)
	}

	// 6. The still-held access token no longer resolves a user.
	res = do(t, r, http.MethodGet, "/api/v1/me", token, "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/me after delete status = %d, want 401", res.StatusCode)
	}

	// 7. The refresh token was revoked by the cascade.
	res = do(t, r, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"`+refresh+`"}`)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("refresh after delete status = %d, want 401", res.StatusCode)
	}
}

// TestAccountRequiresAuth confirms every account endpoint rejects an unauthenticated
// request.
func TestAccountRequiresAuth(t *testing.T) {
	r, _, _, _ := newAccountFlowRouter()
	cases := []struct {
		method, path, body string
	}{
		{http.MethodPatch, "/api/v1/me", `{"display_name":"x"}`},
		{http.MethodGet, "/api/v1/me/export", ""},
		{http.MethodDelete, "/api/v1/me", ""},
	}
	for _, tc := range cases {
		res := do(t, r, tc.method, tc.path, "", tc.body)
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", tc.method, tc.path, res.StatusCode)
		}
	}
}
