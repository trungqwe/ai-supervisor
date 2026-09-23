# DRAFT ADR-016 Addendum: Restore ownership và dispatch pre-send resolution

> **Status**: `DRAFT_PENDING_EXTERNAL_APPROVAL` — không sửa hoặc thay thế ADR-016 đã accepted.
> **Proposal**: `docs/proposals/PROPOSAL-P03-003-restore-and-pre-send-semantics.md` — direction được Supervisor phê duyệt; DDL/transition bên dưới chưa được phê duyệt.
> **Baseline**: `95bf8e3329d0261eeafdcd035cbe97dbe26b12ce`; TASK-P03-003B contract `base_sha=b8b0c95576d87677e8d48210d9838cc2f599752a`.
> **Pinned AO**: `Untrivial-ai/agent-orchestrator` v0.13.0, `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.
> **Gate**: `TASK_P03_003B=NOT_RELEASED`; `P03_CODE=HELD_PENDING_TASK_P03_003B_CONTRACT_RELEASE`; `ACTIVE_GATE=TASK_P03_003B_CONTRACT_PLANNING`; `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL=OPEN`; bảo lưu `DESIGN_BLOCKER_3D_STARTUP_WIRING`.

## 1. Authority và quyết định đang xin duyệt

Supervisor **đã duyệt hướng** durable restore operation + Pair guard, chỉ operator-authorized restore trên pinned AO, `AUTOMATIC_RESTORE=DISABLED`, cấm retry `/restore` khi outcome không xác định, và semantics `DELIVERY_OUTCOME_UNKNOWN`: terminal cho dispatch operation, độc lập với quarantine clearance. Đây chưa phải phê duyệt DDL, transition, quyền operator hoặc release 3B. Addendum này xin phê duyệt các phần ấy; không tự thay ADR.

ADR-016 D3 giữ WorkerSession hiện hữu, D4 giữ bound snapshot, D5–D6 quy định unknown send và Class A/B/C, D7/§22 quy định pre-send, D11 phân biệt stop purpose, D12 bắt buộc atomic `DISPATCHED -> FAILED`/attempt closure/audit, D13 giao startup scanner. ADR §4 và token registry đã có `WORKER_TERMINATION_UNKNOWN` và `STALE_EXECUTION_GENERATION`; chúng không phải token mới. `docs/02_REQUIREMENTS.md` FR-005/FR-006/NFR-003/NFR-005, `docs/12_UPSTREAM_INTEGRATION.md`, `docs/14_FAILURE_RECOVERY.md` REC-002 và SOURCE_REGISTRY/REUSE_MATRIX giữ AO làm process/worktree owner.

Pinned AO `openapi.yaml:5245-5274,10693-10712` + `controllers/sessions.go:195,1250-1261` xác nhận `/restore` nhận session ID, trả 200/404/409/500, không nhận client operation/idempotency key. `service/session/service.go:580-590` gọi manager; `session_manager/manager.go:1961-2025,2212-2365,4702-4800` có `native`/`saved_prompt`/`fresh` và có thể tạo effect trước lỗi response. `session_input.go:100-129` chỉ khóa agent operation trong AO process. `GET` thấy generation mới sau crash **không** chứng minh causal restore, mode hoặc prompt replay; không dùng cùng session ID như dedup key. AO 200 là call provenance, chưa chứng minh lane idle, old execution đã dừng hoặc quarantine được giải phóng.

## 2. Proposed schema V4 và token registry bổ sung

Migration mới, không sửa v3. DDL sau là **candidate**, mọi row được append/transition qua audited Store API; không cho delete operation/authorization đã dùng. `restore_authorizations` là one-shot capability record, được tạo bởi trusted Supervisor operator boundary, không nhận trực tiếp từ worker/TaskContract. `operation_id` được chọn trước khi cấp authorization.

```sql
CREATE TABLE restore_authorizations (
  authorization_id TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL UNIQUE,
  pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
  session_id TEXT NOT NULL,
  expected_generation TEXT NOT NULL,
  risk_scope TEXT NOT NULL CHECK (risk_scope IN
    ('POSSIBLE_PROMPT_REPLAY','PROMPT_REPLAY_AND_UNRESOLVED_EXECUTION')),
  authorized_principal TEXT NOT NULL,
  authorization_event_id TEXT NOT NULL UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
  issued_at TEXT NOT NULL,
  consumed_at TEXT,
  consumed_by_operation_id TEXT UNIQUE,
  UNIQUE (authorization_id,operation_id,pair_id,session_id,expected_generation),
  CHECK ((consumed_at IS NULL AND consumed_by_operation_id IS NULL) OR
         (consumed_at IS NOT NULL AND consumed_by_operation_id IS NOT NULL AND
          consumed_by_operation_id = operation_id))
);

