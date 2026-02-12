package storage

import (
	"database/sql"
	"errors"

	compilerdomain "central_server/internal/compiler/domain"
	gatewaydomain "central_server/internal/gateway/domain"
)

// PostgresObjectStore implements compiler interfaces.ObjectStore
// by reading from the PostgreSQL functions table
type PostgresObjectStore struct {
	db *sql.DB
}

func NewPostgresObjectStore(db *sql.DB) *PostgresObjectStore {
	return &PostgresObjectStore{db: db}
}

func (s *PostgresObjectStore) FetchCompilationRequest(lambdaID string) (compilerdomain.CompilationRequest, error) {
	// Query the functions table
	query := `SELECT id, user_id, name, runtime, memory_mb, source_code FROM functions WHERE id = $1`
	
	var lambda struct {
		ID         string
		UserID     string
		Name       string
		Runtime    string
		MemoryMB   int
		SourceCode []byte
	}
	
	err := s.db.QueryRow(query, lambdaID).Scan(
		&lambda.ID,
		&lambda.UserID,
		&lambda.Name,
		&lambda.Runtime,
		&lambda.MemoryMB,
		&lambda.SourceCode,
	)
	
	if err == sql.ErrNoRows {
		return compilerdomain.CompilationRequest{}, errors.New("request not found")
	}
	if err != nil {
		return compilerdomain.CompilationRequest{}, err
	}
	
	// Map gateway runtime to compiler runtime
	var compilerRuntime compilerdomain.RuntimeType
	switch gatewaydomain.RuntimeEnvironment(lambda.Runtime) {
	case gatewaydomain.RuntimeGo:
		compilerRuntime = compilerdomain.RuntimeGo
	case gatewaydomain.RuntimeC:
		compilerRuntime = compilerdomain.RuntimeC
	case gatewaydomain.RuntimeRust:
		compilerRuntime = compilerdomain.RuntimeRust
	case gatewaydomain.RuntimeCpp:
		compilerRuntime = compilerdomain.RuntimeCpp
	default:
		compilerRuntime = compilerdomain.RuntimeGo
	}
	
	// Convert source code bytes to SourceFile
	sourceFile := compilerdomain.SourceFile{
		Path:    "main.go", // Default name, could be dynamic based on runtime
		Content: lambda.SourceCode,
	}
	
	// Build CompilationRequest
	req := compilerdomain.CompilationRequest{
		LambdaID:    lambda.ID,
		UserID:      lambda.UserID,
		Runtime:     compilerRuntime,
		SourceFiles: []compilerdomain.SourceFile{sourceFile},
		Metadata: compilerdomain.FunctionMetadata{
			Name:          lambda.Name,
			MemoryMB:      lambda.MemoryMB,
			TimeoutSec:    30, // Default timeout
			EntryPoint:    "Handler",
			LambdaRef:     lambda.ID,
			UserID:        lambda.UserID,
		},
		RunImmediate: false,
	}
	
	return req, nil
}

func (s *PostgresObjectStore) StoreWasm(lambdaID string, wasmBytes []byte) error {
	// Update the wasm_ref column with the WASM binary
	// For now, we'll store the binary directly in the DB
	// In production, you'd store in S3 and save the reference
	query := `UPDATE functions SET wasm_ref = $1, updated_at = NOW() WHERE id = $2`
	_, err := s.db.Exec(query, wasmBytes, lambdaID)
	return err
}

func (s *PostgresObjectStore) StoreMetadata(lambdaID string, meta compilerdomain.FunctionMetadata) error {
	// Could store metadata in a separate table or JSON column
	// For now, just return nil as metadata is already in the functions table
	return nil
}
