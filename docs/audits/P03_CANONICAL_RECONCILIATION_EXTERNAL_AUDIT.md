# P03_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT.md — External Supervisor Audit Record (Revision 3)

> **Audited Commit**: `2fc0c73d5181f00cff18a5e2906133b1e39eb7df`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: `REVISION_REQUIRED`
> **Active Gate**: `P03_CANONICAL_RECONCILIATION_REVISION_3`
> **P03 Production Code**: `HELD`
> **TASK_P03_001**: `NOT_RELEASED`

---

## 0. External Supervisor Verdict Summary

The External Supervisor independently audited commit `2fc0c73d5181f00cff18a5e2906133b1e39eb7df`, the approved `PROPOSAL-P03-001`, canonical specifications, and pinned AO v0.13.0 public source.

### Canonical Governance Status:
```yaml
PROPOSAL_P03_001: EXTERNAL_APPROVED
P03_ARCHITECTURE_CHANGE: NO
P03_ADR_REQUIRED: NO
P03_CANONICAL_RECONCILIATION: REVISION_REQUIRED
P03_CODE: HELD
TASK_P03_001: NOT_RELEASED
ACTIVE_GATE: P03_CANONICAL_RECONCILIATION_REVISION_3
```

While the core architectural directions of `PROPOSAL-P03-001` remain approved, four remaining canonical contract defects were identified that would force the `TASK-P03-001` implementation to invent behavior or violate canonical models.

---

## 1. External Audit Findings (Revision 3)

| Finding ID | Name | Status | Classification & Root Cause | Required Remediation |
|---|---|---|---|---|
| **P03R3-001** | `FR015_AOADAPTER_INTERFACE_INCOMPLETE` | `OPEN` | FR-015 requires daemon readiness, agent inventory, agent readiness, and public schema compatibility, but canonical `IAOAdapter` in `docs/12` only exposed `checkHealth()`. | Add normalized preflight probe methods (`checkReadiness`, `listAgents`, `getAgentReadiness`, `getAPIContract`) to `IAOAdapter` in `docs/12_UPSTREAM_INTEGRATION.md`. Explicitly classify `getAPIContract` as an API compatibility signal, not release identity proof. |
| **P03R3-002** | `STALE_SUBSCRIBE_EVENTS_CONTRACT` | `OPEN` | `IAOAdapter` in `docs/12` retained `subscribeEvents(...)`, contradicting the approved baseline of authoritative session snapshot polling and risking accidental SSE client implementation. | Remove `subscribeEvents(...)` from active canonical `IAOAdapter`. Document CDC event stream as a deferred optional future optimization only. |
| **P03R3-003** | `REC005_STATE_MODEL_INCONSISTENT` | `OPEN` | REC-005 directed marking task attempt blocked/failed with `failure_reason = WORKTREE_DIRTY`. But `TaskAttempt` has no `state` or `failure_reason` column in P02, and `StateMachine` forbids `READY -> BLOCKED`/`FAILED`. | Reconcile REC-005: Case A (pre-`PrepareDispatch`) rejects dispatch, leaving task `READY` with zero attempt allocated; Case B (post-`PrepareDispatch`) transitions `DISPATCHED -> FAILED` with audit log evidence. |
| **P03R3-004** | `AO_PUBLIC_EXIT_CODE_OVERCLAIM` | `OPEN` | `docs/22_MODULE_PROVENANCE.md` stated "AO reports session exit code", but public session snapshots only expose `IsTerminated` and `Activity` without stable exit codes. | Remove public exit-code overclaim in `EvidenceCollector` and `AOAdapter` provenance. Preserve stop/crash/unknown classification without inventing TaskState. |

---

## 2. Operational Guardrails

* **P03 Production Code**: `HELD`. Zero Go code permitted in `internal/**`.
* **TASK_P03_001**: `NOT_RELEASED`.
* **Scope**: Documentation and governance contract alignment only.
