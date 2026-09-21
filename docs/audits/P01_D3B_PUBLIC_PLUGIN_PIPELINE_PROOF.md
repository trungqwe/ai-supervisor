# P01-D3B — REAL PUBLIC PLUGIN SUBMISSION PIPELINE PROOF DOSSIER

> **Authority**: Phase 1 Upstream Proof Dossier — Track P01-D3B  
> **Date**: 2026-09-21 (Corrected: 2026-09-21 — External Audit Corrections Applied)  
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE` (Strictly Candidate; Not Frozen)  
> **Identity Verification**: `VERIFIED` (Individual identity verified on OpenAI Platform)  
> **Plugin Draft Creation**: `PASS` (Real draft created in submission portal)  
> **Submission Mode**: `WITH_MCP`  
> **P01-D3B Pre-Submit Pipeline**: `PARTIAL` (External Audit Corrections Applied; Portal Preflight PASS; Reclassified FALLBACK_PUBLIC_DISTRIBUTION_PATH)  
> **Overall P01-D Transport Gate**: `PARTIALLY_PROVEN / FINAL_PLUGIN_GATE_PENDING`  
> **P01-D Phase Verdict**: `GAP_REQUIRES_ADR`  
> **P01-A / P01-B / P01-C Status**: `HELD`  
> **Sandbox Path**: `D:\TU_CODE\_ai_supervisor_p01d_public_mcp_proof\` (Strictly isolated outside main repository)

---

## 1. Executive Summary & Ingested User Empirical Evidence

On 2026-09-21, the User completed individual identity verification (Vietnamese national ID + selfie) on the OpenAI Platform. The Plugin Submission Portal was unlocked and verified empirically:
- **`IDENTITY_VERIFICATION = VERIFIED`**
- **`PLUGIN_PORTAL = ACCESSIBLE`**
- **`CREATE_PLUGIN = PASS`**
- **`SUBMISSION_TYPE = WITH_MCP`**

A real plugin draft was created in the official portal. The portal wizard tabs were observed and documented:
`Info` → `MCP` → `Skills` → `Prompts` → `Testing` → `Global` → `Submit`.

Observed fields in the real `Info` tab include:
- Directory icon (256x256 min, PNG) & ChatGPT composer icon (48x48 min, PNG)
- Name, Version, Subtitle, Description, Category
- Developer Identity & Plugin Author
- Website URL, Customer Support URL, Privacy Policy URL, Terms of Service URL
- Demo Recording URL (explicitly stating it validates plugin test cases/functionality and referencing Developer Mode)
- Commerce & Purchasing declaration

---

## 2. D3A Acceptance Boundary & Transport Stratification

- **P01-D3A Functional Transport**: **`PASS`**. Empirically proven end-to-end:
  `Responses API` → `Secure MCP Tunnel` → `Local MCP Server` → `Read/Approval/Write/Denial/Replay/Offline/Reconnect/Payloads`.
- **Architectural Boundary**: The OpenAI Secure MCP Tunnel is an **upstream developer / private testing transport**. Per official OpenAI documentation, public directory publication requires a **stable, publicly reachable HTTPS MCP endpoint**.
- **P01-D3B Purpose**: Validate the public submission pipeline, gateway candidate architecture, domain verification challenge, and pre-submit testing requirements without fabricating product claims.

---

## 3. Security Incident Remediation (`P01-D3A-SEC-001`)

- **Incident Reference**: [`P01_D3A_SECURITY_INCIDENT_001.md`](P01_D3A_SECURITY_INCIDENT_001.md).
- **Incident Summary**: A temporary proof API key was exposed in the execution transcript during local environment variable configuration.
- **Git / Repository Scan**: Independent inspection of the Git working tree, commit history, and GitHub remote confirmed **`NOT FOUND`** (zero credential strings committed or pushed).
- **Remediation Actions Executed**:
  1. `CONTROL_PLANE_API_KEY` and `OPENAI_API_KEY` were completely deleted from `User`, `Process`, and session environments.
  2. The User was instructed to revoke/delete the temporary API key on the OpenAI Platform.
  3. Permanent prohibition against inspecting credential stores (`.codex/auth.json`, Credential Manager) reinforced.
- **Status**: `P01-D3A_FUNCTIONAL = PASS`, `P01-D3A_SECURITY = REMEDIATION_PENDING` (awaits user's platform key revocation confirmation) → `PASS_WITH_REMEDIATED_SECURITY_INCIDENT`.

---

## 4. Current Official OpenAI Source Refresh

Re-read authoritative OpenAI documentation on 2026-09-21:
- `https://developers.openai.com/plugins/deploy/submission`
- `https://developers.openai.com/plugins/deploy/app-review`
- `https://developers.openai.com/plugins/build/mcp-server`
- `https://developers.openai.com/plugins/deploy/connect-chatgpt`
- `https://developers.openai.com/api/docs/guides/secure-mcp-tunnels`

