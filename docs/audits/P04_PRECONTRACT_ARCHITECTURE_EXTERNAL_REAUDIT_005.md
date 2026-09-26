# P04 Pre-Contract Architecture External Re-Audit 005

> **Audit Type**: External Supervisor Pre-Contract Architecture Re-Audit 005
> **Audited Commit**: `742e2e296db99c19bc7a645ca1842520b2ab23e5`
> **Auditor**: External Supervisor
> **Date**: 2026-09-26
> **Scope**: Pre-Contract Architecture Documents for Phase P04 (Proposal Rev 6, Draft ADR-018 Rev 6, Plan Rev 6, Bounded Proof Plan Rev 4, Draft Proof Contract Model A)

---

## 1. Audit Verdict

```
PROPOSAL_P04_001 = REVISION_6_REQUIRED
ADR_018 = REVISION_6_REQUIRED
PLAN_P04_EVIDENCE_REVIEW = REVISION_6_REQUIRED
ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_6
P04_TASK_CONTRACT = NOT_RELEASED
P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT
P05_CODE = NOT_AUTHORIZED
AUTOMATIC_RESTORE = DISABLED
LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK
VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY
```

---

## 2. Status of Prior Findings

- **`P04-ARCH-R5-001`**: `SUBSTANTIVELY_CLOSED_WITH_R6_FOLLOWUP`
  Acknowledged that disposable upstream Agent Orchestrator runtime worker sessions cannot execute pre-contract runtime proofs due to absence of an inert harness in `AllHarnesses`. Static source inspection track retained as design evidence. Operational worktree binding validation transitioned to runtime execution, but naming collisions with canonical Stage B and execution graph prerequisite deadlocks require R6 followup (`P04-ARCH-R6-002`).
- **`P04-ARCH-R5-002`**: `PARTIALLY_CLOSED`
  Authoritative `attempt_workspace_bindings` table specified, but schema types conflicted with P03 domain models and migrations (`terminal_generation` integer vs opaque string), references to nonexistent `nonexistent generation on worker_sessions` remained, lineage cross-pairing lacked complete foreign keys and triggers, and OS handle lifecycle invariants were incomplete (`P04-ARCH-R6-001`).
- **`P04-ARCH-R5-003`**: `PARTIALLY_CLOSED`
  Monotonic lease fencing specified with preserved rows on release, but lacked contract lineage foreign keys, strict CHECK constraint invariants on state timestamps, and complete 3-boundary pipeline transaction atomicity (`P04-ARCH-R6-006`).
- **`P04-ARCH-R5-004`**: `PARTIALLY_CLOSED`
  Strict append-only scope reduction and physical GC deferral adopted, but hash CHECK constraints lacked lowercase hex validation, retrieval metadata lacked media type, encoding, and canonical relative path, and staging/final atomic `MoveFileExW` durability and post-rename handle reopening verification was incomplete (`P04-ARCH-R6-005`).
- **`P04-ARCH-R5-005`**: `PARTIALLY_CLOSED`
  Traceability updated to FR-008 and 5-second language removed, but NFR-008 latency measurement semantics were modified without formal change governance approval, conflicting with canonical `docs/02_REQUIREMENTS.md` (`P04-ARCH-R6-006`).
- **`P04-ARCH-R4-001..006`**: Preserved in their recorded statuses (`R4-001=CLOSED`, `R4-002..003=SUBSTANTIVELY_CLOSED`, `R4-004..005=PARTIALLY_CLOSED`, `R4-006=SUBSTANTIVELY_CLOSED`).
- **`P04-ARCH-R3-001..007`**: Preserved in their recorded statuses.

---

## 3. New Findings (Round 6)

