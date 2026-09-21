package workflow_test

import (
	"errors"
	"testing"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/workflow"
)

// expectedCanonical25Edges contains the 25 edges independently defined by docs/06_WORKFLOW_STATE_MACHINE.md.
var expectedCanonical25Edges = []struct {
	From string
	To   string
}{
	{"[*]", "DRAFT"},
	{"DRAFT", "READY"},
	{"DRAFT", "CANCELLED"},
	{"READY", "DISPATCHED"},
	{"READY", "CANCELLED"},
	{"DISPATCHED", "RUNNING"},
	{"DISPATCHED", "FAILED"},
	{"RUNNING", "REPORT_READY"},
	{"RUNNING", "FAILED"},
	{"RUNNING", "BLOCKED"},
	{"REPORT_READY", "EVIDENCE_READY"},
	{"REPORT_READY", "FAILED"},
	{"EVIDENCE_READY", "REVIEWING"},
	{"EVIDENCE_READY", "FAILED"},
	{"REVIEWING", "APPROVED"},
	{"REVIEWING", "REVISION_REQUIRED"},
	{"REVIEWING", "BLOCKED"},
	{"REVISION_REQUIRED", "READY"},
	{"BLOCKED", "HUMAN_REQUIRED"},
	{"FAILED", "HUMAN_REQUIRED"},
	{"FAILED", "READY"},
	{"HUMAN_REQUIRED", "DRAFT"},
	{"HUMAN_REQUIRED", "CANCELLED"},
	{"APPROVED", "[*]"},
	{"CANCELLED", "[*]"},
}

// expectedDomainTransitions defines the 22 canonical state-to-state transitions.
var expectedDomainTransitions = [][2]domain.TaskState{
	{domain.StateDraft, domain.StateReady},
	{domain.StateDraft, domain.StateCancelled},
	{domain.StateReady, domain.StateDispatched},
	{domain.StateReady, domain.StateCancelled},
	{domain.StateDispatched, domain.StateRunning},
	{domain.StateDispatched, domain.StateFailed},
	{domain.StateRunning, domain.StateReportReady},
	{domain.StateRunning, domain.StateFailed},
	{domain.StateRunning, domain.StateBlocked},
	{domain.StateReportReady, domain.StateEvidenceReady},
	{domain.StateReportReady, domain.StateFailed},
	{domain.StateEvidenceReady, domain.StateReviewing},
	{domain.StateEvidenceReady, domain.StateFailed},
	{domain.StateReviewing, domain.StateApproved},
	{domain.StateReviewing, domain.StateRevisionRequired},
	{domain.StateReviewing, domain.StateBlocked},
	{domain.StateRevisionRequired, domain.StateReady},
	{domain.StateBlocked, domain.StateHumanRequired},
	{domain.StateFailed, domain.StateHumanRequired},
	{domain.StateFailed, domain.StateReady},
	{domain.StateHumanRequired, domain.StateDraft},
	{domain.StateHumanRequired, domain.StateCancelled},
}

// TestCanonicalCounts asserts canonical counts: 13 states, 22 domain transitions, and 25 graph edges.
func TestCanonicalCounts(t *testing.T) {
	states := domain.AllTaskStates()
	if len(states) != workflow.TaskStateCount {
		t.Fatalf("expected state count %d, got %d", workflow.TaskStateCount, len(states))
	}
	if len(states) != 13 {
		t.Fatalf("expected 13 states, got %d", len(states))
	}

	allowed := workflow.AllowedTransitions()
	if len(allowed) != workflow.DomainTransitionCount {
		t.Fatalf("expected domain transition count %d, got %d", workflow.DomainTransitionCount, len(allowed))
	}
	if len(allowed) != 22 {
		t.Fatalf("expected 22 domain transitions, got %d", len(allowed))
	}

	if len(expectedCanonical25Edges) != workflow.CanonicalGraphEdgeCount {
		t.Fatalf("expected fixture edge count %d, got %d", workflow.CanonicalGraphEdgeCount, len(expectedCanonical25Edges))
	}
	if len(expectedCanonical25Edges) != 25 {
		t.Fatalf("expected 25 canonical graph edges, got %d", len(expectedCanonical25Edges))
	}

	graphEdges := workflow.CanonicalGraphEdges()
	if len(graphEdges) != 25 {
		t.Fatalf("expected CanonicalGraphEdges() to return 25 items, got %d", len(graphEdges))
	}

	for i, edge := range expectedCanonical25Edges {
		if graphEdges[i].From != edge.From || graphEdges[i].To != edge.To {
			t.Errorf("edge[%d] mismatch: expected %s -> %s, got %s -> %s",
				i, edge.From, edge.To, graphEdges[i].From, graphEdges[i].To)
		}
	}
}

