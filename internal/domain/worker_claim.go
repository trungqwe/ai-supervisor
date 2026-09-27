package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	shaHexRegex = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

// WorkerClaimRecord represents an immutable worker claim persisted in SQLite.
type WorkerClaimRecord struct {
	ClaimID          string `json:"claim_id"`
	TaskID           string `json:"task_id"`
	AttemptID        string `json:"attempt_id"`
	ContractID       string `json:"contract_id"`
	ReportedHeadSHA  string `json:"reported_head_sha"`
	PayloadJSON      string `json:"payload_json"`
	CreatedAtEpochMS int64  `json:"created_at_epoch_ms"`
}

// Validate validates worker claim record invariants.
func (c *WorkerClaimRecord) Validate() error {
	if strings.TrimSpace(c.ClaimID) == "" {
		return errors.New("claim_id must not be empty")
	}
	if strings.TrimSpace(c.TaskID) == "" {
		return errors.New("task_id must not be empty")
	}
	if strings.TrimSpace(c.AttemptID) == "" {
		return errors.New("attempt_id must not be empty")
	}
	if strings.TrimSpace(c.ContractID) == "" {
		return errors.New("contract_id must not be empty")
	}
	if !shaHexRegex.MatchString(c.ReportedHeadSHA) {
		return fmt.Errorf("reported_head_sha must be 7..40 lowercase hex chars, got %q", c.ReportedHeadSHA)
	}
	if c.CreatedAtEpochMS <= 0 {
		return errors.New("created_at_epoch_ms must be positive")
	}

	// Validate payload_json structure
	var payload map[string]any
	if err := json.Unmarshal([]byte(c.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("payload_json is not valid JSON: %w", err)
	}

	for _, requiredArr := range []string{"claimed_files_changed", "tests", "textual_claims"} {
		v, ok := payload[requiredArr]
		if !ok {
			return fmt.Errorf("payload_json missing required field %q", requiredArr)
		}
		if _, isSlice := v.([]any); !isSlice {
			return fmt.Errorf("payload_json field %q must be an array", requiredArr)
		}
	}

	return nil
}

// WorkerReport represents the structured report submitted by a worker.
type WorkerReport struct {
	HeadSHA      string              `json:"head_sha"`
	FilesChanged []string            `json:"files_changed"`
	Tests        []ClaimedTestResult `json:"tests,omitempty"`
	WorkerClaims []string            `json:"worker_claims,omitempty"`
	BuildStatus  string              `json:"build_status,omitempty"`
}

// WorkerClaimPayload encapsulates structured worker claim payload fields.
type WorkerClaimPayload struct {
	ClaimedFilesChanged []string            `json:"claimed_files_changed"`
	Tests               []ClaimedTestResult `json:"tests"`
	TextualClaims       []string            `json:"textual_claims"`
	BuildStatus         string              `json:"build_status,omitempty"`
}

// CanonicalizeWorkerClaimPayload converts WorkerClaimPayload to RFC 8785 JCS canonical JSON.
func CanonicalizeWorkerClaimPayload(p WorkerClaimPayload) (string, error) {
	if p.ClaimedFilesChanged == nil {
		p.ClaimedFilesChanged = []string{}
	}
	if p.Tests == nil {
		p.Tests = []ClaimedTestResult{}
	}
	if p.TextualClaims == nil {
		p.TextualClaims = []string{}
	}

	m := map[string]any{
		"claimed_files_changed": p.ClaimedFilesChanged,
		"tests":                 p.Tests,
		"textual_claims":        p.TextualClaims,
	}
	if strings.TrimSpace(p.BuildStatus) != "" {
		m["build_status"] = p.BuildStatus
	}

	// json.Marshal on map sorts keys lexicographically per RFC 8785 rules.
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return "", fmt.Errorf("failed to canonicalize payload: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// GitEvidenceResult captures independent Git status and commit facts.
// Strictly conveys independent evidence collected by Subtask P04B without ReportedHeadSHA field.
type GitEvidenceResult struct {
	IsClean          bool     `json:"is_clean"`
	UncommittedFiles []string `json:"uncommitted_files,omitempty"`
	ActualHeadSHA    string   `json:"actual_head_sha,omitempty"`
	ActualBaseSHA    string   `json:"actual_base_sha,omitempty"`
	ChangedFiles     []string `json:"changed_files,omitempty"`
	DiffStat         string   `json:"diff_stat,omitempty"`
	DiagnosticError  string   `json:"diagnostic_error,omitempty"`
}

// GitEvidenceCollector provides independent Git status and inspection.
type GitEvidenceCollector interface {
	Collect(ctx context.Context, worktreePath string) (GitEvidenceResult, error)
}
