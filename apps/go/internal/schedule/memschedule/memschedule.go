// Package memschedule is an in-memory schedule.Scheduler for tests.
package memschedule

import (
	"context"
	"sync"

	"github.com/0muji4/Runa/apps/go/internal/schedule"
)

// Scheduler keeps tasks by name and remembers cancellations.
type Scheduler struct {
	mu        sync.Mutex
	tasks     map[string]schedule.Task
	cancelled []string
	FailWith  error
}

// New returns an empty scheduler.
func New() *Scheduler {
	return &Scheduler{tasks: make(map[string]schedule.Task)}
}

var _ schedule.Scheduler = (*Scheduler)(nil)

func (s *Scheduler) Schedule(_ context.Context, t schedule.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.FailWith != nil {
		return s.FailWith
	}
	if _, exists := s.tasks[t.Name]; !exists {
		s.tasks[t.Name] = t
	}
	return nil
}

func (s *Scheduler) Cancel(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.FailWith != nil {
		return s.FailWith
	}
	delete(s.tasks, name)
	s.cancelled = append(s.cancelled, name)
	return nil
}

// Tasks returns the pending tasks, in unspecified order.
func (s *Scheduler) Tasks() []schedule.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]schedule.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out
}

// Task returns the pending task with that name.
func (s *Scheduler) Task(name string) (schedule.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[name]
	return t, ok
}

// Cancelled returns every name Cancel was called with, in order.
func (s *Scheduler) Cancelled() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.cancelled...)
}

// Pop removes and returns the pending task with that name (simulates the scheduler firing it).
func (s *Scheduler) Pop(name string) (schedule.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[name]
	delete(s.tasks, name)
	return t, ok
}
