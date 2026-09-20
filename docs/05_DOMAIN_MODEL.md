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
        +int attempt_number
        +string task_id
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
        +TaskContract contract
        +WorkerClaim claims
        +Evidence evidence
        +PolicyFinding[] findings
        +datetime generated_at
    }

    class ReviewDecision {
        +string decision_id
        +string task_id
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
    Task "1" o-- "0..*" ReviewBundle
    Task "1" o-- "0..*" ReviewDecision
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
3. **Task & TaskContract**:
   - `Task` manages lifecycle and attempts. `TaskContract` defines the static work specification.
   - *Invariant*: Once dispatched, `TaskContract` is completely immutable (`is_immutable == true`).
4. **WorkerClaim vs. Evidence**:
   - `WorkerClaim`: Self-reported statements from the worker process.
   - `Evidence`: Verified facts collected directly from Git and OS process execution logs by the Supervisor.
   - *Invariant*: Evidence cannot be written or modified by the worker.
5. **ReviewBundle**:
   - The compiled data structure delivered to ChatGPT for auditing.
   - *Invariant*: Must include both worker claims and independent evidence side-by-side with policy findings.
