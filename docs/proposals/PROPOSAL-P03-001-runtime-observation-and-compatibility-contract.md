# PROPOSAL-P03-001: Upstream Runtime Observation, Compatibility Contract, and Interface Reconciliation

> **Proposal ID**: `PROPOSAL-P03-001`
> **Date**: 2026-09-22
> **Author**: AI Engineering Supervisor Worker
> **Target Phase**: P03 (Agent Orchestrator Integration)
> **Status**: `RECOMMENDED_FOR_EXTERNAL_APPROVAL`
> **Decision Priority**: Level 3 Proposal under `docs/24_CHANGE_GOVERNANCE.md`
> **Governing Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)

---

# Problem Statement

During the Phase P03 pre-code upstream contract audit against the pinned Agent Orchestrator (AO) v0.13.0 authority (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`), rigorous verification of upstream source revealed four critical contract mismatches and documentation gaps between canonical specifications and upstream literal capabilities:

1. **FR-006 Worker Heartbeat Mismatch**: Canonical requirement FR-006 literally requires the Supervisor to "monitor worker process health, heartbeats, and status events via AO public API". Upstream AO v0.13.0 provides session status, activity state, and activity timestamps, but provides **zero native worker heartbeat events**. The 10-second ticker in `backend/internal/httpd/events.go` (`eventsHeartbeatInterval`) emits an empty SSE comment line (`: \n\n`) intended strictly as a transport keepalive for HTTP intermediaries, not a worker process heartbeat. Proceeding with literal `worker.heartbeat` event synthesis without governance approval violates the anti-reinvention principle and produces unverified synthetic lifecycle telemetry.
2. **FR-015 Runtime Compatibility Surface Gap**: Canonical requirement FR-015 requires verifying "daemon status, version, worker harness availability". The prior audit claimed version mismatch is detected via a health probe version field. Literal source inspection of `daemonProbePayload()` in `backend/internal/httpd/router.go` proves that neither `/healthz` nor `/readyz` exposes an AO version field. Pinned version compatibility is established out-of-band by deployment automation and/or verifiable API schema fingerprinting (`/api/v1/openapi.yaml`), rather than a runtime health probe field.
3. **Canonical AOAdapter Interface Omission**: ADR-011 establishes that worker report artifacts (`worker-report.json`) must be retrieved through AO's workspace file read endpoint (`GET /api/v1/sessions/{id}/workspace/file?path=...`). However, the canonical `IAOAdapter` interface in `docs/12_UPSTREAM_INTEGRATION.md` contains no workspace file retrieval operation, creating a formal interface deficiency.
4. **REC-005 Read-Only Policy Contradiction**: `docs/14_FAILURE_RECOVERY.md` REC-005 currently prescribes that AOAdapter should "Stash or clean untracked artifacts before initializing new task attempt". This instruction directly contradicts the Supervisor's fundamental architectural constraint as a **read-only control plane** that never modifies the target repository, and violates AO's exclusive ownership of Git worktrees.
5. **Phase Task Boundary Alignment**: The initial P03 task decomposition leaked Phase P04 evidence collection responsibilities (WorkerReport JSON parsing, schema validation, WorkerClaim creation, and `REPORT_READY` transition) into P03. Canonical traceability in `docs/21_TRACEABILITY_MATRIX.md` strictly assigns FR-007 to EvidenceCollector in P04.

Under `docs/24_CHANGE_GOVERNANCE.md`, no worker may silently amend canonical requirements or canonical architecture documents. This proposal establishes the formal evaluation and proposed resolutions for external supervisor approval prior to any canonical document modification or production code release.

---

# Authority Conflict Matrix

Resolving the observed gaps requires applying the strict 9-level decision hierarchy established in `docs/24_CHANGE_GOVERNANCE.md`:

| Level | Authority Source | Precedence Status in this Conflict | Conflict Resolution Principle |
|---|---|---|---|
| **1** | Confirmed User Requirement | Supreme Authority | User requires robust, verifiable supervision without false green flags or silent failures. |
| **2** | Approved ADRs (`ADR-002`, `ADR-009`, `ADR-011`, `ADR-012`) | High Architectural Authority | `ADR-002` establishes AO as execution control plane. `ADR-009` mandates upstream reuse over reimplementation (prohibits reinvention of worker daemon wrappers). `ADR-011` mandates report retrieval through `/workspace/file` and forbids direct `agy` invocation in P03. `ADR-012` governs attempt identity and pre-dispatch preconditions. |
| **3** | Canonical Architecture (`docs/04_ARCHITECTURE.md`) | System Boundary Baseline | Supervisor is a read-only control plane. Zero direct modification of target repository worktrees. |
| **4** | Requirement Specification (`docs/02_REQUIREMENTS.md`) | Functional Specification Baseline | FR-006, FR-015, FR-007 specify functional requirements. Where literal text assumes upstream capabilities not present in pinned AO, requirements must be clarified via proposal, not silently ignored or bypassed. |
| **5** | Approved Roadmap (`docs/17_ROADMAP.md`) | Phase Boundary Baseline | Phase P03 owns AOAdapter, session lifecycle, worker management. Phase P04 owns EvidenceCollector, WorkerReport parsing, schema validation. |
| **6** | Task Contract | Operational Boundary | Workers operate strictly within approved task contracts. |
| **7** | Source Repo / Reference (AO v0.13.0) | Upstream Reality | Pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` defines literal upstream behavior. Specs cannot demand what upstream physically lacks. |
| **8** | Worker Suggestion | Advisory | Suggestions (e.g. prior audit draft) must be verified against source. |
| **9** | Chat Conversation | Volatile Context | Non-authoritative. System truth lives in repo documents. |

### Evaluation of Conflicts Under the Hierarchy:
1. **FR-006 vs Source Reality**: Level 4 specification requires worker heartbeats, but Level 7 upstream reality provides only session status and activity state. Level 2 ADR-009 prohibits reinventing a worker daemon sidecar. Resolution: Clarify Level 4 requirement through governance (Option A) to define liveness observation via authoritative AO session/activity state.
2. **FR-015 vs Source Reality**: Level 4 specification requires version detection via daemon status, but Level 7 source `daemonProbePayload()` contains no version. Level 2 ADR-009 prohibits modifying upstream AO code. Resolution: Clarify Level 4 requirement to verify daemon liveness/readiness via `/healthz`/`/readyz`, harness availability via `/api/v1/agents`, and version provenance out-of-band / via OpenAPI schema fingerprint.
3. **ADR-011 vs Canonical Spec Omission**: Level 2 ADR-011 mandates report retrieval via `/workspace/file`, which takes precedence over Level 4 spec omission. Resolution: Reconcile `IAOAdapter` interface in `docs/12_UPSTREAM_INTEGRATION.md` by proposing `GetWorkspaceFile`.
4. **REC-005 vs Read-Only Architecture**: Level 4 failure recovery table instructs AOAdapter to stash/clean target worktree, directly contradicting Level 1/2/3 read-only architecture and Level 2 ADR-009 anti-reinvention. Level 3 Architecture takes precedence over Level 4 failure recovery text. Resolution: Amend REC-005 to fail-closed (`WORKTREE_DIRTY`) and escalate.

---

# Pinned AO Evidence

Literal verification of `Untrivial-ai/agent-orchestrator` at pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`:

1. **Health Probe Payload** (`backend/internal/httpd/router.go:394-420`):
   ```go
   func daemonProbePayload(status string, cfg config.Config) map[string]any {
       payload := map[string]any{
           "status":  status,
           "service": daemonmeta.ServiceName,
           "pid":     os.Getpid(),
       }
       if exe, err := os.Executable(); err == nil && exe != "" {
           payload["executablePath"] = exe
       }
       if cwd, err := os.Getwd(); err == nil && cwd != "" {
           payload["workingDirectory"] = cwd
       }
       if cfg.StartupWorkingDirectory != "" {
           payload["startupWorkingDirectory"] = cfg.StartupWorkingDirectory
       }
       if appImage := os.Getenv("AO_APPIMAGE"); appImage != "" {
           payload["appImagePath"] = appImage
       }
       return payload
   }
   ```
   **Literal Fact**: Returns status, service name, pid, and path metadata. **No version field exists**.

2. **Agent Readiness Endpoints** (`backend/internal/httpd/controllers/agents.go:36-44`):
   ```go
   func (c *AgentsController) Register(r chi.Router) {
       r.Get("/agents", c.list)
       r.Post("/agents/refresh", c.refresh)
       r.Get("/agents/readiness", c.readiness)
       r.Post("/agents/readiness/ensure", c.ensureReadiness)
       r.Post("/agents/{agent}/probe", c.probe)
       r.Get("/agents/{agent}/models", c.models)
       r.Post("/agents/{agent}/models/refresh", c.refreshModels)
   }
   ```
   **Literal Fact**: Public endpoints exist to inspect agent harness inventory and ensure readiness.

3. **OpenAPI Compatibility Spec** (`backend/internal/httpd/api.go:182` & `apispec/openapi.yaml`):
   ```go
   r.Get("/openapi.yaml", apispec.ServeYAML)
   ```
   **Literal Fact**: Serves committed OpenAPI 3.1.0 specification verbatim at `GET /api/v1/openapi.yaml`. `info.version` is `"0.1.0-route-shell"`.

4. **SSE Event Stream & Transport Keepalive** (`backend/internal/httpd/events.go:44-55`):
   ```go
   var eventsHeartbeatInterval = 10 * time.Second
   ```
   **Literal Fact**: Mounts `GET /api/v1/events` backed by `cdc.Source` (`change_log` table). Emits comment frame `: \n\n` every 10s to keep HTTP connection alive. It does not carry worker state or process telemetry.

5. **CDC Event Types** (`backend/internal/cdc/event.go:17-30`):
   * `session_created`, `session_updated`, `pr_created`, `pr_updated`, etc.
   * Captured by database triggers on SQLite tables.
   * Monotonic `Seq` ordering with replay cursor via `Last-Event-ID` / `X-AO-Event-After`.

6. **Authoritative Session State** (`backend/internal/httpd/controllers/sessions.go:401-413` & `1968-2000`):
   * `GET /api/v1/sessions/{sessionId}` returns `SessionView` containing `domain.Session`.
   * `Activity.State`: `"active"`, `"idle"`, `"waiting_input"`, `"blocked"`, `"exited"`.
   * `IsTerminated`: boolean flag.

7. **Agy Harness Activity Mapping** (`backend/internal/adapters/agent/agy/activity.go:13-22`):
   ```go
   func DeriveActivityState(event string, _ []byte) (domain.ActivityState, bool) {
       switch event {
       case "pre-invocation", "post-tool-use":
           return domain.ActivityActive, true
       case "stop":
           return domain.ActivityIdle, true
       default:
           return "", false
       }
   }
   ```
   **Literal Fact**: Agent hook callbacks translate directly: `"pre-invocation"`/`"post-tool-use"` -> `ActivityActive`; `"stop"` -> `ActivityIdle`.

8. **Workspace File Read Primitive** (`backend/internal/httpd/controllers/sessions.go:570-600`):
   * `GET /api/v1/sessions/{sessionId}/workspace/file?path={relPath}`
   * Returns `WorkspaceFileResponse` with `Content`, `Size`, `FileFingerprint`. Strictly confined to session workspace.

9. **Upstream Request Timeout Default** (`backend/internal/config/config.go:31`):
   ```go
   DefaultRequestTimeout = 60 * time.Second
   ```
   **Literal Fact**: Server-side request timeout default is 60 seconds.

---

# FR-006 Heartbeat Gap

### Current Canonical Requirement:
FR-006 literally states: "monitor worker process health, heartbeats, and status events via AO public API". Furthermore, P03 phase specifications mention `worker_started`, `worker_heartbeat`, `worker_finished`.

### Pinned AO Source Reality:
Upstream AO provides:
1. `Activity.State`: `"active"`, `"idle"`, `"waiting_input"`, `"blocked"`, `"exited"`.
2. `Activity.LastActivityAt`: Timestamp updated when CLI hooks fire.
3. `IsTerminated`: Boolean indicating session termination.
4. CDC `session_created` and `session_updated` events.
5. SSE connection keepalive comment frames (`eventsHeartbeatInterval = 10s`).

Upstream AO **DOES NOT** provide an authoritative worker heartbeat event. The 10-second `eventsHeartbeatInterval` in `backend/internal/httpd/events.go` writes an empty comment `: \n\n` strictly to prevent idle TCP drops across reverse proxies. It is not associated with any worker process, contains no worker identity, and emits no payload. It must NOT be mapped to `worker.heartbeat`.

### Evaluation of Options:
* **OPTION A — RECOMMENDED**: Clarify FR-006 so that "heartbeat" means **bounded liveness observation** through authoritative AO session snapshot state (`Activity.State`, `IsTerminated`), without inventing a synthetic `worker.heartbeat` event.
  - The canonical lifecycle event taxonomy remains: `worker.started`, `worker.stopped`, `worker.crashed`.
  - Turn completion remains: AO activity `idle` (derived authoritatively from agent `stop` hook callback).
  - No synthetic lifecycle event is emitted merely because a periodic poll succeeded.
  - If a worker becomes unresponsive or times out, bounded observation policy detects inactivity and transitions to `FAILED`/`CRASHED`.
* **OPTION B**: Introduce an explicit Supervisor-generated liveness sample concept (e.g. `supervisor.liveness_sample`) distinct from worker lifecycle events.
  - Must use a completely distinct semantic name.
  - Must NOT claim to be an upstream worker heartbeat.
  - Added complexity with minimal observability value over poll logs.
* **OPTION C**: Retain literal upstream worker-heartbeat requirement.
  - **UNSATISFIABLE** on pinned AO v0.13.0. Would require intrusive sidecar or modifying upstream AO, violating ADR-002 and ADR-009.

---

# FR-015 Runtime Compatibility Gap

### Current Canonical Requirement:
FR-015 requires verifying "daemon status, version, worker harness availability" during startup preflight.

### Prior Worker Inaccuracy:
Prior audit claimed `ErrAOVersionMismatch` is detected via the health probe version field. This was factually false: `daemonProbePayload()` in `backend/internal/httpd/router.go` provides `status`, `service`, `pid`, `executablePath`, `workingDirectory`, `startupWorkingDirectory`, `appImagePath`, but **NO version field**.

### Audit of Exact Public Compatibility Surfaces:
1. `GET /healthz`: Returns 200 OK with `status="ok"`, `service="agent-orchestrator-daemon"`, `pid`. Proves daemon liveness. (`DAEMON_LIVENESS = PROVEN`)
2. `GET /readyz`: Returns 200 OK with `status="ready"`, `service="agent-orchestrator-daemon"`, `pid`. Proves daemon readiness. (`DAEMON_READINESS = PROVEN`)
3. `GET /api/v1/agents`: Returns list of configured harnesses (`"agy"`, `"claude-code"`, etc.). (`HARNESS_INVENTORY = PUBLIC_API_AVAILABLE`)
4. `GET /api/v1/agents/readiness`: Returns cached readiness evaluation of all agents. (`HARNESS_READINESS = PUBLIC_API_AVAILABLE`)
5. `POST /api/v1/agents/readiness/ensure`: Actively triggers readiness checks. (`HARNESS_READINESS = PUBLIC_API_AVAILABLE`)
6. `GET /api/v1/openapi.yaml`: Serves the embedded OpenAPI 3.1.0 specification. (`API_COMPATIBILITY_FINGERPRINT = EVALUATE`)

### Recommended Semantic:
1. Runtime daemon liveness/readiness is verified via `/healthz` and `/readyz`.
2. Worker harness availability and readiness are verified via `/api/v1/agents` and `/api/v1/agents/readiness`.
3. AO version compatibility is guaranteed by **pinned deployment provenance** (Git commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`, release `v0.13.0`) and can be cross-verified via cryptographic fingerprinting of `/api/v1/openapi.yaml`, NOT a non-existent health JSON field.

