# P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_002.md — External Supervisor Re-Audit Record (PROPOSAL-P03-002 Revision 2)

> **Audited Commit**: `d2100633ef92c225cb250c95277202251cfacb51`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: `PROPOSAL_P03_002 = REVISION_3_REQUIRED`
> **Architecture Impact**: `P03_ARCHITECTURE_CHANGE = YES`
> **ADR Requirement**: `P03_ADR_REQUIRED = YES`
> **ADR-016 Authorization**: `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET`
> **TASK_P03_001 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_002 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_003 Status**: `NOT_RELEASED`
> **P03 Code Status**: `HELD`
> **Active Gate**: `P03_TASK_003_PROPOSAL_REVISION_3`

---

## 0. External Supervisor Re-Audit Decision Summary

Independent external re-audit of `PROPOSAL-P03-002 Revision 2` (commit `d2100633ef92c225cb250c95277202251cfacb51`) is complete.
Revision 2 corrected most Revision-1 defects, but the proposal is **NOT yet approved** for ADR drafting.
Six (6) external findings (`P03T3PR3-001` through `P03T3PR3-006`) were recorded.

**Verdict**: `PROPOSAL_P03_002 = REVISION_3_REQUIRED`.

---

## 1. External Re-Audit Findings Matrix

| Finding ID | Title | Verdict | Severity | Remediation Mandate |
|---|---|---|---|---|
| `P03T3PR3-001` | `DISPATCH_SAGA_NOT_GATING_LIFECYCLE_RECONCILIATION` | **REVISION_3_REQUIRED** | Blocker | Matrix interpreted `DISPATCHED` + AO activity without conditioning on dispatch saga stage. Invariant: `DISPATCH_BOUND` activity is NOT evidence of current attempt; `SEND_REQUESTED` without `SEND_CONFIRMED` activity is not safely attributable; `DISPATCHED + active -> RUNNING` is permitted ONLY IF `dispatch_stage == SEND_CONFIRMED` AND session/generation match. Matrix must incorporate Dispatch Saga Stage. |
| `P03T3PR3-002` | `SEND_REQUESTED_EVIDENCE_AND_STATE_PATH_INCORRECT` | **REVISION_3_REQUIRED** | Blocker | `SEND_REQUESTED` is committed before transport call; it proves only `SEND_INTENT_DURABLY_RECORDED`, NOT that "request was transmitted to network". Canonical state graph has NO direct `DISPATCHED -> HUMAN_REQUIRED`; legal path is `DISPATCHED -> FAILED`, followed by `FAILED -> HUMAN_REQUIRED`. Correct evidence claims and transition paths. |
| `P03T3PR3-003` | `UNCERTAIN_DELIVERY_RETRY_QUARANTINE_UNDEFINED` | **REVISION_3_REQUIRED** | Blocker | Crash at `SEND_REQUESTED` leaves `DELIVERY_OUTCOME = UNKNOWN`. Moving to `FAILED` allows canonical `FAILED -> READY`, which would permit a blind duplicate retry. Define `UNCERTAIN_DELIVERY_QUARANTINE` invariant and evaluate PrepareDispatch retry guard options (Pair/WorkerSession metadata, dispatch-op flag, attempt disposition) for ADR-016. |
| `P03T3PR3-004` | `STOP_PROVENANCE_WIRE_CONTRACT_WRONG` | **REVISION_3_REQUIRED** | Major | Pinned AO router and Supervisor StopWorker client call `POST /api/v1/sessions/{sessionId}/kill` (returning `KillSessionResponse`), NOT `DELETE /api/v1/sessions/{id}`. Correct `STOP_REQUESTED` wire contract to `/kill` and do not equate HTTP 200 with confirmed termination. |
| `P03T3PR3-005` | `PRESPAWN_MOVES_BUT_DOES_NOT_SOLVE_SPAWN_CRASH_WINDOW` | **REVISION_3_REQUIRED** | Major | Pre-creating Pair WorkerSession decouples task dispatch from session spawn, but does NOT solve spawn idempotency (`PRESPAWN = TASK_DISPATCH_DECOUPLING_OPTION`, `PRESPAWN != DETERMINISTIC_SPAWN_RECOVERY`). Pair provisioning path still has T2 -> T3 crash window. Keep `P03T3_SESSION_SPAWN_RECOVERY = UNRESOLVED_BLOCKER`. |
| `P03T3PR3-006` | `PRE_SEND_ACTIVITY_CONTRACT_INCOMPLETE` | **REVISION_3_REQUIRED** | Major | Remove `ActivityWorking` (not a valid AO activity state). Analyze pre-send safety at `DISPATCH_BOUND` across all valid states (`active`, `idle`, `waiting_input`, `blocked`, `exited`, `isTerminated`). Define `PRE_SEND_ADMISSIBILITY` (`DECIDED` or `UNRESOLVED_DECISION`) for each. Do not treat `active` as safe first-dispatch state. |

---

## 2. Governance Determination

1. `P03_ARCHITECTURE_CHANGE = YES` (amends dispatch saga integration, uncertain delivery retry quarantine, pre-send activity admissibility, and domain relationship models).
2. `P03_ADR_REQUIRED = YES`.
3. `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET` (must wait until Proposal Revision 3 passes independent External Supervisor re-audit).
4. `TASK_P03_003 = NOT_RELEASED`.
5. `P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION`.
6. `ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT_003`.
