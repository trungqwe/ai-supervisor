# P03 ADR-017 / TASK-P03-004 External Design Re-Audit 003

- **Audited Commit SHA**: `2d5a80b5c6eed36c180d5d57aff3d427bc1d911d` trên nhánh `main`.
- **Audited Documents**:
  - `docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md` (Revision 3)
  - `docs/adr/DRAFT-ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md` (Revision 3)
  - `docs/plans/PLAN-HOST-INTEGRATION-DEPENDENCY.md` (Revision 5)
- **Authority**: External Supervisor Design Re-Audit 003
- **APPROVED DIRECTION**: Subtask `TASK-P03-004` thuộc Phase P03 tiếp tục được phê duyệt về mặt định hướng nhằm hoàn tất exit gate P03 trước khi mở Phase P04.
- **DESIGN VERDICT**: `DRAFT_ADR_017 = REVISION_4_REQUIRED` (chưa ACCEPTED, Task Contract 004 `NOT_RELEASED`, chưa viết mã nguồn Go).
- **Phạm vi Cho phép**: Chuẩn bị tài liệu dự thảo `DRAFT_TASK_CONTRACT_P03_004.md` để gộp đợt đánh giá tiếp theo; tuyệt đối KHÔNG release contract, KHÔNG sửa roadmap/canonical theo trạng thái đã chấp thuận, KHÔNG viết Go.

---

## 1. Đánh giá Trạng thái Findings & Yêu cầu Hiệu chỉnh Đợt 4 (ADR17-R4-001..005)

| Finding | Phân loại & Yêu cầu Kỹ thuật | Trạng thái Đợt 4 |
|---|---|---|
| `ADR17-R4-001` | **Vòng Đời Shutdown, Dọn Metadata & Giải Phóng Lock Handle Cuối Cùng**: Khi Process A shutdown (bình thường hoặc takeover): A vẫn giữ `.owner.lock`, đóng Named Pipe listener, drain callers, đóng `Store.Close()`, dọn `.owner.json` **CHỈ KHI** `owner_instance_id` trong file vẫn khớp với A. Đóng handle `.owner.lock` **CUỐI CÙNG** sau khi Store đã đóng. Tuyệt đối không xóa metadata sau khi release lock. Owner mới sau crash được phép ghi đè atomically metadata stale sau khi đã acquire lock thành công. | OPEN (đã xử lý trong Revision 4) |
| `ADR17-R4-002` | **Chốt Hợp Đồng CreateFileW Cho .owner.lock**: Xác định chính xác bộ cờ Windows API: `dwDesiredAccess = GENERIC_READ | GENERIC_WRITE` (nonzero, xung đột giữa hai contender), `dwShareMode = 0` (exclusive), `dwCreationDisposition = OPEN_ALWAYS` (không truncate file lock đang tồn tại), `dwFlagsAndAttributes = FILE_ATTRIBUTE_NORMAL`, `bInheritHandle = FALSE`. Mọi lỗi acquire đều fail-closed. Bổ sung ma trận test hai process dùng chính xác các flags này, kể cả khác Windows logon sessions. | OPEN (đã xử lý trong Revision 4) |
| `ADR17-R4-003` | **Giữ Handle DB/Parent & Đối Chiếu File Identity Sau Store.Open**: Giữ handle DB file (hoặc parent directory) đủ lâu để tính canonical path qua `GetFinalPathNameByHandleW(VOLUME_NAME_DOS)` và kiểm tra `nNumberOfLinks == 1`. Sau `Store.Open()`, mở handle đối chiếu DB file identity thực tế với canonical lock key; nếu mismatch (do symlink swap, redirection hoặc alias mismatch) thì lập tức `Store.Close()` và **FAIL-CLOSED** trước khi mở bất kỳ listener hay effect nào. Alias hoặc filesystem không chứng minh được thì từ chối. | OPEN (đã xử lý trong Revision 4) |
| `ADR17-R4-004` | **Phân Định Binary Readiness Probe vs Chứng Minh Pair Hold Trong Harness**: Binary readiness probe trên `cmd/supervisor` CHỈ chứng minh socket đóng trước Run và mở sau Run Complete (`r.ready == true`, 200 OK). Bằng chứng Pair hold chặn admission được chứng minh bằng Store/admission guards trong P03 Integration Test Harness; tuyệt đối KHÔNG thêm Pair HTTP routes hay effectful Supervisor API vào daemon P03. | OPEN (đã xử lý trong Revision 4) |
| `ADR17-R4-005` | **Xác Thực SID Qua Impersonation/Token & Revert Context**: Named Pipe takeover xác thực caller SID qua `ImpersonateNamedPipeClient` và token check, kiểm tra lỗi trả về và bắt buộc hoàn nguyên context qua `RevertToSelf()`. `owner_instance_id` chỉ đóng vai trò là **freshness marker** (chống replay stale request), không phải là secret xác thực. Request không hợp lệ không được khiến owner shutdown. | OPEN (đã xử lý trong Revision 4) |

---

## 2. Gate & Dependency Status
- `ACTIVE_GATE = TASK_P03_003D_HANDOFF_VERIFICATION` (chờ audit gói thiết kế Revision 4 và DRAFT Task Contract 004)
- `HOST_QUIESCENCE_INTEGRATION = OPEN`
- `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
- `TASK_P03_004_CONTRACT = NOT_RELEASED`
- Verified host principal unproven at runtime; `AUTOMATIC_RESTORE = DISABLED`.
