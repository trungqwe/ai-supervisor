# DRAFT ADR-018: Evidence Collection Authority, Untrusted Verification Execution Isolation, and Review Synthesis Boundary

- **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`
- **Date:** 2026-09-26
- **Authority:** Architecture Decision Record
- **Deciders:** External Supervisor, Engineering Team
- **Tracks:** SEC-003, FR-007, FR-008, FR-009, FR-010, NFR-004, NFR-008
- **Precedence:** Level 2 (Architecture Decision Record)
- **Related Proposals:** [`PROPOSAL-P04-001`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md)
- **Supersedes:** Legacy single-layer verification models; unisolated subprocess runners

---

## 1. Context and Problem Statement

Phase P03 established the upstream Agent Orchestrator integration, proving decoupled session provisioning, turn dispatch, observation reconciliation, and raw workspace file retrieval via `AOAdapter.GetWorkspaceFile`.

However, the transition from worker turn completion to supervisor review faces three acute architectural challenges:
1. **Worktree Disambiguation**: The Supervisor requires objective Git ground truth from the exact local directory where the worker executed. In current schemas, `TaskAttempt` lacks a direct worktree identity column, and AO does not expose private worktree paths via its public session API.
2. **Untrusted Code Execution Threat in Verification**: [`ADR-013`](ADR-013-trusted-verification-command-spec.md) formalized Layer A (Command Authority Containment) using discrete verification profiles, but explicitly documented that Windows Job Objects do not provide filesystem sandboxing, network isolation, or credential stripping. Verification profiles (e.g. `go test`, `pytest`) execute arbitrary repository code written by an untrusted AI worker:
   ```
   SAFE_COMMAND_SPEC != TRUSTED_CODE_EXECUTION
   ```
3. **Drift in Review Synthesis Specifications**: Canonical specifications (`docs/10_REVIEW_BUNDLE.md`), JSON schemas (`review-bundle.schema.json`), and Go domain entities (`internal/domain/entities.go`) have diverged across field names, typing, and evidence representations.

This ADR establishes the formal architectural decisions governing worktree binding, read-only Git evidence extraction, two-layer execution isolation, scope policy evaluation, and atomic review bundle synthesis.

---

## 2. Decision

### Decision 1: Immutable Worktree Binding (`AttemptWorkspaceBinding`)
1. **Host-Owned Pre-Dispatch Binding**: Prior to issuing `POST /api/v1/sessions/{id}/send`, the Supervisor Core must authoritatively resolve and validate the physical local Git worktree path assigned to the attempt.
2. **Canonical Path Resolution**: The path must be canonicalized via `filepath.EvalSymlinks` and Windows file handle identity checks (`GetFinalPathNameByHandleW`). Path aliases, junction points, and relative escapes are forbidden.
3. **Persistence**: The binding is persisted in SQLite table `attempt_workspace_bindings` with foreign key to `task_attempts(attempt_id)`.
4. **Fail-Closed Invariant**: If the worktree cannot be physically verified as a clean Git worktree matching `projects.root_path` before dispatch, dispatch is aborted (`FAIL_CLOSED`). During evidence collection, the collector verifies the worktree identity against this binding; any mismatch halts collection.

---

### Decision 2: Read-Only Git Evidence Collection Authority & Strict Ancestry Verification
1. **Trusted Absolute Executable**: All Git operations execute via a trusted absolute path to `git.exe` configured at Supervisor startup. Resolving `git` via `PATH` or worker-controlled `cwd` is strictly prohibited.
2. **Read-Only / Non-Mutating Command Set**: The evidence collector is restricted to read-only queries: `rev-parse`, `cat-file`, `status --porcelain=v1 -z`, `diff --binary --no-color`. All mutating commands (`checkout`, `reset`, `clean`, `stash`, `commit`, `merge`, `rebase`) are forbidden.
3. **Sanitized Child Environment**: Environment variables `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `HOME`, and user profile directories are stripped. Flags `-c core.hooksPath=/dev/null -c diff.external= -c core.fsmonitor=false -c diff.textconv=false` are injected on every invocation.
4. **Strict Ancestry Verification (`merge-base --is-ancestor`)**:
   - Before extracting diffs, the collector executes `git merge-base --is-ancestor <base_sha> <actual_head_sha>`.
   - If true, `base_sha` is an ancestor of `actual_head_sha`; two-dot (`base..head`) and three-dot (`base...head`) notations yield identical diff trees.
   - If false, the worker branch has diverged, rebased onto another branch, or swapped branches. The collector flags `ANCESTRY_DIVERGENCE_ERROR` and halts fail-closed; it does NOT guess diff notation.
