package storage

import (
	"central_server/internal/auth/domain"
	"context"
	"errors"
	"sync"
)

// InMemoryUserRepo implements interfaces.UserRepository
type InMemoryUserRepo struct {
	users map[string]*domain.User
	mu    sync.RWMutex
}

// NewInMemoryUserRepo creates a new instance
func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		users: make(map[string]*domain.User),
	}
}

// Create adds a new user to the map
func (r *InMemoryUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[user.Username]; exists {
		return errors.New("user already exists")
	}
	// Copy user to avoid external mutation
	u := *user
	r.users[user.Username] = &u
	return nil
}

// GetByUsername retrieves a user by username
func (r *InMemoryUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, exists := r.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}
	// unique copy
	u := *user
	return &u, nil
}
