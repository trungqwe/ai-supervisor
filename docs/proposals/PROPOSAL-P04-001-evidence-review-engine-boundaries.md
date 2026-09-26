# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Boundaries, and Verification Isolation

> **Proposal ID**: `PROPOSAL-P04-001`
> **Revision**: 12
> **Title**: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Audited Baseline**: `6e1993da150031a9465901a7019c71257de44312`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_11`
> **Supersedes**: `PROPOSAL-P04-001` Revision 11
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R11-001` through `P04-ARCH-R11-004` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_010.md`).
> **Requirement & Governance Note**: Concurrently submits `PROPOSAL-P04-002` (Revision 6) for formal NFR-008 performance SLA reconciliation. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until approved by External Supervisor.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 010 recorded four architectural findings requiring Revision 12 remediation:
1. `P04-ARCH-R11-001`: Canonical NFR-008 measures from worker completion. The internal supervisor interval (`evidence_finalized_at -> bundle_assembled_at`) must be designated the **ReviewBundle assembly diagnostic**, and NFR-008 compliance status must remain strictly **`UNVERIFIED`** in v1. Furthermore, audit events committed inside SQLite transactions cannot contain telemetry regarding the duration of their own future disk commit (`commit_duration_ms`).
2. `P04-ARCH-R11-002`: The verification lease model required refactoring to **append-only lease history** with `lease_id PRIMARY KEY`, partial unique indexes enforcing at most one `ACTIVE` lease per task and attempt, immutability triggers blocking `worker_id` changes, atomic reclaim, and a complete crash/replay/concurrent reclaim matrix.
3. `P04-ARCH-R11-003`: Elimination of blanket transition rules ("any non-REVIEWING state with a bundle to BLOCKED"), strict canonical TaskState preservation on invariant mismatch, and isolated diagnostic failure transactions for dirty worktree and bundle rejection.
4. `P04-ARCH-R11-004`: Strict stream limit inequality (`hard_safety_limit_bytes > capture_limit_bytes`), a bounded overflow reader algorithm reading up to `hard + 1` bytes, and mutual exclusion precedence between stream states.

---

## 2. Model 1 Persistence Ownership & Subtask Division

To maintain strict architectural boundaries, the four P04 subtasks are assigned non-overlapping responsibilities:

1. **Subtask P04A (Report Intake & Workspace Verification)**:
   - Owns Schema Migration v6 (`attempt_workspace_bindings` and `worker_claims`).
   - Owns Transaction A (`RUNNING -> REPORT_READY`), physical worktree handle verification (`FileIdInfo`), and clean worktree validation.
2. **Subtask P04B (Hardened Git Evidence Collector)**:
   - Pure in-memory collector with **ZERO SQLite writes**.
   - Executes strictly read-only Git commands against verified snapshots to produce in-memory `GitEvidence`.
3. **Subtask P04C (Isolated Verification Runner & AppContainer Boundary)**:
   - Pure in-memory execution engine with **ZERO SQLite writes**.
   - Spawns verification commands sequentially in Windows AppContainers with handle inheritance whitelisting and network denial, producing in-memory `TestEvidence`.
4. **Subtask P04D (Pipeline Orchestrator, CAS, and CAS Persistence Orchestrator)**:
   - **SOLE SQLite persistence authority** for Schema Migration v9 (`task_verification_leases`, `evidence_sets`, `review_artifacts`, and `review_bundles`).
   - Owns Content-Addressed Store directory structure (`artifacts/<first-two-hex>/<captured_sha256>`).
   - Owns lease acquisition, TTL monitoring, atomic reclaim, and completion.
   - Owns Transaction B (`REPORT_READY -> EVIDENCE_READY`) and Transaction C (`EVIDENCE_READY -> REVIEWING`).
   - Owns all proposed audit events.

---

## 3. Worktree Validation, Verbatim Claim Mapping & Dirty Worktree Rollback

### 3.1. Verbatim Claim Ingestion
- `worker_claims.reported_head_sha` accepts verbatim 7-40 hex characters matching `docs/schemas/worker-report.schema.json`.
- Supervisor independently discovers actual verified HEAD SHA via `git rev-parse HEAD^{commit}`.

### 3.2. Clean Worktree Verification
- Evaluated via `git status --porcelain=v1 --untracked-files=all` and `git rev-parse --absolute-git-dir`.
- Any staged change, unstaged change, untracked file, or submodule modification rejects intake.

### 3.3. Dirty Worktree Rollback & Isolated Diagnostic Audit
- If a dirty worktree is detected:
  * Transaction A rolls back completely.
  * `tasks.state` remains `RUNNING` (preserved; attempt report is rejected).
  * In a separate diagnostic transaction executed after rollback, diagnostic audit event `EVIDENCE_COLLECTION_FAILED` is appended with `failure_reason = 'DIRTY_WORKTREE_DETECTED'` and an `idempotency_key` (`audit:<attempt_id>:DIRTY_WORKTREE_DETECTED`).
  * If appending this diagnostic audit event fails, a compound failure error is returned to the caller, and the system explicitly does not claim the audit event was recorded.

---

## 4. Append-Only Lease History, CAS Reclaim & Crash Matrix

### 4.1. Append-Only Lease Schema
- `lease_id TEXT PRIMARY KEY`.
- All identity, timestamp, and token columns are immutable: `task_id`, `attempt_id`, `contract_id`, `worker_id`, `fencing_token`, `acquired_at_epoch_ms`, `ttl_seconds`, `expires_at_epoch_ms`, `predecessor_lease_id`.
- `UNIQUE(attempt_id, fencing_token)` ensures monotonically increasing tokens.

### 4.2. Single Active Lease Guarantee
- Partial unique indexes guarantee at most one `ACTIVE` lease per task and attempt:
  `CREATE UNIQUE INDEX idx_leases_single_active_attempt ON task_verification_leases(attempt_id) WHERE state = 'ACTIVE';`
  `CREATE UNIQUE INDEX idx_leases_single_active_task ON task_verification_leases(task_id) WHERE state = 'ACTIVE';`

### 4.3. CAS State Transitions & Atomic Reclaim
- `ACTIVE -> COMPLETED | EXPIRED | REVOKED`.
- `EXPIRED -> RECLAIMED`.
- Immutability trigger strictly prevents modifying `worker_id` or tokens during terminal transitions, and prevents `RECLAIMED -> ACTIVE` on the same row.
- Atomic reclaim: (1) Predecessor is updated: `state = 'RECLAIMED', released_at_epoch_ms = now` where `state = 'EXPIRED'`; (2) New lease is inserted with `state = 'ACTIVE'`, `fencing_token = predecessor.fencing_token + 1`, and `predecessor_lease_id = predecessor.lease_id`.

### 4.4. Lease Crash, Replay & Concurrent Reclaim Matrix

| Initial State | Event / Trigger | Preconditions | Database Action | Outcome & Invariants |
| :--- | :--- | :--- | :--- | :--- |
| No lease | Acquisition | Attempt in `REPORT_READY` | `INSERT INTO task_verification_leases (..., state='ACTIVE', token=1)` | Success. Single active lease guaranteed by partial index. |
| `ACTIVE` | Concurrent Acquisition | Another worker attempts acquisition | `INSERT INTO task_verification_leases (..., state='ACTIVE')` | **Rejected**. Unique index conflict on `idx_leases_single_active_attempt`. |
| `ACTIVE` | Normal Completion | Verification succeeds before TTL | `UPDATE ... SET state='COMPLETED', released_at=? WHERE state='ACTIVE' AND worker_id=?` | Success. Lease marked `COMPLETED`. Next stage unlocked. |
| `ACTIVE` | Expiration | Wall clock exceeds `expires_at` | `UPDATE ... SET state='EXPIRED', released_at=? WHERE state='ACTIVE'` | Success. Lease marked `EXPIRED`. Active lease slot vacated. |
| `EXPIRED` | Atomic Reclaim | Old worker confirmed dead | Tx: `UPDATE ... SET state='RECLAIMED'` then `INSERT ... state='ACTIVE', token=token+1, pred=old_id` | Success. Old lease becomes `RECLAIMED`; new lease becomes `ACTIVE`. |
| `EXPIRED` | Concurrent Reclaim Race | Two workers attempt reclaim | Both attempt `UPDATE ... SET state='RECLAIMED' WHERE state='EXPIRED'` | First commits; second sees 0 rows affected and aborts without inserting duplicate. |
| `RECLAIMED` | Late Stalled Worker Write | Stalled worker attempts release | `UPDATE ... WHERE lease_id=? AND state='ACTIVE' AND token=old_token` | **Zero rows updated**. Stalled worker detects CAS failure and aborts. |
| `ACTIVE` | Daemon Crash & Restart | Daemon crashes mid-verification | Recovery scan detects expired lease or dead worker PID | Recovery scanner marks `EXPIRED`. Allows subsequent atomic reclaim. |

---

## 5. ReviewBundle Latency Semantics & Telemetry Separation

### 5.1. ReviewBundle Assembly Diagnostic
- `evidence_finalized_at_epoch_ms`: Application timestamp chosen before Transaction B commit and persisted durably by Transaction B.
- `bundle_assembled_at_epoch_ms`: Application timestamp captured upon ReviewBundle payload assembly, RFC 8785 canonicalization, and validation, before initiating Transaction C commit.
- `compilation_latency_ms`: The deterministic difference representing the assembly diagnostic latency:
  `compilation_latency_ms = bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms`.

### 5.2. Canonical NFR-008 Compliance Held as UNVERIFIED
- Canonical NFR-008 measures from worker completion. The supervisor control plane cannot self-declare NFR-008 as met using the internal assembly diagnostic interval.
- `nfr008_compliance_status TEXT NOT NULL CHECK (nfr008_compliance_status = 'UNVERIFIED')` is strictly enforced at the SQLite layer.

### 5.3. Telemetry Separation
- `REVIEW_BUNDLE_GENERATED` is inserted inside Transaction C without `commit_duration_ms`.
- `commit_duration_ms` is measured using in-process monotonic clock after `tx.Commit()` returns, purely as best-effort in-process telemetry (structured logging/metrics).

---

## 6. TaskState Discipline & Invariant Mismatch Protocol

### 6.1. Elimination of Blanket Transition Rules
- The non-canonical blanket rule transitioning "any non-REVIEWING state with a bundle to BLOCKED" is eliminated.

### 6.2. Invariant Mismatch Handling
- If an invariant mismatch is detected (e.g. existing bundle found while task is in an unexpected state):
  * TaskState remains preserved in its current state.
  * Automated review approval and intake admission are locked.
  * Escalation requires human supervisor reconciliation (`HUMAN_SUPERVISOR_RECONCILIATION_REQUIRED`).
  * Zero non-canonical state transitions are performed.

### 6.3. Isolated Diagnostic Transactions on Rejection
- When ReviewBundle compilation or hash verification fails:
  * Transaction C rolls back completely.
  * TaskState remains preserved in `EVIDENCE_READY`.
  * Proposed audit event `REVIEW_BUNDLE_COMPILATION_REJECTED` is appended in a separate diagnostic transaction after rollback using an idempotency key (`audit:<attempt_id>:BUNDLE_HASH_CONFLICT`).
  * If the diagnostic transaction fails, a compound failure error is returned without claiming audit was recorded. Automated review approval remains locked.

---

## 7. Content-Addressed Store, Stream Limits & Artifacts Layout

### 7.1. Strict Inequality of Stream Limits
- `CHECK (hard_safety_limit_bytes > capture_limit_bytes)` is enforced at the database level.

### 7.2. Bounded Overflow Reader Algorithm
- The supervisor stream reader reads up to at most `hard_safety_limit_bytes + 1` bytes.
- Observing byte `hard_safety_limit_bytes + 1` triggers immediate process termination and terminates stream reading.

### 7.3. Mutually Exclusive Stream States & Precedence
- Precedence 1: Hard limit breach -> `HARD_LIMIT_TERMINATED` (`total = hard + 1`, `captured = capture`, `is_truncated = 1`, `full_stream_sha256 = NULL`).
- Precedence 2: Timeout abort -> `TIMEOUT_ABORTED` (`total <= hard`, `captured = MIN(total, capture)`, `is_truncated = 1`, `full_stream_sha256 = NULL`).
- Precedence 3: Clean EOF:
  * `total <= capture` -> `COMPLETE_EOF` (`captured = total`, `is_truncated = 0`, `full_stream_sha256 = captured_sha256 IS NOT NULL`).
  * `capture < total <= hard` -> `TRUNCATED_AT_CAPTURE_LIMIT` (`captured = capture`, `is_truncated = 1`, `full_stream_sha256 IS NOT NULL`).

---

## 8. Complete Database DDL Schemas & Triggers

### 8.1. Schema v6 (Owned by Subtask P04A)

```sql
CREATE TABLE attempt_workspace_bindings (
    binding_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL,
    terminal_generation INTEGER NOT NULL CHECK (terminal_generation >= 0),
    workspace_path TEXT NOT NULL,
    volume_serial_hex TEXT NOT NULL CHECK (
        LENGTH(volume_serial_hex) = 16 AND NOT (volume_serial_hex GLOB '*[^0-9a-f]*')
    ),
    file_id_hex TEXT NOT NULL CHECK (
        LENGTH(file_id_hex) = 32 AND NOT (file_id_hex GLOB '*[^0-9a-f]*')
    ),
    linked_gitdir_path TEXT NOT NULL,
    linked_gitdir_volume_serial_hex TEXT NOT NULL CHECK (
        LENGTH(linked_gitdir_volume_serial_hex) = 16 AND NOT (linked_gitdir_volume_serial_hex GLOB '*[^0-9a-f]*')
    ),
    linked_gitdir_file_id_hex TEXT NOT NULL CHECK (
        LENGTH(linked_gitdir_file_id_hex) = 32 AND NOT (linked_gitdir_file_id_hex GLOB '*[^0-9a-f]*')
    ),
    pinned_ao_commit TEXT NOT NULL CHECK (
        LENGTH(pinned_ao_commit) = 40 AND NOT (pinned_ao_commit GLOB '*[^0-9a-f]*')
    ),
    binding_status TEXT NOT NULL CHECK (
        binding_status IN ('ACTIVE', 'RETAINED_FOR_VERIFICATION', 'RELEASED', 'INVALIDATED')
    ),
    bound_at TEXT NOT NULL CHECK (LENGTH(bound_at) > 0)
);

