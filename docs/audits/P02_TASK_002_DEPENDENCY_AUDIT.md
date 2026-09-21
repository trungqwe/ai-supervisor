# TASK-P02-002 DEPENDENCY AUDIT DOSSIER

> **Audit Record**: `P02_TASK_002_DEPENDENCY_AUDIT.md`  
> **Date**: 2026-09-22  
> **Auditor**: External Supervisor / Engineering Governance  
> **Subject**: Direct Dependency for TaskContract Structural Validation  
> **Verdict**: `DEPENDENCY_GOOGLE_JSONSCHEMA_GO = APPROVED_FOR_TASK_P02_002`  

---

## 1. Module Identification & Facts

- **Module**: `github.com/google/jsonschema-go`
- **Target Package**: `github.com/google/jsonschema-go/jsonschema`
- **Target Version**: `v0.4.3`
- **Release / Tag**: `refs/tags/v0.4.3` (commit `8c4ab4f02ef64dcea5502e47a6113e8292944087`)
- **License**: MIT License (Permissive open source)
- **Go Floor Requirement**: `go 1.23.0` (Well within the project language floor of `go 1.26.0`)
- **JSON Schema Dialect**: Full validation support for JSON Schema Draft-07 (required for `task-contract.schema.json`), 2020-12, 2019-09, Draft-06, and Draft-04.

---

## 2. Direct Dependency Justification & Transitive Graph

1. **Direct Dependency Justification**:
   - Required by `FR-004` (Contract Validation) and `SEC-002` (Contract Tamper Resistance) to perform runtime structural validation of incoming raw TaskContract JSON before typed domain instantiation.
   - Provides strict validation of types, arrays, required fields, and `additionalProperties: false` without ad-hoc custom validation logic.

2. **Transitive Module Footprint**:
   - Production package `github.com/google/jsonschema-go/jsonschema` imports **only** standard library packages (`bytes`, `context`, `encoding/json`, `errors`, `fmt`, `io`, `net/url`, `regexp`, `sort`, `strings`, `sync`, etc.).
   - The upstream `require github.com/google/go-cmp v0.7.0` in `go.mod` is used exclusively for unit testing in the upstream repository.
   - Runtime transitive external dependencies added to `ai-supervisor`: **0**.

3. **Future MCP SDK Alignment**:
   - The official Model Context Protocol Go SDK (`github.com/modelcontextprotocol/go-sdk`) adopted `github.com/google/jsonschema-go` (`v0.3.0` in SDK v0.8.0; `v0.4.2` in SDK v1.4.0).
   - Selecting `github.com/google/jsonschema-go v0.4.3` aligns Phase P02 directly with future Phase P05 architecture, avoiding duplicate JSON Schema engines.

---

## 3. Known Limitations & Security Posture

1. **No Network Schema Resolution**:
   - In accordance with `SEC-002` and P02 architecture directives, schema validation must be purely local and offline.
   - The validator must not configure an HTTP/HTTPS remote resource loader. Canonical `task-contract.schema.json` and profile schemas contain no network `$ref` dependencies.
2. **Security Claims Precision**:
   - No absolute claim of "zero vulnerabilities" is made; rather, risk is strictly minimized through source audit, MIT licensing, Google maintenance, minimal surface area, and zero third-party runtime dependencies.

---

## 4. Decision

```
DEPENDENCY_GOOGLE_JSONSCHEMA_GO = APPROVED_FOR_TASK_P02_002
AUTHORIZED_VERSION = v0.4.3
PACKAGE = github.com/google/jsonschema-go/jsonschema
DIRECT_EXTERNAL_DEPENDENCY_COUNT = 1
TRANSITIVE_RUNTIME_EXTERNAL_DEPENDENCY_COUNT = 0
```
