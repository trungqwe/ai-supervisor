# PROPOSAL-P04-003: Subtask P04A Recovery Authority Seam Remediation, Embedded Canonical Schema Mirror, Differentiated Lease Close Semantics, and Contract Revision 2

> **Proposal ID**: PROPOSAL-P04-003
> **Revision**: Revision 2
> **Target Task Contract**: `CONTRACT-TASK-P04-001-02` (Revision 2 of `TASK_CONTRACT_P04_001.md`)
> **Active Gate**: `TASK_P04A_IMPLEMENTATION_REMEDIATION`
> **Authority**: `docs/24_CHANGE_GOVERNANCE.md` (Level 3 Canonical Architecture / Level 6 Task Contract Governance)
> **Status**: `EXTERNAL_APPROVED_FOR_CANDIDATE_FORMULATION` (Design Re-Audit 001)
> **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
> **Affected Files**:
> - `docs/tasks/DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md`
> - `internal/recovery/poller.go` (Added to `allowed_scope`; direct owner authority forwarding)
> - `internal/recovery/scanner.go` (Elimination of `storeAuthorities` global map; recovery `lease.Close()` error handling)
> - `internal/store/worker_report_schema.json` (Added to `allowed_scope`; embedded byte-exact mirror of canonical schema)
> - `internal/store/report_intake_transactions.go` (Raw JSON sole public intake path, private helper)
> - `internal/host/workspace_binding_authority_test.go` (Explicit symlink privilege check and acceptance gating)
> - `internal/dispatch/coordinator.go` (Pre-effect vs post-effect lease close error semantics)
> **Architectural Alignment**: Fully aligned with ADR-018 (Accepted), ADR-016 (Accepted), ADR-017 (Accepted). Zero ADR amendments required.

---

## 1. Governance Context & Problem Statement

During External Re-Audit 001 of Subtask P04A implementation commit `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6` (`docs/audits/P04_TASK_001_EXTERNAL_REAUDIT_001.md`) and Design Audit 001 of Revision 2 drafts (`docs/audits/P04_TASK_001_REVISION_2_DESIGN_AUDIT_001.md`), five design findings were identified:

### 1.1. Root Cause Analysis of Seam 1: Hidden Recovery Authority Reuse (Finding P04A-I2-001)
1. **Contract Scope Restriction**: In `CONTRACT-TASK-P04-001-01`, `internal/recovery/poller.go` was placed in `forbidden_scope`.
2. **Implementation Workaround & Failure**: In `internal/recovery/poller.go:144`, a transient `recovery.Runner` was constructed within `Poller.PollOnce` using only historical P03 fields (`Store, AO, Handoff, Actor`), omitting `p.Owner.Authority`. To bridge this without violating task scope immutability, the implementation agent introduced a global mutable map in `internal/recovery/scanner.go`:
   ```go
   var storeAuthorities = make(map[*store.Store]domain.WorkspaceBindingAuthority)
   ```
3. **Probe Proof of Defect**: This global map caused persistent cross-operation authority leakage. An independent probe demonstrated that a newly created `Runner` with `Authority = nil` silently reused the authority registered during an earlier run on the same store pointer, incorrectly preserving `binding_state = ACTIVE` instead of failing closed to `INVALIDATED`:
   ```text
   nil authority reused hidden prior authority:
   binding_state=ACTIVE, want INVALIDATED
   PROBE_EXIT=1
   ```
4. **Resolution**: Delete `storeAuthorities` entirely; add `internal/recovery/poller.go` to `allowed_scope`; forward `p.Owner.Authority` directly in `poller.go:144`.

### 1.2. Root Cause Analysis of Seam 2: WorkerReport Schema Bypass & Packaging (Findings P04A-I2-002 & P04A-REV2-D1-001)
1. In `internal/store/report_intake_transactions.go`, the store exported `IngestWorkerReport` accepting an already-unmarshaled `domain.WorkerReport` Go struct.
2. Standard JSON unmarshaling into Go structs silently discards unrecognized fields. Callers using this struct-based entry point bypassed the Draft-07 constraint `"additionalProperties": false` specified in `docs/schemas/worker-report.schema.json`.
3. To enforce canonical schema containment, raw JSON validation must be the **sole public admission path**.
4. Furthermore, supervisor binaries cannot assume execution from within a Git clone or read `docs/schemas/` via relative filesystem paths. Go does not permit `go:embed` on paths outside the package tree. An embedded byte-exact mirror `internal/store/worker_report_schema.json` must be packaged in the store package and verified for byte-exact parity against `docs/schemas/worker-report.schema.json`.

