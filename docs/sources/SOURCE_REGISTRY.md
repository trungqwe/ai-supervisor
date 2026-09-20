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
