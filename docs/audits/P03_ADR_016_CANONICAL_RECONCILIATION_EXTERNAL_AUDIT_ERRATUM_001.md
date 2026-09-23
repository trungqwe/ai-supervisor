# P03 ADR-016 Canonical Specification Reconciliation External Audit Erratum 001

**Erratum Target**: `docs/audits/P03_ADR_016_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT.md` (Section 4: "Pinned Upstream Wire Reality Boundary: /restore vs /resume-agent")
**Target Audited Commit**: `41cf769b4cf8a980c95de3aa4be63140b04922c3`
**Referenced Upstream Commit**: `Untrivial-ai/agent-orchestrator` commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`
**Authority**: External Supervisor
**Date**: 2026-09-23
**Status**: APPROVED_ERRATUM

---

## 1. Purpose & Scope of Erratum

This document records a formal citation erratum for line number references in Section 4 of `docs/audits/P03_ADR_016_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT.md`.

During independent verification of pinned `Untrivial-ai/agent-orchestrator` commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`, exact line numbers for route definitions and controller handlers were verified via `git show`. The historical audit record cited provisional line numbers from an earlier draft revision. This erratum corrects the citations to match the byte-for-byte reality of commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.

> [!NOTE]
> This erratum is strictly a **citation correction**.
> The substantive wire facts, OpenAPI specifications, controller behavior, and the External Supervisor audit verdict (`P03_ADR_016_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`) remain completely **unchanged and fully valid**.

---

## 2. Line Number Citation Corrections

The line citations in Section 4 of `docs/audits/P03_ADR_016_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT.md` are corrected as follows:

| Endpoint / Symbol | Target File in AO Pinned Commit `15e9ea971` | Provisional Citation in Audit Record | Verified Exact Line Number (`git show`) | Verified Content at Pinned Commit |
|---|---|---|---|---|
| `/restore` Route Spec | `backend/internal/httpd/apispec/openapi.yaml` | Line 186 | **Line 5245** | `  /api/v1/sessions/{sessionId}/restore:` |
| `restoreSession` Operation | `backend/internal/httpd/apispec/openapi.yaml` | (Unnumbered) | **Line 5247** | `      operationId: restoreSession` |
| `/restore` Route Registration | `backend/internal/httpd/controllers/sessions.go` | Line 61 | **Line 195** | `r.Post("/sessions/{sessionId}/restore", c.restore)` |
| `restore` Handler Implementation | `backend/internal/httpd/controllers/sessions.go` | Line 183 | **Line 1250** | `func (c *SessionsController) restore(w http.ResponseWriter, r *http.Request)` |
| `/resume-agent` Route Spec | `backend/internal/httpd/apispec/openapi.yaml` | Line 214 | **Line 5284** | `  /api/v1/sessions/{sessionId}/resume-agent:` |
| `resumeAgent` Operation | `backend/internal/httpd/apispec/openapi.yaml` | (Unnumbered) | **Line 5286** | `      operationId: resumeAgent` |
| `/resume-agent` Route Registration | `backend/internal/httpd/controllers/sessions.go` | Line 62 | **Line 197** | `r.Post("/sessions/{sessionId}/resume-agent", c.resumeAgent)` |
| `resumeAgent` Handler Implementation | `backend/internal/httpd/controllers/sessions.go` | Line 216 | **Line 1289** | `func (c *SessionsController) resumeAgent(w http.ResponseWriter, r *http.Request)` |

---

## 3. Governance Status & Invariants

1. **Audit Record Preservation**: The original audit document `docs/audits/P03_ADR_016_CANONICAL_RECONCILIATION_EXTERNAL_AUDIT.md` remains unaltered as historical evidence, accompanied by this erratum.
2. **Canonical Reconciliation Verdict**: Remains **`P03_ADR_016_CANONICAL_RECONCILIATION = EXTERNAL_AUDIT_APPROVED`**.
3. **Wire Fact Reaffirmation**:
   - `POST /api/v1/sessions/{sessionId}/restore` (`operationId: restoreSession`) restores a terminated session and worktree.
   - `POST /api/v1/sessions/{sessionId}/resume-agent` (`operationId: resumeAgent`) resumes an agent within an active session without restoring workspace or terminated session.
