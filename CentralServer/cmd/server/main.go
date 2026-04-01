package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	authcore "central_server/internal/auth/core"
	authhandler "central_server/internal/auth/handler"
	compilercore "central_server/internal/compiler/core"
	gwcore "central_server/internal/gateway/core"
	"central_server/internal/gateway/domain"
	gwhandler "central_server/internal/gateway/handler" // gwinterfaces alias in original, reusing
	gwinterfaces "central_server/internal/gateway/interfaces"
	orchcore "central_server/internal/orchestrator/core"
	policy "central_server/internal/policy"
	navqueue "central_server/internal/queueService/core"
	sinkcore "central_server/internal/sinkManager/core"
	sinkhandler "central_server/internal/sinkManager/handler"
	sinkinterfaces "central_server/internal/sinkManager/interfaces"
	"central_server/internal/storage"
	"central_server/internal/filestore"
	"central_server/utils"
)

func main() {
	ctx := context.Background()

	// Setup logging to file and stdout
	logFile, _ := os.OpenFile("/tmp/milf_server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	multi := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multi)
	if utils.Logger != nil {
		utils.Logger.SetOutput(multi)
	}
	log.Printf("[Main] Server starting...")

	// Load .env file
	godotenv.Overload(".env")

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
	
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret"
	}
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")

	// Policy Manager (billing + quota enforcement)
	policyMgr := policy.NewManager(userRepo.GetDB())

	authService := authcore.NewAuthService(userRepo, jwtSecret, googleClientID, policyMgr)
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
	clangPath := os.Getenv("CLANG_PATH")
	log.Printf("[Main] Using CLANG_PATH from env: %s", clangPath)
	if _, err := os.Stat(clangPath); os.IsNotExist(err) {
		log.Printf("[CRITICAL] Clang binary NOT FOUND at %s. Compilation will fail.", clangPath)
	}
	compiler.SetClangPath(clangPath)
	go compiler.Start(ctx)

	// 4. Wire Circular Dependencies
	lambdaService.SetOrchestrator(orchestrator)

	// 4.5 Start Scheduler (Phase 2: Event-Driven)
	scheduler := gwcore.NewScheduler(functionRepo, orchestrator)
	lambdaService.SetScheduler(scheduler)
	go scheduler.Start(ctx)

	// 4.6 Wire Log Repository (Phase 4: Observability)
	logRepo := storage.NewPostgresLogRepo(userRepo.GetDB())
	lambdaService.SetLogRepo(logRepo)

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

	// ResultCallback: when Android returns a result, notify the lambdaService
	// so that synchronous HTTP requests (TriggerLambda) can unblock.
	resultCallback := func(ctx context.Context, executionID string, result interface{}, execErr error) {
		lambdaService.NotifyResult(ctx, executionID, result, execErr)
	}

	sinkService := sinkcore.NewSinkManagerService(sinkRepo, taskRepo, resultRepo, sinkClient, queueService, resultCallback, jwtSecret)

	// Break circular dependency
	queueService.SetSinkManager(sinkService)

	// Wire Recent Function Fetcher: Enables the testing mode where the most recent
	// function is auto-dispatched to any node that heartbeats if the queue is empty.
	/*
	// Wire Recent Function Fetcher: Enables the testing mode where the most recent
	// function is auto-dispatched to any node that heartbeats if the queue is empty.
	sinkService.SetRecentFuncFetcher(func(ctx context.Context) (string, error) {
		// FOR TESTING: return a special ID for the local addd.wasm file
		return "test-local-addd", nil
	})
	*/

	// Wire WASM fetcher: lets the WorkManager inline compiled WASM bytes
	// into the task payload so Android can execute without a separate download.
	sinkService.SetWasmFetcher(func(ctx context.Context, lambdaID string) ([]byte, error) {
		/*
		// FALLBACK FOR LOCAL TESTING:
		if lambdaID == "test-local-addd" {
			log.Printf("[WorkManager] 📁 Reading local test file for lambda: %s", lambdaID)
			return os.ReadFile("/Users/adarsh/Projects/devlup/milf/consumeronlywamr/test/addd.wasm")
		}
		*/

		lambda, err := functionRepo.FindByID(ctx, lambdaID)
		if err != nil {
			return nil, err
		}
		if len(lambda.WasmRef) == 0 {
			return nil, fmt.Errorf("WASM ref is empty")
		}
		return base64.StdEncoding.DecodeString(lambda.WasmRef)
	})

	sinkHandler := sinkhandler.NewSinkHandler(sinkService)
	sinkRouter := sinkinterfaces.NewRouter(sinkHandler, authHandler.AuthMiddleware)

	// Automatically re-enqueue pending tasks from the database into the memory queue
	go func() {
		time.Sleep(2 * time.Second)
		if err := lambdaService.SyncPendingTasks(ctx); err != nil {
			log.Printf("[Gateway] Error syncing pending tasks: %v", err)
		}
	}()

	// --- HTTP SERVER ---
	mux := http.NewServeMux()

	// CORS middleware
	cors := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Filename")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	mux.HandleFunc("/api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/google", authHandler.GoogleLogin)

	// Usage & billing endpoint (requires auth)
	mux.Handle("/api/v1/users/me/usage", authHandler.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := r.Context().Value("user_id").(string)
		report, err := policyMgr.GetUsage(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to get usage", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})))

	// Mount gateway routes
	gatewayMux := gatewayRouter.Setup()
	mux.Handle("/api/v1/", gatewayMux)

	// Direct mount for execution and simulations (Phase 1)
	mux.Handle("POST /api/v1/execute/{id}", authHandler.AuthMiddleware(http.HandlerFunc(lambdaHandler.Execute)))

	// Mount sink manager routes
	sinkMux := sinkRouter.Setup()
	mux.Handle("/api/v1/sinks/", sinkMux)
	mux.Handle("/api/v1/sinks", sinkMux)
	mux.Handle("/api/v1/tasks/", sinkMux)

	// Mount file store routes (for binary file upload/download)
	fileStore, err := filestore.NewFileStore("./uploads")
	if err != nil {
		log.Fatalf("failed to create file store: %v", err)
	}
	mux.HandleFunc("POST /api/v1/files", fileStore.HandleUpload)
	mux.HandleFunc("GET /api/v1/files/{id}", fileStore.HandleDownload)

	addr := ":8080"
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, cors(mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
