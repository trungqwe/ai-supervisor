# TASK-P03-003D — External audit 002

**Audited implementation:** `d9d2fcc57ac95b098ac02f81686a891afa1aab19` trên `codex/p03-003d`; code base `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`.
**Contract release artifact:** `505319592d2360df62f2b31ce629f78942038b49` (`docs/tasks/TASK_CONTRACT_P03_003D_REVISION_3.md`).

**External Supervisor verdict:** `REVISION_REQUIRED`; chưa merge. Suite gốc race PASS. Hai probe Supervisor FAIL cho `3D-R2-001` và `3D-R2-002`; finding `3D-R1-005` chưa đóng (`3D-R1-005=OPEN`).

| Finding | Phân loại & Phạm vi kiểm toán | Trạng thái |
| --- | --- | --- |
| 3D-R2-001 | **Timeout của waiting_input**: Lifecycle ghi telemetry `AO_WAITING_INPUT_OBSERVED` nhưng `TimeoutMonitor` bỏ qua mọi disposition khác NULL, đồng thời `ReserveStopOperation` và `ValidateTimeoutEffectOwner` cũng đòi NULL, dẫn tới telemetry vô hiệu hóa execution deadline. Cần sửa đồng bộ eligibility và Store guards để telemetry `waiting_input` không chặn deadline, giữ exact CAS/observation hiện tại, không xóa disposition hàng loạt và tuân thủ strict precedence (`RECOVERY_PENDING`, protocol hold, blocked decision, quarantine, stop ownership). | OPEN (chờ re-audit) |
| 3D-R2-002 | **Validate policy trước effect**: Validation policy giữa coordinator và Store chưa đồng bộ. `PolicyRef` rỗng hoặc toàn whitespace bị Store từ chối sau HTTP 200, dẫn tới đưa dispatch vào D5 (`DELIVERY_OUTCOME_UNKNOWN`). Cần từ chối trước `SEND_REQUESTED` và trước AO `/send`; giữ snapshot policy cố định trong lần dispatch. | OPEN (chờ re-audit) |
| 3D-R1-005 | Injected execution deadline và durable terminal transaction / outcome. | OPEN (chưa đóng) |

## Kết quả kiểm tra đối chiếu

- **Suite gốc:** PASS (kiểm tra với `-race` và `-count=1`).
- **Probe Supervisor:** 2 probe FAIL:
  - Probe 1 (`3D-R2-001`): Task chuyển `RUNNING` với disposition `AO_WAITING_INPUT_OBSERVED` không được `TimeoutMonitor` kích hoạt timeout khi quá hạn.
  - Probe 2 (`3D-R2-002`): Dispatch với `PolicyRef` toàn whitespace gọi AO `/send` và rơi vào containment D5 thay vì bị từ chối trước effect.

## Remediation commit

- Remediation implementation: `7513f1b9f39fa459be15e0abc836c6258a610d8b` trên branch `codex/p03-003d`.
- Whitelist compliance: 23 files thay đổi từ `base_sha` `583e700eb125a08cc6bd7d63b6b27a6f3d4cc527`, 0 file ngoài 32 allowed paths của contract revision 3.
- `git diff 583e700eb125a08cc6bd7d63b6b27a6f3d4cc527..HEAD --check`: exit code 0.

## Gate & Dependency Status

- `TASK_P03_003D_IMPLEMENTATION=REVISION_REQUIRED`
- `P03_CODE=AUTHORIZED_3D_ONLY`
- `ACTIVE_GATE=TASK_P03_003D_REMEDIATION`
- `AUTOMATIC_RESTORE=DISABLED`
- `HOST_QUIESCENCE_INTEGRATION=OPEN`
- `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`
- Verified host principal vẫn là dependency mở; fake principal trong harness không chứng minh host authentication.
- Audit record này ghi nhận verdict của kiểm toán độc lập, không sửa audit lịch sử, không tự ý đóng finding. Chờ re-audit chính thức.
