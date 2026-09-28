# P04_TASK_001_EXTERNAL_REAUDIT_002.md

> **Target**: Subtask P04A Implementation Remediation Commit `9436e6adbf1892ce21ba8dd1996728b0ebe6087d` on Branch `codex/p04-001`
> **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
> **Governing Contract**: `CONTRACT-TASK-P04-001-02` (Revision 2, `docs/tasks/TASK_CONTRACT_P04_001_REVISION_2.md`)
> **Authority**: External Supervisor
> **Classification**: External Implementation Re-Audit Record
> **Verdict**: `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED`
> **Date**: 2026-09-28

---

## 1. Executive Summary & Audit Context

The External Supervisor performed an independent re-audit of the remediation implementation committed at SHA `9436e6adbf1892ce21ba8dd1996728b0ebe6087d` on isolated branch `codex/p04-001`, evaluated strictly against governing contract `CONTRACT-TASK-P04-001-02` (Revision 2, allowed_scope 41 files).

1. **Test Suite Verification**:
   - All 6 Verification Requests (`VR-P04A-DOMAIN`, `VR-P04A-HOST`, `VR-P04A-STORE`, `VR-P04A-DISPATCH`, `VR-P04A-RECOVERY`, `VR-P04A-FULL-RACE`) executed cleanly with exit code 0 under the Go race detector.
   - Non-race full test suite (`go test -count=1 ./...`) passed with exit code 0.
   - Source formatting (`gofmt -d`) across all modified Go files produced zero diff.
   - Git diff check (`git diff db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2..HEAD --check`) was clean.
   - Scope compliance: Exactly 37 files modified from base SHA `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`, all within the 41 files of `allowed_scope` (outside scope count: 0).
   - Zero modifications to `docs/**` on branch `codex/p04-001`.

2. **Symlink Capability Gating**:
   - `TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe` was executed on the Windows host:
     ```text
     === RUN   TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe
         workspace_binding_authority_test.go:310: skipping symlink probe: SeCreateSymbolicLinkPrivilege not held by client: symlink ...: A required privilege is not held by the client.
     --- SKIP: TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe (0.00s)
     ```
   - The test correctly called `t.Skip()` without generating a false-positive `PASS`.
   - However, pursuant to Section 1 of `TASK_CONTRACT_P04_001_REVISION_2.md`, executed-PASS on a capability-qualified Windows host is a mandatory acceptance gate.
   - Acceptance criteria `AC-P04A-07` and `AC-P04A-24` remain `UNVERIFIED`.
   - `BLOCKER_P04A_SYMLINK_CAPABILITY` remains `OPEN`.

3. **Defect Findings Identified during External Probe**:
   - **Finding P04A-I3-001**: Recovery scanner ignores `lease.Close()` error on `Revalidate()` failure and `Snapshot().MatchesBinding(...)` failure paths (`internal/recovery/scanner.go:468`, `477`).
     Probe observation: `close failure was lost: report={Complete:true PendingAO:false Classified:1} err=<nil>`.
   - **Finding P04A-I3-002**: Dispatch coordinator ignores errors from `RecordVariantBDiagnostic` pre-send transactions (`internal/dispatch/coordinator.go:290`, `375`) and premature classification of `wireInitiated` before `RecordSendRequested` commit.

4. **Verdict**:
   - `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED`.
   - `P04_CODE = AUTHORIZED_P04A_REVISION_2_ONLY`.
   - `ACTIVE_GATE = TASK_P04A_IMPLEMENTATION_REMEDIATION`.
   - `MERGE_AUTHORITY = WITHHELD`.

---

## 2. Findings Reconciliation Matrix

