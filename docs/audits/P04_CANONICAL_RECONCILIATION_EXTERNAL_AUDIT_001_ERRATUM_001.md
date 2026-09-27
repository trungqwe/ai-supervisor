# P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001_ERRATUM_001.md

> **Target Audit**: [P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md](P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md)
> **Authority**: External Supervisor
> **Classification**: Append-Only Record Erratum (Citation Rectification)
> **Status**: APPROVED
> **Date**: 2026-09-27

---

## 1. Context & Purpose

In [P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md](P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md), Section 2, Finding `P04-CANON-R1-001`, the text cited the Win32 physical identity column names as:
`volume_serial_number_hex`, `file_index_high_hex`, `file_index_low_hex`.

This erratum formally rectifies the audit citation to accurately match the canonical DDL defined in accepted [ADR-018](../adr/ADR-018-evidence-review-and-verification-isolation.md).

---

## 2. Erratum Specifications

1. **Canonical Win32 Identity Field Names**:
   - The authoritative Win32 identity columns in table `attempt_workspace_bindings` (Schema v6) are:
     * `volume_serial_hex TEXT NOT NULL CHECK (LENGTH(volume_serial_hex) = 16 AND NOT (volume_serial_hex GLOB '*[^0-9a-f]*'))`
     * `file_id_hex TEXT NOT NULL CHECK (LENGTH(file_id_hex) = 32 AND NOT (file_id_hex GLOB '*[^0-9a-f]*'))`
   - The authoritative linked Gitdir identity columns are:
     * `linked_gitdir_volume_serial_hex TEXT NOT NULL CHECK (LENGTH(linked_gitdir_volume_serial_hex) = 16 AND NOT (linked_gitdir_volume_serial_hex GLOB '*[^0-9a-f]*'))`
     * `linked_gitdir_file_id_hex TEXT NOT NULL CHECK (LENGTH(linked_gitdir_file_id_hex) = 32 AND NOT (linked_gitdir_file_id_hex GLOB '*[^0-9a-f]*'))`

2. **Citation Rectification**:
   - References in Audit Record 001 to `volume_serial_number_hex`, `file_index_high_hex`, and `file_index_low_hex` are recognized as citation inaccuracies.
   - The substantive requirement of Finding `P04-CANON-R1-001` (requiring exact, verbatim adoption of accepted ADR-018 DDL without abbreviation or reinterpretation) and all historical audit verdicts remain unchanged.

3. **Finding Closure Status**:
   - Finding `P04-CANON-R2-002` is formally recorded as `CLOSED_BY_APPEND_ONLY_ERRATUM`.
