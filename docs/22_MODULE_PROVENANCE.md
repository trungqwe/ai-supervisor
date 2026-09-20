# 22. MODULE PROVENANCE & ANTI-REINVENTION JUSTIFICATION

> **Authority**: Core Module Ownership Register & Anti-Reinvention Proof  
> **Status**: Approved Baseline

---

# 1. Planned Modules Provenance

### Module: `ProjectRegistry`
- **Purpose**: Tracks registered local projects, validates local directory existence, and stores project metadata.
- **Origin**: Mieruko workspace binding pattern.
- **Existing Upstream Capability Checked**: YES (AO has project registration, but lacks Supervisor-specific documentation indexing).
- **Reason This Module Exists in Our Code**: Links local workspace paths to canonical architecture docs and Pair state.
- **Why We Own This**: AO manages execution paths; Supervisor owns project documentation indexing and governance metadata.
- **Forbidden Responsibility**: Does NOT manage Git repositories or create worktrees.

### Module: `PairRegistry`
- **Purpose**: Manages the engineering lane binding one project, one ChatGPT session, and one active worker session.
- **Origin**: Mieruko workspace binding concepts.
- **Existing Upstream Capability Checked**: YES (AO manages sessions, but has no concept of ChatGPT Supervisor binding).
- **Reason This Module Exists in Our Code**: Prevents multi-project interference and validates session tokens.
- **Why We Own This**: Core abstraction of the Supervisor Control Plane.
- **Forbidden Responsibility**: Does NOT manage OS processes or ConPTY allocations.

### Module: `TaskContractManager`
- **Purpose**: Validates, stores, and enforces immutable Task Contracts.
- **Origin**: OpenAI Symphony work units and Codencer task schemas.
- **Existing Upstream Capability Checked**: YES (AO accepts task instructions, but enforces no schema or scope immutability).
- **Reason This Module Exists in Our Code**: Enforces strict mathematical scope bounds on worker tasks.
- **Why We Own This**: Defines the authoritative work contract between Supervisor and Worker.
- **Forbidden Responsibility**: Does NOT execute tasks or edit code.

### Module: `PolicyEngine`
- **Purpose**: Compares Git diffs against contract `allowed_scope` and flags violations.
- **Origin**: Proxide containment and security model.
- **Existing Upstream Capability Checked**: YES (AO has no scope filtering; workers can touch any file in the worktree).
- **Reason This Module Exists in Our Code**: Prevents worker agents from altering architecture, root configs, or other modules.
- **Why We Own This**: Upstream AO does not enforce architectural boundary policies.
- **Forbidden Responsibility**: Does NOT block OS file access directly; operates as post-execution audit gate.

### Module: `EvidenceCollector`
- **Purpose**: Independently collects base/head SHAs, Git diffs, and process exit codes from the local filesystem.
- **Origin**: AIWorkHub candidate verification and Codencer evidence model.
- **Existing Upstream Capability Checked**: YES (AO reports session exit code, but does not correlate Git diffs or build test audit packets).
- **Reason This Module Exists in Our Code**: Zero-trust verification: worker claims must be corroborated by independent evidence.
- **Why We Own This**: Bridge between Git ground truth and ChatGPT audit.
- **Forbidden Responsibility**: Does NOT execute worker builds; only inspects resulting artifacts.

### Module: `ReviewBundleBuilder`
- **Purpose**: Assembles Task Contract, Worker Claims, Git Diff, Test Artifacts, and Policy Findings into a unified JSON bundle.
- **Origin**: AIWorkHub / Codencer review model.
- **Existing Upstream Capability Checked**: YES (No existing open-source tool builds ChatGPT-tailored review bundles).
- **Reason This Module Exists in Our Code**: Drastically reduces ChatGPT tool calls from 25+ down to 1.
- **Why We Own This**: Proprietary innovation of the AI Engineering Supervisor Control Plane.
- **Forbidden Responsibility**: Does NOT approve or reject tasks; only packages data for ChatGPT.

### Module: `AuditLogger`
- **Purpose**: Records append-only, secret-sanitized chronological logs of all pair, task, and evidence events.
- **Origin**: Proxide audit sanitization.
- **Existing Upstream Capability Checked**: YES (AO logs daemon events, but lacks domain-level pair and review governance trails).
- **Reason This Module Exists in Our Code**: Ensures full auditability and post-mortem diagnosis.
- **Why We Own This**: Authoritative record of supervisor-worker coordination.
- **Forbidden Responsibility**: Does NOT store plaintext secrets or environment passwords.

### Module: `AOAdapter`
- **Purpose**: Translates domain operations into Untrivial Agent Orchestrator public REST API calls.
- **Origin**: AWS CAO provider abstraction.
- **Existing Upstream Capability Checked**: YES (AO provides the public REST API).
- **Reason This Module Exists in Our Code**: Decouples Supervisor domain logic from AO internal changes.
- **Why We Own This**: Anti-corruption layer shielding the domain.
- **Forbidden Responsibility**: Does NOT access AO internal SQLite database or vendor AO code.

### Module: `ChatGPTToolSurface`
- **Purpose**: Exposes the 12 high-level tools via local MCP relay / API.
- **Origin**: Mieruko slim tool surface.
- **Existing Upstream Capability Checked**: YES (Mieruko/Proxide provide generic MCP tools, but none implement our 12-tool contract).
- **Reason This Module Exists in Our Code**: Provides the cognitive interface for ChatGPT Web.
- **Why We Own This**: Tailored cognitive interface for our supervision model.
- **Forbidden Responsibility**: Does NOT expose arbitrary shell, raw file writing, or GUI automation.
