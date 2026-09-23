# ADR-016 clarification: audit phục hồi 3D

**Trạng thái:** `EXTERNAL_APPROVED` theo quyết định External Supervisor tháo `BLOCKER-3D-001/002` cho TASK-P03-003D. Clarification này chỉ bổ sung hai `event_type` và mở rộng một trường hợp dùng `STOP_OPERATION_RESOLVED`; không đổi DDL, AO wire, TaskState graph, accepted ADR, các clarification hay contract release lịch sử.

## 1. Post-send AO outage

`POST_SEND_RECOVERY_PENDING` chỉ áp dụng cho current open attempt chính xác ở `DISPATCHED/SEND_CONFIRMED` hoặc `RUNNING/SEND_CONFIRMED`. Một SQLite transaction CAS exact Pair/task/contract/attempt/session/generation, TaskState, dispatch stage và old disposition `NULL → RECOVERY_PENDING`, rồi ghi audit. Giữ TaskState, `ended_at`, snapshot và quarantine. Replay `RECOVERY_PENDING` chỉ đọc; disposition chẩn đoán mạnh hơn và protocol hold không bị hạ cấp. Pre-send giữ semantics/event riêng.

`POST_SEND_STATUS_RECOVERED` yêu cầu fresh valid `GetWorkerStatus` đúng session/generation. Một transaction CAS exact lineage, stage/state và `RECOVERY_PENDING → NULL` hoặc diagnostic được lifecycle matrix duyệt, đồng thời ghi audit và mọi TaskState transition. Terminal observation đi qua D12 trong transaction đó; không clear hold trước trong transaction riêng. GET không clear quarantine, chứng minh report/completion hay cấp quyền resend. CAS loser đọc lại winner; audit lỗi rollback toàn bộ.

Details hai event gồm lineage chính xác, dispatch operation/stage, old/new TaskState và disposition, observation time/source, error classification hoặc observed activity, actor và recovery invocation reference nếu có. Không lưu raw payload nhạy cảm hoặc thêm persisted token/column.

## 2. Stop intent cũ gặp target còn sống

Mở rộng `STOP_OPERATION_RESOLVED` cho `STOP_REQUESTED/IN_FLIGHT → STOP_REQUESTED/STOP_REISSUE_REQUIRES_HUMAN` khi recovery intent đã tồn tại và fresh GET đúng session/generation báo `isTerminated=false`. Transaction CAS old stage/resolution, purpose/lineage và restore ownership nếu linked; ghi `resolved_at` cùng audit. Giữ stage, call provenance/deadline, TaskState, `ended_at` và quarantine. Audit details đã duyệt bổ sung observation và lý do cấm automatic reissue. Replay chỉ đọc; stale writer không ghi đè winner. Resolution này không phải physical proof hay risk clearance, không cấp effect permit và không gọi `/kill`. Unknown/query failure không phải target alive. Nếu không chứng minh được intent đã tồn tại và caller effect đã hết quyền, fail closed để Supervisor quyết định ownership.

## 3. Quan hệ quyết định

Clarification này bổ sung ADR-016 D13 §§19/22 và clarification `ADR-016-CLARIFICATION-stop-operation-resolved-audit.md` đúng các trường hợp trên. Các trường hợp stop resolution khác giữ nguyên. `AUTOMATIC_RESTORE=DISABLED`; verified host principal và `DESIGN_BLOCKER_3D_STARTUP_WIRING` không thay đổi.
