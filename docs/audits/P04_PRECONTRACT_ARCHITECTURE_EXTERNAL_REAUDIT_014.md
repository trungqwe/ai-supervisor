# External Re-Audit Report 014: Phase P04 Pre-Contract Architecture (Revision 15 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `e349f32994edca259405465a2ecda576c29e62f2`
> **Date**: 2026-09-26
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_15`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 014 evaluated the Phase P04 architecture remediation deliverables submitted in commit `e349f32994edca259405465a2ecda576c29e62f2` (Revision 15 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 15)
2. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 15)
3. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 15)
4. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_013.md`
5. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

Finding `P04-ARCH-R14-001` (canonical audit model alignment: actor column, `details_json.actor_role`, mandatory `pair_id`, sanitized replay comparison, and UNIQUE `event_id` conflict) is verified **CLOSED_AT_DESIGN_LEVEL**. Finding `P04-ARCH-R14-002` is **PARTIALLY_CLOSED** because the occurrence lifecycle was established, but the specification introduced a circular dependency in `hold_id` derivation and underspecified resolution crash/replay semantics for concurrent operators. Follow-up findings `P04-ARCH-R15-001` and `P04-ARCH-R15-002` are opened requiring Revision 16 remediation:
- `P04-ARCH-R15-001`: Elimination of circular self-reference in `hold_id` derivation via two distinct RFC 8785 JCS descriptors with explicit domain discriminator.
- `P04-ARCH-R15-002`: Resolution crash/replay semantics and concurrent caller race handling with deterministic `resolution_event_id`, atomic CAS, idempotent retry on lost response, and zero orphan audit events on lost CAS races.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_15_REQUIRED`
- `ADR_018 = REVISION_15_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_15_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_15`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 14 Findings (P04-ARCH-R14-001 and R14-002)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R14-001` | Canonical Audit Model Alignment & Replay Semantics | **CLOSED_AT_DESIGN_LEVEL** | Fully aligned with `domain.AuditEvent` and `audit_events` schema (`actor TEXT NOT NULL`, `details_json.actor_role`, mandatory `pair_id`, sanitized replay comparison, and UNIQUE `event_id` conflict). |
| `P04-ARCH-R14-002` | Diagnostic Occurrence Lifecycle | **PARTIALLY_CLOSED** | Occurrence lifecycle added (`occurrence_number INTEGER NOT NULL`, unique constraints, recurrence after resolution); but circular reference in `hold_id` derivation and resolution concurrency need remediation. |

---

## 3. New Findings (P04-ARCH-R15-001 and P04-ARCH-R15-002)

### `P04-ARCH-R15-001`: Elimination of Circular Self-Reference in `hold_id` Derivation
- **Severity**: High
- **Description**: The Revision 15 specification stated that `hold_id` is derived from a JCS descriptor that itself contained `hold_id`. To avoid circular self-reference, hold derivation must be decoupled into two separate descriptors:
  1. `hold_identity_descriptor`:
     ```json
     {
       "version": 1,
       "kind": "review_integrity_hold",
       "pair_id": "...",
       "task_id": "...",
       "contract_id": "...",
       "attempt_id": "...",
       "hold_reason": "...",
       "diagnostic_fingerprint": "...",
       "occurrence_number": N
     }
     ```
     `hold_id = "hold-" + SHA256(RFC8785_JCS(hold_identity_descriptor))`
  2. `rejection_event_identity_descriptor`:
     ```json
     {
       "version": 1,
       "event_type": "...",
       "pair_id": "...",
       "task_id": "...",
       "contract_id": "...",
       "attempt_id": "...",
       "reason": "...",
       "diagnostic_fingerprint": "...",
       "sanitized_input_fingerprint": "...",
       "occurrence_number": N,
       "hold_id": "<pre-computed from descriptor 1>"
     }
     ```
     `rejection_event_id = SHA256(RFC8785_JCS(rejection_event_identity_descriptor))`
- **Remediation Directives**:
  1. `hold_identity_descriptor` MUST NOT contain `hold_id`.
  2. Prohibit UUID / random fallback; derivation must be 100% deterministic.
  3. Exact retries with identical lineage/diagnostic/occurrence must produce the identical `hold_id` and `rejection_event_id`.
  4. Occurrence N+1 must produce distinct `hold_id` and `event_id` from occurrence N.
  5. Include domain discriminator `kind: "review_integrity_hold"` to prevent collision with audit event hashes.
  6. All producers of integrity holds (including `REVIEW_INTEGRITY_CONFLICT`) must follow this unified derivation algorithm.

### `P04-ARCH-R15-002`: Resolution Crash/Replay & Concurrent Caller Race Handling
- **Severity**: High
- **Description**: Resolution audit event IDs were not derived deterministically, and crash/replay semantics during hold resolution (e.g. response lost after commit, or concurrent operator attempts) lacked formal transaction boundaries and conflict rules.
- **Remediation Directives**:
  1. Define deterministic `resolution_event_id` from RFC 8785 JCS descriptor:
     ```json
     {
       "version": 1,
       "kind": "review_integrity_resolution_event",
       "event_type": "REVIEW_INTEGRITY_HOLD_RESOLVED",
       "pair_id": "...",
       "task_id": "...",
       "contract_id": "...",
       "attempt_id": "...",
       "hold_id": "...",
       "occurrence_number": N,
       "resolved_by_principal": "<verified_principal>",
       "sanitized_resolution_rationale_fingerprint": "<64_hex_hash>"
     }
     ```
     `resolution_event_id = SHA256(RFC8785_JCS(resolution_event_identity_descriptor))`
  2. Implement atomic resolution transaction:
     a. Authenticate trusted operator principal before `BEGIN`.
     b. Sanitize and canonicalize rationale; fail-closed on key collision or sanitization error.
     c. `BEGIN IMMEDIATE`.
     d. Read hold by `hold_id` and exact lineage.
     e. If hold is `ACTIVE`:
        - Derive deterministic `resolution_event_id`.
        - Append audit event or accept exact semantic replay if `event_id` already exists.
        - CAS update: `SET hold_state = 'RESOLVED', resolved_at_epoch_ms = :now, resolution_audit_event_id = :resolution_event_id, resolved_by_principal = :principal WHERE hold_id = :hold_id AND hold_state = 'ACTIVE'`.
        - Require `affected_rows == 1`.
        - Commit.
     f. If hold is already `RESOLVED`:
        - Read `resolution_audit_event_id` and corresponding audit event.
        - Compare exact sanitized semantic payload, principal, lineage, `hold_id`, `occurrence_number`, and rationale fingerprint.
        - Exact match => idempotent success, do NOT append duplicate audit event.
        - Differing field => fail closed with `ALREADY_RESOLVED_CONFLICT`; do not alter hold and do not append orphan audit.
     g. If CAS returns 0 affected rows (lost race):
        - Re-read hold in transaction.
        - Exact persisted resolution matches caller => idempotent success.
        - Differing resolution => rollback losing caller's transaction completely and fail closed.
     h. Audit append and hold CAS must be in the same transaction.
     i. Commit success with lost response must safely succeed upon exact retry.
     j. Zero unlinked/orphan resolution audit events permitted from lost CAS races.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 16 incorporating R15-001 and R15-002.
2. Remediate `DRAFT-ADR-018` to Revision 16 incorporating R15-001 and R15-002.
3. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 16 incorporating R15-001 and R15-002.
4. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_15`.
5. Maintain `PROPOSAL-P04-002` at Revision 7 (latency semantics unchanged).
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
