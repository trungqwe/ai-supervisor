# ADR-016 Addendum — Send-confirmation-based execution budget

> **Status:** `EXTERNAL_APPROVED_DESIGN` theo quyết định External Supervisor tại governance SHA `3babaa2ba573fcdd7d6b34cfb5ea2fd44f1fffe4` và các ràng buộc EXEC-R1-001..003. Đây là phê duyệt **thiết kế**, không release contract revision 3, không phê duyệt implementation. `3D-R1-005=OPEN` đến implementation audit.
>
> **Authority/quan hệ:** bổ sung ADR-016 D8/D9/D11/D12/D13, FR-006, REC-004 và addendum exclusive host quiescence; [PROPOSAL-P03-004](../proposals/PROPOSAL-P03-004-send-confirmation-execution-budget.md) là hồ sơ so sánh. Chỉ supersede cách xác định origin/binding execution budget, admission timeout stop và xử lý legacy nêu rõ tại §§1–6; không supersede D11 physical proof, D12 closure, D13 no-effect-replay, stop confirmation deadline, TaskState graph, AO wire, restore/clearance hoặc các clarification lịch sử. `RelinquishTimeoutEffect` bị loại khỏi revision này.

## 1. Quyết định và phạm vi supersede

Ngân sách execution bắt đầu tại **send confirmation** HTTP 200 (`dispatch_operations.confirmed_at`), không phải actual execution start. Có thể tính cả độ trễ trước execution thực tế. D9 vẫn giữ `waiting_input` là `RUNNING` cho đến khi ngân sách hết; addendum này chỉ bổ sung persistence, deadline monitor, cause và thứ tự phục hồi. D11/D12 physical proof, TaskState graph, AO `/kill` wire chỉ có `sessionId`, hold/quarantine và D13 startup effect-replay prohibition không thay đổi. Execution deadline và `confirmation_deadline_at` của stop là hai mốc độc lập.

## 2. Registry bổ sung đã duyệt

| Loại | Literal | Ý nghĩa, giới hạn |
| --- | --- | --- |
| `attempt_execution_budgets.binding_basis` | `SEND_CONFIRMATION_ATOMIC` | Policy snapshot và deadline được ghi cùng transaction xác nhận send. |
| Cùng enum | `LEGACY_OPERATOR_VERIFIED` | Operator đã xác thực cung cấp bằng chứng policy lịch sử; không dựng lịch sử từ policy hiện tại. |
| `stop_operations.initiating_failure_reason` | `TIMEOUT` | Cause của yêu cầu stop do hết execution budget; lưu bền vững trước `/kill`. Không phải D11 resolution/disposition. |
| `audit_events.event_type` mới | `EXECUTION_BUDGET_LEGACY_BOUND` | Ghi authority, evidence ref, exact lineage, policy/origin/deadline của legacy binding; không là physical proof hoặc quarantine clearance. |

Các event đã duyệt được dùng đúng vai trò: `DISPATCH_SEND_CONFIRMED` ghi budget atomic; `STOP_OPERATION_REQUESTED` ghi cause TIMEOUT khi reserve; `TASK_STATE_TRANSITION` ghi `failure_reason=TIMEOUT` khi D11 đóng live attempt; `STOP_OPERATION_CONFIRMED`, `STOP_CONFIRMATION_TIMEOUT`, `STOP_OPERATION_TARGET_ABSENT`, `STOP_OPERATION_RESOLVED` ghi outcome tương ứng. `QUARANTINE_RESOLVED_PHYSICAL` chỉ xuất hiện khi D11/D6 thật sự clear. Không phát event physical/clearance chỉ vì execution budget hết. `recovery_disposition` tiếp tục theo D11 outcome, không gán `TIMEOUT` vào enum disposition.

## 3. DDL v5 đã duyệt ở cấp thiết kế

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

`EXECUTION_BUDGET_LEGACY_BOUND` audit details chứa exact Pair/task/contract/attempt/session/generation, dispatch operation, confirmed origin, duration/deadline, policy ref, binding basis, verified principal, authority scope `LEGACY_EXECUTION_BUDGET_RECONCILIATION` và evidence ref; không ghi raw secret. Stop manual dùng authority scope riêng `MANUAL_STOP_RECONCILIATION`, ghi verified principal/reason/evidence refs vào `STOP_OPERATION_REQUESTED` và outcome audits của 3C trên exact lineage; không dùng event budget binding khi không có policy evidence. Hai scope không tự thay thế nhau và một principal string không là bằng chứng host authentication.

