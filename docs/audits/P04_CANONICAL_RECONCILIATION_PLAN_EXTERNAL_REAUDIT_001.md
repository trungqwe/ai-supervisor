# P04_CANONICAL_RECONCILIATION_PLAN_EXTERNAL_REAUDIT_001.md

**Audit Target**: `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` (Revision 1), `docs/tasks/DRAFT_TASK_CONTRACT_P04_DOC_RECONCILIATION_ADR_018.md`, `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_DOC_RECONCILIATION_ADR_018.md`
**Audited Commit**: `2cedd63edcc2770ad73db2a060b0e3898c4e2df2`
**Authority**: External Supervisor
**Status**: REVISION_2_REQUIRED (Remediation Submitted in Revision 2 / Candidate Remediation)
**Date**: 2026-09-27
**Active Gate**: `P04_CANONICAL_RECONCILIATION_PLAN_REMEDIATION_2`

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
   External Supervisor evaluated Round 1 remediation deliverables submitted in commit `2cedd63edcc2770ad73db2a060b0e3898c4e2df2`:
   - `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` (Revision 1)
   - `docs/audits/P04_CANONICAL_RECONCILIATION_PLAN_EXTERNAL_AUDIT_001.md`
   - `docs/tasks/DRAFT_TASK_CONTRACT_P04_DOC_RECONCILIATION_ADR_018.md`
   - `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_DOC_RECONCILIATION_ADR_018.md`
   - `AGENTS.md` and `docs/18_CURRENT_STATE.md`

3. **Audit Verdict**:
   - `P04-CRPLAN-R1-001` = `CLOSED` (link portability verified, 0 non-portable link, 0 broken link).
   - `P04-CRPLAN-R1-002` = `CLOSED` (exact enumeration into 16 `MODIFY_ALLOWED` and 14 `VERIFY_ONLY` files/groups).
   - `P04-CRPLAN-R1-003` = `CLOSED` (verbatim `git status --porcelain=v1 -z --untracked-files=all` and `artifacts/<first-two-hex>/<captured_sha256>`).
   - `P04-CRPLAN-R2-001` = `OPEN` / `REMEDIATION_SUBMITTED` (CR-05 omitted full hex validation clause).
   - `P04-CRCONTRACT-R1-001` = `OPEN` / `REMEDIATION_SUBMITTED` (base_sha must reference Commit A containing Revision 2 plan).
   - `P04-CRCONTRACT-R1-002` = `OPEN` / `REMEDIATION_SUBMITTED` (allowed_scope restricted strictly to 16 canonical target files; governance, plan, and contract files moved to forbidden_scope).
   - `P04-CRCONTRACT-R1-003` = `OPEN` / `REMEDIATION_SUBMITTED` (AC-P04-DOC-13 explicit two-way validation: valid fixture passes, invalid fixture rejected on expected constraint).
   - `P04-CRCONTRACT-R1-004` = `OPEN` / `REMEDIATION_SUBMITTED` (Verification Profile Catalog Delta for doc-reconciliation-check added; actual TaskContractValidator execution with positive and 5 negative cases).
   - Overall Verdict: `PLAN_P04_CANONICAL_RECONCILIATION = REVISION_2_REQUIRED`.

4. **Execution Restriction**:
   Canonical specification reconciliation has **NOT** been executed. Zero canonical specification mutations are permitted until the reconciliation scope plan and task contract are formally released.

---

## 2. Audit Findings & Remediation Verification

### Finding P04-CRPLAN-R2-001 — Omission of Full Hex Validation Clause in CR-05
- **Observation**: In item CR-05 of `PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` Revision 1, the `reported_head_sha` column constraint was abbreviated to `CHECK (LENGTH(reported_head_sha) BETWEEN 7 AND 40)`, omitting the mandatory hex character validation clause.
- **Requirement**: Synchronize CR-05 with verbatim SQLite check constraint established in ADR-018:
  ```sql
  CHECK (
    LENGTH(reported_head_sha) BETWEEN 7 AND 40
    AND NOT (reported_head_sha GLOB '*[^0-9a-f]*')
  )
  ```
  Do not abbreviate or omit the lowercase hex character exclusion clause.
