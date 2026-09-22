# P02 TASK-004 External Audit Record (Revision 2 Required)

## Audit Metadata

| Field | Value |
|---|---|
| **Audited Remote Commit** | `224ce840f06131706f3a693420484128fcd5348a` |
| **Logical Task Baseline** | `9fd5ac89fb6f781292658a6931e3d373c4860a6e` |
| **Audit Date** | 2026-09-22 |
| **Phase Gate** | `P02_TASK_004_REVISION_2` |

---

## Verdict

```
TASK_P02_003 = EXTERNAL_AUDIT_APPROVED
TASK_P02_004 = REVISION_REQUIRED
TASK_P02_004_REVISION = 2
P02_FINAL_AUDIT = NOT_RELEASED
ACTIVE_GATE = P02_TASK_004_REVISION_2
```

---

## Approved Portions

```
SQLITE_V1_TO_V2_MIGRATION = APPROVED
AUDIT_APPEND_TRANSACTION = APPROVED
AUDIT_HASH_ENCODING = APPROVED
AUDIT_CHAIN_STATE = APPROVED
AUDIT_LINEAGE_VALIDATION = APPROVED
AUDIT_APPEND_ONLY_TRIGGER_DEFENSE = APPROVED
CONCURRENT_APPEND = APPROVED
```

---

## Evidence Classification

```
REMOTE_SOURCE_ARTIFACT = COMPLETE
WORKER_CLEAN_WORKTREE_BUILD = PASS_REPORTED_WITH_EXPLICIT_GO_C
WORKER_CLEAN_WORKTREE_TEST = PASS_REPORTED_WITH_EXPLICIT_GO_C
WORKER_CLEAN_WORKTREE_VET = PASS_REPORTED_WITH_EXPLICIT_GO_C
GITHUB_REMOTE_CI = NOT_PRESENT
```

> [!WARNING]
> The previous `RAW_SECRET_DISK_SCAN` proves only the tested secret placements in values and actor strings.
> It does NOT prove secrets embedded in JSON OBJECT KEYS are scrubbed.

---

## Blocking Findings (Revision 2 Scope)

### P02T004-R2-001 — AUDIT_VERIFY_INCONSISTENT_SNAPSHOT
`VerifyAuditChain` performed two separate queries through `s.db` outside a shared transaction:
1. Read `audit_chain_state` head;
2. Then queried `audit_events` ordered by sequence.
A concurrent `AppendAuditEvent` committing between (1) and (2) causes an inconsistent view where `audit_events` contains more records than the previously read `audit_chain_state`, producing a false `ErrAuditChainInvalid` on an otherwise completely valid database.
**Required Fix**: Establish a single consistent database transaction / read snapshot (`BeginTx`) and execute all verification reads on the same transaction snapshot.

### P02T004-R2-002 — SECRET_IN_MAP_KEY_PERSISTENCE_BYPASS
`ScrubValue` in `internal/audit/scrubber.go` only sanitized map values, leaving arbitrary map key strings verbatim. Consequently, credentials placed as JSON object keys (e.g. `{"ghp_1234567890abcdef1234": "x"}`) bypass scrubbing and are persisted raw into `details_json`, SQLite tables, and WAL/SHM files.
**Required Fix**:
1. All arbitrary map keys must pass through `ScrubString`. Recognized structured keys (`password`, `api_key`, `session_token`, etc.) retain their key name while their values are unconditionally redacted.
2. If sanitizing keys creates a collision (two distinct input keys mapping to the same sanitized key), fail closed with a deterministic error. Do NOT silently overwrite entries.
3. Apply key sanitization recursively to nested maps.
4. Non-string map keys must fail closed deterministically (do not invent representations via `fmt.Sprintf`).
5. Prove via extended file-backed SQLite disk scan that raw secret bytes never survive in keys.
