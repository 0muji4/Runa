package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const (
	installA = "11111111-1111-4111-8111-000000000001"
	installB = "11111111-1111-4111-8111-000000000002"
)

// deviceBody is a valid registration; callers override fields by string replacement.
func deviceBody(install, pushToken, platform, notifyTime string, enabled bool) string {
	on := "false"
	if enabled {
		on = "true"
	}
	return `{"install_id":"` + install + `","push_token":"` + pushToken + `","platform":"` + platform +
		`","notify_time":"` + notifyTime + `","time_zone":"Asia/Tokyo","enabled":` + on + `}`
}

func TestDevicesRegisterFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "devices@example.com")

	// 初回登録は200で作成される。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", token, deviceBody(installA, "token-abc", "ios", "22:00", true))
	checkStatus(t, res, http.StatusOK)
	var created deviceResp
	decode(t, res, &created)
	if created.ID == "" {
		t.Error("registered device id is empty, want a generated id")
	}
	if diff := cmp.Diff(
		deviceResp{InstallID: installA, PushToken: "token-abc", Platform: "ios", NotifyTime: "22:00", TimeZone: "Asia/Tokyo", Enabled: true},
		created,
		cmpopts.IgnoreFields(deviceResp{}, "ID", "CreatedAt", "UpdatedAt"),
	); diff != "" {
		t.Errorf("registered device mismatch (-want +got):\n%s", diff)
	}

	// 同一 install の再PUTは冪等upsert：トークンが変わっても同じidのまま設定が更新される。
	res = do(t, env.r, http.MethodPut, "/api/v1/devices", token, deviceBody(installA, "token-rotated", "ios", "23:00", false))
	checkStatus(t, res, http.StatusOK)
	var updated deviceResp
	decode(t, res, &updated)
	if updated.ID != created.ID {
		t.Errorf("re-registering the same install created id %q, want the existing %q", updated.ID, created.ID)
	}
	if updated.PushToken != "token-rotated" || updated.NotifyTime != "23:00" || updated.Enabled {
		t.Errorf("updated device = %+v, want token-rotated / 23:00 / disabled", updated)
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
		{name: "install_idがUUIDでない", body: `{"install_id":"phone","push_token":"t","platform":"ios","notify_time":"22:00","time_zone":"Asia/Tokyo","enabled":true}`},
		{name: "push_tokenが空", body: deviceBody(installA, "", "ios", "22:00", true)},
		{name: "不正なplatform", body: deviceBody(installA, "t", "web", "22:00", true)},
		{name: "不正なnotify_time", body: deviceBody(installA, "t", "android", "9pm", true)},
		{name: "範囲外のnotify_time", body: deviceBody(installA, "t", "android", "25:00", true)},
		{name: "未知のtime_zone", body: `{"install_id":"` + installA + `","push_token":"t","platform":"ios","notify_time":"22:00","time_zone":"Moon/Tranquility","enabled":true}`},
		{name: "time_zoneのLocal", body: `{"install_id":"` + installA + `","push_token":"t","platform":"ios","notify_time":"22:00","time_zone":"Local","enabled":true}`},
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
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", "", deviceBody(installA, "token-abc", "ios", "22:00", true))
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()
}

func TestDevicesTokenFollowsLatestUser(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	tokenA := signupToken(t, env.r, "devices-a@example.com")
	tokenB := signupToken(t, env.r, "devices-b@example.com")

	// 同じ端末（同じ push_token）で別ユーザーがログインすると、トークンは後から登録した
	// ユーザーに移り、前ユーザーの行は消える（前ユーザーのリマインドが届かないように）。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", tokenA, deviceBody(installA, "shared-token", "ios", "22:00", true))
	checkStatus(t, res, http.StatusOK)
	var a deviceResp
	decode(t, res, &a)
	aTask := env.scheduler.Tasks()[0]

	res = do(t, env.r, http.MethodPut, "/api/v1/devices", tokenB, deviceBody(installB, "shared-token", "ios", "21:00", true))
	checkStatus(t, res, http.StatusOK)
	var b deviceResp
	decode(t, res, &b)

	if b.ID == a.ID {
		t.Errorf("second user reused device id %q, want a new row", a.ID)
	}
	if _, err := env.devices.GetDevice(t.Context(), a.ID); err == nil {
		t.Error("previous user's device row still exists, want it deleted")
	}
	// 前ユーザーの予約は残ってよい（取り消しはベストエフォート）が、届いても何もしない。
	env.clock.Set(fireAt.Add(time.Minute))
	if got := callback(t, env, aTask.Body); got != "stale" {
		t.Errorf("previous user's task outcome = %q, want stale", got)
	}
	if len(env.ios.Sent()) != 0 {
		t.Error("the previous user's task produced a send")
	}
}