**Key Ingested Specifications**:
1. **Public MCP URL Requirement**: Must be public HTTPS, valid TLS certificate, reachable by OpenAI review bots and users.
2. **Domain Verification**: Requires serving a plain-text challenge at `https://<domain>/.well-known/openai-apps-challenge`.
3. **Scan Tools**: Automated portal parser queries `tools/list` over Streamable HTTP (`/mcp`), validating tool schemas and safety annotations (`readOnlyHint`, `destructiveHint`, `openWorldHint`).
4. **Developer Mode (Official Fact)**: Per current official OpenAI documentation, full MCP access including write/modify operations via Developer Mode is documented for **Business, Enterprise, and Edu workspaces**. Developer Mode for **Personal ChatGPT Plus** is subject to phased rollout and is NOT currently confirmed as generally available. Do NOT assume published-plugin submission bypasses the demo/developer-mode requirement until the actual portal proves it empirically.

> [!CAUTION]
> **ASSUMPTION**: Statements 1-4 above are ingested from official documentation pages but have not been independently verified by executing the actual portal submission workflow. Mark as `ASSUMPTION_FROM_OFFICIAL_DOCS` until portal preflight produces empirical evidence.


---

## 5. Hosting & Relay Candidate Architecture Research

OpenAI specifies that the public MCP endpoint can act as a public proxy/gateway to a private backend, utilizing mTLS or OAuth 2.1 if required. To fulfill P01-D3B without direct inbound port-forwarding on the user workstation, three candidate hosting approaches were researched across 10 evaluation dimensions:

| Evaluation Dimension | Candidate 1: Managed Serverless / Container (Fly.io / Cloudflare Workers) | Candidate 2: Dedicated Cloud VPS (Hetzner / DigitalOcean + Caddy) | Candidate 3: Edge Reverse Tunnel (Cloudflare Named Tunnel + Custom Domain) |
|---|---|---|---|
| **Cost** | Free tier or ~$5/mo | $4 - $6/month fixed | Free tier (requires owned domain) |
| **Streaming HTTP Compatibility** | High (Supports SSE & Streamable HTTP) | Excellent (Native unbuffered HTTP/1.1 & HTTP/2) | Good (Subject to default 100s proxy timeout) |
| **Stable Hostname** | Custom domain with auto-TLS | Static public IPv4/IPv6 + custom domain | Custom domain with Cloudflare edge TLS |
| **Domain Verification (`/.well-known`)** | Native route in worker/container | Trivial (Caddy static route) | Trivial (routed or Cloudflare Page Rule) |
| **Secret Management** | Native platform secrets (Fly Secrets / CF Secrets) | Environment variables / systemd credentials | Zero secrets on public servers |
| **Logging** | Structured cloud logs (`fly logs`, CF Tail) | Systemd journald + Caddy access JSON logs | Cloudflare Zero Trust logs + local client logs |
| **Cold Start** | 0ms - 2s depending on scale-to-zero | 0ms (permanently active daemon) | 0ms (persistent outbound tunnel) |
| **Long-Lived / Streaming Behavior** | High (minutes duration supported) | Unlimited duration | Bounded by edge proxy timeouts |
| **Outbound Local-Node Relay Support** | Persistent WebSocket relay initiated by local node | Persistent WebSocket / SSH / gRPC relay | Native outbound-only tunnel daemon |
| **Operational Complexity** | Low to Medium | Medium (Linux updates, firewall, DNS) | Low (if domain already on Cloudflare) |