---

# AOAdapter Workspace Read Interface Gap

### Architectural Context:
ADR-011 mandates that worker reports (`worker-report.json`) must be retrieved through AO's workspace file read endpoint (`GET /api/v1/sessions/{id}/workspace/file?path=...`). However, canonical `IAOAdapter` in `docs/12_UPSTREAM_INTEGRATION.md` contains no workspace file read method.

### Proposed Interface Primitive:
Add to `IAOAdapter`:
```go
// GetWorkspaceFile retrieves a raw file from the session's workspace.
// It is strictly read-only, session-scoped, and path-confined.
GetWorkspaceFile(ctx context.Context, sessionID string, relativePath string) ([]byte, error)
```

### Strict Operational Constraints:
* **Read-only**: Never modifies workspace files.
* **Session-scoped**: Must target an active or retained session.
* **Relative path confinement**: Paths must be clean, relative to workspace root; reject `..`, absolute paths, or path traversal.
* **Zero host filesystem bypass**: Must use AO's HTTP API, never raw OS filesystem reads across worktrees.
* **Zero schema interpretation**: Transport returns raw bytes; does NOT parse JSON, validate schemas, create `WorkerClaim`, or transition `StateStore`. Those remain strictly Phase P04 responsibilities.

---

# REC-005 Readonly Conflict

