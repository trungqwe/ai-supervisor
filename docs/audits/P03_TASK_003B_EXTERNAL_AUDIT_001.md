# TASK-P03-003B External Audit 001

- Verdict: `REVISION_REQUIRED`.
- Audited implementation SHA: `4b45c55d47a9bc7ede4c778e21355a476e198169`.
- Contract `base_sha`: `b8b0c95576d87677e8d48210d9838cc2f599752a`.
- `release_artifact_sha`: `333b3922a2b79a6a414003d1959060c9a72d37eb`.
- Original implementation suite: **PASS**, per External Supervisor verdict.
- Five audit probes: **FAIL**, per External Supervisor verdict; findings `3B-R1-001` through `3B-R1-005` are OPEN.

## Findings recorded

1. `3B-R1-001`: Class A physical resolution requires positive D11 stop evidence transactionally matched to restore operation, session, generation, deadline, and all relevant lineage. Operator assertion is not physical proof; Class B/C use administrative basis.
2. `3B-R1-002`: `ReservePairRestore` must CAS `WorkerSession.status=TERMINATED` with exact Pair/session/generation/quarantine and all other admission guards. Following AO HTTP 200, `GetWorkerStatus` must confirm the same identity and generation; ambiguity retains locks and cannot retry restore.
3. `3B-R1-003`: After a genuine HTTP 200 confirmation transaction rollback, reread the exact durable tuple and audit; if still `SEND_REQUESTED`/NULL, run D5 containment in a new CAS. Preserve a concurrent winner. If containment fails, return the error and leave the intent for recovery; never resend. Apply equivalent containment after invalid response when `RecordUnknownDelivery` fails.
4. `3B-R1-004`: Persist the approved audit `event_type` for provisioning requested/confirmed/failed, pre-send rejection, dispatch send requested/confirmed, and uncertain-delivery quarantine. Use `TASK_STATE_TRANSITION` only for actual TaskState edges; details do not substitute for event type.
5. `3B-R1-005`: `ClaimRestoreCleanup` and linked `STOP_REQUESTED` must atomically commit ownership CAS, stop row, and both audit events. Rollback/crash cannot leave `CLEANUP_CLAIMED` without stop intent. Preserve exact 3A attempt snapshot; new-generation unlinked-attempt `PAIR_MAINTENANCE` cannot clear old-attempt quarantine.

## Evidence limits and gate

This record captures the External Supervisor verdict and its summary. The original suite PASS and five probe FAIL results are not treated as independently reproduced in this governance commit; this audit record does not assert implementation approval, host-principal authentication, or runtime restore enablement. Remediation must supply uncached test logs, exit codes, state/audit assertions, and a whitelist diff for re-audit.

Current gate: `TASK_P03_003B_REMEDIATION`. `TASK_P03_003B_IMPLEMENTATION=REVISION_REQUIRED`; `P03_CODE=AUTHORIZED_3B_ONLY`; `AUTOMATIC_RESTORE=DISABLED`; 3C/3D remain `NOT_RELEASED`; `DESIGN_BLOCKER_3D_STARTUP_WIRING=PRESERVED`.
