package handler

import (
	"encoding/json"
	"net/http"

	"central_server/internal/sinkManager/domain"
	"github.com/gorilla/websocket"
	"log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for the execution nodes
	},
}

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

type SinkHandler struct {
	service domain.SinkManagerService
}

func NewSinkHandler(service domain.SinkManagerService) *SinkHandler {
	return &SinkHandler{service: service}
}

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

func (h *SinkHandler) ListSinks(w http.ResponseWriter, r *http.Request) {
	sinks, err := h.service.ListSinks(r.Context())
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, sinks)
}

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

func (h *SinkHandler) ConnectWebSocket(w http.ResponseWriter, r *http.Request) {
	sinkID := r.URL.Query().Get("sinkId")
	if sinkID == "" {
		writeError(w, http.StatusBadRequest, "Missing sinkId", "")
		return
	}

	// 1. Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %v", err)
		return
	}
	defer conn.Close()

	// 2. Register connection
	h.service.RegisterWebSocket(sinkID, conn)
	defer h.service.UnregisterWebSocket(sinkID)

	log.Printf("WebSocket connected for sink: %s", sinkID)

	// 3. Listen for incoming messages (Results, Heartbeats)
	for {
		var msg domain.WebSocketMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket closed/error for sink %s: %v", sinkID, err)
			break
		}

		switch msg.Type {
		case domain.MsgHeartbeat:
			// Process heartbeat via existing service method by converting payload if needed.
			// To keep it simple, we can just mark the sink online manually or parse payload.
			log.Printf("Received WS Heartbeat from %s", sinkID)
			
		case domain.MsgTaskResult:
			// Parse the payload into a TaskResultRequest
			payloadBytes, _ := json.Marshal(msg.Payload)
			var resultReq domain.TaskResultRequest
			if err := json.Unmarshal(payloadBytes, &resultReq); err == nil {
				_, _ = h.service.ProcessTaskResult(r.Context(), &resultReq)
				log.Printf("Received WS TaskResult for %s", resultReq.ExecutionID)
			}
		}
	}
}
