# P03 Pre-Code Upstream Contract Audit

> **Status**: READY_FOR_EXTERNAL_AUDIT
> **Date**: 2026-09-22
> **Phase Gate**: P03_PRECODE_UPSTREAM_CONTRACT_AUDIT
> **Audit Mode**: ANALYSIS AND DOCUMENTATION ONLY — ZERO PRODUCTION CODE

---

## Repository Identity

| Field | Value |
|---|---|
| **Repository** | `D:\\TU_CODE\\ai-supervisor` |
| **HEAD at Audit Start** | `6ccba2706dcffe01fb558ac45f9a1a073106f4af` (Stage A closeout commit) |
| **Approved P02 Baseline** | `b073a1969d0b64d151e96f0d9b59b005d4cf518a` |
| **Working Tree Status** | CLEAN |
| **Branch** | `main` |

---

## Authority Chain

Documents read in sequence before audit:

1. `AGENTS.md` — Phase P03 pre-code gate rules (updated in Stage A)
2. `docs/18_CURRENT_STATE.md` — Active gate: `P03_PRECODE_UPSTREAM_CONTRACT_AUDIT`
3. `docs/02_REQUIREMENTS.md` — FR-005, FR-006, FR-015
4. `docs/04_ARCHITECTURE.md` — Architecture V2.1
5. `docs/06_WORKFLOW_STATE_MACHINE.md` — 13-state model
6. `docs/12_UPSTREAM_INTEGRATION.md` — IAOAdapter interface (TypeScript placeholder)
7. `docs/14_FAILURE_RECOVERY.md` — Recovery classification
8. `docs/15_OBSERVABILITY.md` — Canonical event taxonomy
9. `docs/16_TEST_STRATEGY.md` — Test requirements
10. `docs/17_ROADMAP.md` — Phase sequence
11. `docs/21_TRACEABILITY_MATRIX.md` — Requirement traceability
12. `docs/22_MODULE_PROVENANCE.md` — Module provenance
13. `docs/24_CHANGE_GOVERNANCE.md` — ADR process
14. `docs/phases/P03_AO_INTEGRATION.md` — Phase specification
15. `docs/sources/SOURCE_REGISTRY.md` — Upstream registry
16. `docs/sources/REUSE_MATRIX.md` — Reinvention checks
17. `docs/sources/UPSTREAM_CONTRACT_BASELINE.md` — Proven endpoints
18. `docs/sources/01_AGENT_ORCHESTRATOR.md` — AO documentation
19. `docs/adr/ADR-002-ao-as-execution-control-plane.md`
20. `docs/adr/ADR-003-agy-primary-antigravity-interface.md`
21. `docs/adr/ADR-004-no-gui-automation-by-default.md`
22. `docs/adr/ADR-009-upstream-over-reimplementation.md`
23. `docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`
24. `docs/adr/ADR-012-task-contract-revision-and-attempt-binding.md`
25. `docs/audits/P01_A_AO_RUNTIME_PROOF.md`
26. `docs/audits/P01_C_AO_AGY_INTEGRATION_PROOF.md`
27. `docs/audits/P01_C_EXTERNAL_AUDIT.md`

---

## Pinned AO Baseline

| Field | Value |
|---|---|
| **Repository** | `Untrivial-ai/agent-orchestrator` |
| **Version Tag** | `v0.13.0` |
| **Pinned Commit** | `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` |
| **Commit Verification** | PASS — `git rev-parse --verify 15e9ea971f...` returns exact SHA |
| **Local Path** | `D:\\TU_CODE\\Orchestrator\\agent-orchestrator` |
| **Current HEAD** | `e617c48c0b659682745182ec5f344e7fa72ec0cb` (v0.13.1-nightly; NOT used) |
| **Authority Rule** | Pinned commit `15e9ea9...` is the ONLY authoritative source for this audit |

---

## Public Endpoint Matrix

All entries reference pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.

### Health & Daemon Probes

| Supervisor Operation | AO HTTP Method | AO Path | Request DTO | Response Fields | Status Codes | Source File | Source Symbol | Runtime Proof | Classification |
|---|---|---|---|---|---|---|---|---|---|
| `checkHealth` | `GET` | `/healthz` | None | `{status:"ok", service, pid, executablePath, workingDirectory}` | `200` | `backend/internal/httpd/router.go` | `mountHealth` | `RUNTIME_TESTED_PASS` (P01-A) | `NATIVE_UPSTREAM` |
| `checkReadiness` | `GET` | `/readyz` | None | `{status:"ready", service, pid, executablePath, workingDirectory}` | `200` | `backend/internal/httpd/router.go` | `mountHealth` | `RUNTIME_TESTED_PASS` (P01-A) | `NATIVE_UPSTREAM` |

### Project Management

| Supervisor Operation | AO HTTP Method | AO Path | Request DTO | Response Fields | Status Codes | Source File | Source Symbol | Runtime Proof | Classification |
|---|---|---|---|---|---|---|---|---|---|
| `registerProject` | `POST` | `/api/v1/projects` | `{path, projectId?, name?, config?, asWorkspace?, clonePreparationId?}` | `{project:{...}}` | `201`, `400`, `409` | `backend/internal/httpd/controllers/projects.go` | `add` / `AddInput` | `RUNTIME_TESTED_PASS` (P01-A) | `NATIVE_UPSTREAM` |
| `getProject` | `GET` | `/api/v1/projects/{id}` | Path param `id` | `{project:{...}}` | `200`, `404` | `backend/internal/httpd/controllers/projects.go` | `get` | `RUNTIME_TESTED_PASS` (P01-A) | `NATIVE_UPSTREAM` |

