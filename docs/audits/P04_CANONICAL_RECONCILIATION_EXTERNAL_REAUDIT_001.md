# P04_CANONICAL_RECONCILIATION_EXTERNAL_REAUDIT_001.md

**Audit Target**: Phase P04 Canonical Specification Reconciliation Remediation (Round 1)
**Audited Branch**: `codex/p04-doc-reconciliation`
**Audited Commit**: `3018bbd32a26469bb8393921ae2fb8448e2a635c`
**Base SHA**: `b174ef3e88409f82b10afe0fcce88ba218530863`
**Governance Baseline**: `ed1920728ccd91750a09da7cb6954343925b1f93`
**Authority**: External Supervisor
**Verdict**: `P04_CANONICAL_RECONCILIATION = REVISION_REQUIRED`
**Date**: 2026-09-27
**Active Gate**: `P04_CANONICAL_RECONCILIATION_REMEDIATION_2`

---

## 1. Executive Summary & Audit Determination

1. **Re-Audit Evaluation**:
   External Supervisor evaluated remediation commit `3018bbd32a26469bb8393921ae2fb8448e2a635c` on branch `codex/p04-doc-reconciliation` against the four findings recorded in [P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md](P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md).

2. **Resolution of Round 1 Findings**:
   - `P04-CANON-R1-001` (DDL Drift): **CLOSED**. Automated SQL statement probe confirmed all 31 P04 SQL objects (7 tables, 3 indexes, 21 triggers) match byte-for-byte between `docs/05_DOMAIN_MODEL.md` and accepted ADR-018.
   - `P04-CANON-R1-002` (ReviewBundle Schema Missing Required Fields): **CLOSED**. Draft-07 JSON Schema validation confirmed `evidence_set_id` and `contract_id` are defined and required in `docs/schemas/review-bundle.schema.json`. Valid fixture passes; invalid fixture fails with expected constraint violation.
   - `P04-CANON-R1-003` (Self-Referencing `bundle_hash` & Mixed Metadata): **CLOSED**. Pure architectural separation achieved: `ReviewBundlePayload` (pre-hash preimage schema validated by `review-bundle.schema.json`) is decoupled from `ReviewBundleRecord` (database row in `review_bundles` table storing post-hash latency metadata and `bundle_hash`).
   - `P04-CANON-R1-004` (Hold/Audit Literals and Transaction Sequencing in docs/14): **CLOSED**. Pre-send binding guard failure correctly sequences diagnostic transaction (recording `REVIEW_INTEGRITY_CONFLICT` with `conflict_source = 'WORKSPACE_BINDING_GUARD'`, `hold_reason = 'INVARIANT_MISMATCH'`), preserving `DISPATCHED`/`DISPATCH_BOUND` prior to separate D12 terminalization by verified operator. Dirty intake emits `EVIDENCE_COLLECTION_FAILED` and `DIRTY_WORKTREE_DETECTED` while preserving `RUNNING`. Zero forbidden literals remain.

3. **New Audit Findings (Round 2)**:
   - `P04-CANON-R2-001`: Ownership discrepancy between Subtask P04A and Subtask P04B regarding Git inspection vs. intake persistence.
   - `P04-CANON-R2-002`: Column naming citation erratum in Audit Record 001, rectified via append-only erratum.

4. **Audit Verdict**:
   - `P04_CANONICAL_RECONCILIATION = REVISION_REQUIRED`.
   - Active Gate: `P04_CANONICAL_RECONCILIATION_REMEDIATION_2`.
   - Implementation branch `codex/p04-doc-reconciliation` **KHÔNG ĐƯỢC MERGE** vào `main`.
   - Subtask P04A Task Contract **CHƯA ĐƯỢC PHÁT HÀNH** (`TASK_P04A_TASK_CONTRACT = NOT_RELEASED`).
   - Tuyệt đối không viết code Go hay migrations (`P04_CODE = HELD`).
   - Tuyệt đối không sửa accepted ADR-018, approved proposals, released contract, hoặc audit lịch sử.

---

## 2. Status of Audit Findings

### Round 1 Findings

