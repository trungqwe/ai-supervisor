package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func setupProvisioningPair(t *testing.T, s *Store, pairID string) {
	t.Helper()
	ctx := context.Background()
	projectID := "project-" + pairID
	if err := s.CreateProject(ctx, domain.Project{ProjectID: projectID, Name: projectID, RootPath: "/" + projectID}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePair(ctx, domain.Pair{PairID: pairID, ProjectID: projectID, CurrentPhaseID: "P03", State: "ACTIVE"}); err != nil {
		t.Fatal(err)
	}
}

func TestProvisioningReservationAuditFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupProvisioningPair(t, s, "pair-provision-audit")
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_provision_request_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='TASK_STATE_TRANSITION' BEGIN SELECT RAISE(ABORT,'injected provisioning audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	err := s.ReservePairProvisioning(ctx, domain.PairProvisioningOperation{OperationID: "provision-audit", PairID: "pair-provision-audit", ClientToken: "token-audit"}, "supervisor")
	if err == nil {
		t.Fatal("reservation survived audit failure")
	}
	var operations, sessions int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pair_provisioning_operations WHERE pair_id='pair-provision-audit'`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM worker_sessions WHERE pair_id='pair-provision-audit'`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if operations != 0 || sessions != 0 {
		t.Fatalf("partial reservation: operations=%d sessions=%d", operations, sessions)
	}
}

func TestProvisioningConfirmationAuditFailureLeavesIntentAndNoSession(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupProvisioningPair(t, s, "pair-provision-confirm-audit")
	op := domain.PairProvisioningOperation{OperationID: "provision-confirm-audit", PairID: "pair-provision-confirm-audit", ClientToken: "token-confirm-audit"}
	if err := s.ReservePairProvisioning(ctx, op, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_provision_confirmation_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='TASK_STATE_TRANSITION' BEGIN SELECT RAISE(ABORT,'injected provisioning confirmation audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	err := s.ConfirmPairProvisioning(ctx, op.OperationID, domain.WorkerSession{PairID: op.PairID, SessionID: "session-provision-confirm-audit", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionIdle, TerminalGeneration: "generation"}, "supervisor", time.Now().UTC())
	if err == nil {
		t.Fatal("confirmation survived audit failure")
	}
	got, err := s.GetPairProvisioningOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.ProvisionRequested {
		t.Fatalf("intent was not preserved after failed confirmation: %+v %v", got, err)
	}
	var sessions int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM worker_sessions WHERE pair_id=?`, op.PairID).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("partial WorkerSession insert: sessions=%d err=%v", sessions, err)
	}
}

func TestGenericProvisioningUpdateAPIIsClosed(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupProvisioningPair(t, s, "pair-provision-api-closed")
	op := domain.PairProvisioningOperation{OperationID: "provision-api-closed", PairID: "pair-provision-api-closed", ClientToken: "token-api-closed"}
	if err := s.CreatePairProvisioningOperation(ctx, op); !errors.Is(err, ErrAtomicDispatchRequired) {
		t.Fatalf("generic create API error=%v, want ErrAtomicDispatchRequired", err)
	}
	if err := s.UpdatePairProvisioningOperationStage(ctx, op.OperationID, domain.ProvisionRequested, domain.ProvisionConfirmed, PairProvisioningStageUpdate{}); !errors.Is(err, ErrAtomicDispatchRequired) {
		t.Fatalf("generic stage update error=%v, want ErrAtomicDispatchRequired", err)
	}
}
