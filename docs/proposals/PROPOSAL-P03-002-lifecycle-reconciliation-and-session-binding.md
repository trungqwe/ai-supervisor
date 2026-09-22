# PROPOSAL-P03-002: Lifecycle Reconciliation, Durable Dispatch Saga, and Session Binding Specification

> **Proposal ID**: `PROPOSAL-P03-002`
> **Revision**: `2`
> **Status**: `REVISION_2_PENDING_EXTERNAL_REAUDIT`
> **Task Binding**: `TASK-P03-003` (Lifecycle Reconciliation and State Transitions)
> **Baseline Commit**: `70514775b9bb877febc32d5fb39aea70f33513da`
> **Pinned AO Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Architecture Classification**: `P03_ARCHITECTURE_CHANGE = YES`, `P03_ADR_REQUIRED = YES`, `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET`
> **Implementation Guard**: `TASK_P03_003 = NOT_RELEASED`, `P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION`
> **Active Gate**: `EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT_002`

---

## 1. Executive Summary & Problem Statement

This proposal specifies the architectural reconciliation, state reconciliation rules, and crash-recovery protocols for `TASK-P03-003` under pinned Agent Orchestrator (`AO`) v0.13.0 (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`).

Revision 2 incorporates the mandatory findings of External Re-Audit `P03T3PR2-001` through `P03T3PR2-008`:
1. **ActivityExited Matrix Correction (`P03T3PR2-001`)**: Corrects the contradiction in Revision 1 where `ActivityExited` was mapped to `FAILED (DECIDED)`. Under pinned AO semantics, the agent process exiting does not terminate the managed terminal or session; `ActivityExited` is an observation rule (`AO_PROCESS_EXITED`) with `NO AUTOMATIC TRANSITION` to `FAILED`.
2. **AO Blocked Decision Boundary (`P03T3PR2-002`)**: Clarifies that AO `blocked` represents an interactive permission/approval decision dialog in AO runtime semantics. It does not automatically imply Supervisor `TaskState BLOCKED`. The mapping is marked `UNRESOLVED_DECISION` pending ADR-016, and `TaskAttempt.ended_at` is NOT set.
3. **Removal of Unavailable Recovery Evidence (`P03T3PR2-003`)**: Eliminates unpersisted ephemeral spawn metrics ("zero prompt bytes") and non-public fields (`latestUserPrompt`) from recovery rules. Pinned public `SessionView` exposes only `lastUserMessageAt` (real user directions), which cannot prove `/send` delivery.
4. **Durable Dispatch Operation Lifecycle & Send Idempotency (`P03T3PR2-004`)**: Records `PINNED_SEND_IDEMPOTENCY = ABSENT` (`P03T3_SEND_RECOVERY = UNRESOLVED_BLOCKER`). Replaces volatile timing labels with durable dispatch stages (`DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`). Prohibits blind resend when delivery outcome is unknown.
5. **Coherent WorkerSession Persistence Models (`P03T3PR2-005`)**: Evaluates three coherent relational models (Model A: mutable current binding + immutable attempt snapshot; Model B: historical WorkerSession + active pointer; Model C: AttemptSessionBinding join entity). Recommends Model A as `GIẢ ĐỊNH` with non-contradictory schema constraints.
6. **Explicit ADR-012 Conflict & Amendment (`P03T3PR2-006`)**: Analyzes the conflict between pre-dispatch session creation and ADR-012 Section 10, distinguishing Pair Session Lifecycle Side Effects from Task Execution Side Effects, and specifying the exact required amendment.
7. **Public Crash Evidence Boundary (`P03T3PR2-007`)**: Accurately bounds crash classification under public pinned endpoints where exit codes and signals are unavailable: `WORKER_CRASHED_PUBLIC_BASELINE = NOT_CURRENTLY_PROVABLE`. Unexplained terminations map to `WORKER_TERMINATION_UNKNOWN`.
8. **Scoped Agy/TUI Generation Fence (`P03T3PR2-008`)**: Proves from pinned source that canonical V1 harness `agy` is absent from the Chat registry and falls back to TUI, establishing `AGY_V1_EXECUTION_GENERATION = terminalGeneration / RuntimeLaunchID`. Generic harness epochs remain `UNRESOLVED_DECISION`.

---

## 2. Authority Hierarchy & Governance Baseline

All resolutions adhere strictly to the 9-level decision hierarchy defined in `docs/24_CHANGE_GOVERNANCE.md`:

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

### 3.1 Pinned Endpoints & Idempotency Facts
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
3. **Get Session (`GET /api/v1/sessions/{id}`)**:
   - Returns: `SessionView` (`domain.Session`, `branch`, `terminalGeneration`, `previewUrl`, `previewRevision`, `model`, `lastUserMessageAt`, `prs`, `activeAgentSwitch`).
   - Fact: `promptBytes` is an ephemeral spawn response field and is **NOT** present in `SessionView`.
   - Fact: `latestUserPrompt` is an internal domain field and is **NOT** present in `SessionView`. `lastUserMessageAt` records real user directions and explicitly excludes automation/task updates.
   - Fact: Public `SessionView` does **NOT** expose process exit codes, OS signals, or reaper logs (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NOT_CURRENTLY_PROVABLE`).

