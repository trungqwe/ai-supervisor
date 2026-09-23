package recovery

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

// HostQuiescence is a trusted host boundary, not a caller-supplied boolean.
// Acquire must close shared effect admission and join all prior permit owners
// across every process sharing the DB before returning an exclusive scope.
type HostQuiescence interface {
	Acquire(context.Context) (ExclusiveScope, error)
}
type ExclusiveScope interface{ Release() error }
type Observer interface {
	GetWorkerStatus(context.Context, string) (*ao.WorkerStatus, error)
}

type Runner struct {
	Store                *store.Store
	AO                   Observer
	Host                 HostQuiescence
	ActivityPollInterval time.Duration
	ExecutionDeadline    time.Duration
	Actor                string
	InvocationID         func() string
	Now                  func() time.Time
	gateMu               sync.Mutex
	running              bool
	ready                bool
	activePoller         *Poller
}

func (r *Runner) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

type Report struct {
	Complete   bool
	PendingAO  bool
	Classified int
}

var invocationSequence atomic.Uint64

func (r *Runner) Run(ctx context.Context) (report Report, err error) {
	if r == nil {
		return report, errors.New("recovery: Runner required")
	}
	if r.Store == nil || r.AO == nil || r.Host == nil || r.ActivityPollInterval <= 0 || r.ExecutionDeadline <= 0 || r.Actor == "" {
		r.invalidate()
		return report, errors.New("recovery: trusted host, Store, AO and injected policies required")
	}
	if err = ctx.Err(); err != nil {
		r.invalidate()
		return report, err
	}
	r.gateMu.Lock()
	if r.running {
		r.gateMu.Unlock()
		return report, errors.New("recovery: Run already active in this host instance")
	}
	if r.activePoller != nil {
		if e := r.activePoller.Stop(); e != nil {
			r.ready = false
			r.gateMu.Unlock()
			return report, e
		}
	}
	r.running = true
	r.ready = false
	r.gateMu.Unlock()
	defer func() { r.gateMu.Lock(); r.running = false; r.ready = err == nil && report.Complete; r.gateMu.Unlock() }()
	scope, err := r.Host.Acquire(ctx)
	if err != nil {
		return report, err
	}
	if scope == nil {
		return report, errors.New("recovery: host returned nil exclusive scope")
	}
	defer func() {
		if e := scope.Release(); e != nil {
			report.Complete = false
			err = errors.Join(err, e)
		}
	}()
	if err = ctx.Err(); err != nil {
		return report, err
	}
	id := fmt.Sprintf("startup-%d", invocationSequence.Add(1))
	if r.InvocationID != nil {
		id = r.InvocationID()
	}
	if id == "" {
		return report, errors.New("recovery: empty invocation ID")
	}
	if err = r.Store.RecordRecoverySweepEvent(ctx, id, r.Actor, "STARTUP_RECOVERY_SWEEP_STARTED", r.now()); err != nil {
		return report, err
	}
	snap, e := r.Store.ListRecoverySnapshot(ctx)
	if e != nil {
		return report, e
	}
	for _, op := range snap.RestoreIDs {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		if err = r.Store.ClassifyInterruptedRestore(ctx, op, r.Actor, r.now()); err != nil {
			if errors.Is(err, store.ErrStateConflict) {
				winner, e := r.Store.GetPairRestoreOperation(ctx, op)
				if e == nil && winner.ResolutionState != domain.RestoreInFlight {
					report.Classified++
					continue
				}
			}
			return report, err
		}
		report.Classified++
	}
	for _, op := range snap.ProvisionIDs {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		if err = r.Store.FailProvisioningAtStartup(ctx, op, r.Actor, r.now()); err != nil {
			if errors.Is(err, store.ErrStateConflict) {
				winner, e := r.Store.GetPairProvisioningOperation(ctx, op)
				if e == nil && winner.Stage != domain.ProvisionRequested {
					report.Classified++
					continue
				}
			}
			return report, err
		}
		report.Classified++
	}
	for _, x := range snap.Executions {
		if x.DispatchStage == "SEND_CONFIRMED" {
			continue
		}
		if err = ctx.Err(); err != nil {
			return report, err
		}
		if e = r.classifyExecution(ctx, x, id, &report); e != nil {
			if errors.Is(e, store.ErrStateConflict) {
				e = r.executionWinner(ctx, x, e, &report)
			}
			if e == nil {
				report.Classified++
				continue
			}
			return report, e
		}
		report.Classified++
	}
	for _, x := range snap.StopOperations {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		if e = r.classifyStop(ctx, x, id, &report); e != nil {
			if errors.Is(e, store.ErrStateConflict) {
				e = r.stopWinner(ctx, x, e)
			}
			if e == nil {
				report.Classified++
				continue
			}
			return report, e
		}
		report.Classified++
	}
	for _, x := range snap.Blocked {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		if e = r.Store.AtomicAttemptClosureTransition(ctx, x.TaskID, x.AttemptID, domain.RecoveryAOBlockedEscalated); e != nil {
			if errors.Is(e, store.ErrStateConflict) {
				task, te := r.Store.GetTask(ctx, x.TaskID)
				attempt, ae := r.Store.GetTaskAttempt(ctx, x.AttemptID)
				if te == nil && ae == nil && task.State == domain.StateHumanRequired && attempt.EndedAt != nil && attempt.RecoveryDisposition != nil && *attempt.RecoveryDisposition == domain.RecoveryAOBlockedEscalated {
					report.Classified++
					continue
				}
			}
			return report, e
		}
		report.Classified++
	}
	// Re-read after stop and blocked classification: those transactions may
	// close an attempt that appeared in the initial execution snapshot.
	latest, e := r.Store.ListRecoverySnapshot(ctx)
	if e != nil {
		return report, e
	}
	for _, x := range latest.Executions {
		if x.DispatchStage != "SEND_CONFIRMED" {
			continue
		}
		if err = ctx.Err(); err != nil {
			return report, err
		}
		if e = r.classifyExecution(ctx, x, id, &report); e != nil {
			if errors.Is(e, store.ErrStateConflict) {
				e = r.executionWinner(ctx, x, e, &report)
			}
			if e == nil {
				report.Classified++
				continue
			}
			return report, e
		}
		report.Classified++
	}
	if err = ctx.Err(); err != nil {
		return report, err
	}
	if err = r.Store.RecordRecoverySweepEvent(ctx, id, r.Actor, "STARTUP_RECOVERY_SWEEP_COMPLETED", r.now()); err != nil {
		return report, err
	}
	report.Complete = true
	return report, nil
}

