# DRAFT ADR-018: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

- **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`
- **Revision:** 5 (Remediation of External Re-Audit 003)
- **Deciders:** External Supervisor, Supervisor Core Architecture Team
- **Date:** 2026-09-26
- **Technical Precedence:** Level 2 (Architecture Decision Record)
- **Authority:** Approved pursuant to [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md) and [`AGENTS.md`](../../AGENTS.md)
- **Supercedes:** None
- **Related ADRs:** [`ADR-006`](ADR-006-evidence-first-review.md), [`ADR-011`](ADR-011-worker-report-handoff-and-agy-invocation-boundary.md), [`ADR-012`](ADR-012-task-contract-revision-and-attempt-binding.md), [`ADR-013`](ADR-013-trusted-verification-command-spec.md), [`ADR-016`](ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md), [`ADR-017`](ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md)
- **Associated Proposal:** [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 5)
- **Associated Proof Plan:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 3)
- **Associated Draft Proof Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`, Model A Draft Lineage)

---

## 1. Context and Problem Statement

Phase P04 implements the Evidence & Review Engine, which processes worker reports, collects independent Git evidence, executes verification commands, evaluates contract compliance via policy rules, and synthesizes immutable review bundles for ChatGPT supervisor review.

During External Supervisor Re-Audit 003, critical architectural refinements were required:
1. **Repository Provenance & Pinned Citations**: Permalinks must point to the authoritative upstream repository `Untrivial-ai/agent-orchestrator` at pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`, with single official line ranges.
2. **Wire Protocol Lifecycle**: Creating restorable sessions requires `POST /api/v1/sessions/{id}/kill`, bounded `GET` polling for `isTerminated=true`, and `POST /api/v1/sessions/{id}/restore` returning HTTP 200 (never ambiguous HTTP 200 or 201).
3. **Component-Boundary Containment**: String prefix matching is vulnerable to sibling-prefix attacks. Handle-based component-boundary verification (`filepath.Rel`, `FILE_ID_INFO`, volume serial) is mandatory.
4. **Durable Monotonic Lease Fencing**: Deleting lease rows on release resets fencing tokens to 1 (ABA vulnerability). Monotonic non-resetting tokens with epoch expiry are required.
5. **Artifact Immutability & Coordinated GC**: `review_artifacts` must use `ON DELETE RESTRICT` and triggers preventing mutation. GC must coordinate via leases and atomic quarantine renames.
6. **Harness & Profile Authority**: The previously suggested inert harness literal was fabricated and is completely removed. Pinned AO has no user-selectable inert harness; `DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN` remains. Worker profile is `antigravity-standard`.

---

## 2. Decision Drivers

- **Zero-Trust Worker Boundary**: Worker reports and claimed worktree paths are untrusted inputs requiring independent verification.
- **Fail-Closed Execution**: Any policy violation, untrusted path traversal, or uncontainable command terminates the attempt into a blocked/failed state.
- **Durable Crash Consistency**: System crashes during evidence generation must never corrupt WAL state or leave inconsistent metadata.
- **Inspectable ReviewBundles**: Evidence must be permanently inspectable without raw filesystem traversal.

---

## 3. Considered Options

- **Option A (In-Tree Direct Execution & Incremental DB Writes)**: Allow verification commands to run directly in audited worktrees and write partial evidence rows. *Rejected*: Causes dirty worktree state, violates atomicity, and creates race conditions.
- **Option B (Model 1 Pipeline Atomicity, External Absolute Sandbox & Content-Addressed Store)**: *Adopted*.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority and Path Binding
1. **Pinned Source Facts**: Inspection of `backend/internal/adapters/workspace/gitworktree/workspace.go` at pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` in `Untrivial-ai/agent-orchestrator` proves:
   - `Options`: lines 64–73 ([`official permalink`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L64-L73))
   - `Workspace.Create`: lines 230–262 ([`official permalink`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L230-L262))
   - `Workspace.Restore`: lines 1045–1112 ([`official permalink`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1045-L1112))
   - `managedPath`: lines 1754–1762; signature `func (w *Workspace) managedPath(cfg ports.WorkspaceConfig) (string, error)` ([`official permalink`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1754-L1762))
   - `restorePath`: lines 1764–1768; signature `func (w *Workspace) restorePath(cfg ports.WorkspaceConfig) (string, error)` ([`official permalink`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1764-L1768))
   - `defaultSessionBranchName`: lines 1783–1785; signature `func defaultSessionBranchName(id domain.SessionID) string` ([`official permalink`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1783-L1785))
