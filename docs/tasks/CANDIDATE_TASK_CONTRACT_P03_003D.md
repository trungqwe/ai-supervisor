# CANDIDATE TASK CONTRACT: TASK-P03-003D

> **Status:** `CANDIDATE_PENDING_RELEASE_AUDIT`; `TASK_P03_003D=NOT_RELEASED`; `P03_CODE=HELD_PENDING_TASK_P03_003D_CONTRACT_RELEASE`.
> **Proposed code base SHA:** `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527` (merge 3C đã tồn tại). Artifact contract sau release được giao riêng; không nhúng SHA tự tham chiếu.
> **Authority:** ADR-016 D5/D7/D10/D12/D13 và §§19, 22, 25; ADR-016 addendum §§3–7; hai clarification stop; canonical `docs/06`, `docs/12`, `docs/14`; parent decomposition §3.4. Draft không cấp quyền viết Go hay đổi schema.

## 1. JSON TaskContract dự thảo

```json
{
  "contract_id": "CONTRACT-TASK-P03-003D-01",
  "task_id": "TASK-P03-003D",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P03",
  "objective": "Implement a synchronous restart recovery runner and cancellable lifecycle observation poller over approved AOAdapter and Store boundaries, without daemon bootstrap or effect replay.",
  "requirements": ["FR-005", "FR-006", "NFR-003", "NFR-005", "SEC-001"],
  "architecture_refs": [
    "docs/tasks/DRAFT_TASK_CONTRACT_P03_003.md#section-3.4",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-19",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-22",
    "docs/adr/ADR-016-ADDENDUM-restore-and-pre-send-semantics.md#section-5",
    "docs/adr/ADR-016-CLARIFICATION-administrative-stop-risk-acceptance.md",
    "docs/adr/ADR-016-CLARIFICATION-stop-operation-resolved-audit.md",
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
  "base_sha": "583e700eb125a08cc6bd7d63b6b27a6f3d4cc527",
  "allowed_scope": [
    "internal/recovery/scanner.go",
    "internal/recovery/scanner_test.go",
    "internal/recovery/poller.go",
    "internal/recovery/poller_test.go",
    "internal/recovery/integration_test.go",
    "internal/store/recovery.go",
    "internal/store/recovery_test.go",
    "internal/store/recovery_transactions.go",
    "internal/store/recovery_transactions_test.go",
    "internal/store/atomic_transitions.go",
    "internal/store/atomic_transitions_test.go",
    "internal/store/dispatch_transactions.go",
    "internal/store/dispatch_transactions_test.go",
    "internal/store/provisioning_transactions.go",
    "internal/store/provisioning_transactions_test.go",
    "internal/store/restore_transactions.go",
    "internal/store/restore_transactions_test.go",
    "internal/store/stop_transactions.go",
    "internal/store/stop_transactions_test.go"
  ],
  "forbidden_scope": [
    "internal/ao/**",
    "internal/dispatch/**",
    "internal/stop/**",
    "internal/domain/**",
    "internal/workflow/**",
    "cmd/**",
    "docs/adr/**",
    "docs/schemas/**",
    "docs/phases/**",
    "AGENTS.md",
    "docs/18_CURRENT_STATE.md"
  ],
  "constraints": [
    "Run restore preclassification before the five ADR-016 D13 sweeps; keep unresolved restore Pair ownership and quarantine active. Do not call /restore, /send or /kill from persisted intent, including after crash or commit ambiguity.",
    "Use only approved AOAdapter CheckHealth, CheckReadiness and GetWorkerStatus for observation; perform network calls outside SQLite transactions. Reuse 3B/3C Store transitions and guards; add exact CAS/audit transactions where restart recovery cannot be atomic with current APIs.",
    "A startup runner is synchronous and reports classification complete with AO pending when local CAS/audit classification commits but AO remains unavailable with RECOVERY_PENDING and Pair locks; fatal validation/DB/audit error or cancellation reports incomplete and never emits STARTUP_RECOVERY_SWEEP_COMPLETED. Completion means classification finished, not execution risk cleared. A library harness does not prove daemon ordering.",
    "Inject SUPERVISOR_ACTIVITY_POLL_INTERVAL and SUPERVISOR_EXECUTION_DEADLINE where needed. Reject absent/nonpositive policies; never hardcode operational timeout, cadence, retries, backoff, or synthetic heartbeat. Persisted confirmation_deadline_at remains immutable and is compared with strict nanosecond Before.",
    "Within one host instance, Run excludes poller startup until classification completes; across runners/pollers, Store exact row CAS/audit decides each mutation winner and losers reread. No in-memory mutex is treated as cross-process proof. Poller has explicit cancellation, shutdown and bounded concurrency; repeated Run is state-idempotent and never replays AO effects.",
    "Preserve immutable attempt session/generation snapshot, exact old-disposition CAS and hold precedence, both quarantine gates, per-lineage Class A/B/C evidence and verified operator scope. Pre-send PRE_SEND_PROTOCOL_UNVERIFIED needs verified compatibility decision plus fresh exact idle/waiting_input GET; RECOVERY_PENDING from outage needs its approved fresh-exact-observation CAS. Successful GET alone never clears quarantine or protocol hold; administrative resolution is not physical proof.",
    "AUTOMATIC_RESTORE remains DISABLED. Fake test principal proves only library behavior; no runtime restore or linked stop enablement without verified host principal. 3D does not implement daemon/server, P04 runner, SSE, workspace fetch, WorkerReport or AO wire changes."
  ],
  "acceptance_criteria": [
    "AC-3D-01: synchronous runner classifies every unresolved restore before Pair-dependent sweeps; RESTORE_REQUESTED crash becomes RESTORE_OUTCOME_UNKNOWN with atomic lock/quarantine/audit, RESTORE_CONFIRMED retains HTTP 200 provenance, CAS loser rereads and no AO effect is reissued.",
    "AC-3D-02: only PROVISION_REQUESTED advances to PROVISION_FAILED with original Pair lock and audit in one transaction; PROVISION_CONFIRMED and historical resolved rows remain unchanged; replay and audit failure are safe.",
    "AC-3D-03: reuse RecordUnknownDelivery for SEND_REQUESTED with null resolution: one transaction commits terminal DELIVERY_OUTCOME_UNKNOWN with stage retained, DISPATCHED-to-FAILED-to-HUMAN_REQUIRED, exact current-attempt ended_at/UNCERTAIN_DELIVERY_CRASH, WorkerSession/attempt quarantine, both TASK_STATE_TRANSITION audits and UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED. Audit/CAS failure rolls back all; replay is read-only and never resends or changes resolution.",
    "AC-3D-04: STOP_REQUESTED and STOP_CALL_SUCCEEDED with IN_FLIGHT resolution are classified by purpose, exact session/generation and persisted deadline; 404, mismatch, termination, live status and AO outage follow D11/D13, never reissue /kill; live terminal outcome uses the 3C atomic stop/D12/audit boundary.",
    "AC-3D-05: linked stop observes restore claim/owner and cannot clear quarantine before Tx D then separate D6; cleanup/maintenance do not mutate TaskState or ended_at, and old/new generation evidence remains distinct.",
    "AC-3D-06: BLOCKED with matching open current attempt deterministically becomes HUMAN_REQUIRED with ended_at, AO_BLOCKED_ESCALATED and WORKER_BLOCKED_ESCALATED audit in one D12 transaction; no BLOCKED-to-RUNNING edge or duplicate closure.",
    "AC-3D-07: exact SEND_CONFIRMED observation maps active to DISPATCHED-to-RUNNING; waiting_input to DISPATCHED-to-RUNNING with AO_WAITING_INPUT_OBSERVED telemetry/disposition (never MISSED_ACTIVE_WINDOW); blocked to DISPATCHED-to-RUNNING with AO_BLOCKED_DECISION while attempt stays open; idle verifies P04 handoff before MISSED_ACTIVE_WINDOW RUNNING or fail-closed FAILED, with MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM audit. HTTP 200 alone never proves RUNNING.",
    "AC-3D-08: TaskState x dispatch-stage x observation tests cover active, idle, waiting_input, blocked, exited, isTerminated=true, 404, wrong session identity and generation mismatch for DISPATCH_BOUND, SEND_REQUESTED, SEND_CONFIRMED and RUNNING. Exited without isTerminated keeps attempt open; blocked observation never implies failure; terminal branches use D12 exact-attempt closure and approved disposition/audit; no new TaskState edge.",
    "AC-3D-09: AO outage on open DISPATCH_BOUND, DISPATCHED/SEND_CONFIRMED or RUNNING persists RECOVERY_PENDING only from null or itself under exact Pair/task/attempt/stage/old-disposition CAS plus outage audit; Pair stays unavailable. PRE_SEND_PROTOCOL_UNVERIFIED and stronger diagnostic dispositions remain unchanged. Fresh exact GET plus approved CAS resolves outage hold; successful GET alone never clears protocol hold or quarantine.",
    "AC-3D-10: fatal local validation/DB/audit error or cancellation prevents successful completion report/audit; AO unavailable after persisted RECOVERY_PENDING and Pair lock permits a classification-complete/pending-AO report and poller startup without dispatch on locked Pairs. Concurrent runners/pollers and repeated Run produce one winner per row under Store CAS/audit, with completion audit only for each fully classified invocation; cancellation/shutdown leave no owned workers.",
    "AC-3D-11: crash at each sweep boundary, repeated startup, observation transport/protocol failure, CAS/audit fault injection and ambiguous commit preserve immutable snapshots, quarantine, resolution and audit-chain consistency; zero /send, /restore and /kill replay calls.",
    "AC-3D-12: integration harness distinguishes complete, classified-pending-AO and incomplete/fatal Run results; STARTUP_RECOVERY_SWEEP_COMPLETED means classification only. Simulated serve callback is suppressed on fatal/incomplete results; harness does not prove real daemon ordering or host principal authentication, and startup-wiring blocker remains open."
  ],
  "verification_requests": [
    {"id": "VR-P03-003D-RECOVERY", "profile_id": "go-test-p03-003d", "parameters": {"package": "./internal/recovery/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003D-STORE", "profile_id": "go-test-p03-003d", "parameters": {"package": "./internal/store/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003D-ALL", "profile_id": "go-test-p03-003d", "parameters": {"package": "./...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300}
  ],
  "required_evidence": [
    "base_to_head_diff_and_exact_whitelist",
    "git_diff_check_exit_code_zero",
    "uncached_behavior_and_race_logs_with_exit_codes",
    "crash_replay_and_zero_effect_call_counts",
    "CAS_audit_fault_injection_and_lineage_assertions",
    "poller_cancellation_shutdown_and_ownership_evidence",
    "synchronous_runner_harness_and_host_bootstrap_dependency_status"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Any required DDL, token, TaskState edge, AOAdapter change, daemon/bootstrap change or file outside allowed_scope requires Supervisor decision and revised contract.",
    "No existing Store/API boundary can preserve D13 atomic state/audit or startup/poller ownership without an unapproved architectural change.",
    "Host principal, operational policy or exact lineage is absent and an implementation would continue instead of failing closed.",
    "Verification requests exceed approved go-test-p03-003d metadata or semantic validation fails; library validation alone never proves a P04 host runner.",
    "An implementation or test would call live AO, migrate a user database or replay a persisted network effect."
  ]
}
```

