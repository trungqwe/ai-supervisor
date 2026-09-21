# ADR-013: Trusted Verification Command Specification

> **Status**: PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Proposal
> **Tracking**: `P02_PRECODE_GAP_VERIFICATION_COMMAND_SPEC`
> **Related**: ADR-004 (Read-Only Interface), ADR-005 (Supervisor Authority), ADR-010 (Contract Immutability), ADR-012 (Revision & Attempt Binding)

---

## Context & Problem Statement
In the canonical architecture, the Supervisor Evidence subsystem independently validates worker claims prior to transitioning from `REPORT_READY` to `EVIDENCE_READY`. This independent evidence collection requires executing approved test and build verification specifications against the worker worktree.

However, an audit of the pre-code baseline revealed a critical security and contract representation contradiction:
1. `docs/schemas/task-contract.schema.json` and `docs/08_TASK_CONTRACT.md` currently define:
   ```json
   "required_tests": {
     "type": "array",
     "items": { "type": "string" }
   }
   ```
2. Canonical examples specify shell-interpolated command strings such as:
   `powershell -Command './tests/smoke.ps1'`
3. Conversely, `docs/07_SECURITY_MODEL.md` explicitly mandates:
   - Zero arbitrary shell access exposed to ChatGPT;
   - Execution without shell interpolation (`shell = false`);
   - Parameter passing as discrete argument arrays;
   - Execution exclusively through a constrained, allowlisted verification runner.

If `required_tests` strings are executed via a shell (`cmd.exe /c` or `powershell -Command`), arbitrary command injection becomes possible if a malicious prompt or compromised worker manipulates contract strings. Conversely, if executed without a shell, raw string commands fail because arguments, flags, quotes, and executables cannot be deterministically separated without a complex, error-prone shell tokenizer.

A safe, canonical representation for verification commands must be designed and formalized before Phase P02 code can proceed.

---

## Architectural Options Evaluated

### Option A: String Command + Shell Tokenizer / Parser + Allowlist
- **Mechanism**: Keep `required_tests: string[]`, implement a shell lexical analyzer / tokenizer (e.g., POSIX shell words / Windows CommandLineToArgvW parser), extract executable and arguments, and validate against a regex allowlist.
- **Evaluation**:
  - *Command Injection Resistance*: POOR. Windows CommandLine rules differ drastically between `cmd.exe`, PowerShell, and direct Win32 `CreateProcess`. Parsing shell metacharacters (`&`, `|`, `;`, `>`, `<`, `` ` ``, `$()`) safely across platforms is notoriously brittle.
  - *Complexity*: HIGH. Requires bundling a custom grammar/tokenizer into the Supervisor.
  - *Windows Compatibility*: PROBLEMATIC due to varied argument quoting rules across Windows runtimes.
- **Verdict**: REJECTED. Parsing shell strings is an anti-pattern when structured contracts are available.

### Option B: Fully Structured Verification Command Spec
- **Mechanism**: Replace raw string commands with a fully structured object:
  ```json
  {
    "id": "verify-unit-tests",
    "executable": "go",
    "args": ["test", "-v", "./..."],
    "cwd": "tests",
    "timeout_seconds": 120,
    "env": { "CI": "true" }
  }
  ```
- **Evaluation**:
  - *Command Injection Resistance*: EXCELLENT. Discrete array elements are passed directly to OS `execFile` / `exec.Command` / `subprocess.Popen` without shell invocation (`shell = false`). Metacharacters are treated as literal strings.
  - *Windows Compatibility*: EXCELLENT. Direct process spawn bypassing `cmd.exe` or PowerShell shell wrappers.
  - *TaskContract Schema Validation*: EXCELLENT. Clear JSON Schema validation with strict regex patterns on `executable` (e.g., allowlisting `go`, `node`, `npm`, `pytest`, `cargo`, `python`) and `args`.
  - *Auditability*: EXCELLENT. Exact command invocation is serialized unambiguously in the audit trail.
- **Verdict**: STRONGLY RECOMMENDED for general verification flexibility.

### Option C: Server-Side Named Test Profiles
- **Mechanism**: The Supervisor configuration defines pre-approved test profiles on the host machine:
  ```json
  {
    "profiles": {
      "smoke": { "executable": "node", "args": ["tests/smoke.js"] },
      "unit": { "executable": "go", "args": ["test", "./..."] }
    }
  }
  ```
  The TaskContract only specifies profile identifiers: `required_test_profiles: ["smoke", "unit"]`.
- **Evaluation**:
  - *Command Injection Resistance*: MAXIMUM. ChatGPT and workers can only reference pre-declared server profiles; zero dynamic commands are accepted.
  - *Flexibility*: LOW. Dynamic tasks that test a specific new test file or package cannot be parameterized by ChatGPT without reconfiguring the Supervisor daemon.
- **Verdict**: INSUFFICIENT for dynamic pair programming, though highly secure.

### Option D: Hybrid Named Profile with Structured Parameter Overrides
- **Mechanism**: A server-side allowlist defines base profile templates (e.g., `go-test`, `npm-test`, `pytest`), and the TaskContract specifies allowed target paths or arguments constrained by strict whitelist regexes.
- **Evaluation**:
  - Balances server-side allowlisting with task-level parameterization.
  - Higher specification complexity than Option B.
- **Verdict**: Viable alternative, but adds unnecessary abstraction layers for V1 single-user development.

---

## Proposed Direction (Option B with Host Allowlist)

The proposed design establishes a structured `VerificationCommandSpec` governed by a host allowlist policy:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "VerificationCommandSpec",
  "type": "object",
  "required": ["id", "executable", "args"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^[a-z0-9_-]{3,32}$"
    },
    "executable": {
      "type": "string",
      "enum": ["go", "node", "npm", "python", "pytest", "cargo", "git"]
    },
    "args": {
      "type": "array",
      "items": {
        "type": "string",
        "maxLength": 256
      },
      "maxItems": 32
    },
    "cwd": {
      "type": "string",
      "description": "Relative directory path strictly contained within worktree root"
    },
    "timeout_seconds": {
      "type": "integer",
      "minimum": 1,
      "maximum": 600,
      "default": 120
    }
  },
  "additionalProperties": false
}
```

### Execution Invariants:
1. **Zero Shell Invocation**: `shell` is hardcoded to `false`. Commands are executed via direct OS process spawn.
2. **Discrete Parameter Passing**: Arguments are passed as discrete array elements; no command line string concatenation is performed.
3. **Executable Allowlisting**: The `executable` field must match a strictly allowlisted binary verified to exist on the host PATH.
4. **CWD Containment**: If `cwd` is specified, it must resolve to a strict descendant of the allocated worktree root.
5. **Timeout Bounding**: Each command execution enforces a hard OS timeout with clean tree termination.
6. **Output Capture**: Captures exact exit code, stdout (bounded size), and stderr as objective ground truth.

---

## Status and Gate Condition
- **Current Status**: `PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)`.
- **Pre-Code Gate Tracking**: `P02_PRECODE_GAP_VERIFICATION_COMMAND_SPEC`.
- **Policy**: Canonical `docs/schemas/task-contract.schema.json` is **NOT modified** in this task. TaskContract schema mutation and Evidence subsystem integration will occur only after the External Supervisor formally reviews and accepts ADR-013.
- Phase P02 production code remains held pending this decision.