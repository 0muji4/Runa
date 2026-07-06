// Package server assembles the HTTP router and middleware chain.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/0muji4/Runa/apps/go/internal/handler"
)

// requestTimeout bounds how long a single request may run before the server
// gives up. Health checks are instant; this mainly protects future handlers.
const requestTimeout = 30 * time.Second

// New builds the chi router with the standard middleware stack and mounts the
// versioned API routes.
//
// Why /api/v1 mount point: the client contract injects host+port only and
// appends /api/v1/... itself, so versioning lives entirely server-side and can
// evolve (v2) without changing the injected base URL.
func New(health *handler.Health, allowedOrigins []string, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Order matters (chi wraps outermost-first): RequestID first so every later
	// log line can reference it, RealIP to resolve the client address, then the
	// request logger OUTSIDE Recoverer so that even a panicking request still
	// emits one structured line — Recoverer (innermost) turns the panic into a
	// 500 that the logger then records.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.Timeout(requestTimeout))

	// Versioned API surface. New routes mount under this same subrouter.
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/healthz", health.Healthz)
		// TODO: mount feature routes here as they are implemented.
	})

	return r
}

// requestLogger emits one structured slog line per request with method, path,
// status, duration and the chi request id.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
