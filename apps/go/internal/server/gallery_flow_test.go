package server_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/handler"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/server"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

// galleryImage mirrors the JSON shape of the gallery image response for decoding.
type galleryImage struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	URLExpiresAt string `json:"url_expires_at"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Theme        string `json:"theme"`
	CreatedAt    string `json:"created_at"`
}

type galleryUploadURL struct {
	ObjectKey string            `json:"object_key"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt string            `json:"expires_at"`
	MaxSize   int64             `json:"max_size"`
}

type galleryList struct {
	Items      []galleryImage `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}

// fakeObjectStore is an in-memory storage.ObjectStore for the flow tests. It
// records presigned URLs deterministically and lets a test "simulate" the direct
// client upload by seeding an object with put(), then assert on Remove via
// wasRemoved().
type fakeObjectStore struct {
	mu      sync.Mutex
	objects map[string]storage.ObjectInfo
	removed []string
}

func newFakeObjectStore() *fakeObjectStore {
	return &fakeObjectStore{objects: make(map[string]storage.ObjectInfo)}
}

func (f *fakeObjectStore) EnsureBucket(context.Context) error { return nil }

func (f *fakeObjectStore) PresignPut(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://minio.test/runa-gallery/" + key + "?put=1", nil
}

func (f *fakeObjectStore) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://minio.test/runa-gallery/" + key + "?get=1", nil
}

func (f *fakeObjectStore) Stat(_ context.Context, key string) (storage.ObjectInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if info, ok := f.objects[key]; ok {
		return info, nil
	}
	return storage.ObjectInfo{}, storage.ErrObjectNotFound
}

func (f *fakeObjectStore) Remove(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, key)
	delete(f.objects, key)
	return nil
}

func (f *fakeObjectStore) put(key string, info storage.ObjectInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = info
}

func (f *fakeObjectStore) wasRemoved(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range f.removed {
		if k == key {
			return true
		}
	}
	return false
}

// newGalleryFlowRouter wires auth + gallery over in-memory stores. Object removal
// runs synchronously (WithBackgroundRunner) so the deletion assertion is
// deterministic. It returns the fake store so a test can seed/inspect objects.
func newGalleryFlowRouter() (http.Handler, *fakeObjectStore) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	issuer := auth.NewTokenIssuer("test-secret", time.Minute)
	authSvc := service.NewAuthService(service.AuthConfig{
		Store:          memauth.New(),
		Issuer:         issuer,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	ah := handler.NewAuth(authSvc, logger)

	objects := newFakeObjectStore()
	gs := service.NewGalleryService(memgallery.New(), objects, service.GalleryConfig{
		UploadURLTTL:        15 * time.Minute,
		ViewURLTTL:          60 * time.Minute,
		MaxUploadBytes:      10 * 1024 * 1024,
		AllowedContentTypes: []string{"image/jpeg", "image/png", "image/webp", "image/heic"},
	}, nil, service.WithBackgroundRunner(func(f func()) { f() }))
	gh := handler.NewGallery(gs, logger)

	r := server.New(server.Deps{
		Health:         handler.NewHealth(service.NewHealth(), logger),
		Auth:           ah,
		Gallery:        gh,
		RequireAuth:    auth.RequireAuth(issuer, ah.Unauthorized),
		AuthRateLimit:  auth.NewRateLimiter(100, time.Minute).Middleware(ah.RateLimited),
		AllowedOrigins: []string{"*"},
		Logger:         logger,
	})
	return r, objects
}

// requestUploadURL asks for an upload URL and returns the decoded response.
func requestUploadURL(t *testing.T, r http.Handler, token, contentType string, size int64) galleryUploadURL {
	t.Helper()
	body := `{"content_type":"` + contentType + `","size":` + strconv.FormatInt(size, 10) + `}`
	res := do(t, r, http.MethodPost, "/api/v1/gallery/upload-url", token, body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("upload-url status = %d, want 200", res.StatusCode)
	}
	var out galleryUploadURL
	decode(t, res, &out)
	return out
}

// TestGalleryFlow runs the full lifecycle through the real router: upload-url →
// (simulated direct upload) → register → list → get → delete (with async object
// removal), asserting the presigned URLs flow through and the object is removed.
func TestGalleryFlow(t *testing.T) {
	r, objects := newGalleryFlowRouter()
	token := signupToken(t, r, "gallery@example.com")

	// 1. Request an upload URL. object_key is namespaced to the caller.
	up := requestUploadURL(t, r, token, "image/jpeg", 1000)
	if !strings.HasPrefix(up.ObjectKey, "gallery/") {
		t.Fatalf("object_key = %q, want gallery/ prefix", up.ObjectKey)
	}
	if up.UploadURL == "" || up.Method != http.MethodPut || up.Headers["Content-Type"] != "image/jpeg" {
		t.Fatalf("upload target = %+v, unexpected", up)
	}
	if up.MaxSize <= 0 {
		t.Fatalf("max_size = %d, want > 0", up.MaxSize)
	}

	// 2. Simulate the client PUTting the bytes straight to storage.
	objects.put(up.ObjectKey, storage.ObjectInfo{Size: 1000, ContentType: "image/jpeg"})

	// 3. Register metadata.
	res := do(t, r, http.MethodPost, "/api/v1/gallery", token,
		`{"object_key":"`+up.ObjectKey+`","width":800,"height":600,"theme":"pink"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", res.StatusCode)
	}
	var created galleryImage
	decode(t, res, &created)
	if created.ID == "" || created.URL == "" || created.Theme != "pink" || created.Width != 800 {
		t.Fatalf("created image = %+v, unexpected", created)
	}

	// 4. List shows exactly one image with a view URL, no next page.
	res = do(t, r, http.MethodGet, "/api/v1/gallery", token, "")
	var list galleryList
	decode(t, res, &list)
	if len(list.Items) != 1 || list.NextCursor != nil {
		t.Fatalf("list = %+v, want 1 item and no cursor", list)
	}
	if list.Items[0].URL == "" {
		t.Fatalf("list item has no view URL")
	}

	// 5. Get by id returns a fresh view URL.
	res = do(t, r, http.MethodGet, "/api/v1/gallery/"+created.ID, token, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", res.StatusCode)
	}

	// 6. Delete soft-deletes and removes the stored object (runner is sync here).
	res = do(t, r, http.MethodDelete, "/api/v1/gallery/"+created.ID, token, "")
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", res.StatusCode)
	}
	if !objects.wasRemoved(up.ObjectKey) {
		t.Fatalf("object %q was not removed from storage on delete", up.ObjectKey)
	}
	res = do(t, r, http.MethodGet, "/api/v1/gallery", token, "")
	decode(t, res, &list)
	if len(list.Items) != 0 {
		t.Fatalf("list after delete = %d, want 0", len(list.Items))
	}
}

