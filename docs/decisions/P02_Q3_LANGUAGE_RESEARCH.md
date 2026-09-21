# P02 Q3 Language Research Dossier (Final Decision)

> **Authority**: P02 Final Implementation Decision Canonicalization
> **Date**: 2026-09-22
> **Status**: DECISION_ACCEPTED
> **Final Decision**: `GO_SELECTED`
> **Development Toolchain Baseline**: `Go 1.27.x`
> **Minimum Supported Go Line**: `Go 1.26.x`

---

## 1. Final Decision Summary

The External Supervisor and engineering governance have selected **Go** as the canonical implementation language for the Supervisor Control Plane core:
```
SUPERVISOR_CORE_LANGUAGE = GO
DEVELOPMENT_TOOLCHAIN   = Go 1.27.x
MIN_SUPPORTED_GO_LINE   = Go 1.26.x
```
This selection is based on primary technological evidence across MCP protocol conformance, Windows background daemon stability, single-binary distribution (OPS-001), zero-cloud self-contained storage (OPS-003), and native Win32 process containment APIs.

---

## 2. Primary Source Authority & Technology Evidence

### 2.1 Model Context Protocol (MCP) Go Support
- **Repository**: `github.com/modelcontextprotocol/go-sdk`
- **Classification**: Official Tier 1 MCP SDK co-maintained with Google.
- **Protocol Conformance**: Full compliance with the MCP 2026-07-28 protocol specification.
- **Transport Standard**: Implements modern **Streamable HTTP** transport alongside stdio. Legacy standalone SSE is marked as legacy.
- **Proof Alignment**: Track P01-D3C established the architectural viability of tunneling loopback HTTP MCP endpoints to ChatGPT. The future Go MCP implementation will be integrated and verified in Phase **P05**.

### 2.2 Go Toolchain Baseline & Lifecycle
- In accordance with Go's official release policy (supporting the current major and previous major):
  - **Go 1.27.x**: Designated as the active development toolchain baseline.
  - **Go 1.26.x**: Designated as the minimum supported compatibility floor.
  - Stale pre-release references (`Go 1.22+`, `Go 1.23+`, `Go 1.24+`) are formally retired.
- Exact patch releases and module hashes will be pinned in `go.mod` and dependency lockfiles during Phase P02 implementation bootstrap.

### 2.3 Windows Process Management & Win32 Integration
- Standard library `os/exec` and `syscall.SysProcAttr` provide baseline process creation control on Windows.
- Process-tree lifecycle containment (guaranteeing that test runners and compilers do not leak background child processes upon cancellation) is accomplished via Win32 Job Objects using **`golang.org/x/sys/windows`** (`CreateJobObject`, `AssignProcessToJobObject`, `TerminateJobObject`).
- *Note*: `syscall.SysProcAttr` alone does not provide Job Object management. Concrete execution isolation mechanisms will be developed and benchmarked in Phase **P04**.

### 2.4 Local Storage Integration
- **Driver Direction**: `modernc.org/sqlite` provides 100% pure Go SQLite transpiled from SQLite C source via `ccgo`.
- **Properties**: Eliminates CGo compiler requirements on Windows, compiles statically into `supervisor.exe`, and natively supports SQLite Write-Ahead Logging (WAL) and synchronous durability controls.

---

## 3. Technology Comparison Matrix

| Evaluation Criterion | Weight | Go (1.27.x Baseline) | TypeScript (Node 24 LTS) | Python (3.11+) |
|---|:---:|:---:|:---:|:---:|
| Official MCP & Protocol Conformance | 15% | 9.5 (Official Tier 1 SDK) | **10.0** (Reference SDK) | 8.5 (Asyncio SDK) |
| Windows Process & Daemon Control | 15% | **9.5** (Native Win32 / Job Objects) | 7.5 (Emulated signals) | 7.0 (Subprocess quirks) |
| State Store & SQLite Integration | 15% | **9.0** (`modernc.org/sqlite` pure Go) | 7.5 (`node:sqlite` RC 1.2) | 8.0 (Built-in) |
| Binary & Deployment Simplicity | 10% | **10.0** (Single static `supervisor.exe`) | 6.0 (Requires Node 24 runtime) | 5.0 (Venv / packaging) |
| Daemon Stability & Memory Efficiency | 10% | **10.0** (Compiled daemon, goroutines) | 7.0 (V8 runtime footprint) | 6.5 (GIL constraints) |
| Type Safety & Domain Modeling | 10% | 9.0 (Strict static structs) | 9.0 (Discriminated unions) | 6.5 (Optional typing) |
| JSON Schema Validation | 10% | 8.5 (Standards-compliant Go validator) | **10.0** (`Ajv` v8) | 8.0 (`jsonschema`) |
| Developer Velocity & Tooling | 10% | 8.0 (Disciplined typing) | **9.0** (Rapid prototyping) | 8.5 (Scripting speed) |
| Structured Logging & Audit Integrity | 5% | **10.0** (Built-in `log/slog`) | 9.0 (`pino`) | 7.5 (Standard logging) |
| **Weighted Total** | **100%** | **9.30 / 10** | **8.35 / 10** | **7.25 / 10** |

---

## 4. Governance Verdict

- **Decision**: **`GO_SELECTED`**.
- **ADR Reference**: [ADR-014](file:///d:/TU_CODE/ai-supervisor/docs/adr/ADR-014-supervisor-core-language.md) (ACCEPTED).
- Implementation starts in Phase P02 upon completion of the implementation release audit.
