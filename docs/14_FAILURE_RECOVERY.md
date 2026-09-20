# 14. FAILURE RECOVERY PLAYBOOKS

> **Focus**: Deterministic Recovery from 15 Concrete System Failure Scenarios  
> **Status**: Approved Baseline

---

# 1. Recovery Matrix & Owners

| Scenario ID | Failure Scenario | Impact | Recovery Owner | Playbook / Action |
|---|---|---|---|---|
| **REC-001** | Supervisor Daemon Restart | Active pair binding disconnected; memory state lost. | Supervisor Core | Reload state from local State Store; reconnect to existing Pair via `session_token`. |
| **REC-002** | AO Daemon Crash | Worker execution interrupted; ConPTY pipes closed. | AOAdapter | Attempt auto-reconnect to AO daemon; query session state; restore session if supported. |
| **REC-003** | Antigravity CLI Crash | Process exits with non-zero code before report generation. | Supervisor Core | Mark attempt `FAILED`; query Git for partial changes; decide whether to retry or escalate to `HUMAN_REQUIRED`. |
| **REC-004** | Worker Execution Timeout | Worker runs longer than contract `max_execution_time`. | Supervisor Core | Issue `stopWorker` to AO; capture partial logs; mark attempt `FAILED`. |
| **REC-005** | Dirty Worktree Detected | Worktree contains uncommitted files before task dispatch. | AOAdapter | Stash or clean untracked artifacts before initializing new task attempt. |
| **REC-006** | Git Merge Conflict | Worker changes conflict with target main branch. | Supervisor Core | Mark task `BLOCKED`; formulate revision contract instructing worker to rebase and resolve. |
| **REC-007** | Worker Report Absent | Worker terminates with exit code 0 but creates no report file. | Supervisor Core | Mark `EVIDENCE_READY_WITH_WARNING`; synthesize report directly from Git diff; flag missing worker claims. |
| **REC-008** | Malformed Worker Report | Report fails validation against `worker-report.schema.json`. | Supervisor Core | Mark `REPORT_INVALID`; record raw output; generate review bundle with policy warning. |
| **REC-009** | Missing Test Artifacts | Worker claims tests passed but test log files are missing. | Evidence Collector | Mark test claim as `UNVERIFIED`; highlight discrepancies in Review Bundle for ChatGPT audit. |
| **REC-010** | ChatGPT Disconnected | Web connection drops during active task execution. | Supervisor Core | Execution proceeds autonomously; state transitions to `REVIEWING`; awaits Supervisor reconnection. |
| **REC-011** | Stale Pair Mismatch | ChatGPT attempts tool call with expired or mismatched token. | Supervisor Core | Reject request with `PAIR_MISMATCH_ERROR`; require re-binding via `bind_project`. |
| **REC-012** | Project Directory Missing | Local repository directory moved or deleted. | Supervisor Core | Mark project `UNAVAILABLE`; freeze Pair state; alert User. |
| **REC-013** | Upstream Incompatibility | Upstream AO changes API format unexpectedly. | AOAdapter | Halt dispatch; mark `UPSTREAM_INCOMPATIBLE`; prompt developer to run compatibility audit. |
| **REC-014** | Unexpected Machine Restart | Host OS reboots during execution. | Supervisor Startup | On startup, scan State Store for tasks in `DISPATCHED`/`RUNNING`; verify AO worktrees; resume or mark `FAILED`. |
| **REC-015** | Power Loss During Write | Local State Store file write interrupted. | State Store Engine | Use atomic write-and-rename (`tmp` → `.db`) with write-ahead logging to prevent corruption. |
