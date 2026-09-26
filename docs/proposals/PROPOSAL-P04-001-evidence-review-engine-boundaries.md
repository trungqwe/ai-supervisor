# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Boundaries, and Verification Isolation

> **Proposal ID**: `PROPOSAL-P04-001`
> **Revision**: 9
> **Title**: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Audited Baseline**: `a6290ce4c444e345e47aace6c488c3cc4ff33a0d`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_8`
> **Supersedes**: `PROPOSAL-P04-001` Revision 8
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R8-001` through `P04-ARCH-R8-005` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_007.md`).
> **Requirement & Governance Note**: Concurrently submits `PROPOSAL-P04-002` (Revision 3) for formal NFR-008 performance SLA reconciliation. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until approved by External Supervisor.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 007 required remediation of five critical architectural areas:
1. `worker_claims` schema cardinality violated the canonical 1-to-1 attempt model; `review_artifacts` paths included arbitrary file extensions; latency timestamps and intervals lacked durable DDL constraints (`P04-ARCH-R8-001`).
2. AppContainer process I/O specification claiming zero inherited handles contradicted anonymous pipe standard handle redirection; Job Object containment lacked atomic assignment at creation time on Windows 10+ (`P04-ARCH-R8-002`).
3. Snapshot creation timing was improperly positioned before Transaction A report intake; snapshot authority improperly relied on reported head; Git allowlist omitted necessary commands; linked-worktree index path resolution was flawed (`P04-ARCH-R8-003`).
4. Lease admission lacked validation of state, current attempt, and existing evidence; concurrency assumption contradicted v1 sequential model; fencing token isolation in filesystem was incomplete (`P04-ARCH-R8-004`).
5. Audit event literal `AUDIT_EVENT_BUNDLE_GENERATED` was falsely claimed as canonical from Phase P02; failure audit semantics conflated transaction rollback with same-transaction event recording (`P04-ARCH-R8-005`).

This proposal establishes the definitive architecture resolving these findings across six cohesive design sections.

---

## 2. Summary of Architectural Resolutions (Revision 9)

1. **WorkerClaim Cardinality & Canonical Path Derivation (`P04-ARCH-R8-001`)**:
   - `worker_claims` enforces canonical 1-to-1 cardinality with `UNIQUE(attempt_id)`. Reported head SHA, claimed changed files, claimed test results, and claims payload are persisted in structured JCS JSON matching `worker-report.schema.json`.
   - `canonical_relative_path` in `review_artifacts` is deterministically derived from `captured_sha256`: `artifacts/<first-two-hex>/<captured_sha256>` without file extensions. CHECK constraints strictly reject colons, backslashes, path traversal, extra segments, and hash mismatches.
   - `evidence_sets` persists `evidence_committed_at_epoch_ms`. `review_bundles` persists `evidence_committed_at_epoch_ms`, `bundle_committed_at_epoch_ms`, and `compilation_latency_ms` with CHECK constraints enforcing `0 <= compilation_latency_ms <= 3000 ms` and `bundle_committed_at >= evidence_committed_at`.

2. **Windows AppContainer Process I/O & Job Containment (`P04-ARCH-R8-002`)**:
   - Standard process I/O configured with `STARTF_USESTDHANDLES`, stdout/stderr anonymous pipe write handles, `bInheritHandles = TRUE`, and `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` containing strictly the designated standard handles. Parent read pipe ends and all other handles are strictly non-inheritable.
   - Supervisor streaming reader enforces 10 MB byte cap and execution timeout; closes parent write ends immediately; closes handles in proper order to avoid EOF hangs.
   - Process is atomically assigned to the dedicated Job Object at creation on Windows 10+ using `PROC_THREAD_ATTRIBUTE_JOB_LIST`. `breakaway process creation flags` is eliminated.
   - Attribute list contains at minimum `SECURITY_CAPABILITIES`, `HANDLE_LIST`, and `JOB_LIST`.
   - Sandbox DACL replaced with minimal permissions (`FILE_GENERIC_READ | FILE_GENERIC_WRITE | FILE_GENERIC_EXECUTE | DELETE`).
   - AppContainer profile moniker restricted to valid Win32 characters, length <= 64 chars, derived from hash of `attempt_id`, with complete create/derive/delete and startup orphan cleanup lifecycle.

3. **Snapshot Timing, Git Allowlist & Linked-Worktree Resolution (`P04-ARCH-R8-003`)**:
   - Snapshot is created strictly AFTER Transaction A report intake completes (Task in `REPORT_READY`, binding revalidated).
   - Snapshot is sourced strictly from actual independently verified HEAD commit (`actual_head_sha`), never worker claims.
   - Git allowlist explicitly adds `git ls-tree -rz --full-tree <actual_head_sha>`, `git cat-file --batch`, `git rev-parse --git-path index`, and `git rev-parse --absolute-git-dir`.
   - Index path for linked worktrees is resolved using `git rev-parse --git-path index`, canonicalized, and verified for gitdir ownership before hashing.
   - Caps enforced: max 10,000 files, per-blob byte cap, total snapshot cap, streaming extraction, disk quota, and fail-closed cleanup of partial snapshots.
   - Rejection of symlinks, gitlinks, and reparse points documented as v1 fail-closed stop conditions.
   - Manifest binds `actual_head_sha`, relative path, Git file mode, byte length, and blob object ID.

4. **Lease Admission Prerequisites & Sequential Fencing (`P04-ARCH-R8-004`)**:
   - `BEGIN IMMEDIATE` admission requires: `tasks.state = 'REPORT_READY'`, attempt is `tasks.current_attempt`, exact contract/attempt lineage, valid worktree binding, unique `worker_claims` persisted, and no `evidence_sets` for the attempt.
   - If task is already `EVIDENCE_READY` or `REVIEWING`, supervisor reads committed authority and does not re-run collectors or test runners.
   - V1 verification requests execute strictly sequentially; total budget is the sum of request timeouts plus Git collector timeout (10s) and bounded overhead (15s). Concurrent request aggregation is eliminated.
   - Filesystem isolation scopes sandboxes and staging directories by fencing token: `sandboxes/<attempt_id>/<fencing_token>/` and `.staging/<attempt_id>-<fencing_token>-<uuid>.tmp`.
   - Transaction B validates fencing token, `ACTIVE` state, unexpired deadline, and current attempt immediately prior to commit.
   - Reclaiming expired lease terminates and joins the previous Job Object.

5. **Audit Event Governance & Two-Phase Failure Semantics (`P04-ARCH-R8-005`)**:
   - Proposed audit event types introduced with status `PROPOSED_UNTIL_ADR_ACCEPTANCE`: `REVIEW_BUNDLE_GENERATED` and `REVIEW_BUNDLE_COMPILATION_REJECTED`. False claims of canonical approval are eliminated.
   - Two-phase failure audit semantics: Transaction C rolls back cleanly via SQLite WAL upon clock regression, latency violation, schema failure, or hash conflict; a separate fail-closed diagnostic transaction writes `REVIEW_BUNDLE_COMPILATION_REJECTED` into `audit_events` without transitioning TaskState to `REVIEWING`.
   - Canonical event catalog reconciliation deferred until ADR-018 acceptance.

---

## 3. Worktree Authority, Immutable Bindings & Seam Integration

### 3.1. Two-Transaction Lifecycle
1. **Dispatch Binding Registration Transaction**: Integrated into the P03 dispatch seam (`PrepareBoundDispatch` / `DISPATCH_BOUND`) in `internal/dispatch` and `internal/store`. Gathers physical volume serial number and 128-bit file ID via `FileIdInfo`, formats them as 16 and 32 lowercase hex characters, and commits `attempt_workspace_bindings` before `RecordSendRequested`.
2. **Transaction A (Report Intake)**: Invoked when worker signals report completion. Reads and revalidates existing physical binding; persists structured claims into `worker_claims` (Schema v6); transitions `tasks.state`: `RUNNING -> REPORT_READY`. Transaction A never creates workspace bindings.

### 3.2. Schema v6 DDL: `attempt_workspace_bindings` and `worker_claims`

```sql
-- Schema v6: attempt_workspace_bindings
CREATE TABLE attempt_workspace_bindings (
    attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL CHECK (LENGTH(session_id) > 0),
    terminal_generation TEXT NOT NULL CHECK (LENGTH(terminal_generation) > 0),
    canonical_worktree_path TEXT NOT NULL CHECK (LENGTH(canonical_worktree_path) > 0),
    managed_root_final_path TEXT NOT NULL CHECK (LENGTH(managed_root_final_path) > 0),
    volume_serial_hex TEXT NOT NULL CHECK (LENGTH(volume_serial_hex) = 16 AND NOT (volume_serial_hex GLOB '*[^0-9a-f]*')),
    file_id_hex TEXT NOT NULL CHECK (LENGTH(file_id_hex) = 32 AND NOT (file_id_hex GLOB '*[^0-9a-f]*')),
    pinned_ao_commit TEXT NOT NULL CHECK (LENGTH(pinned_ao_commit) = 40 AND NOT (pinned_ao_commit GLOB '*[^0-9a-f]*')),
    bound_at TEXT NOT NULL CHECK (LENGTH(bound_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
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

CREATE TRIGGER trg_attempt_workspace_bindings_no_update
BEFORE UPDATE ON attempt_workspace_bindings
BEGIN
    SELECT RAISE(ABORT, 'attempt_workspace_bindings is immutable');
END;

CREATE TRIGGER trg_attempt_workspace_bindings_no_delete
BEFORE DELETE ON attempt_workspace_bindings
BEGIN
    SELECT RAISE(ABORT, 'attempt_workspace_bindings is immutable');
END;

-- Schema v6: worker_claims (Canonical cardinality: exactly one WorkerClaim per attempt)
CREATE TABLE worker_claims (
    claim_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    reported_head_sha TEXT NOT NULL CHECK (LENGTH(reported_head_sha) = 40 AND NOT (reported_head_sha GLOB '*[^0-9a-f]*')),
    claimed_files_json TEXT NOT NULL CHECK (LENGTH(claimed_files_json) > 0),
    claimed_tests_json TEXT NOT NULL CHECK (LENGTH(claimed_tests_json) > 0),
    claims_payload_json TEXT NOT NULL CHECK (LENGTH(claims_payload_json) > 0),
    reported_at TEXT NOT NULL CHECK (LENGTH(reported_at) > 0),
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

## 4. Hardened Git Evidence Collector & Linked Worktrees

1. **Host-Pinned Binary Resolution**: Git executable resolved strictly from `SUPERVISOR_GIT_BIN`, version and SHA-256 hash verified per NFR-007.
2. **Windows Configuration Isolation**: Dedicated `<SUPERVISOR_STATE_ROOT>/trusted_empty_git/` with empty config and empty hooks directory.
3. **Mandatory Invocations & Flags**:
   - Every Git invocation passes `-c core.hooksPath=<trusted_empty_hooks>` and `-c core.fsmonitor=false`.
   - `--no-pager` is strictly passed as a CLI argument, not an environment variable (`GIT_PAGER=cat`).
4. **Git Command Allowlist**:
   - `git rev-parse --verify <ref>`
   - `git rev-parse --git-path index`
   - `git rev-parse --absolute-git-dir`
   - `git merge-base <base> <head>`
   - `git log --no-ext-diff --no-textconv --format=...`
   - `git diff --no-ext-diff --no-textconv --raw / --patch`
   - `git ls-tree -rz --full-tree <actual_head_sha>`
   - `git cat-file --batch`
5. **Linked Worktree Index Resolution**: For Git linked worktrees, `.git` is a file pointing to a gitdir. Supervisor resolves index path using `git rev-parse --git-path index`, canonicalizes path, and verifies handle ownership before computing the authoritative index SHA-256.
6. **Streaming Controls**: 10s execution timeout; 10 MB streaming byte cap terminating process tree immediately on exceedance.
7. **Traceability**: Independent Git diff collection maps to **FR-008**; scope comparison maps to **FR-009** and **SEC-005**.

---

## 5. Windows AppContainer Process Isolation & Job Containment

1. **Exclusive Architecture Choice**: Windows AppContainer for v1. Dual-path implementations and non-AppContainer token alternatives are eliminated.
2. **Process I/O & Handle Inheritance**:
   - Standard process I/O configured with `STARTF_USESTDHANDLES`.
   - Stdout and stderr redirected to anonymous pipe write handles; stdin closed or directed to nul.
   - `bInheritHandles = TRUE` in `CreateProcessW`.
   - `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` passed to `UpdateProcThreadAttribute` containing strictly the designated stdout and stderr write handles.
   - Parent read pipe ends and all other supervisor handles created without `HANDLE_FLAG_INHERIT`.
   - Supervisor streaming reader applies 10 MB byte cap and execution timeout; closes parent write ends immediately after process creation; closes pipe ends in proper order so EOF is signaled cleanly without hangs.
3. **Job Object Containment**:
   - Sandboxed process is assigned to dedicated Job Object atomically at creation on Windows 10+ using `PROC_THREAD_ATTRIBUTE_JOB_LIST`.
   - `breakaway process creation flags` is eliminated.
   - Job Object limits: `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, breakaway disabled, 2 GB memory cap.
4. **Attribute List Specification**: Contains at minimum `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES`, `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`, and `PROC_THREAD_ATTRIBUTE_JOB_LIST`.
5. **Sandbox Root & DACLs**:
   - Sandbox root: `<SUPERVISOR_STATE_ROOT>/sandboxes/<attempt_id>/<fencing_token>/`.
   - DACL granted to AppContainer SID: minimal necessary rights (`FILE_GENERIC_READ | FILE_GENERIC_WRITE | FILE_GENERIC_EXECUTE | DELETE`) strictly within the attempt sandbox directory.
   - Snapshot root: granted read/execute DACL.
6. **AppContainer Profile Moniker Lifecycle**:
   - Valid Win32 characters only (alphanumerics, periods, dashes, underscores).
   - Length <= 64 characters.
   - Moniker format: `appcontainer-attempt-<first_16_hex_of_sha256>`.
   - Lifecycle: create/derive via `CreateAppContainerProfile` / `DeriveAppContainerSidFromAppContainerName`, DACL cleanup, deletion via `DeleteAppContainerProfile`, and fail-closed orphan cleanup on supervisor daemon startup.

---

## 6. Immutable Source Snapshot Protocol

1. **Snapshot Timing**: Created strictly AFTER Transaction A report intake completes (Task in `REPORT_READY`, physical binding revalidated).
2. **Source Authority**: Sourced strictly from actual independently verified HEAD commit (`actual_head_sha`), never worker claims.
3. **Streaming Extraction**: Streamed via Git `ls-tree -rz --full-tree <actual_head_sha>` and `git cat-file --batch` into `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/`.
4. **Validation Caps & Quotas**:
   - File-count cap: maximum 10,000 files.
   - Per-blob byte cap: 10 MB per file.
   - Total snapshot byte cap: 100 MB total.
   - Disk quota checking and fail-closed cleanup of partial snapshots on error.
5. **Path Validation & Stop Conditions**: Every path component validated; directory traversal (`..`), junctions, reparse points, symlinks, and gitlinks are strictly rejected as a v1 fail-closed stop condition.
6. **Immutable Manifest**: Binds `actual_head_sha`, relative path, Git mode, byte length, and blob object ID; verified before test invocation.
7. **Handle Invariant**: Root worktree handle opened without `FILE_SHARE_DELETE` and held across verification. Tests never touch the active worktree, and changes are never copied back.

---

## 7. Pipeline Transactions, Fencing & Content-Addressed Store

### 7.1. Three Durable Transaction Boundaries
1. **Transaction A (Report Intake)**: Validates report, stores structured `worker_claims` (Schema v6), transitions `tasks.state`: `RUNNING -> REPORT_READY`.
2. **Transaction B (Evidence Finalization)**: Ingests in-memory outputs, stages and deduplicates artifacts into `artifacts/<first-two-hex>/<captured_sha256>`, persists `evidence_sets` and `review_artifacts(evidence_set_id)` (Schema v9) with `evidence_committed_at_epoch_ms`, releases lease, transitions `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
3. **Transaction C (ReviewBundle Compilation)**: Synthesizes RFC 8785 JCS canonical ReviewBundle, inserts `review_bundles(evidence_set_id)` with persisted `bundle_committed_at_epoch_ms` and `compilation_latency_ms`, records proposed audit event `REVIEW_BUNDLE_GENERATED`, transitions `tasks.state`: `EVIDENCE_READY -> REVIEWING`.

### 7.2. Lease Admission Prerequisites & Reclaim
- `BEGIN IMMEDIATE` admission query validates:
  `tasks.state = 'REPORT_READY'`, attempt is `tasks.current_attempt`, lineage is intact, worktree binding exists and is revalidated, `worker_claims` exists, and `evidence_sets` does not exist for the attempt.
- If task is already `EVIDENCE_READY` or `REVIEWING`, supervisor reads committed authority and does not re-run tests.
- Reclaiming an expired lease terminates and joins the previous Job Object.
- Fencing token isolation applies to sandbox directory (`sandboxes/<attempt_id>/<fencing_token>/`) and staging path (`.staging/<attempt_id>-<fencing_token>-<uuid>.tmp`).
- Transaction B verifies fencing token, `ACTIVE` state, unexpired deadline, and current attempt immediately prior to commit.

### 7.3. Schema v9 DDL: Leases, Evidence Sets, Artifacts, Bundles

```sql
-- Schema v9: task_verification_leases
CREATE TABLE task_verification_leases (
    task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'RELEASED')),
    worker_id TEXT NOT NULL CHECK (LENGTH(worker_id) > 0),
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    acquired_at_epoch_ms INTEGER NOT NULL CHECK (acquired_at_epoch_ms > 0),
    expires_at_epoch_ms INTEGER NOT NULL CHECK (expires_at_epoch_ms > acquired_at_epoch_ms),
    released_at_epoch_ms INTEGER,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        (state = 'ACTIVE' AND released_at_epoch_ms IS NULL) OR
        (state = 'RELEASED' AND released_at_epoch_ms IS NOT NULL AND released_at_epoch_ms >= acquired_at_epoch_ms)
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

CREATE TRIGGER trg_task_verification_leases_lineage_update_guard
BEFORE UPDATE ON task_verification_leases
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

-- Schema v9: evidence_sets
CREATE TABLE evidence_sets (
    evidence_set_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    git_evidence_json TEXT NOT NULL CHECK (LENGTH(git_evidence_json) > 0),
    test_evidence_json TEXT NOT NULL CHECK (LENGTH(test_evidence_json) > 0),
    policy_findings_json TEXT NOT NULL CHECK (LENGTH(policy_findings_json) > 0),
    unverified_claims_json TEXT NOT NULL CHECK (LENGTH(unverified_claims_json) > 0),
    evidence_committed_at_epoch_ms INTEGER NOT NULL CHECK (evidence_committed_at_epoch_ms > 0),
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

-- Schema v9: review_artifacts (Owned by Subtask P04D, references evidence_sets)
CREATE TABLE review_artifacts (
    artifact_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    artifact_type TEXT NOT NULL CHECK (LENGTH(artifact_type) > 0),
    media_type TEXT NOT NULL CHECK (LENGTH(media_type) > 0),
    encoding TEXT NOT NULL CHECK (LENGTH(encoding) > 0),
    canonical_relative_path TEXT NOT NULL,
    captured_sha256 TEXT NOT NULL CHECK (LENGTH(captured_sha256) = 64 AND NOT (captured_sha256 GLOB '*[^0-9a-f]*')),
    full_stream_sha256 TEXT NOT NULL CHECK (LENGTH(full_stream_sha256) = 64 AND NOT (full_stream_sha256 GLOB '*[^0-9a-f]*')),
    original_bytes INTEGER NOT NULL CHECK (original_bytes >= 0),
    captured_bytes INTEGER NOT NULL CHECK (captured_bytes >= 0 AND captured_bytes <= original_bytes),
    is_truncated INTEGER NOT NULL CHECK (
        is_truncated IN (0, 1) AND (
            (is_truncated = 1 AND captured_bytes < original_bytes) OR
            (is_truncated = 0 AND captured_bytes = original_bytes)
        )
    ),
    created_at TEXT NOT NULL CHECK (LENGTH(created_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256 AND
        canonical_relative_path NOT GLOB '*:*' AND
        canonical_relative_path NOT GLOB '*\\*' AND
        canonical_relative_path NOT GLOB '*..*' AND
        canonical_relative_path NOT GLOB '*//*'
    )
);

CREATE TRIGGER trg_review_artifacts_lineage_guard
BEFORE INSERT ON review_artifacts
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match task_id, attempt_id, contract_id')
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

-- Schema v9: review_bundles (Owned by Subtask P04D, references evidence_sets)
CREATE TABLE review_bundles (
    bundle_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL UNIQUE REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    bundle_payload_json TEXT NOT NULL CHECK (LENGTH(bundle_payload_json) > 0),
    bundle_hash TEXT NOT NULL CHECK (LENGTH(bundle_hash) = 64 AND NOT (bundle_hash GLOB '*[^0-9a-f]*')),
    evidence_committed_at_epoch_ms INTEGER NOT NULL CHECK (evidence_committed_at_epoch_ms > 0),
    bundle_committed_at_epoch_ms INTEGER NOT NULL CHECK (bundle_committed_at_epoch_ms > 0),
    compilation_latency_ms INTEGER NOT NULL CHECK (compilation_latency_ms BETWEEN 0 AND 3000),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        bundle_committed_at_epoch_ms >= evidence_committed_at_epoch_ms AND
        compilation_latency_ms = (bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms)
    )
);

CREATE TRIGGER trg_review_bundles_lineage_guard
BEFORE INSERT ON review_bundles
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match task_id, attempt_id, contract_id')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.task_id = NEW.task_id
          AND e.attempt_id = NEW.attempt_id
          AND e.contract_id = NEW.contract_id
    );
END;

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

## 8. Audit Event Registration & Failure Semantics

Proposed audit events with status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `REVIEW_BUNDLE_GENERATED`:
   - Actor: `Supervisor`
   - Trigger: Successful commit of Transaction C.
   - Details: `bundle_id`, `bundle_hash`, `compilation_latency_ms`, `evidence_set_id`.
2. `REVIEW_BUNDLE_COMPILATION_REJECTED`:
   - Actor: `Supervisor`
   - Trigger: Failure of Transaction C (clock regression, latency > 3000 ms, JCS schema violation, bundle hash conflict).
   - Semantics: Transaction C rolls back cleanly via SQLite WAL. A separate fail-closed diagnostic transaction writes `REVIEW_BUNDLE_COMPILATION_REJECTED` into `audit_events` without altering TaskState (`EVIDENCE_READY` preserved).
   - Details: `attempt_id`, `failure_reason`, timestamps, error diagnostics.

Canonical event catalog reconciliation will occur only after formal External Supervisor approval of ADR-018.

---

## 9. Conclusion & Next Steps

This proposal completely resolves all findings (`P04-ARCH-R8-001` through `P04-ARCH-R8-005`). Formal Task Contract release and code authoring remain blocked until External Supervisor review and approval.
