# DRAFT ADR-017 — Host Quiescence & Daemon Lifecycle Architecture

> **Status:** `REVISION_1_PENDING_SUPERVISOR_REAUDIT`
> **Authority:** `docs/24_CHANGE_GOVERNANCE.md` (Level 2 Approved ADR)
> **Liên quan:** ADR-001, ADR-007, ADR-016 (D8, D9, D11, D12, D13), ADR-016 Addenda, `PROPOSAL-P03-005`.
> **Phạm vi:** Kiến trúc Host Quiescence, Windows Exclusivity, Daemon Lifecycle và ranh giới Phase P03/P04.

---

## 1. Bối cảnh & Động lực Kiến trúc

Phase P03 đã hoàn tất nghiệm thu toàn bộ các mô-đun thư viện Go nội bộ quản lý giao tiếp với Untrivial Agent Orchestrator (AO), session lifecycle, crash recovery scanner, execution budget và timeout monitor (các subtask 3A–3D).

Tuy nhiên, `docs/17_ROADMAP.md` quy định exit gate tổng thể của Phase P03 đòi hỏi phải kiểm chứng tự động vòng đời phiên làm việc mà không phụ thuộc vào Phase P04. Hiện tại codebase chưa có daemon entrypoint (`cmd/`), chưa có cơ chế kiểm soát độc quyền liên tiến trình (cross-process exclusivity), và chưa có bằng chứng runtime về việc scanner hoàn tất trước khi mở cổng nhận request (`DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`).

Đồng thời, Phase P04 được quy hoạch chuyên biệt cho Evidence & Review Engine, hoàn toàn không sở hữu daemon bootstrap. Do đó, cần có một Architecture Decision Record chính thức quy định kiến trúc Host Quiescence và Daemon Lifecycle cho subtask `TASK-P03-004` trong Phase P03.

---

## 2. Quyết định Kiến trúc

### 2.1. Bổ sung Subtask `TASK-P03-004` vào Phase P03
Quyết định bổ sung subtask `TASK-P03-004 Host Quiescence & Daemon Bootstrap Integration` thuộc Phase P03 để hoàn tất exit gate P03 trước khi mở Phase P04.

### 2.2. Phân Tách Rõ Ràng Hai Khái Niệm Quiescence & Lease
1. **`ProcessOwnerLease` (Dài hạn, cấp Process/Host)**:
   - Phạm vi: Toàn bộ database (gắn với canonical DB path trên máy Windows).
   - Vòng đời: Giữ độc quyền từ **TRƯỚC** khi gọi `Runner.Run(ctx)`, xuyên suốt quá trình daemon hoạt động, cho đến **SAU** khi shutdown drain hoàn tất và đóng Store / DB connection.
   - Mục đích: Bảo đảm tại một thời điểm trên một máy Windows chỉ có DUY NHẤT một tiến trình daemon sở hữu DB (Single Active Daemon).
2. **`ExclusiveScope` (Ngắn hạn, cấp Recovery Sweep)**:
   - Phạm vi: Được định nghĩa tại `internal/recovery/scanner.go` (`HostQuiescence.Acquire(ctx)`).
   - Vòng đời: `Runner.Run(ctx)` **tự động gọi** `HostQuiescence.Acquire(ctx)` để lấy `ExclusiveScope` khi bắt đầu sweep và gọi `scope.Release()` ngay khi sweep hoàn tất.
   - Tách biệt: Khi `ExclusiveScope.Release()` được gọi, `ProcessOwnerLease` **vẫn tiếp tục được giữ** bởi daemon cho đến khi daemon tắt hoàn toàn. Entrypoint tuyệt đối không gọi `HostQuiescence.Acquire()` lần hai.

### 2.3. Cơ chế Exclusivity Phù hợp Nền tảng Windows (Single Windows Host)
1. **Cấm POSIX Primitives & Khóa Trực tiếp SQLite**:
   - Tuyệt đối không dùng `flock`, `SIGKILL`, hoặc tín hiệu POSIX không tồn tại trên Windows.
   - Tuyệt đối không đặt khóa file trực tiếp trên database file SQLite (`.db`), do cơ chế mandatory locking của Windows sẽ gây xung đột trực tiếp với internal pager locking của SQLite (`SQLITE_BUSY` hoặc access denied).
2. **Cơ chế Khóa Chính: Windows Named Mutex**:
   - Khóa độc quyền chính là Windows Named Mutex đặt trong namespace cục bộ: `Local\AISupervisor_<SHA256(canonical_db_path)>`.
   - Vòng đời: Kernel Windows tự động giải phóng mutex khi tiến trình kết thúc (kể cả crash/killed), trả về trạng thái `WAIT_ABANDONED` cho tiến trình kế tiếp nhận diện để chạy recovery sweep.
   - ACL & Chống Pre-creation DoS: Sử dụng `SECURITY_ATTRIBUTES` với explicit DACL chỉ cấp quyền cho current user SID (và Administrators/SYSTEM), từ chối `Everyone` để ngăn chặn unprivileged user pre-create mutex.
   - Handle Inheritance: Mọi handle tạo ra đều bắt buộc thiết lập `bInheritHandle = FALSE` để ngăn chặn tiến trình con (như git hoặc test process) kế thừa handle làm rò rỉ lock.
