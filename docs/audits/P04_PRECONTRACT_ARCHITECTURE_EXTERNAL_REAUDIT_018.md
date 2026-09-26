# External Re-Audit Report 018: Phase P04 Pre-Contract Architecture (Revision 19 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `5e2cff1b1edd3c065dd6786efc9b95d356c3f045`
> **Date**: 2026-09-27
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_19`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 018 evaluated the Phase P04 architecture remediation deliverables submitted in commit `5e2cff1b1edd3c065dd6786efc9b95d356c3f045` (Revision 19 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 19)
2. `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 9)
3. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 19)
4. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 19)
5. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_017.md`
6. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

### Finding Review Status
- **`P04-ARCH-R17-001`**: **CLOSED_AT_DESIGN_LEVEL**. Verified latency reconciliation with `PROPOSAL-P04-002 Revision 9`, zero boolean compliance flags, and RFC 8785 JCS event descriptors.
- **`P04-ARCH-R17-002`**: **CLOSED_AT_DESIGN_LEVEL**. Verified accurate evidence classification in the readiness matrix without scratch script citations.
- **`P04-ARCH-R17-003`**: **CLOSED_AT_DESIGN_LEVEL**. Verified synchronization of document metadata labels and diagrams across normative specifications.
- **`P04-ARCH-R18-001`**: **CLOSED_AT_DESIGN_LEVEL**. Verified decoupling of `latency_measurement_status` from the 3,000 ms threshold; measurement provenance strictly reflects execution path (`MEASURED_IN_PROCESS` vs `RECOVERED_AFTER_RESTART`).
- **`P04-ARCH-R18-002`**: **CLOSED_AT_DESIGN_LEVEL**. Verified commit return duration telemetry is measured post-commit purely as in-process monotonic telemetry and is strictly excluded from `REVIEW_BUNDLE_GENERATED`, database tables, and the audit chain.
- **`PROPOSAL-P04-002 Revision 9`**: **APPROVED_AS_DESIGN_BASELINE**. Frozen byte-for-byte; no edits in this turn.
- **`P04-ARCH-R19-001`**: **OPEN**. Independent workspace binding creation transaction violates existing code and DDL constraints (`task_attempts` FK, `dispatch_operations` lineage trigger), opening a crash/concurrency window prior to dispatch.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_20_REQUIRED`
- `PROPOSAL_P04_002 = APPROVED_AS_DESIGN_BASELINE`
- `ADR_018 = REVISION_20_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_20_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_19`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`
- `VERDICT = REVISION_20_REQUIRED`

---

## 2. Review of Round 17 and 18 Findings

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R17-001` | Latency Reconciliation & RFC 8785 JCS Event IDs in PROPOSAL-002 | **CLOSED_AT_DESIGN_LEVEL** | Delimiter concatenation eliminated; JCS descriptors specified; compliance status strictly UNVERIFIED; threshold breach diagnostics decoupled. |
| `P04-ARCH-R17-002` | Accurate Evidence Classification in Readiness Matrix | **CLOSED_AT_DESIGN_LEVEL** | Distinctions between verified static probes, design consistency, and unexecuted runtime contract tests preserved. |
| `P04-ARCH-R17-003` | Synchronization of Document Metadata and Diagrams | **CLOSED_AT_DESIGN_LEVEL** | Metadata headers, gate labels, and diagrams synchronized across the Revision 19 suite and Revision 9 baseline. |
| `P04-ARCH-R18-001` | Measurement Provenance Decoupled from Latency Threshold | **CLOSED_AT_DESIGN_LEVEL** | 2 × 2 matrix implemented in `PROPOSAL-002` and `ADR-018`; provenance reflects continuous vs recovery lifetime without threshold conflation. |
| `P04-ARCH-R18-002` | Contradictory Commit Return Telemetry in PLAN-P04 | **CLOSED_AT_DESIGN_LEVEL** | Line 81 corrected; post-commit monotonic telemetry strictly excluded from audit events and persistence. |

---

## 3. New Finding: P04-ARCH-R19-001 — Workspace Binding / Dispatch Atomicity Unresolved

