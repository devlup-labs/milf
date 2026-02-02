package handler

import (
	"encoding/json"
<<<<<<< HEAD
	"net/http"	
=======
	"net/http"
>>>>>>> 16d7b57203b122e0bcce904fb6466fbaf28fa986
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
	// What the client sends
	var req struct {
		ID    string                 `json:"id"`
		Input map[string]interface{} `json:"input"`
	}

	// Read JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Call existing Gateway logic
	resp, err := h.lambdaService.TriggerLambda(
		r.Context(),
		&domain.LambdaExecRequest{
			ReferenceID: req.ID,
			Input:       req.Input,
		},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return what the old client expects
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"requestId": resp.ExecutionID,
	})
}

func (h *CompatHandler) Create(w http.ResponseWriter, r *http.Request) {
	// What the client sends (simplified mockApi shape)
	var req struct {
<<<<<<< HEAD
		Name       string                 `json:"name"`
		Runtime    string                 `json:"runtime"`
		Memory     int                    `json:"memory"`
		SourceCode string                 `json:"sourceCode"`
		MetaData   map[string]string      `json:"metadata,omitempty"`
=======
		Name       string            `json:"name"`
		Runtime    string            `json:"runtime"`
		Memory     int               `json:"memory"`
		SourceCode []byte            `json:"sourceCode"`
		MetaData   map[string]string `json:"metadata,omitempty"`
>>>>>>> 16d7b57203b122e0bcce904fb6466fbaf28fa986
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
	// Client may pass execution id as query or path
	execID := r.URL.Query().Get("execution_id")
	if execID == "" {
		http.Error(w, "execution_id is required", http.StatusBadRequest)
		return
	}

	// Call real Gateway logic
	exec, err := h.lambdaService.GetExecution(r.Context(), execID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Client expects a LIST, so we wrap it in an array
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
}
