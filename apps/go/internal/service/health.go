// Package service holds the application-layer business logic.
package service

import "context"

// HealthStatus is the value object returned by a liveness check.
type HealthStatus struct {
	Status string
}

// Health is the liveness/health service boundary.
type Health interface {
	// Check reports process liveness; keep it dependency-free so /healthz never fails on infra.
	Check(ctx context.Context) HealthStatus
}

type health struct{}

// NewHealth constructs the default liveness service.
func NewHealth() Health {
	return &health{}
}

// Check always reports "ok" for liveness.
func (h *health) Check(_ context.Context) HealthStatus {
	return HealthStatus{Status: "ok"}
}
