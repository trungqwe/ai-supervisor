# P01-A — AGENT ORCHESTRATOR RUNTIME PROOF DOSSIER

> **Authority**: Phase 1 Upstream Proof Authority (Track P01-A)
> **Status**: PASS
> **Verdict**: EMPIRICALLY_PROVEN_ON_TARGET_WINDOWS
> **Date**: 2026-09-21
> **Repository**: `D:\TU_CODE\ai-supervisor`
> **Baseline Commit**: `fea488a5410cef5557160516302953c4ad9b6468`
> **Historical Frozen Baseline**: `phase0-architecture-v1` (`6f72eaca30be3fc3ac00f25829dd4283ed98c3f5`)
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN`
> **Isolated Sandbox**: `D:\TU_CODE\_ai_supervisor_p01a_ao_runtime`

---

# 1. Executive Summary & Verification Matrix

Track P01-A rigorously proves that pinned upstream **Untrivial Agent Orchestrator (AO)** (`v0.13.0`, commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) executes faithfully on the target Windows host environment, delivering all architectural primitives claimed in ADR-002, ADR-003, and ADR-004.

All **21 acceptance items** and the **cleanup protocol** passed with 100% compliance:

| Check ID | Acceptance Item | Result | Verification Evidence |
|---|---|---|---|
| **P01A-01** | `AO_PIN_COMMIT` | **PASS** | Exact commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` verified via Git rev-parse |
| **P01A-02** | `AO_PIN_TAG` | **PASS** | Pinned tag `v0.13.0` resolves identically to commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` |
| **P01A-03** | `AO_BUILD_BINARY` | **PASS** | Built canonical binary from `./cmd/ao` using Go 1.26.0 (SHA256: `DA8C92810E81964396BBEB69FF27C50CA7241D347AE5F239E5AE084B678EDA13`) |
| **P01A-04** | `AO_DAEMON_START` | **PASS** | Daemon started in background on isolated port `4140`, PID `28732` |
| **P01A-05** | `AO_CONTROL_LOOPBACK` | **PASS** | Loopback binding verified on `127.0.0.1:4140` |
| **P01A-06** | `AO_HEALTH` | **PASS** | `GET /healthz` returned HTTP 200 OK (`pid: 28732`, `status: "ok"`) |
| **P01A-07** | `AO_READY` | **PASS** | `GET /readyz` returned HTTP 200 OK (`status: "ready"`) |
| **P01A-08** | `AO_PROJECT_REGISTER` | **PASS** | `POST /api/v1/projects` registered fixture repo as `fixture-repo` (HTTP 200) |
| **P01A-09** | `AO_PROJECT_GET` | **PASS** | `GET /api/v1/projects/fixture-repo` returned HTTP 200 OK with correct paths and default branch |
| **P01A-10** | `AO_SESSION_SPAWN` | **PASS** | `POST /api/v1/sessions` spawned session `fixture-repo-1` (HTTP 201 Created) |
| **P01A-11** | `AO_SESSION_GET` | **PASS** | `GET /api/v1/sessions/fixture-repo-1` returned HTTP 200 OK (`status: "working"`, `activity.state: "active"`) |
| **P01A-12** | `AO_WORKTREE_CREATED` | **PASS** | Worktree created at `runtime\data\worktrees\fixture-repo\fixture-repo-1` |
| **P01A-13** | `AO_WORKTREE_ISOLATED` | **PASS** | Worktree on dedicated branch `ao/fixture-repo-1/root`; base fixture repo working tree remains untouched |
| **P01A-14** | `AO_PATH_GUARD_RUNTIME` | **PASS** | Path traversal (`../../outside.txt`) and absolute paths (`C:\Windows\System32`) rejected HTTP 400; upstream path guard unit tests 100% PASS |
| **P01A-15** | `AO_WINDOWS_CONPTY` | **PASS** | Win32 process tree confirmed: `ao daemon` (PID 28732) -> `ao pty-host` (PID 35108) -> `conhost.exe --headless` (PID 47528) -> `cmd.exe /c agy.cmd` (PID 25104) |
| **P01A-16** | `AO_SESSION_SEND` | **PASS** | `POST /api/v1/sessions/fixture-repo-1/send` delivered instruction into session (HTTP 200 OK) |
| **P01A-17** | `AO_WORKSPACE_EVENTS` | **PASS** | SSE stream `GET /api/v1/sessions/fixture-repo-1/workspace/events` connected (HTTP 200) and emitted `workspace_changed` upon worktree file modification |
| **P01A-18** | `AO_SESSION_KILL` | **PASS** | `POST /api/v1/sessions/fixture-repo-1/kill` cleanly terminated session (HTTP 200 OK) |
| **P01A-19** | `AO_ORPHAN_PROCESS_CHECK` | **PASS** | Verified all child processes (PIDs 35108, 47528, 25104) terminated immediately; zero orphan processes; daemon PID 28732 remained alive and healthy |
| **P01A-20** | `AO_SESSION_RESTORE` | **PASS** | `POST /api/v1/sessions/fixture-repo-1/restore` restored terminated session (`restoreMode: "native"`, `status: "idle"`, HTTP 200 OK) |
| **P01A-21** | `AO_RESTORE_LIVENESS` | **PASS** | Post-restore instruction delivery `POST /send` confirmed active liveness (HTTP 200 OK) |
| **CLEANUP** | `P01A_CLEANUP` | **PASS** | Test session killed, isolated daemon stopped, port 4140 confirmed closed, zero lingering processes |

---

# 2. Upstream Baseline & Build Metadata

### 2.1 Repository & Commit Pinning
- **Upstream Repository**: `Untrivial-ai/agent-orchestrator`
- **Pinned Git Tag**: `v0.13.0`
- **Pinned Git Commit**: `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`
- **Commit Author Date**: `2026-09-12T06:30:25Z`
- **Commit Committer Date**: `2026-09-12T06:30:25Z`
- **SPDX License**: `Apache-2.0`

```bash
git -C D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\upstream rev-parse HEAD
# Output: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6

