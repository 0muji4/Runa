// Command api is the Runa backend HTTP server entry point.
//
// Boot sequence:
//  1. load config from env
//  2. init a JSON slog logger at the configured level
//  3. try to open a pgx pool with a short retry loop, but DO NOT fail if the DB
//     is unreachable — /healthz is a pure liveness check and must keep serving
//  4. if the pool pings, run golang-migrate Up() (ErrNoChange = success)
//  5. build the chi router and serve with graceful shutdown on SIGINT/SIGTERM
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/config"
	"github.com/0muji4/Runa/apps/go/internal/handler"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/server"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

const (
	// authRateLimitMax / authRateLimitWindow throttle signup/login per client IP.
	authRateLimitMax    = 10
	authRateLimitWindow = time.Minute
)

const (
	// dbConnectAttempts / dbConnectBackoff bound the startup retry loop so a
	// slow-starting Postgres (docker compose) is tolerated without blocking boot.
	dbConnectAttempts = 5
	dbConnectBackoff  = 2 * time.Second

	// migrationsPath is the golang-migrate file source. Migrations are copied
	// next to the binary in the container image (see Dockerfile).
	migrationsPath = "file://migrations"

	// shutdownTimeout bounds graceful shutdown before forced close.
	shutdownTimeout = 10 * time.Second
)

func main() {
	cfg := config.Load()
	logger := newLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Best-effort DB pool: nil when unreachable. Liveness never depends on it.
	pool := connectDB(ctx, cfg.DatabaseURL, logger)
	if pool != nil {
		defer pool.Close()
		runMigrations(cfg.DatabaseURL, logger)
	}

	healthHandler := handler.NewHealth(service.NewHealth(), logger)

	// Auth wiring. The repository tolerates a nil pool (returns ErrNoDatabase),
	// so the process still boots for liveness when the DB is down; the auth
	// endpoints themselves require a live DB.
	authRepo := repository.NewAuthRepository(pool)
	issuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := service.NewAuthService(service.AuthConfig{
		Store:          authRepo,
		Issuer:         issuer,
		Apple:          auth.NewOIDCVerifier(auth.AppleIssuers, cfg.AppleClientIDs, auth.NewRemoteJWKS(auth.AppleJWKSURL)),
		Google:         auth.NewOIDCVerifier(auth.GoogleIssuers, cfg.GoogleClientIDs, auth.NewRemoteJWKS(auth.GoogleJWKSURL)),
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     cfg.RefreshTokenTTL,
	})
	authHandler := handler.NewAuth(authService, logger)

	// Diary wiring. Same nil-pool tolerance as auth: the repository returns
	// ErrNoDatabase when the DB is down, so liveness still boots.
	diaryRepo := repository.NewDiaryRepository(pool)
	diaryService := service.NewDiaryService(diaryRepo, nil)
	diaryHandler := handler.NewDiary(diaryService, logger)

	// Today wiring (daily quote + song, archive, play log). Same nil-pool
	// tolerance as the other features.
	todayRepo := repository.NewTodayRepository(pool)
	todayService := service.NewTodayService(todayRepo, nil)
	todayHandler := handler.NewToday(todayService, logger)

	// Insights wiring: the auxiliary server-side aggregation reads the same diary
	// store (no new table). The client renders from its own local aggregation.
	insightsService := service.NewInsightsService(diaryRepo)
	insightsHandler := handler.NewInsights(insightsService, logger)

	// Gallery wiring. The object store is nil when S3_ENDPOINT is unset (the
	// gallery URL endpoints then answer 503) so the process still boots without
	// storage. When present, ensure the bucket exists (best-effort).
	objectStore := newObjectStore(ctx, cfg, logger)
	galleryRepo := repository.NewGalleryRepository(pool)
	galleryService := service.NewGalleryService(galleryRepo, objectStore, service.GalleryConfig{
		UploadURLTTL:        cfg.GalleryUploadURLTTL,
		ViewURLTTL:          cfg.GalleryViewURLTTL,
		MaxUploadBytes:      cfg.GalleryMaxUploadBytes,
		AllowedContentTypes: cfg.GalleryAllowedContentTypes,
	}, nil)
	galleryHandler := handler.NewGallery(galleryService, logger)

	// Account wiring: profile update, self-service export and account deletion.
	// It composes the auth, diary and gallery stores plus the object store because
	// "the account" spans all of them. Export presigns image URLs with the same
	// lifetime as gallery view URLs.
	accountService := service.NewAccountService(authRepo, diaryRepo, galleryRepo, objectStore, service.AccountConfig{
		ExportURLTTL: cfg.GalleryViewURLTTL,
	}, nil)
	accountHandler := handler.NewAccount(accountService, logger)

	router := server.New(server.Deps{
		Health:         healthHandler,
		Auth:           authHandler,
		Account:        accountHandler,
		Diary:          diaryHandler,
		Today:          todayHandler,
		Insights:       insightsHandler,
		Gallery:        galleryHandler,
		RequireAuth:    auth.RequireAuth(issuer, authHandler.Unauthorized),
		AuthRateLimit:  auth.NewRateLimiter(authRateLimitMax, authRateLimitWindow).Middleware(authHandler.RateLimited),
		RequireAdmin:   auth.RequireAdmin(cfg.AdminAPIToken, todayHandler.Forbidden),
		AllowedOrigins: cfg.CORSAllowedOrigins,
		Logger:         logger,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Serve in the background so main can wait on the shutdown signal.
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", slog.String("addr", srv.Addr), slog.String("env", cfg.AppEnv))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		logger.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}

// newLogger builds a JSON slog logger at the given level (default info).
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

// connectDB opens a pgx pool with a short retry loop. On persistent failure it
// logs a warning and returns nil so the server keeps serving liveness traffic.
func connectDB(ctx context.Context, url string, logger *slog.Logger) *pgxpool.Pool {
	for attempt := 1; attempt <= dbConnectAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				logger.Info("database connected")
				return pool
			} else {
				pool.Close()
				err = pingErr
			}
		}

		logger.Warn("database not ready",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", dbConnectAttempts),
			slog.Any("error", err),
		)

		if attempt == dbConnectAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(dbConnectBackoff):
		}
	}

	logger.Warn("continuing without database; /api/v1/healthz remains a liveness check")
	return nil
}

