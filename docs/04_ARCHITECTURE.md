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
        Note over SCP,DB: Saga Stage 1: Bound Dispatch + Workspace Binding Tx (P04A & ADR-018 D1)
        SCP->>SCP: Coordinator acquires live WorkspaceBindingLease via WorkspaceBindingAuthority
        SCP->>DB: Atomically persist TaskAttempt, dispatch_operations (DISPATCH_BOUND), attempt_workspace_bindings (ACTIVE), and READY -> DISPATCHED
    end

    rect rgb(245, 245, 255)
        Note over SCP,AO: Pre-Send Admissibility & Live Lease Revalidation (ADR-016 D7 & ADR-018 D1)
        SCP->>AO: AOAdapter.getWorkerStatus(sessionId)
        AO-->>SCP: Return AOWorkerStatus (status in 'idle', 'waiting_input' & matching terminal_generation)
        SCP->>SCP: Live lease.Revalidate() confirms open handle matches DB binding; verify 14 pre-send guards
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
        SCP->>SCP: Verify clean worktree (git status --porcelain=v1 -z --untracked-files=all) & clean index (git diff-index --quiet HEAD --)
        SCP->>DB: Store WorkerClaim with reported_head_sha (Transition state to REPORT_READY)
        rect rgb(240, 248, 255)
            Note over SCP,Git: Independent Evidence Collection (Zero Trust, P04B & P04C)
            SCP->>Git: Pure in-memory Git collector: git diff base_sha..actual_head_sha (P04B)
            SCP->>SCP: Check touched files against allowed_scope / forbidden_scope
            SCP->>Runner: Execute host-owned verification profiles in isolated Windows AppContainers (P04C)
            Runner-->>SCP: Capture exit code, stdout/stderr (10MB capture limit), test artifacts
        end
        SCP->>DB: Store EvidenceSet & ReviewArtifacts (Schema v9, Transition state to EVIDENCE_READY)
        SCP->>SCP: Build ReviewBundle (RFC 8785 JCS, dual head SHAs, compilation_latency_ms)
        SCP->>DB: Store ReviewBundle (Schema v9, Transition state to REVIEWING)

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

## 2.3 ADR-016 addendum: restore và dispatch safety

Pair restore dùng v4 durable authorization/operation. Tx A consume one-shot authorization và commit RESTORE_REQUESTED/audit trước AO effect; không network trong SQLite transaction; chỉ trusted host principal enable restore. Admission chặn mọi open attempt, SEND_REQUESTED/resolution NULL và current_attempt bất nhất; closed historical DISPATCH_BOUND/SEND_CONFIRMED sạch không khóa mãi. Tx B HTTP 200 valid xác nhận cùng WorkerSession; ambiguous outcome giữ Pair lock/quarantine, không retry. Bound snapshot bất biến. Unknown send ghi DELIVERY_OUTCOME_UNKNOWN terminal cho dispatch operation; D6 clearance tách biệt, theo từng lineage. Pre-send protocol hold không bị timeout/GET tự hạ. Xem ADR-016 addendum §§2–8.

## 2.4 Execution budget và trusted maintenance (ADR-016 addendum)