git -C D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\upstream rev-parse v0.13.0^{commit}
# Output: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
```

### 2.2 Compilation & Binary Verification
- **Go Compiler Version**: `go version go1.26.0 windows/amd64`
- **Build Entrypoint**: `./cmd/ao` (from `backend/` directory)
- **Binary Target**: `D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\runtime\ao.exe`
- **Binary Size**: `40,482,816 bytes`
- **SHA-256 Hash**: `DA8C92810E81964396BBEB69FF27C50CA7241D347AE5F239E5AE084B678EDA13`

> [!NOTE]
> On Windows, AO requires the binary to be built from `backend/cmd/ao` rather than `backend/main.go`. `main.go` only invokes `daemon.Run()`, whereas the multi-call entrypoint in `cmd/ao` handles both `ao daemon` and the `ao pty-host` sub-process used by Windows ConPTY (`backend/internal/adapters/runtime/conpty/spawn_windows.go`).

---

# 3. Daemon Execution & Health Verification

### 3.1 Startup Configuration
The isolated daemon was launched with the following environment:
- `AO_PORT = "4140"`
- `AO_DATA_DIR = "D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\runtime\data"`
- `AO_RUN_FILE = "D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\runtime\running.json"`
- `AO_TELEMETRY_EVENTS = "off"`
- `AO_TELEMETRY_METRICS = "off"`
- `AO_TELEMETRY_REMOTE = "off"`
- **Assigned Process PID**: `28732`
- **Listen Address**: `127.0.0.1:4140` (Loopback only)

### 3.2 Health Check (`GET /healthz`)
```http
GET /healthz HTTP/1.1
Host: 127.0.0.1:4140
```

**Response (HTTP 200 OK)**:
```json
{
  "executablePath": "D:\\TU_CODE\\_ai_supervisor_p01a_ao_runtime\\runtime\\ao.exe",
  "pid": 28732,
  "service": "agent-orchestrator-daemon",
  "startupWorkingDirectory": "D:\\TU_CODE\\_ai_supervisor_p01a_ao_runtime\\runtime",
  "status": "ok",
  "workingDirectory": "D:\\TU_CODE\\_ai_supervisor_p01a_ao_runtime\\runtime\\data"
}
```

### 3.3 Readiness Check (`GET /readyz`)
```http
GET /readyz HTTP/1.1
Host: 127.0.0.1:4140
```

**Response (HTTP 200 OK)**:
```json
{
  "status": "ready"
}
```

---

# 4. Project & Session Lifecycle Verification

### 4.1 Fixture Repository Initialization
A clean Git fixture repository was initialized at:
- **Path**: `D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\fixture-repo`
- **Initial Commit**: `b198d9f` (`Initial commit for P01-A AO proof`)
- **Default Branch**: `main`
- **Config**: `git config --local ao.defaultBranch main`

### 4.2 Project Registration (`POST /api/v1/projects`)
```http
POST /api/v1/projects HTTP/1.1
Host: 127.0.0.1:4140
Content-Type: application/json

