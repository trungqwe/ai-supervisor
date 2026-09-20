# Implementation Plan (Patched) — Phase 0 Technical Foundation & Specifications

Initialize and establish the complete technical foundation, specifications, and architecture documentation for **Phase 0** of the **AI Engineering Supervisor Control Plane**, incorporating all 19 required audit corrections. This plan produces a drift-resistant, evidence-based repository baseline, places the canonical `PROJECT OVERVIEW` as document `00`, verifies internal consistency, and securely publishes to GitHub (`https://github.com/trungqwe/ai-supervisor`).

## User Review Required

> [!IMPORTANT]
> **Audit Verdict & Authority Boundaries (Patches #1 & #16)**:
> - The agent **CANNOT** self-declare `ARCHITECTURE_FROZEN`.
> - Deliverable of this phase is `docs/audits/PHASE0_INTERNAL_AUDIT.md` concluding either **`PHASE0_READY_FOR_EXTERNAL_AUDIT`** or **`PHASE0_BLOCKED`**.
> - Status in `docs/18_CURRENT_STATE.md` will be set strictly to `PHASE0_READY_FOR_EXTERNAL_AUDIT`.
> - `ARCHITECTURE_FROZEN` and `docs/audits/PHASE0_FREEZE_RECORD.md` are reserved exclusively for User + External Supervisor audit.

> [!IMPORTANT]
> **Execution Constraint**:
> As mandated by Phase 0 rules, **NO application source code**, backend/frontend frameworks, or database engines will be implemented or scaffolded. All work is purely technical specifications, JSON schemas, schema test examples, ADRs, matrices, and Markdown documentation.

> [!NOTE]
> **Git Safety Gate (Patch #13)**:
> No force-push (`--force`) and no history rewrite will ever be used. Pre-push checks will run `git ls-remote origin`. If unexpected remote history exists, execution halts in `BLOCKED` status.

---

## Approved Phase 0 Structure (Patches #2, #5, #8, #9, #11, #15, #18)

```text
D:\TU_CODE\ai-supervisor\
├── README.md                                 # Summary, architecture thumbnail, status, pointers to canonical docs
├── AGENTS.md                                 # Root operational rules & immutable constraints for agents
├── project-overview.md                       # Pointer to canonical docs/00_PROJECT_OVERVIEW.md
├── MASTER EXECUTION PROMPT.md                # Preserved prompt specification
├── .gitignore                                # Minimal neutral gitignore (no premature stack bias)
├── docs/
│   ├── 00_PROJECT_OVERVIEW.md                # CANONICAL Project Overview (single source of truth)
│   ├── 01_PROJECT_CHARTER.md                 # Purpose, Problem, Goals, Principles
│   ├── 02_REQUIREMENTS.md                    # FR-xxx, NFR-xxx, SEC-xxx, OPS-xxx
│   ├── 03_SYSTEM_CONTEXT.md                  # System context diagram, actors, trust boundaries
│   ├── 04_ARCHITECTURE.md                    # Canonical 3-plane architecture & execution flows
│   ├── 05_DOMAIN_MODEL.md                    # Entities, value objects, relationships
│   ├── 06_WORKFLOW_STATE_MACHINE.md          # 13 canonical states, transitions, triggers
│   ├── 07_SECURITY_MODEL.md                  # Readonly default, explicit roots, threat model
│   ├── 08_TASK_CONTRACT.md                   # Immutable Task Contract specification
│   ├── 09_WORKER_REPORT.md                   # Worker Report contract (claims vs independent proof)
│   ├── 10_REVIEW_BUNDLE.md                   # Review Bundle specification & builder logic
│   ├── 11_CHATGPT_TOOL_SURFACE.md            # High-level tool surface specification
│   ├── 12_UPSTREAM_INTEGRATION.md            # AOAdapter & Agy fallback adapter contracts
│   ├── 13_UPSTREAM_UPDATE_POLICY.md          # Active upstream update policy & step-by-step workflow
│   ├── 14_FAILURE_RECOVERY.md                # 15 failure scenarios, recovery owners & playbooks
│   ├── 15_OBSERVABILITY.md                   # Canonical event taxonomy, audit trail, sanitization
│   ├── 16_TEST_STRATEGY.md                   # Test layers (unit to E2E verification)
│   ├── 17_ROADMAP.md                         # Phases P00 to P07
│   ├── 18_CURRENT_STATE.md                   # Initial state: PHASE0_READY_FOR_EXTERNAL_AUDIT
│   ├── 19_OPEN_QUESTIONS.md                  # Open questions marked UNDECIDED / GIẢ ĐỊNH
│   ├── 20_BACKLOG_AND_NON_GOALS.md           # Explicit V1 non-goals and future backlog
│   ├── 21_TRACEABILITY_MATRIX.md             # Requirement to Architecture & Module traceability
│   ├── 22_MODULE_PROVENANCE.md               # Module provenance, upstream checks, & ownership justification
│   ├── 23_COMPATIBILITY_MATRIX.md            # Platform & upstream compatibility tracking
│   ├── 24_CHANGE_GOVERNANCE.md               # Change workflow & strict 9-level Decision Hierarchy
│   │
│   ├── sources/
│   │   ├── SOURCE_REGISTRY.md                # Master index of evaluated technologies
│   │   ├── REUSE_MATRIX.md                   # Anti-reinvention matrix with exact source areas & proof
│   │   ├── UPSTREAM_CONTRACT_BASELINE.md     # Baseline public operations expected from AO & Agy
│   │   ├── 01_AGENT_ORCHESTRATOR.md          # Dossier: Untrivial-ai/agent-orchestrator (Active Upstream)
│   │   ├── 02_ANTIGRAVITY_CLI.md             # Dossier: Official Antigravity CLI (Active Worker Interface)
│   │   ├── 03_PROXIDE.md                     # Dossier: tt-a1i/proxide (Design Source)
│   │   ├── 04_MIERUKO.md                     # Dossier: Mieruko/MCP_Plugins_With_ChatGPTWeb (Design Source)
│   │   ├── 05_AIWORKHUB.md                   # Dossier: shrec/AIWorkHub (Design Source)
│   │   ├── 06_SYMPHONY.md                    # Dossier: openai/symphony (Design Source)
│   │   ├── 07_CODENCER.md                    # Dossier: lookmanrays/codencer (Design Source)
│   │   ├── 08_AWS_CAO.md                     # Dossier: AWS CLI Agent Orchestrator (Design Source)
│   │   └── 09_ANTIGRAVITY_LINK.md            # Dossier: Antigravity Link (Fallback Only)
│   │
│   ├── adr/
│   │   ├── ADR-001-three-plane-architecture.md
│   │   ├── ADR-002-ao-as-execution-control-plane.md
│   │   ├── ADR-003-agy-primary-antigravity-interface.md
│   │   ├── ADR-004-no-gui-automation-by-default.md
│   │   ├── ADR-005-supervisor-does-not-edit-source.md
│   │   ├── ADR-006-evidence-first-review.md
│   │   ├── ADR-007-state-outside-chat.md
│   │   ├── ADR-008-chatgpt-transport-abstraction.md
│   │   ├── ADR-009-upstream-over-reimplementation.md
│   │   └── ADR-010-task-contract-immutability.md
│   │
│   ├── phases/
│   │   ├── P00_ARCHITECTURE_FREEZE.md
│   │   ├── P01_UPSTREAM_PROOF.md
│   │   ├── P02_SUPERVISOR_DOMAIN.md
│   │   ├── P03_AO_INTEGRATION.md
│   │   ├── P04_EVIDENCE_REVIEW.md
│   │   ├── P05_CHATGPT_INTERFACE.md
│   │   ├── P06_SINGLE_PAIR_LOOP.md
│   │   └── P07_MULTI_PAIR_UI.md
│   │
│   ├── proposals/
│   │   └── README.md                         # Proposal template & intake rules
│   │
│   ├── schemas/
│   │   ├── task-contract.schema.json         # JSON Schema for Task Contract
│   │   ├── worker-report.schema.json         # JSON Schema for Worker Report
│   │   ├── review-bundle.schema.json         # JSON Schema for Review Bundle
│   │   └── examples/
│   │       ├── task-contract.valid.json
│   │       ├── task-contract.invalid.json
│   │       ├── worker-report.valid.json
│   │       ├── worker-report.invalid.json
│   │       ├── review-bundle.valid.json
│   │       └── review-bundle.invalid.json
│   │
│   └── audits/
│       └── PHASE0_INTERNAL_AUDIT.md          # Internal audit concluding READY_FOR_EXTERNAL_AUDIT
│
└── third_party/
    ├── SOURCE_VERSIONS.md                    # Active upstreams vs design sources tracking
    └── NOTICES.md                            # Phase 0 third-party notice (no vendored code)
```

---

## Detailed Implementation Breakdown

### 1. Root Layer & Governance (Patches #4, #5, #12)
- **`README.md`**: Executive summary, architecture thumbnail, status banner (`PHASE0_READY_FOR_EXTERNAL_AUDIT`), navigation table directly referencing `docs/00_PROJECT_OVERVIEW.md`.
- **`project-overview.md`**: Clean pointer file redirecting directly to canonical `docs/00_PROJECT_OVERVIEW.md` to prevent multi-source drift.
- **`AGENTS.md`**: Immutable operational directives for coding agents: strict check against `SOURCE_REGISTRY`, no duplicate upstream capabilities, no architectural modifications without ADR, immutable task scopes.
- **`.gitignore`**: Minimal, neutral ignore file containing only `.env`, `.env.*`, `*.log`, `.DS_Store`, `Thumbs.db`, `.vscode`, `.idea`, temporary files, and secrets (no premature Node/Go/Python stack assumptions).

### 2. 25 Canonical Documents (00–24) (Patches #3, #4, #6, #7, #15)
- **`docs/00_PROJECT_OVERVIEW.md`**: Canonical single source of truth describing original pain point, manual vs target workflow, 3 planes, roles, Pair, Task Contract, Review Bundle, and non-goals.
- **`docs/01_PROJECT_CHARTER.md`**: Mission, problem, non-goals, and core principles.
- **`docs/02_REQUIREMENTS.md`**: Testable, atomic IDs (`FR-xxx`, `NFR-xxx`, `SEC-xxx`, `OPS-xxx`).
- **`docs/03_SYSTEM_CONTEXT.md`**: Context diagram, actors, external boundaries (ChatGPT, Supervisor, AO, Agy, Git, GitHub).
- **`docs/04_ARCHITECTURE.md`**: 3 planes, data flow, control flow, dispatch flow, review flow, failure flow, authority boundaries.
- **`docs/05_DOMAIN_MODEL.md`**: Domain entities (Project, Pair, SupervisorBinding, WorkerSession, Task, TaskContract, TaskAttempt, WorkerClaim, Evidence, ReviewBundle, ReviewDecision, Blocker).
- **`docs/06_WORKFLOW_STATE_MACHINE.md`**: 13 canonical states (`DRAFT` → `READY` → `DISPATCHED` → `RUNNING` → `REPORT_READY` → `EVIDENCE_READY` → `REVIEWING` → `APPROVED` + branches `REVISION_REQUIRED`, `BLOCKED`, `FAILED`, `HUMAN_REQUIRED`, `CANCELLED`).
- **`docs/07_SECURITY_MODEL.md`**: Readonly by default, explicit project roots, path containment, threat modeling.
- **`docs/08_TASK_CONTRACT.md`**: Immutability after dispatch, complete schema specification.
- **`docs/09_WORKER_REPORT.md`**: Canonical fields, explicit disclaimer: *Claims, not verified truth*.
- **`docs/10_REVIEW_BUNDLE.md`**: Bundle composition, automated evidence correlation, drill-down reduction.
- **`docs/11_CHATGPT_TOOL_SURFACE.md`**: 12 high-level Supervisor tools (`bind_project`, `get_project_state`, `get_current_task`, `get_worker_status`, `search_project`, `read_project_file`, `get_review_bundle`, `dispatch_task`, `request_revision`, `approve_task`, `block_task`, `get_phase_status`). Explicitly rejects arbitrary shell and source-write tools.
- **`docs/12_UPSTREAM_INTEGRATION.md`**: `AOAdapter` contract and `Agy` fallback adapter.
- **`docs/13_UPSTREAM_UPDATE_POLICY.md`**: Release discovery, contract diff, compatibility verification pipeline.
- **`docs/14_FAILURE_RECOVERY.md`**: 15 concrete failure scenarios and recovery playbooks.
- **`docs/15_OBSERVABILITY.md`**: Event taxonomy and sanitized audit logging.
- **`docs/16_TEST_STRATEGY.md`**: Multi-layer testing strategy from unit and contract tests to E2E loop.
- **`docs/17_ROADMAP.md`**: Phase definitions from P00 to P07 with deliverables, entrance/exit criteria.
- **`docs/18_CURRENT_STATE.md`**: Initial status: `PHASE0_READY_FOR_EXTERNAL_AUDIT`.
- **`docs/19_OPEN_QUESTIONS.md`**: Track unresolved implementation questions explicitly marked `UNDECIDED / GIẢ ĐỊNH`.
- **`docs/20_BACKLOG_AND_NON_GOALS.md`**: Explicit V1 non-goals (no 100-task scheduler, no distributed k8s, no Redis/Kafka, no multi-agent debate).
- **`docs/21_TRACEABILITY_MATRIX.md`**: End-to-end matrix mapping Requirement ↔ Architecture ↔ ADR ↔ Module ↔ Phase ↔ Acceptance Test ↔ Source.
- **`docs/22_MODULE_PROVENANCE.md`**: Explicit provenance for every module with mandatory fields: `Existing upstream capability checked: YES/NO`, `Reason this module exists in our code`, `Why we own this`, `Forbidden responsibility`.
- **`docs/23_COMPATIBILITY_MATRIX.md`**: Tracking platforms, dependencies, and contract versions.
- **`docs/24_CHANGE_GOVERNANCE.md`**: Formal change intake pipeline and the immutable 9-level **Decision Hierarchy** (1. Confirmed User Requirement, 2. Approved ADR, 3. Canonical Architecture, 4. Requirement Specification, 5. Approved Roadmap, 6. Task Contract, 7. Source Repo Reference, 8. Worker Suggestion, 9. Chat Conversation).

### 3. Source Research & Baseline Contracts (Patches #2, #7, #8, #9)
- **`docs/sources/SOURCE_REGISTRY.md`**: Master index and technology mapping.
- **`docs/sources/REUSE_MATRIX.md`**: Comprehensive anti-reinvention map with columns: `Capability`, `Requirement`, `Source`, `Exact source area`, `Strategy`, `Our module`, `Why`, `Forbidden duplicate`.
- **`docs/sources/UPSTREAM_CONTRACT_BASELINE.md`**: Machine/audit-friendly contract baseline defining required operations for Agent Orchestrator and Antigravity CLI.
- **9 Detailed Source Dossiers** (`docs/sources/01_...` to `09_...`):
  - `01_AGENT_ORCHESTRATOR.md`: Active runtime dependency analysis (pinned Windows release v0.13.0, daemon lifecycle, ConPTY, session management, worktree isolation).
  - `02_ANTIGRAVITY_CLI.md`: Active worker interface (headless execution, stream JSON, session continuation).
  - `03_PROXIDE.md`: Design source (readonly default, containment, trust boundaries).
  - `04_MIERUKO.md`: Design source (workspace binding, slim tool philosophy, review UX).
  - `05_AIWORKHUB.md`: Design source (evidence-first review, candidate result vs proof).
  - `06_SYMPHONY.md`: Design source (workflow-as-policy, isolated run boundaries).
  - `07_CODENCER.md`: Design source (bridge-not-brain, state-not-chat, evidence vocabulary).
  - `08_AWS_CAO.md`: Design source (provider/runtime separation).
  - `09_ANTIGRAVITY_LINK.md`: Fallback only (CDP session bridge).
- **`third_party/SOURCE_VERSIONS.md`**: Strict separation between Active Upstreams (version, commit, candidate, contract test requirement) and Design Sources (reference commit, reference date, selected reason, no auto-update).
- **`third_party/NOTICES.md`**: Clear statement that no third-party source code is vendored in Phase 0.

### 4. Schemas and Validation Suite (Patch #11)
- **`docs/schemas/task-contract.schema.json`**
- **`docs/schemas/worker-report.schema.json`**
- **`docs/schemas/review-bundle.schema.json`**
- **`docs/schemas/examples/`**:
  - `task-contract.valid.json` & `task-contract.invalid.json`
  - `worker-report.valid.json` & `worker-report.invalid.json`
  - `review-bundle.valid.json` & `review-bundle.invalid.json`

### 5. ADRs, Roadmap Phases & Internal Audit (Patches #1, #16)
- **`docs/adr/`**: 10 canonical Architecture Decision Records (ADR-001 through ADR-010).
- **`docs/phases/`**: Detailed phase plans for P00 through P07.
- **`docs/proposals/README.md`**: Proposal workflow and template.
- **`docs/audits/PHASE0_INTERNAL_AUDIT.md`**: Internal self-audit by agent, strictly concluding **`PHASE0_READY_FOR_EXTERNAL_AUDIT`**.

### 6. Git & GitHub Operations (Patches #13, #14, #17)
- Pre-flight safety check: verify local `.git` status, run `git ls-remote origin`.
- Enforce: NO force push (`--force`), NO history rewrites.
- Initialize Git at `d:\TU_CODE\ai-supervisor` (preserving workspace integrity).
- Structured commits:
  1. `commit 1`: `docs: establish Phase 0 architecture baseline`
  2. `commit 2`: `docs: add source provenance and reuse research`
  3. `commit 3`: `docs: complete Phase 0 internal consistency audit`
- Push cleanly to `main` on `https://github.com/trungqwe/ai-supervisor`.

---

## Verification Plan (Patch #10)

### Automated Architectural Consistency Checks (PowerShell Script)
An automated audit script will run locally to verify:
1. **Requirement Uniqueness**: Verify all requirement IDs (`FR-xxx`, `NFR-xxx`, `SEC-xxx`, `OPS-xxx`) are unique.
2. **Requirements ↔ Architecture**: Every requirement is mapped in `docs/21_TRACEABILITY_MATRIX.md`.
3. **Architecture ↔ Module**: Every module in `docs/04_ARCHITECTURE.md` is present in `docs/22_MODULE_PROVENANCE.md`.
4. **Provenance Justification**: Every module in `docs/22_MODULE_PROVENANCE.md` has `Why we own this` and `Forbidden responsibility`.
5. **Anti-Reinvention**: Every row in `docs/sources/REUSE_MATRIX.md` specifies `Exact source area` and `Forbidden duplicate`.
6. **ADR Completeness**: All 10 ADRs exist and follow the canonical 7-section structure.
7. **Phases Gate**: Every phase document P00–P07 has defined acceptance criteria and audit exit gates.
8. **Upstream Pinning**: All active upstreams in `third_party/SOURCE_VERSIONS.md` have explicit version numbers; design sources have reference commits/dates.
9. **Tool Security**: Every tool in `docs/11_CHATGPT_TOOL_SURFACE.md` has read/write and authority classifications.
10. **State Machine Integrity**: All 13 states in `docs/06_WORKFLOW_STATE_MACHINE.md` have explicit transition rules and owners.
11. **JSON Schema Syntax**: All 3 schema files parse as 100% valid JSON.
12. **Schema Examples Validation**:
    - Valid examples pass schema checks.
    - Invalid examples fail schema checks with expected violation points.
13. **No Orphan Requirements or Modules**: Zero unlinked entities in the traceability matrix.
14. **No Application Code**: Verify that no `.go`, `.ts`, `.js`, `.py`, `.rs`, `.sql`, or framework scaffolding files exist in the repository.
15. **Audit Verdict Restraint**: Verify `docs/audits/PHASE0_INTERNAL_AUDIT.md` contains `PHASE0_READY_FOR_EXTERNAL_AUDIT` and does NOT contain premature `ARCHITECTURE_FROZEN`.

### Git & Remote Verification
- `git status` confirms a clean working tree.
- `git log --oneline` confirms the 3 structured commits.
- `git ls-remote origin` confirms remote HEAD points to the newly pushed commits on `main`.
