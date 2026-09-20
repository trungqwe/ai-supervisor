# PHASE 0 EXTERNAL AUDIT FINDINGS

> **Auditor**: External Independent Auditor  
> **Date**: 2026-09-20  
> **Baseline Commit Audited**: `6af79a2a1141179ba06f4daa167836289663ed70`  
> **External Verdict**: `PHASE0_BLOCKED`

---

# 1. Executive Summary

An independent external audit of the Phase 0 baseline commit (`6af79a2`) identified multiple critical discrepancies between claimed technical facts and verified upstream repository evidence.

While the structural organization, governance model, and no-code enforcement were recognized as sound, the integrity of upstream provenance, source evidence citations, and version pinning was found to be compromised by synthetic or unverified assumptions.

The project is placed in **`PHASE0_BLOCKED`** status until all findings are remediated and independently re-audited.

---

# 2. Authoritative External Audit Blockers

### Finding EXT-001: Agent Orchestrator Version Pinning Inaccuracy
- **Observation**: Documentation states AO `v0.13.0` resolves to commit `e8f4a1c`.
- **Finding**: Tag `v0.13.0` on `Untrivial-ai/agent-orchestrator` actually resolves to commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`.
- **Severity**: BLOCKER.
- **Required Action**: Independently re-verify tag and full 40-character commit hash via GitHub API. Re-audit all AO claims against this exact commit.

### Finding EXT-002: Antigravity CLI Version Unpinned
- **Observation**: Documentation relies on `official-latest` for Antigravity CLI.
- **Finding**: Violates `NFR-007` (Version Pinning). Official release `google-antigravity/antigravity-cli` `1.2.7` exists at tag commit `7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`. Furthermore, documentation improperly blended documented baseline with local runtime proof.
- **Severity**: BLOCKER.
- **Required Action**: Pin exact documentation baseline version and commit. Strictly separate `DOCUMENTATION_BASELINE_VERSION` from `LOCAL_RUNTIME_TESTED_VERSION` (to be proven in Phase P01).

### Finding EXT-003: Passive Design Source Commits Unreliable
- **Observation**: Several short commit hashes recorded in `third_party/SOURCE_VERSIONS.md` do not resolve on their upstream repositories.
- **Finding**: References for Proxide, Mieruko, AIWorkHub, Symphony, Codencer, and AWS CAO contain synthetic or unverifiable short hashes.
- **Severity**: BLOCKER.
- **Required Action**: Treat all passive source pins as untrusted. Inspect each repository independently, verify real commits, and record full 40-character commit SHAs, commit dates, licenses, and files actually inspected.

### Finding EXT-004: Synthetic SOURCE EVIDENCE Paths
- **Observation**: Dossiers cite implementation paths (e.g., `pkg/terminal/conpty_windows.go`, `pkg/worktree/manager.go`, `src/security/containment.rs`, `core/review/evidence.py`, `daemon/bridge.go`) that do not exist at the referenced upstream commits.
- **Finding**: Concept citations were fabricated or assumed rather than pointing to real source files or README/documentation sections.
- **Severity**: BLOCKER.
- **Required Action**: Re-verify every single `Exact Source Area` and `SOURCE EVIDENCE` citation against the real pinned commit. If a concept originates from a README or architectural doc, cite that exact document/heading. Never fabricate an implementation path.

---

# 3. Required Remediation Scope

1. **Commit Pinning**: Replace all unverified/short hashes with full 40-character verified commit SHAs across all 9 source dossiers and `third_party/SOURCE_VERSIONS.md`.
2. **Evidence Standardization**: Convert all claims to the standard 11-field `Claim ID / Claim / Repository / Pinned tag / Pinned commit / Evidence type / Exact evidence / Section / Verification / Confidence / Notes` structure.
3. **Upstream Status Decoupling**: Separate documented contracts from local runtime testing across all matrices.
4. **AO Contract Repair**: Remove invented endpoints (`/api/v1/health`, `/stop`); document verified endpoints (`/healthz`, `/readyz`, `/kill`, `/restore`, `/send`).
5. **AO ↔ Agy Integration Realism**: Acknowledge that AO invokes Agy interactively; mark structured WorkerReport normalization as `P01_PROOF_REQUIRED`.
6. **Agy Contract Repair**: Document actual official headless mechanisms (`stream-json`, `--json-schema`, `--input-format`) rather than invented flags.
7. **Readonly Approval**: Remove automatic source merge from `approve_task`.
8. **ChatGPT Transport Feasibility**: Advance transport feasibility proof to Phase P01 and add formal requirement `FR-016`.
