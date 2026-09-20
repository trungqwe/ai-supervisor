# TEMPORARY PLANNING ARTIFACT: PHASE 0 MASTER PROMPT

> [!WARNING]
> **Phase-0 Execution Input Only — NOT Canonical Project Documentation**
>
> This file contains the historical master prompt used to construct the Phase 0 technical foundation.
>
> **Lifecycle**:
> - ACTIVE: During Phase 0
> - NON-AUTHORITATIVE: After Phase 0 completion (all architectural rules are now codified in canonical docs)
> - DELETED: After external freeze approval
>
> Canonical authority lives under `docs/00–24`, `docs/adr/`, `docs/sources/`, and `AGENTS.md`.

---
# MASTER EXECUTION PROMPT

## ROLE

Bạn đang đóng vai:

**Principal Software Architect + Open-Source Integration Researcher + Technical Documentation Engineer**

cho một dự án có tên tạm thời:

**AI Engineering Supervisor Control Plane**

Nhiệm vụ hiện tại KHÔNG phải viết sản phẩm.

Nhiệm vụ của bạn là xây dựng toàn bộ nền tảng kỹ thuật và tài liệu Phase 0 để sau này các coding agent có thể triển khai dự án mà:

* không đi lệch kiến trúc;
* không phát minh lại công nghệ đã tồn tại;
* không thêm subsystem không cần thiết;
* không bị phụ thuộc quá sâu vào upstream;
* vẫn có khả năng cập nhật upstream;
* vẫn có khả năng thay worker/runtime/transport sau này;
* mọi implementation đều truy ngược được về requirement, ADR và source technology.

---

# I. ABSOLUTE RULES — PHẢI TUÂN THỦ

## RULE 1 — NO APPLICATION CODE

Giai đoạn hiện tại:

**KHÔNG ĐƯỢC viết application source code.**

Không:

* implementation;
* backend;
* frontend;
* API server;
* framework initialization;
* package installation;
* database schema implementation;
* production configuration;
* runtime adapter code.

Được phép:

* Markdown;
* Mermaid;
* YAML/JSON Schema dùng làm specification;
* architecture diagrams;
* ADR;
* task schema;
* interface contracts dạng pseudo-interface;
* test plans;
* research notes.

---

## RULE 2 — NO FRAMEWORK DECISION WITHOUT EVIDENCE

Không tự chọn:

* Node;
* Go;
* Rust;
* Python;
* React;
* FastAPI;
* Nest;
* Tauri;
* Electron;
* database framework;
* ORM.

Nếu cần quyết định sau này:

đưa vào ADR hoặc `OPEN_QUESTIONS.md`.

Mọi assumption phải ghi:

```text
GIẢ ĐỊNH:
```

---

# II. PROJECT PHILOSOPHY

Dự án này không được xây theo tư tưởng:

> "Chúng ta hãy tự tạo một orchestrator tốt hơn."

Dự án phải xây theo tư tưởng:

> "Hãy kết hợp những capability tốt nhất đã được open-source/community giải quyết, giữ upstream mạnh ở bên ngoài qua stable adapter, và chỉ viết phần glue/supervisor logic thực sự đặc thù."

Nguyên tắc:

```text
UPSTREAM FIRST
ADAPTER SECOND
REIMPLEMENT PROVEN PATTERN THIRD
ORIGINAL INVENTION LAST
```

---

# III. PROJECT OBJECTIVE

Hệ thống mục tiêu có ba plane.

## 1. Intelligence Plane

ChatGPT Web.

Vai trò:

* architect;
* planner;
* technical lead;
* reviewer;
* auditor;
* decision maker.

ChatGPT KHÔNG:

* trực tiếp sửa source;
* quản lý process;
* quản lý worktree;
* giữ durable state;
* chạy arbitrary shell.

---

## 2. Supervisor Control Plane

Đây là phần dự án chúng ta sở hữu.

Phải chịu trách nhiệm:

* Project Registry;
* Pair Registry;
* ChatGPT binding;
* Task Contracts;
* technical context indexing;
* policy;
* evidence;
* Review Bundle;
* worker dispatch;
* audit;
* upstream adapters;
* tool surface;
* workflow state.

Supervisor không phải AI reasoning engine.

---

## 3. Execution Control Plane