| Finding ID | Classification | Re-Audit 002 Status | Description & Remediation Target |
|---|---|---|---|
| `P04A-I2-001` | Recovery Authority Seam | `SUBSTANTIVELY_CLOSED` | Package-level global map `storeAuthorities` deleted; `poller.go:144` directly forwards `Authority: p.Owner.Authority`; transient runner with nil authority fail-closed Variant B verified. |
| `P04A-I2-002` | Schema Admission Bypass | `SUBSTANTIVELY_CLOSED_PENDING_FINAL_REAUDIT` | Embedded canonical schema mirror `internal/store/worker_report_schema.json` compiled and validated strictly via `internal/contract`; `IngestWorkerReportRaw` is sole public intake; negative validation tests pass. |
| `P04A-I2-003` | Symlink Capability Gate | `OPEN_BLOCKED` | Symlink probe skipped due to unheld `SeCreateSymbolicLinkPrivilege`; criteria `AC-P04A-07` and `AC-P04A-24` marked `UNVERIFIED`; `BLOCKER_P04A_SYMLINK_CAPABILITY` remains `OPEN`. |
| `P04A-REV2-D1-004` | Lease Close Semantics | `NOT_CLOSED` | Scanner lease close error was lost; coordinator pre-wire/post-wire error handling seam required remediation per findings `P04A-I3-001` and `P04A-I3-002`. |
| `P04A-I3-001` | Scanner Lease Close Loss | `HIGH` (NEW) | `internal/recovery/scanner.go` uses `_ = lease.Close()` on `Revalidate()` and `MatchesBinding()` failure branches, losing close errors and erroneously reporting `Complete=true`. |
| `P04A-I3-002` | Coordinator Variant B Ignored | `HIGH` (NEW) | `internal/dispatch/coordinator.go` ignores `RecordVariantBDiagnostic` error return values, and sets `wireInitiated=true` before `RecordSendRequested` commits. |

---

## 3. Detailed Re-Audit Findings

### Finding P04A-I3-001: Scanner Loses lease.Close() Error on Revalidation/Mismatch Branches
- **Severity**: HIGH (Resource Containment & Error Propagation)
- **Status**: OPEN
- **Evidence Location**: `internal/recovery/scanner.go:468`, `477`
- **Description**:
  1. In `scanner.go`, when `lease.Revalidate()` fails or `lease.Snapshot().MatchesBinding(b)` returns false, the scanner executed `_ = lease.Close()`.
  2. If `lease.Close()` fails, the close error was completely discarded. If `RecordVariantBDiagnostic` succeeded, `classifyExecution` returned `nil`.
  3. External Supervisor probe reproduced the lost error:
     ```text
     close failure was lost: report={Complete:true PendingAO:false Classified:1} err=<nil>
     ```
  4. Any `lease.Close()` failure must cause `Runner.Run` to return a non-nil error and `report.Complete = false`, without issuing any subsequent AO calls, and without rolling back already-committed Variant B transactions.

### Finding P04A-I3-002: Coordinator Discards Variant B Diagnostic Transaction Errors
- **Severity**: HIGH (Audit / Isolation Integrity)
- **Status**: OPEN
- **Evidence Location**: `internal/dispatch/coordinator.go:290`, `375`, `379`
- **Description**:
  1. In `coordinator.go`, pre-wire lease close failure handling in `defer` and `lease.Revalidate()` failure called `_ = c.Store.RecordVariantBDiagnostic(...)`, ignoring errors from the authoritative diagnostic transaction.
  2. If `RecordVariantBDiagnostic` rolls back or fails, containment cannot be claimed; the error must be captured and combined via `errors.Join`.
  3. Furthermore, `wireInitiated = true` was set before `RecordSendRequested` committed. If `RecordSendRequested` fails, the transaction is strictly pre-effect; wireInitiated must only be marked true after successful commit of `RecordSendRequested`.

---

## 4. Governance Verdict & Active Gate

| Metric / Check | Value | Verification Authority |
| :--- | :--- | :--- |
| **Audit Target Commit** | `9436e6adbf1892ce21ba8dd1996728b0ebe6087d` | Branch `codex/p04-001` |
| **Audit Verdict** | `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED` | External Supervisor Re-Audit 002 |
| **Symlink Probe Status** | `SKIPPED (UNVERIFIED)` | Host lacks `SeCreateSymbolicLinkPrivilege` |
| **Open Blockers** | `BLOCKER_P04A_SYMLINK_CAPABILITY = OPEN` | Capability-qualified host required |
| **Active Findings** | `P04A-I2-003`, `P04A-I3-001`, `P04A-I3-002` | Tracked for remediation |
| **P04 Production Code Writing** | `AUTHORIZED_P04A_REVISION_2_ONLY` | Strictly 41 files in `allowed_scope` |
| **Active Gate** | `TASK_P04A_IMPLEMENTATION_REMEDIATION` | Governance checkpoint |
| **Merge Authority** | `WITHHELD` | Branch `codex/p04-001` must NOT be merged |
