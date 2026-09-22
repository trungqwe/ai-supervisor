# PROPOSAL-P03-001: Upstream Runtime Observation, Compatibility Contract, and Interface Reconciliation

> **Proposal ID**: `PROPOSAL-P03-001`
> **Date**: 2026-09-22
> **Author**: AI Engineering Supervisor Worker
> **Target Phase**: P03 (Agent Orchestrator Integration)
> **Status**: `EXTERNAL_APPROVED`
> **Governance Stage**: `CHANGE_INTAKE_PROPOSAL`
> **External Decision**: `APPROVED_WITH_REVISION_2_CORRECTIONS_APPLIED`
> **Architecture Change**: `NO`
> **ADR Required**: `NO`
> **Governing Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)

---

# Problem Statement

During the Phase P03 Pre-Code Upstream Contract Audit against pinned Untrivial Agent Orchestrator (AO) v0.13.0, the External Supervisor identified critical contract discrepancies between canonical specifications and upstream source reality:

1. **FR-006 Heartbeat Discrepancy**: Canonical FR-006 requires monitoring "heartbeats" via AO public API. Pinned AO v0.13.0 contains no worker heartbeat; its SSE keepalive comment frame (`: \n\n`) is transport-level only.
2. **FR-015 Health Probe Field Discrepancy**: Pinned AO `GET /healthz` and `GET /readyz` expose daemon status, service, and PID, but **NO version field**. Version `v0.13.0` is proven via deployment provenance; OpenAPI contains `info.version = 0.1.0-route-shell`.
3. **IAOAdapter Interface Gap**: Canonical `IAOAdapter` lacked a workspace file retrieval method, while ADR-011 and FR-007 require retrieving worker reports via AO REST.
4. **REC-005 Readonly Boundary Conflict**: Canonical `docs/14_FAILURE_RECOVERY.md` REC-005 directed AOAdapter to "stash or clean" worktrees, violating the Supervisor read-only architectural invariant.
5. **StateStore Ownership Separation**: AOAdapter must strictly remain an anti-corruption transport adapter with zero direct dependency on StateStore or SQLite, and zero Task state transition authority.

---

# Authority Conflict Matrix

| Conflict Area | Canonical Specification | Pinned Upstream AO Reality | Governance Impact |
|---|---|---|---|
| **Worker Heartbeat** | `docs/02_REQUIREMENTS.md` FR-006: monitor process health and heartbeats. | Pinned AO v0.13.0 has no worker heartbeat event. SSE keepalive `: \n\n` is transport-only. | Clarify FR-006: Authoritative session snapshot observation; no synthetic worker heartbeat. |
| **Health Probe Version** | `docs/02_REQUIREMENTS.md` FR-015: returns daemon status, version, harness. | `backend/internal/httpd/router.go` probe payload has `status`, `service`, `pid`, `paths`, but **no version field**. | Clarify FR-015: Liveness/readiness via `/healthz`/`/readyz`; harness via `/api/v1/agents`; version via pinned deployment provenance. |
| **OpenAPI Schema Version** | Historical worker claim: OpenAPI proves release version v0.13.0. | Pinned `openapi.yaml` defines `info.version = 0.1.0-route-shell`, NOT `v0.13.0`. | Classify OpenAPI fingerprint strictly as API compatibility signal, NOT release identity proof. |
| **Workspace File Read** | `docs/04_ARCHITECTURE.md` ADR-011 / FR-007: retrieve report via AO HTTP API. | Canonical `IAOAdapter` interface lacked `GetWorkspaceFile`. | Add session-scoped, read-only `GetWorkspaceFile` primitive to `IAOAdapter`. |
| **Dirty Worktree Handling** | `docs/14_FAILURE_RECOVERY.md` REC-005: "Stash or clean untracked artifacts". | Violates Supervisor read-only invariant and ADR-009 anti-reinvention. | Reconcile REC-005: Fail closed (`WORKTREE_DIRTY`), block dispatch, escalate to operator/human. |
| **StateStore Ownership** | Prior revision ambiguity: adapter checks dispatch authority and executes transitions. | AOAdapter is an anti-corruption transport adapter. | Strict separation: `AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`. Supervisor orchestration layer owns transitions. |

---

# Pinned AO Ground Truth Evidence

### 1. Daemon Probe Payload (`backend/internal/httpd/router.go:394-420`)
```go
func daemonProbePayload(r *http.Request, state *daemonLifecycleState) map[string]any {
    // ...
    res := map[string]any{
        "status":  "ok",
        "service": "agent-orchestrator-daemon",
        "pid":     os.Getpid(),
        "paths":   paths,
    }
    // NO "version" field exists in the returned JSON object.
    return res
}
```

