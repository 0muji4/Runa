package server_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/0muji4/Runa/apps/go/internal/push"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/schedule"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

var (
	// The router clock starts at 21:00 JST on 2026-09-20; a 22:00 reminder is due at 13:00Z.
	fireAt     = time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC)
	nextFireAt = fireAt.Add(24 * time.Hour)
)

type reminderOutcome struct {
	Outcome string `json:"outcome"`
}

func registerDevice(t *testing.T, env *testEnv, bearer, body string) (deviceResp, schedule.Task) {
	t.Helper()
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", bearer, body)
	checkStatus(t, res, http.StatusOK)
	var d deviceResp
	decode(t, res, &d)
	tasks := env.scheduler.Tasks()
	if len(tasks) != 1 {
		t.Fatalf("scheduled tasks after registration = %d, want 1", len(tasks))
	}
	return d, tasks[0]
}

func callback(t *testing.T, env *testEnv, body []byte) string {
	t.Helper()
	res := doCallback(t, env.r, callbackToken, body)
	checkStatus(t, res, http.StatusOK)
	var out reminderOutcome
	decode(t, res, &out)
	return out.Outcome
}

func TestPushRegistrationSchedulesNextReminder(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-schedule@example.com")
	d, task := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	if !task.RunAt.Equal(fireAt) {
		t.Errorf("task RunAt = %s, want %s (22:00 JST tonight)", task.RunAt.UTC(), fireAt)
	}
	if task.Path != service.ReminderCallbackPath {
		t.Errorf("task Path = %q, want %q", task.Path, service.ReminderCallbackPath)
	}
	if !strings.HasPrefix(task.Name, "r-"+d.ID+"-2026-09-20-") {
		t.Errorf("task Name = %q, want the device id and local date in it", task.Name)
	}
	var body service.ReminderTask
	if err := json.Unmarshal(task.Body, &body); err != nil {
		t.Fatalf("task body is not a ReminderTask: %v", err)
	}
	want := service.ReminderTask{DeviceID: d.ID, LocalDate: "2026-09-20", NotifyTime: "22:00", TimeZone: "Asia/Tokyo"}
	if diff := cmp.Diff(want, body); diff != "" {
		t.Errorf("task body mismatch (-want +got):\n%s", diff)
	}

	// 時刻を変えると旧タスクは取り消され、別名の新タスクが予約される。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", token, deviceBody(installA, "apns-token", "ios", "23:00", true))
	checkStatus(t, res, http.StatusOK)
	res.Body.Close()
	tasks := env.scheduler.Tasks()
	if len(tasks) != 1 || tasks[0].Name == task.Name || !tasks[0].RunAt.Equal(fireAt.Add(time.Hour)) {
		t.Errorf("after changing the time: tasks = %+v, want one new task at 23:00 JST", tasks)
	}
	if got := env.scheduler.Cancelled(); len(got) != 1 || got[0] != task.Name {
		t.Errorf("cancelled = %v, want [%s]", got, task.Name)
	}

	res = do(t, env.r, http.MethodPut, "/api/v1/devices", token, deviceBody(installA, "apns-token", "ios", "23:00", false))
	checkStatus(t, res, http.StatusOK)
	res.Body.Close()
	if tasks := env.scheduler.Tasks(); len(tasks) != 0 {
		t.Errorf("tasks after disabling = %d, want 0", len(tasks))
	}
}

