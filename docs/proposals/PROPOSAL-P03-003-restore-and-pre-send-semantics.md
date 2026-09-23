# PROPOSAL-P03-003: Giao thức restore có chủ quyền và hoàn thiện dispatch pre-send

## Metadata

- **Author / Date**: Codex / 2026-09-23
- **Related Phase / Task**: P03 / TASK-P03-003B; draft `docs/tasks/DRAFT_TASK_CONTRACT_P03_003B.md`
- **Status**: `PENDING_REVIEW`; `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL = OPEN`; 3B chưa release
- **Decision baseline**: `c632707da306d45cc1689fa4c1bf7d83846ecf24`; contract `base_sha = b8b0c95576d87677e8d48210d9838cc2f599752a`
- **Upstream pin**: `Untrivial-ai/agent-orchestrator` v0.13.0, `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`
- **Authority**: `docs/02_REQUIREMENTS.md` FR-005/FR-006/NFR-003/NFR-005; `docs/24_CHANGE_GOVERNANCE.md`; accepted ADR-016 D3–D7/D12/D13, §4/§22/§27; `docs/12_UPSTREAM_INTEGRATION.md`; `docs/14_FAILURE_RECOVERY.md` REC-002; `docs/sources/SOURCE_REGISTRY.md` và `REUSE_MATRIX.md`.

## 1. Observation và ranh giới đã có

ADR-016 D3 yêu cầu restore WorkerSession hiện hữu thay vì spawn session thay thế; D4 giữ snapshot attempt bất biến; D6 khóa cả Pair/attempt khi có rủi ro chưa giải quyết. ADR §4 và registry đã có `WORKER_TERMINATION_UNKNOWN`; D7 đã chọn `FAILED`/`STALE_EXECUTION_GENERATION` cho generation mismatch; D12 đã bắt buộc `AtomicTerminalTransition` với `ended_at`, disposition và `TASK_STATE_TRANSITION` chứa error details trong cùng transaction. Các điểm này **không phải** quyết định mới. Phần còn thiếu là restore intent/exclusion, outcome không xác định, literal `dispatch_operations.resolution_state` và vài observation trước send.

SOURCE_REGISTRY xác nhận AO là active runtime; REUSE_MATRIX giao AO quản lý worktree, process và harness. Supervisor chỉ thêm durable ownership/guards, không tự dựng runtime hoặc gọi AO ngoài adapter. `go-test-p03-003b` metadata đã được phê duyệt; phê duyệt này không mở gate implementation.

### 1.1 Problem Solved

Đóng khoảng crash giữa `/restore` và local confirmation, khóa mọi caller dùng cùng Pair trong giai đoạn restore, và định nghĩa kết quả pre-send/send còn thiếu mà không biến observation không chắc chắn thành delivery hoặc physical termination đã được chứng minh.

### 1.2 Existing Upstream Capability Checked — pinned AO wire và implementation

| Nguồn tại đúng pin | Sự kiện kiểm chứng được | Giới hạn suy luận |
|---|---|
| `backend/internal/httpd/apispec/openapi.yaml:5245-5274`, `:10693-10712`; `controllers/sessions.go:195,1250-1261`; `controllers/dto.go:722-728` | `POST /api/v1/sessions/{sessionId}/restore`, không request body/idempotency key; HTTP 200 trả `ok`, `sessionId`, `restoreMode`, `session`; 404/409/500 có thể trả về. | Cùng `sessionId` không phải dedup key, HTTP 409 không chứng minh effect của caller này chưa xảy ra. |
| `backend/internal/service/session/service.go:580-590` | Service gọi `manager.RestoreWithMode`, chuyển kết quả thành SessionView. | Service không cung cấp operation ID để Supervisor đối chiếu. |
| `backend/internal/session_manager/manager.go:1961-2025,2212-2365,4702-4800` | Manager đòi session terminated; kiểm tra workspace/handle; restore workspace, relaunch runtime, gán `RuntimeLaunchID`; có `native`, `saved_prompt`, `fresh`. Fallback `saved_prompt` có thể phát lại prompt cũ; lỗi sau khi AO đã có effect vẫn có thể xảy ra. | `GET SessionView` sau restart thấy generation mới chỉ chứng minh runtime khác, không chứng minh restore nào, mode nào hoặc không replay prompt. Không coi response mất là idempotent success. |
| `backend/internal/session_manager/session_input.go:100-129` | AO chặn thao tác agent chồng nhau bằng `agentOpMu`/`agentOperations` trong process. | Không phải exclusion bền vững của Supervisor qua crash, nhiều caller hoặc các loại operation khác. |
| `backend/internal/adapters/agent/agy/agy.go:GetRestoreCommand` | Agy dùng `--conversation` nếu có agent session ID. | Không chứng minh mọi restore sẽ là `native`; manager có fallback và public preflight không tiết lộ mode trước effect. |

