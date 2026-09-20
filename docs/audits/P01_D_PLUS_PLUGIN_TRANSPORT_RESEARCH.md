# P01-D — CHATGPT PLUS PUBLISHED PLUGIN / REMOTE MCP TRANSPORT RESEARCH

> **Authority**: Phase 1 Upstream Proof Dossier  
> **Track**: P01-D (Re-evaluated Hypothesis)  
> **Date**: 2026-09-20  
> **Status**: COMPLETE — GAP IDENTIFIED (`GAP_REQUIRES_ADR`)  
> **Related Proposal**: [`PROPOSAL-001-PLUS-PUBLISHED-PLUGIN-TRANSPORT.md`](../proposals/PROPOSAL-001-PLUS-PUBLISHED-PLUGIN-TRANSPORT.md)  
> **Target Requirement**: FR-016 (Supervisor Tool Transport)  

---

## 1. Executive Summary & Re-Framing

This dossier re-evaluates the feasibility of using **ChatGPT Plus Web** as the primary Intelligence Plane interface.

The initial evaluation concluded that personal ChatGPT Plus is blocked because it lacks reliable, documented access to **Private Developer Mode** for local Model Context Protocol (MCP) server deployment. While that fact remains true for private development, it incorrectly assumed private Developer Mode was the *only* OpenAI-supported path.

OpenAI provides an officially supported alternative:
> **Published Plugin / Published App backed by an OpenAI-reviewed Remote MCP Server**.

Under this model:
1. The **AI Engineering Supervisor** is packaged as a real, multi-tenant product plugin and submitted to the OpenAI Plugin Directory.
2. The plugin points to a public, production **Remote MCP Gateway** that terminates OpenAI's MCP invocations.
3. The Remote Gateway authenticates the ChatGPT user via OAuth and relays domain tool invocations down to the user's **Local Supervisor Node** over an authenticated, outbound-only secure channel (e.g., WebSocket / TLS).
4. Any personal ChatGPT Plus user can install the published plugin directly from the Plugin Directory without requiring Developer Mode, workspace administrative privileges, or local firewall modifications.

Because this path introduces new deployment and architectural components (Remote MCP Gateway, outbound node relay, OpenAI publication lifecycle) not present in the frozen V1 transport baseline, the verdict is formally classified as:
```text
P01-D VERDICT: GAP_REQUIRES_ADR
```

---

## 2. Product Capability Matrix: Private MCP vs. Published Plugin/App

| Capability Dimension | Private Custom MCP (Developer Mode) | Published Plugin / App (Remote MCP) | Official Source Citation |
|---|---|---|---|
| **Developer Mode Required?** | **YES** (Must be toggled by user/admin) | **NO** for end-user installation | OpenAI Help: *Plugins in ChatGPT and Codex* |
| **Personal Plus Private Development?** | Not documented as stable / restricted to Business/Enterprise/Edu | N/A (Developed via OpenAI Platform Developer Portal) | OpenAI Help: *Developer Mode for MCP Apps* |
| **Public Distribution?** | NO (Single workspace / private testing) | **YES** (Available in Plugin Directory) | OpenAI Platform: *Plugin Submission Portal* |
| **Personal Plus Consumption?** | Flaky / experimental / account-dependent | **YES** (Officially eligible to install published plugins) | OpenAI Help: *Apps in ChatGPT* (July 9, 2026 migration) |
| **Write / State-Mutating Actions?** | Full MCP (workspace-dependent) | **YES** (Supported by action-capable published apps) | OpenAI Help: *Connected Apps & Permissions* |
| **OpenAI Review Required?** | NO (Internal workspace testing) | **YES** (Mandatory formal review prior to publication) | OpenAI Platform: *Plugin Review Guidelines* |
| **Target Infrastructure** | Local PC or Secure MCP Tunnel | Public Production HTTPS Endpoint | OpenAI Platform: *MCP Server Submission Requirements* |
| **Suitable as Frozen V1 Primary Path?** | NO (Fails on unmanaged Plus accounts) | **YES** (Supported by OpenAI, but requires architecture ADR) | Architecture Assessment |

---

## 3. Authoritative OpenAI Documentation Evidence

### 3.1 Plugins & Apps Migration Baseline
- **Source**: OpenAI Help Center — *Plugins in ChatGPT and Codex* & *Apps in ChatGPT* (`https://help.openai.com`).
- **Access Date**: 2026-09-20.
- **Key Finding**: As of July 9, 2026, OpenAI migrated the App Directory into the unified **Plugin Directory**. Plugins serve as the container for workflow capabilities, combining **Skills** (system instructions) and **Connected Apps** (integrations that connect to external accounts and execute actions).
- **Plan Eligibility**: Users across all paid plans (including ChatGPT Plus) have access to the Plugin Directory to install and invoke published plugins.
- **Action Permissions**: State-changing actions are governed by user-level authorization and explicit action confirmations ("Important actions").

