# DRAFT TASK CONTRACT: TASK-P04-001 (REVISION 2)

> **Contract ID**: CONTRACT-TASK-P04-001-02
> **Task ID**: TASK-P04-001 (Workspace Binding Authority, Schema Migration v6, Seam Extension & Report Intake)
> **Revision Number**: 2
> **Supersedes Contract ID**: CONTRACT-TASK-P04-001-01
> **Phase ID**: P04
> **Base SHA**: db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2
> **Status**: NOT_RELEASED (Draft Proposed)
> **Authority**: Formulated pursuant to accepted [docs/adr/ADR-018-evidence-review-and-verification-isolation.md](../adr/ADR-018-evidence-review-and-verification-isolation.md), approved PROPOSAL-P04-001 (Revision 22), approved baseline PROPOSAL-P04-002 (Revision 9), approved PLAN-P04-EVIDENCE-REVIEW.md (Revision 22), approved subtask plan PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md, External Re-Audit 001 (docs/audits/P04_TASK_001_EXTERNAL_REAUDIT_001.md), and proposed [docs/proposals/PROPOSAL-P04-003-p04a-recovery-authority-seam-remediation.md](../proposals/PROPOSAL-P04-003-p04a-recovery-authority-seam-remediation.md).
> **Implementation Scope**: Authorized strictly within allowed_scope (40 files) on isolated implementation branch codex/p04-001 once formally released.
> **Runtime Invariants**: AUTOMATIC_RESTORE = DISABLED; VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY; zero live AO calls; zero implementation of Subtasks P04B, P04C, or P04D; P04_CODE = HELD_PENDING_P04A_CONTRACT_REVISION; P05_CODE = NOT_AUTHORIZED.

---

> [!IMPORTANT]
> **GOVERNANCE STATUS: DRAFT PROPOSED (NOT RELEASED)**.
> This document is the proposed Revision 2 Task Contract for TASK-P04-001 (Subtask P04A).
> Implementation changes remain strictly **HELD** pending formal External Supervisor audit and release.
> Base SHA remains pinned to `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`.
> Production code writing remains unauthorized until formal contract release (`P04_CODE = HELD_PENDING_P04A_CONTRACT_REVISION`).
> Subtasks P04B, P04C, and P04D remain strictly **NOT_RELEASED**.
> Phase P05 code remains strictly **NOT_AUTHORIZED**.
> AUTOMATIC_RESTORE = DISABLED.

## 1. Immutable TaskContract JSON

