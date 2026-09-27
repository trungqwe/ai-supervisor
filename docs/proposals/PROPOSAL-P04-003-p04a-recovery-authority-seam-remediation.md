# PROPOSAL-P04-003: Subtask P04A Recovery Authority Seam Remediation, Canonical Schema Admission, and Contract Revision 2

> **Proposal ID**: PROPOSAL-P04-003
> **Target Task Contract**: `CONTRACT-TASK-P04-001-02` (Revision 2 of `TASK_CONTRACT_P04_001.md`)
> **Active Gate**: `TASK_P04A_IMPLEMENTATION_REMEDIATION`
> **Authority**: `docs/24_CHANGE_GOVERNANCE.md` (Level 3 Canonical Architecture / Level 6 Task Contract Governance)
> **Status**: `PROPOSED` (Awaiting External Supervisor Audit & Approval)
> **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
> **Affected Files**:
> - `docs/tasks/DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md`
> - `internal/recovery/poller.go` (Added to `allowed_scope`; removed from `forbidden_scope`)
> - `internal/recovery/scanner.go` (Elimination of `storeAuthorities` global map; `lease.Close()` error handling)
> - `internal/store/report_intake_transactions.go` (Raw JSON sole public intake path)
> - `internal/host/workspace_binding_authority_test.go` (Explicit symlink privilege check and skip policy)
> **Architectural Alignment**: Fully aligned with ADR-018 (Accepted), ADR-016 (Accepted), ADR-017 (Accepted). Zero ADR amendments required.

---

## 1. Governance Context & Problem Statement

During External Re-Audit 001 of Subtask P04A implementation commit `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6` (`docs/audits/P04_TASK_001_EXTERNAL_REAUDIT_001.md`), the External Supervisor recorded finding `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED` with three substantive findings:

### 1.1. Root Cause Analysis of Seam 1: Hidden Recovery Authority Reuse (Finding P04A-I2-001)
1. **Contract Scope Restriction**: In `CONTRACT-TASK-P04-001-01`, `internal/recovery/poller.go` was placed in `forbidden_scope`.
2. **Implementation Workaround & Failure**: In `internal/recovery/poller.go:144`, a transient `recovery.Runner` is constructed within `Poller.PollOnce` using only historical P03 fields (`Store, AO, Handoff, Actor`), omitting `p.Owner.Authority`. Because the implementation agent could not modify `poller.go` without violating task scope immutability, it introduced a global mutable map in `internal/recovery/scanner.go`:
   ```go
   var storeAuthorities = make(map[*store.Store]domain.WorkspaceBindingAuthority)
   ```
3. **Probe Proof of Defect**: This global map caused persistent cross-operation authority leakage. An independent probe demonstrated that a newly created `Runner` with `Authority = nil` silently reused the authority registered during an earlier run on the same store pointer, incorrectly preserving `binding_state = ACTIVE` instead of failing closed to `INVALIDATED`:
   ```text
   nil authority reused hidden prior authority:
   binding_state=ACTIVE, want INVALIDATED
   PROBE_EXIT=1
   ```
4. **Lease Error Handling**: Furthermore, `scanner.go:482` ignored the return value of `lease.Close()`, failing closed resource safety rules.

### 1.2. Root Cause Analysis of Seam 2: WorkerReport Schema Bypass (Finding P04A-I2-002)
1. In `internal/store/report_intake_transactions.go`, the store exported `IngestWorkerReport` accepting an already-unmarshaled `domain.WorkerReport` Go struct.
2. Standard JSON unmarshaling into Go structs silently discards unrecognized fields. Callers utilizing this struct-based entry point bypassed the Draft-07 constraint `"additionalProperties": false` specified in `docs/schemas/worker-report.schema.json`.
3. To enforce canonical schema containment, raw JSON validation must be the **sole public admission path**.

### 1.3. Root Cause Analysis of Seam 3: Symlink Test Privilege Policy (Finding P04A-I2-003)
1. Windows requires `SeCreateSymbolicLinkPrivilege` or Developer Mode to create symbolic links.
2. Under non-elevated CI/local environments, `TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe` logged a message and returned nil, resulting in an unexecuted test falsely reporting `PASS`.
3. Test execution must strictly distinguish between verified passing execution and skipped unprivileged probes, keeping `AC-P04A-07` symlink status explicitly documented.

---

## 2. Proposed Remediation Architecture

This proposal defines the architectural rules and contract revision required to resolve findings `P04A-I2-001`, `P04A-I2-002`, `P04A-I2-003`, and `P04A-I1-004`:

### 2.1. Rule 1: Scoped Seam Expansion (Delta = +1 File)
- **Authorized Expansion**: Add strictly `internal/recovery/poller.go` to `allowed_scope` in `CONTRACT-TASK-P04-001-02`.
- **Allowed Scope Total**: Increases from 39 to exactly 40 files.
- **Forbidden Scope Boundary**: Remove `internal/recovery/poller.go` from `forbidden_scope`. Preserve `internal/recovery/timeout_monitor.go` in `forbidden_scope`.
- **Zero Scope Overlap**: Guaranteed zero overlap between `allowed_scope` and `forbidden_scope`.
- **Prohibition on Scope Creep**: Absolutely NO expansion of scope to P04B, P04C, P04D, P05, or daemon bootstrap (`cmd/supervisor/**`).

