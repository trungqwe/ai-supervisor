# 00. PROJECT OVERVIEW — CANONICAL ARCHITECTURE & OPERATIONAL VISION

> **Project Name**: AI Engineering Supervisor Control Plane  
> **Authority**: Canonical Project Overview (Single Source of Truth)  
> **Status**: Approved Baseline for Phase 0

---

# 1. Origin & Motivation

The project originates from a real-world software engineering workflow that demonstrated high efficacy in practical development:

1. The developer presents high-level ideas to **ChatGPT Web** (operating with high-reasoning models).
2. ChatGPT analyzes requirements, challenges assumptions, creates technical designs, and formulates specifications.
3. Across several dialogue iterations, ChatGPT formalizes:
   - Requirements;
   - Architecture & module boundaries;
   - Roadmap & phases;
   - Acceptance criteria & test specifications.
4. ChatGPT generates an implementation prompt for a coding agent.
5. The coding agent within **Antigravity** executes the task:
   - Reads project context & architecture documents;
   - Edits source code;
   - Executes build scripts and test suites;
   - Diagnoses and fixes failures;
   - Commits changes to Git;
   - Produces an execution report.
6. The developer manually copies the worker report back into ChatGPT Web.
7. ChatGPT performs a comprehensive audit:
   - Evaluates worker claims against the Git diff;
   - Verifies test executions and pass/fail states;
   - Checks compliance with architectural constraints;
   - Determines whether to approve the task or request revisions.
8. The developer copies ChatGPT's revision prompt or next task back into Antigravity.
9. This loop repeats iteratively until phase or project completion.

### The Real Problem
This division of labor is fundamentally sound:
- **ChatGPT** functions as **Architect, Tech Lead, Planner, Auditor, and Decision Maker**.
- **Antigravity** functions as **Implementer, Coder, Tester, Fixer, and Local Executor**.

The critical bottleneck is that communication between the two AI planes is currently manual: relying on human copy-pasting, visual checking, and error-prone copy-back. 

The **AI Engineering Supervisor Control Plane** exists to automate this operational bridge completely, while preserving the proven separation of responsibilities.

---

# 2. Project Mission & Objective

The objective is to establish an autonomous local control plane where:
- **ChatGPT Web** remains the **Supervisor AI** (understanding project intent, planning, dispatching tasks, auditing evidence, approving or demanding revisions).
- **Antigravity** remains the **Worker AI** (executing code changes, running tests, fixing errors in an isolated workspace).
- **Untrivial Agent Orchestrator (AO)** provides the **Execution Control Plane** (daemon, process ConPTY, session lifecycles, worktree isolation, event streams).
- **The Supervisor Control Plane (Our Project)** manages project registration, Pair coordination, immutable Task Contracts, independent evidence collection, Review Bundle assembly, policy enforcement, and audit logs.

```text
┌──────────────────────────────────────────────────────────────┐
│                    INTELLIGENCE PLANE                        │
│                       ChatGPT Web                            │
│  Architect • Lead Planner • Code Reviewer • Auditor          │
└──────────────────────────────┬───────────────────────────────┘
                               │ High-Level Tool Surface
                               │ (Read Project, Dispatch, Audit)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                 SUPERVISOR CONTROL PLANE                     │
│                       (OUR PROJECT)                          │
│  Project Registry • Pair Registry • Task Contracts           │
│  Policy Engine • Evidence Collector • Review Bundle Builder  │
│  Audit Trail • AO Public Adapter                             │
└──────────────────────────────┬───────────────────────────────┘
                               │ Stable Public API Contract
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                 EXECUTION CONTROL PLANE                      │
│               Untrivial Agent Orchestrator                   │
│  Daemon • Session Lifecycle • Worktree Isolation • Events    │
└──────────────────────────────┬───────────────────────────────┘
                               │ Harness / Headless CLI (agy)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                       PRIMARY WORKER                         │
│                  Official Antigravity CLI                    │
│  Read Task • Inspect Code • Edit • Test • Commit • Report    │
└──────────────────────────────────────────────────────────────┘
```

---

# 3. Explicit Non-Goals: Not a Remote Desktop or OS-level Automation AI

This project is **NOT** an OS-level computer use or desktop automation system.

