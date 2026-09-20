# SOURCE DOSSIER: 02 — OFFICIAL ANTIGRAVITY CLI (`agy`)

## 1. Metadata
- **Repository / Binary**: `google/antigravity` (Official CLI distribution)
- **Role in Architecture**: Active External Worker Interface (Primary Worker)
- **Inspected Version**: Official Latest Release
- **Inspection Date**: 2026-09-20
- **License**: Google Proprietary / Developer Terms of Service
- **Execution Mode**: Headless CLI (`--headless --mode autonomous`)

---

# 2. Capabilities Evaluated & Adopted
- **Autonomous Implementation**: Capable of reading task context, editing source files, and executing builds.
- **Headless Execution**: Programmatic launch without desktop UI or graphical window dependencies.
- **Structured JSON Output**: Streaming execution events and final completion reports.

---

# 3. Capabilities Explicitly Rejected
- **Direct Architecture Modification**: The worker is forbidden from altering `docs/` or project charter files.
- **Self-Triggered Scope Expansion**: Worker cannot alter files outside `allowed_scope`.
- **GUI Window Interception**: We do not drive the Antigravity desktop Electron UI.

---

# 4. Integration Strategy & Upstream Boundary
- **Strategy**: `UPSTREAM` via AO Harness Adapter.
- **Boundary**: AO invokes `agy` inside the designated worktree; Supervisor receives normalized outputs.
- **What We Must NOT Rebuild**: Autonomous code synthesis, compiler diagnostics, or language server protocol integration.

---

# 5. SOURCE EVIDENCE

### Evidence Item 2.1: Headless Autonomous Execution Mode
- **Claim**: Antigravity CLI supports headless, non-interactive execution suitable for daemon invocation.
- **Repository / Distribution**: Google Antigravity Official CLI (`agy`)
- **Reference**: Official CLI release documentation (`agy --help`)
- **Source Module**: CLI runtime options (`--headless`, `--output-format=json`)
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 2.2: Structured Execution Reporting
- **Claim**: `agy` outputs structured JSON logs containing touched files and executed test commands.
- **Repository / Distribution**: Google Antigravity Official CLI (`agy`)
- **Reference**: CLI harness integration guides
- **Source Module**: Structured logger / JSON report output
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
