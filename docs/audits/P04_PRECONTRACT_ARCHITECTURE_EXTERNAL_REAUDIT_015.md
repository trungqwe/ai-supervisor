# External Re-Audit Report 015: Phase P04 Pre-Contract Architecture (Revision 16 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `43b22bafa4f8b7c4e99ec17e80d8e076c0ab80a6`
> **Date**: 2026-09-27
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_16`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 015 evaluated the Phase P04 architecture remediation deliverables submitted in commit `43b22bafa4f8b7c4e99ec17e80d8e076c0ab80a6` (Revision 16 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 16)
2. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 16)
3. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 16)
4. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_014.md`
5. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

Findings `P04-ARCH-R15-001` (elimination of circular self-reference in `hold_id` derivation via two distinct RFC 8785 JCS descriptors with explicit domain discriminator `kind: "review_integrity_hold"`) and `P04-ARCH-R15-002` (resolution crash/replay semantics and concurrent caller race handling with deterministic `resolution_event_id`, atomic CAS, idempotent retry on lost response, and zero orphan audit events on lost CAS races) are verified **CLOSED_AT_DESIGN_LEVEL**.

However, a structural persistence ownership inconsistency was identified during sequencing validation. In Revision 16, Subtask P04A was tasked with recording dirty-intake diagnostics by writing into `review_integrity_holds`, yet the DDL for `review_integrity_holds` was defined under Schema v9 owned by Subtask P04D. Furthermore, the specification described Subtask P04D as a blanket persistence orchestrator for all audit events, contradicting the requirement that P04A autonomously commits Transaction A and its diagnostic transactions.

Follow-up finding `P04-ARCH-R16-001` is recorded, requiring Revision 17 remediation:
- `P04-ARCH-R16-001`: Persistence Ownership Alignment & Contract Sequencing (relocate `review_integrity_holds` to Schema v6 owned by P04A; demarcate transaction and audit event mutation authorities; eliminate ambiguous "sole orchestrator" statements; and formalize contract sequencing P04A -> P04B -> P04C -> P04D).

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_16_REQUIRED`
- `ADR_018 = REVISION_16_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_16_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_16`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 15 Findings (P04-ARCH-R15-001 and R15-002)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R15-001` | Elimination of Circular Self-Reference in `hold_id` Derivation | **CLOSED_AT_DESIGN_LEVEL** | Fully resolved via decoupled descriptors: Descriptor A (`hold_identity_descriptor`, `kind: "review_integrity_hold"`, without `hold_id`) derives `hold_id`; Descriptor B (`rejection_event_identity_descriptor`) embeds pre-computed `hold_id`. Strict deterministic RFC 8785 JCS without UUID fallback. |
| `P04-ARCH-R15-002` | Resolution Crash/Replay & Concurrent Caller Race Handling | **CLOSED_AT_DESIGN_LEVEL** | Fully resolved via deterministic `resolution_event_id` (Descriptor C), atomic resolution transaction with CAS `affected_rows == 1`, idempotent retry on lost response with exact semantic readback, and total rollback of losing callers to guarantee zero orphan audit events. |

---

## 3. New Finding (P04-ARCH-R16-001)

### `P04-ARCH-R16-001`: Persistence Ownership Alignment & Contract Sequencing
- **Severity**: High
- **Description**:
  1. *Schema Ownership Anomaly*: `review_integrity_holds` DDL was assigned to Schema v9 (owned by P04D). However, Subtask P04A executes the clean worktree verification and must atomically record dirty-intake diagnostics (`EVIDENCE_COLLECTION_FAILED` and active hold insert) during Transaction A intake failure. P04A cannot insert into a table defined in a future migration (v9).
  2. *Ambiguous CAS Orchestration*: Section 7.1 stated that Subtask P04D was a blanket persistence orchestrator owning all audit events, which contradicted P04A's autonomous ownership of Transaction A, `WORKSPACE_BINDING_CREATED`, and intake failure diagnostic transactions.
  3. *Contract Sequencing*: P04A must be releaseable, implementable, and testable independently without depending on P04D tables or admission gates.
