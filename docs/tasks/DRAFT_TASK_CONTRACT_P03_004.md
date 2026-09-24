# DRAFT TASK CONTRACT: TASK-P03-004

> **Contract Identifier**: `DRAFT-CONTRACT-TASK-P03-004-01`
> **Task ID**: `TASK-P03-004` (Host Quiescence & Daemon Bootstrap Integration)
> **Phase ID**: `P03`
> **Status**: `DRAFT_PENDING_SUPERVISOR_AUDIT / NOT_RELEASED`
> **Authority**: Formulated pursuant to proposed `DRAFT-ADR-017` (Revision 4), `PROPOSAL-P03-005` (Revision 4), and `PLAN-HOST-INTEGRATION-DEPENDENCY.md` (Revision 6).

---

> [!CRITICAL]
> **GOVERNANCE STATUS: DRAFT ONLY / NOT_RELEASED**.
> This document is a **draft candidate contract** prepared for External Supervisor audit.
> It does **NOT** authorize production code execution.
> Production coding remains strictly **`HELD_PENDING_TASK_CONTRACT_RELEASE`** (`TASK_P03_004 = NOT_RELEASED`).
> Do **NOT** modify canonical roadmap or accepted ADRs as if accepted.

---

## 1. Task Objective

Triển khai daemon bootstrap nhị phân (`cmd/supervisor`), bộ quản lý độc quyền máy Windows (`ProcessOwnerLease` qua Windows Exclusive Sidecar Lock File Handle), cơ chế startup-before-serve và integration test harness kiểm chứng exit gate Phase P03 mà không phụ thuộc vào Phase P04 hoặc Phase P05.

---

## 2. Allowed Scope (Khi Được Phê Duyệt Release)

Chỉ các file và thư mục sau được phép tạo hoặc chỉnh sửa khi contract được chính thức release:
- `cmd/supervisor/main.go`
- `cmd/supervisor/`
- `internal/host/` (chứa implementation của `HostQuiescence`, Windows lock handle, và Named Pipe cooperative takeover)
- `test/integration/p03_exit_gate_test.go`
- `test/integration/`

---

## 3. Forbidden Scope (Bất Biến)

Tuyệt đối cấm tạo hoặc sửa đổi các file và thư mục sau:
- Mọi file thuộc Phase P04 (`internal/evidence/`, `internal/review/`, v.v.).
- Mọi tool thuộc Phase P05 (`internal/tools/`, bộ 12 domain tools).
- Các accepted ADRs (`docs/adr/ADR-001.md` đến `ADR-016.md`).
- `docs/17_ROADMAP.md` (chỉ cập nhật sau khi ADR-017 và Task Contract được External Supervisor formally accepted).
- Tuyệt đối không thêm Pair HTTP routes hay effectful Supervisor API (create session, dispatch, workspace read, stop) vào daemon listener P03.
- Tuyệt đối không cấp arbitrary shell execution.

---

## 4. Acceptance Criteria (AC1..AC8)

- **AC1 (Windows Exclusive Sidecar Lock Contract)**: Mở `<canonical_db_path>.owner.lock` bằng `CreateFileW` với `GENERIC_READ | GENERIC_WRITE`, `dwShareMode = 0`, `OPEN_ALWAYS`, `FILE_ATTRIBUTE_NORMAL`, `bInheritHandle = FALSE`. Mọi lỗi acquire đều fail-closed.
- **AC2 (Canonical DB Algorithm & Post-Open Verification)**: Tính canonical path trước `Store.Open()` bằng `GetFinalPathNameByHandleW(VOLUME_NAME_DOS)` (trên DB handle nếu đã tồn tại, hoặc trên parent dir handle với `FILE_FLAG_BACKUP_SEMANTICS` nếu DB mới). Hard links (`nNumberOfLinks > 1`) bị từ chối fail-closed (`ERR_HARDLINK_ALIAS_UNSUPPORTED`). Sau `Store.Open()`, đối chiếu DB file identity thực tế với canonical lock key; nếu mismatch lập tức `Store.Close()` và fail-closed.
- **AC3 (Shutdown Lifecycle & Metadata Cleanup)**: Thứ tự dừng: đóng Named Pipe listener -> drain callers -> `Store.Close()` -> xóa `.owner.json` (chỉ khi `owner_instance_id` khớp với owner hiện tại) -> đóng handle `.owner.lock` **CUỐI CÙNG**. Không xóa metadata sau khi release lock. Owner mới sau crash được phép ghi đè atomically metadata stale sau khi acquire lock.
- **AC4 (Named Pipe Authentication & Revert Context)**: Named Pipe takeover xác thực caller SID qua `ImpersonateNamedPipeClient` và token check, kiểm tra lỗi và bắt buộc gọi `RevertToSelf()`. `owner_instance_id` chỉ là freshness marker. Request không hợp lệ bị từ chối và không làm owner shutdown.
- **AC5 (Binary Readiness Probe)**: Readiness probe trên `cmd/supervisor` chứng minh socket đóng trước Run và mở (200 OK) sau Run Complete (`r.ready == true`).
- **AC6 (P03 Integration Test Harness Exit Gate)**: Kiểm chứng trọn vẹn 5 bước AO (session create, dispatch, observation reconciliation, raw workspace file read, teardown) và chứng minh Pair hold chặn admission khi `PendingAO == true` bằng cách gọi trực tiếp các API điều phối nội bộ của thư viện Go Supervisor (`internal/dispatch`, `internal/store`, `internal/stop`, `internal/ao`, `internal/recovery`).
- **AC7 (Quiescence Separation & No Replay)**: `ExclusiveScope` của `Runner.Run(ctx)` là ngắn hạn; in-flight ambiguous wire effects trước crash giữ nguyên fail-closed, không replay; `AUTOMATIC_RESTORE = DISABLED`.
- **AC8 (Fail-Closed Operational Policies)**: 8 operational policies giữ nguyên `UNSET` trong tài liệu; runtime fail-closed khi khởi động nếu thiếu bất kỳ giá trị nào.
