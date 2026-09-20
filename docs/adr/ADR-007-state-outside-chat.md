# ADR-007: State Stored Outside Chat History

## Status
ACCEPTED

## Context
LLM chat interfaces (ChatGPT Web) manage conversation history via token windows. When threads grow long, context is truncated, leading to "amnesia" regarding past architectural decisions, task statuses, and pair bindings.

## Decision
All authoritative engineering state (project metadata, pair bindings, task contracts, review bundles, audit logs) resides **outside the conversation thread** in the Supervisor Control Plane's local database. Chat history is treated as transient working memory.

## Alternatives Considered
- *In-Chat State Serialization*: Appending state JSON blobs to prompt turns. Rejected because it wastes prompt tokens, is prone to hallucinated state edits, and is lost if the thread is cleared.

## Consequences
- Robust recoverability across ChatGPT session restarts or browser refreshes.
- High-reasoning prompt tokens are reserved strictly for technical thinking and code review.

## Source Evidence
- Codencer design philosophy: "State not chat; bridge not brain".

## Revisit Conditions
Permanent core policy.
