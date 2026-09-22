# 12. UPSTREAM INTEGRATION SPECIFICATION

> **Focus**: AOAdapter Interface, Antigravity CLI Integration & Fallback Boundary
> **Status**: Approved Baseline

---

# 1. AOAdapter Interface Contract

The `AOAdapter` is an anti-corruption layer shielding the Supervisor Control Plane from Untrivial Agent Orchestrator's internal data structures.

### Canonical Interface Methods

```typescript
interface IAOAdapter {
  // Daemon Health & Registration
  checkHealth(): Promise<AOHealthStatus>;
  registerProject(projectId: string, rootPath: string): Promise<AOResult>;
  getProject(projectId: string): Promise<AOProjectDetails>;

  // Session & Worker Lifecycle
  createWorkerSession(params: CreateWorkerParams): Promise<AOSessionDetails>;
  getWorkerStatus(sessionId: string): Promise<AOWorkerStatus>;
  stopWorker(sessionId: string): Promise<AOResult>;
  resumeWorker(sessionId: string): Promise<AOResult>;

  // Task Dispatch & Communication
  dispatchTaskContract(sessionId: string, contract: TaskContract): Promise<AODispatchResult>;
  subscribeEvents(sessionId: string, onEvent: (e: AOEvent) => void): Subscription;

  // Workspace & Git Inspection
  getWorktreePath(sessionId: string): Promise<string>;
  getActiveBranch(sessionId: string): Promise<string>;

  // Workspace Inspection & File Retrieval (Raw Transport)
  getWorkspaceFile(sessionId: string, relativePath: string): Promise<Uint8Array>;
}
```

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
