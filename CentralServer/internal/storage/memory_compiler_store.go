package storage

import (
	"central_server/internal/compiler/domain"
	"errors"
	"sync"
)

// MemoryObjectStore implements compiler interfaces.ObjectStore
type MemoryObjectStore struct {
	mu       sync.RWMutex
	requests map[string]domain.CompilationRequest
	wasm     map[string][]byte
	meta     map[string]domain.FunctionMetadata
}

func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{
		requests: make(map[string]domain.CompilationRequest),
		wasm:     make(map[string][]byte),
		meta:     make(map[string]domain.FunctionMetadata),
	}
}

func (s *MemoryObjectStore) FetchCompilationRequest(lambdaID string) (domain.CompilationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.requests[lambdaID]
	if !ok {
		return domain.CompilationRequest{}, errors.New("request not found")
	}
	return req, nil
}

func (s *MemoryObjectStore) StoreWasm(lambdaID string, wasmBytes []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wasm[lambdaID] = wasmBytes
	return nil
}

func (s *MemoryObjectStore) StoreMetadata(lambdaID string, meta domain.FunctionMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meta[lambdaID] = meta
	return nil
}

// Add helper to seed request for testing
func (s *MemoryObjectStore) SaveRequest(req domain.CompilationRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests[req.LambdaID] = req
}


// DummyRunTrigger implements compiler interfaces.RunTrigger
type DummyRunTrigger struct{}

func (t *DummyRunTrigger) TriggerRun(lambdaID string) error {
	// No-op or log
	return nil
}