### 1.3 Why Existing Capability is Insufficient

AO có restore runtime và bảo vệ thao tác trong một process, nhưng public route không nhận Supervisor operation ID, expected generation hoặc idempotency key. AO không lưu Supervisor Pair reservation/audit và không thể xác nhận causal ownership sau mất response. Vì vậy cần StateStore protocol riêng; provenance local vẫn không thể biến một effect upstream mơ hồ thành xác nhận tự động.

## 2. Proposed Idea — phương án khuyến nghị

**Khuyến nghị: bổ sung một restore operation bền vững có khóa Pair chung, giữ route `/restore` của AO, và fail-closed mọi outcome không có bằng chứng HTTP 200 đã commit.** Đây là một bổ sung kiến trúc hẹp cho lifecycle, không đổi wire AO. Restore vẫn thuộc mục tiêu 3B; chưa được triển khai trước khi ADR addendum và canonical reconciliation được duyệt.

### 2.1 Schema, token và audit đề xuất mới (chưa được phê duyệt)

Migration kế tiếp (không sửa v3 lịch sử) thêm `pair_restore_operations`:

```sql
CREATE TABLE pair_restore_operations (
  operation_id TEXT PRIMARY KEY,
  pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
  session_id TEXT NOT NULL,
  expected_generation TEXT NOT NULL,
  observed_generation TEXT,
  stage TEXT NOT NULL CHECK (stage IN
    ('RESTORE_REQUESTED','RESTORE_CONFIRMED','RESTORE_OUTCOME_UNKNOWN','RESTORE_RESOLVED')),
  restore_mode TEXT CHECK (restore_mode IN ('native','saved_prompt','fresh')),
  actor TEXT NOT NULL,
  authorization_event_id TEXT REFERENCES audit_events(event_id) ON DELETE RESTRICT,
  requested_at TEXT NOT NULL,
  confirmed_at TEXT,
  resolved_at TEXT,
  resolved_by TEXT,
  resolution_notes TEXT
);
CREATE UNIQUE INDEX idx_pair_restore_unresolved
  ON pair_restore_operations(pair_id)
  WHERE stage IN ('RESTORE_REQUESTED','RESTORE_CONFIRMED','RESTORE_OUTCOME_UNKNOWN');
```

`RESTORE_REQUESTED` là intent đã commit trước effect, đồng thời là quyền sở hữu duy nhất trên Pair. `RESTORE_CONFIRMED` chỉ chứng minh response HTTP 200 hợp lệ và generation/status đã được kiểm tra trong post-effect transaction; lane vẫn khóa, nó **không** tự chứng minh worker idle hoặc xóa quarantine. `RESTORE_OUTCOME_UNKNOWN` giữ khóa sau lỗi/khởi động lại. `RESTORE_RESOLVED` chỉ sau quan sát quiescent hợp lệ đối với safe automatic subset, hoặc hành động operator có thẩm quyền; không đồng nghĩa clearance nếu quarantine còn hiệu lực. Thêm audit types `PAIR_RESTORE_RISK_ACCEPTED`, `PAIR_RESTORE_REQUESTED`, `PAIR_RESTORE_CONFIRMED`, `PAIR_RESTORE_OUTCOME_UNKNOWN`, `PAIR_RESTORE_RESOLVED`; `authorization_event_id` trỏ tới risk acceptance event trước effect, không phải clearance event. Hash chain dùng cơ chế hiện có; provenance phải chứa operation ID, Pair, session, expected/observed generation, actor, restore mode và loại bằng chứng. Không sao chép tên token sang `pair_provisioning_operations` hoặc `dispatch_operations`.

