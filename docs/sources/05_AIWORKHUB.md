# SOURCE DOSSIER: 05 — AIWORKHUB

## 1. Metadata
- **Repository**: `shrec/AIWorkHub`
- **Role in Architecture**: Passive Design Source (Evidence-First Review & Durable Memory)
- **Pinned Commit**: `19c8ce548c27316a06117c0638b4f853e00292b8`
- **Commit Date**: 2026-09-20T06:48:06Z
- **License**: MIT
- **Primary Language**: Python

---

# 2. Capabilities Evaluated & Adopted
- **Evidence-First Verification**: Treating worker output as candidate claims requiring Git verification.
- **Durable Task Memory**: Maintaining task state and evidence graphs locally across agent turns.
- **Independent Git Verification**: Querying commit diffs, tree SHAs, and exit codes directly.

---

# 3. Capabilities Explicitly Rejected & Corrected Claims
- **VS Code Extension Tight Coupling**: We do not bind our core logic to the VS Code UI extension ecosystem.
- **Correction of Synthetic Path**: In previous drafts, `core/review/evidence.py` was cited. Real codebase organizes documentation under `docs/ARCHITECTURE.md` and `docs/QUALITY_CONTROL.md`.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` in `EvidenceCollector` and `ReviewBundleBuilder`.

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-AWH-001
Claim: AIWorkHub establishes an evidence-first review process where candidate results must be verified against repository facts before acceptance.
Repository: shrec/AIWorkHub
Pinned tag: N/A
Pinned commit: 19c8ce548c27316a06117c0638b4f853e00292b8
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: docs/ARCHITECTURE.md
Section / symbol: "Verification & Review Architecture"
Verification: VERIFIED
Confidence: HIGH
Notes: Specifies candidate results vs. authoritative verification gates.

Claim ID: CLM-AWH-002
Claim: AIWorkHub utilizes local-first durable task graphs to maintain project context without cloud service lock-in.
Repository: shrec/AIWorkHub
Pinned tag: N/A
Pinned commit: 19c8ce548c27316a06117c0638b4f853e00292b8
Evidence type: OFFICIAL_REPO_DOC
Exact evidence: docs/CONTEXT_GRAPH.md
Section / symbol: "Local-First Context Graph Architecture"
Verification: VERIFIED
Confidence: HIGH
Notes: Details maintaining durable context locally within the repository.
