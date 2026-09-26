# DRAFT-ADR-018: Evidence Review and Verification Isolation Architecture

- **ADR ID:** `DRAFT-ADR-018`
- **Revision:** 3 (Remediation of External Re-Audit 001)
- **Status:** `DRAFT_PENDING_EXTERNAL_APPROVAL`
- **Author:** AI Engineering Supervisor Control Plane Team
- **Date:** 2026-09-26
- **Governance Authority:** [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), [`AGENTS.md`](../../AGENTS.md)
- **Related Proposal:** [`PROPOSAL-P04-001`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 3)
- **Audit References:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (`REVISION_2_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md) (`FORMALLY_RECORDED`)
- **Source Provenance:** [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md), [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md), [`docs/22_MODULE_PROVENANCE.md`](../22_MODULE_PROVENANCE.md)
- **Traceability:** FR-007, FR-008, FR-009, FR-010, SEC-003, NFR-004, NFR-008, REC-007, REC-008

---

## 1. Context and Problem Statement

Phase P04 Evidence & Review Engine must perform four functions that each require architectural clarity:
1. **WorkerReport Ingestion** (`P04A`): Parse and validate the worker-produced `worker-report.json` using `GetWorkspaceFile`, extract `WorkerClaim` entities, and persist initial evidence records.
2. **Independent Git Evidence Collection** (`P04B`): Without trusting the worker, independently collect diff patches, numstat, file list, and ancestry verification from the exact physical worktree.
3. **Isolated Verification Command Execution** (`P04C`): Execute profile-constrained verification commands against the worker's committed code under full process isolation (AppContainer) without leaking secrets or allowing network/filesystem escape.
4. **Atomic Evidence Synthesis** (`P04D`): Persist all evidence, policy findings, test results, artifact references, and the RFC 8785-hashed ReviewBundle in a single atomic CAS SQLite transaction.

Two open questions remained from External Audit 001 and were re-evaluated in Re-Audit 001:
- **Worktree Authority**: How does the Supervisor determine the exact physical filesystem path of the worker worktree without querying private AO databases or building custom worktree managers?
- **Sandbox Contradiction**: How can the audited worktree be strictly read-only while verification toolchains require writable cache directories?

---

## 2. Decision

### Decision 1: Worktree Binding Model (`AttemptWorkspaceBinding`) — CONDITIONAL / OPEN_PENDING_BOUNDED_PROOF

**Static Source Evidence (Factual, Proven):**

Inspection of pinned Agent Orchestrator (`v0.13.0`, commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) in `backend/internal/adapters/workspace/gitworktree/workspace.go` establishes as proven static facts:
- `Workspace.managedPath` (line 1887): `filepath.Join(w.managedRoot, string(cfg.ProjectID), string(cfg.SessionID))`
- `defaultSessionBranchName` (line 1918): `"ao/" + string(id)`
- `Workspace.Create` (lines 276, 287): provisions worktree via `managedPath`
- `Workspace.Restore` (lines 1142, 1150): validates worktree at `managedPath`
- `Options.ManagedRoot` (line 105): host-configured base directory

The naming formulas are:
```text
worktree_path = filepath.Join(managedRoot, projectID, sessionID)
session_branch = "ao/" + sessionID
```

**Unproven Runtime Seams (Five Mandatory Items):**
1. Host configuration source of `managedRoot` (CLI flag, environment variable, default path).
2. Supervisor authority to read that configuration value.
3. Shared physical filesystem namespace between AO and Supervisor.
4. Daemon restart/restore preservation of worktree path identity.
5. Junction, symlink, and tamper resistance.

**Governed Resolution Plan:**
- Dedicated bounded empirical proof plan: [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md)
- Dedicated draft TaskContract: [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`)
- Three-track execution: static source proof, isolated disposable runtime proof, authority conclusion.

**Prohibitions:**
- Supervisor MUST NOT query private AO SQLite databases directly.
- Supervisor MUST NOT build a custom, unapproved worktree manager.
- `git worktree list --porcelain -z` MUST NOT be treated as sole binding authority; it is cross-check only.

**Status:** `DECISION_1 = CONDITIONAL / OPEN_PENDING_BOUNDED_PROOF`

---

### Decision 2: Read-Only Git Evidence Collection Authority & Strict Descendant Policy

**Primary mechanism: Hardened, sanitized Git CLI invocations only.**

**Injected Environment Variables:**
- `GIT_TERMINAL_PROMPT=0` (MUST be injected, not stripped — prevents interactive prompts)
- `LC_ALL=C`

