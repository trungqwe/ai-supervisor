# DRAFT ADR-017 — Host Quiescence & Daemon Lifecycle Architecture

> **Status: PROPOSED / REVISION_6_PENDING_EXTERNAL_AUDIT (Audit: docs/audits/P03_ADR_017_EXTERNAL_REAUDIT_005.md)
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
   - Phạm vi: Toàn bộ database (gắn với canonical lock key trên máy Windows).
   - Vòng đời: Giữ độc quyền từ **TRƯỚC** khi gọi `Store.Open()` hoặc thực thi DB migration, xuyên suốt quá trình daemon hoạt động, cho đến **SAU** khi shutdown drain hoàn tất và `Store.Close()` đã đóng kết nối database.
   - Mục đích: Bảo đảm tại một thời điểm trên toàn bộ máy Windows chỉ có DUY NHẤT một tiến trình daemon sở hữu database (Single Active Daemon).
2. **`ExclusiveScope` (Ngắn hạn, cấp Recovery Sweep)**:
   - Phạm vi: Được định nghĩa tại `internal/recovery/scanner.go` (`HostQuiescence.Acquire(ctx)`).
   - Vòng đời: `Runner.Run(ctx)` **tự động gọi** `HostQuiescence.Acquire(ctx)` để lấy `ExclusiveScope` khi bắt đầu sweep và gọi `scope.Release()` ngay khi sweep snapshot hoàn tất.
   - Tách biệt: Khi `ExclusiveScope.Release()` được gọi, `ProcessOwnerLease` **vẫn tiếp tục được giữ** bởi daemon cho đến khi daemon shutdown hoàn toàn. Entrypoint tuyệt đối không gọi `HostQuiescence.Acquire()` lần hai.

### 2.3. Cơ chế Machine-Wide Exclusivity trên Windows (Single Windows Host, Local Filesystem)

1. **Hợp Đồng `CreateFileW` Cho `.owner.lock`**:
   Khóa độc quyền chính là một Windows Exclusive Sidecar Lock File Handle mở trên file `<canonical_db_path>.owner.lock` với bộ tham số API kernel bất biến:
   - `dwDesiredAccess = GENERIC_READ | GENERIC_WRITE`: Yêu cầu quyền truy cập non-zero xung đột trực tiếp giữa các tiến trình cạnh tranh.
   - `dwShareMode = 0`: Chế độ độc quyền hoàn toàn (không chia sẻ đọc, không chia sẻ ghi, không chia sẻ xóa).
   - `dwCreationDisposition = OPEN_ALWAYS`: Tạo file nếu chưa tồn tại; mở file hiện có mà **TUYỆT ĐỐI KHÔNG TRUNCATE** file lock đang bị tiến trình khác nắm giữ.
   - `dwFlagsAndAttributes = FILE_ATTRIBUTE_NORMAL`: Kết hợp thuộc tính chuẩn của hệ thống tệp.
   - `lpSecurityAttributes.bInheritHandle = FALSE`: Ngăn chặn handle leak sang child process.
   - **Xử lý lỗi**: Mọi lỗi acquire (như `ERROR_SHARING_VIOLATION` (32), `ERROR_ACCESS_DENIED` (5)) đều được xử lý **FAIL-CLOSED** ngay lập tức.

