# PHASE SPECIFICATION: P03 — AO INTEGRATION (EXECUTION ADAPTER)

## 1. Objective
Implement `AOAdapter` to connect the Supervisor Control Plane to the Untrivial Agent Orchestrator public REST API for session lifecycle management, authoritative snapshot observation, and raw workspace artifact read transport.

## 2. Deliverables
- `AOAdapter` implementation against pinned public REST surfaces:
  - **Preflight Probes & API Surface**: `GET /healthz` (liveness), `GET /readyz` (readiness), `GET /api/v1/agents` (harness inventory), `GET /api/v1/agents/readiness` (harness readiness), `GET /api/v1/openapi.yaml` (API schema compatibility signal; does not prove daemon release version).
  - **Project & Session Lifecycle**: `POST /api/v1/projects`, `POST /api/v1/sessions`, `POST /api/v1/sessions/{id}/send`, `POST /api/v1/sessions/{id}/kill` (session identity only wire request), `POST /api/v1/sessions/{id}/restore` (verified pinned wire route restoring terminated session; `operationId: restoreSession`), `GET /api/v1/sessions/{id}`.
  - **Raw Workspace Transport**: `GET /api/v1/sessions/{id}/workspace/file?path={relPath}`.
- Decoupled worker session lifecycle primitives: Pair provisioning separated from Task dispatch, supporting reuse across tasks via `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`.
- Strict pre-send status whitelist validation (`AOWorkerStatus.status IN ('idle', 'waiting_input')`) prior to issuing `POST /api/v1/sessions/{id}/send`.
- Durable 3-stage dispatch saga integration (`DISPATCH_BOUND` -> `SEND_REQUESTED` -> `SEND_CONFIRMED`) with fail-closed quarantine containment on unknown delivery.
- Purpose-aware stop operations (`purpose IN ('RUNNING_ATTEMPT_STOP', 'QUARANTINE_CLEANUP', 'PAIR_MAINTENANCE')`) with restart-stable `confirmation_deadline_at` tracking and generation prechecks.
- Double-gated quarantine integration (`Pair Lane Gate` and `Task Attempt Gate`) with Class A/B/C clearance playbooks.
- Normalized lifecycle observation over authoritative session snapshots (`GET /api/v1/sessions/{id}`), observing active, idle, waiting_input, blocked, and exited states.
- Active/idle/terminated state observation and turn completion mapping per ADR-011 and ADR-016.
- Supervisor orchestration layer integration for `DISPATCHED -> RUNNING` transition via P02 StateStore/domain APIs (no StateStore dependency inside AOAdapter).
- Raw session workspace artifact read primitive (`GetWorkspaceFile`) for retrieving attempt artifacts without semantic report interpretation or claim creation (strictly reserved for Phase P04 EvidenceCollector).
- Automated contract tests asserting zero domain pollution from AO DTOs.
- Zero synthetic worker heartbeat requirement.
- Zero active CDC/SSE event subscription deliverable (stream integration deferred as optional future optimization; baseline observation uses authoritative GET session snapshot polling).

## 3. Exit Gate
- Automated integration tests successfully perform preflight health/readiness/agent probes, decoupled session provisioning, durable 3-stage dispatch saga, observation reconciliation across active/idle/waiting_input/blocked states, raw workspace artifact fetch, and purpose-aware termination/quarantine enforcement without P04 EvidenceCollector dependencies.


## 4. ADR-016 addendum release boundaries

Lịch sử 3B candidate gồm v4 durable restore authorization/operation, Pair guard, provisioning/dispatch saga, pre-send hold và unknown delivery resolution. Trusted interface được test fail-closed; host/bootstrap integration phải chứng minh authenticated operator principal trước khi enable operator restore hoặc linked stop; AUTOMATIC_RESTORE=DISABLED. 3C sở hữu stop coordinator và clearance orchestration cho từng lineage; 3D sở hữu synchronous startup scanner/poller và còn DESIGN_BLOCKER_3D_STARTUP_WIRING. Contracts 3B/3C và 3D (revisions 1, 2 và revision 3 cuối cùng) đã RELEASED; implementation 3A/3B/3C/3D đều đã EXTERNAL_AUDIT_APPROVED và MERGED.

## 5. Execution budget design bổ sung cho 3D

Addendum ADR-016 execution budget phê duyệt origin `dispatch_operations.confirmed_at`, immutable policy/deadline v5, startup zero timeout effect và runtime monitor dùng shared host admission + 3C one-use stop. Tx R atomic stop intent/double quarantine/audit; sau reservation observation không admissible thì giữ `STOP_REQUESTED/IN_FLIGHT`, không replay kill. Legacy Run incomplete dùng trusted maintenance độc quyền; historical policy binding và manual stop khác authority, manual stop chỉ exact open `RUNNING`, không ép `DISPATCHED`. Contract revision 3 đã RELEASED tại `68194f739fc5a514cd1f7850027771a77a866d47`; implementation 3D EXTERNAL_AUDIT_APPROVED tại commit `7513f1b9f39fa459be15e0abc836c6258a610d8b`; `3D-R1-001..005` và `3D-R2-001..002` đã CLOSED ở library scope; code đã MERGED tại `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`. Handoff dependencies (`HOST_QUIESCENCE_INTEGRATION=OPEN`, verified host principal chưa có bằng chứng runtime, `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`, `AUTOMATIC_RESTORE=DISABLED`) tiếp tục duy trì đến khi có host bootstrap integration.
