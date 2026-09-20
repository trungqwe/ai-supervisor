# P01-D — CHATGPT PLUS PUBLISHED PLUGIN TRANSPORT PROOF GATE

> **Authority**: Phase 1 Upstream Proof Dossier  
> **Track**: P01-D Proof Gate (Candidate Architecture V2)  
> **Date**: 2026-09-20  
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE` (Not Frozen)  
> **P01-D Transport Gate**: `NOT_CLEARED` (`TRANSPORT_PROOF_INCOMPLETE`)  
> **P01-D Phase Verdict**: `GAP_REQUIRES_ADR`  
> **Spike Location**: `D:\TU_CODE\_ai_supervisor_p01d_plugin_spike\` (Isolated outside project repo)  

---

## 1. Objective & Gate Governance

This dossier records the empirical evaluation and audit reconciliation of the proposed Architecture V2 transport candidate:
```text
ChatGPT Plus Web
  │
  ▼
Published "AI Engineering Supervisor" Plugin (OpenAI Plugin Directory)
  │  HTTPS / JSON-RPC (MCP)
  ▼
Remote MCP Gateway (Public Cloud)
  │
  ▲  Outbound Authenticated TLS/WSS (Initiated by Local Node)
  │
Local Supervisor Node (User Windows Workstation)
  │
  ▼
