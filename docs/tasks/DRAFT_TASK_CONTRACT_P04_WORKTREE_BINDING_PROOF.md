# DRAFT TASK CONTRACT: TASK-P04-WORKTREE-BINDING-PROOF

> **Contract ID**: `CONTRACT-TASK-P04-WORKTREE-BINDING-PROOF-01`
> **Task ID**: `TASK-P04-WORKTREE-BINDING-PROOF` (Bounded Empirical Proof for Worktree Path Authority and Binding)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P04`
> **Base SHA**: `0ab0d8bd151dfd0501c6d40ac0862e1574adec77`
> **Status**: `NOT_RELEASED (Draft for Change Governance and Empirical Proof Planning)`
> **Authority**: Formulated pursuant to accepted `docs/24_CHANGE_GOVERNANCE.md`, `AGENTS.md`, `PROPOSAL-P04-001`, `DRAFT-ADR-018`, and `PLAN-P04-WORKTREE-BINDING-PROOF.md`.
> **Implementation Scope**: Authorized strictly within `allowed_scope` upon formal release by External Supervisor.
> **Runtime Invariants**: `AUTOMATIC_RESTORE = DISABLED`; zero production database modification; zero autonomous agent prompts.

---

## 1. Immutable TaskContract JSON

```json
{
  "contract_id": "CONTRACT-TASK-P04-WORKTREE-BINDING-PROOF-01",
  "task_id": "TASK-P04-WORKTREE-BINDING-PROOF",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P04",
  "objective": "Execute bounded empirical proof across static pinned AO source inspection, isolated disposable AO runtime testing with dual sessions, and authority reconciliation to establish deterministic worktree path binding for Phase P04 without touching user databases or running autonomous agent coding.",
  "requirements": [
    "FR-004",
    "NFR-003",
    "SEC-001",
    "SEC-003"
  ],
  "architecture_refs": [
    "docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md",
    "docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md",
    "docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/07_SECURITY_MODEL.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/sources/SOURCE_REGISTRY.md",
    "docs/sources/REUSE_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md"
  ],
  "base_sha": "0ab0d8bd151dfd0501c6d40ac0862e1574adec77",
  "allowed_scope": [
    "docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md",
    "test/proof/**"
  ],
  "forbidden_scope": [
    "internal/**",
    "cmd/**",
    "migrations/**",
    "docs/schemas/**",
    "docs/adr/ADR-016*",
    "docs/adr/ADR-017*"
  ],
  "constraints": [
    "Strictly zero interaction with user databases, user sessions, or user repositories",
    "Strictly zero autonomous worker coding prompts or LLM generation during proof execution",
    "All runtime proof activities must execute within disposable sandbox (.supervisor/proof/ao_disposable/)",
    "Proof must test minimum two sessions on the same project to verify uniqueness and non-collision",
    "All commands, exit codes, sanitized paths, and artifact hashes must be documented in proof report",
    "This draft contract is NOT_RELEASED and does not authorize code implementation"
  ],
  "acceptance_criteria": [
    "AC-P04-PROOF-01: Static pinned-source proof documents Workspace.Create, Workspace.managedPath, defaultSessionBranchName, and Workspace.Restore in backend/internal/adapters/workspace/gitworktree/workspace.go at pinned AO commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6",
    "AC-P04-PROOF-02: Isolated runtime proof executes AO v0.13.0 against disposable database and managed root, creating two distinct sessions for the same project without collisions",
    "AC-P04-PROOF-03: Dual sessions demonstrate exact path formula <managedRoot>/<projectID>/<sessionID> matching git worktree list --porcelain -z and branch ao/<sessionID>",
    "AC-P04-PROOF-04: Daemon restart and session restore verify deterministic worktree path preservation across process cycles",
    "AC-P04-PROOF-05: Missing directory, junction/symlink alias, and stale worktree failure modes are characterized with exit codes and diagnostics",
    "AC-P04-PROOF-06: Proof report documents definitive conclusion (Determination A, B, or C) with complete sanitized logs and artifact hashes in docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md"
  ],
  "required_evidence": [
    "docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md",
    "Git diff showing zero changes outside allowed_scope",
    "Command execution logs and exit codes for all proof steps"
  ],
  "worker_profile": "proof-investigator",
  "report_contract": "docs/09_WORKER_REPORT.md",
  "stop_conditions": [
    "AO runtime failure or incompatibility in disposable sandbox",
    "Discovery of non-deterministic worktree path allocation in pinned AO",
    "Attempted access or pollution of user database, user sessions, or primary repository"
  ],
  "verification_requests": []
}
```

---

## 2. Governance Context & Non-Release Guardrails

1. **Non-Release Status**: This draft Task Contract is **`NOT_RELEASED`**. It is created to satisfy the governance mandate of [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md) and [`AGENTS.md`](../../AGENTS.md) that all tasks dispatched to workers must be governed by an immutable, schema-validated TaskContract.
2. **Distinct Proof Task**: This task is strictly a bounded empirical research and proof task. It does not authorize writing production Go code in `internal/**`, modifying database schemas or migrations, or modifying accepted ADRs.
3. **Execution Guardrails**:
   - Zero interaction with live developer databases or user repositories.
   - All runtime commands must execute within `.supervisor/proof/ao_disposable/`.
   - Results will be formally submitted to External Supervisor for re-audit before any Phase P04 implementation contract is released.
