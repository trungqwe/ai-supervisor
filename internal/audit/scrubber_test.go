package audit

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestScrubString_Patterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		notMatch []string
	}{
		{
			name:     "bearer token in authorization header",
			input:    "Authorization: Bearer my-secret-token-1234567890",
			contains: []string{"Authorization: Bearer [REDACTED]"},
			notMatch: []string{"my-secret-token"},
		},
		{
			name:     "bearer case insensitivity",
			input:    "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.token",
			contains: []string{"bearer [REDACTED]"},
			notMatch: []string{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		},
		{
			name:     "github oauth token",
			input:    "Found token gho_1234567890abcdef1234 in git log",
			contains: []string{"Found token [REDACTED] in git log"},
			notMatch: []string{"gho_"},
		},
		{
			name:     "github pat classic",
			input:    "Access via ghp_abcdefghijklmnopqrstuvwxyz123456",
			contains: []string{"Access via [REDACTED]"},
			notMatch: []string{"ghp_"},
		},
		{
			name:     "github fine-grained pat",
			input:    "PAT github_pat_11ABCDEF_1234567890abcdefghijklmnopqrstuvwxyz",
			contains: []string{"PAT [REDACTED]"},
			notMatch: []string{"github_pat_"},
		},
		{
			name:     "openai standard key",
			input:    "sk-1234567890abcdefghijklmnopqrstuvwxyz",
			contains: []string{"[REDACTED]"},
			notMatch: []string{"sk-1234567890"},
		},
		{
			name:     "openai project key",
			input:    "Model key sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
			contains: []string{"Model key [REDACTED]"},
			notMatch: []string{"sk-proj-"},
		},
		{
			name:     "aws access key id",
			input:    "AWS credentials: AKIAIOSFODNN7EXAMPLE",
			contains: []string{"AWS credentials: [REDACTED]"},
			notMatch: []string{"AKIAIOSFODNN7EXAMPLE"},
		},
		{
			name:     "env assignment OPENAI_API_KEY",
			input:    "OPENAI_API_KEY=sk-test-1234567890123456",
			contains: []string{"OPENAI_API_KEY=[REDACTED]"},
			notMatch: []string{"sk-test-"},
		},
		{
			name:     "env assignment API_KEY and PASSWORD",
			input:    "Connecting with API_KEY=secret-api-val and PASSWORD=hunter2",
			contains: []string{"API_KEY=[REDACTED]", "PASSWORD=[REDACTED]"},
			notMatch: []string{"secret-api-val", "hunter2"},
		},
		{
			name:     "env assignment CLIENT_SECRET and TOKEN",
			input:    "Set CLIENT_SECRET=topsecret and TOKEN=tok-123456",
			contains: []string{"CLIENT_SECRET=[REDACTED]", "TOKEN=[REDACTED]"},
			notMatch: []string{"topsecret", "tok-123456"},
		},
		{
			name:     "normal prose without secrets preserved",
			input:    "Task completed successfully with 5 iterations and 0 errors.",
			contains: []string{"Task completed successfully with 5 iterations and 0 errors."},
		},
		{
			name:     "normal word token preserved",
			input:    "This is a token count of 5",
			contains: []string{"This is a token count of 5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScrubString(tt.input)
			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					t.Errorf("expected output to contain %q, got %q", c, got)
				}
			}
			for _, nm := range tt.notMatch {
				if strings.Contains(got, nm) {
					t.Errorf("expected output to NOT contain raw secret %q, got %q", nm, got)
				}
			}
		})
	}
	t.Logf("SECRET_EMBEDDED_PATTERN_SCRUB = PASS")
}