### Architecture Candidates (Not Selected / Not Approved):

> [!IMPORTANT]
> All three candidates remain **`CANDIDATE_ONLY`**. No transport has been approved, deployed, or made canonical.
> No ADR-004 decision has been made. No deployment shall occur until: (1) `D3B-PORTAL-EMPIRICAL-PREFLIGHT` completes with User screenshot evidence, and (2) the External Supervisor approves the ADR.

- **Candidate 3 (Cloudflare Named Tunnel + Custom Domain)**: Evaluated for P01-D3B proof feasibility. Zero inbound ports. Domain challenge trivially routed. Cold-start: 0ms. SSE support bounded by edge proxy timeouts.
- **Candidate 1 (Managed Container — Fly.io / CF Workers)**: Evaluated for P01-D3B proof feasibility. Rapid deploy. Scale-to-zero cold start risk for long-lived SSE.
- **Candidate 2 (Dedicated Cloud VPS + Caddy)**: Evaluated for production V1 shape. Clean boundary. Higher operational complexity.

**Status**: `ADR_REQUIRED` before any deployment decision. Do NOT create ADR-004 yet.

---

## 6. Public MCP Gateway Prototype in Sandbox

Staged in disposable sandbox: `D:\TU_CODE\_ai_supervisor_p01d_public_mcp_proof\gateway_server.js`.
- **Protocol**: Streamable HTTP MCP on `/mcp`.
- **Challenge Route**: `GET /.well-known/openai-apps-challenge` returns challenge token.
- **Health Route**: `GET /healthz` returns JSON status.
- **Safety Boundary**: Exposes bounded probes only (`supervisor_probe_read`, `supervisor_probe_write`). Zero arbitrary shell execution, zero arbitrary filesystem access, zero Git mutation.
- **Empirical Validation**:
  - `GET /healthz` -> HTTP 200 OK.
  - `GET /.well-known/openai-apps-challenge` -> HTTP 200 OK (`PROOF_STAGE_CHALLENGE_PLACEHOLDER`).
  - `@modelcontextprotocol/inspector` v2.7.0 CLI:
    - `tools/list`: Discovered `supervisor_probe_read` and `supervisor_probe_write`.
    - `tools/call supervisor_probe_read`: Succeeded, returned node identity.
    - `tools/call supervisor_probe_write`: Succeeded, applied mutation #1.

> [!CAUTION]
> **Relay Evidence Correction**:
> `PUBLIC_GATEWAY_TO_LOCAL_NODE_RELAY = NOT_PROVEN`
>
> The current D3B prototype stores disposable state **locally within the sandbox process** on the same machine running the gateway server. It has **NOT** empirically demonstrated an independently hosted public gateway (running on a separate remote host) relaying MCP requests to the private Windows Supervisor node over an outbound tunnel. The relay architecture in Section 7 is a **target diagram** — it represents the intended production shape, not a proven empirical result.

---

## 7. Private Local Node Boundary

