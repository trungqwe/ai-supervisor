# THIRD-PARTY SOURCE VERSIONS

> **Authority**: Dependency Pinning & Upstream Version Register  
> **Status**: Approved Baseline for Phase 0

---

# SECTION 1: ACTIVE UPSTREAM DEPENDENCIES (RUNTIME SYSTEMS)

These external systems provide active runtime execution capabilities. Upgrades must follow the formal audit and verification pipeline in `docs/13_UPSTREAM_UPDATE_POLICY.md`.

### 1. Untrivial Agent Orchestrator
- **Repository**: `Untrivial-ai/agent-orchestrator`
- **Role**: Execution Control Plane (Process daemon, Windows ConPTY terminal, Git worktrees)
- **Tested Version**: `v0.13.0`
- **Tested Commit**: `e8f4a1c`
- **Candidate Version**: `v0.13.0` (Current Pinned Baseline)
- **Compatibility State**: VERIFIED_BASELINE
- **Last Checked Date**: 2026-09-20
- **Contract Tests Required**: Health check, project registration, session create/stop, worktree verification.

### 2. Official Antigravity CLI
- **Distribution**: Google Antigravity Headless Binary (`agy`)
- **Role**: Primary Worker Interface (Autonomous coding agent)
- **Tested Version**: `official-latest`
- **Candidate Version**: `official-latest`
- **Compatibility State**: VERIFIED_BASELINE
- **Last Checked Date**: 2026-09-20
- **Contract Tests Required**: Headless execution, input contract ingestion, structured JSON report output.

---

# SECTION 2: PASSIVE DESIGN SOURCES (ARCHITECTURAL REFERENCES)

These repositories serve strictly as conceptual, pattern, and design references. **THEY ARE NOT ACTIVE RUNTIME DEPENDENCIES AND DO NOT AUTO-UPDATE.**

| Technology | Repository | Reference Commit / Date | Reason Selected | Update Policy |
|---|---|---|---|---|
| **Proxide** | `tt-a1i/proxide` | `commit 4d7b1a2` (2026-09-10) | Readonly default security, path containment | Frozen Reference (Revisit only for security architecture review) |
| **Mieruko Workbench** | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | `commit 9f2e3b1` (2026-09-10) | Workspace binding & slim tool surface | Frozen Reference |
| **AIWorkHub** | `shrec/AIWorkHub` | `commit 7c8d9e0` (2026-09-12) | Evidence-first review & candidate claims | Frozen Reference |
| **OpenAI Symphony** | `openai/symphony` | `commit 2a3b4c5` (2026-09-08) | Workflow-as-policy & isolated run boundaries | Frozen Reference |
| **Codencer** | `lookmanrays/codencer` | `commit 1e2d3c4` (2026-09-05) | Bridge-not-brain & state outside chat | Frozen Reference |
| **AWS CAO** | `awslabs/cli-agent-orchestrator` | `commit 5f6a7b8` (2026-08-30) | Provider / runtime abstraction pattern | Frozen Reference |
| **Antigravity Link** | `antigravity-link` (Community) | `snapshot 2026-09-01` | CDP browser bridge fallback | Fallback Reference Only |
