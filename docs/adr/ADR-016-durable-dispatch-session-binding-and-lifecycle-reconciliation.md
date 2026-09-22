# ADR-016: Durable Worker Session Binding, Dispatch Saga, and Lifecycle Reconciliation (Revision 3)

> **Status**: `PROPOSED  -  PENDING EXTERNAL REAUDIT` (Revision 3)
> **Date**: 2026-09-22 (Revision 1: 2026-09-23, Revision 2: 2026-09-23, Revision 3: 2026-09-23)
> **Authority**: Architecture Decision Record
> **Proposal Authority**: `PROPOSAL-P03-002 Revision 6` (`EXTERNAL_APPROVED`, commit `8709af4b6aaf8613c69a10514971e320e3f90048`)
> **Deciders**: External Supervisor, Engineering Team
> **Amends**: ADR-012 (Section 10, where explicitly selected by this draft)
> **Related**: ADR-002, ADR-009, ADR-011, ADR-015
> **Implementation Guard**: `TASK-P03-003` remains `NOT_RELEASED`, `P03_CODE = HELD_PENDING_ADR_016_APPROVAL`.

---

## 1. Context

In Phase P01, runtime proof activities demonstrated that the Agent Orchestrator (`AO`) daemon provides process isolation, ConPTY virtual terminal management, git worktree lifecycle integration, and public session monitoring (`docs/audits/P01_A_AO_RUNTIME_PROOF.md`). In Phase P02, the Supervisor Control Plane domain model, state machine, and SQLite persistence engine were formally implemented and verified (`docs/audits/P02_FINAL_AUDIT = EXTERNAL_AUDIT_APPROVED`).

During pre-code architectural analysis for `TASK-P03-003` (Lifecycle Reconciliation and State Transitions), multiple structural gaps and contract contradictions were identified between canonical specifications, accepted ADRs, and upstream AO server behavior:
1. **Relational Tension**: Canonical `docs/05_DOMAIN_MODEL.md` mandates `Pair "1" o-- "1" WorkerSession`, while `docs/14_FAILURE_RECOVERY.md` requires immutable auditability of historical `TaskAttempt` execution identities across session recreation.
2. **Missing Upstream Idempotency**: Upstream AO v0.13.0 endpoints (`POST /api/v1/sessions` and `POST /api/v1/sessions/{id}/send`) lack client-supplied idempotency keys or deduplication guarantees (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`, `PINNED_SEND_IDEMPOTENCY = ABSENT`).
3. **Uncertain Delivery Window**: A host crash or network failure after dispatch intent is recorded but before delivery confirmation leaves delivery outcome unknown; blind resend risks catastrophic duplicate code execution.
4. **ADR-012 Conflict**: Accepted ADR-012 Section 10 stated that external calls (`createWorkerSession`, `sendTask`) occur strictly *after* durable `DISPATCHED`, conflicting with pre-provisioning execution environments for Pairs.
5. **Observability Boundary**: Upstream AO provides no public worker crash evidence, exit codes, or heartbeat mechanisms in its pinned baseline (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`).

`PROPOSAL-P03-002 Revision 6` resolved these gaps through extensive architectural analysis and was formally approved by External Supervisor Re-Audit 006 (commit `8709af4b6aaf8613c69a10514971e320e3f90048`).

Initial draft of ADR-016 (commit `928b0d110058d8ca90d8d9b68d9aa478c1ee089a`) was independently audited by External Supervisor in `docs/audits/P03_ADR_016_EXTERNAL_AUDIT.md`, recording nine findings (`ADR16R1-001` through `ADR16R1-009`) requiring Revision 1. This document establishes Revision 1, surgically addressing all audit findings while remaining strictly within the approved architectural choice space of Proposal Revision 6.

---

## 2. Problem Statement

How does the Supervisor Control Plane achieve deterministic, crash-consistent worker session binding, task dispatching, and lifecycle reconciliation under an upstream execution engine that lacks caller-provided idempotency, provides no public process exit evidence, and separates sandbox management from task execution?

Specifically, the architecture must resolve:
1. How to durably bind worker sessions to Pairs while maintaining immutable execution lineage for individual task attempts.
2. How to sequence session creation and task dispatch without violating state machine invariants or coupling sandbox provisioning failures to task dispatch.
3. How to represent dispatch progress durably across crash windows where network delivery cannot be verified.
4. How to prevent duplicate execution when delivery outcome is unknown without violating canonical state machine transitions.
5. How to reconcile upstream activity observations (`active`, `idle`, `waiting_input`, `blocked`, `exited`) into canonical TaskStates without violating the canonical state graph or over-interpreting unproven signals.
6. How to execute intentional stop operations with cryptographic-grade provenance and without race conditions.
7. How to guarantee crash-consistent state transitions in SQLite without performing network calls inside database transactions.
8. How to perform startup restart reconciliation without assuming success or premature failure.

---

## 3. Decision Drivers

- **Automated Duplicate Dispatch Prevention (Fail-Closed)**: Automated retry or re-dispatch is strictly prohibited (`AUTOMATED_DUPLICATE_DISPATCH_PREVENTION = FAIL_CLOSED`) whenever prior execution state remains unverified (`NEW_SESSION_OR_GENERATION != RESOLUTION_OF_PRIOR_UNCERTAIN_EXECUTION`). Class C administrative override provides an explicit, auditable human risk exception that knowingly accepts residual duplicate execution risk when upstream physical state cannot be proven.
- **Strict Canonical State Machine Adherence**: All transitions must execute via valid edges in the canonical workflow graph (`docs/06_WORKFLOW_STATE_MACHINE.md`). No invented TaskStates or illegal transitions (such as `BLOCKED -> RUNNING`) are permitted.
- **Transactional SQLite Invariants**: In accordance with ADR-015, database transactions must remain short, local, and synchronous. Absolutely NO network or AO API calls are permitted inside SQLite transactions.
- **Strict Component Boundaries**: Per `PROPOSAL-P03-001`, TASK-P03-003 owns lifecycle observation and state transitions only. TASK-P03-004 owns raw workspace file transport. Phase P04 owns WorkerReport semantic ingestion and validation. TASK-P03-003 must never read files, evaluate semantic report validity, or transition `REPORT_READY`.
- **Honest Observability**: The Supervisor logs only what is positively proven by authoritative evidence. Absences of evidence must never be synthesized into facts (e.g., termination without cause evidence is logged as `WORKER_TERMINATION_UNKNOWN`, never `WORKER_CRASHED`).
- **Configurable Operational Policies**: Numeric operational parameters (timeouts, intervals, deadlines) remain strictly unhardcoded (`UNSET`), requiring caller injection using existing approved policy surfaces.

---

## 4. Pinned AO Public Facts

All architecture decisions in this ADR are derived strictly from the verified behavior of `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`):
1. **Spawn Wire Route**: `POST /api/v1/sessions` generates an upstream session, allocates a git worktree, and spawns a ConPTY terminal. Returns `201 Created` with a server-generated UUID. Lacks caller-provided idempotency keys or correlation fields (`PINNED_SPAWN_IDEMPOTENCY = ABSENT`).
2. **Send Wire Route**: `POST /api/v1/sessions/{id}/send` writes text/prompt to the terminal. Returns `200 OK` on write acceptance. Lacks caller-provided message IDs or idempotency keys (`PINNED_SEND_IDEMPOTENCY = ABSENT`). HTTP 200 proves only that the upstream write was accepted; it does NOT prove the agent entered `ActivityActive`, does NOT prove the prompt was semantically consumed, and does NOT prove task execution completed.
3. **Stop Wire Route**: `POST /api/v1/sessions/{sessionId}/kill` terminates the session terminal and marks it terminated (`isTerminated = true`). Returns `200 OK` (`KillSessionResponse`). HTTP `DELETE` does NOT exist in the upstream API.
4. **Session Observation Wire Route**: `GET /api/v1/sessions/{id}` returns the public `SessionView` struct.
5. **Activity Taxonomy**: Exactly five (5) activity states exist in pinned AO: `active`, `idle`, `waiting_input`, `blocked`, `exited`.
6. **SessionGuard Rules**: The upstream session guard refuses automated input writes when activity is `blocked` (`SuppressedAwaitingUser`), `exited` (`SuppressedExited`), `terminated` (`SuppressedTerminated`), or during initial startup (`ErrStartupPending`).
7. **Execution Generation**: Pinned AO exposes `SessionView.TerminalGeneration` as an opaque `string` (mapping to `Metadata.RuntimeLaunchID`). It serves as an opaque renderer/runtime launch fence for V1 `agy` TUI sessions. It is NOT an integer counter, NOT a monotonic number, and CANNOT be compared using arithmetic ordering (`>` / `<`). Comparisons are strictly equality (`MATCH` vs `MISMATCH`).
8. **Diagnostic Timestamp**: `lastActivityAt` is an internal diagnostic timestamp updated on terminal activity. It is NOT a heartbeat, NOT a dispatch acknowledgement, and NOT proof of current attempt execution.
9. **No Public Crash Evidence**: Pinned AO exposes no OS exit codes, signals, or process crash logs on its public API (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`). `ActivityExited` denotes process termination inside an open session, which alone does NOT prove worker crash or session termination.
10. **Kill Generation Fence Absence (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`)**: Pinned route `POST /api/v1/sessions/{sessionId}/kill` (`sessions.go:206`) takes session identity only. It accepts NO `expectedGeneration`, NO `terminalGeneration` request body field, NO `If-Match` header, and NO CAS token. Calling `GET` before `POST /kill` provides a client-side `GENERATION_PRECHECK` only, NOT an atomic generation fence (`GENERATION_PRECHECK != GENERATION_FENCE`). A TOCTOU window exists where generation can change between observation and mutation.
11. **Worktree Path Public Availability Absence (`WORKTREE_PATH_PUBLIC_AVAILABILITY = UNAVAILABLE_IN_PINNED_V1`)**: While pinned AO internally stores `SessionMetadata.WorkspacePath`, the field is marked `json:"-"` in `domain.Session` and is omitted from public `SessionView` and `SpawnSessionResponse`. The approved Supervisor adapter (`WorkerStatus`, `CreateWorkerSessionResult`) exposes no workspace path. Therefore, workspace worktree paths cannot be verified or populated through public AO contracts in V1.

---

## 5. Canonical Constraints

1. **Architecture Decision Hierarchy**: Per `docs/24_CHANGE_GOVERNANCE.md`, confirmed user requirements and canonical architecture supersede worker proposals. Modifications to canonical documents require formal external approval.
2. **TaskState Graph Immutability**: The canonical TaskState graph (`internal/workflow/state_machine.go`) consists strictly of 25 graph edges: 1 initial pseudo-edge, 22 executable domain transitions, and 2 terminal pseudo-edges.
   - **Initial Pseudo-Edge (1)**:
     `[*] -> DRAFT`
   - **Executable Domain Transitions (22)**:
     1. `DRAFT -> READY`
     2. `DRAFT -> CANCELLED`
     3. `READY -> DISPATCHED`
     4. `READY -> CANCELLED`
     5. `DISPATCHED -> RUNNING`
     6. `DISPATCHED -> FAILED`
     7. `RUNNING -> REPORT_READY`
     8. `RUNNING -> FAILED`
     9. `RUNNING -> BLOCKED`
     10. `REPORT_READY -> EVIDENCE_READY`
     11. `REPORT_READY -> FAILED`
     12. `EVIDENCE_READY -> REVIEWING`
     13. `EVIDENCE_READY -> FAILED`
     14. `REVIEWING -> APPROVED`
     15. `REVIEWING -> REVISION_REQUIRED`
     16. `REVIEWING -> BLOCKED`
     17. `REVISION_REQUIRED -> READY`
     18. `BLOCKED -> HUMAN_REQUIRED`
     19. `FAILED -> HUMAN_REQUIRED`
     20. `FAILED -> READY`
     21. `HUMAN_REQUIRED -> DRAFT`
     22. `HUMAN_REQUIRED -> CANCELLED`
   - **Terminal Pseudo-Edges (2)**:
     `APPROVED -> [*]`
     `CANCELLED -> [*]`
   - **Exact Accounting (Finding `ADR16R3-001`)**: The 22 executable domain transitions are distinct from the initial pseudo-edge `[*] -> DRAFT`. Transition `HUMAN_REQUIRED -> CANCELLED` is domain transition #22, not a pseudo-edge. Total graph edge count is strictly 25.
   - **Normal Forward Flow**: Strictly `DRAFT -> READY -> DISPATCHED -> RUNNING -> REPORT_READY -> EVIDENCE_READY -> REVIEWING -> APPROVED`. There is **ZERO** direct `RUNNING -> REVIEWING` edge.
   - **Edge Immutability**: Zero new TaskStates may be added. There is NO `BLOCKED -> RUNNING` edge. All saga stages, quarantine flags, and recovery dispositions must reside in relational operation tables and metadata columns, never in invented TaskStates or illegal graph edges.
3. **Contract Immutability**: Per ADR-010 and ADR-012, once dispatched, a `TaskContract` revision is permanently immutable.

---

## 6. Decision Summary

The thirteen core decisions (D1 through D13) established by this ADR are summarized in the decision completeness matrix below:

### D1 - D13 Architectural Decision Completeness Table