```text
┌─────────────────────────┐          ┌─────────────────────────┐          ┌─────────────────────────┐
│       OPENAI /          │  HTTPS   │    PUBLIC MCP GATEWAY   │ Outbound │  LOCAL SUPERVISOR NODE  │
│      CHATGPT PLUS       │ ───────> │    (Fly.io / Cloud)     │ <─────── │     (Workstation)       │
└─────────────────────────┘          └───────────┬─────────────┘   Relay  └───────────┬─────────────┘
                                                 │                                    │
                                                 │ • TLS / Domain Verification        │ • Bounded Probe State
                                                 │ • mTLS Termination                │ • Task Contract Checks
                                                 │ • Tool Schema Advertisement        │ • Local Test Logs / Git
                                                 │ • Zero Code/Task Authority         │ • Zero Inbound Ports
```
The public gateway acts purely as a communication and schema bridge; authority over code, task contracts, and review bundles resides strictly within the local Supervisor node.

---

## 8. Domain Verification Workflow

When the OpenAI Plugin Submission Portal issues the challenge for the registered custom domain:
1. Challenge URL: `https://<registered-domain>/.well-known/openai-apps-challenge`
2. Challenge Format: Plain-text token matching the portal-provided challenge value.
3. Verification Trigger: Portal verifies the route and issues a green verified badge for the domain.
4. **Status**: **`D3B-DOMAIN-VERIFICATION = HUMAN_REQUIRED`** (awaits user attaching public domain and receiving portal token).

---

## 9. Portal Scan Tools Specification

When the portal executes **Scan tools** against `https://<registered-domain>/mcp`:
- **Expected Tools**:
  1. `supervisor_probe_read`:
     - Parameters: `correlation_id` (string, max 64, optional).
     - Annotations: `readOnlyHint: true`, `destructiveHint: false`, `openWorldHint: false`.
  2. `supervisor_probe_write`:
     - Parameters: `test_value` (string, 1-256 chars, required), `correlation_id` (string, 1-64 chars, required).
     - Annotations: `readOnlyHint: false`, `destructiveHint: true`, `openWorldHint: false`.
- **Status**: Staged and locally validated; pending live public domain attachment.

---

## 10. Truthful Submission Info Preparation

Draft values prepared for the 16 observed fields in the `Info` tab:

| Field | Truthful Draft Value | Deployment / Governance Status |
|---|---|---|
| **Directory Icon** | 256x256 min PNG (User-observed UI minimum) | Asset drafted; pending deployment |
| **Composer Icon** | 48x48 min PNG (User-observed UI minimum) | Asset drafted; pending deployment |
| **Plugin Name** | `AI Engineering Supervisor` | Compliant (Functional, non-deceptive) |
| **Version** | `0.1.0` | Initial release candidate slice |
| **Subtitle** | Supervised software engineering control plane for autonomous coding agents. | Compliant (< 100 chars, accurate) |
| **Description** | Provides an evidence-first supervisory control plane for autonomous coding tasks. ChatGPT reviews immutable Task Contracts, inspects git diffs, verifies test execution artifacts, and coordinates isolated worker pairs without direct shell exposure. | Compliant (Accurate boundary description) |
| **Category** | `Developer Tools` / `Productivity` | Compliant |
| **Developer Identity** | Verified Individual / GPT Orchestrator | `VERIFIED` |
| **Plugin Author** | Matching verified legal name | `VERIFIED` |
| **Website URL** | `https://ai-supervisor.dev` (candidate) | **`BLOCKED_PENDING_PUBLIC_POLICY_PAGES`** |
| **Customer Support URL** | `https://github.com/trungqwe/ai-supervisor/issues` | Candidate public repository issues page |
| **Privacy Policy URL** | Candidate public privacy policy page | **`BLOCKED_PENDING_PUBLIC_POLICY_PAGES`** |
| **Terms of Service URL** | Candidate public terms of service page | **`BLOCKED_PENDING_PUBLIC_POLICY_PAGES`** |
| **Demo Recording URL** | Candidate video demonstration link | **`BLOCKED_PENDING_DEMO_RECORDING`** |
| **Commerce & Purchasing** | Free / Non-commercial tool declaration | Compliant |

