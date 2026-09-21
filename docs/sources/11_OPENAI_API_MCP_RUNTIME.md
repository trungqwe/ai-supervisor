# 11. OPENAI API & MCP RUNTIME ARCHITECTURAL DOSSIER

> **Authority**: Official OpenAI API, Model & MCP Transport Runtime Specification  
> **Status**: Official Upstream Product Ingestion (Phase P01-D3A)  
> **Ingestion Date**: 2026-09-21  
> **Official Sources Ingested**:
> - `https://developers.openai.com/api/docs/guides/secure-mcp-tunnels`
> - `https://developers.openai.com/api/docs/guides/tools-connectors-mcp`
> - `https://developers.openai.com/api/docs`
> - `https://developers.openai.com/api/docs/guides/text`
> - `https://developers.openai.com/api/docs/models/gpt-5.6-sol`
> - `https://developers.openai.com/api/docs/guides/agents/quickstart`
> - `https://developers.openai.com/learn/developers-codex-plugin`

---

# 1. Architectural Role & Classification Framework

To protect the canonical architecture from implicit dependency creep while enabling empirical validation, every component in this runtime dossier is classified into one of four immutable governance tiers:

1. **`CANONICAL_CURRENT_TRANSPORT`**: Upstream transport or protocol actively used for Phase P01-D empirical validation.
2. **`SUPPORTING_REFERENCE`**: Authoritative upstream reference material providing context or secondary mechanisms.
3. **`RUNTIME_REFERENCE_ONLY`**: Ephemeral or pricing-sensitive product features (e.g. specific model snapshots) that can be used for execution testing but must NOT form an architectural dependency.
4. **`NOT_CURRENT_DEPENDENCY`**: Upstream SDKs, frameworks, or agents libraries explicitly evaluated and excluded from production architecture.

---

# 2. Upstream Subsystem Evaluations

## 2.1 Secure MCP Tunnel
- **Classification**: **`CANONICAL_CURRENT_TRANSPORT`** (`UPSTREAM — OpenAI`)
- **Core Architecture Facts**:
  - **Outbound-Only Ingress**: Operates entirely via outbound HTTPS long-polling (`/v1/tunnel/*`) initiated by the customer-run `tunnel-client` from inside the local network.
  - **Zero Firewall Ingress**: Eliminates any requirement for inbound open Windows firewall ports, router NAT forwarding, dynamic DNS, or public IP addresses.
  - **Private Endpoint Scope**: The local MCP server (e.g. `http://127.0.0.1:<port>/mcp` or stdio) remains strictly private and inaccessible from the public internet.
  - **Tunnel Addressing**: Managed via an OpenAI-hosted tunnel identifier (`tunnel_id`).
  - **Supported Surfaces**: Supported directly by the Responses API, Codex CLI/host, and ChatGPT developer-mode apps.
  - **Governance Separation**: Platform tunnel permissions (organization-scoped Tunnels `Read`, `Use`, `Manage`) are strictly independent of ChatGPT workspace developer-mode policies.
  - **Non-Replacement Warning**: Secure MCP Tunnel does **NOT** replace the stable public HTTPS MCP endpoint required for public plugin submission or store distribution.
- **Anti-Reinvention Directive**: Treat Secure MCP Tunnel and official `tunnel-client` as an external upstream capability. **DO NOT REIMPLEMENT** custom reverse WebSocket relays or HTTP tunnels.

## 2.2 Responses API MCP Tool Invocation
- **Classification**: **`CANONICAL_CURRENT_TRANSPORT`** (`UPSTREAM — OpenAI Product Capability`)
- **Core Architecture Facts**:
  - **Tool Type**: First-class `"type": "mcp"` in the Responses API (`v1/responses`).
  - **Tunnel Binding**: Ingests tunnel-backed MCP tools directly by specifying `tunnel_id: "tunnel_..."` instead of `server_url`. (The `server_url` parameter is reserved exclusively for publicly routable endpoints).
  - **Tool Discovery**: Responses API issues an internal `tools/list` RPC to retrieve tool definitions, producing an `mcp_list_tools` item in the response context.
  - **Tool Execution**: Calling tools produces `mcp_call` containing inputs and tool outputs.
  - **Approval Policy Safety Boundary**: Supports `require_approval` settings (`"always"`, `"never"`, or granular tool-name filters). When triggered, an `mcp_approval_request` item is generated, requiring a subsequent `mcp_approval_response` with `previous_response_id`.
  - **Separation of Safety Layers**: Responses API approval controls data egress at the model transport layer. This must not be conflated with the Supervisor's domain-level Task Contract and Git audit approval.

## 2.3 Generic API & Text / Responses Guides
- **Classification**: **`SUPPORTING_REFERENCE`**
- **Core Architecture Facts**:
  - Documents standard conversation state, session chaining (`previous_response_id`), and output streaming.
  - Confirms standard error models, authentication patterns, and rate-limiting behaviors.

## 2.4 Agents SDK
- **Classification**: **`NOT_CURRENT_DEPENDENCY`**
- **Core Architecture Facts**:
  - High-level multi-agent orchestration framework (`openai-agents` / AgentKit).
  - **Governance Decision**: The AI Engineering Supervisor Control Plane implements its own authoritative state machine, task contracts, pair registry, and policy engine. Introducing the Agents SDK would create architectural bloat and duplicate the Supervisor's core responsibilities.
  - **Prohibition**: Do NOT introduce the Agents SDK into production architecture unless evaluated in a formal ADR.

## 2.5 OpenAI Developers Codex Plugin Reference
- **Classification**: **`SUPPORTING_REFERENCE`**
- **Core Architecture Facts**:
  - Explains local developer interaction patterns between Codex and ChatGPT web.
  - Serves as a reference design for local agent tooling.

## 2.6 Model Reference: GPT-5.6 Sol (`gpt-5.6-sol`)
- **Classification**: **`RUNTIME_REFERENCE_ONLY`**
- **Core Architecture Facts**:
  - Flagship model with extensive context window and native MCP support.
  - **Dependency Prohibition**: Model IDs, pricing ($4/M input, $20/M output), promotional terms, and availability are mutable product attributes. GPT-5.6 Sol must **NOT** become a hard architectural dependency.
  - **Test-Model Policy for Phase P01-D**: Use the most inexpensive MCP-capable model available on the user's Platform account (e.g. `gpt-4o-mini`, `gpt-4.1-mini`, or equivalent) to keep validation strictly within the $1 spending boundary. If a specific model is gated behind pending identity verification, record `MODEL_ACCESS_BLOCKED_BY_IDENTITY` rather than transport failure.

---

# 3. Source Priority Hierarchy for Track P01-D

```text
PRIMARY / CANONICAL (Binding for P01-D Transport)
  ├── Plugin Architecture & Build Specifications
  ├── Plugin MCP Server Specification
  ├── Plugin Submission & Review Requirements
  ├── Secure MCP Tunnel Specification & Official tunnel-client
  └── Responses API MCP Invocation (via tunnel_id)
        │
SUPPORTING REFERENCES (Context & Protocol Guidance)
  ├── Plugin Skills Specification
  ├── General API & Responses Guides
  └── OpenAI Developers Codex Plugin Documentation
        │
REFERENCE ONLY / NOT DEPENDENCIES (Zero Production Coupling)
  ├── Agents SDK Quickstart (NOT_CURRENT_DEPENDENCY)
  └── GPT-5.6 Sol Model Page (RUNTIME_REFERENCE_ONLY)
```
