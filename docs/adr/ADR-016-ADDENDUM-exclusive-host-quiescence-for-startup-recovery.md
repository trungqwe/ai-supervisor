# ADR-016 addendum: exclusive host quiescence trước startup recovery

**Trạng thái:** `EXTERNAL_APPROVED` theo quyết định External Supervisor tháo `BLOCKER-3D-003` ở cấp thiết kế. Bổ sung ownership contract cho ADR-016 D13 và TASK-P03-003D; không thay DDL, AO wire, TaskState graph hoặc các quyết định lịch sử.

## Lý do

Persisted intent không chứng minh caller tạo intent đã chết. Store CAS chỉ bảo vệ mutation bền vững; nó không thu hồi quyền gọi `/kill`, `/send`, `/restore` hay provisioning đang nằm trong stack frame của caller. Đặc biệt, caller có thể commit `STOP_REQUESTED`, tạm dừng trước effect, rồi tiếp tục sau khi scanner đã phân loại row. Timestamp, timeout, lease expiry, row reread và process-local mutex không chứng minh caller cũ không thể phát effect.

## Quyết định ownership

Trusted host cung cấp boundary admission/ownership dùng chung. Trước `Run`, host đóng admission cho mọi caller có thể tạo effect hoặc mutation xung đột với sweep: stop, send, restore, provisioning và các Store mutation liên quan. Host drain và **join** mọi caller đang giữ quyền; gửi cancellation request đơn thuần chưa phải join. Chỉ sau khi drain hoàn tất, scanner nhận exclusive ownership scoped cho cùng DB/Pair/effect admission. Ownership giữ đến khi `Run` hoàn tất hoặc trả incomplete/error; release đúng một lần. Không giữ SQLite transaction trong lúc drain hoặc gọi AO.

API thư viện dùng scoped acquire/release hoặc callback tương đương, không nhận bool `quiescent=true` từ caller thông thường. Thiếu provider, acquire/drain lỗi hoặc bị hủy thì `Run` fail closed trước khi phân loại intent. `Run` lặp lại khi host đang hoạt động phải acquire/drain lại. Poller không gọi startup-only classification hoặc suy persisted intent là abandoned; chỉ observation/CAS theo guard đã duyệt. Hai `Run` cạnh tranh phải được provider serialize. Với nhiều process, provider phải chứng minh exclusive ownership trên cùng DB/Pair/effect admission và caller cũ đã kết thúc; mutex trong một process không đủ. Scope 3D không xây distributed lease/fencing.

Sau classification-complete, host có thể mở admission theo Pair guards. Với incomplete/error/cancellation, release ownership không tự mở admission hoặc serve. Classification-complete/pending-AO vẫn giữ Pair có unresolved risk bị khóa. Audit `STARTUP_RECOVERY_SWEEP_COMPLETED` chứng minh hoàn tất phân loại, không chứng minh risk được giải quyết.

## Dependency và bằng chứng

`HOST_QUIESCENCE_INTEGRATION=OPEN`: host/bootstrap integration phải chứng minh authenticated trusted provider, admission close, drain/join, cross-process exclusion nếu dùng chung DB, exclusive lifetime và fail-closed reopening tại entrypoint thực tế. Fake provider và deterministic barrier test chỉ chứng minh call contract thư viện, không chứng minh daemon ordering hoặc host quiescence thật. `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`; verified host principal dependency và `AUTOMATIC_RESTORE=DISABLED` giữ nguyên.

Acceptance tests chặn caller A sau intent trước effect; B `Run` không GET/classify trước khi A kết thúc; caller C bị từ chối khi admission đóng; sau join B mới xử lý durable state. Kiểm tra provider thiếu/lỗi, drain/Run cancellation, audit failure, hai Run, Run/poller và release đúng một lần bằng barriers, không dùng sleep làm oracle.
