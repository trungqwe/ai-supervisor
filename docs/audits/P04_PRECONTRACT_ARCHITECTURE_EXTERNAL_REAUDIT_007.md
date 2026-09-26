# AUDIT RECORD: P04 Pre-Contract Architecture External Re-Audit 007

> **Audit Type**: External Supervisor Pre-Contract Architecture Re-Audit
> **Audited Baseline**: `a6290ce4c444e345e47aace6c488c3cc4ff33a0d`
> **Date**: 2026-09-26
> **Auditor**: External Supervisor
> **Target Scope**: Phase P04 Pre-Contract Architecture Remediation Round 7 Deliverables:
> - `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 8)
> - `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 2)
> - `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 8)
> - `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 8)
> - `AGENTS.md`
> - `docs/18_CURRENT_STATE.md`

---

## 1. Executive Summary & Verdict

The External Supervisor audited the remediation deliverables submitted for Pre-Contract Architecture Round 7 at commit `a6290ce4c444e345e47aace6c488c3cc4ff33a0d`.

The remediation resolved the registration boundary and mutable session foreign key coupling (`P04-ARCH-R7-001 = CLOSED_AT_DESIGN_LEVEL`), but five critical architectural areas require further remediation:
1. `worker_claims` schema cardinality violated canonical 1-to-1 attempt model; `review_artifacts` paths included arbitrary file extensions; latency timestamps and intervals lacked durable DDL constraints (`P04-ARCH-R8-001`).
2. AppContainer process I/O specification claiming `bInheritHandles = FALSE` contradicted anonymous pipe standard handle redirection; Job Object containment lacked atomic assignment at creation time on Windows 10+ (`P04-ARCH-R8-002`).
3. Snapshot creation timing was improperly positioned before Transaction A report intake; snapshot authority improperly relied on reported head; Git allowlist omitted necessary commands; linked-worktree index path resolution was flawed (`P04-ARCH-R8-003`).
4. Lease admission lacked validation of state, current attempt, and existing evidence; concurrency assumption contradicted v1 sequential model; fencing token isolation in filesystem was incomplete (`P04-ARCH-R8-004`).
5. Audit event literal `AUDIT_EVENT_BUNDLE_GENERATED` was falsely claimed as canonical from Phase P02; failure audit semantics conflated transaction rollback with same-transaction event recording (`P04-ARCH-R8-005`).

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_8_REQUIRED`
- `PROPOSAL_P04_002 = REVISION_2_REQUIRED`
- `ADR_018 = REVISION_8_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_8_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_8`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 7 Findings (P04-ARCH-R7-001 through R7-006)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R7-001` | Worktree Registration Boundary & Session FK Coupling | **CLOSED_AT_DESIGN_LEVEL** | Dispatch seam decoupled from Transaction A; mutable session foreign key removed; 16/32 hex strings verified. |
| `P04-ARCH-R7-002` | Durable Evidence Model & Relational Hierarchy | **PARTIALLY_CLOSED** | Inverted relational hierarchy accepted, but `worker_claims` cardinality and artifact path format require remediation (`P04-ARCH-R8-001`). |
| `P04-ARCH-R7-003` | Windows AppContainer & Snapshot Protocol | **PARTIALLY_CLOSED** | AppContainer committed, but process I/O handle inheritance and Job Object containment require remediation (`P04-ARCH-R8-002`, `R8-003`). |
| `P04-ARCH-R7-004` | Git Authority & Traceability | **PARTIALLY_CLOSED** | Flag isolation accepted, but snapshot timing, Git allowlist commands, and linked-worktree index path require remediation (`P04-ARCH-R8-003`). |
| `P04-ARCH-R7-005` | Lease Acquire/Reclaim & Bundle Replay | **PARTIALLY_CLOSED** | Atomic CAS accepted, but admission prerequisites, sequential execution, and sandbox directory fencing require remediation (`P04-ARCH-R8-004`). |
| `P04-ARCH-R7-006` | NFR-008 Token Governance & Typography | **PARTIALLY_CLOSED** | Arbitrary tokens eliminated, but event literal governance and latency DDL constraints require remediation (`P04-ARCH-R8-001`, `R8-005`). |