### Current Problem:
`docs/14_FAILURE_RECOVERY.md` REC-005 prescribes:
* *Dirty Worktree Detected*: Worktree contains uncommitted files before task dispatch.
* *Owner*: AOAdapter.
* *Action*: "Stash or clean untracked artifacts before initializing new task attempt."

### Architectural Conflict:
1. **Read-Only Invariant**: The Supervisor is strictly a read-only governance control plane. It must never run mutating git commands (`git stash`, `git clean`, `git checkout`) on target workspaces.
2. **Upstream Separation**: AO owns Git worktree lifecycle (`backend/internal/worktree/`).
3. **Anti-Reinvention**: Supervisor must not reimplement worktree state management.

### Recommended Resolution:
Amend REC-005 action:
* When a dirty worktree is detected prior to dispatch (via AO session/worktree read inspection or pre-dispatch check), the Supervisor **fails closed**.
* It rejects dispatch with error `WORKTREE_DIRTY`.
* It marks task attempt blocked/escalated (`HUMAN_REQUIRED` or `BLOCKED`), requiring operator resolution.
* Zero automated destructive git mutations are performed by the Supervisor.

---

# Alternatives

| Decision Area | Alternative | Pros | Cons | Verdict |
|---|---|---|---|---|
| **Heartbeat Observation** | **Alt 1A**: Pure Bounded Session Snapshot Polling (`GET /sessions/{id}`) | Simple, deterministic, reliable, zero dependency on CDC DB triggers. | Periodic HTTP request overhead (e.g. 1s–2s interval). | **RECOMMENDED** |
| | **Alt 1B**: CDC `/events` stream wake-up + snapshot verify | Low latency event push. | CDC stream not proven in P01; complex reconnection logic. | DEFERRED / OPTIONAL ENHANCEMENT |
| | **Alt 1C**: Treat CDC payload as lifecycle authority | Zero polling. | Fragile, binds Supervisor to internal AO DB schema, no turn-completion guarantee. | REJECTED |
| | **Alt 1D**: Synthetic `worker.heartbeat` on every poll | Satisfies literal FR-006 text. | Fabricates fake events not emitted by upstream, corrupts audit trail. | REJECTED |
| **Runtime Compatibility** | **Alt 2A**: Health probe + Agent endpoints + Pinned git provenance | 100% honest to upstream source, no artificial fields. | Requires understanding that version is verified out-of-band. | **RECOMMENDED** |
| | **Alt 2B**: Attempt to parse binary metadata or Electron files | Might extract a string. | Violates container/daemon isolation, highly brittle. | REJECTED |
| **Workspace Read** | **Alt 3A**: `GetWorkspaceFile` in `IAOAdapter` via AO REST API | Session-scoped, follows ADR-011, honors read-only boundary. | Requires adding one method to adapter interface. | **RECOMMENDED** |
| | **Alt 3B**: Direct host filesystem read (`os.ReadFile`) | Avoids adding adapter method. | Violates ADR-002, ADR-011, and container isolation. | REJECTED |
| **Dirty Worktree** | **Alt 4A**: Fail closed with `WORKTREE_DIRTY` / escalate | Upholds read-only invariant, prevents data loss, safe. | Requires manual or AO-level reset. | **RECOMMENDED** |
| | **Alt 4B**: Run `git stash` / `git clean` from Supervisor | Cleans worktree automatically. | Destructive, violates read-only architecture and ADR-009. | REJECTED |

