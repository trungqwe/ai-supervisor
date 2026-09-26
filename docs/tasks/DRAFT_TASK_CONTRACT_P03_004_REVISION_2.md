# DRAFT TASK CONTRACT: TASK-P03-004 (REVISION 2)

> **Contract ID**: `CONTRACT-TASK-P03-004-02`
> **Task ID**: `TASK-P03-004` (Host Quiescence & Daemon Bootstrap Integration)
> **Revision Number**: `2`
> **Supersedes Contract ID**: `CONTRACT-TASK-P03-004-01`
> **Phase ID**: `P03`
> **Base SHA**: `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`
> **Status**: `PROPOSED (Dự thảo phục vụ Change Governance theo PROPOSAL-P03-006)`
> **Authority**: Formulated pursuant to accepted `ADR-017` (§2.6, §4.1, §4.2), `ADR-016`, approved `PROPOSAL-P03-005`, and proposed `PROPOSAL-P03-006`.
> **Implementation Scope**: Authorized strictly within `allowed_scope` on branch `codex/p03-004` upon formal release.
> **Runtime Invariants**: `AUTOMATIC_RESTORE = DISABLED`; verified host principal remains `OPEN` dependency at trusted boundary; zero Phase P04/P05 dependencies.

---

## 1. Immutable TaskContract JSON (Revision 2)

