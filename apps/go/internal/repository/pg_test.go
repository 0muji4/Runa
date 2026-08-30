// Package repository_test runs the repository contract suites against a real
// Postgres in a throwaway container. Without Docker, or under -short, they skip
// with a reason rather than failing.
package repository_test

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/repotest"
)

// templateDB holds the migrated schema; each test copies it with CREATE DATABASE
// ... TEMPLATE. Per-test databases, not just per-test rows, because the curated
// content tables are keyed by date globally rather than per user.
const templateDB = "runa_template"

var (
	adminURL   string
	skipReason string

	// Postgres refuses to copy a template another session is reading.
	createMu sync.Mutex
	dbSeq    atomic.Int64
)

func TestMain(m *testing.M) {
	// testing.Short() reads a flag, and TestMain runs before they are parsed.
	flag.Parse()

	if testing.Short() {
		skipReason = "-short is set; skipping the Postgres-backed contract suites"
		os.Exit(m.Run())
	}

	ctx := context.Background()
	teardown, err := startPostgres(ctx)
	if err != nil {
		skipReason = fmt.Sprintf(
			"could not start a Postgres container (%v); "+
				"start Docker to run the repository contract suites, or pass -short to skip them", err)
		os.Exit(m.Run())
	}

	code := m.Run()
	teardown()
	os.Exit(code)
}

// startPostgres boots the container and migrates the template database.
func startPostgres(ctx context.Context) (func(), error) {
	// Same major version as docker-compose.yml.
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("runa"),
		tcpostgres.WithUsername("runa"),
		tcpostgres.WithPassword("runa"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("run container: %w", err)
	}
	terminate := func() { _ = testcontainers.TerminateContainer(container) }

	adminURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminate()
		return nil, fmt.Errorf("connection string: %w", err)
	}

	if err := migrateTemplate(ctx); err != nil {
		terminate()
		return nil, fmt.Errorf("prepare template database: %w", err)
	}
	return terminate, nil
}

// migrateTemplate creates the template database and migrates it.
func migrateTemplate(ctx context.Context) error {
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		return fmt.Errorf("connect admin: %w", err)
	}
	defer admin.Close()

	if _, err := admin.Exec(ctx, "CREATE DATABASE "+templateDB); err != nil {
		return fmt.Errorf("create %s: %w", templateDB, err)
	}

	m, err := migrate.New("file://../../migrations", databaseURL(templateDB))
	if err != nil {
		return fmt.Errorf("init migrations: %w", err)
	}
	if err := m.Up(); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	// migrate holds a connection; CREATE DATABASE ... TEMPLATE fails until it goes.
	if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
		return fmt.Errorf("close migrator: %v / %v", srcErr, dbErr)
	}
	return nil
}

// databaseURL rewrites the admin URL to point at the named database.
func databaseURL(name string) string {
	u, err := url.Parse(adminURL)
	if err != nil {
		panic(fmt.Sprintf("parsing the container URL %q: %v", adminURL, err))
	}
	u.Path = "/" + name
	return u.String()
}

func TestPostgresStoresMeetTheContract(t *testing.T) {
	if skipReason != "" {
		t.Skip(skipReason)
	}
	t.Parallel()
	repotest.RunStoreSuites(t, newFixture)
}

// newFixture gives one test its own database, copied from the template.
func newFixture(t *testing.T) repotest.Fixture {
	t.Helper()

	name := fmt.Sprintf("runa_test_%d", dbSeq.Add(1))
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("connecting to the container: %v", err)
	}
	defer admin.Close()

	createMu.Lock()
	_, err = admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, templateDB))
	createMu.Unlock()
	if err != nil {
		t.Fatalf("creating the test database %s: %v", name, err)
	}

	pool, err := pgxpool.New(ctx, databaseURL(name))
	if err != nil {
		t.Fatalf("connecting to %s: %v", name, err)
	}
	t.Cleanup(pool.Close)

	users := repository.NewAuthRepository(pool)
	var seq atomic.Int64
	return repotest.Fixture{
		Auth:    users,
		Diary:   repository.NewDiaryRepository(pool),
		Gallery: repository.NewGalleryRepository(pool),
		Today:   repository.NewTodayRepository(pool),
		Devices: repository.NewDeviceRepository(pool),
		NewUser: func(t *testing.T) string {
			t.Helper()
			email := fmt.Sprintf("fixture-%d-%d@example.com", seq.Add(1), time.Now().UnixNano())
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