| Decision ID | Selected Proposed Decision | Primary Rationale | Rejected Alternative(s) | Persistence Impact | TaskState Impact | External Approval Status |
|---|---|---|---|---|---|---|
| **D1** | **Model A: Mutable Current Session with Nullable WorktreePath, Independent Quarantine State & 1:1 Enforcement** | Preserves canonical `Pair 1 o-- 1 WorkerSession` with `UNIQUE(session_id)`; `worktree_path TEXT NULL` acknowledges public AO unavailability; separates runtime lifecycle `status` from `quarantine_state`; immutable attempt snapshot columns guarantee historical auditability. | Model B (Historical table + active pointer); Model C (Join table); Mandatory worktree_path from unavailable field. | Adds snapshot columns to `task_attempts`; creates `worker_sessions` table with unique constraints on both `pair_id` and `session_id`. | None. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D2** | **Formally Amend ADR-012 Section 10** | Decouples Pair-scoped sandbox provisioning from task-scoped prompt dispatch; `/send` remains strictly post-`DISPATCHED`. | Retain strict post-dispatch session spawn. | None directly. | None. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D3** | **Fail-Closed Spawn Disposition with Explicit Session Exclusivity Guard (`CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`) & Durable Provisioning Operations** | Pinned AO spawn exposes no caller correlation ID; pre-binding spawn crashes tracked durably in `pair_provisioning_operations`; spawn allowed ONLY if no current WorkerSession exists; unowned sessions cannot be proven owned and are not killed automatically. | Automatic orphan reaper; Blind spawn when WorkerSession exists; Ephemeral in-memory quarantine. | Creates table `pair_provisioning_operations` with partial unique index; locks Pair lane on unresolved crash. | None. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D4** | **Dedicated `dispatch_operations` Table with 1:1 Attempt Cardinality & Candidate TaskAttempt Extensions** | Tracks `DISPATCH_BOUND -> SEND_REQUESTED -> SEND_CONFIRMED`; enforces `UNIQUE(attempt_id)` (`# ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION`); correctly references `task_attempts(attempt_id)`; defines candidate snapshot and quarantine fields on `task_attempts`. | Referencing non-existent `task_attempts(id)`; Permitting multiple dispatch sagas per attempt. | Creates `dispatch_operations` table with `UNIQUE(attempt_id)`. | None (TaskState remains `DISPATCHED`). | `PENDING_EXTERNAL_REAUDIT_004` |
| **D5** | **Fail-Closed Terminal Failure (`DISPATCHED -> FAILED`) with Immediate Quarantine** | Pinned `/send` lacks delivery proof on crash; assuming failure and quarantining prevents split-brain execution; clearance requires Class A, Class B, or Class C resolution. | Blind resend; Silent continuation; Direct `DISPATCHED -> HUMAN_REQUIRED`. | Records `resolution_state` in dispatch operations and audit log; sets `quarantine_state = 'QUARANTINED'` on both session and attempt. | `DISPATCHED -> FAILED -> HUMAN_REQUIRED`. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D6** | **Option D: Double-Gated Quarantine with Dedicated Attempt Quarantine State** | Provides defense-in-depth across Pair lane and task attempt levels; separates independent safety gate (`task_attempts.quarantine_state`) from diagnostic outcome (`recovery_disposition`). | Single-level lane check; Conflating diagnostic disposition with quarantine safety truth. | `worker_sessions.quarantine_state` and dedicated `task_attempts.quarantine_state`. | Blocks `FAILED -> READY` while quarantined. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D7** | **Strict Whitelist Pre-Send Admissibility (`idle`, `waiting_input` only) via Approved Adapter Surface** | SessionGuard blocks writes on `blocked`/`exited`/`terminated`; `active` before send indicates foreign turn; `terminal_generation` is opaque string; inspects activity via approved `GetWorkerStatus`. | Permitting `active` before send; Raw transport calls. | Records rejection audit event. | Rejection halts dispatch or fails attempt. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D8** | **Option C: Persist Ambiguity Disposition (`MISSED_ACTIVE_WINDOW`) and Handoff Downstream with Verified Pre-Transition Check** | Respects task boundaries; verifies handoff capability prior to `DISPATCHED -> RUNNING`; fails `DISPATCHED -> FAILED` if unavailable at checkpoint; later failures use `RUNNING -> FAILED`. | Transition order contradiction; Immediate fail-closed; Predefining "no modified files = FAILED". | `task_attempts.recovery_disposition = 'MISSED_ACTIVE_WINDOW'`. | `DISPATCHED -> RUNNING` (or `DISPATCHED -> FAILED` if handoff unavailable). | `PENDING_EXTERNAL_REAUDIT_004` |
| **D9** | **Preserve `RUNNING`, Emit Diagnostic Telemetry, Continue Polling** | `waiting_input` indicates worker is at empty prompt; does not indicate failure or permission blockage. | Transition to `BLOCKED`; Transition to `FAILED`. | Diagnostic audit telemetry only. | None (remains `RUNNING`). | `PENDING_EXTERNAL_REAUDIT_004` |
| **D10** | **Graph-Compatible Blocked Handling: Preserve `RUNNING` with `AO_BLOCKED_DECISION` Disposition; One-Way Escalation to `BLOCKED -> HUMAN_REQUIRED`** | Canonical graph has NO `BLOCKED -> RUNNING` edge; AO `blocked` leaves TaskState in `RUNNING` while attempt remains open (`ended_at = NULL`); automated input suppressed; formal escalation follows canonical one-way path. | Claiming `BLOCKED -> RUNNING` resumption; Terminating attempt on blocked observation. | `task_attempts.recovery_disposition = 'AO_BLOCKED_DECISION'`. | TaskState remains `RUNNING` unless formally escalated. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D11** | **Purpose-Aware Stop Lifecycle with Generation Precheck, Restart-Stable Deadlines & Preserved Provenance** | Pinned `/kill` lacks atomic generation fence (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`); stop operations bind explicit `purpose` (`RUNNING_ATTEMPT_STOP`, `QUARANTINE_CLEANUP`, `PAIR_MAINTENANCE`); `stage` preserves wire-effect facts while `resolution_state` records outcome; persists `confirmation_deadline_at`. | Claiming atomic kill fence; Erasing wire-effect stage provenance; Unconditional `RUNNING -> FAILED` for cleanup stops. | Complete `stop_operations` table with `purpose`, `confirmation_deadline_at`, and `resolved_at`. | `PURPOSE_DEPENDENT` (`RUNNING -> FAILED` only for `RUNNING_ATTEMPT_STOP` on `RUNNING` tasks; ZERO transition for cleanup/maintenance). | `PENDING_EXTERNAL_REAUDIT_004` |
| **D12** | **Unified StateStore Atomic Terminal Transition (`AtomicTerminalTransition`) without Undeclared Columns** | Ensures crash consistency: TaskState CAS, attempt `ended_at`, and `recovery_disposition` committed in one SQLite transaction; full error details logged in `audit_events.details_json`; zero undeclared columns. | Referencing nonexistent `terminal_error` on `task_attempts`; Non-transactional terminal updates. | Atomically updates `tasks`, `task_attempts`, `audit_events`. | Executes terminal failure transitions. | `PENDING_EXTERNAL_REAUDIT_004` |
| **D13** | **Synchronous Startup Recovery Sweep Covering Provisioning, Dispatch, and Purpose-Aware Stops** | Enumerates `pair_provisioning_operations`, `dispatch_operations`, and in-flight `stop_operations`; prohibits automatic `/kill` reissue (`STOP_REISSUE_REQUIRES_HUMAN`); evaluates restart-stable `confirmation_deadline_at`; uses approved adapter methods. | Scanning `STOP_REQUESTED` only; Automatic reissue of kill without atomic fence; Resetting stop timeout on restart. | Recovery audit records. | Reconciles in-flight tasks and sagas. | `PENDING_EXTERNAL_REAUDIT_004` |

---

### Canonical ADR-016 V1 Token Registry

To guarantee strict cross-section consistency across documentation, candidate schemas, and future implementation contracts, all lifecycle, saga, and audit operations must strictly use the canonical tokens defined in this registry:

1. **Pair Provisioning Stages (`pair_provisioning_operations.stage`)**:
   - `PROVISION_REQUESTED`: Pre-effect intent committed prior to `POST /api/v1/sessions`.
   - `PROVISION_CONFIRMED`: WorkerSession created and bound upon HTTP 201 response.
   - `PROVISION_FAILED`: Spawn call rejected or crashed before local commit.
   - `PROVISION_RESOLVED`: Operator-audited resolution of failed provisioning.

2. **Task Dispatch Stages (`dispatch_operations.stage`)**:
   - `DISPATCH_BOUND`: Task committed `DISPATCHED`, attempt allocated, session bound.
   - `SEND_REQUESTED`: Pre-effect dispatch intent committed prior to `POST .../send`.
   - `SEND_CONFIRMED`: Upstream HTTP 200 write acceptance received.

3. **Stop Wire-Effect Stages (`stop_operations.stage`)**:
   - `STOP_REQUESTED`: Stop intent committed prior to `POST .../kill`.
   - `STOP_CALL_SUCCEEDED`: Kill HTTP call positively returned HTTP 200 OK.
   - `STOP_CALL_FAILED`: Kill HTTP call positively failed at wire/protocol level.
   - `STOP_TERMINATION_CONFIRMED`: Subsequent observation returned `isTerminated == true`.
   - `STOP_TARGET_ABSENT`: Upstream returned HTTP 404 Not Found.

4. **Stop Operation Purposes (`stop_operations.purpose`)**:
   - `RUNNING_ATTEMPT_STOP`: Live stop of an active attempt in `RUNNING` state.
   - `QUARANTINE_CLEANUP`: Cleanup of a terminated or crashed session for an attempt already in `FAILED` or `HUMAN_REQUIRED`.
   - `PAIR_MAINTENANCE`: Infrastructure cleanup without an associated task attempt.

5. **Stop Resolution States (`stop_operations.resolution_state`)**:
   - `IN_FLIGHT`: Awaiting observation, confirmation, or resolution.
   - `TERMINATION_CONFIRMED`: Target confirmed terminated matching expected execution generation.
   - `STOP_TARGET_ABSENT`: Target session absent (HTTP 404); physical termination unproven.
   - `STOP_CONFIRMATION_TIMEOUT`: `SUPERVISOR_KILL_STOP_TIMEOUT` deadline expired without confirmed exit.
   - `STOP_GENERATION_MISMATCH`: Target replaced by a newer execution generation.
   - `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED`: Target was already dead before kill was transmitted/accepted.
   - `STOP_REISSUE_REQUIRES_HUMAN`: Live target found on restart; automatic reissue prohibited due to lack of atomic generation fence.
   - `ADMINISTRATIVE_RISK_ACCEPTED`: Authorized human operator administrative risk acceptance.

6. **TaskAttempt Recovery Dispositions (`task_attempts.recovery_disposition`)**:
   - `UNCERTAIN_DELIVERY_CRASH`: Crash between `SEND_REQUESTED` and `SEND_CONFIRMED`.
   - `MISSED_ACTIVE_WINDOW`: Prompt write confirmed; observed `idle` without observing active window.
   - `AO_WAITING_INPUT_OBSERVED`: Worker paused at prompt; turn incomplete.
   - `AO_BLOCKED_DECISION`: Worker blocked on tool approval or permission dialog.
   - `WORKER_STOPPED`: Worker terminated via verified intentional live stop.
   - `SESSION_ABSENT`: Session absent upstream (HTTP 404).
   - `STOP_CONFIRMATION_TIMEOUT`: Stop confirmation deadline expired without confirmed exit.
   - `GENERATION_MISMATCH`: Upstream execution generation changed unexpectedly.
   - `WORKER_TERMINATION_UNKNOWN`: Termination observed without verified intentional stop provenance.
   - `RECOVERY_PENDING`: Upstream daemon unreachable; awaiting reconnection.
   - `STALE_EXECUTION_GENERATION`: Reconnected generation does not match attempt snapshot.

7. **Quarantine States (`worker_sessions.quarantine_state`, `task_attempts.quarantine_state`)**:
   - `CLEAN`: Safe for dispatch / allocation.
   - `QUARANTINED`: Locked against dispatch / retry pending governed resolution.

8. **Audit Event Types (`audit_events.event_type`)**:
   - `PAIR_SESSION_PROVISION_REQUESTED`
   - `PAIR_SESSION_PROVISIONED`
   - `TASK_DISPATCH_BOUND`
   - `DISPATCH_SEND_REQUESTED`
   - `DISPATCH_SEND_CONFIRMED`
   - `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED`
   - `QUARANTINE_RESOLVED_PHYSICAL`
   - `QUARANTINE_RESOLVED_ADMINISTRATIVE`
   - `WORKER_BLOCKED_ON_DECISION`
   - `STOP_OPERATION_REQUESTED`
   - `STOP_OPERATION_CALL_SUCCEEDED`
   - `STOP_OPERATION_CONFIRMED`
   - `STOP_OPERATION_TARGET_ABSENT`
   - `STOP_CONFIRMATION_TIMEOUT`
   - `MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM`
   - `STARTUP_RECOVERY_SWEEP_STARTED`
   - `STARTUP_RECOVERY_SWEEP_COMPLETED`

---

## 7. D1: Pair / WorkerSession Persistence Model

### Source / Canonical Facts
- Canonical `docs/05_DOMAIN_MODEL.md` defines `Pair "1" o-- "1" WorkerSession` with fields: `session_id`, `pair_id`, `runtime_type`, `worktree_path`, `worker_agent_id`, and `status`.
- Pinned AO `session_id` persists across terminal restore operations.
- `TaskAttempt` requires immutable record of the execution identity under which work was performed.

### Proposal Revision-6 Recommendation
- Model A (`GIẢ ĐỊNH`): Mutable Current Session in `worker_sessions` + Plain Historical Attempt Snapshot in `task_attempts`.

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Adopt Model A (Mutable Current WorkerSession with Nullable WorktreePath, Independent Quarantine State & Self-Contained TaskAttempt Execution Snapshot)**.
1. Authoritative Candidate Schema for `worker_sessions`:
   ```sql
   CREATE TABLE worker_sessions (
       pair_id TEXT PRIMARY KEY REFERENCES pairs(pair_id) ON DELETE RESTRICT,
       session_id TEXT NOT NULL UNIQUE,
       runtime_type TEXT NOT NULL,
       worktree_path TEXT,
       worker_agent_id TEXT NOT NULL,
       status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'IDLE', 'TERMINATED')),
       terminal_generation TEXT NOT NULL,
       quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED')),
       created_at TEXT NOT NULL,
       updated_at TEXT NOT NULL
   );
   ```
   - `pair_id`: Primary key binding exactly one current WorkerSession per Pair.
   - `session_id`: Upstream AO session UUID. Invariant: `AO_SESSION_IDENTITY MUST_BIND_TO_AT_MOST_ONE_PAIR` enforced via `session_id TEXT NOT NULL UNIQUE`.
   - `runtime_type`: Normalized value for V1 (`'agy_tui'` for Agy in ConPTY mode).
   - `worktree_path`: Absolute local worktree path. In pinned AO v0.13.0, `SessionMetadata.WorkspacePath` is `json:"-"` in `domain.Session` and is omitted from public `SessionView` and `SpawnSessionResponse`. The approved Supervisor adapter exposes no workspace path. Under pinned V1, this column is nullable (`TEXT NULL`, unknown/unavailable through public contract). TASK-P03-003 must not populate synthetic values or inspect private AO internals (`WORKTREE_PATH_PUBLIC_AVAILABILITY = UNAVAILABLE_IN_PINNED_V1`, Finding `ADR16R4-006`).
   - `worker_agent_id`: Canonical worker/harness identity bound to the Pair (`'agy'`).
   - `status`: Supervisor-owned normalized status (`'ACTIVE'`, `'IDLE'`, `'TERMINATED'`). Represents runtime process lifecycle only. `QUARANTINED` is removed from `status CHECK` to prevent dual competing safety truths. Safety gating is handled exclusively by `quarantine_state` (Finding `ADR16R4-002`, Section 12).
   - `terminal_generation`: Upstream opaque string launch fence (`SessionView.TerminalGeneration`).
   - `quarantine_state`: Independent lifecycle quarantine flag (`'CLEAN'` or `'QUARANTINED'`).
   - `created_at` / `updated_at`: ISO8601 text timestamps.
2. The `task_attempts` table stores self-contained execution snapshot columns: `session_id TEXT` and `terminal_generation TEXT` (opaque string), populated at `DISPATCH_BOUND` and permanently immutable, along with `recovery_disposition TEXT` and dedicated `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN'` (Finding `ADR16R4-002`).
3. No relational Foreign Key constraint exists between `task_attempts.session_id` and `worker_sessions.session_id`.
4. Session Replacement & Exclusivity (`ADR16R4-005`): Invariant `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF` requires that no `worker_sessions` row currently exists for the Pair. If a WorkerSession already exists (ACTIVE, IDLE, or QUARANTINED), new session creation is prohibited; the existing session must be reused, resumed, or administratively resolved.
5. Session Restore: Restoring a session within AO preserves `session_id` while updating `terminal_generation` to a new opaque string. The `worker_sessions` record updates its generation accordingly.
6. Stale Observation Protection: Polling and recovery workers match upstream `terminalGeneration` string against `task_attempts.terminal_generation`. If they differ (`MISMATCH`), the observation is rejected with `STALE_EXECUTION_GENERATION`.

