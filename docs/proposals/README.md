# PROPOSALS SYSTEM & SPECIFICATION

This directory contains formal proposals for capabilities, optimizations, or architectural adjustments discovered during implementation.

---

# 1. Mandatory Rule
If an agent or developer discovers a new idea, performance improvement, or architectural alternative during execution:
1. **DO NOT IMPLEMENT IT IMMEDIATELY**.
2. Create a new proposal file: `docs/proposals/PROPOSAL-xxx.md`.
3. Set status to `PENDING_REVIEW`.
4. Await formal User and Supervisor evaluation before any roadmap or code change is permitted.

---

# 2. Proposal Template

```markdown
# PROPOSAL-xxx: [Short Descriptive Title]

## Metadata
- **Author**: [Agent / Developer]
- **Date**: [YYYY-MM-DD]
- **Related Phase / Task**: [e.g., P02 / TASK-012]
- **Status**: PENDING_REVIEW (PENDING_REVIEW | APPROVED | REJECTED)

## 1. Observation
What specific challenge, bottleneck, or opportunity was observed during implementation?

## 2. Proposed Idea
What is the proposed change or addition?

## 3. Problem Solved
What concrete engineering problem does this solve?

## 4. Existing Upstream Capability Checked
Which evaluated repositories in `docs/sources/` were checked to ensure this capability does not already exist upstream?

## 5. Why Existing Capability is Insufficient
Technical justification for why existing upstream features cannot meet the need.

## 6. Expected Benefits vs. Implementation Cost
- **Benefits**: [e.g., reduces bundle generation latency by 40%]
- **Costs / Complexity**: [e.g., adds 100 lines of adapter code]

## 7. Architectural Impact
Does this require a new ADR or modification of canonical documents?

## 8. Decision & Rationale
*(Filled by Supervisor / User during review)*
- **Verdict**: PENDING
- **Rationale**:
```
