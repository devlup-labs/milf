package interfaces

import (
	"central_server/internal/sinkManager/handler"
	"net/http"
)

type Router struct {
	mux            *http.ServeMux
	handler        *handler.SinkHandler
	authMiddleware func(http.Handler) http.Handler
}

func NewRouter(h *handler.SinkHandler, authMiddleware func(http.Handler) http.Handler) *Router {
	return &Router{
		mux:            http.NewServeMux(),
		handler:        h,
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

	r.mux.HandleFunc("POST /api/v1/sinks/register", r.handler.Register)
	r.mux.HandleFunc("POST /api/v1/sinks/login", r.handler.Login)
	r.mux.HandleFunc("POST /api/v1/sinks/heartbeat", r.handler.Heartbeat)
	r.mux.HandleFunc("POST /api/v1/sinks/result", r.handler.SubmitResult)
	r.mux.HandleFunc("GET /api/v1/sinks", wrap(r.handler.ListSinks))
	r.mux.HandleFunc("GET /api/v1/sinks/{id}", wrap(r.handler.GetSink))
	r.mux.HandleFunc("DELETE /api/v1/sinks/{id}", wrap(r.handler.UnregisterSink))
	r.mux.HandleFunc("GET /api/v1/tasks/{id}/result", wrap(r.handler.GetTaskResult))

	return r.mux
}