### Rationale
Model A perfectly preserves the canonical 1:1 relationship between Pair and WorkerSession. It avoids the cyclic Foreign Key constraints that Model B introduces (where `pairs` points to `worker_sessions` and `worker_sessions` points to `pairs`), and avoids the join overhead and entity proliferation of Model C (`attempt_session_bindings`). Making `worktree_path` nullable truthfully respects the pinned AO public contract boundary. Separating `status` from `quarantine_state` prevents dual competing safety truths.

### Rejected Alternatives
- **Model B (Historical WorkerSession Table + Active Pointer)**: Rejected because it mutates the canonical domain model to `Pair 1 o-- * WorkerSession` and creates relational cycle issues in SQLite.
- **Model C (Pair Current Session + AttemptSessionBinding Join Table)**: Rejected because introducing a third entity adds schema complexity without providing any auditability benefit over plain snapshot columns.
- **Mandatory `worktree_path TEXT NOT NULL`**: Rejected under Finding `ADR16R4-006` because pinned AO public contract does not expose workspace path.

### Safety Invariant
Snapshot columns `(session_id, terminal_generation)` on `task_attempts` are permanently immutable once written.

### Persistence Impact
Adds columns `session_id TEXT`, `terminal_generation TEXT`, `recovery_disposition TEXT`, and `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN'` to `task_attempts`. Defines `worker_sessions` schema with nullable `worktree_path` and independent `quarantine_state`.

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

## 8. D2: ADR-012 External-Side-Effect Amendment

### Source / Canonical Facts
- Accepted ADR-012 Section 10 currently states: *"External AO API calls (`createWorkerSession`, `sendTask`) occur strictly after the durable `DISPATCHED` record exists."*

### Proposal Revision-6 Recommendation
- Formally amend ADR-012 Section 10.

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Formally Amend ADR-012 Section 10 to Decouple Pair Session Lifecycle Side Effects from Task Execution Side Effects**.
1. **Pair Session Lifecycle Side Effect (`createWorkerSession`)**: The creation, provisioning, or restoration of a worker session sandbox is classified as infrastructure management scoped to the `Pair`. It MAY occur prior to task dispatch, establishing a durable `WorkerSession` binding on the Pair.
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

## 9. D3: Spawn Uncertainty / Orphan Handling

### Source / Canonical Facts
- Pinned AO `POST /api/v1/sessions` has `PINNED_SPAWN_IDEMPOTENCY = ABSENT` and exposes NO caller correlation field in the spawn request.
- A crash between upstream session creation (T2) and local database commit (T3) results in an active AO session whose ID is uncommitted in the Supervisor.
- Fact: `P03_SPAWN_ORPHAN_IDENTITY = NOT_DETERMINISTICALLY_RECOVERABLE`.
- Safety Invariant: `UNOWNED_OR_UNCORRELATED_AO_SESSION MUST_NOT_BE_AUTOMATICALLY_KILLED`.

