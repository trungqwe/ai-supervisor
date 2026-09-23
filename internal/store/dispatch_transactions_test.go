package store

import (
	"context"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestDurableSendIntentConfirmationAndUnknownDelivery(t *testing.T) {
	ctx := context.Background()
	t.Run("HTTP 200 confirms send but not task running", func(t *testing.T) {
		s, _ := createTestStore(t)
		defer s.Close()
		pair, attempt := setupBoundAttempt(t, s, "task-send-confirm", "contract-send-confirm", "attempt-send-confirm")
		if err := s.RecordSendRequested(ctx, "dispatch-attempt-send-confirm", *attempt.SessionID, *attempt.TerminalGeneration, "idle", false, "supervisor", time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordSendConfirmed(ctx, "dispatch-attempt-send-confirm", "supervisor", true, time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordSendConfirmed(ctx, "dispatch-attempt-send-confirm", "supervisor", true, time.Now()); err == nil {
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
		if err := s.RecordSendConfirmed(ctx, "dispatch-attempt-send-unknown", "supervisor", true, time.Now()); err == nil {
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
		session, err := s.GetWorkerSessionByPair(ctx, pair)
		if err != nil || session.QuarantineState != domain.QuarantineQuarantined {
			t.Fatalf("session quarantine: %+v %v", session, err)
		}
	})
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
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_send_confirmation_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='TASK_STATE_TRANSITION' AND json_extract(NEW.details_json,'$.stage')='SEND_CONFIRMED' BEGIN SELECT RAISE(ABORT,'injected send confirmation audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSendConfirmed(ctx, opID, "supervisor", true, time.Now()); err == nil {
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
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_send_intent_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='TASK_STATE_TRANSITION' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); err != nil {
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
