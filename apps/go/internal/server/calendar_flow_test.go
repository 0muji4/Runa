package server_test

import (
	"net/http"
	"testing"
)

// calendarResponse mirrors the JSON of GET /api/v1/diary/calendar.
type calendarResponse struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Days  []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	} `json:"days"`
}

func (c calendarResponse) count(date string) int {
	for _, d := range c.Days {
		if d.Date == date {
			return d.Count
		}
	}
	return 0
}

// createOn posts a diary entry with an explicit created_at (UTC RFC3339).
func createOn(t *testing.T, r http.Handler, token, clientID, createdAt string) {
	t.Helper()
	res := do(t, r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月あかり","client_id":"`+clientID+`","created_at":"`+createdAt+`"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %s status = %d, want 201", createdAt, res.StatusCode)
	}
}

// TestCalendarGroupsByLocalDate verifies that GET /diary/calendar groups entries by
// the caller's local date under the requested tz — the boundary case that makes the
// server's counts match the client's local-first grouping.
func TestCalendarGroupsByLocalDate(t *testing.T) {
	r := newDiaryFlowRouter()
	token := signupToken(t, r, "calendar@example.com")

	// A: 2026-07-10 23:30 JST  (UTC 14:30) → day 10 in both zones.
	// B: 2026-07-04 07:00 JST  (UTC 2026-07-03 22:00) → day 4 JST, day 3 UTC.
	// C: 2026-07-11 05:00 JST  (UTC 2026-07-10 20:00) → day 11 JST, day 10 UTC.
	createOn(t, r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-07-10T14:30:00Z")
	createOn(t, r, token, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "2026-07-03T22:00:00Z")
	createOn(t, r, token, "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "2026-07-10T20:00:00Z")

	// Tokyo grouping: 04→1, 10→1, 11→1.
	var jst calendarResponse
	res := do(t, r, http.MethodGet, "/api/v1/diary/calendar?year=2026&month=7&tz=Asia/Tokyo", token, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("calendar (JST) status = %d, want 200", res.StatusCode)
	}
	decode(t, res, &jst)
	if jst.Year != 2026 || jst.Month != 7 {
		t.Fatalf("calendar echo = %d/%d, want 2026/7", jst.Year, jst.Month)
	}
	if got := jst.count("2026-07-04"); got != 1 {
		t.Fatalf("JST 2026-07-04 = %d, want 1", got)
	}
	if got := jst.count("2026-07-10"); got != 1 {
		t.Fatalf("JST 2026-07-10 = %d, want 1", got)
	}
	if got := jst.count("2026-07-11"); got != 1 {
		t.Fatalf("JST 2026-07-11 = %d, want 1", got)
	}
	if got := jst.count("2026-07-03"); got != 0 {
		t.Fatalf("JST 2026-07-03 = %d, want 0 (that instant is the 4th in Tokyo)", got)
	}

	// UTC grouping: 03→1, 10→2 (A and C collapse onto the 10th).
	var utc calendarResponse
	res = do(t, r, http.MethodGet, "/api/v1/diary/calendar?year=2026&month=7", token, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("calendar (UTC) status = %d, want 200", res.StatusCode)
	}
	decode(t, res, &utc)
	if got := utc.count("2026-07-03"); got != 1 {
		t.Fatalf("UTC 2026-07-03 = %d, want 1", got)
	}
	if got := utc.count("2026-07-10"); got != 2 {
		t.Fatalf("UTC 2026-07-10 = %d, want 2", got)
	}
	if got := utc.count("2026-07-04"); got != 0 {
		t.Fatalf("UTC 2026-07-04 = %d, want 0", got)
	}
}

// TestCalendarIsScopedAndValidated confirms entries are per-user and that the query
// params are validated.
func TestCalendarIsScopedAndValidated(t *testing.T) {
	r := newDiaryFlowRouter()
	owner := signupToken(t, r, "owner@example.com")
	createOn(t, r, owner, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "2026-07-15T03:00:00Z")

	// Another user sees none of the owner's entries.
	other := signupToken(t, r, "stranger@example.com")
	var empty calendarResponse
	res := do(t, r, http.MethodGet, "/api/v1/diary/calendar?year=2026&month=7", other, "")
	decode(t, res, &empty)
	if len(empty.Days) != 0 {
		t.Fatalf("stranger calendar = %+v, want no days", empty.Days)
	}

	// Validation: missing year, out-of-range month, and a bad tz are all 400.
	for _, path := range []string{
		"/api/v1/diary/calendar?month=7",
		"/api/v1/diary/calendar?year=2026&month=13",
		"/api/v1/diary/calendar?year=2026&month=7&tz=Mars/Phobos",
	} {
		res := do(t, r, http.MethodGet, path, owner, "")
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want 400", path, res.StatusCode)
		}
	}
}