5. **Dirty Worktree Containment**: `git status --porcelain=v1 -z -uall` detects uncommitted modifications. Any tracked/staged/untracked files outside `.supervisor/` are recorded as dirty evidence and evaluated against project cleanliness policy.
6. **Binary Diff Hashing**: The canonical diff hash `git_diff_hash` is computed as the SHA-256 digest of the raw binary diff output.

---

### Decision 3: Two-Layer Verification Isolation Architecture
Verification command execution is divided into two distinct security layers:

```
+-------------------------------------------------------------------------+
| Layer A: Command Authority Containment (Host Policy & Profile Registry) |
| - Strict VerificationProfilePolicy (no shell, no raw args, no PATH)    |
| - Parameter schema validation & trusted absolute binary lookup          |
+-------------------------------------------------------------------------+
                                    |
                                    v
+-------------------------------------------------------------------------+
| Layer B: Untrusted Code Execution Isolation (Sandbox Execution Boundary)|
| - Windows AppContainer or Restricted Token (Low Integrity / Sandboxed)  |
| - Windows Job Object (Process tree kill-on-close, memory/time limits)   |
| - Zero-Secret Environment (No OpenAI/GitHub/AO credentials)             |
| - Default Deny Network Access (Inbound/Outbound blocked)                |
| - Isolated Attempt-Local TEMP/TMP Directories                           |
+-------------------------------------------------------------------------+
```

1. **Rejection of Job Object as Sole Sandbox**: A Windows Job Object provides process lifecycle and resource limit containment; it does NOT provide filesystem sandboxing, network isolation, or secret stripping. Claiming Job Object alone as Layer B isolation is formally rejected.
2. **Layer B Sandbox Implementation on Windows**:
   - Primary Candidate: **Windows AppContainer Isolation** paired with **Windows Job Object**. AppContainer natively enforces capability-based filesystem boundaries (restricting access strictly to the assigned worktree) and blocks network access by default.
   - Secondary Candidate: **Restricted Token with Low Integrity Level**, firewall rules, and explicit directory ACLs.
