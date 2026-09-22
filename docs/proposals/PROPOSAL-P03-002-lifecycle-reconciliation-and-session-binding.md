# PROPOSAL-P03-002: Lifecycle Observation, Session Binding & State Reconciliation (Revision 1)

> **Proposal ID**: `PROPOSAL-P03-002`
> **Revision**: `1`
> **Status**: `REVISION_1_PENDING_EXTERNAL_REAUDIT`
> **Author**: AI Engineering Supervisor Control Plane Team
> **Date**: 2026-09-22
> **Target Phase**: `P03` (AO Integration & Lifecycle Management)
> **Pre-Code Findings Reconciled**: `P03T3-PRE-001` through `P03T3-PRE-007`
> **Proposal Audit Findings Closed**: `P03T3PR1-001` through `P03T3PR1-010`
> **Architecture Impact**: `P03_ARCHITECTURE_CHANGE = YES`
> **ADR Requirement**: `P03_ADR_REQUIRED = YES` (`ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET`)
> **Policy State**: `SUPERVISOR_ACTIVITY_POLL_INTERVAL = UNSET`, `SUPERVISOR_EXECUTION_DEADLINE = UNSET`

---

## 1. Problem Statement

Phase P03 requires implementing **TASK-P03-003** (*Lifecycle Observation & State Reconciliation*) to bridge the pure AO transport layer (`internal/ao`, approved in TASK-P03-001 and TASK-P03-002) with the Supervisor domain state machine (`internal/domain`, `internal/workflow`) and local persistence engine (`internal/store`).

