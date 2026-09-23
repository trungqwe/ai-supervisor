# CANDIDATE TASK CONTRACT: TASK-P03-003B

> Status: CANDIDATE_PENDING_EXTERNAL_AUDIT — NOT_RELEASED.
> Code base_sha: b8b0c95576d87677e8d48210d9838cc2f599752a; artifact/spec commit giao riêng, không tự tham chiếu.
> Authority: ADR-016 accepted + approved ADR-016 addendum at audited SHA 3807343f06f99c0e11460c74522a2efb3be19491. Implementation chưa được chứng minh.
> Verification policy: go-test-p03-003b metadata đã duyệt; policy validation không chứng minh P04 runner.

## 1. Immutable TaskContract JSON

```json
{
  "contract_id": "CONTRACT-TASK-P03-003B-01",
  "task_id": "TASK-P03-003B",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P03",
  "objective": "Implement Pair session provisioning, reuse/restore, and the durable dispatch saga under ADR-016 using the audited 3A Store foundation, with atomic persistence and audit boundaries around each external AO effect.",
  "requirements": [
    "FR-005",
    "FR-006",
    "NFR-003",
    "NFR-005",
    "SEC-001"
  ],
  "architecture_refs": [
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-8",
    "docs/adr/ADR-016-ADDENDUM-restore-and-pre-send-semantics.md",
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
    "internal/domain/session_lifecycle.go",
    "internal/domain/session_lifecycle_test.go",
    "internal/store/migrations.go",
    "internal/store/migrations_v3_test.go",
    "internal/store/store_test.go",
    "internal/store/audit_test.go",
    "internal/store/migrations_v4_test.go",
    "internal/store/dispatch.go",
    "internal/store/dispatch_test.go",
    "internal/store/session_lifecycle.go",
    "internal/store/session_lifecycle_test.go",
    "internal/store/session_lifecycle_remediation_test.go",
    "internal/store/atomic_transitions.go",
    "internal/store/atomic_transitions_test.go",
    "internal/store/recovery.go",
    "internal/store/recovery_test.go",
    "internal/store/provisioning_transactions.go",
    "internal/store/provisioning_transactions_test.go",
    "internal/store/dispatch_transactions.go",
    "internal/store/dispatch_transactions_test.go",
    "internal/store/restore_transactions.go",
    "internal/store/restore_transactions_test.go"
  ],
  "forbidden_scope": [
    "internal/ao/**",
    "internal/poller/**",
    "internal/stop/**",
    "internal/workflow/**",
    "docs/adr/**",
    "docs/proposals/**",
    "docs/phases/**",
    "docs/schemas/**",
    "AGENTS.md",
    "docs/18_CURRENT_STATE.md"
  ],
  "constraints": [
    "Use approved ADR-016 addendum v4 DDL and registry exactly; migration v3 remains intact. Never alter immutable attempt snapshot or 3A stop lineage.",
    "In audit_test.go, update only expected CURRENT schema version after Open/reopen for v4; preserve audit-chain, historical migration data and rollback assertions. Do not mechanically change source-version or rollback literals 1/2/3 to 4.",
    "Only verified host principal may enable operator-authorized restore or linked stop; 3B implements/test fail-closed trusted interface. AUTOMATIC_RESTORE=DISABLED.",
    "Tx A consumes one-shot authorization and commits RESTORE_REQUESTED/audit before /restore; no AO call in SQLite transaction; one Pair owner. Lost/invalid response, 404/409, canceled context or commit ambiguity never triggers retry.",
    "Tx B confirms valid HTTP 200, WorkerSession generation and restore operation/audit atomically. GET generation is not causal proof. Restore after DISPATCH_BOUND never rewrites bound snapshot.",
    "ReservePairRestore rejects any open attempt, SEND_REQUESTED with NULL resolution even closed, inconsistent current_attempt, unresolved Pair operations or quarantine; historical clean closed DISPATCH_BOUND/SEND_CONFIRMED is allowed.",
    "Provisioning reservation and HTTP 201 confirmation each use atomic Pair/operation/WorkerSession/audit transactions; no replacement spawn or duplicate side effect.",
    "All P03 dispatch paths use PrepareBoundDispatch. Existing PrepareDispatch and generic Store mutation APIs must check RESTORE_UNRESOLVED in the same transaction or be closed to P03.",
    "Pre-send exact session/generation and idle/waiting_input only; protocol hold outranks RECOVERY_PENDING. RecordSendRequested requires NULL hold and commits intent/audit before /send.",
    "Only validated AO HTTP 200 commits SEND_CONFIRMED. HTTP 200 does not transition TaskState to RUNNING; 204 is not success.",
    "Unknown delivery sets terminal dispatch resolution DELIVERY_OUTCOME_UNKNOWN, D12 FAILED then HUMAN_REQUIRED, ended_at, UNCERTAIN_DELIVERY_CRASH, double quarantine and audit atomically; never blind resend or lane release.",
    "Linked stop Store lineage preserves 3A checks: old-generation QUARANTINE_CLEANUP attaches exact closed attempt; approved new-generation PAIR_MAINTENANCE has no attempt IDs, restore link and operator provenance. 3C owns effect/clearance.",
    "3B persists restore resolution/lineage evidence and guards, but 3C owns stop coordinator and D6 clearance orchestration; 3D owns startup scanner/poller. No operational timeout default, P04 runner or AO wire change."
  ],
  "acceptance_criteria": [
    "AC-3B-01: provisioning reservation CASes no-session/no-unresolved/CLEAN and writes PROVISION_REQUESTED/audit atomically; duplicate caller, CAS or audit failure leaves no intent.",
    "AC-3B-02: HTTP 201 confirmation atomically inserts WorkerSession and PROVISION_CONFIRMED/audit; insert/CAS/audit rollback leaves PROVISION_REQUESTED for 3D.",
    "AC-3B-03: failure records PROVISION_FAILED/audit without replacement spawn; crash intent remains durable and no automatic orphan kill.",
    "AC-3B-04: v4 migration fresh/v3-to-v4/rerun preserves data, FK/unique/CHECK/index/trigger; one-shot authorization replay and wrong principal/scope fail closed.",
    "AC-3B-05: two restore callers and cross-type Pair mutations serialize; any open attempt, SEND_REQUESTED/NULL closed attempt or current pointer inconsistency rejects restore, while clean historical closed dispatch does not.",
    "AC-3B-06: Tx A precedes AO call; HTTP 200 Tx B updates same WorkerSession and operation/audit atomically. Lost/invalid response, HTTP 409, context cancellation, crash and CAS/audit/commit ambiguity never retry or clear Pair lock.",
    "AC-3B-07: only PrepareBoundDispatch allocates bound attempt; old legacy and generic Store APIs cannot bypass restore/provision/quarantine guard. After binding snapshot is immutable.",
    "AC-3B-08: pre-send exact identity/generation/idle-or-waiting only; active/blocked/exited keep DISPATCH_BOUND, DISPATCHED, ended_at NULL and rejection audit; D7/§22 terminal cases use D12 closure.",
    "AC-3B-09: protocol failure then timeout then valid GET retains PRE_SEND_PROTOCOL_UNVERIFIED; only verified operator decision plus fresh GET CAS clears. RecordSendRequested rejects any unresolved hold; competing writer cannot downgrade.",
    "AC-3B-10: SEND_REQUESTED intent/audit commits before effect; validated HTTP 200 alone permits SEND_CONFIRMED, leaving task DISPATCHED. Duplicate send/confirm and stale HTTP 200 are rejected.",
    "AC-3B-11: unknown delivery sets DELIVERY_OUTCOME_UNKNOWN terminal with D12 closure, double quarantine/audit; stale confirmation cannot overwrite; rollback or winner reread preserves durable result.",
    "AC-3B-12: linked stop persistence rejects old-attempt session/generation mismatch and accepts only exact old-generation cleanup or approved no-attempt new-generation PAIR_MAINTENANCE with restore link/authority provenance. Mixed evidence cannot clear old attempt or WorkerSession in 3B.",
    "AC-3B-13: fake AO behavior tests cover all crash boundaries, no duplicate provisioning/restore/send, stale CAS, audit rollback and fail-closed host principal boundary; test principal is not host authentication evidence."
  ],
  "verification_requests": [
    {
      "id": "VR-P03-003B-DISPATCH",
      "profile_id": "go-test-p03-003b",
      "parameters": {
        "package": "./internal/dispatch/...",
        "flags": [
          "-v",
          "-race"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    },
    {
      "id": "VR-P03-003B-STORE",
      "profile_id": "go-test-p03-003b",
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
      "id": "VR-P03-003B-ALL",
      "profile_id": "go-test-p03-003b",
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
    "Any required file or AO capability outside allowed_scope; any change to accepted ADR, approved addendum or TaskState graph.",
    "Host principal integration is absent and implementation would enable restore or linked stop instead of fail-closed interface.",
    "An existing Store mutation path can bypass RESTORE_UNRESOLVED, quarantine, provisioning or bound lineage and cannot be closed inside allowed_scope.",
    "Migration v4 or atomic restore/provision/dispatch/hold transaction cannot satisfy approved addendum without new schema or token.",
    "Any invented operational timeout, automatic restore, blind retry, replacement spawn, or unowned AO effect.",
    "Validation with approved go-test-p03-003b catalog fails, or implementation scope expands into 3C/3D."
  ]
}
```

