package repotest

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/google/go-cmp/cmp"
)

// RunGalleryStoreSuite exercises the GalleryStore contract.
func RunGalleryStoreSuite(t *testing.T, newFixture NewFixture) {
	insert := func(t *testing.T, f Fixture, userID string, n int) []repository.GalleryImage {
		t.Helper()
		out := make([]repository.GalleryImage, 0, n)
		for i := 1; i <= n; i++ {
			img, err := f.Gallery.InsertImage(t.Context(), repository.InsertGalleryParams{
				UserID:    userID,
				ObjectKey: fmt.Sprintf("gallery/%s/k%d", userID, i),
				Width:     100 + i, Height: 200 + i, Theme: "pink",
			})
			if err != nil {
				t.Fatalf("seeding image %d: InsertImage() error = %v, want nil", i, err)
			}
			out = append(out, img)
		}
		return out
	}

	t.Run("InsertUpsertsOnObjectKey", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		key := "gallery/" + user + "/retried"

		first, err := f.Gallery.InsertImage(ctx, repository.InsertGalleryParams{
			UserID: user, ObjectKey: key, Width: 800, Height: 600, Theme: "pink",
		})
		if err != nil {
			t.Fatalf("first InsertImage() error = %v, want nil", err)
		}
		second, err := f.Gallery.InsertImage(ctx, repository.InsertGalleryParams{
			UserID: user, ObjectKey: key, Width: 1024, Height: 768, Theme: "monotone",
		})
		if err != nil {
			t.Fatalf("second InsertImage() error = %v, want nil", err)
		}
		if second.ID != first.ID {
			t.Errorf("a retried registration created id %q, want the existing %q", second.ID, first.ID)
		}

		list, err := f.Gallery.ListImages(ctx, repository.ListGalleryParams{UserID: user, Limit: 10})
		if err != nil {
			t.Fatalf("ListImages() error = %v, want nil", err)
		}
		if len(list) != 1 {
			t.Errorf("a retried registration left %d rows, want 1", len(list))
		}
	})

	t.Run("ListIsNewestFirstAndPagesWithoutGaps", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		seeded := insert(t, f, user, 5)

		var walked []string
		var cursor *repository.GalleryCursor
		for page := 0; page < 10; page++ {
			images, err := f.Gallery.ListImages(ctx, repository.ListGalleryParams{
				UserID: user, Limit: 2, Cursor: cursor,
			})
			if err != nil {
				t.Fatalf("ListImages(page %d) error = %v, want nil", page, err)
			}
			if len(images) == 0 {
				break
			}
			for i := 1; i < len(images); i++ {
				if images[i-1].CreatedAt.Before(images[i].CreatedAt) {
					t.Errorf("page %d is not newest-first at index %d", page, i)
				}
			}
			for _, img := range images {
				walked = append(walked, img.ID)
			}
			last := images[len(images)-1]
			cursor = &repository.GalleryCursor{CreatedAt: last.CreatedAt, ID: last.ID}
		}
		if len(walked) != len(seeded) {
			t.Errorf("paging walked %d images, want %d (gaps or duplicates)", len(walked), len(seeded))
		}
		if len(uniq(walked)) != len(walked) {
			t.Errorf("paging returned duplicate ids: %v", walked)
		}
	})

	t.Run("GetAndDeleteAreOwnerScoped", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user, stranger := f.NewUser(t), f.NewUser(t)
		img := insert(t, f, user, 1)[0]

		if _, err := f.Gallery.GetImage(ctx, stranger, img.ID); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("GetImage(stranger) error = %v, want %v", err, repository.ErrNotFound)
		}
		if _, err := f.Gallery.SoftDeleteImage(ctx, stranger, img.ID); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("SoftDeleteImage(stranger) error = %v, want %v", err, repository.ErrNotFound)
		}
		if _, err := f.Gallery.GetImage(ctx, user, img.ID); err != nil {
			t.Errorf("GetImage(owner) after a stranger's delete error = %v, want nil", err)
		}
	})

	t.Run("SoftDeleteReturnsTheObjectKeyAndIsIdempotent", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		img := insert(t, f, user, 1)[0]

		// The returned key is what the service hands to object storage.
		key, err := f.Gallery.SoftDeleteImage(ctx, user, img.ID)
		if err != nil {
			t.Fatalf("SoftDeleteImage() error = %v, want nil", err)
		}
		if key != img.ObjectKey {
			t.Errorf("SoftDeleteImage() object key = %q, want %q", key, img.ObjectKey)
		}
		if _, err := f.Gallery.SoftDeleteImage(ctx, user, img.ID); err != nil {
			t.Errorf("second SoftDeleteImage() error = %v, want nil", err)
		}
		if _, err := f.Gallery.GetImage(ctx, user, img.ID); !errors.Is(err, repository.ErrNotFound) {
			t.Errorf("GetImage() on a deleted image error = %v, want %v", err, repository.ErrNotFound)
		}
	})

	t.Run("ListObjectKeysIncludesDeletedRows", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)
		images := insert(t, f, user, 2)

		if _, err := f.Gallery.SoftDeleteImage(ctx, user, images[0].ID); err != nil {
			t.Fatalf("SoftDeleteImage() error = %v, want nil", err)
		}

		// Account deletion purges from this list, so it must include the rows a
		// soft delete already hid.
		keys, err := f.Gallery.ListObjectKeys(ctx, user)
		if err != nil {
			t.Fatalf("ListObjectKeys() error = %v, want nil", err)
		}
		want := []string{images[0].ObjectKey, images[1].ObjectKey}
		sort.Strings(want)
		sort.Strings(keys)
		if diff := cmp.Diff(want, keys); diff != "" {
			t.Errorf("ListObjectKeys() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("UnknownThemeIsRejected", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		// The handler validates theme too, but the store must not be the more
		// permissive of the two.
		tests := []struct {
			name    string
			theme   string
			wantErr bool
		}{
			{
				name:    "monotoneは受け入れる",
				theme:   "monotone",
				wantErr: false,
			},
			{
				name:    "pinkは受け入れる",
				theme:   "pink",
				wantErr: false,
			},
			{
				name:    "未知の色は拒否する",
				theme:   "blue",
				wantErr: true,
			},
			{
				name:    "空文字は拒否する",
				theme:   "",
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := f.Gallery.InsertImage(ctx, repository.InsertGalleryParams{
					UserID:    user,
					ObjectKey: fmt.Sprintf("gallery/%s/theme-%s", user, tt.name),
					Width:     10, Height: 10, Theme: tt.theme,
				})
				if tt.wantErr && err == nil {
					t.Errorf("InsertImage(theme=%q) error = nil, want a rejection", tt.theme)
				}
				if !tt.wantErr && err != nil {
					t.Errorf("InsertImage(theme=%q) error = %v, want nil", tt.theme, err)
				}
			})
		}
	})

	t.Run("ImageCarriesItsMetadata", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser(t)

		key := "gallery/" + user + "/meta"
		created, err := f.Gallery.InsertImage(ctx, repository.InsertGalleryParams{
			UserID: user, ObjectKey: key, Width: 1280, Height: 960, Theme: "monotone",
		})
		if err != nil {
			t.Fatalf("InsertImage() error = %v, want nil", err)
		}
		if created.CreatedAt.IsZero() {
			t.Error("created_at is zero, want the insert time")
		}
		if created.DeletedAt != nil {
			t.Errorf("deleted_at = %s on a fresh row, want nil", created.DeletedAt.UTC())
		}

		got, err := f.Gallery.GetImage(ctx, user, created.ID)
		if err != nil {
			t.Fatalf("GetImage() error = %v, want nil", err)
		}
		type meta struct {
			Key           string
			Width, Height int
			Theme         string
		}
		want := meta{key, 1280, 960, "monotone"}
		gotMeta := meta{got.ObjectKey, got.Width, got.Height, got.Theme}
		if gotMeta != want {
			t.Errorf("GetImage() metadata = %+v, want %+v", gotMeta, want)
		}
	})
}

// uniq returns the distinct values of in.
func uniq(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
