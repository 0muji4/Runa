package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/google/go-cmp/cmp"
)

func TestDiaryService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		firstInput           service.CreateDiaryInput
		secondInput          *service.CreateDiaryInput
		wantFirstNew         bool
		wantSecondNew        bool
		wantBody             string
		wantCount            int
		wantCreatedAtNonZero bool
	}{
		{
			name:                 "新規エントリを作成する",
			firstInput:           service.CreateDiaryInput{ClientID: clientID(1), BodyText: "夜の記録"},
			secondInput:          nil,
			wantFirstNew:         true,
			wantSecondNew:        false,
			wantBody:             "夜の記録",
			wantCount:            1,
			wantCreatedAtNonZero: false,
		},
		{
			name:                 "同じclient_idは重複せず上書きする",
			firstInput:           service.CreateDiaryInput{ClientID: clientID(1), BodyText: "夜の記録"},
			secondInput:          &service.CreateDiaryInput{ClientID: clientID(1), BodyText: "夜の記録（推敲）"},
			wantFirstNew:         true,
			wantSecondNew:        false,
			wantBody:             "夜の記録（推敲）",
			wantCount:            1,
			wantCreatedAtNonZero: false,
		},
		{
			name:                 "CreatedAtが未指定ならサーバ時刻を補う",
			firstInput:           service.CreateDiaryInput{ClientID: clientID(1), BodyText: "既定時刻"},
			secondInput:          nil,
			wantFirstNew:         true,
			wantSecondNew:        false,
			wantBody:             "既定時刻",
			wantCount:            1,
			wantCreatedAtNonZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newDiaryService()
			ctx := context.Background()

			first, created, err := svc.Create(ctx, userA, tt.firstInput)
			if err != nil {
				t.Fatalf("Create(%+v) error = %v, want nil", tt.firstInput, err)
			}
			if created != tt.wantFirstNew {
				t.Errorf("Create(%+v) created = %t, want %t", tt.firstInput, created, tt.wantFirstNew)
			}

			final := first
			if tt.secondInput != nil {
				second, created, err := svc.Create(ctx, userA, *tt.secondInput)
				if err != nil {
					t.Fatalf("second Create(%+v) error = %v, want nil", *tt.secondInput, err)
				}
				if created != tt.wantSecondNew {
					t.Errorf("second Create(%+v) created = %t, want %t",
						*tt.secondInput, created, tt.wantSecondNew)
				}
				if second.ID != first.ID {
					t.Errorf("second Create() id = %q, want the first entry's id %q (upsert, not insert)",
						second.ID, first.ID)
				}
				final = second
			}

			if final.BodyText != tt.wantBody {
				t.Errorf("entry body_text = %q, want %q", final.BodyText, tt.wantBody)
			}
			if tt.wantCreatedAtNonZero && final.CreatedAt.IsZero() {
				t.Error("entry created_at is zero, want the server clock to fill it in")
			}

			page, err := svc.List(ctx, userA, 0, nil)
			if err != nil {
				t.Fatalf("List() error = %v, want nil", err)
			}
			if len(page.Entries) != tt.wantCount {
				t.Errorf("List() returned %d entries, want %d", len(page.Entries), tt.wantCount)
			}
		})
	}
}

