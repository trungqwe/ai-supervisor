# P03 ADR-017 / TASK-P03-004 External Design Re-Audit 004

- **Audited Commit SHA**: `b023d941330dd4d381c551a2a63faed5a820a748` trên nhánh `main`.
- **Audited Documents**:
  - `docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md` (Revision 4)
  - `docs/adr/DRAFT-ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md` (Revision 4)
  - `docs/plans/PLAN-HOST-INTEGRATION-DEPENDENCY.md` (Revision 6)
  - `docs/tasks/DRAFT_TASK_CONTRACT_P03_004.md`
- **Authority**: External Supervisor Design Re-Audit 004
- **APPROVED DIRECTION**: Subtask `TASK-P03-004` thuộc Phase P03 tiếp tục được phê duyệt về mặt định hướng nhằm hoàn tất exit gate P03 trước khi mở Phase P04.
- **DESIGN VERDICT**: `DRAFT_ADR_017 = REVISION_5_PENDING` (chưa ACCEPTED, Task Contract 004 `NOT_RELEASED`, chưa viết mã nguồn Go).
- **Phân loại Trạng thái Findings**:
  - `ADR17-R4-001` (Shutdown drain, dọn metadata, giải phóng lock cuối cùng): **CLOSED** ở cấp thiết kế.
  - `ADR17-R4-002` (Chốt CreateFileW contract cho .owner.lock): **CLOSED** ở cấp thiết kế.
  - `ADR17-R4-004` (Binary readiness probe vs chứng minh Pair hold trong harness): **CLOSED** ở cấp thiết kế.
  - `ADR17-R4-005` (SID impersonation/token check, revert context, owner_instance_id as freshness marker): **CLOSED** ở cấp thiết kế.
  - `ADR17-R4-003` (Hoàn tất chứng minh DB identity): Chuyển thành `ADR17-R5-001` giải quyết trọn vẹn trong Revision 5.

---

## 1. Danh sách Findings Hiệu chỉnh Đợt 5 (ADR17-R5-001..002)

| Finding | Phân loại & Yêu cầu Kỹ thuật | Trạng thái Đợt 5 |
|---|---|---|
| `ADR17-R5-001` | **Phân Biệt Canonical Lock Key vs Physical File Identity & Đối Chiếu Post-Store.Open**: Phân biệt rành mạch giữa Canonical Lock Key (path-based) `<canonical_db_path>.owner.lock` với Physical File Identity (volume + file ID). Định nghĩa phép đối chiếu identity trước và sau `Store.Open()` bằng VolumeSerialNumber + FileId (trên ReFS bắt buộc dùng 128-bit `FILE_ID_INFO` via `GetFileInformationByHandleEx`; trên NTFS dùng `FILE_ID_INFO` hoặc 64-bit index). Xử lý DB mới qua parent directory volume check. Fail-closed ngay lập tức khi không chứng minh được cùng DB/lock key (mismatch, symlink swap, hardlink `nNumberOfLinks > 1`, filesystem không hỗ trợ). Không khẳng định subst, junction, casing luôn tự động hội tụ mà đưa vào Ma Trận Kiểm Chứng Hai Tiến Trình (Two-Process Probe Matrix) để probe xác nhận. | OPEN (đã xử lý trong Revision 5) |
| `ADR17-R5-002` | **Hoàn Thiện DRAFT Task Contract 004 JSON Hợp Chuẩn docs/08 (Giữ NOT_RELEASED)**: Hoàn thiện `docs/tasks/DRAFT_TASK_CONTRACT_P03_004.md` thành tài liệu chứa TaskContract JSON object đầy đủ các trường theo `docs/08_TASK_CONTRACT.md` và `docs/schemas/task-contract.schema.json`. Chốt `base_sha = 35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea` (post-3D clean merge code base SHA) theo giao thức tách biệt artifact/base; allowed/forbidden globs rõ ràng; verification requests sử dụng profile `go-test` đã duyệt; required evidence đầy đủ cho Windows cross-session lock, startup-before-serve probe, shutdown drain, DB identity post-open check, Pair hold admission guard và AO integration harness 5 bước. Tách bạch rõ mock AO test khỏi live AO proof; ghi rõ test library không phải bằng chứng host principal thật. Giữ nguyên trạng thái `DRAFT_NOT_RELEASED`. | OPEN (đã xử lý trong Revision 5) |

---

## 2. Gate & Dependency Status
- `ACTIVE_GATE = TASK_P03_003D_HANDOFF_VERIFICATION` (chờ audit gói thiết kế Revision 5 và DRAFT Task Contract 004)
- `HOST_QUIESCENCE_INTEGRATION = OPEN`
- `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
- `TASK_P03_004_CONTRACT = NOT_RELEASED`
- Verified host principal unproven at runtime; `AUTOMATIC_RESTORE = DISABLED`.
