package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestSendConfirmationBindsImmutableExecutionBudget(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-budget", "contract-budget", "attempt-budget")
	opID := "dispatch-attempt-budget"
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	origin := time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)
	policy := domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "policy-v1"}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, origin); err == nil {
		t.Fatal("missing policy accepted")
	}
	if _, err := s.GetExecutionBudget(ctx, attempt.AttemptID); err == nil {
		t.Fatal("budget before confirmation")
	}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, origin, policy); err != nil {
		t.Fatal(err)
	}
	budget, err := s.GetExecutionBudget(ctx, attempt.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if !budget.OriginAt.Equal(origin) || !budget.DeadlineAt.Equal(origin.Add(time.Hour)) || budget.PolicyRef != "policy-v1" || budget.BindingBasis != "SEND_CONFIRMATION_ATOMIC" {
		t.Fatalf("budget=%+v", budget)
	}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, origin, domain.ExecutionBudgetPolicy{Duration: 2 * time.Hour, PolicyRef: "policy-v2"}); err == nil {
		t.Fatal("policy change after confirmation")
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE attempt_execution_budgets SET deadline_at=? WHERE attempt_id=?`, formatTime(origin.Add(2*time.Hour)), attempt.AttemptID); err == nil {
		t.Fatal("mutable deadline")
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE dispatch_operations SET confirmed_at=? WHERE operation_id=?`, formatTime(origin.Add(time.Second)), opID); err == nil {
		t.Fatal("mutable confirmed_at")
	}
	assertAuditEventTypes(t, s, map[string]int{domain.AuditDispatchSendConfirmed: 1}, opID)
}

func TestSendConfirmationAuditFailureRollsBackBudget(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-budget-rollback", "contract-budget-rollback", "attempt-budget-rollback")
	opID := "dispatch-attempt-budget-rollback"
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_budget_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='DISPATCH_SEND_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected budget audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "policy-v1"})
	if err == nil || !strings.Contains(err.Error(), "injected budget audit failure") {
		t.Fatalf("audit failure=%v", err)
	}
	if _, err := s.GetExecutionBudget(ctx, attempt.AttemptID); err == nil {
		t.Fatal("partial budget")
	}
	op, err := s.GetDispatchOperation(ctx, opID)
	if err != nil || op.Stage != domain.SendRequested {
		t.Fatalf("partial confirmation: %+v %v", op, err)
	}
}

