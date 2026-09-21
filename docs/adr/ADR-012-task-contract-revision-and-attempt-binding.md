# ADR-012: TaskContract Revision Model and Attempt Binding

> **Status**: ACCEPTED
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Supercedes/Amends**: Amends ADR-010 (Contract Immutability) and ADR-011 (Attempt Identity Binding)

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

### 5. Revision Mechanics (`REVISION_REQUIRED -> READY`)
- When ChatGPT or the Supervisor issues a revision request from `REVIEWING`:
  1. The task transitions `REVIEWING -> REVISION_REQUIRED`, recording the `ReviewDecision` (feedback and required fixes) bound to the current `attempt_id`.
  2. To re-enter execution, the task transitions `REVISION_REQUIRED -> READY`.
  3. This transition **must generate a NEW TaskContract**:
     - Same `task_id`;
     - New, unique `contract_id`;
     - `revision_number = previous_revision_number + 1`;
     - `supersedes_contract_id = previous_contract_id`;
     - Updated `objective`, `requirements`, `allowed_scope`, `required_tests`, or `acceptance_criteria` incorporating the required fixes.
  4. The previous `TaskContract` remains unchanged in historical storage as immutable evidence.

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

---

## Consequences
- **Positive**: Resolves the contradiction between immutable contracts and iterative revision loops.
- **Positive**: Complete audit trail: historical contracts, execution attempts, and review decisions are never overwritten.
- **Positive**: Clear separation between work specification (`TaskContract`), execution iteration (`TaskAttempt`), and conversational memory (`conversation_id`).
- **Negative / Trade-off**: The State Store must persist multiple contract revisions per task rather than treating `TaskContract` as a singleton child of `Task`.

---

## Related Documents
- `docs/05_DOMAIN_MODEL.md`
- `docs/06_WORKFLOW_STATE_MACHINE.md`
- `docs/08_TASK_CONTRACT.md`
- `docs/schemas/task-contract.schema.json`
- `docs/adr/ADR-010-task-contract-boundaries-and-immutability.md`
- `docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`
