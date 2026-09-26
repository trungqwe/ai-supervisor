# External Re-Audit Report 016: Phase P04 Pre-Contract Architecture (Revision 17 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `d6fe53c396befd2eacd87787e58bd9b86cc62196`
> **Date**: 2026-09-27
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_17`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 016 evaluated the Phase P04 architecture remediation deliverables submitted in commit `d6fe53c396befd2eacd87787e58bd9b86cc62196` (Revision 17 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 17)
2. `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 7)
3. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 17)
4. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 17)
5. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_015.md`
6. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

Finding `P04-ARCH-R16-001` (persistence ownership alignment and contract sequencing: relocation of `review_integrity_holds` to Schema v6 owned by Subtask P04A, demarcation of mutation authority, elimination of ambiguous blanket orchestrator phrasing, and contract sequencing P04A -> P04B -> P04C -> P04D) is verified **CLOSED_AT_DESIGN_LEVEL**.

However, three architectural, evidence, and metadata issues were identified across the deliverables, requiring Revision 18 remediation:
- **`P04-ARCH-R17-001`**: Incomplete canonical latency reconciliation and delimiter-concatenated audit event IDs in `PROPOSAL-P04-002 Revision 7`. The proposal still retained `nfr008_met` alongside `nfr008_compliance_status = 'UNVERIFIED'`, and computed event IDs via string concatenation (`attempt_id || ":" || contract_id || ":" || reason || ":" || sanitized_input_fingerprint`), violating RFC 8785 JCS canonicalization standards.
- **`P04-ARCH-R17-002`**: Evidence classification anomaly. Non-repository scratch script `test_probes_r17.py` was cited as verification evidence, and behavior probes for unimplemented P04 components were prematurely marked PASS. Pre-contract readiness must distinguish verified SQL DDL / static consistency from unexecuted runtime behavior (`PLANNED_CONTRACT_ACCEPTANCE_TEST` / `UNEXECUTED_PENDING_IMPLEMENTATION`).
- **`P04-ARCH-R17-003`**: Obsolete metadata and diagram revision labels. Multiple diagrams and header metadata fields retained stale labels (Revision 12, Revision 14) that did not reflect the active document suite.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_17_REQUIRED`
- `PROPOSAL_P04_002 = REVISION_7_REQUIRED`
- `ADR_018 = REVISION_17_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_17_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_17`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 16 Finding (P04-ARCH-R16-001)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R16-001` | Persistence Ownership Alignment & Contract Sequencing | **CLOSED_AT_DESIGN_LEVEL** | Fully resolved: `review_integrity_holds` DDL, index, and triggers are located in Schema v6 owned by P04A; Schema v9 reuses the table without recreation; P04B/P04C are pure in-memory with zero writes; each audit event is appended by its authoritative transaction owner; and P04A -> P04B -> P04C -> P04D sequencing is enforced. |

---

## 3. New Findings (P04-ARCH-R17-001 through P04-ARCH-R17-003)

### `P04-ARCH-R17-001`: Incomplete Latency Reconciliation & Delimiter Concatenation in PROPOSAL-P04-002
- **Severity**: High
- **Description**:
  1. `PROPOSAL-P04-002 Revision 7` retained references to `nfr008_met` across its truth table, pipeline description, proposed canonical text, and crash/recovery semantics, conflicting with `nfr008_compliance_status = 'UNVERIFIED'` in `review_bundles`.
  2. `PROPOSAL-P04-002` Section 6 defined `event_id` derivation via colon string concatenation (`attempt_id || ":" || contract_id || ":" || reason || ":" || sanitized_input_fingerprint`), directly conflicting with ADR-018's ban on delimiter concatenation.
- **Remediation Directives**:
  1. Eliminate all references to `nfr008_met`; use strictly `nfr008_compliance_status = 'UNVERIFIED'`.
  2. Treat `compilation_latency_ms` solely as an in-process assembly diagnostic measurement. The condition `compilation_latency_ms <= 3000` is a measurable target, but does NOT make a ReviewBundle canonical compliance evidence.
  3. If compilation latency exceeds 3000 ms: commit Transaction C, keep `nfr008_compliance_status = 'UNVERIFIED'`, record diagnostic, and do NOT deadlock or hold the task at `EVIDENCE_READY`.
  4. Replace delimiter concatenation with RFC 8785 JCS domain-separated descriptors:
     - `review_bundle_generated_event_descriptor`: `version`, `kind`, `event_type`, `pair_id`, `task_id`, `contract_id`, `attempt_id`, `bundle_id`, `bundle_hash`.
     - `review_bundle_rejection_event_descriptor` (non-hold): `version`, `kind`, `event_type`, `pair_id`, `task_id`, `contract_id`, `attempt_id`, `reason`, `sanitized_input_fingerprint`.
     - Rejection creating an integrity hold continues to use Descriptor A then Descriptor B as defined in ADR-018.
     - Prohibit delimiter concatenation, UUID fallback, and optional-field ambiguity.
     - Duplicate `event_id` readback must compare canonical semantic fields; mismatch fails closed as integrity conflict.
  5. Advance `PROPOSAL-P04-002` to Revision 8.

### `P04-ARCH-R17-002`: Evidence Classification Anomaly in Readiness Matrix & Report
- **Severity**: Medium
- **Description**: The Revision 17 verification report cited a non-repository scratch script (`test_probes_r17.py`) and reported behavior probes for unimplemented P04 components as PASS. Because Phase P04 code is held pending architecture approval, runtime behavior cannot be verified as passed.
- **Remediation Directives**:
  1. Clearly distinguish between:
     - Design consistency verified (`DESIGN_CONSISTENCY_VERIFIED`);
     - SQL schema DDL compiled and verified via in-repo tooling (`SQL_MODEL_PROBE_VERIFIED: PASS`);
     - Runtime behavior unverified (`UNEXECUTED_PENDING_IMPLEMENTATION` / `PLANNED_CONTRACT_ACCEPTANCE_TEST`).
  2. In the Final Architecture Readiness Matrix, mark falsification tests requiring P04A–P04D code execution as `PLANNED_CONTRACT_ACCEPTANCE_TEST (UNEXECUTED_PENDING_IMPLEMENTATION)`.
  3. Prohibit citing non-repository scratch files as official verification evidence.

### `P04-ARCH-R17-003`: Obsolete Revision Labels in Metadata and Diagrams
- **Severity**: Low
- **Description**: Stale revision labels (Revision 12, Revision 14) persisted in `PROPOSAL-P04-001` metadata line 4, `PLAN-P04-EVIDENCE-REVIEW` Mermaid flowchart lines 25–28, and `AGENTS.md` deliverable summary line 122.
- **Remediation Directives**:
  1. Synchronize all active document revision metadata to Revision 18 (`PROPOSAL-P04-001`, `DRAFT-ADR-018`, `PLAN-P04-EVIDENCE-REVIEW`) and Revision 8 (`PROPOSAL-P04-002`).
  2. Update Mermaid flowcharts and governance summaries to reflect current revision numbers.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-002` to Revision 8 incorporating R17-001.
2. Remediate `PROPOSAL-P04-001`, `DRAFT-ADR-018`, and `PLAN-P04-EVIDENCE-REVIEW` to Revision 18 incorporating R17-001, R17-002, and R17-003.
3. Update the Final Architecture Readiness Matrix across deliverables to classify falsification tests accurately as `PLANNED_CONTRACT_ACCEPTANCE_TEST (UNEXECUTED_PENDING_IMPLEMENTATION)`.
4. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_17`.
5. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
