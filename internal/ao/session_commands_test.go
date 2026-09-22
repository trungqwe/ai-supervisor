package ao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// 1. CreateWorkerSession Tests (Section 7, Section 14.A)
// -----------------------------------------------------------------------------
func TestClient_CreateWorkerSession(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	t.Run("successful_spawn_minimal_contract", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/sessions" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}

			body, _ := io.ReadAll(r.Body)
			var rawMap map[string]any
			if err := json.Unmarshal(body, &rawMap); err != nil {
				t.Fatalf("failed to decode JSON request: %v", err)
			}

			// Must contain EXACTLY projectId, kind, harness. Zero other keys!
			if len(rawMap) != 3 {
				t.Errorf("expected exactly 3 keys in payload, got %d: %v", len(rawMap), rawMap)
			}
			if p, ok := rawMap["projectId"]; !ok || p != "proj-123" {
				t.Errorf("unexpected projectId in body: %v", p)
			}
			if k, ok := rawMap["kind"]; !ok || k != "worker" {
				t.Errorf("unexpected kind in body (must be 'worker'): %v", k)
			}
			if h, ok := rawMap["harness"]; !ok || h != "agy" {
				t.Errorf("unexpected harness in body (must preserve 'agy'): %v", h)
			}

			// Prohibited keys MUST NOT be present
			for _, forbiddenKey := range []string{"prompt", "branch", "mode", "model", "displayName", "issueId", "trackerProvider", "attachments", "config"} {
				if _, ok := rawMap[forbiddenKey]; ok {
					t.Errorf("prohibited key %q present in spawn body", forbiddenKey)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(fmt.Sprintf(`{
				"session": {
					"id": "sess-spawned-1",
					"projectId": "proj-123",
					"kind": "worker",
					"harness": "agy",
					"status": "idle",
					"isTerminated": false,
					"activity": {
						"state": "idle",
						"lastActivityAt": %q
					}
				},
				"promptBytes": 0,
				"systemPromptBytes": 1024
			}`, nowStr)))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)

		// Successful call with harness "agy"
		res, err := c.CreateWorkerSession(context.Background(), "proj-123", "agy")
		if err != nil {
			t.Fatalf("unexpected CreateWorkerSession error: %v", err)
		}
		if res.Session.ID != "sess-spawned-1" {
			t.Errorf("expected session ID 'sess-spawned-1', got %q", res.Session.ID)
		}
		if res.Session.ProjectID != "proj-123" {
			t.Errorf("expected project ID 'proj-123', got %q", res.Session.ProjectID)
		}
		if res.Session.Harness != "agy" {
			t.Errorf("expected harness 'agy', got %q", res.Session.Harness)
		}
		if res.Session.Activity.State != ActivityStateIdle {
			t.Errorf("expected activity state 'idle', got %q", res.Session.Activity.State)
		}
		if res.PromptBytes != 0 || res.SystemPromptBytes != 1024 {
			t.Errorf("expected promptBytes=0, systemPromptBytes=1024, got %d, %d", res.PromptBytes, res.SystemPromptBytes)
		}
	})

	t.Run("input_validation", func(t *testing.T) {
		c, _ := NewClient("http://127.0.0.1:3001", &http.Client{Timeout: time.Second})

		if _, err := c.CreateWorkerSession(context.Background(), "", "agy"); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on empty projectID, got %v", err)
		}
		if _, err := c.CreateWorkerSession(context.Background(), "   ", "agy"); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on whitespace projectID, got %v", err)
		}
		if _, err := c.CreateWorkerSession(context.Background(), "p1", ""); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on empty harness, got %v", err)
		}
		if _, err := c.CreateWorkerSession(context.Background(), "p1", "   "); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on whitespace harness, got %v", err)
		}
	})

	t.Run("response_validation_and_failures", func(t *testing.T) {
		cases := []struct {
			name        string
			respStatus  int
			respBody    string
			expectedErr error
		}{
			{
				name:        "wrong_status_200",
				respStatus:  http.StatusOK,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"idle"}},"promptBytes":0,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "null_session",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":null,"promptBytes":0,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "empty_session_id",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"idle"}},"promptBytes":0,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "mismatched_project_id",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"other-proj","kind":"worker","harness":"agy","activity":{"state":"idle"}},"promptBytes":0,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "mismatched_harness",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"claude-code","activity":{"state":"idle"}},"promptBytes":0,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "wrong_kind",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"orchestrator","harness":"agy","activity":{"state":"idle"}},"promptBytes":0,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "missing_prompt_bytes",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"idle"}},"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "negative_prompt_bytes",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"idle"}},"promptBytes":-1,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "missing_system_prompt_bytes",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"idle"}},"promptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "negative_system_prompt_bytes",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"idle"}},"promptBytes":0,"systemPromptBytes":-5}`,
				expectedErr: ErrProtocolViolation,
			},
			{
				name:        "unknown_activity_state",
				respStatus:  http.StatusCreated,
				respBody:    `{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"unsupported"}},"promptBytes":0,"systemPromptBytes":0}`,
				expectedErr: ErrProtocolViolation,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.respStatus)
					w.Write([]byte(tc.respBody))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				_, err := c.CreateWorkerSession(context.Background(), "p1", "agy")
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.expectedErr != nil && !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected errors.Is(err, %v), got: %v", tc.expectedErr, err)
				}
			})
		}

		t.Run("spawn_unknown_activity_reports_status_201", func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"session":{"id":"s1","projectId":"p1","kind":"worker","harness":"agy","activity":{"state":"unsupported"}},"promptBytes":0,"systemPromptBytes":0}`))
			}))
			defer s.Close()

			c, _ := newTestClient(t, s)
			_, err := c.CreateWorkerSession(context.Background(), "p1", "agy")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, ErrProtocolViolation) {
				t.Fatalf("expected errors.Is(err, ErrProtocolViolation), got %v", err)
			}
			var protoErr *ProtocolError
			if !errors.As(err, &protoErr) {
				t.Fatalf("expected *ProtocolError, got %T (%v)", err, err)
			}
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
	})
}

// -----------------------------------------------------------------------------
// 2. DispatchTaskContract Tests (Section 9, Section 14.B)
// -----------------------------------------------------------------------------
func TestClient_DispatchTaskContract(t *testing.T) {
	t.Run("successful_dispatch", func(t *testing.T) {
		messagePayload := `{"contractId":"TASK-1","instructions":"Run verification"}`

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.RequestURI != "/api/v1/sessions/sess-123/send" {
				t.Errorf("unexpected request URI: %s %s", r.Method, r.RequestURI)
			}

			body, _ := io.ReadAll(r.Body)
			var rawMap map[string]any
			if err := json.Unmarshal(body, &rawMap); err != nil {
				t.Fatalf("failed to decode JSON request: %v", err)
			}

			// Wire request body must contain exactly "message"
			if len(rawMap) != 1 {
				t.Errorf("expected exactly 1 key, got %d: %v", len(rawMap), rawMap)
			}
			msg, ok := rawMap["message"].(string)
			if !ok || msg != messagePayload {
				t.Errorf("message was altered! got %q, want %q", msg, messagePayload)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			respBytes, _ := json.Marshal(map[string]any{
				"ok":        true,
				"sessionId": "sess-123",
				"message":   messagePayload,
			})
			w.Write(respBytes)
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)
		res, err := c.DispatchTaskContract(context.Background(), "sess-123", messagePayload)
		if err != nil {
			t.Fatalf("unexpected DispatchTaskContract error: %v", err)
		}
		if res.SessionID != "sess-123" {
			t.Errorf("expected SessionID 'sess-123', got %q", res.SessionID)
		}
		if res.Message != messagePayload {
			t.Errorf("expected returned Message %q, got %q", messagePayload, res.Message)
		}
	})

	t.Run("input_validation", func(t *testing.T) {
		c, _ := NewClient("http://127.0.0.1:3001", &http.Client{Timeout: time.Second})

		if _, err := c.DispatchTaskContract(context.Background(), "", "msg"); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on empty sessionID, got %v", err)
		}
		if _, err := c.DispatchTaskContract(context.Background(), "   ", "msg"); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on whitespace sessionID, got %v", err)
		}
		if _, err := c.DispatchTaskContract(context.Background(), "s1", ""); !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest on empty message, got %v", err)
		}
	})

	t.Run("url_path_escaping_reserved_chars", func(t *testing.T) {
		reservedCases := []struct {
			sessionID   string
			expectedURI string
		}{
			{"sess/slash", "/api/v1/sessions/sess%2Fslash/send"},
			{"sess?query", "/api/v1/sessions/sess%3Fquery/send"},
			{"sess#frag", "/api/v1/sessions/sess%23frag/send"},
			{"sess%pct", "/api/v1/sessions/sess%25pct/send"},
		}

		for _, tc := range reservedCases {
			t.Run(tc.sessionID, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.RequestURI != tc.expectedURI {
						t.Errorf("URI restructuring! got %q, want %q", r.RequestURI, tc.expectedURI)
					}
					if r.URL.RawQuery != "" {
						t.Errorf("query string was injected! %q", r.URL.RawQuery)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(fmt.Sprintf(`{"ok":true,"sessionId":%q,"message":"ok"}`, tc.sessionID)))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				res, err := c.DispatchTaskContract(context.Background(), tc.sessionID, "hello")
				if err != nil {
					t.Fatalf("DispatchTaskContract(%q) error: %v", tc.sessionID, err)
				}
				if res.SessionID != tc.sessionID {
					t.Errorf("expected sessionID %q, got %q", tc.sessionID, res.SessionID)
				}
			})
		}
	})

	t.Run("ao_400_message_too_long_propagated", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"bad_request","code":"MESSAGE_TOO_LONG","message":"Message is too long"}`))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)
		_, err := c.DispatchTaskContract(context.Background(), "s1", "huge-message")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrBadRequest) {
			t.Errorf("expected ErrBadRequest, got %v", err)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.Code != "MESSAGE_TOO_LONG" {
			t.Errorf("expected Code 'MESSAGE_TOO_LONG', got %q", apiErr.Code)
		}
	})

	t.Run("response_validation_failures", func(t *testing.T) {
		cases := []struct {
			name        string
			respStatus  int
			respBody    string
			expectedErr error
		}{
			{"wrong_status_201", http.StatusCreated, `{"ok":true,"sessionId":"s1","message":"m"}`, ErrProtocolViolation},
			{"ok_false", http.StatusOK, `{"ok":false,"sessionId":"s1","message":"m"}`, ErrProtocolViolation},
			{"empty_sessionId", http.StatusOK, `{"ok":true,"sessionId":"","message":"m"}`, ErrProtocolViolation},
			{"mismatched_sessionId", http.StatusOK, `{"ok":true,"sessionId":"other","message":"m"}`, ErrProtocolViolation},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.respStatus)
					w.Write([]byte(tc.respBody))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				_, err := c.DispatchTaskContract(context.Background(), "s1", "msg")
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected %v, got %v", tc.expectedErr, err)
				}
			})
		}
	})
}

