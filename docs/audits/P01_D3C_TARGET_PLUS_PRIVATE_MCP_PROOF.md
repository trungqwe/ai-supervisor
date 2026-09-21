# P01-D3C — TARGET PERSONAL CHATGPT PLUS PRIVATE MCP TRANSPORT PROOF

> **Authority**: Phase 1 Upstream Proof Dossier — Track P01-D3C
> **Date**: 2026-09-21
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE` (Strictly Candidate; Not Frozen)
> **D3B External Audit**: `APPROVED` (commit `d7ddd67b60abba1497fbf2dc550c2d0abaddd96f`)
> **D3B Portal Empirical Preflight**: `PASS` (MCP tab, Testing tab, Submit tab — empirically documented 2026-09-21)
> **Target Account**: Personal ChatGPT Plus
> **Plus Developer Mode**: `EMPIRICALLY_PROVEN` on target account (Settings → Security and login → Developer mode: VISIBLE and ENABLED)
> **P01-D3C Status**: `IN_PROGRESS`
> **P01-D3A-SEC-001**: `REMEDIATION_PENDING` (awaits User platform key revocation confirmation)
> **P01-A / P01-B / P01-C**: `HELD`
> **Sandbox**: `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\` (D3A assets reused; strictly isolated)

---

## 1. Purpose and Replan Rationale

### 1.1 D3B Disposition

Phase P01-D3B (Public Plugin / Published App path) evaluated the full public distribution pipeline.

D3B established:
- `IDENTITY_VERIFICATION = VERIFIED`
- `D3B-PORTAL-EMPIRICAL-PREFLIGHT = PASS` (all three tabs now empirically documented)
- `D3B-DEVELOPER-MODE-FOR-PLUS = EMPIRICALLY_PROVEN` on target account (supersedes prior `ASSUMPTION_UNVERIFIED`)
- `PUBLIC_GATEWAY_TO_LOCAL_NODE_RELAY = NOT_PROVEN` (prototype was co-located; no independent remote relay)
- Hosting candidates remain `CANDIDATE_ONLY`; no ADR-004 created; no deployment executed

D3B public distribution path is reclassified as: **`FALLBACK_PUBLIC_DISTRIBUTION_PATH`** pending D3C outcome.

### 1.2 D3C Goal

Determine whether the target Personal ChatGPT Plus account can use an **existing Secure MCP Tunnel** to invoke the **private local Windows MCP server** directly, without any public internet exposure.

```
TARGET PERSONAL CHATGPT PLUS WEB (browser)
        |
        v  Developer Mode App
ChatGPT Plugins -> + -> Connection = Tunnel
        |
        v
OpenAI Secure MCP Tunnel (tunnel_6ab0ae480cec81919b3db157c622eb53)
        |
        v
official tunnel-client v0.0.14 (Windows workstation)
        |
        v
Private local MCP server (127.0.0.1:3182/mcp)
        |
        v