---

## 3. New Findings (P04-ARCH-R8-001 through P04-ARCH-R8-005)

### `P04-ARCH-R8-001`: WorkerClaim Cardinality, Artifact Path Format & Latency DDL Mismatch
- **Severity**: High
- **Description**:
  1. `worker_claims` DDL implemented a multi-row per `claim_type` model, contradicting the canonical domain model (`docs/05_DOMAIN_MODEL.md`) and schema (`docs/schemas/worker-report.schema.json`) where each `TaskAttempt` has exactly one `WorkerClaim` containing reported head, claimed changed files, claimed test results, and implementation claims.
  2. `review_artifacts.canonical_relative_path` permitted arbitrary file extensions (`.txt`, `.diff`), violating strict content-addressed hashing and deterministic path derivation.
  3. `evidence_sets` and `review_bundles` lacked durable epoch timestamps enforcing the NFR-008 latency invariant ($0 \le T_1 - T_0 \le 3000 ext{ ms}$) at the database layer.
- **Remediation Directive**:
  1. Enforce unique cardinality: `UNIQUE(attempt_id)` on `worker_claims`, storing structured JCS JSON for reported head, changed files, test results, and claims payload.
  2. Enforce exact content-addressed relative path: `canonical_relative_path = 'artifacts/' || substr(captured_sha256,1,2) || '/' || captured_sha256` without extensions. Reject colons, backslashes, path traversal, extra segments, and hash mismatches.
  3. Add `evidence_committed_at_epoch_ms` to `evidence_sets`; add `bundle_committed_at_epoch_ms` and `compilation_latency_ms` to `review_bundles` with check constraints enforcing $0 \le  ext{compilation\_latency\_ms} \le 3000 ext{ ms}$ and $ ext{bundle\_committed} \ge  ext{evidence\_committed}$.

### `P04-ARCH-R8-002`: AppContainer Process I/O & Job Containment Contradiction
- **Severity**: High
- **Description**:
  1. ADR-018 and Plan claimed `bInheritHandles = FALSE` while simultaneously requiring anonymous pipes for stdout and stderr redirection. On Windows, child processes cannot inherit pipe handles if handle inheritance is globally disabled without explicit attribute list inheritance.
  2. Creation flag `CREATE_BREAKAWAY_FROM_JOB` was specified alongside subsequent `AssignProcessToJobObject`, leaving a race window where the sandboxed process runs outside the Job Object.
- **Remediation Directive**:
  1. Configure `STARTF_USESTDHANDLES`, pass pipe write ends in `STARTUPINFOEXW`, set `bInheritHandles = TRUE`, and specify `PROC_THREAD_ATTRIBUTE_HANDLE_LIST` containing strictly the designated standard handles.
  2. Ensure parent pipe ends and all other supervisor handles are created without `HANDLE_FLAG_INHERIT`.
  3. Use `PROC_THREAD_ATTRIBUTE_JOB_LIST` to associate the Job Object atomically at process creation on Windows 10+; remove `CREATE_BREAKAWAY_FROM_JOB`.
  4. Attribute list must contain at minimum `SECURITY_CAPABILITIES`, `HANDLE_LIST`, and `JOB_LIST`.
  5. Replace `GENERIC_ALL` on sandbox directory with minimal necessary rights (`FILE_GENERIC_READ | FILE_GENERIC_WRITE | FILE_GENERIC_EXECUTE | DELETE`).
  6. Restrict AppContainer profile moniker to valid Win32 characters, length $\le 64$, derived from attempt ID hash, with full lifecycle management.