2. **Phân Biệt Rành Mạch: Canonical Lock Key vs. Physical File Identity**:
   - **Canonical Lock Key (Path-based)**: Chuỗi đường dẫn canonical NT DOS `<canonical_db_path>.owner.lock` được xác lập trước khi mở DB nhằm thiết lập cơ chế loại trừ tương hỗ (mutual exclusion) giữa các tiến trình trên filesystem.
   - **Physical File Identity (Volume + File ID)**: Cặp định danh vật lý `(VolumeSerialNumber, FileId)` đại diện cho chính xác thực thể inode/file trên volume đĩa vật lý của Windows.
     * Trên **NTFS**: Sử dụng `FILE_ID_INFO` (128-bit `FileId` + `VolumeSerialNumber`) qua `GetFileInformationByHandleEx`, hoặc `BY_HANDLE_FILE_INFORMATION` (64-bit FileIndex).
     * Trên **ReFS**: ReFS sử dụng không gian 128-bit File ID; API cũ `BY_HANDLE_FILE_INFORMATION` chỉ trả về 64-bit bị cắt ngắn hoặc không ổn định. Do đó, trên ReFS **BẮT BUỘC** sử dụng cấu trúc 128-bit `FILE_ID_INFO` (`GetFileInformationByHandleEx(h, FileIdInfo, &info, sizeof(info))`).

3. **Phép Đối Chiếu Identity Trước và Sau `Store.Open()`**:
   - **Trường hợp A: Database file ĐÃ TỒN TẠI trước `Store.Open()`**:
     1. *Pre-Open*: Host mở handle DB file (`FILE_READ_ATTRIBUTES`, share all, `OPEN_EXISTING`). Giữ handle đủ lâu để:
        - Kiểm tra hard links: Gọi `GetFileInformationByHandle`. Nếu `nNumberOfLinks > 1` (DB có hard link alias), daemon **FAIL-CLOSED NGAY LẬP TỨC** (`ERR_HARDLINK_ALIAS_UNSUPPORTED`).
        - Trích xuất định danh vật lý: `(preVolume, preFileId128)` qua `GetFileInformationByHandleEx(FileIdInfo)`.
        - Lấy canonical path: `GetFinalPathNameByHandleW(hDB, VOLUME_NAME_DOS)` làm cơ sở tạo lock key `<canonical_db_path>.owner.lock`.
        - Đóng handle DB.
     2. *Acquire Lock*: Mở `.owner.lock` bằng hợp đồng `CreateFileW` độc quyền.
     3. *Post-Open Verification*: Ngay sau khi `Store.Open()` mở kết nối SQLite, host mở handle kiểm tra tới file DB mà SQLite đang giữ, gọi `GetFileInformationByHandleEx(FileIdInfo)` lấy `(postVolume, postFileId128)`.
     4. *So sánh*: `postVolume == preVolume && postFileId128 == preFileId128`. Nếu có bất kỳ mismatch nào (do symlink swap, directory redirection, hoặc alias mismatch), host lập tức gọi `Store.Close()`, đóng lock handle, và **FAIL-CLOSED** ngay lập tức trước khi mở bất kỳ listener hay effect nào.
   - **Trường hợp B: Database file CHƯA TỒN TẠI (DB Mới)**:
     1. *Pre-Open*: Thư mục cha bắt buộc phải tồn tại. Host mở handle tới thư mục cha với cờ `FILE_FLAG_BACKUP_SEMANTICS`.
        - Trích xuất `parentVolume` qua `GetFileInformationByHandleEx(FileIdInfo)`.
        - Lấy canonical parent dir qua `GetFinalPathNameByHandleW`.
        - Ghép canonical parent dir với lowercase base name thành `<canonical_db_path>`, suy ra `<canonical_db_path>.owner.lock`.
        - Đóng handle thư mục cha.
     2. *Acquire Lock*: Mở `.owner.lock` độc quyền.
     3. *Post-Open Verification*: Sau khi `Store.Open()` tạo mới và mở file DB, host mở handle tới file DB mới tạo, kiểm tra:
        - `dbVolume == parentVolume`.
        - Đường dẫn canonical của file mới qua `GetFinalPathNameByHandleW` khớp chính xác với `<canonical_db_path>`.
        - `nNumberOfLinks == 1`.
        - Nếu mismatch hoặc hardlink xuất hiện: lập tức `Store.Close()` và **FAIL-CLOSED**.
   - **Quy tắc Fail-Closed Tuyệt Đối**: Bất kỳ sự cố nào khi trích xuất handle/identity (GetFinalPathNameByHandleW lỗi, FileIdInfo lỗi, hard links > 1, network filesystem SMB/UNC) đều khiến daemon **FAIL-CLOSED NGAY LẬP TỨC**.

