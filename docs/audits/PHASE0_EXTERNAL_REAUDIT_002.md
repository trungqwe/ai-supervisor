# EXTERNAL RE-AUDIT #2 REPORT & FINDINGS REGISTER

> **Audit ID**: AUDIT-P00-EXT-002  
> **Date**: 2026-09-20  
> **Auditor**: External Independent Auditor & Supervisor Pair  
> **Audited Baseline**: `e0868fd377f9e497b310b4ce56b11660431f4b6b`  
> **Scope**: Evidence Integrity, Source Metadata, Section/Symbol Precision, and Upstream Harness Alignment  
> **Verdict**: `PHASE0_BLOCKED` (Requires Evidence Hygiene Patch)

---

# 1. Executive Summary

While the architectural remediation completed in commits `fcc8af8` through `e0868fd` successfully stabilized the domain interfaces, eliminated synthetic HTTP routes, removed automatic branch merge from task approval, and established the 4-track P01 proof plan, External Re-Audit #2 identified several residual inaccuracies in upstream metadata, source-section headings, symbol names, and CLI invocation syntax.

To prevent architectural drift and uphold the zero-invention evidence standard, this document records the authoritative external findings that must be resolved prior to final Phase 0 freeze.

---

# 2. Authoritative External Findings

### Finding EXT2-001 — Agent Orchestrator Commit Date Discrepancy
- **Observation**: AO tag `v0.13.0` (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) was documented with inconsistent dates (`2026-03-05T08:52:13Z` and `2026-06-03`).
- **Verified GitHub API Truth**:
  - `Commit Author Date (UTC)`: `2026-09-12T06:30:25Z`
  - `Commit Committer Date (UTC)`: `2026-09-12T06:30:25Z`
- **Remediation Requirement**: Update all authoritative records to cite verified UTC timestamp `2026-09-12T06:30:25Z`.

### Finding EXT2-002 — Antigravity CLI Commit Date Discrepancy
- **Observation**: Agy release `1.2.7` (`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`) was documented with date `2026-03-12T18:34:01Z`.
- **Verified GitHub API Truth**:
  - `Commit Author Date (UTC)`: `2026-09-19T01:01:48Z`
  - `Commit Committer Date (UTC)`: `2026-09-19T01:01:48Z`
- **Remediation Requirement**: Update all authoritative records to cite verified UTC timestamp `2026-09-19T01:01:48Z`.

### Finding EXT2-003 — Antigravity CLI License & Terms Classification
- **Observation**: Documenting Agy as "Google Proprietary License" overstates verified repository metadata, as GitHub declares no SPDX license.
- **Verified Repository Truth**:
  - `Repository SPDX License`: `NOT DECLARED`
  - `Usage Terms`: `Subject to applicable Google / Antigravity Terms of Service`
- **Remediation Requirement**: Strictly separate SPDX license from usage terms across all source registries and dossiers.

### Finding EXT2-004 — Symphony Evidence Semantics & Section Names
- **Observation**: The dossier cited nonexistent sections (`Section 3: Work Unit Lifecycle & State Transitions`, `Section 4: Workspace Runner Isolation`) and described workspaces as purely ephemeral.
- **Verified Repository Truth**:
  - `SPEC.md` actual headings: `## 4. Core Domain Model`, `## 7. Orchestration State Machine`, `## 9. Workspace Management and Safety`.
  - Upstream spec explicitly states: "Create deterministic per-issue workspaces and preserve them across runs."
  - Upstream Symphony informs our architecture, but our 13-state machine is OUR design.
- **Remediation Requirement**: Re-cite actual headings, document workspace preservation, and adjust claims to reflect conceptual adaptation rather than literal 1-to-1 adoption.

### Finding EXT2-005 — Proxide Evidence & Section Headings
- **Observation**: The dossier cited a broad README section (`Security & Containment`) and generic code files.
- **Verified Repository Truth**:
  - Canonical containment and security boundaries are formally specified in `SECURITY.md`.
  - Actual headings: `## Safer Defaults` (`trust_level=readonly`, allowed roots, path containment), `## Hard Rules` (`write_mode=handoff`), `## Mode Boundaries`.
- **Remediation Requirement**: Re-anchor Proxide containment evidence to `SECURITY.md` sections.

### Finding EXT2-006 — Mieruko Exported Symbol & Conversation Binding Evidence
- **Observation**: Dossier cited `registerAdminRoutes` in `src/admin/routes.ts`, but the exported function is `createAdminRouter`. Furthermore, `routes.ts` does not prove conversation/task binding.
- **Verified Repository Truth**:
  - Exported symbol: `createAdminRouter`.
  - Conversation/task binding is explicitly specified in `AGENTS.md` ("ChatGPT có `_meta["openai/session"]` được gắn task theo cuộc trò chuyện, độc lập kết nối MCP; reconnect và đổi Dashboard không đổi task") and `README.md` (`## Active Agents and session recovery`).
- **Remediation Requirement**: Correct symbol name, cite `AGENTS.md` / `README.md` for session persistence, and clarify that the Pair abstraction is OUR design.

### Finding EXT2-007 — AO Exact Source Area Precision
- **Observation**: `REUSE_MATRIX.md` used high-level directory paths rather than exact implementation files.
- **Verified Repository Truth**:
  - Windows ConPTY implementation: `backend/internal/adapters/runtime/conpty/host_conpty_windows.go` (function `newConPTY`).
  - Git worktree isolation: `backend/internal/adapters/workspace/gitworktree/workspace.go` (`validateManagedPath`, `managedPath`).
- **Remediation Requirement**: Update REUSE_MATRIX to cite exact implementation files and symbols.

### Finding EXT2-008 — AIWorkHub Quality Control Canonical Evidence
- **Observation**: Evidence for worker result skepticism and evidence bundles should cite the dedicated quality specification.
- **Verified Repository Truth**:
  - `docs/QUALITY_CONTROL.md` sections: `## Five quality control layers` ("Worker self-reports are evidence only; they never own PASS/FAIL"), `## Manager acceptance and integration proof`.
- **Remediation Requirement**: Re-anchor evidence to `docs/QUALITY_CONTROL.md`.

### Finding EXT2-009 — P01 Antigravity CLI Command Syntax
- **Observation**: `P01_UPSTREAM_PROOF.md` specified `agy --print -p "<prompt>"`, redundantly combining aliases.
- **Verified CLI Syntax**: `-p`, `--print`, and `--prompt` are aliases. Streaming JSON stdin (`--input-format stream-json`) must not be combined with `-p`.
- **Remediation Requirement**: Standardize on canonical `agy -p "<prompt>"` for non-interactive prompt execution, and document stdin piping for stream-json.

### Finding EXT2-010 — Canonical Architecture AO ↔ Agy Boundary Wording
- **Observation**: The architecture thumbnail labeled the boundary `Headless Process / Harness Invocation`, which asserts headless execution before P01-C proof.
- **Remediation Requirement**: Rename boundary to `Managed Agy Harness Invocation`.

---

# 3. Action Plan

1. Record Phase 0 External Re-Audit #2 triage and transition status to `PHASE0_BLOCKED`.
2. Normalize all repository metadata, dates, licenses, and citations across authoritative files.
3. Conduct final evidence integrity audit and record verdict.
