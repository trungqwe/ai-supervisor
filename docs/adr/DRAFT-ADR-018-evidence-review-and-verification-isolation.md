# DRAFT ADR-018: Evidence Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

> **Status**: `DRAFT_PENDING_EXTERNAL_APPROVAL`
> **Revision**: 8
> **Date**: 2026-09-26
> **Audited Baseline**: `e7c57965ca8d85c0efa0b4f5b5dd409ce7387234`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_7`
> **Decision Owners**: AI Engineering Supervisor Team
> **Supersedes**: `DRAFT-ADR-018` Revision 7
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R7-001` through `P04-ARCH-R7-006` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_006.md`).
> **Requirement & Governance Note**: Cannot be accepted or marked ready until `PROPOSAL-P04-002-review-bundle-latency-semantics.md` is approved by External Supervisor. `docs/02_REQUIREMENTS.md` remains unmodified.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine** to provide independent, tamper-proof verification of AI worker outputs. Under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`), the Supervisor must:
1. Verify physical worktree authority without relying on unverified worker claims;
2. Collect tamper-proof Git diffs without risk of hook execution, config poisoning, or directory mutation;
3. Execute independent test suites and linters in isolated Windows execution environments with strict resource, filesystem, and network isolation;
4. Manage content-addressed review artifacts with rock-solid durability and crash consistency;
5. Compile an attempt-scoped `ReviewBundle` adhering to `docs/10_REVIEW_BUNDLE.md` within governed latency budgets.

External Re-Audit 006 required remediation of six critical architectural findings:
- `P04-ARCH-R7-001`: Registration boundary collision and invalid foreign key coupling to mutable worker session lanes;
- `P04-ARCH-R7-002`: Flawed forward foreign key ordering between artifacts and bundles, missing durable evidence schema, and unsafe file replacement;
- `P04-ARCH-R7-003`: Ambiguous process security boundary, incomplete Win32 specification, and ungrounded snapshot protocol;
- `P04-ARCH-R7-004`: Incomplete Git execution isolation and requirement traceability misalignment;
- `P04-ARCH-R7-005`: Loose lease acquire/reclaim mechanics, unthreaded fencing tokens, and missing ReviewBundle idempotency;
- `P04-ARCH-R7-006`: Unapproved verification budget tokens, missing derivation rules, and typography defects.

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
3. **Option 3: Hardened In-Memory Engine with AppContainer Security Boundary, Durable Evidence Sets, and Three Transaction Boundaries (Accepted)**: Independent physical OS handle validation, hardened in-memory Git collection, Windows AppContainer security boundary with DACL-protected immutable source snapshots, append-only content-addressed artifact storage, and three distinct durable SQLite transaction boundaries. Dual-path isolation and non-AppContainer token alternatives are rejected for v1.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority, Immutable Attempt Workspace Bindings & Dispatch Seam (P04-ARCH-R7-001)

#### 1. Separation of Registration and Intake Transactions
Registration and verification intake are decoupled into two distinct transactional boundaries:
1. **Dispatch Binding Registration Transaction**: Executed immediately after Agent Orchestrator successfully materializes the worktree, integrated directly into the P03 dispatch seam (`PrepareBoundDispatch` / `DISPATCH_BOUND`) and committed prior to `RecordSendRequested`. This binds the physical worktree identity (`volume_serial_hex`, `file_id_hex`, canonical path) to the attempt before worker execution commences.
2. **Transaction A (Report Intake)**: Executed when a worker signals completion. Transaction A strictly reads and verifies that a valid physical binding already exists for the attempt. It persists structured claims into `worker_claims` and transitions `tasks.state`: `RUNNING -> REPORT_READY`. Transaction A is strictly prohibited from creating or inserting workspace bindings.

#### 2. Immutable History Decoupled from Mutable Session Lanes
`attempt_workspace_bindings` represents permanent, immutable historical evidence of dispatch. Foreign key coupling to `worker_sessions(session_id)` is removed because `worker_sessions` represents the mutable, single-lane state of a Pair and can be rotated, reassigned, or purged across attempts. Historical attempt bindings must never block session rotation or Pair maintenance. Integrity is enforced via triggers cross-referencing `task_attempts` and `dispatch_operations`.

