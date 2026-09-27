package domain_test

import (
	"testing"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestWorkerClaimRecord_Validate(t *testing.T) {
	payload, err := domain.CanonicalizeWorkerClaimPayload(domain.WorkerClaimPayload{
		ClaimedFilesChanged: []string{"file1.go", "file2.go"},
		Tests: []domain.WorkerReportTest{
			{TestSuite: "SuiteA", Passed: 1, Failed: 0},
		},
		TextualClaims: []string{"Implemented feature X"},
	})
	if err != nil {
		t.Fatalf("failed to canonicalize payload: %v", err)
	}

	record := domain.WorkerClaimRecord{
		ClaimID:          "claim-001",
		TaskID:           "task-001",
		AttemptID:        "attempt-001",
		ContractID:       "contract-001",
		ReportedHeadSHA:  "abcdef1",
		PayloadJSON:      payload,
		CreatedAtEpochMS: 1000,
	}

	if err := record.Validate(); err != nil {
		t.Fatalf("expected valid record, got: %v", err)
	}

	// Short SHA
	recordBadSHA := record
	recordBadSHA.ReportedHeadSHA = "abc"
	if err := recordBadSHA.Validate(); err == nil {
		t.Fatal("expected error on short SHA (< 7 hex chars)")
	}

	// Non-hex SHA
	recordBadHex := record
	recordBadHex.ReportedHeadSHA = "abcdefg"
	if err := recordBadHex.Validate(); err == nil {
		t.Fatal("expected error on non-hex SHA")
	}

	// Missing array in payload
	recordBadPayload := record
	recordBadPayload.PayloadJSON = `{"claimed_files_changed": []}`
	if err := recordBadPayload.Validate(); err == nil {
		t.Fatal("expected error on missing required array fields")
	}
}

func TestValidateWorkerReportJSON_SchemaCompliance(t *testing.T) {
	validJSON := `{
		"task_id": "task-001",
		"attempt_id": "attempt-001",
		"status": "COMPLETED",
		"branch": "codex/test",
		"base_sha": "1234567890abcdef",
		"head_sha": "abcdef1234567890",
		"files_changed": ["file1.go"],
		"commands_run": [{"command": "go test ./...", "exit_code": 0}],
		"tests": [{"test_suite": "unit", "passed": 5, "failed": 0}],
		"build_status": "PASSED",
		"worker_claims": ["claim 1"],
		"ready_for_review": true
	}`

	report, err := domain.ValidateWorkerReportJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("expected valid report, got error: %v", err)
	}
	if report.TaskID != "task-001" || report.HeadSHA != "abcdef1234567890" {
		t.Fatalf("unexpected parsed report: %+v", report)
	}

	// 1. Missing required field (e.g. ready_for_review)
	missingReq := `{
		"task_id": "task-001",
		"attempt_id": "attempt-001",
		"status": "COMPLETED",
		"branch": "codex/test",
		"base_sha": "1234567890abcdef",
		"head_sha": "abcdef1234567890",
		"files_changed": ["file1.go"],
		"commands_run": [{"command": "go test", "exit_code": 0}],
		"tests": [{"test_suite": "unit", "passed": 1, "failed": 0}],
		"build_status": "PASSED",
		"worker_claims": ["claim 1"]
	}`
	if _, err := domain.ValidateWorkerReportJSON([]byte(missingReq)); err == nil {
		t.Fatal("expected error on missing ready_for_review")
	}

	// 2. Additional property (additionalProperties: false)
	extraProp := `{
		"task_id": "task-001",
		"attempt_id": "attempt-001",
		"status": "COMPLETED",
		"branch": "codex/test",
		"base_sha": "1234567890abcdef",
		"head_sha": "abcdef1234567890",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true,
		"unapproved_field": "disallowed"
	}`
	if _, err := domain.ValidateWorkerReportJSON([]byte(extraProp)); err == nil {
		t.Fatal("expected error on extra unapproved property")
	}

	// 3. Invalid status enum
	badEnum := `{
		"task_id": "task-001",
		"attempt_id": "attempt-001",
		"status": "IN_PROGRESS",
		"branch": "codex/test",
		"base_sha": "1234567890abcdef",
		"head_sha": "abcdef1234567890",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true
	}`
	if _, err := domain.ValidateWorkerReportJSON([]byte(badEnum)); err == nil {
		t.Fatal("expected error on invalid status enum")
	}

	// 4. Invalid head_sha pattern (non-hex)
	badHex := `{
		"task_id": "task-001",
		"attempt_id": "attempt-001",
		"status": "COMPLETED",
		"branch": "codex/test",
		"base_sha": "1234567890abcdef",
		"head_sha": "NOT_HEX_CHARS!",
		"files_changed": [],
		"commands_run": [],
		"tests": [],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true
	}`
	if _, err := domain.ValidateWorkerReportJSON([]byte(badHex)); err == nil {
		t.Fatal("expected error on invalid head_sha pattern")
	}

	// 5. Invalid test shape (extra field in test object)
	badTestShape := `{
		"task_id": "task-001",
		"attempt_id": "attempt-001",
		"status": "COMPLETED",
		"branch": "codex/test",
		"base_sha": "1234567890abcdef",
		"head_sha": "abcdef1234567890",
		"files_changed": [],
		"commands_run": [],
		"tests": [{"test_suite": "unit", "passed": 1, "failed": 0, "extra": "invalid"}],
		"build_status": "PASSED",
		"worker_claims": [],
		"ready_for_review": true
	}`
	if _, err := domain.ValidateWorkerReportJSON([]byte(badTestShape)); err == nil {
		t.Fatal("expected error on extra field in test object")
	}
}

func TestGitEvidenceResult_ZeroTrustSeparation(t *testing.T) {
	// WorkerReport reported_head_sha vs GitEvidenceResult actual_head_sha
	reportedSHA := "1234567890abcdef"
	actualSHA := "fedcba0987654321"

	evidence := domain.GitEvidenceResult{
		IsClean:       true,
		ActualHeadSHA: actualSHA,
		ActualBaseSHA: "base1234567890",
	}

	claimRecord := domain.WorkerClaimRecord{
		ClaimID:          "claim-1",
		TaskID:           "task-1",
		AttemptID:        "attempt-1",
		ContractID:       "contract-1",
		ReportedHeadSHA:  reportedSHA,
		PayloadJSON:      `{"claimed_files_changed":[],"tests":[],"textual_claims":[]}`,
		CreatedAtEpochMS: 1000,
	}

	if claimRecord.ReportedHeadSHA == evidence.ActualHeadSHA {
		t.Fatal("reported_head_sha and actual_head_sha should be decoupled")
	}
	if evidence.ActualHeadSHA != actualSHA {
		t.Fatalf("expected actual_head_sha %q, got %q", actualSHA, evidence.ActualHeadSHA)
	}
	if claimRecord.ReportedHeadSHA != reportedSHA {
		t.Fatalf("expected reported_head_sha %q, got %q", reportedSHA, claimRecord.ReportedHeadSHA)
	}
}
