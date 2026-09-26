# PLAN-P04-EVIDENCE-REVIEW: Phase P04 Evidence & Review Engine Execution and Decomposition Plan

- **Plan ID:** `PLAN-P04-EVIDENCE-REVIEW`
- **Revision:** 3 (Remediation of External Re-Audit 001)
- **Status:** `PLANNING_PENDING_EXTERNAL_AUDIT`
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Author:** AI Engineering Supervisor Control Plane Team
- **Governance Authority:** [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), [`AGENTS.md`](../../AGENTS.md)
- **Audit References:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (`REVISION_2_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md) (`FORMALLY_RECORDED`)
- **Related Documents:**
  - [`PROPOSAL-P04-001`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 3)
  - [`DRAFT-ADR-018`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md) (Revision 3)
  - [`PLAN-P04-WORKTREE-BINDING-PROOF`](PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 1)
  - [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`)
- **Source Provenance:** [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md), [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md), [`docs/22_MODULE_PROVENANCE.md`](../22_MODULE_PROVENANCE.md)

---

## 1. Executive Summary & Governance Baseline

This plan governs the execution and decomposition of Phase P04 Evidence & Review Engine implementation, following External Re-Audit 001 which issued verdict `REVISION_2_REQUIRED` with four new findings (`P04-ARCH-R2-001..004`).

**Governance State at Revision 3:**
- `PHASE_P03 = COMPLETE` (Approved by External Supervisor)
- `P03_EXIT_GATE = EXTERNAL_AUDIT_APPROVED`
- `PROPOSAL_P04_001 = REVISION_2_REQUIRED` (Remediated to Revision 3)
- `ADR_018 = REVISION_2_REQUIRED` (Remediated to Revision 3)
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_2_REQUIRED` (Remediated to Revision 3)
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_2`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`
- `LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`
- `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`
- `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`

**Five Open Design Blockers:**
1. `DESIGN_BLOCKER_P04_WORKTREE_BINDING = OPEN_PENDING_BOUNDED_PROOF`
2. `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY = OPEN`
3. `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`
4. `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`
5. `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY = OPEN`

**Pre-Requisite (Mandatory Before P04A Contract):**
- Execute [`PLAN-P04-WORKTREE-BINDING-PROOF`](PLAN-P04-WORKTREE-BINDING-PROOF.md) (3-track empirical proof).
- Obtain External Supervisor approval of proof conclusion.
- Only upon proof approval may P04A implementation Task Contract be released.

---

## 2. Phased Scope Decomposition Plan

### Phase P04 Subtask Sequence

```
PRE-REQUISITE: TASK-P04-WORKTREE-BINDING-PROOF (NOT_RELEASED, Bounded Proof Only)
        |
        v (Proof Approved)
P04A: WorkerReport Ingestion, Schema Validation & Claim Persistence
        |
        v (P04A Approved)
P04B: Git EvidenceCollector & PolicyEngine (Pure In-Memory)
        |
        v (P04B Approved)
P04C: Isolated VerificationRunner (Pure In-Memory)
        |
        v (P04C Approved)
