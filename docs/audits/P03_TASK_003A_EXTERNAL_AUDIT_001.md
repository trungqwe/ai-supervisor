# TASK-P03-003A — External Audit 001

> **Audited implementation SHA**: `c793fb66069e780f2fcc97129456fb5068b44f3d`
> **Branch**: `codex/p03-003a`
> **Contract**: `CONTRACT-TASK-P03-003A-01` (đã release; không thay đổi)
> **Base SHA**: `1daf0b91efc7f065eb07146d642c0b6f6c259212` (không thay đổi)
> **External Supervisor verdict**: `TASK_P03_003A_IMPLEMENTATION = REVISION_REQUIRED`
> **Date**: 2026-09-23

## Bằng chứng và phạm vi quyết định

Suite gốc của implementation tại SHA đã audit **PASS**: `go test -race ./internal/domain/... ./internal/store/...`, `go test ./...` và kiểm tra diff theo báo cáo implementation. External Supervisor thực hiện thêm **5 probe audit, kết quả FAIL**. Kết quả PASS của suite gốc không chứng minh bốn invariant dưới đây; thông tin được chuyển giao không kèm lệnh hoặc log riêng của từng probe, nên record này không gán kết quả giả định cho từng probe.

| Finding | Vấn đề được External Supervisor xác nhận | Yêu cầu remediation |
|---|---|---|
| `3A-R1-001` — Dispatch guard | Transaction commit `DISPATCH_BOUND` chưa kiểm tra đầy đủ session, lịch sử attempt và provisioning của Pair. | Trong cùng transaction, đòi session `CLEAN`, mọi attempt trước của task `CLEAN`, và không có provisioning stage `PROVISION_REQUESTED` hoặc `PROVISION_FAILED`; nếu từ chối thì rollback toàn bộ state, current_attempt, contract freeze, attempt, operation và audit. |
| `3A-R1-002` — Stop lineage | Stop gắn attempt chưa so khớp `session_id` và `terminal_generation` với snapshot của attempt. | Kiểm tra toàn bộ task/contract/Pair/session/generation trong transaction insert; xác định nhánh optional lineage theo purpose. `QUARANTINE_CLEANUP` có thể tham chiếu attempt đã đóng; `PAIR_MAINTENANCE` có thể không có attempt. |
| `3A-R1-003` — Deadline | Update hoặc retry có thể thay đổi `call_completed_at` và `confirmation_deadline_at` đã ghi. | Chỉ ghi lần đầu; replay đúng cùng giá trị có thể idempotent, ghi khác giá trị phải bị từ chối và giữ nguyên dữ liệu. |
| `3A-R1-004` — Stop resolution CAS | CAS hiện chỉ xét stage, cho phép stale writer ghi đè resolution. | CAS phải xét resolution hiện hành; confirmation thường không được biến `STOP_CONFIRMATION_TIMEOUT` thành `TERMINATION_CONFIRMED` hoặc mở lại `IN_FLIGHT`. Administrative reconciliation cần đường xử lý riêng có điều kiện và audit. |

## Governance sau audit

`TASK_P03_003A = RELEASED`; `TASK_P03_003A_IMPLEMENTATION = REVISION_REQUIRED`; `P03_CODE = AUTHORIZED_3A_ONLY`; `ACTIVE_GATE = TASK_P03_003A_REMEDIATION`. Parent `TASK-P03-003` chỉ release 3A. `TASK_P03_003B`, `TASK_P03_003C`, `TASK_P03_003D` vẫn `NOT_RELEASED`; `DESIGN_BLOCKER_3D_STARTUP_WIRING` được bảo lưu. Remediation chỉ trong `internal/domain/**` và `internal/store/**`; cần re-audit độc lập trước mọi tuyên bố phê duyệt implementation.
