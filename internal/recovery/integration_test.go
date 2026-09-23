package recovery

import (
	"context"
	"errors"
	"testing"
	"time"
)

// This harness tests a library call contract only. It does not prove daemon
// bootstrap order, real host quiescence or operator authentication.
func simulatedHostServe(ctx context.Context, r *Runner, serve func()) (Report, error) {
	report, err := r.Run(ctx)
	if err == nil && report.Complete {
		serve()
	}
	return report, err
}

func TestSimulatedHostServeContract(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		s := newRecoveryStore(t)
		served := 0
		r := &Runner{Store: s, AO: &testObserver{}, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
		report, err := simulatedHostServe(context.Background(), r, func() { served++ })
		if err != nil || !report.Complete || report.PendingAO || served != 1 {
			t.Fatalf("complete: %+v %v served=%d", report, err, served)
		}
	})
	t.Run("classified pending AO", func(t *testing.T) {
		s := newRecoveryStore(t)
		session, generation, attempt := seedBoundExecution(t, s, "pendingserve")
		if err := s.RecordSendRequested(context.Background(), "dispatch-pendingserve", session, generation, "idle", false, "fixture", time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := s.RecordSendConfirmed(context.Background(), "dispatch-pendingserve", "fixture", true, time.Now()); err != nil {
			t.Fatal(err)
		}
		served := 0
		r := &Runner{Store: s, AO: &testObserver{err: errors.New("AO unavailable")}, Host: &testHost{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
		report, err := simulatedHostServe(context.Background(), r, func() { served++ })
		if err != nil || !report.Complete || !report.PendingAO || served != 1 {
			t.Fatalf("pending: %+v %v served=%d", report, err, served)
		}
		a, _ := s.GetTaskAttempt(context.Background(), attempt)
		if a.RecoveryDisposition == nil || *a.RecoveryDisposition != "RECOVERY_PENDING" {
			t.Fatalf("pending Pair hold: %+v", a)
		}
		if err := s.RecordSendRequested(context.Background(), "dispatch-pendingserve", session, generation, "idle", false, "supervisor", time.Now()); err == nil {
			t.Fatal("pending Pair admitted send")
		}
	})
	t.Run("incomplete", func(t *testing.T) {
		s := newRecoveryStore(t)
		served := 0
		r := &Runner{Store: s, AO: &testObserver{}, ActivityPollInterval: time.Second, ExecutionDeadline: time.Minute, Actor: "supervisor"}
		report, err := simulatedHostServe(context.Background(), r, func() { served++ })
		if err == nil || report.Complete || served != 0 {
			t.Fatalf("incomplete: %+v %v served=%d", report, err, served)
		}
	})
}
