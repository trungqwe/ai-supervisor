# P03 ADR-017 / TASK-P03-004 External Design Re-Audit 001

- **Audited Commit SHA**: `917a8ebcdfdd66b6ed1727df3eed16c919c555eb` trên nhánh `main`.
- **Audited Documents**:
  - `docs/proposals/PROPOSAL-P03-005-host-quiescence-and-daemon-bootstrap-integration.md` (Revision 1)
  - `docs/adr/DRAFT-ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md` (Revision 1)
  - `docs/plans/PLAN-HOST-INTEGRATION-DEPENDENCY.md` (Revision 3)
- **Authority**: External Supervisor Design Re-Audit 001
- **APPROVED DIRECTION**: Subtask `TASK-P03-004` thuộc Phase P03 (tiếp tục APPROVED_DIRECTION nhằm thỏa mãn exit gate P03 trước khi mở Phase P04).
- **DESIGN VERDICT**: `DRAFT_ADR_017 = REVISION_2_REQUIRED` (chưa release Task Contract, chưa viết Go, chưa tuyên bố Phase P03 hoàn tất).

---

## 1. Đánh giá Trạng thái Findings Đợt 1 (ADR17-R1-001..006)

| Finding | Phân loại & Yêu cầu Kỹ thuật | Trạng thái Đợt 1 | Trạng thái Đợt 2 |
|---|---|---|---|
| `ADR17-R1-001` | Tách `ProcessOwnerLease` vs `ExclusiveScope` | OPEN | REVISION_REQUIRED -> Chuyển thành `ADR17-R2-001` (khóa chưa chứng minh machine-wide exclusivity) |
| `ADR17-R1-002` | Cooperative Takeover V1 (Bỏ Auto-kill) | OPEN | **CLOSED** (đã đóng ở cấp thiết kế) |
| `ADR17-R1-003` | Chốt Cơ chế Owner Lock Chính trên Windows | OPEN | REVISION_REQUIRED -> Chuyển thành `ADR17-R2-001` (Local\ mutex bị cô lập session, cần đổi sang exclusive sidecar file handle) |
| `ADR17-R1-004` | Ma trận Kiểm chứng Binary Thật & Tách Mock/Live AO | OPEN | **CLOSED** (đã đóng ở cấp thiết kế) |
| `ADR17-R1-005` | Giới hạn Scope P03 Host & Đối soát SEC-003 | OPEN | REVISION_REQUIRED -> Chuyển thành `ADR17-R2-002` (production verification runner bị đưa nhầm vào P03) |
| `ADR17-R1-006` | Chính sách Vận hành UNSET & Fail-closed Runtime | OPEN | **CLOSED** (đã đóng ở cấp thiết kế) |

---

## 2. Danh sách Findings Mới Cần Hiệu chỉnh (ADR17-R2-001..003)

| Finding | Phân loại & Yêu cầu Kỹ thuật | Trạng thái |
|---|---|---|
| `ADR17-R2-001` | **Windows Exclusive Sidecar Lock File Handle thay thế `Local\` Named Mutex**: V1 sử dụng một Windows exclusive sidecar lock file handle (`CreateFileW`, `dwShareMode = 0`) làm `ProcessOwnerLease` chính, giữ từ TRƯỚC khi `Store.Open()`/migration đến SAU khi shutdown drain hoàn tất và `Store.Close()` đóng database. Tách file metadata (`.owner.json`) / Named Pipe để tiến trình khác đọc và yêu cầu cooperative takeover. Không dùng `Local\` named mutex làm bằng chứng exclusivity (vì bị cô lập theo logon session). Định nghĩa canonical DB/lock identity, local filesystem được hỗ trợ (NTFS/ReFS, cấm network/SMB shares), handle không kế thừa (`bInheritHandle = FALSE`), crash cleanup (kernel tự giải phóng handle) và cạnh tranh giữa hai Windows logon sessions (`ERROR_SHARING_VIOLATION`). Lock lỗi hoặc owner cũ không hợp tác => fail-closed; không auto-kill. | OPEN (đã xử lý trong Revision 2) |
| `ADR17-R2-002` | **Bỏ Production Verification Runner khỏi Scope P03**: Loại bỏ production verification runner khỏi scope `TASK-P03-004`. Phase P03 chỉ sở hữu host/bootstrap (`cmd/supervisor`), admission và test harness kiểm chứng exit gate. `EvidenceCollector`, verification runner và tool ChatGPT yêu cầu chạy profile thuộc sở hữu độc quyền của P04/P05 theo `docs/22_MODULE_PROVENANCE.md` và `docs/21_TRACEABILITY_MATRIX.md`. Sửa toàn bộ citation thành `docs/11_CHATGPT_TOOL_SURFACE.md`. Ghi rõ yêu cầu người dùng: ChatGPT Web là reasoning agent, Control Plane là cầu nối công cụ local; tuyệt đối không cấp arbitrary shell execution. | OPEN (đã xử lý trong Revision 2) |
| `ADR17-R2-003` | **Harness Kích hoạt 5 Bước Exit Gate trên Binary Thật (Khi Chưa có 12 Tool P05)**: Giữ ma trận binary thật cho 5 bước exit gate P03 (session create, dispatch, observation reconciliation, raw workspace-file read, teardown). Nêu cụ thể cơ chế test harness / binary kích hoạt các bước này qua local HTTP admission endpoints của daemon khi chưa có 12 tool P05. Network probe kiểm tra trạng thái startup-before-serve (port đóng fail-closed đến khi `r.ready == true`, mở sau khi hoàn tất). Log chỉ đóng vai trò bổ trợ. Chính sách vận hành thiếu giá trị vẫn fail-closed. | OPEN (đã xử lý trong Revision 2) |

---

## 3. Gate & Dependency Status
- `ACTIVE_GATE = TASK_P03_003D_HANDOFF_VERIFICATION` (chờ re-audit 002 đối với ADR-017 Revision 2 và Proposal-005 Revision 2)
- `HOST_QUIESCENCE_INTEGRATION = OPEN`
- `DESIGN_BLOCKER_3D_STARTUP_WIRING = PRESERVED`
- Verified host principal unproven at runtime; `AUTOMATIC_RESTORE = DISABLED`.
