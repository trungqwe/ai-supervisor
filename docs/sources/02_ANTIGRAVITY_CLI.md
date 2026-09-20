# SOURCE DOSSIER: 02 — OFFICIAL ANTIGRAVITY CLI

## 1. Metadata
- **Repository / Distribution**: `google-antigravity/antigravity-cli`
- **Role in Architecture**: Active External Worker Interface (Primary Worker)
- **Pinned Documentation Version**: `1.2.7`
- **Pinned Commit**: `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`
- **Commit Date**: 2026-03-12T18:34:01Z
- **License**: Google Proprietary / Developer Terms of Service
- **Local Host Installed Version**: `1.2.7` (Verified via `agy --version`)

---

# 2. Capabilities Evaluated & Adopted
- **Headless Non-Interactive Print Mode**: `--print` / `-p` flag executes a single turn and prints output without launching TUI.
- **Input / Output Format Negotiation**: Supports `--output-format (text, json, stream-json)` and `--input-format (text, stream-json)`.
- **Structured Schema Enforcement**: `--json-schema` enforces JSON schema validation on the final turn.
- **Unattended Permission Handling**: `--dangerously-skip-permissions` auto-approves tool permission requests.
- **Workspace Confinement**: `--add-dir` sets explicit directories added to workspace.
- **Conversation Continuity**: `--conversation <id>` or `--continue` / `-c` resumes existing conversation.

---

# 3. Capabilities Explicitly Rejected & Corrected Claims
- **Rejected Synthetic Flags**: Fictional flags previously claimed (`--mode autonomous`, `--input-contract`, `--output-report`) DO NOT EXIST in official Agy.
- **Supported Modes**: The actual `--mode` flag only accepts `(accept-edits, plan)`.
- **No Native WorkerReport Generation**: Agy does not natively emit our proprietary `WorkerReport` schema; it enforces schemas only on its response payload when `--json-schema` is passed.

---

# 4. Integration Strategy & Upstream Boundary
- **Strategy**: `UPSTREAM` via AO Harness Adapter.
- **Boundary**: Dispatched within isolated Git worktrees.
- **What We Must NOT Rebuild**: Autonomous code synthesis, compiler diagnostics, tool-calling loop.

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-AGY-001
Claim: Official Antigravity CLI supports non-interactive single-prompt execution mode.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_PRODUCT_DOC
Exact evidence: agy --help
Section / symbol: --print, -p ("Run a single prompt non-interactively and print the response")
Verification: VERIFIED
Confidence: HIGH
Notes: Verified empirically on local host binary version 1.2.7.

Claim ID: CLM-AGY-002
Claim: Official Antigravity CLI supports structured JSON output and schema validation.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_PRODUCT_DOC
Exact evidence: agy --help
Section / symbol: --output-format (text, json, stream-json), --json-schema
Verification: VERIFIED
Confidence: HIGH
Notes: Schema enforcement applies to the final response turn.

Claim ID: CLM-AGY-003
Claim: Official Antigravity CLI supports unattended tool permission auto-approval.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_PRODUCT_DOC
Exact evidence: agy --help
Section / symbol: --dangerously-skip-permissions ("Auto-approve all tool permission requests without prompting")
Verification: VERIFIED
Confidence: HIGH
Notes: Required for headless non-interactive execution inside isolated worktrees.