*Note*: No false claims are made that public policy pages are deployed. They are drafted honestly.

---

## 11. Skills Decision: MCP Only

- **Decision**: **`MCP ONLY`**.
- **Rationale**: The core value proposition of the AI Engineering Supervisor is evidence-first validation and immutable Task Contracts. These invariants must be enforced by code on the local Supervisor node. Prompt skills cannot enforce security boundaries.

---

## 12. Prompts Tab Draft

Prepared starter prompts (constrained to currently exposed proof tools only):
1. *"Inspect the operational status of the AI Engineering Supervisor proof endpoint."*
2. *"Record a supervisory probe marker 'TEST_CHECKPOINT' for correlation id 'probe-001'."*
3. *"Verify what tools are available in the AI Engineering Supervisor proof endpoint."*

> [!WARNING]
> **FUTURE_TARGET_CAPABILITY** (NOT for current submission slice):
> - "Display the active Task Contract and worker status" — requires Task Contract system implementation.
> - "Inspect Git diff for worker report" — requires Git diff tool exposure.
> - "Coordinate worker-pair assignment" — requires AO/Agy runtime integration.
> - "Review and approve/deny a worker report" — requires Approval Gate implementation.
> These must NOT appear in the current proof submission listing until implemented in a legitimate product slice.

---

## 13. Testing Tab Specification (5 Positive + 3 Negative Cases)

Prepared in exact accordance with portal review guidelines:

### Positive Test Cases (5):
1. **Case 1 (Probe Read)**:
   - Input: *"Inspect the operational status of the local supervisor node."*
   - Expected Tool: `supervisor_probe_read`
   - Expected Arguments: `{}` or `{"correlation_id": "test-001"}`
   - Expected State Effect: Zero mutation (read-only).
   - Expected Response: Returns node identity `host-windows-dev-node-01` and current mutation count.
   - Forbidden Side Effects: No file modification, no external command execution.
2. **Case 2 (Controlled Write)**:
   - Input: *"Record a supervisory audit marker 'PROPOSAL_ACCEPTED' for correlation task-001."*
   - Expected Tool: `supervisor_probe_write`
   - Expected Arguments: `{"test_value": "PROPOSAL_ACCEPTED", "correlation_id": "task-001"}`
   - Expected State Effect: Mutation counter incremented by 1; value stored in disposable state.
   - Expected Response: Returns `status: "APPLIED"`, previous value, current value.
   - Forbidden Side Effects: Zero modification outside sandbox `data/state.json`.
3. **Case 3 (Idempotent Replay)**:
   - Input: *"Re-send audit marker for correlation task-001."*
   - Expected Tool: `supervisor_probe_write`
   - Expected Arguments: `{"test_value": "PROPOSAL_ACCEPTED", "correlation_id": "task-001"}`
   - Expected State Effect: Mutation counter NOT incremented (remains unchanged).
   - Expected Response: Returns `status: "DUPLICATE_REPLAY"`.
   - Forbidden Side Effects: No duplicate task dispatch.
4. **Case 4 (State Read-Back)**:
   - Input: *"Verify the recorded audit marker for task-001."*
   - Expected Tool: `supervisor_probe_read`
   - Expected Arguments: `{"correlation_id": "verify-001"}`
   - Expected State Effect: Zero mutation.
   - Expected Response: Confirms value `"PROPOSAL_ACCEPTED"` persists in local state.
   - Forbidden Side Effects: None.
5. **Case 5 (Proof State Round-Trip)**:
   - Input: *"Confirm the last recorded audit marker for correlation probe-001."*
   - Expected Tool: `supervisor_probe_read`
   - Expected Arguments: `{"correlation_id": "verify-probe-001"}`
   - Expected State Effect: Zero mutation.
   - Expected Response: Returns current stored value and mutation timestamp from disposable local state.
   - Forbidden Side Effects: None.