```json
{
  "contract_id": "CONTRACT-TASK-P04-001-02",
  "task_id": "TASK-P04-001",
  "revision_number": 2,
  "supersedes_contract_id": "CONTRACT-TASK-P04-001-01",
  "phase_id": "P04",
  "objective": "Implement Schema Migration v6, host-boundary WorkspaceBindingAuthority with Win32 128-bit physical identity and anti-rename handle locks, unified PrepareBoundDispatch dispatch seam, 14 pre-send guards, single coordinator effect gate, recovery scanner direct authority forwarding via Poller seam without global mutable state, pre-send diagnostic Variant B, and Transaction A report intake via exclusive raw JSON schema admission consuming typed GitEvidenceResult interface seam.",
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
    "docs/proposals/PROPOSAL-P04-003-p04a-recovery-authority-seam-remediation.md",
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
    "internal/recovery/poller.go",
    "internal/recovery/poller_test.go",
    "internal/recovery/timeout_monitor_test.go",
    "test/fakes/git_evidence_fake.go",
    "test/integration/ao_harness_test.go"
  ],
  "forbidden_scope": [
    "cmd/supervisor/**",
    "internal/ao/**",
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
    "RecordSendRequested non-variadic snapshot guard: Coordinator.Dispatch must pass verified WorkspaceBindingSnapshot to Store.RecordSendRequested after lease.Revalidate(); in the same transaction, Store must compare canonical_worktree_path, worktree_volume_serial_hex, worktree_file_id_hex, linked_gitdir_path, linked_gitdir_volume_serial_hex, linked_gitdir_file_id_hex, pinned_ao_commit, and exact task/attempt/contract/Pair/session/generation lineage against active attempt_workspace_bindings row; snapshot mismatch strictly forbids /send (AO call count = 0), preserves task DISPATCHED, preserves operation DISPATCH_BOUND, keeps attempt open, and triggers diagnostic Variant B",
    "Atomic PrepareBoundDispatch rollback: if any insertion, audit append, or trigger fails, the entire dispatch transaction rolls back leaving zero orphan attempts, operations, bindings, or audits",
    "Pre-send binding failure isolation: on binding guard failure, task state remains DISPATCHED, operation remains DISPATCH_BOUND, and attempt remains open; pre-send diagnostic Variant B records REVIEW_INTEGRITY_CONFLICT, inserts hold INVARIANT_MISMATCH, and CAS invalidates active binding",
    "Deterministic hold descriptors: hold IDs and rejection audit event IDs must be derived via RFC 8785 JCS canonicalization of Descriptors A and B without self-referential preimage dependencies; exact replay returns existing row; recurrence after resolution assigns occurrence N+1",
    "Zero unapproved audit event literals: no creation of new event literals; all events must exist in accepted ADRs",
    "Claim/evidence zero-trust separation: WorkerReport.head_sha is the sole source of worker_claims.reported_head_sha (length 7..40 lowercase hex, stored verbatim); GitEvidenceResult strictly provides independent evidence (actual_head_sha, actual_base_sha, diff details) without ReportedHeadSHA field; Subtask P04A strictly never writes actual_head_sha into worker_claims",
    "Canonical worker claims mapping: WorkerReport fields map into worker_claims (head_sha -> reported_head_sha) and RFC 8785 JCS canonicalized payload_json (files_changed -> claimed_files_changed, tests -> tests, worker_claims -> textual_claims, build_status -> build_status)",
    "Daemon restart recovery invariant: daemon restart invalidates all in-memory lease capability; recovery scanner reads durable paths/lineage from DB, creates WorkspaceBindingCandidate, and calls authority.Acquire(ctx, candidate) to open a fresh lease and compare physical identity against durable attempt_workspace_bindings; scanner strictly never invokes Store.RecordSendRequested, never transitions stage to SEND_REQUESTED, and never invokes AO /send (RecordSendRequested call count = 0, AO /send call count = 0); exact match allows pre-send classification while preserving task DISPATCHED, operation DISPATCH_BOUND, and attempt open; identity mismatch or acquire failure strictly fails closed with zero /send, executes diagnostic Variant B, leaves task DISPATCHED, operation DISPATCH_BOUND, and attempt open; terminalization is strictly a separate D12 transaction performed by verified operator (DISPATCHED -> FAILED, recovery_disposition = WORKSPACE_BINDING_INTEGRITY_FAILURE); automated re-send after restart is not authorized",
    "Direct recovery authority forwarding without global state: Poller.PollOnce forwards authority directly from completed owner (Runner{..., Authority: p.Owner.Authority}); mutable global map storeAuthorities and all package-level registries are strictly prohibited; Runner with nil authority unconditionally executes Variant B",
    "Lease close error handling: any error returned by WorkspaceBindingLease.Close() in recovery or coordinator must be captured and handled; closing handles must not fail silently",
    "Sole public raw JSON report admission: Store.IngestWorkerReportRaw is the sole public admission entry point for worker reports, validating raw JSON bytes against docs/schemas/worker-report.schema.json using the Draft-07 JSON Schema engine; exported struct bypass is strictly prohibited",
    "Explicit symlink privilege policy: host symlink probe in workspace_binding_authority_test.go must check client privileges; if privilege is unheld, test must call t.Skip() and AC-P04A-07 symlink status must remain UNVERIFIED; false PASS via unasserted return is strictly forbidden",
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
    "AC-P04A-07: Win32 128-bit physical identity capture verified via GetFileInformationByHandleEx(FileIdInfo); directory substitution probes (junctions, symlinks, subst, rename) fail physical identity verification; unheld symlink privilege explicitly skips probe without false PASS",
    "AC-P04A-08: WorkspaceBindingLease Revalidate() detects handle invalidation, path changes, and identity mismatch; Close() idempotently releases both handles and propagates close errors",
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
    "AC-P04A-20: RecordSendRequested non-variadic snapshot guard behavior tests: exact snapshot PASS; stale or fake snapshot FAIL; path or worktree FileId mismatch FAIL; linked gitdir identity mismatch FAIL; pinned AO commit mismatch FAIL; all failure probes verify AO send call count = 0 and atomic rollback of audit, hold, and binding CAS",
    "AC-P04A-21: Exact-match restart recovery effect boundary: scanner Acquire and physical identity comparison PASS verifies durable binding, but RecordSendRequested call count = 0, AO /send call count = 0, and dispatch operation stage remains DISPATCH_BOUND; directory mismatch or Acquire failure executes atomic Variant B, CAS invalidates binding, and keeps attempt open; zero snapshot-only or scanner-direct effect pathways permitted",
    "AC-P04A-22: Poller transient runner direct authority forwarding verified: Poller.PollOnce forwards authority directly from completed owner; package-level mutable global map storeAuthorities is deleted; Runner with nil authority unconditionally executes Variant B, leaving binding INVALIDATED",
    "AC-P04A-23: Exclusive raw JSON worker report intake verified: IngestWorkerReportRaw is the sole public admission entry point; payloads with unrecognized fields violate additionalProperties: false and fail admission before initiating store transaction; bypass via decoded struct is eliminated",
    "AC-P04A-24: Symlink probe execution integrity: test checks Windows privilege; executes and verifies substitution when elevated; explicitly calls t.Skip() when unheld without producing false-positive PASS"
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
    "test_log_restart_recovery_scanner_exact_match_zero_send_effect",
    "test_log_recovery_poller_direct_authority_forwarding_zero_global_map",
    "test_log_raw_worker_report_schema_rejection_additional_properties"
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
    "Any introduction of unapproved audit literals or unapproved verification profiles",
    "Any introduction of mutable global authority state across runner instances",
    "Any bypass of Draft-07 JSON Schema validation during worker report admission"
  ]
}
```

