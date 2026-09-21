# P02 Research Dossier: Q3 — Supervisor Core Implementation Language

> **Status**: RESEARCH_COMPLETE (PENDING_EXTERNAL_DECISION)
> **Date**: 2026-09-22
> **Related**: ADR-001 (Architecture Foundation), ADR-008 (Minimal ChatGPT Surface), ADR-014 (Proposed Language Decision)

---

## 1. Executive Summary & Objective
This research dossier evaluates implementation language candidates for the **AI Engineering Supervisor Control Plane** (Phase P02+). The control plane is a long-running local daemon on Windows 10/11 x64 that maintains durable workflow state, enforces immutable TaskContract governance, manages multi-turn attempt lineages, invokes Agent Orchestrator via REST/SSE, executes constrained test verification commands, and exposes a high-level 12-tool surface to ChatGPT Web via the Model Context Protocol (MCP).

Candidates evaluated:
1. **TypeScript / Node.js**
2. **Go (Golang)**
3. **Python**

---

## 2. Target Operational Environment & Functional Requirements
- **Host OS**: Windows 10 / Windows 11 x64 (PowerShell 5.1 / PowerShell 7, Win32 API).
- **Process Profile**: Long-running background daemon listening on loopback `127.0.0.1:3182`.
- **Inbound Transport**: MCP (Model Context Protocol) over HTTP/SSE, tunneled to ChatGPT via OpenAI `tunnel-client` (proven in Track P01-D3C).
- **Outbound Transport**: HTTP client with Server-Sent Events (SSE) streaming connected to Untrivial Agent Orchestrator daemon (`http://127.0.0.1:8080`).
- **Data Persistence**: Local embedded State Store managing atomic transitions, multi-attempt lineage, and append-only audit records.
- **Process Control**: Spawning and bounding constrained verification test runners with strict timeouts and exit code capture.
- **Contract Enforcement**: Strict JSON Schema (Draft-07) validation on task contracts, worker reports, and review bundles.

---

## 3. Evaluation Criteria & Weights

| ID | Criterion | Weight | Description |
|---|---|---|---|
| **C1** | **MCP Ecosystem Maturity & Upstream Alignment** | 20% | Quality, currency, and official support of Model Context Protocol server SDKs. |
| **C2** | **Runtime Reliability & Daemon Stability on Windows** | 15% | Long-running daemon uptime, memory stability, and signal/shutdown reliability. |
| **C3** | **JSON Schema Tooling & Contract Validation** | 15% | Robust, performant, standards-compliant Draft-07 validation libraries. |
| **C4** | **Process Execution & Containment on Windows** | 15% | Subprocess spawn, argument array passing, tree termination, and timeout enforcement. |
| **C5** | **State Store & SQLite Integration** | 10% | Embedded database drivers, transaction semantics, crash-safety, and CGO/native bindings. |
| **C6** | **Type Safety & Domain Modeling** | 10% | Expressiveness and compile-time guarantees for domain aggregates and state machines. |
| **C7** | **Deployment Simplicity & Footprint** | 10% | Host prerequisites, packaging, memory consumption, and cold start time. |
| **C8** | **Development Velocity & Maintainability** | 5% | Speed of implementation, testing ergonomics, and test assertion tooling. |

---

## 4. Candidate Deep Dive & Empirical Analysis

### Candidate 1: TypeScript / Node.js
- **Ecosystem & MCP (Score: 10/10)**: The Model Context Protocol was created with TypeScript as its canonical, primary reference implementation (`@modelcontextprotocol/sdk`). All latest protocol features, SSE transport adapters, and tool routing primitives debut first and most maturely in TypeScript. Track P01-D3C empirical proof was implemented with `@modelcontextprotocol/sdk` and validated live against ChatGPT Plus.
- **JSON Schema (Score: 10/10)**: `ajv` is the industry benchmark for JSON Schema Draft-07 validation, offering ultra-fast compiled schema execution and detailed error reporting.
- **Windows Process Execution (Score: 7/10)**: `child_process.spawn` and `child_process.execFile` support discrete argument arrays (`shell: false`). Windows tree-kill requires care (e.g. using `taskkill /pid ... /t /f` or Windows job objects) to prevent orphan child processes upon timeout.
- **Storage & SQLite (Score: 9/10)**: Node v22 includes experimental built-in `node:sqlite` (zero external dependencies). In addition, `better-sqlite3` is a battle-tested synchronous SQLite driver with excellent transaction and WAL performance.
- **Type Safety (Score: 8/10)**: TypeScript provides strict type checking, discriminated unions (ideal for state machines and failure reasons), and interfaces. Types are erased at runtime, but paired with `ajv` this provides end-to-end static and runtime safety.
- **Deployment & Footprint (Score: 7/10)**: Requires Node.js runtime on host machine. Memory footprint is ~50–80 MB idle. Cold start is < 200 ms.
- **Upstream Alignment**: Antigravity CLI and extension ecosystem are built on Node.js/TypeScript.

