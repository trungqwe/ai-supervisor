# 08. TASK CONTRACT SPECIFICATION

> **Focus**: Immutable Work Unit, Scope Boundaries, Revision Lineage & Schema Definition
> **Status**: Approved Baseline (Updated Architecture V2.1 / ADR-012)

---

# 1. Purpose & Contractual Invariants

A **Task Contract** is the authoritative specification dispatched to an AI coding worker. Unlike conversational prompts, a Task Contract is a rigorous, structured agreement with three absolute invariants:

1. **Immutability Post-Dispatch (ADR-010, ADR-012)**:
   - Once a contract transitions to `DISPATCHED`, that specific `contract_id` can never be altered in place.
   - If revisions are required (`REVISION_REQUIRED -> READY`), a NEW `TaskContract` revision is created with incremented `revision_number` and `supersedes_contract_id` pointing to the previous revision.
   - The historical contract remains immutable evidence.
2. **Explicit Scope Enforcement**:
   - Every contract declares `allowed_scope` (whitelist of glob patterns).
   - Any file touched outside `allowed_scope` or inside `forbidden_scope` triggers an automatic `PolicyViolation` during verification.
3. **Separation of Specification from Execution (ADR-010, ADR-012, ADR-016)**:
   - A Task Contract specifies *what* to do and within what boundaries; it **never** contains transient execution identities such as `attempt_id`, `session_id`, or `terminal_generation`.
   - Execution iterations are tracked independently via `TaskAttempt` under Model A persistence: execution identity is captured exclusively in relational snapshot columns on `task_attempts` (`session_id`, `terminal_generation`, `recovery_disposition`, `quarantine_state`) and in the associated `dispatch_operations` record (`UNIQUE(attempt_id)`). The concept of attempt execution identity is strictly explanatory and does NOT introduce new contract fields or properties.
   - The `TaskContract` entity and its JSON schema remain permanently immutable, containing zero execution or runtime fields.

---

# 2. Canonical Field Specification

| Field Name | Type | Required | Description |
|---|---|---|---|
| `contract_id` | string | YES | Unique immutable contract revision identifier (pattern: `^CONTRACT-[A-Za-z0-9_-]+$`, e.g., `CONTRACT-TASK-P01-001-01`). |
| `task_id` | string | YES | Stable logical task identifier (pattern: `^TASK-[A-Za-z0-9_-]+$`). |
| `revision_number` | integer | YES | Monotonically increasing revision number within task (`1, 2, 3...`). Initial is `1`. |
| `supersedes_contract_id` | string / null | NO | Reference to the immediate previous contract revision (`null` for revision 1). |
| `phase_id` | string | YES | The roadmap phase to which this task belongs (e.g., `P01`). |
| `objective` | string | YES | Clear, unambiguous goal of the implementation task. |
| `requirements` | array of strings | YES | Traceable requirement IDs addressed (e.g., `["FR-001", "FR-002"]`). |
| `architecture_refs` | array of strings | YES | Pointers to canonical design docs (e.g., `["docs/04_ARCHITECTURE.md#section-2"]`). |
| `base_sha` | string (Git SHA) | YES | The base commit from which the worker must branch/execute. |
| `allowed_scope` | array of globs | YES | Whitelist of directory/file patterns the worker is authorized to edit. |
| `forbidden_scope` | array of globs | YES | Blacklist of paths the worker is forbidden from touching (e.g., `docs/adr/*`). |
| `constraints` | array of strings | YES | Architectural rules (e.g., "No new external dependencies"). |
| `acceptance_criteria` | array of strings | YES | Objective conditions required for task completion. |
| `verification_requests` | array of objects | YES | Structured verification requests referencing host-owned profiles with typed parameters (ADR-013). |
| `required_evidence` | array of strings | YES | Expected artifacts (e.g., `["git_diff", "test_exit_code_zero", "coverage_report"]`). |
| `worker_profile` | string | YES | Target worker harness profile (e.g., `antigravity-standard`). |
| `report_contract` | string | YES | Reference to expected worker report format (`worker-report.schema.json`). |
| `stop_conditions` | array of strings | YES | Explicit conditions under which worker must immediately halt and report `BLOCKED`. |

---


> [!IMPORTANT]
> **Two-Stage Verification Request Validation**:
> - **Stage A (JSON Schema Structural Validation)**: `task-contract.schema.json` verifies only that `cwd` is an optional non-empty string bounded to 256 characters, and that parameters form a valid JSON object. Generic JSON Schema shape validation does **NOT** guarantee filesystem containment or safety.
> - **Stage B (P02 TaskContractValidator Semantic Validation)**: The Supervisor's `TaskContractValidator` evaluates `cwd` against the assigned workspace worktree: rejecting absolute paths, volume/UNC escapes, and directory traversal (`..`) escaping root, ensuring the resolved target path is either exactly the assigned worktree root (`.` / root directory) or a descendant contained within the assigned worktree root (`TARGET == ROOT || TARGET_IS_DESCENDANT_OF_ROOT`).
> ```
> JSON SCHEMA SHAPE VALIDATION != PATH CONTAINMENT VALIDATION
> ```

# 3. Handling Blockers and Scope Deviations

If during implementation the worker discovers:
- Architectural gaps requiring design changes;
- Inability to satisfy tests without modifying forbidden scope;
- Missing external tools or environment conflicts;

The worker **MUST NOT** make unilateral decisions or expand its scope. It must halt immediately, emit a `WorkerReport` with `status: "BLOCKED"`, and state the exact blocking rationale. The task transitions to `BLOCKED` and escalates to `HUMAN_REQUIRED`.

> [!CRITICAL]
> **Atomic Attempt Closure on Blocker Escalation (ADR-016 §16, §18)**:
> When a task transitions from `BLOCKED` to `HUMAN_REQUIRED`, the active `TaskAttempt` must be closed atomically (`ended_at = now`, `recovery_disposition = 'AO_BLOCKED_ESCALATED'`) via `AtomicAttemptClosureTransition` under invariant `# HUMAN_REQUIRED_WITH_PRIOR_EXECUTION MUST_NOT_RETAIN_RESUMABLE_OPEN_ATTEMPT`. No resumable attempt may remain dangling across human replanning or cancellation.
