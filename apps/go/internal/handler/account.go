package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// exportSchemaVersion is the export payload's schema version. Bump it on any
// breaking change to the export shape so clients can branch on it.
const exportSchemaVersion = 1

// Account is the HTTP transport for account-data management: display-name update,
// self-service export and account deletion. GET /me stays on the Auth handler
// (identity read); these operations mutate or aggregate the whole account and
// compose several stores, so they live in their own handler backed by
// AccountService.
type Account struct {
	svc    *service.AccountService
	logger *slog.Logger
}

// NewAccount constructs the account handler from its service dependency.
func NewAccount(svc *service.AccountService, logger *slog.Logger) *Account {
	return &Account{svc: svc, logger: logger}
}

type updateMeRequest struct {
	DisplayName string `json:"display_name"`
}

type exportResponse struct {
	ExportedAt    string             `json:"exported_at"`
	SchemaVersion int                `json:"schema_version"`
	User          userResponse       `json:"user"`
	Diaries       []exportDiaryEntry `json:"diaries"`
	Images        []exportImage      `json:"images"`
}

type exportDiaryEntry struct {
	ID        string  `json:"id"`
	ClientID  string  `json:"client_id"`
	BodyText  string  `json:"body_text"`
	Mood      *string `json:"mood"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type exportImage struct {
	ID           string  `json:"id"`
	ObjectKey    string  `json:"object_key"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Theme        string  `json:"theme"`
	CreatedAt    string  `json:"created_at"`
	URL          *string `json:"url,omitempty"`
	URLExpiresAt *string `json:"url_expires_at,omitempty"`
}

// UpdateMe handles PATCH /api/v1/me (update the caller's display name).
func (a *Account) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, a.logger)
		return
	}

	var req updateMeRequest
	if !a.decode(w, r, &req) {
		return
	}

	user, err := a.svc.UpdateDisplayName(r.Context(), userID, req.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDisplayNameRequired):
			writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
				[]FieldError{{Field: "display_name", Message: "must not be empty"}}, a.logger)
		case errors.Is(err, service.ErrDisplayNameTooLong):
			writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
				[]FieldError{{Field: "display_name", Message: "must be at most 50 characters"}}, a.logger)
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusUnauthorized, CodeTokenInvalid, "access token is invalid", nil, a.logger)
		default:
			a.internal(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user), a.logger)
}

// Export handles GET /api/v1/me/export (self-service data export as JSON).
func (a *Account) Export(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, a.logger)
		return
	}

	export, err := a.svc.Export(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, CodeTokenInvalid, "access token is invalid", nil, a.logger)
			return
		}
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toExportResponse(export), a.logger)
}

// DeleteMe handles DELETE /api/v1/me (permanent account deletion). Returns 204.
func (a *Account) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, a.logger)
		return
	}

	if err := a.svc.DeleteAccount(r.Context(), userID); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, CodeTokenInvalid, "access token is invalid", nil, a.logger)
			return
		}
		a.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil, a.logger)
}

// decode reads a JSON body into dst, rejecting unknown fields. It writes a 400 on
// failure and reports whether decoding succeeded.
func (a *Account) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, a.logger)
		return false
	}
	return true
}

func (a *Account) internal(w http.ResponseWriter, r *http.Request, err error) {
	a.logger.ErrorContext(r.Context(), "account handler internal error", slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, a.logger)
}

func toExportResponse(e service.AccountExport) exportResponse {
	out := exportResponse{
		ExportedAt:    e.ExportedAt.UTC().Format(time.RFC3339),
		SchemaVersion: exportSchemaVersion,
		User:          toUserResponse(e.User),
		Diaries:       make([]exportDiaryEntry, 0, len(e.Diaries)),
		Images:        make([]exportImage, 0, len(e.Images)),
	}
	for _, d := range e.Diaries {
		out.Diaries = append(out.Diaries, exportDiaryEntry{
			ID:        d.ID,
			ClientID:  d.ClientID,
			BodyText:  d.BodyText,
			Mood:      d.Mood,
			CreatedAt: d.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt: d.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	for _, img := range e.Images {
		item := exportImage{
			ID:        img.Image.ID,
			ObjectKey: img.Image.ObjectKey,
			Width:     img.Image.Width,
			Height:    img.Image.Height,
			Theme:     img.Image.Theme,
			CreatedAt: img.Image.CreatedAt.UTC().Format(time.RFC3339),
		}
		if img.URL != "" {
			url := img.URL
			exp := img.ExpiresAt.UTC().Format(time.RFC3339)
			item.URL = &url
			item.URLExpiresAt = &exp
		}
		out.Images = append(out.Images, item)
	}
	return out
}
