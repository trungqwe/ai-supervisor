# P03_PRECODE_EXTERNAL_REAUDIT_002.md — External Supervisor Decision Record (Revision 2)

> **Audited Commit**: `0eefe64ce47a400e384b240b45d91074900db9d1`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: `PROPOSAL_P03_001_CORE_DIRECTION = EXTERNAL_APPROVED`
> **Canonical Reconciliation Status**: `CANONICAL_RECONCILIATION_APPLIED`
> **Active Gate**: `EXTERNAL_SUPERVISOR_P03_CANONICAL_RECONCILIATION`
> **P03 Production Code**: `HELD`
> **TASK_P03_001**: `NOT_RELEASED`

---

## 0. External Supervisor Decision Summary

The External Supervisor independently reviewed:
* Commit `0eefe64ce47a400e384b240b45d91074900db9d1`;
* `PROPOSAL-P03-001-runtime-observation-and-compatibility-contract.md`;
* `P03_PRECODE_UPSTREAM_CONTRACT_REAUDIT_001.md`;
* Worker final execution report;
* Canonical governance and specifications;
* Pinned AO public source.

### Verdict on Core Directions:
The core directions of `PROPOSAL-P03-001` are **APPROVED**:
1. **FR-006**: Authoritative AO session snapshot observation is the P03 baseline.
2. **Worker Heartbeat**: No synthetic worker heartbeat event shall be invented.
3. **CDC**: `/api/v1/events` is NOT the lifecycle authority. CDC may be considered later as a wake-up optimization only. P03 baseline uses authoritative GET session snapshots.
4. **FR-015**: Liveness/readiness is verified via `/healthz` and `/readyz`. Harness inventory/readiness is verified through public agent endpoints. Runtime AO release version is NOT exposed by health endpoints.
5. **GetWorkspaceFile**: A session-scoped, read-only raw workspace-file transport primitive belongs in `AOAdapter` because ADR-011 already requires this upstream surface.
6. **REC-005**: Supervisor must fail closed. No `git stash`, `git clean`, checkout, reset, or custom worktree mutation.
7. **P03/P04 Boundary**: P03 owns raw AO transport only. P04 owns WorkerReport semantic ingestion, schema validation, WorkerClaim creation, and `REPORT_READY` transition.
8. **Architecture Impact**: NO architecture change.
9. **ADR Requirement**: NO new ADR required.

However, `P03_PRECODE_REAUDIT_001` was determined `REVISION_REQUIRED` due to policy value inventions, semantic overclaims, and ownership ambiguity recorded in findings `P03R2-001` through `P03R2-008`.

---

## 1. Revision-2 Findings & Resolution Matrix

