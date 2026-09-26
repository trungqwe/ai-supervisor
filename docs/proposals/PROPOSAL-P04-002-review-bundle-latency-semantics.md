# PROPOSAL-P04-002: ReviewBundle Latency Measurement Semantics (NFR-008 Reconciliation)

> **Proposal ID**: `PROPOSAL-P04-002`
> **Revision**: 6
> **Title**: Formal Latency Measurement Semantics for ReviewBundle Compilation & Pipeline Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Target Requirement**: `docs/02_REQUIREMENTS.md` (NFR-008)
> **Related Architecture**: `docs/04_ARCHITECTURE.md` (Section 7), `docs/10_REVIEW_BUNDLE.md`, `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`
> **Audited Baseline**: `6e1993da150031a9465901a7019c71257de44312`
> **External Audit Tracking**: Remediates Finding `P04-ARCH-R11-001` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_010.md`).

---

## 1. Executive Summary

Canonical non-functional requirement **NFR-008** specifies:
> *"The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

At the same time, functional requirement **FR-008** and canonical **Architecture Section 7** require that the `ReviewBundle` contain **independent verification evidence** (`actual_test_evidence` and `actual_git_evidence`) captured by running Git collectors, test suites, compilers, and linters in isolated Windows AppContainers under the Supervisor Control Plane's authority.

Running real-world test suites (e.g., `go test -race ./...`, `npm test`, `pytest`) legitimately requires durations ranging from tens of seconds to several minutes, inherently exceeding 3 seconds. If NFR-008's 3-second clock begins at worker report submission and includes external test execution, every realistic engineering task will breach NFR-008 regardless of Supervisor Control Plane efficiency. Conversely, omitting test execution from the ReviewBundle violates audit integrity.

Furthermore, External Re-Audit 010 established two crucial governance and measurement invariants:
1. Canonical NFR-008 measures from **worker completion** (the moment the worker completes its turn and emits its completion signal). The interval measured between evidence finalization and ReviewBundle assembly is internal to the supervisor pipeline and represents an assembly diagnostic; the supervisor control plane **must not** use this internal interval to self-declare NFR-008 compliance. NFR-008 compliance status must remain strictly **`UNVERIFIED`** until an approved canonical worker-completion origin timestamp is formally incorporated or canonical requirements are amended via change governance (`docs/24_CHANGE_GOVERNANCE.md`).
2. Audit events committed inside SQLite transactions cannot contain telemetry regarding the duration of their own future disk commit (`commit_duration_ms`). The proposed audit event `REVIEW_BUNDLE_GENERATED` is committed atomically inside Transaction C without `commit_duration_ms`, while `commit_duration_ms` is measured after commit returns purely as best-effort in-process telemetry.

This proposal establishes a rigorous, crash-safe measurement model that separates the ReviewBundle assembly diagnostic from canonical NFR-008 compliance, eliminates database deadlocks, explicitly defines the pre-commit assembly boundary, and defines the change governance process required before canonical `docs/02_REQUIREMENTS.md` or ADR-018 can be formally adopted.

---

## 2. Problem Statement & Tension Analysis

### 2.1. The Operational Conflict
1. **Canonical NFR-008**: Stipulates a hard performance budget of <= 3.0 seconds from worker completion on repositories up to 10,000 files.
2. **Canonical FR-008 & ReviewBundle Schema**: Stipulates that `ReviewBundle.actual_test_evidence` must record the Supervisor's independently executed test commands and exit codes.
3. **Execution Reality**:
   - Compiling and executing test suites in Windows AppContainers with explicit handle inheritance and network denial is bounded by project build times, external toolchain performance, and test complexity.
   - For real projects, verification tests typically run between 5 and 60 seconds.
4. **Worker Completion vs Pipeline Assembly**:
   - The supervisor plane does not currently have an approved, tamper-proof canonical origin timestamp recording exact worker completion in the schema.
   - Attempting to declare canonical NFR-008 "MET" using an internal supervisor interval (`evidence-finalized -> bundle-assembled`) misrepresents specification compliance.

### 2.2. Timestamp Semantics & Measurement Boundaries
1. **Application Pre-Commit Timestamps**:
   - `evidence_finalized_at_epoch_ms` is an application timestamp chosen immediately before Transaction B commit and persisted durably by Transaction B (it is not called the commit completion timestamp).
   - `bundle_assembled_at_epoch_ms` is captured in application memory immediately upon successful assembly, RFC 8785 canonicalization, and schema validation of the ReviewBundle payload, *prior* to issuing SQLite `tx.Commit()` in Transaction C.
   - The difference `bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms` is strictly the **ReviewBundle assembly diagnostic** (`compilation_latency_ms`).
2. **Monotonic Timers Cannot Bridge Restarts**:
   - An in-process monotonic timer (`time.Since()`) initialized during Transaction B is lost if the daemon crashes or restarts before Transaction C.
   - Post-restart recovery must rely on durable wall-clock timestamps (`evidence_finalized_at_epoch_ms` and `bundle_assembled_at_epoch_ms`) for diagnostic measurement (`latency_measurement_status = 'RECOVERED_AFTER_RESTART'`).
3. **Post-Commit Telemetry Causality**:
   - SQLite WAL commit completion occurs when the database engine completes disk writes and returns control to Go.
   - Since `REVIEW_BUNDLE_GENERATED` is an audit event inserted *inside* Transaction C, it cannot contain `commit_duration_ms`.
   - `commit_duration_ms` is measured using monotonic clocks *after* `tx.Commit()` returns, purely as best-effort in-process telemetry.

---

## 3. Formal Measurement Contract

To eliminate ambiguity, this proposal defines a unified, non-conflicting measurement contract:

### 3.1. Primary Immutable Measurement: ReviewBundle Assembly Diagnostic
The columns in `review_bundles` strictly measure the **ReviewBundle Assembly Diagnostic Interval**:
1. **`evidence_finalized_at_epoch_ms`**: Epoch millisecond timestamp chosen by the application before Transaction B commit and persisted durably by Transaction B when verification evidence sets and artifacts are finalized.
2. **`bundle_assembled_at_epoch_ms`**: Epoch millisecond timestamp captured when ReviewBundle JSON assembly, RFC 8785 JCS canonicalization, and schema validation succeed, immediately before initiating Transaction C commit.
3. **`compilation_latency_ms`**: The deterministic difference representing the ReviewBundle assembly diagnostic:
   `compilation_latency_ms = bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms`
4. **`latency_measurement_status`**:
   - `'MEASURED_IN_PROCESS'`: Measured during continuous execution without process restart.
   - `'RECOVERED_AFTER_RESTART'`: Measured upon daemon restart recovery; records wall-clock recovery latency as a durable diagnostic.
5. **`nfr008_compliance_status`**:
   - Strictly enforced as **`'UNVERIFIED'`** in v1:
     `nfr008_compliance_status TEXT NOT NULL CHECK (nfr008_compliance_status = 'UNVERIFIED')`
   - Canonical NFR-008 compliance cannot be evaluated or self-declared until an approved worker-completion origin timestamp is integrated or canonical requirements are amended via change governance.

### 3.2. Secondary Telemetry: Actual Commit Return Duration
Upon return from SQLite `tx.Commit()` in Transaction C, the orchestrator computes the actual commit duration using in-process monotonic measurement (`time.Since(commitStart)`). In v1:
- `commit_duration_ms` is purely best-effort in-process process telemetry (emitted via structured logger/metrics).
- It is **NOT** part of the immutable `review_bundles` record.
- It is **NOT** included inside the atomic `REVIEW_BUNDLE_GENERATED` audit event.
- It is **NOT** used for SLA determination.
- No new database event type or schema column is created for this telemetry.

### 3.3. Measurement Truth Table & Invariants

| `latency_measurement_status` | `compilation_latency_ms` | `nfr008_compliance_status` | Operational Meaning |
| :--- | :--- | :--- | :--- |
| `MEASURED_IN_PROCESS` | `0` to `3000` | `UNVERIFIED` | Normal execution; assembly diagnostic <= 3s; NFR-008 unverified against worker completion. |
| `MEASURED_IN_PROCESS` | `> 3000` | `UNVERIFIED` | In-process execution delay; assembly diagnostic > 3s (diagnostic recorded, no deadlock). |
| `RECOVERED_AFTER_RESTART` | `0` to `3000` | `UNVERIFIED` | Rapid restart recovery; assembly diagnostic <= 3s; NFR-008 unverified. |
| `RECOVERED_AFTER_RESTART` | `> 3000` | `UNVERIFIED` | Normal restart recovery; assembly diagnostic > 3s due to downtime (diagnostic recorded, no deadlock). |

All other combinations (e.g. `nfr008_compliance_status != 'UNVERIFIED'`, negative latency, or latency calculation mismatch) are strictly rejected by SQLite CHECK constraints.

---

## 4. Proposed Measurement Model: Two-Interval Pipeline

### 4.1. Interval 1: Evidence Acquisition Window (Work Execution Phase)
- **Start**: Worker signals report completion; Transaction A validates report against canonical schema, verifies clean worktree/index, stores structured `worker_claims` (Schema v6), and transitions `tasks.state`: `RUNNING -> REPORT_READY`.
- **Operations**:
  1. Supervisor revalidates physical worktree handle identity (`FileIdInfo`) and re-verifies clean index/worktree.
  2. Supervisor creates snapshot `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/` from actual verified HEAD commit via Git `ls-tree -rz --full-tree` and `cat-file --batch`.
  3. Supervisor executes hardened in-memory Git evidence collection (`P04B`).
  4. Supervisor executes verification test commands sequentially (`P04C`) inside isolated Windows AppContainers against the immutable snapshot.
  5. Supervisor stages, deduplicates, and flushes content-addressed artifacts to `artifacts/<first-two-hex>/<captured_sha256>`.
  6. Transaction B (owned by `P04D`) commits durable `evidence_sets` and `review_artifacts`, marks active lease `COMPLETED`, persists application pre-commit timestamp `evidence_finalized_at_epoch_ms`, and transitions `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
