# PROPOSAL-P03-002: Lifecycle Observation, Session Binding & State Reconciliation

> **Proposal ID**: `PROPOSAL-P03-002`
> **Status**: `PENDING_EXTERNAL_APPROVAL`
> **Author**: AI Engineering Supervisor Control Plane Team
> **Date**: 2026-09-22
> **Target Phase**: `P03` (AO Integration & Lifecycle Management)
> **Pre-Code Findings Reconciled**: `P03T3-PRE-001` through `P03T3-PRE-007`
> **Architecture Impact**: `ARCHITECTURE_CHANGE = NO`
> **ADR Requirement**: `ADR_REQUIRED = YES`
> **Policy State**: `SUPERVISOR_ACTIVITY_POLL_INTERVAL = UNSET`, `SUPERVISOR_EXECUTION_DEADLINE = UNSET`

---

## 1. Problem Statement

Phase P03 requires implementing **TASK-P03-003** (*Lifecycle Observation & State Reconciliation*) to bridge the pure AO transport layer (`internal/ao`, completed in TASK-P03-001 and TASK-P03-002) with the Supervisor domain state machine (`internal/domain`, `internal/workflow`) and local persistence engine (`internal/store`).

However, pre-code analysis across commits `e5d3337` and `b85ddf6` revealed seven (7) architectural and persistence gaps:
1. **P03T3-PRE-001**: Neither `Pair` nor `TaskAttempt` persists the AO `session_id`, and SQLite has no `worker_sessions` table.
2. **P03T3-PRE-002**: `ClassifyRestartRecovery()` queries only `state = 'DISPATCHED'`, ignoring tasks in `RUNNING`.
3. **P03T3-PRE-003**: The state machine forbids `DISPATCHED -> REPORT_READY`; if the Supervisor restarts after a task has executed to `idle` while offline, reconciliation is undefined.
4. **P03T3-PRE-004**: AO activity states `waiting_input` and `blocked` have no formal mapping to Supervisor `TaskState`.
5. **P03T3-PRE-005**: Intentional termination (`StopWorker`) has no durable provenance in `StateStore`, risking confusion with abnormal worker crashes on restart.
6. **P03T3-PRE-006**: `TaskAttempt.ended_at` exists in Schema V1 but is never mutated by any Store method.
7. **P03T3-PRE-007**: Operational timeout and polling values remain `UNSET` by governance; the code cannot introduce arbitrary hard-coded numbers.

This proposal provides the architectural reconciliation and formal specification required before TASK-P03-003 can be safely released for production coding.

---

## 2. Current Implementation Ground Truth

1. **Approved AO Adapter (`internal/ao`)**:
   - `CreateWorkerSession(ctx, projectID, harness)`: Spawns worker session via `POST /api/v1/sessions` (exact HTTP 201).
   - `DispatchTaskContract(ctx, sessionID, message)`: Dispatches rendered contract via `POST /api/v1/sessions/{id}/send` (exact HTTP 200).
   - `StopWorker(ctx, sessionID)`: Terminates session via `POST /api/v1/sessions/{id}/kill` (exact HTTP 200, returns `freed`).
   - `ResumeWorker(ctx, sessionID)`: Restores session via `POST /api/v1/sessions/{id}/restore` (exact HTTP 200, validates `restoreMode`).
   - `GetWorkerStatus(ctx, sessionID)`: Queries authoritative read model via `GET /api/v1/sessions/{id}` (exact HTTP 200).
   - Anti-corruption boundary: All wire types are private; zero `StateStore` dependency inside `internal/ao`.
2. **State Store Engine (`internal/store`)**:
   - Schema V1 & V2: Contains tables `projects`, `pairs`, `tasks`, `task_contracts`, `task_attempts`, `audit_events`, `audit_chain_state`.
   - `PrepareDispatch(ctx, params)`: Commits atomic transaction `READY -> DISPATCHED` and inserts `task_attempts` with `ended_at = NULL`.
   - `ClassifyRestartRecovery(ctx)`: Scans only `tasks WHERE state = 'DISPATCHED'`.
