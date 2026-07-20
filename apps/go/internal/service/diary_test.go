package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)
			assert.Equal(t, tt.wantFirstNew, created)

			final := first
			if tt.secondInput != nil {
				second, created, err := svc.Create(ctx, userA, *tt.secondInput)
				require.NoError(t, err)
				assert.Equal(t, tt.wantSecondNew, created)
				assert.Equal(t, first.ID, second.ID)
				final = second
			}

			assert.Equal(t, tt.wantBody, final.BodyText)
			if tt.wantCreatedAtNonZero {
				assert.False(t, final.CreatedAt.IsZero())
			}

			page, err := svc.List(ctx, userA, 0, nil)
			require.NoError(t, err)
			assert.Len(t, page.Entries, tt.wantCount)
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
				require.NoError(t, err)
			}

			page1, err := svc.List(ctx, userA, tt.limit, nil)
			require.NoError(t, err)
			assert.Len(t, page1.Entries, tt.wantPage1Len)
			assert.Equal(t, tt.wantCursor, page1.NextCursor != nil)
			for i := 1; i < len(page1.Entries); i++ {
				assert.False(t, page1.Entries[i-1].CreatedAt.Before(page1.Entries[i].CreatedAt), "page1 not newest-first at index %d", i)
			}
			if !tt.wantCursor {
				return
			}

			page2, err := svc.List(ctx, userA, tt.limit, page1.NextCursor)
			require.NoError(t, err)
			assert.Len(t, page2.Entries, tt.wantPage2Len)
			if len(page2.Entries) > 0 {
				assert.True(t, page2.Entries[0].CreatedAt.Before(page1.Entries[len(page1.Entries)-1].CreatedAt), "pages overlap across the cursor boundary")
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
			require.NoError(t, err)
			id := entry.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			got, err := svc.Get(ctx, tt.reader, id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantBody, got.BodyText)
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
			require.NoError(t, err)
			id := entry.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			updated, err := svc.Update(ctx, tt.updater, id, tt.newBody, tt.newMood)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newBody, updated.BodyText)
				assert.Equal(t, tt.newMood, updated.Mood)
			}

			owner, err := svc.Get(ctx, userA, entry.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOwnerBody, owner.BodyText)
			assert.Equal(t, tt.wantOwnerMood, owner.Mood)
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
			require.NoError(t, err)
			id := entry.ID
			if !tt.useRealID {
				id = "does-not-exist"
			}

			var derr error
			for i := 0; i < tt.deleteCount; i++ {
				derr = svc.Delete(ctx, tt.deleter, id)
			}
			if tt.wantErr != nil {
				assert.ErrorIs(t, derr, tt.wantErr)
			} else {
				assert.NoError(t, derr)
			}

			page, err := svc.List(ctx, userA, 0, nil)
			require.NoError(t, err)
			assert.Len(t, page.Entries, tt.wantOwnerList)
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
				require.NoError(t, err)

				full, err := svc.Sync(ctx, userA, time.Time{})
				require.NoError(t, err)
				require.Len(t, full.Entries, 1)
				assert.Nil(t, full.Entries[0].DeletedAt)
			},
		},
		{
			name: "論理削除は差分にトゥームストーンとして乗る",
			run: func(t *testing.T, svc *service.DiaryService, ctx context.Context) {
				entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "消す"})
				require.NoError(t, err)
				require.NoError(t, svc.Delete(ctx, userA, entry.ID))

				delta, err := svc.Sync(ctx, userA, time.Time{})
				require.NoError(t, err)
				require.Len(t, delta.Entries, 1)
				assert.NotNil(t, delta.Entries[0].DeletedAt)
			},
		},
		{
			name: "ウォーターマーク以降の差分は変更分だけ返す",
			run: func(t *testing.T, svc *service.DiaryService, ctx context.Context) {
				entry, _, err := svc.Create(ctx, userA, service.CreateDiaryInput{ClientID: clientID(1), BodyText: "v1"})
				require.NoError(t, err)

				full, err := svc.Sync(ctx, userA, time.Time{})
				require.NoError(t, err)
				require.Len(t, full.Entries, 1)

				_, err = svc.Update(ctx, userA, entry.ID, "v2", ptr("calm"))
				require.NoError(t, err)

				delta, err := svc.Sync(ctx, userA, full.ServerTime)
				require.NoError(t, err)
				require.Len(t, delta.Entries, 1)
				assert.Equal(t, "v2", delta.Entries[0].BodyText)
				assert.Equal(t, ptr("calm"), delta.Entries[0].Mood)

				after, err := svc.Sync(ctx, userA, delta.ServerTime)
				require.NoError(t, err)
				assert.Empty(t, after.Entries)
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
				require.NoError(t, err)
			}

			days, err := svc.Calendar(ctx, userA, tt.year, tt.month, tt.loc)
			require.NoError(t, err)
			assert.Equal(t, tt.want, days)
		})
	}
}
