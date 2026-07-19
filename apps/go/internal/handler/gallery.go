package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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

// Gallery is the HTTP transport for the gallery endpoints. Like the other
// handlers it only translates requests/responses and maps service errors to the
// shared envelope; the logic lives in the service layer.
type Gallery struct {
	svc    *service.GalleryService
	logger *slog.Logger
}

// NewGallery constructs the gallery handler from its service dependency.
func NewGallery(svc *service.GalleryService, logger *slog.Logger) *Gallery {
	return &Gallery{svc: svc, logger: logger}
}

type uploadURLRequest struct {
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type uploadURLResponse struct {
	ObjectKey string            `json:"object_key"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt string            `json:"expires_at"`
	MaxSize   int64             `json:"max_size"`
}

type createGalleryRequest struct {
	ObjectKey string `json:"object_key"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Theme     string `json:"theme"`
}

type galleryImageResponse struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	URLExpiresAt string `json:"url_expires_at"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Theme        string `json:"theme"`
	CreatedAt    string `json:"created_at"`
}

type galleryListResponse struct {
	Items      []galleryImageResponse `json:"items"`
	NextCursor *string                `json:"next_cursor"`
}

// UploadURL handles POST /api/v1/gallery/upload-url — issues a presigned PUT URL
// plus the object_key the client registers after uploading.
func (g *Gallery) UploadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := g.userID(w, r)
	if !ok {
		return
	}
	var req uploadURLRequest
	if !g.decode(w, r, &req) {
		return
	}
	if details := validateUploadURL(req); len(details) > 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed", details, g.logger)
		return
	}

	target, err := g.svc.CreateUploadURL(r.Context(), userID, req.ContentType, req.Size)
	if err != nil {
		g.writeGalleryError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, uploadURLResponse{
		ObjectKey: target.ObjectKey,
		UploadURL: target.URL,
		Method:    http.MethodPut,
		Headers:   map[string]string{"Content-Type": target.ContentType},
		ExpiresAt: target.ExpiresAt.UTC().Format(time.RFC3339Nano),
		MaxSize:   target.MaxSize,
	}, g.logger)
}

// Create handles POST /api/v1/gallery — registers metadata after upload.
func (g *Gallery) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := g.userID(w, r)
	if !ok {
		return
	}
	var req createGalleryRequest
	if !g.decode(w, r, &req) {
		return
	}
	if details := validateCreateGallery(req); len(details) > 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed", details, g.logger)
		return
	}

	view, err := g.svc.RegisterImage(r.Context(), userID, req.ObjectKey, req.Width, req.Height, req.Theme)
	if err != nil {
		g.writeGalleryError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toGalleryImageResponse(view), g.logger)
}

// List handles GET /api/v1/gallery?limit=&cursor= (newest first, keyset paging).
func (g *Gallery) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := g.userID(w, r)
	if !ok {
		return
	}
	limit, ok := g.parseLimit(w, r)
	if !ok {
		return
	}
	cursor, ok := g.parseCursor(w, r)
	if !ok {
		return
	}

	page, err := g.svc.List(r.Context(), userID, limit, cursor)
	if err != nil {
		g.writeGalleryError(w, r, err)
		return
	}

	resp := galleryListResponse{Items: toGalleryImageResponses(page.Items)}
	if page.NextCursor != nil {
		c := encodeGalleryCursor(*page.NextCursor)
		resp.NextCursor = &c
	}
	writeJSON(w, http.StatusOK, resp, g.logger)
}

// Get handles GET /api/v1/gallery/{id}.
func (g *Gallery) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := g.userID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !uuidPattern.MatchString(id) {
		g.notFound(w)
		return
	}

	view, err := g.svc.Get(r.Context(), userID, id)
	if err != nil {
		g.writeGalleryError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toGalleryImageResponse(view), g.logger)
}

// Delete handles DELETE /api/v1/gallery/{id} (soft delete; object removed async).
func (g *Gallery) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := g.userID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !uuidPattern.MatchString(id) {
		g.notFound(w)
		return
	}

	if err := g.svc.Delete(r.Context(), userID, id); err != nil {
		g.writeGalleryError(w, r, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil, g.logger)
}

func (g *Gallery) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, g.logger)
	}
	return id, ok
}

func (g *Gallery) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, g.logger)
		return false
	}
	return true
}

func (g *Gallery) parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0, true // service applies the default
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "limit", Message: "must be a non-negative integer"}}, g.logger)
		return 0, false
	}
	return limit, true
}

func (g *Gallery) parseCursor(w http.ResponseWriter, r *http.Request) (*repository.GalleryCursor, bool) {
	raw := r.URL.Query().Get("cursor")
	if raw == "" {
		return nil, true
	}
	cursor, err := decodeGalleryCursor(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "cursor", Message: "is malformed"}}, g.logger)
		return nil, false
	}
	return cursor, true
}

func (g *Gallery) writeGalleryError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrGalleryNotFound), errors.Is(err, service.ErrInvalidObjectKey):
		// Both collapse to 404: a stranger's id and a key outside the caller's
		// namespace must be indistinguishable from "does not exist".
		g.notFound(w)
	case errors.Is(err, service.ErrObjectMissing):
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "object_key", Message: "object has not been uploaded"}}, g.logger)
	case errors.Is(err, service.ErrUploadTooLarge):
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "size", Message: "exceeds the maximum upload size"}}, g.logger)
	case errors.Is(err, service.ErrContentTypeNotAllowed):
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed",
			[]FieldError{{Field: "content_type", Message: "is not an allowed image type"}}, g.logger)
	case errors.Is(err, service.ErrStorageUnavailable):
		writeError(w, http.StatusServiceUnavailable, CodeServiceUnavailable, "image storage is unavailable", nil, g.logger)
	default:
		g.internal(w, r, err)
	}
}

func (g *Gallery) notFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, CodeNotFound, "gallery image not found", nil, g.logger)
}

func (g *Gallery) internal(w http.ResponseWriter, r *http.Request, err error) {
	g.logger.ErrorContext(r.Context(), "gallery handler internal error", slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, g.logger)
}

func validateUploadURL(req uploadURLRequest) []FieldError {
	var details []FieldError
	if strings.TrimSpace(req.ContentType) == "" {
		details = append(details, FieldError{Field: "content_type", Message: "is required"})
	}
	if req.Size <= 0 {
		details = append(details, FieldError{Field: "size", Message: "must be a positive integer"})
	}
	return details
}

func validateCreateGallery(req createGalleryRequest) []FieldError {
	var details []FieldError
	if strings.TrimSpace(req.ObjectKey) == "" {
		details = append(details, FieldError{Field: "object_key", Message: "is required"})
	}
	if req.Width <= 0 {
		details = append(details, FieldError{Field: "width", Message: "must be a positive integer"})
	}
	if req.Height <= 0 {
		details = append(details, FieldError{Field: "height", Message: "must be a positive integer"})
	}
	if req.Theme != "monotone" && req.Theme != "pink" {
		details = append(details, FieldError{Field: "theme", Message: "must be one of monotone|pink"})
	}
	return details
}

// encodeGalleryCursor packs a keyset boundary into an opaque base64url token of
// "<createdAt RFC3339Nano>|<id>". Clients treat it as opaque and echo it back.
func encodeGalleryCursor(c repository.GalleryCursor) string {
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeGalleryCursor(s string) (*repository.GalleryCursor, error) {
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
	return &repository.GalleryCursor{CreatedAt: createdAt, ID: id}, nil
}

func toGalleryImageResponses(views []service.ImageView) []galleryImageResponse {
	out := make([]galleryImageResponse, 0, len(views))
	for _, v := range views {
		out = append(out, toGalleryImageResponse(v))
	}
	return out
}

func toGalleryImageResponse(v service.ImageView) galleryImageResponse {
	return galleryImageResponse{
		ID:           v.Image.ID,
		URL:          v.ViewURL,
		URLExpiresAt: v.ExpiresAt.UTC().Format(time.RFC3339Nano),
		Width:        v.Image.Width,
		Height:       v.Image.Height,
		Theme:        v.Image.Theme,
		CreatedAt:    v.Image.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}
