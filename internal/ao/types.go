package ao

import (
	"errors"
	"time"
)

// DaemonServiceAO is the pinned upstream service identifier returned by the AO daemon in health and readiness probes.
const DaemonServiceAO = "agent-orchestrator-daemon"

// DaemonProbeResponse represents the normalized payload returned by /healthz and /readyz probes.
type DaemonProbeResponse struct {
	Status                  string
	Service                 string
	PID                     int
	ExecutablePath          string
	WorkingDirectory        string
	StartupWorkingDirectory string
	AppImagePath            string
}

// HealthStatus is an alias for DaemonProbeResponse returned by CheckHealth.
type HealthStatus = DaemonProbeResponse

// ReadinessStatus is an alias for DaemonProbeResponse returned by CheckReadiness.
type ReadinessStatus = DaemonProbeResponse

// AgentInfo describes a single agent harness in the AO inventory.
type AgentInfo struct {
	ID         string
	Label      string
	AuthStatus string
	UsageCount int
	LastUsedAt *time.Time
}

// AgentInventory is the response returned by GET /api/v1/agents.
type AgentInventory struct {
	Supported  []AgentInfo
	Installed  []AgentInfo
	Authorized []AgentInfo
}

// AgentInstallationObservation captures installation state and freshness for an agent harness.
type AgentInstallationObservation struct {
	State       string
	Freshness   string
	CheckedAt   *time.Time
	AttemptedAt *time.Time
	ReasonCode  string
	Reason      string
}

// AgentAuthenticationObservation captures authentication state and freshness for an agent harness.
type AgentAuthenticationObservation struct {
	State       string
	Freshness   string
	CheckedAt   *time.Time
	AttemptedAt *time.Time
	ReasonCode  string
	Reason      string
}

// AgentReadinessSnapshot describes the complete readiness observation of an agent harness.
type AgentReadinessSnapshot struct {
	ID                 string
	Label              string
	Installation       AgentInstallationObservation
	Authentication     AgentAuthenticationObservation
	EffectiveReadiness string
	UsageCount         int
	LastUsedAt         *time.Time
}

// Project represents a registered codebase workspace in AO.
type Project struct {
	ID            string
	Name          string
	Kind          string
	Path          string
	Repo          string
	DefaultBranch string
	Agent         string
	FolderMissing bool

	// Status indicates if project configuration is healthy ("ok") or degraded ("degraded").
	Status       string
	IsDegraded   bool
	ResolveError string
}

// ActivityState defines canonical runtime activity states observed by AO.
type ActivityState string

const (
	ActivityStateActive       ActivityState = "active"
	ActivityStateIdle         ActivityState = "idle"
	ActivityStateWaitingInput ActivityState = "waiting_input"
	ActivityStateBlocked      ActivityState = "blocked"
	ActivityStateExited       ActivityState = "exited"
)

// ActivitySnapshot captures the low-level runtime activity state of an AO session.
type ActivitySnapshot struct {
	State          ActivityState
	LastActivityAt time.Time
}

// WorkerStatus represents the normalized read model of an AO worker session returned by GET /api/v1/sessions/{id}.
type WorkerStatus struct {
	ID                 string
	ProjectID          string
	Status             string
	DisplayStatus      string
	IsTerminated       bool
	Activity           ActivitySnapshot
	Harness            string
	Branch             string
	Model              string
	TerminalGeneration string
	PreviewURL         string
}

// CreateWorkerSessionResult represents the normalized result of a successful worker session creation.
type CreateWorkerSessionResult struct {
	Session           WorkerStatus
	PromptBytes       int
	SystemPromptBytes int
}

// DispatchTaskResult represents the normalized confirmation of an immutable task message dispatch.
type DispatchTaskResult struct {
	SessionID string
	Message   string
}

// StopWorkerResult represents the normalized outcome of an intentional worker stop/kill request.
type StopWorkerResult struct {
	SessionID string
	Freed     bool
}

// RestoreMode indicates the provider continuation mode used when restoring a session.
type RestoreMode string

const (
	RestoreModeNative      RestoreMode = "native"
	RestoreModeSavedPrompt RestoreMode = "saved_prompt"
	RestoreModeFresh       RestoreMode = "fresh"
)

// ResumeWorkerResult represents the normalized outcome of restoring a terminated worker session.
type ResumeWorkerResult struct {
	SessionID   string
	RestoreMode RestoreMode
	Session     WorkerStatus
}
// WorkspaceReadOptions defines bounded limits for reading workspace files from AO.
type WorkspaceReadOptions struct {
	MaxWireBytes int64 // Maximum allowed wire envelope payload size in bytes (> 0)
	MaxBytes     int64 // Maximum allowed decoded file content size in bytes (> 0)
}

// WorkspaceFileStatus defines valid enum values for WorkspaceFileResponse.Status.
type WorkspaceFileStatus string

const (
	WorkspaceFileStatusUnmodified WorkspaceFileStatus = "unmodified"
	WorkspaceFileStatusModified   WorkspaceFileStatus = "modified"
	WorkspaceFileStatusAdded      WorkspaceFileStatus = "added"
	WorkspaceFileStatusDeleted    WorkspaceFileStatus = "deleted"
)

// WorkspaceFileResponse represents the JSON envelope returned by GET /api/v1/sessions/{sessionId}/workspace/file
// pursuant to Untrivial-ai/agent-orchestrator v0.13.0 (commit 15e9ea971f1711ec8b50e157d6eb300db6cbe0d6).
type WorkspaceFileResponse struct {
	SessionID        string `json:"sessionId"`
	Path             string `json:"path"`
	Content          string `json:"content"`
	Binary           bool   `json:"binary"`
	Deleted          bool   `json:"deleted"`
	ContentTruncated bool   `json:"contentTruncated"`
	Size             int64  `json:"size"`
	Status           string `json:"status"` // "unmodified" | "modified" | "added" | "deleted"
	WorkspaceVersion string `json:"workspaceVersion"`
	Diff             string `json:"diff"`
	DiffTruncated    bool   `json:"diffTruncated"`
	Editable         bool   `json:"editable"`
	FileFingerprint  string `json:"fileFingerprint"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
}

// Workspace file retrieval errors
var (
	ErrPayloadTooLarge      = errors.New("ao: payload exceeds maximum allowed size")
	ErrWorkspaceFileDeleted = errors.New("ao: workspace file is marked deleted")
)
