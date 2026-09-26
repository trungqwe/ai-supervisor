# PROPOSAL-P04-001: Evidence & Review Engine Architecture, Execution Isolation, and ReviewBundle Reconciliation

- **Proposal ID:** `PROPOSAL-P04-001`
- **Revision:** 4 (Remediation of External Re-Audit 002)
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Status:** `PENDING_EXTERNAL_REVIEW`
- **Audit References:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md) (`REVISION_3_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (`REVISION_2_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md) (`REVISION_1_REQUIRED`)
- **Associated Proof Plan:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 2)
- **Associated Draft Proof Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`, Revision 2)
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
4. **Pre-Contract Blockers**: Prior to drafting and releasing any formal implementation Task Contract for Phase P04, five critical architectural seams must be evaluated, remediated, and approved:
   - `DESIGN_BLOCKER_P04_WORKTREE_BINDING`: Establishing an authoritative, non-tamperable binding between `TaskAttempt` and physical local workspace worktree. Status: **`OPEN_PENDING_BOUNDED_PROOF`** ([`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md)).
   - `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY`: Enforcing read-only, non-mutating, sanitized Git diff collection with `DESCENDANT_OR_EQUAL_POLICY`, `MERGE_COMMITS_FORBIDDEN`, and TOCTOU protection. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION`: Implementing two-layer verification command execution separating Layer A (command authority via VerificationPolicyCatalog) from Layer B (Windows AppContainer untrusted code execution isolation with absolute external attempt sandbox root `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\` and read-only source tree protection). Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION`: Reconciling structural drift across specification markdown, JSON schemas, valid examples, and domain entity types, ensuring inspectable evidence payloads via durable content-addressed artifact store (`<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`) and RFC 8785 canonical hashing. Status: **`OPEN`**.
   - `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY`: Enforcing **Model 1 Pipeline Atomicity**: P04B/P04C are pure in-memory collectors/runners with zero intermediate SQLite writes. P04D manages a pre-command reservation lease in `task_verification_leases` (Schema v9) with a fencing token, and executes a single atomic SQLite CAS transaction updating `tasks.state` to `EVIDENCE_READY`. Status: **`OPEN`**.

---

## 2. Current-State Gap Matrix

| Architectural Area | Requirements | Current P03 Baseline State | Proposed P04 Target Architecture | Subtask Ownership | Upstream Seam |
|---|---|---|---|---|---|
| **Local Worktree Authority** | FR-008, SEC-003 | Only remote HTTP `GetWorkspaceFile` implemented; local worktree paths unbound | Pinned static AO inspection + isolated 3-track empirical proof (`PLAN-P04-WORKTREE-BINDING-PROOF`); dynamic server-generated session ID resolution | P04A | AO REST API & `backend/internal/adapters/workspace/gitworktree/workspace.go` |
| **Git Evidence Collection** | FR-008, SEC-003 | Raw Git CLI in test harness; no production collector | Read-only in-memory `GitCollector` (`DESCENDANT_OR_EQUAL_POLICY`, `MERGE_COMMITS_FORBIDDEN`, `GIT_TERMINAL_PROMPT=0`, diff boundary enforcement) | P04B | Host Git Binary (`Options.Binary`) |
| **Verification Runner** | FR-009, SEC-003 | Mock verification in domain tests | Windows AppContainer isolation, absolute external sandbox root (`<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`), Window Station + Desktop isolation, read-only worktree ACLs, bounded logs, timeout enforcement, typed policy profiles | P04C | Verification Policy Catalog |
| **Durable Artifact Store** | FR-010, SEC-003 | Direct text storage | Content-addressed store (`<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`), SQLite metadata table `review_artifacts`, dual hashing (full stream metadata vs captured prefix), crash-consistent staging/rename protocol, opaque ID retrieval for P05 | P04D | SQLite WAL Store & Filesystem |
| **ReviewBundle Synthesis** | FR-010, REC-008 | Draft schema in docs/schemas/ | Canonical JSON schema reconciliation, RFC 8785 (JCS) deterministic hashing, structured inspection endpoints | P04D | ChatGPT Tool Interface (P05) |
| **Pipeline Atomicity & Concurrency** | REC-007, REC-008 | Direct status transitions | Model 1: In-memory collection in P04B/P04C (zero SQLite writes); pre-command lease in `task_verification_leases` with fencing token; single final atomic CAS updating `tasks.state` to `EVIDENCE_READY` in P04D | P04D | SQLite Transaction Management |

---

## 3. DESIGN_BLOCKER_P04_WORKTREE_BINDING

### 3.1. Problem & Finding P04-ARCH-R1-001 / R3-001 Analysis
In [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md), the External Supervisor identified that relying on `attempt_workspace_bindings.worktree_path` without proving the authoritative origin of that path creates a fatal security flaw: if an untrusted worker or caller could manipulate or alias that path, verification commands could execute against the wrong repository or outside containment.

