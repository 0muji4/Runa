// Package repository is the data-access layer. It owns the database pool and
// exposes typed stores to the service layer. Persistence models and the store
// interfaces live here; concrete pgx implementations sit alongside (e.g.
// auth_repository.go).
package repository

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a lookup matches no row.
	ErrNotFound = errors.New("repository: not found")
	// ErrEmailTaken is returned when inserting a user whose email already exists.
	ErrEmailTaken = errors.New("repository: email already registered")
	// ErrNoDatabase is returned when the pool is nil (DB unreachable at boot).
	ErrNoDatabase = errors.New("repository: database not available")
)

// User is the persistence model for the users table (migrations 0001 + 0002).
// Nullable columns are pointers so an absent value is distinguishable from a
// zero value.
type User struct {
	ID               string
	Email            *string
	AuthProvider     string // "email" | "apple" | "google"
	AppleSub         *string
	GoogleSub        *string
	DisplayName      string
	PasswordHash     *string
	IsPremium        bool
	PremiumExpiresAt *time.Time
	CreatedAt        time.Time
}

// RefreshToken is the persistence model for the refresh_tokens table.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

// CreateUserParams carries the fields needed to insert a user. Email/AppleSub/
// GoogleSub/PasswordHash are optional depending on the auth provider.
type CreateUserParams struct {
	Email        *string
	AuthProvider string
	AppleSub     *string
	GoogleSub    *string
	DisplayName  string
	PasswordHash *string
}

// InsertRefreshTokenParams carries the fields needed to persist a refresh token
// (only its hash is stored).
type InsertRefreshTokenParams struct {
	UserID    string
	TokenHash string
	ExpiresAt time.Time
}

// AuthStore is the data-access boundary for authentication. The service depends
// on this interface so tests can substitute an in-memory fake.
type AuthStore interface {
	CreateUser(ctx context.Context, p CreateUserParams) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	// GetUserByProviderSub looks a user up by ("apple"|"google", subject).
	GetUserByProviderSub(ctx context.Context, provider, sub string) (User, error)

	// UpdateDisplayName sets a user's display_name and returns the updated row.
	// Returns ErrNotFound when the id matches no user.
	UpdateDisplayName(ctx context.Context, id, displayName string) (User, error)
	// DeleteUser permanently removes a user. The users table's ON DELETE CASCADE
	// (migrations 0002–0005) removes the caller's refresh_tokens, diary_entries,
	// gallery_images and song_history in the same statement; object-storage
	// cleanup is NOT cascaded and stays the service's concern. Returns ErrNotFound
	// when the id matches no user.
	DeleteUser(ctx context.Context, id string) error

	InsertRefreshToken(ctx context.Context, p InsertRefreshTokenParams) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	// RevokeRefreshToken marks a token revoked. Revoking an unknown token is a
	// no-op (logout is idempotent).
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}
