# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL AUDIT 001 — ERRATUM 003

- **Erratum ID:** `P04-AUDIT-001-ERRATUM-003`
- **Target Documents:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md)
- **Audited Baseline Commit Reference:** `f27d9ef0754a5b55d0a389b6f2e83a974af3a2c4`
- **Date:** 2026-09-26
- **Status:** `FORMALLY_RECORDED`
- **Governance Mandate:** Append-Only Audit Trail ([`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), [`AGENTS.md`](../../AGENTS.md))

---

## 1. Statement of Erratum and Supersession

In [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md) and carried into [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md), several citation, provenance, and protocol inaccuracies occurred:

1. **Incorrect Upstream Repository Provenance:**
   Erratum 002 cited permalinks referencing unauthorized fork `trungqwe/agent-orchestrator`, which is a fork/personal namespace, rather than the authoritative upstream repository `Untrivial-ai/agent-orchestrator` formally pinned in [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md).
2. **Confusing Dual-Line Number System:**
   Erratum 002 introduced two competing sets of line numbers ("Supervisor Reference Range" vs "raw blob lines"), creating ambiguity.
3. **Restore Wire Protocol Violation:**
   Erratum 002 suggested using `DELETE /api/v1/sessions/{id}` to induce a restorable state. Under Agent Orchestrator's wire protocol, `DELETE` permanently deprovisions/unregisters sessions. The valid lifecycle transition to create a restorable terminated session is `POST /api/v1/sessions/{id}/kill`, followed by bounded polling on `GET /api/v1/sessions/{id}` to observe `isTerminated=true` prior to issuing `POST /api/v1/sessions/{id}/restore` (which returns HTTP 200, not ambiguous HTTP 200 or 201).
4. **Incorrect Restore Recreate Line Attribution:**
   Erratum 002 incorrectly cited lines 1113–1140 for the worktree recreate logic of `Workspace.Restore`. In `workspace.go`, lines 1113–1140 belong to the `existingWorktree`/Create path; the recreate logic inside `Workspace.Restore` is located within lines 1069–1111.
5. **Fabricated Harness Literal:**
   Erratum 002 referenced `mock-inert` as an AO harness. Pinned Agent Orchestrator defines supported harnesses in `backend/internal/domain/harness.go`, and `mock-inert` does not exist.

This Erratum 003 formally records these errors and supersedes the citation provenance and wire protocol descriptions of Erratum 002.

---

## 2. Authoritative Pinned Citations and Official Permalinks

- **Authoritative Upstream Repository:** `github.com/Untrivial-ai/agent-orchestrator`
- **Pinned Commit:** `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` (v0.13.0)
- **Authoritative File:** `backend/internal/adapters/workspace/gitworktree/workspace.go`

Single authoritative symbols, line ranges, and official GitHub permalinks:

1. **`Options` Struct:**
   - **Range:** Lines 64–73
   - **Official Permalink:** [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L64-L73`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L64-L73)

2. **`Workspace.Create`:**
   - **Range:** Lines 230–262 (invokes `w.managedPath(cfg)` at line 243)
   - **Official Permalink:** [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L230-L262`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L230-L262)

3. **`Workspace.Restore`:**
   - **Range:** Lines 1045–1112 (invokes `w.restorePath(cfg)` at line 1055; worktree recreate fallback at lines 1069–1111)
   - **Official Permalink:** [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1045-L1112`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1045-L1112)

4. **`managedPath`:**
   - **Range:** Lines 1754–1762
   - **Signature:** `func (w *Workspace) managedPath(cfg ports.WorkspaceConfig) (string, error)`
   - **Official Permalink:** [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1754-L1762`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1754-L1762)

5. **`restorePath`:**
   - **Range:** Lines 1764–1768
   - **Signature:** `func (w *Workspace) restorePath(cfg ports.WorkspaceConfig) (string, error)`
   - **Official Permalink:** [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1764-L1768`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1764-L1768)

6. **`defaultSessionBranchName`:**
   - **Range:** Lines 1783–1785
   - **Signature:** `func defaultSessionBranchName(id domain.SessionID) string`
   - **Official Permalink:** [`https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1783-L1785`](https://github.com/Untrivial-ai/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1783-L1785)

---

## 3. Wire Protocol Lifecycle Rectification

As confirmed in [`internal/ao/session_commands.go`](../../internal/ao/session_commands.go) (`StopWorker`, `ResumeWorker`) and [`internal/ao/wire_types.go`](../../internal/ao/wire_types.go):
1. **Kill Operation:**
   - Client sends `POST /api/v1/sessions/{sessionId}/kill`.
   - Response requires HTTP 200 OK and valid `wireKillSessionResponse{OK: true, SessionID: ...}`.
2. **Terminal Observation Polling:**
   - Polling `GET /api/v1/sessions/{sessionId}` with bounded timeout until `isTerminated=true` is observed.
3. **Restart & Restore Sequence:**
   - Only after observing terminal state, restart disposable AO with identical DB and managed root.
   - Client sends `POST /api/v1/sessions/{sessionId}/restore`.
   - Response requires HTTP 200 OK (never HTTP 201), valid `restoreMode` (`native`, `saved_prompt`, or `fresh`), top-level `SessionID`, and matching nested `Session.ID`.
4. **Missing Directory Probe:**
   - Session 2 is explicitly killed via `POST /api/v1/sessions/{session_2_id}/kill`.
   - Bounded polling confirms terminal state.
   - Verified disposable directory is removed.
   - `POST /api/v1/sessions/{session_2_id}/restore` is invoked to characterize whether AO recreates the worktree (lines 1069–1111) or returns an error.
   - Calling restore on an active session is strictly forbidden.

---

## 4. Governance Integrity

Pursuant to [`AGENTS.md`](../../AGENTS.md) and [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), all historical audit records remain immutable. This Erratum 003 serves as the definitive correction superseding prior erroneous citations.