func TestScrubValue_StructuredKeys(t *testing.T) {
	input := map[string]any{
		"password":            "plaintext_pwd",
		"passwd":              "shadow_passwd",
		"secret":              "deep_secret",
		"client_secret":       "client_sec_val",
		"client-secret":       "client_hyphen_sec",
		"api_key":             "api_key_val",
		"apikey":              "apikey_val",
		"api-key":             "api_hyphen_val",
		"token":               "bearer_raw",
		"access_token":        "access_tok_val",
		"refresh_token":       "refresh_tok_val",
		"id_token":            "id_tok_val",
		"session_token":       "sess_tok_val",
		"authorization":       "Basic dXNlcjpwYXNz",
		"proxy_authorization": "ProxyAuth abc",
		"cookie":              "session_id=12345",
		"set_cookie":          "auth=deleted",
		"private_key":         "-----BEGIN PRIVATE KEY-----",
		"public_field":        "safe_value",
		"count":               42,
		"active":              true,
		"ratio":               3.14,
	}

	scrubbed, err := ScrubValue(input)
	if err != nil {
		t.Fatalf("ScrubValue failed: %v", err)
	}

	resMap, ok := scrubbed.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", scrubbed)
	}

	sensitiveCheckKeys := []string{
		"password", "passwd", "secret", "client_secret", "client-secret",
		"api_key", "apikey", "api-key", "token", "access_token",
		"refresh_token", "id_token", "session_token", "authorization",
		"proxy_authorization", "cookie", "set_cookie", "private_key",
	}

	for _, k := range sensitiveCheckKeys {
		val, exists := resMap[k]
		if !exists {
			t.Errorf("expected key %q to exist in scrubbed map", k)
			continue
		}
		if val != RedactedMarker {
			t.Errorf("expected key %q to be %q, got %v", k, RedactedMarker, val)
		}
	}

	// Safe fields preserved
	if resMap["public_field"] != "safe_value" {
		t.Errorf("expected public_field to be preserved, got %v", resMap["public_field"])
	}
	if resMap["count"] != 42 {
		t.Errorf("expected count to be preserved, got %v", resMap["count"])
	}
	if resMap["active"] != true {
		t.Errorf("expected active to be preserved, got %v", resMap["active"])
	}
	if resMap["ratio"] != 3.14 {
		t.Errorf("expected ratio to be preserved, got %v", resMap["ratio"])
	}

	// Ensure input map was NOT mutated
	if input["password"] != "plaintext_pwd" {
		t.Errorf("input map was mutated: password = %v", input["password"])
	}
	if input["session_token"] != "sess_tok_val" {
		t.Errorf("input map was mutated: session_token = %v", input["session_token"])
	}

	t.Logf("SECRET_STRUCTURED_KEY_SCRUB = PASS")
}

func TestScrubValue_SecretInKeys(t *testing.T) {
	rawGHKey := "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
	rawOpenAIKey := "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz"
	rawBearerKey := "Authorization: Bearer super-secret-key-token-12345"
	rawNestedKey := "github_pat_11ABCDEF_1234567890abcdefghijklmnopqrstuvwxyz"

	input := map[string]any{
		rawGHKey:     "val_1",
		rawBearerKey: "val_2",
		"nested": map[string]any{
			rawNestedKey: "nested_val",
		},
		"session_token": "secret_sess",
	}

	scrubbed, err := ScrubValue(input)
	if err != nil {
		t.Fatalf("ScrubValue failed: %v", err)
	}

	resMap, ok := scrubbed.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", scrubbed)
	}

	// Raw GH key must not exist in resMap
	if _, exists := resMap[rawGHKey]; exists {
		t.Errorf("raw GitHub token key was NOT scrubbed: %s", rawGHKey)
	}
	// Instead, scrubbed key [REDACTED] must exist with value "val_1"
	if resMap[RedactedMarker] != "val_1" {
		t.Errorf("expected key %s with val_1, got: %v", RedactedMarker, resMap[RedactedMarker])
	}

	// Raw Bearer key must not exist
	if _, exists := resMap[rawBearerKey]; exists {
		t.Errorf("raw Bearer key was NOT scrubbed: %s", rawBearerKey)
	}
	scrubbedBearerKey := "Authorization: Bearer [REDACTED]"
	if resMap[scrubbedBearerKey] != "val_2" {
		t.Errorf("expected scrubbed bearer key %s with val_2, got: %v", scrubbedBearerKey, resMap[scrubbedBearerKey])
	}

	// Structured key session_token retains key name and redacts value
	if resMap["session_token"] != RedactedMarker {
		t.Errorf("expected session_token to be %s, got: %v", RedactedMarker, resMap["session_token"])
	}

	// Check nested map
	nestedMap, ok := resMap["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested is not map[string]any: %T", resMap["nested"])
	}
	if _, exists := nestedMap[rawNestedKey]; exists {
		t.Errorf("raw nested key was NOT scrubbed: %s", rawNestedKey)
	}
	if nestedMap[RedactedMarker] != "nested_val" {
		t.Errorf("expected nested %s with nested_val, got: %v", RedactedMarker, nestedMap[RedactedMarker])
	}

	// Verify caller input was NOT mutated
	if _, exists := input[rawGHKey]; !exists {
		t.Errorf("input map was mutated: key %s missing", rawGHKey)
	}

	// Also test OpenAI key in a separate map to avoid key collision with ghp_
	inputOpenAI := map[string]any{
		rawOpenAIKey: "val_openai",
	}
	scrubbedOpenAI, err := ScrubValue(inputOpenAI)
	if err != nil {
		t.Fatalf("ScrubValue failed on OpenAI key: %v", err)
	}
	resOpenAIMap := scrubbedOpenAI.(map[string]any)
	if _, exists := resOpenAIMap[rawOpenAIKey]; exists {
		t.Errorf("raw OpenAI key was NOT scrubbed: %s", rawOpenAIKey)
	}
	if resOpenAIMap[RedactedMarker] != "val_openai" {
		t.Errorf("expected %s with val_openai, got: %v", RedactedMarker, resOpenAIMap[RedactedMarker])
	}

	t.Logf("SECRET_IN_JSON_KEY = REDACTED")
	t.Logf("NESTED_SECRET_IN_JSON_KEY = REDACTED")
}