## 2. API và ownership tại code base

| Có sẵn sau merge 3C | Còn thiếu cho 3D; chủ sở hữu |
|---|---|
| AOAdapter `CheckHealth`, `CheckReadiness`, `GetWorkerStatus` | `internal/recovery` chỉ đọc status; không sửa AOAdapter hoặc dùng raw `/sessions` listing. |
| Store `ClassifyRestartRecovery` trong `internal/store/recovery.go` | Hiện chỉ phân loại `DISPATCHED`, không mutate. Mở rộng query cho restore, provisioning, dispatch, stop, blocked và in-flight tasks; Store sở hữu CAS/audit theo exact lineage. |
| Store `FailPairProvisioning`, `RecordUnknownDelivery`, `RecordPreSendHold`, `AtomicAttemptClosureTransition` | Reuse khi exact expected state/lineage đủ; bổ sung transaction API trong `recovery_transactions.go` cho mọi D13 boundary hiện chưa nguyên tử. Không ghép các public transaction độc lập để giả định atomic. |
| Store `RecordPairRestoreUnknown`, `GetPairRestoreOperation`, `CommitStopOutcome`, `ClearQuarantineWithEvidence` | Reuse stage/resolution, D11/D12 và D6 guards. Scanner chỉ phân loại, không gọi operator resolution hoặc AO effect; linked recovery phải giữ Tx D/D6 riêng. |
| Dispatch/stop coordinators 3B/3C | Không sửa hoặc gọi effect methods từ persisted intents. Poller dùng observation và Store CAS, stop reconciliation single observation chỉ khi 3C API chứng minh không tạo effect. |
| Không có `cmd/` hoặc `main.go` trong tree tại base | 3D cung cấp `Run(ctx)` đồng bộ và harness; host/bootstrap tương lai chịu trách nhiệm gọi xong trước listen/serve, cung cấp verified principal. |

