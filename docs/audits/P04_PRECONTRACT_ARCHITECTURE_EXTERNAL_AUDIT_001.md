# Phase P04 Pre-Contract Architecture — External Audit 001

- **Audited Target:** Pre-Contract Architecture Planning for Phase P04 (Evidence & Review Engine)
- **Audited Baseline Commit:** `08cc9c3e4f24404cc57a5c949d556959b710c38a`
- **Audited Artifacts:**
  - Proposal: [`docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`](../proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md)
  - Draft ADR: [`docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`](../adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md)
  - Execution Plan: [`docs/plans/PLAN-P04-EVIDENCE-REVIEW.md`](../plans/PLAN-P04-EVIDENCE-REVIEW.md)
- **Authority:** External Supervisor Audit
- **Diff Hygiene & Baseline Whitelist Verification:** PASS (commit `08cc9c3` adhered strictly to 5-file whitelist with zero Go code mutation and zero whitespace errors).
- **External Supervisor Verdict:**
  - `PROPOSAL_P04_001 = REVISION_1_REQUIRED`
  - `ADR_018 = REVISION_1_REQUIRED`
  - `PLAN_P04_EVIDENCE_REVIEW = REVISION_1_REQUIRED`
- **Active Gate:** `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION`

---

## 1. Audit Summary & Findings Matrix

While baseline commit `08cc9c3` properly refrained from unauthorized code writing and observed process hygiene, substantive technical evaluation identified six architectural defects requiring remediation prior to any Task Contract release.

| Finding ID | Severity | Category | Summary & Core Directive | Status |
| :--- | :--- | :--- | :--- | :--- |
| **`P04-ARCH-R1-001`** | **BLOCKER** | Architecture Authority | **WORKTREE_AUTHORITY_UNPROVEN**: Host cannot unilaterally resolve AO session worktrees without an established authority source. Pinned AO (`v0.13.0`) does not expose `worktree_path` via its public session API; `Project.RootPath` does not prove session worktree. Decision 1 in ADR-018 must remain `CONDITIONAL / UNRESOLVED`. File scope and seams must include `internal/dispatch/**` and Store dispatch APIs. | **OPEN** |
| **`P04-ARCH-R1-002`** | **BLOCKER** | Security & Sandboxing | **VERIFICATION_ISOLATION_AMBIGUITY**: AppContainer and Restricted Token cannot be treated as equivalent; standalone Restricted Token lacks network isolation. Designate Windows AppContainer as primary; verification must never run with direct write permissions on audited worktree (use disposable copy/snapshot if mutation needed); enforce child desktop/window station isolation; add 5-vector falsification matrix. Keep `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`. | **OPEN** |
| **`P04-ARCH-R1-003`** | **CRITICAL** | Store & Concurrency | **MIGRATION_CAS_AND_ATOMICITY_DEFECTS**: Ambiguous multi-table "Schema v6" rejected; adopt sequential migrations (v6..v9) across subtasks P04A..P04D. Atomic ingestion must enforce full CAS on task state, attempt open, and session/generation lineage (`affected_rows == 1`). `failure_reason` is strictly an audit/event attribute, not a `task_attempts` column. Delineate external execution outside SQLite transactions with CAS persistence. | **OPEN** |
| **`P04-ARCH-R1-004`** | **CRITICAL** | Review & Cognitive Load | **REVIEWABLE_EVIDENCE_PAYLOAD_DEFICIENCY**: ReviewBundle must provide inspectable content for ChatGPT audit, not just hashes. Add bounded Git patch artifact (inline or opaque artifact ID), bounded test stdout/stderr, strictly typed `task_contract`, non-circular `bundle_hash` definition, and unify field name strictly to `test_results` (eliminate `details` drift). Define P05 bounded artifact retrieval without raw filesystem access. | **OPEN** |
| **`P04-ARCH-R1-005`** | **CRITICAL** | Git Authority & Security | **GIT_EVIDENCE_SECURITY_AND_ANCESTRY_IMPRECISION**: Replace inaccurate config hardening with explicit CLI flags (`--no-ext-diff --no-textconv --ignore-submodules=none`); strip `GIT_EXTERNAL_DIFF`, `GIT_DIFF_OPTS`, and askpass variables; rename ancestry rule to "strict descendant policy"; verify zero merge commits if linear history is required; re-verify HEAD and status after diff to prevent TOCTOU drift; pin trusted `git.exe` binary with hash validation. Add `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY = OPEN`. | **OPEN** |
| **`P04-ARCH-R1-006`** | **MAJOR** | Governance & Provenance | **PROVENANCE_AND_RECOVERY_SCENARIO_OVERCLAIM**: Fix incorrect code reference `internal/adapter/ao/session.go` to `internal/ao/client.go`. Explicitly cite `SOURCE_REGISTRY.md`, `REUSE_MATRIX.md`, and `22_MODULE_PROVENANCE.md`. Clarify reuse of AO workspace read and Git worktree isolation; prohibit custom unapproved worktree managers. Synchronize 5 open design blockers across all documents. Rename recovery scenarios to "13 proposed P04 recovery scenarios" until formally approved. | **OPEN** |

