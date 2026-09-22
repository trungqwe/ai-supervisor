# PROPOSAL-P03-002: Lifecycle Reconciliation, Durable Dispatch Saga, and Session Binding Specification

> **Proposal ID**: `PROPOSAL-P03-002`
> **Revision**: `5`
> **Status**: `REVISION_5_PENDING_EXTERNAL_REAUDIT`
> **Task Binding**: `TASK-P03-003` (Lifecycle Reconciliation and State Transitions)
> **Baseline Commit**: `38154d7fa65888cbf25b0bc5206384198a9a8a40`
> **Pinned AO Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Architecture Classification**: `P03_ARCHITECTURE_CHANGE = YES`, `P03_ADR_REQUIRED = YES`, `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET`
> **Implementation Guard**: `TASK_P03_003 = NOT_RELEASED`, `P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION`
> **Active Gate**: `EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT_005`

---

## 1. Executive Summary & Problem Statement

This proposal establishes the architectural specifications, lifecycle reconciliation rules, durable dispatch saga mechanics, and restart recovery protocols for `TASK-P03-003` under pinned Agent Orchestrator (`AO`) v0.13.0 (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`).

Revision 5 surgically addresses and resolves the internal consistency findings of External Re-Audit `P03T3PR5-001`, `P03T3PR5-002`, and `P03T3PR5-003`:
1. **Quarantine Resolution Ledger Consistency (`P03T3PR5-001`)**: Formally establishes the semantic distinction between:
   - `PHYSICAL_EXECUTION_RESOLUTION`: Positive evidence that old execution can no longer continue (e.g. old generation confirmed terminated via `POST /api/v1/sessions/{sessionId}/kill` followed by authoritative `isTerminated == true` public observation).
   - `ADMINISTRATIVE_RISK_RESOLUTION`: Operator explicitly acknowledges the inability to prove physical termination and authorizes a governed replacement/recovery path. This MUST be durably audited and must NOT be reported as `TERMINATION_CONFIRMED` or `WORKER_STOPPED`.
   Reconciles D5 and D6 with Section 9 resolution classes so that if ADR-016 chooses automatic `/kill` reconciliation, quarantine clears strictly after confirmed termination of the old execution; if ADR-016 chooses Class B (authoritative absence) or Class C (human risk acceptance), the ADR defines resolution type, required evidence, operator authority, durable audit record, whether future dispatch is permitted, and preserves that old execution remains physically unresolved without collapsing into `TERMINATED`.
2. **TASK-P03-003 Scope Guard & Removal of Report Probing (`P03T3PR5-002`)**: Enforces strict task boundaries established in `PROPOSAL-P03-001`: TASK-P03-003 is strictly Lifecycle Observation & State Reconciliation; TASK-P03-004 owns raw `GetWorkspaceFile` transport; P04 owns WorkerReport semantic interpretation and evidence collection. Removes all candidate/recommended wording proposing that TASK-P03-003 probe report artifacts or workspace files in D8 (`MISSED_ACTIVE_WINDOW`). Confines D8 strictly to lifecycle-level choices (fail closed vs preserve DISPATCHED and escalate ambiguity vs persist ambiguity disposition and hand off downstream).
3. **BLOCKED / Attempt End Semantics Alignment (`P03T3PR5-003`)**: Separates the StateStore atomicity invariant from whether `BLOCKED` ends a `TaskAttempt`. Terminal failure transitions (`RUNNING/DISPATCHED -> FAILED`) atomically commit TaskState + `ended_at = now` + audit event append. For `RUNNING -> BLOCKED`, whether it terminates an attempt is `UNRESOLVED_FOR_ADR` and cross-referenced between D10 and D12: IF ADR-016 decides BLOCKED ends the attempt, transition + `ended_at = now` + audit persistence occur atomically; IF ADR-016 retains the open attempt, `ended_at` remains NULL; IF ADR-016 chooses no transition, attempt remains open. Canonical `BLOCKED` is not labeled as an unconditional terminal outcome.
4. **Preservation of All Prior Closures**: Strictly preserves `NEW_SESSION_OR_GENERATION != RESOLUTION_OF_PRIOR_UNCERTAIN_EXECUTION`, dispatch saga attribution gating on `SEND_CONFIRMED`, `SEND_REQUESTED` pre-effect intent semantics, zero blind resend, zero direct `DISPATCHED -> HUMAN_REQUIRED`, wire route `/kill`, pre-spawn decoupling boundaries, valid pinned activity states, `REPORT_READY` ownership separation, V1 `agy` TUI generation scope, Model A/B/C persistence comparison, and `UNSET` operational policies.

---

## 2. Authority Hierarchy & Governance Baseline

All resolutions strictly adhere to the 9-level decision hierarchy defined in `docs/24_CHANGE_GOVERNANCE.md`:

| Authority Reference | Source Document | Binding Governance Principle |
|---|---|---|
| **FR-005** | `docs/02_REQUIREMENTS.md` | Worker Task Dispatch: Supervisor control plane is the sole dispatcher of tasks to workers. |
| **FR-006** | `docs/02_REQUIREMENTS.md` | Worker Lifecycle Observation: Supervisor tracks worker state, detects turn completion, failures, and termination. |
| **NFR-005** | `docs/02_REQUIREMENTS.md` | Upstream Decoupling: Anti-corruption layer isolates Supervisor core from upstream AO internals. Note: NFR-005 establishes architectural isolation; it is **NOT** authority for TaskState failure transitions. |
| **REC-005** | `docs/14_FAILURE_RECOVERY.md` | Task recovery timing and open attempt reconciliation. |
| **REC-014** | `docs/14_FAILURE_RECOVERY.md` | Upstream session loss recovery and terminal outcome assignment. |
| **ADR-002** | `docs/adr/ADR-002*` | AO is the execution control plane; Supervisor is the governance control plane. |
| **ADR-011** | `docs/adr/ADR-011*` | `AO_IDLE != REPORT_READY`. Turn completion observation precedes evidence collection and report ingestion. |
| **ADR-012** | `docs/adr/ADR-012*` | `TaskAttempt` binds monotonically to `TaskContract` revision; atomic pre-dispatch commit invariant. |
| **ADR-015** | `docs/adr/ADR-015*` | Local SQLite StateStore with write transactions and append-only hash-chained audit log. |

---

## 3. Pinned AO Authority & API Contract Facts

Pinned Upstream Repository: `Untrivial-ai/agent-orchestrator` v0.13.0, commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.

### 3.1 Pinned Endpoints & Wire Contracts
1. **Spawn Session (`POST /api/v1/sessions`)**:
   - Accepts: `projectId`, `issueId`, `trackerProvider`, `kind`, `harness`, `branch`, `mode`, `prompt`, `model`, `displayName`, `attachments`.
   - Returns: `SpawnSessionResponse` (`id`, `workspacePath`, `promptBytes`, `activity`).
   - Fact: `PINNED_SPAWN_IDEMPOTENCY = ABSENT`.
2. **Send Session Message (`POST /api/v1/sessions/{id}/send`)**:
   - Accepts: `SendSessionMessageRequest` (`message`, `attachment`).
   - Returns: `SendSessionMessageResponse` (`ok`, `sessionId`, `message`).
   - Fact: Contains NO `idempotencyKey`, `clientMessageId`, caller operation ID, or correlation ID.
   - Fact: `PINNED_SEND_IDEMPOTENCY = ABSENT`.
   - Classification: `P03T3_SEND_RECOVERY = UNRESOLVED_BLOCKER`.
3. **Stop Worker / Kill Session (`POST /api/v1/sessions/{sessionId}/kill`)**:
   - Pinned route: `backend/internal/httpd/controllers/sessions.go:206` (`r.Post("/sessions/{sessionId}/kill", c.kill)`).
   - Approved Supervisor adapter: `internal/ao/session_commands.go:StopWorker`.
   - Returns: `KillSessionResponse{OK: true, SessionID, Freed: bool}`.
   - Fact: The wire contract is HTTP `POST .../kill`. There is **NO** `DELETE` endpoint for stopping workers.
4. **Get Session (`GET /api/v1/sessions/{id}`)**:
   - Returns: `SessionView` (`domain.Session`, `branch`, `terminalGeneration`, `previewUrl`, `previewRevision`, `model`, `lastUserMessageAt`, `prs`, `activeAgentSwitch`).
   - Fact: `promptBytes` is ephemeral to spawn and absent from `SessionView`.
   - Fact: `latestUserPrompt` is an internal domain field and absent from `SessionView`. `lastUserMessageAt` records real user directions and explicitly excludes automated updates.
   - Fact: Public `SessionView` does **NOT** expose process exit codes, OS signals, or reaper death logs (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`, `WORKER_CRASHED_PUBLIC_BASELINE = NOT_CURRENTLY_PROVABLE`).

