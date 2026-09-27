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

func TestWorkerReport_Validate(t *testing.T) {
	validReport := domain.WorkerReport{
		TaskID:         "task-001",
		AttemptID:      "attempt-001",
		Status:         "COMPLETED",
		Branch:         "codex/test",
		BaseSHA:        "1234567890abcdef",
		HeadSHA:        "abcdef1234567890",
		FilesChanged:   []string{"file1.go"},
		CommandsRun:    []domain.CommandRun{{Command: "go test ./...", ExitCode: 0}},
		Tests:          []domain.WorkerReportTest{{TestSuite: "unit", Passed: 5, Failed: 0}},
		BuildStatus:    "PASSED",
		WorkerClaims:   []string{"claim 1"},
		ReadyForReview: true,
	}

	if err := validReport.Validate(); err != nil {
		t.Fatalf("expected valid report, got error: %v", err)
	}

	// 1. Missing task_id
	badTask := validReport
	badTask.TaskID = ""
	if err := badTask.Validate(); err == nil {
		t.Fatal("expected error on empty task_id")
	}

	// 2. Invalid status enum
	badStatus := validReport
	badStatus.Status = "IN_PROGRESS"
	if err := badStatus.Validate(); err == nil {
		t.Fatal("expected error on invalid status enum")
	}

	// 3. Invalid head_sha
	badHex := validReport
	badHex.HeadSHA = "NOT_HEX_CHARS!"
	if err := badHex.Validate(); err == nil {
		t.Fatal("expected error on invalid head_sha pattern")
	}

	// 4. Invalid base_sha
	badBase := validReport
	badBase.BaseSHA = "short"
	if err := badBase.Validate(); err == nil {
		t.Fatal("expected error on invalid base_sha pattern")
	}

	// 5. Invalid build_status
	badBuild := validReport
	badBuild.BuildStatus = "RUNNING"
	if err := badBuild.Validate(); err == nil {
		t.Fatal("expected error on invalid build_status")
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
