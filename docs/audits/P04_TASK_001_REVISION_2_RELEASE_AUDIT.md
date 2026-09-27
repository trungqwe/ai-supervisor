# P04_TASK_001_REVISION_2_RELEASE_AUDIT.md

**Audit Target**: `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001_REVISION_2.md`
**Candidate Commit**: `7d80a4838ce7728058b7cd18758fea90cc2fa315`
**Candidate File Blob**: `65f3e02e0959a101d2182bb0be630dc117a12024`
**Candidate JSON Blob**: `c05a569990b2d4b5b4d1194f5aa4b123984b8151`
**Released File Blob**: `d329cc438fee756fb5c77077747412c4cfb271a8`
**Released JSON Blob**: `c05a569990b2d4b5b4d1194f5aa4b123984b8151`
**Implementation Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Existing Implementation Branch HEAD**: `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6`
**Authority**: External Supervisor
**Verdict**: `P04A_REVISION_2_RELEASE_AUDIT = EXTERNAL_APPROVED`, `CONTRACT_TASK_P04_001_02 = RELEASE_AUTHORIZED`
**Date**: 2026-09-28
**Active Gate**: `TASK_P04A_IMPLEMENTATION_REMEDIATION`
**Merge Authority**: `WITHHELD`

---

## 1. Executive Summary & Release Determination

1. **Pre-Contract Planning & Remediation Governance Baseline**:
   - Subtask P04A Task Contract Revision 1 (`CONTRACT-TASK-P04-001-01`) was released at commit `dda382dee2c64456dc281768275082be73107056` (`docs/audits/P04_TASK_001_CONTRACT_RELEASE_AUDIT.md`).
   - Implementation Audit 001 (`docs/audits/P04_TASK_001_EXTERNAL_AUDIT_001.md`) and Re-Audit 001 (`docs/audits/P04_TASK_001_EXTERNAL_REAUDIT_001.md`) recorded findings `P04A-R1-001` through `P04A-R1-005` requiring remediation.
   - Design Re-Audit 001 (`docs/audits/P04_TASK_001_REVISION_2_DESIGN_REAUDIT_001.md`) evaluated `PROPOSAL-P04-003` (Revision 2) and `DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md`, closing findings `P04A-REV2-D1-001` through `P04A-REV2-D1-005` at design level and authorizing formulation of `CANDIDATE_TASK_CONTRACT_P04_001_REVISION_2.md`.
   - Candidate contract `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001_REVISION_2.md` was formulated at candidate commit `7d80a4838ce7728058b7cd18758fea90cc2fa315` (file blob `65f3e02e0959a101d2182bb0be630dc117a12024`, raw JSON blob `c05a569990b2d4b5b4d1194f5aa4b123984b8151`).

2. **Candidate Contract Verification**:
   - **Lineage**: Strict forward progression from Revision 1 to Revision 2 (`contract_id = CONTRACT-TASK-P04-001-02`, `revision_number = 2`, `supersedes_contract_id = CONTRACT-TASK-P04-001-01`, `task_id = TASK-P04-001`).
   - **Base SHA**: Strictly immutable reference to `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`.
   - **Allowed Scope**: Strictly comprises 41 unique files (+`internal/recovery/poller.go`, +`internal/store/worker_report_schema.json`) with zero duplicate entries.
   - **Forbidden Scope**: Strictly comprises 7 entries/patterns (`cmd/supervisor/**`, `internal/ao/**`, `internal/recovery/timeout_monitor.go`, `internal/stop/**`, `docs/adr/**`, `docs/audits/**`, `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md`).
   - **Scope Non-Overlap**: Overlap between `allowed_scope` and `forbidden_scope` is exactly 0 files.
   - **Verification Requests**: 6 typed verification requests using host profile `go-test-p04-001` covering domain, host, store, dispatch, recovery, and full race test suite.
   - **Symlink Probe Mandatory Gate**: Unheld symlink privilege calling `t.Skip()` explicitly leaves criteria `AC-P04A-07` and `AC-P04A-24` `UNVERIFIED`; executed and passing symlink probe evidence on capability-qualified Windows host is a mandatory acceptance gate.
   - **Supersession**: Contract Revision 2 formally supersedes Revision 1 for remaining remediation.

