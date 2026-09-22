# CONTRACT-TASK-P03-DOC-RECONCILIATION-01

> **Contract ID**: `CONTRACT-TASK-P03-DOC-RECONCILIATION-01`  
> **Task ID**: `TASK-P03-DOC-RECONCILIATION-ADR-016`  
> **Revision Number**: `1`  
> **Supersedes Contract ID**: `null`  
> **Phase ID**: `P03`  
> **Base SHA**: `906d5969347773bd7cf5bbe273b34fd4f549efce`  
> **Status**: `DISPATCHED_AND_ACTIVE`  
> **Authority**: Issued pursuant to External Supervisor approval of `PLAN-P03-CANONICAL-RECONCILIATION-ADR-016.md` at commit `906d5969347773bd7cf5bbe273b34fd4f549efce` with mandatory pinned AO restore route correction.

---

## 1. Objective
Execute single-pass canonical specification reconciliation across items S1 through S9 (`docs/04`, `docs/05`, `docs/06`, `docs/08`, `docs/12`, `docs/14`, `docs/22`, `docs/phases/P03_AO_INTEGRATION.md`, and `docs/21`) pursuant to accepted [ADR-016](../adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md) and approved plan [PLAN-P03-CANONICAL-RECONCILIATION-ADR-016.md](../plans/PLAN-P03-CANONICAL-RECONCILIATION-ADR-016.md).

> [!CRITICAL]
> This contract authorizes **DOCUMENTATION RECONCILIATION ONLY**.
> Production coding remains strictly **`HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT`**.
> Production task contract `TASK-P03-003` is strictly **`NOT_RELEASED`**.

---

## 2. Requirements & Architecture Traceability
- **Traceable Requirements**: `FR-005` (Worker Dispatch), `FR-006` (Worker Observation), `FR-015` (Upstream Health), `OPS-002` (Clean Termination), `NFR-003` (Recoverability), `NFR-005` (Upstream Decoupling).
- **Governing Architecture References**:
  - `docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md` (accepted, immutable)
  - `docs/plans/PLAN-P03-CANONICAL-RECONCILIATION-ADR-016.md` (approved)
  - `docs/24_CHANGE_GOVERNANCE.md` (Level 2 ADR authority)

---

## 3. Scope Boundaries

### Allowed Scope (`allowed_scope`):
```text
docs/04_ARCHITECTURE.md
docs/05_DOMAIN_MODEL.md
docs/06_WORKFLOW_STATE_MACHINE.md
docs/08_TASK_CONTRACT.md
docs/12_UPSTREAM_INTEGRATION.md
docs/14_FAILURE_RECOVERY.md
docs/22_MODULE_PROVENANCE.md
docs/phases/P03_AO_INTEGRATION.md
docs/21_TRACEABILITY_MATRIX.md
docs/plans/PLAN-P03-CANONICAL-RECONCILIATION-ADR-016.md
AGENTS.md
docs/18_CURRENT_STATE.md
docs/tasks/TASK_CONTRACT_P03_CANONICAL_RECONCILIATION_ADR_016.md
```

### Forbidden Scope (`forbidden_scope`):
```text
internal/**/*.go
migrations/**
docs/adr/ADR-016-*.md
docs/proposals/**
docs/02_REQUIREMENTS.md
docs/17_ROADMAP.md
* (any file not explicitly enumerated in allowed_scope)
```

---

## 4. Architectural Constraints
1. **Zero Production Code**: Absolutely no creation or alteration of production Go code (`internal/**/*.go`).
2. **Zero Schema Migrations**: Absolutely no SQLite DDL execution or migration script release.
3. **ADR-016 Immutability**: Accepted ADR-016 is permanently frozen and must not be altered.
4. **Preserve Canonical Counts**: Maintain strictly `TASK_STATE_COUNT = 13`, `DOMAIN_TRANSITION_COUNT = 22`, and `CANONICAL_GRAPH_EDGE_COUNT = 25`. Zero new TaskStates or transition edges.
5. **Preserve Unset Operational Policies**: Maintain all eight operational timeout policies as `VALUE = UNSET`; zero invented numeric defaults.
6. **Preserve S10 and S11**: `docs/02_REQUIREMENTS.md` and `docs/17_ROADMAP.md` remain strictly unmutated.
7. **Task Contract Production Guard**: Production Task Contract `TASK-P03-003` remains `NOT_RELEASED`.

