# SOURCE DOSSIER: 07 — CODENCER

## 1. Metadata
- **Repository**: `lookmanrays/codencer`
- **Role in Architecture**: Passive Design Source (Bridge-Not-Brain & Execution Vocabulary)
- **Pinned Commit**: `8d4908b1acf049cd97a9c4dcb752d1e44cbb8655`
- **Commit Date**: 2026-08-06T14:35:56Z
- **License**: Apache-2.0
- **Primary Language**: Go

---

# 2. Capabilities Evaluated & Adopted
- **"Bridge, Not Brain" Philosophy**: The orchestrator is a durable, stateful bridge; it does not replace the planner AI's cognitive reasoning.
- **Execution Vocabulary**: Standardizing concepts around `runs`, `attempts`, `artifacts`, `validations`, and `blockers`.
- **State Outside Chat**: Maintaining state in a local persistent database rather than in chat logs.

---

# 3. Capabilities Explicitly Rejected & Corrected Claims
- **Direct Shell Invocation Tools**: Codencer's generic shell execution tools are replaced by our structured Task Contract execution.
- **Correction of Synthetic Path**: In previous drafts, `daemon/bridge.go` was cited. Real repository structures code in `cmd/broker/main.go`, `internal/domain/`, and documents architecture in `README.md`.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` across domain vocabulary and state management.

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-CDC-001
Claim: Codencer establishes the "Bridge, Not Brain" principle: the control plane is an execution and audit broker, not the AI planner.
Repository: lookmanrays/codencer
Pinned tag: N/A
Pinned commit: 8d4908b1acf049cd97a9c4dcb752d1e44cbb8655
Evidence type: README
Exact evidence: README.md
Section / symbol: "Core Philosophy: Bridge, Not Brain"
Verification: VERIFIED
Confidence: HIGH
Notes: Explicitly defines the separation between external planner and stateful local broker.

Claim ID: CLM-CDC-002
Claim: Codencer maintains stateful execution runs, attempts, validations, and blockers outside chat memory.
Repository: lookmanrays/codencer
Pinned tag: N/A
Pinned commit: 8d4908b1acf049cd97a9c4dcb752d1e44cbb8655
Evidence type: README
Exact evidence: README.md
Section / symbol: "Stateful Execution & Audit"
Verification: VERIFIED
Confidence: HIGH
Notes: Details database-backed run tracking rather than conversation memory.
