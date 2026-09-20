# PHASE SPECIFICATION: P04 — EVIDENCE & REVIEW ENGINE

## 1. Objective
Implement the independent Git evidence collector and Review Bundle builder.

## 2. Deliverables
- `EvidenceCollector`: Direct Git diff, base/head SHA extractor, and exit code verifier.
- `ReviewBundleBuilder`: Assembles Contract, Claims, Diffs, Test Artifacts, and Policy Findings into a single payload.
- `PolicyEngine`: Detects touched files outside `allowed_scope` or inside `forbidden_scope`.

## 3. Exit Gate
- Evidence collector mathematically matches actual Git commit diffs; Review Bundle validates 100% against `review-bundle.schema.json`.
