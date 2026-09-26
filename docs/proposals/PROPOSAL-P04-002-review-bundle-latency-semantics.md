# PROPOSAL-P04-002: ReviewBundle Latency Measurement Semantics (NFR-008 Reconciliation)

> **Proposal ID**: `PROPOSAL-P04-002`
> **Revision**: 7
> **Title**: Formal Latency Measurement Semantics for ReviewBundle Compilation & Pipeline Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Target Requirement**: `docs/02_REQUIREMENTS.md` (NFR-008)
> **Related Architecture**: `docs/04_ARCHITECTURE.md` (Section 7), `docs/10_REVIEW_BUNDLE.md`, `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`
> **Audited Baseline**: `b5a1d8ef0c38e52ed71370a1e0dfa17e0d3e5f2d`
> **Preservation Baseline Commit**: `6e1993da150031a9465901a7019c71257de44312` (Revision 11)
> **Active Gate**: `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_12`
> **Supersedes**: `PROPOSAL-P04-002` Revision 6
> **External Audit Tracking**: Remediates Finding `P04-ARCH-R11-001` (`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_011.md`).

---

## 1. Executive Summary

Canonical non-functional requirement **NFR-008** specifies:
> *"The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

At the same time, functional requirement **FR-008** and canonical **Architecture Section 7** require that the `ReviewBundle` contain **independent verification evidence** (`actual_test_evidence` and `actual_git_evidence`) captured by running Git collectors, test suites, compilers, and linters in isolated Windows AppContainers under the Supervisor Control Plane's authority.

Running real-world test suites (e.g., `go test -race ./...`, `npm test`, `pytest`) legitimately requires durations ranging from tens of seconds to several minutes, inherently exceeding 3 seconds. If NFR-008's 3-second clock begins at worker report submission and includes external test execution, every realistic engineering task will breach NFR-008 regardless of Supervisor Control Plane efficiency. Conversely, omitting test execution from the ReviewBundle violates audit integrity.

Furthermore, External Re-Audit 011 confirmed finding `P04-ARCH-R11-001` as `CLOSED_AT_DESIGN_LEVEL`, establishing that the pre-commit measurement interval is strictly a ReviewBundle assembly diagnostic interval, separate from canonical NFR-008. Canonical NFR-008 compliance status is held strictly as `UNVERIFIED` pending future canonical reconciliation. Post-commit telemetry is decoupled from Transaction C audit events.

This proposal establishes a rigorous, crash-safe measurement model that designates compilation latency as an assembly diagnostic interval, decouples post-commit telemetry from atomic database transactions, enforces strict integer typing constraints, and establishes deterministic audit event idempotency mapping.

---

## 2. Problem Statement & Tension Analysis

### 2.1. The Operational Conflict
1. **Canonical NFR-008**: Stipulates a hard performance budget of <= 3.0 seconds from worker completion on repositories up to 10,000 files.
2. **Canonical FR-008 & ReviewBundle Schema**: Stipulates that `ReviewBundle.actual_test_evidence` must record the Supervisor's independently executed test commands and exit codes.
3. **Execution Reality**:
   - Compiling and executing test suites in Windows AppContainers with explicit handle inheritance and network denial is bounded by project build times, external toolchain performance, and test complexity.
   - For real projects, verification tests typically run between 5 and 60 seconds.

### 2.2. Timestamp Semantics & Measurement Boundaries
1. **Application Pre-Commit vs Actual Commit-Return**:
   - The timestamp recorded inside the immutable `review_bundles` table is captured in application memory immediately upon successful assembly, canonicalization, and validation of the payload, *prior* to issuing SQLite `tx.Commit()`.
   - The actual SQLite WAL commit completion occurs when the database engine completes disk writes and returns control to Go.
   - Calling the pre-commit timestamp "commit completion" was inaccurate.
2. **Monotonic Timers Cannot Bridge Restarts**:
   - An in-process monotonic timer (`time.Since()`) initialized during Transaction B is lost if the daemon crashes or restarts before Transaction C.
   - Post-restart recovery must rely on durable wall-clock timestamps (`evidence_finalized_at_epoch_ms` and `bundle_assembled_at_epoch_ms`) for diagnostic measurement.
3. **SLA Breach Must Not Block Persistence**:
   - If `compilation_latency_ms <= 3000` is enforced as a SQLite CHECK constraint, any restart recovery attempt (where elapsed time exceeds 3,000 ms) fails SQL validation, permanently stranding the task in `EVIDENCE_READY`.
   - The SLA outcome must be recorded as durable diagnostic data (`nfr008_met = 0`), allowing Transaction C to commit and advance state to `REVIEWING`.

