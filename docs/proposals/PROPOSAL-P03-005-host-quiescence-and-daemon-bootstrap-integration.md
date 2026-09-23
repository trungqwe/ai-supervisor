# PROPOSAL-P03-005 — Host Quiescence & Daemon Bootstrap Integration (TASK-P03-004)

> **Status:** `PROPOSED_FOR_SUPERVISOR_REVIEW`
> **Authority:** `docs/24_CHANGE_GOVERNANCE.md` (Level 3 Canonical Architecture / Level 5 Roadmap Alignment)
> **Active Gate:** `TASK_P03_003D_HANDOFF_VERIFICATION`
> **Proposed Phase Allocation:** Phase P03 subtask `TASK-P03-004`

---

## 1. Căn cứ và Vấn đề Cần Giải Quyết

1. **Hiện trạng Hoàn thành Thư viện P03 (3A–3D)**:
   - Các subtask TASK-P03-003A, 3B, 3C, 3D đã hoàn thành toàn bộ phạm vi triển khai trong thư viện Go nội bộ (`internal/ao`, `internal/dispatch`, `internal/stop`, `internal/recovery`, `internal/store`).
   - Implementation commit `7513f1b9f39fa459be15e0abc836c6258a610d8b` đã được External Supervisor phê duyệt (`EXTERNAL_AUDIT_APPROVED`) và merge vào main tại commit `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`.

