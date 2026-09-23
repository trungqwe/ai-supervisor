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

# SECTION 2: CURRENT PHASE RULES — P03 TASK-P03-003D IMPLEMENTATION

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
> Parent `TASK-P03-003` is released for 3A, 3B, 3C and 3D; only the 3D contract authorizes current implementation.
> External Supervisor approved `TASK_P03_003B_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED` at `442e87e075e1540738ceeae00fdfa045d440cea8` (`docs/audits/P03_TASK_003B_EXTERNAL_REAUDIT_003.md`). The approved implementation was merged without code changes at `246f74eabd09bbc2ea66624ee217035167824a86`; post-merge race/full-suite checks passed.
> External Supervisor released `CONTRACT-TASK-P03-003C-01` from candidate `a8f8edc7a64f3d1aa66acf5519397fe46d8f6177`, blob `7a689f16f04b35acecb7952dc43873ece6f7df6f`; findings `3C-C1-001..003=CLOSED` at design level. Historical `BLOCKER-3C-001` closed at design level. Historical audit of `90a168daa2fcd114e1fcbba3c15319897c257b42` required remediation (`docs/audits/P03_TASK_003C_EXTERNAL_AUDIT_001.md`). External Supervisor approved implementation `3d223e47eb4ba05809b196fc77caca61f94421eb`, closed `3C-R1-001..003`, and merged the audited code at `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527` (`docs/audits/P03_TASK_003C_EXTERNAL_REAUDIT_001.md`). `TASK_P03_003C_IMPLEMENTATION=EXTERNAL_AUDIT_APPROVED`; `TASK_P03_003D=RELEASED`; BLOCKER-3D-001/002/003 closed at design level by approved clarification/addendum. External audit of 3D implementation `710b27f2d41dcbadd385c6ac07e30bb1765d1cbe` requires revision (`3D-R1-001..005=OPEN`); (Lịch sử: khi đó `TASK_P03_003D_IMPLEMENTATION=REVISION_REQUIRED` theo contract revision 2). Hiện tại: contract revision 3 đã RELEASED; implementation 3D EXTERNAL_AUDIT_APPROVED tại `7513f1b`; findings 3D-R1-001..005 và 3D-R2-001..002 đã CLOSED ở library scope; code đã merge tại `35909d7`; ACTIVE_GATE=TASK_P03_003D_HANDOFF_VERIFICATION.
> Lịch sử trước decision `3babaa2`: Supervisor xác nhận hướng cho `3D-R1-005`: `dispatch_operations.confirmed_at` là origin của send-confirmation-based execution budget (không phải actual execution start); deadline/policy bất biến; startup zero timeout effect; runtime monitor đi qua shared host admission và stop coordinator 3C. Khi đó `PROPOSAL-P03-004`, draft ADR-016 addendum và draft contract revision 3 chờ design audit. Implementation branch `codex/p03-003d` tại `00f64d0b26db2749eadd21b7307c98faa6b6280d` được giữ nguyên; `3D-R1-005=OPEN`, chưa cấp quyền migration hoặc code deadline.
> External Supervisor chấp thuận thiết kế execution budget tại governance `3babaa2ba573fcdd7d6b34cfb5ea2fd44f1fffe4` với ràng buộc EXEC-R1-001..003: trusted maintenance scope, exact open RUNNING manual stop, Tx R tái dùng atomic double quarantine 3C, bỏ quyền effect giữ STOP_REQUESTED/IN_FLIGHT, không dùng RelinquishTimeoutEffect, và whitelist migration v4 test có giới hạn. ADR addendum chính thức/canonical reconciliation đã chuẩn bị; candidate contract revision 3 vẫn NOT_RELEASED và chờ release audit. `3D-R1-005=OPEN`; implementation `00f64d0b26db2749eadd21b7307c98faa6b6280d` giữ nguyên, chưa có quyền migration/code deadline mới. `HOST_QUIESCENCE_INTEGRATION=OPEN`, verified host principal và startup wiring dependency giữ nguyên.
> `TASK_P03_003D_REVISION_3=NOT_RELEASED`; `TASK_P03_003D_IMPLEMENTATION=REVISION_REQUIRED`; `P03_CODE=AUTHORIZED_3D_ONLY` theo contract revision 2 đã release; `ACTIVE_GATE=TASK_P03_003D_REMEDIATION`. Addendum design approval không tự thay đổi gate implementation.
> External Supervisor approved canonical reconciliation at `68194f739fc5a514cd1f7850027771a77a866d47` and released `CONTRACT-TASK-P03-003D-03` from candidate blob `0c15ca55403681f8647e5b2dc242fcf91ec5a521`. **Current:** `TASK_P03_003D_REVISION_3=RELEASED`; `TASK_P03_003D_IMPLEMENTATION=EXTERNAL_AUDIT_APPROVED`; findings `3D-R1-001..005` và `3D-R2-001..002` CLOSED ở library scope; code đã MERGED tại `35909d7`; `ACTIVE_GATE=TASK_P03_003D_HANDOFF_VERIFICATION`. `AUTOMATIC_RESTORE=DISABLED`; verified host principal, `HOST_QUIESCENCE_INTEGRATION=OPEN` và startup wiring dependency (`DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`) persist.
> External audit 002 of 3D remediation commit `d9d2fcc57ac95b098ac02f81686a891afa1aab19` recorded findings `3D-R2-001` (waiting_input timeout) and `3D-R2-002` (pre-send policy validation) with verdict `REVISION_REQUIRED`; suite gốc race PASS, hai probes FAIL; `3D-R1-005` chưa đóng (`docs/audits/P03_TASK_003D_EXTERNAL_AUDIT_002.md`). Chưa merge. `TASK_P03_003D_IMPLEMENTATION=REVISION_REQUIRED`; `P03_CODE=AUTHORIZED_3D_ONLY`; `ACTIVE_GATE=TASK_P03_003D_REMEDIATION`; `AUTOMATIC_RESTORE=DISABLED`; `HOST_QUIESCENCE_INTEGRATION=OPEN`; `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`.
> External re-audit 001 of 3D remediation commit `7513f1b9f39fa459be15e0abc836c6258a610d8b` (code base `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`, governance baseline `422ca9e5fa41b263b02d888c81850f8ebc9c0e31`) closed findings `3D-R1-001..005` and `3D-R2-001..002` at library implementation scope with verdict `TASK_P03_003D_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED` (`docs/audits/P03_TASK_003D_EXTERNAL_REAUDIT_001.md`). External Supervisor verified `go test -race -count=1 ./...` exit 0, whitelist diff 23/32, outside scope 0, `diff --check` exit 0. Chưa suy ra host integration hoặc runtime deployment đã đạt; `AUTOMATIC_RESTORE = DISABLED`; `HOST_QUIESCENCE_INTEGRATION = OPEN`; `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`; verified host principal chưa có bằng chứng runtime.
> Audited implementation `7513f1b9f39fa459be15e0abc836c6258a610d8b` was merged into main at `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea` (tree `internal/` khớp `a91fcdc207116bff3a40a58beea5021de98392eb`); post-merge integration `go test -race -count=1 ./...` exit 0, `diff --check` exit 0 (`docs/audits/P03_TASK_003D_MERGE_INTEGRATION_AUDIT.md`). Handoff dependencies: `HOST_QUIESCENCE_INTEGRATION = OPEN`, verified host principal chưa có bằng chứng runtime, `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`, `AUTOMATIC_RESTORE = DISABLED`. Chưa cấp quyền P04 hoặc daemon/bootstrap.

