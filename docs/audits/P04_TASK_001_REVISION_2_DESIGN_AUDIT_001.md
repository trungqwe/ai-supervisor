# P04_TASK_001_REVISION_2_DESIGN_AUDIT_001.md

**Audit Target**: Subtask P04A Task Contract Revision 2 & Remediation Proposal Design Audit (Round 1)
**Audited Documents**:
- `docs/proposals/PROPOSAL-P04-003-p04a-recovery-authority-seam-remediation.md` (Revision 1)
- `docs/tasks/DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md` (Draft Revision 2)
**Audited Governance Commit**: `f066e501182ffb6a1d18d07512e41b85ae036502`
**Implementation Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Audited Implementation SHA**: `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6` (Branch `codex/p04-001`, merge authority WITHHELD)
**Authority**: External Supervisor
**Date**: 2026-09-28
**Verdict**: `REVISION_REQUIRED` (`PROPOSAL_P04_003 = REVISION_1_REQUIRED`, `DRAFT_CONTRACT_P04_001_REVISION_2 = REVISION_REQUIRED`)
**Active Gate**: `TASK_P04A_IMPLEMENTATION_REMEDIATION`

---

## 1. Executive Summary & Audit Determination

External Supervisor performed an independent design audit of the governance proposal `PROPOSAL-P04-003` (Revision 1) and the draft Task Contract `CONTRACT-TASK-P04-001-02` committed at `f066e501182ffb6a1d18d07512e41b85ae036502`.

### 1.1. Verified Compliance Strengths
1. **Governance Hygiene**: `HEAD = origin/main = f066e501...`, working tree clean, diff strictly restricted to 5 governance/documentation files, zero Go diff on `main`.
2. **Formatting & Lineage**: `git diff --check` exited `0`; Draft-07 JSON Schema validation passed; `TaskContractValidator.ValidateRaw` against policy catalog `go-test-p04-001` and contract lineage checks passed.
3. **Scope Control**: Scoped seam delta (+`internal/recovery/poller.go`, 40 files total) verified with 0 overlap against `forbidden_scope`.
4. **Architectural Direction**: Elimination of `storeAuthorities` global mutable registry in `scanner.go` and direct forwarding of `p.Owner.Authority` in `poller.go` approved in direction.

### 1.2. Determination
However, 5 findings were identified requiring remediation before candidate formulation or contract release can be authorized.
- `TASK_P04A_REVISION_2 = DRAFT_REVISION_REQUIRED`
- `P04_CODE = HELD_PENDING_P04A_CONTRACT_REVISION`
- `Merge Authority` remains strictly **WITHHELD**.

---

## 2. Audit Findings & Required Remediation

### Finding `P04A-REV2-D1-001`: Runtime JSON Schema Packaging Undefined
- **Severity**: High
- **Status**: `OPEN_PENDING_REVISION_2`
- **Description**: `PROPOSAL-P04-003` required validation of raw worker report payloads against `docs/schemas/worker-report.schema.json`. However, supervisor binaries cannot assume execution from within a Git repository clone or rely on filesystem relative paths to `docs/schemas/`. Go cannot use `go:embed` on paths outside the package directory tree. The draft did not specify how schema bytes are packaged at runtime or assert byte parity with canonical schemas.
- **Required Remediation**:
  1. Adopt embedded schema mirror: add `internal/store/worker_report_schema.json` as an embedded byte-exact mirror of `docs/schemas/worker-report.schema.json` using `//go:embed` and repository JSON Schema compilation.
  2. Add `internal/store/worker_report_schema.json` to `allowed_scope` in `CONTRACT-TASK-P04-001-02` (total `allowed_scope` = 41 files).
  3. Require parity verification tests asserting byte-exact or SHA-256 equivalence between the embedded mirror and `docs/schemas/worker-report.schema.json`.
  4. Prohibit any parallel hand-written validator from operating as an independent authority.

---

### Finding `P04A-REV2-D1-002`: Symlink Skip Cannot Satisfy Acceptance
- **Severity**: High
- **Status**: `OPEN_PENDING_REVISION_2`
- **Description**: In `DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md`, `AC-P04A-07` and `AC-P04A-24` specified that unheld Windows privileges cause `t.Skip()`. While `t.Skip()` eliminates false-positive `PASS` logs, a skipped test still yields an overall test suite exit code `0`. If symlink probes are skipped, the symlink substitution dimension of the Win32 binding authority remains unproven, yet the worker could declare the task ready for review.
- **Required Remediation**:
  1. Explicitly mandate that if the symlink probe is skipped due to unheld privileges, `WorkerReport` must record `status = BLOCKED` or `ready_for_review = false`, and criteria `AC-P04A-07` and `AC-P04A-24` must remain `UNVERIFIED`.
  2. Implementation cannot receive External Audit Approval without verified execution logs proving symlink probe execution on a capability-qualified Windows host holding `SeCreateSymbolicLinkPrivilege`.
  3. Add `test_log_symlink_probe_executed_and_passed_on_capability_qualified_windows_host` to `required_evidence`.
  4. Add a stop condition blocking `ready_for_review = true` when symlink probe is skipped or unverified.