CREATE TABLE pair_restore_operations (
  operation_id TEXT PRIMARY KEY,
  authorization_id TEXT NOT NULL UNIQUE REFERENCES restore_authorizations(authorization_id) ON DELETE RESTRICT,
  pair_id TEXT NOT NULL REFERENCES pairs(pair_id) ON DELETE RESTRICT,
  session_id TEXT NOT NULL,
  expected_generation TEXT NOT NULL,
  observed_generation TEXT,
  restore_mode TEXT CHECK (restore_mode IN ('native','saved_prompt','fresh')),
  stage TEXT NOT NULL CHECK (stage IN ('RESTORE_REQUESTED','RESTORE_CONFIRMED')),
  resolution_state TEXT NOT NULL CHECK (resolution_state IN
    ('IN_FLIGHT','RESTORE_OUTCOME_UNKNOWN','RESTORE_RECOVERY_CLAIMED',
     'RESTORE_CLEANUP_CLAIMED','RESTORE_RESOLVED')),
  resolution_basis TEXT CHECK (resolution_basis IN
    ('RESTORE_HTTP_200_CONFIRMED','PHYSICAL_EXECUTION_RESOLUTION',
     'ADMINISTRATIVE_RISK_RESOLUTION')),
  recovery_principal TEXT,
  recovery_claimed_at TEXT,
  requested_at TEXT NOT NULL,
  http_200_at TEXT,
  resolved_at TEXT,
  resolution_event_id TEXT UNIQUE REFERENCES audit_events(event_id) ON DELETE RESTRICT,
  version INTEGER NOT NULL DEFAULT 0 CHECK (version >= 0),
  FOREIGN KEY (authorization_id,operation_id,pair_id,session_id,expected_generation)
    REFERENCES restore_authorizations
      (authorization_id,operation_id,pair_id,session_id,expected_generation) ON DELETE RESTRICT,
  CHECK ((stage = 'RESTORE_CONFIRMED' AND
          http_200_at IS NOT NULL AND observed_generation IS NOT NULL AND restore_mode IS NOT NULL) OR
         (stage = 'RESTORE_REQUESTED' AND
          http_200_at IS NULL AND observed_generation IS NULL AND restore_mode IS NULL)),
  CHECK ((resolution_state = 'RESTORE_RESOLVED' AND
          resolved_at IS NOT NULL AND resolution_basis IS NOT NULL AND resolution_event_id IS NOT NULL) OR
         (resolution_state <> 'RESTORE_RESOLVED' AND
          resolved_at IS NULL AND resolution_basis IS NULL AND resolution_event_id IS NULL)),
  CHECK (resolution_basis <> 'RESTORE_HTTP_200_CONFIRMED' OR stage = 'RESTORE_CONFIRMED'),
  CHECK (resolution_state NOT IN ('RESTORE_RECOVERY_CLAIMED','RESTORE_CLEANUP_CLAIMED') OR
         (recovery_principal IS NOT NULL AND recovery_claimed_at IS NOT NULL))
);
CREATE UNIQUE INDEX idx_pair_restore_unresolved ON pair_restore_operations(pair_id)
  WHERE resolution_state <> 'RESTORE_RESOLVED';
CREATE INDEX idx_restore_stage_resolution ON pair_restore_operations(stage,resolution_state);

