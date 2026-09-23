# DRAFT TASK CONTRACT: TASK-P03-003C

> **Status:** `DRAFT_PENDING_EXTERNAL_SUPERVISOR_APPROVAL`; `TASK_P03_003C=NOT_RELEASED`. Tài liệu này không cấp quyền sửa production code.
> **Proposed code base SHA:** `246f74eabd09bbc2ea66624ee217035167824a86` (merge 3B đã tồn tại). Artifact contract sẽ được giao riêng sau audit/release; không nhúng SHA tự tham chiếu.
> **Authority:** ADR-016 accepted, addendum approved, parent decomposition §3.3. `AUTOMATIC_RESTORE=DISABLED`; host/bootstrap phải cung cấp verified principal trước khi enable linked stop hoặc administrative clearance.

## 1. JSON TaskContract dự thảo

```json
{
  "contract_id": "CONTRACT-TASK-P03-003C-01",
  "task_id": "TASK-P03-003C",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P03",
  "objective": "Implement purpose-aware stop coordination and lineage-specific D6 quarantine clearance using approved ADR-016 D5/D6/D11 and addendum, without startup scanning or polling.",
  "requirements": ["FR-005", "FR-006", "NFR-003", "NFR-005", "SEC-001"],
  "architecture_refs": [
    "docs/tasks/DRAFT_TASK_CONTRACT_P03_003.md#section-3.3",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-12",
    "docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md#section-17",
    "docs/adr/ADR-016-ADDENDUM-restore-and-pre-send-semantics.md#section-5",
    "docs/04_ARCHITECTURE.md",
    "docs/05_DOMAIN_MODEL.md",
    "docs/06_WORKFLOW_STATE_MACHINE.md",
    "docs/08_TASK_CONTRACT.md",
    "docs/12_UPSTREAM_INTEGRATION.md",
    "docs/14_FAILURE_RECOVERY.md",
    "docs/21_TRACEABILITY_MATRIX.md",
    "docs/22_MODULE_PROVENANCE.md",
    "docs/phases/P03_AO_INTEGRATION.md",
    "docs/sources/SOURCE_REGISTRY.md",
    "docs/sources/REUSE_MATRIX.md"
  ],
  "base_sha": "246f74eabd09bbc2ea66624ee217035167824a86",
  "allowed_scope": [
    "internal/stop/coordinator.go",
    "internal/stop/coordinator_test.go",
    "internal/dispatch/coordinator.go",
    "internal/dispatch/coordinator_test.go",
    "internal/domain/session_lifecycle.go",
    "internal/domain/session_lifecycle_test.go",
    "internal/store/session_lifecycle.go",
    "internal/store/session_lifecycle_test.go",
    "internal/store/session_lifecycle_remediation_test.go",
    "internal/store/stop_transactions.go",
    "internal/store/stop_transactions_test.go",
    "internal/store/clearance_transactions.go",
    "internal/store/clearance_transactions_test.go",
    "internal/store/restore_transactions.go",
    "internal/store/restore_transactions_test.go",
    "internal/store/atomic_transitions.go",
    "internal/store/atomic_transitions_test.go"
  ],
  "forbidden_scope": [
    "internal/ao/**",
    "internal/poller/**",
    "internal/workflow/**",
    "docs/adr/**",
    "docs/proposals/**",
    "docs/schemas/**",
    "docs/phases/**",
    "AGENTS.md",
    "docs/18_CURRENT_STATE.md"
  ],
  "constraints": [
    "Use pinned AO StopWorker(sessionId) and GetWorkerStatus; /kill wire has only sessionId and HTTP 200 proves acceptance, not termination. No AOAdapter or schema change.",
    "Commit STOP_REQUESTED and audit before the one /kill effect; make no network call in SQLite transactions. Persist validated HTTP 200, exact call_completed_at, and one caller-injected confirmation_deadline_at atomically.",
    "SUPERVISOR_KILL_STOP_TIMEOUT is injected by the host; reject absent/invalid policy, never invent a numeric default or reset a persisted deadline.",
    "Precheck exact session and opaque generation, but never claim an atomic generation fence; after a mismatched observation do not call /kill. No blind /kill reissue after ambiguous transport, invalid response, process crash or context cancellation.",
    "For RUNNING_ATTEMPT_STOP, stop stage/resolution, RUNNING-to-FAILED D12 closure, disposition and audit must commit under one transaction and exact current attempt/session/generation; WORKER_STOPPED requires all six D11 conditions including strict termination_confirmed_at before deadline.",
    "QUARANTINE_CLEANUP requires the immutable snapshot of its own closed FAILED/HUMAN_REQUIRED attempt; PAIR_MAINTENANCE has no fake attempt. Neither cleanup nor maintenance changes TaskState or ended_at.",
    "Linked stop confirmation records only evidence. Restore Tx D resolution remains distinct from D6 clearance; Class A requires positive physical proof of current runtime and every quarantined old lineage, while Class B/C require verified operator risk acceptance for each unresolved lineage.",
    "Clear WorkerSession and every affected TaskAttempt quarantine through one exact-lineage CAS/audit transaction only after required resolution; rollback leaves all quarantine gates intact, and clearance never rewrites DELIVERY_OUTCOME_UNKNOWN.",
    "Trusted operator boundary fails closed without host-authenticated principal and approved scope; fake test principals do not enable runtime restore/linked stop. AUTOMATIC_RESTORE remains DISABLED.",
    "3D owns startup scanner and poller. 3C exposes bounded single-observation/reconciliation operations without ticker, background loop, daemon bootstrap, SSE, synthetic heartbeat or live AO testing."
  ],
  "acceptance_criteria": [
    "AC-3C-01: purpose-aware unlinked STOP_REQUESTED reservation and audit commit before one AO /kill; exactly one Pair/operation reservation winner receives effect permission, replay is read-only, and duplicate or audit/CAS failure leaves zero extra effect. Linked cleanup reuses the atomic 3B claim/stop intent.",
    "AC-3C-02: exact pre-call GET session/generation prevents /kill on mismatch, absent or already terminated target; tests acknowledge the remaining session-scoped TOCTOU window and never assert a generation fence.",
    "AC-3C-03: validated HTTP 200 commits STOP_CALL_SUCCEEDED, call_completed_at, immutable injected confirmation_deadline_at and STOP_OPERATION_CALL_SUCCEEDED audit; HTTP 200 alone never records termination or WORKER_STOPPED.",
    "AC-3C-04: 404 records STOP_TARGET_ABSENT/absence audit and retains quarantine; definite upstream rejection and ambiguous transport/invalid response use their exact D11 stage/resolution/audit, never blind reissue.",
    "AC-3C-05: single status observation within the persisted deadline confirms termination only for exact session/generation with isTerminated=true; before/equal/after deadline and restart replay preserve strict nanosecond comparison and original deadline.",
    "AC-3C-06: RUNNING_ATTEMPT_STOP commits D11 result with D12 RUNNING-to-FAILED, current-attempt closure, exact disposition and audit in one transaction; only all six D11 conditions permit WORKER_STOPPED.",
    "AC-3C-07: QUARANTINE_CLEANUP and PAIR_MAINTENANCE produce zero TaskState/ended_at mutation; linked new-generation maintenance has restore link/authority but no attempt, and cannot clear old-generation attempt quarantine.",
    "AC-3C-08: Class A clearance checks positive D11 proof for current runtime and every quarantined attempt; old-only/new-only evidence or mismatched lineage cannot release WorkerSession or attempt.",
    "AC-3C-09: Class B uses 404 absence plus verified administrative risk acceptance, Class C uses explicit verified human risk acceptance; each unresolved lineage and current runtime has scoped basis/evidence/audit, never false physical evidence.",
    "AC-3C-10: linked restore Tx D resolves operation before separate D6 clearance; D6 performs exact WorkerSession and every-attempt CAS plus per-lineage audit atomically; no-attempt case clears only WorkerSession, and rollback leaves quarantine unchanged.",
    "AC-3C-11: competing stop/clearance writers, context cancellation and crash boundaries preserve committed intent/winner, reject stale confirmation, prevent duplicate /kill and preserve DELIVERY_OUTCOME_UNKNOWN.",
    "AC-3C-12: trusted boundary rejects absent or wrong-scope principal; behavior tests use fake AO and fake principal only as library evidence, not host authentication proof."
  ],
  "verification_requests": [
    {"id": "VR-P03-003C-STOP", "profile_id": "go-test-p03-003c", "parameters": {"package": "./internal/stop/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003C-DISPATCH", "profile_id": "go-test-p03-003c", "parameters": {"package": "./internal/dispatch/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003C-STORE", "profile_id": "go-test-p03-003c", "parameters": {"package": "./internal/store/...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300},
    {"id": "VR-P03-003C-ALL", "profile_id": "go-test-p03-003c", "parameters": {"package": "./...", "flags": ["-v", "-race"]}, "cwd": ".", "timeout_seconds": 300}
  ],
  "required_evidence": [
    "git_diff_base_to_implementation_and_whitelist",
    "git_diff_check_exit_code_zero",
    "behavior_test_logs_and_exit_codes",
    "race_detector_results",
    "single_effect_call_counts_and_crash_boundary_state",
    "stop_and_clearance_CAS_audit_rollback_results",
    "lineage_specific_Class_A_B_C_evidence_and_audit_chain",
    "host_principal_dependency_status"
  ],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [
    "Any required change outside allowed_scope, new DDL/token/TaskState edge, or contradiction in accepted ADR-016/addendum requires escalation before coding.",
    "No trusted host principal or injected kill timeout is available and implementation would enable linked stop or administrative clearance instead of failing closed.",
    "AOAdapter cannot distinguish the required HTTP 200, 404, definite failure and ambiguous/protocol outcome without a change to forbidden internal/ao scope.",
    "Stop outcome, D12 closure and D6 clearance cannot meet their atomic CAS/audit boundary with approved v4 schema.",
    "Implementation would require 3D scanner/poller, daemon bootstrap, live AO call, default operational timeout or blind /kill retry.",
    "Verification profile/catalog remains unapproved at release or semantic validation with the approved catalog fails."
  ]
}
```

