# P03_ADR_016_EXTERNAL_REAUDIT_001.md — Independent External Supervisor Re-Audit of ADR-016 Revision 1

> **Audit Type**: Independent External Supervisor Re-Audit 001
> **Audited Document**: docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md (Revision 1)
> **Audited Commit**: 5fadbeef86f6bd3a1b782e42c0affdfc32c3165
> **Proposal Authority**: PROPOSAL-P03-002 Revision 6 (EXTERNAL_APPROVED, commit 8709af4b6aaf8613c69a10514971e320e3f90048)
> **Pinned Upstream Authority**: Untrivial-ai/agent-orchestrator 0.13.0 (commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6)
> **Audit Date**: 2026-09-23

---

## 1. Executive Summary & Verdict

Independent External Supervisor Re-Audit 001 of ADR-016 Revision 1 (commit 5fadbeef86f6bd3a1b782e42c0affdfc32c3165) is complete.

All nine previous findings from Initial External Audit (ADR16R1-001 through ADR16R1-009) have been verified as **CLOSED**.

However, detailed architectural consistency analysis revealed nine new findings (ADR16R2-001 through ADR16R2-009) requiring Revision 2 remediation before formal ADR acceptance can be considered.

### Formal Verdicts
- **PROPOSAL_P03_002** = EXTERNAL_APPROVED
- **ADR_016** = REVISION_2_REQUIRED
- **ADR_016_ACCEPTANCE** = NOT_GRANTED
- **TASK_P03_003** = NOT_RELEASED
- **P03_CODE** = HELD_PENDING_ADR_016_APPROVAL
- **ACTIVE_GATE** = P03_ADR_016_REVISION_2

---

## 2. Status of Previous Findings (Revision 1)

| Finding ID | Title | Re-Audit Status |
| :--- | :--- | :--- |
| ADR16R1-001 | ORPHAN_REAPER_OWNERSHIP_UNPROVEN | **CLOSED** |
| ADR16R1-002 | DISPATCH_OPERATION_FOREIGN_KEY_INVALID | **CLOSED** |
| ADR16R1-003 | BLOCKED_RESUMPTION_VIOLATES_STATE_GRAPH | **CLOSED** |
| ADR16R1-004 | TERMINAL_GENERATION_SEMANTICS_MISSTATED | **CLOSED** |
| ADR16R1-005 | WORKERSESSION_CANONICAL_FIELD_MAPPING_INCOMPLETE | **CLOSED** |
| ADR16R1-006 | STOP_OPERATION_PROVENANCE_SCHEMA_INCOMPLETE | **CLOSED** |
| ADR16R1-007 | RESTART_STOP_SAGA_COVERAGE_INCOMPLETE | **CLOSED** |
| ADR16R1-008 | D8_DOWNSTREAM_FAILURE_POLICY_OVERREACH | **CLOSED** |
| ADR16R1-009 | BACKGROUND_RECOVERY_RETRY_POLICY_UNDEFINED | **CLOSED** |

---

## 3. New Findings (Revision 2 Requirements)

### ADR16R2-001: CANONICAL_STATE_GRAPH_SUMMARY_INVALID
- **Severity**: HIGH (Architectural Soundness)
- **Description**: Section 5 describes the normal state graph sequence as DRAFT -> READY -> DISPATCHED -> RUNNING -> REVIEWING. However, internal/workflow/state_machine.go contains no direct RUNNING -> REVIEWING edge. The canonical workflow graph requires execution through RUNNING -> REPORT_READY -> EVIDENCE_READY -> REVIEWING.
- **Required Action**: Correct the normal path to faithfully include REPORT_READY and EVIDENCE_READY and enumerate canonical executable edges without claiming non-existent direct transitions.

### ADR16R2-002: SPAWN_PROVISIONING_QUARANTINE_NOT_DURABLY_MODELLED
- **Severity**: HIGH (Restart Durability)
- **Description**: Finding ADR16R1-001 established that unowned AO sessions must not be killed automatically and that pre-binding spawn failures cause Pair lane quarantine. However, if a crash occurs between spawn request and session creation, no worker_sessions row exists yet. Storing quarantine state on worker_sessions.quarantine_state fails because the entity does not exist.
- **Required Action**: Durably model Pair provisioning operations via a dedicated record (pair_provisioning_operations) tracking stages PROVISION_REQUESTED, PROVISION_CONFIRMED, PROVISION_FAILED, and PROVISION_RESOLVED queryable by the pre-dispatch guard.

