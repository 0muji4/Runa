// Package memauth is an in-memory implementation of repository.AuthStore. It is
// used by the auth unit/integration tests (and is handy for running the API
// without Postgres) so the test suite stays green in CI, which has no database.
package memauth

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// Store is a goroutine-safe, in-memory AuthStore.
type Store struct {
	mu      sync.Mutex
	users   map[string]repository.User         // keyed by user id
	refresh map[string]repository.RefreshToken // keyed by token hash
	now     func() time.Time
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		users:   make(map[string]repository.User),
		refresh: make(map[string]repository.RefreshToken),
		now:     time.Now,
	}
}

var _ repository.AuthStore = (*Store)(nil)

func (s *Store) CreateUser(_ context.Context, p repository.CreateUserParams) (repository.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.Email != nil {
		for _, u := range s.users {
			if u.Email != nil && *u.Email == *p.Email {
				return repository.User{}, repository.ErrEmailTaken
			}
		}
	}

	u := repository.User{
		ID:           newID(),
		Email:        p.Email,
		AuthProvider: p.AuthProvider,
		AppleSub:     p.AppleSub,
		GoogleSub:    p.GoogleSub,
		DisplayName:  p.DisplayName,
		PasswordHash: p.PasswordHash,
		CreatedAt:    s.now().UTC(),
	}
	s.users[u.ID] = u
	return u, nil
}

func (s *Store) GetUserByID(_ context.Context, id string) (repository.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return repository.User{}, repository.ErrNotFound
}

func (s *Store) GetUserByEmail(_ context.Context, email string) (repository.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email != nil && *u.Email == email {
			return u, nil
		}
	}
	return repository.User{}, repository.ErrNotFound
}

func (s *Store) GetUserByProviderSub(_ context.Context, provider, sub string) (repository.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		switch provider {
		case "apple":
			if u.AppleSub != nil && *u.AppleSub == sub {
				return u, nil
			}
		case "google":
			if u.GoogleSub != nil && *u.GoogleSub == sub {
				return u, nil
			}
		}
	}
	return repository.User{}, repository.ErrNotFound
}

func (s *Store) UpdateDisplayName(_ context.Context, id, displayName string) (repository.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return repository.User{}, repository.ErrNotFound
	}
	u.DisplayName = displayName
	s.users[id] = u
	return u, nil
}

func (s *Store) DeleteUser(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return repository.ErrNotFound
	}
	delete(s.users, id)
	// Mirror the DB's ON DELETE CASCADE for refresh tokens so tests see a deleted user
	// can't refresh. Diary/gallery cascades are DB-level, exercised only against Postgres.
	for hash, t := range s.refresh {
		if t.UserID == id {
			delete(s.refresh, hash)
		}
	}
	return nil
}

func (s *Store) InsertRefreshToken(_ context.Context, p repository.InsertRefreshTokenParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh[p.TokenHash] = repository.RefreshToken{
		ID:        newID(),
		UserID:    p.UserID,
		TokenHash: p.TokenHash,
		ExpiresAt: p.ExpiresAt,
		CreatedAt: s.now().UTC(),
	}
	return nil
}

func (s *Store) GetRefreshTokenByHash(_ context.Context, tokenHash string) (repository.RefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.refresh[tokenHash]; ok {
		return t, nil
	}
	return repository.RefreshToken{}, repository.ErrNotFound
}

func (s *Store) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.refresh[tokenHash]; ok {
		t.Revoked = true
		s.refresh[tokenHash] = t
	}
	return nil
}

// newID returns a random v4-style UUID string without pulling in a dependency.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
