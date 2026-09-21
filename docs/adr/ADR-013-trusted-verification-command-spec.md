# ADR-013: Trusted Verification Command Specification & Execution Isolation Architecture

> **Status**: ACCEPTED
> **Date**: 2026-09-22
> **Authority**: Architecture Decision Record
> **Deciders**: External Supervisor, Engineering Team
> **Tracks**: SEC-003, NFR-004, FR-004, FR-008
> **Supersedes**: Legacy `required_tests: string[]` model and initial draft allowlist proposals

---

## 1. Context and Problem Statement

Canonical `docs/08_TASK_CONTRACT.md` originally specified:
```json
"required_tests": [
  "powershell -Command './tests/smoke.ps1'"
]
```
This contradicts Security Directive **SEC-003** (No Arbitrary Shell Execution):
1. **Shell Injection Vulnerabilities**: Passing raw string commands to shell interpreters (`powershell`, `cmd`, `sh`) invites command injection via chaining operators (`&`, `|`, `;`), subshells, and uncontrolled parameter interpolation.
2. **Executable-Native Code Execution Threat in Simple Allowlists**: Generic executable allowlists (e.g. allowlisting `node`, `python`, `npm`, `git`, `cargo`) are fundamentally insufficient:
   - Evaluator flags allow arbitrary code execution (`node -e "..."`, `python -c "..."`).
   - Package runners can download and execute arbitrary code (`npm exec`, `npx`).
   - Version control tools can invoke external commands via configuration flags or hooks (`git -c core.fsmonitor=...`).
   - Resolving executables via worker-controlled `cwd` or mutable `PATH` allows binary substitution and DLL hijacking.
3. **Internal vs External Separation**: Internal Git evidence collection (`git diff`, `git status`) is fixed, internal Supervisor functionality. It must never be exposed as an arbitrary user-controllable test runner executable.

---

## 2. Decision: Two-Layer Security Model (HYBRID_PROFILE_STRUCTURED_PARAMETERS)

The Supervisor Control Plane adopts the **Hybrid Profile with Structured Parameters** model across two decoupled security layers.

### Layer A — Command Authority Containment
Command authority is owned exclusively by host configuration and project policy, completely separating command construction from worker-controlled contracts:
- **Host `ProjectPolicy` Owns**:
  - Profile definitions (`profile_id`).
  - Trusted executable identity (resolved to immutable absolute paths at Supervisor startup via a Host-Owned Executable Registry; never resolved via mutable `PATH` or worker `cwd`).
  - Fixed command-family semantics and immutable argument prefixes.
  - Allowed parameter JSON schemas.
  - Allowed target and path patterns.
  - Hard timeout ceilings and output capture limits.
  - Strict environment variable policies and network access permissions.
- **TaskContract Supplies Only**:
  - Verification request identifier (`id`).
  - Target profile identifier (`profile_id`).
  - Strictly typed, structured parameters (`parameters: object`).
  - Relative working directory (`cwd`), constrained within the worktree.
  - Requested execution timeout (`timeout_seconds`), bounded by profile limits.
- **Strict Invariants**:
  - **No shell interpreters** (`shell = false`).
  - **No PATH-selected executables**.
  - **No arbitrary argument vectors** (`args`).
  - **No generic process launcher authority**.

### Layer B — Untrusted Code Execution Isolation
Command containment (Layer A) prevents shell injection and arbitrary binary invocation, but does **not** make code execution safe:
```
SAFE_PROFILE != TRUSTED_CODE
```
Tools such as `go test`, `pytest`, and `npm test` execute repository-controlled code written by AI workers. Therefore, the future VerificationRunner (implemented in Phase P04) must execute all verification commands inside a least-privilege execution boundary:
1. **Zero Secret Exposure**: The verification execution environment must strictly contain zero Supervisor secrets, zero OpenAI API keys, zero AO credentials, and zero tunnel credentials. Secrets are excluded by construction.
2. **Filesystem Boundary**:
   - Access to arbitrary user directories (`USERPROFILE`, `HOME`) is denied.
   - Filesystem visibility is constrained strictly to the assigned workspace worktree.
   - `TEMP` and `TMP` directories are redirected to isolated, attempt-local transient directories cleaned up after execution.
   - Language and module caches are isolated or strictly read-only.
3. **Network Boundary**:
   - Outbound and inbound network access is denied by default.
   - Explicit bounded network access is granted only if specifically permitted by host profile policy.
4. **Process & Resource Containment**:
   - Execution is constrained by hard wall-clock timeouts.
   - stdout and stderr streams are capped (e.g. 1 MB maximum) to prevent buffer exhaustion DoS.
   - On Windows, child processes are assigned to a **Windows Job Object** to guarantee deterministic process-tree lifecycle containment and prevent orphaned background processes upon timeout, crash, or cancellation.
   - *Architecture Note*: A Windows Job Object provides process lifecycle and resource limit containment; it does **not** by itself provide filesystem sandboxing, network isolation, or credential stripping. The full implementation mechanism for Windows execution isolation remains a Phase P04 research deliverable.

---

## 3. Environment Policy