2. **Ranh giới Exit Gate của Phase P03**:
   - Theo `docs/17_ROADMAP.md`, Phase P03 có Exit Gate tổng thể: *"Automated session creation, dispatch, observation reconciliation, raw workspace file read, and teardown pass without P04 EvidenceCollector dependencies"*.
   - Hiện tại, repository hoàn toàn chưa có binary daemon entrypoint (`cmd/`), chưa có daemon lifecycle phục vụ môi trường runtime thật, và các dependency then chốt gồm:
     * `HOST_QUIESCENCE_INTEGRATION = OPEN`
     * `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
     * Verified operator principal unproven at runtime
     vẫn đang ở trạng thái OPEN. Do đó, **chưa thể tuyên bố toàn bộ Phase P03 hoàn tất**.

3. **Ranh giới với Phase P04**:
   - Theo `docs/17_ROADMAP.md` và `docs/22_MODULE_PROVENANCE.md`, Phase P04 sở hữu độc quyền Evidence & Review Engine (`EvidenceCollector`, `ReviewBundleBuilder`, policy validator, git diff verification).
   - P04 **hoàn toàn không sở hữu** daemon bootstrap, server entrypoint hay host quiescence. Việc gán trách nhiệm daemon bootstrap vào P04 là vi phạm phân định ranh giới kiến trúc.

4. **Nhu cầu Cấp thiết về Subtask `TASK-P03-004`**:
   - Để thỏa mãn exit gate của Phase P03 trước khi chuyển giao sang P04, đề xuất bổ sung subtask `TASK-P03-004 Host Quiescence & Daemon Bootstrap Integration` trong Phase P03 theo đúng quy trình `docs/24_CHANGE_GOVERNANCE.md`.

---

## 2. Mục tiêu & Nội dung Đề xuất của `TASK-P03-004`

1. **Xây dựng Daemon Entrypoint**:
   - Tạo entrypoint nhị phân thực thi `cmd/supervisor/main.go` trên nền tảng Windows.
   - Điều phối toàn bộ vòng đời khởi động, phục vụ và dừng an toàn của tiến trình Supervisor.

2. **Hiện thực Thống nhất Ba Interface Host tại Một Trusted Boundary**:
   - Cùng một host provider phải thực hiện đồng thời:
     * `recovery.HostQuiescence`: `Acquire(context.Context) (ExclusiveScope, error)` (phạm vi toàn bộ DB và shared admission).
     * `recovery.LegacyMaintenanceHost`: `AcquireMaintenance(ctx context.Context, pairID string, purpose string) (ExclusiveScope, error)`.
     * `stop.TimeoutAdmission`: `AcquireEffect(ctx context.Context, pairID string, purpose string) (TimeoutPermit, error)` (tham số thứ ba là purpose string, ví dụ `"TIMEOUT_MONITOR"`).
   - Đóng admission, drain/join, cấp exclusive ownership và release permit theo cùng một authority duy nhất.

3. **Cơ chế Exclusivity An toàn trên Windows**:
   - Không sử dụng các primitive POSIX như `flock` hay `SIGKILL`.
   - Không đặt khóa trực tiếp trên file database SQLite chính (`.db`) để tránh xung đột với internal locking của SQLite pager.
   - Sử dụng sidecar lock file (ví dụ `<db_path>.supervisor.lock`) với Windows `LockFileEx` (hoặc mở file với share mode bằng 0) kết hợp với Windows Named Mutex (`Global\AISupervisor_<DBHash>`).
   - Xử lý tiến trình cũ không hợp tác: gửi tín hiệu qua local IPC; nếu quá hạn, sử dụng `OpenProcess` + `TerminateProcess` và `WaitForSingleObject` để kernel Windows xác nhận tiến trình cũ đã thoát trước khi cấp `ExclusiveScope`. Nếu chưa chứng minh được exclusivity, tuyệt đối không cấp `ExclusiveScope`.

4. **Phân biệt Wire Effect Ranh giới vs. Outcome Chưa biết**:
   - Bỏ khẳng định cho rằng việc terminate process hoặc đóng socket chứng minh AO chưa nhận effect.
   - Phân biệt rõ:
     * *"Caller cũ không thể phát effect MỚI sau khi quiescence được xác nhận"*: được bảo đảm bởi việc thu hồi permit và xác nhận process cũ đã thoát.
     * *"Effect ĐÃ PHÁT có outcome chưa biết"*: gói tin đã gửi ra dây mạng trước thời điểm quiescence.
   - Mọi intent mơ hồ (`STOP_REQUESTED`, `SEND_REQUESTED`, `RESTORE_REQUESTED`) phải giữ nguyên trạng thái fail-closed, dựa vào quan sát và đối soát (reconcile) qua fresh GET / observation, tuyệt đối không replay.

5. **Quy trình Khởi động Startup-Before-Serve & Sơ đồ Ma trận**:
   - `Runner.Run(ctx)` tự động gọi `HostQuiescence.Acquire(ctx)` và `scope.Release()`. Entrypoint không gọi acquire lần hai.
   - Xử lý ma trận kết quả:
     * `report.Complete == true && err == nil`: Mọi intent đã được phân loại hoặc đóng. `Runner` tự đánh dấu `ready = true`. Host kiểm tra Pair guards, khởi động `Poller.Start(ctx)`, khởi động scheduler gọi `TimeoutMonitor.Tick(ctx)`, sau đó mở listener admission.
     * `report.Complete == true && report.PendingAO == true`: Sweep hoàn tất với Pair hold. `Runner` tự đánh dấu `ready = true`. Host kiểm tra Pair guards (giữ hold/quarantine cho các Pair liên quan), khởi động `Poller.Start(ctx)` để quan sát, khởi động `TimeoutMonitor.Tick(ctx)`, và mở listener admission chỉ cho các Pair sạch.
     * `Incomplete` (`report.Complete == false` hoặc `err != nil`): Gặp lỗi, cancel, hoặc attempt thiếu budget. `Runner` giữ `ready = false`. Startup admission tiếp tục ĐÓNG fail-closed, daemon dừng phục vụ.
   - Host quản lý listener admission và Pair guards từ bên ngoài, không can thiệp vào các trường private của `Runner`.

6. **Graceful Shutdown**:
   - Khi nhận OS interrupt/signal: Đóng listener admission -> Dừng `Poller` (`Poller.Stop()`) -> Dừng scheduler của `TimeoutMonitor` -> Drain và join toàn bộ in-flight effect callers đang giữ permit TRƯỚC KHI đóng Store và kết nối database.

---

## 3. Tiêu chuẩn Bằng chứng Runtime & Invariants

1. **Bằng chứng Runtime Thật**:
   - Yêu cầu kiểm chứng trên tiến trình binary thực tế (`cmd/supervisor`), chứng minh scanner chạy xong trước khi mở port tiếp nhận, và graceful drain hoàn tất trước khi tắt tiến trình.
   - Fake integration harness trong thư viện không được tính là bằng chứng runtime của daemon thật.
2. **Bảo toàn Invariants**:
   - `AUTOMATIC_RESTORE = DISABLED`: Tiếp tục tắt fail-closed cho đến khi có cơ chế authenticated operator principal tại trusted boundary.
   - 8 Operational policies tiếp tục giữ nguyên ở trạng thái **`UNSET`**, tuyệt đối không bịa số hoặc hardcode giá trị mặc định.
   - Không gọi AO thật trong môi trường test chưa được kiểm toán.

---

## 4. Kiến nghị Hành động

1. Kính trình External Supervisor phê duyệt Proposal `PROPOSAL-P03-005` và Draft `DRAFT-ADR-017`.
2. Sau khi được phê duyệt, chuẩn bị Task Contract `CONTRACT-TASK-P03-004` để triển khai Host Quiescence & Daemon Bootstrap Integration.
