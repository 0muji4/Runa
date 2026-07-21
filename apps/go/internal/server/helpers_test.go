package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/handler"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdevices"
	"github.com/0muji4/Runa/apps/go/internal/repository/memdiary"
	"github.com/0muji4/Runa/apps/go/internal/repository/memgallery"
	"github.com/0muji4/Runa/apps/go/internal/repository/memtoday"
	"github.com/0muji4/Runa/apps/go/internal/server"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/0muji4/Runa/apps/go/internal/storage/memobject"
)

const adminToken = "seed-secret"

type testEnv struct {
	r       http.Handler
	objects *memobject.Store
	users   *memauth.Store
	diaries *memdiary.Store
	gallery *memgallery.Store
	today   *memtoday.Store
	devices *memdevices.Store
}

func newRouter(t *testing.T) *testEnv {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	issuer := auth.NewTokenIssuer("test-secret", time.Minute)

	users := memauth.New()
	diaries := memdiary.New()
	gallery := memgallery.New()
	todayStore := memtoday.New()
	devices := memdevices.New()
	objects := memobject.New()

	authSvc := service.NewAuthService(service.AuthConfig{
		Store:          users,
		Issuer:         issuer,
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	ah := handler.NewAuth(authSvc, logger)

	dh := handler.NewDiary(service.NewDiaryService(diaries, nil), logger)
	ih := handler.NewInsights(service.NewInsightsService(diaries), logger)

	th := handler.NewToday(service.NewTodayService(todayStore, nil), logger)

	gs := service.NewGalleryService(gallery, objects, service.GalleryConfig{
		UploadURLTTL:        15 * time.Minute,
		ViewURLTTL:          60 * time.Minute,
		MaxUploadBytes:      10 * 1024 * 1024,
		AllowedContentTypes: []string{"image/jpeg", "image/png", "image/webp", "image/heic"},
	}, nil, service.WithBackgroundRunner(func(f func()) { f() }))
	gh := handler.NewGallery(gs, logger)

	accountSvc := service.NewAccountService(users, diaries, gallery, objects, service.AccountConfig{
		ExportURLTTL: time.Hour,
	}, nil, service.WithAccountBackgroundRunner(func(f func()) { f() }))
	acc := handler.NewAccount(accountSvc, logger)

	dvh := handler.NewDevices(service.NewDeviceService(devices, nil), logger)

	r := server.New(server.Deps{
		Health:         handler.NewHealth(service.NewHealth(), logger),
		Auth:           ah,
		Account:        acc,
		Diary:          dh,
		Today:          th,
		Insights:       ih,
		Gallery:        gh,
		Devices:        dvh,
		RequireAuth:    auth.RequireAuth(issuer, ah.Unauthorized),
		AuthRateLimit:  auth.NewRateLimiter(100, time.Minute).Middleware(ah.RateLimited),
		RequireAdmin:   auth.RequireAdmin(adminToken, th.Forbidden),
		AllowedOrigins: []string{"*"},
		Logger:         logger,
	})

	return &testEnv{
		r:       r,
		objects: objects,
		users:   users,
		diaries: diaries,
		gallery: gallery,
		today:   todayStore,
		devices: devices,
	}
}

func do(t *testing.T, r http.Handler, method, path, bearer, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Result()
}

func doAdmin(t *testing.T, r http.Handler, method, path, token, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		req.Header.Set("X-Admin-Token", token)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Result()
}

func decode(t *testing.T, res *http.Response, dst any) {
	t.Helper()
	defer res.Body.Close()
	require.NoError(t, json.NewDecoder(res.Body).Decode(dst))
}

func signupToken(t *testing.T, r http.Handler, email string) string {
	t.Helper()
	res := do(t, r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"`+email+`","password":"password123","display_name":"U"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	var tok tokens
	decode(t, res, &tok)
	return tok.AccessToken
}

func createOn(t *testing.T, r http.Handler, token, clientID, createdAt string) {
	t.Helper()
	res := do(t, r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月あかり","client_id":"`+clientID+`","created_at":"`+createdAt+`"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	res.Body.Close()
}

func createMood(t *testing.T, r http.Handler, token, clientID, createdAt, mood string) {
	t.Helper()
	res := do(t, r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月あかり","client_id":"`+clientID+`","created_at":"`+createdAt+`","mood":"`+mood+`"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	res.Body.Close()
}

type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
}

type diaryEntry struct {
	ID        string  `json:"id"`
	ClientID  string  `json:"client_id"`
	BodyText  string  `json:"body_text"`
	Mood      *string `json:"mood"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	DeletedAt *string `json:"deleted_at"`
}

type diaryList struct {
	Entries    []diaryEntry `json:"entries"`
	NextCursor *string      `json:"next_cursor"`
}

type diarySync struct {
	Entries    []diaryEntry `json:"entries"`
	ServerTime string       `json:"server_time"`
}

type calendarResponse struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Days  []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	} `json:"days"`
}

type insightsResponse struct {
	Period           string `json:"period"`
	Start            string `json:"start"`
	DaysJournaled    int    `json:"days_journaled"`
	EntryCount       int    `json:"entry_count"`
	UnmoodedCount    int    `json:"unmooded_count"`
	MoodDistribution []struct {
		Mood  string `json:"mood"`
		Count int    `json:"count"`
	} `json:"mood_distribution"`
}

func (i insightsResponse) moods() map[string]int {
	m := make(map[string]int, len(i.MoodDistribution))
	for _, md := range i.MoodDistribution {
		m[md.Mood] = md.Count
	}
	return m
}

type galleryImage struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	URLExpiresAt string `json:"url_expires_at"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Theme        string `json:"theme"`
	CreatedAt    string `json:"created_at"`
}

type galleryUploadURL struct {
	ObjectKey string            `json:"object_key"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt string            `json:"expires_at"`
	MaxSize   int64             `json:"max_size"`
}

type galleryList struct {
	Items      []galleryImage `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}

type quoteResp struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	BodyText string `json:"body_text"`
}

type songResp struct {
	ID         string `json:"id"`
	Date       string `json:"date"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	ArtworkURL string `json:"artwork_url"`
	AudioURL   string `json:"audio_url"`
}

type todayResp struct {
	Date  string     `json:"date"`
	Quote *quoteResp `json:"quote"`
	Song  *songResp  `json:"song"`
}

type songsResp struct {
	Songs      []songResp `json:"songs"`
	NextCursor *string    `json:"next_cursor"`
}

type deviceResp struct {
	ID         string `json:"id"`
	PushToken  string `json:"push_token"`
	Platform   string `json:"platform"`
	NotifyTime string `json:"notify_time"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}
