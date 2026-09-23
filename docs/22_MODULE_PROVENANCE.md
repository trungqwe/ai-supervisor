# 22. MODULE PROVENANCE & ANTI-REINVENTION JUSTIFICATION

> **Authority**: Core Module Ownership Register & Anti-Reinvention Proof
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

# 1. Planned Modules Provenance

### Module: `ProjectRegistry`
- **Purpose**: Tracks registered local projects, validates local directory existence, and stores project metadata.
- **Origin**: Mieruko workspace binding pattern (`docs/sources/04_MIERUKO.md`).
- **Existing Upstream Capability Checked**: YES (AO has project registration, but lacks Supervisor-specific documentation indexing).
- **Reason This Module Exists in Our Code**: Links local workspace paths to canonical architecture docs and Pair state.
- **Why We Own This**: AO manages execution paths; Supervisor owns project documentation indexing and governance metadata.
- **Forbidden Responsibility**: Does NOT manage Git repositories or create worktrees.

### Module: `PairRegistry`
- **Purpose**: Manages the engineering lane binding one project, one ChatGPT session, and one active worker session.
- **Origin**: Mieruko workspace binding concepts (`docs/sources/04_MIERUKO.md`).
- **Existing Upstream Capability Checked**: YES (AO manages sessions, but has no concept of ChatGPT Supervisor binding).
- **Reason This Module Exists in Our Code**: Prevents multi-project interference and validates session tokens.
- **Why We Own This**: Core abstraction of the Supervisor Control Plane.
- **Forbidden Responsibility**: Does NOT manage OS processes or ConPTY allocations.

### Module: `TaskContractManager`
- **Purpose**: Validates, stores, and enforces immutable Task Contracts.
- **Origin**: OpenAI Symphony work units and Codencer task schemas (`docs/sources/06_SYMPHONY.md`, `docs/sources/07_CODENCER.md`).
- **Existing Upstream Capability Checked**: YES (AO accepts task instructions, but enforces no schema or scope immutability).
- **Reason This Module Exists in Our Code**: Enforces strict mathematical scope bounds on worker tasks.
- **Why We Own This**: Defines the authoritative work contract between Supervisor and Worker.
- **Forbidden Responsibility**: Does NOT execute tasks or edit code.

### Module: `PolicyEngine`
- **Purpose**: Compares Git diffs against contract `allowed_scope` and flags violations.
- **Origin**: Proxide containment and security model (`docs/sources/03_PROXIDE.md`).
- **Existing Upstream Capability Checked**: YES (AO has no scope filtering; workers can touch any file in the worktree).
- **Reason This Module Exists in Our Code**: Prevents worker agents from altering architecture, root configs, or other modules.
- **Why We Own This**: Upstream AO does not enforce architectural boundary policies.
- **Forbidden Responsibility**: Does NOT block OS file access directly; operates as post-execution audit gate.

### Module: `EvidenceCollector`
- **Purpose**: Orchestrates independent evidence acquisition (Git base/head SHAs, diffs, changed files, scope compliance, and trusted verification test results). It MAY invoke an internal constrained verification runner to execute approved host-owned verification profiles (`verification_requests` per ADR-013) within an isolated execution boundary.
- **Origin**: AIWorkHub candidate verification and Codencer evidence model (`docs/sources/05_AIWORKHUB.md`, `docs/sources/07_CODENCER.md`).
- **Existing Upstream Capability Checked**: YES (AO exposes session lifecycle/termination observations through its public API, but the Supervisor MUST NOT assume public process exit-code availability unless a separately verified public surface proves it; AO does not correlate Git diffs or build test audit packets).
- **Reason This Module Exists in Our Code**: Zero-trust verification: worker claims must be corroborated by independent evidence.
- **Why We Own This**: Bridge between Git/test ground truth and ChatGPT audit.
- **Forbidden Responsibility**: Does NOT execute arbitrary shell; does NOT perform worker implementation; does NOT expose command execution primitives to ChatGPT; does NOT accept unconstrained user shell strings. Implementation remains Phase P04.

