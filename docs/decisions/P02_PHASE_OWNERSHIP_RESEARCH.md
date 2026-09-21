# P02 Phase Ownership Research & Reconciliation Dossier

> **Authority**: P02 Pre-Code Implementation Decision Gate (Reaudit Remediation)
> **Date**: 2026-09-22
> **Status**: RESEARCH_COMPLETE_PENDING_EXTERNAL_DECISION
> **Reference Document**: `docs/21_TRACEABILITY_MATRIX.md`

---

## 1. Architectural Scope Principles & Governance Rules

The canonical roadmap establishes an upper-level phase progression:
- **P02 — Supervisor Domain Core**: Headless domain entities, finite state machine, task contract schema validation, local durable state store, crash recovery reconciler, and append-only audit event emission.
- **P03 — Agent Orchestrator Integration**: `AOAdapter`, external worker session lifecycle, process supervision, event stream observation, and workspace report retrieval.
- **P04 — Evidence & Review Engine**: Independent `EvidenceCollector`, Git base/head diff analysis, `VerificationRunner` (host-owned profile execution), `ReviewBundleBuilder`, and scope policy enforcement.
- **P05 — ChatGPT Tool Interface & Transport**: High-level tool surface (`ToolSurface`), Fastify/MCP server, loopback transport, and end-to-end autonomous supervision loop.
- **P06 — Production Packaging & Observability**: Standalone CLI distribution, operator controls, and web UI.

### Architectural Invariants:
1. **No Premature Upstream Integration in P02**: P02 must not absorb `AOAdapter` implementation (owned by P03).
2. **No Evidence or Bundle Assembly in P02**: P02 must not absorb `EvidenceCollector`, `VerificationRunner`, or `ReviewBundleBuilder` (owned by P04).
3. **No Transport or Model Surface in P02**: P02 must not absorb `ToolSurface`, Fastify daemon server, or MCP transport layers (owned by P05).
4. **Structural Distinction**: We distinguish:
   - **Domain Types & Entities**: Immutable data definitions required by StateStore schemas and StateMachine transitions (owned in P02).
   - **Domain Services**: Business logic operating strictly within the core domain (owned in P02).
   - **Infrastructure Adapters**: External I/O implementations (Git, AO REST API, child-process runners) (owned in P03/P04).
   - **Transport Exposure**: Exposing operations over HTTP/MCP tools (owned in P05).

---

## 2. Comprehensive P02 Requirement Traceability Audit

The following table evaluates every requirement mapped to Phase P02 in canonical `docs/21_TRACEABILITY_MATRIX.md`:

| Requirement ID | Canonical Name | Canonical Target Module | Current Phase in Matrix | Required for P02 Core? | Recommended Owning Phase | Architectural Rationale & Scope Boundaries | Dependencies |
|---|---|---|:---:|:---:|:---:|---|---|
| **FR-001** | Project Registration | `ProjectRegistry` | P02 | **PARTIAL** | **P02** (Entity/Schema) / **P05** (Service/CLI) | The `Project` domain entity and StateStore table are required in P02 to associate tasks with workspaces. Dynamic project registration via MCP/CLI belongs to P05/P06. | Local StateStore |
| **FR-002** | Pair Binding | `PairRegistry` | P02 | **PARTIAL** | **P02** (Entity/Schema) / **P05** (Service) | The `PairBinding` domain entity and 1:1 active constraint must be modeled in P02 StateStore. The interactive binding handshake and session token binding occur in P05. | `ProjectRegistry`, StateStore |
| **FR-003** | Context Access | `ContextEngine` | P02 | **NO** | **P05** | Read-only workspace indexing (code outlines/symbols) is not required for core task state transitions. It serves as an advisory tool for ChatGPT in P05. | Host filesystem / Tree-sitter |
| **FR-004** | Task Contract | `TaskContractManager` | P02 | **YES** | **P02** | Task contract schema validation (draft-07 / 2020-12), immutability enforcement, revision number progression, and pre-dispatch validation are fundamental to P02. | JSON Schema Validator |
| **FR-011** | Supervisor Decision | `StateMachine` | P02 | **YES** | **P02** | The 13-state machine, transition guards, attempt lifecycle binding, and state change persistence must be implemented and verified in P02. | `StateStore` |
| **FR-013** | Audit Trail | `AuditLogger` | P02 | **PARTIAL** | **P02** (Core Store) / **P04** (CLI/Export) | Emitting append-only, tamper-evident audit records on every state transition and contract revision is core to P02. Formatting and CLI log export tools belong to P04/P05. | `StateStore`, JSON serializer |
| **FR-014** | Multi-Project Domain | `DomainModel` | P02 | **YES** | **P02** | Multi-project schema partitioning and foreign key isolation in SQLite ensure tasks cannot cross project boundaries. | `StateStore` |
| **NFR-003** | Recoverability | `StateStore` | P02 | **YES** | **P02** | Reconciling interrupted or uncommitted states upon daemon restart (e.g. recovering inflight attempts) is a mandatory P02 core capability. | SQLite WAL mode |
| **NFR-004** | Auditability | `AuditLogger` | P02 | **YES** | **P02** | Verifying cryptographic or append-only integrity of state history across attempts must be validated in P02. | `AuditLogger` |
| **SEC-002** | Path Containment | `PolicyEngine` | P02 | **PARTIAL** | **P02** (Contract Scope) / **P04** (Evidence Check) | Pre-dispatch validation of `allowed_scope` and `forbidden_scope` paths against project root belongs to P02. Post-execution diff violation checking belongs to P04. | Path normalization library |
| **SEC-004** | Secret Sanitization | `AuditLogger` | P02 | **YES** | **P02** | Regex and heuristic token scrubbing for audit records and persisted error payloads must be active in P02 to prevent secret leakage. | Regular expression engine |
| **OPS-002** | Clean Termination | `SupervisorCore` | P02 | **YES** | **P02** | Graceful daemon shutdown, releasing file locks, completing active transactions, and closing SQLite cleanly upon SIGTERM/Ctrl+C. | Process signal handling |
| **OPS-003** | Self-Contained Store | `StateStore` | P02 | **YES** | **P02** | Local SQLite engine requiring zero external cloud services, daemons, or network round-trips. | SQLite runtime driver |

