# P04_TASK_001_CONTRACT_RELEASE_AUDIT_ERRATUM_001.md

> **Target Audit**: [P04_TASK_001_CONTRACT_RELEASE_AUDIT.md](P04_TASK_001_CONTRACT_RELEASE_AUDIT.md)
> **Authority**: External Supervisor
> **Classification**: Append-Only Record Erratum (Release Artifact Provenance & Wrapper Integrity)
> **Status**: APPROVED
> **Date**: 2026-09-28

---

## 1. Context & Purpose

During the external audit of Subtask P04A implementation (`docs/audits/P04_TASK_001_EXTERNAL_AUDIT_001.md`), finding `P04A-I1-001` identified provenance and wrapper character discrepancies in the release artifact committed at `dda382dee2c64456dc281768275082be73107056`:
1. In `docs/audits/P04_TASK_001_CONTRACT_RELEASE_AUDIT.md`, Section 2, Item 2 and Section 4 recorded the released contract blob as `1f5e31774705470beb914a3c3c75d95d5bc2d845`.
2. The actual Git blob SHA of `docs/tasks/TASK_CONTRACT_P04_001.md` at commit `dda382dee2c64456dc281768275082be73107056` is `aebcdaf942410f24655d81c143a543683079e4c4`.
3. The released file wrapper contains 3 ASCII Bell (`BEL`, `0x07`) bytes at offsets 1317, 1942, and 35335, as well as a typographical artifact string `"ull"`.

Pursuant to `docs/24_CHANGE_GOVERNANCE.md`, this append-only erratum formally rectifies the provenance record and establishes the verified integrity boundaries without in-place mutation of historical release artifacts.

---

## 2. Erratum Specifications & Provenance Verification

1. **Authoritative Blob SHA Rectification**:
   - The actual Git blob SHA of released contract `docs/tasks/TASK_CONTRACT_P04_001.md` at commit `dda382dee2c64456dc281768275082be73107056` is:
     `aebcdaf942410f24655d81c143a543683079e4c4`
   - The blob SHA `1f5e31774705470beb914a3c3c75d95d5bc2d845` cited in `P04_TASK_001_CONTRACT_RELEASE_AUDIT.md` is formally recorded as an erroneous citation.

2. **Wrapper Defect Scope**:
   - The 3 `BEL` bytes (offset 1317 in table separator, offset 1942 in bullet marker, offset 35335 in footer section) and the string `"ull"` reside strictly in the markdown prose wrapper.
   - The raw JSON contract block enclosed between ` ```json ` and ` ``` ` has been independently verified:
     * Raw JSON block Git blob SHA: `7344a11cf22c0262a52fe598c938c532d56f5b8c`
     * This hash is 100% byte-for-byte identical to candidate JSON block blob `7344a11cf22c0262a52fe598c938c532d56f5b8c` extracted from approved candidate `docs/tasks/CANDIDATE_TASK_CONTRACT_P04_001.md` (file blob `75dadfd8a3cc53ff361602a1d5ae5bbe7794fbd7`).
   - The normative task contract semantics, allowed_scope, forbidden_scope, triggers, and acceptance criteria remain 100% intact and uncorrupted.

3. **Immutability Directive**:
   - Released contract `docs/tasks/TASK_CONTRACT_P04_001.md` and historical release audit `P04_TASK_001_CONTRACT_RELEASE_AUDIT.md` shall NOT be edited in-place.
   - Any clean wrapper replacement shall follow formal governance corrigendum procedures if required by External Supervisor.

4. **Finding Status**:
   - Finding `P04A-I1-001` is formally documented and rectified via this append-only erratum.
