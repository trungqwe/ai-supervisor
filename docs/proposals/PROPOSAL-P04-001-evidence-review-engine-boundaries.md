# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Boundaries, and Verification Isolation

> **Proposal ID**: `PROPOSAL-P04-001`
> **Revision**: 11
> **Title**: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Audited Baseline**: `cd0417e641468abfac254cc57cca29be54d1e8e1`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_10`
> **Supersedes**: `PROPOSAL-P04-001` Revision 10
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R10-001` through `P04-ARCH-R10-005` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_009.md`).
> **Requirement & Governance Note**: Concurrently submits `PROPOSAL-P04-002` (Revision 5) for formal NFR-008 performance SLA reconciliation. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until approved by External Supervisor.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 009 evaluated Revision 10 and recorded five architectural findings requiring remediation:
1. `P04-ARCH-R10-001`: Regression in DDL lineage columns, composite foreign keys, immutability triggers, and exact content-address artifact path equality.
2. `P04-ARCH-R10-002`: Violation of Model 1 persistence ownership (incorrectly attributing Schema v9 to Subtask P04C instead of P04D).
3. `P04-ARCH-R10-003`: Ambiguity in NFR-008 measurement boundaries and lack of strict truth table constraints.
4. `P04-ARCH-R10-004`: Lack of mathematical enforcement for lease TTL (`expires_at_epoch_ms = acquired_at_epoch_ms + ttl_seconds * 1000`) and stream state combinations.
5. `P04-ARCH-R10-005`: Use of non-canonical compound state labels and unregistered audit literals.

This proposal formalizes Revision 11 architectural resolutions restoring all required safeguards without regressing previous audit gains.

---

## 2. Summary of Architectural Resolutions (Revision 11)

| Finding ID | Core Architectural Resolution in Revision 11 | Target Section |
| :--- | :--- | :--- |
| `P04-ARCH-R10-001` | Restore all lineage columns (`contract_id`, `session_id`, `terminal_generation`, `pinned_ao_commit`), composite foreign keys, and lineage triggers across all tables. Restore immutable update and delete triggers. Enforce exact equality `canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256` with anti-traversal checks. Restore `media_type` and `encoding` in `review_artifacts`. Restore `git rev-parse --absolute-git-dir` to the allowlist. | Section 3, Section 4, Section 7 |
| `P04-ARCH-R10-002` | Strictly enforce Model 1 persistence ownership: Subtask P04A owns Schema Migration v6 and Transaction A; Subtasks P04B and P04C are pure in-memory collectors with ZERO SQLite writes; Subtask P04D is the SOLE SQLite CAS orchestrator owning Schema Migration v9, Content-Addressed Store, leases, Transaction B, Transaction C, and audit events. | Section 7.1, Section 9 |
| `P04-ARCH-R10-003` | Select application pre-commit assembly & validation boundary for immutable ReviewBundle columns: `evidence_finalized_at_epoch_ms`, `bundle_assembled_at_epoch_ms`, `compilation_latency_ms`. Record commit duration via in-process monotonic clock in proposed audit event `REVIEW_BUNDLE_GENERATED` (`commit_duration_ms`). Remove `MEASUREMENT_TIMEOUT`. Enforce strict truth table CHECK constraints. | Section 7.3, `PROPOSAL-P04-002` |
| `P04-ARCH-R10-004` | Enforce `CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000))` in `task_verification_leases`. Add CAS state transition trigger enforcing allowed lease transitions and fencing token increment on reclaim. Enforce exact stream state combination CHECK constraints in `review_artifacts`. | Section 7.2, Section 7.3 |
| `P04-ARCH-R10-005` | Use strictly canonical states and transitions from `docs/06_WORKFLOW_STATE_MACHINE.md` (no compound labels like `BLOCKED / HUMAN_REQUIRED`). Dirty worktree report intake preserves `RUNNING` state while aborting Transaction A. Bundle hash conflict at `REVIEWING` preserves `REVIEWING` state, aborts Transaction C, records `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`), and blocks automated approval. | Section 6.2, Section 7.4, Section 8 |

---

## 3. Worktree Authority, Immutable Bindings & Seam Integration

### 3.1. Two-Transaction Workspace Lifecycle
To preserve the Stage B TaskContract seam without mutating existing task creation transactions:
1. **Binding Creation Transaction (Subtask P04A)**:
   - Executed after TaskContract release and before worker dispatch.
   - Inserts row into `attempt_workspace_bindings` with initial state `ACTIVE`, binding physical worktree identity (`FileIdInfo`) and linked gitdir identity.
2. **Report Intake Transaction A (Subtask P04A)**:
   - Executed upon worker completion signaling.
   - Validates physical worktree handle identity via Windows `FileIdInfo`.
   - Runs pre-intake cleanliness probe (`git status --porcelain=v1 -z --untracked-files=all` and `git diff-index --quiet HEAD --`). If dirty, Transaction A rolls back, task state remains `RUNNING` preserved (attempt report rejected), and `EVIDENCE_COLLECTION_FAILED` (`DIRTY_WORKTREE_DETECTED`) is recorded.
   - Validates `WorkerReport` JSON against canonical schema `docs/schemas/worker-report.schema.json` in Go.
   - JCS-canonicalizes payload (RFC 8785).
   - Inserts verbatim `reported_head_sha` and canonicalized `payload_json` into `worker_claims`.
   - Updates `attempt_workspace_bindings.binding_state` to `RETAINED_FOR_VERIFICATION` via CAS.
   - Transitions `tasks.state`: `RUNNING -> REPORT_READY`.

### 3.2. Schema v6 DDL: `attempt_workspace_bindings` and `worker_claims` (Owned by Subtask P04A)

```sql
-- Schema v6: attempt_workspace_bindings
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

