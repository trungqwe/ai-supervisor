# DRAFT ADR-018: Evidence Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

- **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`
- **Revision:** 6 (Remediation of External Re-Audit 004)
- **Date:** 2026-09-26
- **Authors:** AI Engineering Supervisor Control Plane Team
- **Deciders:** External Supervisor, System Architect
- **Consulted:** [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md), [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md), [`docs/22_MODULE_PROVENANCE.md`](../22_MODULE_PROVENANCE.md)
- **Informed:** [`docs/04_ARCHITECTURE.md`](../04_ARCHITECTURE.md), [`docs/05_DOMAIN_MODEL.md`](../05_DOMAIN_MODEL.md), [`docs/07_SECURITY_MODEL.md`](../07_SECURITY_MODEL.md), [`docs/08_TASK_CONTRACT.md`](../08_TASK_CONTRACT.md), [`docs/09_WORKER_REPORT.md`](../09_WORKER_REPORT.md), [`docs/10_REVIEW_BUNDLE.md`](../10_REVIEW_BUNDLE.md)
- **Governing Proposal:** [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 6)
- **Governing Proof Plan:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 4)
- **Draft Proof Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`BLOCKED_NOT_RELEASEABLE`, Model A Draft Lineage, status invariant: `NOT_RELEASED`)
- **Audit References:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_004.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_004.md) (`REVISION_5_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_003.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_003.md) (`REVISION_4_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_003.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_003.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md) (`REVISION_3_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md) (`SUPERSEDED_BY_ERRATUM_003`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (`REVISION_2_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md) (`REVISION_1_REQUIRED`)
- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_5`

---

## 1. Context and Problem Statement

Phase P04 implements the Evidence & Review Engine, translating untrusted worker reports and workspace changes into authoritative, inspectable `ReviewBundle` artifacts for supervisor evaluation.

Following External Supervisor Re-Audit 004, this document establishes architectural boundaries resolving critical seams:
1. **Worktree Authority & Binding (P04-ARCH-R5-001, R5-002)**: Disposable runtime AO worker-session testing cannot execute because pinned AO contains no user-selectable inert harness. Track 1 static source inspection is retained as design evidence, while physical worktree binding transitions into fail-closed **Stage B validation** on authorized sessions during normal operation, backed by the authoritative `attempt_workspace_bindings` schema (Schema v6).
2. **Atomic Lease Fencing (P04-ARCH-R5-003)**: `task_verification_leases` (Schema v9) prevents simultaneous and stale evidence collection via monotonically increasing `fencing_token`, explicit owner states, `expires_at_epoch_ms > now_ms` validity, atomic `BEGIN IMMEDIATE` acquisition, and single final CAS transaction.
3. **Artifact Store Scope Reduction (P04-ARCH-R5-004)**: Initial P04 implementation is strictly append-only; background physical garbage collection is DEFERRED. `review_artifacts` enforces `ON DELETE RESTRICT`, immutable append-only triggers, and strict CHECK constraints.
4. **Requirement Tracing & NFR-008 Canonical SLA (P04-ARCH-R5-005)**: Canonical `NFR-008` (ReviewBundle generation within 3 seconds on repos up to 10,000 files) is cited without unauthorized drift; compilation benchmarks are strictly separated from external verification commands.

---

## 2. Decision Drivers

- **Zero-Trust Worker Boundary**: Worker claims and session reports are hypotheses requiring independent verification.
- **Fail-Closed Safety**: Incomplete bindings, lease expirations, or missing artifacts abort evidence generation without mutating state.
- **Auditable Lineage**: Every artifact and evidence row links directly to an immutable `attempt_id` and `contract_id`.
- **Concurrency & Crash Safety**: Single final atomic SQLite CAS transaction guarantees all-or-nothing evidence persistence.
- **Simplicity & Predictability**: Physical GC machinery is deferred, avoiding complex quarantine locking races in initial delivery.

---

## 3. Considered Options

1. **Option 1: In-Flight Intermediate SQLite Writes**: Persisting evidence incrementally across subtasks P04B, P04C, and P04D. *Rejected*: Complex rollback and crash-recovery state machines.
2. **Option 2: Disposable AO Runtime Proof as Release Gate**: Attempting to run disposable AO sessions to prove path binding. *Rejected (P04-ARCH-R5-001)*: Pinned AO has no inert harness; running live agent CLIs generates unauthorized LLM tokens; empty harness payloads fail upstream validation.
3. **Option 3: Selected Architecture (Model 1 + Stage B Binding Validation)**:
   - Pure in-memory collectors for P04B and P04C.
   - Authoritative `attempt_workspace_bindings` table (Schema v6) with Stage B fail-closed validation on authorized sessions.
   - Atomic `BEGIN IMMEDIATE` lease fencing in `task_verification_leases` (Schema v9).
   - Append-only content-addressed artifact store with physical GC deferred.
   - Single final SQLite CAS transaction updating `tasks.state` to `EVIDENCE_READY`.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority and Path Binding (P04-ARCH-R5-001, R5-002)
1. **Authoritative Upstream Reference**: Pinned AO commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` in `Untrivial-ai/agent-orchestrator` ([`workspace.go`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go)).
2. **Disposable Runtime Proof Deferred**: Disposable AO worker-session runtime testing is **NOT** a pre-contract release gate due to the lack of an upstream inert harness (`DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN`). Static source inspection is retained as design evidence.
3. **Authoritative Schema: `attempt_workspace_bindings` (Schema v6)**:
   ```sql
   CREATE TABLE attempt_workspace_bindings (
       attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
       task_id TEXT NOT NULL,
       contract_id TEXT NOT NULL,
       session_id TEXT NOT NULL,
       terminal_generation INTEGER NOT NULL,
       canonical_worktree_path TEXT NOT NULL,
       managed_root_final_path TEXT NOT NULL,
       volume_serial_number TEXT NOT NULL,
       file_id TEXT NOT NULL,
       pinned_ao_commit TEXT NOT NULL,
       bound_at TEXT NOT NULL,
       FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE RESTRICT
   );

   CREATE TRIGGER attempt_workspace_bindings_no_update
   BEFORE UPDATE ON attempt_workspace_bindings
   BEGIN
       SELECT RAISE(FAIL, 'attempt_workspace_bindings is immutable and cannot be updated');
   END;

   CREATE TRIGGER attempt_workspace_bindings_no_delete
   BEFORE DELETE ON attempt_workspace_bindings
   BEGIN
       SELECT RAISE(FAIL, 'attempt_workspace_bindings is immutable and cannot be deleted');
   END;
   ```
4. **Binding Invariants**:
   - `worker_sessions.worktree_path` is strictly an ephemeral session observation/cache.
   - `attempt_workspace_bindings` is the sole authoritative audit evidence for physical worktree authority.
   - Binding is registered in a single atomic transaction during task dispatch intake.
   - P04 verification lease acquisition fails closed if a matching, valid binding is missing.
   - Physical worktree binding is enforced via fail-closed **Stage B validation** during normal authorized operations.

### Decision 2: Hardened In-Memory Git Collector
1. Subtask P04B executes Git CLI with `-c core.hooksPath=/dev/null` and sanitized environment.
2. Evaluates `DESCENDANT_OR_EQUAL_POLICY` and `MERGE_COMMITS_FORBIDDEN`.
3. Operates entirely in memory; performs **zero intermediate SQLite writes**.

### Decision 3: Component-Boundary Containment & Desktop Isolation
1. Containment checks use OS handles, volume serial numbers, and component-boundary traversal (`filepath.Rel`). String prefix matching is strictly forbidden.
2. Subprocesses execute in isolated Windows Window Stations and Desktops (`CreateWindowStationW` + `CreateDesktopW`).
3. External attempt sandbox root `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\` isolates toolchain writes.

### Decision 4: Verification Authority via VerificationPolicyCatalog
1. Verification commands execute strictly via host-governed profiles in `VerificationPolicyCatalog`.
2. Worker contracts cannot supply arbitrary shell binaries or command strings.

### Decision 5: Model 1 Pipeline Atomicity & Durable Monotonic Lease Fencing (P04-ARCH-R5-003)
1. **Table Schema: `task_verification_leases` (Schema v9)**:
   ```sql
   CREATE TABLE task_verification_leases (
       task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE RESTRICT,
       fencing_token INTEGER NOT NULL,
       state TEXT NOT NULL CHECK(state IN ('ACTIVE', 'RELEASED')),
       worker_id TEXT NOT NULL,
       attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
       contract_id TEXT NOT NULL,
       acquired_at_epoch_ms INTEGER NOT NULL,
       expires_at_epoch_ms INTEGER NOT NULL,
       released_at_epoch_ms INTEGER
   );
   ```
2. **State & Expiry Invariants**:
   - `state = 'ACTIVE'` requires non-null worker, attempt, contract, acquired, and expiry timestamps.
   - `state = 'RELEASED'` preserves owner info for auditability; no active lease may have a null owner.
   - Validity: `expires_at_epoch_ms > now_ms`. Equality (`==`) is considered expired.
3. **Atomic Acquisition (`BEGIN IMMEDIATE`)**:
   Acquires lease verifying `tasks.state = 'REPORT_READY'`, matching `current_attempt`, matching `task_attempts`, matching `attempt_workspace_bindings`, and prior lease `RELEASED` or expired.
   Reclaim increments `fencing_token`. Rows are **never deleted**.
4. **Final Single Atomic CAS Transaction**:
   Releases lease, verifies binding, persists evidence rows, and updates `tasks.state` to `EVIDENCE_READY` asserting `rows_affected == 1`. On failure, the transaction rolls back completely.

### Decision 6: Durable Artifact Store, Immutability & Scope Reduction (P04-ARCH-R5-004)
1. **Append-Only Scope Reduction**: P04 initial delivery is strictly append-only with **zero physical garbage collection**. Background file deletion is DEFERRED to a future proposal/ADR.
2. **Crash Consistency Boundary**: Crashes prior to SQLite commit may leave orphan physical files in `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`, but produce zero authoritative review rows. SQLite WAL is the sole source of truth.
3. **Table Schema: `review_artifacts` (Schema v9)**:
   ```sql
   CREATE TABLE review_artifacts (
       artifact_id TEXT PRIMARY KEY,
       bundle_id TEXT NOT NULL REFERENCES review_bundles(bundle_id) ON DELETE RESTRICT,
       attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
       artifact_type TEXT NOT NULL,
       captured_sha256 TEXT NOT NULL CHECK(LENGTH(captured_sha256) = 64),
       full_stream_sha256 TEXT NOT NULL CHECK(LENGTH(full_stream_sha256) = 64),
       original_bytes INTEGER NOT NULL CHECK(original_bytes >= 0),
       captured_bytes INTEGER NOT NULL CHECK(captured_bytes >= 0 AND captured_bytes <= original_bytes),
       is_truncated INTEGER NOT NULL CHECK(is_truncated IN (0, 1) AND ((is_truncated = 1 AND captured_bytes < original_bytes) OR (is_truncated = 0 AND captured_bytes = original_bytes))),
       created_at TEXT NOT NULL
   );

   CREATE TRIGGER review_artifacts_no_update
   BEFORE UPDATE ON review_artifacts
   BEGIN
       SELECT RAISE(FAIL, 'review_artifacts is immutable and cannot be updated');
   END;

   CREATE TRIGGER review_artifacts_no_delete
   BEFORE DELETE ON review_artifacts
   BEGIN
       SELECT RAISE(FAIL, 'review_artifacts is immutable and cannot be deleted');
   END;
   ```
4. **Storage Protocol**: `.staging\<attempt_id>-<uuid>.tmp` -> `fsync` file -> `fsync` parent dir -> atomic rename (`MoveFileExW` with `MOVEFILE_REPLACE_EXISTING`) to `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>` -> final SQLite CAS metadata commit.
5. **Canonical Bundle Hashing**: ReviewBundle hashes formalized under **RFC 8785 (JCS)**.

---

## 5. Consequences

### Positive
- **Eliminates Test-Seam Deadlock**: Avoids blocking P04 on a non-existent upstream inert harness by relying on static source proof and fail-closed Stage B validation.
- **Robust Lineage**: `attempt_workspace_bindings` binds physical storage identity (VolumeSerialNumber + FileId) to attempts immutably.
- **Zero Partial State on Crash**: Model 1 pipeline atomicity ensures that failed attempts leave zero partial SQLite records.
- **No Deletion Races**: Deferring physical GC eliminates complex quarantine locking races from initial implementation.

### Negative / Trade-offs
- **Disk Storage Overhead**: Without physical GC, orphaned staging files and abandoned artifacts accumulate until a future GC proposal is implemented.
- **Deferred Live Harness Proof**: Live AO integration remains unverified on automated harnesses until upstream inert seams or mock adapters are formally developed.

---

## 6. Migration and Schema Ownership

- **Migration v6 (Subtask P04A / Intake)**:
  * Table `attempt_workspace_bindings` (with immutable triggers)
  * Table `claims`
- **Migration v7 (Subtask P04B)**:
  * Table `evidence`
  * Table `policy_findings`
- **Migration v8 (Subtask P04C)**:
  * Table `actual_test_results`
- **Migration v9 (Subtask P04D)**:
  * Table `review_artifacts` (with immutable triggers and CHECK constraints)
  * Table `review_bundles`
  * Table `task_verification_leases`
