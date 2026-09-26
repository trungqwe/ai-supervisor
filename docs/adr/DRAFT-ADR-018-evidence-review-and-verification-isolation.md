# ADR-018: Evidence & Review Engine Architecture, Execution Isolation, and Verification Governance

> **Status**: DRAFT (`REVISION_11_REQUIRED` / Remediated to Revision 11)
> **Date**: 2026-09-26
> **Audited Baseline**: `cd0417e641468abfac254cc57cca29be54d1e8e1`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_10`
> **Deciders**: AI Engineering Supervisor Architecture Council, External Supervisor
> **Related Architecture**: `docs/04_ARCHITECTURE.md` (Section 7), `docs/05_DOMAIN_MODEL.md`, `docs/10_REVIEW_BUNDLE.md`
> **Related Requirements**: `docs/02_REQUIREMENTS.md` (FR-008, NFR-008 via PROPOSAL-P04-002)
> **Supersedes**: `DRAFT-ADR-018` Revision 10
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R10-001` through `P04-ARCH-R10-005` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_009.md`).

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 009 recorded five architectural findings requiring Revision 11 remediation:
1. `P04-ARCH-R10-001`: Regression in DDL lineage columns, composite foreign keys, immutability triggers, and exact content-address artifact path equality.
2. `P04-ARCH-R10-002`: Violation of Model 1 persistence ownership (incorrectly attributing Schema v9 to Subtask P04C instead of P04D).
3. `P04-ARCH-R10-003`: Ambiguity in NFR-008 measurement boundaries and lack of strict truth table constraints.
4. `P04-ARCH-R10-004`: Lack of mathematical enforcement for lease TTL (`expires_at_epoch_ms = acquired_at_epoch_ms + ttl_seconds * 1000`) and stream state combinations.
5. `P04-ARCH-R10-005`: Use of non-canonical compound state labels and unregistered audit literals.

This ADR formalizes the architectural decisions and complete schema contracts governing Phase P04.

---

## 2. Decision Drivers

1. **Evidence Integrity & Non-Repudiation**: The supervisor must independently corroborate worker claims using immutable source snapshots and sandboxed execution without trusting worker memory or worker assertions.
2. **Canonical Schema Alignment & Lineage Guarantees**: All persistence contracts must enforce task, attempt, and contract lineage via composite foreign keys and SQLite triggers, maintaining strict compliance with `worker-report.schema.json`, `review-bundle.schema.json`, and `docs/06_WORKFLOW_STATE_MACHINE.md`.
3. **Model 1 Persistence Ownership**: Subtasks P04B and P04C must remain strictly in-memory collectors with zero SQLite writes, concentrating all Schema v9 mutations, leases, and CAS transactions in Subtask P04D.
4. **Crash Consistency & Deadlock Elimination**: Database transactions and latency metrics must support crash recovery across daemon restarts without permanent deadlocks.
5. **Clean Worktree Guarantee**: Worker outputs must be fully committed; dirty staged, unstaged, or untracked state must fail closed.
6. **Execution Containment**: Verification tests must run under OS-level isolation (Windows AppContainers) with explicit handle inheritance and network denial.

---

## 3. Considered Options

- **Option 1**: Monolithic in-process execution directly in worker worktree. (Rejected: Vulnerable to TOCTOU, handle leaks, network escape, and worktree contamination).
- **Option 2**: Ephemeral Git branches in worker repository without AppContainer sandboxing. (Rejected: Test execution could compromise host OS; non-isolated processes lack network denial).
- **Option 3 (Selected - Model 1 Architecture)**: Decoupled verification pipeline with Windows AppContainer isolation, deterministic source snapshot extraction, strict clean worktree enforcement, pure in-memory collectors (P04B, P04C), and centralized CAS orchestration in P04D.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority, Immutable Bindings & Canonical WorkerClaim (P04-ARCH-R10-001)

1. **Two-Transaction Workspace Lifecycle**:
   - **Binding Creation (P04A)**: Binds `attempt_id` to the physical worktree (`FileIdInfo`) and linked gitdir identity prior to dispatch, in state `ACTIVE`.
   - **Report Intake Transaction A (P04A)**: Triggered upon worker completion. Validates worktree identity, verifies clean worktree/index via porcelain check, validates `WorkerReport` via JSON schema in Go, canonicalizes via RFC 8785 JCS, inserts into `worker_claims`, updates binding to `RETAINED_FOR_VERIFICATION` via CAS, and transitions `tasks.state`: `RUNNING -> REPORT_READY`.
2. **Verbatim Head SHA & Lineage Invariants**:
   - `worker_claims.reported_head_sha` stores the exact reported value (`CHECK (LENGTH(reported_head_sha) BETWEEN 7 AND 40 AND NOT (reported_head_sha GLOB '*[^0-9a-f]*'))`).
   - The verified `actual_head_sha` belongs strictly to Evidence (`evidence_sets.git_evidence_json`), never in `worker_claims`.
   - `worker_claims` includes `contract_id` with composite lineage triggers and immutable update/delete triggers.
3. **Canonical Mapping**:
   - `WorkerReport.head_sha` -> `WorkerClaim.reported_head_sha`
   - `WorkerReport.files_changed` -> `WorkerClaim.claimed_files_changed`
   - `WorkerReport.tests` -> `WorkerClaim.claimed_tests`
   - `WorkerReport.worker_claims` (array of strings) -> `WorkerClaim.textual_claims`
   - `WorkerReport.build_status` -> `WorkerClaim.build_status`
   - Canonical `WorkerClaim` object is embedded directly into ReviewBundle `worker_claims`.

```sql
-- Schema v6: attempt_workspace_bindings (Owned by Subtask P04A)
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
    created_at_epoch_ms INTEGER NOT NULL CHECK (created_at_epoch_ms > 0),
    released_at_epoch_ms INTEGER NULL,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        (binding_state IN ('ACTIVE', 'RETAINED_FOR_VERIFICATION') AND released_at_epoch_ms IS NULL) OR
        (binding_state IN ('RELEASED', 'INVALIDATED') AND released_at_epoch_ms IS NOT NULL AND released_at_epoch_ms >= created_at_epoch_ms)
    )
);

