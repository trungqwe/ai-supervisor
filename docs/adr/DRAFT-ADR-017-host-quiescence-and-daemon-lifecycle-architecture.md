# DRAFT ADR-017 — Host Quiescence & Daemon Lifecycle Architecture

> **Status:** `REVISION_2_PENDING_SUPERVISOR_REAUDIT`
> **Authority:** `docs/24_CHANGE_GOVERNANCE.md` (Level 2 Approved ADR)
> **Liên quan:** ADR-001, ADR-007, ADR-016 (D8, D9, D11, D12, D13), ADR-016 Addenda, `PROPOSAL-P03-005`.
> **Phạm vi:** Kiến trúc Host Quiescence, Windows Exclusivity, Daemon Lifecycle và ranh giới Phase P03/P04.

---

## 1. Bối cảnh & Động lực Kiến trúc

Phase P03 đã hoàn tất nghiệm thu toàn bộ các mô-đun thư viện Go nội bộ quản lý giao tiếp với Untrivial Agent Orchestrator (AO), session lifecycle, crash recovery scanner, execution budget và timeout monitor (các subtask 3A–3D).

Tuy nhiên, `docs/17_ROADMAP.md` quy định exit gate tổng thể của Phase P03 đòi hỏi phải kiểm chứng tự động vòng đời phiên làm việc mà không phụ thuộc vào Phase P04. Hiện tại codebase chưa có daemon entrypoint (`cmd/`), chưa có cơ chế kiểm soát độc quyền liên tiến trình toàn máy Windows (machine-wide cross-process exclusivity), và chưa có bằng chứng runtime về việc scanner hoàn tất trước khi mở cổng nhận request (`DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`).

Đồng thời, Phase P04 được quy hoạch chuyên biệt cho Evidence & Review Engine (`EvidenceCollector`, `ReviewBundleBuilder`, policy validator), hoàn toàn không sở hữu daemon bootstrap. Do đó, cần có một Architecture Decision Record chính thức quy định kiến trúc Host Quiescence và Daemon Lifecycle cho subtask `TASK-P03-004` trong Phase P03.

---

## 2. Quyết định Kiến trúc

### 2.1. Bổ sung Subtask `TASK-P03-004` vào Phase P03
Quyết định bổ sung subtask `TASK-P03-004 Host Quiescence & Daemon Bootstrap Integration` thuộc Phase P03 để hoàn tất exit gate P03 trước khi mở Phase P04.

### 2.2. Phân Tách Rõ Ràng Hai Khái Niệm Quiescence & Lease
1. **`ProcessOwnerLease` (Dài hạn, cấp Process/Host)**:
   - Phạm vi: Toàn bộ database (gắn với canonical DB path trên máy Windows).
   - Vòng đời: Giữ độc quyền từ **TRƯỚC** khi gọi `Store.Open()` hoặc thực thi DB migration, xuyên suốt quá trình daemon hoạt động, cho đến **SAU** khi shutdown drain hoàn tất và `Store.Close()` đã đóng kết nối database.
   - Mục đích: Bảo đảm tại một thời điểm trên toàn bộ máy Windows chỉ có DUY NHẤT một tiến trình daemon sở hữu database (Single Active Daemon).
2. **`ExclusiveScope` (Ngắn hạn, cấp Recovery Sweep)**:
   - Phạm vi: Được định nghĩa tại `internal/recovery/scanner.go` (`HostQuiescence.Acquire(ctx)`).
   - Vòng đời: `Runner.Run(ctx)` **tự động gọi** `HostQuiescence.Acquire(ctx)` để lấy `ExclusiveScope` khi bắt đầu sweep và gọi `scope.Release()` ngay khi sweep snapshot hoàn tất.
   - Tách biệt: Khi `ExclusiveScope.Release()` được gọi, `ProcessOwnerLease` **vẫn tiếp tục được giữ** bởi daemon cho đến khi daemon shutdown hoàn toàn. Entrypoint tuyệt đối không gọi `HostQuiescence.Acquire()` lần hai.

