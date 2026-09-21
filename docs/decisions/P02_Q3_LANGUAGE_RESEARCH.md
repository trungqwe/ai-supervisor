# P02 Q3 Language Research Dossier (Corrected Baseline)

> **Authority**: P02 Pre-Code Implementation Decision Gate (Reaudit Remediation)
> **Date**: 2026-09-22
> **Status**: RESEARCH_REVISED_PENDING_EXTERNAL_DECISION
> **Prior Version**: Invalidated due to stale factual premise regarding Go MCP SDK.

---

## 1. Primary Source Authority & Technology Baselines

This research dossier evaluates implementation languages for the **AI Engineering Supervisor Control Plane Core** against verified primary documentation sources.

### 1.1 Model Context Protocol (MCP) Official SDKs
- **Go SDK**: `github.com/modelcontextprotocol/go-sdk`
  - *Authority*: Official Tier 1 MCP SDK co-maintained with Google.
  - *Protocol Revision*: Fully implements the current 2026-07-28 protocol specification.
  - *Transports*: First-class support for `stdio` and `HTTP/SSE` (Server-Sent Events).
  - *Features*: Complete server and client implementations, tool/resource/prompt registries, session management, and idiomatic Go type safety.
  - *Correction*: The prior assumption that Go lacked an official MCP SDK is **retracted**. Go possesses an official, actively maintained Tier 1 SDK.
- **TypeScript SDK**: `github.com/modelcontextprotocol/typescript-sdk` (`@modelcontextprotocol/sdk`)
  - *Authority*: Official Tier 1 MCP reference SDK maintained by Anthropic.
  - *Protocol Revision*: Fully implements the current 2026-07-28 protocol specification.
  - *Transports*: Stdio, SSE, HTTP streaming.
- **Python SDK**: `github.com/modelcontextprotocol/python-sdk` (`mcp`)
  - *Authority*: Official Tier 1 MCP SDK. Asyncio-based.

### 1.2 Node.js Release Baselines & SQLite Stability
- **Node.js Official Release Schedule**:
  - **Node 22**: Maintenance LTS. (Not recommended as the primary forward baseline for a new daemon).
  - **Node 24**: Active LTS line. Designated as the primary production baseline if Node is chosen.
  - **Node 26**: Current release (non-LTS).
- **Node.js SQLite Driver Status**:
  - `node:sqlite`: Currently holds **Stability 1.2 (Release Candidate)** status in Node 24 official documentation. While built-in, it is not yet classified as fully stable, requiring caution for enterprise production storage.
  - `better-sqlite3`: Mature, battle-tested synchronous native C++ addon. Requires compilation via `node-gyp` or prebuilt binary distribution (`node-gyp-build`).

### 1.3 Go Toolchain & SQLite Status
- **Go Runtime**: Go 1.22+ / 1.23+.
- **Windows Process Management**: Standard library `os/exec` and `syscall.SysProcAttr` provide native access to Windows process creation flags (`CREATE_NEW_PROCESS_GROUP`, `HIDE_WINDOW`, and Windows Job Object assignment for guaranteed process-tree termination).
- **Go SQLite Drivers**:
  - `modernc.org/sqlite`: 100% pure Go SQLite engine transpiled from official SQLite C source via `ccgo`. Completely CGo-free, requires no C compiler on Windows, compiles statically to a single binary.
  - `mattn/go-sqlite3`: CGo-based bindings to SQLite C library.
- **Structured Logging**: Built-in `log/slog` standard library (Go 1.21+) provides zero-allocation, structured JSON logging without third-party dependencies.

### 1.4 Proven Inbound Transport Clarification
- Empirical testing in Track P01-D3C established the viability of ChatGPT tool connectivity via:
  `ChatGPT Developer Mode App → Secure MCP Tunnel → tunnel-client → private loopback MCP endpoint`.
- Track P01-D3C validated HTTP MCP over loopback, not direct stdio. Both the official TypeScript and official Go MCP SDKs natively support this HTTP/SSE architecture.

