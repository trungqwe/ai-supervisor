# P03 TASK-P03-003A Contract Release Audit

> **Audited candidate commit**: `56b5ef38ba5a45d56d58dd368a0c47cf45e0886f`
> **Audited candidate blob**: `5364edf076b31b8a42a37fee29193fc36ccdf413`
> **Contract ID**: `CONTRACT-TASK-P03-003A-01`
> **Authority**: External Supervisor
> **Verdict**: `CONTRACT_TASK_P03_003A = EXTERNAL_APPROVED_AND_RELEASED`
> **Date**: `2026-09-23`

## 1. Release decision

External Supervisor phê duyệt nguyên trạng JSON contract tại candidate commit và cho phép release riêng phần 3A. `base_sha` của implementation được giữ nguyên là `1daf0b91efc7f065eb07146d642c0b6f6c259212`. Candidate artifact được giữ lại làm lịch sử.

Parent `TASK-P03-003` chỉ được release phần `TASK-P03-003A`; `TASK-P03-003B`, `TASK-P03-003C` và `TASK-P03-003D` chưa được release. `DESIGN_BLOCKER_3D_STARTUP_WIRING` tiếp tục được bảo lưu.

## 2. Approved verification policy metadata

```yaml
ProfileID: go-test
CwdPolicy: worktree_root
MaxTimeoutSeconds: 300
ParameterSchema:
  type: object
  additionalProperties: false
  required:
    - package
    - flags
  properties:
    package:
      type: string
      enum:
        - ./internal/store/...
        - ./internal/domain/...
    flags:
      type: array
      minItems: 2
      maxItems: 2
      uniqueItems: true
      items:
        type: string
        enum:
          - -v
          - -race
```

Đây là policy kiểm chứng contract 3A. `MaxTimeoutSeconds` là trần của verification request, không phải operational timeout của AO. Metadata này không cung cấp executable, không dựng command và không chứng minh P04 runner hoặc process isolation đã được triển khai.

## 3. Validation evidence

Validation được thực hiện bằng full schema và `TaskContractValidator.ValidateRaw` với đúng catalog tại §2. Không dùng catalog nới lỏng.

| Check | Expected | Result | Exit code |
|---|---|---|---:|
| Released JSON khớp nguyên trạng candidate JSON | Exact match | `PASS` | 0 |
| Full-schema structural validation | Accept | `PASS` | 0 |
| `ValidateRaw` với approved catalog | Accept | `PASS` | 0 |
| `timeout_seconds = 301` | Reject | `PASS — REJECTED` | 0 |
| `flags` chứa `-exec` | Reject | `PASS — REJECTED` | 0 |
| `package = ./internal/contract/...` | Reject | `PASS — REJECTED` | 0 |
| `cwd = ../escape` | Reject | `PASS — REJECTED` | 0 |

Lệnh đã chạy:

```text
gofmt -w internal/contract/p03_003a_release_validation_test.go
go test ./internal/contract -run TestP03003AReleaseValidation -v
```

Exit code tổng: `0`. Harness tạm gọi trực tiếp `CompileSchema`, `ParseAndValidateRaw` và `TaskContractValidator.ValidateRaw`; sau khi thu bằng chứng, harness đã bị xóa và không nằm trong release artifact. Catalog đầu vào chính là YAML policy tại §2, không dùng wildcard package, không cho thêm flag và không nới `cwd` hay timeout.

## 4. Governance state sau release

```yaml
TASK_P03_003A: RELEASED
TASK_P03_003B: NOT_RELEASED
TASK_P03_003C: NOT_RELEASED
TASK_P03_003D: NOT_RELEASED
P03_CODE: AUTHORIZED_3A_ONLY
ACTIVE_GATE: TASK_P03_003A_IMPLEMENTATION
DESIGN_BLOCKER_3D_STARTUP_WIRING: PRESERVED
```

`release_artifact_sha` là SHA thực tế của commit chứa hồ sơ này và được báo cáo ngoài artifact sau khi commit được tạo; tài liệu không nhúng SHA tự tham chiếu.
