# P04_TASK_001_CONTRACT_PLANNING_EXTERNAL_AUDIT_001.md

**Audit Target**: Subtask P04A Task Contract & Implementation Planning Draft Review (Round 1)
**Audited Documents**:
- `docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md`
- `docs/tasks/DRAFT_TASK_CONTRACT_P04_001.md`
**Audited Draft Commit**: `80b997c89d57280286c63599c1c539161b31b5eb`
**Integration Audit Reference Commit**: `33cf3b11db82e355086172764ff7df8821f4cdef`
**Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
**Authority**: External Supervisor
**Date**: 2026-09-27
**Verdict**: `DRAFT_P04A = REVISION_REQUIRED`
**Active Gate**: `TASK_P04A_CONTRACT_PLANNING`

---

## 1. Executive Summary & Audit Determination

External Supervisor conducted an independent audit of the initial draft planning and contract documentation for Subtask P04A (`PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md` and `DRAFT_TASK_CONTRACT_P04_001.md`) committed at `80b997c89d57280286c63599c1c539161b31b5eb`.

The audit evaluated compliance against accepted `ADR-018`, `PROPOSAL-P04-001 Revision 22`, canonical requirements (`FR-008`, `SEC-002`, `SEC-003`, `NFR-005`, `NFR-007`, `NFR-008`), and repository task contract governance (`docs/08_TASK_CONTRACT.md`, `docs/24_CHANGE_GOVERNANCE.md`).

**Audit Verdict**: `DRAFT_P04A = REVISION_REQUIRED`.
Release authorization is withheld (`TASK_P04A_TASK_CONTRACT = NOT_RELEASED`).
Production code writing remains strictly held (`P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`).
Zero Phase P05 code authorized (`P05_CODE = NOT_AUTHORIZED`).
Zero candidate contracts or release contracts may be formulated until all findings are remediated and verified.

---

## 2. Audit Findings & Required Remediation

### Finding `P04A-C1-001`: Schema v6 DDL Drift
- **Severity**: Critical
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**: The DDL presented in `docs/plans/PLAN-P04A-WORKSPACE-BINDING-AND-CLAIMS.md` Section 3.1 deviated from the accepted canonical DDL established in `ADR-018` and `docs/05_DOMAIN_MODEL.md`. Specifically:
  1. `attempt_workspace_bindings`: The draft introduced a redundant `binding_id TEXT PRIMARY KEY`, whereas canonical ADR-018 specifies `attempt_id TEXT PRIMARY KEY REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT`. In addition, missing fields and constraints included `session_id`, `terminal_generation`, `pinned_ao_commit`, `created_at_epoch_ms`, composite foreign key `FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id)`, and state-dependent `released_at_epoch_ms` CHECK constraints.
  2. `worker_claims`: The draft omitted the `attempt_id UNIQUE` constraint, used unvalidated `claim_payload_json` instead of `payload_json TEXT NOT NULL CHECK (json_valid(payload_json) = 1 AND ...)`, used `claimed_at_epoch_ms` instead of canonical `created_at_epoch_ms`, and omitted the composite lineage foreign key.
  3. `review_integrity_holds`: The draft declared only 3 hold reasons instead of the 5 canonical hold reasons (`DIRTY_WORKTREE_DETECTED`, `BUNDLE_HASH_CONFLICT`, `INVARIANT_MISMATCH`, `UNVERIFIED_CLAIM_DETECTED`, `SECURITY_POLICY_VIOLATION`), used `status` instead of canonical `hold_state`, omitted `diagnostic_fingerprint` (64-char lowercase hex), omitted `occurrence_number`, and omitted explicit foreign key references for `rejection_audit_event_id` and `resolution_audit_event_id`.
  4. Index: Canonical index `idx_review_integrity_holds_active_dedup` must be on `(attempt_id, hold_reason, diagnostic_fingerprint) WHERE hold_state = 'ACTIVE'`.
  5. Triggers: The 9 canonical triggers must be reproduced verbatim from ADR-018 without summarization or modification.