### 3.2 Pinned Harness & Mode Architecture
1. **Chat Driver Registry (`backend/internal/adapters/chatdriver/registry/registry.go`)**:
   - Registered harnesses: `codex`, `claude-code`, `opencode`, `droid`, `kimi`, `kimchi`, `pi`, `cursor`, `omp`.
   - Harness `agy` is **NOT** registered.
2. **Spawn Fallback (`backend/internal/session_manager/manager.go:886`, `chat_spawn.go:344`)**:
   - When Chat mode is defaulted or unresolvable for `agy`, AO returns `ports.ErrChatUnsupported` and falls back cleanly to `SessionModeTUI`.
3. **Execution Fence Alignment**:
   - In TUI mode, AO sets `terminalGeneration = s.Metadata.RuntimeLaunchID`.
   - In Chat mode, AO sets `ControllerGeneration` and leaves `RuntimeLaunchID` unset.
   - For canonical V1 `agy` workers: `AGY_V1_EXECUTION_GENERATION = terminalGeneration / RuntimeLaunchID`.
   - For generic/future Chat harnesses: `GENERIC_EXECUTION_EPOCH = UNRESOLVED_DECISION`.

---

## 4. Session Spawn Crash Window & Blocker Status

### 4.1 The Spawn Crash Window
```text
T0: Supervisor commits PrepareDispatch (Task = DISPATCHED, TaskAttempt = durable, ended_at = NULL).
T1: Supervisor dispatches POST /api/v1/sessions to AO.
T2: AO creates session upstream, allocates worktree/ConPTY, assigns session_id, returns HTTP 201.
T3: Supervisor commits session_id to local StateStore.
```
- A crash between **T2 and T3** leaves an active AO session running upstream with its identity lost to the Supervisor.
- Because pinned AO lacks caller-provided session IDs or spawn idempotency keys:
```text
PINNED_SPAWN_IDEMPOTENCY = ABSENT
P03T3_SESSION_SPAWN_RECOVERY = UNRESOLVED_BLOCKER
```
- A deterministic recovery cannot be fabricated in client code without upstream API changes or architectural pre-allocation.

---

## 5. ADR-012 Conflict and Required Amendment (P03T3PR2-006 Reconciled)

### 5.1 The Existing ADR-012 Invariant
Accepted ADR-012 (`docs/adr/ADR-012-task-contract-revision-and-attempt-binding.md`) Section 10 states:
> "- From the Supervisor domain perspective, `TaskAttempt` allocation and the `READY -> DISPATCHED` state transition must be persisted **atomically** in the State Store before invoking any external execution side effects on Agent Orchestrator.
> - External AO API calls (`createWorkerSession`, `sendTask`) occur strictly *after* the durable `DISPATCHED` record exists."

### 5.2 The Architectural Conflict
If the future ADR-016 chooses to resolve the spawn T2 -> T3 crash window by pre-creating and durably binding a `WorkerSession` to the `Pair` *before* task dispatch (so `PrepareDispatch` binds to an already-known `session_id`):
- Creating the AO session prior to task dispatch directly violates the sentence: *"External AO API calls (`createWorkerSession`, `sendTask`) occur strictly after the durable `DISPATCHED` record exists."*
- Therefore, ADR-016 **MUST EXPLICITLY AMEND** ADR-012 Section 10.

### 5.3 Required Amendment Analysis
To reconcile this conflict cleanly, the architecture must distinguish:
1. **PAIR SESSION LIFECYCLE SIDE EFFECT (`createWorkerSession`)**:
   - Establishes or reuses the execution sandbox/worktree for a Pair.
   - Pertains to lane readiness and environment provisioning, not specific task execution.
2. **TASK EXECUTION SIDE EFFECT (`sendTask` / `/send`)**:
   - Transmits the immutable task specification and prompt to the worker.
   - Pertains to work execution and consumption of an attempt.

