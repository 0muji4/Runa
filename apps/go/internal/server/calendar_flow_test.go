package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalendarGroupsByLocalDate(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	token := signupToken(t, env.r, "calendar@example.com")

	// 2026-07-03 22:00 UTC は Tokyo では 2026-07-04 07:00。
	createOn(t, env.r, token, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "2026-07-03T22:00:00Z")

	localDay := func(t *testing.T, tz string) string {
		t.Helper()
		path := "/api/v1/diary/calendar?year=2026&month=7"
		if tz != "" {
			path += "&tz=" + tz
		}
		res := do(t, env.r, http.MethodGet, path, token, "")
		require.Equal(t, http.StatusOK, res.StatusCode)
		var cal calendarResponse
		decode(t, res, &cal)
		require.Len(t, cal.Days, 1)
		require.Equal(t, 1, cal.Days[0].Count)
		return cal.Days[0].Date
	}

	assert.Equal(t, "2026-07-03", localDay(t, ""))
	assert.Equal(t, "2026-07-04", localDay(t, "Asia/Tokyo"))
}

func TestCalendarIsScopedAndValidated(t *testing.T) {
	t.Parallel()

	env := newRouter(t)
	owner := signupToken(t, env.r, "owner@example.com")
	createOn(t, env.r, owner, "dddddddd-dddd-4ddd-8ddd-dddddddddddd", "2026-07-15T03:00:00Z")

	other := signupToken(t, env.r, "stranger@example.com")
	res := do(t, env.r, http.MethodGet, "/api/v1/diary/calendar?year=2026&month=7", other, "")
	var empty calendarResponse
	decode(t, res, &empty)
	assert.Empty(t, empty.Days)

	tests := []struct {
		name, path string
	}{
		{
			name: "yearが欠落",
			path: "/api/v1/diary/calendar?month=7",
		},
		{
			name: "monthが範囲外",
			path: "/api/v1/diary/calendar?year=2026&month=13",
		},
		{
			name: "未知のtz",
			path: "/api/v1/diary/calendar?year=2026&month=7&tz=Mars/Phobos",
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
