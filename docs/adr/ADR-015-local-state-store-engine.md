# ADR-015: Local State Store Storage Engine

> **Status**: PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Proposal
> **Tracking**: `Q4_LOCAL_STATE_STORE_ENGINE`
> **Research Dossier**: `docs/decisions/P02_Q4_STATE_STORE_RESEARCH.md`

---

## Context
Phase P02 commences the implementation of the Supervisor Domain Core, requiring the selection of a local embedded storage engine to persist domain entities, 13-state workflow transitions, TaskContract revisions, and TaskAttempt lineages.

The P02 pre-code gate established a strict atomic persistence requirement:
`TaskAttempt` pre-allocation and `READY → DISPATCHED` transition must be durably committed within a single atomic transaction prior to invoking external AO execution side effects.

A comprehensive evaluation across SQLite, durable JSON/file store, and embedded key-value engines was documented in `docs/decisions/P02_Q4_STATE_STORE_RESEARCH.md`.

---

## Decision Proposal
It is proposed to adopt **SQLite in Write-Ahead Logging (WAL) mode** as the local embedded State Store for the Supervisor Control Plane.

Key drivers:
1. Native transactional ACID support (`BEGIN IMMEDIATE; ... COMMIT;`), guaranteeing that `TaskAttempt` allocation and `READY → DISPATCHED` commit atomically;
2. WAL journal mode (`PRAGMA journal_mode = WAL;`) allowing concurrent non-blocking reads while writing, avoiding Windows file locking contention;
3. Relational integrity with foreign keys enforcing `Task 1 -> 1..* TaskContract` (revisions) and `TaskAttempt -> 1 TaskContract`;
4. Self-contained single-file storage (`.supervisor/supervisor.db`) satisfying zero-cloud local execution requirements (OPS-003, NFR-003).

---

## Consequences
- **Positive**: Complete transactional crash safety and zero custom write-ahead logging code required in application logic.
- **Positive**: Sub-millisecond indexed queries across historical tasks, attempts, and audit references.
- **Negative / Trade-off**: Requires standard schema migration versioning tooling as domain models evolve across phases.

---

## Status
- **Current Status**: `PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)`.
- No application code or database migrations may be written until this proposal is formally approved by the External Supervisor.