## 3. Ma trận TaskState × dispatch stage × observation

Thứ tự runner: phân loại restore trước năm bước D13 (provisioning, dispatch, stop, blocked, in-flight task/Pair). Mọi observation AO nằm ngoài SQLite transaction. Mỗi mutation cần exact Pair/task/current open attempt/session/generation, stage, TaskState và old disposition CAS trong cùng transaction với audit; CAS loser đọc lại winner. Một operation risk chưa resolved tiếp tục khóa Pair. Ưu tiên observation: protocol và `sessionId` → generation opaque equality → `isTerminated` → activity; riêng `SEND_REQUESTED`/resolution NULL đi D5 không phụ thuộc GET. Không phát lại `/send`, `/restore`, `/kill` từ intent.

| TaskState / dispatch stage hoặc operation | Observation | Mutation bền vững / audit | Tiếp tục và rủi ro |
|---|---|---|---|
| Pair `RESTORE_REQUESTED/IN_FLIGHT` | Crash; outcome không chứng minh | Tx C `RESTORE_OUTCOME_UNKNOWN`, lock/quarantine, `PAIR_RESTORE_OUTCOME_UNKNOWN` | Operator claim; không `/restore`; phân loại trước sweep khác |
| Pair `RESTORE_CONFIRMED/IN_FLIGHT` | HTTP 200 đã persist | Giữ provenance và Pair guard; phân loại Tx D/D6 | Không tự clear hoặc gọi effect |
| Pair `PROVISION_REQUESTED` | Bất kể AO | Tx `PROVISION_FAILED`, Pair lock, `PAIR_SESSION_PROVISION_FAILED` | Human reconciliation; không spawn lại |
| `DISPATCHED/SEND_REQUESTED`, resolution NULL | Mọi GET, kể cả 404/mismatch | Reuse `RecordUnknownDelivery`: **một transaction** CAS `DELIVERY_OUTCOME_UNKNOWN` terminal, giữ `SEND_REQUESTED`; `DISPATCHED→FAILED→HUMAN_REQUIRED`, exact attempt `ended_at`, `UNCERTAIN_DELIVERY_CRASH`, WorkerSession và attempt `QUARANTINED`, hai `TASK_STATE_TRANSITION` và `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED` | Replay read-only, không resend; D6 clearance không đổi resolution |
| `DISPATCHED/SEND_REQUESTED`, resolution `DELIVERY_OUTCOME_UNKNOWN` | Replay hoặc late HTTP 200 | Không mutation; giữ winner/audit | Không resend, không `SEND_CONFIRMED` |
| `DISPATCHED/DISPATCH_BOUND` | Exact `idle` hoặc `waiting_input` | Giữ `DISPATCHED`, `ended_at=NULL`; fresh admission riêng mới được CAS `SEND_REQUESTED` | Không phát `/send` trong scanner từ persisted binding |
| `DISPATCHED/DISPATCH_BOUND` | Exact `active`, `blocked` hoặc `exited` | Giữ `DISPATCHED`, open attempt; `PRE_SEND_ADMISSIBILITY_REJECTED` audit | Không send; `exited` không tự chứng minh terminated |
| `DISPATCHED/DISPATCH_BOUND` | `isTerminated=true` | D12 `DISPATCHED→FAILED`, close exact attempt, `WORKER_TERMINATION_UNKNOWN`, `TASK_STATE_TRANSITION` | Không suy nguyên nhân chết hoặc tự clear quarantine |
| `DISPATCHED/DISPATCH_BOUND` | HTTP 404 | D12 `FAILED`, `SESSION_ABSENT`, double quarantine/audit theo addendum §6 | Class B/C; không thay session để bỏ guard |
| `DISPATCHED/DISPATCH_BOUND` | Session identity sai hoặc generation mismatch | D12 `FAILED`, `PRE_SEND_IDENTITY_MISMATCH` hoặc `STALE_EXECUTION_GENERATION`, double quarantine/audit | Không sửa bound snapshot |
| `DISPATCHED/DISPATCH_BOUND` | Query unavailable | Exact old-disposition `NULL/RECOVERY_PENDING→RECOVERY_PENDING`, outage audit, open attempt; protocol hold mạnh hơn giữ nguyên | Pair unavailable; không send |
| `DISPATCHED/SEND_CONFIRMED` | Exact `active` | CAS `DISPATCHED→RUNNING`, `TASK_STATE_TRANSITION`, attempt open | Poll cùng lineage; HTTP 200 đơn lẻ không đủ |
| `DISPATCHED/SEND_CONFIRMED` | Exact `waiting_input` | CAS `DISPATCHED→RUNNING`, `AO_WAITING_INPUT_OBSERVED` diagnostic telemetry/disposition; state audit | Không `MISSED_ACTIVE_WINDOW`, không tự gửi thêm input |
| `DISPATCHED/SEND_CONFIRMED` | Exact `blocked` | CAS `DISPATCHED→RUNNING`, `AO_BLOCKED_DECISION`, state/observation audit, attempt open | D10: không tự `FAILED`; formal escalation riêng `RUNNING→BLOCKED→HUMAN_REQUIRED` |
| `DISPATCHED/SEND_CONFIRMED` | Exact `idle`, chưa từng thấy active | Trước CAS kiểm tra P04 handoff: khả dụng → `RUNNING`, không khả dụng → D12 `FAILED`/close attempt; cả hai `MISSED_ACTIVE_WINDOW` và `MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM` audit | Không suy REPORT_READY; handoff thuộc P04 |
| `DISPATCHED/SEND_CONFIRMED` | Exact `exited`, `isTerminated=false` | Giữ `DISPATCHED`, `ended_at=NULL`, ghi observation; không suy session chết hoặc completion | Process-exit reconciliation/operator; không `/send`/`resume-agent` tự động |
| `DISPATCHED/SEND_CONFIRMED` | `isTerminated=true` | D12 `DISPATCHED→FAILED`, exact closure/audit, `WORKER_TERMINATION_UNKNOWN` | Không suy nguyên nhân chết; không tự clear quarantine |
| `DISPATCHED/SEND_CONFIRMED` | HTTP 404 | D12 `DISPATCHED→FAILED`, exact closure/audit, `SESSION_ABSENT` | 404 không phải physical proof; Class B cần acceptance |
| `DISPATCHED/SEND_CONFIRMED` | Generation mismatch | D12 `DISPATCHED→FAILED`, exact closure/audit, `STALE_EXECUTION_GENERATION` | Không sửa snapshot; Pair/attempt quarantine |
| `DISPATCHED/SEND_CONFIRMED` | Response `sessionId` sai hoặc protocol invalid | Bác observation vì không thuộc bound attempt; không áp activity/termination, giữ `DISPATCHED`/open attempt và Pair lock, ghi diagnostic telemetry; không tạo audit event_type mới | Không dùng `PRE_SEND_IDENTITY_MISMATCH` sai stage; cần operator reconciliation, không tự send/clear |
| `DISPATCHED/SEND_CONFIRMED` | AO unavailable | Exact old-disposition `NULL/RECOVERY_PENDING→RECOVERY_PENDING` + outage audit, giữ `DISPATCHED`, open attempt | Pair unavailable; không downgrade diagnostic hold |
| `RUNNING/SEND_CONFIRMED` | Exact `active` | Giữ `RUNNING`, open attempt; observation | Poll tiếp |
| `RUNNING/SEND_CONFIRMED` | Exact `idle` | Giữ `RUNNING`, open attempt; P04 report handoff | AO idle không là REPORT_READY |
| `RUNNING/SEND_CONFIRMED` | Exact `waiting_input` | Giữ `RUNNING`; `AO_WAITING_INPUT_OBSERVED` diagnostic telemetry/disposition | Không synthetic input; execution deadline vẫn áp dụng |
| `RUNNING/SEND_CONFIRMED` | Exact `blocked` | Giữ `RUNNING`, `AO_BLOCKED_DECISION`, open attempt, `WORKER_BLOCKED_ON_DECISION` audit | Formal D10 escalation riêng; không suy blocked là failure |
| `RUNNING/SEND_CONFIRMED` | Exact `exited`, `isTerminated=false` | Giữ `RUNNING`, `ended_at=NULL`; observation/process-exit reconciliation | Không suy crash, death hoặc `WORKER_STOPPED` |
| `RUNNING/SEND_CONFIRMED` | `isTerminated=true` | D12 `RUNNING→FAILED`, exact closure/audit, `WORKER_TERMINATION_UNKNOWN` | Không suy crash; không tự clear quarantine |
| `RUNNING/SEND_CONFIRMED` | HTTP 404 | D12 `RUNNING→FAILED`, exact closure/audit, `SESSION_ABSENT` | Không suy physical stop; Class B cần acceptance |
| `RUNNING/SEND_CONFIRMED` | Generation mismatch | D12 `RUNNING→FAILED`, exact closure/audit, `STALE_EXECUTION_GENERATION` | Không sửa snapshot; giữ quarantine |
| `RUNNING/SEND_CONFIRMED` | Response `sessionId` sai hoặc protocol invalid | Bác observation vì không thuộc bound attempt; giữ `RUNNING`/open attempt và Pair lock, ghi diagnostic telemetry; không tạo audit event_type mới | Operator reconciliation, không suy session chết hoặc clear |
| `RUNNING/SEND_CONFIRMED` | AO unavailable | Exact old-disposition `NULL/RECOVERY_PENDING→RECOVERY_PENDING` + outage audit; giữ `RUNNING`, open attempt | Pair unavailable; không thay stronger disposition |
| `BLOCKED/SEND_CONFIRMED`, current attempt open | Startup, không cần AO | `AtomicAttemptClosureTransition`: `BLOCKED→HUMAN_REQUIRED`, `ended_at`, `AO_BLOCKED_ESCALATED`, `WORKER_BLOCKED_ESCALATED` cùng transaction | Không `BLOCKED→RUNNING`, không duplicate closure |
| Stop `STOP_REQUESTED/IN_FLIGHT` | Exact alive / terminated / 404 / mismatch | D13/3C purpose-specific `STOP_REISSUE_REQUIRES_HUMAN` / unproven terminated / target absent / mismatch; `STOP_OPERATION_RESOLVED` cho hai logical outcome; live D12 atomic | Không `/kill`, giữ quarantine trừ clearance đúng D6 |
| Stop `STOP_CALL_SUCCEEDED/IN_FLIGHT` | Exact terminated trước deadline / alive / at-or-after deadline / 404 / mismatch | 3C `CommitStopOutcome` hoặc Store CAS; `STOP_OPERATION_CONFIRMED` chỉ khi strict Before; audit token đúng từng outcome | Linked: stop evidence → Tx D → D6; không reissue |

