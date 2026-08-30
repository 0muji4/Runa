// Package memstores_test runs the repository contract suites against the
// in-memory fakes. They live in separate packages but are only testable together
// (a diary entry needs a user), so this one wires them into a Fixture.
package memstores_test

import (
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdevices"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/repository/memtoday"
	"github.com/0muji4/Runa/apps/go/internal/repository/repotest"
)

func TestInMemoryStoresMeetTheContract(t *testing.T) {
	t.Parallel()
	repotest.RunStoreSuites(t, newFixture)
}

// newFixture builds a fresh set of fakes per test.
func newFixture(t *testing.T) repotest.Fixture {
	t.Helper()
	users := memauth.New()

	var seq int
	return repotest.Fixture{
		Auth:    users,
		Diary:   memdiary.New(),
		Gallery: memgallery.New(),
		Today:   memtoday.New(),
		Devices: memdevices.New(),
		NewUser: func(t *testing.T) string {
			t.Helper()
			seq++
			email := fixtureEmail(seq)
			u, err := users.CreateUser(t.Context(), repository.CreateUserParams{
				Email:        &email,
				AuthProvider: "email",
				DisplayName:  "Fixture",
			})
			if err != nil {
				t.Fatalf("creating a fixture user: %v", err)
			}
			return u.ID
		},
	}
}

func fixtureEmail(n int) string {
	return "fixture-" + string(rune('a'+n%26)) + string(rune('0'+n/26%10)) + "@example.com"
}
