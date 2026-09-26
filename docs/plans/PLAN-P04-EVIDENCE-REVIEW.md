# PLAN: Phase P04 — Evidence & Review Engine Execution Plan

- **Plan ID:** `PLAN-P04-EVIDENCE-REVIEW`
- **Revision:** 4 (Remediation of External Re-Audit 002)
- **Status:** `PLANNING_PENDING_EXTERNAL_AUDIT`
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Governing Proposal:** [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 4)
- **Governing Architecture:** [`docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md) (Revision 4)
- **Pre-requisite Proof Plan:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 2)
- **Pre-requisite Draft Contract:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`, Revision 2)
- **Audit References:**
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_002.md) (`REVISION_3_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_002.md) (`FORMALLY_RECORDED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_001.md) (`REVISION_2_REQUIRED`)
  - [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](../audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md) (`REVISION_1_REQUIRED`)
- **Author:** AI Engineering Supervisor Control Plane Team
- **Governance Authority:** [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md)
- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_3`

---

## 1. Overview and Phasing Strategy

Phase P04 implements the Evidence & Review Engine, translating untrusted worker reports and workspace changes into authoritative, inspectable `ReviewBundle` artifacts for supervisor evaluation.

Execution is strictly phased into a pre-requisite proof task followed by four sequential subtasks (P04A..P04D):

```mermaid
flowchart TD
    P04_Proof["Pre-requisite: TASK-P04-WORKTREE-BINDING-PROOF
(PLAN-P04-WORKTREE-BINDING-PROOF Rev 2)"] --> P04A
    P04A["Subtask P04A: Worker Report Parser & Claims Extractor
(Schema v6: claims)"] --> P04B
    P04B["Subtask P04B: Git Evidence Collector & Policy Engine
(Schema v7: evidence, policy_findings - in-memory)"] --> P04C
    P04C["Subtask P04C: Verification Runner & AppContainer Sandbox
(Schema v8: actual_test_results - in-memory)"] --> P04D
    P04D["Subtask P04D: Pipeline Orchestrator, Artifact Store & ReviewBundle
(Schema v9: review_artifacts, review_bundles, task_verification_leases)"]
```

---

## 2. Pre-requisite: Bounded Worktree Authority Proof

- **Task ID:** `TASK-P04-WORKTREE-BINDING-PROOF`
- **Plan Reference:** [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 2)
- **Contract Reference:** [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (`NOT_RELEASED`, Revision 2)
- **Objectives:**
  1. Inspect `backend/internal/adapters/workspace/gitworktree/workspace.go` at pinned AO commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` with verified line citations and GitHub permalinks (rectified in Erratum 002).
  2. Execute isolated disposable runtime proof with dual worker sessions capturing server-generated session IDs (`resp.Session.ID`).
  3. Verify deterministic worktree path preservation across restart and restore.
  4. Formally determine worktree authority (Determination A, B, or C).

---

## 3. Subtask Decomposition (P04A through P04D)

### 3.1. Subtask P04A: Worker Report Parser & Claims Extractor
- **Goal:** Parse `<worktree>/worker-report.json`, enforce schema validation against `docs/schemas/worker-report.schema.json`, verify attempt identity, and extract `WorkerClaim` entities.
- **Components:** `internal/evidence/report_parser.go`, unit test suite.
- **Migration:** Schema v6 (`claims`).

### 3.2. Subtask P04B: Git Evidence Collector & Policy Engine (Model 1: In-Memory)
- **Goal:** Collect independent Git diffs, evaluate `DESCENDANT_OR_EQUAL_POLICY` and `MERGE_COMMITS_FORBIDDEN`, check TaskContract scope compliance (`allowed_scope` vs `forbidden_scope`).
- **Atomicity Invariant:** Pure in-memory execution; performs **zero SQLite writes**. Returns `GitEvidenceResult` and `PolicyEvaluationResult`.
- **Components:** `internal/evidence/git_collector.go`, `policy_engine.go`.
- **Migration:** Schema v7 (`evidence`, `policy_findings`).

### 3.3. Subtask P04C: Verification Runner & Isolation Sandbox (Model 1: In-Memory)
- **Goal:** Execute verification commands specified in the TaskContract using typed profiles in `VerificationPolicyCatalog`.
- **Isolation Architecture:**
  * Absolute attempt sandbox root: `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>\`.
  * Pre-operation containment verification ensuring paths are outside all Git worktrees, primary repository, AO live database, and recovery directory.
  * Windows non-interactive Window Station + Desktop isolation (`CreateWindowStationW` + `CreateDesktopW`).
  * Read-only audited worktree; toolchain write variables (`TEMP`, `TMP`, `GOCACHE`, `GOPATH`) redirected into the external sandbox.
  * Disposable source snapshot for modifying tests.
  * Pinned AO inert harness requirement (`DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN`).
- **Atomicity Invariant:** Pure in-memory execution; performs **zero SQLite writes**. Returns `TestExecutionResult`.
- **Components:** `internal/evidence/verifier.go`, `sandbox_windows.go`.
- **Migration:** Schema v8 (`actual_test_results`).

### 3.4. Subtask P04D: Pipeline Orchestrator, Durable Artifact Store & ReviewBundle
- **Goal:** Orchestrate the end-to-end evidence collection pipeline, persist durable content-addressed artifacts, execute single atomic CAS transaction, and assemble `ReviewBundle`.
- **Pre-Command Reservation Lease:**
  * Acquires lease in `task_verification_leases` (Schema v9) with an incremented `fencing_token`.
  * Losers are denied execution; zero external commands executed.
- **Durable Artifact Store:**
  * Canonical path: `<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`.
  * Dual hashing: `captured_sha256` for file storage; `full_stream_sha256` for SQLite metadata in `review_artifacts`.
  * Crash-consistent protocol: write staging (`.staging/<uuid>`) -> fsync file + parent directory -> atomic rename -> SQLite CAS metadata insert.
  * Crash recovery: startup GC cleans unreferenced staging/orphans after 24h grace period following containment check.
- **Single Final Atomic SQLite CAS Transaction:**
  * Opens single SQLite transaction:
    1. Verifies lease fencing token is valid and unexpired;
    2. Inserts `evidence`, `policy_findings`, `actual_test_results`, `review_artifacts`, `review_bundles`, `task_audit_events`;
    3. Releases `task_verification_leases`;
    4. Executes CAS update on `tasks.state`:
       ```sql
       UPDATE tasks
       SET state = 'EVIDENCE_READY',
           updated_at = ?
       WHERE task_id = ?
         AND state = 'REPORT_READY'
         AND current_attempt = ?;
       ```
  * Note: `task_attempts` has no status column; the state transition belongs strictly to `tasks.state`.
- **ReviewBundle Deterministic Hashing:** Formalized under **RFC 8785 (JCS)**.
- **Components:** `internal/evidence/orchestrator.go`, `artifact_store.go`, `bundle_builder.go`.
- **Migration:** Schema v9 (`review_artifacts`, `review_bundles`, `task_verification_leases`).

---

## 4. Crash Consistency Matrix

| Boundary | Staging Path | Canonical Store | SQLite State | Startup / Recovery Action |
|---|---|---|---|---|
| **1. Before write** | None | None | Clean | Attempt failed cleanly |
| **2. During write / pre-fsync** | Incomplete `.staging/<uuid>` | None | Clean | GC removes stale staging file after grace period |
| **3. Post-fsync / pre-rename** | Complete `.staging/<uuid>` | None | Clean | GC removes stale staging file after grace period |
| **4. Post-rename / pre-DB** | None | `<captured_sha256>` | Clean | Orphan file; GC cleans after 24h grace period |
| **5. During DB transaction** | None | `<captured_sha256>` | WAL uncommitted | WAL rolls back; orphan file cleaned by GC |
| **6. DB commit failure** | None | `<captured_sha256>` | Rolled back | Orphan file cleaned by GC; attempt retried |
| **7. DB commit success** | None | `<captured_sha256>` | Committed | Fully consistent and durable |

---

## 5. Requirement Reconciliation Matrix

| Requirement | Description | Subtask | Verification Mechanism |
|---|---|---|---|
| **FR-007** | Worker Report Ingestion | P04A | Strict JSON schema validation against `docs/schemas/worker-report.schema.json` |
| **FR-008** | Git Evidence Collection | P04B | In-memory read-only Git diff collector, `DESCENDANT_OR_EQUAL_POLICY` |
| **FR-009** | Verification Execution | P04C | VerificationPolicyCatalog profiles, AppContainer sandbox, Window Station+Desktop |
| **FR-010** | ReviewBundle Synthesis | P04D | RFC 8785 (JCS) canonical JSON hashing, durable artifact store |
| **SEC-003** | Worker Isolation | P04C | Low-integrity AppContainer, network egress denial, read-only worktree mount |
| **NFR-004** | Audit Trail & Immutability | P04D | Single atomic SQLite CAS transaction, append-only `task_audit_events` |
| **NFR-008** | ReviewBundle Performance SLA | P04D | Under 5s for <1000 lines (Milestone 1 library; Milestone 2 daemon) |
| **REC-007** | Policy Engine Compliance | P04B | Strict evaluation of `allowed_scope` vs `forbidden_scope` |
| **REC-008** | Synthesized Decision Evidence | P04D | Opaque artifact retrieval via Control Plane API for Phase P05 |

---

## 6. Governance Tracking

- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_3`
- **Proof Plan Status:** `PLAN-P04-WORKTREE-BINDING-PROOF` Revision 2 (`PROPOSED`)
- **Proof Contract Status:** `DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF` Revision 2 (`NOT_RELEASED`)
- **Code Authorization:** `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`; `P05_CODE = NOT_AUTHORIZED`.