We explicitly reject:
- Controlling mice or simulated keyboard strokes.
- Desktop screenshot streaming for interaction.
- Driving IDE graphical user interfaces via OCR or pixel coordinates.
- Using GitHub Issues or PRs as a slow, noisy message bus for internal loops.

All interactions are structured, headless, and programmatic via verified APIs, CLI headless protocols, and the Model Context Protocol (MCP) or ChatGPT Apps/Actions.

---

# 4. Architectural Philosophy: The Three Planes

### Plane 1: Intelligence Plane (ChatGPT Web)
- **Authority**: Architecture, Planning, Task Formulation, Audit, Approval/Rejection.
- **Constraints**: ChatGPT does NOT directly manage OS processes, touch local disks, manage worktrees, or write code directly.

### Plane 2: Supervisor Control Plane (Our System)
- **Authority**: Task Contracts, Workflow State Machine, Evidence Gathering, Review Bundle Generation, Policy Enforcement, Audit Logging.
- **Definition**: A state manager, validator, and secure bridge — NOT a second reasoning brain. It does not replace ChatGPT's judgment.

### Plane 3: Execution Control Plane (Agent Orchestrator)
- **Authority**: Process lifecycle, local daemon, ConPTY terminal handling, worktree creation/destruction, agent session persistence.
- **Principle**: We do not reinvent AO. We integrate through AO's public daemon API.

---

# 5. Core Operational Concepts

## 5.1 The `Pair` Abstraction
A `Pair` represents an active engineering lane binding one project, one Supervisor session, one active worker runtime, and one task workflow.
In V1:
- 1 Pair = 1 Project + 1 ChatGPT Session + 1 Active Worker Session + 1 Active Task.
- Ensures total containment and prevents cross-project interference.

## 5.2 The Immutable Task Contract
Worker agents are never given unconstrained chat prompts. Every task is dispatched as an immutable **Task Contract** containing:
- `task_id`, `phase_id`, `objective`;
- Strict `allowed_scope` (whitelist of files/directories);
- Explicit `forbidden_scope` (architecture docs, core schemas, unrelated modules);
- Concrete acceptance criteria and mandatory test commands;
- Required evidence types and stop conditions.
Once dispatched, a contract is frozen. If unexpected architectural requirements emerge, the task is marked `BLOCKED` and escalated back to ChatGPT.

## 5.3 Evidence-First Review
A worker completion report is treated as a set of **claims**, not ground truth.
The Supervisor Control Plane independently gathers:
- Base Git commit SHA & Head Git commit SHA;
- Actual Git diff and list of touched files;
- Exact commands executed and their OS exit codes;
- Test execution output and artifact hashes.

## 5.4 The Review Bundle
To prevent ChatGPT from making 20–30 low-level tool calls to inspect changes, the Supervisor compiles an integrated **Review Bundle**:
- Aggregates task requirements, worker claims, actual diff, executed tests, policy validation findings, and scope compliance checks into a single concise, high-signal package.
- ChatGPT evaluates the bundle, drilling down only when suspicious deviations or test discrepancies appear.

---

# 6. Separation of Authorities (Source of Truth)

The system rejects a monolithic "single database of everything" approach:
- **Technical Truth**: Canonical documentation in `docs/` (Requirements, Architecture, ADRs).
- **Task & Governance Truth**: Supervisor Control Plane database (Task states, Pair bindings, Review decisions).
- **Execution Truth**: Agent Orchestrator runtime (Session state, process health, active worktree).
- **Code Truth**: Git repository (Commit history, tree hashes, actual diffs, test logs).
- **Chat History**: Volatile conversation memory — never treated as durable system state.

---

# 7. V1 Scope vs. Long-Term Roadmap

### In-Scope for V1:
- Local single-machine execution (Windows-native with ConPTY support via AO).
- Single Pair binding (1 Project, 1 ChatGPT Supervisor, 1 Antigravity Worker).
- Headless Antigravity CLI (`agy`) execution via AO harness.
- Full evidence collection loop: Dispatch → Run → Collect Diff/Logs → Review Bundle → Approve/Revise.
- Sanitized local audit trail.

### Explicitly Excluded from V1 (Post-V1 Backlog):
- Multi-worker parallel swarms and 100-task dependency schedulers.
- Cloud Kubernetes, Redis, or Kafka infrastructure.
- Automated multi-agent debates or autonomous model routers.
- Direct source code modification by the Supervisor.
