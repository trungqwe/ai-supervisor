# SOURCE DOSSIER: 08 — AWS CLI AGENT ORCHESTRATOR

> **Authority**: Passive Design Source Evidence Dossier  
> **Status**: Verified Documentation Baseline (Post-Re-Audit #2 Hygiene)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `awslabs/cli-agent-orchestrator` | GitHub API |
| **Role in Architecture** | Passive Design Reference (Provider abstraction pattern for agent backends) | ADR-009, docs/12_UPSTREAM_INTEGRATION.md |
| **Pinned Commit SHA** | `156cf1edcdb9de1ee01a2f17c2a4026e5f305e56` | GitHub API Verification |
| **Commit Author Date (UTC)** | `2026-09-20T05:52:34Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-09-20T05:52:34Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `Apache-2.0` | `LICENSE` in repository root |
| **Usage Terms** | Apache License 2.0 | `LICENSE` file |
| **Operational Nature** | Passive Design Source (Never imported as runtime package) | Project Governance |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: CAO-CLAIM-001
Claim: AWS CAO defines a modular provider abstraction pattern for dispatching tasks across heterogeneous CLI coding agent backends.
Repository: awslabs/cli-agent-orchestrator
Pinned tag: None
Pinned commit: 156cf1edcdb9de1ee01a2f17c2a4026e5f305e56
Evidence type: SOURCE_CODE
Exact evidence: src/cli_agent_orchestrator/providers/ & README.md
Section / symbol: Provider interface / Architecture
Verification: VERIFIED
Confidence: HIGH
Notes: Demonstrates clean separation between core dispatch orchestration and provider-specific CLI harnesses.
```

---

# 3. Adopted Concepts vs. Architectural Boundaries

### Adopted Concepts:
- Provider adapter pattern (informing our `AOAdapter` anti-corruption layer).

### Rejected Elements:
- Do NOT adopt AWS cloud-specific dependencies, AWS credentials, or DynamoDB bindings.
