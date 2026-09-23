# TASK-P03-003C — External audit 001

- **Audited implementation SHA:** `90a168daa2fcd114e1fcbba3c15319897c257b42`.
- **Code base_sha:** `246f74eabd09bbc2ea66624ee217035167824a86`.
- **Verdict từ External Supervisor:** `TASK_P03_003C_IMPLEMENTATION=REVISION_REQUIRED`; suite gốc PASS, ba Supervisor probe FAIL. Suite gốc không chứng minh các invariant bị probe bác bỏ.
- **3C-R1-001 OPEN:** D6 thiếu guard transaction-local trên mọi stop/restore risk hiện hành của Pair. Evidence stop cũ hoặc restore đã resolved có thể vượt risk mới chưa reconcile.
- **3C-R1-002 OPEN:** Class B cần tách WorkerSession lineage không có AttemptID khỏi TaskAttempt snapshot; acceptance phải bao phủ từng lineage chính xác.
- **3C-R1-003 OPEN:** Hai nonphysical terminal resolution `STOP_GENERATION_MISMATCH` và `STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED` thiếu audit `event_type` riêng. Supervisor phê duyệt clarification `STOP_OPERATION_RESOLVED` cho hai trường hợp này.
- **Giới hạn:** Record này ghi verdict và quyết định thiết kế tại audited SHA; remediation đang chờ re-audit, không tự đóng finding hay phê duyệt implementation. Contract `TASK_CONTRACT_P03_003C.md` vẫn bất biến; clarification không đổi scope hoặc AC nên không tạo superseding revision.
- **Gate:** `P03_CODE=AUTHORIZED_3C_ONLY`; `ACTIVE_GATE=TASK_P03_003C_REMEDIATION`; 3D `NOT_RELEASED`, `AUTOMATIC_RESTORE=DISABLED`, verified host principal vẫn là dependency fail-closed.
