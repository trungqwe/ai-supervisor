# P04 Pre-Contract Architecture External Re-Audit 021 Report

**Audit Target**: `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 22), `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 22), `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 22), `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 9)
**Audited Commit**: `9c53ac3ee82a7035d458b1abe15f7a6bc315987d`
**Authority**: External Supervisor
**Status**: EXTERNAL_AUDIT_APPROVED
**Date**: 2026-09-27

---

## 1. Executive Summary & Git Verification Evidence

External Supervisor Re-Audit 021 was conducted directly on commit `9c53ac3ee82a7035d458b1abe15f7a6bc315987d`.

### Git Evidence:
- **Audited Commit SHA**: `9c53ac3ee82a7035d458b1abe15f7a6bc315987d`
- **Branch Synchronization**: `HEAD` = `origin/main` at `9c53ac3ee82a7035d458b1abe15f7a6bc315987d`
- **Working Tree**: Clean (`git status --porcelain` returns empty)
- **Diff Whitelist**: Diff from baseline commit `64ff30a9d228303cd3da92e006cbe3e8a7e216b4` strictly and exclusively modified the 6 whitelisted governance files:
  1. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_020.md` (new, append-only)
  2. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 22)
  3. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 22)
  4. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 22)
  5. `AGENTS.md`
  6. `docs/18_CURRENT_STATE.md`
- **Diff Stat**: `6 files changed, 533 insertions(+), 188 deletions(-)`
- **Format Hygiene**: `git diff baseline..HEAD --check` passed with exit code 0 (zero whitespace or formatting errors).
- **Byte-for-Byte Preservation**: `PROPOSAL-P04-002 Revision 9` remained byte-for-byte identical to baseline (`git diff baseline..HEAD -- docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` returned empty).
- **Zero Scope Contamination**: Zero Go production code (`internal/**/*.go`), zero schema or database migration changes, zero Task Contract modifications, and zero unreleased code alterations.

### Test Suite & Runtime Verification Evidence:
- **Tracked Package Test Suite**:
  ```bash
  go test -race -count=1 ./cmd/... ./internal/... ./test/...
  ```
  **Result**: `PASS` (exit code 0 across all 11 packages: `cmd/supervisor`, `internal/ao`, `internal/audit`, `internal/contract`, `internal/dispatch`, `internal/domain`, `internal/host`, `internal/recovery`, `internal/stop`, `internal/store`, `internal/workflow`, and `test/integration`).
- **Clarification on Root `go test ./...`**:
  The workspace contains an ignored directory `scratch/` with an untracked scratch file (`scratch/drain_current.go`) originating from prior ad-hoc developer exploration. Because `go test ./...` discovers all subdirectories with `.go` files regardless of `.gitignore`, running `go test ./...` fails on that scratch file. This is **not a tracked artifact, not part of the repository, and not a regression**. The tracked production and test codebase under `./cmd/...`, `./internal/...`, and `./test/...` compiles and passes with race detection enabled with exit code 0.
- **SQLite Model Probes**:
  Probes 1 through 6 passed with exit code 0:
  1. Exact P03 restore and provisioning SQL predicates.
  2. Bound Dispatch + Workspace Binding Atomic Transaction commit.
  3. `REVIEW_INTEGRITY_CONFLICT` Variant A (`AUDIT_EVENT_ID_COLLISION`) records hold with zero binding mutation.
  4. `REVIEW_INTEGRITY_CONFLICT` Variant B (`WORKSPACE_BINDING_GUARD`) records hold and CAS invalidates binding; `colliding_event_id` is strictly absent.
  5. Governed terminal resolution: D12 atomic terminal transition (`tasks`: `DISPATCHED -> FAILED`, `task_attempts`: `ended_at = now`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'`, appends `TASK_STATE_TRANSITION` audit), followed by separate hold resolution transaction; operator principal data validation fails closed.
  6. Rollback atomicity on fault injection leaves zero orphan audit events.
- **Link & Consistency Scans**:
  - `check_consistency.py`: All consistency checks passed.
  - `check_links.py`: 175 relative markdown links verified with 0 broken links.

---

## 2. Technical Evaluation & Findings Disposition

### A. Resolution of `P04-ARCH-R20-001` and `P04-ARCH-R21-001`: Live Lease Binding & Coordinator Effect Gate
1. **Capability vs. Data Snapshot Separation**:
   - `WorkspaceBindingSnapshot` is strictly defined as an immutable data structure containing canonical worktree and gitdir paths, Volume Serial Numbers, 128-bit `FileIdInfo` hashes, and pinned AO commit hash, used solely by Store for SQL lineage validation and CAS state checks.
   - `WorkspaceBindingLease` is strictly defined as an active, in-memory OS capability holding open Win32 directory handles (`FILE_READ_ATTRIBUTES | FILE_FLAG_BACKUP_SEMANTICS`, omitting `FILE_SHARE_DELETE`).
   - The Store layer explicitly checks database rows only and **never claims to prove live lease capability**.
