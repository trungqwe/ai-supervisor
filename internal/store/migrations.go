package store

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	CurrentSchemaVersion = 2
	GenesisAuditHash     = "0000000000000000000000000000000000000000000000000000000000000000"
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

const v2Schema = `
CREATE TABLE IF NOT EXISTS audit_events (
    sequence INTEGER PRIMARY KEY,
    event_id TEXT UNIQUE NOT NULL,
    event_type TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    pair_id TEXT,
    task_id TEXT,
    contract_id TEXT,
    attempt_id TEXT,
    actor TEXT NOT NULL,
    details_json TEXT NOT NULL,
    prev_hash TEXT NOT NULL,
    event_hash TEXT UNIQUE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_events_task_id ON audit_events(task_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_contract_id ON audit_events(contract_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_attempt_id ON audit_events(attempt_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_pair_id ON audit_events(pair_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp);

CREATE TABLE IF NOT EXISTS audit_chain_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_sequence INTEGER NOT NULL,
    last_hash TEXT NOT NULL
);

INSERT OR IGNORE INTO audit_chain_state (id, last_sequence, last_hash)
VALUES (1, 0, '0000000000000000000000000000000000000000000000000000000000000000');

CREATE TRIGGER IF NOT EXISTS trg_audit_events_prevent_update
BEFORE UPDATE ON audit_events
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'cannot update audit events: append-only');
END;

CREATE TRIGGER IF NOT EXISTS trg_audit_events_prevent_delete
BEFORE DELETE ON audit_events
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'cannot delete audit events: append-only');
END;

CREATE TRIGGER IF NOT EXISTS trg_audit_events_prevent_replace
BEFORE INSERT ON audit_events
FOR EACH ROW
WHEN EXISTS (
    SELECT 1 FROM audit_events
    WHERE sequence = NEW.sequence OR event_id = NEW.event_id
)
BEGIN
    SELECT RAISE(ABORT, 'cannot replace existing audit event');
END;

CREATE TRIGGER IF NOT EXISTS trg_audit_chain_state_prevent_delete
BEFORE DELETE ON audit_chain_state
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'cannot delete audit chain state');
END;

CREATE TRIGGER IF NOT EXISTS trg_audit_chain_state_prevent_insert
BEFORE INSERT ON audit_chain_state
FOR EACH ROW
WHEN EXISTS (SELECT 1 FROM audit_chain_state WHERE id = NEW.id)
BEGIN
    SELECT RAISE(ABORT, 'cannot insert duplicate audit chain state');
END;
`

func migrate(ctx context.Context, db *sql.DB) error {
	return migrateWithSchemas(ctx, db, v1Schema, v2Schema)
}

func migrateWithSchemas(ctx context.Context, db *sql.DB, v1DDL, v2DDL string) error {
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

	// Apply V1 if fresh DB
	if userVersion < 1 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: failed to begin v1 migration transaction: %w", err)
		}
		if _, err := tx.ExecContext(ctx, v1DDL); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to execute v1 migration DDL: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to set PRAGMA user_version = 1: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: failed to commit v1 migration: %w", err)
		}
		userVersion = 1
	}

	// Apply V2 if userVersion is 1
	if userVersion < 2 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: failed to begin v2 migration transaction: %w", err)
		}
		if _, err := tx.ExecContext(ctx, v2DDL); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to execute v2 migration DDL: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 2"); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to set PRAGMA user_version = 2: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: failed to commit v2 migration: %w", err)
		}
	}

	return nil
}