### 3.2 Plugin Submission & Remote MCP Server Requirements
- **Source**: OpenAI Platform Documentation — *Building and Submitting Plugins* (`https://platform.openai.com`).
- **Access Date**: 2026-09-20.
- **Mandatory Permissions**: Submitter organization role must have `api.apps.write` (Apps Management write access).
- **Identity Verification**: Submissions require completed individual or business identity verification in the OpenAI Platform Dashboard.
- **Production Endpoint**: The MCP server backing the plugin **must be hosted on a stable, publicly accessible HTTPS endpoint**. Local tunnels (like `tunnel-client` or ephemeral localhost ports) are explicitly rejected for public directory listings.
- **Reviewer Test Package**:
  - Exactly **five positive test cases** and **three negative test cases** demonstrating reliable handling and boundary enforcement.
  - Dedicated **reviewer test credentials** (without MFA/SMS/email challenge dependencies) allowing the OpenAI review team to authenticate and execute every exposed tool.
  - Privacy policy, terms of service, and public support website.
- **Tool Hint Annotations**: Every exposed MCP tool must include explicit boolean annotations:
  - `readOnlyHint`: `true` only if the tool does not mutate any state.
  - `destructiveHint`: `true` if the tool produces difficult-to-reverse outcomes (triggers user approval prompt).
  - `openWorldHint`: `true` if accessing the open internet; `false` for private/account-bounded domain endpoints.

---

## 4. Community Evidence vs. Official Evidence

### 4.1 Observations in OpenAI Developer Community (Supplemental)
In recent 2026 developer discussions (`community.openai.com`):
- Several users on ChatGPT Plus report seeing "Developer Mode" or being able to connect local MCP endpoints.
- However, other Plus users report that the toggle is missing, tools disappear after one conversation turn, or connections throw "Resource Forbidden" errors.
- Community developers confirm that submitting a custom plugin via the Platform portal and having it approved enables consistent access on personal Plus accounts.

### 4.2 Defensible Explanation for Community Discrepancies
Why do some Plus users report working custom MCP?
1. **Phased Rollouts / Feature Flags**: OpenAI frequently tests features across random user cohorts before formal plan rollouts.
2. **Grandfathered Beta Access**: Users who activated developer features during earlier preview windows may retain experimental flags.
3. **Desktop App Codex Confusion**: The ChatGPT desktop app features a separate "Developer Mode" for Codex (Chrome DevTools Protocol debugging), which users frequently conflate with MCP Developer Mode.

> [!IMPORTANT]
> **Governance Rule**: An undocumented experimental feature flag cannot serve as the canonical foundation for production architecture. If the User's personal Plus account happens to expose Developer Mode, it is classified as an **`OPTIONAL_ACCOUNT_FAST_PATH`** for rapid testing, not the official V1 transport specification.

---

## 5. Investigation of Existing Generic Relay Plugins

Per Section 15 of Phase P01 directives, existing automation plugins were evaluated to see if a pre-existing generic webhook/relay plugin could bridge tool calls to a local host:
- **Zapier AI Actions**: Connects to pre-defined SaaS actions across 6,000+ apps. Cannot dynamically expose our 12 custom, strongly typed domain schemas (`bind_project`, `dispatch_task`, `approve_task`, etc.) directly into ChatGPT's structured tool-calling interface.
- **Make Plugin**: Manages and triggers existing Make scenarios. Introduces proprietary scenario syntax, non-deterministic table formatting, and external third-party multi-tenant relay latency.
- **Pipedream / n8n / Generic Webhooks**: No official published plugin exists in the Plugin Directory that acts as a transparent, bidirectional, type-safe MCP bridge for arbitrary local agents.
- **Prohibited Workarounds**: Domain-specific SaaS tools (GitHub, Linear, Airtable) must **not** be repurposed as an asynchronous message bus.

**Conclusion**:
```text
NO_SUITABLE_EXISTING_GENERIC_RELAY_FOUND
```
A dedicated AI Engineering Supervisor Plugin backed by a dedicated Remote MCP Gateway is required.

---

## 6. Remote MCP Gateway & Outbound Node Architecture

### 6.1 Conceptual Multi-Tenant Product Model
The published plugin will be built as a legitimate multi-tenant tool:
- Any ChatGPT user installs the "AI Engineering Supervisor" plugin.
- The user authenticates with the Supervisor cloud service via OAuth 2.0.
- The user downloads and runs the local Supervisor node on their Windows development machine.
- The local node pairs securely with their cloud account using a cryptographic device pairing token.
- When ChatGPT invokes a Supervisor tool, the Remote MCP Gateway routes the request strictly to the user's authenticated, online local node.

