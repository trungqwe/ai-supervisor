# 10. OPENAI PLUGIN PLATFORM ARCHITECTURAL DOSSIER

> **Authority**: Official OpenAI Plugin Platform Specification & Architectural Ingestion Dossier  
> **Status**: Official Upstream Product Ingestion (Phase P01-D3A)  
> **Ingestion Date**: 2026-09-21  
> **Official Sources Ingested**:
> - `https://developers.openai.com/plugins/quickstart`
> - `https://developers.openai.com/plugins/build/skills`
> - `https://developers.openai.com/plugins/build/mcp-server`
> - `https://developers.openai.com/plugins/build/plugins`
> - `https://developers.openai.com/plugins/deploy/submission`
> - `https://developers.openai.com/plugins/deploy/app-review`

---

# 1. Executive Architectural Summary

OpenAI Plugins extend and customize ChatGPT and Codex across a shared, universal product directory. An official plugin is composed of:
1. **An MCP Server** (`With MCP`): Exposes live tools, schemas, and actions over Model Context Protocol (MCP).
2. **Skills** (`Skills only` or combined): Packages structured markdown instructions (`SKILL.md`), triggers, templates, and references.
3. **Both**: An MCP server combined with uploaded or server-advertised static skills.

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                     UNIVERSAL PLUGIN DIRECTORY                          │
│                        (ChatGPT + Codex)                                │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                 ┌───────────────────┴───────────────────┐
                 ▼                                       ▼
    ┌─────────────────────────┐             ┌─────────────────────────┐
    │       MCP SERVER        │             │         SKILLS          │
    │   (Runtime Execution)   │             │   (Behavioral Context)  │
    ├─────────────────────────┤             ├─────────────────────────┤
    │ • Live data queries     │             │ • Workflow guidance     │
    │ • Authenticated actions │             │ • Decision points       │
    │ • Task dispatch/mutations│            │ • Structured templates  │
    │ • Formal JSON schemas   │             │ • Policy references     │
    │ • Safety annotations    │             │ • Static snapshots      │
    └─────────────────────────┘             └─────────────────────────┘
```

---

# 2. Functional Allocation & Boundaries

### 2.1 MCP Responsibility
- **State & Actions**: MCP provides the authoritative runtime mechanism for retrieving live operational data, authenticating callers, authorizing state changes, dispatching tasks, and returning structured data.
- **Contract Integrity**: MCP tools enforce explicit input schemas (JSON Schema / Zod), output schemas, and semantic annotations.
- **Security Boundaries**: The MCP server is responsible for server-side authorization on every tool call. The model's decision to call a tool never substitutes for server-side access control.

### 2.2 Skills Responsibility
- **Instructions & Templates**: Skills package repeatable workflow instructions, step-by-step guidance, decision matrices, and file templates.
- **Static Snapshot Semantics**: Skills imported via MCP (**Scan Tools**) or uploaded via bundle are static snapshots captured at submission time. They do not update live at runtime.
- **Non-Replacement Rule**: Skills **CANNOT** and **DO NOT** replace MCP servers for dynamic runtime state, external process orchestration, filesystem mutation, or verified evidence auditing.

### 2.3 Skills Decision for AI Engineering Supervisor V1
- **Working Hypothesis**:
  - **MCP**: Required and canonical for all Supervisor tool surfaces (`supervisor_probe_read`, `supervisor_probe_write`, task contract dispatch, review bundle retrieval).
  - **Skills**: Optional. While a future skill could provide conversational guidelines for task formulation, it must never duplicate or bypass server-side `TaskContractManager` or `PolicyEngine` invariants.
- **V1 Recommendation**: Retain Supervisor V1 as **MCP-Only** (`With MCP`). Do not introduce unnecessary packaging dependencies or duplicate policy governance into static prompt skills during transport validation.

---

# 3. Platform Submission & Publication Governance

Empirical platform inspection and official deployment specifications establish the following binding publication gates:

### 3.1 Organization Roles & Apps Management
- Public plugin authoring and submission requires explicit Platform permissions labeled **Apps Management**:
  - `api.apps.write`: Required to create, edit, and submit plugin drafts.
  - `api.apps.read`: Required to inspect existing drafts and review status.
- **Platform Owner Role**: An organization member with the `Owner` role automatically holds `api.apps.read` and `api.apps.write`.

### 3.2 Publisher Identity Verification Blocker
- **Requirement**: Every public submission requires a verified developer identity (individual national identity verification or business verification) in the OpenAI Platform.
- **Verification Blocker Scope**: Pending identity verification strictly blocks:
  1. `PLUGIN_DRAFT_COMPLETION_OR_PUBLICATION_PATH`
  2. `PUBLIC_PLUGIN_SUBMISSION`
  3. `OPENAI_PLUGIN_REVIEW`
  4. `PUBLICATION`
  5. `TARGET_PLUS_INSTALLATION_OF_OUR_PLUGIN`
- **Non-Blocker Scope**: Identity verification does **NOT** block private runtime proofs, Secure MCP Tunnels, or direct Responses API invocations that the Platform account already supports.
- **State Classification**: `IDENTITY_VERIFICATION: EXTERNAL_DEPENDENCY_PENDING`.

### 3.3 Public MCP Server Endpoint Requirements
- **Stable Public Endpoint**: Public plugin submission requires a stable, publicly reachable production HTTPS endpoint (e.g. `https://mcp.domain.com/mcp`).
- **Transport Standard**: Streamable HTTP transport (typically over `/mcp`).
- **Domain Verification**: Requires verifying domain ownership via an HTTPS challenge token hosted at `https://<challenge-base-host>/.well-known/openai-apps-challenge`.
- **Private / Local Prohibition**: Public review does **not** accept a private/local-only MCP endpoint, an unverified domain, or Secure MCP Tunnel alone as the distribution endpoint. Secure MCP Tunnel is supported for developer-mode and Responses API testing, not for public marketplace distribution.

### 3.4 Tool Scan & Safety Annotations
- **Tool Scan**: The portal executes `Scan Tools` against the declared MCP endpoint to discover tool definitions, schemas, and safety annotations.
- **Annotation Standards**:
  - `readOnlyHint`: `true` only when the tool cannot alter state; `false` for any creation, mutation, execution, or log-writing action.
  - `destructiveHint`: `true` when a write tool causes irreversible or hard-to-reverse changes.
  - `openWorldHint`: `true` when accessing public internet or open-ended resources; `false` when strictly scoped to a bounded local node or workspace.
- **Security & Content Policy**: Tool responses must be strictly sanitized. Unnecessary personal identifiers, internal credentials, debug dumps, and auth secrets must not be returned in tool payloads.

---

# 4. Impact on Project Architecture

1. **Architecture Candidate Retention**:
   - Official documentation confirms that a public plugin backed by remote MCP is an official, first-class ChatGPT Plus capability.
   - However, because public distribution requires a verified publisher identity, a public HTTPS gateway, domain challenge verification, and OpenAI review approval, Architecture V2 remains strictly `ARCHITECTURE_V2_CANDIDATE`.
2. **Phase P01-D Decoupling**:
   - Validation proceeds along two distinct layers:
     - **P01-D3A (Immediate Private Runtime Proof)**: Proves that OpenAI-managed transport (Secure MCP Tunnel + Responses API) can reach and execute our private local MCP server on this actual Platform account without waiting for public review.
     - **P01-D3B / P01-D4 (Public Gateway & Plugin Publication Proof)**: Deferred until identity verification completes and production gateway design is finalized.
