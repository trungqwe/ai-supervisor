# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Boundaries, and Verification Isolation

> **Proposal ID**: `PROPOSAL-P04-001`
> **Revision**: 14
> **Title**: Evidence & Review Engine Architecture, Execution Isolation, and Verification Governance
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Audited Baseline**: `8055c35e78e4d9fd5a6e1315334e6bfb0f8dc7c5`
> **Preservation Baseline Commit**: `6e1993da150031a9465901a7019c71257de44312` (Revision 11)
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_13`
> **Supersedes**: `PROPOSAL-P04-001` Revision 13
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R13-001` through `P04-ARCH-R13-004` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_012.md`).
> **Requirement & Governance Note**: Concurrently submits `PROPOSAL-P04-002` (Revision 7) for formal ReviewBundle latency measurement semantics. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until approved by External Supervisor.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 012 (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_012.md`) evaluated Round 12 remediation at commit `8055c35e78e4d9fd5a6e1315334e6bfb0f8dc7c5`, recording verdict `REVISION_13_REQUIRED`. It confirmed design-level closure on `P04-ARCH-R12-002` (linear lease chain) and `P04-ARCH-R12-004` (integer typing and overflow bounds), partially closed `P04-ARCH-R12-001` (residual matrix transition to BLOCKED remained) and `P04-ARCH-R12-003` (single-hold index and colon concatenation required overhaul), and opened findings `P04-ARCH-R13-001` through `P04-ARCH-R13-004`.

Revision 14 establishes complete design-level resolution of findings `P04-ARCH-R13-001` through `P04-ARCH-R13-004` via surgical patches applied directly to the Revision 13 architecture:
1. `P04-ARCH-R13-001`: Total elimination of erroneous TaskState transitions to `BLOCKED`. On invariant mismatch or corrupt state during ReviewBundle compilation, current TaskState is strictly preserved, active business transaction rolls back, a durable integrity hold is recorded in an independent diagnostic transaction, admission and automated approval are closed, and resolution is restricted to authenticated human reconciliation.
2. `P04-ARCH-R13-002`: Foreign key audit references and principal constraints on `review_integrity_holds` (`rejection_audit_event_id UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`, `resolution_audit_event_id UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`, `resolved_by_principal` non-empty/non-whitespace, atomic audit-then-hold creation transaction, atomic principal-audit-CAS resolution transaction, and fail-closed runtime dependency on `VERIFIED_OPERATOR_PRINCIPAL`).
3. `P04-ARCH-R13-003`: Multi-diagnostic hold tracking via `diagnostic_fingerprint` (64-character lowercase hex SHA-256) and partial unique index on `(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE'`. Dropped single-active-hold restriction; pipeline gates fail-closed on `EXISTS` any active hold; human reconciliation resolves each hold independently.
4. `P04-ARCH-R13-004`: Canonical audit event ID derivation via RFC 8785 JSON Canonicalization Scheme (JCS) hash; definition of sanitized input fingerprint (prohibiting raw host paths and secrets); duplicate key readback semantic comparison on AppendAuditEvent; and formal registration of `REVIEW_INTEGRITY_CONFLICT` and `REVIEW_INTEGRITY_HOLD_RESOLVED`.

---

## 2. Summary of Architectural Resolutions & Preservation Matrix

### 2.1. Architectural Resolutions in Revision 14
| Finding ID | Core Architectural Resolution in Revision 14 | Target Section |
| :--- | :--- | :--- |
| `P04-ARCH-R13-001` | Elimination of erroneous TaskState transitions to `BLOCKED`: preserve current TaskState on invariant mismatch; rollback active Tx; insert ACTIVE hold in separate diagnostic Tx; close admission and approval; require authenticated human reconciliation. | Section 4.1, Section 7.2, Section 7.4 |
| `P04-ARCH-R13-002` | Foreign key audit references and principal constraints on `review_integrity_holds`: `rejection_audit_event_id UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`, `resolution_audit_event_id UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`, `LENGTH(TRIM(resolved_by_principal)) > 0`, atomic creation/resolution transactions, fail-closed runtime constraint on `VERIFIED_OPERATOR_PRINCIPAL`. | Section 7.2, Section 7.3, Section 8 |
| `P04-ARCH-R13-003` | Multi-diagnostic hold tracking: `diagnostic_fingerprint` (64-char lowercase hex SHA-256); partial unique index `idx_review_integrity_holds_active_dedup` on `(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE'`; pipeline fail-closed on `EXISTS` any ACTIVE hold; independent human resolution. | Section 7.2, Section 7.3 |
| `P04-ARCH-R13-004` | Canonical audit event ID via RFC 8785 JCS (prohibiting colon concatenation); definition of `sanitized_input_fingerprint` (excluding secrets and raw host paths); duplicate key readback semantic comparison on `AppendAuditEvent`; formal registration of `REVIEW_INTEGRITY_CONFLICT` and `REVIEW_INTEGRITY_HOLD_RESOLVED`. | Section 4.1, Section 8 |

