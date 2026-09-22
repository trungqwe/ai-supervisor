# P03_PRECODE_EXTERNAL_AUDIT.md - External Supervisor Audit Record (P03 Pre-Code Upstream Contract)

> **Audited Commit**: `681bbd92bc720d59b1e422ac40808ecc1e4e1436`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: `REVISION_REQUIRED`
> **Active Gate**: `P03_PRECODE_AUDIT_REVISION_1`
> **P03 Production Code**: `HELD`
> **TASK_P03_001**: `NOT_RELEASED`

---

## 0. External Supervisor Verdict Summary

The External Supervisor independently audited:
* Worker P03 pre-code report;
* Remote repository at commit `681bbd92bc720d59b1e422ac40808ecc1e4e1436`;
* P03 pre-code audit dossier (`docs/audits/P03_PRECODE_UPSTREAM_CONTRACT_AUDIT.md`);
* Canonical Requirements (`docs/02_REQUIREMENTS.md`);
* Canonical Architecture (`docs/04_ARCHITECTURE.md`);
* Upstream Integration Specification (`docs/12_UPSTREAM_INTEGRATION.md`);
* Failure Recovery Specification (`docs/14_FAILURE_RECOVERY.md`);
* Observability Specification (`docs/15_OBSERVABILITY.md`);
* Roadmap (`docs/17_ROADMAP.md`);
* Traceability Matrix (`docs/21_TRACEABILITY_MATRIX.md`);
* Module Provenance (`docs/22_MODULE_PROVENANCE.md`);
* Change Governance (`docs/24_CHANGE_GOVERNANCE.md`);
* Pinned AO v0.13.0 source at commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.

### Canonical Verdict Flags
```yaml
P02_FINAL_AUDIT: EXTERNAL_AUDIT_APPROVED
P02: COMPLETE
P03_PRECODE_AUDIT: REVISION_REQUIRED
P03_ARCHITECTURE_CHANGE: NO
P03_ADR_REQUIRED: NO - PROVISIONAL
P03_REQUIREMENT_GOVERNANCE: REQUIRED
P03_CODE: HELD
TASK_P03_001: NOT_RELEASED
ACTIVE_GATE: P03_PRECODE_AUDIT_REVISION_1
```

---

## 1. External Audit Findings (Blocking P03 Code Release)

| Finding ID | Name | Status | Classification & Root Cause | Required Remediation |
|---|---|---|---|---|
| **P03PRE-001** | `REQUIREMENT_AMENDMENT_PIPELINE_SKIPPED` | **OPEN** | FR-006 heartbeat removal, FR-015 version handshake fallback, and missing workspace file primitive require canonical governance proposals before code release. | Submit formal governance proposal `PROPOSAL-P03-001` in `docs/proposals/` under `docs/24_CHANGE_GOVERNANCE.md`. Mark status `RECOMMENDED_FOR_EXTERNAL_APPROVAL`. Do not edit canonical specs until approved. |
| **P03PRE-002** | `FR015_RUNTIME_COMPATIBILITY_EVIDENCE_INCOMPLETE` | **OPEN** | Pinned AO `GET /api/v1/health` has no `version` field. Initial audit proposed skipping without documenting verification evidence or fallback strategy. | Document in proposal and dossier that compatibility is verified by out-of-band pinned binary deployment provenance and `GET /api/v1/openapi.yaml` schema fingerprinting. |
| **P03PRE-003** | `P04_SCOPE_LEAK_IN_TASK_DECOMPOSITION` | **OPEN** | Proposed TASK-P03-004 included WorkerReport JSON parsing, schema validation, and WorkerClaim creation, which belongs strictly to Phase P04 (EvidenceCollector / FR-007). | Move all report parsing, schema validation, and WorkerClaim creation to P04. Confine TASK-P03-004 strictly to `GetWorkspaceFile` raw read transport primitive. |
| **P03PRE-004** | `AOADAPTER_CANONICAL_INTERFACE_INCOMPLETE` | **OPEN** | Canonical `IAOAdapter` interface lacks workspace file retrieval method despite FR-007 and ADR-003 mandating workspace report retrieval via AO HTTP API. | Propose adding `GetWorkspaceFile(ctx, sessionID, relativePath) ([]byte, error)` to `IAOAdapter` via `PROPOSAL-P03-001`. |
| **P03PRE-005** | `CURRENT_STATE_INCONSISTENT` | **OPEN** | `docs/18_CURRENT_STATE.md` listed stale open decisions and failed to reflect `P03 = PRECODE_AUDIT_REVISION_REQUIRED`, `P03_CODE = HELD`, and `TASK_P03_001 = NOT_RELEASED`. | Reconcile `docs/18_CURRENT_STATE.md` with operational statuses, audit history, and exact list of 5 open decisions awaiting External Supervisor approval. |
| **P03PRE-006** | `POLICY_INVENTORY_INACCURATE` | **OPEN** | Prior audit cited 8 policies instead of 9, omitting upstream request timeout or conflating supervisor timeouts with AO defaults. | Re-audit full policy inventory to exactly 9 items: 8 Supervisor policies + 1 Upstream AO request timeout (`config.DefaultRequestTimeout = 60s`). |
| **P03PRE-007** | `RUNNING_STATE_TRANSITION_OMITTED` | **OPEN** | FR-006 acceptance criteria require `DISPATCHED -> RUNNING` transition upon worker activity detection. Initial task decomposition lacked an explicit subtask for this. | Explicitly assign `DISPATCHED -> RUNNING` transition to TASK-P03-003 upon observing worker transition to active execution. |
| **P03PRE-008** | `WORKTREE_REC005_READONLY_CONFLICT` | **OPEN** | `docs/14_FAILURE_RECOVERY.md` REC-005 mandates AOAdapter "stash or clean" worktrees, directly violating AOAdapter read-only boundary. | Propose amending REC-005 to fail closed (`WORKTREE_DIRTY`) without performing git mutations in `PROPOSAL-P03-001`. |

