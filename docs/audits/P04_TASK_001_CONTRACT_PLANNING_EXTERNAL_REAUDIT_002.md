# P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_002.md

**Audit Target**: Subtask P04A Task Contract & Implementation Planning Draft Re-Audit (Round 2)
**Audited Documents**:
- `docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md`
- `docs/tasks/DRAFT_TASK_CONTRACT_P04_001.md`
**Audited Remediation Commit**: `bcefbba9fd0875253b61e2a10f992b9be9a929d9`
**Prior Audit Record**: `docs/audits/P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_001.md`
**Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Authority**: External Supervisor
**Date**: 2026-09-27
**Verdict**: `DRAFT_P04A = REVISION_REQUIRED`
**Active Gate**: `TASK_P04A_CONTRACT_PLANNING`

---

## 1. Executive Summary & Audit Determination

External Supervisor conducted an independent re-audit of the remediated draft planning and contract documentation for Subtask P04A committed at `bcefbba9fd0875253b61e2a10f992b9be9a929d9`.

The audit evaluated compliance against accepted `ADR-018`, `PROPOSAL-P04-001 Revision 22`, canonical specifications, and the findings recorded in [P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_001.md](P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_REAUDIT_001.md).

### Independent Verification Confirmation
- `HEAD = origin/main = bcefbba9fd0875253b61e2a10f992b9be9a929d9`; working tree clean.
- Diff strictly matches the 5 whitelisted governance files; zero Go production or test code modified in git.
- Schema v6 DDL matches **13/13 SQL objects** byte-for-byte with accepted `ADR-018`.
- `TaskContractValidator.ValidateRaw` passes; all 6 negative probes are strictly rejected.
- Full race test suite (`go test -race -count=1 ./...`) passes cleanly with exit code `0`.
- Round 1 finding `P04A-C1-005` and Round 2 findings `P04A-R1-001` and `P04A-R1-002` are fully verified and closed.
- `P04A-R1-003` is partially closed because caller matrix and restart recovery specifications had remaining gaps.
- Three new findings (`P04A-R2-001`, `P04A-R2-002`, `P04A-R2-003`) are recorded.

**Audit Verdict**: `DRAFT_P04A = REVISION_REQUIRED`.
Release authorization remains withheld (`TASK_P04A_TASK_CONTRACT = NOT_RELEASED`).
Production code writing remains strictly held (`P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`).
Zero Phase P05 code authorized (`P05_CODE = NOT_AUTHORIZED`).
Zero candidate contracts or release contracts may be formulated until all findings are remediated and verified.

---

## 2. Status of Audit Findings

### Prior Findings Status

| Finding ID | Title | Status | Audit Verification Evidence |
|---|---|---|---|
| `P04A-C1-001` | Schema v6 DDL Drift | **CLOSED** | Parity script confirmed all 13 Schema v6 SQL objects match byte-for-byte with ADR-018. |
| `P04A-C1-002` | Sai ownership P04A/P04B | **CLOSED** | P04A consumes injected typed `GitEvidenceResult`; zero Git CLI execution or raw output parsing. |
| `P04A-C1-003` | Descriptor và hold replay sai | **CLOSED** | Descriptors A and B standardized verbatim from ADR-018; recurrence lifecycle aligned. |
| `P04A-C1-004` | Verification policy không hợp lệ | **CLOSED** | Typed `go-test-p04-001` catalog defined; `ValidateRaw` passed; 6 negative probes rejected. |
| `P04A-C1-005` | Audit literal chưa được duyệt | **CLOSED** | Literal token completely erased from plan and contract; "zero unapproved audit event literals" enforced. |
| `P04A-R1-001` | Trộn nguồn bằng chứng | **CLOSED** | `ReportedHeadSHA` removed from `GitEvidenceResult`; `WorkerReport.head_sha` sole source of `reported_head_sha`; canonical mapping and RFC 8785 JCS payload; AC-P04A-18 added. |
| `P04A-R1-002` | Thiếu verification request cho full suite | **CLOSED** | Added `"./..."` to package enum; added `VR-P04A-FULL-RACE` (timeout 300s, flags `["-v", "-race", "-count=1"]`) backing required evidence. |
| `P04A-R1-003` | Whitelist và restart recovery chưa khép kín | **PARTIALLY_CLOSED** | Progress made on scope and scanner, but caller matrix omitted store/recovery direct callers, `migrations_v5_test.go` was omitted, and recovery API diverged from ADR-018. |

