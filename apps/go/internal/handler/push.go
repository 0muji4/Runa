package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/0muji4/Runa/apps/go/internal/service"
)

// Push is the HTTP transport for the scheduler's reminder callback.
type Push struct {
	svc    *service.PushService
	logger *slog.Logger
}

// NewPush constructs the push handler from its service dependency.
func NewPush(svc *service.PushService, logger *slog.Logger) *Push {
	return &Push{svc: svc, logger: logger}
}

type reminderOutcomeResponse struct {
	Outcome string `json:"outcome"`
}

// Reminder handles POST /api/v1/hooks/push/reminder, the scheduled callback.
// 200 means "done, do not retry" whatever the outcome; 503 asks the scheduler
// to retry (provider or database hiccup).
func (p *Push) Reminder(w http.ResponseWriter, r *http.Request) {
	var task service.ReminderTask
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body", nil, p.logger)
		return
	}
	if task.DeviceID == "" || task.LocalDate == "" || task.NotifyTime == "" || task.TimeZone == "" {
		writeError(w, http.StatusBadRequest, CodeValidation, "device_id, local_date, notify_time and time_zone are required", nil, p.logger)
		return
	}

	outcome, err := p.svc.Deliver(r.Context(), task)
	if err != nil {
		if errors.Is(err, service.ErrPushTransient) {
			p.logger.WarnContext(r.Context(), "reminder delivery deferred", slog.String("device_id", task.DeviceID), slog.Any("error", err))
			writeError(w, http.StatusServiceUnavailable, CodeServiceUnavailable, "reminder delivery failed, retry later", nil, p.logger)
			return
		}
		p.logger.ErrorContext(r.Context(), "push handler internal error", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred", nil, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, reminderOutcomeResponse{Outcome: string(outcome)}, p.logger)
}

// Forbidden writes the 403 body for RequireCallbackToken (satisfies auth.ErrorResponder).
func (p *Push) Forbidden(w http.ResponseWriter, _ *http.Request, _ error) {
	writeError(w, http.StatusForbidden, CodeForbidden, "callback access forbidden", nil, p.logger)
}