3. **Release Verdict**:
   - `P04A_REVISION_2_RELEASE_AUDIT = EXTERNAL_APPROVED`.
   - `CONTRACT_TASK_P04_001_02` is formally **RELEASED** as `docs/tasks/TASK_CONTRACT_P04_001_REVISION_2.md`.
   - `TASK_P04A_REVISION_2 = RELEASED`.
   - `PROPOSAL_P04_003 = EXTERNAL_APPROVED`.
   - Implementation is authorized exclusively on branch `codex/p04-001` starting from existing HEAD `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6`.
   - `P04_CODE = AUTHORIZED_P04A_REVISION_2_ONLY`.
   - `ACTIVE_GATE = TASK_P04A_IMPLEMENTATION_REMEDIATION`.
   - `MERGE_AUTHORITY = WITHHELD`.
   - Subtasks P04B, P04C, and P04D remain strictly `NOT_RELEASED`.
   - Phase P05 code remains strictly `NOT_AUTHORIZED`.
   - `AUTOMATIC_RESTORE = DISABLED`.

---

## 2. Pre-Release Validation Evidence

### 2.1. Structural Schema Validation (JSON Schema Draft-07)
The candidate and released contract JSON blocks were verified against `docs/schemas/task-contract.schema.json`:
- **Result**: PASS (exit code 0).
- **Candidate JSON Blob SHA**: `c05a569990b2d4b5b4d1194f5aa4b123984b8151`.
- **Released JSON Blob SHA**: `c05a569990b2d4b5b4d1194f5aa4b123984b8151` (byte-for-byte identical).

### 2.2. Semantic Validation via `internal/contract.TaskContractValidator`
Go validator execution against the declared catalog and previous contract `CONTRACT-TASK-P04-001-01` yielded:
```text
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation/Positive_Candidate_Validation
    PASS: Positive candidate validation succeeded (contract_id=CONTRACT-TASK-P04-001-02, base_sha=db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2, supersedes=CONTRACT-TASK-P04-001-01)
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation/Negative_Case_1:_unknown_profile_id
    PASS: unknown profile_id rejected: [VERIFICATION_PROFILE] field "unknown-profile": profile "unknown-profile" not found in policy catalog
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation/Negative_Case_2:_timeout_exceeds_maximum
    PASS: timeout_seconds = 301 rejected: [VERIFICATION_PROFILE] field "verification_requests[0].timeout_seconds": requested timeout 301 exceeds profile maximum 300
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation/Negative_Case_3:_flag_injection
    PASS: flag injection rejected: [VERIFICATION_PROFILE] field "verification_requests[0].parameters": parameter validation failed for profile "go-test-p04-001": validating root: validating /properties/flags: const: [-v -race -count=1 -exec=malicious] does not equal [-v -race -count=1]
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation/Negative_Case_4:_package_outside_enum
    PASS: package outside enum rejected: [VERIFICATION_PROFILE] field "verification_requests[0].parameters": parameter validation failed for profile "go-test-p04-001": validating root: validating /properties/package: enum: ./cmd/... does not equal any of: [./... ./internal/domain/... ./internal/host/... ./internal/store/... ./internal/dispatch/... ./internal/recovery/...]
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation/Negative_Case_5:_cwd_path_traversal
    PASS: cwd = ../escape rejected: [PATH_CONTAINMENT] field "cwd": parent traversal "../escape" escaping root is forbidden
=== RUN   TestTaskContractValidator_P04Revision2ReleaseAuditValidation/Negative_Case_6:_extra_parameter
    PASS: extra parameter rejected: [VERIFICATION_PROFILE] field "verification_requests[0].parameters": parameter validation failed for profile "go-test-p04-001": validating root: unexpected additional properties ["extra_property"]
--- PASS: TestTaskContractValidator_P04Revision2ReleaseAuditValidation (0.07s)
PASS
```
**Exit Code**: 0.

