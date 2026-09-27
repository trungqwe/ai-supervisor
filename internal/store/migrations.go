package store

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	CurrentSchemaVersion = 6
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

const v3Schema = `
CREATE TABLE worker_sessions (
    pair_id TEXT PRIMARY KEY REFERENCES pairs(pair_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL UNIQUE,
    runtime_type TEXT NOT NULL,
    worktree_path TEXT,
    worker_agent_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'IDLE', 'TERMINATED')),
    terminal_generation TEXT NOT NULL,
    quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

ALTER TABLE task_attempts ADD COLUMN session_id TEXT;
ALTER TABLE task_attempts ADD COLUMN terminal_generation TEXT;
ALTER TABLE task_attempts ADD COLUMN recovery_disposition TEXT;
ALTER TABLE task_attempts ADD COLUMN quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'));

CREATE TABLE pair_provisioning_operations (
    operation_id TEXT PRIMARY KEY,
    pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
    stage TEXT NOT NULL CHECK (stage IN ('PROVISION_REQUESTED', 'PROVISION_CONFIRMED', 'PROVISION_FAILED', 'PROVISION_RESOLVED')),
    client_token TEXT NOT NULL,
    session_id TEXT,
    requested_at TEXT NOT NULL,
    completed_at TEXT,
    resolved_at TEXT,
    resolved_by TEXT,
    resolution_notes TEXT
);
CREATE UNIQUE INDEX idx_pair_provisioning_unresolved ON pair_provisioning_operations(pair_id) WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED');

CREATE TABLE dispatch_operations (
    operation_id TEXT PRIMARY KEY,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL,
    terminal_generation TEXT NOT NULL,
    stage TEXT NOT NULL CHECK (stage IN ('DISPATCH_BOUND', 'SEND_REQUESTED', 'SEND_CONFIRMED')),
    requested_at TEXT NOT NULL,
    confirmed_at TEXT,
    resolution_state TEXT
);

CREATE TABLE stop_operations (
    operation_id TEXT PRIMARY KEY,
    purpose TEXT NOT NULL CHECK (purpose IN ('RUNNING_ATTEMPT_STOP', 'QUARANTINE_CLEANUP', 'PAIR_MAINTENANCE')),
    pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
    task_id TEXT REFERENCES tasks(task_id) ON DELETE RESTRICT,
    contract_id TEXT REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    attempt_id TEXT REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL,
    terminal_generation TEXT NOT NULL,
    stage TEXT NOT NULL CHECK (stage IN ('STOP_REQUESTED', 'STOP_CALL_SUCCEEDED', 'STOP_CALL_FAILED', 'STOP_TERMINATION_CONFIRMED', 'STOP_TARGET_ABSENT')),
    actor TEXT NOT NULL,
    requested_at TEXT NOT NULL,
    call_completed_at TEXT,
    confirmation_deadline_at TEXT,
    termination_confirmed_at TEXT,
    resolved_at TEXT,
    resolution_state TEXT NOT NULL DEFAULT 'IN_FLIGHT'
);
`

