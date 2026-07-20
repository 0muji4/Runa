package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiaryFlow(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "diary@example.com")
	const cid = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

	res := do(t, env.r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月を見上げた","mood":"calm","client_id":"`+cid+`","created_at":"2026-07-01T12:00:00Z"}`)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	var created diaryEntry
	decode(t, res, &created)
	assert.Equal(t, cid, created.ClientID)
	require.NotNil(t, created.Mood)
	assert.Equal(t, "calm", *created.Mood)

	// 同じclient_idの再POSTは200で同じidを返す（冪等）。
	res = do(t, env.r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月を見上げた（推敲）","client_id":"`+cid+`","created_at":"2026-07-01T12:00:00Z"}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var repeated diaryEntry
	decode(t, res, &repeated)
	assert.Equal(t, created.ID, repeated.ID)

	res = do(t, env.r, http.MethodGet, "/api/v1/diary", token, "")
	require.Equal(t, http.StatusOK, res.StatusCode)
	var list diaryList
	decode(t, res, &list)
	assert.Len(t, list.Entries, 1)
	assert.Nil(t, list.NextCursor)

	res = do(t, env.r, http.MethodGet, "/api/v1/diary/"+created.ID, token, "")
	require.Equal(t, http.StatusOK, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/diary/sync", token, "")
	var full diarySync
	decode(t, res, &full)
	require.Len(t, full.Entries, 1)

	res = do(t, env.r, http.MethodPatch, "/api/v1/diary/"+created.ID, token,
		`{"body_text":"翌朝、読み返した","mood":"gentle"}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/diary/sync?since="+full.ServerTime, token, "")
	var delta diarySync
	decode(t, res, &delta)
	require.Len(t, delta.Entries, 1)
	assert.Equal(t, "翌朝、読み返した", delta.Entries[0].BodyText)

	res = do(t, env.r, http.MethodDelete, "/api/v1/diary/"+created.ID, token, "")
	require.Equal(t, http.StatusNoContent, res.StatusCode)
	res.Body.Close()

	res = do(t, env.r, http.MethodGet, "/api/v1/diary", token, "")
	decode(t, res, &list)
	assert.Empty(t, list.Entries)

	// 削除は同期でtombstone（deleted_at付き）として伝播する。
	res = do(t, env.r, http.MethodGet, "/api/v1/diary/sync?since="+delta.ServerTime, token, "")
	var afterDelete diarySync
	decode(t, res, &afterDelete)
	require.Len(t, afterDelete.Entries, 1)
	assert.NotNil(t, afterDelete.Entries[0].DeletedAt)

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
			assert.Equal(t, http.StatusNotFound, res.StatusCode)
			res.Body.Close()
		})
	}
}

func TestDiaryRequiresAuth(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	res := do(t, env.r, http.MethodGet, "/api/v1/diary", "", "")
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	res.Body.Close()
}
