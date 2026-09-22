# PLAN-P03-CANONICAL-RECONCILIATION-ADR-016: Canonical Specification Reconciliation Scope Plan

> **Authority**: Formulated pursuant to accepted [ADR-016](file:///d:/TU_CODE/ai-supervisor/docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md) (EXTERNAL_APPROVED by External Supervisor Re-Audit 006, commit `40d51ffe9af4dab5545f060d5bee2ffd441b6109`), `docs/24_CHANGE_GOVERNANCE.md` (Decision Hierarchy Level 2: Approved ADR takes precedence over Level 3: Canonical Architecture and Level 4: Requirement Specifications), and External Supervisor directive.
> **Status**: DRAFT PLAN — PENDING USER SCOPE REVIEW
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
  - Section 3.1 includes `POST /api/v1/sessions/{id}/restore` as an active process control endpoint. ADR-016 §4 names the pinned public routes, while §§7, 9, and 13 also refer to session restore / `ResumeWorker`; the plan must not silently resolve this ambiguity.
- **Governing ADR-016 Decisions**: **D2** (ADR-016 §8), **D4** (ADR-016 §10), and **D11** (ADR-016 §17); public wire facts are in ADR-016 §4 and token definitions in §6. The `/restore` capability ambiguity is tracked separately in §7 of this plan.
- **Required Specification Synchronization**:
  1. *Sequence Diagram Update*: Update Section 2.1 Mermaid diagram to decouple Pair session provisioning from task prompt dispatch. The task dispatch flow commences with a pre-existing, verified Pair WorkerSession in `idle` or `waiting_input` status (or triggers Pair provisioning if no session exists under `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`).
  2. *Durable Dispatch Saga Steps*: Reflect the 3-stage dispatch saga in `dispatch_operations`:
     - Stage 1: `DISPATCH_BOUND` (persisted simultaneously with TaskState `READY -> DISPATCHED` and candidate `TaskAttempt` execution snapshot);
     - Stage 2: `SEND_REQUESTED` (persisted immediately prior to issuing loopback `POST /api/v1/sessions/{id}/send`);
     - Stage 3: `SEND_CONFIRMED` (persisted only upon HTTP 200 response from AO; HTTP 204 is not an ADR-approved success condition).
  3. *Reconcile the `/restore` wire claim*: Do not document `POST /api/v1/sessions/{id}/restore` as a pinned AO wire route unless the exact pinned public source proves it exists. Before canonical mutation, resolve the separate ADR ambiguity recorded in §7 below about the approved `ResumeWorker` / session-restore capability.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (entities `PairProvisioningOperation`, `DispatchOperation`), `docs/12_UPSTREAM_INTEGRATION.md` (wire methods).
- **Execution Order**: Phase 2 (Architecture & Adapter Synchronization).
- **Consistency Checks**:
  - Assert sequence diagram shows no inline worktree creation during normal dispatch over an existing session.
  - Assert no unverified `/restore` wire route is documented; verify the route against the pinned public API and apply the External Supervisor's resolution for the ADR restore-capability blocker in §7.
- **Completion Criteria**: Sequence diagram accurately models decoupled provisioning and durable 3-stage dispatch saga adhering to D2, D4, and D11.

---

### Item S2: `docs/05_DOMAIN_MODEL.md` — Section 1 (Domain Entities Diagram) & Section 2 (Entity Dictionary)

- **Target File & Section**: `docs/05_DOMAIN_MODEL.md`, Section 1 (Class Diagram) and Section 2 (Entity Dictionary & Invariants).
- **Current Content Causing Drift**:
  - `WorkerSession` entity in class diagram lacks `quarantine_state` attribute.
  - `TaskAttempt` entity lacks ADR-defined snapshot additions (`session_id`, `terminal_generation`, `recovery_disposition`, `quarantine_state`); it must not gain WorkerSession columns such as `runtime_type`, `worker_agent_id`, or `worktree_path`.
  - Class diagram and dictionary lack entities for durable saga tracking: `PairProvisioningOperation`, `DispatchOperation`, and `StopOperation`.
  - Invariant 1 in Section 2 implies `worktree_path` is always known and non-null, whereas pinned AO v0.13.0 public API does not return worktree path (`worktree_path` is `TEXT NULL`).
- **Governing ADR-016 Decisions**: **D1** (ADR-016 §7), **D3** (§9), **D4** (§10), **D6** (§12), **D11** (§17), and **D12** (§18); authoritative tokens are in §6 and the sole candidate DDL is in §27.
- **Required Specification Synchronization**:
  1. *Update `WorkerSession`*: Add `quarantine_state` with exactly the ADR values `CLEAN` / `QUARANTINED`; keep it separate from runtime `status` (`ACTIVE` / `IDLE` / `TERMINATED`). Note `worktree_path: string | null`.
  2. *Update `TaskAttempt`*: Formalize Model A persistence relationship while preserving domain relationship `Pair "1" o-- "1" WorkerSession`. Add only the exact ADR candidate extensions: `session_id TEXT`, `terminal_generation TEXT`, `recovery_disposition TEXT`, and `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))`. Do not add `worker_session_id`, `runtime_type`, `worker_agent_id`, or `worktree_path` to `task_attempts`; those are not D1's declared attempt snapshot columns.
  3. *Add New Aggregate Entities*:
     - `PairProvisioningOperation`: Mirror the sole `pair_provisioning_operations` DDL in ADR-016 §27, including its exact columns, `stage` CHECK values (`PROVISION_REQUESTED`, `PROVISION_CONFIRMED`, `PROVISION_FAILED`, `PROVISION_RESOLVED`), and partial unique index predicate (`PROVISION_REQUESTED`, `PROVISION_FAILED`). Do not substitute `SPAWN_*` tokens or add undeclared columns.
     - `DispatchOperation`: Mirror the sole `dispatch_operations` DDL in ADR-016 §27, including exact columns, `stage` CHECK values (`DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`), and `UNIQUE(attempt_id)`. Keep `stage` distinct from `resolution_state`.
     - `StopOperation`: Mirror the sole `stop_operations` DDL in ADR-016 §27, including exact columns, `purpose` CHECK values, and `stage` CHECK values (`STOP_REQUESTED`, `STOP_CALL_SUCCEEDED`, `STOP_CALL_FAILED`, `STOP_TERMINATION_CONFIRMED`, `STOP_TARGET_ABSENT`). Keep `resolution_state` separate; `STOP_CONFIRMATION_TIMEOUT`, `STOP_CALL_OUTCOME_UNKNOWN`, and other resolution tokens are not `stage` values. Do not add `STOP_TIMED_OUT` or substitute `stop_id` for `operation_id`.
  4. *Invariants & Vocabulary*: Document invariant `# ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION`, `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`, and zero undeclared columns on `task_attempts` (rejecting invented columns like `terminal_error`). Use `CLEAN` / `QUARANTINED` for both quarantine columns and do not conflate quarantine state with diagnostic `recovery_disposition`.
- **Co-requisite Synchronizations**: `docs/08_TASK_CONTRACT.md` (execution identity separation), `docs/22_MODULE_PROVENANCE.md` (StateStore entity registry).
- **Execution Order**: Phase 1 (Core Domain & State Model Synchronization).
- **Consistency Checks**:
  - Verify all 3 new entities appear in class diagram and entity dictionary.
  - Verify `task_attempts` contains only the four ADR-declared candidate additions and exact CHECK/default; compare all three operation diagrams/dictionaries against §6 token groups and §27 DDL, including the distinction between `stage` and `resolution_state`.
- **Completion Criteria**: Model A persistence, double-gated quarantine, and durable operations are formally defined with exact column and constraint invariants.

---

### Item S3: `docs/06_WORKFLOW_STATE_MACHINE.md` — Section 2 (Canonical States & Transition Rules)

- **Target File & Section**: `docs/06_WORKFLOW_STATE_MACHINE.md`, Section 2 ("Canonical States & Transition Rules").
- **Current Content Causing Drift**:
  - `DISPATCHED` state description assumes automatic synchronous transition on spawn/send failure without detailing the fail-closed quarantine requirement.
  - `RUNNING` state transition table lists `RUNNING -> BLOCKED` upon worker reporting blocker, which misaligns with upstream AO runtime reality (where AO status `blocked` does NOT correspond to TaskState `BLOCKED`).
  - `FAILED -> READY` retry description lacks the mandatory double-gated quarantine safety check.
  - `BLOCKED -> HUMAN_REQUIRED` escalation does not specify atomic attempt closure.
- **Governing ADR-016 Decisions**: **D5** (ADR-016 §11), **D6** (§12), **D8** (§14), **D9** (§15), **D10** (§16), **D11** (§17), **D12** (§18), and the lifecycle/crash matrices in §§22–24; token and TaskState invariants are in §§6 and 5.
- **Required Specification Synchronization**:
  1. *Preserve Canonical Graph Invariant*: Confirm that canonical state count (`13`) and transition edge count (`22 domain / 25 graph`) remain **strictly unchanged**. Zero new states or transitions are added.
  2. *`DISPATCHED` State Rule*: For an unconfirmed `SEND_REQUESTED` / unknown delivery outcome, execute `DISPATCHED -> FAILED`, set `ended_at = now`, `recovery_disposition = 'UNCERTAIN_DELIVERY_CRASH'`, and immediate double-gated quarantine (`quarantine_state = 'QUARANTINED'` on both session and attempt), then follow the ADR's `FAILED -> HUMAN_REQUIRED` escalation. Prohibit blind resend. Do not classify a `SEND_CONFIRMED` write as unknown delivery.
  3. *`RUNNING` State Rules*:
     - Observation of AO status `waiting_input` preserves `RUNNING` and continues polling (D9).
     - Observation of fast `active -> idle` window records `MISSED_ACTIVE_WINDOW` on attempt recovery disposition and verifies downstream handoff capability prior to `DISPATCHED -> RUNNING` (D8).
     - Observation of AO status `blocked` records `recovery_disposition = 'AO_BLOCKED_DECISION'` while keeping TaskState `RUNNING`. Formal escalation to developer executes canonical sequence `RUNNING -> BLOCKED -> HUMAN_REQUIRED` (D10).
  4. *`BLOCKED -> HUMAN_REQUIRED` Invariant*: When escalated to `HUMAN_REQUIRED`, the open `TaskAttempt` must be closed atomically (`ended_at = now`, `recovery_disposition = 'AO_BLOCKED_ESCALATED'`) under invariant `HUMAN_REQUIRED_WITH_PRIOR_EXECUTION MUST_NOT_RETAIN_RESUMABLE_OPEN_ATTEMPT` via `AtomicAttemptClosureTransition` (D10, D12).
  5. *`FAILED -> READY` Retry Gate*: Specify that retry is prohibited unless `worker_sessions.quarantine_state == 'CLEAN'`, all prior attempts for the task have `quarantine_state == 'CLEAN'`, and the Pair has zero provisioning rows with stage `PROVISION_REQUESTED` or `PROVISION_FAILED` (D6, §12).
  6. *Stop Purpose Mapping*: Document that only stops initiated with `purpose = 'RUNNING_ATTEMPT_STOP'` on `RUNNING` tasks execute `RUNNING -> FAILED`. Stops initiated for `QUARANTINE_CLEANUP` or `PAIR_MAINTENANCE` execute zero TaskState transitions (D11).
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (quarantine states), `docs/14_FAILURE_RECOVERY.md` (recovery playbooks).
- **Execution Order**: Phase 1 (Core Domain & State Model Synchronization).
- **Consistency Checks**:
  - Assert canonical counts (`TASK_STATE_COUNT = 13`, `DOMAIN_TRANSITION_COUNT = 22`, `GRAPH_EDGE_COUNT = 25`) and D5/D6 quarantine gates match ADR-016 §§5, 11–12 and registry §6.
  - Assert no `BLOCKED -> RUNNING` transition edge is introduced.
- **Completion Criteria**: Operational runtime observations (AO status values) are cleanly reconciled with canonical task transitions without altering the frozen state graph.

---

### Item S4: `docs/08_TASK_CONTRACT.md` — Section 1 (Invariants) & Section 3 (Handling Blockers)

- **Target File & Section**: `docs/08_TASK_CONTRACT.md`, Section 1 ("Purpose & Contractual Invariants") and Section 3 ("Handling Blockers and Scope Deviations").
- **Current Content Causing Drift**:
  - Section 1, Invariant 3 ("Separation of Specification from Execution") mentions attempt tracking but does not document execution identity snapshot binding.
  - Section 3 describes blocker escalation but does not document the mandatory attempt closure invariant upon escalation to `HUMAN_REQUIRED`.
- **Governing ADR-016 Decisions**: **D1** (ADR-016 §7), **D2** (§8), **D4** (§10), **D10** (§16), and **D12** (§18); candidate columns are constrained by §27 and the TaskContract immutability rule by §5.
- **Required Specification Synchronization**:
  1. *Execution Identity Binding*: In Section 1, Invariant 3, document that while `TaskContract` remains specification-only and immutable, execution identity is captured in the exact `task_attempts` additions `session_id` and `terminal_generation` and in the §27 `dispatch_operations` row. Treat `attempt_execution_identity` only as explanatory terminology, not a new field, column, token, or contract property.
  2. *Attempt Closure on Blocker Escalation*: In Section 3, specify that when a task transitions `BLOCKED -> HUMAN_REQUIRED`, the active `TaskAttempt` is closed atomically (`ended_at = now`, `recovery_disposition = 'AO_BLOCKED_ESCALATED'`), ensuring no resumable attempt remains dangling across human replanning or cancellation.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (TaskAttempt schema), `docs/06_WORKFLOW_STATE_MACHINE.md` (transition rules).
- **Execution Order**: Phase 1 (Core Domain & State Model Synchronization).
- **Consistency Checks**:
  - Assert `TaskContract` fields remain strictly identical (zero new fields); compare attempt/dispatch fields and escalation disposition to ADR-016 §§6, 7, 10, 16, 18, and 27.
- **Completion Criteria**: Execution identity separation and blocker attempt closure invariants are clearly articulated.

---

### Item S5: `docs/12_UPSTREAM_INTEGRATION.md` — Section 1 (AOAdapter Interface Contract) & Section 2 (Guards)

- **Target File & Section**: `docs/12_UPSTREAM_INTEGRATION.md`, Section 1 ("AOAdapter Interface Contract") and Section 2 ("Invariants and Architectural Guards").
- **Current Content Causing Drift**:
  - `IAOAdapter` interface lines 30-36 include `resumeWorker(sessionId: string): Promise<AOResult>` and `getWorktreePath(sessionId: string): Promise<string>`.
  - Interface lacks explicit representation of opaque string `terminal_generation` in status responses.
  - Section 1 lacks specification of the strict pre-send admissibility whitelist (`idle`, `waiting_input` only).
  - Absence of documentation regarding pinned AO `/kill` wire contract limitations (no atomic generation fence, `PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`).
- **Governing ADR-016 Decisions**: **D1** (ADR-016 §7), **D7** (§13), **D11** (§17), and **D13** (§19); public field/route facts are in §4, registry tokens in §6, and candidate persistence definitions in §27.
- **Required Specification Synchronization**:
  1. *Clean Interface Methods*:
     - Do not assert that the ADR-approved `ResumeWorker` capability is absent: ADR-016 §§7 and 9 require a governed resume path for a restorable session, but §4 does not identify its public wire mapping. Resolve the explicit blocker in §7 before changing the canonical adapter contract; do not invent a `/restore` route.
     - Update worktree-path semantics: pinned V1 public responses do not expose it; model `worktree_path` as nullable and do not add a public lookup endpoint or synthetic value.
     - Update `stopWorker`: document only the pinned wire route `POST /api/v1/sessions/{sessionId}/kill`, whose request identifies the session only. Keep stop `purpose`, generation precheck, and `confirmation_deadline_at` in Supervisor logic/operation metadata; they are not wire parameters and do not form an atomic generation fence.
  2. *Authoritative Status Structure (`AOWorkerStatus`)*:
     - Document only the public fields/semantics established by ADR-016 §4 and §13: status (`active`, `idle`, `waiting_input`, `blocked`, `exited`), `isTerminated`, and opaque string `terminal_generation`; state that pinned public AO exposes no process exit timestamp, exit code, OS signal, or crash-causality evidence. A generation observation is not a kill generation fence.
  3. *Strict Pre-Send Admissibility Whitelist*:
     - Document that `dispatchTaskContract` (invoking `POST /api/v1/sessions/{id}/send`) is permitted ONLY when `AOWorkerStatus.status IN ('idle', 'waiting_input')`. Rejection is mandatory if status is `active` (foreign turn in flight) or terminal (`blocked`, `exited`, `isTerminated == true`).
  4. *Causality Boundary & Wire Contract Realism*:
     - Document that pinned AO's `/kill` wire request contains only `sessionId`: `POST /api/v1/sessions/{sessionId}/kill`. `purpose`, generation precheck, and `confirmation_deadline_at` belong to Supervisor logic/`stop_operations` metadata; they are not wire parameters and do not create an atomic generation fence. Preserve the session-scoped TOCTOU limitation. Stop confirmation follows ADR-016 §17's positive call acceptance and deadline rules; do not claim absolute causal attribution.
- **Co-requisite Synchronizations**: `docs/04_ARCHITECTURE.md` (§3.1), `docs/phases/P03_AO_INTEGRATION.md` (§2).
- **Execution Order**: Phase 2 (Architecture & Adapter Synchronization).
- **Consistency Checks**:
  - Assert any `ResumeWorker` adapter operation is tied to the resolution of the §7 blocker and verified pinned public capability; never infer a `/restore` wire route from the adapter name.
  - Assert the `/kill` wire contract has only session identity and no purpose/generation/deadline parameter.
  - Assert pre-send whitelist contains strictly `idle` and `waiting_input`.
- **Completion Criteria**: `IAOAdapter` interface and upstream semantics match pinned AO v0.13.0 wire reality and ADR-016 requirements.

---

### Item S6: `docs/14_FAILURE_RECOVERY.md` — Section 1 (Recovery Matrix) & Section 1.2 (New: Double-Gated Quarantine & Startup Recovery Sweep)

- **Target File & Section**: `docs/14_FAILURE_RECOVERY.md`, Section 1 ("Recovery Matrix & Owners") and new subsections.
- **Current Content Causing Drift**:
  - REC-002 mentions "restore session if supported"; reconcile the unsupported wire-route claim without overriding ADR-016 §§7, 9, and 13's restore / `ResumeWorker` references.
  - REC-004 ("Worker Execution Timeout") prescribes issuing `stopWorker` and marking `FAILED (TIMEOUT)` without specifying purpose-aware stop operation tracking, generation precheck, or restart-stable deadlines.
  - REC-014 ("Unexpected Machine Restart") prescribes an informal scan without the deterministic decision order for in-flight provisioning, dispatch, stop operations, or mandatory blocked escalation.
  - Complete absence of Double-Gated Quarantine clearance playbooks (Class A, Class B, Class C).
- **Governing ADR-016 Decisions**: **D5** (ADR-016 §11), **D6** (§12), **D11** (§17), and **D13** (§19); clearance vocabulary is in §6 items 5–8 and persistence CHECK values in §27.
- **Required Specification Synchronization**:
  1. *Update REC-002*: Remove any unverified `/restore` wire-route instruction. Document session failure containment via quarantine, and apply the External Supervisor's resolution of the §7 restore-capability blocker before changing the `ResumeWorker` behavior statement.
  2. *Update REC-004*: Align timeout recovery with ADR-016 §17: allocate a `stop_operations` row (`purpose = 'RUNNING_ATTEMPT_STOP'`), capture target generation for a Supervisor-side precheck, persist restart-stable `confirmation_deadline_at`, and observe termination within the window. Do not describe generation as a `/kill` argument or atomic fence.
  3. *Update REC-014 (Comprehensive Startup Recovery Sweep)*:
     - Replace informal scan with the synchronous recovery scanner from ADR-016 §19 (D13), preserving each exact stage predicate and outcome:
       1. Sweep only `pair_provisioning_operations.stage = 'PROVISION_REQUESTED'`; transition it to `PROVISION_FAILED` and lock the affected Pair lane. Do not scan `PROVISION_CONFIRMED` as an unresolved startup provisioning stage.
       2. Sweep `dispatch_operations.stage = 'SEND_REQUESTED'` as unknown delivery and apply D5's fail-closed `DISPATCHED -> FAILED` plus quarantine on attempt and session. `SEND_CONFIRMED` is an accepted write, not unknown delivery; reconcile it against approved AO status evidence instead of applying the unknown-delivery rule.
       3. Sweep in-flight `stop_operations` (`resolution_state = 'IN_FLIGHT'`) whose stage is `STOP_REQUESTED` or `STOP_CALL_SUCCEEDED`; preserve the exact §17 stage CHECK values and keep resolution tokens (including `STOP_CALL_OUTCOME_UNKNOWN` and `STOP_CONFIRMATION_TIMEOUT`) out of `stage`. Observe status under the §17 decision order and enforce `STOP_REISSUE_REQUIRES_HUMAN` (zero blind re-kill).
       4. Evaluate restart-stable `confirmation_deadline_at`:
          - For a prior `STOP_CALL_SUCCEEDED` with matching generation and `isTerminated == true`, if first observed before deadline (`now < confirmation_deadline_at`), advance to `STOP_TERMINATION_CONFIRMED` / `TERMINATION_CONFIRMED`; `WORKER_STOPPED` and physical quarantine clearance are allowed only when all ADR-016 §17 `WORKER_STOPPED_ALLOWED_IFF` conditions hold and the stop purpose permits it.
          - For that same prior `STOP_CALL_SUCCEEDED` case first observed at/after deadline (`now >= confirmation_deadline_at`), retain stage `STOP_CALL_SUCCEEDED`, set resolution `STOP_CONFIRMATION_TIMEOUT`, prohibit `WORKER_STOPPED`, and retain quarantine. Other stop stages follow their specific §17 truth-table outcomes; do not apply this deadline branch to every stop.
       5. Mandatory blocked attempt escalation: `# BLOCKED_WITH_OPEN_CURRENT_ATTEMPT MANDATORY_ESCALATION_RECOVERY` finds any Task in `BLOCKED` with an open `TaskAttempt` and atomically escalates `BLOCKED -> HUMAN_REQUIRED` while closing the attempt (`AO_BLOCKED_ESCALATED`).
  4. *Add Section 1.2: Double-Gated Quarantine Clearance Rules*:
     - Document independent gate evaluation: `Pair Lane Gate` (`worker_sessions.quarantine_state`) and `Task Attempt Gate` (`task_attempts.quarantine_state`).
     - Detail clearance classes:
       - **Class A (`PHYSICAL_EXECUTION_RESOLUTION`)**: Verify matching-generation physical termination using the approved D11 stop evidence and applicable purpose/deadline rules; clear only under the ADR's physical-resolution rules. Do not introduce clean-worktree inspection as an extra class condition.
       - **Class B (`ADMINISTRATIVE_RISK_RESOLUTION`, HTTP 404 absence)**: 404 proves public session absence, not physical termination. Keep both quarantine gates active until an authorized operator performs the ADR-defined administrative risk resolution, recording absence evidence, risk acknowledgement, and `QUARANTINE_RESOLVED_ADMINISTRATIVE`. Session absence or replacement alone never clears quarantine.
       - **Class C (`ADMINISTRATIVE_RISK_RESOLUTION` / Human Risk Acceptance)**: When execution cannot be proven stopped or contacted, an authorized human explicitly accepts the residual duplicate-execution risk; persist `stop_operations.resolution_state = 'ADMINISTRATIVE_RISK_ACCEPTED'` and emit `QUARANTINE_RESOLVED_ADMINISTRATIVE`. This is not physical termination and must not be recorded as `WORKER_STOPPED` / `TERMINATION_CONFIRMED`.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (quarantine states), `docs/06_WORKFLOW_STATE_MACHINE.md` (retry gate).
- **Execution Order**: Phase 3 (Recovery Playbooks & Store Entity Provenance).
- **Consistency Checks**:
  - Assert REC-002 contains no unverified `/restore` wire route and its `ResumeWorker` behavior matches the §7 blocker resolution.
  - Assert only `PROVISION_REQUESTED` is startup-marked `PROVISION_FAILED`; `SEND_REQUESTED` is the unknown-delivery path; `SEND_CONFIRMED` is reconciled as an accepted write; stop `stage` and `resolution_state` remain distinct; and Class A/B/C clearances follow §§12 and 17 with audited clearance.
- **Completion Criteria**: Recovery playbooks accurately codify the crash-consistency, quarantine, and startup sweep requirements of ADR-016.

---

### Item S7: `docs/22_MODULE_PROVENANCE.md` — Section 1 (Planned Modules) & Section 3 (New: Store Entities Provenance)

- **Target File & Section**: `docs/22_MODULE_PROVENANCE.md`, Section 1 ("Planned Modules Provenance") and Section 3 ("Store Entities Provenance").
- **Current Content Causing Drift**:
  - Under `AOAdapter`, lists `/restore` among translated endpoints.
  - Lacks architectural registration and provenance justification for new store entities introduced by ADR-016.
- **Governing ADR-016 Decisions**: **D1** (ADR-016 §7), **D3** (§9), **D4** (§10), **D11** (§17), **D12** (§18), persistence DDL (§27), and §28 Item 7 ("Register new store entities").
- **Required Specification Synchronization**:
  1. *Clean `AOAdapter` Endpoint List*: Do not list an unverified `/restore` wire endpoint; resolve the §7 restore-capability blocker before changing any adapter capability claim. Document purpose, generation precheck, and deadline as Supervisor-owned stop-operation metadata/logic. The pinned AO `/kill` wire call remains `POST /api/v1/sessions/{sessionId}/kill` with session identity only; do not add purpose or generation parameters or describe the precheck as an atomic fence. Include the D7 pre-send whitelist status check.
  2. *Add Store Entities Provenance*:
     - `pair_provisioning_operations`: Records durable Pair provisioning intent and unresolved spawn uncertainty for operator-audited handling; unowned orphan sessions are not automatically identified or killed (D3).
     - `dispatch_operations`: Records durable 3-stage dispatch sagas and enforces 1:1 attempt cardinality; unknown delivery remains possible and is handled fail-closed (D4).
     - `stop_operations`: Tracks purpose-aware stop operations with restart-stable deadlines and stage provenance; prevents blind re-kill (D11).
     - `task_attempts` extensions: exactly `session_id`, `terminal_generation`, `recovery_disposition`, and `quarantine_state` with the §27 CHECK/default; do not add other columns (D1, D6, D12).
     - Document anti-reinvention justification: these entities implement distributed saga consistency and crash recovery over external process runtimes, an area not provided by upstream AO.
- **Co-requisite Synchronizations**: `docs/05_DOMAIN_MODEL.md` (aggregate entities).
- **Execution Order**: Phase 3 (Recovery Playbooks & Store Entity Provenance).
- **Consistency Checks**:
  - Assert all new tables are registered with anti-reinvention justifications.
  - Assert no unverified `/restore` wire route is listed and the adapter capability statement follows the External Supervisor's §7 blocker resolution.
- **Completion Criteria**: Module provenance fully documents all new persistent structures and maintains compliance with `docs/sources/REUSE_MATRIX.md`.

---

### Item S8: Cross-Check & Alignment of `docs/phases/P03_AO_INTEGRATION.md`

- **Target File & Section**: `docs/phases/P03_AO_INTEGRATION.md`, Section 2 ("Deliverables") and Section 3 ("Exit Gate").
- **Current Content Causing Drift**:
  - Section 2 Deliverables explicitly lists `POST /api/v1/sessions/{id}/restore`.
  - Lacks mention of decoupled Pair provisioning, durable dispatch saga, and double-gated quarantine integration.
- **Governing ADR-016 Decisions**: **D2** (ADR-016 §8), **D4** (§10), **D6** (§12), **D11** (§17), and **D13** (§19); token, wire, and DDL checks are in §§4, 6, and 27.
- **Required Specification Synchronization**:
  1. Do not list `POST /api/v1/sessions/{id}/restore` as an AO wire route unless the pinned public source verifies it; resolve the §7 restore-capability blocker before changing any adapter capability statement.
  2. In Section 2, clarify that `AOAdapter` delivers decoupled session lifecycle primitives, strict pre-send status whitelist validation, and purpose-aware stop operations.
  3. In Section 3 Exit Gate, confirm that integration tests verify preflight probes, session creation, dispatch saga, observation reconciliation, and purpose-aware termination.
- **Co-requisite Synchronizations**: `docs/12_UPSTREAM_INTEGRATION.md`.
- **Execution Order**: Phase 2 (Architecture & Adapter Synchronization).
- **Consistency Checks**:
  - Assert no unverified `/restore` wire route appears in the phase specification; apply the External Supervisor's resolution of the §7 blocker.
- **Completion Criteria**: Phase P03 deliverable scope accurately mirrors approved ADR-016 adapter boundaries.

---

### Item S9: Cross-Check & Alignment of `docs/21_TRACEABILITY_MATRIX.md`

- **Target File & Section**: `docs/21_TRACEABILITY_MATRIX.md`, Table rows `FR-005` (Worker Dispatch) and `OPS-002` (Clean Termination).
- **Current Content Causing Drift**:
  - `FR-005` references only `ADR-002, ADR-012` (omits `ADR-016`).
  - `OPS-002` references only `ADR-002` (omits `ADR-016`).
- **Governing ADR-016 Decisions**: **D2** (ADR-016 §8), **D4** (§10), **D11** (§17), and **D13** (§19); trace tokens against §6 and operation definitions against §27.
- **Required Specification Synchronization**:
  1. Update `FR-005` row: add `ADR-016` to ADR Ref column; note durable 3-stage dispatch saga and decoupled session provisioning.
  2. Update `OPS-002` row: add `ADR-016` to ADR Ref column; note purpose-aware stop lifecycle and quarantine cleanup.
- **Co-requisite Synchronizations**: `docs/04_ARCHITECTURE.md`, `docs/14_FAILURE_RECOVERY.md`.
- **Execution Order**: Phase 3 (Recovery Playbooks & Store Entity Provenance).
- **Consistency Checks**:
  - Assert `ADR-016` appears in `FR-005` and `OPS-002` rows and any stage/purpose/quarantine token in those rows matches §6 and §27 exactly.
- **Completion Criteria**: Traceability matrix correctly traces requirements to accepted ADR-016.

---

### S1–S9 ADR Literal Cross-Check Matrix

Before any future canonical edit, verify each item against the cited accepted ADR sections. ADR-016 §6 is the sole token registry; §27 is the authoritative candidate DDL. If a target document conflicts with either, the target document is corrected only after the planning gate is approved. If ADR-016 conflicts with itself or lacks a required upstream mapping, stop at the blocker in §7; do not amend ADR-016 under this plan.

| Plan item | Exact ADR-016 source sections | Required literal comparison before/after reconciliation |
|---|---|---|
| **S1** | §§4, 6(2–3), 8 (D2), 10 (D4), 17 (D11), 27 | `dispatch_operations.stage` is exactly `DISPATCH_BOUND` / `SEND_REQUESTED` / `SEND_CONFIRMED`; `SEND_CONFIRMED` only follows HTTP 200; match its §27 columns and `UNIQUE(attempt_id)`. `/kill` wire carries session identity only. Keep the restore-capability issue blocked per §7. |
| **S2** | §§6(1–7), 7 (D1), 9 (D3), 10 (D4), 12 (D6), 17 (D11), 27 | Match `CLEAN` / `QUARANTINED`; runtime `status` remains `ACTIVE` / `IDLE` / `TERMINATED`. Compare every entity column, stage CHECK, purpose CHECK, partial unique index, `UNIQUE(attempt_id)`, and quarantine CHECK/default to §27. Compare resolution states to §6; keep `stage` separate from `resolution_state` and do not add CHECK constraints/columns absent from the authoritative DDL. |
| **S3** | §§5, 6(5–8), 11–18 (D5–D12), 22–24 | Match the 13 states, 22 domain transitions / 25 graph edges, allowed transitions, recovery dispositions, audit-event names, quarantine gate predicates, and stop purpose outcomes. Verify retry also checks unresolved provisioning stages `PROVISION_REQUESTED` / `PROVISION_FAILED`. |
| **S4** | §§5, 7 (D1), 8 (D2), 10 (D4), 16 (D10), 18 (D12), 27 | Keep `TaskContract` fields unchanged. Compare only `session_id`, `terminal_generation`, `recovery_disposition`, and `quarantine_state` as TaskAttempt additions; compare `AO_BLOCKED_ESCALATED`, `ended_at`, and atomic escalation behavior to §6/§16/§18. |
| **S5** | §§4, 6(2–5), 13 (D7), 17 (D11), 19 (D13), 27 | Match public status/generation facts and strict `idle` / `waiting_input` whitelist. Verify `/kill` path and session-only wire identity; no `purpose`, generation, or deadline wire field and no atomic fence claim. Match stop stage CHECK and `resolution_state` separately to §6/§27. |
| **S6** | §§6(1–8), 9 (D3), 11 (D5), 12 (D6), 17 (D11), 19 (D13), 22–25, 27 | Startup provisioning predicate is only `PROVISION_REQUESTED` -> `PROVISION_FAILED`; unknown dispatch is `SEND_REQUESTED`; reconcile `SEND_CONFIRMED`; stop stage/resolution predicates match §§17/19. Check Class A/B/C outcomes, audit events, and clearance conditions against §§6/11/12/17/25; no absence/replacement-only clearance. |
| **S7** | §§6, 7, 9, 10, 12, 17, 18, 27–29 | Compare registered store entities, exact columns/constraints and task-attempt additions against §27; compare operation and event names against §6/§25; preserve AO boundary and reuse justification without inventing upstream capability. |
| **S8** | §§4, 6, 8, 10, 12, 17, 19, 27–29 | Compare phase deliverables to approved adapter/wire boundaries and the three operation DDLs. Do not assert a restore wire route or remove the approved resume behavior until §7 is resolved. |
| **S9** | §§5, 6, 8, 10, 17, 27–28 | Verify `FR-005` and `OPS-002` references and descriptions against the exact dispatch stages, stop purposes, quarantine values, and DDL; no new token or state may be introduced by traceability prose. |

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
1. **Batch 1**: Domain entities and state rules are compared field-for-field with ADR-016 §§6/27, preserve the 13 canonical states and 22 executable transitions, and use only Model A snapshot fields.
2. **Batch 2**: Architecture sequence diagram aligns with decoupled provisioning and the exact 3-stage dispatch saga; no unverified `/restore` wire route is documented, and the §7 restore-capability blocker is resolved before changing canonical adapter/phase capability claims.
3. **Batch 3**: Recovery playbooks REC-002, REC-004, REC-014 accurately model the 5-step startup recovery sweep and double-gated quarantine clearance rules without inventing numeric timeouts.

---

## 6. Completion & Handoff Criteria

Reconciliation execution will be considered complete and ready for External Supervisor Audit when:
1. All 9 target items (S1..S9) are implemented strictly within their specified sections.
2. Items S10 and S11 are verified as compatible without mutation.
3. All 13 decisions D1..D13 are 100% traceable to updated canonical sections.
4. `git diff --check` passes with zero errors.
5. Zero production code (`internal/**/*.go`), zero schema migrations, and zero Task Contracts have been created.
6. This plan's approval does not authorize canonical edits or change phase state. Until a separate execution authorization and Task Contract exist, retain `TASK_P03_003 = NOT_RELEASED`, `P03_CODE = HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT`, and `ACTIVE_GATE = CANONICAL_SPEC_RECONCILIATION_PLANNING`.

---

## 7. External Supervisor Blockers (No ADR Mutation)

### `ADR16-PLAN-BLOCKER-001` — Resume / Restore Capability Has No Pinned Public Wire Mapping

- **Evidence in accepted ADR-016**: §7 item 4 requires resuming an existing terminated-but-restorable session through a governed `ResumeWorker` path; §7 item 5 describes restoring a session while preserving `session_id` and changing `terminal_generation`; §9 item 1 also requires a governed `ResumeWorker` path; §13 item 2 says an `exited` session requires session restore.
- **Boundary gap**: §4 enumerates the pinned public spawn, send, kill, and observation routes but does not define a public resume/restore route or map `ResumeWorker` to an approved AO operation. D11 (§17) identifies the `/kill` wire request and explicitly limits it to session identity. The ADR therefore leaves the adapter capability/wire boundary for resume/restore unresolved; this plan does not assume that an endpoint exists or that the capability is absent.
- **External decision required**: Confirm the exact pinned public AO capability and approved adapter mapping for `ResumeWorker`, or direct a separately governed ADR correction. Until decided, do not add a `/restore` wire route, delete the approved resume behavior from canonical specs, or claim session restore is unavailable.
- **Gate impact**: This blocker remains open for External Supervisor resolution before S1, S5, S6, S7, or S8 canonical changes that depend on resume/restore behavior. It does not change the accepted status of ADR-016 and does not authorize production implementation.
