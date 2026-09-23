# DRAFT TASK CONTRACT: TASK-P03-003B

> **Contract ID**: `CONTRACT-TASK-P03-003B-01`
> **Status**: `DRAFT_PENDING_EXTERNAL_SUPERVISOR_APPROVAL` — 3B chưa được release
> **Base SHA**: `b8b0c95576d87677e8d48210d9838cc2f599752a` (3A merge commit đã tồn tại)
> **Authority**: ADR-016 đã chấp thuận, canonical specifications đã đối chiếu và implementation 3A đã được External Supervisor phê duyệt.
> **Verification metadata**: `go-test-p03-003b` đã được External Supervisor phê duyệt đúng như catalog ở §6; phê duyệt metadata không release contract hoặc production code.
> **Design blocker**: `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL` — cần quyết định trước khi release phần restore tự động.

## 1. Canonical TaskContract JSON

```json
{
  "contract_id": "CONTRACT-TASK-P03-003B-01",
  "task_id": "TASK-P03-003B",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P03",
  "objective": "Implement Pair session provisioning, reuse/restore, and the durable dispatch saga under ADR-016 using the audited 3A Store foundation, with atomic persistence and audit boundaries around each external AO effect.",
  "requirements": ["FR-005", "FR-006", "NFR-003", "NFR-005", "SEC-001"],
  "architecture_refs": [
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-8",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-9",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-10",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-11",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-12",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-13",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-18",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-22",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-23",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-24",
    "docs/04_ARCHITECTURE.md#section-2.1",
    "docs/05_DOMAIN_MODEL.md#section-3",
    "docs/06_WORKFLOW_STATE_MACHINE.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/12_UPSTREAM_INTEGRATION.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md",
    "docs/phases/P03_AO_INTEGRATION.md",
    "docs/sources/SOURCE_REGISTRY.md",
    "docs/sources/REUSE_MATRIX.md"
  ],
  "base_sha": "b8b0c95576d87677e8d48210d9838cc2f599752a",
  "allowed_scope": [
    "internal/dispatch/coordinator.go",
    "internal/dispatch/coordinator_test.go",
    "internal/store/provisioning_transactions.go",
    "internal/store/provisioning_transactions_test.go",
    "internal/store/dispatch_transactions.go",
    "internal/store/dispatch_transactions_test.go",
    "internal/store/atomic_transitions.go",
    "internal/store/atomic_transitions_test.go"
  ],
  "forbidden_scope": [
    "internal/ao/**",
    "internal/domain/**",
    "internal/poller/**",
    "internal/recovery/**",
    "internal/stop/**",
    "internal/store/migrations.go",
    "docs/adr/**",
    "docs/proposals/**",
    "docs/phases/**",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/06_WORKFLOW_STATE_MACHINE.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/12_UPSTREAM_INTEGRATION.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/18_CURRENT_STATE.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md",
    "AGENTS.md"
  ],
  "constraints": [
    "Use existing AOAdapter methods CreateWorkerSession, GetWorkerStatus, ResumeWorker and DispatchTaskContract; do not add raw AO HTTP calls or modify internal/ao without a new approval.",
    "Never call AO inside a SQLite transaction. Persist intent and audit before each non-idempotent AO effect; atomically commit each confirmed outcome with its related Store rows and audit event.",
    "Provisioning reservation must CAS the Pair guard and write PROVISION_REQUESTED plus audit in one transaction. On HTTP 201, WorkerSession insert, PROVISION_CONFIRMED CAS and PAIR_SESSION_PROVISIONED audit must share one transaction. No replacement spawn if a WorkerSession already exists.",
    "Reuse CLEAN ACTIVE or IDLE WorkerSession. Automated restore of a terminated session is held by DESIGN_BLOCKER_3B_RESTORE_PROTOCOL until External Supervisor approves a durable pre-effect reservation, single-caller exclusion and ambiguous-outcome reconciliation without inventing a saga or audit token. Never clear quarantine because a session is missing or replaced.",
    "Restore before DISPATCH_BOUND may update the current WorkerSession generation only after approved confirmation. After DISPATCH_BOUND, the attempt snapshot session_id and terminal_generation remain immutable; no restore may rewrite the bound snapshot or silently continue its send path.",
    "All P03 dispatch paths must use PrepareBoundDispatch. Never invoke legacy PrepareDispatch, which does not bind a session snapshot. Preserve 3A transaction guards and UNIQUE(attempt_id).",
    "After DISPATCH_BOUND, use GetWorkerStatus to match session ID and opaque terminal generation; only idle or waiting_input is admissible. Follow the §4 pre-send outcome table, marking unapproved cases as decision gaps. Persist SEND_REQUESTED and audit before /send; only AO HTTP 200 and validated response permits SEND_CONFIRMED. HTTP 200 does not itself transition the task to RUNNING.",
    "Before SEND_REQUESTED commits, no /send effect is permitted and a bounded same-attempt recheck uses the existing DISPATCH_BOUND operation. After SEND_REQUESTED commits, transport error, invalid response, canceled caller context or failed confirmation commit leave delivery unproven: no blind resend, new attempt or lane release; fail-closed containment and 3D startup ownership are defined in §5.",
    "Unknown /send delivery must never trigger blind resend. Atomically persist DISPATCHED -> FAILED -> HUMAN_REQUIRED through legal edges, attempt ended_at and recovery_disposition UNCERTAIN_DELIVERY_CRASH, both quarantine states, and audit; D5 requires dispatch resolution_state but its exact value remains a pre-release decision gap. Preserve Class A/B/C clearance as separately governed work.",
    "Do not implement stop coordinator (3C), poller or startup scanner (3D), schema/ADR edits, new operational timeout defaults, or P04 WorkerReport/runner behavior."
  ],
  "acceptance_criteria": [
    "AC-3B-01: atomic Pair provisioning reservation rejects existing WorkerSession, unresolved provisioning and quarantine, and leaves no operation/audit on CAS or audit failure.",
    "AC-3B-02: HTTP 201 confirmation commits WorkerSession, PROVISION_CONFIRMED and audit together; injected insert/CAS/audit failures roll back all three; crash before commit leaves only PROVISION_REQUESTED.",
    "AC-3B-03: HTTP or transport failure records PROVISION_FAILED and audit without replacing current binding; a crash before outcome commit leaves PROVISION_REQUESTED for 3D, without duplicate spawn or automatic orphan kill.",
    "AC-3B-04: existing CLEAN ACTIVE/IDLE session is reused; automated restore is not executed until DESIGN_BLOCKER_3B_RESTORE_PROTOCOL is resolved by External Supervisor. A released restore design must prove pre-effect durable evidence, one-caller exclusion, post-effect CAS/audit atomicity and no retry on uncertain outcome.",
    "AC-3B-05: coordinator exclusively calls PrepareBoundDispatch; state DISPATCHED, attempt snapshot, DISPATCH_BOUND and audit commit before any /send.",
    "AC-3B-06: pre-send inspection follows the §4 outcome table: matching idle/waiting_input may advance; active/blocked/exited remain DISPATCH_BOUND with DISPATCHED, ended_at NULL and PRE_SEND_ADMISSIBILITY_REJECTED; termination and generation mismatch fail the bound attempt, with undecided fields resolved before release. Decision-gap rows never call /send.",
    "AC-3B-07: RecordSendRequested atomically rechecks bound lineage, CLEAN quarantine and zero unresolved provisioning, then CASes stage and audit before /send; RecordSendConfirmed CASes unresolved SEND_REQUESTED -> SEND_CONFIRMED and audit only after validated HTTP 200; duplicate or stale send/confirm is rejected.",
    "AC-3B-08: send HTTP 200 records only acceptance, leaves TaskState DISPATCHED until separately justified activity observation; no HTTP 204 success alias.",
    "AC-3B-09: unknown delivery closes and quarantines the current attempt and Pair with UNCERTAIN_DELIVERY_CRASH, legal FAILED then HUMAN_REQUIRED transitions and audit in one transaction; exact dispatch resolution_state value must be approved before release; rollback failure preserves prior state and no resend occurs.",
    "AC-3B-10: behavior tests cover happy path, every guard, generation mismatch, T1-T6 crash boundaries, unknown delivery, duplicate provisioning/send, and transaction rollback; race and full repository suites exit zero, diff matches whitelist.",
    "AC-3B-11: restore tests cover two concurrent callers, missing/invalid response, HTTP 409, transport loss, canceled context, crash before/after AO effect, CAS/audit/commit failure, generation change before and after DISPATCH_BOUND, and no automatic reissue; these cases are release-blocked until the restore protocol decision is approved.",
    "AC-3B-12: pre-send table tests assert exact stage/resolution, TaskState, ended_at, disposition, quarantine, audit and same-attempt continuation for every §4 row; unresolved decision-gap rows remain fail-closed and cannot be claimed approved.",
    "AC-3B-13: send failure tests distinguish pre-intent errors from every post-SEND_REQUESTED ambiguous outcome, including transport error, response validation failure, HTTP 200 with confirmation CAS/audit failure and context cancellation; no resend or lane release is observed."
  ],
  "verification_requests": [
    {"id": "VR-P03-003B-DISPATCH", "profile_id": "go-test-p03-003b", "parameters": {"package": "./internal/dispatch/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003B-STORE", "profile_id": "go-test-p03-003b", "parameters": {"package": "./internal/store/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003B-ALL", "profile_id": "go-test-p03-003b", "parameters": {"package": "./...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300}
  ],
  "required_evidence": [
    "git_diff_base_to_implementation_and_whitelist",
    "git_diff_check_exit_code_zero",
    "behavior_test_logs_and_exit_codes",
    "race_detector_zero_warnings",
    "transaction_rollback_and_crash_boundary_results",
    "AOAdapter_method_and_HTTP_status_mapping_evidence",
    "audit_chain_integrity_verification"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Any required change outside allowed_scope, especially internal/ao or ADR/schema changes",
    "AOAdapter lacks an approved method or public response fact required for a mandatory 3B invariant",
    "An architectural contradiction in accepted ADR-016 or canonical specifications requires External Supervisor decision",
    "Atomic provisioning confirmation or unknown-delivery quarantine cannot be achieved within the approved 3B Store scope",
    "DESIGN_BLOCKER_3B_RESTORE_PROTOCOL or another decision gap remains unresolved at contract release",
    "The approved verification metadata is unavailable or ValidateRaw rejects the contract; no dispatch may be claimed",
    "Implementation requires an invented operational timeout default, blind resend, second spawn or unowned AO session kill"
  ]
}
```

