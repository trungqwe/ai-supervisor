# ADR-014: Selection of Implementation Language for Supervisor Core

> **Status**: PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Deciders**: External Supervisor, Engineering Team
> **Consulted**: `docs/decisions/P02_Q3_LANGUAGE_RESEARCH.md`

---

## 1. Context and Problem Statement

The AI Engineering Supervisor Control Plane core requires a robust, long-running local daemon capable of:
1. Serving an HTTP/SSE Model Context Protocol (MCP) tool surface to ChatGPT over a loopback tunnel;
2. Enforcing a strict 13-state finite state machine with immutable `TaskContract` revisions and `TaskAttempt` lineage;
3. Safely supervising and terminating Windows child processes (verification commands, test runners) without process leaks (OPS-002, SEC-003);
4. Providing self-contained local SQLite persistence with zero cloud dependencies (OPS-003).

A prior draft of this ADR assumed Go lacked an official MCP SDK. That premise has been refuted: `github.com/modelcontextprotocol/go-sdk` is an official Tier 1 SDK co-maintained with Google. This ADR re-evaluates the language selection based on primary evidence.

---

## 2. Decision Candidates

- **Option 1: Go (v1.22+)** — Standalone compiled binary, native Windows process/job-object management, official Tier 1 Go MCP SDK, built-in `log/slog`, pure Go SQLite (`modernc.org/sqlite`).
- **Option 2: TypeScript on Node.js (Node 24 LTS)** — MCP reference SDK (`@modelcontextprotocol/sdk`), `Ajv` schema engine, `better-sqlite3` addon, rich JSON/type ecosystem.
- **Option 3: Python (v3.11+)** — Official `mcp` Python SDK, built-in `sqlite3`.

---

## 3. Evaluation Summary

| Criterion | Go | TypeScript (Node 24 LTS) | Python |
|---|:---:|:---:|:---:|
| Official MCP Tier 1 SDK (2026-07-28 protocol) | YES (`go-sdk`) | YES (`@modelcontextprotocol/sdk`) | YES (`mcp`) |
| Native Windows Process Tree & Job Objects | First-class (`SysProcAttr`) | Limited (requires external tools) | Limited |
| Single Static Binary Distribution (OPS-001) | YES (`supervisor.exe`) | NO (requires runtime/complex SEA) | NO |
| SQLite WAL Durability & CGo-free option | YES (`modernc.org/sqlite`) | Native C++ addon (`better-sqlite3`) | Built-in |
| Weighted Score (from Research Dossier) | **9.30 / 10** | 8.35 / 10 | 7.25 / 10 |

---

## 4. Proposed Recommendation

**Recommend Option 1: Go (v1.22+)** as the primary implementation language for the Supervisor Core daemon, with **Option 2: TypeScript (Node 24 LTS)** as the formally accepted runner-up.

### Rationale:
1. **Daemon Operational Excellence**: As a local daemon on Windows, the Supervisor requires deterministic process tree termination, resilient signal handling, and crash safety. Go's standard library provides direct OS-level control.
2. **Deployment Simplicity**: Distributing a single self-contained executable (`supervisor.exe`) fulfills the zero-dependency requirements of OPS-001 and OPS-003.
3. **Official MCP Parity**: The official Google-co-maintained `github.com/modelcontextprotocol/go-sdk` provides Tier 1 protocol coverage matching the TypeScript SDK.

---

## 5. Consequences

### Positive:
- Single executable distribution with zero runtime prerequisites on user machines.
- Guaranteed Windows child process containment via Job Objects.
- Minimal memory footprint (15–25 MB RSS) and high daemon uptime stability.
- Zero CGo compiler dependency when using `modernc.org/sqlite`.

### Negative / Tradeoffs:
- Slightly more verbose boilerplate compared to TypeScript for JSON manipulation.
- Schema definitions (`task-contract.schema.json`) are validated via Go schema libraries rather than shared TypeScript types.

---

## 6. Status

This ADR remains **PROPOSED** pending External Supervisor audit and formal authorization.