**Khóa giữa các loại caller**: trong cùng SQLite write transaction, `ReservePairRestore` kiểm tra WorkerSession thuộc Pair, exact session/generation, `TERMINATED`, không có provisioning unresolved, dispatch open/bound/send unresolved, stop unresolved hoặc restore unresolved. Automatic path đòi Pair/attempt `CLEAN`; Pair/attempt đang `QUARANTINED` chỉ được vào human-authorized recovery path với risk/lineage audit, và restore không tự clear quarantine. Chỉ sau audit intent commit mới gọi AO. Mọi transaction `ReservePairProvisioning`, `PrepareBoundDispatch`, `RecordSendRequested`, `ReserveStop`/stop reissue và clearance phải kiểm tra restore unresolved; ngược lại reserve restore phải kiểm tra operation của các loại ấy. Scanner 3D đọc restore intent trước các bước provisioning/dispatch/stop và không tự phát effect. Thứ tự SQLite write serialization + CAS/index loại hai caller cùng thắng. Emergency safety stop trong khi restore unresolved cần quyết định operator riêng, không được lách guard bằng generic stop.

**Pre-effect eligibility**: restore cùng session ID, không replacement; phải có bằng chứng Pair được Supervisor sở hữu và không còn execution chưa resolve. Vì AO có thể `saved_prompt` và tái chạy lệnh cũ, automatic restore chỉ được phép nếu provenance của session chứng minh chưa có task prompt/attempt cũ có thể replay **và** phiên AO thuộc phạm vi độc quyền của Supervisor. Với dữ liệu pin/schema hiện tại, safe automatic subset **chưa được chứng minh**: mặc định 3B chỉ cho phép restore do operator có thẩm quyền khởi tạo, lưu `authorization_event_id`/risk acknowledgement trước effect và giữ quarantine đến clearance D6. Không dùng `saved_prompt` response sau effect để hợp thức hóa việc gọi tự động đã không đủ điều kiện. Cần audit thực nghiệm với pinned AO xem safe automatic subset có tồn tại; nếu rỗng, giữ restore tích hợp có giám sát trong 3B và đề xuất upstream preflight/idempotency capability ở roadmap, không bỏ restore khỏi 3B.

### 2.2 Transaction và bằng chứng

1. **T0 — reservation**: transaction CAS các guard trên, xác minh `PAIR_RESTORE_RISK_ACCEPTED` đã được người có thẩm quyền ghi cho exact Pair/session/expected generation (hoặc safe automatic eligibility đã được phê duyệt), insert `RESTORE_REQUESTED` với `authorization_event_id` và `PAIR_RESTORE_REQUESTED` audit, commit. Audit fail/commit fail ⇒ không gọi AO. Không đặt network call trong transaction.
2. **T1 — effect**: duy nhất owner của operation gọi `ResumeWorker(sessionId)` một lần. Không tự reissue kể cả khi local biết chắc request chưa phát byte nhưng intent đã commit; sau crash không thể chứng minh điều đó. HTTP 409/404/transport/context error/response malformed đi theo unknown path trừ khi có bằng chứng chắc chắn không effect được phê duyệt riêng.
3. **T2 — confirmation**: chỉ sau HTTP 200, `ok=true`, response identity đúng và SessionView có nonempty generation, so với `GetWorkerStatus` cùng identity/generation và `isTerminated=false`. Transaction CAS `RESTORE_REQUESTED` + Pair/session/old generation, ghi `RESTORE_CONFIRMED`, observed generation/mode, cập nhật **cùng** WorkerSession, append `PAIR_RESTORE_CONFIRMED`; tất cả cùng commit. Không sửa `task_attempts`/`dispatch_operations` đã bind. Nếu mode/status cho thấy turn cũ còn chạy, vẫn chặn dispatch; `RESTORE_CONFIRMED` không phải pre-send admission.
4. **T3 — resolution/unknown**: với `RESTORE_CONFIRMED`, chỉ CAS sang `RESTORE_RESOLVED` + audit khi quan sát cùng generation `idle`/`waiting_input`, safe automatic eligibility còn đúng và không còn quarantine; nếu không, giữ lock và chuyển human. Nếu response/confirmation không chứng minh được, transaction CAS `RESTORE_REQUESTED -> RESTORE_OUTCOME_UNKNOWN` + audit, giữ Pair bị khóa và áp dụng double-gated quarantine cho attempt liên quan theo D6/REC-002. Nếu transaction này cũng fail, `RESTORE_REQUESTED` vẫn là unresolved lock. Scanner 3D phân loại và giao người có thẩm quyền; không retry effect.