**Stripped Environment Variables:**
- `GIT_ASKPASS`
- `SSH_ASKPASS`
- `GIT_EXTERNAL_DIFF`
- `GIT_DIFF_OPTS`

**Fixed CLI Subcommand Argument Vectors:**

Each subcommand has exactly one fixed argument vector; no generic string substitution shared across subcommands:
1. Ancestry check:
   ```
   git --no-optional-locks merge-base --is-ancestor <base_sha> <head_sha>
   ```
2. Merge commit check:
   ```
   git --no-optional-locks rev-list --merges <base_sha>..<head_sha>
   ```
3. Diff patch collection:
   ```
   git --no-optional-locks diff --no-ext-diff --no-textconv --ignore-submodules=none --full-index -U3 <base_sha>..<head_sha>
   ```
4. Status check:
   ```
   git --no-optional-locks status --porcelain=v1 --untracked-files=all --ignored=no
   ```
5. Object verification:
   ```
   git --no-optional-locks cat-file -e <sha>^{commit}
   ```

**Ancestry Policy:**
- Name: **`DESCENDANT_OR_EQUAL_POLICY`**
- Implementation: `git merge-base --is-ancestor <base_sha> <head_sha>`
- Semantics: Passes if `base_sha` is an ancestor of `head_sha` OR if `base_sha == head_sha`.
- Implication of equality: Zero commits added by worker; diff is expected to be empty.

**Merge Commit Policy:**
- Name: **`MERGE_COMMITS_FORBIDDEN`**
- Implementation: `git rev-list --merges <base_sha>..<head_sha>`
- Semantics: Result must contain exactly 0 commit hashes; any merge commit triggers `POLICY_MERGE_COMMITS_REJECTED`.

**TOCTOU Mitigation:**
- Step 1 (Pre-Check): Record `HEAD` SHA and worktree status.
- Step 2 (Collection): Execute diff, ancestry, and file listing commands.
- Step 3 (Post-Check): Re-record `HEAD` SHA and re-verify status.
- Abort condition: If `HEAD` changed between steps 1 and 3, abort with `ERR_GIT_EVIDENCE_TOCTOU_MUTATION`.

**Status:** `DECISION_2 = APPROVED_AT_DESIGN_LEVEL`

---

### Decision 3: Two-Layer Verification Isolation Architecture

**Layer A — Command Specification & Authority:**
- Verification command, arguments, environment, and cwd derived strictly from immutable TaskContract `verification_requests` validated against `VerificationPolicyCatalog`.
- Worker has zero control over Layer A.

**Layer B — Process & Environment Isolation:**
- **Primary Mechanism:** Windows AppContainer Isolation + Job Object.
- **Secondary Fallback (only when AppContainer not available):** Restricted Token with Low Integrity Level. Requires external WFP (Windows Filtering Platform) firewall rules for network isolation (Restricted Tokens provide zero network isolation by default; this fallback cannot be used in production without external firewall configuration).

**Writable Sandbox Architecture:**
The fatal contradiction between read-only source tree and writable toolchain is resolved by external attempt sandbox root:
```
.supervisor/sandboxes/<attempt_id>/
  ├── tmp/            # TEMP, TMP redirect
  ├── gocache/        # GOCACHE redirect
  ├── gopath/         # GOPATH redirect
  ├── scratch/        # general output scratch
  └── source_snapshot/ # disposable read-write copy for tests requiring in-tree writes
```

**Audited Worktree:** Strictly `GENERIC_READ` (deny write, delete, append).
- If a test tool requires writing to the source tree, it MUST execute against a source snapshot:
  * Copied strictly from verified commit `HEAD` (tracked files only).
  * Excludes all untracked worker files.
  * Verified by SHA-256 content manifest.
  * Placed in `.supervisor/sandboxes/<attempt_id>/source_snapshot/`.
  * Destroyed atomically upon verification completion.

**Window Station & Desktop Isolation:**
- `CreateDesktopW` alone does NOT constitute window-station isolation.
- Required lifecycle:
  1. `CreateWindowStationW("SupervisorStation_...", 0, GENERIC_ALL, ...)` — create dedicated station.
  2. `CreateDesktopW("SupervisorDesktop_...", 0, 0, 0, GENERIC_ALL, ...)` — create desktop within station.
  3. Bind `STARTUPINFO.lpDesktop` to `"SupervisorStation_...\\SupervisorDesktop_..."`.
  4. Grant AppContainer SID access only to the dedicated station and desktop.
  5. Close and tear down upon process exit.

