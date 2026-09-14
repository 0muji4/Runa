// Package repository is the data-access layer: persistence models, store
// interfaces and their pgx implementations.
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

// User is the persistence model for the users table.
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

// CreateUserParams carries the fields needed to insert a user.
type CreateUserParams struct {
	Email        *string
	AuthProvider string
	AppleSub     *string
	GoogleSub    *string
	DisplayName  string
	PasswordHash *string
}

// InsertRefreshTokenParams carries the fields needed to persist a refresh token.
type InsertRefreshTokenParams struct {
	UserID    string
	TokenHash string
	ExpiresAt time.Time
}

// AuthStore is the data-access boundary for authentication.
type AuthStore interface {
	CreateUser(ctx context.Context, p CreateUserParams) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	// GetUserByProviderSub looks a user up by ("apple"|"google", subject).
	GetUserByProviderSub(ctx context.Context, provider, sub string) (User, error)

	// UpdateDisplayName sets a user's display_name; ErrNotFound when the id matches no user.
	UpdateDisplayName(ctx context.Context, id, displayName string) (User, error)
	// DeleteUser permanently removes a user; the DB cascade removes their rows,
	// object-storage cleanup does NOT cascade. ErrNotFound when the id matches no user.
	DeleteUser(ctx context.Context, id string) error

	InsertRefreshToken(ctx context.Context, p InsertRefreshTokenParams) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	// RevokeRefreshToken marks a token revoked; revoking an unknown token is a no-op.
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}
