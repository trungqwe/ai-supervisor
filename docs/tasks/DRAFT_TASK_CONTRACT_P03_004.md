# DRAFT TASK CONTRACT: TASK-P03-004

> **Contract ID**: `DRAFT-CONTRACT-TASK-P03-004-01`
> **Task ID**: `TASK-P03-004` (Host Quiescence & Daemon Bootstrap Integration)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P03`
> **Base SHA**: `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`
> **Status**: `DRAFT_NOT_RELEASED`
> **Authority**: Formulated pursuant to proposed `DRAFT-ADR-017` (Revision 5), `PROPOSAL-P03-005` (Revision 5), and `PLAN-HOST-INTEGRATION-DEPENDENCY.md` (Revision 7).

---

> [!CRITICAL]
> **GOVERNANCE STATUS: DRAFT ONLY / NOT_RELEASED**.
> This document is a **draft candidate contract** prepared for External Supervisor audit.
> It does **NOT** authorize production code execution.
> Production coding remains strictly **`HELD_PENDING_TASK_CONTRACT_RELEASE`** (`TASK_P03_004 = NOT_RELEASED`).
> Do **NOT** modify canonical roadmap or accepted ADRs as if accepted.
> Tách bạch hoàn toàn kiểm chứng: Mock AO test chạy tự động trong CI/harness không được suy diễn thành bằng chứng host principal thật.

## 1. Authoritative Canonical TaskContract JSON Object

```json
{
  "contract_id": "CONTRACT-TASK-P03-004-01",
  "task_id": "TASK-P03-004",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P03",
  "objective": "Implement host bootstrap daemon, Windows machine-wide exclusivity via exclusive sidecar lock file handle, startup-before-serve admission control, DB post-open identity validation, graceful shutdown drain, and P03 integration test harness verifying 5 AO exit gate steps without Phase P04/P05 dependencies.",
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
    "docs/adr/DRAFT-ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md",
    "docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md",
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
    "test/integration/**"
  ],
  "forbidden_scope": [
    "internal/ao/**",
    "internal/dispatch/**",
    "internal/domain/**",
    "internal/recovery/**",
    "internal/stop/**",
    "internal/store/**",
    "internal/evidence/**",
    "internal/review/**",
    "internal/tools/**",
    "docs/adr/**",
    "docs/proposals/**",
    "docs/02_REQUIREMENTS.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/06_WORKFLOW_STATE_MACHINE.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/11_CHATGPT_TOOL_SURFACE.md",
    "docs/12_UPSTREAM_INTEGRATION.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/17_ROADMAP.md",
    "docs/18_CURRENT_STATE.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md",
    "docs/phases/**",
    "AGENTS.md"
  ],
  "constraints": [
    "Exclusive sidecar lock handle contract on Windows: CreateFileW on <canonical_db_path>.owner.lock with GENERIC_READ|GENERIC_WRITE, dwShareMode=0, OPEN_ALWAYS, FILE_ATTRIBUTE_NORMAL, bInheritHandle=FALSE",
    "Canonical DB path via GetFinalPathNameByHandleW(VOLUME_NAME_DOS); hard links (nNumberOfLinks > 1) rejected fail-closed with ERR_HARDLINK_ALIAS_UNSUPPORTED",
    "Post-Store.Open identity validation: verify VolumeSerialNumber and FileId of opened DB matches pre-open canonical key; mismatch triggers immediate Store.Close() and fail-closed",
    "Shutdown order: close Named Pipe listener -> drain callers -> Store.Close() -> remove .owner.json ONLY IF owner_instance_id matches -> CloseHandle(.owner.lock) LAST",
    "Named Pipe security: authenticate caller SID via ImpersonateNamedPipeClient and token check, verify return errors, revert context via RevertToSelf(); owner_instance_id is strictly a freshness marker",
    "Binary readiness probe on cmd/supervisor proves socket refused before Run, and 200 OK only after Run Complete; zero Pair HTTP routes or effectful Supervisor API on daemon listener",
    "Pair hold and 5 AO exit gate steps proven via P03 Integration Test Harness directly invoking approved library APIs without P04/P05 dependencies",
    "All 8 operational policies remain UNSET in documentation, failing closed at runtime if missing; AUTOMATIC_RESTORE remains DISABLED"
  ],
  "acceptance_criteria": [
    "AC-004-01: cmd/supervisor implements Windows exclusive sidecar lock file handle (.owner.lock) with share mode 0, failing closed on contention",
    "AC-004-02: Path lock key derivation is strictly distinguished from physical file ID; aliases (subst, junctions, casing) are only admitted upon runtime probe proof of convergence to identical canonical lock key, otherwise fail-closed without unconditional support promises",
    "AC-004-03: Host holds pinned DB handle (no FILE_SHARE_DELETE) preventing file substitution during entire Store lifecycle; converts validated Win32 DOS path to Store DBPath while UNC/device paths fail-closed; invokes store.Open(ctx, store.Config{DBPath, BusyTimeoutMs}) without inspecting private DB pragmas; physical identity validations compare VolumeSerialNumber and 128-bit FileId between OS handles and parent volume, closing Store and failing closed on any mismatch",
    "AC-004-04: Shutdown drain sequence closes pipe listener, drains callers, closes Store, cleans metadata if instance matches, and closes lock handle LAST",
    "AC-004-05: Named Pipe takeover verifies caller SID via Impersonation and token check, reverts context via RevertToSelf(), and treats owner_instance_id as freshness marker",
    "AC-004-06: Binary readiness probe proves port closed before Run and open after Complete; Pair hold proven via Store/admission guards in harness",
    "AC-004-07: P03 Integration Test Harness verifies 5 AO exit gate steps (session create, dispatch, observation reconciliation, raw file read, teardown) via internal library APIs",
    "AC-004-08: Zero effectful HTTP routes on daemon; 8 policies remain UNSET and fail closed; AUTOMATIC_RESTORE remains DISABLED; test suite passes with -race"
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
    "ao_integration_harness_pass_evidence"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Attempting to modify files outside allowed_scope or touching forbidden_scope",
    "Windows filesystem fails to support mandatory exclusive file locking or GetFinalPathNameByHandleW",
    "Hard link alias detected on database file (nNumberOfLinks > 1)",
    "Post-Store.Open physical DB file identity does not match pre-open pinned OS handle identity or parent volume",
    "Attempting to introduce effectful HTTP routes, arbitrary shell execution, or automatic restore into P03",
    "Operational policies assigned default fallback values instead of failing closed when UNSET"
  ]
}
```