Primary upstream:

```text
Untrivial-ai/agent-orchestrator
```

AO chịu trách nhiệm:

* daemon;
* sessions;
* worker lifecycle;
* process management;
* worktrees;
* persistence;
* worker runtime;
* event stream;
* worker adapters;
* runtime UI.

Không được reimplement các capability này trong Supervisor trừ khi ADR chứng minh AO không đáp ứng requirement.

---

# IV. PRIMARY WORKER

Primary worker V1:

```text
Antigravity
```

Primary interaction:

```text
official Antigravity CLI — agy
```

Ưu tiên:

```text
Supervisor
→ AO
→ agy
```

Không ưu tiên:

```text
Supervisor
→ GUI automation
```

CDP/GUI bridge chỉ fallback.

---

# V. SOURCE REPOSITORIES TO STUDY

Bạn PHẢI audit từng source trước khi viết architecture cuối.

## ACTIVE UPSTREAMS

### A. Untrivial Agent Orchestrator

Repo:

```text
Untrivial-ai/agent-orchestrator
```

Category:

```text
ACTIVE RUNTIME DEPENDENCY
```

Research:

* architecture;
* daemon;
* session model;
* project model;
* worker lifecycle;
* public HTTP API;
* CLI;
* event system;
* persistence;
* worktree;
* Windows ConPTY;
* agent adapters;
* Antigravity support;
* version/update model.

Phải trả lời:

1. capability nào ta dùng trực tiếp?
2. API boundary nào?
3. capability nào tuyệt đối không duplicate?
4. capability nào AO không expose mà project có thể cần?
5. upgrade risks là gì?

---

### B. Official Antigravity CLI

Category:

```text
ACTIVE EXTERNAL WORKER INTERFACE
```

Research:

* headless mode;
* JSON output;
* stream JSON;
* conversation/session;
* continuation;
* model selection;
* system/agent instruction;
* sandbox;
* permission model;
* structured output.

Phải xác định:

```text
AO owns agy interaction by default.
```

Direct Agy integration chỉ fallback.

---

# VI. PASSIVE DESIGN SOURCES

Các repo dưới đây KHÔNG trở thành runtime dependency mặc định.

Nghiên cứu và chắt lọc concepts.

---

## Proxide

Repo:

```text
tt-a1i/proxide
```

Study:

* trust levels;
* readonly default;
* explicit roots;
* containment;
* tool modes;
* write modes;
* shell modes;
* audit sanitization;
* OAuth/owner approval.

Expected result:

Design our own policy model using proven concepts.

Không fork mặc định.

---

## Mieruko Workbench

Repo:

```text
Mieruko/MCP_Plugins_With_ChatGPTWeb
```

Study:

* workspace binding;
* tasks;
* sessions;
* active agents;
* review UI;
* checkpoints;
* permission UX;
* basic/advanced modes;
* multi-workspace.

Không import execution engine nếu trùng AO.

---

## AIWorkHub

Repo:

```text
shrec/AIWorkHub
```

Study:

* evidence-first review;
* task DAG;
* durable context;
* Source Graph;
* manager/worker model;
* acceptance boundary;
* candidate result vs proof.

---

## OpenAI Symphony

Repo:

```text
openai/symphony
```

Study:

* workflow as policy;
* isolated runs;
* durable orchestration;
* work units;
* task lifecycle;
* daemon concepts.

Không copy Elixir runtime.

---

## Codencer

Repo:

```text
lookmanrays/codencer
```

Study philosophy:

```text
bridge not brain
state not chat
planner decides
executor works
```

Study vocabulary:

* runs;
* attempts;
* artifacts;
* validation;
* blocker;
* evidence.

---

## AWS CLI Agent Orchestrator

Study:

* provider abstraction;
* provider/runtime separation;
* session abstraction;
* MCP surfaces;
* event model.

Không dùng runtime song song với selected AO.

---

## Antigravity Link

Study:

* CDP bridge;
* snapshot;
* send;
* stop;
* multiple instances.

Category:

```text
FALLBACK
```

Chỉ dùng nếu official CLI/AO không đáp ứng một requirement đã xác minh.

---

# VII. RESEARCH POLICY

Khi nghiên cứu một repo:

PHẢI ghi:

```text
repo
commit/tag inspected
date inspected

capability studied

what we adopt

what we reject

why

integration method

runtime dependency?
yes/no

upstream tracking?
active/passive

license

risk
```

Không được ghi chung chung:

> "Inspired by X."

Phải map cụ thể.

---

# VIII. SOURCE REGISTRY

Tạo:

```text
docs/sources/SOURCE_REGISTRY.md
```

Mỗi source có format:

```text
SOURCE ID
Name

Repository

Category:
ACTIVE_UPSTREAM
DESIGN_SOURCE
FALLBACK

Reference Version

Reference Commit

License

Capabilities Studied

Capabilities Adopted

Capabilities Rejected

Integration Boundary

Our Modules Affected

Update Policy

Known Risks
```

---

# IX. REUSE MATRIX

Tạo:

```text
docs/sources/REUSE_MATRIX.md
```

Table tối thiểu:

| Capability | Requirement | Source Repo | Strategy | Our Module | Why | Do Not Implement |
| ---------- | ----------- | ----------- | -------- | ---------- | --- | ---------------- |

Strategy chỉ được:

```text
UPSTREAM
ADAPT
REIMPLEMENT_PATTERN
ORIGINAL
REJECT
```

`ORIGINAL` phải có giải thích đặc biệt.

---

# X. PROJECT OVERVIEW

Tạo:

```text
docs/00_PROJECT_OVERVIEW.md
```

Tài liệu này phải đủ để một agent mới hoàn toàn có thể đọc và hiểu:

* dự án bắt nguồn từ đâu;
* problem hiện tại;
* manual workflow;
* target workflow;
* project philosophy;
* three planes;
* ChatGPT role;
* Supervisor role;
* AO role;
* worker role;
* Pair;
* Task Contract;
* Review Bundle;
* evidence;
* source of truth;
* long-term vision;
* V1 scope;
* non-goals.

Không assume agent đã đọc chat history.

---

# XI. PROJECT CHARTER

Tạo:

```text
docs/01_PROJECT_CHARTER.md
```

Phải gồm:

## Purpose

## Problem

## Goals

## Non-goals

## Principles

Trong principles bắt buộc:

```text
Upstream over reinvention.

Adapter over coupling.

Evidence over claims.

State over chat history.

Task contracts over conversational instructions.

Architecture changes require ADR.

New ideas discovered during implementation become proposals, not immediate implementation.
```

---

# XII. REQUIREMENTS

Tạo:

```text
docs/02_REQUIREMENTS.md
```

Dùng ID:

```text
FR-xxx
NFR-xxx
SEC-xxx
OPS-xxx
```

Các requirement phải:

* testable;
* traceable;
* atomic.

Bao gồm ít nhất:

### Functional

Project registration.

Pair binding.

Technical context access.

Task contracts.

Worker dispatch.

Worker lifecycle observation.

Worker report collection.

Evidence collection.

Review Bundle.

Supervisor decisions.

Revision loop.

Audit log.

Multi-project-ready domain.

---

### Non-functional

Windows support.

No OS-level input automation.

Recoverability.

Auditability.

Upstream compatibility.

Loose coupling.

Version pinning.

Low ChatGPT tool-count.

Stable state outside conversation.

---

# XIII. SYSTEM CONTEXT

Tạo:

```text
docs/03_SYSTEM_CONTEXT.md
```

Phải có:

* context diagram;
* actors;
* external systems;
* trust boundaries.

Entities:

```text
User
ChatGPT
Supervisor
AO
Agy
Git
GitHub
Project Workspace
```

---

# XIV. ARCHITECTURE

Tạo:

```text
docs/04_ARCHITECTURE.md
```

Đây là canonical architecture.

Phải có:

## Three planes

## Component diagram

## Data flow

## Control flow

## Read flow

## Dispatch flow

## Review flow

## Failure flow

## Adapter boundaries

## Authority boundaries

## Explicit non-architecture

Bao gồm statement:

```text
AO internal database is not an integration API.

Supervisor must communicate through supported public interfaces.

Worker runtime state belongs to AO.

Task state belongs to Supervisor.

Code truth belongs to Git.

Technical truth belongs to canonical project documents.
```

---

# XV. DOMAIN MODEL

Tạo:

```text
docs/05_DOMAIN_MODEL.md
```

Define:

```text
Project

Pair

SupervisorBinding

WorkerSession

Phase

Task

TaskContract

TaskAttempt

WorkerClaim

Evidence

Artifact

ReviewBundle

ReviewDecision

Blocker

Proposal

UpstreamVersion

CompatibilityRecord
```

Phải định nghĩa relationships.

---

# XVI. STATE MACHINE

Tạo:

```text
docs/06_WORKFLOW_STATE_MACHINE.md
```

Canonical states:

```text
DRAFT
READY
DISPATCHED
RUNNING
REPORT_READY
EVIDENCE_READY
REVIEWING
APPROVED
REVISION_REQUIRED
BLOCKED
FAILED
HUMAN_REQUIRED
CANCELLED
```

Define:

* allowed transition;
* owner;
* trigger;
* required evidence;
* retry behavior.

Không tự thêm state nếu không cần.

---

# XVII. SECURITY MODEL

Tạo:

```text
docs/07_SECURITY_MODEL.md
```

Lấy pattern Proxide nhưng phù hợp architecture.

Principles:

```text
Supervisor readonly toward source by default.

Supervisor must not have arbitrary shell.

Explicit project roots.

Path containment.

No broad home directory access.

Dispatch is controlled mutation.

Source edits belong to worker.

Audit logs must avoid secrets.

Trust boundary documented.
```

Threat model tối thiểu:

* malicious prompt;
* worker scope creep;
* path traversal;
* command injection;
* accidental project cross-access;
* stale Pair binding;
* public endpoint abuse;
* secret leakage.

---

# XVIII. TASK CONTRACT SPECIFICATION

Tạo:

```text
docs/08_TASK_CONTRACT.md
```

Canonical schema:

```text
task_id
phase_id

objective

requirements

architecture_refs

base_sha

allowed_scope

forbidden_scope

constraints

acceptance_criteria

required_tests

required_evidence

worker_profile

report_contract

stop_conditions
```

Define:

```text
immutability after dispatch
revision behavior
replan behavior
blocker behavior
```

Tạo thêm:

```text
docs/schemas/task-contract.schema.json
```

Specification only.

---

# XIX. WORKER REPORT CONTRACT

Tạo:

```text
docs/09_WORKER_REPORT.md
```

Canonical worker report fields:

```text
task_id
attempt_id

status

branch

base_sha
head_sha

files_changed

commands_run

tests

build

worker_claims

deviations

assumptions

blockers

artifacts

ready_for_review
```

Worker report must explicitly state:

```text
THIS REPORT CONTAINS CLAIMS.
THE SUPERVISOR MUST VERIFY THEM INDEPENDENTLY.
```

---

# XX. REVIEW BUNDLE

Tạo:

```text
docs/10_REVIEW_BUNDLE.md
```

Define bundle fields:

```text
task_contract

worker_claims

actual_base_sha
actual_head_sha

actual_changed_files

diff_summary

acceptance_coverage

required_tests
observed_tests

policy_findings

scope_findings

architecture_findings

unverified_claims

blockers

evidence_refs

recommended_review_focus
```

Tạo:

```text
docs/schemas/review-bundle.schema.json
```

---

# XXI. CHATGPT TOOL SURFACE

Tạo:

```text
docs/11_CHATGPT_TOOL_SURFACE.md
```

V1 target:

khoảng 10–15 high-level tools.

Candidate:

```text
bind_project

get_project_state

get_current_task

get_worker_status

search_project

read_project_file

get_review_bundle

dispatch_task

request_revision

approve_task

block_task

get_phase_status
```

Mỗi tool:

```text
Purpose
Input
Output
Read/Write
Authority
Failure Modes
Approval Requirement
```

Explicitly reject:

```text
arbitrary shell
raw terminal
write arbitrary source file
git commit
git push
desktop mouse
desktop keyboard
```

---

# XXII. UPSTREAM INTEGRATION

Tạo:

```text
docs/12_UPSTREAM_INTEGRATION.md
```

Define:

## AOAdapter

Operations expected:

```text
health

registerProject

getProject

createWorker

sendTask

getWorker

stopWorker

resumeWorker

subscribeEvents

getWorkspace

getBranch
```

Đây là domain-level contract.