> [!IMPORTANT]
> **Protocol Evidence Only (NOT a user-facing test case)**:
> `tools/list` is a protocol-level MCP initialization step, not a submission test case. It is recorded as protocol evidence from `@modelcontextprotocol/inspector` v2.7.0 CLI execution:
> `tools/list → supervisor_probe_read, supervisor_probe_write (PASS)`
> It must NOT appear as a positive user-facing submission test case.

> [!WARNING]
> **Evidence Scope Correction**:
> Case 5 expected response "Confirms active connection between public gateway and local node" was incorrect — the D3B prototype stores state locally within the same gateway process (no independently hosted relay proven). Corrected to reflect actual local-store behavior.
> `PUBLIC_GATEWAY_TO_LOCAL_NODE_RELAY = NOT_PROVEN`

### Negative Test Cases (3):
1. **Case 1 (Arbitrary Command Execution Refusal)**:
   - Input: *"Run shell command `rmdir /s /q C:\` or execute `dir`."*
   - Expected Behavior: Model or server refuses; no shell execution tool exists in schema.
   - Forbidden Side Effects: Zero shell or OS command execution.
2. **Case 2 (Arbitrary Filesystem Access Refusal)**:
   - Input: *"Read `/etc/passwd` or `C:\Windows\System32\drivers\etc\hosts`."*
   - Expected Behavior: Refused; no filesystem exploration tools exposed.
   - Forbidden Side Effects: Zero arbitrary file read.
3. **Case 3 (Oversized Payload Schema Rejection)**:
   - Input: *"Write an audit marker containing 5,000 characters."*
   - Expected Behavior: Schema validation rejection (Zod/JSON Schema bounds `test_value` to 256 characters max).
   - Forbidden Side Effects: Zero state mutation.

---

## 14. Developer Mode & Demo Recording Gate

- **Portal Observation**: The portal UI explicitly requests a `Demo Recording URL` to validate plugin functionality and references Developer Mode.

> [!NOTE]
> **SUPERSEDED (2026-09-21)**: Prior state `D3B-DEVELOPER-MODE-FOR-PLUS = ASSUMPTION_UNVERIFIED` is superseded by new empirical evidence.
>
> **NEW EMPIRICAL EVIDENCE (2026-09-21)**:
> `PLUS_DEVELOPER_MODE_ON_TARGET_ACCOUNT = EMPIRICALLY_PROVEN`
> Target Personal ChatGPT Plus account: `Settings → Security and login → Developer mode` is **VISIBLE and ENABLED**.
> UI warning observed: "Developer Mode allows adding unverified connectors which may modify or erase data permanently."
>
> **Account-specific caveat**: This is TARGET_ACCOUNT_EMPIRICAL_CAPABILITY, not universal plan-level policy.
> Official docs state: "Developer mode availability can depend on account and workspace policy."
> Do NOT generalize: "All ChatGPT Plus accounts have full MCP write access."

- **Account Surface Evaluation**:
  - **Personal ChatGPT Plus (target account)**: `DEVELOPER_MODE_VISIBLE = TRUE`, `DEVELOPER_MODE_ENABLED = TRUE` — EMPIRICALLY_PROVEN 2026-09-21.
  - **Edu K12 Member Workspace**: Availability is governed by workspace administrator policies.
- **Gate Classification**: **`D3B-DEVELOPER-MODE = EMPIRICALLY_PROVEN`** on target account.
  - Does NOT block public endpoint construction, domain verification, or tool scanning.
- **Demo Recording**: `DEMO_RECORDING_FINAL_REQUIREMENT = NOT_YET_PROVEN` — Submit tab issue panel did not list Demo Recording as a required error; final mandatory/optional status remains empirically undetermined.

---

## 15. Global Tab Requirements

- **Distribution Scope**: Worldwide or restricted by developer preference.
- **Compliance Declarations**: Declarations of compliance with OpenAI developer terms, export laws, and safety policies.

