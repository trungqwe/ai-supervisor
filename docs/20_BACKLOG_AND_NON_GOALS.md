# 20. BACKLOG & NON-GOALS SPECIFICATION

> **Focus**: Explicit V1 Exclusions & Future Evolution Backlog  
> **Status**: Approved Baseline

---

# 1. Explicit V1 Non-Goals (Scope Defenses)

The following capabilities are **STRICTLY EXCLUDED** from Phases P00 through P06. No agent may implement or scaffold these features without an approved ADR and revised roadmap:

1. **100-Task Dependency Scheduler**: V1 executes a single active task per Pair. Complex DAG scheduling is deferred.
2. **Distributed Cloud Worker Fleets**: No remote worker provisioning or Kubernetes cluster management.
3. **Enterprise Message Brokers**: No Redis, Kafka, RabbitMQ, or Celery. All communication is local loopback.
4. **Autonomous ChatGPT Wakeup / Polling**: The Supervisor AI is conversational; we do not build proactive background polling loops for ChatGPT Web.
5. **Dynamic Model Routing / Quota Optimizer**: No real-time cost arbitrage or dynamic model switching during execution.
6. **Multi-Agent Debate Protocols**: Coding agents do not debate each other; ChatGPT alone holds supervisory authority.
7. **OS Desktop / Browser Automation**: No mouse, keyboard, or screenshot automation (ADR-004).
8. **Supervisor Source Code Editing**: ChatGPT and Supervisor never directly edit application source files (ADR-005).
9. **Generic Third-Party Plugin Framework**: No arbitrary plugin marketplace or runtime code loading.

---

# 2. Future Evolution Backlog (Post-V1 Candidates)

- **Phase P07**: Multi-Pair Visual Control Dashboard.
- **Worker Harness Expansion**: Official support for Codex CLI, Claude Code, and Cursor CLI via AO harness adapters.
- **Automated Task DAGs**: Graph-based task dependencies for complex multi-module migrations.
- **Git Worktree Branch Merging**: Automated PR creation and local branch merging upon approval.
- **Local Language Model Supervision**: Supporting high-reasoning open weights models (e.g., Llama 3.1 405B, Qwen 2.5) as local supervisors.