2. **Server-Generated Session IDs**:
   - `POST /api/v1/sessions` assigns server-generated session IDs (`resp.Session.ID`). Caller cannot inject custom session IDs.
   - Expected path (`<managedRoot>/<projectID>/<sessionID>`) and branch (`ao/<sessionID>`) are computed dynamically at runtime from spawn response evidence.
3. **Wire Lifecycle Contract**:
   - Termination uses `POST /api/v1/sessions/{id}/kill` (HTTP 200).
   - Polling `GET /api/v1/sessions/{id}` confirms `isTerminated=true`.
   - Restore uses `POST /api/v1/sessions/{id}/restore` returning strictly HTTP 200 (never 201), valid `restoreMode`, and matching IDs.
4. **Empirical Proof Gating**: Local worktree path authority is conditional upon the successful execution and approval of [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 3).

### Decision 2: Hardened In-Memory Git Collector
- Git evidence is collected in-memory without mutating the worktree.
- Enforces `DESCENDANT_OR_EQUAL_POLICY` (`git merge-base --is-ancestor <base_sha> HEAD`) and `MERGE_COMMITS_FORBIDDEN` (`git rev-list --min-parents=2 <base_sha>..HEAD`).
- Injects `GIT_TERMINAL_PROMPT=0` and 30-second timeout.
- Compares claims against `git diff --name-only <base_sha>..HEAD` and `git status --porcelain -uall`.

