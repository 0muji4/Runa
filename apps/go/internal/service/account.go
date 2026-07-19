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

// MaxDisplayNameLength bounds a user-chosen display name (in runes, so a name of
// CJK characters is measured by character count, not byte length).
const MaxDisplayNameLength = 50

// exportImagePageSize is how many gallery rows Export pages through at a time.
const exportImagePageSize = 100

// accountObjectRemoveTimeout bounds the background object purge on account
// deletion so a slow store never leaks a goroutine.
const accountObjectRemoveTimeout = 30 * time.Second

var (
	// ErrDisplayNameRequired means the display name was empty after trimming.
	ErrDisplayNameRequired = errors.New("service: display name is required")
	// ErrDisplayNameTooLong means the display name exceeded MaxDisplayNameLength.
	ErrDisplayNameTooLong = errors.New("service: display name too long")
)

// ExportedImage is one image's metadata plus an optional presigned GET URL. URL
// is empty when object storage is unconfigured or a URL could not be signed; the
// metadata still exports so a storage outage never blocks the diary export.
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

// AccountService implements the account-data use cases: display-name update,
// self-service export and permanent account deletion.
//
// Why it composes four stores: "the account" is not a single bounded context —
// it spans the user record (auth), the diary, the gallery and object storage.
// Export must aggregate them and deletion must purge them, so this service is the
// one place that legitimately depends on all four. Feature services stay scoped
// to their own store; only the cross-cutting account concern reaches across.
type AccountService struct {
	users      repository.AuthStore
	diaries    repository.DiaryStore
	gallery    repository.GalleryStore
	objects    storage.ObjectStore // may be nil when storage is unconfigured
	cfg        AccountConfig
	now        func() time.Time
	background func(func())
}

// AccountOption customizes an AccountService (tests run the object purge
// synchronously so deletion assertions are deterministic).
type AccountOption func(*AccountService)

// WithAccountBackgroundRunner overrides how the deferred object purge is run.
func WithAccountBackgroundRunner(run func(func())) AccountOption {
	return func(s *AccountService) { s.background = run }
}

// NewAccountService constructs the service, defaulting now to time.Now and the
// background runner to a goroutine.
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
// updated user. A missing user maps to ErrUserNotFound (the account was deleted
// under a still-valid access token).
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

// Export aggregates the caller's profile, diary entries (tombstones excluded —
// export is live data) and gallery images (metadata plus a presigned GET URL when
// storage is available).
func (s *AccountService) Export(ctx context.Context, userID string) (AccountExport, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return AccountExport{}, ErrUserNotFound
		}
		return AccountExport{}, err
	}

	// ListChangedSince(epoch) returns every entry including tombstones in one call;
	// keep only the live ones for the export.
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

// exportImages pages through the user's visible images, attaching a presigned GET
// URL when storage is available. A presign failure degrades to metadata-only for
// that image rather than failing the whole export.
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

// DeleteAccount permanently removes the user and every row that cascades from it
// (refresh tokens, diary entries, gallery rows, song history), then purges the
// user's stored objects in the background (the spec allows async storage cleanup).
//
// Token invalidation is structural, not a separate revocation step: the refresh
// tokens vanish with the cascade, and the now-missing user row makes any
// still-valid access token fail its next user lookup (401). The residual window is
// bounded by the short access-token TTL.
func (s *AccountService) DeleteAccount(ctx context.Context, userID string) error {
	// Read the object keys BEFORE the delete cascades the gallery rows away.
	keys, err := s.objectKeysToPurge(ctx, userID)
	if err != nil {
		return err
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

// objectKeysToPurge lists every object key to remove on deletion. Without a
// configured store there is nothing to purge, so it skips the query entirely.
func (s *AccountService) objectKeysToPurge(ctx context.Context, userID string) ([]string, error) {
	if s.objects == nil {
		return nil, nil
	}
	return s.gallery.ListObjectKeys(ctx, userID)
}

// purgeObjects removes stored objects best-effort with a bounded context. A failed
// removal only leaks a byte blob (the row is already gone and view URLs expire),
// so errors are logged nowhere and ignored — orphan cleanup is not a correctness
// or security concern here.
func (s *AccountService) purgeObjects(keys []string) {
	ctx, cancel := context.WithTimeout(context.Background(), accountObjectRemoveTimeout)
	defer cancel()
	for _, key := range keys {
		_ = s.objects.Remove(ctx, key)
	}
}