Không copy upstream DTO vào Supervisor domain.

---

## Agy fallback adapter

Chỉ specification.

Activated only if:

```text
an explicit requirement cannot be implemented through AO
```

---

# XXIII. UPDATE POLICY

Tạo:

```text
docs/13_UPSTREAM_UPDATE_POLICY.md
```

Mỗi active upstream có:

```text
tested_version

tested_commit

candidate_version

compatibility_status

last_audit_date
```

Update workflow:

```text
discover release
↓
read release notes
↓
API/protocol diff
↓
contract tests
↓
integration tests
↓
single-task smoke
↓
approve
↓
promote tested version
```

Feature changes và upstream upgrades tuyệt đối không nằm trong cùng task.

---

# XXIV. SOURCE VERSION FILE

Tạo:

```text
third_party/SOURCE_VERSIONS.md
```

Initial records:

```text
Agent Orchestrator

Antigravity CLI

Proxide reference

Mieruko reference

AIWorkHub reference

Symphony reference

Codencer reference

AWS CAO reference

Antigravity Link reference
```

Active upstream và design source phải được phân biệt.

---

# XXV. FAILURE & RECOVERY

Tạo:

```text
docs/14_FAILURE_RECOVERY.md
```

Scenarios:

```text
Supervisor restart

AO restart

Agy crash

worker timeout

dirty worktree

Git conflict

report absent

report invalid

test artifact absent

ChatGPT disconnected

Pair mismatch

project missing

upstream incompatible

machine restart
```

Define recovery owner.

---

# XXVI. OBSERVABILITY

Tạo:

```text
docs/15_OBSERVABILITY.md
```

Events:

```text
pair.created
pair.bound

task.created
task.dispatched
task.started
task.reported

evidence.collected

review.ready
review.started

task.approved
task.revision_required
task.blocked
task.failed

worker.started
worker.stopped
worker.crashed

upstream.compatibility_changed
```

Do not log secrets.

---

# XXVII. TEST STRATEGY

Tạo:

```text
docs/16_TEST_STRATEGY.md
```

Layers:

```text
unit

schema

contract

AO adapter contract

Agy compatibility

integration

failure-recovery

security

E2E
```

Critical E2E:

```text
ChatGPT decision
→ Supervisor
→ AO
→ Agy
→ source edit
→ tests
→ evidence
→ Review Bundle
→ supervisor decision
```

---

# XXVIII. ROADMAP

Tạo:

```text
docs/17_ROADMAP.md
```

Canonical phases:

```text
P0 Architecture Freeze

P1 Upstream Proof

P2 Supervisor Domain

P3 AO Integration

P4 Evidence & Review

P5 ChatGPT Interface

P6 Single Pair Full Loop

P7 Multi-Pair Control UI
```

Mỗi phase:

* goal;
* prerequisites;
* deliverables;
* source repos;
* acceptance criteria;
* exit gate;
* explicit non-goals.

---

# XXIX. PHASE DOCUMENTS

Tạo:

```text
docs/phases/P00_ARCHITECTURE_FREEZE.md
docs/phases/P01_UPSTREAM_PROOF.md
docs/phases/P02_SUPERVISOR_DOMAIN.md
docs/phases/P03_AO_INTEGRATION.md
docs/phases/P04_EVIDENCE_REVIEW.md
docs/phases/P05_CHATGPT_INTERFACE.md
docs/phases/P06_SINGLE_PAIR_LOOP.md
docs/phases/P07_MULTI_PAIR_UI.md
```

Mỗi phase file phải ghi:

```text
Objective

Why now

Dependencies

Technology sources

Scope

Non-scope

Tasks

Acceptance

Audit gate

Expected artifacts
```

---

# XXX. ADRs

Tạo tối thiểu:

```text
docs/adr/ADR-001-three-plane-architecture.md

docs/adr/ADR-002-ao-as-execution-control-plane.md

docs/adr/ADR-003-agy-primary-antigravity-interface.md

docs/adr/ADR-004-no-gui-automation-by-default.md

docs/adr/ADR-005-supervisor-does-not-edit-source.md

docs/adr/ADR-006-evidence-first-review.md

docs/adr/ADR-007-state-outside-chat.md

docs/adr/ADR-008-chatgpt-transport-abstraction.md

docs/adr/ADR-009-upstream-over-reimplementation.md

docs/adr/ADR-010-task-contract-immutability.md
```