- **Governing SLA / Budget**:
  - Bound by TaskContract `verification_requests[].timeout_seconds` validated by Stage B, or profile `MaxTimeoutSeconds` from `VerificationPolicyCatalog`.
  - In v1, verification requests execute strictly sequentially; the total budget is the exact sum of request timeouts plus Git collector timeout (10s) and bounded orchestration overhead (15s).

### 4.2. Interval 2: ReviewBundle Assembly Diagnostic (Synthesis & Validation Phase)
- **Start ($T_0$)**: `evidence_finalized_at_epoch_ms` persisted durably in Transaction B.
- **Operations**:
  1. Pipeline orchestrator loads attempt-scoped `WorkerClaim`, `EvidenceSet`, and `TaskContract`.
  2. Pipeline orchestrator synthesizes the `ReviewBundle` payload JSON, including policy validation findings.
  3. Pipeline orchestrator validates payload against canonical `docs/schemas/review-bundle.schema.json`.
  4. Pipeline orchestrator computes SHA-256 bundle hash.
  5. Mark $T_1$ = `bundle_assembled_at_epoch_ms` immediately before initiating Transaction C.
  6. Transaction C (owned by `P04D`) inserts `review_bundles` with persisted `evidence_finalized_at_epoch_ms`, `bundle_assembled_at_epoch_ms`, `compilation_latency_ms`, `latency_measurement_status`, and `nfr008_compliance_status = 'UNVERIFIED'`.
  7. Inserts proposed audit event `REVIEW_BUNDLE_GENERATED`, and transitions `tasks.state`: `EVIDENCE_READY -> REVIEWING`.