---

## 2. Detailed Findings Description & Required Remediation

### 2.1. P04-ARCH-R1-001: WORKTREE_AUTHORITY_UNPROVEN
- **Observation:** Proposal P04-001 stated that the host binds `AttemptWorkspaceBinding` prior to dispatch, but failed to identify the canonical authority source from which the host obtains the exact worktree path. In pinned upstream AO (`v0.13.0`), the public REST session model (`GET /api/v1/sessions/{id}`) does not return `worktree_path`. Assuming `Project.RootPath` is the worktree path contradicts multi-worktree patterns and leaves concurrent attempts unisolated.
- **Required Remediation:**
  1. Contrast against `SOURCE_REGISTRY.md`, `REUSE_MATRIX.md`, and pinned AO upstream code (`backend/internal/adapters/agent/agy/agy.go`).
  2. Analyze feasibility of querying `git worktree list --porcelain -z` and matching branch/session metadata, including collision handling, restore/restart behavior, and TOCTOU checks.
  3. Because public API support is absent in pinned AO, mark Decision 1 in ADR-018 as **CONDITIONAL / UNRESOLVED** until proven via bounded upstream runtime proof.
  4. Expand P04A scope to include dispatch/provisioning integration seams (`internal/ao/client.go`, `internal/store/**`, `internal/dispatch/**`).

### 2.2. P04-ARCH-R1-002: VERIFICATION_ISOLATION_AMBIGUITY
- **Observation:** DRAFT-ADR-018 loosely treated Windows AppContainer and Restricted Token as interchangeable options and failed to recognize that a Restricted Token provides zero network isolation by default. Furthermore, the design allowed verification tests to execute with write access directly inside the audited worktree, enabling untrusted test scripts to pollute Git status, modify files, or hide malicious code.
- **Required Remediation:**
  1. Select **Windows AppContainer Isolation** as the primary Layer B mechanism. Designate Restricted Token with Low Integrity Level strictly as a secondary fallback requiring external firewalling.
  2. Mandate that the audited worktree is **STRICTLY READ-ONLY** during verification (`GENERIC_READ`, deny write/delete). If a verification tool requires writing to the source tree, it must execute against a **disposable copy / snapshot directory** with verified provenance.
  3. Implement child desktop / window station isolation (`CreateDesktop`) to prevent UI message hooking.
  4. Include a 5-vector falsification matrix (filesystem escape, network egress, secret access, child process escape, source mutation).
  5. Keep `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`.

### 2.3. P04-ARCH-R1-003: MIGRATION_CAS_AND_ATOMICITY_DEFECTS
- **Observation:** PROPOSAL-P04-001 proposed creating six tables all designated under "Schema v6", while decomposing P04 into four sequential subtasks, creating ambiguity over migration ownership. In addition, the CAS logic in the ingestion transaction omitted task and lineage guards, and `failure_reason` was incorrectly described as a column in `task_attempts`.
- **Required Remediation:**
  1. Adopt **Sequential Schema Migrations**: Schema v6 (P04A: `worker_claims`, `attempt_workspace_bindings`), Schema v7 (P04B: `evidence`, `policy_findings`), Schema v8 (P04C: `actual_test_results`), Schema v9 (P04D: `review_bundles`).
  2. Formulate complete CAS query for ingestion (`state = 'RUNNING'`, `current_attempt` matching, `ended_at IS NULL`, exact lineage, `affected_rows == 1`).
  3. Reaffirm that `failure_reason` is strictly an audit/disposition attribute, not a schema column.
  4. Document that external Git and test executions occur outside short SQLite transactions; persistence relies on immutable snapshots and CAS.