func TestLegacyBudgetBindingUsesHistoricalConfirmationAndAudit(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-legacy-budget", "contract-legacy-budget", "attempt-legacy-budget")
	opID := "dispatch-attempt-legacy-budget"
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	// A v4 row predates the v5 trigger. Construct its exact on-disk shape.
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER trg_dispatch_requires_execution_budget`); err != nil {
		t.Fatal(err)
	}
	origin := time.Date(2026, 3, 4, 5, 6, 7, 987654321, time.UTC)
	if _, err := s.db.ExecContext(ctx, `UPDATE dispatch_operations SET stage='SEND_CONFIRMED',confirmed_at=? WHERE operation_id=?`, formatTime(origin), opID); err != nil {
		t.Fatal(err)
	}
	policy := domain.ExecutionBudgetPolicy{Duration: 90 * time.Minute, PolicyRef: "historical-p1"}
	if err := s.BindLegacyExecutionBudget(ctx, attempt.AttemptID, "operator", "LEGACY_EXECUTION_BUDGET_RECONCILIATION", "policy-record-1", policy, origin.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	b, err := s.GetExecutionBudget(ctx, attempt.AttemptID)
	if err != nil || !b.OriginAt.Equal(origin) || !b.DeadlineAt.Equal(origin.Add(policy.Duration)) || b.BindingBasis != "LEGACY_OPERATOR_VERIFIED" {
		t.Fatalf("budget=%+v err=%v", b, err)
	}
	if err := s.BindLegacyExecutionBudget(ctx, attempt.AttemptID, "operator", "LEGACY_EXECUTION_BUDGET_RECONCILIATION", "policy-record-1", policy, origin.Add(time.Hour)); err != nil {
		t.Fatalf("exact replay=%v", err)
	}
	if err := s.BindLegacyExecutionBudget(ctx, attempt.AttemptID, "operator", "LEGACY_EXECUTION_BUDGET_RECONCILIATION", "policy-record-2", policy, origin.Add(time.Hour)); err == nil {
		t.Fatal("changed evidence accepted")
	}
	assertAuditEventTypes(t, s, map[string]int{domain.AuditExecutionBudgetLegacyBound: 1}, opID)
}

func TestDurableSendIntentConfirmationAndUnknownDelivery(t *testing.T) {
	ctx := context.Background()
	t.Run("HTTP 200 confirms send but not task running", func(t *testing.T) {
		s, _ := createTestStore(t)
		defer s.Close()
		pair, attempt := setupBoundAttempt(t, s, "task-send-confirm", "contract-send-confirm", "attempt-send-confirm")
		if err := s.RecordSendRequested(ctx, "dispatch-attempt-send-confirm", *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordSendConfirmed(ctx, "dispatch-attempt-send-confirm", "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordSendConfirmed(ctx, "dispatch-attempt-send-confirm", "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err == nil {
			t.Fatal("duplicate send confirmation unexpectedly succeeded")
		}
		task, err := s.GetTask(ctx, attempt.TaskID)
		if err != nil || task.State != domain.StateDispatched {
			t.Fatalf("HTTP 200 must leave Task DISPATCHED: %+v %v", task, err)
		}
		op, err := s.GetDispatchOperation(ctx, "dispatch-attempt-send-confirm")
		if err != nil || op.Stage != domain.SendConfirmed {
			t.Fatalf("dispatch confirmation: %+v %v", op, err)
		}
		assertAuditEventTypes(t, s, map[string]int{domain.AuditDispatchSendRequested: 1, domain.AuditDispatchSendConfirmed: 1}, "dispatch-attempt-send-confirm")
		_ = pair
	})
	t.Run("unknown delivery is terminal and stale confirmation loses", func(t *testing.T) {
		s, _ := createTestStore(t)
		defer s.Close()
		pair, attempt := setupBoundAttempt(t, s, "task-send-unknown", "contract-send-unknown", "attempt-send-unknown")
		if err := s.RecordSendRequested(ctx, "dispatch-attempt-send-unknown", *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordUnknownDelivery(ctx, "dispatch-attempt-send-unknown", "supervisor", time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordSendConfirmed(ctx, "dispatch-attempt-send-unknown", "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err == nil {
			t.Fatal("late HTTP 200 overwrote unknown delivery")
		}
		op, err := s.GetDispatchOperation(ctx, "dispatch-attempt-send-unknown")
		if err != nil || op.Stage != domain.SendRequested || op.ResolutionState == nil || *op.ResolutionState != domain.RecoveryDeliveryOutcomeUnknown {
			t.Fatalf("unknown operation state: %+v %v", op, err)
		}
		task, err := s.GetTask(ctx, attempt.TaskID)
		if err != nil || task.State != domain.StateHumanRequired {
			t.Fatalf("D12 failed-to-human-required Task state: %+v %v", task, err)
		}
		got, err := s.GetTaskAttempt(ctx, attempt.AttemptID)
		if err != nil || got.EndedAt == nil || got.RecoveryDisposition == nil || *got.RecoveryDisposition != domain.RecoveryUncertainDeliveryCrash || got.QuarantineState != domain.QuarantineQuarantined {
			t.Fatalf("closed quarantined attempt: %+v %v", got, err)
		}
		var transitions int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE task_id=? AND event_type='TASK_STATE_TRANSITION' AND (json_extract(details_json,'$.from_state')='DISPATCHED' AND json_extract(details_json,'$.to_state')='FAILED' OR json_extract(details_json,'$.from_state')='FAILED' AND json_extract(details_json,'$.to_state')='HUMAN_REQUIRED')`, attempt.TaskID).Scan(&transitions); err != nil || transitions != 2 {
			t.Fatalf("D12 transition audit count=%d err=%v, want both legal edges", transitions, err)
		}
		assertAuditEventTypes(t, s, map[string]int{domain.AuditDispatchSendRequested: 1, domain.AuditUncertainDeliveryQuarantine: 1}, "dispatch-attempt-send-unknown")
		session, err := s.GetWorkerSessionByPair(ctx, pair)
		if err != nil || session.QuarantineState != domain.QuarantineQuarantined {
			t.Fatalf("session quarantine: %+v %v", session, err)
		}
	})
}