## 2. Ownership, phụ thuộc và transaction boundaries

`internal/dispatch/coordinator.go` sở hữu điều phối đồng bộ 3B và interface nhỏ cho các phương thức AOAdapter đã tồn tại. `internal/store/provisioning_transactions.go` sở hữu reservation, confirmation và failure theo Pair; restore CAS chỉ được bổ sung nếu blocker ở §3 được giải quyết trong quyền hạn kiến trúc đã phê duyệt. `internal/store/dispatch_transactions.go` sở hữu send-stage CAS, pre-send rejection, unknown-delivery quarantine và audit. `internal/store/atomic_transitions.go` chỉ được chỉnh để tái sử dụng logic chuyển trạng thái/đóng attempt trong cùng transaction nếu cần; các file test tương ứng chỉ kiểm chứng hành vi. Không chuyển trạng thái `RUNNING` trên HTTP 200; việc quan sát runtime thuộc 3D.

Phụ thuộc đã tồn tại: ADR-016 D2–D7/D12, 3A `PrepareBoundDispatch` tại `internal/store/dispatch.go`, các model và Store CRUD/CAS tại `internal/store/session_lifecycle.go`, hash-chain audit tại `internal/store/atomic_transitions.go`, `TaskState` graph tại `internal/workflow`, và AOAdapter `Client.CreateWorkerSession` (HTTP 201), `GetWorkerStatus` (HTTP 200), `ResumeWorker` (HTTP 200, `/restore`), `DispatchTaskContract` (HTTP 200) tại `internal/ao`. Các API này đủ cho wire actions 3B; draft không đề xuất sửa `internal/ao`. SOURCE_REGISTRY và REUSE_MATRIX giao AO quản lý process/worktree/harness; 3B chỉ sở hữu Pair binding và durable orchestration.

