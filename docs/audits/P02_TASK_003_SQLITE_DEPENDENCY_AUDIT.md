# TASK-P02-003 SQLITE DEPENDENCY AUDIT DOSSIER

> **Audit Record**: `P02_TASK_003_SQLITE_DEPENDENCY_AUDIT.md`  
> **Date**: 2026-09-22  
> **Auditor**: External Supervisor / Engineering Governance  
> **Subject**: Embedded SQLite Engine for StateStore Persistence  
> **Verdict**: `MODERNC_SQLITE_V1_59_0 = APPROVED_FOR_TASK_P02_003`  

---

## 1. Module Identification & Upstream Facts

| Property | Value | Notes |
|---|---|---|
| **Driver Module** | `modernc.org/sqlite` | Pure Go SQLite engine |
| **Driver Target Version** | `v1.59.0` | Published September 15, 2026 |
| **License** | BSD-3-Clause | Permissive open source license |
| **Driver Go Requirement** | `go 1.25.0` | Per `go.mod` in module cache |
| **Project Go Requirement** | Go 1.26.x minimum / 1.27.x baseline | Fully compatible (`go 1.25.0` <= `go 1.26.x`) |
| **Windows amd64 Support** | YES | Direct native support for `windows/amd64` |
| **CGO Requirement** | `NONE` | Pure Go transpiled implementation, zero CGO toolchain required |
| **Canonical Upstream Repository** | `https://gitlab.com/cznic/sqlite` | Commit `c96a4e6cb22254bf70026502a781a54a053c2cf0` |
| **modernc.org/libc Alignment Requirement** | `v1.75.7` | Pinned exactly in `modernc.org/sqlite v1.59.0` |
| **Expected libc Version** | `v1.75.7` | Must match without skew |

---

## 2. Upstream Warning: libc Version Skew Protection

Upstream maintainers explicitly document that `modernc.org/sqlite` is tightly coupled to `modernc.org/libc`. Version skew or mismatched transitive dependencies can cause fatal runtime panics or undefined ABI behaviors.

- The authoritative `go.mod` of `modernc.org/sqlite v1.59.0` pins `modernc.org/libc v1.75.7`.
- During installation, module resolution MUST verify that `modernc.org/libc` resolves strictly to `v1.75.7`.
- Any automatic upgrade or downgrade of `modernc.org/libc` is considered an installation failure that requires immediate halt.

---

## 3. Dependency Footprint & Security Posture

1. **Storage Engine Role**:
   - Implements local SQLite persistence for `Project`, `Pair`, `Task`, `TaskContract`, and `TaskAttempt` (ADR-015).
   - Operates entirely locally with PRAGMAs: `journal_mode = WAL`, `synchronous = FULL`, `foreign_keys = ON`, bounded `busy_timeout`.
2. **Pure Go / Zero CGO**:
   - Allows deterministic Windows builds, cross-compilation, and testing without requiring MinGW/MSVC toolchains.
3. **Security Claim Precision**:
   - No claim of "zero vulnerabilities" is made without continuous verification; safety is maintained through version pinning, BSD-3-Clause compliance, and offline operation with no network drivers.

---

## 4. Decision

```
MODERNC_SQLITE_V1_59_0 = APPROVED_FOR_TASK_P02_003
DRIVER_VERSION = v1.59.0
REQUIRED_LIBC_VERSION = v1.75.7
CGO_REQUIRED = NONE
SUPPORTED_PLATFORM = windows/amd64
```
