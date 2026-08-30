package server_test

import (
	"net/http"
	"testing"
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
		checkStatus(t, res, http.StatusOK)
		var cal calendarResponse
		decode(t, res, &cal)
		if len(cal.Days) != 1 {
			t.Fatalf("calendar (tz=%q) returned %d days, want 1", tz, len(cal.Days))
		}
		if cal.Days[0].Count != 1 {
			t.Fatalf("calendar (tz=%q) day count = %d, want 1", tz, cal.Days[0].Count)
		}
		return cal.Days[0].Date
	}

	if got, want := localDay(t, ""), "2026-07-03"; got != want {
		t.Errorf("calendar day with no tz = %q, want %q (UTC)", got, want)
	}
	if got, want := localDay(t, "Asia/Tokyo"), "2026-07-04"; got != want {
		t.Errorf("calendar day with tz=Asia/Tokyo = %q, want %q", got, want)
	}
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
	if len(empty.Days) != 0 {
		t.Errorf("a stranger's calendar returned %d days, want 0 (entries are owner-scoped)",
			len(empty.Days))
	}

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
			checkStatus(t, res, http.StatusBadRequest)
			res.Body.Close()
		})
	}
}