> External Supervisor approved ADR-016 addendum design at audited SHA 3807343f06f99c0e11460c74522a2efb3be19491; ADD-R1-001..003 CLOSED. DESIGN_BLOCKER_3B_RESTORE_PROTOCOL=CLOSED_BY_APPROVED_ADDENDUM. Canonical reconciliation per addendum is EXTERNAL_AUDIT_APPROVED.
> Host/bootstrap integration owns verified operator principal evidence at the trusted boundary. Library fake-principal tests do not enable runtime restore or linked stop. AUTOMATIC_RESTORE=DISABLED. 3D is RELEASED for implementation; DESIGN_BLOCKER_3D_STARTUP_WIRING preserved.
> External Supervisor approved and released `CONTRACT-TASK-P03-003D-01` from candidate commit `7701b8b02c68006c4e692a1c8a61609c72a6c033`, blob `604c4786281354565c8e37238f0b5ba95542e4f9`, code base `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527` (`docs/audits/P03_TASK_003D_CONTRACT_RELEASE_AUDIT.md`).
> Historical 3B release: External Supervisor released CONTRACT-TASK-P03-003B-01 from candidate SHA `ae8deb9ee41479d0e0868dced53ed89135252923`, blob `84bd84535352c2d28dbc1eba820aa8b22382dd99`, with code base SHA `b8b0c95576d87677e8d48210d9838cc2f599752a`.

