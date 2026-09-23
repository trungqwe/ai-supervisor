# TASK-P03-003B External Re-audit 001

- Audited implementation SHA: `d42b8086a6a54dcf191d8880e044c21d61ad23eb`.
- Governance baseline: `a1c7d3521e86d77ff6747d31058a806d94c13b20`.
- Release artifact SHA: `333b3922a2b79a6a414003d1959060c9a72d37eb`.
- Contract code `base_sha`: `b8b0c95576d87677e8d48210d9838cc2f599752a`.
- External Supervisor verdict: `REVISION_REQUIRED`. Bộ kiểm thử implementation gốc **PASS**; ba probe re-audit của Supervisor **FAIL**. Đây là kết quả được Supervisor cung cấp, không phải kiểm chứng độc lập trong commit governance này.

## Trạng thái finding

| Finding | Trạng thái tại SHA được audit | Bằng chứng còn thiếu |
|---|---|---|
| `3B-R1-001` | `PARTIALLY_CLOSED` | Class A phải kiểm tra deadline strict và physical evidence bao phủ runtime hiện tại cùng từng quarantined execution lineage theo addendum §5.4. |
| `3B-R1-002` | `CLOSED` | Giữ CAS `TERMINATED`, xác nhận `GetWorkerStatus` sau AO 200 và khóa khi outcome không xác định. |
| `3B-R1-003` | `CLOSED` | Giữ reread durable tuple và containment CAS sau confirmation rollback hoặc invalid response; không resend. |
| `3B-R1-004` | `CLOSED` | Giữ đúng `audit_events.event_type` đã duyệt. |
| `3B-R1-005` | `PARTIALLY_CLOSED` | Chứng minh no-attempt `PAIR_MAINTENANCE` từ restore unknown và positive-control rollback của claim + stop + hai audit events. |
| `3B-R2-001` | `OPEN` | `termination_confirmed_at < confirmation_deadline_at` ở stop confirmation và physical-resolution validation; equality phải bị từ chối nguyên tử. |
| `3B-R2-002` | `OPEN` | Một linked stop không đủ chứng minh Class A cho mọi lineage; thiếu coverage phải giữ lock/quarantine. |
| `3B-R2-003` | `OPEN` | Observation/authority hợp lệ cho no-attempt cleanup; test trigger phải chứng minh đúng injected audit failure và rollback đầy đủ. |

## Giới hạn và gate

Re-audit này chỉ đánh giá implementation tại SHA nêu trên. Không coi bộ kiểm thử gốc PASS là kết quả của ba probe Supervisor, không kết luận sửa đổi tiếp theo đã được phê duyệt, và không dùng fake test principal làm bằng chứng host authentication. Kết quả remediation mới cần được Supervisor re-audit riêng.

`TASK_P03_003B_IMPLEMENTATION=REVISION_REQUIRED`; `P03_CODE=AUTHORIZED_3B_ONLY`; `ACTIVE_GATE=TASK_P03_003B_REMEDIATION`. `AUTOMATIC_RESTORE=DISABLED`; 3C/3D `NOT_RELEASED`; dependency verified host principal và `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`. Không merge implementation vào main.
