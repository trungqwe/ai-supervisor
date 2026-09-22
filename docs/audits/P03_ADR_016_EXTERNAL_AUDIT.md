# P03_ADR_016_EXTERNAL_AUDIT.md — External Supervisor Audit Record (ADR-016 Draft)

> **Audited Commit**: `928b0d110058d8ca90d8d9b68d9aa478c1ee089a`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Proposal Authority**: `PROPOSAL-P03-002 Revision 6` (`EXTERNAL_APPROVED`, commit `8709af4b6aaf8613c69a10514971e320e3f90048`)
> **Audit Date**: 2026-09-23
> **External Supervisor Verdict**: `ADR_016 = REVISION_1_REQUIRED`
> **ADR-016 Acceptance Status**: `ADR_016_ACCEPTANCE = NOT_GRANTED`
> **Proposal Status**: `PROPOSAL_P03_002 = EXTERNAL_APPROVED`
> **TASK_P03_001 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_002 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_003 Status**: `NOT_RELEASED`
> **P03 Code Status**: `HELD_PENDING_ADR_016_APPROVAL`
> **Active Gate**: `P03_ADR_016_REVISION_1`

---

## 0. External Supervisor Audit Decision Summary

Independent external audit of ADR-016 draft (commit `928b0d110058d8ca90d8d9b68d9aa478c1ee089a`) is complete.

While ADR-016 successfully synthesizes the core recommendations of approved `PROPOSAL-P03-002 Revision 6`, nine (9) specific architectural findings require remediation before the decision record can be considered for acceptance.

**Verdict**: `ADR_016 = REVISION_1_REQUIRED`. `ADR_016_ACCEPTANCE = NOT_GRANTED`.

---

## 1. External Audit Findings Matrix

| Finding ID | Title | Verdict | Severity | Remediation Mandate |
|---|---|---|---|---|
| `ADR16R1-001` | `ORPHAN_REAPER_OWNERSHIP_UNPROVEN` | **REVISION_1_REQUIRED** | Blocker | Pinned AO spawn exposes no client correlation ID. Unreferenced AO sessions cannot be proven to belong to the Supervisor. Automatic deletion of unreferenced sessions is prohibited. Orphan cleanup must be operator-audited. State that spawn orphan identity is not deterministically recoverable. |
| `ADR16R1-002` | `DISPATCH_OPERATION_FOREIGN_KEY_INVALID` | **REVISION_1_REQUIRED** | Blocker | Candidate DDL references `task_attempts(id)`, which does not exist in the P02 schema. Primary key is `attempt_id`. Correct to `FOREIGN KEY (attempt_id) REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT`. |
| `ADR16R1-003` | `BLOCKED_RESUMPTION_VIOLATES_STATE_GRAPH` | **REVISION_1_REQUIRED** | Blocker | Canonical state machine has no `BLOCKED -> RUNNING` edge. D10's proposal to resume execution after operator unblocking under the same attempt violates the graph. AO `blocked` observation must keep TaskState in `RUNNING` with `AO_BLOCKED_DECISION` disposition, or formal escalation must follow `RUNNING -> BLOCKED -> HUMAN_REQUIRED`. |
| `ADR16R1-004` | `TERMINAL_GENERATION_SEMANTICS_MISSTATED` | **REVISION_1_REQUIRED** | Major | `SessionView.TerminalGeneration` is an opaque `string` (`RuntimeLaunchID`), not an integer counter or monotonic number. Comparisons are `MATCH` or `MISMATCH`. Remove all claims of numeric increment or ordering. |
| `ADR16R1-005` | `WORKERSESSION_CANONICAL_FIELD_MAPPING_INCOMPLETE` | **REVISION_1_REQUIRED** | Major | Canonical `docs/05` defines `runtime_type`, `worker_agent_id`, and `status`. Model A persistence must explicitly map these canonical fields rather than silently omitting them. |
| `ADR16R1-006` | `STOP_OPERATION_PROVENANCE_SCHEMA_INCOMPLETE` | **REVISION_1_REQUIRED** | Major | D11 requires stop operations to bind `pair_id`, `task_id`, `contract_id`, `attempt_id`, timestamps, and actor. Section 27 schema omitted these fields. Reconcile schema to match D11 provenance requirements. |
| `ADR16R1-007` | `RESTART_STOP_SAGA_COVERAGE_INCOMPLETE` | **REVISION_1_REQUIRED** | Major | Startup scanner must enumerate all non-terminal stop operations, including `STOP_CALL_SUCCEEDED` lacking `STOP_TERMINATION_CONFIRMED`, not just `STOP_REQUESTED`. |
| `ADR16R1-008` | `D8_DOWNSTREAM_FAILURE_POLICY_OVERREACH` | **REVISION_1_REQUIRED** | Major | D8 must not predefine downstream failure rules such as "no modified files = FAILED". Analysis or verification tasks may produce no modified files. D8 defines ambiguity handoff only; P04 owns evidence evaluation. |
| `ADR16R1-009` | `BACKGROUND_RECOVERY_RETRY_POLICY_UNDEFINED` | **REVISION_1_REQUIRED** | Major | No recovery retry interval policy exists in the approved inventory of 8 unhardcoded policies. Use existing `SUPERVISOR_ACTIVITY_POLL_INTERVAL` for recovery scheduling while affected Pair is locked; do not invent new operational policies. |

---

## 2. Governance Determination

1. `PROPOSAL_P03_002 = EXTERNAL_APPROVED` (remains frozen).
2. `ADR_016 = REVISION_1_REQUIRED`.
3. `ADR_016_ACCEPTANCE = NOT_GRANTED`.
4. `TASK_P03_003 = NOT_RELEASED`.
5. `P03_CODE = HELD_PENDING_ADR_016_APPROVAL`.
6. `ACTIVE_GATE = P03_ADR_016_REVISION_1`.