func TestPushReminderDeliveredOnceWhenUnwritten(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-deliver@example.com")
	_, task := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	env.clock.Set(fireAt.Add(30 * time.Second))
	env.scheduler.Pop(task.Name)
	if got := callback(t, env, task.Body); got != "sent" {
		t.Fatalf("outcome = %q, want sent", got)
	}

	sent := env.ios.Sent()
	if len(sent) != 1 {
		t.Fatalf("iOS sends = %d, want 1", len(sent))
	}
	n := sent[0].Notification
	if sent[0].Token != "apns-token" || n.Title != service.ReminderTitle || n.Body != service.ReminderBody {
		t.Errorf("sent = %+v, want the reminder copy to apns-token", sent[0])
	}
	wantData := map[string]string{"kind": "diary_reminder", "date": "2026-09-20", "title": service.ReminderTitle, "body": service.ReminderBody}
	if diff := cmp.Diff(wantData, n.Data); diff != "" {
		t.Errorf("data mismatch (-want +got):\n%s", diff)
	}
	if n.CollapseID != "diary_reminder" || !n.Expiry.Equal(fireAt.Add(time.Hour)) {
		t.Errorf("collapse/expiry = (%q, %s), want (diary_reminder, %s)", n.CollapseID, n.Expiry.UTC(), fireAt.Add(time.Hour))
	}
	if len(env.android.Sent()) != 0 {
		t.Error("an iOS device was sent through the Android sender")
	}

	// 翌日分が自己継続で予約される。
	tasks := env.scheduler.Tasks()
	if len(tasks) != 1 || !tasks[0].RunAt.Equal(nextFireAt) {
		t.Fatalf("tasks after delivery = %+v, want tomorrow's at %s", tasks, nextFireAt)
	}

	// 同じ日のコールバックが再送されても二度は送らない。
	if got := callback(t, env, task.Body); got != "already_claimed" {
		t.Errorf("repeat outcome = %q, want already_claimed", got)
	}
	if len(env.ios.Sent()) != 1 {
		t.Errorf("iOS sends after a repeat = %d, want still 1", len(env.ios.Sent()))
	}
}

func TestPushReminderSkippedWhenWrittenToday(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-written@example.com")
	_, task := registerDevice(t, env, token, deviceBody(installA, "fcm-token", "android", "22:00", true))
	createOn(t, env.r, token, "aaaaaaaa-aaaa-4aaa-8aaa-000000000001", "2026-09-20T10:00:00+09:00")

	env.clock.Set(fireAt.Add(time.Minute))
	env.scheduler.Pop(task.Name)
	if got := callback(t, env, task.Body); got != "skipped_written" {
		t.Errorf("outcome = %q, want skipped_written", got)
	}
	if len(env.android.Sent()) != 0 {
		t.Error("a reminder was sent although today's entry exists")
	}
	if tasks := env.scheduler.Tasks(); len(tasks) != 1 || !tasks[0].RunAt.Equal(nextFireAt) {
		t.Errorf("tasks after skip = %+v, want tomorrow's", tasks)
	}
}

func TestPushReminderIgnoresStaleTask(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-stale@example.com")
	_, old := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	// 設定変更後に届いた旧タスクは無視される（新タスクが正）。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", token, deviceBody(installA, "apns-token", "ios", "23:00", true))
	checkStatus(t, res, http.StatusOK)
	res.Body.Close()

	env.clock.Set(fireAt.Add(time.Minute))
	if got := callback(t, env, old.Body); got != "stale" {
		t.Errorf("outcome = %q, want stale", got)
	}
	if len(env.ios.Sent()) != 0 {
		t.Error("a stale task produced a send")
	}
}

func TestPushReminderExpiresWhenLate(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-late@example.com")
	_, task := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	env.clock.Set(fireAt.Add(2 * time.Hour))
	env.scheduler.Pop(task.Name)
	if got := callback(t, env, task.Body); got != "expired" {
		t.Errorf("outcome = %q, want expired", got)
	}
	if len(env.ios.Sent()) != 0 {
		t.Error("a late callback produced a send")
	}
	if tasks := env.scheduler.Tasks(); len(tasks) != 1 {
		t.Errorf("tasks after an expired callback = %d, want tomorrow's chain to continue", len(tasks))
	}
}

