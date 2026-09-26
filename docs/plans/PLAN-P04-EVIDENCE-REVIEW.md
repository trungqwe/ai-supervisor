# PLAN: Phase P04 — Evidence & Review Engine Execution Plan

> **Plan ID**: `PLAN-P04-EVIDENCE-REVIEW`
> **Revision**: 9
> **Status**: `PLANNING_PENDING_EXTERNAL_AUDIT`
> **Date**: 2026-09-26
> **Audited Baseline**: `a6290ce4c444e345e47aace6c488c3cc4ff33a0d`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_8`
> **Supersedes**: `PLAN-P04-EVIDENCE-REVIEW` Revision 8
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R8-001` through `P04-ARCH-R8-005` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_007.md`).
> **Prerequisite Decoupling**: Unblocks Subtask P04A as the first releaseable subtask. The unexecutable proof contract is transitioned to a historical deferred non-blocking track (`RETIRED_NON_EXECUTABLE_DRAFT`). Operational binding validation is embedded directly as `WORKTREE_BINDING_RUNTIME_VALIDATION` in P04A and P04D.

---

## 1. Overview and Phasing Strategy

Phase P04 delivers the **Evidence & Review Engine**, responsible for independent verification of AI worker reports, automated collection of Git diffs and test results, tamper-proof artifact management, and synthesis of attempt-scoped `ReviewBundle` documents.

### 1.1. Execution Flowchart & Subtask Dependency Graph
```mermaid
graph TD
    Gov["Approved Architecture (ADR-018 Accepted)"] --> P04A

    subgraph PhaseP04ExecutionGraph["Phase P04 Core Execution Graph"]
        P04A["Subtask P04A: Dispatch Seam, Intake & Workspace Binding Authority
(Schema v6: attempt_workspace_bindings, worker_claims, Boundary A)"]
        P04B["Subtask P04B: Hardened Git Collector
(In-Memory GitEvidenceResult, Host-Pinned Binary)"]
        P04C["Subtask P04C: Verification Runner & Security Boundary
(In-Memory TestEvidenceResult, Windows AppContainer)"]
        P04D["Subtask P04D: Pipeline Orchestrator, Artifact Store & ReviewBundle
(Schema v9: leases, evidence_sets, artifacts, bundles, Boundaries B & C)"]

        P04A --> P04B
        P04A --> P04C
        P04B --> P04D
        P04C --> P04D
    end

    HistoricalTrack["Historical Deferred Track: Proof Plan & Draft Contract
(DEFERRED_NON_BLOCKING / RETIRED_NON_EXECUTABLE_DRAFT)"]
```

### 1.2. Key Architectural Invariants
1. **P04A Unblocked**: Subtask P04A is the first releaseable task once architecture is approved.
2. **Two Decoupled Transactions for Worktree Binding**:
   - **Dispatch Binding Registration Transaction**: Materialized immediately after Agent Orchestrator prepares the physical worktree, integrated into P03 dispatch seam (`PrepareBoundDispatch` / `DISPATCH_BOUND`) before `RecordSendRequested`.
   - **Transaction A (Report Intake)**: Strictly reads and validates existing physical binding, persists unique `worker_claims` (Schema v6), and transitions `tasks.state`: `RUNNING -> REPORT_READY`. Transaction A never creates bindings.
3. **Inverted Relational Hierarchy & Content-Addressed Paths**:
   - **Transaction B (Evidence Finalization)**: Persists durable `evidence_sets` with `evidence_committed_at_epoch_ms` and `review_artifacts(evidence_set_id)` at exact paths `artifacts/<first-two-hex>/<captured_sha256>`, releases lease, transitions `REPORT_READY -> EVIDENCE_READY`. Evidence persistence is verified prior to state transition.
   - **Transaction C (ReviewBundle Compilation)**: Synthesizes RFC 8785 JCS canonical ReviewBundle, persists `review_bundles(evidence_set_id)` with `bundle_committed_at_epoch_ms` and `compilation_latency_ms`, records proposed audit event `REVIEW_BUNDLE_GENERATED`, transitions `EVIDENCE_READY -> REVIEWING`.
4. **Authority Separation**: Subtasks P04B and P04C execute purely in memory; runtime SQLite write authority belongs exclusively to P04A (Dispatch Seam & Transaction A) and P04D (Transactions B & C).

---

## 2. Decoupling of Historical Proof Track & Runtime Validation (P04-ARCH-R8-001)

1. **Retirement of Prerequisite Gate**: Pre-contract disposable worker runtime proof is inexecutable because upstream Agent Orchestrator (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`, `backend/internal/domain/harness.go`) contains no inert harness in `AllHarnesses`. Modeling the proof contract as an unreleaseable prerequisite created an execution deadlock.
2. **Transition to Historical Deferred Track**:
   - `docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`: Preserved as `HISTORICAL_DEFERRED_NON_BLOCKING`.
   - `docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`: Preserved as `RETIRED_NON_EXECUTABLE_DRAFT` (status invariant: `NOT_RELEASED`).
3. **Operational Runtime Verification**: Physical binding verification is performed during live execution via `WORKTREE_BINDING_RUNTIME_VALIDATION` inside P04A (Dispatch Seam & Intake) and P04D (Verification Orchestrator).

---

## 3. Work Breakdown Structure (Subtasks P04A – P04D)

### 3.1. Subtask P04A: Dispatch Seam, Intake & Workspace Binding Authority
- **Target Task Contract**: `TASK_CONTRACT_P04_001A`
- **Ownership**: Dispatch Binding Seam, Schema v6 Migration (`attempt_workspace_bindings`, `worker_claims`), Transaction Boundary A, OS Handle Management.
- **Deliverables**:
  1. **Dispatch Binding Registration Seam**:
     - Integrated into `internal/dispatch` and `internal/store` at `PrepareBoundDispatch` / `DISPATCH_BOUND` prior to `RecordSendRequested`.
     - Validates physical worktree handle, queries `FileIdInfo` (VolumeSerialNumber + 128-bit FileId), and records immutable `attempt_workspace_bindings`.
     - Hexadecimal normalization: `volume_serial_hex` (16 lowercase hex characters), `file_id_hex` (32 lowercase hex characters).
  2. **Schema v6 DDL**:
     - `attempt_workspace_bindings`: Primary key `attempt_id`, foreign keys to `tasks` and `task_contracts`, `session_id` and `terminal_generation` strings without foreign key coupling to mutable `worker_sessions(session_id)`. Lineage triggers ensure consistency with `task_attempts` and `dispatch_operations`. Immutable update/delete triggers.
     - `worker_claims`: Canonical 1-to-1 cardinality with `UNIQUE(attempt_id)`. Persists `reported_head_sha` (40 hex), `claimed_files_json`, `claimed_tests_json`, `claims_payload_json`, and `reported_at`. Lineage triggers guard against cross-pairing; immutable triggers prevent modification.
  3. **Transaction Boundary A (Report Intake)**:
     - Invoked upon worker report submission.
     - Reads and verifies that `attempt_workspace_bindings` exists and matches physical handle identity.
     - Persists structured claims into `worker_claims`.
     - CAS state transition: `tasks.state`: `RUNNING -> REPORT_READY`.
  4. **OS Handle Identity Verification**:
     - Opens root directory handle with sharing flags `FILE_SHARE_READ | FILE_SHARE_WRITE` (strictly omitting `FILE_SHARE_DELETE`).
     - Revalidates identity immediately prior to CAS state transitions.
- **Traceability**: FR-008, SEC-001, SEC-005, NFR-007.

### 3.2. Subtask P04B: Hardened Git Evidence Collector (Pure In-Memory)
- **Target Task Contract**: `TASK_CONTRACT_P04_001B`
- **Ownership**: Pure in-memory Git data collection (`GitEvidenceResult`). Zero runtime SQLite writes.
- **Deliverables**:
  1. **Host-Pinned Binary Resolution**: Git binary resolved from host configuration (`SUPERVISOR_GIT_BIN`), version and SHA-256 hash verified against pinned values per NFR-007. Arbitrary `PATH` resolution prohibited.
  2. **Windows Configuration Isolation**: Dedicated trusted empty directory `<SUPERVISOR_STATE_ROOT>/trusted_empty_git/` with empty config and empty hooks directory.
  3. **Mandatory CLI Flags & Invocations**:
     - Every Git invocation passes `-c core.hooksPath=<trusted_empty_hooks>` and `-c core.fsmonitor=false`.
     - `--no-pager` strictly passed as CLI argument, not an environment variable (`GIT_PAGER=cat`).
  4. **Environment Sanitization**:
     - Unset: `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, object dirs.
     - Sanitize: `GIT_CONFIG_COUNT`, `GIT_CONFIG_KEY_*`, `GIT_CONFIG_VALUE_*`, `GIT_EXTERNAL_DIFF`, `GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_SSH`, `GIT_SSH_COMMAND`, `GIT_PROTOCOL_FROM_USER`, `GIT_CEILING_DIRECTORIES`.
     - Set: `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_SYSTEM` pointing to empty config, `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`.
  5. **Git Command Allowlist**:
     - `git rev-parse --verify <ref>`
     - `git rev-parse --git-path index`
     - `git rev-parse --absolute-git-dir`
     - `git merge-base <base> <head>`
     - `git log --no-ext-diff --no-textconv --format=...`
     - `git diff --no-ext-diff --no-textconv --raw / --patch`
     - `git ls-tree -rz --full-tree <actual_head_sha>`
     - `git cat-file --batch`
  6. **Linked Worktree Index Resolution**: For Git linked worktrees, resolves index path via `git rev-parse --git-path index`, canonicalizes path, and verifies handle ownership before computing the authoritative index SHA-256.
  7. **Integrity Controls**: 10s execution timeout; 10 MB streaming byte cap terminating entire process tree immediately upon exceedance.
- **Traceability**: FR-008, FR-009, SEC-005, NFR-007.

### 3.3. Subtask P04C: Verification Runner & Security Boundary (Pure In-Memory)
- **Target Task Contract**: `TASK_CONTRACT_P04_001C`
- **Ownership**: Pure in-memory test execution (`TestEvidenceResult`). Zero runtime SQLite writes.
- **Deliverables**:
  1. **Windows AppContainer Security Boundary**: Exclusively Windows AppContainer for v1. Non-AppContainer token alternatives are eliminated.
  2. **Process I/O & Handle Inheritance**:
     - Standard process I/O configured with `STARTF_USESTDHANDLES`.
     - Stdout and stderr redirected to anonymous pipe write handles; stdin closed or directed to nul.
     - `bInheritHandles = TRUE` in `CreateProcessW`.
     - `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` passed to `UpdateProcThreadAttribute` containing strictly the designated stdout and stderr write handles.
     - Parent read pipe ends and all other supervisor handles created without `HANDLE_FLAG_INHERIT`.
     - Supervisor streaming reader applies 10 MB byte cap and execution timeout; closes parent write ends immediately after process creation; closes pipe ends in proper order so EOF is signaled cleanly without hangs.
  3. **Atomic Job Object Association**:
     - Sandboxed process is assigned to dedicated Job Object atomically at creation on Windows 10+ using `PROC_THREAD_ATTRIBUTE_JOB_LIST`.
     - Breakaway process creation flags are eliminated.
     - Job Object limits: `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, breakaway disabled, 2 GB memory cap.
  4. **Attribute List Specification**: Contains at minimum `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES`, `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`, and `PROC_THREAD_ATTRIBUTE_JOB_LIST`.
  5. **Sandbox Root & DACLs**:
     - Sandbox root: `<SUPERVISOR_STATE_ROOT>/sandboxes/<attempt_id>/<fencing_token>/`.
     - DACL granted to AppContainer SID: minimal necessary rights (`FILE_GENERIC_READ | FILE_GENERIC_WRITE | FILE_GENERIC_EXECUTE | DELETE`) strictly within the attempt sandbox directory.
     - Snapshot root: granted read/execute DACL.
  6. **AppContainer Profile Moniker Lifecycle**:
     - Valid Win32 characters only, length <= 64 characters, moniker format `appcontainer-attempt-<first_16_hex_of_sha256>`, full lifecycle (create/derive, DACL cleanup, deletion, startup orphan cleanup).
  7. **Pinned Immutable Snapshot Protocol**:
     - Timing: Created strictly AFTER Transaction A report intake completes (Task in `REPORT_READY`, binding revalidated).
     - Sourced strictly from actual independently verified HEAD commit (`actual_head_sha`), never worker claims.
     - Streamed via Git `ls-tree -rz --full-tree <actual_head_sha>` and `git cat-file --batch` into `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/`.
     - Validation caps: max 10,000 files, 10 MB per-blob cap, 100 MB total snapshot cap, streaming extraction, disk quota checking, and fail-closed cleanup of partial snapshots on error.
     - Path validation: directory traversal (`..`), junctions, reparse points, symlinks, and gitlinks strictly rejected as a v1 fail-closed stop condition.
     - Immutable manifest binds `actual_head_sha`, relative path, Git mode, byte length, and blob object ID; verified before test invocation.
     - Root worktree handle opened without `FILE_SHARE_DELETE` and held across verification. Tests never touch the active worktree, and changes are never copied back.
- **Traceability**: SEC-001, SEC-003, OPS-003, NFR-005.

### 3.4. Subtask P04D: Pipeline Orchestrator, Artifact Store & ReviewBundle
- **Target Task Contract**: `TASK_CONTRACT_P04_001D`
- **Ownership**: Schema v9 Migration (`task_verification_leases`, `evidence_sets`, `review_artifacts`, `review_bundles`), Transaction Boundaries B & C, Content-Addressed Artifact Store, ReviewBundle Synthesis.
- **Deliverables**:
  1. **Schema v9 DDL**:
     - `task_verification_leases`: Monotonic fencing token (`> 0`), contract lineage FK, anti-cross-pairing triggers, durable epoch timestamp CHECK constraints. Atomic `BEGIN IMMEDIATE` admission query and CAS reclaim.
     - `evidence_sets`: Primary key `evidence_set_id`, unique `attempt_id`, full JSON payloads for Git evidence, test evidence, policy findings, and unverified claims, and `evidence_committed_at_epoch_ms`. Lineage guard trigger.
     - `review_artifacts`: Foreign key referencing `evidence_sets(evidence_set_id)`. Exact canonical path format `artifacts/<first-two-hex>/<captured_sha256>` without extensions, anti-traversal CHECK (`GLOB '*..*'` and `GLOB '*//*'` rejected, colons and backslashes rejected), full byte tracking, immutable triggers.
     - `review_bundles`: Foreign key referencing `evidence_sets(evidence_set_id)`, `UNIQUE(attempt_id)`, JCS canonical JSON payload, 64 lowercase hex check, `evidence_committed_at_epoch_ms`, `bundle_committed_at_epoch_ms`, `compilation_latency_ms` (CHECK `0 <= compilation_latency_ms <= 3000 ms` and `bundle_committed >= evidence_committed`), immutable triggers.
  2. **Sequential Verification Budgeting**:
     - Verification requests execute strictly sequentially in v1; total budget is the sum of request timeouts plus Git collector timeout (10s) and bounded overhead (15s).
  3. **Transaction Boundary B (Evidence Finalization)**:
     - Ingests in-memory outputs from P04B and P04C.
     - Staged file writes to `.staging/<attempt_id>-<fencing_token>-<uuid>.tmp` flushed via `FlushFileBuffers`.
     - Atomic rename via `MoveFileExW` without file replacement flags.
     - Destination collision: reopen handle, compare size and SHA-256 hash. Deduplicate if identical; fail closed if mismatching.
     - Persists `evidence_sets` and `review_artifacts(evidence_set_id)` with `evidence_committed_at_epoch_ms`. Verifies evidence persistence before state transition.
     - Releases verification lease (`task_verification_leases.state = 'RELEASED'`).
     - CAS transition: `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
  4. **Transaction Boundary C (ReviewBundle Compilation)**:
     - Synthesizes RFC 8785 JCS canonical `ReviewBundle` payload.
     - Validates against `docs/schemas/review-bundle.schema.json`.
     - Inserts into `review_bundles(evidence_set_id)` with `bundle_committed_at_epoch_ms` and `compilation_latency_ms`. Replay with identical hash is idempotent; differing hash fails closed.
     - Inserts proposed audit event `REVIEW_BUNDLE_GENERATED` into `audit_events`.
     - CAS transition: `tasks.state`: `EVIDENCE_READY -> REVIEWING`.
  5. **Two-Phase Failure Audit Semantics**:
     - Clock regression, latency violation (> 3000 ms), schema failure, or hash conflict causes Transaction C to roll back cleanly via SQLite WAL.
     - A separate fail-closed diagnostic transaction writes proposed audit event `REVIEW_BUNDLE_COMPILATION_REJECTED` into `audit_events` without altering TaskState (`EVIDENCE_READY` preserved).
- **Traceability**: FR-008, FR-009, FR-013, NFR-004, NFR-007, NFR-008.

---

## 4. Crash Consistency & Replay Matrix

| Crash Point | Durable State | Physical Storage State | Recovery / Replay Mechanism | Invariant Guaranteed |
| :--- | :--- | :--- | :--- | :--- |
| **Crash during Dispatch Seam** | `READY` or `DISPATCH_BOUND` | Uncommitted SQLite write; worktree exists | Dispatch transaction rolls back. Re-dispatch re-acquires physical worktree identity and commits binding. | Physical worktree identity is committed prior to `RecordSendRequested`. |
| **Crash during Transaction A** | `RUNNING` | Uncommitted SQLite write; staged report temp file | SQLite WAL rolls back. Task remains `RUNNING`. Worker re-submits report. | Zero partial SQLite records. |
| **Crash during P04B / P04C Execution** | `REPORT_READY` | In-memory buffers lost; active lease expires durably | Active lease reaches `expires_at_epoch_ms` as determined from durable row timestamps. New supervisor instance re-acquires lease with incremented `fencing_token`, re-validates binding, terminates old Job Object, and re-executes collectors. | Monotonic fencing token invalidates old worker effects; zombie processes terminated. |
| **Crash during Transaction B** | `REPORT_READY` | Staged or final physical files exist without SQLite metadata | SQLite WAL rolls back. Task remains `REPORT_READY`. Orphan physical files produce zero authoritative review rows. Next lease holder re-runs and re-commits cleanly. | Physical files without SQLite metadata are inert. |
| **Crash between B and C** | `EVIDENCE_READY` | Evidence metadata, artifacts, and lease release committed | Task is durably in `EVIDENCE_READY`. ReviewBundle compiler restarts directly at Boundary C without re-running Git or tests. Unique attempt constraint ensures single authority wins. | Evidence is durable; zero duplicate test runs. |
| **Crash during Transaction C** | `EVIDENCE_READY` | In-memory bundle lost | SQLite WAL rolls back. Task remains `EVIDENCE_READY`. ReviewBundle compiler re-attempts Transaction C. Replay is idempotent. | Idempotent deterministic compilation. |

---

## 5. Requirement Reconciliation Matrix

| Requirement | Implementation Component | Verification Method | Governance Status |
| :--- | :--- | :--- | :--- |
| **FR-008** (Audit-Grade Evidence Collection) | Subtasks P04A, P04B, P04D (`attempt_workspace_bindings`, Git collector, `evidence_sets`, `review_artifacts`) | Automated unit/integration tests with Git diff assertions and handle verification | Approved at Architecture Level |
| **FR-009** (Scope Comparison & Path Validation) | Subtasks P04B, P04D (Changed-file extraction, canonical path checking, anti-traversal guards) | Contract scope comparison probes, path normalization tests | Approved at Architecture Level |
| **FR-013** / **NFR-004** (Append-Only Audit Trail) | Subtasks P04A, P04D (`audit_events` integration, immutable triggers on all tables) | Tamper resistance tests, hash chain verification | Approved at Architecture Level |
| **SEC-001** / **SEC-003** / **OPS-003** (Process Containment & Execution Isolation) | Subtask P04C (Windows AppContainer, explicit handle inheritance, dedicated Job Object) | Windows sandbox isolation probes, token inspection | Approved at Architecture Level |
| **SEC-005** (Workspace Boundary Enforcement) | Subtasks P04A, P04B, P04C, P04D (`FILE_ID_128` handle validation, snapshot isolation) | Worktree handle revalidation probes, anti-traversal validation | Approved at Architecture Level |
| **NFR-005** (Upstream Adapter Separation) | Subtasks P04A, P04B, P04C (Strict separation of AO session logic and independent verification) | Pure in-memory execution, mock AO harnesses | Approved at Architecture Level |
| **NFR-007** (Pinned Upstream Tool Dependencies) | Subtasks P04B, P04C (`SUPERVISOR_GIT_BIN`, toolchain pinning, catalog profiles) | SHA-256 and version pinning checks | Approved at Architecture Level |
| **NFR-008** (ReviewBundle Latency Budget) | Subtask P04D (Interval 1 sequential request sum, Interval 2 `0 <= compilation_latency_ms <= 3000 ms`) | Latency instrumentation probes, durable timestamp validation | PENDING_EXTERNAL_REVIEW (`PROPOSAL-P04-002`) |