// -----------------------------------------------------------------------------
// 3. StopWorker Tests (Section 10, Section 14.C)
// -----------------------------------------------------------------------------
func TestClient_StopWorker(t *testing.T) {
	t.Run("successful_kill_freed_true", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.RequestURI != "/api/v1/sessions/sess-kill-1/kill" {
				t.Errorf("unexpected request: %s %s", r.Method, r.RequestURI)
			}
			body, _ := io.ReadAll(r.Body)
			if len(body) > 0 {
				t.Errorf("expected empty body for kill request, got %s", string(body))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true,"sessionId":"sess-kill-1","freed":true}`))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)
		res, err := c.StopWorker(context.Background(), "sess-kill-1")
		if err != nil {
			t.Fatalf("unexpected StopWorker error: %v", err)
		}
		if res.SessionID != "sess-kill-1" || !res.Freed {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("successful_kill_freed_false_preserved", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true,"sessionId":"sess-kill-2","freed":false}`))
		}))
		defer s.Close()

		c, _ := newTestClient(t, s)
		res, err := c.StopWorker(context.Background(), "sess-kill-2")
		if err != nil {
			t.Fatalf("unexpected StopWorker error: %v", err)
		}
		if res.SessionID != "sess-kill-2" || res.Freed != false {
			t.Errorf("expected Freed=false, got %+v", res)
		}
	})

	t.Run("url_path_escaping_reserved_chars", func(t *testing.T) {
		reservedCases := []struct {
			sessionID   string
			expectedURI string
		}{
			{"sess/slash", "/api/v1/sessions/sess%2Fslash/kill"},
			{"sess?query", "/api/v1/sessions/sess%3Fquery/kill"},
			{"sess#frag", "/api/v1/sessions/sess%23frag/kill"},
			{"sess%pct", "/api/v1/sessions/sess%25pct/kill"},
		}

		for _, tc := range reservedCases {
			t.Run(tc.sessionID, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.RequestURI != tc.expectedURI {
						t.Errorf("URI restructuring! got %q, want %q", r.RequestURI, tc.expectedURI)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(fmt.Sprintf(`{"ok":true,"sessionId":%q,"freed":true}`, tc.sessionID)))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				res, err := c.StopWorker(context.Background(), tc.sessionID)
				if err != nil {
					t.Fatalf("StopWorker(%q) error: %v", tc.sessionID, err)
				}
				if res.SessionID != tc.sessionID {
					t.Errorf("expected sessionID %q, got %q", tc.sessionID, res.SessionID)
				}
			})
		}
	})

	t.Run("validation_failures", func(t *testing.T) {
		cases := []struct {
			name        string
			sessionID   string
			respStatus  int
			respBody    string
			expectedErr error
		}{
			{"empty_sessionID", "", http.StatusOK, "", ErrBadRequest},
			{"whitespace_sessionID", "   ", http.StatusOK, "", ErrBadRequest},
			{"wrong_status_202", "s1", http.StatusAccepted, `{"ok":true,"sessionId":"s1","freed":true}`, ErrProtocolViolation},
			{"ok_false", "s1", http.StatusOK, `{"ok":false,"sessionId":"s1","freed":true}`, ErrProtocolViolation},
			{"empty_response_sessionID", "s1", http.StatusOK, `{"ok":true,"sessionId":"","freed":true}`, ErrProtocolViolation},
			{"mismatched_response_sessionID", "s1", http.StatusOK, `{"ok":true,"sessionId":"other","freed":true}`, ErrProtocolViolation},
			{"ao_404_session_not_found", "s1", http.StatusNotFound, `{"error":"not_found","code":"SESSION_NOT_FOUND","message":"session not found"}`, ErrSessionNotFound},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.respStatus)
					w.Write([]byte(tc.respBody))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				_, err := c.StopWorker(context.Background(), tc.sessionID)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected %v, got %v", tc.expectedErr, err)
				}
			})
		}
	})
}

