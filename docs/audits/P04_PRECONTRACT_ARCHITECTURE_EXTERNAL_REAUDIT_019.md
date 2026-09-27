# External Re-Audit Report 019: Phase P04 Pre-Contract Architecture (Revision 20 Remediation)

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `9dff1f001cf34598ed141dc257142a14f91390f7`
> **Date**: 2026-09-27
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_20`
> **Deciders**: External Supervisor

---

## 1. Executive Summary & Verdict

External Re-Audit 019 evaluated the Phase P04 architecture remediation deliverables submitted in commit `9dff1f001cf34598ed141dc257142a14f91390f7` (Revision 20 deliverables):
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 20)
2. `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 9 - Frozen Baseline)
3. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 20)
4. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 20)
5. `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_018.md`
6. `AGENTS.md` and `docs/18_CURRENT_STATE.md`

### Independent Verification Results
- `HEAD = origin/main = 9dff1f001cf34598ed141dc257142a14f91390f7`.
- Working tree clean.
- Diff from baseline `5e2cff1` contained exactly the 6 whitelisted governance files.
- `git diff --check`: exit 0 (zero whitespace/formatting errors).
- `PROPOSAL-P04-002 Revision 9`: verified byte-exact unchanged.
- Relative Markdown links scan: 175 relative links verified, 0 errors.
- SQLite model probe with `PRAGMA foreign_keys = ON`: verified PASS for insert-before-parent rejection, stage guard, combined transaction commit, and atomic rollback at 6 fault injection points.

### Finding Review Status
- **`P04-ARCH-R19-001`**: **CLOSED_AT_DESIGN_LEVEL**. Verified replacement of independent binding creation transaction with unified Bound Dispatch + Workspace Binding Transaction in Store seam `PrepareBoundDispatch`, pre-transaction OS handle acquisition, total rollback atomicity, stage-aware lineage trigger (`d.stage = 'DISPATCH_BOUND'`), and fail-closed pre-send verification.
- **`PROPOSAL-P04-002 Revision 9`**: **APPROVED_AS_DESIGN_BASELINE**. Frozen byte-for-byte; unchanged.
- **`P04-ARCH-R20-001`**: **OPEN**. Trusted workspace identity capability and handle lifetime undefined.
- **`P04-ARCH-R20-002`**: **OPEN**. Pre-send failure semantics and lifecycle predicates insufficiently implementable.

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_21_REQUIRED`
- `PROPOSAL_P04_002 = APPROVED_AS_DESIGN_BASELINE`
- `ADR_018 = REVISION_21_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_21_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_20`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`
- `VERDICT = REVISION_21_REQUIRED`

---

