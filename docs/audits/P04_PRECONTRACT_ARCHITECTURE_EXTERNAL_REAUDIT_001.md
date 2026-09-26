# P04 PRE-CONTRACT ARCHITECTURE EXTERNAL RE-AUDIT 001

- **Audit Record ID:** `P04-PRECONTRACT-ARCH-REAUDIT-001`
- **Audited Commit SHA:** `0ab0d8bd151dfd0501c6d40ac0862e1574adec77` (baseline)
- **Target Phase:** Phase P04 — Evidence & Review Engine
- **Date:** 2026-09-26
- **Lead Auditor:** External Supervisor (Independent Control Plane Authority)
- **Status:** `REVISION_2_REQUIRED`
- **Preceding Audit Record:** [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001.md)
- **Erratum Reference:** [`docs/audits/P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md`](P04_PRECONTRACT_ARCHITECTURE_EXTERNAL_AUDIT_001_ERRATUM_001.md)
- **Active Gate:** `P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_2`

---

## 1. Executive Re-Audit Verdict & Scope

Pursuant to [`docs/24_CHANGE_GOVERNANCE.md`](../24_CHANGE_GOVERNANCE.md), [`docs/sources/SOURCE_REGISTRY.md`](../sources/SOURCE_REGISTRY.md), [`docs/sources/REUSE_MATRIX.md`](../sources/REUSE_MATRIX.md), and [`AGENTS.md`](../../AGENTS.md), the External Supervisor conducted Re-Audit 001 on the pre-contract architecture revisions submitted in response to External Audit 001.

### 1.1. Evaluated Deliverables
1. `docs/proposals/PROPOSAL-P04-001-evidence-review-engine-boundaries.md`
2. `docs/adr/DRAFT-ADR-018-evidence-review-and-verification-isolation.md`
3. `docs/plans/PLAN-P04-EVIDENCE-REVIEW.md`

