package ao

import "time"

// DaemonProbeResponse represents the payload returned by /healthz and /readyz probes.
type DaemonProbeResponse struct {
	Status                  string `json:"status"`
	Service                 string `json:"service"`
	PID                     int    `json:"pid"`
	ExecutablePath          string `json:"executablePath,omitempty"`
	WorkingDirectory        string `json:"workingDirectory,omitempty"`
	StartupWorkingDirectory string `json:"startupWorkingDirectory,omitempty"`
	AppImagePath            string `json:"appImagePath,omitempty"`
}

// HealthStatus is an alias for DaemonProbeResponse returned by CheckHealth.
type HealthStatus = DaemonProbeResponse

// ReadinessStatus is an alias for DaemonProbeResponse returned by CheckReadiness.
type ReadinessStatus = DaemonProbeResponse

// AgentInfo describes a single agent harness in the AO inventory.
type AgentInfo struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	AuthStatus string     `json:"authStatus,omitempty"`
	UsageCount int        `json:"usageCount,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

// AgentInventory is the response returned by GET /api/v1/agents.
type AgentInventory struct {
	Supported  []AgentInfo `json:"supported"`
	Installed  []AgentInfo `json:"installed"`
	Authorized []AgentInfo `json:"authorized"`
}

// AgentReadinessSnapshot describes the detailed readiness observation of an agent harness.
type AgentReadinessSnapshot struct {
	ID                 string     `json:"id"`
	Label              string     `json:"label"`
	EffectiveReadiness string     `json:"effectiveReadiness"`
	UsageCount         int        `json:"usageCount,omitempty"`
	LastUsedAt         *time.Time `json:"lastUsedAt,omitempty"`
}

// AgentReadinessResponse is the response returned by GET /api/v1/agents/readiness.
type AgentReadinessResponse struct {
	Agents []AgentReadinessSnapshot `json:"agents"`
}

// Project represents a registered codebase workspace in AO.
type Project struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Kind          string `json:"kind,omitempty"`
	Path          string `json:"path"`
	Repo          string `json:"repo,omitempty"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	Agent         string `json:"agent,omitempty"`
	FolderMissing bool   `json:"folderMissing,omitempty"`
}

// RegisterProjectRequest is the payload sent to POST /api/v1/projects.
type RegisterProjectRequest struct {
	Path      string  `json:"path"`
	ProjectID *string `json:"projectId,omitempty"`
	Name      *string `json:"name,omitempty"`
}

// ProjectResponse is the response returned by POST /api/v1/projects.
type ProjectResponse struct {
	Project Project `json:"project"`
}

// GetProjectResponse is the response returned by GET /api/v1/projects/{id}.
type GetProjectResponse struct {
	Status  string  `json:"status"`
	Project Project `json:"project"`
}

// ActivitySnapshot captures the low-level runtime activity state of an AO session.
type ActivitySnapshot struct {
	State          string    `json:"state"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// WorkerStatus represents the normalized read model of an AO worker session returned by GET /api/v1/sessions/{id}.
type WorkerStatus struct {
	ID                 string           `json:"id"`
	ProjectID          string           `json:"projectId,omitempty"`
	Status             string           `json:"status"`
	DisplayStatus      string           `json:"displayStatus,omitempty"`
	IsTerminated       bool             `json:"isTerminated"`
	Activity           ActivitySnapshot `json:"activity"`
	Harness            string           `json:"harness,omitempty"`
	Branch             string           `json:"branch,omitempty"`
	Model              string           `json:"model,omitempty"`
	TerminalGeneration string           `json:"terminalGeneration,omitempty"`
	PreviewURL         string           `json:"previewUrl,omitempty"`
}

// SessionResponse is the response envelope returned by GET /api/v1/sessions/{id}.
type SessionResponse struct {
	Session WorkerStatus `json:"session"`
}
