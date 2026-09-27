# P04_TASK_001_EXTERNAL_REAUDIT_001.md — External Re-Audit 001: Subtask P04A Implementation Remediation

> **Target**: Subtask P04A Implementation Remediation (`CONTRACT-TASK-P04-001-01`)
> **Contract Release Commit**: `dda382dee2c64456dc281768275082be73107056`
> **Implementation Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
> **Audited Implementation SHA**: `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6`
> **Branch**: `codex/p04-001`
> **Authority**: External Supervisor
> **Verdict**: `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED`
> **Merge Authority**: `WITHHELD`
> **Date**: 2026-09-28

---

## 1. Executive Summary

An external re-audit was conducted on the remediated implementation of Subtask P04A (Workspace Binding Authority, Schema Migration v6, Seam Extension & Report Intake) submitted on branch `codex/p04-001` at commit `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6`.

### Independent Verification Results
1. **Repository & Branch Alignment**: `HEAD = origin/codex/p04-001 = b955fc40ef3cc66a5ea456dc4a7c13262c9229d6`. Working tree is clean.
2. **Test Suite**: `go test -race -count=1 ./...` exited `0` across all packages:
   - `internal/ao`: PASS (1.954s)
   - `internal/audit`: PASS (1.201s)
   - `internal/contract`: PASS (1.357s)
   - `internal/dispatch`: PASS (9.103s)
   - `internal/domain`: PASS (1.180s)
   - `internal/host`: PASS (2.334s)
   - `internal/recovery`: PASS (17.662s)
   - `internal/stop`: PASS (3.644s)
   - `internal/store`: PASS (44.731s)
   - `internal/workflow`: PASS (1.149s)
   - `test/integration`: PASS (26.213s)
3. **Diff & Format Cleanliness**: `git diff db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2..HEAD --check` exited `0`. `gofmt -d` exited `0` (empty output).
4. **Whitelist Scope Adherence**: Exactly 35 files modified, 100% within the 39 whitelisted files in `allowed_scope`. Zero files modified in `docs/**`, `cmd/**`, or unapproved recovery paths on this implementation branch.
5. **Pre-Send Effect Gate Hardening**:
   - `RecordSendRequested` signature converted to strictly non-variadic with mandatory `domain.WorkspaceBindingSnapshot`.
   - Store fallback copying binding from database removed; 100% callers pass snapshot.
   - Resend pathway from `DISPATCHED` in `Coordinator.Dispatch` eliminated.

### Re-Audit Evaluation & Independent Probes
Despite passing the existing test suite, independent probing revealed critical architectural seam defects and admission bypasses:
1. **Critical Seam Defect (P04A-I2-001)**: The recovery scanner in `scanner.go` relied on a mutable global map `storeAuthorities` keyed by pointer `*store.Store`. An independent probe proved that a newly instantiated `Runner` with `Authority = nil` silently reused the authority registered by a prior run and preserved `binding_state = ACTIVE` instead of failing closed to `INVALIDATED` (Probe exit code 1).
2. **Schema Admission Bypass (P04A-I2-002)**: An exported `IngestWorkerReport` helper accepts an already-decoded struct and only runs `report.Validate()`, enabling callers using standard `encoding/json` to discard unrecognized fields and bypass `additionalProperties: false`.
3. **Unverified Capability Probe (P04A-I2-003)**: The symlink test in `workspace_binding_authority_test.go` skipped execution due to unheld Windows privileges but returned `PASS`, creating a false-positive verification for AC-P04A-07.
4. **Report Discrepancies (REPORT-DISCREPANCY-001)**: The implementation report asserted zero BEL control bytes in the released contract (in contradiction to Erratum 001) and hallucinated schema fields not present in `worker-report.schema.json`.

Consequently, the implementation verdict is **REVISION_REQUIRED** and merge authority into `main` remains strictly **WITHHELD**.

---

## 2. Audit Reconciliation Table

