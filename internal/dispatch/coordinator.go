package dispatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/domain"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

// AO is the pinned adapter surface used by P03. Coordinator methods never hold
// a SQLite transaction while calling it.
type AO interface {
	CreateWorkerSession(context.Context, string, string) (*ao.CreateWorkerSessionResult, error)
	GetWorkerStatus(context.Context, string) (*ao.WorkerStatus, error)
	ResumeWorker(context.Context, string) (*ao.ResumeWorkerResult, error)
	DispatchTaskContract(context.Context, string, string) (*ao.DispatchTaskResult, error)
}

// OperatorBoundary is supplied by the host/bootstrap integration. Tests may
// exercise this boundary, but a fake principal does not authenticate a host.
type OperatorBoundary interface {
	VerifiedRestorePrincipal(context.Context, string, string, string, string) (string, bool, error)
}

type Coordinator struct {
	Store           *store.Store
	AO              AO
	Operator        OperatorBoundary
	RestoreEnabled  bool
	ExecutionPolicy domain.ExecutionBudgetPolicy // injected; no operational default
}

func (c *Coordinator) Provision(ctx context.Context, operation domain.PairProvisioningOperation, projectID, harness, actor string) error {
	if c.Store == nil || c.AO == nil {
		return errors.New("dispatch: coordinator dependencies are not configured")
	}
	if err := c.Store.ReservePairProvisioning(ctx, operation, actor); err != nil {
		return err
	}
	result, err := c.AO.CreateWorkerSession(ctx, projectID, harness)
	if err != nil {
		commitCtx := context.WithoutCancel(ctx)
		_ = c.Store.FailPairProvisioning(commitCtx, operation.OperationID, actor, "upstream create result unresolved: "+err.Error(), time.Now().UTC())
		return fmt.Errorf("dispatch: AO create failed; Pair remains reserved: %w", err)
	}
	if result == nil {
		_ = c.Store.FailPairProvisioning(context.WithoutCancel(ctx), operation.OperationID, actor, "create response missing; provisioning cannot be replayed", time.Now().UTC())
		return fmt.Errorf("dispatch: AO create returned empty result; provisioning intent remains reserved")
	}
	if result.Session.ID == "" || result.Session.TerminalGeneration == "" || result.Session.IsTerminated {
		_ = c.Store.FailPairProvisioning(context.WithoutCancel(ctx), operation.OperationID, actor, "create response failed session identity/state validation", time.Now().UTC())
		return errors.New("dispatch: invalid or terminated create response; Pair remains reserved")
	}
	commitCtx := context.WithoutCancel(ctx)
	confirmErr := c.Store.ConfirmPairProvisioning(commitCtx, operation.OperationID, domain.WorkerSession{PairID: operation.PairID, SessionID: result.Session.ID, RuntimeType: harness, WorkerAgentID: harness, Status: domain.WorkerSessionIdle, TerminalGeneration: result.Session.TerminalGeneration, QuarantineState: domain.QuarantineClean}, actor, time.Now().UTC())
	if confirmErr == nil {
		return nil
	}
	durable, readErr := c.Store.GetPairProvisioningOperation(commitCtx, operation.OperationID)
	if readErr == nil && durable.Stage == domain.ProvisionConfirmed && durable.SessionID != nil && *durable.SessionID == result.Session.ID {
		return nil
	}
	return fmt.Errorf("dispatch: AO create succeeded but confirmation was not proven; durable reread=%+v read_error=%v; never spawn replacement: %w", durable, readErr, confirmErr)
}