---

## 3. Detailed Component Decomposition & Boundary Analysis

### 3.1 `ProjectRegistry` & `PairRegistry`
- **P02 Scope**: Define `Project` and `PairBinding` domain models; implement SQL schema (`projects`, `pair_bindings`); enforce relational foreign keys and uniqueness constraints (`workspace_path` unique, active pair binding 1:1).
- **Deferred to P05**: HTTP/MCP tool surface for registering projects dynamically, listing projects, and managing session pairings.

### 3.2 `ContextEngine`
- **Audit Finding**: Currently mapped to P02 in `docs/21_TRACEABILITY_MATRIX.md`.
- **Recommendation**: Defer full implementation to **P05**.
- **Rationale**: `ContextEngine` generates structural code outlines and repository summaries to assist ChatGPT during task formulation. It has zero interaction with the Task Attempt State Machine or StateStore invariants. Pulling it into P02 expands scope with AST parsing / symbol extraction before core domain persistence is proven.

### 3.3 `TaskContractManager` & `PolicyEngine` (Pre-Dispatch Scope)
- **P02 Scope**: Validate incoming `TaskContract` revision JSON against schema; enforce immutability; ensure `base_sha` matches revision 1 baseline; validate that all paths in `allowed_scope` and `forbidden_scope` resolve strictly within the project boundary without traversal (`..` or symlink escapes).
- **Deferred to P04**: `PolicyEngine` post-execution evaluation (comparing Git diff touch list against `allowed_scope` / `forbidden_scope`).

### 3.4 `StateMachine` & `SupervisorCore`
- **P02 Scope**:
  - Implement the 13 canonical states and 25 transitions.
  - Enforce the atomic pre-dispatch persistence invariant: allocate `TaskAttempt` and commit `READY → DISPATCHED` in SQLite before any external side effects occur.
  - Implement restart reconciliation: upon startup, detect dangling `DISPATCHED` attempts and transition them or flag them for reconciliation.
  - Implement graceful termination on OS interrupt (`SIGINT`, `SIGTERM`), closing SQLite connections cleanly.

### 3.5 `AuditLogger` & `SEC-004`
- **P02 Scope**: Append-only event logging with timestamp, correlation ID (`task_id`, `contract_id`, `attempt_id`), and sanitization of sensitive environment variables or tokens.

---

## 4. Reconciliation Status & Matrix Alignment

- **Action Taken**: Analysis complete.
- **Traceability Matrix Status**: `docs/21_TRACEABILITY_MATRIX.md` remains strictly unmodified in this task.
- **External Approval Required**: Following External Supervisor approval of this research dossier, a targeted update to `docs/21_TRACEABILITY_MATRIX.md` will formally split `FR-001`, `FR-002`, `FR-003`, `FR-013`, and `SEC-002` across their respective domain and integration phases.
