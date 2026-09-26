# PLAN-P04-WORKTREE-BINDING-PROOF: Bounded Empirical Proof Plan for Worktree Path Authority and Binding

- **Plan ID:** `PLAN-P04-WORKTREE-BINDING-PROOF`
- **Revision:** 1
- **Status:** `PROPOSED (PENDING_EXTERNAL_AUDIT)`
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Target Finding:** `P04-ARCH-R2-001` (WORKTREE_AUTHORITY_SEAM_AND_RUNTIME_BINDING_UNPROVEN)
- **Target Blocker:** `DESIGN_BLOCKER_P04_WORKTREE_BINDING`
- **Associated Draft Task Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`)
- **Author:** AI Engineering Supervisor Control Plane Team
- **Governance Authority:** [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), [`AGENTS.md`](../../AGENTS.md)
- **Source Registry & Reuse:** [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md), [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md), [`docs/22_MODULE_PROVENANCE.md`](../22_MODULE_PROVENANCE.md)
- **Upstream Pinned Source:** Agent Orchestrator (`v0.13.0`, commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`), file `backend/internal/adapters/workspace/gitworktree/workspace.go`

---

## 1. Executive Summary & Problem Statement

Phase P04 requires that the Supervisor Control Plane establish an authoritative, non-tamperable binding (`AttemptWorkspaceBinding`) between a given `TaskAttempt` and its underlying physical Git worktree on the local machine. This binding is essential because:
1. `GitEvidenceCollector` must execute read-only Git commands against the exact worktree where the worker committed changes.
2. `VerificationRunner` must bind the exact worktree commit to a read-only source tree and mount an external attempt sandbox for verification commands.