### 3.2 Pinned Activity States & SessionGuard Protection
1. **Canonical Pinned Activity States**:
   - In `backend/internal/domain/activity.go`, the valid activity states are strictly:
     `active`, `idle`, `waiting_input`, `blocked`, `exited`.
   - Any other terminology is invalid under pinned AO.
2. **SessionGuard Invariant (`backend/internal/sessionguard/guard.go`)**:
   - AO enforces pane-write admission guard before any message delivery:
     - `SuppressedTerminated`: Session is dead; write refused.
     - `SuppressedExited`: Agent process exited; pane may be shell input; write refused.
     - `SuppressedAwaitingUser`: Blocked on permission dialog; automated write refused.
     - `SuppressedBusy`: Session mid-turn; write refused.

### 3.3 Harness & Generation Fence Alignment
1. **Chat Driver Registry (`backend/internal/adapters/chatdriver/registry/registry.go`)**:
   - Canonical V1 worker harness `agy` is **NOT** registered for Chat mode.
2. **Spawn Fallback (`backend/internal/session_manager/manager.go:886`)**:
   - Spawning `agy` in Chat mode returns `ports.ErrChatUnsupported` and falls back cleanly to `SessionModeTUI`.
3. **Execution Fence**:
   - In `SessionModeTUI`, AO sets `terminalGeneration = s.Metadata.RuntimeLaunchID`.
   - Baseline fence: `AGY_V1_EXECUTION_GENERATION = terminalGeneration / RuntimeLaunchID`.
   - Generic/Chat harness epoch: `GENERIC_EXECUTION_EPOCH = UNRESOLVED_DECISION`.

---

## 4. Spawn Crash Window & Pre-Provisioning Boundary

### 4.1 The Spawn Crash Window
```text
T0: Prepare session spawn request.
T1: Transmit POST /api/v1/sessions to AO.
T2: AO creates session upstream, allocates worktree/ConPTY, returns HTTP 201 Created.
T3: Supervisor commits session_id to local StateStore.
```
- A crash between **T2 and T3** leaves an active AO session upstream with its generated `session_id` lost to the Supervisor.
- Because pinned AO lacks caller-provided session IDs or spawn idempotency keys:
```text
PINNED_SPAWN_IDEMPOTENCY = ABSENT
P03T3_SESSION_SPAWN_RECOVERY = UNRESOLVED_BLOCKER
```

### 4.2 Pre-Spawn Decoupling vs Deterministic Recovery
Pre-creating a `WorkerSession` on the `Pair` decouples task dispatch from session creation. However, this architectural separation does **NOT** eliminate the spawn crash window:
```text
PRESPAWN = TASK_DISPATCH_DECOUPLING_OPTION
PRESPAWN != DETERMINISTIC_SPAWN_RECOVERY
```
- Pre-provisioning shifts the crash window from task dispatch time to Pair provisioning time.
- If a crash occurs during Pair provisioning between T2 and T3, an orphaned AO session is still leaked upstream.
- ADR-016 must define how uncertain Pair-session provisioning operations are quarantined, audited, and reaped.

---

## 5. ADR-012 Conflict and Required Amendment

### 5.1 The Conflict
Accepted ADR-012 Section 10 states:
> *"External AO API calls (`createWorkerSession`, `sendTask`) occur strictly after the durable `DISPATCHED` record exists."*

If ADR-016 adopts pre-provisioning the Pair `WorkerSession` prior to task dispatch, calling `createWorkerSession` before `DISPATCHED` violates ADR-012 Section 10.

### 5.2 Required Amendment
ADR-016 must explicitly amend ADR-012 Section 10 by distinguishing:
1. **PAIR SESSION LIFECYCLE SIDE EFFECT (`createWorkerSession`)**:
   - Provisions or reuses the execution sandbox/worktree for a Pair.
   - Pertains to lane provisioning, not task work consumption.
