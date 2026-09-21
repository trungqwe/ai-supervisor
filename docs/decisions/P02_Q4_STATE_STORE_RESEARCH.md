# P02 Q4 State Store Research & Durability Specification

> **Authority**: P02 Pre-Code Implementation Decision Gate (Reaudit Remediation)
> **Date**: 2026-09-22
> **Status**: SQLITE_DIRECTION_APPROVED_CONFIGURATION_PENDING_EXTERNAL_DECISION
> **Prior Direction**: SQLite direction accepted; durability configuration corrected to `synchronous=FULL`.

---

## 1. Engine Selection Baseline

The External Supervisor has confirmed the storage engine direction:
```
STATE_STORE_ENGINE_DIRECTION = SQLITE
```
This dossier establishes the exact operational, concurrency, durability, and backup configuration required to satisfy the Supervisor Control Plane's atomic pre-dispatch persistence invariant.

---

## 2. Durability Configuration & Power-Loss Safety

### 2.1 The Dispatch Transaction Invariant
The canonical dispatch boundary requires:
1. `TaskAttempt` is allocated;
2. Foreign key to `contract_id` is bound;
3. State transition `READY → DISPATCHED` is committed in the database;
4. **All steps 1–3 must be durably committed to disk BEFORE invoking external AO dispatch.**

### 2.2 Correction: `synchronous = FULL` vs `NORMAL`
In SQLite WAL mode:
- `synchronous = NORMAL`: SQLite syncs the WAL file on checkpoint operations, but not on every transaction commit. While resistant to application crashes, **a hard OS crash or power loss can lose recently committed transactions**.
- **Failure Mode with `NORMAL`**: If the Supervisor commits `READY → DISPATCHED`, invokes AO to spawn an external worker, and power is suddenly interrupted before a WAL sync, the database rolls back upon reboot. The external worker would be executing an attempt for which the Supervisor has zero record, corrupting the attempt lineage.
- **Canonical V1 Durability Baseline**:
  ```sql
  PRAGMA journal_mode = WAL;
  PRAGMA synchronous = FULL;
  PRAGMA foreign_keys = ON;
  PRAGMA busy_timeout = 5000;
  ```
  Under `synchronous = FULL` in WAL mode, the WAL file is flushed to physical storage on every commit, guaranteeing full ACID durability across hard OS crashes and power loss.

---

## 3. WAL Concurrency Model & Precision

### 3.1 Concurrency Semantics
- **Clarification**: WAL mode **does not eliminate write contention**.
- **Precise Invariant**:
  - WAL allows **any number of concurrent readers** alongside **exactly one writer**. Readers never block writers, and writers never block readers.
  - However, **writers are strictly serialized**. If two threads or processes attempt to write simultaneously, one must wait or will receive `SQLITE_BUSY`.
- **Mitigation & Invariant**:
  - `PRAGMA busy_timeout = 5000;` ensures SQLite will automatically retry for up to 5 seconds if a lock contention occurs.
  - All write transactions must be **strictly bounded and short-lived (< 50ms)**. No external I/O, child process execution, or network calls may ever take place inside an active database transaction.

### 3.2 Runtime Multi-File Topology
- **Clarification**: While commonly referred to as a "single-file database", an active SQLite database in WAL mode consists of up to three runtime files:
  1. `supervisor.db` — The primary database file containing committed schema pages.
  2. `supervisor.db-wal` — The write-ahead log file containing committed transaction pages pending checkpoint.
  3. `supervisor.db-shm` — The shared-memory index file used for concurrent reader coordination.
- On clean daemon shutdown, SQLite automatically checkpoints and removes or truncates the `-wal` and `-shm` files, leaving a single portable `supervisor.db`.

---

## 4. Architecture-Level Backup Policy

### 4.1 Strict Invariant on Active Backups
**DO NOT backup an active WAL database by naively copying only `supervisor.db`.**
- *Failure Mode*: Naive filesystem copying of `supervisor.db` while the database is open will capture an inconsistent snapshot missing all pages residing in `supervisor.db-wal`.

### 4.2 Approved Backup Mechanisms
Backups must use one of the following canonical SQLite mechanisms:
1. **SQLite Online Backup API** (`sqlite3_backup`):
   - Atomically streams database pages to a target file while the source database remains live and accessible.
2. **`VACUUM INTO 'backup_path.db';`**:
   - Single atomic SQL command that writes a fully checkpointed, consistent snapshot into a new standalone database file.
3. **Controlled Checkpoint Snapshot**:
   - Execute `PRAGMA wal_checkpoint(TRUNCATE);` during an idle window, then perform filesystem snapshot.

---

## 5. Schema Migration & Integrity

1. **Schema Versioning**: Track schema evolution via `PRAGMA user_version;`.
2. **Integrity Check**: On daemon startup, execute `PRAGMA integrity_check;` and `PRAGMA foreign_key_check;` as part of the `OPS-003` self-contained storage initialization.
3. **Foreign Key Enforcement**: `PRAGMA foreign_keys = ON;` must be executed for every opened connection to preserve relational integrity across tasks, contracts, attempts, and audit logs.
