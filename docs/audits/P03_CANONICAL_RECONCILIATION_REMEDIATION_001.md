# P03 Canonical Reconciliation Remediation 001

## Repository Identity
* **Repository**: `D:\TU_CODE\ai-supervisor`
* **Audited Commit**: `2fc0c73d5181f00cff18a5e2906133b1e39eb7df`
* **Pinned AO Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
* **Audit Date**: 2026-09-22
* **Active Gate**: `P03_CANONICAL_RECONCILIATION_REVISION_3`

---

## External Findings

All four findings from the External Supervisor Audit on commit `2fc0c73d5181f00cff18a5e2906133b1e39eb7df` have been addressed:

| Finding ID | Name | Remediated Status | Canonical Remedy Applied |
|---|---|---|---|
| **P03R3-001** | `FR015_AOADAPTER_INTERFACE_INCOMPLETE` | `CLOSED` | Expanded canonical `IAOAdapter` in `docs/12_UPSTREAM_INTEGRATION.md` to include normalized read-only preflight operations: `checkReadiness()`, `listAgents()`, `getAgentReadiness()`, `getAPIContract()`. |
| **P03R3-002** | `STALE_SUBSCRIBE_EVENTS_CONTRACT` | `CLOSED` | Removed `subscribeEvents(...)` from active canonical `IAOAdapter` in `docs/12_UPSTREAM_INTEGRATION.md`. Documented CDC event streaming as a deferred optional future optimization. Observation baseline remains bounded snapshot polling. |
| **P03R3-003** | `REC005_STATE_MODEL_INCONSISTENT` | `CLOSED` | Reconciled REC-005 in `docs/14_FAILURE_RECOVERY.md` against P02 domain/persistence reality: Case A (pre-`PrepareDispatch`) rejects dispatch, leaving task `READY` with zero attempt allocated; Case B (post-`PrepareDispatch`) transitions `DISPATCHED -> FAILED` with audit log evidence. No attempt schema mutation. |
| **P03R3-004** | `AO_PUBLIC_EXIT_CODE_OVERCLAIM` | `CLOSED` | Removed unsupported assumption in `docs/22_MODULE_PROVENANCE.md` that AO public API provides process exit codes. Reconciled `EvidenceCollector` and `AOAdapter` descriptions around observable session termination evidence. |

---

## IAOAdapter Preflight Reconciliation
In `docs/12_UPSTREAM_INTEGRATION.md`, the conceptual `IAOAdapter` interface has been reconciled to provide the complete preflight probe surface demanded by FR-015:
* `checkHealth(): Promise<AOHealthStatus>` -> Loopback `GET /healthz` (daemon liveness, `status = "ok"`).
* `checkReadiness(): Promise<AOReadinessStatus>` -> Loopback `GET /readyz` (daemon readiness, `status = "ready"`).
* `listAgents(): Promise<AOAgentList>` -> `GET /api/v1/agents` (verifies required harness presence, e.g. `"agy"`).
* `getAgentReadiness(agentName?: string): Promise<AOAgentReadiness>` -> `GET /api/v1/agents/readiness` (harness execution readiness).
* `getAPIContract(): Promise<string>` -> `GET /api/v1/openapi.yaml` (public OpenAPI contract surface).

### Public Contract vs Release Version Classification:
* `getAPIContract()` returns the raw embedded OpenAPI 3.1.0 specification (`info.version = 0.1.0-route-shell`).
* `OPENAPI_CONTRACT = API_COMPATIBILITY_SIGNAL_ONLY`: Validates expected route and schema surface compatibility.
* `RUNTIME_AO_RELEASE_VERSION_PUBLIC_FIELD = ABSENT`: No public REST probe returns binary release string `v0.13.0`.
* `PINNED_RELEASE_IDENTITY = DEPLOYMENT_PROVENANCE`: Release identity is established out-of-band via verified build provenance and commit SHA.

---

## Event Subscription Contract Cleanup
1. `subscribeEvents(sessionId, onEvent)` has been formally removed from active canonical `IAOAdapter`.
2. Pinned AO `/api/v1/events` is a global SQLite change-data-capture (CDC) stream (`session_created`, `session_updated`, etc.). It is not session-scoped, does not provide worker heartbeats, and does not govern turn completion.
3. `CDC Event Subscription = DEFERRED OPTIONAL FUTURE OPTIMIZATION`: Reserved strictly for future wake-up hints.
4. Active P03 baseline worker observation relies exclusively on authoritative session snapshot polling via `getWorkerStatus(sessionId)` (`GET /api/v1/sessions/{id}`).

---

## REC-005 State-Model Reconciliation
In `docs/14_FAILURE_RECOVERY.md`, REC-005 was reconciled with P02 domain/store ground truth (`TaskAttempt` has no `failure_reason` column; `StateMachine` forbids `READY -> BLOCKED` or `READY -> FAILED`):

