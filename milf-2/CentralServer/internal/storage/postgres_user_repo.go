package storage

import (
	"central_server/internal/auth/domain"
	"context"
	"database/sql"
	"errors"
	"sync"

	_ "github.com/lib/pq"
)

// PostgresUserRepo implements UserRepository using PostgreSQL
type PostgresUserRepo struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewPostgresUserRepo creates a new PostgreSQL user repository
func NewPostgresUserRepo(connString string) (*PostgresUserRepo, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create table if not exists
	if err := createUserTable(db); err != nil {
		return nil, err
	}

	return &PostgresUserRepo{db: db}, nil
}

// createUserTable creates the users table if it doesn't exist
func createUserTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`
	_, err := db.Exec(query)
	return err
}

// Create stores a new user in the database
func (r *PostgresUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	query := `
	INSERT INTO users (username, password_hash)
	VALUES ($1, $2)
	RETURNING id
	`

	var id string
	err := r.db.QueryRowContext(ctx, query, user.Username, user.Password).Scan(&id)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_username_key\"" {
			return errors.New("user already exists")
		}
		return err
	}

	user.ID = id
	return nil
}

// GetByUsername retrieves a user by username
func (r *PostgresUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
	SELECT id, username, password_hash
	FROM users
	WHERE username = $1
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return user, nil
}

// Close closes the database connection
func (r *PostgresUserRepo) Close() error {
	return r.db.Close()
}

// GetDB returns the underlying database connection
func (r *PostgresUserRepo) GetDB() *sql.DB {
	return r.db
}
