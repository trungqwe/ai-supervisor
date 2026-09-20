# PROPOSAL-001: ChatGPT Plus Published Plugin & Remote MCP Gateway Transport

## Metadata
- **Author**: Antigravity Assistant & Engineering Team
- **Date**: 2026-09-20
- **Related Phase / Task**: Phase 1 (P01-D) / FR-016
- **Status**: PENDING_REVIEW

## 1. Observation
Initial Phase 0 planning assumed that ChatGPT Web could connect to a locally hosted Model Context Protocol (MCP) server or local relay directly using "Developer Mode".
Empirical research in Phase 1 (Track P01-D) revealed that:
1. Full MCP support (including write/modify actions) in Developer Mode is restricted by OpenAI to Business, Enterprise, and Edu workspaces. Personal ChatGPT Plus accounts do not have official, reliable access to Developer Mode for private MCP servers.
2. Legacy Custom GPTs / Actions are scheduled for retirement on December 11, 2026, and creation of new Custom GPTs on personal accounts is restricted.
3. However, personal ChatGPT Plus accounts *do* have full access to install and use public plugins from the OpenAI Plugin Directory, and published plugins can be backed by a remote MCP server with write actions.

## 2. Proposed Idea
Adopt a **Published Plugin backed by a Remote MCP Gateway and Outbound Local Node Relay** as the canonical transport for personal ChatGPT Plus:
1. Package the 12 domain tools as an official "AI Engineering Supervisor" Plugin in the OpenAI Plugin Directory.
2. Deploy a lightweight, multi-tenant **Remote MCP Gateway** on a public HTTPS endpoint that handles OpenAI's MCP invocations and user OAuth 2.0 authentication.
3. The user runs a Local Supervisor Node on their development machine, which initiates an authenticated, outbound-only WebSocket connection to the Remote MCP Gateway.
4. When ChatGPT invokes a tool (e.g., `dispatch_task` or `approve_task`), the Gateway routes the call to the user's paired local node.

## 3. Problem Solved
1. Fulfills requirement `FR-016` (Supervisor Tool Transport) for personal ChatGPT Plus accounts without relying on Edu/Business workspace admin approval.
2. Completely avoids OS/browser GUI automation and manual prompt/report copy-pasting.
3. Requires zero open inbound ports or public IP addresses on the user's local Windows development machine.
4. Provides a stable, long-term, officially supported integration path that survives the December 2026 Custom GPT retirement.

## 4. Existing Upstream Capability Checked
- OpenAI Official Plugin Directory & Apps SDK (`@modelcontextprotocol/sdk`).
- `Untrivial-ai/agent-orchestrator`: Remains the execution engine on the local node.
- `google-antigravity/antigravity-cli`: Remains the primary worker on the local node.
- `tt-a1i/proxide`: Verified for reverse-proxy and outbound agent relay concepts.

## 5. Why Existing Capability is Insufficient
The existing frozen architecture (`ADR-008`) specified a local MCP relay or direct HTTP endpoint. Because OpenAI does not allow personal ChatGPT Plus accounts to connect to arbitrary private localhost MCP servers without workspace-level Developer Mode, a cloud-facing Remote Gateway is required to terminate OpenAI's HTTPS requests.

## 6. Expected Benefits vs. Implementation Cost
- **Benefits**:
  - True end-to-end automation for ChatGPT Plus users.
  - No dependency on enterprise workspace administrators.
  - Zero attack surface on the user's workstation (outbound-only connection).
  - Scalable multi-tenant architecture that can serve multiple users/devices.
- **Costs / Complexity**:
  - Adds a lightweight Remote MCP Gateway service (Go or Node.js) deployed to a cloud provider (e.g., Cloudflare Workers, Fly.io, AWS Lambda).
  - Requires one-time OpenAI Plugin submission, identity verification, and review process (5 positive + 3 negative test cases).

## 7. Architectural Impact
- Requires formal Architecture Decision Record (**ADR-011: Published Plugin & Remote MCP Gateway Transport**).
- Modifies `docs/04_ARCHITECTURE.md` Section 1.1 (`ChatGPTTransport` component boundary only).
- All other core components (`TaskContract`, `ReviewBundle`, `PolicyEngine`, `AOAdapter`, `AO`, `Agy`, Git authority) remain 100% unchanged.

## 8. Decision & Rationale
*(Filled by Supervisor / User during review)*
- **Verdict**: PENDING_REVIEW
- **Rationale**: Awaiting User and External Supervisor decision on whether to adopt ADR-011 or proceed with alternative plan upgrades.