func (r *Runner) invalidate() {
	r.gateMu.Lock()
	r.ready = false
	p := r.activePoller
	r.gateMu.Unlock()
	if p != nil {
		_ = p.Stop()
	}
}

func (r *Runner) executionWinner(ctx context.Context, x store.RecoveryExecution, original error, report *Report) error {
	op, e := r.Store.GetDispatchOperation(ctx, x.OperationID)
	if e != nil {
		return errors.Join(original, e)
	}
	if op.PairID != x.PairID || op.TaskID != x.TaskID || op.AttemptID != x.AttemptID || op.SessionID != x.SessionID || op.TerminalGeneration != x.Generation {
		return original
	}
	task, e := r.Store.GetTask(ctx, x.TaskID)
	if e != nil {
		return errors.Join(original, e)
	}
	attempt, e := r.Store.GetTaskAttempt(ctx, x.AttemptID)
	if e != nil {
		return errors.Join(original, e)
	}
	if attempt.ContractID != x.ContractID || attempt.SessionID == nil || *attempt.SessionID != x.SessionID || attempt.TerminalGeneration == nil || *attempt.TerminalGeneration != x.Generation {
		return original
	}
	changed := task.State != domain.TaskState(x.TaskState) || attempt.EndedAt != nil || op.Stage != domain.DispatchStage(x.DispatchStage)
	if (attempt.RecoveryDisposition == nil) != (!x.Disposition.Valid) {
		changed = true
	} else if attempt.RecoveryDisposition != nil && *attempt.RecoveryDisposition != x.Disposition.String {
		changed = true
	}
	if op.ResolutionState != nil && *op.ResolutionState == "DELIVERY_OUTCOME_UNKNOWN" {
		changed = true
	}
	if !changed {
		return original
	}
	if attempt.RecoveryDisposition != nil && *attempt.RecoveryDisposition == "RECOVERY_PENDING" {
		report.PendingAO = true
	}
	return nil
}

