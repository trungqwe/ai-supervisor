# DRAFT ADR-018: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

- **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`
- **Revision:** 4 (Remediation of External Re-Audit 002)
- **Deciders:** External Supervisor, Supervisor Core Architecture Team
- **Date:** 2026-09-26
- **Technical Precedence:** Level 2 (Architecture Decision Record)
- **Authority:** Approved pursuant to [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md) and [`AGENTS.md`](../../AGENTS.md)
- **Supercedes:** None
- **Related ADRs:** [`ADR-006`](ADR-006-evidence-first-review.md), [`ADR-011`](ADR-011-worker-report-handoff-and-agy-invocation-boundary.md), [`ADR-012`](ADR-012-task-contract-revision-and-attempt-binding.md), [`ADR-013`](ADR-013-trusted-verification-command-spec.md), [`ADR-016`](ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md), [`ADR-017`](ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md)
- **Associated Proposal:** [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 4)
- **Associated Proof Plan:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 2)
- **Associated Draft Proof Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`, Revision 2)

---

## 1. Context and Problem Statement

Phase P04 implements the Evidence & Review Engine, which processes worker reports, collects independent Git evidence, executes verification commands, evaluates contract compliance via policy rules, and synthesizes immutable review bundles for ChatGPT supervisor review.

During External Supervisor Audit 001 and Re-Audits 001 and 002, seven critical architectural seams were identified:
1. **Worktree Authority**: Local worktree path authority was unproven across host environments, and symbol citations / signatures were inaccurate.
2. **Relative Roots**: Sandbox and artifact paths used unsafe relative `.supervisor/...` directories.
3. **Verification Authority & Inert Harness**: Workers lacked typed execution boundaries, and AO test mode inertness (zero LLM calls) was unproven.
4. **Restore & Edge Cases**: Restore lifecycle was underspecified and treated expected probe outcomes as infrastructure crashes.
5. **Artifact Store & Crash Consistency**: Artifact addressing used ambiguous hashes, and filesystem rename rollback was incorrectly attributed to SQLite.
6. **TaskState CAS & Lease Schema Mismatch**: Pseudo-SQL targeted non-existent `task_attempts.status` column instead of `tasks.state`, and pre-command verification leases were undefined.
7. **Governance Drift**: Governance documents retained outdated revision references.

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
1. **Pinned Source Facts**: Inspection of `backend/internal/adapters/workspace/gitworktree/workspace.go` at pinned AO commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` proves:
   - `Options`: lines 64–73; `ManagedRoot` at line 68 (raw blob lines 71–80, line 75).
     [`backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80)
   - `Workspace.Create`: lines 230–262; invokes `w.managedPath(cfg)` at line 243 (raw blob lines 246–266, line 257).
     [`backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266)
   - `Workspace.Restore`: lines 1045–1112; invokes `w.restorePath(cfg)` at line 1055 (raw blob lines 1097–1140, line 1105).
     [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140)
   - `managedPath`: lines 1754–1762 (raw blob lines 1837–1846); signature `func (w *Workspace) managedPath(cfg ports.WorkspaceConfig) (string, error)`.
     [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846)
   - `restorePath`: lines 1764–1768 (raw blob lines 1848–1853); signature `func (w *Workspace) restorePath(cfg ports.WorkspaceConfig) (string, error)`.
     [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853)
   - `defaultSessionBranchName`: lines 1783–1785 (raw blob lines 1868–1870); signature `func defaultSessionBranchName(id domain.SessionID) string`.
     [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870)
2. **Server-Generated Session IDs**:
   - `POST /api/v1/sessions` assigns server-generated session IDs (`resp.Session.ID`). Caller cannot inject custom session IDs.
   - Expected path (`<managedRoot>/<projectID>/<sessionID>`) and branch (`ao/<sessionID>`) are computed dynamically at runtime from spawn response evidence.
3. **Empirical Proof Gating**: Local worktree path authority is conditional upon the successful execution and approval of [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 2).

### Decision 2: Hardened In-Memory Git Collector
- Git evidence is collected in-memory without mutating the worktree.
- Enforces `DESCENDANT_OR_EQUAL_POLICY` (`git merge-base --is-ancestor <base_sha> HEAD`) and `MERGE_COMMITS_FORBIDDEN` (`git rev-list --min-parents=2 <base_sha>..HEAD`).
- Injects `GIT_TERMINAL_PROMPT=0` and 30-second timeout.
- Compares claims against `git diff --name-only <base_sha>..HEAD` and `git status --porcelain -uall`.

### Decision 3: Windows AppContainer & Desktop Isolation
1. **Absolute State Root Hierarchy:**
   - `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`
   - All toolchain write targets (`TEMP`, `TMP`, `GOCACHE`, `GOPATH`) redirected into the external sandbox.
2. **Containment & Safe Cleanup:**
   - Pre-operation canonicalization and containment verification ensuring paths are strictly outside all Git worktrees, primary repository, AO live database, and recovery directory.
   - Marker file `.supervisor-owner-marker.json` verified before recursive cleanup.
   - Missing Windows privileges for junctions/AppContainers recorded as `UNVERIFIED_CAPABILITY` or `BLOCKED` (never `PASS`).
3. **Window Station & Desktop Isolation:**
   - Dedicated non-interactive Window Station via `CreateWindowStationW` + `CreateDesktopW`.
4. **Disposable Source Snapshot:**
   - Extracted from verified Git `HEAD` into sandbox for modifying tests; destroyed during cleanup.

### Decision 4: Verification Execution Authority via VerificationPolicyCatalog
- Verification commands are strictly mediated via typed profiles in `VerificationPolicyCatalog`.
- Workers cannot supply raw executable paths, shell strings, endpoints, or roots.
- Pinned AO harness mode must be proven inert (zero LLM calls, zero credentials, zero prompts) before execution (`DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN`).
- Stage B policy check is recorded as `UNVERIFIED` until registered in live catalog.

### Decision 5: Model 1 Pipeline Atomicity & Lease Reservation
1. **In-Memory Collectors:** Subtasks P04B and P04C perform strictly zero SQLite writes.
2. **Pre-Command Reservation Lease:** P04D acquires a lease in `task_verification_leases` (Schema v9) with an incremented `fencing_token`. Losers are denied execution (zero commands executed).
3. **State Invariant:** The task remains in `state = 'REPORT_READY'` on `tasks` throughout execution.
4. **Single Final Atomic SQLite CAS Transaction:** P04D executes a single transaction persisting all evidence, findings, test results, review artifacts, audit events, and updating `tasks.state`:
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
           AND l.expires_at > ?
     );
   ```
5. **Stale Worker Protection:** Resumed crashed workers with expired/reclaimed leases cannot commit evidence.

### Decision 6: Durable Content-Addressed Artifact Store & Crash Consistency
1. **Canonical Path Key:** Stored files containing captured prefix bytes use `captured_sha256`:
   `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`
2. **Dual Hashing:** `full_stream_sha256` is recorded as stream metadata in `review_artifacts`.
3. **Crash Consistency Protocol:** Write staging (`.staging/<uuid>`) -> fsync file + parent directory -> atomic rename -> SQLite CAS metadata insert.
4. **Crash Recovery:** Startup GC removes unreferenced staging/orphan files after a 24-hour grace period following containment verification.
5. **Canonical Hashing:** `ReviewBundle` deterministic hashing is formalized using **RFC 8785 (JCS)**.

---

## 5. Consequences

### Positive
- Zero risk of worktree mutation or credential prompt hanging.
- Complete audit trail with durable, content-addressed verification logs.
- Guaranteed atomicity: zero partial evidence rows in SQLite.
- Strong containment: zero cross-worktree or repository pollution.

### Negative / Trade-offs
- Additional disk I/O for staging and fsync of large artifacts.
- Requires Windows host administrative privileges for AppContainer and Window Station allocation.

---

## 6. Migration and Schema Ownership

- **Schema v6 (P04A)**: `claims` table.
- **Schema v7 (P04B)**: `evidence`, `policy_findings` tables.
- **Schema v8 (P04C)**: `actual_test_results` table.
- **Schema v9 (P04D)**: `review_artifacts`, `review_bundles`, `task_verification_leases` tables.
