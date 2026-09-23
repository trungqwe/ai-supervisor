# PROPOSAL-P03-003: Restore ownership và dispatch pre-send semantics

## Metadata

- **Author / Date**: Codex / 2026-09-23
- **Related Phase / Task**: P03 / TASK-P03-003B
- **Status**: `PENDING_REVIEW` cho detailed design; direction đã được Supervisor phê duyệt, **không** phải `EXTERNAL_APPROVED` cho toàn proposal.
- **Decision baseline**: `95bf8e3329d0261eeafdcd035cbe97dbe26b12ce`; contract `base_sha=b8b0c95576d87677e8d48210d9838cc2f599752a`.
- **Design artifact**: `docs/adr/DRAFT-ADR-016-ADDENDUM-restore-and-pre-send-semantics.md` — `DRAFT_PENDING_EXTERNAL_APPROVAL`.
- **Upstream pin**: `Untrivial-ai/agent-orchestrator` v0.13.0, `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.
- **Open blockers**: `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL`; `DESIGN_BLOCKER_3D_STARTUP_WIRING` được bảo lưu.

## 1. Observation

ADR-016 D3 yêu cầu giữ WorkerSession hiện hữu và restore thay vì spawn replacement; D4 giữ snapshot attempt bất biến; D5–D6 giữ unknown delivery/quarantine; D7/§22 quy định pre-send; D11 phân biệt stop purpose; D12 quy định atomic terminal closure. Draft 3B phát hiện thiếu durable restore intent/exclusion, operator recovery, pre-send error recovery và literal `dispatch_operations.resolution_state`. `WORKER_TERMINATION_UNKNOWN` (§4/registry), `STALE_EXECUTION_GENERATION` (D7) và `TASK_STATE_TRANSITION`/attempt closure (D12) **đã có authority**, không chờ quyết định mới.

`docs/02_REQUIREMENTS.md` FR-005/FR-006/NFR-003/NFR-005, `docs/12_UPSTREAM_INTEGRATION.md`, `docs/14_FAILURE_RECOVERY.md` REC-002, `docs/sources/SOURCE_REGISTRY.md` và `REUSE_MATRIX.md` giữ AO làm chủ process/worktree/harness; Supervisor sở hữu durable Pair/Task state và audit.

## 2. Proposed Idea và phần direction đã được duyệt

Supervisor **đã phê duyệt hướng** durable restore operation + guard theo Pair, pinned AO **chỉ operator-authorized restore**, `AUTOMATIC_RESTORE=DISABLED`, cấm reissue `/restore` khi outcome không xác định. Supervisor cũng chấp nhận semantics `DELIVERY_OUTCOME_UNKNOWN`: terminal cho dispatch operation, **không** đồng nghĩa quarantine clearance. Mục tiêu restore vẫn thuộc 3B. Các decision này cho phép soạn addendum draft; chưa phê duyệt DDL, toàn bộ transition protocol, contract release hay production code.

Addendum draft khuyến nghị one-shot authorization ràng buộc operation ID/Pair/session/expected generation/risk scope, reservation/audit trong một transaction trước AO effect, positive HTTP 200 confirmation trong transaction sau effect, Pair-wide guard cho dispatch/provisioning/stop, và operator recovery/clearance tách biệt. Chỉ đường linked stop có exact restore operation và D11 purpose hợp lệ mới đi qua restore unresolved; các caller khác tiếp tục bị khóa. Pre-send query outage dùng hold trên attempt đang mở, identity mismatch/quarantine và operator abandonment dùng D12 terminal closure. DDL, token, CAS, guard matrix và test oracle nằm trong addendum để External Supervisor audit một lượt.

## 3. Problem Solved

- Chặn hai caller cùng `/restore` và các caller loại khác dùng cùng Pair trong crash window.
- Không biến GET generation mới, HTTP 404/409 hay mất response thành bằng chứng causal restore hoặc physical termination.
- Cho operator nhận quyền phục hồi, cleanup và clearance theo đúng thứ tự mà không mở lane sớm.
- Tách dispatch operation đã terminal khỏi quarantine còn hiệu lực; stale confirmation không ghi đè containment.

## 4. Existing Upstream Capability Checked

| Nguồn tại đúng pin | Wire/implementation fact | Giới hạn |
|---|---|---|
| `backend/internal/httpd/apispec/openapi.yaml:5245-5274,10693-10712`; `controllers/sessions.go:195,1250-1261`; `controllers/dto.go:722-728` | `POST /api/v1/sessions/{sessionId}/restore`, response 200 `{ok,sessionId,restoreMode,session}`, 404/409/500. | Không request operation ID/idempotency key; session ID lặp lại không phải dedup proof. |
| `backend/internal/service/session/service.go:580-590`; `session_manager/manager.go:1961-2025,2212-2365,4702-4800` | Manager restore terminated session và có mode `native`, `saved_prompt`, `fresh`; runtime effect có thể xảy ra trước lỗi. | `saved_prompt` có thể replay prompt; GET generation mới không xác nhận causal caller/mode. |
| `backend/internal/session_manager/session_input.go:100-129`; `adapters/agent/agy/agy.go:GetRestoreCommand` | AO process có agent-operation lock; Agy có đường native `--conversation`. | Lock không bền qua Supervisor crash; Agy/native không chứng minh mọi lần restore là native. |

## 5. Why Existing Capability Is Insufficient

AO quản lý đúng process/worktree nhưng không ghi Supervisor Pair reservation, TaskAttempt lineage, one-shot operator authorization hoặc audit hash chain. AO public API không cung cấp restore preflight mode hay causal token để xác nhận effect sau lost response. Local CAS có thể ngăn Supervisor duplicate call nhưng không làm upstream effect thành idempotent; outcome mơ hồ cần human. Thử nghiệm tìm safe automatic subset chỉ phục vụ quyết định **sau này**, không phải điều kiện release operator-gated restore.

## 6. Expected Benefits vs. Implementation Cost và alternatives

| Phương án | Lợi ích/chi phí và giới hạn |
|---|---|
| **Khuyến nghị: operator-gated restore operation + Pair-wide guards** | Giữ D3/REC-002, dùng pinned AO route; cần migration mới, guard trên cả Store API cũ, 3C linked cleanup và 3D scanner. Mất response vẫn cần human; không khai mở automatic. |
| Hoãn toàn bộ restore tích hợp | Ít thay đổi ngắn hạn nhưng không đáp ứng D3/REC-002, thiếu audit/exclusion, kéo dài 3B; nếu chọn phải đổi roadmap/contract bằng quyết định riêng. |
| Dựa vào AO same-session ID/lock hoặc GET để tự retry | Ít code nhưng không có durable ownership, có thể replay prompt và đụng send/stop; bác bỏ. |

## 7. Architectural Impact và sequencing

**Khuyến nghị ADR-016 addendum**, không phải clarification thuần túy: cần schema/index/token mới, recovery CAS, dispatch resolution và pre-send semantics. Addendum draft chưa có authority. Sau approval: reconcile `docs/04_ARCHITECTURE.md`, `05_DOMAIN_MODEL.md`, `06_WORKFLOW_STATE_MACHINE.md`, `12_UPSTREAM_INTEGRATION.md`, `14_FAILURE_RECOVERY.md`, `21_TRACEABILITY_MATRIX.md`, `22_MODULE_PROVENANCE.md`, `docs/phases/P03_AO_INTEGRATION.md`; sửa scope/JSON/AC của `docs/tasks/DRAFT_TASK_CONTRACT_P03_003B.md`; full schema/`ValidateRaw` với metadata `go-test-p03-003b` đã duyệt; external release rồi mới code. 3B cần scope mới cho migration/domain/Store guard; 3C sở hữu stop/clearance, 3D scanner. Không sửa pinned AO wire, accepted ADR hoặc canonical specs trong lượt này.

## 8. Decision & Rationale — dành cho External Supervisor

- **Direction verdict**: Durable restore + Pair guard, operator-only pinned restore, no ambiguous retry, `DELIVERY_OUTCOME_UNKNOWN` semantics: **APPROVED_DIRECTION**.
- **Detailed design verdict**: `PENDING_EXTERNAL_AUDIT`; DDL, authorization boundary, CAS/cleanup/clearance và pre-send recovery chưa được phê duyệt.
- **Gate**: `TASK_P03_003B=NOT_RELEASED`; `P03_CODE=HELD_PENDING_TASK_P03_003B_CONTRACT_RELEASE`; `ACTIVE_GATE=TASK_P03_003B_CONTRACT_PLANNING`. `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL=OPEN`.