### Session Lifecycle

| Supervisor Operation | AO HTTP Method | AO Path | Request DTO | Response Fields | Status Codes | Source File | Source Symbol | Runtime Proof | Classification |
|---|---|---|---|---|---|---|---|---|---|
| `createWorkerSession` | `POST` | `/api/v1/sessions` | `{projectId, kind?, harness?, branch?, mode?, prompt?, model?, displayName?, attachments?}` | `{session:{...}, promptBytes, systemPromptBytes}` | `201`, `400`, `404` | `backend/internal/httpd/controllers/sessions.go` | `spawn` / `SpawnSessionRequest` | `RUNTIME_TESTED_PASS` (P01-A, P01-C) | `NATIVE_UPSTREAM` |
| `getWorkerStatus` | `GET` | `/api/v1/sessions/{sessionId}` | Path param `sessionId` | `{session:{id, activity:{state, lastActivityAt}, status, metadata:{workspacePath, branch}, isTerminated, ...}}` | `200`, `404` | `backend/internal/httpd/controllers/sessions.go` | `get` | `RUNTIME_TESTED_PASS` (P01-A, P01-C) | `NATIVE_UPSTREAM` |
| `dispatchTaskContract` | `POST` | `/api/v1/sessions/{sessionId}/send` | `{message, attachment?}` | `{ok, sessionId, message}` | `200`, `400`, `404` | `backend/internal/httpd/controllers/sessions.go` | `send` / `SendSessionMessageRequest` | `RUNTIME_TESTED_PASS` (P01-C) | `ADAPTER_NORMALIZATION` (contract→message serialization) |
| `stopWorker` | `POST` | `/api/v1/sessions/{sessionId}/kill` | None | `{ok, sessionId, freed}` | `200`, `404` | `backend/internal/httpd/controllers/sessions.go` | `kill` | `RUNTIME_TESTED_PASS` (P01-A) | `NATIVE_UPSTREAM` |
| `resumeWorker` | `POST` | `/api/v1/sessions/{sessionId}/restore` | None | `{ok, sessionId, restoreMode, session:{...}}` | `200`, `404`, `409` | `backend/internal/httpd/controllers/sessions.go` | `restore` | `RUNTIME_TESTED_PASS` (P01-C) | `NATIVE_UPSTREAM` |

### Workspace & Report Retrieval

| Supervisor Operation | AO HTTP Method | AO Path | Request DTO | Response Fields | Status Codes | Source File | Source Symbol | Runtime Proof | Classification |
|---|---|---|---|---|---|---|---|---|---|
| `getWorkspaceFile / report retrieval` | `GET` | `/api/v1/sessions/{sessionId}/workspace/file?path=` | Query param `path` (relative), optional `section`, optional `commitSha` | `{file:{...}, content}` | `200`, `400`, `404` | `backend/internal/httpd/controllers/sessions.go` | `getWorkspaceFile` | `RUNTIME_TESTED_PASS` (P01-C WorkerReport fetch) | `NATIVE_UPSTREAM` |
| `getWorktreePath` | `GET` | `/api/v1/sessions/{sessionId}` | Path param `sessionId` | `session.metadata.workspacePath` (within Session response) | `200`, `404` | `backend/internal/domain/session.go` | `SessionMetadata.WorkspacePath` | `RUNTIME_TESTED_PASS` (P01-C) | `ADAPTER_NORMALIZATION` (field extraction from existing endpoint) |
| `getActiveBranch` | `GET` | `/api/v1/sessions/{sessionId}` | Path param `sessionId` | `session.metadata.branch` (within Session response) | `200`, `404` | `backend/internal/domain/session.go` | `SessionMetadata.Branch` | `RUNTIME_TESTED_PASS` (P01-C) | `ADAPTER_NORMALIZATION` (field extraction from existing endpoint) |

### Event Streaming

| Supervisor Operation | AO HTTP Method | AO Path | Description | Source File | Source Symbol | Runtime Proof | Classification |
|---|---|---|---|---|---|---|---|
| `subscribeEvents` (CDC) | `GET` | `/api/v1/events` | Global change-data-capture SSE stream; emits `session_created`, `session_updated`, etc. | `backend/internal/httpd/events.go` | `EventsController.stream` / `cdc.Event` | `UNPROVEN` (not used in P01) | `ADAPTER_NORMALIZATION` (event type mapping required) |
| `subscribeEvents` (workspace) | `GET` | `/api/v1/sessions/{sessionId}/workspace/events` | Session-scoped SSE stream of FILESYSTEM CHANGE events (NOT worker lifecycle events) | `backend/internal/httpd/controllers/sessions.go` | `streamWorkspaceChanges` | `RUNTIME_TESTED_PASS` (P01-A) | `NOT_LIFECYCLE` — workspace file change stream only |

---

## AO Request / Response Constraints

Literal upstream constraint values from pinned commit `15e9ea971f...`:

