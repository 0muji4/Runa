// Package service holds the application-layer business logic. For the walking
// skeleton it only contains the health check; feature services land here later.
package service

import "context"

// HealthStatus is the value object returned by a liveness check.
type HealthStatus struct {
	// Status is "ok" when the process is alive and able to serve requests.
	Status string
}

// Health is the liveness/health service boundary.
//
// Why an interface: the handler depends on this abstraction rather than a
// concrete type, so readiness checks (DB, cache, downstream) can be added
// behind the same seam and swapped/mocked in tests.
type Health interface {
	// Check reports process liveness. It intentionally has NO dependencies so
	// that /healthz stays a pure liveness signal that never fails on infra.
	Check(ctx context.Context) HealthStatus
}

// health is the thin default implementation of Health.
type health struct{}

// NewHealth constructs the default liveness service.
func NewHealth() Health {
	return &health{}
}

// Check always reports "ok" for liveness.
//
// TODO(readiness): add a separate readiness method that pings the DB pool and
// any downstream dependencies once those exist. Keep liveness dependency-free.
func (h *health) Check(_ context.Context) HealthStatus {
	return HealthStatus{Status: "ok"}
}