const v4Schema = `
CREATE TABLE restore_authorizations (
 authorization_id TEXT PRIMARY KEY, operation_id TEXT NOT NULL UNIQUE,
 pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
 session_id TEXT NOT NULL, expected_generation TEXT NOT NULL,
 risk_scope TEXT NOT NULL CHECK (risk_scope IN ('POSSIBLE_PROMPT_REPLAY','PROMPT_REPLAY_AND_UNRESOLVED_EXECUTION')),
 authorized_principal TEXT NOT NULL,
 authorization_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
 issued_at TEXT NOT NULL, consumed_at TEXT, consumed_by_operation_id TEXT UNIQUE,
 UNIQUE(authorization_id,operation_id,pair_id,session_id,expected_generation),
 CHECK ((consumed_at IS NULL AND consumed_by_operation_id IS NULL) OR
        (consumed_at IS NOT NULL AND consumed_by_operation_id IS NOT NULL AND consumed_by_operation_id = operation_id))
);
CREATE TABLE pair_restore_operations (
 operation_id TEXT PRIMARY KEY,
 authorization_id TEXT NOT NULL UNIQUE REFERENCES restore_authorizations(authorization_id) ON DELETE RESTRICT,
 pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
 session_id TEXT NOT NULL, expected_generation TEXT NOT NULL, observed_generation TEXT,
 restore_mode TEXT CHECK (restore_mode IN ('native','saved_prompt','fresh')),
 stage TEXT NOT NULL CHECK (stage IN ('RESTORE_REQUESTED','RESTORE_CONFIRMED')),
 resolution_state TEXT NOT NULL CHECK (resolution_state IN ('IN_FLIGHT','RESTORE_OUTCOME_UNKNOWN','RESTORE_RECOVERY_CLAIMED','RESTORE_CLEANUP_CLAIMED','RESTORE_RESOLVED')),
 resolution_basis TEXT CHECK (resolution_basis IN ('RESTORE_HTTP_200_CONFIRMED','PHYSICAL_EXECUTION_RESOLUTION','ADMINISTRATIVE_RISK_RESOLUTION')),
 recovery_principal TEXT, recovery_claimed_at TEXT, requested_at TEXT NOT NULL, http_200_at TEXT,
 resolved_at TEXT, resolution_event_id TEXT UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
 version INTEGER NOT NULL DEFAULT 0 CHECK(version >= 0),
 FOREIGN KEY(authorization_id,operation_id,pair_id,session_id,expected_generation)
  REFERENCES restore_authorizations(authorization_id,operation_id,pair_id,session_id,expected_generation) ON DELETE RESTRICT,
 CHECK ((stage='RESTORE_CONFIRMED' AND http_200_at IS NOT NULL AND observed_generation IS NOT NULL AND restore_mode IS NOT NULL) OR
        (stage='RESTORE_REQUESTED' AND http_200_at IS NULL AND observed_generation IS NULL AND restore_mode IS NULL)),
 CHECK ((resolution_state='RESTORE_RESOLVED' AND resolved_at IS NOT NULL AND resolution_basis IS NOT NULL AND resolution_event_id IS NOT NULL) OR
        (resolution_state<>'RESTORE_RESOLVED' AND resolved_at IS NULL AND resolution_basis IS NULL AND resolution_event_id IS NULL)),
 CHECK (resolution_basis<>'RESTORE_HTTP_200_CONFIRMED' OR stage='RESTORE_CONFIRMED'),
 CHECK (resolution_state NOT IN ('RESTORE_RECOVERY_CLAIMED','RESTORE_CLEANUP_CLAIMED') OR (recovery_principal IS NOT NULL AND recovery_claimed_at IS NOT NULL))
);
CREATE UNIQUE INDEX idx_pair_restore_unresolved ON pair_restore_operations(pair_id) WHERE resolution_state <> 'RESTORE_RESOLVED';
CREATE INDEX idx_restore_stage_resolution ON pair_restore_operations(stage,resolution_state);
ALTER TABLE stop_operations ADD COLUMN restore_operation_id TEXT REFERENCES pair_restore_operations(operation_id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX idx_stop_restore_operation ON stop_operations(restore_operation_id) WHERE restore_operation_id IS NOT NULL;
CREATE TRIGGER trg_restore_auth_no_delete BEFORE DELETE ON restore_authorizations BEGIN SELECT RAISE(ABORT,'restore authorization is append-only'); END;
CREATE TRIGGER trg_restore_op_no_delete BEFORE DELETE ON pair_restore_operations BEGIN SELECT RAISE(ABORT,'restore operation is append-only'); END;
CREATE TRIGGER trg_restore_auth_immutable BEFORE UPDATE ON restore_authorizations
WHEN NEW.authorization_id IS NOT OLD.authorization_id OR NEW.operation_id IS NOT OLD.operation_id OR
 NEW.pair_id IS NOT OLD.pair_id OR NEW.session_id IS NOT OLD.session_id OR NEW.expected_generation IS NOT OLD.expected_generation OR
 NEW.risk_scope IS NOT OLD.risk_scope OR NEW.authorized_principal IS NOT OLD.authorized_principal OR
 NEW.authorization_event_id IS NOT OLD.authorization_event_id OR NEW.issued_at IS NOT OLD.issued_at OR
 (OLD.consumed_at IS NOT NULL AND (NEW.consumed_at IS NOT OLD.consumed_at OR NEW.consumed_by_operation_id IS NOT OLD.consumed_by_operation_id))
BEGIN SELECT RAISE(ABORT,'restore authorization immutable or already consumed'); END;
CREATE TRIGGER trg_restore_op_forward_only BEFORE UPDATE ON pair_restore_operations
WHEN NEW.operation_id IS NOT OLD.operation_id OR NEW.authorization_id IS NOT OLD.authorization_id OR NEW.pair_id IS NOT OLD.pair_id OR
 NEW.session_id IS NOT OLD.session_id OR NEW.expected_generation IS NOT OLD.expected_generation OR NEW.requested_at IS NOT OLD.requested_at OR
 NEW.version <> OLD.version+1 OR
 (OLD.stage='RESTORE_CONFIRMED' AND (NEW.stage<>'RESTORE_CONFIRMED' OR NEW.http_200_at IS NOT OLD.http_200_at OR NEW.observed_generation IS NOT OLD.observed_generation OR NEW.restore_mode IS NOT OLD.restore_mode)) OR
 (OLD.stage='RESTORE_REQUESTED' AND NEW.stage='RESTORE_CONFIRMED' AND OLD.resolution_state<>'IN_FLIGHT') OR
 NOT (NEW.resolution_state=OLD.resolution_state OR
 (OLD.resolution_state='IN_FLIGHT' AND NEW.resolution_state IN ('RESTORE_OUTCOME_UNKNOWN','RESTORE_RECOVERY_CLAIMED')) OR
 (OLD.resolution_state='RESTORE_OUTCOME_UNKNOWN' AND NEW.resolution_state='RESTORE_RECOVERY_CLAIMED') OR
 (OLD.resolution_state='RESTORE_RECOVERY_CLAIMED' AND NEW.resolution_state IN ('RESTORE_CLEANUP_CLAIMED','RESTORE_RESOLVED')) OR
 (OLD.resolution_state='RESTORE_CLEANUP_CLAIMED' AND NEW.resolution_state='RESTORE_RESOLVED')) OR OLD.resolution_state='RESTORE_RESOLVED'
BEGIN SELECT RAISE(ABORT,'restore operation identity or lifecycle violation'); END;
`

