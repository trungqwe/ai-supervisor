# RELEASED TASK CONTRACT: TASK-P03-003C

> **Status:** `TASK_P03_003C=RELEASED`; `TASK_P03_003C_IMPLEMENTATION=IN_PROGRESS`. JSON contract giữ nguyên candidate đã phê duyệt; chỉ 3C được cấp quyền code theo `allowed_scope`.
> **Code base SHA:** `246f74eabd09bbc2ea66624ee217035167824a86`. Release artifact được giao riêng; không nhúng SHA tự tham chiếu.
> **Authority:** ADR-016 accepted, addendum approved, parent decomposition §3.3. `AUTOMATIC_RESTORE=DISABLED`; host/bootstrap phải cung cấp verified principal trước khi enable linked stop hoặc administrative clearance.

## 1. JSON TaskContract đã release

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
    "GetWorkerStatus observation is outside SQLite transactions. For each RUNNING_ATTEMPT_STOP terminal outcome, stop stage/resolution, RUNNING-to-FAILED D12 attempt closure, disposition, WorkerSession/attempt quarantine and all stop/state/clearance audit commit in ONE transaction under exact current attempt/session/generation CAS; no separate confirmation commit may precede closure. WORKER_STOPPED requires all six D11 conditions including strict termination_confirmed_at before deadline.",
    "QUARANTINE_CLEANUP requires the immutable snapshot of its own closed FAILED/HUMAN_REQUIRED attempt; PAIR_MAINTENANCE has no fake attempt. Neither cleanup nor maintenance changes TaskState or ended_at.",
    "Linked stop confirmation records only evidence. Restore Tx D resolution remains distinct from D6 clearance; Class A requires positive physical proof of current runtime and every quarantined old lineage, while Class B/C require verified operator risk acceptance for each unresolved lineage. A persisted STOP_REQUESTED or GetStopOperation read is never an effect permit.",
    "Clear WorkerSession and every affected TaskAttempt quarantine through one exact-lineage CAS/audit transaction only after required resolution; rollback leaves all quarantine gates intact, and clearance never rewrites DELIVERY_OUTCOME_UNKNOWN.",
    "Trusted operator boundary fails closed without host-authenticated principal and approved scope; fake test principals do not enable runtime restore/linked stop. AUTOMATIC_RESTORE remains DISABLED.",
    "3D owns startup scanner and poller. 3C exposes bounded single-observation/reconciliation operations without ticker, background loop, daemon bootstrap, SSE, synthetic heartbeat or live AO testing."
  ],
  "acceptance_criteria": [
    "AC-3C-01: purpose-aware unlinked STOP_REQUESTED reservation and audit commit before one AO /kill; only the caller receiving unambiguous success from its own new reservation/atomic 3B ClaimRestoreCleanupWithStop call receives one-use effect permission. Existing STOP_REQUESTED/GetStopOperation, same-operation/same-Pair losers, restart replay and commit ambiguity never grant effect; duplicate or audit/CAS failure leaves zero extra /kill calls.",
    "AC-3C-02: exact pre-call GET session/generation prevents /kill on mismatch, absent or already terminated target; tests acknowledge the remaining session-scoped TOCTOU window and never assert a generation fence.",
    "AC-3C-03: validated HTTP 200 commits STOP_CALL_SUCCEEDED, call_completed_at, immutable injected confirmation_deadline_at and STOP_OPERATION_CALL_SUCCEEDED audit; HTTP 200 alone never records termination or WORKER_STOPPED.",
    "AC-3C-04: 404 records STOP_TARGET_ABSENT/absence audit and retains quarantine; definite upstream rejection and ambiguous transport/invalid response use their exact D11 stage/resolution/audit, never blind reissue.",
    "AC-3C-05: single status observation within the persisted deadline confirms termination only for exact session/generation with isTerminated=true; before/equal/after deadline and restart replay preserve strict nanosecond comparison and original deadline.",
    "AC-3C-06: GET observation occurs outside SQLite transaction; every terminal RUNNING_ATTEMPT_STOP outcome commits D11 stage/resolution, D12 RUNNING-to-FAILED/current-attempt closure, exact disposition, both quarantine states and all stop/state/clearance audits in ONE transaction. Inject failure at stop CAS, attempt CAS, quarantine update and every audit insert: no partial confirmation, closure or clearance survives; only all six D11 conditions permit WORKER_STOPPED.",
    "AC-3C-07: QUARANTINE_CLEANUP and PAIR_MAINTENANCE produce zero TaskState/ended_at mutation; linked new-generation maintenance has restore link/authority but no attempt, and cannot clear old-generation attempt quarantine.",
    "AC-3C-08: Class A clearance checks positive D11 proof for current runtime and every quarantined attempt; old-only/new-only evidence or mismatched lineage cannot release WorkerSession or attempt.",
    "AC-3C-09: Class B uses 404 absence plus verified administrative risk acceptance, Class C uses explicit verified human risk acceptance; each unresolved lineage and current runtime has scoped basis/evidence/audit, never false physical evidence.",
    "AC-3C-10: linked restore Tx D resolves operation before separate D6 clearance; D6 performs exact WorkerSession and every-attempt CAS plus per-lineage audit atomically; no-attempt case clears only WorkerSession, and rollback leaves quarantine unchanged.",
    "AC-3C-11: two callers with the same operation or Pair, pre-existing linked intent, restart replay, context cancellation and ambiguous claim commit yield at most one /kill only for the unambiguous new-intent winner; all other paths fail closed. Competing stop/clearance writers reject stale confirmation, preserve committed winner and DELIVERY_OUTCOME_UNKNOWN.",
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
    "Semantic validation with the approved go-test-p03-003c catalog fails or host verification metadata differs at release; library validation alone never proves the P04 runner exists."
  ]
}
```

## 2. API và ownership sau merge

| Hiện có tại `246f74e` | Phần 3C phải bổ sung hoặc siết |
|---|---|
| AOAdapter `StopWorker(sessionID)`, `GetWorkerStatus(sessionID)`; `StopWorker` xác nhận HTTP 200 và response identity, `ao.ErrNotFound` biểu thị 404. | Dùng qua interface trong `internal/stop/coordinator.go`; không sửa `internal/ao/**`, không thêm generation vào wire `/kill`. |
| Store `CreateStopOperation`, `GetStopOperation`, `UpdateStopOperationStage`, `StopStageUpdate`. | `ReserveStopOperation` phải commit unlinked intent + `STOP_OPERATION_REQUESTED` audit; chặn đường `CreateStopOperation` generic bỏ qua audit. `CommitStopCallAccepted` CAS stage/resolution, call time/deadline/audit. `CommitStopOutcome` CAS stage/resolution và D12 closure/audit của live attempt trong một transaction; không ghép hai transaction hiện hữu. |
| Store `ClaimRestoreCleanupWithStop`, `GetPairRestoreOperation`, `ResolveAmbiguousRestore`; dispatch `OperatorBoundary` và `ClaimRestoreCleanup`. | 3C gọi chính `ClaimRestoreCleanupWithStop` cho intent **mới** dưới verified operator boundary; chỉ return thành công rõ ràng của lần claim này cấp quyền effect dùng một lần trong call stack. Intent đã tồn tại trước lời gọi, `GetStopOperation`, CAS thua hoặc commit ambiguity đều fail-closed, không cấp quyền `/kill`. Không nới 3A snapshot check hoặc tự clear quarantine khi Tx D thắng. |
| Store `GetWorkerSessionByPair`, `GetTaskAttempt`, `AtomicTerminalTransition`; generic `UpdateWorkerSessionStatusAndQuarantine` không cho đổi quarantine. | `ClearQuarantineWithEvidence` mới làm D6 CAS exact WorkerSession + mọi quarantined attempt, kiểm chứng Class A/B/C và append audit từng đối tượng trong một transaction; giữ generic update bị khóa. |

`internal/domain/session_lifecycle.go` chỉ bổ sung input/result model và approved audit literals đã có trong ADR registry; không tạo token mới. `internal/store/session_lifecycle.go`, `restore_transactions.go` và `atomic_transitions.go` là file hiện hữu phải sửa guard/atomic path, không chỉ thêm file mới. `internal/dispatch/coordinator.go` chỉ được sửa để nối trusted boundary đã có với linked stop; package `internal/stop` sở hữu stop effect và single-observation orchestration. `internal/poller`, daemon/bootstrap, AO adapter và schema nằm ngoài scope 3C.

## 3. Transaction, effect và recovery boundary

1. **Tx S0 — intent và quyền effect:** Store CAS exact Pair/session/generation và loại trừ mọi stop chưa terminal trong cùng write transaction; kiểm tra purpose, authority, attempt snapshot và restore ownership. Unlinked `ReserveStopOperation` insert **mới** `STOP_REQUESTED` + audit; linked coordinator 3C tự gọi `ClaimRestoreCleanupWithStop` của 3B để CAS ownership + insert stop + hai audit atomic. Chỉ return thành công rõ ràng từ lần insert mới của chính caller cấp một quyền effect dùng một lần, giữ trong call stack, không persist thêm token. Same-operation/same-Pair loser, `GetStopOperation`, row `STOP_REQUESTED` đã có trước lời gọi, restart replay và kết quả commit mơ hồ **không** cấp quyền, kể cả khi reread thấy row của mình. CAS thua/audit fail rollback toàn bộ. Không gọi AO trong transaction.
2. **Precheck và một effect:** Caller giữ quyền từ Tx S0 mới gọi `GetWorkerStatus` ngoài transaction để xác nhận exact session/opaque generation và target chưa terminated, rồi dùng quyền một lần gọi `StopWorker`; wire chỉ có `sessionId`, generation/purpose là metadata Supervisor. Nếu precheck mâu thuẫn, ghi D11 outcome, tiêu quyền mà không gọi `/kill`. Không chuyển quyền cho caller khác. Crash sau S0, lỗi commit không phân biệt được và linked intent đã có trước khi 3C nhận request đều giữ khóa, chờ phân loại/operator; 3D không reissue. Verified principal là điều kiện cho **mọi** cleanup session-scoped, linked lẫn unlinked; stop row không tự cấp quyền operator.
3. **Tx S1 — call outcome:** Validated HTTP 200 mới CAS `STOP_REQUESTED → STOP_CALL_SUCCEEDED`, ghi `call_completed_at`, deadline từ timeout injected và audit cùng transaction. 404 chỉ ghi `STOP_TARGET_ABSENT`, giữ quarantine. Definite failure, invalid 200/transport unknown, context cancellation được phân loại fail-closed theo ADR D11; nếu commit/containment thất bại, reread durable row/audit, giữ winner hoặc intent chưa giải quyết, tuyệt đối không gửi lại effect.
4. **Observation ngoài Tx S2:** Mỗi lời gọi chỉ một `GetWorkerStatus` ngoài transaction, không loop/ticker. Exact `isTerminated=true`, session/generation và `termination_confirmed_at.Before(confirmation_deadline_at)` mới đủ điều kiện `STOP_TERMINATION_CONFIRMED`; equality/late thành `STOP_CONFIRMATION_TIMEOUT`. `STOP_CALL_SUCCEEDED` phải là bằng chứng bền vững trước confirmation. Observation không tự commit resolution hoặc clearance.
5. **Tx S2 — terminal outcome nguyên tử:** Với `RUNNING_ATTEMPT_STOP`, **cùng một transaction** CAS stop stage/resolution và exact current attempt/session/generation, D12 `RUNNING → FAILED` + `ended_at`/closure, disposition, WorkerSession/attempt quarantine và mọi stop/`TASK_STATE_TRANSITION`/clearance audit. Không có Tx confirmation độc lập rồi Tx closure; lỗi ở bất kỳ CAS/audit nào rollback cả stop confirmation lẫn closure/quarantine. Cleanup/maintenance ghi stop evidence, không đổi TaskState hoặc `ended_at`; linked cleanup không clear trước Tx D. `WORKER_STOPPED` chỉ cho live stop đủ sáu điều kiện D11. Stale writer không ghi đè terminal result.
6. **Tx D rồi Tx D6:** Với linked restore, Tx D CAS operation và audit basis tổng thể, ghi provenance runtime và từng attempt; không clear quarantine. Tx D6 riêng kiểm tra exact lineage, stop proof hoặc verified Class B/C acceptance cho *từng* quarantine row; CAS tất cả WorkerSession/attempt và append `QUARANTINE_RESOLVED_PHYSICAL`/`QUARANTINE_RESOLVED_ADMINISTRATIVE` tương ứng. Một CAS/audit fail rollback toàn bộ Tx D6, còn Tx D đã resolved và quarantine vẫn khóa. Không có attempt thì chỉ xử lý WorkerSession. Dispatch `DELIVERY_OUTCOME_UNKNOWN` không đổi.

**Crash matrix:** trước Tx S0: không có stop/effect; sau Tx S0 trước effect: giữ `STOP_REQUESTED`, quyền trong call stack mất khi crash, không reissue sau restart; Tx S0 commit mơ hồ: reread chỉ để containment, không tự cấp quyền effect; sau effect mất response hoặc Tx S1 lỗi: giữ/reread durable intent, xử lý `STOP_CALL_OUTCOME_UNKNOWN` khi có thể chứng minh CAS, không retry; sau Tx S1 trước observation: dùng deadline cũ, 3D chỉ kích hoạt phân loại; Tx S2 terminal rollback: cả confirmation, closure, quarantine và audit rollback; sau Tx D trước Tx D6: restore resolved nhưng quarantine còn; Tx D6 rollback: retry CAS local với cùng evidence/authority, zero AO effect. Scope 3C không gồm startup scanner/poller.

### Purpose × outcome (ADR-016 D5/D6/D11/D12, addendum §5)

Ký hiệu: `L` = `RUNNING_ATTEMPT_STOP`, `Q` = `QUARANTINE_CLEANUP` của attempt đã đóng, `M` = `PAIR_MAINTENANCE` không gắn attempt; `WS/A` = quarantine WorkerSession/attempt; `giữ` nghĩa là không clear risk chưa có bằng chứng. Với L, mọi outcome terminal D11 đều đi qua **một Tx S2** cho stop + D12 + quarantine + audit. Với Q/M, stop evidence không tự chứng minh tất cả lineage; clearance chỉ qua D6 theo exact lineage. Linked Q/M luôn phải chờ Tx D rồi Tx D6, kể cả khi stop xác nhận thành công. `TASK_STATE_TRANSITION` chỉ phát khi có cạnh TaskState thực.

| Purpose → observation | Stage / resolution | TaskState / `ended_at` | Disposition | WS / A quarantine | Audit / transaction | Điều kiện tiếp tục |
|---|---|---|---|---|---|---|
| L → HTTP 200 rồi exact terminated **trước** deadline | `STOP_TERMINATION_CONFIRMED` / `TERMINATION_CONFIRMED` | `RUNNING→FAILED` / set | `WORKER_STOPPED` chỉ khi đủ 6 D11 | Clear A đúng lineage; WS chỉ CLEAN khi mọi risk khác của Pair đã giải | `STOP_OPERATION_CONFIRMED`, `TASK_STATE_TRANSITION`, clearance audit nếu clear; **Tx S2** | Chỉ sau tất cả Pair guard/D6 còn lại |
| L → 404 | `STOP_TARGET_ABSENT` / `STOP_TARGET_ABSENT` | `RUNNING→FAILED` / set | `SESSION_ABSENT` | WS/A quarantine | `STOP_OPERATION_TARGET_ABSENT`, `TASK_STATE_TRANSITION`; **Tx S2** | Class B operator + D6, không tự clear |
| L → generation mismatch | Giữ `STOP_REQUESTED` hoặc `STOP_CALL_SUCCEEDED` / `STOP_GENERATION_MISMATCH` | `RUNNING→FAILED` / set | `STOP_GENERATION_MISMATCH` | WS/A quarantine | Stop resolution + `TASK_STATE_TRANSITION`; **Tx S2** | Exact lineage Class A/B/C + D6 |
| L → đã terminated nhưng kill chưa được chứng minh | Giữ `STOP_REQUESTED` / `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED` | `RUNNING→FAILED` / set | `WORKER_TERMINATION_UNKNOWN` | WS/A quarantine | Stop resolution + `TASK_STATE_TRANSITION`; **Tx S2** | Không coi GET là kill proof; D6 sau basis hợp lệ |
| L → đúng/beyond deadline | Giữ `STOP_CALL_SUCCEEDED` / `STOP_CONFIRMATION_TIMEOUT` | `RUNNING→FAILED` / set | `STOP_CONFIRMATION_TIMEOUT` | WS/A quarantine | `STOP_CONFIRMATION_TIMEOUT`, `TASK_STATE_TRANSITION`; **Tx S2** | Human reconciliation, không đổi deadline |
| L → definite failure | `STOP_CALL_FAILED` / `STOP_CALL_FAILED` | `RUNNING→FAILED` / set | `WORKER_TERMINATION_UNKNOWN` | WS/A quarantine | `STOP_OPERATION_CALL_FAILED`, `TASK_STATE_TRANSITION`; **Tx S2** | Class C hoặc proof khác, không retry |
| L → ambiguous call/invalid response | Giữ `STOP_REQUESTED` / `STOP_CALL_OUTCOME_UNKNOWN` | `RUNNING→FAILED` / set | `STOP_CALL_OUTCOME_UNKNOWN` | WS/A quarantine | `STOP_OPERATION_CALL_OUTCOME_UNKNOWN`, `TASK_STATE_TRANSITION`; **Tx S2** | Reread winner, không reissue |
| L → restart, effect chưa rõ | Giữ `STOP_REQUESTED` / `STOP_REISSUE_REQUIRES_HUMAN` nếu target còn live; observation khác theo D13 | Giữ durable TaskState; D13 xử lý cạnh D12 nếu có outcome terminal / chỉ set khi có cạnh | Giữ hoặc outcome D11 tương ứng | WS/A giữ/impose quarantine | Audit reconciliation; D13 transaction tương ứng, **không thuộc scanner 3C** | Operator; không reissue |
| Q → exact terminated trước deadline (`now < confirmation_deadline_at`) | `STOP_TERMINATION_CONFIRMED` / `TERMINATION_CONFIRMED` | Giữ `FAILED/HUMAN_REQUIRED` / giữ | Giữ disposition lịch sử | Linked: WS/A giữ; unlinked: D6 clear đúng lineage nếu đủ proof, WS chỉ sau mọi risk | `STOP_OPERATION_CONFIRMED`; stop evidence Tx S2, unlinked D6 theo lineage, linked Tx D rồi D6 | Không mở lane trước clearance |
| M → exact terminated trước deadline (`now < confirmation_deadline_at`) | `STOP_TERMINATION_CONFIRMED` / `TERMINATION_CONFIRMED` | Không task/attempt / N/A | N/A | Linked: WS và mọi A cũ giữ; unlinked: D6 chỉ WS sau mọi lineage | `STOP_OPERATION_CONFIRMED`; stop evidence Tx S2, D6 riêng theo basis | New-generation proof không clear old A |
| Q/M → 404 | `STOP_TARGET_ABSENT` / `STOP_TARGET_ABSENT` | Giữ / giữ | Giữ / N/A | WS/A giữ | `STOP_OPERATION_TARGET_ABSENT`; stop Tx S2; Class B D6 riêng | Verified Class B authority + evidence từng lineage |
| Q/M → generation mismatch | Giữ wire stage / `STOP_GENERATION_MISMATCH` | Giữ / giữ | Giữ / N/A | WS/A giữ | Stop resolution audit; Tx S2; D6 riêng | Không suy termination generation cũ |
| Q/M → already terminated, effect unproven | Giữ `STOP_REQUESTED` / `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED` | Giữ / giữ | Giữ / N/A | WS/A giữ | Stop resolution audit; Tx S2; D6 riêng | GET không cấp physical basis |
| Q/M → đúng/beyond deadline | Giữ `STOP_CALL_SUCCEEDED` / `STOP_CONFIRMATION_TIMEOUT` | Giữ / giữ | Giữ / N/A | WS/A giữ | `STOP_CONFIRMATION_TIMEOUT`; Tx S2; D6 riêng | Class C nếu không có proof hợp lệ |
| Q/M → definite failure | `STOP_CALL_FAILED` / `STOP_CALL_FAILED` | Giữ / giữ | Giữ / N/A | WS/A giữ | `STOP_OPERATION_CALL_FAILED`; Tx S2; D6 riêng | Không retry; verified authority cho admin basis |
| Q/M → ambiguous/invalid response | Giữ `STOP_REQUESTED` / `STOP_CALL_OUTCOME_UNKNOWN` | Giữ / giữ | Giữ / N/A | WS/A giữ | `STOP_OPERATION_CALL_OUTCOME_UNKNOWN`; Tx S2; D6 riêng | Không reissue; operator reconciliation |
| Q/M → restart, effect chưa rõ | Giữ `STOP_REQUESTED` / `STOP_REISSUE_REQUIRES_HUMAN` nếu target live; outcome khác theo D13 | Giữ / giữ | Giữ / N/A | WS/A giữ | Audit reconciliation; D13 transaction, ngoài scanner 3C | Human action; không reissue |

Mỗi stop resolution không có audit literal riêng dùng event D11/registry tương ứng và details có cấu trúc, không tạo token mới. Class A phải có positive D11 evidence đúng runtime và từng quarantined attempt; Class B (404) hoặc Class C dùng `ADMINISTRATIVE_RISK_RESOLUTION`/`ADMINISTRATIVE_RISK_ACCEPTED` với verified principal, basis và audit **riêng từng lineage**. Administrative evidence không được ghi là physical. Stop row, kể cả unlinked Q/M, không thay thế authorization cho session-scoped `/kill` hay D6. Quan sát sai identity/protocol hoặc query unavailable fail-closed theo D11/D13, giữ/impose quarantine; không suy physical proof từ lỗi transport.

## 4. Kiểm chứng dự kiến và dependency

Behavior tests dùng fake AO và SQLite test DB: toàn bộ hàng purpose × outcome phía trên; HTTP 200/404/transport/protocol; hai caller cùng operation/Pair, linked intent tồn tại trước call, restart replay và ambiguous claim commit đều assert số `/kill` không tăng; crash tại từng boundary; strict deadline trước/đúng/sau và độ chính xác nanosecond; old/new mixed evidence, no-attempt maintenance, wrong attempt snapshot, Class B/C authority, Tx D trước D6 và audit chain. Fault injection lần lượt tại stop CAS, attempt CAS, WorkerSession/attempt quarantine update và từng audit insert phải chứng minh live terminal confirmation + D12 closure rollback đồng thời, không có partial terminal state. Không chạy AO thật hoặc database người dùng. Không gọi các scenario này là implementation PASS trước khi có code/evidence.

Supervisor đã phê duyệt **policy metadata** `go-test-p03-003c`: `CwdPolicy=worktree_root`, `MaxTimeoutSeconds=300`, parameter object `additionalProperties=false`, required `package`/`flags`, package enum `./internal/stop/...`, `./internal/dispatch/...`, `./internal/store/...`, `./...`, flags đúng hai phần tử theo thứ tự `[-v,-race]`. Đây là catalog đầu vào cho `TaskContractValidator.ValidateRaw`, không chứng minh host P04 runner đã được triển khai. `timeout_seconds` là policy verification, không phải `SUPERVISOR_KILL_STOP_TIMEOUT`. Kết quả full-schema/semantic validation của draft và candidate được ghi trong evidence bàn giao lượt này; không dùng catalog nới lỏng. Runtime còn phụ thuộc authenticated host principal và timeout policy injected; thư viện phải fail closed khi thiếu hai dependency này. Chưa xác định blocker thiết kế mới từ tài liệu đã duyệt; nếu implementation đòi schema/token hoặc thay AO wire, áp dụng `stop_conditions` để báo External Supervisor.

**Validation evidence của candidate:** `go run p03_3c_contract_validate_tmp.go` (harness tạm, đã xóa sau kiểm chứng), đọc `docs/schemas/task-contract.schema.json`, gọi `CompileSchema` + `ParseAndValidateRaw` (full schema) và `TaskContractValidator.ValidateRaw` với in-memory catalog đúng metadata ở trên. Draft/candidate đều PASS, JSON đồng nhất; bốn mutation `timeout=301`, `flags=[-v,-exec]`, `package=./internal/ao/...`, `cwd=../escape` đều REJECTED; exit code cuối `0`. Thử nghiệm xác nhận validation library/catalog, **không** xác nhận executable host runner P04 hoặc runtime authentication.

**Gate:** `TASK_P03_003C/3D=NOT_RELEASED`; `P03_CODE=HELD_PENDING_TASK_P03_003C_CONTRACT_RELEASE`; `ACTIVE_GATE=TASK_P03_003C_CONTRACT_PLANNING`; `AUTOMATIC_RESTORE=DISABLED`; `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`.