An independent external audit of the initial proposal identified ten (10) critical defects (`P03T3PR1-001` through `P03T3PR1-010`) requiring formal contract revision:
1. Authority matrix misattribution (FR-005/FR-006, NFR-005).
2. Unresolved crash window between AO session creation and local durable binding (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`).
3. Unaddressed domain relationship drift regarding `WorkerSession`.
4. Overclaimed proofs regarding `lastActivityAt`.
5. Scope violation regarding `REPORT_READY` ownership (owned by P04, not TASK-P03-003).
6. Inaccurate mapping of AO `waiting_input`, `blocked`, and `exited` activity states.
7. Non-fail-closed worker termination classification (must preserve `WORKER_TERMINATION_UNKNOWN`).
8. Stop provenance that is not execution-generation safe.
9. Unresolved transaction atomicity and phase boundaries for `TaskAttempt.ended_at`.
10. Invented retry and threshold policies without canonical authority.

This Revision 1 resolves each finding with rigorous canonical authority alignment and evidence from the pinned AO repository.

---

## 2. Current Implementation Ground Truth

1. **Approved AO Adapter (`internal/ao`)**:
   - `CreateWorkerSession(ctx, projectID, harness)`: Spawns worker session via `POST /api/v1/sessions` (exact HTTP 201).
   - `DispatchTaskContract(ctx, sessionID, message)`: Dispatches rendered contract via `POST /api/v1/sessions/{id}/send` (exact HTTP 200).
   - `StopWorker(ctx, sessionID)`: Terminates session via `POST /api/v1/sessions/{id}/kill` (exact HTTP 200, returns `freed`).
   - `ResumeWorker(ctx, sessionID)`: Restores session via `POST /api/v1/sessions/{id}/restore` (exact HTTP 200, validates `restoreMode`).
   - `GetWorkerStatus(ctx, sessionID)`: Queries authoritative read model via `GET /api/v1/sessions/{id}` (exact HTTP 200). Exposes normalized `TerminalGeneration`.
   - Anti-corruption boundary: All wire types are private; zero `StateStore` dependency inside `internal/ao`.
2. **State Store Engine (`internal/store`)**:
   - Schema V1 & V2: Contains tables `projects`, `pairs`, `tasks`, `task_contracts`, `task_attempts`, `audit_events`, `audit_chain_state`.
   - `PrepareDispatch(ctx, params)`: Commits atomic transaction `READY -> DISPATCHED` and inserts `task_attempts` with `ended_at = NULL`.
   - `ClassifyRestartRecovery(ctx)`: Scans only `tasks WHERE state = 'DISPATCHED'`.
3. **Workflow State Machine (`internal/workflow`)**:
   - Enforces exactly 13 canonical `TaskState`s, 22 domain transitions, and 25 lifecycle edges.
   - Transitions out of `DISPATCHED`: only `RUNNING` and `FAILED`. (No `DISPATCHED -> REPORT_READY`, no `DISPATCHED -> CANCELLED`).
   - Transitions out of `RUNNING`: only `REPORT_READY`, `FAILED`, and `BLOCKED`. (No `RUNNING -> CANCELLED`).
   - Transitions out of `BLOCKED`: only `HUMAN_REQUIRED`.

---

## 3. Canonical Authority Matrix (P03T3PR1-001 Reconciled)

| Requirement / Spec | Authority Document | Canonical Mandate |
|---|---|---|
| **FR-005** | `docs/02_REQUIREMENTS.md` | **Worker Task Dispatch**: Atomic dispatch of immutable Task Contracts to worker sessions. |
| **FR-006** | `docs/02_REQUIREMENTS.md` | **Worker Lifecycle Observation**: Authoritative observation of worker lifecycle and distinguishing intentional stops from unexpected crashes. |
| **FR-015** | `docs/02_REQUIREMENTS.md` | **Preflight Verification**: Preflight daemon identity and readiness verification. |
| **NFR-003** | `docs/02_REQUIREMENTS.md` | **Recoverability**: Crash-consistent state store with zero data corruption across restarts. |
| **NFR-004** | `docs/02_REQUIREMENTS.md` | **Auditability**: Tamper-evident append-only audit trail for all lifecycle decisions. |
| **NFR-005** | `docs/02_REQUIREMENTS.md` | **Upstream Decoupling**: Anti-corruption adapters isolating internal domain from external protocol changes. |
| **Bounded Polling Authority** | `docs/12_UPSTREAM_INTEGRATION.md`, `docs/phases/P03_AO_INTEGRATION.md`, approved `PROPOSAL-P03-001`, **FR-006** | Bounded snapshot polling via `GET /api/v1/sessions/{id}`. (NFR-005 is Upstream Decoupling; it is not the polling authority). |
| **REC-014** | `docs/14_FAILURE_RECOVERY.md` | **Unexpected Machine Restart**: On startup, scan StateStore for tasks in `DISPATCHED`/`RUNNING`; verify live state; resume or mark `FAILED`. |
| **ADR-002** | `docs/adr/ADR-002*` | AO is the execution control plane; Supervisor is the governance control plane. |
| **ADR-011** | `docs/adr/ADR-011*` | `AO_IDLE != REPORT_READY`. Turn completion observation precedes report ingestion. |
| **ADR-012** | `docs/adr/ADR-012*` | `TaskAttempt` binds monotonically to `TaskContract` revision; atomic pre-dispatch commit. |
| **ADR-015** | `docs/adr/ADR-015*` | Local SQLite StateStore with write transactions and append-only hash-chained audit log. |

---

## 4. Session Spawn Crash Window & Idempotency Analysis (P03T3PR1-002 Reconciled)

### 4.1 Pinned API Fact
Inspection of exact pinned AO source (`backend/internal/httpd/controllers/dto.go:SpawnSessionRequest` at commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) confirms:
```text
PINNED_SPAWN_IDEMPOTENCY = ABSENT
```
`POST /api/v1/sessions` accepts only:
`projectId`, `issueId`, `trackerProvider`, `kind`, `harness`, `branch`, `mode`, `prompt`, `model`, `displayName`, `attachments`.
It contains **NO** `idempotencyKey`, **NO** caller-supplied `sessionId`, and **NO** opaque `correlationId`.
The session ID is generated internally by AO at spawn time.

### 4.2 The Crash Window Timeline
```text
T0: Supervisor commits PrepareDispatch (Task = DISPATCHED, TaskAttempt = durable, ended_at = NULL).
T1: Supervisor dispatches POST /api/v1/sessions to AO.
T2: AO successfully creates session, allocates worktree/ConPTY, assigns generated session_id, and returns HTTP 201 Created.
T3: Supervisor commits session_id to local StateStore.
```
A crash or power failure between **T2 and T3** leaves the system in a critical state:
- An AO worker session exists upstream and is consuming system resources.
- The Supervisor StateStore contains a durable `TaskAttempt`, but its `session_id` is completely unknown.

### 4.3 Evaluation of Recovery Options

| Option | Description | Pinned AO Feasibility | Crash Point Coverage | Deterministic Guarantee | Stale Session Risk | Verdict |
|---|---|---|---|---|---|---|
| **Option A** | Pre-existing Pair WorkerSession reuse | Feasible for existing session; does not solve initial spawn | T1, T2, T3 | High (session already persisted before dispatch) | Low | **Partially Viable** (only if session pre-spawned) |
| **Option B** | Post-PrepareDispatch spawn followed by local bind | Current model | T2 -> T3 crash window | **Zero** (session ID lost if crash occurs before T3) | High | **Defective** |
| **Option C** | Durable "spawn pending" saga record | Requires local saga record | T1 -> T2 -> T3 | Low without correlation key in AO | High | **Incomplete** |
| **Option D** | Query `GET /api/v1/sessions?projectId=...` on restart | AO exposes session query | T2 -> T3 | **Heuristic Only** (cannot distinguish our session from another session in multi-session project) | Very High | **Unsafe Heuristic** |
| **Option E** | Pinned API Correlation Extension | Not supported in pinned commit | N/A | Impossible without upstream modification | N/A | **Infeasible** (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`) |

