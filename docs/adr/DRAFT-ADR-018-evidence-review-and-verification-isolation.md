# ADR-018: Evidence & Review Engine Architecture, Execution Isolation, and Verification Governance

> **Status**: `DRAFT_PENDING_EXTERNAL_APPROVAL (REVISION 15)`
> **Date**: 2026-09-26
> **Audited Baseline**: `5fa0f5d291cb5f1952f688c971b8b047d3529bca`
> **Preservation Baseline Commit**: `6e1993da150031a9465901a7019c71257de44312` (Revision 11)
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_14`
> **Deciders**: AI Engineering Supervisor Architecture Council, External Supervisor
> **Related Architecture**: `docs/04_ARCHITECTURE.md` (Section 7), `docs/05_DOMAIN_MODEL.md`, `docs/10_REVIEW_BUNDLE.md`
> **Related Requirements**: `docs/02_REQUIREMENTS.md` (FR-008, NFR-008 via PROPOSAL-P04-002)
> **Supersedes**: `DRAFT-ADR-018` Revision 14
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R14-001` and `P04-ARCH-R14-002` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_013.md`).

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine**, providing independent, tamper-proof verification of AI worker outputs under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`).

External Re-Audit 013 (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_013.md`) evaluated Round 13 remediation at commit `5fa0f5d291cb5f1952f688c971b8b047d3529bca`, recording verdict `REVISION_14_REQUIRED`. It confirmed design-level closure on `P04-ARCH-R13-001` (TaskState preservation) and `P04-ARCH-R13-002` (foreign key audit references and principal constraints), partially closed `P04-ARCH-R13-003` and `P04-ARCH-R13-004`, and opened findings `P04-ARCH-R14-001` (canonical audit model alignment & replay semantics) and `P04-ARCH-R14-002` (diagnostic occurrence lifecycle).

Revision 15 establishes complete design-level resolution of findings `P04-ARCH-R14-001` and `P04-ARCH-R14-002` via surgical patches applied directly to the Revision 14 architecture:
1. `P04-ARCH-R14-001`: Canonical Audit Model Alignment & Replay Semantics. Strict adherence to canonical `domain.AuditEvent` and `audit_events` schema (`actor TEXT NOT NULL`, role in `details_json.actor_role`, mandatory `pair_id`). System events use `actor = 'ai-supervisor-daemon'`, `details_json.actor_role = 'SUPERVISOR'`; operator resolution uses `actor = verified resolved_by_principal`, `details_json.actor_role = 'OPERATOR'`. Semantic replay upon duplicate key on UNIQUE `event_id` compares sanitized fields (`event_type`, `pair_id`, `task_id`, `contract_id`, `attempt_id`, `actor`, and canonical `details_json`), explicitly ignoring only `sequence`, `timestamp`, `prev_hash`, and `event_hash`. Incoming events are sanitized via `audit.SanitizeAuditEvent` before append and readback comparison; sanitization failure or sanitized-key collision fails closed.
2. `P04-ARCH-R14-002`: Diagnostic Occurrence Lifecycle in `review_integrity_holds`. Added `occurrence_number INTEGER NOT NULL CHECK (typeof(occurrence_number) = 'integer' AND occurrence_number > 0)` and `UNIQUE(attempt_id, hold_reason, diagnostic_fingerprint, occurrence_number)`. Rejection audit event JCS identity descriptor includes `occurrence_number` and `hold_id`. Atomic creation algorithm assigns `occurrence_number = COALESCE(MAX(occurrence_number), 0) + 1` in `BEGIN IMMEDIATE` diagnostic transaction. If the same diagnostic recurs after occurrence N is RESOLVED, occurrence N+1 is created with a new distinct rejection audit event and new ACTIVE hold; hold N remains immutable; admission remains closed.

### Revision 11 to Revision 15 Preservation Matrix
| Revision 11 Section | Revision 15 Section & Location | Status & Surgical Changes |
| :--- | :--- | :--- |
| Section 1: Context & Problem Statement | Section 1: Context & Problem Statement | Preserved; updated with Re-Audit 013 findings and baseline tracking. |
| Section 2: Decision Drivers | Section 2: Decision Drivers | Preserved verbatim. |
| Section 3: Considered Options | Section 3: Considered Options | Preserved verbatim. |
| Section 4, Decision 1: Worktree Authority & Schema v6 | Section 4, Decision 1 | Preserved full Schema v6 (`terminal_generation TEXT`, canonical worktree, hex checks, triggers, integer typing, canonical WorkerClaim array checks). |
| Section 4, Decision 2: Hardened Git Collector & Allowlist | Section 4, Decision 2 | Preserved clean worktree policy, 10-command allowlist, snapshot extraction; updated rejection audit event with canonical audit model, `occurrence_number` & `hold_id` in JCS descriptor (`P04-ARCH-R14-001`, `R14-002`). |
| Section 4, Decision 3: Windows Verification Isolation | Section 4, Decision 3 | Preserved AppContainer handle list, Job Object assignment/limits, authoritative process-death proof requirement. |
| Section 4, Decision 4: Verification Authority & Latency | Section 4, Decision 4 | Preserved ReviewBundle assembly diagnostic; confirmed NFR-008 UNVERIFIED status; commit_duration_ms decoupled from Tx C. |
| Section 4, Decision 5: Pipeline Transactions & Fencing | Section 4, Decision 5 | Preserved 3 pipeline transactions; linear lease chain; preserved replay matrix; updated `review_integrity_holds` DDL with `occurrence_number`, unique occurrence constraint, CAS guard, and multi-occurrence lifecycle algorithm (`P04-ARCH-R14-002`). |
| Section 4, Decision 6: Artifact Store & Schema v9 | Section 4, Decision 6 | Preserved artifact streaming and ReviewBundle DDL; updated canonical event ID derivation with `occurrence_number` & `hold_id`, aligned with canonical `audit_events` schema (`actor`, mandatory `pair_id`, UNIQUE `event_id` conflict, `audit.SanitizeAuditEvent` pipeline) (`P04-ARCH-R14-001`, `R14-002`). |
| Section 5: Consequences | Section 5: Consequences | Preserved and updated with multi-occurrence diagnostic lifecycle and canonical audit model alignment. |
| Section 6: Migration & Schema Ownership | Section 6: Migration & Schema Ownership | Preserved and updated to include Schema v9 `review_integrity_holds` occurrence lifecycle. |

---

## 2. Decision Drivers

1. **Evidence Integrity & Non-Repudiation**: The supervisor must independently corroborate worker claims using immutable source snapshots and sandboxed execution without trusting worker memory or worker assertions.
2. **Canonical Schema Alignment & Lineage Guarantees**: All persistence contracts must enforce task, attempt, and contract lineage via composite foreign keys and SQLite triggers, maintaining strict compliance with `worker-report.schema.json`, `review-bundle.schema.json`, and `docs/06_WORKFLOW_STATE_MACHINE.md`.
3. **Model 1 Persistence Ownership**: Subtasks P04B and P04C must remain strictly in-memory collectors with zero SQLite writes, concentrating all Schema v9 mutations, leases, and CAS transactions in Subtask P04D.
4. **Crash Consistency & Deadlock Elimination**: Database transactions and latency metrics must support crash recovery across daemon restarts without permanent deadlocks.
5. **Clean Worktree Guarantee**: Worker outputs must be fully committed; dirty staged, unstaged, or untracked state must fail closed.
6. **Execution Containment**: Verification tests must run under OS-level isolation (Windows AppContainers) with explicit handle inheritance and network denial.

---

## 3. Considered Options

- **Option 1**: Monolithic in-process execution directly in worker worktree. (Rejected: Vulnerable to TOCTOU, handle leaks, network escape, and worktree contamination).
- **Option 2**: Ephemeral Git branches in worker repository without AppContainer sandboxing. (Rejected: Test execution could compromise host OS; non-isolated processes lack network denial).
- **Option 3 (Selected - Model 1 Architecture)**: Decoupled verification pipeline with Windows AppContainer isolation, deterministic source snapshot extraction, strict clean worktree enforcement, pure in-memory collectors (P04B, P04C), and centralized CAS orchestration in P04D.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority, Immutable Bindings & Canonical WorkerClaim (P04-ARCH-R10-001)

1. **Two-Transaction Workspace Lifecycle**:
   - **Binding Creation (P04A)**: Binds `attempt_id` to the physical worktree (`FileIdInfo`) and linked gitdir identity prior to dispatch, in state `ACTIVE`.
   - **Report Intake Transaction A (P04A)**: Triggered upon worker completion. Validates worktree identity, verifies clean worktree/index via porcelain check, validates `WorkerReport` via JSON schema in Go, canonicalizes via RFC 8785 JCS, inserts into `worker_claims`, updates binding to `RETAINED_FOR_VERIFICATION` via CAS, and transitions `tasks.state`: `RUNNING -> REPORT_READY`.
2. **Verbatim Head SHA & Lineage Invariants**:
   - `worker_claims.reported_head_sha` stores the exact reported value (`CHECK (LENGTH(reported_head_sha) BETWEEN 7 AND 40 AND NOT (reported_head_sha GLOB '*[^0-9a-f]*'))`).
   - The verified `actual_head_sha` belongs strictly to Evidence (`evidence_sets.git_evidence_json`), never in `worker_claims`.
   - `worker_claims` includes `contract_id` with composite lineage triggers and immutable update/delete triggers.
3. **Canonical Mapping**:
   - `WorkerReport.head_sha` -> `WorkerClaim.reported_head_sha`
   - `WorkerReport.files_changed` -> `WorkerClaim.claimed_files_changed`
   - `WorkerReport.tests` -> `WorkerClaim.claimed_tests`
   - `WorkerReport.worker_claims` (array of strings) -> `WorkerClaim.textual_claims`
   - `WorkerReport.build_status` -> `WorkerClaim.build_status`
   - Canonical `WorkerClaim` object is embedded directly into ReviewBundle `worker_claims`.

```sql
-- Schema v6: attempt_workspace_bindings (Owned by Subtask P04A)
CREATE TABLE attempt_workspace_bindings (
    attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    session_id TEXT NOT NULL CHECK (LENGTH(session_id) > 0),
    terminal_generation TEXT NOT NULL CHECK (LENGTH(terminal_generation) > 0),
    canonical_worktree_path TEXT NOT NULL CHECK (LENGTH(canonical_worktree_path) > 0),
    volume_serial_hex TEXT NOT NULL CHECK (
        LENGTH(volume_serial_hex) = 16 AND NOT (volume_serial_hex GLOB '*[^0-9a-f]*')
    ),
    file_id_hex TEXT NOT NULL CHECK (
        LENGTH(file_id_hex) = 32 AND NOT (file_id_hex GLOB '*[^0-9a-f]*')
    ),
    linked_gitdir_path TEXT NOT NULL CHECK (LENGTH(linked_gitdir_path) > 0),
    linked_gitdir_volume_serial_hex TEXT NOT NULL CHECK (
        LENGTH(linked_gitdir_volume_serial_hex) = 16 AND NOT (linked_gitdir_volume_serial_hex GLOB '*[^0-9a-f]*')
    ),
    linked_gitdir_file_id_hex TEXT NOT NULL CHECK (
        LENGTH(linked_gitdir_file_id_hex) = 32 AND NOT (linked_gitdir_file_id_hex GLOB '*[^0-9a-f]*')
    ),
    pinned_ao_commit TEXT NOT NULL CHECK (
        LENGTH(pinned_ao_commit) = 40 AND NOT (pinned_ao_commit GLOB '*[^0-9a-f]*')
    ),
    binding_state TEXT NOT NULL CHECK (
        binding_state IN ('ACTIVE', 'RETAINED_FOR_VERIFICATION', 'RELEASED', 'INVALIDATED')
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
    released_at_epoch_ms INTEGER NULL CHECK (
        released_at_epoch_ms IS NULL OR (
            typeof(released_at_epoch_ms) = 'integer' AND released_at_epoch_ms >= created_at_epoch_ms
        )
    ),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        (binding_state IN ('ACTIVE', 'RETAINED_FOR_VERIFICATION') AND released_at_epoch_ms IS NULL) OR
        (binding_state IN ('RELEASED', 'INVALIDATED') AND released_at_epoch_ms IS NOT NULL)
    )
);

CREATE TRIGGER trg_attempt_workspace_bindings_lineage_guard
BEFORE INSERT ON attempt_workspace_bindings
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id, contract_id in task_attempts or session in dispatch_operations')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        JOIN dispatch_operations d ON d.attempt_id = a.attempt_id
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
          AND d.session_id = NEW.session_id
          AND d.terminal_generation = NEW.terminal_generation
    );
