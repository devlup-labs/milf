package interfaces

import (
	"encoding/json"
	"net/http"

	"central_server/internal/gateway/domain"
)

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string, details string) {
	writeJSON(w, status, domain.ErrorResponse{
		Code:    status,
		Message: message,
		Details: details,
	})
}

// decodeJSON decodes a JSON request body into the provided struct
func decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return domain.ErrInvalidRequest
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// mapErrorToHTTPStatus maps domain errors to HTTP status codes
func mapErrorToHTTPStatus(err error) int {
	switch err {
	case domain.ErrLambdaNotFound:
		return http.StatusNotFound
	case domain.ErrInvalidRuntime, domain.ErrInvalidRunType, domain.ErrInvalidRequest:
		return http.StatusBadRequest
	case domain.ErrCompilationFailed:
		return http.StatusUnprocessableEntity
	case domain.ErrExecutionFailed:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