- **Required Remediation**: Replace all self-drafted DDL in `PLAN-P04A` with the exact 13 SQL objects (3 tables, 1 index, 9 triggers) from ADR-018. Run an automated parity verification script confirming 13/13 objects match ADR-018.

---

### Finding `P04A-C1-002`: Subtask Ownership Boundary Confusion (P04A vs P04B)
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**: The draft plan and contract assigned process execution of Git commands (`git status --porcelain=v1 -z --untracked-files=all`, `git diff-index --quiet HEAD --`) and Git allowlist parsing to Subtask P04A. This violates the Model 1 architectural boundary established in ADR-018 and Re-Audit 002:
  * Subtask P04B is the dedicated Git evidence collector responsible for invoking git processes, enforcing the 10-command allowlist with `GIT_OPTIONAL_LOCKS=0`, and extracting source snapshots.
  * Subtask P04A must NOT execute git processes or parse raw git command outputs. P04A is the persistence and dispatch seam owner; it consumes a structured, typed `GitEvidenceResult` from an injected interface.
- **Required Remediation**:
  1. Update `PLAN-P04A` and `DRAFT_TASK_CONTRACT_P04_001.md` (objective, constraints, acceptance criteria, test plan, and required evidence) to state clearly that P04A only defines the input/interface seam for `GitEvidenceResult`.
  2. In P04A, clean/dirty behavior tests must be exercised via fake/mock results (`test/fakes/git_evidence_fake.go`).
  3. All actual execution of Git commands is reserved strictly for Subtask P04B.

---

### Finding `P04A-C1-003`: Descriptor Formulations and Hold Replay Deficiencies
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**:
  1. The draft plan used ad-hoc formulas for Descriptors A and B rather than the exact structures mandated by ADR-018 §7.5.
  2. Descriptor A must contain: `attempt_id`, `contract_id`, `diagnostic_fingerprint`, `hold_reason`, `kind: "review_integrity_hold"`, `occurrence_number`, `pair_id`, `task_id`, `version: 1`.
  3. Descriptor B (generalized for `EVIDENCE_COLLECTION_FAILED`) must contain: `attempt_id`, `contract_id`, `diagnostic_fingerprint`, `event_type`, `hold_id`, `occurrence_number`, `pair_id`, `reason`, `sanitized_input_fingerprint`, `task_id`, `version: 1`.
  4. Variant B (`WORKSPACE_BINDING_GUARD`) must declare: `conflict_source = 'WORKSPACE_BINDING_GUARD'`, `dispatch_operation_id`, `attempted_reason` (one of the 4 approved literals), `conflict_type = 'LINEAGE_MISMATCH'`, `sanitized_input_fingerprint`, `diagnostic_fingerprint`, and `version: 2`; `colliding_event_id` must be strictly absent. The audit event `details_json` must contain both fingerprints, `actor_role = 'SUPERVISOR'`, and all variant fields.
  5. The draft hardcoded `occurrence_number = 1`, failing to specify the recurrence lifecycle: exact replay of an existing active diagnostic returns the existing hold without re-insertion; recurrence after resolution computes `occurrence_number = COALESCE(MAX(occurrence_number), 0) + 1` (occurrence N+1); and concurrent caller races must have dedicated rollback and zero-orphan audit guarantees.
- **Required Remediation**: Standardize Descriptors A and B, Variant B, exact replay, recurrence lifecycle, and zero-orphan audit semantics to match ADR-018 verbatim.

---