**AppContainer SID ACL Lifecycle:**
1. Create AppContainer via `CreateAppContainerProfile`.
2. Grant: Read-only ACE for Go SDK and source paths.
3. Grant: Read/write ACE for `.supervisor/sandboxes/<attempt_id>/` and subdirectories.
4. Deny: All access to user profile, credentials, secrets.
5. Verify: ACL probe before child process launch; fail-closed on probe error.
6. Cleanup: `DeleteAppContainerProfile` and restore/cleanup ACLs upon sandbox teardown.

**Compatibility Proof Matrix:**
| Vector | Behavior | Mechanism | Failure Probe |
| :--- | :--- | :--- | :--- |
| Go toolchain | Compiles and runs tests | Read-only SDK + writable sandbox caches | Write attempt to worktree = `ERROR_ACCESS_DENIED` |
| Subprocess spawning | Child processes within Job Object | Job Object hierarchy | Child exceeds resource limits = terminated |
| Source file reading | Reads test code | `GENERIC_READ` ACL | — |
| Network egress | Denied | AppContainer: zero network capabilities | Socket connect = `WSAEACCES` (10013) |
| Secret access | Denied | Deny ACL on credentials paths | File read `%USERPROFILE%/.ssh/id_rsa` = `ERROR_ACCESS_DENIED` |
| Source tree mutation | Zero pollution | Read-only worktree ACL | `git status` before/after verification must be byte-identical |

**Status:** `DECISION_3 = APPROVED_AT_DESIGN_LEVEL; DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`

---

### Decision 4: Deterministic PolicyEngine Semantics

**Finding Types and Mappings:**
| PolicyEngine Finding | Trigger | Outcome |
| :--- | :--- | :--- |
| `POLICY_SCOPE_VIOLATION` | File changed outside `allowed_scope` | Mandatory rejection in ReviewBundle |
| `POLICY_FORBIDDEN_SCOPE_VIOLATION` | File changed inside `forbidden_scope` | Mandatory rejection in ReviewBundle |
| `POLICY_MERGE_COMMITS_REJECTED` | Merge commit found in `rev-list --merges` | Mandatory rejection |
| `POLICY_BINARY_FILE_DETECTED` | Git diff reports binary content | Flagged for ChatGPT review |
| `POLICY_SYMLINK_MUTATION_DETECTED` | Symlink or file mode change detected | Flagged for ChatGPT review |
| `POLICY_TEST_EXECUTION_FAILED` | Verification exit code != 0 | Included in ReviewBundle as verified failure |
| `POLICY_TOCTOU_MUTATION` | HEAD changed during evidence collection | Attempt retry; zero evidence committed |

**Worker Report Identity Verification:**
1. `report.task_id` must match `durable task.task_id` (from immutable task record).
2. `report.attempt_id` must match `durable attempt.attempt_id`.
3. `report.base_sha` must match `immutable contract.base_sha`.
4. `report.head_sha` is a `WorkerClaim` until `GitEvidenceCollector` verifies the actual worktree HEAD.
5. `report.branch` is a `WorkerClaim` until verified against worktree binding.
6. WorkerReport schema does NOT contain `contract_id`; contract lineage is traced via `task_id` -> `attempt_id` -> `contract.base_sha`.

**Status:** `DECISION_4 = APPROVED_AT_DESIGN_LEVEL`

---

### Decision 5: Two-Phase Evidence State Progression & Persistence Transactions

**Model 1 Pipeline: Pure Collectors + Single CAS Orchestrator**

Phase P04B (`GitEvidenceCollector`, `PolicyEngine`) and Phase P04C (`VerificationRunner`) are **pure in-memory engines** with zero SQLite writes. Phase P04D is the sole SQLite persistence orchestrator.

**Sequential Schema Migrations:**
- Schema v6 (P04A): `worker_claims`, `attempt_workspace_bindings`
- Schema v7 (P04B): `evidence`, `policy_findings` (tables present, production write APIs wired in P04D only)
- Schema v8 (P04C): `actual_test_results` (table present, production write APIs wired in P04D only)
- Schema v9 (P04D): `review_artifacts`, `review_bundles`

**Task State Invariant During Collection:**
Task remains in `REPORT_READY` throughout all P04B and P04C execution.

**P04D Atomic CAS Transaction:**
```sql
BEGIN IMMEDIATE;
-- 1. CAS Check
UPDATE task_attempts SET status = 'EVIDENCE_READY', updated_at = ?
WHERE attempt_id = ? AND status = 'REPORT_READY';
-- 2. Insert Evidence, Policy Findings, Test Results, Artifact Metadata, ReviewBundle, Audit Event
COMMIT;
```
If any step fails, transaction rolls back; zero partial rows committed.

