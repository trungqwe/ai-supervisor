# CANDIDATE TASK CONTRACT: TASK-P04-DOC-RECONCILIATION-ADR-018

> **Contract ID**: `CONTRACT-TASK-P04-DOC-RECONCILIATION-01`
> **Task ID**: `TASK-P04-DOC-RECONCILIATION-ADR-018` (Canonical Specification Reconciliation pursuant to Accepted ADR-018)
> **Revision Number**: `1`
> **Supersedes Contract ID**: `null`
> **Phase ID**: `P04`
> **Base SHA**: `b174ef3e88409f82b10afe0fcce88ba218530863`
> **Status**: `CANDIDATE_NOT_RELEASED` (Status invariant: `NOT_RELEASED`)
> **Authority**: Formulated pursuant to accepted [`docs/adr/ADR-018-evidence-review-and-verification-isolation.md`](../adr/ADR-018-evidence-review-and-verification-isolation.md) (`EXTERNAL_APPROVED` and `ADR_018_ACCEPTANCE = GRANTED` by External Supervisor Re-Audit 021), approved `PROPOSAL-P04-001` (Revision 22), approved baseline `PROPOSAL-P04-002` (Revision 9), and scope plan [`docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md`](../plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md) (Revision 2).
> **Governance Invariant**: Candidate contract submitted for External Supervisor audit. Strictly `NOT_RELEASED`. Zero canonical documentation edits, zero Go production code, zero migration scripts, and zero live AO calls authorized.

---

> [!CRITICAL]
> **GOVERNANCE STATUS: CANDIDATE ONLY / NOT_RELEASED**.
> This document is a candidate contract awaiting release audit.
> It does **NOT** authorize production code or specification execution.
> Production coding remains strictly **`HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT`**.
> Production task contract `TASK-P04-001` (Subtask P04A) remains strictly **`NOT_RELEASED`**.
> Candidate contract does **NOT** self-claim Stage B runtime catalog or task release authorization.
> Canonical specification reconciliation has **NOT** yet been executed.

## 1. Authoritative Canonical TaskContract JSON Object

