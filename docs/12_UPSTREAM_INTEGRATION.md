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
}
```

---

# 2. Invariants and Architectural Guards
1. **No Direct SQLite Access**: The adapter exclusively calls AO's loopback REST endpoints or CLI subcommands.
2. **Session Containment**: Every task execution runs in a dedicated Git worktree managed by AO.
3. **Graceful Fallback**: If an AO API endpoint fails, the adapter emits domain-typed exceptions (`AODaemonUnavailableException`, `AOSessionNotFoundException`) rather than unhandled transport crashes.

---

# 3. Antigravity Fallback Adapter (`AgyFallbackAdapter`)
If an unexpected operational issue prevents Agent Orchestrator from running a specific Antigravity workflow:
- The `AgyFallbackAdapter` defines direct invocation of `agy --headless` via Windows ConPTY.
- **Activation Gate**: This adapter remains inactive and dormant unless an approved ADR explicitly authorizes direct invocation.
