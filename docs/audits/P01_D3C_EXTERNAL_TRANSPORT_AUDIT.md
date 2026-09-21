# P01-D3C EXTERNAL TRANSPORT AUDIT

> **Authority**: External Supervisor Independent Audit Record
> **Audit Date**: 2026-09-21
> **Target Track**: Phase 1 Track P01-D3C (Target Personal ChatGPT Plus Private MCP Proof)
> **Verified Remote Baseline**: `ec353001c7ef237dbe73417d2c01222942e5d465`
> **Historical Frozen Baseline**: `phase0-architecture-v1` (`6f72eaca30be3fc3ac00f25829dd4283ed98c3f5`)
> **External Audit Verdict**: `APPROVED`
> **P01-D3C Verdict**: `PASS`
> **Transport Scope**: `PROVEN_ON_TARGET_ACCOUNT`
> **P01-D Overall Status**: `TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED`
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN`
> **P01-A Track Status**: `READY` (Released from Hold)
> **P01-B / P01-C Track Status**: `HELD`

---

## 1. Audit Scope

The External Supervisor has independently and authoritatively audited the complete Phase 1 Track P01-D3C transport proof.

This verdict is authoritative for the current architectural and operational transition.

The accepted proof scope is strictly bounded to:

```text
TARGET PERSONAL CHATGPT PLUS ACCOUNT
  → Developer Mode App ("AI Supervisor P01-D3C Proof")
  → OpenAI Secure MCP Tunnel ("ai-supervisor-p01d" / tunnel_6ab0ae480cec81919b3db157c622eb53)
  → Official tunnel-client daemon (v0.0.14)
  → Private loopback Windows MCP server (127.0.0.1:3182/mcp)
  → Bounded server-side state (data/state.json)