func (r *Runner) stopWinner(ctx context.Context, x store.RecoveryStop, original error) error {
	op, e := r.Store.GetStopOperation(ctx, x.OperationID)
	if e != nil {
		return errors.Join(original, e)
	}
	if op.SessionID != x.SessionID || op.TerminalGeneration != x.Generation || string(op.Purpose) != x.Purpose {
		return original
	}
	if op.ResolutionState == domain.StopResolutionInFlight && string(op.Stage) == x.Stage {
		return original
	}
	return nil
}

func (r *Runner) classifyExecution(ctx context.Context, x store.RecoveryExecution, id string, report *Report) error {
	if x.DispatchStage == "SEND_REQUESTED" {
		return r.Store.RecordUnknownDelivery(ctx, x.OperationID, r.Actor, r.now())
	}
	status, err := r.AO.GetWorkerStatus(ctx, x.SessionID)
	if err != nil {
		var protocol *ao.ProtocolError
		if errors.As(err, &protocol) {
			if x.DispatchStage == "DISPATCH_BOUND" {
				if x.Disposition.Valid && x.Disposition.String == "PRE_SEND_PROTOCOL_UNVERIFIED" {
					report.PendingAO = true
					return nil
				}
				if e := r.Store.RecordPreSendHold(ctx, x.OperationID, "PRE_SEND_PROTOCOL_UNVERIFIED", r.Actor, r.now()); e != nil {
					return e
				}
				report.PendingAO = true
				return nil
			}
			return err
		}
		if errors.Is(err, ao.ErrNotFound) {
			if x.DispatchStage == "DISPATCH_BOUND" {
				return r.Store.AtomicTerminalTransition(ctx, x.TaskID, domain.StateDispatched, domain.StateFailed, "AO session absent before send", x.AttemptID, "SESSION_ABSENT")
			}
			return r.Store.RecordRecoveryAbsent(ctx, x, r.Actor, r.now())
		}
		if x.DispatchStage == "DISPATCH_BOUND" {
			if x.Disposition.Valid && x.Disposition.String != "RECOVERY_PENDING" {
				report.PendingAO = true
				return nil
			}
			disposition := "RECOVERY_PENDING"
			var protocol *ao.ProtocolError
			if errors.As(err, &protocol) {
				disposition = "PRE_SEND_PROTOCOL_UNVERIFIED"
			}
			if e := r.Store.RecordPreSendHold(ctx, x.OperationID, disposition, r.Actor, r.now()); e != nil {
				return e
			}
		} else {
			if x.Disposition.Valid && x.Disposition.String != "RECOVERY_PENDING" {
				report.PendingAO = true
				return nil
			}
			if e := r.Store.RecordPostSendOutage(ctx, x, r.Actor, "AO_UNAVAILABLE", id, r.now()); e != nil {
				return e
			}
		}
		report.PendingAO = true
		return nil
	}
	if status == nil {
		return errors.New("recovery: nil AO status")
	}
	if x.DispatchStage == "DISPATCH_BOUND" {
		return r.classifyPreSend(ctx, x, status)
	}
	return r.Store.ObservePostSend(ctx, x, store.PostSendObservation{SessionID: status.ID, Generation: status.TerminalGeneration, Activity: string(status.Activity.State), Terminated: status.IsTerminated}, r.Actor, id, r.now())
}

func (r *Runner) classifyPreSend(ctx context.Context, x store.RecoveryExecution, status *ao.WorkerStatus) error {
	if status.ID != x.SessionID {
		return r.Store.AtomicTerminalTransition(ctx, x.TaskID, domain.StateDispatched, domain.StateFailed, "pre-send identity mismatch", x.AttemptID, "PRE_SEND_IDENTITY_MISMATCH")
	}
	if status.TerminalGeneration != x.Generation {
		return r.Store.AtomicTerminalTransition(ctx, x.TaskID, domain.StateDispatched, domain.StateFailed, "pre-send generation mismatch", x.AttemptID, "STALE_EXECUTION_GENERATION")
	}
	if status.IsTerminated {
		return r.Store.AtomicTerminalTransition(ctx, x.TaskID, domain.StateDispatched, domain.StateFailed, "pre-send worker terminated", x.AttemptID, "WORKER_TERMINATION_UNKNOWN")
	}
	if status.Activity.State != ao.ActivityStateIdle && status.Activity.State != ao.ActivityStateWaitingInput {
		return r.Store.RecordPreSendAdmissibilityRejected(ctx, x.OperationID, string(status.Activity.State), r.Actor, r.now())
	}
	// A pre-send RECOVERY_PENDING hold is cleared only by the dedicated exact
	// lineage API. A protocol hold needs operator authority and stays locked.
	if x.Disposition.Valid && x.Disposition.String == "RECOVERY_PENDING" {
		return r.Store.ResolvePreSendHold(ctx, x.OperationID, x.AttemptID, "RECOVERY_PENDING", x.SessionID, x.Generation, string(status.Activity.State), "", r.Actor, r.now())
	}
	return nil
}

