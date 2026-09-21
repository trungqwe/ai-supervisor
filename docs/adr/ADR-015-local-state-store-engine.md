# ADR-015: Local State Store Engine and Durability Specification

> **Status**: PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Deciders**: External Supervisor, Engineering Team
> **Consulted**: `docs/decisions/P02_Q4_STATE_STORE_RESEARCH.md`
> **Replaces**: Initial draft of ADR-015 (Corrected durability policy from `NORMAL` to `FULL`)

---

## 1. Context and Problem Statement

The Supervisor Control Plane requires a local, zero-cloud data store (OPS-003) to persist:
1. Multi-project registrations and active pair bindings (FR-001, FR-002, FR-014);
2. Immutable `TaskContract` revisions and `TaskAttempt` lineage (FR-004, ADR-012);
3. The 13-state attempt workflow (FR-011);
4. Append-only audit events (FR-013, NFR-004).

Crucially, the pre-dispatch invariant dictates that the transition `READY → DISPATCHED` and `TaskAttempt` allocation must be durably committed before any external Agent Orchestrator invocation occurs.

---

## 2. Engine Selection

The storage engine direction is confirmed:
```
STATE_STORE_ENGINE = SQLITE
```
SQLite provides relational integrity, foreign key cascading/enforcement, zero-configuration local execution, and crash-resilient ACID transactions.

---

## 3. Canonical V1 Durability & Concurrency Policy

To guarantee durability across hard OS crashes and power interruptions, all database connections must be initialized with the following pragmas:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = FULL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

### Rationale:
- **WAL Mode**: Enables concurrent read operations without blocking or being blocked by the single active writer.
- **`synchronous = FULL`**: Guarantees that every transaction commit flushes the write-ahead log to physical media before returning. This prevents the loss of `READY → DISPATCHED` commitments in the event of an OS crash or power failure.
- **`foreign_keys = ON`**: Enforces relational constraints between contracts, attempts, and claims.
- **`busy_timeout = 5000`**: Prevents immediate lock errors during brief writer serialization.

---

## 4. Operational Invariants

1. **Short-Lived Transactions**: Database write transactions must be kept strictly below 50ms. No external network I/O, AO REST calls, or process executions may occur within a transaction.
2. **Runtime Files**: The runtime database comprises `supervisor.db`, `supervisor.db-wal`, and `supervisor.db-shm`.
3. **Backup Requirement**: Naive copying of `supervisor.db` while WAL is active is strictly prohibited. Backups must be performed via the SQLite Online Backup API or `VACUUM INTO 'backup.db'`.

---

## 5. Status

This ADR is **PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)**.