-- Schema v6: worker_claims
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

---

## 4. Hardened Git Evidence Collector & Allowlist

### 4.1. Clean Worktree & Index Verification Policy (v1)
To prevent workers from hiding changes in staged index entries, unstaged working copies, or untracked files:
1. **Intake Policy**: The Supervisor Control Plane strictly enforces that the workspace worktree and index must be clean at report intake.
2. **Rejection Semantics**: If any uncommitted change or untracked file is detected, Transaction A rolls back immediately, preserving task state `RUNNING` and recording audit event `EVIDENCE_COLLECTION_FAILED` with detail `DIRTY_WORKTREE_DETECTED`.

### 4.2. Hardened Git Invocation Allowlist
All Git commands execute under strict environment isolation (`GIT_DIR`, `GIT_WORK_TREE`, `GIT_OPTIONAL_LOCKS=0`, clean system environment, no user config):

| Subcommand & Arguments | Purpose | Security Constraints |
| :--- | :--- | :--- |
| `git status --porcelain=v1 -z --untracked-files=all` | Clean worktree & untracked check | Machine-parseable, NUL-delimited, unescaped paths |
| `git diff-index --quiet HEAD --` | Staged/index clean check | Exit code 0 indicates clean index vs HEAD |
| `git rev-parse --verify --quiet <ref>^{commit}` | Resolve ref to 40-char SHA | Commit object dereferencing |
| `git diff --raw -z --no-renames --no-ext-diff <base> <head> --` | Raw change list | NUL-delimited path parsing |
| `git diff --numstat --no-renames <base> <head> --` | Diff statistics | Bound by line count |
| `git rev-list --count <base>..<head>` | Commit count validation | Integer output |
| `git ls-tree -rz --full-tree <head>` | Tree manifest for snapshot | NUL-delimited tree walk |
| `git cat-file --batch` | Content extraction for snapshot | Strict stdin/stdout protocol |
| `git rev-parse --git-path index` | Linked worktree index path | Resolves exact index file |
| `git rev-parse --absolute-git-dir` | Common repository git directory | Restored for linked worktree validation |

---

## 5. Windows AppContainer Process Isolation & Job Containment

### 5.1. Handle Inheritance & Sandbox DACL Architecture
To prevent handle leakage while maintaining child process stdio communication:
1. **STARTUPINFOEXW with Attribute Lists**:
   - `bInheritHandles = TRUE` in `CreateProcessW`.
   - `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` explicitly passes *only* the specific write handles for stdout, stderr, and the read handle for stdin. All database, network, and file handles remain strictly non-inherited.
