# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

- **Proposal ID:** `PROPOSAL-P04-001`
- **Revision:** 3 (Remediation of External Re-Audit 001)
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Status:** `PENDING_EXTERNAL_REVIEW`
- **Audit References:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (`REVISION_2_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md) (`REVISION_1_REQUIRED`)
- **Associated Proof Plan:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md)
- **Associated Draft Proof Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`)
- **Author:** AI Engineering Supervisor Control Plane Team
- **Governance Authority:** [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md)
- **Decision Precedence:** Level 3 (Canonical Architecture) / Level 2 (ADR Required)
- **Related ADRs:** [`ADR-006`](../adr/ADR-006-evidence-first-review.md), [`ADR-011`](../adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md), [`ADR-012`](../adr/ADR-012-task-contract-revision-and-attempt-binding.md), [`ADR-013`](../adr/ADR-013-trusted-verification-command-spec.md), [`DRAFT-ADR-018`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md)
- **Source Provenance:** [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md), [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md), [`docs/22_MODULE_PROVENANCE.md`](../22_MODULE_PROVENANCE.md)
- **Primary Traceability:** FR-007, FR-008, FR-009, FR-010, SEC-003, NFR-004, NFR-008, REC-007, REC-008

---

## 1. Executive Summary, Provenance & Reuse Boundaries

Phase P03 established the foundational integration between the Supervisor Control Plane and the upstream Agent Orchestrator (AO REST daemon API `v0.13.0`), achieving clean session provisioning, task dispatch saga, observation reconciliation, raw bounded workspace artifact transport (`GetWorkspaceFile` in `internal/ao/client.go`), and exclusive daemon host bootstrap.

Pursuant to the formal governance boundaries frozen across Phase P01, P02, and P03, and citing [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md) and [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md):
1. **Reuse of Upstream Capabilities**: Phase P04 strictly reuses:
   - Untrivial Agent Orchestrator public workspace file reading primitive (`GetWorkspaceFile` in `internal/ao/client.go`), avoiding private database queries or terminal scraping.
   - Upstream Git worktree isolation mechanisms, strictly prohibiting building a custom, unapproved worktree manager or querying private AO SQLite databases.
   - Existing Supervisor domain entities and SQLite store seams established in Phase P02 and P03 (`internal/domain/**`, `internal/store/**`).
