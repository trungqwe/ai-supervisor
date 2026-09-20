# SOURCE DOSSIER: 02 — OFFICIAL ANTIGRAVITY CLI

> **Authority**: Upstream Source Evidence Dossier  
> **Status**: Verified Documentation Baseline (Post-Re-Audit #2 Hygiene)

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
| **Local Runtime Status** | `1.2.7` Installed; Empirical Headless Proof Pending P01 | Local binary smoke test |

---

# 2. Verified Technical Claims & Source Evidence

```text
Claim ID: AGY-CLAIM-001
Claim: Antigravity CLI v1.2.7 supports non-interactive single-prompt execution and result printing via prompt flags.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: README.md & agy --help
Section / symbol: -p, --print, --prompt <prompt>
Verification: VERIFIED
Confidence: HIGH
Notes: In official documentation and binary help, -p, --print, and --prompt are aliases. Canonical non-interactive invocation syntax is agy -p "<prompt>".

Claim ID: AGY-CLAIM-002
Claim: Antigravity CLI v1.2.7 supports structured JSON schema enforcement on output.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: README.md & agy --help
Section / symbol: --json-schema <schema>
Verification: VERIFIED
Confidence: HIGH
Notes: Flag exists and specifies output JSON schema validation. Verification of exact behavior when writing files vs emitting JSON is subject to P01 Track P01-B.

Claim ID: AGY-CLAIM-003
Claim: Antigravity CLI v1.2.7 supports streaming JSON input and output format negotiation.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: README.md & agy --help
Section / symbol: --output-format text|json|stream-json / --input-format text|stream-json
Verification: VERIFIED
Confidence: HIGH
Notes: When --input-format stream-json is specified, input prompts are piped via stdin stream rather than supplied via -p.

Claim ID: AGY-CLAIM-004
Claim: Antigravity CLI supports unattended autonomous execution via permission bypass and multi-directory workspace binding.
Repository: google-antigravity/antigravity-cli
Pinned tag: 1.2.7
Pinned commit: 7bb195acaec9e7788df5210d0dc3e15f3cefc6b3
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: README.md & agy --help
Section / symbol: --dangerously-skip-permissions / --add-dir <path>
Verification: VERIFIED
Confidence: HIGH
Notes: Flags allow headless runs to access secondary roots and execute file edits and commands without interactive confirmation.
```

---

# 3. Adopted Concepts vs. Upstream Gaps

### Adopted Concepts:
- Official headless execution flags (`-p`, `--output-format`, `--input-format`, `--json-schema`).
- Unattended execution (`--dangerously-skip-permissions`).
- Worktree root mounting via `--add-dir`.

### Empirical Gaps to Prove in Phase P01 (Track P01-B & P01-C):
- Does Agy write files to disk while simultaneously emitting `--json-schema` completion payloads?
- How does AO's interactive harness (`--prompt-interactive`) interact with Agy's structured output flags?
