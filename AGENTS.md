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

# SECTION 2: CURRENT PHASE RULES (PHASE 1 — UPSTREAM PROOF)

> [!CRITICAL]
> Phase 0 is **FROZEN** (`docs/audits/PHASE0_FREEZE_RECORD.md`). The following rules govern **Phase 1 (Upstream Proof)**.  
> This Phase-1 governance change does **NOT** authorize application coding. Application/source implementation remains prohibited until a later User-authorized implementation phase.

1. **NO SUPERVISOR APPLICATION IMPLEMENTATION YET**:
   - Absolutely no Supervisor backend, frontend, API server, or CLI runtime implementation code may be written.
   - No application/source code (`.go`, `.ts`, `.js`, `.py`, `.rs`, `.sql`) or scaffolding files may be created.

2. **NO AOADAPTER IMPLEMENTATION YET**:
   - Do not implement custom adapter packages or translation layers during Phase 1.

3. **NO CHATGPT TRANSPORT IMPLEMENTATION YET**:
   - Do not implement ChatGPT transport abstractions, connectors, or MCP server code yet.

4. **RUNTIME PROOF AND EVIDENCE COLLECTION ONLY**:
   - Permitted activities in Phase 1 are strictly limited to running isolated runtime proofs and collecting empirical evidence against upstream tools.

5. **USE PINNED AO/AGY BASELINES**:
   - All proofs must evaluate the pinned upstream baselines (AO `v0.13.0` commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`, Agy `1.2.7` commit `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`) unless a proof explicitly concerns upstream version compatibility.

6. **NO SILENT UPSTREAM UPGRADES**:
   - Do not upgrade upstream versions without explicit governance review and documented ADR.

7. **NO WORKAROUND CODE**:
   - Do not write custom integration code or workarounds to force a failed proof to pass.

8. **FAILURE PROTOCOL — GAP_REQUIRES_ADR OR BLOCKER**:
   - If an upstream capability fails or behaves differently than required, record either `GAP_REQUIRES_ADR` or `BLOCKER`. Do not attempt workarounds.

9. **FOUR INDEPENDENT TRACKS**:
   - Execution is strictly organized into four independent tracks:
     * **P01-A**: AO Runtime Proof
     * **P01-B**: Direct Antigravity CLI Capability Proof
     * **P01-C**: AO ↔ Agy Structured Completion / WorkerReport Proof
     * **P01-D**: ChatGPT Plus Transport Feasibility Proof

10. **STOP AFTER PROOF RESULTS**:
    - Record literal evidence in the corresponding Phase 1 audit dossier and **STOP**. Do not automatically proceed to Phase P02.
