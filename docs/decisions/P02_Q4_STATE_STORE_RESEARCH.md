# P02 Research Dossier: Q4 — Local State Store Storage Engine

> **Status**: RESEARCH_COMPLETE (PENDING_EXTERNAL_DECISION)
> **Date**: 2026-09-22
> **Related**: ADR-007 (Durable State Persistence), ADR-012 (TaskContract Revision & Attempt Binding), ADR-015 (Proposed Storage Decision)

---

## 1. Executive Summary & Objective
This research dossier evaluates local embedded storage engine candidates for the **Supervisor Control Plane State Store** (Phase P02+). The State Store persists project registration metadata, active pair session tokens, 13-state workflow transitions, immutable TaskContract revisions, multi-turn TaskAttempt lineages, worker claims, independent evidence packets, and review decisions.

A paramount new operational requirement established at the P02 pre-code gate is the **atomic pre-dispatch persistence invariant**:
```text
TaskAttempt allocation (attempt_id, attempt_number, contract_id, expected_report_path)
+
Task state transition (READY -> DISPATCHED)
```
must be committed atomically and durably to disk before any external side effects (AO session spawn or task send) are initiated.

Candidates evaluated:
1. **SQLite (WAL Mode)**
2. **Durable Atomic JSON / File Store**
3. **Embedded Key-Value Engine (LMDB / LevelDB)**

---

## 2. Technical & Governance Requirements
- **Crash Recovery & ACID Durability (NFR-003)**: Uncorrupted recovery after sudden power loss, OS reboot, or daemon crash.
- **Atomic Multi-Entity Mutation**: Ability to insert a new attempt record and update task state within a single atomic transaction.
- **Relational Lineage**: Clean representation of `Task 1 -> 1..* TaskContract` (revisions) and `Task 1 -> 1..* TaskAttempt` (execution iterations).
- **Windows Filesystem Behavior**: Safe concurrency and file handle semantics under Windows NTFS/ReFS, handling file locks cleanly.
- **Zero Cloud & Zero External Daemon Dependency (OPS-003)**: Entirely self-contained on the host machine without external database server processes.
- **Schema Evolution & Migration**: Clear versioning mechanism as domain models expand across phases.
- **Auditability**: Direct read-only inspection by human developers or diagnostic tools.

---

## 3. Candidate Deep Dive & Empirical Analysis

### Candidate 1: SQLite (in WAL Mode)
- **Transaction Model & Atomicity (Score: 10/10)**: Full ACID compliance. The atomic pre-dispatch persistence requirement is natively satisfied via a standard SQL transaction:
  ```sql
  BEGIN IMMEDIATE;
  INSERT INTO task_attempts (attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at)
    VALUES (?, ?, ?, ?, ?, ?);
  UPDATE tasks SET state = 'DISPATCHED', current_attempt = ? WHERE task_id = ?;
  COMMIT;
  ```
  If any crash occurs before `COMMIT`, the entire operation rolls back cleanly.
- **Crash Consistency & WAL (Score: 10/10)**: Configured with Write-Ahead Logging (`PRAGMA journal_mode = WAL; PRAGMA synchronous = NORMAL;`), writes append sequentially to a `-wal` file while reads continue concurrently without blocking writers. On crash, recovery is automatic during next connection open.
- **Windows Locking Behavior (Score: 8/10)**: In default rollback journal mode, Windows mandatory locking frequently triggers `busy` errors if multiple processes or threads touch the file. In **WAL mode**, readers and writers do not block each other, drastically mitigating Windows file-locking contention.
- **Queryability & Lineage (Score: 10/10)**: Foreign keys enforce relational integrity across `tasks`, `task_contracts`, `task_attempts`, `worker_claims`, `evidence`, and `review_decisions`. Complex governance queries (e.g., "fetch latest reviewable attempt for task X under contract revision Y") execute in sub-millisecond indexed queries.
- **Migrations & Backup (Score: 9/10)**: Standard migration version tables (`schema_migrations`). Online backup via `VACUUM INTO 'backup.db'` or native backup APIs. Single-file portability.