## 2. Review of Round 19 Finding

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R19-001` | Workspace Binding / Dispatch Atomicity Unresolved | **CLOSED_AT_DESIGN_LEVEL** | Unified Bound Dispatch + Workspace Binding Transaction specified; store seam PrepareBoundDispatch extended; pre-transaction handle check added; stage-aware lineage trigger verified; rollback atomicity proven via SQLite model probe. |

---

## 3. New Findings: P04-ARCH-R20-001 and P04-ARCH-R20-002

### `P04-ARCH-R20-001`: Trusted Workspace Identity Capability & Handle Lifetime Undefined
- **Severity**: Critical
- **Description**: Revision 20 requires `RecordSendRequested` to revalidate physical identity using "managed, held OS handles", but failed to define:
  1. Which component owns and opens handles.
  2. How the Store receives identity proof via API.
  3. How long handles exist.
  4. How path rename/replacement is prevented between revalidation, commit of `SEND_REQUESTED`, and AO `/send`.
  5. How handles are reacquired after daemon restart.
  6. Ensuring the Store layer never covertly accesses Win32/paths outside its transaction contract.
- **Mandatory Remediation Directives**:
  1. **Trusted Interface at Host Boundary**: Define `WorkspaceBindingAuthority.Acquire(ctx, candidate) -> WorkspaceBindingLease` providing:
     - Immutable snapshot: canonical worktree path, worktree `VolumeSerialNumber`/`FileIdInfo` (16-hex/32-hex), linked gitdir path & identity.
     - `Revalidate() error`: compares current path, held handle, and durable binding.
     - `Close() error`: idempotent cleanup.
     - Capability valid strictly within process lifetime; no persistent authority token.
  2. **Windows Handle Semantics**:
     - Open worktree root and linked gitdir via `CreateFileW` with `FILE_READ_ATTRIBUTES` and `FILE_FLAG_BACKUP_SEMANTICS`.
     - Grant sharing mode `FILE_SHARE_READ | FILE_SHARE_WRITE`.
     - Strictly omit `FILE_SHARE_DELETE` to block rename, deletion, or root directory replacement during the effect window.
     - Query canonical final path and 128-bit `FILE_ID_INFO` directly from the handle.
     - Fail closed if filesystem or API cannot supply reliable physical identity.
  3. **Mandatory Lifetime**:
     - Acquire lease before Bound Dispatch + Workspace Binding Transaction.
     - Hold lease across transaction commit.
     - Revalidate same lease immediately before `RecordSendRequested`.
     - Hold lease across commit of `SEND_REQUESTED` and until AO `/send` returns result or enters containment.
     - Close in `defer` after effect boundary.
     - Never preserve handles across daemon restart.
  4. **Daemon Restart Recovery**:
     - Reopen worktree and linked gitdir from durable canonical paths.
     - Compare exact `VolumeSerialNumber` and `FileIdInfo` against `attempt_workspace_bindings`.
     - Pre-send recovery proceeds only after successful reacquire/revalidate.
     - Never treat persisted rows, path strings, or dead handle tokens as physical proof after restart.
  5. **Layering & Separation of Concerns**:
     - Host/coordinator owns `WorkspaceBindingLease` and Win32 calls.
     - `Store.PrepareBoundDispatch` receives immutable `WorkspaceBindingSnapshot`.
     - `Store.RecordSendRequested` receives exact verified snapshot and performs only database lineage/CAS comparisons.
     - Store never directly accesses paths or Win32 handles.
     - No boolean `trusted=true` or caller-provided path is accepted as authority.
  6. **Planned Falsification Tests**:
     - Rename worktree root between validation and `SEND_REQUESTED` rejected by OS or fails guard.
     - Junction/hardlink/path alias does not alter physical identity.
     - Daemon restart invalidates prior capability.
     - Reacquire with matching identity passes; differing `VolumeSerialNumber`/`FileIdInfo` fails.
     - Fake snapshot without live lease grants zero `/send` permission.
     - Lease always closed on success, failure, cancellation, and panic cleanup.

### `P04-ARCH-R20-002`: Pre-Send Failure Semantics & Lifecycle Predicates Insufficiently Implementable
- **Severity**: Critical
- **Description**: Generic phrases such as "stronger lifecycle hold", "emit diagnostic under approved authority", "Fail-Closed Hold", and "Integrity Conflict" lacked concrete predicates, audit event schemas, transaction specifications, and TaskState rules.
- **Mandatory Remediation Directives**:
  1. **Comprehensive Guard Enumeration in `RecordSendRequested`**: Enforce all 14 mandatory guards:
     - `dispatch_operations.stage = 'DISPATCH_BOUND'`
     - `dispatch_operations.resolution_state IS NULL`
     - `task_attempts.recovery_disposition IS NULL`
     - `Task = DISPATCHED`
     - Exact current open attempt, `ended_at IS NULL`
     - Exact task, contract, Pair, session, generation lineage
     - Current WorkerSession identity matches and `quarantine_state = 'CLEAN'`
     - Current TaskAttempt `quarantine_state = 'CLEAN'`
     - Zero unresolved `pair_restore_operations`
     - Zero unresolved `pair_provisioning_operations`
     - Fresh AO observation is `idle` or `waiting_input`, not terminated
     - Exactly one `attempt_workspace_bindings` row with `binding_state = 'ACTIVE'`
     - Live `WorkspaceBindingLease` revalidation passes
     - Zero `ACTIVE` `review_integrity_holds` for the attempt
  2. **Standardized Attempted Reason Literals**:
     - `WORKSPACE_BINDING_MISSING`
     - `WORKSPACE_BINDING_LINEAGE_MISMATCH`
     - `WORKSPACE_BINDING_NOT_ACTIVE`
     - `WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`
  3. **Atomic Diagnostic Transaction on Binding Failure**:
     - Never record `SEND_REQUESTED`; never invoke `/send`.
     - Preserve Task in `DISPATCHED`, dispatch operation in `DISPATCH_BOUND`, attempt remains open.
     - Execute atomic diagnostic transaction:
       a. Append `REVIEW_INTEGRITY_CONFLICT` (`conflict_type = 'LINEAGE_MISMATCH'`, `attempted_reason` set to one of the 4 literals).
       b. Insert `ACTIVE` `review_integrity_holds` (`hold_reason = 'INVARIANT_MISMATCH'`).
       c. If `ACTIVE` binding exists but lineage/identity is invalid, CAS binding to `INVALIDATED` and record `released_at_epoch_ms`.
     - Diagnostic failure still prevents `/send`; no fallback.
  4. **Deterministic Replay Handling**:
     - Employ existing Descriptors A/B and deterministic fingerprints.
     - Idempotent readback on lost response; semantic mismatch fails closed; no duplicate active holds.
  5. **Governed Terminal Resolution & Retry Rules**:
     - Recreating or backfilling binding for `DISPATCH_BOUND` attempt strictly prohibited.
     - Resolving hold does not restore send permission for the same attempt.
     - Verified operator must terminalize attempt via `DISPATCHED -> FAILED` with `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'`.
     - Integrity hold resolved only after terminalization.
     - Retrying requires new `TaskAttempt` and new binding.
     - When `VERIFIED_OPERATOR_PRINCIPAL` is absent, remain fail-closed; fake principals cannot resolve.
  6. **Crash/Replay Matrix Scenarios**:
     - Guard rejection before diagnostic commit, diagnostic transaction rollback, lost diagnostic response, concurrent diagnostic callers, lost CAS on binding invalidation, operator resolution before terminalization rejected, retry creates new attempt.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001`, `DRAFT-ADR-018`, and `PLAN-P04-EVIDENCE-REVIEW` to Revision 21 incorporating all directives for `P04-ARCH-R20-001` and `P04-ARCH-R20-002`.
2. Keep `PROPOSAL-P04-002 Revision 9` byte-for-byte preserved.
3. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_20`, `P04_ARCH_R20_001 = REMEDIATION_SUBMITTED`, and `P04_ARCH_R20_002 = REMEDIATION_SUBMITTED`.
4. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
