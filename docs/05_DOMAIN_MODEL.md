# 05. DOMAIN MODEL SPECIFICATION

> **Focus**: Ubiquitous Language, Aggregates, Entities, Value Objects & Domain Relationships
> **Status**: Approved Baseline

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
        +string objective
        +string[] requirements
        +string[] allowed_scope
        +string[] forbidden_scope
        +string[] required_tests
        +string base_sha
        +boolean is_immutable
    }

    class TaskAttempt {
        +string attempt_id
        +int attempt_number
        +string task_id
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
    Task "1" *-- "1" TaskContract
    Task "1" *-- "1..*" TaskAttempt
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
   - `Task` manages overall task lifecycle and attempt history.
   - `TaskContract` defines the static, immutable work specification (`is_immutable == true`). It does **not** contain transient execution identities such as `attempt_id`.
   - `TaskAttempt` represents a single execution, retry, or revision iteration.
   - *Invariants*:
     - `attempt_id` is unique within the system.
     - Canonical report path derives deterministically from `task_id` and `attempt_id`: `.supervisor/reports/<task_id>/<attempt_id>.json`.
     - Review of an attempt binds strictly to that attempt's claims and evidence; an attempt cannot consume claims or evidence from another attempt.
4. **WorkerClaim vs. Evidence**:
   - `WorkerClaim`: Self-reported statements from the worker process, bound to `attempt_id`.
   - `Evidence`: Verified facts collected directly from Git and OS process execution logs by the Supervisor, bound to `attempt_id`.
   - *Invariant*: Evidence cannot be written or modified by the worker.
5. **ReviewBundle & ReviewDecision**:
   - `ReviewBundle` is attempt-scoped (`attempt_id`) and compiles the immutable contract, worker claims, independent Git/test evidence, policy findings, and review focus.
   - `ReviewDecision` is explicitly bound to both `task_id` and `attempt_id`, preventing review decisions from becoming ambiguous across revision cycles.
