# P03_TASK_002_REMEDIATION_001.md - TASK-P03-002 Revision 1 Remediation Record

> **Audited Commit**: `056629ff4d38bbf4caaa94fc860825a2e9991a7c`
> **Upstream Authority**: `Untrivial-ai/agent-orchestrator` v0.13.0 (Commit `15e9ea971f1711ec8b50e157d6eb300db6cbe0d6`)
> **Remediation Date**: 2026-09-22
> **Remediation Verdict**: `P03T2R1-001 = CLOSED_PENDING_EXTERNAL_REAUDIT`
> **Active Gate**: `EXTERNAL_SUPERVISOR_P03_TASK_002_REAUDIT`
> **Governance State**: `TASK_P03_002 = REVISION_1_READY_FOR_EXTERNAL_REAUDIT`, `P03_CODE = HELD_FOR_EXTERNAL_AUDIT`, `TASK_P03_003 = NOT_RELEASED`

---

## 1. Remediation Summary for P03T2R1-001

| Finding | Title | Remediation Status | Implementation Details |
|---|---|---|---|
| `P03T2R1-001` | `ACTIVITY_PROTOCOL_ERROR_STATUS_MISREPORTED` | **CLOSED_PENDING_EXTERNAL_REAUDIT** | Made `validateCanonicalActivityState` status-aware by accepting explicit `statusCode int`. Callers pass exact observed HTTP status: `CreateWorkerSession` passes `http.StatusCreated` (201); `GetWorkerStatus` and `ResumeWorker` pass `http.StatusOK` (200). ProtocolError accurately preserves observed response status. |

---

## 2. Implementation Evidence

### 2.1 internal/ao/sessions.go
```go
// validateCanonicalActivityState validates raw state string fail-closed against canonical specification.
// statusCode preserves the actual HTTP response status observed on the wire.
func validateCanonicalActivityState(rawState string, statusCode int, method string, path string) (ActivityState, error) {
	switch rawState {
	case string(ActivityStateActive):
		return ActivityStateActive, nil
	case string(ActivityStateIdle):
		return ActivityStateIdle, nil
	case string(ActivityStateWaitingInput):
		return ActivityStateWaitingInput, nil
	case string(ActivityStateBlocked):
		return ActivityStateBlocked, nil
	case string(ActivityStateExited):
		return ActivityStateExited, nil
	default:
		return "", &ProtocolError{
			StatusCode: statusCode,
			Method:     method,
			Path:       path,
			Reason:     fmt.Sprintf("unknown activity state %q (must be active, idle, waiting_input, blocked, or exited)", rawState),
		}
	}
}
```

In `GetWorkerStatus`:
```go
state, err := validateCanonicalActivityState(wire.Session.Activity.State, http.StatusOK, http.MethodGet, path)
```

### 2.2 internal/ao/session_commands.go
In `CreateWorkerSession`:
```go
state, err := validateCanonicalActivityState(resp.Session.Activity.State, http.StatusCreated, http.MethodPost, path)
```

In `ResumeWorker`:
```go
state, err := validateCanonicalActivityState(resp.Session.Activity.State, http.StatusOK, http.MethodPost, path)
```

---

## 3. Test Oracle Evidence

### 3.1 Spawn Unknown Activity (HTTP 201)
`internal/ao/session_commands_test.go`:
```go
t.Run("spawn_unknown_activity_reports_status_201", func(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"unsupported"}},"promptBytes":0,"systemPromptBytes":0}`))
	}))
	defer s.Close()

	c, _ := newTestClient(t, s)
	_, err := c.CreateWorkerSession(context.Background(), "p1", "agy")
	...
	if protoErr.StatusCode != http.StatusCreated {
		t.Errorf("expected StatusCode %d (HTTP 201), got %d", http.StatusCreated, protoErr.StatusCode)
	}
	if protoErr.Method != http.MethodPost {
		t.Errorf("expected Method %s, got %s", http.MethodPost, protoErr.Method)
	}
	if protoErr.Path != "/api/v1/sessions" {
		t.Errorf("expected Path %s, got %s", "/api/v1/sessions", protoErr.Path)
	}
})
```
Test output: `PASS`

### 3.2 GetWorkerStatus Unknown Activity (HTTP 200)
`internal/ao/client_test.go`:
```go
_, err := c.GetWorkerStatus(context.Background(), "sess-unknown-state")
var getProtoErr *ProtocolError
if !errors.As(err, &getProtoErr) {
	t.Fatalf("expected *ProtocolError for GetWorkerStatus unknown activity, got %T (%v)", err, err)
}
if getProtoErr.StatusCode != http.StatusOK {
	t.Errorf("expected StatusCode %d for GetWorkerStatus unknown activity, got %d", http.StatusOK, getProtoErr.StatusCode)
}
if getProtoErr.Method != http.MethodGet {
	t.Errorf("expected Method %s, got %s", http.MethodGet, getProtoErr.Method)
}
if getProtoErr.Path != "/api/v1/sessions/sess-unknown-state" {
	t.Errorf("expected Path %s, got %s", "/api/v1/sessions/sess-unknown-state", getProtoErr.Path)
}
```
Test output: `PASS`

### 3.3 ResumeWorker Unknown Activity (HTTP 200)
`internal/ao/session_commands_test.go`:
```go
t.Run("restore_unknown_activity_reports_status_200", func(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"sessionId":"s1","restoreMode":"native","session":{"id":"s1","activity":{"state":"hibernating"}}}`))
	}))
	defer s.Close()

	c, _ := newTestClient(t, s)
	_, err := c.ResumeWorker(context.Background(), "s1")
	...
	if protoErr.StatusCode != http.StatusOK {
		t.Errorf("expected StatusCode %d (HTTP 200), got %d", http.StatusOK, protoErr.StatusCode)
	}
	if protoErr.Method != http.MethodPost {
		t.Errorf("expected Method %s, got %s", http.MethodPost, protoErr.Method)
	}
	if protoErr.Path != "/api/v1/sessions/s1/restore" {
		t.Errorf("expected Path %s, got %s", "/api/v1/sessions/s1/restore", protoErr.Path)
	}
})
```
Test output: `PASS`
