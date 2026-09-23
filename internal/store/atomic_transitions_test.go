package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func setupRunningBoundAttempt(t *testing.T, s *Store, taskID, contractID, attemptID string) domain.TaskAttempt {
	t.Helper()
	ctx := context.Background()
	_, attempt := setupBoundAttempt(t, s, taskID, contractID, attemptID)
	if err := s.TransitionTask(ctx, taskID, domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatalf("transition DISPATCHED -> RUNNING: %v", err)
	}
	return attempt
}

func TestStore_AtomicTerminalTransition(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	attempt := setupRunningBoundAttempt(t, s, "task-terminal", "contract-terminal", "attempt-terminal")
	if err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateRunning, domain.StateFailed,
		"stop confirmation deadline expired", attempt.AttemptID, "STOP_CONFIRMATION_TIMEOUT"); err != nil {
		t.Fatalf("AtomicTerminalTransition: %v", err)
	}
	task, _ := s.GetTask(ctx, attempt.TaskID)
	closed, _ := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if task.State != domain.StateFailed || closed.EndedAt == nil || closed.RecoveryDisposition == nil || *closed.RecoveryDisposition != "STOP_CONFIRMATION_TIMEOUT" {
		t.Fatalf("terminal transition was not atomic: task=%+v attempt=%+v", task, closed)
	}
	events, err := s.ListAuditEvents(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ListAuditEvents: %v", err)
	}
	if len(events) != 2 || events[1].Event.EventType != domain.AuditTaskStateTransition || events[1].Event.AttemptID != attempt.AttemptID {
		t.Fatalf("terminal audit event mismatch: %+v", events)
	}
	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("VerifyAuditChain after terminal transition: %v", err)
	}
}

func TestStore_AtomicTerminalTransitionCASConflictRollsBack(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-cas", "contract-cas", "attempt-cas")
	err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateRunning, domain.StateFailed,
		"wrong expected state", attempt.AttemptID, "WORKER_TERMINATION_UNKNOWN")
	if !errors.Is(err, ErrStateConflict) {
		t.Fatalf("CAS conflict error = %v, want ErrStateConflict", err)
	}
	task, _ := s.GetTask(ctx, attempt.TaskID)
	open, _ := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if task.State != domain.StateDispatched || open.EndedAt != nil || open.RecoveryDisposition != nil {
		t.Fatalf("CAS conflict did not roll back: task=%+v attempt=%+v", task, open)
	}
}

func TestStore_AtomicTransitionRejectsCrossTaskAndOldAttempt(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	attemptA := setupRunningBoundAttempt(t, s, "task-attempt-a", "contract-attempt-a", "attempt-a1")
	attemptB := setupRunningBoundAttempt(t, s, "task-attempt-b", "contract-attempt-b", "attempt-b1")

	err := s.AtomicTerminalTransition(ctx, attemptA.TaskID, domain.StateRunning, domain.StateFailed,
		"cross task", attemptB.AttemptID, "WORKER_TERMINATION_UNKNOWN")
	if !errors.Is(err, ErrAttemptLineageMismatch) {
		t.Fatalf("cross-task attempt error = %v, want ErrAttemptLineageMismatch", err)
	}
	taskA, _ := s.GetTask(ctx, attemptA.TaskID)
	if taskA.State != domain.StateRunning {
		t.Fatalf("cross-task attempt changed task A to %s", taskA.State)
	}

	if err := s.AtomicTerminalTransition(ctx, attemptA.TaskID, domain.StateRunning, domain.StateFailed,
		"first failure", attemptA.AttemptID, "WORKER_TERMINATION_UNKNOWN"); err != nil {
		t.Fatalf("close first attempt: %v", err)
	}
	if err := s.TransitionTask(ctx, attemptA.TaskID, domain.StateFailed, domain.StateReady); err != nil {
		t.Fatalf("FAILED -> READY: %v", err)
	}
	reportPath, _ := CanonicalExpectedReportPath(attemptA.TaskID, "attempt-a2")
	second, err := s.PrepareBoundDispatch(ctx, attemptA.TaskID, attemptA.ContractID, "attempt-a2", reportPath, time.Now().UTC(), DispatchBinding{
		OperationID:        "dispatch-attempt-a2",
		SessionID:          *attemptA.SessionID,
		TerminalGeneration: *attemptA.TerminalGeneration,
	})
	if err != nil {
		t.Fatalf("prepare second attempt: %v", err)
	}
	if err := s.TransitionTask(ctx, attemptA.TaskID, domain.StateDispatched, domain.StateRunning); err != nil {
		t.Fatalf("second attempt RUNNING: %v", err)
	}
	err = s.AtomicTerminalTransition(ctx, attemptA.TaskID, domain.StateRunning, domain.StateFailed,
		"stale attempt", attemptA.AttemptID, "STALE_EXECUTION_GENERATION")
	if !errors.Is(err, ErrAttemptLineageMismatch) {
		t.Fatalf("old attempt error = %v, want ErrAttemptLineageMismatch", err)
	}
	currentTask, _ := s.GetTask(ctx, attemptA.TaskID)
	currentAttempt, _ := s.GetTaskAttempt(ctx, second.AttemptID)
	if currentTask.State != domain.StateRunning || currentAttempt.EndedAt != nil {
		t.Fatalf("old attempt closed current execution: task=%+v attempt=%+v", currentTask, currentAttempt)
	}
}

func TestStore_AtomicTerminalTransitionAuditFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	attempt := setupRunningBoundAttempt(t, s, "task-audit-fail", "contract-audit-fail", "attempt-audit-fail")
	if _, err := s.db.ExecContext(ctx, `
CREATE TRIGGER reject_atomic_transition_audit
BEFORE INSERT ON audit_events
BEGIN
    SELECT RAISE(ABORT, 'forced atomic audit failure');
END;
`); err != nil {
		t.Fatalf("create audit rejection trigger: %v", err)
	}
	err := s.AtomicTerminalTransition(ctx, attempt.TaskID, domain.StateRunning, domain.StateFailed,
		"forced audit rollback", attempt.AttemptID, "STOP_CALL_OUTCOME_UNKNOWN")
	if err == nil {
		t.Fatal("AtomicTerminalTransition unexpectedly succeeded with rejected audit insert")
	}
	task, _ := s.GetTask(ctx, attempt.TaskID)
	open, _ := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if task.State != domain.StateRunning || open.EndedAt != nil || open.RecoveryDisposition != nil {
		t.Fatalf("audit failure did not roll back task and attempt: task=%+v attempt=%+v", task, open)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events").Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit chain changed after rejected insert: count=%d err=%v", count, err)
	}
}

func TestStore_AtomicAttemptClosureTransition(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	attempt := setupRunningBoundAttempt(t, s, "task-blocked", "contract-blocked", "attempt-blocked")
	if err := s.TransitionTask(ctx, attempt.TaskID, domain.StateRunning, domain.StateBlocked); err != nil {
		t.Fatalf("RUNNING -> BLOCKED: %v", err)
	}
	if err := s.AtomicAttemptClosureTransition(ctx, attempt.TaskID, attempt.AttemptID, domain.RecoveryAOBlockedEscalated); err != nil {
		t.Fatalf("AtomicAttemptClosureTransition: %v", err)
	}
	task, _ := s.GetTask(ctx, attempt.TaskID)
	closed, _ := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if task.State != domain.StateHumanRequired || closed.EndedAt == nil || closed.RecoveryDisposition == nil || *closed.RecoveryDisposition != domain.RecoveryAOBlockedEscalated {
		t.Fatalf("attempt closure mismatch: task=%+v attempt=%+v", task, closed)
	}
	events, err := s.ListAuditEvents(ctx, 0, 10)
	if err != nil || len(events) != 2 || events[1].Event.EventType != domain.AuditWorkerBlockedEscalated {
		t.Fatalf("blocked escalation audit mismatch: events=%+v err=%v", events, err)
	}
}

func TestStore_AtomicAttemptClosureCASConflictLeavesAttemptOpen(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	attempt := setupRunningBoundAttempt(t, s, "task-closure-cas", "contract-closure-cas", "attempt-closure-cas")
	err := s.AtomicAttemptClosureTransition(ctx, attempt.TaskID, attempt.AttemptID, domain.RecoveryAOBlockedEscalated)
	if !errors.Is(err, ErrStateConflict) {
		t.Fatalf("closure CAS error = %v, want ErrStateConflict", err)
	}
	task, _ := s.GetTask(ctx, attempt.TaskID)
	open, _ := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if task.State != domain.StateRunning || open.EndedAt != nil || open.RecoveryDisposition != nil {
		t.Fatalf("closure CAS conflict mutated state: task=%+v attempt=%+v", task, open)
	}
}
