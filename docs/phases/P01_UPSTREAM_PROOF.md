# PHASE SPECIFICATION: P01 — UPSTREAM PROOF

## 1. Objective
Empirically test and prove Untrivial Agent Orchestrator (AO `v0.13.0`) and official Antigravity CLI (`agy`) headless execution on the local Windows environment prior to Supervisor code implementation.

## 2. Why Now
Do not build architectural adapters against unverified assumptions. Upstream capabilities must be proven live.

## 3. Test Scope
- Install / verify AO Windows daemon execution.
- Validate daemon `/api/v1/health` on loopback.
- Verify project registration and isolated Git worktree creation.
- Launch `agy` in headless mode within the worktree and verify structured output.
- Verify clean process termination and session recovery across restarts.

## 4. Exit Gate
- All items in `docs/sources/UPSTREAM_CONTRACT_BASELINE.md` verified live with exit code 0.