**Outage/hold:** Bổ sung Store API trong `internal/store/recovery_transactions.go` để ghi `RECOVERY_PENDING` cho `DISPATCH_BOUND`, `DISPATCHED/SEND_CONFIRMED` và `RUNNING` trên chính current attempt: CAS old disposition `NULL` hoặc `RECOVERY_PENDING`, exact stage/state/lineage và audit trong một transaction. `PRE_SEND_PROTOCOL_UNVERIFIED`, `AO_BLOCKED_DECISION`, `MISSED_ACTIVE_WINDOW` hoặc disposition chẩn đoán mạnh hơn không bị ghi đè. Pair lane bị khóa bởi open attempt/hold, không phụ thuộc local mutex. Pre-send `RECOVERY_PENDING` chỉ clear qua fresh exact `idle/waiting_input` GET và `ResolvePreSendHold` CAS/audit; protocol hold đòi thêm verified compatibility decision. Post-send outage hold chỉ được giải bằng fresh exact observation và Store CAS/audit tương ứng: `RECOVERY_PENDING→NULL` khi observation không đặt diagnostic mới, hoặc `RECOVERY_PENDING→AO_WAITING_INPUT_OBSERVED`/`AO_BLOCKED_DECISION` theo đúng activity. Không gọi `ResolvePreSendHold` sai stage. Response `sessionId` sai/protocol invalid không được coi là outage thuần để ghi đè hold; Store giữ Pair lock và diagnostic telemetry trước human reconciliation. GET thành công không tự clear quarantine. Audit failure rollback hold và state; CAS loser reread.

