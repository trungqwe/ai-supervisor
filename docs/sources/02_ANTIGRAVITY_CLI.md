# SOURCE DOSSIER: 02 -- OFFICIAL ANTIGRAVITY CLI

> **Authority**: Upstream Source Evidence Dossier
> **Status**: Partially Runtime Tested (P01-B accepted empirical subset B1-B8 PASS; B9/B10 NOT EVALUATED)

---

# 1. Source Identification & Verification

| Metadata Field | Authoritative Value | Evidence Source |
|---|---|---|
| **Repository Name** | `google-antigravity/antigravity-cli` | Official Google Distribution |
| **Role in Architecture** | Primary Worker Runtime (Autonomous coding agent in isolated worktree) | ADR-003, docs/04_ARCHITECTURE.md |
| **Pinned Documentation Version** | `1.2.7` | GitHub Release & Local CLI `agy --version` |
| **Pinned Commit SHA** | `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3` | GitHub Commit Verification |
| **Commit Author Date (UTC)** | `2026-09-19T01:01:48Z` | GitHub Commit Metadata |
| **Commit Committer Date (UTC)** | `2026-09-19T01:01:48Z` | GitHub Commit Metadata |
| **Repository SPDX License** | `NOT DECLARED` | GitHub Repository Metadata |
| **Usage Terms** | Subject to applicable Google / Antigravity Terms of Service | Official Distribution Terms |
| **Local Runtime Status** | `PARTIALLY_RUNTIME_TESTED` (Agy 1.2.7; P01-B empirical subset B1-B8 PASS; B9/B10 NOT EVALUATED pending quota recovery) | P01-B dossier: `docs/audits/P01_B_AGY_CLI_PROOF.md` |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: AGY-CLAIM-001
Claim: Antigravity CLI v1.2.7 supports non-interactive single-prompt execution and result printing via prompt flags.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC + RUNTIME_PROOF
Exact evidence: README.md & agy --help & P01-B1 (marker echo, exit 0)
Section / symbol: -p, --print, --prompt <prompt>
Verification: VERIFIED
Runtime status: RUNTIME_TESTED_PASS
Notes: In official documentation and binary help, -p, --print, and --prompt are aliases. Canonical
       non-interactive invocation syntax is agy -p "<prompt>". P01-B1 confirmed marker echo with exit 0.

Claim ID: AGY-CLAIM-002
Claim: Antigravity CLI v1.2.7 supports structured JSON schema enforcement on output.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC + RUNTIME_PROOF
Exact evidence: README.md & agy --help & P01-B5 (structured_output field confirmed) & P01-B6 (file + schema)
Section / symbol: --json-schema <schema>
Verification: VERIFIED
Runtime status: RUNTIME_TESTED_PASS
Notes: P01-B5 confirmed structured_output field in JSON envelope. P01-B6 confirmed simultaneous file
       creation and structured output emission. Flag requires --output-format json or stream-json.

Claim ID: AGY-CLAIM-003
Claim: Antigravity CLI v1.2.7 supports streaming JSON input and output format negotiation.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC + RUNTIME_PROOF
Exact evidence: README.md & agy --help & P01-B3 (stream-json output) & P01-B4 (stream-json input multi-turn)
Section / symbol: --output-format text|json|stream-json / --input-format text|stream-json
Verification: VERIFIED
Runtime status: RUNTIME_TESTED_PASS
Notes: P01-B3 confirmed NDJSON event sequence (init/step_update/result). P01-B4 confirmed stdin NDJSON
       accepted and in-process multi-turn context retained. Cross-process conversation resume NOT proven
       (P01-B9 ENV-P01B-001 quota blocked).

Claim ID: AGY-CLAIM-004
Claim: Antigravity CLI supports unattended autonomous execution via permission bypass and multi-directory workspace binding.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC + RUNTIME_PROOF
Exact evidence: README.md & agy --help & P01-B7 (skip-permissions) & P01-B8 (add-dir)
Section / symbol: --dangerously-skip-permissions / --add-dir <path>
Verification: VERIFIED
Runtime status: RUNTIME_TESTED_PASS
Notes: P01-B7 confirmed file write without permission prompt. P01-B8 confirmed out-of-band marker in
       secondary directory was read successfully by the agent.
```

---

# 3. Adopted Concepts vs. Upstream Gaps

### Proven (P01-B empirical subset B1-B8):
- Headless text execution (`-p`, `-p --output-format json`, `-p --output-format stream-json`)
- Stream-JSON stdin input protocol (`--input-format stream-json`)
- Same-process multi-turn context retention
- JSON Schema structured output (`--json-schema`)
- Real filesystem side effects combined with structured output
- Permission bypass (`--dangerously-skip-permissions`)
- Multi-directory workspace binding (`--add-dir`)

### Not Yet Proven (pending quota recovery -- ENV-P01B-001):
- Cross-process conversation resume (`--conversation <id>`) -- P01-B9
- Workspace continue (`--continue`, `-c`) -- P01-B10

### Remaining Questions for P01-C (HELD):
- Does AO's interactive harness (`--prompt-interactive`) interact cleanly with Agy's structured output flags?
- Can the existing AO-Agy integration satisfy the WorkerReport contract without custom adapter code?