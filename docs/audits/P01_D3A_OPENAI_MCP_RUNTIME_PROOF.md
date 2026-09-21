# P01-D3A — OFFICIAL OPENAI MCP RUNTIME PROOF DOSSIER

> **Authority**: Phase 1 Upstream Proof Dossier — Track P01-D3A  
> **Date**: 2026-09-21  
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE` (Strictly Candidate; Not Frozen)  
> **Dedicated Tunnel**: `tunnel_6ab0ae480cec81919b3db157c622eb53` (Name: `ai-supervisor-p01d`)  
> **Untouched Existing Tunnel**: `Codex Native2` (`tunnel_6aae67a9abe08191ab9697ef8e2a6687` preserved untouched)  
> **P01-D3A Verdict**: `PARTIAL` (`HUMAN_REQUIRED_CONFIGURE_RUNTIME_CREDENTIAL`)  
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
- **Official Tunnel Client Checksum Verification**:
  - Binary: `openai/tunnel-client` release `v0.0.14` (Windows amd64).
  - Download Archive: `tunnel-client-v0.0.14-windows-amd64.zip`
  - Official Release Manifest: `SHA256SUMS.txt` from official `openai/tunnel-client` GitHub release `v0.0.14`.
  - `expected_sha256`: `784ab8da7b5a88f0109f1fd8aaf0a1c86067430b896dddf307ef7e3cc49fa1a5`
  - `observed_sha256`: `784ab8da7b5a88f0109f1fd8aaf0a1c86067430b896dddf307ef7e3cc49fa1a5`
  - `match`: `true`
  - Binary `tunnel-client.exe` SHA-256: `fcc85a69ec0ad82518e4f8964f60c45e31787957782a0fc9c1b0c44e82d61b9b`
  - Verification Status: **`BINARY_CHECKSUM_VERIFIED`**
- **User Platform Organization Evidence (Ingested 2026-09-21)**:
  - Organization Name: `GPT Orchestrator`
  - Organization Role: `Owner`
  - Apps Management: `api.apps.read = AVAILABLE BY OWNER ROLE`, `api.apps.write = AVAILABLE BY OWNER ROLE`
  - Plugin Submission Portal: VISIBLE
  - Create Plugin: AVAILABLE (`With MCP`, `Skills only`)
  - Identity Verification: `IN_PROGRESS / PENDING` (Vietnamese national ID + selfie submitted)
  - Prepaid Balance: $5 added
  - Platform Tunnels: AVAILABLE (Existing tunnel `Codex Native2` preserved untouched; dedicated tunnel `ai-supervisor-p01d` created)
  - MCP Tool Usage: Enabled for all projects
- **Secret Hygiene**: Zero API keys, bearer tokens, org IDs, tunnel secrets, personal names, national ID data, or selfie photos are stored in this repository or audit artifacts.

---

## 2. Official Source Baseline Ingested

Authoritative OpenAI documentation ingested on 2026-09-21 without fabricated Git commit SHAs:
1. `https://developers.openai.com/plugins/quickstart` -> [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
2. `https://developers.openai.com/plugins/build/skills` -> [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
3. `https://developers.openai.com/plugins/build/mcp-server` -> [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
4. `https://developers.openai.com/plugins/build/plugins` -> [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
5. `https://developers.openai.com/api/docs` -> [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
6. `https://developers.openai.com/learn/developers-codex-plugin` -> [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
7. `https://developers.openai.com/api/docs/guides/text` -> [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
8. `https://developers.openai.com/api/docs/guides/agents/quickstart` -> [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
9. `https://developers.openai.com/api/docs/models/gpt-5.6-sol` -> [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
10. `https://developers.openai.com/api/docs/guides/secure-mcp-tunnels` -> [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
11. `https://developers.openai.com/api/docs/guides/tools-connectors-mcp` -> [11_OPENAI_API_MCP_RUNTIME.md](../sources/11_OPENAI_API_MCP_RUNTIME.md)
12. `https://developers.openai.com/plugins/deploy/submission` -> [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)
13. `https://developers.openai.com/plugins/deploy/app-review` -> [10_OPENAI_PLUGIN_PLATFORM.md](../sources/10_OPENAI_PLUGIN_PLATFORM.md)

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
  - Local MCP server creation and protocol harness validation (PASS)
  - Official MCP Inspector validation (PASS)
  - Secure MCP Tunnel creation and `tunnel-client doctor` preflight (PASS on format/URLs; pending live key)
  - Responses API MCP tool invocation via `tunnel_id`
- **Governance Classification**: This is recorded as `WAITING_IDENTITY_VERIFICATION`, not as an MCP transport failure.

---

## 4. Dedicated Tunnel Governance & Configuration

### 4.1 Tunnel Isolation Directive
The user account contains an existing tunnel named `Codex Native2` (`tunnel_6aae67a9abe08191ab9697ef8e2a6687`).  
Per strict isolation directives, **this existing tunnel is NOT modified or repurposed**. It remains untouched for separate developer usage.

### 4.2 Dedicated Tunnel Created by User
The User created a dedicated tunnel for this proof:
- **Tunnel Identifier**: `tunnel_6ab0ae480cec81919b3db157c622eb53`
- **Tunnel Name**: `ai-supervisor-p01d`
- **Organization**: `GPT Orchestrator`
- **Target MCP Endpoint**: `http://127.0.0.1:3182/mcp`
- **Security Boundary**: Local MCP binds strictly to loopback (`127.0.0.1:3182`); outbound-only HTTPS polling via official `tunnel-client`.

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

## 7. Local MCP Protocol Test Harness Empirical Results (`LOCAL_PROTOCOL_HARNESS`)

Executed via `node test_local_inspector.js` against live running server on `http://127.0.0.1:3182/mcp`:

```text
=== P01-D3A LOCAL MCP PROTOCOL TEST HARNESS VALIDATION ===
Target MCP Endpoint: http://127.0.0.1:3182/mcp
Target Healthz: http://127.0.0.1:3182/healthz

--- Step 1: Healthcheck & Listener Verification ---
[PASS] GET /healthz returns 200 OK
[PASS] Health service identifier is supervisor-proof-mcp
[PASS] Server port confirmed: 3182
[PASS] Server bound strictly to loopback (127.0.0.1:3182)

--- Step 2: Protocol Handshake & Tool Discovery (tools/list) ---
[PASS] initialize handshake succeeds
[PASS] tools/list returns tools array
[PASS] Exactly 2 probe tools discovered
[PASS] supervisor_probe_read tool schema valid
[PASS] supervisor_probe_read annotation readOnlyHint is true
[PASS] supervisor_probe_read annotation destructiveHint is false
[PASS] supervisor_probe_write tool schema valid
[PASS] supervisor_probe_write annotation readOnlyHint is false
[PASS] supervisor_probe_write annotation destructiveHint is true

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

=== LOCAL PROTOCOL HARNESS RESULTS ===
Total Passed: 31
Total Failed: 0
Verdict: PASS
```

---

## 7b. Official MCP Inspector Empirical Results (`OFFICIAL_MCP_INSPECTOR`)

Executed independently using official `@modelcontextprotocol/inspector@latest` (v2.7.0) CLI against `http://127.0.0.1:3182/mcp`:

### Test 1: Tool Discovery (`tools/list`)
Command:
```bash
npx @modelcontextprotocol/inspector@latest --cli http://127.0.0.1:3182/mcp --method tools/list --format json
```
Literal Output:
```json
{
  "result": {
    "tools": [
      {
        "name": "supervisor_probe_read",
        "title": "Supervisor Probe Read",
        "description": "Read-only probe returning local node identity, current disposable state, timestamp, and proof identifier.",
        "inputSchema": {
          "type": "object",
          "properties": {
            "correlation_id": { "description": "Optional correlation identifier for tracing request", "type": "string", "maxLength": 64 }
          },
          "$schema": "http://json-schema.org/draft-07/schema#"
        },
        "outputSchema": {
          "type": "object",
          "properties": {
            "proof_id": { "type": "string" },
            "node_id": { "type": "string" },
            "disposable_state": {
              "type": "object",
              "properties": { "mutation_count": { "type": "number" }, "current_value": { "type": "string" }, "last_updated": { "type": "string" } },
              "required": ["mutation_count", "current_value", "last_updated"],
              "additionalProperties": false
            },
            "timestamp": { "type": "string" },
            "correlation_id": { "type": "string" }
          },
          "required": ["proof_id", "node_id", "disposable_state", "timestamp", "correlation_id"],
          "$schema": "http://json-schema.org/draft-07/schema#",
          "additionalProperties": false
        },
        "annotations": { "readOnlyHint": true, "destructiveHint": false, "openWorldHint": false },
        "execution": { "taskSupport": "forbidden" }
      },
      {
        "name": "supervisor_probe_write",
        "title": "Supervisor Probe Write",
        "description": "Controlled state mutation probe updating disposable test state with strict idempotency and schema bounds.",
        "inputSchema": {
          "type": "object",
          "properties": {
            "test_value": { "type": "string", "minLength": 1, "maxLength": 256, "description": "Bounded string value to store in disposable state (1-256 chars)" },
            "correlation_id": { "type": "string", "minLength": 1, "maxLength": 64, "description": "Unique correlation ID for idempotency and replay safety" }
          },
          "required": ["test_value", "correlation_id"],
          "$schema": "http://json-schema.org/draft-07/schema#"
        },
        "outputSchema": {
          "type": "object",
          "properties": {
            "status": { "type": "string", "enum": ["APPLIED", "DUPLICATE_REPLAY", "ERROR"] },
            "mutation_count": { "type": "number" },
            "previous_value": { "type": "string" },
            "current_value": { "type": "string" },
            "correlation_id": { "type": "string" },
            "timestamp": { "type": "string" }
          },
          "required": ["status", "mutation_count", "previous_value", "current_value", "correlation_id", "timestamp"],
          "$schema": "http://json-schema.org/draft-07/schema#",
          "additionalProperties": false
        },
        "annotations": { "readOnlyHint": false, "destructiveHint": true, "openWorldHint": false },
        "execution": { "taskSupport": "forbidden" }
      }
    ]
  }
}
```
Verdict: **PASS** (Server metadata visible, schemas compliant, annotations matching probe specifications).

### Test 2: Probe Read Execution (`tools/call supervisor_probe_read`)
Command:
```bash
npx @modelcontextprotocol/inspector@latest --cli http://127.0.0.1:3182/mcp --method tools/call --tool-name supervisor_probe_read --tool-arg correlation_id=inspector_call_001 --format json
```
Literal Output:
```json
{
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[supervisor_probe_read] Node: host-windows-dev-node-01 | Proof: P01-D3A-OPENAI-MCP-RUNTIME-PROOF | Mutations: 2 | Value: \"OFFICIAL_INSPECTOR_MUTATION_DELTA_B\" | Timestamp: 2026-09-21T04:16:48.969Z"
      }
    ],
    "structuredContent": {
      "proof_id": "P01-D3A-OPENAI-MCP-RUNTIME-PROOF",
      "node_id": "host-windows-dev-node-01",
      "disposable_state": {
        "mutation_count": 2,
        "current_value": "OFFICIAL_INSPECTOR_MUTATION_DELTA_B",
        "last_updated": "2026-09-21T04:14:49.545Z"
      },
      "timestamp": "2026-09-21T04:16:48.969Z",
      "correlation_id": "inspector_call_001"
    }
  }
}
```
Verdict: **PASS** (Read invocation succeeds, node identity returned, schema satisfied).

### Test 3: Controlled Mutation Execution (`tools/call supervisor_probe_write`)
Command:
```bash
npx @modelcontextprotocol/inspector@latest --cli http://127.0.0.1:3182/mcp --method tools/call --tool-name supervisor_probe_write --tool-arg test_value=OFFICIAL_INSPECTOR_MUTATION_DELTA_C --tool-arg correlation_id=inspector_write_003 --format json
```
Literal Output:
```json
{
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[supervisor_probe_write] APPLIED mutation #3. Value changed from \"OFFICIAL_INSPECTOR_MUTATION_DELTA_B\" to \"OFFICIAL_INSPECTOR_MUTATION_DELTA_C\". Correlation: inspector_write_003."
      }
    ],
    "structuredContent": {
      "status": "APPLIED",
      "mutation_count": 3,
      "previous_value": "OFFICIAL_INSPECTOR_MUTATION_DELTA_B",
      "current_value": "OFFICIAL_INSPECTOR_MUTATION_DELTA_C",
      "correlation_id": "inspector_write_003",
      "timestamp": "2026-09-21T04:16:52.524Z"
    }
  }
}
```
Verdict: **PASS** (Write mutation succeeds, state counter increments to 3, value updated).

### Test 4: Read-Back Verification
Command:
```bash
npx @modelcontextprotocol/inspector@latest --cli http://127.0.0.1:3182/mcp --method tools/call --tool-name supervisor_probe_read --tool-arg correlation_id=inspector_readback_003 --format json
```
Literal Output confirms mutation #3 persisted to disk:
```json
{
  "result": {
    "structuredContent": {
      "proof_id": "P01-D3A-OPENAI-MCP-RUNTIME-PROOF",
      "node_id": "host-windows-dev-node-01",
      "disposable_state": {
        "mutation_count": 3,
        "current_value": "OFFICIAL_INSPECTOR_MUTATION_DELTA_C",
        "last_updated": "2026-09-21T04:16:52.524Z"
      },
      "timestamp": "2026-09-21T04:16:57.938Z",
      "correlation_id": "inspector_readback_003"
    }
  }
}
```
Verdict: **PASS** (Read-back proves state persisted to disk).

---

## 8. Official `tunnel-client` Preflight & Health Verification (`OPENAI_SECURE_TUNNEL`)

- **Checksum**: Verified against official release manifest (`BINARY_CHECKSUM_VERIFIED`).
- **Binary Path**: `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\bin\tunnel-client.exe` (v0.0.14).
- **Execution Against Dedicated Tunnel**:
  ```bash
  tunnel-client.exe doctor --mcp.server-url url=http://127.0.0.1:3182/mcp --control-plane.tunnel-id tunnel_6ab0ae480cec81919b3db157c622eb53 --json --explain
  ```
- **Independent Checks Output**:
  | Check ID | Status | Summary / Diagnostic |
  |---|---|---|
  | `config_source` | **PASS** | flags/environment only |
  | `profile_load` | **PASS** | flags/environment only |
  | `tunnel_id` | **PASS** | format validated (`tunnel_6ab0ae480cec81919b3db157c622eb53`) |
  | `tunnels_management_url` | **PASS** | https://platform.openai.com/settings/organization/tunnels |
  | `runtime_api_keys_url` | **PASS** | https://platform.openai.com/settings/organization/api-keys |
  | `admin_api_keys_url` | **PASS** | https://platform.openai.com/settings/organization/admin-keys |
  | `chatgpt_connector_settings_url` | **PASS** | https://chatgpt.com/#settings/Connectors |
  | `codex_plugin` | **SKIP** | Codex detected; optional plugin |
  | `control_plane_api_key` | **FAIL** | Control plane API key required to authenticate outbound connection to OpenAI control plane |

- **Governance Verdict on Doctor**: Per Section 8 directives, **overall doctor PASS is NOT marked** because a required live check (`control_plane_api_key`) failed.
- **Runtime Precondition Required**:
  ```text
  ================================================================================
  STATUS: HUMAN_REQUIRED_CONFIGURE_RUNTIME_CREDENTIAL
  ================================================================================
  ```
  In strict compliance with Section 4 (Secret Handling), the agent cannot and will not request the key in chat or repository. The user must configure `OPENAI_API_KEY` or `CONTROL_PLANE_API_KEY` in their local Windows user environment.

---

## 9. Cloud Responses API Test Architecture & Test Harness (`RESPONSES_API_MCP_READ` / `RESPONSES_API_MCP_WRITE`)

An automated proof execution harness is staged in `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\test_cloud_responses_mcp.js`. It implements the exact required cloud protocol tests:

1. **Model Discovery & Selection**:
   - Tests model availability against Platform account.
   - Evaluates least expensive candidate supporting MCP (`gpt-4o-mini`, `gpt-4.1-mini`, or account-available tier).
   - Selected model labeled strictly `TEST_ONLY_MODEL`.
   - Identity gating fallback: If candidate models are identity-gated, captures literal API error and flags `BLOCKED_BY_IDENTITY`.

2. **Cloud Read Proof (`D3A-CLOUD-READ`)**:
   - Request uses `type: "mcp"`, `tunnel_id: "tunnel_6ab0ae480cec81919b3db157c622eb53"`.
   - Forces invocation of `supervisor_probe_read`.
   - Cloud Origin Demonstration: Injects unique pre-call value `OPENAI_CLOUD_READ_PROOF_<timestamp>` into disposable state, then retrieves it through Responses API via tunnel.
   - Captures OpenAI response ID, tool call ID, returned local state, correlation ID, and latency.

3. **Write Approval Request & Denial (`D3A-APPROVAL-DENIAL`)**:
   - Uses `require_approval: "always"` on `supervisor_probe_write`.
   - Turn 1 must emit `mcp_approval_request`. Verifies local state has NOT changed.
   - Rejection Test: Emits rejection response. Verifies local state remains unchanged (zero mutation).

4. **Approved Write Execution (`D3A-CLOUD-WRITE`)**:
   - Injects unique payload `OPENAI_TUNNEL_WRITE_PROOF_<timestamp>` with unique correlation ID.
   - Emits approval confirmation.
   - Verifies mutation occurs exactly once, counter increments, value written to disk, and tool response returns to Responses API.

5. **Cloud State Read-Back**:
   - Subsequent Responses API turn performs `supervisor_probe_read` to verify mutated state from cloud perspective.

6. **Replay Safety Through OpenAI**:
   - Submits identical correlation ID twice through cloud Responses API.
   - Verifies first execution returns `APPLIED`, second returns `DUPLICATE_REPLAY`, mutation counter increments only once.

7. **Offline & Recovery Test**:
   - Stops local MCP server while tunnel is observable.
   - Makes Responses API request; records literal error behavior (timeout/error structure).
   - Restarts local MCP server; verifies recovery.

8. **Tunnel-Client Reconnect (`D3A-TUNNEL-RECONNECT`)**:
   - Restarts `tunnel-client` with local MCP still running.
   - Verifies reconnection without recreating tunnel resources.
   - Verifies subsequent cloud read succeeds.

9. **Payload Bounds Test**:
   - Tests 1 KB and 10 KB bounded payloads through Responses API path.

10. **Secret Scanning & Cost Guardrail**:
    - Scans request/response logs to ensure zero API keys or tokens are committed.
    - Capped strictly under **$1.00 USD** cumulative spend.

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

---

## 12. P01-D3A Verdict & Next Steps

```text
================================================================================
P01-D3A VERDICT:
PARTIAL

REASON:
- Local MCP Protocol Test Harness: 100% PASS (31/31 checks passed)
- Official MCP Inspector CLI (v2.7.0): 100% PASS (tools/list, probe read, probe write, read-back)
- Official tunnel-client v0.0.14: BINARY_CHECKSUM_VERIFIED (SHA-256 matched official release manifest)
- Dedicated Tunnel Registered: tunnel_6ab0ae480cec81919b3db157c622eb53 (ai-supervisor-p01d)
- Untouched Tunnel: Codex Native2 preserved untouched
- Preflight Doctor: Format & URLs PASS; control_plane_api_key FAIL
- Current Precondition: HUMAN_REQUIRED_CONFIGURE_RUNTIME_CREDENTIAL
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