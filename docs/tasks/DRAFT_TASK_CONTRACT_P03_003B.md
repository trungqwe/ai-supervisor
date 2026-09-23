# DRAFT TASK CONTRACT: TASK-P03-003B

> **Contract ID**: `CONTRACT-TASK-P03-003B-01`
> **Status**: `DRAFT_PENDING_EXTERNAL_SUPERVISOR_APPROVAL` — 3B chưa được release
> **Base SHA**: `b8b0c95576d87677e8d48210d9838cc2f599752a` (3A merge commit đã tồn tại)
> **Authority**: ADR-016 đã chấp thuận, canonical specifications đã đối chiếu và implementation 3A đã được External Supervisor phê duyệt.

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
    "Reuse CLEAN ACTIVE or IDLE WorkerSession; restore terminated-but-restorable session through ResumeWorker POST /api/v1/sessions/{sessionId}/restore with the same session identity and governed generation update. Never clear quarantine because a session is missing or replaced.",
    "All P03 dispatch paths must use PrepareBoundDispatch. Never invoke legacy PrepareDispatch, which does not bind a session snapshot. Preserve 3A transaction guards and UNIQUE(attempt_id).",
    "After DISPATCH_BOUND, use GetWorkerStatus to match session ID and opaque terminal generation; only idle or waiting_input is admissible. Persist SEND_REQUESTED and audit before /send; only AO HTTP 200 and validated response permits SEND_CONFIRMED. HTTP 200 does not itself transition the task to RUNNING.",
    "Unknown /send delivery must never trigger blind resend. Atomically persist resolution, DISPATCHED -> FAILED -> HUMAN_REQUIRED through legal edges, attempt ended_at and recovery_disposition UNCERTAIN_DELIVERY_CRASH, both quarantine states, and audit; preserve Class A/B/C clearance as separately governed work.",
    "Do not implement stop coordinator (3C), poller or startup scanner (3D), schema/ADR edits, new operational timeout defaults, or P04 WorkerReport/runner behavior."
  ],
  "acceptance_criteria": [
    "AC-3B-01: atomic Pair provisioning reservation rejects existing WorkerSession, unresolved provisioning and quarantine, and leaves no operation/audit on CAS or audit failure.",
    "AC-3B-02: HTTP 201 confirmation commits WorkerSession, PROVISION_CONFIRMED and audit together; injected insert/CAS/audit failures roll back all three; crash before commit leaves only PROVISION_REQUESTED.",
    "AC-3B-03: HTTP or transport failure records PROVISION_FAILED and audit without replacing current binding; a crash before outcome commit leaves PROVISION_REQUESTED for 3D, without duplicate spawn or automatic orphan kill.",
    "AC-3B-04: existing CLEAN ACTIVE/IDLE session is reused; terminated-but-restorable session follows the existing restore route and identity/generation CAS, with no second spawn or quarantine clearance by replacement.",
    "AC-3B-05: coordinator exclusively calls PrepareBoundDispatch; state DISPATCHED, attempt snapshot, DISPATCH_BOUND and audit commit before any /send.",
    "AC-3B-06: pre-send inspection accepts only matching session/generation with idle or waiting_input; active, blocked, exited, terminated, 404 and generation mismatch are fail-closed, audit rejection, and never call /send.",
    "AC-3B-07: RecordSendRequested atomically rechecks bound lineage, CLEAN quarantine and zero unresolved provisioning, then CASes stage and audit before /send; RecordSendConfirmed CASes unresolved SEND_REQUESTED -> SEND_CONFIRMED and audit only after validated HTTP 200; duplicate or stale send/confirm is rejected.",
    "AC-3B-08: send HTTP 200 records only acceptance, leaves TaskState DISPATCHED until separately justified activity observation; no HTTP 204 success alias.",
    "AC-3B-09: unknown delivery closes and quarantines the current attempt and Pair with UNCERTAIN_DELIVERY_CRASH, legal FAILED then HUMAN_REQUIRED transitions and audit in one transaction; rollback failure preserves prior state and no resend occurs.",
    "AC-3B-10: behavior tests cover happy path, every guard, generation mismatch, T1-T6 crash boundaries, unknown delivery, duplicate provisioning/send, and transaction rollback; race and full repository suites exit zero, diff matches whitelist."
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
    "Verification profile metadata is not approved at contract release; no semantic validation or dispatch may be claimed",
    "Implementation requires an invented operational timeout default, blind resend, second spawn or unowned AO session kill"
  ]
}
```

## 2. Ownership, phụ thuộc và transaction boundaries

`internal/dispatch/coordinator.go` sở hữu điều phối đồng bộ 3B và interface nhỏ cho các phương thức AOAdapter đã tồn tại. `internal/store/provisioning_transactions.go` sở hữu reservation, confirmation, failure và restore CAS theo Pair; `internal/store/dispatch_transactions.go` sở hữu send-stage CAS, pre-send rejection, unknown-delivery quarantine và audit. `internal/store/atomic_transitions.go` chỉ được chỉnh để tái sử dụng logic chuyển trạng thái/đóng attempt trong cùng transaction nếu cần; các file test tương ứng chỉ kiểm chứng hành vi. Không chuyển trạng thái `RUNNING` trên HTTP 200; việc quan sát runtime thuộc 3D.

Phụ thuộc đã tồn tại: ADR-016 D2–D7/D12, 3A `PrepareBoundDispatch` tại `internal/store/dispatch.go`, các model và Store CRUD/CAS tại `internal/store/session_lifecycle.go`, hash-chain audit tại `internal/store/atomic_transitions.go`, `TaskState` graph tại `internal/workflow`, và AOAdapter `Client.CreateWorkerSession` (HTTP 201), `GetWorkerStatus` (HTTP 200), `ResumeWorker` (HTTP 200, `/restore`), `DispatchTaskContract` (HTTP 200) tại `internal/ao`. Các API này đủ cho wire actions 3B; draft không đề xuất sửa `internal/ao`. SOURCE_REGISTRY và REUSE_MATRIX giao AO quản lý process/worktree/harness; 3B chỉ sở hữu Pair binding và durable orchestration.

### Store API cần bổ sung trong whitelist

| API đề xuất | Transaction boundary bắt buộc |
|---|---|
| `ReservePairProvisioning` | CAS no-session/no-unresolved/CLEAN, insert `PROVISION_REQUESTED`, audit `PAIR_SESSION_PROVISION_REQUESTED`; commit trước spawn. |
| `ConfirmPairProvisioning` | CAS operation `PROVISION_REQUESTED`, insert WorkerSession, stage `PROVISION_CONFIRMED`, audit `PAIR_SESSION_PROVISIONED` trong một transaction sau HTTP 201. |
| `FailPairProvisioning` | CAS `PROVISION_REQUESTED -> PROVISION_FAILED` và audit `PAIR_SESSION_PROVISION_FAILED` cùng transaction; không tự giải phóng Pair hoặc spawn lại. |
| `CommitRestoredSession` | CAS Pair/session/status/generation/quarantine, cập nhật cùng session identity và audit sau `/restore`; không tạo WorkerSession mới. |
| `RecordPreSendRejection` | Kiểm tra operation/attempt lineage, ghi `PRE_SEND_ADMISSIBILITY_REJECTED`; nếu fail terminal, CAS task, đóng attempt và audit cùng transaction. |
| `RecordSendRequested` | Recheck current bound tuple, session/attempt `CLEAN`, zero unresolved provisioning và task/attempt lineage; CAS `DISPATCH_BOUND -> SEND_REQUESTED`, append `DISPATCH_SEND_REQUESTED` cùng transaction trước `/send`. |
| `RecordSendConfirmed` | CAS operation còn `SEND_REQUESTED` và `resolution_state IS NULL` với current attempt/task hợp lệ, set `SEND_CONFIRMED` và `confirmed_at`, append `DISPATCH_SEND_CONFIRMED` cùng transaction sau HTTP 200. |
| `FailAndEscalateUnknownDelivery` | CAS operation `SEND_REQUESTED` chưa resolve, task `DISPATCHED -> FAILED -> HUMAN_REQUIRED` theo hai cạnh hợp lệ, đóng current attempt, set disposition và hai quarantine flags, append `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED` cùng transaction. Không gọi `AtomicTerminalTransition` công khai trong transaction lồng nhau. |

Các thao tác `CreateWorkerSession`, `UpdatePairProvisioningOperationStage`, `UpdateDispatchOperationStage` và `AppendAuditEvent` hiện tại là API riêng lẻ; coordinator không được ghép các lời gọi đó và coi là atomic. `PrepareBoundDispatch` đã gộp binding/attempt/operation/audit và giữ các guard của finding 3A-R1-001.

## 3. Kiểm chứng và gate release

Test plan bắt buộc dùng fake AOAdapter để quan sát thứ tự commit và số lần gọi. Kiểm tra happy path spawn, reuse, restore, dispatch `idle`/`waiting_input`; mọi guard và generation mismatch; T1–T6 crash windows; lỗi HTTP, lỗi transport không xác định; duplicate provisioning/send; CAS conflict và audit insert failure rollback. Khởi động lại để xử lý tồn đọng là ownership 3D: 3B phải để lại state bền vững và không tự quét startup. Stop coordinator thuộc 3C.

Full-schema validation đã chạy bằng `go test ./internal/contract -run TestP03003BDraftFullSchema -count=1 -v`, exit `0`, dùng `CompileSchema` và `ParseAndValidateRaw` với `docs/schemas/task-contract.schema.json`. Harness kiểm tra tạm được xóa sau khi thu bằng chứng và không thuộc draft commit.

`verification_requests` dùng **metadata đề xuất** `go-test-p03-003b`: `CwdPolicy = worktree_root`, `MaxTimeoutSeconds = 300`, `ParameterSchema` là object với `additionalProperties = false`, required `package` và `flags`; `package` enum `./internal/dispatch/...`, `./internal/store/...`, `./...`; `flags` đúng hai phần tử duy nhất `-v`, `-race`. Profile và catalog này **chưa được External Supervisor phê duyệt**; trước release phải phê duyệt metadata và chạy `TaskContractValidator.ValidateRaw` với đúng catalog, không nới policy để test pass. Full JSON schema chỉ xác thực hình dạng contract. `timeout_seconds` là trần verification request, không phải operational timeout của AO; semantic validation cũng không chứng minh P04 runner, executable binding hoặc process isolation.

Draft này ghim vào merge commit có thật; commit chứa draft được giao riêng và không tự tham chiếu SHA. `TASK_P03_003B = NOT_RELEASED`; `P03_CODE = HELD_PENDING_TASK_P03_003B_CONTRACT_RELEASE`; `ACTIVE_GATE = TASK_P03_003B_CONTRACT_PLANNING`.
