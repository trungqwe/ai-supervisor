# DRAFT ADR-017 — Host Quiescence & Daemon Lifecycle Architecture

> **Status:** `REVISION_3_PENDING_SUPERVISOR_REAUDIT`
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

3. **Thuật toán Định Danh Canonical DB & Lock Path Duy Nhất Trước Store.Open**:
   Tuyệt đối không dùng `filepath.Clean` thông thường làm giải pháp thay thế, vì các đường dẫn tương đối, symlinks, directory junctions, subst drives, hard links, và sự khác biệt chữ hoa/thường trên Windows sẽ tạo ra các chuỗi path khác nhau cho cùng một physical database. Thuật toán định danh bắt buộc tuân thủ quy trình sau:
   - **Trường hợp A: Database file ĐÃ TỒN TẠI trên đĩa**:
     1. Mở handle tới DB file bằng `CreateFileW` với `dwDesiredAccess = FILE_READ_ATTRIBUTES`, `dwShareMode = FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE`, `dwCreationDisposition = OPEN_EXISTING`, `dwFlagsAndAttributes = FILE_ATTRIBUTE_NORMAL`.
     2. Kiểm tra Hard Link Alias: Gọi `GetFileInformationByHandle(hDB, &info)`. Nếu `info.nNumberOfLinks > 1`, database file đang có hard link alias (hai hoặc nhiều tên file trỏ tới cùng dữ liệu trên MFT). Do hard link alias không thể bảo đảm tính duy nhất của path-based lock file, daemon **FAIL-CLOSED NGAY LẬP TỨC** (`ERR_HARDLINK_ALIAS_UNSUPPORTED`).
     3. Gọi `GetFinalPathNameByHandleW(hDB, pathBuf, len, VOLUME_NAME_DOS)` để nhận đường dẫn chuẩn hóa tuyệt đối (canonical NT DOS path, e.g. `\?\C:\path\db.sqlite`).
     4. Đóng handle `hDB`.
   - **Trường hợp B: Database file CHƯA TỒN TẠI (DB Mới)**:
     1. Tách đường dẫn thành thư mục cha (`parent_dir = filepath.Dir(input_path)`) và tên file cơ sở (`base_name = filepath.Base(input_path)`).
     2. Mở handle tới `parent_dir` (bắt buộc phải tồn tại) bằng `CreateFileW` với `FILE_FLAG_BACKUP_SEMANTICS` (cần thiết cho directory handle trên Windows).
     3. Gọi `GetFinalPathNameByHandleW(hDir, dirBuf, len, VOLUME_NAME_DOS)` để nhận canonical path của thư mục cha.
     4. Ghép canonical parent directory với `base_name` (được chuẩn hóa lowercase/case-folded): `<canonical_db_path> = canonicalDir + "\" + base_name`.
     5. Đóng handle `hDir`.
   - **Quy tắc Fail-Closed khi Gặp Alias Không Thể Chứng Minh**:
     Nếu `GetFinalPathNameByHandleW` thất bại, hoặc volume filesystem không phải là NTFS/ReFS cục bộ (ví dụ network share SMB/UNC), hoặc alias không thể chứng minh duy nhất: daemon **FAIL-CLOSED NGAY LẬP TỨC**, tuyệt đối không fallback sang naive normalization.
   - **Đường dẫn Lock File**: `<canonical_db_path>.owner.lock`.

4. **Bảo Mật Metadata `.owner.json` & Quản Lý Trạng Thái**:
   - File metadata `<canonical_db_path>.owner.json` là kênh truyền thông tin nhưng **KHÔNG ĐƯỢC TIN CẬY MÙ QUÁNG** khi chưa xác thực.
   - **Ghi Atomically**: Sau khi acquire thành công `.owner.lock`, owner ghi nội dung metadata gồm: `pid`, `owner_instance_id` (cryptographically secure random UUIDv4), `pipe_name`, `started_at`, `canonical_db_path` vào file tạm `<canonical_db_path>.owner.json.tmp`, sau đó thực hiện atomic replace qua `MoveFileExW(..., MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH)`.
   - **DACL & Phân Quyền**: File `.owner.json` có explicit DACL chỉ cho phép Owner SID, Administrators và SYSTEM được ghi; các tiến trình khác chỉ có quyền đọc (`FILE_SHARE_READ`).
   - **Xử lý Stale / Corrupt Metadata**: Khi Process B gặp `ERROR_SHARING_VIOLATION` khi mở `.owner.lock`, Process B đọc `.owner.json`. Nếu file không tồn tại, bị rỗng, JSON bị hỏng (corrupt), hoặc trường `canonical_db_path` không khớp: Process B coi metadata là không đáng tin cậy. Vì lock file chắc chắn đang bị giữ ở kernel, Process B **FAIL-CLOSED NGAY LẬP TỨC**; tuyệt đối không tự suy đoán PID từ OS để can thiệp.