4. **Cảnh Báo Về Alias & Ma Trận Kiểm Chứng Hai Tiến Trình (Two-Process Probe Matrix)**:
   - Kiến trúc **không bao giờ khẳng định võ đoán** rằng subst drive, directory junction, hay casing luôn tự động hội tụ nếu chưa có kết quả probe thực nghiệm.
   - Các trường hợp này được quy định thành các ca kiểm thử bắt buộc trong **Ma Trận Kiểm Chứng Hai Tiến Trình**:
     * Contender A mở DB qua đường dẫn canonical gốc.
     * Contender B mở DB qua đường dẫn alias (subst drive, junction, hoặc casing khác).
     * Probe kiểm tra thực tế: Contender B bắt buộc phải nhận lỗi `ERROR_SHARING_VIOLATION` (32) vì cả hai giải quyết về cùng một `(Volume, FileId)` và cùng một canonical lock key.

5. **Bảo Mật Metadata `.owner.json` & Phân Định Quyền Hạn**:
   - File metadata `<canonical_db_path>.owner.json` chứa: `pid`, `owner_instance_id` (cryptographic UUIDv4), `pipe_name`, `started_at`, `canonical_db_path`.
   - **Ghi Atomically**: Owner ghi vào `<canonical_db_path>.owner.json.tmp`, thiết lập DACL chỉ cho Owner SID, Administrators và SYSTEM; sau đó atomic rename qua `MoveFileExW(..., MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH)`.
   - **Xử lý Stale / Corrupt Metadata**: File metadata không đáng tin cậy nếu chưa xác thực. Nếu file thiếu, rỗng, corrupt JSON, hoặc mismatch `canonical_db_path`: contender coi metadata là invalid, **FAIL-CLOSED NGAY LẬP TỨC**; tuyệt đối không suy đoán PID để can thiệp.
   - **Ghi đè Sau Crash**: Owner mới sau crash, sau khi đã **acquire thành công `.owner.lock`**, được phép ghi đè atomically metadata mới thay thế metadata stale cũ.

6. **Xác Thực Named Pipe Takeover & Revert Context**:
   - Named Pipe `\\.\pipe\<pipe_name>` có security descriptor chỉ cấp quyền cho current user SID (hoặc authorized operator principal).
   - Khi nhận takeover request:
     1. Owner gọi `ImpersonateNamedPipeClient(hPipe)` và kiểm tra lỗi trả về.
     2. Lấy caller token (`OpenThreadToken`), đối chiếu caller SID có trùng khớp với owner SID hoặc authorized operator hay không.
     3. Bắt buộc gọi `RevertToSelf()` trong khối cleanup/defer để hoàn nguyên security context của thread và kiểm tra lỗi của `RevertToSelf()`.
     4. `owner_instance_id` truyền trong payload **CHỈ LÀ FRESHNESS MARKER** (chống replay stale request tới instance cũ), **KHÔNG PHẢI SECRET XÁC THỰC**.
     5. Yêu cầu không đúng SID hoặc sai freshness marker bị từ chối; tuyệt đối **KHÔNG ĐƯỢC KHIẾN OWNER SHUTDOWN**.