### 4.4 Resolution & Blocker Status
Because pinned AO lacks idempotency or correlation keys on spawn, a crash between T2 and T3 cannot be deterministically resolved via public APIs without risk of attaching to the wrong session or leaking an orphan session.
```text
P03T3_SESSION_SPAWN_RECOVERY = UNRESOLVED_BLOCKER
```
`GIẢ ĐỊNH`: To achieve deterministic recovery, the future ADR must decide whether:
1. Worker sessions must be explicitly spawned and durably bound to the `Pair` *prior* to task dispatch (so `PrepareDispatch` binds to an already-durable `session_id`); OR
2. If crash occurs with `session_id = NULL` post-dispatch, the attempt is fail-closed to `FAILED` and an operator/reconciliation tool reaps unreferenced AO sessions.

---

## 5. WorkerSession Domain Model Alignment (P03T3PR1-003 Reconciled)

### 5.1 Canonical Domain Specification
In `docs/05_DOMAIN_MODEL.md`:
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
External Supervisor Decision:
```text
P03_ARCHITECTURE_CHANGE = YES
P03_ADR_REQUIRED = YES
```
*Rationale*: Reconciling `WorkerSession` persistence and introducing attempt-session bindings modifies domain entity relationships and execution ownership semantics not presently defined in `docs/05`.

### 5.2 Relationship Options Analysis

1. **Option 1: Pair owns exactly one current WorkerSession; TaskAttempt stores immutable session/generation snapshot**
   - `Pair 1 -> 1 WorkerSession`: Exactly matches canonical `docs/05`.
   - `TaskAttempt` stores snapshot columns (`session_id`, `terminal_generation`) recorded at dispatch time.
   - *Trade-off*: Clean, minimal schema; perfectly preserves canonical Pair single-lane model; historical session records exist through attempt lineage.
2. **Option 2: Pair owns many historical WorkerSessions with an explicit active pointer**
   - `Pair 1 -> 1..* WorkerSession` with `Pair.active_worker_session_id`.
   - *Trade-off*: Retains full session history in a dedicated table; requires foreign key cycle (`Pair -> WorkerSession -> Pair`); amends canonical `Pair "1" o-- "1" WorkerSession`.
3. **Option 3: Dedicated AttemptSessionBinding entity / join table**
   - Independent relational mapping between attempts and sessions.
   - *Trade-off*: High flexibility; adds query complexity; excessive overhead for 1-to-1 attempt dispatch.

`GIẢ ĐỊNH`: **Option 1** is recommended because it strictly preserves canonical `docs/05` domain boundaries (`Pair 1 o-- 1 WorkerSession`) while equipping `TaskAttempt` with the immutable audit snapshot (`session_id`, `terminal_generation`) needed for recovery and audit trails.

### 5.3 Candidate Schema V3 (Proposal-Level Only)
```sql
-- Candidate WorkerSessions Table (Mapping to all canonical fields)
CREATE TABLE IF NOT EXISTS worker_sessions (
    session_id TEXT PRIMARY KEY,
    pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
    runtime_type TEXT NOT NULL,       -- Canonical: 'ao'
    worktree_path TEXT NOT NULL,      -- Canonical: worktree path
    worker_agent_id TEXT NOT NULL,    -- Canonical: e.g. 'agy' (harness identifier)
    status TEXT NOT NULL,             -- Canonical: WorkerStatus ('active', 'idle', 'terminated')
    created_at TEXT NOT NULL,
    ended_at TEXT
);

-- TaskAttempt snapshot enhancement (Proposed canonical amendment)
ALTER TABLE task_attempts ADD COLUMN session_id TEXT REFERENCES worker_sessions(session_id) ON DELETE RESTRICT;
ALTER TABLE task_attempts ADD COLUMN terminal_generation TEXT;
```
- **Existing Canonical Mapping**: `session_id`, `pair_id`, `runtime_type`, `worktree_path`, `worker_agent_id`, `status`.
- **New Persistence Metadata**: `created_at`, `ended_at`.
- **Proposed Canonical Amendment**: `task_attempts.session_id`, `task_attempts.terminal_generation`.

---

## 6. Execution Epoch & Terminal Generation Analysis (Section 7 Reconciled)

### 6.1 Session ID vs Execution Epoch
An AO session ID persists across restore and restart operations. Therefore:
```text
SESSION_ID != EXECUTION_EPOCH
```
If a session is restored, its `session_id` remains identical, but its runtime process, hook registration, and terminal generation change.