3. **Workflow State Machine (`internal/workflow`)**:
   - Enforces exactly 13 canonical `TaskState`s, 22 domain transitions, and 25 lifecycle edges.
   - Transitions out of `DISPATCHED`: only `RUNNING` and `FAILED`.
   - Transitions out of `RUNNING`: only `REPORT_READY`, `FAILED`, and `BLOCKED`.
   - Transitions out of `BLOCKED`: only `HUMAN_REQUIRED`.

---

## 3. Canonical Authority Matrix

| Requirement / Spec | Authority Document | Canonical Mandate |
|---|---|---|
| **FR-005** | `docs/02_REQUIREMENTS.md` | Authoritative worker state observation via AO session activity. |
| **FR-006** | `docs/02_REQUIREMENTS.md` | Intentional vs unexpected termination distinguished by operation provenance. |
| **FR-015** | `docs/02_REQUIREMENTS.md` | Preflight identity and readiness verification. |
| **NFR-003** | `docs/02_REQUIREMENTS.md` | Crash-consistent state store with zero data corruption across restarts. |
| **NFR-005** | `docs/02_REQUIREMENTS.md` | Non-blocking bounded observation polling; fail-closed network errors. |
| **REC-014** | `docs/14_FAILURE_RECOVERY.md` | Host restart recovery scanning `DISPATCHED` and `RUNNING` tasks. |
| **ADR-002** | `docs/adr/ADR-002*` | AO is the runtime execution control plane; Supervisor is the governance control plane. |
| **ADR-011** | `docs/adr/ADR-011*` | `AO_IDLE != REPORT_READY`. Turn completion observation precedes report ingestion. |
| **ADR-012** | `docs/adr/ADR-012*` | `TaskAttempt` binds monotonically to `TaskContract` revision; atomic pre-dispatch commit. |
| **ADR-015** | `docs/adr/ADR-015*` | Local SQLite StateStore with write transactions and append-only hash-chained audit log. |

---

## 4. Durable WorkerSession Binding Analysis

### 4.1 Canonical Domain Model Alignment
In `docs/05_DOMAIN_MODEL.md`, the entity `WorkerSession` is canonically defined:
```text
class WorkerSession {
    +string session_id
    +string pair_id
    +string runtime_type
    +string worktree_path
    +string worker_agent_id
    +WorkerStatus status
}
Pair "1" o-- "1" WorkerSession
```

### 4.2 Current Schema Deficiency
Currently, SQLite Schema V1 omits `worker_sessions`. Neither `pairs` nor `task_attempts` records an AO `session_id`.
As a result:
- When a worker session is created in AO, the supervisor cannot store its `session_id`.
- The Supervisor poller in TASK-P03-003 would have no persistent key to pass to `GetWorkerStatus(ctx, sessionID)`.

### 4.3 Binding Relationship Resolution
- `GIẢ ĐỊNH`: A `Pair` represents the long-lived collaboration lane for a project workspace.
- `Pair 1 -> 1 WorkerSession`: At any given moment, a `Pair` has at most one active `WorkerSession`.
- `TaskAttempt 1 -> 1 WorkerSession`: Crucially, each `TaskAttempt` represents an immutable execution iteration. For auditability, each `TaskAttempt` must record the exact AO `session_id` to which it was dispatched.
- **Why Pair-only binding is insufficient**: If a worker session crashes and Attempt 2 spawns a fresh session, a Pair-only binding would overwrite the session ID of Attempt 1, corrupting the historical audit lineage of Attempt 1.
- **Recommended Schema Enhancement (Schema V3)**:
  1. Add table `worker_sessions`:
     ```sql
     CREATE TABLE IF NOT EXISTS worker_sessions (
         session_id TEXT PRIMARY KEY,
         pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
         runtime_type TEXT NOT NULL, -- e.g. 'ao'
         harness TEXT NOT NULL,      -- e.g. 'agy'
         status TEXT NOT NULL,       -- e.g. 'active', 'idle', 'terminated'
         created_at TEXT NOT NULL,
         ended_at TEXT
     );
     ```
  2. Add foreign key column to `task_attempts`:
     ```sql
     session_id TEXT REFERENCES worker_sessions(session_id) ON DELETE RESTRICT
     ```
  3. Add `active_worker_session_id` to `pairs`:
     ```sql
     active_worker_session_id TEXT REFERENCES worker_sessions(session_id) ON DELETE SET NULL
     ```

---

## 5. Attempt / Session Identity Analysis

