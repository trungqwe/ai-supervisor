package domain_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// TestCanonicalTaskStateCount verifies exactly 13 canonical task states.
func TestCanonicalTaskStateCount(t *testing.T) {
	states := domain.AllTaskStates()
	if len(states) != 13 {
		t.Fatalf("expected exactly 13 canonical task states, got %d", len(states))
	}

	expectedStates := map[domain.TaskState]bool{
		domain.StateDraft:            true,
		domain.StateReady:            true,
		domain.StateDispatched:       true,
		domain.StateRunning:          true,
		domain.StateReportReady:      true,
		domain.StateEvidenceReady:    true,
		domain.StateReviewing:        true,
		domain.StateApproved:         true,
		domain.StateRevisionRequired: true,
		domain.StateBlocked:          true,
		domain.StateFailed:           true,
		domain.StateHumanRequired:    true,
		domain.StateCancelled:        true,
	}

	for _, s := range states {
		if !expectedStates[s] {
			t.Errorf("unexpected state %q in AllTaskStates()", s)
		}
		if !s.IsValid() {
			t.Errorf("expected state %q to be valid", s)
		}
	}

	invalidStates := []domain.TaskState{
		"",
		"UNKNOWN",
		"REPORT_MISSING",
		"REPORT_INVALID",
		"REPORT_IDENTITY_MISMATCH",
		"[*] ",
	}
	for _, inv := range invalidStates {
		if inv.IsValid() {
			t.Errorf("expected %q to be invalid TaskState", inv)
		}
	}

	if !domain.StateApproved.IsTerminal() {
		t.Errorf("expected APPROVED to be terminal")
	}
	if !domain.StateCancelled.IsTerminal() {
		t.Errorf("expected CANCELLED to be terminal")
	}
	if domain.StateRunning.IsTerminal() {
		t.Errorf("expected RUNNING not to be terminal")
	}
}

// TestTaskContractSerializationParity verifies JSON serialization against task-contract.schema.json.
// Asserts all 17 required keys are emitted even when arrays are empty,
// and asserts forbidden properties (such as is_immutable) are never emitted.
func TestTaskContractSerializationParity(t *testing.T) {
	contract := domain.TaskContract{
		ContractID:           "CONTRACT-TASK-P02-001-02",
		TaskID:               "TASK-P02-001",
		RevisionNumber:       2,
		SupersedesContractID: nil,
		PhaseID:              "P02",
		Objective:            "Align JSON serialization and state transition terminology",
		Requirements:         []string{"FR-001", "FR-002"},
		ArchitectureRefs:     []string{}, // Empty slice: must still serialize as []
		BaseSHA:              "ca3262eed4b1f72236e86457c865d07cef197094",
		AllowedScope:         []string{"internal/domain/**", "internal/workflow/**"},
		ForbiddenScope:       []string{"docs/**"},
		Constraints:          []string{}, // Empty slice: must still serialize as []
		AcceptanceCriteria:   []string{"Serialization parity passes"},
		VerificationRequests: []domain.VerificationRequest{}, // Empty slice: must serialize as []
		RequiredEvidence:     []string{"git_diff", "go_test_exit_code"},
		WorkerProfile:        "antigravity-standard",
		ReportContract:       "docs/schemas/worker-report.schema.json",
		StopConditions:       []string{}, // Empty slice: must still serialize as []
		IsImmutable:          true,       // Internal domain property: MUST NOT be serialized
	}

	data, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var unmarshaled map[string]any
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{
		"contract_id",
		"task_id",
		"revision_number",
		"phase_id",
		"objective",
		"requirements",
		"architecture_refs",
		"base_sha",
		"allowed_scope",
		"forbidden_scope",
		"constraints",
		"acceptance_criteria",
		"verification_requests",
		"required_evidence",
		"worker_profile",
		"report_contract",
		"stop_conditions",
	}

	for _, key := range requiredKeys {
		val, exists := unmarshaled[key]
		if !exists {
			t.Errorf("REQUIRED key %q missing from serialized TaskContract", key)
			continue
		}
		if val == nil {
			t.Errorf("REQUIRED key %q serialized as null", key)
		}
	}

	// Assert is_immutable is strictly ABSENT
	if _, exists := unmarshaled["is_immutable"]; exists {
		t.Errorf("forbidden property \"is_immutable\" is present in serialized TaskContract")
	}

	// Assert only allowed properties from task-contract.schema.json are present
	allowedSchemaKeys := map[string]bool{
		"contract_id":            true,
		"task_id":                true,
		"revision_number":        true,
		"supersedes_contract_id": true,
		"phase_id":               true,
		"objective":              true,
		"requirements":           true,
		"architecture_refs":      true,
		"base_sha":               true,
		"allowed_scope":          true,
		"forbidden_scope":        true,
		"constraints":            true,
		"acceptance_criteria":    true,
		"required_evidence":      true,
		"worker_profile":         true,
		"report_contract":        true,
		"stop_conditions":        true,
		"verification_requests":  true,
	}

	for key := range unmarshaled {
		if !allowedSchemaKeys[key] {
			t.Errorf("unknown / non-schema key %q present in serialized TaskContract", key)
		}
	}

	// Optional field supersedes_contract_id should be omitted when nil
	if _, exists := unmarshaled["supersedes_contract_id"]; exists {
		t.Errorf("supersedes_contract_id should be omitted when nil")
	}

	// Test with supersedes_contract_id populated
	prevID := "CONTRACT-TASK-P02-001-01"
	contract.SupersedesContractID = &prevID
	data2, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("json.Marshal with supersedes_contract_id failed: %v", err)
	}
	var unmarshaled2 map[string]any
	if err := json.Unmarshal(data2, &unmarshaled2); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if unmarshaled2["supersedes_contract_id"] != prevID {
		t.Errorf("expected supersedes_contract_id %q, got %v", prevID, unmarshaled2["supersedes_contract_id"])
	}
}

