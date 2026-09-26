# TASK-P03-004 — External Re-Audit 003 Erratum 001

**Parent Audit Record:** [`docs/audits/P03_TASK_004_EXTERNAL_REAUDIT_003.md`](P03_TASK_004_EXTERNAL_REAUDIT_003.md) (commit `25fbd299d66ba32175f5d09ae6a8e10d50b9c803`).

**Audit Authority:** External Supervisor.

**Purpose:** Append-only audit erratum correcting implementation citation commit hash and recording final formatted implementation commit hash following hygiene check.

---

### 1. Citation Hash Correction (Đính chính SHA trích dẫn)
- Trong văn bản kiểm toán `P03_TASK_004_EXTERNAL_REAUDIT_003.md`, commit hash của bản remediated implementation được trích dẫn nhầm thành:
  `6d91d5bf9dc62a265ae58f00a4dfb7d85c49d8c0`
  Giá trị này là một citation hash không chính xác (do lỗi sao chép) và **không resolve** thành bất kỳ Git commit nào trong repository.
- **Commit SHA chính xác của bản remediation hành vi (Store.Close trước CleanMetadata):**
  `6d91d5b7e9c105f9d2e3689145b880183b486c8e` trên nhánh `codex/p03-004`.
  (Commit này đã giải quyết toàn bộ yêu cầu về thứ tự teardown ADR-017 §4.2 / AC-004-04 và bổ sung barrier regression test).

### 2. Hygiene & Final Formatted Implementation (Chuẩn hóa gofmt)
- Exact audited tree tại `6d91d5b7e9c105f9d2e3689145b880183b486c8e` được phát hiện còn sai lệch định dạng `gofmt` (thứ tự import trong `internal/host/host_test.go` và khoảng trắng trong các file Go thuộc diff).
- Đã thực thi `gofmt` chuẩn hóa định dạng trên toàn bộ các file Go thuộc diff; xác nhận kiểm tra `gofmt -d` hoàn toàn rỗng trên toàn bộ 15 file Go. Không có bất kỳ thay đổi hành vi logic nào.
- **Commit SHA implementation hoàn thiện cuối cùng sau gofmt (Final Implementation SHA):**
  `d9b7c1344a8a41cfad6e3bd8bb4db87080afcc03` trên nhánh `codex/p03-004`.

### 3. Verification & Evidence at Final SHA `d9b7c1344a8a41cfad6e3bd8bb4db87080afcc03`
1. `gofmt -d`: Hoàn toàn rỗng trên toàn bộ 15 file Go thuộc diff.
2. `VR-P03-004-HOST-TESTS`: `go test -v -race -count=1 ./internal/host/...` -> EXIT 0 (PASS, 2.107s)
3. `VR-P03-004-INTEGRATION-TESTS`: `go test -v -race -count=1 ./test/integration/...` -> EXIT 0 (PASS, 24.965s)
4. Repository Full Race Suite: `go test -race -count=1 ./...` -> EXIT 0 (ALL PASS, 11 packages)
5. Diff Check: `git diff 35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea..HEAD --check` -> EXIT 0 (clean, no warnings)
6. Whitelist Check: 15 files changed, 100% within `allowed_scope` of Revision 2, 0 files in `forbidden_scope`.

### 4. Finding Status & Active Gate
- Finding `P03-004-R3-001` = `CLOSED`.
- Finding `P03-004-R3-002` = `CLOSED`.
- Finding `P03-004-R4-001` = `SUBSTANTIVELY_CLOSED_PENDING_FINAL_EXACT_SHA_AUDIT`.
- `ACTIVE_GATE = TASK_P03_004_REMEDIATION` (Dừng trước merge; không tự tuyên bố `EXTERNAL_AUDIT_APPROVED` hoặc merge vào `main`).
- `AUTOMATIC_RESTORE = DISABLED`.
- Live AO, verified host principal và Stage B runtime vẫn `UNVERIFIED`.
