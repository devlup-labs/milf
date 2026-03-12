package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	authdomain "central_server/internal/auth/domain"
	"central_server/internal/gateway/domain"
)

// This struct just reuses existing LambdaService
type CompatHandler struct {
	lambdaService domain.LambdaService
}

// Constructor
func NewCompatHandler(svc domain.LambdaService) *CompatHandler {
	return &CompatHandler{lambdaService: svc}
}

// This matches mockApi.functions.invoke()
func (h *CompatHandler) Invoke(w http.ResponseWriter, r *http.Request) {
	// What the client sends — input is already a TaskEnvelope: {type, data}
	var req struct {
		ID    string      `json:"id"`
		Input interface{} `json:"input"` // accepts string OR object
	}

	// Read JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Normalise input: client may send parsed JSON object or a string
	var inputMap map[string]interface{}
	switch v := req.Input.(type) {
	case map[string]interface{}:
		inputMap = v
	case string:
		// Try to unwrap JSON-string → map
		if err := json.Unmarshal([]byte(v), &inputMap); err != nil {
			inputMap = map[string]interface{}{"type": "json", "data": v}
		}
	default:
		inputMap = map[string]interface{}{"type": "null"}
	}

	// Call existing Gateway logic
	resp, err := h.lambdaService.TriggerLambda(
		r.Context(),
		&domain.LambdaExecRequest{
			ReferenceID: req.ID,
			Input:       inputMap,
		},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Return execution_id so the frontend can poll for the result
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":           true,
		"execution_id": resp.ExecutionID,
		"acknowledged":  resp.ExecutionID != "",
		"status":       string(resp.Status),
	})
}

func (h *CompatHandler) Create(w http.ResponseWriter, r *http.Request) {
	// What the client sends (simplified mockApi shape)
	var req struct {
		Name       string                 `json:"name"`
		Runtime    string                 `json:"runtime"`
		Memory     int                    `json:"memory"`
		SourceCode string                 `json:"sourceCode"`
		MetaData   map[string]string      `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Translate → Gateway store request
	userID, ok := authdomain.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	storeReq := &domain.LambdaStoreRequest{
		UserID:     userID,
		FuncID:     req.Name,
		SourceCode: []byte(req.SourceCode),
		Runtime:    domain.RuntimeEnvironment(req.Runtime),
		MemoryMB:   req.Memory,
		RunType:    domain.RunTypeOnCommand,
		MetaData:   req.MetaData,
	}

	resp, err := h.lambdaService.StoreLambda(r.Context(), storeReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-activate the function after creation
	// Give compiler a moment to process (in production, this should poll or use events)
	time.Sleep(2 * time.Second)
	_, activateErr := h.lambdaService.ActivateLambda(r.Context(), &domain.LambdaExecRequest{
		ReferenceID: req.Name,
	})
	if activateErr != nil {
		// Log but don't fail - user can manually activate later
		fmt.Printf("[Gateway] Warning: Auto-activation failed for %s: %v\n", req.Name, activateErr)
	}

	// Return what the old client expects
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        resp.ID,
		"name":      resp.Name,
		"wasmRef":   resp.WasmRef,
		"createdAt": time.Now().UTC(),
		"updatedAt": time.Now().UTC(),
	})
}

func (h *CompatHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Read function ID from URL
	funcID := r.PathValue("id")
	if funcID == "" {
		http.Error(w, "function id required", http.StatusBadRequest)
		return
	}

	// Call existing Gateway logic
	lambda, err := h.lambdaService.GetLambda(r.Context(), funcID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return in client-friendly shape
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        lambda.ID,
		"name":      lambda.Name,
		"runtime":   lambda.Runtime,
		"memory":    lambda.MemoryMB,
		"createdAt": lambda.CreatedAt,
		"updatedAt": lambda.UpdatedAt,
		"wasmRef":   lambda.WasmRef,
		"runType":   lambda.RunType,
	})
}

func (h *CompatHandler) ListInvocations(w http.ResponseWriter, r *http.Request) {
	// Client may pass execution id as query or path (optional)
	execID := r.URL.Query().Get("execution_id")
	
	if execID != "" {
		// Return specific execution
		exec, err := h.lambdaService.GetExecution(r.Context(), execID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":         exec.ID,
				"functionId": exec.LambdaID,
				"status":     exec.Status,
				"output":     exec.Output,
				"error":      exec.Error,
				"startedAt":  exec.StartedAt,
				"finishedAt": exec.FinishedAt,
			},
		})
		return
	}

	// Client may pass function name to filter by
	functionName := r.URL.Query().Get("q")
	
	userID, _ := authdomain.UserIDFromContext(r.Context())
	executions, err := h.lambdaService.ListExecutions(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by function name if provided
	if functionName != "" {
		var filtered []*domain.Execution
		for _, exec := range executions {
			if exec.LambdaID == functionName || exec.ReferenceID == functionName {
				filtered = append(filtered, exec)
			}
		}
		executions = filtered
	}

	// Map to client format
	result := []map[string]interface{}{}
	for _, exec := range executions {
		result = append(result, map[string]interface{}{
			"id":         exec.ID,
			"functionId": exec.LambdaID,
			"status":     exec.Status,
			"output":     exec.Output,
			"error":      exec.Error,
			"startedAt":  exec.StartedAt,
			"finishedAt": exec.FinishedAt,
		})
	}

	json.NewEncoder(w).Encode(result)
}