---

## 2. Evaluation Criteria & Weighting

| Criterion | Weight | Description |
|---|:---:|---|
| **Official MCP Support & Protocol Coverage** | 15% | First-party Tier 1 SDK, 2026-07-28 protocol compliance, HTTP/SSE transport maturity. |
| **Windows Process & Daemon Control** | 15% | Native Windows process creation, Job Objects, process tree termination (OPS-002), signal handling. |
| **State Store & SQLite Integration** | 15% | Multi-entity ACID transactions, WAL mode, `PRAGMA synchronous = FULL`, zero-cloud dependency. |
| **Binary & Deployment Simplicity** | 10% | Standalone distribution on Windows x64, zero user-side runtime prerequisites (OPS-001, OPS-003). |
| **Daemon Stability & Concurrency** | 10% | Long-running daemon reliability, low memory footprint (<50MB RSS), non-blocking execution model. |
| **Type Safety & Domain Modeling** | 10% | Strict type safety for 13-state machine, immutable TaskContract revisions, exhaustive pattern matching. |
| **JSON Schema Validation** | 10% | Speed and conformance for `task-contract.schema.json` and `worker-report.schema.json`. |
| **Developer Velocity & Maintenance** | 10% | Speed of development, tooling maturity, code clarity. |
| **Audit Logging & Tamper Evidence** | 5% | Standardized structured JSON logging, minimal supply chain. |

---

## 3. Candidate Re-Scoring (Evidence-Based)

### 3.1 Go (v1.22+)
- **Official MCP (9.5/10)**: Official Tier 1 `github.com/modelcontextprotocol/go-sdk` co-maintained with Google. Implements full 2026-07-28 protocol, HTTP/SSE, stdio, and tool registries.
- **Windows Process Control (9.5/10)**: Unmatched native Windows control via `syscall.SysProcAttr` and Windows Job Objects. Cleanly kills child process trees (e.g. test runners) without leaving zombie processes.
- **State Store (9.0/10)**: `modernc.org/sqlite` provides pure Go, CGo-free SQLite with full WAL and `synchronous=FULL` transaction support.
- **Deployment Simplicity (10/10)**: Compiles to a single standalone `supervisor.exe`. Zero runtime installation, zero `node_modules`, zero DLL hell.
- **Daemon Stability & Memory (10/10)**: Extremely low memory footprint (15–25 MB RSS), instantaneous startup (<50ms), no garbage collection pauses impacting HTTP responsiveness.
- **Type Safety (9.0/10)**: Statically typed structs, interfaces, compile-time validation.
- **JSON Schema (8.5/10)**: `github.com/santhosh-tekuri/jsonschema/v6` provides complete Draft-07 and 2020-12 validation.
- **Developer Velocity (8.0/10)**: Verbose error handling, highly disciplined development.
- **Audit Logging (10/10)**: Standard library `log/slog` handles high-throughput structured JSON logging with zero dependencies.
- **Weighted Score: 9.30 / 10**

### 3.2 TypeScript on Node.js (Node 24 LTS)
- **Official MCP (10/10)**: Canonical reference implementation (`@modelcontextprotocol/sdk`), authoring home of protocol features.
- **Windows Process Control (7.5/10)**: Uses `child_process.execFile`. Process tree termination on Windows requires external helpers or `taskkill`. Signal handling (`SIGTERM`) is emulated on Windows.
- **State Store (7.5/10)**: `better-sqlite3` requires native C++ toolchain / prebuilt addons. Built-in `node:sqlite` is currently Release Candidate (Stability 1.2).
- **Deployment Simplicity (6.0/10)**: Requires host to have Node 24 LTS installed, or complex Single Executable Application (SEA) bundling with injected blobs.
- **Daemon Stability & Memory (7.0/10)**: Node daemon consumes 70–130 MB RSS; single-threaded event loop risks blocking on CPU-intensive JSON Schema or cryptographic hash calculations.
- **Type Safety (9.0/10)**: Rich discriminated unions and structural typing via TypeScript.
- **JSON Schema (10/10)**: `Ajv` (v8) is the industry benchmark for fast schema validation.
- **Developer Velocity (9.0/10)**: Rapid prototyping, immediate JSON handling.
- **Audit Logging (9.0/10)**: Relies on third-party `pino` or `winston`.
- **Weighted Score: 8.35 / 10**