### ADR16R2-003: ATOMIC_TERMINAL_TRANSITION_REFERENCES_UNDECLARED_COLUMN
- **Severity**: MEDIUM (Schema Consistency)
- **Description**: D12 states that AtomicTerminalTransition writes 	ask_attempts.terminal_error = error_details. Neither the current production TaskAttempt entity nor SQLite 	ask_attempts schema has a 	erminal_error column, and Section 27 does not declare one.
- **Required Action**: Remove 	ask_attempts.terminal_error from AtomicTerminalTransition. Machine-readable disposition resides in
ecovery_disposition, and detailed error context resides in append-only udit_events.details_json.

### ADR16R2-004: STOP_RECOVERY_REISSUE_NOT_GENERATION_FENCED
- **Severity**: HIGH (Safety Invariant)
- **Description**: D13 stop recovery stated that if an operation is in STOP_REQUESTED, the recovery scanner may reissue /kill if the session exists and is alive. However, if the session has undergone restart/replacement, its 	erminalGeneration will mismatch the target attempt. Reissuing /kill would terminate an unrelated subsequent execution generation.
- **Required Action**: Enforce generation fencing before reissuing /kill: verify current terminalGeneration == stop_operation.terminal_generation using opaque string equality. If mismatched, kill is prohibited, disposition is marked STOP_GENERATION_MISMATCH, and resolution escalates to administrative triage.

### ADR16R2-005: AO_404_MISCLASSIFIED_AS_STOP_TERMINATION_CONFIRMED
- **Severity**: HIGH (Truthful Observation)
- **Description**: D13 allowed HTTP 404 on GET /sessions/{id} to advance stop operations to STOP_TERMINATION_CONFIRMED and record WORKER_STOPPED. HTTP 404 proves absence of the public session identity, but does not prove affirmative process termination or intentional stop provenance.
- **Required Action**: HTTP 404 must produce disposition STOP_TARGET_ABSENT (Class B administrative absence), not STOP_TERMINATION_CONFIRMED, and must NOT log WORKER_STOPPED. Only positive observation of isTerminated == true with matching generation proves intentional stop completion.

### ADR16R2-006: D8_FALLBACK_TRANSITION_ORDER_CONTRADICTION
- **Severity**: MEDIUM (State Machine Flow)
- **Description**: D8 stated that on observing idle post-send without an active window, the task transitions DISPATCHED -> RUNNING, and if downstream handoff is unavailable, transitions DISPATCHED -> FAILED. Once transitioned to RUNNING, a transition DISPATCHED -> FAILED is invalid.
- **Required Action**: Clarify transition ordering: verify downstream handoff readiness before committing DISPATCHED -> RUNNING. If unavailable at pre-transition checkpoint, transition DISPATCHED -> FAILED. Any subsequent failure must use canonical RUNNING -> FAILED.

### ADR16R2-007: ADMINISTRATIVE_RISK_ACCEPTANCE_CONTRADICTS_ZERO_DUPLICATE_GUARANTEE
- **Severity**: MEDIUM (Documentation Integrity)
- **Description**: ADR-016 permits Class C administrative risk overrides where an operator accepts residual risk that an unobservable process may still be running. Claiming absolute 'Zero Duplicate Execution under all circumstances' contradicts the existence of this human override.
- **Required Action**: State that automated duplicate prevention is fail-closed (AUTOMATED_DUPLICATE_DISPATCH_PREVENTION = FAIL_CLOSED), automated retries are strictly prohibited while unresolved, and Class C is an explicit human risk override that acknowledges residual risk.

### ADR16R2-008: DISPATCH_OPERATION_CARDINALITY_UNENFORCED
- **Severity**: HIGH (Saga Integrity)
- **Description**: A TaskAttempt represents a single execution/retry iteration. A second dispatch saga must never be created for the same attempt. The dispatch_operations candidate DDL permitted multiple rows for the same ttempt_id.
- **Required Action**: Add UNIQUE(attempt_id) constraint on dispatch_operations and formalize invariant # ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION.

### ADR16R2-009: WORKERSESSION_ONE_TO_ONE_NOT_FULLY_ENFORCED
- **Severity**: HIGH (Relational Integrity)
- **Description**: While worker_sessions had pair_id PRIMARY KEY, it lacked a UNIQUE constraint on session_id. This allowed the same AO session_id to be bound to multiple Pair records concurrently.
- **Required Action**: Add session_id TEXT NOT NULL UNIQUE on worker_sessions and formalize invariant AO_SESSION_IDENTITY MUST_BIND_TO_AT_MOST_ONE_PAIR.

---

## 4. Next Required Action
Remediate ADR-016 to Revision 2 closing findings ADR16R2-001 through ADR16R2-009. Production coding remains held (TASK_P03_003 = NOT_RELEASED).
