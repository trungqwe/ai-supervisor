# DRAFT ADR-018: Evidence Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

> **Status**: `DRAFT_PENDING_EXTERNAL_APPROVAL`
> **Revision**: 9
> **Date**: 2026-09-26
> **Audited Baseline**: `a6290ce4c444e345e47aace6c488c3cc4ff33a0d`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_8`
> **Decision Owners**: AI Engineering Supervisor Team
> **Supersedes**: `DRAFT-ADR-018` Revision 8
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R8-001` through `P04-ARCH-R8-005` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_007.md`).
> **Requirement & Governance Note**: Cannot be accepted or marked ready until `PROPOSAL-P04-002-review-bundle-latency-semantics.md` is approved by External Supervisor. `docs/02_REQUIREMENTS.md` remains unmodified.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine** to provide independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 007 required remediation of five critical architectural areas:
1. `worker_claims` schema cardinality violated the canonical 1-to-1 attempt model; `review_artifacts` paths included arbitrary file extensions; latency timestamps and intervals lacked durable DDL constraints (`P04-ARCH-R8-001`).
2. AppContainer process I/O specification claiming zero inherited handles contradicted anonymous pipe standard handle redirection; Job Object containment lacked atomic assignment at creation time on Windows 10+ (`P04-ARCH-R8-002`).
3. Snapshot creation timing was improperly positioned before Transaction A report intake; snapshot authority improperly relied on reported head; Git allowlist omitted necessary commands; linked-worktree index path resolution was flawed (`P04-ARCH-R8-003`).
4. Lease admission lacked validation of state, current attempt, and existing evidence; concurrency assumption contradicted v1 sequential model; fencing token isolation in filesystem was incomplete (`P04-ARCH-R8-004`).
5. Audit event literal `AUDIT_EVENT_BUNDLE_GENERATED` was falsely claimed as canonical from Phase P02; failure audit semantics conflated transaction rollback with same-transaction event recording (`P04-ARCH-R8-005`).

This ADR establishes the definitive architectural decisions resolving these findings.

---

## 2. Decision Drivers

- **Security & Integrity**: Complete isolation against malicious or buggy worker processes, hooks, external diff tools, and network exfiltration (`SEC-001`, `SEC-003`, `OPS-003`).
- **Upstream Alignment**: Respect pinned Agent Orchestrator (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) authority over worker sessions and physical worktrees.
- **Fail-Closed Verification**: Zero reliance on worker self-reporting; missing or mismatched physical bindings, handles, or hashes abort verification immediately.
- **State Machine Atomicity**: Clear transactional boundaries with monotonic lease fencing and deterministic crash/replay recovery.
- **Requirement Governance**: Strict adherence to the 9-level change governance hierarchy (`docs/24_CHANGE_GOVERNANCE.md`); zero silent modification of canonical requirements.

---

## 3. Considered Options

1. **Option 1: In-Process Ephemeral Verification (Rejected)**: Running Git and test commands directly under the Supervisor daemon process without OS-level isolation. Rejected due to severe security and tampering vulnerabilities (`SEC-002`, `SEC-003`).
2. **Option 2: Disposable AO Session Proof Gate (Rejected)**: Gating Phase P04 contract release on running disposable worker sessions via upstream AO. Rejected (`P04-ARCH-R5-001`, `P04-ARCH-R6-002`) because AO contains no inert harness in `AllHarnesses`, and running live agent CLIs creates uncontrollable safety and token hazards.
3. **Option 3: Hardened In-Memory Engine with AppContainer Security Boundary, Durable Evidence Sets, and Three Transaction Boundaries (Accepted)**: Independent physical OS handle validation, hardened in-memory Git collection, Windows AppContainer security boundary with DACL-protected immutable source snapshots, append-only content-addressed artifact storage, and three distinct durable SQLite transaction boundaries. Non-AppContainer token alternatives are rejected for v1.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority, Immutable Bindings & Canonical WorkerClaim (P04-ARCH-R8-001)

#### 1. Separation of Registration and Intake Transactions
1. **Dispatch Binding Registration Transaction**: Executed immediately after Agent Orchestrator successfully materializes the worktree, integrated directly into the P03 dispatch seam (`PrepareBoundDispatch` / `DISPATCH_BOUND`) and committed prior to `RecordSendRequested`. This binds the physical worktree identity (`volume_serial_hex`, `file_id_hex`, canonical path) to the attempt before worker execution commences.
2. **Transaction A (Report Intake)**: Executed when a worker signals completion. Transaction A strictly reads and verifies that a valid physical binding already exists for the attempt. It persists structured claims into `worker_claims` (Schema v6) with `UNIQUE(attempt_id)` and transitions `tasks.state`: `RUNNING -> REPORT_READY`. Transaction A is strictly prohibited from creating or inserting workspace bindings.

#### 2. Immutable History Decoupled from Mutable Session Lanes
`attempt_workspace_bindings` represents permanent, immutable historical evidence of dispatch. Foreign key coupling to `worker_sessions(session_id)` is removed because `worker_sessions` represents the mutable, single-lane state of a Pair and can be rotated, reassigned, or purged across attempts. Integrity is enforced via triggers cross-referencing `task_attempts` and `dispatch_operations`.

#### 3. Strict Hexadecimal Identity Specification
- `volume_serial_hex`: Exactly 16 lowercase hex characters (`CHECK (LENGTH(volume_serial_hex) = 16 AND NOT (volume_serial_hex GLOB '*[^0-9a-f]*'))`).
- `file_id_hex`: Exactly 32 lowercase hex characters (`CHECK (LENGTH(file_id_hex) = 32 AND NOT (file_id_hex GLOB '*[^0-9a-f]*'))`).

#### 4. Canonical 1-to-1 WorkerClaim Cardinality
Each `TaskAttempt` has exactly one `WorkerClaim` row (`attempt_id UNIQUE`). Reported head SHA, claimed changed files, claimed test results, and claims payload are persisted in structured JCS JSON matching `worker-report.schema.json` and `docs/05_DOMAIN_MODEL.md`.

#### 5. Schema v6 DDL: `attempt_workspace_bindings` and `worker_claims` (Owned by Subtask P04A)

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

### Decision 2: Hardened In-Memory Git Collector & Linked Worktrees (P04-ARCH-R8-003, R8-004)

1. **Host-Pinned Binary Resolution**: Git executable resolved strictly from `SUPERVISOR_GIT_BIN`, version and SHA-256 pinned per `NFR-007`. Arbitrary `PATH` lookups are prohibited.
2. **Windows Configuration Isolation**: Windows cannot use Unix null device syntax for directories. Supervisor creates `<SUPERVISOR_STATE_ROOT>/trusted_empty_git/` with an empty config file and empty hooks directory.
3. **Mandatory CLI Flags & Invocations**:
   - Every Git invocation passes `-c core.hooksPath=<trusted_empty_hooks>` and `-c core.fsmonitor=false`.
   - `--no-pager` is strictly passed as a CLI argument, not an environment variable (`GIT_PAGER=cat`).
4. **Environment Sanitization**:
   - Unset `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_ALTERNATE_OBJECT_DIRECTORIES`.
   - Sanitize `GIT_CONFIG_COUNT`, `GIT_CONFIG_KEY_*`, `GIT_CONFIG_VALUE_*`, `GIT_EXTERNAL_DIFF`, `GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_SSH`, `GIT_SSH_COMMAND`, `GIT_PROTOCOL_FROM_USER`, `GIT_CEILING_DIRECTORIES`.
   - Set `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_SYSTEM`, `GIT_CONFIG_GLOBAL` to empty config, `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`.
5. **Git Command Allowlist**:
   - `git rev-parse --verify <ref>`
   - `git rev-parse --git-path index`
   - `git rev-parse --absolute-git-dir`
   - `git merge-base <base> <head>`
   - `git log --no-ext-diff --no-textconv --format=...`
   - `git diff --no-ext-diff --no-textconv --raw / --patch`
   - `git ls-tree -rz --full-tree <actual_head_sha>`
   - `git cat-file --batch`
6. **Linked Worktree Index Resolution**: For Git linked worktrees, `.git` is a file pointing to a gitdir. Supervisor resolves index path using `git rev-parse --git-path index`, canonicalizes path, and verifies handle ownership before computing the authoritative index SHA-256.
7. **Execution Controls**: 10s execution timeout; 10 MB streaming byte cap terminating the process tree immediately upon exceedance.
8. **Traceability**: Independent Git diff collection maps to **FR-008**; scope comparison maps to **FR-009** and **SEC-005**.

---

### Decision 3: Windows Verification Isolation Security Boundary & Job Containment (P04-ARCH-R8-002, R8-003)

#### 1. Exclusive Selection of Windows AppContainer for v1
Phase P04 v1 commits exclusively to **Windows AppContainer** as the verification execution security boundary. Non-AppContainer token alternatives are eliminated.

#### 2. Process I/O & Handle Inheritance
- Standard process I/O configured with `STARTF_USESTDHANDLES`.
- Stdout and stderr redirected to anonymous pipe write handles; stdin closed or directed to nul.
- `bInheritHandles = TRUE` in `CreateProcessW`.
- `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` passed to `UpdateProcThreadAttribute` containing strictly the designated stdout and stderr write handles.
- Parent read pipe ends and all other supervisor handles created without `HANDLE_FLAG_INHERIT`.
- Supervisor streaming reader applies 10 MB byte cap and execution timeout; closes parent write ends immediately after process creation; closes pipe ends in proper order so EOF is signaled cleanly without hangs.

#### 3. Atomic Job Object Association
- Sandboxed process is assigned to dedicated Job Object atomically at creation on Windows 10+ using `PROC_THREAD_ATTRIBUTE_JOB_LIST`.
- Breakaway process creation flags are eliminated.
- Job Object limits: `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, breakaway disabled, 2 GB memory cap.

