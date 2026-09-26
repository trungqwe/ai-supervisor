# DRAFT ADR-018: Evidence Collection Authority, Untrusted Verification Execution Isolation, and Review Synthesis Boundary

- **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`
- **Revision:** 2 (Remediation of External Audit 001)
- **Date:** 2026-09-26
- **Authority:** Architecture Decision Record
- **Audit Reference:** [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md)
- **Deciders:** External Supervisor, Engineering Team
- **Tracks:** SEC-003, FR-007, FR-008, FR-009, FR-010, NFR-004, NFR-008
- **Precedence:** Level 2 (Architecture Decision Record)
- **Related Proposals:** [`PROPOSAL-P04-001`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md)
- **Source Provenance:** [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md), [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md), [`docs/22_MODULE_PROVENANCE.md`](../22_MODULE_PROVENANCE.md)
- **Supersedes:** Legacy single-layer verification models; unisolated subprocess runners

---

## 1. Context and Problem Statement

Phase P03 established the upstream Agent Orchestrator integration, proving decoupled session provisioning, turn dispatch, observation reconciliation, and raw workspace file retrieval via `AOAdapter.GetWorkspaceFile` (`internal/ao/client.go`).

However, transitioning from worker turn completion to supervisor review requires resolving five architectural seams:
1. **Worktree Disambiguation**: The Supervisor requires objective Git ground truth from the exact local directory where the worker executed. In current schemas, `TaskAttempt` lacks a direct worktree identity column, and pinned AO (`v0.13.0`) does not expose private worktree paths via its public session API.
2. **Untrusted Code Execution Threat in Verification**: [`ADR-013`](ADR-013-trusted-verification-command-spec.md) formalized Layer A (Command Authority Containment) using discrete verification profiles, but explicitly documented that Windows Job Objects do not provide filesystem sandboxing, network isolation, or credential stripping. Verification profiles (e.g. `go test`, `pytest`) execute arbitrary repository code written by an untrusted AI worker:
   ```
   SAFE_COMMAND_SPEC != TRUSTED_CODE_EXECUTION
   ```
3. **Drift in Review Synthesis Specifications**: Canonical specifications (`docs/10_REVIEW_BUNDLE.md`), JSON schemas (`review-bundle.schema.json`), and Go domain entities (`internal/domain/entities.go`) have diverged across field names, typing, and evidence representations. Furthermore, ReviewBundle must provide inspectable, reviewable content for ChatGPT rather than raw hashes.
4. **Git Authority & Security**: Git commands must be strictly read-only, hardened with explicit CLI overrides, and governed by a strict descendant ancestry policy with TOCTOU protection.
5. **State Atomicity & Schema Migration Strategy**: Store operations must enforce full CAS guards across short SQLite transactions with sequential migrations (v6..v9) aligned with subtask decomposition.

This ADR establishes the formal architectural decisions governing worktree binding, read-only Git evidence extraction, two-layer execution isolation, scope policy evaluation, and atomic review bundle synthesis.

---

## 2. Decision

### Decision 1: Worktree Binding Model (`AttemptWorkspaceBinding`) — CONDITIONAL / UNRESOLVED
1. **Authority Limitation of Pinned AO**:
   - Pinned AO (`v0.13.0`) does **NOT** expose physical `worktree_path` via its public REST API (`GET /api/v1/sessions/{id}`).
   - `Project.RootPath` does not prove session worktree isolation in multi-session or branch workflows.
2. **Conditional Status**:
   - The architectural design of `AttemptWorkspaceBinding` is formally marked as **CONDITIONAL / UNRESOLVED**.
   - Prior to formal ADR-018 approval, a bounded runtime proof must establish the authoritative mechanism to map:
     ```
     (session_id, terminal_generation, attempt_id, branch) -> physical worktree
     ```
3. **Candidate Mechanism & Invariants**:
   - If querying `git worktree list --porcelain -z`: mapping must verify unique session/branch identity, handle collision detection, support crash recovery, and enforce TOCTOU checks.
   - The binding is persisted in SQLite table `attempt_workspace_bindings` (Schema v6) tied to `task_attempts(attempt_id)`.
   - Dispatch integration seam spans `internal/ao/client.go`, `internal/store/**`, and `internal/dispatch/**`.
   - **Fail-Closed Invariant**: If the physical worktree cannot be deterministically verified as a clean Git worktree matching `projects.root_path` before dispatch, dispatch is aborted (`FAIL_CLOSED`). During evidence collection, the collector verifies worktree identity against this binding; any mismatch halts collection.

---

### Decision 2: Read-Only Git Evidence Collection Authority & Strict Descendant Policy
1. **Pinned Trusted Executable**: All Git operations execute via a trusted absolute path to `git.exe` configured at Supervisor startup, verified via SHA-256 cryptographic hash and PE binary header validation. Resolving `git` via `PATH` or worker-controlled `cwd` is strictly prohibited.
2. **Explicit CLI Hardening Overrides**:
   - Every git command must execute with explicit overrides:
     `git --no-optional-locks -c core.hooksPath=/dev/null -c diff.external= -c diff.textconv=false -c core.fsmonitor=false diff --no-ext-diff --no-textconv --ignore-submodules=none`
   - All mutating commands (`checkout`, `reset`, `clean`, `stash`, `commit`, `merge`, `rebase`, `pull`, `push`, `tag`) are prohibited.
3. **Sanitized Child Environment**:
   - Strip `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, `GIT_ALTERNATE_OBJECT_DIRECTORIES`.
   - Strip `GIT_EXTERNAL_DIFF`, `GIT_DIFF_OPTS`, `GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_TERMINAL_PROMPT=0`.
   - Strip `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`.
   - Inject `GIT_CONFIG_NOSYSTEM=1` and `LC_ALL=C`.
4. **Strict Descendant Ancestry Policy (`merge-base --is-ancestor`)**:
   - The collector executes `git merge-base --is-ancestor <base_sha> <actual_head_sha>`.
   - If true, `base_sha` is an ancestor of `actual_head_sha` (Strict Descendant).
   - If project policy requires linear history, verify `git rev-list --merges <base_sha>..<actual_head_sha>` returns zero commits. If merge commits exist, they must be recorded in evidence.
   - If false, the branch has diverged, rebased, or swapped. Flag `ANCESTRY_DIVERGENCE_ERROR` and halt fail-closed; do NOT attempt to guess diff notation.
5. **TOCTOU Re-Verification**:
   - Record `pre_head = git rev-parse HEAD` and `pre_status = git status --porcelain=v1 -z -uall`.
   - Collect diffs.
   - Re-verify `post_head = git rev-parse HEAD` and `post_status = git status --porcelain=v1 -z -uall`.
   - If `pre_head != post_head` or `pre_status != post_status`, abort with `CONCURRENT_WORKTREE_MUTATION_ERROR`.
6. **Machine-Readable Manifest vs Diff Hash**:
   - Changed files parsed into a structured manifest via `git diff -z --raw --no-color`.
   - Diff byte hash `git_diff_hash` computed as SHA-256 of raw binary patch (`git diff --binary --no-ext-diff --no-textconv`).

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
| - Primary: Windows AppContainer Isolation (Low Integrity / Sandboxed)   |
| - Secondary Fallback: Restricted Token + WFP/Firewall Network Deny      |
| - Windows Job Object (Process tree kill-on-close, memory/time limits)   |
| - Worktree Mount: STRICTLY READ-ONLY (GENERIC_READ; deny write/delete)  |
| - Disposable Copy / Snapshot: Mandatory if test tool requires writing   |
| - Child Desktop & Window Station Isolation (CreateDesktopW)             |
| - Zero-Secret Environment (No OpenAI/GitHub/AO credentials)             |
| - Default Deny Network Access (Inbound/Outbound blocked)                |
| - Isolated Attempt-Local TEMP/TMP & Tool Caches                         |
+-------------------------------------------------------------------------+
```

1. **Primary Mechanism Selection**: **Windows AppContainer Isolation** paired with **Windows Job Object**.
   - AppContainer natively provides low-integrity, capability-based sandboxing, denying network by default and restricting filesystem access strictly to explicitly granted directory ACLs.
   - Secondary Fallback: Restricted Token with Low Integrity Level is permitted **only** if paired with external firewalling (WFP / Windows Firewall) to enforce network isolation, because a Restricted Token standalone **CANNOT** isolate network egress.
2. **Worktree Read-Only Mount & Disposable Copy Policy**:
   - The audited worktree is granted strictly **READ-ONLY** access (`GENERIC_READ`, deny write/delete) during verification.
   - **Prohibition on Direct Worktree Mutation**: Untrusted verification processes are strictly forbidden from modifying the audited worktree.
   - **Disposable Copy / Snapshot Policy**: If a test toolchain requires write access (e.g. building binaries, updating source files, code generation), execution MUST occur in a **temporary disposable copy / snapshot directory** with verified provenance, never mutating the audited worktree directly.
3. **Child Desktop & Window Station Isolation**:
   - Verification child processes execute on an isolated Window Station and Desktop (`CreateDesktopW`) to prevent Windows GUI message hooking, window injection, or synthetic keyboard/mouse interaction.
4. **Zero Secret Exposure**:
   - Under no circumstances may Supervisor daemon secrets, LLM API keys, AO tokens, or tunnel credentials be inherited by verification child processes.
5. **Process Tree Termination**:
   - Windows Job Object enforces `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` to terminate all background child processes upon timeout or completion.
6. **Falsification Matrix**:
   - Implementation must satisfy 5-vector falsification testing: (1) Filesystem escape attempt, (2) Network egress attempt, (3) Environment secret access attempt, (4) Detached child process escape attempt, (5) Source tree modification attempt.
7. **Operational Ceilings**:
   - Operational limits (timeouts, output caps) remain `VALUE = UNSET` or host-injected until formal approval.

---

### Decision 4: Deterministic PolicyEngine Semantics
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
RUNNING ---> [Atomic Ingestion CAS Tx] ---> REPORT_READY
                                                  |
                                                  v
REPORT_READY ---> [Atomic Evidence CAS Tx] ---> EVIDENCE_READY
                                                      |
                                                      v
EVIDENCE_READY ---> [Atomic Bundle CAS Tx] ---> REVIEWING
```

1. **Sequential Schema Migrations**:
   - Schema v6 (P04A): `worker_claims`, `attempt_workspace_bindings`
   - Schema v7 (P04B): `evidence`, `policy_findings`
   - Schema v8 (P04C): `actual_test_results`
   - Schema v9 (P04D): `review_bundles`
2. **Complete Ingestion CAS Query (`RUNNING -> REPORT_READY`)**:
   - Updates `task_attempts` (`ended_at = :ended_at`, `worker_report_raw = :raw`) WHERE `attempt_id = :attempt_id`, `ended_at IS NULL`, exact lineage (`task_id`, `contract_id`, `session_id`, `terminal_generation`).
   - Updates `tasks` (`state = 'REPORT_READY'`) WHERE `task_id = :task_id`, `state = 'RUNNING'`, `current_attempt = :attempt_number`.
   - Asserts `affected_rows == 1` across both updates; if not, rolls back immediately.
   - Inserts `worker_claims` and logs audit event in the same transaction.
   - If report is missing or malformed: Applies `REC-007` / `REC-008`, transitions task state to `FAILED`, and generates zero normal ReviewBundle.
3. **Separation of External Execution from DB Transactions**:
   - External Git diff commands and verification runners execute **OUTSIDE** database transactions.
   - Persistence takes immutable snapshots and commits via short atomic transactions with CAS guards.

---

### Decision 6: Reviewable Evidence Payload & Schema Reconciliation
The ReviewBundle schema and domain models are reconciled under a single authoritative standard providing **inspectable, reviewable evidence**:
1. **Bounded Git Patch Artifact (`bounded_git_patch`)**:
   - Inline bounded diff patch (up to bounded ceiling, e.g. 100 KB) and immutable local `artifact_id` for larger patches.
   - Metadata: `git_diff_hash` (SHA-256), `original_bytes`, `captured_bytes`, `is_truncated: boolean`, `diff_stat: string`.
   - Remove obsolete `full_diff_url`.
2. **Bounded Test Output Logs (`test_results`)**:
   - Strictly unify field name to `test_results` (eliminate historical `details` drift).
   - Capture bounded stdout and stderr text logs (or artifact ID), `exit_code`, `passed: boolean`, `stdout_hash`, `stderr_hash`, `duration_ms`, `is_truncated: boolean`.
   - Worker-reported test logs are NEVER used as verified evidence.
3. **Strictly Typed TaskContract**:
   - `task_contract` is an immutable snapshot object containing contract requirements, scopes, and criteria.
4. **Non-Circular Bundle Hashing**:
   - `bundle_hash` is computed as SHA-256 over canonical JSON serialization of ReviewBundle **excluding** the `bundle_hash` field itself.
5. **Phase P05 Bounded Artifact Retrieval**:
   - Phase P05 will access large truncated artifacts via an internal bounded artifact retrieval tool (`get_artifact(artifact_id)`) returning chunked content without granting arbitrary filesystem access to ChatGPT.
6. **Nested Schema Closure**: Apply `additionalProperties: false` recursively across all nested schema definitions.
7. **Domain Entity Alignment**: Implement `ReviewBundle` Go struct in `internal/domain/entities.go` matching reconciled schema.

---

## 3. Consequences

### Positive Consequences
- **Zero-Trust Integrity**: Worker claims can never spoof verified test outcomes or Git diffs.
- **Robust Execution Safety**: Host secrets cannot leak to worker-generated test scripts, and test processes cannot escape worktree boundaries or hang the host.
- **Audit Usability**: ReviewBundle provides inspectable, bounded diffs and logs directly to ChatGPT, preserving cognitive compression.
- **Deterministic State Lineage**: Evidence, claims, and review bundles are permanently bound to exact immutable `TaskAttempt` and `TaskContract` records with full CAS.

### Negative / Trade-Off Consequences
- **Upstream Worktree Authority Dependency**: P04 implementation cannot proceed until bounded runtime proof confirms AO session worktree resolution.
- **Platform-Specific Isolation Complexity**: Implementing Windows AppContainer isolation requires dedicated Windows API interop.
- **SLA Clarification Required**: Running real verification test suites exceeds the literal 3-second SLA of NFR-008, requiring formal two-milestone amendment.

### Invariants Preserved
- `AUTOMATIC_RESTORE = DISABLED`
- `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`
- `LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`
- `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`