---

### Round 3 Findings (New Findings)

#### Finding `P04A-R2-001`: Scope và caller matrix chưa khép kín
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**:
  1. `internal/store/migrations_v5_test.go` hardcodes `user_version == 5` on `Open()`. When `CurrentSchemaVersion` advances to 6 in P04A, this test is directly impacted, yet it was not included in `allowed_scope`.
  2. Multiple direct callers of `RecordSendRequested` (`internal/store/stop_transactions_test.go`, `restore_transactions_test.go`, `recovery_transactions_test.go`, `internal/recovery/integration_test.go`, `poller_test.go`, `timeout_monitor_test.go`) were omitted from `allowed_scope` or mischaracterized as "in allowed_scope via scanner_test.go".
- **Required Remediation**:
  1. Add to `allowed_scope`:
     - `internal/store/migrations_v5_test.go`
     - `internal/store/recovery_transactions_test.go`
     - `internal/store/restore_transactions_test.go`
     - `internal/store/stop_transactions_test.go`
     - `internal/recovery/integration_test.go`
     - `internal/recovery/poller_test.go`
     - `internal/recovery/timeout_monitor_test.go`
  2. Remove the three recovery test files from `forbidden_scope`, keeping only production files (`internal/recovery/poller.go`, `internal/recovery/timeout_monitor.go`).
  3. Update `allowed_scope` count from 32 to 39; verify zero duplicate and zero overlap with `forbidden_scope`.
  4. Re-catalog the Caller Impact Matrix directly from actual `rg` results for all direct callers of `PrepareBoundDispatch`, `RecordSendRequested`, and `dispatch.Coordinator` construction. Eliminate all "via another file" phrasing; every file must be directly in `allowed_scope` with verified role.
  5. Explicitly state that `migrations_v5_test.go` may only be modified to preserve v5 testing when `CurrentSchemaVersion` advances to 6, without weakening v5 fixtures, rollback tests, or historical migration coverage.

---

#### Finding `P04A-R2-002`: Thiếu snapshot comparison tại `RecordSendRequested`
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**:
  ADR-018 §4 Decision 1 mandates that `Coordinator.Dispatch` passes the verified `WorkspaceBindingSnapshot` (obtained from live lease revalidation) to `Store.RecordSendRequested`. In the same SQLite transaction, Store must compare the provided snapshot against the active `attempt_workspace_bindings` row. The previous draft only required `lease.Revalidate()` and checking that an active row existed, without requiring Store to perform the full durable snapshot comparison in the transaction.
- **Required Remediation**:
  1. `RecordSendRequested` must accept a verified `WorkspaceBindingSnapshot` from Coordinator after `lease.Revalidate()`.
  2. In the same SQLite transaction, Store must compare the provided snapshot against the `attempt_workspace_bindings` row that is `ACTIVE`:
     - `canonical_worktree_path`
     - `worktree_volume_serial_hex`
     - `worktree_file_id_hex`
     - `linked_gitdir_path`
     - `linked_gitdir_volume_serial_hex`
     - `linked_gitdir_file_id_hex`
     - `pinned_ao_commit`
     - exact task/attempt/contract/Pair/session/generation lineage
  3. Merely having an `ACTIVE` binding is NOT sufficient. Snapshot mismatch must:
     - strictly forbid `/send`;
     - keep task `DISPATCHED`;
     - keep dispatch operation `DISPATCH_BOUND`;
     - keep attempt open;
     - execute diagnostic Variant B transaction (`WORKSPACE_BINDING_GUARD`, literal `WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH` or `WORKSPACE_BINDING_LINEAGE_MISMATCH`, active hold `INVARIANT_MISMATCH`, binding CAS `INVALIDATED`).
  4. Add AC and behavior tests:
     - exact snapshot PASS;
     - stale/fake snapshot FAIL;
     - path or worktree FileId mismatch FAIL;
     - linked gitdir identity mismatch FAIL;
     - pinned AO commit mismatch FAIL;
     - all failure cases have AO send call count = 0;
     - atomic rollback of audit/hold/binding CAS.
  5. All direct callers of `RecordSendRequested` must supply a valid snapshot; zero snapshot-less bypass pathways.

