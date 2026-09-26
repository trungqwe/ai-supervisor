# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL RE-AUDIT 004

- **Audit Record ID:** `P04-REAUDIT-004`
- **Audited Baseline Commit Reference:** `06720c0a9eff4f83c8d1bca7b42a98bdcad4d278`
- **Target Deliverables:**
  - [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 5)
  - [`docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md) (Revision 5)
  - [`docs/plans/PLAN-P04-EVIDENCE-REVIEW.md`](../plans/PLAN-P04-EVIDENCE-REVIEW.md) (Revision 5)
  - [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 3)
  - [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (Model A Draft Lineage, Rev 1 draft)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_003.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_003.md)
- **Date:** 2026-09-26
- **Status:** `FORMALLY_RECORDED`
- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_5`
- **Authority:** External Supervisor Governance Directives ([`AGENTS.md`](../../AGENTS.md), [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md))

---

## 1. Executive Summary & Verdict

The External Supervisor conducted Re-Audit 004 on the Phase P04 pre-contract architecture deliverables at baseline commit `06720c0a9eff4f83c8d1bca7b42a98bdcad4d278`.

### Formal Governance Verdicts
- `PROPOSAL_P04_001 = REVISION_5_REQUIRED`
- `ADR_018 = REVISION_5_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_5_REQUIRED`
- `P04_WORKTREE_BINDING_PROOF_CONTRACT = BLOCKED_NOT_RELEASEABLE`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_5`

---

## 2. Evaluation of Prior Findings

- **`P04-ARCH-R4-001`**: `CLOSED` (Citations rectified to canonical `Untrivial-ai/agent-orchestrator` at pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`; zero unauthorized permalinks; Erratum 003 formally recorded).
- **`P04-ARCH-R4-002`**: `SUBSTANTIVELY_CLOSED_AT_WIRE_DESIGN_LEVEL` (Restore wire sequence corrected to `POST /sessions/{id}/kill` -> bounded poll `GET /sessions/{id}` for `isTerminated=true` -> restart -> `POST /sessions/{id}/restore` HTTP 200).
- **`P04-ARCH-R4-003`**: `SUBSTANTIVELY_CLOSED_AT_DESIGN_LEVEL` (Component-boundary verification using OS handles, `FILE_ID_INFO`, volume serial, and ownership marker nonce designed).
- **`P04-ARCH-R4-004`**: `PARTIALLY_CLOSED` (Monotonic lease fencing preserved lease row on release, but atomic acquisition transaction predicates, expiration boundary semantics, and concurrency matrix require completion).
- **`P04-ARCH-R4-005`**: `PARTIALLY_CLOSED` (Immutable `review_artifacts` with `ON DELETE RESTRICT` designed, but physical GC claims must be removed and deferred, and CHECK constraints must be completed).
- **`P04-ARCH-R4-006`**: `SUBSTANTIVELY_CLOSED` (Fabricated inert harness literal completely removed; worker profile aligned to `antigravity-standard`; contract lineage adopted Model A).
- **`P04-ARCH-R3-001..006`**: `PARTIALLY_CLOSED` (Superceded or partially addressed by R4/R5 findings).
- **`P04-ARCH-R3-007`**: `CLOSED` (Governance state synchronization maintained).
- **`P04-ARCH-R2-001..004`**: `PARTIALLY_CLOSED` (Awaiting completed architecture and contract resolution).

---

## 3. New Findings (Round 5)

