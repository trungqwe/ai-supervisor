# PHASE SPECIFICATION: P01 — UPSTREAM PROOF & TRANSPORT FEASIBILITY

> **Phase Status**: COMPLETE
> **Exit Verdict**: GAP_REQUIRES_ADR_RESOLVED
> **Resolved ADR**: ADR-011 (`docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`)
> **Final Audit Record**: `docs/audits/P01_FINAL_EXTERNAL_AUDIT.md`

---

## 1. Objective
Empirically test and verify Untrivial Agent Orchestrator (AO `v0.13.0`), official Antigravity CLI (`1.2.7`), their integration interface, and ChatGPT Web transport feasibility on the local Windows environment prior to Supervisor application implementation.

## 2. Why Now
Do not build architectural adapters against unverified assumptions. All upstream capabilities, integration gaps, and transport channels must be empirically verified or explicitly classified before Phase P02.

---

## 3. The Four Proof Tracks

### Track P01-A — AO Runtime Proof
Verify actual pinned AO `v0.13.0` on Windows:
- Daemon startup and loopback binding.
- Health and readiness endpoints (`GET /healthz`, `GET /readyz`).
- Project registration (`POST /api/v1/projects`, `GET /api/v1/projects/{id}`).
- Session spawn and isolated worktree creation (`POST /api/v1/sessions`).
- Task instruction delivery (`POST /api/v1/sessions/{id}/send`).
- Process termination (`POST /api/v1/sessions/{id}/kill`).
- Session restore (`POST /api/v1/sessions/{id}/restore`).
- Workspace filesystem event streaming (`GET /api/v1/sessions/{id}/workspace/events`).

### Track P01-B — Direct Antigravity CLI Capability Proof
Verify pinned Antigravity CLI `1.2.7` on Windows:
- Verification of binary version (`agy --version`) and help flags (`agy --help`).
- Non-interactive prompt execution (`agy -p "<prompt>"`). Note: `-p`, `--print`, and `--prompt` are documented aliases; `agy -p "<prompt>"` is canonical.
- Output format negotiation (`--output-format text|json|stream-json`).
- Input format negotiation (`--input-format text|stream-json`). Note: `--input-format stream-json` receives prompts via stdin stream and is not combined with `-p`.
- JSON schema enforcement on output (`agy -p "<prompt>" --json-schema <schema>`).
- Auto-approve permissions (`--dangerously-skip-permissions`).
- Multi-directory workspace binding (`--add-dir <path>`).
- Conversation continuation (`--conversation <id>` and `--continue`).

### Track P01-C — AO ↔ Agy Adapter Integration Proof
Determine exactly how pinned AO invokes Agy and evaluate the integration gap:
- Empirical tracing of AO's invocation argv (verifying use of `--prompt-interactive`, `--add-dir`, `--dangerously-skip-permissions`, `--conversation`).
- Assess worker completion detection in AO session lifecycle.
- **Critical Evaluation Question**: "Can the existing AO Agy adapter satisfy our WorkerReport requirements without custom integration code?"
- Deliverable: Documented finding with explicit verdict (`YES`, `NO`, or `PARTIAL`) and concrete evidence.

### Track P01-D — ChatGPT Plus Transport Feasibility Proof
Determine the viable transport path for ChatGPT Web to invoke Supervisor tools:
- Analyze supported surfaces for target user account (ChatGPT Plus Web / Local MCP relay / Custom Action).
- Determine whether tools can be invoked without desktop browser automation and without copy-pasting.
- Record exact authentication, local loopback requirements, and approval prompts.
- **Gate Rule**: No Supervisor application code in Phase P02 may begin until Track P01-D proves a feasible transport path.

---

## 4. Exit Gate & Verdict Model

The exit gate of Phase P01 is NOT a simplistic "every aspirational contract passes with exit code 0". It strictly uses the following tri-state verdict model:

```text
P01 EXIT VERDICT:
- PASS: All baseline contracts proven live.
- GAP_REQUIRES_ADR: Upstream behavior differs from initial assumptions but is accommodated via an approved ADR (e.g., adapter normalization layer).
- BLOCKER: Core product loop is infeasible on host environment.
```
