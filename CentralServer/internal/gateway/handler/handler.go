package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	authdomain "central_server/internal/auth/domain"
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

	// Return in snake_case so the frontend polling hook can read status/output correctly
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":          execution.ID,
		"lambda_id":   execution.LambdaID,
		"status":      string(execution.Status),
		"output":      execution.Output,
		"error":       execution.Error,
		"started_at":  execution.StartedAt,
		"finished_at": execution.FinishedAt,
	})
}

func (h *LambdaHandler) GetWasm(w http.ResponseWriter, r *http.Request) {
	lambdaID := r.PathValue("id")
	if lambdaID == "" {
		writeError(w, http.StatusBadRequest, "Lambda ID is required", "")
		return
	}

	// FALLBACK FOR LOCAL TESTING:
	if lambdaID == "test-local-addd" {
		log.Printf("[Handler] 📁 Reading local test file for lambdaID: %s", lambdaID)
		wasmBytes, err := os.ReadFile("/Users/adarsh/Projects/devlup/milf/consumeronlywamr/test/addd.wasm")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read local WASM file", err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/wasm")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.wasm\"", lambdaID))
		w.WriteHeader(http.StatusOK)
		w.Write(wasmBytes)
		return
	}

	lambda, err := h.service.GetLambda(r.Context(), lambdaID)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	if len(lambda.WasmRef) == 0 {
		writeError(w, http.StatusNotFound, "WASM binary not found or not compiled yet", "")
		return
	}

	// WasmRef is stored as base64 (BYTEA read from Postgres) — decode to raw bytes.
	wasmBytes, decErr := base64.StdEncoding.DecodeString(lambda.WasmRef)
	if decErr != nil {
		writeError(w, http.StatusInternalServerError, "Failed to decode WASM binary", decErr.Error())
		return
	}

	w.Header().Set("Content-Type", "application/wasm")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.wasm\"", lambda.Name))
	w.WriteHeader(http.StatusOK)
	w.Write(wasmBytes)
}

func (h *LambdaHandler) Execute(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Handler] 🚀 Execute hit for path: %s", r.URL.Path)
	lambdaID := r.PathValue("id")
	if lambdaID == "" {
		writeError(w, http.StatusBadRequest, "Lambda ID is required", "")
		return
	}

	// Read body
	var body interface{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		// If not JSON, treat as raw string if body is present
		body = nil 
	}

	// AWS Lambda Parity: Inject HTTP Context
	// This allows the WASM function to see headers, query params, etc.
	input := map[string]interface{}{
		"_http_context": map[string]interface{}{
			"method":  r.Method,
			"path":    r.URL.Path,
			"headers": r.Header,
			"query":   r.URL.Query(),
		},
		"body": body,
	}

	// Check for invocation type header
	invocationType := r.Header.Get("X-Amz-Invocation-Type")
	if invocationType == "" {
		invocationType = "RequestResponse" // Default to sync
	}

	inputBytes, _ := json.Marshal(input)
	
	if invocationType == "Event" {
		// Async execution (AWS Lambda 'Event' type)
		ack, err := h.service.ExecuteJob(r.Context(), lambdaID, string(inputBytes))
		if err != nil {
			writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"status":       "Execution requested",
			"acknowledged": ack,
		})
		return
	}

	// Sync execution (AWS Lambda 'RequestResponse' type)
	resp, err := h.service.TriggerLambda(r.Context(), &domain.LambdaExecRequest{
		ReferenceID: lambdaID,
		Input:       input,
	})
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	// For now, TriggerLambda is async. We will soon upgrade it to wait for result.
	// But let's keep the DTO consistent.
	writeJSON(w, http.StatusOK, resp)
}

// List all lambdas for the authenticated user
func (h *LambdaHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := authdomain.UserIDFromContext(r.Context())
	if !ok {
		userID = ""
	}

	lambdas, err := h.service.ListLambdas(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, lambdas)
}

// Delete a lambda function
func (h *LambdaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	lambdaID := r.PathValue("id")
	if lambdaID == "" {
		writeError(w, http.StatusBadRequest, "Lambda ID is required", "")
		return
	}

	err := h.service.DeleteLambda(r.Context(), lambdaID)
	if err != nil {
		writeError(w, mapErrorToHTTPStatus(err), err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Lambda deleted",
		"id":      lambdaID,
	})
}
