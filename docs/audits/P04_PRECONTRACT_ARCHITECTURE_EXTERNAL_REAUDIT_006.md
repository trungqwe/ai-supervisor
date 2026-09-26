# P04 Pre-Contract Architecture External Re-Audit 006

> **Audit Type**: External Supervisor Pre-Contract Architecture Re-Audit 006
> **Audited Commit**: `e7c57965ca8d85c0efa0b4f5b5dd409ce7387234`
> **Auditor**: External Supervisor
> **Date**: 2026-09-26
> **Scope**: Pre-Contract Architecture Documents for Phase P04 (Proposal P04-001 Rev 7, Proposal P04-002, Draft ADR-018 Rev 7, Plan Rev 7, Proof Plan Rev 5, Draft Proof Contract Model A)

---

## 1. Audit Verdict

```
PROPOSAL_P04_001 = REVISION_7_REQUIRED
PROPOSAL_P04_002 = REVISION_1_REQUIRED
ADR_018 = REVISION_7_REQUIRED
PLAN_P04_EVIDENCE_REVIEW = REVISION_7_REQUIRED
ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_7
P04_TASK_CONTRACT = NOT_RELEASED
P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT
P05_CODE = NOT_AUTHORIZED
AUTOMATIC_RESTORE = DISABLED
LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK
VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY
```

---

## 2. Status of Prior Findings (Round 6)

- **`P04-ARCH-R6-001`**: `PARTIALLY_CLOSED`
  `terminal_generation` was aligned to `TEXT` and nonexistent column references were eliminated, but registration boundary was improperly conflated with Report Intake (Transaction A) instead of dispatch materialization, session FK created an invalid dependency on mutable current pair sessions, and identity strings lacked strict 16-hex and 32-hex representations (`P04-ARCH-R7-001`).
- **`P04-ARCH-R6-002`**: `CLOSED`
  Decoupling of unexecutable proof contract from execution graph verified; Stage B terminology collision eliminated and worktree binding validation established as `WORKTREE_BINDING_RUNTIME_VALIDATION`; proof plan and contract successfully retired to historical non-blocking track. Subtask P04A is confirmed unblocked as the first releaseable task once architecture is approved.
- **`P04-ARCH-R6-003`**: `PARTIALLY_CLOSED`
  Hardened Git collection specified with host binary and config isolation, but Git arguments omitted required `-c core.fsmonitor=false`, `--no-pager` was miscategorized as an environment variable, environment sanitization missed `GIT_CONFIG_*` / credential variables, byte cap lacked process-tree streaming termination, and requirement traceability contained misattributions (`P04-ARCH-R7-004`).
- **`P04-ARCH-R6-004`**: `PARTIALLY_CLOSED`
  Window Station/Desktop relegated to UI isolation, but architectural choice retained an ambiguous "non-AppContainer token" fallback, Win32 AppContainer parameter structures were not detailed, snapshot creation protocol lacked verification against exact HEAD commits, and root handle sharing semantics were overclaimed (`P04-ARCH-R7-003`).
- **`P04-ARCH-R6-005`**: `PARTIALLY_CLOSED`
  Content-addressed artifact store and hex checks improved, but `review_artifacts` had an impossible FK reference to `review_bundles` created in a later transaction, canonical rename used `MOVEFILE_REPLACE_EXISTING`, structured `worker_claims` were omitted from DDL, and canonical relative path derivation was underspecified (`P04-ARCH-R7-002`).
- **`P04-ARCH-R6-006`**: `PARTIALLY_CLOSED`
  Three-boundary architecture established and `task_scoped_audit_events` was erroneously introduced, lease acquire/reclaim lacked exact atomic `BEGIN IMMEDIATE` CAS semantics, `verification_budget_ms_token` was an unapproved token, formatting artifacts corrupted text formulas, and bundle replay uniqueness was incomplete (`P04-ARCH-R7-005`, `P04-ARCH-R7-006`).

---

## 3. New Findings (Round 7)