### 5.1 Single Session Spanning Multiple Attempts
- Can one AO session span multiple `TaskAttempt`s?
  - `GIẢ ĐỊNH`: YES. In conversational / interactive workflows, if Attempt 1 requires revision (`REVISION_REQUIRED`), the supervisor may dispatch Attempt 2 to the same session via `restoreMode: native` or `saved_prompt` to preserve LLM conversational context, OR it may spawn a fresh session.
  - Therefore, the relationship is `WorkerSession 1 -> 0..* TaskAttempt`, while `TaskAttempt 1 -> 1 WorkerSession`.

### 5.2 Preventing Stale Session Reconciliation
- To prevent accidentally reconciling a stale session against a newly allocated attempt:
  - An attempt must only reconcile against the session explicitly recorded in its own `session_id` field.
  - The supervisor verifies that the session's current upstream activity corresponds to a timestamp `lastActivityAt >= attempt.started_at`.

---

## 6. Restart Reconciliation Analysis

### 6.1 Requirements for Both DISPATCHED and RUNNING
Per REC-014, on supervisor daemon restart, the StateStore must scan for all tasks in:
- `DISPATCHED`
- `RUNNING`

### 6.2 Updated Classification Algorithm
`Store.ClassifyRestartRecovery()` must be expanded:
1. Query tasks where `state IN ('DISPATCHED', 'RUNNING')`.
2. For each task, fetch the latest attempt (`attempt_number = current_attempt`).
3. If attempt is missing or `ended_at IS NOT NULL`:
   - Classify as `INCONSISTENT_PERSISTED_STATE`.
4. If attempt has `ended_at IS NULL`:
   - Classify as `EXTERNAL_RECONCILIATION_REQUIRED`.
5. Return candidates to the Supervisor lifecycle reconciler.

---

## 7. Missed Active Window Options

### 7.1 Scenario
1. Supervisor commits `PrepareDispatch` (`READY -> DISPATCHED`).
2. Supervisor calls `DispatchTaskContract(sessionID, message)`.
3. Supervisor crashes or host reboots.
4. Worker executes, finishes turn, and returns to `idle`.
5. Supervisor restarts: Task is persisted as `DISPATCHED`, but AO reports `ActivityIdle`.

### 7.2 Option Comparison

| Metric | Option A: Fail Task on Missed Window | Option B: Controlled Sequential Reconciliation (Recommended) | Option C: Introduce Reconciling State |
|---|---|---|---|
| **Safety** | High (conservative). | High (requires cryptographic proof of turn execution). | Medium (adds unverified intermediate states). |
| **Auditability** | Abrupt failure (`MISSED_ACTIVE_WINDOW`). | Complete sequence: `DISPATCHED -> RUNNING` then `RUNNING -> REPORT_READY`. | Non-standard lifecycle states in audit trail. |
| **State Machine Impact** | **ZERO**: uses `DISPATCHED -> FAILED`. | **ZERO**: follows exact canonical transitions `DISPATCHED -> RUNNING -> REPORT_READY`. | **BREAKING**: requires new `TaskState` and transitions. |
| **ADR Impact** | None. | None (conforms to existing 22 transitions). | Requires new Architecture Decision Record. |
| **Worker Effort Preservation** | **Zero**: discards valid work completed by worker. | **High**: preserves valid work if verified by `lastActivityAt > started_at`. | High. |
| **Stale Session Risk** | None. | Mitigated: requires `lastActivityAt > attempt.started_at`. | Mitigated. |

### 7.3 Recommendation
- `GIẢ ĐỊNH`: Adopt **Option B**. If `lastActivityAt > attempt.started_at`, the supervisor reconciler executes the canonical atomic sequence:
  1. Transition `DISPATCHED -> RUNNING` with reason `RECONCILIATION_ACTIVE_INFERRED`.
  2. Transition `RUNNING -> REPORT_READY` with reason `RECONCILIATION_IDLE_OBSERVED` (upon verifying report readiness).
- If `lastActivityAt <= attempt.started_at`: The worker never executed post-dispatch; transition `DISPATCHED -> FAILED` with reason `DISPATCH_UNACKNOWLEDGED_IDLE`.

---

## 8. Waiting / Blocked Mapping Options