### P04-ARCH-R5-001 — Disposable Proof Runtime Inexecutable (Harness Deficiency)
- **Classification:** CRITICAL BLOCKER / DESIGN DEFECT
- **Observation:**
  1. Pinned upstream Agent Orchestrator (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`, `backend/internal/domain/harness.go`) has no user-selectable inert harness. Supported harnesses in `AllHarnesses` are live agent CLIs (`agy`, `codex`, `claude-code`, etc.). The constant `HarnessFake="fake"` exists only in internal tests and is not in `AllHarnesses`.
  2. `POST /api/v1/sessions` requires a non-empty `harness` parameter. The Supervisor's `CreateWorkerSession` also rejects empty harness.
  3. The proposed payload `{"projectId", "kind"}` omitting `harness` is rejected by upstream AO with HTTP 400.
  4. Setting `worker_profile="antigravity-standard"` while asserting absolute "zero LLM token generation" is contradictory if the test attempts to run live agent sessions.
- **Architectural Directive:**
  - Do NOT use disposable AO worker-session runtime proof as a pre-contract release gate.
  - Retain Track 1 (Static Pinned-Source Inspection) as design evidence.
  - Transition runtime physical worktree binding into fail-closed **Stage B validation** executed on authorized sessions during normal operation.
  - Do not create, kill, restore, or prompt disposable AO sessions merely to prove binding.
  - `LIVE_AO_INTEGRATION` remains `UNVERIFIED_EVIDENCE_TRACK`.
  - Mark `DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF` as `BLOCKED_NOT_RELEASEABLE` (status remains `NOT_RELEASED`). Do not create a candidate.

### P04-ARCH-R5-002 — Authoritative Durable Attempt Worktree Binding Schema
- **Classification:** ARCHITECTURE & DATA INTEGRITY DEFECT
- **Observation:** Proposal Rev 5 and Plan Rev 5 referenced attempt worktree binding, but lacked an authoritative table schema, ownership definition, and atomic registration rules.
- **Architectural Directive:**
  - Introduce authoritative table `attempt_workspace_bindings` in Schema v6 (owned by Subtask P04A intake):
    * `attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT`
    * `task_id TEXT NOT NULL`
    * `contract_id TEXT NOT NULL`
    * `session_id TEXT NOT NULL`
    * `terminal_generation INTEGER NOT NULL`
    * `canonical_worktree_path TEXT NOT NULL`
    * `managed_root_final_path TEXT NOT NULL`
    * `volume_serial_number TEXT NOT NULL`
    * `file_id TEXT NOT NULL` (128-bit FileId)
    * `pinned_ao_commit TEXT NOT NULL`
    * `bound_at TEXT NOT NULL`
    * SQLite triggers rejecting `UPDATE` and `DELETE` (immutable append-only).
  - Binding must be registered in a single transaction verifying exact task, current_attempt, contract, session, and generation.
  - `worker_sessions.worktree_path` is strictly an ephemeral session observation/cache, NOT attempt audit authority.
  - P04 evidence lease acquisition must fail closed if a matching, valid `attempt_workspace_bindings` row is missing.

### P04-ARCH-R5-003 — Atomic Fencing Lease Acquisition Protocol
- **Classification:** CONCURRENCY & FENCING DEFECT
- **Observation:** `task_verification_leases` lacked rigorous specification of state invariants, atomic acquisition within a transaction, strict expiration equality semantics, and a complete fault matrix.
- **Architectural Directive:**
  - State `ACTIVE` strictly requires non-null `worker_id`, `attempt_id`, `contract_id`, `acquired_at_epoch_ms`, and `expires_at_epoch_ms`.
  - State `RELEASED` preserves owner information for auditability; no active lease may have a null owner.
  - Validity condition: `expires_at_epoch_ms > now_ms`. Equality (`expires_at_epoch_ms == now_ms`) is expired.
  - Acquisition must be executed within `BEGIN IMMEDIATE` with exact lineage predicates: `tasks.state = 'REPORT_READY'`, matching `current_attempt`, `task_attempts` exists, active contract revision, valid immutable `attempt_workspace_bindings`, and prior lease `RELEASED` or expired.
  - Reclaim always increments `fencing_token`; never delete rows.
  - Final CAS update on `tasks.state` verifies active lease, matching token, matching binding, unexpired epoch, and asserts `rows_affected == 1`, rolling back evidence on failure.
  - Document complete crash and concurrency fault matrix.

### P04-ARCH-R5-004 — Artifact Store Scope Reduction & Crash Consistency
- **Classification:** SCOPE & SYSTEM INTEGRITY DEFECT
- **Observation:** Proposal Rev 5 claimed active coordinated garbage collection (GC) for artifacts, but no durable lease or quarantine schema exists.
- **Architectural Directive:**
  - P04 initial implementation is strictly **append-only** with **zero physical GC**.
  - Background physical deletion of staging or canonical files is DEFERRED to a future proposal/ADR.
  - Crashes prior to SQLite commit may leave orphan physical files in `<SUPERVISOR_STATE_ROOT>/artifacts/<captured_sha256>`, but create zero authoritative review rows.
  - Add SQLite CHECK constraints on `review_artifacts`: SHA-256 length/format (64 hex characters), non-negative byte counts, `captured_bytes <= original_bytes`, consistent `is_truncated`.
  - Formalize protocol: `.staging/<attempt_id>-<uuid>.tmp` -> `fsync` file -> `fsync` parent dir -> atomic rename -> final SQLite CAS transaction.

### P04-ARCH-R5-005 — Requirement Traceability & NFR-008 Clarification Drift
- **Classification:** REQUIREMENT GOVERNANCE DEFECT
- **Observation:**
  1. Draft proof contract cited irrelevant requirements (`FR-004`, `NFR-003`, `SEC-001`).
  2. Proposal Rev 5 introduced an unauthorized "clarification" altering `NFR-008` to "under 5 seconds for diffs < 1000 lines".
- **Architectural Directive:**
  - Update draft contract traceability to: `FR-008`, `SEC-002`, `SEC-003`, `NFR-005`, `NFR-007`.
  - Quote canonical `NFR-008` accurately: *"The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*
  - Eliminate all occurrences of "5 seconds", "5s", "1000 lines", "1,000 lines".
  - Benchmark for ReviewBundle compilation pipeline must be strictly separated from external verification subprocess execution time.

---

## 4. Remediation Directives Summary

1. Update `PROPOSAL-P04-001` to Revision 6.
2. Update `DRAFT-ADR-018` to Revision 6.
3. Update `PLAN-P04-EVIDENCE-REVIEW` to Revision 6.
4. Update `PLAN-P04-WORKTREE-BINDING-PROOF` to Revision 4.
5. Update `DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF` to `BLOCKED_NOT_RELEASEABLE`.
6. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md`.
7. Zero implementation code (P04_CODE = HELD, P05_CODE = NOT_AUTHORIZED).
