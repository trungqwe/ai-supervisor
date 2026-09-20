# SOURCE DOSSIER: 03 — PROXIDE

## 1. Metadata
- **Repository**: `tt-a1i/proxide`
- **Role in Architecture**: Passive Design Source (Security & Containment Reference)
- **Pinned Commit**: `c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee`
- **Commit Date**: 2026-06-20T15:17:16Z
- **License**: MIT
- **Primary Language**: Rust

---

# 2. Capabilities Evaluated & Adopted
- **Readonly Default Philosophy**: AI workspace bridges must operate read-only by default.
- **Explicit Project Roots**: Containment boundaries strictly enforced via canonical path checking.
- **Sanitized Audit Trail**: Scrubbing tokens and secrets from outbound logs and MCP payloads.

---

# 3. Capabilities Explicitly Rejected & Corrected Claims
- **Rust Binary Vendoring**: We do not vendor Proxide's Rust codebase.
- **Generic Terminal Shell Tools**: Arbitrary shell execution tools are explicitly rejected.
- **Correction of Synthetic Path**: In previous drafts, `src/security/containment.rs` was cited. Real codebase uses `connector-rs/src/main.rs` and `README.md`.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` (Incorporate verified containment logic into our `PolicyEngine`).

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-PRX-001
Claim: Proxide establishes security containment around registered workspace roots.
Repository: tt-a1i/proxide
Pinned tag: N/A (Tracking default branch commit)
Pinned commit: c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee
Evidence type: README
Exact evidence: README.md
Section / symbol: "Security & Containment"
Verification: VERIFIED
Confidence: HIGH
Notes: Explicitly restricts operations to allowed roots and documents readonly defaults.

Claim ID: CLM-PRX-002
Claim: Proxide Rust connector implements local MCP workspace bridge.
Repository: tt-a1i/proxide
Pinned tag: N/A
Pinned commit: c1621e313c6cdfe3a10c8f6e929d46ba8a8c27ee
Evidence type: SOURCE_CODE
Exact evidence: connector-rs/src/main.rs
Section / symbol: main()
Verification: VERIFIED
Confidence: HIGH
Notes: Implements local loopback MCP connector in Rust.
