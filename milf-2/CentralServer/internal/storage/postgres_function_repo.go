package storage

import (
	"central_server/internal/gateway/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresFunctionRepo implements FuncGatewayDB using PostgreSQL
type PostgresFunctionRepo struct {
	db *sql.DB
}

// NewPostgresFunctionRepo creates a new PostgreSQL function repository
func NewPostgresFunctionRepo(db *sql.DB) *PostgresFunctionRepo {
	return &PostgresFunctionRepo{db: db}
}

// Save stores a new lambda function in the database
func (r *PostgresFunctionRepo) Save(ctx context.Context, lambda *domain.Lambda) error {
	query := `
	INSERT INTO functions (id, user_id, name, runtime, memory_mb, source_code, wasm_ref, run_type, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (id) DO UPDATE SET
		name = EXCLUDED.name,
		runtime = EXCLUDED.runtime,
		memory_mb = EXCLUDED.memory_mb,
		source_code = EXCLUDED.source_code,
		wasm_ref = EXCLUDED.wasm_ref,
		run_type = EXCLUDED.run_type,
		updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		lambda.ID,
		lambda.UserID,
		lambda.Name,
		string(lambda.Runtime),
		lambda.MemoryMB,
		string(lambda.SourceCode),
		lambda.WasmRef,
		string(lambda.RunType),
		lambda.CreatedAt,
		lambda.UpdatedAt,
	)

	return err
}

// FindByID retrieves a lambda function by ID
func (r *PostgresFunctionRepo) FindByID(ctx context.Context, id string) (*domain.Lambda, error) {
	query := `
	SELECT id, user_id, name, runtime, memory_mb, source_code, wasm_ref, run_type, created_at, updated_at
	FROM functions
	WHERE id = $1
	`

	lambda := &domain.Lambda{}
	var runtime, runType string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&lambda.ID,
		&lambda.UserID,
		&lambda.Name,
		&runtime,
		&lambda.MemoryMB,
		&lambda.SourceCode,
		&lambda.WasmRef,
		&runType,
		&lambda.CreatedAt,
		&lambda.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrLambdaNotFound
		}
		return nil, err
	}

	lambda.Runtime = domain.RuntimeEnvironment(runtime)
	lambda.RunType = domain.RunType(runType)

	return lambda, nil
}

// FindByWasmRef retrieves a lambda function by WasmRef
func (r *PostgresFunctionRepo) FindByWasmRef(ctx context.Context, wasmRef string) (*domain.Lambda, error) {
	query := `
	SELECT id, user_id, name, runtime, memory_mb, source_code, wasm_ref, run_type, created_at, updated_at
	FROM functions
	WHERE wasm_ref = $1
	`

	lambda := &domain.Lambda{}
	var runtime, runType string

	err := r.db.QueryRowContext(ctx, query, wasmRef).Scan(
		&lambda.ID,
		&lambda.UserID,
		&lambda.Name,
		&runtime,
		&lambda.MemoryMB,
		&lambda.SourceCode,
		&lambda.WasmRef,
		&runType,
		&lambda.CreatedAt,
		&lambda.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrLambdaNotFound
		}
		return nil, err
	}

	lambda.Runtime = domain.RuntimeEnvironment(runtime)
	lambda.RunType = domain.RunType(runType)

	return lambda, nil
}

// Delete removes a lambda function from the database
func (r *PostgresFunctionRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM functions WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrLambdaNotFound
	}

	return nil
}

// GetStatus retrieves the status of a lambda function
func (r *PostgresFunctionRepo) GetStatus(ctx context.Context, funcID string) (string, error) {
	query := `SELECT id FROM functions WHERE id = $1`
	
	var id string
	err := r.db.QueryRowContext(ctx, query, funcID).Scan(&id)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("lambda not found")
		}
		return "", err
	}
	
	return "compiled", nil
}

// List returns all lambdas, optionally filtered by userID
func (r *PostgresFunctionRepo) List(ctx context.Context, userID string) ([]*domain.Lambda, error) {
	var query string
	var rows *sql.Rows
	var err error

	if userID != "" && userID != "unknown" {
		query = `SELECT id, user_id, name, runtime, memory_mb, source_code, wasm_ref, run_type, created_at, updated_at FROM functions WHERE user_id = $1 ORDER BY created_at DESC`
		rows, err = r.db.QueryContext(ctx, query, userID)
	} else {
		query = `SELECT id, user_id, name, runtime, memory_mb, source_code, wasm_ref, run_type, created_at, updated_at FROM functions ORDER BY created_at DESC`
		rows, err = r.db.QueryContext(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lambdas []*domain.Lambda
	for rows.Next() {
		lambda := &domain.Lambda{}
		var runtime, runType string

		err := rows.Scan(
			&lambda.ID,
			&lambda.UserID,
			&lambda.Name,
			&runtime,
			&lambda.MemoryMB,
			&lambda.SourceCode,
			&lambda.WasmRef,
			&runType,
			&lambda.CreatedAt,
			&lambda.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		lambda.Runtime = domain.RuntimeEnvironment(runtime)
		lambda.RunType = domain.RunType(runType)
		lambdas = append(lambdas, lambda)
	}

	return lambdas, rows.Err()
}

// GetLambdaMetadata returns minimal metadata needed by the orchestrator
func (r *PostgresFunctionRepo) GetLambdaMetadata(ctx context.Context, funcID string) (map[string]string, error) {
	query := `
	SELECT user_id, memory_mb
	FROM functions
	WHERE id = $1
	`

	var userID string
	var memoryMB int

	err := r.db.QueryRowContext(ctx, query, funcID).Scan(&userID, &memoryMB)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("lambda not found")
		}
		return nil, err
	}

	meta := make(map[string]string)
	meta["user_id"] = userID
	meta["status"] = "compiled"
	meta["maxRam"] = fmt.Sprintf("%d", memoryMB)
	return meta, nil
}
