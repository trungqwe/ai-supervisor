# 05. DOMAIN MODEL SPECIFICATION

> **Focus**: Ubiquitous Language, Aggregates, Entities, Value Objects & Domain Relationships
> **Status**: Approved Baseline (Updated Architecture V2.1 / ADR-012)

---

# 1. Domain Entities and Value Objects

```mermaid
classDiagram
    class Project {
        +string project_id
        +string name
        +string root_path
        +string repo_url
        +datetime registered_at
    }

    class Pair {
        +string pair_id
        +string project_id
        +string current_phase_id
        +string active_task_id
        +PairState state
    }

    class SupervisorBinding {
        +string binding_id
        +string pair_id
        +string supervisor_type
        +string session_token
        +datetime bound_at
    }

    class WorkerSession {
        +string session_id
        +string pair_id
        +string runtime_type
        +string worktree_path
        +string worker_agent_id
        +WorkerStatus status
    }

    class Task {
        +string task_id
        +string phase_id
        +string pair_id
        +TaskState state
        +int current_attempt
    }

    class TaskContract {
        +string contract_id
        +string task_id
        +int revision_number
        +string supersedes_contract_id
        +string objective
        +string[] requirements
        +string[] allowed_scope
        +string[] forbidden_scope
        +VerificationRequest[] verification_requests
        +string base_sha
        +boolean is_immutable
    }

    class TaskAttempt {
        +string attempt_id
        +int attempt_number
        +string task_id
        +string contract_id
        +string expected_report_path
        +datetime started_at
        +datetime ended_at
        +string worker_report_raw
    }

    class WorkerClaim {
        +string claim_id
        +string attempt_id
        +string reported_head_sha
        +string[] claimed_files_changed
        +ClaimedTestResult[] tests
    }

    class Evidence {
        +string evidence_id
        +string attempt_id
        +string actual_base_sha
        +string actual_head_sha
        +string[] actual_files_changed
        +string git_diff_hash
        +ActualTestResult[] test_logs
        +boolean scope_verified
    }

    class ReviewBundle {
        +string bundle_id
        +string task_id
        +string attempt_id
        +TaskContract task_contract
        +WorkerClaim worker_claims
        +ActualGitEvidence actual_git_evidence
        +ActualTestEvidence actual_test_evidence
        +PolicyFinding[] policy_findings
        +string[] unverified_claims
        +string recommended_review_focus
        +datetime generated_at
    }

    class ReviewDecision {
        +string decision_id
        +string task_id
        +string attempt_id
        +DecisionType decision
        +string reviewer_rationale
        +string[] required_revisions
        +datetime decided_at
    }

    class Blocker {
        +string blocker_id
        +string task_id
        +string reason
        +string source_entity
        +datetime raised_at
    }

    class Proposal {
        +string proposal_id
        +string title
        +string author
        +ProposalStatus status
    }

    Project "1" *-- "1..*" Pair
    Pair "1" o-- "1" SupervisorBinding
    Pair "1" o-- "1" WorkerSession
    Pair "1" *-- "1..*" Task
    Task "1" *-- "1..*" TaskContract
    Task "1" *-- "1..*" TaskAttempt
    TaskAttempt "1" --> "1" TaskContract
    TaskAttempt "1" o-- "1" WorkerClaim
    TaskAttempt "1" o-- "1" Evidence
    TaskAttempt "1" o-- "0..1" ReviewBundle
    TaskAttempt "1" o-- "0..1" ReviewDecision
    Task "1" o-- "0..*" Blocker
```

---

# 2. Entity Dictionary & Invariants

1. **Project**:
   - Represents a registered codebase workspace.
   - *Invariant*: `root_path` must exist locally and contain a valid Git repository.
2. **Pair**:
   - Represents the active engineering collaboration lane between a Supervisor and Worker.
   - *Invariant*: Exactly one task may be in `DISPATCHED`, `RUNNING`, or `REVIEWING` state per Pair at any time.
3. **Task, TaskContract & TaskAttempt**:
   - `Task` manages overall task lifecycle and attempt history across revision cycles.
   - `TaskContract` defines the immutable work specification (`is_immutable == true`).
     - Each `Task` has one or more `TaskContract` revisions (`Task 1 -> 1..* TaskContract`).
     - Every revision has a unique `contract_id`, a monotonically increasing `revision_number` within the task, and an optional `supersedes_contract_id` (ADR-012).
     - Once dispatched, a `TaskContract` revision is permanently immutable. It **never** contains transient execution identities such as `attempt_id`.
   - `TaskAttempt` represents a single execution, retry, or revision iteration.
     - Each `TaskAttempt` binds to exactly one `TaskContract` revision (`contract_id`).
     - Attributes: `attempt_id` (unique opaque immutable attempt identity), `attempt_number` (monotonically increasing integer within the task), `task_id`, `contract_id`, `expected_report_path`, `started_at`, `ended_at`, `worker_report_raw`.
   - *Invariants*:
     - A `TaskAttempt` is allocated before every `READY -> DISPATCHED` transition.
     - Canonical report path derives deterministically: `.supervisor/reports/<task_id>/<attempt_id>.json`.
     - `REVISION_REQUIRED -> READY` creates a new `TaskContract` revision (`revision_number + 1`, `supersedes_contract_id`).
     - `FAILED -> READY` retry without specification changes reuses the same `contract_id` and allocates a new `TaskAttempt` upon dispatch.
4. **WorkerClaim vs. Evidence**:
   - `WorkerClaim`: Self-reported statements from the worker process, bound strictly to `attempt_id`.
   - `Evidence`: Verified facts collected directly from Git, file trees, and the trusted verification runner by the Supervisor, bound strictly to `attempt_id`.
   - *Invariant*: Evidence cannot be written or modified by the worker.
5. **ReviewBundle & ReviewDecision**:
   - `ReviewBundle` is attempt-scoped (`attempt_id`) and compiles the immutable contract revision, worker claims, independent Git/test evidence, policy findings, and recommended review focus.
   - `ReviewDecision` is explicitly bound to both `task_id` and `attempt_id`, preventing review decisions from becoming ambiguous across revision cycles.
