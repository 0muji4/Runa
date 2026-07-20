package server_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0muji4/Runa/apps/go/internal/storage"
)

func requestUploadURL(t *testing.T, r http.Handler, token, contentType string, size int64) galleryUploadURL {
	t.Helper()
	body := `{"content_type":"` + contentType + `","size":` + strconv.FormatInt(size, 10) + `}`
	res := do(t, r, http.MethodPost, "/api/v1/gallery/upload-url", token, body)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var out galleryUploadURL
	decode(t, res, &out)
	return out
}

func TestGalleryFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "gallery@example.com")

	up := requestUploadURL(t, env.r, token, "image/jpeg", 1000)
	assert.True(t, strings.HasPrefix(up.ObjectKey, "gallery/"))
	assert.NotEmpty(t, up.UploadURL)
	assert.Equal(t, http.MethodPut, up.Method)
	assert.Equal(t, "image/jpeg", up.Headers["Content-Type"])
	assert.Positive(t, up.MaxSize)

	// クライアントの直アップロードを模してオブジェクトをseedする。
	env.objects.Put(up.ObjectKey, storage.ObjectInfo{Size: 1000, ContentType: "image/jpeg"})

	res := do(t, env.r, http.MethodPost, "/api/v1/gallery", token,
		`{"object_key":"`+up.ObjectKey+`","width":800,"height":600,"theme":"pink"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	var created galleryImage
	decode(t, res, &created)
	assert.NotEmpty(t, created.ID)
	assert.NotEmpty(t, created.URL)
	assert.Equal(t, "pink", created.Theme)
	assert.Equal(t, 800, created.Width)

	res = do(t, env.r, http.MethodGet, "/api/v1/gallery", token, "")
	var list galleryList
	decode(t, res, &list)
	require.Len(t, list.Items, 1)
	assert.Nil(t, list.NextCursor)
	assert.NotEmpty(t, list.Items[0].URL)

	res = do(t, env.r, http.MethodGet, "/api/v1/gallery/"+created.ID, token, "")
	require.Equal(t, http.StatusOK, res.StatusCode)
	var got galleryImage
	decode(t, res, &got)
	assert.Equal(t, created.ID, got.ID)
	assert.NotEmpty(t, got.URL)

	res = do(t, env.r, http.MethodDelete, "/api/v1/gallery/"+created.ID, token, "")
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res.Body.Close()
	assert.True(t, env.objects.Removed(up.ObjectKey))

	res = do(t, env.r, http.MethodGet, "/api/v1/gallery", token, "")
	decode(t, res, &list)
	assert.Empty(t, list.Items)
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
			require.Equal(t, http.StatusBadRequest, res.StatusCode)
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
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	res.Body.Close()

	// 自分のキーでもオブジェクト未アップロードなら400。
	upB := requestUploadURL(t, env.r, tokenB, "image/jpeg", 1000)
	res = do(t, env.r, http.MethodPost, "/api/v1/gallery", tokenB,
		`{"object_key":"`+upB.ObjectKey+`","width":10,"height":10,"theme":"pink"}`)
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	res.Body.Close()

	// 不正なthemeは400。
	res = do(t, env.r, http.MethodPost, "/api/v1/gallery", tokenA,
		`{"object_key":"`+upA.ObjectKey+`","width":10,"height":10,"theme":"rainbow"}`)
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
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
	require.Equal(t, http.StatusCreated, res.StatusCode)
	var created galleryImage
	decode(t, res, &created)

	// 他人のlistは空。
	res = do(t, env.r, http.MethodGet, "/api/v1/gallery", tokenB, "")
	var list galleryList
	decode(t, res, &list)
	assert.Empty(t, list.Items)

	// 他人は取得も削除もできず、オブジェクトも消えない。
	res = do(t, env.r, http.MethodGet, "/api/v1/gallery/"+created.ID, tokenB, "")
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodDelete, "/api/v1/gallery/"+created.ID, tokenB, "")
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	res.Body.Close()
	assert.False(t, env.objects.Removed(up.ObjectKey))
}