## 2. Ownership và transaction boundaries

3B sở hữu v4 migration/domain models, provisioning/restore/dispatch Store transactions và coordinator đồng bộ. Tx A reserve Pair, consume exact one-shot authorization, insert RESTORE_REQUESTED/audit rồi commit trước /restore. Tx B sau validated HTTP 200 CAS exact operation/version/old generation, cập nhật cùng WorkerSession và audit atomic. Lost/invalid response, HTTP 404/409, context cancellation hoặc commit ambiguity ghi/giữ unresolved và quarantine; không retry. 3B chỉ triển khai trusted interface fail-closed: host/bootstrap integration sở hữu verified principal và bằng chứng enable; test principal giả không chứng minh host authentication.

3B phải sửa/khóa API hiện hữu trong whitelist: PrepareDispatch (không cho P03 bypass PrepareBoundDispatch), PrepareBoundDispatch, CreateWorkerSession, UpdateWorkerSessionStatusAndQuarantine, Create/UpdatePairProvisioningOperation, Create/UpdateDispatchOperation; Create/UpdateStopOperation chỉ thêm restore ownership guard và linked lineage persistence, không gọi /kill. AtomicTerminalTransition/AtomicAttemptClosureTransition và recovery Store path phải giữ D12/CAS/audit. Không mở rộng sang internal/ao: CreateWorkerSession, GetWorkerStatus, ResumeWorker, DispatchTaskContract đã có theo pinned AO. No network trong SQLite transaction.

