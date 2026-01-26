package handler

import (
	"encoding/json"
	"net/http"

	"central_server/internal/gateway/domain"
)

// --- HTTP Utilities ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message, details string) {
	writeJSON(w, status, domain.ErrorResponse{
		Code:    status,
		Message: message,
		Details: details,
	})
}

func decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return domain.ErrInvalidRequest
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

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

// --- Lambda Handler ---

type LambdaHandler struct {
	service domain.LambdaService
}

func NewLambdaHandler(service domain.LambdaService) *LambdaHandler {
	return &LambdaHandler{service: service}
}

func (h *LambdaHandler) Store(w http.ResponseWriter, r *http.Request) {
	var req domain.LambdaStoreRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.StoreLambda(r.Context(), &req)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *LambdaHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	var req domain.LambdaExecRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.TriggerLambda(r.Context(), &req)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

func (h *LambdaHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var req domain.LambdaExecRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.ActivateLambda(r.Context(), &req)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}


func (h *LambdaHandler) Get(w http.ResponseWriter, r *http.Request) {
	lambdaID := r.PathValue("id")
	if lambdaID == "" {
		writeError(w, http.StatusBadRequest, "Lambda ID is required", "")
		return
	}

	lambda, err := h.service.GetLambda(r.Context(), lambdaID)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, lambda)
}

func (h *LambdaHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	executionID := r.PathValue("id")
	if executionID == "" {
		writeError(w, http.StatusBadRequest, "Execution ID is required", "")
		return
	}

	execution, err := h.service.GetExecution(r.Context(), executionID)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, execution)
}