### Module: `ReviewBundleBuilder`
- **Purpose**: Assembles Task Contract, Worker Claims, Git Diff, Test Artifacts, and Policy Findings into a unified JSON bundle.
- **Origin**: AIWorkHub / Codencer review model (`docs/sources/05_AIWORKHUB.md`, `docs/sources/07_CODENCER.md`).
- **Existing Upstream Capability Checked**: YES (No existing open-source tool builds ChatGPT-tailored review bundles).
- **Reason This Module Exists in Our Code**: Drastically reduces ChatGPT tool calls from 25+ down to 1.
- **Why We Own This**: Core cognitive compression innovation of the AI Engineering Supervisor Control Plane.
- **Forbidden Responsibility**: Does NOT approve or reject tasks; only packages data for ChatGPT.

### Module: `AuditLogger`
- **Purpose**: Records append-only, secret-sanitized chronological logs of all pair, task, and evidence events.
- **Origin**: Proxide audit sanitization (`docs/sources/03_PROXIDE.md`).
- **Existing Upstream Capability Checked**: YES (AO logs daemon events, but lacks domain-level pair and review governance trails).
- **Reason This Module Exists in Our Code**: Ensures full auditability and post-mortem diagnosis.
- **Why We Own This**: Authoritative record of supervisor-worker coordination.
- **Forbidden Responsibility**: Does NOT store plaintext secrets or environment passwords.

### Module: `AOAdapter`
- **Purpose**: Translates domain operations into Untrivial Agent Orchestrator public REST API calls (`GET /healthz`, `GET /readyz`, `GET /api/v1/agents`, `GET /api/v1/agents/readiness`, `GET /api/v1/openapi.yaml`, `POST /api/v1/projects`, `POST /api/v1/sessions`, `POST /api/v1/sessions/{id}/send`, `POST /api/v1/sessions/{id}/kill`, `POST /api/v1/sessions/{id}/restore`, `GET /api/v1/sessions/{id}`, `GET /api/v1/sessions/{id}/workspace/file`), provides normalized session telemetry/status translation, enforces pre-send status whitelist validation (`idle`, `waiting_input` only), and provides a session-scoped raw workspace-file read primitive.
- **Origin**: AWS CAO provider abstraction (`docs/sources/08_AWS_CAO.md`).
- **Existing Upstream Capability Checked**: YES (AO provides the public REST API; public API does not guarantee process exit code availability).
- **Wire Contract Details**:
  - `POST /api/v1/sessions/{id}/kill`: Wire request carries session identity only. Stop `purpose`, generation precheck, and `confirmation_deadline_at` are Supervisor-owned logic/metadata in `stop_operations`; they are not wire parameters and do not form an atomic generation fence.
  - `POST /api/v1/sessions/{id}/restore`: Verified pinned wire route (`operationId: restoreSession`) restoring a terminated session (`ResumeWorker`). Distinguish from `POST /api/v1/sessions/{id}/resume-agent` which resumes an exited agent process within an active session without restoring workspace or terminated session.
  - Pre-send validation: Mandatory check that `AOWorkerStatus.status IN ('idle', 'waiting_input')` before issuing `/send`.
- **Reason This Module Exists in Our Code**: Decouples Supervisor domain logic from AO internal changes and adapts worker session telemetry.
- **Why We Own This**: Anti-corruption layer shielding the domain; handles empirical gap resolution identified in P01-C.
- **Forbidden Responsibility**: Does NOT own or access StateStore or SQLite directly (`AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`, `AOADAPTER_DIRECT_SQL = FORBIDDEN`); does NOT allocate TaskAttempt; does NOT hold Task state transition authority (owned by Supervisor orchestration layer using P02 APIs); does NOT perform WorkerReport semantic interpretation, schema validation, or WorkerClaim creation (strictly P04 EvidenceCollector); does NOT mutate Git worktrees (no `git stash`, `git clean`, checkout, reset); does NOT invoke Antigravity CLI (`agy`) directly; does NOT vendor AO code, couple to AO internal packages, or access AO internal SQLite database.

