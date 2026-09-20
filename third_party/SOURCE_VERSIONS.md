# THIRD-PARTY SOURCE VERSIONS

> **Authority**: Dependency Pinning & Upstream Version Register  
> **Status**: Verified Documentation Baseline (Post-Remediation)

---

# SECTION 1: ACTIVE UPSTREAM DEPENDENCIES (RUNTIME SYSTEMS)

These external systems provide active runtime execution capabilities. In Phase 0, all references represent **Documentation Baselines**. Runtime proof is deferred strictly to Phase P01.

### 1. Untrivial Agent Orchestrator
- **Repository**: `Untrivial-ai/agent-orchestrator`
- **Role in Architecture**: Execution Control Plane (Process daemon, Windows ConPTY terminal, Git worktree isolation)
- **Pinned Documentation Version**: `v0.13.0`
- **Pinned Commit**: `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`
- **Commit Date**: 2026-03-05T08:52:13Z
- **License**: Apache-2.0
- **Documentation Contract Status**: `DOCUMENTED`
- **Local Installed Version**: `RUNTIME_UNTESTED` (Pending P01 proof)
- **Runtime Proof Status**: `RUNTIME_UNTESTED`
- **Compatibility Status**: `COMPATIBILITY_PENDING_P01`
- **Verified Public Interfaces**:
  - `GET /healthz` (Daemon health probe)
  - `GET /readyz` (Daemon readiness probe)
  - `POST /api/v1/projects` (Project registration)
  - `GET /api/v1/projects/{id}` (Project inspection)
  - `POST /api/v1/sessions` (Spawn session / worktree)
  - `GET /api/v1/sessions/{id}` (Session status inspection)
  - `POST /api/v1/sessions/{id}/send` (Send prompt / instruction)
  - `POST /api/v1/sessions/{id}/kill` (Terminate session process)
  - `POST /api/v1/sessions/{id}/restore` (Restore existing conversation session)

### 2. Official Antigravity CLI
- **Repository**: `google-antigravity/antigravity-cli`
- **Role in Architecture**: Primary Worker Interface (Autonomous coding agent inside worktrees)
- **Pinned Documentation Version**: `1.2.7`
- **Pinned Commit**: `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`
- **Commit Date**: 2026-03-12T18:34:01Z
- **License**: Google Proprietary / Developer Terms of Service
- **Documentation Contract Status**: `DOCUMENTED`
- **Local Installed Version**: `1.2.7` (Verified via `agy --version` on local host)
- **Runtime Proof Status**: `RUNTIME_UNTESTED` (Comprehensive headless contract testing pending P01)
- **Compatibility Status**: `COMPATIBILITY_PENDING_P01`
- **Verified CLI Flags (from official distribution)**:
  - `--print` / `-p`: Run single prompt non-interactively and print response
  - `--output-format`: Output format for print mode (`text`, `json`, `stream-json`)
  - `--input-format`: Input format for print mode (`text`, `stream-json`)
  - `--json-schema`: Enforce JSON schema on structured final output
  - `--add-dir`: Add directory to workspace root
  - `--dangerously-skip-permissions`: Auto-approve tool executions
  - `--conversation`: Resume previous conversation by ID
  - `--continue` / `-c`: Continue most recent conversation
  - `--mode`: Agent execution mode (`accept-edits`, `plan`)

---

# SECTION 2: PASSIVE DESIGN SOURCES (ARCHITECTURAL REFERENCES)

These repositories serve strictly as conceptual, pattern, and design references. **THEY ARE NOT ACTIVE RUNTIME DEPENDENCIES AND DO NOT AUTO-UPDATE.**

| Technology | Repository | Pinned Commit (Full 40-Char SHA) | Commit Date | License | Reason Selected & Role |
|---|---|---|---|---|---|
| **Proxide** | `tt-a1i/proxide` | `c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee` | 2026-06-20 | MIT | Readonly default security, path containment |
| **Mieruko Workbench** | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | `4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df` | 2026-09-19 | MIT | Workspace binding, slim tool surface philosophy |
| **AIWorkHub** | `shrec/AIWorkHub` | `19c8ce548c27316a06117c0638b4f853e00292b8` | 2026-09-20 | MIT | Evidence-first review, candidate claims vs. ground truth |
| **OpenAI Symphony** | `openai/symphony` | `be10a1b79df723d6d7612b5651c8522704dafb2e` | 2026-09-15 | Apache-2.0 | Workflow-as-policy, isolated run boundaries (`SPEC.md`) |
| **Codencer** | `lookmanrays/codencer` | `8d4908b1acf049cd97a9c4dcb752d1e44cbb8655` | 2026-08-06 | Apache-2.0 | Bridge-not-brain, state outside chat, execution vocabulary |
| **AWS CAO** | `awslabs/cli-agent-orchestrator` | `156cf1edcdb9de1ee01a2f17c2a4026e5f305e56` | 2026-09-20 | Apache-2.0 | Provider / runtime abstraction pattern |
| **Antigravity Link** | `cafeTechne/antigravity-link-extension` | `dae4483275acba8fff093b14bb25abe8e9495f94` | 2026-06-04 | MIT | CDP browser/GUI bridge (retained strictly as fallback) |