func TestScrubValue_KeyCollision(t *testing.T) {
	// Two distinct keys that both scrub to [REDACTED]
	input := map[string]any{
		"ghp_1234567890abcdefghijklmnopqrstuvwxyz": "val_1",
		"ghp_9999999999abcdefghijklmnopqrstuvwxyz": "val_2",
	}

	_, err := ScrubValue(input)
	if err == nil {
		t.Fatalf("expected error on sanitized key collision, got nil")
	}
	if !errors.Is(err, ErrSanitizedKeyCollision) {
		t.Errorf("expected ErrSanitizedKeyCollision, got: %v", err)
	}

	t.Logf("SANITIZED_KEY_COLLISION = REJECTED")
}

func TestScrubValue_NonStringMapKey(t *testing.T) {
	// Non-string map key (e.g. map[int]string)
	input := map[int]string{
		1: "bad_key",
	}

	_, err := ScrubValue(input)
	if err == nil {
		t.Fatalf("expected error on non-string map key, got nil")
	}
	if !errors.Is(err, ErrNonStringMapKey) {
		t.Errorf("expected ErrNonStringMapKey, got: %v", err)
	}

	t.Logf("NON_STRING_MAP_KEY = REJECTED")
}

func TestScrubValue_NestedAndBinary(t *testing.T) {
	input := map[string]any{
		"nested_map": map[string]any{
			"api_key":   "nested-secret-key",
			"log_entry": "Failed auth: Authorization: Bearer nested-bearer-123456",
			"sub_nested": map[string]any{
				"password": "inner-password",
			},
		},
		"nested_slice": []any{
			map[string]any{
				"token": "slice-token",
			},
			"OPENAI_API_KEY=sk-proj-nested1234567890abcdef",
			100,
		},
		"binary_data": []byte("raw secret payload"),
	}

	scrubbed, err := ScrubValue(input)
	if err != nil {
		t.Fatalf("ScrubValue failed: %v", err)
	}

	resMap := scrubbed.(map[string]any)
	nestedMap := resMap["nested_map"].(map[string]any)
	if nestedMap["api_key"] != RedactedMarker {
		t.Errorf("nested api_key not redacted: %v", nestedMap["api_key"])
	}
	if strings.Contains(nestedMap["log_entry"].(string), "nested-bearer") {
		t.Errorf("nested log_entry contains raw bearer: %v", nestedMap["log_entry"])
	}
	subNested := nestedMap["sub_nested"].(map[string]any)
	if subNested["password"] != RedactedMarker {
		t.Errorf("sub_nested password not redacted: %v", subNested["password"])
	}

	nestedSlice := resMap["nested_slice"].([]any)
	sliceMap := nestedSlice[0].(map[string]any)
	if sliceMap["token"] != RedactedMarker {
		t.Errorf("slice map token not redacted: %v", sliceMap["token"])
	}
	if strings.Contains(nestedSlice[1].(string), "sk-proj-") {
		t.Errorf("slice string contains raw openai key: %v", nestedSlice[1])
	}
	if nestedSlice[2] != 100 {
		t.Errorf("slice number corrupted: %v", nestedSlice[2])
	}

	// Binary data must be RedactedBinaryMarker
	if resMap["binary_data"] != RedactedBinaryMarker {
		t.Errorf("binary data not converted to RedactedBinaryMarker: got %v", resMap["binary_data"])
	}
}

