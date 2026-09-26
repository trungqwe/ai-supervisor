# PROPOSAL-P03-006: Seam Scope Reconciliation and Task Contract P03-004 Revision 2

> **Status**: EXTERNAL_APPROVED (Phê chuẩn bởi External Supervisor theo docs/audits/P03_TASK_004_REVISION_2_RELEASE_AUDIT.md)
> **Authority**: docs/24_CHANGE_GOVERNANCE.md (Level 3 Canonical Architecture / Level 6 Task Contract Governance)
> **Active Gate**: TASK_P03_004_REMEDIATION
> **Target Task Contract**: CONTRACT-TASK-P03-004-02 (Revision 2 của TASK-P03-004)
> **Affected Specifications**: docs/tasks/TASK_CONTRACT_P03_004.md (Draft Revision 2), docs/14_FAILURE_RECOVERY.md, docs/sources/SOURCE_REGISTRY.md
> **Architectural Alignment**: Hoàn toàn nhất quán với ADR-017 (Accepted), ADR-016 (Accepted), ADR-011 (Accepted). Không phát hiện xung đột kiến trúc với các ADR đã duyệt; không yêu cầu sửa đổi ADR-017.

---

## 1. Căn Cứ Quản Trị & Vấn Đề Cần Giải Quyết

Trong quá trình thực hiện và tái thẩm định TASK-P03-004 (commit `09b47bf`, `ef0a496`, và `565e62f`), External Supervisor đã ghi nhận 2 điểm nối kiến trúc (architectural seams) vượt ra ngoài phạm vi cho phép (`allowed_scope`) của `CONTRACT-TASK-P03-004-01`:

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
   - Hoàn toàn chưa có typed method để đọc workspace file. Bước 4 trong `test/integration/ao_harness_test.go` hiện phải gọi raw HTTP (`http.NewRequestWithContext`), dù đã bổ sung cơ chế defensive read giới hạn kích thước qua `io.LimitReader`.
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

1. **WorkspaceReadOptions với hai giới hạn độc lập**:
   ```go
   type WorkspaceReadOptions struct {
       MaxWireBytes int64 // Bounded limit for wire JSON envelope payload (> 0)
       MaxBytes     int64 // Bounded limit for decoded file content (> 0)
   }
   ```
   - Cả `MaxWireBytes` và `MaxBytes` bắt buộc phải được truyền tường minh và `> 0`.
   - Kiểm tra overflow trước khi thực hiện `+ 1` (ví dụ `if opts.MaxWireBytes >= math.MaxInt64 { return ErrBadRequest }`).
   - **Tuyệt đối cấm** sử dụng công thức giả định `MaxBytes + 64KB` hay bất kỳ fallback giá trị ngầm định nào.
2. **Đọc stream có giới hạn & Phân biệt Wire vs Content Limit**:
   - Đọc tối đa `MaxWireBytes + 1` bytes của HTTP response body qua `io.LimitReader(resp.Body, opts.MaxWireBytes + 1)`.
   - Nếu tổng số bytes đọc được vượt quá `opts.MaxWireBytes`, lập tức trả về `ao.ErrPayloadTooLarge` fail-closed.
   - Sau khi decode JSON thành công, trích xuất `contentBytes := []byte(envelope.Content)`.
   - Kiểm tra `int64(len(contentBytes)) <= opts.MaxBytes`. Nếu nội dung vượt quá `opts.MaxBytes`, trả về `ao.ErrPayloadTooLarge`.
   - Các ca kiểm thử bắt buộc:
     * Envelope lớn do diff nhưng content nhỏ (content <= MaxBytes, wire <= MaxWireBytes -> thành công).
     * Wire overflow (`wireBytes > MaxWireBytes` -> ErrPayloadTooLarge).
     * Content overflow (`contentBytes > MaxBytes` dù wire hợp lệ -> ErrPayloadTooLarge).
     * Kiểm thử sát biên (`wireBytes == MaxWireBytes`, `wireBytes == MaxWireBytes + 1`, `contentBytes == MaxBytes`, `contentBytes == MaxBytes + 1`).
3. **Kiểm tra Envelope & Validation Required Fields**:
   - Kiểm tra sự hiện diện và kiểu dữ liệu của toàn bộ 15 required fields: `sessionId`, `path`, `content`, `binary`, `deleted`, `contentTruncated`, `size`, `status`, `workspaceVersion`, `diff`, `diffTruncated`, `editable`, `fileFingerprint`, `additions`, `deletions` (cùng kiểm tra giá trị enum hợp lệ cho `status`: `unmodified`, `modified`, `added`, `deleted`).
   - **Không để zero value của Go che mất trường thiếu**: sử dụng struct có con trỏ hoặc validation map/raw JSON để bảo đảm các trường bắt buộc thực sự tồn tại trong payload JSON.
   - Kiểm tra `sessionId == requestedSessionID` và `path == requestedFilePath`; sai lệch trả về `*ao.ProtocolError`.
   - Từ chối payload có dữ liệu thừa sau JSON (kiểm tra `dec.Decode(&trailing) == io.EOF`).
