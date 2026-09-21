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

# SECTION 2: CURRENT PHASE RULES — P02 SUPERVISOR DOMAIN CORE

> [!CRITICAL]
> Phase P01 runtime proof activity is **COMPLETE**. The architecture baseline is frozen at **`phase1-architecture-v2.1`** (commit `62d3fe0df4a3a05697da77349ff085430ea452f7`) together with accepted amendments **ADR-012**, **ADR-013**, **ADR-014**, and **ADR-015**.
> The following rules govern **Phase P02 (Supervisor Domain Core)**.

1. **PHASE P02 STRICT DOMAIN BOUNDARY**:
   - Implementation in Phase P02 is strictly confined to the self-contained headless core:
     * `Project`, `Pair`, `Task` domain entities and relational invariants;
     * `TaskContract` (immutable revision model, baseline `base_sha` immutability, `verification_requests`);
     * `TaskAttempt` (attempt lineage, allocation);
     * `WorkerClaim` and core evidence value types required by the domain model;
     * `ReviewDecision` and associated state transition value types;
     * 13-state `StateMachine` engine with 25 canonical transitions;
     * `TaskContractValidator` enforcing schema, revision progression, and pre-dispatch path containment (SEC-002);
     * Pure validation policy interfaces required by P02 (`VerificationPolicyCatalog` / `VerificationProfilePolicy`);
     * Local SQLite `StateStore` (schema DDL, migrations, relational foreign keys, WAL mode, `synchronous = FULL`);
     * Atomic pre-dispatch intent persistence (allocating `TaskAttempt` and committing `READY → DISPATCHED` in StateStore before external calls);
     * Restart recovery classification (detecting dangling `DISPATCHED` attempts and recording `EXTERNAL_RECONCILIATION_REQUIRED` without querying AO);
     * `AuditEvent` model, append-only persistence, and automated token/credential secret scrubbing (SEC-004);
     * Deterministic StateStore component close and transaction cleanup API (OPS-002).

2. **EXPLICITLY FORBIDDEN DURING PHASE P02**:
   - Absolutely NO `AOAdapter` implementation or external AO REST client code (owned by Phase P03);
   - Absolutely NO direct Antigravity CLI (`agy`) harness integration code (owned by Phase P03);
   - Absolutely NO concrete `VerificationRunner` execution or Windows verification sandbox / Job Object runner implementation (owned by Phase P04);
   - Absolutely NO `ReviewBundleBuilder` assembly or review policy engine implementation (owned by Phase P04);
   - Absolutely NO Model Context Protocol (MCP) server, Go MCP server, or loopback transport code (owned by Phase P05);
   - Absolutely NO `ContextEngine` workspace AST / tree-sitter indexing code (owned by Phase P05);
   - Absolutely NO human-facing operator UI, CLI distribution packaging, or telemetry dashboards (owned by Phase P06).

3. **P02 CODE AUTHORIZATION GUARD**:
   - Production P02 coding in this repository is authorized **ONLY** when:
     1. `docs/18_CURRENT_STATE.md` explicitly declares: `P02_CODE = AUTHORIZED`, **AND**
     2. The active implementation task is accompanied by an explicit Task Contract / External Supervisor execution authorization.
   - Until both conditions are met, `P02_CODE` remains strictly **HELD**, and no application source files (`.go`, `go.mod`, `go.sum`) may be created.
