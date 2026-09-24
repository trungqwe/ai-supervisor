# External Supervisor Contract Release Audit: TASK-P03-004

- **Task ID**: `TASK-P03-004` (Host Quiescence & Daemon Bootstrap Integration)
- **Contract ID**: `CONTRACT-TASK-P03-004-01`
- **Candidate Commit**: `fd2b516e7c801898d67b41083faf185c24ab4e87`
- **Candidate Blob**: `2db164f1ab5336f61085d5b8d24b7903417ba225` (`docs/tasks/CANDIDATE_TASK_CONTRACT_P03_004.md`)
- **Released Contract**: `docs/tasks/TASK_CONTRACT_P03_004.md`
- **Code Base SHA**: `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`
- **Release Verdict**: `RELEASE_APPROVED_WITH_MANDATORY_TEXTUAL_ERRATA`
- **Active Gate**: `TASK_P03_004_IMPLEMENTATION`
- **Code Gate**: `P03_CODE = AUTHORIZED_004_ONLY`

---

## 1. Audit Evaluation & Approval Summary

External Supervisor audited candidate contract `CONTRACT-TASK-P03-004-01` at candidate commit `fd2b516e7c801898d67b41083faf185c24ab4e87` (blob `2db164f1ab5336f61085d5b8d24b7903417ba225`).
- The JSON object conforms strictly to `docs/08_TASK_CONTRACT.md` and schema validation.
- All verification requests use the approved `go-test-p03-004` catalog profile with const flags `["-v", "-race", "-count=1"]`.
- The architecture directions of `ADR-017` (`ACCEPTED`) and `PROPOSAL-P03-005` (`APPROVED`) are established without opening design review.
- All required acceptance criteria (`AC-004-01` through `AC-004-08`) and negative validation cases are verified.

The External Supervisor grants **`RELEASE_APPROVED_WITH_MANDATORY_TEXTUAL_ERRATA`** with the following four errata resolved during release handoff:

1. **Task Contract Release File**:
   - `docs/tasks/TASK_CONTRACT_P03_004.md` is promulgated with proper `RELEASED` header, status, authority, and invariants.
   - All fields and values of the canonical JSON TaskContract are preserved verbatim from candidate blob `2db164f1ab5336f61085d5b8d24b7903417ba225`.
   - `docs/tasks/CANDIDATE_TASK_CONTRACT_P03_004.md` is preserved intact as immutable historical audit evidence.

2. **Traceability Matrix Reference Correction**:
   - Non-existent references to ADR-017 sections in `docs/21_TRACEABILITY_MATRIX.md` are corrected to map precisely to existing canonical sections in `docs/adr/ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md`:
     - Machine-wide Exclusivity -> `§2.3`
     - Alias Proof & Fail-Closed -> `§2.3, §4.3`
     - Host Pinned DB Handle & 4 Invariants -> `§2.3, §4.4`
     - Shutdown Drain & Order -> `§2.3, §2.6`
     - Named Pipe Takeover & SID Check -> `§2.3`
     - Startup Readiness Probe -> `PLAN-HOST-INTEGRATION-DEPENDENCY.md §4; ADR-017 §2.6`
     - P03 Integration Harness -> `PLAN-HOST-INTEGRATION-DEPENDENCY.md §5; ADR-017 §4.1`
     - Zero Effectful HTTP & UNSET Policy -> `ADR-017 §3.1, §4.2, §4.4`

3. **ADR-017 Title & Architecture Heading Structure**:
   - Title of `docs/adr/ADR-017-host-quiescence-and-daemon-lifecycle-architecture.md` stripped of legacy "DRAFT" prefix to reflect official `ACCEPTED` status.
   - Heading structure in `docs/04_ARCHITECTURE.md` reconciled so that all Section 2 execution/review flows (`2.1` through `2.5`) reside inside `# 2. Canonical Execution & Review Flows`, Section 3 retains adapter boundaries (`3.1`, `3.2`), and Section 4 retains authority policies without altering architectural decisions.

4. **Current State & AGENTS.md Synchronization**:
   - Obsolete "Current" lines in `docs/18_CURRENT_STATE.md` and active gate directives in `AGENTS.md` are synchronized to `TASK_P03_004 = RELEASED`, `ACTIVE_GATE = TASK_P03_004_IMPLEMENTATION`, and `P03_CODE = AUTHORIZED_004_ONLY`.

---

## 2. Invariants & Implementation Protocol

1. **Isolation of Release & Implementation**:
   - Release artifact commit is created on `main` separately before implementation begins.
   - Implementation branch is `codex/p03-004` based strictly on `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`.
   - Governance artifacts must NOT appear in the implementation diff.
2. **Three Split Evidence Tracks**:
   - Track 1: Library & Mock AO automated unit/integration tests in CI (`internal/host/...`, `test/integration/...`).
   - Track 2: Real Windows binary runtime verification (`cmd/supervisor`) verifying machine-wide lock competition between two processes (including cross-session), startup-before-serve admission, and graceful shutdown drain.
   - Track 3: Live AO isolated integration (if isolated test environment is available). Mock tests must NEVER be used to claim Live AO proof or authenticated host principal evidence.
3. **Runtime Invariants**:
   - `AUTOMATIC_RESTORE = DISABLED`.
   - Verified host principal remains `OPEN` at the trusted boundary until runtime completion evidence exists.
   - Zero Phase P04/P05 dependencies. Zero arbitrary shell primitives.