| Constraint | Upstream Value | Source File | Source Symbol | Classification |
|---|---|---|---|---|
| `prompt` max length | `16384` bytes (16 KiB) | `backend/internal/httpd/controllers/sessions.go` | `maxPromptLen = 16 << 10` | `UPSTREAM_LIMIT` |
| `message` max length | `4096` bytes | `backend/internal/httpd/controllers/sessions.go` | `maxMessageLen = 4096` | `UPSTREAM_LIMIT` |
| `model` field max length | `256` characters | `backend/internal/httpd/controllers/sessions.go` | `maxModelLen = 256` | `UPSTREAM_LIMIT` |
| `displayName` max length | `20` characters (Unicode rune count) | `backend/internal/httpd/controllers/sessions.go` | `maxDisplayNameLen = 20` | `UPSTREAM_LIMIT` |
| Max spawn body | `maxAttachmentsBytes * 4/3 + 2MiB` (~35 MiB) | `backend/internal/httpd/controllers/sessions.go` | `maxSpawnBodyBytes` | `UPSTREAM_LIMIT` |
| Max attachments per spawn | `8` | `backend/internal/httpd/controllers/sessions.go` | `maxAttachments = 8` | `UPSTREAM_LIMIT` |
| Max attachment size | `10 MiB` decoded | `backend/internal/httpd/controllers/sessions.go` | `maxAttachmentBytes = attachmentstore.MaxFileBytes` | `UPSTREAM_LIMIT` |
| Max attachments total decoded | `25 MiB` | `backend/internal/httpd/controllers/sessions.go` | `maxAttachmentsBytes = 25 << 20` | `UPSTREAM_LIMIT` |
| `harness` allowed values | `claude-code, codex, aider, opencode, grok, droid, amp, agy, crush, cursor, qwen, copilot, goose, auggie, continue, devin, cline, kimi, muse, kiro, kilocode, vibe, pi, kimchi, omp, prime-agent, autohand` | `backend/internal/httpd/controllers/dto.go` | `SpawnSessionRequest.Harness` | `UPSTREAM_LIMIT` |
| `mode` allowed values | `chat, tui` | `backend/internal/httpd/controllers/dto.go` | `SpawnSessionRequest.Mode` | `UPSTREAM_LIMIT` |
| Session kind allowed values | `worker, orchestrator` | `backend/internal/domain/session.go` | `KindWorker, KindOrchestrator` | `UPSTREAM_LIMIT` |
| SSE keepalive interval (workspace events) | `15 seconds` | `backend/internal/httpd/controllers/sessions.go` | `time.NewTicker(15 * time.Second)` | `UPSTREAM_LIMIT` |
| SSE heartbeat interval (CDC events) | `10 seconds` | `backend/internal/httpd/events.go` | `eventsHeartbeatInterval = 10 * time.Second` | `UPSTREAM_LIMIT` |
| Default request timeout | Configured via `config.DefaultRequestTimeout` | `backend/internal/httpd/api.go` | `API.Register` timeout middleware | `UNRESOLVED_POLICY` — exact value not read in this audit |
| DNS rebinding / CORS | Loopback-only (`127.0.0.1`, `::1`, `localhost`) for control endpoints | `backend/internal/httpd/router.go` | `localControlRequest` | `UPSTREAM_LIMIT` |

---

## Session Activity Semantics

Canonical activity state values from `backend/internal/domain/activity.go` (pinned commit `15e9ea9...`):

| AO ActivityState | JSON Value | Meaning | Sticky? | Source |
|---|---|---|---|---|
| `ActivityActive` | `"active"` | Agent is executing (pre-invocation or post-tool-use hook fired) | No | `backend/internal/domain/activity.go` |
| `ActivityIdle` | `"idle"` | Agent turn is complete (Stop hook fired) | No | `backend/internal/domain/activity.go` |
| `ActivityWaitingInput` | `"waiting_input"` | Agent paused at empty prompt, awaiting next instruction | **YES** | `backend/internal/domain/activity.go` |
| `ActivityBlocked` | `"blocked"` | Agent stopped on pending tool-permission or approval dialog | **YES** | `backend/internal/domain/activity.go` |
| `ActivityExited` | `"exited"` | Agent process has exited | No | `backend/internal/domain/activity.go` |

**Activity signal derivation** (from `backend/internal/adapters/agent/agy/activity.go`):
- Agy hook `pre-invocation` → `ActivityActive`
- Agy hook `post-tool-use` → `ActivityActive`
- Agy hook `stop` → `ActivityIdle`

**`lastActivityAt`** field: Records when the activity state was last observed. This is the timestamp of the last received hook signal. It is **NOT** a heartbeat. There is no autonomous periodic signal from AO or Agy driving `lastActivityAt`; it is updated only when a hook fires. Calling it a "heartbeat" would be semantically incorrect.

### AOAdapter Normalized Supervisor Events (Proposed)

| Supervisor Domain Event | AO Observable Signal | Mapping Mechanism | Certainty |
|---|---|---|---|
| `STARTED` | Session `activity.state == "active"` AND `isTerminated == false` observed for first time | Poll `GET /api/v1/sessions/{id}` after spawn; detect first `active` state | `ADAPTER_NORMALIZATION` |
| `ACTIVE` | `activity.state == "active"` | Direct mapping from GET response | `ADAPTER_NORMALIZATION` |
| `IDLE / TURN_COMPLETE` | `activity.state == "idle"` | Direct mapping; this is the canonical turn-completion signal per ADR-011 | `ADAPTER_NORMALIZATION` |
| `STOPPED` | `isTerminated == true` | Direct mapping from GET response | `ADAPTER_NORMALIZATION` |
| `CRASHED` | Session `activity.state == "exited"` AND unexpected termination (not via kill/restore) | Requires context from prior state; heuristic classification | `ADAPTER_NORMALIZATION` (needs policy decision) |
| `UNKNOWN / UNAVAILABLE` | HTTP error (503, timeout, connection refused) on GET | AOAdapter maps transport errors to domain-typed errors | `ADAPTER_NORMALIZATION` |

