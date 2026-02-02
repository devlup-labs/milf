package storage

import (
	"central_server/internal/gateway/domain"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
)

// PostgresExecutionRepo implements ExecutionRepository using PostgreSQL
type PostgresExecutionRepo struct {
	db *sql.DB
}

// NewPostgresExecutionRepo creates a new PostgreSQL execution repository
func NewPostgresExecutionRepo(db *sql.DB) *PostgresExecutionRepo {
	return &PostgresExecutionRepo{db: db}
}

// Create stores a new execution in the database
func (r *PostgresExecutionRepo) Create(ctx context.Context, execution *domain.Execution) error {
	inputJSON, err := json.Marshal(execution.Input)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO executions (id, lambda_id, reference_id, input, status, started_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.ExecContext(ctx, query,
		execution.ID,
		execution.LambdaID,
		execution.ReferenceID,
		inputJSON,
		execution.Status,
		execution.StartedAt,
	)

	return err
}

// GetByID retrieves an execution by ID
func (r *PostgresExecutionRepo) GetByID(ctx context.Context, id string) (*domain.Execution, error) {
	query := `
	SELECT id, lambda_id, reference_id, input, status, output, error, started_at, finished_at
	FROM executions
	WHERE id = $1
	`

	execution := &domain.Execution{}
	var inputJSON, outputJSON []byte
	var errorMsg sql.NullString
	var finishedAt pq.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&execution.ID,
		&execution.LambdaID,
		&execution.ReferenceID,
		&inputJSON,
		&execution.Status,
		&outputJSON,
		&errorMsg,
		&execution.StartedAt,
		&finishedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("execution not found")
		}
		return nil, err
	}

	// Unmarshal JSON fields
	if len(inputJSON) > 0 {
		json.Unmarshal(inputJSON, &execution.Input)
	}
	if len(outputJSON) > 0 {
		json.Unmarshal(outputJSON, &execution.Output)
	}

	if errorMsg.Valid {
		execution.Error = errorMsg.String
	}

	if finishedAt.Valid {
		execution.FinishedAt = &finishedAt.Time
	}

	return execution, nil
}

// UpdateStatus updates the status and result of an execution
func (r *PostgresExecutionRepo) UpdateStatus(ctx context.Context, id string, status domain.ExecutionStatus, output interface{}, errorMsg string) error {
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return err
	}

	finishedAt := time.Now()

	query := `
	UPDATE executions
	SET status = $1, output = $2, error = $3, finished_at = $4
	WHERE id = $5
	`

	_, err = r.db.ExecContext(ctx, query, status, outputJSON, errorMsg, finishedAt, id)
	return err
}
