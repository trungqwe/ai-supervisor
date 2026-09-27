# P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md

**Audit Target**: Canonical Specification Reconciliation implementation (CR-01..CR-12)
**Audited Branch**: `codex/p04-doc-reconciliation`
**Audited Commit**: `06d255a3b0e80ba3ee29ebde7d828a24cf28f5ff`
**Base SHA**: `b174ef3e88409f82b10afe0fcce88ba218530863`
**Governance Baseline**: `6d38c5fdf12fe12da64f3f7a9a484f97cffc6140`
**Authority**: External Supervisor
**Verdict**: `P04_CANONICAL_RECONCILIATION = REVISION_REQUIRED`
**Date**: 2026-09-27
**Active Gate**: `P04_CANONICAL_RECONCILIATION_REMEDIATION`

---

## 1. Executive Summary & Audit Determination

1. **Reconciliation Implementation Review**:
   External Supervisor evaluated the canonical specification reconciliation changes submitted on isolated branch `codex/p04-doc-reconciliation` at commit `06d255a3b0e80ba3ee29ebde7d828a24cf28f5ff` against base SHA `b174ef3e88409f82b10afe0fcce88ba218530863`.

2. **Audit Findings Summary**:
   - `P04-CANON-R1-001`: DDL drift trong `docs/05_DOMAIN_MODEL.md` so với văn bản chuẩn hóa của accepted ADR-018.
   - `P04-CANON-R1-002`: `docs/schemas/review-bundle.schema.json` thiếu hai thuộc tính bắt buộc `evidence_set_id` và `contract_id`.
   - `P04-CANON-R1-003`: `bundle_hash` bị đưa vào payload gây tự tham chiếu trong RFC 8785 JCS hash preimage; payload bị trộn lẫn với metadata của Transaction C / SQLite row.
   - `P04-CANON-R1-004`: Vị ngữ hold, tên audit event và trình tự giao dịch xử lý binding guard / dirty intake trong `docs/14_FAILURE_RECOVERY.md` sai lệch so với ADR-018.

3. **Audit Verdict**:
   - `P04_CANONICAL_RECONCILIATION = REVISION_REQUIRED`.
   - Implementation branch `codex/p04-doc-reconciliation` **KHÔNG ĐƯỢC MERGE** vào `main`.
   - Subtask P04A Task Contract **CHƯA ĐƯỢC PHÁT HÀNH** (`TASK_P04A_TASK_CONTRACT = NOT_RELEASED`).
   - Tuyệt đối không viết code Go hay migrations (`P04_CODE = HELD`).
   - Tuyệt đối không sửa accepted ADR-018, approved proposals, released contract, hoặc audit lịch sử.

---

## 2. Detailed Findings

### Finding P04-CANON-R1-001 — DDL Drift trong docs/05_DOMAIN_MODEL.md
- **Severity**: HIGH
- **Description**: Các định nghĩa bảng, ràng buộc CHECK, index và trigger trong `docs/05_DOMAIN_MODEL.md` đã bị giản lược hoặc sai lệch so với DDL chuẩn hóa trong accepted ADR-018:
  * `attempt_workspace_bindings`: thiếu trạng thái `'RETAINED_FOR_VERIFICATION'` trong `binding_state CHECK`; thiếu các trường định danh Win32 dạng hex (`volume_serial_number_hex`, `file_index_high_hex`, `file_index_low_hex`), thiếu `linked_gitdir` identity fields; thiếu `pinned_ao_commit`.
  * `worker_claims`: thiếu cột `payload_json`.
  * `review_integrity_holds`: hold reasons và foreign keys tới audit events bị sai lệch.
  * `task_verification_leases`: thiếu chuỗi lease tuyến tính (`predecessor_lease_id`, `ttl_seconds`, `worker_id`, state enum), thiếu hai partial active indexes và các trigger bảo vệ bất biến / CAS / lineage.
- **Remediation Requirement**: Thay thế toàn bộ DDL, index và trigger của 4 thực thể trên bằng nguyên văn văn bản canonical từ accepted ADR-018 mà không giản lược hay diễn giải lại.
- **Status**: `OPEN` (Remediation Required).

### Finding P04-CANON-R1-002 — ReviewBundle Schema thiếu evidence_set_id và contract_id
- **Severity**: HIGH
- **Description**: Schema `docs/schemas/review-bundle.schema.json` và các tài liệu liên quan không khai báo và không require hai định danh quan trọng kết nối bundle với contract và bằng chứng: `evidence_set_id` và `contract_id`.
- **Remediation Requirement**: Khai báo rõ ràng và đưa vào mảng `required` của `review-bundle.schema.json` hai thuộc tính `evidence_set_id` và `contract_id`. Cập nhật fixtures tương ứng.
- **Status**: `OPEN` (Remediation Required).

