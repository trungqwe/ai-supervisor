# SOURCE DOSSIER: 09 — ANTIGRAVITY LINK EXTENSION

> **Authority**: Passive Fallback Source Evidence Dossier  
> **Status**: Verified Documentation Baseline (Post-Re-Audit #2 Hygiene)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `cafeTechne/antigravity-link-extension` | GitHub API |
| **Role in Architecture** | Passive Fallback Reference (CDP browser / GUI bridge) | Docs Governance |
| **Pinned Commit SHA** | `dae4483275acba8fff093b14bb25abe8e9495f94` | GitHub API Verification |
| **Commit Author Date (UTC)** | `2026-06-04T05:55:30Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-06-04T05:55:30Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `MIT` | `LICENSE` in repository root |
| **Usage Terms** | MIT License | `LICENSE` file |
| **Operational Nature** | Passive Fallback Source (Strictly Non-V1) | Project Governance |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: LINK-CLAIM-001
Claim: Antigravity Link Extension provides Chrome DevTools Protocol (CDP) and WebSocket bridge mechanisms for browser and MCP integration.
Repository: cafeTechne/antigravity-link-extension
Pinned tag: None
Pinned commit: dae4483275acba8fff093b14bb25abe8e9495f94
Evidence type: SOURCE_CODE
Exact evidence: src/services/cdp.ts, src/server/index.ts, mcp-server.mjs
Section / symbol: cdpService / createServer
Verification: VERIFIED
Confidence: HIGH
Notes: Real codebase provides WebSocket / CDP connectivity for browser extension automation. Retained strictly as fallback; not part of V1 architecture (NFR-002).
```

---

# 3. Architectural Boundary Notice

In accordance with `NFR-002` (Zero Desktop GUI Automation) and `ADR-004` (Pure API Integration), this repository is **NOT** an active runtime dependency. It is preserved strictly as an audited fallback design reference.