2. **TASK EXECUTION SIDE EFFECT (`sendTask` / `/send`)**:
   - Pushes the immutable task contract and prompt to the worker.
   - Strictly gated by prior `TaskAttempt` allocation and durable `DISPATCHED` commit.

**Exact Sentence Amendment Required in ADR-016**:
> *ADR-012 Section 10 is amended as follows:*
> "Pair worker session provisioning (`createWorkerSession`) MAY occur prior to task dispatch as a Pair-scoped lifecycle side effect, establishing a durable `WorkerSession` binding. However, task execution side effects (`sendTask` / `/send`) occur strictly *after* `TaskAttempt` allocation and the durable `DISPATCHED` state record exist in the State Store."

---

## 6. WorkerSession Domain Model & Coherent Persistence Models

Canonical `docs/05_DOMAIN_MODEL.md` specifies `Pair "1" o-- "1" WorkerSession`.

### 6.1 Evaluation of Persistence Models

| Dimension | Model A: Mutable Current Session + Plain Attempt Snapshot | Model B: Historical WorkerSession Table + Active Pointer | Model C: Pair Current Session + AttemptSessionBinding Join Table |
|---|---|---|---|
| **Description** | `worker_sessions` stores current active session keyed by `pair_id` (`UNIQUE`). `task_attempts` stores immutable execution snapshot columns (`session_id TEXT`, `terminal_generation TEXT`) without an FK to `worker_sessions`. | `worker_sessions` is an append-only log of all spawned sessions. `pairs` table has `active_worker_session_id` FK. `task_attempts.session_id` has an FK to `worker_sessions`. | `worker_sessions` remains 1-to-1 with `pairs`. An immutable join table `attempt_session_bindings` records historical mappings (`attempt_id`, `session_id`, `terminal_generation`). |
| **Canonical Relationship Impact** | Perfectly preserves canonical `Pair 1 o-- 1 WorkerSession`. | Amends domain to `Pair 1 o-- * WorkerSession`. Introduces cyclic FKs (`pairs` <-> `worker_sessions`). | Preserves `Pair 1 o-- 1 WorkerSession`, but adds entity `AttemptSessionBinding`. |
| **Historical Integrity** | High: `task_attempts` retains exact historical snapshot permanently. | High: full session lifecycle history preserved in session table. | High: full binding history preserved in dedicated binding table. |
| **Session Replacement** | Trivial: UPDATE `worker_sessions` in place. Old attempts unaffected. | Moderate: INSERT new session row, UPDATE `pairs.active_worker_session_id`. | Trivial: UPDATE `worker_sessions`, INSERT new binding row. |
| **Stale Observation Protection** | Complete: Poller validates `snapshot.generation == attempt.terminal_generation`. | Complete: Poller validates generation against attempt or session. | Complete: Poller validates generation against binding. |

### 6.2 Recommendation (`GIẢ ĐỊNH`)
`GIẢ ĐỊNH`: **Model A** is recommended for ADR-016. It preserves canonical `Pair 1 o-- 1 WorkerSession`, avoids cyclic relational constraints, and guarantees permanent attempt auditability via self-contained snapshot columns.

---

## 7. Execution Epoch & Terminal Generation Scope

- `SESSION_ID != EXECUTION_EPOCH`: An AO session ID survives restore/restart, but the execution generation changes.
- In pinned AO, `terminalGeneration` maps directly to `s.Metadata.RuntimeLaunchID` in `SessionModeTUI`.
- Canonical V1 `agy` workers execute under `SessionModeTUI`.
- Specification:
```text
AGY_V1_EXECUTION_GENERATION = terminalGeneration / RuntimeLaunchID
GENERIC_EXECUTION_EPOCH = UNRESOLVED_DECISION
```

---

## 8. Pinned /send Idempotency & Durable Dispatch Saga Stages

### 8.1 Pinned /send Idempotency Fact
- Inspection of `SendSessionMessageRequest` (`dto.go:838`) confirms: contains only `Message` and optional `Attachment`. Contains **NO** `idempotencyKey`, `clientMessageId`, or correlation ID.
- Upstream Record: `PINNED_SEND_IDEMPOTENCY = ABSENT`.
- Classification: `P03T3_SEND_RECOVERY = UNRESOLVED_BLOCKER`.
- Blind resend risks injecting duplicate instructions into an active agent process. **Blind resend is strictly prohibited**.

### 8.2 Durable Dispatch Stages & Evidence Boundaries
The dispatch saga models three durable stages:

```text
1. DISPATCH_BOUND:
   - Task = DISPATCHED, TaskAttempt allocated, session_id and terminal_generation bound.
   - Network /send has NOT been initiated.
   - Evidence Proven: Local attempt and session binding exist. Zero dispatch network I/O.

2. SEND_REQUESTED:
   - Durable dispatch-operation record flushed to disk immediately BEFORE calling POST /api/v1/sessions/{id}/send.
   - Evidence Boundary: Proves SEND_INTENT_DURABLY_RECORDED / EXTERNAL_SEND_MAY_HAVE_BEEN_ATTEMPTED.
   - Critical Limit: Does NOT prove that the request was transmitted to the network. A crash may occur after database commit but before network I/O.
   - Delivery Outcome: UNKNOWN.

3. SEND_CONFIRMED:
   - Durable dispatch-operation record flushed to disk immediately AFTER receiving HTTP 200 from AO.
   - Evidence Proven: AO received and acknowledged the task prompt for this attempt.
```

### 8.3 Dispatch Saga Gating Invariant
```text
LIFECYCLE ACTIVITY CANNOT BE ATTRIBUTED TO CURRENT ATTEMPT UNTIL SEND_CONFIRMED
```
- **`DISPATCH_BOUND`**: Any AO activity observed (e.g. `active` or `idle`) reflects pre-existing session state, not current task execution. TaskState **MUST NOT** transition.
- **`SEND_REQUESTED`**: Because delivery outcome is `UNKNOWN`, observed AO activity cannot safely be attributed to the current attempt. TaskState **MUST NOT** transition to `RUNNING`.
- **`SEND_CONFIRMED`**: Only after `SEND_CONFIRMED` is committed may AO activity be interpreted as evidence of current attempt execution (provided `session_id` and execution generation also match).

---

## 9. Uncertain Delivery Retry Quarantine & Pre-Dispatch Guard (P03T3PR4-001 Reconciled)

