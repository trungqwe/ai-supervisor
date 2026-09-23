package store

import (
	"context"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"testing"
	"time"
)

func TestStartupRestoreClassificationAtomicReplayAndConfirmedProvenance(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-startup-restore", "contract-startup-restore")
	pair := "pair-d-task-startup-restore"
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pair, SessionID: "session-startup-restore", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "gen-old", QuarantineState: domain.QuarantineClean})
	req := domain.RestoreReservation{AuthorizationID: "auth-startup-restore", OperationID: "restore-startup", PairID: pair, SessionID: "session-startup-restore", ExpectedGeneration: "gen-old", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	var version int64
	if err := s.db.QueryRowContext(ctx, `SELECT version FROM pair_restore_operations WHERE operation_id=?`, req.OperationID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_startup_restore_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_RESTORE_OUTCOME_UNKNOWN' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", time.Now()); err == nil {
		t.Fatal("audit failure did not roll back restore classification")
	}
	op, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.ResolutionState != domain.RestoreInFlight || op.Version != version {
		t.Fatalf("partial restore classification: %+v %v", op, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER reject_startup_restore_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	op, err = s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil || op.ResolutionState != domain.RestoreOutcomeUnknown || op.Version != version+1 {
		t.Fatalf("unknown restore state: %+v %v", op, err)
	}
	ws, err := s.GetWorkerSessionByPair(ctx, pair)
	if err != nil || ws.QuarantineState != domain.QuarantineQuarantined {
		t.Fatalf("session quarantine: %+v %v", ws, err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_RESTORE_OUTCOME_UNKNOWN' AND json_extract(details_json,'$.operation_id')=?`, req.OperationID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("replay audit count=%d err=%v", events, err)
	}
}

func TestStartupProvisioningFailureAtomicReplay(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	pair := "pair-startup-provision"
	setupProvisioningPair(t, s, pair)
	op := domain.PairProvisioningOperation{OperationID: "provision-startup", PairID: pair, ClientToken: "token-startup"}
	if err := s.ReservePairProvisioning(ctx, op, "supervisor"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER reject_startup_provision_audit BEFORE INSERT ON audit_events WHEN NEW.event_type='PAIR_SESSION_PROVISION_FAILED' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.FailProvisioningAtStartup(ctx, op.OperationID, "supervisor", time.Now()); err == nil {
		t.Fatal("audit failure did not roll back provisioning")
	}
	got, err := s.GetPairProvisioningOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.ProvisionRequested {
		t.Fatalf("partial provisioning classification: %+v %v", got, err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TRIGGER reject_startup_provision_audit`); err != nil {
		t.Fatal(err)
	}
	if err := s.FailProvisioningAtStartup(ctx, op.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetPairProvisioningOperation(ctx, op.OperationID)
	if err != nil || got.Stage != domain.ProvisionFailed {
		t.Fatalf("failed provisioning state: %+v %v", got, err)
	}
	var notes string
	if err := s.db.QueryRowContext(ctx, `SELECT resolution_notes FROM pair_provisioning_operations WHERE operation_id=?`, op.OperationID).Scan(&notes); err != nil || notes != "STARTUP_RECOVERY_SWEEP" {
		t.Fatalf("resolution notes=%q err=%v", notes, err)
	}
	if err := s.FailProvisioningAtStartup(ctx, op.OperationID, "supervisor", time.Now()); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_SESSION_PROVISION_FAILED' AND json_extract(details_json,'$.operation_id')=?`, op.OperationID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("replay audit count=%d err=%v", events, err)
	}
}

func TestStartupConfirmedRestoreRetainsHTTP200(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()
	setupReadyTask(t, s, "task-startup-confirmed", "contract-startup-confirmed")
	pair := "pair-d-task-startup-confirmed"
	seedWorkerSessionForTest(t, s, domain.WorkerSession{PairID: pair, SessionID: "session-startup-confirmed", RuntimeType: "agy_tui", WorkerAgentID: "agy", Status: domain.WorkerSessionTerminated, TerminalGeneration: "old", QuarantineState: domain.QuarantineClean})
	req := domain.RestoreReservation{AuthorizationID: "auth-startup-confirmed", OperationID: "restore-startup-confirmed", PairID: pair, SessionID: "session-startup-confirmed", ExpectedGeneration: "old", RiskScope: "POSSIBLE_PROMPT_REPLAY", AuthorizedPrincipal: "verified-subject", Actor: "supervisor"}
	if err := s.ReservePairRestore(ctx, req); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	if err := s.ConfirmPairRestore(ctx, req.OperationID, req.SessionID, req.ExpectedGeneration, "new", "native", domain.WorkerSessionIdle, "supervisor", at); err != nil {
		t.Fatal(err)
	}
	before, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ClassifyInterruptedRestore(ctx, req.OperationID, "supervisor", at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	after, err := s.GetPairRestoreOperation(ctx, req.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Stage != after.Stage || before.ResolutionState != after.ResolutionState || before.Version != after.Version || after.Stage != "RESTORE_CONFIRMED" || after.ResolutionState != domain.RestoreInFlight || after.HTTP200At == nil {
		t.Fatalf("confirmed restore provenance changed: before=%+v after=%+v", before, after)
	}
	var events int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_type='PAIR_RESTORE_OUTCOME_UNKNOWN' AND json_extract(details_json,'$.operation_id')=?`, req.OperationID).Scan(&events); err != nil || events != 0 {
		t.Fatalf("confirmed restore misclassified: count=%d err=%v", events, err)
	}
}
