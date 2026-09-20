# SOURCE DOSSIER: 04 — MIERUKO WORKBENCH

> **Authority**: Passive Design Source Evidence Dossier  
> **Status**: Literal Evidence Corrected (Post-Re-Audit #4 Patch)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | GitHub API |
| **Role in Architecture** | Passive Design Reference (Conversation-to-task binding, session recovery, slim tool surface) | ADR-001, ADR-008, docs/11_CHATGPT_TOOL_SURFACE.md |
| **Pinned Commit SHA** | `4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df` | GitHub API Verification |
| **Commit Author Date (UTC)** | `2026-09-19T20:17:06Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-09-19T20:17:06Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `MIT` | `LICENSE` in repository root |
| **Usage Terms** | MIT License | `LICENSE` file |
| **Operational Nature** | Passive Design Source (Never imported as runtime package) | Project Governance |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: MIER-CLAIM-001
Claim: Mieruko implements persistent conversation-to-task binding surviving transport reconnects and UI state changes.
Repository: Mieruko/MCP_Plugins_With_ChatGPTWeb
Pinned tag: None
Pinned commit: 4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: AGENTS.md, README.md
Section / symbol:
  AGENTS.md: ## Quyền truy cập
  README.md: ## Active Agents and session recovery
Verification: VERIFIED
Confidence: HIGH
Notes: AGENTS.md under heading "## Quyền truy cập" explicitly states: "ChatGPT có _meta['openai/session'] được gắn task theo cuộc trò chuyện, độc lập kết nối MCP; reconnect và đổi Dashboard không đổi task." Cross-chat continuity is also described under "### Tiếp tục công việc qua nhiều chat". README heading "## Active Agents and session recovery" documents the session recovery model.

Claim ID: MIER-CLAIM-002
Claim: Mieruko provides administrative router infrastructure for managing upstream server connections and monitoring live activity streams.
Repository: Mieruko/MCP_Plugins_With_ChatGPTWeb
Pinned tag: None
Pinned commit: 4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df
Evidence type: SOURCE_CODE
Exact evidence: src/admin/routes.ts
Section / symbol: createAdminRouter
Verification: VERIFIED
Confidence: HIGH
Notes: The exported function begins with "export function createAdminRouter(manager: McpUpstreamManager, options: {". Implements admin endpoints for environment config, upstream MCP server registration, tool listing, activity filtering, and Server-Sent Events (SSE) streaming.

Claim ID: MIER-CLAIM-003
Claim: Mieruko advocates a slim, structured tool surface tailored for conversational LLM reasoning without unstructured terminal scraping.
Repository: Mieruko/MCP_Plugins_With_ChatGPTWeb
Pinned tag: None
Pinned commit: 4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: README.md
Section / symbol: ## MCP tools
Verification: VERIFIED
Confidence: HIGH
Notes: README section "## MCP tools" covers the tool surface. Upstream text states: "Tool responses use structured data so ChatGPT can reason over results without scraping terminal text where a structured representation is available." The previously cited heading was not found in the pinned README; the correct heading is "## MCP tools".
```

---

# 3. Adopted Concepts vs. Architectural Boundaries

### Adopted Concepts:
- Persistent conversation-to-task binding patterns across client reconnects.
- Slim, domain-level tool interface (informing our 12-tool contract in `docs/11_CHATGPT_TOOL_SURFACE.md`).
- Session activity observability and structured tool results.

### Clarification on Ownership:
- Mieruko proves the viability of conversation-level MCP session tracking.
- The formal `PairRegistry` entity in our architecture (binding 1 Project, 1 ChatGPT Session, and 1 Worker Session) is **OUR design abstraction**.
