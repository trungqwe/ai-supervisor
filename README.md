# AI Engineering Supervisor Control Plane

> **Autonomous Coordination, Independent Verification, and Architectural Governance for AI Coding Agents**

---

## 1. What It Is

The **AI Engineering Supervisor Control Plane** is a local coordination and audit layer designed to bridge high-reasoning conversational intelligence (**ChatGPT Web** as Supervisor/Architect) and local coding agents (**Antigravity CLI** as Worker/Implementer).

By leveraging **Untrivial Agent Orchestrator** as the execution runtime, this system automates the task dispatch, progress observation, and evidence collection loop. It eliminates manual prompt/report copy-pasting while strictly preventing architectural drift, enforcing immutable task scopes, and independently verifying Git diffs and test results before approving code changes.

---

## 2. Architecture Thumbnail

```text
┌──────────────────────────────────────────────────────────────┐
│                    INTELLIGENCE PLANE                        │
│                       ChatGPT Web                            │
│     Architect • Lead Planner • Code Reviewer • Auditor       │
└──────────────────────────────┬───────────────────────────────┘
                               │ High-Level Tool Interface (12 Tools)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                  SUPERVISOR CONTROL PLANE                    │
│                        (OUR PROJECT)                         │
│  Project/Pair Registry • Task Contracts • Evidence Collector │
│      Review Bundle Builder • Policy Engine • Audit Trail     │
└──────────────────────────────┬───────────────────────────────┘
                               │ Public REST API / Stable Adapter
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                  EXECUTION CONTROL PLANE                     │
│               Untrivial Agent Orchestrator                   │
│   Daemon • Session Lifecycle • Worktrees • Process ConPTY    │
└──────────────────────────────┬───────────────────────────────┘
                               │ Managed Agy Harness Invocation
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                        PRIMARY WORKER                        │
│                   Official Antigravity CLI                   │
│     Autonomous Edit • Compile • Test • Fix • Git Commit      │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. Current Project Status

| Attribute | Value |
|---|---|
| **Current Stage** | **Phase 0 — Architecture Freeze** |
| **Current State** | [`PHASE0_READY_FOR_EXTERNAL_AUDIT`](docs/18_CURRENT_STATE.md) |
| **Governance Mode** | Strict No-Code Enforcement in Phase 0; External Audit Gate Required |
| **Audit Status** | Literal Evidence Hygiene Complete; all findings from Re-Audit #3 & #4 resolved (see [`docs/audits/PHASE0_LITERAL_EVIDENCE_FINAL_AUDIT.md`](docs/audits/PHASE0_LITERAL_EVIDENCE_FINAL_AUDIT.md)) |
| **GitHub Repository** | [https://github.com/trungqwe/ai-supervisor](https://github.com/trungqwe/ai-supervisor) |

---

## 4. Key Documentation Links

### Canonical Architecture & Project Truth
- **Canonical Project Overview**: [`docs/00_PROJECT_OVERVIEW.md`](docs/00_PROJECT_OVERVIEW.md)
- **Canonical Architecture Blueprint**: [`docs/04_ARCHITECTURE.md`](docs/04_ARCHITECTURE.md)
- **Requirements Specification**: [`docs/02_REQUIREMENTS.md`](docs/02_REQUIREMENTS.md)
- **Workflow State Machine**: [`docs/06_WORKFLOW_STATE_MACHINE.md`](docs/06_WORKFLOW_STATE_MACHINE.md)
- **Security & Containment**: [`docs/07_SECURITY_MODEL.md`](docs/07_SECURITY_MODEL.md)
- **Task Contract Specification**: [`docs/08_TASK_CONTRACT.md`](docs/08_TASK_CONTRACT.md)
- **Review Bundle Specification**: [`docs/10_REVIEW_BUNDLE.md`](docs/10_REVIEW_BUNDLE.md)
- **Dynamic Current State**: [`docs/18_CURRENT_STATE.md`](docs/18_CURRENT_STATE.md)
- **Change Governance & Decision Hierarchy**: [`docs/24_CHANGE_GOVERNANCE.md`](docs/24_CHANGE_GOVERNANCE.md)

### Technology Research & Anti-Reinvention
- **Source Registry**: [`docs/sources/SOURCE_REGISTRY.md`](docs/sources/SOURCE_REGISTRY.md)
- **Anti-Reinvention Reuse Matrix**: [`docs/sources/REUSE_MATRIX.md`](docs/sources/REUSE_MATRIX.md)
- **Upstream Contract Baseline**: [`docs/sources/UPSTREAM_CONTRACT_BASELINE.md`](docs/sources/UPSTREAM_CONTRACT_BASELINE.md)
- **Source Dossiers with Evidence**: [`docs/sources/`](docs/sources/)

### Audits & Governance
- **Phase 0 Literal Evidence Final Audit**: [`docs/audits/PHASE0_LITERAL_EVIDENCE_FINAL_AUDIT.md`](docs/audits/PHASE0_LITERAL_EVIDENCE_FINAL_AUDIT.md)
- **Phase 0 Process Deviations Register**: [`docs/audits/PHASE0_PROCESS_DEVIATIONS.md`](docs/audits/PHASE0_PROCESS_DEVIATIONS.md)
- **External Re-Audit #3 Findings Register**: [`docs/audits/PHASE0_EXTERNAL_REAUDIT_003.md`](docs/audits/PHASE0_EXTERNAL_REAUDIT_003.md)
- **Phase 0 Evidence Integrity Final Audit (Superseded)**: [`docs/audits/PHASE0_EVIDENCE_FINAL_AUDIT.md`](docs/audits/PHASE0_EVIDENCE_FINAL_AUDIT.md)
- **External Re-Audit #2 Findings Register (Historical)**: [`docs/audits/PHASE0_EXTERNAL_REAUDIT_002.md`](docs/audits/PHASE0_EXTERNAL_REAUDIT_002.md)
- **Phase 0 Remediation Audit (Historical)**: [`docs/audits/PHASE0_REMEDIATION_AUDIT.md`](docs/audits/PHASE0_REMEDIATION_AUDIT.md)
- **External Audit Findings Register (Historical)**: [`docs/audits/PHASE0_EXTERNAL_AUDIT_FINDINGS.md`](docs/audits/PHASE0_EXTERNAL_AUDIT_FINDINGS.md)

### Agent Operational Directives
- **Coding Agent Rules & Constraints**: [`AGENTS.md`](AGENTS.md)