### P04-ARCH-R6-001 — Worktree Binding Schema & Lifecycle Incompatible
- **Classification**: DATA INTEGRITY & ARCHITECTURAL INCOMPATIBILITY
- **Observation**:
  1. `terminal_generation` was typed as `INTEGER` in proposal, ADR, and plan, but in canonical P03 domain models (`internal/store/migrations.go`, `docs/05_DOMAIN_MODEL.md`) `worker_sessions.terminal_generation` and `task_attempts.terminal_generation` are `TEXT` (opaque string).
  2. Proposal and ADR referenced nonexistent `nonexistent generation on worker_sessions`.
  3. `attempt_workspace_bindings` lacked foreign keys on `contract_id` and `session_id` with `ON DELETE RESTRICT`, and lacked lineage integrity protection preventing cross-pairing between `attempt_id`, `task_id`, `contract_id`, and `session_id`.
  4. CHECK constraints for non-empty paths, volume serial numbers, 128-bit FileId, and exact 40-character lowercase hex format for `pinned_ao_commit` were missing or incomplete.
  5. OS handle lifecycle lacked rigorous phase definitions: binding must be established when worktree is materialized via open handle verification before `SEND_REQUESTED`, re-opened and verified before evidence lease acquisition, and held or revalidated immediately prior to final durable CAS.
- **Remediation Directives**:
  - Update `terminal_generation` to `TEXT NOT NULL CHECK (LENGTH(terminal_generation) > 0)` across all DDL, proposals, ADRs, and plans.
  - Eradicate every occurrence of `nonexistent generation on worker_sessions` and use `worker_sessions.terminal_generation`.
  - Add `contract_id REFERENCES task_contracts(contract_id) ON DELETE RESTRICT` and `session_id REFERENCES worker_sessions(session_id) ON DELETE RESTRICT`. Add composite foreign keys and lineage triggers preventing cross-pairing of task, attempt, and contract.
  - Enforce CHECK constraints: `LENGTH(canonical_worktree_path) > 0`, `LENGTH(managed_root_final_path) > 0`, `LENGTH(volume_serial_number) > 0`, `LENGTH(file_id) > 0`, and `LENGTH(pinned_ao_commit) = 40 AND NOT (pinned_ao_commit GLOB '*[^0-9a-f]*')`.
  - Formalize handle lifecycle: verify VolumeSerialNumber + 128-bit FileId via OS handle -> register immutable binding before `SEND_REQUESTED` -> re-open handle and compare before acquiring verification lease -> hold root handle or revalidate immediately before durable CAS. Fail closed if verification fails.

### P04-ARCH-R6-002 — Stage B Namespace Collision & Blocked Prerequisite Deadlock
- **Classification**: ARCHITECTURE & EXECUTION GOVERNANCE DEFECT
- **Observation**:
  1. Proposal Rev 6 and Plan Rev 6 used the term "Stage B" for worktree binding validation, colliding with canonical "Stage B" semantic validation of `TaskContractValidator` and `STAGE_B_RUNTIME_CATALOG` (`docs/08_TASK_CONTRACT.md`, `docs/18_CURRENT_STATE.md`).
  2. Modeling `DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF` as an unreleaseable prerequisite blocking Subtask P04A (`BLOCKED_NOT_RELEASEABLE proof contract -> P04A`) created an architectural deadlock where Phase P04 could never begin.
- **Remediation Directives**:
  - Cease all use of "Stage B" for worktree binding. Name the operational mechanism `WORKTREE_BINDING_RUNTIME_VALIDATION`.
  - Reserve "Stage B" strictly for `TaskContractValidator` semantic validation and `STAGE_B_RUNTIME_CATALOG`.
  - Remove the blocked proof contract dependency edge from the execution graph. Subtask P04A is the first releaseable subtask once architecture is approved.
  - Transition the proof plan and contract to a historical deferred non-blocking track (`NOT_RELEASED`, `RETIRED_NON_EXECUTABLE_DRAFT` / `DEFERRED_NON_BLOCKING`).
  - Physical worktree binding validation becomes a fail-closed operational acceptance and integration gate for P04A and P04D.

### P04-ARCH-R6-003 — Worktree TOCTOU & Git Command Authority Incomplete
- **Classification**: SECURITY & COMMAND AUTHORITY DEFECT
- **Observation**:
  1. Git executable resolution relied on implicit host `PATH`, violating NFR-007.
  2. Hook and config isolation proposed Unix null device syntax, which is invalid and non-portable on Windows.
  3. Git diff/log invocations omitted `--no-ext-diff` and `--no-textconv`, allowing potential external tool execution via repository `.gitattributes` or config. Literal pathspecs were not mandated.
  4. Pre/post Git execution invariants lacked identity checks ensuring zero worktree or HEAD mutation.
  5. Traceability for Git Evidence Collection cited `FR-007` instead of `FR-008`.
