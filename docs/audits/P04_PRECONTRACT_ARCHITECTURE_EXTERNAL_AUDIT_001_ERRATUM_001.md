# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL AUDIT 001 — ERRATUM 001

- **Erratum ID:** `P04-AUDIT-001-ERRATUM-001`
- **Target Document:** [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md)
- **Target Finding:** `P04-ARCH-R1-001` (WORKTREE_AUTHORITY_UNPROVEN)
- **Audited Commit Reference:** `08cc9c3e4f24404cc57a5c949d556959b710c38a`
- **Date:** 2026-09-26
- **Status:** `FORMALLY_RECORDED`
- **Governance Mandate:** Append-Only Audit Trail ([`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), [`AGENTS.md`](../../AGENTS.md))

---

## 1. Statement of Erratum

In Section 2.1 of [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md), under finding `P04-ARCH-R1-001` (Observation & Required Remediation Item 1), the audit record stated:

> *"1. Contrast against SOURCE_REGISTRY.md, REUSE_MATRIX.md, and pinned AO upstream code (backend/internal/adapters/agent/agy/agy.go)."*

### Correction:
The citation to `backend/internal/adapters/agent/agy/agy.go` was **erroneous**. That file implements the Antigravity agent adapter in Agent Orchestrator, not workspace/worktree management.

The **authoritative upstream source file** implementing Git worktree lifecycle management in pinned Agent Orchestrator (`v0.13.0`, commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) is:
```text
backend/internal/adapters/workspace/gitworktree/workspace.go
```

---

## 2. Key Symbols and Line Citations in Pinned AO Commit `15e9ea9`

Inspection of `backend/internal/adapters/workspace/gitworktree/workspace.go` in upstream repository Agent Orchestrator reveals the following concrete implementation symbols:

1. **`Options.ManagedRoot`** (line 105):
   ```go
   ManagedRoot string
   ```
   Specifies the base filesystem directory hosting managed worktrees.

2. **`Workspace.Create`** (lines 276, 287):
   Provisions a new physical worktree for a session and executes `git worktree add`.

3. **`Workspace.managedPath`** (line 1887):
   ```go
   func (w *Workspace) managedPath(cfg domain.WorkspaceConfig) string {
       return filepath.Join(w.managedRoot, string(cfg.ProjectID), string(cfg.SessionID))
   }
   ```
   Authoritatively calculates the local worktree filesystem path following the deterministic formula:
   ```text
   <managedRoot>/<projectID>/<sessionID>
   ```

4. **`defaultSessionBranchName`** (line 1918):
   ```go
   func defaultSessionBranchName(id domain.SessionID) string {
       return "ao/" + string(id)
   }
   ```
   Authoritatively constructs the session Git branch name following the deterministic formula:
   ```text
   ao/<sessionID>
   ```

5. **`Workspace.Restore`** (lines 1142, 1150):
   Restores session worktree state, reusing the existing managed path if valid.

---

## 3. Governance Effect & Static Evidence Boundary

1. **Append-Only Integrity**: Pursuant to [`AGENTS.md`](../../AGENTS.md) and [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), historical audit records are immutable. [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md) is not edited directly; this erratum serves as the authoritative rectification.
2. **Static Evidence Limitation**: Static source code inspection proves the existence of the formula `<managedRoot>/<projectID>/<sessionID>` and branch `ao/<sessionID>` in pinned AO commit `15e9ea9`. However, **static inspection does not constitute runtime proof**. It does not prove how `managedRoot` is configured on the host, whether Supervisor has authority to read it, whether AO and Supervisor share the exact physical filesystem namespace, whether restore/restart preserves the path across daemon cycles, or whether path aliases/junctions can be manipulated.
3. **Requirement for Dedicated Proof**: Empirical runtime proof across 3 tracks remains mandatory and is governed by [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md).