---

## 3. Formal Measurement Contract

To eliminate ambiguity, this proposal defines a unified, non-conflicting measurement contract:

### 3.1. Primary Immutable Measurement: Assembly Diagnostic Boundary
The columns in `review_bundles` strictly measure the **ReviewBundle Assembly Diagnostic Interval** (Interval 2):
1. **`evidence_finalized_at_epoch_ms`**: Epoch millisecond timestamp durably committed in Transaction B when verification evidence sets and artifacts are finalized.
2. **`bundle_assembled_at_epoch_ms`**: Epoch millisecond timestamp captured when ReviewBundle JSON assembly, RFC 8785 JCS canonicalization, and schema validation succeed, immediately before initiating Transaction C commit.
3. **`compilation_latency_ms`**: The deterministic difference:
   `compilation_latency_ms = bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms`
   Designated strictly as an in-process synthesis assembly diagnostic interval, separate from canonical NFR-008.
4. **`latency_measurement_status`**:
   - `'MEASURED_IN_PROCESS'`: Measured during continuous execution without process restart.
   - `'RECOVERED_AFTER_RESTART'`: Measured upon daemon restart recovery; records wall-clock recovery latency as a durable diagnostic.
5. **`nfr008_compliance_status`**:
   - Strictly `'UNVERIFIED'` in Schema v9 (`CHECK (nfr008_compliance_status = 'UNVERIFIED')`).
   - Canonical reconciliation of NFR-008 remains pending until External Supervisor approval.

### 3.2. Secondary Telemetry: Decoupled Commit Return Duration
1. **Decoupled from Transaction C Audit Event**: The proposed audit event `REVIEW_BUNDLE_GENERATED` emitted in Transaction C does NOT contain `commit_duration_ms`.
2. **Best-Effort Telemetry**: Upon return from SQLite `tx.Commit()` in Transaction C, the orchestrator computes the actual commit duration using in-process monotonic measurement (`time.Since(commitStart)`) purely as best-effort in-process telemetry.

### 3.3. Measurement Truth Table & Invariants

| `latency_measurement_status` | `compilation_latency_ms` | `nfr008_compliance_status` | Operational Meaning |
| :--- | :--- | :--- | :--- |
| `MEASURED_IN_PROCESS` | `>= 0` | `'UNVERIFIED'` | Normal in-process execution; assembly diagnostic recorded. |
| `RECOVERED_AFTER_RESTART` | `>= 0` | `'UNVERIFIED'` | Daemon restart recovery; recovery assembly diagnostic recorded. |

