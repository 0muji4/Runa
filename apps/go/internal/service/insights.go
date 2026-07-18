package service

import (
	"context"
	"errors"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/repository"
)

// InsightPeriodType is the aggregation window kind. It mirrors the shared client's
// InsightPeriodType so the two agree on semantics.
type InsightPeriodType string

const (
	InsightWeekly  InsightPeriodType = "weekly"
	InsightMonthly InsightPeriodType = "monthly"
)

// ErrInvalidPeriod is returned for an unrecognised period (the handler validates
// first, so this is a defensive fallback).
var ErrInvalidPeriod = errors.New("service: invalid insight period")

// insightMoods is the canonical mood order — it MUST match the shared
// com.runa.shared.feature.diary.DiaryMood so the server's distribution lines up
// with the client's local aggregation.
var insightMoods = []string{"calm", "gentle", "tired", "hopeful", "heavy"}

// InsightMoodCount is one mood and its count within the period.
type InsightMoodCount struct {
	Mood  string
	Count int
}

// InsightSummary is the server-side aggregation of a period. It is deliberately a
// subset of what the client computes locally (no streak/moon overlap — the moon
// stays client-only): this endpoint is the auxiliary cross-device count of record,
// never the render path.
type InsightSummary struct {
	Period           InsightPeriodType
	Start            string // local start date "YYYY-MM-DD"
	DaysJournaled    int
	EntryCount       int
	UnmoodedCount    int
	MoodDistribution []InsightMoodCount
}

// InsightsStore is the narrow data access the insights aggregation needs: the
// user's entries in an instant range. Both the pgx DiaryRepository and memdiary
// satisfy it, so no change to DiaryStore (and its fakes) is required.
type InsightsStore interface {
	EntriesInRange(ctx context.Context, userID string, lo, hi time.Time) ([]repository.DiaryEntry, error)
}

// InsightsService aggregates a user's diary entries for a period, grouped in the
// requested time zone so day counts match the client's local-date grouping.
type InsightsService struct {
	store InsightsStore
}

// NewInsightsService constructs the service over an InsightsStore.
func NewInsightsService(store InsightsStore) *InsightsService {
	return &InsightsService{store: store}
}

// Insight aggregates the entries in the half-open window that starts at [start]
// 00:00 in [loc] and spans one week or one month. Days are distinct local dates;
// mood distribution counts the canonical moods (unknown/absent moods fall into
// UnmoodedCount, matching the client's "未選択" handling).
func (s *InsightsService) Insight(ctx context.Context, userID string, period InsightPeriodType, start time.Time, loc *time.Location) (InsightSummary, error) {
	lo := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	var hi time.Time
	switch period {
	case InsightWeekly:
		hi = lo.AddDate(0, 0, 7)
	case InsightMonthly:
		hi = lo.AddDate(0, 1, 0)
	default:
		return InsightSummary{}, ErrInvalidPeriod
	}

	entries, err := s.store.EntriesInRange(ctx, userID, lo, hi)
	if err != nil {
		return InsightSummary{}, err
	}

	days := make(map[string]struct{})
	moodCounts := make(map[string]int)
	unmooded := 0
	for _, e := range entries {
		days[e.CreatedAt.In(loc).Format("2006-01-02")] = struct{}{}
		if e.Mood != nil && isKnownMood(*e.Mood) {
			moodCounts[*e.Mood]++
		} else {
			unmooded++
		}
	}

	dist := make([]InsightMoodCount, 0, len(insightMoods))
	for _, m := range insightMoods {
		dist = append(dist, InsightMoodCount{Mood: m, Count: moodCounts[m]})
	}

	return InsightSummary{
		Period:           period,
		Start:            lo.Format("2006-01-02"),
		DaysJournaled:    len(days),
		EntryCount:       len(entries),
		UnmoodedCount:    unmooded,
		MoodDistribution: dist,
	}, nil
}

func isKnownMood(m string) bool {
	for _, k := range insightMoods {
		if k == m {
			return true
		}
	}
	return false
}