ALTER TABLE stop_operations ADD COLUMN restore_operation_id TEXT
  REFERENCES pair_restore_operations(operation_id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX idx_stop_restore_operation ON stop_operations(restore_operation_id)
  WHERE restore_operation_id IS NOT NULL;

CREATE TRIGGER trg_restore_auth_no_delete BEFORE DELETE ON restore_authorizations
BEGIN SELECT RAISE(ABORT,'restore authorization is append-only'); END;
CREATE TRIGGER trg_restore_op_no_delete BEFORE DELETE ON pair_restore_operations
BEGIN SELECT RAISE(ABORT,'restore operation is append-only'); END;
CREATE TRIGGER trg_restore_auth_immutable BEFORE UPDATE ON restore_authorizations
WHEN NEW.authorization_id IS NOT OLD.authorization_id
  OR NEW.operation_id IS NOT OLD.operation_id
  OR NEW.pair_id IS NOT OLD.pair_id
  OR NEW.session_id IS NOT OLD.session_id
  OR NEW.expected_generation IS NOT OLD.expected_generation
  OR NEW.risk_scope IS NOT OLD.risk_scope
  OR NEW.authorized_principal IS NOT OLD.authorized_principal
  OR NEW.authorization_event_id IS NOT OLD.authorization_event_id
  OR NEW.issued_at IS NOT OLD.issued_at
  OR (OLD.consumed_at IS NOT NULL AND
      (NEW.consumed_at IS NOT OLD.consumed_at OR
       NEW.consumed_by_operation_id IS NOT OLD.consumed_by_operation_id))
BEGIN SELECT RAISE(ABORT,'restore authorization immutable or already consumed'); END;
CREATE TRIGGER trg_restore_op_forward_only BEFORE UPDATE ON pair_restore_operations
WHEN NEW.operation_id IS NOT OLD.operation_id
  OR NEW.authorization_id IS NOT OLD.authorization_id
  OR NEW.pair_id IS NOT OLD.pair_id
  OR NEW.session_id IS NOT OLD.session_id
  OR NEW.expected_generation IS NOT OLD.expected_generation
  OR NEW.requested_at IS NOT OLD.requested_at
  OR NEW.version <> OLD.version + 1
  OR (OLD.stage = 'RESTORE_CONFIRMED' AND
      (NEW.stage <> 'RESTORE_CONFIRMED' OR NEW.http_200_at IS NOT OLD.http_200_at OR
       NEW.observed_generation IS NOT OLD.observed_generation OR
       NEW.restore_mode IS NOT OLD.restore_mode))
  OR (OLD.stage = 'RESTORE_REQUESTED' AND NEW.stage = 'RESTORE_CONFIRMED' AND
      OLD.resolution_state <> 'IN_FLIGHT')
  OR NOT (NEW.resolution_state = OLD.resolution_state
      OR (OLD.resolution_state = 'IN_FLIGHT' AND
          NEW.resolution_state IN ('RESTORE_OUTCOME_UNKNOWN','RESTORE_RECOVERY_CLAIMED'))
      OR (OLD.resolution_state = 'RESTORE_OUTCOME_UNKNOWN' AND
          NEW.resolution_state = 'RESTORE_RECOVERY_CLAIMED')
      OR (OLD.resolution_state = 'RESTORE_RECOVERY_CLAIMED' AND
          NEW.resolution_state IN ('RESTORE_CLEANUP_CLAIMED','RESTORE_RESOLVED'))
      OR (OLD.resolution_state = 'RESTORE_CLEANUP_CLAIMED' AND
          NEW.resolution_state = 'RESTORE_RESOLVED'))
  OR OLD.resolution_state = 'RESTORE_RESOLVED'
BEGIN SELECT RAISE(ABORT,'restore operation identity or lifecycle violation'); END;
```

Composite FK/UNIQUE buộc `authorization_id` có **cùng** `operation_id/pair_id/session_id/expected_generation`; mọi write transaction phải bật SQLite foreign keys. `authorization_event_id` phải dẫn tới event `PAIR_RESTORE_RISK_ACCEPTED` có exact tuple/scope và authenticated principal; audit row/actor string đơn thuần không chứng minh authority. Trigger chặn delete, thay identity/risk scope, tái consume authorization, hạ stage hoặc sửa HTTP 200 provenance; Store CAS chốt thêm exact old state/version và audit trong transaction. `authorization` được consume và restore row/audit intent được insert trong **cùng** transaction; `UNIQUE(operation_id)` và `UNIQUE(authorization_id)` ngăn dùng lại dù generation chưa đổi. Authorization không được tự chuyển sang operation mới.

New restore audit types: `PAIR_RESTORE_RISK_ACCEPTED`, `PAIR_RESTORE_REQUESTED`, `PAIR_RESTORE_CONFIRMED`, `PAIR_RESTORE_OUTCOME_UNKNOWN`, `PAIR_RESTORE_RECOVERY_CLAIMED`, `PAIR_RESTORE_RECOVERY_TRANSFERRED`, `PAIR_RESTORE_CLEANUP_CLAIMED`, `PAIR_RESTORE_RESOLVED`. New dispatch token `DELIVERY_OUTCOME_UNKNOWN` dành **riêng** cho `dispatch_operations.resolution_state`; pre-send dispositions đề xuất `PRE_SEND_IDENTITY_MISMATCH`, `PRE_SEND_PROTOCOL_UNVERIFIED`, `PRE_SEND_STATUS_UNAVAILABLE`, `PRE_SEND_PROCESS_EXITED`; audit `PRE_SEND_STATUS_RECOVERED` cho hold được giải. `RECOVERY_PENDING`, `SESSION_ABSENT`, `WORKER_TERMINATION_UNKNOWN`, `STALE_EXECUTION_GENERATION`, `TASK_STATE_TRANSITION`, `PRE_SEND_ADMISSIBILITY_REJECTED` giữ nghĩa ADR cũ. `RESTORE_HTTP_200_CONFIRMED` là basis hoàn tất operation sạch sau positive response và quiescence, **không** là physical termination evidence; `PHYSICAL_EXECUTION_RESOLUTION` chỉ với D11/Class A positive proof; `ADMINISTRATIVE_RISK_RESOLUTION` chỉ với Class B/C operator acceptance. Không ghi physical khi chỉ có 404, generation mới hoặc response `/restore`.

**Trusted caller boundary**: một Supervisor service nội bộ nhận principal/role đã được host xác thực và quyết định scope; chỉ service đó được insert authorization/risk audit hoặc gọi recovery/clearance. API worker, TaskContract parameters, AO response, `actor` string, event FK hay raw Store method không cấp quyền. Test thư viện phải chứng minh interface từ chối caller không có verified principal, sai scope và authorization replay; principal giả trong test **không chứng minh** host đã xác thực operator. Việc enable `/restore` trên host thực tế phụ thuộc host cung cấp verified principal cho interface này; nếu chưa có, issuance fail closed và operator restore vẫn disabled. Không yêu cầu xây daemon/IAM/UI mới trong 3B. Trước release contract 3B, Supervisor phải chỉ ra host boundary hiện hữu và owner tích hợp principal, hoặc phê duyệt dependency cụ thể để 3B chỉ triển khai fail-closed interface, giữ restore disabled tới khi dependency hoàn tất; 3C không được phát linked stop và 3D chỉ phân loại unresolved state trong thời gian đó. Không coi test thư viện là điều kiện đủ để enable restore.

## 3. Pair guard, unresolved predicate và API ownership

`RESTORE_UNRESOLVED(pair) := EXISTS pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED'`. Stage `RESTORE_CONFIRMED` vẫn khóa đến resolution. Restore admission từ chối khi Pair còn **bất kỳ attempt mở nào** (`ended_at IS NULL`), độc lập dispatch stage, task state và `current_attempt`; gồm `SEND_CONFIRMED`, `DISPATCHED`, `RUNNING`, `BLOCKED`. `DISPATCH_UNRESOLVED(pair)` là hợp của open-attempt, mọi `SEND_REQUESTED`/`resolution_state IS NULL` kể cả attempt đã đóng, và `CURRENT_ATTEMPT_INCONSISTENT` bên dưới. Không suy ra an toàn từ current pointer sai hoặc attempt thiếu. Historical `DISPATCH_BOUND`/`SEND_CONFIRMED` của attempt đã đóng không khóa vĩnh viễn nếu không còn quarantine hay operation khác unresolved. `SEND_REQUESTED` đã có terminal `DELIVERY_OUTCOME_UNKNOWN` khóa qua D6 quarantine, không qua predicate stage. `STOP_UNRESOLVED(pair)` bao gồm stop in-flight theo D11/D13 (stage/`resolution_state` chưa terminal), không chỉ `STOP_REQUESTED`.

```sql
-- Cùng transaction reservation; kết quả 1 là reject và audit inconsistency.
SELECT
  EXISTS (SELECT 1 FROM task_attempts a JOIN tasks t ON t.task_id = a.task_id
          WHERE t.pair_id = :pair_id AND a.ended_at IS NULL)
  OR EXISTS (SELECT 1 FROM dispatch_operations d
             WHERE d.pair_id = :pair_id AND d.stage = 'SEND_REQUESTED'
               AND d.resolution_state IS NULL)
  OR EXISTS (SELECT 1 FROM tasks t WHERE t.pair_id = :pair_id
             AND ((t.current_attempt > 0 AND NOT EXISTS
                   (SELECT 1 FROM task_attempts a WHERE a.task_id = t.task_id
                      AND a.attempt_number = t.current_attempt))
               OR (t.state IN ('DISPATCHED','RUNNING','BLOCKED') AND
                   (SELECT COUNT(*) FROM task_attempts a
                    WHERE a.task_id = t.task_id AND a.ended_at IS NULL
                      AND a.attempt_number = t.current_attempt) <> 1)));
```

`STOP_UNRESOLVED` phải xét cả `STOP_REQUESTED` **và** `STOP_CALL_SUCCEEDED` với `resolution_state=IN_FLIGHT`, cùng `STOP_CALL_OUTCOME_UNKNOWN`/`STOP_REISSUE_REQUIRES_HUMAN` chưa được operator xử lý; stop đã terminal nhưng còn quarantine tiếp tục bị D6 guard chặn. `RESTORE_UNRESOLVED` được kiểm tra trong mọi mutation path bên dưới, kể cả API Store hiện có; partial index tự nó chỉ ngăn restore trùng, không ngăn dispatch/stop/spawn.

| Mutation path | Guard cùng SQLite transaction | Owner dự kiến |
|---|---|---|
| `ReservePairRestore` | Exact Pair/session/generation `TERMINATED`; reject mọi open attempt, `SEND_REQUESTED`/NULL kể cả attempt closed, current-attempt inconsistency, provisioning/stop/restore unresolved và quarantine. Historical closed `DISPATCH_BOUND`/`SEND_CONFIRMED` sạch được đi qua. Consume exact one-shot authorization, intent/audit atomic. | 3B |
| `PrepareBoundDispatch` **và** legacy `PrepareDispatch`; `CreateDispatchOperation`, `UpdateDispatchOperationStage`, `RecordSendRequested` | Reject khi `RESTORE_UNRESOLVED` hoặc quarantine/provisioning guard; legacy path không được lách binding. Generic Store methods phải delegate tới guarded transaction hoặc bị cấm cho P03. | 3B + Store common |
| `ReservePairProvisioning`, `CreateWorkerSession`, `CreatePairProvisioningOperation`, `UpdatePairProvisioningOperationStage` | Reject khi `RESTORE_UNRESOLVED`; confirmation chỉ cho operation đã reserve trước đó nếu restore chưa chiếm Pair, giữ non-replacement D3. | 3B + Store common |
| `UpdateWorkerSessionStatusAndQuarantine`, restore confirmation, D6 clearance | CAS exact Pair/session/generation; không đổi status/generation hay clear quarantine khi restore unresolved ngoài guarded restore/recovery API. Read-only getters không cần guard. | 3B, 3C clearance |
| `CreateStopOperation`, `UpdateStopOperationStage`, stop coordinator | Reject khi restore unresolved **trừ** `ClaimRestoreCleanup` liên kết same restore operation/authorized principal trong transaction §5; không dùng generic bypass. Không reissue `/kill` tự động. | 3C |
| `AtomicTerminalTransition`, `AtomicAttemptClosureTransition`, startup scanner | Chỉ terminal/closure đúng attempt lineage; không đóng attempt khác để mở restore. Scanner quét restore trước dispatch/provisioning/stop và không phát `/restore`/`/kill` tự động. | Store common, 3D |

Mọi check ở bảng là transaction-local, cùng write serialization/CAS; check ở coordinator trước đó chỉ là early rejection. Unresolved restore độc lập với session quarantine: cả hai gate phải thỏa trước dispatch. Không có attempt liên quan thì chỉ WorkerSession/Pair bị giữ/quarantine; **không tạo TaskAttempt giả** để mô phỏng double gate. Nếu có attempt liên quan đã đóng, giữ quarantine của chính attempt đó theo D6.

## 4. Restore transitions và transaction boundaries

| Expected stage/resolution + điều kiện | CAS/effect/next | Invariant |
|---|---|---|
| Trusted authorization chưa consume, exact tuple/scope; no competing Pair operation | Tx A: `consumed_at IS NULL` → consume auth; insert `RESTORE_REQUESTED/IN_FLIGHT`, append `PAIR_RESTORE_REQUESTED`. Commit trước wire. | Một authorization → một operation → tối đa một `/restore`; rollback ⇒ zero call. |
| `RESTORE_REQUESTED/IN_FLIGHT` sau Tx A | AO call ngoài transaction, chỉ caller giữ operation mới được gọi đúng một lần. | Context cancellation hoặc crash không cho phép reissue. |
| HTTP 200 + valid `ok`, session ID, nonempty generation, `restoreMode`; exact operation/version vẫn `REQUESTED/IN_FLIGHT` | Tx B: CAS stage → `RESTORE_CONFIRMED`, set immutable `http_200_at`, observed generation/mode, update cùng WorkerSession bằng expected old generation, append `PAIR_RESTORE_CONFIRMED`. `resolution_state` giữ `IN_FLIGHT`. | HTTP 200 provenance được giữ kể cả sau resolution; task/attempt/dispatch bound snapshot không đổi. Nếu CAS/audit/commit fail, reread; không suy từ GET rằng call này thành công. |
| Lost/invalid response, HTTP 404/409, transport/context error, hoặc Tx B rollback thực sự | Tx C: CAS `REQUESTED/IN_FLIGHT → REQUESTED/RESTORE_OUTCOME_UNKNOWN`, quarantine WorkerSession và attempt liên quan đã có, append unknown audit; nếu Tx C rollback, `REQUESTED/IN_FLIGHT` vẫn khóa. | Không gọi lại restore hoặc release lane. 404/409 không chứng minh không effect. |
| Sau crash, `REQUESTED/IN_FLIGHT` | 3D Tx C như trên; không dựa vào GET generation để nâng stage `CONFIRMED`. | Bằng chứng local còn intent, không có positive response đã commit. |
| Sau crash, `CONFIRMED/IN_FLIGHT` | 3D giữ stage; quan sát status để báo operator. Trusted resolution CAS theo §5 sau quiescence/risk decision. | Positive HTTP 200 đã commit, nhưng lane vẫn khóa. |
| `REQUESTED/RESTORE_OUTCOME_UNKNOWN` | Trusted `ClaimRestoreRecovery`: CAS `version` + exact tuple → `RESTORE_RECOVERY_CLAIMED`, set principal/time, audit. | Chuyển quyền xử lý một lần; không phải resolution hay clearance. |
| `CONFIRMED/IN_FLIGHT` | Trusted `ClaimRestoreRecovery`: CAS `version` + exact tuple → `RESTORE_RECOVERY_CLAIMED`, set principal/time, audit; stage vẫn `CONFIRMED`. | Positive HTTP 200 giữ nguyên; claim không phải clearance. |
| `RESTORE_RECOVERY_CLAIMED` hoặc `RESTORE_CLEANUP_CLAIMED` | Resolution Tx D theo §5, terminal `RESTORE_RESOLVED` + basis/event; giữ `stage` nguyên. | Operation resolved ≠ quarantine cleared; dispatch còn bị D6 gate nếu quarantine. |

Mọi CAS thua phải reread durable operation, audit, authorization và WorkerSession trong transaction/consistent snapshot: nếu đúng result đã commit thì replay read-only; nếu khác tuple/version hoặc không đọc được thì stale caller fail closed. Không khẳng định DB vẫn `RESTORE_REQUESTED` khi caller khác đã thắng. `http_200_at`, observed generation/mode và authorization consumption là immutable sau commit; không biến `RESTORE_OUTCOME_UNKNOWN` thành `RESTORE_CONFIRMED` chỉ từ status GET.

## 5. Operator recovery, authorized stop và clearance

`RESTORE_RECOVERY_CLAIMED` giữ unresolved unique index và mọi Pair guard. Chỉ trusted recovery principal giữ claim mới được lựa chọn một trong các transaction sau. Claim không phải generic operator override và không cho gọi `/restore` lại. Claim stale phải bị CAS `version` từ chối. Nếu đổi người xử lý, `TransferRestoreRecovery` transaction CAS exact operation/stage/resolution/old principal/version, set new verified principal/time, increment version và append `PAIR_RESTORE_RECOVERY_TRANSFERRED`; audit fail rollback cả chuyển quyền. Không sửa actor string âm thầm.

1. **Positive restore, no unsafe execution**: với `RESTORE_CONFIRMED`, HTTP 200 evidence đã commit, GET hiện tại cùng session/observed generation và `idle`/`waiting_input`, operator xác nhận không có execution/stop/quarantine unresolved. Tx D CAS exact stage/version/claim → `RESTORE_RESOLVED`, `resolution_basis=RESTORE_HTTP_200_CONFIRMED`, resolution audit. Không clear quarantine trong transaction này; nếu bất kỳ quarantine cũ tồn tại, dùng D6 clearance riêng trước dispatch. GET chỉ chứng minh trạng thái hiện tại, HTTP 200 row chứng minh call response, hai bằng chứng không bị nhập làm một.
2. **Unknown outcome nhưng có physical evidence**: operator yêu cầu cleanup của *session hiện quan sát*, không suy từ generation khác. `ClaimRestoreCleanup` transaction CAS `RESTORE_RECOVERY_CLAIMED → RESTORE_CLEANUP_CLAIMED`, insert **một** `stop_operations` với `restore_operation_id`, exact Pair/session/target generation, `PAIR_RESTORE_CLEANUP_CLAIMED` và `STOP_OPERATION_REQUESTED` audit atomic. Nếu target generation **bằng** immutable `terminal_generation` và session của chính attempt đã đóng `FAILED/HUMAN_REQUIRED`, dùng `QUARANTINE_CLEANUP` theo D11 và giữ kiểm tra `CreateStopOperation`/3A-R1-002 nguyên vẹn. Nếu runtime sau restore có **generation mới**, tuyệt đối không gắn stop với attempt cũ: đề xuất mở rộng D11 cho `PAIR_MAINTENANCE` không `task_id/contract_id/attempt_id`, nhưng có `restore_operation_id`, verified operator authority và audit provenance ghi old/new generation. Mở rộng này **chờ phê duyệt**; chưa phê duyệt thì không gọi `/kill` theo nhánh này, dùng Class C có audit. Không có attempt thì `PAIR_MAINTENANCE` không gắn attempt như D11 hiện hành. Verified operator phải chấp nhận rủi ro `/kill` chỉ session-scoped, không có atomic generation fence; sau commit 3C mới gọi pinned `/kill`. Nếu generation đổi giữa claim và call, không call, chuyển Class C. Stop HTTP 200 + within-deadline `isTerminated=true` đúng target generation chỉ là Class A **cho generation ấy**. Termination generation mới không chứng minh execution generation cũ đã kết thúc, không tự clear quarantine attempt cũ. Stop `404`, timeout, transport unknown hoặc mismatch không được ghi physical/`WORKER_STOPPED`; operator dùng Class B/C nếu chấp nhận rủi ro. Không tự reissue `/kill` hoặc `/restore`; stop in-flight giữ restore lock.
3. **Administrative risk resolution**: với 404/Class B hoặc outcome không thể chứng minh/Class C, trusted principal ghi rõ absence evidence/risk acceptance (`ADMINISTRATIVE_RISK_ACCEPTED` khi Class C), exact Pair/session/generation/operation, lý do và scope. Tx D CAS claimed operation/version → `RESTORE_RESOLVED`, basis `ADMINISTRATIVE_RISK_RESOLUTION`, audit; không ghi `TERMINATION_CONFIRMED`/`WORKER_STOPPED`. Nếu stop đã claim mà chưa terminal, trước Tx D phải ghi stop resolution theo D11; không bỏ mặc stop in-flight. Authorization này **khác** one-shot authorization ban đầu và không thể tự phát từ worker.
4. **Tx D và D6 clearance theo từng lineage**: Tx D chỉ CAS exact restore operation/stage/resolution/claim/version sau khi mọi linked stop đã terminal; `resolution_basis=PHYSICAL_EXECUTION_RESOLUTION` **chỉ nếu** bằng chứng Class A giải quyết cả runtime hiện tại lẫn từng execution lineage cũ đang quarantine. Nếu generation mới được stop vật lý nhưng attempt cũ thiếu bằng chứng, operator phải ghi Class B/C acceptance riêng cho attempt cũ; Tx D dùng basis tổng thể bảo thủ `ADMINISTRATIVE_RISK_RESOLUTION`, audit details nêu từng attempt ID/session/generation, evidence/basis và WorkerSession generation. HTTP 200 restore chỉ là provenance, không là Class A cho attempt cũ. Sau Tx D, transaction D6 riêng CAS exact WorkerSession **và từng attempt liên quan** còn quarantined, kiểm tra lineage/evidence/authority tương ứng rồi append audit physical hoặc administrative cho từng đối tượng; bất kỳ CAS/audit fail đều rollback toàn bộ clearance. WorkerSession chỉ clear khi risk của runtime hiện tại **và tất cả** attempt liên quan đã được xử lý; new-generation termination riêng không đủ. Không có attempt thì chỉ WorkerSession được clear, không tạo fake attempt. Nếu clearance rollback, operation vẫn `RESOLVED` nhưng quarantine vẫn khóa dispatch/provisioning/restore mới. Nếu clearance thắng trước Tx D thì phải bị từ chối. `DELIVERY_OUTCOME_UNKNOWN` trên dispatch operation giữ nguyên xuyên suốt.

**Attempt đang mở**: `QUARANTINE_CLEANUP` không hợp lệ theo D11; restore admission đã chặn attempt mở. Pre-send hold/quarantine phải tiếp tục cùng attempt theo §6 hoặc đóng đúng attempt bằng D12 + disposition/audit trước khi `ClaimRestoreCleanup`; không đóng attempt của task khác. `PAIR_MAINTENANCE` không gắn attempt được đề xuất cho runtime generation mới dù còn *historical closed attempt*; đây là mở rộng D11 cần duyệt, không lách lineage của attempt cũ. Trong toàn bộ `RECOVERY_CLAIMED`/`CLEANUP_CLAIMED`, dispatch, provisioning và restore mới tiếp tục bị chặn; chỉ linked stop API và resolution/clearance đúng thứ tự được đi qua. Stop coordinator 3C xử lý `/kill`, scanner 3D chỉ phân loại/đọc state; không dồn hai phần này vào 3B.

## 6. Pre-send hold, quarantine và kết thúc attempt

Thứ tự observation: (1) CAS ownership/stage; (2) response protocol + `sessionId`; (3) opaque generation equality; (4) `isTerminated`; (5) activity. ADR D7/§22 giữ `active/blocked/exited` matched ở `DISPATCH_BOUND`, `DISPATCHED`, `ended_at=NULL`, audit `PRE_SEND_ADMISSIBILITY_REJECTED`; `idle/waiting_input` matched mới có thể CAS `SEND_REQUESTED` trên **cùng attempt**. `isTerminated=true`/same generation dùng `WORKER_TERMINATION_UNKNOWN`; mismatch dùng `STALE_EXECUTION_GENERATION`. Mọi `DISPATCHED -> FAILED` phải atomic D12 (`ended_at`, disposition, `TASK_STATE_TRANSITION` và error details); không chờ phê duyệt lại các điều này.

| Observation/hold | State và authority đề xuất | Cùng attempt hay đóng/cleanup |
|---|---|---|
| Query transport unavailable/timeout, chưa có contradictory identity | `DISPATCH_BOUND`, `DISPATCHED`, open attempt; set existing `RECOVERY_PENDING` + rejection audit trong tx; **không quarantine** chỉ vì thiếu GET. Pair lane đã bận bởi `DISPATCHED`; `RecordSendRequested` đòi fresh positive GET. | 3D retry query theo caller-injected policy, không `/send`; valid exact identity/generation + idle/waiting → transaction CAS `RECOVERY_PENDING → NULL` + `PRE_SEND_STATUS_RECOVERED`, rồi same-attempt send. Nếu operator chọn bỏ vì outage kéo dài: D12 `FAILED`/new `PRE_SEND_STATUS_UNAVAILABLE`, impose double quarantine/audit atomic; sau đó closed-attempt cleanup/administrative resolution. Không có timeout mặc định. |
| Malformed status/protocol, identity chưa xác định | Giữ bound/open, disposition new `PRE_SEND_PROTOCOL_UNVERIFIED` và audit; không quarantine chỉ từ lỗi decode. | Trusted operator xác nhận AO pinned compatibility, sau đó fresh valid GET exact tuple/idle-waiting và transaction clear hold/audit mới được same-attempt send. Nếu không xác nhận được, operator D12 `FAILED`/`PRE_SEND_PROTOCOL_UNVERIFIED`, double quarantine; closed-attempt cleanup. Một GET thành công đơn lẻ không tự giải hold. |
| Response `sessionId` khác bound ID | Contradictory identity: D12 `FAILED`/new `PRE_SEND_IDENTITY_MISMATCH`, double quarantine + audit atomic; không tin generation/activity trong response. | Không tiếp tục attempt cũ. Chỉ closed-attempt `QUARANTINE_CLEANUP` với exact lineage hoặc Class B/C; clearance D6 không hồi sinh snapshot đã đóng. |
| HTTP 404 | Đề xuất mới D12 `FAILED`/existing `SESSION_ABSENT`, double quarantine; 404 không chứng minh physical stop. | Class B/C operator resolution; không tự replace hoặc same-attempt send. |
| Generation mismatch | D7 `FAILED`/`STALE_EXECUTION_GENERATION`, D12 closure; đề xuất double quarantine cùng transaction. | Không sửa snapshot; Class A/B/C theo D6. |
| `exited` matched, not terminated | ADR §22 giữ open và rejection audit; không gọi `/restore` trên bound attempt. | Operator quyết định bỏ attempt: D12 `FAILED`/new `PRE_SEND_PROCESS_EXITED` + double quarantine; chỉ sau đó `QUARANTINE_CLEANUP` 3C hoặc Class B/C. Nếu không bỏ, giữ open và chặn send. |
| `isTerminated=true` matched | ADR §22/D12 `FAILED`/`WORKER_TERMINATION_UNKNOWN`; §22 không áp thêm quarantine chỉ từ observation. | Attempt đóng; operator-authorized restore cùng session có thể bắt đầu nếu các guard khác sạch. |

Hold disposition và recovery audit là candidate mới, không xóa bound snapshot. Query outage chỉ làm chậm lane, không tự chi phí Class A/B/C cho mọi mất mạng. Identity mismatch hoặc operator-abandoned hold mới áp double quarantine; successful GET **không** clear quarantine. `PRE_SEND_ADMISSIBILITY_REJECTED` cho protocol/query cần addendum mở rộng details; raw payload phải được redacted/bounded trong audit. `RecordSendRequested` kiểm tra lại hold/disposition, restore unresolved, quarantine và current lineage trong cùng transaction, dựa trên positive AO observation vừa thu trước transaction; đây là precheck, **không phải** atomic AO generation fence. Nếu local state đổi sau GET, CAS fail; nếu AO đổi ngoài Supervisor sau GET, các bước observation/recovery tiếp theo xử lý mà không sửa bound snapshot.

**Thứ tự hold và CAS**: `PRE_SEND_PROTOCOL_UNVERIFIED` chặt hơn `RECOVERY_PENDING`; timeout/query outage chỉ có thể tạo hoặc giữ `RECOVERY_PENDING`, không hạ protocol hold. GET hợp lệ là observation, không phải authority để clear protocol hold. Mọi mutation dưới đây CAS exact `attempt_id/task_id/contract_id/Pair/session_id/terminal_generation`, `DISPATCH_BOUND`, open/current attempt và **old recovery_disposition** trong một transaction với audit; CAS thua phải reread, không ghi đè winner. Clear quarantine theo D6 là transaction khác và không biến hold thành NULL.

| Current disposition | Observation + authority | Proposed transition và expected result |
|---|---|---|
| `NULL` hoặc `RECOVERY_PENDING` | Query timeout/transport unavailable, không có contradictory identity | `NULL → RECOVERY_PENDING` hoặc giữ nguyên `RECOVERY_PENDING`, rejection audit; attempt mở, không send/quarantine chỉ vì outage. |
| `PRE_SEND_PROTOCOL_UNVERIFIED` | Query timeout/transport unavailable, kể cả scanner 3D | Giữ nguyên `PRE_SEND_PROTOCOL_UNVERIFIED`, audit thêm observation; **không** ghi `RECOVERY_PENDING` và không send. |
| `NULL` hoặc `RECOVERY_PENDING` | Malformed/protocol failure | CAS lên `PRE_SEND_PROTOCOL_UNVERIFIED`, audit; không send. |
| `RECOVERY_PENDING` | Fresh valid GET exact tuple + `idle/waiting_input` | CAS exact `RECOVERY_PENDING → NULL` + `PRE_SEND_STATUS_RECOVERED`; sau đó same-attempt admission mới. Concurrent protocol failure thắng trước thì CAS clear thua. |
| `PRE_SEND_PROTOCOL_UNVERIFIED` | Fresh valid GET, thiếu verified operator compatibility decision | Giữ hold, không send dù GET lặp lại; timeout kế tiếp vẫn giữ hold. |
| `PRE_SEND_PROTOCOL_UNVERIFIED` | Verified operator xác nhận AO pinned compatibility **và** fresh valid GET exact tuple + `idle/waiting_input` | CAS exact hold/lineage/verified authority → `NULL` + `PRE_SEND_STATUS_RECOVERED` với decision provenance; nếu CAS/audit fail, hold còn nguyên hoặc reread winner. Không clear quarantine. |
| Bất kỳ hold nào | Termination, mismatch hoặc 404 theo thứ tự observation | D7/§22/D12 terminal branch tương ứng thắng; không clear hold để send. Nếu task chuyển FAILED thì closure/audit/quarantine theo rule riêng. |
| `PRE_SEND_PROTOCOL_UNVERIFIED` hoặc `RECOVERY_PENDING` | Operator quyết định bỏ attempt | D12 CAS exact hold/lineage → `FAILED`, đóng attempt, disposition tương ứng và double quarantine/audit atomic; không `QUARANTINE_CLEANUP` khi attempt còn mở. |

`RecordSendRequested` phải CAS `recovery_disposition IS NULL` cùng guard lineage/quarantine/restore và positive pre-send observation trong transaction. Mọi hold chưa giải đều bị từ chối, kể cả khi một caller khác vừa thu GET hợp lệ; stale caller không tự clear hold rồi send. Nếu disposition đã đổi, reread exact state và thực hiện nhánh recovery tương ứng, không thử lại blind send.

## 7. Dispatch resolution và stale writer

Với unconfirmed `SEND_REQUESTED`, D5 transaction CAS `resolution_state IS NULL` → `DELIVERY_OUTCOME_UNKNOWN` cùng `DISPATCHED -> FAILED -> HUMAN_REQUIRED`, `ended_at`, `UNCERTAIN_DELIVERY_CRASH`, double quarantine và audit. `stage` giữ `SEND_REQUESTED`; resolution literal terminal/immutable. D6 Class A/B/C chỉ clear quarantine riêng, không đổi dispatch resolution hoặc tuyên bố delivery thành công/thất bại. Replay đúng tuple sau commit là read-only idempotent; late HTTP 200 không được chuyển `SEND_CONFIRMED`. `RecordSendConfirmed` chỉ CAS khi `stage=SEND_REQUESTED`, `resolution_state IS NULL`, exact current attempt/task và HTTP 200 validated; loser reread durable state.

| Reread sau confirmation/containment CAS fail | Test oracle |
|---|---|
| `SEND_CONFIRMED`, đúng tuple + audit | Stale caller dừng; không D5, không resend. |
| `SEND_REQUESTED` + `DELIVERY_OUTCOME_UNKNOWN`, D5 audit/quarantine | Idempotent result; late confirmation từ chối; clearance không đổi literal. |
| `SEND_REQUESTED` + NULL, attempt vẫn current/open | Transaction rollback thật; D5 containment CAS mới, không resend. |
| Mismatch lineage, không đọc được DB/audit hoặc state khác | Fail closed và chuyển 3D/operator; không khẳng định DB vẫn `SEND_REQUESTED`. |

## 8. Crash matrix và acceptance scenarios

| Cửa sổ crash | Durable evidence | Recovery owner và hành động |
|---|---|---|
| Trước Tx A commit | Chưa consume authorization/intent | Không AO call; authorization vẫn one-shot chưa dùng. |
| Sau Tx A, trước hoặc trong `/restore` | `REQUESTED/IN_FLIGHT`, auth consumed | 3D → `RESTORE_OUTCOME_UNKNOWN`; operator claim, không reissue. |
| AO effect/HTTP 200 trước Tx B commit | Vẫn `REQUESTED`, có thể AO generation mới | 3D unknown; GET không causal proof; operator Class A/B/C. |
| Tx B đã commit, trước quiescent/resolution | `CONFIRMED/IN_FLIGHT`, HTTP 200 evidence | 3D giữ provenance, giao operator; lane khóa. |
| Tx C/claim/cleanup stop commit fail | State cũ hoặc concurrent winner | Reread version/audit; không thêm effect khi chưa xác định owner. |
| Linked stop intent commit, crash/lost `/kill` response | Stop D11 + restore `CLEANUP_CLAIMED` | 3D/3C theo D11, không reissue; Class B/C nếu effect unproven. |
| Tx D resolved, clearance chưa commit | Restore resolved, WorkerSession/attempt vẫn quarantined | 3C/operator retry **clearance CAS local** với cùng evidence; zero AO effect. |

Acceptance scenarios **đề xuất**, chưa phải kết quả test implementation:

| Scenario/probe | Expected result |
|---|---|
| Pair còn open attempt ở `DISPATCH_BOUND`, `SEND_CONFIRMED`, `DISPATCHED`, `RUNNING` hoặc `BLOCKED`; `current_attempt` lệch/thiếu; closed attempt nhưng `SEND_REQUESTED`/NULL | `ReservePairRestore` reject, zero authorization consumption/restore intent/AO call; inconsistent lineage được audit. Hai caller restore và cross-type mutation tranh Pair chỉ có một winner hợp lệ. |
| Chỉ còn historical closed `DISPATCH_BOUND`/`SEND_CONFIRMED`, không quarantine hay operation unresolved | Admission không bị lịch sử khóa vĩnh viễn; các guard Pair/session/generation/authorization khác vẫn áp dụng. |
| One-shot authorization exact tuple/scope; replay và untrusted caller | Một operation duy nhất; replay, principal sai hoặc thiếu verified host principal bị từ chối. Library test principal không chứng minh host authentication. |
| Crash `REQUESTED`, `CONFIRMED`, `OUTCOME_UNKNOWN`; lost/invalid response, HTTP 404/409, HTTP 200 rồi Tx B rollback | Durable state và crash matrix ở trên; zero blind `/restore` retry, GET generation mới không tự xác nhận call. |
| Linked stop target generation = closed attempt snapshot; target = new runtime generation; không có attempt | Old-match dùng `QUARANTINE_CLEANUP` với exact 3A lineage; new-gen không gắn old attempt, chỉ candidate `PAIR_MAINTENANCE` có restore link/operator provenance sau phê duyệt; no-attempt không tạo fake attempt. Stale/mismatched insert rollback. |
| New-gen stop Class A nhưng old attempt chưa được chứng minh; 404/Class B; unknown/Class C; clearance CAS/audit fail | New-gen stop không clear old quarantine; Tx D basis administrative nếu old risk được operator chấp nhận; WorkerSession và từng attempt chỉ clear sau đủ evidence/authority trong D6 transaction, rollback giữ toàn bộ quarantine. Không reissue kill khi outcome unknown. |
| Protocol failure → timeout → valid GET, không có operator; tiếp theo verified operator + fresh valid GET | Hold luôn `PRE_SEND_PROTOCOL_UNVERIFIED` qua timeout/GET, `RecordSendRequested` reject; chỉ exact authorized CAS + audit mới clear hold và cho same-attempt admission. |
| Concurrent writer protocol failure và recovery clear; stale send; operator abandonment | CAS loser reread, không hạ hold/không send; abandonment D12 đóng đúng attempt và quarantine atomic. |
| D5 writer race, late confirmation, replay, clearance | `DELIVERY_OUTCOME_UNKNOWN` terminal trên dispatch operation, late 200 bị từ chối, clearance không đổi resolution. |
| Migration fresh/v3→v4, rerun, FK/unique/CHECK, linked-stop nullability, rollback audit/CAS | DDL candidate và transaction invariant được kiểm chứng sau phê duyệt, không tuyên bố PASS trong draft. |

Test dùng fake AO; **không gọi restore trên session thật** trong lượt tài liệu này. DDL candidate §2 không đổi trong revision này: nullable `stop_operations.restore_operation_id` cho cả linked cleanup có/không có `attempt_id`; exact lineage và basis từng đối tượng do Store CAS + audit details kiểm chứng, không nới 3A-R1-002. Đây là invariant phải được probe ở implementation sau phê duyệt, không phải bằng chứng migration đã chạy.

## 9. Scope sau phê duyệt và quyết định còn mở

Supervisor cần audit/chốt DDL, trusted operator boundary, risk scope, recovery claim, linked-stop CAS, pre-send hold/terminal dispositions và audit tokens; `AUTOMATIC_RESTORE=DISABLED` trong release 3B dự kiến. Thử nghiệm tìm safe automatic subset chỉ phục vụ quyết định **sau này**, không là điều kiện để làm operator-gated restore. Không tự đóng blocker 3B hoặc phê duyệt proposal toàn phần.

Sau khi addendum được phê duyệt: reconcile `docs/04_ARCHITECTURE.md`, `05_DOMAIN_MODEL.md`, `06_WORKFLOW_STATE_MACHINE.md`, `12_UPSTREAM_INTEGRATION.md`, `14_FAILURE_RECOVERY.md`, `21_TRACEABILITY_MATRIX.md`, `22_MODULE_PROVENANCE.md`, `docs/phases/P03_AO_INTEGRATION.md`; cập nhật scope/JSON/AC của `docs/tasks/DRAFT_TASK_CONTRACT_P03_003B.md` và release bằng quyết định riêng. Contract 3B cần whitelist mới cho `internal/store/migrations.go`, migration tests, `internal/domain/session_lifecycle.go`, `internal/store/session_lifecycle.go`, `internal/store/dispatch.go`, `internal/store/restore_transactions.go` và tests; guard stop/clearance thuộc 3C, startup scanner thuộc 3D. Các generic Store mutation API hiện có cũng phải được guard hoặc đóng đối với P03; chỉ liệt kê file mới mà không xử lý đường cũ là không đạt acceptance. Không sửa ADR accepted/canonical/code trong lượt draft.
