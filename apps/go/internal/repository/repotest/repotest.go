// Package repotest holds the contract test suites for the repository stores.
// Each suite runs against both implementations of a store: the in-memory fake
// (internal/repository/memstores) and the pgx one against a real Postgres
// (internal/repository), so the two cannot drift apart unnoticed.
package repotest

import (
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Fixture is one isolated set of stores to test, supplied by a backend. NewUser
// inserts a users row: every feature table has an FK to it, so Postgres needs one
// to exist before anything can reference it.
type Fixture struct {
	Auth    repository.AuthStore
	Diary   repository.DiaryStore
	Gallery repository.GalleryStore
	Today   repository.TodayStore
	Devices repository.DeviceStore

	NewUser func(t *testing.T) string
}

// NewFixture builds a fixture isolated from every other test. The suites run
// their subtests in parallel, so each call needs its own storage.
type NewFixture func(t *testing.T) Fixture

// RunStoreSuites runs every contract suite against one backend.
func RunStoreSuites(t *testing.T, newFixture NewFixture) {
	t.Run("AuthStore", func(t *testing.T) { RunAuthStoreSuite(t, newFixture) })
	t.Run("DiaryStore", func(t *testing.T) { RunDiaryStoreSuite(t, newFixture) })
	t.Run("GalleryStore", func(t *testing.T) { RunGalleryStoreSuite(t, newFixture) })
	t.Run("TodayStore", func(t *testing.T) { RunTodayStoreSuite(t, newFixture) })
	t.Run("DeviceStore", func(t *testing.T) { RunDeviceStoreSuite(t, newFixture) })
}

func ptr[T any](v T) *T { return &v }
