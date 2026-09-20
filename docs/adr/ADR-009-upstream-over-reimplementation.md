# ADR-009: Upstream Over Reimplementation

## Status
ACCEPTED

## Context
Software engineering projects often succumb to the "Not Invented Here" (NIH) syndrome, rebuilding process daemons, terminal parsers, and agent harnesses that have already been developed and tested in the open-source community.

## Decision
We enforce an explicit policy: **Upstream First, Adapter Second, Reimplement Pattern Third, Original Invention Last**. Before any module or subsystem is proposed, existing capabilities in evaluated repositories (Agent Orchestrator, Antigravity CLI, Proxide, Symphony, Codencer) must be checked.

## Alternatives Considered
- *Greenfield Complete Stack*: Building every layer from scratch. Rejected as wasteful, slow, and unmaintainable.

## Consequences
- Dramatically reduced lines of proprietary code to maintain.
- Maximum leverage of battle-tested open-source components.

## Source Evidence
- Architectural reuse principles documented in `docs/sources/REUSE_MATRIX.md`.

## Revisit Conditions
Permanent core policy.
