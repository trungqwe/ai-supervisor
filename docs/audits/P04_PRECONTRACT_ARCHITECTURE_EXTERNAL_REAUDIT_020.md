# External Re-Audit Report 020: Phase P04 Pre-Contract Architecture (Revision 21 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `64ff30a9d228303cd3da92e006cbe3e8a7e216b4`
> **Date**: 2026-09-27
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_21`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 020 evaluated the Phase P04 architecture remediation deliverables submitted in commit `64ff30a9d228303cd3da92e006cbe3e8a7e216b4` (Revision 21 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 21)
2. `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 9 - Frozen Baseline)
3. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 21)
4. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 21)
5. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_019.md`
6. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

### Independent Verification Results
- `HEAD = origin/main = 64ff30a9d228303cd3da92e006cbe3e8a7e216b4`.
- Working tree clean.
- Diff from baseline `9dff1f0` contained exactly the 6 whitelisted governance files.
- `git diff --check`: exit 0 (zero whitespace/formatting errors).
- `PROPOSAL-P04-002 Revision 9`: verified byte-exact unchanged.
- Relative Markdown links scan: 175 relative links verified, 0 errors.
- SQLite model probes (A through F): verified PASS for insert-before-parent rejection, stage guard, combined transaction commit, atomic rollback at fault injection points, pre-send binding diagnostic transaction, and governed terminal resolution lifecycle.

### Finding Review Status
- **`P04-ARCH-R20-001`**: **PARTIALLY_CLOSED**. Host-boundary interface `WorkspaceBindingAuthority`, anti-rename handle flags (`FILE_SHARE_READ | FILE_SHARE_WRITE`, omitting `FILE_SHARE_DELETE`), 128-bit `FileIdInfo`, and restart recovery reopening canonical paths were specified. However, the boundary between the immutable data snapshot and the live lease capability was conflated: Store was improperly framed as verifying the live lease, and a single coordinator effect gate holding the live lease across AO `/send` was not formally bound. Follow-up finding `P04-ARCH-R21-001` opened.
- **`P04-ARCH-R20-002`**: **PARTIALLY_CLOSED**. 14 guards in `RecordSendRequested`, 4 attempted reason literals, atomic diagnostic transaction, and governed operator terminalization (`DISPATCHED -> FAILED`, `WORKSPACE_BINDING_INTEGRITY_FAILURE`) were established. However: (1) `REVIEW_INTEGRITY_CONFLICT` conflated audit event ID collisions with workspace binding guard failures, forcing non-applicable fields (`colliding_event_id`) onto binding guards and risking unintended workspace binding mutation by P04D pipeline conflicts; (2) restore and provisioning guards cited invalid columns (`status`) and literals (`PENDING`, `IN_FLIGHT`) not present in the canonical P03 schema. Follow-up findings `P04-ARCH-R21-002` and `P04-ARCH-R21-003` opened.
- **`P04-ARCH-R21-001`**: **OPEN**. Live lease capability binding to effect authority & single coordinator effect gate.
- **`P04-ARCH-R21-002`**: **OPEN**. Variant discrimination for `REVIEW_INTEGRITY_CONFLICT` (`AUDIT_EVENT_ID_COLLISION` vs `WORKSPACE_BINDING_GUARD`).
- **`P04-ARCH-R21-003`**: **OPEN**. Canonical P03 schema alignment for restore and provisioning predicates.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_22_REQUIRED`
- `PROPOSAL_P04_002 = APPROVED_AS_DESIGN_BASELINE`
- `ADR_018 = REVISION_22_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_22_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_21`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`
- `VERDICT = REVISION_22_REQUIRED`

---

## 2. Review of Round 20 Findings

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R20-001` | Trusted Workspace Identity Capability & Handle Lifetime | **PARTIALLY_CLOSED** | Interface, Win32 handle semantics, anti-rename sharing mode, and restart recovery specified. Conflated Store snapshot with live capability; lacks single effect gate holding live lease across `/send`. Follow-up in `P04-ARCH-R21-001`. |
| `P04-ARCH-R20-002` | Pre-Send Failure Semantics & Lifecycle Predicates | **PARTIALLY_CLOSED** | 14 guards, 4 literals, and governed terminal resolution specified. Conflated audit collision with binding failure under `REVIEW_INTEGRITY_CONFLICT`; used non-canonical P03 schema columns for restore/provisioning. Follow-up in `P04-ARCH-R21-002` and `P04-ARCH-R21-003`. |

