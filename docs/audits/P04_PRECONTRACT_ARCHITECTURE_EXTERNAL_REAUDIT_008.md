# AUDIT RECORD: P04 Pre-Contract Architecture External Re-Audit 008

> **Audit Type**: External Supervisor Pre-Contract Architecture Re-Audit
> **Audited Baseline**: `6dbd7e26d59e22171271921b86c01bd6f215e59a`
> **Date**: 2026-09-26
> **Auditor**: External Supervisor
> **Target Scope**: Phase P04 Pre-Contract Architecture Remediation Round 8 Deliverables:
> - `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 9)
> - `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 3)
> - `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 9)
> - `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 9)
> - `AGENTS.md`
> - `docs/18_CURRENT_STATE.md`

---

## 1. Executive Summary & Verdict

The External Supervisor audited the remediation deliverables submitted for Pre-Contract Architecture Round 8 at commit `6dbd7e26d59e22171271921b86c01bd6f215e59a`.

The remediation successfully resolved AppContainer process I/O handle inheritance and atomic Job Object assignment (`P04-ARCH-R8-002 = CLOSED_AT_DESIGN_LEVEL`), and substantively closed snapshot timing, Git allowlists, and linked-worktree index path resolution (`P04-ARCH-R8-003 = SUBSTANTIVELY_CLOSED`).

However, six new architectural findings were recorded requiring Revision 10 remediation:
1. `worker_claims` schema enforced a strict 40-char SHA check, conflicting with canonical `worker-report.schema.json` allowing 7-40 hex chars, and lacked explicit mappings between WorkerReport, WorkerClaim, and ReviewBundle (`P04-ARCH-R9-001`).
2. NFR-008 latency check `compilation_latency_ms <= 3000` created a permanent database deadlock if the daemon restarted between Transaction B and C or exceeded 3 seconds (`P04-ARCH-R9-002`).
3. Terminating processes upon exceeding hard byte limits contradicted metadata claiming `full_stream_sha256` and `original_bytes` were known (`P04-ARCH-R9-003`).
4. Staged, unstaged, and untracked worktree modifications were unconstrained before snapshot extraction (`P04-ARCH-R9-004`).
5. Lease reclaim across daemon restarts claimed to terminate old Job Objects without valid handles, and verification request timeouts lacked checked arithmetic and aggregate caps (`P04-ARCH-R9-005`).
6. ReviewBundle replay and hash conflict handling assigned incorrect TaskStates (`P04-ARCH-R9-006`).

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_9_REQUIRED`
- `PROPOSAL_P04_002 = REVISION_3_REQUIRED`
- `ADR_018 = REVISION_9_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_9_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_9`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 8 Findings (P04-ARCH-R8-001 through R8-005)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R8-001` | WorkerClaim Cardinality, Artifact Path & Latency DDL | **PARTIALLY_CLOSED** | Cardinality resolved, but 7-40 hex pattern and latency deadlock require remediation (`P04-ARCH-R9-001`, `R9-002`). |
| `P04-ARCH-R8-002` | AppContainer Process I/O & Job Containment | **CLOSED_AT_DESIGN_LEVEL** | Explicit handle inheritance, Job Object list, and minimal sandbox DACLs verified. |
| `P04-ARCH-R8-003` | Snapshot Timing, Git Allowlist & Linked Worktree | **SUBSTANTIVELY_CLOSED** | Post-Transaction A timing verified, allowlist expanded, but clean worktree policy needed (`P04-ARCH-R9-004`). |
| `P04-ARCH-R8-004` | Lease Admission Prerequisites & Concurrency | **PARTIALLY_CLOSED** | Sequential model adopted, but cross-process reclaim authority and TTL arithmetic require remediation (`P04-ARCH-R9-005`). |
| `P04-ARCH-R8-005` | Audit Event Governance & Rollback Semantics | **PARTIALLY_CLOSED** | Proposed events adopted, but replay state matrix requires remediation (`P04-ARCH-R9-006`). |

---

## 3. New Findings (P04-ARCH-R9-001 through P04-ARCH-R9-006)

### `P04-ARCH-R9-001`: WorkerClaim Schema Mismatch with Canonical Specifications
- **Severity**: High
- **Description**:
  1. `worker-report.schema.json` permits `head_sha` of 7 to 40 characters (`^[0-9a-f]{7,40}$`), but `worker_claims.reported_head_sha` enforced exact length 40.
  2. `worker-report.schema.json` defines `worker_claims` as an array of strings, whereas `review-bundle.schema.json` defines `worker_claims` as an object.
  3. DDL checked only string length without validating JSON structure or data types.
