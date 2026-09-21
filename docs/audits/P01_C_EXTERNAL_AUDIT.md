# P01-C -- EXTERNAL SUPERVISOR AUDIT DOSSIER

> **Authority**: External Supervisor Independent Audit Authority
> **Status**: APPROVED_WITH_ADR_011
> **Verdict**: P01_C_EXTERNAL_AUDIT = APPROVED_WITH_ADR_011
> **Date**: 2026-09-21
> **Repository**: `D:\TU_CODE\ai-supervisor`
> **Canonical Scope Specification**: `docs/phases/P01_UPSTREAM_PROOF.md`
> **Audited Baseline**: `404ae2f8bca48fc2bb3a3ac8c77779249afd0b3f`
> **Prior Track Milestone (P01-B)**: `85837fe5f27d75aec0e0996b60c03a26b2464856`
> **Historical Frozen Baseline**: Tag `phase0-architecture-v1` (`6f72eaca30be3fc3ac00f25829dd4283ed98c3f5`)
> **Pinned AO**: `Untrivial-ai/agent-orchestrator` `v0.13.0` (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Pinned Agy**: `google-antigravity/antigravity-cli` `1.2.7` (`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`)
> **Architecture Status**: `ARCHITECTURE_V2 = CANDIDATE_READY_FOR_FREEZE_REVIEW`

---

# 1. External Supervisor Audit Verdict

The External Supervisor has independently re-audited the empirical findings of Track P01-C against the canonical requirements in `docs/phases/P01_UPSTREAM_PROOF.md` and approves Track P01-C with Architecture Decision Record **ADR-011**.

```text
P01_C_EXTERNAL_AUDIT = APPROVED_WITH_ADR_011
P01_C_CANONICAL_RUNTIME = PASS
P01_C_RUNTIME_INTEGRATION = EMPIRICALLY_PROVEN_ON_TARGET_WINDOWS
NATIVE_AO_WORKERREPORT = NO
EXISTING_PUBLIC_SURFACES_SUFFICIENT = YES
SUPERVISOR_NORMALIZATION_REQUIRED = YES
AO_UPSTREAM_PATCH_REQUIRED = NO
AGY_UPSTREAM_PATCH_REQUIRED = NO
P01C_AO_ONLY_HIDDEN_MARKER_RESTORE_TEST = SUPPLEMENTAL_DEFERRED

P01-A = EXTERNAL_AUDIT_APPROVED
P01-B = EXTERNAL_AUDIT_APPROVED
P01-C = GAP_RESOLVED_BY_ADR_PENDING_PHASE_P01_FINAL_AUDIT
P01-D = TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED

P01 = READY_FOR_EXTERNAL_FINAL_AUDIT
ARCHITECTURE_V2 = CANDIDATE_READY_FOR_FREEZE_REVIEW
ACTIVE_GATE = EXTERNAL_SUPERVISOR_P01_FINAL_AUDIT
```

---

# 1.1 Historical WorkerReport Schema Finding & Handoff Transport Distinction

External audit re-evaluation of the historical P01-C WorkerReport sample identified that while top-level field presence passed, the nested `tests` array used generic `{ "name": "verify.ps1", "status": "PASSED" }` entries rather than the schema-required `test_suite`, `passed`, and `failed` integer properties from `docs/schemas/worker-report.schema.json`.

```text
HISTORICAL_P01C_REPORT_FIELD_PRESENCE = PASS
HISTORICAL_P01C_REPORT_SCHEMA_VALIDATION = FAIL
HISTORICAL_P01C_REPORT_TRANSPORT = PASS
```

**Interpretation**:
1. **Public Transport Validated**: AO public workspace file API successfully transported the worker artifact with HTTP 200 OK.
2. **Worker Artifact Invalid**: Under production ADR-011 semantics, this artifact would be rejected as `REPORT_INVALID`.
3. **Architecture Conclusion Preserved**: This finding reinforces that `WORKER_CLAIMS != TRUSTED_INPUT` and demonstrates the necessity of Supervisor-side schema validation established in ADR-011. It does not invalidate the P01-C architectural conclusion.

---

# 2. Canonical Scope Reconciliation

1. **Canonical P01-C Requirements**: Pursuant to the original canonical Phase P01 specification (`docs/phases/P01_UPSTREAM_PROOF.md`), the scope of Track P01-C is strictly defined as:
   - Tracing live AO invocation command-line arguments for Agy.
   - Detecting agent turn completion reliably via native hooks/state transitions.
   - Evaluating whether the existing AO Agy adapter can satisfy `WorkerReport` requirements without custom upstream source modifications.
   - Returning an explicit integration verdict supported by concrete evidence.
2. **Supplemental Hidden-Marker Restore Test**: The end-to-end hidden-marker context-recall test via AO was introduced as an additional validation probe, not as a mandatory Phase P01 exit criterion.
3. **Formal Scope Determination**:
   - `P01C_AO_AGY_CONTEXT_RESTORE_E2E_MARKER = SUPPLEMENTAL_NOT_PROVEN`
   - `P01C_AO_AGY_CONTEXT_RESTORE_E2E_MARKER_BLOCKS_P01 = NO`
   - Re-execution loops driven by transient provider quota exhaustion are permanently halted.

---

# 3. Accepted Compositional Restore Evidence

The empirical viability of session restoration and conversational context retention across the AO ↔ Agy stack is established compositionally through verified, non-redundant evidence across Tracks P01-A, P01-B, and P01-C:

| Verification Layer | Empirical Fact / Proof | Status |
|---|---|---|
| **Layer A (P01-A)** | AO session restore lifecycle endpoint (`POST /api/v1/sessions/{id}/restore`) operates successfully on target Windows host; spawns restored process tree. | **`EMPIRICALLY_PROVEN`** |
| **Layer B (P01-B)** | Direct Antigravity CLI cross-process continuation with `--conversation <id>` reliably preserves conversational memory and accurately recalls hidden context markers. | **`EMPIRICALLY_PROVEN`** |
| **Layer C1 (P01-C)** | AO session manager captures native Agy session UUID from initialization output (`AO_RESTORE_NATIVE_ID_PROPAGATION`). | **`EMPIRICALLY_PROVEN`** |
| **Layer C2 (P01-C)** | AO restore lifecycle command line explicitly appends `--conversation <same-native-id>` to restored process argv (`P01C_RESTORE_ARGV_TRACE`). | **`EMPIRICALLY_PROVEN`** |
| **Layer C3 (P01-C)** | Restored AO session receives and acknowledges subsequent lifecycle traffic via `POST /send` (`AO_RESTORED_SESSION_MESSAGE_DELIVERY`). | **`EMPIRICALLY_PROVEN`** |
| **Supplemental E2E** | Full hidden-marker recall exclusively through AO `POST /send` without direct CLI queries (`FULL_HIDDEN_MARKER_AO_ONLY_E2E`). | **`SUPPLEMENTAL_NOT_PROVEN`** (Non-blocking) |

---

# 4. Attempt Chronology & Reclassification

1. **Attempt 1 Preserved Telemetry**:
   - Dispatched fresh marker `P01C_AO_RESTORE_1790007240247` into Turn 1; session reached `idle` at 138s via Stop hook.
   - Worktree scan verified zero marker leaks (`RESTORE_MARKER_WORKSPACE_LEAK = NO`).
   - Terminated via `POST /kill` (200), restored via `POST /restore` (200). Process argv verified with native conversation ID.
   - Turn 2 prompt sent via `POST /send` (200). Restored worker inference never executed due to upstream model capacity exhaustion (`RESOURCE_EXHAUSTED` / 429).
   - Canonical classification: `P01C_TARGETED_RESTORE_ATTEMPT_1 = INCONCLUSIVE_TRANSIENT_PROVIDER_FAILURE`.
   - It is operational telemetry only and does not constitute negative evidence against AO restore correctness.
2. **Round-2 Scope Disposition**:
   - Because canonical Phase P01 exit gates do not require the supplemental hidden-marker restore test, Round 2 was **NOT EXECUTED** (`P01C_TARGETED_RESTORE_ROUND_2 = NOT_EXECUTED`).
   - Planned test procedures are strictly decoupled from empirical evidence.
3. **Model Capacity & Quota Telemetry**:
   - `QUOTA_POLICY = NON_BLOCKING_OPERATIONAL_CONCERN`.
   - All model routes tested during the observed capacity window returned `RESOURCE_EXHAUSTED` under the active authenticated Agy environment.
   - Quota investigation loops are permanently halted for Phase P01.

---

# 5. Process Hygiene Deviations

| Deviation ID | Designation | Classification | Description & Corrective Directive |
|---|---|---|---|
| **P01C-DEV-001** | `MISMATCH_EXPECTED_SKIPPED_CLASSIFICATION` | `CLASSIFICATION_CORRECTION` | Build status mismatch in fixture test was expected due to absence of independent build phase; corrected from defect to expected skip. |
| **P01C-DEV-002** | `INDEPENDENT_EVIDENCE_SEMANTICS_CORRECTION` | `SEMANTICS_CORRECTION` | Clarified that worker claims (`commands_run`, `tests`) are assertions; independent re-execution proves current state, not historical worker execution. |
| **P01C-DEV-003** | `PRIVATE_PROVIDER_SESSION_TRANSCRIPT_INSPECTION` | `PROCESS_HYGIENE_DEVIATION` | Inspected provider internal session transcript during Attempt 1 root-cause diagnosis. Security incident: NO; evidence validity: PRESERVED; architecture impact: NONE. Re-inspection permanently prohibited. |
| **P01C-DEV-004** | `GLOBAL_AGY_IMAGE_NAME_TERMINATION` | `PROCESS_HYGIENE_DEVIATION` | Broad `taskkill /F /IM agy.exe` was executed during process cleanup troubleshooting. Security incident: NO EVIDENCE; architecture impact: NONE; evidence impact: NONE. Permanent rule: future test cleanups must use exact verified PIDs and process ancestry; global image name termination is strictly prohibited. |

---

# 6. Architectural Gap Resolution via ADR-011

The empirical finding that AO does not natively construct or return `WorkerReport` is formally accommodated by **ADR-011** (`docs/adr/ADR-011-worker-report-handoff-and-agy-invocation-boundary.md`):
1. **Turn Completion**: Native Agy Stop hook transitions session to `IDLE`. Process termination is not required (`PROCESS_ALIVE != TURN_RUNNING`). `AO_IDLE != REPORT_READY`.
2. **Attempt-Scoped Report Path**: Reports are addressed as `.supervisor/reports/<task_id>/<attempt_id>.json` to prevent race conditions and enforce attempt identity (`report.task_id == expected_task_id && report.attempt_id == expected_attempt_id`).
3. **Report Ready Progression**: Progression from `IDLE` through bounded fetch, schema validation, and identity matching to `REPORT_READY`. Failure classes: `REPORT_MISSING`, `REPORT_INVALID`, `REPORT_IDENTITY_MISMATCH`.
4. **Public Surface Transport**: Retrieved via public REST endpoint `GET /api/v1/sessions/{id}/workspace/file?path=...`. Zero upstream patches to AO or Agy required.
5. **Zero-Trust Verification**: Worker claims are strictly segregated from independent Supervisor-gathered Git diffs, test logs, exit codes, and artifact hashes in `ReviewBundle`.
6. **Git Hygiene**: `.supervisor/` directory tree is strictly separated from application source code.
7. **Invocation Boundary**: `NO_RAW_AGY_ARGV_PASSTHROUGH_FROM_SUPERVISOR`. AO owns internal Agy command construction. Supervisor validates control-plane inputs.
8. **Model Policy**: Supervisor validates `model ∈ allowed_models`; provider quota is operational telemetry only.
9. **Identity Separation**: Native conversation ID (context continuity) is decoupled from Supervisor attempt ID (execution audit identity).
10. **Bounded Fetch**: Supervisor uses bounded retry with exponential backoff for report retrieval.

---

# 7. Final Phase P01 Posture

With all four proof tracks approved (P01-A, P01-B, P01-C with ADR-011, P01-D), Phase P01 is formally **READY FOR EXTERNAL FINAL AUDIT**. Architecture V2 Candidate is transitioned to **CANDIDATE_READY_FOR_FREEZE_REVIEW**.