5. **Xác Thực Named Pipe Takeover & Chống DoS**:
   - Named Pipe `\\.\pipe\<pipe_name>` được tạo với security descriptor chỉ cấp quyền cho current user SID (hoặc authorized operator principal).
   - Khi Process B gửi request cooperative takeover qua Named Pipe:
     * Request phải mang cấu trúc có kèm `owner_instance_id` khớp với instance ID trong `.owner.json`.
     * Process A (owner hiện tại) gọi `ImpersonateNamedPipeClient` / `GetNamedPipeHandleStateW` để xác thực caller SID trùng khớp với owner SID hoặc authorized operator.
     * Mọi request không hợp lệ, không đúng SID, sai `owner_instance_id`, hoặc malformed **ĐỀU BỊ TỪ CHỐI**; tuyệt đối **KHÔNG ĐƯỢC KHIẾN OWNER SHUTDOWN**.
   - Nếu takeover request hợp lệ: Process A phản hồi ACK, đóng admission, drain/join toàn bộ callers, đóng Store, đóng handle `.owner.lock`, xóa `.owner.json`, và thoát.
   - Nếu Process A không phản hồi, không hợp tác, hoặc không thoát đúng hạn: Process B **FAIL-CLOSED NGAY LẬP TỨC**. V1 cấm tuyệt đối tự động gọi `TerminateProcess`. Mọi đề xuất auto-kill phải lập proposal riêng kèm bằng chứng an toàn.

6. **Handle Inheritance & Crash Cleanup**:
   - Mọi handle file và pipe bắt buộc tạo với `SECURITY_ATTRIBUTES{bInheritHandle: FALSE}` để chống handle leak sang child processes.
   - Kernel Windows tự động giải phóng toàn bộ file handles khi tiến trình thoát/crash, exclusive share lock tự động biến mất ngay lập tức mà không để lại stale lock.

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
   - Subtask `TASK-P03-004` chỉ giới hạn ở daemon bootstrap (`cmd/supervisor`), admission control, và integration test harness phục vụ kiểm chứng exit gate của Phase P03.
   - Tuyệt đối **LOẠI BỎ production verification runner khỏi scope Phase P03**.
2. **Phân bổ Phù hợp với Roadmap & Module Provenance**:
   - `EvidenceCollector`, `ReviewBundleBuilder`, policy validator, git diff verification runner và các công cụ thực thi profile verification thuộc sở hữu độc quyền của Phase P04 (`docs/22_MODULE_PROVENANCE.md`) và Phase P05 (`docs/21_TRACEABILITY_MATRIX.md`).

### 3.2. Ghi nhận Yêu cầu Người dùng & An ninh SEC-003
1. **Mô hình Tác nhân Suy luận vs Cầu nối Công cụ**:
   - ChatGPT Web đóng vai trò là **agent suy luận** (reasoning agent). Supervisor Control Plane đóng vai trò là **cầu nối công cụ local** (local tool bridge, bridge-not-brain).
2. **Đối soát Nghiêm ngặt**:
   - Ở các phase P04/P05, các công cụ phục vụ ChatGPT Web (bao gồm tool đọc log và tool yêu cầu chạy profile verification) bắt buộc phải đối soát với `docs/11_CHATGPT_TOOL_SURFACE.md`, `docs/02_REQUIREMENTS.md`, và `docs/07_SECURITY_MODEL.md` (SEC-003).
   - **Tuyệt đối KHÔNG cấp arbitrary shell execution**; mọi lệnh kiểm thử chỉ được thực thi thông qua verification runner độc lập có whitelist tham số và hard timeout ở Phase P04/P05.

---

## 4. Tách Bạch Hai Track Bằng Chứng & Ma trận Kiểm chứng Runtime

### 4.1. Tách Bạch Hai Loại Bằng Chứng Kiểm Chứng
1. **Xóa Giả Định Effectful Supervisor HTTP Routes**:
   - Bản thiết kế này **xóa bỏ hoàn toàn giả định** cho rằng daemon P03 đã có sẵn các HTTP routes tạo session, dispatch, đọc workspace và stop. Supervisor Control Plane ở Phase P03 CHƯA có production HTTP control routes cho các tác vụ effectful này.
   - Tuyệt đối **KHÔNG dùng HTTP routes của Untrivial AO trực tiếp** để tuyên bố "đạt" exit gate, vì làm như vậy sẽ hoàn toàn bỏ qua state machine, sagas, store guards và audit trail của Supervisor.
   - Mọi dự định bổ sung effectful Supervisor HTTP API phải được lập hồ sơ governance độc lập theo `docs/24_CHANGE_GOVERNANCE.md`; không được đưa vào phạm vi ADR-017.
2. **Track 1: Binary Daemon Thật (`cmd/supervisor`)**:
   - Chứng minh các năng lực cốt lõi của daemon host:
     * Machine-wide exclusive lock (`ProcessOwnerLease` trên `.owner.lock`).
     * Xử lý cạnh tranh hai tiến trình (trong cùng session hoặc xuyên qua 2 Windows logon sessions).
     * Cơ chế startup-before-serve: Socket admission listener bị từ chối kết nối (`Connection Refused`) trong suốt thời gian `Runner.Run(ctx)` đang sweep snapshot; chỉ mở sau khi scan hoàn tất (`r.ready == true`).
     * PendingAO admission hold: Cổng mở nhưng các Pair bị hold trả về lỗi 409/503.
     * Run Incomplete: Cổng tiếp tục đóng fail-closed, daemon dừng.
     * Shutdown drain: Drain/join toàn bộ callers trước khi đóng Store và giải phóng lock handle.
   - Admission listener của daemon ở P03 chỉ phục vụ probe trạng thái (health/readiness) hoặc cooperative takeover IPC.
