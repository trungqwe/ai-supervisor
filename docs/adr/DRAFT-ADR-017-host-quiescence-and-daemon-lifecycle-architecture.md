# DRAFT ADR-017 — Host Quiescence & Daemon Lifecycle Architecture

> **Status: PROPOSED / REVISION_7_PENDING_EXTERNAL_AUDIT (Audit: docs/audits/P03_ADR_017_EXTERNAL_REAUDIT_006.md)
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

### 2.3. Cơ chế Machine-Wide Exclusivity trên Windows & Hai Đường Đi Khởi Tạo DB

1. **Hợp đồng `CreateFileW` Cho `.owner.lock`**:
   Khóa độc quyền chính là một Windows Exclusive Sidecar Lock File Handle mở trên file `<canonical_db_path>.owner.lock` với bộ tham số API kernel bất biến:
   - `dwDesiredAccess = GENERIC_READ | GENERIC_WRITE`: Yêu cầu quyền truy cập non-zero xung đột trực tiếp giữa các tiến trình cạnh tranh.
   - `dwShareMode = 0`: Chế độ độc quyền hoàn toàn (không chia sẻ đọc, không chia sẻ ghi, không chia sẻ xóa).
   - `dwCreationDisposition = OPEN_ALWAYS`: Tạo file nếu chưa tồn tại; mở file hiện có mà **TUYỆT ĐỐI KHÔNG TRUNCATE** file lock đang bị tiến trình khác nắm giữ.
   - `dwFlagsAndAttributes = FILE_ATTRIBUTE_NORMAL`: Kết hợp thuộc tính chuẩn của hệ thống tệp.
   - `lpSecurityAttributes.bInheritHandle = FALSE`: Ngăn chặn handle leak sang child process.
   - **Xử lý lỗi**: Mọi lỗi acquire (như `ERROR_SHARING_VIOLATION` (32), `ERROR_ACCESS_DENIED` (5)) đều được xử lý **FAIL-CLOSED** ngay lập tức.

2. **Phân Biệt Rành Mạch: Canonical Lock Key vs. Physical File Identity**:
   - **Canonical Lock Key (Path-based Token)**: Chuỗi đường dẫn canonical NT DOS `<canonical_db_path>.owner.lock` được xác lập nhằm thiết lập cơ chế loại trừ tương hỗ (mutual exclusion) giữa các tiến trình trên filesystem.
   - **Physical File Identity (Volume + File ID)**: Cặp định danh vật lý `(VolumeSerialNumber, FileId)` đại diện cho chính xác thực thể inode/file trên volume đĩa vật lý của Windows.
     * Trên **NTFS**: Sử dụng `FILE_ID_INFO` (128-bit `FileId` + `VolumeSerialNumber`) qua `GetFileInformationByHandleEx`.
     * Trên **ReFS**: Bắt buộc sử dụng cấu trúc 128-bit `FILE_ID_INFO` (`GetFileInformationByHandleEx(h, FileIdInfo, &info, sizeof(info))`).