```json
{
  "contract_id": "CONTRACT-TASK-P04-DOC-RECONCILIATION-01",
  "task_id": "TASK-P04-DOC-RECONCILIATION-ADR-018",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P04",
  "objective": "Execute single-pass canonical specification reconciliation across items CR-01 through CR-12 in docs/02, docs/04, docs/05, docs/06, docs/10, docs/12, docs/14, docs/17, docs/21, docs/22, docs/phases/P04_EVIDENCE_REVIEW.md, docs/sources/SOURCE_REGISTRY.md, docs/sources/REUSE_MATRIX.md, docs/schemas/review-bundle.schema.json, and review-bundle examples pursuant to accepted ADR-018 and approved PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md.",
  "requirements": [
    "FR-008",
    "NFR-008",
    "SEC-002",
    "SEC-003"
  ],
  "architecture_refs": [
    "docs/adr/ADR-018-evidence-review-and-verification-isolation.md",
    "docs/plans/PLAN-P04-CANONICAL-RECONCILIATION-ADR-018.md",
    "docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md",
    "docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/24_CHANGE_GOVERNANCE.md"
  ],
  "base_sha": "b174ef3e88409f82b10afe0fcce88ba218530863",
  "allowed_scope": [
    "docs/02_REQUIREMENTS.md",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/06_WORKFLOW_STATE_MACHINE.md",
    "docs/10_REVIEW_BUNDLE.md",
    "docs/12_UPSTREAM_INTEGRATION.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/17_ROADMAP.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md",
    "docs/phases/P04_EVIDENCE_REVIEW.md",
    "docs/sources/SOURCE_REGISTRY.md",
    "docs/sources/REUSE_MATRIX.md",
    "docs/schemas/review-bundle.schema.json",
    "docs/schemas/examples/review-bundle.valid.json",
    "docs/schemas/examples/review-bundle.invalid.json"
  ],
  "forbidden_scope": [
    "cmd/**",
    "internal/**",
    "test/**",
    "migrations/**",
    "docs/adr/**",
    "docs/proposals/**",
    "docs/audits/**",
    "docs/plans/**",
    "docs/tasks/**",
    "AGENTS.md",
    "docs/18_CURRENT_STATE.md",
    "docs/schemas/task-contract.schema.json",
    "docs/schemas/worker-report.schema.json",
    "docs/schemas/examples/task-contract.*.json",
    "docs/schemas/examples/worker-report.*.json"
  ],
  "constraints": [
    "Zero Go production code, test code, or build artifact modification (cmd/**, internal/**, test/**)",
    "Zero database migrations or SQLite schema DDL execution",
    "Accepted ADR-018, approved proposals, and audit records are permanently frozen and immutable",
    "Full synchronization with accepted ADR-018, approved PROPOSAL-P04-001 (Rev 22), and approved baseline PROPOSAL-P04-002 (Rev 9)",
    "Zero semantic contradiction with ADR-018 decisions D1 through D7 across all reconciled specifications",
    "Verbatim literal compliance for commands (git status --porcelain=v1 -z --untracked-files=all), paths (artifacts/<first-two-hex>/<captured_sha256>), predicates (Guard 9 and Guard 10), and descriptors (A, B, C)",
    "Worker report schema (docs/schemas/worker-report.schema.json) and task contract schema (docs/schemas/task-contract.schema.json) remain strictly unmutated",
    "AUTOMATIC_RESTORE remains permanently DISABLED; VERIFIED_OPERATOR_PRINCIPAL remains OPEN_FAIL_CLOSED_DEPENDENCY",
    "Subtask P04A Task Contract remains NOT_RELEASED pending approved canonical reconciliation",
    "Candidate contract does not self-claim Stage B runtime catalog or task release authorization"
  ],
  "acceptance_criteria": [
    "AC-P04-DOC-01: Reconciles CR-01 in docs/04 and docs/12 separating WorkspaceBindingSnapshot from live WorkspaceBindingLease and establishing Coordinator.Dispatch as single effect gate",
    "AC-P04-DOC-02: Reconciles CR-02 in docs/04, docs/12, and docs/14 aligning exact Guard 9 and Guard 10 SQL predicates with 100% eradication of status, PENDING, and IN_FLIGHT descriptions",
    "AC-P04-DOC-03: Reconciles CR-03 in docs/06 and docs/14 documenting atomic D12 terminal transition (tasks DISPATCHED -> FAILED, recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE') and distinct diagnostic hold creation",
    "AC-P04-DOC-04: Reconciles CR-04 in docs/04 and docs/14 eliminating invalid directory hardlink swap claims and formalizing valid NTFS directory substitution probe matrix",
    "AC-P04-DOC-05: Reconciles CR-05 in docs/02 (FR-008), docs/04, and docs/06 documenting Subtask P04B pure in-memory collector, git status --porcelain=v1 -z --untracked-files=all, and clean-intake diagnostic transaction preserving RUNNING",
    "AC-P04-DOC-06: Reconciles CR-06 in docs/02 (NFR-008), docs/04, and docs/12 documenting Subtask P04C Windows AppContainer runner with explicit handle list, atomic Job Object kill-on-close, and 10MB/50MB stream limits",
    "AC-P04-DOC-07: Reconciles CR-07 in docs/12, docs/phases/P04_EVIDENCE_REVIEW.md, and docs/sources/SOURCE_REGISTRY.md documenting inert fake AO adapter harness and decoupling live AO integration track",
    "AC-P04-DOC-08: Reconciles CR-08 in docs/04, docs/10, docs/schemas/review-bundle.schema.json, and review-bundle examples documenting CAS write-through to artifacts/<first-two-hex>/<captured_sha256>, dual head SHAs, and ReviewBundle latency semantics (compilation_latency_ms, 2x2 provenance matrix, nfr008_compliance_status = 'UNVERIFIED', decoupled commit telemetry)",
    "AC-P04-DOC-09: Reconciles CR-09 in docs/05, docs/17, and docs/22 formalizing Model 1 persistence ownership (Schema v6 owned by P04A, Schema v9 owned by P04D, P04B/C pure in-memory) and contract sequencing P04A -> P04B -> P04C -> P04D",
    "AC-P04-DOC-10: Reconciles CR-10 in docs/04, docs/05, and docs/14 documenting RFC 8785 JCS event derivation, Descriptors A/B/C, and REVIEW_INTEGRITY_CONFLICT variant discrimination (AUDIT_EVENT_ID_COLLISION vs WORKSPACE_BINDING_GUARD)",
    "AC-P04-DOC-11: Reconciles CR-11 in docs/17, docs/21, and docs/phases/P04_EVIDENCE_REVIEW.md updating FR-008/NFR-008 traceability links and establishing Subtask P04A as first releaseable implementation work package",
    "AC-P04-DOC-12: Reconciles CR-12 in docs/sources/SOURCE_REGISTRY.md and docs/sources/REUSE_MATRIX.md updating reuse boundaries for crypto/sha256, kernel32.dll, and jsonschema-go with zero duplicate upstream logic",
    "AC-P04-DOC-13: JSON Schema draft-07 two-way validation: docs/schemas/examples/review-bundle.valid.json MUST validate successfully; docs/schemas/examples/review-bundle.invalid.json MUST be rejected due to expected schema constraint violation (not random JSON syntax/parse failure)",
    "AC-P04-DOC-14: Git diff hygiene passes with zero trailing whitespaces (git diff --check exit code 0) and zero modified files outside allowed_scope",
    "AC-P04-DOC-15: Portable relative markdown link validation passes with zero broken links and zero absolute file:// or Windows drive references"
  ],
  "verification_requests": [
    {
      "id": "vr-01-git-diff-check",
      "profile_id": "doc-reconciliation-check",
      "parameters": {
        "check_kind": "git-diff-hygiene"
      },
      "cwd": ".",
      "timeout_seconds": 60
    },
    {
      "id": "vr-02-markdown-links",
      "profile_id": "doc-reconciliation-check",
      "parameters": {
        "check_kind": "portable-relative-link-validation"
      },
      "cwd": ".",
      "timeout_seconds": 60
    },
    {
      "id": "vr-03-schema-validation",
      "profile_id": "doc-reconciliation-check",
      "parameters": {
        "check_kind": "json-schema-draft07-validation"
      },
      "cwd": ".",
      "timeout_seconds": 60
    }
  ],
  "required_evidence": [
    "git diff --check exit code 0 across all modified files",
    "portable relative markdown link validation report with zero errors",
    "JSON schema draft-07 validation exit code 0 for review-bundle.schema.json and examples",
    "static grep report confirming 100% eradication of non-canonical restore/provisioning predicates and hardlink swap claims",
    "git diff stat confirming zero changes outside allowed_scope and zero production Go code touched"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/09_WORKER_REPORT.md",
  "stop_conditions": [
    "Discovery of unavoidable contradiction between ADR-018 decisions that cannot be resolved from approved ADR text",
    "Attempted access or modification of files outside allowed_scope, including internal Go code, migrations, or accepted ADRs",
    "Literal drift detected against ADR-018 verbatim tokens, DDL definitions, or descriptor schemas",
    "Failure of review-bundle schema or examples under JSON Schema draft-07 validation"
  ]
}
```

