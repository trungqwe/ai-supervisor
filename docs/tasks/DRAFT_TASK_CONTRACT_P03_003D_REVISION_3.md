# TASK-P03-003D — Draft Task Contract revision 3: execution deadline

> **Trạng thái:** `DRAFT_NOT_RELEASED`, chờ External Supervisor audit thiết kế và scope. Revision này dự kiến supersede `CONTRACT-TASK-P03-003D-02`; artifact revision 1/2 vẫn bất biến. Không cấp quyền migration hoặc Go code cho deadline.
> **Code base_sha:** `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`. Khi được duyệt, tiếp tục branch `codex/p03-003d` từ implementation chưa merge `00f64d0b26db2749eadd21b7307c98faa6b6280d`; không reset hoặc chép governance vào diff. Specification/release artifact sẽ được giao riêng.
> **Design dependency:** DRAFT ADR send-confirmation execution budget chưa được duyệt; `3D-R1-005=OPEN`. `HOST_QUIESCENCE_INTEGRATION=OPEN`, host principal dependency và `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`; `AUTOMATIC_RESTORE=DISABLED`.

## JSON TaskContract dự kiến

```json
{
  "contract_id": "CONTRACT-TASK-P03-003D-03",
  "task_id": "TASK-P03-003D",
  "revision_number": 3,
  "supersedes_contract_id": "CONTRACT-TASK-P03-003D-02",
  "phase_id": "P03",
  "objective": "Complete TASK-P03-003D recovery and add a persistent send-confirmation-based execution budget with a post-startup, host-admitted timeout monitor that reuses the 3C stop coordinator.",
  "requirements": [
    "FR-005",
    "FR-006",
    "NFR-003",
    "NFR-005",
    "SEC-001"
  ],
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
    "docs/sources/REUSE_MATRIX.md",
    "docs/adr/ADR-016-CLARIFICATION-3d-recovery-audit-events.md",
    "docs/adr/ADR-016-ADDENDUM-exclusive-host-quiescence-for-startup-recovery.md",
    "docs/proposals/PROPOSAL-P03-004-send-confirmation-execution-budget.md",
    "docs/adr/DRAFT-ADR-016-ADDENDUM-send-confirmation-execution-budget.md"
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
    "internal/store/stop_transactions_test.go",
    "internal/recovery/timeout_monitor.go",
    "internal/recovery/timeout_monitor_test.go",
    "internal/domain/session_lifecycle.go",
    "internal/domain/session_lifecycle_test.go",
    "internal/dispatch/coordinator.go",
    "internal/dispatch/coordinator_test.go",
    "internal/stop/coordinator.go",
    "internal/stop/coordinator_test.go",
    "internal/store/migrations.go",
    "internal/store/migrations_v5_test.go",
    "internal/store/session_lifecycle.go",
    "internal/store/session_lifecycle_test.go"
  ],
  "forbidden_scope": [
    "internal/ao/**",
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
    "Startup and lifecycle observation use only approved AOAdapter CheckHealth, CheckReadiness and GetWorkerStatus; network calls stay outside SQLite transactions. The new runtime timeout monitor delegates one-use StopWorker effect solely to guarded stop.Coordinator.StartTimeout after trusted shared host admission; ordinary Start rejects TIMEOUT cause. Reuse the 3C reservation/effect flow and Store guards; add exact CAS/audit transactions where needed.",
    "A startup runner is synchronous and reports classification complete with AO pending when local CAS/audit classification commits but AO remains unavailable with RECOVERY_PENDING and Pair locks; fatal validation/DB/audit error or cancellation reports incomplete and never emits STARTUP_RECOVERY_SWEEP_COMPLETED. Completion means classification finished, not execution risk cleared. A library harness does not prove daemon ordering.",
    "Inject SUPERVISOR_ACTIVITY_POLL_INTERVAL and SUPERVISOR_EXECUTION_DEADLINE where needed. Reject absent/nonpositive policies; never hardcode operational timeout, cadence, retries, backoff, or synthetic heartbeat. Persisted confirmation_deadline_at remains immutable and is compared with strict nanosecond Before.",
    "Within one host instance, Run excludes poller startup until classification completes; across runners/pollers, Store exact row CAS/audit decides each mutation winner and losers reread. No in-memory mutex is treated as cross-process proof. Poller has explicit cancellation, shutdown and bounded concurrency; repeated Run is state-idempotent and never replays AO effects.",
    "Preserve immutable attempt session/generation snapshot, exact old-disposition CAS and hold precedence, both quarantine gates, per-lineage Class A/B/C evidence and verified operator scope. Pre-send PRE_SEND_PROTOCOL_UNVERIFIED needs verified compatibility decision plus fresh exact idle/waiting_input GET; RECOVERY_PENDING from outage needs its approved fresh-exact-observation CAS. Successful GET alone never clears quarantine or protocol hold; administrative resolution is not physical proof.",
    "AUTOMATIC_RESTORE remains DISABLED. Fake test principal proves only library behavior; no runtime restore or linked stop enablement without verified host principal. 3D does not implement daemon/server, P04 runner, SSE, workspace fetch, WorkerReport or AO wire changes.",
    "Run requires a trusted scoped exclusive host quiescence provider: close shared admission for stop, send, restore, provisioning and conflicting mutation callers; drain and join every existing effect permit holder before acquire returns; keep exclusive ownership through every Run outcome; no bool quiescence assertion, timestamp, timeout, lease expiry, row reread or process-local mutex is proof across processes.",
    "Missing/error/cancelled quiescence acquisition fails closed before any intent classification. Release ownership exactly once; release after incomplete/error/cancellation never reopens admission or serve. A repeated Run reacquires and drains. Poller never performs startup-only classification or reclaims a live effect permit.",
    "The trusted host owns cross-process exclusivity for a shared DB/Pair/effect admission boundary. Fake provider verifies only library ordering. HOST_QUIESCENCE_INTEGRATION remains OPEN until actual bootstrap evidence; DESIGN_BLOCKER_3D_STARTUP_WIRING remains preserved.",
    "PENDING DESIGN APPROVAL: the new v5 DDL, two budget binding-basis literals, legacy audit event, TIMEOUT stop cause and expanded dispatch/stop/domain scope cannot be implemented from this draft. Preserve released revision 2 and audited remediation history.",
    "Use dispatch_operations.confirmed_at as the send-confirmation-based budget origin, explicitly not actual execution start. Capture positive duration_ns and policy_ref before SEND_REQUESTED; atomically insert immutable attempt budget, SEND_CONFIRMED stage/confirmed_at and DISPATCH_SEND_CONFIRMED audit. Never derive legacy history from now, started_at or current policy.",
    "Startup Run performs classification only and never creates a timeout stop. Post-startup TimeoutMonitor requires trusted shared host AcquireEffect(ctx, Pair, TIMEOUT_MONITOR) scoped permit held through stop.Coordinator.StartTimeout; missing provider, policy or valid durable budget disables the path. Host quiescence drains timeout permits with all effect callers across processes; no local mutex is cross-process proof.",
    "Only exact current open RUNNING/SEND_CONFIRMED attempt with matching Pair/contract/session/generation, a valid immutable budget and no stop/risk may reserve a new RUNNING_ATTEMPT_STOP at now >= execution_deadline_at. DISPATCHED, blocked escalation, P04 handoff, 404, termination and mismatch retain their approved precedence. No TaskState edge or AO wire change.",
    "Persist initiating_failure_reason=TIMEOUT on the stop row in the reservation transaction. Guarded stop.Coordinator.StartTimeout reuses 3C one-use effect ownership; persisted STOP_REQUESTED never permits replay. D11 result/D12 closure/quarantine/audit remain atomic, with failure_reason=TIMEOUT and recovery_disposition determined by physical/logical stop outcome. Execution deadline does not replace confirmation_deadline_at."
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
    "AC-3D-12: integration harness distinguishes complete, classified-pending-AO and incomplete/fatal Run results; STARTUP_RECOVERY_SWEEP_COMPLETED means classification only. Simulated serve callback is suppressed on fatal/incomplete results; harness does not prove real daemon ordering or host principal authentication, and startup-wiring blocker remains open.",
    "AC-3D-13: deterministic barriers show caller A committed an intent but paused before effect; Run B does not GET or classify until the trusted provider closes admission and joins A; caller C is denied while admission is closed. After join, B may classify exact durable state without effect replay.",
    "AC-3D-14: missing/error/cancelled provider, cancelled drain or Run, and audit failure yield incomplete/error with no successful completion or admission reopening. Competing Runs acquire exclusively, Run/poller ownership is separated, and ownership release occurs exactly once for every acquired scope; fake host harness is not proof of real bootstrap or cross-process quiescence.",
    "AC-3D-15: v5 migration fresh/v1/v2/v3/v4-to-v5 and reopen preserve historical data/audit chain, rollback on DDL/index failure, immutable budget row and stop cause, exact one live stop per attempt, and fail-closed invalid timestamp, policy, lineage or overflow; legacy confirmed rows are not silently backfilled.",
    "AC-3D-16: before AO send, a validated injected duration/policy_ref snapshot is captured; RecordSendConfirmed commits budget origin=confirmed_at, deadline, policy provenance, SEND_CONFIRMED and audit in one transaction. CAS/audit/commit failure leaves no partial budget/confirmation and uses existing D5 containment without resend; restart/policy change never recalculates the persisted deadline.",
    "AC-3D-17: startup Run never creates timeout stop; runtime monitor requires trusted shared admission and exact current open RUNNING/SEND_CONFIRMED attempt. Before deadline no stop; at/after deadline one competing winner reserves a TIMEOUT-cause RUNNING_ATTEMPT_STOP through guarded 3C StartTimeout, and only that caller may effect once. Ordinary Start rejects TIMEOUT bypass. Existing stop, DISPATCHED, blocked/P04/terminal/mismatch/risk paths retain precedence; crash/replay cannot issue another kill.",
    "AC-3D-18: legacy active SEND_CONFIRMED missing budget keeps startup incomplete/admission closed. Only verified operator scope plus historical confirmed_at and policy evidence atomically inserts LEGACY_OPERATOR_VERIFIED budget and EXECUTION_BUDGET_LEGACY_BOUND audit; absent/changed evidence, CAS loser or audit failure leaves it blocked. No now/started_at/current-policy fabrication.",
    "AC-3D-19: timeout cause survives crash and is emitted as TASK_STATE_TRANSITION failure_reason=TIMEOUT when 3C D11/D12 closes a live attempt; recovery_disposition, stop resolution, quarantine and audit retain exact D11 outcome in the same transaction. Confirmation deadline remains independent and strict; stale writer/audit failure rolls back all."
  ],
  "verification_requests": [
    {
      "id": "VR-P03-003D-RECOVERY",
      "profile_id": "go-test-p03-003d",
      "parameters": {
        "package": "./internal/recovery/...",
        "flags": [
          "-v",
          "-race"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    },
    {
      "id": "VR-P03-003D-STORE",
      "profile_id": "go-test-p03-003d",
      "parameters": {
        "package": "./internal/store/...",
        "flags": [
          "-v",
          "-race"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    },
    {
      "id": "VR-P03-003D-ALL",
      "profile_id": "go-test-p03-003d",
      "parameters": {
        "package": "./...",
        "flags": [
          "-v",
          "-race"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    }
  ],
  "required_evidence": [
    "base_to_head_diff_and_exact_whitelist",
    "git_diff_check_exit_code_zero",
    "uncached_behavior_and_race_logs_with_exit_codes",
    "crash_replay_and_zero_effect_call_counts",
    "CAS_audit_fault_injection_and_lineage_assertions",
    "poller_cancellation_shutdown_and_ownership_evidence",
    "synchronous_runner_harness_and_host_bootstrap_dependency_status",
    "deterministic_quiescence_barrier_and_admission_tests",
    "host_quiescence_integration_dependency_open",
    "v5_migration_and_legacy_evidence_authority_tests",
    "atomic_send_confirmation_budget_and_D5_rollback_tests",
    "before_equal_after_deadline_restart_policy_and_waiting_input_tests",
    "competing_timeout_monitor_host_admission_and_stop_effect_count_tests",
    "timeout_cause_D11_outcome_audit_and_rollback_tests"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Any DDL, token or event beyond the externally approved deadline addendum, any TaskState edge, AOAdapter or daemon/bootstrap change, or any file outside allowed_scope requires a new Supervisor decision and revised contract.",
    "No existing Store/API boundary can preserve D13 atomic state/audit or startup/poller ownership without an unapproved architectural change.",
    "Host principal, operational policy or exact lineage is absent and an implementation would continue instead of failing closed.",
    "Verification requests exceed approved go-test-p03-003d metadata or semantic validation fails; library validation alone never proves a P04 host runner.",
    "An implementation or test would call live AO, migrate a user database or replay a persisted network effect.",
    "A trusted provider cannot prove shared effect admission closure, drain/join and cross-process exclusive ownership; fail closed before classifying persisted intents, and seek a scope/architecture decision.",
    "The proposed deadline ADR or revision 3 is not externally approved and released; no deadline migration or production implementation may start.",
    "Legacy historical policy evidence or trusted operator authority is absent; do not synthesize a deadline or open admission.",
    "Timeout monitor requires AOAdapter, TaskState, schema/token, daemon/bootstrap or file changes beyond approved revision 3; stop for a new decision."
  ]
}
```

