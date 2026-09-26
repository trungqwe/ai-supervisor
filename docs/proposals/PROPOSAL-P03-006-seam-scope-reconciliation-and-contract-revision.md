# PROPOSAL-P03-006: Seam Scope Reconciliation and Task Contract P03-004 Revision 2

> **Status**: PROPOSED (Chờ External Supervisor thẩm định và phê duyệt)
> **Authority**: docs/24_CHANGE_GOVERNANCE.md (Level 3 Canonical Architecture / Level 6 Task Contract Governance)
> **Active Gate**: TASK_P03_004_REMEDIATION
> **Target Task Contract**: CONTRACT-TASK-P03-004-02 (Revision 2 của TASK-P03-004)
> **Affected Specifications**: docs/tasks/TASK_CONTRACT_P03_004.md (Draft Revision 2), docs/14_FAILURE_RECOVERY.md, docs/sources/SOURCE_REGISTRY.md
> **Architectural Alignment**: Hoàn toàn nhất quán với ADR-017 (Accepted), ADR-016 (Accepted), ADR-011 (Accepted). Không phát hiện xung đột kiến trúc với các ADR đã duyệt; không yêu cầu sửa đổi ADR-017.

---

## 1. Căn Cứ Quản Trị & Vấn Đề Cần Giải Quyết

Trong quá trình thực hiện và tái thẩm định TASK-P03-004 (commit `09b47bf` và `ef0a496`), External Supervisor đã ghi nhận 2 điểm nối kiến trúc (architectural seams) vượt ra ngoài phạm vi cho phép (`allowed_scope`) của `CONTRACT-TASK-P03-004-01`:

### 1.1. Seam 1 — Tín hiệu lỗi / hoàn tất bất đồng bộ từ Poller (Finding R1-004)

1. **Đối chiếu chính xác điều/khoản của accepted ADR-017**:
   - **ADR-017 §2.6 ("Startup Scanner and Admission Lifecycle")**: Quy định ma trận kết quả phân loại lúc khởi động (Startup Classification Matrix):
     * **Complete (`report.Complete == true && err == nil`)**: Phân loại hoàn tất. `Runner` tự đánh dấu `r.ready = true`. Host kiểm tra Pair guards, khởi động `Poller.Start(ctx)`, khởi động scheduler gọi `TimeoutMonitor.Tick(ctx)`, và mở listener HTTP readiness probe `/readyz` (trả về HTTP 200 OK).
     * **Incomplete (`report.Complete == false` hoặc `err != nil`)**: Gặp lỗi, context cancel, hoặc attempt thiếu budget. `Runner` giữ `r.ready = false`. Listener tiếp tục ĐÓNG fail-closed, daemon dừng phục vụ với exit code khác 0.
   - **ADR-017 §4.2 ("Bảng Tổng hợp Kịch bản")**: Quy định kịch bản Shutdown Drain:
     * Nhận SIGINT / lệnh stop: listener đóng ngay -> poller drain -> timeout permits drain -> `Store.Close()` -> dọn `.owner.json` (nếu khớp instance) -> đóng lock handle cuối cùng.
   - **Kết luận đối chiếu ADR**: Accepted ADR-017 §2.6 và Contract 004 Rev 1 (AC-004-06) **chưa từng có điều/khoản quy định rằng sau khi startup đã hoàn tất thành công và daemon đang phục vụ bình thường, nếu background poller thất bại giữa chừng trong chu kỳ định kỳ thì HTTP probe `/readyz` phải chuyển động từ HTTP 200 sang HTTP 503 trong khi tiến trình vẫn tiếp tục chạy**. AC-004-06 của Contract 004 chỉ quy định: port đóng trước Run và mở sau Complete; probe trả về 200 sau Complete và 503 khi startup hoặc shutdown drain.

2. **Phân loại chính xác**:
   - Yêu cầu host watcher phát hiện lỗi Poller bất đồng bộ trong lúc đang chạy và chuyển `/readyz` sang HTTP 503 là một **đề xuất hardening vận hành mới cho Revision 2 (operational hardening proposal)** nhằm nâng cao tính tự bảo vệ (self-healing / fail-closed) của daemon khi thành phần ngầm gặp sự cố.
   - Đây **không phải** là một yêu cầu đã được phê duyệt sẵn trong ADR-017 hay Contract 004 Rev 1. Việc diễn giải thành yêu cầu đã có của Contract 01 là không đúng căn cứ văn bản pháp quy.

3. **Tác động lên P03 Exit Gate**:
   - Việc phân loại đúng là đề xuất hardening vận hành khẳng định không có vi phạm hợp đồng gốc của Contract 004 Rev 1.
   - Finding `R1-004` được ghi nhận ở trạng thái `OPEN` để bổ sung vào Task Contract Revision 2 (với mã tiêu chí riêng `AC-004-09`), cho phép mở rộng `allowed_scope` để sửa `internal/recovery/poller.go` (xuất tín hiệu hoàn tất) và `cmd/supervisor/main.go` (bổ sung watcher).

