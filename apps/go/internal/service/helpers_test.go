package service_test

import (
	"fmt"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/repository/memtoday"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage"
)

const (
	userA = "11111111-1111-4111-8111-111111111111"
	userB = "22222222-2222-4222-8222-222222222222"
)

// testNow is the frozen clock injected wherever an output timestamp is asserted
// (presigned-URL expiries, the export timestamp), keeping those assertions
// deterministic.
var testNow = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func fixedNow() time.Time { return testNow }

func ptr(s string) *string { return &s }

// clientID builds a deterministic, unique, UUID-shaped client id for n.
func clientID(n int) string {
	return fmt.Sprintf("aaaaaaaa-aaaa-4aaa-8aaa-%012d", n)
}

// syncBackground runs deferred work inline so a test can assert its effect.
func syncBackground(f func()) { f() }

func newDiaryService() *service.DiaryService {
	return service.NewDiaryService(memdiary.New(), nil)
}

func newAuthService(store repository.AuthStore, apple, google auth.IDTokenVerifier) *service.AuthService {
	return service.NewAuthService(service.AuthConfig{
		Store:          store,
		Issuer:         auth.NewTokenIssuer("test-secret", time.Minute),
		Apple:          apple,
		Google:         google,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
}

func newTodayService() *service.TodayService {
	return service.NewTodayService(memtoday.New(), nil)
}

func newAccountService(objects storage.ObjectStore) (*service.AccountService, *memauth.Store, *memdiary.Store, *memgallery.Store) {
	users := memauth.New()
	diaries := memdiary.New()
	gallery := memgallery.New()
	svc := service.NewAccountService(users, diaries, gallery, objects,
		service.AccountConfig{ExportURLTTL: time.Hour}, fixedNow,
		service.WithAccountBackgroundRunner(syncBackground))
	return svc, users, diaries, gallery
}

func newGalleryService(objects storage.ObjectStore, cfg service.GalleryConfig) (*service.GalleryService, *memgallery.Store) {
	store := memgallery.New()
	svc := service.NewGalleryService(store, objects, cfg, fixedNow,
		service.WithBackgroundRunner(syncBackground))
	return svc, store
}

func newInsightsService() (*service.InsightsService, *memdiary.Store) {
	store := memdiary.New()
	return service.NewInsightsService(store), store
}
