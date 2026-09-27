# P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_003.md

**Audit Target**: Subtask P04A Task Contract & Implementation Planning Draft Re-Audit (Round 3)
**Audited Documents**:
- `docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md`
- `docs/tasks/DRAFT_TASK_CONTRACT_P04_001.md`
**Audited Remediation Commit**: `22581943fecd11c63d3c5e6f38586083cfc364ba`
**Prior Audit Record**: `docs/audits/P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_002.md`
**Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Authority**: External Supervisor
**Date**: 2026-09-27
**Verdict**: `DRAFT_P04A = REVISION_REQUIRED`
**Active Gate**: `TASK_P04A_CONTRACT_PLANNING`

---

## 1. Executive Summary & Audit Determination

External Supervisor conducted an independent re-audit of the remediated draft planning and contract documentation for Subtask P04A committed at `22581943fecd11c63d3c5e6f38586083cfc364ba`.

The audit evaluated compliance against accepted `ADR-018`, `PROPOSAL-P04-001 Revision 22`, canonical specifications, and the findings recorded in [P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_002.md](P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_002.md).

### Independent Verification Confirmation
- `HEAD = origin/main = 22581943fecd11c63d3c5e6f38586083cfc364ba`; working tree clean.
- Diff strictly matches the 5 whitelisted governance files; zero Go production or test code modified in git.
- Schema v6 DDL matches **13/13 SQL objects** byte-for-byte with accepted `ADR-018`.
- `TaskContractValidator.ValidateRaw` passes; all 6 negative probes are strictly rejected.
- Full race test suite (`go test -race -count=1 ./...`) passes cleanly with exit code `0`.
- Allowed scope contains strictly 39 files with zero duplicate entries and zero overlap with the 8 forbidden scope entries.
- Snapshot equality comparison guard, `WorkspaceBindingAuthority.Acquire`, and operator D12 terminalization were confirmed in contract.
- Finding `P04A-R2-002` is verified and closed at contract level (`CLOSED_AT_CONTRACT_LEVEL`).
- One new blocking finding (`P04A-R3-001`) is recorded regarding `RecordSendRequested` caller matrix misclassification and recovery effect semantics.

**Audit Verdict**: `DRAFT_P04A = REVISION_REQUIRED`.
Release authorization remains withheld (`TASK_P04A_TASK_CONTRACT = NOT_RELEASED`).
Production code writing remains strictly held (`P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`).
Zero Phase P05 code authorized (`P05_CODE = NOT_AUTHORIZED`).
Zero candidate contracts or release contracts may be formulated until all findings are remediated and verified.

---

## 2. Status of Audit Findings

### Cumulative Findings Status

| Finding ID | Title | Status | Audit Verification Evidence |
|---|---|---|---|
| `P04A-C1-001` | Schema v6 DDL Drift | **CLOSED** | Parity script confirmed all 13 Schema v6 SQL objects match byte-for-byte with ADR-018. |
| `P04A-C1-002` | Sai ownership P04A/P04B | **CLOSED** | P04A consumes injected typed `GitEvidenceResult`; zero Git CLI execution or raw output parsing. |
| `P04A-C1-003` | Descriptor và hold replay sai | **CLOSED** | Descriptors A and B standardized verbatim from ADR-018; recurrence lifecycle aligned. |
| `P04A-C1-004` | Verification policy không hợp lệ | **CLOSED** | Typed `go-test-p04-001` catalog defined; `ValidateRaw` passed; 6 negative probes rejected. |
| `P04A-C1-005` | Audit literal chưa được duyệt | **CLOSED** | Literal token completely erased from plan and contract; "zero unapproved audit event literals" enforced. |
| `P04A-R1-001` | Trộn nguồn bằng chứng | **CLOSED** | `ReportedHeadSHA` removed from `GitEvidenceResult`; `WorkerReport.head_sha` sole source of `reported_head_sha`; canonical mapping and RFC 8785 JCS payload; AC-P04A-18 added. |
| `P04A-R1-002` | Thiếu verification request cho full suite | **CLOSED** | Added `"./..."` to package enum; added `VR-P04A-FULL-RACE` (timeout 300s, flags `["-v", "-race", "-count=1"]`) backing required evidence. |
| `P04A-R1-003` | Whitelist và restart recovery chưa khép kín | **PARTIALLY_CLOSED** | Progress achieved on scope and scanner, but caller categorization and restart recovery semantics required further precision. |
| `P04A-R2-001` | Scope và caller matrix chưa khép kín | **PARTIALLY_CLOSED** | 39 allowed files confirmed with zero overlap; caller matrix required structured categorization. |
| `P04A-R2-002` | Thiếu snapshot comparison tại RecordSendRequested | **CLOSED_AT_CONTRACT_LEVEL** | ADR-018 8-field equality comparison guard specified in transaction; AC-P04A-20 added; test evidence defined. |
| `P04A-R2-003` | API restart recovery và terminalization lệch ADR | **PARTIALLY_CLOSED** | `Acquire(ctx, candidate)` restored; operator D12 terminalization specified; scanner effect boundaries required tightening. |
| `P04A-R3-001` | Xung đột quyền effect và phân loại caller matrix của RecordSendRequested | **OPEN_PENDING_EXTERNAL_REAUDIT** | Scanner must not call RecordSendRequested; caller matrix requires structured 4-way categorization. |

