# P03_TASK_003_PRECODE_EXTERNAL_AUDIT.md - Pre-Code Reconciliation Audit for TASK-P03-003

> **Audited Base SHA**: `b85ddf686b7432f087ef776444eaabe183176e67`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **TASK-P03-001 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK-P03-002 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK-P03-003 Status**: `NOT_RELEASED`
> **P03 Code Status**: `HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION`
> **Active Gate**: `P03_TASK_003_PRECODE_RECONCILIATION`

---

## 0. Executive Audit Summary

Independent external analysis has identified seven (7) critical architectural, lifecycle, and persistence gaps that prevent the safe implementation of **TASK-P03-003** (*Lifecycle Observation & State Reconciliation*).
These findings establish that persistent session binding, recovery classification for `RUNNING` tasks, missed active window reconciliation, activity state mapping, intentional stop provenance, attempt `ended_at` semantics, and policy parameter injection must be formally reconciled and governed before production code may be released.

**Verdict**: Production coding for TASK-P03-003 remains strictly **HELD**. Findings P03T3-PRE-001 through P03T3-PRE-007 are **OPEN**.

---

## 1. Findings Matrix

| Finding ID | Title | Canonical Source | Current Code File | Severity | Implementation Status |
|---|---|---|---|---|---|
| `P03T3-PRE-001` | `DURABLE_WORKER_SESSION_BINDING_NOT_IMPLEMENTED` | `docs/05_DOMAIN_MODEL.md` | `internal/domain/entities.go`, `internal/store/migrations.go` | **BLOCKER** | **HELD** |
| `P03T3-PRE-002` | `RESTART_RECONCILIATION_COVERAGE_INCOMPLETE` | `docs/14_FAILURE_RECOVERY.md` (REC-014) | `internal/store/recovery.go` | **BLOCKER** | **HELD** |
| `P03T3-PRE-003` | `MISSED_ACTIVE_WINDOW_SEMANTICS_UNDEFINED` | `docs/06_WORKFLOW_STATE_MACHINE.md`, ADR-011 | `internal/workflow/state_machine.go` | **BLOCKER** | **HELD** |
| `P03T3-PRE-004` | `WAITING_BLOCKED_TERMINATION_MAPPING_UNDEFINED` | Pinned AO `backend/internal/domain/activity.go` | `internal/ao/sessions.go`, `internal/domain/task_state.go` | **BLOCKER** | **HELD** |
| `P03T3-PRE-005` | `INTENTIONAL_STOP_PROVENANCE_NOT_DURABLY_DEFINED` | `docs/02_REQUIREMENTS.md` (FR-006) | `internal/ao/session_commands.go`, `internal/store/` | **BLOCKER** | **HELD** |
| `P03T3-PRE-006` | `TASK_ATTEMPT_END_SEMANTICS_UNDEFINED` | `docs/05_DOMAIN_MODEL.md`, ADR-012 | `internal/store/dispatch.go`, `internal/store/migrations.go` | **BLOCKER** | **HELD** |
| `P03T3-PRE-007` | `POLL_AND_DEADLINE_POLICY_VALUES_UNSET` | `docs/12_UPSTREAM_INTEGRATION.md`, NFR-005 | `docs/18_CURRENT_STATE.md` | **BLOCKER** | **HELD** |

---

## 2. Detailed Finding Records

### P03T3-PRE-001: DURABLE_WORKER_SESSION_BINDING_NOT_IMPLEMENTED
- **Literal Current-Code Evidence**:
  - `internal/domain/entities.go`: `Pair` struct (lines 19-26) contains only `PairID`, `ProjectID`, `CurrentPhaseID`, `ActiveTaskID`, `State`, `CreatedAt`. It has no session identifier.
  - `internal/domain/entities.go`: `TaskAttempt` struct (lines 72-81) contains `AttemptID`, `AttemptNumber`, `TaskID`, `ContractID`, `ExpectedReportPath`, `StartedAt`, `EndedAt`, `WorkerReportRaw`. It has no AO session identifier.
  - `internal/store/migrations.go`: Schema V1 tables `pairs` and `task_attempts` lack session columns. There is zero `worker_sessions` table in SQLite schema.
