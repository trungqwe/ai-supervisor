# DRAFT ADR-017 — Host Quiescence & Daemon Lifecycle Architecture

> **Status:** `DRAFT_PENDING_SUPERVISOR_AUDIT`
> **Authority:** `docs/24_CHANGE_GOVERNANCE.md` (Level 2 Approved ADR)
> **Liên quan:** ADR-001, ADR-007, ADR-016 (D8, D9, D11, D12, D13), ADR-016 Addenda, `PROPOSAL-P03-005`.
> **Phạm vi:** Kiến trúc Host Quiescence, Windows Exclusivity, Daemon Lifecycle và ranh giới Phase P03/P04.

---

## 1. Bối cảnh & Động lực Kiến trúc

Phase P03 đã hoàn tất nghiệm thu toàn bộ các mô-đun thư viện Go nội bộ quản lý giao tiếp với Untrivial Agent Orchestrator (AO), session lifecycle, crash recovery scanner, execution budget và timeout monitor (các subtask 3A–3D).

Tuy nhiên, `docs/17_ROADMAP.md` quy định exit gate tổng thể của Phase P03 đòi hỏi phải kiểm chứng tự động vòng đời phiên làm việc mà không phụ thuộc vào Phase P04. Hiện tại codebase chưa có daemon entrypoint (`cmd/`), chưa có cơ chế kiểm soát độc quyền liên tiến trình (cross-process exclusivity), và chưa có bằng chứng runtime về việc scanner hoàn tất trước khi mở cổng nhận request (`DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`).

Đồng thời, Phase P04 được quy hoạch chuyên trách cho Evidence & Review Engine, hoàn toàn không sở hữu daemon bootstrap. Do đó, cần có một Architecture Decision Record chính thức quy định kiến trúc Host Quiescence và Daemon Lifecycle cho subtask `TASK-P03-004` trong Phase P03.

---

## 2. Quyết định Kiến trúc

### 2.1. Bổ sung Subtask `TASK-P03-004` vào Phase P03
Quyết định bổ sung subtask `TASK-P03-004 Host Quiescence & Daemon Bootstrap Integration` thuộc Phase P03 để hoàn tất exit gate P03 trước khi mở Phase P04.

### 2.2. Cơ chế Exclusivity Phù hợp Nền tảng Windows
1. **Cấm Dùng POSIX Primitives & Khóa Trực tiếp SQLite**:
   - Tuyệt đối không dùng `flock`, `SIGKILL`, hoặc tín hiệu POSIX không tồn tại trên Windows.
   - Tuyệt đối không đặt khóa file trực tiếp trên database file của SQLite (`.db`), do cơ chế mandatory locking của Windows sẽ gây xung đột trực tiếp với internal pager locking của SQLite (`SQLITE_BUSY` hoặc I/O error).
2. **Cơ chế Khóa Sidecar & Windows Named Mutex**:
   - Exclusivity trên mỗi database được định danh bằng canonical path của file database.
   - Sử dụng sidecar lock file (ví dụ `<db_path>.supervisor.lock`) mở với quyền truy cập độc quyền không chia sẻ (`syscall.FILE_SHARE_READ = 0` hoặc sử dụng `LockFileEx`), kết hợp với Windows Named Mutex theo định dạng `Global\AISupervisor_<SHA256(canonical_db_path)>`.
3. **Quy trình Xử lý Tiến trình Cũ & Bằng chứng Đã Thoát**:
   - Khi một tiến trình Supervisor mới khởi động, nó kiểm tra mutex/sidecar lock.
   - Nếu tiến trình cũ đang nắm giữ, tiến trình mới gửi yêu cầu bàn giao (cooperative shutdown) qua local IPC (Named Pipe hoặc tín hiệu sidecar).
   - Tiến trình cũ nhận yêu cầu: đóng admission, drain/join toàn bộ in-flight effect callers, giải phóng lock và thoát.
   - Xử lý tiến trình cũ không hợp tác: Nếu quá thời hạn chờ an toàn, tiến trình mới mở handle tiến trình cũ qua `OpenProcess(PROCESS_TERMINATE, ...)` và gọi `TerminateProcess`.
   - **Bằng chứng đã thoát**: Tiến trình mới dùng `WaitForSingleObject` trên process handle để xác nhận kernel Windows đã hoàn tất việc giải phóng tiến trình cũ và đóng toàn bộ network socket descriptors.
   - **Ràng buộc Fail-Closed**: Nếu không thể acquire exclusivity hoặc không chứng minh được tiến trình cũ đã thoát, **tuyệt đối không cấp `ExclusiveScope`**. `Runner.Run(ctx)` phải dừng ngay và không mở cổng phục vụ.