---

## 18. New Hard Gate: D3B-PORTAL-EMPIRICAL-PREFLIGHT

> [!IMPORTANT]
> **Status: `HARD_GATE_OPEN` — Awaiting User Portal Screenshots / Empirical Evidence**
>
> This gate must be completed before any further submission-pipeline decisions are made.
> The portal wizard tabs have been visually inspected (Info confirmed); the following tabs have **NOT** been empirically documented with screenshots or structured evidence:
> `MCP`, `Testing`, `Submit`.

### Required User Actions

The User must navigate the real OpenAI Plugin Submission Portal for the existing draft and provide:

1. **MCP Tab screenshot & answers**:
   - What URL entry modes does the MCP tab offer? (URL field, auto-detection, SSE-only vs. Streamable HTTP, etc.)
   - Are authentication fields (OAuth 2.1, Bearer token, mTLS) exposed at this tab?
   - What is the exact domain verification workflow? Does the tab show a challenge token to copy?
   - What does "Scan Tools" button state / progress look like before and after scanning?

2. **Testing Tab screenshot & answers**:
   - What is the exact structure of a test case entry in the portal? (input/output fields, tool selection, etc.)
   - Is a Demo Recording URL field present in the Testing tab or only in the Info tab?

3. **Submit Tab screenshot & answers**:
   - Is a Demo Recording URL strictly mandatory before the Submit button becomes active?
   - Does the Submit tab show a reviewer checklist or pre-flight validation output?
   - Is there a supported workflow for submitting without Developer Mode access on Personal Plus?

4. **Developer Mode check on Plus account**:
   - Navigate to `ChatGPT Settings → Security & login` (or equivalent) on the User's Personal ChatGPT Plus account.
   - Is a Developer Mode toggle present and enabled?
   - If absent: confirm `D3B-DEVELOPER-MODE-FOR-PLUS = BLOCKED_BY_PLATFORM`.

### Gate Verdict

```text
D3B-PORTAL-EMPIRICAL-PREFLIGHT: PASS
Status: EMPIRICALLY_COMPLETED (2026-09-21)

MCP tab: EMPIRICALLY_CAPTURED
  - MCP Server URL field: present (text input)
  - Authentication: No Auth (default observed)
  - Scan Tools button: present
  - Domain verification: Domain not verified / Challenge Base URL / URL+Token+Verify Domain

Testing tab: EMPIRICALLY_CAPTURED
  - 5 positive test cases required
  - 3 negative test cases required
  - Fields: Scenario, User prompt, Tool triggered, Expected output

Submit tab: EMPIRICALLY_CAPTURED
  - Release Notes field: required
  - Compliance attestations: reviewed; validated
  - Current required field errors: Name, MCP URL, Test case scenario, Release notes
  - Demo Recording: NOT in current error list (mandatory/optional status still UNKNOWN)

Developer Mode on Plus: EMPIRICALLY_PROVEN (target account)
```

> This gate is CLOSED. Proceed to D3C.

---
## 16. Pre-Submit Readiness Report (`P01-D3B_PRE_SUBMIT_READINESS_REPORT`)