CREATE TRIGGER trg_attempt_workspace_bindings_lineage_guard
BEFORE INSERT ON attempt_workspace_bindings
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
    );
    SELECT RAISE(ABORT, 'lineage mismatch: session_id does not match dispatch_operations')
    WHERE NOT EXISTS (
        SELECT 1 FROM dispatch_operations d
        WHERE d.attempt_id = NEW.attempt_id
          AND d.task_id = NEW.task_id
          AND d.session_id = NEW.session_id
    );
END;

CREATE TRIGGER trg_attempt_workspace_bindings_cas_guard
BEFORE UPDATE ON attempt_workspace_bindings
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in attempt_workspace_bindings')
    WHERE NEW.binding_id != OLD.binding_id
       OR NEW.task_id != OLD.task_id
       OR NEW.attempt_id != OLD.attempt_id
       OR NEW.session_id != OLD.session_id
       OR NEW.terminal_generation != OLD.terminal_generation
       OR NEW.workspace_path != OLD.workspace_path
       OR NEW.volume_serial_hex != OLD.volume_serial_hex
       OR NEW.file_id_hex != OLD.file_id_hex
       OR NEW.linked_gitdir_path != OLD.linked_gitdir_path
       OR NEW.linked_gitdir_volume_serial_hex != OLD.linked_gitdir_volume_serial_hex
       OR NEW.linked_gitdir_file_id_hex != OLD.linked_gitdir_file_id_hex
       OR NEW.pinned_ao_commit != OLD.pinned_ao_commit
       OR NEW.bound_at != OLD.bound_at;

    SELECT RAISE(ABORT, 'illegal binding status transition')
    WHERE NOT (
        (OLD.binding_status = 'ACTIVE' AND NEW.binding_status IN ('RETAINED_FOR_VERIFICATION', 'RELEASED', 'INVALIDATED')) OR
        (OLD.binding_status = 'RETAINED_FOR_VERIFICATION' AND NEW.binding_status IN ('RELEASED', 'INVALIDATED'))
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
    claimed_files_json TEXT NOT NULL CHECK (
        json_valid(claimed_files_json) = 1 AND
        json_type(claimed_files_json) = 'array'
    ),
    reported_head_sha TEXT NOT NULL CHECK (
        LENGTH(reported_head_sha) BETWEEN 7 AND 40 AND
        NOT (reported_head_sha GLOB '*[^0-9a-f]*')
    ),
    claim_payload_json TEXT NOT NULL CHECK (
        json_valid(claim_payload_json) = 1 AND
        json_type(claim_payload_json) = 'object'
    ),
    persisted_at TEXT NOT NULL CHECK (LENGTH(persisted_at) > 0),
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
```

### 8.2. Schema v9 (Owned by Subtask P04D)

```sql
CREATE TABLE task_verification_leases (
    lease_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'COMPLETED', 'EXPIRED', 'RECLAIMED', 'REVOKED')),
    worker_id TEXT NOT NULL CHECK (LENGTH(worker_id) > 0),
    ttl_seconds INTEGER NOT NULL CHECK (ttl_seconds BETWEEN 1 AND 600),
    acquired_at_epoch_ms INTEGER NOT NULL CHECK (acquired_at_epoch_ms > 0),
    expires_at_epoch_ms INTEGER NOT NULL,
    released_at_epoch_ms INTEGER NULL,
    predecessor_lease_id TEXT NULL REFERENCES task_verification_leases(lease_id) ON DELETE RESTRICT,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    UNIQUE(attempt_id, fencing_token),
    CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000)),
    CHECK (
        (state = 'ACTIVE' AND released_at_epoch_ms IS NULL) OR
        (state IN ('COMPLETED', 'EXPIRED', 'RECLAIMED', 'REVOKED') AND released_at_epoch_ms IS NOT NULL AND released_at_epoch_ms >= acquired_at_epoch_ms)
    )
);

