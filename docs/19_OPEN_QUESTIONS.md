# 19. OPEN QUESTIONS & UNRESOLVED DECISIONS

> **Focus**: Unresolved Architecture & Technical Questions
> **Status**: P01 Architecture Questions Resolved; P02 Implementation Decisions Pending (Updated Architecture V2.1 / Reaudit 001)

---

# 1. Resolved Architecture Questions (Phase P01)

### Q1: AO ↔ Agy Structured Completion & WorkerReport Delivery
- **Status**: `RESOLVED_BY_P01_C_AND_ADR_011`
- **Resolution**: Track P01-C proved that AO detects turn completion via the Agy `Stop` hook transitioning session activity state to `IDLE`. In accordance with **ADR-011** and **ADR-012**, structured worker reports are delivered via attempt-scoped files (`.supervisor/reports/<task_id>/<attempt_id>.json`) retrieved through AO's public workspace file API (`GET /api/v1/sessions/{id}/workspace/file?path=...`) under zero-trust Supervisor validation rules without requiring upstream code patches.

### Q2: ChatGPT Transport Protocol for Target User Environment
- **Status**: `RESOLVED_BY_P01_D3C (PROVEN_ON_TARGET_ACCOUNT)`
- **Resolution**: Track P01-D3C empirically proved the canonical target ChatGPT Plus transport chain: Target Personal ChatGPT Plus -> Developer Mode App -> OpenAI Secure MCP Tunnel -> official `tunnel-client` v0.0.14 -> private loopback MCP server (`127.0.0.1:3182/mcp`). This enables direct invocation of Supervisor domain tools without desktop browser automation and without manual copy-pasting.

---

# 2. Active Implementation Decisions (Phase P02 Decision Gate)

### Q3: Supervisor Core Implementation Language
- **Status**: `P02_DECISION_REQUIRED`
- **Candidates**: TypeScript / Node.js vs. Go vs. Python
- **Resolution Plan**: To be evaluated and decided under Change Governance via a dedicated ADR during the Phase P02 implementation decision gate prior to application code authoring.

### Q4: Local State Store Storage Engine
- **Status**: `P02_DECISION_REQUIRED`
- **Candidates**: SQLite vs. Embedded Key-Value / JSON file store
- **Resolution Plan**: To be evaluated and decided under Change Governance via a dedicated ADR during the Phase P02 implementation decision gate prior to state store implementation.