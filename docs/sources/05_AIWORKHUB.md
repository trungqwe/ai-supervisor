# SOURCE DOSSIER: 05 — AIWORKHUB

## 1. Metadata
- **Repository**: `shrec/AIWorkHub`
- **Role in Architecture**: Passive Design Source (Evidence-First Review & Durable Task Memory)
- **Reference Commit**: `main` (`2026-09-12`)
- **Inspection Date**: 2026-09-20
- **License**: MIT
- **Primary Language**: Python

---

# 2. Capabilities Evaluated & Adopted
- **Evidence-First Verification**: Treating worker output as candidate claims requiring Git verification.
- **Durable Task Memory**: Maintaining task state and evidence graphs locally across agent turns.
- **Independent Git Verification**: Querying commit diffs, tree SHAs, and exit codes directly.

---

# 3. Capabilities Explicitly Rejected
- **VS Code Extension Tight Coupling**: We do not bind our core logic to the VS Code UI extension ecosystem.
- **Direct Multi-Model Swarms**: Multi-model parallel routing is excluded from V1.

---

# 4. Integration Strategy
- **Strategy**: `REIMPLEMENT_PATTERN` in `EvidenceCollector` and `ReviewBundleBuilder`.

---

# 5. SOURCE EVIDENCE

### Evidence Item 5.1: Candidate Claims vs. Verification
- **Claim**: AI completion claims must be audited against actual Git diffs before acceptance.
- **Repository**: `shrec/AIWorkHub`
- **Reference**: `main`
- **Source File / Module**: `core/review/evidence.py` (`verify_candidate_result`)
- **Verification Status**: VERIFIED
- **Confidence**: HIGH

### Evidence Item 5.2: Durable Repository Memory
- **Claim**: Local-first task memory avoids dependency on transient chat logs or remote cloud accounts.
- **Repository**: `shrec/AIWorkHub`
- **Reference**: `main`
- **Source File / Module**: `core/storage/task_graph.py`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