const v5Schema = `
CREATE TABLE attempt_execution_budgets (
    attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    dispatch_operation_id TEXT NOT NULL UNIQUE REFERENCES dispatch_operations(operation_id) ON DELETE RESTRICT,
    origin_at TEXT NOT NULL,
    deadline_at TEXT NOT NULL,
    duration_ns INTEGER NOT NULL CHECK(duration_ns > 0),
    policy_ref TEXT NOT NULL CHECK(length(trim(policy_ref)) > 0),
    binding_basis TEXT NOT NULL CHECK(binding_basis IN ('SEND_CONFIRMATION_ATOMIC','LEGACY_OPERATOR_VERIFIED')),
    authorized_principal TEXT,
    evidence_ref TEXT,
    bound_at TEXT NOT NULL,
    CHECK ((binding_basis='SEND_CONFIRMATION_ATOMIC' AND authorized_principal IS NULL AND evidence_ref IS NULL)
        OR (binding_basis='LEGACY_OPERATOR_VERIFIED'
            AND authorized_principal IS NOT NULL AND evidence_ref IS NOT NULL
            AND length(trim(authorized_principal)) > 0 AND length(trim(evidence_ref)) > 0))
);
CREATE TRIGGER trg_execution_budget_immutable_update BEFORE UPDATE ON attempt_execution_budgets
BEGIN SELECT RAISE(ABORT,'execution budget immutable'); END;
CREATE TRIGGER trg_execution_budget_immutable_delete BEFORE DELETE ON attempt_execution_budgets
BEGIN SELECT RAISE(ABORT,'execution budget immutable'); END;
ALTER TABLE stop_operations ADD COLUMN initiating_failure_reason TEXT
    CHECK (initiating_failure_reason IS NULL OR initiating_failure_reason='TIMEOUT');
CREATE UNIQUE INDEX idx_one_live_stop_per_attempt ON stop_operations(attempt_id)
    WHERE purpose='RUNNING_ATTEMPT_STOP';
CREATE TRIGGER trg_stop_cause_immutable BEFORE UPDATE OF initiating_failure_reason ON stop_operations
WHEN NEW.initiating_failure_reason IS NOT OLD.initiating_failure_reason
BEGIN SELECT RAISE(ABORT,'stop cause immutable'); END;
CREATE TRIGGER trg_execution_budget_lineage BEFORE INSERT ON attempt_execution_budgets
WHEN NOT EXISTS (
    SELECT 1 FROM dispatch_operations d
    JOIN task_attempts a ON a.attempt_id=d.attempt_id
    JOIN tasks t ON t.task_id=a.task_id AND t.current_attempt=a.attempt_number
    WHERE d.operation_id=NEW.dispatch_operation_id AND d.attempt_id=NEW.attempt_id
      AND a.ended_at IS NULL
      AND ((NEW.binding_basis='SEND_CONFIRMATION_ATOMIC'
            AND t.state='DISPATCHED' AND d.stage='SEND_REQUESTED' AND d.confirmed_at IS NULL)
        OR (NEW.binding_basis='LEGACY_OPERATOR_VERIFIED'
            AND t.state IN ('DISPATCHED','RUNNING')
            AND d.stage='SEND_CONFIRMED' AND d.confirmed_at=NEW.origin_at))
)
BEGIN SELECT RAISE(ABORT,'execution budget lineage invalid'); END;
CREATE TRIGGER trg_dispatch_requires_execution_budget
BEFORE UPDATE OF stage,confirmed_at ON dispatch_operations
WHEN NEW.stage='SEND_CONFIRMED' AND
    (NEW.confirmed_at IS NULL OR NOT EXISTS (
        SELECT 1 FROM attempt_execution_budgets b
        WHERE b.dispatch_operation_id=NEW.operation_id
          AND b.attempt_id=NEW.attempt_id AND b.origin_at=NEW.confirmed_at))
BEGIN SELECT RAISE(ABORT,'send confirmation requires exact budget'); END;
CREATE TRIGGER trg_dispatch_confirmed_immutable
BEFORE UPDATE OF stage,confirmed_at ON dispatch_operations
WHEN OLD.stage='SEND_CONFIRMED' AND
    (NEW.stage<>'SEND_CONFIRMED' OR NEW.confirmed_at IS NOT OLD.confirmed_at)
BEGIN SELECT RAISE(ABORT,'confirmed send provenance immutable'); END;
`