2. **Atomic Job Object Assignment**:
   - `PROC_THREAD_ATTRIBUTE_JOB_LIST` atomically binds the child process to a newly created Windows Job Object at the moment of creation.
   - Prohibits child processes from breaking out via `JOB_OBJECT_LIMIT_BREAKAWAY_OK` (flag explicitly omitted).
3. **AppContainer SID Isolation**:
   - Ephemeral AppContainer profile generated per verification task.
   - Minimal DACL granted only to snapshot directory and ephemeral scratch directory (`GENERIC_READ | GENERIC_EXECUTE`).
   - Network capabilities (`capabilitySid`) completely omitted, enforcing OS-level network denial.

---

## 6. Immutable Source Snapshot Protocol & Clean Verification Matrix

### 6.1. Snapshot Extraction Protocol
The verification runner never executes directly inside the worker's mutable worktree. Instead:
1. **Pre-Snapshot Cleanliness Verification**: Re-verify `git status --porcelain=v1 -z` and index identity.
2. **Deterministic Tree Extraction**:
   - Read directory tree from `actual_head_sha` via `git ls-tree -rz --full-tree`.
   - Extract file blobs via `git cat-file --batch` into `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/`.
3. **ReadOnly ACL Enforcement**: Apply read-only ACLs (`GENERIC_READ | GENERIC_EXECUTE`) before spawning any verification processes.

### 6.2. TOCTOU & Crash Matrix

| Scenario | Detection Mechanism | Consequence / Recovery Action | Canonical Resulting State |
| :--- | :--- | :--- | :--- |
| Worker leaves unstaged edits | `git status` check in Pre-Transaction A | Transaction A aborted; `DIRTY_WORKTREE_DETECTED` logged | `RUNNING` (Preserved) |
| Worker stages edits in index | `git diff-index --quiet HEAD --` in Pre-Transaction A | Transaction A aborted; `DIRTY_WORKTREE_DETECTED` logged | `RUNNING` (Preserved) |
| Worker leaves untracked files | `git status --untracked-files=all` | Transaction A aborted; `DIRTY_WORKTREE_DETECTED` logged | `RUNNING` (Preserved) |
| Worktree modified between Tx A and snapshot | Re-verify `git status` before snapshot extraction | Snapshot aborted; lease released as failure | `FAILED` |
| Verification process attempts file modification | Read-only ACLs on snapshot directory | OS rejects write with `ACCESS_DENIED` | `REPORT_READY` (Process fails) |
| Worker touches worktree during verification | Verification runs in isolated snapshot dir | Worker mutations have zero effect on verification | Independent verification proceeds |

---

## 7. Pipeline Transactions, Fencing & Content-Addressed Store

### 7.1. Model 1 Persistence Ownership Discipline
To preserve architectural boundaries and prevent concurrency bugs:
- **Subtask P04A**: Owns Schema Migration v6 (`attempt_workspace_bindings`, `worker_claims`) and Transaction A.
- **Subtask P04B**: Pure in-memory Git evidence collection; returns `GitEvidenceResult` in memory; ZERO SQLite writes.
- **Subtask P04C**: Pure in-memory verification runner; returns `TestEvidenceResult` and artifact streams in memory; ZERO SQLite writes.
- **Subtask P04D**: Pipeline Orchestrator and SOLE SQLite CAS persistence orchestrator for Schema Migration v9 (`task_verification_leases`, `evidence_sets`, `review_artifacts`, `review_bundles`), Content-Addressed Store, lease lifecycle, Transaction B, Transaction C, and associated audit events.

### 7.2. Lease Admission, Cross-Process Reclaim & Bounded TTL (P04-ARCH-R10-004)

#### Reclaim Authority Taxonomy
1. **Same-Daemon Reclaim**:
   - In-memory Job Object handle registry tracks active child processes.
   - Daemon invokes `TerminateJobObject(handle, 1)` and joins child process exits via `WaitForSingleObject`.