// newObjectStore builds the S3-compatible object store from config. It returns
// nil when storage is unconfigured (S3_ENDPOINT unset), so the server boots with
// the gallery URL endpoints disabled (503) rather than failing. When present it
// ensures the bucket exists (best-effort; a failure only defers bucket creation
// to first use).
func newObjectStore(ctx context.Context, cfg config.Config, logger *slog.Logger) storage.ObjectStore {
	store, err := storage.NewMinioObjectStore(storage.Config{
		Endpoint:       cfg.S3Endpoint,
		PublicEndpoint: cfg.S3PublicEndpoint,
		Region:         cfg.S3Region,
		Bucket:         cfg.S3Bucket,
		AccessKey:      cfg.S3AccessKey,
		SecretKey:      cfg.S3SecretKey,
		UseSSL:         cfg.S3UseSSL,
	})
	if err != nil {
		logger.Warn("object storage disabled: init failed", slog.Any("error", err))
		return nil
	}
	if store == nil {
		logger.Info("object storage not configured; gallery endpoints return 503")
		return nil
	}
	if err := store.EnsureBucket(ctx); err != nil {
		logger.Warn("could not ensure gallery bucket at boot", slog.Any("error", err))
	}
	return store
}

// runMigrations applies all up migrations. ErrNoChange is treated as success.
func runMigrations(databaseURL string, logger *slog.Logger) {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		logger.Error("failed to init migrations", slog.Any("error", err))
		return
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("failed to apply migrations", slog.Any("error", err))
		return
	}
	logger.Info("migrations applied")
}
