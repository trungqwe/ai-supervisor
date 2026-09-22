# P03 Pre-Code Upstream Contract Reaudit 002

## Repository Identity
* **Repository**: `D:\TU_CODE\ai-supervisor`
* **Audited Commit**: `0eefe64ce47a400e384b240b45d91074900db9d1`
* **Pinned AO Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
* **Audit Date**: 2026-09-22
* **Audit Stage**: `P03_PRECODE_UPSTREAM_CONTRACT_REAUDIT_002`

---

## External Decision
The External Supervisor independently reviewed commit `0eefe64ce47a400e384b240b45d91074900db9d1`, `PROPOSAL-P03-001`, `P03_PRECODE_UPSTREAM_CONTRACT_REAUDIT_001`, canonical governance, and pinned AO public source.

**Core Directions Approved**:
1. **FR-006**: Authoritative AO session snapshot observation is the P03 baseline.
2. **Worker Heartbeat**: No synthetic worker heartbeat event shall be invented.
3. **CDC**: `/api/v1/events` is NOT the lifecycle authority. CDC is deferred as a wake-up optimization only. P03 baseline uses authoritative GET session snapshots.
4. **FR-015**: Liveness/readiness is verified via `/healthz` and `/readyz`. Harness inventory/readiness via public agent endpoints. Runtime AO release version is NOT exposed by health endpoints.
5. **GetWorkspaceFile**: A session-scoped, read-only raw workspace-file transport primitive belongs in AOAdapter because ADR-011 already requires this upstream surface.
6. **REC-005**: Supervisor must fail closed. No `git stash`, `git clean`, checkout, reset, or custom worktree mutation.
7. **P03/P04 Boundary**: P03 owns raw AO transport only. P04 owns WorkerReport semantic ingestion, schema validation, WorkerClaim creation, and `REPORT_READY` transition.
8. **Architecture Impact**: NO architecture change.
9. **ADR**: NO new ADR required.

However, `P03_PRECODE_REAUDIT_001` was marked `REVISION_REQUIRED` due to 8 specific defects (findings `P03R2-001` through `P03R2-008`).

---

## Revision-1 Defects
* **P03R2-001 (UNAUTHORIZED_POLICY_VALUE_INVENTION)**: Revision 1 and worker prose invented operational numbers (10s, 15s, 5s, 1s-2s, 300s, 30s, 3 retries, 5-failure circuit breaker, 60s reset) without governance approval.
* **P03R2-002 (OPENAPI_FINGERPRINT_IS_NOT_RELEASE_VERSION)**: Pinned OpenAPI contains `info.version = 0.1.0-route-shell`, NOT `v0.13.0`. Fingerprint was incorrectly described as proving binary release version.
* **P03R2-003 (LAST_ACTIVITY_IS_NOT_HANG_HEARTBEAT)**: Pinned `Activity.LastActivityAt` was mischaracterized as a hang detection heartbeat. Legitimate long-running tools produce no intermediate callbacks.
* **P03R2-004 (STOPPED_VS_CRASHED_NOT_DERIVABLE_FROM_SNAPSHOT_ALONE)**: Pinned session snapshots expose `IsTerminated` and `Activity.State` without exit reason/code; stop vs crash cannot be derived from a single snapshot alone.
* **P03R2-005 (AOADAPTER_MUST_NOT_OWN_STATESTORE)**: Revision 1 blurred ownership; AOAdapter must have zero StateStore dependency and zero Task state transition authority.
* **P03R2-006 (PROPOSAL_GOVERNANCE_RANK_INCORRECT)**: Proposal was falsely labeled "Decision Priority: Level 3 Proposal".
* **P03R2-007 (WORKER_FINAL_PROSE_DEFECTS)**: Worker final response contained `/api/v1/health` (instead of `/healthz`), non-existent `ActivityBusy`, unestablished auth headers, and unapproved circuit breakers.
* **P03R2-008 (CANONICAL_AMENDMENT_SET_INCOMPLETE)**: Canonical specifications had not been reconciled across the required documents.

---

## Proposal Corrections
`PROPOSAL-P03-001` has been revised in place:
1. Status transitioned to `EXTERNAL_APPROVED` with `External Decision: APPROVED_WITH_REVISION_2_CORRECTIONS_APPLIED`.
2. Removed false hierarchy rank `Decision Priority: Level 3 Proposal`.
3. Stripped all invented operational defaults; policies marked `STATUS = UNRESOLVED_POLICY, VALUE = UNSET`.
4. Corrected OpenAPI fingerprint semantics to API compatibility signal only.
5. Clarified `lastActivityAt` as diagnostic evidence, not hang heartbeat.
6. Reconciled stop vs crash vs unknown termination semantics.
7. Explicitly asserted `AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`.