7. **Thứ Tự Shutdown Drain & Giải Phóng Lock Cuối Cùng**:
   Khi Process A dừng (bình thường hoặc takeover hợp lệ):
   1. Đóng Named Pipe listener để chặn request takeover mới.
   2. Đóng listener admission (readiness probe).
   3. Dừng Poller và TimeoutMonitor scheduler, drain/join toàn bộ callers.
   4. Đóng kết nối cơ sở dữ liệu: `Store.Close()`.
   5. Xóa file `.owner.json` **CHỈ KHI** `owner_instance_id` trong file vẫn khớp với instance ID của Process A (tránh xóa nhầm metadata của process khác).
   6. **ĐÓNG LOCK HANDLE (`.owner.lock`) CUỐI CÙNG** sau khi Store đã đóng và sau khi đã dọn metadata.
   7. Tuyệt đối **KHÔNG XÓA METADATA SAU KHI ĐÃ ĐÓNG LOCK HANDLE** (tránh race condition xóa nhầm metadata của owner mới vừa acquire lock).

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
   - **Complete (`report.Complete == true && err == nil`)**: Phân loại hoàn tất. `Runner` tự đánh dấu trường nội bộ `r.ready = true`. Host kiểm tra Pair guards; khởi động `Poller.Start(ctx)`; khởi động scheduler gọi `TimeoutMonitor.Tick(ctx)`; mở listener readiness probe.
   - **PendingAO (`report.Complete == true && report.PendingAO == true`)**: Phân loại hoàn tất nhưng có session được đưa vào Pair hold. `Runner` tự đánh dấu `r.ready = true`. Host khởi động `Poller.Start(ctx)`. Việc hold được bảo vệ bằng Store guards bên trong thư viện.
   - **Incomplete (`report.Complete == false` hoặc `err != nil`)**: Gặp lỗi, context cancel, hoặc attempt thiếu execution budget. `Runner` giữ `r.ready = false`. Listener tiếp tục ĐÓNG fail-closed, daemon dừng phục vụ.
3. **Phân tách Trách nhiệm**:
   - Host điều khiển listener readiness probe từ bên ngoài; không gán trường private `r.ready` của `Runner`.
4. **`TimeoutMonitor` Scheduling**:
   - `TimeoutMonitor` chỉ có phương thức `Tick(ctx)`. Host daemon chịu trách nhiệm thiết lập ticker scheduler định kỳ gọi `Tick(ctx)`.

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
1. **Track 1: Binary Daemon Thật (`cmd/supervisor`)**:
   - Chứng minh machine-wide exclusivity (`.owner.lock` với cờ `CreateFileW` chuẩn), cạnh tranh hai process, startup-before-serve và shutdown drain.
   - **Phạm vi Readiness Probe**: Probe cổng HTTP của daemon **CHỈ CHỨNG MINH** socket đóng (Connection Refused) trước Run và mở (200 OK) sau Run Complete (`r.ready == true`).
   - Tuyệt đối **KHÔNG thêm Pair HTTP routes hay effectful Supervisor API** vào daemon P03.
2. **Track 2: P03 Integration Test Harness (Kiểm Chứng 5 Bước AO & Pair Hold)**:
   - Chứng minh 5 bước exit gate AO (session creation, dispatch, observation reconciliation, raw workspace file read, teardown) bằng cách **gọi trực tiếp các API điều phối nội bộ đã được duyệt của thư viện Go Supervisor** (`internal/dispatch`, `internal/store`, `internal/stop`, `internal/ao`, `internal/recovery`).
   - Chứng minh **Pair hold chặn admission** khi `PendingAO == true` thông qua các StateStore guards và admission check nội bộ trong integration harness.
   - Đi qua đầy đủ sagas, guards, transitions và audit trail mà không bypass state machine và không phụ thuộc vào code P04/P05.

### 4.2. Ma trận Kiểm chứng Runtime trên Binary Thật (`cmd/supervisor`)