```json
{
  "contract_id": "CONTRACT-TASK-P03-004-02",
  "task_id": "TASK-P03-004",
  "revision_number": 2,
  "supersedes_contract_id": "CONTRACT-TASK-P03-004-01",
  "phase_id": "P03",
  "objective": "Implement host bootstrap daemon, Windows machine-wide exclusivity via exclusive sidecar lock file handle, startup-before-serve admission control, graceful shutdown drain preserving lock until callers join, poller asynchronous error notification to host watcher, typed AO workspace file retrieval with JSON envelope validation and bounded reading, and P03 integration test harness verifying 5 AO exit gate steps without Phase P04/P05 dependencies.",
  "requirements": [
    "FR-004",
    "FR-005",
    "NFR-003",
    "NFR-004",
    "OPS-001",
    "SEC-001",
    "SEC-003"
  ],
  "architecture_refs": [
    "docs/adr/ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md",
    "docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md",
    "docs/proposals/PROPOSAL-P03-006-seam-scope-reconciliation-and-contract-revision.md",
    "docs/plans/PLAN-HOST-INTEGRATION-DEPENDENCY.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/11_CHATGPT_TOOL_SURFACE.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md"
  ],
  "base_sha": "35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea",
  "allowed_scope": [
    "cmd/supervisor/**",
    "internal/host/**",
    "internal/recovery/poller.go",
    "internal/recovery/poller_test.go",
    "internal/ao/client.go",
    "internal/ao/types.go",
    "internal/ao/client_test.go",
    "test/integration/**"
  ],
  "forbidden_scope": [
    "internal/recovery/runner.go",
    "internal/recovery/scanner.go",
    "internal/recovery/timeout_monitor.go",
    "internal/recovery/integration_test.go",
    "internal/recovery/timeout_monitor_test.go",
    "internal/recovery/scanner_test.go",
    "internal/ao/probes.go",
    "internal/ao/projects.go",
    "internal/ao/sessions.go",
    "internal/ao/session_commands.go",
    "internal/ao/session_commands_test.go",
    "internal/ao/wire_types.go",
    "internal/dispatch/**",
    "internal/domain/**",
    "internal/stop/**",
    "internal/store/**",
    "internal/evidence/**",
    "internal/review/**"
  ],
  "constraints": [
    "Exclusive sidecar lock handle contract on Windows: CreateFileW on <canonical_db_path>.owner.lock with GENERIC_READ|GENERIC_WRITE, dwShareMode=0, OPEN_ALWAYS, FILE_ATTRIBUTE_NORMAL, bInheritHandle=FALSE",
    "Canonical DB path via GetFinalPathNameByHandleW(VOLUME_NAME_DOS); hard links (nNumberOfLinks > 1) rejected fail-closed with ERR_HARDLINK_ALIAS_UNSUPPORTED",
    "Post-Store.Open identity validation: verify VolumeSerialNumber and FileId of opened DB matches pre-open pinned OS handle identity (and volume matches parent volume for new DB); mismatch triggers immediate Store.Close() and fail-closed",
    "Shutdown order: close Named Pipe listener -> drain callers -> Store.Close() -> remove .owner.json ONLY IF owner_instance_id matches -> CloseHandle(.owner.lock) LAST",
    "Named Pipe security: authenticate caller SID via ImpersonateNamedPipeClient and token check, verify return errors, revert context via RevertToSelf(); owner_instance_id is strictly a freshness marker",
    "Binary readiness probe on cmd/supervisor proves socket refused before Run, and 200 OK only after Run Complete; zero Pair HTTP routes or effectful Supervisor API on daemon listener",
    "Pair hold and 5 AO exit gate steps proven via P03 Integration Test Harness directly invoking approved library APIs without P04/P05 dependencies",
    "All 8 operational policies remain UNSET in documentation, failing closed at runtime if missing; AUTOMATIC_RESTORE remains DISABLED",
    "Poller error notification constraint: recovery.Poller must expose Done() <-chan struct{} fresh per Start and Err() error with synchronized access; single close on done avoiding double close; Err recorded before done close; clean Stop/context cancel yields nil Err; host watcher observes closed channel and marks admission unavailable fail-closed via SetUnavailable() without crashing daemon or releasing lock; watcher joined during host shutdown",
    "AO workspace read constraint: WorkspaceReadOptions must explicitly accept MaxWireBytes and MaxBytes independently (both >0) with overflow pre-check; no MaxBytes+64KB or implicit fallback; read at most MaxWireBytes+1 via LimitReader; validate presence and types of all required fields of WorkspaceFileResponse preventing zero-value masking; reject binary, deleted, contentTruncated, and trailing payload fail-closed; HTTP 200 with deleted=true returns ErrWorkspaceFileDeleted without fabricating 404 APIError; check content byte length <= MaxBytes with ErrPayloadTooLarge; preserve TransportError/APIError/ProtocolError without domain WorkerReport/TaskState parsing"
  ],
  "acceptance_criteria": [
    "AC-004-01: cmd/supervisor implements Windows exclusive sidecar lock file handle (.owner.lock) with share mode 0, failing closed on contention",
    "AC-004-02: Path lock key derivation is strictly distinguished from physical file ID; aliases (subst, junctions, casing) are only admitted upon runtime probe proof of convergence to identical canonical lock key, otherwise fail-closed without unconditional support promises",
    "AC-004-03: Host holds pinned DB handle (no FILE_SHARE_DELETE) preventing file substitution during entire Store lifecycle; converts validated Win32 DOS path to Store DBPath while UNC/device paths fail-closed; invokes store.Open(ctx, store.Config{DBPath, BusyTimeoutMs}) without inspecting private DB pragmas; physical identity validations compare VolumeSerialNumber and 128-bit FileId between OS handles and parent volume, closing Store and failing closed on any mismatch",
    "AC-004-04: Shutdown drain sequence closes pipe listener, drains callers, closes Store, cleans metadata if instance matches, and closes lock handle LAST",
    "AC-004-05: Named Pipe takeover verifies caller SID via ImpersonateNamedPipeClient and token check, reverts context via RevertToSelf(), and treats owner_instance_id as freshness marker",
    "AC-004-06: Binary readiness probe proves port closed before Run and open after Complete; Pair hold proven via Store/admission guards in harness",
    "AC-004-07: P03 Integration Test Harness verifies 5 AO exit gate steps (session create, dispatch, observation reconciliation, raw file read, teardown) via internal library APIs",
    "AC-004-08: Zero effectful HTTP routes on daemon; 8 policies remain UNSET and fail closed; AUTOMATIC_RESTORE remains DISABLED; test suite passes with -race",
    "AC-004-09: recovery.Poller exports Done() <-chan struct{} (fresh per Start) and Err() error with synchronized lifecycle and single close; host watcher joins on shutdown, distinguishes clean cancellation (Err==nil) from background failure (Err!=nil), transitions /readyz to HTTP 503 and marks admission unavailable via SetUnavailable() without process crash or lock release; regression tests cover real PollOnce error, clean stop, cancel, restart, and race",
    "AC-004-10: ao.Client provides GetWorkspaceFile accepting independent MaxWireBytes and MaxBytes (>0 with overflow check); reads at most MaxWireBytes+1; decodes WorkspaceFileResponse JSON from pinned AO (15e9ea971f1711ec8b50e157d6eb300db6cbe0d6) validating presence and types of all required fields; rejects trailing data, binary, contentTruncated, and deleted files (HTTP 200 with deleted=true returns ErrWorkspaceFileDeleted); verifies content byte length <= MaxBytes with ErrPayloadTooLarge; preserves TransportError/APIError/ProtocolError without domain WorkerReport/TaskState parsing; mock harness tests large envelope with small content, wire overflow, content overflow, exact boundary, and negative cases"
  ],
  "verification_requests": [
    {
      "id": "VR-P03-004-HOST-TESTS",
      "profile_id": "go-test-p03-004",
      "parameters": {
        "package": "./internal/host/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    },
    {
      "id": "VR-P03-004-CMD-TESTS",
      "profile_id": "go-test-p03-004",
      "parameters": {
        "package": "./cmd/supervisor/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    },
    {
      "id": "VR-P03-004-INTEGRATION-TESTS",
      "profile_id": "go-test-p03-004",
      "parameters": {
        "package": "./test/integration/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    },
    {
      "id": "VR-P03-004-RECOVERY-POLLER-TESTS",
      "profile_id": "go-test-p03-004",
      "parameters": {
        "package": "./internal/recovery/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    },
    {
      "id": "VR-P03-004-AO-CLIENT-TESTS",
      "profile_id": "go-test-p03-004",
      "parameters": {
        "package": "./internal/ao/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      },
      "cwd": ".",
      "timeout_seconds": 300
    }
  ],
  "required_evidence": [
    "git_diff",
    "git_diff_check",
    "test_exit_code_zero",
    "race_detector_zero_warnings",
    "windows_cross_session_lock_evidence",
    "startup_before_serve_probe_evidence",
    "shutdown_drain_cleanup_evidence",
    "db_identity_post_open_match_evidence",
    "pair_hold_admission_guard_evidence",
    "ao_integration_harness_pass_evidence",
    "poller_async_error_notification_evidence",
    "ao_workspace_file_typed_client_evidence"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Attempting to modify files outside allowed_scope or touching forbidden_scope",
    "Windows filesystem fails to support mandatory exclusive file locking or GetFinalPathNameByHandleW",
    "Hard link alias detected on database file (nNumberOfLinks > 1)",
    "Post-Store.Open physical DB file identity does not match pre-open pinned OS handle identity or parent volume",
    "Attempting to introduce effectful HTTP routes, arbitrary shell execution, or automatic restore into P03",
    "Operational policies assigned default fallback values instead of failing closed when UNSET",
    "Modifying recovery or ao files outside the specifically allowed files (poller.go, poller_test.go, client.go, types.go, client_test.go)"
  ]
}
```