func TestSanitizeAuditEvent_SessionToken(t *testing.T) {
	// Canonical Section 36 requirement: pair.bound session_token scrubbing
	rawEvent := domain.AuditEvent{
		EventID:   "evt-bound-01",
		EventType: "pair.bound",
		Timestamp: time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC),
		PairID:    "pair-alpha",
		Actor:     "worker-actor Authorization: Bearer actor-token-12345",
		Details: map[string]any{
			"session_token": "super-secret-session-token",
			"project_id":    "proj-001",
		},
	}

	sanitized, detailsJSON, err := SanitizeAuditEvent(rawEvent)
	if err != nil {
		t.Fatalf("SanitizeAuditEvent failed: %v", err)
	}

	// Actor must be scrubbed
	if strings.Contains(sanitized.Actor, "actor-token") {
		t.Errorf("actor contains raw token: %s", sanitized.Actor)
	}
	if !strings.Contains(sanitized.Actor, RedactedMarker) {
		t.Errorf("actor does not contain redacted marker: %s", sanitized.Actor)
	}

	// Details map session_token must be redacted
	if sanitized.Details["session_token"] != RedactedMarker {
		t.Errorf("sanitized.Details[session_token] = %v, expected %s", sanitized.Details["session_token"], RedactedMarker)
	}

	// detailsJSON must contain "[REDACTED]" and NOT the raw secret
	if strings.Contains(detailsJSON, "super-secret-session-token") {
		t.Errorf("detailsJSON contains raw session_token: %s", detailsJSON)
	}
	if !strings.Contains(detailsJSON, RedactedMarker) {
		t.Errorf("detailsJSON does not contain redacted marker: %s", detailsJSON)
	}

	// Raw event must NOT be mutated
	if rawEvent.Details["session_token"] != "super-secret-session-token" {
		t.Errorf("original rawEvent was mutated: %v", rawEvent.Details["session_token"])
	}

	t.Logf("SESSION_TOKEN_SCRUB = PASS")
}

func TestSanitizeDetails_CanonicalEmpty(t *testing.T) {
	// Nil details
	_, jsonNil, err := SanitizeDetails(nil)
	if err != nil {
		t.Fatalf("SanitizeDetails(nil) failed: %v", err)
	}
	if jsonNil != "{}" {
		t.Errorf("expected \"{}\" for nil details, got %q", jsonNil)
	}

	// Empty details
	_, jsonEmpty, err := SanitizeDetails(map[string]any{})
	if err != nil {
		t.Fatalf("SanitizeDetails(empty) failed: %v", err)
	}
	if jsonEmpty != "{}" {
		t.Errorf("expected \"{}\" for empty details, got %q", jsonEmpty)
	}
}

func TestSanitizeDetails_PreservesJSONNumber(t *testing.T) {
	num := json.Number("12345678901234567890")
	input := map[string]any{
		"big_num": num,
	}

	_, detailsJSON, err := SanitizeDetails(input)
	if err != nil {
		t.Fatalf("SanitizeDetails failed: %v", err)
	}

	if !strings.Contains(detailsJSON, "12345678901234567890") {
		t.Errorf("json.Number precision lost: %s", detailsJSON)
	}
}
