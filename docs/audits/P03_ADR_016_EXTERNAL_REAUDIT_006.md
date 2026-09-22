# P03 ADR-016 External Re-Audit 006 Report

**Audit Target**: `docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md` (Revision 6)
**Audited Commit**: `40d51ffe9af4dab5545f060d5bee2ffd441b6109`
**Authority**: External Supervisor
**Status**: EXTERNAL_AUDIT_APPROVED
**Date**: 2026-09-23

---

## 1. Executive Summary & Git Verification Evidence

External Supervisor Re-Audit 006 was performed directly on remote commit `40d51ffe9af4dab5545f060d5bee2ffd441b6109`.

### Git Evidence:
- **Audited Commit SHA**: `40d51ffe9af4dab5545f060d5bee2ffd441b6109`
- **Branch Synchronization**: `HEAD` = `origin/main` at `40d51ffe9af4dab5545f060d5bee2ffd441b6109`
- **Working Tree**: Clean (`git status --porcelain` returns empty)
- **Diff Whitelist**: Diff from parent commit `4caf9e6e1890a737475196f0ef00369431a58a28` strictly and exclusively modifies `docs/adr/ADR-016-durable-dispatch-session-binding-and-lifecycle-reconciliation.md`
- **Diff Stat**: `1 file changed, 23 insertions(+), 11 deletions(-)`
- **Format Hygiene**: `git diff --check` passed with zero errors or whitespace warnings
- **Zero Scope Contamination**: Zero production code modifications (`internal/**/*.go`), zero schema or database migration changes, zero canonical specification alterations, and zero proposal drift.

---

## 2. Technical Evaluation & Decision Order Verification

Re-Audit 006 verified the surgical resolution of the contradiction between **D11** (`WORKER_STOPPED_ALLOWED_IFF`) and **D13** (startup recovery sweep):

### A. Pinned AO Wire Contract Boundary & Absence of Absolute Exit Causality
- Pinned `Untrivial-ai/agent-orchestrator` v0.13.0 wire contract (`GET /api/v1/sessions/{id}`) returns only high-level status strings (`active`, `idle`, `waiting_input`, `blocked`, `exited`), `isTerminated` boolean, and opaque `terminal_generation` (`PUBLIC_CRASH_EVIDENCE_AVAILABLE_IN_PINNED_BASELINE = NONE`).
- Upstream exposes **no process exit timestamp, exit code, or OS signal**.
- Absolute causal proof (whether process death was strictly caused by `/kill` versus autonomous worker exit or concurrent host crash) is wire-unverifiable.
- ADR-016 Revision 6 truthfully restricts evidence to a **lineage-bounded temporal window** (`now < confirmation_deadline_at`) under verified kill acceptance (`STOP_CALL_SUCCEEDED`) and matching execution generation, strictly prohibiting exit timestamp speculation.

### B. Precise Disposition for the Two Observation Timings

| Criterion | Case 1: Termination Observed Before Deadline (`now < confirmation_deadline_at`) | Case 2: Termination First Observed At/After Deadline (`now >= confirmation_deadline_at`) |
| :--- | :--- | :--- |
| **Upstream Evidence** | HTTP 200, matching `terminal_generation`, `isTerminated == true`, observed when `now < confirmation_deadline_at` | HTTP 200, matching `terminal_generation`, `isTerminated == true`, first observed when `now >= confirmation_deadline_at` |
| **Exit Timing Speculation** | Not required; observation lies strictly inside the lineage-bounded temporal window | **Strictly PROHIBITED**; cannot speculate if exit occurred prior to deadline |
| **`WORKER_STOPPED_ALLOWED_IFF`** | **MET** (all 6 conditions satisfied) | **NOT MET** (Condition 6 failed: `now < confirmation_deadline_at` is false) |
| **Stop Operation `stage`** | Advances to `STOP_TERMINATION_CONFIRMED` | Retains `STOP_CALL_SUCCEEDED` |
| **Stop Operation `resolution_state`** | `TERMINATION_CONFIRMED` | `STOP_CONFIRMATION_TIMEOUT` |
| **`task_attempts.recovery_disposition`** | `WORKER_STOPPED` | `STOP_CONFIRMATION_TIMEOUT` |
| **TaskState Effect (Live Stop)** | `RUNNING -> FAILED` (reason: `WORKER_STOPPED`) | `RUNNING -> FAILED` (disposition: `STOP_CONFIRMATION_TIMEOUT`) |
| **`WORKER_STOPPED` Permitted?** | **YES** | **NO** (Fail-closed) |
| **Attempt Quarantine Effect** | **CLEARED** (Pair lane is clean for subsequent dispatch) | **REMAINS ACTIVE** (operator resolution required) |