### Candidate 2: Go (Golang)
- **Ecosystem & MCP (Score: 5/10)**: There is **no official first-party Go SDK** released by Anthropic or OpenAI for the Model Context Protocol. Community libraries exist (e.g., `github.com/mark3labs/mcp-go`), but protocol evolution, SSE keep-alives, and session routing risk falling out of sync with OpenAI's tunnel-client requirements.
- **JSON Schema (Score: 7/10)**: Several Draft-07 validators exist (`santhosh-tekuri/jsonschema`, `xeipuuv/gojsonschema`), though schema compilation and dynamic structural error extraction require significantly more boilerplate than Ajv.
- **Windows Process Execution (Score: 9/10)**: `os/exec.Command` with `syscall.SysProcAttr` on Windows provides native Win32 process creation, token assignment, and Job Object integration for clean tree termination.
- **Storage & SQLite (Score: 7/10)**: Standard `mattn/go-sqlite3` requires CGO and a MinGW/GCC toolchain on Windows, creating significant build friction. CGO-free pure-Go SQLite (`modernc.org/sqlite`) works on Windows without GCC, but has higher memory usage and slightly slower query performance.
- **Type Safety (Score: 9/10)**: Compiled static type system, explicit error handling (`if err != nil`), struct tags, and fast compiler.
- **Deployment & Footprint (Score: 10/10)**: Compiles to a single, zero-dependency static executable (`supervisor.exe`). Memory footprint is tiny (< 15–20 MB idle). Near-instant cold start.
- **Upstream Alignment**: Untrivial Agent Orchestrator backend is written entirely in Go (`v0.13.0`).

### Candidate 3: Python
- **Ecosystem & MCP (Score: 8/10)**: Anthropic maintains an official Python SDK (`mcp`) with FastMCP primitives. However, SSE integration with OpenAI's loopback tunnel on Windows has had documented event-loop buffering quirks.
- **JSON Schema (Score: 8/10)**: `jsonschema` library is mature and standard, though slower than compiled Ajv.
- **Windows Process Execution (Score: 6/10)**: `asyncio` subprocesses on Windows using `ProactorEventLoop` have well-documented issues with pipe closing, signal handling, and clean tree termination on Windows.
- **Storage & SQLite (Score: 9/10)**: Native `sqlite3` built into standard library.
- **Type Safety (Score: 6/10)**: Dynamic runtime with optional type hints (`mypy`/`pyright`). Lacks the strict compile-time enforcement of Go or TypeScript.
- **Deployment & Footprint (Score: 5/10)**: Requires Python installation, virtual environments (`venv`), dependency management (`pip`/`uv`), and has a larger disk footprint. Packaging as a Windows binary via PyInstaller is notoriously fragile.

---

## 5. Scoring Matrix

| Criterion | Weight | TypeScript / Node.js | Go | Python |
|---|---|---|---|---|
| **C1: MCP Ecosystem** | 20% | **10** (2.0) | 5 (1.0) | 8 (1.6) |
| **C2: Windows Daemon Stability** | 15% | 8 (1.2) | **10** (1.5) | 7 (1.05) |
| **C3: JSON Schema Tooling** | 15% | **10** (1.5) | 7 (1.05) | 8 (1.2) |
| **C4: Windows Process Control** | 15% | 8 (1.2) | **9** (1.35) | 6 (0.9) |
| **C5: State Store & SQLite** | 10% | **9** (0.9) | 7 (0.7) | 9 (0.9) |
| **C6: Type Safety & Domain** | 10% | 8 (0.8) | **9** (0.9) | 6 (0.6) |
| **C7: Deployment Footprint** | 10% | 7 (0.7) | **10** (1.0) | 5 (0.5) |
| **C8: Development Velocity** | 5% | **9** (0.45) | 8 (0.4) | 8 (0.4) |
| **TOTAL WEIGHTED SCORE** | **100%** | **8.75 / 10** | **7.90 / 10** | **7.15 / 10** |

---

## 6. Synthesis & Recommendation

### Recommended Language: **TypeScript / Node.js (v22+)**
- **Core Rationale**:
  1. The primary architectural innovation of the Supervisor Control Plane is bridging ChatGPT reasoning to local execution via the Model Context Protocol (ADR-008). The **official, primary-source MCP SDK** is TypeScript (`@modelcontextprotocol/sdk`). Choosing a language without an official MCP SDK (like Go) introduces protocol divergence risk at the single most critical external interface of the system.
  2. JSON Schema Draft-07 enforcement is pervasive across TaskContracts, WorkerReports, and ReviewBundles; Node's `ajv` is the gold standard.
  3. Node v22 LTS is already installed on the target development environment, possesses built-in `node:sqlite`, and runs natively on Windows x64.
  4. Discriminated unions in TypeScript model the 13 canonical states, 25 transitions, and failure reasons with total type safety.

### Strong Runner-Up: **Go (Golang)**
- If the User or External Supervisor determines that **single static binary distribution** (`supervisor.exe` with zero Node runtime dependency) outweighs official MCP SDK backing, Go is the clear alternative. Go's concurrency model, memory footprint (<20MB), and native Windows process management are exceptional.

---

## 7. Conditions That Would Change the Recommendation
1. Anthropic/OpenAI releases an official, supported Go SDK for the Model Context Protocol.
2. The user mandates zero runtime dependencies on the target host machine (requiring single static binary distribution).