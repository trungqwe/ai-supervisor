# SOURCE DOSSIER: 06 — OPENAI SYMPHONY

## 1. Metadata
- **Repository**: `openai/symphony`
- **Role in Architecture**: Passive Design Source (Workflow-as-Policy & Isolated Runs)
- **Reference Commit**: `main` (`2026-09-08`)
- **Inspection Date**: 2026-09-20
- **License**: Apache-2.0
- **Primary Language**: Elixir (Reference) / Language-Agnostic Specification (`SPEC.md`)

---

# 2. Capabilities Evaluated & Adopted
- **Workflow-as-Policy**: Expressing engineering workflows as strict, deterministic state machines.
- **Isolated Implementation Runs**: Ensuring each agent execution occurs in an ephemeral, isolated workspace.
- **Specification-First Philosophy**: Relying on formal specifications rather than ad-hoc framework defaults.

---

# 3. Capabilities Explicitly Rejected
- **Elixir Runtime**: We do not adopt the Elixir language runtime or BEAM VM.
- **Linear Issue Tracker Hardcoding**: We do not tie our task contracts exclusively to Linear.

---

# 4. Integration Strategy
- **Strategy**: `ADAPT` specification patterns into `docs/06_WORKFLOW_STATE_MACHINE.md` and `TaskContract`.

---

# 5. SOURCE EVIDENCE

### Evidence Item 6.1: Workflow State Machine Contract
- **Claim**: Defining explicit transition states prevents agents from drifting or skipping review phases.
- **Repository**: `openai/symphony`
- **Reference**: `SPEC.md`
- **Source File / Module**: Section 3: "Work Unit Lifecycle & State Transitions"
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 6.2: Isolated Run Sandboxing
- **Claim**: Ephemeral workspace branches isolate unfinished code changes until verified.
- **Repository**: `openai/symphony`
- **Reference**: `SPEC.md`
- **Source File / Module**: Section 4: "Workspace Runner Isolation"
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