Verification child processes execute with an aggressively stripped environment:
- **Prohibited Variables**: Under no circumstances may `USERPROFILE`, `HOME`, `NODE_PATH`, `GOPATH`, `TEMP`, `TMP`, or any authentication tokens / credentials be broadly inherited from the host or Supervisor daemon.
- **Canonical Child Environment**:
  ```
  Child Environment =
      Minimum OS startup variables (SYSTEMROOT, WINDIR, COMSPEC where required)
    + Trusted host-owned toolchain paths (GOROOT)
    + Attempt-local temporary directories (TEMP, TMP)
    + Profile-specific explicit allowlist of non-sensitive variables
  ```

---

## 4. Profile-Specific Semantic Precision

1. **`go-test` Profile**:
   - Fixed executable: Trusted absolute path to `go.exe` from host registry.
   - Fixed argument prefix: `["test"]`.
   - Parameter constraints: Target packages must begin with `./` and resolve within the worktree. Allowed flags are enumerated (`-v`, `-race`, `-count=1`, `-run`). Flags allowing external command execution (`-exec`, `-toolexec`) are rejected.
   - Untrusted code execution: Executes repository Go code under Layer B isolation.
2. **`npm-test` Profile**:
   - Executable identity: Must **not** use `npm.cmd` directly as a generic safe executable, because on Windows `.cmd` files trigger cmd.exe batch execution. When implemented, the profile must invoke trusted `node.exe` passing a trusted absolute path to the npm CLI JavaScript entrypoint.
   - Fixed prefix: `["test", "--"]`.
   - Arbitrary package installation or `npm exec` is strictly prohibited. Executes package scripts under Layer B isolation.
3. **`pytest` Profile**:
   - Fixed executable: Absolute path to virtualenv Python interpreter.
   - Fixed prefix: `["-m", "pytest"]`.
   - Executes repository Python code under Layer B isolation.
4. **Git Evidence Isolation**:
   - `git` is **never** exposed as a TaskContract verification profile. Internal Git evidence collection remains fixed Supervisor functionality owned by `EvidenceCollector`.

---

## 5. Contract Schema & Two-Layer Validation Model

### 5.1 Serialized Verification Request Structure
Legacy `required_tests: string[]` is migrated to `verification_requests`:
```json
{
  "id": "smoke-test",
  "profile_id": "go-test",
  "parameters": {
    "package": "./pkg/statemachine/...",
    "run_pattern": "TestAttemptLineage",
    "flags": ["-v", "-count=1"]
  },
  "cwd": ".",
  "timeout_seconds": 60
}
```

### 5.2 Two-Stage Validation & VerificationPolicyCatalog Boundary

To maintain complete architectural decoupling between Phase P02 (domain contract validation) and Phase P04 (concrete execution runner):

```
JSON SCHEMA SHAPE VALIDATION != PATH CONTAINMENT VALIDATION
```

1. **Stage A — Generic JSON Schema Validation (`task-contract.schema.json`)**:
   - Validates structural shape: `id` and `profile_id` format, `parameters` is an object, `cwd` is an optional non-empty string bounded to 256 characters, `timeout_seconds` is an integer bounded by `[1, 3600]`.
   - Rejects extraneous properties (`additionalProperties: false`), forbidding `executable`, `command`, `shell`, `args`, or `environment`.
   - Makes **zero claim** of filesystem containment or safety.

2. **Stage B — Semantic Validation (`TaskContractValidator` in Phase P02)**:
   - Validates `cwd`: Rejects absolute paths, Windows volume/UNC syntax, and `..` traversals that escape the assigned workspace root. The resolved directory must be either exactly the assigned worktree root or a descendant contained within the worktree (`TARGET == ROOT || TARGET_IS_DESCENDANT_OF_ROOT`).
   - Depends exclusively on a pure domain abstraction: **`VerificationPolicyCatalog`**:
     ```go
     type VerificationPolicyCatalog interface {
         LookupProfile(profileID string) (VerificationProfilePolicy, bool)
     }
     ```
   - **`VerificationProfilePolicy`** exposes only validation metadata required by Phase P02:
     * `ProfileID`: Identifier string;
     * `ParameterSchema`: Schema specification defining permitted parameter properties;
     * `CwdPolicy`: Constraints on allowed relative working directories;
     * `MaxTimeoutSeconds`: Upper bound ceiling on allowable execution time;
     * `AllowedCapabilities`: Declarative capability flags.
   - It strictly **does NOT expose** arbitrary executable paths, raw argument templates, or shell options to TaskContract.
   - P02 unit tests remain fully testable using an in-memory mock/fake `VerificationPolicyCatalog`.

3. **Execution Phase (Phase P04)**:
   - Concrete `VerificationRunner` implementation;
   - Trusted executable resolution and host registry binding;
   - Immutable argument prefix construction;
   - Windows Job Object assignment and execution isolation boundary enforcement.

---

## 6. Consequences

### Positive:
- Total elimination of shell injection vectors across all contracts (SEC-003).
- Strict separation between contract request parameters and host execution authority.
- Clear architectural boundary establishing that verification tools execute untrusted code requiring Layer B isolation.
- Deterministic, machine-validatable JSON Schema contract representation.

### Negative / Tradeoffs:
- Repository test suites must map to predefined host profiles. Ad-hoc arbitrary test scripts cannot be executed without a supporting profile in `ProjectPolicy`.
