package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// uuidPattern loosely validates the client-supplied client_id and path ids. A
// malformed id is rejected before it reaches Postgres (which would 500 on a bad
// UUID cast); for path ids it collapses to a 404, matching "not your entry".
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Diary is the HTTP transport for the diary endpoints. Like Auth it only
// translates requests/responses and maps service errors to the shared envelope;
// the logic lives in the service layer.
type Diary struct {
	svc    *service.DiaryService
	logger *slog.Logger
}

// NewDiary constructs the diary handler from its service dependency.
func NewDiary(svc *service.DiaryService, logger *slog.Logger) *Diary {
	return &Diary{svc: svc, logger: logger}
}

type createDiaryRequest struct {
	BodyText  string  `json:"body_text"`
	Mood      *string `json:"mood"`
	ClientID  string  `json:"client_id"`
	CreatedAt string  `json:"created_at"` // RFC3339; optional (server clock when absent)
}

type updateDiaryRequest struct {
	BodyText *string `json:"body_text"`
	Mood     *string `json:"mood"`
}

type diaryEntryResponse struct {
	ID        string  `json:"id"`
	ClientID  string  `json:"client_id"`
	BodyText  string  `json:"body_text"`
	Mood      *string `json:"mood"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	DeletedAt *string `json:"deleted_at"`
}

type diaryListResponse struct {
	Entries    []diaryEntryResponse `json:"entries"`
	NextCursor *string              `json:"next_cursor"`
}

type diarySyncResponse struct {
	Entries    []diaryEntryResponse `json:"entries"`
	ServerTime string               `json:"server_time"`
}

// List handles GET /api/v1/diary?limit=&cursor= (newest first, keyset paging).
func (d *Diary) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := d.userID(w, r)
	if !ok {
		return
	}

	limit, ok := d.parseLimit(w, r)
	if !ok {
		return
	}
	cursor, ok := d.parseCursor(w, r)
	if !ok {
		return
	}

	page, err := d.svc.List(r.Context(), userID, limit, cursor)
	if err != nil {
		d.internal(w, r, err)
		return
	}

	resp := diaryListResponse{Entries: toDiaryEntryResponses(page.Entries)}
	if page.NextCursor != nil {
		c := encodeCursor(*page.NextCursor)
		resp.NextCursor = &c
	}
	writeJSON(w, http.StatusOK, resp, d.logger)
}

// Create handles POST /api/v1/diary. Idempotent by client_id: a repeated
// client_id upserts the same row and returns 200 instead of 201.
func (d *Diary) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := d.userID(w, r)
	if !ok {
		return
	}

	var req createDiaryRequest
	if !d.decode(w, r, &req) {
		return
	}
	if details := validateCreateDiary(req); len(details) > 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed", details, d.logger)
		return
	}

	createdAt, ok := d.parseCreatedAt(w, req.CreatedAt)
	if !ok {
		return
	}

	entry, created, err := d.svc.Create(r.Context(), userID, service.CreateDiaryInput{
		ClientID:  req.ClientID,
		BodyText:  req.BodyText,
		Mood:      req.Mood,
		CreatedAt: createdAt,
	})
	if err != nil {
		d.internal(w, r, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, toDiaryEntryResponse(entry), d.logger)
}

// Get handles GET /api/v1/diary/{id}.
func (d *Diary) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := d.userID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !uuidPattern.MatchString(id) {
		d.notFound(w)
		return
	}

	entry, err := d.svc.Get(r.Context(), userID, id)
	if err != nil {
		d.writeDiaryError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDiaryEntryResponse(entry), d.logger)
}

// Update handles PATCH /api/v1/diary/{id} (replaces body_text and mood).
func (d *Diary) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := d.userID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !uuidPattern.MatchString(id) {
		d.notFound(w)
		return
	}

	var req updateDiaryRequest
	if !d.decode(w, r, &req) {
		return
	}
	if req.BodyText == nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "body_text", Message: "is required"}}, d.logger)
		return
	}
	if strings.TrimSpace(*req.BodyText) == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "body_text", Message: "must not be empty"}}, d.logger)
		return
	}

	entry, err := d.svc.Update(r.Context(), userID, id, *req.BodyText, req.Mood)
	if err != nil {
		d.writeDiaryError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDiaryEntryResponse(entry), d.logger)
}

// Delete handles DELETE /api/v1/diary/{id} (soft delete; idempotent).
func (d *Diary) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := d.userID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !uuidPattern.MatchString(id) {
		d.notFound(w)
		return
	}

	if err := d.svc.Delete(r.Context(), userID, id); err != nil {
		d.writeDiaryError(w, r, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil, d.logger)
}

// Sync handles GET /api/v1/diary/sync?since= (delta of changes incl. tombstones).
func (d *Diary) Sync(w http.ResponseWriter, r *http.Request) {
	userID, ok := d.userID(w, r)
	if !ok {
		return
	}

	var since time.Time
	if raw := r.URL.Query().Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
				[]FieldError{{Field: "since", Message: "must be an RFC3339 timestamp"}}, d.logger)
			return
		}
		since = parsed
	}

	delta, err := d.svc.Sync(r.Context(), userID, since)
	if err != nil {
		d.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, diarySyncResponse{
		Entries:    toDiaryEntryResponses(delta.Entries),
		ServerTime: delta.ServerTime.UTC().Format(time.RFC3339Nano),
	}, d.logger)
}

func (d *Diary) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		// Should not happen behind RequireAuth, but fail closed.
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, d.logger)
	}
	return id, ok
}

func (d *Diary) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, d.logger)
		return false
	}
	return true
}

func (d *Diary) parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0, true // service applies the default
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "limit", Message: "must be a non-negative integer"}}, d.logger)
		return 0, false
	}
	return limit, true
}

func (d *Diary) parseCursor(w http.ResponseWriter, r *http.Request) (*repository.DiaryCursor, bool) {
	raw := r.URL.Query().Get("cursor")
	if raw == "" {
		return nil, true
	}
	cursor, err := decodeCursor(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "cursor", Message: "is malformed"}}, d.logger)
		return nil, false
	}
	return cursor, true
}

func (d *Diary) parseCreatedAt(w http.ResponseWriter, raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, true // service falls back to the server clock
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "created_at", Message: "must be an RFC3339 timestamp"}}, d.logger)
		return time.Time{}, false
	}
	return t, true
}

func (d *Diary) writeDiaryError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, service.ErrDiaryNotFound) {
		d.notFound(w)
		return
	}
	d.internal(w, r, err)
}

func (d *Diary) notFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, CodeNotFound, "diary entry not found", nil, d.logger)
}

func (d *Diary) internal(w http.ResponseWriter, r *http.Request, err error) {
	d.logger.ErrorContext(r.Context(), "diary handler internal error", slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, d.logger)
}

func validateCreateDiary(req createDiaryRequest) []FieldError {
	var details []FieldError
	if strings.TrimSpace(req.BodyText) == "" {
		details = append(details, FieldError{Field: "body_text", Message: "must not be empty"})
	}
	if !uuidPattern.MatchString(req.ClientID) {
		details = append(details, FieldError{Field: "client_id", Message: "must be a UUID"})
	}
	return details
}

// encodeCursor packs a keyset boundary into an opaque base64url token of
// "<createdAt RFC3339Nano>|<id>". Clients treat it as opaque and echo it back.
func encodeCursor(c repository.DiaryCursor) string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (*repository.DiaryCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	createdRaw, id, found := strings.Cut(string(b), "|")
	if !found || id == "" {
		return nil, errors.New("malformed cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdRaw)
	if err != nil {
		return nil, err
	}
	return &repository.DiaryCursor{CreatedAt: createdAt, ID: id}, nil
}

func toDiaryEntryResponses(entries []repository.DiaryEntry) []diaryEntryResponse {
	out := make([]diaryEntryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toDiaryEntryResponse(e))
	}
	return out
}

func toDiaryEntryResponse(e repository.DiaryEntry) diaryEntryResponse {
	resp := diaryEntryResponse{
		ID:        e.ID,
		ClientID:  e.ClientID,
		BodyText:  e.BodyText,
		Mood:      e.Mood,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if e.DeletedAt != nil {
		s := e.DeletedAt.UTC().Format(time.RFC3339Nano)
		resp.DeletedAt = &s
	}
	return resp
}
