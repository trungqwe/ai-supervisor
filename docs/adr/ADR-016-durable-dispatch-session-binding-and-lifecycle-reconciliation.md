# ADR-016: Durable Worker Session Binding, Dispatch Saga, and Lifecycle Reconciliation

> **Status**: `PROPOSED — PENDING EXTERNAL APPROVAL`
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Proposal Authority**: `PROPOSAL-P03-002 Revision 6` (`EXTERNAL_APPROVED`, commit `8709af4b6aaf8613c69a10514971e320e3f90048`)
> **Deciders**: External Supervisor, Engineering Team
> **Amends**: ADR-012 (Section 10, where explicitly selected by this draft)
> **Related**: ADR-002, ADR-009, ADR-011, ADR-015
> **Implementation Guard**: `TASK-P03-003` remains `NOT_RELEASED`, `P03_CODE = HELD_PENDING_ADR_016_APPROVAL`.

---

## 1. Context

In Phase P01, runtime proof activities demonstrated that the Agent Orchestrator (`AO`) daemon provides process isolation, ConPTY virtual terminal management, git worktree lifecycle integration, and public session monitoring (`docs/audits/P01_A_AO_RUNTIME_PROOF.md`). In Phase P02, the Supervisor Control Plane domain model, state machine, and SQLite persistence engine were formally implemented and verified (`docs/audits/P02_FINAL_AUDIT = EXTERNAL_AUDIT_APPROVED`).

During pre-code architectural analysis for `TASK-P03-003` (Lifecycle Reconciliation and State Transitions), multiple structural gaps and contract contradictions were identified between the canonical specifications, accepted ADRs, and upstream AO server behavior:
1. **Relational Tension**: Canonical `docs/05_DOMAIN_MODEL.md` mandates `Pair "1" o-- "1" WorkerSession`, while `docs/14_FAILURE_RECOVERY.md` requires immutable auditability of historical `TaskAttempt` execution identities across session recreation.
2. **Missing Upstream Idempotency**: Upstream AO v0.13.0 endpoints (`POST /api/v1/sessions` and `POST /api/v1/sessions/{id}/send`) lack client-supplied idempotency keys or deduplication guarantees (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`, `PINNED_SEND_IDEMPOTENCY = ABSENT`).
3. **Uncertain Delivery Window**: A host crash or network failure after dispatch intent is recorded but before delivery confirmation leaves delivery outcome unknown; blind resend risks catastrophic duplicate code execution.
4. **ADR-012 Conflict**: Accepted ADR-012 Section 10 stated that external calls (`createWorkerSession`, `sendTask`) occur strictly *after* durable `DISPATCHED`, conflicting with pre-provisioning execution environments for Pairs.
5. **Observability Boundary**: Upstream AO provides no public worker crash evidence, exit codes, or heartbeat mechanisms in its pinned baseline (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`).

`PROPOSAL-P03-002 Revision 6` resolved these gaps through extensive architectural analysis and was formally approved by External Supervisor Re-Audit 006 (commit `8709af4b6aaf8613c69a10514971e320e3f90048`). This ADR establishes the canonical architectural decisions resulting from that proposal.

---

## 2. Problem Statement

How does the Supervisor Control Plane achieve deterministic, crash-consistent worker session binding, task dispatching, and lifecycle reconciliation under an upstream execution engine that lacks caller-provided idempotency, provides no public process exit evidence, and separates sandbox management from task execution?

Specifically, the architecture must resolve:
1. How to durably bind worker sessions to Pairs while maintaining immutable execution lineage for individual task attempts.
2. How to sequence session creation and task dispatch without violating state machine invariants or coupling sandbox provisioning failures to task dispatch.
3. How to represent dispatch progress durably across crash windows where network delivery cannot be verified.
4. How to prevent duplicate execution when delivery outcome is unknown without violating canonical state machine transitions.
5. How to reconcile upstream activity observations (`active`, `idle`, `waiting_input`, `blocked`, `exited`) into canonical TaskStates without over-interpreting unproven signals.
6. How to execute intentional stop operations with cryptographic-grade provenance and without race conditions.
7. How to guarantee crash-consistent state transitions in SQLite without performing network calls inside database transactions.
8. How to perform startup restart reconciliation without assuming success or premature failure.

---

## 3. Decision Drivers

- **Zero Duplicate Execution**: Under no circumstances may the Supervisor re-dispatch or resend a prompt to an unquarantined session whose execution state is uncertain (`NEW_SESSION_OR_GENERATION != RESOLUTION_OF_PRIOR_UNCERTAIN_EXECUTION`).
- **Strict Canonical State Machine Adherence**: All transitions must execute via valid edges in the canonical workflow graph (`docs/06_WORKFLOW_STATE_MACHINE.md`). No invented TaskStates are permitted.
- **Transactional SQLite Invariants**: In accordance with ADR-015, database transactions must remain short, local, and synchronous. Absolutely NO network or AO API calls are permitted inside SQLite transactions.
- **Strict Component Boundaries**: Per `PROPOSAL-P03-001`, TASK-P03-003 owns lifecycle observation and state transitions only. TASK-P03-004 owns raw workspace file transport. Phase P04 owns WorkerReport semantic ingestion and validation. TASK-P03-003 must never read files or transition `REPORT_READY`.
- **Honest Observability**: The Supervisor logs only what is positively proven by authoritative evidence. Absences of evidence must never be synthesized into facts (e.g., termination without cause evidence is logged as `WORKER_TERMINATION_UNKNOWN`, never `WORKER_CRASHED`).
- **Configurable Operational Policies**: Numeric operational parameters (timeouts, intervals, deadlines) remain strictly unhardcoded (`UNSET`), requiring caller injection.

---

## 4. Pinned AO Public Facts

All architecture decisions in this ADR are derived strictly from the verified behavior of `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`):
1. **Spawn Wire Route**: `POST /api/v1/sessions` generates an upstream session, allocates a git worktree, and spawns a ConPTY terminal. Returns `201 Created` with a server-generated UUID. Lacks caller-provided idempotency keys (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`).
2. **Send Wire Route**: `POST /api/v1/sessions/{id}/send` writes text/prompt to the terminal. Returns `200 OK` on write success. Lacks caller-provided message IDs or idempotency keys (`PINNED_SEND_IDEMPOTENCY = ABSENT`).
3. **Stop Wire Route**: `POST /api/v1/sessions/{sessionId}/kill` terminates the session terminal and marks it terminated (`isTerminated = true`). Returns `200 OK` (`KillSessionResponse`). HTTP `DELETE` does NOT exist in the upstream API.
4. **Session Observation Wire Route**: `GET /api/v1/sessions/{id}` returns the public `SessionView` struct.
5. **Activity Taxonomy**: Exactly five (5) activity states exist in pinned AO: `active`, `idle`, `waiting_input`, `blocked`, `exited`.
6. **SessionGuard Rules**: The upstream session guard refuses automated input writes when activity is `blocked` (`SuppressedAwaitingUser`), `exited` (`SuppressedExited`), `terminated` (`SuppressedTerminated`), or during initial startup (`ErrStartupPending`).
7. **Execution Generation**: Pinned AO exposes `terminalGeneration` (int) / `RuntimeLaunchID` (string) on `SessionView`. This generation increments on terminal restart or restore, providing an authoritative execution fence for V1 `agy` TUI sessions.
8. **Diagnostic Timestamp**: `lastActivityAt` is an internal diagnostic timestamp updated on terminal activity. It is NOT a heartbeat, NOT a dispatch acknowledgement, and NOT proof of current attempt execution.
9. **No Public Crash Evidence**: Pinned AO exposes no OS exit codes, signals, or process crash logs on its public API (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`). `ActivityExited` denotes process termination inside an open session, which alone does NOT prove worker crash or session termination.