| Finding ID | Title | Status | Audit Verification Evidence |
|---|---|---|---|
| `P04-CANON-R1-001` | DDL drift trong `docs/05_DOMAIN_MODEL.md` | **CLOSED** | Probe verified 31/31 SQL objects match byte-for-byte with accepted ADR-018. |
| `P04-CANON-R1-002` | ReviewBundle schema thiếu `evidence_set_id` và `contract_id` | **CLOSED** | `review-bundle.schema.json` declares and requires both fields; valid fixture passes. |
| `P04-CANON-R1-003` | `bundle_hash` tự tham chiếu; payload trộn row metadata | **CLOSED** | `ReviewBundlePayload` separated from `ReviewBundleRecord`; pre-hash schema excludes hash and post-hash metadata; JCS hash preimage validated. |
| `P04-CANON-R1-004` | Vị ngữ hold, tên audit event và sequencing sai trong `docs/14` | **CLOSED** | Binding guard and dirty intake sequences rectified; zero forbidden literals across repository. |

### Round 2 Findings

### Finding P04-CANON-R2-001 — Sai lệch Ownership giữa Subtask P04A và Subtask P04B
- **Severity**: HIGH
- **Description**: Trong các tài liệu canonical reconciled, có sự sai lệch về ranh giới trách nhiệm giữa Subtask P04A và Subtask P04B:
  1. Subtask P04B là module in-memory thuần túy: chỉ thực hiện read-only Git inspection (`git status`, `git diff-index`, `git diff`) và trả về kết quả cấu trúc trong bộ nhớ (`GitEvidenceResult`). P04B có ZERO quyền ghi SQLite, ZERO quyền append audit event, ZERO quyền mutate hold, và KHÔNG sở hữu Transaction A hay diagnostic transaction.
  2. Subtask P04A là chủ sở hữu Schema v6, sở hữu Transaction A (report intake & `worker_claims` persistence), nhận kết quả từ P04B, thực hiện rollback Transaction A khi worktree/index dirty, và sở hữu diagnostic transaction (append `EVIDENCE_COLLECTION_FAILED`, insert active hold `review_integrity_holds` với `DIRTY_WORKTREE_DETECTED`, giữ `RUNNING`).
  3. Một số tài liệu đã gán nhầm việc ghi diagnostic hold cho P04B, hoặc dùng cụm từ mâu thuẫn "P04A / in-memory collector".
- **Remediation Requirement**:
  - Chuẩn hóa đồng bộ ranh giới: P04B thực hiện read-only inspection; P04A sở hữu Transaction A và diagnostic persistence khi intake thất bại.
  - Cập nhật đồng bộ: `docs/04_ARCHITECTURE.md`, `docs/05_DOMAIN_MODEL.md` (không sửa các khối SQL), `docs/14_FAILURE_RECOVERY.md`, `docs/17_ROADMAP.md`, `docs/21_TRACEABILITY_MATRIX.md`, `docs/22_MODULE_PROVENANCE.md`, và `docs/phases/P04_EVIDENCE_REVIEW.md`.
- **Status**: `OPEN` (Remediation Required).

### Finding P04-CANON-R2-002 — Citation Erratum trong Audit Record 001
- **Severity**: LOW (Governance Citation)
- **Description**: Finding `P04-CANON-R1-001` trong Audit Record 001 trích dẫn sai tên cột định danh Win32 (`volume_serial_number_hex`, `file_index_high_hex`, `file_index_low_hex`).
- **Remediation**: Đã ban hành append-only erratum [P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001_ERRATUM_001.md](P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001_ERRATUM_001.md) xác nhận tên canonical chuẩn là `volume_serial_hex`, `file_id_hex`, `linked_gitdir_volume_serial_hex`, `linked_gitdir_file_id_hex`.
- **Status**: `CLOSED_BY_APPEND_ONLY_ERRATUM`.

---

## 3. Mandatory Remediation Directives

1. Tiếp tục triển khai remediation trên branch `codex/p04-doc-reconciliation` đối với finding `P04-CANON-R2-001`.
2. Không nhập commit governance vào implementation branch.
3. Chạy toàn bộ các probe kiểm chứng tĩnh và động trước khi đệ trình re-audit.
