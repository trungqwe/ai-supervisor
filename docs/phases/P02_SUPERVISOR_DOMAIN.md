# PHASE SPECIFICATION: P02 — SUPERVISOR DOMAIN CORE

> **Phase Status**: READY_FOR_DECISION_GATE
> **Code Status**: NOT_STARTED
> **Active Gate**: P02_IMPLEMENTATION_DECISION_GATE

---

## 0. Phase Entry Prerequisites
Before any application code or framework initialization begins in Phase P02, the following governance prerequisites must be satisfied:
1. **Architecture V2.1 Frozen**: Tag `phase1-architecture-v2.1` applied and verified (`phase1-architecture-v2` preserved as immutable historical snapshot superseded by reaudit 001).
2. **Phase P01 Complete**: All four proof tracks approved (`P01-A`, `P01-B`, `P01-C` with `ADR-011`, `P01-D`), with contract reconciliation finalized under `ADR-012`.
3. **Language Decision Required (Q3)**: Supervisor Core Implementation Language decision finalized via approved ADR under Change Governance.
4. **Storage Engine Decision Required (Q4)**: Local State Store Storage Engine decision finalized via approved ADR under Change Governance.

---

## 1. Objective
Implement the core domain model, 13-state workflow state machine, immutable Task Contract validator, and durable local State Store without external transport or adapter dependencies.

## 2. Deliverables
The deliverables of Phase P02 encompass only the domain entities, aggregates, and value types required by the core state machine:
- **Domain Entities & Value Objects**:
  - `Project`
  - `Pair`
  - `Task`
  - `TaskContract` (with `contract_id`, `task_id`, `revision_number`, `supersedes_contract_id` per ADR-012)
  - `TaskAttempt` (with `attempt_id`, `attempt_number`, `task_id`, `contract_id`, `expected_report_path` per ADR-012)
  - `WorkerClaim`
  - `Evidence`
  - `ReviewDecision` & associated value types
- **State Machine**:
  - 13-state `StateMachine` engine implementing all 25 canonical allowed transitions with 100% edge parity against `docs/06_WORKFLOW_STATE_MACHINE.md`.
- **Validation**:
  - Immutable `TaskContractValidator` enforcing `task-contract.schema.json`.
- **Persistence**:
  - Local `StateStore` abstraction and durable implementation satisfying NFR-003.
- **Unit Test Suite**:
  - 100% state coverage covering all 25 allowed transitions and representative forbidden transitions.

### Explicit Phase Boundaries (Anti-Scope-Creep):
- `ReviewBundleBuilder` implementation remains Phase **P04** (Evidence & Review Engine).
- `AOAdapter` implementation remains Phase **P03** (AO Adapter).
- `ChatGPTToolSurface` and transport integration remain Phase **P05** (ChatGPT Transport & Tools).

## 3. Exit Gate
- Unit tests verify that task contracts are immutable post-dispatch, revision cycles produce new contract revisions per ADR-012, and all 13 states transition strictly according to `docs/06_WORKFLOW_STATE_MACHINE.md`.