- **Canonical Evidence**:
  - `docs/05_DOMAIN_MODEL.md`: Defines entity `WorkerSession` (attributes: `session_id`, `pair_id`, `runtime_type`, `worktree_path`, `worker_agent_id`, `status`). Relationship specifies: `Pair "1" o-- "1" WorkerSession`.
  - `docs/12_UPSTREAM_INTEGRATION.md` & `docs/04_ARCHITECTURE.md`: Specify that the Supervisor creates and manages worker sessions via Agent Orchestrator.
- **Conflict / Gap**:
  The Supervisor control plane cannot durably correlate an execution attempt or collaboration lane to an AO upstream session. When `CreateWorkerSession` succeeds, the returned `session_id` cannot be persisted in SQLite. Re-attaching to a session across restarts or during lifecycle observation is impossible without durable binding.
- **Severity**: **BLOCKER**.
- **Code Status**: **HELD**. Production implementation of TASK-P03-003 is blocked until a durable binding schema is authorized.

---

### P03T3-PRE-002: RESTART_RECONCILIATION_COVERAGE_INCOMPLETE
- **Literal Current-Code Evidence**:
  - `internal/store/recovery.go`: `ClassifyRestartRecovery` queries:
    ```sql
    SELECT task_id, current_attempt FROM tasks WHERE state = 'DISPATCHED' ORDER BY task_id ASC
    ```
  - Zero logic or queries exist for tasks in `RUNNING` state.
- **Canonical Evidence**:
  - `docs/14_FAILURE_RECOVERY.md` (REC-014, line 25):
    "On startup, scan State Store for tasks in `DISPATCHED`/`RUNNING`; verify AO worktrees; resume or mark `FAILED`."
- **Conflict / Gap**:
  P02 implemented persistence-only classification solely for `DISPATCHED`. If the host machine reboots or the Supervisor daemon crashes while a worker is actively executing a task in `RUNNING`, the restart recovery scanner ignores the task completely. This causes orphaned `RUNNING` tasks that never reconcile.
- **Severity**: **BLOCKER**.
- **Code Status**: **HELD**.

---

### P03T3-PRE-003: MISSED_ACTIVE_WINDOW_SEMANTICS_UNDEFINED
- **Literal Current-Code Evidence**:
  - `internal/workflow/state_machine.go`: `allowedTransitions[domain.StateDispatched]` permits transitions only to `domain.StateRunning` and `domain.StateFailed`. There is NO edge `DISPATCHED -> REPORT_READY`.
- **Canonical Evidence**:
  - `docs/06_WORKFLOW_STATE_MACHINE.md`: Canonical state machine defines exactly 22 domain transitions.
  - ADR-011: Requires `ActivityActive` to observe execution entry, and `ActivityIdle` to observe turn completion.
- **Conflict / Gap**:
  If the Supervisor crashes post-dispatch and the worker executes quickly to completion before Supervisor re-launch, the authoritative AO snapshot on restart reports `ActivityIdle`. The Supervisor sees persisted state `DISPATCHED`. Because `DISPATCHED -> REPORT_READY` is illegal in the state machine, attempting to mark the completed turn directly violates domain invariants.
- **Severity**: **BLOCKER**.
- **Code Status**: **HELD**.

---

### P03T3-PRE-004: WAITING_BLOCKED_TERMINATION_MAPPING_UNDEFINED
- **Literal Current-Code Evidence**:
  - `internal/ao/sessions.go`: Normalizes 5 canonical activity states: `active`, `idle`, `waiting_input`, `blocked`, `exited`.
  - `internal/domain/task_state.go` & `internal/workflow/state_machine.go`: Zero mappings exist translating AO activity states to Supervisor task states.
- **Canonical Evidence**:
  - Pinned AO `backend/internal/domain/activity.go`:
    "waiting_input is an agent at an empty prompt awaiting its next INSTRUCTION (safe to message or nudge), while blocked is an agent stopped on a pending DECISION — a tool-permission or approval dialog — where a stray keystroke could answer the dialog on the user's behalf. Automated senders must never inject input into a blocked session."
