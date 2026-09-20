# REUSE MATRIX — ANTI-REINVENTION MAP

> **Authority**: Binding Capability Mapping & Anti-Reinvention Constraints  
> **Status**: Verified Documentation Baseline (Post-Remediation)

---

# 1. Verified Anti-Reinvention Capability Map

| Capability | Requirement | Source Repo | Exact Source Area (Verified in Upstream) | Strategy | Our Module | Why We Own This | Forbidden Duplicate |
|---|---|---|---|---|---|---|---|
| **Process Daemon & Lifecycle** | NFR-001, OPS-002 | `Untrivial-ai/agent-orchestrator` | `backend/internal/daemon/`, `backend/internal/httpd/router.go` | `UPSTREAM` | `AOAdapter` | AO provides industrial daemon; we connect via loopback HTTP. | Custom process manager or background runner |
| **Windows ConPTY Terminal** | NFR-001 | `Untrivial-ai/agent-orchestrator` | `backend/internal/terminal/` | `UPSTREAM` | `AOAdapter` | Windows pseudoterminal allocation is handled upstream by AO. | Proprietary PTY or raw cmd.exe wrapper |
| **Git Worktree Isolation** | FR-005, SEC-005 | `Untrivial-ai/agent-orchestrator` | `backend/internal/service/session/` (spawn controllers) | `UPSTREAM` | `AOAdapter` | AO manages workspace paths and session isolation. | Custom Git clone or worktree manager |
| **Agent Harness Adaption** | FR-006 | `Untrivial-ai/agent-orchestrator` | `backend/internal/adapters/agent/agy/agy.go` | `UPSTREAM` | `AOAdapter` | AO integrates agent harnesses including Antigravity CLI. | Custom harness runner or CLI spawner |
| **Autonomous Headless Coding** | FR-005 | `google-antigravity/antigravity-cli` | `README.md` ("Usage"), `agy --help` (`--print`, `--mode`) | `UPSTREAM` | `AOAdapter` / Worker | Antigravity is the code editing engine. | Custom code generation or editing bot |
| **Security Containment** | SEC-001, SEC-002 | `tt-a1i/proxide` | `connector-rs/src/main.rs`, `README.md` ("Security & Containment") | `REIMPLEMENT_PATTERN` | `PolicyEngine` | Proxide proves readonly default and canonical roots. | Unchecked filesystem access or arbitrary paths |
| **Workspace / Pair Binding** | FR-002 | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | `src/admin/routes.ts`, `README.md` ("Workspace Integration") | `REIMPLEMENT_PATTERN` | `PairRegistry` | Mieruko demonstrates ChatGPT-to-workspace binding. | Unbound multi-project shared state |
| **Evidence-First Audit** | FR-008, FR-010 | `shrec/AIWorkHub` | `docs/ARCHITECTURE.md` ("Evidence-First Review"), `docs/QUALITY_CONTROL.md` | `REIMPLEMENT_PATTERN` | `EvidenceCollector` | AIWorkHub proves candidate claims vs. Git ground truth. | Trusting worker claims without independent diff |
| **Review Bundle Builder** | FR-010 | `shrec/AIWorkHub` / `lookmanrays/codencer` | `docs/ARCHITECTURE.md` (AIWorkHub), `README.md` (Codencer) | `ORIGINAL` | `ReviewBundleBuilder` | Specific to our ChatGPT supervision paradigm. | Forcing ChatGPT to make 20 raw inspection calls |
| **Workflow as Policy** | FR-004, FR-011 | `openai/symphony` | `SPEC.md` (Section 3: "Work Unit Lifecycle & State Transitions") | `ADAPT` | `StateMachine` | Symphony formalizes work units and state transitions. | Free-form chat prompt iteration |
| **Bridge-not-Brain Model** | FR-003, ADR-007 | `lookmanrays/codencer` | `README.md` ("Bridge, Not Brain", "Stateful Execution") | `REIMPLEMENT_PATTERN` | `SupervisorCore` | Codencer proves state resides outside LLM chat. | Implementing an autonomous LLM planner in daemon |
| **Provider Abstraction** | NFR-005 | `awslabs/cli-agent-orchestrator` | `src/cli_agent_orchestrator/providers/`, `README.md` | `ADAPT` | `AOAdapter` | Clean abstraction between control plane and runtimes. | Hardcoding direct AO calls throughout domain |
| **CDP Browser Bridge** | Fallback | `cafeTechne/antigravity-link-extension`| `src/services/cdp.ts`, `src/server/index.ts` | `FALLBACK` | `AgyFallbackAdapter` | Retained strictly as fallback if CLI proves inadequate. | GUI automation as default workflow |