3. **Cơ chế Khóa Phụ: Sidecar Lock File (Metadata & IPC Carrier)**:
   - File sidecar: `<db_path>.supervisor.owner`.
   - Mục đích: Chứa thông tin nhận diện owner (`pid`, `owner_instance_id`, `pipe_name`, `started_at`).
   - Mở với chế độ `FILE_SHARE_READ` để tiến trình khác có thể đọc metadata liên lạc, nhưng chỉ owner đang nắm Named Mutex mới được phép ghi.
4. **Thứ tự Acquire & Rollback**:
   - Bước 1: Acquire Named Mutex (`CreateMutexW`). Nếu thất bại -> không ghi sidecar, fail-closed.
   - Bước 2: Ghi metadata vào Sidecar Lock File.
   - Rollback: Nếu ghi sidecar thất bại -> xóa sidecar (nếu có), release Named Mutex, fail-closed.
5. **V1 Chỉ Áp dụng Cooperative Takeover (Bỏ Auto-kill)**:
   - Khi Process B thấy Process A đang nắm mutex: Process B kết nối tới Named Pipe của Process A yêu cầu bàn giao (cooperative shutdown).
   - Process A nhận yêu cầu: đóng admission, drain/join toàn bộ in-flight effect callers, release `ProcessOwnerLease`, và tự thoát.
   - **Ràng buộc Fail-Closed**: Nếu Process A không hợp tác, không phản hồi hoặc không thoát trong thời hạn chờ an toàn, Process B **FAIL-CLOSED NGAY LẬP TỨC**. Tuyệt đối không tự ý gọi `TerminateProcess` trong V1.

### 2.4. Phân biệt Wire Effect Mới vs. Outcome Đã phát Chưa biết
1. **Bản chất Bằng chứng**:
   - Việc xác nhận owner cũ đã thoát hoặc thu hồi permit **chỉ chứng minh rằng caller cũ không thể phát effect MỚI**.
   - Việc đóng socket hoặc process cũ thoát **hoàn toàn KHÔNG chứng minh AO chưa nhận effect** đối với các request đã phát ra dây mạng trước đó.
2. **Bảo toàn Nguyên tắc Fail-Closed đối với Intent Mơ hồ**:
   - Mọi intent chưa được xác nhận outcome (`STOP_REQUESTED`, `SEND_REQUESTED`, `RESTORE_REQUESTED`) phải được duy trì fail-closed.
   - Scanner và poller phải đối soát trạng thái qua fresh GET / observation, hoặc chuyển sang governed resolution audit (`STOP_OPERATION_RESOLVED`), hoặc giữ nguyên quarantine và chờ can thiệp thủ công. Tuyệt đối không tự ý phát lại (replay) effect.

### 2.5. Thống nhất Ba Interface Host tại Một Trusted Authority
Cùng một trusted host boundary phải đồng thời hiện thực và đảm bảo tính nhất quán giữa 3 interface:
1. `recovery.HostQuiescence`: `Acquire(ctx context.Context) (ExclusiveScope, error)` — phạm vi toàn bộ DB và shared admission chung.
2. `recovery.LegacyMaintenanceHost`: `AcquireMaintenance(ctx context.Context, pairID string, purpose string) (ExclusiveScope, error)`.
3. `stop.TimeoutAdmission`: `AcquireEffect(ctx context.Context, pairID string, purpose string) (TimeoutPermit, error)` — tham số thứ ba là `purpose string` (ví dụ `"TIMEOUT_MONITOR"`).

Toàn bộ việc đóng admission, drain/join, cấp exclusive scope và release permit phải do cùng một trusted host authority điều phối.

### 2.6. Vòng đời Khởi động (Startup-Before-Serve) & Ma trận Kết quả
1. **`Runner.Run(ctx)` Tự Acquire Quiescence**:
   - `Runner.Run(ctx)` tự động gọi `HostQuiescence.Acquire(ctx)` và giải phóng qua `defer scope.Release()`. Host entrypoint chỉ inject provider, không gọi acquire lần hai.
2. **Ma trận Kết quả Phân loại**:
   - **Complete (`report.Complete == true && err == nil`)**: Phân loại hoàn tất. `Runner` tự đánh dấu trường nội bộ `r.ready = true`. Host kiểm tra Pair guards; khởi động `Poller.Start(ctx)`; khởi động scheduler gọi `TimeoutMonitor.Tick(ctx)`; mở listener admission tiếp nhận request từ bên ngoài.
   - **PendingAO (`report.Complete == true && report.PendingAO == true`)**: Phân loại hoàn tất nhưng có session được đưa vào Pair hold (chờ AO quan sát ngoài). `Runner` tự đánh dấu `r.ready = true`. Host khởi động `Poller.Start(ctx)` để quan sát nền; các Pair bị hold tiếp tục bị khóa admission; chỉ các Pair sạch mới được mở cổng tiếp nhận.
   - **Incomplete (`report.Complete == false` hoặc `err != nil`)**: Gặp lỗi, context cancel, hoặc attempt thiếu execution budget. `Runner` giữ `r.ready = false`. Startup admission tiếp tục ĐÓNG fail-closed, daemon dừng phục vụ.
