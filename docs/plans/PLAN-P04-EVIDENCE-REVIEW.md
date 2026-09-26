# PLAN-P04-EVIDENCE-REVIEW: Phase P04 Evidence & Review Engine Implementation and Governance Decomposition Plan

- **Plan ID:** `PLAN-P04-EVIDENCE-REVIEW`
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Status:** `PLANNING_PENDING_EXTERNAL_AUDIT`
- **Governance Baseline:** `3e1848d5d88a873e6338586c23d0b7e9a7a4c62d`
- **Authority:** Engineering Execution Plan under [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md)
- **Decision Precedence:** Level 5 (Approved Roadmap) / Level 6 (Task Contract Precursor)
- **Governing Proposal:** [`PROPOSAL-P04-001`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md)
- **Governing Architecture:** [`DRAFT-ADR-018`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md)
- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_AUDIT`

---

## 1. Executive Summary & Governance Baseline

Phase P03 established the upstream Agent Orchestrator integration and host bootstrap exclusivity (`HOST_QUIESCENCE_INTEGRATION = IMPLEMENTED_AT_LIBRARY_AND_DAEMON_SCOPE`).

Phase P04 implements the **Evidence & Review Engine**:
1. Ingesting, parsing, and validating attempt-scoped `worker-report.json` artifacts from AO sessions without trusting worker content.
2. Collecting authoritative Git evidence directly from the assigned worktree using a read-only, non-mutating Git adapter.
3. Evaluating TaskContract `allowed_scope` and `forbidden_scope` compliance using a deterministic `PolicyEngine`.
4. Executing discrete verification test profiles using an isolated, zero-secret `VerificationRunner` (Layer A command containment + Layer B execution isolation on Windows).
5. Synthesizing Contract, Claims, Git Evidence, Test Results, and Policy Findings into a unified, schema-validated `ReviewBundle`.

Pursuant to [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md) and External Supervisor directive, **Phase P04 is not yet released for code implementation**. This document establishes the formal pre-contract architectural decomposition, subtask dependency graph, blast radius containment, and acceptance criteria.

---

## 2. Phased Scope Decomposition Plan

A monolithic implementation of Phase P04 would span schema migrations, external process invocation, Git repository manipulation, OS sandboxing, and workflow orchestration, creating excessive blast radius.

P04 is decomposed into **four strictly sequential subtasks**:

```mermaid
graph TD
    P04A[TASK-P04-001: P04A - Ingestion & Claims Core]
    P04B[TASK-P04-002: P04B - Git Evidence & Scope Policy]
    P04C[TASK-P04-003: P04C - Verification Catalog & Runner]
    P04D[TASK-P04-004: P04D - ReviewBundle & Orchestration]

    P04A --> P04B
    P04B --> P04C
    P04C --> P04D
