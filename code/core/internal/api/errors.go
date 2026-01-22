package api

import (
	"encoding/json"
	"net/http"
)

// Error represents a structured error response.
type Error struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Common error codes.
const (
	ErrCodeBadRequest     = "bad_request"
	ErrCodeNotFound       = "not_found"
	ErrCodeUnauthorized   = "unauthorised"
	ErrCodeForbidden      = "forbidden"
	ErrCodeConflict       = "conflict"
	ErrCodeInternal       = "internal_error"
	ErrCodeValidation     = "validation_error"
	ErrCodeMethodNotAllow = "method_not_allowed"
)

// writeJSON writes a JSON response with the given status code and payload.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		//nolint:errcheck // Best-effort write to response; connection may be closed
		json.NewEncoder(w).Encode(v)
	}
}

// writeError writes a structured error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, Error{
		Status:  status,
		Code:    code,
		Message: message,
	})
}

// writeBadRequest writes a 400 error response.
func writeBadRequest(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, ErrCodeBadRequest, message)
}

// writeNotFound writes a 404 error response.
func writeNotFound(w http.ResponseWriter, message string) {
	writeError(w, http.StatusNotFound, ErrCodeNotFound, message)
}

// writeUnauthorized writes a 401 error response.
func writeUnauthorized(w http.ResponseWriter, message string) {
	writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// writeInternalError writes a 500 error response.
func writeInternalError(w http.ResponseWriter, message string) {
	writeError(w, http.StatusInternalServerError, ErrCodeInternal, message)
}
