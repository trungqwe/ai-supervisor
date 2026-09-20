# SOURCE DOSSIER: 04 — MIERUKO WORKBENCH

## 1. Metadata
- **Repository**: `Mieruko/MCP_Plugins_With_ChatGPTWeb`
- **Role in Architecture**: Passive Design Source (Workspace Binding & Slim Tool Interface)
- **Pinned Commit**: `4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df`
- **Commit Date**: 2026-09-19T20:17:06Z
- **License**: MIT
- **Primary Language**: TypeScript / Node.js

---

# 2. Capabilities Evaluated & Adopted
- **Workspace Binding Pattern**: Explicitly binding a ChatGPT Web session to a registered local project.
- **Slim Tool Surface Philosophy**: Exposing minimal, high-signal tools to avoid LLM context bloat.
- **Local MCP Integration**: Exposing tools via local MCP server.

---

# 3. Capabilities Explicitly Rejected & Corrected Claims
- **Execution Engine Duplication**: We reject Mieruko's local shell scripts in favor of AO.
- **Correction of Synthetic Path**: In previous drafts, `plugins/workspace/binding.py` was cited. Real codebase is TypeScript with real paths `src/admin/routes.ts` and `src/tools/`.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` in `PairRegistry` and `ChatGPTToolSurface`.

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-MRK-001
Claim: Mieruko implements a local administration and workspace routing server for ChatGPT Web plugins.
Repository: Mieruko/MCP_Plugins_With_ChatGPTWeb
Pinned tag: N/A
Pinned commit: 4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df
Evidence type: SOURCE_CODE
Exact evidence: src/admin/routes.ts
Section / symbol: registerAdminRoutes
Verification: VERIFIED
Confidence: HIGH
Notes: Mounts local administrative and workbench routes for ChatGPT plugin coordination.

Claim ID: CLM-MRK-002
Claim: Mieruko implements local MCP tools interface for ChatGPT Web.
Repository: Mieruko/MCP_Plugins_With_ChatGPTWeb
Pinned tag: N/A
Pinned commit: 4c09b02d87e63a765b7a9b3bbb0c28e7a1b750df
Evidence type: README
Exact evidence: README.md
Section / symbol: "Overview & Features"
Verification: VERIFIED
Confidence: HIGH
Notes: Details bridging ChatGPT Web to local workspaces using MCP tools.