- **End ($T_1$)**: Transaction C commit initiation and atomic persistence.
- **Governing SLA & Deadlock Prevention**:
  - `compilation_latency_ms = bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms`
  - In normal uninterrupted execution: `0 <= compilation_latency_ms <= 3000 ms`, `latency_measurement_status = 'MEASURED_IN_PROCESS'`.
  - In post-crash restart or transient delay: `compilation_latency_ms > 3000 ms`, `latency_measurement_status = 'RECOVERED_AFTER_RESTART'`.
  - **Assembly Delay is NOT a Persistence Blocker**: Regardless of latency, Transaction C commits successfully, advances state to `REVIEWING`, and records diagnostic data. The task is never stranded in `EVIDENCE_READY`.

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
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (evidence_finalized_at_epoch_ms > 0),
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
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (evidence_finalized_at_epoch_ms > 0),
    bundle_assembled_at_epoch_ms INTEGER NOT NULL CHECK (bundle_assembled_at_epoch_ms >= evidence_finalized_at_epoch_ms),
    compilation_latency_ms INTEGER NOT NULL CHECK (
        compilation_latency_ms >= 0 AND
        compilation_latency_ms = (bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms)
    ),
    latency_measurement_status TEXT NOT NULL CHECK (
        latency_measurement_status IN ('MEASURED_IN_PROCESS', 'RECOVERED_AFTER_RESTART')
    ),
    nfr008_compliance_status TEXT NOT NULL CHECK (
        nfr008_compliance_status = 'UNVERIFIED'
    ),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT
);