### 2.3. Cơ chế Machine-Wide Exclusivity trên Windows (Single Windows Host, Local Filesystem)
1. **Loại bỏ `Local\` Named Mutex khỏi Bằng chứng Exclusivity**:
   - Trong Windows OS, các đối tượng trong namespace `Local\` bị cô lập theo Windows Logon Session (Session 0 dành cho Windows Services / Task Scheduler, Session 1+ dành cho người dùng tương tác, SSH sessions, Fast User Switching). Một named mutex tạo trong `Local\` không thể ngăn chặn một tiến trình ở session khác hoặc elevated session truy cập cùng DB.
   - Đồng thời, Named Mutex dễ bị name squatting / pre-creation DoS từ các tiến trình không có đặc quyền. Do đó, V1 **không dùng `Local\` named mutex** làm bằng chứng độc quyền.
2. **Khóa Độc quyền Chính: Windows Exclusive Sidecar Lock File Handle**:
   - V1 sử dụng một **Windows Exclusive Sidecar Lock File Handle** mở bằng API hệ điều hành `CreateFileW` với cờ `dwShareMode = 0` (exclusive, không chia sẻ đọc, không chia sẻ ghi, không chia sẻ xóa).
   - Đường dẫn lock file: `<canonical_db_path>.owner.lock`.
   - Vòng đời: Handle này được mở từ **TRƯỚC** khi gọi `Store.Open()` hoặc chạy migration, và được giữ liên tục trong bộ nhớ tiến trình cho đến **SAU** khi shutdown drain hoàn tất và `Store.Close()` đã đóng kết nối database.
   - Tính chất Machine-Wide: Khóa độc quyền qua filesystem filter manager của Windows có hiệu lực toàn máy (machine-wide), áp dụng cho mọi Windows logon session, mọi user account và mọi mức đặc quyền.
3. **Tách Biệt File Metadata & IPC Carrier**:
   - Đường dẫn metadata: `<canonical_db_path>.owner.json`.
   - Mục đích: Chứa thông tin nhận diện owner (`pid`, `owner_instance_id`, `pipe_name`, `started_at`, `windows_session_id`).
   - Mở với chế độ `FILE_SHARE_READ | FILE_SHARE_WRITE` để các tiến trình khác trên máy có thể đọc thông tin liên lạc ngay cả khi `.owner.lock` đang bị khóa độc quyền với `dwShareMode = 0`.
4. **Canonical DB / Lock Identity**:
   - Đường dẫn database và lock file được chuẩn hóa canonical qua Windows API `GetFinalPathNameByHandleW` (hoặc `filepath.Clean` kết hợp resolve symlink/junction và volume letter uppercase) để đảm bảo hai tiến trình dù trỏ bằng đường dẫn tương đối, chữ hoa/thường khác nhau hay subst drive đều quy về một canonical identity duy nhất: `<canonical_db_path>.owner.lock`.
5. **Local Filesystem Được Hỗ Trợ**:
   - Chỉ hỗ trợ các hệ thống tệp cục bộ: **NTFS** và **ReFS** trên ổ đĩa cố định cục bộ.
   - Cấm tuyệt đối chạy DB và lock file trên network filesystems (SMB, CIFS, NFS, UNC shares) do cơ chế oplocks, leasing và caching của mạng làm mất tính tin cậy của share locking.
6. **Handle Inheritance (Cấm Kế thừa Handle)**:
   - Mọi handle file và IPC pipe bắt buộc phải được tạo với thuộc tính bảo mật `SECURITY_ATTRIBUTES{bInheritHandle: FALSE}`. Điều này ngăn chặn các tiến trình con (như `git` sub-process) vô tình kế thừa handle làm rò rỉ lock sau khi daemon chính kết thúc.
7. **Crash Cleanup (Kernel Quản lý Vòng đời)**:
   - Hệ điều hành Windows kernel tự động đóng toàn bộ file handle khi tiến trình kết thúc vì bất kỳ lý do gì (bình thường, crash, terminate, out of memory, crashdump). Khi handle của lock file đóng, exclusive share lock tự động biến mất ngay lập tức mà không để lại stale lock state.
8. **Cạnh tranh giữa Hai Windows Logon Sessions**:
   - Khi Process A (ví dụ Session 1) đang nắm `.owner.lock`: nếu Process B (ví dụ Session 0) cố gắng mở `.owner.lock` với `dwShareMode = 0`, kernel Windows lập tức trả về mã lỗi `ERROR_SHARING_VIOLATION` (32) xuyên session.
   - Process B nhận diện lock đang bị giữ, đọc file `.owner.json`, lấy `pipe_name` và kết nối tới Windows Named Pipe để yêu cầu cooperative takeover.
9. **Cooperative Takeover V1 & Ràng buộc Fail-Closed**:
   - V1 áp dụng duy nhất cơ chế **Cooperative Takeover** qua Windows Named Pipe.
   - Process A nhận tín hiệu yêu cầu: đóng admission, drain/join toàn bộ callers, đóng Store, đóng handle lock file, và tự thoát.
   - **Ràng buộc Fail-Closed**: Nếu Process A không phản hồi, không hợp tác hoặc không thoát trong thời hạn chờ, Process B **FAIL-CLOSED NGAY LẬP TỨC**. Tuyệt đối không tự ý gọi `TerminateProcess` trong V1. Mọi cơ chế auto-kill phải lập proposal riêng kèm bằng chứng an toàn.

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
5. Sau khi `Store.Close()` hoàn tất, đóng Windows exclusive lock file handle (`.owner.lock`) và xóa file metadata (`.owner.json`) để giải phóng `ProcessOwnerLease`.

---

## 3. Ranh giới Tooling P03/P04/P05 & An ninh SEC-003

### 3.1. Loại bỏ Production Verification Runner khỏi Scope P03
1. **Giới hạn Scope P03 Host**:
   - Subtask `TASK-P03-004` chỉ giới hạn ở daemon bootstrap (`cmd/supervisor`), admission control, và test harness phục vụ kiểm chứng 5 bước exit gate của Phase P03.
   - Tuyệt đối **LOẠI BỎ production verification runner khỏi scope Phase P03**.
2. **Phân bổ Phù hợp với Roadmap & Module Provenance**:
   - `EvidenceCollector`, `ReviewBundleBuilder`, policy validator, git diff verification runner và các công cụ thực thi profile verification thuộc sở hữu độc quyền của Phase P04 (`docs/22_MODULE_PROVENANCE.md`) và Phase P05 (`docs/21_TRACEABILITY_MATRIX.md`).
   - Việc tách bạch này đảm bảo Phase P03 không bị phình to phạm vi (scope creep) và tuân thủ chặt chẽ ranh giới module provenance đã được phê duyệt.

### 3.2. Ghi nhận Yêu cầu Người dùng & An ninh SEC-003
1. **Mô hình Tác nhân Suy luận vs Cầu nối Công cụ**:
   - ChatGPT Web đóng vai trò là **agent suy luận** (reasoning agent). Supervisor Control Plane đóng vai trò là **cầu nối công cụ local** (local tool bridge, bridge-not-brain).
2. **Đối soát Nghiêm ngặt**:
   - Ở các phase P04/P05, các công cụ phục vụ ChatGPT Web (bao gồm tool đọc log và tool yêu cầu chạy profile verification) bắt buộc phải đối soát với `docs/11_CHATGPT_TOOL_SURFACE.md`, `docs/02_REQUIREMENTS.md`, và `docs/07_SECURITY_MODEL.md` (SEC-003).
   - **Tuyệt đối KHÔNG cấp arbitrary shell execution**; mọi lệnh kiểm thử chỉ được thực thi thông qua verification runner độc lập có whitelist tham số và hard timeout ở Phase P04/P05.

---

## 4. Ma trận Kiểm chứng Runtime trên Binary Thật & Kích hoạt Exit Gate P03

### 4.1. Ma trận Kiểm chứng Runtime trên Binary Thật (`cmd/supervisor`)

| Kịch bản Kiểm chứng | Hành vi & Trạng thái Mong đợi | Phương pháp Kiểm tra Thực tế |
|---|---|---|
| **1. Trước Run** | `ProcessOwnerLease` được acquire (`.owner.lock` share mode 0). Listener chưa bind socket. | Network probe: Socket connect bị từ chối (Connection Refused). |
| **2. Đang Run** | `Runner.Run(ctx)` nắm `ExclusiveScope`. Listener tiếp tục đóng. | Network probe: Socket connect bị từ chối; log chỉ bổ trợ. |
| **3. Sau Run (Complete)** | `r.ready = true`. Pair guards sạch. Listener bind port thành công. | HTTP probe tới listener port trả về `200 OK` (Admission OPEN). |
| **4. Sau Run (PendingAO)** | `r.ready = true`. Listener mở; Pair có hold trả về 409/503. | HTTP probe verify Pair sạch = 200, Pair hold = 409/503; Poller chạy nền. |
| **5. Run Lỗi / Incomplete** | `r.ready = false`. Admission đóng fail-closed; daemon exit non-zero. | Socket không bao giờ mở; process exit code != 0. |
| **6. Cạnh tranh 2 Process** | Process B phát hiện `.owner.lock` bị giữ (`ERROR_SHARING_VIOLATION`); cooperative takeover an toàn. | Process A drain và exit; Process B tiếp quản; nếu A không thoát thì B fail-closed. |
| **7. Shutdown Drain** | Nhận SIGINT: listener đóng -> poller drain -> timeout permit release -> Store.Close() -> đóng lock handle. | Probe socket đóng ngay; verify mọi in-flight connection hoàn tất trước khi DB đóng. |
| **8. P03 Exit Gate** | 5 bước: session create, dispatch, observation reconciliation, raw file read, teardown. | Test harness kích hoạt trọn vẹn kịch bản mà không import/phụ thuộc code P04. |

### 4.2. Cơ chế Kích hoạt 5 Bước Exit Gate P03 khi Chưa có 12 Tool P05
Do bộ 12 công cụ domain của Phase P05 chưa tồn tại, việc kích hoạt và kiểm chứng 5 bước Exit Gate của Phase P03 được thực hiện thông qua **Exit Gate Test Harness**:
1. **Khởi động Daemon**: Daemon `cmd/supervisor` khởi động, acquire `ProcessOwnerLease` (`.owner.lock`), thực thi `Runner.Run(ctx)` để sweep DB snapshot.
2. **Kiểm tra Startup-Before-Serve**: Network probe kiểm tra cổng HTTP cục bộ: socket bị từ chối cho đến khi `Runner.ready == true`, sau đó listener mở cổng.
3. **Kích hoạt 5 Bước qua HTTP Admission Endpoints**: Test harness gửi HTTP request trực tiếp vào daemon:
   - **Bước 1 (Session Create)**: Gửi request tạo session mới.
   - **Bước 2 (Dispatch)**: Gửi dispatch prompt cho session đã tạo.
   - **Bước 3 (Observation Reconciliation)**: Poller nền quét hoặc trigger observation probe để đối soát trạng thái session.
   - **Bước 4 (Raw Workspace-File Read)**: Gọi endpoint đọc nội dung file raw từ workspace.
   - **Bước 5 (Teardown)**: Gửi request dừng và thu hồi tài nguyên session.
4. **Bằng chứng Mạng vs Log**: Bằng chứng đạt yêu cầu là network probe và HTTP status code trả về thực tế; dữ liệu log chỉ mang tính chất bổ trợ phân tích.

### 4.3. Invariants Bất biến
- `AUTOMATIC_RESTORE = DISABLED`: Giữ nguyên disable fail-closed.
- 8 Operational policies tiếp tục giữ nguyên ở trạng thái **`UNSET`** trong tài liệu; runtime bắt buộc phải được inject giá trị cấu hình hợp lệ khi khởi động, thiếu bất kỳ giá trị nào thì fail-closed ngay lập tức.
- Tách biệt hoàn toàn test Mock AO tự động trong CI khỏi bài kiểm chứng Live AO có kiểm soát.
