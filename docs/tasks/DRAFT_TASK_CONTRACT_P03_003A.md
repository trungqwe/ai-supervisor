# DRAFT TASK CONTRACT: TASK-P03-003A

> **Contract ID**: `CONTRACT-TASK-P03-003A-01` (DRAFT — UNRELEASED)
> **Task ID**: `TASK-P03-003A` (Schema Migration, Domain Models & StateStore Operations Core)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P03`
> **Base SHA**: To be pinned to the formal release commit SHA at dispatch time (Draft reference: latest released commit; not defaulting to historical canonical audit SHA `41cf769b4cf8a980c95de3aa4be63140b04922c3`)
> **Status**: `DRAFT_PENDING_EXTERNAL_SUPERVISOR_APPROVAL`
> **Authority**: Formulated pursuant to accepted [ADR-016](../adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md) and approved canonical specifications ([docs/04](../04_ARCHITECTURE.md), [docs/05](../05_DOMAIN_MODEL.md), [docs/06](../06_WORKFLOW_STATE_MACHINE.md), [docs/08](../08_TASK_CONTRACT.md), [docs/12](../12_UPSTREAM_INTEGRATION.md), [docs/14](../14_FAILURE_RECOVERY.md), [docs/21](../21_TRACEABILITY_MATRIX.md), [docs/22](../22_MODULE_PROVENANCE.md), [docs/phases/P03_AO_INTEGRATION.md](../phases/P03_AO_INTEGRATION.md)).

---

> [!CRITICAL]
> **GOVERNANCE STATUS: DRAFT ONLY / NOT RELEASED**.
> Production coding remains strictly **`HELD_PENDING_TASK_CONTRACT_RELEASE`** (`TASK_P03_003 = NOT_RELEASED`).
> Absolutely ZERO Go code implementation (`internal/**/*.go`) and ZERO database migration execution is authorized by this document until formally approved and released by External Supervisor audit.

---

## 1. Objective

Implement the foundational persistence and domain layer for ADR-016 in `internal/store` and `internal/domain`:
1. Version 3 SQLite schema migration in `internal/store/migrations.go` creating `worker_sessions`, migrating `task_attempts` snapshot columns, and creating `pair_provisioning_operations`, `dispatch_operations`, and `stop_operations` with strict `ON DELETE RESTRICT` foreign key constraints and partial unique index `idx_pair_provisioning_unresolved`;
2. Corresponding aggregate domain structs in `internal/domain`;
3. StateStore CRUD operations and multi-table atomic CAS transitions (`AtomicTerminalTransition`, `AtomicAttemptClosureTransition`) in `internal/store`.

---

## 2. Requirements & Architecture Traceability

- **Traceable Requirements**:
  * `FR-005`: Worker Dispatch (persistence of dispatch operations and session bindings)
  * `FR-006`: Worker Observation (persistence of session activity states and terminal generation)
  * `NFR-003`: Crash Recoverability & Idempotence (durable saga stage persistence and atomic transitions)
  * `NFR-005`: Upstream Decoupling & Anti-Corruption (Model A snapshot persistence, zero domain pollution)
  * `OPS-002`: Clean Termination & Lifecycle Management (stop operations persistence and purpose tracking)
  * `SEC-001`: Sandboxed Containment & Quarantine (quarantine state persistence on session and attempt)
- **Governing Architecture References**:
  * `docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md` (§27 Persistence Impact DDL)
  * `docs/05_DOMAIN_MODEL.md` (§2 Entity Dictionary, §3 Durable Saga Operations, §4 Model A Persistence)
  * `docs/04_ARCHITECTURE.md` (§2.1 Task Dispatch Flow, §2.2 Completion Flow)
  * `docs/14_FAILURE_RECOVERY.md` (§1.2 Double-Gated Quarantine & Startup Sweep)
  * `docs/22_MODULE_PROVENANCE.md` (§3 Store Entities Provenance)
  * `docs/sources/SOURCE_REGISTRY.md` & `docs/sources/REUSE_MATRIX.md`

---

## 3. Scope Boundaries

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

> [!IMPORTANT]
> - **Zero Changes to Coordination Packages**: Sub-Contract 3A strictly forbids modifying `internal/ao/**` or creating/modifying coordination packages (`internal/dispatch/**`, `internal/recovery/**`, `internal/poller/**`). Those components belong to Sub-Contracts 3B, 3C, and 3D.
> - **Actual Migration Path**: Schema migrations MUST be added directly to the existing migration engine in `internal/store/migrations.go`. Do **NOT** create a separate top-level `migrations/` directory.

---

## 4. Schema Comparison & Migration Specification (ADR-016 §27)

Comparison between existing `internal/store/migrations.go` (v2) and target schema (v3):

| Entity / Table | Current State in v2 (`internal/store/migrations.go`) | Target State in v3 (ADR-016 §27) | Migration Action Required |
|---|---|---|---|
| **`worker_sessions`** | **ABSENT** | `CREATE TABLE worker_sessions (...)` | Create new table with `pair_id PRIMARY KEY REFERENCES pairs(pair_id) ON DELETE RESTRICT`, `session_id UNIQUE`, `worktree_path TEXT NULL`, `status CHECK`, `terminal_generation`, `quarantine_state DEFAULT 'CLEAN' CHECK`. |
| **`task_attempts`** | Exists in v1 (lines 76–88); lacks snapshot columns | Snapshot columns added | Migrate table to add: `session_id TEXT`, `terminal_generation TEXT`, `recovery_disposition TEXT`, `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))`. Zero undeclared columns. |
| **`pair_provisioning_operations`** | **ABSENT** | `CREATE TABLE pair_provisioning_operations (...)` | Create new table with `pair_id REFERENCES pairs(pair_id) ON DELETE RESTRICT`, `stage CHECK`, and partial unique index `idx_pair_provisioning_unresolved` on `(pair_id) WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')`. |
| **`dispatch_operations`** | **ABSENT** | `CREATE TABLE dispatch_operations (...)` | Create new table with `attempt_id UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT`, `pair_id REFERENCES pairs(pair_id) ON DELETE RESTRICT`, `task_id REFERENCES tasks(task_id) ON DELETE RESTRICT`, `session_id`, `terminal_generation`, `stage CHECK`, `resolution_state TEXT`. |
| **`stop_operations`** | **ABSENT** | `CREATE TABLE stop_operations (...)` | Create new table with `purpose CHECK`, `pair_id REFERENCES pairs(pair_id) ON DELETE RESTRICT`, optional references to `tasks`, `task_contracts`, `task_attempts` with `ON DELETE RESTRICT`, `stage CHECK`, `resolution_state DEFAULT 'IN_FLIGHT'`. |
| **Schema Version** | `CurrentSchemaVersion = 2` | `CurrentSchemaVersion = 3` | Update constant to `3`; add `v3Schema` DDL and sequential migration block in `migrateWithSchemas`. |

---

## 5. Constraints

1. **Exact DDL Fidelity**: Verbatim compliance with ADR-016 §27. No renaming, column omission, or invented fields.
2. **Foreign Key Integrity**: Every foreign key MUST include `ON DELETE RESTRICT`. SQLite foreign keys pragma must be verified in tests.
3. **Partial Unique Index**: `idx_pair_provisioning_unresolved` must enforce at most one unresolved provisioning operation per Pair (`WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')`).
4. **Idempotence**: Running `migrate(ctx, db)` on a fresh database, a v1 database, a v2 database, or an already-migrated v3 database must succeed idempotently without data loss or duplicate index errors.
5. **Atomic Transitions**:
   - `AtomicTerminalTransition(ctx, taskID, expectedCurrentState, newState, failureReason, attemptID, disposition)`: Executes CAS state transition on `tasks`, sets `task_attempts.ended_at = now`, records `task_attempts.recovery_disposition`, and records audit event in a single SQLite transaction.
   - `AtomicAttemptClosureTransition(ctx, taskID, attemptID, disposition)`: Atomically transitions `tasks.state` from `BLOCKED` to `HUMAN_REQUIRED`, sets `task_attempts.ended_at = now`, records `recovery_disposition = 'AO_BLOCKED_ESCALATED'`, and logs audit event in a single transaction.
6. **Pure Go**: No CGO dependencies (`modernc.org/sqlite` baseline).
7. **Thread Safety & Zero Leaks**: All store queries must manage connection pools safely; tests must pass with `-race`.

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

## 7. Verification Requests (ADR-013)

```json
[
  {
    "request_id": "VR-P03-003A-STORE-TESTS",
    "profile": "go-test",
    "parameters": {
      "package_pattern": "./internal/store/...",
      "flags": ["-v", "-race"]
    }
  },
  {
    "request_id": "VR-P03-003A-DOMAIN-TESTS",
    "profile": "go-test",
    "parameters": {
      "package_pattern": "./internal/domain/...",
      "flags": ["-v", "-race"]
    }
  },
  {
    "request_id": "VR-P03-003A-FORMAT-HYGIENE",
    "profile": "git-diff-check",
    "parameters": {
      "flags": ["--check"]
    }
  }
]
```

---

## 8. Required Evidence

1. `git_diff`: Git diff confirming changes are strictly within `internal/domain/**` and `internal/store/**`.
2. `test_exit_code_zero`: Command output demonstrating `go test -v -race ./internal/store/... ./internal/domain/...` exited with code 0.
3. `sqlite_schema_verification`: Test logs explicitly proving table DDL, `PRAGMA user_version = 3`, foreign key `ON DELETE RESTRICT` enforcement, and partial unique index behavior.
4. `race_detector_zero_warnings`: `-race` execution logs clean of data races.

---

## 9. Worker Profile & Report Contract

- **Worker Profile**: `antigravity-standard`
- **Report Contract**: `docs/09_WORKER_REPORT.md` (validated against `worker-report.schema.json`)

---

## 10. Stop Conditions

The worker must immediately halt, make NO changes, and emit a `WorkerReport` with `status: "BLOCKED"` if:
1. SQLite migration requires tables or columns conflicting with ADR-016 §27;
2. `internal/store` existing interfaces cannot support atomic transactions without touching `internal/ao/**` or forbidden files;
3. CGO dependency or non-pure-Go driver is required;
4. Any instruction attempts to modify files outside `allowed_scope`.