### 8.1 Analysis of Pinned AO Activity States
From pinned AO authority (`backend/internal/domain/activity.go`):
- `ActivityActive`: Agent is actively running tool operations.
- `ActivityIdle`: Agent completed turn; prompt loop is at rest.
- `ActivityWaitingInput`: Agent at empty prompt awaiting next instruction.
- `ActivityBlocked`: Agent stopped on a pending tool-permission or approval dialog.
- `ActivityExited`: ConPTY process / agent exited.

### 8.2 Supervisor Mapping Decisions

1. **`ActivityWaitingInput`**:
   - In an autonomous supervisor lane, an agent should not ask interactive questions.
   - `GIẢ ĐỊNH`: If observed during `RUNNING`, transition `RUNNING -> BLOCKED` with trigger `WORKER_WAITING_INPUT`.
   - `BLOCKED` then triggers `BLOCKED -> HUMAN_REQUIRED`, allowing human intervention to unblock the session.
2. **`ActivityBlocked`**:
   - Represents an agent blocked on a permission/approval dialog.
   - `GIẢ ĐỊNH`: Mapped directly to canonical transition `RUNNING -> BLOCKED` with trigger `WORKER_PERMISSION_BLOCKED`.
   - Automated senders MUST NOT send keystrokes into a blocked session (pinned AO invariant).
   - Transitions to `HUMAN_REQUIRED` so operator can grant/deny permissions.
3. **`ActivityExited`**:
   - ConPTY process exited before turn completion.
   - If unexpected: transition `RUNNING -> FAILED` (or `DISPATCHED -> FAILED`) with reason `WORKER_PROCESS_EXITED`.
4. **`isTerminated: true`**:
   - Session terminated in AO.
   - Evaluated against intentional stop provenance (Section 10).

---

## 9. Termination Classification

1. **Intentional Termination**:
   - Initiated by Supervisor / Operator via `StopWorker`.
   - Matched against durable stop provenance.
   - Mapped to `DISPATCHED -> FAILED` or `RUNNING -> FAILED` with specific reason `INTENTIONAL_STOP_BY_SUPERVISOR`.
   - Does NOT trigger automated retry.
2. **Unexpected Termination / Crash**:
   - Absence of stop provenance.
   - ConPTY died, host killed process, or AO reaper terminated session.
   - Mapped to `FAILED` with reason `WORKER_ABNORMAL_TERMINATION`.
   - Eligible for retry iteration up to max attempts.

---

## 10. Intentional Stop Provenance

### 10.1 Provenance Architecture
- `internal/ao.StopWorker` remains a pure transport call.
- The Supervisor orchestration layer manages provenance:
  1. Before calling `AOAdapter.StopWorker(ctx, sessionID)`, the orchestration layer records a durable audit event:
     - `event_type`: `WORKER_STOP_REQUESTED`
     - `details`: `{"session_id": "...", "task_id": "...", "attempt_id": "...", "actor": "SUPERVISOR"}`
  2. Orchestration layer marks `worker_sessions.status = 'terminating'` or sets `ended_at`.
  3. Orchestration layer calls `AOAdapter.StopWorker(ctx, sessionID)`.
  4. Upon restart or polling observation of `isTerminated: true` / `ActivityExited`, the presence of `WORKER_STOP_REQUESTED` proves intentionality.

---

## 11. TaskAttempt ended_at Ownership

### 11.1 Owner & Moment for Setting `ended_at`
- `TaskAttempt.ended_at` marks the conclusion of the attempt's active execution.
- `PROCESS_ALIVE != TURN_RUNNING` and `AO_IDLE != REPORT_READY`.
- `GIẢ ĐỊNH`: `ended_at` MUST NOT be set upon merely seeing `ActivityIdle`. An attempt is still evaluating report presence during the handoff to `REPORT_READY`.
- `ended_at` is set when:
  1. Task successfully transitions to `REPORT_READY` (normal execution turn ended, report exists).
  2. Task transitions to `FAILED` (abnormal exit, timeout, crash, missing report).
  3. Task transitions to `BLOCKED` (stopped on permission dialog).
  4. Task transitions to `CANCELLED` (operator cancellation).
- **Missing Store Operation**:
  - `Store.RecordAttemptEnd(ctx, attemptID, endedAt, rawReport)` must be introduced to update `task_attempts.ended_at` atomically during state transition.