`dispatch_operations.confirmed_at` là origin của **send-confirmation-based execution budget**, không là actual execution start. Tx xác nhận `/send` HTTP 200 ghi cùng lúc `SEND_CONFIRMED`, budget/deadline/policy bất biến và audit; schema v5 thuộc Supervisor Store. Startup `Run` chỉ phân loại, không phát timeout effect. Runtime monitor sau startup dùng shared host admission và stop coordinator 3C: Tx R tái sử dụng `ReserveStopOperation` để commit intent, hai quarantine và audit nguyên tử; chỉ caller thắng còn permit sống được `/kill` một lần. GET mới không bảo đảm AO đứng yên đến effect. Nếu caller bỏ quyền, giữ `STOP_REQUESTED/IN_FLIGHT`, attempt mở và quarantine; startup sau exclusive drain/join hoặc human reconciliation phân loại, không replay. Legacy thiếu budget giữ normal admission/serve đóng; trusted maintenance scope độc quyền dùng authority riêng cho historical binding hoặc manual stop của exact open `RUNNING`, không ép `DISPATCHED` sang `RUNNING`; release maintenance không mở admission, phải Run lại. Xem [addendum execution budget](adr/ADR-016-ADDENDUM-send-confirmation-execution-budget.md) §§3–6. Contract Revision 3 đã RELEASED; implementation 3D EXTERNAL_AUDIT_APPROVED tại `7513f1b9f39fa459be15e0abc836c6258a610d8b`, findings `3D-R1-001..005` và `3D-R2-001..002` đã CLOSED ở library scope, code đã MERGED tại `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`. Dependency host quiescence/principal và startup wiring tiếp tục là handoff dependency runtime (`HOST_QUIESCENCE_INTEGRATION=OPEN`, `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`, `AUTOMATIC_RESTORE=DISABLED`).

## 2.5 Host Quiescence và Daemon Lifecycle (ADR-017)

ADR-017 (Accepted) thiết lập kiến trúc host bootstrap và quản lý vòng đời daemon trên Windows:
- **ProcessOwnerLease**: Khóa độc quyền toàn máy sử dụng Windows Exclusive Sidecar Lock File Handle (`<canonical_db_path>.owner.lock`, share mode 0, `OPEN_ALWAYS`, không kế thừa handle) nắm giữ liên tục từ trước khi mở DB đến sau shutdown drain.
- **Host Pinned DB Handle**: Host nắm giữ handle `hPinnedDB` trực tiếp ở tầng OS (không cấp `FILE_SHARE_DELETE`), kernel Windows bảo đảm không thể xóa hoặc đổi tên file trong suốt thời gian Store hoạt động.
- **Hai Đường Đi Khởi Tạo DB**:
  1. *DB Hiện Hữu*: Mở handle pin pre-open -> kiểm tra `nNumberOfLinks == 1` & `FILE_ID_INFO` -> acquire lock -> gọi `store.Open` với DOS path đã xác thực.
  2. *DB Mới*: Chuẩn hóa thư mục cha -> acquire lock -> tạo file bằng Win32 `CREATE_NEW` và pin trước `store.Open` -> gọi `store.Open` khởi tạo file 0-byte -> kiểm tra volume cha.
- **Ranh Giới Tích Hợp Store & 4 Invariants**: Host chuyển đổi `\\?\<Drive>:\...` thành `<Drive>:\...` khi đã chứng minh local DOS volume (UNC/device fail-closed); gọi `store.Open` thực tế; thay thế việc đọc `PRAGMA database_list` bằng 4 Invariants.
- **Takeover Hợp Tác & Shutdown**: Named Pipe xác thực caller token SID; shutdown đóng listener -> drain -> `Store.Close()` -> đóng `hPinnedDB` -> đóng lock cuối cùng.

---

# 3. Adapter Boundaries & Integration Realism

## 3.1 AOAdapter Boundary
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

---

# 7. Evidence & Review Engine Architecture (ADR-018)

> **Authority**: Formally governed by accepted [ADR-018](adr/ADR-018-evidence-review-and-verification-isolation.md) (Level 2 Approved ADR), [PROPOSAL-P04-001](proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md) Revision 22, and approved baseline [PROPOSAL-P04-002](proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md) Revision 9.

## 7.1 Workspace Binding Authority, Live Lease vs Snapshot, Coordinator Effect Gate, and Pre-Send Verification Guards

### 7.1.1 Strict Capability vs Data Separation (P04-ARCH-R21-001)
To prevent worktree substitution, path-alias aliasing, and time-of-check to time-of-use (TOCTOU) rename races on Windows, the system strictly separates immutable persistence data from live OS-level capabilities:
1. **`WorkspaceBindingSnapshot`**:
   - An immutable data structure containing database lineage, canonical volume serial, 128-bit `FILE_ID_INFO`, canonical absolute path, and binding epoch timestamps.
   - Used strictly for database lineage verification, SQLite trigger enforcement, and CAS comparisons.
   - The Store layer receives only this snapshot and evaluates database constraints. Store NEVER claims to prove or hold a live OS handle lease.
