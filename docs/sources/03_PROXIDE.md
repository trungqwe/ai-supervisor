# SOURCE DOSSIER: 03 — PROXIDE

## 1. Metadata
- **Repository**: `tt-a1i/proxide`
- **Role in Architecture**: Passive Design Source (Security & Containment Reference)
- **Reference Commit**: `main` (`2026-09-10`)
- **Inspection Date**: 2026-09-20
- **License**: MIT
- **Primary Language**: Rust

---

# 2. Capabilities Evaluated & Adopted
- **Readonly Default Philosophy**: AI workspace bridges must operate read-only by default.
- **Explicit Project Roots**: Containment boundaries strictly enforced via canonical path checking.
- **Sanitized Audit Trail**: Scrubbing tokens and secrets from outbound logs and MCP payloads.

---

# 3. Capabilities Explicitly Rejected
- **Rust Runtime Dependency**: We do not vendor or depend on Proxide's Rust binary.
- **Generic Terminal Shell Tools**: Proxide's arbitrary shell execution tools are explicitly rejected.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` (Incorporate verified containment logic into our `PolicyEngine`).

---

# 5. SOURCE EVIDENCE

### Evidence Item 3.1: Canonical Path Containment
- **Claim**: Strict prefix matching prevents directory traversal attacks outside the workspace root.
- **Repository**: `tt-a1i/proxide`
- **Reference**: `commit 4d7b1a2`
- **Source File / Module**: `src/security/containment.rs`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 3.2: Readonly Tool Enforcement
- **Claim**: Separating read tools from mutating tools prevents accidental codebase corruption.
- **Repository**: `tt-a1i/proxide`
- **Reference**: `docs/security_model.md`
- **Source Module**: Tool permission matrix
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
