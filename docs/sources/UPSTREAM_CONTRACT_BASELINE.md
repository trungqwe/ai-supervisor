# UPSTREAM CONTRACT BASELINE

> **Focus**: Evidence-Driven Mapping of Upstream Public Interfaces
> **Status**: Literal Evidence Updated (Track P01-A AO Runtime PASS; Track P01-B Agy PARTIALLY_RUNTIME_TESTED)
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
| **Non-Interactive Prompt Exec**| Agy `1.2.7` | `agy -p "<prompt>"` | `agy --help` / `README.md` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: marker echo, exit 0, correct text output. Evidence: P01-B1. |
| **JSON Output Envelope** | Agy `1.2.7` | `agy --output-format json` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: `{conversation_id, status, response, ...}` envelope. Evidence: P01-B2. |
| **Stream-JSON Output** | Agy `1.2.7` | `agy --output-format stream-json` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: NDJSON `init`/`step_update`/`result` event sequence. Evidence: P01-B3. |
| **Stream JSON Input Protocol** | Agy `1.2.7` | `agy --input-format stream-json --output-format stream-json` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: stdin NDJSON accepted, consistent `conversation_id` maintained across turns. Evidence: P01-B4. Note: `QUOTA_ERROR_OBSERVED_ON_FINAL_RESULT` on Turn 2 retry; marker still recovered. |
| **Structured Output Schema** | Agy `1.2.7` | `agy --json-schema <schema>` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: `structured_output` field populated per schema. Evidence: P01-B5. |
| **Schema + File Side Effect** | Agy `1.2.7` | `agy --json-schema ... --dangerously-skip-permissions` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: file created on disk AND structured output emitted in same turn. Evidence: P01-B6. |
| **Auto-Approve Tool Exec** | Agy `1.2.7` | `agy --dangerously-skip-permissions` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: file write without permission prompt. Evidence: P01-B7. |
| **Workspace Dir Binding** | Agy `1.2.7` | `agy --add-dir <path>` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: out-of-band marker in secondary dir read successfully. Evidence: P01-B8. |
| **Conversation ID Resume** | Agy `1.2.7` | `agy --conversation <id>` | `agy --help` | `DOCUMENTED` | `RUNTIME_NOT_EVALUATED` | ENV-P01B-001: `AGY_INDIVIDUAL_QUOTA_EXHAUSTED` blocked Turn 2. Flag accepted; full cross-process resume not proven. Evidence: P01-B9. |
| **Workspace Continue** | Agy `1.2.7` | `agy --continue` / `agy -c` | `agy --help` | `DOCUMENTED` | `RUNTIME_NOT_EVALUATED` | Not yet tested. P01-B10 deferred pending quota recovery. |

---

# 2. AO ↔ Agy Integration Breakdown & Gap Analysis

### A. Official Antigravity CLI Capability
Official Agy `1.2.7` independently supports (partially runtime-tested — see P01-B dossier):
- Headless non-interactive execution (`-p "<prompt>"`, `--output-format json|stream-json`). **PROVEN.**
- Structured completion schema validation (`--json-schema <schema>`). **PROVEN.**
- Persistent streaming input/output protocol (`--input-format stream-json --output-format stream-json` via stdin). **PROVEN** (same-process multi-turn context).
- Secondary directory binding (`--add-dir`). **PROVEN.**
- Permission bypass (`--dangerously-skip-permissions`). **PROVEN.**
- Session continuation (`--conversation <id>`, `--continue`). **NOT EVALUATED** (ENV-P01B-001 quota gate).

### B. AO Agy Adapter Capability (at Pinned Baseline v0.13.0)
Inspection of `backend/internal/adapters/agent/agy/agy.go` confirms that AO invokes Agy using:
```text
agy --add-dir <path> --dangerously-skip-permissions --prompt-interactive <prompt> --conversation <session-id>
```
AO wraps Agy in an interactive harness rather than passing `--json-schema` or using `--print`.

### C. The Integration Gap (`P01_PROOF_REQUIRED`)
The crucial unanswered architectural question is:
> *Can the existing AO Agy integration satisfy our WorkerReport contract without custom integration code?*

Because AO invokes Agy interactively, it is currently unproven whether AO's session event stream or completion detection will cleanly expose the structured `WorkerReport`. This question is isolated to **Phase P01 Track P01-C** (currently HELD).