#### 4. Attribute List Specification
Contains at minimum `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES`, `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`, and `PROC_THREAD_ATTRIBUTE_JOB_LIST`.

#### 5. Sandbox Root & DACLs
- Sandbox root: `<SUPERVISOR_STATE_ROOT>/sandboxes/<attempt_id>/<fencing_token>/`.
- DACL granted to AppContainer SID: minimal necessary rights (`FILE_GENERIC_READ | FILE_GENERIC_WRITE | FILE_GENERIC_EXECUTE | DELETE`) strictly within the attempt sandbox directory.
- Snapshot root: granted read/execute DACL.

#### 6. AppContainer Profile Moniker Lifecycle
- Valid Win32 characters only (alphanumerics, periods, dashes, underscores).
- Length <= 64 characters.
- Moniker format: `appcontainer-attempt-<first_16_hex_of_sha256>`.
- Lifecycle: create/derive via `CreateAppContainerProfile` / `DeriveAppContainerSidFromAppContainerName`, DACL cleanup, deletion via `DeleteAppContainerProfile`, and fail-closed orphan cleanup on supervisor daemon startup.

#### 7. Pinned Immutable Snapshot Protocol
- **Snapshot Timing**: Created strictly AFTER Transaction A report intake completes (Task in `REPORT_READY`, binding revalidated).
- **Source Authority**: Sourced strictly from actual independently verified HEAD commit (`actual_head_sha`), never worker claims.
- **Streaming Extraction**: Streamed via Git `ls-tree -rz --full-tree <actual_head_sha>` and `git cat-file --batch` into `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/`.
- **Validation Caps & Quotas**: Max 10,000 files, 10 MB per-blob cap, 100 MB total snapshot cap, streaming extraction, disk quota checking, and fail-closed cleanup of partial snapshots on error.
- **Path Validation & Stop Conditions**: Every path component validated; directory traversal (`..`), junctions, reparse points, symlinks, and gitlinks are strictly rejected as a v1 fail-closed stop condition.
- **Immutable Manifest**: Binds `actual_head_sha`, relative path, Git mode, byte length, and blob object ID; verified before test invocation.
- **Handle Invariant**: Root worktree handle opened without `FILE_SHARE_DELETE` and held across verification. Tests never touch the active worktree, and changes are never copied back.

