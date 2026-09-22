# 14. FAILURE RECOVERY PLAYBOOKS

> **Focus**: Deterministic Recovery from 15 Concrete System Failure Scenarios
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

# 1. Recovery Matrix & Owners

| Scenario ID | Failure Scenario | Impact | Recovery Owner | Playbook / Action |
|---|---|---|---|---|
| **REC-001** | Supervisor Daemon Restart | Active pair binding disconnected; memory state lost. | Supervisor Core | Reload state from local State Store; reconnect to existing Pair via `session_token`. |
| **REC-002** | AO Daemon Crash | Worker execution interrupted; ConPTY pipes closed. | AOAdapter | Attempt auto-reconnect to AO daemon; query session state; restore session if supported. |
| **REC-003** | Antigravity CLI Crash | Process exits with non-zero code before report generation. | Supervisor Core | Mark attempt `FAILED` (`failure_reason = PROCESS_CRASH`); query Git for partial changes; decide whether to retry or escalate to `HUMAN_REQUIRED`. |
| **REC-004** | Worker Execution Timeout | Worker runs longer than configured bounded execution timeout policy. | Supervisor Core | Issue `stopWorker` to AO; capture partial logs; mark attempt `FAILED` (`failure_reason = TIMEOUT`). |
| **REC-005** | Dirty Worktree Detected | Worktree contains uncommitted files or unsafe working tree state before or during dispatch. | Supervisor Core | Fail closed with zero mutating Git commands. **Case A (Pre-PrepareDispatch)**: reject dispatch; Task remains `READY`; zero `TaskAttempt` allocated; emit audit evidence `WORKTREE_DIRTY`; escalate for human/upstream resolution. **Case B (Post-PrepareDispatch)**: transition `DISPATCHED -> FAILED` with failure rationale in audit log; no attempt schema mutation. |
| **REC-006** | Git Merge Conflict | Worker changes conflict with target main branch. | Supervisor Core | Mark task `BLOCKED` (`blocker_reason = MERGE_CONFLICT`); escalate to `HUMAN_REQUIRED` per canonical workflow. |
| **REC-007** | Worker Report Absent | Worker terminates with exit code 0 but creates no report file within bounded fetch window. | Supervisor Core | Mark attempt `FAILED` (`failure_reason = REPORT_MISSING`). Diagnostic Git evidence may be collected for triage, but no promotion to `REPORT_READY`, `EVIDENCE_READY`, or `REVIEWING` occurs. |
| **REC-008** | Malformed Worker Report | Report fails validation against `worker-report.schema.json` or contains mismatched identities. | Supervisor Core | Mark attempt `FAILED` (`failure_reason = REPORT_INVALID` or `REPORT_IDENTITY_MISMATCH`); record raw output for debugging; normal ReviewBundle compilation is aborted. |
| **REC-009** | Missing Test Artifacts | Worker claims tests passed but test log files are missing. | Evidence Collector | Mark test claim as `UNVERIFIED`; highlight discrepancies in Review Bundle for ChatGPT audit. |
| **REC-010** | ChatGPT Disconnected | Web connection drops during active task execution. | Supervisor Core | Execution proceeds autonomously; state transitions to `REVIEWING` once evidence is collected; awaits Supervisor reconnection. |
| **REC-011** | Stale Pair Mismatch | ChatGPT attempts tool call with expired or mismatched token. | Supervisor Core | Reject request with `PAIR_MISMATCH_ERROR`; require re-binding via `bind_project`. |
| **REC-012** | Project Directory Missing | Local repository directory moved or deleted. | Supervisor Core | Mark project `UNAVAILABLE`; freeze Pair state; alert User. |
| **REC-013** | Upstream Incompatibility | Upstream AO changes API format unexpectedly. | AOAdapter | Halt dispatch; mark `UPSTREAM_INCOMPATIBLE`; prompt developer to run compatibility audit. |
| **REC-014** | Unexpected Machine Restart | Host OS reboots during execution. | Supervisor Startup | On startup, scan State Store for tasks in `DISPATCHED`/`RUNNING`; verify AO worktrees; resume or mark `FAILED`. |
| **REC-015** | Power Loss During Write | Local State Store file write interrupted. | State Store Engine | Selected State Store must provide crash-safe durable persistence and recovery semantics satisfying NFR-003. Concrete engine mechanism is decided by Q4 ADR. |

### 1.1 REC-005 Worktree Recovery Timing & State Model Invariants

The Supervisor maintains strict zero-mutation isolation over target Git worktrees and adheres to the canonical P02 StateStore/domain model:

1. **P02 Domain Ground Truth**:
   - `TaskAttempt` contains identity, contract, report path, timestamps, and raw report (`attempt_id`, `attempt_number`, `task_id`, `contract_id`, `expected_report_path`, `started_at`, `ended_at`, `worker_report_raw`). It possesses **no** `state` field and **no** `failure_reason` column in persistence.
   - Canonical `StateMachine` forbids transitions `READY -> BLOCKED` or `READY -> FAILED`. `READY` allows only `READY -> DISPATCHED` (via atomic `PrepareDispatch`) or `READY -> CANCELLED`.
   - Atomic `PrepareDispatch` simultaneously commits `READY -> DISPATCHED` and allocates the persistent `TaskAttempt`.

2. **Case A — Dirty Worktree Detected Prior to `PrepareDispatch`**:
   - Pre-dispatch inspection identifies dirty files, untracked artifacts, or uncommitted worktree state before `PrepareDispatch` is invoked.
   - **Behavior**: Dispatch is rejected immediately.
   - **State Invariant**: Task remains in `READY`; zero `TaskAttempt` is allocated; zero AO execution side-effect is triggered.
   - **Audit Record**: Log operational event with evidence label `WORKTREE_DIRTY`.
   - **Resolution**: Escalate to human developer or upstream orchestrator. Pre-dispatch validation may be retried after external clean.

3. **Case B — Unsafe Worktree Condition Discovered Post-`PrepareDispatch`**:
   - An unsafe condition is detected after atomic `PrepareDispatch` has completed (Task is `DISPATCHED`, `TaskAttempt` exists).
   - **Behavior**: Execute canonical transition `DISPATCHED -> FAILED`.
   - **State Invariant**: Task transitions to `FAILED`. Zero schema mutation: no `failure_reason` column is added to `TaskAttempt`.
   - **Audit Record**: Record failure cause `WORKTREE_DIRTY` within the append-only `AuditLogger` event payload.
   - **Resolution**: Escalate to human or failure recovery handler. Zero `git stash`, `git clean`, `git checkout`, or `git reset` is executed by the Supervisor.
