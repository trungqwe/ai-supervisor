package stop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

// AO is the pinned session-scoped interface. No generation is sent on /kill.
type AO interface {
	GetWorkerStatus(context.Context, string) (*ao.WorkerStatus, error)
	StopWorker(context.Context, string) (*ao.StopWorkerResult, error)
}

// OperatorBoundary must be backed by a host-authenticated principal. A test
// implementation proves only the library's fail-closed use of this interface.
type OperatorBoundary interface {
	VerifiedRestorePrincipal(context.Context, string, string, string, string) (string, bool, error)
}

// TimeoutAdmission is the same trusted host admission boundary drained by
// startup quiescence. A fake implementation proves only library ordering.
type TimeoutAdmission interface {
	AcquireEffect(context.Context, string, string) (TimeoutPermit, error)
}
type TimeoutPermit interface{ Release() error }

var ErrTimeoutPreflightUnavailable = errors.New("stop: timeout preflight unavailable")
var ErrTimeoutPreflightInadmissible = errors.New("stop: timeout preflight not admissible")

type Coordinator struct {
	Store       *store.Store
	AO          AO
	Operator    OperatorBoundary
	KillTimeout time.Duration // injected SUPERVISOR_KILL_STOP_TIMEOUT; no default
	TimeoutHost TimeoutAdmission
	Now         func() time.Time // trusted timeout monitor clock; never request-supplied
}

// Start owns the one-use effect opportunity for an unambiguously committed
// new intent. A pre-existing STOP_REQUESTED is never an execution permit.
func (c *Coordinator) Start(ctx context.Context, stop domain.StopOperation) error {
	if stop.InitiatingFailureReason != nil {
		return errors.New("stop: timeout cause requires guarded StartTimeout")
	}
	return c.start(ctx, stop, false)
}

