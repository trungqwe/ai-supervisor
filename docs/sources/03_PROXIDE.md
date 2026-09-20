# SOURCE DOSSIER: 03 — PROXIDE

> **Authority**: Passive Design Source Evidence Dossier  
> **Status**: Verified Documentation Baseline (Post-Re-Audit #2 Hygiene)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `tt-a1i/proxide` | GitHub API |
| **Role in Architecture** | Passive Design Reference (Readonly default security, containment, mode boundaries) | ADR-004, ADR-005, docs/07_SECURITY_MODEL.md |
| **Pinned Commit SHA** | `c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee` | GitHub API Verification |
| **Commit Author Date (UTC)** | `2026-06-20T15:17:16Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-06-20T15:17:16Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `MIT` | `LICENSE` in repository root |
| **Usage Terms** | MIT License | `LICENSE` file |
| **Operational Nature** | Passive Design Source (Never imported as runtime package) | Project Governance |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: PROX-CLAIM-001
Claim: Proxide establishes a secure default trust boundary where the connector defaults to readonly mode and enforces explicit project roots.
Repository: tt-a1i/proxide
Pinned tag: None
Pinned commit: c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SECURITY.md
Section / symbol: ## Safer Defaults
Verification: VERIFIED
Confidence: HIGH
Notes: Explicitly states "The connector defaults to trust_level=readonly" and "Allowed roots must be explicit project paths, not broad directories such as / or ~".

Claim ID: PROX-CLAIM-002
Claim: Proxide enforces strict workspace path containment, rejecting absolute paths, path traversal (..), and symlink escapes.
Repository: tt-a1i/proxide
Pinned tag: None
Pinned commit: c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SECURITY.md
Section / symbol: ## Safer Defaults
Verification: VERIFIED
Confidence: HIGH
Notes: Explicitly specifies "Workspace paths are containment-checked and reject absolute paths, .., and final symlink escapes".

Claim ID: PROX-CLAIM-003
Claim: Proxide defines discrete mode boundaries separating tool visibility, file mutation, and shell execution.
Repository: tt-a1i/proxide
Pinned tag: None
Pinned commit: c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SECURITY.md
Section / symbol: ## Mode Boundaries & ## Hard Rules
Verification: VERIFIED
Confidence: HIGH
Notes: Formalizes tool_mode (minimal|standard|full), write_mode (off|handoff|workspace), and shell_mode (off|safe|full) guards.
```

---

# 3. Adopted Concepts vs. Rejected Elements

### Adopted Concepts:
- Readonly Supervisor principle (Supervisor possesses zero file-mutation or shell tools).
- Explicit workspace root containment and directory traversal rejection.
- Audit trail secret sanitization (omitting sensitive file bodies and credentials).

### Rejected Elements:
- Do NOT adopt Proxide's multi-tenant proxy server or Rust daemon implementation.
- Do NOT adopt Proxide's interactive terminal web viewer.