---

# Recommended Resolution

1. **FR-006 Clarification**: Adopt Option A. Define worker process health and lifecycle observation via periodic bounded polling of authoritative AO session state (`GET /api/v1/sessions/{id}`). Turn completion is observed when AO activity transitions to `idle` (`ActivityIdle`), which AO derives directly from the agent's native `stop` hook. Lifecycle events emitted by Supervisor remain strictly `worker.started`, `worker.stopped`, `worker.crashed`. No synthetic `worker.heartbeat` event is fabricated.
2. **FR-015 Clarification**: Reconcile FR-015 preflight: verify daemon liveness (`/healthz`) and readiness (`/readyz`); verify agent harness availability and readiness (`/api/v1/agents` and `/api/v1/agents/readiness`); assert AO version compatibility via pinned deployment provenance and build manifest, optionally verified via `/api/v1/openapi.yaml` schema fingerprint.
3. **IAOAdapter Interface Extension**: Add `GetWorkspaceFile(ctx, sessionID, relPath)` to canonical `IAOAdapter`. Strictly bounded, read-only, session-confined.
4. **REC-005 Amendment**: Amend REC-005 in `docs/14_FAILURE_RECOVERY.md` so that detected dirty worktrees fail closed with `WORKTREE_DIRTY` without running mutating git commands.
5. **Phase Boundary Enforcement**: Preserve strict phase separation. P03 implements `GetWorkspaceFile` raw transport primitive only. WorkerReport parsing, schema validation, WorkerClaim creation, and `REPORT_READY` transition remain strictly Phase P04 responsibilities.

