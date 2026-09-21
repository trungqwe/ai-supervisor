# 07. SECURITY & CONTAINMENT MODEL

> **Focus**: Trust Boundaries, Read-Only Enforcement, Path Containment & Threat Modeling
> **Status**: Approved Baseline (Updated Architecture V2.1 / Reaudit 001)

---

# 1. Security Architecture Principles

1. **Readonly Supervisor by Default**:
   - The tool interface provided to ChatGPT has **zero ability to write to project source files directly**.
   - ChatGPT acts as an auditor and planner, not a code editor.
   - Task approval records the review decision only; it does NOT trigger automatic branch merges, commits, or pushes.
2. **Explicit Workspace Containment**:
   - All filesystem operations are restricted strictly to registered workspace boundaries.
   - Canonical containment algorithm:
     1. Canonicalize and resolve registered project root;
     2. Canonicalize and resolve requested target path;
     3. Ensure target remains a strict descendant of the registered root;
     4. Reject relative path traversal (`../`);
     5. Reject absolute path escape outside root boundaries;
     6. Handle unsafe symlink, directory junction, or reparse-point escapes according to the selected implementation platform and runtime library.
   - **Two-Stage Path Containment Architecture**:
     * **Stage A (Structural JSON Schema)**: `task-contract.schema.json` enforces string types and bounded lengths (`minLength: 1`, `maxLength: 256`) on `cwd` and scope arrays. Generic schema validation makes zero claim of filesystem containment.
     * **Stage B (Domain Semantic Validation in P02)**: `TaskContractValidator` evaluates paths against the assigned workspace worktree: rejecting absolute paths, Windows drive/UNC syntax, and `..` traversals that escape the assigned root, ensuring target directories strictly resolve as descendants of the worktree root.
     * **Rule**: `JSON SCHEMA SHAPE VALIDATION != PATH CONTAINMENT VALIDATION`.
3. **No Arbitrary Shell or Terminal Primitives for ChatGPT**:
   - Neither ChatGPT nor external web requests can execute arbitrary bash, PowerShell, or shell commands.
   - ChatGPT is provided with zero shell or terminal command execution primitives.
4. **Constrained Verification Runner for Supervisor Evidence**:
   - Independent test verification by the Supervisor Evidence subsystem executes exclusively through a constrained, allowlisted verification runner.
   - The runner executes only approved host-owned verification profiles (`verification_requests` per ADR-013) within an execution isolation boundary.
   - Arguments are passed as discrete parameter arrays without uncontrolled shell string interpolation.
   - Captures process exit codes, stdout, stderr, and test artifact hashes as objective ground truth.
5. **Sanitized Audit Trail**:
   - Sensitive tokens (`gho_*`, `Bearer *`, `.env` secrets, API keys) are deterministically scrubbed before logging or transmitting to ChatGPT.

---

# 2. Threat Modeling & Mitigations

| Threat ID | Threat Description | Attack Vector | Mitigation Strategy |
|---|---|---|---|
| **THR-001** | Malicious Prompt Injection | Third-party dependencies or issues contain instructions directing the AI to exfiltrate data. | Supervisor tool surface is narrow and readonly; no network exfiltration tools exist. |
| **THR-002** | Worker Scope Creep | Coding agent modifies files outside its assigned task (e.g., config, credentials, architecture). | Pre-commit and post-execution Git diff checks enforce `allowed_scope` and flag `PolicyViolation`. |
| **THR-003** | Path Traversal & Escape | Malicious inputs query files outside workspace (`../../etc/passwd`, `C:\Windows`). | Canonical path containment algorithm: canonicalize root and target, verify target is descendant of root, reject traversal, absolute escapes, and reparse-point escapes. |
| **THR-004** | Command Injection in Tests | Malicious task payload injects shell delimiters into test command fields. | Supervisor verification runner executes commands without shell interpolation, passing arguments as discrete parameter arrays; never exposes generic shell to ChatGPT. |
| **THR-005** | Cross-Project Data Leakage | Multiple workspaces share memory or un-isolated sessions. | `Pair` model strictly isolates project IDs; each project uses dedicated Git worktrees and AO sessions. |
| **THR-006** | Stale Pair Binding | An old ChatGPT session commands a newly reassigned worker. | Ephemeral `session_token` validated on every tool invocation; mismatch yields `PAIR_MISMATCH_ERROR`. |
| **THR-007** | Local IPC Abuse | External software on host machine calls Supervisor local daemon. | Daemon listens strictly on loopback (`127.0.0.1`) with a generated local authorization token. |
| **THR-008** | Secret Exposure | Worker outputs environment secrets in execution reports. | Regular expression secret scanners redact common token patterns before assembling Review Bundles. |