Inspection of the pinned Agent Orchestrator repository (`v0.13.0`, commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`), specifically `backend/internal/adapters/workspace/gitworktree/workspace.go` (as formally rectified in Erratum 002), reveals the authoritative symbols and formulas:
1. **`Options` Struct and `ManagedRoot`** (lines 64–73; `ManagedRoot` at line 68; raw blob lines 71–80, `ManagedRoot` at line 75):
   [`backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80)
   ```go
   type Options struct {
       Binary       string
       ManagedRoot  string
       RepoResolver RepoResolver
       Logger       *slog.Logger
   }
   ```
2. **`Workspace.Create`** (lines 230–262; invokes `w.managedPath(cfg)` at line 243; raw blob lines 246–266, call at line 257):
   [`backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266)
3. **`Workspace.Restore`** (lines 1045–1112; invokes `w.restorePath(cfg)` at line 1055; raw blob lines 1097–1140, call at line 1105):
   [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140)
4. **`Workspace.managedPath`** (lines 1754–1762; raw blob lines 1837–1846):
   [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846)
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
   Authoritative formula: `filepath.Join(managedRoot, projectID, sessionID)`. Notice parameter is `ports.WorkspaceConfig` and return type is `(string, error)`.
5. **`Workspace.restorePath`** (lines 1764–1768; raw blob lines 1848–1853):
   [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853)
   ```go
   func (w *Workspace) restorePath(cfg ports.WorkspaceConfig) (string, error) {
       if cfg.Path != "" {
           return w.validateManagedPath(cfg.Path)
       }
       return w.managedPath(cfg)
   }
   ```
6. **`defaultSessionBranchName`** (lines 1783–1785; raw blob lines 1868–1870):
   [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870)
   ```go
   func defaultSessionBranchName(id domain.SessionID) string {
       return "ao/" + string(id)
   }
   ```
   Authoritative formula: `"ao/" + sessionID`.

### 3.2. Server-Generated Session ID Wire Interface
As established in [`internal/ao/session_commands.go`](../../internal/ao/session_commands.go) and [`internal/ao/wire_types.go`](../../internal/ao/wire_types.go):
- `CreateWorkerSession` issues `POST /api/v1/sessions` with `wireSpawnWorkerRequest{ProjectID, Kind: "worker", Harness}`.
- AO REST route returns `wireSpawnSessionResponse` with server-generated `resp.Session.ID`.
- Callers cannot pre-select or inject custom session IDs.
- The control plane and proof runtime must dynamically capture the returned `session_id` and compute the expected path and branch dynamically.

### 3.3. Empirical Proof Governance Resolution
To transition `DESIGN_BLOCKER_P04_WORKTREE_BINDING` from `OPEN` to `CLOSED`, this proposal mandates the execution of [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 2) under Task Contract [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`, Revision 2) across 3 tracks:
- **Track 1**: Static pinned-source proof of configuration seams and option resolution.
- **Track 2**: Isolated disposable runtime proof executing pinned AO v0.13.0 on an ephemeral port within an absolute state root `<SUPERVISOR_STATE_ROOT>\proof\p04-worktree-binding\<proof_run_id>`, testing dual sessions, non-collision, restart/restore invariance, and negative edge cases.
- **Track 3**: Authoritative determination selecting Determination A (trusted host seam + startup probe), Determination B (upstream PR / new ADR), or Determination C (fail-closed redesign).

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

### 5.1. The Contradiction of Read-Only Worktrees and Toolchain Writes
Remediation round 1 mandated that audited worktrees must be read-only during verification. However, compilers and test harnesses (`go test`) require writable directories for build caches and temporary files (`GOCACHE`, `GOPATH`, `TEMP`, `TMP`).

### 5.2. Absolute External Attempt Sandbox Architecture
Phase P04 resolves this by mandating an absolute, host-injected state root architecture:
1. **Root Hierarchy:**
   - Absolute Host State Root: `<SUPERVISOR_STATE_ROOT>`
   - Attempt Sandbox Root: `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`
   - Disposable Subdirectories:
     * `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\tmp`
     * `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\gocache`
     * `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\gopath`
     * `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\source_snapshot`
2. **Pre-Operation Containment Verification:**
   Before any sandbox creation, write, or cleanup:
   - Canonicalize path using `filepath.Clean` and `filepath.EvalSymlinks`.
   - Verify physical containment strictly within `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>`.
   - Verify strictly outside all Git worktrees.
   - Verify strictly outside primary repository (`D:\TU_CODE\ai-supervisor`).
   - Verify strictly outside AO live database and live managed root.
   - Verify strictly outside user recovery directory (`D:\TU_CODE\ai-supervisor-user-wip-recovery\P03-EXIT-R2-001`).
