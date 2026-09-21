# PHASE SPECIFICATION: P02 — SUPERVISOR DOMAIN CORE

> **Phase Status**: READY_FOR_DECISION_GATE
> **Code Status**: NOT_STARTED
> **Active Gate**: P02_IMPLEMENTATION_DECISION_GATE

---

## 0. Phase Entry Prerequisites
Before any application code or framework initialization begins in Phase P02, the following governance prerequisites must be satisfied:
1. **Architecture V2 Frozen**: Tag `phase1-architecture-v2` applied and verified.
2. **Phase P01 Complete**: All four proof tracks approved (`P01-A`, `P01-B`, `P01-C` with `ADR-011`, `P01-D`).
3. **Language Decision Required (Q3)**: Supervisor Core Implementation Language decision finalized via approved ADR under Change Governance.
4. **Storage Engine Decision Required (Q4)**: Local State Store Storage Engine decision finalized via approved ADR under Change Governance.

---

## 1. Objective
Implement the core domain model, 13-state workflow state machine, immutable Task Contract validator, and durable local State Store.

## 2. Deliverables
- Domain entity classes (Project, Pair, Task, TaskContract, WorkerClaim, Evidence).
- Workflow State Machine transition engine with strict transition rules.
- Local State Store persistence layer.
- Unit test suite achieving 100% state coverage.

## 3. Exit Gate
- Unit tests verify that task contracts are immutable post-dispatch and all 13 states transition strictly according to `docs/06_WORKFLOW_STATE_MACHINE.md`.
