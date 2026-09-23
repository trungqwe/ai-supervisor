# DRAFT TASK CONTRACT: TASK-P03-003D

> **Status:** `DRAFT_PENDING_EXTERNAL_SUPERVISOR_APPROVAL`; `TASK_P03_003D=NOT_RELEASED`; `P03_CODE=HELD_PENDING_TASK_P03_003D_CONTRACT_RELEASE`.
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
    "A startup runner is synchronous and returns a report/error to its caller. This repository has no daemon entrypoint; tests prove only its call contract, not serving-order at a real host. Host/bootstrap integration owns the future before-listener invocation and verified principal evidence.",
    "Inject SUPERVISOR_ACTIVITY_POLL_INTERVAL and SUPERVISOR_EXECUTION_DEADLINE where needed. Reject absent/nonpositive policies; never hardcode operational timeout, cadence, retries, backoff, or synthetic heartbeat. Persisted confirmation_deadline_at remains immutable and is compared with strict nanosecond Before.",
    "Poller has one owner for each task/Pair transition, an explicit cancellation and shutdown contract, bounded concurrency and no retained goroutine after shutdown. Startup sweep and poller cannot concurrently own the same transition; stale observations and CAS losers reread durable state, never replay AO effects.",
    "Preserve immutable attempt session/generation snapshot, old-disposition hold precedence, both quarantine gates, per-lineage Class A/B/C evidence and verified operator scope. A successful GET neither clears PRE_SEND_PROTOCOL_UNVERIFIED nor quarantines; administrative resolution is not physical proof.",
    "AUTOMATIC_RESTORE remains DISABLED. Fake test principal proves only library behavior; no runtime restore or linked stop enablement without verified host principal. 3D does not implement daemon/server, P04 runner, SSE, workspace fetch, WorkerReport or AO wire changes."
  ],
  "acceptance_criteria": [
    "AC-3D-01: synchronous runner classifies every unresolved restore before Pair-dependent sweeps; RESTORE_REQUESTED crash becomes RESTORE_OUTCOME_UNKNOWN with atomic lock/quarantine/audit, RESTORE_CONFIRMED retains HTTP 200 provenance, CAS loser rereads and no AO effect is reissued.",
    "AC-3D-02: only PROVISION_REQUESTED advances to PROVISION_FAILED with original Pair lock and audit in one transaction; PROVISION_CONFIRMED and historical resolved rows remain unchanged; replay and audit failure are safe.",
    "AC-3D-03: SEND_REQUESTED with null resolution becomes terminal DELIVERY_OUTCOME_UNKNOWN while DISPATCHED-to-FAILED attempt closure, UNCERTAIN_DELIVERY_CRASH, WorkerSession/attempt quarantine and audit commit atomically; SEND_CONFIRMED and already-resolved rows are not resent or rewritten.",
    "AC-3D-04: STOP_REQUESTED and STOP_CALL_SUCCEEDED with IN_FLIGHT resolution are classified by purpose, exact session/generation and persisted deadline; 404, mismatch, termination, live status and AO outage follow D11/D13, never reissue /kill; live terminal outcome uses the 3C atomic stop/D12/audit boundary.",
    "AC-3D-05: linked stop observes restore claim/owner and cannot clear quarantine before Tx D then separate D6; cleanup/maintenance do not mutate TaskState or ended_at, and old/new generation evidence remains distinct.",
    "AC-3D-06: BLOCKED with matching open current attempt deterministically becomes HUMAN_REQUIRED with ended_at, AO_BLOCKED_ESCALATED and WORKER_BLOCKED_ESCALATED audit in one D12 transaction; no BLOCKED-to-RUNNING edge or duplicate closure.",
    "AC-3D-07: reachable AO observations reconcile DISPATCHED/RUNNING on exact attempt/session/generation and approved stage/hold CAS. SEND_CONFIRMED plus idle first verifies downstream handoff availability: if available, CAS DISPATCHED-to-RUNNING with MISSED_ACTIVE_WINDOW; if unavailable, atomically fail DISPATCHED-to-FAILED with MISSED_ACTIVE_WINDOW. Both emit MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM audit; HTTP 200 alone never proves RUNNING.",
    "AC-3D-08: active/idle/waiting_input/blocked/exited/404/terminated/mismatch map to the canonical TaskState/disposition table; blocked remains open until formal escalation, and terminal outcomes close only their own attempt atomically.",
    "AC-3D-09: unavailable AO places only eligible DISPATCH_BOUND null/RECOVERY_PENDING disposition in RECOVERY_PENDING under exact CAS; PRE_SEND_PROTOCOL_UNVERIFIED or stronger hold is never downgraded by outage, timeout or subsequent GET. Pair remains unavailable.",
    "AC-3D-10: poller rejects absent injected policy, cancels on context and shuts down without leaked workers; startup ownership excludes simultaneous poller mutation on the same Pair/attempt. Concurrent writers and stale observations yield one committed transition/audit.",
    "AC-3D-11: crash at each sweep boundary, repeated startup, observation transport/protocol failure, CAS/audit fault injection and ambiguous commit preserve immutable snapshots, quarantine, resolution and audit-chain consistency; zero /send, /restore and /kill replay calls.",
    "AC-3D-12: integration harness proves runner invocation/completion/error contract before a simulated serve callback; it explicitly does not claim real daemon ordering or host principal authentication. Runtime bootstrap dependency remains open."
  ],
  "verification_requests": [
    {"id": "VR-P03-003D-RECOVERY", "profile_id": "go-test-p03-003d-proposed", "parameters": {"package": "./internal/recovery/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003D-STORE", "profile_id": "go-test-p03-003d-proposed", "parameters": {"package": "./internal/store/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003D-ALL", "profile_id": "go-test-p03-003d-proposed", "parameters": {"package": "./...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300}
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
    "A verification profile or catalog is treated as approved without independent metadata approval; semantic validation alone does not prove a P04 runner.",
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

## 3. Ma trận durable state → observation → mutation/audit

Thứ tự bắt buộc: restore preclassification, rồi provisioning, dispatch, stop, blocked, task/Pair reconciliation. Mỗi row CAS exact old state, Pair/session/generation/attempt; lỗi CAS đọc lại winner. Network observation ngoài SQLite transaction. Audit đi cùng mutation; audit failure rollback toàn bộ row. Không row nào phát lại effect từ persisted intent.

| Durable state | Observation | Mutation/audit dự kiến | Tiếp tục |
|---|---|---|---|
| `RESTORE_REQUESTED/IN_FLIGHT` | Crash; response chưa chứng minh | Tx C `RESTORE_OUTCOME_UNKNOWN`, lock/quarantine, `PAIR_RESTORE_OUTCOME_UNKNOWN`; không `/restore` | Verified operator claim; guard Pair tiếp tục |
| `RESTORE_CONFIRMED/IN_FLIGHT` | HTTP 200 đã bền vững | Giữ provenance và guard; phân loại cho quy trình Tx D/D6 | Không tự clear; operator/physical basis đúng lineage |
| `PROVISION_REQUESTED` | Bất kể AO | Tx `PROVISION_FAILED`, Pair lock, `PAIR_SESSION_PROVISION_FAILED` | Human reconciliation, không respawn |
| `SEND_REQUESTED`/resolution null | Crash hoặc query bất kỳ | Tx D5 `DELIVERY_OUTCOME_UNKNOWN`, `DISPATCHED→FAILED`, ended_at/disposition/quarantine, `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED` + state audit | Không resend; D6 riêng |
| `SEND_CONFIRMED`, attempt open | Exact identity/generation, `active` | CAS `DISPATCHED→RUNNING`, state audit | Poll cùng attempt |
| `SEND_CONFIRMED`, attempt open | Exact, `idle`/`waiting_input` sau send | Kiểm tra handoff trước CAS: khả dụng thì `DISPATCHED→RUNNING`, không khả dụng thì `DISPATCHED→FAILED` và đóng attempt; cả hai giữ `MISSED_ACTIVE_WINDOW` và audit `MISSED_ACTIVE_WINDOW_HANDOFF_DOWNSTREAM` | Không suy turn đã hoàn tất; P04 handoff |
| `DISPATCH_BOUND`, attempt open | Exact `active`/`blocked`/`exited` | Giữ `DISPATCHED`, `ended_at=NULL`; `PRE_SEND_ADMISSIBILITY_REJECTED` | Không send; observation tiếp |
| `DISPATCH_BOUND`, attempt open | AO unavailable | Exact old-disposition CAS `RECOVERY_PENDING`; audit; giữ protocol hold nếu đã có | Pair unavailable, poll theo policy injected |
| Stop `STOP_REQUESTED/IN_FLIGHT` | Exact alive / terminated / 404 / mismatch | D13 purpose-specific `STOP_REISSUE_REQUIRES_HUMAN` / unproven terminated / target absent / mismatch; `STOP_OPERATION_RESOLVED` cho hai logical resolution; live D12 atomic | Không `/kill`; quarantine theo evidence |
| Stop `STOP_CALL_SUCCEEDED/IN_FLIGHT` | Exact terminated trước deadline / alive / at-or-after deadline / 404 / mismatch | 3C `CommitStopOutcome` hoặc Store recovery CAS; `STOP_OPERATION_CONFIRMED` chỉ khi strict Before; timeout/404/mismatch audit đúng token | Linked: Tx D rồi D6; ordinary theo D11 |
| `BLOCKED` + open current attempt | Không cần AO | `AtomicAttemptClosureTransition`: `HUMAN_REQUIRED`, ended_at, `AO_BLOCKED_ESCALATED`, `WORKER_BLOCKED_ESCALATED` | Human |
| `RUNNING`/`DISPATCHED` + open attempt | AO reachable/unavailable, terminated/404/mismatch | D7/D12 exact state/hold/lineage CAS; unavailable không thay stronger hold | Tiếp tục hoặc fail-closed đúng canonical |

Các ô status phụ thuộc TaskState và dispatch stage phải đối chiếu chi tiết ADR-016 §22 trước release; ma trận này không cấp quyền suy luận transition mới. Hold `PRE_SEND_PROTOCOL_UNVERIFIED` chỉ được clear bởi quyết định compatibility có verified authority và fresh matching GET; successful GET đơn lẻ không clear. D6 kiểm tra mọi stop/restore risk trên Pair trong transaction, không chỉ evidence do caller chọn.

## 4. Startup wiring, poller và kiểm chứng dự kiến

- **Ownership:** `Run(ctx)` lấy exclusive startup ownership trước sweep; poller chỉ khởi động sau khi runner trả về thành công. Trong cùng process, poller có một owner cho mỗi Pair/attempt và mỗi tick chốt snapshot rồi CAS; tick chồng, shutdown hoặc stale writer không tạo hai transition. Sau process crash, chỉ durable CAS quyết định winner; mutex trong bộ nhớ không là bằng chứng đủ.
- **Policy:** Host inject `SUPERVISOR_ACTIVITY_POLL_INTERVAL` và `SUPERVISOR_EXECUTION_DEADLINE` khi cần; policy vắng/không hợp lệ fail-closed. Không chọn trị số, retry count hoặc backoff. `confirmation_deadline_at` đã persist không bị đổi; strict `now.Before(deadline)`.
- **Crash/evidence:** Probe trước/sau từng transaction, invalid response, 404, generation mismatch, context cancellation, audit insert failure và competing writer. Assert durable stage/resolution, TaskState, ended_at, disposition, cả hai quarantine, event_type/hash chain và zero effect calls.
- **Verification metadata:** `go-test-p03-003d-proposed` ở JSON là **đề xuất**, chưa được External Supervisor phê duyệt catalog/ProfileID, timeout/cwd/parameters. Full schema có thể kiểm tra cấu trúc; `TaskContractValidator.ValidateRaw` chỉ có giá trị khi catalog nghiêm ngặt được phê duyệt ở release. Không xem validation thư viện là bằng chứng runner P04 hoạt động.
- **Dependency `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`:** Hiện không có daemon entrypoint. Chỉ có synchronous runner và integration harness chứng minh call contract. Host/bootstrap owner phải gắn runner hoàn tất trước listener/request acceptance, fail-closed khi runner lỗi, rồi cung cấp integration evidence trên entrypoint thật; đây là tiêu chí đóng blocker sau này, không phải kết quả 3D library test. Verified host principal là dependency riêng; `AUTOMATIC_RESTORE=DISABLED`.