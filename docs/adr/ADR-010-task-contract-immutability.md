# ADR-010: Task Contract Immutability

## Status
ACCEPTED

## Context
In unconstrained multi-agent systems, agents frequently modify their own goals, expand scope to touch shared libraries, or skip tests when encountering difficulty.

## Decision
Once a `TaskContract` is validated and dispatched, its objective, allowed scope, forbidden scope, and test requirements are **strictly immutable**. If the worker encounters an unexpected dependency or architectural blocker, it must halt and report `BLOCKED`. Scope amendments can only be authorized by the Supervisor via a new contract or revision cycle.

## Alternatives Considered
- *Dynamic Scope Expansion*: Allowing workers to request scope broadening mid-execution. Rejected as prone to cascading unintended file modifications.

## Consequences
- Deterministic, bounded worker execution.
- Scope containment verified mathematically against Git diff before review.

## Source Evidence
- OpenAI Symphony isolated execution runs and Mieruko task boundaries.

## Revisit Conditions
Permanent core policy.
