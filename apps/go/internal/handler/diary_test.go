package handler

import (
	"fmt"
	"net/http"
	"testing"
)

const (
	knownClientID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	unknownID     = "99999999-9999-4999-8999-999999999999"
)

func TestDiary_Create(t *testing.T) {
	t.Parallel()

	validCreate := fmt.Sprintf(`{"body_text":"月を見た","client_id":%q}`, knownClientID)

	tests := []struct {
		name        string
		setup       func(t *testing.T, r http.Handler)
		body        string
		wantStatus  int
		wantCode    ErrorCode
		wantDetails int
	}{
		{
			name:        "新規エントリを作成する",
			setup:       nil,
			body:        validCreate,
			wantStatus:  http.StatusCreated,
			wantCode:    "",
			wantDetails: -1,
		},
		{
			name:        "同一client_idはupsertで200",
			setup:       func(t *testing.T, r http.Handler) { createDiaryEntry(t, r, knownClientID, "月を見た") },
			body:        validCreate,
			wantStatus:  http.StatusOK,
			wantCode:    "",
			wantDetails: -1,
		},
		{
			name:        "空の本文と非UUIDのclient_idは検証エラー",
			setup:       nil,
			body:        `{"body_text":"  ","client_id":"nope"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantDetails: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newDiaryRouter()
			if tt.setup != nil {
				tt.setup(t, r)
			}

			res := doDiary(t, r, http.MethodPost, "/diary", tt.body)
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("POST /diary %s = %d, want %d", tt.body, res.StatusCode, tt.wantStatus)
			}
			checkErrorEnvelope(t, res, tt.wantCode, tt.wantDetails)
		})
	}
}

func TestDiary_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       func(t *testing.T, r http.Handler) string
		wantStatus int
		wantCode   ErrorCode
		wantBody   string
	}{
		{
			name: "既存エントリは200",
			path: func(t *testing.T, r http.Handler) string {
				return "/diary/" + createDiaryEntry(t, r, knownClientID, "月を見た").ID
			},
			wantStatus: http.StatusOK,
			wantCode:   "",
			wantBody:   "月を見た",
		},
		{
			name:       "未知のidは404",
			path:       func(t *testing.T, r http.Handler) string { return "/diary/" + unknownID },
			wantStatus: http.StatusNotFound,
			wantCode:   CodeNotFound,
			wantBody:   "",
		},
		{
			name:       "不正な形式のidは404",
			path:       func(t *testing.T, r http.Handler) string { return "/diary/not-a-uuid" },
			wantStatus: http.StatusNotFound,
			wantCode:   CodeNotFound,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newDiaryRouter()
			res := doDiary(t, r, http.MethodGet, tt.path(t, r), "")
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("GET diary = %d, want %d", res.StatusCode, tt.wantStatus)
			}
			checkErrorEnvelope(t, res, tt.wantCode, -1)
			if tt.wantBody != "" {
				if got := decodeJSON[diaryEntryResponse](t, res).BodyText; got != tt.wantBody {
					t.Errorf("GET diary body_text = %q, want %q", got, tt.wantBody)
				}
			}
		})
	}
}

func TestDiary_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       func(t *testing.T, r http.Handler) string
		body       string
		wantStatus int
		wantCode   ErrorCode
		wantBody   string
	}{
		{
			name:       "body_text欠落は拒否する",
			path:       func(t *testing.T, r http.Handler) string { return "/diary/" + unknownID },
			body:       `{"mood":"calm"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   CodeValidation,
			wantBody:   "",
		},
		{
			name:       "空白のみのbody_textは拒否する",
			path:       func(t *testing.T, r http.Handler) string { return "/diary/" + unknownID },
			body:       `{"body_text":"   "}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   CodeValidation,
			wantBody:   "",
		},
		{
			name: "正常な更新は200",
			path: func(t *testing.T, r http.Handler) string {
				return "/diary/" + createDiaryEntry(t, r, knownClientID, "月を見た").ID
			},
			body:       `{"body_text":"更新後","mood":"calm"}`,
			wantStatus: http.StatusOK,
			wantCode:   "",
			wantBody:   "更新後",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newDiaryRouter()
			res := doDiary(t, r, http.MethodPatch, tt.path(t, r), tt.body)
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("PATCH diary = %d, want %d", res.StatusCode, tt.wantStatus)
			}
			checkErrorEnvelope(t, res, tt.wantCode, -1)
			if tt.wantBody != "" {
				if got := decodeJSON[diaryEntryResponse](t, res).BodyText; got != tt.wantBody {
					t.Errorf("PATCH diary body_text = %q, want %q", got, tt.wantBody)
				}
			}
		})
	}
}

func TestDiary_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       func(t *testing.T, r http.Handler) string
		wantStatus int
		wantCode   ErrorCode
	}{
		{
			name: "自分のエントリを削除する",
			path: func(t *testing.T, r http.Handler) string {
				return "/diary/" + createDiaryEntry(t, r, knownClientID, "月を見た").ID
			},
			wantStatus: http.StatusNoContent,
			wantCode:   "",
		},
		{
			name: "削除は冪等で繰り返し204",
			path: func(t *testing.T, r http.Handler) string {
				id := createDiaryEntry(t, r, knownClientID, "月を見た").ID
				first := doDiary(t, r, http.MethodDelete, "/diary/"+id, "")
				first.Body.Close()
				if first.StatusCode != http.StatusNoContent {
					t.Fatalf("first DELETE /diary/%s = %d, want %d",
						id, first.StatusCode, http.StatusNoContent)
				}
				return "/diary/" + id
			},
			wantStatus: http.StatusNoContent,
			wantCode:   "",
		},
		{
			name:       "未知のidは404",
			path:       func(t *testing.T, r http.Handler) string { return "/diary/" + unknownID },
			wantStatus: http.StatusNotFound,
			wantCode:   CodeNotFound,
		},
		{
			name:       "不正な形式のidは404",
			path:       func(t *testing.T, r http.Handler) string { return "/diary/not-a-uuid" },
			wantStatus: http.StatusNotFound,
			wantCode:   CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newDiaryRouter()
			res := doDiary(t, r, http.MethodDelete, tt.path(t, r), "")
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("DELETE diary = %d, want %d", res.StatusCode, tt.wantStatus)
			}
			checkErrorEnvelope(t, res, tt.wantCode, -1)
		})
	}
}

func TestDiary_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(t *testing.T, r http.Handler)
		wantEntries int
	}{
		{
			name:        "空ストアはエントリなし",
			setup:       nil,
			wantEntries: 0,
		},
		{
			name:        "1件のみのストアは1件返す",
			setup:       func(t *testing.T, r http.Handler) { createDiaryEntry(t, r, knownClientID, "一件目") },
			wantEntries: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newDiaryRouter()
			if tt.setup != nil {
				tt.setup(t, r)
			}

			res := doDiary(t, r, http.MethodGet, "/diary", "")
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				t.Fatalf("GET /diary = %d, want %d", res.StatusCode, http.StatusOK)
			}
			got := decodeJSON[diaryListResponse](t, res)
			if len(got.Entries) != tt.wantEntries {
				t.Errorf("GET /diary returned %d entries, want %d", len(got.Entries), tt.wantEntries)
			}
			if got.NextCursor != nil {
				t.Errorf("GET /diary next_cursor = %q, want nil", *got.NextCursor)
			}
		})
	}
}

func TestDiary_Sync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(t *testing.T, r http.Handler)
		query       string
		wantStatus  int
		wantCode    ErrorCode
		wantEntries int
	}{
		{
			name:        "不正な形式のsinceは拒否する",
			setup:       nil,
			query:       "?since=not-a-time",
			wantStatus:  http.StatusBadRequest,
			wantCode:    CodeValidation,
			wantEntries: -1,
		},
		{
			name:        "変更エントリとserver_timeを返す",
			setup:       func(t *testing.T, r http.Handler) { createDiaryEntry(t, r, knownClientID, "月を見た") },
			query:       "",
			wantStatus:  http.StatusOK,
			wantCode:    "",
			wantEntries: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newDiaryRouter()
			if tt.setup != nil {
				tt.setup(t, r)
			}

			res := doDiary(t, r, http.MethodGet, "/diary/sync"+tt.query, "")
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("GET /diary/sync%s = %d, want %d", tt.query, res.StatusCode, tt.wantStatus)
			}
			checkErrorEnvelope(t, res, tt.wantCode, -1)
			if tt.wantEntries >= 0 {
				got := decodeJSON[diarySyncResponse](t, res)
				if len(got.Entries) != tt.wantEntries {
					t.Errorf("GET /diary/sync%s returned %d entries, want %d",
						tt.query, len(got.Entries), tt.wantEntries)
				}
				if got.ServerTime == "" {
					t.Errorf("GET /diary/sync%s server_time is empty, want a timestamp", tt.query)
				}
			}
		})
	}
}