### 2.2. Revision 11 to Revision 14 Preservation Matrix
| Revision 11 Section | Revision 14 Section & Location | Status & Surgical Changes |
| :--- | :--- | :--- |
| Section 1: Context & Problem Statement | Section 1: Context & Problem Statement | Preserved; updated with Re-Audit 012 findings and baseline tracking. |
| Section 2: Summary of Architectural Resolutions | Section 2: Summary & Preservation Matrix | Preserved and updated with R13 resolutions and preservation matrix. |
| Section 3.1: Two-Transaction Workspace Lifecycle | Section 3.1: Two-Transaction Workspace Lifecycle | Preserved verbatim. |
| Section 3.2: Schema v6 DDL (`attempt_workspace_bindings`, `worker_claims`) | Section 3.2: Schema v6 DDL | Preserved full Schema v6 (`terminal_generation TEXT`, canonical worktree, triggers); added integer typing and canonical WorkerClaim array checks (`P04-ARCH-R12-001`, `R12-004`). |
| Section 4.1: Clean Worktree & Index Verification Policy | Section 4.1: Clean Worktree Policy | Preserved; updated with RFC 8785 JCS audit event ID and atomic audit-then-hold diagnostic creation transaction (`P04-ARCH-R13-001`, `R13-002`, `R13-004`). |
| Section 4.2: Hardened Git Invocation Allowlist | Section 4.2: Hardened Git Invocation Allowlist | Preserved 10-command allowlist table verbatim (`P04-ARCH-R12-001`). |
| Section 5.1: Windows AppContainer & Sandbox DACL | Section 5.1: Windows AppContainer Isolation | Preserved explicit handle list, Job Object assignment, DACL denial verbatim; added process-death proof requirement (`P04-ARCH-R12-001`, `R12-002`). |
| Section 6.1: Immutable Snapshot Extraction Protocol | Section 6.1: Snapshot Extraction Protocol | Preserved `git ls-tree -rz --full-tree` and `git cat-file --batch` protocol verbatim (`P04-ARCH-R12-001`). |
| Section 6.2: TOCTOU & Crash Matrix | Section 6.2: TOCTOU & Crash Matrix | Preserved matrix verbatim (`P04-ARCH-R12-001`). |
| Section 7.1: Model 1 Persistence Ownership Discipline | Section 7.1: Model 1 Persistence Ownership | Preserved verbatim (`P04-ARCH-R12-001`). |
| Section 7.2: Lease Admission, Cross-Process Reclaim & Bounded TTL | Section 7.2: Lease Admission, Reclaim & Bounded TTL | Preserved linear lease chain; updated durable holds with multi-diagnostic fingerprint, audit FKs, and fail-closed runtime principal check (`P04-ARCH-R13-002`, `R13-003`). |
| Section 7.3: Schema v9 DDL: Artifact Streaming & Review Schema | Section 7.3: Schema v9 DDL | Updated with multi-diagnostic `review_integrity_holds` DDL, audit FKs, principal trimming, and CAS triggers (`P04-ARCH-R13-002`, `R13-003`). |
| Section 7.4: ReviewBundle Replay & Conflict State Matrix | Section 7.4: Replay & Conflict Matrix | Updated matrix: eliminated transition to BLOCKED, current state preserved on invariant mismatch (`P04-ARCH-R13-001`). |
| Section 8: Audit Event Registration & Failure Semantics | Section 8: Audit Event Registration | Added RFC 8785 JCS event ID derivation, sanitized input fingerprint specification, duplicate-key readback comparison, and formal registration of `REVIEW_INTEGRITY_CONFLICT` and `REVIEW_INTEGRITY_HOLD_RESOLVED` (`P04-ARCH-R13-004`). |
| Section 9: Conclusion & Next Steps | Section 9: Conclusion & Next Steps | Updated governance status and active gate (`P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_13`). |

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
```

---

## 4. Hardened Git Evidence Collector & Allowlist

### 4.1. Clean Worktree & Index Verification Policy (v1)
To prevent workers from hiding changes in staged index entries, unstaged working copies, or untracked files:
1. **Intake Policy**: The Supervisor Control Plane strictly enforces that the workspace worktree and index must be clean at report intake.
2. **Rejection Semantics (P04-ARCH-R13-001, P04-ARCH-R13-002, P04-ARCH-R13-004)**: If any uncommitted change or untracked file is detected, Transaction A rolls back immediately, preserving task state `RUNNING` (strictly zero blanket transitions to `BLOCKED`). In a separate diagnostic transaction executed after rollback: appends rejection audit event `EVIDENCE_COLLECTION_FAILED` to `audit_events` first, then inserts an ACTIVE hold into `review_integrity_holds` (`hold_reason = 'DIRTY_WORKTREE_DETECTED'`) in the same diagnostic transaction. Rejection audit event ID is computed deterministically via RFC 8785 JCS: `event_id = SHA256(RFC8785_JCS(identity_descriptor))` using `version=1`, `event_type='EVIDENCE_COLLECTION_FAILED'`, `pair_id`, `task_id`, `attempt_id`, `contract_id`, `reason='DIRTY_WORKTREE_DETECTED'`, and `sanitized_input_fingerprint` (strictly omitting raw host paths and secrets). If the diagnostic transaction rolls back, fail-closed without claiming the audit event or hold was recorded.

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
   - Configures `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, ensuring that if the daemon process terminates or closes the job handle, all child verification processes are forcibly terminated by the OS kernel.
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