| Finding ID | Name | Initial Status | Remediated Status | Resolution & Verification |
|---|---|---|---|---|
| **P03R2-001** | `UNAUTHORIZED_POLICY_VALUE_INVENTION` | `REVISION_REQUIRED` | `CLOSED` | Removed all unapproved proposed values (10s, 15s, 5s, 1s-2s, 300s, 30s, 3 retries, circuit breaker). All 8 Supervisor policies set to `VALUE = UNSET, STATUS = UNRESOLVED_POLICY`. Only upstream fact retained is `AO DefaultRequestTimeout = 60s`. |
| **P03R2-002** | `OPENAPI_FINGERPRINT_IS_NOT_RELEASE_VERSION` | `REVISION_REQUIRED` | `CLOSED` | Reconciled classification: `info.version = 0.1.0-route-shell` is an API compatibility signal only, NOT release version proof. Release version identity is verified via out-of-band pinned deployment provenance. |
| **P03R2-003** | `LAST_ACTIVITY_IS_NOT_HANG_HEARTBEAT` | `REVISION_REQUIRED` | `CLOSED` | Clarified that `Activity.LastActivityAt` is diagnostic evidence, not hang heartbeat proof. Legitimate tools produce no intermediate callbacks. Bounded execution deadlines govern timeouts. |
| **P03R2-004** | `STOPPED_VS_CRASHED_NOT_DERIVABLE_FROM_SNAPSHOT_ALONE` | `REVISION_REQUIRED` | `CLOSED` | Reconciled event classification: `WORKER_STOPPED` requires Supervisor intentional-stop provenance; `WORKER_CRASHED` requires unexpected exit with observable signals; otherwise fail closed as `WORKER_TERMINATION_UNKNOWN`. |
| **P03R2-005** | `AOADAPTER_MUST_NOT_OWN_STATESTORE` | `REVISION_REQUIRED` | `CLOSED` | Explicitly asserted `AOADAPTER_STATESTORE_DEPENDENCY = FORBIDDEN` and `AOADAPTER_DIRECT_SQL = FORBIDDEN`. All Task state transitions are owned and executed exclusively by the Supervisor orchestration layer using P02 APIs. |
| **P03R2-006** | `PROPOSAL_GOVERNANCE_RANK_INCORRECT` | `REVISION_REQUIRED` | `CLOSED` | Removed false "Level 3 Proposal" rank. Proposal metadata updated to `Governance Stage: CHANGE_INTAKE_PROPOSAL`, `Status: EXTERNAL_APPROVED`. |
| **P03R2-007** | `WORKER_FINAL_PROSE_DEFECTS` | `REVISION_REQUIRED` | `CLOSED` | Identified and quarantined non-authoritative worker prose defects (`/api/v1/health`, `ActivityBusy`, auth headers, circuit breakers). Repository canonical records remain authoritative. |
| **P03R2-008** | `CANONICAL_AMENDMENT_SET_INCOMPLETE` | `REVISION_REQUIRED` | `CLOSED` | Reconciled all impacted canonical documents: `docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, and `docs/phases/P03_AO_INTEGRATION.md`. |

---

## 2. Canonical Reconciliation Impact Assessment

* **Architecture Impact**: `NO_CHANGE_REQUIRED`. `docs/04_ARCHITECTURE.md` audited; already models session idle observation and workspace file fetching without contradictions.
* **Observability Impact**: `NO_CHANGE_REQUIRED`. `docs/15_OBSERVABILITY.md` audited; canonical events remain `worker.started`, `worker.stopped`, `worker.crashed`. Zero `worker.heartbeat` present.
* **Requirements Reconciliation**: `docs/02_REQUIREMENTS.md` updated for FR-006 and FR-015.
* **Interface Reconciliation**: `docs/12_UPSTREAM_INTEGRATION.md` updated with `getWorkspaceFile` and StateStore prohibition.
* **Failure Recovery Reconciliation**: `docs/14_FAILURE_RECOVERY.md` updated: REC-005 fails closed (`WORKTREE_DIRTY`) with zero git mutations.
* **Roadmap Reconciliation**: `docs/17_ROADMAP.md` updated: P03 confined to transport/observation; P04 retains report validation.
* **Traceability Reconciliation**: `docs/21_TRACEABILITY_MATRIX.md` updated for FR-006 and FR-015.
* **Module Provenance Reconciliation**: `docs/22_MODULE_PROVENANCE.md` updated with `AOAdapter` workspace read primitive and explicit forbidden responsibilities.
* **Phase Spec Reconciliation**: `docs/phases/P03_AO_INTEGRATION.md` updated to remove legacy native heartbeat deliverable.

---

## 3. Operational Guardrails

* **PROPOSAL_P03_001_CORE_DIRECTION**: `EXTERNAL_APPROVED`
* **PROPOSAL_P03_001**: `EXTERNAL_APPROVED`
* **P03_ARCHITECTURE_CHANGE**: `NO`
* **P03_ADR_REQUIRED**: `NO`
* **P03_CANONICAL_RECONCILIATION**: `APPLIED`
* **P03_CODE**: `HELD`
* **TASK_P03_001**: `NOT_RELEASED`
* **ACTIVE_GATE**: `EXTERNAL_SUPERVISOR_P03_CANONICAL_RECONCILIATION`

Production Go code remains strictly **`HELD`**. No source files in `internal/**` may be created or modified until the External Supervisor audits this canonical reconciliation and releases an immutable Task Contract for `TASK-P03-001`.
