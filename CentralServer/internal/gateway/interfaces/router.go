package interfaces

import (
	"net/http"
)

// Router sets up the HTTP routes for the gateway module
type Router struct {
	mux     *http.ServeMux
	handler *LambdaHandler
}

// NewRouter creates a new Router with the provided handler
func NewRouter(handler *LambdaHandler) *Router {
	return &Router{
		mux:     http.NewServeMux(),
		handler: handler,
	}
}

// Setup configures all routes and returns the http.Handler
func (r *Router) Setup() http.Handler {
	// Lambda endpoints
	r.mux.HandleFunc("POST /api/v1/lambdas", r.handler.Store)
	r.mux.HandleFunc("GET /api/v1/lambdas/{id}", r.handler.Get)
	r.mux.HandleFunc("POST /api/v1/lambdas/execute", r.handler.Execute)

	// Execution endpoints
	r.mux.HandleFunc("GET /api/v1/executions/{id}", r.handler.GetExecution)

	// Health check
	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	return r.mux
}
