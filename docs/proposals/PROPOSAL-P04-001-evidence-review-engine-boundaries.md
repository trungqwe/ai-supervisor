# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

- **Proposal ID:** `PROPOSAL-P04-001`
- **Revision:** 5 (Remediation of External Re-Audit 003)
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Status:** `PENDING_EXTERNAL_REVIEW`
- **Audit References:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_003.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_003.md) (`REVISION_4_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_003.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_003.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md) (`REVISION_3_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (`REVISION_2_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md) (`REVISION_1_REQUIRED`)
- **Associated Proof Plan:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 3)
- **Associated Draft Proof Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`, Model A Draft Lineage)
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
   - Upstream Agent Orchestrator public workspace file reading primitive (`GetWorkspaceFile` in `internal/ao/client.go`), avoiding private database queries or terminal scraping.
   - Upstream Git worktree isolation mechanisms, strictly prohibiting building a custom, unapproved worktree manager or querying private AO SQLite databases.
   - Existing Supervisor domain entities and SQLite store seams established in Phase P02 and P03 (`internal/domain/**`, `internal/store/**`).
2. **Zero-Trust Boundary**: AO session idleness (`AO_IDLE != REPORT_READY`) and worker execution output represent unverified worker hypotheses (`WORKER_CLAIMS`), not objective truth.
3. **P04 Ownership**: Phase P04 owns the exclusive domain authority to parse `worker-report.json`, enforce schema validation, extract `WorkerClaim` entities, collect independent Git evidence, evaluate TaskContract scope compliance via `PolicyEngine`, execute profile-constrained verification commands via `VerificationRunner`, persist immutable evidence rows, and assemble the synthesized `ReviewBundle` for ChatGPT supervisor review.
4. **Pre-Contract Blockers**: Prior to drafting and releasing any formal implementation Task Contract for Phase P04, critical architectural seams must be evaluated, remediated, and approved:
   - `DESIGN_BLOCKER_P04_WORKTREE_BINDING`: Establishing an authoritative, non-tamperable binding between `TaskAttempt` and physical local workspace worktree. Status: **`OPEN_PENDING_BOUNDED_PROOF`** ([`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md)).
   - `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY`: Enforcing read-only, non-mutating, sanitized Git diff collection with `DESCENDANT_OR_EQUAL_POLICY`, `MERGE_COMMITS_FORBIDDEN`, and TOCTOU protection. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION`: Implementing verification command execution with Windows AppContainer isolation, component-boundary handle verification, absolute external attempt sandbox root `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`, and read-only source tree protection. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION`: Reconciling structural drift across specification markdown, JSON schemas, valid examples, and domain entity types, ensuring inspectable evidence payloads via durable content-addressed artifact store (`<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`), `ON DELETE RESTRICT` immutability, coordinated GC lease protocol, and RFC 8785 canonical hashing. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY`: Enforcing **Model 1 Pipeline Atomicity**: P04B/P04C are pure in-memory collectors/runners with zero intermediate SQLite writes. P04D manages durable monotonic lease fencing in `task_verification_leases` (Schema v9) with `fencing_token`, and executes a single atomic SQLite CAS transaction updating `tasks.state` to `EVIDENCE_READY`. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_INERT_AO_HARNESS`: Pinned AO `backend/internal/domain/harness.go` contains no user-selectable inert harness. An inert test mode or upstream test seam must be formally approved before executing worker sessions. Status: **`OPEN`**.

---

## 2. Current-State Gap Matrix

| Architectural Area | Requirements | Current P03 Baseline State | Proposed P04 Target Architecture | Subtask Ownership | Upstream Seam |
|---|---|---|---|---|---|
| **Local Worktree Authority** | FR-008, SEC-003 | Only remote HTTP `GetWorkspaceFile` implemented; local worktree paths unbound | Pinned static AO inspection in `Untrivial-ai/agent-orchestrator` + isolated 3-track empirical proof (`PLAN-P04-WORKTREE-BINDING-PROOF`); dynamic server-generated session ID resolution | P04A | AO REST API & `backend/internal/adapters/workspace/gitworktree/workspace.go` |
| **Git Evidence Collection** | FR-008, SEC-003 | Raw Git CLI in test harness; no production collector | Read-only in-memory `GitCollector` (`DESCENDANT_OR_EQUAL_POLICY`, `MERGE_COMMITS_FORBIDDEN`, `GIT_TERMINAL_PROMPT=0`, diff boundary enforcement) | P04B | Host Git Binary (`Options.Binary`) |
| **Verification Runner** | FR-009, SEC-003 | Mock verification in domain tests | Windows AppContainer isolation, absolute external sandbox root (`<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`), component-boundary handle containment, Window Station + Desktop isolation, read-only worktree ACLs, bounded logs, timeout enforcement, typed policy profiles | P04C | Verification Policy Catalog |
| **Durable Artifact Store** | FR-010, SEC-003 | Direct text storage | Content-addressed store (`<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`), SQLite metadata table `review_artifacts` with `ON DELETE RESTRICT` and immutable triggers, dual hashing, coordinated GC lease protocol, tamper-checking retrieval, opaque ID retrieval for P05 | P04D | SQLite WAL Store & Filesystem |
| **ReviewBundle Synthesis** | FR-010, REC-008 | Draft schema in docs/schemas/ | Canonical JSON schema reconciliation, RFC 8785 (JCS) deterministic hashing, structured inspection endpoints | P04D | ChatGPT Tool Interface (P05) |
| **Pipeline Atomicity & Concurrency** | REC-007, REC-008 | Direct status transitions | Model 1: In-memory collection in P04B/P04C (zero SQLite writes); durable monotonic lease in `task_verification_leases` with non-resetting `fencing_token` and epoch expiry; single final atomic CAS updating `tasks.state` to `EVIDENCE_READY` in P04D | P04D | SQLite Transaction Management |

---

## 3. DESIGN_BLOCKER_P04_WORKTREE_BINDING

### 3.1. Problem & Finding P04-ARCH-R4-001 Analysis
In [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_003.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_003.md), the External Supervisor established that citations must reference exclusively the authoritative upstream repository `Untrivial-ai/agent-orchestrator` at pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`, using a single set of line ranges and official permalinks.

Inspection of `backend/internal/adapters/workspace/gitworktree/workspace.go` in `Untrivial-ai/agent-orchestrator` establishes:
1. **`Options` Struct** (lines 64–73):
   [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L64-L73`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L64-L73)
   ```go
   type Options struct {
       Binary       string
       ManagedRoot  string
       RepoResolver RepoResolver
       Logger       *slog.Logger
   }
   ```
2. **`Workspace.Create`** (lines 230–262; invokes `w.managedPath(cfg)` at line 243):
   [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L230-L262`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L230-L262)
3. **`Workspace.Restore`** (lines 1045–1112; invokes `w.restorePath(cfg)` at line 1055; worktree recreate fallback at lines 1069–1111):
   [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1045-L1112`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1045-L1112)
4. **`Workspace.managedPath`** (lines 1754–1762):
   [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1754-L1762`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1754-L1762)
   ```go
   func (w *Workspace) managedPath(cfg ports.WorkspaceConfig) (string, error) {
       var path string
       if cfg.Kind == domain.KindOrchestrator {
           prefix := resolvedSessionPrefix(cfg)
           path = filepath.Join(w.managedRoot, string(cfg.ProjectID), "orchestrator", prefix+"-orchestrator")
       } else {
           path = filepath.Join(w.managedRoot, string(cfg.ProjectID), string(cfg.SessionID))
       }
       return w.validateManagedPath(path)
   }
   ```
   Deterministic formula: `filepath.Join(managedRoot, projectID, sessionID)`. Parameter is `ports.WorkspaceConfig` and return type is `(string, error)`.
5. **`Workspace.restorePath`** (lines 1764–1768):
   [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1764-L1768`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1764-L1768)
   ```go
   func (w *Workspace) restorePath(cfg ports.WorkspaceConfig) (string, error) {
       if cfg.Path != "" {
           return w.validateManagedPath(cfg.Path)
       }
       return w.managedPath(cfg)
   }
   ```
6. **`defaultSessionBranchName`** (lines 1783–1785):
   [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1783-L1785`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1783-L1785)
   ```go
   func defaultSessionBranchName(id domain.SessionID) string {
       return "ao/" + string(id)
   }
   ```
   Deterministic formula: `"ao/" + sessionID`.

### 3.2. Server-Generated Session ID Wire Interface
As established in [`internal/ao/session_commands.go`](../../internal/ao/session_commands.go) and [`internal/ao/wire_types.go`](../../internal/ao/wire_types.go):
- `CreateWorkerSession` issues `POST /api/v1/sessions` with `wireSpawnWorkerRequest{ProjectID, Kind: "worker"}`.
- AO REST route returns `wireSpawnSessionResponse` with server-generated `resp.Session.ID`.
- Callers cannot pre-select or inject custom session IDs.
- The control plane and proof runtime must dynamically capture the returned `session_id` and compute the expected path and branch dynamically.

### 3.3. Restore Wire Contract
As defined in `internal/ao/session_commands.go` (`StopWorker`, `ResumeWorker`):
1. Session termination uses `POST /api/v1/sessions/{id}/kill`, requiring HTTP 200 and valid `wireKillSessionResponse`.
2. Bounded polling on `GET /api/v1/sessions/{id}` confirms terminal observation (`isTerminated=true`).
3. Session restore uses `POST /api/v1/sessions/{id}/restore`, requiring HTTP 200 (never 201), valid `restoreMode`, and matching top-level/nested session IDs.
4. Calling restore on an active session is strictly forbidden.

---

## 4. DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY

### 4.1. Threat Vectors in Worker Worktrees
An untrusted or buggy worker executing in a git worktree can:
1. Rewrite Git history (`git commit --amend`, `git rebase`, `git reset`);
2. Create merge commits obscuring diffs or introducing untracked changes;
3. Exploit Git hooks (`.git/hooks/*`, core.hooksPath);
4. Cause TOCTOU race conditions by modifying files during evidence collection;
5. Hang the evidence collector by triggering credential prompts (`git push`, `git fetch`).

### 4.2. Hardened In-Memory Git Collector Architecture
Phase P04 implements an independent in-memory `GitCollector` governed by strict security policies:
1. **DESCENDANT_OR_EQUAL_POLICY**:
   - The current commit `HEAD` of the worker session branch must be a direct descendant of or equal to the recorded `base_sha` from the `TaskContract`.
   - Verification command: `git merge-base --is-ancestor <base_sha> HEAD`. If non-zero, immediately flag `POLICY_VIOLATION_NOT_DESCENDANT` and transition task to `BLOCKED`.
2. **MERGE_COMMITS_FORBIDDEN**:
   - Verification command: `git rev-list --min-parents=2 <base_sha>..HEAD`.
   - If any merge commits exist, immediately flag `POLICY_VIOLATION_MERGE_COMMIT_DETECTED` and reject.
3. **Execution Guardrails**:
   - `GIT_TERMINAL_PROMPT=0` and `GIT_OPTIONAL_LOCKS=0` injected on all Git invocations.
   - Timeout strictly enforced at 30 seconds per Git command.
4. **Scope Compliance & Diff Boundary Enforcement**:
   - Worker report claims are compared against `git diff --name-only <base_sha>..HEAD` and `git status --porcelain -uall`.
   - Any modification in `forbidden_scope` or outside `allowed_scope` results in an immediate `VIOLATION` policy finding.
   - Any untracked or uncommitted changes trigger a policy warning or violation based on contract strictness.

---

## 5. DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION

### 5.1. Component-Boundary Handle Containment
To eliminate sibling-prefix vulnerabilities (e.g. `C:\root-sibling` falsely matching prefix `C:\root`):
1. **Handle-Based Pre-Create Verification:**
   - `SUPERVISOR_STATE_ROOT` must exist and be an absolute path.
   - Open root via OS handle (`CreateFileW`); obtain normalized physical path via `GetFinalPathNameByHandleW`, `VolumeSerialNumber`, and `FILE_ID_INFO`.
   - For non-existent child targets, traverse up to the nearest existing ancestor and open its handle.
   - Component-boundary validation: compute `rel, err := filepath.Rel(rootPhysicalPath, targetPhysicalPath)`. Reject if `err != nil`, or `rel == ".."`, or `strings.HasPrefix(rel, ".."+string(filepath.Separator))`. Raw string prefix matching (`strings.HasPrefix`) is strictly forbidden.
   - Reject volume mismatch and unexpected reparse points.
   - Assert strictly outside all Git worktrees, primary repository (`D:\TU_CODE\ai-supervisor`), live AO databases, and recovery folder (`D:\TU_CODE\ai-supervisor-user-wip-recovery\P03-EXIT-R2-001`).
2. **Safe Cleanup Protocol with Ownership Nonce:**
   - Root initialization writes `.supervisor-owner-marker.json` containing random `proof_run_id`, expected root physical identity, and cryptographic ownership nonce.
   - Pre-cleanup revalidation: open target handle immediately before deletion; verify final physical path remains inside run root and marker nonce matches.
   - Target must differ from state root and proof parent root.
   - Any mismatch or TOCTOU anomaly fails closed: abort cleanup, log diagnostic, leave target untouched.
3. **Falsification Test Matrix:**
   - Sibling Prefix: `<root>-attacker` vs `<root>` rejected.
   - Junction Swap: Reparse point swap detected and rejected.
   - Non-Existent Child: Deep child ancestor handle resolved correctly.
   - Case Alias: Resolves to identical physical `FILE_ID_INFO`.
   - Different Volume: Rejected.
   - Marker Replacement: Nonce mismatch aborts cleanup.

### 5.2. Windows AppContainer & Desktop Isolation
1. **Window Station & Desktop Isolation:**
   - Dedicated non-interactive Window Station via `CreateWindowStationW`, followed by `CreateDesktopW` within that station, preventing UI interaction, synthetic input injection, and screen capture.
2. **AppContainer SID ACL Lifecycle:**
   - Low integrity SID profile with read-only access to Go SDK / system runtimes.
   - Read/write access strictly to `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`.
   - Network capabilities denied.
   - Job Object terminates all child processes upon completion.
3. **Windows Capabilities Fallback:**
   - If host Windows environment lacks privileges for AppContainer or directory junctions, record `UNVERIFIED_CAPABILITY` or `BLOCKED` (never `PASS`).

### 5.3. Execution Authority via VerificationPolicyCatalog
Workers must never supply raw executable paths, shell command strings, API endpoints, roots, or harnesses.
Execution is mediated exclusively through typed profiles in the `VerificationPolicyCatalog`:
- Fixed host-owned executables, typed parameters, parameter enums, fixed cwd policies, hard timeouts, and maximum stdout/stderr capture bounds (e.g. 64KB).
- Worker profile must be approved profile `antigravity-standard`.
- `DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN` remains until an inert test mode or upstream test seam is established.

---

## 6. DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION & ARTIFACT STORE

### 6.1. Durable Content-Addressed Artifact Store
1. **Content-Address Key Determination:**
   - Files containing captured prefix bytes use `captured_sha256`:
     `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`
   - `full_stream_sha256` is stream metadata recorded in SQLite `review_artifacts`.
2. **Schema for `review_artifacts` (Schema v9, P04D):**
   ```sql
   CREATE TABLE review_artifacts (
       artifact_id TEXT PRIMARY KEY,
       attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
       kind TEXT NOT NULL CHECK (kind IN ('STDOUT_CAPTURE', 'STDERR_CAPTURE', 'GIT_DIFF_PATCH', 'TEST_REPORT', 'SCREENSHOT')),
       media_type TEXT NOT NULL,
       encoding TEXT NOT NULL CHECK (encoding IN ('UTF-8', 'BINARY', 'BASE64')),
       full_stream_sha256 TEXT NOT NULL,
       captured_sha256 TEXT NOT NULL,
       original_bytes INTEGER NOT NULL,
       captured_bytes INTEGER NOT NULL,
       is_truncated INTEGER NOT NULL CHECK (is_truncated IN (0, 1)),
       durable_location TEXT NOT NULL,
       created_at TEXT NOT NULL
   );

   CREATE TRIGGER trg_prevent_review_artifacts_update
   BEFORE UPDATE ON review_artifacts
   BEGIN
       SELECT RAISE(FAIL, 'review_artifacts rows are immutable');
   END;

   CREATE TRIGGER trg_prevent_review_artifacts_delete
   BEFORE DELETE ON review_artifacts
   BEGIN
       SELECT RAISE(FAIL, 'review_artifacts rows are immutable');
   END;
   ```
3. **Retrieval Tamper Detection:**
   - Retrieval must re-hash stored bytes and compare against `captured_sha256` before returning content.
4. **Coordinated GC Lease Protocol:**
   - Step 1: GC acquires exclusive `artifact_store_gc` lease.
   - Step 2: Identifies unreferenced file candidates in `<SUPERVISOR_STATE_ROOT>\artifacts`.
   - Step 3: Begins DB read transaction and rechecks `SELECT 1 FROM review_artifacts WHERE captured_sha256 = ?`.
   - Step 4: Atomically renames candidate file to `.quarantine/<captured_sha256>.<timestamp>`.
   - Step 5: Commits mark.
   - Step 6: Physical deletion occurs only after 24-hour grace period in `.quarantine/` and a final check.
   - Writer coordination: active writer checks canonical store, or if in quarantine, re-promotes or writes new staging file.
5. **Crash Matrix & Regression Design:**
   - Covers concurrent writer and GC, two attempts same captured bytes, same captured prefix but different full stream hash, crash after rename, DB rollback, metadata pointing to missing file, hash mismatch.

---

## 7. DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY & PIPELINE ORCHESTRATION

### 7.1. Model 1 Pipeline Atomicity
1. **Pure In-Memory Execution:**
   - Subtasks P04B (`GitCollector`, `PolicyEngine`) and P04C (`VerificationRunner`) perform strictly zero SQLite writes.
2. **Durable Monotonic Lease Fencing:**
   - Pre-command reservation lease in `task_verification_leases` (Schema v9, owned by P04D):
   ```sql
   CREATE TABLE IF NOT EXISTS task_verification_leases (
       task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE RESTRICT,
       attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
       fencing_token INTEGER NOT NULL DEFAULT 0,
       worker_id TEXT,
       state TEXT NOT NULL CHECK (state IN ('ACTIVE', 'RELEASED')),
       acquired_at TEXT,
       expires_at_epoch_ms INTEGER NOT NULL DEFAULT 0
   );
   ```
   - Row is kept by `task_id`. `fencing_token` is monotonically increasing and never resets (no DELETE row on release; prevents ABA token reuse).
   - Expiry uses numeric epoch milliseconds (`expires_at_epoch_ms`).
   - Acquisition pre-conditions: verify `tasks.state = 'REPORT_READY'`, `tasks.current_attempt` matching `task_attempts.attempt_number`, valid lineage, attempt not superseded.
   - Release: CAS updating `state = 'RELEASED'` matching exact `task_id`, `attempt_id`, `worker_id`, and `fencing_token`.
3. **Single Final Atomic SQLite CAS Transaction (P04D):**
   - Validates lease fencing token, active state, unexpired epoch, and updates `tasks.state`:
     ```sql
     UPDATE tasks
     SET state = 'EVIDENCE_READY',
         updated_at = ?
     WHERE task_id = ?
       AND state = 'REPORT_READY'
       AND current_attempt = ?
       AND EXISTS (
           SELECT 1 FROM task_verification_leases l
           WHERE l.task_id = tasks.task_id
             AND l.attempt_id = ?
             AND l.fencing_token = ?
             AND l.worker_id = ?
             AND l.state = 'ACTIVE'
             AND l.expires_at_epoch_ms >= ?
       );
     ```
   - If CAS fails, transaction rolls back cleanly, leaving zero evidence rows.
4. **Fault Matrix:**
   - Crash after acquire -> lease expires -> reclaimed with incremented token.
   - Stale owner resumes after expiry -> final CAS fails due to token mismatch.
   - TaskState or current_attempt changed before command -> acquire or final CAS fails.
   - Token ABA attempt -> prevented by monotonic increment without row deletion.

---

## 8. Worker Report Identity & Verification

1. **Format Compliance:**
   - Must be valid JSON adhering strictly to `docs/schemas/worker-report.schema.json`.
   - File location: `<worktree>/worker-report.json`.
2. **Identity Verification:**
   - `task_id` must match `tasks.task_id`.
   - `attempt_number` must match `tasks.current_attempt`.
   - `contract_id` must match active contract.
   - Any mismatch causes immediate rejection with structured diagnostic.

---

## 9. Requirement Clarification: NFR-008 Performance SLA

NFR-008 requires: *"ReviewBundle generation under 5 seconds for diffs < 1000 lines"*.
Clarification:
- Milestone 1 (Library Scope): Execution under 5 seconds measured in unit and integration test harnesses excluding external subprocess execution time.
- Milestone 2 (Daemon Scope): End-to-end SLA including caching, subprocess bounds, and WAL commits, validated during daemon deployment.

---

## 10. Phased Scope Decomposition Plan

| Subtask | Scope | Key Deliverables | Migration Dependency |
|---|---|---|---|
| **TASK-P04-001** | Bounded Empirical Proof for Worktree Authority | `docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md` | None |
| **TASK-P04-002** (P04A) | Worker Report Parser & Claims Extractor | `internal/evidence/report_parser.go`, schema validation, unit tests | Schema v6 (`claims`) |
| **TASK-P04-003** (P04B) | Git Evidence Collector & Policy Engine | `internal/evidence/git_collector.go`, `policy_engine.go` (pure in-memory) | Schema v7 (`evidence`, `policy_findings`) |
| **TASK-P04-004** (P04C) | Verification Runner & AppContainer Sandbox | `internal/evidence/verifier.go`, AppContainer isolation, Window Station+Desktop | Schema v8 (`actual_test_results`) |
| **TASK-P04-005** (P04D) | Pipeline Orchestrator, Artifact Store & ReviewBundle | `internal/evidence/orchestrator.go`, `artifact_store.go`, single atomic CAS | Schema v9 (`review_artifacts`, `review_bundles`, `task_verification_leases`) |

---

## 11. Governance Tracking & Gate

- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_4`
- **Current Status:** `PROPOSAL_P04_001 = REVISION_4_REQUIRED` (Remediated to Revision 5; submitted for External Supervisor Re-Audit 004).
- **Invariants:** `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`; `P05_CODE = NOT_AUTHORIZED`; `AUTOMATIC_RESTORE = DISABLED`.
