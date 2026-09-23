# TASK-P03-003C administrative stop clarification decision

- Authority: External Supervisor decision in the 2026-09-23 task instruction, confirming `BLOCKER-3C-001` and approving the exact Class B/C semantics recorded in `docs/adr/ADR-016-CLARIFICATION-administrative-stop-risk-acceptance.md`.
- Prior release artifact: `7dc3732a5a3cea58800ae3a370fcee9dd64ac742`; code base: `246f74eabd09bbc2ea66624ee217035167824a86`.
- Decision: `BLOCKER-3C-001=CLOSED_AT_DESIGN_LEVEL`; `TASK_P03_003C_IMPLEMENTATION=IN_PROGRESS`. No WIP code or AC-3C-01..12 is approved by this decision.
- Scope: audit event `ADMINISTRATIVE_RISK_ACCEPTED`, Class B/C CAS and provenance, linked order stop reconciliation → Tx D → D6. No schema, wire or TaskState change.
- Contract: the released JSON already requires Class B/C authority, evidence and audit within the same allowed scope and AC. The clarification supplies the missing approved audit literal and does not revise scope or AC; `TASK_CONTRACT_P03_003C.md` remains immutable.
- Evidence limit: this is a design decision record, not a claim that implementation tests or host authentication passed. The historical blocker record remains unchanged.