func (c *Coordinator) Restore(ctx context.Context, request domain.RestoreReservation) error {
	if !c.RestoreEnabled || c.Operator == nil || c.Store == nil || c.AO == nil {
		return errors.New("dispatch: operator restore is disabled; verified host principal boundary unavailable")
	}
	principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, request.PairID, request.SessionID, request.ExpectedGeneration, request.RiskScope)
	if err != nil {
		return fmt.Errorf("dispatch: verify operator principal: %w", err)
	}
	if !verified || principal == "" {
		return errors.New("dispatch: restore requires verified operator principal")
	}
	request.AuthorizedPrincipal = principal
	request.Actor = principal
	if err := c.Store.ReservePairRestore(ctx, request); err != nil {
		return err
	}
	result, err := c.AO.ResumeWorker(ctx, request.SessionID)
	if err != nil {
		recoveryCtx := context.WithoutCancel(ctx)
		transitionErr := c.Store.RecordPairRestoreUnknown(recoveryCtx, request.OperationID, request.Actor)
		durable, readErr := c.Store.GetPairRestoreOperation(recoveryCtx, request.OperationID)
		return fmt.Errorf("dispatch: restore outcome unresolved; CAS=%v durable=%+v read_error=%v; no retry permitted: %w", transitionErr, durable, readErr, err)
	}
	if result == nil || result.SessionID != request.SessionID {
		recoveryCtx := context.WithoutCancel(ctx)
		transitionErr := c.Store.RecordPairRestoreUnknown(recoveryCtx, request.OperationID, request.Actor)
		durable, readErr := c.Store.GetPairRestoreOperation(recoveryCtx, request.OperationID)
		return fmt.Errorf("dispatch: restore response invalid; CAS=%v durable=%+v read_error=%v; no retry permitted", transitionErr, durable, readErr)
	}
	if result.Session.ID != request.SessionID || result.Session.TerminalGeneration == "" {
		recoveryCtx := context.WithoutCancel(ctx)
		transitionErr := c.Store.RecordPairRestoreUnknown(recoveryCtx, request.OperationID, request.Actor)
		return fmt.Errorf("dispatch: restore HTTP 200 body lacks exact session/generation provenance; CAS=%v; no retry permitted", transitionErr)
	}
	observed, observeErr := c.AO.GetWorkerStatus(context.WithoutCancel(ctx), request.SessionID)
	if observeErr != nil || observed == nil || observed.ID != request.SessionID || observed.TerminalGeneration != result.Session.TerminalGeneration {
		recoveryCtx := context.WithoutCancel(ctx)
		transitionErr := c.Store.RecordPairRestoreUnknown(recoveryCtx, request.OperationID, request.Actor)
		return fmt.Errorf("dispatch: post-restore GetWorkerStatus did not confirm same identity/generation; observation=%+v error=%v CAS=%v; no retry permitted", observed, observeErr, transitionErr)
	}
	status := domain.WorkerSessionActive
	switch observed.Activity.State {
	case ao.ActivityStateIdle, ao.ActivityStateWaitingInput:
		status = domain.WorkerSessionIdle
	case ao.ActivityStateActive, ao.ActivityStateBlocked:
		status = domain.WorkerSessionActive
	case ao.ActivityStateExited:
		status = domain.WorkerSessionTerminated
	}
	recoveryCtx := context.WithoutCancel(ctx)
	confirmErr := c.Store.ConfirmPairRestore(recoveryCtx, request.OperationID, request.SessionID, request.ExpectedGeneration, result.Session.TerminalGeneration, string(result.RestoreMode), status, request.Actor, time.Now().UTC())
	if confirmErr == nil {
		return nil
	}
	durable, readErr := c.Store.GetPairRestoreOperation(recoveryCtx, request.OperationID)
	if readErr == nil && durable.Stage == "RESTORE_CONFIRMED" && durable.HTTP200At != nil && durable.ObservedGeneration != nil && *durable.ObservedGeneration == result.Session.TerminalGeneration && durable.RestoreMode != nil && *durable.RestoreMode == string(result.RestoreMode) {
		return nil
	}
	if readErr == nil && durable.ResolutionState == domain.RestoreInFlight {
		_ = c.Store.RecordPairRestoreUnknown(recoveryCtx, request.OperationID, request.Actor)
		durable, readErr = c.Store.GetPairRestoreOperation(recoveryCtx, request.OperationID)
	}
	return fmt.Errorf("dispatch: restore HTTP 200 confirmation did not persist; durable reread=%+v read_error=%v; do not retry restore: %w", durable, readErr, confirmErr)
}

// ResolveRestore requires a fresh idle/waiting AO observation and a verified
// host principal. It resolves only the restore operation; quarantine clearance
// remains a separate lineage-specific decision.
func (c *Coordinator) ResolveRestore(ctx context.Context, operationID, pairID, sessionID, generation, actor string) error {
	if c.Operator == nil || c.Store == nil || c.AO == nil {
		return errors.New("dispatch: restore resolution requires verified host boundary")
	}
	status, err := c.AO.GetWorkerStatus(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("dispatch: restore resolution observation failed: %w", err)
	}
	if status == nil || status.ID != sessionID || status.TerminalGeneration != generation || status.IsTerminated {
		return errors.New("dispatch: restore resolution observation is not admissible")
	}
	principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, pairID, sessionID, generation, "RESTORE_RESOLUTION")
	if err != nil {
		return err
	}
	if !verified || principal == "" {
		return errors.New("dispatch: verified operator principal required")
	}
	return c.Store.ResolveConfirmedRestore(context.WithoutCancel(ctx), operationID, sessionID, generation, string(status.Activity.State), principal, principal, time.Now().UTC())
}