### P04-ARCH-R7-001 — Binding Registration Boundary & Session FK Contradiction
- **Classification**: ARCHITECTURAL BOUNDARY & DATA INTEGRITY DEFECT
- **Observation**:
  1. Proposal Rev 7, ADR-018 Rev 7, and Plan Rev 7 claimed that Transaction A (Report Intake) inserts `attempt_workspace_bindings`. This is fundamentally incorrect: the physical worktree is materialized by AO during dispatch, and binding must be registered before task execution begins (`PrepareBoundDispatch` / `DISPATCH_BOUND` / prior to `RecordSendRequested`).
  2. Transaction A is an intake transaction occurring after worker completion; creating binding during intake allows unverified worker execution in an unbound worktree and creates TOCTOU vulnerabilities.
  3. `attempt_workspace_bindings.session_id` had a foreign key `REFERENCES worker_sessions(session_id)`. However, `worker_sessions` represents the mutable current session of a pair lane. If a worker session is killed, rotated, or replaced on the pair, the foreign key would block or be corrupted by session changes, violating attempt audit immutability.
  4. Physical identity was represented as unstructured strings, failing to enforce exact 16 lowercase hex characters (zero-padded) for volume serial number and 32 lowercase hex characters for 128-bit FileId.
- **Remediation Directives**:
  - Separate two distinct transactions:
    * **Dispatch Binding Registration Transaction**: Executes immediately upon AO worktree materialization, integrates into P03 seam `PrepareBoundDispatch` / `DISPATCH_BOUND`, and must complete prior to `RecordSendRequested`.
    * **Transaction A (Report Intake)**: Strictly reads and verifies that authoritative `attempt_workspace_bindings` exists and matches; persists `WorkerReport` / `worker_claims`; transitions `tasks.state`: `RUNNING -> REPORT_READY`. Creating bindings in Transaction A is strictly prohibited.
  - Remove FK on `session_id` to `worker_sessions(session_id)`. Reconcile `session_id` and `terminal_generation` against `task_attempts` and `dispatch_operations` in the registration trigger.
  - Standardize identity columns: `volume_serial_hex` (16 lowercase hex, zero-padded) and `file_id_hex` (32 lowercase hex, 16-byte `FILE_ID_128` order). Enforce CHECK constraints rejecting short, uppercase, or non-hex characters.
  - Explicitly document that future Subtask P04A scope includes necessary dispatch seams in `internal/dispatch` and `internal/store` (code implementation remains held).

### P04-ARCH-R7-002 — Durable Evidence Model & Inexecutable FK Order
- **Classification**: SCHEMA DEPENDENCY & RELATIONAL INTEGRITY DEFECT
- **Observation**:
  1. `review_artifacts.bundle_id` had a foreign key `REFERENCES review_bundles(bundle_id)`. However, Transaction B generates and persists artifact metadata before Transaction C creates the `review_bundles` row. This foreign key dependency is impossible to execute in sequential transactions.
  2. The proposal lacked authoritative DDL for structured `worker_claims` (leaving claims solely in raw report text) and durable `evidence_sets` encompassing Git evidence, test evidence, policy findings, and artifact references.
  3. Transaction C referenced nonexistent `task_scoped_audit_events` instead of the canonical `audit_events` table established in Phase P02.
  4. Replay of Transaction C lacked a `UNIQUE(attempt_id)` constraint on `review_bundles`, risking duplicate bundles per attempt. Idempotent replay for identical hashes vs fail-closed rejection of divergent hashes was underspecified.
  5. The durability protocol used `MOVEFILE_REPLACE_EXISTING` on canonical paths and ungrounded "quarantine" terminology without schema backing.
- **Remediation Directives**:
  - Restructure relational schema:
    * Transaction B creates durable `evidence_sets` and inserts `review_artifacts` referencing `evidence_set_id`.
    * Transaction C creates `review_bundles` referencing the same `evidence_set_id`.
  - Add explicit DDL for `worker_claims` (Schema v6) and `evidence_sets` (Schema v9).
  - Transaction B must prove durable persistence of evidence before transitioning `REPORT_READY -> EVIDENCE_READY`.
  - Transaction C must insert audit records into canonical `audit_events`. Eradicate all references to `task_scoped_audit_events`.
  - Enforce `UNIQUE(attempt_id)` on `review_bundles`. Idempotent replay succeeds if canonical hash matches; conflicting hash fails closed.
  - Enforce deterministic derivation of `canonical_relative_path` from content hash with anti-traversal CHECK constraints.
  - Remove `MOVEFILE_REPLACE_EXISTING` on canonical artifact paths. If destination exists, reopen, verify size and hash, and deduplicate; mismatch fails closed. Remove ungrounded quarantine terminology.

