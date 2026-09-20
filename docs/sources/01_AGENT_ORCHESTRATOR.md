# SOURCE DOSSIER: 01 — UNTRIVIAL AGENT ORCHESTRATOR

> **Authority**: Upstream Source Evidence Dossier  
> **Status**: Literal Evidence Corrected (Post-Re-Audit #4 Patch)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `Untrivial-ai/agent-orchestrator` | GitHub API |
| **Role in Architecture** | Execution Control Plane (Process daemon, ConPTY, Git worktrees) | ADR-002, docs/04_ARCHITECTURE.md |
| **Pinned Documentation Version** | `v0.13.0` | GitHub Tag `v0.13.0` |
| **Pinned Commit SHA** | `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` | GitHub API Commit Verification |
| **Commit Author Date (UTC)** | `2026-09-12T06:30:25Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-09-12T06:30:25Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `Apache-2.0` | `LICENSE` in repository root |
| **Usage Terms** | Open source under Apache License 2.0 | `LICENSE` file |
| **Local Runtime Status** | `RUNTIME_UNTESTED` | Deferred to Phase P01 Track P01-A |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: AO-CLAIM-001
Claim: Untrivial AO provides native Windows ConPTY terminal emulation for spawned processes.
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/adapters/runtime/conpty/host_conpty_windows.go
Section / symbol: newConPTY
Verification: VERIFIED
Confidence: HIGH
Notes: Uses github.com/aymanbagabas/go-pty to initialize Windows ConPTY, resize buffers, and capture process exit codes.

Claim ID: AO-CLAIM-002
Claim: Untrivial AO provides Git worktree isolation with path traversal checks and containment guards.
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/adapters/workspace/gitworktree/workspace.go
Section / symbol:
  validateManagedPath
  managedPath
Verification: VERIFIED
Confidence: HIGH
Notes: Evaluates physical absolute paths, checks that paths remain within w.managedRoot, and creates dedicated git worktree directories per session.

Claim ID: AO-CLAIM-003
Claim: Untrivial AO exposes a loopback REST API for daemon health, project registration, session management, and process kill/restore.
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/httpd/router.go
Section / symbol:
  NewRouterWithControl
  mountHealth
Verification: VERIFIED
Confidence: HIGH
Notes: NewRouterWithControl is the sole exported router constructor. It calls mountHealth, mountTerminalMux, mountControl, mountAgentSwitchPolicyControl, mountTelemetry, mountMobile, mountMobileDevices, and api.Register to wire routes. Exact REST endpoint strings are documented in UPSTREAM_CONTRACT_BASELINE.md.

Claim ID: AO-CLAIM-004
Claim: AO's built-in Antigravity CLI adapter fulfils the ports.Agent interface and builds launch and restore argv arrays; it does not natively enforce structured output schema.
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/adapters/agent/agy/agy.go
Section / symbol:
  GetLaunchCommand
  GetRestoreCommand
Verification: VERIFIED
Confidence: HIGH
Notes: agy.go defines the Plugin struct and func New() *Plugin. GetLaunchCommand and GetRestoreCommand are the interface-mandated methods for argv construction (defined in backend/internal/ports/agent.go as the Agent interface). GetAgentHooks is also implemented (hooks.go). Integration gap regarding structured WorkerReport collection is documented as P01_PROOF_REQUIRED under Track P01-C.
```

---

# 3. Adopted Concepts vs. Forbidden Duplications

### Adopted Concepts:
- ConPTY Windows terminal management (leveraged via AO daemon).
- Git worktree filesystem isolation per worker session.
- Process lifecycle management (`/kill`, `/restore`, session recovery).

### Forbidden Duplications:
- Do NOT re-implement local ConPTY allocation or process monitoring in Supervisor code.
- Do NOT re-implement Git worktree branch creation or worktree directory management.
- Do NOT bypass AO daemon to manage worker OS processes directly.