3. **Ownership Marker & Safe Cleanup:**
   - Write `.supervisor-owner-marker.json` inside the sandbox root at initialization.
   - Recursive cleanup permitted ONLY after containment proof passes and marker ownership matches `attempt_id`.
   - If physical containment or identity cannot be proven, cleanup must abort without deleting the target.
4. **Read-Only Audited Worktree:**
   - The audited worktree is mounted strictly read-only (`GENERIC_READ`).
   - All toolchain write environment variables (`TEMP`, `TMP`, `GOCACHE`, `GOPATH`) are explicitly redirected into the external sandbox.
5. **Disposable Source Snapshot for Modifying Tests:**
   - For verification tests requiring in-tree file modifications, a source snapshot is extracted from verified Git `HEAD` into `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\source_snapshot\`.
   - Untracked worker files are ignored; only tracked Git content is copied.
   - The snapshot is verified via SHA-256 manifest and deleted during attempt cleanup.

### 5.3. Windows AppContainer & Desktop Isolation
1. **Window Station & Desktop Isolation:**
   - Calling `CreateDesktopW` alone does not prevent window message interception if attached to the interactive window station.
   - P04 mandates creating a dedicated non-interactive Window Station via `CreateWindowStationW`, followed by `CreateDesktopW` within that station, preventing UI interaction, synthetic input injection, and screen capture.
2. **AppContainer SID ACL Lifecycle:**
   - Create ephemeral AppContainer profile with low integrity SID.
   - Grant read-only access to Go SDK / system runtimes.
   - Grant read/write access strictly to `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`.
   - Deny network capabilities (no internet client, no private network access).
   - On completion, terminate Job Object and delete AppContainer profile.
3. **Windows Capabilities Fallback:**
   - If host Windows environment lacks privileges to create AppContainer or directory junctions, the capability is recorded as `UNVERIFIED_CAPABILITY` or `BLOCKED` (never recorded as `PASS`).

### 5.4. Execution Authority via VerificationPolicyCatalog (Option B)
Workers must never supply raw executable paths, shell command strings, API endpoints, or roots.
Execution is mediated exclusively through typed profiles in the `VerificationPolicyCatalog`:
- Each profile defines:
  * Profile ID (e.g. `probe-git-readonly`, `probe-ao-ephemeral-runner`, `probe-fs-containment`);
  * Typed parameters and parameter enums;
  * Strict cwd policy (e.g. confined to attempt sandbox);
  * Hard execution timeout;
  * Pinned executable provenance;
  * Maximum stdout/stderr capture bounds (e.g. 64KB).
- Stage B policy checks must be confirmed against live catalog at dispatch time (`UNVERIFIED` until registered).
- Pinned AO harness mode must be proven inert (zero LLM calls, zero credentials, zero prompts) before execution (`DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN`).

---

## 6. DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION & ARTIFACT STORE

### 6.1. Durable Content-Addressed Artifact Store
To support inspectable verification evidence without payload bloat or raw filesystem traversal:
1. **Content-Address Key Determination:**
   - If an artifact file contains captured prefix bytes (truncated stream), its canonical path uses `captured_sha256`:
     `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`
   - `full_stream_sha256` is strictly stream metadata recorded in SQLite `review_artifacts`.
   - If a path uses `full_stream_sha256`, it must store the exact, untruncated full stream bytes.
2. **Schema for `review_artifacts` (Schema v9, P04D):**
   ```sql
   CREATE TABLE review_artifacts (
       artifact_id TEXT PRIMARY KEY,
       attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE CASCADE,
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
   ```
3. **Crash-Consistent Protocol:**
   - Step 1: Write bytes to staging file `<SUPERVISOR_STATE_ROOT>\artifacts\.staging\<uuid>`.
   - Step 2: `fsync` staging file and parent directory.
   - Step 3: Atomic rename from staging path to canonical path `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`.
   - Step 4: Record metadata row in SQLite `review_artifacts` inside final CAS transaction.
   - Note: If SQLite rolls back or host crashes, the rename leaves an unreferenced orphan file. SQLite rollback cannot rollback filesystem renames.
   - Startup GC: Background cleanup removes unreferenced staging files and unreferenced canonical artifacts after a 24-hour grace period following canonical containment verification.

### 6.2. Crash Matrix

| Crash Boundary | Filesystem State | SQLite State | Recovery Action on Startup / Resume |
|---|---|---|---|
| **1. Before staging write** | No file created | No DB changes | Clean; attempt failed prior to execution |
| **2. During staging write / before fsync** | Partial staging file in `.staging/<uuid>` | No DB changes | Startup GC removes stale staging file after grace period |
| **3. After fsync / before atomic rename** | Complete staging file in `.staging/<uuid>` | No DB changes | Startup GC removes stale staging file after grace period |
| **4. After atomic rename / before DB BEGIN** | Canonical file at `<captured_sha256>` | No DB changes | File is unreferenced orphan; startup GC cleans after 24h grace period |
| **5. During SQLite INSERTs / before COMMIT** | Canonical file at `<captured_sha256>` | Transaction active in WAL | Transaction rolls back automatically; file becomes orphan cleaned by GC |
| **6. On SQLite COMMIT failure** | Canonical file at `<captured_sha256>` | Rolled back | File becomes orphan cleaned by GC; attempt transitions to retry/failed |
| **7. After SQLite COMMIT success** | Canonical file at `<captured_sha256>` | Committed in WAL | Fully durable and consistent; review bundle references artifact |

### 6.3. Deduplication & Opaque Retrieval
- **Byte-Exact Deduplication:** If an artifact with identical `captured_sha256` already exists, filesystem write is skipped; SQLite metadata records the new attempt reference.
- **Retrieval Isolation:** Phase P05 and ChatGPT supervisor access artifacts exclusively via opaque `artifact_id` over the Control Plane REST API. Raw filesystem paths are never exposed.
- **Canonical Hashing:** `ReviewBundle` deterministic hashing is formalized using **RFC 8785 (JSON Canonicalization Scheme - JCS)**. All fields except `bundle_hash` are serialized under RFC 8785 rules and hashed via SHA-256.

---

## 7. DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY & PIPELINE ORCHESTRATION

### 7.1. Model 1 Pipeline Atomicity
To guarantee zero partial evidence rows during pipeline failures:
1. **Pure In-Memory Execution:**
   - Subtasks P04B (`GitCollector`, `PolicyEngine`) and P04C (`VerificationRunner`) perform **strictly zero SQLite writes**.
   - They execute in-memory and return structured Go structs (`GitEvidenceResult`, `PolicyEvaluationResult`, `TestExecutionResult`).
2. **Pre-Command Verification Reservation Lease:**
   - To prevent concurrent verification processes from executing external commands simultaneously, P04D acquires a reservation lease in a separate pre-command SQLite transaction:
   ```sql
   CREATE TABLE IF NOT EXISTS task_verification_leases (
       task_id TEXT PRIMARY KEY REFERENCES tasks(task_id) ON DELETE CASCADE,
       attempt_id TEXT NOT NULL REFERENCES task_attempts(attempt_id) ON DELETE CASCADE,
       fencing_token INTEGER NOT NULL,
       worker_id TEXT NOT NULL,
       acquired_at TEXT NOT NULL,
       expires_at TEXT NOT NULL
   );
   ```
   - Acquisition logic:
   ```sql
   INSERT INTO task_verification_leases (task_id, attempt_id, fencing_token, worker_id, acquired_at, expires_at)
   VALUES (?, ?, 1, ?, ?, ?)
   ON CONFLICT(task_id) DO UPDATE SET
       fencing_token = task_verification_leases.fencing_token + 1,
       attempt_id = excluded.attempt_id,
       worker_id = excluded.worker_id,
       acquired_at = excluded.acquired_at,
       expires_at = excluded.expires_at
   WHERE task_verification_leases.expires_at < excluded.acquired_at;
   ```
   - If 0 rows are affected, another active process holds the lease; the loser is denied execution and executes zero external commands.
3. **State Invariant During Collection:**
   - The task remains in `state = 'REPORT_READY'` on `tasks` throughout the entire collection and verification pipeline.
4. **Single Final Atomic SQLite CAS Transaction (P04D):**
   - After in-memory collection and artifact staging are complete, P04D opens a single SQLite transaction:
     * Verify `task_verification_leases` fencing token matches current worker and has not expired;
     * Insert into `evidence`;
     * Insert into `policy_findings`;
     * Insert into `actual_test_results`;
     * Insert into `review_artifacts`;
     * Insert into `review_bundles`;
     * Insert into `task_audit_events`;
     * Delete or release `task_verification_leases`;
     * Execute CAS update on `tasks.state`:
       ```sql
       UPDATE tasks
       SET state = 'EVIDENCE_READY',
           updated_at = ?
       WHERE task_id = ?
         AND state = 'REPORT_READY'
         AND current_attempt = ?;
       ```
   - If CAS fails (0 rows affected), transaction rolls back, leaving zero evidence rows, and staged artifacts become unreferenced orphans.
   - Stale resumed workers cannot commit evidence if their lease was reclaimed, as the fencing token will not match.

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

- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_3`
- **Current Status:** `PROPOSAL_P04_001 = REVISION_3_REQUIRED` (Remediated to Revision 4; submitted for External Supervisor Re-Audit 003).
- **Invariants:** `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`; `P05_CODE = NOT_AUTHORIZED`; `AUTOMATIC_RESTORE = DISABLED`.
