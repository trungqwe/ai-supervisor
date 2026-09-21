# P01-A -- EXTERNAL SUPERVISOR AUDIT DOSSIER

> **Authority**: External Supervisor Independent Audit Authority
> **Status**: APPROVED
> **Verdict**: P01_A_EXTERNAL_AUDIT = APPROVED
> **Date**: 2026-09-21
> **Repository**: `D:\TU_CODE\ai-supervisor`
> **Audited Baseline Commit**: `283287e1bbc46412f42ae4110ef8fec1ea00abc2`
> **Prior Execution Baseline**: `fea488a5410cef5557160516302953c4ad9b6468`
> **Historical Frozen Baseline**: `phase0-architecture-v1` (`6f72eaca30be3fc3ac00f25829dd4283ed98c3f5`)
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN`

---

# 1. External Supervisor Audit Verdict

The External Supervisor has independently audited the empirical runtime evidence submitted in commit `283287e1bbc46412f42ae4110ef8fec1ea00abc2` and documented in `docs/audits/P01_A_AO_RUNTIME_PROOF.md`.

The underlying empirical proof of Untrivial Agent Orchestrator (`v0.13.0`, commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) running natively on the target Windows host is **accepted and approved**.

```text
P01_A_RUNTIME = PASS
P01_A_EXTERNAL_AUDIT = APPROVED
AO_RUNTIME = EMPIRICALLY_PROVEN_ON_TARGET_WINDOWS
P01-B = READY
P01-C = HELD
P01-D = TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED
ARCHITECTURE_V2 = CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN
ACTIVE_GATE = P01_B_RUNTIME_EXECUTION_AUTHORIZED
```

---

# 2. Required Literal Evidence Corrections & Verification

The External Supervisor identified seven documentation/evidence corrections to align the audit dossier precisely with the execution transcript and upstream contracts. These corrections were applied without altering the underlying empirical proof results:

### A. Project Registration Request
- **Finding**: Dossier previously formatted the project creation body with explicit `id` and `name` fields.
- **Correction**: Dossier updated to reflect the literal request actually executed:
  ```json
  {
    "path": "D:\\TU_CODE\\_ai_supervisor_p01a_ao_runtime\\fixture-repo"
  }
  ```

### B. Project Registration Response Status
- **Finding**: Controller `backend/internal/httpd/controllers/projects.go` (`add` function) returns `http.StatusCreated` (HTTP 201) on successful project creation.
- **Correction**: Corrected `POST /api/v1/projects` response status from HTTP 200 OK to **HTTP 201 Created**. (`GET /api/v1/projects/{id}` remains HTTP 200 OK).

### C. Session Spawn Request Transcript
- **Finding**: Dossier previously omitted the `prompt` field from the spawn request snippet.
- **Correction**: Dossier updated to show the complete literal payload executed:
  ```json
  {
    "projectId": "fixture-repo",
    "harness": "agy",
    "prompt": "Harmless lifecycle test probe. Do not modify files.",
    "displayName": "p01a-probe"
  }
  ```

### D. Path-Guard Architecture & Evidence Layering
- **Finding**: Path containment operates across distinct architectural layers; test attribution required precision.
- **Correction**: Clarified the two layers:
  1. *API Workspace Path Guard*: Evaluated at HTTP layer, rejecting `../../outside.txt` and `C:\Windows\System32` with HTTP 400 (`INVALID_WORKSPACE_PATH`).
  2. *Gitworktree Managed-Root Guard*: Implemented in `backend/internal/adapters/workspace/gitworktree/workspace.go` via `managedPath` and `validateManagedPath`. Verified via upstream test suite:
     `go test -v -run "TestManagedPathSafety|TestValidateConfigRejectsPathEscapingIDs" ./internal/adapters/workspace/gitworktree` (100% PASS).

### E. Acceptance Count Canonicalization
- **Finding**: Counting summary was ambiguous ("21/21 including cleanup").
- **Correction**: Standardized to canonical accounting:
  - 21 Numbered Runtime/Provenance Checks = PASS
  - P01A_CLEANUP Gate = PASS
  - Total Observed Validations = 22

### F. Architecture Status Deduplication
- **Finding**: `docs/18_CURRENT_STATE.md` contained two duplicate Architecture Status entries.
- **Correction**: Deleted the stale duplicate entry. Retained single canonical row:
  `Architecture Status = ARCHITECTURE_V2_CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN`.

### G. Scope Statement Bounding
- **Finding**: Scope text in executive summary was overbroad ("delivering all architectural primitives claimed in ADR-002, ADR-003, and ADR-004").
- **Correction**: Bounded strictly to empirical proof scope:
  "empirically proving the P01-A scoped AO runtime primitives required by the current architecture baseline on the target Windows host."

---

# 3. Status Transition & Release Authorization

With the approval of this External Audit and verification of the correction commit:
1. **Track P01-A** is formally marked **`EXTERNAL_AUDIT_APPROVED`**.
2. **Track P01-B** (Direct Antigravity CLI Capability Proof) is released from `HELD` to **`READY`**, and runtime execution in disposable sandbox `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\` is **AUTHORIZED**.
3. **Track P01-C** remains **`HELD`**.
4. **Architecture V2** remains strictly **`CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN`** (no architecture freeze is authorized until upstream runtime proofs P01-A, P01-B, and P01-C complete).
