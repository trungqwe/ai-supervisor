# ADR-005: Supervisor Does Not Edit Source Code

## Status
ACCEPTED

## Context
When an AI supervisor identifies a bug or a missing line of code during review, there is a temptation to allow the supervisor to directly patch the file.

## Decision
The **Supervisor Control Plane and ChatGPT Web are strictly prohibited from writing, editing, or committing application source code**. If code changes are required, the Supervisor must issue a `REVISION_REQUIRED` decision with structured feedback to the Worker.

## Alternatives Considered
- *Supervisor Direct Patching*: Allowing ChatGPT to execute file edits during review. Rejected because it blurs accountability, bypasses worker compilation/test loops, and breaks the division of labor.

## Consequences
- Clean audit trail: 100% of code changes originate in the Worker's worktree with associated test runs.
- Supervisor remains an objective auditor with zero self-authored bias.

## Source Evidence
- Codencer and AIWorkHub separation of planner/manager from executor/worker.

## Revisit Conditions
Immutable architectural rule. No revisit planned.