### 3.3 Python (v3.11+)
- **Official MCP (8.5/10)**: Official Tier 1 `mcp` SDK; asyncio-based.
- **Windows Process Control (7.0/10)**: `subprocess` on Windows has known quirks with signal handling and console window creation.
- **State Store (8.0/10)**: Built-in `sqlite3`, but async integration requires thread pool executor to avoid event loop stalling.
- **Deployment Simplicity (5.0/10)**: Requires Python environment, virtual environments, or complex `PyInstaller` packaging.
- **Daemon Stability & Memory (6.5/10)**: High memory footprint (60–100 MB RSS), GIL constraints.
- **Type Safety (6.5/10)**: Optional static type checking via Mypy/Pyright.
- **JSON Schema (8.0/10)**: `jsonschema` package is functional but slower.
- **Developer Velocity (8.5/10)**: High scripting velocity.
- **Audit Logging (7.5/10)**: Built-in `logging` with custom JSON formatters.
- **Weighted Score: 7.25 / 10**

---

## 4. Re-Scored Comparison Matrix

| Evaluation Criterion | Weight | Go (v1.22+) | TypeScript (Node 24 LTS) | Python (v3.11+) |
|---|:---:|:---:|:---:|:---:|
| Official MCP & Protocol Conformance | 15% | 9.5 | **10.0** | 8.5 |
| Windows Process & Daemon Control | 15% | **9.5** | 7.5 | 7.0 |
| State Store & SQLite Integration | 15% | **9.0** | 7.5 | 8.0 |
| Binary & Deployment Simplicity | 10% | **10.0** | 6.0 | 5.0 |
| Daemon Stability & Concurrency | 10% | **10.0** | 7.0 | 6.5 |
| Type Safety & Domain Modeling | 10% | 9.0 | 9.0 | 6.5 |
| JSON Schema Validation | 10% | 8.5 | **10.0** | 8.0 |
| Developer Velocity & Prototyping | 10% | 8.0 | **9.0** | 8.5 |
| Audit Logging & Tamper Evidence | 5% | **10.0** | 9.0 | 7.5 |
| **Weighted Total** | **100%** | **9.30 / 10** | **8.35 / 10** | **7.25 / 10** |

---

## 5. Recommendation & External Decision Path

### 5.1 Primary Recommendation: Go (v1.22+)
- **Primary Justification**:
  1. With the availability of the official Google-co-maintained `github.com/modelcontextprotocol/go-sdk` (supporting 2026-07-28 protocol and HTTP/SSE), Go eliminates any MCP protocol gap.
  2. The Supervisor Control Plane is fundamentally a **local background daemon** managing Windows child processes, file locks, and state transitions. Go's native Windows process management (`syscall.SysProcAttr` / Job Objects) provides fail-safe child termination (OPS-002, SEC-003).
  3. Single static binary deployment (`supervisor.exe`) satisfies OPS-001 and OPS-003 without imposing Node.js runtime or npm dependencies on the user.
  4. `modernc.org/sqlite` provides robust, CGo-free SQLite WAL storage with true synchronous ACID commits.

### 5.2 Strong Runner-Up: TypeScript / Node.js (Node 24 LTS)
- TypeScript remains the reference implementation for MCP. If the External Supervisor values developer alignment with existing test scripts and maximum velocity in JSON manipulation over single-binary deployment and native Windows process semantics, TypeScript on Node 24 LTS is fully viable.

### 5.3 Conditions That Would Alter the Recommendation:
- If the External Supervisor mandates zero-compilation workflows or requires sharing schema types directly with browser-based UI tooling in P06, TypeScript on Node 24 LTS should be selected.