CREATE TRIGGER trg_attempt_workspace_bindings_lineage_guard
BEFORE INSERT ON attempt_workspace_bindings
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id, contract_id in task_attempts or session in dispatch_operations')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        JOIN dispatch_operations d ON d.attempt_id = a.attempt_id
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
          AND d.session_id = NEW.session_id
          AND d.terminal_generation = NEW.terminal_generation
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

-- Schema v6: worker_claims (Owned by Subtask P04A)
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
    created_at_epoch_ms INTEGER NOT NULL CHECK (created_at_epoch_ms > 0),
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

### Decision 2: Hardened In-Memory Git Collector & Clean Worktree Allowlist (P04-ARCH-R10-001)

1. **Clean Worktree Policy (v1)**:
   - Worker must commit all changes; workspace worktree and index must be clean at report intake.
   - Transaction A rolls back if staged, unstaged, or untracked changes exist; task state remains `RUNNING` preserved.
2. **Hardened Allowlist**:
   - `git status --porcelain=v1 -z --untracked-files=all`: Evaluates clean worktree and untracked files.
   - `git diff-index --quiet HEAD --`: Evaluates clean index state.
   - `git rev-parse --verify --quiet <ref>^{commit}`: Resolves commit SHA.
   - `git diff --raw -z --no-renames --no-ext-diff <base> <head> --`: Parses modified files.
   - `git diff --numstat --no-renames <base> <head> --`: Computes diff statistics.
   - `git rev-list --count <base>..<head>`: Validates commit distance.
   - `git ls-tree -rz --full-tree <head>`: Walks commit tree for snapshot manifest.
   - `git cat-file --batch`: Streams file contents for snapshot extraction.
   - `git rev-parse --git-path index`: Resolves linked worktree index file.
   - `git rev-parse --absolute-git-dir`: Resolves primary repository git directory (restored).
3. **Execution Environment**:
   - Strictly isolated environment variables (`GIT_DIR`, `GIT_WORK_TREE`, `GIT_OPTIONAL_LOCKS=0`, clean system environment, no user config).

### Decision 3: Windows Verification Isolation Security Boundary & Job Containment

1. **Explicit Handle Inheritance via Attribute Lists**:
   - `bInheritHandles = TRUE` in `CreateProcessW`.
   - `STARTUPINFOEXW` configures `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` passing *only* the specific write handles for stdout, stderr, and the read handle for stdin. All database and network handles remain non-inherited.
2. **Atomic Job Object Assignment**:
   - `PROC_THREAD_ATTRIBUTE_JOB_LIST` atomically assigns the child process to a Windows Job Object upon creation.
   - `JOB_OBJECT_LIMIT_BREAKAWAY_OK` is explicitly omitted, preventing subprocess escape.
