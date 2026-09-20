# 07. SECURITY & CONTAINMENT MODEL

> **Focus**: Trust Boundaries, Read-Only Enforcement, Path Containment & Threat Modeling  
> **Status**: Approved Baseline

---

# 1. Security Architecture Principles

1. **Readonly Supervisor by Default**:
   - The tool interface provided to ChatGPT has **zero ability to write to project source files directly**.
   - ChatGPT acts as an auditor and planner, not a code editor.
2. **Explicit Workspace Containment**:
   - All filesystem reads are restricted to the registered `root_path`.
   - Path traversal attempts (`../`, symlink attacks) are strictly rejected.
3. **No Arbitrary Shell or Terminal Access**:
   - Neither ChatGPT nor external web requests can execute arbitrary bash/PowerShell commands.
   - All code execution occurs via predefined test commands executed by the Worker in isolated AO worktrees.
4. **Sanitized Audit Trail**:
   - Any sensitive tokens (`gho_*`, `Bearer *`, `.env` keys) are scrubbed before logging or transmitting to ChatGPT.

---

# 2. Threat Modeling & Mitigations

| Threat ID | Threat Description | Attack Vector | Mitigation Strategy |
|---|---|---|---|
| **THR-001** | Malicious Prompt Injection | Third-party dependencies or issues contain instructions directing the AI to exfiltrate data. | Supervisor tool surface is narrow and readonly; no network exfiltration tools exist. |
| **THR-002** | Worker Scope Creep | Coding agent modifies files outside its assigned task (e.g., config, credentials, architecture). | Pre-commit and post-execution Git diff checks enforce `allowed_scope` and flag `PolicyViolation`. |
| **THR-003** | Path Traversal | Malicious inputs query files outside workspace (`../../etc/passwd`, `C:\Windows`). | Strict canonical path validation (`os.path.commonpath`) against registered project root. |
| **THR-004** | Command Injection in Tests | Malicious task payload injects shell delimiters into test command fields. | Test commands are executed without shell interpolation, passing arguments as discrete parameter arrays. |
| **THR-005** | Cross-Project Data Leakage | Multiple workspaces share memory or un-isolated sessions. | `Pair` model strictly isolates project IDs; each project uses dedicated Git worktrees and AO sessions. |
| **THR-006** | Stale Pair Binding | An old ChatGPT session commands a newly reassigned worker. | Ephemeral `session_token` validated on every tool invocation; mismatch yields `PAIR_MISMATCH_ERROR`. |
| **THR-007** | Local IPC Abuse | External software on host machine calls Supervisor local daemon. | Daemon listens strictly on loopback (`127.0.0.1`) with a generated local authorization token. |
| **THR-008** | Secret Exposure | Worker outputs environment secrets in execution reports. | Regular expression secret scanners redact common token patterns before assembling Review Bundles. |