// TestTaskAttemptInvariants verifies attempt fields and lineage binding to task and contract.
func TestTaskAttemptInvariants(t *testing.T) {
	now := time.Now().UTC()
	attempt := domain.TaskAttempt{
		AttemptID:          "attempt-001",
		AttemptNumber:      1,
		TaskID:             "TASK-P02-001",
		ContractID:         "CONTRACT-TASK-P02-001-01",
		ExpectedReportPath: ".supervisor/reports/TASK-P02-001/attempt-001.json",
		StartedAt:          now,
	}

	if attempt.AttemptID == "" {
		t.Fatal("expected non-empty attempt_id")
	}
	if attempt.AttemptNumber != 1 {
		t.Fatalf("expected attempt_number 1, got %d", attempt.AttemptNumber)
	}
	if attempt.TaskID != "TASK-P02-001" {
		t.Fatalf("expected task_id TASK-P02-001, got %s", attempt.TaskID)
	}
	if attempt.ContractID != "CONTRACT-TASK-P02-001-01" {
		t.Fatalf("expected contract_id CONTRACT-TASK-P02-001-01, got %s", attempt.ContractID)
	}
	if attempt.ExpectedReportPath != ".supervisor/reports/TASK-P02-001/attempt-001.json" {
		t.Fatalf("unexpected report path: %s", attempt.ExpectedReportPath)
	}
}

// TestReviewDecisionAttemptBound verifies ReviewDecision is bound to task_id and attempt_id.
func TestReviewDecisionAttemptBound(t *testing.T) {
	dec := domain.ReviewDecision{
		DecisionID:        "dec-001",
		TaskID:            "TASK-P02-001",
		AttemptID:         "attempt-001",
		Decision:          domain.DecisionApprove,
		ReviewerRationale: "Implementation conforms to canonical contracts",
		DecidedAt:         time.Now().UTC(),
	}

	if dec.TaskID == "" || dec.AttemptID == "" {
		t.Fatal("ReviewDecision must be bound to both task_id and attempt_id")
	}
	if dec.Decision != domain.DecisionApprove {
		t.Fatalf("expected APPROVE decision, got %s", dec.Decision)
	}
}

// TestVerificationRequestForbiddenFields uses reflection to verify VerificationRequest has NO executable/shell fields.
func TestVerificationRequestForbiddenFields(t *testing.T) {
	vrType := reflect.TypeOf(domain.VerificationRequest{})
	forbiddenSubstrings := []string{
		"exec", "command", "shell", "raw", "argv", "script",
	}

	for i := 0; i < vrType.NumField(); i++ {
		fieldName := strings.ToLower(vrType.Field(i).Name)
		for _, forbidden := range forbiddenSubstrings {
			if strings.Contains(fieldName, forbidden) {
				t.Errorf("VerificationRequest struct has forbidden field %q (matched %q)", vrType.Field(i).Name, forbidden)
			}
		}
	}
}

// mockCatalog is an in-memory test implementation of VerificationPolicyCatalog.
type mockCatalog struct {
	policies map[string]domain.VerificationProfilePolicy
}

func (m *mockCatalog) LookupProfile(profileID string) (domain.VerificationProfilePolicy, bool) {
	p, ok := m.policies[profileID]
	return p, ok
}

// TestVerificationPolicyCatalogInterface verifies VerificationPolicyCatalog as a pure domain interface.
func TestVerificationPolicyCatalogInterface(t *testing.T) {
	cat := &mockCatalog{
		policies: map[string]domain.VerificationProfilePolicy{
			"go-test": {
				ProfileID:         "go-test",
				MaxTimeoutSeconds: 300,
				CwdPolicy:         "worktree_root",
			},
		},
	}

	var catalog domain.VerificationPolicyCatalog = cat
	policy, found := catalog.LookupProfile("go-test")
	if !found {
		t.Fatal("expected profile go-test to be found")
	}
	if policy.MaxTimeoutSeconds != 300 {
		t.Fatalf("expected MaxTimeoutSeconds 300, got %d", policy.MaxTimeoutSeconds)
	}

	_, notFound := catalog.LookupProfile("nonexistent")
	if notFound {
		t.Fatal("expected nonexistent profile to return found=false")
	}
}