### Proposal Revision-6 Recommendation
- Fail-closed operational disposition + local audit correlation + operator-audited cleanup (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Adopt Fail-Closed Spawn Disposition with Durable Provisioning Operations and Operator-Audited Cleanup**.
1. **Durable Provisioning Operation Record (`pair_provisioning_operations`)**:
   Because a crash between upstream session creation (T2) and local database commit (T3) leaves no `worker_sessions` row, Pair-level provisioning uncertainty cannot be recorded on `worker_sessions.quarantine_state`. The Supervisor records provisioning operations in a dedicated durable table:
   ```sql
   CREATE TABLE pair_provisioning_operations (
       operation_id TEXT PRIMARY KEY,
       pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
       stage TEXT NOT NULL CHECK (stage IN ('PROVISION_REQUESTED', 'PROVISION_CONFIRMED', 'PROVISION_FAILED', 'PROVISION_RESOLVED')),
       client_token TEXT NOT NULL,
       session_id TEXT,
       requested_at TEXT NOT NULL,
       completed_at TEXT,
       resolved_at TEXT,
       resolved_by TEXT,
       resolution_notes TEXT
   );
   CREATE UNIQUE INDEX idx_pair_provisioning_unresolved ON pair_provisioning_operations(pair_id) WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED');
   ```

   - **Pre-Provisioning Exclusivity & Session Non-Replacement Invariant (Findings `ADR16R3-007`, `ADR16R4-005`)**:
     Invariant: `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`:
     1. No `worker_sessions` row currently exists for the Pair (`COUNT(*) == 0`).
     2. No unresolved `pair_provisioning_operations` exists for the Pair (`# ONE_PAIR = AT_MOST_ONE_UNRESOLVED_PROVISIONING_OPERATION`).
     3. No active Pair or task quarantine blocks provisioning (`quarantine_state == 'CLEAN'`).

     **`PROVISION_CONFIRMED` Does NOT Authorize a Second Spawn**: The partial unique index `idx_pair_provisioning_unresolved` locks unresolved stages. However, once an operation reaches `PROVISION_CONFIRMED` and `worker_sessions` exists, subsequent provisioning for that Pair is strictly prohibited by orchestration guard. If a WorkerSession already exists:
     - `ACTIVE` or `IDLE`: The existing session MUST be reused for subsequent tasks; new spawn is rejected.
     - Terminated but restorable: The existing session MUST be resumed via governed `ResumeWorker` path; allocating a new session ID is prohibited.
     - `QUARANTINED`: Provisioning is strictly locked.
     - Unresolved crash / 404: Fail closed; requires operator administrative resolution. Blind overwrite of Model-A current binding is strictly prohibited.
2. **Provisioning Transaction Protocol (Section 15)**:
   - **Pre-Call Transaction**: Atomically evaluate `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF` guard, insert `pair_provisioning_operations` row in `stage = 'PROVISION_REQUESTED'`, emit `PAIR_SESSION_PROVISION_REQUESTED` audit event, and commit SQLite transaction. Zero network calls occur inside this transaction.
   - **Upstream Wire Call**: Issue `POST /api/v1/sessions` outside transaction boundaries.
   - **Post-Call Transaction (Success)**: On HTTP 201 response, execute single SQLite transaction: insert `worker_sessions` row with upstream `session_id`, update `pair_provisioning_operations.stage = 'PROVISION_CONFIRMED'`, and emit `PAIR_SESSION_PROVISIONED` audit event.
   - **Post-Call Transaction (Definite Failure)**: On HTTP/transport error, execute single SQLite transaction updating `stage = 'PROVISION_FAILED'`.
   - **Post-Call Transaction (Ambiguous Failure / Crash)**: Startup recovery marks unresolved `PROVISION_REQUESTED` as `PROVISION_FAILED`.
   - **Pre-Dispatch Guard**: Prior to dispatching a task, the Supervisor verifies that no unresolved provisioning operations exist for the target Pair:
     `SELECT COUNT(*) FROM pair_provisioning_operations WHERE pair_id = ? AND stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')`.
     If count > 0, the Pair lane is QUARANTINED, and dispatch is refused fail-closed.
3. **Unowned Orphan Safety Invariant**:
   - `P03_SPAWN_ORPHAN_IDENTITY = NOT_DETERMINISTICALLY_RECOVERABLE`: Pinned AO exposes no client token in spawn requests; an uncommitted session cannot be deterministically proven to belong to the Supervisor.
   - `UNOWNED_OR_UNCORRELATED_AO_SESSION MUST_NOT_BE_AUTOMATICALLY_KILLED`: The Supervisor NEVER automatically terminates upstream AO sessions solely because they are unreferenced in its database. Doing so risks destroying sessions created by manual users, test suites, or other orchestrators sharing the AO daemon.
   - **Operator-Audited Cleanup**: Diagnostic tooling presents candidate unreferenced sessions to an authorized operator. Terminating any session requires positive human operator selection and authority.

### Rationale
Because pinned AO spawn provides no client correlation field, unreferenced sessions cannot be proven to belong to the Supervisor. The same principle that forbids heuristic adoption (path does not prove identity) equally forbids automatic killing (path does not prove ownership). Enforcing session exclusivity prevents orphan generation and preserves Model-A binding integrity.

### Rejected Alternatives
- **Automatic Background Orphan Reaper**: Rejected under Finding `ADR16R1-001` as dangerous; risks destroying sessions created by other tools or users sharing the AO daemon.
- **Heuristic Path Adoption**: Rejected as unsafe; path matching cannot prove that a session was created for the specific pending dispatch.
- **Second Spawn Authorization on `PROVISION_CONFIRMED`**: Rejected under Finding `ADR16R4-005` because overwriting existing sessions destroys quarantine truth and orphans running agents.

### Safety Invariant
No upstream AO session is ever automatically terminated without an authoritative local record of its ownership or explicit operator authorization.

### Persistence Impact
Records `PAIR_SESSION_PROVISION_REQUESTED` audit event with local correlation token.

### State-Machine Impact
None.

### Restart Impact
Crashed spawn operations do not block Supervisor startup; Pair marked `PROVISION_FAILED` awaiting operator attention.

### Auditability Impact
Audit log explicitly documents failed spawn attempts and operator-authorized cleanup actions.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 10. D4: Durable Dispatch Saga Representation

### Source / Canonical Facts
- Pinned AO `POST /api/v1/sessions/{id}/send` has `PINNED_SEND_IDEMPOTENCY = ABSENT`.
- Current P02 database schema defines `task_attempts` primary key as `attempt_id` (there is NO `task_attempts.id` column).
- Client `AOAdapter` must remain strictly stateless.

### Proposal Revision-6 Recommendation
- Dedicated table (`dispatch_operations`) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Dedicated `dispatch_operations` Table with Correct Foreign Key Reference to `task_attempts(attempt_id)`**.
1. Dispatch execution is tracked via a dedicated table:
   ```sql
   CREATE TABLE dispatch_operations (
       operation_id TEXT PRIMARY KEY,
       attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
       pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
       task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
       session_id TEXT NOT NULL,
       terminal_generation TEXT NOT NULL,
       stage TEXT NOT NULL CHECK (stage IN ('DISPATCH_BOUND', 'SEND_REQUESTED', 'SEND_CONFIRMED')),
       requested_at TEXT NOT NULL,
       confirmed_at TEXT,
       resolution_state TEXT
   );
   ```

   - **Invariant: One Dispatch Saga per TaskAttempt (`# ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION`, Finding `ADR16R3-003`)**:
     A `TaskAttempt` represents exactly one execution/retry iteration. Retrying requires allocating a NEW `TaskAttempt` entity and recording a separate dispatch operation. Enforced via `attempt_id TEXT NOT NULL UNIQUE`.
   - **Schema Reconciliation**: Table strictly includes `resolution_state TEXT` (relied upon by D5) and `confirmed_at TEXT`. Extraneous unapproved fields (such as `prompt_hash`) are omitted.
2. The 3 semantic stages:
   - `DISPATCH_BOUND`: TaskState committed `DISPATCHED`, `TaskAttempt` allocated, session and generation bound.
   - `SEND_REQUESTED`: Pre-effect intent durably committed immediately before calling `POST /api/v1/sessions/{id}/send`.
   - `SEND_CONFIRMED`: Committed immediately upon HTTP 200 response from AO `/send`.
3. **TaskAttempt Candidate Extensions (Finding `ADR16R4-002`, Section 14)**:
   ADR-016 defines the following candidate columns on `task_attempts`:
   - `session_id TEXT`: Bound WorkerSession ID at `DISPATCH_BOUND`; permanently immutable snapshot.
   - `terminal_generation TEXT`: Bound execution generation string at `DISPATCH_BOUND`; permanently immutable snapshot.
   - `recovery_disposition TEXT`: Mutable diagnostic disposition recording observed lifecycle/recovery facts.
   - `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))`: Independent attempt safety gate; mutable only through explicit quarantine impose/resolve transactions.
4. **Evidence Boundary**: `SEND_CONFIRMED` denotes that the upstream HTTP `/send` write was accepted. It does NOT prove the agent entered `ActivityActive`, does NOT prove prompt semantic consumption, and does NOT prove task completion.
5. `AOAdapter` remains completely stateless.
6. Attribution Rule: Upstream execution activity is attributed to a task attempt ONLY if `stage == SEND_CONFIRMED`, `session_id` matches, and `terminal_generation` matches.

### Rationale
Fixes Finding `ADR16R1-002` by referencing the canonical primary key `task_attempts(attempt_id)`. A dedicated table keeps execution saga state distinct from domain aggregate entities, simplifies crash recovery queries, and guarantees crash-consistent state reconstruction.

### Rejected Alternatives
- **Foreign Key to `task_attempts(id)`**: Rejected as invalid; column `id` does not exist in `task_attempts`.
- **Columns on `task_attempts`**: Rejected because multiple dispatch sagas may occur across retries, cluttering attempt records.

### Safety Invariant
`POST .../send` is never invoked unless `stage = SEND_REQUESTED` is durably committed in SQLite.

### Persistence Impact
Creates table `dispatch_operations` with valid foreign keys.

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

## 11. D5: SEND_REQUESTED Unknown-Delivery Disposition

### Source / Canonical Facts
- Crash between `SEND_REQUESTED` and `SEND_CONFIRMED` leaves `DELIVERY_OUTCOME = UNKNOWN`.
- Blind resend is strictly prohibited.
- Canonical state machine allows `DISPATCHED -> FAILED`, followed by `FAILED -> HUMAN_REQUIRED`. Direct `DISPATCHED -> HUMAN_REQUIRED` does not exist.

### Proposal Revision-6 Recommendation
- Class A physical resolution primary with Class C administrative risk acceptance fallback (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Fail Closed via `DISPATCHED -> FAILED`, Impose Immediate Quarantine, and Require Governed Resolution (Class A, Class B, or Class C)**.
1. When restart or error handling detects an unconfirmed `SEND_REQUESTED`:
   - Transition `DISPATCHED -> FAILED` (failure reason: `UNCERTAIN_DELIVERY_CRASH`).
   - Atomically set `TaskAttempt.ended_at = now`.
   - Set `task_attempts.quarantine_state = 'QUARANTINED'` (Finding `ADR16R4-002`).
   - Set `worker_sessions.quarantine_state = 'QUARANTINED'`.
   - Set `task_attempts.recovery_disposition = 'UNCERTAIN_DELIVERY_CRASH'`.
   - Transition `FAILED -> HUMAN_REQUIRED` for operator escalation.
   - Any automated retry (`FAILED -> READY`) is strictly rejected by the quarantine guard.
2. Governed Clearance Classes:
   - **Class A (`PHYSICAL_EXECUTION_RESOLUTION`)**: The Supervisor issues `POST /api/v1/sessions/{sessionId}/kill` to the old session and authoritatively observes `isTerminated == true` matching the execution generation string. Because pinned AO lacks an atomic kill generation fence (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`, Finding `ADR16R4-001`), automatic Class A kill is session-scoped; if the session cannot be proven safe to kill, Class B or Class C is required. Once physical termination is confirmed, quarantine clears with disposition `TERMINATION_CONFIRMED`.
   - **Class B (`ADMINISTRATIVE_RISK_RESOLUTION` / Absence Disposition)**: Upstream returns HTTP 404 (`STOP_TARGET_ABSENT`). Under the fail-closed Class B policy established in D11 and Finding `ADR16R3-006`, HTTP 404 proves public session absence but does NOT prove physical process termination. Quarantine remains ACTIVE. Quarantine clearance and future replacement dispatch strictly require an authorized operator executing `ADMINISTRATIVE_RISK_RESOLUTION` with recorded absence evidence and risk acknowledgement.
   - **Class C (`ADMINISTRATIVE_RISK_RESOLUTION` / Human Risk Acceptance)**: When target status is indeterminate, uncontactable, or unproven, an authorized operator explicitly acknowledges that delivery may have occurred and old execution cannot be positively proven stopped, and authorizes a replacement recovery path. This is durably recorded in the audit log as `ADMINISTRATIVE_RISK_ACCEPTED`. It MUST NOT be logged as `TERMINATION_CONFIRMED` or `WORKER_STOPPED`.
3. Invariant: `NEW_SESSION_OR_GENERATION != RESOLUTION_OF_PRIOR_UNCERTAIN_EXECUTION`. Creating a new session or advancing generation does not clear quarantine.

### Rationale
Because pinned AO `/send` has no deduplication key, assuming the message was lost risks dual-worker collision if the prompt actually arrived. Failing closed and quarantining guarantees safety.

### Rejected Alternatives
- **Blind Resend**: Catastrophic risk of duplicate execution; strictly prohibited.
- **Direct `DISPATCHED -> HUMAN_REQUIRED`**: Prohibited as an illegal transition edge in the canonical workflow state machine.

### Safety Invariant
An uncertain execution is never cleared without verifiable physical termination (Class A) or explicit, audited administrative risk resolution (Class B or Class C).

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

## 12. D6: Uncertain-Delivery Quarantine Guard

### Source / Canonical Facts
- Canonical graph allows `FAILED -> READY`. An unquarantined retry risks duplicate execution while uncertain execution runs.

### Proposal Revision-6 Recommendation
- Option D (Combination: Attempt disposition + Pair active lane lockout) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Double-Gated Quarantine with Dedicated Attempt Quarantine State (Findings `ADR16R4-002`, `ADR16R4-005`)**.
1. **Separation of Safety State from Diagnostic Disposition**:
   - `task_attempts.recovery_disposition` records machine-readable diagnostic outcome (`UNCERTAIN_DELIVERY_CRASH`, `STOP_TARGET_ABSENT`, `STOP_CONFIRMATION_TIMEOUT`).
   - `task_attempts.quarantine_state` is the independent safety gate (`CLEAN` vs `QUARANTINED`). Subsequent diagnostic events update `recovery_disposition` but MUST NOT clear `quarantine_state`.
2. **Double-Gated Enforcement**:
   - **Pair Lane Gate**: `worker_sessions.quarantine_state = 'QUARANTINED'` locks the Pair lane against new task or attempt allocation.
   - **TaskAttempt Gate**: `task_attempts.quarantine_state = 'QUARANTINED'` prevents retry of that specific task.
3. **Dispatch Guard (`PrepareDispatch`)**:
   Prior to dispatching or retrying a task, the Supervisor verifies:
   - Target `worker_sessions.quarantine_state == 'CLEAN'`, AND
   - All prior attempts for the target task have `task_attempts.quarantine_state == 'CLEAN'`, AND
   - Target Pair has no unresolved provisioning operations (`stage NOT IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')`).
   If any check fails, dispatch is rejected with `ErrQuarantinedExecution`.
4. **Clearance Protocol**:
   Quarantine is cleared in a single atomic SQLite transaction ONLY when governed Class A (physical kill confirmed) or Class B / Class C (operator administrative risk resolution) is formally executed:
   - Verifies exact lineage tuple `(attempt_id, session_id, terminal_generation)`.
   - Sets `task_attempts.quarantine_state = 'CLEAN'`.
   - Sets `worker_sessions.quarantine_state = 'CLEAN'` (only if authorized).
   - Updates `task_attempts.recovery_disposition` to final resolution disposition.
   - Emits `QUARANTINE_RESOLVED_PHYSICAL` or `QUARANTINE_RESOLVED_ADMINISTRATIVE` audit event.

### Rationale
Single-level gating leaves either the Pair lane exposed to reuse or the TaskAttempt exposed to re-dispatch. Separating `quarantine_state` from `recovery_disposition` ensures that diagnostic updates (e.g. recording 404 absence) never accidentally clear active safety locks.

### Rejected Alternatives
- **Option A (Pair metadata only)**: Single point of failure; risks retrying the task on another lane while old execution continues.
- **Option C (Attempt disposition only)**: Risks dispatching a different task onto the corrupted session.
- **Conflating Diagnostic Disposition with Safety Quarantine**: Rejected under Finding `ADR16R4-002`; overwriting `recovery_disposition` erased quarantine truth.

### Safety Invariant
No dispatch can proceed on a Pair or TaskAttempt while an active quarantine flag exists.

### Persistence Impact
Adds `quarantine_state TEXT` to `worker_sessions` and `quarantine_state TEXT` to `task_attempts`.

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

## 13. D7: Pre-Send Admissibility

### Source / Canonical Facts
- Pinned AO SessionGuard prohibits writes on `blocked`, `exited`, `terminated`. `waiting_input` accepts input. `idle` is clean prompt. Generation mismatch breaks fence.
- `terminalGeneration` is an opaque `string` launch fence (`Metadata.RuntimeLaunchID`).

### Proposal Revision-6 Recommendation
- Strict whitelist (`idle`, `waiting_input` permitted; all others prohibited) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Enforce Strict Whitelist Admissibility for Pre-Send Verification with Opaque String Generation Matching**.
1. Immediately before transitioning from `DISPATCH_BOUND` to `SEND_REQUESTED`, the Supervisor inspects session activity via approved adapter method `GetWorkerStatus(ctx, sessionID)` (calling upstream wire route `GET /api/v1/sessions/{id}`).
2. Admissibility Rules:
   - `idle`: **SAFE_TO_SEND**. SessionGuard is authoritative startup safety gate; proceed to send.
   - `waiting_input`: **SAFE_TO_SEND**. Upstream allows instruction delivery; proceed to send.
   - `blocked`: **SEND_PROHIBITED**. Agent is stopped on tool permission or approval dialog; halt dispatch, await operator unblock or fail attempt.
   - `exited`: **SEND_PROHIBITED**. Agent process has exited; halt dispatch, require session restore.
   - `isTerminated: true`: **SEND_PROHIBITED**. Session dead; fail attempt (`DISPATCHED -> FAILED`).
   - `active`: **SEND_PROHIBITED**. Supervisor has not yet sent current task; session is executing foreign or lingering work. Halt dispatch, do not interleave prompts.
   - Generation Mismatch: **SEND_PROHIBITED**. Upstream `terminalGeneration` string does not equal `task_attempts.terminal_generation` string (`MISMATCH`); fence broken; fail attempt (`STALE_EXECUTION_GENERATION`).
3. Automated Nudging: Absolutely NO automated keystrokes, simulated Enter keys, or blind prompts may be sent to clear blocked dialogs.

### Rationale
A strict whitelist guarantees that prompts are delivered only when the upstream terminal is in a receptive state, completely preventing input corruption or shell command execution in exited panes. Treating terminal generation as an opaque string adheres strictly to the pinned AO DTO contract.

### Rejected Alternatives
- **Permitting `active` before send**: Catastrophic risk of interleaving prompts mid-turn; rejected.
- **Automated unblock nudges**: Violates human supervision boundaries and risks unintended tool approvals; rejected.

### Safety Invariant
Task prompt `/send` is never called unless pre-send activity check confirms `idle` or `waiting_input` with matching generation string.

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

## 14. D8: Missed Active Window

### Source / Canonical Facts
- `SEND_CONFIRMED` committed, but subsequent observation snapshot is `idle`. Does not prove whether prompt ran or was missed. Cannot auto-transition `DISPATCHED -> RUNNING -> REPORT_READY`.
- Strict Boundary: TASK-P03-003 does NOT perform report artifact reads, workspace file probing, or report validation (owned by TASK-P03-004 and P04).
- Finding `ADR16R1-008`: ADR-016 must NOT predefine downstream failure policy (such as "no modified files = FAILED"). Analysis or verification tasks may legitimately produce no modified files.

### Proposal Revision-6 Recommendation
- Option C (Persist ambiguity disposition and hand off downstream) (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Adopt Option C (Persist Ambiguity Disposition and Hand Off Downstream with Strict Pre-Transition Readiness Check)**.
1. State Transition Ordering:
   When the Supervisor observes `idle` following `SEND_CONFIRMED` without having observed an intervening `active` state:
   - **Pre-Transition Check**: Prior to committing `DISPATCHED -> RUNNING`, the Supervisor verifies that downstream handoff capability is available (StateStore writable, handoff queue accessible).
   - **If Handoff Capability is Unavailable at Pre-Transition**: The Supervisor fails closed using the legal canonical edge `DISPATCHED -> FAILED` (reason: `HANDOFF_UNAVAILABLE`).
   - **If Handoff Capability is Available**: The Supervisor commits canonical transition `DISPATCHED -> RUNNING` and marks `task_attempts.recovery_disposition = 'MISSED_ACTIVE_WINDOW'`.
   - **Post-Commit Failure Invariant**: Once `DISPATCHED -> RUNNING` has committed, any subsequent failure must strictly follow the canonical edge `RUNNING -> FAILED`. Under no circumstances is `DISPATCHED -> FAILED` ever attempted after `RUNNING` has been reached.
2. Downstream Handoff Boundary:
   - The task is passed to downstream authorized phases: TASK-P03-004 (raw workspace file transport) and Phase P04 (Evidence Collector / report semantic validation).
   - TASK-P03-003 supplies the verified execution identity `(session_id, terminal_generation)` and ambiguity disposition to the handoff contract.
   - Downstream phases independently evaluate their own authorized evidence contracts and report schemas.
   - ADR-016 explicitly refrains from dictating downstream failure rules (e.g. "no modified files = FAILED" is rejected because read-only or analysis contracts legitimately modify no files).

### Rationale
Resolves Finding `ADR16R1-008`. Short CLI commands or cached tool turns can execute and return to `idle` in sub-second intervals between polling ticks. Handing off with an explicit ambiguity flag preserves task boundaries and allows downstream components to evaluate reports against specific contract requirements.

### Rejected Alternatives
- **Predefining "no modified files = FAILED"**: Rejected under Finding `ADR16R1-008` as factually incorrect for read-only or analysis contracts.
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

## 15. D9: waiting_input Mapping

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

## 16. D10: blocked Mapping

### Source / Canonical Facts
- Pinned AO `blocked` means agent stopped on tool permission or human approval dialog. Automated input is suppressed by SessionGuard (`SuppressedAwaitingUser`).
- Canonical State Machine (`internal/workflow/state_machine.go`): The ONLY transition out of `BLOCKED` is `{From: "BLOCKED", To: "HUMAN_REQUIRED"}`. There is NO `BLOCKED -> RUNNING` edge.
- Finding `ADR16R1-003`: Claiming that operator unblocking resumes execution from `BLOCKED` to `RUNNING` under the same attempt violates the canonical state graph.

### Proposal Revision-6 Recommendation
- Preserve `RUNNING` with `AO_BLOCKED_DECISION` disposition; one-way formal escalation to `BLOCKED -> HUMAN_REQUIRED` (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Preserve `TaskState = RUNNING` with `AO_BLOCKED_DECISION` Lifecycle Disposition; Execute One-Way Workflow Escalation (`RUNNING -> BLOCKED -> HUMAN_REQUIRED`) Only on Formal Escalation**.
1. **Observation-Level Handling**:
   - When AO activity `blocked` is observed during task execution, the TaskState remains `RUNNING`.
   - The Supervisor records `task_attempts.recovery_disposition = 'AO_BLOCKED_DECISION'`.
   - `TaskAttempt.ended_at` remains `NULL` (the attempt remains open).
   - Automated task input remains strictly prohibited (SessionGuard suppression respected).
   - The poller continues authoritative read-only observation.
   - If the operator resolves the tool permission dialog upstream in AO while no workflow escalation has occurred, the session returns to `active` or `idle`. The task remains in `RUNNING`, and normal lifecycle observation continues without any invalid state machine transition.
2. **Formal Workflow Escalation**:
   - If the operator or policy dictates formal workflow escalation, the workflow state machine executes the canonical one-way path:
     `RUNNING -> BLOCKED` followed by `BLOCKED -> HUMAN_REQUIRED`.
   - `TaskAttempt.ended_at` remains `NULL` upon entering `BLOCKED`, and is closed only if terminal failure or replacement occurs.
   - Under no circumstances does the Supervisor attempt a non-existent `BLOCKED -> RUNNING` transition.

### Rationale
Resolves Finding `ADR16R1-003`. Strictly conforms to `state_machine.go` where `BLOCKED` has no return edge to `RUNNING`. Separating observation-level tool approval pauses from formal workflow escalation allows interactive dialogs to be cleared upstream without corrupting the state graph.

### Rejected Alternatives
- **Same-Attempt `BLOCKED -> RUNNING` Resumption**: Rejected under Finding `ADR16R1-003` as an illegal transition edge absent from the canonical state machine.
- **Immediate Termination on Blocked**: Aborts executions that merely require brief operator confirmation; rejected.

### Safety Invariant
The Supervisor never attempts a state transition that violates the canonical workflow state graph defined in `internal/workflow/state_machine.go`.

### Persistence Impact
`task_attempts.recovery_disposition = 'AO_BLOCKED_DECISION'`.

### State-Machine Impact
TaskState remains `RUNNING` during observation; one-way canonical path `RUNNING -> BLOCKED -> HUMAN_REQUIRED` used for formal escalation.

### Restart Impact
Restart scanner preserves `AO_BLOCKED_DECISION` disposition and open attempt status.

### Auditability Impact
Audit event `WORKER_BLOCKED_ON_DECISION` recorded.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 17. D11: Purpose-Aware Stop Lifecycle & Honest Provenance

### Source / Canonical Facts
- Pinned AO wire route is `POST /api/v1/sessions/{sessionId}/kill` (`sessions.go:206`). HTTP `DELETE` does not exist.
- HTTP `200 OK` indicates only that AO accepted the kill signal; it does NOT prove process termination.
- **Absence of Atomic Generation Fence (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`, Finding `ADR16R4-001`)**: Pinned `/kill` takes session identity only. It accepts NO `expectedGeneration`, NO `terminalGeneration` request body field, NO `If-Match` header, and NO CAS token. Calling `GET` before `POST /kill` provides a client-side `GENERATION_PRECHECK` only, NOT an atomic generation fence (`GENERATION_PRECHECK != GENERATION_FENCE`). A TOCTOU window exists.

### Proposal Revision-6 Recommendation
- 3-stage durable stop operation lifecycle with complete relational provenance (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Purpose-Aware Stop Operation Lifecycle with Generation Precheck, Restart-Stable Deadlines, and Provenance Preservation (Findings `ADR16R4-001`, `ADR16R4-003`, `ADR16R4-004`, `ADR16R4-007`)**.
1. **Defined Wire-Effect Stages (`stop_operations.stage`)**:
   `stage` reflects the highest positively proven wire-effect fact and is NEVER rewritten to represent subsequent diagnostic outcomes:
   - `STOP_REQUESTED`: Durably committed to SQLite immediately *before* calling `POST /api/v1/sessions/{sessionId}/kill`.
   - `STOP_CALL_SUCCEEDED`: Committed immediately upon receiving HTTP 200 (`KillSessionResponse`).
   - `STOP_CALL_FAILED`: Committed ONLY when an actual HTTP call positively returns/decodes as failure at wire/protocol level.
   - `STOP_TERMINATION_CONFIRMED`: Committed when a subsequent observation snapshot confirms `isTerminated == true` matching target generation following `STOP_CALL_SUCCEEDED`.
   - `STOP_TARGET_ABSENT`: Committed when upstream returns HTTP 404 Not Found. Proves session absence, but does NOT prove physical process termination.
2. **Complete Authoritative Schema**:
   ```sql
   CREATE TABLE stop_operations (
       operation_id TEXT PRIMARY KEY,
       purpose TEXT NOT NULL CHECK (purpose IN ('RUNNING_ATTEMPT_STOP', 'QUARANTINE_CLEANUP', 'PAIR_MAINTENANCE')),
       pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
       task_id TEXT REFERENCES tasks(task_id) ON DELETE RESTRICT,
       contract_id TEXT REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
       attempt_id TEXT REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
       session_id TEXT NOT NULL,
       terminal_generation TEXT NOT NULL,
       stage TEXT NOT NULL CHECK (stage IN ('STOP_REQUESTED', 'STOP_CALL_SUCCEEDED', 'STOP_CALL_FAILED', 'STOP_TERMINATION_CONFIRMED', 'STOP_TARGET_ABSENT')),
       actor TEXT NOT NULL,
       requested_at TEXT NOT NULL,
       call_completed_at TEXT,
       confirmation_deadline_at TEXT,
       termination_confirmed_at TEXT,
       resolved_at TEXT,
       resolution_state TEXT NOT NULL DEFAULT 'IN_FLIGHT'
   );
   ```
3. **Stop Operation Purpose & TaskState Semantics (Finding `ADR16R4-003`)**:
   - `RUNNING_ATTEMPT_STOP`: Active live stop during a `RUNNING` attempt. When stop completes, drives valid `RUNNING -> FAILED` via `AtomicTerminalTransition` (`ended_at = now`).
   - `QUARANTINE_CLEANUP`: Cleanup of a terminated or crashed session for an attempt already in `FAILED` or `HUMAN_REQUIRED`. Drives **ZERO TaskState transitions** and **ZERO `ended_at` mutations**; updates quarantine and reconciliation truth only.
   - `PAIR_MAINTENANCE`: Infrastructure cleanup without an active task attempt (`task_id`, `contract_id`, and `attempt_id` may be NULL). Drives **ZERO TaskState transitions**.
4. **Generation Precheck & Automatic Reissue Prohibition (Finding `ADR16R4-001`)**:
   - Because pinned AO `/kill` lacks an atomic generation fence, client-side generation checks are prechecks only.
   - **Automatic `/kill` Reissue on Restart is PROHIBITED**: If daemon restart encounters a persisted `STOP_REQUESTED` where upstream target is alive, the Supervisor does NOT automatically reissue `/kill`. Instead, it transitions `resolution_state = 'STOP_REISSUE_REQUIRES_HUMAN'`, retains quarantine, and awaits operator reconciliation.
   - For live worker stop: Calling `StopWorker` is session-scoped. Precheck/post-check observations are attribution evidence only; residual TOCTOU risk is acknowledged and cannot be eliminated under pinned V1.
   - For quarantine cleanup: Because kill is session-scoped, automatic Class A kill risks killing a newer generation. Prefer Class B or Class C administrative reconciliation unless a human explicitly authorizes session-scoped kill.
5. **Wire-Effect Stage Provenance Preservation (Finding `ADR16R4-004`)**:
   - `stage` preserves the physical wire-effect fact. Subsequent timeouts or mismatches MUST NOT rewrite `stage`.
   - `STOP_CALL_SUCCEEDED` + timeout: `stage` remains `STOP_CALL_SUCCEEDED`, `resolution_state = 'STOP_CONFIRMATION_TIMEOUT'`.
   - `STOP_CALL_SUCCEEDED` + generation mismatch: `stage` remains `STOP_CALL_SUCCEEDED`, `resolution_state = 'STOP_GENERATION_MISMATCH'`.
   - `STOP_REQUESTED` + target already dead: `stage` remains `STOP_REQUESTED`, `resolution_state = 'STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED'`.
   - `STOP_REQUESTED` + restart alive: `stage` remains `STOP_REQUESTED`, `resolution_state = 'STOP_REISSUE_REQUIRES_HUMAN'`.
   - Terminal resolutions record timestamp in `resolved_at`. Startup recovery scans `WHERE resolution_state = 'IN_FLIGHT'`.
6. **Restart-Stable Stop Confirmation Deadline (Finding `ADR16R4-007`)**:
   - When `STOP_CALL_SUCCEEDED` is committed, the Supervisor computes once:
     `confirmation_deadline_at = call_completed_at + injected SUPERVISOR_KILL_STOP_TIMEOUT`.
   - On daemon restart, if `resolution_state == 'IN_FLIGHT'`:
     - If `now < confirmation_deadline_at`: continue bounded observation until deadline.
     - If `now >= confirmation_deadline_at`: advance to `resolution_state = 'STOP_CONFIRMATION_TIMEOUT'`, `resolved_at = now`.
   - The timeout is NOT reset on restart; the persisted deadline is absolute.
7. **Formal Class-B Administrative Absence Policy (Finding `ADR16R3-006`)**:
   - HTTP 404 sets `stage = 'STOP_TARGET_ABSENT'`, `resolution_state = 'STOP_TARGET_ABSENT'`, `resolved_at = now`.
   - If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: drives `RUNNING -> FAILED` (reason: `SESSION_ABSENT`).
   - **404 alone NEVER clears quarantine**: Physical process termination remains unproven.
   - Quarantine clearance requires an explicit human operator executing `ADMINISTRATIVE_RISK_RESOLUTION` with durable audit record `(actor, timestamp, target_session_id, terminal_generation, absence_evidence, risk_acknowledgement)`.
   - Old execution remains `PRIOR_EXECUTION_PHYSICALLY_UNRESOLVED`. System never logs `TERMINATION_CONFIRMED` or `WORKER_STOPPED`.
8. **Stop / Quarantine TaskState Matrix (Finding `ADR16R4-003`, Section 16)**:

| Stop Purpose | Current TaskState | Observed Result | Allowed TaskState Transition | Attempt `ended_at` Action | Quarantine Action |
|---|---|---|---|---|---|
| `RUNNING_ATTEMPT_STOP` | `RUNNING` | `STOP_TERMINATION_CONFIRMED` | `RUNNING -> FAILED` (`WORKER_STOPPED`) | Set `now` via `AtomicTerminalTransition` | Clears attempt quarantine; Pair lane clean |
| `RUNNING_ATTEMPT_STOP` | `RUNNING` | `STOP_TARGET_ABSENT` (404) | `RUNNING -> FAILED` (`SESSION_ABSENT`) | Set `now` via `AtomicTerminalTransition` | Quarantine remains ACTIVE on attempt & Pair |
| `RUNNING_ATTEMPT_STOP` | `RUNNING` | `STOP_CONFIRMATION_TIMEOUT` | `RUNNING -> FAILED` (`STOP_CONFIRMATION_TIMEOUT`) | Set `now` via `AtomicTerminalTransition` | Quarantine remains ACTIVE on attempt & Pair |
| `RUNNING_ATTEMPT_STOP` | `RUNNING` | `STOP_GENERATION_MISMATCH` | `RUNNING -> FAILED` (`GENERATION_MISMATCH`) | Set `now` via `AtomicTerminalTransition` | Quarantine remains ACTIVE on attempt & Pair |
| `RUNNING_ATTEMPT_STOP` | `RUNNING` | `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED` | `RUNNING -> FAILED` (`WORKER_TERMINATION_UNKNOWN`) | Set `now` via `AtomicTerminalTransition` | Quarantine remains ACTIVE on attempt & Pair |
| `QUARANTINE_CLEANUP` | `FAILED` / `HUMAN_REQUIRED` | `STOP_TERMINATION_CONFIRMED` | **NONE** (Zero transition) | **NO-OP** (already terminal) | Clears quarantine on verified physical kill |
| `QUARANTINE_CLEANUP` | `FAILED` / `HUMAN_REQUIRED` | `STOP_TARGET_ABSENT` (404) | **NONE** (Zero transition) | **NO-OP** (already terminal) | Quarantine remains ACTIVE until operator Class B |
| `QUARANTINE_CLEANUP` | `FAILED` / `HUMAN_REQUIRED` | `STOP_CONFIRMATION_TIMEOUT` | **NONE** (Zero transition) | **NO-OP** (already terminal) | Quarantine remains ACTIVE |
| `PAIR_MAINTENANCE` | Any / No Task | Any Result | **NONE** (Zero transition) | **NO-OP** (No attempt) | Updates Pair lane maintenance state |

9. **Stop Recovery Truth Table (Reconciled)**:

| Stage | Upstream State Observed | Proven | Not Proven | Call `/kill`? | Next Stage | Resolution State | TaskState Effect | `WORKER_STOPPED` Allowed? |
|---|---|---|---|---|---|---|---|---|
| `STOP_REQUESTED` | Same Generation, Alive | Stop requested; target generation alive | Termination | **NO** (auto-reissue prohibited on restart) | `STOP_REQUESTED` | `STOP_REISSUE_REQUIRES_HUMAN` | Purpose-dependent (remains `RUNNING` or `HUMAN_REQUIRED`) | **NO** (Finding `ADR16R4-001`) |
| `STOP_REQUESTED` | Generation Mismatch | Upstream runtime replaced; old generation gone | Exit reason of old process | **NO** (prohibited) | `STOP_REQUESTED` | `STOP_GENERATION_MISMATCH` | Purpose-dependent (`RUNNING -> FAILED` for live stop) | **NO** |
| `STOP_REQUESTED` | Same Generation, `isTerminated == true` | Stop intent logged; target currently dead | Whether `/kill` was called/accepted or caused exit | **NO** (already dead) | `STOP_REQUESTED` | `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED` | Purpose-dependent (`RUNNING -> FAILED` for live stop) | **NO** (Finding `ADR16R3-004`) |
| `STOP_REQUESTED` | HTTP 404 (Not Found) | Public session absent | Affirmative termination; intentional stop effect | **NO** (absent) | `STOP_TARGET_ABSENT` | `STOP_TARGET_ABSENT` | Purpose-dependent (`RUNNING -> FAILED` for live stop) | **NO** (Finding `ADR16R3-006`) |
| `STOP_CALL_SUCCEEDED` | Same Generation, Alive | Kill accepted upstream; process not yet exited | Termination | **NO** (kill accepted; observe bounded by deadline) | `STOP_CALL_SUCCEEDED` | `IN_FLIGHT` (or `STOP_CONFIRMATION_TIMEOUT` on expiry) | Purpose-dependent (remains `RUNNING` until deadline) | **NO** (Finding `ADR16R4-007`) |
| `STOP_CALL_SUCCEEDED` | Generation Mismatch | Kill accepted, but newer generation launched | Old termination timing | **NO** (prohibited) | `STOP_CALL_SUCCEEDED` | `STOP_GENERATION_MISMATCH` | Purpose-dependent (`RUNNING -> FAILED` for live stop) | **NO** |
| `STOP_CALL_SUCCEEDED` | Same Generation, `isTerminated == true` | Kill accepted AND process termination confirmed | None | **NO** (already dead) | `STOP_TERMINATION_CONFIRMED` | `TERMINATION_CONFIRMED` | Purpose-dependent (`RUNNING -> FAILED` for live stop) | **YES** (Live stop only) |
| `STOP_CALL_SUCCEEDED` | HTTP 404 (Not Found) | Kill accepted, session now absent | Exit code / graceful vs crash | **NO** (absent) | `STOP_TARGET_ABSENT` | `STOP_TARGET_ABSENT` | Purpose-dependent (`RUNNING -> FAILED` for live stop) | **NO** (Finding `ADR16R3-006`) |

### Rationale
Resolves Finding `ADR16R1-006`. Binds every stop operation to its full task attempt, contract revision, and execution generation lineage, preventing unrecorded or dangling terminations.

### Rejected Alternatives
- **Reduced Schema Omitting Task/Attempt Keys**: Rejected under Finding `ADR16R1-006` as insufficient for generation-safe provenance.
- **Synchronous Immediate Termination Assumption**: Race condition hazard; rejected.

### Safety Invariant
`WORKER_STOPPED` is recorded only after authoritative `isTerminated == true` is verified.

### Persistence Impact
Creates table `stop_operations` with full relational provenance.

### State-Machine Impact
Drives terminal failure `RUNNING -> FAILED` (reason: `WORKER_STOPPED`).

### Restart Impact
Startup scanner reconciles all non-terminal stop operations.

### Auditability Impact
Full multi-stage provenance of stop operations recorded in both dedicated table and audit log.

### Implementation Ownership
TASK-P03-003.

### External Approval Required
YES.

---

## 18. D12: Atomic Terminal Transition & ended_at Ownership

### Source / Canonical Facts
- ADR-015 specifies SQLite engine and transactional audit log. Canonical auditability requires crash-consistent state transitions.
- SQLite Invariant: Absolutely ZERO network calls inside database transactions.

### Proposal Revision-6 Recommendation
- Unified StateStore atomic transition method (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Unified StateStore Atomic Terminal Transition Method (`AtomicTerminalTransition`) without Undeclared Columns**.
1. For terminal failure transitions (`RUNNING -> FAILED`, `DISPATCHED -> FAILED`):
   - Atomically committed in a single SQLite transaction:
     ```text
     1. tasks.state CAS update (old_state -> FAILED)
     2. task_attempts.ended_at = now
     3. task_attempts.recovery_disposition = terminal_disposition
     4. audit_events row append (TASK_STATE_TRANSITION with full error details in details_json)
     ```
   - **Schema Invariant (Finding `ADR16R2-003`)**: Neither the production `TaskAttempt` entity nor the SQLite `task_attempts` schema has a `terminal_error` or `failure_reason` column. All machine-readable failure dispositions are persisted strictly in `task_attempts.recovery_disposition`, while comprehensive diagnostic error payloads reside in append-only `audit_events.details_json`.
2. Decoupling Observation from State CAS:
   - Mere observation of transient activity states (such as `waiting_input` or `blocked` without formal escalation) does NOT trigger a TaskState CAS update.
   - Metadata dispositions (`task_attempts.recovery_disposition`) are updated without modifying `tasks.state` or `task_attempts.ended_at`.
3. If formal escalation occurs (`RUNNING -> BLOCKED`):
   - Atomically committed: TaskState CAS update to `BLOCKED` + audit event append.
   - `task_attempts.ended_at` remains `NULL` because `BLOCKED` is non-terminal.
4. Success Closure (`RUNNING -> REPORT_READY`):
   - Strictly NOT owned by TASK-P03-003. Owned exclusively by Phase P04 Evidence Collector upon report validation.
5. Transaction Rule: All upstream AO queries occur *before* opening the SQLite transaction. The transaction performs only local SQLite writes and commits immediately.

### Rationale
Ensures crash consistency without risking SQLite database lock contention or introducing spurious state machine transitions for read-only observations.

### Rejected Alternatives
- **Separate Un-coordinated Store Calls**: Risks partial writes on crash; rejected.
- **Network Calls Inside Transactions**: Violates ADR-015; strictly prohibited.

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

## 19. D13: Restart Scanner Architecture

### Source / Canonical Facts
- Power loss leaves in-flight tasks in `DISPATCHED` or `RUNNING` with open attempts (`ended_at IS NULL`).
- Finding `ADR16R1-007`: Startup recovery must enumerate ALL non-terminal stop operations (`STOP_REQUESTED` and `STOP_CALL_SUCCEEDED`), not just `STOP_REQUESTED`.
- Finding `ADR16R1-009`: No approved recovery retry policy exists. Recovery must use existing policy surfaces (`SUPERVISOR_ACTIVITY_POLL_INTERVAL`) without inventing new policies or numbers.

### Proposal Revision-6 Recommendation
- Synchronous startup recovery sweep covering all in-flight tasks and non-terminal operations with existing poll cadence (`GIẢ ĐỊNH`).

### Selected PROPOSED_DECISION
**PROPOSED_DECISION: Implement Synchronous Startup Recovery Sweep Covering Provisioning, Dispatch, and Purpose-Aware Generation-Prechecked Stops via Approved Adapter Methods**.
1. **Startup Enumeration**:
   On Supervisor daemon startup, prior to accepting incoming client API requests:
   - Query all `pair_provisioning_operations` in `stage = 'PROVISION_REQUESTED'`. Mark as `PROVISION_FAILED`, locking affected Pair lanes against automatic dispatch.
   - Query all tasks in `DISPATCHED` or `RUNNING` state.
   - Query all non-terminal `dispatch_operations` (`stage = 'SEND_REQUESTED'`).
   - Query all non-terminal `stop_operations` (`stage IN ('STOP_REQUESTED', 'STOP_CALL_SUCCEEDED')`).
2. **Approved Adapter Boundary (Finding `ADR16R3-008`)**:
   - Daemon availability is probed exclusively using approved adapter methods: `CheckHealth(ctx)` and/or `CheckReadiness(ctx)`. Raw calls to `GET /api/v1/sessions` (which is not exposed on the approved adapter) are strictly prohibited in automated orchestration.
   - Session reconciliation queries use approved adapter method `GetWorkerStatus(ctx, sessionID)`.
3. **Stop Operation Recovery Protocol (Governed by Stop Recovery Truth Table & Purpose Matrix)**:
   Startup scanner identifies in-flight operations via `SELECT * FROM stop_operations WHERE resolution_state = 'IN_FLIGHT'`.
   - `STOP_REQUESTED`:
     - If alive with matching generation: Automatic `/kill` reissue is PROHIBITED (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`, Finding `ADR16R4-001`). Set `resolution_state = 'STOP_REISSUE_REQUIRES_HUMAN'`, `resolved_at = now`. Retain active quarantine; require human operator reconciliation.
     - If `isTerminated == true`: Kill call was never confirmed transmitted/accepted. Retain `stage = 'STOP_REQUESTED'`, set `resolution_state = 'STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED'`, `resolved_at = now`. If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: transition task `-> FAILED` (reason: `WORKER_TERMINATION_UNKNOWN`). Do NOT record `WORKER_STOPPED` (Finding `ADR16R3-004`). For cleanup/maintenance stops: ZERO TaskState transition.
     - If generation mismatch: Retain `stage = 'STOP_REQUESTED'`, set `resolution_state = 'STOP_GENERATION_MISMATCH'`, `resolved_at = now`. If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: transition task `-> FAILED`.
     - If HTTP 404: Set `stage = 'STOP_TARGET_ABSENT'`, `resolution_state = 'STOP_TARGET_ABSENT'`, `resolved_at = now`. If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: transition task `-> FAILED` (reason: `SESSION_ABSENT`). Do NOT mark `STOP_TERMINATION_CONFIRMED` (Finding `ADR16R3-006`).
   - `STOP_CALL_SUCCEEDED`:
     - Kill signal was already accepted upstream. Reissuing `/kill` is redundant and prohibited.
     - If `isTerminated == true` with matching generation: Set `stage = 'STOP_TERMINATION_CONFIRMED'`, `resolution_state = 'TERMINATION_CONFIRMED'`, `resolved_at = now`. If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: `ended_at = now`, record `WORKER_STOPPED`. For cleanup/maintenance: ZERO TaskState transition.
     - If alive with matching generation: Evaluate restart-stable deadline `confirmation_deadline_at`. If `now < confirmation_deadline_at`: remain in `STOP_CALL_SUCCEEDED` and continue bounded observation until deadline. If `now >= confirmation_deadline_at`: set `resolution_state = 'STOP_CONFIRMATION_TIMEOUT'`, `resolved_at = now`. If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: transition task `-> FAILED`, do NOT log `WORKER_STOPPED` (Findings `ADR16R3-009`, `ADR16R4-007`).
     - If generation mismatch: Retain `stage = 'STOP_CALL_SUCCEEDED'`, set `resolution_state = 'STOP_GENERATION_MISMATCH'`, `resolved_at = now`.
     - If HTTP 404: Set `stage = 'STOP_TARGET_ABSENT'`, `resolution_state = 'STOP_TARGET_ABSENT'`, `resolved_at = now`.
4. **Dispatch & In-Flight Task Recovery**:
   - Probe AO daemon availability via approved `CheckHealth` / `CheckReadiness`.
   - **If AO is Reachable**:
     - Reconcile in-flight tasks against public `SessionView` via `GetWorkerStatus(ctx, sessionID)` using opaque string generation matching.
     - If `stage == 'SEND_REQUESTED'`: execute D5 uncertain delivery failure and quarantine.
     - If session missing (404) or `isTerminated == true`: execute `AtomicTerminalTransition` (`-> FAILED`, `ended_at = now`).
     - If generation mismatch: fail attempt with `STALE_EXECUTION_GENERATION`.
     - If session active/idle with matching generation: resume standard lifecycle observation.
   - **If AO is Unreachable**:
     - Do NOT blindly fail in-flight tasks.
     - Mark tasks with `recovery_disposition = 'RECOVERY_PENDING'`.
     - Lock affected Pair lanes against new task dispatch.
     - Schedule background reconciliation retries using the existing caller-injected `SUPERVISOR_ACTIVITY_POLL_INTERVAL` as the checking cadence.
     - Do NOT introduce any new operational policies, retry counts, or backoff formulas.

### Rationale
Resolves Findings `ADR16R1-007` and `ADR16R1-009`. Enumerating all non-terminal stop operations closes the crash window where kill was accepted but termination was unconfirmed. Using `SUPERVISOR_ACTIVITY_POLL_INTERVAL` leverages existing unhardcoded policy surfaces without inventing unapproved configuration parameters.

### Rejected Alternatives
- **Scanning `STOP_REQUESTED` only**: Rejected under Finding `ADR16R1-007`; misses operations in `STOP_CALL_SUCCEEDED`.
- **Inventing New Recovery Policies**: Rejected under Finding `ADR16R1-009`; violates policy freeze.

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
2. `terminal_generation`: The upstream opaque string (`SessionView.TerminalGeneration` / `Metadata.RuntimeLaunchID`) fencing the session against terminal restarts or resets.
3. Every `TaskAttempt` snapshot permanently binds this tuple. Any upstream observation bearing a mismatched session or generation string is rejected as foreign, eliminating cross-turn attribution hazards.

---

## 21. Agy V1 Runtime Generation Fence

1. For V1, the primary worker harness is `agy` running in interactive TUI mode inside an AO-managed ConPTY pane.
2. The authoritative execution fence is upstream AO's `terminalGeneration` (exposed as an opaque `string` on `SessionView` matching `Metadata.RuntimeLaunchID`).
3. Comparison semantics are strictly string equality (`MATCH` vs `MISMATCH`). There is NO numeric increment, integer arithmetic, or ordering comparison.
4. **Scope Guard**: This fence applies specifically to V1 `agy` TUI mode. It is NOT generalized to non-TUI headless modes or arbitrary processes, which remain subject to future phase governance.

---

## 22. Lifecycle Observation Matrix

The complete lifecycle evaluation matrix governing TASK-P03-003 under graph-compatible rules:

| TaskState | Saga Stage | Observed Activity | Generation Match | Proven Facts | Unproven Facts | Transition | ended_at Action | Disposition / Quarantine Action |
|---|---|---|---|---|---|---|---|---|
| `DISPATCHED` | `DISPATCH_BOUND` | `idle` | Match | Pre-send ready | Execution occurred | None | Keep NULL | None |
| `DISPATCHED` | `DISPATCH_BOUND` | `waiting_input` | Match | Waiting input | Execution occurred | None | Keep NULL | None |
| `DISPATCHED` | `DISPATCH_BOUND` | `active` | Match | Foreign/lingering turn | Current task sent | None | Keep NULL | `PRE_SEND_BLOCKED` |
| `DISPATCHED` | `DISPATCH_BOUND` | `blocked` | Match | Dialog blocked | Current task sent | None | Keep NULL | `PRE_SEND_BLOCKED` |
| `DISPATCHED` | `DISPATCH_BOUND` | `exited` | Match | Process exit signal | Termination/crash | None | Keep NULL | `PRE_SEND_BLOCKED` |
| `DISPATCHED` | `DISPATCH_BOUND` | `isTerminated: true` | Match | Session dead | Cause of death | `-> FAILED` | Set `now` | None |
| `DISPATCHED` | `SEND_REQUESTED` | Any | Match/Mismatch | Delivery unknown | Delivery success | `-> FAILED` | Set `now` | **`UNCERTAIN_DELIVERY_QUARANTINE`** |
| `DISPATCHED` | `SEND_CONFIRMED` | `active` | Match | Upstream write accepted, active | Task completion | `-> RUNNING` | Keep NULL | None |
| `DISPATCHED` | `SEND_CONFIRMED` | `idle` | Match | Upstream write accepted, idle | Whether prompt ran or missed | `-> RUNNING` | Keep NULL | `MISSED_ACTIVE_WINDOW` |
| `DISPATCHED` | `SEND_CONFIRMED` | `waiting_input` | Match | Agent at prompt | Failure | `-> RUNNING` | Keep NULL | `AO_WAITING_INPUT_OBSERVED` |
| `DISPATCHED` | `SEND_CONFIRMED` | `blocked` | Match | Blocked on dialog | Attempt failure | `-> RUNNING` | Keep NULL | `AO_BLOCKED_DECISION` |
| `DISPATCHED` | `SEND_CONFIRMED` | `exited` | Match | Process exited | Session dead | None | Keep NULL | Process exit rule |
| `DISPATCHED` | `SEND_CONFIRMED` | `isTerminated: true` | Match | Session terminated | Crash vs clean | `-> FAILED` | Set `now` | None |
| `RUNNING` | `SEND_CONFIRMED` | `active` | Match | Active tool execution | Task completion | None | Keep NULL | None |
| `RUNNING` | `SEND_CONFIRMED` | `idle` | Match | Turn complete | Report valid (`AO_IDLE != REPORT_READY`) | None (for P04) | Keep NULL | P04 handoff |
| `RUNNING` | `SEND_CONFIRMED` | `waiting_input` | Match | Paused at prompt | Failure | None (D9) | Keep NULL | `AO_WAITING_INPUT_OBSERVED` |
| `RUNNING` | `SEND_CONFIRMED` | `blocked` | Match | Blocked on tool permission | Attempt failure | None (D10) | Keep NULL | `AO_BLOCKED_DECISION` |
| `RUNNING` | `SEND_CONFIRMED` | `exited` | Match | Process exited | Session dead | None | Keep NULL | Process exit rule |
| `RUNNING` | `SEND_CONFIRMED` | `isTerminated: true` | Match | Session dead | Crash vs clean | `-> FAILED` | Set `now` | None |
| `RUNNING` | `SEND_CONFIRMED` | 404 Not Found | N/A | Session purged | Purge reason | `-> FAILED` | Set `now` | None |
| `RUNNING` | `SEND_CONFIRMED` | AO Unavailable | N/A | Daemon unreachable | Worker death | None | Keep NULL | `RECOVERY_PENDING` (poll cadence) |

---

## 23. Crash / Restart Matrix

| Crash Point | Host Failure Window | Persisted State on Disk | Detection on Startup | Reconciled Action |
|---|---|---|---|---|
| **T0** | Before session spawn | Pair idle, no session | Normal startup | None; Pair available for provisioning |
| **T1** | During spawn wire call | `pair_provisioning_operations.PROVISION_REQUESTED` | Operation unconfirmed on startup | Advance to `PROVISION_FAILED`; lock Pair lane in quarantine; unowned AO session not auto-killed; operator-audited cleanup |
| **T2** | AO spawned, before DB commit | `pair_provisioning_operations.PROVISION_REQUESTED`, no `worker_sessions` | Operation unconfirmed on startup | Advance to `PROVISION_FAILED`; lock Pair lane in quarantine; unowned AO session not auto-killed; operator-audited cleanup |
| **T3** | Session committed, before dispatch | `worker_sessions` active, `PROVISION_CONFIRMED` | Valid idle session | Ready for dispatch |
| **T4** | `DISPATCH_BOUND` committed | `tasks.state = DISPATCHED` | Saga = `DISPATCH_BOUND` | Check pre-send admissibility; execute `/send` |
| **T5** | `SEND_REQUESTED` committed, crash during `/send` | Saga = `SEND_REQUESTED` | Delivery unknown | `-> FAILED`, `ended_at = now`, quarantine, `-> HUMAN_REQUIRED` |
| **T6** | `SEND_CONFIRMED` committed | Saga = `SEND_CONFIRMED` | Write accepted | Check AO activity; resume observation |
| **T7** | `RUNNING` state, daemon power loss | `tasks.state = RUNNING` | In-flight attempt | Reconcile via `GetWorkerStatus` against public SessionView |
| **T8** | `STOP_REQUESTED` committed, crash before `/kill` | Saga = `STOP_REQUESTED` | Stop intent logged | Reconcile via `GetWorkerStatus`; if alive: auto-reissue prohibited (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`), advance to `STOP_REISSUE_REQUIRES_HUMAN`, retain quarantine; if dead: advance to `resolution_state = 'STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED'` (stage remains `STOP_REQUESTED`); if mismatch: `STOP_GENERATION_MISMATCH` |
| **T9** | `STOP_CALL_SUCCEEDED` committed, crash before confirm | Saga = `STOP_CALL_SUCCEEDED` | Kill accepted | Observe within persisted `confirmation_deadline_at`; if `isTerminated == true` advance to `STOP_TERMINATION_CONFIRMED` (`TERMINATION_CONFIRMED`, `WORKER_STOPPED` if live stop); if 404 advance to `STOP_TARGET_ABSENT`; if deadline reached advance to `resolution_state = 'STOP_CONFIRMATION_TIMEOUT'` (stage remains `STOP_CALL_SUCCEEDED`) |

---

## 24. StateStore Transaction Boundaries

All operations adhere strictly to ADR-015:
1. `PrepareDispatch`: Single SQLite transaction committing `tasks.state = DISPATCHED`, allocating `task_attempts`, binding `session_id`, and creating `dispatch_operations` in `DISPATCH_BOUND`.
2. `RecordSendRequested`: Single SQLite transaction updating `dispatch_operations.stage = SEND_REQUESTED` immediately before network call.
3. `RecordSendConfirmed`: Single SQLite transaction updating `dispatch_operations.stage = SEND_CONFIRMED` immediately after HTTP 200.
4. `AtomicTerminalTransition`: Single SQLite transaction committing TaskState update, `task_attempts.ended_at`, error details, and audit log.
5. Strict Boundary: No network call or AO interaction is ever permitted inside these transaction blocks.

---

## 25. Audit Event / Operation Provenance

The following structured audit event types are established (strictly aligned to Token Registry):
- `PAIR_SESSION_PROVISION_REQUESTED`: Pre-effect session spawn intent recorded.
- `PAIR_SESSION_PROVISIONED`: Infrastructure session bound.
- `TASK_DISPATCH_BOUND`: Task contract allocated and attempt bound.
- `DISPATCH_SEND_REQUESTED`: Pre-effect dispatch intent recorded.
- `DISPATCH_SEND_CONFIRMED`: Upstream write acceptance recorded.
- `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED`: Delivery unknown; quarantine active.
- `QUARANTINE_RESOLVED_PHYSICAL`: Verified termination via `/kill` confirmed.
- `QUARANTINE_RESOLVED_ADMINISTRATIVE`: Explicit operator risk acceptance logged.
- `WORKER_BLOCKED_ON_DECISION`: Tool permission dialog observed.
- `STOP_OPERATION_REQUESTED`: Kill signal intent recorded.
- `STOP_OPERATION_CALL_SUCCEEDED`: Kill signal acceptance recorded.
- `STOP_OPERATION_CONFIRMED`: Authoritative termination confirmed.
- `STOP_OPERATION_TARGET_ABSENT`: Upstream 404 absence observed; physical termination unproven.
- `STOP_CONFIRMATION_TIMEOUT`: Expiration of stop confirmation deadline without confirmed exit.
- `MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM`: Ambiguity handoff to P04 logged.
- `STARTUP_RECOVERY_SWEEP_STARTED`: Supervisor startup recovery scan initiated.
- `STARTUP_RECOVERY_SWEEP_COMPLETED`: Supervisor startup recovery scan completed.

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

1. Schema Updates (Unified Authoritative DDL):
   - `worker_sessions` (enforcing 1:1 bidirectional uniqueness, nullable worktree_path, independent quarantine state, Findings `ADR16R3-002`, `ADR16R4-002`, `ADR16R4-006`):
     ```sql
     CREATE TABLE worker_sessions (
         pair_id TEXT PRIMARY KEY REFERENCES pairs(pair_id) ON DELETE RESTRICT,
         session_id TEXT NOT NULL UNIQUE,
         runtime_type TEXT NOT NULL,
         worktree_path TEXT,
         worker_agent_id TEXT NOT NULL,
         status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'IDLE', 'TERMINATED')),
         terminal_generation TEXT NOT NULL,
         quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED')),
         created_at TEXT NOT NULL,
         updated_at TEXT NOT NULL
     );
     ```
   - `pair_provisioning_operations` (durable pre-binding quarantine & concurrency guard, Findings `ADR16R3-007`, `ADR16R4-005`):
     ```sql
     CREATE TABLE pair_provisioning_operations (
         operation_id TEXT PRIMARY KEY,
         pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
         stage TEXT NOT NULL CHECK (stage IN ('PROVISION_REQUESTED', 'PROVISION_CONFIRMED', 'PROVISION_FAILED', 'PROVISION_RESOLVED')),
         client_token TEXT NOT NULL,
         session_id TEXT,
         requested_at TEXT NOT NULL,
         completed_at TEXT,
         resolved_at TEXT,
         resolved_by TEXT,
         resolution_notes TEXT
     );
     CREATE UNIQUE INDEX idx_pair_provisioning_unresolved ON pair_provisioning_operations(pair_id) WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED');
     ```
   - `dispatch_operations` (enforcing `# ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION`, Finding `ADR16R3-003`):
     ```sql
     CREATE TABLE dispatch_operations (
         operation_id TEXT PRIMARY KEY,
         attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
         pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
         task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
         session_id TEXT NOT NULL,
         terminal_generation TEXT NOT NULL,
         stage TEXT NOT NULL CHECK (stage IN ('DISPATCH_BOUND', 'SEND_REQUESTED', 'SEND_CONFIRMED')),
         requested_at TEXT NOT NULL,
         confirmed_at TEXT,
         resolution_state TEXT
     );
     ```
   - `stop_operations` (purpose-aware, preserving wire-effect stage, restart-stable deadline, Findings `ADR16R4-001`, `ADR16R4-003`, `ADR16R4-004`, `ADR16R4-007`):
     ```sql
     CREATE TABLE stop_operations (
         operation_id TEXT PRIMARY KEY,
         purpose TEXT NOT NULL CHECK (purpose IN ('RUNNING_ATTEMPT_STOP', 'QUARANTINE_CLEANUP', 'PAIR_MAINTENANCE')),
         pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
         task_id TEXT REFERENCES tasks(task_id) ON DELETE RESTRICT,
         contract_id TEXT REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
         attempt_id TEXT REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
         session_id TEXT NOT NULL,
         terminal_generation TEXT NOT NULL,
         stage TEXT NOT NULL CHECK (stage IN ('STOP_REQUESTED', 'STOP_CALL_SUCCEEDED', 'STOP_CALL_FAILED', 'STOP_TERMINATION_CONFIRMED', 'STOP_TARGET_ABSENT')),
         actor TEXT NOT NULL,
         requested_at TEXT NOT NULL,
         call_completed_at TEXT,
         confirmation_deadline_at TEXT,
         termination_confirmed_at TEXT,
         resolved_at TEXT,
         resolution_state TEXT NOT NULL DEFAULT 'IN_FLIGHT'
     );
     ```
   - `task_attempts` candidate extensions (Findings `ADR16R4-002`, Section 14):
     Columns added to `task_attempts`:
     `session_id TEXT`, `terminal_generation TEXT`, `recovery_disposition TEXT`, `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))`.
     (Zero undeclared columns; diagnostic error details reside in append-only `audit_events.details_json`).
2. Migrations: Implementation requires a schema migration executed under future authorized task contract. Zero migrations executed in this draft.

---

## 28. Canonical Documentation Impact

Upon formal External Supervisor acceptance of ADR-016, the following canonical documents will be reconciled:
1. `docs/04_ARCHITECTURE.md`: Update sequence diagrams to reflect decoupled pair session provisioning and durable 3-stage dispatch saga.
2. `docs/05_DOMAIN_MODEL.md`: Formalize Model A persistence relationship and snapshot columns, mapping canonical fields `runtime_type`, `worker_agent_id`, and `status`.
3. `docs/06_WORKFLOW_STATE_MACHINE.md`: Cross-reference observation-level blocked handling and one-way formal escalation.
4. `docs/08_TASK_CONTRACT.md`: Document attempt execution identity snapshot binding.
5. `docs/12_UPSTREAM_INTEGRATION.md`: Document wire route `/kill`, opaque string generation fence, and strict pre-send admissibility whitelist.
6. `docs/14_FAILURE_RECOVERY.md`: Document quarantine guard, Class A/B/C clearance rules, non-terminal stop recovery, and restart recovery scanner.
7. `docs/22_MODULE_PROVENANCE.md`: Register new store entities.

---

## 29. TASK-P03-003 Implementation Boundary

`TASK-P03-003` owns exclusively:
- Domain persistence of `worker_sessions`, `dispatch_operations`, `stop_operations`, and `task_attempts` snapshot fields.
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
- NO automatic killing of unowned AO sessions.
- NO blind retries or synthetic heartbeats.
- NO SSE `/api/v1/events` integration in this task.

---

## 31. Consequences

### Positive
- Enforces fail-closed automated duplicate dispatch prevention (`AUTOMATED_DUPLICATE_DISPATCH_PREVENTION = FAIL_CLOSED`), strictly preventing automated retries while execution state is unverified.
- Resolves relational domain contradictions without introducing cyclic foreign keys.
- Decouples sandbox provisioning from task dispatch, enabling warm session reuse.
- Strictly conforms to the canonical workflow state graph without inventing edges.
- Enforces strict component boundaries and clean testability.

### Negative / Trade-offs
- Unknown-delivery crashes require explicit operator resolution (Class A kill or Class B/C administrative risk resolution) before tasks can be retried. Class B and Class C knowingly accept residual duplicate-execution risk as explicit human risk overrides.
- Adding dedicated operations tables slightly increases database schema surface.

---

## 32. Rejected Alternatives

- **Model B (Historical Table + Pointer)**: Rejected due to cyclic FKs in SQLite.
- **Model C (Join Table)**: Rejected due to unnecessary schema complexity.
- **Automatic Orphan Reaper Deletions**: Rejected under Finding `ADR16R1-001` as unowned sessions cannot be proven owned by Supervisor.
- **Foreign Key to `task_attempts(id)`**: Rejected under Finding `ADR16R1-002` as non-existent column.
- **`BLOCKED -> RUNNING` State Graph Resumption**: Rejected under Finding `ADR16R1-003` as non-existent edge.
- **Direct `RUNNING -> REVIEWING` Edge**: Rejected under Finding `ADR16R2-001` as contrary to the 22-edge canonical workflow graph.
- **Ephemeral Spawn Quarantine**: Rejected under Finding `ADR16R2-002` because pre-binding crashes require restart-durable tracking in `pair_provisioning_operations`.
- **Undeclared `task_attempts.terminal_error`**: Rejected under Finding `ADR16R2-003` as non-existent column.
- **Reissuing `/kill` Without Generation Fencing**: Rejected under Finding `ADR16R2-004` as risk of killing newer execution generation.
- **Treating 404 as `WORKER_STOPPED`**: Rejected under Finding `ADR16R2-005` as absence does not prove affirmative termination.
- **D8 Transition Order Contradiction**: Rejected under Finding `ADR16R2-006` because `DISPATCHED -> FAILED` cannot be executed after `RUNNING`.
- **Unconditional Zero-Duplicate Claim**: Rejected under Finding `ADR16R2-007` because Class C is an authorized human risk override.
- **Multiple Dispatch Sagas per TaskAttempt**: Rejected under Finding `ADR16R2-008` as contrary to `# ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION`.
- **Non-Unique `worker_sessions.session_id`**: Rejected under Finding `ADR16R2-009` because an AO session must bind to at most one Pair.
- **Inaccurate Edge Accounting**: Rejected under Finding `ADR16R3-001`; graph consists strictly of 22 executable domain transitions + 1 initial + 2 terminal pseudo-edges = 25 graph edges.
- **Multiple `worker_sessions` Schema Versions**: Rejected under Finding `ADR16R3-002`; unified single schema with `UNIQUE(session_id)` and ISO8601 timestamps.
- **Multiple `dispatch_operations` Schema Versions**: Rejected under Finding `ADR16R3-003`; unified single schema with `UNIQUE(attempt_id)`, `confirmed_at`, and `resolution_state`.
- **Overclaiming Intentional Stop on `STOP_REQUESTED` Crash**: Rejected under Finding `ADR16R3-004`; `WORKER_STOPPED` requires verified `STOP_CALL_SUCCEEDED`.
- **Stale Crash Matrix Semantics**: Rejected under Finding `ADR16R3-005`; matrix updated to reflect durable provisioning and purpose-aware generation-prechecked stop recovery.
- **Unbounded / Ambiguous Class-B Absence Clearance**: Rejected under Finding `ADR16R3-006`; 404 requires explicit operator risk resolution to clear quarantine.
- **Concurrent Unresolved Pair Provisioning**: Rejected under Finding `ADR16R3-007`; partial unique index enforces at most one unresolved provisioning per Pair.
- **Raw ListSessions Transport Call in Orchestration**: Rejected under Finding `ADR16R3-008`; orchestration uses approved adapter surface (`CheckHealth`, `CheckReadiness`, `GetWorkerStatus`).
- **Unbounded Stop Confirmation Observation**: Rejected under Finding `ADR16R3-009`; observation bounded by `SUPERVISOR_KILL_STOP_TIMEOUT`.
- **Governance Gate Inconsistency**: Rejected under Finding `ADR16R3-010`; live governance state synchronized.
- **Generation-Atomic Kill Claim**: Rejected under Finding `ADR16R4-001`; pinned `/kill` takes session ID only; GET->POST is a precheck with TOCTOU window.
- **Automatic `/kill` Reissue on Restart**: Rejected under Finding `ADR16R4-001`; automatic kill reissue prohibited without atomic generation fence.
- **Overwriting Attempt Quarantine with Diagnostic Disposition**: Rejected under Finding `ADR16R4-002`; dedicated `task_attempts.quarantine_state` preserves safety truth.
- **Unconditional Stop TaskState Transition**: Rejected under Finding `ADR16R4-003`; `purpose` binds TaskState effects strictly to `RUNNING_ATTEMPT_STOP`.
- **Stop Stage Provenance Collapse**: Rejected under Finding `ADR16R4-004`; `stage` preserves wire-effect facts while `resolution_state` records outcome.
- **Second Spawn Authorization on `PROVISION_CONFIRMED`**: Rejected under Finding `ADR16R4-005`; existing WorkerSession requires reuse or resumption.
- **Mandatory `worktree_path TEXT NOT NULL`**: Rejected under Finding `ADR16R4-006`; pinned AO public contract does not expose workspace path.
- **Restart-Resettable Stop Confirmation Timeout**: Rejected under Finding `ADR16R4-007`; absolute `confirmation_deadline_at` frozen in SQLite.
- **Vocabulary Drift Across Sections**: Rejected under Finding `ADR16R4-008`; unified under Canonical Token Registry.
- **Numeric / Monotonic TerminalGeneration**: Rejected under Finding `ADR16R1-004` as contrary to upstream DTO contract.
- **Predefining "no modified files = FAILED"**: Rejected under Finding `ADR16R1-008` as contrary to task boundaries and legitimate no-op contracts.
- **Inventing Recovery Retry Policies**: Rejected under Finding `ADR16R1-009` as contrary to policy inventory rules.
- **Blind Resend on Crash**: Rejected as critical safety violation.
- **Direct `DISPATCHED -> HUMAN_REQUIRED`**: Rejected as invalid state machine transition.
- **`ActivityExited` Automatic Failure**: Rejected because managed session remains open.
- **Report Probing in TASK-P03-003**: Rejected as strict component boundary violation.

---

## 33. External Approval Gate

```text
PROPOSAL_P03_002 = EXTERNAL_APPROVED
ADR_016 = REVISION_4_READY_FOR_EXTERNAL_REAUDIT
ADR_016_ACCEPTANCE = NOT_GRANTED
TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_PENDING_ADR_016_APPROVAL
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_ADR_016_REAUDIT_004
```

Execution is halted awaiting independent External Supervisor re-audit of ADR-016 Revision 4.
