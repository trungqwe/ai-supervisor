# 02. REQUIREMENTS SPECIFICATION

> **Standard**: IEEE 830-compliant requirements with atomic, traceable IDs  
> **Status**: Baseline Specification

---

# 1. Functional Requirements (FR)

| ID | Title | Description | Acceptance Criteria |
|---|---|---|---|
| **FR-001** | Project Registration | The system shall register local workspaces with unique project IDs, root paths, and canonical documentation pointers. | Project is persisted and validated against local filesystem root. |
| **FR-002** | Pair Binding | The system shall bind exactly one active project, one ChatGPT Supervisor session, and one Worker session into a managed `Pair`. | Exactly one active Pair exists per project; state is durable across restarts. |
| **FR-003** | Technical Context Access | The system shall provide the Supervisor with indexed access to project requirements, architecture, and source tree metadata. | Supervisor retrieves documentation outline and specific sections without loading full files into prompt context. |
| **FR-004** | Task Contract Creation | The system shall create immutable Task Contracts containing objective, allowed scope, forbidden scope, acceptance tests, and stop conditions. | Contract is assigned a unique `task_id` and verified against schema before dispatch. |
| **FR-005** | Worker Task Dispatch | The system shall dispatch the immutable Task Contract to the Execution Control Plane (AO) to initiate or resume worker execution. | Task state transitions from `READY` to `DISPATCHED`; AO worker session is triggered. |
| **FR-006** | Worker Lifecycle Observation | The system shall monitor worker process health, heartbeats, and status events via the AO public API. | State transitions to `RUNNING`; crashes or hangs trigger timeout and failure handlers. |
| **FR-007** | Worker Report Ingestion | The system shall ingest structured worker completion reports and extract claims, changed files, and reported test results. | Ingested report conforms to `worker-report.schema.json` and is labeled as `WorkerClaim`. |
| **FR-008** | Independent Evidence Collection | The system shall independently query Git for actual base/head SHAs, diffs, touched files, and check command exit codes. | Git evidence is collected directly from the worktree, independent of worker claims. |
| **FR-009** | Scope Violation Detection | The system shall compare touched files in the Git diff against the Task Contract `allowed_scope` and `forbidden_scope`. | Any edit outside `allowed_scope` flags a `PolicyViolation` in the review bundle. |
| **FR-010** | Review Bundle Compilation | The system shall synthesize Task Contract, Worker Claims, Git Diff, Test Logs, and Policy Findings into a single Review Bundle. | Review Bundle conforms to `review-bundle.schema.json` and reduces ChatGPT inspection tool calls. |
| **FR-011** | Supervisor Decision Ingestion | The system shall accept ChatGPT's decision: `APPROVE`, `REVISION_REQUIRED`, or `BLOCKED`. | State machine updates task status; triggers next task or initiates revision cycle. |
| **FR-012** | Revision Cycle Management | When revision is required, the system shall formulate a revision contract referencing prior unverified claims or policy violations. | Worker is reinvoked with the targeted revision contract and existing worktree. |
| **FR-013** | Audit Trail Generation | The system shall log all pair events, task transitions, evidence hashes, and review decisions to a local append-only log. | Audit log is queryable, chronological, and sanitized of secret tokens. |
| **FR-014** | Multi-Project Ready Domain | The domain model shall support multiple project records, even though V1 executes a single active Pair. | Database and domain entities isolate project records by unique `project_id`. |
| **FR-015** | Upstream Health Check | The system shall verify connectivity and API compatibility with the AO daemon before dispatching tasks. | Returns daemon status, version, and worker harness availability. |

---

# 2. Non-Functional Requirements (NFR)

| ID | Category | Description |
|---|---|---|
| **NFR-001** | Platform | The system shall run natively on Windows 10/11 64-bit environments supporting ConPTY. |
| **NFR-002** | Zero GUI Automation | The system shall not depend on OS-level mouse, keyboard, or screenshot automation. |
| **NFR-003** | Recoverability | The system shall recover workflow state without data corruption following unexpected process termination or machine restarts. |
| **NFR-004** | Auditability | All state transitions, dispatched prompts, Git SHAs, and review decisions shall be permanently auditable. |
| **NFR-005** | Upstream Decoupling | Third-party dependencies (AO, Antigravity CLI, ChatGPT transport) shall interact strictly via anti-corruption adapters. |
| **NFR-006** | Minimal Tool Surface | The ChatGPT tool interface shall expose no more than 12–15 high-level tools to optimize reasoning and context efficiency. |
| **NFR-007** | Version Pinning | All external runtimes and design sources shall have pinned versions or reference commits recorded in `third_party/SOURCE_VERSIONS.md`. |
| **NFR-008** | Performance | The Supervisor Control Plane shall generate a Review Bundle within 3 seconds of worker completion on repos up to 10,000 files. |

---

# 3. Security Requirements (SEC)

| ID | Title | Description |
|---|---|---|
| **SEC-001** | Readonly Supervisor | The Supervisor tool surface shall be strictly readonly with respect to project source code. |
| **SEC-002** | Path Containment | All file reads and operations shall be strictly contained within the registered project root directory. |
| **SEC-003** | No Arbitrary Shell | ChatGPT shall not possess tools allowing arbitrary shell command execution or raw terminal input. |
| **SEC-004** | Secret Sanitization | Audit logs and review bundles shall scrub recognized tokens, API keys, and environment passwords. |
| **SEC-005** | Immutable Scope Enforcement | The worker's execution worktree shall be monitored, and any write to protected paths shall immediately halt approval. |

---

# 4. Operational Requirements (OPS)

| ID | Title | Description |
|---|---|---|
| **OPS-001** | Single-Command Startup | The local control plane service shall start via a single documented command. |
| **OPS-002** | Clean Process Termination | Stopping the control plane service shall cleanly close open sessions without leaving orphaned worker processes. |
| **OPS-003** | Self-Contained Storage | All task state and evidence shall be stored locally without requiring external cloud accounts or paid third-party databases. |
