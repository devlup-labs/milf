package interfaces

import (
	"net/http"

	"central_server/internal/gateway/domain"
)

// LambdaHandler handles HTTP requests for lambda operations
type LambdaHandler struct {
	service LambdaServiceInterface
}

// NewLambdaHandler creates a new LambdaHandler with the provided service
func NewLambdaHandler(service LambdaServiceInterface) *LambdaHandler {
	return &LambdaHandler{
		service: service,
	}
}

// Store handles POST /lambdas - stores a new lambda function
func (h *LambdaHandler) Store(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "Use POST method")
		return
	}

	var req domain.LambdaStoreRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.StoreLambda(r.Context(), &req)
	if err != nil {
		status := mapErrorToHTTPStatus(err)
		writeError(w, status, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Execute handles POST /lambdas/execute - executes a stored lambda
func (h *LambdaHandler) Execute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "Use POST method")
		return
	}

	var req domain.LambdaExecRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	resp, err := h.service.ExecuteLambda(r.Context(), &req)
	if err != nil {
		status := mapErrorToHTTPStatus(err)
		writeError(w, status, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// Get handles GET /lambdas/{id} - retrieves a lambda by ID
func (h *LambdaHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "Use GET method")
		return
	}

	lambdaID := r.PathValue("id")
	if lambdaID == "" {
		writeError(w, http.StatusBadRequest, "Lambda ID is required", "")
		return
	}

	lambda, err := h.service.GetLambda(r.Context(), lambdaID)
	if err != nil {
		status := mapErrorToHTTPStatus(err)
		writeError(w, status, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, lambda)
}

// GetExecution handles GET /executions/{id} - retrieves an execution by ID
func (h *LambdaHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "Use GET method")
		return
	}

	executionID := r.PathValue("id")
	if executionID == "" {
		writeError(w, http.StatusBadRequest, "Execution ID is required", "")
		return
	}

	execution, err := h.service.GetExecution(r.Context(), executionID)
	if err != nil {
		status := mapErrorToHTTPStatus(err)
		writeError(w, status, err.Error(), "")
		return
	}

	writeJSON(w, http.StatusOK, execution)
}