func (c *Coordinator) ResolveAmbiguousRestore(ctx context.Context, operationID, pairID, sessionID, generation string, basis domain.RestoreResolutionBasis, evidence, actor string) error {
	if c.Operator == nil || c.Store == nil {
		return errors.New("dispatch: restore risk resolution requires host operator boundary")
	}
	scope := "RESTORE_PHYSICAL_EXECUTION_RESOLUTION"
	if basis == domain.AdministrativeRiskResolution {
		scope = "RESTORE_ADMINISTRATIVE_RISK_RESOLUTION"
	}
	principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, pairID, sessionID, generation, scope)
	if err != nil {
		return err
	}
	if !verified || principal == "" {
		return errors.New("dispatch: verified operator principal required")
	}
	return c.Store.ResolveAmbiguousRestore(ctx, operationID, pairID, sessionID, generation, basis, evidence, principal, principal, time.Now().UTC())
}

func (c *Coordinator) ClaimRestoreCleanup(ctx context.Context, stop domain.StopOperation, actor string) error {
	if c.Operator == nil || c.Store == nil {
		return errors.New("dispatch: cleanup claim requires host operator boundary")
	}
	if stop.RestoreOperationID == nil {
		return errors.New("dispatch: linked cleanup requires restore operation ID")
	}
	principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, stop.PairID, stop.SessionID, stop.TerminalGeneration, "RESTORE_CLEANUP_CLAIM")
	if err != nil {
		return err
	}
	if !verified || principal == "" {
		return errors.New("dispatch: verified operator principal required")
	}
	stop.RestorePrincipal = &principal
	stop.Actor = principal
	if stop.Purpose == domain.PairMaintenance {
		if c.AO == nil {
			return errors.New("dispatch: cleanup runtime observation requires AO status reader")
		}
		status, err := c.AO.GetWorkerStatus(ctx, stop.SessionID)
		if err != nil {
			return fmt.Errorf("dispatch: cleanup runtime observation failed: %w", err)
		}
		if status == nil || status.ID != stop.SessionID || status.TerminalGeneration != stop.TerminalGeneration || status.IsTerminated {
			return errors.New("dispatch: cleanup runtime observation differs from target")
		}
		return c.Store.ClaimRestoreCleanupWithStop(ctx, stop, principal, principal, time.Now().UTC(), store.RestoreCleanupObservation{SessionID: status.ID, TerminalGeneration: status.TerminalGeneration, IsTerminated: status.IsTerminated})
	}
	return c.Store.ClaimRestoreCleanupWithStop(ctx, stop, principal, principal, time.Now().UTC())
}

func (c *Coordinator) ClaimRestoreRecovery(ctx context.Context, operationID, pairID, sessionID, generation, actor string) error {
	if c.Operator == nil || c.Store == nil {
		return errors.New("dispatch: restore recovery claim requires host operator boundary")
	}
	principal, verified, err := c.Operator.VerifiedRestorePrincipal(ctx, pairID, sessionID, generation, "RESTORE_RECOVERY_CLAIM")
	if err != nil {
		return err
	}
	if !verified || principal == "" {
		return errors.New("dispatch: verified operator principal required")
	}
	return c.Store.ClaimRestoreRecovery(ctx, operationID, pairID, sessionID, generation, principal, principal, time.Now().UTC())
}

