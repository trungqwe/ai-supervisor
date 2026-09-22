# P03_TASK_003_PROPOSAL_EXTERNAL_AUDIT.md - External Supervisor Audit Record (PROPOSAL-P03-002)

> **Audited Commit**: `3c07db43d26cc2546dc3c2d67d523b3e5b3c20a2`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Audit Date**: 2026-09-22
> **External Supervisor Verdict**: `PROPOSAL_P03_002 = REVISION_REQUIRED`
> **Architecture Impact**: `P03_ARCHITECTURE_CHANGE = YES`
> **ADR Requirement**: `P03_ADR_REQUIRED = YES`
> **ADR-016 Authorization**: `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET`
> **TASK_P03_001 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_002 Status**: `EXTERNAL_AUDIT_APPROVED`
> **TASK_P03_003 Status**: `NOT_RELEASED`
> **P03 Code Status**: `HELD`
> **Active Gate**: `P03_TASK_003_PROPOSAL_REVISION_1`

---

## 0. External Supervisor Decision Summary

Independent external audit of `PROPOSAL-P03-002` (commit `3c07db43d26cc2546dc3c2d67d523b3e5b3c20a2`) is complete.
The original seven (7) pre-code gaps (`P03T3-PRE-001` through `P03T3-PRE-007`) remain valid.
However, the proposed resolutions contain ten (10) critical defects across requirement attribution, crash-window survivability, domain model alignment, activity observation proofs, task ownership boundaries, termination fail-closed rules, stop provenance, transaction atomicity, and retry policy overreach.

**Verdict**: `PROPOSAL_P03_002 = REVISION_REQUIRED`.

---

## 1. External Audit Findings Matrix

| Finding ID | Title | Verdict | Severity | Remediation Mandate |
|---|---|---|---|---|
| `P03T3PR1-001` | `AUTHORITY_MATRIX_MISATTRIBUTED` | **REVISION_REQUIRED** | Major | Correct citations: FR-005 is Worker Task Dispatch; FR-006 is Worker Lifecycle Observation; NFR-005 is Upstream Decoupling / anti-corruption adapters; bounded polling authority derives from `docs/12`, `docs/phases/P03`, approved `PROPOSAL-P03-001`, and FR-006. |
| `P03T3PR1-002` | `SESSION_SPAWN_CRASH_WINDOW_UNRESOLVED` | **REVISION_REQUIRED** | Blocker | Pinned AO `POST /api/v1/sessions` has `PINNED_SPAWN_IDEMPOTENCY = ABSENT`. Analyze crash window T1/T2/T3. If public API cannot guarantee deterministic recovery, mark `P03T3_SESSION_SPAWN_RECOVERY = UNRESOLVED_BLOCKER`. |
| `P03T3PR1-003` | `WORKER_SESSION_DOMAIN_RELATIONSHIP_DRIFT` | **REVISION_REQUIRED** | Blocker | Proposed changes to `Pair 1 -> 1 WorkerSession` and adding attempt session binding constitute an architecture change (`P03_ARCHITECTURE_CHANGE = YES`). Detail domain relationship options with explicit trade-offs and mark recommendation `GIẢ ĐỊNH`. |
| `P03T3PR1-004` | `LAST_ACTIVITY_ATTEMPT_PROOF_OVERCLAIM` | **REVISION_REQUIRED** | Blocker | Remove claims of "cryptographic proof". `lastActivityAt` does not prove attempt execution or dispatch acknowledgement. Re-evaluate missed active window with attempt-specific generation fencing or mark `MISSED_ACTIVE_WINDOW = UNRESOLVED_DECISION`. |
| `P03T3PR1-005` | `REPORT_READY_OWNERSHIP_SCOPE_VIOLATION` | **REVISION_REQUIRED** | Blocker | `AO IDLE != REPORT_READY`. TASK-P03-003 is lifecycle observation only; it must not fetch, parse, validate reports, write `worker_report_raw`, or transition `RUNNING -> REPORT_READY`. P04 EvidenceCollector owns `REPORT_READY`. |
| `P03T3PR1-006` | `AO_ACTIVITY_MAPPING_OVERCLAIM` | **REVISION_REQUIRED** | Major | Distinguish `AO_WAITING_INPUT` from `AO_BLOCKED_DECISION`. Pinned AO `ActivityExited` does not imply `isTerminated == true`. Remove unconditional mapping of `ActivityExited` to crash / `FAILED`. |
| `P03T3PR1-007` | `TERMINATION_CLASSIFICATION_NOT_FAIL_CLOSED` | **REVISION_REQUIRED** | Major | Restore `PROPOSAL-P03-001` contract: `WORKER_STOPPED`, `WORKER_CRASHED`, and `WORKER_TERMINATION_UNKNOWN`. Absence of stop event proves only `INTENTIONAL_STOP_NOT_PROVEN`, not crash. |
| `P03T3PR1-008` | `STOP_PROVENANCE_NOT_GENERATION_SAFE` | **REVISION_REQUIRED** | Major | Stop provenance must be generation-safe operation lifecycle (`STOP_REQUESTED`, `STOP_CALL_SUCCEEDED / FAILED`, `STOP_TERMINATION_CONFIRMED`) bound to attempt/generation, preventing stale stop inheritance on later restore. |
| `P03T3PR1-009` | `ATTEMPT_END_TRANSACTION_AND_PHASE_OWNERSHIP_UNRESOLVED` | **REVISION_REQUIRED** | Major | Remove `RecordAttemptEnd(..., rawReport)`. Normal attempt `ended_at` is committed atomically with `REPORT_READY` by P04. Terminal outcomes in P03 must specify atomic CAS on Task state + `ended_at` + audit append. Remove claims of active attempt transitioning to `CANCELLED`. |
| `P03T3PR1-010` | `UNAUTHORIZED_RETRY_POLICY_INVENTED` | **REVISION_REQUIRED** | Major | Remove invented retry policies ("retry up to max attempts", "recovery threshold", fixed retry counts). Define fail-closed behavior for AO daemon unavailable without numeric assumptions. |

---

## 2. Governance Determination

1. `P03_ARCHITECTURE_CHANGE = YES` (amends durable domain relationships and attempt ownership not present in canonical `docs/05`).
2. `P03_ADR_REQUIRED = YES`.
3. `ADR_016 = NOT_AUTHORIZED_TO_DRAFT_YET` (must wait until Proposal Revision 1 passes external re-audit).
4. `TASK_P03_003 = NOT_RELEASED`.
5. `P03_CODE = HELD_FOR_TASK_P03_003_PRECODE_RECONCILIATION`.