### 9.1 The Re-Execution Hazard
When a crash occurs at `SEND_REQUESTED` without `SEND_CONFIRMED`:
- `DELIVERY_OUTCOME = UNKNOWN`.
- The task contract may already be executing upstream in AO.
- In canonical `internal/workflow/state_machine.go`:
  - `DISPATCHED` can only transition to: `RUNNING` or `FAILED`.
  - A direct transition from `DISPATCHED` to `HUMAN_REQUIRED` does **NOT** exist in the workflow graph.
- If the Supervisor transitions `DISPATCHED -> FAILED`, canonical workflow later permits:
  - `FAILED -> READY` (for retry), and
  - `FAILED -> HUMAN_REQUIRED` (for human escalation).
- If a task is transitioned `FAILED -> READY` while an uncertain execution is still running upstream, an automated retry would dispatch a duplicate task execution into the same session.

### 9.2 The Quarantine Invariant & Old Execution Resolution Boundary
```text
NEW_SESSION_OR_GENERATION != RESOLUTION_OF_PRIOR_UNCERTAIN_EXECUTION
UNCERTAIN_DELIVERY_QUARANTINE = MANDATORY
```
An attempt or session with `DELIVERY_OUTCOME = UNKNOWN` **MUST NOT** become eligible for automated retry (`PrepareDispatch`) or a new task send until the uncertainty is explicitly resolved.

A fresh session or generation provides a prospective execution target; it does **not** prove that the prior uncertain execution stopped running. Therefore, uncertain-delivery quarantine remains strictly bound to the original `(task_id, attempt_id, session_id, terminal_generation)` lineage until that prior execution is positively resolved.

#### Required Semantic Distinction:
- **`PHYSICAL_EXECUTION_RESOLUTION`**: Positive evidence that old execution can no longer continue.
  *Examples*: Old generation confirmed terminated by authoritative public AO observation (`isTerminated == true`) following an intentional stop operation (`POST /api/v1/sessions/{sessionId}/kill`); or another ADR-approved authoritative physical resolution.
- **`ADMINISTRATIVE_RISK_RESOLUTION`**: Operator explicitly acknowledges the inability to prove physical termination and authorizes a governed replacement/recovery path.
  *Audit & Truth Mandate*: This MUST be durably recorded in the append-only audit trail. Do NOT report administrative acceptance as `TERMINATION_CONFIRMED`. Do NOT report `WORKER_STOPPED` unless intentional termination evidence actually proves it. The system state must honestly reflect that old execution remains physically unresolved.

#### Acceptable Resolution Classes:
1. **Class A: Old Execution Termination Confirmed (`PHYSICAL_EXECUTION_RESOLUTION`)**:
   The known old `(session_id, terminal_generation)` is positively confirmed terminated by authoritative public AO observation (`isTerminated == true`) following an intentional stop/cleanup operation (`POST /api/v1/sessions/{sessionId}/kill`). Only after confirmed termination of the old execution may the uncertainty be marked physically resolved, allowing subsequent replacement dispatch.
2. **Class B: Old Session Authoritatively Absent (`ADMINISTRATIVE_RISK_RESOLUTION` / Absence Disposition)**:
   The public AO contract returns an approved deterministic absence result (e.g. HTTP 404 Not Found on `GET /api/v1/sessions/{id}`). ADR-016 must decide whether absence alone is sufficient to retire uncertainty or whether human confirmation remains required. Absence alone must not be silently classified as a crash. If ADR-016 chooses Class B, the ADR must explicitly define the resolution type, required evidence, operator authority, durable audit record, whether future dispatch is permitted, and record that old execution remains physically unresolved without collapsing into `TERMINATED`.
3. **Class C: Human Risk Acceptance (`ADMINISTRATIVE_RISK_RESOLUTION`)**:
   An operator explicitly acknowledges that delivery may have occurred and old execution cannot be positively proven stopped. This decision is durable and auditable. ADR-016 determines whether this permits replacement dispatch and what cleanup/quarantine evidence is required. It MUST NOT be reported as `TERMINATION_CONFIRMED` or `WORKER_STOPPED`.
4. **Class D: Other Source-Backed / Canonical-Approved Resolution**:
   Permissible only if explicitly justified by canonical authority and ADR approval.

#### Prohibited Clear Conditions:
The following alone **MUST NOT** clear quarantine:
- Creation of a new session;
- Allocation of a new `terminalGeneration`;
- Retry attempt allocation;
- `FAILED -> READY` transition;
- Elapsed time / timers;
- Stale `lastActivityAt`;
- AO daemon restart.

### 9.3 Legal State Transition Paths & Semantic Flow
- Direct transition from `DISPATCHED` to `HUMAN_REQUIRED` is **non-existent and illegal** under canonical `state_machine.go`.
- Legal semantic flow:
  ```text
  SEND_REQUESTED without SEND_CONFIRMED
  -> DELIVERY_OUTCOME_UNKNOWN
  -> DISPATCHED -> FAILED (reason: UNCERTAIN_DELIVERY_CRASH)
  -> UNCERTAIN_DELIVERY_QUARANTINE (bound to original attempt lineage)
  -> FAILED -> HUMAN_REQUIRED
  -> Explicit reconciliation of OLD execution (Class A physical, or Class B/C administrative)
  -> QUARANTINE_RESOLVED (or ADMINISTRATIVE_RISK_ACCEPTED)
  -> Only then may a future retry path (e.g. HUMAN_REQUIRED -> DRAFT -> READY) be considered.
  ```
  Automated transition `FAILED -> READY` is strictly prohibited while quarantine is active (`resolution_state = UNRESOLVED`).

### 9.4 Pre-Dispatch Retry Guard Options Analysis (For ADR-016)