---

## 5. Acceptance Criteria
1. **S1 (`docs/04_ARCHITECTURE.md`)**: Decouple Pair provisioning from task dispatch sequence; reflect 3-stage dispatch saga (`DISPATCH_BOUND` -> `SEND_REQUESTED` -> `SEND_CONFIRMED`); document verified `POST /api/v1/sessions/{id}/kill` and `POST /api/v1/sessions/{id}/restore`.
2. **S2 (`docs/05_DOMAIN_MODEL.md`)**: Formalize Model A persistence relationship; add `quarantine_state` to `WorkerSession` and `TaskAttempt`; add aggregate entities `PairProvisioningOperation`, `DispatchOperation`, and `StopOperation` matching ADR-016 §27 DDL verbatim; document `# ONE_TASK_ATTEMPT = ONE_DISPATCH_OPERATION` and `CREATE_NEW_WORKER_SESSION_ALLOWED_IFF`.
3. **S3 (`docs/06_WORKFLOW_STATE_MACHINE.md`)**: Update transition rules for `DISPATCHED` fail-closed quarantine, `RUNNING` observation rules (`waiting_input`, `MISSED_ACTIVE_WINDOW`, `AO_BLOCKED_DECISION`, stop purpose mapping), `BLOCKED -> HUMAN_REQUIRED` atomic attempt closure (`AO_BLOCKED_ESCALATED`), and `FAILED -> READY` retry gate.
4. **S4 (`docs/08_TASK_CONTRACT.md`)**: Document execution identity snapshot binding in Section 1 Invariant 3; document attempt closure upon blocker escalation in Section 3.
5. **S5 (`docs/12_UPSTREAM_INTEGRATION.md`)**: Document `resumeWorker` mapping to verified `POST /api/v1/sessions/{sessionId}/restore`; document nullable `worktree_path`; document `/kill` wire contract limitations; document `AOWorkerStatus` fields and strict pre-send admissibility whitelist (`idle`, `waiting_input` only).
6. **S6 (`docs/14_FAILURE_RECOVERY.md`)**: Align REC-002 with verified pinned restore route; align REC-004 with purpose-aware stop lifecycle; replace REC-014 with synchronous 5-step startup recovery sweep; add Section 1.2 detailing Double-Gated Quarantine and Class A, B, C clearance playbooks.
7. **S7 (`docs/22_MODULE_PROVENANCE.md`)**: Clean `AOAdapter` endpoints; add Section 3 registering new store entities with anti-reinvention justifications.
8. **S8 (`docs/phases/P03_AO_INTEGRATION.md`)**: Reconcile deliverables and exit gate with decoupled provisioning, durable dispatch saga, and double-gated quarantine.
9. **S9 (`docs/21_TRACEABILITY_MATRIX.md`)**: Update rows `FR-005` and `OPS-002` to reference `ADR-016` and reflect durable dispatch saga and purpose-aware stop lifecycle.
10. **Clean Diff**: `git diff --check` passes with zero errors.

---

## 6. Stop Conditions
The agent must immediately halt execution and report `BLOCKED` if:
1. An unavoidable contradiction is discovered between ADR-016 decisions that cannot be resolved from the approved ADR text;
2. Satisfying an acceptance criterion requires modifying a file in `forbidden_scope`;
3. Verification checks reveal literal drift against ADR-016 §6 Token Registry or §27 Authoritative DDL.
