package workflow

import (
	"fmt"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// TaskStateCount is the number of canonical task states (13).
const TaskStateCount = 13

// CanonicalStateCount is an alias for TaskStateCount (13).
const CanonicalStateCount = TaskStateCount

// DomainTransitionCount is the count of valid direct state-to-state transitions (22).
const DomainTransitionCount = 22

// CanonicalGraphEdgeCount is the total number of canonical edges in the workflow graph (25),
// as specified in docs/06_WORKFLOW_STATE_MACHINE.md.
// This comprises:
// - 1 initial pseudo-edge: [*] -> DRAFT
// - 22 executable state-to-state domain transitions
// - 2 terminal pseudo-edges: APPROVED -> [*], CANCELLED -> [*]
const CanonicalGraphEdgeCount = 25

// CanonicalEdgeCount is an alias for CanonicalGraphEdgeCount (25).
const CanonicalEdgeCount = CanonicalGraphEdgeCount

// InvalidTransitionError represents a disallowed transition between two states.
type InvalidTransitionError struct {
	From domain.TaskState
	To   domain.TaskState
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid task state transition from %q to %q", e.From, e.To)
}

// CanonicalEdge represents a directed edge in the canonical workflow specification graph.
type CanonicalEdge struct {
	From string
	To   string
}

// CanonicalGraphEdges returns all 25 canonical workflow graph edges from docs/06_WORKFLOW_STATE_MACHINE.md.
func CanonicalGraphEdges() []CanonicalEdge {
	return []CanonicalEdge{
		{From: "[*]", To: "DRAFT"},
		{From: "DRAFT", To: "READY"},
		{From: "DRAFT", To: "CANCELLED"},
		{From: "READY", To: "DISPATCHED"},
		{From: "READY", To: "CANCELLED"},
		{From: "DISPATCHED", To: "RUNNING"},
		{From: "DISPATCHED", To: "FAILED"},
		{From: "RUNNING", To: "REPORT_READY"},
		{From: "RUNNING", To: "FAILED"},
		{From: "RUNNING", To: "BLOCKED"},
		{From: "REPORT_READY", To: "EVIDENCE_READY"},
		{From: "REPORT_READY", To: "FAILED"},
		{From: "EVIDENCE_READY", To: "REVIEWING"},
		{From: "EVIDENCE_READY", To: "FAILED"},
		{From: "REVIEWING", To: "APPROVED"},
		{From: "REVIEWING", To: "REVISION_REQUIRED"},
		{From: "REVIEWING", To: "BLOCKED"},
		{From: "REVISION_REQUIRED", To: "READY"},
		{From: "BLOCKED", To: "HUMAN_REQUIRED"},
		{From: "FAILED", To: "HUMAN_REQUIRED"},
		{From: "FAILED", To: "READY"},
		{From: "HUMAN_REQUIRED", To: "DRAFT"},
		{From: "HUMAN_REQUIRED", To: "CANCELLED"},
		{From: "APPROVED", To: "[*]"},
		{From: "CANCELLED", To: "[*]"},
	}
}

// CanonicalEdges is an alias for CanonicalGraphEdges.
func CanonicalEdges() []CanonicalEdge {
	return CanonicalGraphEdges()
}

// StateMachine provides pure evaluation of workflow state transitions.
type StateMachine struct{}

// NewStateMachine creates a new StateMachine instance.
func NewStateMachine() *StateMachine {
	return &StateMachine{}
}

// CanTransition evaluates whether a transition between from and to is permitted.
func (sm *StateMachine) CanTransition(from, to domain.TaskState) bool {
	return CanTransition(from, to)
}

// Transition validates that a transition between from and to is permitted,
// returning an *InvalidTransitionError if not.
func (sm *StateMachine) Transition(from, to domain.TaskState) error {
	return Transition(from, to)
}

// allowedTransitions encodes the exact 22 canonical state-to-state domain transitions.
// Terminal states APPROVED and CANCELLED have zero outgoing transitions.
var allowedTransitions = map[domain.TaskState]map[domain.TaskState]struct{}{
	domain.StateDraft: {
		domain.StateReady:     {},
		domain.StateCancelled: {},
	},
	domain.StateReady: {
		domain.StateDispatched: {},
		domain.StateCancelled:  {},
	},
	domain.StateDispatched: {
		domain.StateRunning: {},
		domain.StateFailed:  {},
	},
	domain.StateRunning: {
		domain.StateReportReady: {},
		domain.StateFailed:      {},
		domain.StateBlocked:     {},
	},
	domain.StateReportReady: {
		domain.StateEvidenceReady: {},
		domain.StateFailed:        {},
	},
	domain.StateEvidenceReady: {
		domain.StateReviewing: {},
		domain.StateFailed:    {},
	},
	domain.StateReviewing: {
		domain.StateApproved:         {},
		domain.StateRevisionRequired: {},
		domain.StateBlocked:          {},
	},
	domain.StateApproved: {
		// Terminal state: 0 outgoing transitions
	},
	domain.StateRevisionRequired: {
		domain.StateReady: {},
	},
	domain.StateBlocked: {
		domain.StateHumanRequired: {},
	},
	domain.StateFailed: {
		domain.StateHumanRequired: {},
		domain.StateReady:         {},
	},
	domain.StateHumanRequired: {
		domain.StateDraft:     {},
		domain.StateCancelled: {},
	},
	domain.StateCancelled: {
		// Terminal state: 0 outgoing transitions
	},
}

// CanTransition evaluates whether transitioning from 'from' to 'to' is allowed by canonical rules.
func CanTransition(from, to domain.TaskState) bool {
	targets, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, allowed := targets[to]
	return allowed
}

// Transition validates that a transition from 'from' to 'to' is permitted,
// returning an *InvalidTransitionError preserving From and To if rejected.
func Transition(from, to domain.TaskState) error {
	if !CanTransition(from, to) {
		return &InvalidTransitionError{From: from, To: to}
	}
	return nil
}

// AllowedTransitions returns a copy of all 22 allowed domain transitions.
func AllowedTransitions() [][2]domain.TaskState {
	transitions := make([][2]domain.TaskState, 0, DomainTransitionCount)
	for from, targets := range allowedTransitions {
		for to := range targets {
			transitions = append(transitions, [2]domain.TaskState{from, to})
		}
	}
	return transitions
}