### `P04-ARCH-R8-003`: Snapshot Timing, Git Allowlist & Linked-Worktree Index Flaws
- **Severity**: High
- **Description**:
  1. Documents stated the snapshot was created "at dispatch intake completion" prior to worker execution, contradicting the requirement that verification tests must run against the post-execution worktree state.
  2. The snapshot protocol lacked verification of actual verified HEAD against reported claims.
  3. Git allowlist omitted necessary commands for full-tree tree-listing and batch object extraction.
  4. Index hashing assumed literal `.git/index`, which fails for Git linked worktrees where `.git` is a file pointing to a shared gitdir.
- **Remediation Directive**:
  1. Specify snapshot creation strictly after Transaction A report intake completes (Task in `REPORT_READY`, binding revalidated).
  2. Snapshot must be sourced from actual verified HEAD commit, never worker claims.
  3. Expand Git allowlist to explicitly include `git ls-tree -rz --full-tree <actual_head_sha>`, `git cat-file --batch`, `git rev-parse --git-path index`, and `git rev-parse --absolute-git-dir`.
  4. Resolve index path using `git rev-parse --git-path index`, canonicalize and verify gitdir ownership before hashing.
  5. Enforce caps: max 10,000 files, per-blob byte cap, total snapshot cap, streaming extraction, disk quota, and fail-closed cleanup.
  6. Document rejection of symlinks, gitlinks, and reparse points as a v1 fail-closed stop condition.

### `P04-ARCH-R8-004`: Incomplete Lease Admission Prerequisites & Concurrency Overreach
- **Severity**: Medium
- **Description**:
  1. Lease acquisition in `BEGIN IMMEDIATE` did not check whether the task was in `REPORT_READY`, whether the attempt was current, or whether an `evidence_set` already existed.
  2. Lease TTL derivation mentioned concurrent request aggregation ("max for concurrent"), but TaskContracts and v1 execution contain no concurrency semantics.
  3. Fencing tokens were not reflected in sandbox directory paths, creating collision risks for zombie processes.
- **Remediation Directive**:
  1. Enforce strict admission checks in `BEGIN IMMEDIATE`: task state must be `REPORT_READY`, attempt must be `tasks.current_attempt`, lineage must be verified, binding must be revalidated, `worker_claims` must exist, and no `evidence_sets` may exist for the attempt.
  2. If task is already `EVIDENCE_READY` or `REVIEWING`, do not re-run collectors; read committed authority.
  3. Restrict v1 verification requests strictly to sequential execution; sum effective timeouts.
  4. Deterministically derive lease TTL from request timeouts, Git collector timeout (10s), and bounded overhead (15s).
  5. Isolate sandbox and staging directories using fencing token: `sandboxes/<attempt_id>/<fencing_token>/`.
  6. Transaction B must re-verify token, `ACTIVE` state, and current attempt immediately prior to commit.
  7. Terminate and join old Job Object upon lease reclaim.

### `P04-ARCH-R8-005`: Audit Event Governance & Transaction Rollback Conflation
- **Severity**: Medium
- **Description**:
  1. Deliverables referred to `AUDIT_EVENT_BUNDLE_GENERATED` as a "canonical event from P02", but this literal does not exist in domain constants or canonical event catalogs.
  2. Documents suggested that clock regression or hash conflict transactions should record audit events within the failing transaction, which is impossible if the transaction rolls back.
- **Remediation Directive**:
  1. Eradicate false claims of canonical approval. Introduce proposed audit events with status `PROPOSED_UNTIL_ADR_ACCEPTANCE`: `REVIEW_BUNDLE_GENERATED` and `REVIEW_BUNDLE_COMPILATION_REJECTED`.
  2. Specify two-phase failure audit semantics: transaction rolls back cleanly; a separate fail-closed transaction records `REVIEW_BUNDLE_COMPILATION_REJECTED` without altering TaskState.
  3. Defer formal canonical event catalog reconciliation until ADR-018 acceptance.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 9.
2. Remediate `PROPOSAL-P04-002` to Revision 3.
3. Remediate `DRAFT-ADR-018` to Revision 9.
4. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 9.
5. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_8`.
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
