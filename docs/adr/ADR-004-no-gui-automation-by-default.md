# ADR-004: No GUI Automation by Default

## Status
ACCEPTED

## Context
Some AI automation tools attempt to drive development environments by simulating mouse clicks, sending OS keystrokes, or taking periodic screen captures.

## Decision
We strictly prohibit OS-level mouse, keyboard, or screenshot automation by default. All system operations, task dispatches, and verification routines must occur through headless APIs, CLI streams, and filesystem VCS inspection.

## Alternatives Considered
- *CDP / OS accessibility driver (Antigravity Link)*: Kept strictly as a documented fallback in case a critical workflow cannot be triggered via CLI.

## Consequences
- The developer's desktop remains completely free for normal human work while agents run in the background.
- Zero screen resolution, DPI scaling, or window focus dependencies.

## Source Evidence
- Proxide and Symphony design philosophies: headless, protocol-driven workspace management over visual automation.

## Revisit Conditions
Revisit only if a mandatory tool lacks any CLI, API, or RPC interface.
