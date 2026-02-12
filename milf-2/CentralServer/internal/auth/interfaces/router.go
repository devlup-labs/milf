package interfaces

import (
	"net/http"
)

type AuthHandler interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	AuthMiddleware(next http.Handler) http.Handler
}
type Router struct {
	mux     *http.ServeMux
	handler AuthHandler
}

func NewRouter(h AuthHandler) *Router {
	return &Router{
		mux:     http.NewServeMux(),
		handler: h,
	}
}

func (r *Router) Setup() http.Handler {
	r.mux.HandleFunc("POST /api/v1/auth/register", r.handler.Register)
	r.mux.HandleFunc("POST /api/v1/auth/login", r.handler.Login)

	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	return r.mux
}

func (r *Router) GetAuthMiddleware() func(http.Handler) http.Handler {
	return r.handler.AuthMiddleware
}
