# P04_CANONICAL_RECONCILIATION_PLAN_EXTERNAL_AUDIT_001.md

**Audit Target**: `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` (Revision 0)
**Audited Commit**: `ffb431b1c39b870f5689eb80e830d13f2151df57`
**Authority**: External Supervisor
**Status**: REVISION_1_REQUIRED (Remediation Submitted in Revision 1)
**Date**: 2026-09-27
**Active Gate**: `P04_CANONICAL_RECONCILIATION_PLAN_REMEDIATION_1`

---

## 1. Executive Summary & Architecture Status

1. **Pre-Contract Architecture Invariant**:
   Phase P04 pre-contract architecture remains firmly **EXTERNAL_AUDIT_APPROVED** pursuant to External Supervisor Re-Audit 021 (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_021.md`, commit `9c53ac3ee82a7035d458b1abe15f7a6bc315987d`):
   - `PROPOSAL_P04_001 = EXTERNAL_APPROVED` (Revision 22)
   - `PROPOSAL_P04_002 = APPROVED_AS_DESIGN_BASELINE` (Revision 9, byte-for-byte unchanged)
   - `ADR_018 = EXTERNAL_APPROVED` & `ADR_018_ACCEPTANCE = GRANTED` (`docs/adr/ADR-018-evidence-review-and-verification-isolation.md`)
   - `PLAN_P04_EVIDENCE_REVIEW = EXTERNAL_APPROVED` (Revision 22)
   - All six critical architecture design blockers (`DESIGN_BLOCKER_P04_*`) remain `CLOSED_AT_DESIGN_LEVEL`.
   - Zero modifications to accepted ADRs, approved proposals, or past audits are permitted.

2. **Audit Subject**:
   External Supervisor evaluated initial Revision 0 of `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` introduced at commit `ffb431b1c39b870f5689eb80e830d13f2151df57`.

3. **Audit Verdict**:
   `PLAN_P04_CANONICAL_RECONCILIATION = REVISION_1_REQUIRED`.
   Three governance and specification hygiene findings were recorded (`P04-CRPLAN-R1-001`, `P04-CRPLAN-R1-002`, `P04-CRPLAN-R1-003`).

4. **Execution Restriction**:
   Canonical specification reconciliation has **NOT** been executed. Zero canonical specification mutations are permitted until this reconciliation scope plan is formally approved and a dedicated reconciliation task contract is released.

---

## 2. Audit Findings & Remediation Verification

### Finding P04-CRPLAN-R1-001 — Non-Portable Absolute File URI in Scope Plan Authority
- **Observation**: Line 3 of `PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` contained an absolute Windows drive URI referencing ADR-018 with prefix `file:///d:/TU_CODE/ai-supervisor/...`. This violates link portability across repository clones, host platforms, and remote CI runners.
- **Requirement**: Replace absolute file URI with portable relative markdown link (`../adr/ADR-018-evidence-review-and-verification-isolation.md`). Update link validation test scripts to strictly reject any `file://`, Windows drive letters (`C:`, `D:`), or external references within documentation links.
- **Status in Revision 1**: `REMEDIATION_SUBMITTED` (Relative portable link established; link scanner verifies zero non-portable links).

### Finding P04-CRPLAN-R1-002 — Ambiguous Scope Enumeration & Generic Schema References
- **Observation**: Section 3 of Revision 0 referred ambiguously to "14 target files" and used a generic phrasing "and evidence JSON schema specifications" without itemizing specific files or separating allowed modifications from verify-only documents.
- **Requirement**:
  1. Eliminate the ambiguous "14 target files" phrase and all wildcards.
  2. Enumerate every file into two explicit, non-overlapping groups:
     - Group A: `MODIFY_ALLOWED`
     - Group B: `VERIFY_ONLY`
  3. `docs/schemas/review-bundle.schema.json` is authorized under `MODIFY_ALLOWED` pursuant to ADR-018.
  4. Specific review-bundle examples (`docs/schemas/examples/review-bundle.valid.json` and `docs/schemas/examples/review-bundle.invalid.json`) are authorized under `MODIFY_ALLOWED` to maintain schema synchronization.
  5. `docs/schemas/worker-report.schema.json` and its examples are strictly `VERIFY_ONLY` because accepted ADR-018 introduces zero deltas to the worker report schema.
  6. `docs/schemas/task-contract.schema.json` and its examples are strictly `VERIFY_ONLY` pursuant to canonical task contract immutability.
  7. Do not create new schema files solely to fulfill arbitrary count requirements.
- **Status in Revision 1**: `REMEDIATION_SUBMITTED` (Exhaustive categorization established: exactly 16 files in `MODIFY_ALLOWED`, 14 items/categories in `VERIFY_ONLY`).

### Finding P04-CRPLAN-R1-003 — Shorthand Literals & Command Drift against ADR-018 / Proposal-P04-002
- **Observation**:
  - In item CR-05, the clean worktree verification command omitted `--untracked-files=all`, stating `git status --porcelain=v1 -z` instead of `git status --porcelain=v1 -z --untracked-files=all`.
  - In item CR-08, the CAS artifact store path was stated as shorthand `artifacts/<hex>/<sha256>` rather than the authoritative pattern `artifacts/<first-two-hex>/<captured_sha256>`.
- **Requirement**: Restore verbatim literals, exact commands, DDL check clauses, and hashing rules established in accepted ADR-018 and approved PROPOSAL-P04-002 Revision 9 across all items CR-01 through CR-12.
- **Status in Revision 1**: `REMEDIATION_SUBMITTED` (Verbatim commands and artifact path literals synchronized across all items CR-01..CR-12).

---

## 3. Governance Verdict & Active Gate

| Gate / Metric | Value | Verification Source |
| :--- | :--- | :--- |
| **P04 Pre-Contract Architecture** | `EXTERNAL_AUDIT_APPROVED` | External Supervisor Re-Audit 021 |
| **ADR-018 Formal Acceptance** | `GRANTED` | Level 2 Approved ADR |
| **Reconciliation Scope Plan** | `REVISION_1_REQUIRED` | Remediated in Revision 1 (`REVISION_1_SUBMITTED`) |
| **P04 Task Contract Release** | `NOT_RELEASED` | Draft/Candidate prepared; zero execution authorization |
| **P04 Production Code Writing** | `HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT` | Strict governance hold |
| **P05 Code Writing** | `NOT_AUTHORIZED` | Strict governance hold |
| **Active Gate** | `P04_CANONICAL_RECONCILIATION_PLAN_REMEDIATION_1` | Governance checkpoint |
| **Canonical Reconciliation Execution** | `NOT_EXECUTED` | Zero canonical specifications altered in this round |