CREATE UNIQUE INDEX idx_leases_single_active_attempt ON task_verification_leases(attempt_id) WHERE state = 'ACTIVE';
CREATE UNIQUE INDEX idx_leases_single_active_task ON task_verification_leases(task_id) WHERE state = 'ACTIVE';

CREATE TRIGGER trg_task_verification_leases_lineage_guard
BEFORE INSERT ON task_verification_leases
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );

    SELECT RAISE(ABORT, 'predecessor lease mismatch: predecessor must be RECLAIMED and have lower fencing token')
    WHERE NEW.predecessor_lease_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM task_verification_leases p
        WHERE p.lease_id = NEW.predecessor_lease_id
          AND p.task_id = NEW.task_id
          AND p.attempt_id = NEW.attempt_id
          AND p.state = 'RECLAIMED'
          AND p.fencing_token < NEW.fencing_token
    );
END;

CREATE TRIGGER trg_task_verification_leases_cas_guard
BEFORE UPDATE ON task_verification_leases
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in task_verification_leases')
    WHERE NEW.lease_id != OLD.lease_id
       OR NEW.task_id != OLD.task_id
       OR NEW.attempt_id != OLD.attempt_id
       OR NEW.contract_id != OLD.contract_id
       OR NEW.worker_id != OLD.worker_id
       OR NEW.fencing_token != OLD.fencing_token
       OR NEW.acquired_at_epoch_ms != OLD.acquired_at_epoch_ms
       OR NEW.ttl_seconds != OLD.ttl_seconds
       OR NEW.expires_at_epoch_ms != OLD.expires_at_epoch_ms
       OR (NEW.predecessor_lease_id IS NOT OLD.predecessor_lease_id);

    SELECT RAISE(ABORT, 'illegal lease state transition')
    WHERE NOT (
        (OLD.state = 'ACTIVE' AND NEW.state IN ('COMPLETED', 'EXPIRED', 'REVOKED')) OR
        (OLD.state = 'EXPIRED' AND NEW.state = 'RECLAIMED')
    );
