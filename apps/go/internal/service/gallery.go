package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

// Gallery pagination bounds: limit is clamped into [1, MaxGalleryLimit], absent → DefaultGalleryLimit.
const (
	DefaultGalleryLimit = 30
	MaxGalleryLimit     = 100
)

// objectRemoveTimeout bounds the background object deletion.
const objectRemoveTimeout = 30 * time.Second

var (
	// ErrGalleryNotFound means the image does not exist, was deleted, or belongs to another user.
	ErrGalleryNotFound = errors.New("service: gallery image not found")
	// ErrStorageUnavailable means object storage is not configured/reachable.
	ErrStorageUnavailable = errors.New("service: object storage unavailable")
	// ErrObjectMissing means the object to register was never uploaded to the store.
	ErrObjectMissing = errors.New("service: object not uploaded")
	// ErrInvalidObjectKey means the object_key is not in the caller's namespace.
	ErrInvalidObjectKey = errors.New("service: object key not owned by caller")
	// ErrContentTypeNotAllowed / ErrUploadTooLarge are the upload-constraint violations.
	ErrContentTypeNotAllowed = errors.New("service: content type not allowed")
	ErrUploadTooLarge        = errors.New("service: upload exceeds size limit")
)

// GalleryConfig carries the upload constraints and URL lifetimes.
type GalleryConfig struct {
	UploadURLTTL        time.Duration
	ViewURLTTL          time.Duration
	MaxUploadBytes      int64
	AllowedContentTypes []string
}

// UploadTarget is what the client needs to PUT bytes directly to the store.
type UploadTarget struct {
	ObjectKey   string
	URL         string
	ContentType string
	ExpiresAt   time.Time
	MaxSize     int64
}

// ImageView is an image plus its short-lived presigned GET URL.
type ImageView struct {
	Image     repository.GalleryImage
	ViewURL   string
	ExpiresAt time.Time
}

// GalleryPage is one keyset page of image views. NextCursor is nil on the last.
type GalleryPage struct {
	Items      []ImageView
	NextCursor *repository.GalleryCursor
}

// GalleryService implements the gallery use cases; with a nil ObjectStore the URL methods return ErrStorageUnavailable.
type GalleryService struct {
	store      repository.GalleryStore
	objects    storage.ObjectStore
	cfg        GalleryConfig
	now        func() time.Time
	newKey     func(userID string) string
	background func(func())
}

// GalleryOption customizes a GalleryService.
type GalleryOption func(*GalleryService)

// WithBackgroundRunner overrides how deferred work (object removal) is run.
func WithBackgroundRunner(run func(func())) GalleryOption {
	return func(s *GalleryService) { s.background = run }
}

// WithObjectKeyFunc overrides object-key generation.
func WithObjectKeyFunc(fn func(userID string) string) GalleryOption {
	return func(s *GalleryService) { s.newKey = fn }
}