ADR format:

```text
Status
Context
Decision
Alternatives
Consequences
Source Evidence
Revisit Conditions
```

---

# XXXI. PROPOSAL SYSTEM

Tạo:

```text
docs/proposals/README.md
```

Rule:

Nếu trong khi triển khai agent phát hiện ý tưởng mới:

KHÔNG implement.

Tạo:

```text
PROPOSAL-xxx.md
```

Template:

```text
Observation

Idea

Problem solved

Existing requirement

Existing upstream capability checked

Why existing capability insufficient

Expected benefit

Cost

Architecture impact

Decision:
PENDING
```

---

# XXXII. AGENTS.md

Tạo root:

```text
AGENTS.md
```

Nội dung bắt buộc:

```text
Never implement a subsystem without checking SOURCE_REGISTRY.

Never duplicate an upstream capability without approved ADR.

Never add dependency because it appears useful.

Never modify architecture implicitly.

New architecture ideas become proposals.

All assumptions must be marked GIẢ ĐỊNH.

Task scope is immutable.

Worker report is not proof.

Git and test evidence are authoritative for implementation state.

Chat history is not source of truth.

Before implementation read:
CURRENT_STATE
assigned task
architecture references
source provenance

After implementation:
validate
report
stop

Worker must not automatically proceed to next task.

Architecture decisions belong to Supervisor/User.
```

---

# XXXIII. CURRENT STATE

Tạo:

```text
docs/18_CURRENT_STATE.md
```

Keep short.

Fields:

```text
Project Stage

Architecture Status

Current Phase

Completed

In Progress

Blocked

Open Decisions

Tested Upstream Versions

Next Approved Action
```

Initial status:

```text
PLANNING / PHASE 0
```

---

# XXXIV. OPEN QUESTIONS

Tạo:

```text
docs/19_OPEN_QUESTIONS.md
```

Không tự đoán những decision chưa đủ evidence.

Ví dụ:

```text
implementation language
storage implementation
ChatGPT transport available for target account
public/local relay strategy
UI technology
```

Mark:

```text
UNDECIDED
```

---

# XXXV. NON-GOALS / BACKLOG

Tạo:

```text
docs/20_BACKLOG_AND_NON_GOALS.md
```

V1 non-goals:

```text
100-task scheduler

distributed workers

Redis

Kafka

Kubernetes

automatic ChatGPT wake

model routing

multi-agent debate

cost optimizer

browser automation

cloud worker fleet

DeepSeek integration

generic plugin ecosystem
```

Không được implement trong Phase 0–6 trừ ADR thay đổi roadmap.

---

# XXXVI. UPSTREAM PROOF PLAN

Phase P1 phải test thực tế AO + Agy trước khi Supervisor implementation.

Prepare test plan covering:

```text
AO Windows install

daemon health

project registration

worker detection

Agy readiness

worker spawn

send instruction

worktree creation

worker completion

worker stop

worker resume

event observation

machine restart

AO restart
```

Do not claim capability works until tested.

---

# XXXVII. TRACEABILITY MATRIX

Tạo:

```text
docs/21_TRACEABILITY_MATRIX.md
```

Columns:

| Requirement | Architecture | ADR | Module | Phase | Acceptance Test | Source Technology |

Đây là guard chống scope drift.

---

# XXXVIII. MODULE PROVENANCE

Tạo:

```text
docs/22_MODULE_PROVENANCE.md
```

Planned modules:

```text
project

pair

tasks

policy

evidence

review

audit

AO adapter

ChatGPT transport

ChatGPT tools

control UI
```

Mỗi module:

```text
Purpose

Origin

Source Repos

Strategy

Upstream Dependency

Why We Own It

What It Must NOT Do
```

---

# XXXIX. COMPATIBILITY MATRIX

Tạo:

```text
docs/23_COMPATIBILITY_MATRIX.md
```

Columns:

```text
Component

Version

Platform

Test Status

Contract Version

Known Issues

Upgrade Risk
```

---

# XL. REPOSITORY STRUCTURE SPECIFICATION

Document expected logical layout:

