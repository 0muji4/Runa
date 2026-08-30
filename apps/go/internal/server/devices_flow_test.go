package server_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDevicesRegisterFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "devices@example.com")

	// 初回登録は200で作成される。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", token,
		`{"push_token":"token-abc","platform":"ios","notify_time":"22:00","enabled":true}`)
	checkStatus(t, res, http.StatusOK)
	var created deviceResp
	decode(t, res, &created)
	if created.ID == "" {
		t.Error("registered device id is empty, want a generated id")
	}
	if diff := cmp.Diff(
		deviceResp{PushToken: "token-abc", Platform: "ios", NotifyTime: "22:00", Enabled: true},
		created,
		cmpopts.IgnoreFields(deviceResp{}, "ID", "CreatedAt", "UpdatedAt"),
	); diff != "" {
		t.Errorf("registered device mismatch (-want +got):\n%s", diff)
	}

	// 同一トークンの再PUTは冪等upsert：同じidのまま設定が更新される。
	res = do(t, env.r, http.MethodPut, "/api/v1/devices", token,
		`{"push_token":"token-abc","platform":"ios","notify_time":"23:00","enabled":false}`)
	checkStatus(t, res, http.StatusOK)
	var updated deviceResp
	decode(t, res, &updated)
	// Same push token ⇒ the same row is updated, not a second device.
	if updated.ID != created.ID {
		t.Errorf("re-registering the same push token created id %q, want the existing %q",
			updated.ID, created.ID)
	}
	if got, want := updated.NotifyTime, "23:00"; got != want {
		t.Errorf("updated notify_time = %q, want %q", got, want)
	}
	if updated.Enabled {
		t.Error("updated enabled = true, want false")
	}
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
			checkStatus(t, res, http.StatusBadRequest)
			res.Body.Close()
		})
	}
}

func TestDevicesRegisterRequiresAuth(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", "",
		`{"push_token":"token-abc","platform":"ios","notify_time":"22:00","enabled":true}`)
	checkStatus(t, res, http.StatusUnauthorized)
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
	checkStatus(t, res, http.StatusOK)
	var a deviceResp
	decode(t, res, &a)

	res = do(t, env.r, http.MethodPut, "/api/v1/devices", tokenB,
		`{"push_token":"shared-token","platform":"android","notify_time":"21:00","enabled":true}`)
	checkStatus(t, res, http.StatusOK)
	var b deviceResp
	decode(t, res, &b)

	if b.ID == a.ID {
		t.Errorf("a second push token reused device id %q, want a distinct device", a.ID)
	}
	if got, want := b.Platform, "android"; got != want {
		t.Errorf("second device platform = %q, want %q", got, want)
	}
}
