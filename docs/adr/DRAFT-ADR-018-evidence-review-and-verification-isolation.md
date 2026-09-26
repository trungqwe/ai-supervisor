# DRAFT ADR-018: Evidence Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

> **Status**: `DRAFT_PENDING_EXTERNAL_APPROVAL`
> **Revision**: 7
> **Date**: 2026-09-26
> **Audited Baseline**: `742e2e296db99c19bc7a645ca1842520b2ab23e5`
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_6`
> **Decision Owners**: AI Engineering Supervisor Team
> **Supersedes**: `DRAFT-ADR-018` Revision 6
> **External Audit Tracking**: Remediates Findings `P04-ARCH-R6-001` through `P04-ARCH-R6-006` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_005.md`).
> **Requirement & Governance Note**: Cannot be accepted or marked ready until `PROPOSAL-P04-002-review-bundle-latency-semantics.md` is approved by External Supervisor. `docs/02_REQUIREMENTS.md` remains unmodified.

---

## 1. Context and Problem Statement

Phase P04 implements the **Evidence & Review Engine** to provide independent, tamper-proof verification of AI worker outputs. Under canonical architecture (`docs/04_ARCHITECTURE.md` Section 7) and requirements (`docs/02_REQUIREMENTS.md`), the Supervisor must:
1. Verify physical worktree authority without relying on unverified worker claims;
2. Collect tamper-proof Git diffs without risk of hook execution, config poisoning, or directory mutation;
3. Execute independent test suites and linters in isolated Windows execution environments with strict resource, filesystem, and network isolation;
4. Manage content-addressed review artifacts with rock-solid durability and crash consistency;
5. Compile an attempt-scoped `ReviewBundle` adhering to `docs/10_REVIEW_BUNDLE.md` within governed latency budgets.

External Re-Audit 005 required remediation of six critical findings (`P04-ARCH-R6-001` through `P04-ARCH-R6-006`). This ADR establishes the architectural decisions resolving these findings.

---

## 2. Decision Drivers

- **Security & Integrity**: Complete isolation against malicious or buggy worker processes, hooks, external diff tools, and network exfiltration.
- **Upstream Alignment**: Respect pinned Agent Orchestrator (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) authority over worker sessions and physical worktrees.
- **Fail-Closed Verification**: Zero reliance on worker self-reporting; missing or mismatched physical bindings, handles, or hashes abort verification immediately.
- **State Machine Atomicity**: Clear transactional boundaries with monotonic lease fencing and deterministic crash/replay recovery.
- **Requirement Governance**: Strict adherence to the 9-level change governance hierarchy (`docs/24_CHANGE_GOVERNANCE.md`); zero silent modification of canonical requirements.

---

## 3. Considered Options

1. **Option 1: In-Process Ephemeral Verification (Rejected)**: Running Git and test commands directly under the Supervisor daemon process without OS-level isolation. Rejected due to severe security and tampering vulnerabilities (SEC-002, SEC-003).
2. **Option 2: Disposable AO Session Proof Gate (Rejected)**: Gating Phase P04 contract release on running disposable worker sessions via upstream AO. Rejected (`P04-ARCH-R5-001`, `P04-ARCH-R6-002`) because AO contains no inert harness in `AllHarnesses`, and running live agent CLIs creates uncontrollable safety and token hazards.
3. **Option 3: Hardened In-Memory Engine with AppContainer Security Boundary & Three Transaction Boundaries (Accepted)**: Independent physical OS handle validation, hardened in-memory Git collection, Windows AppContainer / restricted token security boundaries with DACL-protected immutable source snapshots, append-only content-addressed artifact storage, and three distinct durable SQLite transaction boundaries.

---

## 4. Decision Outcome

### Decision 1: Worktree Authority, Durable Attempt Binding & OS Handle Lifecycle (P04-ARCH-R6-001, R6-002)
1. **Schema v6: `attempt_workspace_bindings`**:
   - `terminal_generation TEXT NOT NULL CHECK (LENGTH(terminal_generation) > 0)` (opaque string matching P03 domain models and migrations).
   - Foreign keys: `contract_id REFERENCES task_contracts(contract_id) ON DELETE RESTRICT`, `session_id REFERENCES worker_sessions(session_id) ON DELETE RESTRICT`.
   - Lineage protection: `FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT` and `trg_attempt_workspace_bindings_lineage_guard` trigger aborting any cross-pairing across task, attempt, and contract.
   - CHECK constraints enforcing non-empty paths, volume serial, file_id, and exact 40 lowercase hex for `pinned_ao_commit`:
     `LENGTH(pinned_ao_commit) = 40 AND NOT (pinned_ao_commit GLOB '*[^0-9a-f]*')`.
   - Eradicate every single occurrence of `nonexistent generation on worker_sessions`; use `worker_sessions.terminal_generation`.
