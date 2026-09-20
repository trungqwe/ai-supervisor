# SOURCE DOSSIER: 01 — UNTRIVIAL AGENT ORCHESTRATOR

## 1. Metadata
- **Repository**: `Untrivial-ai/agent-orchestrator`
- **Role in Architecture**: Active Runtime Dependency (Execution Control Plane)
- **Inspected Release / Tag**: `v0.13.0` (Windows native release)
- **Inspected Commit**: `e8f4a1c` (Release tag commit)
- **Inspection Date**: 2026-09-20
- **License**: Apache-2.0
- **Primary Language**: Go

---

# 2. Capabilities Evaluated & Adopted
- **Daemon Lifecycle**: Long-running background daemon managing sessions and processes.
- **Windows ConPTY**: Native pseudoterminal allocation on Windows avoiding terminal hang bugs.
- **Git Worktree Manager**: Creation, isolation, and teardown of dedicated worktrees per agent run.
- **Agent Harnesses**: Unified integration layer supporting 25+ coding agent CLIs.

---

# 3. Capabilities Explicitly Rejected
- **Internal SQLite DB Access**: Direct reading or writing of AO's internal database is forbidden.
- **Automatic Multi-Agent Debate**: AO's multi-agent routing features are rejected for V1.
- **Cloud Fleet Orchestration**: Cloud remote agent spawning is excluded for our local architecture.

---

# 4. Integration Strategy & Upstream Boundary
- **Strategy**: `UPSTREAM` (Used directly as binary/service without in-tree vendoring).
- **Boundary**: `AOAdapter` translates domain commands into public REST calls.
- **What We Must NOT Rebuild**: Daemon process manager, Windows ConPTY terminal handler, Git worktree isolation scripts.

---

# 5. SOURCE EVIDENCE

### Evidence Item 1.1: Windows ConPTY Support
- **Claim**: AO natively manages Windows pseudoterminal processes without third-party wrapper dependencies.
- **Repository**: `Untrivial-ai/agent-orchestrator`
- **Reference**: `v0.13.0`
- **Source File / Module**: `pkg/terminal/conpty_windows.go`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 1.2: Isolated Worktree Management
- **Claim**: AO allocates separate Git worktrees for agent tasks, preventing working directory contamination.
- **Repository**: `Untrivial-ai/agent-orchestrator`
- **Reference**: `v0.13.0`
- **Source File / Module**: `pkg/worktree/manager.go` (`CreateWorktree`, `RemoveWorktree`)
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 1.3: Public REST Daemon API
- **Claim**: AO exposes HTTP daemon endpoints for session creation, task dispatch, and status polling.
- **Repository**: `Untrivial-ai/agent-orchestrator`
- **Reference**: `v0.13.0`
- **Source File / Module**: `cmd/daemon/server.go`, `docs/api_reference.md`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
