package interfaces

import "CentralServer/internal/compiler/domain"

type ObjectStore interface {

	// Fetch source code + metadata for compilation (future)
	FetchCompilationRequest(lambdaID string) (domain.CompilationRequest, error)

	// Store compiled WASM binary
	StoreWasm(lambdaID string, wasm []byte) error

	// Store metadata (entry point, memory, timeout)
	StoreMetadata(lambdaID string, meta domain.FunctionMetadata) error
}