func TestDiaryService_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		seed         int
		limit        int
		wantPage1Len int
		wantCursor   bool
		wantPage2Len int
	}{
		{
			name:         "空ストアはエントリもカーソルも返さない",
			seed:         0,
			limit:        2,
			wantPage1Len: 0,
			wantCursor:   false,
			wantPage2Len: 0,
		},
		{
			name:         "2ページが新しい順で重複なく連結する",
			seed:         5,
			limit:        2,
			wantPage1Len: 2,
			wantCursor:   true,
			wantPage2Len: 2,
		},
		{
			name:         "端数のない満杯ページはカーソルを返さない",
			seed:         3,
			limit:        5,
			wantPage1Len: 3,
			wantCursor:   false,
			wantPage2Len: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newDiaryService()
			ctx := context.Background()
			base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
			for i := 1; i <= tt.seed; i++ {
				_, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{
					ClientID:  clientID(i),
					BodyText:  "entry",
					CreatedAt: base.Add(time.Duration(i) * time.Minute),
				})
				if err != nil {
					t.Fatalf("seeding entry %d: Create() error = %v, want nil", i, err)
				}
			}

			page1, err := svc.List(ctx, userA, tt.limit, nil)
			if err != nil {
				t.Fatalf("List(limit=%d) error = %v, want nil", tt.limit, err)
			}
			if len(page1.Entries) != tt.wantPage1Len {
				t.Errorf("List(limit=%d) page 1 returned %d entries, want %d",
					tt.limit, len(page1.Entries), tt.wantPage1Len)
			}
			if got := page1.NextCursor != nil; got != tt.wantCursor {
				t.Errorf("List(limit=%d) page 1 has a next cursor = %t, want %t",
					tt.limit, got, tt.wantCursor)
			}
			for i := 1; i < len(page1.Entries); i++ {
				if page1.Entries[i-1].CreatedAt.Before(page1.Entries[i].CreatedAt) {
					t.Errorf("List() page 1 is not newest-first at index %d: %s before %s",
						i, page1.Entries[i-1].CreatedAt, page1.Entries[i].CreatedAt)
				}
			}
			if !tt.wantCursor {
				return
			}

			page2, err := svc.List(ctx, userA, tt.limit, page1.NextCursor)
			if err != nil {
				t.Fatalf("List(limit=%d, cursor) error = %v, want nil", tt.limit, err)
			}
			if len(page2.Entries) != tt.wantPage2Len {
				t.Errorf("List(limit=%d, cursor) page 2 returned %d entries, want %d",
					tt.limit, len(page2.Entries), tt.wantPage2Len)
			}
			if len(page2.Entries) > 0 {
				lastOfPage1 := page1.Entries[len(page1.Entries)-1].CreatedAt
				if !page2.Entries[0].CreatedAt.Before(lastOfPage1) {
					t.Errorf("pages overlap across the cursor boundary: page 2 starts at %s, page 1 ended at %s",
						page2.Entries[0].CreatedAt, lastOfPage1)
				}
			}
		})
	}
}

func TestDiaryService_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		reader    string
		useRealID bool
		wantErr   error
		wantBody  string
	}{
		{
			name:      "所有者は自分のエントリを読める",
			reader:    userA,
			useRealID: true,
			wantErr:   nil,
			wantBody:  "秘密",
		},
		{
			name:      "別ユーザーは読めずErrDiaryNotFound",
			reader:    userB,
			useRealID: true,
			wantErr:   service.ErrDiaryNotFound,
			wantBody:  "",
		},
		{
			name:      "未知のIDはErrDiaryNotFound",
			reader:    userA,
			useRealID: false,
			wantErr:   service.ErrDiaryNotFound,
			wantBody:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newDiaryService()
			ctx := context.Background()
			entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "秘密"})
			if err != nil {
				t.Fatalf("Create() error = %v, want nil", err)
			}
			id := entry.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			got, err := svc.Get(ctx, tt.reader, id)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Get(%q, %q) error = %v, want %v", tt.reader, id, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get(%q, %q) error = %v, want nil", tt.reader, id, err)
			}
			if got.BodyText != tt.wantBody {
				t.Errorf("Get(%q, %q) body_text = %q, want %q", tt.reader, id, got.BodyText, tt.wantBody)
			}
		})
	}
}

