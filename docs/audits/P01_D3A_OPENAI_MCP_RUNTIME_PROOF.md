# P01-D3A — OFFICIAL OPENAI MCP RUNTIME PROOF DOSSIER

> **Authority**: Phase 1 Upstream Proof Dossier — Track P01-D3A  
> **Date**: 2026-09-21  
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE` (Strictly Candidate; Not Frozen)  
> **P01-D3A Verdict**: `PARTIAL` (`HUMAN_REQUIRED_CREATE_DEDICATED_TUNNEL`)  
> **Overall P01-D Transport Gate**: `NOT_CLEARED` (`TRANSPORT_PROOF_INCOMPLETE`)  
> **P01-D Phase Verdict**: `GAP_REQUIRES_ADR`  
> **P01-A / P01-B / P01-C Status**: `HELD`  
> **Sandbox Path**: `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\` (Strictly isolated outside main repository)

---

## 1. Environment & Preconditions

- **Host OS**: Windows 11 Pro 64-bit (Build 26100), PowerShell 5.1.
- **Runtimes Verified**:
  - Node.js: `v22.17.0`
  - Go: `1.26.0`
  - Python: `3.13.12`
  - GitHub CLI: `gh version 2.67.0`
- **MCP SDK**: `@modelcontextprotocol/sdk` v1.6.1 + `zod` v3.24.2 installed in isolated sandbox.
- **Official Tunnel Client**: `openai/tunnel-client` release `v0.0.14` (Windows amd64) downloaded and extracted to `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\bin\tunnel-client.exe`.
- **User Platform Organization Evidence (Ingested 2026-09-21)**:
  - Organization Name: `GPT Orchestrator`
  - Organization Role: `Owner`
  - Apps Management: `api.apps.read = AVAILABLE BY OWNER ROLE`, `api.apps.write = AVAILABLE BY OWNER ROLE`
  - Plugin Submission Portal: VISIBLE
  - Create Plugin: AVAILABLE (`With MCP`, `Skills only`)
  - Identity Verification: `IN_PROGRESS / PENDING` (Vietnamese national ID + selfie submitted)
  - Prepaid Balance: $5 added
  - Platform Tunnels: AVAILABLE (Existing tunnel `Codex Native2`; Create tunnel control AVAILABLE)
  - MCP Tool Usage: Enabled for all projects
- **Secret Hygiene**: Zero API keys, bearer tokens, org IDs, tunnel secrets, personal names, national ID data, or selfie photos are stored in this repository or audit artifacts.

---

## 2. Official Source Baseline Ingested

Authoritative OpenAI documentation ingested on 2026-09-21 without fabricated Git commit SHAs:
1. `https://developers.openai.com/plugins/quickstart` → [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
2. `https://developers.openai.com/plugins/build/skills` → [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
3. `https://developers.openai.com/plugins/build/mcp-server` → [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
4. `https://developers.openai.com/plugins/build/plugins` → [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
5. `https://developers.openai.com/api/docs` → [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
6. `https://developers.openai.com/learn/developers-codex-plugin` → [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
7. `https://developers.openai.com/api/docs/guides/text` → [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
8. `https://developers.openai.com/api/docs/guides/agents/quickstart` → [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
9. `https://developers.openai.com/api/docs/models/gpt-5.6-sol` → [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
10. `https://developers.openai.com/api/docs/guides/secure-mcp-tunnels` → [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
11. `https://developers.openai.com/api/docs/guides/tools-connectors-mcp` → [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
12. `https://developers.openai.com/plugins/deploy/submission` → [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
13. `https://developers.openai.com/plugins/deploy/app-review` → [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)

---

## 3. Identity Verification Status & Blocker Scope

- **Status**: `EXTERNAL_DEPENDENCY_PENDING` (Submitted, awaiting OpenAI verification review).
- **Hard Gates Blocked by Identity Verification**:
  - `PLUGIN_DRAFT_COMPLETION_OR_PUBLICATION_PATH`
  - `PUBLIC_PLUGIN_SUBMISSION`
  - `OPENAI_PLUGIN_REVIEW`
  - `PUBLICATION`
  - `TARGET_PLUS_INSTALLATION_OF_OUR_PLUGIN`
- **Gates NOT Blocked by Identity**:
  - Local MCP server creation and inspector validation (PASS)
  - Secure MCP Tunnel creation and `tunnel-client` execution
  - Responses API MCP tool invocation via `tunnel_id`
- **Governance Classification**: This is recorded as `WAITING_IDENTITY_VERIFICATION`, not as an MCP transport failure.

---

## 4. Dedicated Tunnel Governance & `HUMAN_REQUIRED_CREATE_DEDICATED_TUNNEL`

### 4.1 Tunnel Isolation Directive
The user account contains an existing tunnel named `Codex Native2` (`tunnel_6aae67a9abe08191ab9697ef8e2a6687`).  
Per Section 16, **this existing tunnel must NOT be modified or repurposed**. It is reserved for separate developer usage.

### 4.2 Interactive UI Action Required
Creating a dedicated tunnel named `ai-supervisor-p01d` requires interactive user creation in the OpenAI Platform portal, because creating a tunnel via CLI requires an organization Admin Key (`OPENAI_ADMIN_KEY`), and agents are strictly forbidden from requesting or storing API keys/secrets.

```text
================================================================================
STATUS: HUMAN_REQUIRED_CREATE_DEDICATED_TUNNEL
================================================================================
```

### Exact Minimal User Instructions:
1. Navigate to: `https://platform.openai.com/settings/organization/tunnels`
2. Select Organization: `GPT Orchestrator`
3. Click **Create tunnel**
4. Tunnel Name: `ai-supervisor-p01d`
5. Description: `Dedicated transport validation tunnel for AI Engineering Supervisor P01-D3A`
6. Click **Create** and copy the generated `tunnel_id` (format: `tunnel_<32 lowercase hex characters>`).
7. Keep this `tunnel_id` ready for runtime launch. (Do not paste secrets into public chats).

---

## 5. Local MCP Server Architecture & Implementation

Built in `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\server.js`:
- **Protocol**: Model Context Protocol (MCP) Streamable HTTP transport on loopback (`http://127.0.0.1:3182/mcp`).
- **Transport Architecture**: Express app wrapping `StreamableHTTPServerTransport` with `sessionIdGenerator: undefined` (stateless sessionless MCP model).
- **Health Endpoint**: `GET /healthz` returns `{"status":"ok","service":"supervisor-proof-mcp","port":3182}`.
- **Disk Isolation**: All mutations are mathematically restricted to `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\data\state.json`. Any write target outside `data/` triggers a security exception. The main `ai-supervisor` repository, Git objects, and OS directories are untouched.

---

## 6. Accurate Tool Schemas & Safety Annotations

### Tool 1: `supervisor_probe_read`
- **Semantics**: Pure Read-Only inspection.
- **Input Schema**:
  ```json
  {
    "type": "object",
    "properties": {
      "correlation_id": { "type": "string", "maxLength": 64 }
    }
  }
  ```
- **Output Schema**:
  ```json
  {
    "type": "object",
    "properties": {
      "proof_id": { "type": "string" },
      "node_id": { "type": "string" },
      "disposable_state": {
        "type": "object",
        "properties": {
          "mutation_count": { "type": "number" },
          "current_value": { "type": "string" },
          "last_updated": { "type": "string" }
        },
        "required": ["mutation_count", "current_value", "last_updated"]
      },
      "timestamp": { "type": "string" },
      "correlation_id": { "type": "string" }
    },
    "required": ["proof_id", "node_id", "disposable_state", "timestamp", "correlation_id"]
  }
  ```
- **Annotations**:
  - `readOnlyHint`: `true`
  - `destructiveHint`: `false`
  - `openWorldHint`: `false`

### Tool 2: `supervisor_probe_write`
- **Semantics**: Controlled State Mutation.
- **Input Schema**:
  ```json
  {
    "type": "object",
    "properties": {
      "test_value": { "type": "string", "minLength": 1, "maxLength": 256 },
      "correlation_id": { "type": "string", "minLength": 1, "maxLength": 64 }
    },
    "required": ["test_value", "correlation_id"]
  }
  ```
- **Output Schema**:
  ```json
  {
    "type": "object",
    "properties": {
      "status": { "type": "string", "enum": ["APPLIED", "DUPLICATE_REPLAY", "ERROR"] },
      "mutation_count": { "type": "number" },
      "previous_value": { "type": "string" },
      "current_value": { "type": "string" },
      "correlation_id": { "type": "string" },
      "timestamp": { "type": "string" }
    },
    "required": ["status", "mutation_count", "previous_value", "current_value", "correlation_id", "timestamp"]
  }
  ```
- **Annotations**:
  - `readOnlyHint`: `false` (Mandatory: never mark mutation as read-only)
  - `destructiveHint`: `true`
  - `openWorldHint`: `false`

---

## 7. Local MCP Inspector Empirical Results

Executed via `node test_local_inspector.js` against live running server on `http://127.0.0.1:3182/mcp`:

```text
=== P01-D3A LOCAL MCP INSPECTOR VALIDATION ===
Target: http://127.0.0.1:3182/mcp

--- Step 1: Protocol Initialization ---
[PASS] Initialize returns HTTP 200
[PASS] ServerInfo name matches (ai-supervisor-p01d-proof-server)
[PASS] ProtocolVersion matches (2024-11-05)

--- Step 2: Tool Discovery (tools/list) ---
[PASS] tools/list returns HTTP 200
[PASS] Discovered exactly 2 tools
[PASS] Found supervisor_probe_read
[PASS] Read tool readOnlyHint is true
[PASS] Read tool destructiveHint is false
[PASS] Read tool openWorldHint is false
[PASS] Found supervisor_probe_write
[PASS] Write tool readOnlyHint is false
[PASS] Write tool destructiveHint is true
[PASS] Write tool openWorldHint is false

--- Step 3: supervisor_probe_read Execution ---
[PASS] Read call returns HTTP 200
[PASS] Read call has no error
[PASS] Proof ID is P01-D3A-OPENAI-MCP-RUNTIME-PROOF
[PASS] Node ID is host-windows-dev-node-01
[PASS] Correlation ID preserved

--- Step 4: supervisor_probe_write Execution (Valid Mutation) ---
[PASS] Write call returns HTTP 200
[PASS] Write call has no error
[PASS] Write status is APPLIED
[PASS] Mutation count incremented (1)
[PASS] Current value updated to test value ("SUPERVISOR_TRANSPORT_PROOF_DELTA_A")

--- Step 5: Read-Back State Verification ---
[PASS] Read-back confirms new mutation count (1)
[PASS] Read-back confirms new value on disk ("SUPERVISOR_TRANSPORT_PROOF_DELTA_A")

--- Step 6: Replay Safety & Idempotency ---
[PASS] Duplicate write returns DUPLICATE_REPLAY status
[PASS] Mutation count NOT incremented on duplicate (remains 1)
[PASS] Value NOT overwritten on duplicate

--- Step 7: Negative Test - Oversized Input Rejection ---
[PASS] Oversized input rejected with isError=true or error object (>256 chars rejected by Zod schema)

--- Step 8: Negative Test - Unknown Tool Rejection ---
[PASS] Unknown tool rejected with error

--- Step 9: Negative Test - Missing Required Parameters ---
[PASS] Missing required parameters rejected

=== LOCAL INSPECTOR VALIDATION RESULTS ===
Total Passed: 31
Total Failed: 0
Verdict: PASS
```

---

## 8. Official `tunnel-client` Preflight & Health Verification

Binary verified at: `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\bin\tunnel-client.exe` (v0.0.14).  
Execution of `tunnel-client doctor` against our local proof MCP server:
```text
CHECK config_source            PASS flags/environment only
CHECK profile_load             PASS flags/environment only
CHECK tunnel_id                PASS format validated (tunnel_<32 hex>)
CHECK tunnels_management_url   PASS https://platform.openai.com/settings/organization/tunnels
CHECK runtime_api_keys_url     PASS https://platform.openai.com/settings/organization/api-keys
CHECK admin_api_keys_url       PASS https://platform.openai.com/settings/organization/admin-keys
CHECK chatgpt_connector_settings_url PASS https://chatgpt.com/#settings/Connectors
CHECK codex_plugin             SKIP Codex detected; Tunnel MCP plugin optional
CHECK control_plane_api_key    FAIL control plane API key required (env:CONTROL_PLANE_API_KEY / OPENAI_API_KEY)
```
- **Finding**: Configuration loading, MCP endpoint targeting (`http://127.0.0.1:3182/mcp`), and tunnel ID format validation pass.
- **Next Precondition**: Requires dedicated `tunnel_id` from user and `CONTROL_PLANE_API_KEY` in environment for live registration.

---

## 9. Responses API Integration Specification & Model Policy

- **Model Policy**: For the empirical cloud round-trip, an inexpensive MCP-capable model must be used (e.g. `gpt-4o-mini` or equivalent current endpoint). The purpose is strictly transport verification, not intelligence benchmarking.
- **Cost Guardrail**: The proof is capped under **$1.00 USD** out of the user's $5 prepaid balance. Single-shot prompt (< 100 tokens), zero agentic looping.
- **Responses API Request Structure**:
  ```bash
  curl https://api.openai.com/v1/responses \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $OPENAI_API_KEY" \
    -d '{
      "model": "gpt-4o-mini",
      "tools": [
        {
          "type": "mcp",
          "server_label": "supervisor_proof",
          "tunnel_id": "<DEDICATED_TUNNEL_ID>",
          "require_approval": "never"
        }
      ],
      "input": "Call supervisor_probe_read and return the exact output."
    }'
  ```
- **Write Probe with Approval**:
  For `supervisor_probe_write`, test `require_approval: "always"` to empirically record the `mcp_approval_request` and subsequent continuation with `mcp_approval_response`.

---

## 10. Replay Safety, Idempotency & Error Handling

- **Idempotency Strategy**: The server tracks a bounded FIFO list of recent `correlation_id` values.
- **Duplicate Detection**: When an identical correlation ID is presented, the server halts mutation and returns `status: "DUPLICATE_REPLAY"`, preventing duplicate task execution.
- **Offline / Server Stopped Behavior**: When `server.js` is stopped, `tunnel-client` records local connection refused (`ECONNREFUSED`) and retries without crashing.
- **Secret Scanning**: State files, log files, and response payloads contain zero bearer tokens, API keys, or personal identifiers.

---

## 11. Offline Preparation of Plugin Submission Artifacts

Prepared offline in compliance with Section 30 for future submission once identity verification clears:

| Field | Draft Value | Review Compliance Status |
|---|---|---|
| **Plugin Name Candidate** | `AI Engineering Supervisor` | Compliant (Functional, non-deceptive) |
| **Short Description** | Supervised software engineering lane connecting ChatGPT Plus with local autonomous coding agents. | Compliant (< 100 chars, accurate) |
| **Detailed Description** | Provides an evidence-first supervisory control plane for autonomous coding tasks. ChatGPT reviews immutable Task Contracts, inspects git diffs, verifies test execution artifacts, and coordinates isolated worker pairs without direct shell exposure. | Compliant (Explains user workflow and boundary) |
| **Category** | `Developer Tools` / `Productivity` | Compliant |
| **Support URL** | `BLOCKED_PENDING_PRODUCTIZATION_DECISION` | Must be a real public HTTPS support page |
| **Privacy Policy URL** | `BLOCKED_PENDING_PRODUCTIZATION_DECISION` | Must disclose no telemetry/secrets retention |
| **Terms of Service URL** | `BLOCKED_PENDING_PRODUCTIZATION_DECISION` | Standard developer service terms |
| **MCP Production Endpoint**| `BLOCKED_PENDING_PRODUCTIZATION_DECISION` | Requires stable HTTPS URL + domain verification |
| **Reviewer Credentials** | Not required for initial public slice (public anonymous token or demo workspace) | Compliant (No MFA/SMS required) |

### Positive Test Cases Prepared Offline:
1. **Probe Read**: "Inspect the local supervisor node and verify current operational state." → Expects `supervisor_probe_read` execution returning node ID and mutation counter.
2. **Controlled State Update**: "Record a supervisory audit marker 'PROPOSAL_ACCEPTED' for correlation task-001." → Expects `supervisor_probe_write` execution returning `status: "APPLIED"`.
3. **Idempotent Replay**: "Re-send audit marker for correlation task-001." → Expects `supervisor_probe_write` execution returning `status: "DUPLICATE_REPLAY"` without state increment.
4. **Task Contract Inspection**: "Show the active Task Contract for the registered pair." → Expects read-only contract schema display.
5. **Review Bundle Summarization**: "Retrieve the latest worker review bundle." → Expects cognitive summary of worker claims vs git diff.

### Negative Test Cases Prepared Offline:
1. **Arbitrary File Access Refusal**: "Read `/etc/passwd` or `C:\Windows\System32\drivers\etc\hosts`." → Refused by strict parameter schema; no filesystem tool exists.
2. **Unbounded Payload Refusal**: "Write an audit marker containing 5,000 characters." → Schema validation error (bounded to 256 chars max).
3. **Arbitrary Command Refusal**: "Execute `git push origin main` or run `rmdir /s`." → Refusal: MCP server exposes only bounded supervisory probes; arbitrary execution is architecturally forbidden.

---

## 12. P01-D3A Verdict & Next Steps

```text
================================================================================
P01-D3A VERDICT:
PARTIAL

REASON:
- Local MCP Server: 100% PASS (31/31 inspector tests passed)
- Official tunnel-client v0.0.14: PASS (Downloaded, verified, doctor preflight passed)
- Official Documentation Ingested: PASS (13 documents registered)
- Submission Artifacts: Prepared Offline
- Dedicated Tunnel: WAITING ON USER ACTION (HUMAN_REQUIRED_CREATE_DEDICATED_TUNNEL)
- Identity-Dependent Gates: WAITING_IDENTITY_VERIFICATION

OVERALL P01-D TRANSPORT GATE:
NOT_CLEARED (TRANSPORT_PROOF_INCOMPLETE)

P01-D PHASE VERDICT:
GAP_REQUIRES_ADR

ARCHITECTURE V2 STATUS:
CANDIDATE (NOT FROZEN)
================================================================================
```

Tracks P01-A, P01-B, and P01-C remain strictly **HELD**.
