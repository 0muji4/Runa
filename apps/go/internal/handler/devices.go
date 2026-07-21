package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/0muji4/Runa/apps/go/internal/auth"
	"github.com/0muji4/Runa/apps/go/internal/repository"
	"github.com/0muji4/Runa/apps/go/internal/service"
)

// notifyTimePattern matches a 24-hour local reminder time "HH:MM".
var notifyTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// Devices is the HTTP transport for the device-registration endpoint. Like the
// other handlers it only translates requests/responses; the logic lives in the
// service layer.
type Devices struct {
	svc    *service.DeviceService
	logger *slog.Logger
}

// NewDevices constructs the devices handler from its service dependency.
func NewDevices(svc *service.DeviceService, logger *slog.Logger) *Devices {
	return &Devices{svc: svc, logger: logger}
}

type registerDeviceRequest struct {
	PushToken  string `json:"push_token"`
	Platform   string `json:"platform"`
	NotifyTime string `json:"notify_time"`
	Enabled    bool   `json:"enabled"`
}

type deviceResponse struct {
	ID         string `json:"id"`
	PushToken  string `json:"push_token"`
	Platform   string `json:"platform"`
	NotifyTime string `json:"notify_time"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// Register handles PUT /api/v1/devices — idempotently registers the caller's push
// token plus reminder preference (upsert by user + token).
func (d *Devices) Register(w http.ResponseWriter, r *http.Request) {
	userID, ok := d.userID(w, r)
	if !ok {
		return
	}
	var req registerDeviceRequest
	if !d.decode(w, r, &req) {
		return
	}
	if details := validateRegisterDevice(req); len(details) > 0 {
		writeError(w, http.StatusBadRequest, CodeValidation, "validation failed", details, d.logger)
		return
	}

	device, err := d.svc.Register(r.Context(), userID, service.RegisterDeviceInput{
		PushToken:  req.PushToken,
		Platform:   req.Platform,
		NotifyTime: req.NotifyTime,
		Enabled:    req.Enabled,
	})
	if err != nil {
		d.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDeviceResponse(device), d.logger)
}

func (d *Devices) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, CodeUnauthorized, "authentication required", nil, d.logger)
	}
	return id, ok
}

func (d *Devices) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, d.logger)
		return false
	}
	return true
}

func (d *Devices) internal(w http.ResponseWriter, r *http.Request, err error) {
	d.logger.ErrorContext(r.Context(), "devices handler internal error", slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, d.logger)
}

func validateRegisterDevice(req registerDeviceRequest) []FieldError {
	var details []FieldError
	if strings.TrimSpace(req.PushToken) == "" {
		details = append(details, FieldError{Field: "push_token", Message: "is required"})
	}
	if req.Platform != "ios" && req.Platform != "android" {
		details = append(details, FieldError{Field: "platform", Message: "must be one of ios|android"})
	}
	if !notifyTimePattern.MatchString(req.NotifyTime) {
		details = append(details, FieldError{Field: "notify_time", Message: "must be a 24-hour time HH:MM"})
	}
	return details
}

func toDeviceResponse(d repository.Device) deviceResponse {
	return deviceResponse{
		ID:         d.ID,
		PushToken:  d.PushToken,
		Platform:   d.Platform,
		NotifyTime: d.NotifyTime,
		Enabled:    d.Enabled,
		CreatedAt:  d.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:  d.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
