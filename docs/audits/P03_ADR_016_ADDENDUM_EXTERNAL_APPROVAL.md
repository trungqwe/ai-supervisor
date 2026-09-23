# P03 ADR-016 Addendum — External Supervisor design approval

- **Audited SHA**: `3807343f06f99c0e11460c74522a2efb3be19491` (draft addendum).
- **Verdict**: `ADR_016_ADDENDUM_DESIGN = EXTERNAL_APPROVED`; `ADD-R1-001`, `ADD-R1-002`, `ADD-R1-003 = CLOSED`.
- **Authority**: External Supervisor decision in this task. The formal artifact is `docs/adr/ADR-016-ADDENDUM-restore-and-pre-send-semantics.md`; historical draft, accepted ADR-016, proposals and audits are unchanged.

| Finding | Design evidence at audited SHA | Decision |
|---|---|---|
| ADD-R1-001 | Addendum §3 admission predicate rejects every open attempt, unresolved `SEND_REQUESTED` even after closure and inconsistent current pointer; clean historical closed dispatch is not a permanent lock. | CLOSED |
| ADD-R1-002 | §5 retains exact 3A stop lineage for old generation; new runtime generation uses linked `PAIR_MAINTENANCE` without attempt IDs, operator provenance and separate per-lineage clearance. | CLOSED; D11 extension approved |
| ADD-R1-003 | §6 hold transition table and exact-disposition CAS prevent timeout or GET from downgrading `PRE_SEND_PROTOCOL_UNVERIFIED`; `RecordSendRequested` rejects unresolved hold. | CLOSED |

Design evidence includes the audited draft SQL predicate, guard matrix, transaction and acceptance tables. Prior focused SQLite admission probes and existing 3A lineage test supported review; neither is proof of v4 implementation. No restore was executed on a live AO session. DDL, authorization/restore transitions and dispatch semantics are **design approved**; migration, runtime behavior, host authentication and P04 runner remain **unverified**.

Host/bootstrap integration must deliver an authenticated operator principal to the trusted boundary before operator restore or linked stop can be enabled. 3B may implement/test the fail-closed interface. `AUTOMATIC_RESTORE=DISABLED`; 3B/3C/3D contracts remain `NOT_RELEASED`. `DESIGN_BLOCKER_3B_RESTORE_PROTOCOL` is closed by this addendum, distinct from runtime enablement. `DESIGN_BLOCKER_3D_STARTUP_WIRING` remains preserved. Canonical reconciliation and 3B candidate are ready for separate External Supervisor audit, not approved implementation.

## Handover verification

The formal v4 SQL fence compiled in SQLite with foreign keys enabled against minimal `pairs`, `audit_events`, and `stop_operations` prerequisites (exit `0`); this is a DDL syntax probe, not a migration test. A temporary Go harness called `CompileSchema`, `ParseAndValidateRaw`, and `TaskContractValidator.ValidateRaw` on both draft and candidate JSON with the approved `go-test-p03-003b` catalog: `CwdPolicy=worktree_root`, `MaxTimeoutSeconds=300`, object parameter schema with `additionalProperties=false`, required `package`/`flags`, package enum `./internal/dispatch/...`, `./internal/store/...`, `./...`, and exactly two unique flags from `-v`/`-race`. Both positive validations passed; `timeout_seconds=301`, `-exec`, package `./internal/ao/...`, and `cwd=../escape` were rejected for both documents (`go run p03_contract_check_tmp.go`, exit `0`). The temporary harness was removed. Semantic validation does not prove a P04 runner, executable binding or process isolation.
