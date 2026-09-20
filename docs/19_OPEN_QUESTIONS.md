# 19. OPEN QUESTIONS & UNRESOLVED DECISIONS

> **Focus**: Unresolved Architecture & Technical Questions  
> **Status**: Recorded as UNDECIDED / GIẢ ĐỊNH

---

# 1. Open Architectural Questions

The following technical decisions lack conclusive empirical evidence and are intentionally deferred. No coding agent is permitted to make unilateral assumptions regarding these decisions until an approved ADR is established:

### Q1: Supervisor Implementation Language
- **Status**: `UNDECIDED` (GIẢ ĐỊNH: TypeScript / Node.js vs. Go vs. Python)
- **Considerations**:
  - Go aligns natively with Untrivial Agent Orchestrator and produces single static binaries.
  - TypeScript aligns with Model Context Protocol (MCP) TypeScript SDKs and fast JSON schema manipulation.
  - Python offers simplicity and fast prototyping.
- **Resolution Plan**: Address in Phase P02 following Phase P01 upstream proof.

### Q2: Local State Store Engine
- **Status**: `UNDECIDED` (GIẢ ĐỊNH: SQLite vs. Embedded Key-Value / JSON file store)
- **Considerations**:
  - SQLite provides ACID transactions, relational integrity, and easy querying.
  - Flat JSON file store / JSONL provides zero-dependency human readability.
- **Resolution Plan**: Evaluate during Phase P02 domain implementation.

### Q3: ChatGPT Transport Protocol for Target User Environment
- **Status**: `UNDECIDED` (GIẢ ĐỊNH: Local MCP stdio / SSE vs. Local HTTP REST Relay)
- **Considerations**:
  - Depends on user's ChatGPT Web account features (Custom GPT Actions vs. Desktop MCP client).
- **Resolution Plan**: Address during Phase P05 interface implementation.

### Q4: Post-V1 Control UI Framework
- **Status**: `UNDECIDED` (GIẢ ĐỊNH: Local Web app vs. Tauri vs. Lightweight terminal TUI)
- **Considerations**:
  - Deferred entirely to Phase P07 (Explicit non-goal for V1).
