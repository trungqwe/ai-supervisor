# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL AUDIT 001 — ERRATUM 002

- **Erratum ID:** `P04-AUDIT-001-ERRATUM-002`
- **Target Documents:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md) (Section 2)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (Section 3.1)
- **Audited Baseline Commit Reference:** `a7e80d7c677467380cbb754429672705563ca056`
- **Date:** 2026-09-26
- **Status:** `FORMALLY_RECORDED`
- **Governance Mandate:** Append-Only Audit Trail ([`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), [`AGENTS.md`](../../AGENTS.md))

---

## 1. Statement of Erratum

In [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md) (Section 2) and carried forward into [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (Section 3.1), symbol citations and line ranges for `backend/internal/adapters/workspace/gitworktree/workspace.go` in upstream Agent Orchestrator were inaccurate and did not match pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` (v0.13.0).

Specifically:
1. `Workspace.managedPath` was cited at line 1887 with signature `func (w *Workspace) managedPath(cfg domain.WorkspaceConfig) string`. This was double-erroneous:
   - The configuration parameter is typed as `ports.WorkspaceConfig` (package `ports`), NOT `domain.WorkspaceConfig`.
   - The function returns `(string, error)`, NOT a bare `string`.
2. `Options.ManagedRoot` was cited at line 105; `Workspace.Create` was cited at lines 276, 287; `defaultSessionBranchName` was cited at line 1918; and `Workspace.Restore` was cited at lines 1142, 1150. These line citations derived from an unpinned branch/commit.
3. In addition, the runtime proof descriptions in Erratum 001 and Re-Audit 001 referred to caller-selected session IDs (`sess-proof-alpha`, `sess-proof-beta`). In the pinned Agent Orchestrator REST interface, `POST /api/v1/sessions` assigns server-generated session IDs (`resp.Session.ID`). Caller cannot inject custom session IDs at spawn time.

---

## 2. Authoritative Corrections for Pinned AO Commit `15e9ea9`

Inspection of `backend/internal/adapters/workspace/gitworktree/workspace.go` at pinned AO commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` establishes the following authoritative symbols, signatures, and permalinks:

### 2.1. `Options` Struct and `ManagedRoot`
- **Supervisor Reference Range:** Lines 64–73; `ManagedRoot` at line 68 (raw blob lines 71–80; `ManagedRoot` field at line 75).
- **Permalink:** [`backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80)
- **Concrete Source:**
  ```go
  // Options configures a gitworktree Workspace. ManagedRoot and RepoResolver are
  // required; Binary falls back to git from PATH.
  type Options struct {
      Binary       string
      ManagedRoot  string
      RepoResolver RepoResolver
      Logger       *slog.Logger
  }
  ```

### 2.2. `Workspace.Create`
- **Supervisor Reference Range:** Lines 230–262; invokes `w.managedPath(cfg)` at line 243 (raw blob lines 246–266; call at line 257).
- **Permalink:** [`backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266)
- **Signature & Call:**
  ```go
  func (w *Workspace) Create(ctx context.Context, cfg ports.WorkspaceConfig) (ports.WorkspaceInfo, error) {
      // ...
      path, err := w.managedPath(cfg)
      // ...
  }
  ```

### 2.3. `Workspace.Restore`
- **Supervisor Reference Range:** Lines 1045–1112; invokes `w.restorePath(cfg)` at line 1055 (raw blob lines 1097–1140; call at line 1105).
- **Permalink:** [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140)
- **Signature & Call:**
  ```go
  func (w *Workspace) Restore(ctx context.Context, cfg ports.WorkspaceConfig) (ports.WorkspaceInfo, error) {
      // ...
      path, err := w.restorePath(cfg)
      // ...
  }
  ```

### 2.4. `Workspace.managedPath`
- **Supervisor Reference Range:** Lines 1754–1762 (raw blob lines 1837–1846).
- **Permalink:** [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846)
- **Authoritative Signature & Implementation:**
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

### 2.5. `Workspace.restorePath`
- **Supervisor Reference Range:** Lines 1764–1768 (raw blob lines 1848–1853).
- **Permalink:** [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853)
- **Authoritative Signature & Implementation:**
  ```go
  func (w *Workspace) restorePath(cfg ports.WorkspaceConfig) (string, error) {
      if cfg.Path != "" {
          return w.validateManagedPath(cfg.Path)
      }
      return w.managedPath(cfg)
  }
  ```

### 2.6. `defaultSessionBranchName`
- **Supervisor Reference Range:** Lines 1783–1785 (raw blob lines 1868–1870).
- **Permalink:** [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870)
- **Authoritative Signature & Implementation:**
  ```go
  func defaultSessionBranchName(id domain.SessionID) string {
      return "ao/" + string(id)
  }
  ```

---

## 3. Session ID Generation and Wire Protocol Correction

As documented in Supervisor control plane types [`internal/ao/session_commands.go`](../../internal/ao/session_commands.go) and [`internal/ao/wire_types.go`](../../internal/ao/wire_types.go):
1. `CreateWorkerSession` issues `POST /api/v1/sessions` with wire body `wireSpawnWorkerRequest{ProjectID, Kind: "worker", Harness}`.
2. The endpoint returns `wireSpawnSessionResponse` containing `resp.Session.ID`, which is assigned authoritatively by Agent Orchestrator.
3. Callers do not provide or control session IDs.
4. Consequently, runtime proof procedures must **not** assume fixed names such as `sess-proof-alpha` or `sess-proof-beta`. The proof harness must capture the actual server-generated session IDs from the spawn responses, record the response evidence, and dynamically compute the expected worktree path (`<managedRoot>/<projectID>/<sessionID>`) and branch (`ao/<sessionID>`) from the runtime values.

---

## 4. Governance Integrity

Pursuant to [`AGENTS.md`](../../AGENTS.md) and [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md):
- Historical audit documents [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md), [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md), and [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) remain preserved without in-place modification.
- This Erratum 002 serves as the sole authoritative correction for citations in both Erratum 001 and Re-Audit 001.
