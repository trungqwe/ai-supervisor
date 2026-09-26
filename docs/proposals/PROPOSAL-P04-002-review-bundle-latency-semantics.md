# PROPOSAL-P04-002: ReviewBundle Latency Measurement Semantics (NFR-008 Reconciliation)

> **Proposal ID**: `PROPOSAL-P04-002`
> **Revision**: 4
> **Title**: Formal Latency Measurement Semantics for ReviewBundle Compilation & Pipeline Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Target Requirement**: `docs/02_REQUIREMENTS.md` (NFR-008)
> **Related Architecture**: `docs/04_ARCHITECTURE.md` (Section 7), `docs/10_REVIEW_BUNDLE.md`, `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`
> **Audited Baseline**: `6dbd7e26d59e22171271921b86c01bd6f215e59a`
> **External Audit Tracking**: Remediates Finding `P04-ARCH-R9-002` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_008.md`).

---

## 1. Executive Summary

Canonical non-functional requirement **NFR-008** specifies:
> *"The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

At the same time, functional requirement **FR-008** and canonical **Architecture Section 7** require that the `ReviewBundle` contain **independent verification evidence** (`actual_test_evidence` and `actual_git_evidence`) captured by running Git collectors, test suites, compilers, and linters in isolated Windows AppContainers under the Supervisor Control Plane's authority.

Running real-world test suites (e.g., `go test -race ./...`, `npm test`, `pytest`) legitimately requires durations ranging from tens of seconds to several minutes, inherently exceeding 3 seconds. If NFR-008's 3-second clock begins at worker report submission and includes external test execution, every realistic engineering task will breach NFR-008 regardless of Supervisor Control Plane efficiency. Conversely, omitting test execution from the ReviewBundle violates audit integrity.

Furthermore, External Re-Audit 008 recorded that previously proposed DDL constraints (`compilation_latency_ms <= 3000`) caused an unrecoverable database deadlock: if a daemon crashed between evidence finalization (Transaction B) and bundle persistence (Transaction C), or if compilation experienced transient scheduling delay exceeding 3.0 seconds, Transaction C was permanently aborted by SQLite, trapping the task forever in `EVIDENCE_READY`.

This proposal establishes a rigorous, crash-safe measurement model that reconciles NFR-008 without weakening audit guarantees, guarantees crash recovery, and defines the change governance process required before canonical `docs/02_REQUIREMENTS.md` or ADR-018 can be formally adopted.

---

## 2. Problem Statement & Tension Analysis

### 2.1. The Operational Conflict
1. **Canonical NFR-008**: Stipulates a hard performance budget of <= 3.0 seconds from worker completion on repositories up to 10,000 files.
2. **Canonical FR-008 & ReviewBundle Schema**: Stipulates that `ReviewBundle.actual_test_evidence` must record the Supervisor's independently executed test commands and exit codes.
3. **Execution Reality**:
   - Compiling and executing test suites in Windows AppContainers with explicit handle inheritance and network denial is bounded by project build times, external toolchain performance, and test complexity.
   - For real projects, verification tests typically run between 5 and 60 seconds.

### 2.2. Timestamp Semantics & Crash Recovery Flaw
Previous drafts failed to distinguish between in-process monotonic intervals and durable timestamps across restarts:
1. **Monotonic timers cannot bridge process restarts**: A Go `time.Since()` timer initialized during Transaction B is lost if the daemon process terminates or crashes before Transaction C.
2. **Permanent Deadlock under CHECK <= 3000**: If `compilation_latency_ms <= 3000` is enforced as a SQLite CHECK constraint, any post-restart recovery attempt (where wall-clock elapsed time naturally exceeds 3,000 ms) will fail SQL constraint validation. The task becomes permanently unfinalizable in `EVIDENCE_READY`.
3. **Write Timestamp vs WAL Commit Completion**: The timestamp selected in application memory immediately prior to calling `tx.Commit()` differs slightly from the exact microsecond the SQLite WAL fsync completes.

### 2.3. Governance Conflict
Modifying the text or interpretation of NFR-008 unilaterally violates the 9-level decision hierarchy of `docs/24_CHANGE_GOVERNANCE.md` (Level 4 Requirement Specification cannot be implicitly altered by Level 6 Task Contracts or Level 8 Suggestions). Formal approval of this proposal by the External Supervisor is mandatory before modifying canonical documentation.

---

## 3. Four-Tier Timestamp Taxonomy

To eliminate ambiguity between in-process timers and durable records, this proposal formalizes four distinct timestamp and duration concepts:

1. **Transaction Pre-Commit Timestamp (`write_timestamp_epoch_ms`)**:
   - Chosen in application Go code (`time.Now().UnixMilli()`) immediately before issuing `tx.Commit()`.
   - Used as the deterministic durable column value in SQLite (`evidence_committed_at_epoch_ms`, `bundle_committed_at_epoch_ms`).
2. **Actual WAL Commit Completion**:
   - The exact point in time when SQLite WAL write and optional fsync return control to the application.
   - Used for internal runtime telemetry and operational diagnostics.
3. **Monotonic In-Process Duration (`monotonic_in_process_duration`)**:
   - High-resolution measurement using `time.Now()` (monotonic clock reading) initialized immediately upon Transaction B commit and stopped immediately upon Transaction C commit.
   - **Scope Invariant**: Valid *only* within a single running process lifecycle. Never serialized to persistent storage or assumed to survive daemon restart.
4. **Durable Wall-Clock Diagnostic Interval (`compilation_latency_ms`)**:
   - Calculated deterministically as `bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms`.
   - Guaranteed monotonically non-negative (`CHECK (compilation_latency_ms >= 0)`).
   - Persisted durably to SQLite to record whether the NFR-008 SLA was met, regardless of daemon crash/restart.

---

## 4. Proposed Measurement Model: Two-Interval Pipeline

The verification pipeline is structured into two precisely bounded operational intervals:

### 4.1. Interval 1: Evidence Acquisition Window (Work Execution Phase)
- **Start**: Worker signals report completion; Transaction A validates report against canonical schema, verifies clean worktree/index, stores structured `worker_claims` (Schema v6), and transitions `tasks.state`: `RUNNING -> REPORT_READY`.
- **Operations**:
  1. Supervisor revalidates physical worktree handle identity (`FileIdInfo`) and re-verifies clean index/worktree.
  2. Supervisor creates snapshot `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/` from actual verified HEAD commit via Git `ls-tree -rz --full-tree` and `cat-file --batch`.
  3. Supervisor executes hardened in-memory Git evidence collection (`P04B`).
  4. Supervisor executes verification test commands sequentially (`P04C`) inside isolated Windows AppContainers against the immutable snapshot.
  5. Supervisor stages, deduplicates, and flushes content-addressed artifacts to `artifacts/<first-two-hex>/<captured_sha256>`.
  6. Transaction B commits durable `evidence_sets` and `review_artifacts(evidence_set_id)` (Schema v9), releases lease, and transitions `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
