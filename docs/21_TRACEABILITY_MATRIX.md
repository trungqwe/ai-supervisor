# 21. TRACEABILITY MATRIX

> **Authority**: End-to-End Requirement Traceability
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

| Requirement ID | Canonical Architecture Ref | ADR Ref | Target Module | Phase | Acceptance Test | Source Technology |
|---|---|---|---|---|---|---|
| **FR-001** (Project Registration) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-001 | `ProjectRegistry` | P02 / P05 | Test project schema validation (P02); test registration tool surface (P05) | Mieruko |
| **FR-002** (Pair Binding) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-001 | `PairRegistry` | P02 / P05 | Test 1:1 pair constraint in schema (P02); test interactive handshake (P05) | Mieruko |
| **FR-003** (Context Access) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-007 | `ContextEngine` | P05 | Test read-only indexed outline retrieval tool for ChatGPT | Codencer |
| **FR-004** (Task Contract) | `docs/08_TASK_CONTRACT.md` | ADR-010, ADR-012, ADR-013 | `TaskContractManager` | P02 | Validate schema, immutability, revision lineage & verification_requests | Symphony |
| **FR-005** (Worker Dispatch) | `docs/04_ARCHITECTURE.md#sec-2.1` | ADR-002, ADR-012, ADR-016 | `AOAdapter` | P03 | Verify decoupled session provisioning, attempt pre-allocation, durable 3-stage dispatch saga (`DISPATCH_BOUND` -> `SEND_REQUESTED` -> `SEND_CONFIRMED`), & pre-send admissibility whitelist | Agent Orchestrator |
| **FR-006** (Worker Observation) | `docs/04_ARCHITECTURE.md#sec-2.1` | ADR-002 | `AOAdapter` + Supervisor Core | P03 | Authoritative session/activity observation, turn completion detection, and bounded execution timeout validation (no synthetic heartbeat) | Agent Orchestrator |
| **FR-007** (Report Ingestion) | `docs/09_WORKER_REPORT.md` | ADR-006, ADR-011, ADR-012 | `EvidenceCollector` | P04 | Attempt-scoped report fetch via AO workspace API & schema validation | AIWorkHub |
| **FR-008** (Evidence Collection) | `docs/04_ARCHITECTURE.md#sec-2.2` | ADR-006, ADR-011, ADR-012, ADR-013 | `EvidenceCollector` | P04 | Independent git diff & constrained test runner execution bound to TaskAttempt | AIWorkHub |
| **FR-009** (Scope Violation) | `docs/07_SECURITY_MODEL.md#sec-2` | ADR-010 | `PolicyEngine` | P04 | Flag file touched outside allowed_scope | Proxide |
| **FR-010** (Review Bundle) | `docs/10_REVIEW_BUNDLE.md` | ADR-006, ADR-011, ADR-012 | `ReviewBundleBuilder` | P04 | Attempt-scoped bundle schema compliance | AIWorkHub / Codencer |
| **FR-011** (Supervisor Decision)| `docs/06_WORKFLOW_STATE_MACHINE.md` | ADR-005, ADR-011, ADR-012 | `StateMachine` | P02 | Test APPROVE / REVISION transitions bound to task_id and attempt_id | Symphony |
| **FR-012** (Revision Loop) | `docs/06_WORKFLOW_STATE_MACHINE.md` | ADR-005, ADR-011, ADR-012 | `StateMachine` / `AOAdapter` / `ReviewPolicyEngine` | P02 / P03 / P04 | P02: REVISION_REQUIRED transition & new contract validation; P03: AO worker re-dispatch; P04: review outcome feedback | Symphony |
| **FR-013** (Audit Trail) | `docs/15_OBSERVABILITY.md#sec-2` | ADR-007, ADR-012 | `AuditLogger` | P02 / P05 / P06 | Test append-only persistence (P02); tool read (P05); export/dashboard (P06) | Proxide |
| **FR-014** (Multi-Project Domain)| `docs/05_DOMAIN_MODEL.md#sec-1` | ADR-001 | `DomainModel` | P02 | Unit test project isolation | Mieruko |
| **FR-015** (Upstream Health) | `docs/12_UPSTREAM_INTEGRATION.md#sec-1` | ADR-002 | `AOAdapter` | P03 | Preflight daemon liveness (`/healthz`), readiness (`/readyz`), harness presence/readiness (`/api/v1/agents`), and API compatibility/provenance validation | Agent Orchestrator |
| **FR-016** (Supervisor Tool Transport) | `docs/04_ARCHITECTURE.md#sec-3` | ADR-008 | `ToolSurface` / `TransportAdapter` | P01 / P05 | P01-D Feasibility Proof & high-level tool invocation without GUI automation | OpenAI / Mieruko / Proxide |
| **NFR-001** (Windows Support) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-002 | `AOAdapter` | P01 | ConPTY smoke test on Windows | Agent Orchestrator |
| **NFR-002** (Zero GUI Auto) | `docs/04_ARCHITECTURE.md#sec-1` | ADR-004 | Architecture | P00 | Audit verification: no GUI tools | Proxide |
| **NFR-003** (Recoverability) | `docs/14_FAILURE_RECOVERY.md` | ADR-007 | `StateStore` | P02 / P03 | Crash recovery classification (P02); AO-backed session reconciliation (P03) | Codencer |
| **NFR-004** (Auditability) | `docs/15_OBSERVABILITY.md` | ADR-007, ADR-012 | `AuditLogger` | P02 | Audit trail lineage integrity and tamper-evident record verification | Proxide |
| **NFR-005** (Upstream Decoupling)| `docs/12_UPSTREAM_INTEGRATION.md` | ADR-009 | `AOAdapter` | P03 | Verify zero AO DTO leak into domain | AWS CAO |
| **NFR-006** (Minimal Tool Surface)| `docs/11_CHATGPT_TOOL_SURFACE.md` | ADR-008 | `ToolSurface` | P05 | Tool count <= 12 high-level tools with attempt binding | Mieruko |
| **NFR-007** (Version Pinning) | `third_party/SOURCE_VERSIONS.md` | ADR-009 | Specification | P00 | Pinning audit: all sources pinned with 40-char SHA | Inherent |
| **NFR-008** (Performance) | `docs/10_REVIEW_BUNDLE.md` | ADR-006 | `ReviewBundleBuilder` | P04 | Benchmark bundle creation < 3 sec | Inherent |
| **SEC-001** (Readonly Supervisor)| `docs/07_SECURITY_MODEL.md` | ADR-005 | `ToolSurface` | P05 | Verify no file write tools exposed & no auto-merge | Proxide |
| **SEC-002** (Path Containment) | `docs/07_SECURITY_MODEL.md` | ADR-005 | `PolicyEngine` | P02 / P04 | Pre-dispatch scope path containment (P02); post-execution diff checking (P04) | Proxide |
| **SEC-003** (No Arbitrary Shell) | `docs/07_SECURITY_MODEL.md` | ADR-004, ADR-013 | `VerificationRunner` / `ToolSurface` | P04 / P05 | P04: host-profile command authority & Layer-B execution isolation; P05: ChatGPT exposes zero arbitrary shell/process tools | Proxide |
| **SEC-004** (Secret Sanitization) | `docs/07_SECURITY_MODEL.md` | ADR-007 | `AuditLogger` | P02 | Token scrubber regex & credential masking for persisted audit/error data | Proxide |
| **SEC-005** (Scope Enforcement) | `docs/08_TASK_CONTRACT.md` | ADR-010 | `PolicyEngine` | P04 | Rejection test on forbidden scope | Symphony |
| **OPS-001** (Single Command) | `docs/17_ROADMAP.md#sec-1` | ADR-001 | Deployment | P06 | CLI startup command verification | Inherent |
| **OPS-002** (Clean Termination) | `docs/14_FAILURE_RECOVERY.md` | ADR-002, ADR-016 | `SupervisorCore` | P02 / P03 / P05 | StateStore/domain close API (P02); purpose-aware stop lifecycle, restart-stable deadline evaluation, double-gated quarantine cleanup & process signal handling (P03/P05) | Agent Orchestrator |
| **OPS-003** (Self-Contained Store) | `docs/05_DOMAIN_MODEL.md` | ADR-007, ADR-015 | `StateStore` | P02 | Zero cloud database dependency; SQLite WAL + synchronous=FULL durability | AIWorkHub |
