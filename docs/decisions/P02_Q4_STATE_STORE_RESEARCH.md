# P02 Q4 State Store Research Dossier (Final Decision)

> **Authority**: P02 Final Implementation Decision Canonicalization
> **Date**: 2026-09-22
> **Status**: DECISION_ACCEPTED
> **Final Decision**: `SQLITE_SELECTED`
> **Durability Baseline**: `WAL` + `synchronous = FULL`

---

## 1. Final Decision Summary

The External Supervisor and engineering governance have selected **SQLite** as the canonical state store engine for the Supervisor Control Plane core:
```
STATE_STORE_ENGINE = SQLITE
JOURNAL_MODE       = WAL
SYNCHRONOUS        = FULL
FOREIGN_KEYS       = ON
```
This selection is based on verified ACID transactionality, zero cloud dependencies (OPS-003), atomic pre-dispatch intent durability, and robust Windows filesystem operation.

---

## 2. Durability Specification & Pre-Dispatch Invariant

### 2.1 The Atomic Dispatch Boundary
The core workflow requires that before any external Agent Orchestrator invocation occurs:
1. `TaskAttempt` allocation is recorded;
2. Foreign key relationship to `contract_id` is persisted;
3. State transition `READY → DISPATCHED` is committed in the database.

### 2.2 Durability Contract (`synchronous = FULL`)
- Under SQLite's documented VFS and filesystem synchronization contract, Write-Ahead Logging (WAL) combined with `synchronous = FULL` ensures that the WAL is explicitly flushed to physical media upon every commit.
- This satisfies the pre-dispatch invariant across application crashes, daemon restarts, and system interruptions.

### 2.3 Concurrency & Busy Timeout Policy
- WAL mode allows unlimited concurrent readers while serializing writers.
- Writer contention is managed through a **configurable bounded busy timeout policy** (e.g. defaulting to 5000 ms), rather than an immutable constant.
- **Short-Lived Transactions**: All write transactions must be strictly short-lived. Long computations, large file parsing, external process spawning, and AO REST API calls are strictly forbidden inside an open database transaction.

---

## 3. Active Backup & Runtime Topology

### 3.1 Runtime File Composition
At runtime, the active database consists of:
- `supervisor.db` (Main database file)
- `supervisor.db-wal` (Write-ahead log)
- `supervisor.db-shm` (Shared memory coordination index)

### 3.2 Canonical Backup Mechanisms
- Naive filesystem copying of only `supervisor.db` while WAL is active is strictly forbidden.
- Canonical backups must use either:
  1. The **SQLite Online Backup API** (`sqlite3_backup`), or
  2. The SQL atomic snapshot command: **`VACUUM INTO 'backup_path.db';`**.

---

## 4. Technology Selection Matrix

| Evaluation Criterion | Weight | SQLite (WAL Mode) | Durable JSON / File Store | Embedded Key-Value Store |
|---|:---:|:---:|:---:|:---:|
| ACID & Multi-Entity Atomicity | 20% | **10 / 10** (Atomic multi-table commit) | 3 / 10 (Non-atomic across files) | 7 / 10 (Complex multi-key atomicity) |
| Relational Attempt Lineage Queries | 15% | **10 / 10** (Foreign keys, joins, index) | 2 / 10 (Manual memory traversal) | 4 / 10 (Requires secondary index) |
| Crash Consistency & Recovery | 15% | **10 / 10** (Documented VFS WAL flush) | 4 / 10 (Risk of partial write) | 8 / 10 (LSM WAL recovery) |
| Windows Concurrency Handling | 15% | **9 / 10** (Concurrent readers + writer) | 4 / 10 (File lock contention) | 7 / 10 |
| Schema Migration Lifecycle | 10% | **9 / 10** (DDL scripts, user_version) | 4 / 10 (Ad-hoc transformations) | 5 / 10 (Byte format migrations) |
| Online Backup & Zero Cloud | 10% | **10 / 10** (Online backup API, VACUUM) | 6 / 10 (File tree copy) | 7 / 10 (Snapshot directory) |
| Operational Simplicity | 15% | **9 / 10** (Standard tooling, CLI inspect) | 8 / 10 (Raw JSON files) | 6 / 10 (Binary LSM files) |
| **Weighted Total** | **100%** | **9.50 / 10** | **4.95 / 10** | **6.85 / 10** |

---

## 5. Governance Verdict

- **Decision**: **`SQLITE_SELECTED`**.
- **ADR Reference**: [ADR-015](file:///d:/TU_CODE/ai-supervisor/docs/adr/ADR-015-local-state-store-engine.md) (ACCEPTED).
- Implementation starts in Phase P02 upon completion of the implementation release audit.