---

#### Finding `P04A-R2-003`: API restart recovery và terminalization lệch ADR
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**:
  1. The previous draft invented an unapproved method `Reacquire(ctx, snapshot)` and renamed `WorkspaceBindingCandidate` to `WorkspaceConfig`, violating accepted ADR-018 which specifies:
     ```go
     type WorkspaceBindingAuthority interface {
         Acquire(ctx context.Context, candidate WorkspaceBindingCandidate) (WorkspaceBindingLease, error)
     }
     ```
  2. When directory substitution or physical identity mismatch is detected, the draft incorrectly stated "the attempt is escalated to human required/operator failure". Under ADR-018, automated scanner handling must keep task `DISPATCHED`, operation `DISPATCH_BOUND`, and attempt open. Terminalization is strictly a separate D12 transaction performed by a verified operator (`DISPATCHED -> FAILED`, `recovery_disposition = WORKSPACE_BINDING_INTEGRITY_FAILURE`).
- **Required Remediation**:
  1. Restore the approved ADR-018 interface verbatim:
     ```go
     type WorkspaceBindingAuthority interface {
         Acquire(ctx context.Context, candidate WorkspaceBindingCandidate) (WorkspaceBindingLease, error)
     }
     ```
     Delete all references to `Reacquire` and `WorkspaceConfig`.
  2. Startup recovery sequence:
     - read durable canonical paths and lineage from database;
     - construct `WorkspaceBindingCandidate`;
     - call `authority.Acquire(ctx, candidate)` to open a fresh lease;
     - compare `lease.Snapshot()` against durable `attempt_workspace_bindings`;
     - prior handle token or database row alone grants zero effect authority.
  3. Identity mismatch or Acquire failure:
     - zero `/send`;
     - diagnostic Variant B transaction;
     - binding `ACTIVE -> INVALIDATED`;
     - task remains `DISPATCHED`;
     - operation remains `DISPATCH_BOUND`;
     - attempt remains open.
  4. Terminalization must be a separate D12 transaction performed by a verified operator:
     - `tasks`: `DISPATCHED -> FAILED`;
     - `task_attempts`: `ended_at = now`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'`.
     Do NOT automatically transition to `HUMAN_REQUIRED` and do NOT terminalize in the scanner diagnostic transaction.
  5. Erase any phrasing stating "attempt escalated to human required/operator failure" or similar contradictory wording.

---

## 3. Provenance & Baseline Confirmation

- **Audited Remediation Baseline Commit**: `bcefbba9fd0875253b61e2a10f992b9be9a929d9`
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
| `P04A-C1-005` | Audit literal chưa được duyệt | Medium | `CLOSED` |
| `P04A-R1-001` | Trộn nguồn bằng chứng | High | `CLOSED` |
| `P04A-R1-002` | Thiếu verification request cho full suite | High | `CLOSED` |
| `P04A-R1-003` | Whitelist và restart recovery chưa khép kín | Critical | `PARTIALLY_CLOSED` |
| `P04A-R2-001` | Scope và caller matrix chưa khép kín | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-R2-002` | Thiếu snapshot comparison tại RecordSendRequested | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-R2-003` | API restart recovery và terminalization lệch ADR | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |

3. **Remediation Mandate**: Worker must perform remediation strictly on the 5 whitelisted governance files without creating Go code or candidate contracts. Findings remain `OPEN_PENDING_EXTERNAL_REAUDIT`.