// StartTimeout owns a fresh one-use effect opportunity. It never reconstructs
// a permit from a persisted STOP_REQUESTED row after crash or ambiguity.
func (c *Coordinator) StartTimeout(ctx context.Context, stop domain.StopOperation) (err error) {
	if c == nil || c.TimeoutHost == nil || c.AO == nil || c.Store == nil || stop.Purpose != domain.RunningAttemptStop || stop.AttemptID == nil {
		return errors.New("stop: trusted timeout admission and exact live attempt required")
	}
	permit, err := c.TimeoutHost.AcquireEffect(ctx, stop.PairID, "TIMEOUT_MONITOR")
	if err != nil {
		return err
	}
	if permit == nil {
		return errors.New("stop: timeout permit missing")
	}
	defer func() { err = errors.Join(err, permit.Release()) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	pre, err := c.AO.GetWorkerStatus(ctx, stop.SessionID)
	if err != nil {
		return fmt.Errorf("%w; no intent: %w", ErrTimeoutPreflightUnavailable, err)
	}
	if pre == nil || pre.ID != stop.SessionID || pre.TerminalGeneration != stop.TerminalGeneration || pre.IsTerminated || (pre.Activity.State != ao.ActivityStateActive && pre.Activity.State != ao.ActivityStateWaitingInput) {
		return ErrTimeoutPreflightInadmissible
	}
	cause := "TIMEOUT"
	stop.InitiatingFailureReason = &cause
	stop.RequestedAt = time.Now().UTC()
	if c.Now != nil {
		stop.RequestedAt = c.Now().UTC()
	}
	return c.start(ctx, stop, true)
}

func (c *Coordinator) start(ctx context.Context, stop domain.StopOperation, timeout bool) error {
	if c.Store == nil || c.AO == nil || c.KillTimeout <= 0 {
		return errors.New("stop: store, AO and injected kill timeout are required")
	}
	if stop.OperationID == "" || stop.PairID == "" || stop.SessionID == "" || stop.TerminalGeneration == "" {
		return errors.New("stop: exact operation/Pair/session/generation required")
	}
	if stop.RequestedAt.IsZero() {
		stop.RequestedAt = time.Now().UTC()
	}
	if stop.Actor == "" {
		stop.Actor = "supervisor"
	}
	if stop.Purpose != domain.RunningAttemptStop {
		if c.Operator == nil {
			return errors.New("stop: verified operator boundary required for session-scoped cleanup")
		}
		scope := "STOP_UNLINKED_CLEANUP"
		if stop.RestoreOperationID != nil {
			scope = "RESTORE_CLEANUP_CLAIM"
		}
		principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, stop.PairID, stop.SessionID, stop.TerminalGeneration, scope)
		if err != nil {
			return err
		}
		if !verified || principal == "" {
			return errors.New("stop: verified operator principal required")
		}
		stop.Actor = principal
		if stop.RestoreOperationID != nil {
			stop.RestorePrincipal = &principal
		}
	}
	// A linked maintenance claim requires a fresh observation of the *new*
	// runtime generation; it does not borrow an old attempt's immutable snapshot.
	var claimObservation *ao.WorkerStatus
	if stop.RestoreOperationID != nil && stop.Purpose == domain.PairMaintenance {
		var err error
		claimObservation, err = c.AO.GetWorkerStatus(ctx, stop.SessionID)
		if err != nil {
			return err
		}
		if claimObservation == nil || claimObservation.ID != stop.SessionID || claimObservation.TerminalGeneration != stop.TerminalGeneration || claimObservation.IsTerminated {
			return errors.New("stop: linked maintenance observation differs from target")
		}
	}
	var err error
	if stop.RestoreOperationID != nil {
		if claimObservation != nil {
			err = c.Store.ClaimRestoreCleanupWithStop(ctx, stop, *stop.RestorePrincipal, stop.Actor, stop.RequestedAt, store.RestoreCleanupObservation{SessionID: claimObservation.ID, TerminalGeneration: claimObservation.TerminalGeneration, IsTerminated: claimObservation.IsTerminated})
		} else {
			err = c.Store.ClaimRestoreCleanupWithStop(ctx, stop, *stop.RestorePrincipal, stop.Actor, stop.RequestedAt)
		}
	} else {
		err = c.Store.ReserveStopOperation(ctx, stop)
	}
	if err != nil {
		return err
	} // including commit ambiguity: never reconstruct a permit by reading the row
	// The permit is confined to this stack frame. No caller can resume it after
	// crash; one exact GET is made outside the reservation transaction.
	status, getErr := c.AO.GetWorkerStatus(ctx, stop.SessionID)
	if getErr != nil {
		if errors.Is(getErr, ao.ErrNotFound) {
			return c.commitOutcome(ctx, stop.OperationID, domain.StopRequested, domain.StopTargetAbsent, domain.StopResolutionTargetAbsent, nil)
		}
		return fmt.Errorf("stop: pre-effect observation unavailable; intent remains locked: %w", getErr)
	}
	if status == nil || status.ID != stop.SessionID {
		return errors.New("stop: pre-effect identity invalid; intent remains locked")
	}
	if status.TerminalGeneration != stop.TerminalGeneration {
		return c.commitOutcome(ctx, stop.OperationID, domain.StopRequested, domain.StopRequested, domain.StopResolutionGenerationMismatch, status)
	}
	if status.IsTerminated {
		return c.commitOutcome(ctx, stop.OperationID, domain.StopRequested, domain.StopRequested, domain.StopResolutionEffectUnprovenAlreadyTerminated, status)
	}
	if timeout {
		if status.Activity.State != ao.ActivityStateActive && status.Activity.State != ao.ActivityStateWaitingInput {
			return errors.New("stop: timeout pre-effect activity changed; intent retained for exclusive recovery")
		}
		if err := c.Store.ValidateTimeoutEffectOwner(ctx, stop.OperationID); err != nil {
			return fmt.Errorf("stop: timeout pre-effect ownership changed; intent retained: %w", err)
		}
	}
	result, callErr := c.AO.StopWorker(ctx, stop.SessionID)
	commitCtx := context.WithoutCancel(ctx)
	if callErr != nil {
		stage, resolution := domain.StopRequested, domain.StopResolutionCallOutcomeUnknown
		if errors.Is(callErr, ao.ErrNotFound) {
			stage, resolution = domain.StopTargetAbsent, domain.StopResolutionTargetAbsent
		} else {
			var apiErr *ao.APIError
			if errors.As(callErr, &apiErr) {
				stage, resolution = domain.StopCallFailed, domain.StopResolutionCallFailed
			}
		}
		if err := c.commitOutcome(commitCtx, stop.OperationID, domain.StopRequested, stage, resolution, nil); err != nil {
			return fmt.Errorf("stop: call error %v; containment failed: %w", callErr, err)
		}
		return callErr
	}
	if result == nil || result.SessionID != stop.SessionID {
		if err := c.commitOutcome(commitCtx, stop.OperationID, domain.StopRequested, domain.StopRequested, domain.StopResolutionCallOutcomeUnknown, nil); err != nil {
			return fmt.Errorf("stop: invalid HTTP 200 response; containment failed: %w", err)
		}
		return errors.New("stop: invalid HTTP 200 response; delivery outcome unknown")
	}
	callAt := time.Now().UTC()
	if err := c.Store.CommitStopCallAccepted(commitCtx, stop.OperationID, callAt, callAt.Add(c.KillTimeout)); err != nil {
		// The confirmation transaction may have committed ambiguously or another
		// writer may have won. Reread; contain only an exact still-unresolved intent.
		current, readErr := c.Store.GetStopOperation(commitCtx, stop.OperationID)
		if readErr != nil {
			return fmt.Errorf("stop: HTTP 200 confirmation failed (%v), reread failed: %w", err, readErr)
		}
		if current.Stage == domain.StopRequested && current.ResolutionState == domain.StopResolutionInFlight {
			if containErr := c.commitOutcome(commitCtx, stop.OperationID, domain.StopRequested, domain.StopRequested, domain.StopResolutionCallOutcomeUnknown, nil); containErr != nil {
				return fmt.Errorf("stop: HTTP 200 confirmation failed (%v), containment failed: %w", err, containErr)
			}
		}
		return fmt.Errorf("stop: HTTP 200 confirmation failed or writer won: %w", err)
	}
	// One observation only; if still alive, 3D later owns polling/recovery.
	return c.Observe(commitCtx, stop.OperationID)
}

