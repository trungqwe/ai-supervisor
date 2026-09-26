# AUDIT RECORD: P04 Pre-Contract Architecture External Re-Audit 009

> **Audit Type**: External Supervisor Pre-Contract Architecture Re-Audit
> **Audited Baseline**: `cd0417e641468abfac254cc57cca29be54d1e8e1`
> **Date**: 2026-09-26
> **Auditor**: External Supervisor
> **Target Scope**: Phase P04 Pre-Contract Architecture Remediation Round 9 Deliverables (Revision 10):
> - `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 10)
> - `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 4)
> - `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 10)
> - `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 10)
> - `AGENTS.md`
> - `docs/18_CURRENT_STATE.md`

---

## 1. Executive Summary & Verdict

The External Supervisor audited the remediation deliverables submitted for Pre-Contract Architecture Round 9 at commit `cd0417e641468abfac254cc57cca29be54d1e8e1`.

While Round 9 introduced valuable semantics for verbatim reported head SHA handling, non-deadlocking latency metrics, and stream termination states, it suffered regressions in DDL lineage triggers, table immutability guards, content-address path binding, Model 1 persistence ownership, and TaskState transition discipline.

Findings `P04-ARCH-R9-001` through `P04-ARCH-R9-006` are marked **PARTIALLY_CLOSED**, and five new findings (`P04-ARCH-R10-001` through `P04-ARCH-R10-005`) are formally recorded requiring Revision 11 remediation:
1. `P04-ARCH-R10-001`: DDL lineage, immutability, and content-address binding regression.
2. `P04-ARCH-R10-002`: Model 1 persistence ownership violation (P04C assigned Schema v9 instead of P04D).
3. `P04-ARCH-R10-003`: NFR-008 measurement boundary ambiguity and truth table gaps.
4. `P04-ARCH-R10-004`: Lease TTL checked arithmetic and stream DDL enforcement gaps.
5. `P04-ARCH-R10-005`: Non-canonical TaskState transitions and loose audit literals.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_10_REQUIRED`
- `PROPOSAL_P04_002 = REVISION_4_REQUIRED`
- `ADR_018 = REVISION_10_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_10_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_10`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 9 Findings (P04-ARCH-R9-001 through R9-006)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R9-001` | WorkerClaim Schema & Cardinality Mismatch | **PARTIALLY_CLOSED** | Verbatim 7-40 hex check added, but contract_id, composite lineage, and immutable triggers were lost (`P04-ARCH-R10-001`). |
| `P04-ARCH-R9-002` | NFR-008 Timestamps, Process Restarts & Deadlock Prevention | **PARTIALLY_CLOSED** | Deadlock eliminated via non-blocking nfr008_met, but measurement boundary naming requires resolution (`P04-ARCH-R10-003`). |
| `P04-ARCH-R9-003` | Full Stream Hash Contradiction with Hard Byte Cap | **PARTIALLY_CLOSED** | stream_state added, but exact canonical path equality and boundary combinations require DDL enforcement (`P04-ARCH-R10-004`). |
| `P04-ARCH-R9-004` | Clean Worktree, Staged/Unstaged/Untracked Verification | **PARTIALLY_CLOSED** | Clean policy adopted, but git rev-parse --absolute-git-dir was dropped and requires restoration (`P04-ARCH-R10-001`). |
| `P04-ARCH-R9-005` | Cross-Process Lease Reclaim Authority & TTL Overflow Guards | **PARTIALLY_CLOSED** | Reclaim taxonomy defined, but expires_at was not bound to ttl_seconds, and CAS transitions lacked DDL triggers (`P04-ARCH-R10-004`). |
| `P04-ARCH-R9-006` | TaskState Misalignment in Replay and Conflict Handling | **PARTIALLY_CLOSED** | Replay matrix formulated, but non-canonical state transitions and unregistered audit literals were used (`P04-ARCH-R10-005`). |

---

## 3. New Findings (P04-ARCH-R10-001 through P04-ARCH-R10-005)

### `P04-ARCH-R10-001`: DDL Lineage, Immutability & Content-Address Binding Regression
- **Severity**: High
- **Description**:
  1. `worker_claims` dropped `contract_id` and composite lineage triggers.
  2. `attempt_workspace_bindings` dropped `session_id`, `terminal_generation`, `pinned_ao_commit`, exact hex formatting, and CAS state transition triggers.
  3. Immutability triggers (`BEFORE UPDATE`, `BEFORE DELETE`) were omitted from `worker_claims`, `evidence_sets`, `review_artifacts`, and `review_bundles`.
  4. `review_artifacts` dropped `contract_id`, exact equality between canonical path and `captured_sha256`, `media_type`, and `encoding`.