**Exact Sentence Amendment Required in ADR-016**:
> *ADR-012 Section 10 is amended as follows:*
> "Pair worker session provisioning (`createWorkerSession`) MAY occur prior to task dispatch as a Pair-scoped lifecycle side effect, establishing a durable `WorkerSession` binding. However, task execution side effects (`sendTask` / `/send`) occur strictly *after* `TaskAttempt` allocation and the durable `DISPATCHED` state record exist in the State Store."

This amendment strictly preserves the core governance invariant: **no task prompt is ever transmitted to a worker without an atomically persisted attempt and durable `DISPATCHED` record**.

---

## 6. WorkerSession Domain Model & Coherent Persistence Models (P03T3PR2-005 Reconciled)

### 6.1 Domain Specification
Canonical `docs/05_DOMAIN_MODEL.md` establishes:
```text
Pair "1" o-- "1" WorkerSession
```
Revision 1 proposed a candidate schema where `worker_sessions` had `session_id PRIMARY KEY` and non-unique `pair_id`, alongside a foreign key `task_attempts.session_id REFERENCES worker_sessions(session_id)`. This created a schema contradiction: if a Pair has only one current session, replacing that session over time breaks historical FK references from prior `task_attempts`.

### 6.2 Evaluation of Three Coherent Persistence Models

| Dimension | Model A: Mutable Current Session + Plain Attempt Snapshot | Model B: Historical WorkerSession Table + Active Pointer | Model C: Pair Current Session + AttemptSessionBinding Join Table |
|---|---|---|---|
| **Description** | `worker_sessions` stores the current active session keyed by `pair_id` (or `session_id` with unique `pair_id`). `task_attempts` stores immutable execution snapshot columns (`session_id TEXT`, `terminal_generation TEXT`) without an FK to `worker_sessions`. | `worker_sessions` is an append-only historical log of all sessions spawned. `pairs` table has `active_worker_session_id` FK. `task_attempts.session_id` has an FK to `worker_sessions`. | `worker_sessions` remains 1-to-1 with `pairs`. An immutable join table `attempt_session_bindings` records historical mappings (`attempt_id`, `session_id`, `terminal_generation`). |
| **Canonical Relationship Impact** | Perfectly preserves canonical `Pair 1 o-- 1 WorkerSession`. Zero structural drift. | Amends canonical domain to `Pair 1 o-- * WorkerSession`. Introduces relational cycle (`pairs` <-> `worker_sessions`). | Preserves `Pair 1 o-- 1 WorkerSession`, but introduces a new domain entity `AttemptSessionBinding`. |
| **DB Constraints** | Simple: `worker_sessions.pair_id UNIQUE`. Zero cyclic FKs. Snapshot columns in `task_attempts` are append-only. | Complex: cyclic foreign keys between `pairs` and `worker_sessions`, requiring deferred constraints or nullable references. | Moderate: additional table with compound primary key `(attempt_id, session_id)`. |
| **Historical Integrity** | High: `task_attempts` retains exact historical `session_id` and `terminal_generation` permanently, immune to session updates. | High: full session lifecycle history preserved in `worker_sessions` table. | High: full binding history preserved in dedicated binding table. |
| **Session Replacement** | Trivial: UPDATE `worker_sessions` in place or DELETE/INSERT new row. Old attempts unaffected. | Moderate: INSERT new `worker_sessions` row, UPDATE `pairs.active_worker_session_id`. | Trivial: UPDATE `worker_sessions`, INSERT new `attempt_session_bindings` row. |
| **Restore of Same Session** | Trivial: session row updated with new `terminal_generation`; attempts retain their launch epoch. | Trivial: session row updated with new `terminal_generation`. | Trivial: binding records launch epoch. |
| **New Session After Failure** | Trivial: Pair receives new session; previous failed attempt snapshot remains intact. | Trivial: Pair pointer updated; old session row marked terminated. | Trivial: new binding appended. |
| **Attempt Auditability** | High: attempt record contains self-contained execution identity without joins. | Moderate: requires joining historical session rows. | Moderate: requires joining binding table. |
| **Stale Observation Protection** | Complete: poller validates `snapshot.generation == attempt.terminal_generation`. | Complete: poller validates generation against attempt or session. | Complete: poller validates generation against binding. |

### 6.3 Recommendation (`GIẢ ĐỊNH`)
`GIẢ ĐỊNH`: **Model A** is recommended for ADR-016. It strictly upholds canonical `docs/05` (`Pair "1" o-- "1" WorkerSession`), avoids complex cyclic relational constraints, and guarantees permanent attempt auditability by recording immutable execution snapshots directly on `task_attempts`.

