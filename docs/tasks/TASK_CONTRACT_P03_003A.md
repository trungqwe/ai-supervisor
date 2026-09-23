# TASK CONTRACT: TASK-P03-003A

> **Contract ID**: `CONTRACT-TASK-P03-003A-01`
> **Task ID**: `TASK-P03-003A` (Schema Migration, Domain Models & StateStore Operations Core)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P03`
> **Base SHA**: `1daf0b91efc7f065eb07146d642c0b6f6c259212`
> **Status**: `RELEASED`
> **Authority**: External Supervisor phê duyệt candidate commit `56b5ef38ba5a45d56d58dd368a0c47cf45e0886f`, Git blob `5364edf076b31b8a42a37fee29193fc36ccdf413`.

---

> [!CRITICAL]
> Contract này chỉ release phần `TASK-P03-003A`. Các phần `TASK-P03-003B`, `TASK-P03-003C` và `TASK-P03-003D` chưa được release.
> Implementation phải bắt đầu từ đúng `base_sha`, chỉ sửa `internal/domain/**` và `internal/store/**`, rồi dừng chờ audit kết quả 3A.

## 1. Authoritative Canonical TaskContract JSON Object

JSON dưới đây được giữ nguyên từ candidate đã được External Supervisor phê duyệt.

```json
{
  "contract_id": "CONTRACT-TASK-P03-003A-01",
  "task_id": "TASK-P03-003A",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P03",
  "objective": "Implement foundational persistence and domain layer for ADR-016 in internal/store and internal/domain: SQLite schema migration to version 3 creating worker_sessions, migrating task_attempts snapshot columns, creating pair_provisioning_operations, dispatch_operations, and stop_operations with ON DELETE RESTRICT and partial unique index idx_pair_provisioning_unresolved; corresponding domain aggregate models in internal/domain; StateStore CRUD and atomic CAS state transitions in internal/store.",
  "requirements": [
    "FR-005",
    "FR-006",
    "NFR-003",
    "NFR-005",
    "OPS-002",
    "SEC-001"
  ],
  "architecture_refs": [
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-27",
    "docs/04_ARCHITECTURE.md#section-2.1",
    "docs/05_DOMAIN_MODEL.md#section-3",
    "docs/08_TASK_CONTRACT.md",
    "docs/14_FAILURE_RECOVERY.md#section-1.2",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md#section-3"
  ],
  "base_sha": "1daf0b91efc7f065eb07146d642c0b6f6c259212",
  "allowed_scope": [
    "internal/domain/**",
    "internal/store/**"
  ],
  "forbidden_scope": [
    "internal/ao/**",
    "internal/dispatch/**",
    "internal/recovery/**",
    "internal/poller/**",
    "migrations/**",
    "docs/adr/**",
    "docs/proposals/**",
    "docs/02_REQUIREMENTS.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/06_WORKFLOW_STATE_MACHINE.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/12_UPSTREAM_INTEGRATION.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/17_ROADMAP.md",
    "docs/18_CURRENT_STATE.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md",
    "docs/phases/**",
    "AGENTS.md"
  ],
  "constraints": [
    "Verbatim DDL compliance with ADR-016 section 27 (zero field omission, zero column renaming, zero invented fields)",
    "Strict foreign key integrity with ON DELETE RESTRICT on all child foreign keys across worker_sessions, pair_provisioning_operations, dispatch_operations, and stop_operations",
    "Partial unique index idx_pair_provisioning_unresolved on pair_provisioning_operations(pair_id) WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')",
    "Idempotent migration execution: migrate(ctx, db) must succeed idempotently across fresh DB, v1 DB, v2 DB, and already-migrated v3 DB without data corruption",
    "Atomic state transitions: AtomicTerminalTransition and AtomicAttemptClosureTransition must execute CAS, task_attempts updates, and audit events within a single SQLite transaction",
    "Pure Go implementation baseline without CGO dependencies (modernc.org/sqlite)",
    "Thread safety and zero leaks: connection pool safe handling, all tests pass with -race"
  ],
  "acceptance_criteria": [
    "AC-3A-01: internal/store/migrations.go defines CurrentSchemaVersion = 3 and applies all v3 tables and indexes cleanly advancing PRAGMA user_version = 3",
    "AC-3A-02: Executing migrate(ctx, db) repeatedly on the same database succeeds idempotently with nil error and zero corruption",
    "AC-3A-03: Foreign key ON DELETE RESTRICT prevents deleting referenced parent rows across pairs, tasks, and task_attempts",
    "AC-3A-04: Partial unique index idx_pair_provisioning_unresolved prevents duplicate unconfirmed provisioning rows for the same pair_id",
    "AC-3A-05: internal/domain defines WorkerSession aggregate and internal/store implements WorkerSession CRUD and CAS status/quarantine updates",
    "AC-3A-06: internal/domain defines PairProvisioningOperation, DispatchOperation, StopOperation and internal/store implements atomic CRUD and stage transitions",
    "AC-3A-07: TaskAttempt snapshot fields (session_id, terminal_generation, recovery_disposition, quarantine_state) are stored and read with CLEAN default",
    "AC-3A-08: AtomicTerminalTransition executes CAS state transition, ended_at, recovery_disposition, and audit event atomically in a single transaction",
    "AC-3A-09: AtomicAttemptClosureTransition escalates BLOCKED to HUMAN_REQUIRED, closes attempt with AO_BLOCKED_ESCALATED, and logs audit event atomically",
    "AC-3A-10: Test suite passes cleanly with zero failures and zero race warnings under go test -v -race ./internal/domain/... ./internal/store/..."
  ],
  "verification_requests": [
    {
      "id": "VR-P03-003A-STORE-TESTS",
      "profile_id": "go-test",
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
      "id": "VR-P03-003A-DOMAIN-TESTS",
      "profile_id": "go-test",
      "parameters": {
        "package": "./internal/domain/...",
        "flags": [
          "-v",
          "-race"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 120
    }
  ],
  "required_evidence": [
    "git_diff",
    "git_diff_check",
    "test_exit_code_zero",
    "sqlite_schema_verification",
    "race_detector_zero_warnings"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "SQLite migration requires tables or columns conflicting with ADR-016 section 27",
    "internal/store existing interfaces cannot support atomic transactions without modifying forbidden scope",
    "CGO dependency or non-pure-Go SQLite driver is required",
    "Any implementation instruction or dependency modification attempts to touch files outside allowed_scope",
    "Evidence of unexpected schema lock contention or unrecoverable migration failure"
  ]
}
```

## 2. Release Identity và Execution Baseline

- `base_sha = 1daf0b91efc7f065eb07146d642c0b6f6c259212` là code baseline bất biến để tạo branch implementation.
- `release_artifact_sha` là SHA của commit governance chứa contract này, release audit, `AGENTS.md` và `docs/18_CURRENT_STATE.md`; SHA thực tế được báo cáo sau khi tạo commit, không được tự nhúng vào chính commit đó.
- Supervisor phải nạp contract và quyết định release từ `release_artifact_sha`, đồng thời đo diff implementation bằng `git diff base_sha..HEAD`.
- Candidate [CANDIDATE_TASK_CONTRACT_P03_003A.md](CANDIDATE_TASK_CONTRACT_P03_003A.md) được giữ nguyên làm lịch sử.

## 3. Verification Policy và ranh giới P04

Contract được kiểm tra bằng hai lớp độc lập:

1. Full-schema validation dùng `docs/schemas/task-contract.schema.json` để kiểm tra cấu trúc JSON.
2. `TaskContractValidator.ValidateRaw` dùng `VerificationPolicyCatalog` chứa metadata `go-test` đã được External Supervisor phê duyệt để kiểm tra schema tham số, `cwd`, timeout và scope semantics.

Việc semantic validation thành công chỉ chứng minh contract phù hợp với policy metadata. Nó không dựng command, không chạy process, không cung cấp runner isolation và không chứng minh P04 verification runner đã được triển khai. Executable binding, argument construction, process execution và runtime isolation tiếp tục thuộc P04.

Policy `go-test` dùng cho 3A:

- `ProfileID`: `go-test`
- `CwdPolicy`: `worktree_root`
- `MaxTimeoutSeconds`: `300`
- `ParameterSchema`: object; `additionalProperties = false`; bắt buộc `package` và `flags`
- `package`: chỉ nhận `./internal/store/...` hoặc `./internal/domain/...`
- `flags`: đúng hai phần tử duy nhất từ enum `-v`, `-race`

## 4. Scope và gate

- Allowed scope: `internal/domain/**`, `internal/store/**`.
- Không triển khai coordinator, AO calls, poller, startup scanner hoặc phần việc 3B–3D.
- `DESIGN_BLOCKER_3D_STARTUP_WIRING` được bảo lưu cho 3D và không chặn 3A.
- `TASK_P03_003A = RELEASED`
- `P03_CODE = AUTHORIZED_3A_ONLY`
- `ACTIVE_GATE = TASK_P03_003A_IMPLEMENTATION`
