package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// moodDist builds the canonical mood distribution (fixed order calm, gentle,
// tired, hopeful, heavy) the insights aggregation always returns.
func moodDist(calm, gentle, tired, hopeful, heavy int) []service.InsightMoodCount {
	return []service.InsightMoodCount{
		{
			Mood:  "calm",
			Count: calm,
		},
		{
			Mood:  "gentle",
			Count: gentle,
		},
		{
			Mood:  "tired",
			Count: tired,
		},
		{
			Mood:  "hopeful",
			Count: hopeful,
		},
		{
			Mood:  "heavy",
			Count: heavy,
		},
	}
}

func TestInsightsService_Insight(t *testing.T) {
	t.Parallel()

	jst := time.FixedZone("JST", 9*60*60)

	type entrySeed struct {
		at   time.Time
		mood *string
	}

	tests := []struct {
		name    string
		period  service.InsightPeriodType
		loc     *time.Location
		start   time.Time
		seeds   []entrySeed
		wantErr error
		want    service.InsightSummary
	}{
		{
			name:   "週次は件数・記録日数・気分を集計する",
			period: service.InsightWeekly,
			loc:    time.UTC,
			start:  time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC), // window [07-06, 07-13)
			seeds: []entrySeed{
				{
					at:   time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				},
				{
					at:   time.Date(2026, 7, 6, 20, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				}, // same local day
				{
					at:   time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
					mood: ptr("heavy"),
				},
				{
					at:   time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
					mood: nil,
				}, // unmooded
				{
					at:   time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC),
					mood: ptr("weird"),
				}, // unknown -> unmooded
				{
					at:   time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC),
					mood: ptr("gentle"),
				}, // == hi, excluded
				{
					at:   time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				}, // before lo, excluded
			},
			want: service.InsightSummary{
				Period:           service.InsightWeekly,
				Start:            "2026-07-06",
				DaysJournaled:    4,
				EntryCount:       5,
				UnmoodedCount:    2,
				MoodDistribution: moodDist(2, 0, 0, 0, 1),
			},
		},
		{
			name:   "月次は月全体を対象にする",
			period: service.InsightMonthly,
			loc:    time.UTC,
			start:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), // window [07-01, 08-01)
			seeds: []entrySeed{
				{
					at:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				},
				{
					at:   time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC),
					mood: ptr("hopeful"),
				},
				{
					at:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
					mood: ptr("tired"),
				}, // == hi, excluded
				{
					at:   time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				}, // before lo, excluded
			},
			want: service.InsightSummary{
				Period:           service.InsightMonthly,
				Start:            "2026-07-01",
				DaysJournaled:    2,
				EntryCount:       2,
				UnmoodedCount:    0,
				MoodDistribution: moodDist(1, 0, 0, 1, 0),
			},
		},
		{
			name:   "集計と境界は指定タイムゾーンに従う",
			period: service.InsightWeekly,
			loc:    jst,
			// lo = 2026-07-06 00:00 JST = 2026-07-05 15:00 UTC; hi = 2026-07-12 15:00 UTC.
			start: time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC),
			seeds: []entrySeed{
				{
					at:   time.Date(2026, 7, 5, 20, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				}, // 07-06 05:00 JST, in window
				{
					at:   time.Date(2026, 7, 10, 16, 0, 0, 0, time.UTC),
					mood: nil,
				}, // 07-11 01:00 JST, in window
				{
					at:   time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				}, // before lo, excluded
				{
					at:   time.Date(2026, 7, 12, 16, 0, 0, 0, time.UTC),
					mood: ptr("calm"),
				}, // >= hi, excluded
			},
			want: service.InsightSummary{
				Period:           service.InsightWeekly,
				Start:            "2026-07-06",
				DaysJournaled:    2,
				EntryCount:       2,
				UnmoodedCount:    1,
				MoodDistribution: moodDist(1, 0, 0, 0, 0),
			},
		},
		{
			name:    "未知の期間種別はErrInvalidPeriod",
			period:  service.InsightPeriodType("yearly"),
			loc:     time.UTC,
			start:   time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC),
			wantErr: service.ErrInvalidPeriod,
		},
		{
			name:   "空範囲は全気分順のゼロ集計を返す",
			period: service.InsightWeekly,
			loc:    time.UTC,
			start:  time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC),
			want: service.InsightSummary{
				Period:           service.InsightWeekly,
				Start:            "2026-07-06",
				DaysJournaled:    0,
				EntryCount:       0,
				UnmoodedCount:    0,
				MoodDistribution: moodDist(0, 0, 0, 0, 0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, store := newInsightsService()
			ctx := context.Background()
			for i, s := range tt.seeds {
				_, _, err := store.UpsertEntry(ctx, repository.UpsertDiaryParams{
					UserID: userA, ClientID: clientID(i + 1), BodyText: "entry", Mood: s.mood, CreatedAt: s.at,
				})
				require.NoError(t, err)
			}

			got, err := svc.Insight(ctx, userA, tt.period, tt.start, tt.loc)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
