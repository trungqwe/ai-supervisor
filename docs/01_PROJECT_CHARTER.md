# 01. PROJECT CHARTER

> **Project**: AI Engineering Supervisor Control Plane  
> **Status**: Approved Baseline  
> **Authority**: Core Foundation

---

# 1. Purpose
The purpose of the **AI Engineering Supervisor Control Plane** is to establish an autonomous, reliable, and auditable engineering bridge between high-reasoning conversational intelligence (ChatGPT Web) acting as Technical Supervisor and localized agentic coding engines (Antigravity) acting as Workers, mediated by an industrial-grade execution runtime (Untrivial Agent Orchestrator).

---

# 2. Problem Statement
Currently, multi-model AI engineering pairs operate through manual developer intermediation:
- **High cognitive friction**: Developers must manually transfer design specifications, phase constraints, and task scopes from ChatGPT into the coding environment.
- **Verification gap**: Developers must manually read execution terminal dumps and copy them back to ChatGPT, often accepting worker claims without verifying actual Git diffs or exit codes.
- **Architectural drift**: Workers without strict contracts often drift outside their assigned scope, alter shared dependencies, or inadvertently modify global architecture.
- **Chat volatility**: Engineering history and state get lost across long, transient conversation threads.

---

# 3. Core Project Goals
1. **Automate the Supervisor-Worker Loop**: Seamlessly dispatch structured tasks from ChatGPT to Antigravity without human copy-pasting.
2. **Guarantee Scope Containment**: Enforce strict, immutable Task Contracts that confine worker edits to predefined file sets.
3. **Evidence-First Verification**: Automatically collect Git diffs, test logs, and artifact hashes into an aggregated Review Bundle for ChatGPT auditing.
4. **Resilient Local Execution**: Run natively on Windows developer workstations leveraging tested upstream tools (Agent Orchestrator daemon and official Antigravity CLI).
5. **Preserve Upstream Decoupling**: Isolate external runtimes and models behind stable domain adapters, enabling future worker or transport swaps without architectural rewrites.

---

# 4. Explicit Non-Goals (Scope Boundaries)
- **NO OS-Level GUI Automation**: We do not simulate mouse clicks, keyboard presses, or screen capture.
- **NO Reinvention of Execution Orchestrators**: We do not build a proprietary agent daemon or ConPTY terminal manager; Untrivial Agent Orchestrator is used directly.
- **NO Direct Supervisor Code-Writing**: ChatGPT and the Supervisor Control Plane do not write, patch, or commit application source code directly. All code edits belong to the Worker.
- **NO Heavyweight Distributed Infrastructure in V1**: No Kubernetes, Redis, or Kafka. The system operates as a lightweight, robust local service.

---

# 5. Core Architectural Principles
1. **Upstream Over Reinvention**: If an active upstream (e.g., Agent Orchestrator) provides a capability, adopt it via public contracts. Do not rebuild it.
2. **Adapter Over Coupling**: Shield the core domain from third-party DTOs, schemas, or breaking changes using strict anti-corruption adapters.
3. **Evidence Over Claims**: A worker completion report represents claims; the Supervisor verifies truth through independent Git and test evidence.
4. **State Over Chat History**: System state resides in durable local persistence, not in conversational memory.
5. **Task Contracts Over Conversational Prompts**: Every task dispatched is an immutable contract with explicit boundaries.
6. **Architecture Changes Require ADR**: No agent or developer may modify architecture without a formal Architecture Decision Record.
7. **New Ideas Become Proposals**: Any optimization or feature conceived during implementation must be filed as a Proposal, never added spontaneously.
