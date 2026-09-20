# 15. OBSERVABILITY & AUDIT TRAIL SPECIFICATION

> **Focus**: Event Taxonomy, Structured Telemetry & Secret-Sanitized Audit Trail  
> **Status**: Approved Baseline

---

# 1. Canonical Event Taxonomy

All lifecycle occurrences within the Supervisor Control Plane are emitted as structured, chronological events:

| Event Name | Category | Payload Key Data | Description |
|---|---|---|---|
| `pair.created` | Pair | `pair_id, project_id, timestamp` | New engineering lane established. |
| `pair.bound` | Pair | `pair_id, session_token, supervisor_type` | ChatGPT session connected to Pair. |
| `task.created` | Task | `task_id, phase_id, contract_id` | New immutable task contract created. |
| `task.dispatched` | Task | `task_id, session_id, base_sha` | Task transmitted to Agent Orchestrator. |
| `task.started` | Task | `task_id, worker_pid, worktree_path` | Worker process active in worktree. |
| `task.reported` | Task | `task_id, attempt_id, claimed_status` | Worker completed execution and emitted report. |
| `evidence.collected` | Evidence | `task_id, head_sha, diff_hash, test_count` | Git and test verification completed. |
| `review.ready` | Review | `task_id, bundle_id, policy_findings_count` | Review Bundle compiled and available. |
| `review.started` | Review | `task_id, reviewer_id` | ChatGPT begins evaluation of Review Bundle. |
| `task.approved` | Review | `task_id, approver_rationale, merge_sha` | Task formally approved; changes merged. |
| `task.revision_required`| Review | `task_id, feedback_summary, fix_count` | Revision requested; re-entry into cycle. |
| `task.blocked` | Review | `task_id, blocker_reason, source_entity` | Execution halted due to blocker. |
| `task.failed` | Task | `task_id, error_code, stack_snippet` | Task failed due to timeout, crash, or error. |
| `worker.started` | Worker | `session_id, harness_type, pid` | Low-level worker process spawned. |
| `worker.stopped` | Worker | `session_id, exit_code, duration_ms` | Low-level worker process halted. |
| `worker.crashed` | Worker | `session_id, exit_signal, last_log` | Low-level worker process exited abnormally. |
| `upstream.compatibility_changed` | Upstream | `upstream_name, old_version, new_version` | Upstream version or compatibility record updated. |

---

# 2. Sanitized Audit Trail Principles
1. **Append-Only Immutability**: Audit log records cannot be edited or deleted.
2. **Deterministic Scrubbing**: Outbound logs pass through pattern filters to strip API keys, Bearer tokens, and secrets.
3. **Structured Storage**: Events are written as JSON lines (`audit.jsonl`) for local querying.
