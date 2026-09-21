# P02 TASK-002 External Reaudit Record (Revision 4)

## Audit Metadata

| Field | Value |
|---|---|
| **Audited Remote Commit** | `e378ce29e8a8ce90573a3546450d4da880b1e232` |
| **Original Task Baseline** | `87fa16a001a825753bec9e3c5dd36d511e4c5318` |
| **Audit Date** | 2026-09-22 |
| **Phase Gate** | `P02_TASK_002_REVISION_4` |

---

## Verdict

```
TASK_P02_002 = REVISION_REQUIRED
TASK_P02_002_REVISION = 4
P02_003 = NOT_RELEASED
```

---

## Revision-3 Accepted Fixes

```
REV3_PROFILE_POLICY_FIX = RESOLVED
REV3_IMMUTABILITY_FIX = RESOLVED
REV3_ROOT_FAIL_OPEN = PARTIALLY_RESOLVED
```

---

## Revision-4 Findings

### Code Findings

```
P02T002-R4-001 = WINDOWS_COMPONENT_WALK_SEPARATOR_BYPASS
P02T002-R4-002 = STRUCTURAL_SCHEMA_VALIDATION_OPTIONAL
P02T002-R4-003 = NON_INTEGER_SCHEMA_PROJECTION_PRECISION_LOSS
```

#### P02T002-R4-001 — WINDOWS_COMPONENT_WALK_SEPARATOR_BYPASS
`filepath.Clean` converts path separators to the OS-native separator (`\` on Windows). Slicing or splitting cleaned output using literal `/` results in a single component on Windows, bypassing component-by-component walk and intermediate symlink/directory inspection.
Required fix: Normalize untrusted relative path using `filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.ReplaceAll(trimmed, "\\", "/"))))` for canonical splitting and traversal checks; construct physical paths from each canonical segment. In addition, reject paths with leading/trailing whitespace without silent trimming. Add a privilege-free regression test where an intermediate component is a regular file.

#### P02T002-R4-002 — STRUCTURAL_SCHEMA_VALIDATION_OPTIONAL
`NewValidator` allowed `len(canonicalSchemaBytes) == 0` creating schema-less instances where `ParseAndValidateRaw` bypassed structural schema checks.
Required fix: `NewValidator` must require non-empty compiled canonical schema; `ParseAndValidateRaw` must return deterministic `SCHEMA` error if `resolved == nil`. Tests must not use `nil` schema instances.

#### P02T002-R4-003 — NON_INTEGER_SCHEMA_PROJECTION_PRECISION_LOSS
Floating-point projection converted non-integral `json.Number` values to `float64` without verifying that the `float64` representation is mathematically exact, risking silent precision loss and false schema acceptance (e.g. `0.1000000000000000000000001` rounding to `0.1`).
Required fix: Parse `json.Number` using `math/big.Rat`. Check if integer mathematically (denominator == 1); if non-integral, project to `float64` only if converting the `float64` back to `big.Rat` yields exact equality with the original rational. Otherwise fail closed with numeric projection precision error.

---

### Evidence Finding

```
P02T002-R4-004 = CLEAN_DETACHED_WORKTREE_EXECUTION_NOT_PROVEN
```

Previous Revision-3 transcript created a clean detached worktree and used:
`git -C clean-worktree ...` for repository inventory.
However subsequent:
`go mod verify`, `go build ./...`, `go test`, `go vet`
were not explicitly executed with: `go -C clean-worktree ...` and no Set-Location/Push-Location command proved execution directory change.

Therefore:
```
CLEAN_WORKTREE_INVENTORY = PASS
CLEAN_WORKTREE_EXECUTION = NOT_PROVEN
```

Classification:
```
PROCESS_DEVIATION = EVIDENCE_ACCURACY_DEVIATION
Architecture impact = NONE
Source impact = NONE
Acceptance impact = REQUIRES_CORRECT_REEXECUTION
```

All build and verification commands must explicitly target the detached worktree using `go -C $cleanPath ...`.

---

## Revision Contract

| Field | Value |
|---|---|
| contract_id | CONTRACT-TASK-P02-002-04 |
| task_id | TASK-P02-002 |
| revision_number | 4 |
| supersedes_contract_id | CONTRACT-TASK-P02-002-03 |
| phase_id | P02 |
| base_sha | 87fa16a001a825753bec9e3c5dd36d511e4c5318 (immutable per ADR-012) |
