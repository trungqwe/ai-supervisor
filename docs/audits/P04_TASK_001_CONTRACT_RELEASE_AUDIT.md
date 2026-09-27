# P04_TASK_001_CONTRACT_RELEASE_AUDIT.md

**Audit Target**: `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001.md`
**Candidate Commit**: `187904a18eb3fa6a52c299285939f1f6c6272cf9`
**Candidate Blob**: `75dadfd8a3cc53ff361602a1d5ae5bbe7794fbd7`
**Raw JSON Block Blob**: `7344a11cf22c0262a52fe598c938c532d56f5b8c`
**Implementation Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Authority**: External Supervisor
**Verdict**: `EXTERNAL_APPROVED_AND_RELEASED`
**Date**: 2026-09-27
**Active Gate**: `TASK_P04A_IMPLEMENTATION`

---

## 1. Executive Summary & Release Determination

1. **Pre-Contract & Candidate Planning Baseline**:
   - Subtask P04A task contract and implementation planning draft completed four rounds of external audits and re-audits (`P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_AUDIT_001.md`, `P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_001.md`, `P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_002.md`, `P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_003.md`, `P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_004.md`).
   - Re-Audit 004 evaluated remediation commit `8582e339d417c184122421701e3c86b4115d5a7d` and issued verdict `DRAFT_P04A = READY_FOR_RELEASE_AUDIT`.
   - Candidate contract `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001.md` was formulated at commit `187904a18eb3fa6a52c299285939f1f6c6272cf9` (blob `75dadfd8a3cc53ff361602a1d5ae5bbe7794fbd7`) preserving exact byte-for-byte JSON equality with draft.

2. **Candidate Contract Verification**:
   - `base_sha` strictly references `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2` (commit merging P04 Canonical Reconciliation).
   - `allowed_scope` strictly comprises 39 whitelisted files with zero duplicates and zero overlap with the 8 forbidden scope entries.
   - `forbidden_scope` strictly forbids modifications to `cmd/supervisor/**`, `internal/ao/**`, `internal/recovery/poller.go`, `internal/recovery/timeout_monitor.go`, `internal/stop/**`, `docs/adr/**`, `docs/audits/**`, and `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md`.
   - Schema v6 DDL matches all 13 SQL objects byte-for-byte with accepted `ADR-018`.
   - Acceptance criteria AC-P04A-01 through AC-P04A-21 comprehensively cover Schema v6 migrations, Win32 128-bit physical identity, atomic `PrepareBoundDispatch`, single coordinator effect gate with 14 pre-send guards, snapshot equality guard, pre-send diagnostic Variant B, claim/evidence zero-trust separation with RFC 8785 JCS canonicalization, and restart recovery scanner fail-closed isolation without automated re-send.
   - Proposed Verification Policy Catalog defines profile `go-test-p04-001` with `cwd_policy = worktree_root`, `max_timeout_seconds = 300`, `additionalProperties = false`, and strictly enumerated packages and constant flags (`-v`, `-race`, `-count=1`).

3. **Release Verdict**:
   - `CONTRACT-TASK-P04-001-01` is formally **RELEASED** as `docs/tasks/TASK_CONTRACT_P04_001.md`.
   - Implementation is authorized exclusively on isolated branch `codex/p04-001` starting from base SHA `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`.
   - `TASK_P04A_TASK_CONTRACT = RELEASED`.
   - `CONTRACT_TASK_P04_001_01 = RELEASED`.
   - `P04_CODE = AUTHORIZED_P04A_ONLY`.
   - `ACTIVE_GATE = TASK_P04A_IMPLEMENTATION`.
   - Subtasks P04B, P04C, and P04D remain strictly `NOT_RELEASED`.
   - Phase P05 code remains strictly `NOT_AUTHORIZED`.
   - `AUTOMATIC_RESTORE = DISABLED`.

---

## 2. Pre-Release Validation Evidence

### 2.1. Structural Schema Validation (JSON Schema Draft-07)
The candidate and released contract JSON blocks were verified against `docs/schemas/task-contract.schema.json`:
- **Result**: PASS (exit code 0).
- **Blob SHA**: `7344a11cf22c0262a52fe598c938c532d56f5b8c` (byte-identical across draft, candidate, and released contracts).

### 2.2. Semantic Validation via `internal/contract.TaskContractValidator`
Go validator execution against the declared catalog yielded:
```text
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation/Positive_Candidate_Validation
    PASS: Positive candidate validation succeeded (contract_id=CONTRACT-TASK-P04-001-01, base_sha=db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2)
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation/Negative_Case_1:_unknown_profile_id
    PASS: unknown profile_id rejected: [VERIFICATION_PROFILE] field "unknown-profile": profile "unknown-profile" not found in policy catalog
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation/Negative_Case_2:_timeout_exceeds_maximum
    PASS: timeout_seconds = 301 rejected: [VERIFICATION_PROFILE] field "verification_requests[0].timeout_seconds": requested timeout 301 exceeds profile maximum 300
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation/Negative_Case_3:_flag_injection
    PASS: flag injection rejected: [VERIFICATION_PROFILE] field "verification_requests[0].parameters": parameter validation failed for profile "go-test-p04-001": validating root: validating /properties/flags: const: [-v -race -count=1 -exec=malicious] does not equal [-v -race -count=1]
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation/Negative_Case_4:_package_outside_enum
    PASS: package outside enum rejected: [VERIFICATION_PROFILE] field "verification_requests[0].parameters": parameter validation failed for profile "go-test-p04-001": validating root: validating /properties/package: enum: ./cmd/... does not equal any of: [./... ./internal/domain/... ./internal/host/... ./internal/store/... ./internal/dispatch/... ./internal/recovery/...]
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation/Negative_Case_5:_cwd_path_traversal
    PASS: cwd = ../escape rejected: [PATH_CONTAINMENT] field "cwd": parent traversal "../escape" escaping root is forbidden
=== RUN   TestTaskContractValidator_P04ReleaseAuditValidation/Negative_Case_6:_extra_parameter
    PASS: extra parameter rejected: [VERIFICATION_PROFILE] field "verification_requests[0].parameters": parameter validation failed for profile "go-test-p04-001": validating root: unexpected additional properties ["extra_property"]
--- PASS: TestTaskContractValidator_P04ReleaseAuditValidation (0.05s)
PASS
```
**Exit Code**: 0.

