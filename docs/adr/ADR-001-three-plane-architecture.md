# ADR-001: Three-Plane Architectural Separation

## Status
ACCEPTED

## Context
In AI-assisted software development, conversational AI models excel at high-level reasoning, system architecture, and holistic review, but lack deterministic process management, local filesystem isolation, and reliable state retention. Conversely, local coding agents excel at file manipulation, test execution, and compilation, but risk losing architectural alignment when given broad conversational tasks.

## Decision
We formally adopt a **Three-Plane Architecture**:
1. **Intelligence Plane (ChatGPT Web)**: Autonomous supervision, architectural design, task formulation, and final audit.
2. **Supervisor Control Plane (Our System)**: State management, immutable contract dispatch, independent evidence collection, and policy auditing.
3. **Execution Control Plane (Untrivial Agent Orchestrator + Antigravity CLI)**: Local process lifecycle, ConPTY terminals, worktree isolation, and code editing.

## Alternatives Considered
- *Monolithic Local Agent*: Merging reasoning, daemon, and coding into one giant model harness. Rejected due to model context limits, loss of high-reasoning oversight, and excessive coupling.
- *Two-Plane (ChatGPT directly driving CLI)*: Rejected because ChatGPT cannot manage background daemons, worktrees, or reliably collect independent Git diffs without tool explosion.

## Consequences
- Clean separation of concerns; each plane can be updated or swapped independently.
- Supervisor Control Plane remains a lightweight state manager and evidence compiler, not an expensive second AI brain.

## Source Evidence
- Proven effective in manual developer workflows documented in `docs/00_PROJECT_OVERVIEW.md`.
- Architectural patterns from OpenAI Symphony (orchestrator vs workspace runners) and Codencer (bridge-not-brain).

## Revisit Conditions
Revisit if future unified AI runtimes provide built-in, verifiably isolated multi-agent OS execution with formal audit guarantees.
