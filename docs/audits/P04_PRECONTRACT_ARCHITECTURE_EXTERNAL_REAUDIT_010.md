# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL RE-AUDIT 010

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `6e1993da150031a9465901a7019c71257de44312`
> **Audited Artifacts**:
> - `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 11)
> - `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 5)
> - `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 11)
> - `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 11)
> - `AGENTS.md`
> - `docs/18_CURRENT_STATE.md`

---

## 1. Executive Summary & Verdict

The External Supervisor audited the remediation deliverables submitted for Pre-Contract Architecture Round 10 at commit `6e1993da150031a9465901a7019c71257de44312`.

Findings `P04-ARCH-R10-001` (Lineage, Immutability & Content-Address Binding) and `P04-ARCH-R10-002` (Model 1 Persistence Ownership) are verified **CLOSED_AT_DESIGN_LEVEL**. Findings `P04-ARCH-R10-003`, `P04-ARCH-R10-004`, and `P04-ARCH-R10-005` are evaluated as **PARTIALLY_CLOSED**. Four new findings (`P04-ARCH-R11-001` through `P04-ARCH-R11-004`) are formally recorded requiring Revision 12 remediation:
1. `P04-ARCH-R11-001`: NFR-008 interval naming and post-commit telemetry causality separation.
2. `P04-ARCH-R11-002`: Append-only lease history, partial unique index, and CAS reclaim transition guards.
3. `P04-ARCH-R11-003`: Elimination of blanket transition rules, strict canonical TaskState preservation, and isolated diagnostic failure transactions.
4. `P04-ARCH-R11-004`: Strict stream limits inequality (`hard_limit > capture_limit`), read overflow algorithm, and mutual exclusion precedence.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_11_REQUIRED`
- `PROPOSAL_P04_002 = REVISION_5_REQUIRED`
- `ADR_018 = REVISION_11_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_11_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_11`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 10 Findings (P04-ARCH-R10-001 through R10-005)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R10-001` | DDL Lineage, Immutability & Content-Address Binding | **CLOSED_AT_DESIGN_LEVEL** | Lineage columns, triggers, immutable guards, and exact canonical path equality (`artifacts/xx/hash`) are fully restored across all tables. |
| `P04-ARCH-R10-002` | Model 1 Persistence Ownership | **CLOSED_AT_DESIGN_LEVEL** | Model 1 boundaries are strictly established: P04B and P04C are pure in-memory collectors with zero SQLite writes; P04D is the sole CAS and Schema v9 persistence orchestrator. |
| `P04-ARCH-R10-003` | NFR-008 Measurement Boundary Ambiguity & Truth Table | **PARTIALLY_CLOSED** | Pre-commit boundary was defined, but declaring NFR-008 met based on assembly latency contradicts canonical requirements measuring from worker completion (`P04-ARCH-R11-001`). |
| `P04-ARCH-R10-004` | Lease TTL Checked Arithmetic & Stream DDL Enforcement | **PARTIALLY_CLOSED** | TTL math check was added, but lease mutations require append-only history with partial unique index, and stream limit strict inequality requires DDL enforcement (`P04-ARCH-R11-002`, `R11-004`). |
| `P04-ARCH-R10-005` | Non-Canonical TaskState Transitions & Unregistered Audit Literals | **PARTIALLY_CLOSED** | Audit literals were standardized, but blanket non-REVIEWING to BLOCKED transition violates canonical state graph and failure audit requires isolated diagnostic transactions (`P04-ARCH-R11-003`). |

---

## 3. New Findings (P04-ARCH-R11-001 through P04-ARCH-R11-004)

### `P04-ARCH-R11-001`: NFR-008 Interval Naming & Post-Commit Telemetry Causality
- **Severity**: High
- **Description**:
  1. Canonical NFR-008 in `docs/02_REQUIREMENTS.md` measures performance from worker completion. The supervisor control plane cannot self-declare NFR-008 as "MET" based purely on the `evidence_finalized_at -> bundle_assembled_at` interval.
  2. The interval measured by `compilation_latency_ms` is an internal ReviewBundle assembly diagnostic.
  3. `REVIEW_BUNDLE_GENERATED` audit event cannot contain `commit_duration_ms` because the event is inserted inside Transaction C prior to disk commit.
- **Remediation Directive**:
  1. Rename the measurement interval to **ReviewBundle assembly diagnostic** (`compilation_latency_ms`).
  2. Document NFR-008 compliance status as explicitly **`UNVERIFIED`** (`nfr008_compliance_status TEXT NOT NULL CHECK (nfr008_compliance_status = 'UNVERIFIED')`) until an approved canonical worker-completion origin timestamp is established.
  3. Formally document `evidence_finalized_at_epoch_ms` as an application timestamp chosen before Transaction B commit and persisted durably by Transaction B (not called commit completion).
  4. Retain `REVIEW_BUNDLE_GENERATED` inside Transaction C without `commit_duration_ms`.
  5. Record `commit_duration_ms` after `tx.Commit()` returns, purely as best-effort in-process telemetry (not part of immutable evidence, not in atomic audit event, not used for SLA, no new DB event/schema).