| Option | Persistence Owner | Restart Durability | How PrepareDispatch / Retry is Rejected | How Quarantine is Cleared | Audit Evidence | Stale Generation Protection |
|---|---|---|---|---|---|---|
| **Option A: Pair / WorkerSession Quarantine Metadata** | `worker_sessions` table (`quarantine_state TEXT`) | Durable across restarts | `PrepareDispatch` checks Pair's current session quarantine flag; rejects if quarantined | Operator or cleanup saga issues `/kill`, confirms termination, clears quarantine flag | `WORKER_SESSION_QUARANTINED` / `QUARANTINE_CLEARED` audit events | High: tied directly to the execution session |
| **Option B: Dispatch-Operation Unresolved Flag** | `dispatch_operations` table (`resolution_state TEXT`) | Durable across restarts | `PrepareDispatch` queries prior dispatch operations for task/pair; rejects if open uncertain record exists | Explicit reconciliation saga updates operation to `RECONCILED_TERMINATED` | `DISPATCH_OPERATION_RECONCILED` audit event | High: attempt-specific record |
| **Option C: Attempt-Level Recovery Disposition** | `task_attempts` table (`recovery_disposition TEXT`) | Durable across restarts | `PrepareDispatch` scans open/latest attempts; rejects if latest attempt disposition is `QUARANTINED` | Terminal reconciliation marks attempt resolved after upstream kill | `ATTEMPT_RECOVERY_DISPOSITION_SET` audit event | Moderate: requires attempt scan |
| **Option D: Combination (Attempt + Pair Active Lane Lockout)** | Both `task_attempts` and `worker_sessions` | Durable across restarts | Double-gated: lane locked and attempt quarantined; fail-closed at both levels | Coordinated cleanup: kill session, clear lane lock, finalize attempt | Comprehensive multi-entity audit trail | Complete: eliminates single point of failure |

`GIẢ ĐỊNH`: **Option D** is recommended for ADR-016 to guarantee fail-closed enforcement at both the task attempt and pair lane levels.

---

## 10. Pre-Send Activity Admissibility Contract

At `DISPATCH_BOUND`, the Supervisor has committed `TaskState = DISPATCHED`, allocated an open `TaskAttempt`, and bound the `session_id`. However, `/send` has not been invoked.
Before invoking `/send`, the Supervisor inspects current session activity via `GET /api/v1/sessions/{id}`.

### 10.1 Pre-Send Admissibility Matrix (`PRE_SEND_ADMISSIBILITY`)

| Observed AO Activity Snapshot | Upstream Semantics | Pre-Send Admissibility Status | Dispatch Action | Rationale | Resolution Status |
|---|---|---|---|---|---|
| **`idle`** | Worker session initialized, pane at clean prompt | **`SAFE_TO_SEND`** | Write `SEND_REQUESTED`, invoke `/send` | Expected normal baseline for dispatch | **DECIDED** |
| **`waiting_input`** | Agent process paused at prompt awaiting instruction | **`SAFE_TO_SEND`** | Write `SEND_REQUESTED`, invoke `/send` | Pinned AO permits message delivery when waiting for input | **DECIDED** |
| **`blocked`** | Agent stopped on tool permission or approval dialog | **`SEND_PROHIBITED`** | Do NOT send; halt dispatch; await unblock or fail attempt | Pinned SessionGuard prohibits writes into blocked dialogs (`SuppressedAwaitingUser`) | **DECIDED** |
| **`exited`** | Agent process exited; pane may be raw interactive shell | **`SEND_PROHIBITED`** | Do NOT send; require session restart/restore before send | SessionGuard suppresses writes (`SuppressedExited`); writing executes as shell script | **DECIDED** |
| **`isTerminated: true`** | Session terminated upstream | **`SEND_PROHIBITED`** | Do NOT send; transition `DISPATCHED -> FAILED` | Pane gone; SessionGuard returns `SuppressedTerminated` | **DECIDED** |
| **`active`** | Agent actively executing tools / processing turn | **`SEND_PROHIBITED`** | Do NOT send; quarantine dispatch; evaluate whether leftover turn | Supervisor knows current task was not sent; session is executing foreign turn | **UNRESOLVED_DECISION** (ADR-016) |
| **Generation Mismatch** | `terminalGeneration != attempt.terminal_generation` | **`SEND_PROHIBITED`** | Do NOT send; fail attempt (`STALE_EXECUTION_GENERATION`) | Session was restarted or swapped; execution fence broken | **DECIDED** |

---

## 11. Stop Provenance Wire Contract

### 11.1 Wire Route Alignment
- Pinned AO exposes `POST /api/v1/sessions/{sessionId}/kill` (`sessions.go:206`).
- Supervisor client implements `StopWorker` via `POST /api/v1/sessions/{sessionId}/kill` (`internal/ao/session_commands.go:172`).
- HTTP `DELETE` for session termination is **strictly non-existent**.

### 11.2 Generation-Safe Stop Lifecycle
1. `STOP_REQUESTED`: Written to audit log immediately **before** calling `POST /api/v1/sessions/{sessionId}/kill`.
2. `STOP_CALL_SUCCEEDED` / `STOP_CALL_FAILED`: Written immediately upon receiving HTTP response (`KillSessionResponse{OK: true, SessionID, Freed}`).
3. `STOP_TERMINATION_CONFIRMED`: Written **only** when a subsequent authoritative snapshot returns `isTerminated == true` with matching execution generation.
- **Invariant**: HTTP 200 on `/kill` proves only that the kill signal was accepted by AO; it does **not** prove process termination. Attempt `ended_at` is updated only upon confirmed termination.

---

## 12. Corrected Comprehensive Lifecycle Recovery Matrix

The evidence tuple for all lifecycle evaluations is:
`(TaskState, DispatchSagaStage, AO Observation, Execution Generation Match)`

Pinned AO `ActivityExited` denotes that an agent process exited while the managed session remains open. Under Section 11 of the directive: `ActivityExited alone -> NO AUTOMATIC TASK FAILURE`.

