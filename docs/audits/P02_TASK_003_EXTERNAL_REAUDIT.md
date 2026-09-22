# P02 TASK-003 External Reaudit Record (Revision 3 Required)

## Audit Metadata

| Field | Value |
|---|---|
| **Audited Remote Commit** | `a53b744c41a2c19aa89cc7437c1456fa9e3e9051` |
| **Logical Task Baseline** | `272bd038eaf4856ef8e05fd5e21980816d647b9c` |
| **Audit Date** | 2026-09-22 |
| **Phase Gate** | `P02_TASK_003_REVISION_3` |

---

## Verdict

```
TASK_P02_002 = EXTERNAL_AUDIT_APPROVED
TASK_P02_003 = REVISION_REQUIRED
TASK_P02_003_REVISION = 3
P02_004 = NOT_RELEASED
ACTIVE_GATE = P02_TASK_003_REVISION_3
```

---

## Revision-2 Resolved Findings

```
GENERIC_READY_TO_DISPATCHED_BYPASS = RESOLVED
INVALID_INITIAL_TASK_STATE_BYPASS = RESOLVED
NONCANONICAL_REPORT_PATH_ACCEPTED = RESOLVED
STALE_CONTRACT_REVISION_DISPATCHABLE = RESOLVED
PAIR_ACTIVE_LANE_IMPLEMENTATION = APPROVED_IN_SOURCE
CONTRACT_LINEAGE_CORE_CHECKS = APPROVED_IN_PRINCIPLE
```

---

## Evidence Finding & SHA Correction

```
P02T003-R3-004 = WORKER_REPORTED_COMMIT_SHA_MISMATCH
```

| Property | Value |
|---|---|
| **Previous Worker Reported Rev-2 SHA** | `a53b744ec6576b5cf8402dbca252e6fb1c499ea5` |
| **Actual Remote Rev-2 Commit SHA** | `a53b744c41a2c19aa89cc7437c1456fa9e3e9051` |
| **Classification** | `EVIDENCE_REPORT_ACCURACY_DEVIATION` |
| **Source Impact** | `NONE` |
| **Acceptance Impact** | Clean detached worktree must be rerun against exact remote commit |

The previously reported clean-worktree execution SHA contained a transcription error and is not accepted as authoritative proof. Clean-worktree testing will be executed against the exact remote commit.

---

## Blocking Findings (Revision 3 Scope)

### P02T003-R3-001 — READY_ENTRY_CONTRACT_PRECONDITION_MISSING
`TransitionTask` permitted `DRAFT -> READY` without verifying that a governing TaskContract revision has been persisted for the task. Additionally:
- `REVISION_REQUIRED -> READY` must require a new latest contract revision that is still mutable (`is_immutable = false`).
- `FAILED -> READY` (retry without spec change) must require that the latest contract is already frozen (`is_immutable = true`).
All READY-entry contract checks must occur within the same write transaction as the CAS state update.

### P02T003-R3-002 — CONTRACT_INSERT_TASK_STATE_UNGUARDED
`InsertTaskContract` did not validate the lifecycle state of the target task. A new mutable TaskContract may only be inserted while the task is in an allowed planning state:
- Revision 1: Task must be in `DRAFT` state (`supersedes_contract_id == nil`).
- Revision > 1: Task must be in `REVISION_REQUIRED` or `DRAFT` state (supporting replanning).
All other states (including `READY`, `FAILED`, `DISPATCHED`, etc.) must reject new contract insertion with `ErrContractInsertState`. The lifecycle-state check, predecessor lineage verification, and insertion must be wrapped in a single SQLite transaction.

### P02T003-R3-003 — RECOVERY_NON_NULL_EMPTY_ENDED_AT_ACCEPTED
`ClassifyRestartRecovery` previously used `endedAt.Valid && endedAt.String != ""`, which would treat `ended_at = ''` as an open attempt. An attempt is open if and only if `ended_at IS NULL`. Any non-null `ended_at` on a `DISPATCHED` task must be classified as `INCONSISTENT_PERSISTED_STATE` and return `ErrInconsistentPersistedState`.
