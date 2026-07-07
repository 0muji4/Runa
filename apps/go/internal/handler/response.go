package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorCode is a stable, machine-readable error identifier shared by every
// feature endpoint. Clients switch on these codes rather than HTTP status alone,
// so the set is part of the API contract (see api/openapi.yaml).
type ErrorCode string

const (
	CodeValidation           ErrorCode = "validation_error"
	CodeInvalidCredentials   ErrorCode = "invalid_credentials"
	CodeEmailTaken           ErrorCode = "email_taken"
	CodeUnauthorized         ErrorCode = "unauthorized"
	CodeTokenExpired         ErrorCode = "token_expired"
	CodeTokenInvalid         ErrorCode = "token_invalid"
	CodeProviderVerification ErrorCode = "provider_verification_failed"
	CodeRateLimited          ErrorCode = "rate_limited"
	CodeInternal             ErrorCode = "internal_error"
)

// FieldError describes one field that failed validation.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// errorEnvelope is the single error shape returned by all endpoints:
// {"error": {"code": "...", "message": "...", "details": [...]}}.
type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    ErrorCode    `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// writeJSON writes v as a JSON response with the given status. A nil v writes
// only the status line (used for 204 No Content).
func writeJSON(w http.ResponseWriter, status int, v any, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil && logger != nil {
		// Response is already committed; only logging remains.
		logger.Error("failed to encode response", slog.Any("error", err))
	}
}

// writeError writes the shared error envelope.
func writeError(w http.ResponseWriter, status int, code ErrorCode, message string, details []FieldError, logger *slog.Logger) {
	writeJSON(w, status, errorEnvelope{Error: errorPayload{Code: code, Message: message, Details: details}}, logger)
}
