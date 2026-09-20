# UPSTREAM CONTRACT BASELINE

> **Focus**: Evidence-Driven Mapping of Upstream Public Interfaces  
> **Status**: Verified Documentation Baseline (Post-Remediation)

---

# 1. Evidence-Driven Upstream Interface Mapping

| Domain Operation | Pinned Upstream | Documented Interface | Exact Evidence Source in Upstream Repo | Documentation Status | Runtime Proof Status | Notes / Gap |
|---|---|---|---|---|---|---|
| **Daemon Health Probe** | AO `v0.13.0` | `GET /healthz` | `backend/internal/httpd/router.go` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Returns daemon probe JSON payload with PID and cwd. |
| **Daemon Readiness Probe** | AO `v0.13.0` | `GET /readyz` | `backend/internal/httpd/router.go` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Shares daemon probe payload; asserts dependency readiness. |
| **Project Registration** | AO `v0.13.0` | `POST /api/v1/projects` | `backend/internal/httpd/controllers/projects.go` (`add`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Mounts project on router; calls `projectsvc.Manager.Add`. |
| **Project Inspection** | AO `v0.13.0` | `GET /api/v1/projects/{id}` | `backend/internal/httpd/controllers/projects.go` (`get`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Retrieves registered project details. |
| **Session / Worktree Spawn**| AO `v0.13.0` | `POST /api/v1/sessions` | `backend/internal/httpd/controllers/sessions.go` (`spawn`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Creates isolated workspace and spawns worker session. |
| **Session Status Inspection**| AO `v0.13.0` | `GET /api/v1/sessions/{sessionId}` | `backend/internal/httpd/controllers/sessions.go` (`get`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Returns session state, workspace paths, and active process status. |
| **Send Work to Worker** | AO `v0.13.0` | `POST /api/v1/sessions/{sessionId}/send` | `backend/internal/httpd/controllers/sessions.go` (`send`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Sends instruction payload into active session. |
| **Session Process Termination**| AO `v0.13.0`| `POST /api/v1/sessions/{sessionId}/kill` | `backend/internal/httpd/controllers/sessions.go` (`kill`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Terminates active worker process. |
| **Session Conversation Restore**| AO `v0.13.0`| `POST /api/v1/sessions/{sessionId}/restore` | `backend/internal/httpd/controllers/sessions.go` (`restore`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Restores existing agent conversation session. |
| **Workspace Event Streaming**| AO `v0.13.0` | `GET /api/v1/sessions/{sessionId}/workspace/events` | `backend/internal/httpd/controllers/sessions.go` (`streamWorkspaceChanges`) | `DOCUMENTED` | `RUNTIME_UNTESTED` | Streams workspace filesystem change events. |
| **Non-Interactive Prompt Exec**| Agy `1.2.7` | `agy --print -p "<prompt>"` | `agy --help` / `README.md` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Runs single prompt non-interactively; prints response. |
| **Structured Output Schema** | Agy `1.2.7` | `agy --json-schema <schema>` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Enforces JSON schema on output (applicable to final result). |
| **Stream JSON Protocol** | Agy `1.2.7` | `agy --output-format stream-json --input-format stream-json` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Reads NDJSON line-by-line from stdin; writes stream-json. |
| **Workspace Dir Binding** | Agy `1.2.7` | `agy --add-dir <path>` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Adds directory to session workspace root. |
| **Auto-Approve Tool Exec** | Agy `1.2.7` | `agy --dangerously-skip-permissions` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Auto-approves tool permission prompts for autonomous execution. |
| **Conversation Continuation**| Agy `1.2.7` | `agy --conversation <id>` / `--continue` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Resumes existing conversation thread by ID. |

---

# 2. AO ↔ Agy Integration Breakdown & Gap Analysis

### A. Official Antigravity CLI Capability
Official Agy `1.2.7` independently supports:
- Headless print mode (`--print`, `--output-format json|stream-json`);
- Structured schema enforcement (`--json-schema`);
- Unattended tool execution (`--dangerously-skip-permissions`);
- Multi-directory workspace binding (`--add-dir`).

### B. AO Agy Adapter Capability (As Implemented in AO `v0.13.0`)
Empirical inspection of `backend/internal/adapters/agent/agy/agy.go` in AO `v0.13.0` reveals:
- **Launch Command**:
  ```bash
  agy --add-dir <WorkspacePath> [--dangerously-skip-permissions] [--model <Model>] [--prompt-interactive <Prompt>]
  ```
- **Restore Command**:
  ```bash
  agy --add-dir <WorkspacePath> [--dangerously-skip-permissions] [--model <Model>] --conversation <agentSessionId>
  ```
- AO invokes Agy using interactive flags (`--prompt-interactive`) rather than non-interactive headless print mode (`--print`, `--input-format stream-json`).

### C. Integration Gap: Structured WorkerReport Delivery
- **Gap Statement**: AO's existing adapter does not natively pass `--json-schema docs/schemas/worker-report.schema.json` or extract an explicit `WorkerReport` object upon completion.
- **Status**: `P01_PROOF_REQUIRED`.
- **Remediation Rule**: In Phase 0, we acknowledge this gap as an empirical finding. Phase P01 Track P01-C will evaluate whether AO's interactive session output stream can be normalized by the Supervisor, or whether an AOAdapter enhancement / direct harness invocation is required.