---

### Decision 4: Verification Authority & Latency Semantics via PROPOSAL-P04-002 (P04-ARCH-R8-001, R8-004)

1. Verification commands execute strictly via approved profiles in `VerificationPolicyCatalog` with host-governed absolute paths.
2. In v1, verification requests execute strictly sequentially; total budget is the exact sum of request timeouts plus Git collector timeout (10s) and bounded orchestration overhead (15s). Concurrent request aggregation is eliminated.
3. NFR-008 performance SLA reconciliation is governed by `PROPOSAL-P04-002-review-bundle-latency-semantics.md` (`PENDING_EXTERNAL_REVIEW`), establishing a two-interval model:
   - **Interval 1 (Evidence Acquisition Window)**: Bounded by TaskContract `verification_requests[].timeout_seconds` validated by Stage B, or catalog profile `MaxTimeoutSeconds`. Ends at $T_0$ (`evidence_committed_at_epoch_ms`).
   - **Interval 2 (Bundle Compilation Window)**: Persisted epoch timestamps enforce:
     `bundle_committed_at_epoch_ms >= evidence_committed_at_epoch_ms`
     `compilation_latency_ms = bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms`
     `0 <= compilation_latency_ms <= 3000 ms (3.0 seconds)`
     where T0 is `evidence_committed_at_epoch_ms` and T1 is `bundle_committed_at_epoch_ms`. In-process timing utilizes monotonic duration. Clock regression or latency exceedance causes Transaction C rollback.
4. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until formal approval.

---

### Decision 5: Three Durable Pipeline Transactions, Admission & Fencing (P04-ARCH-R8-004, R8-005)