### Candidate 2: Durable Atomic JSON / File Store
- **Transaction Model & Atomicity (Score: 4/10)**: Atomically replacing a single file on Windows is possible via `ReplaceFileW` / `MoveFileExW(MOVEFILE_REPLACE_EXISTING)`. However, updating `tasks/<task_id>.json` AND writing `attempts/<attempt_id>.json` **cannot be made atomic across two files** without building a custom write-ahead transaction log in application code.
- If all state is consolidated into a single monolithic `state.json`, write amplification grows with every event, and concurrent reads/writes require global file-level mutexes.
- **Windows Locking Behavior (Score: 5/10)**: Replacing open files on Windows frequently encounters `EBUSY` / `EPERM` if Windows Defender, search indexers, or backup tools temporarily hold read handles.
- **Queryability & Lineage (Score: 5/10)**: Requires reading and parsing all JSON files into memory to filter by state, attempt, or contract revision.
- **Verdict**: Inadequate for atomic multi-entity dispatch transactions and relational lineage.

### Candidate 3: Embedded Key-Value Engine (e.g. LMDB / LevelDB)
- **Transaction Model & Atomicity (Score: 8/10)**: Key-value engines support atomic batch writes (`WriteBatch`), allowing multiple key mutations to commit atomically.
- **Queryability & Operational Complexity (Score: 5/10)**: Lacks native relational indexing. Querying attempts by task, finding latest revisions, or enforcing foreign keys requires manually designing, writing, and updating secondary index keys (e.g., `idx:task_attempts:<task_id>:<attempt_id>`). Schema evolution is manual and brittle.
- **Windows Ecosystem (Score: 6/10)**: C++ build requirements for LevelDB or RocksDB on Windows create significant integration friction across runtimes.
- **Verdict**: Adds unnecessary complexity without providing any meaningful benefit over SQLite.

---

## 4. Evaluation Matrix

| Criterion | Weight | SQLite (WAL) | Durable JSON Store | Embedded Key-Value |
|---|---|---|---|---|
| **ACID & Atomic Multi-Entity Transactions** | 25% | **10** (2.5) | 4 (1.0) | 8 (2.0) |
| **Crash Consistency & Recovery (NFR-003)** | 20% | **10** (2.0) | 5 (1.0) | 8 (1.6) |
| **Relational Lineage & Queryability** | 20% | **10** (2.0) | 5 (1.0) | 5 (1.0) |
| **Windows Locking & Filesystem Stability** | 15% | **8** (1.2) | 5 (0.75) | 7 (1.05) |
| **Schema Migration & Backup Simplicity** | 10% | **9** (0.9) | 5 (0.5) | 6 (0.6) |
| **Operational & Developer Simplicity** | 10% | **9** (0.9) | 7 (0.7) | 6 (0.6) |
| **TOTAL WEIGHTED SCORE** | **100%** | **9.50 / 10** | **4.95 / 10** | **6.85 / 10** |

---

## 5. Synthesis & Recommendation

### Recommended Storage Engine: **SQLite (in WAL Mode)**
- **Core Rationale**:
  1. The new atomic requirement (`TaskAttempt` allocation + `READY → DISPATCHED` committed before external AO invocation) requires true transactional atomicity. SQLite provides this out of the box with zero custom transaction code.
  2. Multi-turn revision and attempt governance (ADR-011, ADR-012) represents a relational graph: tasks have multiple contract revisions, attempts bind to contracts, and claims/evidence bind to attempts. Relational schema with foreign keys is the natural, safest representation.
  3. WAL mode (`PRAGMA journal_mode = WAL`) eliminates read/write contention and handles Windows process concurrency cleanly.
  4. Single-file database (`.supervisor/supervisor.db`) satisfies self-contained zero-cloud mandates (OPS-003).

### Runner-Up:
- There is no competitive runner-up for multi-entity transactional state. Simple JSON files remain useful exclusively for standalone point-in-time exported artifacts (such as `ReviewBundle` JSON files or append-only `audit.jsonl`), but cannot serve as the core transactional state store.

---

## 6. Conditions That Would Change the Recommendation
- A mandate for human-editable plaintext state files (though this would violate tamper-resistance and transactional atomicity invariants).