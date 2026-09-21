# 06. WORKFLOW STATE MACHINE

> **Authority**: Canonical 13-State Workflow Specification
> **Status**: Approved Baseline (Updated Architecture V2.1)

---

# 1. State Machine Diagram

```mermaid
stateDiagram-v2
    [*] --> DRAFT
    DRAFT --> READY: Contract Finalized & Validated
    DRAFT --> CANCELLED: Developer Aborts Formulation

    READY --> DISPATCHED: Supervisor Triggers Dispatch
    READY --> CANCELLED: Developer Aborts Before Dispatch

    DISPATCHED --> RUNNING: AO Confirms Worker Started
    DISPATCHED --> FAILED: AO Spawn Error / ConPTY Failure

    RUNNING --> REPORT_READY: Turn Completed + Attempt Report Validated
    RUNNING --> FAILED: Process Crash / Timeout / Report Missing, Invalid or Mismatched
    RUNNING --> BLOCKED: Worker Reports Architecture Blocker

    REPORT_READY --> EVIDENCE_READY: Independent Evidence Captured
    REPORT_READY --> FAILED: Verification Pipeline Error

    EVIDENCE_READY --> REVIEWING: Review Bundle Assembled
    EVIDENCE_READY --> FAILED: Evidence Assembly Error

    REVIEWING --> APPROVED: Supervisor Approves
    REVIEWING --> REVISION_REQUIRED: Supervisor Requests Fixes
    REVIEWING --> BLOCKED: Supervisor Flags Architecture Violation

    REVISION_REQUIRED --> READY: New Revision Contract Created (ADR-012)

    BLOCKED --> HUMAN_REQUIRED: Escalated to Developer

    FAILED --> HUMAN_REQUIRED: Automatic Retry Exhausted
    FAILED --> READY: Supervisor Retries Attempt

    HUMAN_REQUIRED --> DRAFT: Developer Resolves & Replans
    HUMAN_REQUIRED --> CANCELLED: Developer Aborts Task

    APPROVED --> [*]
    CANCELLED --> [*]
```

---

# 2. Canonical States & Transition Rules

| State | Owner | Trigger Event | Allowed Transitions | Required Evidence / Precondition |
|---|---|---|---|---|
| **`DRAFT`** | ChatGPT / User | Task formulated | `READY`, `CANCELLED` | Task contract fields populated. |
| **`READY`** | Supervisor | Contract validated | `DISPATCHED`, `CANCELLED` | Validated against `task-contract.schema.json` (`contract_id`, `task_id`, `revision_number`). Pre-allocates `TaskAttempt` before dispatch. |
| **`DISPATCHED`** | Supervisor | Dispatch sent to AO | `RUNNING`, `FAILED` | AO acknowledges receipt with HTTP 202/200. Fails if AO rejects session creation or ConPTY allocation. |
| **`RUNNING`** | AO / Worker | Worker process active in AO | `REPORT_READY`, `FAILED`, `BLOCKED` | Active session in AO daemon. Preconditions for `REPORT_READY`: (1) Turn completion signal observed (Agy Stop hook -> session IDLE); (2) Attempt report retrieved from `.supervisor/reports/<task_id>/<attempt_id>.json`; (3) JSON parses; (4) Validates against `worker-report.schema.json`; (5) `task_id` matches active Task; (6) `attempt_id` matches active TaskAttempt. If report handoff fails within bounded policy, transitions to `FAILED` with `failure_reason` set to `REPORT_MISSING`, `REPORT_INVALID`, or `REPORT_IDENTITY_MISMATCH`. |
| **`REPORT_READY`** | Supervisor | Attempt report verified | `EVIDENCE_READY`, `FAILED` | Attempt-scoped `WorkerReport` verified against schema and identities; candidate worker claims stored. Independent evidence acquisition executed. |
| **`EVIDENCE_READY`** | Supervisor | Evidence collected | `REVIEWING`, `FAILED` | Base/Head SHA verified; Git diff collected; independent test runner executed; ReviewBundle assembled. |
| **`REVIEWING`** | ChatGPT | Review Bundle loaded | `APPROVED`, `REVISION_REQUIRED`, `BLOCKED` | Review Bundle verified and submitted to ChatGPT. All decision tool calls require explicit `task_id` and `attempt_id`. |
| **`APPROVED`** | ChatGPT | Formal approval decision | `[*]` (Terminal) | Review rationale provided; tests passed; scope verified. Decision recorded only; NO automatic git merge or push. |
| **`REVISION_REQUIRED`**| ChatGPT | Revision request | `READY` | Specific defects, failing tests, or unverified claims listed. Transition to `READY` creates a new immutable `TaskContract` revision (`revision_number + 1`, `supersedes_contract_id`) per ADR-012. |
| **`BLOCKED`** | Worker / ChatGPT| Blocker flagged | `HUMAN_REQUIRED` | Explicit architectural or environmental blocker recorded. |
| **`FAILED`** | Supervisor | Crash, hang, or error | `HUMAN_REQUIRED`, `READY` (Retry) | Error log captured; `failure_reason` recorded. Transition to `READY` reuses contract revision and allocates a new `TaskAttempt` upon dispatch per ADR-012. |
| **`HUMAN_REQUIRED`** | Human Developer | Human escalation | `DRAFT`, `CANCELLED` | Escalation notified to User with full diagnosis. |
| **`CANCELLED`** | Human Developer | Manual cancellation | `[*]` (Terminal) | Worktree cleaned up; session terminated. |
