# ADR-002: Agent Orchestrator as Execution Control Plane

## Status
ACCEPTED

## Context
Running coding agents locally on Windows requires robust handling of OS processes, pseudoterminal ConPTY allocation, Git worktree isolation, session persistence across crashes, and multi-harness compatibility. Rebuilding this infrastructure from scratch is a massive undertaking prone to subtle OS concurrency bugs.

## Decision
We adopt **Untrivial Agent Orchestrator (AO)** as our primary, exclusive Execution Control Plane runtime dependency. We interact with AO solely via its public REST API and CLI, completely avoiding direct database or internal package coupling.

## Alternatives Considered
- *Custom Process & Worktree Manager*: Building our own daemon in Go or Rust. Rejected as a severe violation of "Upstream Over Reinvention".
- *Docker / Containerized Workspaces*: Rejected for V1 due to heavy virtualization overhead on Windows developer machines and poor IDE integration.

## Consequences
- Immediate access to industrial-grade Windows ConPTY support, session restoration, and 25+ agent harness compatibility.
- Project development can focus 100% on Supervisor logic and evidence auditing.

## Source Evidence
- Untrivial-ai/agent-orchestrator repository: active development, 12k+ stars, native Windows release `v0.13.0` with daemon and worktree management.

## Revisit Conditions
Revisit if Agent Orchestrator deprecates public API access or changes license from permissive open source.