### Store API cần bổ sung trong whitelist

| API đề xuất | Transaction boundary bắt buộc |
|---|---|
| `ReservePairProvisioning` | CAS no-session/no-unresolved/CLEAN, insert `PROVISION_REQUESTED`, audit `PAIR_SESSION_PROVISION_REQUESTED`; commit trước spawn. |
| `ConfirmPairProvisioning` | CAS operation `PROVISION_REQUESTED`, insert WorkerSession, stage `PROVISION_CONFIRMED`, audit `PAIR_SESSION_PROVISIONED` trong một transaction sau HTTP 201. |
| `FailPairProvisioning` | CAS `PROVISION_REQUESTED -> PROVISION_FAILED` và audit `PAIR_SESSION_PROVISION_FAILED` cùng transaction; không tự giải phóng Pair hoặc spawn lại. |
| `CommitRestoredSession` (**chưa được phép thực hiện**) | Chỉ là API có điều kiện sau quyết định §3; CAS Pair/session/status/generation/quarantine, cập nhật cùng session identity và audit sau `/restore`. Chưa có pre-effect intent/exclusion được phê duyệt nên API hậu effect đơn lẻ không đủ để gọi restore an toàn. |
| `RecordPreSendRejection` | Kiểm tra operation/attempt lineage; với `active/blocked/exited` giữ `DISPATCH_BOUND`, `DISPATCHED`, attempt mở và ghi `PRE_SEND_ADMISSIBILITY_REJECTED` cùng transaction. Với termination/generation mismatch, áp dụng terminal transition và audit theo §4. |
| `RecordSendRequested` | Recheck current bound tuple, session/attempt `CLEAN`, zero unresolved provisioning và task/attempt lineage; CAS `DISPATCH_BOUND -> SEND_REQUESTED`, append `DISPATCH_SEND_REQUESTED` cùng transaction trước `/send`. |
| `RecordSendConfirmed` | CAS operation còn `SEND_REQUESTED` và `resolution_state IS NULL` với current attempt/task hợp lệ, set `SEND_CONFIRMED` và `confirmed_at`, append `DISPATCH_SEND_CONFIRMED` cùng transaction sau HTTP 200. |
| `FailAndEscalateUnknownDelivery` | CAS operation `SEND_REQUESTED` chưa resolve, task `DISPATCHED -> FAILED -> HUMAN_REQUIRED` theo hai cạnh hợp lệ, đóng current attempt, set disposition và hai quarantine flags, append `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED` cùng transaction. D5 đòi cập nhật dispatch `resolution_state` nhưng giá trị cần chốt ở §5. Không gọi `AtomicTerminalTransition` công khai trong transaction lồng nhau. |

