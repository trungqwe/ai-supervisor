# UPSTREAM CONTRACT BASELINE

> **Focus**: Evidence-Driven Mapping of Upstream Public Interfaces
> **Status**: Literal Evidence Updated (Track P01-A AO Runtime PASS; Track P01-B Agy RUNTIME_TESTED_PASS; Track P01-C AO ↔ Agy Integration GAP_RESOLVED_BY_ADR_PENDING_PHASE_P01_FINAL_AUDIT with ADR-011)
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
| **Conversation ID Resume** | Agy `1.2.7` | `agy --conversation <id>` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: cross-process conversation resumed via `--conversation <id>`, hidden marker recovered exactly. Evidence: P01-B9 (`p01b9_conversation_id_resume_proven.json`). |
| **Workspace Continue** | Agy `1.2.7` | `agy --continue` / `agy -c` | `agy --help` | `DOCUMENTED` | `RUNTIME_TESTED_PASS` | Proven: most recent conversation in workspace continued via `--continue`, hidden marker recovered exactly. Evidence: P01-B10 (`p01b10_continue_proven.json`). |
| **Failure Contract & Validation** | Agy `1.2.7` | Multiple flags | `agy --help` / P01-B negative suite | `DOCUMENTED` | `PASS_WITH_CALLER_VALIDATION_CONSTRAINT` | Characterized: missing schema path rejects locally (exit 1); invalid output-format and nonexistent add-dir proceed silently (exit 0); malformed schema errors remotely (exit 3). Constraint: `CALLER_VALIDATION_REQUIRED`. |

---

# 2. AO ↔ Agy Integration Breakdown & Gap Analysis

### A. Official Antigravity CLI Capability
Official Agy `1.2.7` independently supports (fully P01-B runtime-tested on the target Windows host — see P01-B dossier):
- Headless non-interactive execution (`-p "<prompt>"`, `--output-format json|stream-json`). **PROVEN.**
- Structured completion schema validation (`--json-schema <schema>`). **PROVEN.**
- Persistent streaming input/output protocol (`--input-format stream-json --output-format stream-json` via stdin). **PROVEN** (same-process multi-turn context).
- Secondary directory binding (`--add-dir`). **PROVEN.**
- Permission bypass (`--dangerously-skip-permissions`). **PROVEN.**
- Session continuation (`--conversation <id>`, `--continue`). **PROVEN** (P01-B9 and P01-B10).
- Caller input validation: Characterized under P01-B safe negative suite; requires caller pre-validation (`CALLER_VALIDATION_REQUIRED`).

### B. AO Agy Adapter Capability (at Pinned Baseline v0.13.0)
Inspection of `backend/internal/adapters/agent/agy/agy.go` confirms that AO invokes Agy using:
```text
agy --add-dir <path> --dangerously-skip-permissions --prompt-interactive <prompt> --conversation <session-id>
```
AO wraps Agy in an interactive harness rather than passing `--json-schema` or using `--print`.

### C. The Integration Gap & Empirical Resolution (Track P01-C: `GAP_RESOLVED_BY_ADR` via ADR-011)
Track P01-C evaluated the empirical integration between pinned AO `v0.13.0` and pinned Agy `1.2.7`:
> *Can the existing AO Agy integration satisfy our WorkerReport contract without custom integration code?*

**Empirical Resolution:**
1. **Interactive Invocation**: Pinned AO launches Agy in interactive mode via `agy --add-dir <wt> --dangerously-skip-permissions --prompt-interactive "<prompt>"`. It does not pass `--json-schema` or `--output-format`.
2. **Turn Completion Signal**: AO detects turn completion via the Agy `Stop` hook (`.agents/hooks.json`), transitioning session activity from `active` (`working`) to `idle` (`idle`) while the ConPTY process remains alive (`PROCESS_RUNNING` while `WORKER_TURN_COMPLETE`).
3. **Result Exposure**: AO's public session API (`GET /api/v1/sessions/{id}`) does **NOT** natively expose assistant text responses or WorkerReport data (`NATIVE_RESULT_SURFACE = NOT_EXPOSED`). Terminal streams (`/mux`) provide raw ANSI byte frames (`RAW_INTERACTIVE`).
4. **WorkerReport via Workspace File API**: The integration loop is empirically proven by having the worker output `.supervisor/worker-report.json`, which the Supervisor retrieves via AO's public workspace file API (`GET /api/v1/sessions/{id}/workspace/file?path=.supervisor/worker-report.json`).
5. **Context Retention across Restore**: Native conversation UUID is captured and passed to `--conversation <id>` on restore. Context retention of memory-only tokens was proven across restore cycles without disk persistence.
6. **No Upstream Patches**: Neither AO nor Agy requires code modifications (`AO_UPSTREAM_PATCH_REQUIRED = NO`, `AGY_UPSTREAM_PATCH_REQUIRED = NO`).
7. **Architectural Gap Accommodation**: Formally resolved by **ADR-011** (`docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`). Canonical report delivery uses attempt-scoped paths (`.supervisor/reports/<task_id>/<attempt_id>.json`) over AO's public workspace file API under zero-trust validation rules without upstream code modifications (`GAP_RESOLVED_BY_ADR_PENDING_PHASE_P01_FINAL_AUDIT`).
