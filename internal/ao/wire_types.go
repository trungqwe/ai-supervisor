package ao

import (
	"encoding/json"
	"time"
)

type wireDaemonProbe struct {
	Status                  string `json:"status"`
	Service                 string `json:"service"`
	PID                     int    `json:"pid"`
	ExecutablePath          string `json:"executablePath,omitempty"`
	WorkingDirectory        string `json:"workingDirectory,omitempty"`
	StartupWorkingDirectory string `json:"startupWorkingDirectory,omitempty"`
	AppImagePath            string `json:"appImagePath,omitempty"`
}

type wireAgentInfo struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	AuthStatus string     `json:"authStatus,omitempty"`
	UsageCount int        `json:"usageCount,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

type wireAgentInventory struct {
	Supported  []wireAgentInfo `json:"supported"`
	Installed  []wireAgentInfo `json:"installed"`
	Authorized []wireAgentInfo `json:"authorized"`
}

type wireAgentInstallationObservation struct {
	State       string     `json:"state"`
	Freshness   string     `json:"freshness"`
	CheckedAt   *time.Time `json:"checkedAt,omitempty"`
	AttemptedAt *time.Time `json:"attemptedAt,omitempty"`
	ReasonCode  string     `json:"reasonCode,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

type wireAgentAuthenticationObservation struct {
	State       string     `json:"state"`
	Freshness   string     `json:"freshness"`
	CheckedAt   *time.Time `json:"checkedAt,omitempty"`
	AttemptedAt *time.Time `json:"attemptedAt,omitempty"`
	ReasonCode  string     `json:"reasonCode,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

type wireAgentReadinessSnapshot struct {
	ID                 string                             `json:"id"`
	Label              string                             `json:"label"`
	Installation       wireAgentInstallationObservation   `json:"installation"`
	Authentication     wireAgentAuthenticationObservation `json:"authentication"`
	EffectiveReadiness string                             `json:"effectiveReadiness"`
	UsageCount         int                                `json:"usageCount"`
	LastUsedAt         *time.Time                         `json:"lastUsedAt,omitempty"`
}

type wireAgentReadinessResponse struct {
	Agents []wireAgentReadinessSnapshot `json:"agents"`
}

type wireRegisterProjectRequest struct {
	Path      string `json:"path"`
	ProjectID string `json:"projectId"`
}

type wireProjectDetails struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Kind          string `json:"kind,omitempty"`
	Path          string `json:"path"`
	Repo          string `json:"repo,omitempty"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	Agent         string `json:"agent,omitempty"`
	FolderMissing bool   `json:"folderMissing,omitempty"`
}

type wireRegisterProjectResponse struct {
	Project *wireProjectDetails `json:"project"`
}

type wireGetProjectResponse struct {
	Status  string          `json:"status"`
	Project json.RawMessage `json:"project"`
}

type wireProjectOK struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Kind          string `json:"kind,omitempty"`
	Path          string `json:"path"`
	Repo          string `json:"repo,omitempty"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	Agent         string `json:"agent,omitempty"`
	FolderMissing bool   `json:"folderMissing,omitempty"`
}

type wireProjectDegraded struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind,omitempty"`
	Path         string `json:"path"`
	ResolveError string `json:"resolveError"`
}