Các thao tác `CreateWorkerSession`, `UpdatePairProvisioningOperationStage`, `UpdateDispatchOperationStage` và `AppendAuditEvent` hiện tại là API riêng lẻ; coordinator không được ghép các lời gọi đó và coi là atomic. `PrepareBoundDispatch` đã gộp binding/attempt/operation/audit và giữ các guard của finding 3A-R1-001.

## 3. 3B-C1-001 — restore protocol và blocker

ADR-016 D3 yêu cầu dùng lại WorkerSession hiện có và `ResumeWorker` cho session terminated nhưng còn restore được; docs/12 ánh xạ vào pinned AO `POST /api/v1/sessions/{sessionId}/restore` (`restoreSession`), khác `/resume-agent`. REC-002 ở docs/14 yêu cầu reconnect, query status, restore nếu hỗ trợ và double-gated quarantine nếu restore thất bại hoặc không thể phục hồi. Pinned wire chỉ nhận session identity, không có idempotency key/correlation token cho restore. D4 chỉ có `DISPATCH_BOUND`, `SEND_REQUESTED`, `SEND_CONFIRMED`; D3 `PROVISION_REQUESTED` dành cho spawn; D6 không cho phép xóa quarantine bằng một generation/session mới. Không dùng các row/token này như một restore intent trá hình.