{
  "id": "fixture-repo",
  "name": "P01-A Fixture Repo",
  "path": "D:\\TU_CODE\\_ai_supervisor_p01a_ao_runtime\\fixture-repo"
}
```

**Response (HTTP 200 OK)**:
```json
{
  "project": {
    "id": "fixture-repo",
    "name": "P01-A Fixture Repo",
    "path": "D:\\TU_CODE\\_ai_supervisor_p01a_ao_runtime\\fixture-repo",
    "defaultBranch": "main"
  }
}
```

### 4.3 Session Spawn & Worktree Creation (`POST /api/v1/sessions`)
```http
POST /api/v1/sessions HTTP/1.1
Host: 127.0.0.1:4140
Content-Type: application/json

{
  "projectId": "fixture-repo",
  "displayName": "p01a-probe",
  "harness": "agy"
}
```

**Response (HTTP 201 Created)**:
```json
{
  "session": {
    "id": "fixture-repo-1",
    "projectId": "fixture-repo",
    "kind": "worker",
    "harness": "agy",
    "displayName": "p01a-probe",
    "mode": "tui",
    "activity": {
      "state": "active",
      "lastActivityAt": "2026-09-21T09:25:46.3982304Z"
    },
    "isTerminated": false,
    "status": "working",
    "kanbanColumn": "building",
    "displayStatus": "Building",
    "terminalHandleId": "fixture-repo-1",
    "branch": "ao/fixture-repo-1/root"
  }
}
```

### 4.4 Git Worktree Isolation Verification
Verified via Git CLI that the session was allocated an isolated worktree outside the fixture repository's primary working tree:
```text
Worktree Path: D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\runtime\data\worktrees\fixture-repo\fixture-repo-1
Branch:        ao/fixture-repo-1/root
HEAD Commit:   b198d9f

git -C D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\fixture-repo worktree list --porcelain:
worktree D:/TU_CODE/_ai_supervisor_p01a_ao_runtime/fixture-repo
HEAD b198d9ffc71b637a28cebf3e4ca1eaadfb7e1c84
branch refs/heads/main

worktree D:/TU_CODE/_ai_supervisor_p01a_ao_runtime/runtime/data/worktrees/fixture-repo/fixture-repo-1
HEAD b198d9ffc71b637a28cebf3e4ca1eaadfb7e1c84
branch refs/heads/ao/fixture-repo-1/root
```
The fixture repo working tree remained clean with 0 uncommitted changes.

---

# 5. Windows ConPTY & Process Hierarchy Verification

Inspection of the Win32 process tree confirmed that AO utilizes native Windows ConPTY architecture (`conhost.exe --headless`) to manage interactive terminal sessions without window popups or terminal corruption:

```text
[PID 28732] D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\runtime\ao.exe daemon
    └── [PID 35108] D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\runtime\ao.exe pty-host fixture-repo-1 D:\TU_CODE\_ai_supervisor_p01a_ao_runtime\runtime\data\worktrees\fixture-repo\fixture-repo-1 --cmd C:\Windows\system32\cmd.exe --arg /c --arg C:\Users\Admin\AppData\Roaming\npm\agy.cmd --arg --prompt-interactive --arg p01a-probe
            ├── [PID 47528] \\?\C:\Windows\system32\conhost.exe --headless --width 80 --height 25 --signal 0x00000304
            └── [PID 25104] C:\Windows\system32\cmd.exe /c C:\Users\Admin\AppData\Roaming\npm\agy.cmd --prompt-interactive p01a-probe
