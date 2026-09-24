# REUSE MATRIX — ANTI-REINVENTION MAP

> **Authority**: Binding Capability Mapping & Anti-Reinvention Constraints
> **Status**: Literal Evidence Corrected (Post-Re-Audit #4 Patch)

---

# 1. Verified Anti-Reinvention Capability Map

| Capability | Requirement | Source Repo | Exact Source Area (Verified in Upstream) | Strategy | Our Module | Why We Own This | Forbidden Duplicate |
|---|---|---|---|---|---|---|---|
| **Process Daemon & Lifecycle** | NFR-001, OPS-002 | `Untrivial-ai/agent-orchestrator` | `backend/internal/httpd/router.go` (`NewRouterWithControl`, `mountHealth`) | `UPSTREAM` | `AOAdapter` | AO provides industrial daemon; we connect via loopback REST API. Exact endpoints documented in `UPSTREAM_CONTRACT_BASELINE.md`. | Custom process manager or background daemon |
| **Windows ConPTY Terminal** | NFR-001 | `Untrivial-ai/agent-orchestrator` | `backend/internal/adapters/runtime/conpty/host_conpty_windows.go` (`newConPTY`) | `UPSTREAM` | `AOAdapter` | Windows pseudoterminal allocation is handled upstream by AO via go-pty. | Proprietary PTY or raw cmd.exe wrapper |
| **Git Worktree Isolation** | FR-005, SEC-005 | `Untrivial-ai/agent-orchestrator` | `backend/internal/adapters/workspace/gitworktree/workspace.go` (`validateManagedPath`, `managedPath`) | `UPSTREAM` | `AOAdapter` | AO manages workspace paths, traversal validation, and branch isolation. | Custom Git clone or worktree manager |
| **Agent Harness Adaption** | FR-006 | `Untrivial-ai/agent-orchestrator` | `backend/internal/adapters/agent/agy/agy.go` (`GetLaunchCommand`, `GetRestoreCommand`) | `UPSTREAM` | `AOAdapter` | AO integrates agent harnesses including Antigravity CLI via the ports.Agent interface. | Custom harness runner or CLI spawner |
| **Autonomous Headless Coding** | FR-005 | `google-antigravity/antigravity-cli` | `README.md` & `agy --help` (`-p`, `--output-format`, `--input-format`, `--json-schema`, `--add-dir`) | `UPSTREAM` | `AOAdapter` / Worker | Antigravity CLI v1.2.7 is the autonomous code editing engine. | Custom code generation or editing bot |
| **Security Containment** | SEC-001, SEC-002 | `tt-a1i/proxide` | `SECURITY.md` (`## Safer Defaults`, `## Hard Rules`, `## Mode Boundaries`) | `REIMPLEMENT_PATTERN` | `PolicyEngine` | Proxide proves readonly default, explicit roots, and mode boundaries. | Unchecked filesystem access or arbitrary path traversal |
| **Workspace / Pair Binding** | FR-002 | `Mieruko/MCP_Plugins_With_ChatGPTWeb` | `AGENTS.md` (`## Quyền truy cập`), `README.md` (`## Active Agents and session recovery`), `src/admin/routes.ts` (`createAdminRouter`) | `REIMPLEMENT_PATTERN` | `PairRegistry` | Mieruko proves conversation-to-task binding; Pair abstraction is our design. | Unbound multi-project shared state |
| **Evidence-First Audit** | FR-008, FR-010 | `shrec/AIWorkHub` | `docs/QUALITY_CONTROL.md` (`## Acceptance pipeline`, item `2. Worker isolation and self-validation`, item `5. Manager acceptance and integration proof`) | `REIMPLEMENT_PATTERN` | `EvidenceCollector` | AIWorkHub proves worker claims are candidate hypotheses vs. Git ground truth. | Trusting worker claims without independent diff |
| **Review Bundle Builder** | FR-010 | `shrec/AIWorkHub` / `lookmanrays/codencer` | `docs/QUALITY_CONTROL.md` (AIWorkHub), `README.md` (Codencer) | `ORIGINAL` | `ReviewBundleBuilder` | Cognitive compression combining claims, diff, and test evidence for ChatGPT. | Forcing ChatGPT to make 20+ raw tool calls |
| **Workflow as Policy** | FR-004, FR-011 | `openai/symphony` | `SPEC.md` (`## 4. Core Domain Model`, `## 7. Orchestration State Machine`, `## 9. Workspace Management and Safety`) | `ADAPT` | `StateMachine` | Symphony provides workflow-policy, authoritative orchestrator state, retry/reconciliation, and workspace lifecycle patterns adapted into our state model. | Free-form chat prompt iteration |
| **Bridge-not-Brain Model** | FR-003, ADR-007 | `lookmanrays/codencer` | `README.md` (`## Architecture & Philosophy`), `cmd/broker/main.go` | `REIMPLEMENT_PATTERN` | `SupervisorCore` | Codencer proves state resides outside LLM chat context. | Implementing an autonomous LLM planner in daemon |
| **Provider Abstraction** | NFR-005 | `awslabs/cli-agent-orchestrator` | `src/cli_agent_orchestrator/providers/` (`provider.py`), `README.md` | `ADAPT` | `AOAdapter` | Clean anti-corruption abstraction shielding domain from runtime changes. | Hardcoding direct AO calls throughout domain |
| **CDP Browser Bridge** | Fallback | `cafeTechne/antigravity-link-extension`| `src/services/cdp.ts`, `src/server/index.ts`, `mcp-server.mjs` | `FALLBACK` | `AgyFallbackAdapter` | Retained strictly as fallback if CLI proves inadequate (NFR-002 non-V1). | GUI automation as default workflow |

# 2. Evidence Rule

For code evidence in this matrix and all source dossiers, prefer `file path` + `symbol name` over manually copied long signatures. Only include a full signature if copied byte-for-byte from the pinned source. This prevents hybrid-signature errors.

| **Secure MCP Tunnel** | NFR-005, SEC-001 | `openai/tunnel-client` | `https://developers.openai.com/api/docs/guides/secure-mcp-tunnels` | `UPSTREAM` | External / `tunnel-client` | Outbound-only tunnel connecting private local MCP to OpenAI without inbound ports. | Custom WebSocket relay or reverse SSH tunnel |
| **MCP Protocol / SDK** | NFR-005 | `modelcontextprotocol` | `@modelcontextprotocol/sdk` (TypeScript) / `mcp` (Python) | `UPSTREAM` | `SupervisorMCPAdapter` | Official Model Context Protocol SDK provides streamable HTTP server and schema validation. | Custom ad-hoc JSON-RPC protocol |
| **Responses API MCP** | FR-001, NFR-005 | OpenAI Platform | `https://developers.openai.com/api/docs/guides/tools-connectors-mcp` | `UPSTREAM` | OpenAI Product Capability | OpenAI runtime invokes MCP tools via `tunnel_id` with native approval lifecycle. | Reimplementing LLM-side tool calling |
| **Supervisor Domain MCP Tools** | FR-001, FR-002, SEC-002 | `ai-supervisor` (In-House) | `supervisor_probe_read`, `supervisor_probe_write` | `ORIGINAL` | `SupervisorMCPAdapter` | Domain-specific read-only and state-mutating tools strictly bounded to local sandbox. | Exposing raw OS execution or arbitrary filesystem access |
| **Production Public MCP Gateway**| FR-001, NFR-005 | `ai-supervisor` (In-House) | Architecture V2 Proposal | `PENDING_DESIGN`| `SupervisorPublicGateway` | P01-D3B design decision pending identity verification and public marketplace review. | Exposing unprotected local node directly to internet |
| **Plugin Skills Packaging** | FR-004 | OpenAI Plugins Platform | `https://developers.openai.com/plugins/build/skills` | `OPTIONAL` | Future Guidance Layer | Optional static instruction layer; must never duplicate canonical TaskContract/Policy logic. | Duplicating server-side policy in LLM prompt |

## Execution budget reuse decision

Runtime timeout monitor tái sử dụng `stop.Coordinator` và `Store.ReserveStopOperation` của 3C cho one-use `/kill`, D11/D12 closure và atomic double quarantine; startup `Run` không effect. Supervisor bổ sung v5 budget/policy provenance vì pinned AO không lưu dữ liệu này. Trusted host boundary hiện có được mở thành maintenance scope độc quyền, không dựng admission thứ hai hoặc suy cross-process ownership từ mutex. Legacy `DISPATCHED` không được ép `RUNNING` để gọi stop. Contract revision 3 đã RELEASED; implementation 3D EXTERNAL_AUDIT_APPROVED và code đã MERGED tại `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`. Không đổi classification của upstream trong bảng trên.


## Host Bootstrap & Daemon Lifecycle Reuse Decision (ADR-017)

- **Windows Win32 Kernel APIs**: Tái sử dụng `CreateFileW` (share mode 0 cho sidecar lock, và share READ|WRITE không DELETE cho pinned DB handle), `GetFinalPathNameByHandleW` (chuẩn hóa đường dẫn), `GetFileInformationByHandleEx` (`FILE_ID_INFO` 128-bit trên ReFS/NTFS).
- **Store SQLite Layer**: Tái sử dụng trọn vẹn `internal/store` và `store.Open(ctx, store.Config{DBPath, BusyTimeoutMs})` mà không viết lại kết nối SQLite hay duplicate migrations.
- **Domain & Lifecycle Engines**: Tái sử dụng `recovery.Runner`, `stop.Coordinator`, `audit.Store` cho integration harness của Phase P03 mà không tạo mới test surfaces dư thừa.