### 6.2 Upstream Authority: TerminalGeneration
In pinned AO (`backend/internal/controllers/sessions.go:1969`):
```go
terminalGeneration := s.Metadata.RuntimeLaunchID
```
`TerminalGeneration` maps directly to upstream `RuntimeLaunchID`.
- **Fresh Spawn**: Generates initial unique `RuntimeLaunchID`.
- **Restore / Relaunch**: `lifecycle.Manager` allocates a fresh `RuntimeLaunchID`.
- **Kill**: Session is marked `isTerminated = true`; `RuntimeLaunchID` remains associated with the terminated generation.
- **Runtime Replacement**: Session reaper or switch generates a new generation.

### 6.3 Attempt Fencing
`GIẢ ĐỊNH`: To prevent an attempt from observing activity belonging to a previous or future runtime generation, `TaskAttempt` binding must store:
```text
Attempt Binding Identity = (session_id, terminal_generation)
```
When reconciling observations, if the session's upstream `TerminalGeneration` does not match the attempt's recorded `terminal_generation`, the observation belongs to a different execution episode and must not trigger attempt state changes.

---

## 7. Diagnostic Activity Rule: lastActivityAt (P03T3PR1-004 Reconciled)

### 7.1 Immutable Diagnostic Fact
```text
lastActivityAt = DIAGNOSTIC_ACTIVITY_EVIDENCE
lastActivityAt != heartbeat
lastActivityAt != attempt_proof
```
- Pinned AO establishes that `lastActivityAt` is the timestamp of the latest hook callback received by AO.
- **What it PROVES**: A tool execution, prompt submission, or hook event occurred at or before this timestamp.
- **What it DOES NOT PROVE**:
  1. It is NOT cryptographic proof.
  2. It does NOT prove dispatch acknowledgement.
  3. It does NOT prove that `ActivityActive` was ever observed.
  4. It does NOT prove that a specific immutable `TaskContract` was executed.
- In pinned lifecycle code (`backend/internal/lifecycle/manager.go`), the first arriving signal for a launch is explicitly permitted to be `idle` if the `active` hook delivery was lost or omitted.
- Therefore, the comparison `lastActivityAt > attempt.started_at` **CANNOT by itself authorize** `DISPATCHED -> RUNNING` or any turn completion transition.

### 7.2 Missed Active Window Re-evaluation
Without parsing the `WorkerReport` (which is strictly forbidden in P03), the supervisor cannot prove that an unobserved `idle` session completed the specific contract of the current attempt.
```text
MISSED_ACTIVE_WINDOW = UNRESOLVED_DECISION
```
`GIẢ ĐỊNH`: The safe fail-closed behavior on restart when `state = DISPATCHED` and AO reports `idle` is:
- Mark task as `FAILED` (reason: `MISSED_ACTIVE_WINDOW_UNVERIFIED`).
- Escalate to human or failure recovery handler.
- Under NO circumstances should the supervisor fabricate an unobserved `RUNNING` history.

---

## 8. REPORT_READY Ownership & Phase Boundaries (P03T3PR1-005 Reconciled)

### 8.1 Strict Phase Division
```text
AO IDLE = TURN COMPLETION OBSERVATION
AO IDLE != REPORT_READY
```
Canonical `REPORT_READY` transition requires:
1. Bounded retrieval of the raw report artifact (`TASK-P03-004`).
2. JSON deserialization of `WorkerReport`.
3. Schema validation against `worker-report.schema.json`.
4. Semantic validation that `report.task_id == task.id` and `report.attempt_id == attempt.id`.
5. Extraction of `WorkerClaim`.

### 8.2 TASK-P03-003 Hard Boundaries
TASK-P03-003 is strictly confined to **lifecycle observation and state reconciliation**.
TASK-P03-003 **MUST NOT**:
- Fetch report artifacts from workspace (`TASK-P03-004`).
- Parse or validate `WorkerReport` (`P04`).
- Write `worker_report_raw` (`P04`).
- Transition `RUNNING -> REPORT_READY` (`P04`).
- Transition inferred `DISPATCHED -> RUNNING -> REPORT_READY`.

### 8.3 State Consistency in RUNNING
When AO reports `ActivityIdle`, TASK-P03-003 records an internal observation event:
```text
Observation = AO_TURN_COMPLETED (ActivityIdle observed)
```
The Task **remains in `RUNNING`**.
`RUNNING` represents that the attempt execution phase is active, which includes the handoff to the Evidence Collector.
Promotion to `REPORT_READY` occurs **exclusively in P04** when the report has been successfully retrieved, parsed, and validated.

---

## 9. AO Activity State Mapping (P03T3PR1-006 Reconciled)

Pinned AO authority (`backend/internal/domain/activity.go`) defines distinct semantics for interactive states:
- `waiting_input`: Agent paused at empty prompt awaiting next instruction. Safe to message/nudge.
- `blocked`: Agent paused on a tool-permission or approval dialog. Automated senders **must never** inject input.

These states demand distinct normalized observation classifications:
```text
AO_WAITING_INPUT
AO_BLOCKED_DECISION
```