3. **AppContainer Isolation**:
   - Ephemeral AppContainer SID generated per verification attempt.
   - Read-only DACLs granted to snapshot directory; read-write DACLs granted only to ephemeral scratch dir.
   - Network capabilities omitted entirely.

### Decision 4: Verification Authority & Crash-Safe Latency Semantics via PROPOSAL-P04-002 (P04-ARCH-R10-003)

1. **Pre-Commit Assembly Boundary for Immutable Persistence**:
   - `evidence_finalized_at_epoch_ms`: Durably committed timestamp from Transaction B.
   - `bundle_assembled_at_epoch_ms`: Timestamp captured when ReviewBundle JSON assembly, RFC 8785 JCS canonicalization, and schema validation succeed, immediately before initiating Transaction C commit.
   - `compilation_latency_ms = bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms`.
   - Secondary commit duration telemetry (`commit_duration_ms`) is measured upon commit return via in-process monotonic clock and recorded in the audit event `REVIEW_BUNDLE_GENERATED`.
2. **Truth Table & Deadlock Elimination**:
   - Database constraint `compilation_latency_ms <= 3000` is eliminated.
   - ReviewBundles persist with `nfr008_met` (1 if latency <= 3000 ms, 0 if latency > 3000 ms) and `latency_measurement_status` ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART').
   - SLA breaches are recorded as diagnostic evidence; Transaction C commits and advances task to `REVIEWING` without deadlocking `EVIDENCE_READY`.

### Decision 5: Three Durable Pipeline Transactions, Admission & Bounded Fencing (P04-ARCH-R10-002, R10-004, R10-005)

1. **Model 1 Persistence Ownership Discipline**:
   - **Subtask P04A**: Owns Schema Migration v6 and Transaction A.
   - **Subtask P04B**: Pure in-memory Git evidence collection; ZERO SQLite writes.
   - **Subtask P04C**: Pure in-memory verification runner; ZERO SQLite writes.
   - **Subtask P04D**: Pipeline Orchestrator and SOLE SQLite CAS persistence orchestrator for Schema Migration v9 (`task_verification_leases`, `evidence_sets`, `review_artifacts`, `review_bundles`), Content-Addressed Store, leases, Transaction B, Transaction C, and audit events.
2. **Lease TTL Bounds & Mathematical Enforcement**:
   - `task_verification_leases.ttl_seconds CHECK (ttl_seconds BETWEEN 1 AND 600)`.
   - `CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000))` enforced in SQLite DDL.
   - CAS state transition trigger enforces valid transitions and fencing token increments on reclaim.
3. **ReviewBundle Replay and Conflict State Matrix (Pure Canonical States)**:
   - Compile failure: Transaction C rolls back, task remains `EVIDENCE_READY`, emits `REVIEW_BUNDLE_COMPILATION_REJECTED`.
   - Idempotent replay: Returns existing bundle, task remains `REVIEWING`.
   - Hash conflict: Tampering conflict, task remains `REVIEWING`, Transaction C rolls back, emits `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`), locks automated review decision approval.
   - Database invariant corruption: Task transitions to `BLOCKED`, admission is closed, emits `REVIEW_BUNDLE_COMPILATION_REJECTED` (`INVARIANT_CORRUPTION_DETECTED`).

```sql
-- Schema v9: task_verification_leases (Owned by Subtask P04D)
CREATE TABLE task_verification_leases (
    task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'COMPLETED', 'EXPIRED', 'RECLAIMED', 'REVOKED')),
    worker_id TEXT NOT NULL CHECK (LENGTH(worker_id) > 0),
    ttl_seconds INTEGER NOT NULL CHECK (ttl_seconds BETWEEN 1 AND 600),
    acquired_at_epoch_ms INTEGER NOT NULL CHECK (acquired_at_epoch_ms > 0),
    expires_at_epoch_ms INTEGER NOT NULL,
    released_at_epoch_ms INTEGER NULL,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000)),
    CHECK (
        (state = 'ACTIVE' AND released_at_epoch_ms IS NULL) OR
        (state IN ('COMPLETED', 'EXPIRED', 'RECLAIMED', 'REVOKED') AND released_at_epoch_ms IS NOT NULL AND released_at_epoch_ms >= acquired_at_epoch_ms)
    )
);

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
END;

CREATE TRIGGER trg_task_verification_leases_cas_guard
BEFORE UPDATE ON task_verification_leases
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in task_verification_leases')
    WHERE NEW.task_id != OLD.task_id
       OR NEW.attempt_id != OLD.attempt_id
       OR NEW.contract_id != OLD.contract_id
       OR NEW.acquired_at_epoch_ms != OLD.acquired_at_epoch_ms
       OR NEW.ttl_seconds != OLD.ttl_seconds
       OR NEW.expires_at_epoch_ms != OLD.expires_at_epoch_ms;

    SELECT RAISE(ABORT, 'illegal lease state transition')
    WHERE NOT (
        (OLD.state = 'ACTIVE' AND NEW.state IN ('COMPLETED', 'EXPIRED', 'REVOKED')) OR
        (OLD.state = 'EXPIRED' AND NEW.state = 'RECLAIMED' AND NEW.fencing_token > OLD.fencing_token)
    );
END;

CREATE TRIGGER trg_task_verification_leases_no_delete
BEFORE DELETE ON task_verification_leases
BEGIN
    SELECT RAISE(ABORT, 'task_verification_leases is immutable');
END;
```