**Bằng chứng restart**: `RESTORE_CONFIRMED` + audit hash chain + cùng session/generation quan sát được là đủ để tiếp tục kiểm tra activity, không đủ để bỏ qua pre-send. `RESTORE_REQUESTED` hoặc `RESTORE_OUTCOME_UNKNOWN` sau crash, dù AO `GET` trả generation mới, không chứng minh operation nào tạo ra generation hoặc có replay; bắt buộc human reconciliation/audit. 404 chứng minh vắng mặt công khai, không chứng minh physical termination. Operator có thể xác nhận bằng chứng ngoài public API hoặc chấp nhận rủi ro theo Class B/C, ghi `resolved_by`/notes, nhưng chỉ transaction clearance độc lập theo D6 mới gỡ quarantine. Nếu AO đã restore mà local CAS/audit/commit thất bại, reread durable row: nếu caller khác đã `RESTORE_CONFIRMED` với đúng tuple/audit thì nhận kết quả đã commit; nếu chưa, khóa unknown và human. Không phát lại `/restore`.

### 2.3 State/transition table và crash matrix

| Sự kiện | Restore stage | WorkerSession / attempt | Tiếp tục |
|---|---|---|---|
| Reservation CAS/audit fail | Không có row mới | Không đổi | Có thể đánh giá lại guard; zero AO call. |
| Intent commit, AO chưa gọi hoặc crash trong call | `RESTORE_REQUESTED` | Giữ binding cũ, lane khóa | 3D chuyển unknown; không reissue. |
| HTTP 200 hợp lệ, T2 commit | `RESTORE_CONFIRMED`, lane vẫn khóa | Cùng session ID, generation mới; snapshot attempt cũ giữ nguyên | T3 kiểm tra quiescence rồi resolve; vẫn cần quarantine clearance nếu có và pre-send status/lineage check. |
| Lost/invalid response, HTTP 409/404, canceled context | `RESTORE_OUTCOME_UNKNOWN` nếu containment commit | Double-gated khi có attempt liên quan; không thay session | Human; 404 không tự clear. |
| AO thành công nhưng T2 CAS/audit/commit fail | `RESTORE_REQUESTED` hoặc `RESTORE_OUTCOME_UNKNOWN`, trừ khi reread chứng minh caller khác đã commit | DB rollback local updates; AO có thể đang chạy | Reread; stale caller dừng, còn lại human; không retry. |
| Restore sau `DISPATCH_BOUND` hoặc generation đổi | Không reserve nếu bound attempt còn mở; nếu AO đổi ngoài Supervisor, không đổi snapshot | D7 `FAILED`/`STALE_EXECUTION_GENERATION` + D12 closure; quarantine proposal ở §3 | Chỉ attempt mới sau governed resolution. |

Tại `DISPATCH_BOUND` với `exited`, ADR §22 giữ attempt mở và cấm send. Đề xuất không gọi `/restore` trên attempt đang bind. Nếu AO chỉ exited nhưng session chưa terminated, `/restore` có thể trả 409. Sau quyết định operator bỏ attempt cũ, thực hiện D12 `DISPATCHED -> FAILED` với literal disposition mới `PRE_SEND_PROCESS_EXITED`, `ended_at=now`, `TASK_STATE_TRANSITION`/error details và quarantine atomically; sau đó 3C `QUARANTINE_CLEANUP` trên attempt đã đóng xác nhận physical termination theo D11. Chỉ sau governed clearance mới restore cùng session trước binding của attempt mới. Không dùng `/resume-agent` thay thế vì đó là endpoint khác với D3. Đây là semantics mới cần phê duyệt; cho đến lúc đó giữ trạng thái §22.

## 3. Semantics dispatch đề xuất để chốt một lượt

### 3.1 Literal unknown delivery và CAS

Đề xuất thêm **`DELIVERY_OUTCOME_UNKNOWN`** cho `dispatch_operations.resolution_state` (nullable `TEXT`; thêm CHECK khi migration có thể áp dụng an toàn). D5 đã yêu cầu cập nhật cột nhưng chưa cho literal. Chỉ CAS `stage=SEND_REQUESTED AND resolution_state IS NULL` sang literal này trong cùng transaction D5: `DISPATCHED -> FAILED -> HUMAN_REQUIRED` theo cạnh hợp lệ, attempt `ended_at=now`, `recovery_disposition=UNCERTAIN_DELIVERY_CRASH`, hai quarantine `QUARANTINED`, audit `TASK_STATE_TRANSITION` với error details và `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED`. Replay cùng operation sau commit trả kết quả idempotent từ DB, không ghi audit/đóng attempt lần hai; writer thua hoặc tuple khác nhận conflict. Literal là **terminal cho dispatch operation**, không phải bằng chứng `/send` thất bại hoặc đã thành công; không được đổi thành `SEND_CONFIRMED` khi confirmation đến muộn. Clearance Class A/B/C cập nhật quarantine/clearance ledger riêng, **không xóa hoặc đổi** `DELIVERY_OUTCOME_UNKNOWN`. `dispatch resolved != quarantine cleared`.