CREATE TRIGGER trg_review_bundles_lineage_guard
BEFORE INSERT ON review_bundles
FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'lineage mismatch: attempt_id does not match task_id or contract_id in task_attempts')
    WHERE NOT EXISTS (
        SELECT 1 FROM task_attempts a
        WHERE a.attempt_id = NEW.attempt_id
          AND a.task_id = NEW.task_id
          AND a.contract_id = NEW.contract_id
    );
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match attempt_id in evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.attempt_id = NEW.attempt_id
          AND e.task_id = NEW.task_id
          AND e.contract_id = NEW.contract_id
          AND e.evidence_finalized_at_epoch_ms = NEW.evidence_finalized_at_epoch_ms
    );
END;

CREATE TRIGGER trg_review_bundles_no_update
BEFORE UPDATE ON review_bundles
BEGIN
    SELECT RAISE(ABORT, 'review_bundles is immutable');
END;

CREATE TRIGGER trg_review_bundles_no_delete
BEFORE DELETE ON review_bundles
BEGIN
    SELECT RAISE(ABORT, 'review_bundles is immutable');
END;
```

---

## 6. Audit Event Governance & Telemetry Separation

The audit event types associated with ReviewBundle compilation are registered under status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C upon successful ReviewBundle synthesis and persistence. Details include `bundle_id`, `bundle_hash`, `compilation_latency_ms` (ReviewBundle assembly diagnostic), `latency_measurement_status`, and `nfr008_compliance_status` (`'UNVERIFIED'`). It does **NOT** contain `commit_duration_ms`.
2. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed diagnostic transaction if compilation fails, schema validation fails, clock regresses ($T_1 < T_0$), or bundle hash conflicts with a pre-existing bundle. Uses an idempotency key (e.g. `audit:<attempt_id>:BUNDLE_HASH_CONFLICT`). If this diagnostic append fails, a compound failure error is returned to the caller without claiming the audit event was recorded. Automated review approval remains blocked.

Post-commit telemetry (`commit_duration_ms`) is measured via monotonic clock after SQLite `tx.Commit()` returns, purely as in-process structured process telemetry. It is not part of immutable evidence or atomic audit logs.

Canonical event registry and domain constants reconciliation will occur only after formal External Supervisor approval of ADR-018.

---

## 7. Proposed Wording for Canonical NFR-008 Reconciliation

Upon formal approval of this proposal, the text of **NFR-008** in `docs/02_REQUIREMENTS.md` will be updated via single-pass canonical reconciliation:

### Current Canonical Text (Level 4):
> *"NFR-008: The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

### Proposed Reconciled Text:
> *"NFR-008: The Supervisor Control Plane shall synthesize, canonicalize (JCS RFC 8785), validate, and prepare for durable commit the attempt-scoped ReviewBundle within 3.0 seconds (ReviewBundle assembly diagnostic: bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms <= 3.0 seconds) of durable verification evidence finalization (Transaction B commit) on repositories up to 10,000 files. Independent test execution and evidence collection duration (Interval 1) is governed by task contract verification budgets. Until an approved canonical worker-completion origin timestamp is incorporated into the control plane, end-to-end NFR-008 compliance status is recorded as UNVERIFIED."*

---

## 8. Governance Process & Status Invariants

1. **PROPOSAL Status**: `PENDING_EXTERNAL_REVIEW`.
2. **Canonical Spec Immutability**: Neither `docs/02_REQUIREMENTS.md` nor `docs/04_ARCHITECTURE.md` shall be modified prior to explicit External Supervisor approval of this proposal.
3. **Draft ADR-018 Dependency**: `DRAFT-ADR-018` references this proposal; ADR-018 cannot be marked `ACCEPTED` until this proposal is formally approved.
4. **Execution Graph Decoupling**: This proposal governs specification reconciliation and does not block Subtask P04A or P04B execution once architecture is approved.
