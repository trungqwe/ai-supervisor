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

# SECTION 2: CURRENT PHASE RULES (PHASE 0 — ARCHITECTURE FREEZE)

> [!CRITICAL]
> The following restrictions are **STRICTLY ACTIVE** during Phase 0. They remain in force until the User and External Supervisor explicitly authorize transition to Phase 1 following external audit review.

1. **NO APPLICATION CODE**:
   - Absolutely no backend, frontend, API server, or CLI runtime implementation code may be written.
   - No `.go`, `.ts`, `.js`, `.py`, `.rs`, `.sql`, or framework scaffolding files may be created.
   - Permitted artifacts: Markdown (`.md`), JSON Schemas (`.json`), Mermaid diagrams, and configuration files.

2. **NO VALIDATION SCRIPTS OR CODE CREATION**:
   - Do not write custom PowerShell (`.ps1`), Python, Node, or shell scripts for validation.
   - Consistency validation must use native environment inspection and direct verification.

3. **NO PACKAGE / DEPENDENCY INSTALLATION**:
   - Do not install libraries or packages (`npm`, `pip`, `go get`, `cargo`) under any circumstance.
   - Do not install third-party JSON Schema validators in Phase 0.

4. **PRESERVATION OF EXISTING ASSETS**:
   - Never overwrite or delete existing local files without explicit instruction.
   - Never force-push (`--force`) to the Git remote repository.

5. **AUDIT VERDICT RESTRAINT**:
   - The agent performing Phase 0 work cannot self-declare `ARCHITECTURE_FROZEN`.
   - The agent's internal audit may only conclude `PHASE0_READY_FOR_EXTERNAL_AUDIT` or `PHASE0_BLOCKED`.