### 3.2 Pre-send table: authority và phần bổ sung

Thứ tự ưu tiên để không ghi đè evidence: (1) reread operation/CAS ownership và stage; (2) response protocol/identity hợp lệ; (3) generation equality; (4) `isTerminated`; (5) activity whitelist. Nếu response identity sai, không dùng generation/activity của response; nếu identity đúng nhưng generation khác đồng thời terminated, ghi mismatch theo D7 trước và giữ raw observation trong audit. Mọi outcome phải reread durable state khi CAS thua, không áp dụng quyết định trên snapshot cũ.

| Observation tại `DISPATCH_BOUND` | Căn cứ và quyết định đề xuất | Quarantine / continuation |
|---|---|---|
| `isTerminated=true`, identity/generation khớp | ADR §4 + registry `WORKER_TERMINATION_UNKNOWN`; §22 `DISPATCHED -> FAILED`; D12 `ended_at`, disposition, `TASK_STATE_TRANSITION` + error details atomic. Không suy ra crash/intentional stop. | §22 không đòi impose mới; không tự clear. Attempt kết thúc. |
| Generation mismatch, identity hợp lệ | ADR D7 đã chốt `FAILED`/`STALE_EXECUTION_GENERATION`; D12 closure/audit atomic, `PRE_SEND_ADMISSIBILITY_REJECTED` theo D7. | **Bổ sung semantics**: quarantine Pair và attempt trong cùng transaction vì execution identity không còn chứng minh an toàn; Class A/B/C clearance. |
| HTTP 404 | **Bổ sung semantics**: `FAILED`/`SESSION_ABSENT` theo analogy D11/D13 nhưng không gọi đó là quyết định pre-send sẵn có; D12 closure/audit. | Quarantine Pair/attempt; chỉ Class B/C hoặc positive Class A mới clear, không replacement tự động. |
| Response `sessionId` sai / schema hoặc activity sai protocol | **Bổ sung semantics**: chưa đủ evidence terminal; giữ `DISPATCH_BOUND`, `DISPATCHED`, `ended_at=NULL`; audit `PRE_SEND_ADMISSIBILITY_REJECTED` với raw/error details, không dùng status payload. | Quarantine Pair và attempt, giữ attempt mở; human/protocol remediation theo D6, không send. |
| Query unavailable/transport timeout | ADR D13 đã có `RECOVERY_PENDING` và khóa lane cho AO unreachable; **bổ sung phạm vi pre-send**: giữ stage/task/attempt mở, disposition `RECOVERY_PENDING` và audit rejection/error details. | Quarantine Pair và attempt trong cùng transaction; 3D sở hữu retry query với caller-injected cadence, không retry `/send`; clearance theo D6. |
| `exited`, `isTerminated=false`, generation khớp | ADR D7/§22 giữ `DISPATCH_BOUND`, `DISPATCHED`, `ended_at=NULL`, audit `PRE_SEND_ADMISSIBILITY_REJECTED`, cần restore; **bổ sung quy trình đề xuất**: operator bỏ attempt bằng D12 `FAILED`/`PRE_SEND_PROCESS_EXITED`, rồi 3C `QUARANTINE_CLEANUP` như §2.3. | Giữ lane, không tạo attempt mới trước physical/administrative resolution và restore trước binding. |

Với `active/blocked/exited` matched, ADR §22 giữ attempt mở. `idle/waiting_input` matched chỉ cho phép cùng attempt CAS `SEND_REQUESTED`; không gọi `PrepareBoundDispatch` lại. Không chọn disposition mới cho `active/blocked` khi ADR không yêu cầu. Với identity/protocol/query unavailable, đề xuất audit `PRE_SEND_ADMISSIBILITY_REJECTED` cần mở rộng định nghĩa event hiện có trong addendum; nếu Supervisor muốn provenance riêng thì chốt token mới trước release.

