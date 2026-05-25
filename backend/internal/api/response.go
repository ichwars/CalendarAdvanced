package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"calendaradvanced/internal/application"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	resp := errorResponse{Code: "internal_error", Message: "An internal error occurred."}
	var coded application.CodedError
	if errors.As(err, &coded) {
		status = http.StatusBadRequest
		resp = errorResponse{Code: coded.Code, Message: coded.Message, Details: coded.Details}
	} else {
		switch {
		case errors.Is(err, application.ErrUnauthorized):
			status = http.StatusUnauthorized
			resp = errorResponse{Code: "unauthorized", Message: "Authentication is required."}
		case errors.Is(err, application.ErrForbidden):
			status = http.StatusForbidden
			resp = errorResponse{Code: "forbidden", Message: "The operation is not allowed."}
		case errors.Is(err, application.ErrRateLimited):
			status = http.StatusTooManyRequests
			resp = errorResponse{Code: "rate_limited", Message: "Too many requests. Try again later."}
		case errors.Is(err, application.ErrSetupNotRequired):
			status = http.StatusConflict
			resp = errorResponse{Code: "setup_not_required", Message: "Setup is already completed."}
		case errors.Is(err, application.ErrTwoFactorNeeded):
			status = http.StatusUnauthorized
			resp = errorResponse{Code: "two_factor_required", Message: "Two-factor authentication is required."}
		case errors.Is(err, sqlite.ErrNotFound), errors.Is(err, application.ErrNotFound):
			status = http.StatusNotFound
			resp = errorResponse{Code: "not_found", Message: "The requested resource was not found."}
		case errors.Is(err, application.ErrValidation):
			status = http.StatusBadRequest
			resp = errorResponse{Code: "validation_failed", Message: "The request is invalid."}
		case err.Error() == "unsupported_content_type":
			status = http.StatusUnsupportedMediaType
			resp = errorResponse{Code: "unsupported_content_type", Message: "Use application/json for write requests."}
		}
	}
	writeJSON(w, status, resp)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
