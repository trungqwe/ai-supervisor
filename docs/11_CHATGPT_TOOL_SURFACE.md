# 11. CHATGPT TOOL SURFACE SPECIFICATION

> **Focus**: Minimal High-Level Tools for Supervisor Reasoning & Zero Direct Mutation  
> **Status**: Verified Documentation Baseline (Post-Remediation)

---

# 1. Design Philosophy

The Supervisor Tool Surface exposes **12 high-level, domain-specific tools**. It intentionally rejects generic primitives (arbitrary bash, raw file editing, or GUI automation) and avoids leaking low-level OS details (like worker PIDs).

---

# 2. Canonical Tool Catalog

| Tool Name | Type | Input Parameters | Output Structure | Description & Governance |
|---|---|---|---|---|
| `bind_project` | Control | `project_id: string` | `pair_id, status` | Connects ChatGPT session to a registered project Pair. |
| `get_project_state` | Read | `pair_id: string` | Project status, current phase, active task | Returns high-level state of the engineering lane. |
| `get_current_task` | Read | `pair_id: string` | TaskContract & status | Retrieves the active task and its execution progress. |
| `get_worker_status` | Read | `pair_id: string` | `activity_state, session_state, last_activity, blocked_reason, runtime_health` | Returns high-level worker status (no low-level OS PIDs). |
| `search_project` | Read | `query: string, filter: string` | List of matching paths/symbols | Fast structural search across project metadata. |
| `read_project_file` | Read | `path: string, range: [start, end]` | Text content | Read-only inspection of a project file (strictly within root). |
| `get_review_bundle` | Read | `task_id: string` | Full `ReviewBundle` | Ingests synthesized audit packet for task evaluation. |
| `dispatch_task` | Control | `contract: TaskContract` | `task_id, dispatch_status` | Validates and dispatches an immutable task to worker. |
| `approve_task` | Control | `task_id: string, rationale: string` | `new_state: APPROVED` | Formally approves task. **NO automatic git merge or push**. |
| `request_revision` | Control | `task_id: string, feedback: string, required_fixes: string[]` | `new_state: REVISION_REQUIRED` | Re-invokes worker with specific revision goals. |
| `block_task` | Control | `task_id: string, blocker_reason: string` | `new_state: BLOCKED` | Marks task blocked; escalates to human developer. |
| `get_phase_status` | Read | `phase_id: string` | Deliverables, completed tasks, remaining | Evaluates roadmap progress and phase readiness. |

---

# 3. Explicitly Rejected Tools

The following capabilities are **STRICTLY PROHIBITED** from the Supervisor Tool Surface:
- `execute_shell_command` / `run_terminal`: Violates SEC-003 and execution plane separation.
- `write_file` / `patch_file`: Supervisor never edits application code directly (ADR-005).
- `git_commit` / `git_push` / `git_merge`: Source mutations and branch merges do not occur via Supervisor approval.
- `mouse_click` / `send_keys`: OS GUI automation is strictly forbidden (NFR-002, ADR-004).
