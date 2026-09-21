# 21. TRACEABILITY MATRIX

> **Authority**: End-to-End Requirement Traceability
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

| Requirement ID | Canonical Architecture Ref | ADR Ref | Target Module | Phase | Acceptance Test | Source Technology |
|---|---|---|---|---|---|---|
| **FR-001** (Project Registration) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-001 | `ProjectRegistry` | P02 | Test project registration & root validation | Mieruko |
| **FR-002** (Pair Binding) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-001 | `PairRegistry` | P02 | Test single pair binding per workspace | Mieruko |
| **FR-003** (Context Access) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-007 | `ContextEngine` | P02 | Test indexed outline retrieval | Codencer |
| **FR-004** (Task Contract) | `docs/08_TASK_CONTRACT.md` | ADR-010, ADR-012 | `TaskContractManager` | P02 | Validate schema, immutability & revision lineage | Symphony |
| **FR-005** (Worker Dispatch) | `docs/04_ARCHITECTURE.md#sec-2.1` | ADR-002, ADR-012 | `AOAdapter` | P03 | Verify worktree spawn, attempt pre-allocation & task send | Agent Orchestrator |
| **FR-006** (Worker Observation) | `docs/04_ARCHITECTURE.md#sec-2.1` | ADR-002 | `AOAdapter` | P03 | Test heartbeat & event stream | Agent Orchestrator |
| **FR-007** (Report Ingestion) | `docs/09_WORKER_REPORT.md` | ADR-006, ADR-011, ADR-012 | `EvidenceCollector` | P04 | Attempt-scoped report fetch via AO workspace API & schema validation | AIWorkHub |
| **FR-008** (Evidence Collection)| `docs/04_ARCHITECTURE.md#sec-2.2` | ADR-006, ADR-011, ADR-012 | `EvidenceCollector` | P04 | Independent git diff & constrained test runner execution bound to TaskAttempt | AIWorkHub |
| **FR-009** (Scope Violation) | `docs/07_SECURITY_MODEL.md#sec-2` | ADR-010 | `PolicyEngine` | P04 | Flag file touched outside allowed_scope | Proxide |
| **FR-010** (Review Bundle) | `docs/10_REVIEW_BUNDLE.md` | ADR-006, ADR-011, ADR-012 | `ReviewBundleBuilder` | P04 | Attempt-scoped bundle schema compliance | AIWorkHub / Codencer |
| **FR-011** (Supervisor Decision)| `docs/06_WORKFLOW_STATE_MACHINE.md` | ADR-005, ADR-011, ADR-012 | `StateMachine` | P02 | Test APPROVE / REVISION transitions bound to task_id and attempt_id | Symphony |
| **FR-012** (Revision Loop) | `docs/06_WORKFLOW_STATE_MACHINE.md` | ADR-005, ADR-011, ADR-012 | `StateMachine` | P04 | Test re-dispatch with new contract revision & attempt_id | Symphony |
| **FR-013** (Audit Trail) | `docs/15_OBSERVABILITY.md#sec-2` | ADR-007, ADR-012 | `AuditLogger` | P02 | Test append-only JSONL with contract and attempt correlation | Proxide |
| **FR-014** (Multi-Project Domain)| `docs/05_DOMAIN_MODEL.md#sec-1` | ADR-001 | `DomainModel` | P02 | Unit test project isolation | Mieruko |
| **FR-015** (Upstream Health) | `docs/12_UPSTREAM_INTEGRATION.md#sec-1` | ADR-002 | `AOAdapter` | P01 | Live /healthz & /readyz check on AO daemon | Agent Orchestrator |
| **FR-016** (Supervisor Tool Transport) | `docs/04_ARCHITECTURE.md#sec-3` | ADR-008 | `ToolSurface` / `TransportAdapter` | P01 / P05 | P01-D Feasibility Proof & high-level tool invocation without GUI automation | OpenAI / Mieruko / Proxide |
| **NFR-001** (Windows Support) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-002 | `AOAdapter` | P01 | ConPTY smoke test on Windows | Agent Orchestrator |
| **NFR-002** (Zero GUI Auto) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-004 | Architecture | P00 | Audit verification: no GUI tools | Proxide |
| **NFR-003** (Recoverability) | `docs/14_FAILURE_RECOVERY.md` | ADR-007 | `StateStore` | P02 | Crash recovery simulation test | Codencer |
| **NFR-004** (Auditability) | `docs/15_OBSERVABILITY.md` | ADR-007, ADR-012 | `AuditLogger` | P02 | Audit trail lineage integrity check | Proxide |
| **NFR-005** (Upstream Decoupling)| `docs/12_UPSTREAM_INTEGRATION.md` | ADR-009 | `AOAdapter` | P03 | Verify zero AO DTO leak into domain | AWS CAO |
| **NFR-006** (Minimal Tool Surface)| `docs/11_CHATGPT_TOOL_SURFACE.md` | ADR-008 | `ToolSurface` | P05 | Tool count <= 12 high-level tools with attempt binding | Mieruko |
| **NFR-007** (Version Pinning) | `third_party/SOURCE_VERSIONS.md` | ADR-009 | Specification | P00 | Pinning audit: all sources pinned with 40-char SHA | Inherent |
| **NFR-008** (Performance) | `docs/10_REVIEW_BUNDLE.md` | ADR-006 | `ReviewBundleBuilder` | P04 | Benchmark bundle creation < 3 sec | Inherent |
| **SEC-001** (Readonly Supervisor)| `docs/07_SECURITY_MODEL.md` | ADR-005 | `ToolSurface` | P05 | Verify no file write tools exposed & no auto-merge | Proxide |
| **SEC-002** (Path Containment) | `docs/07_SECURITY_MODEL.md` | ADR-005 | `PolicyEngine` | P02 | Test rejection of path traversal & escapes | Proxide |
| **SEC-003** (No Arbitrary Shell)| `docs/07_SECURITY_MODEL.md` | ADR-004 | `ToolSurface` / `EvidenceCollector` | P05 | Verify no shell tools for ChatGPT; constrained runner for tests | Proxide |
| **SEC-004** (Secret Sanitization)| `docs/07_SECURITY_MODEL.md` | ADR-007 | `AuditLogger` | P02 | Token scrubber regex test | Proxide |
| **SEC-005** (Scope Enforcement) | `docs/08_TASK_CONTRACT.md` | ADR-010 | `PolicyEngine` | P04 | Rejection test on forbidden scope | Symphony |
| **OPS-001** (Single Command) | `docs/17_ROADMAP.md#sec-1` | ADR-001 | Deployment | P06 | CLI startup command verification | Inherent |
| **OPS-002** (Clean Termination) | `docs/14_FAILURE_RECOVERY.md` | ADR-002 | `SupervisorCore` | P02 | SIGTERM shutdown test | Agent Orchestrator |
| **OPS-003** (Self-Contained Store)| `docs/05_DOMAIN_MODEL.md` | ADR-007 | `StateStore` | P02 | Zero cloud database dependency test | AIWorkHub |