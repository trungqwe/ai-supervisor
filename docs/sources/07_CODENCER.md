# SOURCE DOSSIER: 07 — CODENCER

## 1. Metadata
- **Repository**: `lookmanrays/codencer`
- **Role in Architecture**: Passive Design Source (Bridge-Not-Brain & Execution Vocabulary)
- **Reference Commit**: `main` (`2026-09-05`)
- **Inspection Date**: 2026-09-20
- **License**: MIT
- **Primary Language**: Go / Python

---

# 2. Capabilities Evaluated & Adopted
- **"Bridge, Not Brain" Philosophy**: The orchestrator is a durable, stateful bridge; it does not replace the planner AI's cognitive reasoning.
- **Execution Vocabulary**: Standardizing concepts around `runs`, `attempts`, `artifacts`, `validations`, and `blockers`.
- **State Outside Chat**: Maintaining state in a local persistent database rather than in chat logs.

---

# 3. Capabilities Explicitly Rejected
- **Direct Shell Invocation Tools**: Codencer's generic shell execution tools are replaced by our structured Task Contract execution.
- **Hosted Cloud Relay Services**: We rely strictly on self-hosted, local loopback communication.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` across domain vocabulary and state management.

---

# 5. SOURCE EVIDENCE

### Evidence Item 7.1: Bridge-not-Brain Architecture
- **Claim**: The control plane functions as a task manager, validator, and executor bridge without an embedded LLM.
- **Repository**: `lookmanrays/codencer`
- **Reference**: `main`
- **Source File / Module**: `README.md`, `daemon/bridge.go`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 7.2: Execution Vocabulary
- **Claim**: Decomposing execution into formal attempts, validations, and blockers provides clear auditability.
- **Repository**: `lookmanrays/codencer`
- **Reference**: `core/types.go`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
