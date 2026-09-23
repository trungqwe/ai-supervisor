# PROPOSAL-P03-004 — Send-confirmation-based execution budget

> **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`. Đây là phương án thiết kế cho `3D-R1-005`, không phải quyền triển khai. Quyết định hướng của External Supervisor: origin là `dispatch_operations.confirmed_at`; startup `Run` chỉ phân loại; runtime timeout đi qua host admission và coordinator 3C. `3D-R1-005` vẫn OPEN.

## 1. Căn cứ và vấn đề

- FR-006 yêu cầu bounded execution policy; ADR-016 D9 giữ `RUNNING` khi `waiting_input` nhưng monitor phải xử lý khi deadline hết; REC-004 yêu cầu live stop qua D11 và `failure_reason=TIMEOUT` trong audit. ADR-016 D11/D12 quy định stop result, closure, disposition, quarantine và audit nguyên tử.
- `RecordSendConfirmed` hiện chỉ ghi `dispatch_operations.confirmed_at`, stage và audit trong một transaction. `TaskAttempt.started_at` được ghi trong `PrepareBoundDispatch` trước `/send`. `stop.Coordinator.Start` chỉ cấp quyền effect cho caller thắng reservation; đọc lại `STOP_REQUESTED` không cấp quyền. `Runner.Run` nắm exclusive host quiescence; `Poller` chỉ quan sát.
- `SOURCE_REGISTRY`/`REUSE_MATRIX`: AO cung cấp session-scoped status và `/kill`; không có upstream execution-start timestamp hay idempotent kill permit. Không thêm wire field hoặc AOAdapter.

## 2. Phương án khuyến nghị và tradeoff

**Ngân sách tính từ xác nhận send HTTP 200**, dùng chính `dispatch_operations.confirmed_at` làm origin. Đây là mốc bền vững và xác định; có thể bắt đầu trước khi execution thực tế bắt đầu. Không gọi nó là actual execution start. Capture policy snapshot (`duration_ns`, `policy_ref`) trước `SEND_REQUESTED`; validate fail-closed trước AO effect; bind origin, deadline và policy trong **cùng transaction** `RecordSendConfirmed`. Không tính lại theo cấu hình sau restart.

| Phương án | Kết quả |
| --- | --- |
| `confirmed_at` — chọn | Atomic với send confirmation, restart-stable; tính cả độ trễ trước execution thực tế. |
| `task_attempts.started_at` — không chọn | Gồm thời gian prepare/binding, thậm chí trước `/send`; có thể timeout một execution chưa được gửi. |
| First `RUNNING` observation — không chọn | Có thể bắt đầu muộn tùy poll interval hoặc bỏ lỡ active window; không cố định tại crash boundary sau HTTP 200. |

## 3. Persistence đề xuất (schema v5)

```sql
CREATE TABLE attempt_execution_budgets (
    attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    dispatch_operation_id TEXT NOT NULL UNIQUE REFERENCES dispatch_operations(operation_id) ON DELETE RESTRICT,
    origin_at TEXT NOT NULL,
    deadline_at TEXT NOT NULL,
    duration_ns INTEGER NOT NULL CHECK (duration_ns > 0),
    policy_ref TEXT NOT NULL CHECK (length(trim(policy_ref)) > 0),
    binding_basis TEXT NOT NULL CHECK (binding_basis IN ('SEND_CONFIRMATION_ATOMIC','LEGACY_OPERATOR_VERIFIED')),
    authorized_principal TEXT,
    evidence_ref TEXT,
    bound_at TEXT NOT NULL,
    CHECK ((binding_basis='SEND_CONFIRMATION_ATOMIC' AND authorized_principal IS NULL AND evidence_ref IS NULL)
        OR (binding_basis='LEGACY_OPERATOR_VERIFIED' AND authorized_principal IS NOT NULL AND evidence_ref IS NOT NULL
            AND length(trim(authorized_principal)) > 0 AND length(trim(evidence_ref)) > 0))
);
CREATE TRIGGER trg_execution_budget_immutable_update BEFORE UPDATE ON attempt_execution_budgets
BEGIN SELECT RAISE(ABORT,'execution budget immutable'); END;
CREATE TRIGGER trg_execution_budget_immutable_delete BEFORE DELETE ON attempt_execution_budgets
BEGIN SELECT RAISE(ABORT,'execution budget immutable'); END;
ALTER TABLE stop_operations ADD COLUMN initiating_failure_reason TEXT
    CHECK (initiating_failure_reason IS NULL OR initiating_failure_reason='TIMEOUT');
CREATE UNIQUE INDEX idx_one_live_stop_per_attempt ON stop_operations(attempt_id)
    WHERE purpose='RUNNING_ATTEMPT_STOP';
