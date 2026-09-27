# 17. ROADMAP & PHASE SPECIFICATION

> **Focus**: Execution Phases P00 through P07 with Deliverables & Exit Gates
> **Status**: Approved Baseline (Remediated Phase 0)

---

# 1. Phase Roadmap Overview

| Phase | Phase Name | Primary Goal | Deliverable | Exit Gate |
|---|---|---|---|---|
| **`P00`** | Architecture Freeze | Complete technical foundation, schemas, dossiers, and audit baseline. | 25 canonical docs, schemas, ADRs, dossiers. | External Supervisor / User Approval (`ARCHITECTURE_FROZEN`). |
| **`P01`** | Upstream Proof & Transport Feasibility | Empirically verify AO daemon (P01-A), Agy CLI (P01-B), AO-Agy adapter gap (P01-C), and ChatGPT Plus transport feasibility (P01-D / FR-016) on Windows. | Upstream proof logs, adapter gap analysis, transport feasibility determination. | Tri-state exit verdict: `PASS` or `GAP_REQUIRES_ADR`. Zero progression if `BLOCKER`. |
| **`P02`** | Supervisor Domain Core | Implement domain entities, 13-state state machine, immutable Task Contract validator, and durable local State Store. | Domain models, state transition engine, unit test suite. | 100% unit test pass on domain state machine and contracts. |
| **`P03`** | AO Integration | Implement `AOAdapter` connecting Supervisor domain to AO REST daemon API, and host bootstrap integration harness (TASK-P03-004) under ADR-016/017. | AOAdapter REST integration, session dispatch saga, observation reconciliation, raw workspace transport, and host bootstrap daemon exclusivity with P03 integration harness. | Automated session creation, dispatch, observation reconciliation, raw workspace file read, and teardown pass without P04 EvidenceCollector dependencies. |
| **`P04`** | Evidence & Review Engine | Implement independent workspace binding, verification isolation, evidence collection, and ReviewBundle compilation. | Subtasks P04A (Workspace Binding & Claims Schema v6), P04B (In-Memory Intake Collector), P04C (In-Memory Windows AppContainer Sandbox Runner), P04D (Verification Lease & ReviewBundle CAS Schema v9). | Verified AppContainer execution, linear lease recovery, and ReviewBundle CAS compilation (`P04_EXIT_GATE = EXTERNAL_AUDIT_APPROVED`). |
| **`P05`** | ChatGPT Tool Interface | Implement the 12 high-level domain tools over proven transport without GUI automation. | ChatGPTToolSurface, TransportAdapter, schema endpoints. | Target ChatGPT Web environment successfully binds project and inspects state via tools. |
| **`P06`** | Single Pair Full Loop | Execute first complete end-to-end task cycle without human copy-paste. | Live demonstration on real code repository. | Successful cycle: Dispatch → Edit → Test → Review → Approval (Decision Recorded, No Auto-Merge). |
| **`P07`** | Multi-Pair Control UI | Develop desktop/web local dashboard for monitoring multiple pairs. | Control UI dashboard, event stream viewer. | Human developer visually monitors multiple concurrent pairs. |

# 2. Phase P04 Subtask Breakdown & Sequencing (ADR-018)

### 2.1 Subtask Sequencing & Model 1 Persistence Ownership (CR-09)
Phase P04 implementation strictly enforces Model 1 persistence ownership and sequential task contract releases:
1. **Subtask P04A (`CONTRACT-TASK-P04-001`) — Workspace Binding & Claims Schema v6**:
   - **Persistence Ownership**: Owns SQLite Schema v6 (`attempt_workspace_bindings`, `worker_claims`, `review_integrity_holds`).
   - **Scope**: First releaseable implementation work package. Canonical workspace binding creation, anti-rename handle management, trigger constraints (`stage = 'DISPATCH_BOUND'`), and active deduplication index.
2. **Subtask P04B (`CONTRACT-TASK-P04-002`) — In-Memory Intake & Git Collector**:
   - **Persistence Ownership**: Pure in-memory (zero migrations, zero SQLite writes, zero audit appends, zero hold mutations).
   - **Scope**: Zero-mutation workspace inspection (`git status --porcelain=v1 -z --untracked-files=all`), clean intake validation, diff collection.
3. **Subtask P04C (`CONTRACT-TASK-P04-003`) — In-Memory Windows AppContainer Sandbox Runner**:
   - **Persistence Ownership**: Pure in-memory (zero migrations, zero SQLite writes, zero audit appends, zero hold mutations).
   - **Scope**: Isolated verification execution using Win32 `CreateProcessW` with `STARTUPINFOEXW`, explicit stdio-only `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`, atomic Job Object assignment with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, network restriction SID, 10MB/50MB stream limits, and process-death proof.
4. **Subtask P04D (`CONTRACT-TASK-P04-004`) — Verification Lease & ReviewBundle CAS Schema v9**:
   - **Persistence Ownership**: Owns SQLite Schema v9 (`task_verification_leases`, `evidence_sets`, `review_artifacts`, `review_bundles`).
   - **Scope**: Linear verification lease chain, Content-Addressed Store write-through to `artifacts/<first-two-hex>/<captured_sha256>`, ReviewBundle assembly with diagnostic latency properties (`compilation_latency_ms`, 2x2 measurement provenance matrix, `nfr008_compliance_status = 'UNVERIFIED'`), RFC 8785 JCS, inert fake AO adapter test harness, and startup recovery scanner. Runtime admission remains closed until P04D startup recovery.

### 2.2 Phase P04 Exit Gate Criteria (CR-11)
- Verified Windows AppContainer sandbox execution with authoritative process-death proof.
- Linear lease recovery and monotonic token validation across process restarts.
- ReviewBundle CAS write-through compilation matching JSON Schema Draft-07.
- Complete unit and integration test suite passing with race detector (`go test -race -count=1`).
