# PHASE SPECIFICATION: P02 — SUPERVISOR DOMAIN CORE

## 1. Objective
Implement the core domain model, 13-state workflow state machine, immutable Task Contract validator, and durable local State Store.

## 2. Deliverables
- Domain entity classes (Project, Pair, Task, TaskContract, WorkerClaim, Evidence).
- Workflow State Machine transition engine with strict transition rules.
- Local State Store persistence layer.
- Unit test suite achieving 100% state coverage.

## 3. Exit Gate
- Unit tests verify that task contracts are immutable post-dispatch and all 13 states transition strictly according to `docs/06_WORKFLOW_STATE_MACHINE.md`.