Điều kiện trước effect phải gồm Pair/session identity hiện hành, session `TERMINATED` nhưng có căn cứ restore, Pair và mọi attempt liên quan `CLEAN`, không có provisioning unresolved, chưa có `DISPATCH_BOUND` cần giữ snapshot generation cũ, và `GetWorkerStatus` cho thấy cùng session/generation. Phải có **bằng chứng bền vững trước effect** phân biệt restore với spawn/dispatch, đồng thời CAS giành quyền cho đúng một caller và chặn caller thứ hai kể cả sau crash. Sau HTTP 200 cùng response hợp lệ và quan sát generation mới, một transaction phải CAS đúng Pair/session/generation đã giữ quyền, cập nhật cùng WorkerSession và ghi audit; CAS, audit hoặc commit lỗi phải rollback toàn bộ local changes. Các điều kiện này là yêu cầu thiết kế, **chưa có cơ chế ghi nhận được chấp thuận để thực hiện**.

| Điểm lỗi restore | Bằng chứng và containment được phép khẳng định lúc này |
|---|---|
| Transport mất response, caller context bị hủy, response sai schema/identity hoặc HTTP 409 | Không thể kết luận effect đã xảy ra hay chưa; không tự retry `/restore`, không tạo session thay thế, không dispatch. Giữ lane/attempt bị chặn và chuyển cho reconciliation có thẩm quyền; trạng thái local không được giả là đã xác nhận. |
| AO đã restore nhưng CAS/audit/commit hậu effect thất bại | DB giữ trạng thái trước transaction; upstream có thể đã đổi generation. Không sửa bound snapshot, không retry hoặc dispatch theo generation suy đoán; cần đối chiếu upstream với intent/ownership đã được phê duyệt trước khi xác nhận. |
| Process crash trước, trong hoặc sau AO call nhưng trước confirmation commit | Local phải còn bằng chứng intent đã commit và quyền sở hữu độc nhất; nếu không có bằng chứng đó thì outcome không thể phân giải tự động. 3D startup scanner chỉ được xử lý theo protocol được phê duyệt, không suy diễn từ session hiện hữu. |
| Restore sau `DISPATCH_BOUND` hoặc generation đổi giữa binding và pre-send | Snapshot `task_attempts.session_id`/`terminal_generation` và `dispatch_operations` bất biến. Không tiếp tục `/send` trên attempt đã bind nếu generation khác; áp dụng `STALE_EXECUTION_GENERATION` theo D7. Restore trước binding, nếu được phê duyệt, phải xác nhận generation mới rồi mới gọi `PrepareBoundDispatch`. |

**DESIGN_BLOCKER_3B_RESTORE_PROTOCOL**: ADR-016, DDL §27 và registry hiện không định nghĩa restore intent, audit provenance, ownership/exclusion hoặc quy tắc reconcile ambiguous restore. Cần External Supervisor quyết định: bổ sung cơ chế đã được phê duyệt qua ADR/schema/token riêng để ghi intent và single-caller trước effect, xác nhận/reconcile hậu effect; hay tách autonomous restore khỏi contract 3B và quy định xử lý có thẩm quyền cho terminated session. Quyết định phải nêu ai được phép tiếp tục sau transport/commit/crash ambiguity và bằng chứng upstream nào đủ chứng minh restore không cần reissue. Đến khi quyết định có hiệu lực, 3B không gọi `ResumeWorker` tự động và không coi `CommitRestoredSession` là API đã được duyệt. Nếu quyết định cần thay ADR/schema hoặc file ngoài whitelist, dừng để điều chỉnh contract trước release.

