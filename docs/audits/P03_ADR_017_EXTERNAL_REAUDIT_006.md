# EXTERNAL SUPERVISOR RE-AUDIT 006: ADR-017 REVISION 6/7 & DRAFT TASK CONTRACT 004

**Audit Date**: 2026-09-24
**Auditor**: External Supervisor (Independent Architecture & Security Review)
**Target Subject**: DRAFT-ADR-017 (Revision 7), PROPOSAL-P03-005 (Revision 7), PLAN-HOST-INTEGRATION-DEPENDENCY (Revision 9), DRAFT_TASK_CONTRACT_P03_004
**Verdict**: REVISION_7_PENDING_EXTERNAL_AUDIT; ADR-017 NOT_ACCEPTED; Task Contract 004 NOT_RELEASED.

---

## 1. Finding Disposition Summary

| Finding ID | Title | Status in Re-Audit 005 | Status in Re-Audit 006 | Resolution / Governance Record |
|---|---|---|---|---|
| **ADR17-R1-001** | Cross-Logon-Session Lock Exclusivity | CLOSED (R3) | CLOSED | Windows sidecar lock handle (CreateFileW, share mode 0). |
| **ADR17-R1-002** | Reconciled HostQuiescence Acquisition Order | CLOSED (R2) | CLOSED | Runner.Run owns HostQuiescence; host owns ProcessOwnerLease. |
| **ADR17-R1-003** | No Cooperative Takeover via Process Kill | CLOSED (R3) | CLOSED | Cooperative takeover only; fail-closed on refusal. |
| **ADR17-R1-004** | Startup Failure Mode Matrix | CLOSED (R2) | CLOSED | Fail-closed startup matrix without default fallbacks. |
| **ADR17-R1-005** | Production Evidence Verification Boundary | CLOSED (R3) | CLOSED | Dedicated P03 harness decoupled from P04/P05. |
| **ADR17-R1-006** | Task-004 Boundary Alignment | CLOSED (R2) | CLOSED | Subtask 004 within P03; exit gate harness only. |
| **ADR17-R2-001** | Production HTTP Surface Boundary | CLOSED (R3) | CLOSED | No effectful HTTP routes on P03 daemon. |
| **ADR17-R2-002** | Drain and Store.Close Ordering | CLOSED (R3) | CLOSED | Lock handle closed LAST after Store.Close. |
| **ADR17-R2-003** | Startup Scanner Effect Admission | CLOSED (R3) | CLOSED | Startup scanner does not mutate/replay effects. |
| **ADR17-R3-001** | Stale Metadata Cleanup Sequence | CLOSED (R4) | CLOSED | Lock handle held during cleanup; closed last. |
| **ADR17-R3-002** | CreateFileW Nonzero Desired Access | CLOSED (R4) | CLOSED | GENERIC_READ|GENERIC_WRITE, share mode 0, OPEN_ALWAYS. |
| **ADR17-R3-003** | Pre/Post Store.Open File Identity | CLOSED (R4) | CLOSED | Pre/post volume + file ID check against canonical key. |
| **ADR17-R3-004** | Hard Link Detection and Rejection | CLOSED (R4) | CLOSED | nNumberOfLinks > 1 rejected fail-closed. |
| **ADR17-R4-001** | Named Pipe Server Ownership & Security | CLOSED (R5) | CLOSED | ImpersonateNamedPipeClient + GetTokenInformation + RevertToSelf. |
| **ADR17-R4-002** | Task-004 Scope Classification in P03 | CLOSED (R5) | CLOSED | Subtask of P03 approved direction. |
| **ADR17-R4-003** | Proof of DB Identity vs Lock Key | CLOSED (R6) | CLOSED | Host-scope pinned DB handle (no FILE_SHARE_DELETE) blocks file replacement; pre/post identity checks; zero changes to internal/store. |
| **ADR17-R4-004** | Integration Harness API Boundary | CLOSED (R5) | CLOSED | Pure library harness; no production HTTP control. |
| **ADR17-R4-005** | Verification Profile Metadata Governance | CLOSED (R6) | CLOSED | Dedicated profile go-test-p03-004 catalog registered; package enum restricted; const flags [-v, -race, -count=1]. |
| **ADR17-R5-001** | Elimination of 'SQLite Handle' Fallacy & Pinned Handle Consistency | PARTIALLY_CLOSED (R5) | **CLOSED** (R7) | ADR-017 §2.3 synchronized with plan: Existing DB holds pinned handle across acquire, Store.Open and runtime; Brand New DB exclusive-create (CREATE_NEW) and pin before Store.Open. Erroneous 'close before acquire' and 'SQLite handle' statements fully eliminated. |
| **ADR17-R5-002** | Physical Handle Comparison & Strict Alias Fail-Closed | PARTIALLY_CLOSED (R5) | **CLOSED** (R7) | AC-004-03 compares physical handles (VolumeSerialNumber & FileId) between OS handles; AC-004-02 admits aliases only on live probe proof. Const flags [-v, -race, -count=1] validated. |

