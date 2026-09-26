# ADR-018: Evidence & Review Engine Architecture, Execution Isolation, and Verification Governance

> **Status**: DRAFT (`REVISION_10_REQUIRED` / Remediated to Revision 10)
> **Date**: 2026-09-26
> **Audited Baseline**: `6dbd7e26d59e22171271921b86c01bd6f215e59a`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_9`
> **Deciders**: AI Engineering Supervisor Architecture Council, External Supervisor
> **Related Architecture**: `docs/04_ARCHITECTURE.md` (Section 7), `docs/05_DOMAIN_MODEL.md`, `docs/10_REVIEW_BUNDLE.md`
> **Related Requirements**: `docs/02_REQUIREMENTS.md` (FR-008, NFR-008 via PROPOSAL-P04-002)
> **Supersedes**: `DRAFT-ADR-018` Revision 9
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R9-001` through `P04-ARCH-R9-006` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_008.md`).

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 008 recorded six architectural findings requiring Revision 10 remediation:
1. `worker_claims` schema enforced an exact 40-char SHA check, conflicting with canonical `worker-report.schema.json` allowing 7-40 hex chars, and lacked unambiguous mappings between WorkerReport, WorkerClaim, and ReviewBundle (`P04-ARCH-R9-001`).
2. NFR-008 latency check `compilation_latency_ms <= 3000` created a permanent database deadlock if the daemon restarted between Transaction B and C or exceeded 3 seconds (`P04-ARCH-R9-002`).
3. Terminating processes upon exceeding hard byte limits contradicted metadata claiming `full_stream_sha256` and `original_bytes` were known (`P04-ARCH-R9-003`).
4. Staged, unstaged, and untracked worktree modifications were unconstrained before snapshot extraction (`P04-ARCH-R9-004`).
5. Lease reclaim across daemon restarts claimed to terminate old Job Objects without valid handles, and verification request timeouts lacked checked arithmetic and aggregate caps (`P04-ARCH-R9-005`).
6. ReviewBundle replay and hash conflict handling assigned incorrect TaskStates (`P04-ARCH-R9-006`).

This ADR formalizes the architectural decisions and schema contracts governing Phase P04.

---

## 2. Decision Drivers

1. **Evidence Integrity & Non-Repudiation**: The supervisor must independently corroborate worker claims using immutable source snapshots and sandboxed execution without trusting worker memory or worker assertions.
2. **Canonical Schema Alignment**: All persistence contracts must adhere to canonical schemas (`worker-report.schema.json`, `review-bundle.schema.json`, `task-contract.schema.json`) and the 9-level decision hierarchy of `docs/24_CHANGE_GOVERNANCE.md`.
3. **Crash Consistency & Deadlock Prevention**: Database transactions and state transitions must be resilient across daemon restarts and scheduling latency without permanently stranding tasks.
4. **Clean Worktree Guarantee**: Worker outputs must be fully committed; dirty staged, unstaged, or untracked state must fail closed.
5. **Execution Containment**: Verification tests must run under OS-level isolation (Windows AppContainers) with explicit handle inheritance and network denial.

---

## 3. Considered Options

- **Option 1**: Monolithic in-process execution directly in worker worktree. (Rejected: Vulnerable to TOCTOU, handle leaks, network escape, and worktree contamination).
- **Option 2**: Ephemeral Git branches in worker repository without AppContainer sandboxing. (Rejected: Test execution could compromise host OS; non-isolated processes lack network denial).
- **Option 3 (Selected)**: Decoupled three-transaction verification pipeline with Windows AppContainer isolation, deterministic source snapshot extraction, strict clean worktree enforcement, and crash-safe ReviewBundle synthesis.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority, Immutable Bindings & Canonical WorkerClaim (P04-ARCH-R9-001)

1. **Two-Transaction Workspace Lifecycle**:
   - **Binding Creation (P04A)**: Binds `attempt_id` to the physical worktree (`FileIdInfo`) and initial commit SHA prior to dispatch, in state `ACTIVE`.
   - **Report Intake Transaction A (P04A)**: Triggered upon worker completion. Validates worktree identity, verifies worktree cleanliness, validates `WorkerReport` via JSON schema in Go, canonicalizes via RFC 8785 JCS, inserts into `worker_claims`, updates binding to `RETAINED_FOR_VERIFICATION`, and transitions `tasks.state`: `RUNNING -> REPORT_READY`.
2. **Verbatim Head SHA Preservation**:
   - `worker_claims.reported_head_sha` stores the exact reported value (`CHECK (LENGTH(reported_head_sha) BETWEEN 7 AND 40 AND NOT (reported_head_sha GLOB '*[^0-9a-f]*'))`).
   - The verified `actual_head_sha` belongs strictly to Evidence (`evidence_sets.git_evidence_json`), never in `worker_claims`.
3. **Canonical Mapping**:
   - `WorkerReport.head_sha` -> `WorkerClaim.reported_head_sha`
   - `WorkerReport.files_changed` -> `WorkerClaim.claimed_files_changed`
   - `WorkerReport.tests` -> `WorkerClaim.claimed_tests`
   - `WorkerReport.worker_claims` (array of strings) -> `WorkerClaim.textual_claims`
   - `WorkerReport.build_status` -> `WorkerClaim.build_status`
   - Canonical `WorkerClaim` object is embedded directly into ReviewBundle `worker_claims`.
4. **Validation and JCS Enforcement**:
   - Go application enforces full schema validation, data types, and RFC 8785 JCS canonicalization before database insertion.
   - SQLite DDL enforces `json_valid()` and `json_type()` as defense-in-depth without claiming SQLite proves JCS.

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

### Decision 2: Hardened In-Memory Git Collector & Clean Worktree Allowlist (P04-ARCH-R9-004)

1. **Clean Worktree Policy (v1)**:
   - Worker must commit all changes; workspace worktree and index must be clean at report intake.
   - Transaction A fails closed if staged, unstaged, or untracked changes exist.
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

### Decision 4: Verification Authority & Crash-Safe Latency Semantics via PROPOSAL-P04-002 (P04-ARCH-R9-002)

1. **Four Timestamp Concepts**:
   - Pre-commit write timestamp (`write_timestamp_epoch_ms` in application layer before `tx.Commit()`).
   - WAL commit completion (when SQLite fsync/commit finishes).
   - Monotonic in-process duration (`time.Since()`, strictly single-process scope).
   - Durable wall-clock diagnostic (`bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms`).
2. **Deadlock Elimination**:
   - Database constraint `compilation_latency_ms <= 3000` is eliminated.
   - ReviewBundles persist with `nfr008_met` (1 if latency <= 3000 ms, 0 if latency > 3000 ms) and `latency_measurement_status` ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART', 'MEASUREMENT_TIMEOUT').
   - SLA breaches are recorded as diagnostic evidence; Transaction C commits and advances task to `REVIEWING` without deadlocking `EVIDENCE_READY`.

### Decision 5: Three Durable Pipeline Transactions, Admission & Bounded Fencing (P04-ARCH-R9-005, R9-006)

1. **Transaction Boundaries**:
   - **Transaction A**: Commits `worker_claims`, retains workspace binding, transitions `RUNNING -> REPORT_READY`.
   - **Transaction B**: Commits `evidence_sets` and `review_artifacts`, releases lease, transitions `REPORT_READY -> EVIDENCE_READY`.
   - **Transaction C**: Commits `review_bundles`, commits proposed audit event `REVIEW_BUNDLE_GENERATED`, transitions `EVIDENCE_READY -> REVIEWING`.
2. **Reclaim Authority Taxonomy**:
   - Same-daemon: TerminateJobObject via in-memory handle registry + WaitForSingleObject join.
   - Daemon restart: P03 machine lock exclusivity + Windows `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`.
   - Unproven owner: Fail-closed; do not reclaim lease.
3. **Bounded TTL Arithmetic**:
   - Checked 64-bit integer arithmetic rejecting overflow.
   - Stage B contract validation enforces `len(verification_requests) <= 20` and `aggregate_timeout <= 600s`.
   - `verification_leases.ttl_seconds CHECK (ttl_seconds BETWEEN 1 AND 600)`.
4. **ReviewBundle Replay and Conflict State Matrix**:
   - Compile failure: Transaction C rolls back, task remains `EVIDENCE_READY`, emits `REVIEW_BUNDLE_COMPILATION_REJECTED`.
   - Idempotent replay: Returns existing bundle, task remains `REVIEWING`.
   - Hash conflict: Tampering conflict, task remains `REVIEWING`, Transaction C rolls back, emits `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`), escalates to human supervisor.
   - Orphaned bundle: Invariant violation, fails closed to `BLOCKED`/`HUMAN_REQUIRED`.

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

### Decision 6: Durable Content-Addressed Artifact Store, Review Schema & Proposed Audit Events (P04-ARCH-R9-003)

1. **Artifact Streaming & Byte Limits**:
   - Separate capture limit (10 MB saved to disk) from hard safety limit (50 MB process termination).
   - `stream_state` captures exact boundary conditions: `COMPLETE_EOF`, `TRUNCATED_AT_CAPTURE_LIMIT`, `HARD_LIMIT_TERMINATED`, `TIMEOUT_ABORTED`.
   - If terminated before EOF, `full_stream_sha256 = NULL`, `is_truncated = 1`, recording observed bytes without claiming complete hash is known.
2. **Content-Addressed Storage Layout**:
   - Relative path: `artifacts/<first-two-hex>/<captured_sha256>` without file extensions.

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

---

## 5. Consequences

### Positive
- Strict conformance with canonical `worker-report.schema.json` and `review-bundle.schema.json`.
- Deadlock-free crash recovery across daemon restarts.
- Complete worktree cleanliness verification preventing hidden uncommitted changes.
- Robust, bounded lease management and OS-level sandboxing.

### Negative / Trade-offs
- Verification execution duration is bounded by real test execution physics rather than sub-3-second expectations (reconciled cleanly via PROPOSAL-P04-002).
- Clean worktree policy v1 requires workers to keep worktree and index completely clean at report time.

---

## 6. Migration and Schema Ownership

- Subtask P04A owns Schema Migration v6 (`attempt_workspace_bindings`, `worker_claims`).
- Subtask P04C owns Schema Migration v9 (`verification_leases`, `evidence_sets`, `review_artifacts`, `review_bundles`).