// NewGalleryService constructs the service, defaulting now to time.Now.
func NewGalleryService(store repository.GalleryStore, objects storage.ObjectStore, cfg GalleryConfig, now func() time.Time, opts ...GalleryOption) *GalleryService {
	if now == nil {
		now = time.Now
	}
	s := &GalleryService{
		store:      store,
		objects:    objects,
		cfg:        cfg,
		now:        now,
		newKey:     defaultObjectKey,
		background: func(f func()) { go f() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CreateUploadURL validates the requested upload and issues a presigned PUT URL plus its object_key.
func (s *GalleryService) CreateUploadURL(ctx context.Context, userID, contentType string, size int64) (UploadTarget, error) {
	if s.objects == nil {
		return UploadTarget{}, ErrStorageUnavailable
	}
	if !s.contentTypeAllowed(contentType) {
		return UploadTarget{}, ErrContentTypeNotAllowed
	}
	if size > s.cfg.MaxUploadBytes {
		return UploadTarget{}, ErrUploadTooLarge
	}

	key := s.newKey(userID)
	url, err := s.objects.PresignPut(ctx, key, s.cfg.UploadURLTTL)
	if err != nil {
		return UploadTarget{}, err
	}
	return UploadTarget{
		ObjectKey:   key,
		URL:         url,
		ContentType: contentType,
		ExpiresAt:   s.now().Add(s.cfg.UploadURLTTL),
		MaxSize:     s.cfg.MaxUploadBytes,
	}, nil
}

// RegisterImage records metadata after upload, re-verifying the real object (the presigned PUT enforces nothing).
func (s *GalleryService) RegisterImage(ctx context.Context, userID, objectKey string, width, height int) (ImageView, error) {
	if s.objects == nil {
		return ImageView{}, ErrStorageUnavailable
	}
	if !s.ownsKey(userID, objectKey) {
		return ImageView{}, ErrInvalidObjectKey
	}

	info, err := s.objects.Stat(ctx, objectKey)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return ImageView{}, ErrObjectMissing
		}
		return ImageView{}, err
	}
	if info.Size > s.cfg.MaxUploadBytes {
		s.removeQuietly(objectKey)
		return ImageView{}, ErrUploadTooLarge
	}
	if s.contentTypeRejected(info.ContentType) {
		s.removeQuietly(objectKey)
		return ImageView{}, ErrContentTypeNotAllowed
	}

	img, err := s.store.InsertImage(ctx, repository.InsertGalleryParams{
		UserID:    userID,
		ObjectKey: objectKey,
		Width:     width,
		Height:    height,
	})
	if err != nil {
		return ImageView{}, err
	}
	return s.withViewURL(ctx, img)
}

// List returns one keyset page of the user's images, newest first, each with a view URL.
func (s *GalleryService) List(ctx context.Context, userID string, limit int, cursor *repository.GalleryCursor) (GalleryPage, error) {
	if s.objects == nil {
		return GalleryPage{}, ErrStorageUnavailable
	}
	limit = clampGalleryLimit(limit)
	images, err := s.store.ListImages(ctx, repository.ListGalleryParams{
		UserID: userID,
		Limit:  limit + 1, // sentinel row reveals a further page
		Cursor: cursor,
	})
	if err != nil {
		return GalleryPage{}, err
	}

	var next *repository.GalleryCursor
	if len(images) > limit {
		last := images[limit-1]
		next = &repository.GalleryCursor{CreatedAt: last.CreatedAt, ID: last.ID}
		images = images[:limit]
	}

	page := GalleryPage{Items: make([]ImageView, 0, len(images)), NextCursor: next}
	for _, img := range images {
		view, err := s.withViewURL(ctx, img)
		if err != nil {
			return GalleryPage{}, err
		}
		page.Items = append(page.Items, view)
	}
	return page, nil
}

// Get returns a single image view.
func (s *GalleryService) Get(ctx context.Context, userID, id string) (ImageView, error) {
	if s.objects == nil {
		return ImageView{}, ErrStorageUnavailable
	}
	img, err := s.store.GetImage(ctx, userID, id)
	if err != nil {
		return ImageView{}, mapGalleryNotFound(err)
	}
	return s.withViewURL(ctx, img)
}

// Delete soft-deletes an image and removes the stored object in the background.
func (s *GalleryService) Delete(ctx context.Context, userID, id string) error {
	objectKey, err := s.store.SoftDeleteImage(ctx, userID, id)
	if err != nil {
		return mapGalleryNotFound(err)
	}
	if s.objects != nil {
		s.background(func() { s.removeQuietly(objectKey) })
	}
	return nil
}

func (s *GalleryService) withViewURL(ctx context.Context, img repository.GalleryImage) (ImageView, error) {
	url, err := s.objects.PresignGet(ctx, img.ObjectKey, s.cfg.ViewURLTTL)
	if err != nil {
		return ImageView{}, err
	}
	return ImageView{Image: img, ViewURL: url, ExpiresAt: s.now().Add(s.cfg.ViewURLTTL)}, nil
}

func (s *GalleryService) removeQuietly(objectKey string) {
	if s.objects == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), objectRemoveTimeout)
	defer cancel()
	_ = s.objects.Remove(ctx, objectKey)
}

func (s *GalleryService) contentTypeAllowed(ct string) bool {
	for _, allowed := range s.cfg.AllowedContentTypes {
		if strings.EqualFold(ct, allowed) {
			return true
		}
	}
	return false
}

// contentTypeRejected reports whether a stored object's content-type is
// definitively disallowed; empty / octet-stream (no header sent) is not rejected.
func (s *GalleryService) contentTypeRejected(ct string) bool {
	if ct == "" || strings.EqualFold(ct, "application/octet-stream") {
		return false
	}
	return !s.contentTypeAllowed(ct)
}

func (s *GalleryService) ownsKey(userID, objectKey string) bool {
	return strings.HasPrefix(objectKey, "gallery/"+userID+"/")
}

// MaxUploadBytes / AllowedContentTypes expose limits for handler-level messages.
func (s *GalleryService) MaxUploadBytes() int64         { return s.cfg.MaxUploadBytes }
func (s *GalleryService) AllowedContentTypes() []string { return s.cfg.AllowedContentTypes }

func clampGalleryLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultGalleryLimit
	case limit > MaxGalleryLimit:
		return MaxGalleryLimit
	default:
		return limit
	}
}

func mapGalleryNotFound(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrGalleryNotFound
	}
	return err
}

func defaultObjectKey(userID string) string {
	return "gallery/" + userID + "/" + newObjectID()
}

// newObjectID returns a random v4-style UUID string.
func newObjectID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