func assertAuditEventTypes(t *testing.T, s *Store, want map[string]int, operationID string) {
	t.Helper()
	events, err := s.ListAuditEvents(context.Background(), 0, MaxAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, event := range events {
		id, _ := event.Event.Details["dispatch_operation_id"].(string)
		if id == operationID {
			got[event.Event.EventType]++
		}
	}
	for eventType, count := range want {
		if got[eventType] != count {
			t.Errorf("event_type %q count=%d, want %d; got=%v", eventType, got[eventType], count, got)
		}
	}
}

func TestSendConfirmationAuditFailureLeavesSendIntent(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-send-confirm-audit", "contract-send-confirm-audit", "attempt-send-confirm-audit")
	opID := "dispatch-attempt-send-confirm-audit"
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_send_confirmation_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='DISPATCH_SEND_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected send confirmation audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, time.Now(), domain.ExecutionBudgetPolicy{Duration: time.Hour, PolicyRef: "fixture-policy"}); err == nil {
		t.Fatal("confirmation survived audit failure")
	}
	op, err := s.GetDispatchOperation(ctx, opID)
	if err != nil || op.Stage != domain.SendRequested || op.ConfirmedAt != nil {
		t.Fatalf("confirmation partially committed: %+v %v", op, err)
	}
}

func TestPreSendProtocolHoldCannotBeDowngradedAndRequiresAuthority(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pair, attempt := setupBoundAttempt(t, s, "task-presend-hold", "contract-presend-hold", "attempt-presend-hold")
	opID := "dispatch-attempt-presend-hold"
	if err := s.RecordPreSendHold(ctx, opID, "RECOVERY_PENDING", "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPreSendHold(ctx, opID, "PRE_SEND_PROTOCOL_UNVERIFIED", "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPreSendHold(ctx, opID, "RECOVERY_PENDING", "supervisor", time.Now()); err == nil {
		t.Fatal("timeout downgraded protocol hold")
	}
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err == nil {
		t.Fatal("send accepted while protocol hold unresolved")
	}
	if err := s.ResolvePreSendHold(ctx, opID, attempt.AttemptID, "PRE_SEND_PROTOCOL_UNVERIFIED", *attempt.SessionID, *attempt.TerminalGeneration, "idle", "", "supervisor", time.Now()); err == nil {
		t.Fatal("successful GET without operator authority cleared protocol hold")
	}
	stillHeld, err := s.GetTaskAttempt(ctx, attempt.AttemptID)
	if err != nil || stillHeld.RecoveryDisposition == nil || *stillHeld.RecoveryDisposition != "PRE_SEND_PROTOCOL_UNVERIFIED" {
		t.Fatalf("unauthorized GET changed protocol hold: %+v %v", stillHeld, err)
	}
	if err := s.ResolvePreSendHold(ctx, "dispatch-attempt-presend-hold", attempt.AttemptID, "PRE_SEND_PROTOCOL_UNVERIFIED", *attempt.SessionID, *attempt.TerminalGeneration, "idle", "verified-subject", "supervisor", time.Now()); err != nil {
		t.Fatalf("authorized exact-lineage recovery: %v", err)
	}
	if err := s.RecordSendRequested(ctx, opID, *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
		t.Fatalf("send after explicit hold resolution: %v", err)
	}
	if got, err := s.GetWorkerSessionByPair(ctx, pair); err != nil || got.QuarantineState != domain.QuarantineClean {
		t.Fatalf("same-lineage hold clear failed: %+v %v", got, err)
	}
}

func TestSendIntentAuditFailureLeavesBoundState(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	_, attempt := setupBoundAttempt(t, s, "task-send-audit-fail", "contract-send-audit-fail", "attempt-send-audit-fail")
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_send_intent_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='DISPATCH_SEND_REQUESTED' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendRequested(ctx, "dispatch-attempt-send-audit-fail", *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err == nil {
		t.Fatal("send intent unexpectedly survived audit failure")
	}
	op, err := s.GetDispatchOperation(ctx, "dispatch-attempt-send-audit-fail")
	if err != nil || op.Stage != domain.DispatchBound {
		t.Fatalf("send intent partial commit: %+v %v", op, err)
	}
	task, err := s.GetTask(ctx, attempt.TaskID)
	if err != nil || task.State != domain.StateDispatched {
		t.Fatalf("task state changed on audit rollback: %+v %v", task, err)
	}
}