| TaskState | Dispatch Saga Stage | AO Public Observation | Execution Generation Match | What Facts PROVE | What Facts DO NOT Prove | TaskState Transition | Attempt ended_at Action | Quarantine Behavior | Resolution Status | ADR-016 Dependency |
|---|---|---|---|---|---|---|---|---|---|---|
| `DISPATCHED` | `DISPATCH_BOUND` | `idle` | Matches | Pre-send baseline ready | Does not prove task execution | **NO TRANSITION** (remain `DISPATCHED`) | Do NOT set | None | **DECIDED** | Pre-send send trigger |
| `DISPATCHED` | `DISPATCH_BOUND` | `active` | Matches | Session busy before send | Current task execution (prompt unsent) | **NO TRANSITION** (remain `DISPATCHED`) | Do NOT set | `PRE_SEND_BLOCKED` | **UNRESOLVED_DECISION** | D7 (Pre-send admissibility) |
| `DISPATCHED` | `DISPATCH_BOUND` | `waiting_input` | Matches | Session waiting for input | Current task execution | **NO TRANSITION** (remain `DISPATCHED`) | Do NOT set | None | **DECIDED** | Safe to send prompt |
| `DISPATCHED` | `DISPATCH_BOUND` | `blocked` | Matches | Session blocked on dialog | Current task execution | **NO TRANSITION** (remain `DISPATCHED`) | Do NOT set | `PRE_SEND_BLOCKED` | **DECIDED** | Send prohibited |
| `DISPATCHED` | `DISPATCH_BOUND` | `exited` | Matches | AO process exit signal observed | Session termination or crash | **NO AUTOMATIC TRANSITION** | Do NOT set | `PRE_SEND_BLOCKED` | **DECIDED** (observation rule) | Send prohibited |
| `DISPATCHED` | `DISPATCH_BOUND` | `isTerminated: true` | Matches | Session dead before send | Crash (cause unproven) | `DISPATCHED -> FAILED` | Set `ended_at = now` | None | **DECIDED** | Pre-send failure |
| `DISPATCHED` | `SEND_REQUESTED` | Any activity | Matches / Any | Send intent durably recorded; delivery unknown | Network delivery or task execution | `DISPATCHED -> FAILED` (Path to `HUMAN_REQUIRED`) | Set `ended_at = now` | **`UNCERTAIN_DELIVERY_QUARANTINE`** | **UNRESOLVED_DECISION** | D5, D6 (Quarantine guard) |
| `DISPATCHED` | `SEND_CONFIRMED` | `active` | Matches | Prompt confirmed; agent executing turn | Task completion | `DISPATCHED -> RUNNING` | Do NOT set | None | **DECIDED** | Standard execution entry |
| `DISPATCHED` | `SEND_CONFIRMED` | `idle` | Matches | Prompt confirmed; session idle | Whether prompt ran or was missed; does NOT authorize TASK-P03-003 report probing (report read owned by downstream TASK-P03-004 / P04) | `UNRESOLVED_DECISION` | Do NOT set in P03 | Potential quarantine | **UNRESOLVED_DECISION** | D8 (Missed Active Window lifecycle policy) |
| `DISPATCHED` | `SEND_CONFIRMED` | `waiting_input` | Matches | Prompt confirmed; agent paused | Failure or task completion | `UNRESOLVED_DECISION` | Do NOT set | None | **UNRESOLVED_DECISION** | D9 (waiting_input mapping) |
| `DISPATCHED` | `SEND_CONFIRMED` | `blocked` | Matches | Prompt confirmed; agent blocked on decision | TaskState BLOCKED | `UNRESOLVED_DECISION` | Do NOT set | None | **UNRESOLVED_DECISION** | D10 (blocked mapping) |
| `DISPATCHED` | `SEND_CONFIRMED` | `exited` | Matches | AO process exit signal observed | Session termination | **NO AUTOMATIC TRANSITION** | Do NOT set | None | **DECIDED** (observation rule) | Process exit rule |
| `DISPATCHED` | `SEND_CONFIRMED` | `isTerminated: true` | Matches | Session terminated | Crash without evidence | `DISPATCHED -> FAILED` | Set `ended_at = now` | None | **DECIDED** | Stop/unknown failure |
| `RUNNING` | `SEND_CONFIRMED` | `active` | Matches | Agent actively executing tools | Task completion | **NO TRANSITION** (remain `RUNNING`) | Do NOT set | None | **DECIDED** | Normal turn polling |
| `RUNNING` | `SEND_CONFIRMED` | `idle` | Matches | Agent turn completed | Report validity (`AO_IDLE != REPORT_READY`) | **NO TRANSITION** (remain `RUNNING` for P04) | Do NOT set in P03 (P04 sets) | None | **DECIDED** | ADR-011 turn observation |
| `RUNNING` | `SEND_CONFIRMED` | `waiting_input` | Matches | Agent paused awaiting input | Failure | `UNRESOLVED_DECISION` | Do NOT set | None | **UNRESOLVED_DECISION** | D9 (waiting_input mapping) |
| `RUNNING` | `SEND_CONFIRMED` | `blocked` | Matches | Agent blocked on permission dialog | TaskState BLOCKED or attempt termination (ADR policy) | `UNRESOLVED_DECISION` | UNRESOLVED_FOR_ADR (see D10/D12) | None | **UNRESOLVED_DECISION** | D10, D12 (blocked TaskState and attempt-end policy) |
| `RUNNING` | `SEND_CONFIRMED` | `exited` | Matches | AO process exit signal observed | Session termination | **NO AUTOMATIC TRANSITION** | Do NOT set | None | **DECIDED** (observation rule) | Process exit rule |
| `RUNNING` | `SEND_CONFIRMED` | `isTerminated: true` | Matches | Session terminated | Crash without evidence | `RUNNING -> FAILED` | Set `ended_at = now` | None | **DECIDED** | Stop/unknown failure |
| `RUNNING` | `SEND_CONFIRMED` | AO 404 Not Found | N/A | Session purged upstream | Purge reason | `RUNNING -> FAILED` | Set `ended_at = now` | None | **DECIDED** | REC-014 / Approved ADR |
| `RUNNING` | `SEND_CONFIRMED` | AO Unavailable | N/A | Daemon unreachable | Worker failure | **NO TRANSITION** (preserve `RUNNING`) | Do NOT set | None | **DECIDED** | Execution deadline |

---

## 13. REPORT_READY Boundary & Evidence Collector Separation

- **Invariant**: `AO_IDLE = TURN COMPLETION OBSERVATION`, `AO_IDLE != REPORT_READY`.
- TASK-P03-003 is strictly confined to lifecycle observation and state reconciliation.
- TASK-P03-003 **MUST NOT**:
  - Fetch report files from workspace worktrees (`TASK-P03-004`).
  - Parse or validate `WorkerReport` JSON schemas (`P04`).
  - Validate report claims against task contract boundaries (`P04`).
  - Write `worker_report_raw` to the StateStore (`P04`).
  - Transition the task to `REPORT_READY` (`P04`).
- Upon observing `ActivityIdle`, TASK-P03-003 leaves the task in `RUNNING`. Transition `RUNNING -> REPORT_READY` is owned exclusively by P04 Evidence Collector.

---

## 14. Public Crash Evidence Boundary