3. **Phân tách Trách nhiệm**:
   - Host điều khiển listener admission và kiểm tra Pair guards từ bên ngoài; không gán trường private `r.ready` của `Runner`.
4. **`TimeoutMonitor` Scheduling**:
   - `TimeoutMonitor` chỉ có phương thức `Tick(ctx)`. Host daemon chịu trách nhiệm thiết lập ticker scheduler định kỳ gọi `Tick(ctx)`.

### 2.7. Vòng đời Dừng An toàn (Graceful Shutdown)
Khi nhận OS interrupt/signal:
1. Đóng listener admission ngay lập tức để ngừng nhận request mới.
2. Dừng `Poller` (`Poller.Stop()`), đợi in-flight ticks drain xong.
3. Dừng scheduler của `TimeoutMonitor`, đợi các caller đang giữ `TimeoutPermit` kết thúc.
4. Đợi tất cả effect callers hoàn tất và release permit (drain/join) **TRƯỚC KHI** đóng Store và kết nối SQLite database.
5. Sau khi Store đóng, giải phóng `ProcessOwnerLease` (xóa sidecar file và release Named Mutex).

---

## 3. Ranh giới Tooling P03/P05 & An ninh SEC-003

1. **Giới hạn Scope P03 Host**:
   - P03 host chỉ giới hạn ở daemon bootstrap (`cmd/supervisor`), host quiescence provider và verification runner tối thiểu phục vụ kiểm chứng exit gate P03.
   - Tuyệt đối **KHÔNG triển khai trước bộ 12 domain tools của P05**.
2. **Ghi nhận Yêu cầu Người dùng & An ninh**:
   - ChatGPT Web đóng vai trò là **agent suy luận** (reasoning agent). Supervisor Control Plane cung cấp local tool surface (bridge-not-brain).
   - Ở các phase P04/P05, các công cụ đọc log và chạy verification profile phải được đối soát nghiêm ngặt với `docs/02_REQUIREMENTS.md`, `docs/11_TOOL_DEFINITIONS.md`, và `docs/07_SECURITY_MODEL.md` (SEC-003).
   - **Tuyệt đối KHÔNG mở arbitrary shell execution**; mọi lệnh kiểm thử phải qua runner độc lập có whitelist và timeout cứng.

---

## 4. Ma trận Kiểm chứng Runtime trên Binary Thật & Invariants

| Kịch bản Kiểm chứng | Hành vi & Trạng thái Mong đợi | Phương pháp Kiểm tra Thực tế |
|---|---|---|
| **1. Trước Run** | `ProcessOwnerLease` được acquire. Listener chưa bind socket. | Network probe: Socket connect bị từ chối (Connection Refused). |
| **2. Đang Run** | `Runner.Run(ctx)` nắm `ExclusiveScope`. Listener tiếp tục đóng. | Network probe: Socket connect bị từ chối; log chỉ bổ trợ. |
| **3. Sau Run (Complete)** | `r.ready = true`. Pair guards sạch. Listener bind port thành công. | HTTP probe tới listener port trả về `200 OK` (Admission OPEN). |
| **4. Sau Run (PendingAO)** | `r.ready = true`. Listener mở; Pair có hold trả về 409/503. | HTTP probe verify Pair sạch = 200, Pair hold = 409/503; Poller chạy nền. |
| **5. Run Lỗi / Incomplete** | `r.ready = false`. Admission đóng fail-closed; daemon exit non-zero. | Socket không bao giờ mở; process exit code != 0. |
| **6. Cạnh tranh 2 Process** | Process B phát hiện Process A đang giữ lock; cooperative takeover an toàn. | Process A drain và exit; Process B tiếp quản; nếu A không thoát thì B fail-closed. |
| **7. Shutdown Drain** | Nhận SIGINT: listener đóng -> poller drain -> timeout permit release -> đóng Store. | Probe socket đóng ngay; verify mọi in-flight connection hoàn tất trước khi DB đóng. |
| **8. P03 Exit Gate** | 5 bước: session create, dispatch, observation reconciliation, raw file read, teardown. | Chạy trọn vẹn kịch bản tự động mà không import/gọi bất kỳ code P04 nào. |

- **Tách bạch Mock vs. Live AO**: Tách biệt hoàn toàn test Mock AO tự động chạy trong CI/harness khỏi bài kiểm chứng Live AO thực tế được kiểm soát qua gate riêng.
- **Invariants Bất biến**:
  - `AUTOMATIC_RESTORE = DISABLED`: Giữ nguyên disable fail-closed.
  - 8 Operational policies tiếp tục giữ nguyên ở trạng thái `UNSET` trong tài liệu; runtime bắt buộc phải được inject giá trị cấu hình hợp lệ khi khởi động, thiếu thì fail-closed ngay lập tức.