### 1.2. Seam 2 — Typed Library API cho AO Workspace File Retrieval (Finding R1-005)

1. **Yêu cầu của ADR-017 §4.1 & AC-004-07**:
   - Track 2 của P03 Integration Test Harness (ADR-017 §4.1) phải kiểm chứng 5 bước exit gate AO (tạo session, dispatch chỉ thị, quan sát/reconciliation, đọc file workspace report, teardown) bằng cách **gọi trực tiếp các API điều phối nội bộ đã được duyệt của thư viện Go Supervisor** (`internal/ao`, `internal/recovery`, `internal/stop`, `internal/store`).
2. **Hiện trạng mã nguồn (internal/ao/client.go)**:
   - Thư viện `ao.Client` hiện chỉ cung cấp: `CreateSession`, `GetWorkerStatus`, `StopWorker`, `RegisterProject`.
   - Hoàn toàn chưa có typed method để đọc workspace file. Bước 4 trong `test/integration/p03_ao_harness_test.go` hiện phải gọi raw HTTP (`http.NewRequestWithContext`), dù đã bổ sung cơ chế defensive read giới hạn kích thước qua `io.LimitReader`.
3. **Xung đột phạm vi & Tác động**:
   - `internal/ao/**` nằm trong `forbidden_scope` của Contract 004 Rev 1. Mọi chỉnh sửa thêm hàm vào `internal/ao/client.go` khi chưa có hợp đồng duyệt đều vi phạm Task Scope Immutability.
   - Do đó, bước đọc workspace chưa thể đi qua API thư viện đã duyệt; finding `R1-005` tiếp tục giữ trạng thái `OPEN` cho đến khi Task Contract Revision 2 được phê duyệt và phát hành chính thức.

---

## 2. Đối Chiếu Nguồn Chuẩn (Source Registry & Upstream Wire Contract)

Đối chiếu với nguồn upstream đã đăng ký tại `docs/sources/SOURCE_REGISTRY.md` (Mục 01) và `docs/sources/01_AGENT_ORCHESTRATOR.md`:

| Trường Thông Tin | Giá Trị Chuẩn Định Danh | Căn Cứ Chứng Cứ |
|---|---|---|
| **Repository Name** | `Untrivial-ai/agent-orchestrator` | SOURCE_REGISTRY.md, 01_AGENT_ORCHESTRATOR.md |
| **Pinned Documentation Version** | `v0.13.0` | GitHub Tag `v0.13.0` |
| **Pinned Commit SHA** | `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` | SOURCE_REGISTRY.md |
| **Vai trò Kiến trúc** | Execution Control Plane daemon | ADR-002, docs/04_ARCHITECTURE.md |

### 2.1. Wire Route & Schema trên Pinned Source