### Decision 3: Component-Boundary Containment & Desktop Isolation
1. **Handle-Based Component-Boundary Pre-Create Check:**
   - Pre-create: Ensure `SUPERVISOR_STATE_ROOT` exists and is absolute. Open root with OS handle, retrieve final physical path, volume serial, and `FILE_ID_INFO`. For non-existent targets, resolve nearest existing ancestor. Use `filepath.Rel` component-boundary checking (reject `rel == ".."` or starting with `..\`). Reject volume mismatch and unexpected reparse points. Raw string prefix matching is strictly forbidden.
   - Post-create: Open target handle, compare physical ancestry, volume, and final path with state root. Revalidate immediately prior to recursive cleanup.
   - Cleanup: Target must differ from state root/proof parent. Marker must contain random `proof_run_id`, expected root identity, and ownership nonce. Mismatch or TOCTOU fails closed (no deletion).
2. **Window Station & Desktop Isolation:**
   - Dedicated non-interactive Window Station via `CreateWindowStationW` + `CreateDesktopW`.
3. **AppContainer Isolation:**
   - Low integrity SID profile; network capabilities denied; write access confined to attempt sandbox.
4. **Disposable Source Snapshot:**
   - Extracted from verified Git `HEAD` into sandbox for modifying tests; destroyed during cleanup.

### Decision 4: Verification Authority via VerificationPolicyCatalog
- Verification commands are strictly mediated via typed profiles in `VerificationPolicyCatalog`.
- Workers cannot supply raw executable paths, shell strings, endpoints, roots, or harnesses.
- Approved worker profile is `antigravity-standard`.
- `DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN` remains until an inert test mode or upstream test seam is established.
- Stage B policy check is recorded as `UNVERIFIED` until registered in live catalog.

### Decision 5: Model 1 Pipeline Atomicity & Durable Monotonic Lease Fencing
1. **In-Memory Collectors:** Subtasks P04B and P04C perform strictly zero SQLite writes.
2. **Durable Monotonic Lease Fencing:**
   - Pre-command reservation lease in `task_verification_leases` (Schema v9, owned by P04D):
     ```sql
     CREATE TABLE IF NOT EXISTS task_verification_leases (
         task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE RESTRICT,
         attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
         fencing_token INTEGER NOT NULL DEFAULT 0,
         worker_id TEXT,
         state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'RELEASED')),
         acquired_at TEXT,
         expires_at_epoch_ms INTEGER NOT NULL DEFAULT 0
     );
     ```
   - Row is kept by `task_id`. `fencing_token` is monotonically increasing and never resets (no DELETE row on release; prevents ABA token reuse).
   - Expiry uses numeric epoch milliseconds (`expires_at_epoch_ms`).
   - Acquisition pre-conditions: verify `tasks.state = 'REPORT_READY'`, `tasks.current_attempt` matching `task_attempts.attempt_number`, valid lineage, attempt not superseded.
   - Release: CAS updating `state = 'RELEASED'` matching exact `task_id`, `attempt_id`, `worker_id`, and `fencing_token`.
3. **Single Final Atomic SQLite CAS Transaction (P04D):**
   - Validates lease fencing token, active state, unexpired epoch, and updates `tasks.state`:
     ```sql
     UPDATE tasks
     SET state = 'EVIDENCE_READY',
         updated_at = ?
     WHERE task_id = ?
       AND state = 'REPORT_READY'
       AND current_attempt = ?
       AND EXISTS (
           SELECT 1 FROM task_verification_leases l
           WHERE l.task_id = tasks.task_id
             AND l.attempt_id = ?
             AND l.fencing_token = ?
             AND l.worker_id = ?
             AND l.state = 'ACTIVE'
             AND l.expires_at_epoch_ms >= ?
       );
     ```
4. **Fault Matrix:**
   - Crash after acquire -> lease expires -> reclaimed with incremented token.
   - Stale owner resumes after expiry -> final CAS fails due to token mismatch.
   - TaskState or current_attempt changed before command -> acquire or final CAS fails.
   - Token ABA attempt -> prevented by monotonic increment without row deletion.

### Decision 6: Durable Artifact Store, Immutability & Coordinated GC
1. **Canonical Path Key:** Stored files containing captured prefix bytes use `captured_sha256`:
   `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`
2. **Schema & Triggers:** `review_artifacts` uses `attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT`. Triggers prevent `UPDATE` and `DELETE`.
3. **Retrieval Tamper Detection:** Stored bytes are re-hashed and verified against `captured_sha256` before returning content.
4. **Coordinated GC Lease Protocol:**
   - GC acquires exclusive `artifact_store_gc` lease.
   - Unreferenced candidates verified against DB.
   - Atomically renamed to `.quarantine/<captured_sha256>.<timestamp>`.
   - Deletion occurs only after 24h grace period and final reference check.
   - Writer coordinates with GC lease to avoid races between existence check and metadata commit.
5. **Canonical Hashing:** `ReviewBundle` deterministic hashing is formalized using **RFC 8785 (JCS)**.

---

## 5. Consequences

### Positive
- Zero risk of worktree mutation or credential prompt hanging.
- Complete audit trail with durable, content-addressed verification logs.
- Guaranteed atomicity: zero partial evidence rows in SQLite.
- Strong containment: zero cross-worktree or repository pollution.
- Robust lease fencing: elimination of token ABA resets and stale worker commits.

### Negative / Trade-offs
- Additional disk I/O for staging and fsync of large artifacts.
- Requires Windows host administrative privileges for AppContainer and Window Station allocation.

---

## 6. Migration and Schema Ownership

- **Schema v6 (P04A)**: `claims` table.
- **Schema v7 (P04B)**: `evidence`, `policy_findings` tables.
- **Schema v8 (P04C)**: `actual_test_results` table.
- **Schema v9 (P04D)**: `review_artifacts`, `review_bundles`, `task_verification_leases` tables.
