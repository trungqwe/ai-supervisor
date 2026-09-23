# P03 TASK-P03-003C Contract Release Audit

- **External Supervisor decision:** `CONTRACT-TASK-P03-003C-01 = APPROVED_FOR_RELEASE`; design findings `3C-C1-001..003 = CLOSED`.
- **Candidate SHA:** `a8f8edc7a64f3d1aa66acf5519397fe46d8f6177`.
- **Candidate blob:** `7a689f16f04b35acecb7952dc43873ece6f7df6f`.
- **Code base SHA:** `246f74eabd09bbc2ea66624ee217035167824a86`.
- **Release artifact SHA:** commit chứa audit này và `docs/tasks/TASK_CONTRACT_P03_003C.md`, ghi trong báo cáo bàn giao để tránh SHA tự tham chiếu.

JSON TaskContract của artifact phải đồng nhất byte với JSON candidate. Chỉ metadata ngoài JSON được đổi để ghi release. Hai hàng Q/M trong bảng diễn giải được sửa từ “exact terminated đúng deadline” thành “exact terminated trước deadline (`now < confirmation_deadline_at`)”, nhất quán với D11 strict Before; candidate lịch sử giữ nguyên.

`go-test-p03-003c` policy metadata được Supervisor phê duyệt: `CwdPolicy=worktree_root`, `MaxTimeoutSeconds=300`, object `additionalProperties=false`, required `package`/`flags`, package enum đúng bốn giá trị trong contract và flags đúng `[-v,-race]`. Full schema và `TaskContractValidator.ValidateRaw` đã PASS trong evidence candidate; timeout 301, `-exec`, package ngoài enum và cwd `../escape` bị từ chối. Đây là validation library/catalog, không chứng minh host runner P04 hoặc host authentication.

Release chỉ cấp quyền triển khai 3C trong whitelist của contract. `TASK_P03_003C_IMPLEMENTATION=IN_PROGRESS`, chưa external audit implementation. `TASK_P03_003D=NOT_RELEASED`; `AUTOMATIC_RESTORE=DISABLED`; runtime linked stop và administrative clearance fail-closed khi chưa có verified host principal hoặc timeout policy được inject. `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`.
