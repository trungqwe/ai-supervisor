# 15. OBSERVABILITY & AUDIT TRAIL SPECIFICATION

> **Focus**: Event Taxonomy, Structured Telemetry & Secret-Sanitized Audit Trail
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

# 1. Canonical Event Taxonomy

All lifecycle occurrences within the Supervisor Control Plane are emitted as structured, chronological events carrying exact immutable contract and execution attempt correlation:

| Event Name | Category | Payload Key Data | Description |
|---|---|---|---|
| `pair.created` | Pair | `pair_id, project_id, timestamp` | New engineering lane established. |
| `pair.bound` | Pair | `pair_id, session_token, supervisor_type, timestamp` | ChatGPT session connected to Pair. |
| `task.created` | Task | `task_id, phase_id, contract_id, revision_number, timestamp` | New immutable task contract revision created. |
| `task.dispatched` | Task | `task_id, contract_id, attempt_id, session_id, base_sha, timestamp` | Task attempt transmitted to Agent Orchestrator. |
| `task.started` | Task | `task_id, contract_id, attempt_id, worker_pid, worktree_path, timestamp` | Worker process active in worktree. |
| `task.reported` | Task | `task_id, contract_id, attempt_id, claimed_status, timestamp` | Worker completed turn and emitted report artifact. |
| `evidence.collected` | Evidence | `task_id, contract_id, attempt_id, head_sha, diff_hash, test_count, timestamp` | Independent Git diff and test runner verification completed. |
| `review.ready` | Review | `task_id, contract_id, attempt_id, bundle_id, policy_findings_count, timestamp` | Attempt-scoped Review Bundle compiled and available for audit. |
| `review.started` | Review | `task_id, contract_id, attempt_id, reviewer_id, timestamp` | ChatGPT begins evaluation of Review Bundle. |
| `task.approved` | Review | `task_id, contract_id, attempt_id, decision_id, approver_rationale, timestamp` | Task formally approved (review decision recorded; NO automatic merge or push). |
| `task.revision_required`| Review | `task_id, contract_id, attempt_id, decision_id, feedback_summary, fix_count, timestamp` | Revision requested; re-entry into revision loop with new contract revision. |
| `task.blocked` | Review | `task_id, contract_id, attempt_id, blocker_reason, source_entity, timestamp` | Execution halted due to blocker; escalated to HUMAN_REQUIRED. |
| `task.failed` | Task | `task_id, contract_id, attempt_id, error_code, failure_reason, stack_snippet, timestamp` | Task failed due to timeout, crash, or report defect (`failure_reason`). |
| `worker.started` | Worker | `session_id, harness_type, pid, timestamp` | Low-level worker process spawned. |
| `worker.stopped` | Worker | `session_id, exit_code, duration_ms, timestamp` | Low-level worker process halted. |
| `worker.crashed` | Worker | `session_id, exit_signal, last_log, timestamp` | Low-level worker process exited abnormally. |
| `upstream.compatibility_changed` | Upstream | `upstream_name, old_version, new_version, timestamp` | Upstream version or compatibility record updated. |

---

# 2. Sanitized Audit Trail Principles
1. **Append-Only Immutability**: Audit log records cannot be edited or deleted.
2. **Deterministic Scrubbing**: Outbound logs pass through pattern filters to strip API keys, Bearer tokens, and secrets.
3. **Structured Storage**: Events are written as JSON lines (`audit.jsonl`) for local querying.
4. **Full Lineage Reconstructability**: The sequence of events allows complete offline reconstruction of:
   ```
   task_id -> contract_id (revision_number) -> attempt_id -> worker claims -> evidence -> review bundle -> review decision
   ```