3. **Zero Secret Exposure**: Under no circumstances may Supervisor daemon secrets, LLM API keys, AO tokens, or tunnel credentials be inherited by verification child processes.
4. **Process Tree Lifecycle**: On execution completion, timeout, or cancellation, the Windows Job Object terminates the entire child process tree (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`), preventing orphaned background processes.
5. **Fail-Closed on Isolation Capability Absence**: If the host environment lacks the configured Layer B isolation capability, verification requests fail-closed (`ISOLATION_CAPABILITY_UNAVAILABLE`).
6. **Operational Limits**: Operational ceilings (timeouts, output caps) remain `VALUE = UNSET` or host-injected until formal approval.

---

### Decision 4: Deterministic PolicyEngine Semantics
The `PolicyEngine` evaluates TaskContract scope compliance against actual changed files:
1. **Precedence Invariant**:
   ```
   forbidden_scope PRECEDENCE > allowed_scope
   ```
   If a path matches any pattern in `forbidden_scope`, it is strictly rejected, regardless of whether it matches `allowed_scope`.
2. **Canonical Slash & Case Normalization**: All paths and glob patterns use `/` separators. On Windows, paths are evaluated case-insensitively to prevent casing bypasses.
3. **Special File Entries**:
   - Renamed/copied files: Both source and destination paths must satisfy `allowed_scope` and not match `forbidden_scope`.
   - Submodules: Submodule paths are treated as file entries and must fall within `allowed_scope`.
   - Symlinks: Both the link entry and its resolved target must be within `allowed_scope`.
   - Control plane path `.supervisor/**`: Excluded from worker application diffs; any application commit modifying `.supervisor/` triggers fatal policy rejection.
4. **Fail-Closed**: Empty `allowed_scope` permits zero file modifications. Invalid glob syntax fails evaluation immediately.

---

### Decision 5: Two-Phase Evidence State Progression & Persistence Transactions
State transitions between worker turn completion and supervisor review are strictly divided into atomic, idempotent stages:

```
RUNNING ---> [Atomic Ingestion Tx] ---> REPORT_READY
                                              |
                                              v
REPORT_READY ---> [Atomic Evidence Tx] ---> EVIDENCE_READY
                                                  |
                                                  v
EVIDENCE_READY ---> [Atomic Bundle Tx] ---> REVIEWING
```

1. **Phase 1 (`RUNNING -> REPORT_READY`)**:
   - Invoked after AO session activity is observed as `idle`.
   - Ingests `worker-report.json` via `GetWorkspaceFile`.
   - Validates 100% against `worker-report.schema.json`.
   - Verifies `task_id` and `attempt_id`.
   - Atomically updates `task_attempts.ended_at`, stores raw report, inserts `worker_claims`, transitions task state to `REPORT_READY`, and logs audit event.
   - If report is missing or malformed: Applies `REC-007` / `REC-008`, transitions task state to `FAILED`, and generates zero normal ReviewBundle.
2. **Phase 2 (`REPORT_READY -> EVIDENCE_READY`)**:
   - Executes Git evidence collection, PolicyEngine scope check, and all mandatory `verification_requests` via isolated `VerificationRunner`.
   - Atomically inserts `evidence`, `actual_test_results`, `policy_findings`, transitions task state to `EVIDENCE_READY`, and logs audit event.
   - Partial evidence commits are prohibited.
3. **Phase 3 (`EVIDENCE_READY -> REVIEWING`)**:
   - `ReviewBundleBuilder` compiles synthesized packet.
   - Validates bundle against `review-bundle.schema.json`.
   - Atomically inserts `review_bundles`, transitions task state to `REVIEWING`, and logs audit event.

---

### Decision 6: Canonical ReviewBundle Schema Standard & Drift Resolution
The ReviewBundle schema and domain models are reconciled under a single authoritative standard:
1. **Diff Metrics**: Adopt `diff_stat: string` for human-readable insertion/deletion summary, and `git_diff_hash: string` for cryptographic proof. Remove obsolete `full_diff_url`.
2. **Structured Test Evidence**: `actual_test_evidence` contains `all_passed: boolean`, `executed_commands: string[]`, and structured `test_results: []ActualTestResult`.
3. **Structured Worker Claims**: Replace unstructured `worker_claims: object` with an explicit schema matching `domain.WorkerClaim`.
4. **Nested Schema Closure**: Apply `additionalProperties: false` across all nested objects in `review-bundle.schema.json`.
5. **Domain Entity Alignment**: Create `ReviewBundle` struct in `internal/domain/entities.go` matching the reconciled schema.

---

## 3. Consequences

### Positive Consequences
- **Zero-Trust Integrity**: Worker claims can never spoof verified test outcomes or Git diffs.
- **Robust Execution Safety**: Host secrets cannot leak to worker-generated test scripts, and test processes cannot escape worktree boundaries or hang the host.
- **Deterministic State Lineage**: Evidence, claims, and review bundles are permanently bound to exact immutable `TaskAttempt` and `TaskContract` records.
- **Fail-Closed Operations**: Missing worktrees, diverged Git branches, isolation failures, or schema mismatches halt cleanly without corrupting the repository.

### Negative / Trade-Off Consequences
- **Platform-Specific Isolation Complexity**: Implementing Windows AppContainer / Restricted Token isolation requires dedicated Windows API interop.
- **SLA Clarification Required**: Running real verification suites (e.g. `go test -race ./...`) exceeds the literal 3-second SLA of NFR-008, requiring formal requirement clarification.

### Invariants Preserved
- `AUTOMATIC_RESTORE = DISABLED`
- `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`
- `LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`
- `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`
