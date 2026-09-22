# P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_001.md — External Supervisor Re-Audit Record (PROPOSAL-P03-002 Revision 1)

> **Audited Commit**: `70514775b9bb877febc32d5fb39aea70f33513da`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: `PROPOSAL_P03_002 = REVISION_2_REQUIRED`
> **Architecture Impact**: `P03_ARCHITECTURE_CHANGE = YES`
> **ADR Requirement**: `P03_ADR_REQUIRED = YES`
> **ADR-016 Authorization**: `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET`
> **TASK_P03_001 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_002 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_003 Status**: `NOT_RELEASED`
> **P03 Code Status**: `HELD`
> **Active Gate**: `P03_TASK_003_PROPOSAL_REVISION_2`

---

## 0. External Supervisor Re-Audit Decision Summary

Independent external re-audit of `PROPOSAL-P03-002 Revision 1` (commit `70514775b9bb877febc32d5fb39aea70f33513da`) is complete.
Revision 1 substantially improved the proposal across authority citations, stop provenance, and task boundary separation.
However, Revision 1 is **NOT approved**. Eight (8) external findings (`P03T3PR2-001` through `P03T3PR2-008`) were recorded.

**Verdict**: `PROPOSAL_P03_002 = REVISION_2_REQUIRED`.

---

## 1. External Re-Audit Findings Matrix

| Finding ID | Title | Verdict | Severity | Remediation Mandate |
|---|---|---|---|---|
| `P03T3PR2-001` | `ACTIVITY_EXITED_MATRIX_CONTRADICTS_CONTRACT` | **REVISION_2_REQUIRED** | Blocker | Revision 1 prose stated `ActivityExited != session termination` and must not trigger `FAILED`, but matrix marked `DISPATCHED/RUNNING + ActivityExited -> FAILED (DECIDED)`. Pinned AO allows process to exit while session/terminal is alive. Baseline rule: `ActivityExited` alone is `AO_PROCESS_EXITED` observation only; `TaskState transition = NO AUTOMATIC TRANSITION`. |
| `P03T3PR2-002` | `AO_BLOCKED_TO_TASK_BLOCKED_NOT_AUTHORIZED` | **REVISION_2_REQUIRED** | Blocker | AO `blocked` indicates decision/permission dialog. It does not automatically establish Supervisor `TaskState BLOCKED`. Change to `Normalized observation = AO_BLOCKED_DECISION`, `TaskState mapping = UNRESOLVED_DECISION` pending ADR-016. Do not set `ended_at`. |
| `P03T3PR2-003` | `DISPATCH_RECOVERY_USES_UNAVAILABLE_EVIDENCE` | **REVISION_2_REQUIRED** | Blocker | Ephemeral `promptBytes` is not in `GET /api/v1/sessions/{id}` or durable store. Pinned public `SessionView` exposes `lastUserMessageAt`, NOT `latestUserPrompt`. Remove claims that restart reconciler checks `latestUserPrompt` or "zero prompt bytes". |
| `P03T3PR2-004` | `DURABLE_DISPATCH_STAGE_MISSING` | **REVISION_2_REQUIRED** | Blocker | Pinned `/send` contains no idempotency key (`PINNED_SEND_IDEMPOTENCY = ABSENT`, `P03T3_SEND_RECOVERY = UNRESOLVED_BLOCKER`). Crash matrix distinguished timing points without durable discriminators. Define durable dispatch stages (`DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`). Crash between `SEND_REQUESTED` and `SEND_CONFIRMED` leaves delivery unknown; prohibit blind resend. |
| `P03T3PR2-005` | `WORKERSESSION_OPTION1_SCHEMA_INCOHERENT` | **REVISION_2_REQUIRED** | Major | Option 1 claimed Pair 1-to-1 WorkerSession, but candidate schema had `session_id PRIMARY KEY`, non-unique `pair_id`, and `task_attempts.session_id FK`. Evaluated coherent data models (Model A: mutable current binding + immutable attempt snapshot; Model B: historical WorkerSession + active pointer; Model C: AttemptSessionBinding join entity). Recommend Model A as `GIẢ ĐỊNH`. |
| `P03T3PR2-006` | `ADR012_EXTERNAL_SIDE_EFFECT_CONFLICT_NOT_EXPLICIT` | **REVISION_2_REQUIRED** | Major | Accepted ADR-012 Section 10 mandates external AO calls (`createWorkerSession`, `sendTask`) occur strictly after durable `DISPATCHED`. Pre-spawning Pair session requires ADR-016 to explicitly amend ADR-012 Section 10 for session creation timing while preserving `/send` after `DISPATCHED`. Add explicit section detailing required amendment. |
| `P03T3PR2-007` | `PUBLIC_CRASH_EVIDENCE_OVERCLAIM` | **REVISION_2_REQUIRED** | Major | Public `GET /api/v1/sessions/{id}` does not expose process exit code, kernel signal, or reaper logs. Baseline classification: `isTerminated == true` + positive stop -> `WORKER_STOPPED`; `isTerminated == true` without cause -> `WORKER_TERMINATION_UNKNOWN`. `WORKER_CRASHED_PUBLIC_BASELINE = NOT_CURRENTLY_PROVABLE` from approved public snapshot. |
| `P03T3PR2-008` | `TERMINAL_GENERATION_SCOPE_OVERCLAIM` | **REVISION_2_REQUIRED** | Major | `terminalGeneration` maps to TUI `RuntimeLaunchID`. Chat mode uses `ControllerGeneration` and has no `RuntimeLaunchID`. Since canonical V1 harness `agy` is not in Chat registry, fallback to TUI applies: `AGY_V1_EXECUTION_GENERATION = terminalGeneration / RuntimeLaunchID`. Generic harness epoch is `UNRESOLVED_DECISION`. |

---

## 2. Governance Determination

1. `P03_ARCHITECTURE_CHANGE = YES` (amends durable domain relationships, attempt execution epoch snapshots, and pre-dispatch session lifecycle timing relative to ADR-012 Section 10).
2. `P03_ADR_REQUIRED = YES`.
3. `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET` (must wait until Proposal Revision 2 passes independent External Supervisor re-audit).
4. `TASK_P03_003 = NOT_RELEASED`.
5. `P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION`.
6. `ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT_002`.