---

### Round 4 Findings (New Finding)

#### Finding `P04A-R3-001`: Xung đột quyền effect và phân loại caller matrix của `RecordSendRequested`
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**:
  1. The Caller Impact Matrix in `PLAN-P04A` and `DRAFT_TASK_CONTRACT_P04_001` incorrectly listed `internal/recovery/scanner.go` as a direct caller of `RecordSendRequested`.
  2. This conflicts directly with ADR-018 (§4 Decision 1 and §3.4), which establishes `Coordinator.Dispatch` as the sole, exclusive effect gate holding the live lease through `RecordSendRequested` and the subsequent AO `/send` invocation.
  3. Scanner code in `internal/recovery/scanner.go` only performs pre-send classification and sweep; it does NOT invoke `RecordSendRequested`, must NOT commit `SEND_REQUESTED` stage, and must NOT trigger wire effects. If scanner invoked `RecordSendRequested`, a crash between intent commit and wire transmission would leave orphaned `SEND_REQUESTED` state without wire effect.
  4. Conversely, removing `scanner.go` without establishing precise post-conditions creates ambiguity where an exact physical identity match upon restart might be misconstrued as authorizing automated re-send. Under ADR-018, exact match allows recovery evaluation/unblocking, but post-conditions remain: task `DISPATCHED`, operation `DISPATCH_BOUND`, attempt open. Automated re-send after restart requires a formal Proposal/ADR addendum and is not authorized in Subtask P04A.
  5. The previous caller matrix lacked granular categorization, conflating seam definitions, direct production callers, direct test callers, and indirect behavior tests under an inaccurate aggregate count, and contained a duplicate row for `session_lifecycle_test.go`.
- **Required Remediation**:
  1. Restructure the Caller Impact Matrix into four explicit categories:
     - **Seam Definition**
     - **Direct Production Caller**
     - **Direct Test Caller**
     - **Indirect Behavior Test**
  2. Remove `internal/recovery/scanner.go` from direct callers of `RecordSendRequested`.
  3. Remove duplicate entries (e.g. redundant `session_lifecycle_test.go` row).
  4. Correctly classify indirect callers (e.g. `coordinator_test.go` tests `PrepareBoundDispatch` and `RecordSendRequested` indirectly through `Coordinator.Dispatch`).
  5. Report exact counts per category rather than an undifferentiated total.
  6. Retain `internal/recovery/scanner.go` in `allowed_scope` (39 files), as the scanner requires `WorkspaceBindingAuthority.Acquire` and physical identity comparison.
  7. Formally seal restart recovery semantics:
     - Scanner reads durable binding from DB, constructs `WorkspaceBindingCandidate`, and invokes `authority.Acquire(ctx, candidate)`.
     - Scanner compares fresh lease snapshot (`VolumeSerialNumber`, `FileIdInfo`) against durable binding.
     - Scanner strictly never calls `Store.RecordSendRequested`, never transitions stage to `SEND_REQUESTED`, and never calls AO `/send` (`RecordSendRequested` call count = 0, AO `/send` call count = 0).
     - Exact identity match: allows proceeding with pre-send classification and clearing `RECOVERY_PENDING` pursuant to approved P03 APIs; post-condition remains task `DISPATCHED`, operation `DISPATCH_BOUND`, attempt open.
     - Mismatch or acquire failure: atomic Variant B transaction, zero send, binding `INVALIDATED`, task `DISPATCHED`, attempt open.
     - Absent an approved recovery effect API, terminalization requires verified operator D12 transaction. Automated re-send after restart is not authorized in P04A.
  8. Add Acceptance Criterion `AC-P04A-21`:
     - Exact-match restart recovery: `Acquire` and identity comparison PASS, but `RecordSendRequested` call count = 0, AO `/send` call count = 0, stage remains `DISPATCH_BOUND`.
     - Mismatch/acquire failure: atomic Variant B, zero send, binding `INVALIDATED`, task `DISPATCHED`, attempt open.
     - Zero snapshot-only or scanner-direct effect pathways.

---

## 3. Provenance & Baseline Confirmation

- **Audited Remediation Baseline Commit**: `22581943fecd11c63d3c5e6f38586083cfc364ba`
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

2. **Remediation Mandate**: Worker must perform remediation strictly on the 5 whitelisted governance files without creating Go code or candidate contracts. Finding `P04A-R3-001` remains `OPEN_PENDING_EXTERNAL_REAUDIT`.