#### Candidate Schema DDL for Model A (Proposal-Level Only)
```sql
-- WorkerSessions: Exactly 1 current session per Pair
CREATE TABLE IF NOT EXISTS worker_sessions (
    session_id TEXT PRIMARY KEY,
    pair_id TEXT NOT NULL UNIQUE REFERENCES pairs(pair_id) ON DELETE RESTRICT,
    runtime_type TEXT NOT NULL,       -- Canonical: 'ao'
    worktree_path TEXT NOT NULL,      -- Canonical: worktree path
    worker_agent_id TEXT NOT NULL,    -- Canonical: 'agy'
    status TEXT NOT NULL,             -- Canonical: 'active', 'idle', 'terminated'
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- TaskAttempt: Immutable execution snapshot columns (Proposed canonical amendment)
ALTER TABLE task_attempts ADD COLUMN session_id TEXT;
ALTER TABLE task_attempts ADD COLUMN terminal_generation TEXT;
```

---

## 7. Execution Epoch & Terminal Generation Scope (P03T3PR2-008 Reconciled)

### 7.1 Session ID vs Execution Epoch
In AO, an upstream session ID survives restarts, relaunch, and recovery. However, a restored session receives a new process and terminal environment. Therefore:
```text
SESSION_ID != EXECUTION_EPOCH
```
An observation poller must fence lifecycle observations by execution epoch to prevent stale signals from a prior run affecting a restored run.

### 7.2 Source Provenance: TerminalGeneration vs ControllerGeneration
Inspection of pinned AO source reveals:
1. `GET /api/v1/sessions/{id}` returns `TerminalGeneration string` (`backend/internal/httpd/controllers/dto.go:253`), which maps directly to `s.Metadata.RuntimeLaunchID`.
2. In Chat mode, AO manages lifecycle via `ControllerGeneration` and leaves `RuntimeLaunchID` empty.
3. The canonical V1 worker harness for the Supervisor is `agy`.
4. Inspection of `backend/internal/adapters/chatdriver/registry/registry.go` confirms that `agy` is **NOT** a registered Chat harness.
5. In `backend/internal/session_manager/manager.go:886`, spawning `agy` in Chat mode fails preflight with `ports.ErrChatUnsupported` and falls back cleanly to `SessionModeTUI`.
6. In `SessionModeTUI`, AO assigns and bumps `RuntimeLaunchID` on every process launch and restore.

### 7.3 Canonical V1 Generation Fence Definition
```text
AGY_V1_EXECUTION_GENERATION = terminalGeneration / RuntimeLaunchID
```
- **Scope Limit**: This contract applies strictly to V1 `agy` workers executing under TUI mode.
- **Generic Harness Scope**: Generic execution epoch fencing across future Chat-capable harnesses is marked:
```text
GENERIC_EXECUTION_EPOCH = UNRESOLVED_DECISION
```
Future Chat integration will require a normalized execution epoch exposed through public AO endpoints.

---

## 8. Pinned /send Idempotency & Durable Dispatch Saga (P03T3PR2-004 Reconciled)

### 8.1 Pinned /send Fact
Inspection of `SendSessionMessageRequest` (`backend/internal/httpd/controllers/dto.go:838`):
```go
type SendSessionMessageRequest struct {
    Message string `json:"message"`
    Attachment *AttachmentInput `json:"attachment,omitempty"`
}
```
- Contains NO idempotency key, client message ID, or caller correlation ID.
- Upstream server processes messages sequentially and appends them to conversation history.
- `PINNED_SEND_IDEMPOTENCY = ABSENT`.
- `P03T3_SEND_RECOVERY = UNRESOLVED_BLOCKER`.

### 8.2 The Uncertain Delivery Window
If the host crashes after transmitting `POST /api/v1/sessions/{id}/send` but before receiving the HTTP 200 response:
- The delivery outcome is **UNKNOWN**.
- Public `GET /api/v1/sessions/{id}` cannot prove delivery (`lastUserMessageAt` ignores automation prompts, and no public message-receipt query exists).
- **Blind Resend Prohibition**: A blind resend risks injecting duplicate instructions into the agent process, potentially corrupting execution. **Blind resend is strictly prohibited**.

### 8.3 Durable Dispatch Operation Stages
To make dispatch crash states distinguishable after restart without timing heuristics, the proposal specifies a durable dispatch-operation lifecycle:

```text
1. DISPATCH_BOUND:
   Task = DISPATCHED, TaskAttempt allocated, session_id and terminal_generation durably bound.
   External /send has NOT been requested.

2. SEND_REQUESTED:
   Durable dispatch-operation record committed immediately BEFORE calling POST /api/v1/sessions/{id}/send.
   Bound to attempt_id, contract_id, session_id, and terminal_generation.

3. SEND_CONFIRMED:
   Durable dispatch-operation record committed immediately AFTER receiving HTTP 200 from AO.
```