---

## Lifecycle Event Transport

### Finding: P03_LIFECYCLE_EVENT_TRANSPORT Classification

**Evidence**:
1. `GET /api/v1/sessions/{sessionId}/workspace/events` (SSE) — confirmed from `backend/internal/httpd/controllers/sessions.go:222` / `streamWorkspaceChanges` — is a **filesystem change notification stream**. It emits when worktree files change, NOT when the worker starts, stops, or crashes. This is NOT a lifecycle event transport.
2. `GET /api/v1/events` (SSE) — confirmed from `backend/internal/httpd/events.go` / `EventsController.stream` / `cdc.Event` — is the global change-data-capture stream emitting `session_created`, `session_updated`, etc. The `session_updated` event fires when the session row changes (including when `activity` or `isTerminated` changes). This IS a potential lifecycle observation channel but requires:
   - Polling or subscribing to the CDC stream
   - Parsing `session_updated` events to detect activity state transitions
   - This is **adapter normalization**, not a native worker lifecycle stream
3. `POST /api/v1/sessions/{sessionId}/activity` — confirmed from `backend/internal/httpd/controllers/sessions.go:1569` — is an **inbound signal endpoint** used by the Agy hook callbacks to report activity state TO AO. It is a signal receiver, not a signal source for the Supervisor.
4. P01-C proved turn completion via: **Agy Stop hook → activity state transitions to `idle` → Supervisor polls `GET /api/v1/sessions/{id}` → observes `activity.state == "idle"`**. There is no native push stream of worker lifecycle events.
5. `docs/phases/P03_AO_INTEGRATION.md` requires `worker_started`, `worker_heartbeat`, `worker_finished` events.
6. `docs/15_OBSERVABILITY.md` defines canonical events `worker.started`, `worker.stopped`, `worker.crashed` — no `worker_heartbeat` or `worker_finished` defined.
7. No native AO stream publishes these events directly. They must be derived by the AOAdapter from activity state polling or CDC event subscription.

**WORKER_HEARTBEAT**: The `lastActivityAt` field is a timestamp of the last hook-triggered activity signal. It is NOT a periodic heartbeat. There is no autonomous periodic signal. Therefore:

```text
WORKER_HEARTBEAT = UNPROVEN
```

No periodic heartbeat signal exists in pinned AO v0.13.0. Fabricating heartbeat events from polling intervals would invent semantics not supported by upstream.

**Classification**:

```text
P03_LIFECYCLE_EVENT_TRANSPORT = SUPERVISOR_NORMALIZATION_POSSIBLE
```

**Rationale**: The AO public API (`GET /api/v1/sessions/{id}`) exposes sufficient status and activity information for the AOAdapter to produce normalized Supervisor domain events (`worker.started`, `worker.stopped`, `worker.crashed`) via bounded polling. The CDC stream (`GET /api/v1/events`) may additionally be used for push-based `session_updated` observation. No native lifecycle stream exists; no upstream patch is required; normalization is sufficient.

---

## Observability Contradiction Matrix

| Document 1 Claim | Document 2 Claim | Contradiction Type | Resolution |
|---|---|---|---|
| `docs/phases/P03_AO_INTEGRATION.md`: requires `worker_started` event | `docs/15_OBSERVABILITY.md`: defines canonical event `worker.started` (with dot) | `WORDING_ONLY` | Align to canonical `worker.started` naming; P03 phase spec uses underscore variant (pre-canonical shorthand). No architecture change needed. |
| `docs/phases/P03_AO_INTEGRATION.md`: requires `worker_finished` event | `docs/15_OBSERVABILITY.md`: defines canonical event `worker.stopped` (not `worker_finished`) | `WORDING_ONLY` | `worker_finished` is non-canonical. Canonical name is `worker.stopped`. Adapter must emit `worker.stopped` (on `isTerminated==true` detection) and optionally `worker.crashed` (on exited+unexpected). |
| `docs/phases/P03_AO_INTEGRATION.md`: requires `worker_heartbeat` event | `docs/15_OBSERVABILITY.md`: does NOT define `worker_heartbeat`; no periodic heartbeat signal exists in AO | `REQUIREMENT_GAP` | `WORKER_HEARTBEAT = UNPROVEN`. No upstream signal. If heartbeat semantics are needed, must be defined as an AOAdapter-generated synthetic event from polling. Requires explicit policy decision (ADR or Task Contract parameter). |
| `docs/12_UPSTREAM_INTEGRATION.md`: abstract `subscribeEvents(sessionId, onEvent)` | AO pinned source: workspace events stream is filesystem changes only; CDC stream is global | `ADAPTER_NORMALIZATION` | The abstract `subscribeEvents` contract must be implemented as: (1) bounded polling of `GET /api/v1/sessions/{id}` for activity state transitions, optionally supplemented by (2) CDC SSE subscription filtered to `session_updated` events. |
| Canonical `getWorktreePath` operation (IAOAdapter) | AO: no dedicated endpoint; `workspacePath` is a field in `GET /api/v1/sessions/{id}` response | `ADAPTER_NORMALIZATION` | AOAdapter extracts from existing endpoint. No new AO endpoint required. |
| Canonical `getActiveBranch` operation (IAOAdapter) | AO: no dedicated endpoint; `branch` is in `SessionMetadata` within `GET /api/v1/sessions/{id}` | `ADAPTER_NORMALIZATION` | AOAdapter extracts from existing endpoint. No new AO endpoint required. |

