# PHASE SPECIFICATION: P04 — EVIDENCE & REVIEW ENGINE

> **Governance Authority**: ADR-018 Accepted Architecture Baseline (`ADR_018 = EXTERNAL_APPROVED`, `ADR_018_ACCEPTANCE = GRANTED`).
> **Status**: Approved Baseline for Implementation Sequencing.

## 1. Objective
Implement independent canonical workspace binding, isolated verification execution, zero-mutation Git evidence intake, and Content-Addressed ReviewBundle compilation pursuant to ADR-018.

## 2. Deliverables & Subtask Breakdown (CR-09, CR-11)
Phase P04 is executed through four sequential task contracts enforcing Model 1 persistence ownership:

1. **Subtask P04A (`CONTRACT-TASK-P04-001`) — Workspace Binding & Claims Schema v6**:
   - **Deliverables**: Canonical workspace binding creation, Win32 anti-rename file handle management, SQLite Schema v6 migration (`attempt_workspace_bindings`, `worker_claims`, `review_integrity_holds`), Transaction A report ingestion & `worker_claims` persistence, dirty rollback on unclean inspection result, and intake diagnostic transaction (`EVIDENCE_COLLECTION_FAILED`, `DIRTY_WORKTREE_DETECTED` hold).
   - **Sequencing**: First releaseable implementation work package unblocked upon canonical reconciliation approval.
2. **Subtask P04B (`CONTRACT-TASK-P04-002`) — In-Memory Intake & Git Collector**:
   - **Deliverables**: Zero-mutation workspace inspection (`git status --porcelain=v1 -z --untracked-files=all`, `git diff-index --quiet HEAD --`), clean intake validation, diff collection, dual head SHA extraction (`actual_head_sha` vs `reported_head_sha`), and structured in-memory handoff (`GitEvidenceResult`) to Subtask P04A.
   - **Persistence Boundary**: Pure in-memory (zero migrations, zero SQLite writes, zero audit appends, zero hold mutations; does NOT own Transaction A or diagnostic transaction).
3. **Subtask P04C (`CONTRACT-TASK-P04-003`) — In-Memory Windows AppContainer Sandbox Runner**:
   - **Deliverables**: Isolated verification runner executing host-profile verification requests in Windows AppContainer sandbox via Win32 `CreateProcessW` with `STARTUPINFOEXW`, explicit stdio-only `PROC_THREAD_ATTRIBUTE_HANDLE_LIST`, atomic Job Object assignment (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`), network restriction SID, 10MB/50MB stream limits, and authoritative process-death proof.
   - **Persistence Boundary**: Pure in-memory (zero migrations, zero SQLite writes, zero audit appends, zero hold mutations).
4. **Subtask P04D (`CONTRACT-TASK-P04-004`) — Verification Lease & ReviewBundle CAS Schema v9**:
   - **Deliverables**: Linear verification lease chain, Content-Addressed Store write-through to `artifacts/<first-two-hex>/<captured_sha256>`, ReviewBundle compilation with RFC 8785 JCS and assembly latency semantics (`compilation_latency_ms`, 2x2 measurement provenance matrix, `nfr008_compliance_status = 'UNVERIFIED'`), SQLite Schema v9 migration (`task_verification_leases`, `evidence_sets`, `review_artifacts`, `review_bundles`), inert fake AO adapter test harness, and startup recovery scanner.

## 3. Inert AO Test Harness & Live AO Decoupling (CR-07)
- **Synthetic Test Harness**: Pure in-process synthetic session harness returning deterministic JSON fixtures for automated integration suites without spawning live external processes.
- **Decoupled Live AO Track**: Live AO integration remains held in an unverified evidence track (`LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`). Live daemon dependencies do not block Phase P04 evidence verification or bundle compilation.
- **Fail-Closed Invariants**: `AUTOMATIC_RESTORE = DISABLED` permanently maintained; `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`.

## 4. Exit Gate Criteria (CR-11)
- Verified Windows AppContainer sandbox execution with authoritative process-death proof.
- Linear lease recovery and monotonic token validation across process restarts.
- ReviewBundle CAS write-through compilation matching JSON Schema Draft-07.
- Complete repository race test suite passing clean (`go test -race -count=1 ./cmd/... ./internal/... ./test/...`).
- Formal External Supervisor review and audit approval (`P04_EXIT_GATE = EXTERNAL_AUDIT_APPROVED`).
