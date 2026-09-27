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
    "No Git process execution or collector implementation: Subtask P04A consumes a typed GitEvidenceResult from an injected interface and exercises clean/dirty behavior using test fakes; running git commands (git status, git diff-index), allowlist enforcement, and snapshot extraction belong strictly to Subtask P04B",
    "No implementation of Subtasks P04B, P04C, or P04D: zero Git collector, snapshot extractor, AppContainer sandbox, verification leases, Transaction B, Transaction C, or ReviewBundle synthesis code permitted in this contract",
    "Zero live AO integration: all coordinator tests must use in-memory mock adapters; no live external network or daemon calls",
    "Fail-closed Win32 identity: directory handles must be opened with FILE_FLAG_BACKUP_SEMANTICS, omitting FILE_SHARE_DELETE to block directory rename/delete; 128-bit FileIdInfo and VolumeSerialNumber must be queried and verified; fail closed on non-NTFS/ReFS filesystems",
    "Single coordinator effect gate: Coordinator.Dispatch must hold the live WorkspaceBindingLease across the entire 9-step sequence through /send; zero public pathways accepting snapshot-only to invoke /send",
    "Atomic PrepareBoundDispatch rollback: if any insertion, audit append, or trigger fails, the entire dispatch transaction rolls back leaving zero orphan attempts, operations, bindings, or audits",
    "Pre-send binding failure isolation: on binding guard failure, task state remains DISPATCHED, operation remains DISPATCH_BOUND, and attempt remains open; pre-send diagnostic Variant B records REVIEW_INTEGRITY_CONFLICT, inserts hold INVARIANT_MISMATCH, and CAS invalidates active binding",
    "Deterministic hold descriptors: hold IDs and rejection audit event IDs must be derived via RFC 8785 JCS canonicalization of Descriptors A and B without self-referential preimage dependencies; exact replay returns existing row; recurrence after resolution assigns occurrence N+1",
    "Zero unapproved audit event literals: no creation of new event literals (WORKER_REPORT_INGESTED is strictly forbidden); all events must exist in accepted ADRs",
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
    "AC-P04A-10: Single coordinator effect gate in Coordinator.Dispatch maintains open WorkspaceBindingLease across fresh AO status revalidation and all 14 pre-send guards before issuing /send",
    "AC-P04A-11: Pre-send binding rejection executes diagnostic Variant B transaction: appends REVIEW_INTEGRITY_CONFLICT (conflict_source = 'WORKSPACE_BINDING_GUARD', attempted_reason = literal, colliding_event_id absent), inserts hold INVARIANT_MISMATCH, CAS invalidates binding, and preserves task DISPATCHED and attempt open",
    "AC-P04A-12: Governed terminalization verified: binding integrity failure requires operator terminalization DISPATCHED -> FAILED (recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE') before hold resolution; retries allocate a fresh TaskAttempt",
    "AC-P04A-13: Ingestion interface seam accepts typed GitEvidenceResult without process execution; behavior tests verify that clean result permits Transaction A whereas dirty result triggers intake diagnostic transaction using test fakes",
    "AC-P04A-14: Transaction A validates worker report, canonicalizes JSON via RFC 8785 JCS, persists worker_claims with verbatim reported_head_sha (length 7..40), CAS advances binding to RETAINED_FOR_VERIFICATION, and CAS advances task to REPORT_READY without unapproved audit events",
    "AC-P04A-15: Dirty report intake cleanly rolls back Transaction A, preserves task state RUNNING, appends EVIDENCE_COLLECTION_FAILED audit event, and inserts ACTIVE hold DIRTY_WORKTREE_DETECTED via deterministic Descriptors A and B; exact replay returns existing row, and recurrence after resolution allocates occurrence N+1",
    "AC-P04A-16: Zero orphaned audit events or partial holds on transaction rollback across all dispatch and intake error pathways",
    "AC-P04A-17: All verification requests declare profile_id go-test-p04-001, cwd \".\", timeout_seconds <= 300, and typed package/flags parameters; pass TaskContractValidator.ValidateRaw with catalog metadata and reject all 6 negative probes"
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

Pursuant to ADR-013 and Finding `P04A-C1-004`, host verification profile `go-test-p04-001` is defined by the host environment catalog as follows:

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
          "./internal/domain/...",
          "./internal/host/...",
          "./internal/store/...",
          "./internal/dispatch/..."
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
1. **Unknown Profile Probe**: Requesting any `profile_id` not registered in catalog (e.g. `unknown-profile`, `antigravity-standard` in verification requests) is rejected.
2. **Timeout Boundary Probe**: Requesting `timeout_seconds = 301` (> 300) is rejected by `TaskContractValidator`.
3. **Flag Injection Probe**: Including flags such as `-exec` or unauthorized arguments violates `const: ["-v", "-race", "-count=1"]` and is rejected.
4. **Package Enumeration Probe**: Requesting packages outside the enum (e.g. `./cmd/...` or `./test/...`) is rejected.
5. **Cwd Traversal Probe**: Specifying relative escape paths (e.g. `../escape`) is rejected by `CwdPolicy: worktree_root`.
6. **Extra Property Probe**: Supplying unauthorized parameters (e.g. `"command": "..."`) violates `additionalProperties: false` and is rejected.

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
