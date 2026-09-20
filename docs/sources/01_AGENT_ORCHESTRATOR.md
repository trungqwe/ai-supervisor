# SOURCE DOSSIER: 01 — UNTRIVIAL AGENT ORCHESTRATOR

## 1. Metadata
- **Repository**: `Untrivial-ai/agent-orchestrator`
- **Role in Architecture**: Active Upstream Dependency (Execution Control Plane)
- **Pinned Tag**: `v0.13.0`
- **Pinned Commit**: `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`
- **Commit Date**: 2026-03-05T08:52:13Z
- **License**: Apache-2.0
- **Primary Language**: Go

---

# 2. Capabilities Evaluated & Adopted
- **Daemon Lifecycle & Health Probes**: `/healthz` and `/readyz` endpoints.
- **Project & Session Management**: REST endpoints for registering projects and spawning isolated agent sessions.
- **Session Control Operations**: Killing (`/kill`), restoring (`/restore`), and sending prompts (`/send`).
- **Terminal Management**: ConPTY allocation for pseudoterminal process execution on Windows.

---

# 3. Capabilities Explicitly Rejected
- **Internal Database Direct Access**: Reading/writing directly to AO's internal storage is strictly prohibited.
- **Autonomous Multi-Agent Routing**: AO's multi-agent routing is rejected for V1.
- **Cloud Fleet Features**: Remote cloud worker spawning is excluded.

---

# 4. Integration Strategy & Upstream Boundary
- **Strategy**: `UPSTREAM` via `AOAdapter`.
- **Boundary**: HTTP REST over loopback (`127.0.0.1`).
- **What We Must NOT Rebuild**: Process daemon, Windows ConPTY terminal mux, Git worktree isolation scripts.

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-AO-001
Claim: AO exposes health and readiness probe endpoints for daemon monitoring.
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/httpd/router.go
Section / symbol: r.Get("/healthz", ...), r.Get("/readyz", ...)
Verification: VERIFIED
Confidence: HIGH
Notes: Returns daemon probe JSON payload with PID and working directory.

Claim ID: CLM-AO-002
Claim: AO provides REST endpoints for project registration and session management.
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/httpd/controllers/projects.go, backend/internal/httpd/controllers/sessions.go
Section / symbol: ProjectsController.Register, SessionsController.Register
Verification: VERIFIED
Confidence: HIGH
Notes: Mounts POST /api/v1/projects, POST /api/v1/sessions, GET /api/v1/sessions/{id}.

Claim ID: CLM-AO-003
Claim: AO provides session execution control endpoints (send, kill, restore).
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/httpd/controllers/sessions.go
Section / symbol: r.Post("/sessions/{sessionId}/kill", c.kill), r.Post("/sessions/{sessionId}/send", c.send), r.Post("/sessions/{sessionId}/restore", c.restore)
Verification: VERIFIED
Confidence: HIGH
Notes: Documented endpoints map to session manager control functions.

Claim ID: CLM-AO-004
Claim: AO launches Antigravity CLI using interactive prompt flags rather than headless stream-json print mode.
Repository: Untrivial-ai/agent-orchestrator
Pinned tag: v0.13.0
Pinned commit: 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6
Evidence type: SOURCE_CODE
Exact evidence: backend/internal/adapters/agent/agy/agy.go
Section / symbol: GetLaunchCommand, GetRestoreCommand
Verification: VERIFIED
Confidence: HIGH
Notes: Pinned adapter generates `agy --add-dir <WorkspacePath> [--dangerously-skip-permissions] [--model <Model>] [--prompt-interactive <Prompt>]`. Does NOT use `--print` or `--json-schema`.
