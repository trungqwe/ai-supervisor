# P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_004.md — External Supervisor Re-Audit Record (PROPOSAL-P03-002 Revision 4)

> **Audited Commit**: 38154d7fa65888cbf25b0bc5206384198a9a8a40
> **Upstream Authority**: Untrivial-ai/agent-orchestrator v0.13.0 (Commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: PROPOSAL_P03_002 = REVISION_5_REQUIRED
> **Architecture Impact**: P03_ARCHITECTURE_CHANGE = YES
> **ADR Requirement**: P03_ADR_REQUIRED = YES
> **ADR-016 Authorization**: ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET
> **TASK_P03_001 Status**: EXTERNAL_AUDIT_APPROVED
> **TASK_P03_002 Status**: EXTERNAL_AUDIT_APPROVED
> **TASK_P03_003 Status**: NOT_RELEASED
> **P03 Code Status**: HELD
> **Active Gate**: P03_TASK_003_PROPOSAL_REVISION_5

---

## 0. External Supervisor Re-Audit Decision Summary

Independent external re-audit of PROPOSAL-P03-002 Revision 4 (commit 38154d7fa65888cbf25b0bc5206384198a9a8a40) is complete.

Prior Revision-4 findings:
- P03T3PR4-001 = CLOSED
- P03T3PR4-002 = CLOSED

Revision 4 correctly establishes:
`	ext
NEW_SESSION_OR_GENERATION != RESOLUTION_OF_PRIOR_UNCERTAIN_EXECUTION
`
and correctly separates source facts from ADR architectural policy.

However, three (3) internal contract contradictions remain.

**Verdict**: PROPOSAL_P03_002 = REVISION_5_REQUIRED.

---

## 1. External Re-Audit Findings Matrix

| Finding ID | Title | Verdict | Severity | Remediation Mandate |
|---|---|---|---|---|
| P03T3PR5-001 | QUARANTINE_RESOLUTION_LEDGER_CONTRADICTION | **REVISION_5_REQUIRED** | Blocker | Section 9 defines resolution classes A, B, C, D, but D5 currently states too broadly that quarantine clears *only* after old execution termination is authoritatively confirmed. That conflicts with Class B (authoritative absence) and Class C (human risk acceptance) being legitimate ADR candidates. Distinguish PHYSICAL_EXECUTION_RESOLUTION (positive proof old execution cannot continue) from ADMINISTRATIVE_RISK_RESOLUTION (operator acknowledges inability to prove termination and authorizes governed replacement/recovery path; must be durably audited and must NOT be reported as TERMINATION_CONFIRMED or WORKER_STOPPED). If ADR-016 chooses automatic /kill, quarantine clears only after confirmed termination; if Class B/C is chosen, ADR-016 defines resolution type, evidence, authority, audit record, and preserves that old execution remains physically unresolved. |
| P03T3PR5-002 | D8_REPORT_PROBE_VIOLATES_TASK_P03_003_BOUNDARY | **REVISION_5_REQUIRED** | Blocker | Approved task ownership from PROPOSAL-P03-001: TASK-P03-003 is strictly Lifecycle Observation & State Reconciliation; TASK-P03-004 owns raw GetWorkspaceFile transport; P04 owns WorkerReport semantic interpretation and evidence collection. D8 must NOT propose TASK-P03-003 directly fetching report artifacts, probing workspace files, parsing reports, or validating reports. Remove wording such as 'probe report artifact' and 'Inspect generation, probe report'. D8 decision space is lifecycle-only: (A) fail closed DISPATCHED -> FAILED, (B) preserve DISPATCHED and escalate lifecycle ambiguity, (C) persist ambiguity disposition and hand off to downstream phase (TASK-P03-004 / P04), or (D) other canonical lifecycle architecture. |
| P03T3PR5-003 | BLOCKED_ATTEMPT_END_SEMANTICS_CONTRADICTORY | **REVISION_5_REQUIRED** | Major | D10 marked TaskAttempt.ended_at = DO NOT SET as unresolved under the current baseline, while D12 listed RUNNING -> BLOCKED as a terminal outcome that must set ended_at = now. Separate the atomicity invariant from whether BLOCKED ends an attempt. Terminal failures (RUNNING/DISPATCHED -> FAILED) atomically commit TaskState + ended_at = now + audit log. For RUNNING -> BLOCKED: IF ADR-016 decides BLOCKED ends the attempt, transition + ended_at = now + audit persistence must occur atomically; IF ADR-016 retains the open attempt, ended_at remains NULL; IF ADR-016 chooses no transition, attempt remains open. Cross-reference D10 and D12 as UNRESOLVED_FOR_ADR. Do not label RUNNING -> BLOCKED as an unconditional terminal outcome. |

---

## 2. Governance Determination

1. P03_ARCHITECTURE_CHANGE = YES (amends quarantine resolution taxonomy, D8 task boundary, and BLOCKED attempt-end semantics).
2. P03_ADR_REQUIRED = YES.
3. ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET (must wait until Proposal Revision 5 passes independent External Supervisor re-audit).
4. TASK_P03_003 = NOT_RELEASED.
5. P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION.
6. ACTIVE_GATE = EXTERNAL_SUPERVISOR_P03_TASK_003_PROPOSAL_REAUDIT_005.
