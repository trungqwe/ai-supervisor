# PHASE 0 — EXTERNAL RE-AUDIT #3 FINDINGS REGISTER

> **Authority**: External Audit Record  
> **Audit Type**: Literal Evidence Hygiene  
> **Status**: Remediation applied. Further defects found by Re-Audit #4.  
> **Date**: 2026-09-20  
> **Baseline Commit**: `7c7b23917436a4ac11c77e42e2b771f5d032389d`

---

## 1. External Verdict

```text
External Re-Audit #3 Verdict: PHASE0_BLOCKED
Reason: Residual non-literal upstream symbols and headings in evidence records.
```

The architecture was accepted as stable. The evidence/provenance layer contained incorrect symbol citations, non-existent headings, and incomplete upstream verification.

---

## 2. Categories of Defects Found

External Re-Audit #3 identified defects across these categories:

1. **AO router symbols** — `setupRouter` and `registerRoutes` cited in `01_AGENT_ORCHESTRATOR.md` and `REUSE_MATRIX.md` do not exist in `router.go`. Correct symbol: `NewRouterWithControl`.
2. **AO Agy adapter symbols** — `CommandBuilder` / `BuildCommand` cited in `01_AGENT_ORCHESTRATOR.md` and `REUSE_MATRIX.md` do not exist. Correct symbols: `GetLaunchCommand`, `GetRestoreCommand`.
3. **Mieruko exact headings** — `## Quy tac chung cho agent` does not exist in AGENTS.md; correct heading is `## Quyen truy cap`. `## Active Agents` is incomplete; correct heading is `## Active Agents and session recovery`. `## Tool catalog overview` does not exist; correct heading is `## MCP tools`. `createAdminRouter` signature was hand-reconstructed inaccurately.
4. **AIWorkHub exact Markdown structure** — `## Five quality control layers` does not exist; correct heading is `## Acceptance pipeline`. `## Manager acceptance` is an abbreviated heading; correct form is item `5. Manager acceptance and integration proof`.
5. **Symphony REUSE_MATRIX headings** — `## 7. State Machine` is abbreviated; correct is `## 7. Orchestration State Machine`. `## 9. Workspace Management` is abbreviated; correct is `## 9. Workspace Management and Safety`.
6. **Premature final audit supersession** — `PHASE0_EVIDENCE_FINAL_AUDIT.md` claimed all evidence was correct, but multiple non-literal citations remained.
7. **Literal evidence exactness rule** — No documented rule prevented hybrid signatures or abbreviated headings in the Exact Source Area column.

---

## 3. Remediation Applied (Re-Audit #3)

The initial Re-Audit #3 remediation (commit `bbebaf0`) corrected only items 1 and 2 (AO symbols). Items 3-7 were identified but not fully corrected until Re-Audit #4.

| Blocker | File Modified | Incorrect Value | Corrected Value |
|---|---|---|---|
| AO router | `01_AGENT_ORCHESTRATOR.md`, `REUSE_MATRIX.md` | `setupRouter / registerRoutes` | `NewRouterWithControl`, `mountHealth` |
| AO adapter | `01_AGENT_ORCHESTRATOR.md`, `REUSE_MATRIX.md` | `CommandBuilder / BuildCommand` | `GetLaunchCommand`, `GetRestoreCommand` |

---

## 4. Remaining Defects (Deferred to Re-Audit #4)

Items 3-7 required a second pass (Re-Audit #4) because:
- PowerShell double-quoted here-strings corrupted Markdown with control characters.
- Mieruko, AIWorkHub, and Symphony dossier headings were not independently re-verified.
- The PHASE0_EVIDENCE_FINAL_AUDIT.md was not properly superseded.
- No evidence rule prevented future hybrid-signature errors.

See `PHASE0_LITERAL_EVIDENCE_FINAL_AUDIT.md` for the complete corrected baseline.
