# ADR-012: TaskContract Revision Model and Attempt Binding

> **Status**: ACCEPTED (Amended 2026-09-22 / P02 Pre-Code Decision Gate)
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Supersedes/Amends**: Amends ADR-010 (Contract Immutability) and ADR-011 (Attempt Identity Binding)

---

## Context
ADR-010 established that a TaskContract dispatched to a worker is permanently immutable. ADR-011 established that execution reports are attempt-scoped (`attempt_id`) and separated transient conversation identity from execution identity.

However, a post-freeze architectural audit revealed a structural contradiction between the domain model, contract specification, and workflow state machine:
1. `docs/05_DOMAIN_MODEL.md` declared `contract_id` and claimed a 1:1 relationship between `Task` and `TaskContract`.
2. `docs/08_TASK_CONTRACT.md` and `docs/schemas/task-contract.schema.json` lacked `contract_id`, `revision_number`, and lineage fields entirely.
3. The canonical state machine allows `REVISION_REQUIRED -> READY` ("Revision Contract Created"), which requires creating an updated work specification without violating the immutability of the previously dispatched contract.
4. `TaskAttempt` lacked an explicit binding to the specific `contract_id` under which the execution attempt was launched.

Without a formal contract revision and attempt-binding model, multi-turn revision cycles risk mutating historical contract truth or creating ambiguity over which specification version governed a given attempt.

---

## Decision

### 1. Task Logical Identity (`task_id`)
- `task_id` represents the stable, logical work-item identity across its entire lifecycle, spanning initial creation, execution attempts, revision cycles, and eventual approval or cancellation.
- Format: Pattern `^TASK-[A-Za-z0-9_-]+$`.

### 2. Contract Revision Identity (`contract_id`)
- Every specific work specification is an immutable entity identified by `contract_id`.
- Format: Pattern `^CONTRACT-[A-Za-z0-9_-]+$` (e.g., `CONTRACT-TASK-P01-001-01`).
- A `Task` aggregate has a 1-to-many relationship with `TaskContract` revisions (`Task 1 -> 1..* TaskContract`).

### 3. Revision Number & Lineage
- Each `TaskContract` carries:
  - `revision_number`: A monotonically increasing integer within the scope of the `task_id`. Initial revision is `revision_number = 1`.
  - `supersedes_contract_id`: An optional reference (`string` or `null`) pointing to the immediately preceding `contract_id` in the revision chain. For `revision_number = 1`, `supersedes_contract_id` is `null`.

### 4. Post-Dispatch Immutability
- Once a `TaskContract` has been dispatched (`READY -> DISPATCHED`), that specific `contract_id` is permanently and strictly immutable. No field may be modified in place.

### 5. Revision Mechanics & Planning Authority (`REVISION_REQUIRED -> READY`)
- When ChatGPT or the Supervisor issues a revision request from `REVIEWING`:
  1. `request_revision(task_id, attempt_id, feedback, required_fixes)` ONLY:
     - Validates that `attempt_id == active_attempt_id`;
     - Formally records the `ReviewDecision` (feedback and required fixes) bound to that `attempt_id`;
     - Transitions state `REVIEWING -> REVISION_REQUIRED`.
  2. `request_revision` **MUST NOT** silently invent, synthesize, or mutate a `TaskContract`.
  3. To continue the workflow, the planning authority (ChatGPT or human) must explicitly formulate and supply a complete NEW `TaskContract` revision.
  4. Conceptually, invoking `dispatch_task(new_contract)` while the task is in `REVISION_REQUIRED` performs strict validation:
     - `new_contract.task_id == current.task_id`;
     - `new_contract.contract_id` is a new, unique identifier;
     - `new_contract.revision_number == previous_revision_number + 1`;
     - `new_contract.supersedes_contract_id == previous_contract_id`;
     - Lineage, allowed scope, and full schema are valid.
  5. Upon successful validation, the task transitions `REVISION_REQUIRED -> READY`.
  6. The task then proceeds through normal pre-dispatch allocation and dispatch.
  7. No hidden automatic scope broadening is permitted. The previous `TaskContract` remains unchanged in historical storage as immutable evidence.

### 6. Retry Mechanics (`FAILED -> READY`)
- If a task execution fails due to process crash, timeout, or report defect (`TaskState = FAILED`), and the Supervisor decides to retry without modifying the work specification or scope:
  - The task transitions `FAILED -> READY`.
  - The retry **reuses the existing `contract_id`** and `revision_number` (no specification change).
  - However, upon dispatch, a **NEW `TaskAttempt`** is allocated.