END;

CREATE TRIGGER trg_attempt_workspace_bindings_cas_guard
BEFORE UPDATE ON attempt_workspace_bindings
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in attempt_workspace_bindings')
    WHERE NEW.attempt_id != OLD.attempt_id
       OR NEW.task_id != OLD.task_id
       OR NEW.contract_id != OLD.contract_id
       OR NEW.session_id != OLD.session_id
       OR NEW.terminal_generation != OLD.terminal_generation
       OR NEW.canonical_worktree_path != OLD.canonical_worktree_path
       OR NEW.volume_serial_hex != OLD.volume_serial_hex
       OR NEW.file_id_hex != OLD.file_id_hex
       OR NEW.linked_gitdir_path != OLD.linked_gitdir_path
       OR NEW.linked_gitdir_volume_serial_hex != OLD.linked_gitdir_volume_serial_hex
       OR NEW.linked_gitdir_file_id_hex != OLD.linked_gitdir_file_id_hex
       OR NEW.pinned_ao_commit != OLD.pinned_ao_commit
       OR NEW.created_at_epoch_ms != OLD.created_at_epoch_ms;

    SELECT RAISE(ABORT, 'illegal state transition in attempt_workspace_bindings')
    WHERE NOT (
        (OLD.binding_state = 'ACTIVE' AND NEW.binding_state IN ('RETAINED_FOR_VERIFICATION', 'RELEASED', 'INVALIDATED')) OR
        (OLD.binding_state = 'RETAINED_FOR_VERIFICATION' AND NEW.binding_state IN ('RELEASED', 'INVALIDATED'))
    );
