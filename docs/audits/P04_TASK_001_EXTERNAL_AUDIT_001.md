# P04_TASK_001_EXTERNAL_AUDIT_001.md — External Audit 001: Subtask P04A Implementation

> **Target**: Subtask P04A Implementation (`CONTRACT-TASK-P04-001-01`)
> **Contract Release Commit**: `dda382dee2c64456dc281768275082be73107056`
> **Implementation Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
> **Audited Implementation SHA**: `ffde8c6ff903cfa49b8dab2db18dc70216752b5d`
> **Branch**: `codex/p04-001`
> **Authority**: External Supervisor
> **Verdict**: `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED`
> **Date**: 2026-09-28

---

## 1. Executive Summary

An external audit was conducted on the candidate implementation of Subtask P04A (Workspace Binding Authority, Schema Migration v6, Seam Extension & Report Intake) submitted on branch `codex/p04-001` at commit `ffde8c6ff903cfa49b8dab2db18dc70216752b5d`.

### Independent Verification Results
1. **Test Suite**: `go test -race -count=1 ./...` exited `0` across all packages.
2. **Scope Compliance**: Diff contains 25 files strictly within the 39 whitelisted files in `allowed_scope`.
3. **Format Integrity**: `git diff --check` exited `0`.

### Audit Evaluation
While the suite passes, the audit identified critical structural defects and bypasses in the implementation that compromise architectural guarantees established in accepted ADR-018 and the task contract. Consequently, the implementation is **NOT APPROVED** and authority to merge into `main` is **WITHHELD**.

---

## 2. Findings Log

### Finding P04A-I1-001: Release Artifact Integrity / Provenance Discrepancy
- **Severity**: HIGH (Governance / Provenance Integrity)
- **Status**: OPEN (Documented & Rectified via Erratum 001)
- **Description**:
  1. `docs/tasks/TASK_CONTRACT_P04_001.md` at commit `dda382dee2c64456dc281768275082be73107056` has actual Git blob SHA `aebcdaf942410f24655d81c143a543683079e4c4`. The release audit recorded blob `1f5e31774705470beb914a3c3c75d95d5bc2d845`.
  2. The released file wrapper contains 3 ASCII Bell (`BEL`, `0x07`) bytes at offsets 1317, 1942, and 35335, and typographical artifact `"ull"`.
  3. The raw JSON block Git blob SHA `7344a11cf22c0262a52fe598c938c532d56f5b8c` remains byte-for-byte identical to candidate JSON block.
- **Remediation Requirement**:
  - Do not edit historical release audit or contract in place.
  - Formal append-only erratum `docs/audits/P04_TASK_001_CONTRACT_RELEASE_AUDIT_ERRATUM_001.md` records actual blob SHA, wrapper defects, and confirms raw JSON block integrity.

### Finding P04A-I1-002: Snapshot-less RecordSendRequested Bypass
- **Severity**: CRITICAL (Safety / Effect Gate Bypass)
- **Status**: OPEN
- **Description**: 41 out of 45 callers of `RecordSendRequested` in existing tests omitted the snapshot parameter because the API was declared variadic (`snapshots ...domain.WorkspaceBindingSnapshot`). In the absence of an explicit snapshot, the store fallback copied the durable binding from SQLite, causing physical identity guards 12–14 to always pass vacuously.
- **Remediation Requirement**:
  - Change `RecordSendRequested` signature to strictly require exactly one non-variadic `snapshot domain.WorkspaceBindingSnapshot`.
  - Completely eliminate store fallback that copies snapshot from durable binding.
  - Zero/stale/mismatched snapshot must fail closed before `SEND_REQUESTED` is committed.
  - Update 100% of callers in `allowed_scope`. Verify via `rg` that zero snapshot-less callers exist.

### Finding P04A-I1-003: Fabricated Workspace Identity / Wrong AO Pin
- **Severity**: CRITICAL (Data Integrity / Architecture Violation)
- **Status**: OPEN
- **Description**:
  1. `PrepareBoundDispatch`, `Coordinator.Dispatch`, and `MemoryWorkspaceBindingAuthority` contained fallback logic generating synthetic worktree path `C:\repo\worktree`, synthetic VolumeSerial/FileID, and 40-character zero hash when snapshots were missing.
  2. `Coordinator.Dispatch` used `TaskContract.BaseSHA` as `pinned_ao_commit`. `BaseSHA` is the repository base checkout commit, NOT the AO upstream commit.
  3. Linked gitdir was derived via simple `filepath.Join(worktree, ".git")`, failing to account for Git worktrees where `.git` is a pointer file (`gitdir: <path>`).
- **Remediation Requirement**:
  - Remove all synthetic fallbacks (`C:\repo\worktree`, synthetic FileID/VolumeSerial, zero AO commit).
  - `PrepareBoundDispatch` must strictly validate full, valid snapshot before mutation; invalid snapshot returns typed error and rolls back transaction.
  - Do not use `TaskContract.BaseSHA` as `pinned_ao_commit`. Pinned AO commit must come from explicit/trusted source.
  - In `Candidate`, canonical worktree path and linked gitdir directory must come from explicit/trusted inputs. Resolve `.git` pointer file if applicable to obtain canonical gitdir directory.

