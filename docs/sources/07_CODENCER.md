# SOURCE DOSSIER: 07 — CODENCER

> **Authority**: Passive Design Source Evidence Dossier  
> **Status**: Verified Documentation Baseline (Post-Re-Audit #2 Hygiene)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `lookmanrays/codencer` | GitHub API |
| **Role in Architecture** | Passive Design Reference (Bridge-not-brain philosophy, state outside chat, execution vocabulary) | ADR-007, docs/04_ARCHITECTURE.md |
| **Pinned Commit SHA** | `8d4908b1acf049cd97a9c4dcb752d1e44cbb8655` | GitHub API Verification |
| **Commit Author Date (UTC)** | `2026-08-06T14:35:56Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-08-06T14:35:56Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `Apache-2.0` | `LICENSE` in repository root |
| **Usage Terms** | Apache License 2.0 | `LICENSE` file |
| **Operational Nature** | Passive Design Source (Never imported as runtime package) | Project Governance |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: COD-CLAIM-001
Claim: Codencer establishes the "Bridge, Not Brain" architectural philosophy separating the conversational intelligence layer from stateful execution coordination.
Repository: lookmanrays/codencer
Pinned tag: None
Pinned commit: 8d4908b1acf049cd97a9c4dcb752d1e44cbb8655
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: README.md
Section / symbol: ## Architecture & Philosophy
Verification: VERIFIED
Confidence: HIGH
Notes: Emphasizes that durable execution state, task tracking, and artifact capture must live outside volatile conversational context.

Claim ID: COD-CLAIM-002
Claim: Codencer provides a structured domain broker separating message ingestion from execution run state.
Repository: lookmanrays/codencer
Pinned tag: None
Pinned commit: 8d4908b1acf049cd97a9c4dcb752d1e44cbb8655
Evidence type: SOURCE_CODE
Exact evidence: cmd/broker/main.go & internal/domain/
Section / symbol: main / domain types
Verification: VERIFIED
Confidence: HIGH
Notes: Real codebase structures broker entry point and domain run definitions without synthetic bridge.go files.
```

---

# 3. Adopted Concepts vs. Rejected Elements

### Adopted Concepts:
- Bridge-not-brain philosophy (Supervisor manages lanes, contracts, and evidence without competing with ChatGPT's cognitive reasoning).
- State outside chat (StateStore maintains authoritative domain ground truth).
- Execution vocabulary (`attempts`, `artifacts`, `blockers`).

### Rejected Elements:
- Do NOT adopt generic unconstrained shell tools exposed to LLMs.
