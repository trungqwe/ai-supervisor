# AI Engineering Supervisor Control Plane

> **Autonomous Coordination, Independent Verification, and Architectural Governance for AI Coding Agents**

---

# 1. PROJECT OVERVIEW

## Context & Motivation
In modern AI-assisted software engineering, combining a high-reasoning conversational AI (such as **ChatGPT Web**) as the **Architect / Tech Lead** with a fast, IDE/CLI-integrated coding agent (such as **Antigravity**) as the **Worker / Implementer** produces exceptional engineering quality.

However, the manual workflow currently suffers from severe operational bottlenecks:
- Manually copy-pasting architectural prompts, constraints, and task scopes into the coding agent.
- Manually copying worker terminal logs and claims back to ChatGPT.
- Risk of coding agents wandering outside assigned task boundaries or modifying global architecture.
- Lack of independent evidence verification (believing claims without diff and test proof).

## Core Objective
The **AI Engineering Supervisor Control Plane** bridges this gap. It provides a local, reliable control plane that coordinates the engineering loop between ChatGPT (Intelligence Plane) and Antigravity (Worker) through **Untrivial Agent Orchestrator** (Execution Control Plane), eliminating manual copy-pasting without relinquishing architectural control or requiring desktop GUI automation.

```text
┌──────────────────────────────────────────────────────────────┐
│                     INTELLIGENCE PLANE                       │
│                        ChatGPT Web                           │
│     Architect • Lead Planner • Code Reviewer • Auditor       │
└──────────────────────────────┬───────────────────────────────┘
                               │ High-Level Tool Interface
                               │ (State, Contracts, Bundles)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                  SUPERVISOR CONTROL PLANE                    │
│                        (OUR PROJECT)                         │
│  Project/Pair Registry • Task Contracts • Evidence Collector │
│      Review Bundle Builder • Policy Engine • Audit Trail     │
└──────────────────────────────┬───────────────────────────────┘
                               │ Public API / Stable Adapter
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                  EXECUTION CONTROL PLANE                     │
│               Untrivial Agent Orchestrator                   │
│   Daemon • Session Lifecycle • Worktrees • Process ConPTY    │
└──────────────────────────────┬───────────────────────────────┘
                               │ Headless CLI / Harness
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                        PRIMARY WORKER                        │
│                   Official Antigravity CLI                   │
│     Autonomous Edit • Compile • Test • Fix • Git Commit      │
└──────────────────────────────────────────────────────────────┘
```

## Key Architectural Tenets
1. **Three-Plane Separation**: Clear decoupling between Intelligence (ChatGPT), Supervisor Management (Our Project), and Execution (Agent Orchestrator + Antigravity).
2. **Upstream First**: Never reinvent what Agent Orchestrator or Antigravity CLI already solve. Connect via stable adapters.
3. **Immutable Task Contracts**: Workers receive strict task boundaries with explicit allowed/forbidden scopes.
4. **Evidence Over Claims**: Worker completion reports are claims; the Supervisor independently gathers Git diffs, exit codes, and test artifacts into a **Review Bundle** for ChatGPT.
5. **State Resides Outside Chat**: Engineering state, pair bindings, and task histories persist reliably in local storage, not in transient chat logs.

---

# 2. CURRENT PROJECT STATUS

