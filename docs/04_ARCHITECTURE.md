# 04. CANONICAL ARCHITECTURE

> **Authority**: System Architecture Blueprint (Single Canonical Reference)  
> **Status**: Verified Documentation Baseline (Post-Remediation)

---

# 1. The Three Planes Architecture

The system is strictly decomposed into three decoupled planes:

```text
┌──────────────────────────────────────────────────────────────┐
│                    1. INTELLIGENCE PLANE                     │
│                        ChatGPT Web                           │
│  Role: Architect, Planner, Auditor, Decision Maker           │
│  Properties: High reasoning, no OS/file execution capability │
└──────────────────────────────┬───────────────────────────────┘
                               │ Structured Tool Invocations (12 Tools)
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                 2. SUPERVISOR CONTROL PLANE                  │
│                        (OUR PROJECT)                         │
│  Role: State Manager, Task Formulator, Evidence Collector    │
│  Modules:                                                    │
│  - Project & Pair Registry                                   │
│  - Task Contract Manager                                     │
│  - Review Bundle Builder                                     │
│  - Policy & Scope Engine                                     │
│  - Independent Evidence Collector                            │
│  - AO Public Adapter (Anti-Corruption Layer)                 │
│  - Sanitized Audit Trail                                     │
└──────────────────────────────┬───────────────────────────────┘
                               │ Domain Operations via AOAdapter
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                 3. EXECUTION CONTROL PLANE                   │
│                 Untrivial Agent Orchestrator                 │
│  Role: Process Daemon, ConPTY Runtime, Worktree Manager      │
│  Modules:                                                    │
│  - Daemon Lifecycle (/healthz, /readyz)                      │
│  - Session Persistence & Worktree Creation (/sessions)       │
│  - Worker Process ConPTY Management (/send, /kill, /restore) │
│  - Agent Harness Adapters (Antigravity CLI)                  │
└──────────────────────────────┬───────────────────────────────┘
                               │ Headless Process / Harness Invocation
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                        PRIMARY WORKER                        │
│                   Official Antigravity CLI                   │
│  Role: Autonomous Code Synthesis, Test Execution, Build      │
└──────────────────────────────────────────────────────────────┘
```

---

# 2. Canonical Execution & Review Flows

## 2.1 Task Dispatch Flow
```mermaid
sequenceDiagram
    autonumber
    actor ChatGPT as ChatGPT Web (Supervisor)
    participant SCP as Supervisor Control Plane
    participant DB as State Store
    participant AO as Agent Orchestrator (AOAdapter)
    participant Agy as Antigravity CLI

    ChatGPT->>SCP: dispatch_task(contract_payload)
    SCP->>SCP: Validate contract against task-contract.schema.json
    SCP->>DB: Store immutable TaskContract (State: READY)
    SCP->>AO: AOAdapter.createWorkerSession(projectId, harness="antigravity")
    AO->>AO: Allocate isolated Git worktree & session
    SCP->>AO: AOAdapter.sendTask(sessionId, TaskContract)
    SCP->>DB: Transition state to DISPATCHED
    AO->>Agy: Launch worker harness in worktree
    Agy-->>AO: Worker process active
    AO-->>SCP: Worker started event / status
    SCP->>DB: Transition state to RUNNING
    SCP-->>ChatGPT: Dispatch confirmed (status: RUNNING)
```

## 2.2 Worker Completion & Independent Review Flow
```mermaid
sequenceDiagram
    autonumber
    participant Agy as Antigravity CLI
    participant AO as Agent Orchestrator (AOAdapter)
    participant SCP as Supervisor Control Plane
    participant Git as Local Git Worktree
    participant DB as State Store
    actor ChatGPT as ChatGPT Web (Supervisor)

    Agy->>AO: Worker execution completes turn
    AO-->>SCP: AOAdapter observes session completion [P01_PROOF_REQUIRED]
    SCP->>DB: Transition state to REPORT_READY
    
    rect rgb(240, 248, 255)
        Note over SCP,Git: Independent Evidence Collection (Zero Trust)
        SCP->>Git: git diff --stat base_sha..head_sha
        SCP->>Git: git diff base_sha..head_sha
        SCP->>SCP: Check touched files against allowed_scope / forbidden_scope
        SCP->>SCP: Verify test command exit codes & artifacts
    end

    SCP->>DB: Store Evidence (State: EVIDENCE_READY)
    SCP->>SCP: Build Review Bundle (Contract + Claims + Diff + Policy findings)
    SCP->>DB: Transition state to REVIEWING
    
    ChatGPT->>SCP: get_review_bundle(task_id)
    SCP-->>ChatGPT: Return compiled ReviewBundle
    
    alt Approval Decision
        ChatGPT->>SCP: approve_task(task_id, rationale)
        SCP->>DB: Transition state to APPROVED
        Note over SCP: Decision recorded. NO automatic git merge or push in V1.
    else Revision Required Decision
        ChatGPT->>SCP: request_revision(task_id, feedback, required_fixes)
        SCP->>DB: Transition state to REVISION_REQUIRED
        SCP->>AO: AOAdapter.sendTask(sessionId, revision_instructions)
    end
```

---

# 3. Adapter Boundaries & Integration Realism

### 3.1 AOAdapter Boundary
All execution interactions flow strictly through `AOAdapter`. The adapter encapsulates:
- Daemon health checks (`GET /healthz`, `GET /readyz`);
- Project registration (`POST /api/v1/projects`);
- Session creation (`POST /api/v1/sessions`);
- Task transmission (`POST /api/v1/sessions/{id}/send`);
- Process control (`POST /api/v1/sessions/{id}/kill`, `POST /api/v1/sessions/{id}/restore`).
Detailed HTTP mappings reside exclusively in `docs/sources/UPSTREAM_CONTRACT_BASELINE.md`.

### 3.2 AO ↔ Agy Integration Realism & P01 Proof
- **Known Upstream Finding**: AO `v0.13.0` invokes Agy interactively using `--prompt-interactive`. Official Agy separately supports headless print mode (`--print`, `--output-format stream-json`, `--json-schema`).
- **Integration Boundary**: The exact mechanism for extracting normalized `WorkerReport` data from AO session runs is designated `P01_PROOF_REQUIRED`. The Supervisor treats worker report generation as a normalized contract requirement, not an unverified upstream native feature.

---

# 4. Authority Boundaries & Readonly Supervisor Policy

> [!CRITICAL]
> 1. **Zero Direct Code Mutation by Supervisor**: ChatGPT and the Supervisor Control Plane are strictly readonly regarding application source files.
> 2. **No Automatic Merge on Approval**: `approve_task` formally records review approval and updates task state to `APPROVED`. It does **NOT** automatically merge branches, commit to main, or push to remotes. Source promotion remains a human/explicit workflow.
> 3. **AO internal database is NOT an integration API**: We never read or write directly to AO SQLite stores.
> 4. **Code truth belongs exclusively to Git**: Commit SHAs, diffs, and worktree states are authoritative.
