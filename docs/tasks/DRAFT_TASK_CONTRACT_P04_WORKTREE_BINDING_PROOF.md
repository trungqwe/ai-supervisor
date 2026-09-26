# DRAFT TASK CONTRACT: TASK-P04-WORKTREE-BINDING-PROOF

> **Contract ID**: `CONTRACT-TASK-P04-WORKTREE-BINDING-PROOF-01`
> **Task ID**: `TASK-P04-WORKTREE-BINDING-PROOF` (Bounded Empirical Proof for Worktree Path Authority and Binding)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P04`
> **Base SHA**: `06720c0a9eff4f83c8d1bca7b42a98bdcad4d278`
> **Status**: `BLOCKED_NOT_RELEASEABLE` (Status invariant: `NOT_RELEASED`)
> **Blocker Reference**: Finding `P04-ARCH-R5-001` (Pinned AO contains no user-selectable inert harness; disposable runtime AO sessions cannot be executed without upstream test seam; runtime physical binding transitioned to fail-closed Stage B validation on authorized sessions).
> **Draft Lineage Model**: Adopted Model A. Pre-contract editing rounds represent editorial draft iterations of an unreleased document. Pursuant to [`docs/08_TASK_CONTRACT.md`](../08_TASK_CONTRACT.md) and [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), formal immutable contract revision numbering commences only upon candidate release.
> **Authority**: Formulated pursuant to accepted `docs/24_CHANGE_GOVERNANCE.md`, `AGENTS.md`, `PROPOSAL-P04-001` (Rev 6), `DRAFT-ADR-018` (Rev 6), and `PLAN-P04-WORKTREE-BINDING-PROOF.md` (Rev 4).
> **Worker Profile**: Orchestration worker may use `antigravity-standard`; disposable AO under test must not invoke LLM models.
> **Runtime Invariants**: `AUTOMATIC_RESTORE = DISABLED`; zero production database modification; zero autonomous agent prompts; zero interaction with user repository or recovery directory.

---

## 1. Immutable TaskContract JSON

```json
{
  "contract_id": "CONTRACT-TASK-P04-WORKTREE-BINDING-PROOF-01",
  "task_id": "TASK-P04-WORKTREE-BINDING-PROOF",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P04",
  "objective": "Execute static pinned AO source inspection and non-runtime authority reconciliation to establish deterministic worktree path binding specifications for Phase P04 without touching user databases or running autonomous agent coding. Runtime disposable AO execution is blocked and deferred due to lack of an upstream inert harness.",
  "requirements": [
    "FR-008",
    "SEC-002",
    "SEC-003",
    "NFR-005",
    "NFR-007"
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
  "base_sha": "06720c0a9eff4f83c8d1bca7b42a98bdcad4d278",
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
    "Strictly zero interaction with user databases, user sessions, or primary repository (D:\\TU_CODE\\ai-supervisor)",
    "Strictly zero interaction with recovery directory (D:\\TU_CODE\\ai-supervisor-user-wip-recovery\\P03-EXIT-R2-001)",
    "Strictly zero disposable AO runtime worker session creation, killing, restoring, or prompting (blocked pending upstream inert seam)",
    "Worker orchestration may utilize approved profile antigravity-standard; disposable AO under test must not invoke LLM models",
    "All file paths must reside strictly within the absolute host-injected state root",
    "AUTOMATIC_RESTORE remains DISABLED"
  ],
  "verification_requests": [
    {
      "id": "vr-01-git-porcelain",
      "profile_id": "proof-git-porcelain",
      "parameters": {
        "subcommand": "worktree-list",
        "format": "porcelain-z"
      },
      "timeout_seconds": 30
    },
    {
      "id": "vr-02-fs-identity",
      "profile_id": "proof-fs-identity-probe",
      "parameters": {
        "probe_kind": "verify-worktree-containment-and-identity"
      },
      "timeout_seconds": 30
    },
    {
      "id": "vr-03-edge-cases",
      "profile_id": "proof-fs-identity-probe",
      "parameters": {
        "probe_kind": "characterize-missing-directory-and-junction"
      },
      "timeout_seconds": 30
    }
  ],
  "required_evidence": [
    "docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md",
    "Git diff showing zero changes outside allowed_scope",
    "Command execution logs and exit codes for all proof steps"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/09_WORKER_REPORT.md",
  "stop_conditions": [
    "Discovery of non-deterministic worktree path allocation in pinned AO",
    "Attempted access or pollution of user database, user sessions, or primary repository outside state root"
  ],
  "acceptance_criteria": [
    "Static source inspection of backend/internal/adapters/workspace/gitworktree/workspace.go at pinned commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6 conclusively documented with official permalinks",
    "Authoritative attempt_workspace_bindings schema specified with immutable triggers, VolumeSerialNumber, and 128-bit FileId",
    "Runtime physical binding transitioned into Stage B fail-closed validation on authorized sessions during normal operation",
    "Zero runtime disposable AO sessions executed or prompted",
    "Zero modifications to production code, schemas, or migrations"
  ]
}
```

---

## 2. Release Guardrails & Governance

- **Release Status**: `BLOCKED_NOT_RELEASEABLE` (Status invariant: `NOT_RELEASED`).
- **Release Condition**: This draft contract cannot be released as an active implementation task because disposable AO runtime testing is blocked by the absence of an upstream inert harness (`DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN`).
- **Stage B Transition**: Physical worktree authority and binding is validated via Stage B fail-closed checks on authorized sessions during normal operation, backed by `attempt_workspace_bindings` (Schema v6).
- **Candidate Release**: No candidate contract is created or released in this state.