2. **Single Coordinator Effect Gate**:
   - `Coordinator.Dispatch` in `internal/dispatch/coordinator.go` exclusively owns the AO `DispatchTaskContract` boundary.
   - A single live `WorkspaceBindingLease` instance is held in the call frame across the entire 9-step sequence:
     1. `authority.Acquire(ctx, candidate)` -> live lease.
     2. Extract immutable `WorkspaceBindingSnapshot`.
     3. Commit `store.PrepareBoundDispatch` with snapshot.
     4. Observe fresh `GetWorkerStatus` from AO client.
     5. Revalidate live lease: `lease.Revalidate()`.
     6. Commit `store.RecordSendRequested` with pure DB guards.
     7. Call AO `/send` while lease remains open.
     8. Commit confirmed or containment transition.
     9. Close lease in `defer`.
   - Zero public paths accept snapshot only to invoke `/send`.
   - Daemon restart recovery requires acquiring a fresh lease; persisted tokens or database rows never grant effect authority.
3. **Eradication of Directory Hardlink References**:
   - All references to "directory hardlink swap" have been eradicated (NTFS does not support directory hardlinks).
   - Valid NTFS substitution probes (junction points, directory symlinks, `subst` virtual drives, directory renames, path-alias probes) are established in the test matrix.

### B. Resolution of `P04-ARCH-R20-002` and `P04-ARCH-R21-002`: Variant Discrimination for `REVIEW_INTEGRITY_CONFLICT`
1. **Standard Discriminator (`conflict_source`)**:
   - `conflict_source` explicitly distinguishes `'AUDIT_EVENT_ID_COLLISION'` (Variant A) from `'WORKSPACE_BINDING_GUARD'` (Variant B).
2. **Variant Details Schema & Transaction Boundaries**:
   - **Variant A (`AUDIT_EVENT_ID_COLLISION`)**:
     * Details schema: `conflict_source = 'AUDIT_EVENT_ID_COLLISION'`, `colliding_event_id`, `attempted_event_type`, `conflict_type`, `sanitized_input_fingerprint`, `diagnostic_fingerprint`, `actor_role = 'SUPERVISOR'`.
     * Transaction boundary: Atomic diagnostic transaction appending audit event and inserting `ACTIVE` hold into `review_integrity_holds`. **Zero workspace binding mutation** (strictly does NOT touch `attempt_workspace_bindings`).
   - **Variant B (`WORKSPACE_BINDING_GUARD`)**:
     * Details schema: `conflict_source = 'WORKSPACE_BINDING_GUARD'`, `dispatch_operation_id`, `attempted_reason` (one of the 4 literals: `WORKSPACE_BINDING_MISSING`, `WORKSPACE_BINDING_LINEAGE_MISMATCH`, `WORKSPACE_BINDING_NOT_ACTIVE`, `WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`), `conflict_type = 'LINEAGE_MISMATCH'`, `sanitized_input_fingerprint`, `diagnostic_fingerprint`, `actor_role = 'SUPERVISOR'`.
     * `colliding_event_id` is **strictly ABSENT** (no fake or sentinel IDs).
     * Transaction boundary: Atomic diagnostic transaction appending audit event, inserting `ACTIVE` hold into `review_integrity_holds`, and conditionally CAS invalidating `attempt_workspace_bindings` (`binding_state = 'INVALIDATED'`, `released_at_epoch_ms = now`).
3. **Pipeline Isolation Invariant**:
   - Subtask P04D pipeline conflicts (such as ReviewBundle synthesis or lease collisions) must never mutate `attempt_workspace_bindings` merely because they share the event type `REVIEW_INTEGRITY_CONFLICT`.
4. **Deterministic Replay via Descriptor B**:
   - Descriptor B incorporates `conflict_source` and the exact variant fields, ensuring deterministic, collision-free derivation of `rejection_event_id`.

### C. Resolution of `P04-ARCH-R21-003`: Canonical P03 Schema Alignment
1. Guard 9 in `RecordSendRequested` is aligned to exact canonical SQL:
   ```sql
   NOT EXISTS (
       SELECT 1 FROM pair_restore_operations
       WHERE pair_id = ? AND resolution_state <> 'RESTORE_RESOLVED'
   )
   ```
2. Guard 10 in `RecordSendRequested` is aligned to exact canonical SQL:
   ```sql
   NOT EXISTS (
       SELECT 1 FROM pair_provisioning_operations
       WHERE pair_id = ? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED')
   )
   ```
3. Non-existent column `status` and non-canonical literals `PENDING` and `IN_FLIGHT` have been completely eradicated from descriptions of these two tables across all documents.