### 2.3. Ranh giới Wire Effect & Phân biệt Trạng thái Mơ hồ
1. **Phân biệt Bản chất Bằng chứng**:
   - Việc kernel giải phóng socket hoặc terminate tiến trình cũ **chỉ chứng minh rằng caller cũ không thể phát effect MỚI**.
   - Việc terminate tiến trình cũ **hoàn toàn KHÔNG chứng minh AO chưa nhận effect** đối với các request đã được phát ra dây mạng trước đó.
2. **Bảo toàn Nguyên tắc Fail-Closed đối với Intent Mơ hồ**:
   - Mọi intent chưa được xác nhận outcome (`STOP_REQUESTED`, `SEND_REQUESTED`, `RESTORE_REQUESTED`) phải được duy trì fail-closed.
   - Scanner và poller phải đối soát trạng thái qua fresh GET / observation, hoặc chuyển sang governed resolution audit (`STOP_OPERATION_RESOLVED`), hoặc giữ nguyên quarantine và chờ can thiệp thủ công. Tuyệt đối không tự ý phát lại (replay) effect.

### 2.4. Thống nhất Ba Interface Host tại Một Trusted Authority
Cùng một trusted host boundary phải đồng thời hiện thực và đảm bảo tính nhất quán giữa 3 interface:
1. `recovery.HostQuiescence`: `Acquire(context.Context) (ExclusiveScope, error)` — phạm vi toàn bộ DB và shared admission chung.
2. `recovery.LegacyMaintenanceHost`: `AcquireMaintenance(ctx context.Context, pairID string, purpose string) (ExclusiveScope, error)`.
3. `stop.TimeoutAdmission`: `AcquireEffect(ctx context.Context, pairID string, purpose string) (TimeoutPermit, error)` — tham số thứ ba là purpose string (ví dụ `"TIMEOUT_MONITOR"`).

Toàn bộ việc đóng admission, drain/join, cấp exclusive scope và release permit phải do cùng một trusted host authority điều phối.

### 2.5. Vòng đời Khởi động (Startup-Before-Serve) & Ma trận Kết quả
1. **`Runner.Run(ctx)` Tự Acquire Quiescence**:
   - `Runner.Run(ctx)` tự động gọi `HostQuiescence.Acquire(ctx)` và giải phóng qua `defer scope.Release()`. Host entrypoint chỉ inject provider, không gọi acquire lần hai.
2. **Ma trận Kết quả Phân loại**:
   - **Complete (`report.Complete == true && err == nil`)**: Phân loại hoàn tất. `Runner` tự đánh dấu trường nội bộ `ready = true`. Host kiểm tra Pair guards; khởi động `Poller.Start(ctx)`; khởi động scheduler gọi `TimeoutMonitor.Tick(ctx)`; mở listener admission tiếp nhận request từ bên ngoài.
   - **PendingAO (`report.Complete == true && report.PendingAO == true`)**: Phân loại hoàn tất nhưng có session được đưa vào Pair hold (chờ AO quan sát ngoài). `Runner` tự đánh dấu `ready = true`. Host khởi động `Poller.Start(ctx)` để quan sát nền; các Pair bị hold tiếp tục bị khóa admission; chỉ các Pair sạch mới được mở cổng tiếp nhận.
   - **Incomplete (`report.Complete == false` hoặc `err != nil`)**: Gặp lỗi, context cancel, hoặc attempt thiếu execution budget. `Runner` giữ `ready = false`. Startup admission tiếp tục ĐÓNG fail-closed, daemon dừng phục vụ.
3. **Phân tách Trách nhiệm**:
   - Host điều khiển listener admission và kiểm tra Pair guards từ bên ngoài; không gán trường private `r.ready` của `Runner`.

### 2.6. Vòng đời Dừng An toàn (Graceful Shutdown)
Khi nhận OS interrupt/signal:
1. Đóng listener admission ngay lập tức để ngừng nhận request mới.
2. Dừng `Poller` (`Poller.Stop()`), đợi in-flight ticks drain xong.
3. Dừng scheduler của `TimeoutMonitor`, đợi các caller đang giữ `TimeoutPermit` kết thúc.
4. Đợi tất cả effect callers hoàn tất và release permit (drain/join).
5. Đóng Store và kết nối SQLite database.

---

## 3. Tiêu chuẩn Bằng chứng Runtime & Invariants Bất biến

1. **Bằng chứng Runtime Thật**:
   - Bằng chứng runtime chỉ được công nhận thông qua việc thực thi binary daemon thật (`cmd/supervisor`), chứng minh thứ tự khởi động (scanner hoàn tất trước khi listener mở) và dừng an toàn.
   - Fake integration harness trong thư viện không thay thế được bằng chứng runtime của daemon.
2. **Invariants Bất biến**:
   - `AUTOMATIC_RESTORE = DISABLED`: Giữ nguyên tắt fail-closed.
   - 8 Operational policies tiếp tục giữ nguyên ở trạng thái `UNSET`.