```

### 2.1. Subtask P04A: WorkerReport Ingestion, Schema Validation & Claim Persistence
- **Objective:** Ingest attempt-scoped `worker-report.json`, enforce schema validation against `worker-report.schema.json`, extract unverified `WorkerClaim` entities, handle `REC-007`/`REC-008` failures, and atomically transition `RUNNING -> REPORT_READY`.
- **Target Deliverables:**
  - `internal/report/parser.go`: Bounded JSON parser and schema validator.
  - `internal/report/ingest.go`: Ingestion coordinator calling `AOAdapter.GetWorkspaceFile`.
  - SQLite Schema v6 Table: `worker_claims`.
  - Atomic transaction: `RUNNING -> REPORT_READY` with `task_attempts.ended_at` and audit event.
  - Failure handlers: `REC-007` (`REPORT_MISSING`) and `REC-008` (`REPORT_INVALID`, `REPORT_IDENTITY_MISMATCH`).
- **Proposed `allowed_scope`:**
  - `internal/domain/entities.go`
  - `internal/report/**`
  - `internal/store/migrations.go`
  - `internal/store/report_store.go`
  - `test/unit/report_test.go`
- **Exit Gate:** Valid reports parse and persist claims atomically; malformed or missing reports cleanly trigger `REC-007`/`REC-008` transition to `FAILED`; zero Git or test runner dependencies.

---

### 2.2. Subtask P04B: Git EvidenceCollector & PolicyEngine
- **Objective:** Implement the read-only, non-mutating Git evidence adapter and the deterministic `PolicyEngine` scope evaluator.
- **Target Deliverables:**
  - `internal/git/evidence_adapter.go`: Read-only trusted absolute `git.exe` executor with sanitized environment, commit object verification, direct HEAD extraction, linear ancestry verification (`git merge-base --is-ancestor`), and binary diff hashing.
  - `internal/policy/engine.go`: Deterministic scope evaluator enforcing `forbidden_scope PRECEDENCE > allowed_scope`, Windows case-insensitivity, slash normalization, and special path handling (renames, deletes, submodules, symlinks).
  - SQLite Schema v6 Tables: `evidence`, `policy_findings`.
- **Proposed `allowed_scope`:**
  - `internal/git/**`
  - `internal/policy/**`
  - `internal/store/evidence_store.go`
  - `test/unit/git_evidence_test.go`
  - `test/unit/policy_engine_test.go`
- **Exit Gate:** Mathematically verifies actual Git commit diffs; detects out-of-scope and forbidden file modifications; detects uncommitted worktree pollution; flags ancestry divergence fail-closed.

---

### 2.3. Subtask P04C: VerificationPolicyCatalog & Isolated VerificationRunner
- **Objective:** Implement the production verification catalog loader and the Layer B execution isolation runner for Windows.
- **Target Deliverables:**
  - `internal/verification/catalog_loader.go`: Host verification profile configuration loader and trusted executable registry.
  - `internal/verification/runner_windows.go`: Windows execution isolation runner combining Windows AppContainer / Restricted Token with Windows Job Object process-tree containment.
  - Environment stripper: Guarantees zero host secrets, redirected attempt-local `TEMP`/`TMP`, and default-deny network access.
  - Stream capper: Enforces hard stdout/stderr byte ceilings and output hashing (SHA-256).
  - SQLite Schema v6 Table: `actual_test_results`.
- **Proposed `allowed_scope`:**
  - `internal/verification/**`
  - `internal/store/verification_store.go`
  - `test/unit/verification_runner_test.go`
- **Exit Gate:** Safely executes verification profiles within isolated sandbox; enforces process-tree termination on timeout/cancel; captures exit codes and output hashes; zero host secret leakage.

---

### 2.4. Subtask P04D: ReviewBundleBuilder & End-to-End Workflow Orchestration
- **Objective:** Assemble all evidence into the synthesized `ReviewBundle`, validate 100% against `review-bundle.schema.json`, manage atomic state progressions (`REPORT_READY -> EVIDENCE_READY -> REVIEWING`), and prove end-to-end integration.
- **Target Deliverables:**
  - `internal/review/builder.go`: ReviewBundle compiler assembling Contract, Claims, Git Evidence, Test Results, Policy Findings, and Review Focus hints.
  - `internal/review/validator.go`: JSON schema validator against reconciled `review-bundle.schema.json`.
  - SQLite Schema v6 Table: `review_bundles`.
  - Orchestrator: Managing atomic multi-step pipeline and audit logging.
  - `test/integration/p04_review_engine_test.go`: End-to-end integration test harness.
- **Proposed `allowed_scope`:**
  - `internal/review/**`
  - `internal/workflow/**`
  - `internal/store/review_store.go`
  - `test/integration/p04_review_engine_test.go`
- **Exit Gate:** ReviewBundles validate 100% against JSON schema; atomic transactions guarantee zero partial evidence state; full end-to-end integration harness passes with zero race conditions (`go test -race -count=1 ./...`).

---

## 3. Blast Radius and Scope Guardrails

1. **Sequential Execution**: Subtasks must proceed strictly in order (P04A -> P04B -> P04C -> P04D). No concurrent code development across subtasks.
2. **Parent Plan Limitation**: This plan is an architectural blueprint. It does **NOT** release or authorize writing code for any subtask. Each subtask requires an approved Task Contract (`TASK_CONTRACT_P04_00xA`).
3. **No Go Code in Current Gate**: In the current `P04_PRECONTRACT_ARCHITECTURE_AUDIT` gate, zero production or test Go code, zero migrations, and zero schema edits may be performed.
4. **Upstream Immutability**: Neither Agent Orchestrator nor Antigravity CLI source code may be modified.

---

## 4. Worktree Binding Resolution Strategy

Pursuant to `PROPOSAL-P04-001` §3 and `DRAFT-ADR-018` Decision 1:
- P04A/P04B will formalize `attempt_workspace_bindings` to lock the physical worktree at dispatch time.
- If physical worktree cannot be verified as a valid Git worktree before dispatch, the system halts fail-closed (`FAIL_CLOSED`).
- Evidence collection compares worktree identity against this binding.

---

## 5. Verification Isolation Architecture & Platform Strategy

Pursuant to `DRAFT-ADR-018` Decision 3:
- Rejects relying solely on Windows Job Object for Layer B.
- Implements Windows AppContainer or Restricted Token isolation combined with Job Object process-tree tracking.
- Implements zero-secret environment and attempt-local temporary directories.
- Operational timeouts and buffer limits remain `VALUE = UNSET` or host-injected.

---

## 6. ReviewBundle Schema Reconciliation Strategy (S1..S10)

Pursuant to `PROPOSAL-P04-001` §8:
- **S1**: Standardize `actual_git_evidence` schema (`diff_stat`, `git_diff_hash`).
- **S2**: Remove obsolete `full_diff_url`.
- **S3**: Formalize `actual_test_evidence` schema with `all_passed` and `test_results`.
- **S4**: Formalize `worker_claims` schema matching `WorkerClaim`.
- **S5**: Apply `additionalProperties: false` to all nested schema objects.
- **S6**: Define explicit string/array length bounds.
- **S7**: Align Go domain entities in `internal/domain/entities.go`.
- **S8**: Update valid example JSON.
- **S9**: Formalize deterministic serialization for `bundle_hash`.
- **S10**: Submit reconciled documentation for External Supervisor audit.

---

## 7. Requirement Clarification: NFR-008 Performance SLA

As established in `PROPOSAL-P04-001` §12, the 3-second SLA in `NFR-008` conflicts with long-running verification suites (e.g. `go test -race ./...`).
- We recommend **Option A**: The 3-second SLA applies strictly to pure ReviewBundle assembly and JSON serialization (`EVIDENCE_READY -> REVIEWING`), excluding test execution time.
- Final determination awaits External Supervisor decision.

---

## 8. Phase P04 Overall Exit Gate

Phase P04 is declared complete when:
1. `EvidenceCollector` mathematically matches actual Git commit diffs on test repositories.
2. `PolicyEngine` detects 100% of out-of-scope and forbidden file modifications.
3. `VerificationRunner` safely executes verification profiles within isolated Layer B boundary without secret leakage.
4. Compiled `ReviewBundle` validates 100% against reconciled `review-bundle.schema.json`.
5. Full integration test harness covering all 13 recovery scenarios passes cleanly under `go test -race -count=1 ./...`.
6. Formal External Supervisor Exit Audit approves Phase P04.

---

## 9. Governance Tracking & Directives

- `DESIGN_BLOCKER_P04_WORKTREE_BINDING = OPEN`
- `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`
- `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`
- `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY = OPEN`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_AUDIT`
- `AUTOMATIC_RESTORE = DISABLED`
- `LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`
- `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`
- `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`