---

## 2. Verification Policy Catalog Metadata (Outside JSON)

Pursuant to ADR-013, host verification profile `go-test-p04-001` is defined by the host environment catalog as follows:

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
4. **Shell Ingestion Probe**: Specifying `command` or raw string scripts in verification request parameters is rejected (`command parameters not permitted under typed runner policy`).
5. **Cwd Traversal Probe**: Setting `cwd: "../.."` violates `worktree_root` containment and is rejected.
6. **Package Path Escaping Probe**: Passing unauthorized package paths outside the enum whitelist is rejected.

---

## 3. Categorized Scope Catalog

### 3.1. Seam Definitions (3 entries)
- `internal/store/dispatch.go`: Seam definition for `PrepareBoundDispatch` and `DispatchBinding`.
- `internal/store/dispatch_transactions.go`: Seam definition for `RecordSendRequested`.
- `internal/dispatch/coordinator.go`: Coordinator seam calling `PrepareBoundDispatch` and `RecordSendRequested`.

### 3.2. Direct Production Callers (3 entries across 2 files)
- `internal/dispatch/coordinator.go`: Invocations of `PrepareBoundDispatch` and `RecordSendRequested`.
- `internal/recovery/poller.go`: Direct forwarding of `Authority: p.Owner.Authority` to transient `Runner`.

### 3.3. Scope Non-Overlap Verification
- `allowed_scope` contains exactly 40 files.
- `forbidden_scope` explicitly lists:
  * `cmd/supervisor/**`
  * `internal/ao/**`
  * `internal/recovery/timeout_monitor.go`
  * `internal/stop/**`
  * `docs/adr/**`
  * `docs/audits/**`
  * `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md`
- Scope overlap between `allowed_scope` and `forbidden_scope`: **0 files**.
