package gateway

import (
	"net/http"

	"central_server/internal/gateway/core"
	"central_server/internal/gateway/domain"
	"central_server/internal/gateway/interfaces"
)

// Gateway represents the main gateway module
type Gateway struct {
	service *core.LambdaService
	router  *interfaces.Router
}

// New creates a new Gateway with the provided dependencies
func New(
	lambdaRepo domain.LambdaRepository,
	execRepo domain.ExecutionRepository,
	compiler domain.CompilerService,
	orchestrator domain.OrchestratorService,
) *Gateway {
	// Create the core service
	service := core.NewLambdaService(lambdaRepo, execRepo, compiler, orchestrator)

	// Create the HTTP handler
	handler := interfaces.NewLambdaHandler(service)

	// Create and setup the router
	router := interfaces.NewRouter(handler)

	return &Gateway{
		service: service,
		router:  router,
	}
}

// Handler returns the HTTP handler for the gateway
func (g *Gateway) Handler() http.Handler {
	return g.router.Setup()
}

// Service returns the underlying lambda service for direct access if needed
func (g *Gateway) Service() *core.LambdaService {
	return g.service
}
