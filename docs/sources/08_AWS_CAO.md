# SOURCE DOSSIER: 08 — AWS CLI AGENT ORCHESTRATOR

## 1. Metadata
- **Repository**: `awslabs/cli-agent-orchestrator`
- **Role in Architecture**: Passive Design Source (Provider Abstraction)
- **Pinned Commit**: `156cf1edcdb9de1ee01a2f17c2a4026e5f305e56`
- **Commit Date**: 2026-09-20T05:52:34Z
- **License**: Apache-2.0
- **Primary Language**: Python

---

# 2. Capabilities Evaluated & Adopted
- **Provider / Runtime Separation**: Abstracting execution engines behind a uniform provider interface.
- **Session Abstraction**: Managing session lifecycles independently of specific worker binaries.
- **Structured Event Notifications**: Emitting lifecycle state events.

---

# 3. Capabilities Explicitly Rejected & Corrected Claims
- **AWS Cloud Tight Coupling**: We do not require AWS credentials, IAM roles, or Bedrock dependencies.
- **Correction of Synthetic Path**: In previous drafts, `cao/providers/base.py` was cited. Real repository structures provider code under `src/cli_agent_orchestrator/providers/`.

---

# 4. Integration Strategy
- **Strategy**: `ADAPT` provider abstraction concepts into our `AOAdapter` interface.

---

# 5. SOURCE EVIDENCE

Claim ID: CLM-CAO-001
Claim: AWS CAO defines modular agent provider interfaces allowing multiple agent backends to be orchestrated under a unified schema.
Repository: awslabs/cli-agent-orchestrator
Pinned tag: N/A
Pinned commit: 156cf1edcdb9de1ee01a2f17c2a4026e5f305e56
Evidence type: SOURCE_CODE
Exact evidence: src/cli_agent_orchestrator/providers/
Section / symbol: provider implementations
Verification: VERIFIED
Confidence: HIGH
Notes: Contains decoupled provider adapters (e.g. Bedrock, Claude Code, Antigravity CLI).