// Dispatch allocates only a bound attempt, checks AO identity/activity before
// committing SEND_REQUESTED, then makes one send call. It never retries an
// ambiguous delivery and HTTP 200 leaves the Task DISPATCHED.
func (c *Coordinator) Dispatch(ctx context.Context, taskID, contractID, attemptID, operationID, sessionID, generation, reportPath, message, actor string) error {
	if c.Store == nil || c.AO == nil {
		return errors.New("dispatch: coordinator dependencies are not configured")
	}
	policy := c.ExecutionPolicy
	if policy.Duration <= 0 || strings.TrimSpace(policy.PolicyRef) == "" {
		return errors.New("dispatch: injected execution budget policy required before send")
	}
	now := time.Now().UTC()
	deadline := now.Add(policy.Duration)
	if !deadline.After(now) || now.Year() < 1 || deadline.Year() > 9999 {
		return errors.New("dispatch: injected execution budget policy required before send")
	}
	task, err := c.Store.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	var attempt domain.TaskAttempt
	if task.State == domain.StateReady {
		attempt, err = c.Store.PrepareBoundDispatch(ctx, taskID, contractID, attemptID, reportPath, time.Now().UTC(), store.DispatchBinding{OperationID: operationID, SessionID: sessionID, TerminalGeneration: generation})
		if err != nil {
			return err
		}
	} else if task.State == domain.StateDispatched {
		attempt, err = c.Store.GetTaskAttempt(ctx, attemptID)
		if err != nil {
			return err
		}
		durable, readErr := c.Store.GetDispatchOperation(ctx, operationID)
		if readErr != nil {
			return readErr
		}
		if task.CurrentAttempt != attempt.AttemptNumber || attempt.EndedAt != nil || attempt.TaskID != taskID || attempt.ContractID != contractID || attempt.SessionID == nil || *attempt.SessionID != sessionID || attempt.TerminalGeneration == nil || *attempt.TerminalGeneration != generation || durable.AttemptID != attemptID || durable.Stage != domain.DispatchBound {
			return errors.New("dispatch: current open attempt/operation does not match retry request")
		}
	} else {
		return fmt.Errorf("dispatch: Task %s is %s; only READY allocation or its existing DISPATCHED attempt is admissible", taskID, task.State)
	}
	status, err := c.AO.GetWorkerStatus(ctx, sessionID)
	if err != nil {
		recoveryCtx := context.WithoutCancel(ctx)
		disposition := "RECOVERY_PENDING"
		var protocolError *ao.ProtocolError
		if errors.As(err, &protocolError) && protocolError.StatusCode == 404 {
			if transitionErr := c.Store.AtomicTerminalTransition(recoveryCtx, taskID, domain.StateDispatched, domain.StateFailed, "AO session absent before send", attempt.AttemptID, "SESSION_ABSENT"); transitionErr != nil {
				return fmt.Errorf("dispatch: HTTP 404 terminal CAS failed; do not send: %w", transitionErr)
			}
			return fmt.Errorf("dispatch: AO returned HTTP 404; D12 failure recorded; no send issued: %w", err)
		} else if errors.As(err, &protocolError) {
			disposition = "PRE_SEND_PROTOCOL_UNVERIFIED"
		}
		if holdErr := c.Store.RecordPreSendHold(recoveryCtx, operationID, disposition, actor, time.Now().UTC()); holdErr != nil {
			return fmt.Errorf("dispatch: pre-send observation unavailable and hold/audit failed; no send: %v; %w", err, holdErr)
		}
		return fmt.Errorf("dispatch: pre-send observation unavailable; SEND was not requested: %w", err)
	}
	if status == nil {
		if holdErr := c.Store.RecordPreSendHold(context.WithoutCancel(ctx), operationID, "PRE_SEND_PROTOCOL_UNVERIFIED", actor, time.Now().UTC()); holdErr != nil {
			return fmt.Errorf("dispatch: missing status and protocol hold/audit failed: %w", holdErr)
		}
		return errors.New("dispatch: pre-send protocol response missing; no send issued")
	}
	if status.ID != sessionID {
		if transitionErr := c.Store.AtomicTerminalTransition(context.WithoutCancel(ctx), taskID, domain.StateDispatched, domain.StateFailed, "pre-send AO session identity mismatch", attempt.AttemptID, "PRE_SEND_IDENTITY_MISMATCH"); transitionErr != nil {
			return fmt.Errorf("dispatch: identity mismatch D12 transaction failed: %w", transitionErr)
		}
		return errors.New("dispatch: pre-send identity/generation mismatch; no send issued")
	}
	if status.TerminalGeneration != generation {
		if transitionErr := c.Store.AtomicTerminalTransition(context.WithoutCancel(ctx), taskID, domain.StateDispatched, domain.StateFailed, "stale execution generation", attempt.AttemptID, "STALE_EXECUTION_GENERATION"); transitionErr != nil {
			return fmt.Errorf("dispatch: stale-generation D12 transaction failed: %w", transitionErr)
		}
		return errors.New("dispatch: stale execution generation; attempt failed closed")
	}
	if status.IsTerminated {
		if transitionErr := c.Store.AtomicTerminalTransition(context.WithoutCancel(ctx), taskID, domain.StateDispatched, domain.StateFailed, "worker termination is unverified", attempt.AttemptID, "WORKER_TERMINATION_UNKNOWN"); transitionErr != nil {
			return fmt.Errorf("dispatch: termination D12 transaction failed: %w", transitionErr)
		}
		return errors.New("dispatch: pre-send termination observed; no send issued")
	}
	if status.Activity.State != ao.ActivityStateIdle && status.Activity.State != ao.ActivityStateWaitingInput {
		activity := string(status.Activity.State)
		if activity == "active" || activity == "blocked" || activity == "exited" {
			if auditErr := c.Store.RecordPreSendAdmissibilityRejected(context.WithoutCancel(ctx), operationID, activity, actor, time.Now().UTC()); auditErr != nil {
				return fmt.Errorf("dispatch: inadmissible state audit failed: %w", auditErr)
			}
		}
		return fmt.Errorf("dispatch: pre-send activity %q is not admissible; attempt remains DISPATCHED", status.Activity.State)
	}
	if err := c.Store.RecordSendRequested(ctx, operationID, status.ID, status.TerminalGeneration, string(status.Activity.State), status.IsTerminated, actor, time.Now().UTC()); err != nil {
		return err
	}
	result, err := c.AO.DispatchTaskContract(ctx, sessionID, message)
	if err != nil {
		return c.containAmbiguousSend(ctx, operationID, attempt.AttemptID, actor, err)
	}
	if result == nil || result.SessionID != sessionID {
		return c.containAmbiguousSend(ctx, operationID, attempt.AttemptID, actor, errors.New("invalid send response"))
	}
	commitCtx := context.WithoutCancel(ctx)
	if err := c.Store.RecordSendConfirmed(commitCtx, operationID, actor, true, time.Now().UTC(), policy); err != nil {
		return c.containAmbiguousSend(commitCtx, operationID, attempt.AttemptID, actor, fmt.Errorf("HTTP 200 confirmation transaction failed: %w", err))
	}
	_ = attempt
	return nil
}