### 7.2. Lease Admission, Linear Reclaim Chain & Bounded TTL (P04-ARCH-R12-002, R12-003, R12-004)

#### Linear Lease Chain Model (Immutable History)
1. **Elimination of `RECLAIMED` Mutation**:
   - In-place mutation `EXPIRED -> RECLAIMED` on predecessor rows is eliminated.
   - Predecessor leases remain permanently in `state = 'EXPIRED'`, preserving their original `released_at_epoch_ms`.
   - Reclaiming creates an entirely new successor lease row referencing its predecessor via `predecessor_lease_id TEXT NULL UNIQUE REFERENCES task_verification_leases(lease_id)`.
2. **Linear Sequence & Single Active Invariant**:
   - Initial lease must have `predecessor_lease_id IS NULL` and `fencing_token = 1`.
   - Successor lease must have `predecessor_lease_id IS NOT NULL`, predecessor must match `(task_id, attempt_id, contract_id)`, predecessor state must be `EXPIRED`, `released_at_epoch_ms IS NOT NULL`, and `NEW.fencing_token = predecessor.fencing_token + 1`.
   - Partial unique indexes ensure at most one `ACTIVE` lease per task and per attempt.
   - Predecessor uniqueness (`UNIQUE(predecessor_lease_id)`) guarantees zero fork or reuse of predecessor leases.

#### Authoritative Process-Death Proof Taxonomy
1. **Same-Daemon Reclaim**:
   - In-memory Job Object handle registry tracks active child processes.
   - Daemon invokes `TerminateJobObject(handle, 1)` and joins child process exit via `WaitForSingleObject` / `GetExitCodeProcess`.
