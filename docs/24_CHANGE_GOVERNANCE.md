# 24. CHANGE GOVERNANCE & DECISION HIERARCHY

> **Authority**: Immutable Change Intake Workflow & Conflict Resolution Hierarchy  
> **Status**: Approved Baseline

---

# 1. The Change Intake Pipeline

Every new capability, requirement amendment, or architectural suggestion discovered during execution must strictly follow this pipeline before any implementation code is permitted:

```mermaid
flowchart TD
    NewIdea[New Idea / Observation Discovered] --> CheckReq[1. Check docs/02_REQUIREMENTS.md]
    CheckReq --> CheckReg[2. Check docs/sources/SOURCE_REGISTRY.md]
    CheckReg --> CheckUpstream{Already Solved Upstream?}
    
    CheckUpstream -- YES --> Integrate[Integrate / Adapt via Stable Adapter]
    CheckUpstream -- NO --> Proposal[Create docs/proposals/PROPOSAL-xxx.md]
    
    Proposal --> SupervisorAudit[Supervisor & User Evaluation]
    SupervisorAudit --> ArchitectureImpact{Requires Architecture Change?}
    
    ArchitectureImpact -- YES --> DraftADR[Draft & Approve formal ADR in docs/adr/]
    ArchitectureImpact -- NO --> UpdateRoadmap[Update docs/17_ROADMAP.md]
    
    DraftADR --> UpdateRoadmap
    UpdateRoadmap --> Contract[Formulate Immutable Task Contract]
    Contract --> ExecAllowed[Implementation Code Authorized]
```

---

# 2. Immutable Decision Hierarchy (Conflict Resolution)

Whenever a coding agent, developer prompt, or tool output presents conflicting guidance, the conflict **MUST** be resolved strictly in order of the following 9-level priority hierarchy:

```text
DECISION PRIORITY (HIGHEST TO LOWEST)

1. Confirmed User Requirement (Explicit User Direction)
2. Approved ADR (docs/adr/)
3. Canonical Architecture (docs/04_ARCHITECTURE.md)
4. Requirement Specification (docs/02_REQUIREMENTS.md)
5. Approved Roadmap (docs/17_ROADMAP.md)
6. Task Contract (docs/08_TASK_CONTRACT.md)
7. Source Repo / Reference Dossier (docs/sources/)
8. Worker Suggestion (Worker Report / Claims)
9. Chat Conversation History (Transient Dialogue)
```

### Concrete Conflict Rules:
- If a worker proposes an optimization that contradicts an ADR: **The ADR wins**.
- If an upstream repository adds a new feature that violates our architecture: **The Architecture wins** until a new ADR is formally approved.
- If a prior chat turn suggests something different from canonical documentation: **The Canonical Documentation wins**.