#### 1. Three Durable Transaction Boundaries
1. **Transaction A (Report Intake)**: Validates worker report, persists `worker_claims` (Schema v6), and transitions `tasks.state`: `RUNNING -> REPORT_READY`. (Does not insert bindings).
2. **Transaction B (Evidence Finalization)**: Ingests in-memory collector outputs, validates and deduplicates artifacts into `artifacts/<first-two-hex>/<captured_sha256>`, persists `evidence_sets` and `review_artifacts(evidence_set_id)` (Schema v9) with `evidence_committed_at_epoch_ms`, releases verification lease, and transitions `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
3. **Transaction C (ReviewBundle Compilation)**: Synthesizes RFC 8785 JCS canonical JSON, persists `review_bundles(evidence_set_id)` (Schema v9) with `bundle_committed_at_epoch_ms` and `compilation_latency_ms`, inserts proposed audit event `REVIEW_BUNDLE_GENERATED`, and transitions `tasks.state`: `EVIDENCE_READY -> REVIEWING`.

#### 2. Atomic Lease Admission & Reclaim
- `BEGIN IMMEDIATE` admission query validates:
  `tasks.state = 'REPORT_READY'`, attempt is `tasks.current_attempt`, lineage is intact, worktree binding exists and is revalidated, `worker_claims` exists, and `evidence_sets` does not exist for the attempt.
- If task is already `EVIDENCE_READY` or `REVIEWING`, supervisor reads committed authority and does not re-run tests.
- Reclaiming an expired lease terminates and joins the previous Job Object.
- Fencing token isolation applies to sandbox directory (`sandboxes/<attempt_id>/<fencing_token>/`) and staging path (`.staging/<attempt_id>-<fencing_token>-<uuid>.tmp`).
- Transaction B verifies fencing token, `ACTIVE` state, unexpired deadline, and current attempt immediately prior to commit.

#### 3. ReviewBundle Uniqueness and Deterministic Replay
`review_bundles` enforces `UNIQUE(attempt_id)`. Replay with identical canonical bundle hash is idempotent. Replay with conflicting hash fails closed. Crash between Transaction B and C preserves persisted `evidence_sets` and does not re-run tests; single-authority CAS ensures exactly one compilation wins.

#### 4. Schema v9 DDL: `task_verification_leases` (Owned by Subtask P04D)

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
```

---

### Decision 6: Durable Content-Addressed Artifact Store, Review Schema & Proposed Audit Events (P04-ARCH-R8-001, R8-005)

#### 1. Inverted Relational Hierarchy: Evidence Sets Precede Bundles
Transaction B persists durable `evidence_sets` and `review_artifacts` referencing `evidence_set_id`. Transaction C subsequently persists `review_bundles` referencing the same `evidence_set_id`.

#### 2. Deterministic Canonical Paths & Anti-Traversal Guards
Artifacts are stored at deterministically derived content-addressed paths without file extensions:
`artifacts/<first-two-hex>/<captured_sha256>`
CHECK constraints strictly reject path traversal (`..`), backslashes, colons, extra directory segments, and path/hash mismatches.

#### 3. Atomic Write-Through Rename without Overwrite
- Staged writes: Artifact is written to `.staging/<attempt_id>-<fencing_token>-<uuid>.tmp` and flushed via `FlushFileBuffers`.
- Non-overwriting rename: Moved via `MoveFileExW` without file replacement flags.
- Deduplication: If destination already exists, the destination file is reopened by handle, its size and SHA-256 hash verified. If identical, the staging file is deleted (deduplicated). If size or hash differs, execution fails closed immediately.

#### 4. Proposed Audit Events & Two-Phase Failure Semantics
Proposed audit events with status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C upon successful ReviewBundle synthesis.
2. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed diagnostic transaction if Transaction C fails (clock regression, latency > 3000 ms, JCS schema violation, bundle hash conflict) without altering TaskState (`EVIDENCE_READY` preserved).

#### 5. Schema v9 DDL: `evidence_sets`, `review_artifacts`, and `review_bundles` (Owned by Subtask P04D)

```sql
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

## 5. Consequences

### Positive
- Strict, kernel-enforced process and network security boundary for all verification commands via Windows AppContainer.
- Correct process I/O handle inheritance eliminates child pipe deadlock risks.
- Atomic Job Object association at creation eliminates race windows.
- Inverted relational hierarchy matches transactional creation order, eliminating forward foreign key dependencies.
- Strict content-addressed artifact path derivation eliminates ambiguity and path traversal vulnerabilities.
- Durable SQLite epoch timestamp constraints guarantee NFR-008 latency enforcement at the persistence layer.
- Exact hexadecimal string formatting and lineage triggers eliminate cross-pairing and TOCTOU vulnerabilities.
- Deterministic crash recovery and idempotent replay across three well-defined transaction boundaries.

### Negative / Trade-offs
- Verification subprocesses execute against an immutable snapshot, requiring snapshot disk staging and cleanup in the sandbox directory.
- AppContainer configuration on Windows requires specialized Win32 API invocations.
- Deferral of physical GC means orphan files from uncommitted crashes must be pruned by future offline maintenance tools.

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