- **Conflict / Gap**:
  It is undefined whether an autonomous worker reporting `waiting_input` (unexpected prompt) or `blocked` (permission dialog) should transition the Task to `BLOCKED`, fail the attempt, or remain in `RUNNING`. Furthermore, equating AO `ActivityBlocked` directly with Supervisor `StateBlocked` without explicit authority risks conflating workflow blocking with interactive dialog waiting.
- **Severity**: **BLOCKER**.
- **Code Status**: **HELD**.

---

### P03T3-PRE-005: INTENTIONAL_STOP_PROVENANCE_NOT_DURABLY_DEFINED
- **Literal Current-Code Evidence**:
  - `internal/ao/session_commands.go`: `StopWorker` calls `POST /api/v1/sessions/{id}/kill` and returns `StopWorkerResult{SessionID, Freed}`. It has zero interaction with `StateStore`.
  - `internal/store/`: Contains zero tables or methods recording intentional worker termination requests.
- **Canonical Evidence**:
  - `docs/02_REQUIREMENTS.md` (FR-006): Requires distinguishing intentional supervisor termination from unexpected crashes using Supervisor operation provenance.
- **Conflict / Gap**:
  If an intentional stop is executed via AO transport and the Supervisor crashes immediately afterward, on restart AO reports `isTerminated: true` or `ActivityExited`. Without a durable record of the intentional stop, the recovery scanner cannot distinguish between an operator cancel and an abnormal worker crash, leading to corrupt audit trails.
- **Severity**: **BLOCKER**.
- **Code Status**: **HELD**.

---

### P03T3-PRE-006: TASK_ATTEMPT_END_SEMANTICS_UNDEFINED
- **Literal Current-Code Evidence**:
  - `internal/store/migrations.go`: `task_attempts` schema includes `ended_at TEXT`.
  - `internal/store/dispatch.go`: `PrepareDispatch` inserts new attempts with `ended_at = NULL`.
  - `internal/store/`: Zero methods exist to update or set `ended_at`.
- **Canonical Evidence**:
  - `docs/05_DOMAIN_MODEL.md`: Defines `ended_at` timestamp on `TaskAttempt`.
  - ADR-012: Governs attempt execution boundaries.
- **Conflict / Gap**:
  There is no defined state or event that owns setting `ended_at`. Setting `ended_at` on worker turn completion (`idle`) risks closing the attempt before the report is validated (`REPORT_READY`); conversely, leaving `ended_at` unset causes attempts to appear perpetually running across restarts. A dedicated StateStore operation is missing.
- **Severity**: **BLOCKER**.
- **Code Status**: **HELD**.

---

### P03T3-PRE-007: POLL_AND_DEADLINE_POLICY_VALUES_UNSET
- **Literal Current-Code Evidence**:
  - `internal/ao`: Contains zero hard-coded timeout numbers or polling interval loops.
  - `docs/18_CURRENT_STATE.md`: Records `SUPERVISOR_ACTIVITY_POLL_INTERVAL = UNSET` and `SUPERVISOR_EXECUTION_DEADLINE = UNSET`.
- **Canonical Evidence**:
  - NFR-005 & `docs/12_UPSTREAM_INTEGRATION.md`: Mandates bounded polling and task execution deadlines.
- **Conflict / Gap**:
  Operational policy values remain unset by governance. Implementing TASK-P03-003 cannot invent arbitrary numbers. The observation and reconciliation engine must be designed with strict policy injection (options pattern / caller configuration) and fail-fast validation, paired with virtual tickers for deterministic test coverage.
- **Severity**: **BLOCKER**.
- **Code Status**: **HELD**.

---

## 3. Mandatory Governance Conclusion

Production coding for TASK-P03-003 is **PROHIBITED** until the accompanying proposal [`PROPOSAL-P03-002-lifecycle-reconciliation-and-session-binding.md`](../proposals/PROPOSAL-P03-002-lifecycle-reconciliation-and-session-binding.md) is reviewed and approved by the External Supervisor.
