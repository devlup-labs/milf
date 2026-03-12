package policy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// UserPolicy holds the configured limits for a user.
type UserPolicy struct {
	UserID               string
	MaxExecutionsPerDay  int
	MaxComputeMbPerCycle int
	MaxConcurrentTasks   int
	AllowedRuntimes      []string
}

// UsageStats holds the current billing cycle usage.
type UsageStats struct {
	ExecutionsCount int
	ComputeMbUsed   int
	CycleStart      string
}

// PolicyReport is returned by GetUsage.
type PolicyReport struct {
	Usage  UsageStats
	Limits UserPolicy
}

// Manager enforces policies and tracks billing usage.
type Manager struct {
	db *sql.DB
}

// NewManager creates a Policy Manager backed by the shared DB.
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// InitForUser inserts default policy + billing rows when a new user registers.
// Safe to call multiple times (uses ON CONFLICT DO NOTHING).
func (m *Manager) InitForUser(ctx context.Context, userID string) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO policies (user_id, max_executions_per_day, max_compute_mb_per_cycle, allowed_runtimes, max_concurrent_tasks)
		VALUES ($1, 100, 10240, ARRAY['c'], 1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	if err != nil {
		return fmt.Errorf("policy.InitForUser: %w", err)
	}
	_, err = m.db.ExecContext(ctx, `
		INSERT INTO billing_usage (user_id, cycle_start, executions_count, compute_mb_used)
		VALUES ($1, date_trunc('month', NOW())::date, 0, 0)
		ON CONFLICT (user_id, cycle_start) DO NOTHING
	`, userID)
	if err != nil {
		return fmt.Errorf("billing.InitForUser: %w", err)
	}
	return nil
}

// Check validates whether a user may dispatch a new task.
// Returns a descriptive error on policy violation, nil if allowed.
func (m *Manager) Check(ctx context.Context, userID string, runtime string) error {
	var p UserPolicy
	var runtimes pq.StringArray
	row := m.db.QueryRowContext(ctx, `
		SELECT max_executions_per_day, max_compute_mb_per_cycle, max_concurrent_tasks, allowed_runtimes
		FROM policies WHERE user_id = $1
	`, userID)
	if err := row.Scan(&p.MaxExecutionsPerDay, &p.MaxComputeMbPerCycle, &p.MaxConcurrentTasks, &runtimes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // No policy yet → allow (pre-init state)
		}
		return fmt.Errorf("policy.Check: %w", err)
	}
	p.AllowedRuntimes = []string(runtimes)

	// Runtime allowlist check
	allowed := false
	for _, r := range p.AllowedRuntimes {
		if r == runtime {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("runtime '%s' is not permitted (allowed: %v)", runtime, p.AllowedRuntimes)
	}

	// Daily execution count check
	var todayCount int
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM executions e
		JOIN functions f ON e.lambda_id = f.id
		WHERE f.user_id = $1 AND e.started_at >= CURRENT_DATE
	`, userID).Scan(&todayCount)
	if err == nil && todayCount >= p.MaxExecutionsPerDay {
		return errors.New("daily execution limit reached — upgrade your plan")
	}

	// Cycle compute quota check
	var computeUsed int
	err = m.db.QueryRowContext(ctx, `
		SELECT compute_mb_used FROM billing_usage
		WHERE user_id = $1 AND cycle_start = date_trunc('month', NOW())::date
	`, userID).Scan(&computeUsed)
	if err == nil && computeUsed >= p.MaxComputeMbPerCycle {
		return errors.New("monthly compute quota exceeded — upgrade your plan")
	}

	return nil
}

// RecordExecution increments billing counters after a completed task.
func (m *Manager) RecordExecution(ctx context.Context, userID string, computeMbUsed int) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO billing_usage (user_id, cycle_start, executions_count, compute_mb_used, last_updated)
		VALUES ($1, date_trunc('month', NOW())::date, 1, $2, NOW())
		ON CONFLICT (user_id, cycle_start) DO UPDATE SET
			executions_count = billing_usage.executions_count + 1,
			compute_mb_used  = billing_usage.compute_mb_used + EXCLUDED.compute_mb_used,
			last_updated     = NOW()
	`, userID, computeMbUsed)
	return err
}

// GetUsage returns current cycle stats for a user.
func (m *Manager) GetUsage(ctx context.Context, userID string) (*PolicyReport, error) {
	report := &PolicyReport{}
	report.Limits.UserID = userID

	var runtimes pq.StringArray
	_ = m.db.QueryRowContext(ctx, `
		SELECT max_executions_per_day, max_compute_mb_per_cycle, max_concurrent_tasks, allowed_runtimes
		FROM policies WHERE user_id = $1
	`, userID).Scan(
		&report.Limits.MaxExecutionsPerDay,
		&report.Limits.MaxComputeMbPerCycle,
		&report.Limits.MaxConcurrentTasks,
		&runtimes,
	)
	report.Limits.AllowedRuntimes = []string(runtimes)

	_ = m.db.QueryRowContext(ctx, `
		SELECT executions_count, compute_mb_used, cycle_start::text
		FROM billing_usage
		WHERE user_id = $1 AND cycle_start = date_trunc('month', NOW())::date
	`, userID).Scan(
		&report.Usage.ExecutionsCount,
		&report.Usage.ComputeMbUsed,
		&report.Usage.CycleStart,
	)

	return report, nil
}