- **Governing SLA / Budget**:
  - Bound by TaskContract `verification_requests[].timeout_seconds` validated by Stage B, or profile `MaxTimeoutSeconds` from `VerificationPolicyCatalog`.
  - In v1, verification requests execute strictly sequentially; the total budget is the exact sum of request timeouts plus Git collector timeout (10s) and bounded orchestration overhead (15s).
  - Aggregate verification budget is strictly capped at `MAX_AGGREGATE_VERIFICATION_BUDGET_SECONDS = 600` (10 minutes) across at most 20 requests.
- **Terminal Boundary**: Mark $T_0$ = `evidence_committed_at_epoch_ms` as the timestamp recorded in Transaction B.

### 4.2. Interval 2: ReviewBundle Compilation Window (Synthesis Phase)
- **Start ($T_0$)**: Transaction B commit completion ($T_0$), marking all verification evidence and artifacts as durably persisted in SQLite WAL and disk.
- **Operations**:
  1. Pipeline orchestrator reads persisted `evidence_sets` and `review_artifacts` from SQLite.
  2. Pipeline orchestrator synthesizes RFC 8785 JCS canonical `ReviewBundle` JSON payload.
  3. Pipeline orchestrator validates payload against canonical `docs/schemas/review-bundle.schema.json`.
  4. Pipeline orchestrator computes SHA-256 bundle hash.
  5. Transaction C inserts `review_bundles(evidence_set_id)` with persisted `bundle_committed_at_epoch_ms`, `compilation_latency_ms`, `latency_measurement_status`, and `nfr008_met`.
  6. Inserts proposed audit event `REVIEW_BUNDLE_GENERATED`, and transitions `tasks.state`: `EVIDENCE_READY -> REVIEWING`.
- **End ($T_1$)**: Transaction C commit completion ($T_1$).
- **Governing SLA & Deadlock Prevention**:
  - Bound by **NFR-008**:
    `compilation_latency_ms = bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms`
  - In normal uninterrupted execution: `0 <= compilation_latency_ms <= 3000 ms` -> `nfr008_met = 1`, `latency_measurement_status = 'MEASURED_IN_PROCESS'`.
  - In post-crash restart or transient delay: `compilation_latency_ms > 3000 ms` -> `nfr008_met = 0`, `latency_measurement_status = 'RECOVERED_AFTER_RESTART'`.
  - **SLA Breach is NOT a Persistence Blocker**: When `nfr008_met = 0`, Transaction C commits successfully, advances state to `REVIEWING`, and logs a diagnostic audit finding. The task is never stranded in `EVIDENCE_READY`.

---

## 5. Database Schema and DDL Constraints

The latency invariants and crash-safe non-deadlocking semantics are enforced directly at the SQLite database layer:

