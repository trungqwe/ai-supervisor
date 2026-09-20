# 09. WORKER REPORT CONTRACT

> **Focus**: Standardized Execution Output, Claims vs. Ground Truth & Auditability  
> **Status**: Approved Baseline

---

# 1. The Core Verification Rule

> [!CRITICAL]
> **THIS REPORT CONTAINS CLAIMS, NOT VERIFIED TRUTH.**  
> The Supervisor Control Plane treats all data in this report as candidate information. The Supervisor independently collects Git diffs, head SHAs, and test exit codes before compiling the Review Bundle.

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
