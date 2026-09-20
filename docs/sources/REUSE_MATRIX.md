# REUSE MATRIX — ANTI-REINVENTION MAP

> **Authority**: Binding Capability Mapping & Anti-Reinvention Constraints  
> **Status**: Approved Baseline

---

# 1. Anti-Reinvention Capability Map

| Capability | Requirement | Source | Exact Source Area | Strategy | Our Module | Why We Own This | Forbidden Duplicate |
|---|---|---|---|---|---|---|---|
| **Process Daemon & Lifecycle** | NFR-001, OPS-002 | Agent Orchestrator | `cmd/daemon/`, `pkg/process/` | `UPSTREAM` | `AOAdapter` | AO provides industrial daemon; we only call public API. | Custom process manager or background runner |
| **Windows ConPTY Terminal** | NFR-001 | Agent Orchestrator | `pkg/terminal/conpty_windows.go` | `UPSTREAM` | `AOAdapter` | Windows pseudoterminal allocation is solved upstream. | Proprietary PTY or raw cmd.exe wrapper |
| **Git Worktree Isolation** | FR-005, SEC-005 | Agent Orchestrator | `pkg/worktree/manager.go` | `UPSTREAM` | `AOAdapter` | AO creates/destroys isolated worktrees per session. | Custom Git clone or worktree manager |
| **Agent Harness Adaption** | FR-006 | Agent Orchestrator | `pkg/harness/antigravity.go` | `UPSTREAM` | `AOAdapter` | AO bridges 25+ harnesses including Antigravity CLI. | Custom harness runner or CLI spawner |
| **Headless Autonomous Coding** | FR-005 | Antigravity CLI | Official `agy` binary distribution | `UPSTREAM` | `AOAdapter` / Worker | Antigravity is the primary code editing intelligence. | Custom code generation or editing bot |
| **Security Containment** | SEC-001, SEC-002 | Proxide | `src/security/containment.rs` | `REIMPLEMENT_PATTERN` | `PolicyEngine` | Proxide proves readonly default and canonical roots. | Unchecked filesystem access or arbitrary paths |
| **Workspace / Pair Binding** | FR-002 | Mieruko | `plugins/workspace/binding.py` | `REIMPLEMENT_PATTERN` | `PairRegistry` | Mieruko demonstrates ChatGPT-to-workspace binding. | Unbound multi-project shared state |
| **Evidence-First Audit** | FR-008, FR-010 | AIWorkHub | `core/review/evidence.py` | `REIMPLEMENT_PATTERN` | `EvidenceCollector` | AIWorkHub proves candidate claims vs. Git ground truth. | Trusting worker claims without independent diff |
| **Review Bundle Builder** | FR-010 | AIWorkHub / Codencer | `core/review/bundle.py` | `ORIGINAL` | `ReviewBundleBuilder` | Specific to our ChatGPT supervision paradigm. | Forcing ChatGPT to make 20 raw inspection calls |
| **Workflow as Policy** | FR-004, FR-011 | OpenAI Symphony | `SPEC.md`, `lib/symphony/workflow.ex` | `ADAPT` | `StateMachine` | Symphony formalizes work units and state transitions. | Free-form chat prompt iteration |
| **Bridge-not-Brain Model** | FR-003, ADR-007 | Codencer | `daemon/bridge.go` | `REIMPLEMENT_PATTERN` | `SupervisorCore` | Codencer proves state resides outside LLM chat. | Implementing an autonomous LLM planner in daemon |
| **Provider Abstraction** | NFR-005 | AWS CAO | `cao/providers/base.py` | `ADAPT` | `AOAdapter` | Clean abstraction between control plane and runtimes. | Hardcoding direct AO calls throughout domain |
| **CDP Browser Bridge** | Fallback | Antigravity Link | `src/cdp/bridge.ts` | `FALLBACK` | `AgyFallbackAdapter` | Only used if official headless CLI proves inadequate. | GUI automation as default workflow |
