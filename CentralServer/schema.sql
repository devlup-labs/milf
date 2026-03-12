-- PostgreSQL Schema for CentralServer

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Functions table
CREATE TABLE IF NOT EXISTS functions (
    id TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    runtime VARCHAR(50),
    memory INT,
    memory_mb INT,
    source_code TEXT,
    wasm_ref BYTEA,
    run_type VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_functions_user_id ON functions(user_id);

-- Executions/Invocations table
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lambda_id TEXT NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    reference_id VARCHAR(255),
    input JSONB,
    status VARCHAR(50),
    output JSONB,
    error TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP,
    duration_ms INT,
    memory_used_mb INT
);

CREATE INDEX IF NOT EXISTS idx_executions_lambda_id ON executions(lambda_id);
CREATE INDEX IF NOT EXISTS idx_executions_reference_id ON executions(reference_id);

-- Logs table
CREATE TABLE IF NOT EXISTS logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id VARCHAR(255),
    function_name VARCHAR(255),
    level VARCHAR(50),
    message TEXT,
    details TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_logs_request_id ON logs(request_id);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);

-- Per-user execution and compute policies
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_executions_per_day INT NOT NULL DEFAULT 100,
    max_compute_mb_per_cycle INT NOT NULL DEFAULT 10240,
    allowed_runtimes TEXT[] NOT NULL DEFAULT ARRAY['c'],
    max_concurrent_tasks INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

-- Rolling billing cycle usage (reset each cycle_start date)
CREATE TABLE IF NOT EXISTS billing_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cycle_start DATE NOT NULL,
    executions_count INT NOT NULL DEFAULT 0,
    compute_mb_used INT NOT NULL DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, cycle_start)
);

CREATE INDEX IF NOT EXISTS idx_billing_usage_user_id ON billing_usage(user_id);
