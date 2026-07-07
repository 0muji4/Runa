package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolation is the PostgreSQL SQLSTATE for a unique-constraint violation.
const uniqueViolation = "23505"

// userColumns is the SELECT list shared by every user query, kept in one place
// so the scan order in scanUser stays in sync.
const userColumns = `id, email, auth_provider, apple_sub, google_sub,
	display_name, password_hash, is_premium, premium_expires_at, created_at`

// AuthRepository is the pgx-backed implementation of AuthStore.
type AuthRepository struct {
	pool *pgxpool.Pool
}

// NewAuthRepository wraps a pgx pool. The pool may be nil when the DB is
// unreachable at boot; in that case every method returns ErrNoDatabase rather
// than panicking, so the process still serves liveness traffic.
func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

var _ AuthStore = (*AuthRepository)(nil)

// row is satisfied by both pgx.Row and pgx.Rows, so scanUser works for single
// and multi-row queries.
type row interface {
	Scan(dest ...any) error
}

func scanUser(r row) (User, error) {
	var u User
	if err := r.Scan(
		&u.ID, &u.Email, &u.AuthProvider, &u.AppleSub, &u.GoogleSub,
		&u.DisplayName, &u.PasswordHash, &u.IsPremium, &u.PremiumExpiresAt, &u.CreatedAt,
	); err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *AuthRepository) CreateUser(ctx context.Context, p CreateUserParams) (User, error) {
	if r.pool == nil {
		return User{}, ErrNoDatabase
	}

	const q = `
		INSERT INTO users (email, auth_provider, apple_sub, google_sub, display_name, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + userColumns

	u, err := scanUser(r.pool.QueryRow(ctx, q,
		p.Email, p.AuthProvider, p.AppleSub, p.GoogleSub, p.DisplayName, p.PasswordHash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, id string) (User, error) {
	return r.getUserBy(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	return r.getUserBy(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
}

func (r *AuthRepository) GetUserByProviderSub(ctx context.Context, provider, sub string) (User, error) {
	var column string
	switch provider {
	case "apple":
		column = "apple_sub"
	case "google":
		column = "google_sub"
	default:
		return User{}, fmt.Errorf("unknown provider %q", provider)
	}
	// column is chosen from a fixed allow-list above, never from user input.
	return r.getUserBy(ctx, `SELECT `+userColumns+` FROM users WHERE `+column+` = $1`, sub)
}

func (r *AuthRepository) getUserBy(ctx context.Context, query string, arg any) (User, error) {
	if r.pool == nil {
		return User{}, ErrNoDatabase
	}
	u, err := scanUser(r.pool.QueryRow(ctx, query, arg))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (r *AuthRepository) InsertRefreshToken(ctx context.Context, p InsertRefreshTokenParams) error {
	if r.pool == nil {
		return ErrNoDatabase
	}
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, q, p.UserID, p.TokenHash, p.ExpiresAt); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	if r.pool == nil {
		return RefreshToken{}, ErrNoDatabase
	}
	const q = `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens WHERE token_hash = $1`

	var t RefreshToken
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshToken{}, ErrNotFound
		}
		return RefreshToken{}, fmt.Errorf("get refresh token: %w", err)
	}
	return t, nil
}

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if r.pool == nil {
		return ErrNoDatabase
	}
	const q = `UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`
	if _, err := r.pool.Exec(ctx, q, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}
