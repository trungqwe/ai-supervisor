# DRAFT TASK CONTRACT: TASK-P04-WORKTREE-BINDING-PROOF

> **Contract ID**: `CONTRACT-TASK-P04-WORKTREE-BINDING-PROOF-01`
> **Task ID**: `TASK-P04-WORKTREE-BINDING-PROOF` (Bounded Empirical Proof for Worktree Path Authority and Binding)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P04`
> **Base SHA**: `a7e80d7c677467380cbb754429672705563ca056`
> **Status**: `NOT_RELEASED (Draft for Change Governance and Empirical Proof Planning)`
> **Draft Lineage Model**: Adopted Model A. Pre-contract editing rounds (Round 1 through Round 4) represent editorial draft iterations of an unreleased document. Pursuant to [`docs/08_TASK_CONTRACT.md`](../08_TASK_CONTRACT.md) and [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), formal immutable contract revision numbering commences only upon candidate release.
> **Authority**: Formulated pursuant to accepted `docs/24_CHANGE_GOVERNANCE.md`, `AGENTS.md`, `PROPOSAL-P04-001` (Rev 5), `DRAFT-ADR-018` (Rev 5), and `PLAN-P04-WORKTREE-BINDING-PROOF.md` (Rev 3).
> **Implementation Scope**: Authorized strictly within `allowed_scope` upon formal release by External Supervisor.
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
  "base_sha": "a7e80d7c677467380cbb754429672705563ca056",
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
    "Strictly zero autonomous worker coding prompts or LLM token generation during proof execution",
    "All runtime proof activities must execute within absolute host-injected root (<SUPERVISOR_STATE_ROOT>\\proof\\p04-worktree-binding\\<proof_run_id>\\)",
    "Proof must test minimum two sessions on the same project using server-generated session IDs to verify uniqueness and non-collision",
    "All commands, exit codes, sanitized paths, and artifact hashes must be documented in proof report",
    "This draft contract is NOT_RELEASED and does not authorize code implementation"
  ],
  "acceptance_criteria": [
    "AC-P04-PROOF-01: Static pinned-source proof documents Workspace.Create, Workspace.managedPath (signature (ports.WorkspaceConfig) (string, error)), Workspace.restorePath, defaultSessionBranchName, and Workspace.Restore in backend/internal/adapters/workspace/gitworktree/workspace.go at pinned AO commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6 in Untrivial-ai/agent-orchestrator with official GitHub permalinks",
    "AC-P04-PROOF-02: Isolated runtime proof executes AO v0.13.0 on ephemeral port against disposable database and managed root under state root, capturing two distinct server-generated session IDs for the same project without collisions",
    "AC-P04-PROOF-03: Dual sessions demonstrate exact path formula <managedRoot>/<projectID>/<sessionID> matching git worktree list --porcelain -z and branch ao/<sessionID>",
    "AC-P04-PROOF-04: Wire kill, terminal observation poll, daemon restart, and session restore verify deterministic worktree path preservation across process cycles with HTTP 200",
    "AC-P04-PROOF-05: Missing directory, junction/symlink alias, and stale worktree behaviors are characterized without treating expected probe outcomes as infrastructure failures",
    "AC-P04-PROOF-06: Proof report documents definitive conclusion (Determination A, B, or C) with complete sanitized logs and artifact hashes in docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md"
  ],
  "required_evidence": [
    "docs/proofs/P04_WORKTREE_BINDING_PROOF_REPORT.md",
    "Git diff showing zero changes outside allowed_scope",
    "Command execution logs and exit codes for all proof steps"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/09_WORKER_REPORT.md",
  "stop_conditions": [
    "Unexpected harness or AO infrastructure crash (panic, segmentation fault, unhandled I/O exception)",
    "Discovery of non-deterministic worktree path allocation in pinned AO",
    "Attempted access or pollution of user database, user sessions, or primary repository outside state root"
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
      "id": "vr-02-ao-spawn-dual",
      "profile_id": "proof-ao-session-probe",
      "parameters": {
        "probe_kind": "spawn-dual-sessions",
        "project_id": "proj-proof-01"
      },
      "timeout_seconds": 60
    },
    {
      "id": "vr-03-fs-identity",
      "profile_id": "proof-fs-identity-probe",
      "parameters": {
        "probe_kind": "verify-worktree-containment-and-identity"
      },
      "timeout_seconds": 30
    },
    {
      "id": "vr-04-ao-restart-restore",
      "profile_id": "proof-ao-session-probe",
      "parameters": {
        "probe_kind": "kill-poll-restart-and-restore"
      },
      "timeout_seconds": 60
    },
    {
      "id": "vr-05-edge-cases",
      "profile_id": "proof-fs-identity-probe",
      "parameters": {
        "probe_kind": "characterize-missing-directory-and-junction"
      },
      "timeout_seconds": 30
    }
  ]
}
```

---

## 2. Governance Context & Non-Release Guardrails

1. **Non-Release Status**: This draft Task Contract is **`NOT_RELEASED`**. It is created to satisfy the governance mandate of [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md) and [`AGENTS.md`](../../AGENTS.md) that all tasks dispatched to workers must be governed by an immutable, schema-validated TaskContract.
2. **Recorded Design Blockers**:
   - `BLOCKER-P04-VERIFICATION-CATALOG = PROFILES_NOT_YET_REGISTERED_IN_LIVE_CATALOG`: Verification profiles `proof-git-porcelain`, `proof-ao-session-probe`, and `proof-fs-identity-probe` are design-level profiles not yet registered in the production VerificationPolicyCatalog.
   - `DESIGN_BLOCKER_P04_INERT_AO_HARNESS = OPEN`: An inert, zero-LLM AO test harness mode must be authoritatively proven before dispatch. Pinned AO `backend/internal/domain/harness.go` contains no user-selectable inert harness.
3. **Stage B Verification Status**:
   - Structural JSON Schema: **`PASS`** (validated against `docs/schemas/task-contract.schema.json`).
   - Stage B Runtime Policy Verification: **`UNVERIFIED`** (pending live catalog registration; design-level catalogs cannot report Stage B PASS).
4. **Execution Authority**:
   - Zero worker authority to supply raw executables, command strings, endpoints, roots, or harnesses.
   - All external execution is mediated exclusively via typed profiles with parameter enums, bounded timeouts, and strict output capture limits.
5. **Negative Probe Distinction**:
   - Expected negative outcomes (e.g. AO recreating worktrees upon missing directories or returning structured HTTP errors) are valid probe characterizations and do **NOT** trigger stop conditions.
   - Stop conditions are strictly reserved for unexpected infrastructure crashes, non-deterministic path deviations, or unauthorized boundary traversal.