---

## ADR-011 Worker Completion Mapping

Canonical ADR-011 worker-turn completion contract (unchanged; verified from `docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md` and P01-C evidence):

```text
Agy Stop Hook fires
  → AO records activity.state = "idle" (via POST /api/v1/sessions/{id}/activity)
  → AOAdapter polls GET /api/v1/sessions/{id}
  → Detects activity.state == "idle"
  → Performs bounded fetch of report
  → GET /api/v1/sessions/{id}/workspace/file?path=.supervisor/reports/<task_id>/<attempt_id>.json
  → JSON parse
  → Schema validation (zero-trust)
  → task_id / attempt_id identity match
  → StateStore: REPORT_READY transition
```

**Explicitly prohibited (ADR-011)**:
- `/mux` terminal stream scraping
- Direct Agy stdout capture
- AO internal SQLite access
- Internal AO Go package coupling
- GUI automation
- Fixed `.supervisor/worker-report.json` production path (non-attempt-scoped)

**Canonical report path** (ADR-011 / ADR-012):
```text
.supervisor/reports/<task_id>/<attempt_id>.json
```

**Runtime proof status**: `P01C_COMPLETION_SIGNAL = PASS` (Stop hook → idle transition proven). `P01C_REPORT_RETRIEVAL = PASS` (workspace file fetch via public API proven). `P01C_REPORT_CONTRACT_VALID = FAIL` (schema validation failed on P01-C output — reinforces zero-trust rule; does NOT invalidate the transport mechanism).

---

## ADR-012 Pre-Dispatch Ordering

Required ordering for P03 external calls (consuming P02 durable state):

```text
1. TaskContract validated (P02 internal)
2. TaskAttempt allocated in StateStore (P02 internal — PrepareDispatch)
3. READY → DISPATCHED committed atomically (P02 internal — PrepareDispatch)
4. ONLY THEN: AOAdapter external side effects begin:
   - createWorkerSession (POST /api/v1/sessions)
   - dispatchTaskContract (POST /api/v1/sessions/{id}/send)
```

**P03 AOAdapter constraints from ADR-012**:
- Must NOT allocate attempts itself (owned by P02 `PrepareDispatch`)
- Must NOT mutate immutable TaskContracts
- Must NOT dispatch while task is merely READY (must receive DISPATCHED state from P02)
- Must NOT silently change `base_sha`
- Must NOT reuse historical `attempt_id`

**AOAdapter boundary**: P03 receives the durable `TaskAttempt` (including `attempt_id`, `task_id`, worktree hints) from P02 and translates it to AO API calls. It does NOT own state persistence.

---

## WorkerReport Retrieval Contract

Canonical report retrieval per ADR-011 and P01-C evidence:

| Step | Operation | AO Endpoint | Source Evidence |
|---|---|---|---|
| 1. Detect turn completion | Poll session status | `GET /api/v1/sessions/{sessionId}` | P01-C: `activity.state == "idle"` |
| 2. Fetch report file | Workspace file GET | `GET /api/v1/sessions/{sessionId}/workspace/file?path=.supervisor/reports/{task_id}/{attempt_id}.json` | P01-C, ADR-011 |
| 3. Parse JSON | AOAdapter parses body | N/A | ADR-011: zero-trust JSON parse |
| 4. Schema validate | AOAdapter validates against canonical schema | N/A | ADR-011: zero-trust schema validation |
| 5. Identity match | Verify `task_id` and `attempt_id` match | N/A | ADR-011: identity match required |
| 6. Transition StateStore | REPORT_READY | P02 StateStore | ADR-012 |

**Report path format**: `.supervisor/reports/{task_id}/{attempt_id}.json`
**Encoding**: JSON (raw body from AO workspace file API)
**Validation**: Zero-trust — report is treated as untrusted worker output regardless of source

---

## Error Mapping

Proposed Go error types/sentinels for the future AOAdapter coding task (design evidence only — NOT implemented):

| AO Condition | HTTP Status / AO Code | Proposed Go Sentinel | Notes |
|---|---|---|---|
| Daemon unavailable (not running) | Connection refused | `ErrAODaemonUnavailable` | Transport-level; pre-HTTP |
| Request timeout | Context deadline exceeded | `ErrAORequestTimeout` | Wrap context error |
| Context cancellation | Context cancelled | Standard `context.Canceled` | Pass through |
| Project not found | `404 / not_found` | `ErrAOProjectNotFound` | AO code: `PROJECT_NOT_FOUND` or `not_found` |
| Session not found | `404 / not_found` | `ErrAOSessionNotFound` | AO code: `SESSION_NOT_FOUND` or `not_found` |
| Invalid request | `400 / bad_request` | `ErrAOInvalidRequest` | Wrap AO code field |
| Session spawn failure | `400` or `500` | `ErrAOSpawnFailure` | Various codes from spawn handler |
| Send failure | `400` or `404` | `ErrAOSendFailure` | Session not found or invalid message |
| Restore failure | `404` or `409` | `ErrAORestoreFailure` | Session not found or conflict |
| Kill failure | `404` or `500` | `ErrAOKillFailure` | Session not found or teardown error |
| Unexpected upstream response | `5xx` generic | `ErrAOUnexpectedResponse` | AO code: `INTERNAL_ERROR` |
| Malformed JSON response | Parse error on valid HTTP 200 | `ErrAOMalformedResponse` | Defensive; should not occur |
| Upstream version incompatibility | N/A | `ErrAOVersionMismatch` | Detected via health probe version field |
| Service unavailable (transient) | `503 / SERVICE_UNAVAILABLE` | `ErrAOServiceUnavailable` | Retryable per envelope semantics |

