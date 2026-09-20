# SOURCE DOSSIER: 05 — AIWORKHUB

> **Authority**: Passive Design Source Evidence Dossier  
> **Status**: Verified Documentation Baseline (Post-Re-Audit #2 Hygiene)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `shrec/AIWorkHub` | GitHub API |
| **Role in Architecture** | Passive Design Reference (Evidence-first review, candidate skepticism, bounded review bundles) | ADR-006, docs/10_REVIEW_BUNDLE.md |
| **Pinned Commit SHA** | `19c8ce548c27316a06117c0638b4f853e00292b8` | GitHub API Verification |
| **Commit Author Date (UTC)** | `2026-09-20T06:48:06Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-09-20T06:48:06Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `MIT` | `LICENSE` in repository root |
| **Usage Terms** | MIT License | `LICENSE` file |
| **Operational Nature** | Passive Design Source (Never imported as runtime package) | Project Governance |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: AIW-CLAIM-001
Claim: AIWorkHub establishes the principle that worker self-reports are candidate claims only and never own the acceptance verdict.
Repository: shrec/AIWorkHub
Pinned tag: None
Pinned commit: 19c8ce548c27316a06117c0638b4f853e00292b8
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: docs/QUALITY_CONTROL.md
Section / symbol: ## Five quality control layers (Layer 2: Bounded execution sandbox)
Verification: VERIFIED
Confidence: HIGH
Notes: Explicitly states "Worker self-reports are evidence only; they never own PASS/FAIL".

Claim ID: AIW-CLAIM-002
Claim: AIWorkHub formalizes manager acceptance based on independent verification of canonical inputs, exit codes, and bounded evidence bundles.
Repository: shrec/AIWorkHub
Pinned tag: None
Pinned commit: 19c8ce548c27316a06117c0638b4f853e00292b8
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: docs/QUALITY_CONTROL.md
Section / symbol: ## Five quality control layers (Layer 5: Manager acceptance and integration proof)
Verification: VERIFIED
Confidence: HIGH
Notes: Specifies "The verified manager re-reads current canonical inputs, re-runs required checks, verifies changed-path hashes and promotes only the exact approved delta... Acceptance records the deterministic verdict, reviewer disposition, rollback identity and complete bounded evidence bundle".

Claim ID: AIW-CLAIM-003
Claim: AIWorkHub defines six falsifiable quality lenses for independent evaluation.
Repository: shrec/AIWorkHub
Pinned tag: None
Pinned commit: 19c8ce548c27316a06117c0638b4f853e00292b8
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: docs/QUALITY_CONTROL.md
Section / symbol: ## Six canonical lenses
Verification: VERIFIED
Confidence: HIGH
Notes: Enumerates Correctness, Does it run, Test adequacy, Security, Code quality, and Requirements and scope as falsifiable yes/no criteria.
```

---

# 3. Adopted Concepts vs. Architectural Boundaries

### Adopted Concepts:
- Evidence-first verification (worker claims are untrusted hypotheses until verified against Git diffs and exit codes).
- Bounded Review Bundle concept (aggregating contract, claims, diff, and test evidence for single-turn review).
- Quality lenses informing our policy validation rules.

### Clarification on Ownership:
- AIWorkHub demonstrates quality control in multi-agent environments.
- The `ReviewBundleBuilder` in our architecture packages this into our specific `docs/schemas/review-bundle.schema.json` contract for ChatGPT.