Theo đặc tả OpenAPI `contracts/cloud/openapi.yaml` và `frontend/src/api/schema.ts` tại pinned commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`:
- **Endpoint**: `GET /api/v1/sessions/{sessionId}/workspace/file?path={filePath}`
- **Response Content-Type**: `application/json` (Không trả trực tiếp raw file bytes!).
- **Response Wire Schema (`WorkspaceFileResponse`)**:
  ```json
  {
    "sessionId": "string",
    "path": "string",
    "content": "string",
    "binary": false,
    "deleted": false,
    "contentTruncated": false,
    "size": 0,
    "status": "unmodified",
    "workspaceVersion": "string",
    "diff": "string",
    "diffTruncated": false,
    "editable": false,
    "fileFingerprint": "string",
    "additions": 0,
    "deletions": 0
  }
  ```

### 2.2. Yêu Cầu Thiết Kế Cho Adapter `internal/ao/client.go`

1. **Phân biệt hai cấp độ giới hạn kích thước**:
   - **Giới hạn Wire Envelope (`MaxWireBytes`)**: Bảo vệ transport chống overflow payload JSON (bao gồm diff và metadata), ví dụ `opts.MaxBytes + 64KB`. Đọc stream body qua `io.LimitReader(resp.Body, maxWireBytes + 1)`. Nếu vượt quá, trả về `ao.ErrPayloadTooLarge` fail-closed, không fallback ngầm.
   - **Giới hạn File Content (`opts.MaxBytes`)**: Sau khi parse envelope, kiểm tra `int64(len(contentBytes)) <= opts.MaxBytes`. Nếu nội dung vượt quá, trả về `ao.ErrPayloadTooLarge`.
2. **Kiểm tra Envelope & Validation Protocol Shape**:
   - Kiểm tra `envelope.SessionID == sessionID` và `envelope.Path == filePath`. Sai lệch trả về `*ao.ProtocolError`.
   - Kiểm tra JSON strictly: không cho phép trailing bytes (`dec.Decode(&trailing) == io.EOF`).
   - Kiểm tra cờ an toàn:
     * `envelope.Binary == true`: Từ chối với `*ao.ProtocolError` ("binary workspace file unsupported in text reader").
     * `envelope.Deleted == true`: Từ chối với `*ao.APIError` tương thích `ao.ErrNotFound` ("workspace file is marked deleted").
     * `envelope.ContentTruncated == true`: Từ chối với `*ao.ProtocolError` ("upstream file content truncated fail-closed").
3. **Giữ nguyên ranh giới lỗi và phân tầng kiến trúc**:
   - Sử dụng đúng hệ thống lỗi chuẩn hiện có: `*ao.TransportError`, `*ao.APIError` (với `c.decodeError`), `*ao.ProtocolError`, `ao.ErrPayloadTooLarge`.
   - Adapter chỉ chịu trách nhiệm lấy `contentBytes []byte`. Tuyệt đối **không parse `WorkerReport`**, không tạo `WorkerClaim`, không chuyển trạng thái `TaskState` (đây là trách nhiệm của supervisor core/review).
4. **Mock Harness Fixtures**:
   - Mock HTTP server trong P03 integration test harness phải trả đúng wire envelope `WorkspaceFileResponse` JSON của pinned AO.
   - Kiểm thử cả positive control và toàn bộ negative error cases (binary, deleted, contentTruncated, wire overflow, content overflow, malformed JSON, trailing payload).

---

## 3. Thiết Kế Vòng Đời Poller Done / Err

Để bảo đảm tính an toàn đa luồng và tái sử dụng, vòng đời `recovery.Poller` được thiết kế chặt chẽ:

1. **Đồng bộ hóa truy cập (Synchronization)**:
   - Các trường `done chan struct{}` và `lastErr error` được bảo vệ bằng `p.mu sync.Mutex`.
   - Phương thức `Done() <-chan struct{}` trả về channel hiện tại dưới lock.
   - Phương thức `Err() error` đọc `p.lastErr` dưới lock.
2. **Channel gắn riêng với từng lần `Start`**:
   - Mỗi lần gọi `Start(ctx)`, Poller khởi tạo channel mới: `p.done = make(chan struct{})`.
   - Khi Poller dừng, channel này được đóng. Nếu gọi lại `Start(ctx)` sau đó (restart), một channel mới được tạo, không tái sử dụng channel đã đóng.
3. **Thứ tự ghi lỗi trước khi đóng channel**:
   - Khi `PollOnce` gặp lỗi phân loại/kết nối không thể hồi phục:
     1. Khóa `p.mu.Lock()`.
     2. Ghi nhận `p.lastErr = err`.
     3. Đóng channel `close(p.done)`.
     4. Mở khóa `p.mu.Unlock()`.
   - Đảm bảo bất kỳ goroutine nào nhận tín hiệu từ `<-p.Done()` đều đọc được `p.Err() != nil` ngay lập tức mà không xảy ra race condition.
4. **Phân biệt lỗi và dừng sạch (Clean Stop / Cancellation)**:
   - Khi dừng do `Stop()` hoặc context caller bị hủy (`<-ctx.Done()`): `p.lastErr` giữ nguyên `nil`.
   - Host watcher phân biệt: nếu `p.Err() == nil` thì đây là shutdown bình thường; nếu `p.Err() != nil` thì mới là sự cố ngầm và kích hoạt chuyển `/readyz` sang 503.
5. **Quản lý Goroutine Watcher phía Host**:
   - Host bootstrap quản lý goroutine watcher (qua `sync.WaitGroup` hoặc tracking channel) và thực hiện join khi daemon shutdown drain, tránh rò rỉ goroutine.
6. **Kiểm thử hồi quy (Regression Tests)**:
   - Kiểm thử kịch bản Start -> Stop -> Start (restart).
   - Kiểm thử concurrency dưới cờ race detector (`-race`).

---

## 4. Đề Xuất Delta Cho Policy Catalog `go-test-p03-004`

Catalog profile `go-test-p03-004` đã được đăng ký trước đó tại `docs/audits/P03_ADR_017_EXTERNAL_REAUDIT_005.md` chỉ cho phép 3 package:
- `./internal/host/...`
- `./cmd/supervisor/...`
- `./test/integration/...`

Nhằm hỗ trợ kiểm thử đơn vị cho Poller asynchronous signal và AO workspace reader trong Contract Revision 2, Proposal này **ĐỀ XUẤT DELTA** mở rộng catalog (chờ External Supervisor phê duyệt, không tự nhận đã được duyệt):

```json
{
  "profile_id": "go-test-p03-004",
  "parameter_schema": {
    "type": "object",
    "properties": {
      "package": {
        "type": "string",
        "enum": [
          "./internal/host/...",
          "./cmd/supervisor/...",
          "./test/integration/...",
          "./internal/recovery/...",
          "./internal/ao/..."
        ]
      },
      "flags": {
        "type": "array",
        "items": { "type": "string" },
        "const": ["-v", "-race", "-count=1"]
      }
    },
    "required": ["package", "flags"],
    "additionalProperties": false
  },
  "cwd_policy": "worktree_root",
  "max_timeout_seconds": 300
}
```

---

## 5. Phạm Vi Cho Phép Thu Hẹp (Allowed Scope Revision 2)

```text
PHẠM VI CHO PHÉP (ALLOWED_SCOPE) ĐỀ XUẤT CHO CONTRACT 004 REVISION 2:

1. cmd/supervisor/**               (Host bootstrap daemon, poller watcher, negative stop CLI)
2. internal/host/**                    (Authority, lease, drain, db_handle, Named Pipe)
3. internal/recovery/poller.go         (Thêm Done() <-chan struct{} và Err() error synchronized)
4. internal/recovery/poller_test.go    (Unit tests cho Done/Err lifecycle, restart, race)
5. internal/ao/client.go               (Thêm method GetWorkspaceFile đọc envelope JSON)
6. internal/ao/types.go                (Thêm WorkspaceFileResponse, WorkspaceReadOptions, ErrPayloadTooLarge)
7. internal/ao/client_test.go          (Unit tests cho wire parsing, bounded read, status, cancellation)
8. test/integration/**                 (Integration harness sử dụng Client.GetWorkspaceFile với mock wire fixtures)
```

Mọi tệp tin khác thuộc `internal/recovery/**` và `internal/ao/**` tiếp tục nằm trong `forbidden_scope`.

---

## 6. Tiêu Chí Nghiệm Thu Mới Cho Revision 2

Giữ nguyên toàn bộ `AC-004-01` đến `AC-004-08` của Contract 01, bổ sung:

- **AC-004-09 (Poller Asynchronous Signal & Host Watcher Fail-Closed)**:
  `recovery.Poller` xuất phương thức `Done() <-chan struct{}` (gắn riêng cho mỗi lần Start) và `Err() error` được bảo vệ bằng mutex. Lỗi được ghi trước khi đóng done; phân biệt dừng sạch (`Err() == nil`) với lỗi ngầm (`Err() != nil`). Host watcher theo dõi signal, join khi shutdown; khi nhận lỗi ngầm, watcher gọi `auth.SetUnavailable()` và chuyển probe `/readyz` sang HTTP 503 mà không crash tiến trình hay nhả lock handle.
- **AC-004-10 (Typed AO Workspace File Retrieval via JSON Envelope)**:
  `ao.Client` cung cấp `GetWorkspaceFile(ctx, sessionID, filePath, opts)` thực hiện đúng wire contract của pinned AO (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`), giải mã envelope `WorkspaceFileResponse`, kiểm tra `sessionId`/`path`, từ chối `binary`/`deleted`/`contentTruncated`, kiểm soát giới hạn `MaxWireBytes` và `opts.MaxBytes` qua `io.LimitReader(max+1)` trả về `ErrPayloadTooLarge`. Không parse `WorkerReport` hay can thiệp `TaskState`. Mock harness kiểm chứng tích hợp thành công và các ca lỗi tiêu cực.

---

## 7. Kế Hoạch Change Governance & Trạng Thái Findings

1. **Trạng thái findings**:
   - `R1-001` = `CLOSED`
   - `R1-002` = `CLOSED`
   - `P03-004-R2-001` = `CLOSED` (Đã ghi nhận tại implementation commit `565e62f5446a5b675b44fef56168d10cc47510bd`)
   - `R1-003` = `CLOSED`
   - `R1-004` = `OPEN` (Phân loại là operational hardening mới cho Revision 2; chờ thẩm định và release Revision 2)
   - `R1-005` = `OPEN` (Chờ release Revision 2 để mở whitelist `internal/ao/client.go`)
   - `R1-006` = `CLOSED`
2. **Kế hoạch phát hành**:
   - Trình `PROPOSAL-P03-006` và hồ sơ ứng viên `CANDIDATE_TASK_CONTRACT_P03_004_REVISION_2.md` lên External Supervisor.
   - Trạng thái hồ sơ hợp đồng: `CANDIDATE_PENDING_EXTERNAL_AUDIT` (`NOT_RELEASED`).
   - Tuyệt đối chưa sửa mã nguồn Go trong `internal/ao/**` hoặc `internal/recovery/**` trước khi External Supervisor phê duyệt và phát hành hợp đồng.