### 2.2. Rule 2: Elimination of Global State & Direct Authority Forwarding
- **Delete Global State**: Remove `var storeAuthorities` and all mutable package-level authority registries from `internal/recovery/scanner.go`.
- **Direct Owner Forwarding**: In `internal/recovery/poller.go:144`, forward authority directly from the completed owner:
  ```go
  runner := &Runner{
      Store:     p.Store,
      AO:        p.AO,
      Handoff:   p.Owner.Handoff,
      Actor:     p.Actor,
      Authority: p.Owner.Authority,
      Now:       p.Owner.Now,
  }
  ```
- **Unconditional Fail-Closed Semantics**: If a `Runner` executes `classifyExecution` with `r.Authority == nil`, it must immediately fail closed and execute the Variant B diagnostic transaction (`WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`, active hold `INVARIANT_MISMATCH`, CAS binding `ACTIVE -> INVALIDATED`) with zero wire calls (`AO send count = 0`).

### 2.3. Rule 3: Robust Lease Lifecycle & Error Handling
- Capture and propagate any error returned by `lease.Close()` in `internal/recovery/scanner.go` and `internal/dispatch/coordinator.go`. Do not silently ignore handle release failures.

### 2.4. Rule 4: Sole Public Raw JSON Intake & Canonical Schema Validation
- **Public API Boundary**:
  - `Store.IngestWorkerReportRaw(ctx context.Context, operationID, attemptID, contractID, sessionID, generation string, rawReportJSON []byte, evidence domain.GitEvidenceResult, actor string, at time.Time) error` is the **sole public admission entry point** for worker reports.
  - Exported struct-based ingestion `IngestWorkerReport` is removed or converted to an internal private helper called only after `IngestWorkerReportRaw` has successfully validated the raw bytes.
- **Canonical Engine Validation**:
  - Raw JSON bytes must be validated against `docs/schemas/worker-report.schema.json` using the pre-compiled Draft-07 JSON Schema validator (`github.com/google/jsonschema-go` or repository validator).
  - No hand-written ad-hoc validation functions that duplicate schema rules without formal parity proof.
- **Negative Regression Testing**:
  - Test suite must include negative probes demonstrating that payloads with unexpected keys (violating `additionalProperties: false`) or invalid types/ranges are rejected before any database transaction is initiated.

### 2.5. Rule 5: Explicit Symlink Privilege Verification Policy
- In `internal/host/workspace_binding_authority_test.go`:
  ```go
  if err := createSymlink(target, link); err != nil {
      if isPrivilegeNotHeldError(err) {
          t.Skip("skipping symlink probe: SeCreateSymbolicLinkPrivilege not held by client")
      }
      t.Fatalf("unexpected symlink creation error: %v", err)
  }
  ```
- If skipped via `t.Skip()`, the test runner outputs `SKIP`, preventing false-positive `PASS` claims. The acceptance criterion `AC-P04A-07` status in governance records will explicitly declare: `Junction = VERIFIED`, `Subst = VERIFIED`, `Symlink = UNVERIFIED_PENDING_ELEVATED_ENVIRONMENT`.

---

## 3. Comparison of Contract Revisions

| Attribute | `CONTRACT-TASK-P04-001-01` (Rev 1) | `CONTRACT-TASK-P04-001-02` (Rev 2 Draft) |
|---|---|---|
| **Contract ID** | `CONTRACT-TASK-P04-001-01` | `CONTRACT-TASK-P04-001-02` |
| **Revision Number** | 1 | 2 |
| **Supersedes Contract ID** | `null` | `CONTRACT-TASK-P04-001-01` |
| **Status** | `RELEASED` (Under Remediation) | `NOT_RELEASED` (Draft Proposed) |
| **Base SHA** | `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2` | `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2` (Preserved) |
| **Allowed Scope File Count** | 39 files | 40 files (+`internal/recovery/poller.go`) |
| **Forbidden Scope Recovery** | Wildcard excluded, `poller.go`, `timeout_monitor.go` | `timeout_monitor.go` only |
| **Scope Overlap** | 0 files | 0 files |
| **Recovery Authority Wiring** | Ambiguous in Poller seam | Explicit direct owner forwarding in `poller.go:144` |
| **Report Intake Admission** | Raw JSON + Exported Struct | Sole public admission path: `IngestWorkerReportRaw` |
| **Symlink Test Policy** | Implicit skip disguised as PASS | Explicit `t.Skip()` on unheld privilege |

---

## 4. Implementation Directives & Sequencing

Upon formal review, approval, and release of `CONTRACT-TASK-P04-001-02` by the External Supervisor:
1. Implementation will proceed exclusively on isolated branch `codex/p04-001` starting from base commit `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`.
2. All modifications will remain strictly bounded to the 40 files in `allowed_scope`.
3. The probe demonstrating `P04A-I2-001` failure must transition to `PASS` with zero global state.
4. Raw report schema admission bypass probe must fail closed.
5. Full race suite `go test -race -count=1 ./...` must exit 0.
