# TASK-P03-003D — External audit 001

**Audited implementation:** `710b27f2d41dcbadd385c6ac07e30bb1765d1cbe` trên `codex/p03-003d`; code base `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`.

**External Supervisor verdict:** `REVISION_REQUIRED`; `3D-R1-001..005=OPEN`. Suite gốc PASS. Ba probe Supervisor FAIL cho R1-001, R1-002 và R1-003; R1-004 và R1-005 là kết quả code review, không phải probe đã chạy. Suite gốc không chứng minh các nhánh thiếu dưới đây.

| Finding | Phạm vi thiếu tại SHA đã audit |
| --- | --- |
| 3D-R1-001 | `PollOnce` public thiếu cùng ownership boundary; tick được cấp quyền có thể cạnh tranh với repeated `Run`, shutdown chưa drain mọi tick public. |
| 3D-R1-002 | Poller generic observation có thể tranh với stop lifecycle; thiếu reconciliation stop-owned `STOP_CALL_SUCCEEDED/IN_FLIGHT` và guard transaction-local. |
| 3D-R1-003 | Poller bỏ `DISPATCH_BOUND`, nên không giải được pre-send outage hold sau startup. |
| 3D-R1-004 | Handoff availability bị hardcode unavailable; thiếu dependency hợp lệ cho nhánh D8. |
| 3D-R1-005 | Injected execution deadline chỉ được validate, chưa được áp dụng với mốc thời gian bền vững và terminal transaction xác định. |

**Gate hiện hành:** `TASK_P03_003D_IMPLEMENTATION=REVISION_REQUIRED`; `P03_CODE=AUTHORIZED_3D_ONLY`; `ACTIVE_GATE=TASK_P03_003D_REMEDIATION`. `HOST_QUIESCENCE_INTEGRATION=OPEN`, `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`, `AUTOMATIC_RESTORE=DISABLED`; verified host principal vẫn là dependency. Audit này ghi nhận verdict, không tự đóng finding hoặc phê duyệt remediation.

**BLOCKER-3D-R1-005 — quyết định deadline còn thiếu:** ADR-016 D9 chỉ yêu cầu monitor khi `waiting_input` vượt execution deadline; FR-006 yêu cầu bounded execution policy và REC-004 chuyển sang live stop. Contract revision 2 chỉ yêu cầu inject `SUPERVISOR_EXECUTION_DEADLINE`. Persisted `task_attempts.started_at` được ghi lúc prepare dispatch, trước HTTP 200 và trước quan sát `RUNNING`; `dispatch_operations.confirmed_at` chứng minh HTTP 200, không chứng minh lúc execution bắt đầu. Chưa có quy định chọn mốc nào làm deadline trên restart, cũng chưa có transaction/disposition/audit cụ thể cho timeout khi stop coordinator 3C nằm ngoài whitelist 3D. Supervisor cần chốt mốc bền vững, cách xử lý khi mốc vắng mặt/không nhất quán, đường khởi tạo `RUNNING_ATTEMPT_STOP` và effect ownership theo REC-004, cùng outcome nguyên tử cho timeout; nếu cần `internal/stop/**`, token hoặc schema thì phải revision scope/ADR tương ứng. Cho đến quyết định này, không báo AC deadline monitor PASS và không tự dùng `started_at` hay `confirmed_at` làm execution start.
