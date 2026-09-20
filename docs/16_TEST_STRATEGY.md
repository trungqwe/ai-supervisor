# 16. TEST STRATEGY

> **Focus**: Multi-Tier Testing, Upstream Verification & Full Task Loop End-to-End Test  
> **Status**: Approved Baseline (Remediated Phase 0)

---

# 1. Verification Levels & Test Taxonomy

```mermaid
flowchart TD
    Unit[Level 1: Domain Unit Tests] --> Schema[Level 2: JSON Schema Validation]
    Schema --> Upstream[Level 3: Upstream Runtime Proofs P01-A & P01-B]
    Upstream --> Adapter[Level 4: AO-Agy Adapter Proof P01-C]
    Adapter --> Transport[Level 5: ChatGPT Transport Feasibility P01-D]
    Transport --> Integration[Level 6: Evidence & Review Integration Tests]
    Integration --> E2E[Level 7: Full Loop End-to-End Acceptance Test]
```

### Test Tiers:
1. **Level 1 — Unit Tests**: Fast, in-memory domain tests verifying entity invariants, state machine transitions (`DRAFT` → `DISPATCHED` → `IN_PROGRESS` → `REPORTED` → `REVIEWING` → `APPROVED` / `REVISION_REQUIRED`), and policy bounds.
2. **Level 2 — Schema Validation**: Validating that all generated TaskContracts, WorkerReports, and ReviewBundles conform to formal JSON schemas (`docs/schemas/`).
3. **Level 3 — Upstream Runtime Proofs (P01-A & P01-B)**:
   - **Track P01-A (AO Runtime)**: Testing actual loopback endpoints (`GET /healthz`, `GET /readyz`, `POST /api/v1/projects`, `POST /api/v1/sessions`, `POST /api/v1/sessions/{id}/send`, `POST /api/v1/sessions/{id}/kill`, `POST /api/v1/sessions/{id}/restore`).
   - **Track P01-B (Antigravity CLI Runtime)**: Smoke tests running `agy 1.2.7` on Windows verifying `--print`, `--output-format stream-json`, `--input-format stream-json`, `--json-schema`, `--add-dir`, and `--conversation`.
4. **Level 4 — AO ↔ Agy Adapter Integration Proof (P01-C)**:
   - Tracing AO's invocation argv for Agy.
   - Determining session termination detection and WorkerReport delivery gap.
   - Producing verified verdict: `YES`, `NO`, or `PARTIAL`.
5. **Level 5 — ChatGPT Transport Feasibility Proof (P01-D / FR-016)**:
   - Proving that the target ChatGPT Plus account can invoke Supervisor tools without OS-level GUI automation and without manual prompt/report copy-paste.
   - Validating loopback security, token scrubbing, and approval prompts.
6. **Level 6 — Evidence Collection & Review Integration Tests**:
   - Independent Git diff extraction from worktree.
   - Process exit code verification.
   - Scope compliance verification against `allowed_scope`.
   - Unified Review Bundle generation in `< 3.0` seconds.
7. **Level 7 — Full Loop E2E Acceptance Test**:
   The critical end-to-end supervision workflow:
   ```text
   ChatGPT Dispatch (create_task_contract)
     ↓
   Supervisor Contract Validation (PolicyEngine)
     ↓
   AO Session Spawn & Task Delivery (AOAdapter)
     ↓
   Antigravity Code Edit & Test in Isolated Worktree
     ↓
   Worker Completion Notification (get_worker_status)
     ↓
   Independent Git Evidence Collection (diff, exit code, hashes)
     ↓
   Unified Review Bundle Generation (get_review_bundle)
     ↓
   ChatGPT Audit & Supervisor Decision (approve_task / request_revision)
     ↓
   State Recorded as APPROVED (Zero automatic merge / zero auto-commit)
   ```

---

# 2. Gate Conditions

- **Phase P01 Exit Gate**: Tri-state verdict model (`PASS`, `GAP_REQUIRES_ADR`, `BLOCKER`). Phase P02 cannot proceed if Track P01-D yields `BLOCKER`.
- **Phase P04 Evidence Gate**: No task may transition to `REVIEWING` unless independent Git evidence has been captured and validated.
- **Phase P05 Tool Gate**: Tool count exposed to ChatGPT must not exceed 12 high-level domain tools; low-level OS/process implementation details (such as worker PID) are hidden behind the domain boundary.
- **Approval Safety Gate**: Supervisor `approve_task` solely records the review decision and updates task state to `APPROVED`. Automatic branch merging or pushing is strictly forbidden in V1.