END;

CREATE TRIGGER trg_task_verification_leases_no_delete
BEFORE DELETE ON task_verification_leases
BEGIN
    SELECT RAISE(ABORT, 'task_verification_leases is immutable');
END;

CREATE TABLE evidence_sets (
    evidence_set_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    git_evidence_json TEXT NOT NULL CHECK (
        json_valid(git_evidence_json) = 1 AND
        json_type(git_evidence_json, '$.actual_changed_files') = 'array'
    ),
    test_evidence_json TEXT NOT NULL CHECK (
        json_valid(test_evidence_json) = 1 AND
        json_type(test_evidence_json, '$.executed_commands') = 'array'
    ),
    policy_findings_json TEXT NOT NULL CHECK (json_valid(policy_findings_json) = 1),
    unverified_claims_json TEXT NOT NULL CHECK (json_valid(unverified_claims_json) = 1),
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (evidence_finalized_at_epoch_ms > 0),
    collected_at TEXT NOT NULL CHECK (LENGTH(collected_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_evidence_sets_lineage_guard
BEFORE INSERT ON evidence_sets
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

CREATE TRIGGER trg_evidence_sets_no_update
BEFORE UPDATE ON evidence_sets
BEGIN
    SELECT RAISE(ABORT, 'evidence_sets is immutable');
END;

CREATE TRIGGER trg_evidence_sets_no_delete
BEFORE DELETE ON evidence_sets
BEGIN
    SELECT RAISE(ABORT, 'evidence_sets is immutable');
END;

CREATE TABLE review_artifacts (
    artifact_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    artifact_type TEXT NOT NULL CHECK (
        artifact_type IN ('VERIFICATION_LOG', 'GIT_DIFF', 'TEST_REPORT', 'WORKER_STDOUT', 'WORKER_STDERR')
    ),
    media_type TEXT NOT NULL CHECK (LENGTH(media_type) > 0),
    encoding TEXT NOT NULL CHECK (encoding IN ('identity', 'gzip')),
    canonical_relative_path TEXT NOT NULL,
    captured_sha256 TEXT NOT NULL CHECK (
        LENGTH(captured_sha256) = 64 AND NOT (captured_sha256 GLOB '*[^0-9a-f]*')
    ),
    full_stream_sha256 TEXT NULL CHECK (
        full_stream_sha256 IS NULL OR (
            LENGTH(full_stream_sha256) = 64 AND NOT (full_stream_sha256 GLOB '*[^0-9a-f]*')
        )
    ),
    capture_limit_bytes INTEGER NOT NULL CHECK (capture_limit_bytes > 0),
    hard_safety_limit_bytes INTEGER NOT NULL CHECK (hard_safety_limit_bytes > capture_limit_bytes),
    captured_bytes INTEGER NOT NULL CHECK (captured_bytes >= 0 AND captured_bytes <= capture_limit_bytes),
    total_observed_bytes INTEGER NOT NULL CHECK (total_observed_bytes >= captured_bytes),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1)),
    stream_state TEXT NOT NULL CHECK (
        stream_state IN ('COMPLETE_EOF', 'TRUNCATED_AT_CAPTURE_LIMIT', 'HARD_LIMIT_TERMINATED', 'TIMEOUT_ABORTED')
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (created_at_epoch_ms > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256 AND
        canonical_relative_path NOT GLOB '*:*' AND
        canonical_relative_path NOT GLOB '*\*' AND
        canonical_relative_path NOT GLOB '*..*' AND
        canonical_relative_path NOT GLOB '*//*'
    ),
    CHECK (
        (stream_state = 'COMPLETE_EOF' AND is_truncated = 0 AND full_stream_sha256 IS NOT NULL AND full_stream_sha256 = captured_sha256 AND captured_bytes = total_observed_bytes AND total_observed_bytes <= capture_limit_bytes)
        OR (stream_state = 'TRUNCATED_AT_CAPTURE_LIMIT' AND is_truncated = 1 AND full_stream_sha256 IS NOT NULL AND captured_bytes = capture_limit_bytes AND total_observed_bytes > capture_limit_bytes AND total_observed_bytes <= hard_safety_limit_bytes)
        OR (stream_state = 'HARD_LIMIT_TERMINATED' AND is_truncated = 1 AND full_stream_sha256 IS NULL AND captured_bytes = capture_limit_bytes AND total_observed_bytes = hard_safety_limit_bytes + 1)
        OR (stream_state = 'TIMEOUT_ABORTED' AND is_truncated = 1 AND full_stream_sha256 IS NULL AND total_observed_bytes <= hard_safety_limit_bytes AND (
            (total_observed_bytes <= capture_limit_bytes AND captured_bytes = total_observed_bytes) OR
            (total_observed_bytes > capture_limit_bytes AND captured_bytes = capture_limit_bytes)
        ))
    )
);

CREATE TRIGGER trg_review_artifacts_lineage_guard
BEFORE INSERT ON review_artifacts
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match task_id, attempt_id, contract_id in evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.task_id = NEW.task_id
          AND e.attempt_id = NEW.attempt_id
          AND e.contract_id = NEW.contract_id
    );
END;

CREATE TRIGGER trg_review_artifacts_no_update
BEFORE UPDATE ON review_artifacts
BEGIN
    SELECT RAISE(ABORT, 'review_artifacts is immutable');
END;

CREATE TRIGGER trg_review_artifacts_no_delete
BEFORE DELETE ON review_artifacts
BEGIN
    SELECT RAISE(ABORT, 'review_artifacts is immutable');
END;

CREATE TABLE review_bundles (
    bundle_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL UNIQUE REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    bundle_payload_json TEXT NOT NULL CHECK (
        json_valid(bundle_payload_json) = 1 AND
        json_type(bundle_payload_json, '$.worker_claims') = 'object' AND
        json_type(bundle_payload_json, '$.actual_git_evidence') = 'object'
    ),
    bundle_hash TEXT NOT NULL CHECK (
        LENGTH(bundle_hash) = 64 AND NOT (bundle_hash GLOB '*[^0-9a-f]*')
    ),
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (evidence_finalized_at_epoch_ms > 0),
    bundle_assembled_at_epoch_ms INTEGER NOT NULL CHECK (bundle_assembled_at_epoch_ms >= evidence_finalized_at_epoch_ms),
    compilation_latency_ms INTEGER NOT NULL CHECK (
        compilation_latency_ms >= 0 AND
        compilation_latency_ms = (bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms)
    ),
    latency_measurement_status TEXT NOT NULL CHECK (
        latency_measurement_status IN ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART')
    ),
    nfr008_compliance_status TEXT NOT NULL CHECK (
        nfr008_compliance_status = 'UNVERIFIED'
    ),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_review_bundles_lineage_guard
BEFORE INSERT ON review_bundles
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match attempt_id in evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.attempt_id = NEW.attempt_id
          AND e.task_id = NEW.task_id
          AND e.contract_id = NEW.contract_id
          AND e.evidence_finalized_at_epoch_ms = NEW.evidence_finalized_at_epoch_ms
    );
END;

CREATE TRIGGER trg_review_bundles_no_update
BEFORE UPDATE ON review_bundles
BEGIN
    SELECT RAISE(ABORT, 'review_bundles is immutable');
END;

CREATE TRIGGER trg_review_bundles_no_delete
BEFORE DELETE ON review_bundles
BEGIN
    SELECT RAISE(ABORT, 'review_bundles is immutable');
END;
```

---

## 9. Proposed Audit Events Registry

The following audit event literals are proposed for Phase P04:
1. `WORKSPACE_BINDING_CREATED`: Emitted in Transaction A upon physical handle verification.
2. `VERIFICATION_LEASE_ACQUIRED`: Emitted by P04D upon lease acquisition or atomic reclaim.
3. `VERIFICATION_LEASE_RELEASED`: Emitted by P04D upon lease completion or expiration.
4. `REVIEW_BUNDLE_GENERATED`: Emitted inside Transaction C upon bundle persistence. Includes `compilation_latency_ms` (ReviewBundle assembly diagnostic) and `nfr008_compliance_status = 'UNVERIFIED'`. (Does not include `commit_duration_ms`).
5. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Emitted in a separate diagnostic transaction after Transaction C rollback on validation or hash conflict.
6. `EVIDENCE_COLLECTION_FAILED`: Emitted in a separate diagnostic transaction after Transaction A or B rollback (e.g. `DIRTY_WORKTREE_DETECTED`).

---

## 10. Traceability and Governance Roadmap

- Concurrently submits `PROPOSAL-P04-002` (Revision 6) for formal NFR-008 performance SLA reconciliation.
- Pre-contract documentation remains strictly decoupled from implementation until formal External Supervisor approval of ADR-018.
- Zero Go code or Task Contract release authorized.
