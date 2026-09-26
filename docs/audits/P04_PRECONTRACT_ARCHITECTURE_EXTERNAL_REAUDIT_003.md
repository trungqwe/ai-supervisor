# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL RE-AUDIT 003

- **Audit Record ID:** `P04-REAUDIT-003`
- **Audited Baseline Commit Reference:** `f27d9ef0754a5b55d0a389b6f2e83a974af3a2c4`
- **Target Deliverables:**
  - [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 4)
  - [`docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md) (Revision 4)
  - [`docs/plans/PLAN-P04-EVIDENCE-REVIEW.md`](../plans/PLAN-P04-EVIDENCE-REVIEW.md) (Revision 4)
  - [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 2)
  - [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (Revision 2)
- **Date:** 2026-09-26
- **Status:** `FORMALLY_RECORDED`
- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_4`
- **Authority:** External Supervisor Governance Directives ([`AGENTS.md`](../../AGENTS.md), [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md))

---

## 1. Executive Summary & Verdict

The External Supervisor conducted Re-Audit 003 on the Phase P04 pre-contract architecture deliverables at baseline `f27d9ef0754a5b55d0a389b6f2e83a974af3a2c4`.

### Formal Governance Verdicts
- `PROPOSAL_P04_001 = REVISION_4_REQUIRED`
- `ADR_018 = REVISION_4_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_4_REQUIRED`
- `P04_WORKTREE_BINDING_PROOF_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_4`

---

## 2. Evaluation of Prior Findings

- **`P04-ARCH-R3-001`**: `NOT_CLOSED` (Erratum 002 cited wrong user repository and dual line ranges; superseded by Erratum 003).
- **`P04-ARCH-R3-002`**: `PARTIALLY_CLOSED` (Absolute state root defined, but containment verification relied on unsafe string prefix matching; component-boundary handle verification required).
- **`P04-ARCH-R3-003`**: `PARTIALLY_CLOSED` (Typed profiles designed, but fabricated harness `mock-inert` was injected and profile `proof-investigator` was unbacked by approved registry).
- **`P04-ARCH-R3-004`**: `PARTIALLY_CLOSED` (Restore sequence defined, but violated wire protocol by citing `DELETE /sessions`, expecting ambiguous HTTP 200 or 201, and misassigning recreate lines).
- **`P04-ARCH-R3-005`**: `PARTIALLY_CLOSED` (Content-addressed store and dual hashing defined, but `review_artifacts` used `ON DELETE CASCADE` and uncoordinated GC risked active writer races).
- **`P04-ARCH-R3-006`**: `PARTIALLY_CLOSED` (CAS moved to `tasks.state`, but releasing leases via row deletion introduced ABA token reuse and lease acquire lacked complete lineage validation).
- **`P04-ARCH-R3-007`**: `CLOSED` (Governance state synchronization achieved at Round 3).
- **`P04-ARCH-R2-001..004`**: `PARTIALLY_CLOSED` (Unresolved pending empirical proof and approved contract release).

---

## 3. Re-Audit 003 Findings & Remediation Mandates

### 3.1. P04-ARCH-R4-001: UPSTREAM_REPOSITORY_AND_CITATION_PROVENANCE_INCORRECT
- **Observation:** Proposal, ADR, and proof plans cited unauthorized fork `trungqwe/agent-orchestrator` with a dual-line system.
- **Mandate:**
  - Pinned upstream repository is `Untrivial-ai/agent-orchestrator` at commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.
  - Remove all permalinks to `trungqwe/agent-orchestrator`.
  - Use exclusively the single authoritative line ranges:
    * Options: lines 64–73;
    * Workspace.Create: lines 230–262;
    * Workspace.Restore: lines 1045–1112;
    * managedPath: lines 1754–1762;
    * restorePath: lines 1764–1768;
    * defaultSessionBranchName: lines 1783–1785.
  - Issue Erratum 003 to supersede Erratum 002.

### 3.2. P04-ARCH-R4-002: RESTORE_EDGE_PROBE_WIRE_CONTRACT_VIOLATION
- **Observation:** Proof plan used `DELETE /sessions` to create a restorable session, expected ambiguous HTTP 200 or 201, did not specify `GET` polling for terminal observation, and incorrectly cited lines 1113–1140 for Restore recreate logic.
- **Mandate:**
  - Session termination strictly uses `POST /api/v1/sessions/{id}/kill`, requiring HTTP 200 and valid `wireKillSessionResponse`.
  - Must poll `GET /api/v1/sessions/{id}` with bounded timeout until `isTerminated=true` is observed.
  - Restart disposable AO with identical DB and managed root.
  - Call `POST /api/v1/sessions/{id}/restore`, requiring strictly HTTP 200 (never 201), valid `restoreMode`, and matching session IDs.
  - Missing-directory probe must kill Session 2, observe terminal state, delete directory strictly within proof root, call restore, and characterize observed recreation (lines 1069–1111) or error.
  - Calling restore on an active session is strictly forbidden.
  - Stale-worktree probe runs only on a terminated disposable session.

### 3.3. P04-ARCH-R4-003: INSUFFICIENT_CONTAINMENT_CHECK_STRING_PREFIX_VULNERABILITY
- **Observation:** Using string prefix checks (`strings.HasPrefix(target, root)`) is vulnerable to sibling-prefix attacks (`root-attacker` vs `root`).
- **Mandate:**
  - Pre-create: Ensure `SUPERVISOR_STATE_ROOT` exists and is absolute. Open root with OS handle, retrieve final physical path (`GetFinalPathNameByHandleW`), volume serial, and `FILE_ID_INFO`. For non-existent targets, resolve nearest existing ancestor. Use `filepath.Rel` component-boundary checking (reject `rel == ".."` or starting with `..\`). Reject volume mismatch and unexpected reparse points.
  - Post-create: Open target handle, compare physical ancestry, volume, and final path with state root. Revalidate immediately prior to recursive cleanup.
  - Cleanup: Target must differ from state root/proof parent. Marker must contain random `proof_run_id`, expected root identity, and ownership nonce. Mismatch or TOCTOU fails closed (no deletion).
  - Include falsification test matrix (sibling-prefix, junction swap, non-existent child, case alias, different volume, marker replacement).

### 3.4. P04-ARCH-R4-004: LEASE_FENCING_ABA_AND_STALE_EFFECT
- **Observation:** Deleting lease rows on release resets `fencing_token` to 1 on subsequent acquire (ABA vulnerability). Lease acquire lacked lineage and state precondition checks.
- **Mandate:**
  - Durable monotonic fencing: table `task_verification_leases` (Schema v9, owned by P04D) keeps row by `task_id`.
  - `fencing_token` only increases monotonically and never resets.
  - Release is a CAS updating `state = 'RELEASED'` (row is never deleted).
  - Use numeric epoch milliseconds (`expires_at_epoch_ms INTEGER`) for expiry.
  - Lease acquisition requires: `tasks.state = 'REPORT_READY'`, `tasks.current_attempt` matching `task_attempts.attempt_number`, valid attempt/contract lineage, and attempt not superseded.
  - Stale attempt cannot run commands just because old lease expired.
  - Final evidence CAS verifies exact lease owner/token, active status, valid epoch, and updates `tasks.state`.
  - Include complete Fault Matrix covering all 8 failure scenarios.

### 3.5. P04-ARCH-R4-005: ARTIFACT_IMMUTABILITY_AND_GC_RACE
- **Observation:** `review_artifacts` used `ON DELETE CASCADE` (violating audit immutability), and uncoordinated GC risked deleting files during concurrent writer deduplication.
- **Mandate:**
  - Foreign key uses `ON DELETE RESTRICT`. Add SQLite triggers preventing `UPDATE` and `DELETE`.
  - Multiple attempts referencing same `captured_sha256` maintain distinct attempt rows.
  - Retrieval re-hashes stored bytes and compares with `captured_sha256` before returning content.
  - Coordinated GC protocol: acquire exclusive GC lease, mark candidates, DB transaction rechecking references, atomic rename to `.quarantine/`, and physical deletion only after 24h grace period and final check.
  - Writer coordinates with GC protocol to avoid race between existence check and metadata commit.
  - Include regression design for all 7 vectors.

### 3.6. P04-ARCH-R4-006: FABRICATED_HARNESS_AND_PROFILE_AUTHORITY
- **Observation:** `mock-inert` was fabricated and does not exist in pinned AO (`backend/internal/domain/harness.go`). `worker_profile: "proof-investigator"` was unbacked by approved registries.
- **Mandate:**
  - Eliminate all occurrences of `mock-inert`.
  - Keep `DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN`. Do not issue or release proof contract until an inert harness or upstream test seam is established.
  - `worker_profile` must be an approved profile: `antigravity-standard`.
  - Verification profiles in catalog must have fixed host-owned executables; endpoint, state root, and binary are not worker parameters.
  - Draft verification requests for AO probe pass only enum `probe_kind`; zero `harness` parameter.
  - Contract lineage: Model A (pre-release draft lineage with `revision_number = 1`, `CONTRACT-TASK-P04-WORKTREE-BINDING-PROOF-01`, `NOT_RELEASED`, editorial draft iterations documented in markdown).

---

## 4. Execution Directives for Remediation Round 4

1. Submit Revision 5 of Proposal, ADR-018, and Execution Plan; Revision 3 of Bounded Proof Plan; Draft TaskContract under Model A Draft Lineage (`NOT_RELEASED`); Erratum 003; Re-Audit 003; and synchronized `AGENTS.md` and `docs/18_CURRENT_STATE.md`.
2. Exactly 9 files in whitelist. Zero Go code, zero schema migrations, zero live AO runs, zero real model calls.
3. `AUTOMATIC_RESTORE = DISABLED`; `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`; `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`.
