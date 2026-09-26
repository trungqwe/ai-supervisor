# External Supervisor Contract Release Audit: TASK-P03-004 Revision 2

- **Task ID**: `TASK-P03-004` (Host Quiescence & Daemon Bootstrap Integration)
- **Contract ID**: `CONTRACT-TASK-P03-004-02`
- **Revision Number**: `2`
- **Candidate Commit**: `e41f1e76abd95fd6143fcf22537ac66feddce9b9`
- **Candidate Blob**: `b436fae01a903b6bcace0fe40b3c070b959b9455` (`docs/tasks/CANDIDATE_TASK_CONTRACT_P03_004_REVISION_2.md`)
- **Released Contract**: `docs/tasks/TASK_CONTRACT_P03_004_REVISION_2.md`
- **Code Base SHA**: `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`
- **Release Verdict**: `RELEASE_APPROVED_WITH_MANDATORY_TEXTUAL_ERRATUM`
- **Proposal Status**: `PROPOSAL-P03-006 = EXTERNAL_APPROVED`
- **Active Gate**: `TASK_P03_004_IMPLEMENTATION`
- **Code Gate**: `P03_CODE = AUTHORIZED_004_ONLY`

---

## 1. Audit Evaluation & Approval Summary

External Supervisor audited candidate contract `CONTRACT-TASK-P03-004-02` at candidate commit `e41f1e76abd95fd6143fcf22537ac66feddce9b9` (blob `b436fae01a903b6bcace0fe40b3c070b959b9455`).
- The JSON object conforms strictly to `docs/08_TASK_CONTRACT.md` and schema validation against `docs/schemas/task-contract.schema.json`.
- Semantic validation against `internal/contract/validator.go` passes cleanly with zero errors.
- Monotonic revision lineage correctly records `supersedes_contract_id: CONTRACT-TASK-P03-004-01`.
- The architectural reconciliation of `PROPOSAL-P03-006` is formally **`EXTERNAL_APPROVED`**.
- Narrowed whitelist extension permits 5 specific files across `recovery` and `ao` while keeping all remaining files in `forbidden_scope`.
- Acceptance criteria `AC-004-01` through `AC-004-08` are preserved, augmented by `AC-004-09` (Poller lifecycle & host watcher fail-closed) and `AC-004-10` (Typed AO workspace file retrieval with JSON envelope validation).
- Evaluation policy catalog delta for `go-test-p03-004` (adding `internal/recovery` and `internal/ao`) is `APPROVED_AT_DESIGN_LEVEL`; Stage B host runtime remains `UNVERIFIED` until dispatch.

The External Supervisor grants **`RELEASE_APPROVED_WITH_MANDATORY_TEXTUAL_ERRATUM`** with the following errata resolved during release handoff:

1. **PROPOSAL-P03-006 Required Fields Erratum**:
   - Section 2.2 #3 is updated to explicitly enumerate all 15 required fields of the pinned AO `WorkspaceFileResponse` schema (`15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`): `sessionId`, `path`, `content`, `binary`, `deleted`, `contentTruncated`, `size`, `status`, `workspaceVersion`, `diff`, `diffTruncated`, `editable`, `fileFingerprint`, `additions`, `deletions`, alongside the status enum validation (`unmodified`, `modified`, `added`, `deleted`).
2. **PROPOSAL-P03-006 Test Harness Filename Erratum**:
   - All references to the integration test harness filename in `PROPOSAL-P03-006` are corrected to `test/integration/ao_harness_test.go`.
3. **Task Contract Release Promulgation**:
   - `docs/tasks/TASK_CONTRACT_P03_004_REVISION_2.md` is promulgated with proper `RELEASED` header, status, authority, and invariants.
   - All fields and values of the canonical JSON TaskContract are preserved verbatim from candidate blob `b436fae01a903b6bcace0fe40b3c070b959b9455`.
   - `docs/tasks/CANDIDATE_TASK_CONTRACT_P03_004_REVISION_2.md` and baseline `docs/tasks/TASK_CONTRACT_P03_004.md` are preserved intact as immutable historical audit evidence.

---

## 2. Invariants & Implementation Protocol

1. **Isolation of Release & Implementation**:
   - Release artifact commit is created and pushed on `main` separately before implementation begins on `codex/p03-004`.
   - Implementation branch is `codex/p03-004` based strictly on `35909d7b21cdfe6b9f5c309ea565c5f9f9fedeea`.
   - Governance artifacts must NOT appear in the implementation diff.
2. **Implementation Scope Discipline**:
   - Permitted files strictly match `allowed_scope` of Revision 2:
     * `cmd/supervisor/**`
     * `internal/host/**`
     * `internal/recovery/poller.go`
     * `internal/recovery/poller_test.go`
     * `internal/ao/client.go`
     * `internal/ao/types.go`
     * `internal/ao/client_test.go`
     * `test/integration/**`
   - All other files in `internal/recovery/**` and `internal/ao/**` remain strictly in `forbidden_scope`.
3. **Runtime Invariants**:
   - `AUTOMATIC_RESTORE = DISABLED`.
   - Verified host principal remains `OPEN` dependency at trusted boundary until runtime completion evidence exists.
   - Zero Phase P04/P05 dependencies. Zero arbitrary shell primitives.