## Ranh giới implementation và verification dự kiến

Revision giữ nguyên AC-3D-01..14, ba verification requests và 19 path whitelist của revision 2; chỉ thêm AC-3D-15..19 và các path cụ thể cho migration/domain/dispatch/stop/timeout monitor. `internal/recovery/timeout_monitor.go` là runtime component riêng, không chạy trong startup `Run`; guarded `stop.Coordinator.StartTimeout` lấy host shared admission permit rồi dùng luồng reservation/effect 3C. `internal/dispatch/coordinator.go` capture policy trước send; `internal/store/dispatch_transactions.go` ghi budget cùng HTTP 200 confirmation. `internal/stop/coordinator.go` tiếp tục là đường duy nhất cấp one-use `/kill` permit; Store reservation và D11 giữ exact lineage, timeout cause và audit atomic. `internal/store/session_lifecycle.go` generic dispatch-stage API phải chịu guard v5, không lách budget.

Ba VR hiện tại vẫn dùng metadata `go-test-p03-003d` đã duyệt (recovery, store, toàn repo; `cwd=worktree_root`, timeout tối đa 300 giây, flags đúng `[-v,-race]`). `./...` bao gồm dispatch/stop. Khi release, yêu cầu thêm regression tập trung không cache cho `./internal/dispatch/...` và `./internal/stop/...`; chưa tuyên bố profile host runner P04 đã triển khai. Schema/ValidateRaw cho draft chỉ chứng minh cấu trúc và policy catalog, không cấp quyền code.

Sau design approval cần reconcile canonical registry và docs theo draft ADR, rồi tạo release artifact revision 3 riêng. Lượt thiết kế này không thay accepted ADR, schema hay implementation `00f64d0`.
