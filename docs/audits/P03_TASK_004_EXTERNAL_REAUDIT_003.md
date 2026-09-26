# TASK-P03-004 — External Re-Audit 003

**Audited implementation:** `480d18c79a130bf7f0b2ca07cfc47ff656c2aadd` trên `codex/p03-004`; code base `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`.

**Remediated implementation:** `6d91d5bf9dc62a265ae58f00a4dfb7d85c49d8c0` trên `codex/p03-004`.

**External Supervisor verdict:** `REVISION_REQUIRED`; finding `P03-004-R4-001` = `OPEN` (mức P2) chờ re-audit; findings `P03-004-R3-001` và `P03-004-R3-002` = `CLOSED`.

| Finding | Mô tả & Hiện trạng kiểm tra | Trạng thái |
| --- | --- | --- |
| P03-004-R3-001 | Lifecycle Poller: `Stop()` phải luôn join vòng `Start` tương ứng (`<-done`), kể cả khi `PollOnce` đã ghi lỗi và `running=false`. Không cho `Start` vòng mới xóa hoặc che `lastErr` trước khi watcher của vòng cũ nhận kết quả; bảo đảm `Done()` và `Err()` gắn đúng vòng chạy qua `ErrFor(done)` và cấu trúc cycle-bound. Thêm regression có barrier xác định cho lỗi nền, `Stop` đồng thời và restart; tránh phụ thuộc sleep. | CLOSED (theo quyết định External Supervisor; đã giải quyết tại `f9620c2`/`480d18c`) |
| P03-004-R3-002 | Bổ sung probe tích hợp trên daemon binary gây lỗi `PollOnce` thật sau khi `/readyz` đã 200. Xác nhận `/readyz` chuyển 503, host admission unavailable, daemon không crash, owner lock vẫn giữ; shutdown join watcher và Poller. Test seam `-test-fail-poller-file` giữ trong `allowed_scope` (`cmd/supervisor` và `recovery.Poller.TestFailHook`), không biến mock AO thành tuyên bố Live AO. Sửa sai lệch exit code: lệnh stop CLI trả exit 0 (`STOP_ACKNOWLEDGED`), shutdown drain hoàn tất các bước teardown và daemon process exit sạch 0. | CLOSED (theo quyết định External Supervisor; đã giải quyết tại `480d18c`) |
| P03-004-R4-001 | Thứ tự teardown trong `ExecuteShutdownDrain`: cleanup metadata `.owner.json` trước `Store.Close()` trái ADR-017 §4.2 và AC-004-04. Quy định bất biến: `Store.Close()` phải được gọi trước khi dọn `.owner.json` để bảo đảm WAL flush và DB connection đóng an toàn khi lock còn bảo vệ; `.owner.json` chỉ được dọn nếu `owner_instance_id` khớp; pinned DB handle đóng tiếp theo; exclusive `.owner.lock` đóng sau cùng. Bổ sung regression unit test có barrier xác định tại `Store.Close()`. | OPEN (mức P2; đã remediate tại `6d91d5b`, regression barrier PASS) |

**Truy vết Lịch sử Audit (Audit History Traceability):**
- Ghi nhận `P03_TASK_004_EXTERNAL_REAUDIT_002.md`: bản ban hành ban đầu tại commit `c34cb8b` (sau implementation `f9620c2`), sau đó được đính chính thông tin probe và exit code tại commit `6ad2d30` (sau implementation `480d18c`). Giữ nguyên tham chiếu cả hai commit `c34cb8b` và `6ad2d30` để bảo đảm tính truy vết lịch sử kiểm toán.

**Quy định Thứ tự Shutdown Drain & Đính chính Mô tả (ADR-017 §4.2, AC-004-04):**
1. Đóng listeners (`PipeServer`, `HTTPServer`) ngăn chặn request mới.
2. Chờ/drain in-flight operations với `drainTimeout`:
   - *Đính chính mô tả timeout (R1-002)*: Khi xảy ra drain timeout (caller giữ permit quá thời hạn), daemon **KHÔNG thoát ngay tại thời điểm timeout**. Thay vào đó, daemon tiếp tục đóng admission, giữ nguyên quyền sở hữu file lock độc quyền, chờ toàn bộ active callers/permits giải phóng xong (`WaitAllReleased`), hoàn tất tuần tự các bước teardown thông thường (schedulers, `Store.Close`, metadata cleanup, `PinnedDB.Close`, nhả file lock), rồi mới trả lỗi timeout khiến daemon process kết thúc với exit code 2.
