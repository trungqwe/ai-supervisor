# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL RE-AUDIT 002

- **Audit Record ID:** `P04-REAUDIT-002`
- **Audited Baseline Commit Reference:** `a7e80d7c677467380cbb754429672705563ca056`
- **Target Deliverables:**
  - [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) (Revision 3)
  - [`docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md) (Revision 3)
  - [`docs/plans/PLAN-P04-EVIDENCE-REVIEW.md`](../plans/PLAN-P04-EVIDENCE-REVIEW.md) (Revision 3)
  - [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md) (Revision 1)
  - [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md) (Revision 1)
- **Date:** 2026-09-26
- **Status:** `FORMALLY_RECORDED`
- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_3`
- **Authority:** External Supervisor Governance Directives ([`AGENTS.md`](../../AGENTS.md), [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md))

---

## 1. Executive Summary & Verdict

The External Supervisor conducted Re-Audit 002 on the Phase P04 pre-contract architecture deliverables at baseline `a7e80d7c677467380cbb754429672705563ca056`.

### Formal Governance Verdicts
- `PROPOSAL_P04_001 = REVISION_3_REQUIRED`
- `ADR_018 = REVISION_3_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_3_REQUIRED`
- `P04_WORKTREE_BINDING_PROOF_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_3`

---

## 2. Evaluation of Prior Findings (Inputs — Preserved)

Pursuant to External Supervisor directive, prior findings are preserved without unilateral worker closure:
- **`P04-ARCH-R2-001`**: `PARTIALLY_CLOSED` (Static path formula and branch convention verified; runtime proof plan and draft contract submitted, but draft contract remains `NOT_RELEASED` and empirical proof is not yet executed).
- **`P04-ARCH-R2-002`**: `PARTIALLY_CLOSED` (AppContainer and Window Station + Desktop isolation specified; external sandbox root requires strict containment and cleanup isolation).
- **`P04-ARCH-R2-003`**: `PARTIALLY_CLOSED` (RFC 8785 JCS canonical hashing designed; durable artifact store needs exact content-address key and crash consistency protocol).
- **`P04-ARCH-R2-004`**: `PARTIALLY_CLOSED` (Model 1 in-memory pipeline designed; lease fencing token model and CAS SQL on `tasks` table require exact schema reconciliation).

---

## 3. Re-Audit 002 Findings & Remediation Mandates

### 3.1. P04-ARCH-R3-001: PINNED_SOURCE_SYMBOL_AND_SESSION_ID_INACCURACY
- **Observation:** In Erratum 001, Re-Audit 001, Proposal Rev 3, ADR-018 Rev 3, and Plan Rev 3, citations to `backend/internal/adapters/workspace/gitworktree/workspace.go` in Agent Orchestrator used invalid line ranges and inaccurate Go signatures (e.g. claiming `managedPath` was at line 1887 with signature `func (w *Workspace) managedPath(cfg domain.WorkspaceConfig) string`). Furthermore, the proof plan hardcoded caller-selected session IDs (`sess-proof-alpha`, `sess-proof-beta`) despite `POST /api/v1/sessions` returning server-generated IDs.
- **Authoritative Facts:**
  - Pinned AO Commit: `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` (v0.13.0).
  - `Options`: lines 64–73; `ManagedRoot` at line 68 (raw blob lines 71–80, `ManagedRoot` at 75).
    Permalink: [`backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L71-L80)
  - `Workspace.Create`: lines 230–262; invokes `w.managedPath(cfg)` at line 243 (raw blob lines 246–266, call at line 257).
    Permalink: [`backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L246-L266)
  - `Workspace.Restore`: lines 1045–1112; invokes `w.restorePath(cfg)` at line 1055 (raw blob lines 1097–1140, call at line 1105).
    Permalink: [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1097-L1140)
  - `managedPath`: lines 1754–1762 (raw blob lines 1837–1846); exact signature `func (w *Workspace) managedPath(cfg ports.WorkspaceConfig) (string, error)`.
    Permalink: [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1837-L1846)
  - `restorePath`: lines 1764–1768 (raw blob lines 1848–1853); exact signature `func (w *Workspace) restorePath(cfg ports.WorkspaceConfig) (string, error)`.
    Permalink: [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1848-L1853)
  - `defaultSessionBranchName`: lines 1783–1785 (raw blob lines 1868–1870); exact signature `func defaultSessionBranchName(id domain.SessionID) string`.
    Permalink: [`backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870`](https://github.com/trungqwe/agent-orchestrator/blob/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/adapters/workspace/gitworktree/workspace.go#L1868-L1870)
  - Cross-reference: `internal/ao/session_commands.go` and `internal/ao/wire_types.go` confirm `POST /api/v1/sessions` assigns server-generated session IDs (`resp.Session.ID`).
- **Remediation Mandate:** Issue Erratum 002. Use GitHub permalinks with full commit SHA and line ranges. Delete all snippets referencing `domain.WorkspaceConfig` or returning single `string`. Proof runtime must capture server-generated session IDs and compute paths dynamically.

### 3.2. P04-ARCH-R3-002: RELATIVE_ROOT_AND_UNSAFE_CLEANUP
- **Observation:** Proposal, ADR, and Plans referenced relative roots (`.supervisor/proof/...`, `.supervisor/sandboxes/...`, `.supervisor/artifacts/...`). Relative paths risk directory traversal, cross-worktree contamination, and accidental deletion of user repositories or live databases.
- **Remediation Mandate:**
  - Eliminate all relative `.supervisor/...` paths.
  - Require an absolute host-injected `SUPERVISOR_STATE_ROOT`.
  - Child roots must be formatted as:
    * `<SUPERVISOR_STATE_ROOT>\proof\p04-worktree-binding\<proof_run_id>`
    * `<SUPERVISOR_STATE_ROOT>\sandboxes\<attempt_id>`
    * `<SUPERVISOR_STATE_ROOT>\artifacts`
  - Pre-operation containment check: canonicalize path (`filepath.Clean`, `EvalSymlinks`), verify physical containment strictly within allowed root, outside all Git worktrees, outside user repository (`D:\TU_CODE\ai-supervisor`), outside AO live DB and managed root, and outside recovery folder `D:\TU_CODE\ai-supervisor-user-wip-recovery\P03-EXIT-R2-001`.
  - Recursive cleanup permitted ONLY when resolved absolute target is strictly under the correct root and marker ownership (`.supervisor-owner-marker.json`) matches. Never delete computed targets if containment or physical identity is unproven.
  - If Windows permissions lack symlink/junction creation, record `UNVERIFIED_CAPABILITY` or `BLOCKED`, NOT `PASS`.

### 3.3. P04-ARCH-R3-003: EXECUTION_AUTHORITY_AND_INERT_HARNESS_ABSENT
- **Observation:** `DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md` specified `"verification_requests": []` despite requiring command execution. Workers must not supply raw executable paths, raw command strings, endpoints, or roots. Furthermore, AO harness inertness (zero LLM token generation, zero credentials, zero prompts) was unproven.
- **Remediation Mandate:**
  - Specify Option B (`VerificationPolicyCatalog` profiles) with typed parameters, enums, cwd policy, timeout, executable provenance, and output bounds.
  - Stage B policy check must be confirmed against live catalog at dispatch. Design-level catalog cannot report Stage B PASS (record `UNVERIFIED`).
  - `worker_profile` must be an existing, approved profile; if `proof-investigator` is unregistered, record `BLOCKER-P04-PROOF-PROFILE = PROFILE_NOT_YET_REGISTERED` and keep contract `NOT_RELEASED`.
  - Pin AO binary provenance: v0.13.0, commit `15e9ea9`, built from pinned source, SHA-256, config, ephemeral port, process ownership.
  - Record `DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN` until inert harness is proven. Do not claim "zero LLM call" merely from not sending a coding prompt.

### 3.4. P04-ARCH-R3-004: RESTORE_AND_NEGATIVE_PROBE_STATE_MACHINE_INVALID
- **Observation:** Restore lifecycle was underspecified, and negative probes treated expected rejections or worktree recreation as infrastructure failures, erroneously triggering stop conditions.
- **Remediation Mandate:**
  - Restore procedure: (1) create disposable session, get server-generated `sessionId`; (2) transition session to terminated state via supported lifecycle; (3) observe and record terminal state evidence; (4) restart disposable AO with same DB and managed root; (5) call `/restore` directly in proof isolation; (6) verify path, branch, physical identity.
  - Reiterate that Supervisor `AUTOMATIC_RESTORE` remains strictly `DISABLED`.
  - Missing-directory probe must "characterize observed behavior" (AO `Workspace.Restore` may re-create the worktree at the same path). Do not treat this as a failure.
  - Distinguish expected negative outcomes from unexpected infrastructure failures. Expected negative outcomes must NOT trigger stop conditions.

### 3.5. P04-ARCH-R3-005: ARTIFACT_ADDRESSING_AND_CRASH_CONSISTENCY_UNDERSPECIFIED
- **Observation:** Artifact addressing used ambiguous `<sha256>`. Truncated output prefix was conflated with full stream hash. Filesystem rename rollback was incorrectly attributed to SQLite transactions.
- **Remediation Mandate:**
  - Pin exact content-address key: stored files containing captured prefix use `captured_sha256` as canonical filename (`<SUPERVISOR_STATE_ROOT>\artifacts\<captured_sha256>`).
  - `full_stream_sha256` is strictly stream metadata recorded in SQLite `review_artifacts`.
  - Specify byte-exact deduplication, collision verification, media/encoding boundaries, and multiple attempt references.
  - Define crash-consistent protocol: write staging -> fsync file + parent directory -> atomic rename -> SQLite CAS metadata transaction.
  - SQLite rollback leaves unreferenced orphan files; startup GC cleans unreferenced staging/orphans after grace period and containment check.
  - Include Crash Matrix covering all boundaries.

### 3.6. P04-ARCH-R3-006: TASKSTATE_CAS_AND_LEASE_MODEL_SCHEMA_MISMATCH
- **Observation:** Pseudo-SQL targeted `task_attempts` with `status = 'EVIDENCE_READY'`. `task_attempts` has no status column; `TaskState` is located in `tasks.state`. Pre-command lease reservation was unspecified.
- **Remediation Mandate:**
  - Correct all pseudo-SQL to update `tasks.state`:
    ```sql
    UPDATE tasks
    SET state = 'EVIDENCE_READY', updated_at = ?
    WHERE task_id = ?
      AND state = 'REPORT_READY'
      AND current_attempt = ?
      AND EXISTS (
          SELECT 1 FROM task_verification_leases l
          WHERE l.task_id = tasks.task_id
            AND l.attempt_id = ?
            AND l.fencing_token = ?
            AND l.expires_at > ?
      );
    ```
  - Define schema, migration owner (Schema v9, owned by P04D), and lifecycle of `task_verification_leases`.
  - Lease acquisition occurs before external commands in a separate transaction with fencing token. Loser denied execution (zero commands executed).
  - P04B and P04C remain pure collectors with zero SQLite writes.
  - P04D executes single final atomic SQLite evidence transaction.
  - Fencing token check prevents stale resumed worker from committing evidence after lease was reclaimed.

### 3.7. P04-ARCH-R3-007: GOVERNANCE_STATE_DRIFT
- **Observation:** `AGENTS.md` and `docs/18_CURRENT_STATE.md` contained drifted references to Revision 1 / Revision 2, outdated `R1-001..006 OPEN_PENDING_REAUDIT`, and inactive gates.
- **Remediation Mandate:** Synchronize `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_3`, `REVISION_3_REQUIRED`, and current finding statuses. Clearly demarcate historical records.

---

## 4. Execution Directives for Remediation Round 3

1. **Deliverables:** Submit Revision 4 of Proposal, ADR-018, and Execution Plan; Revision 2 of Bounded Proof Plan and Draft TaskContract; Erratum 002; and synchronized `AGENTS.md` and `docs/18_CURRENT_STATE.md`.
2. **File Whitelist:** Exactly 9 files authorized. Zero Go code, zero schema migrations, zero live AO runs, zero proof directory creations, zero model calls.
3. **Draft Contract Status:** `DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md` remains strictly **`NOT_RELEASED`**.
4. **Dependencies:** `AUTOMATIC_RESTORE = DISABLED`; `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`; `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`.