// containAmbiguousSend rereads the full durable dispatch tuple and its audit
// chain before attempting a fresh D5 CAS. It never retries the AO send.
func (c *Coordinator) containAmbiguousSend(ctx context.Context, operationID, attemptID, actor string, cause error) error {
	recoveryCtx := context.WithoutCancel(ctx)
	read := func() (domain.DispatchOperation, domain.Task, domain.TaskAttempt, map[string]bool, error) {
		op, err := c.Store.GetDispatchOperation(recoveryCtx, operationID)
		if err != nil {
			return op, domain.Task{}, domain.TaskAttempt{}, nil, err
		}
		task, err := c.Store.GetTask(recoveryCtx, op.TaskID)
		if err != nil {
			return op, task, domain.TaskAttempt{}, nil, err
		}
		attempt, err := c.Store.GetTaskAttempt(recoveryCtx, attemptID)
		if err != nil {
			return op, task, attempt, nil, err
		}
		if err := c.Store.VerifyAuditChain(recoveryCtx); err != nil {
			return op, task, attempt, nil, err
		}
		found := map[string]bool{}
		var after int64
		for {
			events, err := c.Store.ListAuditEvents(recoveryCtx, after, store.MaxAuditLimit)
			if err != nil {
				return op, task, attempt, nil, err
			}
			for _, event := range events {
				after = event.Sequence
				id, _ := event.Event.Details["dispatch_operation_id"].(string)
				if id != operationID || event.Event.PairID != op.PairID || event.Event.TaskID != op.TaskID || event.Event.AttemptID != op.AttemptID {
					continue
				}
				if event.Event.EventType == domain.AuditDispatchSendRequested || event.Event.EventType == domain.AuditUncertainDeliveryQuarantine {
					sessionID, _ := event.Event.Details["session_id"].(string)
					generation, _ := event.Event.Details["terminal_generation"].(string)
					if sessionID != op.SessionID || generation != op.TerminalGeneration {
						continue
					}
				}
				found[event.Event.EventType] = true
				if event.Event.EventType == domain.AuditTaskStateTransition {
					from, _ := event.Event.Details["from_state"].(string)
					to, _ := event.Event.Details["to_state"].(string)
					if from != "" && to != "" {
						found[domain.AuditTaskStateTransition+":"+from+"->"+to] = true
					}
				}
			}
			if len(events) < store.MaxAuditLimit {
				break
			}
		}
		return op, task, attempt, found, nil
	}
	op, task, attempt, auditTypes, readErr := read()
	if readErr != nil {
		return fmt.Errorf("dispatch: ambiguous delivery; durable tuple/audit reread failed, no resend: cause=%v reread=%w", cause, readErr)
	}
	if op.OperationID != operationID || op.AttemptID != attemptID || attempt.AttemptID != attemptID || attempt.TaskID != op.TaskID || attempt.ContractID == "" || task.TaskID != op.TaskID || task.PairID != op.PairID || task.CurrentAttempt != attempt.AttemptNumber {
		return fmt.Errorf("dispatch: ambiguous delivery tuple is stale or inconsistent, no resend: cause=%v", cause)
	}
	if op.Stage == domain.SendConfirmed && auditTypes[domain.AuditDispatchSendConfirmed] {
		return nil
	}
	if op.ResolutionState != nil && *op.ResolutionState == domain.RecoveryDeliveryOutcomeUnknown && auditTypes[domain.AuditUncertainDeliveryQuarantine] && auditTypes[domain.AuditTaskStateTransition+":DISPATCHED->FAILED"] && auditTypes[domain.AuditTaskStateTransition+":FAILED->HUMAN_REQUIRED"] {
		return fmt.Errorf("dispatch: D5 winner is durable; late caller cannot overwrite unknown delivery, no resend: %w", cause)
	}
	if op.Stage != domain.SendRequested || op.ResolutionState != nil || attempt.EndedAt != nil || task.State != domain.StateDispatched || !auditTypes[domain.AuditDispatchSendRequested] || attempt.SessionID == nil || *attempt.SessionID != op.SessionID || attempt.TerminalGeneration == nil || *attempt.TerminalGeneration != op.TerminalGeneration {
		return fmt.Errorf("dispatch: ambiguous delivery durable tuple is not the exact SEND_REQUESTED intent, no resend: cause=%v", cause)
	}
	containErr := c.Store.RecordUnknownDelivery(recoveryCtx, operationID, actor, time.Now().UTC())
	op, task, attempt, auditTypes, readErr = read()
	if readErr != nil {
		return fmt.Errorf("dispatch: D5 containment failed (%v), exact reread failed (%v); intent requires recovery; no resend: %w", containErr, readErr, cause)
	}
	if op.ResolutionState != nil && *op.ResolutionState == domain.RecoveryDeliveryOutcomeUnknown && auditTypes[domain.AuditUncertainDeliveryQuarantine] && auditTypes[domain.AuditTaskStateTransition+":DISPATCHED->FAILED"] && auditTypes[domain.AuditTaskStateTransition+":FAILED->HUMAN_REQUIRED"] && task.State == domain.StateHumanRequired && attempt.EndedAt != nil {
		return fmt.Errorf("dispatch: delivery outcome unknown; D5 containment is durable, no retry permitted: %w", cause)
	}
	return fmt.Errorf("dispatch: D5 containment failed (%v); durable state remains operation=%+v task=%s attempt_ended=%t; recovery required and no resend: %w", containErr, op, task.State, attempt.EndedAt != nil, cause)
}