### 2.3. Hygiene & Control Character Scan
- Control characters (< 32 except tab, newline, carriage return): 0.
- `git diff --check`: PASS (clean).
- Go code diff on `main`: exactly 0 lines / 0 files.

---

## 3. Findings Verification & Governance Matrix

| Finding ID | Classification | Resolution Summary | Release Status |
|---|---|---|---|
| `P04A-REV2-D1-001` | Architecture / Seam | Embedded schema mirror `internal/store/worker_report_schema.json` via `//go:embed` validated strictly through `internal/contract.CompileSchema` and `internal/contract.ParseAndValidateRaw`. Allowed scope expanded to 41 files. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-002` | Test / Acceptance | Symlink test lacking privilege calls `t.Skip()`, marking AC-P04A-07/24 UNVERIFIED. Executed and passing test evidence on capability-qualified host required for acceptance. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-003` | Seam Specification | `IngestWorkerReportRaw` signature reconciled to exact lineage `(ctx, taskID, contractID, attemptID, rawReportJSON, evidence, actor, at)`. Struct helper made package-private. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-004` | Error Semantics | Differentiated lease `Close()` semantics: scanner returns non-nil error, `Complete=false`, 0 AO effects; coordinator pre-wire aborts `/send` (Variant B); coordinator post-wire retains durable outcome and strictly forbids resend. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-005` | Governance | `forbidden_scope` verified at 7 entries/patterns. Caller matrix and scope catalogs aligned. | `CLOSED` |

---

## 4. Governance Verdict & Active Gate

| Metric / Check | Value | Verification Authority |
| :--- | :--- | :--- |
| **Audit Target** | `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001_REVISION_2.md` | Candidate Task Contract Revision 2 |
| **Candidate Commit Reference** | `7d80a4838ce7728058b7cd18758fea90cc2fa315` | Git commit hash |
| **Candidate File Blob SHA** | `65f3e02e0959a101d2182bb0be630dc117a12024` | Git object hash |
| **Candidate JSON Block Blob** | `c05a569990b2d4b5b4d1194f5aa4b123984b8151` | Git object hash |
| **Contract Release Status** | `RELEASED` | `TASK_CONTRACT_P04_001_REVISION_2.md` |
| **Released Contract File Blob** | `d329cc438fee756fb5c77077747412c4cfb271a8` | Git object hash |
| **Released JSON Block Blob** | `c05a569990b2d4b5b4d1194f5aa4b123984b8151` | Byte-for-byte identical to candidate |
| **Lineage** | `Revision 1 -> Revision 2` | `supersedes_contract_id = CONTRACT-TASK-P04-001-01` |
| **Implementation Base SHA** | `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2` | Isolated checkout base (immutable) |
| **Implementation Branch HEAD** | `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6` | Branch `codex/p04-001` |
| **Allowed Scope** | Exactly 41 unique files | Scope whitelist |
| **Forbidden Scope** | Exactly 7 entries/patterns | Scope blacklist |
| **Scope Overlap** | 0 files | Non-overlap verified |
| **P04 Production Code Writing** | `AUTHORIZED_P04A_REVISION_2_ONLY` | Strictly 41 files in `allowed_scope` |
| **Merge Authority** | `WITHHELD` | External Supervisor gate |
| **Subtasks P04B, P04C, P04D** | `NOT_RELEASED` | Pending future contracts |
| **Phase P05 Code Writing** | `NOT_AUTHORIZED` | Strict governance hold |
| **Automatic Restore** | `DISABLED` | Fail-closed runtime invariant |
| **Active Gate** | `TASK_P04A_IMPLEMENTATION_REMEDIATION` | Governance checkpoint |
