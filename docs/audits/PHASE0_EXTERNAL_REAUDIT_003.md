# PHASE 0 — EXTERNAL RE-AUDIT #3 FINDINGS REGISTER

> **Authority**: External Audit Record — Non-Self-Declarable  
> **Audit Type**: Literal Evidence Hygiene Patch  
> **Status**: PHASE0_BLOCKED → Remediation applied → Pending Re-Audit #3 Verification  
> **Date**: 2026-09-20  
> **Baseline Commit**: 7c7b23917436a4ac11c77e42e2b771f5d032389d  

---

## 1. External Verdict Received

`	ext
External Audit #3 Verdict: PHASE0_BLOCKED
Reason: Residual non-literal upstream symbols and headings in evidence records.
`

The architecture itself was accepted as stable. Only the literal accuracy of upstream symbol citations in docs/sources/01_AGENT_ORCHESTRATOR.md and docs/sources/REUSE_MATRIX.md was found defective.

---

## 2. Findings Register

### EXT3-001 — AO-CLAIM-003: Non-Existent Router Symbols Cited

**File**: docs/sources/01_AGENT_ORCHESTRATOR.md  
**Finding**: Section / symbol: setupRouter / registerRoutes — neither setupRouter nor egisterRoutes exist in ackend/internal/httpd/router.go at pinned commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6.  
**Verification**: Directly fetched raw file from https://raw.githubusercontent.com/Untrivial-ai/agent-orchestrator/15e9ea971f1711ec8b50e157d6eb300db6cbe0d6/backend/internal/httpd/router.go. The sole exported function is NewRouterWithControl(cfg config.Config, log *slog.Logger, termMgr *terminal.Manager, deps APIDeps, control ControlDeps) chi.Router. Internal mount functions include mountHealth, mountTerminalMux, mountControl, mountAgentSwitchPolicyControl, mountTelemetry, mountMobile, mountMobileDevices. pi.Register(r) registers remaining API routes.  
**Correction**: Replace Section / symbol: setupRouter / registerRoutes with Section / symbol: func NewRouterWithControl(cfg config.Config, log *slog.Logger, termMgr *terminal.Manager, deps APIDeps, control ControlDeps) chi.Router. Update notes to cite the literal mount function set.  
**Status**: **RESOLVED**. Applied 2026-09-20.

---

### EXT3-002 — AO-CLAIM-004: Non-Existent Adapter Symbol Cited

**File**: docs/sources/01_AGENT_ORCHESTRATOR.md  
**Finding**: Section / symbol: CommandBuilder / argv construction — CommandBuilder does not exist in ackend/internal/adapters/agent/agy/agy.go at pinned commit. gy.go defines a Plugin struct and unc New() *Plugin. The argv construction methods are defined by the ports.Agent interface: GetLaunchCommand and GetRestoreCommand.  
**Verification**: Directly fetched gy.go and ackend/internal/ports/agent.go. The Agent interface at ports/agent.go defines GetLaunchCommand(ctx context.Context, cfg LaunchConfig) (cmd []string, err error) and GetRestoreCommand(ctx context.Context, cfg RestoreConfig) (cmd []string, ok bool, err error). Plugin satisfies this interface.  
**Correction**: Replace evidence with ackend/internal/adapters/agent/agy/agy.go (Plugin struct, unc New() *Plugin) plus ackend/internal/ports/agent.go (Agent interface). Replace symbol citation with the two literal interface method signatures.  
**Status**: **RESOLVED**. Applied 2026-09-20.

---

### EXT3-003 — REUSE_MATRIX: Non-Literal Symbols in Two Rows

**File**: docs/sources/REUSE_MATRIX.md  
**Finding Row 1**: Exact Source Area column for "Process Daemon & Lifecycle" cited outer.go (setupRouter) — setupRouter does not exist.  
**Finding Row 2**: Exact Source Area column for "Agent Harness Adaption" cited gy/agy.go (BuildCommand) — BuildCommand does not exist.  
**Correction Row 1**: Replace setupRouter with NewRouterWithControl (the sole exported router function).  
**Correction Row 2**: Replace BuildCommand with GetLaunchCommand / GetRestoreCommand (the interface-mandated argv construction methods from ports/agent.go implemented by Plugin).  
**Status**: **RESOLVED**. Applied 2026-09-20.

---

## 3. Remediation Evidence

| Blocker | File Modified | Incorrect Value | Corrected Value | Verified From |
|---|---|---|---|---|
| EXT3-001 | docs/sources/01_AGENT_ORCHESTRATOR.md | setupRouter / registerRoutes | unc NewRouterWithControl(...) | ackend/internal/httpd/router.go @ 15e9ea97 |
| EXT3-002 | docs/sources/01_AGENT_ORCHESTRATOR.md | CommandBuilder / argv construction | GetLaunchCommand / GetRestoreCommand | ackend/internal/ports/agent.go @ 15e9ea97 |
| EXT3-003 | docs/sources/REUSE_MATRIX.md | setupRouter, BuildCommand | NewRouterWithControl, GetLaunchCommand / GetRestoreCommand | Same pinned sources |

---

## 4. Scope of Change

This remediation is **literal evidence correction only**.

- Architecture: **UNCHANGED**  
- Technology decisions: **UNCHANGED**  
- All other source dossiers: **UNCHANGED**  
- Phase 0 guardrails: **MAINTAINED** (zero application code, zero packages, zero scripts)

---

## 5. Agent Self-Assessment Verdict

`	ext
================================================================================
PHASE 0 RE-AUDIT #3 AGENT SELF-ASSESSMENT:
PHASE0_READY_FOR_EXTERNAL_AUDIT
================================================================================
`

Per AGENTS.md Section 2, Rule 5: this self-assessment is a **candidate claim only**.  
ARCHITECTURE_FROZEN can only be declared by the User and External Supervisor after independent re-audit.

---

> **Supersession Note**: This record supersedes PHASE0_EVIDENCE_FINAL_AUDIT.md for the three findings above. All other prior audit findings remain in their respective records.
