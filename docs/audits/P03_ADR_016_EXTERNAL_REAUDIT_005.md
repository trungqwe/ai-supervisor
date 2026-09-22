# P03 ADR-016 External Re-Audit 005 Report

**Audit Target**: docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md (Revision 5)
**Audited Commit**: 68b3699cfdeb38a9ecb93771fe3a5f2823938a18
**Authority**: External Supervisor
**Status**: EXTERNAL_AUDIT_PROCESSED
**Date**: 2026-09-23

---

## 1. Executive Summary

External Supervisor Re-Audit 005 was performed directly on remote commit 68b3699cfdeb38a9ecb93771fe3a5f2823938a18. Remote verification confirmed exactly one commit ahead of baseline dedcd9324b77b07d76cf7af36b363510975dfeb2, strictly modifying the four whitelisted documentation files without any production code, database migration, proposal, or canonical specification drift.

Architecturally, Revision 5 resolved the most complex safety issues:
- Truthfully modeled /kill client-side TOCTOU provenance and fail-closed handling under pinned session-scoped /kill.
- Modeled stop transport drop ambiguity explicitly as STOP_CALL_OUTCOME_UNKNOWN with fail-closed quarantine and prohibited automatic reissue.
- Specified atomic attempt closure upon formal escalation from BLOCKED to HUMAN_REQUIRED.
- Set worktree_path TEXT NULL to reflect pinned AO v0.13.0 wire DTO reality.
- Corrected pinned AO SessionView shape and /kill wire contract in the report.

However, literal source audit revealed residual token drift (ADR16R4-008), live governance contradictions in AGENTS.md (ADR16R5-001), report integrity claim drift (ADR16R5-002), and four new findings requiring Revision 6:
- ADR16R6-001 (CRITICAL): Crash window between RUNNING -> BLOCKED and BLOCKED -> HUMAN_REQUIRED leaves open attempt unrecovered because D13 startup recovery misses BLOCKED.
- ADR16R6-002 (HIGH): D12 does not define the atomic transaction shape for attempt-closing nonterminal escalation BLOCKED -> HUMAN_REQUIRED.
- ADR16R6-003 (HIGH): Standalone D11 WORKER_STOPPED safety invariant is too weak (isTerminated == true alone does not prove positive stop provenance).
- ADR16R6-004 (HIGH): D6 parenthetical for unresolved provisioning guard predicate is logically reversed (stage NOT IN vs stage IN).

---

## 2. Findings Disposition

| Finding ID | Title | Severity | Revision 5 Audit Disposition |
| :--- | :--- | :--- | :--- |
| ADR16R1-001..009 | Historical Revision 1 Findings | HIGH/CRITICAL | **CLOSED** |
| ADR16R2-001..009 | Historical Revision 2 Findings | HIGH/CRITICAL | **CLOSED** |
| ADR16R3-001..010 | Historical Revision 3 Findings | MEDIUM/HIGH | **CLOSED** |
| ADR16R4-001..007 | Historical Revision 4 Closed Findings | HIGH/CRITICAL | **CLOSED** / **SUBSTANTIVELY_CLOSED** |
| ADR16R4-008 | Cross-Section Vocabulary Drift (Lifecycle Matrix uses undeclared PRE_SEND_BLOCKED) | MEDIUM | **NOT_CLOSED** |
| ADR16R5-001 | Revision Identity & Governance Drift (AGENTS.md live prose retains Revision 4 / Re-Audit 004) | HIGH | **NOT_CLOSED** |
| ADR16R5-002 | Worker Report Integrity (AGENTS.md claim not fully synchronized) | CRITICAL (PROCESS) | **PARTIALLY_CLOSED** |
| ADR16R5-003 | Human-Required Open Attempt Lifecycle Unresolved | CRITICAL | **SUBSTANTIVELY_CLOSED** |
| ADR16R5-004 | Stop Transport Outcome Ambiguity Unmodeled (STOP_CALL_OUTCOME_UNKNOWN) | HIGH | **CLOSED** |
| ADR16R5-005 | Impossible Zero-Race Objective Remains | MEDIUM | **CLOSED** |
| ADR16R6-001 | Blocked Escalation Crash Window Unrecovered (D13 startup recovery misses BLOCKED) | CRITICAL | **REVISION_6_REQUIRED** |
| ADR16R6-002 | D12 Escalation Atomicity Not Reconciled (BLOCKED -> HUMAN_REQUIRED transaction missing) | HIGH | **REVISION_6_REQUIRED** |
| ADR16R6-003 | WORKER_STOPPED Provenance Invariant Too Weak (Regression to unverified termination) | HIGH | **REVISION_6_REQUIRED** |
| ADR16R6-004 | Provisioning Guard Predicate Reversed (stage NOT IN vs stage IN) | HIGH | **REVISION_6_REQUIRED** |

---

## 3. Formal Verdict

`	ext
PROPOSAL_P03_002 = EXTERNAL_APPROVED

ADR_016 = REVISION_6_REQUIRED
ADR_016_ACCEPTANCE = NOT_GRANTED

TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_PENDING_ADR_016_APPROVAL

ACTIVE_GATE = P03_ADR_016_REVISION_6
`
