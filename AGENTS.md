# AGENTS.md — Operational Directives & Behavioral Guardrails

This document establishes immutable operational directives for all AI coding agents working on the **AI Engineering Supervisor Control Plane** repository.

---

# SECTION 1: PERMANENT PROJECT RULES (APPLY ACROSS ALL PHASES)

1. **Source Registry Mandate**:
   - Never implement any subsystem without consulting `docs/sources/SOURCE_REGISTRY.md` and `docs/sources/REUSE_MATRIX.md`.
   - Never duplicate an upstream capability already provided by an active upstream (e.g., Agent Orchestrator, Antigravity CLI).

2. **Architecture Immutability & ADR Requirement**:
   - The canonical architecture documented in `docs/04_ARCHITECTURE.md` cannot be modified implicitly.
   - Any architectural modification, technology adoption, or upstream boundary adjustment requires an approved Architecture Decision Record (`docs/adr/`).

3. **Proposal System for New Ideas**:
   - If an agent discovers a new capability, optimization, or alternative during execution, **DO NOT IMPLEMENT IT**.
   - Create a structured proposal in `docs/proposals/PROPOSAL-xxx.md` following `docs/24_CHANGE_GOVERNANCE.md`.

4. **Task Scope Immutability**:
   - Every task dispatched to a worker is governed by an immutable **Task Contract** (`docs/08_TASK_CONTRACT.md`).
   - Workers must not expand their assigned task scope, alter files outside `allowed_scope`, or touch files in `forbidden_scope`.
   - If a blocker or architectural defect is discovered, transition the task to `BLOCKED` and stop.

5. **Evidence Over Claims**:
   - A worker report (`docs/09_WORKER_REPORT.md`) is a set of **claims**, not verified truth.
   - The Supervisor must independently verify all claims using Git diffs, test logs, exit codes, and artifact hashes before approval.

6. **State Resides Outside Chat**:
   - Conversation history is volatile context, not an authoritative database.
   - System state, task states, pair bindings, and review decisions reside exclusively in the Supervisor Control Plane and Git repository.

7. **Decision Hierarchy (Strict Governance)**:
   Whenever a conflict arises, resolve it using the immutable 9-level priority hierarchy defined in `docs/24_CHANGE_GOVERNANCE.md`:
   1. Confirmed User Requirement
   2. Approved ADR
   3. Canonical Architecture
   4. Requirement Specification
   5. Approved Roadmap
   6. Task Contract
   7. Source Repo / Reference
   8. Worker Suggestion
   9. Chat Conversation

8. **Mark Unverified Assumptions**:
   - Any assumption not backed by confirmed evidence must be explicitly marked: `GIẢ ĐỊNH:` / `ASSUMPTION:`.

9. **Pre- and Post-Execution Protocol**:
   - **Before implementation**: Read `docs/18_CURRENT_STATE.md`, assigned Task Contract, architectural references, and module provenance.
   - **After implementation**: Validate, collect evidence, generate report, and **STOP**. Do not automatically proceed to the next task.

---

# SECTION 2: CURRENT PHASE RULES — P03 TASK-P03-003B REMEDIATION

