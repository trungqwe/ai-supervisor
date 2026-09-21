# P02 IMPLEMENTATION DECISION EXTERNAL AUDIT DOSSIER

> **Audit Record**: `P02_IMPLEMENTATION_DECISION_EXTERNAL_AUDIT.md`
> **Date**: 2026-09-22
> **Auditor**: External Supervisor / Engineering Governance
> **Status**: APPROVED
> **Verdict**: `P02_IMPLEMENTATION_DECISIONS = APPROVED`
> **Code Readiness**: `P02_CODE = READY_PENDING_EXTERNAL_COMMIT_AUDIT`

---

## 1. Audit Scope & Baseline Verification
This audit evaluates the final pre-code technology decisions, contract migrations, phase boundaries, and security models governing Phase P02 (Supervisor Domain Core):
- **Remote Baseline Commit**: `9c422bd94305e426bf7e323fc4e45bc71f2f9fb6`
- **Peeled Commit**: `62d3fe0df4a3a05697da77349ff085430ea452f7`
- **Architecture Tag**: `phase1-architecture-v2.1` (Remote tag object: `cf8dfeed057271b6e76df443204f61254a703461`).
- **Production Code Status**: `NOT_STARTED` (Verified: zero production code, zero frameworks, zero external dependencies).

---

## 2. ADR-012 Confirmation
- **Status**: `ACCEPTED`
- **Confirmed Invariants**:
  - TaskContract immutability post-dispatch.
  - Revision lineage with monotonic revision numbering and `supersedes_contract_id` tracking.
  - Baseline `base_sha` immutability established in revision 1 and preserved across all revisions of a `task_id`.
  - Atomic pre-dispatch allocation: `TaskAttempt` allocation and `READY → DISPATCHED` commit atomically precede external AO calls.
  - Revision authority: `request_revision` strictly records review decisions and transitions to `REVISION_REQUIRED`; new contracts are supplied exclusively by planning authority.

---

## 3. ADR-013: Trusted Verification Command Specification & Security Architecture
- **Status**: `ACCEPTED`

### 3.1 Command Authority Threat & Layer A Containment
- Legacy `required_tests: string[]` and simple executable allowlists (`node`, `python`, `npm`, `git`) are rejected because allowlisted binaries contain built-in flags (`-e`, `-c`, `exec`) that execute arbitrary code.
- **Layer A**: Host `ProjectPolicy` defines immutable verification profiles, trusted absolute executable paths resolved at daemon startup (never via mutable `PATH`), fixed argument prefixes, parameter schemas, and hard timeout/output caps. TaskContract provides only `verification_requests` referencing approved profiles with typed parameters. No shells, no arbitrary args.

### 3.2 Untrusted Repository Code Threat & Layer B Isolation
- Architectural Axiom: `SAFE_PROFILE != TRUSTED_CODE`.
- Verification tools (`go test`, `pytest`, `npm test`) execute repository-controlled code written by workers.
- **Layer B**: The future VerificationRunner (P04) must execute verification commands inside an execution isolation boundary:
  - Zero Supervisor secrets, API tokens, AO credentials, or tunnel keys.
  - Filesystem containment restricted to the workspace worktree; attempt-local `TEMP`/`TMP`.
  - Inbound and outbound network denied by default.
  - Process lifecycle and resource limits managed on Windows via **Windows Job Objects** (`golang.org/x/sys/windows`).
  - *Distinction*: Job Objects provide process containment, not filesystem or network sandboxing (detailed isolation implementation is a P04 deliverable).

### 3.3 Environment Policy
- Child verification environments are aggressively stripped. Broad inheritance of `USERPROFILE`, `HOME`, `NODE_PATH`, `GOPATH`, `TEMP`, `TMP`, and credentials is prohibited. Only minimum OS variables, host-owned toolchain paths, and profile-specific allowlisted variables are provided.

### 3.4 TaskContract Schema Migration
- Canonical `task-contract.schema.json` and valid fixtures migrated from `required_tests` to `verification_requests`.
- Generic schema validates outer shape (`id`, `profile_id`, `parameters`, `cwd`, `timeout_seconds`), while semantic parameter validation is performed in P02 by `TaskContractValidator`.

---

## 4. ADR-014: Supervisor Core Implementation Language
- **Status**: `ACCEPTED`
- **Canonical Decision**: `SUPERVISOR_CORE_LANGUAGE = GO`