---

## 12. Poll / Execution Policy Injection

### 12.1 Unset Policy Compliance
Per governance mandate, operational values remain strictly `UNSET`:
```text
SUPERVISOR_ACTIVITY_POLL_INTERVAL = UNSET
SUPERVISOR_EXECUTION_DEADLINE = UNSET
```

### 12.2 Implementation Contract
- TASK-P03-003 observer constructor MUST require explicit policy configuration:
  ```go
  type PollerConfig struct {
      ActivityPollInterval time.Duration
      ExecutionDeadline    time.Duration
  }
  ```
- **Fail-Fast Rule**: If `ActivityPollInterval <= 0` or `ExecutionDeadline <= 0`, constructor returns `ErrInvalidPolicy`.
- **Zero Package Defaults**: Package `internal/ao` and `internal/workflow` MUST NOT contain fallback default constants.
- **Deterministic Testability**: Poller accepts an injectable `Clock` / `Ticker` interface for 100% deterministic testing without `time.Sleep`.

---

## 13. Comprehensive Recovery Matrix

| Persisted Task State | Authoritative AO Snapshot | Observation Classification | TaskState Action | State Changed? | Attempt Ends? | Required Provenance | Audit Event | Retry? | Human Escalation? | Authority |
|---|---|---|---|---|---|---|---|---|---|---|
| `DISPATCHED` | `active` | Worker started processing | Transition `RUNNING` | YES | NO | AO snapshot `activity.state = active` | `TASK_STATE_TRANSITION` (`DISPATCHED -> RUNNING`) | N/A | NO | FR-005, ADR-011 |
| `DISPATCHED` | `idle` (`lastActivityAt > started_at`) | Missed active window (turn completed) | Transition `RUNNING` then `REPORT_READY` | YES | YES | `lastActivityAt > attempt.started_at` | `TASK_STATE_TRANSITION` (`RECONCILIATION_ACTIVE_INFERRED`) | N/A | NO | REC-014, P03T3-PRE-003 |
| `DISPATCHED` | `idle` (`lastActivityAt <= started_at`) | Unacknowledged dispatch | Transition `FAILED` | YES | YES | Zero post-dispatch activity | `TASK_STATE_TRANSITION` (`DISPATCH_UNACKNOWLEDGED`) | YES | If retries exhausted | NFR-003, REC-014 |
| `DISPATCHED` | `waiting_input` | Unexpected prompt pre-execution | Transition `RUNNING` then `BLOCKED` | YES | YES | AO snapshot `waiting_input` | `TASK_STATE_TRANSITION` (`BLOCKED`) | NO | YES (`-> HUMAN_REQUIRED`) | Pinned AO `activity.go` |
| `DISPATCHED` | `blocked` | Permission dialog pre-execution | Transition `RUNNING` then `BLOCKED` | YES | YES | AO snapshot `blocked` | `TASK_STATE_TRANSITION` (`BLOCKED`) | NO | YES (`-> HUMAN_REQUIRED`) | Pinned AO `activity.go` |
| `DISPATCHED` | `exited` | Worker process died | Transition `FAILED` | YES | YES | AO snapshot `exited`, no stop provenance | `TASK_STATE_TRANSITION` (`WORKER_PROCESS_EXITED`) | YES | If retries exhausted | `docs/06`, REC-014 |
| `DISPATCHED` | `isTerminated: true` (intentional) | Intentional stop during dispatch | Transition `FAILED` (or cancelled) | YES | YES | AO `isTerminated` + `WORKER_STOP_REQUESTED` | `TASK_STATE_TRANSITION` (`INTENTIONAL_STOP`) | NO | Operator directed | FR-006 |
| `DISPATCHED` | `isTerminated: true` (unintentional)| Worker terminated abnormally | Transition `FAILED` | YES | YES | AO `isTerminated`, zero stop provenance | `TASK_STATE_TRANSITION` (`UNEXPECTED_TERMINATION`) | YES | If retries exhausted | FR-006, REC-014 |
| `RUNNING` | `active` | Active execution ongoing | Remain in `RUNNING` | NO | NO | Snapshot `active` within deadline | Telemetry diagnostic log | N/A | NO | FR-005, ADR-011 |
| `RUNNING` | `idle` | Worker turn completed | Transition `REPORT_READY` (or `FAILED` if no report) | YES | YES | Snapshot `idle` with `lastActivityAt > started_at` | `TASK_STATE_TRANSITION` (`RUNNING -> REPORT_READY`) | N/A | NO | ADR-011 |
| `RUNNING` | `waiting_input` | Unexpected prompt during turn | Transition `BLOCKED` | YES | YES | Snapshot `waiting_input` | `TASK_STATE_TRANSITION` (`RUNNING -> BLOCKED`) | NO | YES (`-> HUMAN_REQUIRED`) | Pinned AO `activity.go` |
| `RUNNING` | `blocked` | Tool permission dialog | Transition `BLOCKED` | YES | YES | Snapshot `blocked` | `TASK_STATE_TRANSITION` (`RUNNING -> BLOCKED`) | NO | YES (`-> HUMAN_REQUIRED`) | Pinned AO `activity.go` |
| `RUNNING` | `exited` | Process crashed mid-turn | Transition `FAILED` | YES | YES | Snapshot `exited`, zero stop provenance | `TASK_STATE_TRANSITION` (`WORKER_CRASHED`) | YES | If retries exhausted | REC-014 |
| `RUNNING` | `isTerminated: true` (intentional) | Operator stopped active run | Transition `FAILED` | YES | YES | Snapshot `isTerminated` + `WORKER_STOP_REQUESTED` | `TASK_STATE_TRANSITION` (`INTENTIONAL_STOP`) | NO | Operator directed | FR-006 |
| `RUNNING` | `isTerminated: true` (unintentional)| Unannounced termination | Transition `FAILED` | YES | YES | Snapshot `isTerminated`, zero stop event | `TASK_STATE_TRANSITION` (`WORKER_TERMINATED`) | YES | If retries exhausted | FR-006, REC-014 |
| `ANY` | AO Session 404 (Not Found) | Session missing in AO | Transition `FAILED` | YES | YES | AO 404 `SESSION_NOT_FOUND` error | `TASK_STATE_TRANSITION` (`AO_SESSION_NOT_FOUND`) | YES | If retries exhausted | NFR-005 |
| `ANY` | AO Daemon Unavailable (Network) | Transient connection error | Retry probe / Hold state | NO | NO | HTTP connection refused / 5xx | Warning diagnostic log | N/A | If past recovery threshold | REC-001, REC-014 |

