// Package handler contains the HTTP transport layer: it translates between
// HTTP requests/responses and the service layer. No business logic lives here.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/0muji4/Runa/apps/go/internal/service"
)

// healthzResponse is the JSON body of GET /api/v1/healthz.
//
// It mirrors the shared/UI contract HealthzResponse(status: String) so the
// Kotlin/Swift client can deserialize it directly.
type healthzResponse struct {
	Status string `json:"status"`
}

// Health is the HTTP handler for health endpoints, wired to the Health service.
type Health struct {
	svc    service.Health
	logger *slog.Logger
}

// NewHealth constructs the health handler from its service dependency.
func NewHealth(svc service.Health, logger *slog.Logger) *Health {
	return &Health{svc: svc, logger: logger}
}

// Healthz handles GET /api/v1/healthz and returns 200 with {"status":"ok"}.
func (h *Health) Healthz(w http.ResponseWriter, r *http.Request) {
	status := h.svc.Check(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(healthzResponse{Status: status.Status}); err != nil {
		// Response is already committed; we can only log the encode failure.
		h.logger.ErrorContext(r.Context(), "failed to encode healthz response", slog.Any("error", err))
	}
}
