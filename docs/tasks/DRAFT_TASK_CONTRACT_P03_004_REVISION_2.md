# DRAFT TASK CONTRACT: TASK-P03-004 (REVISION 2)

> **Contract ID**: `CONTRACT-TASK-P03-004-02`
> **Task ID**: `TASK-P03-004` (Host Quiescence & Daemon Bootstrap Integration)
> **Revision Number**: `2`
> **Supersedes Contract ID**: `CONTRACT-TASK-P03-004-01`
> **Phase ID**: `P03`
> **Base SHA**: `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`
> **Status**: `PROPOSED` (Du thao phuc vu Change Governance theo PROPOSAL-P03-006)
> **Authority**: Formulated pursuant to accepted `ADR-017`, `ADR-016`, approved `PROPOSAL-P03-005`, and proposed `PROPOSAL-P03-006`.
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
  "objective": "Implement host bootstrap daemon, Windows machine-wide exclusivity via exclusive sidecar lock file handle, startup-before-serve admission control, graceful shutdown drain preserving lock until callers join, poller asynchronous error notification to host watcher, typed AO workspace file retrieval with bounded reading and cancellation, and P03 integration test harness verifying 5 AO exit gate steps without Phase P04/P05 dependencies.",
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
    "docs/adr/ADR-016-lifecycle-reconciliation-and-session-binding.md",
    "docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md",
    "docs/proposals/PROPOSAL-P03-006-seam-scope-reconciliation-and-contract-revision.md",
    "docs/plans/PLAN-HOST-INTEGRATION-DEPENDENCY.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/08_TASK_CONTRACT.md",
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
    "internal/dispatch/**",
    "internal/domain/**",
    "internal/stop/**",
    "internal/store/**",
    "internal/evidence/**",
    "internal/review/**",
    "internal/tools/**",
    "docs/adr/**",
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
    "Post-Store.Open identity validation: verify VolumeSerialNumber and FileId of opened DB matches pre-open pinned OS handle identity (and volume matches parent volume for new DB); mismatch triggers immediate Store.Close() and fail-closed",
    "Shutdown order: close Named Pipe listener -> drain callers (if drain times out, wait for all active callers to join before releasing lock) -> Store.Close() -> remove .owner.json ONLY IF owner_instance_id matches -> CloseHandle(.owner.lock) LAST",
    "Named Pipe security: authenticate caller SID via ImpersonateNamedPipeClient and token check, verify return errors, revert context via RevertToSelf(); TAKEOVER response returns STOP_ACKNOWLEDGED; CLI must NOT claim stopped successfully before drain finishes",
    "Binary readiness probe on cmd/supervisor proves socket refused before Run, and 200 OK only after Run Complete; zero Pair HTTP routes or effectful Supervisor API on daemon listener",
    "Poller async notification: recovery.Poller must export Done() <-chan struct{} and Err() error; host watcher must monitor Done() and transition /readyz to 503 and auth.SetUnavailable() fail-closed upon observation error",
    "Typed AO workspace read: ao.Client must provide GetWorkspaceFile with URL query path encoding, context cancellation, bounded read cutoff (io.LimitReader maxBytes+1 returning ErrPayloadTooLarge), and guaranteed body closure",
    "Pair hold and 5 AO exit gate steps proven via P03 Integration Test Harness directly invoking approved library APIs without P04/P05 dependencies",
    "All 8 operational policies remain UNSET in documentation, failing closed at runtime if missing; AUTOMATIC_RESTORE remains DISABLED"
  ],
  "acceptance_criteria": [
    "AC-004-01: cmd/supervisor implements Windows exclusive sidecar lock file handle (.owner.lock) with share mode 0, failing closed on contention",
    "AC-004-02: DB post-open verification confirms opened SQLite physical file matches pre-open pinned handle identity",
    "AC-004-03: ExecuteShutdownDrain blocks until all active callers join via Authority.WaitAllReleased() on timeout, preserving lock ownership until clean exit",
    "AC-004-04: Named Pipe TAKEOVER returns status=STOP_ACKNOWLEDGED; CLI stop does not claim stopped successfully before drain completes",
    "AC-004-05: recovery.Poller exports Done() and Err(); host watcher transitions /readyz to 503 and closes admission immediately on background failure",
    "AC-004-06: ao.Client.GetWorkspaceFile cleanly reads workspace files, rejects oversized payloads with ErrPayloadTooLarge, obeys context cancellation, and closes body",
    "AC-004-07: P03 integration test harness verifies 5 AO exit gate steps via approved library APIs without P04/P05 dependencies"
  ]
}
```