2. **Daemon Restart Recovery**:
   - A newly spawned daemon process *cannot* reopen anonymous Job Object handles from dead predecessor instances.
   - Recovery relies on P03 machine lock exclusivity (guaranteeing only one daemon runs) AND Windows `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` (kernel automatically terminates all child processes when the previous daemon's process handle table closes).
3. **Unproven Owner Guard**:
   - If process death cannot be proven via handle join or post-restart exclusivity, the supervisor fails closed and refuses to reclaim the lease.
   - Fencing token increments *only* after old owner termination is verified (`NEW.fencing_token > OLD.fencing_token`).

#### Bounded Verification TTL Arithmetic
1. **Checked 64-bit Integer Arithmetic**: All timeout calculations use checked integer arithmetic rejecting overflow.
2. **Stage B Contract Constraints**:
   - Maximum verification requests: `MAX_VERIFICATION_REQUESTS = 20`.
   - Maximum aggregate verification budget: `MAX_AGGREGATE_VERIFICATION_BUDGET_SECONDS = 600` (10 minutes).
   - Stage B semantic contract validation strictly rejects any contract violating these limits.
3. **Lease TTL Bounds**: `task_verification_leases.ttl_seconds CHECK (ttl_seconds BETWEEN 1 AND 600)`.
4. **Exact Expiration Check**: `CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000))`.

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

### 7.3. Schema v9 DDL: Artifact Streaming & Review Schema (Owned by Subtask P04D)

```sql
-- Schema v9: evidence_sets
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

-- Schema v9: review_artifacts
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

-- Schema v9: review_bundles
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

### 7.4. ReviewBundle Replay and Conflict State Matrix (P04-ARCH-R10-005)

| Current TaskState | Pre-Existing Bundle | Incoming Bundle Hash vs Stored Hash | Resulting Action & Transaction C Outcome | Final TaskState | Emitted Audit Event |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `EVIDENCE_READY` | None | N/A (Compile error / schema failure) | Transaction C rolls back | `EVIDENCE_READY` (Preserved) | `REVIEW_BUNDLE_COMPILATION_REJECTED` |
| `EVIDENCE_READY` | None | N/A (Successful synthesis) | Transaction C commits bundle atomically | `REVIEWING` (Advanced) | `REVIEW_BUNDLE_GENERATED` |
| `REVIEWING` | Exists | Matches stored hash exactly | Idempotent replay: return existing bundle | `REVIEWING` (Preserved) | None (or diagnostic telemetry) |
| `REVIEWING` | Exists | Conflicts with stored hash | Integrity conflict: Transaction C rolls back; automated review approval locked | `REVIEWING` (Preserved; human resolution required) | `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`) |
| Any non-`REVIEWING` | Exists | Any | Invariant violation: database corruption | `BLOCKED` (Admission closed) | `REVIEW_BUNDLE_COMPILATION_REJECTED` (`INVARIANT_CORRUPTION_DETECTED`) |

---

## 8. Audit Event Registration & Failure Semantics

The proposed audit events are registered under status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `WORKSPACE_BINDING_CREATED`: Recorded in Subtask P04A when workspace binding is established.
2. `VERIFICATION_LEASE_ACQUIRED`: Recorded in Subtask P04D upon lease grant.
3. `VERIFICATION_LEASE_RELEASED`: Recorded in Subtask P04D upon lease completion.
4. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C (Subtask P04D) upon successful ReviewBundle synthesis and persistence. Includes `commit_duration_ms` monotonic telemetry.
5. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed diagnostic transaction if compilation fails, clock regresses ($T_1 < T_0$), or bundle hash conflicts with a pre-existing bundle.
6. `EVIDENCE_COLLECTION_FAILED`: Recorded in Subtask P04A or P04D if pre-intake checks fail (e.g. `DIRTY_WORKTREE_DETECTED`) or evidence extraction encounters unrecoverable errors.

---

## 9. Conclusion & Next Steps

PROPOSAL-P04-001 Revision 11 fully resolves all findings (`P04-ARCH-R10-001` through `P04-ARCH-R10-005`), restoring lineage triggers, immutability guards, content-address path binding, Model 1 persistence ownership, and strict canonical TaskState transitions.

Following External Supervisor review:
1. `DRAFT-ADR-018` is aligned to Revision 11.
2. `PLAN-P04-EVIDENCE-REVIEW` is updated to Revision 11.
3. Once accepted, Task Contract `CONTRACT-TASK-P04-001` may be drafted for Subtask P04A.
