# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL RE-AUDIT 011

> **Authority**: External Supervisor Governance Gate Audit
> **Audited Commit SHA**: `b5a1d8ef0c38e52ed71370a1e0dfa17e0d3e5f2d`
> **Preservation Baseline Commit**: `6e1993da150031a9465901a7019c71257de44312` (Revision 11)
> **Audited Artifacts**:
> - `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md` (Revision 12)
> - `docs/proposals/PROPOSAL-P04-002-review-bundle-latency-semantics.md` (Revision 6)
> - `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md` (Revision 12)
> - `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md` (Revision 12)
> - `docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_REAUDIT_010.md`
> - `AGENTS.md`
> - `docs/18_CURRENT_STATE.md`

---

## 1. Executive Summary & Verdict

The External Supervisor audited the remediation deliverables submitted for Pre-Contract Architecture Round 11 at commit `b5a1d8ef0c38e52ed71370a1e0dfa17e0d3e5f2d`.

Audit Assessment of Round 11 Findings:
- `P04-ARCH-R11-001`: **CLOSED_AT_DESIGN_LEVEL** (Interval correctly designated as ReviewBundle assembly diagnostic, NFR-008 compliance status held as UNVERIFIED, post-commit process telemetry decoupled from atomic audit events).
- `P04-ARCH-R11-002`: **NOT_CLOSED** (In-place row mutation `EXPIRED -> RECLAIMED` failed to achieve an append-only linear lease history; fork prevention and predecessor immutability required structural redesign).
- `P04-ARCH-R11-003`: **PARTIALLY_CLOSED** (Blanket transition removed, but durable integrity holds were missing, and audit event idempotency lacked durable mapping to canonical `audit_events.event_id`).
- `P04-ARCH-R11-004`: **SUBSTANTIVELY_CLOSED_WITH_R12_FOLLOWUP** (Limit inequality and 4 stream states adopted, but integer typing and overflow guards required DDL enforcement).

Four new findings (`P04-ARCH-R12-001` through `P04-ARCH-R12-004`) are formally recorded requiring Revision 13 remediation:
1. `P04-ARCH-R12-001`: Restoration of all normative architecture from Revision 11 lost during whole-document regeneration (Git allowlist, AppContainer handle/job controls, TOCTOU matrix, ReviewBundle replay matrix, Schema v6 opaque `terminal_generation TEXT`, `canonical_worktree_path`, `created_at_epoch_ms`, `released_at_epoch_ms`, exact lineage guards, WorkerClaim payload validation).
2. `P04-ARCH-R12-002`: Linear lease chain design with immutable predecessors (no `RECLAIMED` state, predecessor stays `EXPIRED`, `predecessor_lease_id TEXT NULL UNIQUE`, first lease token=1 & pred NULL, successor token=pred.token+1, single active lease partial index, immutable `released_at_epoch_ms`, authoritative process-death proof requirement).
3. `P04-ARCH-R12-003`: Durable integrity hold mechanism (`review_integrity_holds` table, exact lineage, ACTIVE/RESOLVED lifecycle, partial unique index, fail-closed on active hold, deterministic audit `event_id` mapping).
4. `P04-ARCH-R12-004`: Integer typing and overflow guards (`typeof(col) = 'integer'`, int64 overflow guards, hard limit < MaxInt64 - 1).

### Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_12_REQUIRED`
- `PROPOSAL_P04_002 = REVISION_6_REQUIRED`
- `ADR_018 = REVISION_12_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_12_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_12`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`
- `AUTOMATIC_RESTORE = DISABLED`

---

## 2. Review of Round 11 Findings (P04-ARCH-R11-001 through R11-004)

| Finding ID | Title | Status | Audit Assessment |
| :--- | :--- | :--- | :--- |
| `P04-ARCH-R11-001` | NFR-008 Interval Naming & Post-Commit Telemetry | **CLOSED_AT_DESIGN_LEVEL** | Interval correctly designated ReviewBundle assembly diagnostic; NFR-008 status strictly UNVERIFIED; telemetry decoupled from Tx C. |
| `P04-ARCH-R11-002` | Append-Only Lease History & CAS Reclaim Guards | **NOT_CLOSED** | In-place mutation `EXPIRED -> RECLAIMED` on predecessor row violated append-only principles and allowed forking risks (`P04-ARCH-R12-002`). |
| `P04-ARCH-R11-003` | TaskState Invariant Protection & Failure Audit | **PARTIALLY_CLOSED** | Blanket non-REVIEWING rule eliminated, but durable integrity holds and deterministic audit event_id mapping were missing (`P04-ARCH-R12-003`). |
| `P04-ARCH-R11-004` | Stream Limit Inequality & Precedence Matrix | **SUBSTANTIVELY_CLOSED_WITH_R12_FOLLOWUP** | Inequality `hard > capture` and 4 stream states adopted; integer typing and int64 overflow guards required (`P04-ARCH-R12-004`). |

---

## 3. New Findings (P04-ARCH-R12-001 through P04-ARCH-R12-004)

### `P04-ARCH-R12-001`: Restoration of All Normative Architecture from Revision 11
- **Severity**: Critical
- **Description**: Whole-document rewrites in Revision 12 inadvertently dropped critical normative sections from Revision 11:
  1. Hardened Git invocation allowlist and isolated Git execution environment.
  2. Clean index/worktree verification policy.
  3. Immutable snapshot extraction via `git ls-tree -rz --full-tree` and `git cat-file --batch`.
  4. TOCTOU & crash matrix.
  5. Windows AppContainer handle inheritance whitelisting (`PROC_THREAD_ATTRIBUTE_HANDLE_LIST`), atomic Job Object assignment, `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, no breakaway, DACL and network denial (`NoNetwork`).
  6. ReviewBundle replay and conflict state matrix.
  7. Schema v6 binding columns: `contract_id` composite FK, `terminal_generation TEXT` (must remain opaque string, strictly equality-compared, never INTEGER), `canonical_worktree_path`, `created_at_epoch_ms`, `released_at_epoch_ms`, exact lineage guards.
  8. WorkerClaim payload validation against canonical worker report schema properties (`claimed_files_changed`, `tests`, `textual_claims`).
