package interfaces

import (
	"central_server/internal/gateway/domain"
)

// LambdaServiceInterface is an alias to domain.LambdaService for use in HTTP handlers
// This keeps the interfaces package focused on HTTP concerns while using domain-defined contracts
type LambdaServiceInterface = domain.LambdaService
