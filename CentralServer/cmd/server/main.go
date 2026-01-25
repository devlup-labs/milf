package main

import (
	"log"
	"net/http"

	authcore "central_server/internal/auth/core"
	authhandler "central_server/internal/auth/handler"
	gwcore "central_server/internal/gateway/core"
	gwhandler "central_server/internal/gateway/handler"
	gwinterfaces "central_server/internal/gateway/interfaces"
	"central_server/internal/storage"
)

func main() {
	// Auth
	userRepo := storage.NewMemoryUserRepo()
	authService := authcore.NewAuthService(userRepo, "dev-secret")
	authHandler := authhandler.NewAuthHandler(authService)

	// Gateway
	lambdaRepo := storage.NewMemoryLambdaRepo()
	execRepo := storage.NewMemoryExecutionRepo()
	compiler := storage.DummyCompiler{}
	orchestrator := storage.DummyOrchestrator{}

	lambdaService := gwcore.NewLambdaService(lambdaRepo, execRepo, compiler, orchestrator)
	lambdaHandler := gwhandler.NewLambdaHandler(lambdaService)
	router := gwinterfaces.NewRouter(lambdaHandler, authHandler.AuthMiddleware)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.Handle("/", router.Setup())

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
