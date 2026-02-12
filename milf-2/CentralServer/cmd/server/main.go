package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	authcore "central_server/internal/auth/core"
	authhandler "central_server/internal/auth/handler"
	compilercore "central_server/internal/compiler/core"
	gwcore "central_server/internal/gateway/core"
	"central_server/internal/gateway/domain"
	gwhandler "central_server/internal/gateway/handler" // gwinterfaces alias in original, reusing
	gwinterfaces "central_server/internal/gateway/interfaces"
	orchcore "central_server/internal/orchestrator/core"
	navqueue "central_server/internal/queueService/core"
	sinkcore "central_server/internal/sinkManager/core"
	sinkhandler "central_server/internal/sinkManager/handler"
	sinkinterfaces "central_server/internal/sinkManager/interfaces"
	"central_server/internal/storage"
)

func main() {
	ctx := context.Background()

	// Load .env file
	godotenv.Load(".env")

	// Load database configuration from environment variables
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "central_server_db"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		log.Fatal("DB_PASSWORD environment variable not set")
	}

	// Build connection string
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Auth - Connect to PostgreSQL
	userRepo, err := storage.NewPostgresUserRepo(connString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer userRepo.Close()
	
	// Execution repository (same database connection)
	executionRepo := storage.NewPostgresExecutionRepo(userRepo.GetDB())
	
	authService := authcore.NewAuthService(userRepo, "dev-secret")
	authHandler := authhandler.NewAuthHandler(authService)

	// --- FUNCTION / COMPILER / ORCHESTRATOR WIRING ---

	// 1. Storage - Use PostgreSQL for functions
	functionRepo := storage.NewPostgresFunctionRepo(userRepo.GetDB())
	gatewayDB := functionRepo
	compilerRepo := functionRepo

	// ObjectStore for Compiler - Use PostgreSQL to fetch from same DB
	objectStore := storage.NewPostgresObjectStore(userRepo.GetDB())

	// Trigger for Compiler
	trigger := &storage.DummyRunTrigger{}

	// 2. Queues
	compQueue := domain.NewCompilationQueue()

	queueService := navqueue.NewQueueService()

	lambdaService := gwcore.NewLambdaService(gatewayDB, compilerRepo, nil, compQueue, executionRepo)
	orchestrator := orchcore.NewOrchestrator(functionRepo, lambdaService, queueService)
	compiler := compilercore.NewCompiler(objectStore, trigger, compQueue, orchestrator)
	go compiler.Start(ctx)

	// 4. Wire Circular Dependencies
	lambdaService.SetOrchestrator(orchestrator)

	// 5. Handlers & Routers
	lambdaHandler := gwhandler.NewLambdaHandler(lambdaService)
	compatHandler := gwhandler.NewCompatHandler(lambdaService)
	gatewayRouter := gwinterfaces.NewRouter(lambdaHandler, compatHandler, authHandler.AuthMiddleware)
	// --- SINK MANAGER ---
	sinkRepo := storage.NewMemorySinkRepo()
	taskRepo := storage.NewMemoryTaskRepo()
	resultRepo := storage.NewMemoryTaskResultRepo()
	sinkClient := storage.DummySinkClient{}

	// QueueService - already created above

	sinkService := sinkcore.NewSinkManagerService(sinkRepo, taskRepo, resultRepo, sinkClient, queueService, nil, "dev-secret")

	// Break circular dependency
	queueService.SetSinkManager(sinkService)
	sinkHandler := sinkhandler.NewSinkHandler(sinkService)
	sinkRouter := sinkinterfaces.NewRouter(sinkHandler, authHandler.AuthMiddleware)

	// --- HTTP SERVER ---
	mux := http.NewServeMux()

	// CORS middleware
	cors := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	mux.HandleFunc("/api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)

	// Mount gateway routes
	gatewayMux := gatewayRouter.Setup()
	mux.Handle("/api/v1/lambdas", gatewayMux)
	mux.Handle("/api/v1/lambdas/", gatewayMux)
	mux.Handle("/api/v1/executions/", gatewayMux)
	mux.Handle("/health", gatewayMux)

	mux.Handle("/functions", gatewayMux)
	mux.Handle("/functions/", gatewayMux)
	mux.Handle("/invocations", gatewayMux)

	// Mount sink manager routes
	sinkMux := sinkRouter.Setup()
	mux.Handle("/api/v1/sinks", sinkMux)
	mux.Handle("/api/v1/sinks/", sinkMux)
	mux.Handle("/api/v1/tasks/", sinkMux)

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, cors(mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
