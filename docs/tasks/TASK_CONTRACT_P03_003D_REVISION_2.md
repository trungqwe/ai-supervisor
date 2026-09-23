# TASK-P03-003D — Task Contract revision 2 đã release

> **Trạng thái:** `RELEASED`, supersedes `CONTRACT-TASK-P03-003D-01` mà không sửa artifact lịch sử. Implementation vẫn cần External Supervisor audit.
> **Căn cứ:** ADR-016 và các clarification/addendum 3D đã duyệt được giao riêng qua governance artifact; code baseline giữ `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`.

## JSON TaskContract

```json
{
  "contract_id": "CONTRACT-TASK-P03-003D-02",
  "task_id": "TASK-P03-003D",
  "revision_number": 2,
  "supersedes_contract_id": "CONTRACT-TASK-P03-003D-01",
  "phase_id": "P03",
  "objective": "Implement a synchronous restart recovery runner and cancellable lifecycle observation poller over approved AOAdapter and Store boundaries, without daemon bootstrap or effect replay.",
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
    "docs/adr/ADR-016-ADDENDUM-exclusive-host-quiescence-for-startup-recovery.md"
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
    "AUTOMATIC_RESTORE remains DISABLED. Fake test principal proves only library behavior; no runtime restore or linked stop enablement without verified host principal. 3D does not implement daemon/server, P04 runner, SSE, workspace fetch, WorkerReport or AO wire changes.",
    "Run requires a trusted scoped exclusive host quiescence provider: close shared admission for stop, send, restore, provisioning and conflicting mutation callers; drain and join every existing effect permit holder before acquire returns; keep exclusive ownership through every Run outcome; no bool quiescence assertion, timestamp, timeout, lease expiry, row reread or process-local mutex is proof across processes.",
    "Missing/error/cancelled quiescence acquisition fails closed before any intent classification. Release ownership exactly once; release after incomplete/error/cancellation never reopens admission or serve. A repeated Run reacquires and drains. Poller never performs startup-only classification or reclaims a live effect permit.",
    "The trusted host owns cross-process exclusivity for a shared DB/Pair/effect admission boundary. Fake provider verifies only library ordering. HOST_QUIESCENCE_INTEGRATION remains OPEN until actual bootstrap evidence; DESIGN_BLOCKER_3D_STARTUP_WIRING remains preserved."
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
    "AC-3D-14: missing/error/cancelled provider, cancelled drain or Run, and audit failure yield incomplete/error with no successful completion or admission reopening. Competing Runs acquire exclusively, Run/poller ownership is separated, and ownership release occurs exactly once for every acquired scope; fake host harness is not proof of real bootstrap or cross-process quiescence."
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
    "host_quiescence_integration_dependency_open"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Any required DDL, token, TaskState edge, AOAdapter change, daemon/bootstrap change or file outside allowed_scope requires Supervisor decision and revised contract.",
    "No existing Store/API boundary can preserve D13 atomic state/audit or startup/poller ownership without an unapproved architectural change.",
    "Host principal, operational policy or exact lineage is absent and an implementation would continue instead of failing closed.",
    "Verification requests exceed approved go-test-p03-003d metadata or semantic validation fails; library validation alone never proves a P04 host runner.",
    "An implementation or test would call live AO, migrate a user database or replay a persisted network effect.",
    "A trusted provider cannot prove shared effect admission closure, drain/join and cross-process exclusive ownership; fail closed before classifying persisted intents, and seek a scope/architecture decision."
  ]
}
```

## Ownership và dependency runtime

Trusted host đóng admission chung cho các effect, drain và join caller hiện hữu, rồi chuyển quyền exclusive theo scope cho `Run` đồng bộ. Release ownership đúng một lần; mở lại admission là quyết định riêng của host sau khi classification-complete và Pair guards cho phép. Poller không phân loại persisted effect intent. `HOST_QUIESCENCE_INTEGRATION=OPEN`; test thư viện dùng fake provider không chứng minh daemon ordering hay cross-process exclusion. Artifact release lịch sử, AC-3D-01..12 và whitelist gốc giữ nguyên.