## 4. 3B-C1-002 — bảng kết quả pre-send

Bảng áp dụng tại `DISPATCH_BOUND` với `TaskState=DISPATCHED`, attempt đang mở, `ended_at=NULL`, `resolution_state=NULL` trước quan sát. `resolution_state` của dispatch không có token mới được phê duyệt cho các kết quả này: giữ `NULL` ở mọi hàng, kể cả khi task terminal, cho đến khi một quyết định khác được phê duyệt. Cột “đề xuất/chưa chốt” không phải hành vi ADR đã chấp thuận. Ưu tiên kiểm tra identity, generation, `isTerminated`, rồi activity; mọi hàng bị từ chối đều cấm `/send`.

| Observation | Dispatch stage / resolution | TaskState | `ended_at` | `recovery_disposition` | Quarantine | Audit | Điều kiện tiếp tục |
|---|---|---|---|---|---|---|---|
| Cùng session/generation, `idle` hoặc `waiting_input`, `isTerminated=false` | `DISPATCH_BOUND / NULL` → `SEND_REQUESTED / NULL` chỉ khi transaction intent thành công | `DISPATCHED` | `NULL` | Không đổi | Cả hai `CLEAN` | `DISPATCH_SEND_REQUESTED` khi CAS thành công | Recheck lineage/guard trong transaction, dùng cùng attempt; sau commit mới gọi `/send`. D7, §22. |
| Cùng identity/generation, `active` | `DISPATCH_BOUND / NULL` | `DISPATCHED` | `NULL` | Không đổi | Không đổi (`CLEAN`) | `PRE_SEND_ADMISSIBILITY_REJECTED` | Chờ foreign/lingering turn chấm dứt; query lại cùng attempt, không gửi đan xen. D7, §22. |
| Cùng identity/generation, `blocked` | `DISPATCH_BOUND / NULL` | `DISPATCHED` | `NULL` | Không đổi | Không đổi (`CLEAN`) | `PRE_SEND_ADMISSIBILITY_REJECTED` | Chỉ query lại sau khi dialog được người có thẩm quyền xử lý; không nudge tự động. D7, §22. |
| Cùng identity/generation, `exited`, chưa có `isTerminated=true` | `DISPATCH_BOUND / NULL` | `DISPATCHED` | `NULL` | Không đổi | Không đổi (`CLEAN`) | `PRE_SEND_ADMISSIBILITY_REJECTED` | Không send. D7 đòi restore nhưng generation sau restore sẽ lệch bound snapshot: quyết định tiếp tục/đóng attempt còn thiếu, phụ thuộc blocker §3. |
| Cùng identity/generation, `isTerminated=true` | `DISPATCH_BOUND / NULL` | `FAILED` | Set `now` | `WORKER_TERMINATION_UNKNOWN` (**đề xuất**) | §22: không có hành động quarantine; giữ giá trị cũ, không tự clear | Audit terminal transition; event cụ thể cần chốt (**khoảng trống**) | Không tiếp tục attempt; §22 xác nhận failed/closure nhưng chưa gán disposition/event. Không suy diễn crash hay stop. |
| `terminalGeneration` khác bound snapshot | `DISPATCH_BOUND / NULL` | `FAILED` | Set `now` theo D12 (**cần chốt mapping pre-send**) | `STALE_EXECUTION_GENERATION` theo D7 | Chính sách impose quarantine chưa được §22 chốt; giữ fail-closed, không clear | `PRE_SEND_ADMISSIBILITY_REJECTED` theo D7; audit terminal cùng transaction | Không tiếp tục attempt. Cần chốt quarantine/terminal audit trước release; không sửa snapshot. |
| Response `sessionId` khác bound session dù query đúng ID | `DISPATCH_BOUND / NULL` (**đề xuất giữ**) | `DISPATCHED` (**đề xuất giữ**) | `NULL` | Không đổi | Không đổi, lane bị chặn bởi coordinator | Audit identity rejection cần chốt (**khoảng trống**) | Không send; cần quyết định terminal/quarantine/audit vì D7 chỉ định generation mismatch. |
| `GetWorkerStatus` HTTP 404 | `DISPATCH_BOUND / NULL` (**đề xuất giữ**) | `DISPATCHED` (**đề xuất giữ**) | `NULL` | Không đổi | Không tự clear; lane bị chặn | Audit 404 cần chốt (**khoảng trống**) | Không send hoặc tự thay session. D11/D13 có `SESSION_ABSENT` cho stop/restart; áp dụng vào pre-send cần quyết định riêng. |
| Query transport error, timeout hoặc response protocol/validation failure | `DISPATCH_BOUND / NULL` (**đề xuất giữ**) | `DISPATCHED` (**đề xuất giữ**) | `NULL` | Không đổi | Không tự clear; lane bị chặn | Audit lỗi query cần chốt (**khoảng trống**) | Không send; chỉ query lại cùng attempt sau khi có status hợp lệ, theo chính sách retry/ownership cần chốt. |