---

## 3. New Findings: P04-ARCH-R21-001, P04-ARCH-R21-002, and P04-ARCH-R21-003

### `P04-ARCH-R21-001`: Live Lease Binding to Effect Authority & Coordinator Effect Gate
- **Severity**: Critical
- **Description**: Revision 21 correctly defined `WorkspaceBindingSnapshot` and `WorkspaceBindingLease`, but the execution architecture did not strictly enforce that only a live lease instance grants `/send` effect authority. Specifically:
  1. The Store layer cannot prove or enforce live OS handle retention; Store only evaluates database row lineage, triggers, and snapshot attribute equality.
  2. The dispatch sequence lacked a single coordinator/effect gate owning AO `DispatchTaskContract` that holds the same live `WorkspaceBindingLease` in its call frame across the entire pre-send, verification, and dispatch sequence.
  3. Falsification tests erroneously referenced "directory hardlink swap", which is unsupported by NTFS (NTFS does not support hardlinks for directory objects).
- **Mandatory Remediation Directives**:
  1. **Strict Capability vs Data Separation**:
     - `WorkspaceBindingSnapshot`: Immutable data structure used strictly for database lineage, trigger verification, and CAS comparisons.
     - `WorkspaceBindingLease`: Live OS-level capability holding open Win32 handles (`CreateFileW` with `FILE_READ_ATTRIBUTES | FILE_FLAG_BACKUP_SEMANTICS`, omitting `FILE_SHARE_DELETE`).
     - Clarify that Store layer NEVER claims to prove live lease; Store only verifies database rows, lineage, CAS, snapshot attribute equality, and absence of active holds.
  2. **Single Coordinator Effect Gate (`Coordinator.Dispatch`)**:
     - Define a single coordinator/effect gate (aligning with `Coordinator.Dispatch` in `internal/dispatch/coordinator.go`).
     - The effect gate must acquire the live `WorkspaceBindingLease`, hold the SAME lease instance in its call frame, and execute the exact 9-step sequence:
       1. Acquire lease via `WorkspaceBindingAuthority.Acquire(ctx, candidate)`.
       2. Obtain snapshot via `lease.Snapshot()`.
       3. Commit `PrepareBoundDispatch` with snapshot (atomic bound dispatch + workspace binding transaction).
       4. Fresh `GetWorkerStatus` observation from AO.
       5. Revalidate live lease via `lease.Revalidate()`, checking held handles against durable binding read back from DB.
       6. Commit `RecordSendRequested` using pure DB guards (Store checks DB row, lineage, CAS, snapshot equality, zero active holds).
       7. Call AO `/send` (`AO.DispatchTaskContract(ctx, sessionID, message)`) within the same guarded scope while lease is still held open.
       8. Commit confirmed / containment (`RecordSendConfirmed` or `containAmbiguousSend`).
       9. Close lease in `defer` after effect boundary.
  3. **No Alternative Public Send Path**:
     - Zero public API paths accepting only a snapshot can invoke `/send`.
     - Daemon restart recovery must acquire a NEW lease via `WorkspaceBindingAuthority.Acquire`; persisted database rows or prior lease tokens never grant effect authority.
  4. **Planned Falsification Tests**:
     - Fake snapshot used in Store unit test cannot pass effect gate or increment AO call count.
     - Lease `Close()` or `Revalidate()` failure forbids `SEND_REQUESTED` and `/send`.
     - Replace "directory hardlink swap" with valid NTFS directory substitution probes: junction points, directory symlinks, `subst` virtual drives, path rename, and directory alias probes.

