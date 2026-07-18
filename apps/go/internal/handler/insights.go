package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// Insights is the HTTP transport for the (auxiliary) server-side insight
// aggregation. The client renders insights from its own local DB; this endpoint is
// the cross-device count of record, offered for a future server summary / analytics
// path. Like the other handlers it only translates requests/responses.
type Insights struct {
	svc    *service.InsightsService
	logger *slog.Logger
}

// NewInsights constructs the insights handler from its service dependency.
func NewInsights(svc *service.InsightsService, logger *slog.Logger) *Insights {
	return &Insights{svc: svc, logger: logger}
}

type insightsResponse struct {
	Period           string                `json:"period"`
	Start            string                `json:"start"`
	DaysJournaled    int                   `json:"days_journaled"`
	EntryCount       int                   `json:"entry_count"`
	UnmoodedCount    int                   `json:"unmooded_count"`
	MoodDistribution []insightMoodResponse `json:"mood_distribution"`
}

type insightMoodResponse struct {
	Mood  string `json:"mood"`
	Count int    `json:"count"`
}

// Insights handles GET /api/v1/insights?period=weekly|monthly&start=&tz= — the
// per-period aggregation grouped in the requested IANA time zone (default UTC) so
// it matches the client's local grouping.
func (h *Insights) Insights(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	period, ok := h.parsePeriod(w, r)
	if !ok {
		return
	}
	start, ok := h.parseStart(w, r)
	if !ok {
		return
	}
	loc, ok := h.parseTZ(w, r)
	if !ok {
		return
	}

	summary, err := h.svc.Insight(r.Context(), userID, period, start, loc)
	if err != nil {
		h.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toInsightsResponse(summary), h.logger)
}

func toInsightsResponse(s service.InsightSummary) insightsResponse {
	moods := make([]insightMoodResponse, 0, len(s.MoodDistribution))
	for _, m := range s.MoodDistribution {
		moods = append(moods, insightMoodResponse{Mood: m.Mood, Count: m.Count})
	}
	return insightsResponse{
		Period:           string(s.Period),
		Start:            s.Start,
		DaysJournaled:    s.DaysJournaled,
		EntryCount:       s.EntryCount,
		UnmoodedCount:    s.UnmoodedCount,
		MoodDistribution: moods,
	}
}

func (h *Insights) parsePeriod(w http.ResponseWriter, r *http.Request) (service.InsightPeriodType, bool) {
	switch r.URL.Query().Get("period") {
	case "weekly":
		return service.InsightWeekly, true
	case "monthly":
		return service.InsightMonthly, true
	default:
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "period", Message: "must be weekly or monthly"}}, h.logger)
		return "", false
	}
}

func (h *Insights) parseStart(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	raw := r.URL.Query().Get("start")
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "start", Message: "must be a YYYY-MM-DD date"}}, h.logger)
		return time.Time{}, false
	}
	return t, true
}

func (h *Insights) parseTZ(w http.ResponseWriter, r *http.Request) (*time.Location, bool) {
	raw := r.URL.Query().Get("tz")
	if raw == "" {
		return time.UTC, true
	}
	loc, err := time.LoadLocation(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "tz", Message: "must be an IANA time zone"}}, h.logger)
		return nil, false
	}
	return loc, true
}

func (h *Insights) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, h.logger)
	}
	return id, ok
}

func (h *Insights) internal(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.ErrorContext(r.Context(), "insights handler internal error", slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, h.logger)
}