**One-Winner Reservation Lease:**
- Attempt-scoped exclusive lease acquired before executing external commands.
- Losing concurrent workers abort immediately with zero external command execution.
- Expired leases recovered by crash-recovery monitor.

**Normal Test Failure vs Infrastructure Failure:**
- Normal test failure (exit code != 0): Valid verified outcome; recorded in `actual_test_results`; task reaches `EVIDENCE_READY`; ReviewBundle reports failure to ChatGPT.
- Infrastructure failure (timeout, sandbox crash, OOM, tool not found): Attempt fails; task remains `REPORT_READY` for retry; zero evidence committed.

**Status:** `DECISION_5 = APPROVED_AT_DESIGN_LEVEL; DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY = OPEN`

---

### Decision 6: Reviewable Evidence Payload & Schema Reconciliation

**Durable Artifact Store:**
- Content-addressed filesystem: `.supervisor/artifacts/<sha256>`
- SQLite metadata table: `review_artifacts`
- Schema (fields): `artifact_id`, `attempt_id`, `kind`, `media_type`, `encoding`, `full_stream_sha256`, `captured_sha256`, `original_bytes`, `captured_bytes`, `is_truncated`, `durable_location`, `created_at`
- Dual hashing: `full_stream_sha256` (computed over complete stream) and `captured_sha256` (hash of stored prefix); never conflated.
- Atomic write-rename persistence: temp file -> SHA-256 verify -> rename to canonical path.
- P05 retrieval: strictly via opaque `artifact_id`; raw paths never exposed to ChatGPT tools.

**Canonical ReviewBundle Hashing:**
- Algorithm: **RFC 8785 (JSON Canonicalization Scheme - JCS)**.
- Input: RFC 8785 canonical byte representation of the ReviewBundle object **excluding** the `bundle_hash` field.
- Hash output: SHA-256 of the JCS byte stream.

**Schema Reconciliation (S1..S10):**
- S1: `test_results` replaces `details` throughout all schemas.
- S2: `review_artifacts` table added to Schema v9 (P04D).
- S3: `patch_artifact_id` (opaque) referenced in Git evidence.
- S4: `stdout_artifact_id`, `stderr_artifact_id` (opaque) referenced in `actual_test_results`.
- S5: RFC 8785 (JCS) formalized as canonical hashing algorithm.
- S6: `is_patch_truncated`, `is_stdout_truncated`, `is_stderr_truncated` flags mandated.
- S7: Opaque artifact retrieval service API for P05.
- S8: Binary file change manifest in ReviewBundle.
- S9: Domain struct definitions (`internal/domain`) synchronized.
- S10: Valid example JSON fixtures reconciled with Draft-07 schemas.

**Status:** `DECISION_6 = APPROVED_AT_DESIGN_LEVEL; DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`

---

## 3. Consequences

### Positive Consequences
- **Zero Private AO Database Access**: Binding formula is derived from public AO source facts, not private SQLite.
- **Reproducible Evidence**: Deterministic formulas for path, branch, diff collection, and artifact hashing produce identical results across reproducible audit cycles.
- **Fail-Closed Security**: AppContainer + Job Object + read-only source tree prevent source mutation, secret leakage, and network egress during verification.
- **Atomic Evidence**: Model 1 guarantees zero partial evidence rows; a failed P04D transaction leaves the system in a retry-able state.
- **P05 Isolation**: Opaque artifact IDs prevent raw filesystem traversal from being exposed to ChatGPT tools.

### Negative / Trade-Off Consequences
- **Bounded Proof Overhead**: Resolving Decision 1 requires executing the Worktree Binding Proof task before P04A can proceed.
- **Latency**: AppContainer + sandbox setup introduces 100-500ms overhead per verification attempt.
- **Disk I/O**: Content-addressed artifact store and sandbox teardown add disk operations beyond in-memory approaches.

### Invariants Preserved
- **`AUTOMATIC_RESTORE = DISABLED`** until verified operator principal reaches trusted boundary.
- **`LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`**: Isolated disposable proof does not constitute live user AO evidence.
- **`P04_TASK_CONTRACT = NOT_RELEASED`**: No implementation Task Contract is released until all five blockers are resolved.
- **`P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`**: Zero production Go code written until formal contract release.
- **`P05_CODE = NOT_AUTHORIZED`**: Phase P05 ChatGPT Tool Interface implementation remains blocked.
- **`STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`**: Stage B runtime catalog deferred.
