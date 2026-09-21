# UPSTREAM CONTRACT BASELINE

> **Focus**: Evidence-Driven Mapping of Upstream Public Interfaces
> **Status**: Literal Evidence Corrected (Track P01-A AO Runtime Tested PASS)
> **Date Convention**: All commit timestamps are explicitly recorded in ISO-8601 UTC format (`YYYY-MM-DDTHH:MM:SSZ`).
> **License Convention**: Repository SPDX License is explicitly distinguished from Product / Usage Terms.

---

# 1. Evidence-Driven Upstream Interface Mapping

| Domain Operation | Pinned Upstream | Documented Interface | Exact Evidence Source in Upstream Repo | Documentation Status | Runtime Proof Status | Notes / Gap |
|---|---|---|---|---|---|---|
| **Daemon Health Probe** | AO `v0.13.0` | `GET /healthz` | `backend/internal/httpd/router.go` (`NewRouterWithControl`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Returns daemon probe JSON payload with PID and cwd. |
| **Daemon Readiness Probe** | AO `v0.13.0` | `GET /readyz` | `backend/internal/httpd/router.go` (`NewRouterWithControl`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Shares daemon probe payload; asserts dependency readiness. |
| **Project Registration** | AO `v0.13.0` | `POST /api/v1/projects` | `backend/internal/httpd/controllers/projects.go` (`add`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Mounts project on router; calls `projectsvc.Manager.Add`. |
| **Project Inspection** | AO `v0.13.0` | `GET /api/v1/projects/{id}` | `backend/internal/httpd/controllers/projects.go` (`get`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Retrieves registered project details. |
| **Session / Worktree Spawn**| AO `v0.13.0` | `POST /api/v1/sessions` | `backend/internal/httpd/controllers/sessions.go` (`spawn`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Creates isolated workspace and spawns worker session. |
| **Session Status Inspection**| AO `v0.13.0` | `GET /api/v1/sessions/{sessionId}` | `backend/internal/httpd/controllers/sessions.go` (`get`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Returns session state, workspace paths, and active process status. |
| **Send Work to Worker** | AO `v0.13.0` | `POST /api/v1/sessions/{sessionId}/send` | `backend/internal/httpd/controllers/sessions.go` (`send`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Sends instruction payload into active session. |
| **Session Process Termination**| AO `v0.13.0`| `POST /api/v1/sessions/{sessionId}/kill` | `backend/internal/httpd/controllers/sessions.go` (`kill`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Terminates active worker process. |
| **Session Conversation Restore**| AO `v0.13.0`| `POST /api/v1/sessions/{sessionId}/restore` | `backend/internal/httpd/controllers/sessions.go` (`restore`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Restores existing agent conversation session. |
| **Workspace Event Streaming**| AO `v0.13.0` | `GET /api/v1/sessions/{sessionId}/workspace/events` | `backend/internal/httpd/controllers/sessions.go` (`streamWorkspaceChanges`) | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Streams workspace filesystem change events. |
| **Non-Interactive Prompt Exec**| Agy `1.2.7` | `agy -p "<prompt>"` | `agy --help` / `README.md` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Runs single prompt non-interactively; prints response. `-p`, `--print`, and `--prompt` are aliases. |
| **Structured Output Schema** | Agy `1.2.7` | `agy -p "<prompt>" --json-schema <schema>` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Enforces JSON schema on output (applicable to final result turn). |
| **Stream JSON Protocol** | Agy `1.2.7` | `agy --output-format stream-json --input-format stream-json` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Reads NDJSON line-by-line from stdin (not via `-p`); writes stream-json. |
| **Workspace Dir Binding** | Agy `1.2.7` | `agy --add-dir <path>` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Adds secondary directory to session workspace root. |
| **Auto-Approve Tool Exec** | Agy `1.2.7` | `agy --dangerously-skip-permissions` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Auto-approves tool permission prompts for autonomous execution. |
| **Conversation Continuation**| Agy `1.2.7` | `agy --conversation <id>` / `--continue` | `agy --help` | `DOCUMENTED` | `RUNTIME_UNTESTED` | Resumes existing conversation thread by ID. |

---

# 2. AO ↔ Agy Integration Breakdown & Gap Analysis

### A. Official Antigravity CLI Capability
Official Agy `1.2.7` independently supports:
- Headless non-interactive execution (`-p "<prompt>"`, `--output-format json|stream-json`).
- Structured completion schema validation (`--json-schema <schema>`).
- Persistent streaming input/output protocol (`--input-format stream-json --output-format stream-json` via stdin).
- Session continuation (`--conversation <id>`, `--continue`).

### B. AO Agy Adapter Capability (at Pinned Baseline v0.13.0)
Inspection of `backend/internal/adapters/agent/agy/agy.go` confirms that AO invokes Agy using:
```text
agy --add-dir <path> --dangerously-skip-permissions --prompt-interactive <prompt> --conversation <session-id>
```
AO wraps Agy in an interactive harness rather than passing `--json-schema` or using `--print`.

### C. The Integration Gap (`P01_PROOF_REQUIRED`)
The crucial unanswered architectural question is:
> *Can the existing AO Agy integration satisfy our WorkerReport contract without custom integration code?*

Because AO invokes Agy interactively, it is currently unproven whether AO's session event stream or completion detection will cleanly expose the structured `WorkerReport`. This question is isolated to **Phase P01 Track P01-C**. In Phase 0, we do not invent a fictional adapter solution; we establish the boundary and require empirical verification.
