# EXTERNAL SUPERVISOR APPROVAL DECISION: ADR-017 & PROPOSAL-P03-005

**Audit Date**: 2026-09-24
**Auditor**: External Supervisor (Independent Architecture & Security Review)
**Audited SHA**: `583f0db545fcf2a902305f811c22d9dd116e9728`
**Verdict**: `ADR_017 = EXTERNAL_APPROVED`, `ADR_017_ACCEPTANCE = GRANTED`, `PROPOSAL_P03_005 = EXTERNAL_APPROVED`.
**Subtask Authorization**: Subtask `TASK-P03-004` is formally incorporated into Phase P03.
**Contract Status**: `TASK_P03_004 = NOT_RELEASED` (Candidate prepared for release audit).
**Code Status**: `P03_CODE = HELD_PENDING_TASK_P03_004_CONTRACT_RELEASE`.

---

## 1. Approval Scope & Mandatory Textual Errata

1. **Approved Design**:
   - Machine-wide exclusivity on Windows via single sidecar lock handle (`<canonical_db_path>.owner.lock`, share mode 0, `OPEN_ALWAYS`, no inherit).
   - Host-scope pinned DB handle `hPinnedDB` (`CreateFileW` with `GENERIC_READ`, `FILE_SHARE_READ | FILE_SHARE_WRITE`, strictly omitting `FILE_SHARE_DELETE`) held continuously from pre-open to after `Store.Close()`, kernel-blocking file replacement or rename.
   - Two distinct DB initialization paths (Existing DB vs Brand New DB with exclusive-create `CREATE_NEW` and pre-Store.Open pinning).
   - Store integration boundary: host validates local DOS volume path and strips `\\?\` before invoking real `store.Open(ctx, store.Config{DBPath, BusyTimeoutMs})`; UNC and device paths fail-closed.
   - Four Invariants replacing private `PRAGMA database_list` inspection.
   - Evaluation profile `go-test-p03-004` with package enum and `const: ["-v", "-race", "-count=1"]` flags.
2. **Mandatory Textual Errata Applied**:
   - Any residual language comparing a physical file ID against a path-based lock key string is corrected.
   - Physical identity (`VolumeSerialNumber` + 128-bit `FileId`) is compared strictly between OS handles (`hPinnedDB` vs pre-open handle or parent directory handle).
   - The sidecar lock key is recognized strictly as a canonical normalized absolute path token.
3. **No Additional Design Audits**:
   - The design is formally closed. No additional design audit cycles are permitted unless Section 2.3 algorithms are fundamentally modified.

---

## 2. Policy Catalog Decision: `go-test-p03-004`

Registered catalog profile policy with const flags:
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

---

## 3. Governance Status Following Decision

- `ADR_017 = EXTERNAL_APPROVED`
- `ADR_017_ACCEPTANCE = GRANTED`
- `PROPOSAL_P03_005 = EXTERNAL_APPROVED`
- `TASK_P03_004 = CANDIDATE_PREPARED_NOT_RELEASED`
- `P03_CODE = HELD_PENDING_TASK_P03_004_CONTRACT_RELEASE`
- `ACTIVE_GATE = TASK_P03_004_RELEASE_AUDIT`