1. **TASK-P03-003D IMPLEMENTATION GATE**:
   - `TASK_P03_003C = RELEASED`; implementation `EXTERNAL_AUDIT_APPROVED` tại SHA `3d223e47eb4ba05809b196fc77caca61f94421eb`. `TASK_P03_003D = RELEASED` (contract revision 3); implementation `EXTERNAL_AUDIT_APPROVED` tại SHA `7513f1b9f39fa459be15e0abc836c6258a610d8b` và đã MERGED tại `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`.
   - Code baseline cho released contract 3D là merge commit `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`; release artifact được giao riêng, không đưa governance vào implementation diff.
   - Host/bootstrap verified principal vẫn là dependency fail-closed; `AUTOMATIC_RESTORE = DISABLED`. Test principal không chứng minh host authentication.

2. **TASK-P03-003 PARTIAL RELEASE GUARD**:
   - `TASK_P03_003A = RELEASED`.
   - `TASK_P03_003B = RELEASED`; `TASK_P03_003B_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED`.
   - `TASK_P03_003C = RELEASED` and `TASK_P03_003D = RELEASED`.
   - `DESIGN_BLOCKER_3D_STARTUP_WIRING` remains preserved for 3D and does not block 3A.

3. **EXPLICITLY FORBIDDEN IN THIS IMPLEMENTATION STAGE**:
   - No daemon bootstrap/server implementation or claim of recovery-before-listener proof;
   - Absolutely NO invented numeric operational timeout or poll interval defaults;
   - Absolutely NO synthetic worker heartbeat generation;
   - Absolutely NO SSE or `/api/v1/events` integration;
   - Absolutely NO workspace file fetching or WorkerReport handling;
   - No production change outside the released 3D contract whitelist;
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
   - `TASK_P03_003B_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_003C = RELEASED`
   - `TASK_P03_003C_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_003D = RELEASED`
   - `TASK_P03_003D_IMPLEMENTATION = EXTERNAL_AUDIT_APPROVED`
   - `TASK_P03_003D_MERGE = MERGED` (merge commit `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`; `internal/` tree `a91fcdc207116bff3a40a58beea5021de98392eb`; `docs/audits/P03_TASK_003D_MERGE_INTEGRATION_AUDIT.md`; erratum 002 `a59e55f3fbe0a341f5d8ac54334962110c4842f4`).
   - `P03_CODE = COMPLETED_PENDING_SUPERVISOR_HANDOFF`
   - `ACTIVE_GATE = TASK_P03_003D_HANDOFF_VERIFICATION` (chờ re-audit thiết kế PROPOSAL-P03-005 Revision 2 và DRAFT-ADR-017 Revision 2 cho TASK-P03-004)
   - `HOST_QUIESCENCE_INTEGRATION = OPEN`
   - `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
   - `ADR_016_ADDENDUM_DESIGN = EXTERNAL_APPROVED` (audited `3807343f06f99c0e11460c74522a2efb3be19491`)
   - `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL = CLOSED_BY_APPROVED_ADDENDUM`
   - `P03_ADDENDUM_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`
   - `AUTOMATIC_RESTORE = DISABLED`
   - Host/bootstrap integration owns verified operator principal evidence; until verified, runtime restore and linked stop remain disabled.

> **Historical 3B remediation (superseded)**: Earlier implementation `4b45c55d47a9bc7ede4c778e21355a476e198169` had verdict `REVISION_REQUIRED` (`docs/audits/P03_TASK_003B_EXTERNAL_AUDIT_001.md`); subsequent re-audits are preserved in `docs/audits/P03_TASK_003B_EXTERNAL_REAUDIT_001.md` through `_003.md`. The current approval and gate are stated above.

> `BLOCKER-3C-001=CLOSED_AT_DESIGN_LEVEL` by `docs/adr/ADR-016-CLARIFICATION-administrative-stop-risk-acceptance.md`; 3C implementation was subsequently approved by External Supervisor.