func TestDiaryService_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		updater       string
		useRealID     bool
		newBody       string
		newMood       *string
		wantErr       error
		wantOwnerBody string
		wantOwnerMood *string
	}{
		{
			name:          "所有者は本文と気分を更新できる",
			updater:       userA,
			useRealID:     true,
			newBody:       "改稿",
			newMood:       ptr("calm"),
			wantErr:       nil,
			wantOwnerBody: "改稿",
			wantOwnerMood: ptr("calm"),
		},
		{
			name:          "別ユーザーは更新できずErrDiaryNotFound",
			updater:       userB,
			useRealID:     true,
			newBody:       "改ざん",
			newMood:       ptr("heavy"),
			wantErr:       service.ErrDiaryNotFound,
			wantOwnerBody: "秘密",
			wantOwnerMood: nil,
		},
		{
			name:          "未知のIDはErrDiaryNotFound",
			updater:       userA,
			useRealID:     false,
			newBody:       "x",
			newMood:       nil,
			wantErr:       service.ErrDiaryNotFound,
			wantOwnerBody: "秘密",
			wantOwnerMood: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newDiaryService()
			ctx := context.Background()
			entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "秘密"})
			if err != nil {
				t.Fatalf("Create() error = %v, want nil", err)
			}
			id := entry.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			updated, err := svc.Update(ctx, tt.updater, id, tt.newBody, tt.newMood)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Update(%q, %q) error = %v, want %v", tt.updater, id, err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("Update(%q, %q) error = %v, want nil", tt.updater, id, err)
				}
				if updated.BodyText != tt.newBody {
					t.Errorf("Update() body_text = %q, want %q", updated.BodyText, tt.newBody)
				}
				if diff := cmp.Diff(tt.newMood, updated.Mood); diff != "" {
					t.Errorf("Update() mood mismatch (-want +got):\n%s", diff)
				}
			}

			// Re-read as the owner: a rejected update must not have changed anything.
			owner, err := svc.Get(ctx, userA, entry.ID)
			if err != nil {
				t.Fatalf("Get(owner, %q) error = %v, want nil", entry.ID, err)
			}
			if owner.BodyText != tt.wantOwnerBody {
				t.Errorf("owner's body_text = %q, want %q", owner.BodyText, tt.wantOwnerBody)
			}
			if diff := cmp.Diff(tt.wantOwnerMood, owner.Mood); diff != "" {
				t.Errorf("owner's mood mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiaryService_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		deleter       string
		useRealID     bool
		deleteCount   int
		wantErr       error
		wantOwnerList int
	}{
		{
			name:          "所有者の論理削除で一覧から消える",
			deleter:       userA,
			useRealID:     true,
			deleteCount:   1,
			wantErr:       nil,
			wantOwnerList: 0,
		},
		{
			name:          "繰り返し削除は冪等",
			deleter:       userA,
			useRealID:     true,
			deleteCount:   2,
			wantErr:       nil,
			wantOwnerList: 0,
		},
		{
			name:          "別ユーザーは削除できずErrDiaryNotFound",
			deleter:       userB,
			useRealID:     true,
			deleteCount:   1,
			wantErr:       service.ErrDiaryNotFound,
			wantOwnerList: 1,
		},
		{
			name:          "未知のIDはErrDiaryNotFound",
			deleter:       userA,
			useRealID:     false,
			deleteCount:   1,
			wantErr:       service.ErrDiaryNotFound,
			wantOwnerList: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newDiaryService()
			ctx := context.Background()
			entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "消す"})
			if err != nil {
				t.Fatalf("Create() error = %v, want nil", err)
			}
			id := entry.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			var derr error
			for i := 0; i < tt.deleteCount; i++ {
				derr = svc.Delete(ctx, tt.deleter, id)
			}
			if tt.wantErr != nil {
				if !errors.Is(derr, tt.wantErr) {
					t.Fatalf("Delete(%q, %q) x%d error = %v, want %v",
						tt.deleter, id, tt.deleteCount, derr, tt.wantErr)
				}
			} else if derr != nil {
				t.Fatalf("Delete(%q, %q) x%d error = %v, want nil",
					tt.deleter, id, tt.deleteCount, derr)
			}

			page, err := svc.List(ctx, userA, 0, nil)
			if err != nil {
				t.Fatalf("List() error = %v, want nil", err)
			}
			if len(page.Entries) != tt.wantOwnerList {
				t.Errorf("owner's List() returned %d entries, want %d",
					len(page.Entries), tt.wantOwnerList)
			}
		})
	}
}