2. **`WorkspaceBindingLease`**:
   - An ephemeral, live OS-level capability holding open Win32 directory handles (`CreateFileW` with `FILE_READ_ATTRIBUTES | FILE_FLAG_BACKUP_SEMANTICS`, omitting `FILE_SHARE_DELETE`).
   - Holding this handle directly prevents external processes from renaming, moving, or deleting the physical worktree directory during critical lifecycle windows.
   - Encapsulates `Revalidate()` which re-queries the held handle's `FILE_ID_INFO` and volume serial number and verifies byte-for-byte identity against durable binding records.

### 7.1.2 Single Coordinator Effect Gate (`Coordinator.Dispatch`)
The system establishes a single coordinator effect gate (`Coordinator.Dispatch` in `internal/dispatch/coordinator.go`) that exclusively owns the AO `DispatchTaskContract(ctx, sessionID, message)` boundary.
The effect gate receives the candidate, acquires `WorkspaceBindingLease`, retains the SAME lease instance in its call frame, and executes the exact 9-step sequence:
1. Acquire live lease via `WorkspaceBindingAuthority.Acquire(ctx, candidate)`.
2. Extract immutable snapshot via `lease.Snapshot()`.
3. Commit `PrepareBoundDispatch` with snapshot (atomic bound dispatch + workspace binding transaction).
4. Observe fresh `GetWorkerStatus` from AO (must be `idle` or `waiting_input`, not terminated).
5. Revalidate live lease via `lease.Revalidate()`, checking held handles against durable binding read back from DB.
6. Commit `RecordSendRequested` using pure DB guards (Store checks DB row, lineage, CAS, snapshot equality, zero active holds).
7. Call AO `/send` (`AO.DispatchTaskContract(ctx, sessionID, message)`) within the same guarded scope while lease is still held open.
8. Commit confirmed / containment (`RecordSendConfirmed` or `containAmbiguousSend`).
9. Close lease in `defer` after effect boundary.

Zero public API paths accepting only a snapshot can invoke `/send`. Recovery must acquire a fresh lease via `WorkspaceBindingAuthority.Acquire`.

### 7.1.3 Fourteen Mandatory Pre-Send Verification Guards
Before transitioning dispatch operation to `SEND_REQUESTED` or issuing `/send`, all 14 mandatory guards must be satisfied:
1. `dispatch_operations.stage = 'DISPATCH_BOUND'`
2. `dispatch_operations.resolution_state IS NULL`
3. `task_attempts.recovery_disposition IS NULL`
4. `tasks.state = 'DISPATCHED'`
5. Exact current open attempt (`current_attempt = attempt_number`, `ended_at IS NULL`)
6. Exact task/contract/Pair/session/generation lineage match
7. Current `WorkerSession` matches and `quarantine_state = 'CLEAN'`
8. Current `TaskAttempt` `quarantine_state = 'CLEAN'`
9. Zero unresolved `pair_restore_operations` (`NOT EXISTS (SELECT 1 FROM pair_restore_operations WHERE pair_id = ? AND resolution_state <> 'RESTORE_RESOLVED')`)
10. Zero unresolved `pair_provisioning_operations` (`NOT EXISTS (SELECT 1 FROM pair_provisioning_operations WHERE pair_id = ? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED'))`)
11. Fresh AO observation is `idle` or `waiting_input`, not terminated
12. Exactly one `attempt_workspace_bindings` row with `binding_state = 'ACTIVE'`
13. Live `WorkspaceBindingLease.Revalidate()` passes and matches database row
14. Zero `ACTIVE` `review_integrity_holds` for the attempt (`NOT EXISTS (SELECT 1 FROM review_integrity_holds WHERE attempt_id = ? AND hold_state = 'ACTIVE')`)

