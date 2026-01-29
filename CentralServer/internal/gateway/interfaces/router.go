package interfaces

import (
	"central_server/internal/gateway/handler"
	"net/http"
)

type Router struct {
	mux            *http.ServeMux
	handler        *handler.LambdaHandler
	compatHandler  *handler.CompatHandler
	authMiddleware func(http.Handler) http.Handler
}

func NewRouter(h *handler.LambdaHandler,ch *handler.CompatHandler, authMiddleware func(http.Handler) http.Handler) *Router {
	return &Router{
		mux:            http.NewServeMux(),
		handler:        h,
		compatHandler:  ch,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) Setup() http.Handler {
	wrap := func(fn http.HandlerFunc) http.HandlerFunc {
		if r.authMiddleware == nil {
			return fn
		}
		return func(w http.ResponseWriter, req *http.Request) {
			r.authMiddleware(http.HandlerFunc(fn)).ServeHTTP(w, req)
		}
	}

	r.mux.HandleFunc("POST /api/v1/lambdas", wrap(r.handler.Store))
	r.mux.HandleFunc("GET /api/v1/lambdas/{id}", wrap(r.handler.Get))
	r.mux.HandleFunc("POST /api/v1/lambdas/trigger", wrap(r.handler.Trigger))
	r.mux.HandleFunc("POST /api/v1/lambdas/activate", wrap(r.handler.Activate))
	r.mux.HandleFunc("GET /api/v1/executions/{id}", wrap(r.handler.GetExecution))

	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
	return r.mux
}

r.mux.HandleFunc(
	"POST /functions/invoke",
	wrap(r.compatHandler.Invoke),
)

r.mux.HandleFunc(
	"POST /functions/create",
	wrap(r.compatHandler.Create),
)

r.mux.HandleFunc(
	"GET /functions/{id}",
	wrap(r.compatHandler.Get),
)

r.mux.HandleFunc(
	"GET /invocations",
	wrap(r.compatHandler.ListInvocations),
)