Negative latency, non-integer timestamps, or values of `nfr008_compliance_status` other than `'UNVERIFIED'` are strictly rejected by SQLite CHECK constraints.

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
  6. Transaction B (owned by `P04D`) commits durable `evidence_sets` and `review_artifacts`, releases verification lease, and transitions `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
- **Governing SLA / Budget**:
  - Bound by TaskContract `verification_requests[].timeout_seconds` validated by Stage B, or profile `MaxTimeoutSeconds` from `VerificationPolicyCatalog`.
  - In v1, verification requests execute strictly sequentially; the total budget is the exact sum of request timeouts plus Git collector timeout (10s) and bounded orchestration overhead (15s).
  - Aggregate verification budget is strictly capped at `MAX_AGGREGATE_VERIFICATION_BUDGET_SECONDS = 600` (10 minutes) across at most 20 requests.
- **Terminal Boundary**: Mark $T_0$ = `evidence_finalized_at_epoch_ms` as the timestamp recorded in Transaction B.

### 4.2. Interval 2: ReviewBundle Compilation Window (Synthesis Phase)
- **Start ($T_0$)**: Transaction B commit completion ($T_0$), marking all verification evidence and artifacts as durably persisted in SQLite WAL and disk.
- **Operations**:
  1. Pipeline orchestrator reads persisted `evidence_sets` and `review_artifacts` from SQLite.
  2. Pipeline orchestrator synthesizes RFC 8785 JCS canonical `ReviewBundle` JSON payload.
  3. Pipeline orchestrator validates payload against canonical `docs/schemas/review-bundle.schema.json`.
  4. Pipeline orchestrator computes SHA-256 bundle hash.
  5. Mark $T_1$ = `bundle_assembled_at_epoch_ms` immediately before initiating Transaction C.
  6. Transaction C (owned by `P04D`) inserts `review_bundles` with persisted `bundle_assembled_at_epoch_ms`, `compilation_latency_ms`, `latency_measurement_status`, and `nfr008_met`.
  7. Inserts proposed audit event `REVIEW_BUNDLE_GENERATED`, and transitions `tasks.state`: `EVIDENCE_READY -> REVIEWING`.
- **End ($T_1$)**: Transaction C commit initiation and atomic persistence.
- **Governing SLA & Deadlock Prevention**:
  - Bound by **NFR-008**:
    `compilation_latency_ms = bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms`
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
    fencing_token INTEGER NOT NULL CHECK (
        typeof(fencing_token) = 'integer' AND fencing_token >= 1 AND fencing_token <= 9223372036854775807
    ),
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
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(evidence_finalized_at_epoch_ms) = 'integer' AND evidence_finalized_at_epoch_ms > 0
    ),
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
    evidence_finalized_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(evidence_finalized_at_epoch_ms) = 'integer' AND evidence_finalized_at_epoch_ms > 0
    ),
    bundle_assembled_at_epoch_ms INTEGER NOT NULL CHECK (
        typeof(bundle_assembled_at_epoch_ms) = 'integer' AND bundle_assembled_at_epoch_ms >= evidence_finalized_at_epoch_ms
    ),
    compilation_latency_ms INTEGER NOT NULL CHECK (
        typeof(compilation_latency_ms) = 'integer' AND
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
    SELECT RAISE(ABORT, 'lineage mismatch: evidence_set_id does not match task_id, attempt_id, contract_id in evidence_sets')
    WHERE NOT EXISTS (
        SELECT 1 FROM evidence_sets e
        WHERE e.evidence_set_id = NEW.evidence_set_id
          AND e.task_id = NEW.task_id
          AND e.attempt_id = NEW.attempt_id
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

## 6. Audit Event Governance

The audit event types associated with ReviewBundle compilation are registered under status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C upon successful ReviewBundle synthesis and persistence. Details include `bundle_id`, `bundle_hash`, `compilation_latency_ms`, `latency_measurement_status`, and `nfr008_compliance_status` ('UNVERIFIED'). Emitted within Transaction C *without* `commit_duration_ms`.
2. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed diagnostic transaction if compilation fails, schema validation fails, clock regresses ($T_1 < T_0$), or bundle hash conflicts with a pre-existing bundle. Details include failure reason, timestamps, and error diagnostics.

#### Audit Event Idempotency Mapping (P04-ARCH-R12-003)
Because canonical `audit_events` lacks an `idempotency_key` column, idempotency maps deterministically to `audit_events.event_id`:
`event_id = SHA256(attempt_id || ":" || contract_id || ":" || reason || ":" || sanitized_input_fingerprint)`
- **Exact Duplicate Event**: Processed as an idempotent replay without duplication.
- **Matching Event ID with Differing Lineage/Payload**: Raised as an integrity conflict.
- **Diagnostic Transaction Rollback**: Does not declare event recorded if transaction fails.

---

## 7. Proposed Wording for Canonical NFR-008 Reconciliation

Upon formal approval of this proposal, the text of **NFR-008** in `docs/02_REQUIREMENTS.md` will be updated via single-pass canonical reconciliation:

### Current Canonical Text (Level 4):
> *"NFR-008: The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

### Proposed Reconciled Text:
> *"NFR-008: The Supervisor Control Plane shall synthesize, canonicalize (JCS RFC 8785), validate, and prepare for durable commit the attempt-scoped ReviewBundle within 3.0 seconds (Interval 2: bundle_assembled_at_epoch_ms - evidence_finalized_at_epoch_ms <= 3.0 seconds) of durable verification evidence finalization (Transaction B commit) on repositories up to 10,000 files. Independent test execution and evidence collection duration (Interval 1) is governed by task contract verification budgets. If assembly latency exceeds 3.0 seconds due to restart recovery or system load, the ReviewBundle is durably persisted with nfr008_met = 0 and an audit diagnostic finding without deadlock."*

---

## 8. Governance Process & Status Invariants

1. **PROPOSAL Status**: `PENDING_EXTERNAL_REVIEW`.
2. **Canonical Spec Immutability**: Neither `docs/02_REQUIREMENTS.md` nor `docs/04_ARCHITECTURE.md` shall be modified prior to explicit External Supervisor approval of this proposal.
3. **Draft ADR-018 Dependency**: `DRAFT-ADR-018` references this proposal; ADR-018 cannot be marked `ACCEPTED` until this proposal is formally approved.
4. **Execution Graph Decoupling**: This proposal governs specification reconciliation and does not block Subtask P04A or P04B execution once architecture is approved.
