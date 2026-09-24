# PROPOSAL-P03-005 — Host Quiescence & Daemon Bootstrap Integration (TASK-P03-004)

> **Status:** `PROPOSED_FOR_SUPERVISOR_REAUDIT` (Revision 3)
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

3. **Ranh giới với Phase P04 & Loại bỏ Verification Runner khỏi P03**:
   - Theo `docs/17_ROADMAP.md` và `docs/22_MODULE_PROVENANCE.md`, Phase P04 sở hữu độc quyền Evidence & Review Engine (`EvidenceCollector`, `ReviewBundleBuilder`, policy validator, git diff verification).
   - Production verification runner và các công cụ thực thi verification profile thuộc sở hữu độc quyền của Phase P04 và Phase P05 (`docs/21_TRACEABILITY_MATRIX.md`).
   - Subtask `TASK-P03-004` **loại bỏ production verification runner khỏi scope**, chỉ tập trung vào host/bootstrap, admission và test harness kiểm chứng exit gate P03.

4. **Phân bổ Subtask `TASK-P03-004` Thuộc Phase P03**:
   - Để thỏa mãn exit gate của Phase P03 trước khi chuyển giao sang P04, đề xuất bổ sung subtask `TASK-P03-004 Host Quiescence & Daemon Bootstrap Integration` trong Phase P03 theo đúng quy trình `docs/24_CHANGE_GOVERNANCE.md`.

---

## 2. Mục tiêu & Nội dung Đề xuất của `TASK-P03-004`

1. **Xây dựng Daemon Entrypoint**:
   - Tạo entrypoint nhị phân thực thi `cmd/supervisor/main.go` trên nền tảng Windows.
   - Điều phối toàn bộ vòng đời khởi động, phục vụ và dừng an toàn của tiến trình Supervisor.

2. **Tách Biệt Hai Cấp Độ Quiescence**:
   - Tách rõ ràng `ProcessOwnerLease` dài hạn cấp process theo DB (giữ từ TRƯỚC `Store.Open()`/migration đến SAU shutdown drain và `Store.Close()`) khỏi `ExclusiveScope` ngắn hạn do `Runner.Run(ctx)` tự acquire và release khi quét snapshot. Entrypoint không acquire lần hai.

3. **Hiện thực Thống nhất Ba Interface Host tại Một Trusted Boundary**:
   - Cùng một host provider phải thực hiện đồng thời:
     * `recovery.HostQuiescence`: `Acquire(context.Context) (ExclusiveScope, error)` (phạm vi toàn bộ DB và shared admission).
     * `recovery.LegacyMaintenanceHost`: `AcquireMaintenance(ctx context.Context, pairID string, purpose string) (ExclusiveScope, error)`.
     * `stop.TimeoutAdmission`: `AcquireEffect(ctx context.Context, pairID string, purpose string) (TimeoutPermit, error)` (tham số thứ ba là `purpose string`, ví dụ `"TIMEOUT_MONITOR"`).
   - Đóng admission, drain/join, cấp exclusive ownership và release permit theo cùng một authority duy nhất.

4. **Thuật toán Canonical DB & Lock Identity Duy Nhất**:
   - Dùng Windows API `GetFinalPathNameByHandleW(VOLUME_NAME_DOS)` trên DB handle (nếu DB đã tồn tại) hoặc trên parent directory handle (nếu DB mới).
   - Kiểm tra hard links (`nNumberOfLinks > 1`) và fail-closed nếu có alias hard link.
   - Khóa độc quyền chính là Windows Exclusive Sidecar Lock File Handle (`CreateFileW` với `dwShareMode = 0`) tại `<canonical_db_path>.owner.lock`, có hiệu lực toàn máy và kernel tự đóng khi crash.
   - File metadata tách biệt `<canonical_db_path>.owner.json` lưu thông tin liên lạc IPC (Named Pipe) được ghi atomically (qua file tạm `.owner.json.tmp` và rename). Nếu metadata hỏng/stale -> fail-closed, không suy đoán PID.
   - Named Pipe takeover có xác thực client token/SID và kiểm tra `owner_instance_id`. Yêu cầu không hợp lệ không được khiến owner shutdown.
   - V1 chỉ áp dụng **Cooperative Takeover** qua Named Pipe. Nếu owner cũ không thoát -> fail-closed ngay lập tức. Bỏ hành vi tự động `TerminateProcess`.