const v6Schema = `
CREATE TABLE attempt_workspace_bindings (
    attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL CHECK (LENGTH(session_id) > 0),
    terminal_generation TEXT NOT NULL CHECK (LENGTH(terminal_generation) > 0),
    canonical_worktree_path TEXT NOT NULL CHECK (LENGTH(canonical_worktree_path) > 0),
    volume_serial_hex TEXT NOT NULL CHECK (
        LENGTH(volume_serial_hex) = 16 AND NOT (volume_serial_hex GLOB '*[^0-9a-f]*')
    ),
    file_id_hex TEXT NOT NULL CHECK (
        LENGTH(file_id_hex) = 32 AND NOT (file_id_hex GLOB '*[^0-9a-f]*')
    ),
    linked_gitdir_path TEXT NOT NULL CHECK (LENGTH(linked_gitdir_path) > 0),
    linked_gitdir_volume_serial_hex TEXT NOT NULL CHECK (
        LENGTH(linked_gitdir_volume_serial_hex) = 16 AND NOT (linked_gitdir_volume_serial_hex GLOB '*[^0-9a-f]*')
    ),
    linked_gitdir_file_id_hex TEXT NOT NULL CHECK (
        LENGTH(linked_gitdir_file_id_hex) = 32 AND NOT (linked_gitdir_file_id_hex GLOB '*[^0-9a-f]*')
    ),
    pinned_ao_commit TEXT NOT NULL CHECK (
        LENGTH(pinned_ao_commit) = 40 AND NOT (pinned_ao_commit GLOB '*[^0-9a-f]*')
    ),
    binding_state TEXT NOT NULL CHECK (
        binding_state IN ('ACTIVE', 'RETAINED_FOR_VERIFICATION', 'RELEASED', 'INVALIDATED')
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
    released_at_epoch_ms INTEGER NULL CHECK (
        released_at_epoch_ms IS NULL OR (
            typeof(released_at_epoch_ms) = 'integer' AND released_at_epoch_ms >= created_at_epoch_ms
        )
    ),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        (binding_state IN ('ACTIVE', 'RETAINED_FOR_VERIFICATION') AND released_at_epoch_ms IS NULL) OR
        (binding_state IN ('RELEASED', 'INVALIDATED') AND released_at_epoch_ms IS NOT NULL)
    )
);

CREATE TRIGGER trg_attempt_workspace_bindings_lineage_guard
BEFORE INSERT ON attempt_workspace_bindings
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id, contract_id in task_attempts or session in dispatch_operations with stage DISPATCH_BOUND')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        JOIN dispatch_operations d ON d.attempt_id = a.attempt_id
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
          AND d.session_id = NEW.session_id
          AND d.terminal_generation = NEW.terminal_generation
          AND d.stage = 'DISPATCH_BOUND'
    );
END;

CREATE TRIGGER trg_attempt_workspace_bindings_cas_guard
BEFORE UPDATE ON attempt_workspace_bindings
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in attempt_workspace_bindings')
    WHERE NEW.attempt_id != OLD.attempt_id
       OR NEW.task_id != OLD.task_id
       OR NEW.contract_id != OLD.contract_id
       OR NEW.session_id != OLD.session_id
       OR NEW.terminal_generation != OLD.terminal_generation
       OR NEW.canonical_worktree_path != OLD.canonical_worktree_path
       OR NEW.volume_serial_hex != OLD.volume_serial_hex
       OR NEW.file_id_hex != OLD.file_id_hex
       OR NEW.linked_gitdir_path != OLD.linked_gitdir_path
       OR NEW.linked_gitdir_volume_serial_hex != OLD.linked_gitdir_volume_serial_hex
       OR NEW.linked_gitdir_file_id_hex != OLD.linked_gitdir_file_id_hex
       OR NEW.pinned_ao_commit != OLD.pinned_ao_commit
       OR NEW.created_at_epoch_ms != OLD.created_at_epoch_ms;

    SELECT RAISE(ABORT, 'illegal state transition in attempt_workspace_bindings')
    WHERE NOT (
        (OLD.binding_state = 'ACTIVE' AND NEW.binding_state IN ('RETAINED_FOR_VERIFICATION', 'RELEASED', 'INVALIDATED')) OR
        (OLD.binding_state = 'RETAINED_FOR_VERIFICATION' AND NEW.binding_state IN ('RELEASED', 'INVALIDATED'))
    );
END;

CREATE TRIGGER trg_attempt_workspace_bindings_no_delete
BEFORE DELETE ON attempt_workspace_bindings
BEGIN
    SELECT RAISE(ABORT, 'attempt_workspace_bindings is immutable');
END;

CREATE TABLE worker_claims (
    claim_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    reported_head_sha TEXT NOT NULL CHECK (
        LENGTH(reported_head_sha) BETWEEN 7 AND 40
        AND NOT (reported_head_sha GLOB '*[^0-9a-f]*')
    ),
    payload_json TEXT NOT NULL CHECK (
        json_valid(payload_json) = 1
        AND json_type(payload_json, '$.claimed_files_changed') = 'array'
        AND json_type(payload_json, '$.tests') = 'array'
        AND json_type(payload_json, '$.textual_claims') = 'array'
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_worker_claims_lineage_guard
BEFORE INSERT ON worker_claims
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );
END;

CREATE TRIGGER trg_worker_claims_no_update
BEFORE UPDATE ON worker_claims
BEGIN
    SELECT RAISE(ABORT, 'worker_claims is immutable');
END;

CREATE TRIGGER trg_worker_claims_no_delete
BEFORE DELETE ON worker_claims
BEGIN
    SELECT RAISE(ABORT, 'worker_claims is immutable');
END;

CREATE TABLE review_integrity_holds (
    hold_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    hold_reason TEXT NOT NULL CHECK (
        hold_reason IN (
            'DIRTY_WORKTREE_DETECTED',
            'BUNDLE_HASH_CONFLICT',
            'INVARIANT_MISMATCH',
            'UNVERIFIED_CLAIM_DETECTED',
            'SECURITY_POLICY_VIOLATION'
        )
    ),
    hold_state TEXT NOT NULL CHECK (hold_state IN ('ACTIVE', 'RESOLVED')),
    diagnostic_fingerprint TEXT NOT NULL CHECK (
        LENGTH(diagnostic_fingerprint) = 64 AND NOT (diagnostic_fingerprint GLOB '*[^0-9a-f]*')
    ),
    occurrence_number INTEGER NOT NULL CHECK (
        typeof(occurrence_number) = 'integer' AND occurrence_number > 0
    ),
    rejection_audit_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
    resolution_audit_event_id TEXT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
    resolved_by_principal TEXT NULL CHECK (
        resolved_by_principal IS NULL OR LENGTH(TRIM(resolved_by_principal)) > 0
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
    resolved_at_epoch_ms INTEGER NULL CHECK (
        resolved_at_epoch_ms IS NULL OR (
            typeof(resolved_at_epoch_ms) = 'integer' AND resolved_at_epoch_ms >= created_at_epoch_ms
        )
    ),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    UNIQUE(attempt_id, hold_reason, diagnostic_fingerprint, occurrence_number),
    CHECK (
        (hold_state = 'ACTIVE' AND resolved_at_epoch_ms IS NULL AND resolution_audit_event_id IS NULL AND resolved_by_principal IS NULL) OR
        (hold_state = 'RESOLVED' AND resolved_at_epoch_ms IS NOT NULL AND resolution_audit_event_id IS NOT NULL AND resolved_by_principal IS NOT NULL AND LENGTH(TRIM(resolved_by_principal)) > 0)
    )
);

CREATE UNIQUE INDEX idx_review_integrity_holds_active_dedup
ON review_integrity_holds(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE';

CREATE TRIGGER trg_review_integrity_holds_lineage_guard
BEFORE INSERT ON review_integrity_holds
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );
END;

CREATE TRIGGER trg_review_integrity_holds_cas_guard
BEFORE UPDATE ON review_integrity_holds
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in review_integrity_holds')
    WHERE NEW.hold_id != OLD.hold_id
       OR NEW.task_id != OLD.task_id
       OR NEW.attempt_id != OLD.attempt_id
       OR NEW.contract_id != OLD.contract_id
       OR NEW.hold_reason != OLD.hold_reason
       OR NEW.diagnostic_fingerprint != OLD.diagnostic_fingerprint
       OR NEW.occurrence_number != OLD.occurrence_number
       OR NEW.rejection_audit_event_id != OLD.rejection_audit_event_id
       OR NEW.created_at_epoch_ms != OLD.created_at_epoch_ms;

    SELECT RAISE(ABORT, 'illegal hold state transition: ACTIVE only transitions to RESOLVED')
    WHERE NOT (OLD.hold_state = 'ACTIVE' AND NEW.hold_state = 'RESOLVED');
END;

CREATE TRIGGER trg_review_integrity_holds_no_delete
BEFORE DELETE ON review_integrity_holds
BEGIN
    SELECT RAISE(ABORT, 'review_integrity_holds is immutable');
END;
`