- **Status in Revision 2**: `REMEDIATION_SUBMITTED` (Full verbatim constraint incorporated into CR-05).

### Finding P04-CRCONTRACT-R1-001 — Base SHA Invariant in Task Contract
- **Observation**: The candidate task contract previously specified `base_sha = ffb431b1c39b870f5689eb80e830d13f2151df57`, which references an earlier commit before Revision 2 scope plan remediation.
- **Requirement**: Update `base_sha` in both draft and candidate task contracts to the exact commit SHA of Commit A containing the approved Revision 2 scope plan.
- **Status in Commit B**: `REMEDIATION_SUBMITTED`.

### Finding P04-CRCONTRACT-R1-002 — Scope Demarcation: allowed_scope vs forbidden_scope
- **Observation**: `allowed_scope` in the task contract included the scope plan, AGENTS.md, 18_CURRENT_STATE.md, and task contracts.
- **Requirement**:
  1. `allowed_scope` must consist strictly of the 16 approved canonical target files.
  2. Remove governance files (`AGENTS.md`, `docs/18_CURRENT_STATE.md`), plans (`docs/plans/**`), task contracts (`docs/tasks/**`), and audits (`docs/audits/**`) from `allowed_scope`.
  3. Explicitly add them to `forbidden_scope`, preserving worker read access while strictly forbidding worker modifications.
- **Status in Commit B**: `REMEDIATION_SUBMITTED`.

### Finding P04-CRCONTRACT-R1-003 — Two-Way Schema Verification Acceptance Criterion
- **Observation**: AC-P04-DOC-13 stated validation in general terms without explicit negative fixture assertion.
- **Requirement**: Reformulate AC-P04-DOC-13 to mandate explicit two-way validation:
  1. `review-bundle.valid.json` MUST validate successfully.
  2. `review-bundle.invalid.json` MUST be rejected on expected constraint failure, not accidental unparseable JSON.
- **Status in Commit B**: `REMEDIATION_SUBMITTED`.

### Finding P04-CRCONTRACT-R1-004 — Explicit Verification Profile Catalog Delta & Actual Validator Execution
- **Observation**: The profile `doc-reconciliation-check` lacked a formal declarative specification outside JSON and was only verified via Python simulation rather than actual `internal/contract.TaskContractValidator`.
- **Requirement**:
  1. Add a dedicated section "Proposed Verification Profile Catalog Delta" outside JSON specifying `profile_id = doc-reconciliation-check`, `MaxTimeoutSeconds = 60`, `cwd = "."`, `additionalProperties = false`, and exact enum `check_kind` (`git-diff-hygiene`, `portable-relative-link-validation`, `json-schema-draft07-validation`).
  2. Execute actual Go `internal/contract.TaskContractValidator` tests verifying positive candidate pass and 5 required negative rejection cases (unknown profile, check_kind outside enum, extra parameter, cwd traversal, timeout > 60).
- **Status in Commit B**: `REMEDIATION_SUBMITTED`.

---

## 3. Governance Verdict & Active Gate

| Gate / Metric | Value | Verification Source |
| :--- | :--- | :--- |
| **P04 Pre-Contract Architecture** | `EXTERNAL_AUDIT_APPROVED` | External Supervisor Re-Audit 021 |
| **ADR-018 Formal Acceptance** | `GRANTED` | Level 2 Approved ADR |
| **Reconciliation Scope Plan** | `REVISION_2_REQUIRED` | Remediated in Revision 2 (`REVISION_2_SUBMITTED`) |
| **P04 Task Contract Release** | `NOT_RELEASED` | Draft/Candidate prepared; zero execution authorization |
| **P04 Production Code Writing** | `HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT` | Strict governance hold |
| **P05 Code Writing** | `NOT_AUTHORIZED` | Strict governance hold |
| **Active Gate** | `P04_CANONICAL_RECONCILIATION_PLAN_REMEDIATION_2` | Governance checkpoint |
| **Canonical Reconciliation Execution** | `NOT_EXECUTED` | Zero canonical specifications altered in this round |
