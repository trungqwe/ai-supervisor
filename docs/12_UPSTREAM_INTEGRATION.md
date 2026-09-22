# 12. UPSTREAM INTEGRATION SPECIFICATION

> **Focus**: AOAdapter Interface, Antigravity CLI Integration & Fallback Boundary
> **Status**: Approved Baseline

---

# 1. AOAdapter Interface Contract

The `AOAdapter` is an anti-corruption layer shielding the Supervisor Control Plane from Untrivial Agent Orchestrator's internal data structures.

### Canonical Interface Methods

```typescript
interface IAOAdapter {
  // Preflight Probes & API Compatibility (FR-015)
  checkHealth(): Promise<AOHealthStatus>;                           // GET /healthz (liveness)
  checkReadiness(): Promise<AOReadinessStatus>;                      // GET /readyz (readiness)
  listAgents(): Promise<AOAgentList>;                               // GET /api/v1/agents (harness presence)
  getAgentReadiness(agentName?: string): Promise<AOAgentReadiness>; // GET /api/v1/agents/readiness (harness readiness)
  getAPIContract(): Promise<string>;                                // GET /api/v1/openapi.yaml (public schema compatibility signal only)

  // Project Registration
  registerProject(projectId: string, rootPath: string): Promise<AOResult>;
  getProject(projectId: string): Promise<AOProjectDetails>;

  // Session & Worker Lifecycle (Decoupled Provisioning, ADR-016 D2, D11)
  createWorkerSession(params: CreateWorkerParams): Promise<AOSessionDetails>; // POST /api/v1/sessions
  getWorkerStatus(sessionId: string): Promise<AOWorkerStatus>;                // GET /api/v1/sessions/{id} (authoritative snapshot)
  stopWorker(sessionId: string): Promise<AOResult>;                           // POST /api/v1/sessions/{id}/kill (session identity only wire request)
  resumeWorker(sessionId: string): Promise<AOResult>;                         // POST /api/v1/sessions/{id}/restore (restoreSession: restore terminated session)

  // Task Dispatch & Communication (Strict Pre-Send Whitelist, ADR-016 D7)
  dispatchTaskContract(sessionId: string, contract: TaskContract): Promise<AODispatchResult>; // POST /api/v1/sessions/{id}/send

  // Workspace Inspection & File Retrieval (Raw Transport)
  getWorkspaceFile(sessionId: string, relativePath: string): Promise<Uint8Array>; // GET /api/v1/sessions/{id}/workspace/file?path={relPath}
}
```

### Authoritative Status Structure (`AOWorkerStatus`)

```typescript
interface AOWorkerStatus {
  sessionId: string;
  status: "active" | "idle" | "waiting_input" | "blocked" | "exited";
  isTerminated: boolean;
  terminal_generation: string | null; // Opaque string generation identifier
}
```

> [!NOTE]
> **Pinned AO Wire Realism & Crash Evidence Absence**:
> - Pinned AO v0.13.0 public API exposes no process exit timestamp, exit code, OS signal, or crash-causality evidence (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`).
> - Worktree paths are not exposed in public session read models (`worktree_path TEXT NULL`, `WORKTREE_PATH_PUBLIC_AVAILABILITY = UNAVAILABLE_IN_PINNED_V1`).
> - Generation observations provide identity correlation, not an atomic kill fence.

### Pre-Send Admissibility & Process Control Policies

1. **Strict Pre-Send Admissibility Whitelist (ADR-016 D7)**:
   - Before executing `dispatchTaskContract` (which calls loopback `POST /api/v1/sessions/{id}/send`), the Supervisor MUST invoke `getWorkerStatus(sessionId)`.
   - Dispatch is permitted **ONLY** when `AOWorkerStatus.status IN ('idle', 'waiting_input')`.
   - Dispatch is **STRICTLY PROHIBITED** and must fail closed if status is `active` (indicating an unexpected foreign or lingering turn in flight) or terminal (`blocked`, `exited`, `isTerminated == true`).

2. **Purpose-Aware Stop Lifecycle & Wire Realism (ADR-016 D11)**:
   - Pinned AO `/kill` wire contract takes only session identity: `POST /api/v1/sessions/{sessionId}/kill`.
   - It possesses **NO** atomic generation fence (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`), no purpose parameter, and no deadline parameter on the wire.
   - Stop `purpose` (`RUNNING_ATTEMPT_STOP`, `QUARANTINE_CLEANUP`, `PAIR_MAINTENANCE`), generation precheck, and restart-stable `confirmation_deadline_at` are owned strictly by Supervisor logic and persisted in `stop_operations` metadata.
   - Stop confirmation evaluates positive call acceptance (`HTTP 200 OK`) and observation within `confirmation_deadline_at`. It enforces `STOP_REISSUE_REQUIRES_HUMAN` (prohibiting blind re-kill on restart).

3. **Session Restoration vs. Agent Resumption Boundary (ADR-016 D1, D3)**:
   - `resumeWorker` maps to pinned wire route `POST /api/v1/sessions/{sessionId}/restore` (`operationId: restoreSession`), which restores a **terminated session**, relaunching it and recreating or preserving workspace. This satisfies ADR-016's governed `ResumeWorker` path for terminated-but-restorable sessions.
   - Pinned AO separately provides `POST /api/v1/sessions/{sessionId}/resume-agent` (`operationId: resumeAgent`), which resumes an exited agent in an active session without restoring workspace or terminated session; this is NOT used for terminated session recovery.