- **Remediation Directive**: Restore Revision 11 as the normative baseline and apply surgical patches only. Provide a preservation matrix mapping all Revision 11 sections to Revision 13.

### `P04-ARCH-R12-002`: Linear Lease Chain Design with Truly Immutable Predecessors
- **Severity**: High
- **Description**: In-place mutation `EXPIRED -> RECLAIMED` modified historical lease rows and risked forks.
- **Remediation Directive**:
  1. Eliminate the `RECLAIMED` state entirely. Predecessor leases remain permanently `EXPIRED`.
  2. Add column `predecessor_lease_id TEXT NULL UNIQUE REFERENCES task_verification_leases(lease_id)`.
  3. First lease: `predecessor_lease_id IS NULL` and `fencing_token = 1`.
  4. Successor lease: `predecessor_lease_id IS NOT NULL`, predecessor must match `(task_id, attempt_id, contract_id)`, predecessor state must be `EXPIRED`, and `NEW.fencing_token = predecessor.fencing_token + 1`.
  5. Enforce `predecessor_lease_id != lease_id` and zero predecessor reuse/forking via `UNIQUE(predecessor_lease_id)`.
  6. `released_at_epoch_ms` is written strictly once during `ACTIVE -> terminal` CAS and cannot be rewritten.
  7. Transaction B must CAS exact `(lease_id, worker_id, fencing_token, state='ACTIVE')`.
  8. Reclaim requires authoritative process-death proof: joined Job Object/process handle within daemon or post-restart P03 host exclusivity + `KILL_ON_JOB_CLOSE`.

### `P04-ARCH-R12-003`: Durable Integrity Hold Mechanism & Audit Event Idempotency
- **Severity**: High
- **Description**: Invariant mismatches and intake rejections lacked durable tracking in the database, and audit idempotency lacked a schema column mapping.
- **Remediation Directive**:
  1. Introduce Schema v9 `review_integrity_holds` table: exact lineage, reason enum, ACTIVE/RESOLVED lifecycle, partial unique index ensuring at most one ACTIVE hold per attempt, immutable identity/reason, and resolution strictly by human supervisor authority.
  2. All intake, verification (Tx B), compilation (Tx C), and approval paths must fail-closed when an ACTIVE hold exists on the attempt.
  3. Startup recovery must scan and load ACTIVE holds before opening admission.
  4. Holds and rejection audit events must be recorded inside a single diagnostic transaction executed after rollback.
  5. Map audit idempotency deterministically to canonical `audit_events.event_id` (`event_id = SHA256(lineage + ":" + reason + ":" + sanitized_input_fingerprint)`). Duplicate exact events are recognized as replay; matching ID with differing payload is an integrity conflict.

### `P04-ARCH-R12-004`: Integer Typing Constraints & Overflow Guards
- **Severity**: Medium
- **Description**: Numeric columns in SQLite without explicit type checking risk coercing to REAL or overflowing 64-bit signed integers.
- **Remediation Directive**:
  1. Add `CHECK (typeof(col) = 'integer')` for all epoch timestamps, TTLs, byte limits, byte counts, and fencing tokens.
  2. Enforce `acquired_at_epoch_ms <= (9223372036854775807 - (ttl_seconds * 1000))`.
  3. Enforce `fencing_token <= 9223372036854775807`.
  4. Enforce `hard_safety_limit_bytes <= 9223372036854775806` (ensuring `hard + 1` cannot overflow int64).
  5. Retain the 4 stream states and precedence matrix from Revision 12.

---

## 4. Remediation Instructions

1. Remediate `PROPOSAL-P04-001` to Revision 13 using Revision 11 as normative baseline.
2. Remediate `PROPOSAL-P04-002` to Revision 7.
3. Remediate `DRAFT-ADR-018` to Revision 13 using Revision 11 as normative baseline.
4. Remediate `PLAN-P04-EVIDENCE-REVIEW` to Revision 13 using Revision 11 as normative baseline.
5. Update `AGENTS.md` and `docs/18_CURRENT_STATE.md` to reflect `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_12`.
6. Maintain `AUTOMATIC_RESTORE = DISABLED`, zero Go code, zero contract release.