---

# Architecture Impact Assessment

* **Canonical 3-Tier Architecture**: Completely preserved. Supervisor Core -> AOAdapter -> Agent Orchestrator.
* **Read-Only Invariant**: Reinforced. By rejecting `git stash`/`git clean` in REC-005, the Supervisor remains 100% read-only on target repositories.
* **ADR-002, ADR-009, ADR-011, ADR-012**: 100% honored and reinforced.
* **Database/Domain Isolation**: Zero AO DTO pollution of Supervisor domain models.

---

# ADR Requirement Assessment

* **Determination**: `NO — PROVISIONAL PENDING PROPOSAL REVIEW`.
* **Rationale**: The proposed resolutions do not alter the canonical system architecture or introduce new external dependencies. They clarify existing requirement semantics (FR-006, FR-015), reconcile an interface omission already established by ADR-011 (`GetWorkspaceFile`), and correct an operational inconsistency in failure recovery (REC-005). Existing ADRs (`ADR-002`, `ADR-009`, `ADR-011`, `ADR-012`) fully govern these boundaries. If the External Supervisor requests a formal ADR, one will be prepared following proposal approval.

---

# Exact Canonical Documents Requiring Future Amendment

Amendments will be applied **ONLY AFTER** formal external approval of this proposal:

| Canonical Document | Section | Proposed Amendment Summary |
|---|---|---|
| `docs/02_REQUIREMENTS.md` | FR-006 | Clarify that worker health and turn observation are performed via authoritative AO session/activity state (`active` -> `idle`) rather than literal upstream worker heartbeats. |
| `docs/02_REQUIREMENTS.md` | FR-015 | Clarify compatibility check: health probe liveness/readiness, harness readiness via `/api/v1/agents`, and version provenance out-of-band / OpenAPI schema fingerprint. |
| `docs/12_UPSTREAM_INTEGRATION.md` | `IAOAdapter` | Add `GetWorkspaceFile(ctx context.Context, sessionID string, relativePath string) ([]byte, error)` to interface definition. |
| `docs/14_FAILURE_RECOVERY.md` | REC-005 | Change action from "Stash or clean untracked artifacts" to "Fail closed with WORKTREE_DIRTY; escalate to HUMAN_REQUIRED; zero destructive git commands". |
| `docs/21_TRACEABILITY_MATRIX.md` | FR-006, FR-015, FR-007 | Reconcile mappings: FR-006 to lifecycle observation; FR-015 to health+agents+provenance; FR-007 exclusively to P04 EvidenceCollector. |
| `docs/phases/P03_AO_INTEGRATION.md` | P03 Scope & Exit Gate | Align task decomposition with revised 5-task structure; restrict P03 exit gate to session lifecycle and raw file transport, excluding P04 report validation. |