// RecoverPreSend is an explicit operator path: it fetches fresh status, asks the
// host principal boundary, then clears only the exact hold and lineage.
func (c *Coordinator) RecoverPreSend(ctx context.Context, attemptID, operationID, expectedDisposition, pairID, sessionID, generation, actor string) error {
	if c.Store == nil || c.AO == nil {
		return errors.New("dispatch: pre-send recovery requires Store and AO")
	}
	status, err := c.AO.GetWorkerStatus(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("dispatch: fresh recovery observation failed: %w", err)
	}
	if status == nil || status.ID != sessionID || status.TerminalGeneration != generation || status.IsTerminated {
		return errors.New("dispatch: fresh observation does not match held attempt")
	}
	principal := ""
	if expectedDisposition == "PRE_SEND_PROTOCOL_UNVERIFIED" {
		if c.Operator == nil {
			return errors.New("dispatch: protocol hold recovery requires host operator boundary")
		}
		verifiedPrincipal, verified, verifyErr := c.Operator.VerifiedRestorePrincipal(ctx, pairID, sessionID, generation, "PRE_SEND_PROTOCOL_COMPATIBILITY_ACK")
		if verifyErr != nil {
			return verifyErr
		}
		if !verified || verifiedPrincipal == "" {
			return errors.New("dispatch: verified operator principal required to clear protocol hold")
		}
		principal = verifiedPrincipal
	}
	if principal != "" {
		actor = principal
	}
	return c.Store.ResolvePreSendHold(context.WithoutCancel(ctx), operationID, attemptID, expectedDisposition, sessionID, generation, string(status.Activity.State), principal, actor, time.Now().UTC())
}