---

## 5. Canonical Constraints

1. **Architecture Decision Hierarchy**: Per `docs/24_CHANGE_GOVERNANCE.md`, confirmed user requirements and canonical architecture supersede worker proposals. Modifications to canonical documents require formal external approval.
2. **TaskState Graph Immutability**: The canonical TaskState graph consists strictly of:
   `DRAFT -> READY -> DISPATCHED -> RUNNING -> REVIEWING -> APPROVED / CANCELLED`
   with failure and revision branches:
   `DISPATCHED -> FAILED`, `RUNNING -> FAILED`, `RUNNING -> BLOCKED`, `BLOCKED -> HUMAN_REQUIRED`, `FAILED -> HUMAN_REQUIRED`, `FAILED -> READY`, `REVIEWING -> REVISION_REQUIRED -> READY`, `HUMAN_REQUIRED -> DRAFT / CANCELLED`.
   Zero new TaskStates may be added. Saga stages and quarantine flags must reside in auxiliary metadata structures.
3. **Contract Immutability**: Per ADR-010 and ADR-012, once dispatched, a `TaskContract` revision is permanently immutable.

---

## 6. Decision Summary

The thirteen core decisions (D1 through D13) established by this ADR are summarized in the decision completeness matrix below. Every decision represents a proposed architectural resolution approved for drafting:

### D1–D13 Architectural Decision Completeness Table

| Decision ID | Selected Proposed Decision | Primary Rationale | Rejected Alternative(s) | Persistence Impact | TaskState Impact | External Approval Status |
|---|---|---|---|---|---|---|
| **D1** | **Model A: Mutable Current Session + Plain Attempt Snapshot** | Preserves canonical `Pair 1 o-- 1 WorkerSession` without cyclic FKs; immutable attempt columns guarantee historical auditability. | Model B (Historical table + active pointer); Model C (Join table). | Adds snapshot columns to `task_attempts`; creates `worker_sessions` table with unique `pair_id`. | None. | `PENDING_EXTERNAL_AUDIT` |
| **D2** | **Formally Amend ADR-012 Section 10** | Decouples Pair-scoped sandbox provisioning from task-scoped prompt dispatch; `/send` remains strictly post-`DISPATCHED`. | Retain strict post-dispatch session spawn. | None directly. | None. | `PENDING_EXTERNAL_AUDIT` |
| **D3** | **Fail-Closed Spawn Disposition with Correlation Token and Orphan Reaper** | Pinned AO spawn lacks idempotency; guessing or adopting unproven sessions risks cross-talk; fail-closed ensures integrity. | Heuristic directory adoption; Upstream idempotency assumption. | Pre-spawn audit event with correlation token. | None. | `PENDING_EXTERNAL_AUDIT` |
| **D4** | **Dedicated `dispatch_operations` Table with 3-Stage Saga Lifecycle** | Cleanly tracks `DISPATCH_BOUND -> SEND_REQUESTED -> SEND_CONFIRMED` without bloating core domain entities; ensures crash recovery. | Columns on `task_attempts`; Pure audit log event sourcing. | Creates `dispatch_operations` table. | None (TaskState remains `DISPATCHED`). | `PENDING_EXTERNAL_AUDIT` |
| **D5** | **Fail-Closed Terminal Failure (`DISPATCHED -> FAILED`) with Immediate Quarantine** | Pinned `/send` lacks delivery proof on crash; assuming failure and quarantining prevents split-brain execution. | Blind resend; Silent continuation; Direct `DISPATCHED -> HUMAN_REQUIRED`. | Records `resolution_state` in dispatch operations and audit log. | `DISPATCHED -> FAILED -> HUMAN_REQUIRED`. | `PENDING_EXTERNAL_AUDIT` |
| **D6** | **Option D: Double-Gated Quarantine (Pair Lane Lock + Attempt Quarantine)** | Provides defense-in-depth against duplicate execution across both pair lane and task attempt levels. | Single-level lane check; Single-level attempt check. | `worker_sessions.quarantine_state` and `task_attempts.recovery_disposition`. | Blocks `FAILED -> READY` while quarantined. | `PENDING_EXTERNAL_AUDIT` |
| **D7** | **Strict Whitelist Pre-Send Admissibility (`idle`, `waiting_input` only)** | SessionGuard blocks writes on `blocked`/`exited`/`terminated`; `active` before send indicates foreign turn. | Permitting `active` before send; Automated dialog nudging. | Records rejection audit event. | Rejection halts dispatch or fails attempt. | `PENDING_EXTERNAL_AUDIT` |
| **D8** | **Option C: Persist Ambiguity Disposition and Handoff Downstream** | Respects task boundaries (TASK-P03-003 does not read files); allows downstream phases to validate fast-completing turns. | Immediate fail-closed; In-task report probing. | `task_attempts.recovery_disposition = 'MISSED_ACTIVE_WINDOW_AMBIGUITY'`. | `DISPATCHED -> RUNNING` (with ambiguity flag). | `PENDING_EXTERNAL_AUDIT` |
| **D9** | **Preserve `RUNNING`, Emit Diagnostic Telemetry, Continue Polling** | `waiting_input` indicates worker is at empty prompt; does not indicate failure or permission blockage. | Transition to `BLOCKED`; Transition to `FAILED`. | Diagnostic audit telemetry only. | None (remains `RUNNING`). | `PENDING_EXTERNAL_AUDIT` |
| **D10** | **Transition `RUNNING -> BLOCKED`, Retain Open Attempt (`ended_at = NULL`)** | Canonical `BLOCKED` allows operator unblocking/escalation; closing attempt would preclude resumption. | Closing attempt (`ended_at = now`); Ignoring blocked activity. | TaskState updated to `BLOCKED`; `ended_at` remains `NULL`. | `RUNNING -> BLOCKED`. | `PENDING_EXTERNAL_AUDIT` |
| **D11** | **3-Stage Durable Stop Operation Lifecycle via `POST .../kill`** | HTTP 200 proves only kill command acceptance; termination requires authoritative generation-matched observation. | Immediate termination assumption; Using HTTP `DELETE`. | Dedicated `stop_operations` records. | Drives `RUNNING -> FAILED` (`WORKER_STOPPED`). | `PENDING_EXTERNAL_AUDIT` |
| **D12** | **Unified StateStore Atomic Terminal Transition (`AtomicTerminalTransition`)** | Ensures crash consistency: TaskState CAS, attempt `ended_at`, and audit event committed in one SQLite transaction. | Multi-method non-transactional updates; Network calls inside transactions. | Atomically updates `tasks`, `task_attempts`, `audit_events`. | Executes terminal failure transitions. | `PENDING_EXTERNAL_AUDIT` |
| **D13** | **Synchronous Startup Recovery Sweep for In-Flight Tasks and Operations** | Reconciles power-loss crash states before accepting new work; handles AO unreachability gracefully without blind failures. | Lazy polling reconciliation; Blind failure of in-flight tasks. | Recovery audit records. | Reconciles in-flight tasks. | `PENDING_EXTERNAL_AUDIT` |

---

## 7. D1 — Pair / WorkerSession Persistence Model

### Source / Canonical Facts
- Canonical `docs/05_DOMAIN_MODEL.md` defines `Pair "1" o-- "1" WorkerSession`.
- Pinned AO `session_id` persists across terminal restore operations.
- `TaskAttempt` requires immutable record of the execution identity under which work was performed.

