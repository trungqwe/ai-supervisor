# PLAN-P04-CANONICAL-RECONCILIATION-ADR-018: Canonical Specification Reconciliation Scope Plan

> **Authority**: Formulated pursuant to accepted [ADR-018](file:///d:/TU_CODE/ai-supervisor/docs/adr/ADR-018-evidence-review-and-verification-isolation.md) (`EXTERNAL_APPROVED` and `ADR_018_ACCEPTANCE = GRANTED` by External Supervisor Re-Audit 021, commit `9c53ac3ee82a7035d458b1abe15f7a6bc315987d`), approved `PROPOSAL-P04-001` (Revision 22), approved baseline `PROPOSAL-P04-002` (Revision 9), and `docs/24_CHANGE_GOVERNANCE.md` (Decision Hierarchy Level 2: Approved ADR takes precedence over Level 3: Canonical Architecture and Level 4: Requirement Specifications).
> **Status**: `PLANNING_PENDING_EXTERNAL_AUDIT` (Reconciliation Scope Plan)
> **Active Gate**: `P04_CANONICAL_RECONCILIATION_PLANNING`
> **Mode**: DOCUMENTATION RECONCILIATION PLANNING ONLY (Zero canonical specification edits until this plan is approved; zero Go production code, zero migrations, zero Task Contract release).
> **Date**: 2026-09-27

---

## 1. Governance Context, Baseline Distinction & Purpose

### 1.1. Context & Purpose
Phase P04 pre-contract architecture remediation is formally **EXTERNAL_AUDIT_APPROVED** by External Supervisor Re-Audit 021 (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_021.md`, commit `9c53ac3ee82a7035d458b1abe15f7a6bc315987d`). ADR-018 is formally **ACCEPTED** (`ADR_018_ACCEPTANCE = GRANTED`). All six tracked architecture design blockers (`DESIGN_BLOCKER_P04_*`) are closed at design level.

Under `docs/24_CHANGE_GOVERNANCE.md`, an accepted ADR (Level 2) supersedes canonical architecture (Level 3) and requirement specifications (Level 4). Before any Phase P04 implementation contract can be released (starting with Subtask P04A), the repository's canonical documentation corpus must undergo a strict, single-pass canonical reconciliation.

This plan defines the **precise inventory, itemized mapping, exact literals/schemas/APIs to synchronize, subtask ownership boundaries, and acceptance criteria** for reconciling canonical specifications with accepted ADR-018, approved PROPOSAL-P04-001 (Revision 22), and approved baseline PROPOSAL-P04-002 (Revision 9).

### 1.2. Baseline Partitioning Discipline

| Dimension | Baseline A: Phase P03 Locked Baselines | Baseline B: Phase P04 ADR-018 Reconciliation (This Plan) |
| :--- | :--- | :--- |
| **Originating Authority** | `PROPOSAL-P03-001`, `ADR-016 Accepted`, `ADR-017 Accepted` | `PROPOSAL-P04-001 Rev 22`, `PROPOSAL-P04-002 Rev 9`, **ADR-018 Accepted** |
| **Audit Status** | `P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED` (Rev 3); `P03_ADR_016_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED` | **`P04_CANONICAL_RECONCILIATION_PLANNING` (PLANNING PENDING EXTERNAL AUDIT)** |
| **Reconciled Scope** | P03 AO integration, durable dispatch saga, Pair session decoupling, Model A attempt snapshot, double-gated quarantine, stop lifecycle, and host quiescence. | P04 Evidence & Review Engine: Model 1 persistence ownership, Schema v6/v9, live lease capability vs snapshot separation, single coordinator effect gate, AppContainer verification runner, CAS store, JCS hashing, latency diagnostic intervals, Descriptor A/B/C, and REVIEW_INTEGRITY_CONFLICT variant discrimination. |
| **Status Invariant** | Permanently locked and approved; zero retroactive modification. | Planning only; zero specification edits permitted until plan audit approval. |

---

## 2. ADR-018 Decisions Traceability & Specification Mapping

The 7 technical decisions and governing invariants of accepted ADR-018 map to canonical documents as follows:

| Decision ID | Summary of Accepted Architecture | Primary Canonical File & Section | Secondary / Cross-Referenced Files |
| :--- | :--- | :--- | :--- |
| **D1** | Workspace Binding Authority, Live Lease vs Snapshot, Coordinator Effect Gate, Exact P03 SQL Guards, Governed Terminalization (`WORKSPACE_BINDING_INTEGRITY_FAILURE`), NTFS Substitution Probes | `docs/04_ARCHITECTURE.md` §7.1, §7.2; `docs/05_DOMAIN_MODEL.md` §1, §2; `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/12_UPSTREAM_INTEGRATION.md` §1; `docs/14_FAILURE_RECOVERY.md` §1; `docs/22_MODULE_PROVENANCE.md` §1 |
| **D2** | Git Evidence Authority, Pure In-Memory Collector (P04B), Clean-Worktree Intake Verification (`git status -z`, `git diff-index --quiet`), 10-Command Allowlist, Dirty Intake Diagnostic Tx (`DIRTY_WORKTREE_DETECTED`, preserving `RUNNING`) | `docs/04_ARCHITECTURE.md` §7.2; `docs/02_REQUIREMENTS.md` (FR-008); `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | `docs/14_FAILURE_RECOVERY.md` §1; `docs/22_MODULE_PROVENANCE.md` §1 |
| **D3** | Windows AppContainer & Job Object Execution Isolation, Explicit Handle Lists (`PROC_THREAD_ATTRIBUTE_HANDLE_LIST`), `KILL_ON_JOB_CLOSE`, Authoritative Process-Death Proof, Pure In-Memory Verification Runner (P04C) | `docs/04_ARCHITECTURE.md` §7.3; `docs/02_REQUIREMENTS.md` (NFR-008); `docs/12_UPSTREAM_INTEGRATION.md` §2 | `docs/14_FAILURE_RECOVERY.md` §1; `docs/22_MODULE_PROVENANCE.md` §1 |
| **D4** | Inert Fake AO Adapter Harness for Verification Pipeline Testing, Synthetic Session Endpoints, Live AO Integration in Unverified Track, Fail-Closed `AUTOMATIC_RESTORE = DISABLED` | `docs/12_UPSTREAM_INTEGRATION.md` §1; `docs/phases/P04_EVIDENCE_REVIEW.md` §2 | `docs/04_ARCHITECTURE.md` §7.4; `docs/sources/SOURCE_REGISTRY.md` |
| **D5** | Content-Addressed Store, Dual Head SHAs (`actual_head_sha` vs `reported_head_sha`), Pre-Commit Assembly Boundary, ReviewBundle Latency Semantics from PROPOSAL-P04-002 Rev 9 (2x2 Provenance Matrix, Monotonic Post-Commit Telemetry, `UNVERIFIED` Status) | `docs/04_ARCHITECTURE.md` §7.4; `docs/02_REQUIREMENTS.md` (NFR-008); `docs/10_REVIEW_BUNDLE.md` §1–§4 | `docs/05_DOMAIN_MODEL.md` §2; `docs/21_TRACEABILITY_MATRIX.md` |
| **D6** | Model 1 Persistence Ownership: Schema v6 owned by P04A, Schema v9 owned by P04D, Linear Lease Chain (`predecessor_lease_id`, token monotonicity), Pure In-Memory P04B/C (Zero SQLite writes/audits/holds), Contract Sequencing (P04A -> P04B -> P04C -> P04D) | `docs/05_DOMAIN_MODEL.md` §1, §2; `docs/17_ROADMAP.md` §1, §2; `docs/22_MODULE_PROVENANCE.md` §1 | `docs/04_ARCHITECTURE.md` §7.5; `docs/phases/P04_EVIDENCE_REVIEW.md` §1 |
| **D7** | Canonical Audit Event Registration, RFC 8785 JCS Derivation, Descriptors A/B/C, `REVIEW_INTEGRITY_CONFLICT` Variant Discrimination (`conflict_source`: `'AUDIT_EVENT_ID_COLLISION'` vs `'WORKSPACE_BINDING_GUARD'`), Pipeline Isolation Invariant, Fail-Closed Operator Principal | `docs/05_DOMAIN_MODEL.md` §2; `docs/14_FAILURE_RECOVERY.md` §1; `docs/04_ARCHITECTURE.md` §7.5 | `docs/06_WORKFLOW_STATE_MACHINE.md` §2; `docs/22_MODULE_PROVENANCE.md` §1 |

---

## 3. Comprehensive Document Inventory for Reconciliation

The following 14 canonical documents and schemas constitute the complete inventory subject to single-pass reconciliation:

1. `docs/02_REQUIREMENTS.md`
2. `docs/04_ARCHITECTURE.md`
3. `docs/05_DOMAIN_MODEL.md`
4. `docs/06_WORKFLOW_STATE_MACHINE.md`
5. `docs/10_REVIEW_BUNDLE.md`
6. `docs/12_UPSTREAM_INTEGRATION.md`
7. `docs/14_FAILURE_RECOVERY.md`
8. `docs/17_ROADMAP.md`
9. `docs/21_TRACEABILITY_MATRIX.md`
10. `docs/22_MODULE_PROVENANCE.md`
11. `docs/phases/P04_EVIDENCE_REVIEW.md`
12. `docs/sources/SOURCE_REGISTRY.md`
13. `docs/sources/REUSE_MATRIX.md`
14. Schemas: `docs/schemas/worker-report.schema.json`, `docs/schemas/review-bundle.schema.json`, and evidence JSON schema specifications.

---

## 4. Itemized Canonical Specification Reconciliation Table

| Item ID | Decision / Invariant | Canonical File & Section | Literals, Schema DDL, & APIs to Synchronize | Subtask Owner | Acceptance & Verification Criteria |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **CR-01** | **D1: Live Lease Capability vs Snapshot & Effect Gate** | `docs/04_ARCHITECTURE.md` §7.1, §7.2; `docs/12_UPSTREAM_INTEGRATION.md` §1 | - Formalize `WorkspaceBindingSnapshot` (immutable data for DB/CAS lineage).<br>- Formalize `WorkspaceBindingLease` (live Win32 handle capability).<br>- Define single coordinator effect gate: `Coordinator.Dispatch` in `internal/dispatch/coordinator.go` executing 9-step sequence in same call frame.<br>- Prohibit invoking `/send` with snapshot only. | **P04A** | Sequence diagram and text show single coordinator effect gate holding live lease across entire dispatch sequence; Store layer checks DB rows only without claiming live lease authority. |
| **CR-02** | **D1: Exact P03 Restore & Provisioning SQL Predicates** | `docs/04_ARCHITECTURE.md` §7.1; `docs/12_UPSTREAM_INTEGRATION.md` §1; `docs/14_FAILURE_RECOVERY.md` §1 | - Guard 9: `NOT EXISTS (SELECT 1 FROM pair_restore_operations WHERE pair_id = ? AND resolution_state <> 'RESTORE_RESOLVED')`<br>- Guard 10: `NOT EXISTS (SELECT 1 FROM pair_provisioning_operations WHERE pair_id = ? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED'))`<br>- Total eradication of non-existent `status` column and `PENDING`/`IN_FLIGHT` descriptions. | **P04A** | Static grep verifies zero references to `pair_restore_operations` or `pair_provisioning_operations` with `status`, `PENDING`, or `IN_FLIGHT`. |
| **CR-03** | **D1: Governed Terminal Resolution** | `docs/06_WORKFLOW_STATE_MACHINE.md` §2; `docs/14_FAILURE_RECOVERY.md` §1 | - Pre-send binding rejection preserves `DISPATCHED` task state, `DISPATCH_BOUND` operation stage, and open attempt; `/send` is forbidden.<br>- Atomic D12 terminal transition: `tasks` (`DISPATCHED -> FAILED`), `task_attempts` (`ended_at = now`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'`), appends `TASK_STATE_TRANSITION` audit in same transaction.<br>- Separate hold resolution transaction after terminalization.<br>- Principal validation fail-closed on `VERIFIED_OPERATOR_PRINCIPAL`. | **P04A** | State machine table and failure recovery procedures document exact atomic D12 transition and distinct follow-up hold resolution. |
| **CR-04** | **D1: NTFS Directory Substitution Probes** | `docs/04_ARCHITECTURE.md` §7.1; `docs/14_FAILURE_RECOVERY.md` §1 | - Eradicate all citations of "directory hardlink swap" (unsupported by NTFS).<br>- Formalize directory substitution test matrix: junction points, directory symlinks, `subst` virtual drives, directory renames, path-alias probes. | **P04A** | Zero invalid directory hardlink swap claims across canonical files; valid NTFS substitution probes documented. |
| **CR-05** | **D2: Git Evidence Authority & Clean Intake** | `docs/02_REQUIREMENTS.md` (FR-008); `docs/04_ARCHITECTURE.md` §7.2; `docs/06_WORKFLOW_STATE_MACHINE.md` §2 | - Subtask P04B: pure in-memory Git evidence collector with zero SQLite writes.<br>- Clean worktree/index verification via `git status --porcelain=v1 -z` and `git diff-index --quiet HEAD --` with `GIT_OPTIONAL_LOCKS=0`.<br>- Dirty intake triggers atomic diagnostic transaction: appends `EVIDENCE_COLLECTION_FAILED` and inserts `ACTIVE` hold (`DIRTY_WORKTREE_DETECTED`) into `review_integrity_holds`, strictly preserving `RUNNING` task state (zero blanket transitions to `BLOCKED`).<br>- Verbatim `reported_head_sha` capture (`CHECK (LENGTH BETWEEN 7 AND 40)`). | **P04A & P04B** | Clean-worktree requirement formalized under FR-008; intake failure preserves `RUNNING` with diagnostic hold; zero SQLite writes in P04B. |
| **CR-06** | **D3: Windows AppContainer Isolation** | `docs/04_ARCHITECTURE.md` §7.3; `docs/02_REQUIREMENTS.md` (NFR-008); `docs/12_UPSTREAM_INTEGRATION.md` §2 | - Subtask P04C: pure in-memory test execution runner with zero SQLite writes.<br>- Windows process creation via `CreateProcessW` with `STARTUPINFOEXW`, explicit stdio-only `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`, atomic Job Object assignment via `PROC_THREAD_ATTRIBUTE_JOB_LIST` with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, and network restriction SID.<br>- Authoritative process-death proof (joined process handle/Job Object or host exclusivity across restarts).<br>- Stream capture separation: 10 MB capture limit vs 50 MB hard safety limit. | **P04C** | AppContainer, handle attribute list, and Job Object constraints accurately specified; stream limit boundary documented. |
| **CR-07** | **D4: Inert AO Test Harness** | `docs/12_UPSTREAM_INTEGRATION.md` §1; `docs/phases/P04_EVIDENCE_REVIEW.md` §2; `docs/sources/SOURCE_REGISTRY.md` | - Pure in-process synthetic session harness returning deterministic JSON fixtures without spawning live processes.<br>- Live AO integration held in unverified evidence track.<br>- Fail-closed invariant: `AUTOMATIC_RESTORE = DISABLED`. | **P04D** | Clear separation between inert harness used for automated test suites and live AO integration track. |
| **CR-08** | **D5: CAS Storage, Dual Hashes & ReviewBundle Latency Semantics** | `docs/04_ARCHITECTURE.md` §7.4; `docs/10_REVIEW_BUNDLE.md` §1–§4; `docs/02_REQUIREMENTS.md` (NFR-008) | - Content-Addressed Store with atomic write-through to `artifacts/<hex>/<sha256>`.<br>- Dual Head SHA validation (`actual_head_sha` vs `reported_head_sha`).<br>- ReviewBundle Latency Semantics (PROPOSAL-P04-002 Rev 9): assembly diagnostic interval (`compilation_latency_ms = bundle_assembled - evidence_finalized`), 2x2 measurement provenance matrix (`MEASURED_IN_PROCESS` vs `RECOVERED_AFTER_RESTART`), `nfr008_compliance_status = 'UNVERIFIED'`, decoupled best-effort post-commit monotonic telemetry outside DB/audit chain. | **P04D** | ReviewBundle specification, domain models, and schemas updated with assembly diagnostic latency semantics, RFC 8785 JCS, and dual hashes. |
| **CR-09** | **D6: Model 1 Persistence Ownership & Schema v6/v9** | `docs/05_DOMAIN_MODEL.md` §1, §2; `docs/17_ROADMAP.md` §1, §2; `docs/22_MODULE_PROVENANCE.md` §1 | - Schema v6 owned by Subtask P04A: `attempt_workspace_bindings` (trigger requiring `stage = 'DISPATCH_BOUND'`), `worker_claims`, `review_integrity_holds` (triggers, active dedup index).<br>- Schema v9 owned by Subtask P04D: `task_verification_leases` (linear lease chain, `predecessor_lease_id`, token monotonicity), `evidence_sets`, `review_artifacts`, `review_bundles`.<br>- Subtasks P04B and P04C: pure in-memory (zero migrations, zero SQLite writes, zero audit appends, zero hold mutations).<br>- Execution sequencing: P04A -> P04B -> P04C -> P04D; P04 runtime admission remains closed until P04D startup recovery. | **P04A & P04D** | Domain model and module provenance specify exact table ownership; DDL and triggers match accepted ADR-018; contract sequencing enforced. |
| **CR-10** | **D7: Audit Event Derivation, Descriptors A/B/C & Variant Discrimination** | `docs/05_DOMAIN_MODEL.md` §2; `docs/14_FAILURE_RECOVERY.md` §1; `docs/04_ARCHITECTURE.md` §7.5 | - RFC 8785 JCS event derivation (prohibiting delimiter string concatenation).<br>- Descriptor A (`hold_identity_descriptor` for `hold_id`).<br>- Descriptor B (`rejection_event_identity_descriptor` for `rejection_event_id`): incorporates `conflict_source` and exact variant fields.<br>- Descriptor C (`resolution_event_identity_descriptor` for `resolution_event_id`).<br>- `REVIEW_INTEGRITY_CONFLICT` Variant Discrimination:<br>  * Variant A (`AUDIT_EVENT_ID_COLLISION`): requires `colliding_event_id`, `attempted_event_type`, mismatch fields; zero binding mutation.<br>  * Variant B (`WORKSPACE_BINDING_GUARD`): requires `dispatch_operation_id`, 4 `attempted_reason` literals; `colliding_event_id` strictly absent; CAS invalidates binding.<br>- Pipeline isolation: P04D conflicts never CAS mutate workspace bindings.<br>- Zero orphan audit events on lost CAS races.<br>- Fail-closed operator principal (`VERIFIED_OPERATOR_PRINCIPAL`). | **P04A & P04D** | Audit registry, event schemas, descriptor formulas, and replay matrices fully document both variants, details schemas, and zero-orphan guarantees. |
| **CR-11** | **Traceability & Roadmap Alignment** | `docs/17_ROADMAP.md` §1, §2; `docs/21_TRACEABILITY_MATRIX.md`; `docs/phases/P04_EVIDENCE_REVIEW.md` | - Traceability links from FR-008 and NFR-008 through ADR-018 to Subtasks P04A–P04D.<br>- Roadmap milestones reflect decoupled Subtask P04A as first releaseable implementation contract.<br>- Phase P04 exit gate criteria updated to require verified AppContainer execution, linear lease recovery, and ReviewBundle CAS compilation. | **All** | Traceability matrix and roadmap reflect completed pre-contract architecture and exact P04A–P04D work breakdown. |
| **CR-12** | **Source Registry & Reuse Boundaries** | `docs/sources/SOURCE_REGISTRY.md`; `docs/sources/REUSE_MATRIX.md` | - Update reuse evaluation for Phase P04: standard library `crypto/sha256`, Windows API `kernel32.dll` (`CreateFileW`, `CreateProcessW`, Job Objects), `github.com/google/jsonschema-go` for schema validation.<br>- Pinned upstream facts: AO v0.13.0 wire contract unchanged; zero reliance on unverified AO endpoints. | **All** | Registry and reuse matrix confirm zero duplicate upstream implementations and document standard library / OS handle usage. |

---

## 5. Critical Invariants & Execution Guardrails

1. **Pre-Contract Gate Discipline**:
   - Single-pass canonical specification reconciliation must be completed, audited, and approved (`P04_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`) **BEFORE** Task Contract P04A (`CONTRACT-TASK-P04-001`) can be released.
   - Code writing for Phase P04 remains strictly held (`P04_CODE = HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT`).
   - Phase P05 ChatGPT Tool Interface code remains strictly unauthorized (`P05_CODE = NOT_AUTHORIZED`).
2. **Runtime Verification Evidence Separation**:
   - SQLite model probes verify database DDL, constraints, triggers, and state machine transitions at design level only.
   - Real Windows OS capabilities (Win32 anti-rename file handle sharing, Windows AppContainer sandbox, Job Object termination, NTFS directory substitution probes) belong strictly to the **Real Windows OS Security Boundary Track** and must be proven during Task Contract execution on physical Windows hosts.
   - Model probes must never be cited as proof of runtime Windows security boundary pass.
3. **Decoupled Historical Proof Draft**:
   - `docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md` Revision 5 remains `HISTORICAL_DEFERRED_NON_BLOCKING`.
   - `docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md` remains `RETIRED_NON_EXECUTABLE_DRAFT` (Model A Draft Lineage, status invariant: `NOT_RELEASED`).
   - Worktree binding runtime validation is embedded directly into Subtask P04A and P04D; it does not block P04A release.
4. **Strict Fail-Closed Dependencies**:
   - `AUTOMATIC_RESTORE = DISABLED` remains permanently fail-closed.
   - `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`: operator restore and linked stop remain disabled until verified operator principal reaches trusted boundary.
   - `LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`.
   - `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`.

---

## 6. Single-Pass Execution Strategy

Following approval of this scope plan by the External Supervisor:
1. A dedicated documentation reconciliation task contract (`CONTRACT-TASK-P04-DOC-RECONCILIATION-01`) will govern execution.
2. All 14 target files will be updated in a single, atomic documentation commit.
3. Post-execution verification will validate:
   - `git diff --check` passes with exit code 0.
   - Relative Markdown links pass with 0 broken links.
   - Static scan confirms 100% eradication of non-canonical predicates (`status`, `PENDING`, `IN_FLIGHT` on restore/provisioning; directory hardlink swaps).
   - Zero production Go code touched.
4. External Supervisor will audit the reconciled commit. Upon approval, Subtask P04A Task Contract will be prepared for release.