Error envelope shape (from `backend/internal/httpd/envelope/envelope.go`):
```json
{
  "error": "not_found",
  "code": "SESSION_NOT_FOUND",
  "message": "Unknown session",
  "requestId": "..."
}
```

---

## Timeout / Retry Policy Gaps

| Policy Area | Status | Notes |
|---|---|---|
| Health probe timeout | `UNRESOLVED_POLICY` | No canonical value specified in docs. Must be specified in P03 Task Contract. |
| Session spawn timeout | `UNRESOLVED_POLICY` | AO request timeout is daemon-configured; Supervisor-side upper bound not specified. |
| `dispatchTaskContract` (send) timeout | `UNRESOLVED_POLICY` | AO side uses `config.DefaultRequestTimeout`; Supervisor-side not specified. |
| Activity poll interval | `UNRESOLVED_POLICY` | P01-C used ad-hoc polling interval (2s–10s). No canonical value specified. |
| Bounded IDLE detection timeout | `UNRESOLVED_POLICY` | How long after Stop hook should Supervisor wait before classifying as TIMEOUT? Not specified. |
| Maximum report-fetch duration | `UNRESOLVED_POLICY` | Maximum time allowed for workspace file fetch after idle detection. Not specified. |
| Backoff policy | `UNRESOLVED_POLICY` | No canonical backoff strategy defined for retryable errors (503). |
| HTTP transport timeout | `UNRESOLVED_POLICY` | No Go HTTP client timeout value specified. |
| Kill/stop timeout | `UNRESOLVED_POLICY` | No timeout for kill operation specified. |

All `UNRESOLVED_POLICY` items must be resolved before P03 implementation. Resolution may come via:
- An ADR (if architectural in nature)
- A concrete P03 Task Contract (operational parameters)
- Configuration values in the Supervisor (not hard-coded in AOAdapter)

---

## Anti-Corruption Boundary

Design evidence only — NOT implemented:

### Proposed Layer Taxonomy

| Concept | Layer | Description |
|---|---|---|
| `SpawnSessionRequest` (AO DTO) | AO Transport DTO | Raw JSON shape sent to `POST /api/v1/sessions`. Owned by AOAdapter, never exposed to domain. |
| `SessionRecord` / `Session` (AO DTO) | AO Transport DTO | JSON shape returned by `GET /api/v1/sessions/{id}`. AOAdapter reads and discards; extracts normalized result. |
| `SendSessionMessageRequest` (AO DTO) | AO Transport DTO | JSON shape for `POST /api/v1/sessions/{id}/send`. Serialized by AOAdapter from domain inputs. |
| `AOWorkerStatus` (proposed) | AOAdapter Normalized Result | Extracts `{sessionId, activityState, isTerminated, workspacePath, branch, lastActivityAt}` from AO session response. Never persisted in domain. |
| `AODispatchResult` (proposed) | AOAdapter Normalized Result | Wraps success/failure of spawn+send sequence with the new AO `sessionId`. |
| `Task`, `TaskAttempt`, `TaskContract` | Domain Entity (P02) | Owned by `internal/domain`. AOAdapter receives these as inputs; never mutates them. |
| `AuditEvent` | Domain Entity (P02) | AOAdapter emits audit events via the P02 audit store interface; does NOT write directly to SQLite. |

### Boundary Invariants

1. AO DTOs (`SpawnSessionRequest`, `Session`, etc.) MUST NOT appear in `internal/domain`, `internal/workflow`, `internal/contract`, or `internal/store`.
2. AOAdapter MUST accept domain types as inputs and return normalized results.
3. No new domain entity is introduced in P03 unless proven necessary by a governance process.
4. All AO transport types live in the future `internal/ao` package (or equivalent adapter package).

---

## Anti-Reinvention Check

| Forbidden Reinvention | Source Registry Check | ADR Check | Status |
|---|---|---|---|
| Custom daemon / AO daemon replacement | `SOURCE_REGISTRY.md`: AO is upstream | ADR-002 | `CONFIRMED_NO_REINVENTION` |
| Custom ConPTY wrapper | `SOURCE_REGISTRY.md`: AO owns ConPTY | ADR-004 | `CONFIRMED_NO_REINVENTION` |
| Custom Git worktree creation | `SOURCE_REGISTRY.md`: AO owns worktrees | ADR-002 | `CONFIRMED_NO_REINVENTION` |
| Custom Agy process launcher | `SOURCE_REGISTRY.md`: AO owns Agy launch | ADR-003, ADR-011 | `CONFIRMED_NO_REINVENTION` |
| Direct Agy CLI invocation (by Supervisor) | `SOURCE_REGISTRY.md` + ADR-011 | ADR-011: Supervisor does NOT invoke Agy directly | `CONFIRMED_NO_REINVENTION` |
| Terminal emulator parser (`/mux` scraping) | ADR-011, ADR-004 | Prohibited | `CONFIRMED_NO_REINVENTION` |
| AO internal DB reader | ADR-002, ADR-009 | Prohibited — only loopback REST | `CONFIRMED_NO_REINVENTION` |
| AO source patch | ADR-009 | Upstream-over-reimplementation: `AO_UPSTREAM_PATCH_REQUIRED = NO` per P01-C | `CONFIRMED_NO_REINVENTION` |
| Custom worker process manager | ADR-002 | AO owns process management | `CONFIRMED_NO_REINVENTION` |