### 1.3. Root Cause Analysis of Seam 3: Symlink Privilege & Acceptance Gating (Findings P04A-I2-003 & P04A-REV2-D1-002)
1. Windows requires `SeCreateSymbolicLinkPrivilege` or Developer Mode to create symbolic links.
2. Under non-elevated environments, `TestWindowsWorkspaceBindingAuthority_AC07_SymlinkProbe` previously logged a message and returned nil, resulting in an unexecuted test falsely reporting `PASS`.
3. Using `t.Skip()` eliminates the false `PASS` log, but leaves the overall suite exiting `0`. If symlink probes are skipped, the symlink substitution dimension of the Win32 binding authority remains unproven.
4. If skipped, the `WorkerReport` must record `status = BLOCKED` or `ready_for_review = false`, leaving `AC-P04A-07` and `AC-P04A-24` `UNVERIFIED`. Final external audit approval requires verified execution logs on a capability-qualified host.

### 1.4. Root Cause Analysis of Seam 4: Intake API Lineage Drift (Finding P04A-REV2-D1-003)
Revision 1 of this proposal inadvertently specified `operationID, sessionID, generation` in `IngestWorkerReportRaw`. The active lineage seam established in Subtask P04A uses `taskID, contractID, attemptID`. Injecting additional identifiers introduces unapproved interface drift without ADR backing and is revoked.

### 1.5. Root Cause Analysis of Seam 5: Lease Close Outcome Semantics (Finding P04A-REV2-D1-004)
Handling of `lease.Close()` errors must be differentiated across execution boundaries: close failures in recovery scanner must not rollback committed transactions; close failures in coordinator pre-wire effect must forbid wire transmission; close failures post-wire effect must strictly NEVER permit resend.

---

## 2. Proposed Remediation Architecture

### 2.1. Rule 1: Scoped Seam Expansion (Delta = +2 Files, Total = 41 Files)
- **Authorized Expansion**:
  1. `internal/recovery/poller.go` (Added to `allowed_scope`; removed from `forbidden_scope`).
  2. `internal/store/worker_report_schema.json` (Added to `allowed_scope`; embedded byte-exact mirror of canonical schema).
- **Allowed Scope Total**: Increases from 39 to exactly 41 files.
- **Forbidden Scope Boundary**: Retains 7 entries/patterns (`cmd/supervisor/**`, `internal/ao/**`, `internal/recovery/timeout_monitor.go`, `internal/stop/**`, `docs/adr/**`, `docs/audits/**`, `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md`).
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

### 2.3. Rule 3: Differentiated Lease Close Outcome Semantics
Handling of `WorkspaceBindingLease.Close()` errors is strictly partitioned across operational boundaries:
1. **Recovery Scanner Boundary**:
   - If `lease.Close()` returns an error during recovery scanner execution, `scanner.Run` must return a non-nil error and report `Complete: false`.
   - The failure does NOT rollback already-committed DB transactions (e.g. classification or Variant B audit entries).
   - Zero AO wire effects may be emitted (`send call count = 0`).
2. **Coordinator Pre-Wire Effect Boundary**:
   - Any failure during `lease.Acquire`, `lease.Revalidate`, or `lease.Close` occurring prior to `/send` strictly forbids wire transmission (`send call count = 0`).
   - Task state remains `DISPATCHED`, operation remains `DISPATCH_BOUND`, attempt remains open, and diagnostic Variant B is triggered.
3. **Coordinator Post-Wire Effect Boundary (after `SEND_REQUESTED` or `SEND_CONFIRMED`)**:
   - If `lease.Close()` fails after wire transmission or durable send commitment, the error is logged and returned to the caller.
   - The close failure strictly **NEVER** grants permission to resend or retry wire dispatch.
   - Durable dispatch outcome remains authoritative (`SEND_REQUESTED` or `SEND_CONFIRMED` persisted).
   - The caller must re-read durable operation state.
4. **Regression Testing**: Test suite must include fake lease implementations exercising close error injection on both pre-effect and post-effect code paths.

### 2.4. Rule 4: Sole Public Raw JSON Intake & Embedded Canonical Schema Mirror
- **Public API Boundary**:
  - The sole public admission entry point for worker reports is:
    ```go
    Store.IngestWorkerReportRaw(
        ctx context.Context,
        taskID, contractID, attemptID string,
        rawReportJSON []byte,
        evidence domain.GitEvidenceResult,
        actor string,
        at time.Time,
    ) error
    ```
  - Zero `operationID`, `sessionID`, or `generation` parameters are added; the lineage seam established in Subtask P04A is preserved.
  - Exported struct-based ingestion `IngestWorkerReport` is removed from the public API or converted into an internal package-private helper (`ingestWorkerReportDecoded`), called exclusively after raw JSON validation succeeds.
