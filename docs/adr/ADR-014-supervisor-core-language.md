# ADR-014: Selection of Implementation Language for Supervisor Core

> **Status**: ACCEPTED
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Deciders**: External Supervisor, Engineering Team
> **Consulted**: `docs/decisions/P02_Q3_LANGUAGE_RESEARCH.md`

---

## 1. Context and Problem Statement

The AI Engineering Supervisor Control Plane core requires a robust, long-running local background daemon on Windows 10/11 x64 capable of:
1. Serving a Model Context Protocol (MCP) tool surface over Streamable HTTP;
2. Enforcing a strict 13-state finite state machine with immutable `TaskContract` revisions and `TaskAttempt` lineage;
3. Managing local SQLite state persistence with zero cloud dependencies (OPS-003);
4. Providing deterministic Windows process creation and cleanup APIs (OPS-002, SEC-003).

---

## 2. Decision: Go (SUPERVISOR_CORE_LANGUAGE = GO)

The implementation language for the Supervisor Core daemon is **Go**.

### Toolchain Baseline and Support Policy:
- **Development Toolchain Baseline**: **`Go 1.27.x`** (Current supported major as of 2026-09-22).
- **Minimum Supported Go Line**: **`Go 1.26.x`** (Previous supported major).
- *Release Policy Rationale*: Go maintains official release support for each major release until two newer major releases exist. Basing the development toolchain on Go 1.27.x with Go 1.26.x as the compatibility floor adheres strictly to official upstream lifecycle standards. Stale baselines (`Go 1.22+`, `Go 1.23+`, `Go 1.24+`) are retired.
- *Dependency Versioning*: Exact patch releases and module dependency versions will be pinned during Phase P02 implementation bootstrap, rather than frozen as immutable architectural constants.

---

## 3. Upstream Technology Alignment

### 3.1 Official MCP Go SDK & Streamable HTTP
- The official Go SDK for the Model Context Protocol is **`github.com/modelcontextprotocol/go-sdk`**, co-maintained with Google as an official Tier 1 SDK.
- Protocol coverage fully implements the MCP 2026-07-28 specification.
- Modern HTTP transport uses **Streamable HTTP** (the current standard for HTTP-based MCP transport). Legacy standalone SSE transport is categorized as legacy.
- *Verification Scope Note*: Track P01-D3C empirically proved loopback HTTP MCP transport using a test client and OpenAI tunnel. It did not evaluate the future Go implementation. Full end-to-end validation of the Go MCP server implementation is a Phase **P05** deliverable.

### 3.2 Windows Process Control & Job Objects
- Go standard library `os/exec` and `syscall.SysProcAttr` provide native Windows process-creation controls (`CREATE_NEW_PROCESS_GROUP`, console window suppression).
- Complete process-tree lifecycle management (preventing leaked child processes from test runners) requires explicit Win32 Job Object integration via **`golang.org/x/sys/windows`** (`CreateJobObject`, `AssignProcessToJobObject`, `TerminateJobObject`).
- *Architecture Note*: `syscall.SysProcAttr` alone does not manage Job Objects; explicit Win32 API calls are required. Full verification runner execution isolation is an implementation deliverable for Phase **P04**.

### 3.3 State Store Integration
- `modernc.org/sqlite` is selected as the preferred Go SQLite implementation:
  - 100% pure Go transpiled from SQLite C source via `ccgo`.
  - Zero CGo compiler requirement on Windows.
  - Compiles cleanly into a single standalone static binary (`supervisor.exe`), fully satisfying OPS-001 and OPS-003.

---

## 4. Consequences

### Positive:
- **Single Static Binary Distribution**: Compiles to a self-contained executable (`supervisor.exe`) with zero runtime prerequisites (no Node.js runtime or npm dependencies on the user's system).
- **Direct OS-Level Control**: Native Win32 API access through `golang.org/x/sys/windows` for robust child process supervision and Job Object binding.
- **Official MCP Compliance**: First-class Tier 1 SDK support implementing the latest 2026-07-28 protocol via Streamable HTTP.
- **Clean Architecture Separation**: The headless domain core runs independently of web runtimes or external browser dependencies.

### Negative / Tradeoffs:
- Slightly more verbose boilerplate for JSON handling compared to dynamic languages.
- Contract JSON schemas (`task-contract.schema.json`) are compiled and validated using a standards-compliant JSON Schema Draft-07 validator selected, audited and pinned during P02 dependency bootstrap rather than direct in-memory TypeScript type sharing.