## 2. API và ownership sau merge

| Hiện có tại `246f74e` | Phần 3C phải bổ sung hoặc siết |
|---|---|
| AOAdapter `StopWorker(sessionID)`, `GetWorkerStatus(sessionID)`; `StopWorker` xác nhận HTTP 200 và response identity, `ao.ErrNotFound` biểu thị 404. | Dùng qua interface trong `internal/stop/coordinator.go`; không sửa `internal/ao/**`, không thêm generation vào wire `/kill`. |
| Store `CreateStopOperation`, `GetStopOperation`, `UpdateStopOperationStage`, `StopStageUpdate`. | `ReserveStopOperation` phải commit unlinked intent + `STOP_OPERATION_REQUESTED` audit; chặn đường `CreateStopOperation` generic bỏ qua audit. `CommitStopCallAccepted` CAS stage/resolution, call time/deadline/audit. `CommitStopOutcome` CAS stage/resolution và D12 closure/audit của live attempt trong một transaction; không ghép hai transaction hiện hữu. |
| Store `ClaimRestoreCleanupWithStop`, `GetPairRestoreOperation`, `ResolveAmbiguousRestore`; dispatch `OperatorBoundary` và `ClaimRestoreCleanup`. | Linked stop dùng intent đã commit bởi 3B; 3C bổ sung trusted stop/clearance boundary và, nếu cần, mở rộng Tx D audit để mô tả evidence/basis từng lineage. Không nới 3A snapshot check hoặc tự clear quarantine khi Tx D thắng. |
| Store `GetWorkerSessionByPair`, `GetTaskAttempt`, `AtomicTerminalTransition`; generic `UpdateWorkerSessionStatusAndQuarantine` không cho đổi quarantine. | `ClearQuarantineWithEvidence` mới làm D6 CAS exact WorkerSession + mọi quarantined attempt, kiểm chứng Class A/B/C và append audit từng đối tượng trong một transaction; giữ generic update bị khóa. |

