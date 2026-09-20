package service

import (
	"testing"
	"time"
)

func TestNextFireAt(t *testing.T) {
	t.Parallel()

	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	newYork, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name       string
		now        time.Time
		notifyTime string
		loc        *time.Location
		wantFire   time.Time
		wantDate   string
	}{
		{
			name: "当日の時刻がまだ先なら当日",
			now:  time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC), notifyTime: "22:00", loc: tokyo,
			wantFire: time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC), wantDate: "2026-09-20",
		},
		{
			name: "当日の時刻を過ぎていれば翌日",
			now:  time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC), notifyTime: "22:00", loc: tokyo,
			wantFire: time.Date(2026, 9, 21, 13, 0, 0, 0, time.UTC), wantDate: "2026-09-21",
		},
		{
			name: "現地では翌日でも UTC の日付に引きずられない",
			now:  time.Date(2026, 9, 20, 15, 30, 0, 0, time.UTC), notifyTime: "01:00", loc: tokyo, // 00:30 JST 9/21
			wantFire: time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC), wantDate: "2026-09-21",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, date, err := nextFireAt(tt.now, tt.notifyTime, tt.loc)
			if err != nil {
				t.Fatalf("nextFireAt() error = %v, want nil", err)
			}
			if !got.Equal(tt.wantFire) {
				t.Errorf("nextFireAt() = %s, want %s", got.UTC(), tt.wantFire)
			}
			if date != tt.wantDate {
				t.Errorf("local date = %q, want %q", date, tt.wantDate)
			}
		})
	}

	// Go は DST の欠落時刻（2026-03-08 02:30、02:00 EST → 03:00 EDT）と重複時刻
	// （2026-11-01 01:30、EDT と EST の二度）をどちらのオフセットで解釈するか保証しない。
	// どちらに倒れても隣接する 1 時間の窓に入り、判定日は変わらない。
	dst := []struct {
		name       string
		now        time.Time
		notifyTime string
		lo, hi     time.Time
		wantDate   string
	}{
		{"DST の欠落時刻", time.Date(2026, 3, 8, 5, 0, 0, 0, time.UTC), "02:30",
			time.Date(2026, 3, 8, 6, 30, 0, 0, time.UTC), time.Date(2026, 3, 8, 7, 30, 0, 0, time.UTC), "2026-03-08"},
		{"DST の重複時刻", time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC), "01:30",
			time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC), time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC), "2026-11-01"},
	}
	for _, tt := range dst {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, date, err := nextFireAt(tt.now, tt.notifyTime, newYork)
			if err != nil {
				t.Fatalf("nextFireAt() error = %v, want nil", err)
			}
			if got.Before(tt.lo) || got.After(tt.hi) {
				t.Errorf("nextFireAt() = %s, want within [%s, %s]", got.UTC(), tt.lo, tt.hi)
			}
			if date != tt.wantDate {
				t.Errorf("local date = %q, want %q", date, tt.wantDate)
			}
		})
	}

	if _, _, err := nextFireAt(time.Now(), "9pm", tokyo); err == nil {
		t.Error("nextFireAt(9pm) error = nil, want an error")
	}
}
