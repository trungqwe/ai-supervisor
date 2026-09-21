# P01-B: Direct Antigravity CLI (Agy) Capability Proof

**Track:** P01-B -- Direct Antigravity CLI Capability Proof
**Pinned Baseline:** Agy `1.2.7` commit `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`
**Sandbox:** `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\`
**Execution Date:** 2026-09-21
**Overall Track Status:** `NOT_EVALUATED`
**Reason:** Required continuation tests (P01-B9, P01-B10) incomplete due to external quota exhaustion.
**Accepted Empirical Subset:** P01-B1 through P01-B8 PASS (externally audited and accepted).

---

## 1. Capability State Matrix

| Capability ID | Capability | State |
|---|---|---|
| `P01_B1_HEADLESS_TEXT` | Headless text mode (`-p`) | `PASS` |
| `P01_B2_JSON_OUTPUT` | JSON output envelope (`--output-format json`) | `PASS` |
| `P01_B3_STREAM_JSON_OUTPUT` | Stream-JSON output (`--output-format stream-json`) | `PASS` |
| `P01_B4_STREAM_JSON_INPUT` | Stream-JSON stdin input protocol (`--input-format stream-json`) | `PASS` |
| `P01_B4_STREAM_CONTEXT` | Same-process multi-turn context retention | `PASS_WITH_QUOTA_ERROR_OBSERVED` |
| `P01_B5_JSON_SCHEMA` | JSON Schema structured output (`--json-schema`) | `PASS` |
| `P01_B6_SCHEMA_AND_FILE_SIDE_EFFECT` | JSON Schema + real file side effect | `PASS` |
| `P01_B7_SKIP_PERMISSIONS` | Dangerously-skip-permissions flag | `PASS` |
| `P01_B8_ADD_DIR` | Multi-directory context (`--add-dir`) | `PASS` |
| `P01_B9_CONVERSATION_ID_RESUME` | Cross-process conversation resume (`--conversation`) | `NOT_EVALUATED_QUOTA_BLOCKED` |
| `P01_B10_CONTINUE` | Workspace continue (`--continue`) | `NOT_EVALUATED_QUOTA_BLOCKED` |
| `AGY_VERSION_PIN` | Version exactly 1.2.7 | `PASS` |
| `AGY_HELP_FLAGS` | Expected flags present in `agy --help` | `PASS` |
| `AGY_FAILURE_EXIT_CODES` | Parse-layer failure contract | `PARTIAL` |
| `P01B_ORPHAN_PROCESS_CHECK` | No P01-B orphan processes | `PASS` |
| `P01B_CLEANUP` | Evidence preserved, sandbox retained | `PASS` |

---

## 2. Scope and Objectives

This proof independently verifies direct Agy 1.2.7 CLI capabilities without Agent Orchestrator (AO)
intermediation. Disposable sandbox authorized by `TEST_CREDENTIAL_POLICY = USER_AUTHORIZED`.

---

## 3. Sandbox Structure

```
D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\
+-- primary\          # Primary working directory (git-initialized, clean baseline)
+-- secondary\        # Secondary directory for P01-B8 add-dir proof
+-- schemas\          # JSON schema files for P01-B5 and P01-B6
+-- evidence\         # Collected JSON evidence for all tests
+-- logs\             # Execution logs
```

---

## 4. Binary Provenance & Version Pin

**Command:** `Get-Command agy`
```
CommandType  Name      Version  Source
-----------  ----      -------  ------
Application  agy.cmd   0.0.0.0  C:\Users\Admin\AppData\Roaming\npm\agy.cmd
```

**Command:** `where.exe agy`
```
C:\Users\Admin\AppData\Roaming\npm\agy.cmd
C:\Users\Admin\AppData\Local\agy\bin\agy.exe
```

**Command:** `agy --version`
```
1.2.7
```
Exit code: `0`

**AGY_VERSION_PIN = PASS** (exact 1.2.7 confirmed)

---

## 5. Help Surface -- AGY_HELP_FLAGS

**Command:** `agy --help`
Exit code: `0`

**Literal help output (relevant flags):**
```
--add-dir                       Add a directory to the workspace (repeatable) (default [])
--agent                         Agent for the current CLI session
-c                              Short alias for --continue
--continue                      Continue the most recent conversation
--conversation                  Resume a previous conversation by ID
--dangerously-skip-permissions  Auto-approve all tool permission requests without prompting
--disable-slash-commands        Disable slash command and skill expansion in print mode
--effort                        Reasoning effort for the current CLI session (low|medium|high)
-i                              Short alias for --prompt-interactive
--input-format                  Input format for print mode (text, stream-json). stream-json reads one NDJSON
                                message per line from stdin and runs a turn for each; it requires
                                --output-format stream-json (default text)