### P04-ARCH-R7-003 — Security Boundary & Snapshot Protocol Underspecified
- **Classification**: SECURITY ARCHITECTURE & ISOLATION DEFECT
- **Observation**:
  1. Draft ADR-018 and Proposal Rev 7 retained "restricted primary token" as an alternative or fallback to AppContainer. A dual path complicates validation and dilutes security guarantees.
  2. Win32 AppContainer parameter setup was described abstractly without detailing `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES`, `SECURITY_CAPABILITIES`, and attribute list structures.
  3. The snapshot creation mechanism was vague ("clones or copy") and lacked verification against exact commit trees, risking TOCTOU contamination.
  4. Root worktree handles omitted file sharing flags and used informal "absolute safety" assertions.
- **Remediation Directives**:
  - Commit exclusively to **Windows AppContainer** for v1. Eliminate all mentions of non-AppContainer tokens as an alternative or fallback.
  - Fully specify AppContainer creation: `CreateAppContainerProfile` / `DeriveAppContainerSidFromAppContainerName`, `SECURITY_CAPABILITIES` with zero capabilities, `InitializeProcThreadAttributeList`, `UpdateProcThreadAttribute` with `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES`, `STARTUPINFOEX`, Job Object (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, no breakaway), `bInheritHandles = FALSE`.
  - Specify deterministic snapshot protocol: export verified HEAD commit tree using pinned Git `ls-tree -rz` and `cat-file --batch`; validate all path components; reject path traversal (`..`), junctions, symlinks, and gitlinks in v1; construct and hash an immutable manifest. Never read active worktree files during test execution.
  - Open worktree root handles strictly without `FILE_SHARE_DELETE` and hold throughout verification and CAS. Replace "absolute safety" claims with testable invariants.

### P04-ARCH-R7-004 — Git Authority & Requirement Traceability Misattributions
- **Classification**: COMMAND AUTHORITY & GOVERNANCE TRACEABILITY DEFECT
- **Observation**:
  1. Git command invocations omitted `-c core.fsmonitor=false`.
  2. `--no-pager` was incorrectly listed as an environment variable rather than a command-line argument.
  3. Environment sanitization missed critical Git configuration overrides (`GIT_CONFIG_COUNT`, `GIT_CONFIG_KEY_*`, `GIT_CONFIG_VALUE_*`, `GIT_EXTERNAL_DIFF`, `GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_SSH`, `GIT_SSH_COMMAND`, `GIT_PROTOCOL_FROM_USER`, `GIT_CEILING_DIRECTORIES`).
  4. Byte cap enforcement failed to mandate immediate process-tree termination upon cap breach.
  5. Index identity check lacked precise definition from repository index files.
  6. Traceability misattributed requirements: FR-009 was misdescribed as test execution (canonical FR-009 is Scope Comparison & Review Bundle Generation); NFR-005 was misdescribed as resource isolation (canonical NFR-005 is Loose Coupling & Upstream Adapter Separation); NFR-007 was misdescribed as audit trail (canonical NFR-007 is Pinned Upstream Dependencies).
- **Remediation Directives**:
  - Mandate `-c core.hooksPath=<trusted_empty_hooks>` and `-c core.fsmonitor=false` on all Git CLI calls.
  - Pass `--no-pager` strictly as a CLI flag.
  - Expand environment sanitization to eliminate all `GIT_CONFIG_*`, credential helper, SSH, protocol, and ceiling variables.
  - Require streaming stdout/stderr byte cap enforcement with immediate process-tree termination.
  - Define index hash as the SHA-1/SHA-256 hash of `.git/index` under verified repository identity.
  - Correct traceability mappings:
    * Independent Git/test evidence -> `FR-008` (Audit-Grade Evidence Collection);
    * Scope comparison -> `FR-009` and `SEC-005`;
    * Append-only audit logging -> `FR-013` and `NFR-004`;
    * Binary and toolchain pinning -> `NFR-007`;
    * Upstream adapter separation -> `NFR-005`;
    * AppContainer / Job Object sandboxing -> `SEC-001`, `SEC-003`, `OPS-003`.