### Proposal Revision-6 Recommendation
- Model A (`GIẢ ĐỊNH`): Mutable Current Session in `worker_sessions` + Plain Historical Attempt Snapshot in `task_attempts`.

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Adopt Model A (Mutable Current WorkerSession + Self-Contained TaskAttempt Execution Snapshot)**.
1. The `worker_sessions` table maintains exactly one current active worker session per Pair, keyed by `pair_id TEXT PRIMARY KEY / UNIQUE`. Columns: `pair_id`, `session_id`, `worktree_path`, `terminal_generation`, `quarantine_state`, `created_at`, `updated_at`.
2. The `task_attempts` table stores self-contained execution snapshot columns: `session_id TEXT` and `terminal_generation TEXT`, populated at `DISPATCH_BOUND` and permanently immutable.
3. No relational Foreign Key constraint exists between `task_attempts.session_id` and `worker_sessions.session_id`.
4. Session Replacement: If a Pair's session is killed or recreated, `worker_sessions` is updated in place. Existing historical `task_attempts` remain completely unaffected.
5. Session Restore: Restoring a session within AO preserves `session_id` but increments `terminal_generation`. The `worker_sessions` record updates its generation accordingly.
6. Stale Observation Protection: Polling and recovery workers match upstream `terminalGeneration` against `task_attempts.terminal_generation`. If they differ, the observation is rejected with `STALE_EXECUTION_GENERATION`.

### Rationale
Model A perfectly preserves the canonical 1:1 relationship between Pair and WorkerSession. It avoids the cyclic Foreign Key constraints that Model B introduces (where `pairs` points to `worker_sessions` and `worker_sessions` points to `pairs`), and avoids the join overhead and entity proliferation of Model C (`attempt_session_bindings`). Historical attempt auditability is completely guaranteed because snapshot columns on `task_attempts` are write-once.

### Rejected Alternatives
- **Model B (Historical WorkerSession Table + Active Pointer)**: Rejected because it mutates the canonical domain model to `Pair 1 o-- * WorkerSession` and creates relational cycle issues in SQLite.
- **Model C (Pair Current Session + AttemptSessionBinding Join Table)**: Rejected because introducing a third entity adds schema complexity without providing any auditability benefit over plain snapshot columns.

### Safety Invariant
Snapshot columns `(session_id, terminal_generation)` on `task_attempts` are permanently immutable once written.

### Persistence Impact
Adds columns `session_id TEXT` and `terminal_generation TEXT` to `task_attempts`. Defines `worker_sessions` schema.

### State-Machine Impact
None.

### Restart Impact
Attempt history is reconstructed entirely from self-contained `task_attempts` columns without joining active session tables.

### Auditability Impact
High: full historical fidelity of every execution identity is preserved permanently.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 8. D2 — ADR-012 External-Side-Effect Amendment

### Source / Canonical Facts
- Accepted ADR-012 Section 10 currently states: *"External AO API calls (`createWorkerSession`, `sendTask`) occur strictly after the durable `DISPATCHED` record exists."*

### Proposal Revision-6 Recommendation
- Formally amend ADR-012 Section 10.

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Formally Amend ADR-012 Section 10 to Decouple Pair Session Lifecycle Side Effects from Task Execution Side Effects**.
1. **Pair Session Lifecycle Side Effect (`createWorkerSession`)**: The creation, provisioning, or restoration of a worker session sandbox is classified as infrastructure provisioning scoped to the `Pair`. It MAY occur prior to task dispatch, establishing a durable `WorkerSession` binding on the Pair.
2. **Task Execution Side Effect (`sendTask` / `/send`)**: The transmission of a task specification and prompt to an agent is classified as task execution. It remains strictly governed by the invariant: `TaskAttempt` MUST be allocated and `TaskState = DISPATCHED` MUST be durably committed in SQLite BEFORE `/send` is called.
3. Amendment text is formally codified in Section 26 of this ADR.

### Rationale
Coupling session creation to task dispatch forces environment setup failures to be recorded as task dispatch failures, and prevents pre-warming agent environments or reusing sessions across multiple revision turns of a contract. Decoupling the two while strictly retaining the post-`DISPATCHED` gate for `/send` maintains 100% of the safety guarantees intended by ADR-012.

### Rejected Alternatives
- **Retain Strict Post-Dispatch Session Spawn**: Rejected because it prevents warm session reuse and forces redundant session teardown and recreate cycles.

### Safety Invariant
No task prompt is EVER transmitted to an upstream worker unless the task is in durable `DISPATCHED` state with an allocated `TaskAttempt`.

### Persistence Impact
None directly.

### State-Machine Impact
None.

### Restart Impact
A provisioned session with no `DISPATCHED` task is safely recognized as idle infrastructure rather than a corrupted dispatch attempt.

### Auditability Impact
Emits distinct audit events: `PAIR_SESSION_PROVISIONED` for infrastructure, `TASK_DISPATCHED` for execution.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 9. D3 — Spawn Uncertainty / Orphan Handling

### Source / Canonical Facts
- Pinned AO `POST /api/v1/sessions` has `PINNED_SPAWN_IDEMPOTENCY = ABSENT`.
- A crash between upstream session creation (T2) and local database commit (T3) results in an active AO session whose ID is unknown to the Supervisor.