bounded disposable local state (data/state.json)
```

If D3C PASS:
- Architecturally simpler than the published-plugin/public-gateway path for single-user V1
- Public plugin directory path becomes `OPTIONAL_FUTURE_DISTRIBUTION`
- Architecture V2 advances to `CANDIDATE_READY_FOR_EXTERNAL_TRANSPORT_AUDIT`

If D3C FAIL:
- Preserve literal error evidence
- Resume D3B public-plugin evaluation only after External Supervisor decision
- No workarounds; no browser automation; no silent architecture switch

### 1.3 D3C Is NOT a Substitute for D3A

D3A proved: Responses API -> Secure MCP Tunnel -> Windows MCP -> read/write/approval/replay/offline.
D3C must independently prove: TARGET PERSONAL CHATGPT PLUS WEB -> Tunnel -> Windows MCP.
The caller must be the actual ChatGPT Plus browser UI, not the Responses API.

---

## 2. New Empirical Evidence Ingested (Pre-D3C)

### 2.1 Target Plus Developer Mode

- **Date**: 2026-09-21
- **Account**: Personal ChatGPT Plus (target account for P01-D proof)
- **Observation**: `Settings -> Security and login -> Developer mode` is VISIBLE and ENABLED
- **UI Warning Text (observed)**: Developer Mode allows adding unverified connectors which may modify or erase data permanently.
- **Evidence Classification**: `TARGET_ACCOUNT_EMPIRICAL_CAPABILITY`
- **Verdict**: `PLUS_DEVELOPER_MODE_ON_TARGET_ACCOUNT = EMPIRICALLY_PROVEN`

> NOTE — Account-specific evidence. Do NOT generalize:
> "All ChatGPT Plus accounts have full MCP write access."
> Official docs state: "Developer mode availability can depend on account and workspace policy."
> (Source: https://developers.openai.com/plugins/deploy/connect-chatgpt, accessed 2026-09-21)
> Distinguish PLAN_LEVEL_DOCUMENTED_POLICY from TARGET_ACCOUNT_EMPIRICAL_CAPABILITY.
> The target account evidence has priority for P01-D feasibility.

### 2.2 Portal Preflight Tab Screenshots (D3B Closure)

Empirical evidence supplied for all three previously-unverified tabs:

#### MCP Tab — D3B-MCP-TAB-EMPIRICAL = PASS

Observed real portal fields:
- MCP Server URL: text input field
- Authentication dropdown: default value = No Auth
- Scan Tools button: present
- Domain verification section:
  - Domain status: Domain not verified
  - Challenge Base URL: Portal states it may use an HTTPS origin on the MCP hostname or parent hostname; paths are ignored
  - Sub-section: URL / Token / Verify Domain controls

SUPERSEDED prior ASSUMPTION: "Universal URL mode, Template URL mode, SSE-only selection"
Actual: Single MCP Server URL text field with No Auth default.

#### Testing Tab — D3B-TESTING-TAB-EMPIRICAL = PASS

Observed portal requirements:
- Exactly 5 positive test cases required
- Exactly 3 negative test cases required
- Positive case fields: Scenario, User prompt, Tool triggered, Expected output

tools/list remains: PROTOCOL_EVIDENCE_ONLY (not a user-facing test case). Previously documented corrections valid.

#### Submit Tab — D3B-SUBMIT-TAB-EMPIRICAL = PASS

Observed fields and compliance attestations:
- Release Notes field
- Policy compliance attestations (reviewed Terms/Guidelines; compliance with laws; no money/crypto trades; no ads; rights to third-party content; not designed for children under 13; adult content declaration)

Current portal validation issues observed:
- Info incomplete: Name is required
- MCP incomplete: MCP server URL is required
- Testing incomplete: Test case scenario is required
- Submit incomplete: Release notes is required

CRITICAL: The current issue panel does NOT display Demo Recording as an error.
DEMO_RECORDING_FINAL_REQUIREMENT = NOT_YET_PROVEN
The absence of Demo Recording from the issue panel does NOT prove it is optional.

#### D3B Portal Preflight Gate Closed:

```
D3B-PORTAL-EMPIRICAL-PREFLIGHT: PASS
MCP tab: EMPIRICALLY_CAPTURED
Testing tab: EMPIRICALLY_CAPTURED
Submit tab: EMPIRICALLY_CAPTURED
Developer Mode on Plus: EMPIRICALLY_PROVEN (target account, 2026-09-21)
```

---

## 3. Official Source Refresh (2026-09-21)

Sources accessed:
1. https://developers.openai.com/plugins/deploy/connect-chatgpt (accessed 2026-09-21)
2. https://developers.openai.com/api/docs/guides/secure-mcp-tunnels (accessed 2026-09-21)

Key confirmed facts:
1. Developer Mode workflow: Settings -> Security and login -> Developer mode -> ON
2. Add MCP server: ChatGPT Plugins -> + -> name/description -> Connection: Tunnel -> select tunnel or paste tunnel_id -> Create
3. Tunnel scope: "Use Secure MCP Tunnel to connect a private MCP server in developer mode without exposing the server to the public internet."
4. Public endpoint not required for developer mode: Tunnel is explicitly an alternative to public HTTPS for private/developer testing.
5. Developer mode and submission are distinct: "These testing options do not replace the public HTTPS endpoint required for plugin submission." D3C proves private access; D3B path is the distribution path.
6. Personal account: Official docs confirm personal accounts can use the personal Platform organization. Availability depends on account/workspace policy.

---

## 4. Security Status

### P01-D3A-SEC-001 — REMEDIATION_PENDING

Status: REMEDIATION_PENDING (unchanged)
Reason: Awaits explicit User confirmation that the temporary Platform API key has been revoked on the OpenAI Platform Dashboard.
Local env: Verified clean (OPENAI_API_KEY and CONTROL_PLANE_API_KEY both False in User and Process scopes).
Repository: Zero sk-proj- strings in commit history or remote repository.
Action required: User must navigate to platform.openai.com -> API Keys and delete/revoke the temporary key.
Closure: When User explicitly confirms revocation, update to REMEDIATED and P01-D3A_SECURITY = PASS_WITH_REMEDIATED_SECURITY_INCIDENT.

### Runtime Key Rule (D3C)

If tunnel-client requires a runtime key: return HUMAN_REQUIRED_CONFIGURE_NEW_TUNNEL_RUNTIME_KEY.
User must create a fresh temporary Platform key locally. Never request the plaintext key.
Only test present/absent as boolean. Do NOT store any key in the repository.

---

## 5. D3A Assets Reused for D3C

| Asset | Path | Status |
|---|---|---|
| Sandbox MCP server | D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\server.js | Reused |
| Tunnel binary | bin\tunnel-client.exe v0.0.14 (checksum verified) | Reused |
| Tunnel ID | tunnel_6ab0ae480cec81919b3db157c622eb53 (ai-supervisor-p01d) | Reused |
| Tools exposed | supervisor_probe_read, supervisor_probe_write | Reused |
| State file | data\state.json | Reset for fresh D3C proof |

D3A Responses API results are NOT substituted for D3C. D3C caller must be the actual ChatGPT Plus browser UI.

---

## 6. D3C Public Deployment Gate

PUBLIC_DEPLOYMENT = PAUSED for duration of D3C.

Prohibited until D3C concludes:
- Cloudflare Tunnel / Fly.io / VPS deployment
- Domain purchase or configuration
- Public domain verification
- Public relay implementation
- ADR-004 creation
- Plugin review submission continuation
- Final Submit click

All D3B research preserved as: FALLBACK_PUBLIC_DISTRIBUTION_PATH

---

## 7. D3C Test Sequence

### Preparation

#### Step P1 — Local MCP Server Health
Status: [TO BE RECORDED]
GET http://127.0.0.1:3182/healthz -> [TO BE RECORDED]

#### Step P2 — Tunnel Doctor
All required live checks: [TO BE RECORDED]

#### Step P3 — Tunnel Start
Tunnel: ai-supervisor-p01d (tunnel_6ab0ae480cec81919b3db157c622eb53)
Status: [TO BE RECORDED] (expected: ready / connected)

HUMAN_REQUIRED_CREATE_CHATGPT_DEV_APP — issued after Steps P1/P2/P3 pass

### D3C-01 — App Creation / Connection

Expected: Target Plus account connects developer app via Connection = Tunnel
Evidence: User screenshot / report
Actual: [TO BE RECORDED]
Verdict: [TO BE RECORDED]

### D3C-02 — Tool Discovery

Expected tools: supervisor_probe_read, supervisor_probe_write (exactly, no others)
Tool names: [TO BE RECORDED]
Descriptions: [TO BE RECORDED]
Schemas: [TO BE RECORDED]
Annotations: [TO BE RECORDED]
Verdict: [TO BE RECORDED]

### D3C-03 — Real ChatGPT Read

Caller: Target Personal ChatGPT Plus Web UI (NOT Responses API, NOT MCP Inspector)
Fresh proof token: [TO BE SET before test]
Tool called: supervisor_probe_read
Response path: ChatGPT Plus -> Tunnel -> local MCP -> ChatGPT Plus
Actual response: [TO BE RECORDED]
Local state verified: [TO BE RECORDED]
Verdict: [TO BE RECORDED]

### D3C-04 — Write Classification

Tool called: supervisor_probe_write
ChatGPT permission UI: [TO BE RECORDED - actual behavior, not inferred from annotations]
Verdict: [TO BE RECORDED]

### D3C-05 — Approved Write

Test value: PLUS_PRIVATE_MCP_WRITE_<timestamp>
Correlation ID: d3c-write-<timestamp>
Request reached local MCP: [TO BE RECORDED]
Local state mutated: [TO BE RECORDED]
Mutation count before: [TO BE RECORDED]
Mutation count after: [TO BE RECORDED]
Result returned to ChatGPT: [TO BE RECORDED]
Verdict: [TO BE RECORDED]

### D3C-06 — ChatGPT Read-Back

Tool called: supervisor_probe_read
Expected value: exact value written in D3C-05
Actual value: [TO BE RECORDED]
Mutation count: [TO BE RECORDED]
Verdict: [TO BE RECORDED]

### D3C-07 — Denied Write

Test value: D3C_DENIED_WRITE_<timestamp>
ChatGPT permission UI: [TO BE RECORDED]
Action taken: Deny (or record actual behavior if no confirmation shown)
Local state after: [TO BE RECORDED - must be unchanged]
Mutation count after: [TO BE RECORDED - must be unchanged]
Note: If no confirmation shown, record actual behavior; do not fabricate denial gate
Verdict: [TO BE RECORDED] — PASS / FAIL / NOT_APPLICABLE_WITH_REASON

### D3C-08 — Replay Protection

Correlation ID: same as D3C-05
First invocation: APPLIED (already recorded in D3C-05)
Second invocation: [TO BE RECORDED - expected: DUPLICATE_REPLAY]
Mutation count change on second call: Expected 0
Caller: ChatGPT Plus
Verdict: [TO BE RECORDED]

### D3C-09 — Tunnel Reconnect

Action: Restart tunnel-client while local MCP server remains running
Tunnel ID recreated: NO (same tunnel_6ab0ae480cec81919b3db157c622eb53)
ChatGPT app recreated: NO (same developer app)
Read after reconnect: [TO BE RECORDED]
Verdict: [TO BE RECORDED]

### D3C-10 — Chat Continuity

Same conversation: read -> write -> natural discussion -> read again -> app callable
New conversation: Same developer app invoked; same local state verified
Note: Do not depend on undocumented ChatGPT conversation IDs
Verdict: [TO BE RECORDED]

### D3C-11 — Local MCP Offline / Recovery

Action: Stop local MCP; tunnel-client remains running
ChatGPT read behavior: [TO BE RECORDED - actual failure behavior]
MCP restarted: YES
Recovery: [TO BE RECORDED - ChatGPT app not recreated]
Verdict: [TO BE RECORDED]

### D3C-12 — No Inbound Exposure

Local MCP bind: 127.0.0.1 only (loopback)
Windows inbound firewall rule: None added
Router port forwarding: None
Public MCP endpoint: None
Verdict: [TO BE RECORDED]

---

## 8. Human Checkpoint — ChatGPT Developer App Creation

```
HUMAN_REQUIRED_CREATE_CHATGPT_DEV_APP
Status: WAITING_FOR_USER_ACTION
```

Pre-conditions confirmed before issuing this checkpoint:
- Local MCP server started on 127.0.0.1:3182 — health check PASS
- Tunnel doctor: all checks PASS
- Tunnel started: ai-supervisor-p01d — ready/connected

User Instructions:

1. Open the target Personal ChatGPT Plus account in a browser.
2. Confirm Developer Mode is still enabled:
   Settings -> Security and login -> Developer mode = ON
3. Navigate to: https://chatgpt.com/plugins
4. Select the + (plus) button.
5. Enter a temporary user-visible name: AI Supervisor P01-D3C Proof
6. Enter a truthful description: Private MCP transport proof for the AI Engineering Supervisor.
7. Under Connection, select: Tunnel
8. Select ai-supervisor-p01d from the available tunnel list, OR paste:
   tunnel_6ab0ae480cec81919b3db157c622eb53
9. Select Create / Connect.
10. Review the discovered tools — expected: supervisor_probe_read and supervisor_probe_write.
11. Take a screenshot of the app connection confirmation page.
12. Report back with the connection result.

PROHIBITED:
- Do NOT automate the ChatGPT UI
- Do NOT use browser DOM automation
- Do NOT use internal ChatGPT APIs
- Do NOT substitute the Responses API for this test

---

## 9. Pass Criteria

P01-D3C verdict may be PASS ONLY if ALL of the following are true:
1. Target Personal ChatGPT Plus Developer Mode app connects via Tunnel
2. Expected tools discovered (supervisor_probe_read, supervisor_probe_write; no others)
3. Real ChatGPT read reaches local Windows MCP (not Responses API, not MCP Inspector)
4. Real ChatGPT write reaches local Windows MCP
5. Local state mutates correctly; mutation count increments exactly once
6. Result returns automatically to ChatGPT (no manual payload relay)
7. Replay protection functions correctly (duplicate blocked)
8. Tunnel reconnect succeeds without recreating ChatGPT app
9. Same app works again after MCP offline/recovery
10. No browser/UI automation used
11. No manual payload relay used
12. No public endpoint required

---

## 10. State After D3C

If PASS:
```
P01-D3C = PASS
PLUS_PRIVATE_MCP_TRANSPORT = PROVEN_ON_TARGET_ACCOUNT
P01-D = TRANSPORT_PROVEN_PENDING_EXTERNAL_ARCHITECTURE_AUDIT
PUBLIC_PLUGIN_PATH = OPTIONAL_FUTURE_DISTRIBUTION
D3B = PRESERVED_FALLBACK_RESEARCH
ARCHITECTURE_V2 = CANDIDATE_READY_FOR_EXTERNAL_TRANSPORT_AUDIT
P01-A / P01-B / P01-C = HELD
```
STOP. External Supervisor must audit D3C before P01-A is released.

If FAIL:
```
P01-D3C = FAIL or GAP_REQUIRES_ADR
```
D3B public-plugin evaluation resumes only after External Supervisor decision.
No workarounds. No browser automation. No silent architecture switch.

---

## 11. D3C Sub-Gate Summary

| Sub-Gate | Verdict | Notes |
|---|---|---|
| P1 — Local MCP Health | [TO BE RECORDED] | |
| P2 — Tunnel Doctor | [TO BE RECORDED] | |
| P3 — Tunnel Ready | [TO BE RECORDED] | |
| HUMAN_REQUIRED_CREATE_CHATGPT_DEV_APP | ISSUED | Awaiting user action |
| D3C-01 — App Connection | [TO BE RECORDED] | |
| D3C-02 — Tool Discovery | [TO BE RECORDED] | |
| D3C-03 — ChatGPT Read | [TO BE RECORDED] | |
| D3C-04 — Write Classification | [TO BE RECORDED] | |
| D3C-05 — Approved Write | [TO BE RECORDED] | |
| D3C-06 — Read-Back | [TO BE RECORDED] | |
| D3C-07 — Denied Write | [TO BE RECORDED] | |
| D3C-08 — Replay | [TO BE RECORDED] | |
| D3C-09 — Tunnel Reconnect | [TO BE RECORDED] | |
| D3C-10 — Chat Continuity | [TO BE RECORDED] | |
| D3C-11 — MCP Offline/Recovery | [TO BE RECORDED] | |
| D3C-12 — No Inbound Exposure | [TO BE RECORDED] | |

---

## 12. P01-D3C Verdict (Post-Execution)

```
================================================================================
P01-D3C VERDICT:
[TO BE RECORDED]

REASON:
[TO BE RECORDED]

P01-D STATUS:
[TO BE RECORDED]

P01-D3A_FUNCTIONAL = PASS
P01-D3A_SECURITY = REMEDIATION_PENDING (SEC-001 awaits User key revocation confirmation)

ARCHITECTURE V2 STATUS:
CANDIDATE (NOT FROZEN — pending External Transport Audit)
================================================================================
```

Tracks P01-A, P01-B, and P01-C remain strictly HELD.