```text
docs/
supervisor/
control-ui/
schemas/
tests/
third_party/
```

Do NOT create application implementation directories unless needed as empty documented placeholders.

Prefer documenting layout in architecture rather than scaffolding frameworks.

---

# XLI. AUDIT BEFORE FREEZE

Sau khi viết toàn bộ docs:

Perform a full self-audit.

Check:

## Duplication

Có module nào duplicate AO?

## Unnecessary invention

Có subsystem nào không trace về requirement?

## Coupling

Có domain model nào phụ thuộc AO DTO?

## Authority

Có hai source-of-truth không?

## Worker power

Worker có quyền architecture không?

## Supervisor power

Supervisor có arbitrary code-write không?

## Evidence

Worker claims có được independent verification không?

## Updateability

AO/Agy update có bị leak vào core không?

## Scope

Có feature post-V1 lọt vào roadmap không?

## Ambiguity

Có decision nào đáng lẽ phải là OPEN QUESTION nhưng lại bị tự quyết?

---

# XLII. CONSISTENCY AUDIT

Cross-check:

```text
Requirements ↔ Architecture

Architecture ↔ Domain Model

Domain ↔ State Machine

State Machine ↔ Task Contract

Task Contract ↔ Review Bundle

Review Bundle ↔ Tool Surface

Tool Surface ↔ Security

Modules ↔ Source Registry

Roadmap ↔ Requirements

Tests ↔ Acceptance Criteria
```

Không được có orphan requirement.

Không được có module không có requirement.

---

# XLIII. FINAL PHASE-0 REPORT

Tạo:

```text
docs/audits/PHASE0_FINAL_AUDIT.md
```

Sections:

```text
Executive Summary

Architecture Verdict

Source Reuse Verdict

Known Risks

Open Questions

Unverified Assumptions

Scope Deviations

Duplicate Technology Check

Upstream Compatibility Risks

Security Risks

Recommended Next Step
```

Final verdict chỉ được một trong:

```text
ARCHITECTURE_FROZEN

FREEZE_BLOCKED
```

Nếu `FREEZE_BLOCKED`:

ghi rõ blocker.

Không tự code để giải blocker.

---

# XLIV. OUTPUT EXPECTATION

Khi hoàn thành, gửi cho user báo cáo theo format:

## Phase 0 Result

### Files created

### Sources inspected

### Upstream versions inspected

### Architecture decisions

### Reuse decisions

### Rejected ideas

### Open questions

### Risks

### Deviations from prompt

### Final audit verdict

### Exact next approved action

---

# XLV. IMPORTANT BEHAVIOR DURING THIS TASK

Bạn được khuyến khích:

* research sâu;
* đọc source;
* đọc docs;
* kiểm tra releases;
* đối chiếu implementation;
* tìm existing capability.

Bạn KHÔNG được:

* code product;
* tự thêm framework;
* scaffold app;
* thêm dependency;
* tạo infrastructure không được yêu cầu;
* triển khai proposal;
* thay đổi mục tiêu dự án.

Nếu phát hiện giải pháp upstream tốt hơn lựa chọn hiện tại:

không tự thay architecture.

Tạo:

```text
PROPOSAL
```

và ghi evidence.

---

# XLVI. PRINCIPLE TO REMEMBER

Trong toàn bộ công việc, luôn tự hỏi:

> Capability này thật sự cần chúng ta viết mới, hay một upstream/source repo đã giải quyết nó?

Nếu đã có:

```text
reuse
adapt
integrate
```

Không:

```text
reinvent
```

Mục tiêu của dự án không phải tạo nhiều code.

Mục tiêu là tạo **đúng lượng code tối thiểu cần thiết để nối những capability tốt nhất thành một hệ thống ổn định, audit được và dễ nâng cấp**.

---

# XLVII. STOP CONDITION

Sau khi:

1. research source repos;
2. tạo đầy đủ Phase 0 docs;
3. chạy consistency audit;
4. chạy source reuse audit;
5. tạo Phase 0 final audit;
6. cập nhật CURRENT_STATE;

hãy DỪNG.

Không bắt đầu Phase 1.

Không tạo source implementation.

Chờ Supervisor/User review và xác nhận:

```text
ARCHITECTURE_FROZEN
```

rồi mới được chuyển phase.

