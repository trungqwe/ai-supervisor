# EXTERNAL SUPERVISOR RE-AUDIT 007: ADR-017 REVISION 8 & TASK CONTRACT 004 INTEGRATION BOUNDARY

**Audit Date**: 2026-09-24
**Auditor**: External Supervisor (Independent Architecture & Security Review)
**Target Subject**: DRAFT-ADR-017 (Revision 8), PROPOSAL-P03-005 (Revision 8), PLAN-HOST-INTEGRATION-DEPENDENCY (Revision 10), DRAFT_TASK_CONTRACT_P03_004
**Verdict**: REVISION_8_PENDING_EXTERNAL_AUDIT; ADR-017 NOT_ACCEPTED; Task Contract 004 NOT_RELEASED.

---

## 1. Finding Disposition & Integration Boundary Reconciliation

| Finding ID | Title | Status in Re-Audit 006 | Status in Re-Audit 007 | Resolution / Governance Record |
|---|---|---|---|---|
| **ADR17-R1-001..006** | Lifecycle, Quiescence, Exclusivity, Scope | CLOSED | CLOSED | Win32 sidecar lock (share mode 0); host owns lease; runner owns quiescence. |
| **ADR17-R2-001..003** | Production Boundary & Shutdown Order | CLOSED | CLOSED | No effectful HTTP routes; lock closed last; non-mutating startup scan. |
| **ADR17-R3-001..004** | Handle Contracts & Cleanup | CLOSED | CLOSED | Stale metadata cleanup under lock; GENERIC_READ|WRITE; hardlinks rejected. |
| **ADR17-R4-001..005** | Named Pipe Security & Profile Catalog | CLOSED | CLOSED | ImpersonateNamedPipeClient + token SID verification; go-test-p03-004 catalog with const flags [-v, -race, -count=1]. |
| **ADR17-R5-001** | Elimination of 'SQLite Handle' Fallacy & Pinned Handle Consistency | CLOSED (R7) | **CLOSED** (R8) | Win32 canonical path/lock key separated from Store DBPath. Local DOS path conversion (\\?\<Drive>:\ -> <Drive>:\) enforced. Real store.Open API used without private DB inspection. |
| **ADR17-R5-002** | Physical Handle Comparison & Strict Alias Fail-Closed | CLOSED (R7) | **CLOSED** (R8) | AC-004-03 compares VolumeSerialNumber and FileId between OS handles; stop condition updated; alias fail-closed verified. |

---

## 2. Live Windows Integration Probe Results (`TestWin32StoreIntegrationProbe`)

Ran live integration probe against real `internal/store` package and `modernc.org/sqlite` on Windows:

1. **Branch 1: Raw Win32 Path Rejection (`RawWin32Path_Rejected`)**:
   - Input path: `\\?\C:\Users\...\raw_win32.db`
   - Store.Open output: `store: failed to ping database: SQL logic error: invalid uri authority: ? (1)`
   - Verdict: **CONFIRMED & REJECTED AS REQUIRED**. Demonstrates necessity of converting local DOS path before passing to `store.Open`.
2. **Branch 2: Brand New DB Exclusive-Create & Lifecycle (`NewDB_ExclusiveCreate_Pinned_StoreOpen`)**:
   - Parent directory canonicalized; sidecar lock acquired.
   - Exclusive-create file via Win32 `CREATE_NEW` (no `FILE_SHARE_DELETE`) -> 0-byte file created.
   - Canonical Win32 path converted to validated DOS path `C:\...`.
   - `store.Open(ctx, store.Config{DBPath, BusyTimeoutMs})` executed successfully: SQLite initialized 0-byte file, verified WAL mode and pragmas, executed migrations up to v5.
   - Read/write verified via `store.CreateProject` and `store.GetProject`.
   - Concurrent `os.Remove` blocked by Windows with `The process cannot access the file because it is being used by another process` (sharing violation).
   - `Store.Close()` succeeded; `hPinnedDB` closed; lock handle closed last.
   - Verdict: **CONFIRMED & PASSED 100%**.
3. **Branch 3: Existing DB Pinned Handle & Lifecycle (`ExistingDB_Pinned_StoreOpen`)**:
   - Pinned handle `hPinnedDB` opened on existing DB file before Store initialization.
   - Sidecar lock acquired.
   - `store.Open(ctx, store.Config{DBPath, BusyTimeoutMs})` with DOS path succeeded.
   - Read/write verified on existing DB schema.
   - `Store.Close()` succeeded; `hPinnedDB` closed; lock handle closed last.
   - Verdict: **CONFIRMED & PASSED 100%**.

---

## 3. Four Invariants Governing Store State Integrity

Host does not access `*sql.DB` private field of Store; file integrity is guaranteed by:
1. **Validated Path**: DOS DBPath is derived 1-to-1 from canonical Win32 volume path.
2. **Pinned Handle**: Host holds `hPinnedDB` (omitting `FILE_SHARE_DELETE`) continuously from before Store.Open to after Store.Close, preventing any file replacement.
3. **Physical Identity Match**: `VolumeSerialNumber` and 128-bit `FileId` are compared directly between OS handles.
4. **Store Self-Validation**: `store.Open` validates connection, verifies PRAGMAs (WAL, FULL, foreign keys, busy timeout), and executes migrations up to v5.

---

## 4. Current Status

- `ADR_017 = REVISION_8_PENDING_EXTERNAL_AUDIT` (NOT_ACCEPTED).
- `TASK_P03_004 = NOT_RELEASED`.
- `P03_CODE = HELD_PENDING_TASK_P03_004_CONTRACT_RELEASE`.
- `ACTIVE_GATE = TASK_P03_004_CONTRACT_PLANNING`.