---

## 14. Diagnostic Activity Rule: lastActivityAt

### 14.1 Immutable Rule
```text
lastActivityAt = DIAGNOSTIC_ACTIVITY_EVIDENCE
lastActivityAt != heartbeat
```
- `lastActivityAt` reflects the timestamp of the latest hook callback received by AO.
- **What it PROVES**: A tool execution, prompt submission, or hook event occurred at or before this timestamp.
- **What it DOES NOT PROVE**: It does **NOT** prove a worker is hung or dead. A long tool call (e.g. build, large test suite, heavy refactor) produces zero intermediate hook signals.
- **Prohibition**: The poller MUST NOT implement logic such as *"if now - lastActivityAt > threshold then mark worker hung/crashed"*. Hang detection is governed strictly by the caller-injected `SUPERVISOR_EXECUTION_DEADLINE`.

---

## 15. Synthetic Heartbeat Prohibition

- `worker_heartbeat`: **NOT IMPLEMENTED**.
- AO SSE comment heartbeat: **TRANSPORT KEEPALIVE ONLY** (used by proxies to keep connections open; conveys zero domain/worker health semantics).
- `/api/v1/events`: **NOT TASK-P03-003 BASELINE**.
- CDC / SSE: **DEFERRED OPTIONAL FUTURE OPTIMIZATION**.
- **Approved TASK-P03-003 Baseline**: Bounded snapshot polling via `GET /api/v1/sessions/{id}`.

---

## 16. State Machine Impact
- **Impact**: `ZERO`.
- All proposed reconciliations strictly conform to the 13 canonical `TaskState`s and 22 domain transitions in `internal/workflow/state_machine.go`:
  - `DISPATCHED -> RUNNING` (canonical)
  - `DISPATCHED -> FAILED` (canonical)
  - `RUNNING -> REPORT_READY` (canonical)
  - `RUNNING -> BLOCKED` (canonical)
  - `RUNNING -> FAILED` (canonical)
  - `BLOCKED -> HUMAN_REQUIRED` (canonical)