### 2. Embedded OpenAPI Specification (`backend/internal/httpd/apispec/openapi.yaml:1-6`)
```yaml
openapi: 3.1.0
info:
  description: Loopback-only HTTP surface served by the Go daemon. Generated from Go (code-first)...
  title: Agent Orchestrator HTTP daemon
  version: 0.1.0-route-shell
```
`info.version` is a route-shell spec version, not daemon release version `v0.13.0`.

### 3. Activity State Machine (`backend/internal/domain/activity.go:17-30`)
```go
const (
    ActivityActive       ActivityState = "active"
    ActivityIdle         ActivityState = "idle"
    ActivityWaitingInput ActivityState = "waiting_input"
    ActivityBlocked      ActivityState = "blocked"
    ActivityExited       ActivityState = "exited"
)
```
* Note: `ActivityBusy` does **not** exist in pinned AO.
* `ActivityActive`: worker actively executing turns/tools.
* `ActivityIdle`: agent turn complete (derived from agent `stop` hook).
* `Activity.LastActivityAt`: timestamp when activity state was last observed. **Not an autonomous worker heartbeat.** Legitimate long-running tool operations produce no intermediate callbacks.

### 4. Session Read Model & Exit Semantics (`backend/internal/domain/session.go:21-39`)
```go
type Session struct {
    ID           string        `json:"id"`
    ProjectID    string        `json:"projectId"`
    Harness      string        `json:"harness"`
    Status       SessionStatus `json:"status"`
    Activity     Activity      `json:"activity"`
    IsTerminated bool          `json:"isTerminated"`
    // ...
}
```
* Public read model exposes `Activity.State`, `IsTerminated`, `Status`.
* It does **not** expose a reliable public exit code or reason.
* A single snapshot cannot distinguish intentional stop from an unexpected crash without Supervisor operational provenance.

---

# FR-006 Lifecycle Observation Resolution

### Reality:
Pinned AO v0.13.0 does not produce worker-level heartbeats.

### Semantics:
1. **Baseline Transport**: Authoritative AO session snapshot observation via `GET /api/v1/sessions/{id}` is the P03 baseline.
2. **State Transition**: Observing `Activity.State == ActivityActive` permits the Supervisor orchestration layer to transition task state `DISPATCHED -> RUNNING` via P02 StateStore APIs.
3. **Turn Completion**: `Activity.State == ActivityIdle` represents worker turn completion per ADR-011.
4. **No Synthetic Heartbeat**: Zero synthetic `worker.heartbeat` events shall be invented or emitted.
5. **Activity Timestamp Clarification**: `Activity.LastActivityAt` is diagnostic activity evidence, NOT hang detection heartbeat proof. A long-running tool may legitimately run without intermediate callbacks. Bounded execution deadlines are governed by Supervisor execution policy, not solely by `lastActivityAt`.

---

# FR-015 Runtime Compatibility Resolution

### Reality:
`GET /healthz` and `GET /readyz` return daemon health and PID, but lack a `version` field.

