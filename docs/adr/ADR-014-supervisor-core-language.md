# ADR-014: Supervisor Core Implementation Language

> **Status**: PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Proposal
> **Tracking**: `Q3_SUPERVISOR_CORE_LANGUAGE`
> **Research Dossier**: `docs/decisions/P02_Q3_LANGUAGE_RESEARCH.md`

---

## Context
Phase P02 commences the implementation of the Supervisor Domain Core, requiring the selection of the core implementation language for the long-running daemon, state machine, local state store, process runner, and MCP tool transport.

A comprehensive evaluation across TypeScript/Node.js, Go, and Python was documented in `docs/decisions/P02_Q3_LANGUAGE_RESEARCH.md`.

---

## Decision Proposal
It is proposed to adopt **TypeScript / Node.js (v22+)** as the implementation language for the AI Engineering Supervisor Control Plane.

Key drivers:
1. First-class alignment with the official `@modelcontextprotocol/sdk` (Model Context Protocol), which forms the primary cognitive bridge to ChatGPT Web;
2. Mature Draft-07 JSON Schema validation via `ajv`;
3. Discriminated unions for zero-cost compile-time state machine and failure reason modeling;
4. Native Windows x64 support with Node.js v22.

If zero-runtime single-binary deployment is mandated by the External Supervisor, **Go** is designated as the ready alternative candidate.

---

## Consequences
- **Positive**: Direct compatibility with proven Phase P01-D3C MCP transport code; eliminates the need for an unofficial or reverse-engineered Go MCP stack.
- **Positive**: Rapid domain modeling and JSON Schema test harness integration.
- **Negative / Trade-off**: Requires Node.js v22+ on the host machine; process tree termination on Windows requires deliberate Win32 job object or taskkill wrapping.

---

## Status
- **Current Status**: `PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)`.
- No application code or project scaffolding may be written until this proposal is formally approved by the External Supervisor.