### `P04-ARCH-R19-001`: Workspace Binding / Dispatch Atomicity Unresolved
- **Severity**: Critical
- **Description**: Current documentation describes an independent "Binding Creation Transaction" performed "after TaskContract release and before worker dispatch". This description is incompatible with active code and DDL:
  1. `attempt_workspace_bindings` defines a foreign key `attempt_id REFERENCES task_attempts(attempt_id)`.
  2. The lineage trigger `trg_attempt_workspace_bindings_lineage_guard` requires `dispatch_operations` of the exact attempt, task, contract, session, and generation to already exist.
  3. P03 currently creates `task_attempts`, `dispatch_operations(stage='DISPATCH_BOUND')`, audit `TASK_DISPATCH_BOUND`, and CAS `READY -> DISPATCHED` in `Store.PrepareBoundDispatch`.
  4. After `PrepareBoundDispatch` commits, Coordinator proceeds immediately to `GetWorkerStatus` and `RecordSendRequested`.
  5. If the binding transaction runs before `PrepareBoundDispatch`, the FK and trigger constraints cannot be satisfied because neither `task_attempts` nor `dispatch_operations` exists yet.
  6. If the binding transaction runs after `PrepareBoundDispatch`, a crash/concurrency window exists: the attempt is already `DISPATCH_BOUND` and the task is `DISPATCHED`, but no workspace binding exists; the documentation lacks recovery rules or guards preventing `SEND_REQUESTED`.
- **Mandatory Remediation Directives**:
  1. **Unified Atomic Transaction**: Replace the independent "Binding Creation Transaction" with a single atomic transaction: **Bound Dispatch + Workspace Binding Transaction**.
  2. **Pre-Transaction OS Handle & Identity Verification**:
     - Before opening the SQLite transaction, open and hold verified OS handles for the worktree root and linked gitdir.
     - Query canonical paths, `VolumeSerialNumber`, and 128-bit `FileIdInfo`.
     - If handles cannot be acquired/held or identities are invalid: fail-closed, abort immediately without opening the SQLite transaction, and do NOT create any attempt.
  3. **Strict In-Transaction Execution Ordering**:
     In the same SQLite transaction, execute in exact sequence:
     - Check Task, Contract, and Pair guards (`state = 'READY'`, latest frozen contract, active lane empty, Pair clean).
     - Issue `task_attempts` row with immutable session and generation snapshot.
     - Create `dispatch_operations` row at stage `DISPATCH_BOUND`.
     - Create `attempt_workspace_bindings` row at state `ACTIVE`.
     - Append audit event `TASK_DISPATCH_BOUND`.
     - Append audit event `WORKSPACE_BINDING_CREATED`.
     - CAS `tasks` from `READY` to `DISPATCHED` and increment `current_attempt`.
     - Commit single transaction.
  4. **Rollback Atomicity Guarantee**:
     Any error prior to commit must roll back the entire transaction:
     - Zero orphaned `task_attempts`.
     - Zero orphaned `dispatch_operations`.
     - Zero orphaned `attempt_workspace_bindings`.
     - Zero orphaned audit events.
     - Task remains in `READY` state.
  5. **Stage-Aware Lineage Trigger**:
     Trigger `trg_attempt_workspace_bindings_lineage_guard` must explicitly require `dispatch_operations.stage = 'DISPATCH_BOUND'` at the time of insertion, in addition to exact attempt, task, contract, session, and terminal generation matches.
  6. **Fail-Closed Guard in `RecordSendRequested`**:
     `RecordSendRequested` must fail-closed unless exactly one binding exists in `attempt_workspace_bindings` satisfying:
     - `attempt_id`, `task_id`, `contract_id`, `session_id`, `terminal_generation` match exact lineage;
     - `binding_state = 'ACTIVE'`;
     - Physical identity is revalidated against managed handles/paths;
     - No active integrity hold or stronger lifecycle hold exists.
  7. **Crash / Replay Matrix Requirements**:
     - Crash before commit: entire transaction rolls back; Task remains `READY`.
     - Crash after commit but before `GetWorkerStatus` or `RecordSendRequested`: `DISPATCH_BOUND` operation and `ACTIVE` workspace binding coexist atomically; recovery proceeds only after identity revalidation.
     - Missing binding, binding mismatch, or binding in `INVALIDATED`: strictly forbid `RecordSendRequested`, forbid `/send`, remain fail-closed, emit diagnostic under approved authority.
     - Replay with exact same binding: idempotent.
     - Replay with different physical identity or lineage mismatch: reject, do not overwrite.
  8. **Store Seam Ownership**:
     Eliminate any claims of non-mutation of existing task creation transactions. Subtask P04A must own the controlled extension of the `PrepareBoundDispatch` store seam.
  9. **PLAN P04A Scope & Acceptance Criteria**:
     Explicitly state that P04A includes the necessary Store dispatch seam, fault injection tests at each step of the transaction, and tests verifying `RecordSendRequested` rejection when binding is missing, mismatched, non-ACTIVE, or identity changed. Runtime admission remains closed after P04A; the chain P04A -> P04B -> P04C -> P04D remains strictly preserved.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001`, `DRAFT-ADR-018`, and `PLAN-P04-EVIDENCE-REVIEW` to Revision 20 incorporating Directives 1 through 9.
2. Keep `PROPOSAL-P04-002 Revision 9` byte-for-byte preserved.
3. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_19` and `P04_ARCH_R19_001 = REMEDIATION_SUBMITTED`.
4. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