### Finding P04-CANON-R1-003 — bundle_hash Tự Tham Chiếu & Payload Bị Trộn Row Metadata
- **Severity**: CRITICAL
- **Description**: Schema và fixtures của ReviewBundle đặt `bundle_hash` ngay trong payload JSON, gây ra nghịch lý tự tham chiếu (preimage paradox) khi tính toán băm RFC 8785 JCS. Đồng thời, payload bị trộn lẫn các trường metadata chỉ được thiết lập sau khi hash và commit dòng cơ sở dữ liệu (`bundle_assembled_at_epoch_ms`, `compilation_latency_ms`, `latency_measurement_status`, `nfr008_compliance_status`).
- **Remediation Requirement**:
  * Phân định dứt khoát giữa `ReviewBundlePayload` (canonical pre-hash JSON, được schema kiểm chứng và dùng làm preimage cho băm JCS) và `ReviewBundleRecord` (dòng SQLite trong bảng `review_bundles` chứa envelope, `bundle_hash`, và metadata Transaction C).
  * `review-bundle.schema.json` chỉ kiểm chứng `ReviewBundlePayload`, không chứa `bundle_hash` và 4 trường metadata hậu băm.
  * `evidence_finalized_at_epoch_ms` được giữ lại trong payload vì đây là mốc thời gian T0 đã xác lập trước khi tổng hợp bundle.
  * Giữ `additionalProperties: false`.
  * Fixture invalid phải vi phạm một invariant thực sự của payload.
- **Status**: `OPEN` (Remediation Required).

### Finding P04-CANON-R1-004 — Hold/Audit Literals và Trình Tự Giao Dịch Sai Lệch trong docs/14
- **Severity**: HIGH
- **Description**:
  * Khi phát hiện vi phạm binding guard trước send: `docs/14_FAILURE_RECOVERY.md` mô tả chuyển ngay sang terminalization D12 là sai trình tự. Trình tự đúng: Task ban đầu giữ `DISPATCHED`, operation giữ `DISPATCH_BOUND`, attempt mở; ghi audit event `REVIEW_INTEGRITY_CONFLICT` với `conflict_source = 'WORKSPACE_BINDING_GUARD'`, `hold_reason = 'INVARIANT_MISMATCH'`; CAS binding `ACTIVE -> INVALIDATED`; sau đó verified operator mới thực hiện giao dịch D12 terminalization riêng biệt; và hold resolution là giao dịch tiếp theo sau terminalization.
  * Khi collector phát hiện dirty worktree trong lúc `RUNNING`: `docs/14` tự phát minh literal mới (`CLEAN_INTAKE_DIRTY_WORKTREE`, `CLEAN_INTAKE_DIRTY_WORKTREE_DETECTED`) thay vì sử dụng canonical literals đã thiết lập trong ADR-018: event `EVIDENCE_COLLECTION_FAILED` và `hold_reason = 'DIRTY_WORKTREE_DETECTED'`.
- **Remediation Requirement**: Sửa lại toàn bộ trình tự và vị ngữ trong `docs/14_FAILURE_RECOVERY.md` đúng với canonical rules của ADR-018; triệt tiêu 100% các literal tự phát minh.
- **Status**: `OPEN` (Remediation Required).

---

## 3. Governance Directives for Remediation

1. Thực hiện ghi nhận governance trên `main`:
   - Tạo audit `docs/audits/P04_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT_001.md`.
   - Tạo erratum `docs/audits/P04_CANONICAL_RECONCILIATION_PLAN_EXTERNAL_AUDIT_ERRATUM_001.md`.
   - Cập nhật plan `PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` sang status `EXTERNAL_AUDIT_APPROVED_WITH_ERRATUM_001`.
   - Cập nhật `AGENTS.md` và `docs/18_CURRENT_STATE.md`.
   - Commit và push governance riêng trên `main`.
2. Chuyển sang nhánh `codex/p04-doc-reconciliation` để thi hành remediation vòng 1:
   - Sửa `P04-CANON-R1-001`, `P04-CANON-R1-002`, `P04-CANON-R1-003`, `P04-CANON-R1-004`.
   - Thực hiện đầy đủ 7 bước kiểm chứng bắt buộc.
   - Commit và push implementation riêng; không merge vào `main`.