### 9.1 Mapping Evaluations

1. **`AO_WAITING_INPUT`**:
   - In an autonomous lane, worker should not pause for conversational input.
   - *Option A*: `RUNNING -> BLOCKED` (reason: `UNEXPECTED_INPUT_PROMPT`).
   - *Option B*: Remain in `RUNNING`, notify operator via telemetry.
   - `GIẢ ĐỊNH`: Marked as `UNRESOLVED_DECISION` pending External Supervisor policy decision.
2. **`AO_BLOCKED_DECISION`**:
   - Agent is stopped on a permission dialog.
   - Automated keystrokes are prohibited by pinned AO contract.
   - Canonical transition `RUNNING -> BLOCKED` exists in `state_machine.go`.
   - `GIẢ ĐỊNH`: Transition `RUNNING -> BLOCKED` (reason: `TOOL_PERMISSION_BLOCKED`), followed canonically by `BLOCKED -> HUMAN_REQUIRED`.
3. **`ActivityExited`**:
   - Pinned AO source explicitly establishes: `ActivityExited` **does not mean** the session is terminated. The managed terminal and worktree remain alive and inspectable.
   - **Rule**: `ActivityExited` MUST NOT automatically trigger worker crash or `FAILED`.
   - `ActivityExited` is a process-level observation; `isTerminated == true` is the session-level terminal state.

---

## 10. Fail-Closed Termination Classification (P03T3PR1-007 Reconciled)

Restoring the approved contract from `PROPOSAL-P03-001`:

```text
1. WORKER_STOPPED:
   Positive intentional-stop provenance successfully correlated to the current execution generation.

2. WORKER_CRASHED:
   Unexpected termination corroborated by positive observable failure evidence (e.g. exit code != 0, kernel signal, reaper death log).

3. WORKER_TERMINATION_UNKNOWN:
   Termination is observed (isTerminated: true), but evidence cannot safely distinguish between crash, external SIGKILL, or unrecorded stop.
```

- **Fail-Closed Principle**: The absence of intentional stop provenance proves **only** `INTENTIONAL_STOP_NOT_PROVEN`. It does NOT prove a crash.
- If no positive crash evidence exists, the classification MUST be `WORKER_TERMINATION_UNKNOWN`.
- In both `WORKER_CRASHED` and `WORKER_TERMINATION_UNKNOWN`, the Task transitions canonically `RUNNING -> FAILED` (or `DISPATCHED -> FAILED`), but the audit payload honestly records `WORKER_TERMINATION_UNKNOWN`.

---

## 11. Generation-Safe Stop Provenance (P03T3PR1-008 Reconciled)

A single static event `WORKER_STOP_REQUESTED` is not generation-safe:
- The HTTP kill call may fail after the request event was written.
- The session may later be restored, inheriting stale stop intent.

### 11.1 Stop Operation Lifecycle
Stop provenance is modeled as a multi-stage operation lifecycle:
1. `STOP_REQUESTED`: Written by orchestration layer prior to transport call.
2. `STOP_CALL_SUCCEEDED` / `STOP_CALL_FAILED`: Written immediately upon receiving HTTP response from AO.
3. `STOP_TERMINATION_CONFIRMED`: Written when observation poller confirms `isTerminated: true`.

### 11.2 Provenance Data Contract
Every stop provenance record must carry:
```json
{
  "operation_id": "op-uuid",
  "pair_id": "pair-123",
  "task_id": "task-456",
  "contract_id": "contract-789",
  "attempt_id": "attempt-001",
  "session_id": "sess-abc",
  "terminal_generation": "launch-gen-1",
  "timestamp": "2026-09-22T12:00:00Z",
  "actor": "SUPERVISOR_OPERATOR"
}
```
- **Generation Safety**: When observation evaluates `isTerminated: true`, it verifies `terminal_generation == attempt.terminal_generation`. A later restore with a new generation will NOT match the old stop provenance.
- `TaskAttempt.ended_at` and `worker_sessions.ended_at` MUST NOT be set when `STOP_REQUESTED` is written; they are updated only upon confirmed termination.

---

## 12. Attempt End Ownership & Atomicity (P03T3PR1-009 Reconciled)

### 12.1 Phase Ownership
- Proposed method `RecordAttemptEnd(..., rawReport)` is **REMOVED** from TASK-P03-003. `rawReport` belongs strictly to P04.
- `ActivityIdle` alone **MUST NOT** set `ended_at`.

