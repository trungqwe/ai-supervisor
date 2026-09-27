# 14. FAILURE RECOVERY PLAYBOOKS

> **Focus**: Deterministic Recovery from 15 Concrete System Failure Scenarios
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

# 1. Recovery Matrix & Owners

| Scenario ID | Failure Scenario | Impact | Recovery Owner | Playbook / Action |
|---|---|---|---|---|
| **REC-001** | Supervisor Daemon Restart | Active pair binding disconnected; memory state lost. | Supervisor Core | Reload state from local State Store; reconnect to existing Pair via `session_token`. |
| **REC-002** | AO Daemon Crash | Worker execution interrupted; ConPTY pipes closed. | AOAdapter | Reconnect to AO daemon; query session state via `GetWorkerStatus`; operator-authorized restore only after durable intent and verified host principal via pinned wire route `POST /api/v1/sessions/{sessionId}/restore` (`ResumeWorker`). If restore outcome is ambiguous, retain Pair lock and double-gated quarantine containment (`worker_sessions.quarantine_state = 'QUARANTINED'` and `task_attempts.quarantine_state = 'QUARANTINED'`). |
| **REC-003** | Antigravity CLI Crash | Process exits with non-zero code before report generation. | Supervisor Core | Mark attempt `FAILED` (`failure_reason = PROCESS_CRASH`); query Git for partial changes; decide whether to retry or escalate to `HUMAN_REQUIRED`. |
| **REC-004** | Worker Execution Timeout | Worker runs longer than configured bounded execution timeout policy. | Supervisor Core | Allocate `stop_operations` row (`purpose = 'RUNNING_ATTEMPT_STOP'`); capture target generation for Supervisor-side precheck; persist restart-stable `confirmation_deadline_at`; issue `stopWorker` (`POST /api/v1/sessions/{sessionId}/kill` carrying session identity only). If confirmed stopped within deadline under `WORKER_STOPPED_ALLOWED_IFF`, record `recovery_disposition = 'WORKER_STOPPED'` and transition `RUNNING -> FAILED` (`failure_reason = TIMEOUT` in audit log). If stop call or confirmation times out, fail closed to `STOP_CONFIRMATION_TIMEOUT`, retain quarantine, and enforce `STOP_REISSUE_REQUIRES_HUMAN`. |
| **REC-005** | Dirty Worktree Detected | Worktree contains uncommitted files or unsafe working tree state before, during pre-send, or during active intake. | Supervisor Core | Fail closed with zero mutating Git commands. **Case A (Pre-PrepareDispatch)**: reject dispatch; Task remains `READY`; zero `TaskAttempt` allocated; emit audit evidence `WORKTREE_DIRTY`. **Case B (Post-PrepareDispatch / Pre-Send)**: Task initially remains `DISPATCHED`, operation remains `DISPATCH_BOUND`, attempt open; diagnostic transaction appends `REVIEW_INTEGRITY_CONFLICT` (`conflict_source = 'WORKSPACE_BINDING_GUARD'`), inserts `ACTIVE` hold (`hold_reason = 'INVARIANT_MISMATCH'`), and CAS invalidates binding `ACTIVE -> INVALIDATED`; verified operator subsequently executes separate D12 terminalization (`DISPATCHED -> FAILED`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'`), followed by hold resolution. **Case C (During RUNNING Intake)**: collector emits `EVIDENCE_COLLECTION_FAILED`, allocates diagnostic hold (`hold_reason = 'DIRTY_WORKTREE_DETECTED'`), preserving `RUNNING`. |
| **REC-006** | Git Merge Conflict | Worker changes conflict with target main branch. | Supervisor Core | Mark task `BLOCKED` (`blocker_reason = MERGE_CONFLICT`); escalate to `HUMAN_REQUIRED` per canonical workflow. |
| **REC-007** | Worker Report Absent | Worker terminates with exit code 0 but creates no report file within bounded fetch window. | Supervisor Core | Mark attempt `FAILED` (`failure_reason = REPORT_MISSING`). Diagnostic Git evidence may be collected for triage, but no promotion to `REPORT_READY`, `EVIDENCE_READY`, or `REVIEWING` occurs. |
| **REC-008** | Malformed Worker Report | Report fails validation against `worker-report.schema.json` or contains mismatched identities. | Supervisor Core | Mark attempt `FAILED` (`failure_reason = REPORT_INVALID` or `REPORT_IDENTITY_MISMATCH`); record raw output for debugging; normal ReviewBundle compilation is aborted. |
| **REC-009** | Missing Test Artifacts | Worker claims tests passed but test log files are missing. | Evidence Collector | Mark test claim as `UNVERIFIED`; highlight discrepancies in Review Bundle for ChatGPT audit. |
| **REC-010** | ChatGPT Disconnected | Web connection drops during active task execution. | Supervisor Core | Execution proceeds autonomously; state transitions to `REVIEWING` once evidence is collected; awaits Supervisor reconnection. |
| **REC-011** | Stale Pair Mismatch | ChatGPT attempts tool call with expired or mismatched token. | Supervisor Core | Reject request with `PAIR_MISMATCH_ERROR`; require re-binding via `bind_project`. |
| **REC-012** | Project Directory Missing | Local repository directory moved or deleted. | Supervisor Core | Mark project `UNAVAILABLE`; freeze Pair state; alert User. |
| **REC-013** | Upstream Incompatibility | Upstream AO changes API format unexpectedly. | AOAdapter | Halt dispatch; mark `UPSTREAM_INCOMPATIBLE`; prompt developer to run compatibility audit. |
| **REC-014** | Unexpected Machine Restart | Host OS reboots during execution. | Supervisor Startup | On startup, execute synchronous recovery scanner across provisioning (`PROVISION_REQUESTED -> PROVISION_FAILED`), dispatch (`SEND_REQUESTED` fail-closed quarantine), in-flight stops (`STOP_REQUESTED`/`STOP_CALL_SUCCEEDED` with restart-stable deadline evaluation and `STOP_REISSUE_REQUIRES_HUMAN`), and open blocked attempts (`# BLOCKED_WITH_OPEN_CURRENT_ATTEMPT MANDATORY_ESCALATION_RECOVERY` to `HUMAN_REQUIRED` with `AO_BLOCKED_ESCALATED`). See §1.2. |
| **REC-015** | Power Loss During Write | Local State Store file write interrupted. | State Store Engine | Selected State Store must provide crash-safe durable persistence and recovery semantics satisfying NFR-003. Concrete engine mechanism is decided by Q4 ADR. |

### 1.1 REC-005 Worktree Recovery Timing & State Model Invariants (ADR-018 D1, D2)

The Supervisor maintains strict zero-mutation isolation over target Git worktrees and adheres to the canonical P02 StateStore/domain model and ADR-018 execution boundaries:

1. **P02 Domain Ground Truth**:
   - `TaskAttempt` contains identity, contract, report path, timestamps, raw report, and execution additions (`attempt_id`, `attempt_number`, `task_id`, `contract_id`, `expected_report_path`, `started_at`, `ended_at`, `worker_report_raw`, `session_id`, `terminal_generation`, `recovery_disposition`, `quarantine_state`). It possesses **no** `state` field and **no** `failure_reason` column in persistence.
   - Canonical `StateMachine` forbids transitions `READY -> BLOCKED` or `READY -> FAILED`. `READY` allows only `READY -> DISPATCHED` (via atomic `PrepareDispatch`) or `READY -> CANCELLED`.
   - Atomic `PrepareDispatch` simultaneously commits `READY -> DISPATCHED`, allocates the persistent `TaskAttempt`, and establishes the canonical workspace binding.

2. **Case A — Dirty Worktree Detected Prior to `PrepareDispatch`**:
   - Pre-dispatch inspection identifies dirty files, untracked artifacts, or uncommitted worktree state before `PrepareDispatch` is invoked.
   - **Behavior**: Dispatch is rejected immediately.
   - **State Invariant**: Task remains in `READY`; zero `TaskAttempt` is allocated; zero AO execution side-effect is triggered.
   - **Audit Record**: Log operational event with evidence label `WORKTREE_DIRTY`.
   - **Resolution**: Escalate to human developer or upstream orchestrator. Pre-dispatch validation may be retried after external clean.

3. **Case B — Unsafe Worktree Condition Discovered Post-`PrepareDispatch` / Pre-Send**:
   - An unsafe condition (workspace binding invalidation, concurrent mutation, or dirty worktree) is detected during the pre-send check sequence while Task is `DISPATCHED`, operation is `DISPATCH_BOUND`, and `TaskAttempt` exists.
   - **Behavior**: Executes a diagnostic transaction:
     1. Task initially remains in `DISPATCHED`, dispatch operation remains in `DISPATCH_BOUND`, and the `TaskAttempt` remains open.
     2. Appends rejection audit event `REVIEW_INTEGRITY_CONFLICT` with Variant B fields (`conflict_source = 'WORKSPACE_BINDING_GUARD'`, `dispatch_operation_id`, `conflict_type = 'LINEAGE_MISMATCH'`, `attempted_reason = '<literal>'`, `sanitized_input_fingerprint`, `diagnostic_fingerprint`, `actor_role = 'SUPERVISOR'`, and `colliding_event_id` strictly ABSENT).
     3. Inserts an `ACTIVE` hold into `review_integrity_holds` with `hold_reason = 'INVARIANT_MISMATCH'` and deterministic `diagnostic_fingerprint`.
     4. If an `ACTIVE` workspace binding exists but lineage or physical identity is invalid, CAS transitions the binding `ACTIVE -> INVALIDATED` (`released_at_epoch_ms = now`).
   - **Subsequent Terminalization & Hold Resolution**:
     - Pre-send `/send` remains strictly forbidden (fail closed).
     - A verified operator (`LENGTH(TRIM(resolved_by_principal)) > 0`) must first execute the D12 atomic terminal transition in a separate transaction:
       * `tasks`: CAS `state = 'FAILED'` where `task_id = ? AND state = 'DISPATCHED'`.
       * `task_attempts`: `ended_at = now`, `recovery_disposition = 'WORKSPACE_BINDING_INTEGRITY_FAILURE'` where `attempt_id = ? AND ended_at IS NULL`.
       * Appends `TASK_STATE_TRANSITION` audit event in that same transaction.
     - Only after terminalization is durably committed may the verified operator resolve the integrity hold in a separate follow-up transaction (`review_integrity_holds.hold_state: ACTIVE -> RESOLVED`, appending `REVIEW_INTEGRITY_HOLD_RESOLVED`).

4. **Case C — Dirty Worktree Discovered During Active Intake (`RUNNING`)**:
   - Subtask P04A / in-memory collector executes `git status --porcelain=v1 -z --untracked-files=all` within the bound worktree and detects modified or untracked files during intake.
   - Transaction A rolls back.
   - **Behavior**: Executes a clean-intake diagnostic transaction:
     1. Derives non-self-referencing `hold_id = 'hold-' + SHA256(RFC8785_JCS(hold_identity_descriptor))` using Descriptor A with `kind = 'review_integrity_hold'`, `hold_reason = 'DIRTY_WORKTREE_DETECTED'`, and `occurrence_number`.
     2. Derives `rejection_event_id = SHA256(RFC8785_JCS(rejection_event_identity_descriptor))` using Descriptor B.
     3. Appends rejection audit event `EVIDENCE_COLLECTION_FAILED` to `audit_events` (`actor = 'ai-supervisor-daemon'`, `details_json.actor_role = 'SUPERVISOR'`).
     4. Inserts an `ACTIVE` hold row into `review_integrity_holds` with `hold_reason = 'DIRTY_WORKTREE_DETECTED'`.
   - **State Invariant**: Task state `RUNNING` is strictly preserved (strictly zero blanket transitions to `BLOCKED` or `FAILED`). Promotion to `REPORT_READY` or `REVIEWING` is blocked while the diagnostic hold is `ACTIVE`.

---

### 1.2 Double-Gated Quarantine Clearance Rules & Startup Recovery Architecture

#### 1.2.1 Independent Double-Gated Quarantine Enforcement (ADR-016 §12, ADR-018 D1)

The Control Plane enforces two independent safety gates across execution and allocation lanes:
1. **Pair Lane Gate (`worker_sessions.quarantine_state`)**:
   - Values: `CLEAN`, `QUARANTINED` (default `CLEAN`).
   - Guard: When `QUARANTINED`, locks the entire Pair lane against new task or attempt allocation.
2. **Task Attempt Gate (`task_attempts.quarantine_state`)**:
   - Values: `CLEAN`, `QUARANTINED` (default `CLEAN`).
   - Guard: When `QUARANTINED`, prevents retry of that specific task (`FAILED -> READY` is blocked).
3. **Dispatch & Retry Guards (14 Verification Guards)**:
   - `PrepareDispatch` and pre-send evaluation verify all 14 mandatory guards:
     - Target `worker_sessions.quarantine_state == 'CLEAN'`; AND
     - All prior attempts for the target task have `task_attempts.quarantine_state == 'CLEAN'`; AND
     - Zero unresolved provisioning operations: `NOT EXISTS (SELECT 1 FROM pair_provisioning_operations WHERE pair_id = ? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED'))`; AND
     - Zero unresolved restore operations: `NOT EXISTS (SELECT 1 FROM pair_restore_operations WHERE pair_id = ? AND resolution_state <> 'RESTORE_RESOLVED')`; AND
     - Canonical workspace binding is active: exactly one row in `attempt_workspace_bindings` with `binding_state = 'ACTIVE'`; AND
     - Live lease validation passes: `WorkspaceBindingLease.Revalidate()` matches the database record; AND
     - Zero active review integrity holds: `NOT EXISTS (SELECT 1 FROM review_integrity_holds WHERE attempt_id = ? AND hold_state = 'ACTIVE')`.
   - If any condition fails, dispatch is rejected immediately with `ErrQuarantinedExecution` or `ErrWorkspaceBindingIntegrity`.
4. **Separation of Safety State from Diagnostic Disposition**:
   - `task_attempts.recovery_disposition` records machine-readable diagnostic outcomes (e.g., `UNCERTAIN_DELIVERY_CRASH`, `STOP_TARGET_ABSENT`, `STOP_CONFIRMATION_TIMEOUT`, `WORKSPACE_BINDING_INTEGRITY_FAILURE`).
   - `task_attempts.quarantine_state` is the immutable safety lock. Diagnostic observations update `recovery_disposition` but MUST NOT clear `quarantine_state`.

#### 1.2.2 Quarantine Clearance Classes (ADR-016 §6, §12)

Quarantine may be cleared ONLY in a single atomic SQLite transaction verifying the exact lineage tuple `(attempt_id, session_id, terminal_generation)` under one of three governed resolution classes. For a stop with `restore_operation_id`, stop confirmation records evidence **only**: first resolve the linked restore operation in Tx D (exact operation/stage/resolution/claim/version and audit), then run a separate D6 clearance transaction checking WorkerSession and **each** related attempt against its own lineage, evidence and authority. A failed CAS/audit rolls back the entire clearance; resolving the restore operation does not clear quarantine. A stop without `restore_operation_id` retains the ADR-016 D6/D11 behavior below and the audited 3A attempt snapshot/stop lineage check.

1. **Class A (`PHYSICAL_EXECUTION_RESOLUTION`)**:
   - **Preconditions**: Physical termination of the matching generation is durably confirmed under positive intentional stop evidence:
     1. Positive kill call acceptance proven (`stage == 'STOP_CALL_SUCCEEDED'`);
     2. Observed `session_id` matches `stop_operations.session_id`;
     3. Observed `terminal_generation` matches `stop_operations.terminal_generation` (string equality);
     4. Upstream returns `isTerminated == true`;
     5. Observation occurs within restart-stable temporal window `now < confirmation_deadline_at`.
   - **Branch A1: Active Attempt Live Stop (`purpose == 'RUNNING_ATTEMPT_STOP'`)**:
     - Governed strictly by `WORKER_STOPPED_ALLOWED_IFF` (ADR-016 §17) on a task in `RUNNING`.
     - Sets `task_attempts.ended_at = now`; records `task_attempts.recovery_disposition = 'WORKER_STOPPED'`; transitions TaskState `RUNNING -> FAILED` (`failure_reason = WORKER_STOPPED`).
     - Clears attempt quarantine (`task_attempts.quarantine_state = 'CLEAN'`) and Pair lane quarantine (`worker_sessions.quarantine_state = 'CLEAN'`); emits audit event `QUARANTINE_RESOLVED_PHYSICAL`.
   - **Branch A2: Governed Quarantine Cleanup (`purpose == 'QUARANTINE_CLEANUP'`)**:
     - Governed by ADR-016 D5/D6/D11 (§11, §12, §17 Purpose Matrix).
     - Precondition: Task is ALREADY in terminal failure (`FAILED` or `HUMAN_REQUIRED`) under active quarantine.
     - For an ordinary stop (`restore_operation_id IS NULL`), confirmed physical termination (`stage = 'STOP_TERMINATION_CONFIRMED'`, `resolution_state = 'TERMINATION_CONFIRMED'`) clears quarantine in a single atomic SQLite transaction:
       - Sets `task_attempts.quarantine_state = 'CLEAN'`;
       - If authorized, sets `worker_sessions.quarantine_state = 'CLEAN'`;
       - Emits audit event `QUARANTINE_RESOLVED_PHYSICAL`.
     - Invariants: Drives **ZERO TaskState transition** (task remains in its terminal state); executes **ZERO `ended_at` mutation** (already closed); and does **NOT** write `recovery_disposition = 'WORKER_STOPPED'` (which is reserved exclusively for live attempt stops under `WORKER_STOPPED_ALLOWED_IFF`). The attempt's diagnostic `recovery_disposition` accurately preserves its original failure cause (e.g. `UNCERTAIN_DELIVERY_CRASH`).
     - For linked `QUARANTINE_CLEANUP` (`restore_operation_id IS NOT NULL`), target session/generation must equal the immutable snapshot of that **closed** attempt. Confirmation records Class A evidence for that lineage but leaves both quarantine gates unchanged until Tx D and the separate D6 clearance. Runtime generation mới không được gắn stop vào attempt cũ để lách 3A-R1-002.
   - **Fail-Closed Invariants across Class A**:
     - If observation occurs at or after deadline (`now >= confirmation_deadline_at`), Condition 6 is violated: fails closed to `STOP_CONFIRMATION_TIMEOUT`. Quarantine remains ACTIVE. Recording `WORKER_STOPPED` or clearing quarantine is strictly PROHIBITED.
     - If generation mismatch occurs: fails closed to `STOP_GENERATION_MISMATCH`. Quarantine remains ACTIVE.
     - If transport fails ambiguously (`STOP_CALL_OUTCOME_UNKNOWN`): quarantine remains ACTIVE; automatic retry prohibited.
     - If upstream returns HTTP 404: `STOP_TARGET_ABSENT`. 404 proves session absence, NOT physical termination; quarantine remains ACTIVE until operator Class B resolution.
     - Clean-worktree inspection is NOT a condition for physical execution clearance.

2. **Class B (`ADMINISTRATIVE_RISK_RESOLUTION` / HTTP 404 Session Absence)**:
   - HTTP 404 is absence evidence only. `STOP_TARGET_ABSENT` remains the stop stage/resolution; it is never physical proof.
   - A verified operator accepts residual risk for each exact lineage in a separate atomic, replay guarded `ADMINISTRATIVE_RISK_ACCEPTED` audit transaction. Quarantine remains active. For linked restore, stop reconciliation precedes Tx D with administrative basis, and D6 clearance follows Tx D. Only D6 emits `QUARANTINE_RESOLVED_ADMINISTRATIVE` when it actually clears a lineage.

3. **Class C (`ADMINISTRATIVE_RISK_RESOLUTION` / Human Risk Acceptance)**:
   - When termination cannot be proven, a verified operator with independent recovery scope may accept residual risk for exact session/attempt lineages. CAS changes `stop_operations.resolution_state` to `ADMINISTRATIVE_RISK_ACCEPTED` and records `resolved_at` plus separate `ADMINISTRATIVE_RISK_ACCEPTED` audit events atomically. Wire stage, timestamps and deadline are retained; quarantine remains active and no physical proof is created.
   - A linked stop checks restore claim/owner. Audited stop reconciliation precedes Tx D; Tx D uses administrative basis when any lineage has only accepted risk; lineage complete D6 clearance is a separate transaction. D6 alone emits `QUARANTINE_RESOLVED_ADMINISTRATIVE`. New generation evidence never clears old attempt risk; no fake attempt is allocated for `PAIR_MAINTENANCE`. Replay is read-only on an exact committed decision; competing/stale decisions fail.
   - See `docs/adr/ADR-016-CLARIFICATION-administrative-stop-risk-acceptance.md`; verified host principal remains a fail-closed runtime dependency.

#### 1.2.3 Synchronous Startup Recovery Sweep (ADR-016 §19)

Upon Supervisor daemon restart, prior to accepting incoming client API requests, the daemon executes a deterministic 5-step synchronous recovery scanner:

**Restore preclassification before Step 1** (approved addendum §§3–5): inspect `pair_restore_operations` and establish `RESTORE_UNRESOLVED` ownership/Pair guards before provisioning, dispatch or stop decisions. `RESTORE_REQUESTED/IN_FLIGHT` after crash enters `RESTORE_OUTCOME_UNKNOWN` with WorkerSession and related-attempt quarantine in Tx C; `RESTORE_CONFIRMED/IN_FLIGHT` retains HTTP 200 provenance and lock. Ambiguous Tx C/CAS failure requires durable reread; no `/restore` or `/kill` reissue. This preclassification is a prerequisite to the existing five sweeps, not a new TaskState transition or a permission for a later sweep to bypass restore ownership. Host principal remains required for operator recovery and linked stop.

1. **Step 1: Pair Provisioning Operations Sweep**:
   - Query all rows with `pair_provisioning_operations.stage = 'PROVISION_REQUESTED'`.
   - Transition each row to `stage = 'PROVISION_FAILED'`, setting resolution metadata (`resolved_at = now`, `resolution_notes = 'STARTUP_RECOVERY_SWEEP'`).
   - Lock affected Pair lanes against automatic dispatch (`PROVISION_CONFIRMED` rows are NOT scanned as unresolved).
2. **Step 2: Dispatch Operations Sweep (Unknown Delivery Containment)**:
   - Query `dispatch_operations.stage = 'SEND_REQUESTED' AND resolution_state IS NULL`; terminal `DELIVERY_OUTCOME_UNKNOWN` is read-only on replay.
   - Apply D5 unknown delivery fail-closed handling:
     - Transition task `DISPATCHED -> FAILED` (reason: `UNCERTAIN_DELIVERY_CRASH`);
     - Set `task_attempts.recovery_disposition = 'UNCERTAIN_DELIVERY_CRASH'`;
     - Set `task_attempts.quarantine_state = 'QUARANTINED'`;
     - Set `worker_sessions.quarantine_state = 'QUARANTINED'`;
     - CAS `dispatch_operations.resolution_state IS NULL` to terminal `DELIVERY_OUTCOME_UNKNOWN`; retain `stage = 'SEND_REQUESTED'` through later clearance.
     - Emit audit event `UNCERTAIN_DELIVERY_QUARANTINE_IMPOSED`.
   - `SEND_CONFIRMED` rows are accepted writes; they are reconciled against authoritative upstream status rather than unknown delivery.
3. **Step 3: In-Flight Stop Operations Sweep**:
   - Query all rows where `stop_operations.resolution_state = 'IN_FLIGHT'` and `stage IN ('STOP_REQUESTED', 'STOP_CALL_SUCCEEDED')`.
   - Separate ordinary stops (`restore_operation_id IS NULL`) from linked `QUARANTINE_CLEANUP`/`PAIR_MAINTENANCE`. For linked stops, verify same restore operation/Pair/session, trusted ownership and target generation; do not bypass the restore guard. If restore/stop CAS or audit fails, reread durable state. No startup `/kill` reissue. 3A-R1-002 still requires exact attempt session/generation whenever `attempt_id` is non-NULL.
   - For `STOP_REQUESTED`:
     - If alive with matching generation: automatic `/kill` reissue is strictly PROHIBITED (`PINNED_KILL_GENERATION_ATOMIC_FENCE = ABSENT`). Record `resolution_state = 'STOP_REISSUE_REQUIRES_HUMAN'`. Retain active quarantine; require human intervention (`STOP_REISSUE_REQUIRES_HUMAN`).
     - If `isTerminated == true`: kill call was never confirmed transmitted/accepted. Retain `stage = 'STOP_REQUESTED'`, set `resolution_state = 'STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED'`. If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: transition task `RUNNING -> FAILED` (reason: `WORKER_TERMINATION_UNKNOWN`). Retain quarantine; do NOT record `WORKER_STOPPED`.
     - If generation mismatch: retain `stage = 'STOP_REQUESTED'`, set `resolution_state = 'STOP_GENERATION_MISMATCH'`. Transition `RUNNING -> FAILED` (disposition: `STOP_GENERATION_MISMATCH`).
     - Hai kết quả `STOP_GENERATION_MISMATCH` và `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED` phát `STOP_OPERATION_RESOLVED` trong cùng transaction với stop resolution và D12 closure của live stop. Recovery intent cũ với fresh exact GET `isTerminated=false` cũng phát event này khi CAS `STOP_REQUESTED/IN_FLIGHT → STOP_REQUESTED/STOP_REISSUE_REQUIRES_HUMAN`; giữ deadline, quarantine và không gọi `/kill`. Query failure không chứng minh target còn sống. Event chỉ ghi logical resolution, không phải physical proof hoặc clearance (clarifications TASK-P03-003C/3D).
     - If HTTP 404: set `stage = 'STOP_TARGET_ABSENT'`, `resolution_state = 'STOP_TARGET_ABSENT'`. Transition `RUNNING -> FAILED` (reason: `SESSION_ABSENT`).
   - For `STOP_CALL_SUCCEEDED`:
     - Kill signal was already accepted upstream. Reissuing `/kill` is redundant and prohibited.
     - Evaluate observed status against restart-stable `confirmation_deadline_at`:
       1. *HTTP 404*: `stage = 'STOP_TARGET_ABSENT'`, `resolution_state = 'STOP_TARGET_ABSENT'`, transition `RUNNING -> FAILED` (`SESSION_ABSENT`), quarantine remains active.
       2. *Generation Mismatch*: `resolution_state = 'STOP_GENERATION_MISMATCH'`, transition `RUNNING -> FAILED` (`STOP_GENERATION_MISMATCH`), quarantine remains active.
       3. *Matching Generation and `isTerminated == true`*:
           - **Case 1: Observation within deadline (`now < confirmation_deadline_at`)**:
             Set `stage = 'STOP_TERMINATION_CONFIRMED'`, `resolution_state = 'TERMINATION_CONFIRMED'`, `resolved_at = now`.
             Outcome diverges strictly based on stop `purpose`:
             - *Sub-case 3a: Live Attempt Stop (`purpose == 'RUNNING_ATTEMPT_STOP'` on task in `RUNNING`)*:
               All 6 verifiable conditions of `WORKER_STOPPED_ALLOWED_IFF` are met. Execute `AtomicTerminalTransition`: set `task_attempts.ended_at = now`, record `task_attempts.recovery_disposition = 'WORKER_STOPPED'`, transition TaskState `RUNNING -> FAILED` (`failure_reason = WORKER_STOPPED`), and release attempt and Pair lane quarantine (`quarantine_state = 'CLEAN'`).
             - *Sub-case 3b: Governed Quarantine Cleanup (`purpose == 'QUARANTINE_CLEANUP'`)*:
               Governed by ADR-016 D5/D6/D11. Task is already in terminal state (`FAILED` or `HUMAN_REQUIRED`). For `restore_operation_id IS NULL`, confirmed physical termination performs the existing Class A clearance: sets attempt/WorkerSession quarantine to `CLEAN` and emits `QUARANTINE_RESOLVED_PHYSICAL`. For `restore_operation_id IS NOT NULL`, confirmation writes **only** stop/Class A evidence for the exact closed-attempt snapshot; quarantine stays `QUARANTINED` until restore Tx D and separate per-lineage D6 clearance. Both paths preserve **ZERO TaskState transition**, **ZERO `ended_at` mutation**, no `WORKER_STOPPED` disposition, and the original diagnostic failure reason.
             - *Sub-case 3c: Infrastructure Maintenance (`purpose == 'PAIR_MAINTENANCE'`)*:
               Ordinary stop: ZERO TaskState transition and no attempt mutation; updates Pair lane maintenance state only. Approved linked new-generation restore cleanup: `restore_operation_id IS NOT NULL`, no `task_id`/`contract_id`/`attempt_id`, verified operator authority and old/new generation provenance. Confirmation is Class A evidence **only for the target runtime generation**; it does not clear WorkerSession or any historical attempt quarantine before Tx D and separate D6 clearance. New-generation termination never proves old-generation execution resolution; do not create a fake attempt.
           - **Case 2: First observation at or after deadline (`now >= confirmation_deadline_at`)**:
             Condition 6 of `WORKER_STOPPED_ALLOWED_IFF` is NOT met. Pinned AO wire contract exposes no process exit timestamp, exit code, or OS signal (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`); the control plane MUST NOT speculate whether process exit occurred before or after the deadline. Under strict fail-closed handling, recording `WORKER_STOPPED` and clearing quarantine are strictly PROHIBITED. Retain `stage = 'STOP_CALL_SUCCEEDED'`, set `resolution_state = 'STOP_CONFIRMATION_TIMEOUT'`, `resolved_at = now`. If `purpose == 'RUNNING_ATTEMPT_STOP'` and task is `RUNNING`: transition task `RUNNING -> FAILED` (`STOP_CONFIRMATION_TIMEOUT`). For both live stops and cleanups, quarantine remains ACTIVE on attempt and Pair.
       4. *Matching Generation and Alive (`isTerminated == false`)*:
          - If `now < confirmation_deadline_at`: remain `stage = 'STOP_CALL_SUCCEEDED'`, `resolution_state = 'IN_FLIGHT'`, and continue bounded observation until deadline.
          - If `now >= confirmation_deadline_at`: retain `stage = 'STOP_CALL_SUCCEEDED'`, set `resolution_state = 'STOP_CONFIRMATION_TIMEOUT'`. Transition `RUNNING -> FAILED` (`STOP_CONFIRMATION_TIMEOUT`), quarantine remains active; do NOT record `WORKER_STOPPED`.
4. **Step 4: Blocked Attempt Crash Escalation (`# BLOCKED_WITH_OPEN_CURRENT_ATTEMPT MANDATORY_ESCALATION_RECOVERY`)**:
   - Query all tasks where `tasks.state = 'BLOCKED'` AND the matching current `TaskAttempt` has `ended_at IS NULL`.
   - Because canonical workflow defines NO `BLOCKED -> RUNNING` edge, an open attempt in `BLOCKED` can never resume execution. Retaining `ended_at = NULL` would permanently orphan the attempt.
   - For each matching task, the scanner immediately executes `AtomicAttemptClosureTransition` in a single SQLite transaction:
     1. Atomically update `tasks.state` from `BLOCKED` to `HUMAN_REQUIRED`;
     2. Set `task_attempts.ended_at = now`;
     3. Set `task_attempts.recovery_disposition = 'AO_BLOCKED_ESCALATED'`;
     4. Append audit event `WORKER_BLOCKED_ESCALATED`.
   - Zero network calls are made (pure control plane deterministic escalation).
5. **Step 5: In-Flight Task & Pair Lane Reconciliation**:
   - Probe AO daemon availability exclusively via approved adapter methods: `CheckHealth(ctx)` / `CheckReadiness(ctx)`.
   - *If AO is Reachable*: Reconcile in-flight tasks via `GetWorkerStatus(ctx, sessionID)` using opaque string generation matching. If active/idle with matching generation, resume standard lifecycle observation **subject to exact attempt/stage/hold CAS**; a successful GET alone never clears `PRE_SEND_PROTOCOL_UNVERIFIED` or either quarantine gate. Protocol hold requires verified operator compatibility decision plus fresh matching GET and audit; `RECOVERY_PENDING` can clear only by its approved same-attempt CAS. If session absent (404) or terminated, execute the applicable atomic failure transition.
   - *If AO is Unreachable*: Do NOT blindly fail in-flight tasks. For open `DISPATCH_BOUND` pre-send attempts, CAS exact lineage, stage and **old disposition**: `NULL -> RECOVERY_PENDING` or `RECOVERY_PENDING -> RECOVERY_PENDING`; if already `PRE_SEND_PROTOCOL_UNVERIFIED`, retain that stronger hold and audit the outage without overwriting it. For exact current open attempt at `DISPATCHED/SEND_CONFIRMED` or `RUNNING/SEND_CONFIRMED`, CAS `NULL → RECOVERY_PENDING` with `POST_SEND_RECOVERY_PENDING` audit in one transaction; replay is read-only and stronger disposition is retained. Fresh valid exact GET may CAS `RECOVERY_PENDING` to `NULL` or approved diagnostic with `POST_SEND_STATUS_RECOVERED` audit; terminal outcome uses D12 in that same transaction. GET alone never clears quarantine or grants resend. Other affected tasks follow their existing recovery rules without replacing a stronger disposition. Keep affected Pair lanes unavailable for send, and schedule background reconciliation using caller-injected `SUPERVISOR_ACTIVITY_POLL_INTERVAL`; no new operational policy, retry count or backoff is invented. A competing writer's CAS win requires durable reread, never blind hold downgrade or send.


### 1.3 ADR-016 addendum restore và hold recovery

REC-002 theo addendum: daemon reconnect/GET chỉ quan sát, không tự gọi /restore. RESTORE_REQUESTED sau crash hoặc response/commit ambiguous phải chuyển RESTORE_OUTCOME_UNKNOWN với Pair lock/quarantine; không retry. Verified operator claim/transfer và linked stop đi theo §5 addendum. Old attempt cleanup dùng exact immutable session/generation; new runtime generation dùng approved linked PAIR_MAINTENANCE không gắn attempt, nhưng termination generation mới không clear old attempt. Tx D resolve operation rồi D6 transaction riêng kiểm tra evidence/authority cho WorkerSession và từng attempt, rollback toàn bộ nếu một CAS/audit fail. Protocol hold không bị timeout hạ cấp; successful GET không clear hold cần operator. SEND_REQUESTED unknown ghi DELIVERY_OUTCOME_UNKNOWN terminal, clearance không đổi resolution. 3D scanner phân loại, 3C stop/clearance.

### Exclusive host quiescence cho startup recovery

ADR-016 addendum `exclusive-host-quiescence-for-startup-recovery` yêu cầu trusted host đóng admission chung cho stop/send/restore/provisioning và mutation xung đột, drain và join mọi caller giữ quyền effect, rồi cấp exclusive scoped ownership cho `Run`. Persisted intent, timestamp hoặc Store CAS không chứng minh caller cũ đã hết quyền effect. Thiếu provider hoặc drain không hoàn tất thì runner fail closed trước khi phân loại intent; poller không tự phân loại startup intent. Release ownership sau incomplete/error/cancellation không tự mở admission hoặc serve. Host nhiều process phải chứng minh exclusion trên cùng DB/Pair/effect admission, ngoài phạm vi thư viện 3D. `HOST_QUIESCENCE_INTEGRATION=OPEN` cùng `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`.

### REC-004 execution budget và legacy maintenance (ADR-016 addendum §§3–6)

Budget bắt đầu ở `dispatch_operations.confirmed_at` khi `/send` HTTP 200 được xác nhận, **không** chứng minh actual execution start. Tx send confirmation ghi origin/deadline/policy bất biến cùng stage/audit; execution deadline và stop confirmation deadline độc lập. Startup `Run` chỉ phân loại, không timeout kill. Runtime monitor sau startup chỉ xét exact current open `RUNNING/SEND_CONFIRMED`, fresh active/waiting_input, valid budget và `now >= deadline_at`; DISPATCHED/blocked/P04/terminated/404/mismatch/outage đi nhánh ưu tiên §5.1, không ép `DISPATCHED→RUNNING` để stop. Tx R dùng `ReserveStopOperation` của 3C, commit stop intent + WorkerSession/attempt quarantine + `STOP_OPERATION_REQUESTED` audit nguyên tử. Guard trước effect chấp nhận quarantine của chính reservation thắng nhưng không chấp nhận risk/owner mới. Nếu GET sau Tx R đổi sang blocked/idle/exited/identity invalid hoặc unavailable, caller bỏ effect, giữ `STOP_REQUESTED/IN_FLIGHT`, `RUNNING` mở và double quarantine; không có resolution audit hoặc retry kill. 404/terminated/generation mismatch đi D11/D12 outcome hiện hữu. Chỉ sau exclusive host drain/join, startup/human reconciliation mới phân loại intent.

Legacy current open `SEND_CONFIRMED` thiếu budget làm Run incomplete và normal admission/serve đóng. Trusted maintenance scope loại trừ Run và mọi effect caller xung đột, drain/join trước Tx L hoặc manual stop; scope dùng cùng host boundary nhưng không lồng normal admission. Verified historical-policy scope `LEGACY_EXECUTION_BUDGET_RECONCILIATION` chỉ bind khi có policy/ref/duration tại send và exact `confirmed_at`; Tx L audit/CAS atomic. Verified manual scope `MANUAL_STOP_RECONCILIATION` chỉ dùng 3C one-use stop cho exact current open **`RUNNING`**; legacy `DISPATCHED` thiếu evidence/đường reconciliation hợp lệ tiếp tục fail-closed. Cancel/crash/release maintenance không tự mở admission; phải Run lại hoàn tất. `EXECUTION_BUDGET_LEGACY_BOUND` là audit binding, không là physical proof. Contract revision 3 đã RELEASED; `3D-R1-005` cùng toàn bộ `3D-R1-001..005` và `3D-R2-001..002` đã CLOSED ở library scope tại implementation commit `7513f1b9f39fa459be15e0abc836c6258a610d8b` (`EXTERNAL_AUDIT_APPROVED`); code đã MERGED tại `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`. Handoff dependency `HOST_QUIESCENCE_INTEGRATION=OPEN`, verified host principal và `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED` tiếp tục duy trì ở runtime.

### 1.4 NTFS Directory Substitution Probe Matrix & Review Integrity Conflict Recovery (ADR-018 D1, D7)

#### 1.4.1 Valid NTFS Directory Substitution Probe Matrix (CR-04)
The Supervisor verifies workspace directory binding integrity on Windows NTFS without relying on invalid directory hard link claims (NTFS strictly prohibits hard links on directories). Valid directory substitution probes include:
1. **Directory Junction Points (`IO_REPARSE_TAG_MOUNT_POINT`)**: Validated via Win32 reparse point tags and volume GUID target resolution.
2. **Directory Symbolic Links (`IO_REPARSE_TAG_SYMLINK`)**: Validated via reparse data buffers and unprivileged developer mode / admin privilege inspection.
3. **Volume Mount Alias (`subst`)**: Validated via `QueryDosDeviceW` device resolution to real volume paths.
4. **Directory Rename / Replacement**: Validated via handle-preserving anti-rename locks (`FILE_SHARE_READ | FILE_SHARE_WRITE`) and file ID tracking.
5. **Path-Alias Probes**: Validated via canonicalized normalized path comparisons and volume serial number + 64-bit file index matching (`BY_HANDLE_FILE_INFORMATION`).

#### 1.4.2 Audit Event Derivation & RFC 8785 JCS (CR-10)
All integrity event IDs and hold IDs are deterministically derived using RFC 8785 JSON Canonicalization Scheme (JCS) and SHA-256 hashing. Delimiter string concatenation is strictly prohibited.
- **Descriptor A (`hold_identity_descriptor` for `hold_id`)**: SHA-256 of canonical JSON containing `attempt_id`, `descriptor_type = 'HOLD_IDENTITY'`, `hold_reason`, and `target_path`.
- **Descriptor B (`rejection_event_identity_descriptor` for `rejection_event_id`)**: SHA-256 of canonical JSON incorporating `conflict_source` and exact variant fields.
- **Descriptor C (`resolution_event_identity_descriptor` for `resolution_event_id`)**: SHA-256 of canonical JSON containing `hold_id`, `resolution_action`, `resolved_by`, and `descriptor_type = 'RESOLUTION_EVENT_IDENTITY'`.

#### 1.4.3 `REVIEW_INTEGRITY_CONFLICT` Variant Discrimination (CR-10)
When an integrity conflict occurs during hold evaluation or binding checks, the recovery system discriminates between two distinct variants:
1. **Variant A (`conflict_source = 'AUDIT_EVENT_ID_COLLISION'`)**:
   - Fields: Requires `colliding_event_id`, `attempted_event_type`, and mismatch details.
   - Binding Invariant: Zero binding mutation (`attempt_workspace_bindings` remains unchanged).
   - Recovery Action: Emits audit event `REVIEW_INTEGRITY_CONFLICT` with Variant A payload. Fails closed to operator investigation without corrupting binding state.
2. **Variant B (`conflict_source = 'WORKSPACE_BINDING_GUARD'`)**:
   - Fields: Requires `dispatch_operation_id`, `conflict_type = 'LINEAGE_MISMATCH'`, `attempted_reason` (strictly one of the 4 canonical literals: `WORKSPACE_BINDING_MISSING`, `WORKSPACE_BINDING_LINEAGE_MISMATCH`, `WORKSPACE_BINDING_NOT_ACTIVE`, `WORKSPACE_BINDING_PHYSICAL_IDENTITY_MISMATCH`), `sanitized_input_fingerprint`, `diagnostic_fingerprint`, and `actor_role = 'SUPERVISOR'`.
   - Invariant: `colliding_event_id` is strictly absent.
   - Diagnostic Hold: Inserts an `ACTIVE` hold in `review_integrity_holds` with `hold_reason = 'INVARIANT_MISMATCH'`.
   - Binding Mutation: CAS invalidates the workspace binding (`binding_state` transitions `ACTIVE -> INVALIDATED`, `released_at_epoch_ms = now`).
   - Pipeline Isolation: Subtask P04D verification conflicts never CAS mutate workspace bindings.
   - Zero Orphan Guarantee: Audit events are committed within the same SQLite transaction as the state transition; lost CAS races emit zero orphan audit events.