### 1.2. Formal Verdicts
- `PROPOSAL_P04_001 = REVISION_2_REQUIRED`
- `ADR_018 = REVISION_2_REQUIRED`
- `PLAN_P04_EVIDENCE_REVIEW = REVISION_2_REQUIRED`
- `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_2`
- `P04_TASK_CONTRACT = NOT_RELEASED`
- `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
- `P05_CODE = NOT_AUTHORIZED`

---

## 2. Status of Phase P04 Pre-Contract Findings

| Finding ID | Severity | Category | Title & Summary | Status in Re-Audit 001 |
| :--- | :--- | :--- | :--- | :--- |
| **`P04-ARCH-R1-001`** | **CRITICAL** | Worktree Authority | **WORKTREE_AUTHORITY_UNPROVEN**: Static code inspection against pinned AO commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6` in `backend/internal/adapters/workspace/gitworktree/workspace.go` confirmed path formula `<managedRoot>/<projectID>/<sessionID>` and branch `ao/<sessionID>`. However, configuration source of `managedRoot`, host authority, shared namespace, and restart/alias safety remain unproven. Requires dedicated proof plan and draft contract. | **PARTIALLY_CLOSED** |
| **`P04-ARCH-R1-002`** | **BLOCKER** | Security & Isolation | **VERIFICATION_ISOLATION_AMBIGUITY**: AppContainer selected as primary; Job Object bound; Restricted Token relegated to secondary fallback requiring external WFP firewall. Audited worktree made strictly read-only (`GENERIC_READ`). External writable sandbox root designed for toolchain caches. Desktop/Window Station isolation specified. 5-vector falsification matrix provided. Blocker remains OPEN pending runtime proof. | **SUBSTANTIVELY_CLOSED_AT_DESIGN_LEVEL** |
| **`P04-ARCH-R1-003`** | **CRITICAL** | Persistence & Schema | **MIGRATION_CAS_AND_ATOMICITY_DEFECTS**: Sequential migrations v6..v9 established; CAS ingestion query completed; `failure_reason` reaffirmed as audit/disposition attribute. However, pipeline atomicity defect requires adopting single-orchestrator Model 1 to guarantee zero partial rows across collector subtasks. | **PARTIALLY_CLOSED** |
| **`P04-ARCH-R1-004`** | **CRITICAL** | Review & Domain Model | **REVIEWABLE_EVIDENCE_PAYLOAD_DEFICIENCY**: Bounded Git patch and stdout/stderr text logs integrated into ReviewBundle; field name unified to `test_results`; circular hash removed. However, durable artifact persistence model (content-addressed store + `review_artifacts` table), full-stream vs captured-prefix hashing, and RFC 8785 canonical JSON hashing must be formalized. | **PARTIALLY_CLOSED** |
| **`P04-ARCH-R1-005`** | **MAJOR** | Git Evidence & Hardening | **GIT_EVIDENCE_SECURITY_AND_ANCESTRY_IMPRECISION**: Strict descendant rule clarified to `DESCENDANT_OR_EQUAL_POLICY` (`git merge-base --is-ancestor`); `MERGE_COMMITS_FORBIDDEN` enforced via `git rev-list --merges`; `GIT_TERMINAL_PROMPT=0` injected; explicit CLI flags pinned per subcommand; before/after HEAD TOCTOU check designed. | **SUBSTANTIVELY_CLOSED** |
| **`P04-ARCH-R1-006`** | **MAJOR** | Governance & Provenance | **PROVENANCE_AND_RECOVERY_SCENARIO_OVERCLAIM**: Code reference corrected to `internal/ao/client.go`; `SOURCE_REGISTRY.md` and `REUSE_MATRIX.md` cited; 5 design blockers synchronized; recovery scenarios renamed to proposed. Audit 001 citation erratum recorded in `ERRATUM_001`. | **PARTIALLY_CLOSED** |
| **`P04-ARCH-R2-001`** | **CRITICAL** | Worktree Authority | **WORKTREE_AUTHORITY_SEAM_AND_RUNTIME_BINDING_UNPROVEN**: Pinned AO source inspection proves path formula `<managedRoot>/<projectID>/<sessionID>` and branch `ao/<sessionID>`. However, 5 elements remain unproven: (1) host configuration source of `managedRoot`; (2) Supervisor authority to read it; (3) shared physical filesystem namespace; (4) restore/restart path preservation; (5) junction/alias/tamper resistance. Requires 3-track bounded empirical proof plan and draft TaskContract (NOT_RELEASED). | **OPEN** |
| **`P04-ARCH-R2-002`** | **BLOCKER** | Security & Isolation | **VERIFICATION_SANDBOX_WRITABLE_ROOT_AND_DESKTOP_ISOLATION_CONTRADICTION**: Resolving contradiction between read-only worktree and writable toolchain caches (`TEMP`, `TMP`, `GOCACHE`, `GOPATH`). Must place all writable directories under an attempt sandbox root outside the audited worktree (`.supervisor/sandboxes/<attempt_id>/`). Source snapshot created from verified HEAD (excluding untracked files). Window Station + Desktop isolation lifecycle. AppContainer SID ACL lifecycle and fail-closed probe. Compatibility proof matrix. | **OPEN** |
| **`P04-ARCH-R2-003`** | **CRITICAL** | Persistence & Review | **DURABLE_ARTIFACT_STORE_AND_CANONICAL_BUNDLE_HASHING_UNSPECIFIED**: Formalize durable artifact model using content-addressed filesystem storage `.supervisor/artifacts/<sha256>` with SQLite metadata table `review_artifacts`. Full stream SHA-256 computed alongside captured prefix SHA-256. Bounded artifact retrieval for P05 via opaque `artifact_id` (no raw filesystem paths). RFC 8785 JSON Canonicalization Scheme (JCS) for ReviewBundle canonical hash. | **OPEN** |
| **`P04-ARCH-R2-004`** | **CRITICAL** | Pipeline & Atomicity | **PIPELINE_ATOMICITY_AND_CONCURRENT_VERIFICATION_EXECUTION_SEAMS**: Adopt Model 1: P04B/P04C are pure in-memory collectors/runners with zero intermediate table commits; task remains `REPORT_READY` throughout collection; P04D is the sole orchestrator that atomically persists evidence, findings, test results, artifact refs, and transitions to `EVIDENCE_READY` in a single CAS transaction; one-winner reservation lease prevents concurrent runs; normal test failure (exit code != 0) treated as verified outcome distinct from runner infra failure. | **OPEN** |

---

## 3. Detailed Re-Audit Findings & Mandatory Remediations (Round 2)

