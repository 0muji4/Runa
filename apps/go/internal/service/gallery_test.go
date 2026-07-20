package service_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
	"github.com/0muji4/Runa/apps/go/internal/storage/memobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func galleryConfig() service.GalleryConfig {
	return service.GalleryConfig{
		UploadURLTTL:        15 * time.Minute,
		ViewURLTTL:          time.Hour,
		MaxUploadBytes:      1024,
		AllowedContentTypes: []string{"image/jpeg", "image/png"},
	}
}

func TestGalleryService_CreateUploadURL(t *testing.T) {
	t.Parallel()

	cfg := galleryConfig()
	tests := []struct {
		name        string
		noStorage   bool
		contentType string
		size        int64
		wantErr     error
	}{
		{
			name:        "許可された種別に署名付きPUT URLを発行する",
			noStorage:   false,
			contentType: "image/jpeg",
			size:        500,
			wantErr:     nil,
		},
		{
			name:        "空のcontent typeはErrContentTypeNotAllowed",
			noStorage:   false,
			contentType: "",
			size:        500,
			wantErr:     service.ErrContentTypeNotAllowed,
		},
		{
			name:        "許可外のcontent typeはErrContentTypeNotAllowed",
			noStorage:   false,
			contentType: "image/gif",
			size:        500,
			wantErr:     service.ErrContentTypeNotAllowed,
		},
		{
			name:        "上限超過のアップロードはErrUploadTooLarge",
			noStorage:   false,
			contentType: "image/jpeg",
			size:        cfg.MaxUploadBytes + 1,
			wantErr:     service.ErrUploadTooLarge,
		},
		{
			name:        "サイズ0は許容される（下限チェックなし）",
			noStorage:   false,
			contentType: "image/jpeg",
			size:        0,
			wantErr:     nil,
		},
		{
			name:        "ストレージ未設定はErrStorageUnavailable",
			noStorage:   true,
			contentType: "image/jpeg",
			size:        500,
			wantErr:     service.ErrStorageUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var objects storage.ObjectStore
			if !tt.noStorage {
				objects = memobject.New()
			}
			svc, _ := newGalleryService(objects, cfg)

			target, err := svc.CreateUploadURL(context.Background(), userA, tt.contentType, tt.size)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(target.ObjectKey, "gallery/"+userA+"/"), "object key %q not in the caller's namespace", target.ObjectKey)
			assert.NotEmpty(t, target.URL)
			assert.Equal(t, tt.contentType, target.ContentType)
			assert.Equal(t, cfg.MaxUploadBytes, target.MaxSize)
			assert.True(t, target.ExpiresAt.Equal(testNow.Add(cfg.UploadURLTTL)))
		})
	}
}