**Kiểm chứng ma trận:** Test từng hàng với exact/foreign session, generation, old disposition, 404, `isTerminated`, five activity states, timeout/protocol error, P04 handoff khả dụng/không khả dụng; assert TaskState, dispatch stage/resolution, ended_at, disposition, cả hai quarantine, actual audit `event_type` và zero effect call. `SEND_REQUESTED` probe phải thấy cả hai cạnh TaskState trong cùng `RecordUnknownDelivery`; fault injection bất kỳ audit/CAS rollback toàn bộ. Repeated scan không sửa terminal resolution.
## 4. Runner/poller contract, crash boundaries và verification

| `Run(ctx)` outcome | Report, audit và poller admission | Dispatch admission |
|---|---|---|
| Tất cả phân loại local đã commit, AO reachable | Báo hoàn tất phân loại; `STARTUP_RECOVERY_SWEEP_COMPLETED` sau các transaction; poller có thể chạy | Vẫn xét Pair/attempt guard từng mutation; completion không giải execution risk |
| AO unavailable nhưng từng attempt liên quan đã persist `RECOVERY_PENDING`/giữ stronger hold và Pair lock | Báo phân loại hoàn tất với AO còn chờ; emit `STARTUP_RECOVERY_SWEEP_COMPLETED` chỉ sau khi mọi local row được xử lý; poller có thể chạy theo injected cadence để GET lại | Pair có pending hold/open attempt vẫn cấm dispatch; không nhận GET thành công là clearance |
| Policy thiếu, local validation/DB/CAS reread/audit fail, hoặc `ctx` hủy giữa sweep | Báo incomplete/error, không emit `STARTUP_RECOVERY_SWEEP_COMPLETED`, không cho host bắt đầu poller/serve theo call contract | Fail closed; các row đã commit giữ nguyên và Run sau tiếp tục idempotent |