func TestPushInvalidTokenIsRecordedUntilReregistration(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-invalid@example.com")
	d, task := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	env.ios.FailWith = errors.New("apns: BadDeviceToken: " + push.ErrInvalidToken.Error())
	env.ios.FailWith = errors.Join(push.ErrInvalidToken, env.ios.FailWith)
	env.clock.Set(fireAt.Add(time.Minute))
	env.scheduler.Pop(task.Name)
	if got := callback(t, env, task.Body); got != "invalid_token" {
		t.Fatalf("outcome = %q, want invalid_token", got)
	}
	row, err := env.devices.GetDevice(t.Context(), d.ID)
	if err != nil || row.TokenInvalidAt == nil {
		t.Fatalf("device after rejection = (%+v, %v), want token_invalid_at set", row, err)
	}
	if tasks := env.scheduler.Tasks(); len(tasks) != 0 {
		t.Errorf("tasks after rejection = %d, want 0 (no chain for a dead token)", len(tasks))
	}

	// 再登録で無効印が消え、予約が再開する。
	env.ios.FailWith = nil
	env.clock.Set(nextFireAt.Add(-time.Hour))
	_, next := registerDevice(t, env, token, deviceBody(installA, "apns-token-2", "ios", "22:00", true))
	env.clock.Set(nextFireAt.Add(time.Minute))
	env.scheduler.Pop(next.Name)
	if got := callback(t, env, next.Body); got != "sent" {
		t.Errorf("outcome after re-registration = %q, want sent", got)
	}
}

func TestPushTransientFailureIsRetriedByScheduler(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-transient@example.com")
	_, task := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	env.ios.FailWith = errors.Join(push.ErrTransient, errors.New("apns: 503"))
	env.clock.Set(fireAt.Add(time.Minute))
	env.scheduler.Pop(task.Name)
	res := doCallback(t, env.r, callbackToken, task.Body)
	checkStatus(t, res, http.StatusServiceUnavailable)
	res.Body.Close()

	// スケジューラの再試行（同じ body）が、復旧後に一度だけ送る。
	env.ios.FailWith = nil
	env.clock.Set(fireAt.Add(5 * time.Minute))
	if got := callback(t, env, task.Body); got != "sent" {
		t.Errorf("outcome on retry = %q, want sent", got)
	}
	if len(env.ios.Sent()) != 1 {
		t.Errorf("iOS sends = %d, want 1", len(env.ios.Sent()))
	}
}

func TestPushLogoutAndAccountDeletionUnregister(t *testing.T) {
	t.Parallel()

	env := newRouter(t)

	// ログアウト本文の install_id で端末行と予約が消える。
	res := do(t, env.r, http.MethodPost, "/api/v1/auth/signup", "",
		`{"email":"push-logout@example.com","password":"password123","display_name":"U"}`)
	checkStatus(t, res, http.StatusCreated)
	var tok tokens
	decode(t, res, &tok)
	_, task := registerDevice(t, env, tok.AccessToken, deviceBody(installA, "apns-token", "ios", "22:00", true))

	res = do(t, env.r, http.MethodPost, "/api/v1/auth/logout", "",
		`{"refresh_token":"`+tok.RefreshToken+`","install_id":"`+installA+`"}`)
	checkStatus(t, res, http.StatusNoContent)
	res.Body.Close()
	if got := env.scheduler.Cancelled(); len(got) != 1 || got[0] != task.Name {
		t.Errorf("cancelled after logout = %v, want [%s]", got, task.Name)
	}
	env.clock.Set(fireAt.Add(time.Minute))
	if got := callback(t, env, task.Body); got != "stale" {
		t.Errorf("outcome after logout = %q, want stale", got)
	}

	other := signupToken(t, env.r, "push-delete@example.com")
	_, task2 := registerDevice(t, env, other, deviceBody(installB, "apns-token-2", "ios", "22:00", true))
	res = do(t, env.r, http.MethodDelete, "/api/v1/me", other, "")
	checkStatus(t, res, http.StatusNoContent)
	res.Body.Close()
	if got := env.scheduler.Cancelled(); len(got) != 2 || got[1] != task2.Name {
		t.Errorf("cancelled after account deletion = %v, want [.., %s]", got, task2.Name)
	}
	if got := callback(t, env, task2.Body); got != "stale" {
		t.Errorf("outcome after account deletion = %q, want stale", got)
	}
	if len(env.ios.Sent()) != 0 {
		t.Error("a removed device was sent a reminder")
	}
}

