# P03_ADR_016_EXTERNAL_REAUDIT_002.md — Independent External Supervisor Re-Audit of ADR-016 Revision 2

> **Audit Type**: Independent External Supervisor Re-Audit 002
> **Audited Document**: `docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md` (Revision 2)
> **Audited Commit**: `11624f08d0c7351c7aeb047fa77305d2d5720bd1`
> **Proposal Authority**: `PROPOSAL-P03-002 Revision 6` (`EXTERNAL_APPROVED`, commit `8709af4b6aaf8613c69a10514971e320e3f90048`)
> **Pinned Upstream Authority**: `Untrivial-ai/agent-orchestrator` `v0.13.0` (commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-23

---

## 1. Executive Summary & Verdict

Independent External Supervisor Re-Audit 002 of ADR-016 Revision 2 (commit `11624f08d0c7351c7aeb047fa77305d2d5720bd1`) is complete.

All nine initial audit findings (`ADR16R1-001` through `ADR16R1-009`) remain **CLOSED**.
All nine Revision 2 findings (`ADR16R2-001` through `ADR16R2-009`) are verified as **SUBSTANTIVELY_CLOSED**; the architectural directions selected in Revision 2 are verified as sound.

However, end-to-end document reconciliation revealed ten consistency and alignment defects (`ADR16R3-001` through `ADR16R3-010`) requiring a final reconciliation pass under Revision 3 before formal ADR acceptance can be granted.

### Formal Verdicts
- **`PROPOSAL_P03_002`** = `EXTERNAL_APPROVED`
- **`ADR_016`** = `REVISION_3_REQUIRED`
- **`ADR_016_ACCEPTANCE`** = `NOT_GRANTED`
- **`TASK_P03_003`** = `NOT_RELEASED`
- **`P03_CODE`** = `HELD_PENDING_ADR_016_APPROVAL`
- **`ACTIVE_GATE`** = `P03_ADR_016_REVISION_3`

---

## 2. Status of Previous Findings

| Finding Range | Scope | Verdict |
| :--- | :--- | :--- |
| `ADR16R1-001` through `ADR16R1-009` | Initial External Audit Findings | **CLOSED** |
| `ADR16R2-001` through `ADR16R2-009` | Revision 2 Architectural Remediation | **SUBSTANTIVELY_CLOSED** |

---

## 3. New Findings (Revision 3 Requirements)

### `ADR16R3-001`: CANONICAL_EDGE_ACCOUNTING_STILL_WRONG
- **Severity**: HIGH (Canonical Specification Consistency)
- **Description**: Production defines exactly 22 domain transitions + 1 initial pseudo-edge (`[*] -> DRAFT`) + 2 terminal pseudo-edges (`APPROVED -> [*]` and `CANCELLED -> [*]`) = 25 canonical graph edges. ADR-016 Revision 2 mistakenly listed `[*] -> DRAFT` inside the 22 executable transitions and mischaracterized `HUMAN_REQUIRED -> CANCELLED` as a terminal pseudo-edge.
- **Required Action**: Accurately enumerate the 22 domain transitions, 3 pseudo-edges, and 25 total graph edges in strict alignment with `internal/workflow/state_machine.go`.

### `ADR16R3-002`: D1_WORKERSESSION_SCHEMA_HAS_TWO_AUTHORITATIVE_VERSIONS
- **Severity**: HIGH (Schema Integrity)
- **Description**: Detailed D1 section defined `worker_sessions` with `session_id TEXT NOT NULL` and timestamp columns `created_at` and `updated_at`. Section 27 defined `session_id TEXT NOT NULL UNIQUE`, omitted timestamps, and had redundant inline and table unique constraints.
- **Required Action**: Establish exactly ONE unified candidate schema across all sections: `pair_id PRIMARY KEY`, `session_id TEXT NOT NULL UNIQUE`, `runtime_type`, `worktree_path`, `worker_agent_id`, `status`, `terminal_generation`, `quarantine_state`, `created_at TEXT NOT NULL`, `updated_at TEXT NOT NULL`.

### `ADR16R3-003`: D4_DISPATCH_SCHEMA_HAS_TWO_AUTHORITATIVE_VERSIONS
- **Severity**: CRITICAL (Relational Consistency)
- **Description**: Detailed D4 omitted `UNIQUE(attempt_id)` and used `confirmed_at + resolution_state`. Section 27 had `attempt_id UNIQUE`, added `prompt_hash`, changed to `completed_at`, but omitted `resolution_state`, which is required by D5.
- **Required Action**: Unify on a single schema: `operation_id PRIMARY KEY`, `attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id)`, `pair_id`, `task_id`, `session_id`, `terminal_generation`, `stage`, `requested_at TEXT NOT NULL`, `confirmed_at TEXT`, `resolution_state TEXT`. Remove extraneous `prompt_hash`.

### `ADR16R3-004`: STOP_REQUESTED_TERMINATION_OVERCLAIMS_INTENTIONAL_STOP
- **Severity**: HIGH (Attribution Integrity)
- **Description**: `STOP_REQUESTED` is committed before issuing `/kill`. If a crash occurs before socket transmission and the target independently exits, restart observing `isTerminated == true` does not prove the stop request caused termination. Classifying this as `WORKER_STOPPED` is an overclaim.
- **Required Action**: Reserve `WORKER_STOPPED` strictly for `STOP_CALL_SUCCEEDED` + later matching `isTerminated == true`. For `STOP_REQUESTED` + `isTerminated == true`, record `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED` and terminal classification `WORKER_TERMINATION_UNKNOWN`.

### `ADR16R3-005`: CRASH_MATRIX_STALE_AFTER_REVISION_2
- **Severity**: MEDIUM (Document Alignment)
- **Description**: Section 23 Crash Matrix contained stale descriptions: T1/T2 claimed "No DB record" despite durable `pair_provisioning_operations`, T8 omitted generation verification before `/kill` reissue, and T9 claimed 404 could advance to `STOP_TERMINATION_CONFIRMED`.
- **Required Action**: Reconcile Section 23 directly from D3, D4, D11, and D13 decisions.

### `ADR16R3-006`: CLASS_B_ABSENCE_POLICY_NOT_ACTUALLY_DECIDED
- **Severity**: HIGH (Governance Policy)
- **Description**: D5 selected Class A and Class C clearance, but D11 stated that HTTP 404 follows Class B Administrative Absence rules without defining the policy.
- **Required Action**: Formally decide the Class B policy: HTTP 404 records `STOP_TARGET_ABSENT`, physical termination remains unproven, quarantine remains active, task fails, and replacement dispatch requires explicit operator administrative resolution (`ADMINISTRATIVE_RISK_RESOLUTION`). 404 alone never clears quarantine.

### `ADR16R3-007`: PAIR_PROVISIONING_CONCURRENCY_NOT_GUARDED
- **Severity**: HIGH (Race Hazard)
- **Description**: Pinned spawn has no idempotency. Two concurrent provisioning requests for the same Pair could create two unreferenced upstream sessions. The D3 guard only protected task dispatch, not pre-provisioning exclusivity.
- **Required Action**: Formalize `# ONE_PAIR = AT_MOST_ONE_UNRESOLVED_PROVISIONING_OPERATION`. Enforce via SQLite partial unique index: `CREATE UNIQUE INDEX idx_pair_provisioning_unresolved ON pair_provisioning_operations(pair_id) WHERE stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED');`.

### `ADR16R3-008`: D13_USES_TRANSPORT_PRIMITIVE_OUTSIDE_APPROVED_ADAPTER_SURFACE
- **Severity**: HIGH (Adapter Boundary)
- **Description**: D13 stated that recovery probes daemon availability via `GET /api/v1/sessions`. The approved Supervisor AO adapter (`internal/ao/`) does not expose `ListSessions`.
- **Required Action**: Eliminate raw `GET /api/v1/sessions` from automated orchestration. Use approved adapter methods: `CheckHealth` / `CheckReadiness` for daemon health and `GetWorkerStatus(sessionID)` for session inspection.

### `ADR16R3-009`: STOP_CONFIRMATION_BOUND_NOT_CONNECTED_TO_EXISTING_POLICY
- **Severity**: MEDIUM (Policy Inventory Binding)
- **Description**: Following `STOP_CALL_SUCCEEDED`, observation of process termination was unbounded. The canonical inventory contains `SUPERVISOR_KILL_STOP_TIMEOUT = UNSET`.
- **Required Action**: Bind stop confirmation observation to caller-injected `SUPERVISOR_KILL_STOP_TIMEOUT`. If timeout expires without confirmed exit, persist honest timeout disposition; do not infer termination and do not log `WORKER_STOPPED`.

### `ADR16R3-010`: LIVE_GOVERNANCE_GATE_INCONSISTENT
- **Severity**: MEDIUM (Process Governance)
- **Description**: `AGENTS.md` and `docs/18_CURRENT_STATE.md` contained conflicting active gate references (`_REAUDIT_001` vs `_REAUDIT_002`).
- **Required Action**: Reconcile live governance references to `P03_ADR_016_REVISION_3` / `EXTERNAL_SUPERVISOR_P03_ADR_016_REAUDIT_003` without modifying historical audit entries.

---

## 4. Next Required Action
Execute end-to-end reconciliation for ADR-016 Revision 3 closing findings `ADR16R3-001` through `ADR16R3-010`. Production coding remains held (`TASK_P03_003 = NOT_RELEASED`).