`STARTUP_RECOVERY_SWEEP_STARTED` và `STARTUP_RECOVERY_SWEEP_COMPLETED` là audit cho invocation; completed chứng minh **đã phân loại mọi candidate thuộc snapshot/sweep**, không chứng minh mọi execution risk đã giải hoặc AO reachable. Nếu ghi completion audit thất bại, runner trả error/incomplete. Khi CAS cạnh tranh, chỉ xem bước hoàn tất sau khi reread winner và chứng minh state phù hợp; không giả định commit lỗi nghĩa là chưa có mutation. Không có network call trong transaction.

**Ownership:** Host library gọi `Run(ctx)` đồng bộ; trong một instance, chỉ cho poller khởi động khi runner báo phân loại hoàn tất (kể cả AO-pending đã persist). Nhiều runner hoặc runner/poller process khác có thể lấy cùng snapshot; transaction Store CAS exact Pair/task/current attempt/operation stage/resolution/old disposition là quyền mutation thực, loser reread. Poller serialize tick theo Pair, không chạy tick chồng trên cùng lineage; cancellation chặn tick mới, đợi tick đang chạy dừng trước shutdown, không giữ goroutine. Repeated `Run` có thể có invocation audit riêng nhưng không lặp state transition hoặc AO effect. Không có durable leader/lease mới trong schema hiện hành; nếu triển khai yêu cầu exclusion mạnh hơn CAS per row, dừng theo stop condition và xin quyết định thiết kế.

