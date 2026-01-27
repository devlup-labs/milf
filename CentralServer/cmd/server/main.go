package main

import (
	"context"
	"log"
	"net/http"

	authcore "central_server/internal/auth/core"
	authhandler "central_server/internal/auth/handler"
	compilercore "central_server/internal/compiler/core"
	gwcore "central_server/internal/gateway/core"
	"central_server/internal/gateway/domain"
	gwhandler "central_server/internal/gateway/handler" // gwinterfaces alias in original, reusing
	gwinterfaces "central_server/internal/gateway/interfaces"
	orchcore "central_server/internal/orchestrator/core"
	navqueue "central_server/internal/queueService/core"
	qscore "central_server/internal/queueService/core"
	sinkcore "central_server/internal/sinkManager/core"
	sinkhandler "central_server/internal/sinkManager/handler"
	sinkinterfaces "central_server/internal/sinkManager/interfaces"
	"central_server/internal/storage"
)

func main() {
	ctx := context.Background()

	// Auth
	userRepo := storage.NewMemoryUserRepo()
	authService := authcore.NewAuthService(userRepo, "dev-secret")
	authHandler := authhandler.NewAuthHandler(authService)

	// --- FUNCTION / COMPILER / ORCHESTRATOR WIRING ---

	// 1. Storage
	lambdaRepo := storage.NewMemoryLambdaRepo()
	gatewayDB := lambdaRepo 
	compilerRepo := lambdaRepo 

	// ObjectStore for Compiler
	objectStore := storage.NewMemoryObjectStore()
	
	// Trigger for Compiler
	trigger := &storage.DummyRunTrigger{} 

	// 2. Queues
	compQueue := domain.NewCompilationQueue()

	queueServiceRaw := navqueue.NewQueueService()

	lambdaService := gwcore.NewLambdaService(gatewayDB, compilerRepo, nil, compQueue)
	orchestrator := orchcore.NewOrchestrator(lambdaRepo, lambdaService, queueServiceRaw)
	compiler := compilercore.NewCompiler(objectStore, trigger, compQueue, orchestrator)
	go compiler.Start(ctx)

	// 4. Wire Circular Dependencies
	lambdaService.SetOrchestrator(orchestrator)
	// lambdaService.SetCompiler(compiler) // Removed as not used by Gateway directly


	// 5. Handlers & Routers
	lambdaHandler := gwhandler.NewLambdaHandler(lambdaService)
	gatewayRouter := gwinterfaces.NewRouter(lambdaHandler, authHandler.AuthMiddleware)

	// --- SINK MANAGER ---
	sinkRepo := storage.NewMemorySinkRepo()
	taskRepo := storage.NewMemoryTaskRepo()
	resultRepo := storage.NewMemoryTaskResultRepo()
	sinkClient := storage.DummySinkClient{}

	// QueueService - directly wired to sinkManager
	queueService := qscore.NewQueueService()

	sinkService := sinkcore.NewSinkManagerService(sinkRepo, taskRepo, resultRepo, sinkClient, queueService, nil, "dev-secret")
	sinkHandler := sinkhandler.NewSinkHandler(sinkService)
	sinkRouter := sinkinterfaces.NewRouter(sinkHandler, authHandler.AuthMiddleware)

	// --- HTTP SERVER ---
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// Mount gateway routes
	gatewayMux := gatewayRouter.Setup()
	mux.Handle("/api/v1/lambdas", gatewayMux)
	mux.Handle("/api/v1/lambdas/", gatewayMux)
	mux.Handle("/api/v1/executions/", gatewayMux)
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
