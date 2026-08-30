package server_test

import (
	"net/http"
	"testing"
)

func TestDiaryFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "diary@example.com")
	const cid = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

	res := do(t, env.r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月を見上げた","mood":"calm","client_id":"`+cid+`","created_at":"2026-07-01T12:00:00Z"}`)
	checkStatus(t, res, http.StatusCreated)
	var created diaryEntry
	decode(t, res, &created)
	if created.ClientID != cid {
		t.Errorf("created client_id = %q, want %q", created.ClientID, cid)
	}
	if created.Mood == nil {
		t.Fatal("created mood = nil, want \"calm\"")
	}
	if *created.Mood != "calm" {
		t.Errorf("created mood = %q, want %q", *created.Mood, "calm")
	}

	// 同じclient_idの再POSTは200で同じidを返す（冪等）。
	res = do(t, env.r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月を見上げた（推敲）","client_id":"`+cid+`","created_at":"2026-07-01T12:00:00Z"}`)
	checkStatus(t, res, http.StatusOK)
	var repeated diaryEntry
	decode(t, res, &repeated)
	if repeated.ID != created.ID {
		t.Errorf("re-POST with the same client_id returned id %q, want the original %q (upsert)",
			repeated.ID, created.ID)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/diary", token, "")
	checkStatus(t, res, http.StatusOK)
	var list diaryList
	decode(t, res, &list)
	if len(list.Entries) != 1 {
		t.Errorf("GET /api/v1/diary returned %d entries, want 1", len(list.Entries))
	}
	if list.NextCursor != nil {
		t.Errorf("GET /api/v1/diary next_cursor = %q, want nil", *list.NextCursor)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/diary/"+created.ID, token, "")
	checkStatus(t, res, http.StatusOK)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/diary/sync", token, "")
	var full diarySync
	decode(t, res, &full)
	if len(full.Entries) != 1 {
		t.Fatalf("full sync returned %d entries, want 1", len(full.Entries))
	}

	res = do(t, env.r, http.MethodPatch, "/api/v1/diary/"+created.ID, token,
		`{"body_text":"翌朝、読み返した","mood":"gentle"}`)
	checkStatus(t, res, http.StatusOK)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/diary/sync?since="+full.ServerTime, token, "")
	var delta diarySync
	decode(t, res, &delta)
	if len(delta.Entries) != 1 {
		t.Fatalf("delta sync returned %d entries, want 1", len(delta.Entries))
	}
	if got, want := delta.Entries[0].BodyText, "翌朝、読み返した"; got != want {
		t.Errorf("delta sync body_text = %q, want %q", got, want)
	}

	res = do(t, env.r, http.MethodDelete, "/api/v1/diary/"+created.ID, token, "")
	checkStatus(t, res, http.StatusNoContent)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/diary", token, "")
	decode(t, res, &list)
	if len(list.Entries) != 0 {
		t.Errorf("GET /api/v1/diary after delete returned %d entries, want 0", len(list.Entries))
	}

	// 削除は同期でtombstone（deleted_at付き）として伝播する。
	res = do(t, env.r, http.MethodGet, "/api/v1/diary/sync?since="+delta.ServerTime, token, "")
	var afterDelete diarySync
	decode(t, res, &afterDelete)
	if len(afterDelete.Entries) != 1 {
		t.Fatalf("sync after delete returned %d entries, want 1 tombstone", len(afterDelete.Entries))
	}
	if afterDelete.Entries[0].DeletedAt == nil {
		t.Error("sync after delete: deleted_at = nil, want a tombstone timestamp")
	}

	other := signupToken(t, env.r, "other@example.com")
	tests := []struct {
		name, method, path, body string
	}{
		{name: "get", method: http.MethodGet, path: "/api/v1/diary/" + created.ID, body: ""},
		{name: "patch", method: http.MethodPatch, path: "/api/v1/diary/" + created.ID, body: `{"body_text":"改ざん"}`},
		{name: "delete", method: http.MethodDelete, path: "/api/v1/diary/" + created.ID, body: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := do(t, env.r, tt.method, tt.path, other, tt.body)
			checkStatus(t, res, http.StatusNotFound)
			res.Body.Close()
		})
	}
}

func TestDiaryRequiresAuth(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodGet, "/api/v1/diary", "", "")
	checkStatus(t, res, http.StatusUnauthorized)
	res.Body.Close()
}
