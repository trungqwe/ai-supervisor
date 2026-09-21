# ADR-015: Local State Store Engine and Durability Specification

> **Status**: ACCEPTED
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Deciders**: External Supervisor, Engineering Team
> **Consulted**: `docs/decisions/P02_Q4_STATE_STORE_RESEARCH.md`

---

## 1. Context and Problem Statement

The Supervisor Control Plane requires a local, zero-cloud data store (OPS-003) to persist:
1. Multi-project registrations and active pair bindings (FR-001, FR-002, FR-014);
2. Immutable `TaskContract` revisions and `TaskAttempt` lineage (FR-004, ADR-012);
3. The 13-state attempt workflow (FR-011);
4. Append-only audit events (FR-013, NFR-004).

Crucially, the pre-dispatch invariant dictates that the state transition `READY → DISPATCHED` and `TaskAttempt` allocation must be durably committed before initiating any external Agent Orchestrator invocation.

---

## 2. Decision: SQLite (STATE_STORE_ENGINE = SQLITE)

The canonical storage engine for the Supervisor Control Plane is **SQLite**.

### Implementation Driver Direction:
- For the Go implementation core (ADR-014), the preferred driver is **`modernc.org/sqlite`**:
  - 100% pure Go implementation transpiled from official SQLite C code via `ccgo`.
  - Zero CGo compiler requirement on Windows.
  - Compiles directly into the standalone `supervisor.exe` binary, satisfying OPS-001 and OPS-003.
  - *Note*: Exact driver dependency versioning will be pinned during Phase P02 implementation bootstrap rather than frozen as an immutable architecture constant.

---

## 3. Durability Baseline & Concurrency Policies

### 3.1 Durability Configuration
All SQLite database connections must be initialized with the following pragma configuration:
```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = FULL;
PRAGMA foreign_keys = ON;
```

### 3.2 Synchronization Guarantees & Contract Wording
- Under SQLite's documented VFS and filesystem synchronization contract, Write-Ahead Logging (WAL) combined with `synchronous = FULL` ensures that the WAL file is flushed to physical storage upon every transaction commit.
- This configuration provides the durability required for the pre-dispatch persistence invariant across daemon restarts and unexpected process terminations.
- *Note*: As with all database engines, durable persistence relies upon the underlying OS and disk subsystem honoring synchronous flush requests.

### 3.3 Concurrency & Writer Serialization
- **Concurrency Semantics**: WAL mode enables multiple concurrent read operations without blocking or being blocked by the single active writer.
- **Serialization Invariant**: SQLite strictly serializes write operations. Concurrent write attempts are handled through a **configurable bounded busy timeout policy** (e.g. an initial default of 5000 ms, configurable via host settings).

### 3.4 Transaction Lifetime Invariant
- **Rule**: Write transactions must be strictly short-lived.
- Under no circumstances may external Agent Orchestrator REST calls, network operations, verification command child process executions, repository scans, large file parsing, or long computations take place inside an open database transaction.

---

## 4. Multi-File Topology & Active Backup Policy

### 4.1 Runtime Files
An active SQLite database in WAL mode comprises up to three files:
1. `supervisor.db` — The primary database file.
2. `supervisor.db-wal` — The write-ahead log containing committed pages pending checkpoint.
3. `supervisor.db-shm` — The shared-memory index coordinating concurrent readers.

### 4.2 Active Database Backup Invariant
- **Prohibited Procedure**: Copying only `supervisor.db` while WAL is active is strictly prohibited, as it risks capturing an inconsistent or corrupted snapshot missing uncheckpointed transactions.
- **Approved Procedures**:
  1. **SQLite Online Backup API** (`sqlite3_backup` API): Streams consistent database pages while the database is live.
  2. **`VACUUM INTO 'backup_path.db';`**: Atomically writes a clean, fully checkpointed snapshot file.
  3. Direct multi-file filesystem snapshots require explicit maintenance / quiesced mode; `wal_checkpoint(TRUNCATE)` followed by file copying is not a standard online backup procedure.

---

## 5. Consequences

### Positive:
- True ACID transaction guarantees for the pre-dispatch invariant.
- Relational integrity across projects, pairs, tasks, contracts, attempts, and audit events.
- Zero external daemon, service, or cloud configuration (OPS-003).
- Safe concurrent read access for background observation without blocking state transitions.

### Negative / Tradeoffs:
- Single active writer requires strict transaction discipline and bounded write durations.
