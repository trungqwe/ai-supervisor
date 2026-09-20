# 17. ROADMAP & PHASE SPECIFICATION

> **Focus**: Execution Phases P00 through P07 with Deliverables & Exit Gates  
> **Status**: Approved Baseline

---

# 1. Phase Roadmap Overview

| Phase | Phase Name | Primary Goal | Deliverable | Exit Gate |
|---|---|---|---|---|
| **`P00`** | Architecture Freeze | Complete technical foundation, schemas, dossiers, and audit baseline. | 25 canonical docs, schemas, ADRs, dossiers. | External Supervisor / User Approval (`ARCHITECTURE_FROZEN`). |
| **`P01`** | Upstream Proof | Empirically verify AO daemon and Antigravity CLI headless execution on Windows. | Upstream proof test logs, verified compatibility report. | All 15 AO/Agy contract baseline checks pass on Windows. |
| **`P02`** | Supervisor Domain Core | Implement domain entities, 13-state state machine, and local State Store. | Domain models, state transition engine, unit test suite. | 100% unit test pass on domain state machine and contracts. |
| **`P03`** | AO Integration | Implement `AOAdapter` connecting Supervisor domain to AO REST daemon API. | AOAdapter, ConPTY integration, session lifecycle tests. | Automated session creation, dispatch, and teardown pass. |
| **`P04`** | Evidence & Review Engine| Implement independent Git diff collector and Review Bundle builder. | EvidenceCollector, ReviewBundleBuilder, policy validator. | Verified diff matches actual git commits; zero unverified claims. |
| **`P05`** | ChatGPT Tool Interface | Implement the 12 high-level tools via local MCP relay / API. | ChatGPTToolSurface, MCP server/client relay, schema endpoints. | ChatGPT Web successfully binds project and inspects state via tools. |
| **`P06`** | Single Pair Full Loop | Execute first complete end-to-end task cycle without human copy-paste. | Live demonstration on real code repository. | Successful cycle: Dispatch → Edit → Test → Review → Approval. |
| **`P07`** | Multi-Pair Control UI | Develop desktop/web local dashboard for monitoring multiple pairs. | Control UI dashboard, event stream viewer. | Human developer visually monitors multiple concurrent pairs. |