### Proposal Revision-6 Recommendation
- Fail-closed operational disposition + deterministic identity verification + Background/startup orphan reaper (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Adopt Fail-Closed Spawn Disposition with Client Correlation Tokens and Background/Manual Orphan Reaper**.
1. When initiating session spawn, the Supervisor generates a unique `client_token` / correlation ID and logs it in the audit trail.
2. If a crash occurs during spawn before the returned `session_id` is committed:
   - On restart, the Supervisor does NOT attempt heuristic adoption of upstream sessions based on timestamps or paths.
   - The Pair provisioning operation is marked `PROVISIONING_FAILED`.
   - The Supervisor requires a fresh, explicit spawn operation.
3. Orphan Cleanup: A standalone background/manual reaper tool queries AO for sessions whose paths match the workspace root but whose IDs do not exist in the Supervisor's `worker_sessions` or recent `task_attempts`. These orphaned sessions are safely terminated via `POST /api/v1/sessions/{id}/kill`.

### Rationale
Without upstream idempotency keys, heuristic adoption based on worktree paths or creation times is dangerous; it risks binding to a session created by another user, test process, or previous run. Fail-closed behavior guarantees complete identity correctness.

### Rejected Alternatives
- **Heuristic Path Adoption**: Rejected as unsafe; path matching cannot prove that a session was created for the specific pending dispatch.
- **Upstream Idempotency Assumption**: Rejected as factually false under pinned AO v0.13.0.

### Safety Invariant
The Supervisor never adopts or binds an upstream session without an authoritative, committed record of its assignment.

### Persistence Impact
Records `SESSION_SPAWN_REQUESTED` audit event with client correlation token.

### State-Machine Impact
None.

### Restart Impact
Crashed spawn attempts do not block Supervisor startup; flagged as uncommitted and safe to re-attempt.

### Auditability Impact
Audit log explicitly documents failed spawn attempts and subsequent cleanup operations.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 10. D4 — Durable Dispatch Saga Representation

### Source / Canonical Facts
- Pinned AO `POST /api/v1/sessions/{id}/send` has `PINNED_SEND_IDEMPOTENCY = ABSENT`.
- Client `AOAdapter` must remain strictly stateless.

### Proposal Revision-6 Recommendation
- Dedicated table (`dispatch_operations`) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Dedicated `dispatch_operations` Table with 3-Stage Lifecycle**.
1. Dispatch execution is tracked via a dedicated table:
   ```sql
   CREATE TABLE dispatch_operations (
       operation_id TEXT PRIMARY KEY,
       attempt_id TEXT NOT NULL,
       pair_id TEXT NOT NULL,
       task_id TEXT NOT NULL,
       session_id TEXT NOT NULL,
       terminal_generation TEXT NOT NULL,
       stage TEXT NOT NULL,
       requested_at TIMESTAMP NOT NULL,
       confirmed_at TIMESTAMP,
       resolution_state TEXT,
       FOREIGN KEY (attempt_id) REFERENCES task_attempts(id)
   );
   ```
2. The 3 semantic stages:
   - `DISPATCH_BOUND`: TaskState committed `DISPATCHED`, `TaskAttempt` allocated, session and generation bound.
   - `SEND_REQUESTED`: Pre-effect intent durably committed immediately before calling `POST /api/v1/sessions/{id}/send`.
   - `SEND_CONFIRMED`: Committed immediately upon HTTP 200 response from AO `/send`.
3. `AOAdapter` remains completely stateless.
4. Attribution Rule: Upstream execution activity is attributed to a task ONLY if `stage == SEND_CONFIRMED` and both `session_id` and `terminal_generation` match.

### Rationale
A dedicated table keeps the core `task_attempts` domain entity clean of transient saga tracking state, simplifies crash recovery queries, and provides an auditable history of network dispatch attempts.

### Rejected Alternatives
- **Columns on `task_attempts`**: Rejected because multiple dispatch sagas may occur across retries, cluttering attempt records.
- **Pure Audit Log Event Sourcing**: Rejected because querying saga states during restart via JSON audit event parsing is inefficient and error-prone.

### Safety Invariant
`POST .../send` is never invoked unless `stage = SEND_REQUESTED` is durably committed in SQLite.

### Persistence Impact
Creates table `dispatch_operations` with indices on `attempt_id` and `stage`.

### State-Machine Impact
None (TaskState remains `DISPATCHED` across all saga stages).

### Restart Impact
Restart scanner directly queries `SELECT * FROM dispatch_operations WHERE stage = 'SEND_REQUESTED'`.

### Auditability Impact
Each stage transition writes a corresponding event to `audit_events`.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 11. D5 — SEND_REQUESTED Unknown-Delivery Disposition

### Source / Canonical Facts
- Crash between `SEND_REQUESTED` and `SEND_CONFIRMED` leaves `DELIVERY_OUTCOME = UNKNOWN`.
- Blind resend is strictly prohibited.
- Canonical state machine allows `DISPATCHED -> FAILED`, followed by `FAILED -> HUMAN_REQUIRED`. Direct `DISPATCHED -> HUMAN_REQUIRED` does not exist.

### Proposal Revision-6 Recommendation
- Class A physical resolution primary with Class C administrative risk acceptance fallback (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Fail Closed via `DISPATCHED -> FAILED`, Impose Immediate Quarantine, and Require Governed Resolution (Class A or Class C)**.
1. When restart or error handling detects an unconfirmed `SEND_REQUESTED`:
   - Transition `DISPATCHED -> FAILED` (failure reason: `UNCERTAIN_DELIVERY_CRASH`).
   - Atomically set `TaskAttempt.ended_at = now`.
   - Impose `UNCERTAIN_DELIVERY_QUARANTINE` on the tuple `(attempt_id, session_id, terminal_generation)`.
   - Transition `FAILED -> HUMAN_REQUIRED` for operator escalation.
   - Any automated retry (`FAILED -> READY`) is strictly rejected by the quarantine guard.
2. Governed Clearance Classes:
   - **Class A (`PHYSICAL_EXECUTION_RESOLUTION`)**: The Supervisor issues `POST /api/v1/sessions/{sessionId}/kill` to the old session and authoritatively observes `isTerminated == true`. Once physical termination is confirmed, quarantine clears with disposition `TERMINATION_CONFIRMED`.
   - **Class C (`ADMINISTRATIVE_RISK_RESOLUTION`)**: An authorized operator explicitly acknowledges that delivery may have occurred and old execution cannot be positively proven stopped, and authorizes a replacement recovery path. This is durably recorded in the audit log as `ADMINISTRATIVE_RISK_ACCEPTED`. It MUST NOT be logged as `TERMINATION_CONFIRMED` or `WORKER_STOPPED`.
3. Invariant: `NEW_SESSION_OR_GENERATION != RESOLUTION_OF_PRIOR_UNCERTAIN_EXECUTION`. Creating a new session or advancing generation does not clear quarantine.

### Rationale
Because pinned AO `/send` has no deduplication key, assuming the message was lost risks dual-worker collision if the prompt actually arrived. Failing closed and quarantining guarantees safety.

### Rejected Alternatives
- **Blind Resend**: Catastrophic risk of duplicate execution; strictly prohibited.
- **Direct `DISPATCHED -> HUMAN_REQUIRED`**: Prohibited as an illegal transition edge in the canonical workflow state machine.

### Safety Invariant
An uncertain execution is never cleared without verifiable physical termination (Class A) or explicit, audited administrative risk acceptance (Class C).

### Persistence Impact
Updates `dispatch_operations.resolution_state` and appends quarantine audit events.

### State-Machine Impact
Follows valid canonical path: `DISPATCHED -> FAILED -> HUMAN_REQUIRED`.

### Restart Impact
Restart recovery automatically marks crashed `SEND_REQUESTED` operations as `FAILED` and quarantined.

### Auditability Impact
Complete audit trail distinguishing physical termination confirmation from administrative risk acceptance.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 12. D6 — Uncertain-Delivery Quarantine Guard

### Source / Canonical Facts
- Canonical graph allows `FAILED -> READY`. An unquarantined retry risks duplicate execution while uncertain execution runs.

### Proposal Revision-6 Recommendation
- Option D (Combination: Attempt disposition + Pair active lane lockout) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Option D (Double-Gated Quarantine Enforcement)**.
1. Enforcement is double-gated across two distinct persistence layers:
   - **Pair Lane Guard**: `worker_sessions.quarantine_state = 'QUARANTINED'` locked to `(pair_id, session_id)`.
   - **Attempt Guard**: `task_attempts.recovery_disposition = 'UNCERTAIN_DELIVERY_QUARANTINE'`.
2. Gating Invariant: Any dispatch preparation (`PrepareDispatch`) checks both flags. If either flag is set, dispatch is rejected with `ErrQuarantinedExecution`.
3. Clearance Protocol: Both flags are cleared simultaneously in a single transaction only when Class A (physical kill confirmed) or Class C (operator risk acceptance) resolution is formally executed.

### Rationale
Double-gating provides defense-in-depth: the lane guard prevents any task from using the tainted worker session, while the attempt guard prevents that specific task contract from being retried elsewhere until the uncertain attempt is accounted for.

### Rejected Alternatives
- **Option A (Pair metadata only)**: Single point of failure; risks retrying the task on another lane while old execution continues.
- **Option C (Attempt disposition only)**: Risks dispatching a different task onto the corrupted session.

### Safety Invariant
No dispatch can proceed on a Pair or TaskAttempt while an active quarantine flag exists.

### Persistence Impact
Adds `quarantine_state TEXT` to `worker_sessions` and `recovery_disposition TEXT` to `task_attempts`.

### State-Machine Impact
Blocks execution of `FAILED -> READY` while quarantine is unresolved.

### Restart Impact
Quarantine persists durably across crashes and restarts.

### Auditability Impact
Emits `QUARANTINE_IMPOSED` and `QUARANTINE_CLEARED` audit events.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 13. D7 — Pre-Send Admissibility

### Source / Canonical Facts
- Pinned AO SessionGuard prohibits writes on `blocked`, `exited`, `terminated`. `waiting_input` accepts input. `idle` is clean prompt. Generation mismatch breaks fence.

### Proposal Revision-6 Recommendation
- Strict whitelist (`idle`, `waiting_input` permitted; all others prohibited) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Enforce Strict Whitelist Admissibility for Pre-Send Verification**.
1. Immediately before transitioning from `DISPATCH_BOUND` to `SEND_REQUESTED`, the Supervisor inspects session activity via `GET /api/v1/sessions/{id}`.
2. Admissibility Rules:
   - `idle`: **SAFE_TO_SEND**. SessionGuard is authoritative startup safety gate; proceed to send.
   - `waiting_input`: **SAFE_TO_SEND**. Upstream allows instruction delivery; proceed to send.
   - `blocked`: **SEND_PROHIBITED**. Agent is stopped on tool permission or approval dialog; halt dispatch, await operator unblock or fail attempt.
   - `exited`: **SEND_PROHIBITED**. Agent process has exited; halt dispatch, require session restore.
   - `isTerminated: true`: **SEND_PROHIBITED**. Session dead; fail attempt (`DISPATCHED -> FAILED`).
   - `active`: **SEND_PROHIBITED**. Supervisor has not yet sent current task; session is executing foreign or lingering work. Halt dispatch, do not interleave prompts.
   - Generation Mismatch: **SEND_PROHIBITED**. Fence broken; fail attempt (`STALE_EXECUTION_GENERATION`).
3. Automated Nudging: Absolutely NO automated keystrokes, simulated Enter keys, or blind prompts may be sent to clear blocked dialogs.

### Rationale
A strict whitelist guarantees that prompts are delivered only when the upstream terminal is in a receptive state, completely preventing input corruption or shell command execution in exited panes.

### Rejected Alternatives
- **Permitting `active` before send**: Catastrophic risk of interleaving prompts mid-turn; rejected.
- **Automated unblock nudges**: Violates human supervision boundaries and risks unintended tool approvals; rejected.

### Safety Invariant
Task prompt `/send` is never called unless pre-send activity check confirms `idle` or `waiting_input` with matching generation.

### Persistence Impact
Rejection records audit event `PRE_SEND_ADMISSIBILITY_REJECTED`.

### State-Machine Impact
Admissibility failure keeps task in `DISPATCHED` (awaiting readiness) or transitions `DISPATCHED -> FAILED`.

### Restart Impact
Admissibility is re-verified cleanly upon any resumed dispatch saga.

### Auditability Impact
Activity snapshot at pre-send check is recorded in dispatch operation record.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 14. D8 — Missed Active Window

### Source / Canonical Facts
- `SEND_CONFIRMED` committed, but subsequent observation snapshot is `idle`. Does not prove whether prompt ran or was missed. Cannot auto-transition `DISPATCHED -> RUNNING -> REPORT_READY`.
- Strict Boundary: TASK-P03-003 does NOT perform report artifact reads, workspace file probing, or report validation (owned by TASK-P03-004 and P04).

### Proposal Revision-6 Recommendation
- Option C (Persist ambiguity disposition and hand off downstream) with Option A fail-closed fallback (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Adopt Option C (Persist Ambiguity Disposition and Hand Off Downstream to Authorized Phase)**.
1. When the Supervisor observes `idle` following `SEND_CONFIRMED` without having observed an intervening `active` state:
   - TASK-P03-003 does NOT assume failure and does NOT assume success.
   - TASK-P03-003 does NOT inspect workspace files or probe report artifacts.
   - It marks the attempt: `task_attempts.recovery_disposition = 'MISSED_ACTIVE_WINDOW_AMBIGUITY'`.
   - It transitions `DISPATCHED -> RUNNING` with the ambiguity disposition flag set.
2. Downstream Handoff: The task is passed to the downstream authorized phases (TASK-P03-004 workspace file transport and Phase P04 Evidence Collector).
   - If downstream phase verifies that a valid `WorkerReport` was generated matching this attempt's execution identity, the ambiguity is resolved as successful turn execution.
   - If downstream phase discovers no valid report or no modified files, downstream phase executes terminal failure (`RUNNING -> FAILED`, reason: `MISSED_ACTIVE_WINDOW_NO_OUTPUT`).
3. Fallback: If downstream phase handoff is disabled or unavailable, the Supervisor fails closed (`DISPATCHED -> FAILED`).

### Rationale
Short CLI commands or cached tool turns can execute and return to `idle` in sub-second intervals between polling ticks. Immediately failing the task would abort valid quick executions; immediately completing it would violate verification boundaries. Handing off with an explicit ambiguity flag preserves task boundaries and ensures full semantic verification downstream.

### Rejected Alternatives
- **Immediate Fail-Closed in TASK-P03-003**: Causes false-positive aborts on fast-completing commands; rejected.
- **Probing Report Files in TASK-P03-003**: Strict violation of component boundaries defined in PROPOSAL-P03-001; rejected.

### Safety Invariant
TASK-P03-003 never reads workspace files, parses reports, or transitions `REPORT_READY`.

### Persistence Impact
Updates `task_attempts.recovery_disposition`.

### State-Machine Impact
`DISPATCHED -> RUNNING` (with ambiguity flag).

### Restart Impact
Restart recovery preserves the ambiguity disposition flag for downstream inspection.

### Auditability Impact
Emits audit event `MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM`.

### Implementation Ownership
TASK-P03-003 lifecycle handoff; TASK-P03-004 / P04 report validation.

### External Approval Required
YES.

---

## 15. D9 — waiting_input Mapping

### Source / Canonical Facts
- Pinned AO `waiting_input` means agent process paused at an empty prompt awaiting instruction. Normalized observation is `AO_WAITING_INPUT`.

### Proposal Revision-6 Recommendation
- Remain in `RUNNING` + operator alert (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Preserve `TaskState = RUNNING`, Emit Diagnostic Telemetry (`AO_WAITING_INPUT_OBSERVED`), and Continue Polling**.
1. Observing `waiting_input` during task execution does NOT transition the task away from `RUNNING`.
2. The poller emits diagnostic telemetry `AO_WAITING_INPUT_OBSERVED` to the log and continues standard monitoring.
3. If `waiting_input` persists until the execution deadline expires, the deadline monitor triggers terminal timeout failure.

### Rationale
Interactive CLI agents frequently pause briefly at empty prompts between tool sub-turns or during setup. Treating this as a failure or blockage causes spurious workflow disruptions.

### Rejected Alternatives
- **Transition to `BLOCKED`**: Incorrect; `waiting_input` is not a permission block.
- **Transition to `FAILED`**: Spurious failure on ordinary prompt pauses; rejected.

### Safety Invariant
The poller never injects automated text or keystrokes when `waiting_input` is observed.

### Persistence Impact
Audit log entries only.

### State-Machine Impact
Zero transition (task remains `RUNNING`).

### Restart Impact
Reconciled as `RUNNING` with ongoing observation.

### Auditability Impact
Diagnostic telemetry records observation time and duration.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 16. D10 — blocked Mapping

### Source / Canonical Facts
- Pinned AO `blocked` means agent stopped on tool permission or human approval dialog. Automated input is suppressed by SessionGuard (`SuppressedAwaitingUser`).
- Normalized observation is `AO_BLOCKED_DECISION`. Under canonical state machine, `BLOCKED` is non-terminal (allows `BLOCKED -> HUMAN_REQUIRED`).

### Proposal Revision-6 Recommendation
- Transition `RUNNING -> BLOCKED` while retaining open `TaskAttempt` (`ended_at = NULL`) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Transition `RUNNING -> BLOCKED`, Retain Open TaskAttempt (`ended_at = NULL`), Halt Poller Input, and Await Operator Resolution**.
1. Upon observing `blocked` during `RUNNING`:
   - Transition TaskState: `RUNNING -> BLOCKED` (reason: `AO_BLOCKED_DECISION`).
   - Attempt Status: `TaskAttempt.ended_at` remains `NULL`. The attempt remains open because execution is suspended awaiting human approval, not terminated.
   - Observation Poller: Halts automated progress polling; transitions to waiting for operator decision or unblock event.
   - Audit Event: Emits `WORKER_BLOCKED_DECISION` recording the blocked tool dialog.
2. Resumption / Escalation:
   - If the operator resolves the dialog upstream, the poller detects activity change and resumes execution.
   - If the operator rejects or cannot resolve, workflow transitions `BLOCKED -> HUMAN_REQUIRED`.

### Rationale
In the canonical state machine, `BLOCKED` is a non-terminal waiting state. Setting `ended_at = now` would terminate the attempt, making it impossible to resume execution under the same attempt identity if the operator approves the tool permission.

### Rejected Alternatives
- **Terminate Attempt (`ended_at = now`)**: Destroys attempt lineage for resumable permissions; rejected.
- **Ignore `blocked` and Remain in `RUNNING`**: Leaves the Supervisor blind to an agent waiting indefinitely for user input; rejected.

### Safety Invariant
`TaskAttempt.ended_at` is set strictly for terminal failure or completion transitions, never for non-terminal `BLOCKED`.

### Persistence Impact
TaskState updated to `BLOCKED`. `ended_at` remains `NULL`.

### State-Machine Impact
Executes canonical edge `RUNNING -> BLOCKED`.

### Restart Impact
Restart recovery identifies `BLOCKED` tasks and preserves open attempt state.

### Auditability Impact
Detailed audit event documenting blocked tool permission.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 17. D11 — Intentional Stop Provenance

### Source / Canonical Facts
- Pinned AO wire route is `POST /api/v1/sessions/{sessionId}/kill` (`sessions.go:206`). HTTP `DELETE` does not exist.
- HTTP `200 OK` indicates that AO accepted the kill signal; it does NOT prove process termination.

### Proposal Revision-6 Recommendation
- 3-stage durable stop operation lifecycle (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement 3-Stage Durable Stop Operation Lifecycle Bound to `(operation_id, pair_id, task_id, attempt_id, session_id, terminal_generation)` via `POST /kill`**.
1. Stop operations execute through three durable stages:
   - `STOP_REQUESTED`: Durably committed to SQLite immediately *before* calling `POST /api/v1/sessions/{sessionId}/kill`.
   - `STOP_CALL_SUCCEEDED` / `STOP_CALL_FAILED`: Committed immediately upon receiving HTTP response (`KillSessionResponse`).
   - `STOP_TERMINATION_CONFIRMED`: Committed ONLY when a subsequent authoritative observation snapshot returns `isTerminated == true` matching the execution generation.
2. All stages are bound to: `(operation_id, pair_id, task_id, attempt_id, session_id, terminal_generation, timestamp, actor)`.
3. Invariant: Terminal state change (`RUNNING -> FAILED`, reason: `WORKER_STOPPED`) and attempt `ended_at = now` are committed strictly upon `STOP_TERMINATION_CONFIRMED`.

### Rationale
Process teardown and resource deallocation in operating systems is asynchronous. Assuming immediate termination upon HTTP 200 causes race conditions if files are still being flushed or locks held.

### Rejected Alternatives
- **Synchronous Termination Assumption on HTTP 200**: Race condition hazard; rejected.
- **HTTP `DELETE` Wire Route**: Non-existent in upstream API; rejected.

### Safety Invariant
`WORKER_STOPPED` is recorded only after authoritative `isTerminated == true` is verified.

### Persistence Impact
Creates `stop_operations` tracking table or structured audit entries.

### State-Machine Impact
Drives terminal failure `RUNNING -> FAILED` (reason: `WORKER_STOPPED`).

### Restart Impact
Unfinished stop operations are resumed and checked on restart.

### Auditability Impact
Full multi-stage provenance of stop operations recorded.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 18. D12 — Atomic Terminal Transition & ended_at Ownership

### Source / Canonical Facts
- ADR-015 specifies SQLite engine and transactional audit log. Canonical auditability requires crash-consistent state transitions.
- SQLite Invariant: Absolutely ZERO network calls inside database transactions.

### Proposal Revision-6 Recommendation
- Unified StateStore atomic transition method (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Unified StateStore Atomic Terminal Transition Method (`AtomicTerminalTransition`)**.
1. For terminal failure transitions (`RUNNING -> FAILED`, `DISPATCHED -> FAILED`):
   - Atomically committed in a single SQLite transaction:
     ```text
     1. tasks.state CAS update (old_state -> FAILED)
     2. task_attempts.ended_at = now
     3. task_attempts.terminal_error = error_details
     4. audit_events row append (TASK_STATE_TRANSITION)
     5. dispatch/recovery disposition update
     ```
2. For non-terminal transitions (`RUNNING -> BLOCKED`):
   - Atomically committed: TaskState CAS update to `BLOCKED` + audit event append.
   - `task_attempts.ended_at` remains `NULL` (per D10).
3. Success Closure (`RUNNING -> REPORT_READY`):
   - Strictly NOT owned by TASK-P03-003. Owned exclusively by Phase P04 Evidence Collector upon report validation.
4. Transaction Rule: All upstream AO queries occur *before* opening the SQLite transaction. The transaction performs only local SQLite writes and commits immediately.

### Rationale
Crash consistency requires that the database never contains a failed task with an open attempt, or an ended attempt without an audit event. A unified method guarantees atomicity without risking SQLite database lock contention.

### Rejected Alternatives
- **Separate Un-coordinated Store Calls**: Risks partial writes on crash; rejected.
- **Network Calls Inside Transactions**: Violates ADR-015 and causes database deadlocks; strictly prohibited.

### Safety Invariant
Zero network operations inside SQLite transaction blocks.

### Persistence Impact
Adds unified method to `internal/store` API.

### State-Machine Impact
Executes all terminal state machine failures.

### Restart Impact
Partial writes are impossible; SQLite rollbacks on crash leave state completely clean.

### Auditability Impact
Atomic pairing of state change and audit event guaranteed.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 19. D13 — Restart Scanner Architecture

### Source / Canonical Facts
- Daemon crash or power loss leaves in-flight tasks in `DISPATCHED` or `RUNNING` with open attempts (`ended_at IS NULL`).

### Proposal Revision-6 Recommendation
- Synchronous startup recovery sweep for in-flight tasks and unresolved operations (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Synchronous Startup Recovery Sweep for In-Flight Tasks and Operations with Non-Blocking Failure Handling**.
1. On Supervisor daemon startup, prior to accepting incoming client API requests:
   - Query StateStore for all tasks in `DISPATCHED` or `RUNNING` state.
   - Query for unresolved `dispatch_operations` (`stage = SEND_REQUESTED`) and `stop_operations` (`stage = STOP_REQUESTED`).
2. AO Daemon Availability Check:
   - Probe AO availability via `GET /api/v1/sessions`.
   - **If AO is Unreachable**: Do NOT fail tasks. Mark in-flight tasks with `recovery_disposition = 'RECOVERY_PENDING'`, do not block daemon startup, and schedule background reconciliation retries.
   - **If AO is Reachable**: Reconcile each in-flight task against public `SessionView`:
     - If session missing (404) or `isTerminated == true`: execute `AtomicTerminalTransition` (`-> FAILED`, `ended_at = now`, reason: `WORKER_TERMINATION_UNKNOWN` or `WORKER_STOPPED`).
     - If `terminalGeneration` mismatch: execute `AtomicTerminalTransition` (`-> FAILED`, reason: `STALE_EXECUTION_GENERATION`).
     - If `stage == SEND_REQUESTED`: execute D5 uncertain delivery failure and quarantine.
     - If session active/idle with matching generation: resume standard lifecycle observation.

### Rationale
A synchronous startup sweep ensures that lingering crash states are reconciled before new tasks can be dispatched to those Pairs, completely eliminating race conditions between old executions and new work.

### Rejected Alternatives
- **Lazy Polling-Driven Reconciliation**: Leaves a race window where new tasks could be dispatched to tainted Pairs before the poller wakes up; rejected.
- **Blind Termination of All In-Flight Tasks**: Aborts long-running worker tasks that may have survived the Supervisor daemon restart; rejected.

### Safety Invariant
No new task can be dispatched on a Pair until that Pair's in-flight task status has been reconciled against upstream reality.

### Persistence Impact
Recovery audit events logged upon startup.

### State-Machine Impact
Reconciles interrupted states to valid canonical destinations.

### Restart Impact
Governs the complete restart recovery lifecycle of the control plane.

### Auditability Impact
Emits `STARTUP_RECOVERY_SWEEP_STARTED` and `STARTUP_RECOVERY_SWEEP_COMPLETED` audit events.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 20. WorkerSession / TaskAttempt Execution Identity

The architecture formally establishes the execution identity model:
```text
Execution Identity = (session_id, terminal_generation)
```
1. `session_id`: The upstream AO UUID identifying the ConPTY pane and workspace worktree.
2. `terminal_generation`: The upstream monotonic execution generation fencing the session against terminal restarts or resets.
3. Every `TaskAttempt` snapshot permanently binds this tuple. Any upstream observation bearing a mismatched session or generation is rejected as foreign, eliminating cross-turn attribution hazards.

---

## 21. Agy V1 Runtime Generation Fence

1. For V1, the primary worker harness is `agy` running in interactive TUI mode inside an AO-managed ConPTY pane.
2. The authoritative execution fence is upstream AO's `terminalGeneration` (exposed as `terminalGeneration` on `SessionView` and matching `RuntimeLaunchID`).
3. **Scope Guard**: This fence applies specifically to V1 `agy` TUI mode. It is NOT generalized to non-TUI headless modes or arbitrary processes, which remain subject to future phase governance.

---

## 22. Lifecycle Observation Matrix

The complete lifecycle evaluation matrix governing TASK-P03-003:

| TaskState | Saga Stage | Observed Activity | Generation Match | Proven Facts | Unproven Facts | Transition | ended_at Action | Quarantine Action |
|---|---|---|---|---|---|---|---|---|
| `DISPATCHED` | `DISPATCH_BOUND` | `idle` | Match | Pre-send ready | Execution occurred | None | Keep NULL | None |
| `DISPATCHED` | `DISPATCH_BOUND` | `waiting_input` | Match | Waiting input | Execution occurred | None | Keep NULL | None |
| `DISPATCHED` | `DISPATCH_BOUND` | `active` | Match | Foreign/lingering turn | Current task sent | None | Keep NULL | `PRE_SEND_BLOCKED` |
| `DISPATCHED` | `DISPATCH_BOUND` | `blocked` | Match | Dialog blocked | Current task sent | None | Keep NULL | `PRE_SEND_BLOCKED` |
| `DISPATCHED` | `DISPATCH_BOUND` | `exited` | Match | Process exit signal | Termination/crash | None | Keep NULL | `PRE_SEND_BLOCKED` |
| `DISPATCHED` | `DISPATCH_BOUND` | `isTerminated: true` | Match | Session dead | Cause of death | `-> FAILED` | Set `now` | None |
| `DISPATCHED` | `SEND_REQUESTED` | Any | Match/Mismatch | Delivery unknown | Delivery success | `-> FAILED` | Set `now` | **`UNCERTAIN_DELIVERY_QUARANTINE`** |
| `DISPATCHED` | `SEND_CONFIRMED` | `active` | Match | Prompt accepted, running | Task completion | `-> RUNNING` | Keep NULL | None |
| `DISPATCHED` | `SEND_CONFIRMED` | `idle` | Match | Turn completed quickly | Report validity | `-> RUNNING` (D8 flag) | Keep NULL | `MISSED_ACTIVE_WINDOW_AMBIGUITY` |
| `DISPATCHED` | `SEND_CONFIRMED` | `waiting_input` | Match | Agent at prompt | Failure | None | Keep NULL | Diagnostic event |
| `DISPATCHED` | `SEND_CONFIRMED` | `blocked` | Match | Blocked on dialog | Terminal failure | None (D10) | Keep NULL | Diagnostic event |
| `DISPATCHED` | `SEND_CONFIRMED` | `exited` | Match | Process exited | Session dead | None | Keep NULL | Process exit rule |
| `DISPATCHED` | `SEND_CONFIRMED` | `isTerminated: true` | Match | Session terminated | Crash vs clean | `-> FAILED` | Set `now` | None |
| `RUNNING` | `SEND_CONFIRMED` | `active` | Match | Active tool execution | Task completion | None | Keep NULL | None |
| `RUNNING` | `SEND_CONFIRMED` | `idle` | Match | Turn complete | Report valid (`AO_IDLE != REPORT_READY`) | None (for P04) | Keep NULL | P04 handoff |
| `RUNNING` | `SEND_CONFIRMED` | `waiting_input` | Match | Paused at prompt | Failure | None (D9) | Keep NULL | Diagnostic event |
| `RUNNING` | `SEND_CONFIRMED` | `blocked` | Match | Blocked on tool permission | Attempt failure | `-> BLOCKED` (D10) | Keep NULL | Diagnostic event |
| `RUNNING` | `SEND_CONFIRMED` | `exited` | Match | Process exited | Session dead | None | Keep NULL | Process exit rule |
| `RUNNING` | `SEND_CONFIRMED` | `isTerminated: true` | Match | Session dead | Crash vs clean | `-> FAILED` | Set `now` | None |
| `RUNNING` | `SEND_CONFIRMED` | 404 Not Found | N/A | Session purged | Purge reason | `-> FAILED` | Set `now` | None |
| `RUNNING` | `SEND_CONFIRMED` | AO Unavailable | N/A | Daemon unreachable | Worker death | None | Keep NULL | Retry scheduled |

---

## 23. Crash / Restart Matrix

| Crash Point | Host Failure Window | Persisted State on Disk | Detection on Startup | Reconciled Action |
|---|---|---|---|---|
| **T0** | Before session spawn | Pair idle, no session | Normal startup | None; Pair available for provisioning |
| **T1** | During spawn wire call | Audit intent logged | Session uncommitted | Mark provisioning failed; orphan reaper cleans AO |
| **T2** | AO spawned, before DB commit | AO session active | No DB record | Fail-closed; orphan reaper cleans unreferenced session |
| **T3** | Session committed, before dispatch | `worker_sessions` active | Valid idle session | Ready for dispatch |
| **T4** | `DISPATCH_BOUND` committed | `tasks.state = DISPATCHED` | Saga = `DISPATCH_BOUND` | Check pre-send admissibility; execute `/send` |
| **T5** | `SEND_REQUESTED` committed, crash during `/send` | Saga = `SEND_REQUESTED` | Delivery unknown | `-> FAILED`, `ended_at = now`, quarantine, `-> HUMAN_REQUIRED` |
| **T6** | `SEND_CONFIRMED` committed | Saga = `SEND_CONFIRMED` | Prompt confirmed | Check AO activity; resume observation |
| **T7** | `RUNNING` state, daemon power loss | `tasks.state = RUNNING` | In-flight attempt | Reconcile against AO public SessionView |

---

## 24. StateStore Transaction Boundaries

All operations adhere to ADR-015:
1. `PrepareDispatch`: Single SQLite transaction committing `tasks.state = DISPATCHED`, allocating `task_attempts`, binding `session_id`, and creating `dispatch_operations` in `DISPATCH_BOUND`.
2. `RecordSendRequested`: Single SQLite transaction updating `dispatch_operations.stage = SEND_REQUESTED` immediately before network call.
3. `RecordSendConfirmed`: Single SQLite transaction updating `dispatch_operations.stage = SEND_CONFIRMED` immediately after HTTP 200.
4. `AtomicTerminalTransition`: Single SQLite transaction committing TaskState update, `task_attempts.ended_at`, error details, and audit log.
5. Strict Boundary: No network call or AO interaction is ever permitted inside these transaction blocks.

---

## 25. Audit Event / Operation Provenance

The following structured audit event types are established:
- `PAIR_SESSION_PROVISIONED`: Infrastructure worktree and session bound.
- `TASK_DISPATCH_BOUND`: Task contract allocated and attempt bound.
- `DISPATCH_SEND_REQUESTED`: Pre-effect dispatch intent recorded.
- `DISPATCH_SEND_CONFIRMED`: Upstream write success recorded.
- `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED`: Delivery unknown; quarantine active.
- `QUARANTINE_RESOLVED_PHYSICAL`: Verified termination via `/kill` confirmed.
- `QUARANTINE_RESOLVED_ADMINISTRATIVE`: Explicit operator risk acceptance logged.
- `WORKER_BLOCKED_ON_DECISION`: Tool permission dialog observed.
- `STOP_OPERATION_REQUESTED`: Kill signal intent recorded.
- `STOP_OPERATION_CONFIRMED`: Authoritative termination confirmed.
- `MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM`: Ambiguity handoff to P04 logged.

---

## 26. ADR-012 Amendment Text

**Formal Amendment to ADR-012 Section 10**:

> *"ADR-012 Section 10 is hereby amended as follows:*
>
> *1. Pair Worker Session Provisioning (`createWorkerSession`): Provisioning or reusing the execution environment, worktree, and sandbox for a Pair is classified as infrastructure management. It MAY occur prior to task dispatch, establishing a durable `WorkerSession` binding on the Pair.*
>
> *2. Task Execution Side Effects (`sendTask` / `/send`): Transmitting a task contract and prompt to an agent is classified as task execution. It occurs strictly AFTER `TaskAttempt` allocation and the durable `DISPATCHED` state record are committed in the State Store.*
>
> *All other provisions of ADR-012 remain in full force."*

---

## 27. Persistence Impact

1. Schema Updates:
   - `worker_sessions`: Table created with unique `pair_id`, `session_id`, `worktree_path`, `terminal_generation`, `quarantine_state`.
   - `task_attempts`: Columns added: `session_id TEXT`, `terminal_generation TEXT`, `recovery_disposition TEXT`.
   - `dispatch_operations`: Table created with `operation_id`, `attempt_id`, `pair_id`, `task_id`, `session_id`, `terminal_generation`, `stage`, `resolution_state`.
   - `stop_operations`: Table created with `operation_id`, `session_id`, `terminal_generation`, `stage`.
2. Migrations: Implementation requires a schema migration executed under future authorized task contract. Zero migrations executed in this draft.

---

## 28. Canonical Documentation Impact

Upon formal External Supervisor acceptance of ADR-016, the following canonical documents will be reconciled:
1. `docs/04_ARCHITECTURE.md`: Update sequence diagrams to reflect decoupled pair session provisioning and durable 3-stage dispatch saga.
2. `docs/05_DOMAIN_MODEL.md`: Formalize Model A persistence relationship and snapshot columns.
3. `docs/06_WORKFLOW_STATE_MACHINE.md`: Cross-reference `RUNNING -> BLOCKED` attempt open semantics.
4. `docs/08_TASK_CONTRACT.md`: Document attempt execution identity snapshot binding.
5. `docs/12_UPSTREAM_INTEGRATION.md`: Document wire route `/kill`, generation fence, and strict pre-send admissibility whitelist.
6. `docs/14_FAILURE_RECOVERY.md`: Document quarantine guard, Class A/C clearance rules, and restart recovery scanner.
7. `docs/22_MODULE_PROVENANCE.md`: Register new store entities.

---

## 29. TASK-P03-003 Implementation Boundary

`TASK-P03-003` owns exclusively:
- Domain persistence of `worker_sessions`, `dispatch_operations`, and `task_attempts` snapshot fields.
- 3-stage dispatch saga coordination.
- Pre-send admissibility evaluation.
- Observation poller activity mapping (`active`, `idle`, `waiting_input`, `blocked`, `exited`).
- Intentional stop provenance lifecycle via `POST /kill`.
- Unified StateStore `AtomicTerminalTransition` method.
- Synchronous startup recovery sweep.

`TASK-P03-003` strictly does NOT own:
- Workspace file fetching or directory tree reading (`TASK-P03-004`).
- `WorkerReport` schema validation or claim parsing (`P04`).
- `worker_report_raw` persistence (`P04`).
- Transition `RUNNING -> REPORT_READY` (`P04`).

---

## 30. Explicit Non-Goals

- NO production code implementation in this draft (`internal/**/*.go`).
- NO database schema migration execution in this draft.
- NO canonical document mutation in this draft.
- NO release of `TASK-P03-003`.
- NO invented numeric timeouts, intervals, or poll rates.
- NO automated nudges into agent dialogs.
- NO blind retries or synthetic heartbeats.
- NO SSE `/api/v1/events` integration in this task.

---

## 31. Consequences

### Positive
- Completely eliminates duplicate code execution risks caused by unknown delivery.
- Resolves relational domain contradictions without introducing cyclic foreign keys.
- Decouples sandbox provisioning from task dispatch, enabling warm session reuse.
- Enforces strict component boundaries and clean testability.

### Negative / Trade-offs
- Unknown-delivery crashes require explicit operator resolution (Class A kill or Class C risk acceptance) before tasks can be retried.
- Adding dedicated operations tables slightly increases database schema surface.

---

## 32. Rejected Alternatives

- **Model B (Historical Table + Pointer)**: Rejected due to cyclic FKs in SQLite.
- **Model C (Join Table)**: Rejected due to unnecessary schema complexity.
- **Blind Resend on Crash**: Rejected as critical safety violation.
- **Automatic Heuristic Session Adoption**: Rejected as cross-session security risk.
- **Direct `DISPATCHED -> HUMAN_REQUIRED`**: Rejected as invalid state machine transition.
- **`ActivityExited` Automatic Failure**: Rejected because managed session remains open.
- **Report Probing in TASK-P03-003**: Rejected as strict component boundary violation.

---

## 33. External Approval Gate

```text
ADR_016 = DRAFT_READY_FOR_EXTERNAL_AUDIT
ADR_016_ACCEPTANCE = NOT_YET_GRANTED
TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_PENDING_ADR_016_APPROVAL
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_ADR_016_AUDIT
```

Execution is halted awaiting independent External Supervisor audit of this proposed ADR.
