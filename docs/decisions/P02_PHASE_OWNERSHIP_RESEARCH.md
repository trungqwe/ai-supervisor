# P02 Phase Ownership Research & Scope Reconciliation Dossier

> **Authority**: P02 Final Implementation Decision Canonicalization
> **Date**: 2026-09-22
> **Status**: DECISION_ACCEPTED
> **Scope Verdict**: `APPROVED_WITH_SCOPE_CORRECTIONS`
> **Reference Document**: `docs/21_TRACEABILITY_MATRIX.md`

---

## 1. Final Approved Canonical Phase Boundaries

Governance has formally finalized the phase ownership boundaries across the six roadmap phases:

### 1.1 Phase P02 — Supervisor Domain Core (Scope Boundary)
P02 is strictly headless and self-contained, owning the following core modules and domain responsibilities:
- **Domain Entities & Value Types**:
  - `Project`, `Pair`, `Task`, `TaskContract`, `TaskAttempt`, `WorkerClaim`.
  - Core `Evidence` value definitions and persistence schemas required by the internal model.
  - `ReviewDecision` and associated state transition payload types.
- **Domain Services**:
  - `StateMachine`: 13 canonical states, 25 allowed transitions.
  - `TaskContractValidator`: Schema validation (`task-contract.schema.json`), immutability enforcement, revision number monotonic increments, baseline `base_sha` immutability.
  - Pre-dispatch validation: Pre-dispatch `allowed_scope` / `forbidden_scope` path containment checks (SEC-002).
  - Pre-dispatch allocation: `TaskAttempt` allocation and atomic commit of `READY → DISPATCHED` before external AO calls.
- **Local StateStore (SQLite)**:
  - DDL schemas, migration engine (`PRAGMA user_version`), foreign key constraints (`foreign_keys = ON`).
  - ACID transactions with `PRAGMA journal_mode = WAL;` and `PRAGMA synchronous = FULL;`.
  - `TaskContract` and `TaskAttempt` relational lineage graph.
  - Restart recovery classification: Detect dangling `DISPATCHED` attempts upon startup and classify them as `EXTERNAL_RECONCILIATION_REQUIRED`.
  - Component shutdown API: Deterministic StateStore close, active transaction rollback, and connection cleanup (OPS-002).
- **Audit Core**:
  - `AuditEvent` model, append-only persistence in SQLite.
  - Secret sanitization: Automated token and credential scrubbing on persisted audit records, domain events, and error payloads (SEC-004).

### 1.2 Subsystems Explicitly Excluded from P02
- **Phase P03 (AO Integration)**:
  - `AOAdapter` implementation and REST API client.
  - External worker session lifecycle management and process spawning.
  - AO liveness monitoring and event stream consumption.
  - AO-backed restart reconciliation (querying AO for live session status to resolve `EXTERNAL_RECONCILIATION_REQUIRED`).
- **Phase P04 (Evidence & Review Engine)**:
  - `EvidenceCollector` implementation (Git diff calculation, tree inspection).
  - `VerificationRunner` implementation (Layer A command construction and Layer B execution isolation with Windows Job Objects).
  - Post-execution scope violation checking (comparing Git diff against `allowed_scope` / `forbidden_scope`).
  - `ReviewBundleBuilder` assembly and review policy engine.
- **Phase P05 (ChatGPT Tool Interface & Transport)**:
  - Model Context Protocol (MCP) server over Streamable HTTP and loopback tunnel transport.
  - `ToolSurface` high-level tool definitions and parameter mapping.
  - Interactive project/pair registration and binding tools (`register_project`, `bind_pair`, `list_projects`).
  - `ContextEngine` workspace AST indexing and outline extraction (FR-003).
  - Process-level lifecycle management: Host daemon process control, Windows console handling, OS signals (`SIGINT`, `SIGTERM`), and HTTP/MCP server shutdown (OPS-002).
  - Bounded audit record querying for ChatGPT tools.
- **Phase P06 (Packaging & Observability)**:
  - Operator UI / CLI packaging and distribution (`supervisor.exe`).
  - Human-facing dashboards, audit log export utilities, and metrics visualization.

---

## 2. Traceability Matrix Audit & Multi-Phase Responsibility Split

The 13 requirements mapped to Phase P02 in `docs/21_TRACEABILITY_MATRIX.md` are reconciled as follows:

| Requirement ID | Canonical Name | Canonical Target Module | Final Owning Phase(s) | Phase Split & Scope Delineation | Dependencies |
|---|---|---|:---:|---|---|
| **FR-001** | Project Registration | `ProjectRegistry` | **P02 / P05** | **P02**: `Project` entity, SQLite schema, workspace path uniqueness.<br/>**P05**: Interactive registration CLI/MCP tools. | Local StateStore |
| **FR-002** | Pair Binding | `PairRegistry` | **P02 / P05** | **P02**: `PairBinding` entity, relational foreign keys, 1:1 active constraint.<br/>**P05**: Interactive binding handshake & token validation. | `ProjectRegistry`, StateStore |
| **FR-003** | Context Access | `ContextEngine` | **P05** | **Moved to P05**: Read-only workspace AST/outline extraction serves as an advisory tool for ChatGPT. Zero P02 domain dependency. | Host filesystem / Tree-sitter |
| **FR-004** | Task Contract | `TaskContractManager` | **P02** | **P02 Core**: JSON Schema validation, immutability, revision lineage, baseline `base_sha` immutability, and `verification_requests` structure (ADR-013). | JSON Schema Validator |
| **FR-011** | Supervisor Decision | `StateMachine` | **P02** | **P02 Core**: 13-state machine, 25 canonical transitions, attempt lifecycle binding, atomic pre-dispatch commitment. | `StateStore` |
| **FR-013** | Audit Trail | `AuditLogger` | **P02 / P05 / P06** | **P02**: Append-only `AuditEvent` persistence & correlation.<br/>**P05**: Bounded audit read tool for ChatGPT.<br/>**P06**: Operator export tooling & log visualization. | `StateStore`, JSON serializer |
| **FR-014** | Multi-Project Domain | `DomainModel` | **P02** | **P02 Core**: Multi-project schema partitioning and foreign key isolation in SQLite. | `StateStore` |
| **NFR-003** | Recoverability | `StateStore` | **P02 / P03** | **P02**: Detects dangling `DISPATCHED` attempts on restart, classifies `EXTERNAL_RECONCILIATION_REQUIRED` in StateStore.<br/>**P03**: Queries AO REST API to reconcile session state. | SQLite WAL mode |
| **NFR-004** | Auditability | `AuditLogger` | **P02** | **P02 Core**: Append-only tamper-evident event log integrity. | `AuditLogger` |
| **SEC-002** | Path Containment | `PolicyEngine` | **P02 / P04** | **P02**: Pre-dispatch validation that `allowed_scope` / `forbidden_scope` paths reside within project boundaries.<br/>**P04**: Post-execution Git diff scope checking. | Path normalization library |
| **SEC-003** | No Arbitrary Shell | `ToolSurface` / `VerificationRunner` | **P04 / P05** | **P04**: VerificationRunner executing host-owned profiles with Layer B execution isolation (ADR-013).<br/>**P05**: Zero shell tools exposed to ChatGPT. | ADR-013 |
| **SEC-004** | Secret Sanitization | `AuditLogger` | **P02** | **P02 Core**: Automated token and credential scrubbing regex for all persisted audit, domain, and error payloads. | Regex engine |
| **OPS-002** | Clean Termination | `SupervisorCore` | **P02 / P05** | **P02**: Deterministic StateStore close, active transaction abort, and component shutdown API.<br/>**P05**: OS process lifecycle, Windows console handling, SIGINT/SIGTERM daemon termination. | Process signal handling |
| **OPS-003** | Self-Contained Store | `StateStore` | **P02** | **P02 Core**: Zero-cloud local SQLite engine with WAL mode, `synchronous = FULL`, and Online Backup API. | SQLite runtime driver |

---

## 3. Implementation Release Verdict

- **Phase P02 Scope**: Strictly bounded to domain entities, state machine, contract validation, SQLite store, and audit core.
- **Traceability Matrix**: Reconciled in `docs/21_TRACEABILITY_MATRIX.md`.
- **Status**: **`APPROVED_WITH_SCOPE_CORRECTIONS`**.

### 3.6 `VerificationPolicyCatalog` (Pure Domain Policy Boundary)
- **P02 Scope**: Expose pure domain abstraction `VerificationPolicyCatalog` and `VerificationProfilePolicy` domain metadata. Enables `TaskContractValidator` to validate profile existence, parameter schemas, cwd policies, and timeout bounds without importing or depending on Phase P04 execution runner logic. P02 unit tests supply an in-memory fake catalog.
- **Deferred to P04**: Concrete `VerificationRunner` implementation, host registry executable resolution, argument construction, Windows Job Object assignment, and execution isolation.
