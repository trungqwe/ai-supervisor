# ADR-016 clarification: audit cho logical stop resolution

**Trạng thái:** `EXTERNAL_APPROVED` theo quyết định External Supervisor trong yêu cầu remediation TASK-P03-003C tại implementation SHA `90a168daa2fcd114e1fcbba3c15319897c257b42`.

## Phạm vi bổ sung

Clarification này bổ sung đúng một `audit_events.event_type`: `STOP_OPERATION_RESOLVED`. Nó áp dụng khi `stop_operations.resolution_state` chuyển từ `IN_FLIGHT` sang `STOP_GENERATION_MISMATCH` hoặc `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED`, vì hai resolution này chưa có event riêng. Event ghi nhận **kết quả logic** của stop; không phải bằng chứng termination, physical proof, acceptance rủi ro hay quarantine clearance. Các outcome đã có event riêng tiếp tục dùng event đó.

Transaction thực hiện CAS theo stop operation và old stage/resolution, cập nhật stage/resolution, đóng TaskState/attempt theo D11/D12 nếu là live stop, giữ quarantine khi chưa có physical proof, rồi ghi `STOP_OPERATION_RESOLVED` trong **cùng transaction**. Nếu CAS hoặc audit lỗi, toàn bộ transaction rollback. Audit details gồm `stop_operation_id`, `purpose`, Pair và lineage chính xác (session/generation, task/contract/attempt nếu có), old/new stage, old/new resolution, observation (session, generation, `isTerminated` khi có) và actor. Event này không tự cho phép `/kill` lại hoặc D6 clearance.

Clarification chỉ bổ sung registry audit theo ADR-016 D11/D12. Accepted ADR-016, addendum restore, contract release và audit lịch sử giữ nguyên. Không đổi DDL, AO wire API, TaskState graph hay nghĩa của `ADMINISTRATIVE_RISK_ACCEPTED`, `QUARANTINE_RESOLVED_ADMINISTRATIVE` và `DELIVERY_OUTCOME_UNKNOWN`.
