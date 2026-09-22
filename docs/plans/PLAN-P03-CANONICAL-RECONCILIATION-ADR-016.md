# PLAN-P03-CANONICAL-RECONCILIATION-ADR-016: Canonical Specification Reconciliation Scope Plan

> **Authority**: Formulated pursuant to accepted [ADR-016](file:///d:/TU_CODE/ai-supervisor/docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md) (EXTERNAL_APPROVED by External Supervisor Re-Audit 006, commit `40d51ffe9af4dab5545f060d5bee2ffd441b6109`), `docs/24_CHANGE_GOVERNANCE.md` (Decision Hierarchy Level 2: Approved ADR takes precedence over Level 3: Canonical Architecture and Level 4: Requirement Specifications), and External Supervisor directive.
> **Status**: DRAFT PLAN — PENDING EXTERNAL SUPERVISOR APPROVAL
> **Mode**: PLANNING ONLY (Zero canonical specification mutation, zero task contract creation, zero migration, zero production code in this turn).
> **Date**: 2026-09-23

---

## 1. Governance Baseline Distinction & Scope Boundary

To ensure complete auditability and prevent historical confusion, the reconciliation history is rigorously partitioned into two separate baselines:

| Dimension | Baseline A: Previous Approved Reconciliation | Baseline B: New ADR-016 Reconciliation (This Plan) |
| :--- | :--- | :--- |
| **Originating Proposal / ADR** | `PROPOSAL-P03-001` (Core directions approved) | `PROPOSAL-P03-002` Revision 6 -> **ADR-016 Accepted** (Re-Audit 006) |
| **Audit Status** | `P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED` (Revision 3) | **`CANONICAL_SPEC_RECONCILIATION_PLANNING` (NOT STARTED / PENDING REVIEW)** |
| **Reconciled Scope** | Preflight probes (`CheckReadiness`, `ListAgents`, `GetAgentReadiness`, `GetAPIContract`), OpenAPI compatibility signal, CDC event subscription deferral in `docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, and `docs/phases/P03_AO_INTEGRATION.md`. | Durable dispatch saga, Pair session provisioning decoupling, Model A attempt snapshot persistence, double-gated quarantine, strict pre-send whitelist, purpose-aware stop lifecycle, synchronous startup recovery sweep, and store entity provenance across `docs/04`, `docs/05`, `docs/06`, `docs/08`, `docs/12`, `docs/14`, `docs/22`, and affected references in `docs/21` and `docs/phases/P03_AO_INTEGRATION.md`. |
| **Claim Distinction** | Baseline A is permanently locked and approved. | **Baseline B is strictly in planning mode. Zero claim of approval, completion, or release is made.** |

---

## 2. ADR-016 Decisions Traceability Matrix

The 13 binding decisions of accepted ADR-016 are mapped to the canonical specification documents:

| Decision ID | ADR-016 Decision Summary | Primary Target Document & Section | Secondary / Cross-Referenced Documents |
| :--- | :--- | :--- | :--- |
| **D1** | Pair-Scoped WorkerSession Storage Alignment & Model A Attempt Persistence | `docs/05_DOMAIN_MODEL.md` §1, §2 | `docs/22_MODULE_PROVENANCE.md` §1 |
| **D2** | Formally Amend ADR-012 §10 (Decouple Pair Provisioning from Task Dispatch) | `docs/04_ARCHITECTURE.md` §2.1 | `docs/08_TASK_CONTRACT.md` §1, `docs/21_TRACEABILITY_MATRIX.md` |
| **D3** | Fail-Closed Spawn Disposition (`CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`) & Durable Provisioning Operations | `docs/05_DOMAIN_MODEL.md` §1, §2; `docs/14_FAILURE_RECOVERY.md` §1 | `docs/22_MODULE_PROVENANCE.md` §1 |
| **D4** | Dedicated `dispatch_operations` Table with 1:1 Attempt Cardinality & Candidate TaskAttempt Extensions | `docs/04_ARCHITECTURE.md` §2.1; `docs/05_DOMAIN_MODEL.md` §1, §2 | `docs/08_TASK_CONTRACT.md` §1, `docs/22_MODULE_PROVENANCE.md` §1 |
| **D5** | Fail-Closed Terminal Failure (`DISPATCHED -> FAILED`) with Immediate Quarantine | `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/14_FAILURE_RECOVERY.md` §1 |
| **D6** | Option D: Double-Gated Quarantine with Dedicated Attempt Quarantine State & Clearance Classes | `docs/05_DOMAIN_MODEL.md` §1, §2; `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/14_FAILURE_RECOVERY.md` §1 |
| **D7** | Strict Whitelist Pre-Send Admissibility (`idle`, `waiting_input` only) via Approved Adapter Surface | `docs/12_UPSTREAM_INTEGRATION.md` §1 | `docs/phases/P03_AO_INTEGRATION.md` §2 |
| **D8** | Option C: Persist Ambiguity Disposition (`MISSED_ACTIVE_WINDOW`) & Downstream Handoff with Pre-Check | `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/14_FAILURE_RECOVERY.md` §1 |
| **D9** | Preserve `RUNNING`, Emit Diagnostic Telemetry, Continue Polling on `waiting_input` | `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/12_UPSTREAM_INTEGRATION.md` §1 |
| **D10** | Graph-Compatible Blocked Handling: Preserve `RUNNING` with `AO_BLOCKED_DECISION`; Close Attempt at `BLOCKED -> HUMAN_REQUIRED` | `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/08_TASK_CONTRACT.md` §3 |
| **D11** | Purpose-Aware Stop Lifecycle with Generation Precheck, Restart-Stable Deadlines, & Preserved Provenance | `docs/05_DOMAIN_MODEL.md` §1, §2; `docs/12_UPSTREAM_INTEGRATION.md` §1; `docs/14_FAILURE_RECOVERY.md` §1 | `docs/06_WORKFLOW_STATE_MACHINE.md` §2, `docs/22_MODULE_PROVENANCE.md` §1 |
| **D12** | Unified StateStore Atomic Terminal Transition (`AtomicTerminalTransition`) without Undeclared Columns | `docs/05_DOMAIN_MODEL.md` §2; `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/22_MODULE_PROVENANCE.md` §1 |
| **D13** | Synchronous Startup Recovery Sweep Covering Provisioning, Dispatch, Stops, and Mandatory Blocked Escalation | `docs/14_FAILURE_RECOVERY.md` §1 | `docs/06_WORKFLOW_STATE_MACHINE.md` §2, `docs/12_UPSTREAM_INTEGRATION.md` §1 |

---

## 3. Itemized Canonical Specification Reconciliation Scope

### Item S1: `docs/04_ARCHITECTURE.md` — Section 2.1 (Task Dispatch Flow) & Section 3.1 (AOAdapter Boundary)

- **Target File & Section**: `docs/04_ARCHITECTURE.md`, Section 2.1 ("Task Dispatch Flow" sequence diagram) and Section 3.1 ("AOAdapter Boundary").
- **Current Content Causing Drift**:
  - Section 2.1 sequence diagram lines 75-79 depict `AOAdapter.createWorkerSession(projectId, harness="antigravity")` executing *inside* the task dispatch flow immediately following `READY -> DISPATCHED`.
  - Section 3.1 includes `POST /api/v1/sessions/{id}/restore` as an active process control endpoint.
- **Governing ADR-016 Decisions**: **D2** (Decouple Pair Provisioning from Task Dispatch, amending ADR-012 §10), **D4** (Durable 3-Stage Dispatch Saga), and **D11** (Process control wire route `/kill`, absence of `/restore`).
- **Required Specification Synchronization**:
  1. *Sequence Diagram Update*: Update Section 2.1 Mermaid diagram to decouple Pair session provisioning from task prompt dispatch. The task dispatch flow commences with a pre-existing, verified Pair WorkerSession in `idle` or `waiting_input` status (or triggers Pair provisioning if no session exists under `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`).
  2. *Durable Dispatch Saga Steps*: Reflect the 3-stage dispatch saga in `dispatch_operations`:
     - Stage 1: `DISPATCH_BOUND` (persisted simultaneously with TaskState `READY -> DISPATCHED` and candidate `TaskAttempt` execution snapshot);
     - Stage 2: `SEND_REQUESTED` (persisted immediately prior to issuing loopback `POST /api/v1/sessions/{id}/send`);
     - Stage 3: `SEND_CONFIRMED` (persisted upon HTTP 200/204 response from AO).
  3. *Remove `/restore`*: In Section 3.1, remove `POST /api/v1/sessions/{id}/restore`. Pinned AO v0.13.0 does not implement session restore; process termination is one-way via `/kill`.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (entities `PairProvisioningOperation`, `DispatchOperation`), `docs/12_UPSTREAM_INTEGRATION.md` (wire methods).
- **Execution Order**: Phase 2 (Architecture & Adapter Synchronization).
- **Consistency Checks**:
  - Assert sequence diagram shows no inline worktree creation during normal dispatch over an existing session.
  - Assert `/restore` string is completely absent from `docs/04_ARCHITECTURE.md`.
- **Completion Criteria**: Sequence diagram accurately models decoupled provisioning and durable 3-stage dispatch saga adhering to D2, D4, and D11.

---

### Item S2: `docs/05_DOMAIN_MODEL.md` — Section 1 (Domain Entities Diagram) & Section 2 (Entity Dictionary)

- **Target File & Section**: `docs/05_DOMAIN_MODEL.md`, Section 1 (Class Diagram) and Section 2 (Entity Dictionary & Invariants).
- **Current Content Causing Drift**:
  - `WorkerSession` entity in class diagram lacks `quarantine_state` attribute.
  - `TaskAttempt` entity lacks snapshot columns (`worker_session_id`, `runtime_type`, `worker_agent_id`, `worktree_path`, `recovery_disposition`, `quarantine_state`).
  - Class diagram and dictionary lack entities for durable saga tracking: `PairProvisioningOperation`, `DispatchOperation`, and `StopOperation`.
  - Invariant 1 in Section 2 implies `worktree_path` is always known and non-null, whereas pinned AO v0.13.0 public API does not return worktree path (`worktree_path` is `TEXT NULL`).
- **Governing ADR-016 Decisions**: **D1** (Model A Attempt Persistence & Nullable Worktree), **D3** (`pair_provisioning_operations`), **D4** (`dispatch_operations` with 1:1 Attempt Cardinality), **D6** (Double-Gated Quarantine), **D11** (`stop_operations`), **D12** (Zero Undeclared Columns).
- **Required Specification Synchronization**:
  1. *Update `WorkerSession`*: Add attribute `+QuarantineState quarantine_state` (`ACTIVE` / `CLEARED`). Note `worktree_path: string | null`.
  2. *Update `TaskAttempt`*: Formalize Model A persistence relationship while preserving domain relationship `Pair "1" o-- "1" WorkerSession`. Add snapshot and quarantine attributes:
     - `+string worker_session_id`
     - `+string runtime_type`
     - `+string worker_agent_id`
     - `+string worktree_path (nullable)`
     - `+RecoveryDisposition recovery_disposition (nullable)`
     - `+QuarantineState quarantine_state (ACTIVE / CLEARED)`
  3. *Add New Aggregate Entities*:
     - `PairProvisioningOperation`: `operation_id`, `pair_id`, `worker_session_id`, `stage` (`SPAWN_REQUESTED`, `SPAWN_CONFIRMED`), `created_at`, `resolved_at`. Partial unique index enforces one in-flight spawn per pair.
     - `DispatchOperation`: `dispatch_id`, `attempt_id`, `pair_id`, `worker_session_id`, `stage` (`DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`), `resolution_state`, `created_at`, `resolved_at`. Enforces `UNIQUE(attempt_id)`.
     - `StopOperation`: `stop_id`, `pair_id`, `worker_session_id`, `task_id`, `attempt_id`, `purpose` (`RUNNING_ATTEMPT_STOP`, `QUARANTINE_CLEANUP`, `PAIR_MAINTENANCE`), `target_generation`, `confirmation_deadline_at`, `stage` (`STOP_REQUESTED`, `STOP_CALL_SUCCEEDED`, `STOP_TERMINATION_CONFIRMED`, `STOP_TARGET_ABSENT`, `STOP_TIMED_OUT`), `resolution_state`, `created_at`, `resolved_at`.
  4. *Invariants & Vocabulary*: Document invariant `# ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION`, `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`, and zero undeclared columns on `task_attempts` (rejecting invented columns like `terminal_error`).
- **Co-requisite Synchronizations**: `docs/08_TASK_CONTRACT.md` (execution identity separation), `docs/22_MODULE_PROVENANCE.md` (StateStore entity registry).
- **Execution Order**: Phase 1 (Core Domain & State Model Synchronization).
- **Consistency Checks**:
  - Verify all 3 new entities appear in class diagram and entity dictionary.
  - Verify `task_attempts` snapshot fields match ADR-016 D1 exactly.
- **Completion Criteria**: Model A persistence, double-gated quarantine, and durable operations are formally defined with exact column and constraint invariants.

---

### Item S3: `docs/06_WORKFLOW_STATE_MACHINE.md` — Section 2 (Canonical States & Transition Rules)

- **Target File & Section**: `docs/06_WORKFLOW_STATE_MACHINE.md`, Section 2 ("Canonical States & Transition Rules").
- **Current Content Causing Drift**:
  - `DISPATCHED` state description assumes automatic synchronous transition on spawn/send failure without detailing the fail-closed quarantine requirement.
  - `RUNNING` state transition table lists `RUNNING -> BLOCKED` upon worker reporting blocker, which misaligns with upstream AO runtime reality (where AO status `blocked` does NOT correspond to TaskState `BLOCKED`).
  - `FAILED -> READY` retry description lacks the mandatory double-gated quarantine safety check.
  - `BLOCKED -> HUMAN_REQUIRED` escalation does not specify atomic attempt closure.
- **Governing ADR-016 Decisions**: **D5** (Fail-Closed Terminal Failure with Quarantine), **D6** (Quarantine Gate on Retry), **D8** (`MISSED_ACTIVE_WINDOW`), **D9** (Preserve `RUNNING` on `waiting_input`), **D10** (Graph-Compatible Blocked Handling & Attempt Closure), **D11** (Purpose-Aware Stop Transitions), **D12** (`AtomicTerminalTransition` & `AtomicAttemptClosureTransition`).
- **Required Specification Synchronization**:
  1. *Preserve Canonical Graph Invariant*: Confirm that canonical state count (`13`) and transition edge count (`22 domain / 25 graph`) remain **strictly unchanged**. Zero new states or transitions are added.
  2. *`DISPATCHED` State Rule*: Add requirement that if dispatch delivery outcome is ambiguous or fails post-send, the Supervisor executes `DISPATCHED -> FAILED` with immediate double-gated quarantine (`quarantine_state = 'QUARANTINED'` on both session and attempt).
  3. *`RUNNING` State Rules*:
     - Observation of AO status `waiting_input` preserves `RUNNING` and continues polling (D9).
     - Observation of fast `active -> idle` window records `MISSED_ACTIVE_WINDOW` on attempt recovery disposition and verifies downstream handoff capability prior to `DISPATCHED -> RUNNING` (D8).
     - Observation of AO status `blocked` records `recovery_disposition = 'AO_BLOCKED_DECISION'` while keeping TaskState `RUNNING`. Formal escalation to developer executes canonical sequence `RUNNING -> BLOCKED -> HUMAN_REQUIRED` (D10).
  4. *`BLOCKED -> HUMAN_REQUIRED` Invariant*: When escalated to `HUMAN_REQUIRED`, the open `TaskAttempt` must be closed atomically (`ended_at = now`, `recovery_disposition = 'AO_BLOCKED_ESCALATED'`) under invariant `HUMAN_REQUIRED_WITH_PRIOR_EXECUTION MUST_NOT_RETAIN_RESUMABLE_OPEN_ATTEMPT` via `AtomicAttemptClosureTransition` (D10, D12).
  5. *`FAILED -> READY` Retry Gate*: Specify precondition that retry is strictly prohibited while `worker_sessions.quarantine_state == 'QUARANTINED'` or `task_attempts.quarantine_state == 'QUARANTINED'` (D6).
  6. *Stop Purpose Mapping*: Document that only stops initiated with `purpose = 'RUNNING_ATTEMPT_STOP'` on `RUNNING` tasks execute `RUNNING -> FAILED`. Stops initiated for `QUARANTINE_CLEANUP` or `PAIR_MAINTENANCE` execute zero TaskState transitions (D11).
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (quarantine states), `docs/14_FAILURE_RECOVERY.md` (recovery playbooks).
- **Execution Order**: Phase 1 (Core Domain & State Model Synchronization).
- **Consistency Checks**:
  - Assert canonical counts (`TASK_STATE_COUNT = 13`, `DOMAIN_TRANSITION_COUNT = 22`) are explicitly preserved.
  - Assert no `BLOCKED -> RUNNING` transition edge is introduced.
- **Completion Criteria**: Operational runtime observations (AO status values) are cleanly reconciled with canonical task transitions without altering the frozen state graph.

---

### Item S4: `docs/08_TASK_CONTRACT.md` — Section 1 (Invariants) & Section 3 (Handling Blockers)

- **Target File & Section**: `docs/08_TASK_CONTRACT.md`, Section 1 ("Purpose & Contractual Invariants") and Section 3 ("Handling Blockers and Scope Deviations").
- **Current Content Causing Drift**:
  - Section 1, Invariant 3 ("Separation of Specification from Execution") mentions attempt tracking but does not document execution identity snapshot binding.
  - Section 3 describes blocker escalation but does not document the mandatory attempt closure invariant upon escalation to `HUMAN_REQUIRED`.
- **Governing ADR-016 Decisions**: **D2** (Decouple Provisioning from Dispatch), **D4** (Attempt Execution Identity Snapshot Binding), **D10** (Attempt Closure upon Escalation).
- **Required Specification Synchronization**:
  1. *Execution Identity Binding*: In Section 1, Invariant 3, explicitly document that while `TaskContract` remains strictly specification-only and immutable, runtime execution binding (`attempt_execution_identity`) is captured durably in `TaskAttempt` snapshot fields and `dispatch_operations` upon dispatch.
  2. *Attempt Closure on Blocker Escalation*: In Section 3, specify that when a task transitions `BLOCKED -> HUMAN_REQUIRED`, the active `TaskAttempt` is closed atomically (`ended_at = now`, `recovery_disposition = 'AO_BLOCKED_ESCALATED'`), ensuring no resumable attempt remains dangling across human replanning or cancellation.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (TaskAttempt schema), `docs/06_WORKFLOW_STATE_MACHINE.md` (transition rules).
- **Execution Order**: Phase 1 (Core Domain & State Model Synchronization).
- **Consistency Checks**:
  - Assert `TaskContract` fields remain strictly identical (zero new fields on `TaskContract`).
- **Completion Criteria**: Execution identity separation and blocker attempt closure invariants are clearly articulated.

---

### Item S5: `docs/12_UPSTREAM_INTEGRATION.md` — Section 1 (AOAdapter Interface Contract) & Section 2 (Guards)

- **Target File & Section**: `docs/12_UPSTREAM_INTEGRATION.md`, Section 1 ("AOAdapter Interface Contract") and Section 2 ("Invariants and Architectural Guards").
- **Current Content Causing Drift**:
  - `IAOAdapter` interface lines 30-36 include `resumeWorker(sessionId: string): Promise<AOResult>` and `getWorktreePath(sessionId: string): Promise<string>`.
  - Interface lacks explicit representation of opaque string `terminal_generation` in status responses.
  - Section 1 lacks specification of the strict pre-send admissibility whitelist (`idle`, `waiting_input` only).
  - Absence of documentation regarding pinned AO `/kill` wire contract limitations (no atomic generation fence, `PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`).
- **Governing ADR-016 Decisions**: **D1** (Worktree Path Nullability), **D7** (Strict Pre-Send Admissibility Whitelist), **D11** (Purpose-Aware Stop Lifecycle & Wire Contract Boundary), **D13** (Startup Recovery Sweep Adapter Usage).
- **Required Specification Synchronization**:
  1. *Clean Interface Methods*:
     - Remove `resumeWorker` (non-existent in pinned AO).
     - Update `getWorktreePath`: note that worktree path is only available if returned in session creation/response, not as a distinct public endpoint.
     - Update `stopWorker`: document wire route `POST /api/v1/sessions/{id}/kill` and signature accepting stop parameters.
  2. *Authoritative Status Structure (`AOWorkerStatus`)*:
     - Document returned fields: status string (`active`, `idle`, `waiting_input`, `blocked`, `exited`), boolean `isTerminated`, and opaque string `terminal_generation` (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`).
  3. *Strict Pre-Send Admissibility Whitelist*:
     - Document that `dispatchTaskContract` (invoking `POST /api/v1/sessions/{id}/send`) is permitted ONLY when `AOWorkerStatus.status IN ('idle', 'waiting_input')`. Rejection is mandatory if status is `active` (foreign turn in flight) or terminal (`blocked`, `exited`, `isTerminated == true`).
  4. *Causality Boundary & Wire Contract Realism*:
     - Document that pinned AO exposes no exit timestamp, exit code, or OS signal. Supervisor stop confirmation relies on lineage-bounded temporal window (`now < confirmation_deadline_at`), not absolute causal attribution.
- **Co-requisite Synchronizations**: `docs/04_ARCHITECTURE.md` (§3.1), `docs/phases/P03_AO_INTEGRATION.md` (§2).
- **Execution Order**: Phase 2 (Architecture & Adapter Synchronization).
- **Consistency Checks**:
  - Assert `resumeWorker` is eliminated from `IAOAdapter`.
  - Assert pre-send whitelist contains strictly `idle` and `waiting_input`.
- **Completion Criteria**: `IAOAdapter` interface and upstream semantics match pinned AO v0.13.0 wire reality and ADR-016 requirements.

---

### Item S6: `docs/14_FAILURE_RECOVERY.md` — Section 1 (Recovery Matrix) & Section 1.2 (New: Double-Gated Quarantine & Startup Recovery Sweep)

- **Target File & Section**: `docs/14_FAILURE_RECOVERY.md`, Section 1 ("Recovery Matrix & Owners") and new subsections.
- **Current Content Causing Drift**:
  - REC-002 mentions "restore session if supported" (contradicts reality; restore does not exist).
  - REC-004 ("Worker Execution Timeout") prescribes issuing `stopWorker` and marking `FAILED (TIMEOUT)` without specifying purpose-aware stop operation tracking, generation precheck, or restart-stable deadlines.
  - REC-014 ("Unexpected Machine Restart") prescribes an informal scan without the deterministic decision order for in-flight provisioning, dispatch, stop operations, or mandatory blocked escalation.
  - Complete absence of Double-Gated Quarantine clearance playbooks (Class A, Class B, Class C).
- **Governing ADR-016 Decisions**: **D5** (Fail-Closed Terminal Failure), **D6** (Double-Gated Quarantine & Clearance Classes), **D11** (Purpose-Aware Stop Lifecycle), **D13** (Synchronous Startup Recovery Sweep).
- **Required Specification Synchronization**:
  1. *Update REC-002*: Remove "restore session if supported". Document session failure containment via quarantine.
  2. *Update REC-004*: Align timeout recovery with ADR-016 D11: allocate `stop_operations` record (`purpose = 'RUNNING_ATTEMPT_STOP'`), capture target generation, enforce restart-stable `confirmation_deadline_at`, observe termination within window.
  3. *Update REC-014 (Comprehensive Startup Recovery Sweep)*:
     - Replace informal scan with the deterministic 5-step synchronous recovery scanner from ADR-016 D13:
       1. Sweep `pair_provisioning_operations`: if `stage IN ('SPAWN_REQUESTED', 'SPAWN_CONFIRMED')`, lock Pair lane as crashed until resolved.
       2. Sweep `dispatch_operations`: if `stage IN ('SEND_REQUESTED', 'SEND_CONFIRMED')`, execute fail-closed `DISPATCHED -> FAILED` and quarantine both attempt and session.
       3. Sweep `stop_operations`: if `stage IN ('STOP_REQUESTED', 'STOP_CALL_SUCCEEDED')`, observe session status under strict decision order; enforce `STOP_REISSUE_REQUIRES_HUMAN` (zero blind re-kill).
       4. Evaluate restart-stable `confirmation_deadline_at`:
          - If termination first observed at/after deadline (`now >= confirmation_deadline_at`): fail closed to `STOP_CONFIRMATION_TIMEOUT`; **strictly prohibit `WORKER_STOPPED`**; retain quarantine.
          - If termination observed before deadline (`now < confirmation_deadline_at`): advance to `STOP_TERMINATION_CONFIRMED`, record `WORKER_STOPPED`, release quarantine.
       5. Mandatory blocked attempt escalation: `# BLOCKED_WITH_OPEN_CURRENT_ATTEMPT MANDATORY_ESCALATION_RECOVERY` finds any Task in `BLOCKED` with an open `TaskAttempt` and atomically escalates `BLOCKED -> HUMAN_REQUIRED` while closing the attempt (`AO_BLOCKED_ESCALATED`).
  4. *Add Section 1.2: Double-Gated Quarantine Clearance Rules*:
     - Document independent gate evaluation: `Pair Lane Gate` (`worker_sessions.quarantine_state`) and `Task Attempt Gate` (`task_attempts.quarantine_state`).
     - Detail clearance classes:
       - **Class A (Clearance)**: Verified clean worktree state and verified session termination.
       - **Class B (Isolation)**: Abandon corrupted session, persist permanent quarantine, provision fresh session.
       - **Class C (Operator Intervention)**: Explicit human inspection and clearance command.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (quarantine states), `docs/06_WORKFLOW_STATE_MACHINE.md` (retry gate).
- **Execution Order**: Phase 3 (Recovery Playbooks & Store Entity Provenance).
- **Consistency Checks**:
  - Assert "restore session" is removed from REC-002.
  - Assert the 5-step recovery sweep and the two observation timing cases are fully documented.
- **Completion Criteria**: Recovery playbooks accurately codify the crash-consistency, quarantine, and startup sweep requirements of ADR-016.

---

### Item S7: `docs/22_MODULE_PROVENANCE.md` — Section 1 (Planned Modules) & Section 3 (New: Store Entities Provenance)

- **Target File & Section**: `docs/22_MODULE_PROVENANCE.md`, Section 1 ("Planned Modules Provenance") and Section 3 ("Store Entities Provenance").
- **Current Content Causing Drift**:
  - Under `AOAdapter`, lists `/restore` among translated endpoints.
  - Lacks architectural registration and provenance justification for new store entities introduced by ADR-016.
- **Governing ADR-016 Decisions**: **D1**, **D3**, **D4**, **D11**, **D12**, ADR-016 §28 Item 7 ("Register new store entities").
- **Required Specification Synchronization**:
  1. *Clean `AOAdapter` Endpoint List*: Remove `/restore`. Add purpose-aware stop parameters and pre-send whitelist status checks.
  2. *Add Store Entities Provenance*:
     - `pair_provisioning_operations`: Tracks durable Pair session creation sagas; prevents orphan sessions across crashes (D3).
     - `dispatch_operations`: Tracks durable 3-stage dispatch sagas; enforces 1:1 attempt cardinality; eliminates ambiguous delivery crashes (D4).
     - `stop_operations`: Tracks purpose-aware stop operations with restart-stable deadlines and stage provenance; prevents blind re-kill (D11).
     - `task_attempts` extensions: Execution snapshot fields and dedicated `quarantine_state` attribute for double-gated quarantine (D1, D6).
     - Document anti-reinvention justification: these entities implement distributed saga consistency and crash recovery over external process runtimes, an area not provided by upstream AO.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (aggregate entities).
- **Execution Order**: Phase 3 (Recovery Playbooks & Store Entity Provenance).
- **Consistency Checks**:
  - Assert all new tables are registered with anti-reinvention justifications.
  - Assert `/restore` is removed from `AOAdapter` section.
- **Completion Criteria**: Module provenance fully documents all new persistent structures and maintains compliance with `docs/sources/REUSE_MATRIX.md`.

---

### Item S8: Cross-Check & Alignment of `docs/phases/P03_AO_INTEGRATION.md`

- **Target File & Section**: `docs/phases/P03_AO_INTEGRATION.md`, Section 2 ("Deliverables") and Section 3 ("Exit Gate").
- **Current Content Causing Drift**:
  - Section 2 Deliverables explicitly lists `POST /api/v1/sessions/{id}/restore`.
  - Lacks mention of decoupled Pair provisioning, durable dispatch saga, and double-gated quarantine integration.
- **Governing ADR-016 Decisions**: **D2**, **D4**, **D6**, **D11**.
- **Required Specification Synchronization**:
  1. Remove `POST /api/v1/sessions/{id}/restore` from Deliverables list.
  2. In Section 2, clarify that `AOAdapter` delivers decoupled session lifecycle primitives, strict pre-send status whitelist validation, and purpose-aware stop operations.
  3. In Section 3 Exit Gate, confirm that integration tests verify preflight probes, session creation, dispatch saga, observation reconciliation, and purpose-aware termination.
- **Co-requisite Synchronizations**: `docs/12_UPSTREAM_INTEGRATION.md`.
- **Execution Order**: Phase 2 (Architecture & Adapter Synchronization).
- **Consistency Checks**:
  - Assert `/restore` is absent from phase specification.
- **Completion Criteria**: Phase P03 deliverable scope accurately mirrors approved ADR-016 adapter boundaries.

---

### Item S9: Cross-Check & Alignment of `docs/21_TRACEABILITY_MATRIX.md`

- **Target File & Section**: `docs/21_TRACEABILITY_MATRIX.md`, Table rows `FR-005` (Worker Dispatch) and `OPS-002` (Clean Termination).
- **Current Content Causing Drift**:
  - `FR-005` references only `ADR-002, ADR-012` (omits `ADR-016`).
  - `OPS-002` references only `ADR-002` (omits `ADR-016`).
- **Governing ADR-016 Decisions**: **D2**, **D4**, **D11**.
- **Required Specification Synchronization**:
  1. Update `FR-005` row: add `ADR-016` to ADR Ref column; note durable 3-stage dispatch saga and decoupled session provisioning.
  2. Update `OPS-002` row: add `ADR-016` to ADR Ref column; note purpose-aware stop lifecycle and quarantine cleanup.
- **Co-requisite Synchronizations**: `docs/04_ARCHITECTURE.md`, `docs/14_FAILURE_RECOVERY.md`.
- **Execution Order**: Phase 3 (Recovery Playbooks & Store Entity Provenance).
- **Consistency Checks**:
  - Assert `ADR-016` appears in `FR-005` and `OPS-002` rows.
- **Completion Criteria**: Traceability matrix correctly traces requirements to accepted ADR-016.

---

### Item S10: Cross-Check of `docs/02_REQUIREMENTS.md`

- **Target File & Section**: `docs/02_REQUIREMENTS.md`, `FR-005` (Worker Task Dispatch) and `FR-006` (Worker Lifecycle Observation).
- **Assessment of Current Content**:
  - `FR-005`: States "The system shall dispatch the immutable Task Contract to the Execution Control Plane (AO) to initiate or resume worker execution. Task state transitions from READY to DISPATCHED; AO worker session is triggered."
  - `FR-006`: Already updated in Baseline A Revision 3 reconciliation to state: "intentional termination and unexpected termination are distinguished using Supervisor operation provenance where possible; task/operation timeout handling uses separately configured bounded execution deadlines; no synthetic worker heartbeat is required; lastActivityAt is diagnostic activity evidence, not heartbeat proof."
- **Reconciliation Verdict**: **COMPATIBLE (NO SPECIFICATION MUTATION REQUIRED)**.
  - The requirements in `docs/02` are high-level and already accommodate decoupled provisioning and provenance tracking. Mutating `docs/02` is unnecessary and introduces risk of specification churn.

---

### Item S11: Cross-Check of `docs/17_ROADMAP.md`

- **Target File & Section**: `docs/17_ROADMAP.md`, Table row `P03` (AO Integration).
- **Assessment of Current Content**:
  - Deliverable: "AOAdapter public REST integration, session lifecycle/dispatch, lifecycle observation/state reconciliation integration, raw session workspace-file read transport (zero P04 WorkerReport semantic ingestion)."
  - Exit Gate: "Automated session creation, dispatch, observation reconciliation, raw workspace file read, and teardown pass without P04 EvidenceCollector dependencies."
- **Reconciliation Verdict**: **COMPATIBLE (NO SPECIFICATION MUTATION REQUIRED)**.
  - The roadmap description is fully compatible with ADR-016 deliverables. No text modification is warranted.

---

## 4. Operational Policies Invariant Guardrail

Pursuant to Section 2 of `AGENTS.md` and ADR-016 §33:
- The eight operational policies remain strictly **UNSET (`VALUE = UNSET`)**:
  1. `SUPERVISOR_HTTP_TIMEOUT`
  2. `SUPERVISOR_HEALTH_PROBE_TIMEOUT`
  3. `SUPERVISOR_SPAWN_TIMEOUT`
  4. `SUPERVISOR_SEND_TIMEOUT`
  5. `SUPERVISOR_ACTIVITY_POLL_INTERVAL`
  6. `SUPERVISOR_EXECUTION_DEADLINE`
  7. `SUPERVISOR_KILL_STOP_TIMEOUT`
  8. `SUPERVISOR_WORKSPACE_READ_TIMEOUT`
- **Reconciliation Invariant**: The canonical specification reconciliation process **MUST NOT** invent, supply, or inject numeric defaults for any of these policies. Any timeout references in canonical specifications must refer to them as configured, bounded operational policies or upstream server facts (`UPSTREAM_AO_REQUEST_TIMEOUT = 60s`).

---

## 5. Phased Execution Order & Consistency Strategy

To ensure zero intermediate contradictions, the canonical specification reconciliation will be executed in three strictly ordered batches:

```mermaid
graph TD
    subgraph Batch 1: Core Domain & State Models
        B1_1["docs/05_DOMAIN_MODEL.md (Item S2)"]
        B1_2["docs/06_WORKFLOW_STATE_MACHINE.md (Item S3)"]
        B1_3["docs/08_TASK_CONTRACT.md (Item S4)"]
    end

    subgraph Batch 2: Architecture & Adapter Contracts
        B2_1["docs/04_ARCHITECTURE.md (Item S1)"]
        B2_2["docs/12_UPSTREAM_INTEGRATION.md (Item S5)"]
        B2_3["docs/phases/P03_AO_INTEGRATION.md (Item S8)"]
    end

    subgraph Batch 3: Recovery Playbooks & Store Entity Provenance
        B3_1["docs/14_FAILURE_RECOVERY.md (Item S6)"]
        B3_2["docs/22_MODULE_PROVENANCE.md (Item S7)"]
        B3_3["docs/21_TRACEABILITY_MATRIX.md (Item S9)"]
    end

    Batch 1 --> Batch 2
    Batch 2 --> Batch 3
```

### Verification Criteria for Each Batch:
1. **Batch 1**: Domain entities (`pair_provisioning_operations`, `dispatch_operations`, `stop_operations`) and state rules compile with zero syntax errors, preserve the 13 canonical states and 22 executable transitions, and enforce Model A snapshot fields.
2. **Batch 2**: Architecture sequence diagram aligns with decoupled provisioning and 3-stage dispatch saga; non-existent `/restore` endpoint is completely removed across architecture, adapter, and phase specs.
3. **Batch 3**: Recovery playbooks REC-002, REC-004, REC-014 accurately model the 5-step startup recovery sweep and double-gated quarantine clearance rules without inventing numeric timeouts.

---

## 6. Completion & Handoff Criteria

Reconciliation execution will be considered complete and ready for External Supervisor Audit when:
1. All 9 target items (S1..S9) are implemented strictly within their specified sections.
2. Items S10 and S11 are verified as compatible without mutation.
3. All 13 decisions D1..D13 are 100% traceable to updated canonical sections.
4. `git diff --check` passes with zero errors.
5. Zero production code (`internal/**/*.go`), zero schema migrations, and zero Task Contracts have been created.
6. The governance status transitions to `P03_CANONICAL_RECONCILIATION = READY_FOR_EXTERNAL_AUDIT`.
