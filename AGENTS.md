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

# SECTION 2: CURRENT PHASE RULES — P03 TASK-P03-002 REVISION 1

> [!CRITICAL]
> Phase P01 runtime proof activity is **COMPLETE**.
> Phase P02 domain and state storage core implementation is **COMPLETE** and approved by External Supervisor final audit (`P02_FINAL_AUDIT = EXTERNAL_AUDIT_APPROVED`, baseline commit `b073a1969d0b64d151e96f0d9b59b005d4cf518a`, evidence commit `7375bbe0c0f17ac6cc50f87a319cb936caf64905`).
> Core directions of `PROPOSAL-P03-001` are **EXTERNAL_APPROVED** by External Supervisor decision.
> Canonical specifications (`docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`) have been reconciled under Revision 3 and approved by External Supervisor (`P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`).
> Task TASK-P03-001 is **EXTERNAL_AUDIT_APPROVED** (`docs/audits/P03_TASK_001_EXTERNAL_REAUDIT_002.md`, commit `e5d3337772f4243ca0f85b4105430d1aa9329bd3`).
> External audit of commit `056629ff4d38bbf4caaa94fc860825a2e9991a7c` identified finding `P03T2R1-001` (`ACTIVITY_PROTOCOL_ERROR_STATUS_MISREPORTED`).
> Revision 1 remediates finding `P03T2R1-001` by making `validateCanonicalActivityState` status-aware and preserving observed HTTP response status (`201` on spawn, `200` on get/restore).
> The active phase gate is **`EXTERNAL_SUPERVISOR_P03_TASK_002_REAUDIT`**.
> Production coding is **`HELD_FOR_EXTERNAL_AUDIT`** (`TASK_P03_002 = REVISION_1_READY_FOR_EXTERNAL_REAUDIT`, `TASK_P03_003 = NOT_RELEASED`).

1. **TASK-P03-002 REVISION 1 SCOPE & BOUNDARY**:
   - Scope is strictly confined to fixing finding `P03T2R1-001` (preserving exact observed HTTP status code in activity state protocol errors).
   - Allowed operations:
     - `CreateWorkerSession(ctx, projectID, harness)` via `POST /api/v1/sessions` (exact HTTP 201, minimal wire contract, protocol errors report 201);
     - `DispatchTaskContract(ctx, sessionID, message)` via `POST /api/v1/sessions/{id}/send` (exact HTTP 200, immutable message transmission);
     - `StopWorker(ctx, sessionID)` via `POST /api/v1/sessions/{id}/kill` (exact HTTP 200, honest `freed` outcome reporting);
     - `ResumeWorker(ctx, sessionID)` via `POST /api/v1/sessions/{id}/restore` (exact HTTP 200, valid `restoreMode` and activity state validation, protocol errors report 200).

2. **EXPLICITLY FORBIDDEN IN TASK-P03-002**:
   - Absolutely NO session activity polling loop (`TASK-P03-003`);
   - Absolutely NO raw session workspace file fetch (`TASK-P03-004`);
   - Absolutely NO `StateStore` dependency or direct SQLite access inside `internal/ao` (`AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN`);
   - Absolutely NO Task attempt allocation, task state transition, or lifecycle event classification (`DISPATCHED`, `RUNNING`, `worker.started`, `worker.stopped`, etc.);
   - Absolutely NO modification of `internal/domain`, `internal/workflow`, `internal/store`, `internal/contract`, `internal/audit`;
   - Absolutely NO dependency additions (`go.mod`, `go.sum`);
   - Absolutely NO direct Antigravity CLI (`agy`) invocation code;
   - Absolutely NO ConPTY process management code;
   - Absolutely NO custom Git worktree management code;
   - Absolutely NO AO internal SQLite database access;
   - Absolutely NO `/mux` terminal stream scraping or parsing;
   - Absolutely NO synthetic worker heartbeat generation;
   - Absolutely NO default operational timeout decisions (the 8 Supervisor operational policy values remain UNSET).

3. **TASK-P03-003 CODE GUARD**:
   - Lifecycle polling and authoritative activity loop implementation remains strictly **HELD** (`TASK_P03_003 = NOT_RELEASED`).
   - Implementation of `TASK-P03-003` becomes authorized only upon formal audit approval and task contract release by External Supervisor.

4. **FINAL TASK-P03-002 REVISION 1 GOVERNANCE STATE**:
   - `P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_001 = EXTERNAL_AUDIT_APPROVED`
   - `P03_CODE = HELD_FOR_EXTERNAL_AUDIT`
   - `TASK_P03_002 = REVISION_1_READY_FOR_EXTERNAL_REAUDIT`
   - `TASK_P03_003 = NOT_RELEASED`
   - `ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_002_REAUDIT`