#### Required Properties of the Dispatch Record:
1. **Pre-Effect Persistence**: `SEND_REQUESTED` must be flushed to disk before network I/O begins.
2. **Attempt Fencing**: Bound to `(task_id, attempt_id, contract_id, session_id, terminal_generation)`.
3. **Distinguishability**: After restart, the recovery scanner can distinguish between:
   - Not yet sent (`DISPATCH_BOUND`);
   - In-flight / unconfirmed (`SEND_REQUESTED`);
   - Successfully acknowledged (`SEND_CONFIRMED`).
4. **Append-Only / Auditable**: Operation transitions are recorded in the audit trail.
5. **Architectural Isolation**: `AOAdapter` remains entirely stateless and StateStore-free; the dispatch saga is orchestrated by the domain workflow layer.

#### Critical Governance Limit:
Durable `SEND_REQUESTED` does **not** solve uncertain delivery—it makes the uncertainty explicit. When restart detects `SEND_REQUESTED` without `SEND_CONFIRMED`, `DELIVERY_OUTCOME = UNKNOWN`. The Supervisor must fail closed and escalate to human intervention (`HUMAN_REQUIRED`) rather than guessing.

---

## 9. Send Crash Matrix Using Only Durable Facts (Section 8 Reconciled)

| Crash State / Discriminator | Local Durable Facts | AO Facts via Public API | What is Proven | What Remains Unknown | Resend Safe? | TaskState Action | Human Escalation Required? | ADR-016 Decision Required? |
|---|---|---|---|---|---|---|---|---|
| **Crash at `DISPATCH_BOUND` (Before Send)** | Task `DISPATCHED`, open attempt, `session_id` bound, `terminal_generation` bound, zero `SEND_REQUESTED` record | Session exists, `ActivityIdle` or `ActivityWorking` | `/send` was NEVER initiated by Supervisor | None regarding dispatch | **YES** (first invocation) | Proceed with dispatch: write `SEND_REQUESTED`, invoke `/send` | NO | YES (saga transition mechanics) |
| **Crash at `SEND_REQUESTED` (No Confirmation)** | Task `DISPATCHED`, open attempt, `session_id` bound, durable `SEND_REQUESTED` exists, NO `SEND_CONFIRMED` | Session exists, activity observed (`active` or `idle`) | Request was transmitted to network | Whether AO processed the message (`PINNED_SEND_IDEMPOTENCY = ABSENT`) | **NO** (blind resend risks duplicate prompt) | **FAIL-CLOSED**: Transition Task to `HUMAN_REQUIRED` or terminal failure; halt automated dispatch | **YES** (operator must inspect session) | YES (escalation / quarantine protocol) |
| **Crash at `SEND_CONFIRMED` (Before First Poll)** | Task `DISPATCHED`, open attempt, `session_id` bound, durable `SEND_CONFIRMED` exists | Current activity snapshot (`active`, `idle`, `blocked`, etc.) | AO received and confirmed task prompt | First post-send activity | **N/A** (dispatch complete) | Resume normal lifecycle poller: if `active`, transition `RUNNING`; if `idle`, see Missed Active Window | NO (unless missed window triggers escalation) | YES (lifecycle reconciliation rules) |

---

## 10. REPORT_READY Boundary & Phase Ownership (P03T3PR1-005 Reconciled)

### 10.1 The Separation Principle
```text
AO_IDLE = TURN COMPLETION OBSERVATION
AO_IDLE != REPORT_READY
```
Observing `ActivityIdle` in AO indicates only that the worker process has concluded its execution turn. It provides **zero cryptographic, schema, or semantic proof** that a valid execution report exists.

### 10.2 Mandatory Handoff Boundary
TASK-P03-003 is strictly confined to **lifecycle observation and state reconciliation**.
TASK-P03-003 **MUST NOT**:
- Fetch report files from workspace worktrees (`TASK-P03-004`).
- Parse or validate `WorkerReport` JSON schemas (`P04`).
- Validate report claims against task contract boundaries (`P04`).
- Write `worker_report_raw` to the StateStore (`P04`).
- Transition the task to `REPORT_READY` (`P04`).

When AO reports `ActivityIdle`, TASK-P03-003 records an internal observation event:
```text
Observation = AO_TURN_COMPLETED
```
The Task **remains in `RUNNING`**. Handoff to the Evidence Collector (`P04`) occurs within `RUNNING`. Only `P04` may transition `RUNNING -> REPORT_READY`.

---

## 11. AO Activity State Mapping & Blocked Decision Boundary (P03T3PR2-001 & P03T3PR2-002 Reconciled)

Pinned AO authority (`backend/internal/domain/activity.go`) defines distinct interactive states:

