# SOURCE DOSSIER: 06 — OPENAI SYMPHONY

## 1. Metadata
- **Repository**: `openai/symphony`
- **Role in Architecture**: Passive Design Source (Workflow-as-Policy & Isolated Runs)
- **Pinned Commit**: `be10a1b79df723d6d7612b5651c8522704dafb2e`
- **Commit Date**: 2026-09-15T22:12:07Z
- **License**: Apache-2.0
- **Primary Language**: Elixir (Reference Implementation) / Language-Agnostic Specification (`SPEC.md`)

---

# 2. Capabilities Evaluated & Adopted
- **Workflow-as-Policy**: Expressing engineering workflows as strict, deterministic state machines.
- **Isolated Implementation Runs**: Ensuring each agent execution occurs in an ephemeral, isolated workspace.
- **Specification-First Philosophy**: Relying on formal specifications rather than ad-hoc framework defaults.

---

# 3. Capabilities Explicitly Rejected & Corrected Claims
- **Elixir Runtime**: We do not adopt the Elixir language runtime or BEAM VM.
- **Linear Issue Tracker Hardcoding**: We do not tie our task contracts exclusively to Linear.
- **Correction of Synthetic Path**: In previous drafts, `lib/symphony/workflow.ex` was cited. Real repository specifies the architecture in `SPEC.md`.

---

# 4. Integration Strategy
- **Strategy**: `ADAPT` specification patterns into `docs/06_WORKFLOW_STATE_MACHINE.md` and `TaskContract`.

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-SYM-001
Claim: Symphony formalizes autonomous coding workflows as explicit state machines governing work unit lifecycles.
Repository: openai/symphony
Pinned tag: N/A
Pinned commit: be10a1b79df723d6d7612b5651c8522704dafb2e
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SPEC.md
Section / symbol: Section 3: "Work Unit Lifecycle & State Transitions"
Verification: VERIFIED
Confidence: HIGH
Notes: Formal specification defines deterministic states from creation to audit.

Claim ID: CLM-SYM-002
Claim: Symphony isolates implementation runs inside dedicated, ephemeral workspace environments.
Repository: openai/symphony
Pinned tag: N/A
Pinned commit: be10a1b79df723d6d7612b5651c8522704dafb2e
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: SPEC.md
Section / symbol: Section 4: "Workspace Runner Isolation"
Verification: VERIFIED
Confidence: HIGH
Notes: Mandates workspace isolation to avoid dirtying primary repository branches.