2. **OS Handle Lifecycle**:
   - **Phase 1 (Materialization)**: Open directory handle with `FILE_FLAG_BACKUP_SEMANTICS` and query `FileIdInfo` (VolumeSerialNumber + 128-bit FileId).
   - **Phase 2 (Immutable Registration)**: Atomically insert `attempt_workspace_bindings` prior to issuing `SEND_REQUESTED`.
   - **Phase 3 (Pre-Lease Re-Opening)**: Reopen handle and compare physical identity before granting verification lease. Mismatch fails closed.
   - **Phase 4 (CAS Revalidation)**: Revalidate identity immediately before committing durable evidence CAS.
3. **Terminology & Prerequisite Decoupling**:
   - Worktree binding validation is named `WORKTREE_BINDING_RUNTIME_VALIDATION`. The term "Stage B" is strictly reserved for `TaskContractValidator` semantic validation and `STAGE_B_RUNTIME_CATALOG`.
   - The unexecutable proof contract prerequisite edge is removed from the execution graph. Subtask P04A is the first releaseable subtask once architecture is approved.

### Decision 2: Hardened In-Memory Git Collector (P04-ARCH-R6-003)
1. **Host-Pinned Binary Resolution**: Git executable resolved strictly from host configuration (`SUPERVISOR_GIT_BIN`), version and SHA-256 pinned per NFR-007. Arbitrary `PATH` lookups are prohibited.
2. **Windows Configuration Isolation**: Windows cannot use Unix null device syntax for directories. Supervisor creates `<SUPERVISOR_STATE_ROOT>/trusted_empty_git/` with an empty config file and empty hooks directory.
3. **Environment Sanitization**: Unset `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, object dirs. Set `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_SYSTEM`, `GIT_CONFIG_GLOBAL` to empty config, `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`, `--no-pager`, `GIT_PAGER=cat`.
4. **Command Allowlist & Flags**: Only allowlisted non-mutating commands (`rev-parse`, `merge-base`, `log`, `diff`). Always pass `--no-ext-diff` and `--no-textconv`. Use literal pathspecs for file inputs.
5. **Execution Controls**: 10s timeout, 10 MB byte cap, process-tree termination, pre/post HEAD/index/worktree identity verification.
6. **Traceability**: Traced to **FR-008** (Git-Verified Diffs).

### Decision 3: Windows Verification Isolation Security Boundary (P04-ARCH-R6-004)
1. **Security Boundary Implementation**:
   - Subprocesses execute inside a Windows AppContainer created via `CreateAppContainerProfile` with **empty `SECURITY_CAPABILITIES`** (zero network capabilities) or restricted primary token (`CreateRestrictedToken`) with network denial.
   - Verification executes against a DACL-protected immutable source snapshot (`<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/`) with read-only permissions granted to the sandbox SID. The active worktree is never exposed, and changes are never copied from snapshot back to worktree.
   - Ephemeral writes (`TEMP`, `TMP`, `GOCACHE`, `GOPATH`), build outputs, and process outputs are confined to `<SUPERVISOR_STATE_ROOT>/sandboxes/<attempt_id>/`.
2. **Job Object Containment**: Dedicated Job Object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, breakaway disabled, memory limit (2 GB), zero inherited handles (`bInheritHandles = FALSE`).
3. **Supplementary UI Isolation**: Dedicated non-interactive Window Station and Desktop (`CreateWindowStationW` + `CreateDesktopW`) isolate UI messages and clipboard.
4. **Fail-Closed Mandate**: If AppContainer, DACLs, or Job Object cannot be established, execution fails closed immediately. Falling back to daemon token execution is strictly prohibited.

### Decision 4: Verification Authority & Latency Semantics via PROPOSAL-P04-002 (P04-ARCH-R6-006)
1. Verification commands execute strictly via approved profiles in `VerificationPolicyCatalog` with host-governed absolute paths.
2. NFR-008 performance SLA reconciliation is governed by `PROPOSAL-P04-002-review-bundle-latency-semantics.md` (`PENDING_EXTERNAL_REVIEW`), establishing a two-interval model:
   - Interval 1 (Evidence Acquisition Window): Governed by attempt verification budgets (`max_verification_budget_ms`).
   - Interval 2 (Bundle Compilation Window): $T_1 - T_0 \le 3.0	ext{ seconds}$ from terminal evidence inputs to persisted ReviewBundle.
3. Canonical `docs/02_REQUIREMENTS.md` remains unmodified until formal approval.

### Decision 5: Three Durable Pipeline Transactions & Lease Fencing (P04-ARCH-R6-006)
1. **Three Durable Transaction Boundaries**:
   - **Transaction A (Report Intake)**: Validate report, persist claims, insert `attempt_workspace_bindings` (Schema v6), transition `tasks.state`: `RUNNING -> REPORT_READY`.
   - **Transaction B (Evidence Finalization)**: Ingest in-memory collector outputs, flush/rename artifacts, release lease, transition `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
   - **Transaction C (ReviewBundle Compilation)**: Synthesize RFC 8785 JCS ReviewBundle, insert `review_bundles`, insert audit event, transition `tasks.state`: `EVIDENCE_READY -> REVIEWING`.