```

The empirical proof evaluated all twelve sub-gates (D3C-01 through D3C-12) and nine milestone checkpoints (Checkpoints A through I) against physical logs, system state, process PIDs, and temporal transcripts.

---

## 2. Evidence Baseline

The evidence base supporting this audit consists of the following verified artifacts:

1. **Remote Baseline Commit**: `ec353001c7ef237dbe73417d2c01222942e5d465` on branch `main`.
2. **Historical Frozen Baseline**: Tag `phase0-architecture-v1` at commit `6f72eaca30be3fc3ac00f25829dd4283ed98c3f5`.
3. **Primary Audit Dossier**: [`docs/audits/P01_D3C_TARGET_PLUS_PRIVATE_MCP_PROOF.md`](P01_D3C_TARGET_PLUS_PRIVATE_MCP_PROOF.md).
4. **Supporting Upstream Proof Dossiers**:
   - [`docs/audits/P01_D3A_OPENAI_MCP_RUNTIME_PROOF.md`](P01_D3A_OPENAI_MCP_RUNTIME_PROOF.md) (Local MCP protocol and Responses API proof)
   - [`docs/audits/P01_D3B_PUBLIC_PLUGIN_PIPELINE_PROOF.md`](P01_D3B_PUBLIC_PLUGIN_PIPELINE_PROOF.md) (Public distribution and submission pipeline evaluation)
5. **Physical Proof Sandbox**: `D:\TU_CODE\_ai_supervisor_p01d_openai_mcp_proof\`.
6. **Active Processes Observed During Verification**:
   - Local MCP Server: Node.js PID 26608 (initial) → stopped in D3C-11 → restarted as PID 36504 (recovered).
   - Tunnel Client Daemon: `tunnel-client.exe` PID 43288 (initial) → stopped in D3C-09 → restarted as PID 38624 (reconnected).
7. **Persisted Disk State**: `data/state.json` (`current_value = "D3C_PLUS_WRITE_1789974395862"`, `mutation_count = 1`).

---

## 3. D3C-01 through D3C-12 Verdict Matrix

| Sub-Gate / Checkpoint | Target Capability | Verification Type | Verdict | Notes / Empirical Evidence |
|---|---|---|---|---|
| **D3C-01** | App Creation / Connection | Cloud UI / Tunnel Handshake | **`PASS`** | Developer app registered; tunnel recognized; tools discovered. |
| **D3C-02** | Tool Discovery | ChatGPT Web Interface | **`PASS`** | Exact tool schema registered (`supervisor_probe_read`, `supervisor_probe_write`). |
| **D3C-03** | Real ChatGPT Read | UI + Server State | **`PASS`** | Initial probe read executed; returned default value; mutation count remained 0; zero read side effects. |
| **D3C-04** | Write Classification & Runtime Approval Behavior | UI Human Interaction | **`PASS`** | Interactive confirmation modal presented with options `[Allow once]`, `[Always allow]`, `[Deny]`. |
| **D3C-05** | Approved Write | Server Log + Disk State | **`PASS`** | User confirmed `[Allow once]`; mutated state persisted to `D3C_PLUS_WRITE_1789974395862` (`mutation_count = 1`). |
| **D3C-06** | Real ChatGPT Read-Back Persistence | UI + Server State | **`PASS`** | Read-back returned persisted state `D3C_PLUS_WRITE_1789974395862`; mutation count remained 1; zero side effects. |
| **D3C-07** | Denied Write | UI + Disk State | **`PASS`** | User clicked `[Deny]`; zero frames forwarded to local MCP; mutation count remained 1; zero side effects. |
| **D3C-08** | Replay / Idempotency | Server Log + State | **`PASS`** | Replay of write correlation ID returned `DUPLICATE_REPLAY`; mutation count unchanged at 1. |
| **D3C-09** | Tunnel Reconnect | Process Kill + Restart | **`PASS`** | Tunnel-client terminated and restarted with identical tunnel ID; ChatGPT continued invocation without app recreation. |
| **D3C-10** | Chat Continuity / New Conversation | Multi-Session Verification | **`PASS`** | Tool invoked seamlessly after unrelated turns in same conversation, and in a brand-new ChatGPT conversation. |
| **D3C-11** | Local MCP Offline / Recovery | Process Kill + Restart | **`PASS`** | Local server terminated: ChatGPT returned truthful error ("The tool failed internally"); local server restarted: instant recovery without tunnel restart. |
| **D3C-12** | No Public Inbound Exposure | Loopback Audit | **`PASS`** | Local MCP bound strictly to 127.0.0.1; zero inbound public IP or open firewall port required. |
| **Checkpoint A** | App Creation & Tool Discovery (D3C-01 + D3C-02) | End-to-End Handshake | **`PASS`** | D3C-01 and D3C-02 verified live. |
| **Checkpoint B** | Real ChatGPT Read (D3C-03) | Read-Only Round-Trip | **`PASS`** | D3C-03 verified live. |
| **Checkpoint C** | Write Classification & Approved Write (D3C-04 + D3C-05) | Approval & Mutation | **`PASS`** | D3C-04 and D3C-05 verified live. |
| **Checkpoint D** | Real ChatGPT Read-Back Persistence (D3C-06) | State Persistence Verification | **`PASS`** | D3C-06 verified live. |
| **Checkpoint E** | Denied Write Safety (D3C-07) | Ingress Interception | **`PASS`** | D3C-07 verified live. |
| **Checkpoint F** | Replay / Idempotency (D3C-08) | Duplicate Detection | **`PASS`** | D3C-08 verified live. |
| **Checkpoint G** | Tunnel Reconnect (D3C-09) | Tunnel Daemon Cycling | **`PASS`** | D3C-09 verified live. |
| **Checkpoint H** | Chat Continuity / New Conversation (D3C-10) | Conversational Scope | **`PASS`** | D3C-10 verified live. |
| **Checkpoint I** | Local MCP Offline / Recovery (D3C-11) | Failure Mode Verification | **`PASS`** | D3C-11 verified live. |

---

## 4. Target Account Scope

The empirical results of P01-D3C are strictly scoped and qualified:

```text
PLUS_PRIVATE_MCP_TRANSPORT = PROVEN_ON_TARGET_ACCOUNT
```

- **Scope Boundary**: This proof definitively validates the private MCP transport on the operator's tested Personal ChatGPT Plus account with Developer Mode enabled.
- **Negative Directive**: Do NOT generalize these findings to ALL ChatGPT Plus accounts, Team workspaces, or Enterprise organizations. Any assumption of universal availability is rejected. Operating on alternative accounts requires empirical re-verification of Developer Mode and tunnel availability.

---

## 5. Read Transport Finding

Phase P01-D3C empirically proved:

1. Personal ChatGPT Plus Web in Developer Mode connects to the official OpenAI Secure MCP Tunnel and discovers exposed tools without schema corruption.
2. Outbound read invocations (`supervisor_probe_read`) query local Windows MCP state over private loopback (`127.0.0.1:3182/mcp`).
3. Millisecond-aligned log correlation across ChatGPT Web, the `tunnel-client` proxy logs, and the local MCP HTTP access logs confirms end-to-end transport fidelity with zero read mutations.

---

## 6. Write / Approval Finding

1. Tools configured with the canonical MCP hint `destructiveHint: true` (alongside `readOnlyHint` and `openWorldHint`) trigger the native ChatGPT interactive approval modal.
2. The interface presents explicit user controls: `[Allow once]`, `[Always allow]`, and `[Deny]`.
3. Upon user approval (`[Allow once]`), the command executes on the local MCP server, mutates server-side state (`D3C_PLUS_WRITE_1789974395862`), increments `mutation_count` from 0 to 1, and records execution timestamps.

---

## 7. Denial Safety Finding

1. When the user selects `[Deny]`, ChatGPT intercepts the invocation at the client layer.
2. Zero HTTP frames or JSON-RPC payloads are forwarded through the tunnel to the local MCP server.
3. Server-side state remains completely unmutated (`mutation_count` remained strictly 1; denial correlation token `d3c-deny-1789974938621` was never applied).
4. ChatGPT provides clean, truthful notification to the user without hung requests or retries (`D3C_DENIAL_SAFETY = EMPIRICALLY_PROVEN_ON_TARGET_PLUS`).

---

## 8. Replay / Exactly-Once Finding

1. Idempotency is enforced by the local MCP server tracking correlation identifiers (`applied_correlations`).
2. Replay of an identical correlation ID (`d3c-write-1789974395862`) was intercepted and rejected with status `DUPLICATE_REPLAY`.
3. Server mutation count was NOT incremented (`mutation_count` remained strictly 1), and state value remained unchanged.
4. Exactly-once execution semantics are confirmed over the transport channel (`D3C_EXACTLY_ONCE = EMPIRICALLY_PROVEN_ON_TARGET_PLUS`).

---

## 9. Reconnect Finding

1. Terminating the `tunnel-client` daemon (killing process PID 43288) while keeping the local MCP server active, followed by starting a new daemon instance (PID 38624) with the same `CONTROL_PLANE_TUNNEL_ID`, cleanly re-established outbound connectivity.
2. The existing Developer Mode app in ChatGPT retained its binding without requiring reconfiguration, deletion, or re-creation.
3. Subsequent tool calls succeeded immediately upon tunnel reconnect (`D3C_TUNNEL_RECONNECT = EMPIRICALLY_PROVEN_ON_TARGET_PLUS`).

---

## 10. Conversation Continuity Finding

1. **Same-Conversation Continuity**: Following arbitrary, unrelated conversational turns ("Explain in one sentence what idempotency means"), the tool remained fully functional and invokable with distinct command IDs.
2. **New-Conversation Access**: Opening a brand-new, independent ChatGPT conversation session allowed immediate tool invocation without re-adding the app or configuring new tokens.
3. The same existing Developer Mode app remained available and callable across separate ChatGPT conversations on the tested target account (`D3C_CHAT_CONTINUITY = EMPIRICALLY_PROVEN_ON_TARGET_PLUS`).

---

## 11. Offline / Recovery Finding

1. **Truthful Offline Failure**: When the local MCP server was offline (PID 26608 stopped, port 3182 closed), ChatGPT surfaced a truthful error: `"The tool failed internally."` Zero fabricated or stale cached state was returned.
2. **Tunnel Diagnostic Logging**: Tunnel proxy logs recorded actionable root causes (HTTP 502, `failure_source = connect`, `transport_error_kind = dial`).
3. **Seamless Live Recovery**: Restarting the local MCP server (PID 36504) immediately restored service. Subsequent ChatGPT tool invocations succeeded without restarting the tunnel daemon or modifying the app.

---

## 12. No-Public-Inbound Finding

D3C-12 is strictly scoped to the proven application and transport boundary:

```text
Local MCP listener = 127.0.0.1:3182 loopback only
Public MCP endpoint = NONE
Direct public inbound listener = NONE
Secure MCP Tunnel = outbound transport path only (127.0.0.1:8080)
```

- **Architectural Claim**:

  ```text
  NO_PUBLIC_INBOUND_MCP_ENDPOINT_REQUIRED_FOR_THIS_V1_PATH = PROVEN
  ```

- **Scope Qualification**: This finding certifies that the Supervisor architecture requires zero open inbound firewall ports or public IP addresses. It does NOT claim that the user's router or broader network perimeter was exhaustively audited.

---

## 13. Health Semantics Finding

During Checkpoint I (D3C-11), an essential operational reality was empirically discovered:

```text
LOCAL MCP = OFFLINE
tunnel-client /healthz = live
tunnel-client /readyz = ready
```

**Empirical Architectural Finding**:

```text
TUNNEL_READINESS_DOES_NOT_PROVE_LOCAL_BACKEND_READINESS = EMPIRICALLY_PROVEN
```

- The `tunnel-client` `/readyz` endpoint indicates only that the tunnel daemon has established an outbound connection to the OpenAI control plane; it does NOT assert loopback backend readiness.
- **Production Architectural Constraint**: Production Supervisor observability MUST implement composed health semantics:

  ```text
  SUPERVISOR_SYSTEM_READY = (TRANSPORT_READY AND SUPERVISOR_BACKEND_READY)
  ```

- No speculative implementation code is permitted at this stage; this finding is recorded as an immutable external audit constraint.

---

## 14. Authority Boundary Finding

ChatGPT UI permission/approval is NOT the Supervisor authority boundary:

```text
CHATGPT_APPROVAL_UI = USER_EXPERIENCE_SAFEGUARD_NOT_SERVER_AUTHORITY
```

- D3C empirically showed that write/destructive tools present approval UI, and users may choose `[Allow once]` or configure broader plugin permissions.
- The server-side Supervisor MUST independently enforce all authority constraints:

  - Task Contract validation and immutability;
  - Strict enforcement of `allowed_scope` and `forbidden_scope`;
  - State transition authority and lifecycle invariants;
  - Idempotency and mutation constraints;
  - Worker authority and independent Git evidence collection.

- Never rely on ChatGPT approval UI alone for authorization.

---

## 15. Conversation Authority Finding

ChatGPT conversation identity MUST NOT be treated as durable task authority:

```text
CHAT_CONVERSATION = NOT_DURABLE_AUTHORITY
PAIR_BINDING = SERVER_SIDE_RESPONSIBILITY
```

- D3C-10 proved the same app functions across unrelated turns in the same conversation and in completely new conversations.
- System state, pair bindings, active tasks, and review decisions reside exclusively server-side in the Supervisor Control Plane and Git repository.
- The architecture maintains zero dependency on undocumented or volatile ChatGPT conversation IDs.

---

## 16. Failure Observability Finding

When local MCP was offline, ChatGPT surfaced only:

```text
"The tool failed internally."
```

while tunnel logs contained actionable operational diagnostics (`status_code = 502`, `failure_source = connect`, `transport_error_kind = dial`).

- **Architectural Consequence**:

  ```text
  USER_ERROR_SURFACE = NOT_SUFFICIENT_FOR_OPERATOR_DIAGNOSIS
  INTERNAL_CORRELATION_LOGGING = REQUIRED_ARCHITECTURAL_CONSTRAINT
  ```

- Production architecture requires internal correlation logging linking client invocations, tunnel command IDs, and Supervisor backend logs. User-facing wording alone is insufficient for operator diagnosis.

---

## 17. Public Plugin Path Disposition

Phase P01-D3B evaluated the public plugin submission pipeline (OAuth, Cloudflare Workers, domain verification, and Plugin Submission Portal).

```text
PUBLIC_PLUGIN_PATH = OPTIONAL_FUTURE_DISTRIBUTION
P01-D3B = PRESERVED_FALLBACK_RESEARCH
```

- D3B is no longer a blocker for single-user V1 control plane operation on the target account.
- D3B research is preserved intact for future multi-user distribution, OpenAI Plugin Directory publication, public marketplace availability, or public HTTPS MCP deployment.

---

## 18. Residual Unknowns

To prevent overclaiming, this audit explicitly records that Phase P01-D3C does **NOT** prove:

1. Untrivial Agent Orchestrator (AO) runtime daemon behavior, loopback REST API, or ConPTY terminal stability on Windows (deferred to Track P01-A).
2. Direct Antigravity CLI (Agy) runtime behavior, non-interactive flags, or JSON schema output generation (deferred to Track P01-B).
3. AO ↔ Agy adapter integration correctness, interactive prompt harness behavior, or session completion detection (deferred to Track P01-C).
4. `WorkerReport` structured JSON schema normalization and enforcement.
5. Task Contract execution and worktree confinement under real workers.
6. Repository read/search tool implementation.
7. Supervisor independent Git evidence collection (diffs, process exit codes, commit SHAs).
8. Server-side Pair Manager implementation.
9. Durable database schema or state persistence implementation.
10. Production authentication lifecycle, token refresh, or secret rotation.
11. Multi-user ChatGPT Plugin Directory publication.
12. Behavior on arbitrary or unconfigured ChatGPT Plus accounts.

---

## 19. Architecture Decision

```text
ARCHITECTURE_V2 = CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN
```

- **Status**: The target private MCP transport path (ChatGPT Plus Web → Secure MCP Tunnel → Loopback MCP) is accepted.
- **Constraint**: Architecture V2 is NOT FROZEN. Remaining execution-plane and upstream runtime proofs (P01-A, P01-B, P01-C) must be empirically completed before an Architecture V2 freeze can be considered.
- **Prohibition**: Do not mark `ARCHITECTURE_FROZEN`. Do not create a freeze tag.

---

## 20. Track Release Decision

With this audit approval:

1. **Track P01-D**: Formally marked **`TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED`**.
2. **Track P01-A (AO Runtime Proof)**: Formally **`RELEASED_FROM_HOLD`** → **`READY`**.
3. **Track P01-B (Agy CLI Proof)**: Remains strictly **`HELD`**.
4. **Track P01-C (AO ↔ Agy Adapter Proof)**: Remains strictly **`HELD`**.
5. **Phase 1 Active Gate**:

   ```text
   ACTIVE GATE: READY_FOR_EXTERNAL_P01_A_EXECUTION_PROMPT
   ```

---

# 21. External Supervisor Final Verdict

```text
================================================================================
D3C_EXTERNAL_TRANSPORT_AUDIT = APPROVED
P01-D3C = PASS
PLUS_PRIVATE_MCP_TRANSPORT = PROVEN_ON_TARGET_ACCOUNT
P01-D = TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED
ARCHITECTURE_V2 = CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN

TRACK STATUSES:
P01-A = READY
P01-B = HELD
P01-C = HELD

ACTIVE GATE: READY_FOR_EXTERNAL_P01_A_EXECUTION_PROMPT
================================================================================
```