3. Dừng background schedulers (poller, TimeoutMonitor) và join watchers.
4. `Store.Close()`: đóng SQLite connection và flush WAL trước khi nhả lock hay dọn metadata (ADR-017 §4.2).
5. Dọn dẹp metadata `.owner.json` IF AND ONLY IF `owner_instance_id` khớp instance của daemon này (`CleanMetadata` kiểm tra `meta.OwnerInstanceID == l.InstanceID`).
6. Đóng pinned DB OS handle.
7. Đóng exclusive `.owner.lock` handle sau cùng (nhượng quyền cho contender tiếp theo).

**Đính chính Probe Assertion & Bằng chứng Barrier (Probe Wording & Regression Evidence):**
- *Đính chính probe contender assertion*: Trong `TestDaemonPollerFailureMidRunProbe`, contender khởi động sau khi daemon thoát được kiểm tra chiếm lock và đạt trạng thái sẵn sàng thông qua `ready-signal-file` (không tự nhận định sai là query HTTP `/readyz` 200).
- *Phân biệt metadata cleanup*: Việc file `.owner.json` được xóa chỉ chứng minh bước `CleanMetadata()` đã chạy thành công; không dùng riêng lẻ assertion này để chứng minh `Store.Close()` đã hoàn tất.
- *Bằng chứng barrier xác định (`TestShutdownDrainStoreCloseBarrierAndOrder`)*:
  * Khi `Store.Close()` bị chặn tại barrier: metadata `.owner.json` vẫn tồn tại trên đĩa; contender cố acquire `.owner.lock` bị từ chối với `ErrSharingViolation` (mã lỗi 32).
  * Sau khi unblock `Store.Close()`: `Store.Close()` hoàn tất; metadata `.owner.json` được dọn sạch; pinned DB handle đóng; `.owner.lock` được nhả sau cùng cho phép contender mới chiếm lock thành công.
- *Bằng chứng kiểm tra instance mismatch (`TestCleanMetadataOnlyRemovesMatchingInstanceID`)*:
  * Khi metadata trên đĩa chứa instance ID lạ, `CleanMetadata()` không xóa file. Khi khớp instance ID, `CleanMetadata()` xóa file sạch sẽ.

**Verification & Evidence Summary at Remediated Commit `6d91d5b`:**
1. `VR-P03-004-HOST-TESTS`: `go test -v -race -count=1 ./internal/host/...` -> EXIT 0 (PASS, 2.126s, gồm `TestShutdownDrainStoreCloseBarrierAndOrder`, `TestCleanMetadataOnlyRemovesMatchingInstanceID`, `TestDrainTimeoutPreservesLockAndStore`)
2. `VR-P03-004-CMD-TESTS`: `go test -v -race -count=1 ./cmd/supervisor/...` -> EXIT 0 (PASS, [no test files])
3. `VR-P03-004-INTEGRATION-TESTS`: `go test -v -race -count=1 ./test/integration/...` -> EXIT 0 (PASS, 24.683s, gồm `TestRealDaemonDrainTimeoutPreservesLock` [ASSERT exit code 2], `TestDaemonReadinessLifecycle` [ASSERT exit 0], `TestDaemonPollerFailureMidRunProbe` [ASSERT exit 0, verify metadata cleanup, verify contender ready-signal file])
4. `VR-P03-004-RECOVERY-POLLER-TESTS`: `go test -v -race -count=1 ./internal/recovery/...` -> EXIT 0 (PASS, 19.845s)
5. `VR-P03-004-AO-CLIENT-TESTS`: `go test -v -race -count=1 ./internal/ao/...` -> EXIT 0 (PASS, 1.780s)
6. Repository Full Race Suite: `go test -race -count=1 ./...` -> EXIT 0 (ALL PASS, 11 packages)
7. Diff Check: `git diff 35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea..HEAD --check` -> EXIT 0 (clean, no warnings)
8. Whitelist Check: 15 files changed, 100% within `allowed_scope` of Revision 2, 0 files in `forbidden_scope`.

**Gate hiện hành:**
- `TASK_P03_004 = RELEASED (Revision 2)`
- `CONTRACT_TASK_P03_004_02 = RELEASED`
- `TASK_P03_004_IMPLEMENTATION = REVISION_REQUIRED`
- `TASK_P03_004_REMEDIATION = IN_PROGRESS`
- `ACTIVE_GATE = TASK_P03_004_REMEDIATION`
- `FINDING_P03_004_R3_001 = CLOSED`
- `FINDING_P03_004_R3_002 = CLOSED`
- `FINDING_P03_004_R4_001 = OPEN`
- `AUTOMATIC_RESTORE = DISABLED`
- Host principal, Live AO và Stage B runtime vẫn `UNVERIFIED`.
- Chưa merge vào `main`, không mở Phase P04/P05.