func TestPushCallbackRequiresToken(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	body := []byte(`{"device_id":"` + installA + `","local_date":"2026-09-20","notify_time":"22:00","time_zone":"Asia/Tokyo"}`)
	for _, token := range []string{"", "wrong"} {
		res := doCallback(t, env.r, token, body)
		checkStatus(t, res, http.StatusForbidden)
		res.Body.Close()
	}

	res := doCallback(t, env.r, callbackToken, []byte(`{"device_id":"x"}`))
	checkStatus(t, res, http.StatusBadRequest)
	res.Body.Close()
}

func TestPushReenableGetsFreshTaskName(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-toggle@example.com")
	_, first := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	// OFF→ON は Cloud Tasks が実行済み/削除済みの名前を拒む窓に入っても予約できるよう、別名になる。
	res := do(t, env.r, http.MethodPut, "/api/v1/devices", token, deviceBody(installA, "apns-token", "ios", "22:00", false))
	checkStatus(t, res, http.StatusOK)
	res.Body.Close()
	_, again := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))
	if again.Name == first.Name {
		t.Errorf("re-enabled task reused name %q, want a fresh one", first.Name)
	}
	if !again.RunAt.Equal(first.RunAt) {
		t.Errorf("re-enabled task RunAt = %s, want %s", again.RunAt.UTC(), first.RunAt.UTC())
	}
}

func TestPushLivePendingClaimAsksForRetry(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-pending@example.com")
	d, task := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	// 送信中に落ちた（pending のまま）直後の再送は 503 で待たせ、5 分過ぎたら送る。
	env.clock.Set(fireAt.Add(time.Minute))
	cfg := service.DefaultPushConfig()
	claim, err := env.devices.ClaimReminder(t.Context(), repository.ClaimReminderParams{
		DeviceID: d.ID, LocalDate: "2026-09-20", Now: env.clock.Now(),
		StalePendingAfter: cfg.StalePendingAfter, MaxAttempts: cfg.MaxAttempts,
	})
	if err != nil || claim != repository.ClaimWon {
		t.Fatalf("seeding a pending claim = (%v, %v), want ClaimWon", claim, err)
	}
	res := doCallback(t, env.r, callbackToken, task.Body)
	checkStatus(t, res, http.StatusServiceUnavailable)
	res.Body.Close()

	env.clock.Set(fireAt.Add(10 * time.Minute))
	if got := callback(t, env, task.Body); got != "sent" {
		t.Errorf("outcome after the pending claim went stale = %q, want sent", got)
	}
}

func TestPushContinuationFailureKeepsExecutingTask(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "push-chain@example.com")
	_, task := registerDevice(t, env, token, deviceBody(installA, "apns-token", "ios", "22:00", true))

	// 翌日分の予約に失敗しても、今夜のタスクは取り消されない（503 で再試行が来る）。
	env.clock.Set(fireAt.Add(time.Minute))
	env.scheduler.Pop(task.Name)
	env.scheduler.FailWith = errors.New("cloudtasks: 503")
	res := doCallback(t, env.r, callbackToken, task.Body)
	checkStatus(t, res, http.StatusServiceUnavailable)
	res.Body.Close()
	if len(env.ios.Sent()) != 1 {
		t.Fatalf("iOS sends = %d, want 1 (the reminder itself went out)", len(env.ios.Sent()))
	}
	if got := env.scheduler.Cancelled(); len(got) != 0 {
		t.Errorf("cancelled = %v, want nothing (the executing task must survive)", got)
	}

	env.scheduler.FailWith = nil
	if got := callback(t, env, task.Body); got != "already_claimed" {
		t.Errorf("retry outcome = %q, want already_claimed", got)
	}
	if len(env.ios.Sent()) != 1 {
		t.Errorf("iOS sends after retry = %d, want still 1", len(env.ios.Sent()))
	}
	if tasks := env.scheduler.Tasks(); len(tasks) != 1 || !tasks[0].RunAt.Equal(nextFireAt) {
		t.Errorf("tasks after retry = %+v, want tomorrow's", tasks)
	}
}
