package main

import (
	"log"
	"net/http"

	authcore "central_server/internal/auth/core"
	authhandler "central_server/internal/auth/handler"
	gwcore "central_server/internal/gateway/core"
	gwhandler "central_server/internal/gateway/handler"
	gwinterfaces "central_server/internal/gateway/interfaces"
	sinkcore "central_server/internal/sinkManager/core"
	sinkhandler "central_server/internal/sinkManager/handler"
	sinkinterfaces "central_server/internal/sinkManager/interfaces"
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
	compilationQueue := storage.NewDummyCompilationQueue()
	orchestrator := storage.DummyOrchestrator{}

	lambdaService := gwcore.NewLambdaService(lambdaRepo, execRepo, compilationQueue, orchestrator)
	lambdaHandler := gwhandler.NewLambdaHandler(lambdaService)
	gatewayRouter := gwinterfaces.NewRouter(lambdaHandler, authHandler.AuthMiddleware)

	// SinkManager
	sinkRepo := storage.NewMemorySinkRepo()
	taskRepo := storage.NewMemoryTaskRepo()
	resultRepo := storage.NewMemoryTaskResultRepo()
	sinkClient := storage.DummySinkClient{}

	sinkService := sinkcore.NewSinkManagerService(sinkRepo, taskRepo, resultRepo, sinkClient, nil, "dev-secret")
	sinkHandler := sinkhandler.NewSinkHandler(sinkService)
	sinkRouter := sinkinterfaces.NewRouter(sinkHandler, authHandler.AuthMiddleware)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Mount gateway routes
	gatewayMux := gatewayRouter.Setup()
	mux.Handle("/api/v1/lambdas", gatewayMux)
	mux.Handle("/api/v1/lambdas/", gatewayMux)
	mux.Handle("/api/v1/executions/", gatewayMux)
	mux.Handle("/api/v1/compilations/", gatewayMux)
	mux.Handle("/health", gatewayMux)

	// Mount sink manager routes
	sinkMux := sinkRouter.Setup()
	mux.Handle("/api/v1/sinks", sinkMux)
	mux.Handle("/api/v1/sinks/", sinkMux)
	mux.Handle("/api/v1/tasks/", sinkMux)

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