// TestEveryCanonicalAllowedTransitionPasses asserts every canonical allowed transition is accepted.
func TestEveryCanonicalAllowedTransitionPasses(t *testing.T) {
	sm := workflow.NewStateMachine()

	if len(expectedDomainTransitions) != workflow.DomainTransitionCount {
		t.Fatalf("expected %d domain transitions, got %d", workflow.DomainTransitionCount, len(expectedDomainTransitions))
	}

	for _, tr := range expectedDomainTransitions {
		from, to := tr[0], tr[1]

		if !workflow.CanTransition(from, to) {
			t.Errorf("CanTransition(%q, %q) expected true, got false", from, to)
		}
		if err := workflow.Transition(from, to); err != nil {
			t.Errorf("Transition(%q, %q) expected nil err, got %v", from, to, err)
		}

		if !sm.CanTransition(from, to) {
			t.Errorf("sm.CanTransition(%q, %q) expected true, got false", from, to)
		}
		if err := sm.Transition(from, to); err != nil {
			t.Errorf("sm.Transition(%q, %q) expected nil err, got %v", from, to, err)
		}
	}
}

// TestEveryNonCanonicalStatePairRejected asserts all non-canonical pairs out of 169 (13x13) are rejected.
func TestEveryNonCanonicalStatePairRejected(t *testing.T) {
	allowedSet := make(map[[2]domain.TaskState]bool)
	for _, tr := range expectedDomainTransitions {
		allowedSet[tr] = true
	}

	states := domain.AllTaskStates()
	totalPairs := 0
	rejectedCount := 0

	for _, from := range states {
		for _, to := range states {
			totalPairs++
			pair := [2]domain.TaskState{from, to}

			if allowedSet[pair] {
				continue
			}

			// Must be rejected
			rejectedCount++
			if workflow.CanTransition(from, to) {
				t.Errorf("CanTransition(%q, %q) expected false, got true", from, to)
			}

			err := workflow.Transition(from, to)
			if err == nil {
				t.Errorf("Transition(%q, %q) expected error, got nil", from, to)
				continue
			}

			var transErr *workflow.InvalidTransitionError
			if !errors.As(err, &transErr) {
				t.Errorf("expected error of type *InvalidTransitionError, got %T", err)
			} else {
				if transErr.From != from || transErr.To != to {
					t.Errorf("InvalidTransitionError mismatch: expected (%q, %q), got (%q, %q)",
						from, to, transErr.From, transErr.To)
				}
			}
		}
	}

	expectedTotalPairs := 13 * 13 // 169
	expectedRejected := 169 - 22  // 147
	if totalPairs != expectedTotalPairs {
		t.Fatalf("expected %d total pairs, checked %d", expectedTotalPairs, totalPairs)
	}
	if rejectedCount != expectedRejected {
		t.Fatalf("expected %d rejected pairs, got %d", expectedRejected, rejectedCount)
	}
}

// TestTerminalStatesHaveZeroOutgoingTransitions explicitly verifies APPROVED and CANCELLED have 0 outgoing transitions.
func TestTerminalStatesHaveZeroOutgoingTransitions(t *testing.T) {
	states := domain.AllTaskStates()

	for _, target := range states {
		if workflow.CanTransition(domain.StateApproved, target) {
			t.Errorf("APPROVED must have zero outgoing transitions, but CanTransition(APPROVED, %q) returned true", target)
		}
		if err := workflow.Transition(domain.StateApproved, target); err == nil {
			t.Errorf("APPROVED must have zero outgoing transitions, but Transition(APPROVED, %q) returned nil", target)
		}

		if workflow.CanTransition(domain.StateCancelled, target) {
			t.Errorf("CANCELLED must have zero outgoing transitions, but CanTransition(CANCELLED, %q) returned true", target)
		}
		if err := workflow.Transition(domain.StateCancelled, target); err == nil {
			t.Errorf("CANCELLED must have zero outgoing transitions, but Transition(CANCELLED, %q) returned nil", target)
		}
	}
}

// TestUnknownAndInvalidStatesRejected asserts deterministic rejection for invalid/unknown states.
func TestUnknownAndInvalidStatesRejected(t *testing.T) {
	invalidStates := []domain.TaskState{
		"",
		"UNKNOWN_STATE",
		"REPORT_MISSING",
		"REPORT_INVALID",
		"REPORT_IDENTITY_MISMATCH",
		"[*]",
	}

	for _, inv := range invalidStates {
		for _, s := range domain.AllTaskStates() {
			if workflow.CanTransition(inv, s) {
				t.Errorf("CanTransition(%q, %q) expected false", inv, s)
			}
			if err := workflow.Transition(inv, s); err == nil {
				t.Errorf("Transition(%q, %q) expected error", inv, s)
			}

			if workflow.CanTransition(s, inv) {
				t.Errorf("CanTransition(%q, %q) expected false", s, inv)
			}
			if err := workflow.Transition(s, inv); err == nil {
				t.Errorf("Transition(%q, %q) expected error", s, inv)
			}
		}
	}
}

// TestNoAccidentalEdges asserts the production table has exactly 22 domain transitions.
func TestNoAccidentalEdges(t *testing.T) {
	allowed := workflow.AllowedTransitions()
	if len(allowed) != 22 {
		t.Fatalf("expected exactly 22 allowed domain transitions, got %d (fail if 23rd edge added)", len(allowed))
	}
}
