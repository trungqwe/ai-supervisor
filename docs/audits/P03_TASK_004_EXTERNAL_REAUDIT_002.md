# TASK-P03-004 — External Re-Audit 002

**Audited implementation:** `a93e9318bf96229576e108d98f835574f6a50a3b` trên `codex/p03-004`; code base `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`.

**Remediated implementation:** `f9620c225c8a00d5bd148c84ccba38152c10c4ec` trên `codex/p03-004`.

**External Supervisor verdict:** `REVISION_REQUIRED`; findings `P03-004-R3-001` và `P03-004-R3-002` = `OPEN` chờ re-audit.

| Finding | Mô tả & Hiện trạng kiểm tra | Trạng thái |
| --- | --- | --- |
| P03-004-R3-001 | Lifecycle Poller: `Stop()` phải luôn join vòng `Start` tương ứng (`<-done`), kể cả khi `PollOnce` đã ghi lỗi và `running=false`. Không cho `Start` vòng mới xóa hoặc che `lastErr` trước khi watcher của vòng cũ nhận kết quả; bảo đảm `Done()` và `Err()` gắn đúng vòng chạy qua `ErrFor(done)` và cấu trúc cycle-bound. Thêm regression có barrier xác định cho lỗi nền, `Stop` đồng thời và restart; tránh phụ thuộc sleep. | OPEN (đã remediate tại `f9620c2`, test barrier PASS) |
| P03-004-R3-002 | Bổ sung probe tích hợp trên daemon binary gây lỗi `PollOnce` thật sau khi `/readyz` đã 200. Xác nhận `/readyz` chuyển 503, host admission unavailable, daemon không crash, owner lock vẫn giữ; shutdown join watcher và Poller. Test seam `-test-fail-poller-file` giữ trong `allowed_scope` (`cmd/supervisor` và `recovery.Poller.TestFailHook`), không biến mock AO thành tuyên bố Live AO. | OPEN (đã remediate tại `f9620c2`, integration probe PASS) |

**Verification & Evidence Summary at Remediated Commit `f9620c2`:**
1. `VR-P03-004-HOST-TESTS`: `go test -v -race -count=1 ./internal/host/...` -> EXIT 0 (PASS, 2.095s)
2. `VR-P03-004-CMD-TESTS`: `go test -v -race -count=1 ./cmd/supervisor/...` -> EXIT 0 (PASS, [no test files])
3. `VR-P03-004-INTEGRATION-TESTS`: `go test -v -race -count=1 ./test/integration/...` -> EXIT 0 (PASS, 24.540s, bao gồm `TestRealDaemonDrainTimeoutPreservesLock`, `TestCLIStopNegativeCases`, `TestDaemonReadinessLifecycle`, `TestDaemonPollerFailureMidRunProbe`)
4. `VR-P03-004-RECOVERY-POLLER-TESTS`: `go test -v -race -count=1 ./internal/recovery/...` -> EXIT 0 (PASS, 13.774s, bao gồm `TestPollerDoneAndErrLifecycle_StopAlwaysJoinsGoroutine`, `TestPollerDoneAndErrLifecycle_BarrierErrorAndRestart`)
5. `VR-P03-004-AO-CLIENT-TESTS`: `go test -v -race -count=1 ./internal/ao/...` -> EXIT 0 (PASS, 1.806s)
6. Repository Full Race Suite: `go test -race -count=1 ./...` -> EXIT 0 (ALL PASS)
7. Diff Check: `git diff 35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea..HEAD --check` -> EXIT 0 (clean, no warnings)
8. Whitelist Check: 15 files changed, 100% within `allowed_scope` of Revision 2, 0 files in `forbidden_scope`.

**Gate hiện hành:**
- `TASK_P03_004 = RELEASED (Revision 2)`
- `CONTRACT_TASK_P03_004_02 = RELEASED`
- `TASK_P03_004_IMPLEMENTATION = REVISION_REQUIRED`
- `TASK_P03_004_REMEDIATION = IN_PROGRESS`
- `ACTIVE_GATE = TASK_P03_004_REMEDIATION`
- `FINDING_P03_004_R3_001 = OPEN`
- `FINDING_P03_004_R3_002 = OPEN`
- `AUTOMATIC_RESTORE = DISABLED`
- Host principal, Live AO và Stage B runtime vẫn `UNVERIFIED`.
- Chưa merge vào `main`, không mở Phase P04/P05.
