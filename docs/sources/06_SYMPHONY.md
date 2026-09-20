# SOURCE DOSSIER: 06 — OPENAI SYMPHONY

> **Authority**: Passive Design Source Evidence Dossier  
> **Status**: Verified Documentation Baseline (Post-Re-Audit #2 Hygiene)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `openai/symphony` | GitHub API |
| **Role in Architecture** | Passive Design Reference (Workflow-as-policy, authoritative orchestrator state, workspace lifecycle) | ADR-005, ADR-010, docs/08_TASK_CONTRACT.md |
| **Pinned Commit SHA** | `be10a1b79df723d6d7612b5651c8522704dafb2e` | GitHub API Verification |
| **Commit Author Date (UTC)** | `2026-09-15T22:12:07Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-09-15T22:12:07Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `Apache-2.0` | `LICENSE` in repository root |
| **Usage Terms** | Apache License 2.0 | `LICENSE` file |
| **Operational Nature** | Passive Design Source (Never imported as runtime package) | Project Governance |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: SYM-CLAIM-001
Claim: Symphony specifies an authoritative core domain model and repository-owned workflow contract for agent orchestration.
Repository: openai/symphony
Pinned tag: None
Pinned commit: be10a1b79df723d6d7612b5651c8522704dafb2e
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SPEC.md
Section / symbol: ## 4. Core Domain Model & ## 5. Workflow Specification (Repository Contract)
Verification: VERIFIED
Confidence: HIGH
Notes: Defines core entities including Issue, Workflow Definition, Service Config, Workspace, Run Attempt, Live Session, Retry Entry, and Orchestrator Runtime State.

Claim ID: SYM-CLAIM-002
Claim: Symphony specifies isolated per-issue workspaces that are deterministically created and preserved across execution runs.
Repository: openai/symphony
Pinned tag: None
Pinned commit: be10a1b79df723d6d7612b5651c8522704dafb2e
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SPEC.md
Section / symbol: ## 9. Workspace Management and Safety
Verification: VERIFIED
Confidence: HIGH
Notes: Explicitly establishes the requirement to "Create deterministic per-issue workspaces and preserve them across runs". Workspaces are not simply ephemeral; they persist across turns for the issue.

Claim ID: SYM-CLAIM-003
Claim: Symphony establishes an orchestrator state machine governing polling, dispatch, reconciliation, and retry queues.
Repository: openai/symphony
Pinned tag: None
Pinned commit: be10a1b79df723d6d7612b5651c8522704dafb2e
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SPEC.md
Section / symbol: ## 7. Orchestration State Machine & ## 8. Polling, Scheduling, and Reconciliation
Verification: VERIFIED
Confidence: HIGH
Notes: Formalizes single-authority mutable state, stopping active runs when issue state becomes non-active, and exponential retry handling.
```

---

# 3. Adopted Concepts vs. Architectural Boundaries

### Adopted Concepts:
- Authoritative orchestrator state outside chat context.
- Workflow policy in-repo governing agent boundaries.
- Workspace lifecycle management and cross-run persistence.
- Reconciliation and bounded retry handling.

### Clarification on Ownership:
- Symphony provides workflow-policy, authoritative orchestration state, retry/reconciliation, and workspace lifecycle patterns.
- The 13-state task lifecycle in `docs/06_WORKFLOW_STATE_MACHINE.md` is **OUR architecture**, tailored specifically for ChatGPT supervision and independent Git evidence review.