---

# Proposed P03 Task Ownership

To ensure strict compliance with canonical phase boundaries, the proposed P03 task decomposition is revised as follows:

### TASK-P03-001: AO Transport Foundation & Compatibility Read Model
* Loopback-only REST client (`http://127.0.0.1:{port}`);
* Base URL validation and security policy;
* Daemon liveness (`/healthz`) and readiness (`/readyz`);
* Harness inventory and readiness (`/api/v1/agents`, `/api/v1/agents/readiness`);
* Project registration/verification (`/api/v1/projects`);
* Session inspection (`GET /api/v1/sessions/{id}`);
* Error envelope decoding (`envelope.APIError`);
* Normalized adapter error mapping;
* Zero session mutation commands; zero polling loop; zero WorkerReport ingestion.

### TASK-P03-002: Session Mutation & Dispatch Boundary
* Session creation (`POST /api/v1/sessions`);
* Session prompt dispatch (`POST /api/v1/sessions/{id}/send`);
* Session termination (`POST /api/v1/sessions/{id}/kill`);
* Session restore (`POST /api/v1/sessions/{id}/restore`);
* ADR-012 durable pre-dispatch state validation (assert task is `DISPATCHED` with valid attempt before calling AO);
* Zero attempt allocation inside adapter; zero direct SQLite access.

