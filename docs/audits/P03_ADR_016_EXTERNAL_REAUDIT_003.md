# P03_ADR_016_EXTERNAL_REAUDIT_003.md — Independent External Supervisor Re-Audit of ADR-016 Revision 3

> **Audit Type**: Independent External Supervisor Re-Audit 003
> **Audited Document**: `docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md` (Revision 3)
> **Audited Commit**: `ff5993970b0fcb97f6dccace9a377404958fca5c`
> **Proposal Authority**: `PROPOSAL-P03-002 Revision 6` (`EXTERNAL_APPROVED`, commit `8709af4b6aaf8613c69a10514971e320e3f90048`)
> **Pinned Upstream Authority**: `Untrivial-ai/agent-orchestrator` `v0.13.0` (commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-23

---

## 1. Executive Summary & Verdict

Independent External Supervisor Re-Audit 003 of ADR-016 Revision 3 (commit `ff5993970b0fcb97f6dccace9a377404958fca5c`) is complete.

All ten Revision 3 findings (`ADR16R3-001` through `ADR16R3-010`) are verified as **CLOSED**:
- `ADR16R3-001` (Exact TaskState Canonical Edge Accounting): CLOSED
- `ADR16R3-002` (Unified `worker_sessions` Candidate Schema): CLOSED
- `ADR16R3-003` (Unified `dispatch_operations` Candidate Schema): CLOSED
- `ADR16R3-004` (`STOP_REQUESTED` Termination Truth Correction): CLOSED
- `ADR16R3-005` (Crash Matrix Reconciliation): CLOSED
- `ADR16R3-006` (Formal Class-B Administrative Absence Policy): CLOSED
- `ADR16R3-007` (Pair Provisioning Concurrency Serialization): CLOSED
- `ADR16R3-008` (Approved AO Adapter Surface Boundary): CLOSED
- `ADR16R3-009` (Stop Confirmation Bounded by Existing Policy): CLOSED
- `ADR16R3-010` (Live Governance Gate Synchronization): CLOSED

However, independent deep feasibility, crash-safety, and public-contract verification exposed eight new findings (`ADR16R4-001` through `ADR16R4-008`), including critical feasibility blockers regarding pinned `/kill` atomicity and worktree path availability.

### Formal Verdicts
- **`PROPOSAL_P03_002`** = `EXTERNAL_APPROVED`
- **`ADR_016`** = `REVISION_4_REQUIRED`
- **`ADR_016_ACCEPTANCE`** = `NOT_GRANTED`
- **`TASK_P03_003`** = `NOT_RELEASED`
- **`P03_CODE`** = `HELD_PENDING_ADR_016_APPROVAL`
- **`ACTIVE_GATE`** = `P03_ADR_016_REVISION_4`

---

## 2. Findings Log (Revision 4)

### `ADR16R4-001` — GENERATION_FENCE_IS_NON_ATOMIC_TOCTOU (CRITICAL)
- **Severity**: Critical (Safety / Protocol Feasibility)
- **Defect**: ADR-016 characterized stop recovery as "generation-fenced". Pinned AO route is `POST /api/v1/sessions/{sessionId}/kill` (`sessions.go:206`), which takes session identity only and does not accept `expectedGeneration`, `terminalGeneration`, or an `If-Match` CAS token. Client-side GET generation -> compare -> POST kill is a `GENERATION_PRECHECK`, not an atomic generation fence; a TOCTOU window exists.
- **Required Remediation**: Document `PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`. Remove claims that client precheck guarantees a newer generation can never be killed. Prohibit automatic `/kill` reissue during restart recovery of `STOP_REQUESTED` (require fail-closed human reconciliation `STOP_REISSUE_REQUIRES_HUMAN`). Truthfully document live stop and quarantine cleanup limitations under session-scoped kill.

### `ADR16R4-002` — ATTEMPT_QUARANTINE_TRUTH_CAN_BE_OVERWRITTEN (CRITICAL)
- **Severity**: Critical (Safety Invariant Integrity)
- **Defect**: `task_attempts.recovery_disposition` was conflated to serve both as a safety quarantine guard (`UNCERTAIN_DELIVERY_QUARANTINE`) and as a diagnostic disposition. Later stop diagnostics (e.g. `STOP_TARGET_ABSENT`) overwrite this field while quarantine is supposed to remain active, erasing the dispatch gate truth.
- **Required Remediation**: Add a dedicated candidate field `task_attempts.quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))`. `recovery_disposition` records diagnostic outcome, while `quarantine_state` independently gates new dispatches. Update `PrepareDispatch` to check `quarantine_state`.

### `ADR16R4-003` — STOP_OPERATION_TASKSTATE_EFFECT_HAS_NO_PURPOSE_CONTEXT (CRITICAL)
- **Severity**: Critical (State Machine Graph Integrity)
- **Defect**: ADR-016 hardcoded stop operation outcomes to drive `RUNNING -> FAILED`. When stop operations are executed for quarantine cleanup after an uncertain delivery attempt has already transitioned `DISPATCHED -> FAILED -> HUMAN_REQUIRED`, attempting `RUNNING -> FAILED` violates canonical graph invariants.
- **Required Remediation**: Add explicit column `purpose TEXT NOT NULL CHECK (purpose IN ('RUNNING_ATTEMPT_STOP', 'QUARANTINE_CLEANUP', 'PAIR_MAINTENANCE'))` to `stop_operations`. Restrict TaskState transitions to `RUNNING_ATTEMPT_STOP` on `RUNNING` tasks. Cleanup and maintenance stops produce ZERO TaskState transitions and ZERO `ended_at` mutations.

### `ADR16R4-004` — STOP_STAGE_PROVENANCE_COLLAPSE (HIGH)
- **Severity**: High (Audit & Provenance Integrity)
- **Defect**: Revision 3 mutated `stage` from `STOP_CALL_SUCCEEDED` to `STOP_CALL_FAILED` upon confirmation timeout or generation mismatch, and mutated `STOP_REQUESTED` to `STOP_CALL_FAILED` when target was already dead. This erases the historical fact of whether the wire call was successfully transmitted and accepted.
- **Required Remediation**: Separate positively proven wire-effect `stage` (`STOP_REQUESTED`, `STOP_CALL_SUCCEEDED`, `STOP_CALL_FAILED`, `STOP_TERMINATION_CONFIRMED`, `STOP_TARGET_ABSENT`) from subsequent reconciliation outcome `resolution_state` (`STOP_CONFIRMATION_TIMEOUT`, `STOP_GENERATION_MISMATCH`, `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED`, `STOP_REISSUE_REQUIRES_HUMAN`). Add `resolved_at TEXT` to persist terminal resolution timestamp.

### `ADR16R4-005` — PROVISION_CONFIRMED_DOES_NOT_AUTHORIZE_A_NEW_SPAWN (CRITICAL)
- **Severity**: Critical (Relational Model & Quarantine Safety)
- **Defect**: Partial unique index on `pair_provisioning_operations` only locks unresolved stages. If a `worker_sessions` row already exists (ACTIVE, IDLE, or QUARANTINED), allowing a new spawn creates a second session and risks overwriting Model-A current binding, bypassing Pair quarantine and orphaning the old runtime.
- **Required Remediation**: Enforce `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`: (1) no `worker_sessions` row exists for the Pair, (2) no unresolved provisioning operations exist, and (3) no Pair/attempt quarantine is active. Existing sessions must be reused, resumed, or administratively resolved. Replace invalid DDL test that asserted subsequent spawn was allowed.

### `ADR16R4-006` — WORKTREE_PATH_HAS_NO_PINNED_PUBLIC_SOURCE (CRITICAL / IMPLEMENTABILITY)
- **Severity**: Critical (Implementability / Pinned Contract Conflict)
- **Defect**: Candidate DDL required `worker_sessions.worktree_path TEXT NOT NULL`. However, pinned AO v0.13.0 does not expose `WorkspacePath` in public API: `SessionMetadata.WorkspacePath` is `json:"-"`, `SessionView` has no workspace path, `SpawnSessionResponse` wraps `SessionView` without workspace path, and Supervisor adapter exposes no workspace path. Requiring `NOT NULL` would force implementation to invent paths or violate encapsulation.
- **Required Remediation**: Change `worker_sessions.worktree_path` to `TEXT NULL` (NULL = unknown/unavailable through pinned public contract). Document `WORKTREE_PATH_PUBLIC_AVAILABILITY = UNAVAILABLE_IN_PINNED_V1`. Do not invent synthetic paths. Record future canonical documentation reconciliation requirement for Phase P03 closeout.

### `ADR16R4-007` — KILL_STOP_TIMEOUT_IS_NOT_RESTART_STABLE (HIGH)
- **Severity**: High (Timeout Determinism Across Restarts)
- **Defect**: Revision 3 bound observation to `SUPERVISOR_KILL_STOP_TIMEOUT` but did not persist an absolute deadline. On daemon restart, restarting the timeout from zero could extend observation indefinitely across repeated crashes.
- **Required Remediation**: Compute and persist `confirmation_deadline_at TEXT` in `stop_operations` upon `STOP_CALL_SUCCEEDED` commit. On restart, evaluate `now >= confirmation_deadline_at` without resetting timer.

### `ADR16R4-008` — CROSS_SECTION_VOCABULARY_STILL_DRIFTS (MEDIUM)
- **Severity**: Medium (Internal Consistency & Traceability)
- **Defect**: Vocabulary tokens drifted across sections: `MISSED_ACTIVE_WINDOW` vs `MISSED_ACTIVE_WINDOW_AMBIGUITY`; `PROVISION_FAILED` vs `PROVISIONING_FAILED`; `PAIR_SESSION_PROVISION_REQUESTED` vs `SESSION_SPAWN_REQUESTED`. D11 summary row implied every stop drives `RUNNING -> FAILED`.
- **Required Remediation**: Establish a single canonical "Canonical ADR-016 V1 Token Registry" subsection defining authoritative tokens across all stages, purposes, resolution states, dispositions, quarantine states, and audit events. Align all sections to this registry. Mark D11 summary TaskState effect as `PURPOSE_DEPENDENT`.

---

## 3. Governance Status

```text
PROPOSAL_P03_002 = EXTERNAL_APPROVED
ADR_016 = REVISION_4_REQUIRED
ADR_016_ACCEPTANCE = NOT_GRANTED
TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_PENDING_ADR_016_APPROVAL
ACTIVE_GATE = P03_ADR_016_REVISION_4
```