---

## Proposed P03 Task Decomposition

Based ONLY on verified public surfaces. These are proposals, not pre-approved tasks.

### TASK-P03-001: AO Transport Contract — Health, Project & Session Read Model
**Objective**: Implement the minimal AOAdapter read-only operations: health/readiness probes, project registration, project inspection, session status polling, and normalized `AOWorkerStatus` extraction.

**Allowed Scope**:
- New `internal/ao/` (or `internal/adapter/ao/`) package
- `AOAdapter` struct implementing health + project + getWorkerStatus operations
- Go HTTP client (loopback only, no TLS)
- Normalized result types (`AOWorkerStatus`, etc.)

**Forbidden Scope**:
- No session spawn/send/kill in this task
- No lifecycle normalization logic
- No polling loop
- No P02 StateStore mutations

**Acceptance Criteria**:
- `checkHealth` returns valid response for `GET /healthz`
- `checkReadiness` returns valid response for `GET /readyz`
- `registerProject` correctly registers and verifies via `getProject`
- `getWorkerStatus` correctly maps AO session response to `AOWorkerStatus`
- Unit tests with AO stub/mock

**Upstream Endpoints**: `GET /healthz`, `GET /readyz`, `POST /api/v1/projects`, `GET /api/v1/projects/{id}`, `GET /api/v1/sessions/{id}`

**Dependency**: None (first P03 task)

---

### TASK-P03-002: Session Spawn / Send / Kill / Restore Commands
**Objective**: Implement the stateful session lifecycle commands: spawn, send, kill, and restore. These commands consume P02 dispatch state (`TaskAttempt`) and emit to AO.

**Allowed Scope**:
- `createWorkerSession`, `dispatchTaskContract`, `stopWorker`, `resumeWorker` methods on AOAdapter
- Contract → prompt serialization (adapter normalization)
- Error classification mapping to Go sentinels

**Forbidden Scope**:
- No polling loop
- No activity observation
- No report fetch
- No P02 StateStore mutations directly (receive attempt from P02)

**Acceptance Criteria**:
- Spawn creates a session with correct harness, project, and prompt
- Send delivers serialized contract to existing session
- Kill terminates session cleanly
- Restore re-activates existing session
- ADR-012 pre-dispatch ordering is enforced (attempt ID validated before any AO call)
- Unit tests with AO stub/mock

**Upstream Endpoints**: `POST /api/v1/sessions`, `POST /api/v1/sessions/{id}/send`, `POST /api/v1/sessions/{id}/kill`, `POST /api/v1/sessions/{id}/restore`

**Dependency**: TASK-P03-001

---

### TASK-P03-003: Activity Observation & Turn Completion Normalization
**Objective**: Implement the bounded polling loop observing `GET /api/v1/sessions/{id}`, detecting `activity.state == "idle"`, and producing normalized Supervisor domain events (`worker.started`, `worker.stopped`, `worker.crashed`). Resolve the `WORKER_HEARTBEAT = UNPROVEN` gap before implementation.

**Allowed Scope**:
- Bounded activity polling loop (interval and max-duration from Task Contract parameters)
- Activity state → Supervisor domain event mapping
- `worker.started`, `worker.stopped`, `worker.crashed` event emission via P02 audit interface
- Turn completion detection (`idle` state detection per ADR-011)

**Forbidden Scope**:
- No report fetch (owned by TASK-P03-004)
- No `/mux` scraping
- No `worker_heartbeat` synthetic event unless explicitly authorized by policy decision

**Open Decision Required Before Coding**:
- Poll interval value
- Maximum bounded-wait duration for IDLE
- Whether to subscribe CDC SSE or pure polling
- Whether synthetic heartbeat events are authorized and at what interval

**Acceptance Criteria**:
- Correct event emission on activity state transitions
- Bounded polling terminates within the specified maximum duration
- Policy timeout values configurable, not hard-coded

**Upstream Endpoints**: `GET /api/v1/sessions/{id}` (polling), optionally `GET /api/v1/events` (CDC SSE)

**Dependency**: TASK-P03-001, TASK-P03-002

---

### TASK-P03-004: Attempt-Scoped WorkerReport Retrieval / Handoff Integration
**Objective**: Implement the WorkerReport fetch after turn completion detection, with zero-trust JSON parse, schema validation, identity match, and StateStore REPORT_READY transition.

**Allowed Scope**:
- `getWorkspaceFile` call for report path `.supervisor/reports/{task_id}/{attempt_id}.json`
- Zero-trust JSON parse and schema validation
- `task_id` / `attempt_id` identity match
- P02 StateStore REPORT_READY transition (via existing P02 interface)

