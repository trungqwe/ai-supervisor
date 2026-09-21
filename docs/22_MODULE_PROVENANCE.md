# 22. MODULE PROVENANCE & ANTI-REINVENTION JUSTIFICATION

> **Authority**: Core Module Ownership Register & Anti-Reinvention Proof
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

# 1. Planned Modules Provenance

### Module: `ProjectRegistry`
- **Purpose**: Tracks registered local projects, validates local directory existence, and stores project metadata.
- **Origin**: Mieruko workspace binding pattern (`docs/sources/04_MIERUKO.md`).
- **Existing Upstream Capability Checked**: YES (AO has project registration, but lacks Supervisor-specific documentation indexing).
- **Reason This Module Exists in Our Code**: Links local workspace paths to canonical architecture docs and Pair state.
- **Why We Own This**: AO manages execution paths; Supervisor owns project documentation indexing and governance metadata.
- **Forbidden Responsibility**: Does NOT manage Git repositories or create worktrees.

### Module: `PairRegistry`
- **Purpose**: Manages the engineering lane binding one project, one ChatGPT session, and one active worker session.
- **Origin**: Mieruko workspace binding concepts (`docs/sources/04_MIERUKO.md`).
- **Existing Upstream Capability Checked**: YES (AO manages sessions, but has no concept of ChatGPT Supervisor binding).
- **Reason This Module Exists in Our Code**: Prevents multi-project interference and validates session tokens.
- **Why We Own This**: Core abstraction of the Supervisor Control Plane.
- **Forbidden Responsibility**: Does NOT manage OS processes or ConPTY allocations.

### Module: `TaskContractManager`
- **Purpose**: Validates, stores, and enforces immutable Task Contracts.
- **Origin**: OpenAI Symphony work units and Codencer task schemas (`docs/sources/06_SYMPHONY.md`, `docs/sources/07_CODENCER.md`).
- **Existing Upstream Capability Checked**: YES (AO accepts task instructions, but enforces no schema or scope immutability).
- **Reason This Module Exists in Our Code**: Enforces strict mathematical scope bounds on worker tasks.
- **Why We Own This**: Defines the authoritative work contract between Supervisor and Worker.
- **Forbidden Responsibility**: Does NOT execute tasks or edit code.

### Module: `PolicyEngine`
- **Purpose**: Compares Git diffs against contract `allowed_scope` and flags violations.
- **Origin**: Proxide containment and security model (`docs/sources/03_PROXIDE.md`).
- **Existing Upstream Capability Checked**: YES (AO has no scope filtering; workers can touch any file in the worktree).
- **Reason This Module Exists in Our Code**: Prevents worker agents from altering architecture, root configs, or other modules.
- **Why We Own This**: Upstream AO does not enforce architectural boundary policies.
- **Forbidden Responsibility**: Does NOT block OS file access directly; operates as post-execution audit gate.

### Module: `EvidenceCollector`
- **Purpose**: Orchestrates independent evidence acquisition (Git base/head SHAs, diffs, changed files, scope compliance, and trusted verification test results). It MAY invoke an internal constrained verification runner to execute approved host-owned verification profiles (`verification_requests` per ADR-013) within an isolated execution boundary.
- **Origin**: AIWorkHub candidate verification and Codencer evidence model (`docs/sources/05_AIWORKHUB.md`, `docs/sources/07_CODENCER.md`).
- **Existing Upstream Capability Checked**: YES (AO reports session exit code, but does not correlate Git diffs or build test audit packets).
- **Reason This Module Exists in Our Code**: Zero-trust verification: worker claims must be corroborated by independent evidence.
- **Why We Own This**: Bridge between Git/test ground truth and ChatGPT audit.
- **Forbidden Responsibility**: Does NOT execute arbitrary shell; does NOT perform worker implementation; does NOT expose command execution primitives to ChatGPT; does NOT accept unconstrained user shell strings. Implementation remains Phase P04.

### Module: `ReviewBundleBuilder`
- **Purpose**: Assembles Task Contract, Worker Claims, Git Diff, Test Artifacts, and Policy Findings into a unified JSON bundle.
- **Origin**: AIWorkHub / Codencer review model (`docs/sources/05_AIWORKHUB.md`, `docs/sources/07_CODENCER.md`).
- **Existing Upstream Capability Checked**: YES (No existing open-source tool builds ChatGPT-tailored review bundles).
- **Reason This Module Exists in Our Code**: Drastically reduces ChatGPT tool calls from 25+ down to 1.
- **Why We Own This**: Core cognitive compression innovation of the AI Engineering Supervisor Control Plane.
- **Forbidden Responsibility**: Does NOT approve or reject tasks; only packages data for ChatGPT.