3. **Track 2: P03 Integration Test Harness (Kiểm Chứng 5 Bước Exit Gate AO)**:
   - Kiểm chứng trọn vẹn 5 bước exit gate AO của Phase P03 (session creation, dispatch, observation reconciliation, raw workspace-file read, teardown).
   - **Phương thức thực thi**: Harness gọi **trực tiếp các API điều phối nội bộ đã được duyệt của thư viện Go Supervisor** (`internal/dispatch`, `internal/store`, `internal/stop`, `internal/ao`, `internal/recovery`).
   - Bảo đảm đi qua toàn bộ Supervisor saga, store guards, state transitions và audit trail, hoàn toàn không cần production HTTP control surface và không phụ thuộc vào code P04/P05.

### 4.2. Ma trận Kiểm chứng Runtime trên Binary Thật (`cmd/supervisor`)

| Kịch bản Kiểm chứng | Hành vi & Trạng thái Mong đợi | Phương pháp Kiểm tra Thực tế |
|---|---|---|
| **1. Trước Run** | `ProcessOwnerLease` được acquire (`.owner.lock` share mode 0). Listener chưa bind socket. | Network probe: Socket connect bị từ chối (Connection Refused). |
| **2. Đang Run** | `Runner.Run(ctx)` nắm `ExclusiveScope`. Listener tiếp tục đóng. | Network probe: Socket connect bị từ chối; log chỉ bổ trợ. |
| **3. Sau Run (Complete)** | `r.ready = true`. Pair guards sạch. Listener bind port thành công. | HTTP probe tới readiness probe port trả về `200 OK` (Admission OPEN). |
| **4. Sau Run (PendingAO)** | `r.ready = true`. Listener mở; Pair có hold trả về 409/503. | HTTP probe verify readiness mở nhưng Pair hold bị khóa admission. |
| **5. Run Lỗi / Incomplete** | `r.ready = false`. Admission đóng fail-closed; daemon exit non-zero. | Socket không bao giờ mở; process exit code != 0. |
| **6. Cạnh tranh 2 Process** | Process B phát hiện `.owner.lock` bị giữ (`ERROR_SHARING_VIOLATION`); cooperative takeover an toàn. | Process A drain và exit; Process B tiếp quản; nếu A không thoát thì B fail-closed. |
| **7. Shutdown Drain** | Nhận SIGINT: listener đóng -> poller drain -> timeout permit release -> Store.Close() -> đóng lock handle. | Probe socket đóng ngay; verify mọi in-flight connection hoàn tất trước khi DB đóng. |

### 4.3. Ma trận Kiểm chứng Canonical DB Identity & Alias Resolution

| Kịch bản Alias / Session | Đường dẫn Đầu vào Thử nghiệm | Kết quả Kỳ vọng |
|---|---|---|
| **Relative Path** | `.\data\db.sqlite` vs `data\..\data\db.sqlite` | Quy về cùng một canonical lock path `\\?\<Drive>:\...\data\db.sqlite.owner.lock` |
| **Case Differences** | `D:\data\db.sqlite` vs `d:\DATA\DB.SQLITE` | Quy về cùng một canonical lock path (case-folded khớp tên đĩa vật lý) |
| **Subst Drive** | `X:\db.sqlite` (với `subst X: D:\data`) | Kernel resolve về đường dẫn thực `\\?\D:\data\db.sqlite.owner.lock` |
| **Directory Junction / Symlink** | `D:\junction\db.sqlite` -> `D:\real\db.sqlite` | `GetFinalPathNameByHandleW` resolve về `\\?\D:\real\db.sqlite.owner.lock` |
| **DB Mới (Chưa tồn tại)** | File chưa có trên đĩa | Resolve canonical parent directory + lowercase base name |
| **Hard Link Detection** | File có `nNumberOfLinks > 1` | Bị từ chối ngay lập tức: **FAIL-CLOSED** (`ERR_HARDLINK_ALIAS_UNSUPPORTED`) |
| **Cross-Session Competition** | Session 0 (Service) vs Session 1 (Interactive) cùng trỏ DB | Process thứ hai nhận ngay `ERROR_SHARING_VIOLATION` (32) xuyên session |

### 4.4. Invariants Bất biến
- `AUTOMATIC_RESTORE = DISABLED`: Giữ nguyên disable fail-closed.
- 8 Operational policies tiếp tục giữ nguyên ở trạng thái **`UNSET`** trong tài liệu; runtime bắt buộc phải được inject giá trị cấu hình hợp lệ khi khởi động, thiếu bất kỳ giá trị nào thì fail-closed ngay lập tức.
- Tách biệt hoàn toàn test Mock AO tự động trong CI khỏi bài kiểm chứng Live AO có kiểm soát.
