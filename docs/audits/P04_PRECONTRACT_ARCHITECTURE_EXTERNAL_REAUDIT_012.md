# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL RE-AUDIT 012

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `8055c35e78e4d9fd5a6e1315334e6bfb0f8dc7c5`
> **Audited Artifacts**:
> - `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 13)
> - `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 13)
> - `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 13)
> - `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 7)
> - `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_011.md`
> - `AGENTS.md`
> - `docs/18_CURRENT_STATE.md`

---

## 1. Executive Summary & Verdict

The External Supervisor audited the remediation deliverables submitted for Pre-Contract Architecture Round 12 at commit `8055c35e78e4d9fd5a6e1315334e6bfb0f8dc7c5`.

### Assessment of Round 12 Findings
- `P04-ARCH-R12-001`: **PARTIALLY_CLOSED** (Revision 11 normative architecture was restored, but residual transitions to `BLOCKED` remained in the conflict and replay matrices).
- `P04-ARCH-R12-002`: **CLOSED_AT_DESIGN_LEVEL** (Linear lease chain with immutable predecessors, token monotonicity, and authoritative process-death proof requirement verified).
- `P04-ARCH-R12-003`: **PARTIALLY_CLOSED** (Hold table was introduced, but single-active-hold unique index dropped multi-diagnostic findings; audit event ID used colon concatenation instead of RFC 8785 JCS; audit foreign keys and non-whitespace principal constraints were missing).
- `P04-ARCH-R12-004`: **CLOSED_AT_DESIGN_LEVEL** (Integer typing constraints `CHECK (typeof(col) = 'integer')` and arithmetic overflow bounds verified).

Four new findings (`P04-ARCH-R13-001` through `P04-ARCH-R13-004`) are formally recorded requiring Revision 14 remediation:
1. `P04-ARCH-R13-001`: Elimination of erroneous TaskState transitions to `BLOCKED` (preserve current TaskState, rollback business transaction, record durable integrity hold in diagnostic transaction, close admission and approval, require authenticated human reconciliation).
2. `P04-ARCH-R13-002`: Foreign key audit references and principal constraints on `review_integrity_holds` (`rejection_audit_event_id UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`, `resolution_audit_event_id UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`, `resolved_by_principal` non-empty/non-whitespace; fail-closed runtime constraint on `VERIFIED_OPERATOR_PRINCIPAL`).
3. `P04-ARCH-R13-003`: Multi-diagnostic hold tracking via `diagnostic_fingerprint` (64-char lowercase hex SHA-256) and partial unique index on `(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE'`; pipeline fail-closed on `EXISTS` any active hold.
4. `P04-ARCH-R13-004`: Canonical event ID derivation via RFC 8785 JCS, replay vs conflict resolution semantics on duplicate key, and formal registration of `REVIEW_INTEGRITY_CONFLICT` and `REVIEW_INTEGRITY_HOLD_RESOLVED`.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_13_REQUIRED`
- `ADR_018 = REVISION_13_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_13_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_13`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 12 Findings (P04-ARCH-R12-001 through R12-004)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R12-001` | Restoration of All Normative Architecture from Revision 11 | **PARTIALLY_CLOSED** | Normative sections restored; residual matrix transitions to `BLOCKED` remained (`P04-ARCH-R13-001`). |
| `P04-ARCH-R12-002` | Linear Lease Chain Design with Immutable Predecessors | **CLOSED_AT_DESIGN_LEVEL** | Permanent `EXPIRED` predecessor rows, `predecessor_lease_id` UNIQUE, token monotonicity, and process-death proof verified. |
| `P04-ARCH-R12-003` | Durable Integrity Hold Mechanism & Audit Event Idempotency | **PARTIALLY_CLOSED** | Hold table created, but single-hold index dropped multi-diagnostics; colon concatenation used; audit FKs missing (`P04-ARCH-R13-002..004`). |
| `P04-ARCH-R12-004` | Integer Typing Constraints & Overflow Guards | **CLOSED_AT_DESIGN_LEVEL** | `typeof() = 'integer'` across all numeric columns and arithmetic overflow checks verified. |

---

## 3. New Findings (P04-ARCH-R13-001 through P04-ARCH-R13-004)

