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
                               │ Managed Agy Harness Invocation
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

    Note over SCP,AO: Pair Session Provisioning Decoupled (ADR-016 D2/D3: CREATE_NEW_WORKER_SESSION_ALLOWED_IFF requires COUNT(*)==0; TERMINATED+CLEAN does not authorize second spawn)
    opt Pair WorkerSession Absent (CREATE_NEW_WORKER_SESSION_ALLOWED_IFF: COUNT(*) == 0)
        SCP->>DB: Record pair_provisioning_operations (PROVISION_REQUESTED)
        SCP->>AO: AOAdapter.createWorkerSession(projectId, harness="antigravity")
        AO-->>SCP: Return session details (HTTP 201 Created)
        SCP->>DB: Store WorkerSession & update provisioning (PROVISION_CONFIRMED)
    end

    ChatGPT->>SCP: dispatch_task(contract_payload)
    SCP->>SCP: Validate contract against task-contract.schema.json (contract_id, task_id, revision_number)
    SCP->>DB: Store immutable TaskContract revision (State: READY)

    rect rgb(245, 245, 255)
        Note over SCP,DB: Saga Stage 1: Atomic Pre-Dispatch Persistence (P02 Core & ADR-016 D1/D4)
        SCP->>DB: Allocate TaskAttempt (session_id, terminal_generation, quarantine_state='CLEAN')
        SCP->>DB: Atomically persist READY -> DISPATCHED and dispatch_operations (DISPATCH_BOUND)
    end

    rect rgb(245, 245, 255)
        Note over SCP,AO: Pre-Send Admissibility Check (ADR-016 D7: between DISPATCH_BOUND & SEND_REQUESTED)
        SCP->>AO: AOAdapter.getWorkerStatus(sessionId)
        AO-->>SCP: Return AOWorkerStatus (status in 'idle', 'waiting_input' & matching terminal_generation)
    end

    rect rgb(255, 250, 240)
        Note over SCP,DB: Saga Stage 2: Send Requested Persistence (ADR-016 D4)
        SCP->>DB: Update dispatch_operations (SEND_REQUESTED)
    end

    SCP->>AO: AOAdapter.sendTask(sessionId, TaskContract + Attempt metadata)

    alt Upstream Write Acceptance (HTTP 200 OK)
        AO-->>SCP: Return write confirmation (HTTP 200 OK)
        rect rgb(240, 255, 240)
            Note over SCP,DB: Saga Stage 3: Send Confirmed Persistence
            SCP->>DB: Update dispatch_operations (SEND_CONFIRMED)
        end
        AO->>Agy: Launch worker turn in existing worktree
        Agy-->>AO: Worker process active
        SCP->>AO: AOAdapter.getWorkerStatus(sessionId)
        AO-->>SCP: Observe status: active
        SCP->>DB: Transition state to RUNNING
        SCP-->>ChatGPT: Dispatch confirmed (status: RUNNING)
    else Unconfirmed Send / Network Failure / Crash (Fail-Closed Quarantine, ADR-016 D5/D6)
        AO-->>SCP: Error / Timeout / Ambiguous Delivery
        SCP->>DB: Transition DISPATCHED -> FAILED (ended_at=now, recovery_disposition='UNCERTAIN_DELIVERY_CRASH')
        SCP->>DB: Impose Double-Gated Quarantine (worker_sessions & task_attempts = 'QUARANTINED')
        SCP->>DB: Escalate FAILED -> HUMAN_REQUIRED
        SCP-->>ChatGPT: Dispatch failed (status: FAILED / HUMAN_REQUIRED, quarantined)
    end
