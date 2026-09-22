package domain

import (
	"time"
)

// Project represents a registered codebase workspace.
type Project struct {
	ProjectID    string    `json:"project_id"`
	Name         string    `json:"name"`
	RootPath     string    `json:"root_path"`
	RepoURL      string    `json:"repo_url,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
}

// Pair represents the collaboration lane between a Supervisor and a Worker.
type Pair struct {
	PairID         string    `json:"pair_id"`
	ProjectID      string    `json:"project_id"`
	CurrentPhaseID string    `json:"current_phase_id"`
	ActiveTaskID   string    `json:"active_task_id,omitempty"`
	State          string    `json:"state"`
	CreatedAt      time.Time `json:"created_at"`
}

// Task manages overall task lifecycle and attempt lineage across revision cycles.
type Task struct {
	TaskID         string    `json:"task_id"`
	PhaseID        string    `json:"phase_id"`
	PairID         string    `json:"pair_id"`
	State          TaskState `json:"state"`
	CurrentAttempt int       `json:"current_attempt"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// VerificationRequest defines a constrained test or verification profile to run during evidence collection.
// Per ADR-013, it contains discrete parameters without shell strings, command lines, or raw executable paths.
type VerificationRequest struct {
	ID             string         `json:"id"`
	ProfileID      string         `json:"profile_id"`
	Parameters     map[string]any `json:"parameters"`
	Cwd            string         `json:"cwd,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
}

// TaskContract represents the immutable work specification dispatched to a worker.
// Revisions are tracked monotonically; contracts never contain transient attempt identities.
// Serialized form strictly adheres to task-contract.schema.json (additionalProperties: false).
type TaskContract struct {
	ContractID           string                `json:"contract_id"`
	TaskID               string                `json:"task_id"`
	RevisionNumber       int                   `json:"revision_number"`
	SupersedesContractID *string               `json:"supersedes_contract_id,omitempty"`
	PhaseID              string                `json:"phase_id"`
	Objective            string                `json:"objective"`
	Requirements         []string              `json:"requirements"`
	ArchitectureRefs     []string              `json:"architecture_refs"`
	BaseSHA              string                `json:"base_sha"`
	AllowedScope         []string              `json:"allowed_scope"`
	ForbiddenScope       []string              `json:"forbidden_scope"`
	Constraints          []string              `json:"constraints"`
	AcceptanceCriteria   []string              `json:"acceptance_criteria"`
	VerificationRequests []VerificationRequest `json:"verification_requests"`
	RequiredEvidence     []string              `json:"required_evidence"`
	WorkerProfile        string                `json:"worker_profile"`
	ReportContract       string                `json:"report_contract"`
	StopConditions       []string              `json:"stop_conditions"`
	IsImmutable          bool                  `json:"-"`
}

// TaskAttempt represents a single execution, retry, or revision iteration bound to a contract.
type TaskAttempt struct {
	AttemptID          string     `json:"attempt_id"`
	AttemptNumber      int        `json:"attempt_number"`
	TaskID             string     `json:"task_id"`
	ContractID         string     `json:"contract_id"`
	ExpectedReportPath string     `json:"expected_report_path"`
	StartedAt          time.Time  `json:"started_at"`
	EndedAt            *time.Time `json:"ended_at,omitempty"`
	WorkerReportRaw    string     `json:"worker_report_raw,omitempty"`
}

// ClaimedTestResult captures a single test result claimed by a worker.
type ClaimedTestResult struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	ExitCode int    `json:"exit_code,omitempty"`
	Output   string `json:"output,omitempty"`
}

// WorkerClaim records self-reported worker claims for an attempt.
type WorkerClaim struct {
	ClaimID             string              `json:"claim_id"`
	AttemptID           string              `json:"attempt_id"`
	ReportedHeadSHA     string              `json:"reported_head_sha"`
	ClaimedFilesChanged []string            `json:"claimed_files_changed"`
	Tests               []ClaimedTestResult `json:"tests,omitempty"`
	CreatedAt           time.Time           `json:"created_at"`
}

// ActualTestResult records verified test execution outcome by the Supervisor evidence runner.
type ActualTestResult struct {
	RequestID  string `json:"request_id"`
	ProfileID  string `json:"profile_id"`
	ExitCode   int    `json:"exit_code"`
	OutputHash string `json:"output_hash,omitempty"`
	Passed     bool   `json:"passed"`
}

// Evidence captures independently collected facts from Git and test runners bound to an attempt.
type Evidence struct {
	EvidenceID         string             `json:"evidence_id"`
	AttemptID          string             `json:"attempt_id"`
	ActualBaseSHA      string             `json:"actual_base_sha"`
	ActualHeadSHA      string             `json:"actual_head_sha"`
	ActualFilesChanged []string           `json:"actual_files_changed"`
	GitDiffHash        string             `json:"git_diff_hash"`
	TestLogs           []ActualTestResult `json:"test_logs,omitempty"`
	ScopeVerified      bool               `json:"scope_verified"`
	CollectedAt        time.Time          `json:"collected_at"`
}

// DecisionType represents the outcome of a supervisor review.
type DecisionType string

const (
	DecisionApprove         DecisionType = "APPROVE"
	DecisionRequestRevision DecisionType = "REQUEST_REVISION"
	DecisionBlock           DecisionType = "BLOCK"
)

// ReviewDecision records a human or LLM supervisor decision bound strictly to an attempt.
type ReviewDecision struct {
	DecisionID        string       `json:"decision_id"`
	TaskID            string       `json:"task_id"`
	AttemptID         string       `json:"attempt_id"`
	Decision          DecisionType `json:"decision"`
	ReviewerRationale string       `json:"reviewer_rationale"`
	RequiredRevisions []string     `json:"required_revisions,omitempty"`
	DecidedAt         time.Time    `json:"decided_at"`
}

// AuditEvent represents an append-only audit trail entry.
type AuditEvent struct {
	EventID    string         `json:"event_id"`
	EventType  string         `json:"event_type"`
	Timestamp  time.Time      `json:"timestamp"`
	TaskID     string         `json:"task_id,omitempty"`
	ContractID string         `json:"contract_id,omitempty"`
	AttemptID  string         `json:"attempt_id,omitempty"`
	PairID     string         `json:"pair_id,omitempty"`
	Actor      string         `json:"actor"`
	Details    map[string]any `json:"details,omitempty"`
}
