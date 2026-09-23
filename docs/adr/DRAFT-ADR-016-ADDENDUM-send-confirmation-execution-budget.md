# DRAFT ADR-016 Addendum — Send-confirmation-based execution budget

> **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`; chỉ hướng origin `dispatch_operations.confirmed_at`, budget bất biến, startup zero timeout effect và runtime reuse coordinator 3C đã được External Supervisor xác nhận. DDL, literal, transition và scope dưới đây **chưa được phê duyệt**. `3D-R1-005=OPEN`.
>
> **Liên quan:** ADR-016 D8/D9/D11/D12/D13, FR-006, REC-004, ADR-016 addendum exclusive host quiescence, [PROPOSAL-P03-004](../proposals/PROPOSAL-P03-004-send-confirmation-execution-budget.md). Không sửa accepted ADR-016 hay các clarification lịch sử.

## 1. Quyết định đề xuất và phạm vi supersede

Ngân sách execution bắt đầu tại **send confirmation** HTTP 200 (`dispatch_operations.confirmed_at`), không phải actual execution start. Có thể tính cả độ trễ trước execution thực tế. D9 vẫn giữ `waiting_input` là `RUNNING` cho đến khi ngân sách hết; addendum này chỉ bổ sung persistence, deadline monitor, cause và thứ tự phục hồi. D11/D12 physical proof, TaskState graph, AO `/kill` wire chỉ có `sessionId`, hold/quarantine và D13 startup effect-replay prohibition không thay đổi. Execution deadline và `confirmation_deadline_at` của stop là hai mốc độc lập.

## 2. Registry bổ sung được đề xuất

| Loại | Literal | Ý nghĩa, giới hạn |
| --- | --- | --- |
| `attempt_execution_budgets.binding_basis` | `SEND_CONFIRMATION_ATOMIC` | Policy snapshot và deadline được ghi cùng transaction xác nhận send. |
| Cùng enum | `LEGACY_OPERATOR_VERIFIED` | Operator đã xác thực cung cấp bằng chứng policy lịch sử; không dựng lịch sử từ policy hiện tại. |
| `stop_operations.initiating_failure_reason` | `TIMEOUT` | Cause của yêu cầu stop do hết execution budget; lưu bền vững trước `/kill`. Không phải D11 resolution/disposition. |
| `audit_events.event_type` mới | `EXECUTION_BUDGET_LEGACY_BOUND` | Ghi authority, evidence ref, exact lineage, policy/origin/deadline của legacy binding; không là physical proof hoặc quarantine clearance. |

Các event đã duyệt được dùng đúng vai trò: `DISPATCH_SEND_CONFIRMED` ghi budget atomic; `STOP_OPERATION_REQUESTED` ghi cause TIMEOUT khi reserve; `TASK_STATE_TRANSITION` ghi `failure_reason=TIMEOUT` khi D11 đóng live attempt; `STOP_OPERATION_CONFIRMED`, `STOP_CONFIRMATION_TIMEOUT`, `STOP_OPERATION_TARGET_ABSENT`, `STOP_OPERATION_RESOLVED` ghi outcome tương ứng. `QUARANTINE_RESOLVED_PHYSICAL` chỉ xuất hiện khi D11/D6 thật sự clear. Không phát event physical/clearance chỉ vì execution budget hết. `recovery_disposition` tiếp tục theo D11 outcome, không gán `TIMEOUT` vào enum disposition.

## 3. DDL v5 đề xuất

```sql
CREATE TABLE attempt_execution_budgets (
    attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    dispatch_operation_id TEXT NOT NULL UNIQUE REFERENCES dispatch_operations(operation_id) ON DELETE RESTRICT,
    origin_at TEXT NOT NULL,
    deadline_at TEXT NOT NULL,
    duration_ns INTEGER NOT NULL CHECK(duration_ns > 0),
    policy_ref TEXT NOT NULL CHECK(length(trim(policy_ref)) > 0),
    binding_basis TEXT NOT NULL CHECK(binding_basis IN ('SEND_CONFIRMATION_ATOMIC','LEGACY_OPERATOR_VERIFIED')),
    authorized_principal TEXT,
    evidence_ref TEXT,
    bound_at TEXT NOT NULL,
    CHECK ((binding_basis='SEND_CONFIRMATION_ATOMIC' AND authorized_principal IS NULL AND evidence_ref IS NULL)
        OR (binding_basis='LEGACY_OPERATOR_VERIFIED'
            AND authorized_principal IS NOT NULL AND evidence_ref IS NOT NULL
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
CREATE TRIGGER trg_stop_cause_immutable BEFORE UPDATE OF initiating_failure_reason ON stop_operations
WHEN NEW.initiating_failure_reason IS NOT OLD.initiating_failure_reason
BEGIN SELECT RAISE(ABORT,'stop cause immutable'); END;
CREATE TRIGGER trg_execution_budget_lineage BEFORE INSERT ON attempt_execution_budgets
WHEN NOT EXISTS (
    SELECT 1 FROM dispatch_operations d
    JOIN task_attempts a ON a.attempt_id=d.attempt_id
    JOIN tasks t ON t.task_id=a.task_id AND t.current_attempt=a.attempt_number
    WHERE d.operation_id=NEW.dispatch_operation_id AND d.attempt_id=NEW.attempt_id
      AND a.ended_at IS NULL
      AND ((NEW.binding_basis='SEND_CONFIRMATION_ATOMIC'
            AND t.state='DISPATCHED' AND d.stage='SEND_REQUESTED' AND d.confirmed_at IS NULL)
        OR (NEW.binding_basis='LEGACY_OPERATOR_VERIFIED'
            AND t.state IN ('DISPATCHED','RUNNING')
            AND d.stage='SEND_CONFIRMED' AND d.confirmed_at=NEW.origin_at))
)
BEGIN SELECT RAISE(ABORT,'execution budget lineage invalid'); END;
CREATE TRIGGER trg_dispatch_requires_execution_budget
BEFORE UPDATE OF stage,confirmed_at ON dispatch_operations
WHEN NEW.stage='SEND_CONFIRMED' AND
    (NEW.confirmed_at IS NULL OR NOT EXISTS (
        SELECT 1 FROM attempt_execution_budgets b
        WHERE b.dispatch_operation_id=NEW.operation_id
          AND b.attempt_id=NEW.attempt_id AND b.origin_at=NEW.confirmed_at))
BEGIN SELECT RAISE(ABORT,'send confirmation requires exact budget'); END;
CREATE TRIGGER trg_dispatch_confirmed_immutable
BEFORE UPDATE OF stage,confirmed_at ON dispatch_operations
WHEN OLD.stage='SEND_CONFIRMED' AND
    (NEW.stage<>'SEND_CONFIRMED' OR NEW.confirmed_at IS NOT OLD.confirmed_at)
BEGIN SELECT RAISE(ABORT,'confirmed send provenance immutable'); END;
```

Các trigger guard cả Store API generic hiện có. SQL text ordering **không** quyết định timestamp: Go parse UTC RFC3339Nano và kiểm tra `origin < deadline`, `deadline = origin + positive duration`, overflow/year range và round-trip; read path fail-closed nếu dữ liệu hỏng. Migration v5 trên fresh/v1/v2/v3/v4 là transactional, giữ dữ liệu/audit v1–v4; duplicate historical live-stop theo unique index làm migration rollback rõ ràng, không tự xóa.

## 4. Transaction và authority

| Boundary | Điều kiện và commit |
| --- | --- |
| Trước AO `/send` | Host inject snapshot `SUPERVISOR_EXECUTION_DEADLINE` gồm `duration_ns` dương và `policy_ref` bền vững; validate trước `SEND_REQUESTED`. Không default. Snapshot giữ trong call frame; thay policy giữa call không đổi snapshot. |
| Send confirmation Tx S | `RecordSendConfirmed` kiểm tra exact Pair/task/contract/current open attempt/session/generation và stage. Lấy **cùng** `at` làm `confirmed_at`/origin; insert budget, CAS stage+timestamp, append `DISPATCH_SEND_CONFIRMED` có provenance, commit nguyên tử. Insert/CAS/audit lỗi rollback toàn bộ. Không có trạng thái SEND_CONFIRMED mới mà thiếu budget. |
| Legacy Tx L | `BindLegacyExecutionBudget` kiểm tra principal từ trusted host boundary với scope `LEGACY_EXECUTION_BUDGET_RECONCILIATION`, evidence ref và policy lịch sử. Exact CAS current open attempt, Pair/session/generation, stage/confirmed_at và budget absent; insert basis legacy + `EXECUTION_BUDGET_LEGACY_BOUND` cùng transaction. Replay cùng evidence read-only; khác evidence/policy từ chối. |
| Timeout reservation Tx R | Sau fresh observation và host shared admission permit, guarded `stop.Coordinator.StartTimeout` dùng chính private Start/reservation flow 3C và gọi `ReserveStopOperation` đúng một lần. `Start` thường từ chối cause TIMEOUT khi thiếu trusted admission wrapper. Tx R kiểm tra exact current open `RUNNING/SEND_CONFIRMED`, clean guards, immutable budget, `now >= deadline`, không có live stop của attempt; insert stop `RUNNING_ATTEMPT_STOP` với `initiating_failure_reason='TIMEOUT'` + `STOP_OPERATION_REQUESTED` có budget ref trong một transaction. Unique index/CAS chọn một winner. |
| AO effect và D11 Tx O | Chỉ stack frame winner Tx R gọi GET và `/kill` ngoài SQLite transaction. `CommitStopCallAccepted` lưu stop confirmation deadline riêng. `CommitStopOutcome` giữ một transaction D11/D12: stop stage/resolution, `RUNNING→FAILED`, attempt closure, disposition, quarantine, `TASK_STATE_TRANSITION` với `failure_reason=TIMEOUT` từ stop row và outcome audits. Không kết luận `WORKER_STOPPED` trước positive proof. |

Không có network trong transaction. `STOP_REQUESTED` đọc sau crash, timeout hoặc commit ambiguity không cấp quyền effect. Mất response/cancel/confirmation tx fail đi qua containment 3C, không retry `/kill`. Đề xuất interface trusted host `AcquireEffect(ctx, pairID, "TIMEOUT_MONITOR") (ScopedPermit, error)`: chỉ cấp sau startup classification-complete và khi admission chung đã mở; `StartTimeout` nhận provider, acquire scoped permit, lấy thời gian UTC sau acquire, dùng luồng reservation/effect 3C, rồi release đúng một lần sau outcome/incomplete. Store kiểm tra lại `now >= deadline` bằng thời gian đó trong Tx R; preflight observation trước permit không thay thế fresh GET của 3C sau reservation. `HostQuiescence.Acquire` phải đóng chính admission này cùng send/restore/provisioning/stop và drain/join mọi permit holder trước startup classification; local mutex không chứng minh nhiều process. Runtime `TimeoutMonitor.Tick` không được chạy trong exclusive startup `Run`. Thiếu provider hoặc policy hợp lệ: timeout path disabled và host báo dependency; không chạy best-effort. Fake provider chỉ chứng minh thư viện gọi đúng thứ tự, không chứng minh host thật hoặc nhiều process.

## 5. Eligibility và thứ tự ưu tiên

1. Startup chỉ phân loại persisted restore/provision/send/stop intent. Không phát timeout stop, ngay cả khi `now >= deadline`.
2. Stop bất kỳ đã gắn attempt (kể cả `STOP_REQUESTED`, `STOP_CALL_SUCCEEDED`, unknown hoặc resolved nhưng attempt còn mở) chặn stop mới. Poller chỉ reconcile stop có quyền observation; không replay kill.
3. `BLOCKED`/`HUMAN_REQUIRED`, AO `blocked`, unresolved Pair risk/hold/quarantine có đường escalation/reconciliation trước; không dùng timeout để vượt Pair guard.
4. `DISPATCHED` chưa có observation execution và D8/P04 handoff không được ép `RUNNING` chỉ vì deadline. Handoff có bằng chứng hợp lệ xử lý trước timeout.
5. Fresh 404, terminated, identity/generation mismatch **trước Tx R** phải đi nhánh lifecycle hiện có, không tạo timeout intent hay suy physical stop. Fresh GET ở ngoài Tx R, Tx R vẫn CAS exact state/lineage. AO unavailable/protocol invalid thì fail-closed, không reserve. Nếu fresh GET riêng của 3C **sau Tx R** thấy 404/terminated/mismatch, D11 ghi logical outcome và không gọi `/kill`; persisted `TIMEOUT` chỉ là trigger của request đã commit, không phải khẳng định nguyên nhân worker chết.
6. Exact current open `RUNNING/SEND_CONFIRMED`, session/generation khớp, no stop/risk, budget hợp lệ: `now < deadline` tiếp tục observation kể cả `waiting_input`; `now >= deadline` mới tạo timeout stop request. Không đóng attempt tại bước này.

Nếu observation và monitor cạnh tranh, Store CAS/unique index quyết định winner; loser đọc lại state. Không đổi TaskState graph. D11 stop confirmation deadline so sánh strict `termination_confirmed_at < confirmation_deadline_at` như đã duyệt; execution deadline dùng `now >= execution_deadline_at` để phát yêu cầu mới, không tự xác nhận termination.

## 6. Legacy, crash và rollback

| Tình huống | Kết quả bắt buộc |
| --- | --- |
| V4 `SEND_CONFIRMED` current open không có budget | Startup `Run` incomplete, host admission/serve giữ đóng. Không backfill từ `now`, `started_at`, policy hiện tại. Chỉ Tx L có verified historical policy/evidence mới mở đường; nếu không có, operator dùng manual stop/reconciliation dưới host authority riêng, không gọi timeout monitor. |
| HTTP 200 nhưng Tx S rollback | `SEND_REQUESTED`/NULL còn lại; D5 containment, không resend. Không có budget mồ côi. |
| Tx S commit ambiguity | Reread exact dispatch+budget+audit; đủ tuple thì giữ winner, thiếu thì D5 containment theo CAS, không suy từ HTTP 200. |
| Crash sau Tx S, đổi policy | Dùng duration/deadline/ref đã persist, không tính lại. |
| Crash trước Tx R | Không stop row/effect; monitor sau admission có thể đánh giá lại. |
| Crash sau Tx R trước effect | Stop row và cause còn nguyên, startup quiescence phân loại; không cấp lại effect permit. |
| `/kill` unknown hoặc Tx O audit lỗi | 3C containment/D11 rollback; cause không thay; không reissue. |

## 7. Verification, reconciliation và quyết định còn cần duyệt

Behavior tests: fresh/v1–v4→v5 và rollback; immutable budget; confirm tx rollback/commit ambiguity; malformed/overflow/legacy row; before/equal/after execution deadline, prolonged `waiting_input`, restart đổi policy; `DISPATCHED`/blocked/P04/terminated/404/mismatch precedence; hai monitor cạnh tranh; host admission và quiescence barrier; stop đã tồn tại; crash sau intent; timeout cause qua D11 outcomes và audit rollback. Fake provider chứng minh library call contract, không chứng minh host bootstrap/cross-process deployment.

**Chưa được duyệt:** v5 DDL/triggers, hai `binding_basis` literal, `EXECUTION_BUDGET_LEGACY_BOUND`, stop cause persistence, legacy authority và các precedence chi tiết, scope `internal/dispatch`, `internal/stop`, `internal/domain` cùng host admission interface. Sau design audit: reconcile `docs/02`, `05`, `06`, `12`, `14`, `21`, `22`, `phases/P03`, registry và migration version; duyệt/release contract revision 3 riêng; tiếp tục code từ `codex/p03-003d` SHA `00f64d0b26db2749eadd21b7307c98faa6b6280d`, không chép governance vào implementation diff. Không tự đóng `3D-R1-005`.