### Module: `AuditLogger`
- **Purpose**: Records append-only, secret-sanitized chronological logs of all pair, task, and evidence events.
- **Origin**: Proxide audit sanitization (`docs/sources/03_PROXIDE.md`).
- **Existing Upstream Capability Checked**: YES (AO logs daemon events, but lacks domain-level pair and review governance trails).
- **Reason This Module Exists in Our Code**: Ensures full auditability and post-mortem diagnosis.
- **Why We Own This**: Authoritative record of supervisor-worker coordination.
- **Forbidden Responsibility**: Does NOT store plaintext secrets or environment passwords.

### Module: `AOAdapter`
- **Purpose**: Translates domain operations into Untrivial Agent Orchestrator public REST API calls (`GET /healthz`, `GET /readyz`, `POST /api/v1/projects`, `POST /api/v1/sessions`, `POST /api/v1/sessions/{id}/send`, `/kill`, `/restore`).
- **Origin**: AWS CAO provider abstraction (`docs/sources/08_AWS_CAO.md`).
- **Existing Upstream Capability Checked**: YES (AO provides the public REST API).
- **Reason This Module Exists in Our Code**: Decouples Supervisor domain logic from AO internal changes and adapts worker session telemetry.
- **Why We Own This**: Anti-corruption layer shielding the domain; handles empirical gap resolution identified in P01-C.
- **Forbidden Responsibility**: Does NOT access AO internal SQLite database or vendor AO code.

### Module: `ChatGPTToolSurface` & `TransportAdapter`
- **Purpose**: Exposes the 12 high-level domain tools over the OpenAI-supported transport proven in Phase P01 Track P01-D (FR-016).
- **Origin**: Mieruko slim tool surface (`docs/sources/04_MIERUKO.md`).
- **Existing Upstream Capability Checked**: YES (Mieruko/Proxide provide generic MCP tools, but none implement our 12-tool contract).
- **Reason This Module Exists in Our Code**: Provides the cognitive interface for ChatGPT Web.
- **Why We Own This**: Tailored cognitive interface for our supervision model.
- **Forbidden Responsibility**: Does NOT expose arbitrary shell, raw file writing, OS GUI automation, or low-level process details (such as worker PID). Does NOT perform branch merges, commits, or pushes upon task approval (V1 approval strictly records decision).

---

# 2. Phase P01-D Transport Provenance & Anti-Reinvention Clarifications

### Component: `Secure MCP Tunnel (tunnel-client)`
- **Classification**: **`UPSTREAM — OpenAI`**
- **Existing Upstream Capability Checked**: YES (`openai/tunnel-client` binary / GitHub releases).
- **Reason We Do Not Implement**: OpenAI provides the official outbound-only secure tunnel connecting loopback MCP servers to OpenAI products.
- **Why We Own This**: We do NOT own or reimplement this; we consume it strictly as an upstream utility.

### Component: `SupervisorMCPAdapter` (Local MCP Server)
- **Classification**: **`OUR ADAPTER SURFACE`**
- **Purpose**: Implements Model Context Protocol (via official `@modelcontextprotocol/sdk`) exposing domain supervisor probes (`supervisor_probe_read`, `supervisor_probe_write`) and tool schemas.
- **Existing Upstream Capability Checked**: YES (Generic MCP servers exist, but none provide Supervisor Control Plane lifecycle and contract tooling).
- **Why We Own This**: Domain-specific cognitive and audit surface for ChatGPT supervision.
- **Forbidden Responsibility**: Does NOT expose arbitrary command execution, raw disk access, or external network egress.

### Component: `SupervisorPublicGateway`
- **Classification**: **`P01-D3B DESIGN DECISION PENDING`**
- **Purpose**: Public HTTPS MCP proxy required if public plugin distribution is pursued after publisher identity verification.
- **Status**: Gated pending identity verification. Zero implementation authorized in Phase P01.

### Component: `Plugin Skills Packaging`
- **Classification**: **`OPTIONAL PACKAGING/WORKFLOW LAYER`**
- **Purpose**: Packaging static prompt guidance (`SKILL.md`) alongside MCP tools.
- **Status**: Optional. Prohibited from duplicating or bypassing server-side `TaskContractManager` or `PolicyEngine` invariants.
