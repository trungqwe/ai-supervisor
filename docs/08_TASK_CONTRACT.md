# 08. TASK CONTRACT SPECIFICATION

> **Focus**: Immutable Work Unit, Scope Boundaries, Test Mandates & Schema Definition  
> **Status**: Approved Baseline

---

# 1. Purpose & Contractual Invariants

A **Task Contract** is the authoritative specification dispatched to an AI coding worker. Unlike conversational prompts, a Task Contract is a rigorous, structured agreement with two absolute invariants:

1. **Immutability Post-Dispatch**:
   - Once a task transitions to `DISPATCHED`, its fields can never be altered.
   - If scope changes are needed, the task must be aborted or marked `BLOCKED`, returning authority to the Supervisor.
2. **Explicit Scope Enforcement**:
   - Every contract declares `allowed_scope` (whitelist of glob patterns).
   - Any file touched outside `allowed_scope` or inside `forbidden_scope` triggers an automatic `PolicyViolation` during verification.

---

# 2. Canonical Field Specification

| Field Name | Type | Required | Description |
|---|---|---|---|
| `task_id` | string (UUID/URN) | YES | Globally unique task identifier (e.g., `TASK-P01-001`). |
| `phase_id` | string | YES | The roadmap phase to which this task belongs (e.g., `P01`). |
| `objective` | string | YES | Clear, unambiguous goal of the implementation task. |
| `requirements` | array of strings | YES | Traceable requirement IDs addressed (e.g., `["FR-001", "FR-002"]`). |
| `architecture_refs` | array of strings | YES | Pointers to canonical design docs (e.g., `["docs/04_ARCHITECTURE.md#section-2"]`). |
| `base_sha` | string (Git SHA) | YES | The base commit from which the worker must branch/execute. |
| `allowed_scope` | array of globs | YES | Whitelist of directory/file patterns the worker is authorized to edit. |
| `forbidden_scope` | array of globs | YES | Blacklist of paths the worker is forbidden from touching (e.g., `docs/adr/*`). |
| `constraints` | array of strings | YES | Architectural rules (e.g., "No new external dependencies"). |
| `acceptance_criteria` | array of strings | YES | Objective conditions required for task completion. |
| `required_tests` | array of strings | YES | Exact CLI test commands that must be executed and pass. |
| `required_evidence` | array of strings | YES | Expected artifacts (e.g., `["git_diff", "test_exit_code_zero", "coverage_report"]`). |
| `worker_profile` | string | YES | Target worker harness profile (e.g., `antigravity-standard`). |
| `report_contract` | string | YES | Reference to expected worker report format (`worker-report.schema.json`). |
| `stop_conditions` | array of strings | YES | Explicit conditions under which worker must immediately halt and report `BLOCKED`. |

---

# 3. Handling Blockers and Scope Deviations

If during implementation the worker discovers:
- Architectural gaps requiring design changes;
- Inability to satisfy tests without modifying forbidden scope;
- Missing external tools or environment conflicts;

The worker **MUST NOT** make unilateral decisions or expand its scope. It must halt immediately, emit a `WorkerReport` with `status: "BLOCKED"`, and state the exact blocking rationale.
