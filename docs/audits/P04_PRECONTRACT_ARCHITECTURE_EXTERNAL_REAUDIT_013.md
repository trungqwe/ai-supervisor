# External Re-Audit Report 013: Phase P04 Pre-Contract Architecture (Revision 14 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `5fa0f5d291cb5f1952f688c971b8b047d3529bca`
> **Date**: 2026-09-26
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_14`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 013 evaluated the Phase P04 architecture remediation deliverables submitted in commit `5fa0f5d291cb5f1952f688c971b8b047d3529bca` (Revision 14 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 14)
2. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 14)
3. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 14)
4. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_012.md`
5. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

Findings `P04-ARCH-R13-001` (TaskState preservation) and `P04-ARCH-R13-002` (foreign key audit references and principal constraints) are verified **CLOSED_AT_DESIGN_LEVEL**. Findings `P04-ARCH-R13-003` and `P04-ARCH-R13-004` are **PARTIALLY_CLOSED** and follow-up findings `P04-ARCH-R14-001` and `P04-ARCH-R14-002` are opened requiring Revision 15 remediation:
- `P04-ARCH-R14-001`: Alignment with canonical audit model (`domain.AuditEvent` and `audit_events` schema: `actor` column, `details_json.actor_role`, mandatory `pair_id`, sanitized replay comparison, and UNIQUE `event_id` conflict).
- `P04-ARCH-R14-002`: Diagnostic occurrence lifecycle (`occurrence_number INTEGER NOT NULL`, `UNIQUE(attempt_id, hold_reason, diagnostic_fingerprint, occurrence_number)`, JCS identity descriptor participation, atomic MAX+1 algorithm, recurrence after resolution creating occurrence N+1 with distinct audit event).

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_14_REQUIRED`
- `ADR_018 = REVISION_14_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_14_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_14`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 13 Findings (P04-ARCH-R13-001 through R13-004)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R13-001` | Elimination of Erroneous TaskState Transitions to BLOCKED | **CLOSED_AT_DESIGN_LEVEL** | Unilateral transitions to BLOCKED eliminated; TaskState preserved; durable hold created in diagnostic Tx. |
| `P04-ARCH-R13-002` | Foreign Key Audit Constraints & Principal Validation | **CLOSED_AT_DESIGN_LEVEL** | Rejection/resolution audit FKs, non-empty principal check, atomic creation/resolution transactions, fail-closed runtime constraint verified. |
| `P04-ARCH-R13-003` | Multi-Diagnostic Hold Tracking via Fingerprint | **PARTIALLY_CLOSED** | Multi-diagnostic hold tracking added, but missing occurrence lifecycle for recurrence after resolution (`P04-ARCH-R14-002`). |
| `P04-ARCH-R13-004` | Canonical Event ID Derivation & Replay Semantics | **PARTIALLY_CLOSED** | JCS event ID hashing introduced, but model alignment on actor_id/actor_role and pair_id was imprecise (`P04-ARCH-R14-001`). |

---

## 3. New Findings (P04-ARCH-R14-001 and P04-ARCH-R14-002)

### `P04-ARCH-R14-001`: Canonical Audit Model Alignment & Replay Semantics
- **Severity**: High
- **Description**: Previous text described `actor_id` and `actor_role` as independent table columns and failed to mandate `pair_id` in replay comparison. In the canonical database schema and domain model (`domain.AuditEvent`), the column is `actor TEXT NOT NULL`, `pair_id TEXT` is an existing indexed column, and role information belongs in `details_json.actor_role`. Furthermore, the collision occurs on the UNIQUE `event_id` constraint, not the INTEGER PRIMARY KEY `sequence`.
- **Remediation Directive**:
  1. Remove `actor_id` and `actor_role` as independent table columns or struct fields in all specifications.
  2. For system events: set `actor = 'ai-supervisor-daemon'` and `details_json.actor_role = 'SUPERVISOR'`.
  3. For operator resolution: set `actor = verified resolved_by_principal` and `details_json.actor_role = 'OPERATOR'`.
  4. In replay comparison upon duplicate key on UNIQUE `event_id`: compare sanitized fields: `event_type`, `pair_id`, `task_id`, `contract_id`, `attempt_id`, `actor`, and canonical `details_json`. Mandatory inclusion of `pair_id`.
  5. Explicitly ignore only `sequence`, `timestamp`, `prev_hash`, and `event_hash`.
  6. Designate conflicts accurately as UNIQUE `event_id` conflicts.
  7. Incoming events must be sanitized and canonicalized via `audit.SanitizeAuditEvent` before append and before readback comparison.
  8. Sanitization failure or sanitized-key collision must fail closed; never compute fingerprints or hashes from unnormalized data.

### `P04-ARCH-R14-002`: Diagnostic Occurrence Lifecycle
- **Severity**: High
- **Description**: If a diagnostic violation was resolved by an operator and the same diagnostic recurred later on the same attempt (e.g. dirty worktree re-introduced), the existing schema either collided on UNIQUE index or overwrote history. A durable multi-occurrence lifecycle is required.
- **Remediation Directive**:
  1. Add column `occurrence_number INTEGER NOT NULL CHECK (typeof(occurrence_number) = 'integer' AND occurrence_number > 0)` to `review_integrity_holds`.
  2. Add table constraint: `UNIQUE(attempt_id, hold_reason, diagnostic_fingerprint, occurrence_number)`.
  3. Maintain partial unique active dedup index:
     `CREATE UNIQUE INDEX idx_review_integrity_holds_active_dedup ON review_integrity_holds(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE';`
  4. Keep `rejection_audit_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`.
  5. Include `occurrence_number` and `hold_id` in the RFC 8785 JCS identity descriptor for the rejection audit event.
  6. Specify the atomic creation algorithm:
     a. `BEGIN IMMEDIATE` diagnostic transaction.
     b. If ACTIVE hold already exists with same tuple: exact replay returns existing hold.
     c. If no ACTIVE hold: `occurrence_number = COALESCE(MAX(occurrence_number), 0) + 1` for the tuple.
     d. Derive new distinct `hold_id` and rejection `event_id` from JCS descriptor containing `occurrence_number`.
     e. Append sanitized audit event.
     f. Insert ACTIVE hold.
     g. Commit.
     h. On unique/CAS race: read back ACTIVE hold; exact tuple returns idempotent success, mismatch fails closed.
  7. Lifecycle after occurrence N is RESOLVED:
     - If the same diagnostic is re-observed, create occurrence N+1 with a new rejection audit event and new ACTIVE hold.
     - Hold N and historical audit events remain immutable.
     - Pipeline admission remains closed due to ACTIVE hold N+1.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 15 incorporating R14-001 and R14-002.
2. Remediate `DRAFT-ADR-018` to Revision 15 incorporating R14-001 and R14-002.
3. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 15 incorporating R14-001 and R14-002.
4. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_14`.
5. Maintain `PROPOSAL-P04-002` at Revision 7 (latency semantics unchanged).
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