func TestDiaryService_Sync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T, svc *service.DiaryService, ctx context.Context)
	}{
		{
			name: "エポックからの全同期は生存エントリを返す",
			run: func(t *testing.T, svc *service.DiaryService, ctx context.Context) {
				_, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "v1"})
				if err != nil {
					t.Fatalf("Create() error = %v, want nil", err)
				}

				full, err := svc.Sync(ctx, userA, time.Time{})
				if err != nil {
					t.Fatalf("Sync(epoch) error = %v, want nil", err)
				}
				if len(full.Entries) != 1 {
					t.Fatalf("Sync(epoch) returned %d entries, want 1", len(full.Entries))
				}
				if full.Entries[0].DeletedAt != nil {
					t.Errorf("Sync(epoch) entry deleted_at = %s, want nil (entry is alive)",
						*full.Entries[0].DeletedAt)
				}
			},
		},
		{
			name: "論理削除は差分にトゥームストーンとして乗る",
			run: func(t *testing.T, svc *service.DiaryService, ctx context.Context) {
				entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "消す"})
				if err != nil {
					t.Fatalf("Create() error = %v, want nil", err)
				}
				if err := svc.Delete(ctx, userA, entry.ID); err != nil {
					t.Fatalf("Delete(%q) error = %v, want nil", entry.ID, err)
				}

				delta, err := svc.Sync(ctx, userA, time.Time{})
				if err != nil {
					t.Fatalf("Sync(epoch) error = %v, want nil", err)
				}
				if len(delta.Entries) != 1 {
					t.Fatalf("Sync(epoch) returned %d entries, want 1 tombstone", len(delta.Entries))
				}
				if delta.Entries[0].DeletedAt == nil {
					t.Error("Sync(epoch) entry deleted_at = nil, want a tombstone timestamp")
				}
			},
		},
		{
			name: "ウォーターマーク以降の差分は変更分だけ返す",
			run: func(t *testing.T, svc *service.DiaryService, ctx context.Context) {
				entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "v1"})
				if err != nil {
					t.Fatalf("Create() error = %v, want nil", err)
				}

				full, err := svc.Sync(ctx, userA, time.Time{})
				if err != nil {
					t.Fatalf("Sync(epoch) error = %v, want nil", err)
				}
				if len(full.Entries) != 1 {
					t.Fatalf("Sync(epoch) returned %d entries, want 1", len(full.Entries))
				}

				if _, err := svc.Update(ctx, userA, entry.ID, "v2", ptr("calm")); err != nil {
					t.Fatalf("Update(%q) error = %v, want nil", entry.ID, err)
				}

				delta, err := svc.Sync(ctx, userA, full.ServerTime)
				if err != nil {
					t.Fatalf("Sync(watermark) error = %v, want nil", err)
				}
				if len(delta.Entries) != 1 {
					t.Fatalf("Sync(watermark) returned %d entries, want 1", len(delta.Entries))
				}
				if got := delta.Entries[0].BodyText; got != "v2" {
					t.Errorf("Sync(watermark) entry body_text = %q, want %q", got, "v2")
				}
				if diff := cmp.Diff(ptr("calm"), delta.Entries[0].Mood); diff != "" {
					t.Errorf("Sync(watermark) entry mood mismatch (-want +got):\n%s", diff)
				}

				// Syncing again from the new watermark must return nothing.
				after, err := svc.Sync(ctx, userA, delta.ServerTime)
				if err != nil {
					t.Fatalf("Sync(new watermark) error = %v, want nil", err)
				}
				if len(after.Entries) != 0 {
					t.Errorf("Sync(new watermark) returned %d entries, want 0", len(after.Entries))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, newDiaryService(), context.Background())
		})
	}
}

func TestDiaryService_Calendar(t *testing.T) {
	t.Parallel()

	jst := time.FixedZone("JST", 9*60*60)
	tests := []struct {
		name  string
		loc   *time.Location
		year  int
		month int
		seeds []time.Time
		want  []service.DiaryCalendarDay
	}{
		{
			name:  "UTCの現地日付で件数を集計する",
			loc:   time.UTC,
			year:  2026,
			month: 7,
			seeds: []time.Time{
				time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC),
				time.Date(2026, 7, 10, 18, 0, 0, 0, time.UTC),
				time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
			},
			want: []service.DiaryCalendarDay{
				{
					Date:  "2026-07-10",
					Count: 2,
				},
				{
					Date:  "2026-07-15",
					Count: 1,
				},
			},
		},
		{
			name:  "指定タイムゾーンの現地日付で集計する",
			loc:   jst,
			year:  2026,
			month: 7,
			seeds: []time.Time{
				time.Date(2026, 7, 9, 20, 0, 0, 0, time.UTC),  // 2026-07-10 05:00 JST
				time.Date(2026, 7, 10, 16, 0, 0, 0, time.UTC), // 2026-07-11 01:00 JST
			},
			want: []service.DiaryCalendarDay{
				{
					Date:  "2026-07-10",
					Count: 1,
				},
				{
					Date:  "2026-07-11",
					Count: 1,
				},
			},
		},
		{
			name:  "月の範囲外のエントリを除外する",
			loc:   time.UTC,
			year:  2026,
			month: 7,
			seeds: []time.Time{
				time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
				time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
				time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
			},
			want: []service.DiaryCalendarDay{
				{
					Date:  "2026-07-15",
					Count: 1,
				},
			},
		},
		{
			name:  "エントリのない月は空を返す",
			loc:   time.UTC,
			year:  2026,
			month: 7,
			seeds: nil,
			want:  []service.DiaryCalendarDay{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newDiaryService()
			ctx := context.Background()
			for i, at := range tt.seeds {
				_, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{
					ClientID:  clientID(i + 1),
					BodyText:  "entry",
					CreatedAt: at,
				})
				if err != nil {
					t.Fatalf("seeding entry at %s: Create() error = %v, want nil", at, err)
				}
			}

			days, err := svc.Calendar(ctx, userA, tt.year, tt.month, tt.loc)
			if err != nil {
				t.Fatalf("Calendar(%d, %d, %s) error = %v, want nil",
					tt.year, tt.month, tt.loc, err)
			}
			if diff := cmp.Diff(tt.want, days); diff != "" {
				t.Errorf("Calendar(%d, %d, %s) mismatch (-want +got):\n%s",
					tt.year, tt.month, tt.loc, diff)
			}
		})
	}
}