### 2.3. Stage B Runtime Catalog Status
- Pre-dispatch semantic validation: **PASS** for `CONTRACT-TASK-P04-001-01`.
- Runtime deployment status: `STAGE_B_RUNTIME_CATALOG = UNVERIFIED` until concrete runner deployment in Phase P04 execution. Zero inference of live runner deployment is claimed.

---

## 3. Cumulative Findings Status

All audit and re-audit findings are formally confirmed closed:

| Finding ID | Title | Status | Audit Verification Evidence |
|---|---|---|---|
| `P04A-C1-001` | Schema v6 DDL Drift | **CLOSED** | Parity script confirmed all 13 Schema v6 SQL objects match byte-for-byte with ADR-018. |
| `P04A-C1-002` | Sai ownership P04A/P04B | **CLOSED** | P04A consumes injected typed `GitEvidenceResult`; zero Git CLI execution or raw output parsing. |
| `P04A-C1-003` | Descriptor và hold replay sai | **CLOSED** | Descriptors A and B standardized verbatim from ADR-018; recurrence lifecycle aligned. |
| `P04A-C1-004` | Verification policy không hợp lệ | **CLOSED** | Typed `go-test-p04-001` catalog defined; `ValidateRaw` passed; 6 negative probes rejected. |
| `P04A-C1-005` | Audit literal chưa được duyệt | **CLOSED** | Literal token completely erased from plan and contract; "zero unapproved audit event literals" enforced. |
| `P04A-R1-001` | Trộn nguồn bằng chứng | **CLOSED** | `ReportedHeadSHA` removed from `GitEvidenceResult`; `WorkerReport.head_sha` sole source of `reported_head_sha`; canonical mapping and RFC 8785 JCS payload; AC-P04A-18 added. |
| `P04A-R1-002` | Thiếu verification request cho full suite | **CLOSED** | Added `"./..."` to package enum; added `VR-P04A-FULL-RACE` (timeout 300s, flags `["-v", "-race", "-count=1"]`) directly backing required evidence. |
| `P04A-R1-003` | Whitelist và restart recovery chưa khép kín | **CLOSED_AT_CONTRACT_PLANNING_LEVEL** | Allowed scope (39 files), forbidden scope (8 files), and zero overlap confirmed; restart recovery semantics locked. |
| `P04A-R2-001` | Scope và caller matrix chưa khép kín | **CLOSED_AT_CONTRACT_LEVEL** | 39 allowed files confirmed; caller matrix reconciled with 21 entries across 16 unique files. |
| `P04A-R2-002` | Thiếu snapshot comparison tại RecordSendRequested | **CLOSED_AT_CONTRACT_LEVEL** | ADR-018 8-field equality comparison guard specified in transaction; AC-P04A-20 added; test evidence defined. |
| `P04A-R2-003` | API restart recovery và terminalization lệch ADR | **CLOSED_AT_CONTRACT_PLANNING_LEVEL** | `Acquire(ctx, candidate)` restored; operator D12 terminalization specified; scanner effect boundaries strictly sealed. |
| `P04A-R3-001` | Xung đột quyền effect và phân loại caller matrix của RecordSendRequested | **CLOSED_AT_CONTRACT_PLANNING_LEVEL** | Scanner strictly excluded from `RecordSendRequested` and `/send`; 4-way categorization implemented; AC-P04A-21 added. |
| `P04A-R4-001` | False direct caller và sai số thống kê trong caller matrix | **CLOSED_AT_CONTRACT_LEVEL** | `internal/store/dispatch_test.go` removed from direct callers of `PrepareBoundDispatch`; numbers corrected to 3 seams, 2 prod callers, 15 test callers/constructions (13 unique files), 1 indirect test; total 21 entries on 16 unique files. |

---

## 4. Governance Verdict & Active Gate

| Metric / Check | Value | Verification Authority |
| :--- | :--- | :--- |
| **Audit Target** | `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001.md` | Candidate Task Contract |
| **Candidate Commit Reference** | `187904a18eb3fa6a52c299285939f1f6c6272cf9` | Git commit hash |
| **Candidate Blob SHA** | `75dadfd8a3cc53ff361602a1d5ae5bbe7794fbd7` | Git object hash |
| **Contract Release Status** | `RELEASED` | `TASK_CONTRACT_P04_001.md` |
| **Released Contract File Blob** | `1f5e31774705470beb914a3c3c75d95d5bc2d845` | Git object hash |
| **TaskContract JSON Block Blob** | `7344a11cf22c0262a52fe598c938c532d56f5b8c` | Byte-identical to candidate |
| **Implementation Base SHA** | `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2` | Isolated checkout base |
| **P04 Production Code Writing** | `AUTHORIZED_P04A_ONLY` | Strictly 39 files in `allowed_scope` |
| **Subtasks P04B, P04C, P04D** | `NOT_RELEASED` | Pending future contracts |
| **Phase P05 Code Writing** | `NOT_AUTHORIZED` | Strict governance hold |
| **Automatic Restore** | `DISABLED` | Fail-closed runtime invariant |
| **Active Gate** | `TASK_P04A_IMPLEMENTATION` | Governance checkpoint |
