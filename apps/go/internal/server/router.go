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

// Deps carries the handlers and middleware the router mounts. Passing a struct
// keeps New's signature stable as feature slices add their own routes.
type Deps struct {
	Health   *handler.Health
	Auth     *handler.Auth
	Account  *handler.Account
	Diary    *handler.Diary
	Today    *handler.Today
	Insights *handler.Insights
	Gallery  *handler.Gallery
	Devices  *handler.Devices
	// RequireAuth guards Bearer-protected routes (verifies the access token).
	RequireAuth func(http.Handler) http.Handler
	// AuthRateLimit throttles the credential endpoints (signup/login).
	AuthRateLimit func(http.Handler) http.Handler
	// RequireAdmin gates the curated seed endpoints behind the admin token.
	RequireAdmin   func(http.Handler) http.Handler
	AllowedOrigins []string
	Logger         *slog.Logger
}

// New builds the chi router with the standard middleware stack and mounts the
// versioned API routes.
//
// Why /api/v1 mount point: the client contract injects host+port only and
// appends /api/v1/... itself, so versioning lives entirely server-side and can
// evolve (v2) without changing the injected base URL.
func New(deps Deps) *chi.Mux {
	r := chi.NewRouter()

	// Order matters (chi wraps outermost-first): RequestID first so every later
	// log line can reference it, RealIP to resolve the client address, then the
	// request logger OUTSIDE Recoverer so that even a panicking request still
	// emits one structured line — Recoverer (innermost) turns the panic into a
	// 500 that the logger then records.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.AllowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.Timeout(requestTimeout))

	// Versioned API surface. New routes mount under this same subrouter.
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/healthz", deps.Health.Healthz)

		api.Route("/auth", func(ar chi.Router) {
			// Credential endpoints are rate limited to blunt brute force.
			ar.Group(func(rl chi.Router) {
				rl.Use(deps.AuthRateLimit)
				rl.Post("/signup", deps.Auth.Signup)
				rl.Post("/login", deps.Auth.Login)
			})
			ar.Post("/apple", deps.Auth.Apple)
			ar.Post("/google", deps.Auth.Google)
			ar.Post("/refresh", deps.Auth.Refresh)
			ar.Post("/logout", deps.Auth.Logout)
		})

		// Bearer-protected routes.
		api.Group(func(pr chi.Router) {
			pr.Use(deps.RequireAuth)
			pr.Get("/me", deps.Auth.Me)

			// Account-data management shares the /me resource but lives on its own
			// handler (profile edit, export, deletion span multiple stores). Guarded
			// like the admin routes so test routers that omit it don't register a nil
			// handler. "/me/export" is a distinct path from "/me"; chi routes both.
			if deps.Account != nil {
				pr.Patch("/me", deps.Account.UpdateMe)
				pr.Delete("/me", deps.Account.DeleteMe)
				pr.Get("/me/export", deps.Account.Export)
			}

			// Diary: all endpoints are Bearer-protected and scoped to the caller.
			// Registered flat (not via Route("/diary")) so the collection matches
			// "/diary" with no trailing slash, which is what the clients send. chi
			// prefers the static "/diary/sync" over the "/diary/{id}" wildcard.
			pr.Get("/diary", deps.Diary.List)
			pr.Post("/diary", deps.Diary.Create)
			pr.Get("/diary/sync", deps.Diary.Sync)
			pr.Get("/diary/calendar", deps.Diary.Calendar)
			pr.Get("/diary/{id}", deps.Diary.Get)
			pr.Patch("/diary/{id}", deps.Diary.Update)
			pr.Delete("/diary/{id}", deps.Diary.Delete)

			// Today: the home payload (daily quote + song), the song archive and
			// the play log. The moon phase is computed client-side, so it has no
			// route here. "/songs" is static and "/songs/{id}/played" is a fixed
			// suffix, so registration order does not collide.
			pr.Get("/today", deps.Today.Today)
			pr.Get("/songs", deps.Today.Songs)
			pr.Post("/songs/{id}/played", deps.Today.Played)

			// Insights: the auxiliary server-side per-period aggregation. The client
			// still renders from its own local aggregation; this is the count of
			// record for a future server summary / cross-device path.
			pr.Get("/insights", deps.Insights.Insights)

			// Gallery: image metadata + presigned URLs (the bytes go client↔store
			// directly, never through here). The static "/gallery/upload-url" is
			// registered before the "/gallery/{id}" wildcard so it never collides.
			pr.Post("/gallery/upload-url", deps.Gallery.UploadURL)
			pr.Get("/gallery", deps.Gallery.List)
			pr.Post("/gallery", deps.Gallery.Create)
			pr.Get("/gallery/{id}", deps.Gallery.Get)
			pr.Delete("/gallery/{id}", deps.Gallery.Delete)

			// Devices: register a push token + reminder preference for FUTURE
			// server-initiated notifications. This slice's nightly reminder is a
			// local, on-device notification, so nothing here sends a push yet.
			// Guarded like Account so a test router that omits it never registers a
			// nil handler (chi would panic).
			if deps.Devices != nil {
				pr.Put("/devices", deps.Devices.Register)
			}
		})

		// Admin seed endpoints: curated content injection, gated by the shared
		// admin token (X-Admin-Token) rather than a user session. Mounted only
		// when the gate is wired (always in production; omitted by test routers
		// that don't exercise admin), since chi panics on a nil middleware.
		if deps.RequireAdmin != nil {
			api.Group(func(ad chi.Router) {
				ad.Use(deps.RequireAdmin)
				ad.Post("/admin/quotes", deps.Today.CreateQuote)
				ad.Post("/admin/songs", deps.Today.CreateSong)
			})
		}
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
