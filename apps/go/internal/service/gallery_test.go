package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
	"github.com/0muji4/Runa/apps/go/internal/storage/memobject"
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
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("CreateUploadURL(%q, %d) error = %v, want %v",
						tt.contentType, tt.size, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateUploadURL(%q, %d) error = %v, want nil",
					tt.contentType, tt.size, err)
			}
			if wantPrefix := "gallery/" + userA + "/"; !strings.HasPrefix(target.ObjectKey, wantPrefix) {
				t.Errorf("CreateUploadURL() object key = %q, want prefix %q (caller's namespace)",
					target.ObjectKey, wantPrefix)
			}
			if target.URL == "" {
				t.Error("CreateUploadURL() url is empty, want a presigned URL")
			}
			if target.ContentType != tt.contentType {
				t.Errorf("CreateUploadURL() content_type = %q, want %q",
					target.ContentType, tt.contentType)
			}
			if target.MaxSize != cfg.MaxUploadBytes {
				t.Errorf("CreateUploadURL() max_size = %d, want %d",
					target.MaxSize, cfg.MaxUploadBytes)
			}
			if want := testNow.Add(cfg.UploadURLTTL); !target.ExpiresAt.Equal(want) {
				t.Errorf("CreateUploadURL() expires_at = %s, want %s", target.ExpiresAt, want)
			}
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
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RegisterImage(%q, %q) error = %v, want %v",
						tt.key, tt.theme, err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("RegisterImage(%q, %q) error = %v, want nil", tt.key, tt.theme, err)
				}
				want := struct {
					key, theme string
					w, h       int
				}{tt.key, tt.theme, 800, 600}
				got := struct {
					key, theme string
					w, h       int
				}{view.Image.ObjectKey, view.Image.Theme, view.Image.Width, view.Image.Height}
				if got != want {
					t.Errorf("RegisterImage() image = %+v, want %+v", got, want)
				}
				if view.ViewURL == "" {
					t.Error("RegisterImage() view url is empty, want a presigned URL")
				}
				if wantExp := testNow.Add(cfg.ViewURLTTL); !view.ExpiresAt.Equal(wantExp) {
					t.Errorf("RegisterImage() expires_at = %s, want %s", view.ExpiresAt, wantExp)
				}
				if _, err := gstore.GetImage(ctx, userA, view.Image.ID); err != nil {
					t.Errorf("GetImage(%q) after register error = %v, want nil", view.Image.ID, err)
				}
			}
			if tt.wantRemoved && !objects.Removed(tt.key) {
				t.Errorf("object %q was not purged after a rejected register", tt.key)
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
				if err != nil {
					t.Fatalf("seeding image %d: InsertImage() error = %v, want nil", i, err)
				}
			}

			page1, err := svc.List(ctx, userA, tt.limit, nil)
			if tt.noStorage {
				if !errors.Is(err, service.ErrStorageUnavailable) {
					t.Fatalf("List() without storage error = %v, want %v",
						err, service.ErrStorageUnavailable)
				}
				return
			}
			if err != nil {
				t.Fatalf("List(limit=%d) error = %v, want nil", tt.limit, err)
			}
			if len(page1.Items) != tt.wantPage1 {
				t.Errorf("List(limit=%d) page 1 returned %d items, want %d",
					tt.limit, len(page1.Items), tt.wantPage1)
			}
			if got := page1.NextCursor != nil; got != tt.wantCursor {
				t.Errorf("List(limit=%d) page 1 has a next cursor = %t, want %t",
					tt.limit, got, tt.wantCursor)
			}
			for i, it := range page1.Items {
				if it.ViewURL == "" {
					t.Errorf("List() item %d view url is empty, want a presigned URL", i)
				}
			}
			for i := 1; i < len(page1.Items); i++ {
				if page1.Items[i-1].Image.CreatedAt.Before(page1.Items[i].Image.CreatedAt) {
					t.Errorf("List() page 1 is not newest-first at index %d: %s before %s",
						i, page1.Items[i-1].Image.CreatedAt, page1.Items[i].Image.CreatedAt)
				}
			}
			if !tt.wantCursor {
				return
			}

			page2, err := svc.List(ctx, userA, tt.limit, page1.NextCursor)
			if err != nil {
				t.Fatalf("List(limit=%d, cursor) error = %v, want nil", tt.limit, err)
			}
			if len(page2.Items) != tt.wantPage2 {
				t.Errorf("List(limit=%d, cursor) page 2 returned %d items, want %d",
					tt.limit, len(page2.Items), tt.wantPage2)
			}
			if len(page2.Items) > 0 {
				lastOfPage1 := page1.Items[len(page1.Items)-1].Image.CreatedAt
				if !page2.Items[0].Image.CreatedAt.Before(lastOfPage1) {
					t.Errorf("pages overlap across the cursor boundary: page 2 starts at %s, page 1 ended at %s",
						page2.Items[0].Image.CreatedAt, lastOfPage1)
				}
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
			if err != nil {
				t.Fatalf("InsertImage(%q) error = %v, want nil", key, err)
			}
			id := img.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			view, err := svc.Get(ctx, tt.reader, id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Get(%q, %q) error = %v, want %v", tt.reader, id, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get(%q, %q) error = %v, want nil", tt.reader, id, err)
			}
			if view.Image.ObjectKey != key {
				t.Errorf("Get() object key = %q, want %q", view.Image.ObjectKey, key)
			}
			if view.ViewURL == "" {
				t.Error("Get() view url is empty, want a presigned URL")
			}
			if want := testNow.Add(galleryConfig().ViewURLTTL); !view.ExpiresAt.Equal(want) {
				t.Errorf("Get() expires_at = %s, want %s", view.ExpiresAt, want)
			}
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
			if err != nil {
				t.Fatalf("InsertImage(%q) error = %v, want nil", key, err)
			}
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
				if !errors.Is(derr, tt.wantErr) {
					t.Fatalf("Delete(%q) error = %v, want %v", tt.deleter, derr, tt.wantErr)
				}
			} else if derr != nil {
				t.Fatalf("Delete(%q) error = %v, want nil", tt.deleter, derr)
			}
			if objects != nil {
				if got := objects.Removed(key); got != tt.wantRemoved {
					t.Errorf("object %q removed = %t, want %t", key, got, tt.wantRemoved)
				}
			}

			_, gerr := gstore.GetImage(ctx, userA, img.ID)
			if tt.wantGone {
				if !errors.Is(gerr, repository.ErrNotFound) {
					t.Errorf("GetImage(%q) after delete error = %v, want %v",
						img.ID, gerr, repository.ErrNotFound)
				}
			} else if gerr != nil {
				t.Errorf("GetImage(%q) error = %v, want nil (row must survive)", img.ID, gerr)
			}
		})
	}
}