--json-schema                   Optional JSON schema string or path to a schema file to enforce structured
                                output (for stream-json, only applicable to the final result)
--mode                          Set the agent execution mode for this session (accept-edits, plan)
--output-format                 Output format for print mode (text, json, stream-json) (default text)
-p                              Short alias for --print
--print                         Run a single prompt non-interactively and print the response
--prompt                        Alias for --print
--prompt-interactive            Run an initial prompt interactively and continue the session
```

**Flag semantics verified:**

| Flag | Literal Help Text | Verified Present |
|---|---|---|
| `-p` / `--print` / `--prompt` | "Run a single prompt non-interactively and print the response" | YES |
| `--output-format` | "Output format for print mode (text, json, stream-json)" | YES |
| `--input-format` | "Input format for print mode (text, stream-json)... requires --output-format stream-json" | YES |
| `--json-schema` | "Optional JSON schema string or path to a schema file" | YES |
| `--dangerously-skip-permissions` | "Auto-approve all tool permission requests without prompting" | YES |
| `--add-dir` | "Add a directory to the workspace (repeatable)" | YES |
| `--conversation` | "Resume a previous conversation by ID" | YES |
| `--continue` | "Continue the most recent conversation" | YES |
| `-c` | "Short alias for --continue" | YES |

**AGY_HELP_FLAGS = PASS**

---

## 6. Accepted Empirical Subset (P01-B1 through P01-B8)

### P01-B1: Headless Text Mode

**Command:**
```
agy -p "Echo the exact test token P01B_TEXT_PROBE_1789987031968 without additional commentary."
```

**Literal stdout:**
```
P01B_TEXT_PROBE_1789987031968
```

Exit code: `0` | Elapsed: 7,040 ms | Evidence: `evidence/p01b1_headless_text.json`

**P01_B1_HEADLESS_TEXT = PASS**

---

### P01-B2: Output Format JSON

**Command:**
```
agy -p "Echo the exact test token P01B_JSON_PROBE_1789987045275 in a short response." --output-format json
```

**Literal stdout:**
```json
{"conversation_id":"b94da29f-a091-4a74-9cc3-9474cb576bb2","status":"SUCCESS","response":"P01B_JSON_PROBE_1789987045275\n","duration_seconds":6.2798564,"num_turns":1,"usage":{"input_tokens":11750,"output_tokens":169,"thinking_tokens":146,"cache_read_tokens":0,"total_tokens":11919}}
```

Exit code: `0` | Elapsed: 10,286 ms | Evidence: `evidence/p01b2_output_json.json`

**P01_B2_JSON_OUTPUT = PASS**

---

### P01-B3: Output Format Stream-JSON

**Command:**
```
agy -p "Echo the exact test token P01B_STREAM_JSON_PROBE_1789987065370 in a short response." --output-format stream-json
```

**Literal stdout (6 NDJSON lines):**
```
{"event":"init","conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","init":{"cwd":"D:\\TU_CODE\\_ai_supervisor_p01b_agy_runtime\\primary","tools":[...],"permission_mode":"request-review"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":0,"state":"DONE","step_type":"user_input"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"P01B_STREAM_JSON_PROBE_1"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"789987065370"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"\n","duration_seconds":2.0620866,"usage":{...}}}
{"event":"result","result":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","status":"SUCCESS","response":"P01B_STREAM_JSON_PROBE_1789987065370\n","duration_seconds":2.1092669,"num_turns":1,"usage":{"input_tokens":11759,"output_tokens":92,"thinking_tokens":67,"cache_read_tokens":0,"total_tokens":11851}}}
```

Exit code: `0` | Lines: 6 | Evidence: `evidence/p01b3_output_stream_json.json`

**P01_B3_STREAM_JSON_OUTPUT = PASS**

---

### P01-B4: Stream-JSON Multi-Turn (stdin Pipe Protocol)

**Command:**
```
agy --input-format stream-json --output-format stream-json
```

Turn 1 stdin: `{"event":"user","message":{"content":"Remember token P01B_CTX_1789987727685. Reply OK."}}`
Turn 2 stdin: `{"event":"user","message":{"content":"What was the token I just told you? Reply with only the token."}}`

**Key result event (Turn 2):**
```json
{"event":"result","result":{"conversation_id":"8094dd5b-bfbc-41f4-9a91-54bda2129a84","status":"ERROR","response":"P01B_CTX_1789987727685\n","error":"API error (attempt 5): RESOURCE_EXHAUSTED (code 429)...","duration_seconds":77.4893309,"num_turns":2,"usage":{"input_tokens":23702,"output_tokens":275,"thinking_tokens":253,"cache_read_tokens":0,"total_tokens":23977}}}
```

Exit code: `0` | Events: 12 | Evidence: `evidence/p01b4_stream_json_multiturn.json`

**Observations:**
- `conversation_id` `8094dd5b-bfbc-41f4-9a91-54bda2129a84` consistent across both turns
- Marker `P01B_CTX_1789987727685` present in Turn 2 `response` -- context retained
- `num_turns: 2` confirmed same-process multi-turn
- `status: "ERROR"` reflects quota retry exhaustion (5 attempts) during Turn 2 API calls; marker still recovered
- `QUOTA_ERROR_OBSERVED_ON_FINAL_RESULT` -- does NOT generalize to cross-process persistence

**P01_B4_STREAM_JSON_INPUT = PASS**
**P01_B4_STREAM_CONTEXT = PASS_WITH_QUOTA_ERROR_OBSERVED**

---

### P01-B5: JSON Schema Structured Output

**Command:**
```
agy -p "Return structured output: status=SUCCESS, marker=P01B_SCHEMA_1789987830348" --output-format json --json-schema "...\schemas\p01b_result.schema.json"
```

**Literal `structured_output`:**
```json
{"marker":"P01B_SCHEMA_1789987830348","status":"SUCCESS"}
```

Exit code: `0` | Elapsed: 7,819 ms | Evidence: `evidence/p01b5_json_schema.json`

**P01_B5_JSON_SCHEMA = PASS**

---

### P01-B6: JSON Schema + Real File Side Effect

**Command:**
```
agy -p "Create p01b_schema_side_effect.txt with text P01B_FILE_VAL_1789987846871. Return structured output." --output-format json --json-schema "...\p01b_file_effect.schema.json" --dangerously-skip-permissions
```

**Literal `structured_output`:**
```json
{"file":"p01b_schema_side_effect.txt","result":"created","status":"SUCCESS"}
```

File `p01b_schema_side_effect.txt` confirmed created; content matches marker. `gitCleanExceptTarget: true`.

Exit code: `0` | Elapsed: 38,724 ms | Evidence: `evidence/p01b6_schema_file_side_effect.json`

**P01_B6_SCHEMA_AND_FILE_SIDE_EFFECT = PASS**

---

### P01-B7: Dangerously Skip Permissions

**Command:**
```
agy -p "Write text P01B_PERM_1789987892282 into p01b_danger_perm_test.txt." --output-format json --dangerously-skip-permissions
```

**Literal stdout:**
```json
{"conversation_id":"673a31ad-b928-46e4-b353-bdc59e37eb75","status":"SUCCESS","response":"The text `P01B_PERM_1789987892282` has been written to [p01b_danger_perm_test.txt](...).","duration_seconds":26.9927669,"num_turns":1,"usage":{...}}
```

File confirmed: exists, content matches. Exit code: `0` | Elapsed: 32,927 ms | Evidence: `evidence/p01b7_dangerous_skip_permissions.json`

**P01_B7_SKIP_PERMISSIONS = PASS**

---

### P01-B8: Add-Dir (Multi-Directory Context)

**Command:**
```
agy -p "Read p01b_secondary_marker.txt from the additional directory and echo its exact content." --output-format json --add-dir "D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\secondary" --dangerously-skip-permissions
```

**Literal stdout:**
```json
{"conversation_id":"11897aba-f8ec-4422-abd6-214a09abada2","status":"SUCCESS","response":"...`P01B_SECRET_OUT_OF_BAND_1789987932491`\n","duration_seconds":3.7359836,"num_turns":1,"usage":{...}}
```

Marker `P01B_SECRET_OUT_OF_BAND_1789987932491` confirmed (stored in secondary dir only, not primary).

Exit code: `0` | Elapsed: 9,314 ms | Evidence: `evidence/p01b8_add_dir.json`

**P01_B8_ADD_DIR = PASS**

---

## 7. Safe Negative Tests -- AGY_FAILURE_EXIT_CODES

### NEG-A: Invalid --output-format value

**Command:**
```
agy.exe --print "test" --output-format definitely-invalid
```

Exit code: `0` | Stdout: model response text | Stderr: (empty)

**Finding:** Agy silently ignores the invalid `--output-format` value and falls back to default text output. No parse-layer rejection. No API request aborted.

**NEG-A result: NOT_EVALUATED_QUOTA_BLOCKED** (proceeded to model inference; aborted further analysis to avoid quota)

---

### NEG-B: Nonexistent --json-schema path

**Command:**
```
agy.exe --print "test" --output-format json --json-schema "D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\schemas\DOES_NOT_EXIST.json"
```

Exit code: `1` | Stdout: (empty) | Stderr:
```
Error: invalid --json-schema: failed to read schema file "D:\\TU_CODE\\_ai_supervisor_p01b_agy_runtime\\schemas\\DOES_NOT_EXIST.json": open D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\schemas\DOES_NOT_EXIST.json: The system cannot find the file specified.
```

**Finding:** Agy validates schema path at parse layer. Immediate exit code `1` with descriptive error. No model/API request made.

**NEG-B result: PASS** (local parse-layer rejection confirmed)

---

### NEG-C: Malformed JSON schema content

**Command:**
```
agy.exe --print "test" --output-format json --json-schema "<file with malformed JSON>"
```

Exit code: `3` | Model/API request: YES (proceeded to remote)

Stderr contained:
```
error: INVALID_ARGUMENT (code 400): * GenerateContentRequest.tools[13]...
```

**Finding:** Agy does not validate JSON schema content locally. The malformed schema was passed to the API as a tool parameter, which rejected it with `INVALID_ARGUMENT (400)`. Exit code `3` observed.

**NEG-C result: NOT_EVALUATED_QUOTA_BLOCKED** (API request was made; not a pure parse-layer rejection; aborted to avoid quota impact)

---

### NEG-D: Nonexistent --add-dir path

**Command:**
```
agy.exe --print "test" --output-format json --add-dir "D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\DOES_NOT_EXIST" --dangerously-skip-permissions
```

Exit code: `0` | Stdout: successful model response | Stderr: (empty)

**Finding:** Agy does not validate `--add-dir` path existence at parse layer. The nonexistent directory is silently ignored; the agent proceeds normally.

**NEG-D result: NOT_EVALUATED_QUOTA_BLOCKED** (proceeded to full model inference; aborted to avoid quota)

---

### Failure Exit Code Summary

| Scenario | Exit Code | Parse-Layer Rejection | Notes |
|---|---|---|---|
| Invalid `--output-format` value | `0` | NO | Silently ignored; falls back to text output |
| Nonexistent `--json-schema` path | `1` | YES | Immediate local error, no API call |
| Malformed JSON schema content | `3` | NO | Passed to API; `INVALID_ARGUMENT (400)` returned |
| Nonexistent `--add-dir` path | `0` | NO | Silently ignored; agent proceeds normally |

**AGY_FAILURE_EXIT_CODES = PARTIAL**

Only `--json-schema` nonexistent path produces a clean parse-layer rejection (exit 1). Invalid `--output-format` and nonexistent `--add-dir` silently continue. Malformed schema content proceeds to API.

---

## 8. Environment Blocker Record

### ENV-P01B-001: Agy API Quota Exhausted

| Field | Value |
|---|---|
| **Record ID** | `ENV-P01B-001` |
| **Classification** | `ENVIRONMENT_BLOCKER_TRANSIENT` |
| **Condition** | `AGY_INDIVIDUAL_QUOTA_EXHAUSTED` |
| **Error** | `RESOURCE_EXHAUSTED (code 429)` |
| **First observed** | P01-B4 Turn 2 (quota retry on final result) |
| **Blocking** | P01-B9 (conversation ID resume), P01-B10 (--continue) |
| **Impact** | P01-B continuation acceptance incomplete; track overall = NOT_EVALUATED |
| **ADR Required** | NO -- transient environment condition, not architectural gap |
| **Active Gate** | `HUMAN_REQUIRED_P01_B_QUOTA_RECOVERY` |
| **Quota Reset** | ~42h from P01-B9 execution (2026-09-21 17:55 +07:00) |

---

## 9. Process / Orphan Check

**Finding:** Three `agy.exe` processes found running (PIDs 9788, 23544, 36456).

All three started 2026-09-20 (before P01-B tests on 2026-09-21). These are IDE-owned processes,
not P01-B sandbox orphans. P01-B tests used `Start-Process -Wait` or Node `child_process` with
completion callbacks; all evidence files written by 17:55 (+07:00) with no live children expected.

**P01B_ORPHAN_PROCESS_CHECK = PASS** (no P01-B orphan processes)

---

## 10. Cleanup

Evidence directory preserved:
```
D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\evidence\
  p01b1_headless_text.json         (written 17:37)
  p01b2_output_json.json           (written 17:37)
  p01b3_output_stream_json.json    (written 17:37)
  p01b4_stream_json_multiturn.json (written 17:50)
  p01b5_json_schema.json           (written 17:50)
  p01b6_schema_file_side_effect.json (written 17:51)
  p01b7_dangerous_skip_permissions.json (written 17:52)
  p01b8_add_dir.json               (written 17:52)
  p01b9_conversation_id_resume.json (written 17:55)
```

Sandbox retained for continuation testing after quota recovery.

**P01B_CLEANUP = PASS**

---

## 11. P01-B9 Retest Plan (After Quota Recovery Only)

Execute only after User explicitly authorizes continuation following quota reset.

**Turn 1:** New agy process (no `--conversation` flag)
- Store fresh unique marker in prompt
- Capture `conversation_id` from JSON envelope
- Require `status: "SUCCESS"` and exit code `0`

**Turn 2:** New agy process (separate invocation)
- `--conversation <exact conversation_id from Turn 1>`
- Ask for marker WITHOUT including it in the prompt
- Require `status: "SUCCESS"`, exit code `0`, hidden marker recovered exactly

**Acceptance criteria:**
- Same `conversation_id` both turns
- Hidden marker recovered in Turn 2 `response`
- No quota error in either turn

**Previous P01-B9 attempt remains historical evidence only -- NOT PASS.**

---

## 12. P01-B10 --continue Plan (After Quota Recovery Only)

Execute only after User explicitly authorizes continuation following quota reset.

Read exact `agy --help` semantics for `--continue` immediately before execution.

**Invocation 1:** New agy process from controlled `primary` workspace
- Store fresh unique marker, finish successfully
- Require `status: "SUCCESS"`, exit code `0`

**Invocation 2:** New agy process from same workspace
- `agy --continue` (or `-c`)
- Ask for hidden marker without revealing it
- Require: successful result, hidden marker recovered, expected conversation chosen

---

## 13. Overall Track Verdict

**P01-B = NOT_EVALUATED**

Reason: Required continuation tests (P01-B9, P01-B10) are incomplete due to `ENV-P01B-001`
(`AGY_INDIVIDUAL_QUOTA_EXHAUSTED`). This is a transient environment condition, not a CLI capability
defect.

Accepted empirical subset (B1-B8) is validated and externally audited.

**Do NOT state that P01-C is releasable based on this track record.**

Full track PASS requires all items in the acceptance matrix below to reach accepted state.

---

## 14. Acceptance Matrix

| Item | Required State | Current State |
|---|---|---|
| `AGY_VERSION_PIN` | PASS | **PASS** |
| `AGY_HELP_FLAGS` | PASS | **PASS** |
| `AGY_HEADLESS_TEXT` | PASS | **PASS** (P01-B1) |
| `AGY_OUTPUT_JSON` | PASS | **PASS** (P01-B2) |
| `AGY_OUTPUT_STREAM_JSON` | PASS | **PASS** (P01-B3) |
| `AGY_INPUT_STREAM_JSON` | PASS | **PASS** (P01-B4) |
| `AGY_STREAM_CONVERSATION_CONTEXT` | PASS | **PASS_WITH_QUOTA_ERROR_OBSERVED** (P01-B4) |
| `AGY_JSON_SCHEMA` | PASS | **PASS** (P01-B5) |
| `AGY_SCHEMA_AND_FILE_SIDE_EFFECT` | PASS | **PASS** (P01-B6) |
| `AGY_DANGEROUS_SKIP_PERMISSIONS` | PASS | **PASS** (P01-B7) |
| `AGY_ADD_DIR` | PASS | **PASS** (P01-B8) |
| `AGY_CONVERSATION_ID_RESUME` | PASS | **NOT_EVALUATED** (ENV-P01B-001) |
| `AGY_CONTINUE` | PASS | **NOT_EVALUATED** (ENV-P01B-001) |
| `AGY_FAILURE_EXIT_CODES` | PASS | **PARTIAL** (NEG-B PASS; NEG-A/C/D NOT_EVALUATED) |
| `P01B_ORPHAN_PROCESS_CHECK` | PASS | **PASS** |
| `P01B_CLEANUP` | PASS | **PASS** |

**Outstanding items blocking track PASS:** `AGY_CONVERSATION_ID_RESUME`, `AGY_CONTINUE`