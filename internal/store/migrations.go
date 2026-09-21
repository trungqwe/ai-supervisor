package store

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	CurrentSchemaVersion = 1
)

const v1Schema = `
CREATE TABLE IF NOT EXISTS projects (
    project_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL,
    repo_url TEXT,
    registered_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pairs (
    pair_id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL UNIQUE REFERENCES projects(project_id) ON DELETE RESTRICT,
    current_phase_id TEXT NOT NULL,
    active_task_id TEXT,
    state TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    task_id TEXT PRIMARY KEY,
    phase_id TEXT NOT NULL,
    pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
    state TEXT NOT NULL CHECK (state IN (
        'DRAFT', 'READY', 'DISPATCHED', 'RUNNING',
        'REPORT_READY', 'EVIDENCE_READY', 'REVIEWING',
        'APPROVED', 'REVISION_REQUIRED', 'BLOCKED',
        'FAILED', 'HUMAN_REQUIRED', 'CANCELLED'
    )),
    current_attempt INTEGER NOT NULL DEFAULT 0 CHECK (current_attempt >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS task_contracts (
    contract_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    revision_number INTEGER NOT NULL CHECK (revision_number >= 1),
    supersedes_contract_id TEXT REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    base_sha TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    is_immutable INTEGER NOT NULL DEFAULT 0 CHECK (is_immutable IN (0, 1)),
    UNIQUE(task_id, revision_number),
    UNIQUE(supersedes_contract_id),
    UNIQUE(contract_id, task_id)
);

CREATE TRIGGER IF NOT EXISTS trg_task_contracts_prevent_update_immutable
BEFORE UPDATE ON task_contracts
FOR EACH ROW
WHEN OLD.is_immutable = 1
BEGIN
    SELECT RAISE(ABORT, 'cannot update immutable contract');
END;

CREATE TRIGGER IF NOT EXISTS trg_task_contracts_prevent_delete_immutable
BEFORE DELETE ON task_contracts
FOR EACH ROW
WHEN OLD.is_immutable = 1
BEGIN
    SELECT RAISE(ABORT, 'cannot delete immutable contract');
END;

CREATE TABLE IF NOT EXISTS task_attempts (
    attempt_id TEXT PRIMARY KEY,
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    expected_report_path TEXT NOT NULL,
    started_at TEXT NOT NULL,
    ended_at TEXT,
    worker_report_raw TEXT,
    UNIQUE(task_id, attempt_number),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);
`

func migrate(ctx context.Context, db *sql.DB) error {
	var userVersion int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&userVersion); err != nil {
		return fmt.Errorf("store: failed to read PRAGMA user_version: %w", err)
	}

	if userVersion > CurrentSchemaVersion {
		return fmt.Errorf("%w: database user_version %d > supported %d",
			ErrUnsupportedSchemaVersion, userVersion, CurrentSchemaVersion)
	}

	if userVersion == CurrentSchemaVersion {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, v1Schema); err != nil {
		return fmt.Errorf("store: failed to execute v1 migration DDL: %w", err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", CurrentSchemaVersion)); err != nil {
		return fmt.Errorf("store: failed to set PRAGMA user_version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: failed to commit migration: %w", err)
	}

	return nil
}
