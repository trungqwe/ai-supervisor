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

# SECTION 2: CURRENT PHASE RULES — P03 CANONICAL RECONCILIATION REVISION 3 / FINAL EXTERNAL REAUDIT

> [!CRITICAL]
> Phase P01 runtime proof activity is **COMPLETE**.
> Phase P02 domain and state storage core implementation is **COMPLETE** and approved by External Supervisor final audit (`P02_FINAL_AUDIT = EXTERNAL_AUDIT_APPROVED`, baseline commit `b073a1969d0b64d151e96f0d9b59b005d4cf518a`, evidence commit `7375bbe0c0f17ac6cc50f87a319cb936caf64905`).
> Core directions of `PROPOSAL-P03-001` are **EXTERNAL_APPROVED** by External Supervisor decision.
> Canonical specifications (`docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`) have been reconciled under Revision 3.
> The active phase gate is **`EXTERNAL_SUPERVISOR_P03_CANONICAL_REAUDIT`**.
> Production coding for Phase P03 remains strictly **HELD** pending release of the immutable `TASK-P03-001` Task Contract.

1. **PHASE P03 CANONICAL RECONCILIATION GATE BOUNDARY**:
   - Work during this gate is strictly confined to governance, architectural documentation, upstream contract alignment, and pre-code audit reconciliation.
   - Zero production Go code may be created, edited, or deleted.

2. **EXPLICITLY FORBIDDEN DURING THIS GATE**:
   - Absolutely NO `AOAdapter` production implementation;
   - Absolutely NO `net/http` production client creation;
   - Absolutely NO new P03 Go packages (`internal/ao`, `internal/adapter`, `internal/integration`, etc.);
   - Absolutely NO modification of `internal/**`;
   - Absolutely NO dependency additions (`go.mod`, `go.sum`);
   - Absolutely NO direct Antigravity CLI (`agy`) invocation code;
   - Absolutely NO ConPTY process management code;
   - Absolutely NO custom Git worktree management code;
   - Absolutely NO AO internal SQLite database access;
   - Absolutely NO `/mux` terminal stream scraping or parsing;
   - Absolutely NO lifecycle-event synthesis without approved canonical semantics.

3. **P03 PRODUCTION CODE AUTHORIZATION GUARD**:
   - Production P03 coding in this repository becomes authorized **ONLY** when:
     1. External Supervisor audits canonical P03 reconciliation and approves exit from gate `EXTERNAL_SUPERVISOR_P03_CANONICAL_REAUDIT`;
     2. External Supervisor releases an immutable Task Contract for `TASK-P03-001`;
     3. `P03_CODE` is transitioned from `HELD` to `AUTHORIZED`.
   - Until all conditions are met, `P03_CODE` remains strictly **HELD**, and no application source files (`.go`) may be created or modified.
