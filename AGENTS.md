# AGENTS.md — Operational Directives & Behavioral Guardrails

This document establishes immutable operational directives for all AI coding agents working on the **AI Engineering Supervisor Control Plane** repository.

---

# SECTION 1: PERMANENT PROJECT RULES (APPLY ACROSS ALL PHASES)

1. **Source Registry Mandate**:
   - Never implement any subsystem without consulting `docs/sources/SOURCE_REGISTRY.md` and `docs/sources/REUSE_MATRIX.md`.
   - Never duplicate an upstream capability already provided by an active upstream (e.g., Agent Orchestrator, Antigravity CLI).

2. **Architecture Immutability & ADR Requirement**:
   - The canonical architecture documented in `docs/04_ARCHITECTURE.md` cannot be modified implicitly.
   - Any architectural modification, technology adoption, or upstream boundary adjustment requires an approved Architecture Decision Record (`docs/adr/`).

3. **Proposal System for New Ideas**:
   - If an agent discovers a new capability, optimization, or alternative during execution, **DO NOT IMPLEMENT IT**.
   - Create a structured proposal in `docs/proposals/PROPOSAL-xxx.md` following `docs/24_CHANGE_GOVERNANCE.md`.

4. **Task Scope Immutability**:
   - Every task dispatched to a worker is governed by an immutable **Task Contract** (`docs/08_TASK_CONTRACT.md`).
   - Workers must not expand their assigned task scope, alter files outside `allowed_scope`, or touch files in `forbidden_scope`.
   - If a blocker or architectural defect is discovered, transition the task to `BLOCKED` and stop.

5. **Evidence Over Claims**:
   - A worker report (`docs/09_WORKER_REPORT.md`) is a set of **claims**, not verified truth.
   - The Supervisor must independently verify all claims using Git diffs, test logs, exit codes, and artifact hashes before approval.

6. **State Resides Outside Chat**:
   - Conversation history is volatile context, not an authoritative database.
   - System state, task states, pair bindings, and review decisions reside exclusively in the Supervisor Control Plane and Git repository.

7. **Decision Hierarchy (Strict Governance)**:
   Whenever a conflict arises, resolve it using the immutable 9-level priority hierarchy defined in `docs/24_CHANGE_GOVERNANCE.md`:
   1. Confirmed User Requirement
   2. Approved ADR
   3. Canonical Architecture
   4. Requirement Specification
   5. Approved Roadmap
   6. Task Contract
   7. Source Repo / Reference
   8. Worker Suggestion
   9. Chat Conversation

8. **Mark Unverified Assumptions**:
   - Any assumption not backed by confirmed evidence must be explicitly marked: `GIẢ ĐỊNH:` / `ASSUMPTION:`.

9. **Pre- and Post-Execution Protocol**:
   - **Before implementation**: Read `docs/18_CURRENT_STATE.md`, assigned Task Contract, architectural references, and module provenance.
   - **After implementation**: Validate, collect evidence, generate report, and **STOP**. Do not automatically proceed to the next task.

---

# SECTION 2: CURRENT PHASE RULES — P03 TASK-P03-001 REVISION 2

> [!CRITICAL]
> Phase P01 runtime proof activity is **COMPLETE**.
> Phase P02 domain and state storage core implementation is **COMPLETE** and approved by External Supervisor final audit (`P02_FINAL_AUDIT = EXTERNAL_AUDIT_APPROVED`, baseline commit `b073a1969d0b64d151e96f0d9b59b005d4cf518a`, evidence commit `7375bbe0c0f17ac6cc50f87a319cb936caf64905`).
> Core directions of `PROPOSAL-P03-001` are **EXTERNAL_APPROVED** by External Supervisor decision.
> Canonical specifications (`docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`) have been reconciled under Revision 3 and approved by External Supervisor (`P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`).
> The active phase gate is **`EXTERNAL_SUPERVISOR_P03_TASK_001_REAUDIT`**.
> Production coding is **`HELD_FOR_EXTERNAL_AUDIT`** (`TASK_P03_001 = REVISION_2_READY_FOR_EXTERNAL_AUDIT`, `TASK_P03_002 = NOT_RELEASED`).

1. **TASK-P03-001 SCOPE & BOUNDARY**:
   - Scope is strictly confined to implementing the AO Transport Foundation & Compatibility Read Model in `internal/ao`.
   - Allowed operations: Loopback HTTP client (`http://127.0.0.1:{port}` or `http://[::1]:{port}`), IP literal loopback validation with explicit port, caller-supplied `*http.Client` injection without default timeouts, redirect fail-closed enforcement, preflight health/readiness checks (`/healthz`, `/readyz`), agent harness inventory & named readiness (`/api/v1/agents`, `/api/v1/agents/readiness`), OpenAPI schema signal (`/api/v1/openapi.yaml`), project registration & retrieval with safe URL escaping and degraded projection (`POST /api/v1/projects`, `GET /api/v1/projects/{id}`), read-only session inspection (`GET /api/v1/sessions/{id}`) with canonical activity state validation, sanitized protocol error reporting, and transport error cause preservation.

2. **EXPLICITLY FORBIDDEN IN TASK-P03-001**:
   - Absolutely NO session spawn/mutation (`POST /api/v1/sessions` belongs to `TASK-P03-002`);
   - Absolutely NO session prompt dispatch (`POST /api/v1/sessions/{id}/send` belongs to `TASK-P03-002`);
   - Absolutely NO session termination or restore (`POST /api/v1/sessions/{id}/kill`, `POST /api/v1/sessions/{id}/restore` belong to `TASK-P03-002`);
   - Absolutely NO session activity polling loop (`TASK-P03-003`);
   - Absolutely NO raw session workspace file fetch (`TASK-P03-004`);
   - Absolutely NO `StateStore` dependency or direct SQLite access inside `internal/ao` (`AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`);
   - Absolutely NO modification of `internal/domain`, `internal/workflow`, `internal/store`, `internal/contract`, `internal/audit`;
   - Absolutely NO dependency additions (`go.mod`, `go.sum`);
   - Absolutely NO direct Antigravity CLI (`agy`) invocation code;
   - Absolutely NO ConPTY process management code;
   - Absolutely NO custom Git worktree management code;
   - Absolutely NO AO internal SQLite database access;
   - Absolutely NO `/mux` terminal stream scraping or parsing;
   - Absolutely NO synthetic worker heartbeat generation;
   - Absolutely NO default operational timeout decisions (the 8 Supervisor operational policy values remain UNSET).

3. **TASK-P03-002 CODE GUARD**:
   - Session mutation and lifecycle command implementation remains strictly **HELD** (`TASK_P03_002 = NOT_RELEASED`).
   - Implementation of `TASK-P03-002` becomes authorized only upon formal audit approval and task contract release by External Supervisor.

4. **FINAL REVISION-2 GOVERNANCE STATE**:
   - `P03_CODE = HELD_FOR_EXTERNAL_AUDIT`
   - `TASK_P03_001 = REVISION_2_READY_FOR_EXTERNAL_AUDIT`
   - `TASK_P03_002 = NOT_RELEASED`
   - `ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_001_REAUDIT`