`internal/domain/session_lifecycle.go` chỉ bổ sung input/result model và approved audit literals đã có trong ADR registry; không tạo token mới. `internal/store/session_lifecycle.go`, `restore_transactions.go` và `atomic_transitions.go` là file hiện hữu phải sửa guard/atomic path, không chỉ thêm file mới. `internal/dispatch/coordinator.go` chỉ được sửa để nối trusted boundary đã có với linked stop; package `internal/stop` sở hữu stop effect và single-observation orchestration. `internal/poller`, daemon/bootstrap, AO adapter và schema nằm ngoài scope 3C.

## 3. Transaction, effect và recovery boundary

1. **Tx S0 — intent:** Store CAS exact Pair/session/generation và loại trừ mọi stop chưa terminal trong cùng write transaction; kiểm tra purpose, attempt snapshot và restore ownership; unlinked stop insert `STOP_REQUESTED` + audit, linked stop dùng `ClaimRestoreCleanupWithStop` của 3B. Chỉ caller thắng reservation mới nhận effect permit; replay cùng operation là read-only và không gọi `/kill`. CAS thua/audit fail rollback toàn bộ. Không gọi AO trong transaction.
2. **Precheck và một effect:** `GetWorkerStatus` xác nhận exact session/opaque generation và target chưa terminated. `StopWorker` gửi wire chỉ có `sessionId`; generation/purpose là metadata Supervisor. Nếu precheck mâu thuẫn, ghi D11 disposition không gọi `/kill`. Sau intent commit, chỉ một caller được quyền effect; crash/commit ambiguity giữ intent, 3D sau này chỉ phân loại, không reissue.
3. **Tx S1 — call outcome:** Validated HTTP 200 mới CAS `STOP_REQUESTED → STOP_CALL_SUCCEEDED`, ghi `call_completed_at`, deadline từ timeout injected và audit cùng transaction. 404 chỉ ghi `STOP_TARGET_ABSENT`, giữ quarantine. Definite failure, invalid 200/transport unknown, context cancellation được phân loại fail-closed theo ADR D11; nếu commit/containment thất bại, reread durable row/audit, giữ winner hoặc intent chưa giải quyết, tuyệt đối không gửi lại effect.
4. **Tx S2 — single observation:** Một `GetWorkerStatus` tại mỗi lời gọi observation, không loop/ticker. Exact `isTerminated=true`, session/generation và thời điểm quan sát `termination_confirmed_at < confirmation_deadline_at` mới tạo `STOP_TERMINATION_CONFIRMED`; equality/late observation thành `STOP_CONFIRMATION_TIMEOUT`. `STOP_CALL_SUCCEEDED` phải là bằng chứng bền vững trước confirmation. Mỗi resolution giữ nguyên stage wire khi không có positive termination.
5. **Tx S3 — purpose outcome:** Live stop ghép stop resolution với `RUNNING → FAILED`, closure exact current attempt, disposition và `TASK_STATE_TRANSITION`/stop audit trong một transaction. Cleanup/maintenance không đổi TaskState hoặc `ended_at`. `WORKER_STOPPED` chỉ cho live stop đủ sáu điều kiện D11. Stale writer không được ghi đè terminal result.
6. **Tx D rồi Tx D6:** Với linked restore, Tx D CAS operation và audit basis tổng thể, ghi provenance runtime và từng attempt; không clear quarantine. Tx D6 riêng kiểm tra exact lineage, stop proof hoặc verified Class B/C acceptance cho *từng* quarantine row; CAS tất cả WorkerSession/attempt và append `QUARANTINE_RESOLVED_PHYSICAL`/`QUARANTINE_RESOLVED_ADMINISTRATIVE` tương ứng. Một CAS/audit fail rollback toàn bộ Tx D6, còn Tx D đã resolved và quarantine vẫn khóa. Không có attempt thì chỉ xử lý WorkerSession. Dispatch `DELIVERY_OUTCOME_UNKNOWN` không đổi.

