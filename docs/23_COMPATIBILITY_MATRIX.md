# 23. COMPATIBILITY MATRIX

> **Focus**: Platform, Dependency & Interface Compatibility Register  
> **Status**: Approved Baseline

---

| Component | Target Version | Platform | Test Status | Contract Baseline Ref | Known Issues | Upgrade Risk |
|---|---|---|---|---|---|---|
| **OS Host** | Windows 10/11 64-bit | Windows | BASELINE | NFR-001 | Requires ConPTY support | Low |
| **Untrivial AO** | `v0.13.0` | Windows x64 | PLANNED (P01) | `docs/sources/UPSTREAM_CONTRACT_BASELINE.md#sec-1` | None recorded in v0.13.0 release notes | Medium (Must follow update policy) |
| **Antigravity CLI** | `official-latest` | Windows x64 | PLANNED (P01) | `docs/sources/UPSTREAM_CONTRACT_BASELINE.md#sec-2` | Requires valid developer environment | Medium |
| **Git Runtime** | Git `>= 2.40` | Windows x64 | VERIFIED (`2.55.0`) | FR-008 | None | Very Low |
| **ChatGPT Interface**| MCP Draft 2024-11 / REST | Web / Loopback | PLANNED (P05) | `docs/11_CHATGPT_TOOL_SURFACE.md` | Transport protocol subject to OpenAI updates | Low (Isolated behind ToolSurface) |