- **Remediation Directives**:
  - Resolve trusted Git binary from host configuration (`SUPERVISOR_GIT_BIN`), version and hash pinned per NFR-007. Arbitrary `PATH` lookup is strictly prohibited.
  - Use a dedicated trusted empty directory under `<SUPERVISOR_STATE_ROOT>/trusted_empty_git/` for hooks and empty config on Windows. Strictly zero use of Unix null device path syntax.
  - Isolate environment: unset `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, object directories. Set `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_SYSTEM` pointing to trusted empty config, `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`, `--no-pager`, `GIT_PAGER=cat`.
  - Explicit allowlist of non-mutating Git commands only. Prohibit mutating and network commands. Always pass `--no-ext-diff` and `--no-textconv`. Use literal pathspecs for file inputs.
  - Implement timeouts, byte caps, process-tree termination, and pre/post HEAD/index/worktree identity verification.
  - Correct requirement traceability to `FR-008` (Git-Verified Diffs).

### P04-ARCH-R6-004 — Windows Verification Isolation Not a Security Boundary
- **Classification**: SECURITY ARCHITECTURE DEFECT
- **Observation**:
  1. Proposal Rev 6 treated Window Station and Desktop isolation (`CreateWindowStationW` + `CreateDesktopW`) as a process security boundary, whereas it is strictly a UI message/clipboard isolation mechanism.
  2. The concept of "mounting the worker worktree read-only" is undefined and infeasible on standard Windows filesystems without specialized drivers.
  3. Security boundary lacked a concrete, viable implementation architecture.
- **Remediation Directives**:
  - Adopt a concrete implementation path: Windows AppContainer via `CreateAppContainerProfile` with empty `SECURITY_CAPABILITIES` (zero network capabilities) or restricted primary token (`CreateRestrictedToken`) with network denial.
  - Filesystem DACLs: verification executes against a DACL-protected immutable source snapshot (`<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/`) with read-only access granted to the sandbox SID. Ephemeral toolchain writes (`TEMP`, `TMP`, `GOCACHE`, `GOPATH`), stdout/stderr, and artifacts are confined to sandbox root (`<SUPERVISOR_STATE_ROOT>/sandboxes/<attempt_id>/`) with write access.
  - The worker worktree is never directly exposed to verification subprocesses; changes from snapshot are never copied back to worktree.
  - Window Station and Desktop isolation is retained purely as supplementary UI isolation.
  - Job Object: `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, breakaway explicitly disabled, process/memory/time limits, zero inherited handles (`bInheritHandles = FALSE`).
  - Executables resolved from approved `VerificationPolicyCatalog` with trusted absolute paths.
  - If security boundary cannot be established, fail closed immediately. Falling back to daemon token execution is strictly prohibited.

### P04-ARCH-R6-005 — Artifact Store Durability & Retrieval Schema Incomplete
- **Classification**: DATA INTEGRITY & STORAGE ARCHITECTURE DEFECT
- **Observation**:
  1. Hash CHECK constraints on `review_artifacts` checked only character length (`LENGTH = 64`), failing to restrict characters to valid lowercase hex (`GLOB '*[^0-9a-f]*'`).
  2. Retrieval metadata lacked media type, encoding, and canonical relative path necessary for safe artifact extraction.
  3. Staging and final directory lacked volume containment checks, same-volume enforcement, and atomic write-through rename semantics.
  4. Post-rename handle reopening and identity/size/hash verification prior to SQLite metadata commit was unspecified.
  5. Existing file collision handling was underspecified.
  6. Lack of an authoritative reconciliation matrix between `docs/10_REVIEW_BUNDLE.md`, `review-bundle.schema.json`, domain models, and proposed DDL.
