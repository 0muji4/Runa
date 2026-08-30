package server_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInsightsMonthlyAggregation(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "insights@example.com")

	createMood(t, env.r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-07-10T09:00:00Z", "calm")
	createMood(t, env.r, token, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "2026-07-10T20:00:00Z", "calm")
	createMood(t, env.r, token, "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "2026-07-11T09:00:00Z", "gentle")
	createOn(t, env.r, token, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "2026-07-12T09:00:00Z")
	createMood(t, env.r, token, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee", "2026-06-25T09:00:00Z", "heavy")

	res := do(t, env.r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01", token, "")
	checkStatus(t, res, http.StatusOK)
	var got insightsResponse
	decode(t, res, &got)

	want := struct {
		period, start                            string
		daysJournaled, entryCount, unmoodedCount int
	}{"monthly", "2026-07-01", 3, 4, 1}
	gotSummary := struct {
		period, start                            string
		daysJournaled, entryCount, unmoodedCount int
	}{got.Period, got.Start, got.DaysJournaled, got.EntryCount, got.UnmoodedCount}
	if gotSummary != want {
		t.Errorf("GET /api/v1/insights summary = %+v, want %+v", gotSummary, want)
	}
	wantMoods := map[string]int{
		"calm":    2,
		"gentle":  1,
		"tired":   0,
		"hopeful": 0,
		"heavy":   0,
	}
	if diff := cmp.Diff(wantMoods, got.moods()); diff != "" {
		t.Errorf("mood distribution mismatch (-want +got):\n%s", diff)
	}
}

func TestInsightsGroupsByLocalDate(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "insights-tz@example.com")

	// 2026-06-30 20:00 UTC = 2026-07-01 05:00 JST。UTCでは6月、Tokyoでは7月。
	createMood(t, env.r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-06-30T20:00:00Z", "calm")

	res := do(t, env.r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01", token, "")
	var utc insightsResponse
	decode(t, res, &utc)
	if utc.EntryCount != 0 {
		t.Errorf("entry_count in UTC = %d, want 0 (the entry falls in June there)", utc.EntryCount)
	}

	res = do(t, env.r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01&tz=Asia/Tokyo", token, "")
	var jst insightsResponse
	decode(t, res, &jst)
	if jst.EntryCount != 1 || jst.DaysJournaled != 1 {
		t.Errorf("in Asia/Tokyo: entry_count = %d, days_journaled = %d, want 1 and 1",
			jst.EntryCount, jst.DaysJournaled)
	}
}

func TestInsightsWeeklyWindowIsSevenDays(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "insights-weekly@example.com")

	createMood(t, env.r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-07-12T09:00:00Z", "calm")
	createMood(t, env.r, token, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", "2026-07-15T09:00:00Z", "gentle")
	createMood(t, env.r, token, "cccccccc-cccc-4ccc-8ccc-cccccccccccc", "2026-07-19T09:00:00Z", "calm")

	res := do(t, env.r, http.MethodGet, "/api/v1/insights?period=weekly&start=2026-07-12", token, "")
	var got insightsResponse
	decode(t, res, &got)
	// The weekly window is [start, start+7d): 07-12 and 07-15 are in, 07-19 is out.
	if got.EntryCount != 2 || got.DaysJournaled != 2 {
		t.Errorf("weekly window: entry_count = %d, days_journaled = %d, want 2 and 2",
			got.EntryCount, got.DaysJournaled)
	}
}

func TestInsightsIsScopedAndValidated(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	owner := signupToken(t, env.r, "insights-owner@example.com")
	createMood(t, env.r, owner, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "2026-07-15T03:00:00Z", "calm")

	other := signupToken(t, env.r, "insights-stranger@example.com")
	res := do(t, env.r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01", other, "")
	var empty insightsResponse
	decode(t, res, &empty)
	if empty.EntryCount != 0 {
		t.Errorf("a stranger's insights entry_count = %d, want 0 (entries are owner-scoped)",
			empty.EntryCount)
	}

	tests := []struct {
		name, path string
	}{
		{
			name: "periodが欠落",
			path: "/api/v1/insights?start=2026-07-01",
		},
		{
			name: "不正なperiod",
			path: "/api/v1/insights?period=daily&start=2026-07-01",
		},
		{
			name: "startが欠落",
			path: "/api/v1/insights?period=monthly",
		},
		{
			name: "不正なstart",
			path: "/api/v1/insights?period=monthly&start=2026-13-01",
		},
		{
			name: "不正なtz",
			path: "/api/v1/insights?period=monthly&start=2026-07-01&tz=Mars/X",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := do(t, env.r, http.MethodGet, tt.path, owner, "")
			checkStatus(t, res, http.StatusBadRequest)
			res.Body.Close()
		})
	}
}