### 2.4. P04-ARCH-R1-004: REVIEWABLE_EVIDENCE_PAYLOAD_DEFICIENCY
- **Observation:** ReviewBundle in `PROPOSAL-P04-001` only aggregated metadata hashes, leaving ChatGPT unable to review the actual code diff or test output without making low-level tool calls. Furthermore, `bundle_hash` had a circular dependency on the bundle itself, and field naming drifted between `test_results` and `details`.
- **Required Remediation:**
  1. Add bounded Git patch artifact: inline patch (up to bounded limit) or immutable local artifact ID, SHA-256, byte count, `is_truncated` flag, binary metadata.
  2. Add bounded test output logs: stdout and stderr text logs (or artifact ID), SHA-256 hash, byte count, truncation flag.
  3. Strictly unify field name to `test_results`; eliminate `details`.
  4. Define canonical serialization where `bundle_hash` is computed over all fields *excluding* `bundle_hash`.
  5. Define bounded local artifact storage service for Phase P05 retrieval without opening raw filesystem access.

### 2.5. P04-ARCH-R1-005: GIT_EVIDENCE_SECURITY_AND_ANCESTRY_IMPRECISION
- **Observation:** Git command hardening relied on incomplete config flags without CLI overrides (`--no-ext-diff --no-textconv`), failed to neutralize `GIT_EXTERNAL_DIFF` and `GIT_DIFF_OPTS`, and used the term "strict linear ancestry" without checking for merge commits. Furthermore, the design lacked protection against HEAD shifting during multi-step collection.
- **Required Remediation:**
  1. Use explicit CLI flags: `--no-ext-diff --no-textconv --ignore-submodules=none`.
  2. Strip all external diff and askpass environment variables (`GIT_ASKPASS`, `SSH_ASKPASS`, `GIT_TERMINAL_PROMPT=0`, `LC_ALL=C`).
  3. Terminology: Rename to **Strict Descendant Policy** (`git merge-base --is-ancestor`). If linear history is required, enforce zero merge commits (`git rev-list --merges <base>..<head>`).
  4. Implement before/after HEAD and status re-verification to defeat TOCTOU worktree race conditions.
  5. Pin absolute trusted `git.exe` with cryptographic hash validation.
  6. Add `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY = OPEN`.

### 2.6. P04-ARCH-R1-006: PROVENANCE_AND_RECOVERY_SCENARIO_OVERCLAIM
- **Observation:** Proposal cited `internal/adapter/ao/session.go` which does not exist in the codebase (the real implementation is `internal/ao/client.go`). Proposal also claimed "13 canonical recovery scenarios" before any supervisor approval, and failed to cross-reference `SOURCE_REGISTRY.md` and `REUSE_MATRIX.md`.
- **Required Remediation:**
  1. Correct code reference to `internal/ao/client.go`.
  2. Directly cite `docs/sources/SOURCE_REGISTRY.md`, `docs/sources/REUSE_MATRIX.md`, and `docs/22_MODULE_PROVENANCE.md`.
  3. Synchronize all 5 open design blockers across all documents.
  4. Rename recovery scenarios to "13 proposed P04 recovery scenarios".

---

## 3. Governance Directives & Active Gate

1. **Active Gate:** `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION`.
2. **Blocker Status:**
   - `DESIGN_BLOCKER_P04_WORKTREE_BINDING = OPEN`
   - `DESIGN_BLOCKER_P04_GIT_EVIDENCE_AUTHORITY = OPEN`
   - `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`
   - `DESIGN_BLOCKER_P04_REVIEW_SCHEMA_RECONCILIATION = OPEN`
   - `DESIGN_BLOCKER_P04_EVIDENCE_ATOMICITY = OPEN`
3. **Execution Guardrails:**
   - `P04_TASK_CONTRACT = NOT_RELEASED`
   - `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
   - `P05_CODE = NOT_AUTHORIZED`
   - `AUTOMATIC_RESTORE = DISABLED`
   - `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`
   - `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`
