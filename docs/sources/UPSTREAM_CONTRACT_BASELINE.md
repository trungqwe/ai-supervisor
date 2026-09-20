# UPSTREAM CONTRACT BASELINE

> **Focus**: Public Interfaces & Operations Expected from Active Upstreams  
> **Status**: Approved Baseline for Phase 0 & Upgrade Audits

---

# 1. Untrivial Agent Orchestrator (AO) Public Contract

The Supervisor Control Plane integrates with AO strictly through its documented REST / CLI endpoints.

### Expected Operations
1. **`GET /api/v1/health`**:
   - *Purpose*: Verify daemon status, version, and platform capability.
   - *Expected Response*: `{"status": "healthy", "version": "0.13.0", "platform": "windows"}`.
2. **`POST /api/v1/projects`**:
   - *Purpose*: Register a project root directory with the AO daemon.
   - *Payload*: `{"project_id": string, "path": string}`.
3. **`POST /api/v1/sessions`**:
   - *Purpose*: Create an isolated agent session with a dedicated Git worktree.
   - *Payload*: `{"project_id": string, "harness": "antigravity", "branch": string}`.
   - *Expected Response*: `{"session_id": string, "worktree_path": string}`.
4. **`POST /api/v1/sessions/{id}/send`**:
   - *Purpose*: Send the immutable Task Contract / instructions to the active worker.
   - *Payload*: `{"task_id": string, "payload": string}`.
5. **`GET /api/v1/sessions/{id}`**:
   - *Purpose*: Query worker execution status, exit code, and activity timestamps.
   - *Expected Response*: `{"status": "running"|"completed"|"failed", "exit_code": int}`.
6. **`POST /api/v1/sessions/{id}/stop`**:
   - *Purpose*: Terminate a hanging or blocked worker session cleanly.
7. **`GET /api/v1/events` (WebSocket / SSE)**:
   - *Purpose*: Stream execution events (`worker_started`, `worker_heartbeat`, `worker_finished`).

---

# 2. Official Antigravity CLI (`agy`) Contract

When invoked in headless mode inside the worktree by AO:
- **Invocation**: `agy --headless --mode autonomous --input-contract <path> --output-report <path>`.
- **Exit Codes**:
  - `0`: Success (Worker claims task completion).
  - `1`: Failure (Compilation, test failure, or internal error).
  - `2`: Blocked (Worker detected architectural constraint and stopped).
- **Output Artifact**: Valid JSON file conforming to `docs/schemas/worker-report.schema.json`.