### 7.1.4 Governed Terminal Resolution
If any binding guard fails (guards 12–14 or binding lineage/identity mismatch):
- Strictly zero write to `SEND_REQUESTED`; do NOT call `/send`.
- Task remains `DISPATCHED`, operation remains `DISPATCH_BOUND`, attempt remains open.
- Determine `attempted_reason` from four standardized literals: `WORKSPACE_BINDING_MISSING`, `WORKSPACE_BINDING_LINEAGE_MISMATCH`, `WORKSPACE_BINDING_NOT_ACTIVE`, `WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`.
- Execute separate atomic diagnostic transaction (Variant B: `WORKSPACE_BINDING_GUARD`):
  1. Append rejection audit event `REVIEW_INTEGRITY_CONFLICT` with Variant B fields (`conflict_source = 'WORKSPACE_BINDING_GUARD'`, `dispatch_operation_id`, `conflict_type = 'LINEAGE_MISMATCH'`, `attempted_reason = '<literal>'`, `sanitized_input_fingerprint`, `diagnostic_fingerprint`, `actor_role = 'SUPERVISOR'`, and `colliding_event_id` strictly ABSENT).
  2. Insert `ACTIVE` hold into `review_integrity_holds` (`hold_reason = 'INVARIANT_MISMATCH'`).
  3. If `ACTIVE` binding exists but lineage/identity is invalid, CAS binding to `INVALIDATED` (`released_at_epoch_ms = now`).
- Governed Terminal Transition (D12): To recover, a verified operator must terminalize the attempt via atomic D12 transition:
  * `tasks`: CAS `state = 'FAILED'` where `task_id = ? AND state = 'DISPATCHED'`.
  * `task_attempts`: `ended_at = now`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'` where `attempt_id = ? AND ended_at IS NULL`.
  * Append `TASK_STATE_TRANSITION` audit event in same transaction.
- Only after the attempt is terminalized may the operator resolve the integrity hold in a separate follow-up transaction (`review_integrity_holds.hold_state: ACTIVE -> RESOLVED`, appending `REVIEW_INTEGRITY_HOLD_RESOLVED`). Fails closed without `VERIFIED_OPERATOR_PRINCIPAL`.

### 7.1.5 NTFS Directory Substitution Probes
Hard links on directories are unsupported by NTFS and are completely eradicated from the system architecture. Testing directory substitution is formalized via a valid NTFS probe matrix:
- Junction points (`mklink /J`)
- Directory symlinks (`mklink /D`)
- Virtual drive mapping (`subst`)
- Directory rename during open handle hold (verified rejected by Windows kernel via sharing violation)
- Path-alias and short 8.3 filename resolution probes

---

## 7.2 Git Evidence Authority, Pure In-Memory Collector (P04B), and Clean-Worktree Intake Verification

### 7.2.1 Pure In-Memory Collector Architecture
Subtask P04B implements the Git evidence collector as a pure in-memory component:
- **Zero SQLite Mutations**: P04B executes ZERO database writes, ZERO audit event appends, and ZERO hold mutations.
- **Read-Only Inspection**: Inspects the physical worktree and extracted immutable snapshot strictly via safe, bounded read-only Git commands.
- **Ten-Command Allowlist**:
  1. `git rev-parse --verify <sha>`
  2. `git diff-index --quiet HEAD --`
  3. `git status --porcelain=v1 -z --untracked-files=all`
  4. `git diff --stat <base>..<head>`
  5. `git diff --raw <base>..<head>`
  6. `git diff -p <base>..<head>`
  7. `git ls-tree -rz --full-tree <head>`
  8. `git cat-file --batch`
  9. `git merge-base <base> <head>`
  10. `git log -n 1 --format=%H <head>`

### 7.2.2 Clean-Worktree Intake Policy (FR-008)
Upon worker report submission:
1. Worktree must be verified clean: `git status --porcelain=v1 -z --untracked-files=all` must return zero entries.
2. Index must be verified clean: `git diff-index --quiet HEAD --` (with `GIT_OPTIONAL_LOCKS=0`) must exit 0.
3. If dirty staged, unstaged, or untracked state is detected:
   - Transaction A rolls back; TaskState remains strictly in `RUNNING` (zero blanket transitions to `BLOCKED`).
   - A separate diagnostic transaction appends `EVIDENCE_COLLECTION_FAILED` to `audit_events` and inserts an `ACTIVE` hold (`DIRTY_WORKTREE_DETECTED`) into `review_integrity_holds`.
   - Worktree state is never modified (no `git stash`, `git clean`, or `git checkout`).
