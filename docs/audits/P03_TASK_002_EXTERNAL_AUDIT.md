# P03_TASK_002_EXTERNAL_AUDIT.md - External Supervisor Audit Record (TASK-P03-002)

> **Audited Commit**: `056629ff4d38bbf4caaa94fc860825a2e9991a7c`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: `TASK_P03_002 = REVISION_REQUIRED`
> **Canonical Reconciliation Status**: `P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_001 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_003 Status**: `NOT_RELEASED`
> **P03 Code Status**: `AUTHORIZED_FOR_TASK_P03_002_REVISION_1_ONLY`
> **Active Gate**: `P03_TASK_002_REVISION_1`

---

## 0. External Supervisor Decision Summary

Independent external audit of TASK-P03-002 (commit `056629ff4d38bbf4caaa94fc860825a2e9991a7c`) found exactly one blocking correctness defect.

### Verdict:
- `P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`
- `TASK_P03_001 = EXTERNAL_AUDIT_APPROVED`
- `TASK_P03_002 = REVISION_REQUIRED`
- `TASK_P03_003 = NOT_RELEASED`
- `P03_CODE = AUTHORIZED_FOR_TASK_P03_002_REVISION_1_ONLY`
- `ACTIVE_GATE = P03_TASK_002_REVISION_1`

---

## 1. External Audit Finding

| Finding | Title | Verdict | Description |
|---|---|---|---|
| `P03T2R1-001` | `ACTIVITY_PROTOCOL_ERROR_STATUS_MISREPORTED` | **REVISION_REQUIRED** | The shared canonical activity validator hard-coded `ProtocolError.StatusCode = 200` even when validating the HTTP-201 spawn response, causing inaccurate normalized protocol-error evidence. |

### Finding Explanation:
Current shared helper `validateCanonicalActivityState(...)` creates `ProtocolError` with `StatusCode: http.StatusOK` unconditionally.
This is incorrect when called from `CreateWorkerSession()` because `POST /api/v1/sessions` has exact successful upstream status `HTTP 201 Created`.
Therefore a malformed activity state received in an otherwise HTTP-201 spawn response was reported as `ProtocolError.StatusCode = 200` instead of the actually observed `ProtocolError.StatusCode = 201`.
This corrupts protocol-error evidence.

No other TASK-P03-002 functionality is reopened.