```sql
-- Schema v9: evidence_sets
CREATE TABLE evidence_sets (
    evidence_set_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    git_evidence_json TEXT NOT NULL CHECK (
        json_valid(git_evidence_json) = 1 AND
        json_type(git_evidence_json, '$.actual_changed_files') = 'array'
    ),
    test_evidence_json TEXT NOT NULL CHECK (
        json_valid(test_evidence_json) = 1 AND
        json_type(test_evidence_json, '$.executed_commands') = 'array'
    ),
    policy_findings_json TEXT NOT NULL CHECK (
        json_valid(policy_findings_json) = 1
    ),
    unverified_claims_json TEXT NOT NULL CHECK (
        json_valid(unverified_claims_json) = 1
    ),
    evidence_committed_at_epoch_ms INTEGER NOT NULL CHECK (evidence_committed_at_epoch_ms > 0),
    collected_at TEXT NOT NULL CHECK (LENGTH(collected_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

-- Schema v9: review_bundles
CREATE TABLE review_bundles (
    bundle_id TEXT PRIMARY KEY,
    evidence_set_id TEXT NOT NULL UNIQUE REFERENCES evidence_sets(evidence_set_id) ON DELETE RESTRICT,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    bundle_payload_json TEXT NOT NULL CHECK (
        json_valid(bundle_payload_json) = 1 AND
        json_type(bundle_payload_json, '$.worker_claims') = 'object' AND
        json_type(bundle_payload_json, '$.actual_git_evidence') = 'object'
    ),
    bundle_hash TEXT NOT NULL CHECK (
        LENGTH(bundle_hash) = 64 AND
        NOT (bundle_hash GLOB '*[^0-9a-f]*')
    ),
    evidence_committed_at_epoch_ms INTEGER NOT NULL CHECK (evidence_committed_at_epoch_ms > 0),
    bundle_committed_at_epoch_ms INTEGER NOT NULL CHECK (bundle_committed_at_epoch_ms >= evidence_committed_at_epoch_ms),
    compilation_latency_ms INTEGER NOT NULL CHECK (
        compilation_latency_ms >= 0 AND
        compilation_latency_ms = (bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms)
    ),
    latency_measurement_status TEXT NOT NULL CHECK (
        latency_measurement_status IN ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART', 'MEASUREMENT_TIMEOUT')
    ),
    nfr008_met INTEGER NOT NULL CHECK (
        nfr008_met IN (0, 1) AND (
            (nfr008_met = 1 AND compilation_latency_ms <= 3000) OR
            (nfr008_met = 0 AND compilation_latency_ms > 3000)
        )
    ),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_review_bundles_evidence_time_guard
BEFORE INSERT ON review_bundles
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'evidence timestamp mismatch: evidence_committed_at_epoch_ms does not match evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.evidence_committed_at_epoch_ms = NEW.evidence_committed_at_epoch_ms
    );
END;
```

---

## 6. Audit Event Governance

The audit event types associated with ReviewBundle compilation are registered with status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C upon successful ReviewBundle synthesis and persistence. Details include `bundle_id`, `bundle_hash`, `compilation_latency_ms`, `latency_measurement_status`, `nfr008_met`, and `evidence_set_id`.
2. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed transaction if Transaction C fails due to clock regression ($T_1 < T_0$), schema validation failure, or bundle hash tampering conflict. Details include failure reason, timestamps, and error diagnostics.

Canonical event registry and domain constants reconciliation will occur only after formal External Supervisor approval of ADR-018.

---

## 7. Proposed Wording for Canonical NFR-008 Reconciliation

Upon formal approval of this proposal, the text of **NFR-008** in `docs/02_REQUIREMENTS.md` will be updated via single-pass canonical reconciliation:

### Current Canonical Text (Level 4):
> *"NFR-008: The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

### Proposed Reconciled Text:
> *"NFR-008: The Supervisor Control Plane shall synthesize, canonicalize (JCS RFC 8785), validate, and persist the attempt-scoped ReviewBundle within 3.0 seconds (Interval 2: T1 - T0 <= 3.0 seconds) of durable verification evidence finalization (Transaction B commit at T0) on repositories up to 10,000 files. Independent test execution and evidence collection duration (Interval 1) is governed by task contract verification budgets. If compilation latency exceeds 3.0 seconds due to restart recovery or system load, the ReviewBundle is durably persisted with nfr008_met = 0 and an audit diagnostic finding without deadlock."*

---

## 8. Governance Process & Status Invariants

1. **PROPOSAL Status**: `PENDING_EXTERNAL_REVIEW`.
2. **Canonical Spec Immutability**: Neither `docs/02_REQUIREMENTS.md` nor `docs/04_ARCHITECTURE.md` shall be modified prior to explicit External Supervisor approval of this proposal.
3. **Draft ADR-018 Dependency**: `DRAFT-ADR-018` references this proposal; ADR-018 cannot be marked `ACCEPTED` until this proposal is formally approved.
4. **Execution Graph Decoupling**: This proposal governs specification reconciliation and does not block Subtask P04A or P04B execution once architecture is approved.