- In pinned AO, `GET /api/v1/sessions/{id}` exposes no process exit codes, signals, or reaper logs.
- Definition:
```text
PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE
WORKER_CRASHED_PUBLIC_BASELINE = NOT_CURRENTLY_PROVABLE
```
- Baseline classification:
  - `isTerminated == true` + matching intentional-stop provenance -> `WORKER_STOPPED`.
  - `isTerminated == true` without positive cause evidence -> `WORKER_TERMINATION_UNKNOWN` (Task transitions to `FAILED`, honestly logging `WORKER_TERMINATION_UNKNOWN`).
  - Administrative risk acceptance (`ADMINISTRATIVE_RISK_RESOLUTION`) MUST NEVER be reported as `WORKER_STOPPED` or `TERMINATION_CONFIRMED`.

---

## 15. Operational Policies & Zero Invented Numbers

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

## 16. ADR-016 Decision Ledger (P03T3PR4-002 Reconciled)

This ledger defines the exact architectural decisions and handoff contract for ADR-016. For each decision, source facts are strictly separated from Supervisor architectural policy.

| Decision ID | Area | Source / Canonical Facts | What Is Already Decided | What Remains Architectural Policy | Candidate Options | Recommended Option | Decision Status |
|---|---|---|---|---|---|---|---|
| **D1** | Pair WorkerSession persistence model | Canonical `Pair 1 o-- 1 WorkerSession` (`docs/05`). Pinned AO session ID persists across restore. | Supervisor must track pair-worker session relationship and attempt execution identity. | Relational storage schema (Model A mutable current record + plain snapshot vs Model B historical table + active pointer vs Model C join table). | Model A, Model B, Model C | **Model A (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D2** | ADR-012 Section 10 amendment | Accepted ADR-012 Section 10 states external calls occur strictly after durable `DISPATCHED`. | Task prompt transmission (`/send`) occurs strictly after `TaskAttempt` allocation and durable `DISPATCHED`. | Whether `createWorkerSession` is decoupled as a Pair session lifecycle side effect permissible before task dispatch. | Formally amend ADR-012 Section 10 for session creation timing vs retain strict post-dispatch session spawn | **Amend ADR-012 Section 10** | `UNRESOLVED_FOR_ADR` |
| **D3** | Spawn uncertainty / orphan handling | Pinned AO `POST /api/v1/sessions` has `PINNED_SPAWN_IDEMPOTENCY = ABSENT`. | `P03T3_SESSION_SPAWN_RECOVERY = UNRESOLVED_BLOCKER`. Client cannot deterministically rediscover session on crash. | Policy for handling possible orphan sessions created in crash window T2 -> T3. | Heuristic project-session query vs background/startup orphan cleanup reaper vs operator audit tool | **Background / manual reaper** | `UNRESOLVED_FOR_ADR` |
| **D4** | Dispatch saga durable representation | Pinned `POST /api/v1/sessions/{id}/send` has `PINNED_SEND_IDEMPOTENCY = ABSENT`. | 3-stage lifecycle: `DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`. `AOAdapter` remains stateless. | Exact database table / column representation of dispatch saga states. | Dedicated `dispatch_operations` table vs columns on `task_attempts` vs audit log derivation | **Dedicated table (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D5** | `SEND_REQUESTED` uncertain-delivery disposition | Crash after `SEND_REQUESTED` before `SEND_CONFIRMED` leaves `DELIVERY_OUTCOME = UNKNOWN`. Blind resend prohibited. Canonical edges: `DISPATCHED -> FAILED`, `FAILED -> HUMAN_REQUIRED`. | Legal transition path is `DISPATCHED -> FAILED`, followed by `FAILED -> HUMAN_REQUIRED`. Zero direct `DISPATCHED -> HUMAN_REQUIRED`. | Specific disposition policy. If ADR-016 chooses automatic `/kill` reconciliation: quarantine MUST NOT clear merely because StopWorker returned HTTP success; it clears through that path only AFTER authoritative old execution termination confirmation (`PHYSICAL_EXECUTION_RESOLUTION`). If ADR-016 chooses Class B (authoritative absence) or Class C (administrative risk acceptance), ADR-016 must explicitly define: resolution type, required evidence, operator authority, durable audit record, whether future dispatch is permitted, and preserve that old execution remains physically unresolved. Do not collapse all classes into `TERMINATED`. | Auto-kill session with physical termination verification (Class A) vs operator pause/escalation with administrative risk acceptance (Class C) vs absence verification (Class B) | **Auto-kill session with physical termination confirmation or explicit administrative risk acceptance (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D6** | Uncertain-delivery retry quarantine guard | Canonical graph allows `FAILED -> READY`. An unquarantined retry risks duplicate execution while uncertain execution runs. | `UNCERTAIN_DELIVERY_QUARANTINE = MANDATORY`. Bound to original `(attempt_id, session_id, terminal_generation)`. Automated retry remains prohibited while `resolution_state = UNRESOLVED`. Replacement session/generation alone remains insufficient. | Durable state model and clearance protocol for quarantine guard across candidate resolution classes (physical termination vs administrative risk acceptance). | Option A (Pair metadata), Option B (Op flag), Option C (Attempt disposition), Option D (Combination) | **Option D: Combo (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D7** | Pre-send admissibility by AO activity state | Pinned SessionGuard prohibits writes on `blocked` (`SuppressedAwaitingUser`), `exited` (`SuppressedExited`), `terminated` (`SuppressedTerminated`). `waiting_input` accepts input. `idle` is clean prompt. Generation mismatch breaks fence. | `blocked`, `exited`, `terminated`, and generation mismatch are strictly `SEND_PROHIBITED`. | Admissibility of `active` before task send, and whether Supervisor enforces strict whitelist (`idle`, `waiting_input`) or includes unblocking routines. | Strict whitelist (`idle`, `waiting_input` permitted; all others prohibited) vs retry wait loop for `active` vs automatic nudge | **Strict whitelist (`idle`, `waiting_input` only) (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D8** | Missed-active-window policy | `SEND_CONFIRMED` committed, but subsequent observation snapshot is `idle`. | Does not prove whether prompt ran or was missed. Cannot auto-transition `DISPATCHED -> RUNNING -> REPORT_READY`. TASK-P03-003 does NOT perform report artifact reads, workspace file probing, or report validation (strictly owned by TASK-P03-004 and P04). | Lifecycle reconciliation choice for `MISSED_ACTIVE_WINDOW`: (A) fail closed `DISPATCHED -> FAILED`, (B) preserve `DISPATCHED` and escalate lifecycle ambiguity to operator, (C) persist ambiguity disposition and hand off to downstream authorized phase (TASK-P03-004 / P04), or (D) other canonical lifecycle architecture. | Option A (Fail closed: `DISPATCHED -> FAILED`) vs Option B (Preserve `DISPATCHED` and escalate ambiguity) vs Option C (Persist ambiguity disposition and hand off to downstream phase) | **Option C: Persist ambiguity disposition and hand off to downstream phase (or Option A fail-closed) (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D9** | `waiting_input` TaskState mapping | Pinned AO `waiting_input` means agent paused at empty prompt. | Normalized observation is `AO_WAITING_INPUT`. | Whether `waiting_input` transitions TaskState from `RUNNING` or remains in `RUNNING`. | Remain in `RUNNING` and emit telemetry warning vs transition `RUNNING -> BLOCKED` (reason: `UNEXPECTED_INPUT_PROMPT`) | **Remain in `RUNNING` + operator alert (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D10** | `blocked` TaskState mapping | Pinned AO `blocked` means agent stopped on tool permission / approval decision. Automated input prohibited. | Normalized observation is `AO_BLOCKED_DECISION`. Under current unresolved baseline, `TaskAttempt.ended_at` is NOT set unless ADR-016 decides BLOCKED terminates the attempt (see D12). Canonical TaskState `BLOCKED` is not terminal (permits `BLOCKED -> HUMAN_REQUIRED`). | Whether `blocked` observation triggers `RUNNING -> BLOCKED` in workflow state machine, and whether such transition terminates the current TaskAttempt (cross-referenced with D12). | Transition `RUNNING -> BLOCKED` (with attempt termination or retaining open attempt) vs pause observation poller in `RUNNING` awaiting operator decision | **Evaluate `RUNNING -> BLOCKED` (attempt-termination policy unresolved, see D12) (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D11** | Intentional-stop operation provenance | Pinned route is `POST /api/v1/sessions/{sessionId}/kill` returning `KillSessionResponse`. HTTP success does not substitute for observed termination. | Termination classification requires generation-matched stop provenance; HTTP 200 alone does not confirm termination. | Multi-stage durable audit/operation representation (`STOP_REQUESTED`, `STOP_CALL_SUCCEEDED / FAILED`, `STOP_TERMINATION_CONFIRMED`). | 3-stage durable operation bound to attempt/generation vs synchronous blocking kill in StateStore | **3-stage durable operation lifecycle (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D12** | Atomic terminal transition + `ended_at` ownership | ADR-015 specifies SQLite engine and transactional audit log. Canonical auditability requires crash-consistent state transitions. | Terminal failure transitions (`RUNNING -> FAILED`, `DISPATCHED -> FAILED`) must atomically commit Task state + `ended_at = now` + audit event append. Success `ended_at` owned by P04. For `RUNNING -> BLOCKED`, whether it ends an attempt is `UNRESOLVED_FOR_ADR` (see D10): IF ADR-016 decides `RUNNING -> BLOCKED` and `BLOCKED` ends the attempt, then TaskState transition + `ended_at = now` + audit persistence must occur atomically; IF ADR-016 decides `RUNNING -> BLOCKED` while retaining the open attempt, `ended_at` remains NULL; IF ADR-016 chooses no transition on blocked, attempt remains open. Do NOT label `RUNNING -> BLOCKED` as an unconditional terminal outcome. | Whether `BLOCKED` ends the current TaskAttempt (jointly decided with D10) and the StateStore API signature / transaction coordination method for atomic terminal updates. | Unified `AtomicTerminalTransition(...)` method with conditional attempt close vs explicit multi-entity transaction closure | **Unified StateStore atomic transition method (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |
| **D13** | Restart scanner coverage for `DISPATCHED` / `RUNNING` | Power loss leaves in-flight tasks in `DISPATCHED` or `RUNNING` with open attempts (`ended_at IS NULL`). | In-flight tasks must be reconciled upon startup without assuming success or failure. | Implementation architecture of the restart recovery scanner (on-startup blocking phase vs background poller integration). | Synchronous pre-flight startup scan vs lazy poller-driven attempt reconciliation | **Synchronous startup recovery sweep (`GIẢ ĐỊNH`)** | `UNRESOLVED_FOR_ADR` |

---

## 17. Architecture Impact & ADR Verdict

```text
P03_ARCHITECTURE_CHANGE = YES
P03_ADR_REQUIRED = YES
ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET
```
- **Rationale for Architecture Change**:
  1. Integrates durable dispatch saga stages (`DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`) as gating preconditions for lifecycle attribution.
  2. Introduces `UNCERTAIN_DELIVERY_QUARANTINE` and pre-dispatch retry guards bound to original execution identity.
  3. Formally amends ADR-012 Section 10 to separate Pair Session Lifecycle Side Effects from Task Execution Side Effects.
  4. Amends domain persistence relationships (`Pair 1 o-- 1 WorkerSession`) and adds attempt execution snapshots (`terminal_generation`).
- **ADR Authorization**: An Architecture Decision Record (`ADR-016`) is mandatory. Drafting remains **strictly unauthorized** until Proposal Revision 4 is independently reviewed and approved by the External Supervisor.

---

## 18. Explicit Non-Goals

- NO production code implementation (`internal/**/*.go`).
- NO schema migration execution or new database tables.
- NO ADR drafting (`ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET`).
- NO task release (`TASK_P03_003 = NOT_RELEASED`).
- NO workspace file fetching or report retrieval (`TASK-P03-004`).
- NO report JSON parsing, validation, or `worker_report_raw` persistence (`P04`).
- NO invented numeric timeout or interval defaults.
- NO SSE `/api/v1/events` or CDC event bus implementation.
- NO blind retries or synthetic heartbeats.

---

## 19. External Approval Gate

```text
PROPOSAL_P03_002 = REVISION_5_PENDING_EXTERNAL_REAUDIT
P03_ARCHITECTURE_CHANGE = YES
P03_ADR_REQUIRED = YES
ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET
TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT_005
```

Execution is halted awaiting independent External Supervisor re-audit of Proposal Revision 5.
