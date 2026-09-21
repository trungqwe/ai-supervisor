# ADR-011: WorkerReport Handoff and Antigravity Invocation Boundary

## Status
ACCEPTED

## Context
During Phase P01 Track P01-C (AO ↔ Antigravity CLI Integration Proof), empirical evaluation of pinned Agent Orchestrator (`v0.13.0`) and Antigravity CLI (`1.2.7`) established the following runtime facts:
1. **No Native WorkerReport in AO**: AO does not natively construct, parse, or return the canonical `WorkerReport` structure via its session execution result (`NATIVE_AO_WORKERREPORT = NO`).
2. **Public Workspace File Surface Is Sufficient**: AO provides a robust public HTTP API (`GET /api/v1/sessions/{sessionId}/workspace/file?path=...`) capable of retrieving arbitrary workspace artifacts written by the worker process (`EXISTING_PUBLIC_SURFACES_SUFFICIENT = YES`).
3. **Upstream Immutability**: Neither AO nor Agy requires source patching to support the supervisor-worker loop (`AO_UPSTREAM_PATCH_REQUIRED = NO`, `AGY_UPSTREAM_PATCH_REQUIRED = NO`).
4. **Supervisor Normalization Requirement**: The handoff of structured worker execution summaries must be formalized at the Supervisor control-plane boundary without relying on private AO databases, raw terminal stream scraping, or unmanaged fixed file paths (`SUPERVISOR_NORMALIZATION_REQUIRED = YES`).

This Architecture Decision Record establishes the canonical contract for turn completion, structured report handoff, zero-trust verification, and process invocation boundaries.

---

## Decision

### 1. Turn Completion Semantics
- **Canonical Completion**: In the pinned AO + Agy integration, a worker turn completes when the native Agy Stop hook triggers AO to transition session activity state from `active` to `idle`.
- **Process Liveness**: Agy runs as a persistent interactive process/session across turns. Process exit is **NOT** required for normal turn completion.
  ```
  PROCESS_ALIVE != TURN_RUNNING
  ```
- **Decoupling from Report Readiness**: Session idleness indicates the agent process has ceased generating output; it does not guarantee a valid, complete report has been flushed to disk:
  ```
  AO_IDLE != REPORT_READY
  ```

### 2. Attempt-Scoped Report Path
- **Fixed Singleton Path Prohibited**: Production V1 strictly prohibits the static singleton file path `.supervisor/worker-report.json` to prevent race conditions, stale report ingestion across retries, and worktree pollution.
- **Canonical Attempt-Scoped Path**:
  ```
  .supervisor/reports/<task_id>/<attempt_id>.json
  ```
- **Pre-Dispatch Ownership**: Prior to dispatching a task to AO, the Supervisor assigns:
  - `task_id` (unique task identifier)
  - `attempt_id` (unique immutable opaque execution identity)
  - `attempt_number` (monotonically increasing integer within a task per ADR-012)
  - `expected_report_path` (exact destination path communicated in the task prompt)
- **Strict Identity Validation**: The worker must write its report to `expected_report_path`. On ingestion, the Supervisor enforces:
  ```
  report.task_id == expected_task_id AND report.attempt_id == expected_attempt_id
  ```
  Any report with mismatched or missing identifiers must be rejected as stale or foreign.

### 3. Report Ready State Progression
- **Canonical Lifecycle**:
  ```
  Agy Stop Hook -> AO Session IDLE -> Bounded Fetch -> JSON Parse -> Schema Validation -> Identity Match -> REPORT_READY
  ```
- **Failure Taxonomy**: If the report cannot be verified, the Task transitions to state `FAILED` with explicit semantic `failure_reason` (these are error codes, NOT workflow states):
  - `REPORT_MISSING`: Artifact does not appear within the bounded fetch window.
  - `REPORT_INVALID`: Artifact is non-JSON or violates the canonical `WorkerReport` schema.
  - `REPORT_IDENTITY_MISMATCH`: Artifact contains incorrect `task_id` or `attempt_id`.

### 4. Canonical Public Report Transport
- **Approved Surface**: All report retrieval from AO sessions must execute via the official public workspace file API:
  ```
  GET /api/v1/sessions/{sessionId}/workspace/file?path=<expected_report_path>
  ```
- **Prohibited Surfaces**:
  - Raw `/mux` terminal stream scraping
  - AO SQLite internal database queries
  - Direct calls to internal AO Go packages
  - Host GUI automation or screen inspection
  - Out-of-band direct Agy stdout capture bypassing AO
- **Upstream Patch Requirement**: None. Pinned AO `v0.13.0` public surfaces are completely sufficient.