func TestGalleryService_RegisterImage(t *testing.T) {
	t.Parallel()

	cfg := galleryConfig()
	tests := []struct {
		name        string
		noStorage   bool
		key         string
		put         bool
		putInfo     storage.ObjectInfo
		theme       string
		wantErr     error
		wantRemoved bool
	}{
		{
			name:        "メタデータを記録し閲覧URLを返す",
			noStorage:   false,
			key:         "gallery/" + userA + "/k1",
			put:         true,
			putInfo:     storage.ObjectInfo{Size: 500, ContentType: "image/jpeg"},
			theme:       "pink",
			wantErr:     nil,
			wantRemoved: false,
		},
		{
			name:        "未アップロードのオブジェクトはErrObjectMissing",
			noStorage:   false,
			key:         "gallery/" + userA + "/missing",
			put:         false,
			putInfo:     storage.ObjectInfo{},
			theme:       "pink",
			wantErr:     service.ErrObjectMissing,
			wantRemoved: false,
		},
		{
			name:        "呼び出し元の名前空間外のキーはErrInvalidObjectKey",
			noStorage:   false,
			key:         "gallery/" + userB + "/k1",
			put:         false,
			putInfo:     storage.ObjectInfo{},
			theme:       "pink",
			wantErr:     service.ErrInvalidObjectKey,
			wantRemoved: false,
		},
		{
			name:        "上限超過の保存オブジェクトは拒否され削除される",
			noStorage:   false,
			key:         "gallery/" + userA + "/big",
			put:         true,
			putInfo:     storage.ObjectInfo{Size: cfg.MaxUploadBytes + 1, ContentType: "image/jpeg"},
			theme:       "pink",
			wantErr:     service.ErrUploadTooLarge,
			wantRemoved: true,
		},
		{
			name:        "許可外の保存content typeは拒否され削除される",
			noStorage:   false,
			key:         "gallery/" + userA + "/gif",
			put:         true,
			putInfo:     storage.ObjectInfo{Size: 100, ContentType: "image/gif"},
			theme:       "pink",
			wantErr:     service.ErrContentTypeNotAllowed,
			wantRemoved: true,
		},
		{
			name:        "ストレージ未設定はErrStorageUnavailable",
			noStorage:   true,
			key:         "gallery/" + userA + "/k1",
			put:         false,
			putInfo:     storage.ObjectInfo{},
			theme:       "pink",
			wantErr:     service.ErrStorageUnavailable,
			wantRemoved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var objects *memobject.Store
			var store storage.ObjectStore
			if !tt.noStorage {
				objects = memobject.New()
				store = objects
			}
			svc, gstore := newGalleryService(store, cfg)
			ctx := context.Background()
			if tt.put {
				objects.Put(tt.key, tt.putInfo)
			}

			view, err := svc.RegisterImage(ctx, userA, tt.key, 800, 600, tt.theme)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.key, view.Image.ObjectKey)
				assert.Equal(t, tt.theme, view.Image.Theme)
				assert.Equal(t, 800, view.Image.Width)
				assert.Equal(t, 600, view.Image.Height)
				assert.NotEmpty(t, view.ViewURL)
				assert.True(t, view.ExpiresAt.Equal(testNow.Add(cfg.ViewURLTTL)))
				_, gerr := gstore.GetImage(ctx, userA, view.Image.ID)
				assert.NoError(t, gerr)
			}
			if tt.wantRemoved {
				assert.True(t, objects.Removed(tt.key), "object %q was not purged after a rejected register", tt.key)
			}
		})
	}
}

func TestGalleryService_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		noStorage  bool
		seed       int
		limit      int
		wantPage1  int
		wantCursor bool
		wantPage2  int
	}{
		{
			name:       "ストレージ未設定はErrStorageUnavailable",
			noStorage:  true,
			seed:       0,
			limit:      2,
			wantPage1:  0,
			wantCursor: false,
			wantPage2:  0,
		},
		{
			name:       "空ギャラリーは何も返さない",
			noStorage:  false,
			seed:       0,
			limit:      2,
			wantPage1:  0,
			wantCursor: false,
			wantPage2:  0,
		},
		{
			name:       "ギャラリーは新しい順にページングする",
			noStorage:  false,
			seed:       5,
			limit:      2,
			wantPage1:  2,
			wantCursor: true,
			wantPage2:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var store storage.ObjectStore
			if !tt.noStorage {
				store = memobject.New()
			}
			svc, gstore := newGalleryService(store, galleryConfig())
			ctx := context.Background()
			for i := 1; i <= tt.seed; i++ {
				_, err := gstore.InsertImage(ctx, repository.InsertGalleryParams{
					UserID: userA, ObjectKey: fmt.Sprintf("gallery/%s/k%d", userA, i), Width: 1, Height: 1, Theme: "pink",
				})
				require.NoError(t, err)
			}

			page1, err := svc.List(ctx, userA, tt.limit, nil)
			if tt.noStorage {
				assert.ErrorIs(t, err, service.ErrStorageUnavailable)
				return
			}
			require.NoError(t, err)
			assert.Len(t, page1.Items, tt.wantPage1)
			assert.Equal(t, tt.wantCursor, page1.NextCursor != nil)
			for _, it := range page1.Items {
				assert.NotEmpty(t, it.ViewURL)
			}
			for i := 1; i < len(page1.Items); i++ {
				assert.False(t, page1.Items[i-1].Image.CreatedAt.Before(page1.Items[i].Image.CreatedAt), "page1 not newest-first at index %d", i)
			}
			if !tt.wantCursor {
				return
			}

			page2, err := svc.List(ctx, userA, tt.limit, page1.NextCursor)
			require.NoError(t, err)
			assert.Len(t, page2.Items, tt.wantPage2)
			if len(page2.Items) > 0 {
				assert.True(t, page2.Items[0].Image.CreatedAt.Before(page1.Items[len(page1.Items)-1].Image.CreatedAt), "pages overlap across the cursor boundary")
			}
		})
	}
}

