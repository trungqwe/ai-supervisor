# P01-B -- EXTERNAL SUPERVISOR AUDIT DOSSIER

> **Authority**: External Supervisor Independent Audit Authority
> **Status**: APPROVED
> **Verdict**: P01_B_EXTERNAL_AUDIT = APPROVED
> **Date**: 2026-09-21
> **Repository**: `D:\TU_CODE\ai-supervisor`
> **Audited Initial Proof**: `daf1266b2e9cf2b150ae804711ba1c26a082cdcc`
> **Remediation / Quota Consistency**: `b1f3204eaa8f828f6d62cec7e6f627ac85bd4575`
> **Continuation Completion / Final Audited Commit**: `4bb3cb0a33f70aafd23f5e4bf0007cafd39c9d85`
> **Prior Approved Milestone (P01-A)**: `a0b4c42de6bb11093cfac86e206e04819afacfdd`
> **Historical Frozen Baseline**: `phase0-architecture-v1` (`6f72eaca30be3fc3ac00f25829dd4283ed98c3f5`)
> **Pinned Agy**: `google-antigravity/antigravity-cli` `1.2.7` (`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`)
> **Pinned AO**: `Untrivial-ai/agent-orchestrator` `v0.13.0` (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Architecture Status**: `ARCHITECTURE_V2_CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN`

---

# 1. External Supervisor Audit History & Remediation Progression

1. **Initial Verdict on `daf1266`**: The External Supervisor independently audited remote commit `daf1266b2e9cf2b150ae804711ba1c26a082cdcc` and rejected the worker-reported `P01-B = PASS` as premature due to incomplete continuation tests (B9/B10 blocked by quota), missing negative test findings, missing process/binary provenance evidence, and source truth desynchronization.
2. **Remediation Phase**: Remediated encoding hygiene, binary provenance, help flag surface, characterization of failure contract (`AGY_FAILURE_EXIT_CODES = PASS_WITH_CALLER_VALIDATION_CONSTRAINT`), orphan process check, and canonical source truth synchronization.
3. **Continuation Execution Phase**: Following the User's quota policy supersession (`QUOTA_POLICY = NON_BLOCKING_OPERATIONAL_CONCERN`), both required continuation tests (P01-B9 and P01-B10) were executed and completed with 100% empirical success.

```text
P01_A = EXTERNAL_AUDIT_APPROVED
P01_B_EXTERNAL_AUDIT = APPROVED
P01-B = EXTERNAL_AUDIT_APPROVED
P01-C = READY
P01-D = TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED
ARCHITECTURE_V2 = CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN
ACTIVE_GATE = P01_C_RUNTIME_EXECUTION_AUTHORIZED
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
   - All empirical evidence artifacts preserved in `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\evidence\`.
   - Sandbox directories intact and clean.
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

---

# 5. User Policy Override & Quota Blocker Supersession

### A. Quota Policy Supersession
The User has explicitly funded additional usage capacity and superseded the previous quota-gating policy:
```text
QUOTA_POLICY = NON_BLOCKING_OPERATIONAL_CONCERN
```
- Quota or capacity events are logged as operational telemetry and do not create architecture gates or project-stop gates.
- Runtime capabilities still strictly require actual empirical proof with real successful invocations.

### B. Superseding ENV-P01B-001
- **Record ID**: `ENV-P01B-001`
- **Historical Classification**: `HISTORICAL_TRANSIENT_ENVIRONMENT_EVENT`
- **Active Blocker**: **`NO`**
- **Active Gate**: **`NONE`**
- **ADR Required**: `NO`

---

# 6. Empirical Continuation Proofs (P01-B9 & P01-B10)

Following policy supersession, both remaining continuation capabilities were executed and verified live:

### A. P01-B9: Cross-Process `--conversation` Resume
- **Sandbox Workspace**: `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\primary`
- **Generated Marker**: `P01B_CONV_1789998249096_4803`
- **Turn 1**:
  - Command: `agy -p "Remember this exact marker: P01B_CONV_1789998249096_4803. Reply only ACK." --output-format json`
  - Exit code: `0` | Elapsed: `7796 ms` | Status: `"SUCCESS"`
  - Captured `conversation_id`: `f6420dd6-e11b-4eb0-9bba-d77813c2fa04`
- **Turn 2**:
  - Command: `agy --conversation f6420dd6-e11b-4eb0-9bba-d77813c2fa04 -p "What exact marker did I ask you to remember in the previous turn? Return only the marker." --output-format json`
  - Exit code: `0` | Elapsed: `12451 ms` | Status: `"SUCCESS"`
  - Response: `"P01B_CONV_1789998249096_4803\n"`
  - Marker match: **EXACT**
- **Evidence File**: `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\evidence\p01b9_conversation_id_resume_proven.json`
- **Verdict**: **`AGY_CONVERSATION_ID_RESUME = PASS`**

### B. P01-B10: Workspace Continue (`--continue`)
- **Sandbox Workspace**: `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\primary` (identical controlled root)
- **Generated Marker**: `P01B_CONTINUE_1789998278237_8582`
- **Turn 1**:
  - Command: `agy -p "Remember this exact marker: P01B_CONTINUE_1789998278237_8582. Reply only ACK." --output-format json`
  - Exit code: `0` | Elapsed: `6923 ms` | Status: `"SUCCESS"`
  - Created `conversation_id`: `738074b7-8676-4841-bea4-9ad0395a4211`
- **Turn 2**:
  - Command: `agy --continue -p "What exact marker did I ask you to remember in the previous conversation? Return only the marker." --output-format json`
  - Exit code: `0` | Elapsed: `6512 ms` | Status: `"SUCCESS"`
  - Selected `conversation_id`: `738074b7-8676-4841-bea4-9ad0395a4211` (automatically continued most recent session)
  - Response: `"P01B_CONTINUE_1789998278237_8582\n"`
  - Marker match: **EXACT**
- **Evidence File**: `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\evidence\p01b10_continue_proven.json`
- **Verdict**: **`AGY_CONTINUE = PASS`**

---

# 7. Final Acceptance Matrix (16 Required Capabilities)

All 16 capabilities have reached verified accepted state:

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
| **AGY_CONVERSATION_ID_RESUME** | Cross-process conversation resume | Runtime P01-B9 | **`PASS`** |
| **AGY_CONTINUE** | Workspace conversation continue | Runtime P01-B10 | **`PASS`** |

### Arithmetic Recount:
```text
14 PLAIN PASS
+ 1 PASS_WITH_CALLER_VALIDATION_CONSTRAINT
+ 1 PASS_WITH_QUOTA_ERROR_OBSERVED
= 16 / 16 ACCEPTED
```

**Outstanding Empirical Blockers**: **NONE**.

---

# 8. Source Truth Reconciliation & Encoding Hygiene Audit

1. **Source Truth Files Updated**:
   - `docs/sources/02_ANTIGRAVITY_CLI.md`: Updated to `RUNTIME_TESTED_PASS` / `P01-B COMPLETE`; documented empirical proof of `--conversation` and `--continue`; recorded `CALLER_VALIDATION_REQUIRED` constraint.
   - `docs/sources/UPSTREAM_CONTRACT_BASELINE.md`: Updated Conversation ID Resume and Workspace Continue to `RUNTIME_TESTED_PASS`.
2. **Encoding Hygiene Verification**:
   - All modified canonical Markdown files verified UTF-8 without BOM.
   - Zero mojibake characters present across all files.
   - Exact Unicode characters (`—`, `↔`) preserved.

---

# 9. Final Track Verdict & Next Authorized Action

```text
P01_A = EXTERNAL_AUDIT_APPROVED
P01_B_EXTERNAL_AUDIT = APPROVED
P01-B = EXTERNAL_AUDIT_APPROVED
P01-C = READY
P01-D = TRANSPORT_PROVEN_EXTERNAL_AUDIT_APPROVED
ARCHITECTURE_V2 = CANDIDATE_TRANSPORT_APPROVED_NOT_FROZEN
ACTIVE_GATE = P01_C_RUNTIME_EXECUTION_AUTHORIZED
```

**Directives**:
1. Track P01-B runtime proof is APPROVED by the External Supervisor.
2. Track P01-C is released to **`READY`** and authorized for runtime execution.
3. Proceed to Track P01-C execution.
4. Do NOT proceed to Phase P02.
5. Do NOT freeze Architecture V2.