3. **Hai Đường Đi Khởi Tạo DB & Cơ Chế Host Pinned Handle**:
   - **Đường đi 1: Database file ĐÃ TỒN TẠI (Existing DB)**:
     1. *Pre-Open Pinned Handle & Identity*:
        Host mở handle pin `hPinnedDB` trực tiếp qua Win32 `CreateFileW(canonicalDBPath, GENERIC_READ, FILE_SHARE_READ | FILE_SHARE_WRITE, NULL, OPEN_EXISTING, 0, NULL)` (tuyệt đối không cấp `FILE_SHARE_DELETE`).
        Lấy canonical path qua `GetFinalPathNameByHandleW(hPinnedDB, VOLUME_NAME_DOS)` làm cơ sở tạo lock key `<canonical_db_path>.owner.lock`.
        Kiểm tra hard links: `GetFileInformationByHandle`. Nếu `nNumberOfLinks > 1` (DB có hard link alias), daemon **FAIL-CLOSED NGAY LẬP TỨC** (`ERR_HARDLINK_ALIAS_UNSUPPORTED`).
        Trích xuất physical identity: `(preVolume, preFileId128)` qua `GetFileInformationByHandleEx(hPinnedDB, FileIdInfo)`.
        **GIỮ HANDLE `hPinnedDB` LIÊN TỤC KHÔNG ĐÓNG**, qua acquire lock, `Store.Open()` và toàn bộ runtime của daemon.
     2. *Acquire Lock*: Host mở `<canonical_db_path>.owner.lock` bằng hợp đồng `CreateFileW` độc quyền (share mode 0).
     3. *Store Initialization*: Host truyền `canonicalDBPath` vào `store.NewStore()`. `Store.Open()` kết nối SQLite tới file database.
     4. *Post-Open Verification*: Host kiểm tra `PRAGMA database_list` xác nhận SQLite kết nối tới đúng `canonicalDBPath`. Host gọi lại `GetFileInformationByHandleEx(hPinnedDB, FileIdInfo)` tái xác nhận physical identity `(postVolume, postFileId128) == (preVolume, preFileId128)`.
     5. *Runtime Custody & Shutdown*: `hPinnedDB` được giữ xuyên suốt runtime, bảo đảm bằng cơ chế kernel Windows rằng không tiến trình nào có thể xóa/đổi tên file dưới chân SQLite. Khi shutdown: `Store.Close()` được gọi trước; sau khi Store đóng hoàn toàn, host mới đóng `hPinnedDB`, và cuối cùng đóng lock handle `.owner.lock` sau cùng.
   - **Đường đi 2: Database file CHƯA TỒN TẠI (Brand New DB)**:
     1. *Pre-Lock & Parent Canonicalization*: Thư mục cha bắt buộc phải tồn tại. Host mở handle tới thư mục cha với cờ `FILE_FLAG_BACKUP_SEMANTICS`. Lấy canonical path của thư mục cha qua `GetFinalPathNameByHandleW` và lấy `parentVolume` qua `GetFileInformationByHandleEx(FileIdInfo)`. Đóng handle thư mục cha.
     2. *Acquire Lock*: Host acquire lock file `<canonical_db_path>.owner.lock` trước (share mode 0) dựa trên `filepath.Join(canonicalParentDir, dbBaseName)`.
     3. *Exclusive-Create & Pin TRƯỚC `Store.Open()`*:
        Host tạo file bằng cơ chế exclusive-create: `CreateFileW(canonicalDBPath, GENERIC_READ | GENERIC_WRITE, FILE_SHARE_READ | FILE_SHARE_WRITE, NULL, CREATE_NEW, FILE_ATTRIBUTE_NORMAL, NULL)` (không cấp `FILE_SHARE_DELETE`).
        Nếu file đã tồn tại hoặc không tạo được: `CREATE_NEW` trả về lỗi `ERROR_FILE_EXISTS` (80) -> host đóng lock handle và **FAIL-CLOSED NGAY LẬP TỨC**.
        Nếu tạo thành công: Host sở hữu `hPinnedDB` trên file 0-byte vừa tạo. Xác thực canonical path qua `GetFinalPathNameByHandleW(hPinnedDB)` khớp chính xác với `canonicalDBPath`. Lấy `FILE_ID_INFO`, kiểm tra volume trùng `parentVolume` và `nNumberOfLinks == 1`.
        **GIỮ HANDLE `hPinnedDB` LIÊN TỤC KHÔNG ĐÓNG**.
     4. *Store.Open() & Migrations*: Host truyền `canonicalDBPath` vào `store.NewStore()`. `Store.Open()` kết nối SQLite vào file 0-byte đã được pin, tự động khởi tạo database, schema migrations, WAL mode và bảng dữ liệu. (Đã probe chứng minh thực tế SQLite hoạt động bình thường, hỗ trợ đầy đủ migration, transaction commit và rollback trên file 0-byte được pin).
     5. *Runtime Custody & Shutdown*: `hPinnedDB` được giữ liên tục trong suốt runtime. Shutdown: `Store.Close()` trước -> đóng `hPinnedDB` -> đóng `.owner.lock` cuối cùng.
   - **Ranh giới Store**: Toàn bộ cơ chế pin và đối chiếu nằm trọn trong `internal/host`; không sửa đổi bất kỳ dòng code nào trong `internal/store`.

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