- **Remediation Directives**:
  1. **Relocate `review_integrity_holds` to Schema v6 (Owned by P04A)**:
     - Schema v6 (Owned by Subtask P04A) contains:
       * `attempt_workspace_bindings`
       * `worker_claims`
       * `review_integrity_holds`
       * All indexes and triggers for the above three tables (`idx_review_integrity_holds_active_dedup`, `trg_review_integrity_holds_lineage_guard`, `trg_review_integrity_holds_cas_guard`, `trg_review_integrity_holds_no_delete`).
     - Schema v9 (Owned by Subtask P04D) contains:
       * `task_verification_leases`
       * `evidence_sets`
       * `review_artifacts`
       * `review_bundles`
       * All indexes and triggers for the above four tables.
       * Schema v9 upgrades from a database that already contains `review_integrity_holds` and does NOT recreate it.
  2. **Demarcate Mutation Authority**:
     - **Subtask P04A owns**:
       * Transaction A.
       * `WORKSPACE_BINDING_CREATED` and audit events belonging to Transaction A.
       * Diagnostic transaction upon intake failure: appends `EVIDENCE_COLLECTION_FAILED` and inserts an `ACTIVE` hold into `review_integrity_holds`.
       * Shared hold creation primitive and Descriptors A and B.
       * Behavior tests for dirty intake, replay, occurrence increment, and rollback atomicity.
       * Completing P04A does NOT open runtime admission for Phase P04.
     - **Subtask P04D owns**:
       * Verification leases, Transaction B, Transaction C, and ReviewBundle synthesis.
       * Diagnostic hold creation arising from Tx B or Tx C failures (`BUNDLE_HASH_CONFLICT`, `INVARIANT_MISMATCH`).
       * `REVIEW_INTEGRITY_CONFLICT`.
       * Hold resolution orchestration, `REVIEW_INTEGRITY_HOLD_RESOLVED`, and Descriptor C.
       * Startup recovery scanning for `ACTIVE` holds before opening P04 runtime admission.
       * Reuses the identical `review_integrity_holds` schema and store API created by P04A without duplicate DDL.
     - **Subtasks P04B and P04C**:
       * Pure in-memory collectors.
       * ZERO SQLite writes, ZERO audit event appends, ZERO hold mutations.
  3. **Clarify Persistence Ownership Discipline**:
     - Eliminate ambiguous blanket orchestrator claims across pipeline transactions.
     - Replace with precise boundaries: P04A is the sole persistence owner of Schema v6 and Transaction A; P04D is the sole persistence owner of Schema v9, Tx B, Tx C, and pipeline CAS orchestration.
     - Every audit event must be appended by its authoritative transaction owner, ensuring atomicity with database state changes.
     - No two subtasks share ownership of a single transaction or define DDL for the same table.
  4. **Strict Contract Sequencing**:
     - Maintain execution graph: P04A -> P04B -> P04C -> P04D.
     - P04A can complete its schema migrations, store primitives, and behavior tests independently.
     - P04 runtime admission remains closed upon P04A/B/C completion; only opened by P04D after startup recovery.
     - Schema Migration v9 upgrades from a schema that already contains `review_integrity_holds`. Both fresh migration (`v0 -> v6 -> v9`) and historical migration (`v6 -> v9`) must be tested and verified with `foreign_keys = ON`.
     - Subtask P04D cannot alter hold semantics or DDL without an approved contract revision.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 17 incorporating R16-001.
2. Remediate `DRAFT-ADR-018` to Revision 17 incorporating R16-001.
3. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 17 incorporating R16-001.
4. Add the Final Architecture Readiness Matrix covering all six tracked design blockers (`WORKTREE_BINDING`, `GIT_EVIDENCE_AUTHORITY`, `VERIFICATION_ISOLATION`, `REVIEW_SCHEMA_RECONCILIATION`, `EVIDENCE_ATOMICITY`, `INERT_AO_HARNESS`) with proposed disposition `READY_FOR_EXTERNAL_APPROVAL`.
5. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_16`.
6. Maintain `PROPOSAL-P04-002` at Revision 7 (latency semantics unchanged).
7. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