// TestGalleryUploadURLValidation exercises the upload-constraint checks.
func TestGalleryUploadURLValidation(t *testing.T) {
	r, _ := newGalleryFlowRouter()
	token := signupToken(t, r, "gallery-validate@example.com")

	cases := []struct {
		name string
		body string
	}{
		{"empty content type", `{"content_type":"","size":100}`},
		{"non-positive size", `{"content_type":"image/jpeg","size":0}`},
		{"disallowed content type", `{"content_type":"application/pdf","size":100}`},
		{"too large", `{"content_type":"image/jpeg","size":99999999999}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := do(t, r, http.MethodPost, "/api/v1/gallery/upload-url", token, tc.body)
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", res.StatusCode)
			}
		})
	}
}

// TestGalleryRegisterAuthorization proves the object_key namespace is enforced and
// registration verifies the real object exists.
func TestGalleryRegisterAuthorization(t *testing.T) {
	r, objects := newGalleryFlowRouter()
	tokenA := signupToken(t, r, "gallery-owner@example.com")
	tokenB := signupToken(t, r, "gallery-stranger@example.com")

	up := requestUploadURL(t, r, tokenA, "image/jpeg", 1000)
	objects.put(up.ObjectKey, storage.ObjectInfo{Size: 1000, ContentType: "image/jpeg"})

	// A stranger registering A's object_key is a 404 (namespace mismatch).
	res := do(t, r, http.MethodPost, "/api/v1/gallery", tokenB,
		`{"object_key":"`+up.ObjectKey+`","width":10,"height":10,"theme":"pink"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-user register status = %d, want 404", res.StatusCode)
	}

	// A's own key but the object was never uploaded → 400.
	upB := requestUploadURL(t, r, tokenB, "image/jpeg", 1000)
	res = do(t, r, http.MethodPost, "/api/v1/gallery", tokenB,
		`{"object_key":"`+upB.ObjectKey+`","width":10,"height":10,"theme":"pink"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing-object register status = %d, want 400", res.StatusCode)
	}

	// Invalid theme → 400.
	res = do(t, r, http.MethodPost, "/api/v1/gallery", tokenA,
		`{"object_key":"`+up.ObjectKey+`","width":10,"height":10,"theme":"rainbow"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad-theme register status = %d, want 400", res.StatusCode)
	}
}

// TestGalleryIsScoped confirms a stranger can never see, fetch or delete another
// user's image.
func TestGalleryIsScoped(t *testing.T) {
	r, objects := newGalleryFlowRouter()
	tokenA := signupToken(t, r, "gallery-a2@example.com")
	tokenB := signupToken(t, r, "gallery-b2@example.com")

	up := requestUploadURL(t, r, tokenA, "image/png", 500)
	objects.put(up.ObjectKey, storage.ObjectInfo{Size: 500, ContentType: "image/png"})
	res := do(t, r, http.MethodPost, "/api/v1/gallery", tokenA,
		`{"object_key":"`+up.ObjectKey+`","width":100,"height":200,"theme":"monotone"}`)
	var created galleryImage
	decode(t, res, &created)

	// B's list is empty.
	res = do(t, r, http.MethodGet, "/api/v1/gallery", tokenB, "")
	var list galleryList
	decode(t, res, &list)
	if len(list.Items) != 0 {
		t.Fatalf("stranger list = %d items, want 0", len(list.Items))
	}
	// B cannot get or delete A's image.
	res = do(t, r, http.MethodGet, "/api/v1/gallery/"+created.ID, tokenB, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("stranger get status = %d, want 404", res.StatusCode)
	}
	res = do(t, r, http.MethodDelete, "/api/v1/gallery/"+created.ID, tokenB, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("stranger delete status = %d, want 404", res.StatusCode)
	}
	if objects.wasRemoved(up.ObjectKey) {
		t.Fatalf("stranger delete must not remove the owner's object")
	}
}