2. **Zero-Trust Boundary**: AO session idleness (`AO_IDLE != REPORT_READY`) and worker execution output represent unverified worker hypotheses (`WORKER_CLAIMS`), not objective truth.
3. **P04 Ownership**: Phase P04 owns the exclusive domain authority to parse `worker-report.json`, enforce schema validation, extract `WorkerClaim` entities, collect independent Git evidence, evaluate TaskContract scope compliance via `PolicyEngine`, execute profile-constrained verification commands via `VerificationRunner`, persist immutable evidence rows, and assemble the synthesized `ReviewBundle` for ChatGPT supervisor review.
4. **Pre-Contract Blockers**: Prior to drafting and releasing any formal implementation Task Contract for Phase P04, five critical architectural seams must be evaluated, remediated, and approved:
   - `DESIGN_BLOCKER_P04_WORKTREE_BINDING`: Establishing an authoritative, non-tamperable binding between `TaskAttempt` and physical local workspace worktree. Status: **`OPEN_PENDING_BOUNDED_PROOF`** ([`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md)).
   - `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY`: Enforcing read-only, non-mutating, sanitized Git diff collection with `DESCENDANT_OR_EQUAL_POLICY`, `MERGE_COMMITS_FORBIDDEN`, and TOCTOU protection. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION`: Implementing two-layer verification command execution separating Layer A (command authority) from Layer B (Windows AppContainer untrusted code execution isolation with external attempt sandbox root and read-only source tree protection). Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION`: Reconciling structural drift across specification markdown, JSON schemas, valid examples, and domain entity types, ensuring inspectable evidence payloads via durable content-addressed artifact store and RFC 8785 canonical hashing. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY`: Enforcing **Model 1 Pipeline Atomicity**: P04B/P04C are pure in-memory collectors/runners with zero intermediate table writes, and P04D executes a single atomic SQLite CAS transaction persisting all evidence, findings, test results, artifact references, and transitioning state to `EVIDENCE_READY`. Status: **`OPEN`**.

---

## 2. Current-State Gap Matrix

| Deliverable / Component | Canonical Requirement | Code Seam Hiện Có (Baseline P03) | Phần Còn Thiếu Cần Xây Dựng (P04) | Owner Subtask | Dependencies |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **WorkerReport Ingestion** | FR-007, SEC-003 | `internal/ao/client.go` (`GetWorkspaceFile`), `internal/domain/worker_report.go` (schema parsing) | Handoff detection, schema validation, `WorkerClaim` entity extraction, `attempt_workspace_bindings` table, `worker_claims` table | P04A | Upstream AO `GetWorkspaceFile` |
| **Worktree Path Authority** | FR-007, SEC-003 | Pinned AO `gitworktree/workspace.go` | Empirical 3-track proof of `managedRoot` host config, shared namespace, and restore invariance; deterministic path resolution | Bounded Proof Task | Pinned AO `v0.13.0` |
| **Git Evidence Collection** | FR-007, FR-008 | Direct CLI invocation in test harnesses | Hardened, sanitized Git executor (`GIT_TERMINAL_PROMPT=0`, `--no-ext-diff`, `--no-textconv`), `DESCENDANT_OR_EQUAL_POLICY`, `MERGE_COMMITS_FORBIDDEN`, TOCTOU verification, bounded patch capture | P04B | Worktree Binding Authority |
| **Policy Engine** | FR-008, SEC-003 | Scope pattern parsing in `internal/domain` | Diff parsing, allowed/forbidden scope matching, binary/symlink/mode checks, `policy_findings` table | P04B | Git Evidence Collector |
| **Verification Runner** | FR-009, SEC-003 | Mock verification in domain tests | Windows AppContainer isolation, external sandbox root (`.supervisor/sandboxes/<attempt_id>/`), Window Station + Desktop isolation, read-only worktree ACLs, bounded logs, timeout enforcement | P04C | Verification Policy Catalog |
| **Durable Artifact Store** | FR-010, SEC-003 | Direct text storage | Content-addressed store (`.supervisor/artifacts/<sha256>`), SQLite metadata table `review_artifacts`, dual hashing (full stream vs prefix), opaque ID retrieval for P05 | P04D | SQLite WAL Store |
| **ReviewBundle Synthesis & Atomicity** | FR-010, NFR-008 | Domain struct definitions (`internal/domain`) | Model 1 Single CAS Orchestrator, RFC 8785 (JCS) canonical bundle hashing, `review_bundles` table, state transition to `EVIDENCE_READY` | P04D | P04A, P04B, P04C |

---

## 3. DESIGN_BLOCKER_P04_WORKTREE_BINDING

### 3.1. Authoritative Upstream Source Facts
In response to Re-Audit 001 and recorded in Erratum 001 ([`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md)), inspection of pinned upstream Agent Orchestrator (`commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) in `backend/internal/adapters/workspace/gitworktree/workspace.go` reveals:

1. **`Workspace.managedPath`** (line 1887):
   ```go
   func (w *Workspace) managedPath(cfg domain.WorkspaceConfig) string {
       return filepath.Join(w.managedRoot, string(cfg.ProjectID), string(cfg.SessionID))
   }
   ```
   Authoritatively proves the deterministic naming formula:
   ```text
   <managedRoot>/<projectID>/<sessionID>
   ```
2. **`defaultSessionBranchName`** (line 1918):
   ```go
   func defaultSessionBranchName(id domain.SessionID) string {
       return "ao/" + string(id)
   }
   ```
   Authoritatively proves the deterministic session branch formula:
   ```text
   ao/<sessionID>
   ```
3. **`Workspace.Create`** (lines 276, 287) and **`Workspace.Restore`** (lines 1142, 1150) use this exact managed path to execute `git worktree add` and restore existing sessions.
4. **`Options.ManagedRoot`** (line 105) configures the base workspace root.

**Conclusion:** The naming formula `<managedRoot>/<projectID>/<sessionID>` and branch `ao/<sessionID>` are proven static facts in pinned AO, not unverified chat assumptions.

### 3.2. Unproven Runtime Seams
Despite the static formula proof, five critical runtime aspects remain unproven:
1. **Host Configuration Source**: How is `managedRoot` configured on the host machine running AO (CLI flag, env var, default path)?
2. **Supervisor Authority**: Does the Supervisor process have authoritative, legal access to read that configuration value?
3. **Shared Filesystem Namespace**: Do AO and Supervisor execute within the exact identical physical filesystem namespace, or does path virtualization (WSL, Docker, drive substitution) intervene?
4. **Restart & Restore Invariance**: Does AO restart or session restore deterministically preserve the exact same physical path without relocation or dangling references?
5. **Tamper & Alias Resistance**: Can directory junctions, NTFS symbolic links, hard links, or stale `git worktree` metadata allow path hijacking, TOCTOU aliasing, or collision?

### 3.3. Prohibitions & Governance Constraints
- **Prohibition 1**: Supervisor **MUST NOT** query private AO SQLite databases directly.
- **Prohibition 2**: Supervisor **MUST NOT** build a custom, unapproved worktree manager.
- **Constraint**: `git worktree list --porcelain -z` **MUST NOT** be treated as sole authority on its own; it is strictly used to cross-check the formula against physical Git repository identity.

### 3.4. Resolution Strategy: Bounded Empirical Proof Task
Pursuant to Re-Audit 001 directives, this blocker is addressed via a dedicated empirical proof plan and draft contract:
- Plan: [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md)
- Draft Contract: [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (Status: `NOT_RELEASED`)
- Execution Tracks:
  * **Track 1**: Static pinned-source proof of lifecycle and configuration seams.
  * **Track 2**: Isolated runtime proof using AO v0.13.0 with dedicated disposable database, managed root, temporary Git repo, dual sessions, and zero autonomous agent coding.
  * **Track 3**: Formal authority conclusion choosing between Option A (trusted configuration seam + startup probe), Option B (upstream capability request / ADR), or Option C (fail-closed redesign). Option A cannot be selected unless runtime proof confirms identical physical namespace and configuration provenance.

Status: **`DESIGN_BLOCKER_P04_WORKTREE_BINDING = OPEN_PENDING_BOUNDED_PROOF`**.

---

## 4. DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY

### 4.1. Strict Read-Only Authority & Isolation
`GitEvidenceCollector` must collect objective Git state without mutating the worktree:
1. **Hardened Invocation Parameters**:
   - Explicit CLI flags: `--no-ext-diff`, `--no-textconv`, `--ignore-submodules=none`.
   - Injected Environment: `GIT_TERMINAL_PROMPT=0` (prevent interactive credential prompts), `LC_ALL=C`.
   - Stripped Environment: `GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_EXTERNAL_DIFF`, `GIT_DIFF_OPTS`.
2. **Fixed Subcommand Argument Vectors**:
   Every Git subcommand must have an explicit, immutable argument vector; no generic string formatting:
   - Diff Collection: `git --no-optional-locks diff --no-ext-diff --no-textconv --ignore-submodules=none --full-index -U3 <base_sha>..<head_sha>`
   - Status Check: `git --no-optional-locks status --porcelain=v1 --untracked-files=all --ignored=no`
   - Ancestry Check: `git --no-optional-locks merge-base --is-ancestor <base_sha> <head_sha>`
   - Merge Check: `git --no-optional-locks rev-list --merges <base_sha>..<head_sha>`
   - Object Verification: `git --no-optional-locks cat-file -e <head_sha>^{commit}`
3. **Binary Path Pinning**:
   - Resolve absolute path to `git.exe` at startup via trusted system PATH.
   - Prohibit resolving `git.exe` dynamically from user-writable directories.

### 4.2. Revision Ancestry: `DESCENDANT_OR_EQUAL_POLICY` & `MERGE_COMMITS_FORBIDDEN`
1. **Ancestry Evaluation**:
   `git merge-base --is-ancestor <base_sha> <head_sha>` returns exit code `0` if `base_sha` is an ancestor of `head_sha` OR if `base_sha == head_sha`.
   - Formally designated as: **`DESCENDANT_OR_EQUAL_POLICY`**.
   - If `base_sha == head_sha`, zero commits were added by the worker; diff must be empty.
2. **Merge Commit Policy**:
   - Formally mandated: **`MERGE_COMMITS_FORBIDDEN`**.
   - The worker worktree must represent a linear development sequence on top of `base_sha`.
   - Verified via: `git rev-list --merges <base_sha>..<head_sha>`. The output must contain exactly `0` commit hashes; if non-zero, PolicyEngine emits finding `POLICY_MERGE_COMMITS_REJECTED`.

### 4.3. TOCTOU Mitigation & Multi-Step Verification
To defeat Time-of-Check to Time-of-Use race conditions where a worker or external process continues writing to the worktree during evidence collection:
1. **Step 1 (Pre-Check)**: Record `HEAD` commit SHA and verify worktree status (`git status --porcelain`).
2. **Step 2 (Collection)**: Collect diff patch, numstat, and name-status between `base_sha` and `HEAD`.
3. **Step 3 (Post-Check)**: Re-read `HEAD` commit SHA and re-verify worktree status.
4. **Integrity Rule**: If `HEAD` changed or untracked/modified files appeared between Step 1 and Step 3, evidence collection aborts immediately with `ERR_GIT_EVIDENCE_TOCTOU_MUTATION`.

Status: **`DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY = OPEN`**.

---

## 5. DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION

### 5.1. Separation of Security Layers
The verification subsystem executes untrusted code built by workers. Security is strictly partitioned:
- **Layer A (Command Specification & Authority)**:
  - Worker has zero control over execution command, arguments, or environment.
  - Commands are derived strictly from immutable TaskContract `verification_requests` validated against pre-approved `VerificationPolicyCatalog`.
- **Layer B (Process & Environment Isolation)**:
  - Windows AppContainer Isolation paired with Job Object.
  - Restricted Token with Low Integrity Level is designated strictly as a secondary fallback requiring external WFP (Windows Filtering Platform) firewall rules (Restricted Tokens provide zero network isolation by default).

### 5.2. Relocating Writable Targets Outside Audited Worktree
To eliminate the fatal contradiction between read-only source code and writable toolchains (`go test` requires writing cache files):
1. **Audited Worktree Read-Only**:
   - The audited worktree is mounted strictly as **`GENERIC_READ`** (deny write, delete, append).
2. **External Attempt Sandbox Root**:
   - Dedicated attempt sandbox directory outside the worktree:
     ```text
     .supervisor/sandboxes/<attempt_id>/
     ```
   - Redirect all writable environment paths to subdirectories inside the sandbox:
     - `TEMP=.supervisor/sandboxes/<attempt_id>/tmp`
     - `TMP=.supervisor/sandboxes/<attempt_id>/tmp`
     - `GOCACHE=.supervisor/sandboxes/<attempt_id>/gocache`
     - `GOPATH=.supervisor/sandboxes/<attempt_id>/gopath`
3. **Source Snapshot for Tests Requiring Local Writes**:
   - If a test tool requires in-tree writing, it **MUST NOT** execute in the audited worktree.
   - It executes against a **Source Snapshot**:
     * Created strictly from verified commit `HEAD` (`git archive` or tracked file copy);
     * Strictly excludes untracked worker files;
     * Verified by SHA-256 manifest;
     * Placed inside `.supervisor/sandboxes/<attempt_id>/source_snapshot/`;
     * Destroyed immediately upon verification completion.

### 5.3. Window Station & Desktop Isolation
Calling `CreateDesktopW` alone does not provide window-station isolation. Untrusted code can enumerate windows on the parent window station (`WinSta0`) and perform message hooking.
- Formal Architecture:
  1. Create a dedicated, non-interactive Window Station: `CreateWindowStationW("SupervisorStation_...", 0, GENERIC_ALL, ...)`.
  2. Create a dedicated Desktop inside that station: `CreateDesktopW("SupervisorDesktop_...", 0, 0, 0, GENERIC_ALL, ...)`.
  3. Bind child process `STARTUPINFO.lpDesktop` to `"SupervisorStation_...\\SupervisorDesktop_..."`.
  4. Ensure AppContainer SID has access rights only to that dedicated desktop and station.
  5. Close and tear down both objects upon process exit.

### 5.4. Compatibility & Falsification Matrix

| Vector | Target Behavior | Isolation Mechanism | Falsification / Failure Probe |
| :--- | :--- | :--- | :--- |
| **Go Toolchain Execution** | Compiles and runs test packages | Read-only SDK path + writable `GOCACHE`/`GOPATH` in external sandbox | Test execution fails if `GOCACHE` write fails; verify caches are placed strictly in sandbox root |
| **Subprocess Spawning** | Test launches child processes | AppContainer allows child processes within Job Object hierarchy | Verify child process cannot exceed Job Object CPU/memory limits or escape AppContainer |
| **Source File Reading** | Reads test code and fixtures | `GENERIC_READ` ACL on audited worktree or source snapshot | Write attempt to worktree fails immediately with `ERROR_ACCESS_DENIED` |
| **Network Egress Denial** | Zero outbound network access | AppContainer network capabilities omitted (zero capabilities) | Socket connect attempt to localhost/external fails with `WSAEACCES` (10013) |
| **Secret / Credential Access** | Cannot access developer secrets | AppContainer SID deny ACL on user profile, `.ssh`, `.aws`, `.env` | File read attempt on `%USERPROFILE%/.ssh/id_rsa` fails with `ERROR_ACCESS_DENIED` |
| **Source Tree Mutation** | Zero pollution of Git worktree | Audited worktree is strictly read-only; writes isolated to sandbox | Worktree `git status` after verification is byte-for-byte identical to pre-verification status |

Status: **`DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`**.

---

## 6. DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION & REVIEWABLE EVIDENCE

### 6.1. Durable Content-Addressed Artifact Model
To support inspectable evidence for ChatGPT supervisor review without payload bloat or raw filesystem traversal:
1. **Storage Mechanism**:
   - Content-Addressed Filesystem Store: `.supervisor/artifacts/<sha256>`.
   - Immutable SQLite Metadata Table: `review_artifacts`.
2. **Schema for `review_artifacts`**:
   ```sql
   CREATE TABLE review_artifacts (
       artifact_id TEXT PRIMARY KEY,
       attempt_id TEXT NOT NULL,
       kind TEXT NOT NULL,              -- 'git_patch', 'test_stdout', 'test_stderr', 'manifest'
       media_type TEXT NOT NULL,        -- 'text/x-diff', 'text/plain', 'application/json'
       encoding TEXT NOT NULL,          -- 'utf-8', 'base64', 'binary'
       full_stream_sha256 TEXT NOT NULL,
       captured_sha256 TEXT NOT NULL,
       original_bytes INTEGER NOT NULL,
       captured_bytes INTEGER NOT NULL,
       is_truncated INTEGER NOT NULL,   -- 0 or 1
       durable_location TEXT NOT NULL,
       created_at TEXT NOT NULL,
       FOREIGN KEY (attempt_id) REFERENCES task_attempts(attempt_id) ON DELETE CASCADE
   );
   ```
3. **Dual Hashing Protocol**:
   - `full_stream_sha256`: Stream-computed over the entire output stream as bytes are generated.
   - `captured_sha256`: Hash of the bounded prefix captured and stored (up to configured byte limit).
   - Prohibited: Never claim that the hash of the bounded prefix is the hash of the full output.
4. **Filesystem Traversal Protection**:
   - Opaque IDs only (UUID / SHA-256).
   - Canonical root checking: `filepath.Clean` and prefix boundary enforcement.
   - Atomic persistence: write to temporary file in `.supervisor/artifacts/tmp/`, verify hash, atomically rename to final location.
5. **Phase P05 Retrieval Isolation**:
   - Phase P05 tools receive strictly opaque `artifact_id` values.
   - Raw filesystem paths are never leaked to ChatGPT or user tools.

### 6.2. ReviewBundle Payload & Canonical Hashing (RFC 8785)
1. **ReviewBundle Payload Content**:
   ```json
   {
     "bundle_id": "BUNDLE-...",
     "task_id": "TASK-...",
     "attempt_id": "ATTEMPT-...",
     "review_status": "READY",
     "git_evidence": {
       "base_sha": "...",
       "head_sha": "...",
       "files_changed_count": 3,
       "patch_artifact_id": "ART-PATCH-...",
       "patch_bounded_text": "diff --git a/... b/...",
       "is_patch_truncated": false
     },
     "policy_findings": [ ... ],
     "test_results": [
       {
         "request_id": "VR-01",
         "exit_code": 0,
         "duration_ms": 1250,
         "stdout_artifact_id": "ART-OUT-...",
         "stdout_bounded_text": "PASS\nok  ./...",
         "is_stdout_truncated": false
       }
     ],
     "bundle_hash": "..."
   }
   ```
2. **Canonical Hashing Specification**:
   - Canonical representation generated pursuant to **RFC 8785 (JSON Canonicalization Scheme - JCS)**.
   - `bundle_hash` is computed over the RFC 8785 canonical byte representation of the ReviewBundle object **excluding** the `bundle_hash` field itself.
   - Hashing Algorithm: SHA-256 of the JCS byte stream.

### 6.3. Reconciliation Items (S1..S10)
- **S1**: Replace `details` with `test_results` across all schemas.
- **S2**: Add `review_artifacts` table to Schema v9.
- **S3**: Reference `patch_artifact_id` in Git evidence.
- **S4**: Reference `stdout_artifact_id` and `stderr_artifact_id` in `actual_test_results`.
- **S5**: Formalize RFC 8785 (JCS) canonical hashing algorithm.
- **S6**: Add `is_patch_truncated`, `is_stdout_truncated`, `is_stderr_truncated` flags.
- **S7**: Establish opaque artifact retrieval service for P05.
- **S8**: Include binary file changes manifest in ReviewBundle.
- **S9**: Synchronize domain struct definitions in `internal/domain`.
- **S10**: Reconcile valid example JSON fixtures with Draft-07 schemas.

Status: **`DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`**.

---

## 7. DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY & PIPELINE ORCHESTRATION

### 7.1. Model 1: Pure Collectors & Single CAS Orchestrator
To guarantee zero partial evidence rows during failures, Phase P04 formally adopts **Model 1**:
1. **Pure In-Memory Collectors**:
   - Subtask P04B (`GitEvidenceCollector`, `PolicyEngine`) and P04C (`VerificationRunner`) are **pure in-memory collection and execution engines**.
   - P04B and P04C perform **ZERO SQLite table writes**.
2. **Sequential Schema Migrations**:
   - Schema v6 (P04A): `worker_claims`, `attempt_workspace_bindings`
   - Schema v7 (P04B): `evidence`, `policy_findings`
   - Schema v8 (P04C): `actual_test_results`
   - Schema v9 (P04D): `review_artifacts`, `review_bundles`
   - *Note:* While tables are migrated sequentially, production write APIs are wired exclusively in P04D.
3. **State Invariance**:
   - The task remains in **`REPORT_READY`** throughout the entire collection and verification pipeline.
4. **Single Atomic CAS Transaction (P04D)**:
   - When all evidence, findings, and test results are successfully gathered in memory, P04D executes a single SQLite transaction:
     ```sql
     BEGIN IMMEDIATE;
     -- 1. CAS Check: TaskAttempt must still be RUNNING/REPORT_READY with matching attempt_id
     UPDATE task_attempts
     SET status = 'EVIDENCE_READY', updated_at = ?
     WHERE attempt_id = ? AND status = 'REPORT_READY';
     -- 2. Insert Evidence Rows
     INSERT INTO evidence (...) VALUES (...);
     -- 3. Insert Policy Findings
     INSERT INTO policy_findings (...) VALUES (...);
     -- 4. Insert Test Results
     INSERT INTO actual_test_results (...) VALUES (...);
     -- 5. Insert Artifact Metadata
     INSERT INTO review_artifacts (...) VALUES (...);
     -- 6. Insert ReviewBundle
     INSERT INTO review_bundles (...) VALUES (...);
     -- 7. Insert Audit Event
     INSERT INTO task_audit_events (...) VALUES (...);
     COMMIT;
     ```
   - If any step fails, the entire transaction rolls back; zero partial rows are committed.

### 7.2. Concurrency Prevention: One-Winner Reservation Lease
To prevent redundant concurrent processes from executing expensive external commands:
1. Before starting Git diff or AppContainer verification, the orchestrator attempts to acquire an exclusive reservation lease:
   ```sql
   INSERT INTO task_verification_leases (task_id, attempt_id, lease_owner, expires_at)
   VALUES (?, ?, ?, datetime('now', '+300 seconds'));
   ```
2. If lease acquisition fails (conflict/already held), the losing worker **ABORTS IMMEDIATELY** and executes zero external commands.
3. If the lease expires due to process crash, a recovery monitor reclaims the lease.

### 7.3. Normal Test Failure vs Infrastructure Failure
1. **Normal Test Failure**:
   - If a verification command runs to completion and exits with code `1` (e.g. unit tests failed), this is a **valid verified outcome**.
   - Exit code, bounded stdout, and stderr are recorded in `actual_test_results`.
   - PolicyEngine emits `POLICY_TEST_EXECUTION_FAILED`.
   - Task transitions to `EVIDENCE_READY` normally; ReviewBundle clearly presents the test failure to ChatGPT for rejection.
2. **Runner Infrastructure Failure**:
   - If the toolchain cannot be found, AppContainer creation crashes, or execution exceeds `timeout_seconds`, this is a **`RUNNER_INFRASTRUCTURE_FAILURE`**.
   - Treated as an attempt failure under [`docs/14_FAILURE_RECOVERY.md`](../14_FAILURE_RECOVERY.md); task transitions to `FAILED` or initiates retry saga.

### 7.4. Transaction & Fault Injection Matrix

| Fault Point | Injected Failure | Expected Behavior | Recovery / Final State |
| :--- | :--- | :--- | :--- |
| **WorkerReport Read** | `GetWorkspaceFile` returns HTTP 404 or corrupted JSON | Attempt aborted; zero DB rows written; task transitions to `FAILED` | Attempt retry or escalation |
| **Git Collector TOCTOU** | Worker mutates worktree during diff collection | Collector detects HEAD shift; aborts in memory; zero DB rows written | Attempt retry; task remains `REPORT_READY` |
| **Verification Timeout** | Test command hangs beyond `timeout_seconds` | Job Object forcefully terminates process tree; marked as infrastructure timeout | Attempt fails; zero partial evidence committed |
| **Sandbox Crash** | Windows AppContainer fails to allocate desktop | Runner aborts; zero DB rows written | Attempt marked infrastructure failure |
| **Store CAS Conflict** | Another worker completed verification concurrently | CAS check matches 0 rows; transaction rolls back | Loser aborts; existing evidence preserved |
| **Power Loss during Commit** | System crash during SQLite commit | SQLite WAL rollbacks uncommitted transaction; zero partial rows | On recovery, task is in `REPORT_READY`; restart pipeline |

Status: **`DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY = OPEN`**.

---

## 8. Worker Report Identity & Verification

Pursuant to Re-Audit 001 directives:
1. The canonical `WorkerReport` schema (`docs/schemas/worker-report.schema.json`) **DOES NOT** contain `contract_id`.
2. The Supervisor **MUST NOT** claim that `contract_id` is validated directly from the WorkerReport.
3. Authoritative Lineage Verification:
   - `report.task_id == durable task.task_id` (from immutable task record).
   - `report.attempt_id == durable attempt.attempt_id` (from active task attempt record).
   - `report.base_sha == immutable contract.base_sha` (from authoritative TaskContract).
   - `report.head_sha` is treated strictly as a **`WorkerClaim`** until `GitEvidenceCollector` authoritatively verifies it against the worktree.
   - `report.branch` is treated strictly as a **`WorkerClaim`** until verified against worktree binding.
4. CAS Ingestion query relies on durable store values for contract, session, and generation, not worker claims.

---

## 9. Requirement Clarification: NFR-008 Performance SLA

### 9.1. Conflict Statement
Canonical requirement `NFR-008` states:
> *"The complete cycle from worker task completion handoff to ReviewBundle generation shall complete within 3 seconds for diffs under 500 lines."*

Running multi-package verification suites (`go test ./...`) under Windows AppContainer isolation routinely requires 5 to 30+ seconds, making a universal 3-second SLA physically impossible.

### 9.2. Proposed Canonical Amendment (Two Milestones)
Status: **`REQUIREMENT_CLARIFICATION_REQUIRED`** (canonical requirement is NOT modified in this proposal round).

Proposed two-milestone SLA for subsequent ADR reconciliation:
1. **Milestone 1 (Verification Execution)**:
   - Execution duration bounded strictly by the immutable TaskContract `timeout_seconds` (e.g. 300s).
2. **Milestone 2 (Durable Synthesis & Transition)**:
   - From durable `EVIDENCE_READY` to validated and persistent `REVIEWING` (ReviewBundle assembly, RFC 8785 hashing, SQLite CAS commit) shall complete within **3.0 seconds** (measured via in-process monotonic elapsed time).
3. **Monotonic Measurement Rule**:
   - Elapsed time must be measured via in-process monotonic clock (`time.Since`).
   - A process restart between milestones constitutes an SLA diagnostic failure; duration must never be inferred from unreliable wall clock differences.

---

## 10. Phased Scope Decomposition Plan

Following the resolution of the Bounded Worktree Proof, Phase P04 will execute across four sequential subtasks:

```
+-----------------------------------------------------------------------------------+
|                        PHASE P04 SUBTASK DECOMPOSITION                            |
+-----------------------------------------------------------------------------------+
|                                                                                   |
|  [PRE-REQUISITE]: PLAN-P04-WORKTREE-BINDING-PROOF (Bounded Empirical Proof)       |
|                                                                                   |
|  [P04A]: WorkerReport Ingestion, Validation & Claim Persistence                   |
|  - Ingest worker-report.json via GetWorkspaceFile                                 |
|  - Schema validation, WorkerClaim extraction                                      |
|  - Schema v6: worker_claims, attempt_workspace_bindings                           |
|                                                                                   |
|  [P04B]: Git Evidence Collector & Policy Engine (Pure In-Memory)                  |
|  - Hardened Git commands (GIT_TERMINAL_PROMPT=0, DESCENDANT_OR_EQUAL)             |
|  - MERGE_COMMITS_FORBIDDEN enforcement, TOCTOU before/after check                 |
|  - PolicyEngine evaluation, Schema v7 (evidence, policy_findings - no write API)  |
|                                                                                   |
|  [P04C]: Isolated Verification Runner (Pure In-Memory)                            |
|  - Windows AppContainer isolation + Job Object + Window Station/Desktop           |
|  - External attempt sandbox root (.supervisor/sandboxes/<id>/)                    |
|  - Read-only worktree mount, dual stream hashing, Schema v8 (test_results)        |
|                                                                                   |
|  [P04D]: Durable Artifact Store & Atomic Pipeline Orchestrator                    |
|  - Content-addressed store (.supervisor/artifacts/<sha256>)                      |
|  - Schema v9: review_artifacts, review_bundles                                    |
|  - One-winner reservation lease                                                   |
|  - Model 1 Single CAS SQLite Transaction (EVIDENCE_READY)                         |
|  - RFC 8785 (JCS) Canonical ReviewBundle hashing                                  |
|                                                                                   |
+-----------------------------------------------------------------------------------+
```

---

## 11. Governance Tracking & Gate

- **Proposal Status:** `PROPOSAL_P04_001 = REVISION_2_REQUIRED` (Remediated to Revision 3)
- **Active Gate:** `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_2`
- **Blocker Status:**
  - `DESIGN_BLOCKER_P04_WORKTREE_BINDING = OPEN_PENDING_BOUNDED_PROOF`
  - `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY = OPEN`
  - `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`
  - `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`
  - `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY = OPEN`
- **Execution Guardrails:**
  - `P04_TASK_CONTRACT = NOT_RELEASED`
  - `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
  - `P05_CODE = NOT_AUTHORIZED`
  - `AUTOMATIC_RESTORE = DISABLED`
  - `LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`
  - `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`