func (r *Runner) classifyStop(ctx context.Context, x store.RecoveryStop, id string, report *Report) error {
	status, err := r.AO.GetWorkerStatus(ctx, x.SessionID)
	if err != nil {
		if errors.Is(err, ao.ErrNotFound) {
			return r.Store.CommitStopOutcome(ctx, x.OperationID, store.StopTerminalOutcome{ExpectedStage: domain.StopStage(x.Stage), Stage: domain.StopTargetAbsent, Resolution: domain.StopResolutionTargetAbsent, At: r.now()})
		}
		if x.Stage == "STOP_CALL_SUCCEEDED" {
			stop, e := r.Store.GetStopOperation(ctx, x.OperationID)
			if e != nil {
				return e
			}
			if stop.ConfirmationDeadlineAt == nil {
				return errors.New("recovery: confirmed call lacks persisted deadline")
			}
			if !r.now().Before(*stop.ConfirmationDeadlineAt) {
				return r.Store.CommitStopOutcome(ctx, x.OperationID, store.StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopCallSucceeded, Resolution: domain.StopResolutionConfirmationTimeout, At: r.now()})
			}
		}
		report.PendingAO = true
		return nil
	}
	if status == nil || status.ID != x.SessionID {
		return errors.New("recovery: stop observation identity invalid")
	}
	if status.TerminalGeneration != x.Generation {
		return r.Store.CommitStopOutcome(ctx, x.OperationID, store.StopTerminalOutcome{ExpectedStage: domain.StopStage(x.Stage), Stage: domain.StopStage(x.Stage), Resolution: domain.StopResolutionGenerationMismatch, At: r.now(), ObservedSessionID: status.ID, ObservedGeneration: status.TerminalGeneration, ObservedIsTerminated: status.IsTerminated})
	}
	if x.Stage == "STOP_REQUESTED" {
		if status.IsTerminated {
			return r.Store.CommitStopOutcome(ctx, x.OperationID, store.StopTerminalOutcome{ExpectedStage: domain.StopRequested, Stage: domain.StopRequested, Resolution: domain.StopResolutionEffectUnprovenAlreadyTerminated, At: r.now(), ObservedSessionID: status.ID, ObservedGeneration: status.TerminalGeneration, ObservedIsTerminated: true})
		}
		return r.Store.ClassifyAbandonedStopAlive(ctx, x.OperationID, status.ID, status.TerminalGeneration, r.Actor, id, r.now())
	}
	stop, e := r.Store.GetStopOperation(ctx, x.OperationID)
	if e != nil {
		return e
	}
	now := r.now()
	if status.IsTerminated && stop.ConfirmationDeadlineAt != nil && now.Before(*stop.ConfirmationDeadlineAt) {
		return r.Store.CommitStopOutcome(ctx, x.OperationID, store.StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopTerminationConfirmed, Resolution: domain.StopResolutionTerminationConfirmed, At: now, ObservedSessionID: status.ID, ObservedGeneration: status.TerminalGeneration, ObservedIsTerminated: true})
	}
	if stop.ConfirmationDeadlineAt != nil && now.Before(*stop.ConfirmationDeadlineAt) {
		report.PendingAO = true
		return nil
	}
	return r.Store.CommitStopOutcome(ctx, x.OperationID, store.StopTerminalOutcome{ExpectedStage: domain.StopCallSucceeded, Stage: domain.StopCallSucceeded, Resolution: domain.StopResolutionConfirmationTimeout, At: now, ObservedSessionID: status.ID, ObservedGeneration: status.TerminalGeneration, ObservedIsTerminated: status.IsTerminated})
}