---

## 2. Proposed Verification Profile Catalog Delta

### 2.1. Declarative Profile Specification: `doc-reconciliation-check`

Pursuant to ADR-013, ADR-018, and finding `P04-CRCONTRACT-R1-004`, this task contract declares the exact Verification Profile Catalog entry required to evaluate the documentation reconciliation verification requests.

- **Profile ID**: `doc-reconciliation-check`
- **Catalog Status**: `APPROVED_AT_DESIGN_LEVEL_PENDING_EXTERNAL_AUDIT`
- **Runtime Stage B Status**: `STAGE_B_RUNTIME_CATALOG = UNVERIFIED` (remains unverified until concrete dispatch execution on trusted host)
- **Maximum Timeout Seconds**: `60` (`MaxTimeoutSeconds = 60`)
- **Cwd Policy**: `worktree_root` (cwd strictly constrained to `"."`; directory escape, parent traversal `..`, volume/drive prefixes, or non-root working directories are strictly forbidden)
- **Execution Capability**: Pure host validation checks; does NOT accept arbitrary command, args, executable, path, or shell fragments.

### 2.2. Parameter Schema Specification (JSON Schema Draft-07)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DocReconciliationCheckParameters",
  "type": "object",
  "required": [
    "check_kind"
  ],
  "additionalProperties": false,
  "properties": {
    "check_kind": {
      "type": "string",
      "enum": [
        "git-diff-hygiene",
        "portable-relative-link-validation",
        "json-schema-draft07-validation"
      ]
    }
  }
}
```

- **Policy Invariant**: `additionalProperties = false` guarantees that any extraneous parameters (such as `command`, `args`, `executable`, `path`, or shell options) are rejected with a deterministic schema validation failure.

---

## 3. Release Guardrails & Governance

- **Release Status**: `CANDIDATE_NOT_RELEASED` (Status invariant: `NOT_RELEASED`).
- **Scope Restriction**: Authorized exclusively for documentation reconciliation across the 16 enumerated files in `allowed_scope` upon formal candidate release by the External Supervisor.
- **Production Code Isolation**: Absolutely zero production or test Go code creation or alteration (`cmd/**`, `internal/**`, `test/**`).
- **Persistence Boundary**: Absolutely zero SQLite migrations or schema modifications.
- **Fail-Closed Dependencies**: `AUTOMATIC_RESTORE = DISABLED`; `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`.
