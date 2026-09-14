// Package repotest holds the contract test suites run against every implementation of the repository stores.
package repotest

import (
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Fixture is one isolated set of stores to test; NewUser inserts the users row the feature tables FK to.
type Fixture struct {
	Auth    repository.AuthStore
	Diary   repository.DiaryStore
	Gallery repository.GalleryStore
	Today   repository.TodayStore
	Devices repository.DeviceStore

	NewUser func(t *testing.T) string
}

// NewFixture builds a fixture isolated from every other test; subtests run in parallel.
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
