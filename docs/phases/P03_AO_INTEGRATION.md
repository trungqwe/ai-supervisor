# PHASE SPECIFICATION: P03 — AO INTEGRATION (EXECUTION ADAPTER)

## 1. Objective
Implement `AOAdapter` to connect the Supervisor Control Plane to the Untrivial Agent Orchestrator public REST API for session lifecycle management, authoritative snapshot observation, and raw workspace artifact read transport.

## 2. Deliverables
- `AOAdapter` implementation against pinned public REST surfaces:
  - **Preflight Probes & API Surface**: `GET /healthz` (liveness), `GET /readyz` (readiness), `GET /api/v1/agents` (harness inventory), `GET /api/v1/agents/readiness` (harness readiness), `GET /api/v1/openapi.yaml` (API schema compatibility signal; does not prove daemon release version).
  - **Project & Session Lifecycle**: `POST /api/v1/projects`, `POST /api/v1/sessions`, `POST /api/v1/sessions/{id}/send`, `POST /api/v1/sessions/{id}/kill`, `POST /api/v1/sessions/{id}/restore`, `GET /api/v1/sessions/{id}`.
  - **Raw Workspace Transport**: `GET /api/v1/sessions/{id}/workspace/file?path={relPath}`.
- Normalized lifecycle observation over authoritative session snapshots (`GET /api/v1/sessions/{id}`), observing active, idle, and terminated states.
- Active/idle/terminated state observation and turn completion mapping per ADR-011.
- Supervisor orchestration layer integration for `DISPATCHED -> RUNNING` transition via P02 StateStore/domain APIs (no StateStore dependency inside AOAdapter).
- Raw session workspace artifact read primitive (`GetWorkspaceFile`) for retrieving attempt artifacts without semantic report interpretation or claim creation (strictly reserved for Phase P04 EvidenceCollector).
- Automated contract tests asserting zero domain pollution from AO DTOs.
- Zero synthetic worker heartbeat requirement.
- Zero active CDC/SSE event subscription deliverable (stream integration deferred as optional future optimization; baseline observation uses authoritative GET session snapshot polling).

## 3. Exit Gate
- Automated integration tests successfully perform preflight health/readiness/agent probes, spawn an AO worker session, dispatch instructions, observe active execution and turn-complete idle states, fetch raw workspace artifact, and cleanly terminate without P04 EvidenceCollector dependencies.
