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

# SECTION 2: CURRENT PHASE RULES — P03 TASK-P03-003 ADR-016 REVISION 1

> [!CRITICAL]
> Phase P01 runtime proof activity is **COMPLETE**.
> Phase P02 domain and state storage core implementation is **COMPLETE** and approved by External Supervisor final audit (`P02_FINAL_AUDIT = EXTERNAL_AUDIT_APPROVED`, baseline commit `b073a1969d0b64d151e96f0d9b59b005d4cf518a`, evidence commit `7375bbe0c0f17ac6cc50f87a319cb936caf64905`).
> Core directions of `PROPOSAL-P03-001` are **EXTERNAL_APPROVED** by External Supervisor decision.
> Canonical specifications (`docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`) have been reconciled under Revision 3 and approved by External Supervisor (`P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`).
> Task TASK-P03-001 is **EXTERNAL_AUDIT_APPROVED** (`docs/audits/P03_TASK_001_EXTERNAL_REAUDIT_002.md`, commit `e5d3337772f4243ca0f85b4105430d1aa9329bd3`).
> Task TASK-P03-002 is **EXTERNAL_AUDIT_APPROVED** (`docs/audits/P03_TASK_002_EXTERNAL_REAUDIT_001.md`, commit `b85ddf686b7432f087ef776444eaabe183176e67`), and finding `P03T2R1-001` is **CLOSED**.
> External re-audit of `PROPOSAL-P03-002 Revision 3` (commit `00910f769368a28954ad03920da55a8885769ca7`) recorded findings `P03T3PR4-001` and `P03T3PR4-002` in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_003.md`.
> External re-audit of `PROPOSAL-P03-002 Revision 4` (commit `38154d7fa65888cbf25b0bc5206384198a9a8a40`) recorded findings `P03T3PR5-001`, `P03T3PR5-002`, and `P03T3PR5-003` in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_004.md`.
> External re-audit of `PROPOSAL-P03-002 Revision 5` (commit `fac2d5b3d763796077505f9f368a4b46056ec140`) recorded finding `P03T3PR6-001` in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_005.md`.
> Proposal `PROPOSAL-P03-002 Revision 6` (commit `8709af4b6aaf8613c69a10514971e320e3f90048`) is **EXTERNAL_APPROVED** in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_006.md`.
> External audit of ADR-016 draft (commit `928b0d110058d8ca90d8d9b68d9aa478c1ee089a`) recorded findings `ADR16R1-001` through `ADR16R1-009` with verdict `ADR_016 = REVISION_1_REQUIRED` in `docs/audits/P03_ADR_016_EXTERNAL_AUDIT.md`.
> ADR-016 has been remediated to Revision 1 (`ADR_016 = REVISION_1_READY_FOR_EXTERNAL_REAUDIT`, `ADR_016_ACCEPTANCE = NOT_GRANTED`).
> Architecture classification established: `P03_ARCHITECTURE_CHANGE = YES`, `P03_ADR_REQUIRED = YES`.
> Production coding remains strictly **`HELD_PENDING_ADR_016_APPROVAL`** (`TASK_P03_003 = NOT_RELEASED`).
> The active phase gate is **`EXTERNAL_SUPERVISOR_P03_ADR_016_REAUDIT_001`**.

1. **TASK-P03-003 ADR-016 REVISION 1 SCOPE**:
   - Scope is strictly confined to ADR-016 Revision 1 remediation and governance tracking.
   - Absolutely ZERO production code implementation (`internal/**/*.go`);
   - Absolutely ZERO schema or database migration implementation;
   - Absolutely ZERO canonical document mutation.

2. **TASK-P03-003 PRODUCTION CODE GUARD**:
   - Production coding of TASK-P03-003 remains strictly **HELD** (`TASK_P03_003 = NOT_RELEASED`).
   - Implementation becomes authorized only upon formal External Supervisor acceptance of ADR-016, canonical specification reconciliation, and an immutable TASK-P03-003 Task Contract release.

3. **EXPLICITLY FORBIDDEN IN THIS STAGE**:
   - Absolutely NO polling loops, tickers, or background worker threads;
   - Absolutely NO StateStore mutations or new tables;
   - Absolutely NO invented numeric operational timeout or poll interval defaults;
   - Absolutely NO synthetic worker heartbeat generation;
   - Absolutely NO SSE or `/api/v1/events` integration;
   - Absolutely NO workspace file fetching or WorkerReport handling;
   - Absolutely NO canonical doc mutation.

4. **FINAL GOVERNANCE STATE**:
   - `P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_001 = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_002 = EXTERNAL_AUDIT_APPROVED`
   - `PROPOSAL_P03_002 = EXTERNAL_APPROVED`
   - `P03_ARCHITECTURE_CHANGE = YES`
   - `P03_ADR_REQUIRED = YES`
   - `ADR_016 = REVISION_1_READY_FOR_EXTERNAL_REAUDIT`
   - `ADR_016_ACCEPTANCE = NOT_GRANTED`
   - `TASK_P03_003 = NOT_RELEASED`
   - `P03_CODE = HELD_PENDING_ADR_016_APPROVAL`
   - `ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_ADR_016_REAUDIT_001`