Không có network trong transaction. `STOP_REQUESTED` đọc sau crash, timeout hoặc commit ambiguity không cấp quyền effect. Mất response/cancel/confirmation tx fail đi qua containment 3C, không retry `/kill`. Đề xuất interface trusted host `AcquireEffect(ctx, pairID, "TIMEOUT_MONITOR") (ScopedPermit, error)`: chỉ cấp sau startup classification-complete và khi admission chung đã mở; `StartTimeout` nhận provider, acquire scoped permit, lấy thời gian UTC sau acquire, dùng luồng reservation/effect 3C, rồi release đúng một lần sau outcome/incomplete. Store kiểm tra lại `now >= deadline` bằng thời gian đó trong Tx R; preflight observation trước permit không thay thế fresh GET của 3C sau reservation. `HostQuiescence.Acquire` phải đóng chính admission này cùng send/restore/provisioning/stop và drain/join mọi permit holder trước startup classification; local mutex không chứng minh nhiều process. Runtime `TimeoutMonitor.Tick` không được chạy trong exclusive startup `Run`. Thiếu provider hoặc policy hợp lệ: timeout path disabled và host báo dependency; không chạy best-effort. Fake provider chỉ chứng minh thư viện gọi đúng thứ tự, không chứng minh host thật hoặc nhiều process.

## 5. Eligibility và thứ tự ưu tiên

1. Startup chỉ phân loại persisted restore/provision/send/stop intent. Không phát timeout stop, ngay cả khi `now >= deadline`.
2. Stop bất kỳ đã gắn attempt (kể cả `STOP_REQUESTED`, `STOP_CALL_SUCCEEDED`, unknown hoặc resolved nhưng attempt còn mở) chặn stop mới. Poller chỉ reconcile stop có quyền observation; không replay kill.
3. `BLOCKED`/`HUMAN_REQUIRED`, AO `blocked`, unresolved Pair risk/hold/quarantine có đường escalation/reconciliation trước; không dùng timeout để vượt Pair guard.
4. `DISPATCHED` chưa có observation execution và D8/P04 handoff không được ép `RUNNING` chỉ vì deadline. Handoff có bằng chứng hợp lệ xử lý trước timeout.
5. Fresh 404, terminated, identity/generation mismatch **trước Tx R** phải đi nhánh lifecycle hiện có, không tạo timeout intent hay suy physical stop. Fresh GET ở ngoài Tx R, Tx R vẫn CAS exact state/lineage. AO unavailable/protocol invalid thì fail-closed, không reserve. Nếu fresh GET riêng của 3C **sau Tx R** thấy 404/terminated/mismatch, D11 ghi logical outcome và không gọi `/kill`; persisted `TIMEOUT` chỉ là trigger của request đã commit, không phải khẳng định nguyên nhân worker chết.
6. Exact current open `RUNNING/SEND_CONFIRMED`, session/generation khớp, no stop/risk, budget hợp lệ: `now < deadline` tiếp tục observation kể cả `waiting_input`; `now >= deadline` mới tạo timeout stop request. Không đóng attempt tại bước này.

Nếu observation và monitor cạnh tranh, Store CAS/unique index quyết định winner; loser đọc lại state. Không đổi TaskState graph. D11 stop confirmation deadline so sánh strict `termination_confirmed_at < confirmation_deadline_at` như đã duyệt; execution deadline dùng `now >= execution_deadline_at` để phát yêu cầu mới, không tự xác nhận termination.

### 5.1. Ranh giới quan sát và quyền effect (EXEC-R1-002; đã duyệt)

Tx R chỉ nhận observation **fresh, exact** từ GET ngoài transaction và kiểm tra lại toàn bộ durable guard bằng CAS. Thứ tự ưu tiên: intent/stop hoặc Pair risk đang tồn tại → identity/generation mismatch, 404, termination, outage/protocol error → blocked/exited/idle và handoff → active/waiting_input. `RUNNING` trong DB không chứng minh AO vẫn active. Mỗi lần GET chỉ mô tả một thời điểm; AO có thể đổi trạng thái ngay sau đó. GET thứ hai của 3C sau Tx R phải xét lại admissibility trước `/kill`, không được coi GET thứ nhất là bảo đảm.

Trong bảng, **I** là Tx R đã commit `STOP_REQUESTED/IN_FLIGHT`, cause `TIMEOUT`, `STOP_OPERATION_REQUESTED` audit, session và exact attempt `QUARANTINED`, TaskState `RUNNING`, `ended_at=NULL`. Đây là quarantine ngăn effect khác, không là D11 closure. Tx R dùng **chính** `ReserveStopOperation` của 3C: intent, hai quarantine và audit commit hoặc rollback cùng nhau, không có transaction quarantine rời. **L** là caller thắng Tx R còn giữ scoped effect permit trong stack frame. Quarantine do I vừa tạo là trạng thái sở hữu hợp lệ của L, không được khiến pre-effect guard tự từ chối L; guard vẫn từ chối mọi risk/owner mới hoặc lineage đổi. Khi L bỏ quyền, giữ I nguyên trạng và kết thúc call; persisted I không cấp quyền cho bất kỳ caller/restart nào. Chỉ exclusive startup recovery sau drain/join hoặc human reconciliation mới phân loại tiếp; không ghi resolution/audit giả ở bước bỏ quyền.

