// Command api is the Runa backend HTTP server entry point.
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
	"github.com/0muji4/Runa/apps/go/internal/itunes"
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
	// dbConnectAttempts / dbConnectBackoff bound the startup retry loop.
	dbConnectAttempts = 5
	dbConnectBackoff  = 2 * time.Second

	// migrationsPath is relative to the binary; the container image copies migrations next to it.
	migrationsPath = "file://migrations"

	// itunesTimeout bounds one iTunes lookup plus its artwork check.
	itunesTimeout = 5 * time.Second

	shutdownTimeout = 10 * time.Second
)

func main() {
	cfg := config.Load()
	logger := newLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Best-effort DB pool: nil when unreachable. Boot and /healthz must not depend on it.
	pool := connectDB(ctx, cfg.DatabaseURL, logger)
	if pool != nil {
		defer pool.Close()
		runMigrations(cfg.DatabaseURL, logger)
	}

	healthHandler := handler.NewHealth(service.NewHealth(), logger)

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

	diaryRepo := repository.NewDiaryRepository(pool)
	diaryService := service.NewDiaryService(diaryRepo, nil)
	diaryHandler := handler.NewDiary(diaryService, logger)

	todayRepo := repository.NewTodayRepository(pool)
	todayService := service.NewTodayService(todayRepo, itunes.NewClient(cfg.ITunesBaseURL, &http.Client{Timeout: itunesTimeout}), nil,
		service.WithTodayLogger(logger))
	todayHandler := handler.NewToday(todayService, logger)

	insightsService := service.NewInsightsService(diaryRepo)
	insightsHandler := handler.NewInsights(insightsService, logger)

	objectStore := newObjectStore(ctx, cfg, logger)
	galleryRepo := repository.NewGalleryRepository(pool)
	galleryService := service.NewGalleryService(galleryRepo, objectStore, service.GalleryConfig{
		UploadURLTTL:        cfg.GalleryUploadURLTTL,
		ViewURLTTL:          cfg.GalleryViewURLTTL,
		MaxUploadBytes:      cfg.GalleryMaxUploadBytes,
		AllowedContentTypes: cfg.GalleryAllowedContentTypes,
	}, nil)
	galleryHandler := handler.NewGallery(galleryService, logger)

	accountService := service.NewAccountService(authRepo, diaryRepo, galleryRepo, objectStore, service.AccountConfig{
		ExportURLTTL: cfg.GalleryViewURLTTL,
	}, nil)
	accountHandler := handler.NewAccount(accountService, logger)

	deviceRepo := repository.NewDeviceRepository(pool)
	deviceService := service.NewDeviceService(deviceRepo, nil)
	deviceHandler := handler.NewDevices(deviceService, logger)

	router := server.New(server.Deps{
		Health:         healthHandler,
		Auth:           authHandler,
		Account:        accountHandler,
		Diary:          diaryHandler,
		Today:          todayHandler,
		Insights:       insightsHandler,
		Gallery:        galleryHandler,
		Devices:        deviceHandler,
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

// connectDB opens a pgx pool with a short retry loop; on persistent failure it returns nil.
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

// newObjectStore builds the object store from config, or returns nil when S3_ENDPOINT is
// unset (gallery URL endpoints then answer 503). Bucket creation is best-effort.
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