---

## 2. Detailed Finding Analysis

### 2.1 P03PRE-001: Requirement Amendment Pipeline Skipped
- **Rule Violated**: `AGENTS.md` Permanent Rule 2 & 7; `docs/24_CHANGE_GOVERNANCE.md`.
- **Analysis**: The worker identified that pinned AO v0.13.0 does not produce worker-level heartbeats (the SSE keepalive comment `:

` is transport-only), that `GET /api/v1/health` lacks a version field, and that AOAdapter needs a workspace file retrieval method. However, the worker did not file a governance proposal in `docs/proposals/` and instead attempted to leave requirements amended informally in audit prose.
- **Remediation**: Create `docs/proposals/PROPOSAL-P03-001-runtime-observation-and-compatibility-contract.md` adhering to the 15-section change governance standard with status `RECOMMENDED_FOR_EXTERNAL_APPROVAL`.

### 2.2 P03PRE-002: FR-015 Runtime Compatibility Evidence Incomplete
- **Rule Violated**: `docs/02_REQUIREMENTS.md` FR-015; `docs/22_MODULE_PROVENANCE.md`.
- **Analysis**: `GET /api/v1/health` in pinned AO (`backend/internal/httpd/router.go:394-420`) returns:
  ```json
  {"status": "ok", "service": "agent-orchestrator", "pid": 1234, "paths": {...}}
  ```
  It has no `version` field. The worker must not blindly remove compatibility verification.
- **Remediation**: Reconcile verification strategy:
  1. Primary: Out-of-band pinned binary deployment provenance (SHA-256 build verification).
  2. Runtime secondary: `GET /api/v1/openapi.yaml` hash/route fingerprinting.
  3. Propose formal FR-015 amendment in `PROPOSAL-P03-001`.

### 2.3 P03PRE-003: P04 Scope Leak in Task Decomposition
- **Rule Violated**: `docs/17_ROADMAP.md` Phase P03/P04 Boundary; `AGENTS.md` Rule 4.
- **Analysis**: Initial TASK-P03-004 included parsing `worker_report.json`, validating against JSON schema, creating `WorkerClaim`, and triggering `REPORT_READY`. These are core domain responsibilities of `EvidenceCollector` (Phase P04 / FR-007). In Phase P03, AOAdapter is strictly an upstream integration adapter and transport boundary.
- **Remediation**: TASK-P03-004 must only implement `GetWorkspaceFile(ctx, sessionID, path)` as a raw transport read. All report parsing, claim extraction, and evaluation belong to P04.