| Finding ID | Title / Subsystem | Re-Audit Verdict | Resolution Summary & Remaining Defect |
|---|---|---|---|
| `P04A-I1-001` | Release Contract Artifact Discrepancy | `CLOSED_WITH_APPEND_ONLY_ERRATUM_LIMITATION` | Erratum 001 committed on `main` at `2022a93837946680fe307b80b5331134aa08e697`. Raw JSON block verified byte-identical. Historical file wrapper defects documented. |
| `P04A-I1-002` | Snapshot-less `RecordSendRequested` Bypass | `CLOSED` | Signature made strictly non-variadic. DB fallback removed. Zero snapshot-less callers across entire codebase. Mismatches execute Variant B. |
| `P04A-I1-003` | Fabricated Workspace Identity & Untrusted Fallbacks | `CLOSED` | Dummy paths (`C:\repo\worktree`), synthetic file IDs, and 40 zero hashes eliminated. Pinned AO commit derived from explicit source; `.git` pointer file properly resolved. |
| `P04A-I1-004` | Implicit Fake Authority & Recovery Lease Lifetime | `NOT_CLOSED` | `defaultRecoveryAuthority` removed and lease held with `defer lease.Close()`. However, superseded by critical finding `P04A-I2-001` due to global `storeAuthorities` registry and ignored `lease.Close()` error. |
| `P04A-I1-005` | Unauthorized `DISPATCHED` Resend Pathway | `CLOSED` | Resend branch removed from `Coordinator.Dispatch`. Tasks in `DISPATCHED` reject duplicate dispatch calls; AO send count remains 0. |
| `P04A-I1-006` | Worker Report Schema & Dirty Intake Lineage | `PARTIALLY_CLOSED` | Lineage checks in dirty intake enforced. However, schema validation bypass remains open via exported decoded struct API (see `P04A-I2-002`). |
| `P04A-I1-007` | Host Filesystem & Substitution Coverage | `PARTIALLY_CLOSED` | ReFS/NTFS checks, UNC normalization, junctions, subst drives, and anti-rename lease locks verified. Symlink probe skipped due to privilege; see `P04A-I2-003`. |
| `P04A-I2-001` | Hidden Recovery Authority Reuse via Global Registry | `CRITICAL` (NEW) | Mutable global map `storeAuthorities` in `scanner.go` circumvents fail-closed semantics for nil authority. Ignored `lease.Close()` error. Requires `poller.go` scope authorization. |
| `P04A-I2-002` | WorkerReport Schema Admission Bypass via Decoded Struct | `HIGH` (NEW) | Exported helper accepts decoded struct, bypassing Draft-07 JSON Schema `additionalProperties: false`. Raw JSON must be sole public admission path. |
| `P04A-I2-003` | AC-P04A-07 Symlink Proof Unexecuted (False PASS) | `HIGH` (NEW) | Symlink probe skipped on Windows due to unheld client privileges but test reported PASS. Capability must be verified or explicitly marked UNVERIFIED. |
| `REPORT-DISCREPANCY-001` | Inaccurate Implementation Report Claims | `MEDIUM` (NEW) | Report asserted zero BEL bytes (wrapper contains 3 BEL bytes) and hallucinated schema fields not present in `worker-report.schema.json`. |

---

## 3. Detailed Re-Audit Findings

### Finding P04A-I2-001: Hidden Recovery Authority Reuse via Global Registry
- **Severity**: CRITICAL (Isolation / Fail-Closed Architecture Violation)
- **Status**: OPEN
- **Evidence Location**: `internal/recovery/scanner.go:437`, `458–463`, `482`
- **Description**:
  1. To circumvent the prohibition on editing `internal/recovery/poller.go` (which constructs a transient `Runner` at line 144 without forwarding `p.Owner.Authority`), `scanner.go` introduced a package-level global registry:
     ```go
     var storeAuthorities = make(map[*store.Store]domain.WorkspaceBindingAuthority)
     ```
  2. When a `Runner` with `Authority = nil` executes `classifyExecution`, it looks up `r.Authority = storeAuthorities[r.Store]`. If any prior `Runner` on the same `*store.Store` had an authority, subsequent `Runner` instances silently inherit that authority across separate operations.
  3. **Independent Probe Execution & Failure**:
     ```text
     === RUN   TestRecoveryScanner_NilAuthorityProbe
     nil authority reused hidden prior authority:
     binding_state=ACTIVE, want INVALIDATED
     PROBE_EXIT=1
     ```
  4. Furthermore, at line 482 of `scanner.go`, the return error from `lease.Close()` is ignored (`defer lease.Close()`), violating fail-closed resource management.
- **Architectural Seam & Scope Conflict**:
  - The root cause of this defect is that `poller.go:144` was excluded from `allowed_scope` in `CONTRACT-TASK-P04-001-01`.
  - A clean, architectural fix requires updating `poller.go:144` to pass `Authority: p.Owner.Authority` directly to the transient runner.
- **Remediation Requirement**:
  - Formulate proposal `PROPOSAL-P04-003` and task contract revision `DRAFT_TASK_CONTRACT_P04_001_REVISION_2` authorizing the addition of strictly `internal/recovery/poller.go` to `allowed_scope`.
  - Delete `storeAuthorities` and all mutable global registries.
  - Forward authority directly in `poller.go`: `Runner{..., Authority: p.Owner.Authority}`.
  - Ensure any `Runner` with `Authority == nil` unconditionally executes Variant B (`WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`, active hold `INVARIANT_MISMATCH`, CAS binding `ACTIVE -> INVALIDATED`).
  - Capture and handle `lease.Close()` errors.