* **Case A — Dirty Worktree Detected Prior to `PrepareDispatch`**:
  - Uncommitted files or untracked artifacts detected during pre-dispatch check before atomic `PrepareDispatch` is invoked.
  - **Behavior**: Reject dispatch immediately.
  - **State Invariant**: Task remains in `READY`; zero `TaskAttempt` allocated; zero AO execution triggered.
  - **Audit Record**: Emit audit evidence `WORKTREE_DIRTY`.
  - **Resolution**: Escalate to human developer or upstream orchestrator for external clean; validation retried without DB surgery.

* **Case B — Unsafe Worktree Condition Discovered Post-`PrepareDispatch`**:
  - Unsafe condition detected after atomic `PrepareDispatch` has succeeded (Task is `DISPATCHED`, `TaskAttempt` exists).
  - **Behavior**: Execute canonical transition `DISPATCHED -> FAILED`.
  - **State Invariant**: Task transitions to `FAILED`. Zero schema mutation (no `failure_reason` column on `TaskAttempt`).
  - **Audit Record**: Failure cause `WORKTREE_DIRTY` recorded in append-only audit event log.
  - **Resolution**: Escalate to human or failure recovery handler. Zero `git stash`, `git clean`, `git checkout`, or `git reset` executed by Supervisor.

---

## AO Public Termination Evidence Boundary
* Public session snapshots (`GET /api/v1/sessions/{id}`) expose `IsTerminated: bool` and `Activity.State`, but do not expose an authoritative process exit code or termination reason.
* `docs/22_MODULE_PROVENANCE.md` has been updated to remove the claim that "AO reports session exit code".
* Observation classification remains:
  - `WORKER_STOPPED`: Requires Supervisor intentional-stop provenance (e.g. correlated `stopWorker`/`kill` action).
  - `WORKER_CRASHED`: Unexpected termination when no intentional stop was requested, backed by observable exit signals.
  - `WORKER_TERMINATION_UNKNOWN`: Fail closed when evidence cannot safely distinguish stop from crash.
  - No new `TaskState` is invented; existing task StateMachine remains authoritative.

---

## FR-015 Surface Alignment
In `docs/phases/P03_AO_INTEGRATION.md`, the preflight REST surface list is aligned with FR-015:
* `GET /healthz`
* `GET /readyz`
* `GET /api/v1/agents`
* `GET /api/v1/agents/readiness`
* `GET /api/v1/openapi.yaml`
In addition to already approved routes: `POST /api/v1/projects`, `POST /api/v1/sessions`, `POST /api/v1/sessions/{id}/send`, `POST /api/v1/sessions/{id}/kill`, `POST /api/v1/sessions/{id}/restore`, `GET /api/v1/sessions/{id}`, and `GET /api/v1/sessions/{id}/workspace/file?path={relPath}`.

---

## Module Provenance Alignment
`docs/22_MODULE_PROVENANCE.md` reflects exact P03 boundaries:
* `AOAdapter` purpose explicitly enumerates preflight probes, lifecycle operations, snapshot observation, and raw workspace read transport.
* `AOAdapter` forbidden responsibilities explicitly prohibit StateStore access, SQLite access, TaskAttempt allocation, Task state transitions, WorkerReport semantic interpretation, worktree mutation, direct Agy invocation, and AO internal package coupling.
* `EvidenceCollector` provenance removes unsupported assumptions regarding AO public process exit codes.

---

## Unresolved Operational Policies

The 8 Supervisor operational policies remain explicitly marked `UNSET`:

| Policy Identifier | Owner | Value | Status |
|---|---|---|---|
| `UPSTREAM_AO_REQUEST_TIMEOUT` | Upstream AO | `60s` | `UPSTREAM_FACT` (`backend/internal/config/config.go:42`) |
| `SUPERVISOR_HTTP_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_HEALTH_PROBE_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_SPAWN_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_SEND_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_ACTIVITY_POLL_INTERVAL`| Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_EXECUTION_DEADLINE` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_KILL_STOP_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_WORKSPACE_READ_TIMEOUT`| Supervisor | `UNSET` | `UNRESOLVED_POLICY` |

---

## Production Code Diff Check
```bash
git diff -- internal/ go.mod go.sum schemas/ docs/adr/
# Result: EMPTY (0 lines added, 0 lines modified, 0 lines deleted)
```
Zero production Go code, zero schema changes, zero ADR modifications.

---

## Gate Verdict

```yaml
P03R3_001: CLOSED
P03R3_002: CLOSED
P03R3_003: CLOSED
P03R3_004: CLOSED
PROPOSAL_P03_001: EXTERNAL_APPROVED
P03_ARCHITECTURE_CHANGE: NO
P03_ADR_REQUIRED: NO
P03_CODE: HELD
TASK_P03_001: NOT_RELEASED
P03_CANONICAL_RECONCILIATION: READY_FOR_EXTERNAL_REAUDIT
```
