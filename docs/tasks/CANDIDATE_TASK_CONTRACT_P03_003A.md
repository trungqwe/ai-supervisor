# CANDIDATE TASK CONTRACT: TASK-P03-003A

> **Contract ID**: `CONTRACT-TASK-P03-003A-01`
> **Task ID**: `TASK-P03-003A` (Schema Migration, Domain Models & StateStore Operations Core)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P03`
> **Base SHA**: `1daf0b91efc7f065eb07146d642c0b6f6c259212`
> **Status**: `CANDIDATE_PENDING_EXTERNAL_SUPERVISOR_AUDIT`
> **Authority**: Formulated pursuant to accepted [ADR-016](../adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md) and approved canonical specifications ([docs/04](../04_ARCHITECTURE.md), [docs/05](../05_DOMAIN_MODEL.md), [docs/06](../06_WORKFLOW_STATE_MACHINE.md), [docs/08](../08_TASK_CONTRACT.md), [docs/12](../12_UPSTREAM_INTEGRATION.md), [docs/14](../14_FAILURE_RECOVERY.md), [docs/21](../21_TRACEABILITY_MATRIX.md), [docs/22](../22_MODULE_PROVENANCE.md), [docs/phases/P03_AO_INTEGRATION.md](../phases/P03_AO_INTEGRATION.md)).

---

> [!CRITICAL]
> **GOVERNANCE STATUS: CANDIDATE ARTIFACT / NOT RELEASED**.
> Production coding remains strictly **`HELD_PENDING_TASK_CONTRACT_RELEASE`** (`TASK_P03_003 = NOT_RELEASED`).
> Absolutely ZERO Go code implementation (`internal/**/*.go`) and ZERO database migration execution is authorized by this document until formally approved and released by External Supervisor audit.

---

## 1. Authoritative Canonical TaskContract JSON Object

This JSON object contains all mandatory and optional properties required by `docs/schemas/task-contract.schema.json`. It passes 100% full-schema structural validation.

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

---

## 2. Baseline SHA vs. Release Artifact SHA Distinction

A rigorous release protocol enforces absolute separation between the execution code baseline and the governance contract artifact:

1. **`base_sha` (`1daf0b91efc7f065eb07146d642c0b6f6c259212`)**:
   - Represents the **audited code baseline** commit that the worker will actually checkout to begin task execution.
   - Contains the verified source code, existing tests, and governance history up to the point of contract candidacy.
   - **Zero Circularity**: There is **no requirement** that `base_sha` must contain the contract artifact that pins `base_sha`. Requiring a commit to contain its own cryptographic hash is mathematically impossible.

2. **`release_artifact_sha` (Future Release Commit)**:
   - The commit SHA created in the repository that records, stores, and formalizes this candidate/final contract artifact along with the External Supervisor's governance release decision.
   - Resides downstream of `base_sha`.

3. **Supervisor Dispatch & Verification Protocol**:
   - When dispatching, the Supervisor delivers the immutable contract loaded from the separate release artifact (`release_artifact_sha`).
   - Prior to execution, the Supervisor verifies that the worker has checked out exactly `base_sha`.
   - Upon completion, the Supervisor gathers the implementation Git diff (`git diff base_sha..HEAD`) exclusively measured from `base_sha`.

---

## 3. Verification Requests & Two-Stage Validation Analysis

### 3.1 Conformance to ADR-013 Specification
- **Profile Identification**: Both requests utilize `profile_id: "go-test"`, complying with ADR-013 §4.1.
- **Parameters**:
  * `package`: `"./internal/store/..."` and `"./internal/domain/..."` start with `./` and resolve within the assigned workspace worktree.
  * `flags`: `["-v", "-race"]` are enumerated as permitted flags in ADR-013 §4.1. No external command execution flags (`-exec`, `-toolexec`) are present.
- **Path Containment (`cwd`)**: Set to `"."`, strictly satisfying the `worktree_root` policy.
- **Execution Ceiling**: `timeout_seconds` (300s and 120s) do not exceed the `MaxTimeoutSeconds = 300` constraint of the `go-test` profile.
- **Forbidden Execution Fields**: Zero occurrences of `executable`, `command`, `shell`, `args`, or `environment`.
- **Git Evidence Isolation**: `git-diff-check` is excluded from `verification_requests` per ADR-013 §4.4; Git evidence is collected directly by the Supervisor `EvidenceCollector`.

### 3.2 Host Catalog Accessibility & Stage B Status
- **Stage A (Structural Schema Validation)**: **PASS** (Exit Code 0). The JSON object fully validates against `docs/schemas/task-contract.schema.json`.
- **Stage B (Runtime Host Catalog Validation)**:
  * In Phase P03, the repository contains the pure domain interface `VerificationPolicyCatalog` (`internal/domain/verification_policy.go`), but concrete host execution runner binding is scheduled for Phase P04.
  * No running host catalog service or registry daemon is currently active or accessible in this environment.
  * **Recorded Status**: `STAGE_B_STATUS = UNVERIFIED`.
  * In accordance with governance directives, this candidate contract does **NOT** claim Stage B clearance until evaluated by a live `VerificationPolicyCatalog` instance at dispatch time.

---

## 4. Scope Boundaries

### Allowed Scope (`allowed_scope`):
```text
internal/domain/**
internal/store/**
```

### Forbidden Scope (`forbidden_scope`):
```text
internal/ao/**
internal/dispatch/**
internal/recovery/**
internal/poller/**
migrations/**
docs/adr/**
docs/proposals/**
docs/02_REQUIREMENTS.md
docs/04_ARCHITECTURE.md
docs/05_DOMAIN_MODEL.md
docs/06_WORKFLOW_STATE_MACHINE.md
docs/08_TASK_CONTRACT.md
docs/12_UPSTREAM_INTEGRATION.md
docs/14_FAILURE_RECOVERY.md
docs/17_ROADMAP.md
docs/18_CURRENT_STATE.md
docs/21_TRACEABILITY_MATRIX.md
docs/22_MODULE_PROVENANCE.md
docs/phases/**
AGENTS.md
* (any file outside allowed_scope)
```

---

## 5. Schema Comparison & Migration Specification (ADR-016 §27)

| Entity / Table | Current State in v2 (`internal/store/migrations.go`) | Target State in v3 (ADR-016 §27) | Migration Action Required |
|---|---|---|---|
| **`worker_sessions`** | **ABSENT** | `CREATE TABLE worker_sessions (...)` | Create new table with `pair_id PRIMARY KEY REFERENCES pairs(pair_id) ON DELETE RESTRICT`, `session_id UNIQUE`, `worktree_path TEXT NULL`, `status CHECK`, `terminal_generation`, `quarantine_state DEFAULT 'CLEAN' CHECK`. |
| **`task_attempts`** | Exists in v1 (lines 76–88); lacks snapshot columns | Snapshot columns added | Migrate table to add: `session_id TEXT`, `terminal_generation TEXT`, `recovery_disposition TEXT`, `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))`. Zero undeclared columns. |
| **`pair_provisioning_operations`** | **ABSENT** | `CREATE TABLE pair_provisioning_operations (...)` | Create new table with `pair_id REFERENCES pairs(pair_id) ON DELETE RESTRICT`, `stage CHECK`, and partial unique index `idx_pair_provisioning_unresolved` on `(pair_id) WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')`. |
| **`dispatch_operations`** | **ABSENT** | `CREATE TABLE dispatch_operations (...)` | Create new table with `attempt_id UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT`, `pair_id REFERENCES pairs(pair_id) ON DELETE RESTRICT`, `task_id REFERENCES tasks(task_id) ON DELETE RESTRICT`, `session_id`, `terminal_generation`, `stage CHECK`, `resolution_state TEXT`. |
| **`stop_operations`** | **ABSENT** | `CREATE TABLE stop_operations (...)` | Create new table with `purpose CHECK`, `pair_id REFERENCES pairs(pair_id) ON DELETE RESTRICT`, optional references to `tasks`, `task_contracts`, `task_attempts` with `ON DELETE RESTRICT`, `stage CHECK`, `resolution_state DEFAULT 'IN_FLIGHT'`. |
| **Schema Version** | `CurrentSchemaVersion = 2` | `CurrentSchemaVersion = 3` | Update constant to `3`; add `v3Schema` DDL and sequential migration block in `migrateWithSchemas`. |

---

## 6. Acceptance Criteria

| ID | Category | Objective Condition |
|---|---|---|
| **AC-3A-01** | **Migration Execution** | `internal/store/migrations.go` defines `CurrentSchemaVersion = 3`. Running migrations on a fresh DB or v2 DB advances `PRAGMA user_version = 3` and creates all 4 new tables/indexes cleanly. |
| **AC-3A-02** | **Migration Idempotence** | Executing `migrate(ctx, db)` multiple times consecutively on the same database yields nil error and does not corrupt schema or data. |
| **AC-3A-03** | **Foreign Key RESTRICT** | Inserting child rows with invalid parent IDs fails. Deleting parent rows referenced by `worker_sessions`, `pair_provisioning_operations`, `dispatch_operations`, or `stop_operations` fails closed with foreign key constraint violation (`ON DELETE RESTRICT`). |
| **AC-3A-04** | **Partial Unique Index** | Inserting two `pair_provisioning_operations` rows for the same `pair_id` with `stage = 'PROVISION_REQUESTED'` or `'PROVISION_FAILED'` fails with unique constraint violation. Inserting a second row with `stage = 'PROVISION_CONFIRMED'` succeeds. |
| **AC-3A-05** | **WorkerSession Model & CRUD** | `internal/domain` defines `WorkerSession` struct with all ADR-016 fields. `internal/store` provides `CreateWorkerSession`, `GetWorkerSessionByPair`, `GetWorkerSessionByID`, and `UpdateWorkerSessionStatusAndQuarantine`. |
| **AC-3A-06** | **Saga Operations Models & CRUD** | `internal/domain` defines `PairProvisioningOperation`, `DispatchOperation`, and `StopOperation`. `internal/store` provides atomic insert and stage-update methods for all three operations. |
| **AC-3A-07** | **TaskAttempt Snapshot Fields** | `internal/domain.TaskAttempt` and store queries handle snapshot fields (`SessionID`, `TerminalGeneration`, `RecoveryDisposition`, `QuarantineState`). Reading attempts populated in v1/v2 defaults `QuarantineState` to `'CLEAN'`. |
| **AC-3A-08** | **AtomicTerminalTransition** | StateStore executes CAS state change, `ended_at = now`, and `recovery_disposition` in a single SQLite transaction. Mismatched `expectedCurrentState` aborts transaction and returns conflict error. |
| **AC-3A-09** | **AtomicAttemptClosureTransition** | StateStore atomically escalates task from `BLOCKED` to `HUMAN_REQUIRED`, closes attempt with `recovery_disposition = 'AO_BLOCKED_ESCALATED'`, and appends audit event. |
| **AC-3A-10** | **Test Coverage & Zero Leaks** | `go test -v -race ./internal/domain/... ./internal/store/...` passes with zero failures and zero race warnings. |

---

## 7. Required Evidence

1. `git_diff`: Git diff confirming changes are strictly within `internal/domain/**` and `internal/store/**`.
2. `git_diff_check`: Command output of `git diff --check` demonstrating zero trailing whitespace, syntax hygiene, or merge conflict markers.
3. `test_exit_code_zero`: Command output demonstrating `go test -v -race ./internal/store/... ./internal/domain/...` exited with code 0.
4. `sqlite_schema_verification`: Test logs explicitly proving table DDL, `PRAGMA user_version = 3`, foreign key `ON DELETE RESTRICT` enforcement, and partial unique index behavior.
5. `race_detector_zero_warnings`: `-race` execution logs clean of data races.

---

## 8. Preserved Architectural Blockers

- **`DESIGN_BLOCKER_3D_STARTUP_WIRING`**:
  * Preserved for Sub-Contract 3D.
  * In the absence of a top-level daemon bootstrap entrypoint (`cmd/`), an isolated test harness can only prove the scanner's invocation interface, preconditions, and return conditions (call contract / interface readiness). It cannot prove actual runtime execution ordering of the daemon.
  * Contract 3A execution is independent of this blocker.

---

## 9. Governance Gate & Stop Conditions

- **Stop Conditions**: The worker must immediately halt, make NO changes, and emit a `WorkerReport` with `status: "BLOCKED"` if:
  1. SQLite migration requires tables or columns conflicting with ADR-016 §27;
  2. `internal/store` existing interfaces cannot support atomic transactions without modifying forbidden scope;
  3. CGO dependency or non-pure-Go driver is required;
  4. Any instruction attempts to modify files outside `allowed_scope`.
- **Active Governance State**:
  * `TASK_P03_003 = NOT_RELEASED`
  * `P03_CODE = HELD_PENDING_TASK_CONTRACT_RELEASE`
  * `ACTIVE_GATE = TASK_P03_003_CONTRACT_PLANNING`