Với attempt còn mở, coordinator nạp lại **đúng** `dispatch_operations.attempt_id` và TaskAttempt/current_attempt đang bind, đối chiếu Pair/task/contract/session/generation rồi query lại; không gọi `PrepareBoundDispatch` lần nữa, không tạo attempt/operation mới. Chỉ một transaction `RecordSendRequested` có thể thắng CAS `DISPATCH_BOUND -> SEND_REQUESTED`. Nếu đã terminal hoặc generation đổi, đường cùng attempt không còn hợp lệ. Trước release cần External Supervisor chốt các ô đánh dấu khoảng trống/đề xuất; mặc định thực thi fail-closed, không ghi token hoặc transition chưa được duyệt.

## 5. Failure boundaries của `/send`

| Boundary / observation | Durable state và transaction containment | Ownership phục hồi |
|---|---|---|
| Guard, query hoặc DB/audit lỗi **trước khi** `RecordSendRequested` commit | `DISPATCH_BOUND / NULL`, `DISPATCHED`, attempt mở, `ended_at=NULL`; transaction lỗi rollback cả stage/audit. Chưa gọi `/send`; dùng lại đúng operation/attempt sau admissibility recheck. | Coordinator 3B trong phiên hiện tại; T4 startup recheck thuộc 3D. |
| `SEND_REQUESTED` và `DISPATCH_SEND_REQUESTED` đã commit; transport error, caller context canceled, process crash trong call hoặc response validation failure (kể cả HTTP 200 thiếu/sai identity) | Intent bền vững nhưng delivery chưa được xác nhận; không được đảo về `DISPATCH_BOUND`, blind resend hoặc giải phóng lane. Transaction D5 ghi `DISPATCHED -> FAILED -> HUMAN_REQUIRED`, đóng attempt, `UNCERTAIN_DELIVERY_CRASH`, cả hai `QUARANTINED`, audit `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED`; nếu transaction này thất bại thì vẫn còn `SEND_REQUESTED` chưa resolve. | Coordinator 3B xử lý lỗi đã thấy; 3D scanner xử lý tồn đọng T5 sau crash/rollback, clearance theo D5–D6/3C. |
| AO trả HTTP 200 response hợp lệ nhưng `RecordSendConfirmed` CAS/audit/commit thất bại | AO có thể đã nhận write; DB vẫn `SEND_REQUESTED`, `confirmed_at=NULL`, `TaskState=DISPATCHED` vì confirmation transaction rollback. Không gọi `/send` lại; dùng cùng unknown-delivery containment như trên, không suy diễn `RUNNING`. | Coordinator 3B; nếu containment rollback/crash thì 3D. |
| HTTP 200 hợp lệ và confirmation commit thành công | `SEND_CONFIRMED`, `confirmed_at` và `DISPATCH_SEND_CONFIRMED` atomic; task vẫn `DISPATCHED`, attempt mở. | 3D poller quan sát activity riêng; 3C sở hữu stop coordinator. |

