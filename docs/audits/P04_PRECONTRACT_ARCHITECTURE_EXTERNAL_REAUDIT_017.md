# External Re-Audit Report 017: Phase P04 Pre-Contract Architecture (Revision 18 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `e3cd8bd54f0c3e39e2c6a5599ff66baa004c7aec`
> **Date**: 2026-09-27
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_18`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 017 evaluated the Phase P04 architecture remediation deliverables submitted in commit `e3cd8bd54f0c3e39e2c6a5599ff66baa004c7aec` (Revision 18 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 18)
2. `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 8)
3. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 18)
4. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 18)
5. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_016.md`
6. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

### Finding Review Status
- **`P04-ARCH-R17-001`**: **SUBSTANTIVELY_CLOSED_WITH_R18_FOLLOWUP**. The elimination of all boolean compliance flags and the replacement of colon-delimited event ID formulas with RFC 8785 JCS domain-separated descriptors in `PROPOSAL-P04-002` were verified. However, follow-up finding `P04-ARCH-R18-001` is recorded regarding incorrect threshold-based derivation of measurement status.
- **`P04-ARCH-R17-002`**: **CLOSED_AT_DESIGN_LEVEL**. References to scratch script `test_probes_r17.py` were eliminated, and readiness matrix entries accurately separate verified static SQL/DDL probes from unexecuted runtime contract tests.
- **`P04-ARCH-R17-003`**: **PARTIALLY_CLOSED**. Most revision labels and diagrams were harmonized, but `docs/18_CURRENT_STATE.md` line 3 retained a stale gate label (`ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_14`).
- **`P04-ARCH-R18-001`**: **OPEN**. `latency_measurement_status` was incorrectly derived from the 3,000 ms threshold instead of the structural provenance of the measurement process.
- **`P04-ARCH-R18-002`**: **OPEN**. `PLAN-P04-EVIDENCE-REVIEW.md` line 81 asserted that commit return duration telemetry is captured in audit event `REVIEW_BUNDLE_GENERATED`, directly contradicting `PROPOSAL-P04-002` and `DRAFT-ADR-018`.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_18_REQUIRED`
- `PROPOSAL_P04_002 = REVISION_8_REQUIRED`
- `ADR_018 = REVISION_18_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_18_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_18`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 17 Findings

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R17-001` | Latency Reconciliation & RFC 8785 JCS Event IDs in PROPOSAL-002 | **SUBSTANTIVELY_CLOSED_WITH_R18_FOLLOWUP** | Delimiter concatenation removed; JCS descriptors specified; boolean compliance flags eliminated; status held strictly as UNVERIFIED. Follow-up R18-001 opened for provenance decoupling. |
| `P04-ARCH-R17-002` | Accurate Evidence Classification in Readiness Matrix | **CLOSED_AT_DESIGN_LEVEL** | Uncommitted scratch script references removed; falsification tests classified as `SQL_MODEL_PROBE_VERIFIED: PASS`, `DESIGN_CONSISTENCY_VERIFIED`, or `PLANNED_CONTRACT_ACCEPTANCE_TEST (UNEXECUTED_PENDING_IMPLEMENTATION)`. |
| `P04-ARCH-R17-003` | Synchronization of Document Metadata and Diagrams | **PARTIALLY_CLOSED** | Proposal-001, ADR-018, and Plan synchronized to Rev 18; Proposal-002 to Rev 8; diagrams updated. Remainder in `18_CURRENT_STATE.md` line 3 header requires closure. |

---

## 3. New Findings (P04-ARCH-R18-001 and P04-ARCH-R18-002)

### `P04-ARCH-R18-001`: Measurement Provenance Decoupled from Latency Threshold
- **Severity**: High
- **Description**: In `PROPOSAL-P04-002 Revision 8` Section 4.2, `latency_measurement_status` was defined such that latency <= 3,000 ms implied `MEASURED_IN_PROCESS` and latency > 3,000 ms implied `RECOVERED_AFTER_RESTART`. This conflates measurement provenance (continuous in-process daemon execution vs restart recovery) with threshold compliance. High load or transient delays in a continuous daemon process must NOT convert provenance to `RECOVERED_AFTER_RESTART`.
- **Remediation Directives**:
  1. `latency_measurement_status` strictly represents measurement provenance:
     - `MEASURED_IN_PROCESS`: Transaction B and ReviewBundle assembly to $T_1$ occur continuously within the same daemon lifetime, regardless of whether `compilation_latency_ms` is <= 3,000 ms or > 3,000 ms.
     - `RECOVERED_AFTER_RESTART`: P04D startup recovery resumes an attempt where Transaction B committed in a prior daemon lifetime, regardless of whether `compilation_latency_ms` is <= 3,000 ms or > 3,000 ms.
  2. Implement an identical 2 × 2 provenance × threshold matrix in `PROPOSAL-P04-002` and `DRAFT-ADR-018`:
     - Continuous process, latency <= 3000: `MEASURED_IN_PROCESS`, `UNVERIFIED`, no diagnostic.
     - Continuous process, latency > 3000: `MEASURED_IN_PROCESS`, `UNVERIFIED`, latency-breach diagnostic logged.
     - Restart recovery, latency <= 3000: `RECOVERED_AFTER_RESTART`, `UNVERIFIED`, no diagnostic.
     - Restart recovery, latency > 3000: `RECOVERED_AFTER_RESTART`, `UNVERIFIED`, latency-breach diagnostic logged.
  3. Ensure that exceeding 3,000 ms never blocks Transaction C, never creates new TaskStates, and never creates integrity holds without independent invariant violations.
  4. Advance `PROPOSAL-P04-002` to Revision 9.

### `P04-ARCH-R18-002`: Contradictory Commit Return Telemetry in PLAN-P04
- **Severity**: Medium
- **Description**: `PLAN-P04-EVIDENCE-REVIEW.md` line 81 stated: *"Telemetry captures commit return duration in audit event REVIEW_BUNDLE_GENERATED"*. This contradicts `PROPOSAL-P04-002` and `DRAFT-ADR-018`, which established that `REVIEW_BUNDLE_GENERATED` emitted in Transaction C does NOT contain `commit_duration_ms` because commit duration cannot be measured before commit returns.
- **Remediation Directives**:
  1. Correct `PLAN-P04-EVIDENCE-REVIEW.md` line 81 to specify that commit return duration is measured post-commit purely as in-process monotonic telemetry and is strictly excluded from `REVIEW_BUNDLE_GENERATED`, database tables, and the audit chain.
  2. Ensure all four normative documents are consistent: post-commit telemetry is best-effort, unbackfilled, not part of the audit chain, and does not affect transaction success.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-002` to Revision 9 incorporating R18-001.
2. Remediate `PROPOSAL-P04-001`, `DRAFT-ADR-018`, and `PLAN-P04-EVIDENCE-REVIEW` to Revision 19 incorporating R18-001, R18-002, and remaining R17-003.
3. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_18`.
4. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