func TestGalleryService_Get(t *testing.T) {
	t.Parallel()

	const key = "gallery/" + userA + "/k1"
	tests := []struct {
		name      string
		noStorage bool
		reader    string
		useRealID bool
		wantErr   error
	}{
		{
			name:      "所有者は閲覧URL付きで画像を取得する",
			noStorage: false,
			reader:    userA,
			useRealID: true,
			wantErr:   nil,
		},
		{
			name:      "別ユーザーは読めずErrGalleryNotFound",
			noStorage: false,
			reader:    userB,
			useRealID: true,
			wantErr:   service.ErrGalleryNotFound,
		},
		{
			name:      "未知のIDはErrGalleryNotFound",
			noStorage: false,
			reader:    userA,
			useRealID: false,
			wantErr:   service.ErrGalleryNotFound,
		},
		{
			name:      "ストレージ未設定はErrStorageUnavailable",
			noStorage: true,
			reader:    userA,
			useRealID: true,
			wantErr:   service.ErrStorageUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var store storage.ObjectStore
			if !tt.noStorage {
				store = memobject.New()
			}
			svc, gstore := newGalleryService(store, galleryConfig())
			ctx := context.Background()
			img, err := gstore.InsertImage(ctx, repository.InsertGalleryParams{
				UserID: userA, ObjectKey: key, Width: 2, Height: 3, Theme: "pink",
			})
			require.NoError(t, err)
			id := img.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			view, err := svc.Get(ctx, tt.reader, id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, key, view.Image.ObjectKey)
			assert.NotEmpty(t, view.ViewURL)
			assert.True(t, view.ExpiresAt.Equal(testNow.Add(galleryConfig().ViewURLTTL)))
		})
	}
}

func TestGalleryService_Delete(t *testing.T) {
	t.Parallel()

	const key = "gallery/" + userA + "/k1"
	tests := []struct {
		name        string
		noStorage   bool
		deleter     string
		useRealID   bool
		deleteCount int
		wantErr     error
		wantRemoved bool
		wantGone    bool
	}{
		{
			name:        "論理削除し保存オブジェクトを削除する",
			noStorage:   false,
			deleter:     userA,
			useRealID:   true,
			deleteCount: 1,
			wantErr:     nil,
			wantRemoved: true,
			wantGone:    true,
		},
		{
			name:        "繰り返し削除は冪等",
			noStorage:   false,
			deleter:     userA,
			useRealID:   true,
			deleteCount: 2,
			wantErr:     nil,
			wantRemoved: true,
			wantGone:    true,
		},
		{
			name:        "別ユーザーは削除できずErrGalleryNotFound",
			noStorage:   false,
			deleter:     userB,
			useRealID:   true,
			deleteCount: 1,
			wantErr:     service.ErrGalleryNotFound,
			wantRemoved: false,
			wantGone:    false,
		},
		{
			name:        "未知のIDはErrGalleryNotFound",
			noStorage:   false,
			deleter:     userA,
			useRealID:   false,
			deleteCount: 1,
			wantErr:     service.ErrGalleryNotFound,
			wantRemoved: false,
			wantGone:    false,
		},
		{
			name:        "ストレージ未設定でも行は論理削除される",
			noStorage:   true,
			deleter:     userA,
			useRealID:   true,
			deleteCount: 1,
			wantErr:     nil,
			wantRemoved: false,
			wantGone:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var objects *memobject.Store
			var store storage.ObjectStore
			if !tt.noStorage {
				objects = memobject.New()
				store = objects
			}
			svc, gstore := newGalleryService(store, galleryConfig())
			ctx := context.Background()
			img, err := gstore.InsertImage(ctx, repository.InsertGalleryParams{
				UserID: userA, ObjectKey: key, Width: 1, Height: 1, Theme: "pink",
			})
			require.NoError(t, err)
			if objects != nil {
				objects.Put(key, storage.ObjectInfo{Size: 1, ContentType: "image/jpeg"})
			}
			id := img.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			var derr error
			for i := 0; i < tt.deleteCount; i++ {
				derr = svc.Delete(ctx, tt.deleter, id)
			}
			if tt.wantErr != nil {
				assert.ErrorIs(t, derr, tt.wantErr)
			} else {
				assert.NoError(t, derr)
			}
			if objects != nil {
				assert.Equal(t, tt.wantRemoved, objects.Removed(key))
			}

			_, gerr := gstore.GetImage(ctx, userA, img.ID)
			if tt.wantGone {
				assert.ErrorIs(t, gerr, repository.ErrNotFound)
			} else {
				assert.NoError(t, gerr)
			}
		})
	}
}
