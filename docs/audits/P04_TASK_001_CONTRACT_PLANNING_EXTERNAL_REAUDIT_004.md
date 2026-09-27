# P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_004.md

**Audit Target**: Subtask P04A Task Contract & Implementation Planning Draft Re-Audit (Round 4)
**Audited Documents**:
- `docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md`
- `docs/tasks/DRAFT_TASK_CONTRACT_P04_001.md`
**Audited Remediation Commit**: `8582e339d417c184122421701e3c86b4115d5a7d`
**Prior Audit Record**: `docs/audits/P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_003.md`
**Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Authority**: External Supervisor
**Date**: 2026-09-27
**Verdict**: `DRAFT_P04A = READY_FOR_RELEASE_AUDIT`
**Active Gate**: `TASK_P04A_CONTRACT_RELEASE_AUDIT`

---

## 1. Executive Summary & Audit Determination

External Supervisor conducted an independent re-audit of the remediated draft planning and contract documentation for Subtask P04A committed at `8582e339d417c184122421701e3c86b4115d5a7d`.

The audit evaluated compliance against accepted `ADR-018`, `PROPOSAL-P04-001 Revision 22`, canonical specifications, and findings recorded in [P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_003.md](P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_003.md).

### Independent Verification Confirmation
- `HEAD = origin/main = 8582e339d417c184122421701e3c86b4115d5a7d`; working tree clean.
- Diff strictly matches governance whitelist; zero Go production or test code modified.
- Schema v6 DDL matches **13/13 SQL objects** byte-for-byte with accepted `ADR-018`.
- `TaskContractValidator.ValidateRaw` passes; all 6 negative probes are strictly rejected.
- Full race test suite (`go test -race -count=1 ./...`) passes cleanly with exit code `0`.
- Allowed scope contains strictly 39 files with zero duplicate entries and zero overlap with the 8 forbidden scope entries.
- Snapshot equality comparison guard at `RecordSendRequested`, `WorkspaceBindingAuthority.Acquire`, and operator D12 terminalization were verified.
- Finding `P04A-R3-001` is resolved and closed at contract planning level (`CLOSED_AT_CONTRACT_PLANNING_LEVEL`).
- Finding `P04A-R4-001` is identified and remediated: the Caller Impact Matrix contained a false direct caller (`internal/store/dispatch_test.go` only had a comment referencing `PrepareBoundDispatch`, calling `prepareLegacyDispatchForTest` instead) and misstated caller statistics.
- Upon removing `internal/store/dispatch_test.go` from the Direct Test Caller table rows while retaining it in `allowed_scope` (39 files), the caller matrix accurately reflects repository call expressions:
  * Seam Definitions: 3 entries
  * Direct Production Callers: 2 seam invocations across 1 file (`internal/dispatch/coordinator.go`)
  * Direct Test Callers/Constructions: 15 entries across 13 distinct test files
  * Indirect Behavior Tests: 1 entry (`internal/dispatch/coordinator_test.go`)
  * Total: 21 entries across 16 distinct repository files (100% in `allowed_scope`).
- Findings `P04A-R2-001` and `P04A-R4-001` are **CLOSED_AT_CONTRACT_LEVEL**.
- The draft task contract is approved as ready for candidate formulation and release audit (`DRAFT_P04A = READY_FOR_RELEASE_AUDIT`).

**Audit Verdict**: `DRAFT_P04A = READY_FOR_RELEASE_AUDIT`.
Candidate formulation authorized (`docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001.md`).
Status invariant remains strictly `NOT_RELEASED` (`TASK_P04A_TASK_CONTRACT = NOT_RELEASED`).
Production code writing remains strictly held (`P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`).
Zero Phase P05 code authorized (`P05_CODE = NOT_AUTHORIZED`).
Zero released contracts may be created until formal release audit approval.

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
| `P04A-R1-002` | Thiếu verification request cho full suite | **CLOSED** | Added `"./..."` to package enum; added `VR-P04A-FULL-RACE` (timeout 300s, flags `["-v", "-race", "-count=1"]`) directly backing `test_log_full_suite_race_exit_zero`. |
| `P04A-R1-003` | Whitelist và restart recovery chưa khép kín | **CLOSED_AT_CONTRACT_PLANNING_LEVEL** | Allowed scope (39 files), forbidden scope (8 files), and zero overlap confirmed; restart recovery semantics locked. |
| `P04A-R2-001` | Scope và caller matrix chưa khép kín | **CLOSED_AT_CONTRACT_LEVEL** | 39 allowed files confirmed; caller matrix reconciled with 21 entries across 16 unique files. |
| `P04A-R2-002` | Thiếu snapshot comparison tại RecordSendRequested | **CLOSED_AT_CONTRACT_LEVEL** | ADR-018 8-field equality comparison guard specified in transaction; AC-P04A-20 added; test evidence defined. |
| `P04A-R2-003` | API restart recovery và terminalization lệch ADR | **CLOSED_AT_CONTRACT_PLANNING_LEVEL** | `Acquire(ctx, candidate)` restored; operator D12 terminalization specified; scanner effect boundaries strictly sealed. |
| `P04A-R3-001` | Xung đột quyền effect và phân loại caller matrix của RecordSendRequested | **CLOSED_AT_CONTRACT_PLANNING_LEVEL** | Scanner strictly excluded from `RecordSendRequested` and `/send`; 4-way categorization implemented; AC-P04A-21 added. |
| `P04A-R4-001` | False direct caller và sai số thống kê trong caller matrix | **CLOSED_AT_CONTRACT_LEVEL** | `internal/store/dispatch_test.go` removed from direct callers of `PrepareBoundDispatch`; numbers corrected to 3 seams, 2 prod callers, 15 test callers/constructions (13 unique files), 1 indirect test; total 21 entries on 16 unique files. |

---

## 3. Provenance & Baseline Confirmation

- **Audited Remediation Baseline Commit**: `8582e339d417c184122421701e3c86b4115d5a7d`
- **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
- **Candidate Formulation Mandate**: Candidate task contract `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001.md` must be created with status `CANDIDATE_NOT_RELEASED`, retaining exact byte-for-byte JSON equality with draft.

---

## 4. Governance Status & Directives

1. **State Invariants**:
   - `DRAFT_P04A = READY_FOR_RELEASE_AUDIT`
   - `TASK_P04A_TASK_CONTRACT = NOT_RELEASED`
   - `P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`
   - `ACTIVE_GATE = TASK_P04A_CONTRACT_RELEASE_AUDIT`
   - `P05_CODE = NOT_AUTHORIZED`
   - `AUTOMATIC_RESTORE = DISABLED`
   - `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`

2. **Transition Mandate**:
   - Formulate `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001.md`.
   - Update `PLAN-P04A`, `DRAFT_TASK_CONTRACT_P04_001`, `AGENTS.md`, and `docs/18_CURRENT_STATE.md`.
   - Zero Go production code permitted.
   - Halt and await External Supervisor release audit.
