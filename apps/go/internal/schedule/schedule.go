// Package schedule defines the "call this URL at this time" seam the push
// reminder is built on; cloudtasks implements it, memschedule fakes it.
package schedule

import (
	"context"
	"time"
)

// Task is one scheduled HTTP callback into this server. Name is the
// idempotency key: scheduling an existing Name is a no-op, not an error.
type Task struct {
	Name  string
	RunAt time.Time
	// Path is relative to the server's public base URL, e.g. "/api/v1/hooks/push/reminder".
	Path string
	Body []byte
}

// Scheduler persists tasks with an external scheduler. Both methods are
// idempotent: Schedule tolerates an existing Name, Cancel a missing one.
type Scheduler interface {
	Schedule(ctx context.Context, t Task) error
	Cancel(ctx context.Context, name string) error
}
