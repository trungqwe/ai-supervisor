# PROPOSAL-P04-002: ReviewBundle Latency Measurement Semantics (NFR-008 Reconciliation)

> **Proposal ID**: `PROPOSAL-P04-002`
> **Revision**: 3
> **Title**: Formal Latency Measurement Semantics for ReviewBundle Compilation & Pipeline Reconciliation
> **Author**: AI Engineering Supervisor Team
> **Status**: `PENDING_EXTERNAL_REVIEW`
> **Date**: 2026-09-26
> **Target Requirement**: `docs/02_REQUIREMENTS.md` (NFR-008)
> **Related Architecture**: `docs/04_ARCHITECTURE.md` (Section 7), `docs/10_REVIEW_BUNDLE.md`, `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`

---

## 1. Executive Summary

Canonical non-functional requirement **NFR-008** specifies:
> *"The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

At the same time, functional requirement **FR-008** and canonical **Architecture Section 7** require that the `ReviewBundle` contain **independent verification evidence** (`actual_test_evidence` and `actual_git_evidence`) captured by running Git collectors, test suites, compilers, and linters in isolated Windows AppContainers under the Supervisor Control Plane's authority.

Running real-world test suites (e.g., `go test -race ./...`, `npm test`, `pytest`) legitimately requires durations ranging from tens of seconds to several minutes, inherently exceeding 3 seconds. If NFR-008's 3-second clock begins at worker report submission and includes external test execution, every realistic engineering task will breach NFR-008 regardless of Supervisor Control Plane efficiency. Conversely, omitting test execution from the ReviewBundle violates audit integrity.

This proposal establishes a rigorous, unambiguous measurement model that reconciles NFR-008 without weakening audit guarantees, and defines the change governance process required before canonical `docs/02_REQUIREMENTS.md` or ADR-018 can be formally adopted.

---

## 2. Problem Statement & Tension Analysis

### 2.1. The Conflict
1. **Canonical NFR-008**: Stipulates a hard performance budget of <= 3.0 seconds from worker completion on repositories up to 10,000 files.
2. **Canonical FR-008 & ReviewBundle Schema**: Stipulates that `ReviewBundle.actual_test_evidence` must record the Supervisor's independently executed test commands and exit codes.
3. **Execution Reality**:
   - Compiling and executing test suites in Windows AppContainers with explicit handle inheritance and network denial is bounded by project build times, external toolchain performance, and test complexity.
   - For real projects, verification tests typically run between 5 and 60 seconds.

### 2.2. Governance Conflict
Modifying the text or interpretation of NFR-008 unilaterally violates the 9-level decision hierarchy of `docs/24_CHANGE_GOVERNANCE.md` (Level 4 Requirement Specification cannot be implicitly altered by Level 6 Task Contracts or Level 8 Suggestions). Formal approval of this proposal by the External Supervisor is mandatory before modifying canonical documentation.

---

## 3. Proposed Measurement Model: Two-Interval Latency Semantics

To reconcile NFR-008 with FR-008 and real-world execution physics while maintaining strict auditability, the verification pipeline is structured into two precisely bounded operational intervals:

### 3.1. Interval 1: Evidence Acquisition Window (Work Execution Phase)
- **Start**: Worker signals report completion; Transaction A validates report, stores structured `worker_claims` (Schema v6), and transitions `tasks.state`: `RUNNING -> REPORT_READY`.
- **Operations**:
  1. Supervisor revalidates physical worktree handle identity (`FileIdInfo`).
  2. Supervisor creates snapshot `<SUPERVISOR_STATE_ROOT>/snapshots/<attempt_id>/` from actual verified HEAD commit via Git `ls-tree -rz --full-tree` and `cat-file --batch`.
  3. Supervisor executes hardened in-memory Git evidence collection (`P04B`).
  4. Supervisor executes verification test commands sequentially (`P04C`) inside isolated Windows AppContainers against the immutable snapshot.
  5. Supervisor stages, deduplicates, and flushes content-addressed artifacts to `artifacts/<first-two-hex>/<captured_sha256>`.
  6. Transaction B commits durable `evidence_sets` and `review_artifacts(evidence_set_id)` (Schema v9), releases lease, and transitions `tasks.state`: `REPORT_READY -> EVIDENCE_READY`.
- **Governing SLA / Budget**:
  - Bound by TaskContract `verification_requests[].timeout_seconds` validated by Stage B, or profile `MaxTimeoutSeconds` from `VerificationPolicyCatalog`.
  - In v1, verification requests execute strictly sequentially; the total budget is the exact sum of request timeouts plus Git collector timeout (10s) and bounded orchestration overhead (15s).
  - Arbitrary unapproved verification budget tokens or ungrounded 60,000 ms defaults are strictly prohibited.
- **Terminal Boundary**: Mark $T_0$ = `evidence_committed_at_epoch_ms` as the exact timestamp immediately before Transaction B commit succeeds.

### 3.2. Interval 2: ReviewBundle Compilation Window (Synthesis Phase)
- **Start ($T_0$)**: Transaction B commit completion ($T_0$), marking all verification evidence and artifacts as durably persisted in SQLite WAL and disk.
- **Operations**:
  1. Pipeline orchestrator reads persisted `evidence_sets` and `review_artifacts` from SQLite.
  2. Pipeline orchestrator synthesizes RFC 8785 JCS canonical `ReviewBundle` JSON payload.
  3. Pipeline orchestrator validates payload against `docs/schemas/review-bundle.schema.json`.
  4. Pipeline orchestrator computes SHA-256 bundle hash.
  5. Transaction C inserts `review_bundles(evidence_set_id)` with persisted `bundle_committed_at_epoch_ms` and `compilation_latency_ms`, inserts proposed audit event `REVIEW_BUNDLE_GENERATED`, and transitions `tasks.state`: `EVIDENCE_READY -> REVIEWING`.