### D. Governed Terminal Resolution
- Pre-send binding rejection strictly preserves `DISPATCHED` task state, `DISPATCH_BOUND` operation stage, and leaves `task_attempts` open pending governed operator resolution; `/send` is forbidden.
- Atomic D12 terminal transition: `tasks` (`DISPATCHED -> FAILED`), `task_attempts` (`ended_at = now`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'`), appends `TASK_STATE_TRANSITION` audit in same transaction.
- Hold resolution is a separate follow-up transaction; crash between transactions leaves task `FAILED` and hold `ACTIVE` (safe, resumable state).
- Operator principal data validation (`LENGTH(TRIM(principal)) > 0`) is distinct from runtime authentication; fails closed while `VERIFIED_OPERATOR_PRINCIPAL` remains an open dependency.

---

## 3. Findings Disposition Summary

| Finding ID | Title | Severity | Disposition in Re-Audit 021 |
| :--- | :--- | :--- | :--- |
| **P04-ARCH-R20-001** | Trusted Workspace Identity Capability & Handle Lifetime | CRITICAL | **CLOSED_AT_DESIGN_LEVEL** |
| **P04-ARCH-R20-002** | REVIEW_INTEGRITY_CONFLICT Schema & Mutation Ambiguity | HIGH | **CLOSED_AT_DESIGN_LEVEL** |
| **P04-ARCH-R21-001** | Live Lease Binding to Effect Authority & Coordinator Effect Gate | CRITICAL | **CLOSED_AT_DESIGN_LEVEL** |
| **P04-ARCH-R21-002** | Variant Discrimination for REVIEW_INTEGRITY_CONFLICT | CRITICAL | **CLOSED_AT_DESIGN_LEVEL** |
| **P04-ARCH-R21-003** | Canonical P03 Schema Alignment for Restore and Provisioning Predicates | MAJOR | **CLOSED_AT_DESIGN_LEVEL** |

### Tracked Design Blockers Disposition:

| Blocker ID | Status | Architectural Resolution |
| :--- | :--- | :--- |
| `DESIGN_BLOCKER_P04_WORKTREE_BINDING` | **CLOSED_AT_DESIGN_LEVEL** | Bound Dispatch + Workspace Binding atomic seam extension in Schema v6; live lease effect gate; NTFS substitution probes. |
| `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY` | **CLOSED_AT_DESIGN_LEVEL** | Pure in-memory Git collector (Subtask P04B); clean worktree verification with `-z` null-byte parsing; diagnostic tx on dirty worktree. |
| `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION` | **CLOSED_AT_DESIGN_LEVEL** | Windows AppContainer sandboxing (`CreateProcessW` with explicit handle lists and Job Objects with `KILL_ON_JOB_CLOSE`). |
| `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION` | **CLOSED_AT_DESIGN_LEVEL** | Schema v6 owned by P04A; Schema v9 owned by P04D; composite foreign keys; RFC 8785 JCS canonicalization. |
| `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY` | **CLOSED_AT_DESIGN_LEVEL** | Model 1 persistence ownership; P04B/C pure in-memory; single-owner transactions; atomic CAS compilation. |
| `DESIGN_BLOCKER_P04_INERT_AO_HARNESS` | **CLOSED_AT_DESIGN_LEVEL** | Inert fake AO adapter harness; synthetic session endpoints; live AO integration in unverified evidence track; `AUTOMATIC_RESTORE = DISABLED`. |

*(Note: Win32 API handles, Windows AppContainer execution, Stage B runtime catalog, Live AO integration, and verified operator principal remain tracked under their respective runtime acceptance / external dependency tracks).*

---

## 4. Formal Verdict & Approved Next Actions

```text
P04_PRECONTRACT_ARCHITECTURE = EXTERNAL_AUDIT_APPROVED

PROPOSAL_P04_001 = EXTERNAL_APPROVED
PROPOSAL_P04_002 = APPROVED_AS_DESIGN_BASELINE

ADR_018 = EXTERNAL_APPROVED
ADR_018_ACCEPTANCE = GRANTED

PLAN_P04_EVIDENCE_REVIEW = EXTERNAL_APPROVED

P04_TASK_CONTRACT = NOT_RELEASED
P04_CODE = HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT
P05_CODE = NOT_AUTHORIZED

AUTOMATIC_RESTORE = DISABLED
VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY
LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK
STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION

ACTIVE_GATE = P04_CANONICAL_RECONCILIATION_PLANNING
```

### Approved Next Action:
1. Promote `DRAFT-ADR-018` to accepted canonical ADR: `docs/adr/ADR-018-evidence-review-and-verification-isolation.md` (`ACCEPTED / EXTERNAL_APPROVED`). Mark draft as superseded while preserving revision history.
2. Update statuses of `PROPOSAL-P04-001` (`EXTERNAL_APPROVED`) and `PLAN-P04-EVIDENCE-REVIEW` (`EXTERNAL_APPROVED_FOR_CANONICAL_RECONCILIATION`). Keep `PROPOSAL-P04-002 Revision 9` byte-for-byte unchanged.
3. Formulate the single-pass canonical documentation reconciliation plan in `docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md` pursuant to approved ADR-018 and approved proposals.
4. Advance `ACTIVE_GATE` to `P04_CANONICAL_RECONCILIATION_PLANNING`. Zero production or test code writing for Phase P04 or P05, and zero canonical specification mutations are authorized until reconciliation plan is approved.
