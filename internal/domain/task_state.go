package domain

// TaskState represents the canonical lifecycle state of a task.
// There are exactly 13 canonical states defined in docs/06_WORKFLOW_STATE_MACHINE.md.
type TaskState string

const (
	StateDraft            TaskState = "DRAFT"
	StateReady            TaskState = "READY"
	StateDispatched       TaskState = "DISPATCHED"
	StateRunning          TaskState = "RUNNING"
	StateReportReady      TaskState = "REPORT_READY"
	StateEvidenceReady    TaskState = "EVIDENCE_READY"
	StateReviewing        TaskState = "REVIEWING"
	StateApproved         TaskState = "APPROVED"
	StateRevisionRequired TaskState = "REVISION_REQUIRED"
	StateBlocked          TaskState = "BLOCKED"
	StateFailed           TaskState = "FAILED"
	StateHumanRequired    TaskState = "HUMAN_REQUIRED"
	StateCancelled        TaskState = "CANCELLED"
)

// AllTaskStates returns a slice of all 13 canonical task states.
func AllTaskStates() []TaskState {
	return []TaskState{
		StateDraft,
		StateReady,
		StateDispatched,
		StateRunning,
		StateReportReady,
		StateEvidenceReady,
		StateReviewing,
		StateApproved,
		StateRevisionRequired,
		StateBlocked,
		StateFailed,
		StateHumanRequired,
		StateCancelled,
	}
}

// IsValid reports whether s is one of the 13 canonical task states.
func (s TaskState) IsValid() bool {
	switch s {
	case StateDraft, StateReady, StateDispatched, StateRunning,
		StateReportReady, StateEvidenceReady, StateReviewing,
		StateApproved, StateRevisionRequired, StateBlocked,
		StateFailed, StateHumanRequired, StateCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether s is a terminal state (APPROVED or CANCELLED).
func (s TaskState) IsTerminal() bool {
	return s == StateApproved || s == StateCancelled
}