// wireSessionView is the shared private projection of an AO public session view,
// reused across GetWorkerStatus, CreateWorkerSession, and ResumeWorker.
type wireSessionView struct {
	ID            string `json:"id"`
	ProjectID     string `json:"projectId,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Harness       string `json:"harness,omitempty"`
	Status        string `json:"status"`
	DisplayStatus string `json:"displayStatus,omitempty"`
	IsTerminated  bool   `json:"isTerminated"`
	Activity      struct {
		State          string    `json:"state"`
		LastActivityAt time.Time `json:"lastActivityAt"`
	} `json:"activity"`
	Branch             string `json:"branch,omitempty"`
	Model              string `json:"model,omitempty"`
	TerminalGeneration string `json:"terminalGeneration,omitempty"`
	PreviewURL         string `json:"previewUrl,omitempty"`
}

type wireSessionResponse struct {
	Session *wireSessionView `json:"session"`
}

type wireSpawnWorkerRequest struct {
	ProjectID string `json:"projectId"`
	Kind      string `json:"kind"`
	Harness   string `json:"harness"`
}

type wireSpawnSessionResponse struct {
	Session           *wireSessionView `json:"session"`
	PromptBytes       *int             `json:"promptBytes"`
	SystemPromptBytes *int             `json:"systemPromptBytes"`
}

type wireSendSessionMessageRequest struct {
	Message string `json:"message"`
}

type wireSendSessionMessageResponse struct {
	OK        bool   `json:"ok"`
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

type wireKillSessionResponse struct {
	OK        bool   `json:"ok"`
	SessionID string `json:"sessionId"`
	Freed     bool   `json:"freed"`
}

type wireRestoreSessionResponse struct {
	OK          bool             `json:"ok"`
	SessionID   string           `json:"sessionId"`
	RestoreMode string           `json:"restoreMode"`
	Session     *wireSessionView `json:"session"`
}

type wireAPIError struct {
	Error          string         `json:"error"`
	Code           string         `json:"code"`
	Message        string         `json:"message"`
	RequestID      string         `json:"requestId,omitempty"`
	Details        map[string]any `json:"details,omitempty"`
	ReportingOwner string         `json:"reporting_owner,omitempty"`
}

func toNormalizedHealthStatus(w *wireDaemonProbe) *HealthStatus {
	if w == nil {
		return nil
	}
	return &DaemonProbeResponse{
		Status:                  w.Status,
		Service:                 w.Service,
		PID:                     w.PID,
		ExecutablePath:          w.ExecutablePath,
		WorkingDirectory:        w.WorkingDirectory,
		StartupWorkingDirectory: w.StartupWorkingDirectory,
		AppImagePath:            w.AppImagePath,
	}
}

func toNormalizedReadinessStatus(w *wireDaemonProbe) *ReadinessStatus {
	return toNormalizedHealthStatus(w)
}

func toNormalizedAgentInfo(w wireAgentInfo) AgentInfo {
	return AgentInfo{
		ID:         w.ID,
		Label:      w.Label,
		AuthStatus: w.AuthStatus,
		UsageCount: w.UsageCount,
		LastUsedAt: w.LastUsedAt,
	}
}

func toNormalizedAgentInventory(w *wireAgentInventory) *AgentInventory {
	if w == nil {
		return nil
	}
	inv := &AgentInventory{
		Supported:  make([]AgentInfo, len(w.Supported)),
		Installed:  make([]AgentInfo, len(w.Installed)),
		Authorized: make([]AgentInfo, len(w.Authorized)),
	}
	for i, a := range w.Supported {
		inv.Supported[i] = toNormalizedAgentInfo(a)
	}
	for i, a := range w.Installed {
		inv.Installed[i] = toNormalizedAgentInfo(a)
	}
	for i, a := range w.Authorized {
		inv.Authorized[i] = toNormalizedAgentInfo(a)
	}
	return inv
}

func toNormalizedAgentReadiness(w *wireAgentReadinessSnapshot) *AgentReadinessSnapshot {
	if w == nil {
		return nil
	}
	return &AgentReadinessSnapshot{
		ID:    w.ID,
		Label: w.Label,
		Installation: AgentInstallationObservation{
			State:       w.Installation.State,
			Freshness:   w.Installation.Freshness,
			CheckedAt:   w.Installation.CheckedAt,
			AttemptedAt: w.Installation.AttemptedAt,
			ReasonCode:  w.Installation.ReasonCode,
			Reason:      w.Installation.Reason,
		},
		Authentication: AgentAuthenticationObservation{
			State:       w.Authentication.State,
			Freshness:   w.Authentication.Freshness,
			CheckedAt:   w.Authentication.CheckedAt,
			AttemptedAt: w.Authentication.AttemptedAt,
			ReasonCode:  w.Authentication.ReasonCode,
			Reason:      w.Authentication.Reason,
		},
		EffectiveReadiness: w.EffectiveReadiness,
		UsageCount:         w.UsageCount,
		LastUsedAt:         w.LastUsedAt,
	}
}

func toNormalizedProjectFromRegister(w *wireProjectDetails) *Project {
	if w == nil {
		return nil
	}
	return &Project{
		ID:            w.ID,
		Name:          w.Name,
		Kind:          w.Kind,
		Path:          w.Path,
		Repo:          w.Repo,
		DefaultBranch: w.DefaultBranch,
		Agent:         w.Agent,
		FolderMissing: w.FolderMissing,
		Status:        "ok",
		IsDegraded:    false,
		ResolveError:  "",
	}
}

func toNormalizedProjectOK(w *wireProjectOK) *Project {
	if w == nil {
		return nil
	}
	return &Project{
		ID:            w.ID,
		Name:          w.Name,
		Kind:          w.Kind,
		Path:          w.Path,
		Repo:          w.Repo,
		DefaultBranch: w.DefaultBranch,
		Agent:         w.Agent,
		FolderMissing: w.FolderMissing,
		Status:        "ok",
		IsDegraded:    false,
		ResolveError:  "",
	}
}

func toNormalizedProjectDegraded(w *wireProjectDegraded) *Project {
	if w == nil {
		return nil
	}
	return &Project{
		ID:           w.ID,
		Name:         w.Name,
		Kind:         w.Kind,
		Path:         w.Path,
		Status:       "degraded",
		IsDegraded:   true,
		ResolveError: w.ResolveError,
	}
}

func toNormalizedWorkerStatusFromView(s *wireSessionView, state ActivityState) WorkerStatus {
	if s == nil {
		return WorkerStatus{}
	}
	return WorkerStatus{
		ID:            s.ID,
		ProjectID:     s.ProjectID,
		Status:        s.Status,
		DisplayStatus: s.DisplayStatus,
		IsTerminated:  s.IsTerminated,
		Activity: ActivitySnapshot{
			State:          state,
			LastActivityAt: s.Activity.LastActivityAt,
		},
		Harness:            s.Harness,
		Branch:             s.Branch,
		Model:              s.Model,
		TerminalGeneration: s.TerminalGeneration,
		PreviewURL:         s.PreviewURL,
	}
}

func toNormalizedWorkerStatus(w *wireSessionResponse, state ActivityState) *WorkerStatus {
	if w == nil || w.Session == nil {
		return nil
	}
	res := toNormalizedWorkerStatusFromView(w.Session, state)
	return &res
}

func toNormalizedCreateWorkerSessionResult(resp *wireSpawnSessionResponse, state ActivityState) *CreateWorkerSessionResult {
	if resp == nil {
		return nil
	}
	var promptBytes, systemPromptBytes int
	if resp.PromptBytes != nil {
		promptBytes = *resp.PromptBytes
	}
	if resp.SystemPromptBytes != nil {
		systemPromptBytes = *resp.SystemPromptBytes
	}
	return &CreateWorkerSessionResult{
		Session:           toNormalizedWorkerStatusFromView(resp.Session, state),
		PromptBytes:       promptBytes,
		SystemPromptBytes: systemPromptBytes,
	}
}