### Decision 6: Durable Content-Addressed Artifact Store, Review Schema & Proposed Audit Events (P04-ARCH-R10-001, R10-004)

1. **Artifact Streaming & Exact Limits**:
   - Separate capture limit (10 MB saved to disk) from hard safety limit (50 MB process termination).
   - Strict mathematical bounds enforced across `stream_state` combinations.
2. **Content-Addressed Storage Layout**:
   - Canonical path exact equality: `canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256`.

```sql
-- Schema v9: evidence_sets (Owned by Subtask P04D)
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

-- Schema v9: review_artifacts (Owned by Subtask P04D)
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
    hard_safety_limit_bytes INTEGER NOT NULL CHECK (hard_safety_limit_bytes >= capture_limit_bytes),
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
        (stream_state = 'COMPLETE_EOF' AND is_truncated = 0 AND full_stream_sha256 IS NOT NULL AND captured_bytes = total_observed_bytes AND total_observed_bytes <= capture_limit_bytes)
        OR (stream_state = 'TRUNCATED_AT_CAPTURE_LIMIT' AND is_truncated = 1 AND full_stream_sha256 IS NOT NULL AND captured_bytes = capture_limit_bytes AND total_observed_bytes > captured_bytes AND total_observed_bytes <= hard_safety_limit_bytes)
        OR (stream_state = 'HARD_LIMIT_TERMINATED' AND is_truncated = 1 AND full_stream_sha256 IS NULL AND captured_bytes = capture_limit_bytes AND total_observed_bytes >= hard_safety_limit_bytes)
        OR (stream_state = 'TIMEOUT_ABORTED' AND is_truncated = 1 AND full_stream_sha256 IS NULL AND captured_bytes <= capture_limit_bytes)
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

-- Schema v9: review_bundles (Owned by Subtask P04D)
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
    nfr008_met INTEGER NOT NULL CHECK (
        nfr008_met IN (0, 1) AND (
            (nfr008_met = 1 AND compilation_latency_ms <= 3000) OR
            (nfr008_met = 0 AND compilation_latency_ms > 3000)
        )
    ),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_review_bundles_lineage_guard
BEFORE INSERT ON review_bundles
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match task_id, attempt_id, contract_id in evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.task_id = NEW.task_id
          AND e.attempt_id = NEW.attempt_id
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

## 5. Consequences

### Positive
- Strict, kernel-enforced process and network security boundary for all verification commands via Windows AppContainer.
- Complete lineage, immutability, and CAS transition enforcement across all pipeline tables.
- Pure in-memory separation for Subtasks P04B and P04C, eliminating concurrency bugs and orphaned records.
- Deadlock-free crash recovery across daemon restarts with mathematically verified lease and stream constraints.

### Negative / Trade-offs
- Verification execution duration is bounded by real test execution physics rather than sub-3-second expectations (reconciled cleanly via PROPOSAL-P04-002).
- Clean worktree policy v1 requires workers to keep worktree and index completely clean at report time.

---

## 6. Migration and Schema Ownership

- **Schema v6 (Owned by Subtask P04A)**:
  - `attempt_workspace_bindings` table, indexes, and triggers.
  - `worker_claims` table, indexes, and triggers.
- **Schema v9 (Owned by Subtask P04D)**:
  - `task_verification_leases` table, indexes, and triggers.
  - `evidence_sets` table, indexes, and triggers.
  - `review_artifacts` table, indexes, and triggers.
  - `review_bundles` table, indexes, and triggers.
- **Subtasks P04B & P04C**:
  - Pure in-memory execution; zero migration ownership, zero runtime SQLite writes.