| Fresh observation | Trước Tx R (chưa có intent timeout) | Sau Tx R, GET thứ hai của L / ngay trước `/kill` | Durable kết quả, audit và quyền tiếp tục |
| --- | --- | --- | --- |
| `active`, exact | `RUNNING/SEND_CONFIRMED`, đủ budget và `now >= deadline`: được thử Tx R; trước hạn chỉ quan sát. | Recheck exact Pair/session/generation, stop ownership, permit còn hiệu lực; L được gọi `/kill` **một lần**. | I đến khi 3C ghi HTTP/call outcome; D11/D12 quyết định closure, disposition và audit. GET không bảo đảm AO không đổi sau đó. |
| `waiting_input`, exact | Cùng nhánh `active`; D9 vẫn tính budget, không reset deadline. | Cùng recheck và quyền một lần như `active`. | I rồi outcome 3C; không coi waiting là stop proof. |
| `blocked`, exact | Nhánh blocked escalation hiện có, không reserve timeout. | L bỏ quyền, không kill theo timeout. | Giữ I `STOP_REQUESTED/IN_FLIGHT`, `RUNNING`, attempt mở và double quarantine; chỉ request audit Tx R. Blocked escalation chỉ được CAS sau stop reconciliation, không ghi đè owner. |
| `idle`, exact; handoff available/unavailable | Handoff/MISSED_ACTIVE_WINDOW hiện có xét trước, không reserve timeout. | L bỏ quyền; handoff không được biến intent thành effect permit. | Giữ I như blocked; handoff chỉ chạy sau giải quyết stop/risk, không xóa I để retry. |
| `exited`, `isTerminated=false` | Lifecycle `exited` hiện có: không suy physical termination, không reserve timeout. | L bỏ quyền, không kill theo timeout khi trạng thái đã đổi. | Giữ I như blocked; giữ quarantine tới reconciliation. |
| `isTerminated=true`, exact generation | D12 terminal branch hiện có, không reserve timeout. | Không kill; 3C `CommitStopOutcome` ghi logical `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED`. | D11/D12 atomic closure/disposition/quarantine và `STOP_OPERATION_RESOLVED` + TaskState audit; TIMEOUT chỉ là cause intent, không là physical proof. |
| HTTP 404 | Nhánh `SESSION_ABSENT` hiện có, không reserve timeout. | Không kill; 3C ghi `STOP_TARGET_ABSENT`. | D11/D12 atomic closure/disposition/quarantine, `STOP_OPERATION_TARGET_ABSENT` + TaskState audit; không clear vì vắng mặt. |
| Session identity hoặc generation mismatch | Nhánh mismatch/containment hiện có, không reserve timeout. | Không kill. Generation mismatch dùng D11 `STOP_GENERATION_MISMATCH`; identity/protocol không được giả thành generation mismatch. | Generation mismatch: D11/D12 atomic closure/quarantine/`STOP_OPERATION_RESOLVED`. Identity sai: L bỏ quyền và giữ I; không ghi physical proof. |
| Query outage hoặc response/protocol invalid | Fail-closed, không reserve, không tự ghi timeout stop. | Không kill; không biến lỗi GET thành 404/terminated/alive. | L bỏ quyền, giữ I `IN_FLIGHT`, request audit duy nhất, quarantine và human/startup reconciliation. Không tự retry effect. |

Sau Tx R, stop owner có ưu tiên trước generic poller và timeout monitor; generic observation CAS phải từ chối khi stop gắn exact attempt còn hiện hành. Trước `/kill`, L kiểm tra permit còn thuộc stack frame đang hoạt động và durable ownership chưa đổi; host quiescence phải **drain/join**, không thu hồi một permit rồi tự nhận caller đã kết thúc. Kiểm tra này không cấp quyền từ row và không hứa hẹn AO đứng yên giữa GET và effect. Nếu ownership/recheck lỗi, không phát effect; giữ I và trả lỗi, stale writer đọc lại winner. Không xóa intent để monitor khác reserve lại. 404, terminated và generation mismatch sau Tx R đi D11 hiện hữu với stop resolution, D12 closure và audit nguyên tử; các nhánh còn lại chỉ phân loại I sau exclusive host quiescence. Không có `RelinquishTimeoutEffect` trong revision này.

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