### Module: `ChatGPTToolSurface` & `TransportAdapter`
- **Purpose**: Exposes the 12 high-level domain tools over the OpenAI-supported transport proven in Phase P01 Track P01-D (FR-016).
- **Origin**: Mieruko slim tool surface (`docs/sources/04_MIERUKO.md`).
- **Existing Upstream Capability Checked**: YES (Mieruko/Proxide provide generic MCP tools, but none implement our 12-tool contract).
- **Reason This Module Exists in Our Code**: Provides the cognitive interface for ChatGPT Web.
- **Why We Own This**: Tailored cognitive interface for our supervision model.
- **Forbidden Responsibility**: Does NOT expose arbitrary shell, raw file writing, OS GUI automation, or low-level process details (such as worker PID). Does NOT perform branch merges, commits, or pushes upon task approval (V1 approval strictly records decision).

---

# 2. Phase P01-D Transport Provenance & Anti-Reinvention Clarifications

### Component: `Secure MCP Tunnel (tunnel-client)`
- **Classification**: **`UPSTREAM — OpenAI`**
- **Existing Upstream Capability Checked**: YES (`openai/tunnel-client` binary / GitHub releases).
- **Reason We Do Not Implement**: OpenAI provides the official outbound-only secure tunnel connecting loopback MCP servers to OpenAI products.
- **Why We Own This**: We do NOT own or reimplement this; we consume it strictly as an upstream utility.

### Component: `SupervisorMCPAdapter` (Local MCP Server)
- **Classification**: **`OUR ADAPTER SURFACE`**
- **Purpose**: Implements Model Context Protocol for future Phase P05 (via official Go MCP SDK: `github.com/modelcontextprotocol/go-sdk`, Transport: Streamable HTTP per proven P01-D3C architecture distinction) exposing domain supervisor probes (`supervisor_probe_read`, `supervisor_probe_write`) and tool schemas. Phase P02 does not implement MCP.
- **Existing Upstream Capability Checked**: YES (Generic MCP servers exist, but none provide Supervisor Control Plane lifecycle and contract tooling).
- **Why We Own This**: Domain-specific cognitive and audit surface for ChatGPT supervision.
- **Forbidden Responsibility**: Does NOT expose arbitrary command execution, raw disk access, or external network egress.

### Component: `SupervisorPublicGateway`
- **Classification**: **`P01-D3B DESIGN DECISION PENDING`**
- **Purpose**: Public HTTPS MCP proxy required if public plugin distribution is pursued after publisher identity verification.
- **Status**: Gated pending identity verification. Zero implementation authorized in Phase P01.

### Component: `Plugin Skills Packaging`
- **Classification**: **`OPTIONAL PACKAGING/WORKFLOW LAYER`**
- **Purpose**: Packaging static prompt guidance (`SKILL.md`) alongside MCP tools.
- **Status**: Optional. Prohibited from duplicating or bypassing server-side `TaskContractManager` or `PolicyEngine` invariants.

### `VerificationPolicyCatalog` (Domain Policy Interface)
- **Phase**: P02 (Interface & In-Memory Fake) / P04 (Host Configuration Loader)
- **Provenance**: Proxide / Architecture V2.1
- **Purpose**: Exposes read-only `VerificationProfilePolicy` metadata to `TaskContractValidator` for semantic verification request validation without exposing executable paths or command construction logic to the core domain.

---

# 3. Store Entities Provenance & Anti-Reinvention Justification (ADR-016)

### Entity: `pair_provisioning_operations`
- **Purpose**: Durably tracks Pair worker session provisioning intent across crash boundaries (`PROVISION_REQUESTED` -> `PROVISION_CONFIRMED` / `PROVISION_FAILED`). Records unresolved spawn uncertainty for operator-audited handling.
- **Origin**: Distributed saga pattern & distributed intent logging.
- **Existing Upstream Capability Checked**: YES (AO provides `POST /api/v1/sessions` but has no concept of Pair lane binding or intent-state durability across crash boundaries; AO does not identify unowned orphan sessions or track provisioning sagas).
- **Reason This Entity Exists in Our Code**: Prevents orphan session leakage and guarantees that crashes during provisioning leave the Pair lane safely locked rather than ambiguously partially initialized (D3). Unowned orphan sessions are not automatically identified or killed.
- **Why We Own This**: Core lifecycle safety and Pair lane concurrency management.