func migrate(ctx context.Context, db *sql.DB) error {
	return migrateWithSchemas(ctx, db, v1Schema, v2Schema, v3Schema, v4Schema, v5Schema, v6Schema)
}

func migrateWithSchemas(ctx context.Context, db *sql.DB, v1DDL, v2DDL, v3DDL string, v4DDLs ...string) error {
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
		userVersion = 2
	}

	// Apply V3 if userVersion is 2.
	if userVersion < 3 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: failed to begin v3 migration transaction: %w", err)
		}
		if _, err := tx.ExecContext(ctx, v3DDL); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to execute v3 migration DDL: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 3"); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to set PRAGMA user_version = 3: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: failed to commit v3 migration: %w", err)
		}
		userVersion = 3
	}

	// V3 is immutable history. Only normal Open supplies the approved V4 schema.
	if userVersion < 4 && len(v4DDLs) > 0 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: failed to begin v4 migration transaction: %w", err)
		}
		if _, err := tx.ExecContext(ctx, v4DDLs[0]); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to execute v4 migration DDL: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 4"); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to set PRAGMA user_version = 4: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: failed to commit v4 migration: %w", err)
		}
		userVersion = 4
	}

	if userVersion < 5 && len(v4DDLs) > 1 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: failed to begin v5 migration transaction: %w", err)
		}
		if _, err := tx.ExecContext(ctx, v4DDLs[1]); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to execute v5 migration DDL: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 5"); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to set PRAGMA user_version = 5: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: failed to commit v5 migration: %w", err)
		}
		userVersion = 5
	}

	if userVersion < 6 && len(v4DDLs) > 2 {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: failed to begin v6 migration transaction: %w", err)
		}
		if _, err := tx.ExecContext(ctx, v4DDLs[2]); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to execute v6 migration DDL: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 6"); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: failed to set PRAGMA user_version = 6: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: failed to commit v6 migration: %w", err)
		}
	}

	return nil
}
