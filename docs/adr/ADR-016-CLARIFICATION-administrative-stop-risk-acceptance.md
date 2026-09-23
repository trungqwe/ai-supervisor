# ADR-016 clarification: administrative stop risk acceptance

**Status:** EXTERNAL_APPROVED by External Supervisor decision of 2026-09-23. This clarification supplements ADR-016 D5 and the approved restore addendum §5(3)–(4) only. It changes no schema, AO wire contract, TaskState edge, physical evidence rule, or previously accepted ADR text.

## Decision

`ADMINISTRATIVE_RISK_ACCEPTED` is an audit `event_type` recording a verified operator's acceptance of residual execution risk for explicitly named WorkerSession and attempt lineages. It is not `QUARANTINE_RESOLVED_ADMINISTRATIVE`; the latter is emitted only by the later D6 transaction that actually clears quarantine. The authority scope for recovery is independent of restore authorization. The host must provide an authenticated principal at the trusted boundary; a library test principal is insufficient to enable runtime recovery.

For Class C, one transaction checks exact stop operation, Pair, session, generation, purpose, attempt snapshot, old stage and old resolution by CAS. It changes only `resolution_state` to `ADMINISTRATIVE_RISK_ACCEPTED`, sets `resolved_at`, and appends scoped `ADMINISTRATIVE_RISK_ACCEPTED` audit events. Wire stage, call timestamps, deadline and prior evidence remain immutable. Each audit records principal, authority scope, reason, evidence references, old/new resolution, optional restore link and each accepted lineage. No AO effect or physical proof is asserted, and quarantine remains. `TERMINATION_CONFIRMED` and another winning terminal decision cannot be overwritten. An exact replay reads the committed decision without writing; a conflicting replay fails. An active effect owner's unresolved intent cannot be bypassed. Linked recovery checks restore claim and owner before Tx D.

For Class B, `STOP_TARGET_ABSENT` remains the stop resolution and 404 remains absence evidence. The verified operator's risk acceptance is appended as separate scoped `ADMINISTRATIVE_RISK_ACCEPTED` audit in an atomic CAS guarded transaction; replay and competing decisions are guarded. No physical proof is inferred and Tx D does not require changing the stop resolution merely to accept Class B evidence.

Linked order is audited stop reconciliation, restore Tx D, then lineage complete D6 clearance. Tx D uses an administrative basis if any affected lineage has only accepted risk. D6 alone clears quarantine and emits `QUARANTINE_RESOLVED_ADMINISTRATIVE` for each administratively cleared lineage. Acceptance never extends to an unlisted lineage. `DELIVERY_OUTCOME_UNKNOWN` is unchanged.

**Implementation status:** clarification approved; TASK-P03-003C implementation remains subject to independent audit. `AUTOMATIC_RESTORE=DISABLED`; 3D remains `NOT_RELEASED`.