// Observe makes one status GET and commits its exact D11 result. It does not
// issue /kill, reset the deadline, or start a loop.
func (c *Coordinator) Observe(ctx context.Context, operationID string) error {
	if c.Store == nil || c.AO == nil {
		return errors.New("stop: dependencies unavailable")
	}
	stop, err := c.Store.GetStopOperation(ctx, operationID)
	if err != nil {
		return err
	}
	if stop.Stage != domain.StopCallSucceeded || stop.ResolutionState != domain.StopResolutionInFlight || stop.ConfirmationDeadlineAt == nil {
		return store.ErrStateConflict
	}
	status, getErr := c.AO.GetWorkerStatus(ctx, stop.SessionID)
	if errors.Is(getErr, ao.ErrNotFound) {
		return c.commitOutcome(ctx, operationID, domain.StopCallSucceeded, domain.StopTargetAbsent, domain.StopResolutionTargetAbsent, nil)
	}
	now := time.Now().UTC()
	if getErr != nil {
		if !now.Before(*stop.ConfirmationDeadlineAt) {
			return c.commitOutcome(ctx, operationID, domain.StopCallSucceeded, domain.StopCallSucceeded, domain.StopResolutionConfirmationTimeout, nil)
		}
		return getErr
	}
	if status == nil || status.ID != stop.SessionID {
		return errors.New("stop: observation identity invalid; quarantine retained")
	}
	if status.TerminalGeneration != stop.TerminalGeneration {
		return c.commitOutcome(ctx, operationID, domain.StopCallSucceeded, domain.StopCallSucceeded, domain.StopResolutionGenerationMismatch, status)
	}
	if !now.Before(*stop.ConfirmationDeadlineAt) {
		return c.commitOutcome(ctx, operationID, domain.StopCallSucceeded, domain.StopCallSucceeded, domain.StopResolutionConfirmationTimeout, status)
	}
	if status.IsTerminated {
		return c.commitOutcome(ctx, operationID, domain.StopCallSucceeded, domain.StopTerminationConfirmed, domain.StopResolutionTerminationConfirmed, status)
	}
	return nil
}

func (c *Coordinator) commitOutcome(ctx context.Context, id string, expected, next domain.StopStage, resolution domain.StopResolutionState, status *ao.WorkerStatus) error {
	outcome := store.StopTerminalOutcome{ExpectedStage: expected, Stage: next, Resolution: resolution, At: time.Now().UTC()}
	if status != nil {
		outcome.ObservedSessionID = status.ID
		outcome.ObservedGeneration = status.TerminalGeneration
		outcome.ObservedIsTerminated = status.IsTerminated
	}
	return c.Store.CommitStopOutcome(context.WithoutCancel(ctx), id, outcome)
}

// ClearQuarantine checks the trusted host boundary before the lineage-complete
// D6 transaction. A fake principal in a library test is not host authentication.
func (c *Coordinator) ClearQuarantine(ctx context.Context, request store.QuarantineClearance) error {
	if c.Store == nil || c.Operator == nil {
		return errors.New("stop: trusted operator boundary required for D6 clearance")
	}
	scope := "STOP_PHYSICAL_CLEARANCE"
	for _, lineage := range append([]store.LineageClearance{request.Session}, request.Attempts...) {
		if lineage.Basis != store.ClearancePhysical {
			scope = "STOP_ADMINISTRATIVE_CLEARANCE"
			break
		}
	}
	principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, request.PairID, request.Session.SessionID, request.Session.TerminalGeneration, scope)
	if err != nil {
		return err
	}
	if !verified || principal == "" || request.Principal != "" && request.Principal != principal {
		return errors.New("stop: verified operator principal required for D6")
	}
	request.Principal = principal
	return c.Store.ClearQuarantineWithEvidence(ctx, request)
}

// AcceptAdministrativeRisk never calls AO or grants permission to issue a
// previously persisted /kill intent. The host must authenticate the principal.
func (c *Coordinator) AcceptAdministrativeRisk(ctx context.Context, decision store.AdministrativeStopDecision) error {
	if c.Store == nil || c.Operator == nil {
		return errors.New("stop: trusted operator boundary required for risk acceptance")
	}
	const scope = "STOP_ADMINISTRATIVE_RECONCILIATION"
	principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, decision.PairID, decision.SessionID, decision.TerminalGeneration, scope)
	if err != nil {
		return err
	}
	if !verified || principal == "" || decision.Principal != "" && decision.Principal != principal {
		return errors.New("stop: verified operator principal required for risk acceptance")
	}
	decision.Principal = principal
	decision.AuthorityScope = scope
	return c.Store.AcceptStopAdministrativeRisk(ctx, decision)
}