### 11.1 AO Blocked vs Supervisor TaskState BLOCKED (`P03T3PR2-002`)
- **Pinned AO Semantics**: `blocked` means the agent process is stopped awaiting a tool-permission or human approval decision. Keystroke/automated injection is prohibited.
- **Supervisor TaskState Mapping**: This describes AO runtime execution semantics, **not** Supervisor task workflow states. A worker waiting for tool permission does not necessarily mean the governance task is in `TaskState BLOCKED`.
- **Resolution**:
  - Normalized Observation: `AO_BLOCKED_DECISION`.
  - TaskState Mapping: `UNRESOLVED_DECISION` pending explicit workflow determination in ADR-016.
  - Attempt Boundary: Do **NOT** set `TaskAttempt.ended_at` on AO blocked observation.

### 11.2 AO Waiting Input
- **Pinned AO Semantics**: `waiting_input` means the agent is paused at an empty prompt awaiting instructions.
- **Resolution**: Normalized observation `AO_WAITING_INPUT`. TaskState mapping: `UNRESOLVED_DECISION`.

### 11.3 ActivityExited Semantics (`P03T3PR2-001`)
- **Pinned AO Semantics**: Pinned AO explicitly establishes that an agent process exiting (`ActivityExited`) **does not mean** the managed session or terminal is dead. The terminal handle and workspace remain alive, inspectable, and potentially restorable.
- **Baseline Rule**:
  - `ActivityExited` alone = `AO_PROCESS_EXITED` observation only.
  - It does **NOT** prove `isTerminated`, worker crash, intentional stop, or session death.
  - **TaskState Action**: `NO AUTOMATIC TRANSITION` to `FAILED`. State is preserved in `DISPATCHED` or `RUNNING` awaiting terminal session confirmation or deadline expiration.
  - Resolution: `DECIDED only as an observation rule, not as a failure transition`.

---

## 12. Fail-Closed Termination Classification & Public Crash Evidence (P03T3PR2-007 Reconciled)

### 12.1 Approved Classification Taxonomy
Restoring the approved taxonomy from `PROPOSAL-P03-001`:
1. `WORKER_STOPPED`: Positive intentional-stop provenance successfully correlated to current execution generation.
2. `WORKER_CRASHED`: Unexpected termination corroborated by positive observable failure evidence.
3. `WORKER_TERMINATION_UNKNOWN`: Session termination is observed (`isTerminated: true`), but evidence cannot distinguish between crash, external SIGKILL, or unrecorded stop.

### 12.2 Public Baseline Crash Evidence Boundary
- Public `GET /api/v1/sessions/{id}` does not expose process exit codes, signals, or reaper logs.
- Pinned AO reaper death logs and SQLite internals are unapproved private surfaces.
- Therefore:
```text
PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE
WORKER_CRASHED_PUBLIC_BASELINE = NOT_CURRENTLY_PROVABLE
```
- **Fail-Closed Rule**: Whenever `isTerminated: true` is observed without matching intentional stop provenance, the termination classification **MUST BE** `WORKER_TERMINATION_UNKNOWN`. The Task transitions canonically `RUNNING -> FAILED`, honestly recording `WORKER_TERMINATION_UNKNOWN` in the audit payload.

---

## 13. Generation-Safe Stop Provenance

Stop provenance is modeled as a generation-safe multi-stage operation lifecycle:
1. `STOP_REQUESTED`: Written to audit log immediately before calling `DELETE /api/v1/sessions/{id}`.
2. `STOP_CALL_SUCCEEDED` / `STOP_CALL_FAILED`: Written immediately upon receiving HTTP response.
3. `STOP_TERMINATION_CONFIRMED`: Written when observation poller confirms `isTerminated: true`.

Every stop provenance record is bound to `(task_id, attempt_id, session_id, terminal_generation)`. A subsequent session restore allocating a new generation will not match the prior stop provenance, preventing stale stop inheritance.

---

## 14. Attempt End Ownership & Transaction Atomicity

1. **Phase Separation**: `RecordAttemptEnd(..., rawReport)` is strictly removed. Normal attempt `ended_at` is committed by `P04` upon `REPORT_READY`.
2. **Atomic Terminal Transitions**: When TASK-P03-003 executes a terminal failure transition (`RUNNING -> FAILED`, `DISPATCHED -> FAILED`), state mutation, attempt termination, and audit logging must be atomic:
```text
Atomic Terminal Transaction:
1. Task state Compare-And-Set (e.g. RUNNING -> FAILED).
2. TaskAttempt ended_at mutation (SET ended_at = now).
3. Audit event append (TASK_STATE_TRANSITION with full provenance).
```
3. **No Active Attempt Cancellation**: In canonical `state_machine.go`, transitions from `DISPATCHED` or `RUNNING` to `CANCELLED` **do not exist**. An operator stop results in `FAILED` with trigger `INTENTIONAL_STOP_BY_SUPERVISOR`, which can subsequently be cancelled from `HUMAN_REQUIRED`.