### 6.1. Trusted legacy maintenance khi Run incomplete (EXEC-R1-001; đã duyệt)

Host cấp `AcquireMaintenance(ctx, Pair, purpose)` dưới **cùng admission/quiescence boundary** nhưng là scope độc quyền riêng, không phải normal `AcquireEffect`: normal admission/serve vẫn đóng. Trước khi cấp, host cấm Run mới, monitor/poller và mọi stop/send/restore/provisioning hoặc mutation xung đột, drain **và join** các caller đã giữ quyền, kể cả qua các process dùng cùng DB/Pair. Không giữ SQLite transaction khi drain hoặc gọi AO. Process-local mutex, bool, timeout hay row reread không chứng minh exclusivity. `Run` không acquire maintenance từ bên trong scope và maintenance không gọi `Run` trước khi release: tránh chu trình chờ. Thiếu provider hoặc không chứng minh cross-process drain thì fail-closed.

`LEGACY_BUDGET_BINDING` chỉ dành cho verified operator principal có scope `LEGACY_EXECUTION_BUDGET_RECONCILIATION` và evidence về `confirmed_at`, policy ref/duration **có hiệu lực tại send**. Tx L CAS exact lineage, `SEND_CONFIRMED`, budget absent, insert immutable budget + `EXECUTION_BUDGET_LEGACY_BOUND` audit nguyên tử; audit/CAS lỗi rollback. Replay exact evidence chỉ đọc. Maintenance scope không tự cấp authority operator.

Nếu không có bằng chứng policy lịch sử, `MANUAL_STOP_RECONCILIATION` cần verified principal với **scope riêng** cho residual execution risk; không tạo budget. Host-gated wrapper dùng đúng reservation/effect một lần của 3C cho `RUNNING_ATTEMPT_STOP` **chỉ khi exact current attempt vẫn mở và TaskState thật sự `RUNNING`**, rồi D11/D12, Class B/C và D6 theo authority/evidence hiện có. Wrapper nhận maintenance permit sống và kiểm tra exact Pair/attempt; không gọi normal `AcquireEffect` trong maintenance (sẽ tự deadlock). Legacy `DISPATCHED` không được ép sang `RUNNING` để stop. Nếu thiếu policy evidence và chưa có đường reconciliation hợp lệ cho `DISPATCHED`, tiếp tục fail-closed, không mở serve/admission. Không dùng principal string hay Store row làm permit, không gọi `/kill` khi không chứng minh caller thắng intent mới. Nếu stop intent đã tồn tại, chỉ reconciliation đã duyệt, không reissue.

Hủy trước Tx L/reservation: không mutation. Hủy sau commit budget: row/audit còn nguyên, không hoàn tác; sau stop intent/effect không rõ: giữ stop/Pair quarantine, không retry kill. Crash host giữ normal admission đóng theo trusted host contract; restart phải acquire exclusive startup quiescence và Run lại. Scope maintenance release đúng một lần trong mọi nhánh, kể cả lỗi/cancel; **release không mở admission/serve**. Chỉ Run sau đó phân loại hoàn tất theo các guard hiện hành mới cho phép host mở normal admission; `classified-pending-AO` vẫn khóa Pair liên quan. Test barrier: Run đợi maintenance drain và ngược lại, caller mới bị từ chối, Tx L audit lỗi rollback, cancel/crash giữ admission đóng, không deadlock với Stop wrapper; fake provider chỉ chứng minh thư viện, không chứng minh host thật.

## 7. Verification, reconciliation và quyết định còn cần duyệt

Behavior tests: fresh/v1–v4→v5 và rollback; immutable budget; confirm tx rollback/commit ambiguity; malformed/overflow/legacy row; before/equal/after execution deadline, prolonged `waiting_input`, restart đổi policy; `DISPATCHED`/blocked/P04/terminated/404/mismatch precedence; hai monitor cạnh tranh; host admission và quiescence barrier; stop đã tồn tại; crash sau intent; timeout cause qua D11 outcomes và audit rollback. Fake provider chứng minh library call contract, không chứng minh host bootstrap/cross-process deployment.

**Quyết định thiết kế được Supervisor chấp thuận tại `3babaa2`:** v5 DDL/triggers và binding immutable, maintenance scope với authority riêng, Tx R tái sử dụng 3C reservation/double quarantine, các precedence §5.1 và giới hạn legacy `DISPATCHED`; không chọn `RelinquishTimeoutEffect`. Addendum chính thức và canonical reconciliation ghi quyết định này; contract revision 3 vẫn phải qua release riêng. Tiếp tục code từ `codex/p03-003d` SHA `00f64d0b26db2749eadd21b7307c98faa6b6280d` chỉ **sau** release, không chép governance vào implementation diff. Không tự đóng `3D-R1-005`.
