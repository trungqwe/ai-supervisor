# 19. OPEN QUESTIONS & UNRESOLVED DECISIONS

> **Focus**: Unresolved Architecture & Technical Questions  
> **Status**: Recorded as UNDECIDED / GIẢ ĐỊNH / P01_PROOF_REQUIRED

---

# 1. Active Open Questions

### Q1: AO ↔ Agy Structured Completion & WorkerReport Delivery
- **Status**: `UNDECIDED — P01_PROOF_REQUIRED`
- **Context**: Pinned AO `v0.13.0` invokes Agy interactively (`--prompt-interactive`) rather than passing `--json-schema docs/schemas/worker-report.schema.json`.
- **Question**: Does AO expose enough session output telemetry for the Supervisor to normalize into a `WorkerReport`, or does the AOAdapter / Agy invocation require a custom integration mechanism?
- **Resolution Plan**: Address empirically in Phase P01 Track P01-C.

### Q2: ChatGPT Transport Protocol for Target User Environment
- **Status**: `UNDECIDED — P01_PROOF_REQUIRED` (Advanced from P05 to P01)
- **Context**: Target workflow relies on ChatGPT Web invoking the Supervisor's 12 tools without manual copy-pasting or browser automation.
- **Question**: What exact mechanism (local MCP stdio/SSE relay, ChatGPT App/Action with loopback bridge) is supported and accessible on the target user's ChatGPT Plus account?
- **Resolution Plan**: Mandatory early feasibility proof in Phase P01 Track P01-D.

### Q3: Supervisor Core Implementation Language
- **Status**: `UNDECIDED` (GIẢ ĐỊNH: TypeScript / Node.js vs. Go vs. Python)
- **Resolution Plan**: Finalize via ADR in Phase P02 following Phase P01 results.

### Q4: Local State Store Storage Engine
- **Status**: `UNDECIDED` (GIẢ ĐỊNH: SQLite vs. Embedded Key-Value / JSON file store)
- **Resolution Plan**: Finalize in Phase P02 domain implementation.
