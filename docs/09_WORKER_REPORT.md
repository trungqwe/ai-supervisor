# 09. WORKER REPORT CONTRACT

> **Focus**: Standardized Execution Output, Claims vs. Ground Truth & Auditability
> **Status**: Approved Baseline

---

# 1. The Core Verification Rule

> [!CRITICAL]
> **THIS REPORT CONTAINS CLAIMS, NOT VERIFIED TRUTH.**
> The Supervisor Control Plane treats all data in this report as candidate information. The Supervisor independently collects Git diffs, head SHAs, and test exit codes before compiling the Review Bundle.

---

# 1.1 Transport & Handoff Boundary (ADR-011)

Per **ADR-011** (`docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`):
- **Attempt-Scoped Report Path**: Reports are addressed as `.supervisor/reports/<task_id>/<attempt_id>.json`. Static singleton paths (such as `.supervisor/worker-report.json`) are strictly prohibited in production V1.
- **Handoff Transport**: Retrieved through AO's public workspace file API (`GET /api/v1/sessions/{id}/workspace/file?path=.supervisor/reports/<task_id>/<attempt_id>.json`).
- **Turn Completion Semantics**: Agy Stop hook transitions session activity to `IDLE` (`PROCESS_ALIVE != TURN_RUNNING`; `AO_IDLE != REPORT_READY`).
- **Zero-Trust Verification**: Report fields are treated strictly as unverified worker claims. Ground truth is independently established by the Supervisor.
- **Git Separation**: `.supervisor/` files are excluded from application Git promotion.

---

# 2. Canonical Report Fields

| Field Name | Type | Required | Description |
|---|---|---|---|
| `task_id` | string | YES | Identifier of the task executed. |
| `attempt_id` | string | YES | Specific execution attempt identifier. |
| `status` | string (enum) | YES | `COMPLETED`, `BLOCKED`, or `FAILED`. |
| `branch` | string | YES | Git branch where work was performed. |
| `base_sha` | string | YES | Starting commit SHA. |
| `head_sha` | string | YES | Resulting commit SHA reported by the worker. |
| `files_changed` | array of strings | YES | List of files the worker claims to have modified. |
| `commands_run` | array of objects | YES | Commands executed with reported exit codes and stdout/stderr snippets. |
| `tests` | array of objects | YES | Detailed test execution summary (passed, failed, skipped). |
| `build_status` | string (enum) | YES | `PASSED`, `FAILED`, or `SKIPPED`. |
| `worker_claims` | array of strings | YES | Explicit claims made by the worker (e.g., "All unit tests pass"). |
| `deviations` | array of strings | NO | Any minor deviations from prompt instructions noted by the worker. |
| `assumptions` | array of strings | NO | Assumptions made during code implementation. |
| `blockers` | array of strings | NO | Details of any encountered blocker if `status == "BLOCKED"`. |
| `artifacts` | array of strings | NO | File paths of generated build artifacts, test logs, or coverage reports. |
| `ready_for_review` | boolean | YES | Worker assertion that code is ready for audit. |

---

# 2.1 Nested Object Schemas (Mandatory Invariants)

To prevent implementation drift during Phase P02, nested array items must strictly conform to `docs/schemas/worker-report.schema.json`:

### `commands_run` Item Schema
Each element in the `commands_run` array represents a process execution claimed by the worker:
| Property | Type | Required | Description |
|---|---|---|---|
| `command` | string | YES | The exact shell command string executed. |
| `exit_code` | integer | YES | The numerical process return code (0 indicates success). |
| `output_summary` | string | NO | Truncated snippet of stdout/stderr for diagnostic context. |

### `tests` Item Schema
Each element in the `tests` array represents a test suite execution summary:
| Property | Type | Required | Description |
|---|---|---|---|
| `test_suite` | string | YES | Name, path, or identifier of the test suite executed. |
| `passed` | integer | YES | Count of passing test cases in this suite. |
| `failed` | integer | YES | Count of failing test cases in this suite. |

> [!CAUTION]
> Historical proof artifacts using generic `{ "name": "...", "status": "..." }` mappings violate this schema and will fail Supervisor validation (`REPORT_INVALID`). The formal schema is authoritative.
