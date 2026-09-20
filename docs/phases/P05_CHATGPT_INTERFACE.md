# PHASE SPECIFICATION: P05 — CHATGPT SUPERVISOR INTERFACE

## 1. Objective
Implement the 12 high-level Supervisor tools via local MCP relay / API for ChatGPT Web interaction.

## 2. Deliverables
- `SupervisorToolSurface` exposing the 12 high-level tools.
- Local MCP relay / HTTP daemon listening strictly on loopback.
- Token-based session authentication binding ChatGPT sessions to registered Pairs.

## 3. Exit Gate
- ChatGPT Web successfully binds to a registered project, calls `get_project_state`, and retrieves a test `ReviewBundle`.
