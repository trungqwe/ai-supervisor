# 17. ROADMAP & PHASE SPECIFICATION

> **Focus**: Execution Phases P00 through P07 with Deliverables & Exit Gates
> **Status**: Approved Baseline (Remediated Phase 0)

---

# 1. Phase Roadmap Overview

| Phase | Phase Name | Primary Goal | Deliverable | Exit Gate |
|---|---|---|---|---|
| **`P00`** | Architecture Freeze | Complete technical foundation, schemas, dossiers, and audit baseline. | 25 canonical docs, schemas, ADRs, dossiers. | External Supervisor / User Approval (`ARCHITECTURE_FROZEN`). |
| **`P01`** | Upstream Proof & Transport Feasibility | Empirically verify AO daemon (P01-A), Agy CLI (P01-B), AO-Agy adapter gap (P01-C), and ChatGPT Plus transport feasibility (P01-D / FR-016) on Windows. | Upstream proof logs, adapter gap analysis, transport feasibility determination. | Tri-state exit verdict: `PASS` or `GAP_REQUIRES_ADR`. Zero progression if `BLOCKER`. |
| **`P02`** | Supervisor Domain Core | Implement domain entities, 13-state state machine, immutable Task Contract validator, and durable local State Store. | Domain models, state transition engine, unit test suite. | 100% unit test pass on domain state machine and contracts. |
| **`P03`** | AO Integration | Implement `AOAdapter` connecting Supervisor domain to AO REST daemon API, and host bootstrap integration harness (TASK-P03-004) under ADR-016/017. | AOAdapter REST integration, session dispatch saga, observation reconciliation, raw workspace transport, and host bootstrap daemon exclusivity with P03 integration harness. | Automated session creation, dispatch, observation reconciliation, raw workspace file read, and teardown pass without P04 EvidenceCollector dependencies. |
| **`P04`** | Evidence & Review Engine| Implement independent Git diff collector and Review Bundle builder. | EvidenceCollector, ReviewBundleBuilder, policy validator. | Verified diff matches actual git commits; zero unverified claims. |
| **`P05`** | ChatGPT Tool Interface | Implement the 12 high-level domain tools over proven transport without GUI automation. | ChatGPTToolSurface, TransportAdapter, schema endpoints. | Target ChatGPT Web environment successfully binds project and inspects state via tools. |
| **`P06`** | Single Pair Full Loop | Execute first complete end-to-end task cycle without human copy-paste. | Live demonstration on real code repository. | Successful cycle: Dispatch → Edit → Test → Review → Approval (Decision Recorded, No Auto-Merge). |
| **`P07`** | Multi-Pair Control UI | Develop desktop/web local dashboard for monitoring multiple pairs. | Control UI dashboard, event stream viewer. | Human developer visually monitors multiple concurrent pairs. |