However, as established in Re-Audit 001 ([`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md)), while pinned AO source inspection establishes the deterministic naming formula `<managedRoot>/<projectID>/<sessionID>` and branch `ao/<sessionID>`, **static source inspection alone does not constitute runtime proof**.

Five crucial elements remain unproven at runtime:
1. **Configuration Seam**: From what authoritative configuration source does Agent Orchestrator obtain `managedRoot` on the host?
2. **Supervisor Authority**: Does the Supervisor process have authoritative and legal access to read that configuration value?
3. **Filesystem Namespace Identity**: Do AO and the Supervisor share the exact identical physical filesystem namespace, or does path virtualization, WSL, drive substitution, or containerization intervene?
4. **Daemon Restart & Restore Invariance**: Does AO restart or session restore deterministically preserve the exact same physical path without relocation, dangling references, or unexpected re-creation?
5. **Tamper & Alias Resistance**: Can directory junctions, NTFS symbolic links, hard links, or stale `git worktree` metadata allow path hijacking, TOCTOU aliasing, or collision?

This document defines the 3-track empirical proof plan to resolve these unproven seams definitively.

---

## 2. Three-Track Proof Architecture

```
+-----------------------------------------------------------------------------------+
|               PLAN-P04-WORKTREE-BINDING-PROOF ARCHITECTURE                        |
+-----------------------------------------------------------------------------------+
|                                                                                   |
|  TRACK 1: Static Pinned-Source Proof                                              |
|  - Pinned AO (15e9ea9) gitworktree/workspace.go inspection                        |
|  - Prove formulas: path = <managedRoot>/<projectID>/<sessionID>, branch = ao/<id> |
|  - Trace configuration seam for Options.ManagedRoot in AO startup wiring           |
|                                                                                   |
|  TRACK 2: Isolated Disposable Runtime Proof                                       |
|  - Standalone AO v0.13.0 in disposable sandbox (.supervisor/proof/ao_disposable/) |
|  - Dedicated temporary DB, managed root, temporary Git repo                       |
|  - ZERO user databases, ZERO user sessions, ZERO worker prompts / LLM calls        |
|  - Dual-session non-collision verification (same project, distinct sessions)      |
|  - Cross-verify REST API, path formula, git worktree list -z, and physical ID     |
|  - Test daemon restart, session restore, missing dir, junction/symlink alias      |
|                                                                                   |
|  TRACK 3: Supervisor Authority Determination & Conclusion                         |
|  - Outcome A: Trusted host configuration seam + startup validation probe          |
|  - Outcome B: Public AO REST API insufficient -> upstream capability / ADR req    |
|  - Outcome C: Mapping infeasible -> fail-closed redesign                          |
|  - Rule: Cannot choose Outcome A unless physical namespace & config proven        |
|                                                                                   |
+-----------------------------------------------------------------------------------+
```

---

## 3. Track 1 — Static Pinned-Source Proof

### 3.1. Upstream Symbols and Lines
In pinned upstream AO (`commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`), inspect `backend/internal/adapters/workspace/gitworktree/workspace.go`:

1. **`Options.ManagedRoot`** (line 105):
   ```go
   type Options struct {
       ManagedRoot string
       // ...
   }
   ```
2. **`Workspace.Create`** (lines 276, 287):
   Provisions the worktree directory via `managedPath(cfg)` and issues `git worktree add`.
3. **`Workspace.managedPath`** (line 1887):
   ```go
   func (w *Workspace) managedPath(cfg domain.WorkspaceConfig) string {
       return filepath.Join(w.managedRoot, string(cfg.ProjectID), string(cfg.SessionID))
   }
   ```
   **Proven Formula:** `managedPath = filepath.Join(managedRoot, projectID, sessionID)`.
4. **`defaultSessionBranchName`** (line 1918):
   ```go
   func defaultSessionBranchName(id domain.SessionID) string {
       return "ao/" + string(id)
   }
   ```
   **Proven Formula:** `defaultBranch = "ao/" + sessionID`.
5. **`Workspace.Restore`** (lines 1142, 1150):
   Evaluates whether the directory at `managedPath(cfg)` exists and contains valid git metadata.

### 3.2. Configuration Seam Tracing
Trace how `Options.ManagedRoot` is populated during AO binary startup:
- Identify CLI flags (e.g. `--workspace-dir` or `--managed-root`), environment variables (e.g. `AO_WORKSPACE_DIR`), or default settings (e.g. `~/.agent-orchestrator/workspaces`).
- Determine whether AO exposes this root path anywhere in its public HTTP API or whether it must be shared via host environment configuration.

---

## 4. Track 2 — Isolated Disposable Runtime Proof

### 4.1. Strict Isolation & Guardrails
To prevent any interference with live developer state:
1. **Dedicated Disposable Roots**:
   - `PROOF_ROOT`: `.supervisor/proof/ao_disposable/`
   - `PROOF_DB`: `.supervisor/proof/ao_disposable/ao_proof.db`
   - `PROOF_MANAGED_ROOT`: `.supervisor/proof/ao_disposable/worktrees/`
   - `PROOF_REPO`: `.supervisor/proof/ao_disposable/repo/`
2. **Zero Production Contamination**:
   - Zero interaction with the user's primary repository (`D:\TU_CODE\ai-supervisor`).
   - Zero interaction with the user's active AO database or running daemon.
   - Zero prompts sent to autonomous coding workers; zero LLM token generation.
3. **Strict Teardown**:
   - All disposable files are confined to `PROOF_ROOT` and removed or archived upon proof completion.

### 4.2. Test Matrix and Verification Procedures

#### Step 1: Initial Setup
1. Initialize a clean, disposable Git repository at `PROOF_REPO` with an initial commit on `main`.
2. Launch a disposable instance of `ao-daemon` (pinned v0.13.0) on an ephemeral port, configured with `PROOF_DB` and `PROOF_MANAGED_ROOT`.

#### Step 2: Dual Session Provisioning (Uniqueness & Non-Collision)
1. Register a test project (`proj-proof-01`) pointing to `PROOF_REPO`.
2. Provision Session 1 (`sess-proof-alpha`):
   - Record response from `POST /api/v1/sessions`.
   - Calculate expected path: `filepath.Join(PROOF_MANAGED_ROOT, "proj-proof-01", "sess-proof-alpha")`.
   - Verify directory exists and is a valid Git worktree.
3. Provision Session 2 (`sess-proof-beta`) on the same project:
   - Record response from `POST /api/v1/sessions`.
   - Calculate expected path: `filepath.Join(PROOF_MANAGED_ROOT, "proj-proof-01", "sess-proof-beta")`.
   - Verify directory exists and is a valid Git worktree.
4. **Collision Check**: Assert that Session 1 and Session 2 paths are distinct, mutually independent, and do not interfere with each other.

#### Step 3: Git & Filesystem Cross-Verification
Execute `git worktree list --porcelain -z` inside `PROOF_REPO` and verify:
1. Entry 1 matches `sess-proof-alpha`:
   - `worktree` path matches expected formula;
   - `branch` matches `refs/heads/ao/sess-proof-alpha`;
   - `HEAD` matches `PROOF_REPO` initial commit.
2. Entry 2 matches `sess-proof-beta`:
   - `worktree` path matches expected formula;
   - `branch` matches `refs/heads/ao/sess-proof-beta`;
   - `HEAD` matches `PROOF_REPO` initial commit.
3. Inspect `.git` file inside each worktree directory:
   - Confirm it contains `gitdir: <PROOF_REPO>/.git/worktrees/...`.
4. Verify physical file identity (ReFS/NTFS 128-bit `FileIdInfo` or Windows volume serial + file index) to prove that the path resolves directly to the worktree without intervening symlink or junction redirection.

#### Step 4: Daemon Restart & Session Restore Invariance
1. Terminate the disposable AO daemon.
2. Restart the disposable AO daemon with the identical `PROOF_DB` and `PROOF_MANAGED_ROOT`.
3. Call `POST /api/v1/sessions/sess-proof-alpha/restore`.
4. Verify:
   - Does the restore succeed?
   - Is the physical worktree path preserved identically?
   - Does `git worktree list` continue to report the exact same path and branch?

#### Step 5: Negative & Edge Case Probes
1. **Missing Directory Probe**: Delete one session directory manually; attempt restore; observe error handling and exit codes.
2. **Junction / Alias Probe**: Create an NTFS directory junction pointing to the worktree path; test whether path canonicalization (`filepath.EvalSymlinks`) identifies the junction.
3. **Stale Worktree Probe**: Prune worktrees externally via `git worktree prune`; observe AO behavior on subsequent session operations.

#### Step 6: Evidence Capture
- Record all terminal commands, exit codes, sanitized standard output/error, and SHA-256 hashes of all artifacts into the proof evidence log.

---

## 5. Track 3 — Supervisor Authority Determination

Based on the empirical findings from Track 1 and Track 2, the proof report must conclude with exactly one of the following determinations:

### Determination A: Trusted Host Configuration Seam + Startup Validation Probe
- **Conditions for Selection**:
  1. Track 2 conclusively proves that AO always constructs paths using `<managedRoot>/<projectID>/<sessionID>` and branches using `ao/<sessionID>`.
  2. Track 1 and Track 2 confirm that AO and Supervisor share the exact same physical host filesystem namespace.
  3. A reliable host configuration seam (e.g. `AO_MANAGED_ROOT` environment variable or shared host configuration file) exists and can be authoritatively read by the Supervisor.
  4. A host startup probe validates at daemon initialization that the configured `AO_MANAGED_ROOT` is accessible, matches AO's root, and contains zero unexpected junctions.
- **Resulting Architecture**: Supervisor can authoritatively compute `worktree_path` prior to dispatch, binding it into `attempt_workspace_bindings`.

### Determination B: Upstream Capability Request / New ADR Required
- **Conditions for Selection**:
  1. The public REST API of AO is found to be insufficient to guarantee deterministic path binding across different host environments.
  2. AO configuration is inaccessible to Supervisor, or path virtualization (WSL/containers) causes path divergence.
- **Resulting Architecture**: An upstream PR or capability request to Agent Orchestrator is required to expose `worktree_path` in `GET /api/v1/sessions/{id}` or a new ADR must be drafted.

### Determination C: Infeasible / Redesign Required
- **Conditions for Selection**:
  1. Worktree path allocation is non-deterministic, dynamic, or subject to unresolvable race conditions.
- **Resulting Architecture**: Fail-closed guardrails engage. The entire worktree binding model must be redesigned before Phase P04 can proceed.

> **CRITICAL RULE**: Determination A **CANNOT** be selected unless both Track 1 (static code) and Track 2 (runtime proof) provide conclusive, reproducible evidence of physical filesystem identity and deterministic configuration provenance.

---

## 6. Deliverable & Acceptance

The output of executing this plan will be:
1. `docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md` containing:
   - Full command logs, exit codes, sanitized paths, and artifact SHA-256 hashes.
   - Dual-session non-collision evidence.
   - Restart and restore verification evidence.
   - Formal conclusion (Determination A, B, or C).
2. The report will be submitted to the External Supervisor for formal evaluation.
