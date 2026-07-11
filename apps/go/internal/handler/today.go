package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// dateLayout is the YYYY-MM-DD form the today endpoints accept and emit. It
// matches the DATE columns' day-only semantics and the client's local date.
const dateLayout = "2006-01-02"

// Today is the HTTP transport for the today feature (daily quote + song, the
// song archive, play history, and the admin seed endpoints). Like the other
// handlers it only translates requests/responses; the logic lives in the service.
type Today struct {
	svc    *service.TodayService
	logger *slog.Logger
}

// NewToday constructs the today handler from its service dependency.
func NewToday(svc *service.TodayService, logger *slog.Logger) *Today {
	return &Today{svc: svc, logger: logger}
}

// ---- request/response DTOs (JSON contract) ----

type quoteResponse struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	BodyText string `json:"body_text"`
}

type songResponse struct {
	ID         string `json:"id"`
	Date       string `json:"date"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	ArtworkURL string `json:"artwork_url"`
	AudioURL   string `json:"audio_url"`
}

type todayResponse struct {
	Date  string         `json:"date"`
	Quote *quoteResponse `json:"quote"`
	Song  *songResponse  `json:"song"`
}

type songsListResponse struct {
	Songs      []songResponse `json:"songs"`
	NextCursor *string        `json:"next_cursor"`
}

type playedRequest struct {
	PlayedAt string `json:"played_at"` // RFC3339; optional (server clock when absent)
}

type createQuoteRequest struct {
	Date     string `json:"date"`
	BodyText string `json:"body_text"`
}

type createSongRequest struct {
	Date       string `json:"date"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	ArtworkURL string `json:"artwork_url"`
	AudioURL   string `json:"audio_url"`
}

// ---- handlers ----

// Today handles GET /api/v1/today?date=YYYY-MM-DD. An absent date uses the
// server's current UTC day. Missing quote/song come back as null (not an error).
func (t *Today) Today(w http.ResponseWriter, r *http.Request) {
	if _, ok := t.userID(w, r); !ok {
		return
	}

	date, ok := t.parseDateQuery(w, r)
	if !ok {
		return
	}

	content, err := t.svc.Today(r.Context(), date)
	if err != nil {
		t.internal(w, r, err)
		return
	}

	resp := todayResponse{Date: date.Format(dateLayout)}
	if content.Quote != nil {
		q := toQuoteResponse(*content.Quote)
		resp.Quote = &q
	}
	if content.Song != nil {
		s := toSongResponse(*content.Song)
		resp.Song = &s
	}
	writeJSON(w, http.StatusOK, resp, t.logger)
}

// Songs handles GET /api/v1/songs?limit=&cursor= (archive, newest first).
func (t *Today) Songs(w http.ResponseWriter, r *http.Request) {
	if _, ok := t.userID(w, r); !ok {
		return
	}

	limit, ok := t.parseLimit(w, r)
	if !ok {
		return
	}
	cursor, ok := t.parseSongCursor(w, r)
	if !ok {
		return
	}

	page, err := t.svc.Archive(r.Context(), limit, cursor)
	if err != nil {
		t.internal(w, r, err)
		return
	}

	resp := songsListResponse{Songs: toSongResponses(page.Songs)}
	if page.NextCursor != nil {
		c := encodeSongCursor(*page.NextCursor)
		resp.NextCursor = &c
	}
	writeJSON(w, http.StatusOK, resp, t.logger)
}

// Played handles POST /api/v1/songs/{id}/played (records a play). An unknown
// song id answers 404.
func (t *Today) Played(w http.ResponseWriter, r *http.Request) {
	userID, ok := t.userID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !uuidPattern.MatchString(id) {
		t.songNotFound(w)
		return
	}

	// Body is optional: a bare POST records a play at the server clock.
	var req playedRequest
	if r.ContentLength != 0 && r.Body != nil {
		if !t.decodeAllowEmpty(w, r, &req) {
			return
		}
	}
	playedAt, ok := t.parsePlayedAt(w, req.PlayedAt)
	if !ok {
		return
	}

	if err := t.svc.MarkPlayed(r.Context(), userID, id, playedAt); err != nil {
		if errors.Is(err, service.ErrSongNotFound) {
			t.songNotFound(w)
			return
		}
		t.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil, t.logger)
}

// CreateQuote handles POST /api/v1/admin/quotes (admin; behind X-Admin-Token).
func (t *Today) CreateQuote(w http.ResponseWriter, r *http.Request) {
	var req createQuoteRequest
	if !t.decode(w, r, &req) {
		return
	}
	date, ok := t.parseRequiredDate(w, req.Date)
	if !ok {
		return
	}
	if strings.TrimSpace(req.BodyText) == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "body_text", Message: "must not be empty"}}, t.logger)
		return
	}

	quote, err := t.svc.CreateQuote(r.Context(), date, req.BodyText)
	if err != nil {
		t.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toQuoteResponse(quote), t.logger)
}