### `P04-ARCH-R21-002`: Variant Discrimination for `REVIEW_INTEGRITY_CONFLICT`
- **Severity**: Critical
- **Description**: Revision 21 registered `REVIEW_INTEGRITY_CONFLICT` with a flat schema that forced `colliding_event_id` to be present across all conflicts, even when triggered by pre-send workspace binding guard failures where no colliding audit event exists. Furthermore, it ambiguously linked binding CAS invalidation to the event type rather than the specific failure variant, creating the hazard that P04D pipeline conflicts (such as ReviewBundle synthesis or audit collisions) might improperly invalidate workspace bindings.
- **Mandatory Remediation Directives**:
  1. **Standard Discriminator (`conflict_source`)**:
     - Introduce `conflict_source`: `'AUDIT_EVENT_ID_COLLISION'` vs `'WORKSPACE_BINDING_GUARD'`.
  2. **Variant Details Schema**:
     - **Variant A (`AUDIT_EVENT_ID_COLLISION`)**:
       * Fields: `conflict_source = 'AUDIT_EVENT_ID_COLLISION'`, `colliding_event_id` (string), `attempted_event_type` (string), `conflict_type = 'PAYLOAD_MISMATCH' | 'LINEAGE_MISMATCH'`, `sanitized_input_fingerprint` (64 hex), `diagnostic_fingerprint` (64 hex), `actor_role = 'SUPERVISOR'`.
       * Workspace binding CAS: NONE. Does NOT touch `attempt_workspace_bindings`.
     - **Variant B (`WORKSPACE_BINDING_GUARD`)**:
       * Fields: `conflict_source = 'WORKSPACE_BINDING_GUARD'`, `dispatch_operation_id` (string), `attempted_reason` (one of the 4 literals: `WORKSPACE_BINDING_MISSING`, `WORKSPACE_BINDING_LINEAGE_MISMATCH`, `WORKSPACE_BINDING_NOT_ACTIVE`, `WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`), `conflict_type = 'LINEAGE_MISMATCH'`, `sanitized_input_fingerprint` (64 hex), `diagnostic_fingerprint` (64 hex), `actor_role = 'SUPERVISOR'`.
       * `colliding_event_id` MUST be strictly ABSENT (do NOT generate fake sentinels like `""` or `"none"`).
       * Workspace binding CAS: Conditional `ACTIVE -> INVALIDATED` (`released_at_epoch_ms = now`) if active binding exists.
  3. **Descriptor B & Deterministic Replay**:
     - Descriptor B derivation must include `conflict_source` and the exact variant fields so replay is deterministic.
     - P04D pipeline conflicts must NEVER CAS workspace binding just because they share the event type `REVIEW_INTEGRITY_CONFLICT`.
  4. **Crash/Replay Matrix**:
     - Add explicit rows distinguishing both variants, duplicate replay, semantic mismatch, CAS race, and zero orphan audit events.

### `P04-ARCH-R21-003`: Canonical P03 Schema Alignment for Restore and Provisioning Predicates
- **Severity**: Major
- **Description**: Revision 21 described pre-send guards for restore and provisioning using non-existent column `status` and non-canonical literals `PENDING` and `IN_FLIGHT` (`status IN ('PENDING', 'IN_FLIGHT')`). In the canonical P03 schema:
  - `pair_restore_operations` has column `resolution_state`, and the exact guard is `resolution_state <> 'RESTORE_RESOLVED'`.
  - `pair_provisioning_operations` has column `stage`, and the exact guard is `stage IN ('PROVISION_REQUESTED', 'PROVISION_FAILED')`.
  - Neither table has a `status` column.
- **Mandatory Remediation Directives**:
  1. Update guard 9 to exact SQL:
     `NOT EXISTS (SELECT 1 FROM pair_restore_operations WHERE pair_id = ? AND resolution_state <> 'RESTORE_RESOLVED')`
  2. Update guard 10 to exact SQL:
     `NOT EXISTS (SELECT 1 FROM pair_provisioning_operations WHERE pair_id = ? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED'))`
  3. Eradicate all descriptions using column `status` or literals `PENDING`/`IN_FLIGHT` for these two tables across all documents.
  4. Ensure static scan fails if non-canonical predicates appear.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001`, `DRAFT-ADR-018`, and `PLAN-P04-EVIDENCE-REVIEW` to Revision 22 incorporating all directives for `P04-ARCH-R21-001`, `P04-ARCH-R21-002`, and `P04-ARCH-R21-003`.
2. Keep `PROPOSAL-P04-002 Revision 9` byte-for-byte preserved.
3. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_21`, `P04-ARCH-R20-001 = PARTIALLY_CLOSED`, `P04-ARCH-R20-002 = PARTIALLY_CLOSED`, `P04_ARCH_R21_001 = REMEDIATION_SUBMITTED`, `P04_ARCH_R21_002 = REMEDIATION_SUBMITTED`, and `P04_ARCH_R21_003 = REMEDIATION_SUBMITTED`.
4. Maintain `AUTOMATIC_RESTORE = DISABLED`, `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`, zero Go code, zero contract release.