### 12.2 Transaction Atomicity
For terminal transitions owned by TASK-P03-003 (`RUNNING -> FAILED`, `DISPATCHED -> FAILED`, `RUNNING -> BLOCKED`), state mutation and attempt termination must not be crash-separable.
The StateStore must provide an atomic transaction:
```text
Atomic Terminal Transaction:
1. Task state Compare-And-Set (e.g. RUNNING -> FAILED).
2. TaskAttempt ended_at mutation (SET ended_at = now).
3. Audit event append (TASK_STATE_TRANSITION with full provenance).
```
For the normal success path, `TaskAttempt.ended_at` is committed atomically by P04 when the task transitions `RUNNING -> REPORT_READY`.

### 12.3 Canonical State Machine Alignment
In canonical `internal/workflow/state_machine.go`, transitions out of `DISPATCHED` and `RUNNING` to `CANCELLED` **DO NOT EXIST**:
- Allowed from `DISPATCHED`: `RUNNING`, `FAILED`.
- Allowed from `RUNNING`: `REPORT_READY`, `FAILED`, `BLOCKED`.
- `CANCELLED` is reachable ONLY from `DRAFT`, `READY`, and `HUMAN_REQUIRED`.
Therefore, an operator stop of an active attempt results in `FAILED` with trigger `INTENTIONAL_STOP_BY_SUPERVISOR`, which can subsequently be cancelled from `HUMAN_REQUIRED` if desired. All claims of `DISPATCHED/RUNNING -> CANCELLED` are removed.

---

## 13. Removal of Invented Retry Policies (P03T3PR1-010 Reconciled)

- **Removed Terms**: "retry up to max attempts", "if retries exhausted", "recovery threshold", fixed retry counts, fixed backoff delays.
- There are **ZERO** approved numeric retry policies in the Supervisor governance baseline.
- **AO Daemon Unavailable Behavior**:
  - Transport errors (`connection refused`, `5xx`, timeout) are logged as diagnostic observation errors.
  - The TaskState is **PRESERVED** in `DISPATCHED` or `RUNNING`.
  - The supervisor poller does not implement automatic infinite retries or arbitrary retry caps.
  - If caller-injected `SUPERVISOR_EXECUTION_DEADLINE` expires while AO is unavailable, the caller-owned deadline produces an unrecoverable timeout failure.

---

## 14. Revised Comprehensive Lifecycle Recovery Matrix

