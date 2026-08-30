package server_test

import (
	"net/http"
	"testing"
)

func TestAuthFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)

	res := do(t, env.r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"flow@example.com","password":"password123","display_name":"Flow"}`)
	checkStatus(t, res, http.StatusCreated)
	var signed tokens
	decode(t, res, &signed)
	if got, want := signed.User.DisplayName, "Flow"; got != want {
		t.Errorf("signup display_name = %q, want %q", got, want)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/me", signed.AccessToken, "")
	checkStatus(t, res, http.StatusOK)
	var me struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &me)
	if got, want := me.Email, "flow@example.com"; got != want {
		t.Errorf("GET /api/v1/me email = %q, want %q", got, want)
	}
	if got, want := me.DisplayName, "Flow"; got != want {
		t.Errorf("GET /api/v1/me display_name = %q, want %q", got, want)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/me", "", "")
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()

	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+signed.RefreshToken+`"}`)
	checkStatus(t, res, http.StatusOK)
	var rotated tokens
	decode(t, res, &rotated)
	if rotated.AccessToken == "" {
		t.Error("refresh returned an empty access_token, want a token")
	}
	if rotated.RefreshToken == signed.RefreshToken {
		t.Error("refresh returned the same refresh_token; it must rotate")
	}

	// 古い（回転前の）refreshトークンは単回使用で、再提示は401。
	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+signed.RefreshToken+`"}`)
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/me", rotated.AccessToken, "")
	checkStatus(t, res, http.StatusOK)
	res.Body.Close()

	res = do(t, env.r, http.MethodPost, "/api/v1/auth/logout", "",
		`{"refresh_token":"`+rotated.RefreshToken+`"}`)
	checkStatus(t, res, http.StatusNoContent)
	res.Body.Close()

	// logout 後は失効済みで refresh できない。
	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+rotated.RefreshToken+`"}`)
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()
}

func TestMeRejectsGarbageToken(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodGet, "/api/v1/me", "not-a-real-token", "")
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()
}