### Finding P04A-I2-002: WorkerReport Schema Admission Bypass via Decoded Struct
- **Severity**: HIGH (Intake Validation & Boundary Integrity)
- **Status**: OPEN
- **Evidence Location**: `internal/store/report_intake_transactions.go:22`
- **Description**:
  - `Store` exports `IngestWorkerReport(ctx, report domain.WorkerReport, ...)` which receives an already-decoded Go struct.
  - When callers unmarshal raw JSON into `domain.WorkerReport` using Go's standard `encoding/json.Unmarshal`, any unexpected properties in the JSON payload are silently dropped.
  - As a result, the Draft-07 constraint `"additionalProperties": false` defined in `docs/schemas/worker-report.schema.json` is completely bypassed whenever this struct API is used.
- **Remediation Requirement**:
  - Raw JSON bytes (`[]byte`) validated against `docs/schemas/worker-report.schema.json` via the approved JSON Schema engine must be the **sole public admission path** for worker reports (`IngestWorkerReportRaw`).
  - Exported struct-based ingestion must be removed or made strictly internal/private, accepting only pre-validated structures.
  - Add negative regression test asserting that raw JSON with unrecognized fields is rejected at admission.

### Finding P04A-I2-003: AC-P04A-07 Symlink Proof Unexecuted (False PASS)
- **Severity**: HIGH (Verification Integrity)
- **Status**: OPEN
- **Evidence Location**: `internal/host/workspace_binding_authority_test.go:TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe`
- **Description**:
  - Creating symbolic links on Windows requires `SeCreateSymbolicLinkPrivilege` or Developer Mode enabled.
  - When run under standard accounts, the test encountered an error and returned early:
    ```text
    skipping symlink probe ... A required privilege is not held by the client.
    --- PASS: TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe
    ```
  - Because the test returned nil/early without marking `t.Skip()` or verifying the substitution behavior, the test suite reported `PASS` without actually executing the symlink verification logic.
- **Remediation Requirement**:
  - If the runtime environment has sufficient privilege, the symlink probe must execute and prove detection/rejection.
  - If the environment lacks privilege, the probe must call `t.Skip()` and the acceptance criterion `AC-P04A-07` symlink dimension must be explicitly documented as `UNVERIFIED` in governance records. It must never report a false-positive `PASS`.

### Finding REPORT-DISCREPANCY-001: Inaccurate Implementation Report Claims
- **Severity**: MEDIUM (Governance Reporting Integrity)
- **Status**: OPEN (Noted for Future Reports)
- **Description**:
  1. The implementation report claimed the released contract contained "0 byte BEL/control characters". However, `docs/tasks/TASK_CONTRACT_P04_001.md` contains 3 ASCII BEL bytes (offsets 1317, 1942, 35335) and typographic artifact `"ull"`, as accurately recognized and bounded by Erratum 001.
  2. The implementation report listed `WorkerReport` fields as containing `test_results`, `execution_metrics`, `environment_fingerprint`, `confidence_score`, `git_status`, and `completion_evidence`. None of these fields exist in `docs/schemas/worker-report.schema.json`.
- **Remediation Requirement**:
  - Governance and audit reports must accurately reflect actual file byte representations and canonical schema definitions without discrepancy.

---

## 4. Governance Directives & Next Gate Actions

1. **Gate Directives**:
   - `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED`
   - `ACTIVE_GATE = TASK_P04A_IMPLEMENTATION_REMEDIATION`
   - `P04_CODE = HELD_PENDING_P04A_CONTRACT_REVISION`
   - `Merge Authority`: **WITHHELD** (Branch `codex/p04-001` must NOT be merged into `main`).
2. **Required Governance Deliverables**:
   - Create `docs/proposals/PROPOSAL-P04-003-p04a-recovery-authority-seam-remediation.md` detailing the seam adjustment (`internal/recovery/poller.go`), global map removal, raw JSON intake requirement, and symlink testing policy.
   - Create `docs/tasks/DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md` (`CONTRACT-TASK-P04-001-02`) with allowed scope expanded strictly to 40 files, status `NOT_RELEASED`.
   - Update `AGENTS.md` and `docs/18_CURRENT_STATE.md`.
3. **Execution Prohibition**:
   - Zero production Go code may be modified on `main` or implementation branches until `CONTRACT-TASK-P04-001-02` is audited and formally released by the External Supervisor.
   - Base commit for implementation remediation remains pinned at `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`.