### P04-ARCH-R7-005 — Lease Acquire/Reclaim & Bundle Replay Guarantees Incomplete
- **Classification**: CONCURRENCY, FENCING & STATE CAS DEFECT
- **Observation**:
  1. Proposal and ADR described lease acquire/reclaim without explicit `BEGIN IMMEDIATE` transaction predicates, initial token insertion, or RowsAffected assertions.
  2. Fencing tokens were not passed to collectors or verified at Transaction B commit.
  3. Memory-based lease expiration terminology ("transient memory lease expiration") lingered, contradicting durable database authority.
  4. Transaction C lacked transactional co-commit of state CAS, `audit_events`, and `review_bundles`.
- **Remediation Directives**:
  - Detail `BEGIN IMMEDIATE` lease acquisition protocol: insert ACTIVE token 1 if row absent; reject if ACTIVE and unexpired; reclaim expired/released via CAS incrementing `fencing_token = fencing_token + 1` and asserting `RowsAffected == 1`.
  - Thread fencing token through collectors; Transaction B commits only if lease is ACTIVE, unexpired, and token matches.
  - State clearly that lease expiration is evaluated strictly from durable SQLite row timestamps (`expires_at_epoch_ms <= now_ms`).
  - Transaction C atomically commits state transition `EVIDENCE_READY -> REVIEWING`, `review_bundles` insert, and `audit_events` insert. Clarify that crash between B and C does not re-run tests, and bundle uniqueness enforces a single winning authority.

### P04-ARCH-R7-006 — NFR-008 Unapproved Budget Token & Document Formatting Corruption
- **Classification**: REQUIREMENT GOVERNANCE & DOCUMENT HYGIENE DEFECT
- **Observation**:
  1. Proposal Rev 7, ADR-018 Rev 7, and Plan Rev 7 introduced `verification_budget_ms_token` with a default of 60,000 ms. This token does not exist in canonical TaskContracts, ADR-013, or `VerificationPolicyCatalog`.
  2. Interval 1 was not grounded in validated task contract parameters.
  3. `REVIEW_BUNDLE_COMPILED` was an unapproved audit event type.
  4. Documents contained TAB characters and corrupted control bytes in mathematical notation (`\text`, `\to`).
- **Remediation Directives**:
  - Eradicate `verification_budget_ms_token` and 60,000 ms defaults.
  - Derive Interval 1 timeout strictly from `verification_requests[].timeout_seconds` in the validated TaskContract, or fallback to `MaxTimeoutSeconds` from the approved `VerificationPolicyCatalog` profile. Specify sequential (sum) or parallel (max) aggregation rules.
  - Define T0 as `evidence_committed_at` (Transaction B commit) and T1 as Transaction C commit. Enforce $0 \le T_1 - T_0 \le 3000	ext{ ms}$; clock regression fails closed. Use monotonic clocks for in-process measurements.
  - Use canonical audit event types from Phase P02 (`AUDIT_EVENT_BUNDLE_GENERATED`).
  - Eliminate all TAB characters and control bytes; format formulas as plain text ("T1 - T0 <= 3 seconds").
  - Retain `PROPOSAL-P04-002` in `PENDING_EXTERNAL_REVIEW`; do not alter `docs/02_REQUIREMENTS.md`.

---

## 4. Remediation Directives Summary for Revision 8

1. Update `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` to Revision 8.
2. Update `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` to Revision 2.
3. Update `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` to Revision 8.
4. Update `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` to Revision 8.
5. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_7`.
6. Zero implementation code (`P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`, `P05_CODE = NOT_AUTHORIZED`).