### `P04-ARCH-R13-001`: Elimination of Erroneous TaskState Transitions to BLOCKED
- **Severity**: High
- **Description**: Residual matrices and prose still specified transitioning `TaskState` to `BLOCKED` upon detecting unusual states or database invariant corruption during ReviewBundle compilation. Under canonical state machine rules, integrity failures must not unilaterally mutate TaskState to `BLOCKED`.
- **Remediation Directive**:
  1. Eliminate all transitions to `BLOCKED` from invariant/diagnostic failure matrices and text.
  2. Preserve current TaskState (e.g. `EVIDENCE_READY`, `REVIEWING`, or `RUNNING`).
  3. Roll back the active business transaction.
  4. Create a durable integrity hold in a separate diagnostic transaction.
  5. Close admission and automated review approval for the attempt.
  6. Require authenticated human reconciliation to resolve holds.

### `P04-ARCH-R13-002`: Foreign Key Audit Constraints & Principal Validation
- **Severity**: High
- **Description**: `review_integrity_holds` lacked foreign key enforcement to canonical `audit_events(event_id)`, allowing dangling audit event references, and lacked string trimming checks on `resolved_by_principal`.
- **Remediation Directive**:
  1. Add `rejection_audit_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`.
  2. Add `resolution_audit_event_id TEXT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT`.
  3. Enforce `resolved_by_principal TEXT NULL CHECK (resolved_by_principal IS NULL OR LENGTH(TRIM(resolved_by_principal)) > 0)`.
  4. Enforce that hold creation executes atomically: append rejection audit event first, then insert hold, in a single diagnostic transaction.
  5. Enforce that hold resolution executes atomically: authenticated operator principal, append resolution audit event, and CAS hold state `ACTIVE -> RESOLVED` in a single transaction.
  6. Enforce that runtime resolution fails closed while `VERIFIED_OPERATOR_PRINCIPAL` remains an open dependency.

### `P04-ARCH-R13-003`: Multi-Diagnostic Hold Tracking via Fingerprint
- **Severity**: High
- **Description**: The partial unique index `idx_review_integrity_holds_single_active` enforced at most one active hold per attempt, inadvertently dropping or blocking secondary independent diagnostic findings (e.g. dirty worktree + unverified claim).
- **Remediation Directive**:
  1. Add column `diagnostic_fingerprint TEXT NOT NULL CHECK (LENGTH(diagnostic_fingerprint) = 64 AND NOT (diagnostic_fingerprint GLOB '*[^0-9a-f]*'))`.
  2. Replace single-active-hold index with:
     `CREATE UNIQUE INDEX idx_review_integrity_holds_active_dedup ON review_integrity_holds(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE';`
  3. Permit multiple distinct active holds on the same attempt when reason or fingerprint differs.
  4. Idempotent replay of an identical violation must return the existing hold without error or duplication.
  5. Pipeline gates (Intake, Tx B, Tx C, Approval) fail-closed if `EXISTS` any ACTIVE hold on the attempt.
  6. Human reconciliation resolves each hold independently.

### `P04-ARCH-R13-004`: Canonical Event ID Derivation via RFC 8785 JCS & Replay Semantics
- **Severity**: High
- **Description**: Concatenating audit event ID components with colons (`attempt_id || ":" || reason || ...`) introduced delimiter injection risks. Furthermore, duplicate key handling lacked explicit readback comparison specifications.
- **Remediation Directive**:
  1. Replace colon concatenation with SHA-256 of an RFC 8785 JCS object containing:
     `version`, `event_type`, `pair_id`, `task_id`, `attempt_id`, `contract_id`, `reason`, `sanitized_input_fingerprint`.
  2. Formally define `sanitized_input_fingerprint`: 64-char lowercase hex SHA-256 of canonicalized diagnostic input, strictly omitting secrets, credentials, and host-specific local filesystem paths.
  3. Define duplicate key handling:
     - Read existing event by `event_id`.
     - Compare semantic fields: `event_type`, `actor_id`, `actor_role`, `task_id`, `attempt_id`, `contract_id`, canonical `details_json`.
     - Do NOT compare `sequence_number`, `timestamp`, `prev_event_hash`, or `event_hash`.
     - Exact match = idempotent success.
     - Differing payload/lineage = audit integrity conflict -> fail-closed, emit `REVIEW_INTEGRITY_CONFLICT` with a new distinct event ID, and insert hold.
  4. Formally register proposed audit events `REVIEW_INTEGRITY_CONFLICT` and `REVIEW_INTEGRITY_HOLD_RESOLVED`.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 14 incorporating R13-001..004.
2. Remediate `DRAFT-ADR-018` to Revision 14 incorporating R13-001..004.
3. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 14 incorporating R13-001..004.
4. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_13`.
5. Do NOT modify `PROPOSAL-P04-002` (latency semantics unchanged).
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