```

V5 còn cần các trigger insert budget, dispatch confirmation và provenance immutable được viết đầy đủ trong draft ADR. Chúng kiểm tra exact attempt/dispatch/stage và chặn Store API generic xác nhận send thiếu budget. Các trigger không so sánh thứ tự timestamp dạng text: Go parse RFC3339Nano UTC và dùng `time.Before`/`Add` với kiểm tra overflow, year range và format/parse round-trip. Invalid/zero/negative duration, origin sau deadline, timestamp hỏng hoặc policy ref rỗng đều fail-closed. V5 không gán budget cho row lịch sử; duplicate live-stop lịch sử làm migration rollback để xử lý riêng, không xóa dữ liệu.

`binding_basis` là hai literal mới cần ADR/registry approval. `TIMEOUT` đã được REC-004 nêu là `failure_reason`, không biến thành `recovery_disposition`. `policy_ref` là ID/version của policy đã được host cấp, không chứa secret; `duration_ns` là giá trị có hiệu lực tại send. Bảng một row/attempt và trigger bất biến ngăn thay deadline bằng policy mới hoặc retry.

## 4. Transaction và API

1. `dispatch.Coordinator.Dispatch` yêu cầu policy snapshot hợp lệ trước `RecordSendRequested`, giữ snapshot trong stack frame của lần gửi; không fetch lại giữa send và confirm. `Store.RecordSendConfirmed` nhận snapshot, bắt đầu transaction, kiểm tra exact Pair/task/contract/current open attempt/session/generation/stage, tính origin từ cùng `at`, insert budget, CAS `SEND_REQUESTED→SEND_CONFIRMED` với `confirmed_at`, ghi `DISPATCH_SEND_CONFIRMED` gồm policy/origin/deadline, rồi commit. Bất kỳ insert/CAS/audit/commit lỗi đều rollback; containment D5 đọc lại exact operation + budget + audit trước khi quyết định unknown delivery, tuyệt đối không resend.
2. `Store.GetExecutionBudget` và `ListDueExecutionBudgets` chỉ trả dữ liệu parse/lineage hợp lệ. `Store.ReserveStopOperation` cho cause `TIMEOUT` phải CAS exact Pair/task/contract/current open RUNNING attempt, immutable session/generation, `SEND_CONFIRMED`, budget cùng attempt/dispatch, `now >= deadline_at`, không có stop nào cho attempt và không có Pair risk/hold chưa xử lý. Insert stop intent + `STOP_OPERATION_REQUESTED` audit mang cause, budget/provenance trong một transaction; unique index chọn một monitor thắng. Không cấp effect permit từ read/replay.
3. Runtime `TimeoutMonitor.Tick` chỉ hoạt động sau startup classification-complete. Guarded `stop.Coordinator.StartTimeout` nhận trusted shared admission `AcquireEffect(ctx, pairID, "TIMEOUT_MONITOR") (ScopedPermit, error)`, lấy clock sau acquire, rồi dùng private Start/reservation flow 3C; `Start` thường từ chối cause TIMEOUT để không có bypass. Permit giữ đến khi effect/containment hoàn tất, release đúng một lần. Host quiescence đóng admission và drain/join cả timeout caller; process-local mutex không chứng minh cross-process. Thiếu provider, injected deadline policy hoặc kill confirmation policy thì timeout path disabled và báo dependency; các đường cần host principal vẫn fail-closed riêng. Monitor không gọi `/kill` hoặc insert stop trực tiếp. Startup `Run` không gọi monitor và không tạo timeout stop.
4. `stop.Coordinator.StartTimeout` nhận timeout cause qua `domain.StopOperation`, rồi gọi Store reservation **trong chính đường Start** sau host permit. `stop_operations.initiating_failure_reason='TIMEOUT'` tồn tại sau crash; D11 outcome dùng giá trị này để ghi `TASK_STATE_TRANSITION.details_json.failure_reason='TIMEOUT'`, còn `recovery_disposition` giữ `WORKER_STOPPED`, `STOP_CONFIRMATION_TIMEOUT`, `SESSION_ABSENT` hoặc outcome D11 tương ứng. Stop result, D12 closure, quarantine và audit vẫn chung một transaction 3C. Không ghi `WORKER_STOPPED` khi chỉ hết execution budget.
5. Legacy `SEND_CONFIRMED` thiếu budget: startup fail-closed trước serve/admission, không backfill bằng `now`, `started_at` hoặc policy hiện tại. `BindLegacyExecutionBudget` chỉ nhận trusted verified operator có scope `LEGACY_EXECUTION_BUDGET_RECONCILIATION`, bằng chứng `confirmed_at` gốc và policy có hiệu lực lúc send (ref + duration). Exact CAS kiểm tra current open lineage/stage, insert budget basis `LEGACY_OPERATOR_VERIFIED` và audit mới `EXECUTION_BUDGET_LEGACY_BOUND` cùng transaction; không đổi `confirmed_at`. Nếu thiếu bằng chứng, Pair tiếp tục không được dispatch/monitor; operator phải dùng đường manual stop/reconciliation đã duyệt dưới host admission, không tạo lịch sử giả. Closed historical attempt không được backfill để phục vụ runtime monitor.

## 5. Eligibility và precedence

| Thứ tự | Durable state / observation | Hành động đề xuất |
| --- | --- | --- |
| 1 | Startup `Run`, hoặc thiếu trusted host admission | Chỉ phân loại persisted intent; zero timeout effect. Incomplete/disabled nếu budget bắt buộc thiếu. |
| 2 | Existing stop của attempt, kể cả resolved nhưng attempt còn mở | Không tạo stop mới, không replay `/kill`; stop lifecycle 3C/3D hoặc human reconciliation nắm quyền. |
| 3 | `BLOCKED`/`HUMAN_REQUIRED`, AO `blocked`, hoặc Pair có unresolved restore/provisioning/delivery/hold/quarantine | Ưu tiên escalation/reconciliation tương ứng; không dùng timeout để vượt guard. |
| 4 | `DISPATCHED` chưa được quan sát execution hoặc handoff P04 đang xử lý | Không ép `RUNNING`; D8/P04 observation/handoff đi trước timeout. |
| 5 | Fresh 404, terminated hoặc identity/generation mismatch trước reservation | Ghi outcome hiện có bằng exact CAS/D12; không tạo timeout intent hoặc suy physical stop. Nếu chỉ phát hiện sau reservation, D11 ghi logical outcome và không gọi `/kill`; TIMEOUT vẫn là trigger đã persist, không là nguyên nhân chết. CAS loser đọc lại. |
| 6 | Exact current open `RUNNING/SEND_CONFIRMED`, policy/budget hợp lệ, no stop/risk, `now < deadline_at` | Chỉ quan sát; `waiting_input` tiếp tục RUNNING theo D9. |
| 7 | Cùng lineage, `now >= deadline_at` | Reserve `RUNNING_ATTEMPT_STOP` mới qua 3C trong host permit; chưa đóng attempt. D11 outcome sau đó quyết định closure. |

Execution deadline là ngân sách trước khi yêu cầu stop; `confirmation_deadline_at = call_completed_at + SUPERVISOR_KILL_STOP_TIMEOUT` là cửa sổ xác nhận D11, không được gộp hoặc reset từ execution deadline. Exact observation/Pair guards được recheck trong Store transaction; preflight GET ngoài transaction chỉ là bằng chứng mới tại thời điểm quan sát, không thay CAS.

## 6. Crash matrix

| Crash/bất định | Durable state và xử lý |
| --- | --- |
| Trước `/send` hoặc HTTP 200 chưa đến | Không có budget; giữ semantics send intent/D5 hiện hành. |
| HTTP 200 nhưng confirm tx rollback | `SEND_REQUESTED` còn nguyên; D5 containment sau exact reread, không resend. |
| Confirm commit mơ hồ | Reread stage + immutable budget + audit; tuple đầy đủ thì giữ winner, thiếu thì D5 fail-closed, không xác nhận bằng suy đoán. |
| Confirm commit xong, trước monitor; restart đổi policy | Origin/deadline/duration/ref bất biến; dùng budget row, không tính lại. |
| Host permit chưa cấp hoặc quiescence bắt đầu | Không reserve; host đóng admission và drain caller. |
| Timeout intent commit, trước `/kill` | Stop intent giữ cause TIMEOUT; startup phân loại theo host quiescence, không cấp effect permit mới. |
| `/kill` mất response hoặc confirmation commit mơ hồ | 3C containment/unknown outcome; không replay effect; cause giữ trong stop row. |
| D11 outcome tx audit lỗi | Stop, TaskState, attempt, quarantine và audit rollback nguyên tử; không giả kết quả physical. |

## 7. Scope, kiểm chứng và điểm cần duyệt

Draft contract revision 3 giữ code base `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`; tiếp tục branch implementation chưa merge tại `00f64d0b26db2749eadd21b7307c98faa6b6280d`, giao ADR/contract artifact riêng, không reset các sửa R1-001..004. Whitelist cụ thể trong draft contract gồm file dispatch, stop, recovery, domain và Store hiện hữu cùng migration/test v5; không sửa AOAdapter, P04 runner, daemon hay accepted ADR.

Test bắt buộc: before/equal/after deadline, `waiting_input` kéo dài, restart với policy mới, invalid/overflow timestamp, legacy có/thiếu evidence, confirm rollback, hai monitor, host quiescence barrier, crash sau intent, stop hiện hữu, D11 cause/disposition tách biệt và audit rollback. Test fake provider chỉ chứng minh library contract.

**Supervisor cần duyệt:** DDL v5 và trigger/legacy policy; các literal `binding_basis`, event `EXECUTION_BUDGET_LEGACY_BOUND`, cause column; thứ tự blocked/P04/terminal observation; phương thức trusted legacy authority; phạm vi mở `internal/dispatch`, `internal/stop`, `internal/domain` và host admission interface. Chỉ sau approval mới reconcile canonical registry/specs và release contract revision 3. `3D-R1-005` còn OPEN.