| Persisted Task State | Authoritative AO Snapshot | Observed AO Facts | What Facts PROVE | What Facts DO NOT Prove | Generation Requirement | Normalized Observation Classification | TaskState Transition | Attempt ended_at Action | Required Positive Provenance | Audit Evidence | Policy Dependency | Resolution Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `DISPATCHED` | `active` | Session active | Agent tool execution active | Does not prove specific contract completed | Current generation matches attempt | `AO_WORKER_ACTIVE` | `DISPATCHED -> RUNNING` | Do NOT set | AO snapshot `active` + generation match | `TASK_STATE_TRANSITION` (`DISPATCHED -> RUNNING`) | Caller context | **DECIDED** |
| `DISPATCHED` | `idle` | Session idle post-dispatch | Agent prompt loop is at rest | Does NOT prove contract was executed | N/A | `AO_TURN_COMPLETED_UNOBSERVED` | `DISPATCHED -> FAILED` | Set `ended_at = now` | None | `TASK_STATE_TRANSITION` (`MISSED_ACTIVE_WINDOW_UNVERIFIED`) | Fail-closed policy | `MISSED_ACTIVE_WINDOW = UNRESOLVED_DECISION` |
| `DISPATCHED` | `waiting_input` | Agent waiting prompt | Agent paused on input | Does not prove failure | Current generation matches attempt | `AO_WAITING_INPUT` | `UNRESOLVED_DECISION` (Hold in `DISPATCHED` or `FAILED`) | Do NOT set | AO snapshot `waiting_input` | Diagnostic observation event | Operator policy | **UNRESOLVED_DECISION** |
| `DISPATCHED` | `blocked` | Agent blocked on decision | Tool permission dialog pending | Does not prove crash | Current generation matches attempt | `AO_BLOCKED_DECISION` | `DISPATCHED -> FAILED` (or hold) | Set `ended_at = now` if failed | AO snapshot `blocked` | `TASK_STATE_TRANSITION` | Operator policy | **UNRESOLVED_DECISION** |
| `DISPATCHED` | `exited` | Process exited | Terminal process ended | Does not prove session terminated | Current generation matches attempt | `AO_PROCESS_EXITED` | `DISPATCHED -> FAILED` | Set `ended_at = now` | AO snapshot `exited` | `TASK_STATE_TRANSITION` (`WORKER_PROCESS_EXITED`) | Fail-closed | **DECIDED** |
| `DISPATCHED` | `isTerminated: true` | Session terminated | AO session is dead | Does not prove intentional vs crash without provenance | Matched generation | `WORKER_STOPPED` (if stop confirmed) or `WORKER_TERMINATION_UNKNOWN` | `DISPATCHED -> FAILED` | Set `ended_at = now` | `STOP_TERMINATION_CONFIRMED` or none | `TASK_STATE_TRANSITION` (`INTENTIONAL_STOP` or `TERMINATION_UNKNOWN`) | FR-006 | **DECIDED** |
| `DISPATCHED` | AO 404 Not Found | Session missing | Session purged or deleted | Does not prove why deleted | N/A | `AO_SESSION_NOT_FOUND` | `DISPATCHED -> FAILED` | Set `ended_at = now` | Typed 404 APIError | `TASK_STATE_TRANSITION` (`SESSION_NOT_FOUND`) | NFR-005 | **DECIDED** |
| `DISPATCHED` | AO Unavailable | Daemon unreachable | Transport failure | Does not prove worker state | N/A | `AO_DAEMON_UNAVAILABLE` | **NO TRANSITION** (preserve state) | Do NOT set | Transport network error | Warning diagnostic log | Execution deadline | **DECIDED** |
| `RUNNING` | `active` | Session active | Agent tool execution active | Does not prove completion | Current generation matches attempt | `AO_WORKER_ACTIVE` | **NO TRANSITION** (remain in `RUNNING`) | Do NOT set | Snapshot `active` within deadline | Telemetry diagnostic log | Execution deadline | **DECIDED** |
| `RUNNING` | `idle` | Session idle | Agent completed execution turn | Does NOT prove report is valid (`AO_IDLE != REPORT_READY`) | Current generation matches attempt | `AO_TURN_COMPLETED` | **NO TRANSITION** (remain in `RUNNING` for P04 handoff) | Do NOT set in P03 (P04 sets on `REPORT_READY`) | Snapshot `idle` + generation match | `WORKER_TURN_COMPLETED` observation log | ADR-011 | **DECIDED** |
| `RUNNING` | `waiting_input` | Agent waiting prompt | Paused awaiting instruction | Does not prove failure | Current generation matches attempt | `AO_WAITING_INPUT` | `UNRESOLVED_DECISION` (Evaluate `RUNNING -> BLOCKED`) | Do NOT set | Snapshot `waiting_input` | Diagnostic log | Operator policy | **UNRESOLVED_DECISION** |
| `RUNNING` | `blocked` | Agent blocked on decision | Stopped on permission dialog | Automated keystrokes prohibited | Current generation matches attempt | `AO_BLOCKED_DECISION` | `RUNNING -> BLOCKED` | Set `ended_at = now` atomically | Snapshot `blocked` + generation match | `TASK_STATE_TRANSITION` (`RUNNING -> BLOCKED`) | Pinned AO `activity.go` | **DECIDED** |
| `RUNNING` | `exited` | Process exited | Process died mid-run | Does not prove session terminated | Current generation matches attempt | `AO_PROCESS_EXITED` | `RUNNING -> FAILED` | Set `ended_at = now` atomically | Snapshot `exited` | `TASK_STATE_TRANSITION` (`WORKER_PROCESS_EXITED`) | Fail-closed | **DECIDED** |
| `RUNNING` | `isTerminated: true` | Session terminated | Session dead | Does not prove crash without evidence | Matched generation | `WORKER_STOPPED` (if confirmed) or `WORKER_TERMINATION_UNKNOWN` | `RUNNING -> FAILED` | Set `ended_at = now` atomically | `STOP_TERMINATION_CONFIRMED` or none | `TASK_STATE_TRANSITION` (`INTENTIONAL_STOP` or `TERMINATION_UNKNOWN`) | FR-006 | **DECIDED** |
| `RUNNING` | AO 404 Not Found | Session missing | Session purged or deleted | Does not prove why deleted | N/A | `AO_SESSION_NOT_FOUND` | `RUNNING -> FAILED` | Set `ended_at = now` atomically | Typed 404 APIError | `TASK_STATE_TRANSITION` (`SESSION_NOT_FOUND`) | NFR-005 | **DECIDED** |
| `RUNNING` | AO Unavailable | Daemon unreachable | Transport failure | Does not prove worker state | N/A | `AO_DAEMON_UNAVAILABLE` | **NO TRANSITION** (preserve state) | Do NOT set | Transport network error | Warning diagnostic log | Execution deadline | **DECIDED** |

---

## 15. Session Binding Recovery Matrix for Crash Timing (Section 16 Reconciled)

