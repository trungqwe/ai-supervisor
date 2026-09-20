# SOURCE DOSSIER: 08 — AWS CLI AGENT ORCHESTRATOR

## 1. Metadata
- **Repository**: `awslabs/cli-agent-orchestrator`
- **Role in Architecture**: Passive Design Source (Provider Abstraction)
- **Reference Commit**: `main` (`2026-08-30`)
- **Inspection Date**: 2026-09-20
- **License**: Apache-2.0
- **Primary Language**: Python

---

# 2. Capabilities Evaluated & Adopted
- **Provider / Runtime Separation**: Abstracting execution engines behind a uniform provider interface.
- **Session Abstraction**: Managing session lifecycles independently of specific worker binaries.
- **Structured Event Notifications**: Emitting lifecycle state events.

---

# 3. Capabilities Explicitly Rejected
- **AWS Cloud Tight Coupling**: We do not require AWS credentials, IAM roles, or Bedrock dependencies.
- **Concurrent Cloud Worker Fleets**: Excluded from local V1 scope.

---

# 4. Integration Strategy
- **Strategy**: `ADAPT` provider abstraction concepts into our `AOAdapter` interface.

---

# 5. SOURCE EVIDENCE

### Evidence Item 8.1: Provider Abstraction Interface
- **Claim**: Separating agent control from underlying harness binaries allows clean worker swapping.
- **Repository**: `awslabs/cli-agent-orchestrator`
- **Reference**: `main`
- **Source File / Module**: `cao/providers/base.py`
- **Verification Status**: VERIFIED
- **Confidence**: HIGH
