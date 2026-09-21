# P01-B: Direct Antigravity CLI (Agy) Capability Proof

**Track:** P01-B — Direct Antigravity CLI Capability Proof  
**Pinned Baseline:** Agy `1.2.7` commit `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`  
**Sandbox:** `D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\`  
**Execution Date:** 2026-09-21  
**Overall Result:** `PASS` (8/9 validations PASS; P01-B9 `QUOTA_EXHAUSTED_GAP`)

---

## 1. Scope and Objectives

This proof independently verifies direct Agy 1.2.7 CLI capabilities without Agent Orchestrator (AO) intermediation, covering:

- Headless text execution mode
- JSON and Stream-JSON output formats
- Multi-turn stream-JSON session (stdin pipe protocol)
- JSON Schema structured output
- Real filesystem side effects (file creation)
- Dangerously-skip-permissions flag
- Multi-directory context (`--add-dir`)
- Conversation ID resume (`--conversation`)

---

## 2. Sandbox Structure

```
D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\
├── primary\          # Primary working directory (git-initialized, clean baseline)
├── secondary\        # Secondary directory for P01-B8 add-dir proof
├── schemas\          # JSON schema files for P01-B5 and P01-B6
├── evidence\         # Collected JSON evidence for all tests
└── logs\             # Execution logs
```

Sandbox git baseline verified clean before each test. Disposable environment authorized by `TEST_CREDENTIAL_POLICY = USER_AUTHORIZED`.

---

## 3. Test Results Summary

| Test | Capability | Result | Exit Code | Elapsed Ms |
|------|-----------|--------|-----------|------------|
| P01-B1 | Headless Text Mode | **PASS** | 0 | 7,040 |
| P01-B2 | Output Format JSON | **PASS** | 0 | 10,286 |
| P01-B3 | Output Format Stream-JSON | **PASS** | 0 | 6,777 |
| P01-B4 | Stream-JSON Multi-Turn (stdin) | **PASS** | 0 | 81,194 |
| P01-B5 | JSON Schema Structured Output | **PASS** | 0 | 7,819 |
| P01-B6 | JSON Schema + Real File Side Effect | **PASS** | 0 | 38,724 |
| P01-B7 | Dangerously Skip Permissions | **PASS** | 0 | 32,927 |
| P01-B8 | Add-Dir (Multi-Directory Context) | **PASS** | 0 | 9,314 |
| P01-B9 | Conversation ID Resume | **QUOTA_EXHAUSTED_GAP** | 3 | 142,247 |

---

## 4. Literal Evidence by Test

### P01-B1: Headless Text Mode

**Command:**
```
agy -p "Echo the exact test token P01B_TEXT_PROBE_1789987031968 without additional commentary."
```

**Evidence file:** `evidence/p01b1_headless_text.json`

**Literal stdout:**
```
P01B_TEXT_PROBE_1789987031968
```

**Validations:**
- Exit code: `0` ✓
- Marker `P01B_TEXT_PROBE_1789987031968` present in stdout: `true` ✓
- Elapsed: 7,040 ms

**Result: PASS**

---

### P01-B2: Output Format JSON

**Command:**
```
agy -p "Echo the exact test token P01B_JSON_PROBE_1789987045275 in a short response." --output-format json
```

**Evidence file:** `evidence/p01b2_output_json.json`

**Literal stdout:**
```json
{"conversation_id":"b94da29f-a091-4a74-9cc3-9474cb576bb2","status":"SUCCESS","response":"P01B_JSON_PROBE_1789987045275\n","duration_seconds":6.2798564,"num_turns":1,"usage":{"input_tokens":11750,"output_tokens":169,"thinking_tokens":146,"cache_read_tokens":0,"total_tokens":11919}}
```

**Validations:**
- Exit code: `0` ✓
- `isValidJson`: `true` ✓
- Top-level keys: `conversation_id`, `status`, `response`, `duration_seconds`, `num_turns`, `usage` ✓
- `status`: `"SUCCESS"` ✓
- Marker `P01B_JSON_PROBE_1789987045275` in `response` field ✓
- Elapsed: 10,286 ms

**Result: PASS**

---

### P01-B3: Output Format Stream-JSON

**Command:**
```
agy -p "Echo the exact test token P01B_STREAM_JSON_PROBE_1789987065370 in a short response." --output-format stream-json
```

**Evidence file:** `evidence/p01b3_output_stream_json.json`

**Literal stdout (6 NDJSON lines):**
```
{"event":"init","conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","init":{"cwd":"D:\\TU_CODE\\_ai_supervisor_p01b_agy_runtime\\primary","tools":[...],"permission_mode":"request-review"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":0,"state":"DONE","step_type":"user_input"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"P01B_STREAM_JSON_PROBE_1"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"789987065370"}}
{"event":"step_update","step_update":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"\n","duration_seconds":2.0620866,"usage":{"input_tokens":11759,"output_tokens":92,"thinking_tokens":67,"cache_read_tokens":0,"total_tokens":11851}}}
{"event":"result","result":{"conversation_id":"3ba21970-8d27-43f6-8148-c7276d971827","status":"SUCCESS","response":"P01B_STREAM_JSON_PROBE_1789987065370\n","duration_seconds":2.1092669,"num_turns":1,"usage":{"input_tokens":11759,"output_tokens":92,"thinking_tokens":67,"cache_read_tokens":0,"total_tokens":11851}}}
```

**Observed event types:** `init`, `step_update`, `result`

**Validations:**
- Exit code: `0` ✓
- All 6 lines valid NDJSON ✓
- Event sequence: `init → step_update(user_input) → step_update(agent_response×3) → result` ✓
- Marker `P01B_STREAM_JSON_PROBE_1789987065370` confirmed in `text_delta` and final `result.response` ✓
- Elapsed: 6,777 ms

**Result: PASS**

---

### P01-B4: Stream-JSON Multi-Turn (stdin Pipe Protocol)

**Command:**
```
agy --input-format stream-json --output-format stream-json
```
(Turn 1: store marker; Turn 2: retrieve marker via stdin pipe)

**Evidence file:** `evidence/p01b4_stream_json_multiturn.json`

**Protocol:** Two sequential `{"event":"user","message":{"content":"..."}}` messages piped to stdin.

**Literal result event (turn 2):**
```json
{"event":"result","result":{"conversation_id":"8094dd5b-bfbc-41f4-9a91-54bda2129a84","status":"ERROR","response":"P01B_CTX_1789987727685\n","error":"API error (attempt 5): RESOURCE_EXHAUSTED (code 429): Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 42h10m51s.","duration_seconds":77.4893309,"num_turns":2,"usage":{"input_tokens":23702,"output_tokens":275,"thinking_tokens":253,"cache_read_tokens":0,"total_tokens":23977}}}
```

**Validations:**
- `conversation_id` consistent across both turns: `8094dd5b-bfbc-41f4-9a91-54bda2129a84` ✓
- Marker `P01B_CTX_1789987727685` present in `response` despite `status:"ERROR"` ✓
- `markerRetrieved`: `true` ✓
- `turnCount`: `2`; `eventsCount`: `12` ✓
- Cross-turn context retention confirmed (marker set in Turn 1, echoed in Turn 2) ✓
- `num_turns`: `2` confirmed multi-turn session ✓

**Note:** `status:"ERROR"` reflects quota retry exhaustion on Turn 2 API calls. Agy retried 5 times then returned the partial response. The context window was maintained and marker retrieved successfully.

**Result: PASS**

---

### P01-B5: JSON Schema Structured Output

**Command:**
```
agy -p "Return a structured JSON output satisfying the schema where status is \"SUCCESS\" and marker is \"P01B_SCHEMA_1789987830348\"." --output-format json --json-schema "D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\schemas\p01b_result.schema.json"
```

**Evidence file:** `evidence/p01b5_json_schema.json`

**Literal `structured_output` field:**
```json
{"marker":"P01B_SCHEMA_1789987830348","status":"SUCCESS"}
```

**Literal stdout (partial):**
```json
{"conversation_id":"fc2bfc8c-da70-49e2-baa9-86feb8d35950","status":"SUCCESS","response":"{\"marker\":\"P01B_SCHEMA_1789987830348\",\"status\":\"SUCCESS\",...}\n","duration_seconds":3.5387071,"num_turns":1,"structured_output":{"marker":"P01B_SCHEMA_1789987830348","status":"SUCCESS"},"json_schema":{...}}
```

**Validations:**
- Exit code: `0` ✓
- `validSchema`: `true` ✓
- `structured_output.marker`: `P01B_SCHEMA_1789987830348` ✓
- `structured_output.status`: `SUCCESS` ✓
- Schema enforced (`additionalProperties:false` in schema) ✓
- Elapsed: 7,819 ms

**Result: PASS**

---

### P01-B6: JSON Schema + Real File Side Effect

**Command:**
```
agy -p "Create a new file named p01b_schema_side_effect.txt containing exact text \"P01B_FILE_VAL_1789987846871\". Then return structured output..." --output-format json --json-schema "...p01b_file_effect.schema.json" --dangerously-skip-permissions
```

**Evidence file:** `evidence/p01b6_schema_file_side_effect.json`

**Literal `structured_output`:**
```json
{"file":"p01b_schema_side_effect.txt","result":"created","status":"SUCCESS"}
```

**Validations:**
- Exit code: `0` ✓
- `validSchema`: `true` ✓
- `fileCreated`: `true` ✓
- `contentMatches`: `true` (file contains exact marker `P01B_FILE_VAL_1789987846871`) ✓
- `gitCleanExceptTarget`: `true` (no unintended side effects) ✓
- Elapsed: 38,724 ms

**Result: PASS**

---

### P01-B7: Dangerously Skip Permissions

**Command:**
```
agy -p "Write the text \"P01B_PERM_1789987892282\" into p01b_danger_perm_test.txt and finish." --output-format json --dangerously-skip-permissions
```

**Evidence file:** `evidence/p01b7_dangerous_skip_permissions.json`

**Literal stdout:**
```json
{"conversation_id":"673a31ad-b928-46e4-b353-bdc59e37eb75","status":"SUCCESS","response":"The text `P01B_PERM_1789987892282` has been written to [p01b_danger_perm_test.txt](file:///D:/TU_CODE/_ai_supervisor_p01b_agy_runtime/primary/p01b_danger_perm_test.txt).\n","duration_seconds":26.9927669,"num_turns":1,"usage":{"input_tokens":132976,"output_tokens":3013,"thinking_tokens":1894,"cache_read_tokens":0,"total_tokens":135989}}
```

**Validations:**
- Exit code: `0` ✓
- `fileExists`: `true` ✓
- `fileMatches`: `true` (file content = `P01B_PERM_1789987892282`) ✓
- No permission prompt required ✓
- Elapsed: 32,927 ms

**Result: PASS**

---

### P01-B8: Add-Dir (Multi-Directory Context)

**Command:**
```
agy -p "Read the file p01b_secondary_marker.txt from the additional directory and echo its exact content." --output-format json --add-dir "D:\TU_CODE\_ai_supervisor_p01b_agy_runtime\secondary" --dangerously-skip-permissions
```

**Evidence file:** `evidence/p01b8_add_dir.json`

**Literal stdout:**
```json
{"conversation_id":"11897aba-f8ec-4422-abd6-214a09abada2","status":"SUCCESS","response":"The exact content of [p01b_secondary_marker.txt](...) is:\n\n```\nP01B_SECRET_OUT_OF_BAND_1789987932491\n```\n","duration_seconds":3.7359836,"num_turns":1,"usage":{"input_tokens":24067,"output_tokens":653,"thinking_tokens":506,"cache_read_tokens":0,"total_tokens":24720}}
```

**Validations:**
- Exit code: `0` ✓
- `markerRecovered`: `true` (marker `P01B_SECRET_OUT_OF_BAND_1789987932491` echoed) ✓
- Marker was stored in secondary directory only (not accessible from primary alone) ✓
- `noExtraWrites`: `true` ✓
- Elapsed: 9,314 ms

**Result: PASS**

---

### P01-B9: Conversation ID Resume

**Command (Turn 1):**
```
agy -p "Remember this secret test token: P01B_CONV_RESUME_1789987948333. Reply with only ACK." --output-format json
```

**Command (Turn 2):**
```
agy -p "What was the secret test token I told you to remember in the previous message? Reply with only the token." --output-format json --conversation "ffc7c21a-5f10-40e4-ba01-2670340faf44"
```

**Evidence file:** `evidence/p01b9_conversation_id_resume.json`

**Literal Turn 1 stdout:**
```json
{"conversation_id":"ffc7c21a-5f10-40e4-ba01-2670340faf44","status":"ERROR","response":"ACK\n","error":"API error (attempt 1): RESOURCE_EXHAUSTED (code 429): Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 42h8m3s.","duration_seconds":7.072729,"num_turns":1,"usage":{"input_tokens":11753,"output_tokens":70,"thinking_tokens":69,"cache_read_tokens":0,"total_tokens":11823}}
```

**Literal Turn 2 stdout:**
```json
{"conversation_id":"ffc7c21a-5f10-40e4-ba01-2670340faf44","status":"ERROR","response":"","error":"Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 42h5m35s.","duration_seconds":125.0682173,"num_turns":2,"usage":{"input_tokens":11753,"output_tokens":70,"thinking_tokens":69,"cache_read_tokens":0,"total_tokens":11823}}
```

**Observations:**
- Turn 1 exit code: `0`; Turn 2 exit code: `3`
- Turn 1: model responded `"ACK"` but quota error triggered on API retry
- Turn 2: entirely blocked — empty `response`, quota exhausted before any API call succeeded
- `markerRecovered`: `false`
- `--conversation` flag accepted without CLI parse error ✓
- `conversation_id` correctly returned by Turn 1 ✓
- Failure is API quota, not Agy CLI capability defect

**Gap Classification:** `QUOTA_EXHAUSTED_GAP`

**Result: QUOTA_EXHAUSTED_GAP** — CLI flag proven accepted; full round-trip proof deferred pending quota reset.

---

## 5. Proven Capabilities Summary

| Capability | Status | Evidence |
|-----------|--------|---------|
| Headless text prompting (`-p`) | **PROVEN** | P01-B1 |
| JSON output envelope (`--output-format json`) | **PROVEN** | P01-B2 |
| Stream-JSON output (`--output-format stream-json`) | **PROVEN** | P01-B3 |
| Multi-turn stdin pipe (`--input-format stream-json`) | **PROVEN** | P01-B4 |
| JSON Schema structured output (`--json-schema`) | **PROVEN** | P01-B5 |
| Real filesystem side effects with schema | **PROVEN** | P01-B6 |
| Dangerously-skip-permissions (`--dangerously-skip-permissions`) | **PROVEN** | P01-B7 |
| Multi-directory context (`--add-dir`) | **PROVEN** | P01-B8 |
| Conversation ID resume (`--conversation`) | **UNCONFIRMED** (quota gap) | P01-B9 |

---

## 6. Gap Record

### GAP-P01B-001: Conversation ID Resume Unconfirmed

| Field | Value |
|-------|-------|
| **Gap ID** | `GAP-P01B-001` |
| **Classification** | `QUOTA_EXHAUSTED_GAP` |
| **Capability** | `--conversation <id>` resume across separate agy invocations |
| **Root Cause** | API quota `RESOURCE_EXHAUSTED` (429); resets ~42h after test execution |
| **CLI Behavior** | Flag accepted; `conversation_id` returned in Turn 1; Turn 2 fully blocked by quota |
| **Impact** | Cannot empirically confirm persistent cross-invocation context via `--conversation` flag |
| **Disposition** | `DEFERRED_PENDING_QUOTA_RESET` — not a blocker for P01-B overall verdict |
| **ADR Required** | No — quota gap, not architectural gap |

---

## 7. Agy 1.2.7 Behavioral Findings

1. **JSON envelope structure:** `{conversation_id, status, response, duration_seconds, num_turns, usage}`
2. **`status` field values observed:** `"SUCCESS"`, `"ERROR"`
3. **Stream-JSON event types:** `init`, `step_update`, `result`
4. **`step_type` values observed:** `user_input`, `agent_response`, `error_message`
5. **Stdin protocol:** `{"event":"user","message":{"content":"..."}}` (NDJSON line per turn)
6. **`--json-schema` requires `--output-format json` or `stream-json`**
7. **`structured_output` field populated in JSON envelope when schema applied**
8. **`--dangerously-skip-permissions` suppresses all permission prompts**
9. **`--add-dir` makes secondary directory files available to agent context**
10. **Exit code `3` observed on quota exhaustion with empty response**
11. **Exit code `0` observed even with `status:"ERROR"` when partial response was produced**

---

## 8. Overall Verdict

**P01-B = PASS**

8 of 9 capabilities empirically proven with literal evidence. GAP-P01B-001 (conversation resume) is `QUOTA_EXHAUSTED_GAP` — the `--conversation` flag is syntactically accepted by Agy 1.2.7 and `conversation_id` was correctly returned in Turn 1; full cross-invocation proof deferred pending quota reset.

**Proven sufficient for P01-C release decision upon External Audit approval.**