### C. D13 Synchronous Startup Recovery Sweep Decision Order
D13 Subsection 3 establishes an unambiguous, deterministic priority order:
1. `HTTP 404 (Target Absent)` -> `stage = 'STOP_TARGET_ABSENT'`, `resolution_state = 'STOP_TARGET_ABSENT'`; quarantine remains ACTIVE.
2. `Generation Mismatch` -> retain `stage = 'STOP_CALL_SUCCEEDED'`, `resolution_state = 'STOP_GENERATION_MISMATCH'`; quarantine remains ACTIVE.
3. `Same Generation and isTerminated == true`:
   - If `now < confirmation_deadline_at`: advance to `STOP_TERMINATION_CONFIRMED` (`TERMINATION_CONFIRMED`), record `WORKER_STOPPED` for live stop, release quarantine.
   - If `now >= confirmation_deadline_at`: retain `STOP_CALL_SUCCEEDED`, advance to `STOP_CONFIRMATION_TIMEOUT`; fail closed, `WORKER_STOPPED` strictly prohibited, quarantine remains ACTIVE.
4. `Same Generation and Alive`:
   - If `now < confirmation_deadline_at`: remain in `STOP_CALL_SUCCEEDED` (`IN_FLIGHT`), continue bounded observation until deadline.
   - If `now >= confirmation_deadline_at`: retain `STOP_CALL_SUCCEEDED`, set `resolution_state = 'STOP_CONFIRMATION_TIMEOUT'`, quarantine remains ACTIVE.

---

## 3. Findings Disposition

| Finding ID | Title | Severity | Disposition in Re-Audit 006 |
| :--- | :--- | :--- | :--- |
| **ADR16R6-001** | Blocked Escalation Crash Window Unrecovered (D13 startup recovery misses BLOCKED) | CRITICAL | **CLOSED** (verified: `# BLOCKED_WITH_OPEN_CURRENT_ATTEMPT MANDATORY_ESCALATION_RECOVERY` in D10, D12, D13) |
| **ADR16R6-002** | D12 Escalation Atomicity Not Reconciled (BLOCKED -> HUMAN_REQUIRED transaction missing) | HIGH | **CLOSED** (verified: `AtomicAttemptClosureTransition` defined in D12) |
| **ADR16R6-003** | WORKER_STOPPED Provenance Invariant Weakness & Causality Boundary | HIGH | **CLOSED** (verified: 6 verifiable conditions in `WORKER_STOPPED_ALLOWED_IFF`, truth table, and exit timestamp speculation prohibited) |
| **ADR16R6-004** | Provisioning Guard Predicate Reversed (`stage NOT IN` vs `stage IN`) | HIGH | **CLOSED** (verified: D6 Item 4 checks zero unresolved operations via `stage IN`) |
| **ADR16R4-008** | Cross-Section Vocabulary Drift | MEDIUM | **CLOSED** (verified: undeclared token `PRE_SEND_BLOCKED` eliminated from matrix) |
| **ADR16R5-001** | Revision Identity & Governance Drift | HIGH | **CLOSED** (verified: synchronized across ADR-016, AGENTS.md, and CURRENT_STATE.md) |
| **ADR16R5-002** | Worker Report Integrity | CRITICAL (PROCESS) | **CLOSED** |

---

## 4. Formal Verdict

```text
PROPOSAL_P03_002 = EXTERNAL_APPROVED

ADR_016 = EXTERNAL_APPROVED
ADR_016_ACCEPTANCE = GRANTED

TASK_P03_003 = NOT_RELEASED
P03_CODE = HELD_PENDING_CANONICAL_RECONCILIATION_AND_TASK_CONTRACT

ACTIVE_GATE = CANONICAL_SPEC_RECONCILIATION_PLANNING
```

### Approved Next Action:
Pursuant to `docs/24_CHANGE_GOVERNANCE.md` (Level 2 Approved ADR takes precedence over Level 3 Canonical Architecture and Level 4 Specifications), formulate the scope of canonical specification reconciliation according to approved ADR-016. Zero canonical specification mutations, Task Contract creation, schema migrations, or production coding are authorized until reconciliation scope is approved.