4. Dual Head SHA capture:
   - `worker_claims.reported_head_sha` stores the exact reported value (`CHECK (LENGTH(reported_head_sha) BETWEEN 7 AND 40 AND NOT (reported_head_sha GLOB '*[^0-9a-f]*'))`).
   - The verified `actual_head_sha` belongs strictly to Evidence (`evidence_sets.git_evidence_json`), never in `worker_claims`.

---

## 7.3 Execution Isolation: Windows AppContainer & Job Objects (P04C)

### 7.3.1 Execution Boundary & Process Containment
Subtask P04C implements the verification test execution runner as a pure in-memory component with zero SQLite writes. Test commands run within an isolated Windows security boundary:
1. **Windows AppContainer Isolation**:
   - Created via Win32 `CreateProcessW` with `STARTUPINFOEXW` and `EXTENDED_STARTUPINFO_PRESENT`.
   - Security capabilities configured via `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES`.
   - Network restriction SID applied (no inbound, no internet access, loopback restricted).
2. **Explicit Handle Allowlist (`PROC_THREAD_ATTRIBUTE_HANDLE_LIST`)**:
   - Handle inheritance is strictly confined to designated stdio pipes (`hStdInput`, `hStdOutput`, `hStdError`).
   - All host handles (including database, locks, and parent pipes) are strictly excluded from child process inheritance.