Untrivial Agent Orchestrator (AO Daemon) ──[ConPTY]──> Antigravity CLI (Agy)
```

### External Supervisor Audit Correction
The initial declaration of `WAITING_EXTERNAL_APPROVAL` was premature:
1. No OpenAI submission has occurred.
2. No public production MCP endpoint has been deployed or verified.
3. No submission or review tracking ID exists.

Per Section 1 of P01-D2 directives:
- **P01-D Transport Gate**: `NOT_CLEARED`
- **P01-D Phase Verdict**: `GAP_REQUIRES_ADR`
- **Architecture V2 Status**: `CANDIDATE` (Strictly prohibited from freezing).
- **Tracks P01-A, P01-B, P01-C**: Strictly `HELD` until the transport gate is proven end-to-end.

---

## 2. Reclassified Test Execution Matrix (16 Tests)

| Test ID | Category | Description | Required | Result | Audit Qualification & Literal Evidence |
|---|---|---|---|---|---|
| **D0-01** | Account | Plus published app availability | YES | **PASS** | Official OpenAI Help documentation confirms Plugin Directory is accessible to personal ChatGPT Plus accounts as of July 9, 2026. |
| **D0-02** | Account | Existing published write action works | YES | **DOCUMENTED_SUPPORTED / NOT_EMPIRICALLY_INVOKED_ON_TARGET_ACCOUNT** | Official OpenAI documentation confirms write actions on action-capable published apps; not yet empirically invoked on the specific target account. |
| **D1-01** | Developer | Developer submission eligibility | YES | **NOT_PROVEN_ON_TARGET_DEVELOPER_ACCOUNT** | Developer eligibility requires empirical inspection of the actual OpenAI Platform organization, `api.apps.write`, and identity verification status. |
| **D1-02** | Developer | Production remote MCP accepted by platform | YES | **NOT_PROVEN** | No publicly accessible production MCP endpoint has been deployed, configured, or scanned by OpenAI. |
| **D2-01** | Gateway | MCP read probe (`transport_probe_read`) | YES | **PASS_LOCAL_SPIKE** | Tested on local disposable spike (`127.0.0.1:54321`). Gateway received request, relayed to local node, and returned `{ probe: "P01-D", node_id: "win-workstation-proof-node", node_status: "ONLINE" }`. |
| **D2-02** | Gateway | MCP write probe (`transport_probe_write`) | YES | **PASS_LOCAL_SPIKE** | Tested on local disposable spike. Gateway forwarded write payload; local node mutated local disk state (`probe_counter: 1`, `last_probe_value: "VERIFIED_END_TO_END_WRITE_VALUE"`). |
| **D3-01** | Local Node | Remote → local routing via correlation ID | YES | **LOCAL_GATEWAY → LOCAL_NODE RELAY: PASS / INTERNET REMOTE GATEWAY → LOCAL NODE: NOT_PROVEN** | Correlation-based routing proven on local WebSocket harness. Real Internet traversal from a cloud host is not yet proven. |
| **D3-02** | Local Node | No inbound local exposure (0 open ports) | YES | **LOCAL_NODE_INBOUND_PORT_REQUIREMENT: PASS — ZERO LISTENING PORTS** | Local node connects strictly outward over WebSocket; opens 0 listening ports on host. (Internet NAT traversal behavior not yet tested). |
| **D4-01** | Integration | OUR plugin installed on target Plus | **HARD GATE** | **NOT_PROVEN** | Requires public gateway deployment, OpenAI Platform verification, review, and publication to the public Plugin Directory before personal Plus accounts can install it. |
| **D4-02** | Integration | OUR read tool invoked by target Plus | **HARD GATE** | **NOT_PROVEN** | Awaiting D4-01 publication. |
| **D4-03** | Integration | OUR write tool invoked by target Plus | **HARD GATE** | **NOT_PROVEN** | Awaiting D4-01 publication. |
| **D4-04** | Integration | Write changes local disposable state via ChatGPT | **HARD GATE** | **NOT_PROVEN** | Local spike proved the internal loopback relay segment only. End-to-end invocation originating from live ChatGPT Plus Web is not yet proven. |
| **D5-01** | Reliability | Node offline behavior | YES | **PASS_LOCAL_SPIKE** | Gateway immediately returned JSON-RPC error `-32001 (LOCAL_NODE_OFFLINE)` when the local node disconnected. |
| **D5-02** | Reliability | Reconnect recovery | YES | **PASS_LOCAL_SPIKE** | When the local node reconnected, Gateway automatically restored routing without server restart. |
| **D5-03** | Security | Authorization isolation | YES | **TOKEN AUTHENTICATION: PASS_LOCAL_SPIKE / TENANT, DEVICE & PAIR ISOLATION: NOT_PROVEN** | Local spike validated Bearer token check (rejected bad token with 401). Real multi-tenant ownership isolation across devices/pairs is not yet proven. |
| **D5-04** | Safety | Replay / Idempotency viability | YES | **PASS_LOCAL_SPIKE** | Duplicate request with identical correlation ID returned cached response without double execution (counter remained 1). |
| **D5-05** | Security | Secret leakage check | YES | **PASS_LOCAL_SPIKE** | Automated scan confirmed 0 secret tokens, passwords, or cookies leaked in JSON-RPC responses. |
| **D5-06** | Performance | Payload boundary & Latency test | YES | **LOCAL LOOPBACK: 1–5 ms / PUBLIC INTERNET: NOT_PROVEN / CHATGPT END-TO-END: NOT_PROVEN** | Payloads of 1KB, 10KB, and 50KB executed under 5ms on loopback. Real network and ChatGPT round-trip latencies remain unproven. |
| **Continuity** | Session | Message and chat session continuity | YES | **NOT_PROVEN** | Architecture assumes Gateway-level correlation, but real ChatGPT chat turn and cross-chat session continuity has not been tested with live plugins. |

---

## 3. Preserved Empirical Spike Evidence

The disposable proof spike in `D:\TU_CODE\_ai_supervisor_p01d_plugin_spike\` is preserved. Its valid empirical results are:

```text
LOCAL RELAY MECHANISM FEASIBILITY:
- LOCAL_GATEWAY_HARNESS:                  PASS
- LOCAL_NODE_OUTBOUND_CONNECTION:         PASS
- LOCAL_READ_PROBE:                       PASS
- LOCAL_WRITE_PROBE:                      PASS
- LOCAL_DISK_STATE_MUTATION:              PASS
- LOCAL_OFFLINE_DETECTION:                PASS
- LOCAL_RECONNECT:                        PASS
- LOCAL_DUPLICATE_REQUEST_DEDUPLICATION:  PASS
```

These results establish that **an outbound-only WebSocket client on Windows can safely receive requests from a gateway and mutate local state with idempotency**. They do **not** prove ChatGPT Plus public plugin viability.

---

## 4. Hard Gate Dependencies Remaining

To advance from `NOT_CLEARED` to `PASS`, the following sequential gates must be cleared:

1. **Gate 1: Developer Account Inspection (`platform.openai.com`)**:
   - Verify developer organization, role, `api.apps.write`, and identity verification.
   - Determine whether remote MCP plugins with write actions can be submitted without publisher having Business/Enterprise/Edu full-MCP Developer Mode.
2. **Gate 2: Public Production Endpoint**:
   - Deploy minimal Remote MCP Gateway to a public, stable HTTPS domain (`https://<domain>/mcp`).
   - Execute OpenAI Platform `Scan tools` workflow.
3. **Gate 3: Plugin Submission**:
   - Submit real, legitimate AI Engineering Supervisor minimal vertical slice with test cases and reviewer credentials.
   - Record submission ID and enter `WAITING_EXTERNAL_APPROVAL`.
4. **Gate 4: Live Target Account Proof**:
   - Install published plugin on personal ChatGPT Plus account.
   - Execute live read and live write from ChatGPT Web.
   - Confirm local Windows disk state mutation from ChatGPT prompt.

---

## 5. Architectural State & Final Verdict

```text
================================================================================
P01-D TRANSPORT GATE:
NOT_CLEARED

P01-D PHASE VERDICT:
GAP_REQUIRES_ADR

ARCHITECTURE V2:
CANDIDATE
================================================================================
```

Architecture V2 is **NOT FROZEN**.  
Tracks P01-A, P01-B, and P01-C remain **HELD**.
