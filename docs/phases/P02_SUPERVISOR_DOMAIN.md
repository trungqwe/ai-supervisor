# PHASE SPECIFICATION: P02 — SUPERVISOR DOMAIN CORE

> **Phase Status**: COMPLETE
> **Code Status**: EXTERNAL_AUDIT_APPROVED
> **Active Gate**: CLOSED
> **Approved Toolchain Baseline**: Go 1.27.x (Development Baseline) / Go 1.26.x (Minimum Supported Line)
> **Approved Storage Engine**: SQLite (WAL mode, synchronous = FULL, foreign_keys = ON)

---

## 0. Phase Entry Decisions (Formally Accepted)
All architectural decisions governing Phase P02 are formally accepted and verified:
1. **Architecture V2.1 Frozen**: Baseline frozen at tag `phase1-architecture-v2.1` (commit `62d3fe0df4a3a05697da77349ff085430ea452f7`, tag object `cf8dfeed057271b6e76df443204f61254a703461`).
2. **Phase P01 Complete**: Proof tracks P01-A, P01-B, P01-C (with ADR-011), and P01-D approved.
3. **ADR-012 Accepted**: Task contract revision model, attempt binding, and baseline `base_sha` immutability established.
4. **ADR-013 Accepted**: Hybrid Profile with Structured Parameters model (`HYBRID_PROFILE_STRUCTURED_PARAMETERS`) across Layer A (command authority containment) and Layer B (untrusted code execution isolation). Contract schema migrated to `verification_requests`.
5. **ADR-014 Accepted (Q3 = GO)**: Go selected as implementation language. Baseline: `Go 1.27.x`, minimum supported line: `Go 1.26.x`. Official `github.com/modelcontextprotocol/go-sdk` using Streamable HTTP.
6. **ADR-015 Accepted (Q4 = SQLITE)**: SQLite selected as local state store with `PRAGMA journal_mode = WAL;`, `PRAGMA synchronous = FULL;`, `PRAGMA foreign_keys = ON;`, configurable bounded busy timeout, and Online Backup API / `VACUUM INTO`.
7. **P02 Phase Ownership Finalized**: Scope strictly confined to headless domain entities, state machine, contract validation, SQLite store, and audit core.

---

## 1. Objective
Implement the headless core domain model, 13-state workflow state machine, immutable Task Contract validator (supporting `verification_requests`), durable local SQLite State Store, and append-only audit event persistence with secret sanitization in Go, with zero external network or transport dependencies.

---

## 2. Approved Deliverables

### 2.1 Domain Entities & Value Types
- `Project` (workspace path, metadata, relational constraints)
- `Pair` (1:1 workspace binding constraint)
- `Task` (stable logical task identity, status)
- `TaskContract` (`contract_id`, `task_id`, `revision_number`, `supersedes_contract_id`, `base_sha`, `allowed_scope`, `forbidden_scope`, `verification_requests`)
- `TaskAttempt` (`attempt_id`, `attempt_number`, `task_id`, `contract_id`, `expected_report_path`)
- `WorkerClaim`
- Core `Evidence` persistence and value structures
- `ReviewDecision` and transition value types

### 2.2 Domain Services
- **StateMachine**:
  - Full 13-state machine implementation: 22 executable domain transitions across 13 states, and 25 total lifecycle graph edges (including 3 pseudo lifecycle edges).
  - Enforcement of pre-dispatch invariants: allocate `TaskAttempt` and atomically commit `READY → DISPATCHED` in StateStore before external calls.
- **TaskContractValidator**:
  - Schema validation against `task-contract.schema.json`.
  - Immutability enforcement: contracts cannot be mutated post-dispatch.
  - Revision lineage validation: monotonic revision numbers, valid `supersedes_contract_id`, baseline `base_sha` immutability.
  - Pre-dispatch path containment checks: verify `allowed_scope` and `forbidden_scope` reside strictly within project boundaries (SEC-002).
  - Pre-dispatch verification request semantic validation via `VerificationPolicyCatalog` (ADR-013 Layer A):
    * Pure domain interface `VerificationPolicyCatalog` (`LookupProfile(profileID) (VerificationProfilePolicy, bool)`);
    * `VerificationProfilePolicy` domain metadata exposing parameter schema, cwd policy, and max timeout;
    * Enforces `JSON SCHEMA SHAPE VALIDATION != PATH CONTAINMENT VALIDATION`, verifying `cwd` resolves either exactly as the assigned worktree root or as a descendant contained within the assigned worktree root (`TARGET == ROOT || TARGET_IS_DESCENDANT_OF_ROOT`);
    * P02 testable via in-memory fake policy catalog.

### 2.3 StateStore (Local SQLite)
- DDL schema and migration manager using `PRAGMA user_version;`.
- Enforcement of durability pragmas: `PRAGMA journal_mode = WAL;`, `PRAGMA synchronous = FULL;`, `PRAGMA foreign_keys = ON;`.
- Relational lineage storage for tasks, contracts, attempts, and audit events.
- Restart recovery classifier (NFR-003): On daemon restart, detect dangling `DISPATCHED` attempts and classify them as `EXTERNAL_RECONCILIATION_REQUIRED` without querying AO.
- Component shutdown API: Deterministic StateStore close, active transaction abort, and file lock release (OPS-002).

### 2.4 Audit Core
- `AuditEvent` model and append-only tamper-evident SQLite persistence.
- Automated secret scrubbing regex for tokens and credentials (SEC-004).

---

## 3. Explicit Phase Boundaries (Anti-Scope-Creep)
- **P03 Owns**: `AOAdapter` implementation, AO REST calls, external worker session lifecycle, and live AO restart reconciliation.
- **P04 Owns**: `EvidenceCollector`, Git diff analysis, `VerificationRunner` (Layer B execution isolation and Windows Job Object enforcement), and `ReviewBundleBuilder`.
- **P05 Owns**: Go Supervisor host / MCP server using the approved MCP Go SDK and Streamable HTTP, ChatGPT tool surface (`ToolSurface`), interactive project/pair registration/binding tools, `ContextEngine`, and process-level signal handling (`SIGINT`, `SIGTERM`).
- **P06 Owns**: Operator UI, packaging, export utilities.

---

## 4. Exit Gate
- Complete unit test suite verifying:
  - 100% of 22 executable domain transitions succeed; 25 canonical graph edges verified; representative invalid transitions fail.
  - Contract immutability and `verification_requests` validation.
  - Pre-dispatch atomic attempt allocation and `READY → DISPATCHED` persistence in SQLite.
  - Crash recovery classification on restart.
  - Secret sanitization on persisted audit records.
  - Clean StateStore shutdown and resource release.
