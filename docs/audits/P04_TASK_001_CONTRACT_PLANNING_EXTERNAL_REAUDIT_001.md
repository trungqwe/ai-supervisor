# P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_001.md

**Audit Target**: Subtask P04A Task Contract & Implementation Planning Draft Re-Audit (Round 1)
**Audited Documents**:
- `docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md`
- `docs/tasks/DRAFT_TASK_CONTRACT_P04_001.md`
**Audited Remediation Commit**: `351164dc063394b6ca3dfa4e430d7c9697c68eff`
**Audited Draft Commit**: `80b997c89d57280286c63599c1c539161b31b5eb`
**Integration Audit Reference Commit**: `33cf3b11db82e355086172764ff7df8821f4cdef`
**Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Authority**: External Supervisor
**Date**: 2026-09-27
**Verdict**: `DRAFT_P04A = REVISION_REQUIRED`
**Active Gate**: `TASK_P04A_CONTRACT_PLANNING`

---

## 1. Executive Summary & Audit Determination

External Supervisor conducted an independent re-audit of the remediated draft planning and contract documentation for Subtask P04A committed at `351164dc063394b6ca3dfa4e430d7c9697c68eff`.

The audit evaluated compliance against accepted `ADR-018`, `PROPOSAL-P04-001 Revision 22`, canonical specifications, and the findings recorded in [P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_AUDIT_001.md](P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_AUDIT_001.md).

### Independent Verification Confirmation
- `HEAD = origin/main = 351164dc063394b6ca3dfa4e430d7c9697c68eff`; working tree clean.
- Diff strictly matches the 5 whitelisted governance files; zero Go production or test code modified in git.
- Schema v6 DDL matches **13/13 SQL objects** byte-for-byte with accepted `ADR-018`.
- `TaskContractValidator.ValidateRaw` passes; all 6 negative probes are strictly rejected.
- Round 1 findings `P04A-C1-001` through `P04A-C1-004` are fully verified and closed.
- `P04A-C1-005` is partially closed because the literal token still appeared in phrasing as a negative prohibition.
- Three new findings (`P04A-R1-001`, `P04A-R1-002`, `P04A-R1-003`) are recorded.

**Audit Verdict**: `DRAFT_P04A = REVISION_REQUIRED`.
Release authorization remains withheld (`TASK_P04A_TASK_CONTRACT = NOT_RELEASED`).
Production code writing remains strictly held (`P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`).
Zero Phase P05 code authorized (`P05_CODE = NOT_AUTHORIZED`).
Zero candidate contracts or release contracts may be formulated until all findings are remediated and verified.

---

## 2. Status of Audit Findings

### Round 1 Findings Status

| Finding ID | Title | Status | Audit Verification Evidence |
|---|---|---|---|
| `P04A-C1-001` | Schema v6 DDL Drift | **CLOSED** | Parity script confirmed all 13 Schema v6 SQL objects (3 tables, 1 index, 9 triggers) match byte-for-byte with ADR-018. |
| `P04A-C1-002` | Sai ownership P04A/P04B | **CLOSED** | Architectural boundaries reconciled: P04A consumes injected typed `GitEvidenceResult`; zero Git CLI execution or raw output parsing in P04A. |
| `P04A-C1-003` | Descriptor và hold replay sai | **CLOSED** | Descriptors A and B standardized verbatim from ADR-018; Variant B fields complete without `colliding_event_id`; recurrence lifecycle aligned. |
| `P04A-C1-004` | Verification policy không hợp lệ | **CLOSED** | `antigravity-standard` isolated to `worker_profile`; typed `go-test-p04-001` catalog defined; `ValidateRaw` passed; 6 negative probes rejected. |
| `P04A-C1-005` | Audit literal chưa được duyệt | **PARTIALLY_CLOSED** | Operational emission removed, but literal token remained present in plan/contract text as a prohibition. Full closure requires complete text erasure. |

---

### Round 2 Findings (New Findings)

#### Finding `P04A-R1-001`: Trộn nguồn bằng chứng (Claim vs. Evidence Conflation)
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**: In `docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md` §3.5.1, `GitEvidenceResult` included a field `ReportedHeadSHA string`. Under ADR-018 and the zero-trust review architecture, `reported_head_sha` is an unverified worker claim whose sole authoritative source is `WorkerReport.head_sha`. Subtask P04B is an independent evidence collector providing only `actual_head_sha` and `actual_base_sha`. Merging `ReportedHeadSHA` into `GitEvidenceResult` blurs the boundary between worker claims and independent evidence. Furthermore, Transaction A described persisting `worker_claims` as "ingests verbatim worker reports" rather than defining the canonical field mapping.
- **Required Remediation**:
  1. Remove `ReportedHeadSHA` from `GitEvidenceResult`.
  2. Declare `WorkerReport.head_sha` as the sole source of `worker_claims.reported_head_sha`, stored verbatim.
  3. `GitEvidenceResult` must strictly contain only independent evidence collected by P04B: `actual_head_sha`, `actual_base_sha`, cleanliness/diff information, and typed collection errors. P04A must never write `actual_head_sha` into `worker_claims`.
  4. Replace "ingests verbatim worker reports" with canonical mapping:
     `WorkerReport.head_sha -> reported_head_sha`
     `files_changed -> claimed_files_changed`
     `tests -> tests`
     `worker_claims -> textual_claims`
     `build_status -> build_status`
     followed by RFC 8785 JCS canonicalization into `payload_json`.
  5. Add AC and behavior test simulating `reported_head_sha != actual_head_sha` to prove zero-trust separation and absence of mutual overwrite.