```

**Empirical Confirmation**:
- `ao.exe pty-host` manages the pseudo-console handle.
- `conhost.exe` runs with flag `--headless` ensuring invisible background terminal processing.
- The harness (`agy.cmd`) is invoked within the pseudo-console.

---

# 6. Path Guard Runtime Containment

AO implements directory boundary validation (`backend/internal/sessionguard/path_safety.go`) to prevent workspace escape attacks. Both public API endpoints and upstream unit tests were verified:

### 6.1 Public Endpoint Directory Traversal Probe
```http
GET /api/v1/sessions/fixture-repo-1/workspace/file?path=../../outside.txt HTTP/1.1
Host: 127.0.0.1:4140
```

**Response (HTTP 400 Bad Request)**:
```json
{
  "error": {
    "code": "INVALID_WORKSPACE_PATH",
    "message": "path escapes session workspace"
  }
}
```

### 6.2 Public Endpoint Absolute Path Probe
```http
GET /api/v1/sessions/fixture-repo-1/workspace/file?path=C:\Windows\System32 HTTP/1.1
Host: 127.0.0.1:4140
```

**Response (HTTP 400 Bad Request)**:
```json
{
  "error": {
    "code": "INVALID_WORKSPACE_PATH",
    "message": "workspace path must be relative"
  }
}
```

### 6.3 Upstream Unit Test Execution
Executed on host Windows environment:
```bash
go test -v ./internal/sessionguard/ -run "TestManagedPathSafety|TestValidateConfigRejectsPathEscapingIDs"
```
**Result**: `ok untrivial/agent-orchestrator/backend/internal/sessionguard 0.316s` (100% PASS).

---

# 7. Instruction Delivery & Workspace Event Streaming

### 7.1 Instruction Delivery (`POST /api/v1/sessions/{sessionId}/send`)
```http
POST /api/v1/sessions/fixture-repo-1/send HTTP/1.1
Host: 127.0.0.1:4140
Content-Type: application/json

{
  "message": "Hello from supervisor test delivery"
}
```

**Response (HTTP 200 OK)**:
```json
{
  "ok": true,
  "sessionId": "fixture-repo-1"
}
```

### 7.2 Workspace Event SSE Stream (`GET /api/v1/sessions/{sessionId}/workspace/events`)
An SSE listener connected to `http://127.0.0.1:4140/api/v1/sessions/fixture-repo-1/workspace/events` (HTTP 200, `Content-Type: text/event-stream; charset=utf-8`).

When a test probe file was written to the session worktree (`runtime\data\worktrees\fixture-repo\fixture-repo-1\p01a_event_test_1789982881054.txt`), the SSE stream emitted:
```text
event: workspace_changed
data: {"workspaceVersion":"7818a0ab0db6a97a4a5742247430c3369c086f0653dd9eff7eaef9f3c8216d92","overflow":true}
```
This empirically proves that filesystem modifications in the worker's worktree immediately trigger real-time workspace invalidation notifications via SSE.

---

# 8. Process Termination, Orphan Audit & Session Restore

### 8.1 Process Termination (`POST /api/v1/sessions/{sessionId}/kill`)
```http
POST /api/v1/sessions/fixture-repo-1/kill HTTP/1.1
Host: 127.0.0.1:4140
Content-Type: application/json

{}
```

**Response (HTTP 200 OK)**:
```json
{
  "ok": true,
  "sessionId": "fixture-repo-1"
}
```

