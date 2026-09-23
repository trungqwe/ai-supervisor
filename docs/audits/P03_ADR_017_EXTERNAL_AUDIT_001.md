# P03 ADR-017 / TASK-P03-004 External Design Audit 001

- **Audited Commit SHA**: `d57735dba23f4d037c195ec4cb3f9e35cd83054f` trên nhánh `main`.
- **Audited Documents**:
  - `docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md`
  - `docs/adr/DRAFT-ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md`
  - `docs/plans/PLAN-HOST-INTEGRATION-DEPENDENCY.md` (Revision 2)
- **Authority**: External Supervisor Design Audit
- **APPROVED DIRECTION**: Subtask `TASK-P03-004` thuộc Phase P03; mục tiêu là hoàn tất exit gate AO integration của P03 trước khi mở P04.
- **DESIGN VERDICT**: `DRAFT_ADR_017 = REVISION_1_REQUIRED` (chưa release Task Contract, chưa viết Go, chưa tuyên bố Phase P03 hoàn tất).

## 1. Danh sách Findings Cần Hiệu chỉnh (ADR17-R1-001..006)

| Finding | Phân loại & Yêu cầu Kỹ thuật | Trạng thái |
|---|---|---|
| `ADR17-R1-001` | **Tách ProcessOwnerLease vs ExclusiveScope**: Tách rõ ràng lease dài hạn cấp process theo DB (`ProcessOwnerLease`) giữ từ trước `Runner.Run` đến sau shutdown drain/Store close, khỏi `ExclusiveScope` ngắn hạn mà `Runner.Run` tự acquire/release. Định nghĩa thứ tự acquire/release, crash/abandoned owner, cạnh tranh hai process, handle inheritance (`bInheritHandle=FALSE`), canonical DB identity và phạm vi single-host Windows. | OPEN (đã xử lý trong Revision 1) |
| `ADR17-R1-002` | **Cooperative Takeover V1 (Bỏ Auto-kill)**: V1 chỉ áp dụng cooperative takeover. Nếu owner cũ không drain/thoát hoặc không xác minh được quyền sở hữu, fail-closed ngay lập tức; không tự động gọi `TerminateProcess`. Mọi đề xuất auto-kill cần lập proposal riêng có bằng chứng an toàn. | OPEN (đã xử lý trong Revision 1) |
| `ADR17-R1-003` | **Chốt Cơ chế Owner Lock Chính trên Windows**: Khóa chính là Windows Named Mutex (`Local\AISupervisor_<DBHash>`) do kernel quản lý vòng đời; khóa phụ là Sidecar Lock File chứa metadata liên lạc IPC. Quy định rõ thứ tự acquire (Mutex trước, Sidecar sau), rollback khi thất bại, Explicit DACL chống pre-creation DoS. Không gộp owner lock với scope của startup scan. | OPEN (đã xử lý trong Revision 1) |
| `ADR17-R1-004` | **Ma trận Kiểm chứng Binary Thật & Tách Mock/Live AO**: Bổ sung ma trận kiểm chứng các kịch bản (trước/đang/sau Run, Complete+PendingAO, Run lỗi, hai process cạnh tranh, shutdown drain, và 5 tiêu chí exit gate P03: session create, dispatch, observation reconciliation, raw file read, teardown). Kiểm tra trạng thái cổng/admission thực tế bằng network probe, log chỉ bổ trợ. Tách test Mock AO tự động khỏi Live AO proof có kiểm soát. | OPEN (đã xử lý trong Revision 1) |
| `ADR17-R1-005` | **Giới hạn Scope P03 Host & Đối soát SEC-003**: Giới hạn P03 host ở bootstrap/admission và verification runner tối thiểu; không làm trước bộ 12 tool P05. Ghi nhận mô hình ChatGPT Web là reasoning agent, Control Plane cung cấp local tools; ở P04/P05 phải đối soát tool đọc log/verification runner với `docs/11`, `docs/02`, và `SEC-003`; tuyệt đối không mở arbitrary shell execution. | OPEN (đã xử lý trong Revision 1) |
| `ADR17-R1-006` | **Chính sách Vận hành UNSET & Fail-closed Runtime**: 8 chính sách vận hành tiếp tục giữ nguyên `UNSET` trong tài liệu; runtime chỉ chạy khi được inject giá trị cấu hình hợp lệ, nếu thiếu thì fail-closed khi khởi động, không dùng default ngầm. | OPEN (đã xử lý trong Revision 1) |

## 2. Gate & Dependency Status
- `ACTIVE_GATE = TASK_P03_003D_HANDOFF_VERIFICATION` (chờ re-audit ADR-017 Revision 1)
- `HOST_QUIESCENCE_INTEGRATION = OPEN`
- `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
- Verified host principal unproven at runtime; `AUTOMATIC_RESTORE = DISABLED`.