| Crash Timing Point | Durable Local State | Possible AO Upstream State | Known Session Identity? | Known Execution Generation? | Safe Recovery Action | Deterministic Recovery Possible? | Human Intervention Required? | Current Pinned Public AO API Sufficient? |
|---|---|---|---|---|---|---|---|---|
| **Crash before `PrepareDispatch`** | Task `READY`, zero attempt allocated | None (no AO call made) | N/A | N/A | None required; task remains in `READY` for normal dispatch | **YES** | NO | YES |
| **Crash after `PrepareDispatch` before AO spawn** | Task `DISPATCHED`, Attempt 1 open (`ended_at = NULL`), `session_id = NULL` | None | NO | NO | Scan detects open attempt with `session_id = NULL`. Fail attempt to `FAILED` or retry spawn | **YES** | NO | YES |
| **Crash while AO spawn request in flight** | Task `DISPATCHED`, Attempt 1 open, `session_id = NULL` | Unknown: spawn may have succeeded or failed | NO | NO | Supervisor cannot query AO without `session_id`. Mark `FAILED` (`SPAWN_IN_FLIGHT_CRASH`) | **NO** (leaks orphaned AO session if created) | YES (to clean orphan session) | **NO** (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`) |
| **Crash after AO creates session before response received** | Task `DISPATCHED`, Attempt 1 open, `session_id = NULL` | Session created in AO, worktree created | NO | NO | Cannot rediscover session ID deterministically. Mark `FAILED` | **NO** (orphaned AO session exists) | YES | **NO** (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`) |
| **Crash after response received before local binding commit** | Task `DISPATCHED`, Attempt 1 open, `session_id = NULL` (response in volatile memory lost) | Session created in AO | NO | NO | Same as above: session ID lost. Mark `FAILED` | **NO** (orphaned session) | YES | **NO** (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`) |
| **Crash after session binding before `/send`** | Task `DISPATCHED`, Attempt 1 open, `session_id` and `generation` durable | Session exists, `ActivityIdle` (pre-prompt) | **YES** | **YES** | On restart, observe session. Verify `ActivityIdle` with zero prompt bytes. Dispatch contract via `/send` | **YES** | NO | YES |
| **Crash while `/send` request in flight** | Task `DISPATCHED`, Attempt 1 open, `session_id` durable | Unknown: worker may or may not have received message | **YES** | **YES** | Query AO: check `latestUserPrompt` or `activity`. If active, transition `RUNNING`. If unacknowledged, fail-closed | **PARTIAL** (depends on AO `latestUserPrompt` inspection) | If unacknowledged | **PARTIAL** |
| **Crash after `/send` 200 before first lifecycle poll** | Task `DISPATCHED`, Attempt 1 open, `session_id` durable | Worker executing (`active`) or turn completed (`idle`) | **YES** | **YES** | On restart, query `GET /api/v1/sessions/{id}`. If active, transition `RUNNING`. If idle, see Missed Active Window | **YES** | If missed active window | YES |

---

## 16. Architecture Impact & ADR Verdict (Section 17 Reconciled)

```text
P03_ARCHITECTURE_CHANGE = YES
P03_ADR_REQUIRED = YES
ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET
```
- **Rationale for Architecture Change**: The proposal amends canonical domain relationships (`Pair 1 o-- 1 WorkerSession`), defines execution generation attempt fencing (`terminal_generation`), specifies durable attempt session bindings, and restructures restart recovery boundaries across `RUNNING` tasks.
- **ADR Authorization**: An Architecture Decision Record (`ADR-016`) is mandatory, but drafting is strictly held until this Proposal Revision 1 is reviewed and approved by the External Supervisor.

---

## 17. Operational Policies Inventory (Section 19 Reconciled)

The eight (8) Supervisor operational policies remain strictly:
```text
SUPERVISOR_HTTP_TIMEOUT = UNSET
SUPERVISOR_HEALTH_PROBE_TIMEOUT = UNSET
SUPERVISOR_SPAWN_TIMEOUT = UNSET
SUPERVISOR_SEND_TIMEOUT = UNSET
SUPERVISOR_ACTIVITY_POLL_INTERVAL = UNSET
SUPERVISOR_EXECUTION_DEADLINE = UNSET
SUPERVISOR_KILL_STOP_TIMEOUT = UNSET
SUPERVISOR_WORKSPACE_READ_TIMEOUT = UNSET
```
Implementation code MUST NOT introduce hard-coded default numbers. Constructors must require caller-injected configurations with fail-fast validation (`<= 0` returns error).

---

## 18. Explicit Non-Goals

- NO workspace file fetching or report retrieval (`TASK-P03-004`).
- NO report JSON parsing or claim validation (`P04`).
- NO direct ConPTY or Agy CLI execution.
- NO SQLite database access inside `internal/ao`.
- NO invented numeric timeout or interval defaults.
- NO SSE `/api/v1/events` or CDC event bus implementation.
- NO production code implementation (`internal/**/*.go`).
- NO ADR drafting until formal proposal approval.

---

## 19. External Approval Gate

```text
PROPOSAL_P03_002 = REVISION_1_PENDING_EXTERNAL_REAUDIT
P03_ARCHITECTURE_CHANGE = YES
P03_ADR_REQUIRED = YES
ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET
TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT
```

Execution is halted awaiting independent External Supervisor re-audit of Proposal Revision 1.