- **Remediation Directive**:
  1. Restore all lineage columns (`contract_id`, `session_id`, `terminal_generation`, `pinned_ao_commit`), composite foreign keys, and lineage triggers across all tables.
  2. Restore immutable update and delete triggers for `worker_claims`, `evidence_sets`, `review_artifacts`, and `review_bundles`.
  3. Enforce exact equality: `canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256` with anti-traversal checks.

### `P04-ARCH-R10-002`: Violation of Model 1 Persistence Ownership
- **Severity**: High
- **Description**: Revision 10 documentation stated that Subtask P04C owned Schema Migration v9 and Transaction B. This directly contradicts Model 1 architecture where P04B and P04C are pure in-memory collectors, and Subtask P04D is the sole SQLite CAS orchestrator.
- **Remediation Directive**:
  1. Synchronize all documents: Subtask P04A owns Schema Migration v6 and Transaction A.
  2. Subtasks P04B and P04C are pure in-memory collectors with zero SQLite writes.
  3. Subtask P04D owns Schema Migration v9, Content-Addressed Store, lease acquisition/reclaim/release, Transaction B, Transaction C, and associated audit events.

### `P04-ARCH-R10-003`: NFR-008 Measurement Boundary Ambiguity & Truth Table Gaps
- **Severity**: Medium
- **Description**: Calling pre-commit timestamps "commit completion" created semantic ambiguity. Furthermore, `MEASUREMENT_TIMEOUT` was listed without clear semantics, and truth table combinations were unconstrained.
- **Remediation Directive**:
  1. Formally select the application pre-commit assembly & validation boundary for the immutable ReviewBundle columns: `evidence_finalized_at_epoch_ms`, `bundle_assembled_at_epoch_ms`, `compilation_latency_ms`.
  2. Record commit duration telemetry upon commit return via in-process monotonic clock inside the proposed audit event `REVIEW_BUNDLE_GENERATED`.
  3. Remove `MEASUREMENT_TIMEOUT`; restrict `latency_measurement_status` to `'MEASURED_IN_PROCESS'` and `'RECOVERED_AFTER_RESTART'`.
  4. Enforce strict truth table CHECK constraints between status, latency, and `nfr008_met`.

### `P04-ARCH-R10-004`: Lease TTL Checked Arithmetic & Stream DDL Enforcement Gaps
- **Severity**: Medium
- **Description**:
  1. `task_verification_leases` did not mathematically enforce `expires_at_epoch_ms = acquired_at_epoch_ms + ttl_seconds * 1000`.
  2. Lease state machine lacked DDL CAS transition guards.
  3. `review_artifacts` stream states lacked strict mathematical boundary constraints between `captured_bytes`, `total_observed_bytes`, `capture_limit_bytes`, and `hard_safety_limit_bytes`.
- **Remediation Directive**:
  1. Add CHECK `expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000)` and CAS transition triggers to `task_verification_leases`.
  2. Add strict stream state combination CHECK constraints enforcing exact limits.

### `P04-ARCH-R10-005`: Non-Canonical TaskState Transitions & Unregistered Audit Literals
- **Severity**: Medium
- **Description**: Documentation introduced non-canonical compound state labels (`BLOCKED / HUMAN_REQUIRED`, `RUNNING (or FAILED)`) and unregistered audit literals (`INTEGRITY_CHECK_FAILED`).
- **Remediation Directive**:
  1. Use only canonical states and transitions from `docs/06_WORKFLOW_STATE_MACHINE.md`.
  2. On dirty worktree report intake: Transaction A rolls back, task state remains `RUNNING` preserved (attempt report rejected), and `EVIDENCE_COLLECTION_FAILED` (`DIRTY_WORKTREE_DETECTED`) is emitted.
  3. On bundle hash conflict at `REVIEWING`: task state remains `REVIEWING`, Transaction C rolls back, `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`) is emitted, and automated review decisions are blocked until human supervisor resolution.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 11.
2. Remediate `PROPOSAL-P04-002` to Revision 5.
3. Remediate `DRAFT-ADR-018` to Revision 11.
4. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 11.
5. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_10`.
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