---

### Finding `P04A-REV2-D1-003`: Intake API Signature Drift in Proposal
- **Severity**: Medium
- **Status**: `OPEN_PENDING_REVISION_2`
- **Description**: `PROPOSAL-P04-003` line 80 formulated the public intake signature as `Store.IngestWorkerReportRaw(ctx, operationID, attemptID, contractID, sessionID, generation, rawReportJSON, evidence, actor, at)`. The active implementation seam in Subtask P04A uses `taskID, contractID, attemptID, rawReportJSON, evidence, actor, at`. Injecting `operationID`, `sessionID`, and `generation` into the report intake transaction introduces unapproved interface drift without requirement or ADR backing.
- **Required Remediation**:
  1. Reconcile the proposal signature to match the active lineage seam:
     ```go
     Store.IngestWorkerReportRaw(ctx context.Context, taskID, contractID, attemptID string, rawReportJSON []byte, evidence domain.GitEvidenceResult, actor string, at time.Time) error
     ```
  2. Specify that any struct-based helper `IngestWorkerReport` must be converted to an internal private helper (`ingestWorkerReportDecoded` or package-private helper) invoked exclusively after successful raw JSON schema validation.

---

### Finding `P04A-REV2-D1-004`: Lease Close Outcome Semantics Underspecified
- **Severity**: High
- **Status**: `OPEN_PENDING_REVISION_2`
- **Description**: Proposal Rule 3 vaguely stated "capture and propagate any error returned by `lease.Close()`". This failed to define distinct outcomes across different operational boundaries (pre-wire vs post-wire, scanner vs coordinator), creating a risk that handle close errors after `/send` could be misconstrued as permission to resend.
- **Required Remediation**:
  1. Clearly separate lease close outcome semantics:
     - **Recovery Scanner**: Close failure returns a non-nil error from `Run`, reports `Complete: false`, preserves already-committed DB transactions without rollback, and issues zero AO wire effects.
     - **Coordinator Pre-Wire Effect**: Any failure in lease acquire, revalidate, or close prior to `/send` strictly forbids wire transmission (`send call count = 0`), preserves task `DISPATCHED` and attempt open, and triggers diagnostic Variant B.
     - **Coordinator Post-Wire Effect (after `SEND_REQUESTED` or `SEND_CONFIRMED`)**: A failure of `lease.Close()` after wire transmission or durable send commitment is logged and returned as an error, but strictly NEVER grants permission to resend. Durable dispatch outcome remains authoritative, and the caller must re-read durable state.
  2. Require regression tests with simulated lease close errors for both pre-effect and post-effect paths.

---

### Finding `P04A-REV2-D1-005`: Governance Drift & Finding Classification
- **Severity**: Low
- **Status**: `OPEN_PENDING_REVISION_2`
- **Description**: `docs/18_CURRENT_STATE.md` incorrectly cited `forbidden_scope 46 files` instead of the canonical `7 entries/patterns`. Additionally, reporting transposed finding references: `P04A-I1-006` designates WorkerReport schema and dirty intake lineage, `P04A-I1-007` designates host filesystem/substitution coverage, while `lease.Close()` belongs under `P04A-I1-004` / `P04A-I2-001`.
- **Required Remediation**:
  1. Correct `docs/18_CURRENT_STATE.md` to reference `7 entries/patterns`.
  2. Align all subsequent reports and governance entries with canonical finding definitions. Historical audit records remain immutable.

---

## 3. Governance Status & Action Directive

- **Verdict**: `REVISION_REQUIRED`
- **Status of Draft Revision 2**: `DRAFT_REVISION_REQUIRED` (NOT_RELEASED)
- **Active Gate**: `TASK_P04A_IMPLEMENTATION_REMEDIATION`
- **P04 Code Authority**: `HELD_PENDING_P04A_CONTRACT_REVISION`
- **Directives for Next Revision**:
  1. Update `PROPOSAL-P04-003` to Revision 2 incorporating embedded schema packaging, corrected intake signature, symlink verification gating, and differentiated lease close semantics.
  2. Update `DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md` to expand `allowed_scope` to exactly 41 files (+`internal/store/worker_report_schema.json`), add required evidence, stop conditions, and update acceptance criteria.
  3. Validate contract JSON with Draft-07 schema and `TaskContractValidator.ValidateRaw`.
  4. Ensure zero Go code modifications on `main`.