4. **Xử lý trạng thái file và phân định mã lỗi**:
   - `envelope.Binary == true`: Từ chối fail-closed với `*ao.ProtocolError` ("binary workspace file unsupported in text reader").
   - `envelope.ContentTruncated == true`: Từ chối fail-closed với `*ao.ProtocolError` / `ao.ErrPayloadTooLarge` ("upstream file content truncated").
   - `envelope.Deleted == true`: **HTTP 200 với deleted=true là kết quả không dùng được**. Tuyệt đối **không tự tạo `APIError/404`** như thể AO đã trả lỗi wire HTTP. Trả về typed error phù hợp:
     `var ErrWorkspaceFileDeleted = errors.New("ao: workspace file is marked deleted")`
     Hỗ trợ kiểm tra qua `errors.Is(err, ao.ErrWorkspaceFileDeleted)` và `errors.As`.
5. **Giữ nguyên ranh giới lỗi và phân tầng kiến trúc**:
   - Giữ nguyên hệ thống lỗi hiện có: `*ao.TransportError`, `*ao.APIError` (với `c.decodeError`), `*ao.ProtocolError`, `ao.ErrPayloadTooLarge`, `ao.ErrWorkspaceFileDeleted`.
   - Adapter chỉ trả về `contentBytes []byte`. Tuyệt đối **không parse `WorkerReport`**, không tạo `WorkerClaim`, không chuyển trạng thái `TaskState` (trách nhiệm thuộc supervisor review/domain).
6. **Mock Harness Fixtures**:
   - Mock HTTP server trong P03 integration test harness phải trả đúng wire envelope `WorkspaceFileResponse` JSON của pinned AO.
   - Kiểm thử cả positive control và toàn bộ negative error cases (binary, deleted, contentTruncated, wire overflow, content overflow, boundary limits, malformed JSON, trailing payload).

---

## 3. Thiết Kế Vòng Đời Poller Done / Err

Vòng đời `recovery.Poller` được chuẩn hóa chặt chẽ theo các nguyên tắc:

1. **Đồng bộ hóa truy cập (Thread-safe)**:
   - Các trường `done chan struct{}` và `lastErr error` được bảo vệ bằng `p.mu sync.Mutex`.
   - `Done() <-chan struct{}` trả về channel dưới lock.
   - `Err() error` đọc `p.lastErr` dưới lock.
2. **Channel gắn riêng với từng lần `Start` & Close đúng một lần**:
   - Mỗi lần gọi `Start(ctx)`, Poller khởi tạo channel mới: `p.done = make(chan struct{})`.
   - Đảm bảo `close(p.done)` được gọi đúng một lần duy nhất trên channel của vòng `Start` đó (thông qua `defer close(p.done)` duy nhất của goroutine loop), **tránh double close** với bất kỳ thao tác đóng thủ công nào khác.
   - Khi restart (`Start` sau `Stop`), một channel hoàn toàn mới được cấp phát, không tái sử dụng channel đã đóng.
3. **Thứ tự ghi lỗi trước khi đóng channel**:
   - Khi `PollOnce` gặp lỗi phân loại/kết nối không thể hồi phục:
     1. Khóa `p.mu.Lock()`.
     2. Ghi nhận `p.lastErr = err`.
     3. Mở khóa `p.mu.Unlock()`.
     4. Thoát goroutine loop để kích hoạt `defer close(p.done)`.
   - Bảo đảm bất kỳ goroutine nào nhận tín hiệu từ `<-p.Done()` đều đọc được `p.Err() != nil` ngay lập tức mà không có race condition.
4. **Phân biệt lỗi và dừng sạch (Clean Stop / Cancellation)**:
   - Khi dừng do `Stop()` hoặc context caller bị hủy (`<-ctx.Done()`): `p.lastErr` giữ nguyên `nil`.
   - Host watcher phân biệt: nếu `p.Err() == nil` thì đây là shutdown bình thường; nếu `p.Err() != nil` thì mới là sự cố ngầm và kích hoạt chuyển `/readyz` sang 503.
5. **Quản trị Watcher phía Host**:
   - Host daemon giữ đúng channel của vòng `Start` hiện tại và quản lý watcher goroutine (qua `sync.WaitGroup` hoặc tracking channel), bảo đảm watcher được join đầy đủ khi daemon shutdown drain.
6. **Kiểm thử hồi quy (Regression Tests)**:
   - Kiểm thử lỗi thật từ `PollOnce`.
   - Kiểm thử clean `Stop()` và context cancellation (`Err() == nil`).
   - Kiểm thử restart (`Start` -> `Stop` -> `Start` -> `Stop`).
   - Kiểm thử concurrency dưới cờ race detector (`-race`).

---

## 4. Đề Xuất Delta Cho Policy Catalog `go-test-p03-004`

Catalog profile `go-test-p03-004` ban đầu (tại `docs/audits/P03_ADR_017_EXTERNAL_REAUDIT_005.md`) chỉ cho phép 3 package:
- `./internal/host/...`
- `./cmd/supervisor/...`
- `./test/integration/...`