4. **FR-015 Preflight Classification**:
   - `checkHealth()` maps to loopback `GET /healthz`, asserting daemon liveness (`status = "ok"`).
   - `checkReadiness()` maps to loopback `GET /readyz`, asserting daemon readiness (`status = "ready"`).
   - `listAgents()` maps to `GET /api/v1/agents`, asserting required harness availability (`"agy"`).
   - `getAgentReadiness(agentName)` maps to `GET /api/v1/agents/readiness`, asserting harness execution readiness.
   - `getAPIContract()` retrieves the raw public OpenAPI schema from `GET /api/v1/openapi.yaml`. It serves strictly as an **API compatibility signal** (`OPENAPI_CONTRACT = API_COMPATIBILITY_SIGNAL_ONLY`). It does **not** prove daemon binary release identity (`RUNTIME_AO_RELEASE_VERSION_PUBLIC_FIELD = ABSENT`, `PINNED_RELEASE_IDENTITY = DEPLOYMENT_PROVENANCE`).

5. **CDC Event Subscription Policy (P03 Baseline)**:
   - Event streaming subscription methods (such as legacy CDC event subscriptions) are formally **removed** from the active canonical `IAOAdapter` interface.
   - Upstream AO `/api/v1/events` is a global database change-data-capture (CDC) stream, not a session lifecycle authority, not a worker heartbeat, and not required for the V1 P03 baseline.
   - `CDC Event Subscription = DEFERRED OPTIONAL FUTURE OPTIMIZATION` for future wake-up hints only.
   - Baseline worker lifecycle observation remains strictly bounded snapshot polling via `getWorkerStatus(sessionId)` (`GET /api/v1/sessions/{id}`).

---

# 2. Invariants and Architectural Guards
1. **No Direct SQLite Access**: The adapter exclusively calls AO's loopback REST endpoints or CLI subcommands.
2. **Session Containment**: Every task execution runs in a dedicated Git worktree managed by AO.
3. **Graceful Fallback**: If an AO API endpoint fails, the adapter emits domain-typed exceptions (`AODaemonUnavailableException`, `AOSessionNotFoundException`) rather than unhandled transport crashes.
4. **No StateStore Ownership**: The `AOAdapter` is strictly an anti-corruption transport adapter. It has zero dependency on StateStore or SQLite (`AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`, `AOADAPTER_DIRECT_SQL = FORBIDDEN`), performs zero `TaskAttempt` allocations, and does not own or execute Task state transitions. All Task state transitions are owned and executed exclusively by the Supervisor orchestration/use-case layer using existing P02 domain/StateStore APIs.
5. **Read-Only Workspace File Retrieval**: `getWorkspaceFile` calls AO public REST (`GET /api/v1/sessions/{id}/workspace/file?path={relPath}`) strictly within session-scoped relative paths; returns raw bounded artifact content; performs zero `WorkerReport` semantic validation, zero `WorkerClaim` creation, and zero StateStore transitions (strictly reserved for Phase P04 `EvidenceCollector`).

---

# 3. Antigravity Fallback Adapter (`AgyFallbackAdapter`)
If an unexpected operational issue prevents Agent Orchestrator from running a specific Antigravity workflow:
- The `AgyFallbackAdapter` defines direct invocation of `agy --headless` via Windows ConPTY.
- **Activation Gate**: This adapter remains inactive and dormant unless an approved ADR explicitly authorizes direct invocation.

---

# 6. OpenAI Transport & MCP Upstream Integration Boundary

> **Governance Authority**: Added in Phase P01-D3A following Official OpenAI Ingestion Dossiers (`10_OPENAI_PLUGIN_PLATFORM.md`, `11_OPENAI_API_MCP_RUNTIME.md`).

### 6.1 Upstream Classifications
1. **Secure MCP Tunnel (`openai/tunnel-client`)**:
   - **Role**: `UPSTREAM — OpenAI`.
   - **Mechanism**: Outbound-only HTTPS long-poll connection initiated by customer-run `tunnel-client` on Windows to OpenAI control plane.
   - **Boundary**: Strict anti-reinvention mandate. We do not implement custom reverse WebSocket tunnels or relay proxies.
2. **Responses API MCP Invocation**:
   - **Role**: `UPSTREAM PRODUCT CAPABILITY`.
   - **Mechanism**: Official OpenAI Responses API (`v1/responses`) invokes MCP tools via `tunnel_id`. Native approval flow (`mcp_approval_request` / `mcp_approval_response`) acts as the transport data-sharing boundary.
3. **MCP Protocol Standard & SDK**:
   - **Role**: `UPSTREAM`.
   - **Standard**: Model Context Protocol (Streamable HTTP on loopback `http://127.0.0.1:<port>/mcp`). Official `@modelcontextprotocol/sdk` (TypeScript) and `mcp` (Python).
4. **Supervisor Domain MCP Surface**:
   - **Role**: `OUR ADAPTER SURFACE`.
   - **Scope**: Implements bounded domain probes and supervisor operations (`supervisor_probe_read`, `supervisor_probe_write`). Strictly isolates local filesystem, Git state, and AO execution from arbitrary LLM mutation.
5. **Production Public Gateway**:
   - **Role**: `P01-D3B DESIGN DECISION PENDING`.
   - **Status**: Subject to formal design decision pending publisher identity verification and public marketplace review requirements.
6. **Plugin Skills Layer**:
   - **Role**: `OPTIONAL PACKAGING/WORKFLOW LAYER`.
   - **Policy**: Non-core in V1. Evaluated for workflow instructions but prohibited from duplicating or bypassing server-side `TaskContractManager` and `PolicyEngine` invariants.