2. **Schema v9: `task_verification_leases`**:
   - `fencing_token INTEGER NOT NULL CHECK (fencing_token > 0)`
   - Lineage foreign keys: `task_id REFERENCES tasks`, `attempt_id REFERENCES task_attempts`, `contract_id REFERENCES task_contracts`.
   - Anti-cross-pairing triggers for insert and update.
   - CHECK: `expires_at_epoch_ms > acquired_at_epoch_ms`, `ACTIVE` requires `released_at_epoch_ms IS NULL`, `RELEASED` requires `released_at_epoch_ms IS NOT NULL AND released_at_epoch_ms >= acquired_at_epoch_ms`.
3. **Atomicity Guarantee**: Atomicity is guaranteed within each transaction via SQLite WAL. Pipeline recovery follows a deterministic crash/replay matrix.
4. **Authority Separation**: Subtasks P04B and P04C are pure in-memory (zero runtime SQLite writes). Write authority belongs exclusively to Subtasks P04A (Transaction A) and P04D (Transactions B & C).

### Decision 6: Durable Content-Addressed Artifact Store & Review Schema (P04-ARCH-R6-005)
1. **Append-Only Scope Reduction**: P04 initial delivery is strictly append-only with **zero physical garbage collection**. Physical GC is DEFERRED.
2. **Schema v9: `review_bundles` & `review_artifacts`**:
   - `review_bundles`: JCS canonical JSON payload, SHA-256 hash check (`LENGTH = 64 AND NOT (bundle_hash GLOB '*[^0-9a-f]*')`), immutable triggers.
   - `review_artifacts`: Full retrieval metadata (`artifact_type`, `media_type`, `encoding`, `canonical_relative_path`, `captured_sha256`, `full_stream_sha256`, byte counts, `is_truncated`), 64 lowercase hex checks, immutable triggers.
3. **Atomic Write-Through Rename**:
   - File written to `.staging/<attempt_id>-<uuid>.tmp`, flushed via `FlushFileBuffers`, atomically renamed via `MoveFileExW(MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH)`. Staging and final roots verified on same volume.
4. **Post-Rename Verification**:
   - Reopen final file by handle, verify volume identity, size, and SHA-256 hash before inserting SQLite metadata.
5. **Collision Handling**:
   - Existing file collision re-hashes and compares size; matching is deduplicated, mismatch fails closed and quarantines file.
6. **Schema Reconciliation Matrix**: Full alignment documented between `docs/10_REVIEW_BUNDLE.md`, `review-bundle.schema.json`, domain models, and proposed DDL.

---

## 5. Consequences

### Positive
- Strict, kernel-enforced process and network security boundary for all verification commands.
- Zero risk of TOCTOU directory swaps, git hook injection, or unauthorized external tool invocation.
- Deterministic crash recovery across three well-defined transaction boundaries.
- Complete fidelity to upstream AO authority without unexecutable proof deadlocks.
- Clear governance path for NFR-008 performance SLA reconciliation.

### Negative / Trade-offs
- Verification subprocesses execute against an immutable snapshot, requiring snapshot disk staging and cleanup in the sandbox directory.
- AppContainer configuration on Windows requires specialized Win32 API invocations.
- Deferral of physical GC means orphan files from uncommitted crashes must be pruned by future offline maintenance tools.

---

## 6. Migration and Schema Ownership

- **Schema v6 (Owned by Subtask P04A)**:
  - `attempt_workspace_bindings` table, indexes, and triggers.
- **Schema v9 (Owned by Subtask P04D)**:
  - `task_verification_leases` table, indexes, and triggers.
  - `review_bundles` table, indexes, and triggers.
  - `review_artifacts` table, indexes, and triggers.
- **Subtasks P04B & P04C**:
  - Pure in-memory execution; zero migration ownership, zero runtime SQLite writes.