// -----------------------------------------------------------------------------
// 4. ResumeWorker Tests (Section 11, Section 14.D)
// -----------------------------------------------------------------------------
func TestClient_ResumeWorker(t *testing.T) {
	now := time.Date(2026, 9, 22, 13, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	t.Run("restore_modes_native_saved_prompt_fresh", func(t *testing.T) {
		modes := []struct {
			rawMode      string
			expectedMode RestoreMode
		}{
			{"native", RestoreModeNative},
			{"saved_prompt", RestoreModeSavedPrompt},
			{"fresh", RestoreModeFresh},
		}

		for _, tc := range modes {
			t.Run(tc.rawMode, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.RequestURI != "/api/v1/sessions/sess-restore-1/restore" {
						t.Errorf("unexpected request: %s %s", r.Method, r.RequestURI)
					}
					body, _ := io.ReadAll(r.Body)
					if len(body) > 0 {
						t.Errorf("expected empty body for restore, got %s", string(body))
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(fmt.Sprintf(`{
						"ok": true,
						"sessionId": "sess-restore-1",
						"restoreMode": %q,
						"session": {
							"id": "sess-restore-1",
							"projectId": "p1",
							"status": "working",
							"isTerminated": false,
							"activity": {
								"state": "active",
								"lastActivityAt": %q
							}
						}
					}`, tc.rawMode, nowStr)))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				res, err := c.ResumeWorker(context.Background(), "sess-restore-1")
				if err != nil {
					t.Fatalf("unexpected ResumeWorker error: %v", err)
				}
				if res.SessionID != "sess-restore-1" {
					t.Errorf("expected sessionID 'sess-restore-1', got %q", res.SessionID)
				}
				if res.RestoreMode != tc.expectedMode {
					t.Errorf("expected RestoreMode %q, got %q", tc.expectedMode, res.RestoreMode)
				}
				if res.Session.Activity.State != ActivityStateActive {
					t.Errorf("expected ActivityStateActive, got %q", res.Session.Activity.State)
				}
			})
		}
	})

	t.Run("canonical_activity_states_on_restore", func(t *testing.T) {
		states := []ActivityState{
			ActivityStateActive,
			ActivityStateIdle,
			ActivityStateWaitingInput,
			ActivityStateBlocked,
			ActivityStateExited,
		}

		for _, state := range states {
			t.Run(string(state), func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(fmt.Sprintf(`{
						"ok": true,
						"sessionId": "s1",
						"restoreMode": "native",
						"session": {
							"id": "s1",
							"projectId": "p1",
							"status": "working",
							"activity": {"state": %q}
						}
					}`, string(state))))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				res, err := c.ResumeWorker(context.Background(), "s1")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.Session.Activity.State != state {
					t.Errorf("expected state %q, got %q", state, res.Session.Activity.State)
				}
			})
		}
	})

	t.Run("url_path_escaping_reserved_chars", func(t *testing.T) {
		reservedCases := []struct {
			sessionID   string
			expectedURI string
		}{
			{"sess/slash", "/api/v1/sessions/sess%2Fslash/restore"},
			{"sess?query", "/api/v1/sessions/sess%3Fquery/restore"},
			{"sess#frag", "/api/v1/sessions/sess%23frag/restore"},
			{"sess%pct", "/api/v1/sessions/sess%25pct/restore"},
		}

		for _, tc := range reservedCases {
			t.Run(tc.sessionID, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.RequestURI != tc.expectedURI {
						t.Errorf("URI restructuring! got %q, want %q", r.RequestURI, tc.expectedURI)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(fmt.Sprintf(`{
						"ok": true,
						"sessionId": %q,
						"restoreMode": "native",
						"session": {"id": %q, "projectId": "p1", "activity": {"state": "active"}}
					}`, tc.sessionID, tc.sessionID)))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				res, err := c.ResumeWorker(context.Background(), tc.sessionID)
				if err != nil {
					t.Fatalf("ResumeWorker(%q) error: %v", tc.sessionID, err)
				}
				if res.SessionID != tc.sessionID {
					t.Errorf("expected sessionID %q, got %q", tc.sessionID, res.SessionID)
				}
			})
		}
	})

	t.Run("validation_failures", func(t *testing.T) {
		cases := []struct {
			name        string
			sessionID   string
			respStatus  int
			respBody    string
			expectedErr error
		}{
			{"empty_sessionID", "", http.StatusOK, "", ErrBadRequest},
			{"whitespace_sessionID", "   ", http.StatusOK, "", ErrBadRequest},
			{"wrong_status_201", "s1", http.StatusCreated, `{"ok":true,"sessionId":"s1","restoreMode":"native","session":{"id":"s1","activity":{"state":"active"}}}`, ErrProtocolViolation},
			{"ok_false", "s1", http.StatusOK, `{"ok":false,"sessionId":"s1","restoreMode":"native","session":{"id":"s1","activity":{"state":"active"}}}`, ErrProtocolViolation},
			{"empty_top_level_sessionId", "s1", http.StatusOK, `{"ok":true,"sessionId":"","restoreMode":"native","session":{"id":"s1","activity":{"state":"active"}}}`, ErrProtocolViolation},
			{"mismatched_top_level_sessionId", "s1", http.StatusOK, `{"ok":true,"sessionId":"other","restoreMode":"native","session":{"id":"s1","activity":{"state":"active"}}}`, ErrProtocolViolation},
			{"unknown_restore_mode", "s1", http.StatusOK, `{"ok":true,"sessionId":"s1","restoreMode":"invalid_mode","session":{"id":"s1","activity":{"state":"active"}}}`, ErrProtocolViolation},
			{"null_nested_session", "s1", http.StatusOK, `{"ok":true,"sessionId":"s1","restoreMode":"native","session":null}`, ErrProtocolViolation},
			{"empty_nested_session_id", "s1", http.StatusOK, `{"ok":true,"sessionId":"s1","restoreMode":"native","session":{"id":"","activity":{"state":"active"}}}`, ErrProtocolViolation},
			{"mismatched_nested_session_id", "s1", http.StatusOK, `{"ok":true,"sessionId":"s1","restoreMode":"native","session":{"id":"different","activity":{"state":"active"}}}`, ErrProtocolViolation},
			{"unknown_activity_state", "s1", http.StatusOK, `{"ok":true,"sessionId":"s1","restoreMode":"native","session":{"id":"s1","activity":{"state":"hibernating"}}}`, ErrProtocolViolation},
			{"ao_404_session_not_found", "s1", http.StatusNotFound, `{"error":"not_found","code":"SESSION_NOT_FOUND","message":"session not found"}`, ErrSessionNotFound},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.respStatus)
					w.Write([]byte(tc.respBody))
				}))
				defer s.Close()

				c, _ := newTestClient(t, s)
				_, err := c.ResumeWorker(context.Background(), tc.sessionID)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("expected %v, got %v", tc.expectedErr, err)
				}
			})
		}

		t.Run("restore_unknown_activity_reports_status_200", func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"ok":true,"sessionId":"s1","restoreMode":"native","session":{"id":"s1","activity":{"state":"hibernating"}}}`))
			}))
			defer s.Close()

			c, _ := newTestClient(t, s)
			_, err := c.ResumeWorker(context.Background(), "s1")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, ErrProtocolViolation) {
				t.Fatalf("expected errors.Is(err, ErrProtocolViolation), got %v", err)
			}
			var protoErr *ProtocolError
			if !errors.As(err, &protoErr) {
				t.Fatalf("expected *ProtocolError, got %T (%v)", err, err)
			}
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
	})
}