**Crash matrix:** trước Tx S0: không có stop/effect; sau Tx S0 trước effect: giữ `STOP_REQUESTED`, không reissue sau restart; sau effect mất response hoặc Tx S1 lỗi: giữ/reread durable intent, xử lý `STOP_CALL_OUTCOME_UNKNOWN` khi có thể chứng minh CAS, không retry; sau Tx S1 trước S2: dùng deadline cũ, 3D chỉ kích hoạt phân loại; sau Tx D trước Tx D6: restore resolved nhưng quarantine còn; Tx D6 rollback: retry CAS local với cùng evidence/authority, zero AO effect. Scope 3C không gồm startup scanner/poller.

## 4. Kiểm chứng dự kiến và dependency

Behavior tests dùng fake AO và SQLite test DB: purpose matrix, HTTP 200/404/transport/protocol, no duplicate effect, concurrent writers, crash tại từng boundary, strict deadline trước/đúng/sau và độ chính xác nanosecond, old/new mixed evidence, no-attempt maintenance, wrong attempt snapshot, Class B/C authority, Tx D trước D6, audit/CAS rollback và audit chain. Không chạy AO thật hoặc database người dùng. Không gọi các scenario này là implementation PASS trước khi có code/evidence.

`go-test-p03-003c` là **catalog đề xuất, chưa được Supervisor phê duyệt hoặc ValidateRaw**: `CwdPolicy=worktree_root`, `MaxTimeoutSeconds=300`, parameter object `additionalProperties=false`, required `package`/`flags`, package enum `./internal/stop/...`, `./internal/dispatch/...`, `./internal/store/...`, `./...`, flags đúng `[-v,-race]`. `timeout_seconds` là policy verification, không phải `SUPERVISOR_KILL_STOP_TIMEOUT` và không chứng minh P04 runner. JSON TaskContract đã qua full schema shape check trong lượt soạn draft; semantic validation với catalog được duyệt vẫn cần trước release, không dùng catalog nới lỏng. Runtime còn phụ thuộc authenticated host principal và timeout policy injected; thư viện phải fail closed khi thiếu hai dependency này. Chưa xác định blocker thiết kế mới từ tài liệu đã duyệt; nếu implementation đòi schema/token hoặc thay AO wire, áp dụng `stop_conditions` để báo External Supervisor.

**Gate:** `TASK_P03_003C/3D=NOT_RELEASED`; `P03_CODE=HELD_PENDING_TASK_P03_003C_CONTRACT_RELEASE`; `ACTIVE_GATE=TASK_P03_003C_CONTRACT_PLANNING`; `AUTOMATIC_RESTORE=DISABLED`; `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`.
