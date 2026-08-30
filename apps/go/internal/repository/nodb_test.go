package repository_test

import (
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// TestNilPoolReportsErrNoDatabase covers the degraded-boot path: with Postgres
// unreachable the server serves liveness on a nil pool, so every store method has
// to answer ErrNoDatabase rather than dereference it.
func TestNilPoolReportsErrNoDatabase(t *testing.T) {
	t.Parallel()

	var (
		auth    = repository.NewAuthRepository(nil)
		diary   = repository.NewDiaryRepository(nil)
		gallery = repository.NewGalleryRepository(nil)
		today   = repository.NewTodayRepository(nil)
		devices = repository.NewDeviceRepository(nil)
	)

	tests := []struct {
		name string
		call func(t *testing.T) error
	}{
		{
			name: "AuthStore/CreateUser",
			call: func(t *testing.T) error {
				_, err := auth.CreateUser(t.Context(), repository.CreateUserParams{DisplayName: "x"})
				return err
			},
		},
		{
			name: "AuthStore/GetUserByID",
			call: func(t *testing.T) error {
				_, err := auth.GetUserByID(t.Context(), "id")
				return err
			},
		},
		{
			name: "AuthStore/GetUserByEmail",
			call: func(t *testing.T) error {
				_, err := auth.GetUserByEmail(t.Context(), "a@b.com")
				return err
			},
		},
		{
			name: "AuthStore/GetUserByProviderSub",
			call: func(t *testing.T) error {
				_, err := auth.GetUserByProviderSub(t.Context(), "apple", "sub")
				return err
			},
		},
		{
			name: "AuthStore/UpdateDisplayName",
			call: func(t *testing.T) error {
				_, err := auth.UpdateDisplayName(t.Context(), "id", "name")
				return err
			},
		},
		{
			name: "AuthStore/DeleteUser",
			call: func(t *testing.T) error { return auth.DeleteUser(t.Context(), "id") },
		},
		{
			name: "AuthStore/InsertRefreshToken",
			call: func(t *testing.T) error {
				return auth.InsertRefreshToken(t.Context(), repository.InsertRefreshTokenParams{})
			},
		},
		{
			name: "AuthStore/GetRefreshTokenByHash",
			call: func(t *testing.T) error {
				_, err := auth.GetRefreshTokenByHash(t.Context(), "hash")
				return err
			},
		},
		{
			name: "AuthStore/RevokeRefreshToken",
			call: func(t *testing.T) error { return auth.RevokeRefreshToken(t.Context(), "hash") },
		},
		{
			name: "DiaryStore/UpsertEntry",
			call: func(t *testing.T) error {
				_, _, err := diary.UpsertEntry(t.Context(), repository.UpsertDiaryParams{})
				return err
			},
		},
		{
			name: "DiaryStore/ListEntries",
			call: func(t *testing.T) error {
				_, err := diary.ListEntries(t.Context(), repository.ListDiaryParams{})
				return err
			},
		},
		{
			name: "DiaryStore/GetEntry",
			call: func(t *testing.T) error {
				_, err := diary.GetEntry(t.Context(), "user", "id")
				return err
			},
		},
		{
			name: "DiaryStore/UpdateEntry",
			call: func(t *testing.T) error {
				_, err := diary.UpdateEntry(t.Context(), "user", "id", repository.UpdateDiaryParams{})
				return err
			},
		},
		{
			name: "DiaryStore/SoftDeleteEntry",
			call: func(t *testing.T) error { return diary.SoftDeleteEntry(t.Context(), "user", "id") },
		},
		{
			name: "DiaryStore/ListChangedSince",
			call: func(t *testing.T) error {
				_, err := diary.ListChangedSince(t.Context(), "user", time.Time{})
				return err
			},
		},
		{
			name: "DiaryStore/CountByLocalDate",
			call: func(t *testing.T) error {
				_, err := diary.CountByLocalDate(t.Context(), "user", time.Time{}, time.Time{}, time.UTC)
				return err
			},
		},
		{
			name: "DiaryStore/EntriesInRange",
			call: func(t *testing.T) error {
				_, err := diary.EntriesInRange(t.Context(), "user", time.Time{}, time.Time{})
				return err
			},
		},
		{
			name: "GalleryStore/InsertImage",
			call: func(t *testing.T) error {
				_, err := gallery.InsertImage(t.Context(), repository.InsertGalleryParams{})
				return err
			},
		},
		{
			name: "GalleryStore/ListImages",
			call: func(t *testing.T) error {
				_, err := gallery.ListImages(t.Context(), repository.ListGalleryParams{})
				return err
			},
		},
		{
			name: "GalleryStore/GetImage",
			call: func(t *testing.T) error {
				_, err := gallery.GetImage(t.Context(), "user", "id")
				return err
			},
		},
		{
			name: "GalleryStore/SoftDeleteImage",
			call: func(t *testing.T) error {
				_, err := gallery.SoftDeleteImage(t.Context(), "user", "id")
				return err
			},
		},
		{
			name: "GalleryStore/ListObjectKeys",
			call: func(t *testing.T) error {
				_, err := gallery.ListObjectKeys(t.Context(), "user")
				return err
			},
		},
		{
			name: "TodayStore/GetQuoteForDate",
			call: func(t *testing.T) error {
				_, err := today.GetQuoteForDate(t.Context(), time.Time{})
				return err
			},
		},
		{
			name: "TodayStore/GetSongForDate",
			call: func(t *testing.T) error {
				_, err := today.GetSongForDate(t.Context(), time.Time{})
				return err
			},
		},
		{
			name: "TodayStore/ListSongs",
			call: func(t *testing.T) error {
				_, err := today.ListSongs(t.Context(), repository.ListSongsParams{})
				return err
			},
		},
		{
			name: "TodayStore/RecordPlay",
			call: func(t *testing.T) error {
				return today.RecordPlay(t.Context(), "user", "song", time.Time{})
			},
		},
		{
			name: "TodayStore/InsertQuote",
			call: func(t *testing.T) error {
				_, err := today.InsertQuote(t.Context(), repository.InsertQuoteParams{})
				return err
			},
		},
		{
			name: "TodayStore/InsertSong",
			call: func(t *testing.T) error {
				_, err := today.InsertSong(t.Context(), repository.InsertSongParams{})
				return err
			},
		},
		{
			name: "DeviceStore/UpsertDevice",
			call: func(t *testing.T) error {
				_, err := devices.UpsertDevice(t.Context(), repository.UpsertDeviceParams{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.call(t); !errors.Is(err, repository.ErrNoDatabase) {
				t.Errorf("%s with a nil pool error = %v, want %v", tt.name, err, repository.ErrNoDatabase)
			}
		})
	}
}