- **End ($T_1$)**: Transaction C commit completion ($T_1$).
- **Governing SLA**:
  - Bound strictly by **NFR-008**:
    `compilation_latency_ms = bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms`
    `0 <= compilation_latency_ms <= 3000 ms (3.0 seconds)`
  - Measured in-process using monotonic timer around the two commit completions; recorded durably in epoch milliseconds in `review_bundles`.
  - Clock regression ($T_1 < T_0$) or latency violation (> 3000 ms) causes Transaction C to roll back via SQLite WAL. A separate fail-closed diagnostic transaction logs `REVIEW_BUNDLE_COMPILATION_REJECTED` into `audit_events` without transitioning TaskState to `REVIEWING`.

---

## 4. Database Schema and DDL Constraints

The latency invariants are enforced directly at the SQLite database layer:

```sql
-- Schema v9: evidence_sets
CREATE TABLE evidence_sets (
    evidence_set_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE RESTRICT,
    attempt_id TEXT NOT NULL UNIQUE REFERENCES task_attempts(attempt_id) ON DELETE RESTRICT,
    contract_id TEXT NOT NULL REFERENCES task_contracts(contract_id) ON DELETE RESTRICT,
    fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
    git_evidence_json TEXT NOT NULL CHECK (LENGTH(git_evidence_json) > 0),
    test_evidence_json TEXT NOT NULL CHECK (LENGTH(test_evidence_json) > 0),
    policy_findings_json TEXT NOT NULL CHECK (LENGTH(policy_findings_json) > 0),
    unverified_claims_json TEXT NOT NULL CHECK (LENGTH(unverified_claims_json) > 0),
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
    bundle_payload_json TEXT NOT NULL CHECK (LENGTH(bundle_payload_json) > 0),
    bundle_hash TEXT NOT NULL CHECK (LENGTH(bundle_hash) = 64 AND NOT (bundle_hash GLOB '*[^0-9a-f]*')),
    evidence_committed_at_epoch_ms INTEGER NOT NULL CHECK (evidence_committed_at_epoch_ms > 0),
    bundle_committed_at_epoch_ms INTEGER NOT NULL CHECK (bundle_committed_at_epoch_ms > 0),
    compilation_latency_ms INTEGER NOT NULL CHECK (compilation_latency_ms BETWEEN 0 AND 3000),
    generated_at TEXT NOT NULL CHECK (LENGTH(generated_at) > 0),
    FOREIGN KEY(contract_id, task_id) REFERENCES task_contracts(contract_id, task_id) ON DELETE RESTRICT,
    CHECK (
        bundle_committed_at_epoch_ms >= evidence_committed_at_epoch_ms AND
        compilation_latency_ms = (bundle_committed_at_epoch_ms - evidence_committed_at_epoch_ms)
    )
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

## 5. Audit Event Governance

The audit event types associated with ReviewBundle compilation are registered with status `PROPOSED_UNTIL_ADR_ACCEPTANCE`:
1. `REVIEW_BUNDLE_GENERATED`: Recorded in Transaction C upon successful ReviewBundle synthesis and persistence. Details include `bundle_id`, `bundle_hash`, `compilation_latency_ms`, and `evidence_set_id`.
2. `REVIEW_BUNDLE_COMPILATION_REJECTED`: Recorded in a separate fail-closed transaction if Transaction C fails due to clock regression, latency violation (> 3000 ms), schema validation failure, or hash conflict. Details include failure reason, timestamps, and error diagnostics.

Canonical event registry and domain constants reconciliation will occur only after formal External Supervisor approval of ADR-018.

---

## 6. Proposed Wording for Canonical NFR-008 Reconciliation

Upon formal approval of this proposal, the text of **NFR-008** in `docs/02_REQUIREMENTS.md` will be updated via single-pass canonical reconciliation:

### Current Canonical Text (Level 4):
> *"NFR-008: The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files."*

### Proposed Reconciled Text:
> *"NFR-008: The Supervisor Control Plane shall synthesize, canonicalize (JCS RFC 8785), validate, and persist the attempt-scoped ReviewBundle within 3.0 seconds (Interval 2: T1 - T0 <= 3.0 seconds) of durable verification evidence finalization (Transaction B commit at T0) on repositories up to 10,000 files. Independent test execution and evidence collection duration (Interval 1) is governed by task contract verification budgets."*

---

## 7. Governance Process & Status Invariants

1. **PROPOSAL Status**: `PENDING_EXTERNAL_REVIEW`.
2. **Canonical Spec Immutability**: Neither `docs/02_REQUIREMENTS.md` nor `docs/04_ARCHITECTURE.md` shall be modified prior to explicit External Supervisor approval of this proposal.
3. **Draft ADR-018 Dependency**: `DRAFT-ADR-018` references this proposal; ADR-018 cannot be marked `ACCEPTED` until this proposal is formally approved.
4. **Execution Graph Decoupling**: This proposal governs specification reconciliation and does not block Subtask P04A or P04B execution once architecture is approved.