### `P04-ARCH-R11-002`: Append-Only Lease History & CAS Reclaim Transition Guards
- **Severity**: High
- **Description**:
  1. `task_verification_leases` previously used `task_id` as PRIMARY KEY, forcing in-place mutations and preventing historical lease tracking or clean reclaim history.
  2. CAS triggers did not prevent modifying `worker_id` during terminal state transitions.
- **Remediation Directive**:
  1. Refactor `task_verification_leases` to append-only lease history: `lease_id TEXT PRIMARY KEY`.
  2. Make all identity, timestamp, and token columns immutable: `task_id`, `attempt_id`, `contract_id`, `worker_id`, `fencing_token`, `acquired_at_epoch_ms`, `ttl_seconds`, `expires_at_epoch_ms`, `predecessor_lease_id`.
  3. Enforce `UNIQUE(attempt_id, fencing_token)`.
  4. Enforce at most one `ACTIVE` lease per task and attempt using partial unique indexes:
     `CREATE UNIQUE INDEX idx_leases_single_active_attempt ON task_verification_leases(attempt_id) WHERE state = 'ACTIVE';`
     `CREATE UNIQUE INDEX idx_leases_single_active_task ON task_verification_leases(task_id) WHERE state = 'ACTIVE';`
  5. Enforce CAS terminal transitions on the existing row: `ACTIVE -> COMPLETED | EXPIRED | REVOKED`, and `EXPIRED -> RECLAIMED`. Trigger must forbid modifying `worker_id`.
  6. Reclaim is executed atomically: predecessor is marked `RECLAIMED` with `released_at_epoch_ms`, then a new `ACTIVE` lease is inserted with a higher fencing token and `predecessor_lease_id`. Strictly forbid `RECLAIMED -> ACTIVE` on the same row.
  7. Formulate complete crash, replay, and concurrent reclaim matrix.

### `P04-ARCH-R11-003`: Elimination of Blanket Transition Rules & Isolated Diagnostic Failure Transactions
- **Severity**: High
- **Description**:
  1. A blanket rule transitioning "any non-REVIEWING state with a bundle to BLOCKED" violates `docs/06_WORKFLOW_STATE_MACHINE.md`, where `BLOCKED` can only be entered from `RUNNING` or `REVIEWING`.
  2. Failure events (`EVIDENCE_COLLECTION_FAILED`, `REVIEW_BUNDLE_COMPILATION_REJECTED`) attempted within the rolled-back business transaction are rolled back with the failure.
- **Remediation Directive**:
  1. Eliminate the blanket non-REVIEWING to BLOCKED transition.
  2. On invariant mismatch: preserve the current TaskState, lock automated review approval and intake admission, and require human supervisor reconciliation (`HUMAN_SUPERVISOR_RECONCILIATION_REQUIRED`).
  3. On dirty worktree during report intake: Transaction A rolls back completely and task state remains `RUNNING`. Diagnostic audit event `EVIDENCE_COLLECTION_FAILED` (`DIRTY_WORKTREE_DETECTED`) is appended in a separate diagnostic transaction after rollback using an idempotency key. If diagnostic append fails, return a compound failure error without claiming audit was recorded.
  4. On ReviewBundle compilation or hash conflict: Transaction C rolls back completely and task state remains in its valid state (`EVIDENCE_READY` or `REVIEWING`). Diagnostic audit event `REVIEW_BUNDLE_COMPILATION_REJECTED` is appended in a separate diagnostic transaction after rollback using an idempotency key.

### `P04-ARCH-R11-004`: Stream Limit Inequality, Overflow Reader Algorithm & Precedence Matrix
- **Severity**: Medium
- **Description**:
  1. `hard_safety_limit_bytes >= capture_limit_bytes` allowed `hard == capture`, violating the architectural principle separating disk capture from process termination.
  2. Mutual exclusion between stream states, hash existence, and truncation required strict mathematical DDL constraints.
- **Remediation Directive**:
  1. Enforce strict inequality in DDL: `CHECK (hard_safety_limit_bytes > capture_limit_bytes)`.
  2. Implement stream reader algorithm that reads up to at most `hard_safety_limit_bytes + 1` bytes with overflow guard.
  3. Enforce mutually exclusive states via DDL:
     - `COMPLETE_EOF`: seen EOF, `total <= capture`, `captured = total`, `is_truncated = 0`, `full_stream_sha256 = captured_sha256 IS NOT NULL`.
     - `TRUNCATED_AT_CAPTURE_LIMIT`: seen EOF, `capture < total <= hard`, `captured = capture`, `is_truncated = 1`, `full_stream_sha256 IS NOT NULL`.
     - `HARD_LIMIT_TERMINATED`: observed `hard + 1` bytes, reader terminated, `total = hard + 1`, `captured = capture`, `is_truncated = 1`, `full_stream_sha256 IS NULL`.
     - `TIMEOUT_ABORTED`: timeout before EOF and `total <= hard`, `captured = MIN(total, capture)`, `is_truncated = 1`, `full_stream_sha256 IS NULL`.
  4. Establish explicit race precedence: (1) Hard limit breach terminates immediately; (2) Timeout abort terminates if prior to hard limit and EOF; (3) Clean EOF classifies as Complete or Truncated.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 12.
2. Remediate `PROPOSAL-P04-002` to Revision 6.
3. Remediate `DRAFT-ADR-018` to Revision 12.
4. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 12.
5. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_11`.
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
