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

// requestTimeout bounds how long a single request may run.
const requestTimeout = 30 * time.Second

// Deps carries the handlers and middleware the router mounts.
type Deps struct {
	Health         *handler.Health
	Auth           *handler.Auth
	Account        *handler.Account
	Diary          *handler.Diary
	Today          *handler.Today
	Insights       *handler.Insights
	Gallery        *handler.Gallery
	Devices        *handler.Devices
	RequireAuth    func(http.Handler) http.Handler
	AuthRateLimit  func(http.Handler) http.Handler
	RequireAdmin   func(http.Handler) http.Handler
	AllowedOrigins []string
	Logger         *slog.Logger
}

// New builds the chi router with the standard middleware stack and mounts the /api/v1 routes.
func New(deps Deps) *chi.Mux {
	r := chi.NewRouter()

	// Order matters: RequestID before the logger, and the logger OUTSIDE Recoverer
	// so a panicking request still emits one log line (with the 500 Recoverer produces).
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

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/healthz", deps.Health.Healthz)

		api.Route("/auth", func(ar chi.Router) {
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

		api.Group(func(pr chi.Router) {
			pr.Use(deps.RequireAuth)
			pr.Get("/me", deps.Auth.Me)

			// Nil guards (here, Devices, RequireAdmin): test routers omit these, and chi panics on a nil handler.
			if deps.Account != nil {
				pr.Patch("/me", deps.Account.UpdateMe)
				pr.Delete("/me", deps.Account.DeleteMe)
				pr.Get("/me/export", deps.Account.Export)
			}

			// Registered flat (not via Route("/diary")) so "/diary" matches without a trailing slash.
			pr.Get("/diary", deps.Diary.List)
			pr.Post("/diary", deps.Diary.Create)
			pr.Get("/diary/sync", deps.Diary.Sync)
			pr.Get("/diary/calendar", deps.Diary.Calendar)
			pr.Get("/diary/{id}", deps.Diary.Get)
			pr.Patch("/diary/{id}", deps.Diary.Update)
			pr.Delete("/diary/{id}", deps.Diary.Delete)

			pr.Get("/today", deps.Today.Today)
			pr.Get("/songs", deps.Today.Songs)
			pr.Post("/songs/{id}/played", deps.Today.Played)

			pr.Get("/insights", deps.Insights.Insights)

			pr.Post("/gallery/upload-url", deps.Gallery.UploadURL)
			pr.Get("/gallery", deps.Gallery.List)
			pr.Post("/gallery", deps.Gallery.Create)
			pr.Get("/gallery/{id}", deps.Gallery.Get)
			pr.Delete("/gallery/{id}", deps.Gallery.Delete)

			if deps.Devices != nil {
				pr.Put("/devices", deps.Devices.Register)
			}
		})

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

// requestLogger emits one structured slog line per request.
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