| Attribute | Value |
|---|---|
| **Current Stage** | **Phase 0 — Architecture Freeze** |
| **Current State** | `PHASE0_IN_PROGRESS` (Pre-freeze baseline established) |
| **Source Repos Inspected** | 9 repositories (2 Active Upstreams, 7 Design Sources) |
| **Governance** | Strict No-Code Enforcement in Phase 0; External Audit Gate |
| **GitHub Repository** | [https://github.com/trungqwe/ai-supervisor](https://github.com/trungqwe/ai-supervisor) |

---

# 3. CANONICAL DOCUMENTATION INDEX

All canonical specifications and architecture documents are organized in [`docs/`](docs/):

### Core Architecture & Specifications (25 Canonical Docs)
1. [`docs/00_PROJECT_OVERVIEW.md`](docs/00_PROJECT_OVERVIEW.md) — Canonical Project Overview & Philosophy
2. [`docs/01_PROJECT_CHARTER.md`](docs/01_PROJECT_CHARTER.md) — Mission, Problem, Goals, & Principles
3. [`docs/02_REQUIREMENTS.md`](docs/02_REQUIREMENTS.md) — Functional, Non-Functional, Security & Ops Requirements
4. [`docs/03_SYSTEM_CONTEXT.md`](docs/03_SYSTEM_CONTEXT.md) — System Context Diagram, Actors & Trust Boundaries
5. [`docs/04_ARCHITECTURE.md`](docs/04_ARCHITECTURE.md) — 3-Plane Architecture, Data/Control/Review Flows
6. [`docs/05_DOMAIN_MODEL.md`](docs/05_DOMAIN_MODEL.md) — Domain Entities, Aggregates & Value Objects
7. [`docs/06_WORKFLOW_STATE_MACHINE.md`](docs/06_WORKFLOW_STATE_MACHINE.md) — 13-State Workflow State Machine
8. [`docs/07_SECURITY_MODEL.md`](docs/07_SECURITY_MODEL.md) — Readonly Defaults, Containment & Threat Model
9. [`docs/08_TASK_CONTRACT.md`](docs/08_TASK_CONTRACT.md) — Task Contract Specification & Immutability
10. [`docs/09_WORKER_REPORT.md`](docs/09_WORKER_REPORT.md) — Worker Report Contract (Claims vs Proof)
11. [`docs/10_REVIEW_BUNDLE.md`](docs/10_REVIEW_BUNDLE.md) — Review Bundle Builder & Correlation Engine
12. [`docs/11_CHATGPT_TOOL_SURFACE.md`](docs/11_CHATGPT_TOOL_SURFACE.md) — 12 High-Level Supervisor Tools
13. [`docs/12_UPSTREAM_INTEGRATION.md`](docs/12_UPSTREAM_INTEGRATION.md) — AOAdapter & Agy Fallback Adapter
14. [`docs/13_UPSTREAM_UPDATE_POLICY.md`](docs/13_UPSTREAM_UPDATE_POLICY.md) — Upstream Upgrade Policy & Contract Diff
15. [`docs/14_FAILURE_RECOVERY.md`](docs/14_FAILURE_RECOVERY.md) — 15 Failure Scenarios & Recovery Playbooks
16. [`docs/15_OBSERVABILITY.md`](docs/15_OBSERVABILITY.md) — Event Taxonomy & Sanitized Audit Trail
17. [`docs/16_TEST_STRATEGY.md`](docs/16_TEST_STRATEGY.md) — Multi-Tier Test Strategy (Unit to E2E Loop)
18. [`docs/17_ROADMAP.md`](docs/17_ROADMAP.md) — Phases P00 through P07
19. [`docs/18_CURRENT_STATE.md`](docs/18_CURRENT_STATE.md) — Dynamic Project State Tracking
20. [`docs/19_OPEN_QUESTIONS.md`](docs/19_OPEN_QUESTIONS.md) — Unresolved Questions (`UNDECIDED / GIẢ ĐỊNH`)
21. [`docs/20_BACKLOG_AND_NON_GOALS.md`](docs/20_BACKLOG_AND_NON_GOALS.md) — Explicit V1 Non-Goals
22. [`docs/21_TRACEABILITY_MATRIX.md`](docs/21_TRACEABILITY_MATRIX.md) — End-to-End Requirement Traceability
23. [`docs/22_MODULE_PROVENANCE.md`](docs/22_MODULE_PROVENANCE.md) — Module Provenance & Anti-Reinvention Justification
24. [`docs/23_COMPATIBILITY_MATRIX.md`](docs/23_COMPATIBILITY_MATRIX.md) — Platform & Upstream Compatibility Matrix
25. [`docs/24_CHANGE_GOVERNANCE.md`](docs/24_CHANGE_GOVERNANCE.md) — Change Intake Pipeline & 9-Level Decision Hierarchy

### Technology Research & Anti-Reinvention Dossiers
- [`docs/sources/SOURCE_REGISTRY.md`](docs/sources/SOURCE_REGISTRY.md) — Evaluated Technologies Index
- [`docs/sources/REUSE_MATRIX.md`](docs/sources/REUSE_MATRIX.md) — Reuse Matrix with Exact Source Locations & Proof
- [`docs/sources/UPSTREAM_CONTRACT_BASELINE.md`](docs/sources/UPSTREAM_CONTRACT_BASELINE.md) — Contract Baselines for AO & Agy
- **9 Detailed Source Dossiers**:
  - [`01_AGENT_ORCHESTRATOR.md`](docs/sources/01_AGENT_ORCHESTRATOR.md) (Active Upstream)
  - [`02_ANTIGRAVITY_CLI.md`](docs/sources/02_ANTIGRAVITY_CLI.md) (Active Worker Interface)
  - [`03_PROXIDE.md`](docs/sources/03_PROXIDE.md) (Design Source: Security Containment)
  - [`04_MIERUKO.md`](docs/sources/04_MIERUKO.md) (Design Source: Workspace Binding)
  - [`05_AIWORKHUB.md`](docs/sources/05_AIWORKHUB.md) (Design Source: Evidence-First Review)
  - [`06_SYMPHONY.md`](docs/sources/06_SYMPHONY.md) (Design Source: Workflow-as-Policy)
  - [`07_CODENCER.md`](docs/sources/07_CODENCER.md) (Design Source: Bridge-not-Brain)
  - [`08_AWS_CAO.md`](docs/sources/08_AWS_CAO.md) (Design Source: Provider Abstraction)
  - [`09_ANTIGRAVITY_LINK.md`](docs/sources/09_ANTIGRAVITY_LINK.md) (Fallback: CDP Bridge)

### Architecture Decision Records (ADRs)
- Ten canonical ADRs covering architecture planes, agent adapters, no-GUI default, evidence-first review, and task immutability in [`docs/adr/`](docs/adr/).

### Formal Schemas & Examples
- JSON Schemas and test fixture examples in [`docs/schemas/`](docs/schemas/).

---

# 4. REPOSITORY GOVERNANCE

All contributions and agent executions are strictly governed by [`AGENTS.md`](AGENTS.md) and [`docs/24_CHANGE_GOVERNANCE.md`](docs/24_CHANGE_GOVERNANCE.md). No application code may be introduced until Phase 0 achieves formal `ARCHITECTURE_FROZEN` status via external review.