---

## FR-006 Final Semantics
The Supervisor shall observe worker lifecycle and bounded execution health via AO public session/activity surfaces:
* Active session observation permits state transition from `DISPATCHED` to `RUNNING`.
* AO `idle` (`ActivityIdle`) represents worker turn completion per ADR-011.
* Intentional termination and unexpected termination are distinguished using Supervisor operation provenance where possible.
* Task/operation timeout handling uses separately configured bounded execution deadlines.
* No synthetic worker heartbeat event is required or fabricated.
* `lastActivityAt` is diagnostic activity evidence, not heartbeat proof.

---

## FR-015 Final Semantics
The Supervisor shall verify daemon connectivity, readiness, harness inventory, and public API compatibility with the AO daemon before dispatching tasks:
1. Daemon liveness verified via `GET /healthz`.
2. Daemon readiness verified via `GET /readyz`.
3. Required harness presence verified via `GET /api/v1/agents`.
4. Required harness readiness verified via `GET /api/v1/agents/readiness`.
5. API compatibility verified against expected public contract/schema surface.
6. Release identity verified via pinned deployment provenance (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`).

---

## Runtime Release vs API Compatibility
* `HEALTH_VERSION_FIELD = ABSENT`: Pinned AO health/ready probes do not return a release version.
* `RUNTIME_RELEASE_VERSION_PUBLIC_FIELD = ABSENT`: No public REST endpoint exposes the binary release version string `v0.13.0`.
* `PINNED_RELEASE_PROVENANCE = OUT_OF_BAND`: Provenance is established by verified build artifacts and commit hash.
* `OPENAPI_SCHEMA_FINGERPRINT = API_COMPATIBILITY_SIGNAL_ONLY`: The embedded OpenAPI specification (`0.1.0-route-shell`) validates route/schema compatibility, NOT the daemon release version.
* `OPENAPI_SCHEMA_FINGERPRINT_RUNTIME_PROOF = NOT_YET_PROVEN`: Runtime digest verification is evaluated for future production compatibility, not claimed as already proven.

---

## Lifecycle Observation Semantics
1. Observation is grounded on authoritative session snapshots (`GET /api/v1/sessions/{id}`).
2. Pinned AO `Activity.State` values: `ActivityActive` ("active"), `ActivityIdle` ("idle"), `ActivityWaitingInput` ("waiting_input"), `ActivityBlocked` ("blocked"), `ActivityExited` ("exited"). (`ActivityBusy` does not exist).
3. The observation loop polls snapshots and translates active activity into the Supervisor domain.
4. The SSE endpoint `/api/v1/events` is an internal CDC event stream, not the lifecycle authority. It may serve only as a future wake-up hint.

---

## Stopped vs Crashed Classification
Because pinned AO session snapshots do not provide an authoritative exit code or reason:
* `WORKER_STOPPED`: Supervisor has authoritative intentional-stop provenance (e.g. successful `stopWorker`/`kill` action correlated to that session/attempt).
* `WORKER_CRASHED`: Session/process exits unexpectedly when no expected intentional termination is in progress, supported by observable termination signals.
* `WORKER_TERMINATION_UNKNOWN`: When evidence cannot safely distinguish stop from crash, the system fails closed and treats the attempt as an unknown termination for failure recovery rather than fabricating certainty.

---

## AOAdapter / StateStore Ownership Boundary
* `AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`
* `AOADAPTER_DIRECT_SQL = FORBIDDEN`
* `TASK_STATE_TRANSITION_OWNER = SUPERVISOR_ORCHESTRATION_LAYER_USING_P02_APIS`

**Execution Separation**:
* Supervisor Orchestration Layer: Prepares dispatch -> creates durable `TaskAttempt` -> commits `READY -> DISPATCHED` in StateStore -> invokes `AOAdapter.dispatchTaskContract`.
* AOAdapter: Accepts domain-safe parameters -> executes HTTP REST call -> returns normalized transport result.
* Observation & State Advancement: Supervisor observation loop inspects session status via AOAdapter; upon observing `ActivityActive`, Supervisor orchestration layer invokes P02 domain/StateStore APIs to commit `DISPATCHED -> RUNNING`.

---

## Workspace Read Boundary
* Operation: `getWorkspaceFile(sessionId: string, relativePath: string): Promise<Uint8Array>` (or Go equivalent `GetWorkspaceFile(ctx, sessionID, relPath) ([]byte, error)`).
* Transport: Calls AO public REST `GET /api/v1/sessions/{id}/workspace/file?path={relPath}`.
* Constraints: Strictly session-scoped, path-confined (rejects traversal, absolute paths), read-only.
* Responsibility: Pure raw transport primitive. Zero `WorkerReport` parsing, zero JSON schema validation, zero `WorkerClaim` creation, zero StateStore transitions.

---

## REC-005 Resolution
* Legacy wording: Directed AOAdapter to "stash or clean" worktrees.
* Reconciled wording: Supervisor **fails closed**.
* Precondition: Worktree contains uncommitted files or unsafe state prior to dispatch.
* Action: Mark attempt blocked/failed (`failure_reason = WORKTREE_DIRTY`), abort dispatch, escalate to human or upstream-owned resolution.
* Guardrail: Zero `git stash`, `git clean`, `git checkout`, `git reset`, or custom worktree mutations executed by the Supervisor.

---

## P03 / P04 Phase Boundary
* **Phase P03 (AO Integration)**: Owns AO loopback REST transport, session lifecycle, dispatch, authoritative snapshot observation, orchestration integration for `DISPATCHED -> RUNNING`, and raw workspace file read transport (`GetWorkspaceFile`).
* **Phase P04 (Evidence & Review Engine)**: Owns `EvidenceCollector`, `worker-report.json` schema validation, `WorkerClaim` extraction, independent Git diff evidence collection, policy compliance evaluation, and `REPORT_READY` / `EVIDENCE_READY` state transitions.

---

## Policy Inventory — No Defaults

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

*Note: Zero invented defaults. No circuit-breaker or retry-count policies are introduced without formal governance.*

---

## Canonical Documents Reconciled
1. `docs/02_REQUIREMENTS.md`: FR-006 and FR-015 amended to reflect authoritative observation and exact health/readiness/provenance checks.
2. `docs/12_UPSTREAM_INTEGRATION.md`: Added `getWorkspaceFile` read primitive; recorded `AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`.
3. `docs/14_FAILURE_RECOVERY.md`: REC-005 amended to fail closed (`WORKTREE_DIRTY`) with zero git mutation.
4. `docs/17_ROADMAP.md`: P03 deliverables and exit gate aligned with pure transport and observation scope (no P04 report validation).
5. `docs/21_TRACEABILITY_MATRIX.md`: FR-006 and FR-015 updated; FR-007 retained in P04.
6. `docs/22_MODULE_PROVENANCE.md`: `AOAdapter` updated with workspace read primitive and forbidden StateStore/worktree responsibilities.
7. `docs/phases/P03_AO_INTEGRATION.md`: Legacy streamed heartbeat deliverable replaced with canonical snapshot observation and P03/P04 boundary.
8. `docs/proposals/PROPOSAL-P03-001-runtime-observation-and-compatibility-contract.md`: Revised in place to `EXTERNAL_APPROVED`.

---

## Architecture / Observability No-Change Assessment
* `docs/04_ARCHITECTURE.md`: Audited; no literal contradictions found. Session idle observation and workspace file fetching are already modeled. `ARCHITECTURE_CHANGE_REQUIRED = NO`.
* `docs/15_OBSERVABILITY.md`: Audited; canonical worker events are exclusively `worker.started`, `worker.stopped`, and `worker.crashed`. Zero `worker.heartbeat` present. `OBSERVABILITY_CHANGE_REQUIRED = NO`.

---

## Remaining Decisions
Operational values for the 8 Supervisor policies remain `UNSET` pending operational calibration in Phase P03 implementation:
1. `SUPERVISOR_HTTP_TIMEOUT`
2. `SUPERVISOR_HEALTH_PROBE_TIMEOUT`
3. `SUPERVISOR_SPAWN_TIMEOUT`
4. `SUPERVISOR_SEND_TIMEOUT`
5. `SUPERVISOR_ACTIVITY_POLL_INTERVAL`
6. `SUPERVISOR_EXECUTION_DEADLINE`
7. `SUPERVISOR_KILL_STOP_TIMEOUT`
8. `SUPERVISOR_WORKSPACE_READ_TIMEOUT`

---

## Gate Verdict

```yaml
PROPOSAL_P03_001: EXTERNAL_APPROVED
P03_ARCHITECTURE_CHANGE: NO
P03_ADR_REQUIRED: NO
WORKER_HEARTBEAT: NOT_REQUIRED
SYNTHETIC_WORKER_HEARTBEAT: FORBIDDEN
RUNTIME_AO_RELEASE_VERSION: NOT_EXPOSED_BY_PUBLIC_HEALTH_API
OPENAPI_FINGERPRINT: API_COMPATIBILITY_ONLY
AOADAPTER_STATESTORE_DEPENDENCY: FORBIDDEN
P04_SCOPE_LEAK: CLOSED
P03_CODE: HELD
TASK_P03_001: NOT_RELEASED
```