### 4.1 Go Version Policy
- **Development Toolchain Baseline**: **`Go 1.27.x`** (Current supported major as of 2026-09-22).
- **Minimum Supported Line**: **`Go 1.26.x`** (Previous supported major per Go's N-1 release policy).
- All stale references (`Go 1.22+`, `Go 1.23+`, `Go 1.24+`) are retired.

### 4.2 Official MCP Go SDK & Streamable HTTP
- Verified primary source: `github.com/modelcontextprotocol/go-sdk` is an official Tier 1 SDK co-maintained with Google.
- Implements MCP 2026-07-28 protocol specification.
- Modern HTTP transport standard is **Streamable HTTP** (standalone SSE labeled legacy).
- Note: Track P01-D3C proved loopback HTTP tunneling to ChatGPT; end-to-end Go MCP server verification will be executed in P05.

### 4.3 Windows Process Control & Job Objects
- `os/exec` and `syscall.SysProcAttr` provide process creation controls.
- Process-tree lifecycle containment is integrated via Win32 Job Object APIs in `golang.org/x/sys/windows` (`CreateJobObject`, `AssignProcessToJobObject`, `TerminateJobObject`).

---

## 5. ADR-015: Local State Store Engine & Durability Policy
- **Status**: `ACCEPTED`
- **Canonical Decision**: `STATE_STORE_ENGINE = SQLITE`

### 5.1 Durability Configuration & Contract
- Pragmas: `PRAGMA journal_mode = WAL;`, `PRAGMA synchronous = FULL;`, `PRAGMA foreign_keys = ON;`.
- Under SQLite's documented VFS/filesystem synchronization contract, WAL + `synchronous = FULL` ensures committed transactions are flushed to physical media on commit, guaranteeing the pre-dispatch invariant across unexpected restarts.

### 5.2 Concurrency & Busy Timeout Policy
- WAL enables concurrent readers alongside a serialized writer.
- Contention is managed via a **configurable bounded busy timeout policy** (defaulting to 5000 ms).
- **Transaction Rule**: Write transactions must be strictly short-lived. No AO REST calls, process executions, network calls, large file parsing, or long computations inside an open transaction.

### 5.3 Active Backup Invariant
- Naive copying of `supervisor.db` while WAL is active is prohibited.
- Approved active backup mechanisms: SQLite Online Backup API (`sqlite3_backup`) or `VACUUM INTO 'backup.db'`.

---

## 6. P02 Phase Ownership & Traceability Reconciliation
- **Status**: `APPROVED_WITH_SCOPE_CORRECTIONS`
- **Approved P02 Scope**:
  - Headless domain entities & value types (`Project`, `Pair`, `Task`, `TaskContract`, `TaskAttempt`, `WorkerClaim`, `Evidence` value types, `ReviewDecision`).
  - Domain services (`StateMachine` 13 states / 25 edges, `TaskContractValidator`, pre-dispatch scope checks, atomic dispatch intent).
  - StateStore SQLite schema, migrations, relational lineage, restart recovery classifier (`EXTERNAL_RECONCILIATION_REQUIRED`), and close API.
  - Audit core: `AuditEvent` model, append-only persistence, secret sanitization regex.
- **Excluded Subsystems**:
  - P03: `AOAdapter`, session lifecycle, AO-backed restart reconciliation.
  - P04: `EvidenceCollector`, Git diff analysis, `VerificationRunner` (Layer B execution isolation), `ReviewBundleBuilder`.
  - P05: Fastify/MCP server, Streamable HTTP, `ToolSurface`, interactive project/pair tools, `ContextEngine`, OS signal handling.
  - P06: Operator UI, packaging, export utilities.
- Traceability matrix reconciled across all 14 affected requirements in `docs/21_TRACEABILITY_MATRIX.md`.

---

## 7. Remaining Implementation Risks & Mitigation
1. **Windows Child Process Isolation Complexity**: Addressed by scheduling detailed isolation research (Job Objects, security tokens, restricted environments) in Phase P04.
2. **Go SQLite CGo Overhead**: Mitigated by selecting pure Go `modernc.org/sqlite`, compiling cleanly without CGo.
3. **Database Write Serialization**: Mitigated by short-lived transaction invariant and configurable busy timeout.

---

## 8. Final Audit Verdict

```
P01_RUNTIME_PROOFS = COMPLETE
ADR_012 = ACCEPTED
ADR_013 = ACCEPTED
ADR_014 = ACCEPTED
ADR_015 = ACCEPTED
Q3_SUPERVISOR_CORE_LANGUAGE = GO
GO_TOOLCHAIN_BASELINE = 1.27.x
GO_MIN_SUPPORTED_LINE = 1.26.x
Q4_LOCAL_STATE_STORE_ENGINE = SQLITE
P02_PHASE_OWNERSHIP = APPROVED
P02_IMPLEMENTATION_DECISIONS = APPROVED
P02_CODE = READY_PENDING_EXTERNAL_COMMIT_AUDIT
```
