# ADR-013: Trusted Verification Command Specification & Security Architecture

> **Status**: PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Deciders**: External Supervisor, Engineering Team
> **Tracks**: SEC-003, NFR-004, FR-004, FR-008
> **Replaces**: Initial draft of ADR-013 (Option B allowlist model invalidated by security audit)

---

## 1. Context and Problem Statement

Canonical `docs/08_TASK_CONTRACT.md` previously specified:
```json
"required_tests": [
  "powershell -Command './tests/smoke.ps1'"
]
```
This contradicts Security Directive **SEC-003** (No Arbitrary Shell Execution):
1. **Shell Injection Vulnerabilities**: Passing raw string commands to a shell interpreter (`powershell`, `cmd`, `sh`) invites command injection via chaining operators (`&`, `|`, `;`), backticks, subshells, and uncontrolled parameter interpolation.
2. **Executable-Native Code Execution Flaw in Option B**: An earlier draft proposed `shell: false` with a simple executable allowlist (`node`, `python`, `npm`, `git`, `cargo`). The External Supervisor security audit proved this is **insufficient**:
   - Allowlisted executables possess built-in evaluation flags that execute arbitrary code (e.g. `node -e '...'`, `python -c '...'`).
   - Package managers support arbitrary package downloading/execution (e.g. `npm exec`, `npx`).
   - Version control tools can invoke external commands via config/hooks (e.g. `git -c core.fsmonitor=...`).
   - Resolving executables via worker-controlled `cwd` or mutable `PATH` allows binary spoofing and DLL hijacking.
3. **Supervisor vs Worker Authority Boundary**: Internal Git evidence collection (`git diff`, `git status`) is fixed, internal Supervisor functionality. It must never be exposed as an arbitrary user-controllable test runner executable.

---

## 2. Decision Candidates Evaluated

### Option A: String Command + Shell Parser + Allowlist
- Parse shell strings into AST, validate AST nodes, reject chained commands.
- *Verdict*: **REJECTED**. Fragile across Windows quoting, PowerShell alias expansions, and cmd.exe edge cases.

### Option B: Structured Executable + Arguments Array with Generic Allowlist
- Pass discrete `args: string[]` with `shell: false` against an allowlist of binary names (`node`, `python`).
- *Verdict*: **REJECTED**. Fails to prevent executable-native arbitrary execution flags (`-e`, `-c`) and is vulnerable to mutable `PATH` resolution.

### Option C: Fixed Project Test Profiles (Server-Side Only)
- The `TaskContract` specifies only a profile identifier (e.g. `"smoke"`, `"unit"`); all command details are static on the host.
- *Verdict*: Highly secure, but lacks flexibility for targeted test execution (e.g. running a specific test file or package).

### Option D: Host-Owned Verification Profiles with Structured Typed Parameters (HYBRID)
- Host `ProjectPolicy` defines immutable verification profiles with validated absolute executable paths and fixed argument prefixes.
- `TaskContract` supplies strictly typed, schema-validated parameters.
- *Verdict*: **RECOMMENDED**. Solves all security and authority containment requirements.

---

## 3. Recommended Security Architecture: Option D

### 3.1 Trusted Executable Identity & Registry
Executables must **never** be resolved from worker-controlled `cwd` or mutable system `PATH`.
The Supervisor maintains a **Host-Owned Executable Registry** resolved at Supervisor startup:
```
go-test     → C:\Program Files\Go\bin\go.exe (validated SHA-256 / absolute path)
node-runner → C:\Program Files\nodejs\node.exe
python-env  → C:\Users\Admin\AppData\Local\Programs\Python\Python311\python.exe
```
Worker workspaces cannot redefine this mapping.

### 3.2 Host-Owned Verification Profiles (`ProjectPolicy`)
Each profile defines strict command-family semantics:

1. **`go-test` Profile**:
   - `trusted_executable`: Absolute `go.exe` path from registry.
   - `fixed_args_prefix`: `["test"]`
   - `allowed_parameters`:
     - `package`: String conforming to `^\./[a-zA-Z0-9_./-]+$` (must resolve inside workspace worktree).
     - `run_pattern`: Optional regex conforming to `^[a-zA-Z0-9_./-]+$`.
     - `timeout_seconds`: Integer bounded by `[5, 300]`.
     - `flags`: Enum allowlist only (`["-v", "-race", "-count=1"]`).
   - *Forbidden*: Flags such as `-exec`, `-toolexec`, or arbitrary compiler arguments are rejected at contract validation.

2. **`npm-test` Profile**:
   - `trusted_executable`: Absolute `npm.cmd` or `node.exe` path.
   - `fixed_args_prefix`: `["test", "--"]`
   - `allowed_parameters`: Approved test target scripts; arbitrary `npm exec` or package injection is strictly prohibited.

3. **`pytest` Profile**:
   - `trusted_executable`: Absolute path to virtualenv Python interpreter.
   - `fixed_args_prefix`: `["-m", "pytest"]`
   - `allowed_parameters`: Relative file path within workspace `tests/` directory; allowed flags limited to `["-v", "-k", "-q"]`.

4. **Internal Git Isolation**:
   - `git` is **never** registered as a user-accessible verification executable. Git operations for diff and hash verification remain internal to `EvidenceCollector`.

### 3.3 Strict Environment Policy
Child verification processes run under a host-controlled environment policy:
- **Minimal Inherited Environment**: Only OS-essential variables (`SYSTEMROOT`, `WINDIR`, `TEMP`).
- **Explicit Allowlist**: Only host-approved development variables (`GOROOT`, `GOPATH`, `NODE_PATH`).
- **Zero Secret Forwarding**: Host API tokens, tunnel credentials, or worker session tokens are strictly excluded from child process environments.

### 3.4 Verification Request Structure in TaskContract
```json
{
  "verification_requests": [
    {
      "id": "verify-state-machine",
      "profile_id": "go-test",
      "parameters": {
        "package": "./pkg/statemachine/...",
        "run_pattern": "TestAttemptLineage",
        "flags": ["-v", "-count=1"]
      },
      "cwd": ".",
      "timeout_seconds": 60
    }
  ]
}
```

---

## 4. Operational Invariants

1. **Direct Child Process Execution**: Executed strictly via OS process APIs (`execFile` / `os/exec.Command`) with `shell = false`.
2. **Workspace Containment**: The execution `cwd` must be verified to reside within the task's assigned worktree. Directory traversal (`..`) is rejected.
3. **Timeout & Resource Limits**: Every verification process is bound to a hard timeout. On Windows, processes are assigned to a **Windows Job Object** to guarantee that child process trees are terminated upon timeout or cancellation (OPS-002).
4. **Output Capture Bounds**: stdout and stderr streams are capped (e.g. 1 MB maximum) to prevent memory exhaustion DoS.

---

## 5. Schema Alignment Status

- **Status**: This ADR is **PROPOSED (PENDING_EXTERNAL_SUPERVISOR_APPROVAL)**.
- **Contract Schema Guard**: `task-contract.schema.json` remains **unmodified** in this phase.
- **Tracking Flag**: The gap between canonical `required_tests: string[]` and this hardened specification is tracked as:
  `P02_PRECODE_GAP_VERIFICATION_COMMAND_SPEC`.
- Implementation of this schema change is blocked until the External Supervisor formally approves ADR-013.
