// Package mempush is a recording push.Sender for tests.
package mempush

import (
	"context"
	"sync"

	"github.com/0muji4/Runa/apps/go/internal/push"
)

// Sent is one recorded delivery.
type Sent struct {
	Token        string
	Notification push.Notification
}

// Sender records every Send; FailWith, when set, is returned instead of sending.
type Sender struct {
	mu       sync.Mutex
	sent     []Sent
	FailWith error
}

// New returns an empty recording sender.
func New() *Sender { return &Sender{} }

var _ push.Sender = (*Sender)(nil)

func (s *Sender) Send(_ context.Context, token string, n push.Notification) (push.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.FailWith != nil {
		return push.Result{}, s.FailWith
	}
	s.sent = append(s.sent, Sent{Token: token, Notification: n})
	return push.Result{ProviderMessageID: "mem-" + token}, nil
}

// Sent returns a copy of everything delivered so far.
func (s *Sender) Sent() []Sent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Sent(nil), s.sent...)
}
