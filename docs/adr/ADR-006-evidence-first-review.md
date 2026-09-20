# ADR-006: Evidence-First Review Architecture

## Status
ACCEPTED

## Context
AI coding agents frequently produce completion summaries stating "all tests passed" or "implementation complete" when, in fact, edge-case tests failed or unnecessary files were modified. Relying on worker claims without independent verification leads to silent defects and architecture drift.

## Decision
We mandate an **Evidence-First Review Architecture**:
1. Worker reports are formally categorized as `WorkerClaim` objects (unverified claims).
2. The Supervisor independently collects base/head Git SHAs, file diffs, and process exit codes directly from the filesystem.
3. Both sets of data are compared side-by-side in a compiled `ReviewBundle`.

## Alternatives Considered
- *Claim-Based Acceptance*: Trusting the worker's reported status if exit code is 0. Rejected due to high risk of hallucinated or incomplete assertions.

## Consequences
- 100% verification guarantee before task approval.
- Any discrepancy between claimed changes and actual Git diff immediately flags a policy alert.

## Source Evidence
- AIWorkHub evidence-first review architecture and candidate result acceptance boundaries.

## Revisit Conditions
Permanent core policy.