- Zero new states or transition edges are added.

---

## 17. Persistence Schema Impact
- **Impact**: Schema V3 migration required in `internal/store/migrations.go`:
  1. Add table `worker_sessions`.
  2. Add `session_id` column to `task_attempts`.
  3. Add `active_worker_session_id` to `pairs`.
  4. Add method `Store.RecordAttemptEnd(ctx, attemptID, endedAt)`.
  5. Expand `Store.ClassifyRestartRecovery(ctx)` to query `state IN ('DISPATCHED', 'RUNNING')`.

---

## 18. Audit Trail Impact
- Append-only hash-chained `audit_events` (Schema V2) will record:
  1. `WORKER_STOP_REQUESTED`: Intentional stop provenance.
  2. `RESTART_RECONCILIATION_EXECUTED`: Details on candidate classification and reconciliation path.
  3. `TASK_STATE_TRANSITION`: Standard state transitions with detailed triggers (`WORKER_ACTIVITY_ACTIVE`, `WORKER_ACTIVITY_IDLE`, `MISSED_ACTIVE_WINDOW`).

---

## 19. Architecture Impact & ADR Assessment

### 19.1 Architecture Impact Classification
```text
ARCHITECTURE_CHANGE = NO
```
*Rationale*: The domain model already canonically defines `WorkerSession` in `docs/05_DOMAIN_MODEL.md`. The workflow state machine remains strictly 13 states and 22 transitions. No architectural boundaries or external system roles are modified.

### 19.2 ADR Requirement Assessment
```text
ADR_REQUIRED = YES
```
*Rationale*: A dedicated Architecture Decision Record (e.g. `ADR-016: WorkerSession Persistence and Lifecycle Reconciliation`) is required under `docs/24_CHANGE_GOVERNANCE.md` to authorize:
1. SQLite Schema V3 migration (`worker_sessions` table, `task_attempts.session_id`, `pairs.active_worker_session_id`).
2. The exact restart recovery classification contract for `RUNNING` tasks.
3. Attempt `ended_at` state store mutation semantics.

---

## 20. Recommended TASK-P03-003 Scope

Upon External Supervisor approval of this proposal and subsequent ADR authorization, TASK-P03-003 scope should consist of:
1. **Schema V3 Migration**: Adding `worker_sessions` and session foreign keys in `internal/store`.
2. **Store Recovery Extension**: Updating `ClassifyRestartRecovery` to handle `DISPATCHED` and `RUNNING` tasks.
3. **Attempt End Management**: Implementing `RecordAttemptEnd` in `internal/store`.
4. **Lifecycle Observer**: Implementing bounded snapshot observation and state reconciliation in an orchestration package (e.g. `internal/workflow` or `internal/observer`), consuming `internal/ao.Client` with injected policy values.
5. **Zero Polling in `internal/ao`**: `internal/ao` remains a pure HTTP transport client.

---

## 21. Exact Files Expected To Change After Approval

- `internal/domain/entities.go` (Add `WorkerSession` struct; add `SessionID` to `TaskAttempt` and `ActiveWorkerSessionID` to `Pair`)
- `internal/store/migrations.go` (Schema V3 migration)
- `internal/store/task_store.go` & `internal/store/dispatch.go` (Session binding & attempt end recording)
- `internal/store/recovery.go` (Handling `DISPATCHED` and `RUNNING`)
- `internal/workflow/` or `internal/supervisor/` (Lifecycle poller and state reconciler)
- Unit and integration tests in corresponding packages.

---

## 22. Explicit Non-Goals

- NO workspace file fetching or report extraction (`TASK-P03-004`).
- NO report JSON parsing or claim validation (`P04`).
- NO direct ConPTY or Agy CLI execution.
- NO SQLite database access inside `internal/ao`.
- NO invented numeric timeout or interval defaults.
- NO SSE `/api/v1/events` or CDC event bus implementation.

---

## 23. External Approval Gate

```text
PROPOSAL_P03_002 = PENDING_EXTERNAL_APPROVAL
TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_003_PRECODE_AUDIT
```

Execution is halted awaiting independent External Supervisor audit and decision on this proposal.
