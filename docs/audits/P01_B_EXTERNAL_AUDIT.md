# P01-B -- EXTERNAL SUPERVISOR AUDIT DOSSIER

> **Authority**: External Supervisor Independent Audit Authority
> **Status**: REMEDIATION_APPLIED_WAITING_FOR_QUOTA
> **Verdict**: P01_B_EXTERNAL_AUDIT = REMEDIATION_APPLIED_WAITING_FOR_QUOTA
> **Date**: 2026-09-21
> **Repository**: `D:\TU_CODE\ai-supervisor`
> **Audited Baseline Commit**: `daf1266b2e9cf2b150ae804711ba1c26a082cdcc`
> **Prior Approved Milestone (P01-A)**: `a0b4c42de6bb11093cfac86e206e04819afacfdd`
> **Historical Frozen Baseline**: `phase0-architecture-v1` (`6f72eaca30be3fc3ac00f25829dd4283ed98c3f5`)
> **Pinned Agy**: `google-antigravity/antigravity-cli` `1.2.7` (`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`)
> **Pinned AO**: `Untrivial-ai/agent-orchestrator` `v0.13.0` (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN`

---

# 1. External Supervisor Verdict & Premature PASS Correction

The External Supervisor independently audited remote commit `daf1266b2e9cf2b150ae804711ba1c26a082cdcc` and rejected the worker-reported `P01-B = PASS` as **premature**.

### Reasons for PASS Rejection:
1. **Continuation Tests Incomplete**: Cross-process conversation resume (`--conversation`) was blocked by transient quota exhaustion (`RESOURCE_EXHAUSTED`, HTTP 429) during Turn 2, and workspace continuation (`--continue`) was not executed. Neither capability was empirically proven.
2. **Missing Local / Parse-Layer Negative Tests**: The canonical dossier lacked safe negative test evidence establishing CLI failure behavior before remote inference.
3. **Missing Binary & Help Evidence**: Binary provenance, exact version pin verification, and literal help flag semantics were absent from the canonical dossier.
4. **Missing Process / Orphan Evidence**: Explicit orphan check and cleanup audit evidence were not recorded.
5. **Canonical Source Truth Desynchronization**: `docs/sources/02_ANTIGRAVITY_CLI.md` and `docs/sources/UPSTREAM_CONTRACT_BASELINE.md` still designated Agy runtime capabilities as untested.
6. **Encoding Hygiene Regression**: Commit `daf1266` introduced UTF-8 BOM headers and mojibake characters into canonical documentation.

```text
P01_B_EXTERNAL_AUDIT = REMEDIATION_APPLIED_WAITING_FOR_QUOTA
P01-B worker-reported PASS = REJECTED_AS_PREMATURE
P01-B = NOT_EVALUATED
P01_B_PROVEN_SUBSET = PASS
P01_B_ENVIRONMENT = BLOCKED_BY_AGY_QUOTA_FOR_CONTINUATION_TESTS
P01-C = HELD
P01-D = TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED
ARCHITECTURE_V2 = CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN
ACTIVE_GATE = HUMAN_REQUIRED_P01_B_QUOTA_RECOVERY
```

---

# 2. Accepted Empirical Subset (P01-B1 through P01-B8)

The following empirical results from the P01-B runtime sandbox are verified, accepted, and preserved without re-execution:

| Validation ID | Capability | Result | Notes / Empirical Finding |
|---|---|---|---|
| **P01-B1** | Headless text mode (`-p`) | **`PASS`** | Echoed test token `HEADLESS_PROMPT_CONFIRMED`; exit code 0. |
| **P01-B2** | JSON envelope (`--output-format json`) | **`PASS`** | Valid JSON envelope containing `conversation_id`, `status:"SUCCESS"`, `response`, `duration_seconds`, `num_turns`, `usage`. |
| **P01-B3** | Stream-JSON output (`--output-format stream-json`) | **`PASS`** | Valid NDJSON event sequence: `init` -> `step_update` -> `result`. |
| **P01-B4** | Stdin Stream-JSON protocol (`--input-format stream-json`) | **`PASS`** | Accepted NDJSON user message format via stdin stream. |
| **P01-B4** | Same-process multi-turn context retention | **`PASS_WITH_QUOTA_ERROR_OBSERVED`** | Maintained identical `conversation_id` across turns within single process; Turn 2 successfully recovered Turn 1 hidden marker despite quota retry on final result. |
| **P01-B5** | JSON schema structured output (`--json-schema`) | **`PASS`** | Envelope populated `structured_output` field matching requested schema. |
| **P01-B6** | Schema + real filesystem side effect | **`PASS`** | File written to disk AND structured output emitted in the same turn. |
| **P01-B7** | Permission bypass (`--dangerously-skip-permissions`) | **`PASS`** | Automated tool execution without interactive prompt. |
| **P01-B8** | Secondary directory binding (`--add-dir`) | **`PASS`** | Agent successfully inspected and read marker from secondary workspace directory. |

---

# 3. Verification & Characterization Breakdown

The non-quota and negative-behavior evaluations are strictly distinguished into local verification versus remote-boundary behavioral characterization:

### Subsection A: Non-Quota Local Verification

1. **Version and Binary Provenance**:
   - `where.exe agy` resolved:
     - `C:\Users\Admin\AppData\Roaming\npm\agy.cmd`
     - `C:\Users\Admin\AppData\Local\agy\bin\agy.exe`
   - `agy --version` returned `1.2.7` (exit code 0).
   - **`AGY_VERSION_PIN = PASS`** (exact match to pinned upstream baseline `1.2.7`).

2. **Help Surface Validation (`agy --help`)**:
   - Verified literal existence and help semantics for all required flags:
     - `-p`, `--print`, `--prompt`
     - `--output-format` (text, json, stream-json)
     - `--input-format` (text, stream-json)
     - `--json-schema`
     - `--dangerously-skip-permissions`
     - `--add-dir`
     - `--conversation`
     - `-c`, `--continue`
   - **`AGY_HELP_FLAGS = PASS`**.

3. **Local Parse-Layer Schema Rejection (NEG-B)**:
   - `agy --json-schema DOES_NOT_EXIST.json -p "test"`:
     - Exited locally with code `1`.
     - Output: `Error: invalid --json-schema: failed to read schema file ... The system cannot find the file specified.`
     - Confirmed: zero model/API network requests initiated before failure.

4. **Process / Orphan Audit**:
   - Audited system processes for running `agy.exe`.
   - Three running `agy.exe` instances found (PIDs 9788, 23544, 36456), all started on 2026-09-20 (IDE-owned processes). Zero orphan child processes were left by P01-B tests.
   - **`P01B_ORPHAN_PROCESS_CHECK = PASS`**.

5. **Evidence & Sandbox Cleanup**:
   - All 9 evidence artifacts preserved in `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\evidence\`.
   - Sandbox directories intact and ready for continuation retests.
   - **`P01B_CLEANUP = PASS`**.

### Subsection B: Negative Behavioral Observations & Process Deviation Record

During remediation execution, negative tests NEG-A, NEG-C, and NEG-D were executed to determine whether Agy 1.2.7 rejects invalid parameters locally before remote inference.

**Process Deviation Record**:
```text
Deviation ID: P01B-DEV-001
Title: NEGATIVE_TEST_REMOTE_BOUNDARY_EXCEEDED
Classification: PROCESS_HYGIENE_DEVIATION
Security Impact: NONE OBSERVED
Evidence Validity: PRESERVED
Architecture Verdict Impact: NONE
Quota Impact: POSSIBLE / NOT QUANTIFIED
```
*Note*: The test runner did not abort before network transmission on NEG-A, NEG-C, and NEG-D because the Agy CLI itself does not perform local syntax/path pre-validation for those arguments. Literal evidence is preserved as an empirical characterization of CLI behavior; no tests were fabricated or re-executed.

**Literal Negative Behavioral Findings**:
1. **NEG-A (Invalid `--output-format` value)**:
   - Command: `agy --output-format definitely-invalid -p "test"`
   - Exit code: `0`.
   - Behavior: CLI silently ignored the invalid format flag and fell back to default text output. Remote model inference occurred.
2. **NEG-B (Nonexistent `--json-schema` file path)**:
   - Command: `agy --json-schema DOES_NOT_EXIST.json -p "test"`
   - Exit code: `1`.
   - Behavior: Immediate local descriptive rejection; zero remote requests.
3. **NEG-C (Malformed local JSON schema content)**:
   - Command: `agy --json-schema malformed.json -p "test"`
   - Exit code: `3`.
   - Behavior: CLI does not parse JSON schema locally; raw content passed to Gemini endpoint which rejected with `INVALID_ARGUMENT (code 400)`. Remote API request occurred.
4. **NEG-D (Nonexistent `--add-dir` path)**:
   - Command: `agy --add-dir DOES_NOT_EXIST_PATH -p "test"`
   - Exit code: `0`.
   - Behavior: CLI silently proceeded without error. Remote model inference occurred.

---

# 4. Failure Contract External Verdict & Architectural Constraint

### External Verdict:
- **`AGY_FAILURE_CONTRACT = EMPIRICALLY_CHARACTERIZED`**
- **`AGY_FAILURE_EXIT_CODES = PASS_WITH_CALLER_VALIDATION_CONSTRAINT`**

*Meaning of PASS*: The tested failure behaviors are fully and empirically characterized. This verdict certifies that caller/CLI failure boundaries are established; it does **NOT** imply that Agy safely rejects all malformed input on its own.

### Architectural Constraint (Carried Forward to P01-C / Implementation):
```text
SUPERVISOR_MUST_VALIDATE_AGY_INVOCATION_INPUTS_BEFORE_EXECUTION = REQUIRED
```
**Constraint Specifications**:
Because Agy 1.2.7 does not reliably reject malformed inputs locally, the future Supervisor Control Plane / AOAdapter integration boundary must enforce pre-flight validation prior to process spawning:
1. Validate `--output-format` against allowed enum values (`text`, `json`, `stream-json`).
2. Verify existence and readability of `--json-schema` file before passing flag.
3. Parse and validate JSON schema syntax locally prior to invocation.
4. Verify existence and accessibility of all `--add-dir` directories.
5. Restrict CLI invocations to strictly bounded, validated arguments.

*(Note: In accordance with Phase 1 directives, no application code or ADR is created at this time; this constraint is recorded for P01-C evaluation and subsequent implementation phases).*

---

# 5. Environment Blocker Record (ENV-P01B-001)

The condition preventing full P01-B track evaluation is classified as an environmental blocker, not an architectural gap:

| Field | Value |
|---|---|
| **Record ID** | `ENV-P01B-001` |
| **Classification** | `ENVIRONMENT_BLOCKER_TRANSIENT` |
| **Condition** | `AGY_INDIVIDUAL_QUOTA_EXHAUSTED` |
| **Error Details** | `RESOURCE_EXHAUSTED (code 429)` |
| **First Observed** | P01-B4 Turn 2 retry |
| **Blocking Scope** | P01-B9 (`--conversation`), P01-B10 (`--continue`), P01-B final external audit, P01-C release |
| **ADR Required** | **NO** (transient account quota limit, no architectural mismatch) |
| **Active Gate** | `HUMAN_REQUIRED_P01_B_QUOTA_RECOVERY` |
| **Quota Reset Timing** | **`ESTIMATED_ONLY`** (~42 hours from ~2026-09-21 17:55 +07:00; no provider SLA) |

---

# 6. Acceptance Matrix (16 Required Capabilities)

The track requires exactly 16 acceptance capabilities:

| Capability ID | Target Capability | Verification Type | Status |
|---|---|---|---|
| **AGY_VERSION_PIN** | Version exactly 1.2.7 | CLI `--version` | **`PASS`** |
| **AGY_HELP_FLAGS** | Expected CLI flags exposed | CLI `--help` | **`PASS`** |
| **AGY_HEADLESS_TEXT** | Headless prompt execution | Runtime P01-B1 | **`PASS`** |
| **AGY_OUTPUT_JSON** | JSON envelope output | Runtime P01-B2 | **`PASS`** |
| **AGY_OUTPUT_STREAM_JSON** | Stream-JSON NDJSON events | Runtime P01-B3 | **`PASS`** |
| **AGY_INPUT_STREAM_JSON** | Stdin Stream-JSON protocol | Runtime P01-B4 | **`PASS`** |
| **AGY_STREAM_CONVERSATION_CONTEXT** | In-process multi-turn context | Runtime P01-B4 | **`PASS_WITH_QUOTA_ERROR_OBSERVED`** |
| **AGY_JSON_SCHEMA** | Structured output schema | Runtime P01-B5 | **`PASS`** |
| **AGY_SCHEMA_AND_FILE_SIDE_EFFECT** | Schema + file side effect | Runtime P01-B6 | **`PASS`** |
| **AGY_DANGEROUS_SKIP_PERMISSIONS** | Autonomous tool permission bypass | Runtime P01-B7 | **`PASS`** |
| **AGY_ADD_DIR** | Multi-directory workspace binding | Runtime P01-B8 | **`PASS`** |
| **AGY_FAILURE_EXIT_CODES** | Failure contract characterization | Runtime Negative Tests | **`PASS_WITH_CALLER_VALIDATION_CONSTRAINT`** |
| **P01B_ORPHAN_PROCESS_CHECK** | No orphaned background processes | Process Audit | **`PASS`** |
| **P01B_CLEANUP** | Evidence preserved, sandbox clean | Filesystem Audit | **`PASS`** |
| **AGY_CONVERSATION_ID_RESUME** | Cross-process conversation resume | Runtime P01-B9 | **`NOT_EVALUATED`** (`ENV-P01B-001`) |
| **AGY_CONTINUE** | Workspace conversation continue | Runtime P01-B10 | **`NOT_EVALUATED`** (`ENV-P01B-001`) |

### Explicit Arithmetic Recount:
```text
13 PASS
+ 1 PASS_WITH_QUOTA_ERROR_OBSERVED
+ 2 NOT_EVALUATED
= 16 required items
```

**Track Verdict**: **`P01-B = NOT_EVALUATED`** (two required continuation capabilities remain unproven due to quota exhaustion).

**Outstanding Empirical Blockers**:
Only two capabilities block P01-B completion:
1. `AGY_CONVERSATION_ID_RESUME` (P01-B9)
2. `AGY_CONTINUE` (P01-B10)

*(Failure exit code characterization is completed and is NOT an outstanding blocker).*

---

# 7. Source Truth Reconciliation & Encoding Hygiene Audit

1. **Source Truth Files Updated**:
   - `docs/sources/02_ANTIGRAVITY_CLI.md`: Reconciled to `PARTIALLY_RUNTIME_TESTED`; recorded failure contract findings and `CALLER_VALIDATION_REQUIRED` constraint; preserved B9/B10 as `RUNTIME_NOT_EVALUATED`.
   - `docs/sources/UPSTREAM_CONTRACT_BASELINE.md`: Preserved B1–B8 granular statuses; updated failure contract row; confirmed B9/B10 as `RUNTIME_NOT_EVALUATED`.
2. **Encoding Hygiene Verification**:
   - All modified canonical Markdown files verified UTF-8 without BOM.
   - Zero mojibake characters present across all files.
   - Exact Unicode characters (`—`, `↔`) preserved.

---

# 8. Next Authorized Action

The Supervisor Control Plane remains held at gate `HUMAN_REQUIRED_P01_B_QUOTA_RECOVERY`.

```text
ACTIVE_GATE = HUMAN_REQUIRED_P01_B_QUOTA_RECOVERY
P01_A = EXTERNAL_AUDIT_APPROVED
P01_B_EXTERNAL_AUDIT = REMEDIATION_APPLIED_WAITING_FOR_QUOTA
P01-B = NOT_EVALUATED
P01-C = HELD
P01-D = TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED
ARCHITECTURE_V2 = CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN
```

**Directives**:
1. Do NOT invoke Agy with any model prompt.
2. Do NOT retry quota or run B9/B10 until explicit User authorization after quota recovery.
3. Do NOT release Track P01-C.
4. Do NOT proceed to Phase P02.
5. Do NOT freeze Architecture V2.