**Forbidden Scope**:
- No fixed non-attempt-scoped report path
- No schema bypasses
- No unvalidated worker output trusted

**Acceptance Criteria**:
- Report correctly fetched from attempt-scoped path
- Invalid JSON fails closed
- Schema validation failure fails closed
- Identity mismatch fails closed
- Correct StateStore transition on success

**Upstream Endpoints**: `GET /api/v1/sessions/{id}/workspace/file?path=`

**Dependency**: TASK-P03-001, TASK-P03-003

---

### TASK-P03-005: Real AO Integration Test / P03 Exit Gate
**Objective**: Execute end-to-end integration test against a live AO v0.13.0 daemon to verify the complete P03 adapter stack: spawn, send, observe, report fetch, and teardown.

**Allowed Scope**:
- Integration test binary (separate from unit tests)
- Local AO daemon required to run

**Forbidden Scope**:
- No changes to production adapter code during this task
- No AO version upgrades

**Acceptance Criteria**:
- All P03 operations proven against live AO daemon
- WorkerReport retrieved and validated
- REPORT_READY transition confirmed in StateStore
- Race detector clean

**Dependency**: All prior P03 tasks

---

## Open Decisions

| Decision | Classification | Notes |
|---|---|---|
| `WORKER_HEARTBEAT` semantics | `REQUIREMENT_GAP` | No upstream signal. Must decide: (a) remove from requirements, or (b) define as synthetic Supervisor-generated polling event. Requires explicit policy decision. |
| Activity poll interval value | `UNRESOLVED_POLICY` | No canonical value. Should be a Task Contract configurable parameter, not hard-coded. |
| Maximum bounded IDLE wait duration | `UNRESOLVED_POLICY` | Upper bound for turn completion polling. Must be specified. |
| HTTP transport timeout value | `UNRESOLVED_POLICY` | Go HTTP client timeout. Must be specified. |
| CDC SSE subscription vs pure polling | `UNRESOLVED_POLICY` | Both are viable. CDC subscription is more efficient; polling is simpler. Policy decision needed. |
| Synthetic `worker_heartbeat` authorization | `REQUIREMENT_GAP` | If heartbeat semantics required by FR-006 or FR-015, must be explicitly defined as adapter-generated synthetic event with a declared polling-based basis. |
| `config.DefaultRequestTimeout` value | `UNRESOLVED_POLICY` | Exact AO default request timeout not read in this audit. Should be verified against AO config package. |

---

## Findings

### Summary

1. **P02 foundation is sound and complete.** All P03 external operations map cleanly to proven, stable AO public endpoints at pinned v0.13.0.
2. **Lifecycle event transport is viable via normalization.** No native lifecycle event stream exists, but the AO `GET /api/v1/sessions/{id}` response provides sufficient observable signals for all required Supervisor events except heartbeat.
3. **Worker heartbeat is unproven and cannot be fabricated.** The `WORKER_HEARTBEAT = UNPROVEN` finding requires an explicit governance decision before implementation.
4. **Observability naming contradictions are wording-only except heartbeat.** The `worker_heartbeat` requirement in P03 phase spec has no canonical upstream support and no canonical Observability doc definition. This is a `REQUIREMENT_GAP`.
5. **ADR-011 turn completion contract is fully intact.** The proven P01-C mechanism (Stop hook → idle → bounded report fetch via workspace file API) is the canonical and sufficient worker completion signal.
6. **ADR-012 pre-dispatch ordering is enforceable.** P02 PrepareDispatch produces the atomic state that P03 AOAdapter consumes. No ordering conflict found.
7. **Anti-corruption boundary is well-defined.** AO DTOs can be fully isolated in the adapter layer with no new domain entities required.
8. **Anti-reinvention verified.** No forbidden reinvention identified.
9. **Eight `UNRESOLVED_POLICY` items** must be resolved before P03 coding tasks are released.
10. **P03 task decomposition into 5 tasks is viable** and matches verified public surface scope.

### Blocking Findings
- **None that block the audit itself.** The audit can proceed to READY_FOR_EXTERNAL_AUDIT.
- The `WORKER_HEARTBEAT = UNPROVEN` gap and 8 `UNRESOLVED_POLICY` items must be resolved before any concrete P03 Task Contract is released for implementation.

---

## Gate Verdict

```text
P03_PRECODE_AUDIT = READY_FOR_EXTERNAL_AUDIT
P03_LIFECYCLE_EVENT_TRANSPORT = SUPERVISOR_NORMALIZATION_POSSIBLE
P03_ARCHITECTURE_CHANGE = NO
P03_ADR_REQUIRED = NO
WORKER_HEARTBEAT = UNPROVEN
P03_CODE = HELD_PENDING_EXTERNAL_RELEASE
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_PRECODE_AUDIT
```

**Rationale for NO ADR required**: The lifecycle event normalization approach is not a new architectural concept; it is an adapter-layer implementation detail within the existing AOAdapter design boundary. The existing ADR-011 (worker report handoff), ADR-002 (AO as execution control plane), and ADR-009 (upstream over reimplementation) already govern this. The heartbeat gap is a requirement clarification, not an architecture change.

**Rationale for READY_FOR_EXTERNAL_AUDIT**: All public AO endpoints required by P03 are mapped to literal source evidence at the pinned commit. Lifecycle transport is classified with evidence. Contradictions are documented and classified. No blocking implementation defects found. No production code was written.