### 7. TaskAttempt Identity and Contract Binding
- Every execution attempt is represented by a `TaskAttempt` entity with the following canonical attributes:
  - `attempt_id`: Unique, opaque, immutable execution identity (e.g., `ATTEMPT-01`, `ATT-UUID`).
  - `attempt_number`: Monotonically increasing integer within the task (1, 2, 3...).
  - `task_id`: Stable logical task identifier.
  - `contract_id`: Explicit binding to the exact `TaskContract` revision governing this attempt.
  - `expected_report_path`: Deterministic report destination: `.supervisor/reports/<task_id>/<attempt_id>.json`.
  - `started_at`: Timestamp of attempt dispatch.
  - `ended_at`: Timestamp of attempt termination (nullable while running).
  - `worker_report_raw`: Raw JSON string of the worker report payload upon handoff (nullable).
- Invariant: `TaskAttempt -> exactly 1 TaskContract`.

### 8. Pre-Dispatch Allocation Invariant
- A `TaskAttempt` must be allocated **BEFORE** every `READY -> DISPATCHED` transition.
- This guarantees that `attempt_id`, `attempt_number`, and `expected_report_path` exist and are communicated to the worker within the dispatch payload prior to worker process launch.
- `TaskContract` describes the static work specification and **never** contains `attempt_id`.

### 9. Auditable Traceability Chain
- All telemetry, evidence, and evaluation entities bind strictly to `attempt_id`:
  - `WorkerClaim` binds to `attempt_id`;
  - `Evidence` binds to `attempt_id`;
  - `ReviewBundle` binds to `attempt_id`;
  - `ReviewDecision` binds to `task_id` and `attempt_id`.
- Because each `TaskAttempt` binds to `contract_id`, the complete lineage:
  ```
  Task -> TaskContract (revision) -> TaskAttempt -> WorkerClaim / Evidence -> ReviewBundle -> ReviewDecision
  ```
  is fully deterministic, auditable, and immutable. Stale reviews across revision cycles are prevented.

### 10. Atomic Pre-Dispatch Persistence Invariant (P02 Pre-Code Amendment)
- From the Supervisor domain perspective, `TaskAttempt` allocation and the `READY -> DISPATCHED` state transition must be persisted **atomically** in the State Store before invoking any external execution side effects on Agent Orchestrator.
- External AO API calls (`createWorkerSession`, `sendTask`) occur strictly *after* the durable `DISPATCHED` record exists.
- In the event of a host daemon crash, power failure, or network disruption before AO responds, the Supervisor upon restart detects the durable `DISPATCHED` state, audits the worktree/session state, and cleanly reconciles the attempt without creating orphaned or unrecorded external processes.

### 11. Immutable Task Baseline Base SHA (P02 Pre-Code Amendment)
- For any logical `task_id`, initial revision (`revision_number = 1`) establishes the immutable baseline `base_sha`.
- All subsequent `TaskContract` revisions under the same `task_id` **MUST preserve that exact baseline `base_sha`**.
- Rationale: Independent Git and evidence collection evaluates the cumulative diff from the start of the task to the current head commit across all attempts. Resetting `base_sha` in a revision would conceal changes made in prior attempts from the cumulative review audit.
- If the target main branch advances and a Git rebase is required, the control plane must not silently modify `base_sha`; it must escalate under an explicit human-governed rebase workflow (`BLOCKED -> HUMAN_REQUIRED`).

---

## Consequences
- **Positive**: Resolves the contradiction between immutable contracts and iterative revision loops.
- **Positive**: Complete audit trail: historical contracts, execution attempts, and review decisions are never overwritten.
- **Positive**: Clear separation between work specification (`TaskContract`), execution iteration (`TaskAttempt`), and conversational memory (`conversation_id`).
- **Positive**: Prevents external worker activity while the task is still `READY`, and prevents silent baseline resets during revisions.
- **Negative / Trade-off**: The State Store must persist multiple contract revisions per task and support atomic multi-entity dispatch transactions.

---

## Related Documents
- `docs/05_DOMAIN_MODEL.md`
- `docs/06_WORKFLOW_STATE_MACHINE.md`
- `docs/08_TASK_CONTRACT.md`
- `docs/schemas/task-contract.schema.json`
- `docs/adr/ADR-010-task-contract-immutability.md`
- `docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`