### 6.2 Architectural Topology

```text
┌──────────────────────────────────────────────────────────────┐
│                    INTELLIGENCE PLANE                        │
│                   ChatGPT Plus Web User                      │
└──────────────────────────────┬───────────────────────────────┘
                               │ Invokes Tool (e.g. dispatch_task)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│            OPENAI INFRASTRUCTURE (PUBLIC CLOUD)              │
│       Published "AI Engineering Supervisor" Plugin           │
└──────────────────────────────┬───────────────────────────────┘
                               │ HTTPS / JSON-RPC (MCP)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                  REMOTE MCP GATEWAY                          │
│               (Managed Cloud Service)                        │
│  - OAuth 2.0 Authorization Server / Token Validation         │
│  - Tenant & Device Registry (Pair Binding)                   │
│  - MCP Protocol Endpoint (Tool Definitions & Annotations)    │
│  - Bi-directional Request-Response Multiplexer               │
└──────────────────────────────┬───────────────────────────────┘
                               │ Outbound Authenticated TLS/WSS
                               ▲ (Initiated by Local Node)
┌──────────────────────────────────────────────────────────────┐
│               LOCAL SUPERVISOR CONTROL PLANE                 │
│                 (User Windows Workstation)                   │
│  - Local Node Agent (Connects outward to Gateway)            │
│  - Pair / Project State Store (Local SQLite)                 │
│  - Task Contract Authority & Policy Engine                   │
│  - Git Evidence Collector & Review Bundle Builder            │
│  - AOAdapter (Interacts with local AO Daemon)                │
└──────────────────────────────┬───────────────────────────────┘
                               │ REST / Loopback
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                  EXECUTION CONTROL PLANE                     │
│               Untrivial Agent Orchestrator (AO)              │
└──────────────────────────────┬───────────────────────────────┘
                               │ ConPTY
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                        PRIMARY WORKER                        │
│                   Antigravity CLI (Agy)                      │
└──────────────────────────────────────────────────────────────┘
```

### 6.3 Separation of Responsibilities

| Responsibility | Remote MCP Gateway (Public Cloud) | Local Supervisor Control Plane (User PC) |
|---|---|---|
| **Public HTTPS MCP Endpoint** | **OWNS** (Terminates OpenAI calls) | Strictly private (No open inbound ports) |
| **OAuth 2.0 Authentication** | **OWNS** (Identifies ChatGPT user) | Validates paired device token |
| **Tenant / Device Routing** | **OWNS** (Maps user to active PC) | N/A |
| **Source Code & Git Repository** | **NEVER** (Zero access to files) | **OWNS** (Reads local repo, runs Git) |
| **Task Contract Immutability** | Relays payload | **OWNS** (Validates and stores contracts) |
| **Review Authority** | Relays decision | **OWNS** (Records state transitions) |
| **AO / Agy Execution** | **NEVER** | **OWNS** (Manages local execution) |

---

## 7. Outbound Local Node Channel & Relay Technology Options

The local workstation must never expose public inbound firewall ports. The connection is strictly outbound:

```text
Local Supervisor Node ──[ Outbound WSS / TLS ]──> Remote MCP Gateway
```

### 7.1 Relay Technology Comparison

| Technology | Architectural Fit | Pros | Cons / Complexity |
|---|---|---|---|
| **Custom WebSocket Relay** (PoC & V1) | **High** | Full control over device pairing, lightweight, native JSON-RPC framing, minimal latency | Requires maintaining a simple Go/Node gateway service |
| **Cloudflare Tunnel (cloudflared)** | Medium | Robust zero-trust infrastructure, automatic TLS | Requires third-party daemon on host, harder to embed seamlessly into a single CLI installer |
| **Tailscale (Funnel / Serve)** | Medium | WireGuard security, excellent NAT traversal | Requires Tailscale account/client on user machine |
| **ngrok** | Low / PoC only | Instant setup for rapid proof | Ephemeral URLs on free tier, not suitable for production review |

**Recommendation for PoC vs V1**:
- **PoC / P01 Verification**: Use Cloudflare Tunnel or a minimal WebSocket bridge to validate remote MCP invocation from ChatGPT Web to local workstation.
- **Production V1**: Implement a lightweight, authenticated WebSocket reverse relay built into the Supervisor CLI.

---

## 8. Canonical 12-Tool Surface Mapping & Annotations

The frozen 12 high-level domain tools are mapped to OpenAI's required MCP descriptor schema with mandatory annotations:

