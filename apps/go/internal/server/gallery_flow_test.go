package server_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/storage"
)

func requestUploadURL(t *testing.T, r http.Handler, token, contentType string, size int64) galleryUploadURL {
	t.Helper()
	body := `{"content_type":"` + contentType + `","size":` + strconv.FormatInt(size, 10) + `}`
	res := do(t, r, http.MethodPost, "/api/v1/gallery/upload-url", token, body)
	checkStatus(t, res, http.StatusOK)
	var out galleryUploadURL
	decode(t, res, &out)
	return out
}

func TestGalleryFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "gallery@example.com")

	up := requestUploadURL(t, env.r, token, "image/jpeg", 1000)
	if !strings.HasPrefix(up.ObjectKey, "gallery/") {
		t.Errorf("upload object_key = %q, want prefix %q", up.ObjectKey, "gallery/")
	}
	if up.UploadURL == "" {
		t.Error("upload_url is empty, want a presigned URL")
	}
	if up.Method != http.MethodPut {
		t.Errorf("upload method = %q, want %q", up.Method, http.MethodPut)
	}
	if got, want := up.Headers["Content-Type"], "image/jpeg"; got != want {
		t.Errorf("upload Content-Type header = %q, want %q", got, want)
	}
	if up.MaxSize <= 0 {
		t.Errorf("upload max_size = %d, want a positive limit", up.MaxSize)
	}

	// クライアントの直アップロードを模してオブジェクトをseedする。
	env.objects.Put(up.ObjectKey, storage.ObjectInfo{Size: 1000, ContentType: "image/jpeg"})

	res := do(t, env.r, http.MethodPost, "/api/v1/gallery", token,
		`{"object_key":"`+up.ObjectKey+`","width":800,"height":600,"theme":"pink"}`)
	checkStatus(t, res, http.StatusCreated)
	var created galleryImage
	decode(t, res, &created)
	if created.ID == "" {
		t.Error("registered image id is empty, want a generated id")
	}
	if created.URL == "" {
		t.Error("registered image url is empty, want a presigned URL")
	}
	if got, want := created.Theme, "pink"; got != want {
		t.Errorf("registered image theme = %q, want %q", got, want)
	}
	if created.Width != 800 {
		t.Errorf("registered image width = %d, want 800", created.Width)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/gallery", token, "")
	var list galleryList
	decode(t, res, &list)
	if len(list.Items) != 1 {
		t.Fatalf("GET /api/v1/gallery returned %d items, want 1", len(list.Items))
	}
	if list.NextCursor != nil {
		t.Errorf("GET /api/v1/gallery next_cursor = %q, want nil", *list.NextCursor)
	}
	if list.Items[0].URL == "" {
		t.Error("listed image url is empty, want a presigned URL")
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/gallery/"+created.ID, token, "")
	checkStatus(t, res, http.StatusOK)
	var got galleryImage
	decode(t, res, &got)
	if got.ID != created.ID {
		t.Errorf("GET /api/v1/gallery/{id} returned id %q, want %q", got.ID, created.ID)
	}
	if got.URL == "" {
		t.Error("fetched image url is empty, want a presigned URL")
	}

	res = do(t, env.r, http.MethodDelete, "/api/v1/gallery/"+created.ID, token, "")
	checkStatus(t, res, http.StatusNoContent)
	res.Body.Close()
	if !env.objects.Removed(up.ObjectKey) {
		t.Errorf("object %q was not purged after the image was deleted", up.ObjectKey)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/gallery", token, "")
	decode(t, res, &list)
	if len(list.Items) != 0 {
		t.Errorf("GET /api/v1/gallery after delete returned %d items, want 0", len(list.Items))
	}
}

func TestGalleryUploadURLValidation(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "gallery-validate@example.com")

	tests := []struct {
		name string
		body string
	}{
		{
			name: "content typeが空",
			body: `{"content_type":"","size":100}`,
		},
		{
			name: "sizeが非正",
			body: `{"content_type":"image/jpeg","size":0}`,
		},
		{
			name: "許可されないcontent type",
			body: `{"content_type":"application/pdf","size":100}`,
		},
		{
			name: "サイズ超過",
			body: `{"content_type":"image/jpeg","size":99999999999}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := do(t, env.r, http.MethodPost, "/api/v1/gallery/upload-url", token, tt.body)
			checkStatus(t, res, http.StatusBadRequest)
			res.Body.Close()
		})
	}
}

func TestGalleryRegisterAuthorization(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	tokenA := signupToken(t, env.r, "gallery-owner@example.com")
	tokenB := signupToken(t, env.r, "gallery-stranger@example.com")

	upA := requestUploadURL(t, env.r, tokenA, "image/jpeg", 1000)
	env.objects.Put(upA.ObjectKey, storage.ObjectInfo{Size: 1000, ContentType: "image/jpeg"})

	// 他人のobject_keyを登録しようとすると名前空間不一致で404。
	res := do(t, env.r, http.MethodPost, "/api/v1/gallery", tokenB,
		`{"object_key":"`+upA.ObjectKey+`","width":10,"height":10,"theme":"pink"}`)
	checkStatus(t, res, http.StatusNotFound)
	res.Body.Close()

	// 自分のキーでもオブジェクト未アップロードなら400。
	upB := requestUploadURL(t, env.r, tokenB, "image/jpeg", 1000)
	res = do(t, env.r, http.MethodPost, "/api/v1/gallery", tokenB,
		`{"object_key":"`+upB.ObjectKey+`","width":10,"height":10,"theme":"pink"}`)
	checkStatus(t, res, http.StatusBadRequest)
	res.Body.Close()

	// 不正なthemeは400。
	res = do(t, env.r, http.MethodPost, "/api/v1/gallery", tokenA,
		`{"object_key":"`+upA.ObjectKey+`","width":10,"height":10,"theme":"rainbow"}`)
	checkStatus(t, res, http.StatusBadRequest)
	res.Body.Close()
}

func TestGalleryIsScoped(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	tokenA := signupToken(t, env.r, "gallery-a2@example.com")
	tokenB := signupToken(t, env.r, "gallery-b2@example.com")

	up := requestUploadURL(t, env.r, tokenA, "image/png", 500)
	env.objects.Put(up.ObjectKey, storage.ObjectInfo{Size: 500, ContentType: "image/png"})
	res := do(t, env.r, http.MethodPost, "/api/v1/gallery", tokenA,
		`{"object_key":"`+up.ObjectKey+`","width":100,"height":200,"theme":"monotone"}`)
	checkStatus(t, res, http.StatusCreated)
	var created galleryImage
	decode(t, res, &created)

	// 他人のlistは空。
	res = do(t, env.r, http.MethodGet, "/api/v1/gallery", tokenB, "")
	var list galleryList
	decode(t, res, &list)
	if len(list.Items) != 0 {
		t.Errorf("a stranger's gallery returned %d items, want 0 (images are owner-scoped)",
			len(list.Items))
	}

	// 他人は取得も削除もできず、オブジェクトも消えない。
	res = do(t, env.r, http.MethodGet, "/api/v1/gallery/"+created.ID, tokenB, "")
	checkStatus(t, res, http.StatusNotFound)
	res.Body.Close()

	res = do(t, env.r, http.MethodDelete, "/api/v1/gallery/"+created.ID, tokenB, "")
	checkStatus(t, res, http.StatusNotFound)
	res.Body.Close()
	if env.objects.Removed(up.ObjectKey) {
		t.Errorf("object %q was purged by a stranger's delete; it must survive", up.ObjectKey)
	}
}