- **Remediation Directives**:
  - Update hash CHECK constraints to: `LENGTH(col) = 64 AND NOT (col GLOB '*[^0-9a-f]*')`.
  - Add retrieval metadata to `review_artifacts`: `artifact_type`, `media_type`, `encoding`, `canonical_relative_path`, `captured_sha256`, `full_stream_sha256`, byte counts, and `is_truncated`.
  - Verify staging and final roots for volume identity, containment, and same-volume requirement. Flush staging file; rename via `MoveFileExW(MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH)`.
  - Reopen final file by handle, verify volume identity, size, and SHA-256 hash before committing metadata in SQLite.
  - On existing file collision: re-hash and compare size; mismatch fails closed and quarantines file, never blindly overwrites.
  - Physical GC remains deferred.
  - Include an authoritative reconciliation matrix across `docs/10`, `review-bundle.schema.json`, domain model, and proposed DDL.

### P04-ARCH-R6-006 — State Machine Atomicity & NFR-008 Contradiction
- **Classification**: REQUIREMENT GOVERNANCE & PIPELINE ATOMICITY DEFECT
- **Observation**:
  1. `task_verification_leases` lacked contract lineage foreign keys and anti-cross-pairing protections. CHECK constraints for `fencing_token > 0`, `expires_at_epoch_ms > acquired_at_epoch_ms`, and valid release states were missing.
  2. Overreaching claim of "zero partial SQLite records across the entire pipeline" contradicted multi-step failure recovery. The pipeline actually comprises three distinct durable transaction boundaries.
  3. Ownership of Schema v6 and v9 DDL vs pure in-memory execution in Subtasks P04B and P04C lacked clear separation.
  4. Proposal Rev 6 unilaterally altered canonical requirement NFR-008 to exclude external verification runtime without an approved change proposal, violating `docs/24_CHANGE_GOVERNANCE.md`.
- **Remediation Directives**:
  - Add contract lineage FK and anti-cross-pairing trigger to `task_verification_leases`. Add strict CHECK constraints on lease state timestamps.
  - Structure the pipeline into three distinct durable transactions:
    * Transaction A (Report Intake): validate report, persist claims, insert `attempt_workspace_bindings`, transition `RUNNING -> REPORT_READY`.
    * Transaction B (Evidence Finalization): persist evidence/policy/test/artifact metadata, release lease, transition `REPORT_READY -> EVIDENCE_READY`.
    * Transaction C (ReviewBundle Compilation): persist RFC 8785 JCS `ReviewBundle`, insert audit event, transition `EVIDENCE_READY -> REVIEWING`.
  - Document a comprehensive crash and replay recovery matrix for all three boundaries.
  - Replace overreaching pipeline-wide atomicity claims with per-transaction atomicity.
  - Separate DDL migration ownership (P04A owns Schema v6, P04D owns Schema v9) from runtime write authority (P04B and P04C are pure in-memory).
  - Do NOT modify canonical `docs/02_REQUIREMENTS.md`. Create `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (`PENDING_EXTERNAL_REVIEW`) to formally define latency measurement semantics ($T_0 	o T_1 \le 3	ext{s}$ with external verification bounded by attempt budgets). ADR-018 remains in DRAFT status pending proposal approval.

---

## 4. Remediation Actions Required for Revision 7

1. **`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`**: Update to Revision 7 addressing `P04-ARCH-R6-001` through `P04-ARCH-R6-006`.
2. **`docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`**: Update to Revision 7 aligning all 6 decisions with Revision 7 directives.
3. **`docs/plans/PLAN-P04-EVIDENCE-REVIEW.md`**: Update to Revision 7 unblocking P04A, removing blocked prerequisite edges, and detailing 3 transaction boundaries.
4. **`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`**: Update to Revision 5 (`HISTORICAL_DEFERRED_NON_BLOCKING`).
5. **`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`**: Update to `RETIRED_NON_EXECUTABLE_DRAFT` (status invariant: `NOT_RELEASED`).
6. **`docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md`**: Create new proposal for NFR-008 measurement semantics (`PENDING_EXTERNAL_REVIEW`).
7. **`AGENTS.md`** & **`docs/18_CURRENT_STATE.md`**: Synchronize governance state (`ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_6`).
8. Zero implementation code (`P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`, `P05_CODE = NOT_AUTHORIZED`).