### Finding `P04A-C1-004`: Invalid Verification Profile and Missing Policy Catalog Schema
- **Severity**: High
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**:
  1. The draft contract used `profile_id: "antigravity-standard"` inside `verification_requests`. Under ADR-013 and canonical requirements, `antigravity-standard` is a worker harness profile valid exclusively in `worker_profile`, not a host verification profile.
  2. The verification requests used generic shell command strings (`"command": "go test ..."`), which bypasses typed verification validation and violates ADR-013.
  3. A typed verification profile `go-test-p04-001` must be formally proposed outside the immutable JSON block with:
     * `CwdPolicy = "worktree_root"`
     * `MaxTimeoutSeconds = 300`
     * `additionalProperties = false`
     * Typed parameters: `package` (enum restricted strictly to packages tested by the contract) and `flags` (const `["-v", "-race", "-count=1"]`).
  4. Each verification request must declare `profile_id: "go-test-p04-001"`, `cwd: "."`, `timeout_seconds <= 300`, and typed `parameters.package` and `parameters.flags`.
  5. The contract must pass `TaskContractValidator.ValidateRaw` against this proposed catalog, and must be tested against 6 mandatory negative probes: unknown profile, timeout = 301, `-exec` flag injection, package outside enum, `cwd = ../escape`, and extra parameter/command property. All 6 must be rejected.
- **Required Remediation**: Update `DRAFT_TASK_CONTRACT_P04_001.md` verification requests and document the proposed catalog metadata in Section 2; validate using `TaskContractValidator.ValidateRaw` and execute all 6 negative probes.

---

### Finding `P04A-C1-005`: Unapproved Audit Event Literal `WORKER_REPORT_INGESTED`
- **Severity**: Medium
- **Status**: `OPEN_PENDING_EXTERNAL_REAUDIT`
- **Description**: The draft plan and contract introduced an unapproved audit event literal `WORKER_REPORT_INGESTED`. This literal does not exist in accepted ADRs (ADR-016, ADR-017, ADR-018), `docs/05_DOMAIN_MODEL.md`, or the canonical audit registry. Under `docs/24_CHANGE_GOVERNANCE.md`, workers are strictly forbidden from inventing audit event types without an approved proposal and ADR.
- **Required Remediation**: Delete `WORKER_REPORT_INGESTED` from `PLAN-P04A`, `DRAFT_TASK_CONTRACT_P04_001.md`, and acceptance criteria. Transaction A atomically persists `worker_claims`, advances `tasks.state` to `REPORT_READY`, and CAS updates `attempt_workspace_bindings` to `RETAINED_FOR_VERIFICATION` without emitting an unapproved event.

---

## 3. Provenance & Baseline Confirmation

- **Canonical Integration Audit Commit**: `33cf3b11db82e355086172764ff7df8821f4cdef`
- **Audited Draft Commit**: `80b997c89d57280286c63599c1c539161b31b5eb`
- **Base SHA**: `db6b654f03a7ce3c8e6ae2cd180f2cb7236bf7c2`
- **Candidate Wipe Verification**: Zero candidate contracts exist; zero release contracts formulated.

---

## 4. Governance Status & Directives

1. **State Update**:
   - `TASK_P04A_TASK_CONTRACT = NOT_RELEASED`
   - `P04_CODE = HELD_PENDING_TASK_P04A_CONTRACT_RELEASE`
   - `ACTIVE_GATE = TASK_P04A_CONTRACT_PLANNING`
   - `P05_CODE = NOT_AUTHORIZED`
2. **Findings Status Table**:

| Finding ID | Title | Severity | Status |
|---|---|---|---|
| `P04A-C1-001` | Schema v6 DDL Drift | Critical | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-C1-002` | Sai ownership P04A/P04B | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-C1-003` | Descriptor và hold replay sai | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-C1-004` | Verification policy không hợp lệ | High | `OPEN_PENDING_EXTERNAL_REAUDIT` |
| `P04A-C1-005` | Audit literal chưa được duyệt | Medium | `OPEN_PENDING_EXTERNAL_REAUDIT` |

3. **Remediation Mandate**: Worker must perform remediation strictly on the 5 whitelisted governance files without creating Go code or candidate contracts. Findings remain `OPEN_PENDING_EXTERNAL_REAUDIT`.
