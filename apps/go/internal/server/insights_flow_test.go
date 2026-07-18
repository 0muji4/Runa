package server_test

import (
	"net/http"
	"testing"
)

// insightsResponse mirrors the JSON of GET /api/v1/insights.
type insightsResponse struct {
	Period           string `json:"period"`
	Start            string `json:"start"`
	DaysJournaled    int    `json:"days_journaled"`
	EntryCount       int    `json:"entry_count"`
	UnmoodedCount    int    `json:"unmooded_count"`
	MoodDistribution []struct {
		Mood  string `json:"mood"`
		Count int    `json:"count"`
	} `json:"mood_distribution"`
}

func (i insightsResponse) mood(name string) int {
	for _, m := range i.MoodDistribution {
		if m.Mood == name {
			return m.Count
		}
	}
	return 0
}

// createMood posts a diary entry with an explicit created_at (UTC RFC3339) and mood.
func createMood(t *testing.T, r http.Handler, token, clientID, createdAt, mood string) {
	t.Helper()
	res := do(t, r, http.MethodPost, "/api/v1/diary", token,
		`{"body_text":"月あかり","client_id":"`+clientID+`","created_at":"`+createdAt+`","mood":"`+mood+`"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %s status = %d, want 201", createdAt, res.StatusCode)
	}
}

// TestInsightsMonthlyAggregation checks day/entry/mood counts and that mood-less
// entries are counted but excluded from the distribution (未選択).
func TestInsightsMonthlyAggregation(t *testing.T) {
	r := newDiaryFlowRouter()
	token := signupToken(t, r, "insights@example.com")

	createMood(t, r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-07-10T09:00:00Z", "calm")
	createMood(t, r, token, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "2026-07-10T20:00:00Z", "calm")
	createMood(t, r, token, "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "2026-07-11T09:00:00Z", "gentle")
	createOn(t, r, token, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "2026-07-12T09:00:00Z")            // no mood
	createMood(t, r, token, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", "2026-06-25T09:00:00Z", "heavy") // out of month

	var got insightsResponse
	res := do(t, r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01", token, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("insights status = %d, want 200", res.StatusCode)
	}
	decode(t, res, &got)

	if got.Period != "monthly" || got.Start != "2026-07-01" {
		t.Fatalf("echo = %s/%s, want monthly/2026-07-01", got.Period, got.Start)
	}
	if got.DaysJournaled != 3 {
		t.Fatalf("days_journaled = %d, want 3 (10, 11, 12)", got.DaysJournaled)
	}
	if got.EntryCount != 4 {
		t.Fatalf("entry_count = %d, want 4 (mood-less included, June excluded)", got.EntryCount)
	}
	if got.UnmoodedCount != 1 {
		t.Fatalf("unmooded_count = %d, want 1 (the 12th)", got.UnmoodedCount)
	}
	if c := got.mood("calm"); c != 2 {
		t.Fatalf("calm = %d, want 2", c)
	}
	if g := got.mood("gentle"); g != 1 {
		t.Fatalf("gentle = %d, want 1", g)
	}
	if h := got.mood("heavy"); h != 0 {
		t.Fatalf("heavy = %d, want 0 (June entry excluded)", h)
	}
}

// TestInsightsGroupsByLocalDate verifies the period window honours the requested tz,
// the boundary case that makes the server's counts match the client's local grouping.
func TestInsightsGroupsByLocalDate(t *testing.T) {
	r := newDiaryFlowRouter()
	token := signupToken(t, r, "insights-tz@example.com")

	// 2026-06-30 20:00 UTC = 2026-07-01 05:00 JST → outside July in UTC, inside in Tokyo.
	createMood(t, r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-06-30T20:00:00Z", "calm")

	var utc insightsResponse
	res := do(t, r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01", token, "")
	decode(t, res, &utc)
	if utc.EntryCount != 0 {
		t.Fatalf("UTC entry_count = %d, want 0 (still June)", utc.EntryCount)
	}

	var jst insightsResponse
	res = do(t, r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01&tz=Asia/Tokyo", token, "")
	decode(t, res, &jst)
	if jst.EntryCount != 1 || jst.DaysJournaled != 1 {
		t.Fatalf("JST entry_count/days = %d/%d, want 1/1", jst.EntryCount, jst.DaysJournaled)
	}
}

// TestInsightsWeeklyWindowIsSevenDays checks the weekly window is [start, start+7).
func TestInsightsWeeklyWindowIsSevenDays(t *testing.T) {
	r := newDiaryFlowRouter()
	token := signupToken(t, r, "insights-weekly@example.com")

	createMood(t, r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-07-12T09:00:00Z", "calm")
	createMood(t, r, token, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "2026-07-15T09:00:00Z", "gentle")
	createMood(t, r, token, "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "2026-07-19T09:00:00Z", "calm") // next week

	var got insightsResponse
	res := do(t, r, http.MethodGet, "/api/v1/insights?period=weekly&start=2026-07-12", token, "")
	decode(t, res, &got)
	if got.EntryCount != 2 || got.DaysJournaled != 2 {
		t.Fatalf("weekly entry_count/days = %d/%d, want 2/2 (07-19 is the next week)", got.EntryCount, got.DaysJournaled)
	}
}

// TestInsightsIsScopedAndValidated confirms per-user scoping and param validation.
func TestInsightsIsScopedAndValidated(t *testing.T) {
	r := newDiaryFlowRouter()
	owner := signupToken(t, r, "insights-owner@example.com")
	createMood(t, r, owner, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "2026-07-15T03:00:00Z", "calm")

	// A stranger sees none of the owner's entries.
	other := signupToken(t, r, "insights-stranger@example.com")
	var empty insightsResponse
	res := do(t, r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01", other, "")
	decode(t, res, &empty)
	if empty.EntryCount != 0 {
		t.Fatalf("stranger entry_count = %d, want 0", empty.EntryCount)
	}

	// Validation: missing/bad period, missing/bad start, and a bad tz are all 400.
	for _, path := range []string{
		"/api/v1/insights?start=2026-07-01",                          // missing period
		"/api/v1/insights?period=daily&start=2026-07-01",             // bad period
		"/api/v1/insights?period=monthly",                            // missing start
		"/api/v1/insights?period=monthly&start=2026-13-01",           // bad start
		"/api/v1/insights?period=monthly&start=2026-07-01&tz=Mars/X", // bad tz
	} {
		res := do(t, r, http.MethodGet, path, owner, "")
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want 400", path, res.StatusCode)
		}
	}
}