---

## 2. Revision Delta (Contract 01 vs Revision 2 Draft/Candidate)

| Field / Section | Contract 01 (Baseline) | Revision 2 (Proposed) | Governance Rationale |
|---|---|---|---|
| `contract_id` | `CONTRACT-TASK-P03-004-01` | `CONTRACT-TASK-P03-004-02` | Monotonic revision identifier |
| `revision_number` | `1` | `2` | Linear revision progression |
| `supersedes_contract_id` | `null` | `CONTRACT-TASK-P03-004-01` | Explicit supersedes lineage link |
| `allowed_scope` | `cmd/supervisor/**`, `internal/host/**`, `test/integration/**` | + `internal/recovery/poller.go`, `internal/recovery/poller_test.go`, `internal/ao/client.go`, `internal/ao/types.go`, `internal/ao/client_test.go` | Narrowest whitelist expansion to resolve Seams 1 & 2 per PROPOSAL-P03-006 |
| `forbidden_scope` | Broad subsystem globs | Fine-grained file-level exclusions for all non-whitelisted files in `recovery` and `ao` | Enforces Task Scope Immutability on untouched files |
| `constraints` | 8 baseline constraints | 8 baseline + 2 new (Poller synchronized single-close notification & AO JSON envelope validation with independent MaxWireBytes and MaxBytes) | Formalizes asynchronous signal & bounded read guards |
| `acceptance_criteria` | `AC-004-01` .. `AC-004-08` | `AC-004-01` .. `AC-004-08` + `AC-004-09` + `AC-004-10` | Distinct IDs for new verifiable acceptance criteria |
| `verification_requests` | 3 requests (host, cmd, integration) | 3 baseline + 2 new (`VR-P03-004-RECOVERY-POLLER-TESTS`, `VR-P03-004-AO-CLIENT-TESTS`) | Targets specific unit tests for modified recovery and ao packages |
| `required_evidence` | 10 baseline evidence items | 10 baseline + 2 new (`poller_async_error_notification_evidence`, `ao_workspace_file_typed_client_evidence`) | Verifiable claims required in worker report |
| `stop_conditions` | 6 baseline stop conditions | 6 baseline + 1 new (Prohibits modifying recovery/ao outside 5 specific files) | Fail-closed guard against unintended scope creep |

---

## 3. Proposed Verification Profile Catalog Delta (`go-test-p03-004`)

> [!NOTE]
> The evaluation policy catalog registered in `docs/audits/P03_ADR_017_EXTERNAL_REAUDIT_005.md` defined `package` enum with 3 paths.
> The following catalog delta expanding the enum to 5 packages is **APPROVED_AT_DESIGN_LEVEL**; Stage B host runtime remains **UNVERIFIED** until task dispatch:

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
          "./test/integration/...",
          "./internal/recovery/...",
          "./internal/ao/..."
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

## 4. Pre-Release Invariants & Guardrails

1. **Pre-Release Code Hold**:
   - No modifications to `internal/recovery/**` or `internal/ao/**` may be committed on `codex/p03-004` until this candidate contract is formally reviewed, approved, and released by External Supervisor.
2. **Deterministic Validation**:
   - The JSON object in Section 1 passes full schema validation against `docs/schemas/task-contract.schema.json` and semantic validation against `internal/contract/validator.go`.
3. **Runtime Invariants**:
   - `AUTOMATIC_RESTORE = DISABLED`.
   - Verified host principal remains `OPEN` dependency at trusted boundary.