| Kịch bản Kiểm chứng | Hành vi & Trạng thái Mong đợi | Phương pháp Kiểm tra Thực tế |
|---|---|---|
| **1. Trước Run** | `ProcessOwnerLease` được acquire (`.owner.lock` với `dwShareMode = 0`). Listener chưa bind. | Network probe: Socket connect bị từ chối (Connection Refused). |
| **2. Đang Run** | `Runner.Run(ctx)` nắm `ExclusiveScope`. Listener tiếp tục đóng. | Network probe: Socket connect bị từ chối; log chỉ bổ trợ. |
| **3. Sau Run (Complete)** | `r.ready = true`. Readiness port mở. | HTTP probe tới readiness port trả về `200 OK`. |
| **4. Run Lỗi / Incomplete** | `r.ready = false`. Admission đóng fail-closed; daemon exit non-zero. | Socket không bao giờ mở; process exit code != 0. |
| **5. Cạnh tranh 2 Process** | Process B mở `.owner.lock` nhận `ERROR_SHARING_VIOLATION` (32); cooperative takeover an toàn. | Process A drain và exit; Process B tiếp quản; nếu A không thoát thì B fail-closed. |
| **6. Shutdown Drain** | Nhận SIGINT: listener đóng -> poller drain -> timeout permits drain -> Store.Close() -> dọn .owner.json (nếu khớp instance) -> đóng lock handle cuối cùng. | Probe socket đóng ngay; verify mọi in-flight connection hoàn tất trước khi DB đóng. |

### 4.3. Ma trận Kiểm Chứng Hai Tiến Trình Cho Alias & Cross-Session Competition

| Kịch bản Thử nghiệm | Đường dẫn / Môi trường Thử nghiệm | Phương Pháp Probe Thực Tế & Kết quả Kỳ vọng |
|---|---|---|
| **Exact Flags Contender** | Process A & Process B cùng mở `.owner.lock` với `GENERIC_READ\|GENERIC_WRITE`, `dwShareMode=0`, `OPEN_ALWAYS` | Process B nhận ngay `ERROR_SHARING_VIOLATION` (32) từ Windows kernel |
| **Cross-Session Competition** | Session 0 (Service) vs Session 1 (Interactive) cùng mở `.owner.lock` | Process thứ hai nhận ngay `ERROR_SHARING_VIOLATION` (32) xuyên session |
| **Relative Path Probe** | `.\data\db.sqlite` vs `data\..\data\db.sqlite` | Probe xác nhận hai tiến trình tranh chấp cùng một canonical lock path |
| **Case Differences Probe** | `D:\data\db.sqlite` vs `d:\DATA\DB.SQLITE` | Probe xác nhận hai tiến trình quy về cùng volume + file ID và lock path |
| **Subst Drive Probe** | `X:\db.sqlite` (với `subst X: D:\data`) vs `D:\data\db.sqlite` | Probe kernel handle resolve về cùng volume + file ID; process 2 nhận violation |
| **Directory Junction Probe** | `D:\junction\db.sqlite` -> `D:\real\db.sqlite` | Probe `GetFinalPathNameByHandleW` resolve về cùng physical identity |
| **DB Mới (Chưa tồn tại)** | File chưa có trên đĩa | Pre-open parent dir volume check; post-open verify volume + file ID match lock key |
| **Post-Open Identity Check** | Handle file DB sau `Store.Open()` đối chiếu canonical lock key | Khớp -> Tiếp tục; Mismatch -> `Store.Close()` & FAIL-CLOSED |
| **Hard Link Detection** | File có `nNumberOfLinks > 1` | Bị từ chối ngay lập tức: **FAIL-CLOSED** (`ERR_HARDLINK_ALIAS_UNSUPPORTED`) |

### 4.4. Invariants Bất biến
- `AUTOMATIC_RESTORE = DISABLED`: Giữ nguyên disable fail-closed.
- 8 Operational policies tiếp tục giữ nguyên ở trạng thái **`UNSET`** trong tài liệu; runtime bắt buộc phải được inject giá trị cấu hình hợp lệ khi khởi động, thiếu bất kỳ giá trị nào thì fail-closed ngay lập tức.
- Tách biệt hoàn toàn test Mock AO tự động trong CI khỏi bài kiểm chứng Live AO có kiểm soát.