// CreateSong handles POST /api/v1/admin/songs (admin; behind X-Admin-Token).
func (t *Today) CreateSong(w http.ResponseWriter, r *http.Request) {
	var req createSongRequest
	if !t.decode(w, r, &req) {
		return
	}
	date, ok := t.parseRequiredDate(w, req.Date)
	if !ok {
		return
	}
	if details := validateCreateSong(req); len(details) > 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed", details, t.logger)
		return
	}

	song, err := t.svc.CreateSong(r.Context(), repository.InsertSongParams{
		Date:       date,
		Title:      req.Title,
		Artist:     req.Artist,
		ArtworkURL: req.ArtworkURL,
		AudioURL:   req.AudioURL,
	})
	if err != nil {
		t.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSongResponse(song), t.logger)
}

// ---- helpers ----

func (t *Today) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, t.logger)
	}
	return id, ok
}

func (t *Today) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, t.logger)
		return false
	}
	return true
}

// decodeAllowEmpty decodes a body that is permitted to be empty (io.EOF is not an
// error); any other malformed JSON is rejected.
func (t *Today) decodeAllowEmpty(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, t.logger)
		return false
	}
	return true
}

func (t *Today) parseDateQuery(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	raw := r.URL.Query().Get("date")
	if raw == "" {
		// Absent date: the server's current UTC calendar day.
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), true
	}
	return t.parseRequiredDate(w, raw)
}

func (t *Today) parseRequiredDate(w http.ResponseWriter, raw string) (time.Time, bool) {
	d, err := time.Parse(dateLayout, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "date", Message: "must be a YYYY-MM-DD date"}}, t.logger)
		return time.Time{}, false
	}
	return d, true
}

func (t *Today) parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0, true // service applies the default
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "limit", Message: "must be a non-negative integer"}}, t.logger)
		return 0, false
	}
	return limit, true
}

func (t *Today) parseSongCursor(w http.ResponseWriter, r *http.Request) (*repository.SongCursor, bool) {
	raw := r.URL.Query().Get("cursor")
	if raw == "" {
		return nil, true
	}
	cursor, err := decodeSongCursor(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "cursor", Message: "is malformed"}}, t.logger)
		return nil, false
	}
	return cursor, true
}

func (t *Today) parsePlayedAt(w http.ResponseWriter, raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, true // service falls back to the server clock
	}
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "played_at", Message: "must be an RFC3339 timestamp"}}, t.logger)
		return time.Time{}, false
	}
	return ts, true
}

func (t *Today) songNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, CodeNotFound, "song not found", nil, t.logger)
}

// Forbidden writes the 403 body when the admin token check fails (satisfies
// auth.ErrorResponder for the RequireAdmin middleware).
func (t *Today) Forbidden(w http.ResponseWriter, _ *http.Request, _ error) {
	writeError(w, http.StatusForbidden, CodeForbidden, "admin access forbidden", nil, t.logger)
}

func (t *Today) internal(w http.ResponseWriter, r *http.Request, err error) {
	t.logger.ErrorContext(r.Context(), "today handler internal error", slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, t.logger)
}

func validateCreateSong(req createSongRequest) []FieldError {
	var details []FieldError
	for _, f := range []struct{ field, value string }{
		{"title", req.Title},
		{"artist", req.Artist},
		{"artwork_url", req.ArtworkURL},
		{"audio_url", req.AudioURL},
	} {
		if strings.TrimSpace(f.value) == "" {
			details = append(details, FieldError{Field: f.field, Message: "must not be empty"})
		}
	}
	return details
}

// encodeSongCursor packs an archive keyset boundary into an opaque base64url
// token of "<date YYYY-MM-DD>|<id>". Clients treat it as opaque and echo it back.
func encodeSongCursor(c repository.SongCursor) string {
	raw := c.Date.UTC().Format(dateLayout) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeSongCursor(s string) (*repository.SongCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	dateRaw, id, found := strings.Cut(string(b), "|")
	if !found || id == "" {
		return nil, errors.New("malformed cursor")
	}
	date, err := time.Parse(dateLayout, dateRaw)
	if err != nil {
		return nil, err
	}
	return &repository.SongCursor{Date: date, ID: id}, nil
}

func toQuoteResponse(q repository.Quote) quoteResponse {
	return quoteResponse{ID: q.ID, Date: q.Date.UTC().Format(dateLayout), BodyText: q.BodyText}
}

func toSongResponse(s repository.Song) songResponse {
	return songResponse{
		ID:         s.ID,
		Date:       s.Date.UTC().Format(dateLayout),
		Title:      s.Title,
		Artist:     s.Artist,
		ArtworkURL: s.ArtworkURL,
		AudioURL:   s.AudioURL,
	}
}

func toSongResponses(songs []repository.Song) []songResponse {
	out := make([]songResponse, 0, len(songs))
	for _, s := range songs {
		out = append(out, toSongResponse(s))
	}
	return out
}
