package recovery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/stop"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

// TimeoutMonitor is a post-startup one-shot evaluator. The trusted host owns
// its scheduling and effect admission; startup Run never invokes Tick.
type TimeoutMonitor struct {
	Owner    *Runner
	Stop     *stop.Coordinator
	Interval time.Duration // injected scheduling policy; no default
	Actor    string
	Now      func() time.Time
}

func (m *TimeoutMonitor) now() time.Time {
	if m.Now != nil {
		return m.Now().UTC()
	}
	return time.Now().UTC()
}

func timeoutOperationID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "timeout-" + hex.EncodeToString(b[:]), nil
}

func (m *TimeoutMonitor) Tick(ctx context.Context) error {
	if m == nil || m.Owner == nil || m.Stop == nil || m.Stop.Store == nil || m.Stop.AO == nil || m.Owner.Store != m.Stop.Store || m.Owner.Host == nil || m.Interval <= 0 || m.Actor == "" {
		return errors.New("recovery: timeout monitor dependencies, trusted host and injected cadence required")
	}
	admission, ok := m.Owner.Host.(stop.TimeoutAdmission)
	if !ok {
		return errors.New("recovery: shared host timeout admission unavailable")
	}
	m.Owner.pollAccess.RLock()
	defer m.Owner.pollAccess.RUnlock()
	m.Owner.gateMu.Lock()
	ready := m.Owner.ready && !m.Owner.running
	m.Owner.gateMu.Unlock()
	if !ready {
		return errors.New("recovery: timeout monitor requires completed startup classification")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	snapshot, err := m.Owner.Store.ListRecoverySnapshot(ctx)
	if err != nil {
		return err
	}
	owned := make(map[string]bool, len(snapshot.StopOwnedAttemptIDs))
	for _, id := range snapshot.StopOwnedAttemptIDs {
		owned[id] = true
	}
	for _, x := range snapshot.Executions {
		if err = ctx.Err(); err != nil {
			return err
		}
		if x.TaskState != "RUNNING" || x.DispatchStage != "SEND_CONFIRMED" || owned[x.AttemptID] || x.Disposition.Valid {
			continue
		}
		budget, e := m.Owner.Store.GetExecutionBudget(ctx, x.AttemptID)
		if e != nil {
			return fmt.Errorf("recovery: budget for %s unavailable: %w", x.AttemptID, e)
		}
		if m.now().Before(budget.DeadlineAt) {
			continue
		}
		id, e := timeoutOperationID()
		if e != nil {
			return e
		}
		task, contract, attempt := x.TaskID, x.ContractID, x.AttemptID
		op := domain.StopOperation{OperationID: id, Purpose: domain.RunningAttemptStop, PairID: x.PairID, TaskID: &task, ContractID: &contract, AttemptID: &attempt, SessionID: x.SessionID, TerminalGeneration: x.Generation, Actor: m.Actor}
		coordinator := *m.Stop
		coordinator.TimeoutHost = admission
		coordinator.Now = m.Now
		if e = coordinator.StartTimeout(ctx, op); e != nil {
			if errors.Is(e, stop.ErrTimeoutPreflightUnavailable) || errors.Is(e, stop.ErrTimeoutPreflightInadmissible) {
				// No Tx R exists yet. A fresh observation takes the existing
				// poller lifecycle path; it never converts DISPATCHED to a
				// timeout stop or derives an effect permit from persistence.
				observer := Runner{Store: m.Owner.Store, AO: m.Stop.AO, Handoff: m.Owner.Handoff, Actor: m.Actor}
				var report Report
				if observeErr := observer.classifyExecution(ctx, x, "timeout-monitor", &report); observeErr != nil {
					if errors.Is(observeErr, store.ErrStateConflict) {
						observeErr = observer.executionWinner(ctx, x, observeErr, &report)
					}
					if observeErr != nil {
						return errors.Join(e, observeErr)
					}
				}
				goto next
			}
			// A competing reservation is a durable winner. Re-read rather than
			// reconstruct an effect permit or retry /kill.
			if errors.Is(e, store.ErrStateConflict) || errors.Is(e, store.ErrAttemptLineageMismatch) {
				fresh, readErr := m.Owner.Store.ListRecoverySnapshot(ctx)
				if readErr != nil {
					return errors.Join(e, readErr)
				}
				for _, winner := range fresh.StopOwnedAttemptIDs {
					if winner == x.AttemptID {
						goto next
					}
				}
				currentAttempt, attemptErr := m.Owner.Store.GetTaskAttempt(ctx, x.AttemptID)
				currentTask, taskErr := m.Owner.Store.GetTask(ctx, x.TaskID)
				if attemptErr == nil && taskErr == nil && currentAttempt.EndedAt != nil && (currentTask.State == domain.StateFailed || currentTask.State == domain.StateHumanRequired) {
					goto next
				}
			}
			return e
		}
	next:
	}
	return nil
}
