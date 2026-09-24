# EXTERNAL SUPERVISOR RE-AUDIT 005: ADR-017 REVISION 5 & DRAFT TASK CONTRACT 004

**Audit Date**: 2026-09-24
**Auditor**: External Supervisor (Independent Architecture & Security Review)
**Target Subject**: DRAFT-ADR-017 (Revision 5), PROPOSAL-P03-005 (Revision 5), PLAN-HOST-INTEGRATION-DEPENDENCY (Revision 7), DRAFT_TASK_CONTRACT_P03_004 at commit `31d1baf96a2b21c220f225fc304ee8dc431b6865`
**Verdict**: REVISION_6_REQUIRED (Design & Governance Iteration); ADR-017 NOT_ACCEPTED; Task Contract 004 NOT_RELEASED.

---

## 1. Finding Disposition Summary

| Finding ID | Title | Status in Re-Audit 004 | Status in Re-Audit 005 | Resolution / Governance Record |
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
| **ADR17-R3-002** | CreateFileW Nonzero Desired Access | CLOSED (R4) | CLOSED | GENERIC_READ\|GENERIC_WRITE, share mode 0, OPEN_ALWAYS. |
| **ADR17-R3-003** | Pre/Post Store.Open File Identity | CLOSED (R4) | CLOSED | Pre/post volume + file ID check against canonical key. |
| **ADR17-R3-004** | Hard Link Detection and Rejection | CLOSED (R4) | CLOSED | nNumberOfLinks > 1 rejected fail-closed. |
| **ADR17-R4-001** | Named Pipe Server Ownership & Security | CLOSED (R5) | CLOSED | ImpersonateNamedPipeClient + GetTokenInformation + RevertToSelf. |
| **ADR17-R4-002** | Task-004 Scope Classification in P03 | CLOSED (R5) | CLOSED | Subtask of P03 approved direction. |
| **ADR17-R4-003** | Proof of DB Identity vs Lock Key | PARTIALLY_CLOSED (R5) | **CLOSED** (R6) | Host-scope pinned DB handle (no FILE_SHARE_DELETE) blocks file replacement; pre/post identity checks; zero changes to internal/store. |
| **ADR17-R4-004** | Integration Harness API Boundary | CLOSED (R5) | CLOSED | Pure library harness; no production HTTP control. |
| **ADR17-R4-005** | Verification Profile Metadata Governance | PARTIALLY_CLOSED (R5) | **CLOSED** (R6) | Dedicated profile go-test-p03-004 catalog registered; package enum restricted; exact flags [-v, -race, -count=1]; positive & negative validation verified. |
| **ADR17-R5-001** | Elimination of 'SQLite Handle' Fallacy | NEW (R5) | **CLOSED** (R6) | PRAGMA database_list recognized as user-mode string only. Host pins DB handle via OS API preventing file swap during Store lifetime. |
| **ADR17-R5-002** | Alias Convergence vs Fail-Closed Policy | NEW (R5) | **CLOSED** (R6) | Subst/junctions/casing not promised unconditionally; admitted only if probe proves convergence to identical canonical key; otherwise fail-closed. |

---

## 2. Policy Catalog Decision: `go-test-p03-004`

Pursuant to ADR-013, the following verification profile policy is formally registered in the P03 evaluation catalog:

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
        "items": {
          "type": "string",
          "enum": ["-v", "-race", "-count=1"]
        },
        "minItems": 3,
        "maxItems": 3
      }
    },
    "required": ["package", "flags"],
    "additionalProperties": false
  },
  "cwd_policy": "worktree_root",
  "max_timeout_seconds": 300
}
```

### Empirical Validation Results:
- **Positive Validation**: Contract `CONTRACT-TASK-P03-004-01` validated via `internal/contract.Validator` -> `PASS`.
- **Negative Test: Timeout 301s**: `requested timeout 301 exceeds profile maximum 300` -> `PASS (Rejected)`.
- **Negative Test: Flags containing `-exec`**: `enum: -exec does not equal any of: [-v -race -count=1]` -> `PASS (Rejected)`.
- **Negative Test: Package outside enum (`./internal/store/...`)**: `enum: ./internal/store/... does not equal any of [...]` -> `PASS (Rejected)`.
- **Negative Test: Cwd traversal (`../escape`)**: `parent traversal "../escape" escaping root is forbidden` -> `PASS (Rejected)`.

---

## 3. Live Windows Probe Evidence

1. **Pinned DB Handle Concurrency & Deletion Protection**:
   - `CreateFileW(GENERIC_READ, FILE_SHARE_READ | FILE_SHARE_WRITE, OPEN_EXISTING, no FILE_SHARE_DELETE)` held by host.
   - SQLite concurrent write: `INSERT INTO t VALUES (2, 'world')` -> **SUCCESS**.
   - Concurrently attempting `os.remove` or `os.rename` on DB file: `[WinError 32] The process cannot access the file because it is being used by another process` -> **SUCCESS (File cannot be swapped/deleted while Store is open)**.
2. **Path Normalization & Alias Probe**:
   - Casing probe: lowercase `caseddir\\file.txt` normalized to `CasedDir\\File.txt` -> **MATCH**.
   - Junction probe: `mklink /J junc real` -> `GetFinalPathNameByHandleW` resolves `junc\\data.db` to `real\\data.db` -> **MATCH**.
   - Subst probe: `subst X: real` -> `GetFinalPathNameByHandleW` resolves `X:\\data.db` to `real\\data.db` -> **MATCH**.
   - **Fail-Closed Principle**: If canonical resolution fails, returns device path, or cannot be proven, system fails closed.

---

## 4. Current Status

- `ADR_017 = REVISION_6_PENDING_EXTERNAL_AUDIT` (NOT_ACCEPTED).
- `TASK_P03_004 = NOT_RELEASED`.
- `P03_CODE = HELD_PENDING_TASK_P03_004_CONTRACT_RELEASE`.
- `ACTIVE_GATE = TASK_P03_004_CONTRACT_PLANNING`.