- **Remediation Directive**:
  1. Preserve verbatim reported head SHA (`LENGTH BETWEEN 7 AND 40 AND NOT GLOB '*[^0-9a-f]*'`).
  2. Maintain `actual_head_sha` strictly within Evidence (`actual_git_evidence`), never in `worker_claims`.
  3. Define explicit, unambiguous mappings from WorkerReport to canonical WorkerClaim object and ReviewBundle `worker_claims`.
  4. Transaction A must validate WorkerReport against canonical schema, enforce data types, JCS-canonicalize payloads, and use SQLite `json_valid()` and `json_type()` checks.

### `P04-ARCH-R9-002`: NFR-008 Timestamps, Process Restarts & Deadlock Prevention
- **Severity**: High
- **Description**:
  1. In-process monotonic timers cannot bridge daemon process crashes or restarts between Transaction B and Transaction C.
  2. Database CHECK constraint `compilation_latency_ms <= 3000` permanently deadlocks tasks in `EVIDENCE_READY` if compilation is delayed or resumed after a crash.
- **Remediation Directive**:
  1. Clearly distinguish write timestamps before commit, commit completion, monotonic in-process duration, and durable wall-clock diagnostics.
  2. Enable Transaction C to persist ReviewBundles even after restarts or latency exceedance, recording `nfr008_met` (0 or 1) and `latency_measurement_status` ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART').
  3. State SLA violations as measurement diagnostics rather than persistence blockers.

### `P04-ARCH-R9-003`: Full Stream Hash Contradiction with Hard Byte Cap
- **Severity**: Medium
- **Description**: Terminating a subprocess at a 10 MB hard byte cap makes it impossible to know `full_stream_sha256` or `original_bytes`.
- **Remediation Directive**:
  1. Separate capture limit (bytes saved to disk) from hard safety limit (process termination).
  2. Introduce `stream_state` ('COMPLETE_EOF', 'TRUNCATED_AT_CAPTURE_LIMIT', 'HARD_LIMIT_TERMINATED', 'TIMEOUT_ABORTED').
  3. Set `full_stream_sha256 = NULL` when processes are terminated before observing EOF.

### `P04-ARCH-R9-004`: Uncontrolled Dirty Worktree, Staged Changes & Untracked Files
- **Severity**: High
- **Description**: Snapshots from `actual_head_sha` do not capture uncommitted staged, unstaged, or untracked changes left in the worktree.
- **Remediation Directive**:
  1. Enforce clean worktree policy for v1: Transaction A fails closed if staged, unstaged, or untracked files exist.
  2. Add `git status --porcelain=v1 -z --untracked-files=all` and `git diff-index --quiet HEAD --` to Git allowlist.
  3. Revalidate clean state before snapshot creation and before Transaction B commit.

### `P04-ARCH-R9-005`: Cross-Process Lease Reclaim Authority & TTL Overflow Guards
- **Severity**: Medium
- **Description**:
  1. New daemon instances cannot reopen anonymous Job Object handles from dead predecessors.
  2. Unbounded verification request counts can cause TTL arithmetic overflow.
- **Remediation Directive**:
  1. Distinguish same-daemon reclaim (Job handle registry join) from daemon restart recovery (P03 host lock exclusivity and `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`).
  2. Cap verification requests to max 20 and aggregate timeout to 600s in Stage B validation; use checked 64-bit integer arithmetic.

### `P04-ARCH-R9-006`: TaskState Misalignment in Replay and Conflict Handling
- **Severity**: Medium
- **Description**: Bundle hash conflicts can only occur after an initial bundle was committed and the task reached `REVIEWING`; describing all failures as preserving `EVIDENCE_READY` is incorrect.
- **Remediation Directive**:
  1. Define explicit state matrix for compile failure (`EVIDENCE_READY`), idempotent replay (`REVIEWING`), and hash conflict (`REVIEWING` preserved with conflict audit).
  2. Enforce atomic commit of ReviewBundle, audit event, and state transition in Transaction C.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 10.
2. Remediate `PROPOSAL-P04-002` to Revision 4.
3. Remediate `DRAFT-ADR-018` to Revision 10.
4. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 10.
5. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_9`.
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