### 8.2 Orphan Process Audit
A complete Win32 process inspection was performed 2 seconds after the kill call:
- **PTY Host (PID 35108)**: Terminated (No longer exists)
- **ConPTY Headless Host (PID 47528)**: Terminated (No longer exists)
- **Worker Command (PID 25104)**: Terminated (No longer exists)
- **Residual Session Processes**: **0** (Zero orphan processes)
- **AO Daemon (PID 28732)**: Alive and fully healthy (`/healthz` returned HTTP 200 OK)

### 8.3 Post-Kill Session State (`GET /api/v1/sessions/{sessionId}`)
```json
{
  "session": {
    "id": "fixture-repo-1",
    "projectId": "fixture-repo",
    "status": "terminated",
    "activity": {
      "state": "exited",
      "lastActivityAt": "2026-09-21T09:28:15.2959944Z"
    },
    "isTerminated": true,
    "displayStatus": "Terminated",
    "branch": "ao/fixture-repo-1/root"
  }
}
```

### 8.4 Session Conversation Restore (`POST /api/v1/sessions/{sessionId}/restore`)
```http
POST /api/v1/sessions/fixture-repo-1/restore HTTP/1.1
Host: 127.0.0.1:4140
Content-Type: application/json

{}
```

**Response (HTTP 200 OK)**:
```json
{
  "ok": true,
  "sessionId": "fixture-repo-1",
  "restoreMode": "native",
  "session": {
    "id": "fixture-repo-1",
    "projectId": "fixture-repo",
    "kind": "worker",
    "harness": "agy",
    "status": "idle",
    "activity": {
      "state": "idle",
      "lastActivityAt": "2026-09-21T09:28:36.8649298Z"
    },
    "isTerminated": false,
    "kanbanColumn": "building",
    "displayStatus": "Awaiting PR",
    "terminalHandleId": "fixture-repo-1",
    "branch": "ao/fixture-repo-1/root"
  }
}
```

### 8.5 Post-Restore Liveness Verification (`POST /send`)
```http
POST /api/v1/sessions/fixture-repo-1/send HTTP/1.1
Host: 127.0.0.1:4140
Content-Type: application/json

{
  "message": "hello after restore"
}
```

**Response (HTTP 200 OK)**:
```json
{
  "ok": true,
  "sessionId": "fixture-repo-1",
  "message": "hello after restore"
}
```

Win32 process inspection confirmed a fresh terminal instance was spawned:
- New PTY Host: PID `45960`
- New CMD Host: PID `10048`
- New Agy Instance: PID `12428`

---

# 9. Cleanup Verification (`P01A_CLEANUP`)

Upon completion of runtime proofs:
1. Active session `fixture-repo-1` was terminated cleanly via `POST /kill`.
2. The isolated daemon process (PID `28732`) was stopped.
3. `Get-NetTCPConnection -LocalPort 4140 -State Listen` returned zero listeners (port 4140 completely closed and released).
4. All child processes verified terminated (0 lingering processes).
5. All artifacts and execution logs preserved in sandbox `D:\TU_CODE\_ai_supervisor_p01a_ao_runtime`.

---

# 10. Audit Verdict & Next Phase Gate

| Metric | Result | Authority |
|---|---|---|
| **Track P01-A Proof Result** | **`PASS`** | 21/21 Acceptance Checks 100% PASS |
| **AO Windows ConPTY** | **`EMPIRICALLY_PROVEN`** | Validated via headless conhost.exe child tree |
| **AO Worktree Isolation** | **`EMPIRICALLY_PROVEN`** | Dedicated worktree branch verified clean |
| **AO Path Guard** | **`EMPIRICALLY_PROVEN`** | Traversal and absolute escapes rejected HTTP 400 |
| **AO Workspace SSE** | **`EMPIRICALLY_PROVEN`** | Invalidation event emitted on filesystem change |
| **AO Kill & Restore** | **`EMPIRICALLY_PROVEN`** | Zero orphans on kill; native restore restored idle state |
| **Next Gate** | **`EXTERNAL_SUPERVISOR_P01_A_AUDIT`** | STOP for External Supervisor review |