3C sở hữu stop coordinator, linked /kill, operator risk/clearance orchestration D6 theo từng lineage. 3D sở hữu startup scanner/poller, xử lý intent còn treo mà không reissue. Existing Store paths phải fail closed ngay trong 3B trong thời gian 3C/3D chưa release. Nếu host principal chưa được tích hợp, restore/linked stop vẫn disabled; các Store API không được tự coi actor string là authority.

## 3. Acceptance coverage và evidence

AC-3B-01..13 trong JSON là acceptance dự kiến, chưa phải PASS implementation. Fake AO test chứng minh thứ tự durable intent/effect/confirmation, concurrent Pair callers, old/new generation, hold precedence, stale confirmation và mọi rollback CAS/audit. Fresh/v3→v4/rerun migration, FK/CHECK/unique/trigger và dữ liệu cũ phải được probe. Cần log lệnh/exit code, diff whitelist, audit chain, số AO calls và state sau crash. Không dùng race detector như bằng chứng đầy đủ về connection leak.

`internal/store/audit_test.go` được phép cập nhật **chỉ** expected CURRENT schema version sau `Open`/reopen khi migration v4 được triển khai; phải giữ nguyên kiểm chứng audit hash chain, rollback migration lỗi và dữ liệu lịch sử. Các literal `1`/`2`/`3` biểu thị source schema, fixture hoặc expected rollback không được đổi máy móc thành `4`. Rà assertion version trong `internal/store/*test.go` cho thấy các file cần sửa expected CURRENT version là `store_test.go`, `migrations_v3_test.go` và `audit_test.go`, đều thuộc `allowed_scope`; lượt này chưa sửa Go test.

go-test-p03-003b: CwdPolicy=worktree_root; MaxTimeoutSeconds=300; ParameterSchema object, additionalProperties=false, required package/flags; package enum ./internal/dispatch/..., ./internal/store/..., ./...; flags đúng hai phần tử duy nhất -v và -race. Negative cases timeout_seconds=301, flag -exec, package ./internal/ao/..., cwd ../escape phải bị ValidateRaw từ chối. timeout_seconds là policy verification, không phải AO operational timeout; semantic validation không chứng minh P04 runner hoặc process isolation.

## 4. Gate

TASK_P03_003B/3C/3D=NOT_RELEASED; P03_CODE=HELD_PENDING_TASK_P03_003B_CONTRACT_RELEASE; ACTIVE_GATE=TASK_P03_003B_CONTRACT_PLANNING. DESIGN_BLOCKER_3B_RESTORE_PROTOCOL=CLOSED_BY_APPROVED_ADDENDUM nhưng runtime restore vẫn disabled khi thiếu verified host principal. Bảo lưu DESIGN_BLOCKER_3D_STARTUP_WIRING.