### 3.3 Send confirmation CAS và failure containment

Trước `SEND_REQUESTED` commit: zero `/send`; lỗi transaction rollback về durable `DISPATCH_BOUND` nếu không có writer khác. Sau commit, transport/canceled context/invalid response là unknown delivery. HTTP 200 hợp lệ mà `RecordSendConfirmed` lỗi cần **reread** `(operation_id, attempt_id, stage, resolution_state, confirmed_at, task state, quarantine)`:

| Reread sau CAS/audit/commit lỗi | Hành động |
|---|---|
| `SEND_CONFIRMED` với đúng tuple và audit | Writer khác đã xác nhận; stale caller dừng, không áp D5 và không gửi lại. |
| `SEND_REQUESTED`, `resolution_state=NULL` | Confirmation thật sự chưa commit; thử D5 containment bằng CAS mới, không resend. |
| `SEND_REQUESTED`, `DELIVERY_OUTCOME_UNKNOWN` hoặc task đã escalated đúng tuple | Containment đã thắng; idempotent observation, không sửa lại. |
| Tuple/stage khác, DB không đọc được hoặc audit chain không kiểm chứng | Fail closed, không khẳng định DB luôn còn `SEND_REQUESTED`; 3D/operator điều tra. Không phát effect mới. |

Nếu D5 transaction rollback mà không có concurrent winner, `SEND_REQUESTED` vẫn còn; nếu CAS thua do writer khác, durable state có thể đã đổi. D5 và `RecordSendConfirmed` phải dùng cùng operation/attempt lineage và terminal resolution fence, không được stale confirmation ghi đè kết quả.

## 4. Alternatives, tác động và kiểm chứng bác bỏ

| Phương án | Fit / chi phí / rủi ro / tính đảo ngược |
|---|---|
| **Khuyến nghị: Supervisor restore operation + Pair-wide guards + operator authorization tại pin hiện tại** | Giữ restore tích hợp trong D3/3B qua AO route hiện có; thêm một table/index/token và sửa guard của nhiều path. Rủi ro `saved_prompt` được operator chấp nhận trước effect; outcome sau mất response vẫn cần human reconciliation. Automatic admission chỉ mở bằng quyết định mới khi proven safe subset. |
| Hoãn toàn bộ restore tích hợp, chỉ xử lý thủ công ngoài Supervisor | Ít schema/code ngắn hạn nhưng không đáp ứng D3/REC-002, không có durable exclusion/audit; kéo dài blocker và cần điều chỉnh roadmap/contract. Không được âm thầm xóa restore khỏi 3B. |
| Chỉ dùng AO same-session ID/in-process lock hoặc query generation để tự retry | Ít code nhưng không có operation correlation/durable exclusion, có thể replay prompt và đụng stop/dispatch; **bác bỏ**. |

**Disconfirming experiment bắt buộc trước release**: với pinned AO và Agy, thử terminated session từng có task prompt, tắt response tại các điểm trước/sau relaunch, quan sát `restoreMode`, generation, activity, số lần prompt chạy và khả năng xác nhận causal operation qua public GET. Nếu không chứng minh được safe automatic subset từ dữ liệu Supervisor đã ghi trước effect, chỉ cho phép operator-gated restore ở 3B; ghi roadmap cho upstream preflight/idempotency capability, không tự nâng pin hoặc sửa adapter wire.

## 5. Architectural Impact và hồ sơ sau phê duyệt

**Phân loại khuyến nghị: ADR-016 addendum**, không phải clarification thuần túy và không cần ADR mới nếu ownership boundary của AO/Supervisor giữ nguyên. Addendum phải phê duyệt lifecycle schema/token/audit, điều kiện restore/clearance và pre-send terminal/quarantine semantics. Chỉ nếu Supervisor chọn thay AO wire/ownership hoặc state machine ngoài ADR-016 thì mở ADR mới. Các dòng đã có authority ở §1 chỉ cần sửa nguồn dẫn trong draft. Trước khi release 3B, External Supervisor cần duyệt phương án §2–§3, đặc biệt automatic eligibility trước `saved_prompt`, literal resolution, pre-send 404/identity/query/exited và quyền human action.

