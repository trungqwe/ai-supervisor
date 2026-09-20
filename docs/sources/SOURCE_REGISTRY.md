# SOURCE REGISTRY — MASTER EVALUATED TECHNOLOGIES INDEX

> **Authority**: Central Technology Audit Register  
> **Status**: Verified Documentation Baseline (Post-Remediation)

---

# 1. Active Upstream Dependencies (Runtime Systems)

| ID | Technology | Repository | Category | Pinned Tag / Full Commit SHA | License | Role in Our Architecture |
|---|---|---|---|---|---|---|
| **SRC-01** | Untrivial Agent Orchestrator | `Untrivial-ai/agent-orchestrator` | Active Upstream | `v0.13.0`<br>`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` | Apache-2.0 | Execution Control Plane: Process daemon, Windows ConPTY terminal, Git worktree isolation, session persistence. |
| **SRC-02** | Official Antigravity CLI | `google-antigravity/antigravity-cli` | Active Worker | `1.2.7`<br>`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3` | Proprietary Google | Primary Worker Interface: Autonomous code editing, testing, compilation, and structured output generation. |

---

# 2. Passive Design Sources (Architectural & Concept References)

| ID | Technology | Repository | Category | Pinned Full Commit SHA / Date | License | Adopted Concept |
|---|---|---|---|---|---|---|
| **SRC-03** | Proxide | `tt-a1i/proxide` | Design Source | `c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee`<br>(2026-06-20) | MIT | Readonly default security, explicit project roots, strict containment. |
| **SRC-04** | Mieruko Workbench | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | Design Source | `4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df`<br>(2026-09-19) | MIT | Workspace binding, slim tool surface philosophy, Review UI concepts. |
| **SRC-05** | AIWorkHub | `shrec/AIWorkHub` | Design Source | `19c8ce548c27316a06117c0638b4f853e00292b8`<br>(2026-09-20) | MIT | Evidence-first review, candidate result vs. verified truth, durable task memory. |
| **SRC-06** | OpenAI Symphony | `openai/symphony` | Design Source | `be10a1b79df723d6d7612b5651c8522704dafb2e`<br>(2026-09-15) | Apache-2.0 | Workflow-as-policy, isolated implementation runs, task lifecycle contracts (`SPEC.md`). |
| **SRC-07** | Codencer | `lookmanrays/codencer` | Design Source | `8d4908b1acf049cd97a9c4dcb752d1e44cbb8655`<br>(2026-08-06) | Apache-2.0 | Bridge-not-brain philosophy, state-outside-chat, attempts and evidence vocabulary. |
| **SRC-08** | AWS CLI Agent Orchestrator | `awslabs/cli-agent-orchestrator` | Design Source | `156cf1edcdb9de1ee01a2f17c2a4026e5f305e56`<br>(2026-09-20) | Apache-2.0 | Provider/runtime separation patterns and event notification models. |
| **SRC-09** | Antigravity Link Extension | `cafeTechne/antigravity-link-extension` | Fallback Only | `dae4483275acba8fff093b14bb25abe8e9495f94`<br>(2026-06-04) | MIT | CDP browser/GUI bridge (retained strictly as fallback if CLI fails). |

---

# 3. Source Dossier Navigation
Detailed dossiers with empirical `SOURCE EVIDENCE` records:
- [`01_AGENT_ORCHESTRATOR.md`](01_AGENT_ORCHESTRATOR.md)
- [`02_ANTIGRAVITY_CLI.md`](02_ANTIGRAVITY_CLI.md)
- [`03_PROXIDE.md`](03_PROXIDE.md)
- [`04_MIERUKO.md`](04_MIERUKO.md)
- [`05_AIWORKHUB.md`](05_AIWORKHUB.md)
- [`06_SYMPHONY.md`](06_SYMPHONY.md)
- [`07_CODENCER.md`](07_CODENCER.md)
- [`08_AWS_CAO.md`](08_AWS_CAO.md)
- [`09_ANTIGRAVITY_LINK.md`](09_ANTIGRAVITY_LINK.md)