### 3.1. P04-ARCH-R2-001: WORKTREE_AUTHORITY_SEAM_AND_RUNTIME_BINDING_UNPROVEN
- **Evaluation:** Direct source inspection of pinned AO (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`) in `backend/internal/adapters/workspace/gitworktree/workspace.go` authoritatively establishes:
  - `Workspace.Create` (lines 276, 287)
  - `Workspace.managedPath` (line 1887): `filepath.Join(w.managedRoot, string(cfg.ProjectID), string(cfg.SessionID))`
  - `defaultSessionBranchName` (line 1918): `"ao/" + string(id)`
  - `Workspace.Restore` (lines 1142, 1150)
  - `Options.ManagedRoot` (line 105)
  Therefore, worker session paths follow `<managedRoot>/<projectID>/<sessionID>` and branches follow `ao/<sessionID>`. This is a proven static fact, not a hypothetical assumption.
- **Unproven Seams:**
  1. Where `managedRoot` is configured on the host (flag, env, config file, default);
  2. Whether Supervisor has legal authority to read that configuration value;
  3. Whether AO and Supervisor execute in the identical physical filesystem namespace (or if path virtualization/WSL/containerization intervenes);
  4. Whether AO restart/restore deterministically preserves the identical physical path;
  5. Whether directory junctions, symlinks, or file locks could allow tampering or aliasing.
- **Mandatory Directives:**
  - Create a dedicated bounded empirical proof plan: [`docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md`](../plans/PLAN-P04-WORKTREE-BINDING-PROOF.md).
  - Create a draft JSON TaskContract: [`docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md`](../tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md).
  - Draft contract must remain **`NOT_RELEASED`**.
  - Must execute across 3 tracks:
    * **Track 1**: Static pinned-source proof of lifecycle, configuration seams, and options.
    * **Track 2**: Isolated runtime proof using AO v0.13.0 with dedicated disposable database, managed root, temporary Git repo, dual sessions, and zero autonomous agent coding.
    * **Track 3**: Formal authority conclusion choosing between Option A (trusted configuration seam + startup probe), Option B (upstream capability request / ADR), or Option C (fail-closed redesign). Option A cannot be selected unless runtime proof confirms identical physical namespace and configuration provenance.
  - Prohibit querying private AO SQLite databases.
  - Prohibit building custom unapproved worktree managers.
  - Do not treat `git worktree list` alone as authoritative; use it only to cross-check path formula against physical Git identity.

### 3.2. P04-ARCH-R2-002: VERIFICATION_SANDBOX_WRITABLE_ROOT_AND_DESKTOP_ISOLATION_CONTRADICTION
- **Evaluation:** Remediation round 1 correctly mandated read-only access for the audited worktree. However, running Go toolchains (`go test`) requires writable cache directories (`GOCACHE`, `GOPATH`, `TEMP`, `TMP`). Stating that the worktree is read-only while toolchains write inside it is a fatal contradiction.
- **Mandatory Directives:**
  - Designate an attempt sandbox root **outside** the audited worktree: `.supervisor/sandboxes/<attempt_id>/`.
  - Audited worktree is strictly read-only (`GENERIC_READ`, deny write/delete).
  - All toolchain write targets (`TEMP`, `TMP`, `GOCACHE`, `GOPATH`, scratch output) must be explicitly redirected into the external attempt sandbox.
  - For tests requiring in-tree file modifications, define a **source snapshot**:
    * Created strictly from verified commit `HEAD`;
    * Copies tracked Git content only; ignores untracked worker files;
    * Verified by SHA-256 content manifest;
    * Placed in disposable sandbox and destroyed after verification.
  - Windows Desktop Isolation: Calling `CreateDesktopW` alone does not provide window-station isolation unless a dedicated Window Station is created and bound (`CreateWindowStationW` + `CreateDesktopW`). Formally specify the combined Window Station + Desktop isolation lifecycle.
  - Formulate AppContainer SID ACL lifecycle: granting read-only access to Go SDK/dependencies, read/write access to sandbox, cleanup/rollback, and fail-closed probe.
  - Compatibility proof matrix for 5 vectors (Go toolchain, subprocesses, file read, cache write, network egress denial).
  - Keep `DESIGN_BLOCKER_P04_VERIFICATION_ISOLATION = OPEN`.

### 3.3. P04-ARCH-R2-003: DURABLE_ARTIFACT_STORE_AND_CANONICAL_BUNDLE_HASHING_UNSPECIFIED
- **Evaluation:** Incorporating bounded Git patches and text logs directly into ReviewBundle without durable persistence leaves artifact storage undefined, risks payload bloat, and leaves P05 unable to inspect larger outputs without raw filesystem traversal. Furthermore, merely stating "canonical JSON" without pinning an exact hashing algorithm is non-deterministic across JSON serializers.
- **Mandatory Directives:**
  - Adopt a **Content-Addressed Filesystem Artifact Store** (`.supervisor/artifacts/<sha256>`) paired with an immutable SQLite metadata table `review_artifacts`.
  - Schema for `review_artifacts`: `artifact_id`, `attempt_id`, `kind`, `media_type`, `encoding`, `full_stream_sha256`, `captured_sha256`, `original_bytes`, `captured_bytes`, `is_truncated`, `durable_location`, `created_at`.
  - Dual hashing: compute `full_stream_sha256` incrementally over the complete output stream as it is generated, alongside `captured_sha256` of the stored prefix. Do not claim the prefix hash is the stream hash.
  - Retrieval isolation: Phase P05 retrieves artifacts strictly via opaque `artifact_id` through the Control Plane API; raw filesystem paths are never exposed to P05.
  - Canonical ReviewBundle Hashing: Formalize canonical bundle hashing using **RFC 8785 (JSON Canonicalization Scheme - JCS)**. All fields except `bundle_hash` are included in the hash input.
  - Migration ownership: `review_artifacts` table owned by Schema v9 (P04D).

### 3.4. P04-ARCH-R2-004: PIPELINE_ATOMICITY_AND_CONCURRENT_VERIFICATION_EXECUTION_SEAMS
- **Evaluation:** Allowing subtasks P04B and P04C to write intermediate rows to `evidence`, `policy_findings`, and `actual_test_results` tables contradicts the requirement of zero partial evidence rows during pipeline failures. Furthermore, preventing duplicate INSERTs via unique constraints is insufficient to stop redundant concurrent verification processes from executing external commands.
- **Mandatory Directives:**
  - Formally adopt **Model 1**:
    * P04B (Git Evidence & Policy Engine) and P04C (Verification Runner) are **pure in-memory collectors and execution engines**. They perform zero SQLite writes.
    * Sequential migrations v6..v9 are defined, but production write APIs are wired exclusively in P04D.
    * The task remains in `REPORT_READY` throughout the entire collection and verification pipeline.
    * P04D is the sole orchestrator that persists evidence, policy findings, test results, artifact records, audit entries, and transitions state to `EVIDENCE_READY` inside a **single atomic SQLite CAS transaction**.
  - Concurrency Prevention: Implement a **one-winner verification reservation lease** prior to running external commands. The loser is denied execution and executes no commands.
  - Normal Test Failure Handling: If a verification command completes with a non-zero exit code (e.g. `go test` failed), this is a **valid verified outcome** with structured exit code and logs, NOT an infrastructure failure. It must be distinguished from runner infrastructure failure (timeout, sandbox crash, OOM).
  - Include transaction and fault injection matrix.

---

## 4. Required Action & Governance Guardrails

1. **Remediation Target**: Update `PROPOSAL-P04-001`, `DRAFT-ADR-018`, and `PLAN-P04-EVIDENCE-REVIEW` to Revision 3.
2. **Dedicated Proof Artifacts**: Create `docs/plans/PLAN-P04-WORKTREE-BINDING-PROOF.md` and `docs/tasks/DRAFT_TASK_CONTRACT_P04_WORKTREE_BINDING_PROOF.md` (keep `NOT_RELEASED`).
3. **Execution Guardrails**:
   - `ACTIVE_GATE = P04_PRECONTRACT_ARCHITECTURE_REMEDIATION_2`
   - `P04_TASK_CONTRACT = NOT_RELEASED`
   - `P04_CODE = HELD_PENDING_APPROVED_ARCHITECTURE_AND_TASK_CONTRACT`
   - `P05_CODE = NOT_AUTHORIZED`
   - `AUTOMATIC_RESTORE = DISABLED`
   - `LIVE_AO_INTEGRATION = UNVERIFIED_EVIDENCE_TRACK`
   - `VERIFIED_OPERATOR_PRINCIPAL = OPEN_FAIL_CLOSED_DEPENDENCY`
   - `STAGE_B_RUNTIME_CATALOG = DEFERRED_TO_P04_RUNTIME_INTEGRATION`
   - Strictly 9 files in whitelist; zero Go code, zero schema changes, zero migrations.