- **Embedded Schema Mirror & Runtime Packaging**:
  - `internal/store/worker_report_schema.json` is packaged directly in the `store` package as an embedded byte-exact mirror of `docs/schemas/worker-report.schema.json`.
  - In `internal/store/report_intake_transactions.go`:
    ```go
    //go:embed worker_report_schema.json
    var workerReportSchemaBytes []byte
    ```
  - **Single Runtime Validation Seam**: Runtime validation must strictly use `internal/contract.CompileSchema` and `internal/contract.ParseAndValidateRaw`. Direct imports or usage of `github.com/google/jsonschema-go` within `internal/store` are prohibited. No disjunctive or parallel validator choices are permitted.
  - Schema compilation may be cached using `sync.Once`; initialization or compilation failure must fail closed immediately.
  - Parallel hand-written validation functions operating as independent authority are strictly prohibited.
- **Parity Verification Test**:
  - Test suite must include an automated parity test comparing `internal/store/worker_report_schema.json` with `docs/schemas/worker-report.schema.json`, asserting 100% byte-exact identity or SHA-256 hash match.
- **Negative Regression Testing**:
  - Negative probes must demonstrate that payloads with unrecognized keys (violating `additionalProperties: false`) or invalid types/ranges fail admission before any database transaction is initiated.

### 2.5. Rule 5: Symlink Privilege & Acceptance Gating Policy
- In `internal/host/workspace_binding_authority_test.go`:
  ```go
  if err := createSymlink(target, link); err != nil {
      if isPrivilegeNotHeldError(err) {
          t.Skip("skipping symlink probe: SeCreateSymbolicLinkPrivilege not held by client")
      }
      t.Fatalf("unexpected symlink creation error: %v", err)
  }
  ```
- **Acceptance Gating**:
  - If skipped via `t.Skip()`, the probe does not produce a false-positive `PASS`.
  - Any skip forces `WorkerReport` to declare `status = BLOCKED` or `ready_for_review = false`.
  - Criteria `AC-P04A-07` and `AC-P04A-24` remain `UNVERIFIED`.
  - Subtask P04A implementation is NOT eligible for External Audit Approval without verified execution logs proving probe execution on a capability-qualified Windows host holding `SeCreateSymbolicLinkPrivilege`.

---

## 3. Comparison of Contract Revisions

| Attribute | `CONTRACT-TASK-P04-001-01` (Rev 1) | `CONTRACT-TASK-P04-001-02` (Rev 2 Draft) |
|---|---|---|
| **Contract ID** | `CONTRACT-TASK-P04-001-01` | `CONTRACT-TASK-P04-001-02` |
| **Revision Number** | 1 | 2 |
| **Supersedes Contract ID** | `null` | `CONTRACT-TASK-P04-001-01` |
| **Status** | `RELEASED` (Under Remediation) | `NOT_RELEASED` (Draft Proposed) |
| **Base SHA** | `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2` | `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2` (Preserved) |
| **Allowed Scope File Count** | 39 files | 41 files (+`poller.go`, +`worker_report_schema.json`) |
| **Forbidden Scope** | 7 entries/patterns | 7 entries/patterns (`poller.go` moved to allowed) |
| **Scope Overlap** | 0 files | 0 files |
| **Recovery Authority Wiring** | Ambiguous in Poller seam | Explicit direct owner forwarding in `poller.go:144` |
| **Schema Runtime Packaging** | Undefined relative path | `//go:embed` mirror + byte parity test |
| **Report Intake Signature** | Exported struct + raw JSON | Exclusive raw JSON: `taskID, contractID, attemptID` |
| **Lease Close Outcome** | Silent error ignore | Differentiated: scanner vs pre-effect vs post-effect |
| **Symlink Acceptance Gate** | False PASS via log/return | Explicit `t.Skip()`; skip blocks `ready_for_review` |

---

## 4. Implementation Directives & Sequencing

Upon formal review, approval, and release of `CONTRACT-TASK-P04-001-02` by the External Supervisor:
1. Implementation will proceed exclusively on isolated branch `codex/p04-001` starting from base commit `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`.
2. All modifications will remain strictly bounded to the 41 files in `allowed_scope`.
3. The probe demonstrating `P04A-I2-001` failure must transition to `PASS` with zero global state.
4. Raw report schema admission bypass probe must fail closed.
5. Schema parity verification test must pass.
6. Lease close error injection probes must verify zero resend authorization.
7. Full race suite `go test -race -count=1 ./...` must exit 0.