> [!CRITICAL]
> Phase P01 runtime proof activity is **COMPLETE**.
> Phase P02 domain and state storage core implementation is **COMPLETE** and approved by External Supervisor final audit (`P02_FINAL_AUDIT = EXTERNAL_AUDIT_APPROVED`, baseline commit `b073a1969d0b64d151e96f0d9b59b005d4cf518a`, evidence commit `7375bbe0c0f17ac6cc50f87a319cb936caf64905`).
> Core directions of `PROPOSAL-P03-001` are **EXTERNAL_APPROVED** by External Supervisor decision.
> Canonical specifications (`docs/02`, `docs/12`, `docs/14`, `docs/17`, `docs/21`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`) have been reconciled under Revision 3 and approved by External Supervisor (`P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`).
> Task TASK-P03-001 is **EXTERNAL_AUDIT_APPROVED** (`docs/audits/P03_TASK_001_EXTERNAL_REAUDIT_002.md`, commit `e5d3337772f4243ca0f85b4105430d1aa9329bd3`).
> Task TASK-P03-002 is **EXTERNAL_AUDIT_APPROVED** (`docs/audits/P03_TASK_002_EXTERNAL_REAUDIT_001.md`, commit `b85ddf686b7432f087ef776444eaabe183176e67`), and finding `P03T2R1-001` is **CLOSED**.
> External re-audit of `PROPOSAL-P03-002 Revision 3` (commit `00910f769368a28954ad03920da55a8885769ca7`) recorded findings `P03T3PR4-001` and `P03T3PR4-002` in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_003.md`.
> External re-audit of `PROPOSAL-P03-002 Revision 4` (commit `38154d7fa65888cbf25b0bc5206384198a9a8a40`) recorded findings `P03T3PR5-001`, `P03T3PR5-002`, and `P03T3PR5-003` in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_004.md`.
> External re-audit of `PROPOSAL-P03-002 Revision 5` (commit `fac2d5b3d763796077505f9f368a4b46056ec140`) recorded finding `P03T3PR6-001` in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_005.md`.
> Proposal `PROPOSAL-P03-002 Revision 6` (commit `8709af4b6aaf8613c69a10514971e320e3f90048`) is **EXTERNAL_APPROVED** in `docs/audits/P03_TASK_003_PROPOSAL_EXTERNAL_REAUDIT_006.md`.
> External audit of ADR-016 draft (commit `928b0d110058d8ca90d8d9b68d9aa478c1ee089a`) recorded findings `ADR16R1-001` through `ADR16R1-009` with verdict `ADR_016 = REVISION_1_REQUIRED` in `docs/audits/P03_ADR_016_EXTERNAL_AUDIT.md`.
> External re-audit 001 of ADR-016 Revision 1 (commit `f5fadbeef86f6bd3a1b782e42c0affdfc32c3165`) verified findings `ADR16R1-001` through `ADR16R1-009` CLOSED and recorded findings `ADR16R2-001` through `ADR16R2-009` with verdict `ADR_016 = REVISION_2_REQUIRED` in `docs/audits/P03_ADR_016_EXTERNAL_REAUDIT_001.md`.
> External re-audit 002 of ADR-016 Revision 2 (commit `11624f08d0c7351c7aeb047fa77305d2d5720bd1`) verified findings `ADR16R2-001` through `ADR16R2-009` SUBSTANTIVELY_CLOSED and recorded findings `ADR16R3-001` through `ADR16R3-010` with verdict `ADR_016 = REVISION_3_REQUIRED` in `docs/audits/P03_ADR_016_EXTERNAL_REAUDIT_002.md`.
> External re-audit 003 of ADR-016 Revision 3 (commit `ff5993970b0fcb97f6dccace9a377404958fca5c`) verified findings `ADR16R3-001` through `ADR16R3-010` CLOSED and recorded findings `ADR16R4-001` through `ADR16R4-008` with verdict `ADR_016 = REVISION_4_REQUIRED` in `docs/audits/P03_ADR_016_EXTERNAL_REAUDIT_003.md`.
> External re-audit 004 of ADR-016 Revision 4 (commit `dedcd9324b77b07d76cf7af36b363510975dfeb2`) verified findings `ADR16R4-001` through `ADR16R4-007` CLOSED/SUBSTANTIVELY_CLOSED, `ADR16R4-008` NOT_CLOSED, and recorded findings `ADR16R5-001` through `ADR16R5-005` with verdict `ADR_016 = REVISION_5_REQUIRED` in `docs/audits/P03_ADR_016_EXTERNAL_REAUDIT_004.md`.
> External re-audit 005 of ADR-016 Revision 5 (commit `68b3699cfdeb38a9ecb93771fe3a5f2823938a18`) verified findings `ADR16R5-004` and `ADR16R5-005` CLOSED, `ADR16R5-003` SUBSTANTIVELY_CLOSED, `ADR16R5-002` PARTIALLY_CLOSED, `ADR16R4-008` and `ADR16R5-001` NOT_CLOSED, and recorded findings `ADR16R6-001` through `ADR16R6-004` with verdict `ADR_016 = REVISION_6_REQUIRED` in `docs/audits/P03_ADR_016_EXTERNAL_REAUDIT_005.md`.
> External re-audit 006 of ADR-016 Revision 6 (commit `40d51ffe9af4dab5545f060d5bee2ffd441b6109`) verified findings `ADR16R6-001` through `ADR16R6-004`, `ADR16R4-008`, `ADR16R5-001`, and `ADR16R5-002` CLOSED with verdict `ADR_016 = EXTERNAL_APPROVED` and `ADR_016_ACCEPTANCE = GRANTED` in `docs/audits/P03_ADR_016_EXTERNAL_REAUDIT_006.md`.
> ADR-016 is formally **ACCEPTED** (`ADR_016 = EXTERNAL_APPROVED`, `ADR_016_ACCEPTANCE = GRANTED`).
> Scope of `PLAN-P03-CANONICAL-RECONCILIATION-ADR-016.md` (Items S1–S9) was formally **APPROVED** by External Supervisor at commit `906d5969347773bd7cf5bbe273b34fd4f549efce` with mandatory pinned AO restore route correction (`POST /api/v1/sessions/{sessionId}/restore`, `operationId: restoreSession`).
> Blocker `ADR16-PLAN-BLOCKER-001` is **CLOSED**.
> Historical canonical reconciliation baseline remains locked (`P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`, Revision 3).
> Single-pass canonical documentation reconciliation (Items S1–S9) pursuant to accepted ADR-016 is **EXTERNAL_AUDIT_APPROVED** at audited commit `41cf769b4cf8a980c95de3aa4be63140b04922c3` (`docs/audits/P03_ADR_016_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT.md`).
> Task Contract for documentation reconciliation `CONTRACT-TASK-P03-DOC-RECONCILIATION-01` completed (`docs/tasks/TASK_CONTRACT_P03_CANONICAL_RECONCILIATION_ADR_016.md`).
> Contract `CONTRACT-TASK-P03-003A-01` was approved by External Supervisor from candidate commit `56b5ef38ba5a45d56d58dd368a0c47cf45e0886f`, blob `5364edf076b31b8a42a37fee29193fc36ccdf413`, and is formally released as `docs/tasks/TASK_CONTRACT_P03_003A.md`.
> External Supervisor audited implementation commit `c793fb66069e780f2fcc97129456fb5068b44f3d` and required revision for findings `3A-R1-001` through `3A-R1-004` (`docs/audits/P03_TASK_003A_EXTERNAL_AUDIT_001.md`). `TASK_P03_003A_IMPLEMENTATION = REVISION_REQUIRED`.
> External Supervisor re-audited implementation commit `deeb4404e801d259cb589a2ef9b031f93b4b0367`, closed findings `3A-R1-001` through `3A-R1-004`, and approved `TASK_P03_003A_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED` (`docs/audits/P03_TASK_003A_EXTERNAL_REAUDIT_001.md`). The audited code was merged at `b8b0c95576d87677e8d48210d9838cc2f599752a`.
> Architecture classification established: `P03_ARCHITECTURE_CHANGE = YES`, `P03_ADR_REQUIRED = YES`.
> Parent `TASK-P03-003` is released for 3A and 3B only. Subtasks 3C and 3D remain **NOT_RELEASED**.
> `TASK_P03_003B_IMPLEMENTATION = REVISION_REQUIRED`; code remains authorized only for 3B remediation.
> The active phase gate is **`TASK_P03_003B_REMEDIATION`**.

> External Supervisor approved ADR-016 addendum design at audited SHA 3807343f06f99c0e11460c74522a2efb3be19491; ADD-R1-001..003 CLOSED. DESIGN_BLOCKER_3B_RESTORE_PROTOCOL=CLOSED_BY_APPROVED_ADDENDUM. Canonical reconciliation per addendum is EXTERNAL_AUDIT_APPROVED.
> Host/bootstrap integration owns verified operator principal evidence at the trusted boundary. Library fake-principal tests do not enable runtime restore or linked stop. AUTOMATIC_RESTORE=DISABLED. 3C/3D remain NOT_RELEASED; DESIGN_BLOCKER_3D_STARTUP_WIRING preserved.
> External Supervisor approved and released CONTRACT-TASK-P03-003B-01 from candidate SHA `ae8deb9ee41479d0e0868dced53ed89135252923`, blob `84bd84535352c2d28dbc1eba820aa8b22382dd99`, with code base SHA `b8b0c95576d87677e8d48210d9838cc2f599752a`. Only 3B implementation is authorized.

1. **TASK-P03-003B IMPLEMENTATION GATE**:
   - `TASK_P03_003B = RELEASED`; `TASK_P03_003B_IMPLEMENTATION = REVISION_REQUIRED`.
   - Remediate only `docs/tasks/TASK_CONTRACT_P03_003B.md`, from code base SHA `b8b0c95576d87677e8d48210d9838cc2f599752a` on branch `codex/p03-003b`.
   - Governance/release artifact stays separate from implementation diff. Read the release artifact and assigned contract as well as approved addendum/canonical specs.
   - Host/bootstrap verified principal is a fail-closed dependency; AUTOMATIC_RESTORE remains DISABLED unless completion evidence reaches the trusted boundary.

2. **TASK-P03-003 PARTIAL RELEASE GUARD**:
   - `TASK_P03_003A = RELEASED`.
   - `TASK_P03_003B = RELEASED`; `TASK_P03_003B_IMPLEMENTATION = REVISION_REQUIRED`.
   - `TASK_P03_003C = NOT_RELEASED` and `TASK_P03_003D = NOT_RELEASED`.
   - `DESIGN_BLOCKER_3D_STARTUP_WIRING` remains preserved for 3D and does not block 3A.

3. **EXPLICITLY FORBIDDEN IN THIS STAGE**:
   - Absolutely NO polling loops, tickers, or background worker threads;
   - Absolutely NO invented numeric operational timeout or poll interval defaults;
   - Absolutely NO synthetic worker heartbeat generation;
   - Absolutely NO SSE or `/api/v1/events` integration;
   - Absolutely NO workspace file fetching or WorkerReport handling;
   - No stop/kill coordinator, D6 clearance orchestration, poller, or startup scanner implementation in 3B;
   - No calls to a live AO or migration against a user's database for testing;
   - Absolutely NO modification to accepted ADR-016 or proposals.

4. **FINAL GOVERNANCE STATE**:
   - `P03_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED` (Historical Baseline A)
   - `P03_ADR_016_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_001 = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_002 = EXTERNAL_AUDIT_APPROVED`
   - `PROPOSAL_P03_002 = EXTERNAL_APPROVED`
   - `P03_ARCHITECTURE_CHANGE = YES`
   - `P03_ADR_REQUIRED = YES`
   - `ADR_016 = EXTERNAL_APPROVED`
   - `ADR_016_ACCEPTANCE = GRANTED`
   - `TASK_P03_003A = RELEASED`
   - `TASK_P03_003A_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_003B = RELEASED`
   - `TASK_P03_003B_IMPLEMENTATION = REVISION_REQUIRED`
   - `TASK_P03_003C = NOT_RELEASED`
   - `TASK_P03_003D = NOT_RELEASED`
   - `P03_CODE = AUTHORIZED_3B_ONLY`
   - `ACTIVE_GATE = TASK_P03_003B_REMEDIATION`
   - `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
   - `ADR_016_ADDENDUM_DESIGN = EXTERNAL_APPROVED` (audited `3807343f06f99c0e11460c74522a2efb3be19491`)
   - `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL = CLOSED_BY_APPROVED_ADDENDUM`
   - `P03_ADDENDUM_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_003B = RELEASED`; `TASK_P03_003B_IMPLEMENTATION = REVISION_REQUIRED`; `AUTOMATIC_RESTORE = DISABLED`
   - `P03_CODE = AUTHORIZED_3B_ONLY`; `ACTIVE_GATE = TASK_P03_003B_REMEDIATION`
   - Host/bootstrap integration owns verified operator principal evidence; until verified, runtime restore and linked stop remain disabled.

> **Current remediation gate (2026-09-23)**: External Supervisor audited implementation `4b45c55d47a9bc7ede4c778e21355a476e198169`: original suite PASS, five audit probes FAIL; findings `3B-R1-001..005` OPEN; verdict `REVISION_REQUIRED` (`docs/audits/P03_TASK_003B_EXTERNAL_AUDIT_001.md`). `TASK_P03_003B_IMPLEMENTATION = REVISION_REQUIRED`; `P03_CODE = AUTHORIZED_3B_ONLY`; `ACTIVE_GATE = TASK_P03_003B_REMEDIATION`. Preserve `AUTOMATIC_RESTORE = DISABLED`, 3C/3D `NOT_RELEASED`, host-principal dependency, and `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`.