END;

CREATE TRIGGER trg_attempt_workspace_bindings_no_delete
BEFORE DELETE ON attempt_workspace_bindings
BEGIN
    SELECT RAISE(ABORT, 'attempt_workspace_bindings is immutable');
END;

-- Schema v6: worker_claims (Owned by Subtask P04A)
CREATE TABLE worker_claims (
    claim_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    reported_head_sha TEXT NOT NULL CHECK (
        LENGTH(reported_head_sha) BETWEEN 7 AND 40
        AND NOT (reported_head_sha GLOB '*[^0-9a-f]*')
    ),
    payload_json TEXT NOT NULL CHECK (
        json_valid(payload_json) = 1
        AND json_type(payload_json, '$.claimed_files_changed') = 'array'
        AND json_type(payload_json, '$.tests') = 'array'
        AND json_type(payload_json, '$.textual_claims') = 'array'
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_worker_claims_lineage_guard
BEFORE INSERT ON worker_claims
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );
END;

CREATE TRIGGER trg_worker_claims_no_update
BEFORE UPDATE ON worker_claims
BEGIN
    SELECT RAISE(ABORT, 'worker_claims is immutable');
END;

CREATE TRIGGER trg_worker_claims_no_delete
BEFORE DELETE ON worker_claims
BEGIN
    SELECT RAISE(ABORT, 'worker_claims is immutable');
END;
```

### Decision 2: Hardened In-Memory Git Collector & Clean Worktree Allowlist (P04-ARCH-R10-001)

1. **Clean Worktree Policy (v1) & Failure Isolation (P04-ARCH-R13-001, P04-ARCH-R13-002, P04-ARCH-R14-001, P04-ARCH-R14-002)**:
   - Worker must commit all changes; workspace worktree and index must be clean at report intake.
   - Transaction A rolls back if staged, unstaged, or untracked changes exist; task state remains `RUNNING` preserved (strictly zero blanket transitions to `BLOCKED`).
   - In a separate diagnostic transaction executed after rollback: appends rejection audit event `EVIDENCE_COLLECTION_FAILED` to `audit_events` first (`actor = 'ai-supervisor-daemon'`, `details_json.actor_role = 'SUPERVISOR'`), then inserts an ACTIVE hold row into `review_integrity_holds` (`hold_reason = 'DIRTY_WORKTREE_DETECTED'`) in the same diagnostic transaction. If any step fails, the entire diagnostic transaction rolls back.
   - The rejection audit event ID is computed deterministically via RFC 8785 JCS: `event_id = SHA256(RFC8785_JCS(identity_descriptor))` using `version=1`, `event_type='EVIDENCE_COLLECTION_FAILED'`, `pair_id`, `task_id`, `attempt_id`, `contract_id`, `reason='DIRTY_WORKTREE_DETECTED'`, `occurrence_number`, `hold_id`, and `sanitized_input_fingerprint` (strictly omitting raw host paths and secrets).
   - If the diagnostic transaction rolls back, fail-closed without claiming the audit event or hold was recorded.
2. **Hardened Allowlist**:
   - `git status --porcelain=v1 -z --untracked-files=all`: Evaluates clean worktree and untracked files.
   - `git diff-index --quiet HEAD --`: Evaluates clean index state.
   - `git rev-parse --verify --quiet <ref>^{commit}`: Resolves commit SHA.
   - `git diff --raw -z --no-renames --no-ext-diff <base> <head> --`: Parses modified files.
   - `git diff --numstat --no-renames <base> <head> --`: Computes diff statistics.
   - `git rev-list --count <base>..<head>`: Validates commit distance.
   - `git ls-tree -rz --full-tree <head>`: Walks commit tree for snapshot manifest.
   - `git cat-file --batch`: Streams file contents for snapshot extraction.
   - `git rev-parse --git-path index`: Resolves linked worktree index file.
   - `git rev-parse --absolute-git-dir`: Resolves primary repository git directory (restored).
3. **Execution Environment**:
   - Strictly isolated environment variables (`GIT_DIR`, `GIT_WORK_TREE`, `GIT_OPTIONAL_LOCKS=0`, clean system environment, no user config).

### Decision 3: Windows Verification Isolation Security Boundary & Job Containment

1. **Explicit Handle Inheritance via Attribute Lists**:
   - `bInheritHandles = TRUE` in `CreateProcessW`.
   - `STARTUPINFOEXW` configures `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` passing *only* the specific write handles for stdout, stderr, and the read handle for stdin. All database and network handles remain non-inherited.
2. **Atomic Job Object Assignment**:
   - `PROC_THREAD_ATTRIBUTE_JOB_LIST` atomically assigns the child process to a Windows Job Object upon creation.
   - `JOB_OBJECT_LIMIT_BREAKAWAY_OK` is explicitly omitted, preventing subprocess escape.
   - `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` is configured, guaranteeing process termination if the daemon closes the handle or crashes.
3. **Authoritative Process-Death Proof for Lease Reclaim**:
   - TTL expiration alone is insufficient to reclaim a verification lease. The engine must possess authoritative proof of worker process termination:
     (a) Within the running daemon, the verification runner joins the Job Object / process handle and confirms termination (`GetExitCodeProcess`).
     (b) Across daemon restart, host exclusivity (ADR-017 / P03-004 machine lock) combined with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` guarantees prior processes died with the previous host process.
   - If authoritative proof cannot be established, the engine must fail-closed and refuse to allocate a successor lease.
4. **AppContainer Isolation**:
   - Ephemeral AppContainer SID generated per verification attempt.
   - Read-only DACLs granted to snapshot directory; read-write DACLs granted only to ephemeral scratch dir.
   - Network capabilities omitted entirely.

### Decision 4: Verification Authority & Crash-Safe Latency Semantics via PROPOSAL-P04-002 (P04-ARCH-R11-001)

1. **ReviewBundle Assembly Diagnostic Interval**:
   - `evidence_finalized_at_epoch_ms`: Durably committed timestamp from Transaction B.
   - `bundle_assembled_at_epoch_ms`: Timestamp captured when ReviewBundle JSON assembly, RFC 8785 JCS canonicalization, and schema validation succeed, immediately before initiating Transaction C commit.
   - `compilation_latency_ms = bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms`: Designated strictly as an in-process synthesis assembly diagnostic interval, separate from canonical NFR-008.
   - Canonical NFR-008 compliance status is held strictly as `UNVERIFIED` in Schema v9 (`nfr008_compliance_status TEXT NOT NULL CHECK (nfr008_compliance_status = 'UNVERIFIED')`).
2. **Telemetry Decoupling & Deadlock Elimination**:
   - The proposed audit event `REVIEW_BUNDLE_GENERATED` emitted in Transaction C does NOT contain `commit_duration_ms`.
   - `commit_duration_ms` is measured post-commit purely as best-effort in-process monotonic telemetry.
   - Eliminates database deadlocks during recovery; Transaction C advances task state to `REVIEWING`.

### Decision 5: Three Durable Pipeline Transactions, Admission & Bounded Fencing (P04-ARCH-R10-002, R10-004, R10-005)

1. **Model 1 Persistence Ownership Discipline**:
   - **Subtask P04A**: Owns Schema Migration v6 and Transaction A.
   - **Subtask P04B**: Pure in-memory Git evidence collection; ZERO SQLite writes.
   - **Subtask P04C**: Pure in-memory verification runner; ZERO SQLite writes.
   - **Subtask P04D**: Pipeline Orchestrator and SOLE SQLite CAS persistence orchestrator for Schema Migration v9 (`task_verification_leases`, `evidence_sets`, `review_artifacts`, `review_bundles`), Content-Addressed Store, leases, Transaction B, Transaction C, and audit events.
2. **Lease TTL Bounds & Mathematical Enforcement**:
   - `task_verification_leases.ttl_seconds CHECK (ttl_seconds BETWEEN 1 AND 600)`.
   - `CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000))` enforced in SQLite DDL.
   - CAS state transition trigger enforces valid transitions and fencing token increments on reclaim.
3. **ReviewBundle Replay and Conflict State Matrix (Pure Canonical States)**:
   - Compile failure: Transaction C rolls back, task remains `EVIDENCE_READY`, emits `REVIEW_BUNDLE_COMPILATION_REJECTED`.
   - Idempotent replay: Returns existing bundle, task remains `REVIEWING`.
   - Hash conflict: Tampering conflict, task remains `REVIEWING`, Transaction C rolls back, emits `REVIEW_BUNDLE_COMPILATION_REJECTED` (`BUNDLE_HASH_CONFLICT`), locks automated review decision approval.
   - Database invariant corruption (P04-ARCH-R13-001): Current TaskState is preserved (strictly zero transitions to `BLOCKED`); active Transaction C rolls back; in a separate diagnostic transaction, appends rejection audit event `REVIEW_BUNDLE_COMPILATION_REJECTED` (`INVARIANT_MISMATCH`) first, then inserts an ACTIVE hold into `review_integrity_holds` (`hold_reason = 'INVARIANT_MISMATCH'`); admission and automated approval are closed for the attempt; resolution requires authenticated human operator reconciliation.

1. **Linear Lease Chain Design (P04-ARCH-R12-002)**:
   - Eliminates the `RECLAIMED` state and in-place row mutations. Predecessors stay permanently `EXPIRED` with their original `released_at_epoch_ms`.
   - Each lease row references its immutable predecessor via `predecessor_lease_id TEXT NULL UNIQUE REFERENCES task_verification_leases(lease_id)`.
   - First lease: `predecessor_lease_id IS NULL` and `fencing_token = 1`.
   - Successor lease: `predecessor_lease_id IS NOT NULL`, predecessor must match `(task_id, attempt_id, contract_id)`, predecessor state must be `EXPIRED`, and `NEW.fencing_token = predecessor.fencing_token + 1`.
   - Exactly one active lease is enforced by partial unique indexes `WHERE state = 'ACTIVE'`.
   - Terminal CAS: `ACTIVE` only transitions to `COMPLETED`, `EXPIRED`, or `REVOKED`. `released_at_epoch_ms` is written strictly once and can never be rewritten.
   - Transaction B CAS: Verifies exact `(lease_id, worker_id, fencing_token, state='ACTIVE')` along with evidence and `REPORT_READY -> EVIDENCE_READY`.
2. **Durable Integrity Holds & Diagnostic Occurrence Lifecycle (P04-ARCH-R12-003, P04-ARCH-R13-002, P04-ARCH-R13-003, P04-ARCH-R14-002)**:
   - Table `review_integrity_holds` tracks intake rejections, worktree dirtiness, bundle hash conflicts, invariant mismatches, and unverified claims with exact attempt lineage.
   - **Multi-Diagnostic Hold Tracking via Fingerprint**: Each diagnostic violation carries a deterministic `diagnostic_fingerprint` (64-character lowercase hex SHA-256). Multiple distinct `ACTIVE` holds on the same attempt are permitted when `hold_reason` or `diagnostic_fingerprint` differs, tracked via partial unique index `idx_review_integrity_holds_active_dedup` on `(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE'`.
   - **Diagnostic Occurrence Lifecycle (P04-ARCH-R14-002)**: Column `occurrence_number INTEGER NOT NULL CHECK (typeof(occurrence_number) = 'integer' AND occurrence_number > 0)` and `UNIQUE(attempt_id, hold_reason, diagnostic_fingerprint, occurrence_number)` maintain an immutable occurrence history per diagnostic tuple. Both `occurrence_number` and `hold_id` participate in the JCS identity descriptor of the rejection audit event.
   - **Atomic Creation Algorithm**:
     a. `BEGIN IMMEDIATE` diagnostic transaction.
     b. Check if an `ACTIVE` hold already exists with the same `(attempt_id, hold_reason, diagnostic_fingerprint)`. If yes: exact replay returns the existing hold without duplicate insertion.
     c. If no `ACTIVE` hold exists: compute `occurrence_number = COALESCE(MAX(occurrence_number), 0) + 1` for that tuple.
     d. Derive new distinct `hold_id` and rejection `event_id` from the JCS descriptor containing `occurrence_number`.
     e. Append sanitized audit event to `audit_events` (`actor = 'ai-supervisor-daemon'`, `details_json.actor_role = 'SUPERVISOR'`).
     f. Insert `ACTIVE` hold row into `review_integrity_holds`.
     g. Commit transaction.
     h. On unique index or CAS race: read back the `ACTIVE` hold; exact tuple succeeds idempotently, differing tuple fails closed.
   - **Recurrence Lifecycle**: After occurrence N is `RESOLVED`, if the identical diagnostic is re-observed, occurrence N+1 is created with a new distinct rejection audit event and a new `ACTIVE` hold. Hold N and historical audit events remain immutable. Admission remains closed because an `ACTIVE` hold (occurrence N+1) exists.
   - **Audit References & Principal Constraints**: Rejection audit event is foreign-keyed via `rejection_audit_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`. Resolution audit event is foreign-keyed via `resolution_audit_event_id TEXT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`. Resolving principal must satisfy `LENGTH(TRIM(resolved_by_principal)) > 0` when resolved.
   - **Atomic Resolution Transaction**: Resolution requires authenticated human operator principal, appends resolution audit event `REVIEW_INTEGRITY_HOLD_RESOLVED` (`actor = verified resolved_by_principal`, `details_json.actor_role = 'OPERATOR'`), and performs CAS update (`hold_state = 'RESOLVED'`, `resolved_at_epoch_ms = now`, `resolution_audit_event_id = event_id`, `resolved_by_principal = principal` WHERE `hold_id = :hold_id AND hold_state = 'ACTIVE'`) in a single transaction. Each hold occurrence is resolved independently.
   - **Fail-Closed Runtime Dependency**: While `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`, runtime resolution fails closed; fake or test principals cannot resolve holds.
   - **Fail-Closed Pipeline Gates**: All intake, verification (Tx B), compilation (Tx C), and review approval paths fail closed if `EXISTS (SELECT 1 FROM review_integrity_holds WHERE attempt_id = :attempt_id AND hold_state = 'ACTIVE')`.
   - **Startup Recovery**: Scans for `ACTIVE` holds on startup before opening admission.

```sql
-- Schema v9: task_verification_leases (Owned by Subtask P04D)
CREATE TABLE task_verification_leases (
    lease_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (
        typeof(fencing_token) = 'integer' AND fencing_token >= 1 AND fencing_token <= 9223372036854775807
    ),
    state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'COMPLETED', 'EXPIRED', 'REVOKED')),
    worker_id TEXT NOT NULL CHECK (LENGTH(worker_id) > 0),
    ttl_seconds INTEGER NOT NULL CHECK (
        typeof(ttl_seconds) = 'integer' AND ttl_seconds BETWEEN 1 AND 600
    ),
    acquired_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(acquired_at_epoch_ms) = 'integer'
        AND acquired_at_epoch_ms > 0
        AND acquired_at_epoch_ms <= (9223372036854775807 - (ttl_seconds * 1000))
    ),
    expires_at_epoch_ms INTEGER NOT NULL CHECK (typeof(expires_at_epoch_ms) = 'integer'),
    released_at_epoch_ms INTEGER NULL CHECK (
        released_at_epoch_ms IS NULL OR (
            typeof(released_at_epoch_ms) = 'integer' AND released_at_epoch_ms >= acquired_at_epoch_ms
        )
    ),
    predecessor_lease_id TEXT NULL UNIQUE REFERENCES task_verification_leases(lease_id) ON DELETE RESTRICT,
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    UNIQUE(attempt_id, fencing_token),
    CHECK (expires_at_epoch_ms = acquired_at_epoch_ms + (ttl_seconds * 1000)),
    CHECK (predecessor_lease_id IS NULL OR predecessor_lease_id != lease_id),
    CHECK ((predecessor_lease_id IS NULL AND fencing_token = 1) OR (predecessor_lease_id IS NOT NULL AND fencing_token > 1)),
    CHECK (
        (state = 'ACTIVE' AND released_at_epoch_ms IS NULL) OR
        (state IN ('COMPLETED', 'EXPIRED', 'REVOKED') AND released_at_epoch_ms IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_leases_single_active_attempt ON task_verification_leases(attempt_id) WHERE state = 'ACTIVE';
CREATE UNIQUE INDEX idx_leases_single_active_task ON task_verification_leases(task_id) WHERE state = 'ACTIVE';

CREATE TRIGGER trg_task_verification_leases_lineage_guard
BEFORE INSERT ON task_verification_leases
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );

    SELECT RAISE(ABORT, 'predecessor lease mismatch: predecessor must match attempt/contract, be EXPIRED, and fencing_token must be predecessor.fencing_token + 1')
    WHERE NEW.predecessor_lease_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM task_verification_leases p
        WHERE p.lease_id = NEW.predecessor_lease_id
          AND p.task_id = NEW.task_id
          AND p.attempt_id = NEW.attempt_id
          AND p.contract_id = NEW.contract_id
          AND p.state = 'EXPIRED'
          AND p.released_at_epoch_ms IS NOT NULL
          AND NEW.fencing_token = (p.fencing_token + 1)
    );
END;

CREATE TRIGGER trg_task_verification_leases_cas_guard
BEFORE UPDATE ON task_verification_leases
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in task_verification_leases')
    WHERE NEW.lease_id != OLD.lease_id
       OR NEW.task_id != OLD.task_id
       OR NEW.attempt_id != OLD.attempt_id
       OR NEW.contract_id != OLD.contract_id
       OR NEW.worker_id != OLD.worker_id
       OR NEW.fencing_token != OLD.fencing_token
       OR NEW.acquired_at_epoch_ms != OLD.acquired_at_epoch_ms
       OR NEW.ttl_seconds != OLD.ttl_seconds
       OR NEW.expires_at_epoch_ms != OLD.expires_at_epoch_ms
       OR (NEW.predecessor_lease_id IS NOT OLD.predecessor_lease_id);

    SELECT RAISE(ABORT, 'released_at_epoch_ms rewrite forbidden')
    WHERE OLD.released_at_epoch_ms IS NOT NULL AND NEW.released_at_epoch_ms != OLD.released_at_epoch_ms;

    SELECT RAISE(ABORT, 'illegal lease state transition: ACTIVE only transitions to COMPLETED, EXPIRED, or REVOKED')
    WHERE NOT (OLD.state = 'ACTIVE' AND NEW.state IN ('COMPLETED', 'EXPIRED', 'REVOKED'));
END;

CREATE TRIGGER trg_task_verification_leases_no_delete
BEFORE DELETE ON task_verification_leases
BEGIN
    SELECT RAISE(ABORT, 'task_verification_leases is immutable');
END;

-- Schema v9: review_integrity_holds (Owned by Subtask P04D, P04-ARCH-R13-002, P04-ARCH-R13-003, P04-ARCH-R14-002)
CREATE TABLE review_integrity_holds (
    hold_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    hold_reason TEXT NOT NULL CHECK (
        hold_reason IN (
            'DIRTY_WORKTREE_DETECTED',
            'BUNDLE_HASH_CONFLICT',
            'INVARIANT_MISMATCH',
            'UNVERIFIED_CLAIM_DETECTED',
            'SECURITY_POLICY_VIOLATION'
        )
    ),
    hold_state TEXT NOT NULL CHECK (hold_state IN ('ACTIVE', 'RESOLVED')),
    diagnostic_fingerprint TEXT NOT NULL CHECK (
        LENGTH(diagnostic_fingerprint) = 64 AND NOT (diagnostic_fingerprint GLOB '*[^0-9a-f]*')
    ),
    occurrence_number INTEGER NOT NULL CHECK (
        typeof(occurrence_number) = 'integer' AND occurrence_number > 0
    ),
    rejection_audit_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
    resolution_audit_event_id TEXT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
    resolved_by_principal TEXT NULL CHECK (
        resolved_by_principal IS NULL OR LENGTH(TRIM(resolved_by_principal)) > 0
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
    resolved_at_epoch_ms INTEGER NULL CHECK (
        resolved_at_epoch_ms IS NULL OR (
            typeof(resolved_at_epoch_ms) = 'integer' AND resolved_at_epoch_ms >= created_at_epoch_ms
        )
    ),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    UNIQUE(attempt_id, hold_reason, diagnostic_fingerprint, occurrence_number),
    CHECK (
        (hold_state = 'ACTIVE' AND resolved_at_epoch_ms IS NULL AND resolution_audit_event_id IS NULL AND resolved_by_principal IS NULL) OR
        (hold_state = 'RESOLVED' AND resolved_at_epoch_ms IS NOT NULL AND resolution_audit_event_id IS NOT NULL AND resolved_by_principal IS NOT NULL AND LENGTH(TRIM(resolved_by_principal)) > 0)
    )
);

CREATE UNIQUE INDEX idx_review_integrity_holds_active_dedup
ON review_integrity_holds(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE';

CREATE TRIGGER trg_review_integrity_holds_lineage_guard
BEFORE INSERT ON review_integrity_holds
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );
END;

CREATE TRIGGER trg_review_integrity_holds_cas_guard
BEFORE UPDATE ON review_integrity_holds
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'immutable column modified in review_integrity_holds')
    WHERE NEW.hold_id != OLD.hold_id
       OR NEW.task_id != OLD.task_id
       OR NEW.attempt_id != OLD.attempt_id
       OR NEW.contract_id != OLD.contract_id
       OR NEW.hold_reason != OLD.hold_reason
       OR NEW.diagnostic_fingerprint != OLD.diagnostic_fingerprint
       OR NEW.occurrence_number != OLD.occurrence_number
       OR NEW.rejection_audit_event_id != OLD.rejection_audit_event_id
       OR NEW.created_at_epoch_ms != OLD.created_at_epoch_ms;

    SELECT RAISE(ABORT, 'illegal hold state transition: ACTIVE only transitions to RESOLVED')
    WHERE NOT (OLD.hold_state = 'ACTIVE' AND NEW.hold_state = 'RESOLVED');
END;

CREATE TRIGGER trg_review_integrity_holds_no_delete
BEFORE DELETE ON review_integrity_holds
BEGIN
    SELECT RAISE(ABORT, 'review_integrity_holds is immutable');
END;
```

### Decision 6: Durable Content-Addressed Artifact Store, Review Schema & Proposed Audit Events (P04-ARCH-R10-001, R10-004)

1. **Artifact Streaming & Exact Limits**:
   - Separate capture limit (10 MB saved to disk) from hard safety limit (50 MB process termination).
   - Strict mathematical bounds enforced across `stream_state` combinations.
2. **Content-Addressed Storage Layout**:
   - Canonical path exact equality: `canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256`.

```sql
-- Schema v9: evidence_sets (Owned by Subtask P04D)
CREATE TABLE evidence_sets (
    evidence_set_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (
        typeof(fencing_token) = 'integer' AND fencing_token >= 1 AND fencing_token <= 9223372036854775807
    ),
    git_evidence_json TEXT NOT NULL CHECK (
        json_valid(git_evidence_json) = 1 AND
        json_type(git_evidence_json, '$.actual_changed_files') = 'array'
    ),
    test_evidence_json TEXT NOT NULL CHECK (
        json_valid(test_evidence_json) = 1 AND
        json_type(test_evidence_json, '$.executed_commands') = 'array'
    ),
    policy_findings_json TEXT NOT NULL CHECK (json_valid(policy_findings_json) = 1),
    unverified_claims_json TEXT NOT NULL CHECK (json_valid(unverified_claims_json) = 1),
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(evidence_finalized_at_epoch_ms) = 'integer' AND evidence_finalized_at_epoch_ms > 0
    ),
    collected_at TEXT NOT NULL CHECK (LENGTH(collected_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_evidence_sets_lineage_guard
BEFORE INSERT ON evidence_sets
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );
END;

CREATE TRIGGER trg_evidence_sets_no_update
BEFORE UPDATE ON evidence_sets
BEGIN
    SELECT RAISE(ABORT, 'evidence_sets is immutable');
END;

CREATE TRIGGER trg_evidence_sets_no_delete
BEFORE DELETE ON evidence_sets
BEGIN
    SELECT RAISE(ABORT, 'evidence_sets is immutable');
END;

-- Schema v9: review_artifacts (Owned by Subtask P04D)
CREATE TABLE review_artifacts (
    artifact_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    artifact_type TEXT NOT NULL CHECK (
        artifact_type IN ('VERIFICATION_LOG', 'GIT_DIFF', 'TEST_REPORT', 'WORKER_STDOUT', 'WORKER_STDERR')
    ),
    media_type TEXT NOT NULL CHECK (LENGTH(media_type) > 0),
    encoding TEXT NOT NULL CHECK (encoding IN ('identity', 'gzip')),
    canonical_relative_path TEXT NOT NULL,
    captured_sha256 TEXT NOT NULL CHECK (
        LENGTH(captured_sha256) = 64 AND NOT (captured_sha256 GLOB '*[^0-9a-f]*')
    ),
    full_stream_sha256 TEXT NULL CHECK (
        full_stream_sha256 IS NULL OR (
            LENGTH(full_stream_sha256) = 64 AND NOT (full_stream_sha256 GLOB '*[^0-9a-f]*')
        )
    ),
    capture_limit_bytes INTEGER NOT NULL CHECK (
        typeof(capture_limit_bytes) = 'integer' AND capture_limit_bytes > 0
    ),
    hard_safety_limit_bytes INTEGER NOT NULL CHECK (
        typeof(hard_safety_limit_bytes) = 'integer'
        AND hard_safety_limit_bytes > capture_limit_bytes
        AND hard_safety_limit_bytes <= 9223372036854775806
    ),
    captured_bytes INTEGER NOT NULL CHECK (
        typeof(captured_bytes) = 'integer'
        AND captured_bytes >= 0
        AND captured_bytes <= capture_limit_bytes
    ),
    total_observed_bytes INTEGER NOT NULL CHECK (
        typeof(total_observed_bytes) = 'integer'
        AND total_observed_bytes >= captured_bytes
        AND total_observed_bytes <= 9223372036854775807
    ),
    is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1)),
    stream_state TEXT NOT NULL CHECK (
        stream_state IN ('COMPLETE_EOF', 'TRUNCATED_AT_CAPTURE_LIMIT', 'HARD_LIMIT_TERMINATED', 'TIMEOUT_ABORTED')
    ),
    created_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(created_at_epoch_ms) = 'integer' AND created_at_epoch_ms > 0
    ),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256 AND
        canonical_relative_path NOT GLOB '*:*' AND
        canonical_relative_path NOT GLOB '*\*' AND
        canonical_relative_path NOT GLOB '*..*' AND
        canonical_relative_path NOT GLOB '*//*'
    ),
    CHECK (
        (stream_state = 'COMPLETE_EOF' AND is_truncated = 0 AND full_stream_sha256 IS NOT NULL AND full_stream_sha256 = captured_sha256 AND captured_bytes = total_observed_bytes AND total_observed_bytes <= capture_limit_bytes)
        OR (stream_state = 'TRUNCATED_AT_CAPTURE_LIMIT' AND is_truncated = 1 AND full_stream_sha256 IS NOT NULL AND captured_bytes = capture_limit_bytes AND total_observed_bytes > capture_limit_bytes AND total_observed_bytes <= hard_safety_limit_bytes)
        OR (stream_state = 'HARD_LIMIT_TERMINATED' AND is_truncated = 1 AND full_stream_sha256 IS NULL AND captured_bytes = capture_limit_bytes AND total_observed_bytes = hard_safety_limit_bytes + 1)
        OR (stream_state = 'TIMEOUT_ABORTED' AND is_truncated = 1 AND full_stream_sha256 IS NULL AND total_observed_bytes <= hard_safety_limit_bytes AND (
            (total_observed_bytes <= capture_limit_bytes AND captured_bytes = total_observed_bytes) OR
            (total_observed_bytes > capture_limit_bytes AND captured_bytes = capture_limit_bytes)
        ))
    )
);

CREATE TRIGGER trg_review_artifacts_lineage_guard
BEFORE INSERT ON review_artifacts
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match task_id, attempt_id, contract_id in evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.task_id = NEW.task_id
          AND e.attempt_id = NEW.attempt_id
          AND e.contract_id = NEW.contract_id
    );
END;

CREATE TRIGGER trg_review_artifacts_no_update
BEFORE UPDATE ON review_artifacts
BEGIN
    SELECT RAISE(ABORT, 'review_artifacts is immutable');
END;

CREATE TRIGGER trg_review_artifacts_no_delete
BEFORE DELETE ON review_artifacts
BEGIN
    SELECT RAISE(ABORT, 'review_artifacts is immutable');
END;

-- Schema v9: review_bundles (Owned by Subtask P04D)
CREATE TABLE review_bundles (
    bundle_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL UNIQUE REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    bundle_payload_json TEXT NOT NULL CHECK (
        json_valid(bundle_payload_json) = 1 AND
        json_type(bundle_payload_json, '$.worker_claims') = 'object' AND
        json_type(bundle_payload_json, '$.actual_git_evidence') = 'object'
    ),
    bundle_hash TEXT NOT NULL CHECK (
        LENGTH(bundle_hash) = 64 AND NOT (bundle_hash GLOB '*[^0-9a-f]*')
    ),
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(evidence_finalized_at_epoch_ms) = 'integer' AND evidence_finalized_at_epoch_ms > 0
    ),
    bundle_assembled_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(bundle_assembled_at_epoch_ms) = 'integer' AND bundle_assembled_at_epoch_ms >= evidence_finalized_at_epoch_ms
    ),
    compilation_latency_ms INTEGER NOT NULL CHECK (
        typeof(compilation_latency_ms) = 'integer' AND
        compilation_latency_ms >= 0 AND
        compilation_latency_ms = (bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms)
    ),
    latency_measurement_status TEXT NOT NULL CHECK (
        latency_measurement_status IN ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART')
    ),
    nfr008_compliance_status TEXT NOT NULL CHECK (
        nfr008_compliance_status = 'UNVERIFIED'
    ),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_review_bundles_lineage_guard
BEFORE INSERT ON review_bundles
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match task_id, attempt_id, contract_id in evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.task_id = NEW.task_id
          AND e.attempt_id = NEW.attempt_id
          AND e.contract_id = NEW.contract_id
          AND e.evidence_finalized_at_epoch_ms = NEW.evidence_finalized_at_epoch_ms
    );
END;

CREATE TRIGGER trg_review_bundles_no_update
BEFORE UPDATE ON review_bundles
BEGIN
    SELECT RAISE(ABORT, 'review_bundles is immutable');
END;

CREATE TRIGGER trg_review_bundles_no_delete
BEFORE DELETE ON review_bundles
BEGIN
    SELECT RAISE(ABORT, 'review_bundles is immutable');
END;
```

### Canonical Event ID Derivation & Audit Event Replay Semantics (P04-ARCH-R13-004, P04-ARCH-R14-001, P04-ARCH-R14-002)

1. **RFC 8785 JSON Canonicalization Scheme (JCS) Derivation**:
   - String concatenation using colons (`attempt_id || ":" || reason || ...`) is strictly prohibited due to delimiter injection vulnerabilities.
   - All deterministic audit event IDs are derived as:
     `event_id = SHA256(RFC8785_JCS(identity_descriptor))`
   - The `identity_descriptor` JSON object contains the minimum required canonical fields (including `occurrence_number` and `hold_id`):
     ```json
     {
       "attempt_id": "<attempt_id>",
       "contract_id": "<contract_id>",
       "event_type": "<event_type>",
       "hold_id": "<hold_id>",
       "occurrence_number": 1,
       "pair_id": "<pair_id>",
       "reason": "<reason>",
       "sanitized_input_fingerprint": "<64_hex_hash>",
       "task_id": "<task_id>",
       "version": 1
     }
     ```
2. **Sanitized Input Fingerprint Definition**:
   - `sanitized_input_fingerprint` is a 64-character lowercase hex SHA-256 hash of the canonicalized (RFC 8785 JCS) diagnostic input payload.
   - **Path Normalization**: Strictly uses canonical relative paths (e.g. `artifacts/ab/...` or `internal/foo.go`). Raw host-specific local paths (e.g. `C:\Users\...`, `/home/...`, drive letters, UNC shares, or temp directories) are strictly prohibited.
   - **Secret Sanitization**: Explicitly scrubbed to exclude secrets, credentials, API keys, tokens, passwords, private keys, environment blocks, and sensitive Git configurations.
   - **Fail-Closed on Un-Normalized Data**: If sanitization fails or encounters key collisions (`ErrSanitizedKeyCollision`), processing fails closed immediately; fingerprints must never be computed from unnormalized data.
3. **Canonical Audit Model & Replay Comparison on UNIQUE event_id Conflict (P04-ARCH-R14-001)**:
   - In the canonical schema (`audit_events`) and domain model (`domain.AuditEvent`), the actor column is `actor TEXT NOT NULL` (there are NO separate `actor_id` or `actor_role` columns). Role metadata belongs in `details_json.actor_role`.
   - The primary key of `audit_events` is `sequence INTEGER PRIMARY KEY`; event deduplication occurs on the `event_id TEXT UNIQUE NOT NULL` constraint (termed accurately as a **UNIQUE event_id conflict**, not a primary-key conflict).
   - **Sanitization Pipeline Pre-Condition**: Incoming events must pass through `audit.SanitizeAuditEvent` before append and before readback comparison.
   - When `AppendAuditEvent` encounters a duplicate key collision on UNIQUE `event_id`:
     * **Readback**: Read existing event from `audit_events` WHERE `event_id = :event_id`.
     * **Semantic Field Comparison**: Compare exact equality on sanitized canonical fields:
       1. `event_type`
       2. `pair_id` (mandatory!)
       3. `task_id`
       4. `contract_id`
       5. `attempt_id`
       6. `actor`
       7. Canonical `details_json`
     * **Ignored Volatile / Chain Fields**: Explicitly ignore only `sequence`, `timestamp`, `prev_hash`, and `event_hash`.
     * **Exact Match -> Idempotent Success**: If all 7 semantic fields match, return existing audit record without inserting a duplicate.
     * **Mismatch -> Audit Integrity Conflict**: If any semantic field or lineage differs (e.g. `pair_id` mismatch or modified payload), fail closed! Do NOT overwrite or reuse the colliding `event_id`. Initiate an atomic diagnostic transaction:
       a. Derive a NEW distinct `event_id` via RFC 8785 JCS using `event_type = 'REVIEW_INTEGRITY_CONFLICT'`, referencing the colliding event ID and mismatch details in `sanitized_input_fingerprint`.
       b. Append `REVIEW_INTEGRITY_CONFLICT` to `audit_events` (`actor = 'ai-supervisor-daemon'`, `details_json.actor_role = 'SUPERVISOR'`).
       c. Insert an `ACTIVE` hold into `review_integrity_holds` with `hold_reason = 'INVARIANT_MISMATCH'`.
       d. Close admission and automated review approval for the attempt.
4. **Formal Audit Event Registrations**:
   - **`REVIEW_INTEGRITY_CONFLICT`**:
     - *Producer*: Subtask P04A (Report Intake) / Subtask P04D (Pipeline CAS Orchestrator).
     - *Actor Authority*: System supervisor daemon (`actor = 'ai-supervisor-daemon'`, `details_json.actor_role = 'SUPERVISOR'`).
     - *Lineage*: `pair_id`, `task_id`, `contract_id`, `attempt_id`.
     - *Details Schema*: `colliding_event_id` (string), `attempted_event_type` (string), `attempted_reason` (string), `conflict_type` (`'PAYLOAD_MISMATCH'` | `'LINEAGE_MISMATCH'`), `sanitized_input_fingerprint` (64 hex), `diagnostic_fingerprint` (64 hex), `actor_role` (`'SUPERVISOR'`).
     - *Transaction Boundary*: Atomic diagnostic transaction (distinct new `event_id`, appends audit event first, then inserts ACTIVE hold into `review_integrity_holds`).
   - **`REVIEW_INTEGRITY_HOLD_RESOLVED`**:
     - *Producer*: Operator Reconciliation Tool.
     - *Actor Authority*: Authenticated Human Operator Principal (`actor = verified resolved_by_principal`, `details_json.actor_role = 'OPERATOR'`). While `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`, runtime resolution fails closed; test/fake principals cannot resolve holds.
     - *Lineage*: `pair_id`, `task_id`, `contract_id`, `attempt_id`.
     - *Details Schema*: `hold_id` (string), `hold_reason` (string), `diagnostic_fingerprint` (64 hex), `occurrence_number` (integer), `rejection_audit_event_id` (string), `resolved_by_principal` (non-empty, trimmed string), `resolution_rationale` (non-empty string), `actor_role` (`'OPERATOR'`).
     - *Transaction Boundary*: Atomic resolution transaction (appends `REVIEW_INTEGRITY_HOLD_RESOLVED` to `audit_events`, CAS updates `review_integrity_holds` from `ACTIVE` to `RESOLVED`).

---

## 5. Consequences

### Positive
- Strict, kernel-enforced process and network security boundary for all verification commands via Windows AppContainer.
- Complete lineage, immutability, and CAS transition enforcement across all pipeline tables.
- Pure in-memory separation for Subtasks P04B and P04C, eliminating concurrency bugs and orphaned records.
- Deadlock-free crash recovery across daemon restarts with mathematically verified lease and stream constraints.

### Negative / Trade-offs
- Verification execution duration is bounded by real test execution physics rather than sub-3-second expectations (reconciled cleanly via PROPOSAL-P04-002).
- Clean worktree policy v1 requires workers to keep worktree and index completely clean at report time.

---

## 6. Migration and Schema Ownership

- **Schema v6 (Owned by Subtask P04A)**:
  - `attempt_workspace_bindings` table, indexes, and triggers.
  - `worker_claims` table, indexes, and triggers.
- **Schema v9 (Owned by Subtask P04D)**:
  - `task_verification_leases` table, indexes, and triggers.
  - `review_integrity_holds` table, indexes, and triggers.
  - `evidence_sets` table, indexes, and triggers.
  - `review_artifacts` table, indexes, and triggers.
  - `review_bundles` table, indexes, and triggers.
- **Subtasks P04B & P04C**:
  - Pure in-memory execution; zero migration ownership, zero runtime SQLite writes.
