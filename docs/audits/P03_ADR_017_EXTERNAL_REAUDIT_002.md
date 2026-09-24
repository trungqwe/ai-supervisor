# P03 ADR-017 / TASK-P03-004 External Design Re-Audit 002

- **Audited Commit SHA**: `da91f46afe923041e1d6c12556bf0a0823e1d375` trên nhánh `main`.
- **Audited Documents**:
  - `docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md` (Revision 2)
  - `docs/adr/DRAFT-ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md` (Revision 2)
  - `docs/plans/PLAN-HOST-INTEGRATION-DEPENDENCY.md` (Revision 4)
- **Authority**: External Supervisor Design Re-Audit 002
- **APPROVED DIRECTION**: Subtask `TASK-P03-004` thuộc Phase P03 (tiếp tục APPROVED_DIRECTION nhằm hoàn tất exit gate P03 trước P04).
- **DESIGN VERDICT**: `DRAFT_ADR_017 = REVISION_3_REQUIRED` (chưa release Task Contract, chưa viết Go).

---

## 1. Đánh giá Trạng thái Findings Đợt 2 (ADR17-R2-001..003)

| Finding | Phân loại & Yêu cầu Kỹ thuật | Trạng thái Đợt 2 | Trạng thái Đợt 3 |
|---|---|---|---|
| `ADR17-R2-001` | Windows Exclusive Sidecar Lock File Handle thay thế `Local\` Named Mutex | OPEN | REVISION_REQUIRED -> Chuyển thành `ADR17-R3-002` (cần chốt một thuật toán canonical DB/lock duy nhất, xử lý alias/hardlinks) và `ADR17-R3-003` (xác thực metadata/pipe) |
| `ADR17-R2-002` | Bỏ Production Verification Runner khỏi Scope P03 | OPEN | **CLOSED** (đã đóng ở cấp thiết kế: P03 không chứa production verification runner; EvidenceCollector, verification runner và tool surface thuộc P04/P05 theo `docs/22` và `docs/21`; citation `docs/11_CHATGPT_TOOL_SURFACE.md`) |
| `ADR17-R2-003` | Harness Kích hoạt 5 Bước Exit Gate trên Binary Thật | OPEN | REVISION_REQUIRED -> Chuyển thành `ADR17-R3-001` (xóa giả định daemon P03 có sẵn effectful HTTP routes, tách 2 track bằng chứng) |

---

## 2. Danh sách Findings Mới Cần Hiệu chỉnh (ADR17-R3-001..003)

| Finding | Phân loại & Yêu cầu Kỹ thuật | Trạng thái |
|---|---|---|
| `ADR17-R3-001` | **Tách Bạch Hai Track Bằng Chứng & Xóa Giả Định Effectful Supervisor HTTP Routes**: Xóa bỏ hoàn toàn giả định cho rằng daemon P03 đã có sẵn các effectful HTTP routes (tạo session, dispatch, workspace read, stop). Tuyệt đối không dùng route AO trực tiếp vì sẽ bỏ qua Supervisor sagas/guards. Tách rõ 2 loại bằng chứng: 1) Binary daemon thật (`cmd/supervisor`) chứng minh machine-wide exclusivity, startup-before-serve (admission probe), PendingAO hold, cạnh tranh tiến trình, và graceful shutdown drain; 2) P03 integration harness gọi trực tiếp các API điều phối nội bộ của thư viện (`internal/dispatch`, `internal/store`, `internal/stop`, `internal/ao`, `internal/recovery`) để kiểm chứng trọn vẹn 5 bước AO mà không cần production HTTP control surface hay code P04/P05. Nếu muốn thêm effectful Supervisor HTTP API, phải đưa qua governance riêng; không nhét vào ADR-017. | OPEN (đã xử lý trong Revision 3) |
| `ADR17-R3-002` | **Thuật toán Canonical DB & Lock Path Duy Nhất Trước Store.Open & Ma trận Alias/Session**: Chốt một thuật toán định danh duy nhất dùng Windows kernel handle canonicalization (`GetFinalPathNameByHandleW(VOLUME_NAME_DOS)`): nếu DB đã tồn tại thì mở handle DB file, nếu DB mới thì mở handle thư mục cha rồi ghép base name chuẩn hóa. Kiểm tra hard link (`nNumberOfLinks > 1`) và fail-closed nếu có hard link alias. Loại bỏ cách viết gộp "GetFinalPathNameByHandleW hoặc normalized path". Bổ sung ma trận test alias (relative path, casing, subst drive, junction/symlink, new vs existing DB, hard link) và test cạnh tranh giữa hai Windows logon sessions (`ERROR_SHARING_VIOLATION`). | OPEN (đã xử lý trong Revision 3) |
| `ADR17-R3-003` | **Bảo Mật Metadata .owner.json & Xác Thực Named Pipe Takeover**: File `.owner.json` chỉ là metadata không tin cậy nếu chưa xác thực. Owner phải ghi atomically (qua file tạm `.owner.json.tmp` và rename), giới hạn DACL chỉ cho owner SID/Admins/SYSTEM. Metadata chứa `pid`, `owner_instance_id` (cryptographic UUID), `pipe_name`, `started_at`, `canonical_db_path`. Xử lý file hỏng/stale fail-closed, tuyệt đối không đoán owner từ PID để hành động. Named Pipe phải xác thực client token/SID (`ImpersonateNamedPipeClient` / `GetNamedPipeHandleStateW`) và kiểm tra `owner_instance_id`. Yêu cầu không hợp lệ không được khiến owner shutdown. Lock acquisition thất bại thì fail-closed. | OPEN (đã xử lý trong Revision 3) |

---

## 3. Gate & Dependency Status
- `ACTIVE_GATE = TASK_P03_003D_HANDOFF_VERIFICATION` (chờ re-audit 003 đối với ADR-017 Revision 3 và Proposal-005 Revision 3)
- `HOST_QUIESCENCE_INTEGRATION = OPEN`
- `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
- Verified host principal unproven at runtime; `AUTOMATIC_RESTORE = DISABLED`.
