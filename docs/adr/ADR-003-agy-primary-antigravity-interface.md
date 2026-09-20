# ADR-003: Antigravity CLI (agy) as Primary Worker Interface

## Status
ACCEPTED

## Context
Google Antigravity provides both an interactive desktop IDE and a headless command-line interface (`agy`). Our system requires an autonomous, scriptable code editing worker that can be launched inside isolated Git worktrees without manual human window interaction.

## Decision
We designate the **official Antigravity CLI (`agy`)** as our primary worker execution engine. Agent Orchestrator manages the invocation of `agy` in headless mode. Direct Supervisor-to-Agy integration is retained only as a fallback specification.

## Alternatives Considered
- *Antigravity Desktop GUI Automation*: Driving the Antigravity GUI via accessibility APIs or CDP. Rejected as brittle, window-focus sensitive, and user-disruptive.
- *Third-party coding harnesses (Claude Code, Aider)*: Viable for future phases, but Antigravity CLI is chosen as the primary V1 reference worker.

## Consequences
- Headless execution with structured JSON streaming output.
- Clean process termination and deterministic exit code inspection.

## Source Evidence
- Antigravity official documentation and headless CLI specifications.

## Revisit Conditions
Revisit if Google Antigravity removes the headless CLI interface or restricts programmatic harness execution.