---

## 2. Policy Catalog Decision: `go-test-p03-004` (Const Flags)

Pursuant to ADR-013, the profile policy is updated to enforce `const` on flags:

```json
{
  "profile_id": "go-test-p03-004",
  "parameter_schema": {
    "type": "object",
    "properties": {
      "package": {
        "type": "string",
        "enum": [
          "./internal/host/...",
          "./cmd/supervisor/...",
          "./test/integration/..."
        ]
      },
      "flags": {
        "type": "array",
        "items": { "type": "string" },
        "const": ["-v", "-race", "-count=1"]
      }
    },
    "required": ["package", "flags"],
    "additionalProperties": false
  },
  "cwd_policy": "worktree_root",
  "max_timeout_seconds": 300
}
```

### Empirical Validation Suite:
- **Positive Validation**: Contract `CONTRACT-TASK-P03-004-01` -> `PASS`.
- **Negative: Duplicate flag (`["-v", "-v", "-race", "-count=1"]`)**: Rejected (`const: [...] does not equal [-v -race -count=1]`) -> `PASS`.
- **Negative: Wrong order (`["-race", "-v", "-count=1"]`)**: Rejected (`const: [...] does not equal [-v -race -count=1]`) -> `PASS`.
- **Negative: Flags containing `-exec`**: Rejected (`const: [...] does not equal [-v -race -count=1]`) -> `PASS`.
- **Negative: Package outside enum (`./internal/store/...`)**: Rejected (`enum: ./internal/store/... does not equal any of [...]`) -> `PASS`.
- **Negative: Timeout 301s**: Rejected (`requested timeout 301 exceeds profile maximum 300`) -> `PASS`.
- **Negative: Cwd traversal (`../escape`)**: Rejected (`parent traversal "../escape" escaping root is forbidden`) -> `PASS`.

*Governance Note*: This evaluation profile is strictly declarative metadata for Task Contract validation and does not enable or substitute for the Phase P04 verification runner.

---

## 3. Live Windows Probe Evidence: Brand New DB Exclusive-Create & Lifecycle

Empirical probe (`probe_new_db_exclusive_create.py`) executed live on Windows:
1. Canonical parent directory resolved via `GetFinalPathNameByHandleW(FILE_FLAG_BACKUP_SEMANTICS)`; parent volume recorded.
2. Sidecar lock `<canonical_db_path>.owner.lock` acquired with share mode 0.
3. Database file created via `CreateFileW(CREATE_NEW, GENERIC_READ|GENERIC_WRITE, FILE_SHARE_READ|FILE_SHARE_WRITE, no FILE_SHARE_DELETE)`.
4. Handle `hPinnedDB` final path matches `canonical_db_path`; volume matches parent volume; 128-bit `FileId` recorded; `nNumberOfLinks == 1`.
5. SQLite connects to 0-byte pinned file: schema migrations, table creation, transaction commit, transaction rollback all succeed.
6. Concurrent `os.remove` and `os.rename` blocked with `[WinError 32] ERROR_SHARING_VIOLATION`.
7. SQLite `conn.close()` completed; `hPinnedDB` closed; `h_lock` closed last.
8. If file already exists, `CREATE_NEW` fails with Win32 Error 80 (`ERROR_FILE_EXISTS`) -> **FAIL-CLOSED PROVEN**.

---

## 4. Current Status

- `ADR_017 = REVISION_7_PENDING_EXTERNAL_AUDIT` (NOT_ACCEPTED).
- `TASK_P03_004 = NOT_RELEASED`.
- `P03_CODE = HELD_PENDING_TASK_P03_004_CONTRACT_RELEASE`.
- `ACTIVE_GATE = TASK_P03_004_CONTRACT_PLANNING`.