| Sub-Gate / Requirement | Status | Evidence / Blocker |
|---|---|---|
| **Identity Verification** | **`PASS`** | Verified via national ID + selfie on OpenAI Platform |
| **Plugin Draft Creation** | **`PASS`** | Draft created in `With MCP` mode |
| **Local Gateway Prototype** | **`PASS`** | Staged in sandbox; inspector verified |
| **Public HTTPS Endpoint** | **`BLOCKED`** | Requires deploying public gateway on verified domain |
| **Domain Verification** | **`BLOCKED`** | Pending public domain & challenge token |
| **Scan Tools** | **`BLOCKED`** | Requires active public HTTPS endpoint |
| **Info Tab Assets & Legal URLs** | **`BLOCKED`** | Policy URLs (`privacy`, `terms`) not yet published |
| **Testing Tab Test Cases** | **`PASS`** | 5 positive + 3 negative cases formulated; corrected to remove `tools/list` as user-facing case and future-capability claims |
| **Developer Mode Availability** | **`BLOCKED_PENDING_EMPIRICAL_PORTAL_CHECK`** | Official docs confirm for Business/Enterprise/Edu; Plus availability unverified. Awaits `D3B-PORTAL-EMPIRICAL-PREFLIGHT` |
| **Demo Recording** | **`BLOCKED`** | Pending Developer Mode empirical verification; mandatory vs optional status unverified |
| **D3B-PORTAL-EMPIRICAL-PREFLIGHT** | **`OPEN`** | MCP tab, Testing tab, Submit tab, Developer Mode on Plus — awaiting User screenshot evidence (Section 18) |
| **Submit Tab Action** | **`STOPPED`** | Final Submit strictly prohibited without User + External approval |

---

## 17. P01-D3B Final Verdict

```text
================================================================================
P01-D3B VERDICT:
PARTIAL

CORRECTIONS APPLIED (External Audit 2026-09-21):
- Icon dimensions corrected: 256x256 min (directory), 48x48 min (composer)
- Plugin Author: [VERIFIED_LEGAL_PUBLISHER_NAME] (legal name not stored in repo)
- Future capabilities marked FUTURE_TARGET_CAPABILITY:
  Task Contract inspection, Git diff, Worker status, Worker-pair coordination, AO/Agy control
- Starter prompts corrected to current proof tools only
- tools/list removed from user-facing test cases (remains protocol evidence only)
- Test case 5 corrected: PUBLIC_GATEWAY_TO_LOCAL_NODE_RELAY = NOT_PROVEN
- Hosting candidates remain CANDIDATE_ONLY; ADR-004 not yet created
- D3B-PORTAL-EMPIRICAL-PREFLIGHT: OPEN (new hard gate added)
- Developer Mode official fact: documented for Business/Enterprise/Edu; Plus = ASSUMPTION_UNVERIFIED
- P01-D3A-SEC-001: REMEDIATION_PENDING (unchanged; awaits User platform key revocation confirmation)

REASON:
- Identity Verification: PASS (VERIFIED)
- Plugin Portal Draft: PASS (With MCP mode)
- Gateway Prototype: PASS (Local Streamable HTTP only, co-located; PUBLIC_GATEWAY_TO_LOCAL_NODE_RELAY = NOT_PROVEN)
- Public HTTPS Endpoint: BLOCKED (Pending public deployment)
- Domain Verification: BLOCKED (Pending domain challenge)
- Scan Tools: BLOCKED (Pending public endpoint)
- Legal URLs & Policy Pages: BLOCKED (Drafted honestly; pending public hosting)
- Developer Mode: BLOCKED_PENDING_EMPIRICAL_PORTAL_CHECK (Plus availability unverified)
- Demo Recording: BLOCKED (mandatory vs optional status unverified)
- D3B-PORTAL-EMPIRICAL-PREFLIGHT: OPEN (MCP/Testing/Submit tabs + Developer Mode on Plus)
- P01-D3A-SEC-001: REMEDIATION_PENDING (awaiting User platform key revocation confirmation)
- Final Submit: STOPPED

P01-D3A_FUNCTIONAL = PASS
P01-D3A_SECURITY = REMEDIATION_PENDING

OVERALL P01-D TRANSPORT GATE:
PARTIALLY_PROVEN / FINAL_PLUGIN_GATE_PENDING

P01-D PHASE VERDICT:
GAP_REQUIRES_ADR

ARCHITECTURE V2 STATUS:
CANDIDATE (NOT FROZEN)
================================================================================
```

Tracks P01-A, P01-B, and P01-C remain strictly **HELD**.