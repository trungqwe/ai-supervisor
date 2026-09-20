# PHASE 0 — ARCHITECTURE FREEZE RECORD

Authority:
User + External Supervisor Decision Record

Freeze Date:
2026-09-20

Status:
ARCHITECTURE_FROZEN

Pre-Freeze Approved Baseline:
7c7f18598516c79741eff04cb742872d552d86da

Freeze Commit:
6365d38a2a774be829dd06d371952e61eb6b9343

Freeze Tag:
phase0-architecture-v1

## Intelligence Plane

ChatGPT Web / GPT-5.6 Sol High

Responsibilities:

* architecture
* planning
* task decomposition
* review
* audit
* decisions

Must not directly mutate project source.

## Supervisor Control Plane

Our project-specific control plane.

Owns:

* Pair binding
* Project Registry
* Task Contract
* task/review authority
* Policy Engine
* project reading/search
* Git evidence collection
* WorkerReport parsing
* Review Bundle
* AO adapter
* ChatGPT transport abstraction
* audit trail
* lightweight Pair Manager

## Execution Control Plane

`Untrivial-ai/agent-orchestrator`

Pinned documentation baseline:

`v0.13.0`

`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`

## Primary Worker

Official Antigravity CLI / Agy

Pinned documentation baseline:

`1.2.7`

`7bb195acaec9e7788df5210d0dc3e15f3cefc6b3`

## Core Authority Model

Technical authority:
repository canonical docs + approved ADRs

Execution authority:
AO

Task/review authority:
Supervisor

Code authority:
Git SHA/diff/tests

Chat history:
non-authoritative

## Frozen V1 Principles

* upstream-first;
* no AO duplication;
* worker claims are not proof;
* evidence-first Review Bundle;
* immutable Task Contract during execution;
* ChatGPT gets narrow domain-level tools;
* no arbitrary shell/write/git tools exposed to ChatGPT;
* no automatic approval→merge;
* transport abstraction isolates ChatGPT product volatility;
* one Pair = one ChatGPT binding + one project + one active worker + one active task in V1;
* no browser/OS input automation as primary path;
* no GitHub message bus;
* local project files remain source of technical truth.

## Accepted Deferred Proofs

P01-A:
AO runtime proof

P01-B:
Direct Agy capability proof

P01-C:
AO ↔ Agy structured completion / WorkerReport proof

P01-D:
ChatGPT Plus transport feasibility proof

These are deliberately NOT claimed as proven by Phase 0.

## Accepted Process Deviations

P00-DEV-001:
ACCEPTED_AT_PHASE0_FREEZE

No architecture/artifact invalidation.

## External Decision

Record exactly:

`ARCHITECTURE_FROZEN_APPROVED`

Approved pre-freeze baseline:

`7c7f18598516c79741eff04cb742872d552d86da`

Any post-freeze architecture modification requires:

* proposal where applicable;
* ADR;
* explicit approval according to `docs/24_CHANGE_GOVERNANCE.md`.
