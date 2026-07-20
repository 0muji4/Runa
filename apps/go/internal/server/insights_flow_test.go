package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.Equal(t, http.StatusOK, res.StatusCode)
	var got insightsResponse
	decode(t, res, &got)

	assert.Equal(t, "monthly", got.Period)
	assert.Equal(t, "2026-07-01", got.Start)
	assert.Equal(t, 3, got.DaysJournaled)
	assert.Equal(t, 4, got.EntryCount)
	assert.Equal(t, 1, got.UnmoodedCount)
	assert.Equal(t, map[string]int{
		"calm":    2,
		"gentle":  1,
		"tired":   0,
		"hopeful": 0,
		"heavy":   0,
	}, got.moods())
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
	assert.Equal(t, 0, utc.EntryCount)

	res = do(t, env.r, http.MethodGet, "/api/v1/insights?period=monthly&start=2026-07-01&tz=Asia/Tokyo", token, "")
	var jst insightsResponse
	decode(t, res, &jst)
	assert.Equal(t, 1, jst.EntryCount)
	assert.Equal(t, 1, jst.DaysJournaled)
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
	assert.Equal(t, 2, got.EntryCount)
	assert.Equal(t, 2, got.DaysJournaled)
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
	assert.Equal(t, 0, empty.EntryCount)

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
			require.Equal(t, http.StatusBadRequest, res.StatusCode)
			res.Body.Close()
		})
	}
}