2. **Daemon Restart Recovery**:
   - A newly spawned daemon process *cannot* reopen anonymous Job Object handles from dead predecessor instances.
   - Recovery relies on P03 machine lock exclusivity (guaranteeing only one daemon runs) AND Windows `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` (kernel automatically terminates all child processes when the previous daemon's process handle table closes).
3. **Authoritative Proof Requirement**:
   - TTL expiration alone is NOT sufficient for reclaim. If process death cannot be proven via handle join or post-restart exclusivity + kill-on-close, the supervisor fails closed and refuses to create a successor lease.

#### Durable Integrity Hold Integration (P04-ARCH-R12-003, P04-ARCH-R13-002, P04-ARCH-R13-003)
- Table `review_integrity_holds` durably tracks intake rejections, worktree dirtiness, bundle hash conflicts, invariant mismatches, and unverified claims with exact attempt lineage.
- **Multi-Diagnostic Hold Tracking (P04-ARCH-R13-003)**: Each diagnostic violation carries a deterministic `diagnostic_fingerprint` (64-character lowercase hex SHA-256). The single-active-hold restriction is eliminated; multiple distinct `ACTIVE` holds on the same attempt are permitted when `hold_reason` or `diagnostic_fingerprint` differs, tracked via partial unique index `idx_review_integrity_holds_active_dedup` on `(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE'`.
- **Audit References & Principal Constraints (P04-ARCH-R13-002)**: Rejection audit event is foreign-keyed via `rejection_audit_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`. Resolution audit event is foreign-keyed via `resolution_audit_event_id TEXT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`. Resolving principal must satisfy `LENGTH(TRIM(resolved_by_principal)) > 0` when resolved.
- **Atomic Diagnostic Creation Transaction**: Append rejection audit event to `audit_events` first, then insert hold row into `review_integrity_holds` second, within a single diagnostic transaction after business transaction rollback. If any step fails, the entire diagnostic transaction rolls back.
- **Atomic Resolution Transaction**: Resolution requires authenticated human operator principal, appends resolution audit event `REVIEW_INTEGRITY_HOLD_RESOLVED`, and performs CAS update (`hold_state = 'RESOLVED'`, `resolved_at_epoch_ms = now`, `resolution_audit_event_id = event_id`, `resolved_by_principal = principal` WHERE `hold_id = :hold_id AND hold_state = 'ACTIVE'`) in a single transaction. Each hold is resolved independently.
- **Fail-Closed Runtime Dependency**: While `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`, runtime resolution fails closed; fake or test principals cannot resolve holds.
- **Idempotent Replay**: Exact replay of an identical diagnostic violation returns the existing hold without creating duplicate rows or erroring.
- **Fail-Closed Pipeline Gates**: All intake, verification (Tx B), compilation (Tx C), and review approval paths fail closed if `EXISTS (SELECT 1 FROM review_integrity_holds WHERE attempt_id = :attempt_id AND hold_state = 'ACTIVE')`.
- **Startup Recovery**: Scans for `ACTIVE` holds on startup before opening admission.

#### Bounded Verification TTL & Integer Typing
1. **Explicit Integer Typing**: `CHECK (typeof(col) = 'integer')` across all epoch, TTL, and token columns.
2. **Checked 64-bit Integer Arithmetic**: All timeout calculations enforce:
   `acquired_at_epoch_ms <= (9223372036854775807 - (ttl_seconds * 1000))`
   `fencing_token <= 9223372036854775807`.
3. **Stage B Contract Constraints**:
   - Maximum verification requests: `MAX_VERIFICATION_REQUESTS = 20`.
   - Maximum aggregate verification budget: `MAX_AGGREGATE_VERIFICATION_BUDGET_SECONDS = 600` (10 minutes).
4. **Lease TTL Bounds**: `task_verification_leases.ttl_seconds CHECK (typeof(ttl_seconds) = 'integer' AND ttl_seconds BETWEEN 1 AND 600)`.
5. **Exact Expiration Check**: `CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000))`.

```sql
-- Schema v9: task_verification_leases (Owned by Subtask P04D)
CREATE TABLE task_verification_leases (
    lease_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (
        typeof(fencing_token) = 'integer' AND fencing_token >= 1 AND fencing_token <= 9223372036854775807
    ),
    state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'COMPLETED', 'EXPIRED', 'REVOKED')),
    worker_id TEXT NOT NULL CHECK (LENGTH(worker_id) > 0),
    ttl_seconds INTEGER NOT NULL CHECK (
        typeof(ttl_seconds) = 'integer' AND ttl_seconds BETWEEN 1 AND 600
    ),
    acquired_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(acquired_at_epoch_ms) = 'integer'
        AND acquired_at_epoch_ms > 0
        AND acquired_at_epoch_ms <= (9223372036854775807 - (ttl_seconds * 1000))
    ),
    expires_at_epoch_ms INTEGER NOT NULL CHECK (typeof(expires_at_epoch_ms) = 'integer'),
    released_at_epoch_ms INTEGER NULL CHECK (
        released_at_epoch_ms IS NULL OR (
            typeof(released_at_epoch_ms) = 'integer' AND released_at_epoch_ms >= acquired_at_epoch_ms
        )
    ),
    predecessor_lease_id TEXT NULL UNIQUE REFERENCES task_verification_leases(lease_id) ON DELETE RESTRICT,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    UNIQUE(attempt_id, fencing_token),
    CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000)),
    CHECK (predecessor_lease_id IS NULL OR predecessor_lease_id != lease_id),
    CHECK ((predecessor_lease_id IS NULL AND fencing_token = 1) OR (predecessor_lease_id IS NOT NULL AND fencing_token > 1)),
    CHECK (
        (state = 'ACTIVE' AND released_at_epoch_ms IS NULL) OR
        (state IN ('COMPLETED', 'EXPIRED', 'REVOKED') AND released_at_epoch_ms IS NOT NULL)
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

    SELECT RAISE(ABORT, 'predecessor lease mismatch: predecessor must match attempt/contract, be EXPIRED, and fencing_token must be predecessor.fencing_token + 1')
    WHERE NEW.predecessor_lease_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM task_verification_leases p
        WHERE p.lease_id = NEW.predecessor_lease_id
          AND p.task_id = NEW.task_id
          AND p.attempt_id = NEW.attempt_id
          AND p.contract_id = NEW.contract_id
          AND p.state = 'EXPIRED'
          AND p.released_at_epoch_ms IS NOT NULL
          AND NEW.fencing_token = (p.fencing_token + 1)
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

    SELECT RAISE(ABORT, 'released_at_epoch_ms rewrite forbidden')
    WHERE OLD.released_at_epoch_ms IS NOT NULL AND NEW.released_at_epoch_ms != OLD.released_at_epoch_ms;

    SELECT RAISE(ABORT, 'illegal lease state transition: ACTIVE only transitions to COMPLETED, EXPIRED, or REVOKED')
    WHERE NOT (OLD.state = 'ACTIVE' AND NEW.state IN ('COMPLETED', 'EXPIRED', 'REVOKED'));
END;

CREATE TRIGGER trg_task_verification_leases_no_delete
BEFORE DELETE ON task_verification_leases
BEGIN
    SELECT RAISE(ABORT, 'task_verification_leases is immutable');
END;

-- Schema v9: review_integrity_holds (Owned by Subtask P04D, P04-ARCH-R13-002, P04-ARCH-R13-003)
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
```

### 7.3. Schema v9 DDL: Artifact Streaming & Review Schema (Owned by Subtask P04D)

```sql
-- Schema v9: evidence_sets
CREATE TABLE evidence_sets (
    evidence_set_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (
        typeof(fencing_token) = 'integer' AND fencing_token >= 1 AND fencing_token <= 9223372036854775807
    ),
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
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(evidence_finalized_at_epoch_ms) = 'integer' AND evidence_finalized_at_epoch_ms > 0
    ),
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
    capture_limit_bytes INTEGER NOT NULL CHECK (
        typeof(capture_limit_bytes) = 'integer' AND capture_limit_bytes > 0
    ),
    hard_safety_limit_bytes INTEGER NOT NULL CHECK (
        typeof(hard_safety_limit_bytes) = 'integer'
        AND hard_safety_limit_bytes > capture_limit_bytes
        AND hard_safety_limit_bytes <= 9223372036854775806
    ),
    captured_bytes INTEGER NOT NULL CHECK (
        typeof(captured_bytes) = 'integer'
        AND captured_bytes >= 0
        AND captured_bytes <= capture_limit_bytes
    ),
    total_observed_bytes INTEGER NOT NULL CHECK (
        typeof(total_observed_bytes) = 'integer'
        AND total_observed_bytes >= captured_bytes
        AND total_observed_bytes <= 9223372036854775807
    ),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1)),
    stream_state TEXT NOT NULL CHECK (
        stream_state IN ('COMPLETE_EOF', 'TRUNCATED_AT_CAPTURE_LIMIT', 'HARD_LIMIT_TERMINATED', 'TIMEOUT_ABORTED')
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
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
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(evidence_finalized_at_epoch_ms) = 'integer' AND evidence_finalized_at_epoch_ms > 0
    ),
    bundle_assembled_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(bundle_assembled_at_epoch_ms) = 'integer' AND bundle_assembled_at_epoch_ms >= evidence_finalized_at_epoch_ms
    ),
    compilation_latency_ms INTEGER NOT NULL CHECK (
        typeof(compilation_latency_ms) = 'integer' AND
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
| Any non-`REVIEWING` | Exists | Any | Invariant violation: database corruption | Current State Preserved (Admission Closed) | `REVIEW_BUNDLE_COMPILATION_REJECTED` (`INVARIANT_MISMATCH`) |

> **Invariant Failure Handling (P04-ARCH-R13-001)**: On invariant corruption or database inconsistency, Transaction C rolls back; the current `TaskState` is strictly preserved (strictly zero blanket transitions to `BLOCKED`); in a separate diagnostic transaction executed after rollback: appends rejection audit event `REVIEW_BUNDLE_COMPILATION_REJECTED` (`INVARIANT_MISMATCH`) to `audit_events` first, then inserts an ACTIVE hold into `review_integrity_holds` (`hold_reason = 'INVARIANT_MISMATCH'`); admission and automated approval are closed for the attempt; resolution requires authenticated human operator reconciliation.

---

## 8. Audit Event Registration & Failure Semantics

The proposed audit events are registered under status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `WORKSPACE_BINDING_CREATED`: Recorded in Subtask P04A when workspace binding is established.
2. `VERIFICATION_LEASE_ACQUIRED`: Recorded in Subtask P04D upon lease grant.
3. `VERIFICATION_LEASE_RELEASED`: Recorded in Subtask P04D upon lease completion.
4. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C (Subtask P04D) upon successful ReviewBundle synthesis and persistence. Emitted within Transaction C *without* `commit_duration_ms`; post-commit duration telemetry is captured purely as best-effort in-process monotonic measurement.
5. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed diagnostic transaction if compilation fails, clock regresses ($T_1 < T_0$), or bundle hash conflicts with a pre-existing bundle.
6. `EVIDENCE_COLLECTION_FAILED`: Recorded in Subtask P04A or P04D if pre-intake checks fail (e.g. `DIRTY_WORKTREE_DETECTED`) or evidence extraction encounters unrecoverable errors.

#### Canonical Event ID Derivation & Replay Semantics (P04-ARCH-R13-004)
Because the canonical `audit_events` schema lacks a dedicated `idempotency_key` column, idempotency is mapped deterministically to the primary key `audit_events.event_id` using RFC 8785 JSON Canonicalization Scheme (JCS). String concatenation with colons is strictly prohibited.

1. **RFC 8785 JCS Event ID Derivation**:
   `event_id = SHA256(RFC8785_JCS(identity_descriptor))`
   where `identity_descriptor` contains:
   ```json
   {
     "attempt_id": "<attempt_id>",
     "contract_id": "<contract_id>",
     "event_type": "<event_type>",
     "pair_id": "<pair_id>",
     "reason": "<reason>",
     "sanitized_input_fingerprint": "<64_hex_hash>",
     "task_id": "<task_id>",
     "version": 1
   }
   ```
2. **Sanitized Input Fingerprint Definition**:
   - `sanitized_input_fingerprint` is a 64-character lowercase hex SHA-256 hash of the canonicalized (RFC 8785 JCS) diagnostic input payload.
   - Strictly uses canonical relative paths (e.g. `artifacts/ab/...` or `internal/foo.go`). Raw host-specific local paths (e.g. `C:\Users\...`, `/home/...`, drive letters, UNC shares, or temp directories) are strictly prohibited.
   - Explicitly scrubbed to exclude secrets, credentials, API keys, tokens, passwords, private keys, environment blocks, and sensitive Git configurations.
3. **Duplicate Key Readback & Idempotency / Conflict Resolution on AppendAuditEvent**:
   When `AppendAuditEvent` encounters a duplicate primary key error on `audit_events.event_id`:
   - **Readback**: Read the existing event from `audit_events` WHERE `event_id = :event_id`.
   - **Semantic Field Comparison**: Compare exact matches on: `event_type`, `actor_id`, `actor_role`, `task_id`, `attempt_id`, `contract_id`, and canonical `details_json` payload.
   - **Ignored Volatile / Chain Fields**: Explicitly do NOT compare `sequence_number`, `timestamp`, `prev_event_hash`, or `event_hash`.
   - **Exact Match -> Idempotent Success**: If all semantic fields match, treat as idempotent success and return the existing audit record without inserting a duplicate.
   - **Mismatch -> Audit Integrity Conflict**: If any semantic field or lineage differs, fail closed! Do NOT overwrite or reuse the colliding `event_id`. Initiate an atomic diagnostic transaction:
     a. Derive a NEW distinct `event_id` via RFC 8785 JCS using `event_type = 'REVIEW_INTEGRITY_CONFLICT'`, referencing the colliding event ID and mismatch details in `sanitized_input_fingerprint`.
     b. Append `REVIEW_INTEGRITY_CONFLICT` to `audit_events`.
     c. Insert an ACTIVE hold into `review_integrity_holds` with `hold_reason = 'INVARIANT_MISMATCH'`.
     d. Close admission and automated approval for the attempt.
4. **Formal Audit Event Registrations**:
   - **`REVIEW_INTEGRITY_CONFLICT`**:
     - *Producer*: Subtask P04A (Report Intake) / Subtask P04D (Pipeline CAS Orchestrator).
     - *Actor Authority*: System supervisor daemon (`actor_id = 'ai-supervisor-daemon'`, `actor_role = 'SUPERVISOR'`).
     - *Lineage*: `task_id`, `attempt_id`, `contract_id`, `pair_id`.
     - *Details Schema*: `colliding_event_id` (string), `attempted_event_type` (string), `attempted_reason` (string), `conflict_type` (`'PAYLOAD_MISMATCH'` | `'LINEAGE_MISMATCH'`), `sanitized_input_fingerprint` (64 hex), `diagnostic_fingerprint` (64 hex).
     - *Transaction Boundary*: Atomic diagnostic transaction (distinct new `event_id`, appends audit event first, then inserts ACTIVE hold into `review_integrity_holds`).
   - **`REVIEW_INTEGRITY_HOLD_RESOLVED`**:
     - *Producer*: Operator Reconciliation Tool.
     - *Actor Authority*: Authenticated Human Operator Principal (`actor_role = 'OPERATOR'`, `actor_id = resolved_by_principal`). While `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`, runtime resolution fails closed; test/fake principals cannot resolve holds.
     - *Lineage*: `task_id`, `attempt_id`, `contract_id`, `pair_id`.
     - *Details Schema*: `hold_id` (string), `hold_reason` (string), `diagnostic_fingerprint` (64 hex), `rejection_audit_event_id` (string), `resolved_by_principal` (non-empty, trimmed string), `resolution_rationale` (non-empty string).
     - *Transaction Boundary*: Atomic resolution transaction (appends `REVIEW_INTEGRITY_HOLD_RESOLVED` to `audit_events`, CAS updates `review_integrity_holds` from `ACTIVE` to `RESOLVED`).

---

## 9. Conclusion & Next Steps

PROPOSAL-P04-001 Revision 14 fully resolves all findings (`P04-ARCH-R13-001` through `P04-ARCH-R13-004`), eliminating erroneous TaskState transitions to BLOCKED, establishing foreign key audit references and principal constraints on `review_integrity_holds`, implementing multi-diagnostic hold tracking via diagnostic fingerprint, and deriving canonical audit event IDs via RFC 8785 JCS.

Following External Supervisor review:
1. `DRAFT-ADR-018` is aligned to Revision 14.
2. `PROPOSAL-P04-002` remains aligned to Revision 7 (latency semantics unchanged).
3. `PLAN-P04-EVIDENCE-REVIEW` is updated to Revision 14.
4. Active gate remains `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_13` pending External Supervisor Re-Audit 013 decision.