```

## 2.2 Worker Completion & Independent Review Flow
```mermaid
sequenceDiagram
    autonumber
    participant Agy as Antigravity CLI
    participant AO as Agent Orchestrator (AOAdapter)
    participant SCP as Supervisor Control Plane
    participant Runner as Trusted Verification Runner
    participant Git as Local Git Worktree
    participant DB as State Store
    actor ChatGPT as ChatGPT Web (Supervisor)

    Agy->>AO: Stop hook triggers; session activity transitions to IDLE
    AO-->>SCP: AOAdapter observes session IDLE
    SCP->>AO: Bounded fetch: GET /workspace/file?path=.supervisor/reports/<task_id>/<attempt_id>.json
    AO-->>SCP: Return report artifact content
    SCP->>SCP: JSON parse + worker-report.schema.json validation
    SCP->>SCP: Validate report.task_id == task_id AND report.attempt_id == attempt_id

    alt Valid Report & Matching Identity (Success Path)
        SCP->>DB: Store WorkerClaim (Transition state to REPORT_READY)
        rect rgb(240, 248, 255)
            Note over SCP,Git: Independent Evidence Collection (Zero Trust)
            SCP->>Git: git diff --stat base_sha..head_sha
            SCP->>Git: git diff base_sha..head_sha
            SCP->>SCP: Check touched files against allowed_scope / forbidden_scope
            SCP->>Runner: Execute host-owned verification profiles (verification_requests per ADR-013)
            Runner-->>SCP: Capture exit code, stdout/stderr, test artifacts
        end
        SCP->>DB: Store Evidence (Transition state to EVIDENCE_READY)
        SCP->>SCP: Build ReviewBundle (Contract + Claims + Diff + Test Evidence + Policy findings)
        SCP->>DB: Transition state to REVIEWING

        ChatGPT->>SCP: get_review_bundle(task_id, attempt_id)
        SCP-->>ChatGPT: Return compiled ReviewBundle

        alt Approval Decision
            ChatGPT->>SCP: approve_task(task_id, attempt_id, rationale)
            SCP->>DB: Record ReviewDecision & transition state to APPROVED
            Note over SCP: Decision recorded. NO automatic git merge or push in V1.
        else Revision Required Decision
            ChatGPT->>SCP: request_revision(task_id, attempt_id, feedback, required_fixes)
            SCP->>DB: Record ReviewDecision & transition state to REVISION_REQUIRED
            Note over SCP: Revision loop creates new TaskContract revision before re-dispatch (ADR-012).
        else Architectural Blocker Decision
            ChatGPT->>SCP: block_task(task_id, attempt_id, blocker_reason)
            SCP->>DB: Record ReviewDecision & transition state to BLOCKED
            Note over SCP: Escalates to HUMAN_REQUIRED.
        end
    else Missing / Invalid / Mismatched Report (Failure Path)
        SCP->>DB: Transition state to FAILED (failure_reason: REPORT_MISSING / REPORT_INVALID / REPORT_IDENTITY_MISMATCH)
        Note over SCP: Handoff aborted. Never reaches REPORT_READY. Review flow terminates.
    end
```

---

# 3. Adapter Boundaries & Integration Realism

### 3.1 AOAdapter Boundary
All execution interactions flow strictly through `AOAdapter`. The adapter encapsulates:
- Daemon health checks (`GET /healthz`, `GET /readyz`);
- Harness inventory & readiness probes (`GET /api/v1/agents`, `GET /api/v1/agents/readiness`);
- Public API contract retrieval (`GET /api/v1/openapi.yaml`, compatibility signal only);
- Project registration (`POST /api/v1/projects`);
- Session creation (`POST /api/v1/sessions`);
- Task transmission (`POST /api/v1/sessions/{id}/send`, strict whitelist pre-send enforcement);
- Process control:
  - Terminate session: `POST /api/v1/sessions/{id}/kill` (wire request carries session identity only; stop purpose, generation precheck, and confirmation deadline reside in Supervisor-owned `stop_operations` metadata);
  - Restore terminated session: `POST /api/v1/sessions/{id}/restore` (`POST /api/v1/sessions/{sessionId}/restore`, `operationId: restoreSession`, restores a terminated session under `ResumeWorker`, distinguished from `/resume-agent`);
- Session observation (`GET /api/v1/sessions/{id}`, authoritative snapshot);
- Raw workspace file read transport (`GET /api/v1/sessions/{id}/workspace/file?path={relPath}`).
Detailed HTTP mappings reside in `docs/sources/UPSTREAM_CONTRACT_BASELINE.md` and `docs/12_UPSTREAM_INTEGRATION.md`.

### 3.2 AO ↔ Agy Integration Realism & P01 Proof
- **Known Upstream Finding**: AO `v0.13.0` invokes Agy interactively using `--prompt-interactive`. Official Agy separately supports headless print mode (`--print`, `--output-format stream-json`, `--json-schema`).
- **Integration Boundary**: Formally resolved by **ADR-011** (`docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`) and **ADR-012** (`docs/adr/ADR-012-task-contract-revision-and-attempt-binding.md`). In pinned AO `v0.13.0`, turn completion is signaled by the Agy Stop hook transitioning session activity to `IDLE` (`PROCESS_ALIVE != TURN_RUNNING`; `AO_IDLE != REPORT_READY`). Attempt-scoped structured reports (`.supervisor/reports/<task_id>/<attempt_id>.json`) are delivered via AO's public workspace file API (`GET /api/v1/sessions/{id}/workspace/file?path=...`) and normalized by the Supervisor under zero-trust verification rules without requiring upstream code patches.

---

# 4. Authority Boundaries & Readonly Supervisor Policy

> [!CRITICAL]
> 1. **Zero Direct Code Mutation by Supervisor**: ChatGPT and the Supervisor Control Plane are strictly readonly regarding application source files.
> 2. **No Automatic Merge on Approval**: `approve_task` formally records review approval and updates task state to `APPROVED`. It does **NOT** automatically merge branches, commit to main, or push to remotes. Source promotion remains a human/explicit workflow.
> 3. **AO internal database is NOT an integration API**: We never read or write directly to AO SQLite stores.
> 4. **Code truth belongs exclusively to Git**: Commit SHAs, diffs, and worktree states are authoritative.
5. **Trusted Verification Runner Boundary**: ChatGPT is never granted arbitrary shell or command execution primitives. Supervisor independent test verification runs exclusively through a constrained, allowlisted verification runner executing host-owned verification profiles (`verification_requests` per ADR-013) within execution isolation boundaries, capturing exit codes and outputs as independent evidence.
