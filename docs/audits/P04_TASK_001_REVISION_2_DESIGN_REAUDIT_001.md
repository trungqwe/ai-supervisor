# P04_TASK_001_REVISION_2_DESIGN_REAUDIT_001.md

**Audit Target**: Subtask P04A Task Contract Revision 2 & Seam Remediation Proposal Design Re-Audit (Round 1)
**Audited Documents**:
- `docs/proposals/PROPOSAL-P04-003-p04a-recovery-authority-seam-remediation.md` (Revision 2)
- `docs/tasks/DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md` (Draft Revision 2)
**Audited Governance Commit**: `c74ea2927ab1638830017cce4bfdbf99eb499cd0`
**Implementation Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Audited Implementation SHA**: `b955fc40ef3cc66a5ea456dc4a7c13262c9229d6` (Branch `codex/p04-001`, merge authority WITHHELD)
**Authority**: External Supervisor
**Date**: 2026-09-28
**Verdict**: `PROPOSAL_P04_003 = APPROVED_FOR_CANDIDATE_FORMULATION`, `DRAFT_CONTRACT_P04_001_REVISION_2 = READY_FOR_RELEASE_AUDIT`
**Contract Status**: `TASK_P04A_REVISION_2 = NOT_RELEASED`
**Active Gate**: `TASK_P04A_REVISION_2_RELEASE_AUDIT`
**P04 Code Authority**: `HELD_PENDING_P04A_CONTRACT_REVISION`
**Merge Authority**: `WITHHELD`

---

## 1. Executive Summary & Audit Determination

External Supervisor performed an independent re-audit of the remediation governance documents submitted at commit `c74ea2927ab1638830017cce4bfdbf99eb499cd0`, specifically reviewing the resolution of Design Audit 001 findings (`P04A-REV2-D1-001` through `P04A-REV2-D1-005`) in `PROPOSAL-P04-003` (Revision 2) and `DRAFT_TASK_CONTRACT_P04_001_REVISION_2.md`.

### 1.1. Finding Verification & Closure Matrix

| Finding ID | Title | Re-Audit Assessment | Status |
|---|---|---|---|
| `P04A-REV2-D1-001` | Runtime JSON Schema Packaging Undefined | Proposal and contract adopt embedded canonical schema mirror `internal/store/worker_report_schema.json` via `//go:embed`, add file to `allowed_scope` (total 41 files), mandate byte parity verification test, and prohibit parallel hand-written validators. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-002` | Symlink Skip Cannot Satisfy Acceptance | Proposal and contract mandate that unheld symlink privilege calling `t.Skip()` explicitly leaves criteria `AC-P04A-07` and `AC-P04A-24` `UNVERIFIED`, forces `WorkerReport` to declare `status = BLOCKED` or `ready_for_review = false`, and prevents External Audit Approval without verified execution logs on a capability-qualified Windows host holding `SeCreateSymbolicLinkPrivilege`. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-003` | Intake API Signature Drift in Proposal | Proposal Section 2.4 reconciles `Store.IngestWorkerReportRaw` signature to exact lineage seam `(ctx, taskID, contractID, attemptID, rawReportJSON, evidence, actor, at)`, eliminates unapproved `operationID, sessionID, generation` drift, and requires struct helper to be private. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-004` | Lease Close Outcome Semantics Underspecified | Proposal Section 2.3 and contract constraints strictly partition `lease.Close()` error semantics across three execution boundaries: (A) scanner returns non-nil error, `Complete=false`, 0 AO effects; (B) coordinator pre-wire forbids `/send` (send call count = 0) and triggers Variant B; (C) coordinator post-wire logs/returns error and strictly never permits resend. | `CLOSED_AT_DESIGN_LEVEL` |
| `P04A-REV2-D1-005` | Governance Drift & Finding Classification | `docs/18_CURRENT_STATE.md` corrected to cite `forbidden_scope = 7 entries/patterns`. Reporting classifications aligned with canonical finding definitions. | `CLOSED` |

---

## 2. Independent Lineage and Contract Validation

1. **Schema Validation**:
   - `CONTRACT-TASK-P04-001-02` JSON block conforms strictly to Draft-07 metaschema `docs/schemas/task-contract.schema.json`.
   - `contract_id`: `CONTRACT-TASK-P04-001-02`
   - `revision_number`: `2`
   - `supersedes_contract_id`: `CONTRACT-TASK-P04-001-01`
   - `base_sha`: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
2. **Scope Analysis**:
   - `allowed_scope`: exactly 41 unique files (+`internal/recovery/poller.go`, +`internal/store/worker_report_schema.json`).
   - `forbidden_scope`: exactly 7 entries/patterns.
   - Scope overlap between allowed and forbidden: **0 files**.
3. **Verification Policy & Probes**:
   - 6 verification requests using profile `go-test-p04-001`.
   - Independent verification via `TaskContractValidator.ValidateRaw` against previous revision contract `CONTRACT-TASK-P04-001-01` passed cleanly with exit code `0`.
   - 6 mandatory negative probes (Unknown Profile, Timeout Boundary, Flag Injection, Shell Ingestion, Cwd Traversal, Package Path Escaping) all rejected cleanly.
4. **Implementation Hygiene**:
   - Working tree on `main` remains clean.
   - Zero Go diff on `main` (`git diff` shows 0 Go code changes).
   - Implementation branch `codex/p04-001` remains held; merge authority remains withheld.

---

## 3. Direction on Runtime Schema Validation Seam

Pursuant to External Supervisor directive:
1. `internal/store` must strictly invoke `internal/contract.CompileSchema` and `internal/contract.ParseAndValidateRaw` for runtime worker report schema validation.
2. Direct imports of `github.com/google/jsonschema-go` within `internal/store` are prohibited.
3. No dual or disjunctive validator options are permitted.
4. Schema compilation may be cached via `sync.Once`; initialization or compilation failure must fail closed immediately.
5. The embedded mirror `internal/store/worker_report_schema.json` must remain byte-exact with `docs/schemas/worker-report.schema.json`.

---

## 4. Re-Audit Determination & Gate Transition

- `PROPOSAL_P04_003 = APPROVED_FOR_CANDIDATE_FORMULATION`
- `DRAFT_TASK_CONTRACT_P04_001_REVISION_2 = READY_FOR_RELEASE_AUDIT`
- `CANDIDATE_TASK_CONTRACT_P04_001_REVISION_2` formulation is **AUTHORIZED**.
- `TASK_P04A_REVISION_2 = NOT_RELEASED` (Release authorization remains withheld pending final candidate release audit).
- `P04_CODE = HELD_PENDING_P04A_CONTRACT_REVISION`.
- `MERGE_AUTHORITY = WITHHELD`.
- `ACTIVE_GATE = TASK_P04A_REVISION_2_RELEASE_AUDIT`.
