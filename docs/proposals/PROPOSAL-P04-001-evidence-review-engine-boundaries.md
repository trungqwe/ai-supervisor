# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Boundaries, and Verification Isolation

> **Proposal ID**: `PROPOSAL-P04-001`
> **Revision**: 10
> **Title**: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Audited Baseline**: `6dbd7e26d59e22171271921b86c01bd6f215e59a`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_9`
> **Supersedes**: `PROPOSAL-P04-001` Revision 9
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R9-001` through `P04-ARCH-R9-006` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_008.md`).
> **Requirement & Governance Note**: Concurrently submits `PROPOSAL-P04-002` (Revision 4) for formal NFR-008 performance SLA reconciliation. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until approved by External Supervisor.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 008 evaluated Revision 9 and recorded six critical architectural findings requiring remediation:
1. `worker_claims` schema enforced an exact 40-char SHA check, conflicting with canonical `worker-report.schema.json` allowing 7-40 hex chars, and lacked unambiguous mappings between WorkerReport, WorkerClaim, and ReviewBundle (`P04-ARCH-R9-001`).
2. NFR-008 latency check `compilation_latency_ms <= 3000` created a permanent database deadlock if the daemon restarted between Transaction B and C or exceeded 3 seconds (`P04-ARCH-R9-002`).
3. Terminating processes upon exceeding hard byte limits contradicted metadata claiming `full_stream_sha256` and `original_bytes` were known (`P04-ARCH-R9-003`).
4. Staged, unstaged, and untracked worktree modifications were unconstrained before snapshot extraction (`P04-ARCH-R9-004`).
5. Lease reclaim across daemon restarts claimed to terminate old Job Objects without valid handles, and verification request timeouts lacked checked arithmetic and aggregate caps (`P04-ARCH-R9-005`).
6. ReviewBundle replay and hash conflict handling assigned incorrect TaskStates (`P04-ARCH-R9-006`).

This proposal formalizes Revision 10 architectural resolutions for these findings.

---

## 2. Summary of Architectural Resolutions (Revision 10)

| Finding ID | Core Architectural Resolution in Revision 10 | Target Section |
| :--- | :--- | :--- |
| `P04-ARCH-R9-001` | Preserve verbatim worker reported head SHA (`LENGTH BETWEEN 7 AND 40 AND NOT GLOB '*[^0-9a-f]*'`). Independent `actual_head_sha` belongs strictly to Evidence. Establish explicit mapping: WorkerReport -> canonical WorkerClaim object -> ReviewBundle `worker_claims`. Enforce Go schema validation, exact data typing, and RFC 8785 JCS canonicalization before insertion. SQLite DDL uses `json_valid()` and `json_type()` as defense-in-depth without claiming SQLite proves JCS. | Section 3 |
| `P04-ARCH-R9-002` | Distinguish 4 timestamp concepts (pre-commit write timestamp, WAL commit completion, monotonic in-process duration, durable wall-clock diagnostic). Drop `CHECK <= 3000` persistence blocker. Persist `nfr008_met` (0/1) and `latency_measurement_status`. SLA miss is diagnostic; Transaction C commits and advances to `REVIEWING` without deadlocking `EVIDENCE_READY`. | Section 7.3, `PROPOSAL-P04-002` |
| `P04-ARCH-R9-003` | Separate capture limit (10 MB stored to disk) from hard safety limit (50 MB process termination). Introduce `stream_state` ('COMPLETE_EOF', 'TRUNCATED_AT_CAPTURE_LIMIT', 'HARD_LIMIT_TERMINATED', 'TIMEOUT_ABORTED'). When process is terminated at hard limit or timeout before observing EOF, set `full_stream_sha256 = NULL`, record `total_observed_bytes`, and mark `is_truncated = 1`. | Section 7.3 |
| `P04-ARCH-R9-004` | Enforce clean worktree policy v1 at report intake: fail-closed if staged, unstaged, or untracked changes exist. Add `git status --porcelain=v1 -z --untracked-files=all` and `git diff-index --quiet HEAD --` to Git allowlist with `GIT_OPTIONAL_LOCKS=0`. Revalidate clean status and HEAD/index identity before snapshot extraction and before Transaction B. | Section 4, Section 6 |
| `P04-ARCH-R9-005` | Distinguish same-daemon lease reclaim (in-memory Job handle registry join) from daemon restart recovery (P03 host lock exclusivity + `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`). Unproven owners fail closed without reclaim. Enforce checked 64-bit TTL arithmetic, cap verification requests to max 20, and cap aggregate budget to 600s in Stage B contract validation. | Section 7.2 |
| `P04-ARCH-R9-006` | Establish explicit TaskState matrix: compile failure preserves `EVIDENCE_READY`; idempotent replay preserves `REVIEWING`; hash conflict preserves `REVIEWING` with `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`) audit failure; orphaned bundle triggers invariant violation. Transaction C enforces atomic commit of ReviewBundle, audit event, and state transition. | Section 7.4, Section 8 |

---

## 3. Worktree Authority, Immutable Bindings & Seam Integration

### 3.1. Two-Transaction Workspace Lifecycle
To preserve the Stage B TaskContract seam without mutating existing task creation transactions:
1. **Binding Creation Transaction (Subtask P04A)**:
   - Executed after TaskContract release and before worker dispatch.
   - Inserts row into `attempt_workspace_bindings` with initial state `ACTIVE` and `initial_head_sha`.
2. **Report Intake Transaction A (Subtask P04A)**:
   - Executed upon worker completion signaling.
   - Validates physical worktree handle identity via Windows `FileIdInfo`.
   - Runs pre-intake cleanliness probe (`git status --porcelain=v1 -z --untracked-files=all` and `git diff-index --quiet HEAD --`). Fails closed with `DIRTY_WORKTREE_DETECTED` if any staged, unstaged, or untracked changes exist.
   - Validates `WorkerReport` JSON against canonical schema `docs/schemas/worker-report.schema.json` in Go.
   - JCS-canonicalizes payload (RFC 8785).
   - Inserts verbatim `reported_head_sha` and canonicalized `payload_json` into `worker_claims`.
   - Updates `attempt_workspace_bindings.binding_state` to `RETAINED_FOR_VERIFICATION`.
   - Transitions `tasks.state`: `RUNNING -> REPORT_READY`.

### 3.2. WorkerClaim Canonical Mapping & Schema Compliance (P04-ARCH-R9-001)

The canonical specifications define distinct representations across schemas:
- `worker-report.schema.json`: defines `head_sha` as `^[0-9a-f]{7,40}$`, `files_changed` as string array, `tests` as test results array, `worker_claims` as string array (`string[]`).
- `docs/05_DOMAIN_MODEL.md`: defines class `WorkerClaim` containing `reported_head_sha`, `claimed_files_changed`, and `tests`.
- `review-bundle.schema.json`: defines `worker_claims` as an object (`"type": "object"`).

To guarantee unambiguous mapping and preserve worker report fidelity:
1. **Verbatim Preservation**: The worker's reported head SHA is stored verbatim in `worker_claims.reported_head_sha` without silent resolution or expansion.
2. **Strict Separation of Evidence**: The independently resolved and verified `actual_head_sha` belongs strictly to `evidence_sets.git_evidence_json` (`actual_git_evidence.actual_head_sha`), never inside `worker_claims`.
3. **Canonical Object Mapping**:
   ```text
   WorkerReport JSON                      Canonical WorkerClaim Object               ReviewBundle.worker_claims Object
   ----------------                      ----------------------------               ---------------------------------
   head_sha (7-40 hex)        ------>    reported_head_sha               ------>    reported_head_sha
   files_changed (string[])   ------>    claimed_files_changed           ------>    claimed_files_changed
   tests (object[])           ------>    tests                           ------>    tests
   worker_claims (string[])   ------>    textual_claims                  ------>    textual_claims
   build_status (enum)        ------>    build_status                    ------>    build_status
   ```
4. **Validation and JCS Canonicalization Pipeline**:
   - Step 1: Go application parses raw worker report JSON.
   - Step 2: Validate against `docs/schemas/worker-report.schema.json` via schema engine.
   - Step 3: Extract and type-check fields according to the mapping above.
   - Step 4: Serialize canonical `WorkerClaim` object using RFC 8785 JSON Canonicalization Scheme (JCS).
   - Step 5: Insert into `worker_claims` table. SQLite DDL validates JSON syntax and array types as defense-in-depth.

```sql
-- Schema v6: attempt_workspace_bindings
CREATE TABLE attempt_workspace_bindings (
    binding_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    canonical_worktree_path TEXT NOT NULL,
    worktree_volume_serial TEXT NOT NULL,
    worktree_file_id TEXT NOT NULL,
    linked_gitdir_path TEXT NOT NULL,
    linked_gitdir_volume_serial TEXT NOT NULL,
    linked_gitdir_file_id TEXT NOT NULL,
    initial_head_sha TEXT NOT NULL CHECK (
        LENGTH(initial_head_sha) = 40 AND NOT (initial_head_sha GLOB '*[^0-9a-f]*')
    ),
    binding_state TEXT NOT NULL CHECK (
        binding_state IN ('ACTIVE', 'RETAINED_FOR_VERIFICATION', 'RELEASED', 'INVALIDATED')
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (created_at_epoch_ms > 0),
    released_at_epoch_ms INTEGER NULL,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

-- Schema v6: worker_claims
CREATE TABLE worker_claims (
    claim_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
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
    created_at_epoch_ms INTEGER NOT NULL CHECK (created_at_epoch_ms > 0)
);
```

---

## 4. Hardened Git Evidence Collector & Clean Worktree Allowlist (P04-ARCH-R9-004)

### 4.1. Clean Worktree & Index Verification Policy (v1)
To prevent workers from hiding changes in staged index entries, unstaged working copies, or untracked files:
1. **Intake Policy**: The Supervisor Control Plane strictly enforces that the workspace worktree and index must be clean at report intake.
2. **Rejection Semantics**: If any uncommitted change or untracked file is detected, Transaction A fails closed immediately, recording audit event `EVIDENCE_COLLECTION_FAILED` with detail `DIRTY_WORKTREE_DETECTED`.

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

| Scenario | Detection Mechanism | Consequence / Recovery Action |
| :--- | :--- | :--- |
| Worker leaves unstaged edits | `git status` check in Pre-Transaction A | Transaction A aborted; task rejected with `DIRTY_WORKTREE_DETECTED`. |
| Worker stages edits in index | `git diff-index --quiet HEAD --` in Pre-Transaction A | Transaction A aborted; task rejected with `DIRTY_WORKTREE_DETECTED`. |
| Worker leaves untracked files | `git status --untracked-files=all` | Transaction A aborted; task rejected with `DIRTY_WORKTREE_DETECTED`. |
| Worktree modified between Tx A and snapshot | Re-verify `git status` before snapshot extraction | Snapshot aborted; lease released as failure; task marked failed. |
| Verification process attempts file modification | Read-only ACLs on snapshot directory | OS rejects write with `ACCESS_DENIED`. |
| Worker touches worktree during verification | Verification runs in isolated snapshot dir | Worker mutations have zero effect on verification accuracy. |

---

## 7. Pipeline Transactions, Fencing & Content-Addressed Store

### 7.1. Three Durable Transaction Boundaries
The verification pipeline is decoupled into three sequential SQLite WAL transactions:
1. **Transaction A (Report Intake)**: Commits `worker_claims`, retains workspace binding, transitions `RUNNING -> REPORT_READY`.
2. **Transaction B (Evidence Finalization)**: Commits `evidence_sets`, commits `review_artifacts`, releases verification lease, transitions `REPORT_READY -> EVIDENCE_READY`.
3. **Transaction C (Bundle Synthesis & Review Transition)**: Commits canonical `review_bundles`, commits proposed audit event `REVIEW_BUNDLE_GENERATED`, transitions `EVIDENCE_READY -> REVIEWING`.

### 7.2. Lease Admission, Cross-Process Reclaim & Bounded TTL (P04-ARCH-R9-005)

#### Reclaim Authority Taxonomy
1. **Same-Daemon Reclaim**:
   - In-memory Job Object handle registry tracks active child processes.
   - Daemon invokes `TerminateJobObject(handle, 1)` and joins child process exits via `WaitForSingleObject`.
2. **Daemon Restart Recovery**:
   - A newly spawned daemon process *cannot* reopen anonymous Job Object handles from dead predecessor instances.
   - Recovery relies on P03 machine lock exclusivity (guaranteeing only one daemon runs) AND Windows `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` (kernel automatically terminates all child processes when the previous daemon's process handle table closes).
3. **Unproven Owner Guard**:
   - If process death cannot be proven via handle join or post-restart exclusivity, the supervisor fails closed and refuses to reclaim the lease.
   - Fencing token increments *only* after old owner termination is verified.

#### Bounded Verification TTL Arithmetic
1. **Checked 64-bit Integer Arithmetic**: All timeout calculations use checked integer arithmetic rejecting overflow.
2. **Stage B Contract Constraints**:
   - Maximum verification requests: `MAX_VERIFICATION_REQUESTS = 20`.
   - Maximum aggregate verification budget: `MAX_AGGREGATE_VERIFICATION_BUDGET_SECONDS = 600` (10 minutes).
   - Stage B semantic contract validation strictly rejects any contract violating these limits.
3. **Lease TTL Bounds**: `verification_leases.ttl_seconds CHECK (ttl_seconds BETWEEN 1 AND 600)`.

```sql
-- Schema v9: verification_leases
CREATE TABLE verification_leases (
    lease_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    lease_owner TEXT NOT NULL,
    lease_state TEXT NOT NULL CHECK (
        lease_state IN ('ACTIVE', 'COMPLETED', 'EXPIRED', 'RECLAIMED', 'REVOKED')
    ),
    lease_fencing_token INTEGER NOT NULL CHECK (lease_fencing_token > 0),
    ttl_seconds INTEGER NOT NULL CHECK (ttl_seconds BETWEEN 1 AND 600),
    acquired_at_epoch_ms INTEGER NOT NULL CHECK (acquired_at_epoch_ms > 0),
    expires_at_epoch_ms INTEGER NOT NULL CHECK (expires_at_epoch_ms > acquired_at_epoch_ms),
    released_at_epoch_ms INTEGER NULL,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);
```

### 7.3. Schema v9 DDL: Artifact Streaming & Latency Invariants (P04-ARCH-R9-002, R9-003)

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
    policy_findings_json TEXT NOT NULL CHECK (
        json_valid(policy_findings_json) = 1
    ),
    unverified_claims_json TEXT NOT NULL CHECK (
        json_valid(unverified_claims_json) = 1
    ),
    evidence_committed_at_epoch_ms INTEGER NOT NULL CHECK (evidence_committed_at_epoch_ms > 0),
    collected_at TEXT NOT NULL CHECK (LENGTH(collected_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

-- Schema v9: review_artifacts
CREATE TABLE review_artifacts (
    artifact_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    artifact_type TEXT NOT NULL CHECK (
        artifact_type IN ('VERIFICATION_LOG', 'GIT_DIFF', 'TEST_REPORT', 'WORKER_STDOUT', 'WORKER_STDERR')
    ),
    file_path TEXT NOT NULL CHECK (
        file_path GLOB 'artifacts/[0-9a-f][0-9a-f]/[0-9a-f]*'
        AND file_path NOT GLOB '*.*'
    ),
    stream_state TEXT NOT NULL CHECK (
        stream_state IN ('COMPLETE_EOF', 'TRUNCATED_AT_CAPTURE_LIMIT', 'HARD_LIMIT_TERMINATED', 'TIMEOUT_ABORTED')
    ),
    captured_bytes INTEGER NOT NULL CHECK (captured_bytes >= 0),
    total_observed_bytes INTEGER NOT NULL CHECK (total_observed_bytes >= captured_bytes),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1)),
    captured_sha256 TEXT NOT NULL CHECK (
        LENGTH(captured_sha256) = 64 AND NOT (captured_sha256 GLOB '*[^0-9a-f]*')
    ),
    full_stream_sha256 TEXT NULL CHECK (
        full_stream_sha256 IS NULL OR (
            LENGTH(full_stream_sha256) = 64 AND NOT (full_stream_sha256 GLOB '*[^0-9a-f]*')
        )
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (created_at_epoch_ms > 0),
    CHECK (
        (stream_state = 'COMPLETE_EOF' AND is_truncated = 0 AND full_stream_sha256 IS NOT NULL AND captured_bytes = total_observed_bytes)
        OR (stream_state = 'TRUNCATED_AT_CAPTURE_LIMIT' AND is_truncated = 1 AND full_stream_sha256 IS NOT NULL AND captured_bytes < total_observed_bytes)
        OR (stream_state = 'HARD_LIMIT_TERMINATED' AND is_truncated = 1 AND full_stream_sha256 IS NULL)
        OR (stream_state = 'TIMEOUT_ABORTED' AND is_truncated = 1 AND full_stream_sha256 IS NULL)
    )
);

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
        LENGTH(bundle_hash) = 64 AND
        NOT (bundle_hash GLOB '*[^0-9a-f]*')
    ),
    evidence_committed_at_epoch_ms INTEGER NOT NULL CHECK (evidence_committed_at_epoch_ms > 0),
    bundle_committed_at_epoch_ms INTEGER NOT NULL CHECK (bundle_committed_at_epoch_ms >= evidence_committed_at_epoch_ms),
    compilation_latency_ms INTEGER NOT NULL CHECK (
        compilation_latency_ms >= 0 AND
        compilation_latency_ms = (bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms)
    ),
    latency_measurement_status TEXT NOT NULL CHECK (
        latency_measurement_status IN ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART', 'MEASUREMENT_TIMEOUT')
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

CREATE TRIGGER trg_review_bundles_evidence_time_guard
BEFORE INSERT ON review_bundles
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'evidence timestamp mismatch: evidence_committed_at_epoch_ms does not match evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.evidence_committed_at_epoch_ms = NEW.evidence_committed_at_epoch_ms
    );
END;
```

### 7.4. ReviewBundle Replay and Conflict State Matrix (P04-ARCH-R9-006)

| Current TaskState | Pre-Existing Bundle | Incoming Bundle Hash vs Stored Hash | Resulting Action & Transaction C Outcome | Final TaskState | Emitted Audit Event |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `EVIDENCE_READY` | None | N/A (Compile error / schema failure) | Transaction C rolls back | `EVIDENCE_READY` (Preserved) | `REVIEW_BUNDLE_COMPILATION_REJECTED` |
| `EVIDENCE_READY` | None | N/A (Successful synthesis) | Transaction C commits bundle atomically | `REVIEWING` (Advanced) | `REVIEW_BUNDLE_GENERATED` |
| `REVIEWING` | Exists | Matches stored hash exactly | Idempotent replay: return existing bundle | `REVIEWING` (Preserved) | None (or diagnostic telemetry) |
| `REVIEWING` | Exists | Conflicts with stored hash | Integrity conflict: Transaction C rolls back | `REVIEWING` (Preserved) | `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`) |
| Any non-`REVIEWING` | Exists | Any | Invariant violation: corrupted persistence | `BLOCKED` / `HUMAN_REQUIRED` | `INTEGRITY_CHECK_FAILED` |

---

## 8. Audit Event Registration & Failure Semantics

The proposed audit events are registered under status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `WORKSPACE_BINDING_CREATED`: Recorded in Subtask P04A when workspace binding is established.
2. `VERIFICATION_LEASE_ACQUIRED`: Recorded in Subtask P04C upon lease grant.
3. `VERIFICATION_LEASE_RELEASED`: Recorded in Subtask P04C upon lease completion.
4. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C upon successful ReviewBundle synthesis and persistence.
5. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed diagnostic transaction if compilation fails, clock regresses ($T_1 < T_0$), or bundle hash conflicts with a pre-existing bundle.

---

## 9. Conclusion & Next Steps

PROPOSAL-P04-001 Revision 10 resolves all six findings (`P04-ARCH-R9-001` through `P04-ARCH-R9-006`) with complete schema alignment, crash-safe latency semantics, clean worktree verification, bounded lease authority, and deterministic state transitions.

Following External Supervisor review:
1. `DRAFT-ADR-018` is aligned to Revision 10.
2. `PLAN-P04-EVIDENCE-REVIEW` is updated to Revision 10.
3. Once accepted, Task Contract `CONTRACT-TASK-P04-001` may be drafted for Subtask P04A.
