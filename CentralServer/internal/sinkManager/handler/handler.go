package handler

import (
	"encoding/json"
	"net/http"

	"central_server/internal/sinkManager/domain"
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
		return domain.ErrInvalidSinkRequest
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func mapErrorToHTTPStatus(err error) int {
	switch err {
	case domain.ErrSinkNotFound:
		return http.StatusNotFound
	case domain.ErrSinkAlreadyExists:
		return http.StatusConflict
	case domain.ErrSinkUnreachable:
		return http.StatusServiceUnavailable
	case domain.ErrInvalidSinkRequest:
		return http.StatusBadRequest
	case domain.ErrInvalidCredentials:
		return http.StatusUnauthorized
	case domain.ErrTaskDeliveryFailed:
		return http.StatusServiceUnavailable
	case domain.ErrResultNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// --- Sink Handler ---

type SinkHandler struct {
	service domain.SinkManagerService
}

func NewSinkHandler(service domain.SinkManagerService) *SinkHandler {
	return &SinkHandler{service: service}
}

// Register handles sink registration requests (email/password)
func (h *SinkHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.SinkRegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.RegisterSink(r.Context(), &req)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login handles sink login requests (email/password)
func (h *SinkHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.SinkLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.LoginSink(r.Context(), &req)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Heartbeat handles heartbeat requests from sinks (every 10 seconds)
func (h *SinkHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req domain.HeartbeatRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.ProcessHeartbeat(r.Context(), &req)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// SubmitResult handles task result submissions from sinks
func (h *SinkHandler) SubmitResult(w http.ResponseWriter, r *http.Request) {
	var req domain.TaskResultRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.ProcessTaskResult(r.Context(), &req)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListSinks returns all registered sinks
func (h *SinkHandler) ListSinks(w http.ResponseWriter, r *http.Request) {
	sinks, err := h.service.ListSinks(r.Context())
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, sinks)
}

// GetSink returns a specific sink by ID
func (h *SinkHandler) GetSink(w http.ResponseWriter, r *http.Request) {
	sinkID := r.PathValue("id")
	if sinkID == "" {
		writeError(w, http.StatusBadRequest, "Sink ID is required", "")
		return
	}

	sink, err := h.service.GetSink(r.Context(), sinkID)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, sink)
}

// UnregisterSink removes a sink from the registry
func (h *SinkHandler) UnregisterSink(w http.ResponseWriter, r *http.Request) {
	sinkID := r.PathValue("id")
	if sinkID == "" {
		writeError(w, http.StatusBadRequest, "Sink ID is required", "")
		return
	}

	if err := h.service.UnregisterSink(r.Context(), sinkID); err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Sink unregistered successfully",
	})
}

// GetTaskResult returns the result of a specific task
func (h *SinkHandler) GetTaskResult(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "Task ID is required", "")
		return
	}

	result, err := h.service.GetTaskResult(r.Context(), taskID)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
