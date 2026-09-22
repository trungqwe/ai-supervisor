# P03 ADR-016 External Re-Audit 004 Report

**Audit Target**: docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md (Revision 4)
**Audited Commit**: dedcd9324b77b07d76cf7af36b363510975dfeb2
**Authority**: External Supervisor
**Status**: EXTERNAL_AUDIT_PROCESSED
**Date**: 2026-09-23

---

## 1. Executive Summary

External Supervisor Re-Audit 004 was performed on remote commit dedcd9324b77b07d76cf7af36b363510975dfeb2. Remote verification confirmed exactly one commit ahead of baseline f5993970b0fcb97f6dccace9a377404958fca5c, strictly modifying the four whitelisted documentation files.

Architecturally, Revision 4 resolved the majority of critical blockers:
- Acknowledged that pinned AO /kill provides only a client-side generation precheck without an atomic generation fence (PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT).
- Prohibited automatic re-kill on daemon restart (STOP_REISSUE_REQUIRES_HUMAN).
- Separated quarantine_state safety gating from
ecovery_disposition diagnostics on 	ask_attempts.
- Bound stop operations to an explicit purpose (RUNNING_ATTEMPT_STOP, QUARANTINE_CLEANUP, PAIR_MAINTENANCE).
- Made worker_sessions.worktree_path nullable (TEXT NULL) due to public contract unavailability in pinned AO v0.13.0.
- Persisted absolute confirmation_deadline_at across restarts.

However, literal source audit revealed residual token drift (ADR16R4-008 NOT_CLOSED) and five new findings requiring Revision 5:
- Title/header/status still labeled the document Revision 3/4 (ADR16R5-001).
- The worker report claimed a DDL and token set that diverged from the committed artifact (ADR16R5-002).
- Open attempt lifecycle across BLOCKED -> HUMAN_REQUIRED remained unresolved (ADR16R5-003).
- Ambiguous /kill transport outcome was unmodeled (ADR16R5-004).
- An impossible 'zero race' stop objective remained in the Problem Statement (ADR16R5-005).

---

## 2. Findings Disposition

| Finding ID | Title | Severity | Revision 4 Audit Disposition |
| :--- | :--- | :--- | :--- |
| ADR16R1-001..009 | Historical Revision 1 Findings | HIGH/CRITICAL | **CLOSED** |
| ADR16R2-001..009 | Historical Revision 2 Findings | HIGH/CRITICAL | **CLOSED** |
| ADR16R3-001..010 | Historical Revision 3 Findings | MEDIUM/HIGH | **CLOSED** |
| ADR16R4-001 | Generation Fence TOCTOU | CRITICAL | **CLOSED** |
| ADR16R4-002 | Attempt Quarantine Separation | CRITICAL | **CLOSED** |
| ADR16R4-003 | Stop Operation Purpose Context | CRITICAL | **SUBSTANTIVELY_CLOSED** |
| ADR16R4-004 | Stop Stage Provenance Preservation | HIGH | **CLOSED** |
| ADR16R4-005 | Pair Provisioning Exclusivity Guard | CRITICAL | **CLOSED** |
| ADR16R4-006 | Worktree Path Public Source Resolution | CRITICAL | **CLOSED** |
| ADR16R4-007 | Restart-Stable Stop Deadline | HIGH | **CLOSED** |
| ADR16R4-008 | Cross-Section Vocabulary Drift | MEDIUM | **NOT_CLOSED** |
| ADR16R5-001 | Revision Identity & Governance Drift | HIGH | **REVISION_5_REQUIRED** |
| ADR16R5-002 | Worker Report Not Bound to Committed Artifact | CRITICAL (PROCESS) | **REVISION_5_REQUIRED** |
| ADR16R5-003 | Human-Required Open Attempt Lifecycle Unresolved | CRITICAL | **REVISION_5_REQUIRED** |
| ADR16R5-004 | Stop Transport Outcome Ambiguity Unmodeled | HIGH | **REVISION_5_REQUIRED** |
| ADR16R5-005 | Impossible Zero-Race Objective Remains | MEDIUM | **REVISION_5_REQUIRED** |

---

## 3. Formal Verdict

`	ext
PROPOSAL_P03_002 = EXTERNAL_APPROVED

ADR_016 = REVISION_5_REQUIRED
ADR_016_ACCEPTANCE = NOT_GRANTED

TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_PENDING_ADR_016_APPROVAL

ACTIVE_GATE = P03_ADR_016_REVISION_5
`