### Entity: `dispatch_operations`
- **Purpose**: Enforces durable 3-stage dispatch saga (`DISPATCH_BOUND` -> `SEND_REQUESTED` -> `SEND_CONFIRMED`) and enforces strict 1:1 cardinality with task execution attempts (`UNIQUE(attempt_id)`).
- **Origin**: Two-phase commit / transactional outbox dispatch pattern.
- **Existing Upstream Capability Checked**: YES (AO provides `POST /api/v1/sessions/{id}/send` but provides no delivery-acknowledgement fence or retry idempotency against socket drops; AO accepts prompts on any session without verifying task contracts).
- **Reason This Entity Exists in Our Code**: Ensures zero instruction loss, prevents duplicate task prompt delivery, and handles transport drop crashes fail-closed (`SEND_REQUESTED` -> `UNCERTAIN_DELIVERY_CRASH` + double-gated quarantine) (D4).
- **Why We Own This**: Execution integrity and fail-closed crash recovery over external process runtimes.

### Entity: `stop_operations`
- **Purpose**: Durably tracks purpose-aware worker termination operations (`purpose IN ('RUNNING_ATTEMPT_STOP', 'QUARANTINE_CLEANUP', 'PAIR_MAINTENANCE')`) across stages (`STOP_REQUESTED`, `STOP_CALL_SUCCEEDED`, `STOP_TERMINATION_CONFIRMED`, `STOP_TARGET_ABSENT`) and resolution states, with restart-stable `confirmation_deadline_at`.
- **Origin**: Purpose-aware distributed cancellation pattern.
- **Existing Upstream Capability Checked**: YES (AO provides `POST /api/v1/sessions/{id}/kill` taking only session ID, but exposes no purpose tagging, no atomic generation fencing, and no deadline management; AO does not track why a session was stopped or reconcile restart outcomes).
- **Reason This Entity Exists in Our Code**: Prevents blind re-killing (`STOP_REISSUE_REQUIRES_HUMAN`), guarantees restart-stable temporal bounding under `WORKER_STOPPED_ALLOWED_IFF`, and distinguishes live task abortion from maintenance cleanup (D11).
- **Why We Own This**: Supervisor owns operational lifecycle semantics and failure attribution.

### Extensions to `task_attempts`
- **Purpose**: Captures execution identity, recovery disposition, and quarantine safety state without altering the frozen `TaskContract` specification.
- **Exact Schema Additions**: Exactly four columns per ADR-016 §27: `session_id TEXT`, `terminal_generation TEXT`, `recovery_disposition TEXT`, and `quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK(quarantine_state IN ('CLEAN', 'QUARANTINED'))`.
- **Origin**: Distributed execution correlation & containment.
- **Existing Upstream Capability Checked**: YES (AO returns `terminalGeneration` and `status` in snapshots, but does not track task attempt boundaries or quarantine state).
- **Reason This Exists in Our Code**: Binds immutable specification to concrete execution attempt; separates diagnostic outcome (`recovery_disposition`) from safety gate (`quarantine_state`) (D1, D6, D12).
- **Why We Own This**: Core task lifecycle and execution containment.

### Anti-Reinvention Justification Summary
These persistent store entities do not duplicate any capability provided by Untrivial Agent Orchestrator. AO is an execution daemon exposing a point-in-time REST API for process/session management. AO does NOT provide distributed saga consistency, crash-resilient transactional intent tracking, double-gated quarantine safety locks, purpose-aware cancellation accounting, or task-to-session execution identity binding. The Supervisor Control Plane must own these entities to satisfy NFR-003 (crash safety) and guarantee deterministic governance over external execution runtimes.


# 4. ADR-016 addendum v4 provenance

restore_authorizations và pair_restore_operations thuộc Supervisor Store vì pinned AO không có idempotency key, cross-process Pair ownership, audit transaction hoặc quarantine clearance. v4 migration/Store guard và trusted operator interface thuộc 3B; AOAdapter dùng method đã có, không sửa wire. Linked stop persistence/guard chung phải fail-closed từ 3B; stop coordinator và D6 clearance orchestration thuộc 3C; scanner/poller thuộc 3D. Existing Store mutation APIs (PrepareDispatch, Create/UpdateDispatchOperation, Create/UpdateProvisioningOperation, CreateWorkerSession, UpdateWorkerSessionStatusAndQuarantine, Create/UpdateStopOperation) không được bypass RESTORE_UNRESOLVED: guard trong cùng transaction hoặc đóng đường P03.