### Finding P04A-I1-004: Implicit Fake Authority / Recovery Lease Lifetime
- **Severity**: HIGH (Isolation / Lease Lifecycle)
- **Status**: OPEN
- **Description**:
  1. `Coordinator.Dispatch` silently defaulted to `host.NewMemoryWorkspaceBindingAuthority()` when `c.Authority` was nil, bypassing physical disk handle validation in non-test contexts.
  2. `internal/recovery/scanner.go` defined `defaultRecoveryAuthority` returning synthetic test snapshots for production recovery sweeps.
  3. `scanner.go` closed the lease before evaluating physical identity and did not invoke `lease.Revalidate()`.
- **Remediation Requirement**:
  - `Coordinator.Dispatch` must fail closed when `c.Authority == nil`. `MemoryWorkspaceBindingAuthority` permitted only via explicit test injection.
  - Delete `defaultRecoveryAuthority` and `defaultRecoveryLease` from production scanner.
  - Recovery sweep for `DISPATCH_BOUND` with active binding and nil `r.Authority` must execute Variant B diagnostic transaction (`WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`) with zero AO /send calls.
  - When authority is present, recovery scanner must Acquire real lease, hold lease via `defer lease.Close()` throughout entire physical identity comparison, invoke `lease.Revalidate()` before decision, and handle `lease.Close()` errors.
  - Add regression tests verifying nil authority, Acquire failure, Revalidate failure, lease lifetime, and AO call count = 0.

### Finding P04A-I1-005: Unauthorized DISPATCHED Resend
- **Severity**: CRITICAL (Effect Gate / State Machine Violation)
- **Status**: OPEN
- **Description**: `Coordinator.Dispatch` contained a branch `else if task.State == domain.StateDispatched` that allowed an existing `DISPATCHED` attempt to proceed through `RecordSendRequested` and wire `/send`. This created an unauthorized re-send pathway after daemon restart, directly violating ADR-018.
- **Remediation Requirement**:
  - `Coordinator.Dispatch` must NOT use `DISPATCHED` as a recovery effect API. Remove the branch allowing existing `DISPATCHED` attempts to proceed to `/send`.
  - Any second call to `Dispatch` after bound dispatch has committed must fail closed, leaving task state `DISPATCHED`, operation stage `DISPATCH_BOUND`, and AO send count = 0.
  - Recovery scanner must remain zero-effect with zero automated resend after restart.

### Finding P04A-I1-006: Worker Report Schema & Dirty Intake Lineage
- **Severity**: HIGH (Intake Validation / Lineage Integrity)
- **Status**: OPEN
- **Description**:
  1. `WorkerReport` struct did not adhere to `docs/schemas/worker-report.schema.json` and lacked JSON Schema validation (required fields, `additionalProperties=false`, enum/pattern checks).
  2. Dirty report intake did not verify that task is `RUNNING`, attempt is current/open, and lineage matches contract before writing audit and hold mutations.
- **Remediation Requirement**:
  - Implement full schema validation matching `docs/schemas/worker-report.schema.json`.
  - Validate `task_id`, `attempt_id`, `contract_id` / `base_sha` match DB and contract before Transaction A.
  - `reported_head_sha` must come strictly from `WorkerReport.head_sha` and never be overwritten by `GitEvidenceResult`.
  - Dirty intake must verify task is `RUNNING`, current attempt matches, attempt is open, and contract lineage matches before writing audit/hold mutations. Stale, ended, or mismatched attempts must produce zero mutations.
  - Add negative tests for missing fields, extra fields, invalid shapes, wrong enums, wrong lineage, and dirty intake on stale attempts.

### Finding P04A-I1-007: Host Filesystem / Substitution Coverage
- **Severity**: HIGH (Host Authority / Win32 Proofs)
- **Status**: OPEN
- **Description**:
  1. Win32 authority did not verify that the target filesystem is NTFS or ReFS. Other or undetermined filesystems were not rejected fail-closed.
  2. Canonical UNC normalization did not guarantee preservation of absolute UNC prefix.
  3. AC-P04A-07 probes for junction, symlink, `subst`/path alias, and rename were absent or substituted by generic two-directory tests.
  4. `Candidate.PinnedAOCommit` was permitted to be empty or default to zero hash.
- **Remediation Requirement**:
  - Windows authority must query volume information and enforce filesystem is NTFS or ReFS. Non-NTFS/ReFS filesystems must fail closed with typed error (`ErrUnsupportedFilesystem`).
  - Ensure canonical UNC normalization preserves absolute UNC format.
  - Add explicit AC-P04A-07 probes covering directory junction, symlink, `subst`, and rename.
  - `Candidate.PinnedAOCommit` must be strictly validated as 40 lowercase hex chars, never defaulting to zero hash.

---

## 3. Governance State & Remediation Directives

1. **Governance State**:
   - `TASK_P04A_IMPLEMENTATION = REVISION_REQUIRED`
   - `ACTIVE_GATE = TASK_P04A_IMPLEMENTATION_REMEDIATION`
   - `P04_CODE = AUTHORIZED_P04A_REMEDIATION_ONLY`
   - `TASK_P04B/C/D = NOT_RELEASED`
   - `P05_CODE = NOT_AUTHORIZED`
   - `AUTOMATIC_RESTORE = DISABLED`
2. **Implementation Directives**:
   - Remediation work must occur strictly on isolated branch `codex/p04-001`.
   - Edits must remain strictly within the 39 whitelisted files in `allowed_scope`.
   - Zero edits permitted to `docs/**` on branch `codex/p04-001`.
   - Do NOT merge into `main`. Await External Supervisor re-audit on exact commit SHA.