### Semantics:
1. **Daemon Liveness**: Verified via `GET /healthz` (returns `status="ok"`).
2. **Daemon Readiness**: Verified via `GET /readyz` (returns `status="ready"`).
3. **Harness Inventory**: Verified via `GET /api/v1/agents` (verifies `"agy"` harness exists).
4. **Harness Readiness**: Verified via `GET /api/v1/agents/readiness` and/or `POST /api/v1/agents/readiness/ensure`.
5. **API Compatibility**: Verified against expected public contract / schema surface.
6. **Release Identity**: Verified via out-of-band pinned deployment provenance (commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`), NOT health JSON.
7. **OpenAPI Schema Fingerprint**: API compatibility signal only. Digest equality indicates expected API schema surface, NOT proof that the running binary is release `v0.13.0`.

---

# Stopped vs Crashed Classification

Pinned AO session snapshots expose `IsTerminated` and `Activity.State`, but do not expose an authoritative exit reason. Therefore:
* `IsTerminated == true` MUST NOT automatically mean `worker.crashed`.
* `ActivityExited` MUST NOT automatically mean intentional `worker.stopped`.

### Normalized Event Mapping:
* **`WORKER_STOPPED`**: Supervisor has authoritative intentional-stop provenance (e.g., successful `stopWorker`/`kill` action correlated to that session/attempt).
* **`WORKER_CRASHED`**: Session/process terminates unexpectedly when no intentional termination was requested by the Supervisor, corroborated by observable termination evidence.
* **`WORKER_TERMINATION_UNKNOWN`**: If evidence cannot safely distinguish intentional stop from crash, fail closed and retain an unknown termination classification for failure recovery handling.

---

# AOAdapter and StateStore Ownership Separation

The `AOAdapter` is strictly an anti-corruption transport adapter.

### Explicit Boundary Rules:
* `AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`
* `AOADAPTER_DIRECT_SQL = FORBIDDEN`
* `TASK_STATE_TRANSITION_OWNER = SUPERVISOR_ORCHESTRATION_LAYER_USING_P02_APIS`

### Execution Flow:
1. **Supervisor Orchestration Layer**: Prepares dispatch, creates durable `TaskAttempt`, commits `READY -> DISPATCHED` in StateStore, then invokes `AOAdapter.dispatchTaskContract`.
2. **AOAdapter Layer**: Translates domain-safe input to AO HTTP requests (`POST /api/v1/sessions/{id}/send`), returns normalized transport result.
3. **Observation & Transition**: Supervisor observation loop polls session snapshot via AOAdapter; upon observing `ActivityActive`, Supervisor orchestration layer invokes existing P02 domain/StateStore APIs to transition `DISPATCHED -> RUNNING`.

---

# AO Workspace Read Transport Primitive

Add to `IAOAdapter`:
```go
// GetWorkspaceFile retrieves raw artifact bytes from the session workspace.
// Strictly session-scoped, path-confined, and read-only.
GetWorkspaceFile(ctx context.Context, sessionID string, relativePath string) ([]byte, error)
```

### Constraints:
* Calls `GET /api/v1/sessions/{id}/workspace/file?path={relPath}`.
* Read-only; relative path confinement (rejects traversal `..`, absolute paths).
* Returns raw byte payload.
* **Zero WorkerReport semantic interpretation**: No JSON parsing, no schema validation, no `WorkerClaim` creation, no `REPORT_READY` transition (strictly reserved for Phase P04 `EvidenceCollector`).

---

# REC-005 Readonly Worktree Resolution

### Problem:
Legacy REC-005 directed AOAdapter to "stash or clean" worktrees, violating the Supervisor read-only invariant.

### Resolution:
* Supervisor **fails closed** upon detecting a dirty or unsafe worktree precondition.
* Rejects dispatch, marks task attempt blocked/failed (`failure_reason = WORKTREE_DIRTY`), and escalates to human/operator resolution.
* Zero mutating git operations (`git stash`, `git clean`, `git checkout`, `git reset`) are executed by the Supervisor.

---

# Policy Inventory — Canonical Status

All Supervisor operational policies are tracked without unauthorized numeric value inventions:

| Policy Identifier | Owner | Value | Status |
|---|---|---|---|
| `UPSTREAM_AO_REQUEST_TIMEOUT` | Upstream AO | `60s` | `UPSTREAM_FACT` (`config.DefaultRequestTimeout`) |
| `SUPERVISOR_HTTP_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_HEALTH_PROBE_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_SPAWN_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_SEND_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_ACTIVITY_POLL_INTERVAL`| Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_EXECUTION_DEADLINE` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_KILL_STOP_TIMEOUT` | Supervisor | `UNSET` | `UNRESOLVED_POLICY` |
| `SUPERVISOR_WORKSPACE_READ_TIMEOUT`| Supervisor | `UNSET` | `UNRESOLVED_POLICY` |

*Note: No circuit-breaker or retry-count policies are introduced without formal governance approval.*

---

# Revised P03 Task Ownership

```
TASK-P03-001: Upstream Contract & Client Foundation (Transport, Probes, Fingerprint)
  └── TASK-P03-002: Session Lifecycle & Dispatch (Create, Send, Kill, Restore)
        └── TASK-P03-003: Lifecycle Observation & State Reconciliation (Poll Loop, DISPATCHED->RUNNING)
              └── TASK-P03-004: AO Workspace Read Transport Primitive (GetWorkspaceFile raw read)
                    └── TASK-P03-005: P03 Live AO Integration & Exit Gate Validation
```

* **TASK-P03-001**: AO loopback HTTP client foundation, `/healthz`, `/readyz`, `/api/v1/agents`, error decoding, OpenAPI fingerprinting.
* **TASK-P03-002**: Session creation, prompt dispatch, termination, restore; pure transport boundary (zero StateStore dependency).
* **TASK-P03-003**: Authoritative session state observation loop (`GET /sessions/{id}`); Supervisor orchestration maps active state to `DISPATCHED -> RUNNING` via P02 APIs.
* **TASK-P03-004**: Session-confined raw workspace file reader (`GET /sessions/{id}/workspace/file`); zero report parsing or claim creation.
* **TASK-P03-005**: End-to-end integration test against live pinned AO daemon proving P03 contract compliance.

---

# Architecture & ADR Impact Assessment

* **Architecture Change**: `NO`. Canonical 3-tier architecture preserved.
* **ADR Required**: `NO`. Existing ADRs (`ADR-002`, `ADR-009`, `ADR-011`, `ADR-012`) govern these boundaries.
* **Canonical Reconciliations**: Completed in `docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`.

---

# External Approval Gate

* **Status**: `EXTERNAL_APPROVED`
* **External Supervisor Decision**: `APPROVED_WITH_REVISION_2_CORRECTIONS_APPLIED`
* **Production Code Guard**: `P03_CODE = HELD`. Implementation authorization pending release of immutable `TASK-P03-001` Task Contract.