### 2.4 P03PRE-004: AOAdapter Canonical Interface Incomplete
- **Rule Violated**: `docs/04_ARCHITECTURE.md` ADR-003; `docs/12_UPSTREAM_INTEGRATION.md`.
- **Analysis**: The canonical `IAOAdapter` interface in `docs/04_ARCHITECTURE.md` only defined session lifecycle methods (`CreateSession`, `SendPrompt`, `KillSession`, `RestoreSession`, `GetSessionStatus`). It lacked any method to read files from the session workspace, making it impossible for P04 to retrieve worker reports via the adapter without violating boundaries.
- **Remediation**: Formally propose amending `IAOAdapter` to include `GetWorkspaceFile(ctx, sessionID, relativePath) ([]byte, error)` in `PROPOSAL-P03-001`.

### 2.5 P03PRE-005: Current State Inconsistent
- **Rule Violated**: `AGENTS.md` Permanent Rule 9; `docs/18_CURRENT_STATE.md`.
- **Analysis**: `docs/18_CURRENT_STATE.md` was not updated to reflect `P03_PRECODE_AUDIT = REVISION_REQUIRED`, `P03_CODE = HELD`, and `TASK_P03_001 = NOT_RELEASED`. It retained outdated open decisions from P02.
- **Remediation**: Update `docs/18_CURRENT_STATE.md` to reflect all operational gates, audit history, and exactly 5 open decisions for P03 pre-code approval.

### 2.6 P03PRE-006: Policy Inventory Inaccurate
- **Rule Violated**: `docs/12_UPSTREAM_INTEGRATION.md` Section 6; `docs/14_FAILURE_RECOVERY.md`.
- **Analysis**: Prior audit miscounted policies as 8 instead of 9, omitting upstream request timeout or conflating supervisor timeouts with AO defaults.
- **Remediation**: Correct inventory to 9 distinct policies (8 Supervisor policies + 1 Upstream AO request timeout `config.DefaultRequestTimeout = 60s`).

### 2.7 P03PRE-007: Running State Transition Omitted
- **Rule Violated**: `docs/02_REQUIREMENTS.md` FR-006 Acceptance Criteria.
- **Analysis**: Task decomposition did not specify how `DISPATCHED -> RUNNING` transition occurs. The StateMachine requires this transition when a worker begins executing work.
- **Remediation**: Explicitly assign responsibility to TASK-P03-003: observation loop detects `ActivityActive` and invokes StateStore transition `DISPATCHED -> RUNNING`.

### 2.8 P03PRE-008: Worktree REC-005 Readonly Conflict
- **Rule Violated**: `docs/04_ARCHITECTURE.md` ADR-003; `docs/14_FAILURE_RECOVERY.md`.
- **Analysis**: REC-005 required AOAdapter to "stash or clean" dirty worktrees before session reuse. But AOAdapter has no git write authority and must not execute destructive mutations on the agent workspace.
- **Remediation**: Propose amending REC-005 in `PROPOSAL-P03-001` so that if a worktree is dirty, the session is marked `WORKTREE_DIRTY` and fails closed, requiring operator or orchestrator remediation.

---

## 3. Mandatory Pre-Conditions for P03 Code Authorization

Production P03 Go code implementation (`internal/ao/**`, `internal/adapter/**`, etc.) will **NOT** be released until all of the following conditions are met:

1. **PROPOSAL-P03-001 Review**: External Supervisor explicitly approves `PROPOSAL-P03-001-runtime-observation-and-compatibility-contract.md`.
2. **Canonical Spec Canonicalization**: Once approved, canonical documents (`docs/02`, `docs/04`, `docs/12`, `docs/14`, `docs/15`, `docs/17`, `docs/21`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`) are updated to incorporate approved amendments.
3. **Task Contract Release**: External Supervisor issues an immutable, formally approved Task Contract for `TASK-P03-001`.
4. **Code Authorization Flag**: `P03_CODE` is transitioned from `HELD` to `AUTHORIZED`.

---

## 4. External Supervisor Sign-off & Active Gate

* **Current Status**: `REVISION_REQUIRED`
* **Active Gate**: `P03_PRECODE_AUDIT_REVISION_1`
* **Target Gate**: `EXTERNAL_SUPERVISOR_P03_PRECODE_REAUDIT`
* **P03 Production Code**: `HELD`
* **TASK_P03_001**: `NOT_RELEASED`
