# P04_CANONICAL_RECONCILIATION_PLAN_EXTERNAL_AUDIT_ERRATUM_001.md

**Target Document**: `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` (Revision 2)
**Authority**: External Supervisor
**Date**: 2026-09-27
**Status**: APPROVED_ERRATUM
**Classification**: CANONICAL_PLAN_CORRECTION

---

## 1. Context & Rationale

During the external audit of canonical specification reconciliation implementation (Audit Record `docs/audits/P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md`), a critical architectural discrepancy was identified in item CR-08 regarding ReviewBundle schema boundaries:

1. **Preimage Paradox Elimination**: Placing `bundle_hash` inside the payload schema creates a self-referential paradox when computing the RFC 8785 JSON Canonicalization Scheme (JCS) hash. The payload cannot hash itself while containing its own hash.
2. **Persistence Envelope Separation**: Properties determined post-synthesis or during SQLite row commitment (`bundle_assembled_at_epoch_ms`, `compilation_latency_ms`, `latency_measurement_status`, `nfr008_compliance_status`) belong strictly to the database envelope/row (`review_bundles` table) and transaction audit event, NOT within the immutable pre-hash payload schema.
3. **Mandatory Contract & Evidence Linkage**: The payload schema must strictly declare and require `evidence_set_id` and `contract_id` to maintain unambiguous cryptographic and domain traceability.

---

## 2. Erratum Directives for CR-08

In `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md`, Section 4, Item CR-08 is amended as follows:

### Amended Specification for CR-08:
- **ReviewBundlePayload**: `docs/schemas/review-bundle.schema.json` validates strictly the canonical pre-hash payload (`ReviewBundlePayload`).
- **Required Properties in Payload Schema**:
  * `schema_version`
  * `bundle_id`
  * `task_id`
  * `contract_id`
  * `attempt_id`
  * `evidence_set_id`
  * `evidence_finalized_at_epoch_ms` (T0 timestamp recorded when evidence collection finalized, prior to bundle synthesis)
  * `worker_report`
  * `actual_head_sha`
  * `reported_head_sha`
  * `git_diff`
  * `scope_audit`
  * `verification_results`
- **Strictly Excluded from Payload Schema**:
  * `bundle_hash` (calculated as SHA-256 of JCS-canonicalized `ReviewBundlePayload`)
  * `bundle_assembled_at_epoch_ms` (T1 timestamp recorded at database row insertion)
  * `compilation_latency_ms` (assembly diagnostic interval computed as `T1 - T0`)
  * `latency_measurement_status`
  * `nfr008_compliance_status`
- **ReviewBundleRecord (Database Envelope)**:
  The `review_bundles` SQLite table persists `bundle_id`, `task_id`, `attempt_id`, `evidence_set_id`, `bundle_hash`, `payload_json`, `evidence_finalized_at_epoch_ms`, `bundle_assembled_at_epoch_ms`, `compilation_latency_ms`, `latency_measurement_status`, and `nfr008_compliance_status`.
- **Validation Fixtures**:
  * `docs/schemas/examples/review-bundle.valid.json`: Validated against `review-bundle.schema.json` without `bundle_hash` or post-hash latency metadata.
  * `docs/schemas/examples/review-bundle.invalid.json`: Valid JSON syntax violating a canonical payload constraint (e.g., missing mandatory `contract_id`).

---

## 3. Plan Status Update

Scope Plan `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` retains its Revision 2 lineage and is marked:
`Status: EXTERNAL_AUDIT_APPROVED_WITH_ERRATUM_001`.