---

#### Finding `P04A-R1-002`: Thiếu verification request cho full suite
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**: The draft contract declared `test_log_full_suite_race_exit_zero` in `required_evidence`, but the verification requests only covered four individual subpackages (`./internal/domain/...`, `./internal/host/...`, `./internal/store/...`, `./internal/dispatch/...`). No verification request executed the repository-wide full race test suite (`./...`), creating an unverified gap against the required evidence.
- **Required Remediation**:
  1. Add `"./..."` to the package enum of profile `go-test-p04-001`.
  2. Add `VR-P04A-FULL-RACE` to `verification_requests`:
     `profile_id = "go-test-p04-001"`
     `cwd = "."`
     `timeout_seconds = 300`
     `parameters.package = "./..."`
     `parameters.flags = ["-v", "-race", "-count=1"]`
  3. Maintain negative package probe using a value outside the enum (e.g. `./cmd/...`).
  4. Ensure `required_evidence` item `test_log_full_suite_race_exit_zero` is directly backed by `VR-P04A-FULL-RACE`.

---

#### Finding `P04A-R1-003`: Whitelist và restart recovery chưa khép kín
- **Severity**: Critical
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**:
  1. Extending `PrepareBoundDispatch` and `RecordSendRequested` to enforce Schema v6 workspace bindings impacts multiple callers across `internal/store`, `internal/recovery`, and `test/integration`.
  2. No legacy API or public caller path may create or send a dispatch without a valid workspace binding.
  3. If existing callers or tests must be updated to maintain invariants and pass the full race test suite, each affected file must be enumerated explicitly in `allowed_scope` without broad wildcards.
  4. Scope containment must be strictly consistent: no file in `allowed_scope` may be covered by `forbidden_scope`. Previously, `forbidden_scope` included a blanket `"internal/recovery/**"`.
  5. ADR-018 mandates daemon restart recovery:
     * Daemon restart invalidates all in-memory capability leases; prior OS handles are closed.
     * The recovery scanner (`internal/recovery/scanner.go`) must reopen the worktree root and linked gitdir using fresh authority.
     * It must match exact canonical path, `VolumeSerialNumber`, and `FileIdInfo` against durable `attempt_workspace_bindings`.
     * A prior handle token or standalone database row does not grant effect authority.
     * Reacquire failure or identity mismatch must fail-closed with zero `/send`.
- **Required Remediation**:
  1. Establish a comprehensive Caller Impact Matrix for `PrepareBoundDispatch`, `RecordSendRequested`, and `dispatch.Coordinator` construction.
  2. Add `internal/recovery/scanner.go`, `internal/recovery/scanner_test.go`, `test/integration/ao_harness_test.go`, and caller store test files to `allowed_scope`.
  3. Remove blanket `"internal/recovery/**"` from `forbidden_scope` and explicitly forbid non-allowed recovery files (`poller.go`, `poller_test.go`, `timeout_monitor.go`, `timeout_monitor_test.go`, `integration_test.go`) to ensure zero scope overlap.
  4. Define a narrow authority interface in domain boundary (`internal/domain/workspace_binding.go`) to prevent circular package dependencies (`internal/host -> internal/recovery -> internal/host`).
  5. Add AC and behavior tests for daemon restart invalidation, exact reacquire PASS, identity mismatch FAIL, and zero effect.

---

## 3. Provenance & Baseline Confirmation

- **Canonical Integration Audit Commit**: `33cf3b11db82e355086172764ff7df8821f4cdef`
- **Audited Draft Commit**: `80b997c89d57280286c63599c1c539161b31b5eb`
- **Remediated Audit Baseline Commit**: `351164dc063394b6ca3dfa4e430d7c9697c68eff`
- **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
- **Candidate Wipe Verification**: Zero candidate contracts exist; zero release contracts formulated; zero Go code added.

---

## 4. Governance Status & Directives

1. **State Invariants**:
   - `TASK_P04A_TASK_CONTRACT = NOT_RELEASED`
   - `P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`
   - `ACTIVE_GATE = TASK_P04A_CONTRACT_PLANNING`
   - `P05_CODE = NOT_AUTHORIZED`
   - `AUTOMATIC_RESTORE = DISABLED`
   - `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`

2. **Findings Tracking Table**:

| Finding ID | Title | Severity | Status |
|---|---|---|---|
| `P04A-C1-001` | Schema v6 DDL Drift | Critical | `CLOSED` |
| `P04A-C1-002` | Sai ownership P04A/P04B | High | `CLOSED` |
| `P04A-C1-003` | Descriptor và hold replay sai | High | `CLOSED` |
| `P04A-C1-004` | Verification policy không hợp lệ | High | `CLOSED` |
| `P04A-C1-005` | Audit literal chưa được duyệt | Medium | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-R1-001` | Trộn nguồn bằng chứng | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-R1-002` | Thiếu verification request cho full suite | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-R1-003` | Whitelist và restart recovery chưa khép kín | Critical | `OPEN_PENDING_EXTERNAL_REAUDIT` |

3. **Remediation Mandate**: Worker must perform remediation strictly on the 5 whitelisted governance files without creating Go code or candidate contracts. Findings remain `OPEN_PENDING_EXTERNAL_REAUDIT`.