---

## 15. Operational Policies & Zero Invented Numbers

- **Zero Numeric Defaults**: There are strictly zero approved default numbers for timeouts, retry counts, or poll intervals.
- **The Eight Operational Policies Remain UNSET**:
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
- **Fail-Closed on AO Unavailable**: Transport errors preserve current TaskState (`DISPATCHED` or `RUNNING`). The poller does not implement automatic retry limits. If the caller-injected `SUPERVISOR_EXECUTION_DEADLINE` expires, the deadline triggers an unrecoverable timeout failure.

---

## 16. Comprehensive Lifecycle Recovery Matrix

| Persisted Task State | Authoritative AO Snapshot | Observed AO Facts | What Facts PROVE | What Facts DO NOT Prove | Generation Requirement | Normalized Observation Classification | TaskState Transition | Attempt ended_at Action | Required Positive Provenance | Audit Evidence | Authority / Dependency | Resolution Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `DISPATCHED` | `active` | Session active | Worker acknowledged prompt and started turn | Does not prove completion | Current generation matches attempt | `AO_WORKER_ACTIVE` | `DISPATCHED -> RUNNING` | Do NOT set | Snapshot `active` within deadline | Telemetry observation log | FR-005, FR-006 | **DECIDED** |
| `DISPATCHED` | `idle` | Session idle | Session at idle state | Does not prove whether prompt ran | Current generation matches attempt | `AO_TURN_COMPLETED` | `UNRESOLVED_DECISION` (Missed Active Window) | Do NOT set | Snapshot `idle` | Diagnostic log | ADR-016 | **UNRESOLVED_DECISION** |
| `DISPATCHED` | `waiting_input` | Agent waiting prompt | Paused awaiting instruction | Does not prove failure | Current generation matches attempt | `AO_WAITING_INPUT` | `UNRESOLVED_DECISION` | Do NOT set | Snapshot `waiting_input` | Diagnostic log | ADR-016 | **UNRESOLVED_DECISION** |
| `DISPATCHED` | `blocked` | Agent blocked on decision | Stopped on permission dialog | Automated keystrokes prohibited | Current generation matches attempt | `AO_BLOCKED_DECISION` | `UNRESOLVED_DECISION` | Do NOT set | Snapshot `blocked` | Diagnostic log | ADR-016 | **UNRESOLVED_DECISION** |
| `DISPATCHED` | `exited` | Process exited | Process died pre-run | Does not prove session terminated | Current generation matches attempt | `AO_PROCESS_EXITED` | **NO AUTOMATIC TRANSITION** (remain `DISPATCHED`) | Do NOT set | Snapshot `exited` | Telemetry diagnostic log | Fail-closed | **DECIDED** (observation rule only) |
| `DISPATCHED` | `isTerminated: true` | Session terminated | Session dead | Does not prove crash without evidence | Matched generation | `WORKER_STOPPED` (if stop proven) or `WORKER_TERMINATION_UNKNOWN` | `DISPATCHED -> FAILED` | Set `ended_at = now` atomically | `STOP_TERMINATION_CONFIRMED` or none | `TASK_STATE_TRANSITION` (`WORKER_TERMINATION_UNKNOWN` or `INTENTIONAL_STOP`) | FR-006, REC-014 | **DECIDED** |
| `DISPATCHED` | AO 404 Not Found | Session missing | Session purged or deleted | Does not prove why deleted | N/A | `AO_SESSION_NOT_FOUND` | `DISPATCHED -> FAILED` | Set `ended_at = now` atomically | Typed 404 APIError | `TASK_STATE_TRANSITION` (`SESSION_NOT_FOUND`) | REC-014 / Approved ADR | **DECIDED** |
| `DISPATCHED` | AO Unavailable | Daemon unreachable | Transport failure | Does not prove worker state | N/A | `AO_DAEMON_UNAVAILABLE` | **NO TRANSITION** (preserve `DISPATCHED`) | Do NOT set | Transport network error | Warning diagnostic log | Execution deadline | **DECIDED** |
| `RUNNING` | `active` | Session active | Agent tool execution active | Does not prove completion | Current generation matches attempt | `AO_WORKER_ACTIVE` | **NO TRANSITION** (remain in `RUNNING`) | Do NOT set | Snapshot `active` within deadline | Telemetry diagnostic log | Execution deadline | **DECIDED** |
| `RUNNING` | `idle` | Session idle | Agent completed execution turn | Does NOT prove report is valid (`AO_IDLE != REPORT_READY`) | Current generation matches attempt | `AO_TURN_COMPLETED` | **NO TRANSITION** (remain in `RUNNING` for P04 handoff) | Do NOT set in P03 (P04 sets on `REPORT_READY`) | Snapshot `idle` + generation match | `WORKER_TURN_COMPLETED` observation log | ADR-011 | **DECIDED** |
| `RUNNING` | `waiting_input` | Agent waiting prompt | Paused awaiting instruction | Does not prove failure | Current generation matches attempt | `AO_WAITING_INPUT` | `UNRESOLVED_DECISION` | Do NOT set | Snapshot `waiting_input` | Diagnostic log | ADR-016 | **UNRESOLVED_DECISION** |
| `RUNNING` | `blocked` | Agent blocked on decision | Stopped on permission dialog | Automated keystrokes prohibited | Current generation matches attempt | `AO_BLOCKED_DECISION` | `UNRESOLVED_DECISION` | Do NOT set | Snapshot `blocked` | Diagnostic log | ADR-016 | **UNRESOLVED_DECISION** |
| `RUNNING` | `exited` | Process exited | Process died mid-run | Does not prove session terminated | Current generation matches attempt | `AO_PROCESS_EXITED` | **NO AUTOMATIC TRANSITION** (remain in `RUNNING`) | Do NOT set | Snapshot `exited` | Telemetry diagnostic log | Fail-closed | **DECIDED** (observation rule only) |
| `RUNNING` | `isTerminated: true` | Session terminated | Session dead | Does not prove crash without evidence | Matched generation | `WORKER_STOPPED` (if stop proven) or `WORKER_TERMINATION_UNKNOWN` | `RUNNING -> FAILED` | Set `ended_at = now` atomically | `STOP_TERMINATION_CONFIRMED` or none | `TASK_STATE_TRANSITION` (`WORKER_TERMINATION_UNKNOWN` or `INTENTIONAL_STOP`) | FR-006, REC-014 | **DECIDED** |
| `RUNNING` | AO 404 Not Found | Session missing | Session purged or deleted | Does not prove why deleted | N/A | `AO_SESSION_NOT_FOUND` | `RUNNING -> FAILED` | Set `ended_at = now` atomically | Typed 404 APIError | `TASK_STATE_TRANSITION` (`SESSION_NOT_FOUND`) | REC-014 / Approved ADR | **DECIDED** |
| `RUNNING` | AO Unavailable | Daemon unreachable | Transport failure | Does not prove worker state | N/A | `AO_DAEMON_UNAVAILABLE` | **NO TRANSITION** (preserve `RUNNING`) | Do NOT set | Transport network error | Warning diagnostic log | Execution deadline | **DECIDED** |

