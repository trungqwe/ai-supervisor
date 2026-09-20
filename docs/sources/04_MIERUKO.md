# SOURCE DOSSIER: 04 — MIERUKO WORKBENCH

## 1. Metadata
- **Repository**: `Mieruko/MCP_Plugins_With_ChatGPTWeb`
- **Role in Architecture**: Passive Design Source (Workspace Binding & Slim Tool Interface)
- **Reference Commit**: `main` (`2026-09-10`)
- **Inspection Date**: 2026-09-20
- **License**: MIT
- **Primary Language**: Python / TypeScript

---

# 2. Capabilities Evaluated & Adopted
- **Workspace Binding Pattern**: Explicitly binding a ChatGPT Web session to a registered local project.
- **Slim Tool Philosophy**: Exposing minimal, high-signal tools to avoid LLM context bloat.
- **Checkpoints & Permissions**: Providing human checkpoints before major mutations.

---

# 3. Capabilities Explicitly Rejected
- **Execution Engine Duplication**: We reject Mieruko's local script executor in favor of Agent Orchestrator.
- **Direct Code Generation**: Mieruko's template generators are not used.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` in `PairRegistry` and `ChatGPTToolSurface`.

---

# 5. SOURCE EVIDENCE

### Evidence Item 4.1: Workspace Binding Mechanics
- **Claim**: ChatGPT sessions can be deterministically bound to specific workspace paths via session tokens.
- **Repository**: `Mieruko/MCP_Plugins_With_ChatGPTWeb`
- **Reference**: `main`
- **Source File / Module**: `plugins/workspace/binding.py`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 4.2: Slim Tool Surface
- **Claim**: Reducing exposed MCP tools from 30+ to under 15 significantly improves ChatGPT reasoning focus.
- **Repository**: `Mieruko/MCP_Plugins_With_ChatGPTWeb`
- **Reference**: `docs/tool_surface_optimization.md`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
