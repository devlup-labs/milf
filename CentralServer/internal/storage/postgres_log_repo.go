package storage

import (
	"central_server/internal/gateway/domain"
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// PostgresLogRepo implements domain.LogRepository using the existing `logs` table.
type PostgresLogRepo struct {
	db *sql.DB
}

func NewPostgresLogRepo(db *sql.DB) *PostgresLogRepo {
	return &PostgresLogRepo{db: db}
}

// Insert writes a single log entry to the database.
func (r *PostgresLogRepo) Insert(ctx context.Context, entry *domain.LogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	query := `
	INSERT INTO logs (id, request_id, function_name, level, message, details, timestamp)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.RequestID,
		entry.FunctionName,
		entry.Level,
		entry.Message,
		entry.Details,
		entry.Timestamp,
	)
	return err
}

// List retrieves log entries. If userID is provided, only logs for that user's
// functions are returned (via JOIN on functions table). If level is non-empty,
// results are filtered by log level.
func (r *PostgresLogRepo) List(ctx context.Context, userID string, level string, limit int) ([]*domain.LogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	// When userID is provided, join on functions to filter by ownership.
	// The function_name column in logs may hold a function ID or name;
	// we match on both functions.id and functions.name for resilience.
	var query string
	var args []interface{}

	if userID != "" {
		query = `
		SELECT l.id, l.request_id, l.function_name, l.level, l.message,
		       COALESCE(l.details, ''), l.timestamp
		FROM logs l
		INNER JOIN functions f ON (f.id = l.function_name OR f.name = l.function_name)
		WHERE f.user_id = $1
		`
		args = append(args, userID)

		if level != "" && level != "all" {
			query += " AND l.level = $2"
			args = append(args, level)
		}

		query += " ORDER BY l.timestamp DESC LIMIT $" + strconv.Itoa(len(args)+1)
		args = append(args, limit)
	} else {
		query = `
		SELECT id, request_id, function_name, level, message,
		       COALESCE(details, ''), timestamp
		FROM logs
		`
		if level != "" && level != "all" {
			query += " WHERE level = $1"
			args = append(args, level)
		}

		query += " ORDER BY timestamp DESC LIMIT $" + strconv.Itoa(len(args)+1)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*domain.LogEntry
	for rows.Next() {
		e := &domain.LogEntry{}
		if err := rows.Scan(&e.ID, &e.RequestID, &e.FunctionName, &e.Level, &e.Message, &e.Details, &e.Timestamp); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
