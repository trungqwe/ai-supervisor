# PHASE SPECIFICATION: P03 — AO INTEGRATION (EXECUTION ADAPTER)

## 1. Objective
Implement `AOAdapter` to connect the Supervisor Control Plane to the AO daemon REST API and event streams.

## 2. Deliverables
- `AOAdapter` implementation mapping domain calls to AO endpoints.
- Event listener streaming worker lifecycle events (`worker_started`, `worker_heartbeat`, `worker_finished`).
- Automated contract tests asserting zero domain pollution from AO DTOs.

## 3. Exit Gate
- Automated integration tests successfully spawn an AO worker session, send test instructions, and cleanly terminate.