`SEND_REQUESTED` là ranh giới bằng chứng: từ sau commit, kể cả hủy context trước khi client thật sự phát byte, local state không chứng minh `/send` chưa tới AO. Chỉ HTTP 200 **và** response hợp lệ cho phép xác nhận; HTTP 204 không phải success alias. ADR-016 D5 yêu cầu cập nhật `dispatch_operations.resolution_state` khi unknown delivery nhưng registry không định nghĩa giá trị cho cột này; External Supervisor phải chốt chính xác giá trị/token trước release, không lấy `task_attempts.recovery_disposition` làm enum của cột khác. D5–D6 áp dụng Class A/B/C khi clearance; session vắng mặt/generation mới không tự giải phóng quarantine.

## 6. Kiểm chứng và gate release

Test plan dùng fake AOAdapter kiểm tra thứ tự commit và số lần gọi: spawn/reuse happy path; guard no-session/no-unresolved/CLEAN và rollback insert/CAS/audit; T1–T3 crash và duplicate spawn; `PrepareBoundDispatch` độc quyền; toàn bộ hàng §4, đặc biệt `active/blocked/exited` vẫn `DISPATCHED`/attempt mở, mismatched identity/generation, termination, 404 và query failure; same-attempt continuation và hai caller tranh CAS. Test restore ở AC-3B-11 chỉ trở thành executable acceptance sau quyết định §3; trước đó chứng minh không gọi `/restore` tự động, không double-call và fail-closed trong các tình huống ambiguous. Test T4–T6 cùng mọi hàng §5: pre-intent rollback không gọi `/send`; post-intent transport/validation/context/confirmation failure và containment rollback không resend, không release lane. Khởi động lại và poller thuộc 3D, stop coordinator thuộc 3C.

`verification_requests` dùng **metadata đã được External Supervisor phê duyệt** `go-test-p03-003b`: `CwdPolicy=worktree_root`, `MaxTimeoutSeconds=300`, `ParameterSchema` object với `additionalProperties=false`, required `package` và `flags`; `package` enum `./internal/dispatch/...`, `./internal/store/...`, `./...`; `flags` đúng hai phần tử duy nhất `-v`, `-race`. Full JSON schema và `TaskContractValidator.ValidateRaw` phải chạy với đúng catalog này; negative cases `timeout_seconds=301`, `flags=["-v","-exec"]`, `package=./internal/ao/...`, `cwd=../escape` đều bị từ chối. Không nới catalog để hợp thức hóa draft. `timeout_seconds` là trần verification request, không phải operational timeout AO; semantic validation không chứng minh P04 runner, executable binding hoặc process isolation.

Kiểm chứng draft bằng harness Go ngoài worktree nạp `docs/schemas/task-contract.schema.json`, chạy `CompileSchema`/`ParseAndValidateRaw` rồi `TaskContractValidator.ValidateRaw` với đúng catalog trên: full-schema và semantic validation PASS; cả bốn negative cases bị từ chối; `go test -overlay <temporary-overlay> ./internal/contract -run '^TestP03003BDraftValidation$' -count=1 -v` exit `0`. Harness/overlay tạm đã xóa, không nằm trong whitelist hoặc commit; đây là policy validation, không phải thực thi verification_requests qua P04 runner.

Draft ghim vào merge commit có thật; commit chứa draft được giao riêng và không tự tham chiếu SHA. `TASK_P03_003B = NOT_RELEASED`; `P03_CODE = HELD_PENDING_TASK_P03_003B_CONTRACT_RELEASE`; `ACTIVE_GATE = TASK_P03_003B_CONTRACT_PLANNING`. Bảo lưu `DESIGN_BLOCKER_3D_STARTUP_WIRING`.
