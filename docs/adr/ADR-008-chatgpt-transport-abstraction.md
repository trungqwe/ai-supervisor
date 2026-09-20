# ADR-008: ChatGPT Transport Abstraction

## Status
ACCEPTED

## Context
OpenAI's interface mechanisms for ChatGPT Web evolve over time (e.g., custom GPT Actions, local Model Context Protocol (MCP) clients, ChatGPT Desktop Apps, and browser web extensions).

## Decision
We decouple the Supervisor core from any specific ChatGPT transport protocol via a `SupervisorToolSurface` interface. In V1, we target local MCP relay and structured HTTP API endpoints. If OpenAI changes client transport protocols, only the transport adapter is updated.

## Alternatives Considered
- *Direct Browser Script Injection*: Injecting JavaScript into the ChatGPT Web browser tab. Rejected due to DOM instability, account security risks, and breakage on UI updates.

## Consequences
- Future-proof interface: can seamlessly transition from MCP to ChatGPT Apps or external CLI supervisors without touching core domain logic.

## Source Evidence
- tt-a1i/proxide and Mieruko MCP integration patterns.

## Revisit Conditions
Revisit when OpenAI formalizes official desktop MCP support or standardized bidirectional workspace hooks.