3. **Atomic Job Object Assignment**:
   - Child process is assigned atomically at creation via `PROC_THREAD_ATTRIBUTE_JOB_LIST`.
   - Configured with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` ensuring guaranteed tree termination when the job handle closes.
4. **Authoritative Process-Death Proof**:
   - Verification runner waits on process/job object handle via `WaitForSingleObject` / `GetExitCodeProcess`.
   - Requires authoritative proof of process termination before collecting stream outputs or advancing pipeline stage.

### 7.3.2 Stream Capture Separation
Artifact capture enforces strict mathematical boundaries:
- **Capture Limit**: 10 MB per stream written to Content-Addressed Storage.
- **Hard Safety Limit**: 50 MB cumulative stream size; exceeding this limit immediately triggers hard process termination (`HARD_LIMIT_TERMINATED`).

---

## 7.4 Content-Addressed Store, Dual Head SHAs, ReviewBundle Latency Semantics, and Inert AO Adapter Harness

### 7.4.1 Content-Addressed Storage Layout
All review artifacts (test logs, git diffs, worker output) are stored in an append-only, content-addressed filesystem hierarchy:
- Path structure: `artifacts/<first-two-hex>/<captured_sha256>`.
- Exact equality constraint: `canonical_relative_path = 'artifacts/' || substr(captured_sha256, 1, 2) || '/' || captured_sha256`.
- Atomic write-through via staging file and atomic rename (`MOVEFILE_REPLACE_EXISTING`).

### 7.4.2 ReviewBundle Latency Semantics (PROPOSAL-P04-002 Revision 9)
ReviewBundle synthesis strictly separates the assembly diagnostic interval from canonical NFR-008:
1. **Two-Interval Pipeline**:
   - *Interval 1 (Evidence Acquisition Window)*: Governed by task contract verification budgets. Bounds worker report intake, Git collection, AppContainer test runs, and artifact persistence ending at Transaction B commit ($T_0 = \text{evidence\_finalized\_at\_epoch\_ms}$).
   - *Interval 2 (ReviewBundle Compilation Window)*: Strictly measures the assembly diagnostic interval:
     $$\text{compilation\_latency\_ms} = \text{bundle\_assembled\_at\_epoch\_ms} - \text{evidence\_finalized\_at\_epoch\_ms}$$
2. **2 × 2 Provenance × Threshold Matrix**:
   - `latency_measurement_status` represents execution provenance:
     * `'MEASURED_IN_PROCESS'`: continuous in-process daemon execution.
     * `'RECOVERED_AFTER_RESTART'`: P04D startup recovery path.
   - Decoupled from the 3,000 ms threshold: high system load never converts provenance to `RECOVERED_AFTER_RESTART`.
   - Exceeding 3,000 ms emits an assembly diagnostic without blocking Transaction C, altering TaskState, or deadlocking persistence.
   - `nfr008_compliance_status` remains held strictly as `'UNVERIFIED'`.
3. **Decoupled Monotonic Telemetry**:
   - Commit duration telemetry (`time.Since(commitStart)`) is captured best-effort in application memory outside the database and audit chain.

### 7.4.3 Inert Fake AO Adapter Harness
For automated verification pipeline testing:
- An in-process, synthetic session harness satisfies `IAOAdapter` without spawning live worker processes or requiring external daemon network ports.
- Returns deterministic JSON fixtures matching AO v0.13.0 wire schemas.
- Live AO integration is maintained in a separate unverified evidence track; `AUTOMATIC_RESTORE = DISABLED` remains permanently fail-closed.

---

## 7.5 Model 1 Persistence Ownership, Schema v6/v9, Audit Event Derivation, and Variant Discrimination

### 7.5.1 Persistence Ownership Discipline & Contract Sequencing
Persistence authority is partitioned under strict single-ownership rules:
1. **Schema v6 (Owned by Subtask P04A)**:
   - `attempt_workspace_bindings` (trigger requiring `stage = 'DISPATCH_BOUND'`).
   - `worker_claims` (composite task/contract/attempt lineage triggers).
   - `review_integrity_holds` (partial unique index `idx_review_integrity_holds_active_dedup`, lineage triggers).
2. **Schema v9 (Owned by Subtask P04D)**:
   - `task_verification_leases` (linear lease chain, `predecessor_lease_id`, token monotonicity).
   - `evidence_sets` (composite foreign keys and lineage guard).
   - `review_artifacts` (exact path and stream state constraints).
   - `review_bundles` (assembly diagnostic latency properties, `nfr008_compliance_status = 'UNVERIFIED'`).
3. **Pure In-Memory Subtasks (P04B & P04C)**:
   - ZERO migrations, ZERO SQLite writes, ZERO audit event appends, ZERO hold mutations.
4. **Execution Sequencing**:
   - Subtasks must execute sequentially: `P04A -> P04B -> P04C -> P04D`.
   - Subtask P04A is the first releaseable implementation work package.
   - P04 runtime admission remains closed until Subtask P04D completes startup recovery.

### 7.5.2 RFC 8785 JCS Derivation & Event Descriptors
Audit event IDs and integrity hold IDs are derived deterministically using RFC 8785 JSON Canonicalization Scheme (JCS). String concatenation with delimiters is strictly prohibited:
- **Descriptor A (`hold_identity_descriptor`)**: Derives `hold_id = "hold-" + SHA256(RFC8785_JCS(descriptor_a))`.
- **Descriptor B (`rejection_event_identity_descriptor`)**: Incorporates standard discriminator `conflict_source` and exact variant fields:
  * *Variant A (`AUDIT_EVENT_ID_COLLISION`)*: requires `colliding_event_id`, `attempted_event_type`, mismatch fields; zero binding mutation.
  * *Variant B (`WORKSPACE_BINDING_GUARD`)*: requires `dispatch_operation_id`, 4 `attempted_reason` literals; `colliding_event_id` strictly absent; CAS invalidates binding.
- **Descriptor C (`resolution_event_identity_descriptor`)**: Derives `resolution_event_id = SHA256(RFC8785_JCS(descriptor_c))`.
- **Pipeline Isolation**: P04D pipeline conflicts never CAS mutate workspace bindings.
- **Zero Orphan Guarantee**: Audit events are committed within the same database transaction as entity state changes, ensuring zero orphan records on rolled-back CAS races.
