package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevicesRegisterFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "devices@example.com")

	// 初回登録は200で作成される。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", token,
		`{"push_token":"token-abc","platform":"ios","notify_time":"22:00","enabled":true}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var created deviceResp
	decode(t, res, &created)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "token-abc", created.PushToken)
	assert.Equal(t, "ios", created.Platform)
	assert.Equal(t, "22:00", created.NotifyTime)
	assert.True(t, created.Enabled)

	// 同一トークンの再PUTは冪等upsert：同じidのまま設定が更新される。
	res = do(t, env.r, http.MethodPut, "/api/v1/devices", token,
		`{"push_token":"token-abc","platform":"ios","notify_time":"23:00","enabled":false}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var updated deviceResp
	decode(t, res, &updated)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "23:00", updated.NotifyTime)
	assert.False(t, updated.Enabled)
}

func TestDevicesRegisterValidation(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "devices-validate@example.com")

	tests := []struct {
		name string
		body string
	}{
		{name: "push_tokenが空", body: `{"push_token":"","platform":"ios","notify_time":"22:00","enabled":true}`},
		{name: "不正なplatform", body: `{"push_token":"t","platform":"web","notify_time":"22:00","enabled":true}`},
		{name: "不正なnotify_time", body: `{"push_token":"t","platform":"android","notify_time":"9pm","enabled":true}`},
		{name: "範囲外のnotify_time", body: `{"push_token":"t","platform":"android","notify_time":"25:00","enabled":true}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := do(t, env.r, http.MethodPut, "/api/v1/devices", token, tt.body)
			require.Equal(t, http.StatusBadRequest, res.StatusCode)
			res.Body.Close()
		})
	}
}

func TestDevicesRegisterRequiresAuth(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", "",
		`{"push_token":"token-abc","platform":"ios","notify_time":"22:00","enabled":true}`)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()
}

func TestDevicesAreScopedPerUser(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	tokenA := signupToken(t, env.r, "devices-a@example.com")
	tokenB := signupToken(t, env.r, "devices-b@example.com")

	// 両ユーザーが同一の push_token 文字列を登録しても、別行として扱われる
	// （ユニークキーは (user_id, push_token)）。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", tokenA,
		`{"push_token":"shared-token","platform":"ios","notify_time":"22:00","enabled":true}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var a deviceResp
	decode(t, res, &a)

	res = do(t, env.r, http.MethodPut, "/api/v1/devices", tokenB,
		`{"push_token":"shared-token","platform":"android","notify_time":"21:00","enabled":true}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var b deviceResp
	decode(t, res, &b)

	assert.NotEqual(t, a.ID, b.ID)
	assert.Equal(t, "android", b.Platform)
}