### TASK-P03-003: Lifecycle Observation & State Reconciliation
* Authoritative session state observation loop (`GET /api/v1/sessions/{id}`);
* Observes worker transition to `ActivityActive` -> executes StateStore transition `DISPATCHED -> RUNNING` via existing P02 StateStore/domain transition APIs;
* Observes turn completion when `Activity.State` transitions to `ActivityIdle` (derived from agent `stop` hook);
* Observes session exit / failure (`IsTerminated = true` or `ActivityExited`) -> maps to `worker.stopped` or `worker.crashed`;
* Injects bounded observation policy (configurable poll interval, inactivity timeout, operation deadlines);
* Zero synthetic worker heartbeat events; zero direct SQL writes; zero StateMachine bypass.

### TASK-P03-004: AO Workspace Read Transport Primitive
* Implements `GetWorkspaceFile(ctx, sessionID, relativePath)`;
* Calls `GET /api/v1/sessions/{id}/workspace/file?path={relPath}`;
* Enforces session-scoped relative path confinement (rejects traversal, absolute paths);
* Returns raw bounded byte payload;
* Zero WorkerReport parsing; zero schema validation; zero WorkerClaim creation; zero StateStore transitions (strictly reserved for P04).

### TASK-P03-005: P03 Live AO Integration & Exit Gate Validation
* End-to-end integration test against live pinned AO daemon;
* Validates daemon compatibility, project setup, session spawn, dispatch, active / running transition, idle turn completion, workspace file retrieval, and clean teardown;
* Validates zero domain model pollution by upstream DTOs;
* Proves P03 exit gate compliance without demanding P04 EvidenceCollector behavior.

---

# Risks

1. **Polling Overhead Risk**: Frequent polling of `GET /sessions/{id}` could impose CPU load.
   - *Mitigation*: Configurable poll interval (default 1.0s to 2.0s) with bounded maximums.
2. **Inactivity False Positive Risk**: Long-running tool executions might appear idle.
   - *Mitigation*: AOAdapter updates `lastActivityAt` on every CLI tool execution (`post-tool-use`, `pre-invocation`). Bounded timeout applies to absence of any activity signal.
3. **Timeout Mismatch Risk**: Upstream AO has a 60s server-side request timeout default (`config.DefaultRequestTimeout`).
   - *Mitigation*: Supervisor client uses dedicated, configurable HTTP client timeouts (`SUPERVISOR_HTTP_TIMEOUT`), distinct from operation deadlines.

---

# External Approval Gate

* **Final Status**: `RECOMMENDED_FOR_EXTERNAL_APPROVAL`
* **Condition**: Awaiting formal review and decision by External Supervisor.
* **Prohibition**: No canonical requirement files may be modified and no production Go code in `internal/**` may be written until this proposal is explicitly approved.
