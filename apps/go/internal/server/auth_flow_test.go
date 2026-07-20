package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)

	res := do(t, env.r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"flow@example.com","password":"password123","display_name":"Flow"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	var signed tokens
	decode(t, res, &signed)
	assert.Equal(t, "Flow", signed.User.DisplayName)

	res = do(t, env.r, http.MethodGet, "/api/v1/me", signed.AccessToken, "")
	require.Equal(t, http.StatusOK, res.StatusCode)
	var me struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	decode(t, res, &me)
	assert.Equal(t, "flow@example.com", me.Email)
	assert.Equal(t, "Flow", me.DisplayName)

	res = do(t, env.r, http.MethodGet, "/api/v1/me", "", "")
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+signed.RefreshToken+`"}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var rotated tokens
	decode(t, res, &rotated)
	assert.NotEmpty(t, rotated.AccessToken)
	assert.NotEqual(t, signed.RefreshToken, rotated.RefreshToken)

	// 古い（回転前の）refreshトークンは単回使用で、再提示は401。
	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+signed.RefreshToken+`"}`)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/me", rotated.AccessToken, "")
	require.Equal(t, http.StatusOK, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodPost, "/api/v1/auth/logout", "",
		`{"refresh_token":"`+rotated.RefreshToken+`"}`)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res.Body.Close()

	// logout 後は失効済みで refresh できない。
	res = do(t, env.r, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refresh_token":"`+rotated.RefreshToken+`"}`)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()
}

func TestMeRejectsGarbageToken(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodGet, "/api/v1/me", "not-a-real-token", "")
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()
}
