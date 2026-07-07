package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository/memauth"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

func newAuthHandler() *Auth {
	svc := service.NewAuthService(service.AuthConfig{
		Store:          memauth.New(),
		Issuer:         auth.NewTokenIssuer("test-secret", time.Minute),
		PasswordParams: auth.DefaultArgon2Params(),
		RefreshTTL:     time.Hour,
	})
	return NewAuth(svc, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func postJSON(t *testing.T, h http.HandlerFunc, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec.Result()
}

func TestSignupSuccess(t *testing.T) {
	h := newAuthHandler()
	res := postJSON(t, h.Signup, `{"email":"a@b.com","password":"password123","display_name":"Runa"}`)
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got authTokensResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.AccessToken == "" || got.RefreshToken == "" {
		t.Error("expected non-empty access and refresh tokens")
	}
	if got.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want Bearer", got.TokenType)
	}
	if got.User == nil || got.User.Email == nil || *got.User.Email != "a@b.com" {
		t.Errorf("user = %+v, want email a@b.com", got.User)
	}
}

func TestSignupValidation(t *testing.T) {
	h := newAuthHandler()
	res := postJSON(t, h.Signup, `{"email":"not-an-email","password":"short"}`)
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Code != CodeValidation {
		t.Errorf("code = %q, want %q", env.Error.Code, CodeValidation)
	}
	if len(env.Error.Details) != 2 {
		t.Errorf("details = %v, want 2 field errors (email + password)", env.Error.Details)
	}
}

func TestSignupDuplicateEmailConflict(t *testing.T) {
	h := newAuthHandler()
	body := `{"email":"dup@b.com","password":"password123"}`
	first := postJSON(t, h.Signup, body)
	first.Body.Close()

	res := postJSON(t, h.Signup, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
	var env errorEnvelope
	_ = json.NewDecoder(res.Body).Decode(&env)
	if env.Error.Code != CodeEmailTaken {
		t.Errorf("code = %q, want %q", env.Error.Code, CodeEmailTaken)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	h := newAuthHandler()
	signup := postJSON(t, h.Signup, `{"email":"c@b.com","password":"password123"}`)
	signup.Body.Close()

	res := postJSON(t, h.Login, `{"email":"c@b.com","password":"wrongpass"}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
	var env errorEnvelope
	_ = json.NewDecoder(res.Body).Decode(&env)
	if env.Error.Code != CodeInvalidCredentials {
		t.Errorf("code = %q, want %q", env.Error.Code, CodeInvalidCredentials)
	}
}

func TestSignupRejectsUnknownFields(t *testing.T) {
	h := newAuthHandler()
	res := postJSON(t, h.Signup, `{"email":"a@b.com","password":"password123","role":"admin"}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for unknown field", res.StatusCode)
	}
}

// Guard against accidental whitespace-only display names sneaking through.
func TestSignupTrimsDisplayName(t *testing.T) {
	h := newAuthHandler()
	res := postJSON(t, h.Signup, `{"email":"trim@b.com","password":"password123","display_name":"`+strings.Repeat(" ", 3)+`"}`)
	defer res.Body.Close()
	var got authTokensResponse
	_ = json.NewDecoder(res.Body).Decode(&got)
	if got.User != nil && got.User.DisplayName != "trim" {
		t.Errorf("display_name = %q, want derived %q", got.User.DisplayName, "trim")
	}
}