Nhằm hỗ trợ 5 verification requests cho Revision 2, Proposal này ghi nhận catalog delta mở rộng thêm 2 package:
- `./internal/recovery/...`
- `./internal/ao/...`

Trạng thái catalog delta: **`APPROVED_AT_DESIGN_LEVEL`** (đã thống nhất tại cấp thiết kế kiến trúc, Stage B host runtime vẫn giữ trạng thái **`UNVERIFIED`** cho đến khi task được dispatch và thực thi). Tuyệt đối không tự tuyên bố đã chạy verification của mã nguồn chưa viết.

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
3. internal/recovery/poller.go         (Thêm Done() <-chan struct{} và Err() error synchronized, single close)
4. internal/recovery/poller_test.go    (Unit tests cho Done/Err lifecycle, restart, race)
5. internal/ao/client.go               (Thêm method GetWorkspaceFile đọc envelope JSON với MaxWireBytes và MaxBytes)
6. internal/ao/types.go                (Thêm WorkspaceFileResponse, WorkspaceReadOptions, ErrPayloadTooLarge, ErrWorkspaceFileDeleted)
7. internal/ao/client_test.go          (Unit tests cho wire parsing, required fields, bounded read, status, cancellation)
8. test/integration/**                 (Integration harness sử dụng Client.GetWorkspaceFile với mock wire fixtures)
```

Mọi tệp tin khác thuộc `internal/recovery/**` và `internal/ao/**` tiếp tục nằm trong `forbidden_scope`.

---

## 6. Tiêu Chí Nghiệm Thu Cho Revision 2

Giữ nguyên toàn bộ `AC-004-01` đến `AC-004-08` của Contract 01, bổ sung:

- **AC-004-09 (Poller Asynchronous Signal & Host Watcher Fail-Closed)**:
  `recovery.Poller` xuất phương thức `Done() <-chan struct{}` (cấp phát mới cho mỗi lần Start) và `Err() error` được bảo vệ bằng mutex; bảo đảm đóng channel đúng một lần trên channel của vòng Start, tránh double close. Lỗi được ghi nhận trước khi đóng channel; phân biệt dừng sạch (`Err() == nil`) với lỗi ngầm (`Err() != nil`). Host watcher giữ đúng channel của vòng Start, join khi shutdown; khi nhận lỗi ngầm, watcher gọi `auth.SetUnavailable()` và chuyển probe `/readyz` sang HTTP 503 mà không crash tiến trình hay nhả lock handle. Kiểm thử hồi quy bao gồm lỗi thật PollOnce, clean stop, cancel, restart và race.
- **AC-004-10 (Typed AO Workspace File Retrieval via JSON Envelope)**:
  `ao.Client` cung cấp `GetWorkspaceFile(ctx, sessionID, filePath, opts)` với `opts.MaxWireBytes` và `opts.MaxBytes` độc lập (>0, có kiểm tra integer overflow trước max+1, không dùng fallback ngầm); đọc tối đa `MaxWireBytes+1`; giải mã `WorkspaceFileResponse` JSON từ pinned AO (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`), kiểm tra sự hiện diện và kiểu của toàn bộ required fields không để zero-value che mất trường thiếu; từ chối trailing data, binary, contentTruncated, và deleted files (HTTP 200 với deleted=true trả về `ErrWorkspaceFileDeleted`, không tự tạo 404 APIError); kiểm tra byte length của content <= MaxBytes với `ErrPayloadTooLarge`; bảo toàn hệ thống lỗi `TransportError`/`APIError`/`ProtocolError` mà không parse `WorkerReport` hay can thiệp `TaskState`. Mock harness kiểm chứng envelope lớn nhưng content nhỏ, wire overflow, content overflow, giới hạn sát biên và các ca lỗi tiêu cực.

---

## 7. Kế Hoạch Change Governance & Trạng Thái Findings

1. **Trạng thái findings**:
   - `R1-001` = `CLOSED`
   - `R1-002` = `CLOSED`
   - `P03-004-R2-001` = `CLOSED` (Đã ghi nhận tại implementation commit `565e62f5446a5b675b44fef56168d10cc47510bd`)
   - `R1-003` = `CLOSED`
   - `R1-004` = `OPEN` (Phân loại là operational hardening mới cho Revision 2; chờ release Revision 2)
   - `R1-005` = `OPEN` (Chờ release Revision 2 để mở whitelist `internal/ao/client.go`)
   - `R1-006` = `CLOSED`
2. **Kế hoạch phát hành**:
   - Trình `PROPOSAL-P03-006` và hồ sơ ứng viên `CANDIDATE_TASK_CONTRACT_P03_004_REVISION_2.md` lên External Supervisor.
   - Trạng thái hồ sơ hợp đồng: `CANDIDATE_PENDING_EXTERNAL_AUDIT` (`NOT_RELEASED`).
   - Dừng chờ release audit; tuyệt đối chưa sửa mã nguồn Go trong `internal/ao/**` hoặc `internal/recovery/**`.
