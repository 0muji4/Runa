// Package repository is the data-access layer. It owns the database pool and
// exposes typed repositories to the service layer. For the walking skeleton the
// repositories are stubs — only the health path is wired end to end.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository is a thin holder around the Postgres connection pool. Concrete
// repositories are constructed from it so they share a single pool.
//
// TODO: as features land, expose repository accessors here (e.g. Users()) or
// inject the pool into individual repository constructors.
type Repository struct {
	pool *pgxpool.Pool
}

// New wraps a pgx pool. The pool may be nil when the DB is unreachable at boot;
// callers must not assume a live connection (liveness stays DB-independent).
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Pool exposes the underlying pool for future repositories. May be nil.
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// User is the persistence model matching the users table (migration 0001).
//
// TODO: expand columns as the product schema grows.
type User struct {
	ID        string // uuid primary key
	CreatedAt string // timestamptz; string placeholder until a real time type is needed
}

// UserRepository is the data-access boundary for the users table. Bodies are
// intentionally unimplemented — no feature logic exists in the skeleton.
type UserRepository interface {
	// Create inserts a new user and returns it.
	// TODO: implement against the users table.
	Create(ctx context.Context) (User, error)
	// GetByID loads a user by its uuid.
	// TODO: implement against the users table.
	GetByID(ctx context.Context, id string) (User, error)
}
