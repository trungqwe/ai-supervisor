# DRAFT TASK CONTRACT: TASK-P04-001

> **Contract ID**: `CONTRACT-TASK-P04-001-01`
> **Task ID**: `TASK-P04-001` (Workspace Binding Authority, Schema Migration v6, Seam Extension & Report Intake)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P04`
> **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
> **Status**: `DRAFT_NOT_RELEASED`
> **Authority**: Formulated pursuant to accepted `ADR-018` (ACCEPTED / Level 2 Canonical Specification), approved `PROPOSAL-P04-001` (Revision 22), approved `PLAN-P04-EVIDENCE-REVIEW.md` (Revision 22), and `PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md`.
> **Governance Notice**: This contract is in DRAFT status and is strictly NOT RELEASED. Code writing is strictly HELD pending external audit and formal release.
> **Runtime Invariants**: `AUTOMATIC_RESTORE = DISABLED`; `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`; zero live AO calls; zero implementation of Subtasks P04B, P04C, or P04D.

---

> [!CAUTION]
> **GOVERNANCE STATUS: DRAFT — NOT RELEASED FOR IMPLEMENTATION**.
> This document is a non-executable draft task contract prepared for External Supervisor audit.
> Absolutely ZERO production Go code, migration files, or unit tests may be created under this draft.
> Release authorization requires formal release audit approval by the External Supervisor.
> Verification profiles are proposed and subject to host catalog verification; Stage B validation is NOT declared as passed.

## 1. Immutable TaskContract JSON

```json
{
  "contract_id": "CONTRACT-TASK-P04-001-01",
  "task_id": "TASK-P04-001",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P04",
  "objective": "Implement Schema Migration v6, host-boundary WorkspaceBindingAuthority with Win32 128-bit physical identity and anti-rename handle locks, unified PrepareBoundDispatch dispatch seam, 14 pre-send guards, single coordinator effect gate, pre-send diagnostic Variant B, and Transaction A report intake with clean worktree verification.",
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
    "internal/store/migrations_v6_test.go",
    "internal/store/models.go",
    "internal/store/dispatch.go",
    "internal/store/dispatch_transactions.go",
    "internal/store/dispatch_transactions_test.go",
    "internal/store/workspace_binding_store.go",
    "internal/store/workspace_binding_test.go",
    "internal/store/report_intake_transactions.go",
    "internal/store/report_intake_test.go",
    "internal/dispatch/coordinator.go",
    "internal/dispatch/coordinator_test.go",
    "test/fakes/git_evidence_fake.go"
  ],
  "forbidden_scope": [
    "cmd/supervisor/**",
    "internal/ao/**",
    "internal/recovery/**",
    "internal/stop/**",
    "docs/adr/**",
    "docs/audits/**",
    "docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md"
  ],
  "constraints": [
    "Model 1 persistence ownership: Subtask P04A is the sole persistence owner of Schema v6, PrepareBoundDispatch extension, Transaction A, and diagnostic transactions; Subtasks P04B and P04C remain strictly pure in-memory collectors with zero SQLite writes",
    "No implementation of Subtasks P04B, P04C, or P04D: zero Git collector, snapshot extractor, AppContainer sandbox, verification leases, Transaction B, Transaction C, or ReviewBundle synthesis code permitted in this contract",
    "Zero live AO integration: all coordinator tests must use in-memory mock adapters; no live external network or daemon calls",
    "Fail-closed Win32 identity: directory handles must be opened with FILE_FLAG_BACKUP_SEMANTICS, omitting FILE_SHARE_DELETE to block directory rename/delete; 128-bit FileIdInfo and VolumeSerialNumber must be queried and verified; fail closed on non-NTFS/ReFS filesystems",
    "Single coordinator effect gate: Coordinator.Dispatch must hold the live WorkspaceBindingLease across the entire 9-step sequence through /send; zero public pathways accepting snapshot-only to invoke /send",
    "Atomic PrepareBoundDispatch rollback: if any insertion, audit append, or trigger fails, the entire dispatch transaction rolls back leaving zero orphan attempts, operations, bindings, or audits",
    "Pre-send binding failure isolation: on binding guard failure, task state remains DISPATCHED, operation remains DISPATCH_BOUND, and attempt remains open; pre-send diagnostic Variant B records REVIEW_INTEGRITY_CONFLICT, inserts hold INVARIANT_MISMATCH, and CAS invalidates active binding",
    "Clean intake verification: worktree and index must be probed via git status --porcelain=v1 -z --untracked-files=all and git diff-index --quiet HEAD -- with GIT_OPTIONAL_LOCKS=0; dirty state rolls back Transaction A, preserves RUNNING, appends EVIDENCE_COLLECTION_FAILED, and inserts hold DIRTY_WORKTREE_DETECTED",
    "Deterministic hold descriptors: hold IDs and rejection audit event IDs must be derived via RFC 8785 JCS canonicalization of Descriptors A and B without self-referential preimage dependencies",
    "No enabling of AUTOMATIC_RESTORE: automatic restore remains fail-closed; operator restore and linked stop remain disabled until verified operator principal reaches trusted boundary",
    "All tests must pass under the Go race detector with -race -count=1"
  ],
  "acceptance_criteria": [
    "AC-P04A-01: Schema Migration v6 applies cleanly on fresh and historical v5 SQLite databases, creating attempt_workspace_bindings, worker_claims, and review_integrity_holds with partial unique active index and strict foreign keys",
    "AC-P04A-02: Migration v6 rollback atomicity verified: intentional mid-migration failure cleanly rolls back leaving database at v5 schema with zero orphaned v6 objects",
    "AC-P04A-03: Lineage triggers (trg_attempt_workspace_bindings_lineage_guard, trg_worker_claims_lineage_guard, trg_review_integrity_holds_lineage_guard) reject mismatched (task_id, contract_id) references against task_attempts and require DISPATCH_BOUND for bindings",
    "AC-P04A-04: CAS triggers (trg_attempt_workspace_bindings_cas_guard, trg_review_integrity_holds_cas_guard) enforce valid forward state transitions and reject invalid downgrades or modifications to resolved holds",
    "AC-P04A-05: Immutability triggers reject DELETE operations on attempt_workspace_bindings, worker_claims, and review_integrity_holds, and reject UPDATE operations on worker_claims",
    "AC-P04A-06: Host WorkspaceBindingAuthority acquires directory handles omitting FILE_SHARE_DELETE; OS returns ERROR_SHARING_VIOLATION on concurrent directory rename or deletion attempts while lease is active",
    "AC-P04A-07: Win32 128-bit physical identity capture verified via GetFileInformationByHandleEx(FileIdInfo); directory substitution probes (junctions, symlinks, subst, rename) fail physical identity verification",
    "AC-P04A-08: WorkspaceBindingLease Revalidate() detects handle invalidation, path changes, and identity mismatch; Close() idempotently releases both handles",
    "AC-P04A-09: PrepareBoundDispatch store seam atomically claims task READY -> DISPATCHED, allocates task_attempts with session snapshot, inserts dispatch_operations (DISPATCH_BOUND), inserts attempt_workspace_bindings (ACTIVE), and appends TASK_DISPATCH_BOUND and WORKSPACE_BINDING_CREATED audit events in a single transaction",
    "AC-P04A-10: Single coordinator effect gate in Coordinator.Dispatch maintains open WorkspaceBindingLease across fresh AO status revalidation and all 14 pre-send guards before issuing /send",
    "AC-P04A-11: Pre-send binding rejection executes diagnostic Variant B transaction: appends REVIEW_INTEGRITY_CONFLICT (conflict_source = 'WORKSPACE_BINDING_GUARD', attempted_reason = literal, colliding_event_id absent), inserts hold INVARIANT_MISMATCH, CAS invalidates binding, and preserves task DISPATCHED and attempt open",
    "AC-P04A-12: Governed terminalization verified: binding integrity failure requires operator terminalization DISPATCHED -> FAILED (recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE') before hold resolution; retries allocate a fresh TaskAttempt",
    "AC-P04A-13: Pre-intake cleanliness probe invokes git status -z and git diff-index HEAD with GIT_OPTIONAL_LOCKS=0; clean worktree permits Transaction A execution",
    "AC-P04A-14: Transaction A validates worker report, canonicalizes JSON via RFC 8785 JCS, persists worker_claims with verbatim reported_head_sha (length 7..40), CAS advances binding to RETAINED_FOR_VERIFICATION, and CAS advances task to REPORT_READY",
    "AC-P04A-15: Dirty report intake cleanly rolls back Transaction A, preserves task state RUNNING, appends EVIDENCE_COLLECTION_FAILED audit event, and inserts ACTIVE hold DIRTY_WORKTREE_DETECTED via deterministic Descriptors A and B",
    "AC-P04A-16: Zero orphaned audit events or partial holds on transaction rollback across all dispatch and intake error pathways",
    "AC-P04A-17: Full test suite passes under go test -race -count=1 ./... with zero data races"
  ],
  "verification_requests": [
    {
      "id": "VR-P04A-MIGRATE",
      "profile_id": "antigravity-standard",
      "parameters": {
        "command": "go test -v -race -run TestMigrationV6 ./internal/store/..."
      }
    },
    {
      "id": "VR-P04A-WBA-PROBES",
      "profile_id": "antigravity-standard",
      "parameters": {
        "command": "go test -v -race -run TestWorkspaceBindingAuthority ./internal/host/..."
      }
    },
    {
      "id": "VR-P04A-DISPATCH-SEAM",
      "profile_id": "antigravity-standard",
      "parameters": {
        "command": "go test -v -race -run TestPrepareBoundDispatch ./internal/store/..."
      }
    },
    {
      "id": "VR-P04A-COORDINATOR-GUARDS",
      "profile_id": "antigravity-standard",
      "parameters": {
        "command": "go test -v -race -run TestCoordinatorDispatchGuards ./internal/dispatch/..."
      }
    },
    {
      "id": "VR-P04A-INTAKE-TRANSACTIONS",
      "profile_id": "antigravity-standard",
      "parameters": {
        "command": "go test -v -race -run TestReportIntake ./internal/store/..."
      }
    },
    {
      "id": "VR-P04A-FULL-RACE",
      "profile_id": "antigravity-standard",
      "parameters": {
        "command": "go test -race -count=1 ./cmd/... ./internal/... ./test/..."
      }
    }
  ],
  "required_evidence": [
    "git_diff_allowed_scope_only",
    "test_log_migration_v6_fresh_and_upgrade",
    "test_log_wba_win32_substitution_and_anti_rename",
    "test_log_prepare_bound_dispatch_atomicity",
    "test_log_coordinator_14_guards_and_variant_b",
    "test_log_report_intake_clean_and_dirty_transactions",
    "test_log_full_suite_race_exit_zero"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Any test failure under go test -race -count=1",
    "Any schema migration defect or foreign key / trigger violation",
    "Any failure of Win32 128-bit FileIdInfo query or anti-rename handle locks",
    "Any modification to files outside allowed_scope or inside forbidden_scope",
    "Any attempt to implement Subtask P04B, P04C, or P04D capabilities",
    "Any live external network or AO communication detected during testing",
    "Any violation of task state machine invariants or compound state usage",
    "Any failure to preserve RUNNING state during dirty report intake"
  ]
}
```

---

## 2. Governance Baseline & Pre-Execution Directives

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