---

## 17. Architecture Impact & ADR Verdict

```text
P03_ARCHITECTURE_CHANGE = YES
P03_ADR_REQUIRED = YES
ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET
```
- **Rationale for Architecture Change**:
  1. Modifies durable domain relationships and schema by introducing attempt snapshot execution fencing (`terminal_generation`).
  2. Introduces durable dispatch-operation stages (`DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`).
  3. Formally amends ADR-012 Section 10 to separate Pair Session Lifecycle Side Effects from Task Execution Side Effects.
  4. Restructures restart recovery boundaries across `DISPATCHED` and `RUNNING` states.
- **ADR Authorization**: An Architecture Decision Record (`ADR-016`) is mandatory. Drafting remains **strictly unauthorized** until this Proposal Revision 2 is independently reviewed and approved by the External Supervisor.

---

## 18. Explicit Non-Goals

- NO production code implementation (`internal/**/*.go`).
- NO schema migration execution or new database tables.
- NO ADR drafting (`ADR-016 = NOT_AUTHORIZED_TO_DRAFT_YET`).
- NO task release (`TASK_P03_003 = NOT_RELEASED`).
- NO workspace file fetching or report retrieval (`TASK-P03-004`).
- NO report JSON parsing, validation, or `worker_report_raw` persistence (`P04`).
- NO invented numeric timeout or interval defaults.
- NO SSE `/api/v1/events` or CDC event bus implementation.
- NO blind retries or synthetic heartbeats.

---

## 19. External Approval Gate

```text
PROPOSAL_P03_002 = REVISION_2_PENDING_EXTERNAL_REAUDIT
P03_ARCHITECTURE_CHANGE = YES
P03_ADR_REQUIRED = YES
ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET
TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT_002
```

Execution is halted awaiting independent External Supervisor re-audit of Proposal Revision 2.
