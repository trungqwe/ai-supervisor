# CANDIDATE TASK CONTRACT: TASK-P04-001

> **Contract ID**: `CONTRACT-TASK-P04-001-01`
> **Task ID**: `TASK-P04-001` (Workspace Binding Authority, Schema Migration v6, Seam Extension & Report Intake)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P04`
> **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
> **Status**: `CANDIDATE_NOT_RELEASED` (Status invariant: `NOT_RELEASED`)
> **Authority**: Formulated pursuant to accepted `ADR-018` (ACCEPTED / Level 2 Canonical Specification), approved `PROPOSAL-P04-001` (Revision 22), approved baseline `PROPOSAL-P04-002` (Revision 9), approved `PLAN-P04-EVIDENCE-REVIEW.md` (Revision 22), and approved subtask plan `PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md`.
> **Governance Invariant**: Candidate contract submitted for External Supervisor release audit. Strictly `NOT_RELEASED`. Zero Go production code, zero migration scripts, and zero live AO calls authorized.
> **Runtime Invariants**: `AUTOMATIC_RESTORE = DISABLED`; `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`; zero live AO calls; zero implementation of Subtasks P04B, P04C, or P04D.

---

> [!CRITICAL]
> **GOVERNANCE STATUS: CANDIDATE ONLY / NOT_RELEASED**.
> This document is a candidate task contract awaiting External Supervisor release audit.
> It does **NOT** authorize production code execution or library implementation.
> Production code writing remains strictly **`HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`**.
> Status invariant remains strictly **`NOT_RELEASED`**.
> Candidate contract does **NOT** self-claim Stage B runtime catalog or task release authorization.

## 1. Immutable TaskContract JSON

```json
{
  "contract_id": "CONTRACT-TASK-P04-001-01",
  "task_id": "TASK-P04-001",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P04",
  "objective": "Implement Schema Migration v6, host-boundary WorkspaceBindingAuthority with Win32 128-bit physical identity and anti-rename handle locks, unified PrepareBoundDispatch dispatch seam, 14 pre-send guards, single coordinator effect gate, pre-send diagnostic Variant B, and Transaction A report intake consuming typed GitEvidenceResult interface seam.",
  "requirements": [
    "FR-008",
    "SEC-002",
    "SEC-003",
    "NFR-005",
    "NFR-007",
    "NFR-008"
  ],
  "architecture_refs": [
    "docs/adr/ADR-018-evidence-review-and-verification-isolation.md",
    "docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md",
    "docs/plans/PLAN-P04-EVIDENCE-REVIEW.md",
    "docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/06_WORKFLOW_STATE_MACHINE.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md"
  ],
  "base_sha": "db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2",
  "allowed_scope": [
    "internal/domain/workspace_binding.go",
    "internal/domain/workspace_binding_test.go",
    "internal/domain/worker_claim.go",
    "internal/domain/worker_claim_test.go",
    "internal/domain/review_hold.go",
    "internal/domain/review_hold_test.go",
    "internal/domain/audit_events.go",
    "internal/domain/entities.go",
    "internal/host/authority.go",
    "internal/host/workspace_binding_authority.go",
    "internal/host/workspace_binding_authority_windows.go",
    "internal/host/workspace_binding_authority_test.go",
    "internal/store/migrations.go",
    "internal/store/migrations_v5_test.go",
    "internal/store/migrations_v6_test.go",
    "internal/store/models.go",
    "internal/store/dispatch.go",
    "internal/store/dispatch_transactions.go",
    "internal/store/dispatch_transactions_test.go",
    "internal/store/recovery_transactions_test.go",
    "internal/store/restore_transactions_test.go",
    "internal/store/stop_transactions_test.go",
    "internal/store/workspace_binding_store.go",
    "internal/store/workspace_binding_test.go",
    "internal/store/report_intake_transactions.go",
    "internal/store/report_intake_test.go",
    "internal/store/session_lifecycle_test.go",
    "internal/store/session_lifecycle_remediation_test.go",
    "internal/store/atomic_transitions_test.go",
    "internal/store/dispatch_test.go",
    "internal/dispatch/coordinator.go",
    "internal/dispatch/coordinator_test.go",
    "internal/recovery/scanner.go",
    "internal/recovery/scanner_test.go",
    "internal/recovery/integration_test.go",
    "internal/recovery/poller_test.go",
    "internal/recovery/timeout_monitor_test.go",
    "test/fakes/git_evidence_fake.go",
    "test/integration/ao_harness_test.go"
  ],
  "forbidden_scope": [
    "cmd/supervisor/**",
    "internal/ao/**",
    "internal/recovery/poller.go",
    "internal/recovery/timeout_monitor.go",
    "internal/stop/**",
    "docs/adr/**",
    "docs/audits/**",
    "docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md"
  ],
  "constraints": [
    "Model 1 persistence ownership: Subtask P04A is the sole persistence owner of Schema v6, PrepareBoundDispatch extension, Transaction A, and diagnostic transactions; Subtasks P04B and P04C remain strictly pure in-memory collectors with zero SQLite writes",
    "No Git process execution or collector implementation: Subtask P04A consumes a typed GitEvidenceResult from an injected interface and exercises clean/dirty behavior using test fakes; running git commands (git status, git diff-index), allowlist enforcement, and snapshot extraction belong strictly to Subtask P04B",
    "No implementation of Subtasks P04B, P04C, or P04D: zero Git collector, snapshot extractor, AppContainer sandbox, verification leases, Transaction B, Transaction C, or ReviewBundle synthesis code permitted in this contract",
    "Zero live AO integration: all coordinator tests must use in-memory mock adapters; no live external network or daemon calls",
    "Fail-closed Win32 identity: directory handles must be opened with FILE_FLAG_BACKUP_SEMANTICS, omitting FILE_SHARE_DELETE to block directory rename/delete; 128-bit FileIdInfo and VolumeSerialNumber must be queried and verified; fail closed on non-NTFS/ReFS filesystems",
    "Single coordinator effect gate: Coordinator.Dispatch must hold the live WorkspaceBindingLease across the entire 9-step sequence through /send; zero public pathways accepting snapshot-only to invoke /send",
    "RecordSendRequested snapshot equality guard: Coordinator.Dispatch must pass verified WorkspaceBindingSnapshot to Store.RecordSendRequested after lease.Revalidate(); in the same transaction, Store must compare canonical_worktree_path, worktree_volume_serial_hex, worktree_file_id_hex, linked_gitdir_path, linked_gitdir_volume_serial_hex, linked_gitdir_file_id_hex, pinned_ao_commit, and exact task/attempt/contract/Pair/session/generation lineage against active attempt_workspace_bindings row; snapshot mismatch strictly forbids /send (AO call count = 0), preserves task DISPATCHED, preserves operation DISPATCH_BOUND, keeps attempt open, and triggers diagnostic Variant B",
    "Atomic PrepareBoundDispatch rollback: if any insertion, audit append, or trigger fails, the entire dispatch transaction rolls back leaving zero orphan attempts, operations, bindings, or audits",
    "Pre-send binding failure isolation: on binding guard failure, task state remains DISPATCHED, operation remains DISPATCH_BOUND, and attempt remains open; pre-send diagnostic Variant B records REVIEW_INTEGRITY_CONFLICT, inserts hold INVARIANT_MISMATCH, and CAS invalidates active binding",
    "Deterministic hold descriptors: hold IDs and rejection audit event IDs must be derived via RFC 8785 JCS canonicalization of Descriptors A and B without self-referential preimage dependencies; exact replay returns existing row; recurrence after resolution assigns occurrence N+1",
    "Zero unapproved audit event literals: no creation of new event literals; all events must exist in accepted ADRs",
    "Claim/evidence zero-trust separation: WorkerReport.head_sha is the sole source of worker_claims.reported_head_sha (length 7..40 lowercase hex, stored verbatim); GitEvidenceResult strictly provides independent evidence (actual_head_sha, actual_base_sha, diff details) without ReportedHeadSHA field; Subtask P04A strictly never writes actual_head_sha into worker_claims",
    "Canonical worker claims mapping: WorkerReport fields map into worker_claims (head_sha -> reported_head_sha) and RFC 8785 JCS canonicalized payload_json (files_changed -> claimed_files_changed, tests -> tests, worker_claims -> textual_claims, build_status -> build_status)",
    "Daemon restart recovery invariant: daemon restart invalidates all in-memory lease capability; recovery scanner reads durable paths/lineage from DB, creates WorkspaceBindingCandidate, and calls authority.Acquire(ctx, candidate) to open a fresh lease and compare physical identity against durable attempt_workspace_bindings; scanner strictly never invokes Store.RecordSendRequested, never transitions stage to SEND_REQUESTED, and never invokes AO /send (RecordSendRequested call count = 0, AO /send call count = 0); exact match allows pre-send classification while preserving task DISPATCHED, operation DISPATCH_BOUND, and attempt open; identity mismatch or acquire failure strictly fails closed with zero /send, executes diagnostic Variant B, leaves task DISPATCHED, operation DISPATCH_BOUND, and attempt open; terminalization is strictly a separate D12 transaction performed by verified operator (DISPATCHED -> FAILED, recovery_disposition = WORKSPACE_BINDING_INTEGRITY_FAILURE); automated re-send after restart is not authorized",
    "Preservation of migrations_v5_test.go: modifications to internal/store/migrations_v5_test.go are strictly limited to preserving historical v5 upgrade and rollback test coverage when CurrentSchemaVersion advances to 6 without weakening v5 fixtures or assertions",
    "Worker profile constraint: antigravity-standard is restricted strictly to worker_profile; all verification requests must use host profile go-test-p04-001",
    "Verification request policy: each request must declare profile_id: go-test-p04-001, cwd: \".\", timeout_seconds <= 300, and typed parameters.package and parameters.flags (const [\"-v\", \"-race\", \"-count=1\"]); no command or shell command strings",
    "No enabling of AUTOMATIC_RESTORE: automatic restore remains fail-closed; operator restore and linked stop remain disabled until verified operator principal reaches trusted boundary",
    "All tests must pass under the Go race detector with -race -count=1"
  ],
  "acceptance_criteria": [
    "AC-P04A-01: Schema Migration v6 applies cleanly on fresh and historical v5 SQLite databases, creating attempt_workspace_bindings, worker_claims, and review_integrity_holds matching ADR-018 verbatim with partial unique active index and strict foreign keys",
    "AC-P04A-02: Migration v6 rollback atomicity verified: intentional mid-migration failure cleanly rolls back leaving database at v5 schema with zero orphaned v6 objects",
    "AC-P04A-03: Lineage triggers (trg_attempt_workspace_bindings_lineage_guard, trg_worker_claims_lineage_guard, trg_review_integrity_holds_lineage_guard) reject mismatched (task_id, contract_id) references against task_attempts and require DISPATCH_BOUND for bindings",
    "AC-P04A-04: CAS triggers (trg_attempt_workspace_bindings_cas_guard, trg_review_integrity_holds_cas_guard) enforce valid forward state transitions and reject invalid downgrades or modifications to resolved holds",
    "AC-P04A-05: Immutability triggers reject DELETE operations on attempt_workspace_bindings, worker_claims, and review_integrity_holds, and reject UPDATE operations on worker_claims",
    "AC-P04A-06: Host WorkspaceBindingAuthority acquires directory handles omitting FILE_SHARE_DELETE; OS returns ERROR_SHARING_VIOLATION on concurrent directory rename or deletion attempts while lease is active",
    "AC-P04A-07: Win32 128-bit physical identity capture verified via GetFileInformationByHandleEx(FileIdInfo); directory substitution probes (junctions, symlinks, subst, rename) fail physical identity verification",
    "AC-P04A-08: WorkspaceBindingLease Revalidate() detects handle invalidation, path changes, and identity mismatch; Close() idempotently releases both handles",
    "AC-P04A-09: PrepareBoundDispatch store seam atomically claims task READY -> DISPATCHED, allocates task_attempts with session snapshot, inserts dispatch_operations (DISPATCH_BOUND), inserts attempt_workspace_bindings (ACTIVE), and appends TASK_DISPATCH_BOUND and WORKSPACE_BINDING_CREATED audit events in a single transaction",
    "AC-P04A-10: Single coordinator effect gate in Coordinator.Dispatch revalidates live WorkspaceBindingLease and passes verified WorkspaceBindingSnapshot to Store.RecordSendRequested; in the same transaction, Store compares worktree/gitdir paths, 128-bit physical file IDs, volume serials, pinned AO commit, and exact lineage against active attempt_workspace_bindings before committing send intent; snapshot mismatch strictly forbids /send, preserves task DISPATCHED, operation DISPATCH_BOUND, and attempt open, and triggers diagnostic Variant B",
    "AC-P04A-11: Pre-send binding rejection executes diagnostic Variant B transaction: appends REVIEW_INTEGRITY_CONFLICT (conflict_source = 'WORKSPACE_BINDING_GUARD', attempted_reason = literal, colliding_event_id absent), inserts hold INVARIANT_MISMATCH, CAS invalidates binding, and preserves task DISPATCHED and attempt open",
    "AC-P04A-12: Governed terminalization verified: binding integrity failure requires operator terminalization DISPATCHED -> FAILED (recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE') before hold resolution; retries allocate a fresh TaskAttempt",
    "AC-P04A-13: Ingestion interface seam accepts typed GitEvidenceResult without process execution; behavior tests verify that clean result permits Transaction A whereas dirty result triggers intake diagnostic transaction using test fakes",
    "AC-P04A-14: Transaction A validates worker report, canonicalizes payload via RFC 8785 JCS, persists worker_claims with verbatim reported_head_sha from WorkerReport.head_sha (length 7..40 hex), CAS advances binding to RETAINED_FOR_VERIFICATION, and CAS advances task to REPORT_READY without unapproved audit events",
    "AC-P04A-15: Dirty report intake cleanly rolls back Transaction A, preserves task state RUNNING, appends EVIDENCE_COLLECTION_FAILED audit event, and inserts ACTIVE hold DIRTY_WORKTREE_DETECTED via deterministic Descriptors A and B; exact replay returns existing row, and recurrence after resolution allocates occurrence N+1",
    "AC-P04A-16: Zero orphaned audit events or partial holds on transaction rollback across all dispatch and intake error pathways",
    "AC-P04A-17: All verification requests declare profile_id go-test-p04-001, cwd \".\", timeout_seconds <= 300, and typed package/flags parameters; pass TaskContractValidator.ValidateRaw with catalog metadata and reject all 6 negative probes",
    "AC-P04A-18: Claim/evidence zero-trust separation: WorkerReport.head_sha is the sole source for worker_claims.reported_head_sha; GitEvidenceResult provides independent actual_head_sha without ReportedHeadSHA field; behavior tests verify that simulating reported_head_sha != actual_head_sha preserves both values independently without mutual overwrite",
    "AC-P04A-19: Daemon restart recovery in recovery.Runner: scanner reopens worktree root and linked gitdir by constructing WorkspaceBindingCandidate and invoking authority.Acquire, matching exact canonical path, VolumeSerialNumber, and FileIdInfo against durable attempt_workspace_bindings; prior in-memory handle token or standalone DB row does not grant effect authority; acquire with identical physical identity passes, while mismatch or acquire failure fails closed with zero /send, executes diagnostic Variant B, leaves task DISPATCHED, operation DISPATCH_BOUND, and attempt open; terminalization requires verified operator D12 transaction",
    "AC-P04A-20: RecordSendRequested snapshot guard behavior tests: exact snapshot PASS; stale or fake snapshot FAIL; path or worktree FileId mismatch FAIL; linked gitdir identity mismatch FAIL; pinned AO commit mismatch FAIL; all failure probes verify AO send call count = 0 and atomic rollback of audit, hold, and binding CAS",
    "AC-P04A-21: Exact-match restart recovery effect boundary: scanner Acquire and physical identity comparison PASS verifies durable binding, but RecordSendRequested call count = 0, AO /send call count = 0, and dispatch operation stage remains DISPATCH_BOUND; directory mismatch or Acquire failure executes atomic Variant B, CAS invalidates binding, and keeps attempt open; zero snapshot-only or scanner-direct effect pathways permitted"
  ],
  "verification_requests": [
    {
      "id": "VR-P04A-DOMAIN",
      "profile_id": "go-test-p04-001",
      "cwd": ".",
      "timeout_seconds": 60,
      "parameters": {
        "package": "./internal/domain/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      }
    },
    {
      "id": "VR-P04A-HOST",
      "profile_id": "go-test-p04-001",
      "cwd": ".",
      "timeout_seconds": 120,
      "parameters": {
        "package": "./internal/host/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      }
    },
    {
      "id": "VR-P04A-STORE",
      "profile_id": "go-test-p04-001",
      "cwd": ".",
      "timeout_seconds": 180,
      "parameters": {
        "package": "./internal/store/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      }
    },
    {
      "id": "VR-P04A-DISPATCH",
      "profile_id": "go-test-p04-001",
      "cwd": ".",
      "timeout_seconds": 120,
      "parameters": {
        "package": "./internal/dispatch/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      }
    },
    {
      "id": "VR-P04A-RECOVERY",
      "profile_id": "go-test-p04-001",
      "cwd": ".",
      "timeout_seconds": 120,
      "parameters": {
        "package": "./internal/recovery/...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      }
    },
    {
      "id": "VR-P04A-FULL-RACE",
      "profile_id": "go-test-p04-001",
      "cwd": ".",
      "timeout_seconds": 300,
      "parameters": {
        "package": "./...",
        "flags": [
          "-v",
          "-race",
          "-count=1"
        ]
      }
    }
  ],
  "required_evidence": [
    "git_diff_allowed_scope_only",
    "test_log_migration_v6_fresh_and_upgrade",
    "test_log_wba_win32_substitution_and_anti_rename",
    "test_log_prepare_bound_dispatch_atomicity",
    "test_log_coordinator_14_guards_and_variant_b",
    "test_log_record_send_requested_snapshot_equality_and_rejection",
    "test_log_report_intake_clean_and_dirty_transactions",
    "test_log_hold_dedup_and_recurrence_lifecycle",
    "test_log_claim_evidence_decoupling_reported_vs_actual_sha",
    "test_log_restart_recovery_scanner_acquire_and_mismatch_fail_closed",
    "test_log_full_suite_race_exit_zero",
    "test_log_restart_recovery_scanner_exact_match_zero_send_effect"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Any test failure under go test -race -count=1",
    "Any schema migration defect or foreign key / trigger violation",
    "Any failure of Win32 128-bit FileIdInfo query or anti-rename handle locks",
    "Any modification to files outside allowed_scope or inside forbidden_scope",
    "Any attempt to execute git processes, parse git output, or implement Subtasks P04B, P04C, or P04D",
    "Any live external network or AO communication detected during testing",
    "Any violation of task state machine invariants or compound state usage",
    "Any failure to preserve RUNNING state during dirty report intake",
    "Any introduction of unapproved audit literals or unapproved verification profiles"
  ]
}
```

---

## 2. Proposed Verification Policy Catalog Metadata (Outside JSON)

Pursuant to ADR-013 and Findings `P04A-C1-004` and `P04A-R1-002`, host verification profile `go-test-p04-001` is defined by the host environment catalog as follows:

```json
{
  "profile_id": "go-test-p04-001",
  "cwd_policy": "worktree_root",
  "max_timeout_seconds": 300,
  "parameter_schema": {
    "type": "object",
    "required": [
      "package",
      "flags"
    ],
    "additionalProperties": false,
    "properties": {
      "package": {
        "type": "string",
        "enum": [
          "./...",
          "./internal/domain/...",
          "./internal/host/...",
          "./internal/store/...",
          "./internal/dispatch/...",
          "./internal/recovery/..."
        ]
      },
      "flags": {
        "type": "array",
        "items": {
          "type": "string"
        },
        "const": [
          "-v",
          "-race",
          "-count=1"
        ]
      }
    }
  }
}
```

### 2.1. Rejection Matrix for Mandatory Negative Probes
1. **Unknown Profile Probe**: Requesting any `profile_id` not registered in catalog (e.g. `unknown-profile`, `antigravity-standard` in verification requests) is rejected (`profile not found in policy catalog`).
2. **Timeout Boundary Probe**: Requesting `timeout_seconds = 301` (> 300) is rejected by `TaskContractValidator` (`requested timeout exceeds profile maximum`).
3. **Flag Injection Probe**: Including flags such as `-exec` or unauthorized arguments violates `const: ["-v", "-race", "-count=1"]` and is rejected.
4. **Package Enumeration Probe**: Requesting packages outside the enum (e.g. `./cmd/...` or `./test/...`) is rejected (`enum constraint violation`).
5. **Cwd Traversal Probe**: Specifying relative escape paths (e.g. `../escape`) is rejected by `CwdPolicy: worktree_root`.
6. **Extra Property Probe**: Supplying unauthorized parameters (e.g. `"command": "..."`) violates `additionalProperties: false` and is rejected.

### 2.2. Categorized Caller Impact Matrix & Scope Reconciliation (Findings P04A-R1-003, P04A-R2-001, P04A-R2-002, P04A-R3-001, P04A-R4-001)
To guarantee that no API pathway or caller can allocate an attempt or issue `/send` without an active, verified workspace binding matching the durable snapshot, all repository files referencing `PrepareBoundDispatch`, `RecordSendRequested`, and `dispatch.Coordinator` construction identified via static repository analysis are explicitly categorized into four granular groups within `allowed_scope`:
1. **Seam Definitions**: Core domain and store API definitions.
2. **Direct Production Callers**: Production code paths invoking the seams.
3. **Direct Test Callers/Constructions**: Test suites invoking seams directly or constructing the coordinator.
4. **Indirect Behavior Tests**: Test suites exercising seams indirectly through higher-level component interfaces.

| Category | Function / Seam | Affected File | Caller Role / Impact | Remediation & Scope Allocation |
|---|---|---|---|---|
| **Seam Definition** | `PrepareBoundDispatch` | `internal/store/dispatch.go` | Seam Definition | Extends transaction to accept `WorkspaceBindingSnapshot` and insert `attempt_workspace_bindings` (ACTIVE) atomically. (Directly in `allowed_scope`). |
| **Seam Definition** | `RecordSendRequested` | `internal/store/dispatch_transactions.go` | Seam Definition | Enforces 14 pre-send guards including Guard 12 (active binding) and Guard 13 (snapshot equality). (Directly in `allowed_scope`). |
| **Seam Definition** | `Coordinator` | `internal/dispatch/coordinator.go` | Struct Definition | Injects `WorkspaceBindingAuthority` domain interface. (Directly in `allowed_scope`). |
| **Direct Production Caller** | `PrepareBoundDispatch` | `internal/dispatch/coordinator.go` | Coordinator Effect Gate | Calls `PrepareBoundDispatch` with live lease snapshot during 9-step atomic dispatch sequence. (Directly in `allowed_scope`). |
| **Direct Production Caller** | `RecordSendRequested` | `internal/dispatch/coordinator.go` | Exclusive Pre-Send Effect Gate | Sole production caller: passes verified `WorkspaceBindingSnapshot` to Store after live lease revalidation before wire `/send`. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `PrepareBoundDispatch` | `internal/store/atomic_transitions_test.go` | Atomic Tests | Updated to pass valid test workspace binding snapshot. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `PrepareBoundDispatch` | `internal/store/session_lifecycle_test.go` | Store Lifecycle Tests | Test helper `setupBoundAttempt` and test cases updated to pass valid `WorkspaceBindingSnapshot`. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `PrepareBoundDispatch` | `internal/store/session_lifecycle_remediation_test.go` | Remediation Tests | Updated to pass valid test workspace binding snapshot. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `PrepareBoundDispatch` | `internal/recovery/scanner_test.go` | Recovery Test Helper | `seedBoundExecution` helper updated to supply valid test workspace binding snapshot. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/store/dispatch_transactions_test.go` | Transaction Tests | Exercises Guard 12 and Guard 13 snapshot equality validation and rejection directly. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/store/recovery_transactions_test.go` | Recovery Store Tests | Direct test fixtures seed dispatch state matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/store/restore_transactions_test.go` | Restore Tests | Direct test fixtures seed dispatch state matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/store/stop_transactions_test.go` | Stop Invariant Tests | Direct test fixtures seed dispatch state matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/store/session_lifecycle_test.go` | Store Lifecycle Tests | Updated to pass valid test workspace binding snapshot matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/recovery/scanner_test.go` | Recovery Scanner Tests | Direct test fixtures seed dispatch state matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/recovery/timeout_monitor_test.go` | Timeout Monitor Tests | Direct test fixtures seed dispatch state matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/recovery/poller_test.go` | Poller Tests | Direct test fixtures seed dispatch state matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `RecordSendRequested` | `internal/recovery/integration_test.go` | Recovery Integration Tests | Direct test fixtures seed dispatch state matching attempt binding. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `Coordinator` Construction | `internal/dispatch/coordinator_test.go` | Unit Tests | Injects mock/fake `WorkspaceBindingAuthority`. (Directly in `allowed_scope`). |
| **Direct Test Caller** | `Coordinator` Construction | `test/integration/ao_harness_test.go` | Full Harness Test | Injects test `WorkspaceBindingAuthority`. (Directly in `allowed_scope`). |
| **Indirect Behavior Test** | `PrepareBoundDispatch` & `RecordSendRequested` | `internal/dispatch/coordinator_test.go` | Dispatch Component Tests | Exercises `PrepareBoundDispatch` and `RecordSendRequested` indirectly through `Coordinator.Dispatch`. (Directly in `allowed_scope`). |

**Summary of Categorized References**:
- **Seam Definitions**: 3 entries (`internal/store/dispatch.go`, `internal/store/dispatch_transactions.go`, `internal/dispatch/coordinator.go`).
- **Direct Production Callers**: 2 seam invocations across 1 file (`internal/dispatch/coordinator.go` for both `PrepareBoundDispatch` and `RecordSendRequested`).
- **Direct Test Callers/Constructions**: 15 entries across 13 distinct test files.
- **Indirect Behavior Tests**: 1 entry (`internal/dispatch/coordinator_test.go`).
- **Total Cataloged Entries**: 21 entries across 16 distinct repository files, 100% of which are directly included in `allowed_scope`.
- **Exclusion of Recovery Scanner from Effect Calls (Finding P04A-R3-001)**: `internal/recovery/scanner.go` is strictly NOT a caller of `RecordSendRequested` and strictly NOT a caller of wire `/send`. It remains in `allowed_scope` because it consumes `WorkspaceBindingAuthority.Acquire` and performs physical identity comparison against durable bindings.
- **Preservation of `internal/store/dispatch_test.go` in `allowed_scope` (Finding P04A-R4-001)**: While `internal/store/dispatch_test.go` does not directly invoke `PrepareBoundDispatch` (testing legacy unbound rejection via `prepareLegacyDispatchForTest` instead), it is explicitly retained in `allowed_scope` to ensure legacy rejection invariants remain unbroken under Schema v6.

**Scope Non-Overlap Guarantee**:
- Blanket wildcard `internal/recovery/**` is eliminated from `forbidden_scope`.
- `allowed_scope` includes strictly 39 files, including `internal/recovery/scanner.go`, `internal/recovery/scanner_test.go`, `internal/recovery/integration_test.go`, `internal/recovery/poller_test.go`, and `internal/recovery/timeout_monitor_test.go`.
- Other recovery files are explicitly listed in `forbidden_scope` (`internal/recovery/poller.go`, `internal/recovery/timeout_monitor.go`), guaranteeing zero overlap between `allowed_scope` and `forbidden_scope`.
- Modifications to `internal/store/migrations_v5_test.go` are strictly limited to preserving v5 upgrade/rollback testing when `CurrentSchemaVersion` advances to 6 without weakening historical test coverage.

### 2.3. Daemon Restart Recovery Specification (Findings P04A-R1-003, P04A-R2-003, P04A-R3-001)
Pursuant to ADR-018:
1. **Capability Invalidation**: Daemon restart completely invalidates all in-memory handle capability; all prior OS handles were closed by the OS on process exit.
2. **Startup Recovery Acquisition Sequence**: When the daemon startup recovery scanner (`internal/recovery/scanner.go`) sweeps in-flight dispatches:
   - Neither a prior in-memory handle token nor a standalone DB row grants effect authority.
   - Scanner reads the durable canonical paths, volume serial, and file IDs from `attempt_workspace_bindings` where `binding_state = 'ACTIVE'`.
   - Scanner constructs a `WorkspaceBindingCandidate` containing canonical worktree and linked gitdir paths.
   - Scanner invokes `authority.Acquire(ctx, candidate)` to open fresh directory handles and acquire a new `WorkspaceBindingLease`.
   - Scanner extracts `lease.Snapshot()` and compares physical identity (`VolumeSerialNumber`, `FileIdInfo`) against durable binding.
3. **Strict Prohibition of Scanner Effect Calls (Finding P04A-R3-001)**:
   - The recovery scanner strictly NEVER invokes `Store.RecordSendRequested`, strictly NEVER transitions dispatch operation stage to `SEND_REQUESTED`, and strictly NEVER invokes AO `/send` (`RecordSendRequested` call count = 0, AO `/send` call count = 0).
   - `Coordinator.Dispatch` remains the sole, exclusive effect gate authorized to hold the live lease through `RecordSendRequested` and wire `/send`.
4. **Physical Identity Matching Semantics**:
   - Exact identity match: `Acquire` succeeds and physical identity matches durable binding; pre-send recovery evaluation proceeds pursuant to approved P03 APIs (e.g. evaluating in-flight state and clearing `RECOVERY_PENDING`).
   - Post-condition strictly remains: task state `DISPATCHED`, dispatch operation stage `DISPATCH_BOUND`, attempt remains open.
   - Absent an approved recovery effect API, the scanner cannot trigger or resume wire effects. Automated re-send after daemon restart is strictly not authorized in Subtask P04A and would require a formal Proposal / ADR-018 addendum.
   - Identity mismatch or failure (directory renamed, swapped, deleted, junction substituted) or `Acquire` failure: fail closed, zero `/send` calls (AO call count = 0), executes diagnostic Variant B transaction (`attempted_reason = 'WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH'`, active hold `INVARIANT_MISMATCH`, CAS binding `ACTIVE -> INVALIDATED`).
   - Task state remains `DISPATCHED`, operation remains `DISPATCH_BOUND`, attempt remains open.
   - Terminalization is strictly deferred to a separate D12 transaction performed by a verified operator (`DISPATCHED -> FAILED`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'`). The scanner never terminalizes the attempt in the diagnostic transaction and never automatically transitions to `HUMAN_REQUIRED`.
5. **Narrow Authority Interface**: To avoid circular package imports (`internal/host` -> `internal/recovery` -> `internal/host`), `WorkspaceBindingAuthority` is declared in `internal/domain/workspace_binding.go` and consumed by both `internal/dispatch` and `internal/recovery`:
   ```go
   type WorkspaceBindingAuthority interface {
       Acquire(ctx context.Context, candidate WorkspaceBindingCandidate) (WorkspaceBindingLease, error)
   }
   ```

---

## 3. Governance Baseline & Pre-Execution Directives

1. **Pre-Release Invariant**:
   - This task contract is a `DRAFT` and is strictly `NOT_RELEASED`.
   - `ACTIVE_GATE = TASK_P04A_CONTRACT_PLANNING`.
   - `P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`.
   - `P05_CODE = NOT_AUTHORIZED`.
2. **Authority & Approvals**:
   - Formulated pursuant to accepted `ADR-018` (ACCEPTED / Level 2 Canonical Specification).
   - Awaits formal release audit by the External Supervisor before branch creation or implementation.
3. **Execution Isolation**:
   - Once released, implementation must occur strictly on isolated branch `codex/p04-001` starting from base SHA `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`.
   - Zero modifications to canonical specifications or historical audits permitted during implementation.