#### 3. Strict Hexadecimal Identity Specification
Volume and file identifiers are normalized to lowercase hexadecimal strings:
- `volume_serial_hex`: Exactly 16 lowercase hex characters (`CHECK (LENGTH(volume_serial_hex) = 16 AND NOT (volume_serial_hex GLOB '*[^0-9a-f]*'))`), representing the 64-bit volume serial number zero-padded.
- `file_id_hex`: Exactly 32 lowercase hex characters (`CHECK (LENGTH(file_id_hex) = 32 AND NOT (file_id_hex GLOB '*[^0-9a-f]*'))`), representing the 128-bit NTFS `FILE_ID_128` encoded in byte order as returned by `GetFileInformationByHandleEx(FileIdInfo)`.
- CHECK constraints strictly reject short strings, uppercase characters, and non-hexadecimal characters.

#### 4. Schema v6 DDL: `attempt_workspace_bindings` and `worker_claims` (Owned by Subtask P04A)

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

-- Schema v6: worker_claims
CREATE TABLE worker_claims (
    claim_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    claim_type TEXT NOT NULL CHECK (claim_type IN ('CHANGED_FILES', 'TEST_RESULTS', 'IMPLEMENTATION_SUMMARY', 'UNVERIFIED_ASSUMPTION')),
    claim_payload_json TEXT NOT NULL CHECK (LENGTH(claim_payload_json) > 0),
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

#### 5. OS Handle Lifecycle & Scope Clarification
- Handle Management: The root directory handle is opened with sharing flags `FILE_SHARE_READ | FILE_SHARE_WRITE` (strictly omitting `FILE_SHARE_DELETE`) and held open across evidence collection and CAS validation. Revalidation confirms identity immediately before committing durable evidence.
- Bounded Subtask Scope: The future scope of Subtask P04A encompasses the necessary dispatch seams in `internal/dispatch` and `internal/store` to execute Dispatch Binding Registration alongside Transaction A. No production code is authorized in this governance stage.

---

### Decision 2: Hardened In-Memory Git Collector & Traceability (P04-ARCH-R7-004)

1. **Host-Pinned Binary Resolution**: Git executable resolved strictly from host configuration (`SUPERVISOR_GIT_BIN`), version and SHA-256 pinned per `NFR-007`. Arbitrary `PATH` lookups are prohibited.
2. **Windows Configuration Isolation**: Windows cannot use Unix null device syntax for directories. Supervisor creates `<SUPERVISOR_STATE_ROOT>/trusted_empty_git/` with an empty config file and empty hooks directory.
3. **Mandatory CLI Flags & Invocations**:
   - Every Git invocation must pass `-c core.hooksPath=<trusted_empty_hooks>` and `-c core.fsmonitor=false`.
   - `--no-pager` is strictly passed as a CLI argument, not an environment variable.
4. **Environment Sanitization**:
   - Unset `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_ALTERNATE_OBJECT_DIRECTORIES`.
   - Sanitize: `GIT_CONFIG_COUNT`, `GIT_CONFIG_KEY_*`, `GIT_CONFIG_VALUE_*`, `GIT_EXTERNAL_DIFF`, `GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_SSH`, `GIT_SSH_COMMAND`, `GIT_PROTOCOL_FROM_USER`, `GIT_CEILING_DIRECTORIES`.
   - Explicitly configure: `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_SYSTEM`, `GIT_CONFIG_GLOBAL` pointing to empty config, `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`, `GIT_PAGER=cat`.
5. **Command Allowlist & Flags**: Only allowlisted non-mutating commands (`rev-parse`, `merge-base`, `log`, `diff`). Always pass `--no-ext-diff` and `--no-textconv`. Use literal pathspecs for file inputs.
6. **Execution Controls**: 10s execution timeout; 10 MB streaming byte cap terminating the entire process tree immediately upon exceedance; pre/post HEAD, index, and worktree identity verification.
7. **Precise Index Hash**: Index hash is strictly defined as the SHA-256 of `.git/index` under verified repository handle identity.
8. **Requirement Traceability**:
   - Independent Git and test evidence collection -> **FR-008**
   - Scope comparison and path validation -> **FR-009**, **SEC-005**
   - Append-only audit integrity -> **FR-013**, **NFR-004**
   - Upstream adapter separation -> **NFR-005**
   - Pinned upstream tool dependencies -> **NFR-007**
   - Process containment and execution isolation -> **SEC-001**, **SEC-003**, **OPS-003**

---

### Decision 3: Windows Verification Isolation Security Boundary (P04-ARCH-R7-003)

#### 1. Exclusive Selection of Windows AppContainer for v1
Phase P04 v1 commits exclusively to **Windows AppContainer** as the verification execution security boundary. Dual-path implementations and non-AppContainer token alternatives are eliminated.

#### 2. Complete Win32 API Implementation Sequence
The verification sandbox is established using the following Win32 sequence:
1. `CreateAppContainerProfile` / `DeriveAppContainerSidFromAppContainerName` creates or derives the AppContainer SID.
2. File security grants: Grant read/execute DACL on snapshot root and tools; grant read/write/delete DACL on sandbox directory to the AppContainer SID.
3. Attribute list initialization: `InitializeProcThreadAttributeList` with count 1.
4. Security capabilities: Zero network and system capabilities configured in `SECURITY_CAPABILITIES` (`CapabilityCount = 0`, `Capabilities = NULL`).
5. Process attribute update: `UpdateProcThreadAttribute` with `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES`.
6. Process creation: `STARTUPINFOEXW` passed to `CreateProcessW` with `EXTENDED_STARTUPINFO_PRESENT`, `CREATE_SUSPENDED`, `CREATE_NO_WINDOW`, `CREATE_BREAKAWAY_FROM_JOB`.
7. Job Object containment: Assigned to dedicated Job Object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, breakaway disabled, and 2 GB memory cap.
8. Handle inheritance: Strictly disabled (`bInheritHandles = FALSE`).
9. Fail-Closed Mandate: Any failure to establish AppContainer, DACLs, or Job Object aborts verification immediately. Falling back to daemon token execution is strictly prohibited.

#### 3. Pinned Immutable Snapshot Protocol
Subprocesses execute strictly against an immutable source snapshot:
- Source: Pinned verified HEAD commit.
- Export mechanism: Streamed via pinned Git `ls-tree -rz` and `cat-file --batch` into `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/`.
- Validation: Every path component is validated; directory traversal (`..`), junctions, reparse points, symlinks, and gitlinks are rejected in v1.
- Integrity: An immutable manifest with SHA-256 manifest hash is generated and verified before test invocation.
- Isolation: Test subprocesses never read the active worktree directly, and mutations are never copied back from the snapshot to the worktree.
- Handle Invariant: The worktree root directory handle is opened without `FILE_SHARE_DELETE` and held across collection and CAS validation. Testable handle validation invariants replace unverifiable safety claims.

---

### Decision 4: Verification Authority & Latency Semantics via PROPOSAL-P04-002 (P04-ARCH-R7-006)

1. Verification commands execute strictly via approved profiles in `VerificationPolicyCatalog` with host-governed absolute paths.
2. NFR-008 performance SLA reconciliation is governed by `PROPOSAL-P04-002-review-bundle-latency-semantics.md` (`PENDING_EXTERNAL_REVIEW`), establishing a two-interval model:
   - **Interval 1 (Evidence Acquisition Window)**: Derived strictly from TaskContract `verification_requests[].timeout_seconds` validated by Stage B, or `MaxTimeoutSeconds` from catalog profiles (aggregated as sum for sequential requests, max for concurrent requests). All arbitrary verification budget tokens and ungrounded 60s defaults are eliminated.
   - **Interval 2 (Bundle Compilation Window)**: Persisted epoch timestamps enforce:
     `0 <= T1 - T0 <= 3.0 seconds`
     where T0 is `evidence_committed_at` (Transaction B commit) and T1 is Transaction C commit timestamp. In-process timing utilizes monotonic duration. Clock regression fails closed and records an audit event.
3. Canonical audit event type: `AUDIT_EVENT_BUNDLE_GENERATED` from canonical audit event catalog is utilized.
4. Clean plain text formatting: Typography is strictly plain text with zero TABs or control characters. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until formal approval.

---

### Decision 5: Three Durable Pipeline Transactions & Lease Fencing (P04-ARCH-R7-005)

#### 1. Three Durable Transaction Boundaries
The verification pipeline is segmented into three distinct SQLite WAL transactions:
1. **Transaction A (Report Intake)**: Validates worker report, persists `worker_claims` (Schema v6), and transitions `tasks.state`: `RUNNING -> REPORT_READY`. (Does not insert bindings).
2. **Transaction B (Evidence Finalization)**: Ingests in-memory collector outputs, validates and moves artifacts into canonical layout, persists `evidence_sets` and `review_artifacts(evidence_set_id)` (Schema v9), releases verification lease, and transitions `tasks.state`: `REPORT_READY -> EVIDENCE_READY`. Evidence persistence is verified prior to state transition.
3. **Transaction C (ReviewBundle Compilation)**: Synthesizes RFC 8785 JCS canonical JSON, persists `review_bundles(evidence_set_id)` (Schema v9), inserts canonical `audit_events`, and transitions `tasks.state`: `EVIDENCE_READY -> REVIEWING`.

#### 2. Atomic Lease Acquisition and Reclaim
Leases are acquired and reclaimed atomically using `BEGIN IMMEDIATE`:
```sql
-- Atomic acquire
INSERT INTO task_verification_leases (
    task_id, fencing_token, state, worker_id, attempt_id, contract_id,
    acquired_at_epoch_ms, expires_at_epoch_ms, released_at_epoch_ms
) VALUES (?, 1, 'ACTIVE', ?, ?, ?, ?, ?, NULL);

-- Atomic reclaim on expired or released lease
UPDATE task_verification_leases
SET fencing_token = fencing_token + 1,
    state = 'ACTIVE',
    worker_id = ?,
    attempt_id = ?,
    contract_id = ?,
    acquired_at_epoch_ms = ?,
    expires_at_epoch_ms = ?,
    released_at_epoch_ms = NULL
WHERE task_id = ?
  AND fencing_token = ?
  AND (state = 'RELEASED' OR expires_at_epoch_ms <= ?);
```
- Assert `RowsAffected == 1`. If 0 rows affected, acquisition fails closed.
- Expiration is determined strictly from durable SQLite row timestamps (`expires_at_epoch_ms <= now_ms`), never from in-memory state.
- Collectors carry the active `fencing_token`. Transaction B verifies lease is ACTIVE, unexpired, and token matches before committing.

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

### Decision 6: Durable Content-Addressed Artifact Store & Review Schema (P04-ARCH-R7-002)

#### 1. Inverted Relational Hierarchy: Evidence Sets Precede Bundles
Transaction B persists durable `evidence_sets` and `review_artifacts` referencing `evidence_set_id`. Transaction C subsequently persists `review_bundles` referencing the same `evidence_set_id`.

#### 2. Deterministic Canonical Paths & Anti-Traversal Guards
Artifacts are stored at deterministically derived content-addressed paths:
`artifacts/<h0h1>/<full_sha256>.<ext>`
CHECK constraints strictly reject path traversal (`..`), backslashes, colons, and non-canonical relative path structures.

#### 3. Atomic Write-Through Rename without Overwrite
- Staged writes: Artifact is written to `.staging/<attempt_id>-<uuid>.tmp` and flushed via `FlushFileBuffers`.
- Non-overwriting rename: Moved via `MoveFileExW` without `MOVEFILE_REPLACE_EXISTING`.
- Deduplication: If destination already exists, the destination file is reopened by handle, its size and SHA-256 hash verified. If identical, the staging file is deleted (deduplicated). If size or hash differs, execution fails closed immediately.

#### 4. Canonical Audit Logging
Audit logging exclusively utilizes the existing, canonical `audit_events` table. Ad-hoc audit table references are eradicated.

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
    artifact_type TEXT NOT NULL CHECK (LENGTH(artifact_type) > 0),
    media_type TEXT NOT NULL CHECK (LENGTH(media_type) > 0),
    encoding TEXT NOT NULL CHECK (LENGTH(encoding) > 0),
    canonical_relative_path TEXT NOT NULL CHECK (
        canonical_relative_path GLOB 'artifacts/[0-9a-f][0-9a-f]/[0-9a-f]*' AND
        canonical_relative_path NOT GLOB '*..*' AND
        canonical_relative_path NOT GLOB '*//*'
    ),
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
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
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

-- Schema v9: review_bundles
CREATE TABLE review_bundles (
    bundle_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL UNIQUE REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    bundle_payload_json TEXT NOT NULL CHECK (LENGTH(bundle_payload_json) > 0),
    bundle_hash TEXT NOT NULL CHECK (LENGTH(bundle_hash) = 64 AND NOT (bundle_hash GLOB '*[^0-9a-f]*')),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
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
- Inverted relational hierarchy matches transactional creation order, eliminating invalid forward foreign key dependencies.
- Dispatch binding registration is cleanly decoupled from intake verification, preserving permanent immutable history while allowing normal worker session rotations.
- Exact hexadecimal string formatting and lineage triggers eliminate cross-pairing and TOCTOU vulnerabilities.
- Deterministic crash recovery and idempotent replay across three well-defined transaction boundaries.
- Complete fidelity to upstream AO authority without unexecutable proof deadlocks.
- Clear governance path for NFR-008 performance SLA reconciliation without arbitrary tokens.

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