**Policy và fault tests:** Host inject `SUPERVISOR_ACTIVITY_POLL_INTERVAL` và `SUPERVISOR_EXECUTION_DEADLINE` khi cần; thiếu/không hợp lệ fail-closed, không chọn giá trị, retry count hoặc backoff. Persisted `confirmation_deadline_at` bất biến; so sánh `now.Before(deadline)` nanosecond. Fault injection trước/sau từng transaction, invalid response, 404, generation mismatch, cancellation sau một phần sweep, hai `Run` song song, Run và poller đồng thời, tick trùng, AO outage rồi fresh GET. Assert state/audit chain, snapshot, holds, Pair lock, không duplicate transition và zero effect replay. Test harness chỉ chứng minh simulated serve callback sau `Run` hoàn tất; không chứng minh daemon ordering thật.

**Verification metadata được duyệt:** `ProfileID=go-test-p03-003d`, `CwdPolicy=worktree_root`, `MaxTimeoutSeconds=300`, `ParameterSchema={type:object, additionalProperties:false, required:[package,flags], package enum:[./internal/recovery/...,./internal/store/...,./...], flags enum duy nhất ["-v","-race"] theo đúng thứ tự}`. Đây là policy validation cho 3D; chưa có bằng chứng host runner P04 triển khai. Full schema và `TaskContractValidator.ValidateRaw` với catalog này phải PASS trước release; negative timeout 301, `-exec`, package ngoài enum, cwd `../escape` phải FAIL.

**Dependency `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`:** Repository chưa có daemon entrypoint. Draft chỉ giao synchronous runner và integration harness. Host/bootstrap owner phải gắn runner hoàn tất trước listener/request acceptance, fail-closed trên incomplete/error, và cung cấp integration evidence trên entrypoint thật để đóng blocker. Verified host principal là dependency riêng; fake principal test không chứng minh host authentication. `AUTOMATIC_RESTORE=DISABLED`; 3D chưa release.