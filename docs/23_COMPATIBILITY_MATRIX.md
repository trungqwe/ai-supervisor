# 23. COMPATIBILITY MATRIX

> **Focus**: Platform, Dependency & Interface Compatibility Register  
> **Status**: Approved Baseline (Remediated Phase 0)

---

| Component | Target Version | Platform | Test Status | Contract Baseline Ref | Known Issues / Gaps | Upgrade / Integration Risk |
|---|---|---|---|---|---|---|
| **OS Host** | Windows 10/11 64-bit | Windows | BASELINE | NFR-001 | Requires ConPTY support | Low |
| **Untrivial AO** | `v0.13.0` (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) | Windows x64 | DOCUMENTATION_BASELINE_VERIFIED / RUNTIME_TEST_PENDING_P01 | `docs/sources/UPSTREAM_CONTRACT_BASELINE.md#sec-1` | Daemon loopback binding and session lifecycle require P01-A Windows runtime test | Medium (Pinned to v0.13.0) |
| **Antigravity CLI** | `1.2.7` (`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`) | Windows x64 | DOCUMENTATION_BASELINE_VERIFIED / RUNTIME_TEST_PENDING_P01 | `docs/sources/UPSTREAM_CONTRACT_BASELINE.md#sec-2` | Local binary `1.2.7` detected; headless execution flags require P01-B empirical test | Medium (Official Google CLI) |
| **AO ↔ Agy Adapter** | Built-in AO Adapter (`backend/internal/adapters/agent/agy/agy.go`) | Windows x64 | RUNTIME_TEST_PENDING_P01 | `docs/sources/UPSTREAM_CONTRACT_BASELINE.md#sec-3` | AO invokes Agy interactively; structured WorkerReport normalization requires P01-C proof | High (Integration gap; may require adapter layer) |
| **Git Runtime** | Git `>= 2.40` | Windows x64 | VERIFIED (`2.55.0` on host) | FR-008 | None | Very Low |
| **ChatGPT Transport** | UNRESOLVED — P01 TRANSPORT FEASIBILITY | Web / Loopback | RUNTIME_TEST_PENDING_P01 (Track P01-D) | `docs/11_CHATGPT_TOOL_SURFACE.md` | Feasibility on target ChatGPT Plus account must be proven before P02 | High (Gating decision for P02/P05) |

| **Secure MCP Tunnel** | `openai/tunnel-client` `v0.0.14` | Windows x64 | RUNTIME_TEST_IN_PROGRESS (P01-D3A) | `docs/sources/11_OPENAI_API_MCP_RUNTIME.md` | Outbound HTTPS tunnel to OpenAI control plane; binary downloaded and verified | Medium (Official Upstream) |
| **Responses API MCP** | OpenAI `v1/responses` | Cloud / API | RUNTIME_TEST_IN_PROGRESS (P01-D3A) | `docs/sources/11_OPENAI_API_MCP_RUNTIME.md` | Invokes MCP tools via `tunnel_id`; supports `require_approval` lifecycle | Medium (Cloud API) |
| **MCP SDK Standard** | `@modelcontextprotocol/sdk` (TypeScript) | Node.js | VERIFIED_LOCAL | `docs/sources/10_OPENAI_PLUGIN_PLATFORM.md` | Official SDK for Streamable HTTP server on loopback | Low (Standard SDK) |
| **ChatGPT Plus Published Plugin** | Universal Plugin Directory (`With MCP`) | Web / HTTPS | GATED_PENDING_IDENTITY_VERIF (Phase P01-D) | `docs/sources/10_OPENAI_PLUGIN_PLATFORM.md` | Requires verified publisher identity, domain challenge verification, and OpenAI review | High (External Policy Gate) |