Danh sách ảnh hưởng dự kiến sau phê duyệt (không sửa trong lượt này): addendum tại `docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md`; reconcile `docs/04_ARCHITECTURE.md`, `docs/05_DOMAIN_MODEL.md`, `docs/06_WORKFLOW_STATE_MACHINE.md`, `docs/12_UPSTREAM_INTEGRATION.md`, `docs/14_FAILURE_RECOVERY.md`, `docs/21_TRACEABILITY_MATRIX.md`, `docs/22_MODULE_PROVENANCE.md`, `docs/phases/P03_AO_INTEGRATION.md`; contract `docs/tasks/DRAFT_TASK_CONTRACT_P03_003B.md`. Nếu automatic subset rỗng, cập nhật thêm `docs/17_ROADMAP.md`. Production module dự kiến: `internal/store/migrations.go`, file mới `internal/store/migrations_v4_test.go`, `internal/domain/session_lifecycle.go`, `internal/store/session_lifecycle.go`, `internal/store/dispatch.go`, `internal/store/atomic_transitions.go`, và các file mới cần contract whitelist `internal/store/restore_transactions.go`, `internal/store/restore_transactions_test.go`, `internal/store/provisioning_transactions.go`, `internal/store/dispatch_transactions.go`, `internal/dispatch/coordinator.go`. Guard stop và scanner thuộc file/task 3C/3D sẽ được chốt tên khi contract tương ứng được lập; 3B không được tự sửa các module đó. `docs/schemas/task-contract.schema.json` không cần đổi vì JSON contract shape không đổi; `internal/ao/**` không cần đổi nếu pinned public response đủ cho phương án đã duyệt. Nếu không, dừng và xét upstream pin/capability qua governance. Scope hiện tại của draft 3B không cho phép migration/domain/stop/recovery: phải sửa và release contract riêng sau canonical reconciliation, không dùng proposal này làm quyền sửa code.

Trình tự: quyết định addendum → reconcile canonical specs/traceability → sửa JSON contract, scope, AC và test plan của draft 3B (giữ `base_sha` theo governance nếu không có quyết định khác) → full schema/`ValidateRaw` với catalog `go-test-p03-003b` đã duyệt → External Supervisor audit/release → triển khai 3B. Scanner/recovery implementation vẫn thuộc 3D; giữ `DESIGN_BLOCKER_3D_STARTUP_WIRING`.

### Acceptance tests cần có sau phê duyệt

1. Hai caller restore cùng Pair: một reservation/audit thắng; caller còn lại không gọi AO. Provisioning/dispatch/send/stop cùng Pair cũng bị chặn khi restore unresolved; reverse guard ngăn restore đi qua operation cũ.
2. Crash trước, trong, sau AO call; mất/sai response, 404/409, context cancel, AO success nhưng CAS/audit/commit fail: không reissue, không replacement, không sửa bound snapshot; 3D thấy durable intent/unknown.
3. HTTP 200 hợp lệ: cùng session ID, nonempty generation, restore mode, status và audit commit atomic; `saved_prompt`/`native` không tự chứng minh an toàn để dispatch.
4. Pre-send từng hàng §3.2: exact TaskState/`ended_at`/disposition/quarantine/audit; D12 closure atomic; identity/generation precedence; same-attempt continuation; zero `/send` khi rejection.
5. `DELIVERY_OUTCOME_UNKNOWN`: hai writer cạnh tranh, replay, late HTTP 200, audit failure, clearance Class A/B/C; dispatch resolution giữ terminal dù quarantine được giải phóng.
6. Confirmation transaction fail với và không có concurrent winner: reread chính xác durable state, không blind resend hoặc D5 overwrite `SEND_CONFIRMED`; audit chain nguyên vẹn.

## 6. Expected Benefits vs. Implementation Cost

Giữ đường restore theo ADR D3 và chặn split-brain giữa restore, send, spawn và stop. Chi phí là migration mới, nhiều guard transaction và re-audit của 3B/3C/3D; bất định AO effect sau mất response vẫn cần human, không thể được che bằng CAS local.

## 7. Decision & Rationale — dành cho External Supervisor

- **Verdict**: `PENDING`; `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL = OPEN`.
- **Cần chốt**: phương án restore/automatic eligibility, schema/token/audit, dispatch resolution, pre-send observation/quarantine, Pair-wide guard, human authority và ADR addendum scope.
- **Gate**: `TASK_P03_003B = NOT_RELEASED`; `P03_CODE = HELD_PENDING_TASK_P03_003B_CONTRACT_RELEASE`; `ACTIVE_GATE = TASK_P03_003B_CONTRACT_PLANNING`.