P04D: Durable Artifact Store & Atomic Pipeline Orchestrator
```

---

### 2.1. Pre-Requisite: TASK-P04-WORKTREE-BINDING-PROOF

| Item | Value |
| :--- | :--- |
| **Plan** | [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](PLAN-P04-WORKTREE-BINDING-PROOF.md) |
| **Draft Contract** | [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) |
| **Contract Status** | `NOT_RELEASED` |
| **Evidence Required** | `docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md` with Track 1, Track 2, Track 3 evidence |
| **Blocker Gated** | `DESIGN_BLOCKER_P04_WORKTREE_BINDING` |
| **Output** | Formal determination (A, B, or C); formally approved before P04A |

**Three Execution Tracks:**
- **Track 1** (Static Source Proof): Inspect pinned AO `gitworktree/workspace.go` at `15e9ea9`; document path formula, branch formula, and configuration seam.
- **Track 2** (Isolated Disposable Runtime): Launch AO v0.13.0 with disposable database and managed root; create dual sessions; verify formulas, `git worktree list`, physical file IDs, daemon restart/restore invariance; execute negative probes.
- **Track 3** (Authority Conclusion): Choose Determination A (trusted config seam), B (upstream capability request), or C (fail-closed redesign) based on Tracks 1+2.

**Guardrails:**
- Zero interaction with user databases, user sessions, primary repository.
- Zero autonomous LLM worker prompts.
- All activities confined to `.supervisor/proof/ao_disposable/`.

---

### 2.2. Subtask P04A: WorkerReport Ingestion, Schema Validation & Claim Persistence

**Objective:** Detect `AO_IDLE` session state, retrieve `worker-report.json` via `GetWorkspaceFile`, validate schema, extract `WorkerClaim` entities, bind the session worktree, and persist initial evidence records.

**Schema Version:** v6

**New Tables:**
- `worker_claims`: Persists structured claims from `worker-report.json` (task_id, attempt_id, head_sha, branch, contract_id claim if present).
- `attempt_workspace_bindings`: Records the computed worktree path and branch formula binding for the active TaskAttempt.

**New Code Seams:**
- `internal/evidence/ingestion.go`: Orchestrates detection, retrieval, validation, and claim persistence.
- `internal/evidence/schema_validator.go`: Validates `worker-report.json` against `docs/schemas/worker-report.schema.json`.

**WorkerReport Identity Verification (Mandatory, per Re-Audit 001):**
1. `report.task_id` MUST match `durable task.task_id` from the immutable store record.
2. `report.attempt_id` MUST match `durable attempt.attempt_id` from the active task attempt record.
3. `report.base_sha` MUST match `immutable contract.base_sha` from the released TaskContract.
4. `report.head_sha` is treated as a `WorkerClaim` until `GitEvidenceCollector` (P04B) verifies the actual worktree HEAD.
5. `report.branch` is treated as a `WorkerClaim` until verified against `attempt_workspace_bindings`.
6. WorkerReport schema does NOT contain `contract_id`; lineage traced via `task_id -> attempt_id -> contract.base_sha`.

**Dispatch Integration Seams:**
- Interaction boundary with `internal/ao/client.go` (`GetWorkspaceFile`).
- Interaction boundary with `internal/store/**` (task_attempts, sessions CAS update).
- Interaction boundary with `internal/dispatch/**` (saga state transitions).

**Acceptance Criteria:**
- AC-P04A-01: `GetWorkspaceFile` transport failure yields `FAILED` attempt, zero `worker_claims` rows.
- AC-P04A-02: Schema validation failure yields `FAILED` attempt, zero `worker_claims` rows.
- AC-P04A-03: Identity mismatch on `task_id`, `attempt_id`, or `base_sha` yields `FAILED` attempt.
- AC-P04A-04: Successful ingestion persists exactly one `worker_claims` row and one `attempt_workspace_bindings` row.
- AC-P04A-05: Concurrent ingestion CAS conflict: exactly one winner persists rows; loser executes zero writes.

---

### 2.3. Subtask P04B: Git EvidenceCollector & PolicyEngine (Pure In-Memory)

**Objective:** Collect authoritative, sanitized Git diff evidence from the exact physical worktree; evaluate PolicyEngine findings. **Zero SQLite writes.**

**Schema Version:** v7 (Tables `evidence`, `policy_findings` created; write APIs wired in P04D only)

**Core Components:**
- `internal/evidence/git_collector.go`: Hardened Git CLI executor collecting diff, ancestry, status, and merge check.
- `internal/evidence/policy_engine.go`: Deterministic evaluation of scope compliance, ancestry, merge commits, binary files, and symlinks.

**Git Evidence Hardening (per Decision 2 of DRAFT-ADR-018 Rev 3):**
- Injected: `GIT_TERMINAL_PROMPT=0`, `LC_ALL=C`
- Stripped: `GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_EXTERNAL_DIFF`, `GIT_DIFF_OPTS`
- Fixed per-subcommand argument vectors (no generic string formatting)
- `DESCENDANT_OR_EQUAL_POLICY`: `git merge-base --is-ancestor <base_sha> <head_sha>`
- `MERGE_COMMITS_FORBIDDEN`: `git rev-list --merges <base_sha>..<head_sha>` must be empty
- TOCTOU: Before/after HEAD and status verification; abort on mutation

**PolicyEngine Finding Types:**
- `POLICY_SCOPE_VIOLATION`: File outside `allowed_scope`
- `POLICY_FORBIDDEN_SCOPE_VIOLATION`: File inside `forbidden_scope`
- `POLICY_MERGE_COMMITS_REJECTED`: Merge commit found
- `POLICY_BINARY_FILE_DETECTED`: Binary content in diff
- `POLICY_SYMLINK_MUTATION_DETECTED`: Symlink or mode change
- `POLICY_TOCTOU_MUTATION`: HEAD changed during collection

**Acceptance Criteria:**
- AC-P04B-01: Git executor applies all mandated environment injections and strippings.
- AC-P04B-02: `DESCENDANT_OR_EQUAL_POLICY` correctly passes `base_sha == head_sha` case.
- AC-P04B-03: `MERGE_COMMITS_FORBIDDEN` rejects commits with merge parents.
- AC-P04B-04: TOCTOU mutation detected; evidence collection aborted; zero rows committed.
- AC-P04B-05: All 6 PolicyEngine finding types generated under correct conditions.
- AC-P04B-06: P04B produces zero direct SQLite writes.

---

### 2.4. Subtask P04C: VerificationPolicyCatalog & Isolated VerificationRunner (Pure In-Memory)

**Objective:** Execute profile-constrained verification commands under Windows AppContainer isolation with external attempt sandbox and read-only worktree. Capture bounded stdout/stderr with dual hashing. **Zero SQLite writes.**

**Schema Version:** v8 (Table `actual_test_results` created; write APIs wired in P04D only)

**Core Components:**
- `internal/verification/runner.go`: AppContainer process launcher with sandbox root management.
- `internal/verification/catalog.go`: Policy catalog lookup and request validation.
- `internal/verification/artifact.go`: Bounded stream capture with dual SHA-256 hashing.

**Isolation Architecture (per Decision 3 of DRAFT-ADR-018 Rev 3):**
- **Audited worktree:** `GENERIC_READ` (deny write, delete, append)
- **External attempt sandbox root:** `.supervisor/sandboxes/<attempt_id>/`
  - `tmp/`, `gocache/`, `gopath/`, `scratch/`, `source_snapshot/`
- **Window Station isolation:** `CreateWindowStationW` + `CreateDesktopW` (combined lifecycle)
- **AppContainer SID ACL lifecycle:** Create, grant, deny, probe, cleanup
- **Primary:** AppContainer + Job Object
- **Fallback (requires external WFP firewall):** Restricted Token + Low Integrity Level

**Source Snapshot (for tests requiring in-tree writes):**
- Copied from verified commit `HEAD` (tracked files only; excludes untracked)
- SHA-256 manifest verified before test execution
- Placed in `.supervisor/sandboxes/<attempt_id>/source_snapshot/`
- Destroyed atomically upon completion

**Dual Stream Hashing:**
- `full_stream_sha256`: Computed incrementally over complete output stream as bytes arrive
- `captured_sha256`: Hash of bounded prefix captured and stored to artifact store
- These MUST NOT be conflated; the bounded prefix hash is NOT presented as the full-stream hash

**Normal Test Failure vs Infrastructure Failure:**
- Exit code != 0 → normal test failure → valid verified outcome → recorded in `actual_test_results`
- Timeout, sandbox crash, OOM, tool not found → infrastructure failure → attempt fails → retry

**Acceptance Criteria:**
- AC-P04C-01: AppContainer SID ACL blocks write to audited worktree path.
- AC-P04C-02: All writable cache paths (`TEMP`, `GOCACHE`, `GOPATH`) redirect to sandbox subdirectories.
- AC-P04C-03: Network connect attempt from child process yields `WSAEACCES`.
- AC-P04C-04: `full_stream_sha256` matches expected hash of complete output; `captured_sha256` matches hash of truncated prefix.
- AC-P04C-05: Source snapshot includes only tracked files; excludes untracked; SHA-256 manifest verified.
- AC-P04C-06: Normal test failure (exit code 1) records structured result; task remains `REPORT_READY`.
- AC-P04C-07: P04C produces zero direct SQLite writes.

---

### 2.5. Subtask P04D: Durable Artifact Store & Atomic Pipeline Orchestrator

**Objective:** Persist all in-memory evidence from P04A, P04B, P04C into durable storage via a single atomic SQLite CAS transaction; assemble RFC 8785-hashed ReviewBundle; transition task to `EVIDENCE_READY`.

**Schema Version:** v9

**New Tables:**
- `review_artifacts`: Content-addressed artifact metadata (fields per DRAFT-ADR-018 Rev 3 Decision 6)
- `review_bundles`: Assembled ReviewBundle with RFC 8785 canonical hash

**Core Components:**
- `internal/evidence/artifact_store.go`: Write-rename content-addressed persistence to `.supervisor/artifacts/<sha256>`.
- `internal/evidence/bundle_builder.go`: ReviewBundle assembly with RFC 8785 JCS canonical hashing.
- `internal/evidence/orchestrator.go`: One-winner reservation lease, Model 1 atomic CAS transaction.

**One-Winner Reservation Lease:**
```sql
INSERT INTO task_verification_leases (task_id, attempt_id, lease_owner, expires_at)
VALUES (?, ?, ?, datetime('now', '+300 seconds'));
```
- Losing worker aborts immediately; zero external commands executed.
- Expired leases recovered by crash-recovery monitor.

**Atomic CAS Transaction:**
```sql
BEGIN IMMEDIATE;
UPDATE task_attempts SET status = 'EVIDENCE_READY', updated_at = ?
WHERE attempt_id = ? AND status = 'REPORT_READY';
-- INSERT INTO evidence, policy_findings, actual_test_results, review_artifacts, review_bundles, task_audit_events
COMMIT;
```

**RFC 8785 (JCS) Canonical Bundle Hashing:**
- Algorithm: SHA-256 of RFC 8785 canonical byte representation of ReviewBundle excluding `bundle_hash` field.
- Must be implemented using a validated RFC 8785 library.

**Acceptance Criteria:**
- AC-P04D-01: Successful pipeline execution produces exactly one `review_bundles` row per attempt.
- AC-P04D-02: CAS transaction conflict (concurrent attempt): transaction rolls back; existing rows preserved.
- AC-P04D-03: Partial failure inside atomic transaction: all rows roll back; task remains `REPORT_READY`.
- AC-P04D-04: Content-addressed artifact: writing same content twice produces same path; no duplication.
- AC-P04D-05: RFC 8785 JCS canonical `bundle_hash` validated with known test vectors.
- AC-P04D-06: Normal test failure: ReviewBundle includes structured failure evidence; ChatGPT receives rejection candidates.
- AC-P04D-07: One-winner lease prevents concurrent orchestrators from both executing external commands.

---

## 3. Blast Radius and Scope Guardrails

**Allowed Scope (cumulative across P04A..P04D):**
- `internal/evidence/**`
- `internal/verification/**`
- `migrations/**` (sequential v6, v7, v8, v9 only)
- `docs/schemas/**` (new or updated schemas)
- `test/integration/**`
- `docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md`

**Forbidden Scope:**
- `internal/ao/client.go` (must not be modified in P04; existing `GetWorkspaceFile` reused as-is)
- `internal/recovery/**` (Phase P03 approved code; no modification)
- `internal/dispatch/**` (Phase P03 approved code; no modification except explicit P04A seam integration approved in contract)
- `internal/store/**` (Phase P03 approved code; no modification except explicit P04D CAS migration seam)
- `docs/adr/ADR-016*`, `docs/adr/ADR-017*` (accepted ADRs; immutable)

---

## 4. Worktree Binding Resolution Strategy

Formal governance boundary: Phase P04 strictly reuses AO's published worktree isolation mechanism. No custom worktree manager will be built. No private AO SQLite databases will be queried.

**Static Evidence (Proven):**
From pinned AO `15e9ea9`, `backend/internal/adapters/workspace/gitworktree/workspace.go`:
- Path formula: `filepath.Join(managedRoot, projectID, sessionID)` (line 1887)
- Branch formula: `"ao/" + sessionID` (line 1918)

**Pending Empirical Proof (Required Before P04A):**
- [`PLAN-P04-WORKTREE-BINDING-PROOF.md`](PLAN-P04-WORKTREE-BINDING-PROOF.md)
- [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`)

**Proof Blocker Status:** `DESIGN_BLOCKER_P04_WORKTREE_BINDING = OPEN_PENDING_BOUNDED_PROOF`

---

## 5. Verification Isolation Architecture & Platform Strategy

**Platform**: Windows only (Phase P04 runtime platform is Windows as established in Phase P02/P03 ADRs).

**Primary Isolation Stack:**
1. Windows AppContainer (creates process-level capability isolation, removes administrator tokens)
2. Job Object (CPU, memory, process count limits; `TerminateJobObject` forceful teardown)
3. Dedicated Window Station + Desktop (`CreateWindowStationW` + `CreateDesktopW`)
4. External Attempt Sandbox Root (`.supervisor/sandboxes/<attempt_id>/`)

**Secondary Fallback (explicit external WFP firewall required):**
- Restricted Token + Low Integrity Level
- Cannot be used in production without verified external WFP network deny rule

**Audited Worktree ACL:** `GENERIC_READ` (deny write, delete, append) — strictly enforced before process launch.

**Source Snapshot (for in-tree write tests):**
- Created from verified `HEAD` (tracked files, SHA-256 manifest)
- Placed in `.supervisor/sandboxes/<attempt_id>/source_snapshot/`
- Destroyed atomically upon completion

**Blocker Status:** `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`

---

## 6. Reviewable Evidence & Schema Reconciliation Strategy (S1..S10)

| Item | Description | Owner | Target Subtask |
| :--- | :--- | :--- | :--- |
| S1 | Replace `details` with `test_results` across all schemas | Schema | P04A |
| S2 | Add `review_artifacts` table to Schema v9 | Schema | P04D |
| S3 | Reference `patch_artifact_id` in Git evidence | Schema | P04D |
| S4 | Add `stdout_artifact_id`, `stderr_artifact_id` | Schema | P04D |
| S5 | Formalize RFC 8785 (JCS) as canonical hashing algorithm | Architecture | P04D |
| S6 | Add truncation flags: `is_patch_truncated`, `is_stdout_truncated`, `is_stderr_truncated` | Schema | P04C |
| S7 | Define opaque artifact retrieval service API for P05 | API Design | P04D |
| S8 | Include binary file manifest in ReviewBundle | Schema | P04B |
| S9 | Synchronize domain struct definitions `internal/domain` | Domain | P04A |
| S10 | Reconcile valid JSON example fixtures with Draft-07 schemas | Schema | P04A |

**Blocker Status:** `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`

---

## 7. Requirement Clarification: NFR-008 Performance SLA

**Conflict:** `NFR-008` states a 3-second cycle from worker task completion to ReviewBundle generation for diffs under 500 lines. Multi-package `go test ./...` under AppContainer isolation routinely requires 5-30+ seconds.

**Status:** `REQUIREMENT_CLARIFICATION_REQUIRED` (canonical requirement NOT modified in this plan revision).

**Proposed Two-Milestone Amendment (for subsequent ADR reconciliation):**
- **Milestone 1 (Verification Execution):** Bounded by TaskContract `timeout_seconds`.
- **Milestone 2 (Durable Synthesis & Transition):** From durable `EVIDENCE_READY` to validated `REVIEWING` ≤ 3.0 seconds measured via in-process monotonic clock (`time.Since`). Process restart between milestones is a diagnostic failure (duration must NOT be inferred from unreliable wall clock differences).

---

## 8. Phase P04 Overall Exit Gate

Upon completion of all subtasks (P04A, P04B, P04C, P04D), Phase P04 exit gate requires:
1. All five design blockers formally resolved and approved by External Supervisor.
2. All schema reconciliation items (S1..S10) implemented and verified.
3. `go test -race -count=1 ./...` exits with code 0 against full test suite.
4. `git diff --check` exits with code 0 (no whitespace errors).
5. Formal audit record submitted for External Supervisor review.
6. External Supervisor exit gate verdict `P04_EXIT_GATE = EXTERNAL_AUDIT_APPROVED` before P05 can open.

---

## 9. Governance Tracking & Directives

- **Active Gate:** `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_2`
- **Next Action:** External Supervisor re-audit of Revision 3 deliverables (PROPOSAL-P04-001, DRAFT-ADR-018, PLAN-P04-EVIDENCE-REVIEW, PLAN-P04-WORKTREE-BINDING-PROOF, DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF, AGENTS.md, 18_CURRENT_STATE.md, Erratum 001, Re-Audit 001 record).
- **Blockers:**
  - `DESIGN_BLOCKER_P04_WORKTREE_BINDING = OPEN_PENDING_BOUNDED_PROOF`
  - `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY = OPEN`
  - `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`
  - `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`
  - `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY = OPEN`
- **Execution Guardrails:**
  - `P04_TASK_CONTRACT = NOT_RELEASED`
  - `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
  - `P05_CODE = NOT_AUTHORIZED`
