# P03 TASK-P03-003C — Blocker triển khai 001

- **Trạng thái:** `TASK_P03_003C_IMPLEMENTATION=BLOCKED`; contract đã release vẫn có hiệu lực. Chưa có commit implementation để audit.
- **Release artifact SHA:** `7dc3732a5a3cea58800ae3a370fcee9dd64ac742`.
- **Code base SHA:** `246f74eabd09bbc2ea66624ee217035167824a86`.
- **Worktree implementation:** `codex/p03-003c`, thay đổi hiện chưa commit và tách khỏi governance/main.

## BLOCKER-3C-001 — audit literal cho administrative stop resolution trước Tx D

ADR-016 §4 Canonical Token Registry liệt kê các `audit_events.event_type` đã duyệt, trong đó có `QUARANTINE_RESOLVED_ADMINISTRATIVE`, nhưng không liệt kê `ADMINISTRATIVE_RISK_ACCEPTED` như một audit event type. Trong khi đó, ADR-016 D5 yêu cầu Class C risk acceptance được ghi bền vững trong audit log dưới tên `ADMINISTRATIVE_RISK_ACCEPTED`. Addendum đã duyệt §5(3) yêu cầu linked stop đã claim nhưng chưa terminal phải có D11 resolution **trước** restore Tx D; §5(4) yêu cầu D6 quarantine clearance ở transaction riêng sau đó. `UpdateStopOperationStage` chủ ý từ chối administrative resolution thiếu audited reconciliation riêng.

Không thể phát `QUARANTINE_RESOLVED_ADMINISTRATIVE` khi mới resolve stop trước Tx D vì quarantine vẫn còn hiệu lực đến D6. `PAIR_RESTORE_RESOLVED` thuộc Tx D, sau stop resolution. Dùng lại `PAIR_RESTORE_RISK_ACCEPTED` của authorization restore dùng một lần sẽ nhập nhằng với recovery authority độc lập về sau. Chỉ ghi resolution trong audit details cũng không xác định được `event_type` đã duyệt cho quyết định này. Chọn event type mới hoặc tái sử dụng với nghĩa khác sẽ đổi semantics đã chấp thuận, vượt contract 3C.

**Quyết định cần từ External Supervisor:** chỉ định `event_type` và CAS/audit transaction chính xác khi chuyển `stop_operations.resolution_state=ADMINISTRATIVE_RISK_ACCEPTED` trước Tx D, gồm Class C linked `STOP_CALL_OUTCOME_UNKNOWN`/timeout/mismatch và Class B 404; xác định đây là clarification của registry hay cần ADR addendum. Giữ audit D6 clearance riêng và quarantine lock đến khi Tx D rồi D6 hoàn tất. Không reissue `/kill`.

Phần WIP độc lập đã có `go test -count=1 ./internal/stop/... ./internal/store/...` PASS và `git diff --check` sạch; **không** suy ra AC-3C-01..12 hoặc contract verification PASS. Bốn verification requests, full regression và implementation whitelist audit chưa hoàn tất. Không gọi AO thật hoặc migration database người dùng. `AUTOMATIC_RESTORE=DISABLED`; verified host principal tiếp tục là dependency fail-closed; 3D vẫn `NOT_RELEASED`.
