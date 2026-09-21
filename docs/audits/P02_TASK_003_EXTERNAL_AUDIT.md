# P02 TASK-003 External Audit Record (Revision 2 Required)

## Audit Metadata

| Field | Value |
|---|---|
| **Audited Remote Commit** | `accc68b27734135d373be849d5c4ad04ca75556f` |
| **Logical Task Baseline** | `272bd038eaf4856ef8e05fd5e21980816d647b9c` |
| **Audit Date** | 2026-09-22 |
| **Phase Gate** | `P02_TASK_003_REVISION_2` |

---

## Verdict

```
TASK_P02_002 = EXTERNAL_AUDIT_APPROVED
TASK_P02_003 = REVISION_REQUIRED
TASK_P02_003_REVISION = 2
P02_004 = NOT_RELEASED
ACTIVE_GATE = P02_TASK_003_REVISION_2
```

---

## Approved Portions

```
SQLITE_DEPENDENCY = APPROVED
SQLITE_PRAGMAS = APPROVED
MIGRATION_V1_BASE = APPROVED
ATOMIC_PREPAREDISPATCH_TRANSACTION = APPROVED_IN_PRINCIPLE
ROLLBACK = APPROVED
CONCURRENT_SAME_TASK_DISPATCH = APPROVED
JSON_NUMBER_STORE_ROUNDTRIP = APPROVED
CLEAN_WORKTREE_EVIDENCE = PASS_REPORTED
```

---

## Accepted Evidence Classification

```
REMOTE_SOURCE_ARTIFACT = COMPLETE
WORKER_CLEAN_WORKTREE_BUILD = PASS_REPORTED_WITH_EXPLICIT_GO_C
WORKER_CLEAN_WORKTREE_TEST = PASS_REPORTED_WITH_EXPLICIT_GO_C
WORKER_CLEAN_WORKTREE_VET = PASS_REPORTED_WITH_EXPLICIT_GO_C
GITHUB_REMOTE_CI = NOT_PRESENT
```

---

## Blocking Findings (Revision 2 Scope)

### P02T003-R2-001 — GENERIC_READY_TO_DISPATCHED_BYPASS
`TransitionTask` permitted `READY -> DISPATCHED` via canonical workflow state machine evaluation, completely bypassing the atomic `PrepareDispatch` pre-dispatch transaction (allocating attempt, binding contract, report path, contract freeze, current_attempt increment).
**Required Fix**: `TransitionTask` must explicitly reject `READY -> DISPATCHED` with `ErrAtomicDispatchRequired`. Only `PrepareDispatch` may persist this state transition.

### P02T003-R2-002 — INVALID_INITIAL_TASK_STATE_BYPASS
`CreateTask` accepted any of the 13 canonical task states and non-zero `current_attempt`. The canonical lifecycle has exactly one initial pseudo-edge (`[*] -> DRAFT`).
**Required Fix**: Production `CreateTask` must require `state == DRAFT` and `current_attempt == 0`, returning `ErrInvalidInitialTaskState` for any other initial state. Corrupt persisted fixtures in tests must use package-internal raw SQL instead of the production API.

### P02T003-R2-003 — NONCANONICAL_REPORT_PATH_ACCEPTED
ADR-011 establishes the immutable canonical report path: `.supervisor/reports/<task_id>/<attempt_id>.json`. `PrepareDispatch` accepted arbitrary caller-provided path strings.
**Required Fix**: Enforce exact equality with `CanonicalExpectedReportPath(taskID, attemptID)`, validating single-segment safety for `taskID` and `attemptID`. Reject mismatches with `ErrReportPathMismatch` before state mutation.

### P02T003-R2-004 — STALE_CONTRACT_REVISION_DISPATCHABLE
`PrepareDispatch` verified contract existence and task ownership but did not check if the contract was the latest persisted revision of the task.
**Required Fix**: Query highest `revision_number` for the task. Require `selected_revision == latest_revision`, otherwise reject with `ErrStaleContractRevision`. Retries (`FAILED -> READY`) may reuse an already-frozen latest revision.

### P02T003-R2-005 — CONTRACT_LINEAGE_PERSISTENCE_INCOMPLETE
`InsertTaskContract` did not validate revision lineage constraints at persistence time.
**Required Fix**: Reject already-immutable contract on insert. For `revision_number == 1`, require `supersedes_contract_id == nil`. For `revision_number > 1`, require referenced previous revision exists, belongs to same `task_id`, has `revision_number - 1`, identical `base_sha`, and `is_immutable == true`. Reject violations with `ErrInvalidContractLineage`.

### P02T003-R2-006 — PAIR_ACTIVE_LANE_INVARIANT_MISSING
Canonical Domain Model requires: Exactly one task may be in `DISPATCHED`, `RUNNING`, or `REVIEWING` per Pair at any time.
**Required Fix**: Both `PrepareDispatch` and `TransitionTask` (when transitioning to `RUNNING` or `REVIEWING`) must verify no OTHER task under the same pair is currently in `DISPATCHED`, `RUNNING`, or `REVIEWING`. Reject concurrency conflicts with `ErrPairBusy`.

### P02T003-R2-007 — RECOVERY_ACCEPTS_ENDED_ATTEMPT
`ClassifyRestartRecovery` matched attempts on `(task_id, current_attempt)` without verifying that the attempt is still open.
**Required Fix**: Require `ended_at IS NULL` for a valid `EXTERNAL_RECONCILIATION_REQUIRED` candidate. If `ended_at IS NOT NULL` while task is still `DISPATCHED`, classify as `INCONSISTENT_PERSISTED_STATE` and return `ErrInconsistentPersistedState`.
