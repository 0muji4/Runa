package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

// MaxDisplayNameLength bounds a user-chosen display name in runes, not bytes.
const MaxDisplayNameLength = 50

const exportImagePageSize = 100

const accountObjectRemoveTimeout = 30 * time.Second

var (
	// ErrDisplayNameRequired means the display name was empty after trimming.
	ErrDisplayNameRequired = errors.New("service: display name is required")
	// ErrDisplayNameTooLong means the display name exceeded MaxDisplayNameLength.
	ErrDisplayNameTooLong = errors.New("service: display name too long")
)

// ExportedImage is one image's metadata plus an optional presigned GET URL; URL
// is empty when object storage is unconfigured or the presign failed.
type ExportedImage struct {
	Image     repository.GalleryImage
	URL       string
	ExpiresAt time.Time
}

// AccountExport is the caller's full self-service export payload.
type AccountExport struct {
	ExportedAt time.Time
	User       repository.User
	Diaries    []repository.DiaryEntry
	Images     []ExportedImage
}

// AccountConfig carries the lifetime used when presigning export image URLs.
type AccountConfig struct {
	ExportURLTTL time.Duration
}

// AccountService implements display-name update, self-service export and account deletion.
type AccountService struct {
	users      repository.AuthStore
	diaries    repository.DiaryStore
	gallery    repository.GalleryStore
	objects    storage.ObjectStore // may be nil when storage is unconfigured
	devices    *DeviceService      // may be nil when push is not wired
	cfg        AccountConfig
	now        func() time.Time
	background func(func())
}

// AccountOption customizes an AccountService.
type AccountOption func(*AccountService)

// WithAccountBackgroundRunner overrides how the deferred object purge is run.
func WithAccountBackgroundRunner(run func(func())) AccountOption {
	return func(s *AccountService) { s.background = run }
}

// WithAccountDevices makes DeleteAccount drop the user's push registrations
// (and their scheduled reminders) before the user row goes.
func WithAccountDevices(devices *DeviceService) AccountOption {
	return func(s *AccountService) { s.devices = devices }
}

// NewAccountService constructs the service, defaulting now to time.Now.
func NewAccountService(users repository.AuthStore, diaries repository.DiaryStore, gallery repository.GalleryStore, objects storage.ObjectStore, cfg AccountConfig, now func() time.Time, opts ...AccountOption) *AccountService {
	if now == nil {
		now = time.Now
	}
	s := &AccountService{
		users:      users,
		diaries:    diaries,
		gallery:    gallery,
		objects:    objects,
		cfg:        cfg,
		now:        now,
		background: func(f func()) { go f() },
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// UpdateDisplayName validates and persists a new display name, returning the
// updated user; a missing user maps to ErrUserNotFound.
func (s *AccountService) UpdateDisplayName(ctx context.Context, userID, displayName string) (repository.User, error) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		return repository.User{}, ErrDisplayNameRequired
	}
	if utf8.RuneCountInString(name) > MaxDisplayNameLength {
		return repository.User{}, ErrDisplayNameTooLong
	}
	user, err := s.users.UpdateDisplayName(ctx, userID, name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return repository.User{}, ErrUserNotFound
		}
		return repository.User{}, err
	}
	return user, nil
}

// Export aggregates the caller's profile, live diary entries (tombstones
// excluded) and gallery images.
func (s *AccountService) Export(ctx context.Context, userID string) (AccountExport, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return AccountExport{}, ErrUserNotFound
		}
		return AccountExport{}, err
	}

	// ListChangedSince(epoch) includes tombstones; keep only the live ones.
	changed, err := s.diaries.ListChangedSince(ctx, userID, time.Time{})
	if err != nil {
		return AccountExport{}, err
	}
	diaries := make([]repository.DiaryEntry, 0, len(changed))
	for _, e := range changed {
		if e.DeletedAt == nil {
			diaries = append(diaries, e)
		}
	}

	images, err := s.exportImages(ctx, userID)
	if err != nil {
		return AccountExport{}, err
	}

	return AccountExport{
		ExportedAt: s.now().UTC(),
		User:       user,
		Diaries:    diaries,
		Images:     images,
	}, nil
}

// exportImages pages through the user's images; a presign failure degrades that image to metadata-only.
func (s *AccountService) exportImages(ctx context.Context, userID string) ([]ExportedImage, error) {
	out := make([]ExportedImage, 0)
	var cursor *repository.GalleryCursor
	for {
		page, err := s.gallery.ListImages(ctx, repository.ListGalleryParams{
			UserID: userID,
			Limit:  exportImagePageSize,
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		for _, img := range page {
			item := ExportedImage{Image: img}
			if s.objects != nil {
				if url, err := s.objects.PresignGet(ctx, img.ObjectKey, s.cfg.ExportURLTTL); err == nil {
					item.URL = url
					item.ExpiresAt = s.now().Add(s.cfg.ExportURLTTL)
				}
			}
			out = append(out, item)
		}
		if len(page) < exportImagePageSize {
			return out, nil
		}
		last := page[len(page)-1]
		cursor = &repository.GalleryCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
}

// DeleteAccount permanently removes the user and every row that cascades from
// it, then purges the user's stored objects in the background. Refresh tokens go
// with the cascade; a still-valid access token fails its next user lookup.
func (s *AccountService) DeleteAccount(ctx context.Context, userID string) error {
	// Read the object keys BEFORE the delete cascades the gallery rows away.
	keys, err := s.objectKeysToPurge(ctx, userID)
	if err != nil {
		return err
	}

	// The DB cascade would drop the rows, but the scheduled callbacks must be
	// cancelled while the rows (and their task names) are still readable.
	if s.devices != nil {
		if err := s.devices.UnregisterAll(ctx, userID); err != nil {
			return err
		}
	}

	if err := s.users.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if s.objects != nil && len(keys) > 0 {
		s.background(func() { s.purgeObjects(keys) })
	}
	return nil
}

func (s *AccountService) objectKeysToPurge(ctx context.Context, userID string) ([]string, error) {
	if s.objects == nil {
		return nil, nil
	}
	return s.gallery.ListObjectKeys(ctx, userID)
}

// purgeObjects removes stored objects best-effort; a failed removal only leaves an orphan blob.
func (s *AccountService) purgeObjects(keys []string) {
	ctx, cancel := context.WithTimeout(context.Background(), accountObjectRemoveTimeout)
	defer cancel()
	for _, key := range keys {
		_ = s.objects.Remove(ctx, key)
	}
}