### 5. Zero-Trust Evidence Boundary
- **Worker Claims vs. Authoritative Evidence**: `WorkerReport` content represents unverified worker assertions (`WORKER_CLAIMS`), not verified truth.
- **Strict Separation in ReviewBundle**:
  - Worker claims (`commands_run`, `tests`, `build_status`, `worker_claims`) are captured as telemetry.
  - The Supervisor independently establishes objective ground truth:
    - Base commit SHA and Head commit SHA
    - Branch name and author
    - Full Git diff and changed files list
    - Task Contract scope compliance (allowed vs. forbidden paths)
    - Trusted re-execution of test/build commands by the Supervisor host
    - Actual process exit codes, test logs, and artifact cryptographic hashes
- **Independent Re-execution Semantics**: A subsequent Supervisor re-execution of a test proves current code state validity; it does not authenticate historical worker execution claims.

### 6. Application Git Separation
- **Worktree Hygiene**: The `.supervisor/` directory tree contains runtime control-plane metadata and must never contaminate application pull requests, commit histories, or production code promotion diffs.
- **Exclusion & Durable Storage**: `.supervisor/` is excluded from application commits (via `.gitignore` or explicit staging exclusion). Durable archival of execution reports belongs exclusively to the Supervisor State Store.

### 7. Antigravity Invocation Boundary
- **No Raw Argv Passthrough**:
  ```
  NO_RAW_AGY_ARGV_PASSTHROUGH_FROM_SUPERVISOR
  ```
  AO internally constructs and owns the CLI command line for Agy (`backend/internal/adapters/agent/agy/agy.go`).
- **Supervisor Control-Plane Boundary**: The Supervisor / AOAdapter validates only control-plane inputs it directly owns prior to dispatching to AO:
  - `TaskContract` compliance
  - Project and workspace directory boundaries
  - Permission policies (`dangerously-skip-permissions`, etc.)
  - Prompt length and character constraints
  - Model selection against allowed project policy
  - Target `task_id`, `attempt_id`, and `expected_report_path`
- The Supervisor does not attempt to validate or sanitize hidden internal arguments generated downstream by AO.

### 8. Model Selection & Policy Boundary
- **Native Parameter Support**: Pinned AO exposes a top-level `model` property in `POST /api/v1/sessions` (`SpawnSessionRequest.Model`) and propagates it via `--model <Model>` during both initial launch and session restore.
- **Supervisor Policy Enforcement**: Model selection is an authorized Supervisor-controlled parameter. The Supervisor validates that `model ∈ allowed_models` before dispatch.
- **Quota & Capacity Classification**: Upstream model quota or capacity exhaustion (HTTP 429 / `RESOURCE_EXHAUSTED`) is operational telemetry only. It does not constitute an architecture failure or restore defect.

### 9. Conversation Continuity vs. Execution Identity
- **Native Agy Conversation ID**: Represents conversational context memory and session continuity across turns.
- **Supervisor Attempt ID**: Represents unique task attempt, audit trail, and review bundle identity.
- **Decoupling**: A task retry, revision, or re-execution gets a new, unique `attempt_id` and new report path, even if it restores the same existing native Agy conversation.

### 10. Bounded Report Fetch Window
- Upon detecting session activity transition to `IDLE`, the Supervisor polls `GET /api/v1/sessions/{id}/workspace/file` using bounded retry with exponential backoff.
- Indefinite polling is prohibited. If the report is not retrievable within policy bounds, the task transitions to state `FAILED` with `failure_reason = REPORT_MISSING`.

---

## Alternatives Considered
- **Direct CLI Output Parsing**: Parsing worker output directly from terminal stdout. Rejected due to vulnerability to unstructured text formatting, terminal control sequences, and lack of atomic completion semantics.
- **AO Upstream Patch for Native WorkerReport**: Modifying AO to implement a native supervisor report bridge. Rejected per Phase 1 governance to avoid upstream maintenance divergence when existing public APIs are fully capable.
- **Fixed File Path (`.supervisor/worker-report.json`)**: Using a fixed path. Rejected due to high risk of cross-attempt race conditions, stale report reads, and lack of execution isolation.

---

## Consequences
- **Positive**: Clean architectural separation between control-plane orchestration (AO) and governance verification (Supervisor).
- **Positive**: Zero upstream patches required for pinned AO `v0.13.0` or Agy `1.2.7`.
- **Positive**: Zero-trust verification prevents acceptance solely on unverified worker claims.
- **Negative / Trade-off**: The Supervisor must implement bounded polling logic and artifact retrieval normalization rather than receiving synchronous report payloads in HTTP response bodies.

---

## Source Evidence
- Agent Orchestrator `v0.13.0` (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) source analysis and runtime proof (`docs/audits/P01_A_AO_RUNTIME_PROOF.md`).
- Antigravity CLI `1.2.7` (`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`) capability proof (`docs/audits/P01_B_AGY_CLI_PROOF.md`).
- P01-C empirical integration proof dossier (`docs/audits/P01_C_AO_AGY_INTEGRATION_PROOF.md`).

---

## Revisit Conditions
This decision may only be revisited if a future upstream AO version natively adopts the canonical `WorkerReport` protocol, supported by an approved ADR.
