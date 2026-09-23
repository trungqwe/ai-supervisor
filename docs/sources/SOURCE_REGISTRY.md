# SOURCE REGISTRY & UPSTREAM REPOSITORY REGISTER

> **Authority**: Comprehensive Repository Register & Architectural Provenance  
> **Status**: Approved Documentation Baseline (Post-Re-Audit #2 Hygiene)  
> **Date Convention**: All commit timestamps are explicitly recorded in ISO-8601 UTC format (`YYYY-MM-DDTHH:MM:SSZ`) separating Author Date from Committer Date.  
> **License Convention**: Repository SPDX License is explicitly distinguished from Product / Usage Terms.

---

# 1. Master Repository Register

| Index | Technology | Repository Name | Role | Pinned Version / Tag | Pinned Commit SHA (40-char) | Commit Author Date (UTC) | Commit Committer Date (UTC) | Repository SPDX License | Usage Terms | Dossier Link |
|---|---|---|---|---|---|---|---|---|---|---|
| **01** | **Agent Orchestrator** | `Untrivial-ai/agent-orchestrator` | Active Upstream (Runtime) | `v0.13.0` | `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` | `2026-09-12T06:30:25Z` | `2026-09-12T06:30:25Z` | `Apache-2.0` | Apache License 2.0 | [01_AGENT_ORCHESTRATOR.md](01_AGENT_ORCHESTRATOR.md) |
| **02** | **Antigravity CLI** | `google-antigravity/antigravity-cli` | Active Upstream (Worker) | `1.2.7` | `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3` | `2026-09-19T01:01:48Z` | `2026-09-19T01:01:48Z` | `NOT DECLARED` | Subject to applicable Google / Antigravity Terms of Service | [02_ANTIGRAVITY_CLI.md](02_ANTIGRAVITY_CLI.md) |
| **03** | **Proxide** | `tt-a1i/proxide` | Passive Design Source | Pinned Commit | `c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee` | `2026-06-20T15:17:16Z` | `2026-06-20T15:17:16Z` | `MIT` | MIT License | [03_PROXIDE.md](03_PROXIDE.md) |
| **04** | **Mieruko Workbench** | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | Passive Design Source | Pinned Commit | `4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df` | `2026-09-19T20:17:06Z` | `2026-09-19T20:17:06Z` | `MIT` | MIT License | [04_MIERUKO.md](04_MIERUKO.md) |
| **05** | **AIWorkHub** | `shrec/AIWorkHub` | Passive Design Source | Pinned Commit | `19c8ce548c27316a06117c0638b4f853e00292b8` | `2026-09-20T06:48:06Z` | `2026-09-20T06:48:06Z` | `MIT` | MIT License | [05_AIWORKHUB.md](05_AIWORKHUB.md) |
| **06** | **OpenAI Symphony** | `openai/symphony` | Passive Design Source | Pinned Commit | `be10a1b79df723d6d7612b5651c8522704dafb2e` | `2026-09-15T22:12:07Z` | `2026-09-15T22:12:07Z` | `Apache-2.0` | Apache License 2.0 | [06_SYMPHONY.md](06_SYMPHONY.md) |
| **07** | **Codencer** | `lookmanrays/codencer` | Passive Design Source | Pinned Commit | `8d4908b1acf049cd97a9c4dcb752d1e44cbb8655` | `2026-08-06T14:35:56Z` | `2026-08-06T14:35:56Z` | `Apache-2.0` | Apache License 2.0 | [07_CODENCER.md](07_CODENCER.md) |
| **08** | **AWS CAO** | `awslabs/cli-agent-orchestrator` | Passive Design Source | Pinned Commit | `156cf1edcdb9de1ee01a2f17c2a4026e5f305e56` | `2026-09-20T05:52:34Z` | `2026-09-20T05:52:34Z` | `Apache-2.0` | Apache License 2.0 | [08_AWS_CAO.md](08_AWS_CAO.md) |
| **09** | **Antigravity Link** | `cafeTechne/antigravity-link-extension` | Passive Fallback Reference | Pinned Commit | `dae4483275acba8fff093b14bb25abe8e9495f94` | `2026-06-04T05:55:30Z` | `2026-06-04T05:55:30Z` | `MIT` | MIT License | [09_ANTIGRAVITY_LINK.md](09_ANTIGRAVITY_LINK.md) |

---

# 2. Upstream Governance & Update Policy

1. **Active Upstreams (`01`, `02`)**:
   - Pinned strictly to verified documentation baselines in Phase 0.
   - Upstream behavioral assertions must be empirically proven in Phase P01 on Windows host.
   - Version updates require an approved Architecture Decision Record (ADR).

2. **Passive Sources (`03` through `09`)**:
   - Fixed architectural and pattern references.
   - Never updated or tracked as moving package dependencies.

---

# 3. Mutable Official Product Documentation Register

> **Governance Notice**: Unlike pinned Git repositories, official vendor product documentation is subject to live updates by OpenAI.  
> This register indexes authoritative documentation surfaces consulted for the ChatGPT Plus / MCP transport architecture without fabricating static Git commit SHAs.

| Index | Source Family | Official Domain / URL | Architectural Role | Access Date | Volatility | Runtime Proof Status | Dossier |
|---|---|---|---|---|---|---|---|
| **DOC-01** | **OpenAI Plugins** | `https://developers.openai.com/plugins/quickstart` | Primary Architecture / Concepts | 2026-09-21 | High (Live Platform) | EVALUATED | [10_OPENAI_PLUGIN_PLATFORM.md](10_OPENAI_PLUGIN_PLATFORM.md) |
| **DOC-02** | **OpenAI Plugins** | `https://developers.openai.com/plugins/build/skills` | Supporting / Workflow Context | 2026-09-21 | Medium (SEP-2640 Draft) | EVALUATED | [10_OPENAI_PLUGIN_PLATFORM.md](10_OPENAI_PLUGIN_PLATFORM.md) |
| **DOC-03** | **OpenAI Plugins** | `https://developers.openai.com/plugins/build/mcp-server` | Canonical Transport / Tool Runtime | 2026-09-21 | Medium (MCP Spec Alignment)| PROVEN_LOCAL / P01-D3A_IN_PROGRESS | [10_OPENAI_PLUGIN_PLATFORM.md](10_OPENAI_PLUGIN_PLATFORM.md) |
| **DOC-04** | **OpenAI Plugins** | `https://developers.openai.com/plugins/build/plugins` | Primary Architecture / Packaging | 2026-09-21 | Medium (Platform Lifecycle) | EVALUATED | [10_OPENAI_PLUGIN_PLATFORM.md](10_OPENAI_PLUGIN_PLATFORM.md) |
| **DOC-05** | **OpenAI API** | `https://developers.openai.com/api/docs` | Supporting / General Reference | 2026-09-21 | Low (Core API Platform) | EVALUATED | [11_OPENAI_API_MCP_RUNTIME.md](11_OPENAI_API_MCP_RUNTIME.md) |
| **DOC-06** | **OpenAI Codex** | `https://developers.openai.com/learn/developers-codex-plugin` | Reference / Local Pattern | 2026-09-21 | Medium (Product Guide) | EVALUATED | [11_OPENAI_API_MCP_RUNTIME.md](11_OPENAI_API_MCP_RUNTIME.md) |
| **DOC-07** | **OpenAI API** | `https://developers.openai.com/api/docs/guides/text` | Supporting / Responses Guide | 2026-09-21 | Low (API Guide) | EVALUATED | [11_OPENAI_API_MCP_RUNTIME.md](11_OPENAI_API_MCP_RUNTIME.md) |
| **DOC-08** | **OpenAI Agents** | `https://developers.openai.com/api/docs/guides/agents/quickstart` | Reference Only (Not Dependency) | 2026-09-21 | High (Evolving SDK) | EXCLUDED_FROM_PRODUCTION | [11_OPENAI_API_MCP_RUNTIME.md](11_OPENAI_API_MCP_RUNTIME.md) |
| **DOC-09** | **OpenAI Models** | `https://developers.openai.com/api/docs/models/gpt-5.6-sol` | Runtime Reference Only | 2026-09-21 | High (Model Lifecycle/Pricing)| TEST_ONLY_EVALUATED | [11_OPENAI_API_MCP_RUNTIME.md](11_OPENAI_API_MCP_RUNTIME.md) |
| **DOC-10** | **OpenAI Transport**| `https://developers.openai.com/api/docs/guides/secure-mcp-tunnels`| Canonical Upstream Transport | 2026-09-21 | Medium (Official Tunnel Client)| RUNTIME_PROOF_IN_PROGRESS (P01-D3A)| [11_OPENAI_API_MCP_RUNTIME.md](11_OPENAI_API_MCP_RUNTIME.md) |
| **DOC-11** | **OpenAI Transport**| `https://developers.openai.com/api/docs/guides/tools-connectors-mcp`| Canonical Tool Protocol (Responses) | 2026-09-21 | Medium (Responses MCP API) | RUNTIME_PROOF_IN_PROGRESS (P01-D3A)| [11_OPENAI_API_MCP_RUNTIME.md](11_OPENAI_API_MCP_RUNTIME.md) |
| **DOC-12** | **OpenAI Deployment**| `https://developers.openai.com/plugins/deploy/submission` | Canonical Deployment Specification | 2026-09-21 | Medium (Platform Portal) | GATED_PENDING_IDENTITY_VERIF | [10_OPENAI_PLUGIN_PLATFORM.md](10_OPENAI_PLUGIN_PLATFORM.md) |
| **DOC-13** | **OpenAI Deployment**| `https://developers.openai.com/plugins/deploy/app-review` | Canonical Security / Review Rules | 2026-09-21 | Medium (Review Guidelines) | GATED_PENDING_IDENTITY_VERIF | [10_OPENAI_PLUGIN_PLATFORM.md](10_OPENAI_PLUGIN_PLATFORM.md) |

## 4. Execution budget source boundary (ADR-016 addendum)

Không thêm upstream hoặc thay pinned AO trong quyết định execution budget. Nguồn AO đã đăng ký chỉ cung cấp status theo session và `/kill` theo `sessionId`; không cung cấp actual execution-start timestamp, policy/deadline bền vững hay one-use kill permit. `dispatch_operations.confirmed_at` là provenance HTTP 200 do Supervisor Store ghi, không phải thuộc tính AO về lúc bắt đầu execution. Xem ADR-016 addendum execution budget §§1, 4–5; host admission/quiescence và historical policy evidence là dependency do host/operator cung cấp, không suy từ upstream.