| Tool Name | Type | `readOnlyHint` | `destructiveHint` | `openWorldHint` | Governance & Confirmation |
|---|---|---|---|---|---|
| `bind_project` | Control | `false` | `false` | `false` | Associates chat pair with a registered local project. |
| `get_project_state` | Read | `true` | `false` | `false` | Read-only inspection of active lane. No confirmation required. |
| `get_current_task` | Read | `true` | `false` | `false` | Fetches active contract and execution status. |
| `get_worker_status` | Read | `true` | `false` | `false` | Returns worker activity state. |
| `search_project` | Read | `true` | `false` | `false` | Structural metadata search. |
| `read_project_file` | Read | `true` | `false` | `false` | Read-only file inspection within project boundary. |
| `get_review_bundle` | Read | `true` | `false` | `false` | Ingests compiled review bundle. |
| `dispatch_task` | Control | `false` | `true` | `false` | State-changing: transitions task to READY/DISPATCHED. Triggers user confirmation prompt. |
| `approve_task` | Control | `false` | `true` | `false` | State-changing: marks task APPROVED. Triggers confirmation prompt. |
| `request_revision` | Control | `false` | `false` | `false` | State-changing: restarts worker turn with feedback. |
| `block_task` | Control | `false` | `false` | `false` | State-changing: halts task on blocker. |
| `get_phase_status` | Read | `true` | `false` | `false` | Reads roadmap deliverables status. |

---

## 9. Security & Isolation Model

The multi-tenant architecture enforces strict containment at four distinct boundaries:
1. **Identity Boundary**: ChatGPT user authenticates via OAuth 2.0 to the Remote Gateway. Gateway issues a scoped JWT.
2. **Device Boundary**: The local Supervisor node connects with a pre-shared, revocable device token. Requests from Tenant A cannot route to Tenant B's workstation.
3. **Pair Boundary**: Invocations must reference a valid `pair_id`. An unrecognized pair is rejected by the local node.
4. **Project / File Boundary**: The local Supervisor node enforces path validation against the registered `project_root`. Arbitrary file paths outside the root return `PERMISSION_DENIED`.

---

## 10. OpenAI Review Requirements & Burden

Publishing a plugin to the OpenAI Plugin Directory imposes formal operational requirements:
- **Developer Verification**: Verified organization in the OpenAI Platform Dashboard.
- **Review Package**:
  - Stable production HTTPS URL.
  - 5 documented positive test cases and 3 negative test cases.
  - Dedicated reviewer test credentials with pre-configured mock project data.
  - Terms of Service and Privacy Policy URLs.
- **Resubmission Lifecycle**: Any schema or tool modifications require submitting a new version through the portal.

---

## 11. Architectural Impact & Preservation of Frozen Components

### 11.1 Affected Architectural Boundary
- **`ChatGPTTransport` / `TransportAdapter`**: Expands from a simple local loopback server to include a public Remote MCP Gateway and outbound client relay adapter.

### 11.2 Frozen Components Strictly Preserved (Unaffected)
- **`TaskContract` Specification** (`docs/08_TASK_CONTRACT.md`): Unchanged.
- **`ReviewBundle` Specification** (`docs/10_REVIEW_BUNDLE.md`): Unchanged.
- **`WorkerReport` Specification** (`docs/09_WORKER_REPORT.md`): Unchanged.
- **`PolicyEngine`**: Unchanged.
- **`EvidenceCollector`**: Unchanged.
- **`AOAdapter` & AO Execution Plane** (`v0.13.0`): Unchanged.
- **Primary Worker (Antigravity CLI `1.2.7`)**: Unchanged.
- **Git Authority Model**: Unchanged.
- **ReadOnly Supervisor Authority Model**: Unchanged.

---

## 12. P01-D Conclusion & Exit Verdict

```text
================================================================================
TRACK P01-D CONCLUSION:
PRIVATE CUSTOM MCP ON PLUS: NOT RELIABLE AS PRIMARY TRANSPORT
PUBLISHED PLUGIN / REMOTE MCP: DOCUMENTED SUPPORTED PATH — PROOF REQUIRED
P01-D VERDICT: GAP_REQUIRES_ADR
================================================================================
```

A supported transport path for ChatGPT Plus Web exists without browser automation or copy-pasting. However, adopting a published plugin backed by a remote MCP gateway requires an explicit Architecture Decision Record (ADR-011) and roadmap adjustment.

Formal proposal created: [`docs/proposals/PROPOSAL-001-PLUS-PUBLISHED-PLUGIN-TRANSPORT.md`](../proposals/PROPOSAL-001-PLUS-PUBLISHED-PLUGIN-TRANSPORT.md).
