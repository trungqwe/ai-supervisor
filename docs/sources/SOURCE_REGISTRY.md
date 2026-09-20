# SOURCE REGISTRY — MASTER EVALUATED TECHNOLOGIES INDEX

> **Authority**: Central Technology Audit Register  
> **Status**: Approved Baseline for Phase 0

---

# 1. Active Upstream Dependencies (Runtime Systems)

| ID | Technology | Repository | Category | Pinned Version / Commit | License | Role in Our Architecture |
|---|---|---|---|---|---|---|
| **SRC-01** | Untrivial Agent Orchestrator | `Untrivial-ai/agent-orchestrator` | Active Upstream | `v0.13.0` (Windows Release) | Apache-2.0 | Execution Control Plane: Process daemon, Windows ConPTY terminal, Git worktree isolation, session persistence. |
| **SRC-02** | Official Antigravity CLI | `google/antigravity` (Binary Distribution) | Active Worker | `official-latest` | Proprietary Google / Terms of Service | Primary Worker Interface: Autonomous code editing, testing, compilation, and commit generation inside isolated worktrees. |

---

# 2. Passive Design Sources (Architectural & Concept References)

| ID | Technology | Repository | Category | Reference Tag / Commit | License | Adopted Concept |
|---|---|---|---|---|---|---|
| **SRC-03** | Proxide | `tt-a1i/proxide` | Design Source | `main` (`2026-09-10`) | MIT | Readonly default security, explicit project roots, strict containment. |
| **SRC-04** | Mieruko Workbench | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | Design Source | `main` (`2026-09-10`) | MIT | Workspace binding, slim tool surface philosophy, Review UI concepts. |
| **SRC-05** | AIWorkHub | `shrec/AIWorkHub` | Design Source | `main` (`2026-09-12`) | MIT | Evidence-first review, candidate result vs. verified truth, durable task memory. |
| **SRC-06** | OpenAI Symphony | `openai/symphony` | Design Source | `main` (`2026-09-08`) | Apache-2.0 | Workflow-as-policy, isolated implementation runs, task lifecycle contracts. |
| **SRC-07** | Codencer | `lookmanrays/codencer` | Design Source | `main` (`2026-09-05`) | MIT | Bridge-not-brain philosophy, state-outside-chat, attempts and evidence vocabulary. |
| **SRC-08** | AWS CLI Agent Orchestrator | `awslabs/cli-agent-orchestrator` | Design Source | `main` (`2026-08-30`) | Apache-2.0 | Provider/runtime separation patterns and event notification models. |
| **SRC-09** | Antigravity Link | `antigravity-link` (Community Tool) | Fallback Only | `reference-snapshot` | MIT | CDP browser/GUI bridge (retained strictly as fallback if CLI fails). |

---

# 3. Source Dossier Navigation
Each technology is audited in depth with exact claims, adopted patterns, rejected parts, and source evidence in dedicated dossier files:
- [`01_AGENT_ORCHESTRATOR.md`](01_AGENT_ORCHESTRATOR.md)
- [`02_ANTIGRAVITY_CLI.md`](02_ANTIGRAVITY_CLI.md)
- [`03_PROXIDE.md`](03_PROXIDE.md)
- [`04_MIERUKO.md`](04_MIERUKO.md)
- [`05_AIWORKHUB.md`](05_AIWORKHUB.md)
- [`06_SYMPHONY.md`](06_SYMPHONY.md)
- [`07_CODENCER.md`](07_CODENCER.md)
- [`08_AWS_CAO.md`](08_AWS_CAO.md)
- [`09_ANTIGRAVITY_LINK.md`](09_ANTIGRAVITY_LINK.md)