5. **Phân biệt Wire Effect Ranh giới vs. Outcome Chưa biết**:
   - Việc process cũ thoát chỉ chứng minh caller cũ không thể phát effect mới; không chứng minh AO chưa nhận effect đã phát trước đó.
   - Mọi intent mơ hồ (`STOP_REQUESTED`, `SEND_REQUESTED`, `RESTORE_REQUESTED`) phải giữ fail-closed, dùng observation/reconciliation, tuyệt đối không replay.

6. **Quy trình Khởi động Startup-Before-Serve & Sơ đồ Ma trận**:
   - `Runner.Run(ctx)` tự động gọi `HostQuiescence.Acquire(ctx)` và `scope.Release()`.
   - Phân loại ma trận:
     * `report.Complete == true && err == nil` (gồm cả `PendingAO == true`): Sweep hoàn tất, `Runner` tự đánh dấu `r.ready = true`. Host kiểm tra Pair guards, start `Poller`, start `TimeoutMonitor` scheduler, mở listener admission cho các Pair sạch.
     * `Incomplete` (`report.Complete == false` hoặc `err != nil`): Gặp lỗi hoặc thiếu budget. `Runner` giữ `r.ready = false`. Listener admission tiếp tục ĐÓNG fail-closed, daemon dừng.
   - `TimeoutMonitor`: Host scheduling định kỳ gọi `m.Tick(ctx)`.

7. **Graceful Shutdown**:
   - Đóng listener admission -> Dừng `Poller` (`Poller.Stop()`) -> Dừng scheduler `TimeoutMonitor` -> Drain/join toàn bộ effect callers TRƯỚC KHI đóng Store và giải phóng lock handle `.owner.lock`.

---

## 3. Tiêu chuẩn Bằng chứng Runtime & Ranh giới Tooling

1. **Tách Bạch Hai Track Bằng Chứng**:
   - Xóa bỏ giả định cho rằng daemon P03 đã có sẵn các effectful HTTP routes. Không dùng route AO trực tiếp vì bypass Supervisor state machine.
   - **Track 1 (Binary Thật `cmd/supervisor`)**: Kiểm chứng lock độc quyền, cạnh tranh tiến trình, startup-before-serve (admission probe), PendingAO hold, và graceful drain.
   - **Track 2 (P03 Integration Harness)**: Gọi trực tiếp các API điều phối nội bộ đã được duyệt của thư viện Go Supervisor (`internal/dispatch`, `internal/store`, `internal/stop`, `internal/ao`, `internal/recovery`) để kiểm chứng trọn vẹn 5 bước AO (session create, dispatch, observation reconciliation, raw file read, teardown) mà không cần production HTTP control surface hay code P04/P05.

2. **Ranh giới Tooling & An ninh SEC-003**:
   - Giới hạn P03 host ở bootstrap/admission và test harness tối thiểu; loại bỏ production verification runner khỏi P03 (chuyển về P04/P05).
   - Ghi nhận ChatGPT Web là reasoning agent, Control Plane cung cấp local tools. Ở P04/P05 đối soát tool log/runner với `docs/11_CHATGPT_TOOL_SURFACE.md`, `docs/02_REQUIREMENTS.md`, và `docs/07_SECURITY_MODEL.md` (SEC-003). Tuyệt đối không cấp arbitrary shell execution.

3. **Invariants Bất biến**:
   - `AUTOMATIC_RESTORE = DISABLED`: Tiếp tục tắt fail-closed.
   - 8 Operational policies tiếp tục giữ nguyên `UNSET` trong tài liệu; runtime bắt buộc phải được inject giá trị cấu hình hợp lệ, thiếu thì fail-closed khi khởi động.
