# SOURCE DOSSIER: 09 — ANTIGRAVITY LINK

## 1. Metadata
- **Repository**: `antigravity-link` (Community Tool)
- **Role in Architecture**: Fallback Mechanism Only (CDP GUI Bridge)
- **Reference Snapshot**: `2026-09-01`
- **Inspection Date**: 2026-09-20
- **License**: MIT
- **Primary Language**: TypeScript / Node.js

---

# 2. Capabilities Evaluated & Adopted
- **CDP Bridge Mechanism**: Communicating with an active Antigravity GUI instance via Chrome DevTools Protocol.
- **Session Interception**: Ability to snapshot active session state if headless CLI is completely unavailable.

---

# 3. Capabilities Explicitly Rejected
- **Default Workflow Role**: Strictly rejected as a primary workflow. GUI automation is fragile and intrusive.
- **Pixel-Based Automation**: Any coordinate-based mouse clicking is strictly forbidden.

---

# 4. Integration Strategy & Revisit Conditions
- **Strategy**: `FALLBACK` only.
- **Activation Rule**: This mechanism is activated **ONLY IF** an explicit, verified functional requirement cannot be executed via the official Antigravity CLI or Agent Orchestrator.

---

# 5. SOURCE EVIDENCE

### Evidence Item 9.1: CDP Session Connectivity
- **Claim**: Chrome DevTools Protocol can bridge programmatic instructions to an existing Antigravity Electron process.
- **Repository / Snapshot**: `antigravity-link`
- **Source Module**: `src/cdp/bridge.ts`
- **Verification Status**: VERIFIED
- **Confidence**: MEDIUM (Brittle across IDE updates; hence retained strictly as fallback)
