package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"modernc.org/sqlite"
)

// WorkerSessionCASUpdate defines a compare-and-set runtime and quarantine update.
type WorkerSessionCASUpdate struct {
	ExpectedStatus          domain.WorkerSessionStatus
	ExpectedQuarantineState domain.QuarantineState
	Status                  domain.WorkerSessionStatus
	QuarantineState         domain.QuarantineState
	TerminalGeneration      *string
	UpdatedAt               time.Time
}

// PairProvisioningStageUpdate carries nullable outcome fields for a stage CAS.
type PairProvisioningStageUpdate struct {
	SessionID       *string
	CompletedAt     *time.Time
	ResolvedAt      *time.Time
	ResolvedBy      *string
	ResolutionNotes *string
}

// DispatchStageUpdate carries nullable outcome fields for a stage CAS.
type DispatchStageUpdate struct {
	ConfirmedAt     *time.Time
	ResolutionState *string
}

// StopStageUpdate carries wire-effect and resolution facts for a stage CAS.
type StopStageUpdate struct {
	ExpectedResolutionState domain.StopResolutionState
	CallCompletedAt         *time.Time
	ConfirmationDeadlineAt  *time.Time
	TerminationConfirmedAt  *time.Time
	ResolvedAt              *time.Time
	ResolutionState         *domain.StopResolutionState
	ObservedSessionID       string
	ObservedGeneration      string
	ObservedIsTerminated    bool
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func setNullableString(target **string, value sql.NullString) {
	if value.Valid {
		copy := value.String
		*target = &copy
	}
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := parseTime(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func mapLifecycleWriteError(err error, entity string) error {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		message := sqliteErr.Error()
		if strings.Contains(message, "FOREIGN KEY constraint failed") {
			return fmt.Errorf("%w: %s lineage: %v", ErrForeignKeyViolation, entity, err)
		}
		if strings.Contains(message, "UNIQUE constraint failed") {
			return fmt.Errorf("%w: %s: %v", ErrDuplicateKey, entity, err)
		}
	}
	return fmt.Errorf("store: failed to write %s: %w", entity, err)
}

type lifecycleQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func rejectUnresolvedRestore(ctx context.Context, queryer lifecycleQueryer, pairID string) error {
	var unresolved int
	if err := queryer.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED')`, pairID).Scan(&unresolved); err != nil {
		return fmt.Errorf("store: check restore ownership: %w", err)
	}
	if unresolved != 0 {
		return fmt.Errorf("%w: Pair has unresolved restore ownership", ErrQuarantinedExecution)
	}
	return nil
}

func rejectUnresolvedProvisioning(ctx context.Context, queryer lifecycleQueryer, pairID string) error {
	var unresolved int
	if err := queryer.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pair_provisioning_operations WHERE pair_id=? AND stage IN ('PROVISION_REQUESTED','PROVISION_FAILED'))`, pairID).Scan(&unresolved); err != nil {
		return fmt.Errorf("store: check provisioning ownership: %w", err)
	}
	if unresolved != 0 {
		return fmt.Errorf("%w: Pair has unresolved provisioning", ErrQuarantinedExecution)
	}
	return nil
}

// CreateWorkerSession is closed for P03. Production WorkerSession creation must
// share the provisioning intent, Pair guard, and audit transaction.
func (s *Store) CreateWorkerSession(ctx context.Context, session domain.WorkerSession) error {
	_ = ctx
	_ = session
	return fmt.Errorf("%w: use ConfirmPairProvisioning so WorkerSession and provisioning evidence commit atomically", ErrAtomicDispatchRequired)
}

func scanWorkerSession(scanner interface{ Scan(...any) error }) (domain.WorkerSession, error) {
	var session domain.WorkerSession
	var worktreePath sql.NullString
	var status, quarantine, createdAt, updatedAt string
	err := scanner.Scan(&session.PairID, &session.SessionID, &session.RuntimeType, &worktreePath,
		&session.WorkerAgentID, &status, &session.TerminalGeneration, &quarantine, &createdAt, &updatedAt)
	if err != nil {
		return domain.WorkerSession{}, err
	}
	setNullableString(&session.WorktreePath, worktreePath)
	session.Status = domain.WorkerSessionStatus(status)
	session.QuarantineState = domain.QuarantineState(quarantine)
	var parseErr error
	session.CreatedAt, parseErr = parseTime(createdAt)
	if parseErr != nil {
		return domain.WorkerSession{}, fmt.Errorf("store: invalid worker session created_at: %w", parseErr)
	}
	session.UpdatedAt, parseErr = parseTime(updatedAt)
	if parseErr != nil {
		return domain.WorkerSession{}, fmt.Errorf("store: invalid worker session updated_at: %w", parseErr)
	}
	return session, nil
}

const workerSessionSelect = `
SELECT pair_id, session_id, runtime_type, worktree_path, worker_agent_id,
       status, terminal_generation, quarantine_state, created_at, updated_at
FROM worker_sessions`

// GetWorkerSessionByPair retrieves the current WorkerSession for a Pair.
func (s *Store) GetWorkerSessionByPair(ctx context.Context, pairID string) (domain.WorkerSession, error) {
	session, err := scanWorkerSession(s.db.QueryRowContext(ctx, workerSessionSelect+" WHERE pair_id = ?", pairID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WorkerSession{}, ErrWorkerSessionNotFound
	}
	return session, err
}

// GetWorkerSessionByID retrieves a WorkerSession by upstream session identity.
func (s *Store) GetWorkerSessionByID(ctx context.Context, sessionID string) (domain.WorkerSession, error) {
	session, err := scanWorkerSession(s.db.QueryRowContext(ctx, workerSessionSelect+" WHERE session_id = ?", sessionID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WorkerSession{}, ErrWorkerSessionNotFound
	}
	return session, err
}

// UpdateWorkerSessionStatusAndQuarantine atomically updates runtime and safety state using CAS.
func (s *Store) UpdateWorkerSessionStatusAndQuarantine(ctx context.Context, pairID string, update WorkerSessionCASUpdate) error {
	if update.ExpectedQuarantineState != update.QuarantineState || update.TerminalGeneration != nil {
		return fmt.Errorf("%w: generic WorkerSession CAS cannot change quarantine or generation; use an authorized lifecycle transaction", ErrAtomicDispatchRequired)
	}
	updatedAt := update.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = timeNow()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := rejectUnresolvedRestore(ctx, tx, pairID); err != nil {
		return err
	}
	if err := rejectUnresolvedProvisioning(ctx, tx, pairID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE worker_sessions
SET status = ?, quarantine_state = ?,
    terminal_generation = COALESCE(?, terminal_generation), updated_at = ?
WHERE pair_id = ? AND status = ? AND quarantine_state = ?
`, string(update.Status), string(update.QuarantineState), nullableString(update.TerminalGeneration),
		formatTime(updatedAt), pairID, string(update.ExpectedStatus), string(update.ExpectedQuarantineState))
	if err != nil {
		return mapLifecycleWriteError(err, "worker session CAS update")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: worker session CAS rows affected: %w", err)
	}
	if rows == 0 {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT 1 FROM worker_sessions WHERE pair_id = ?", pairID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return ErrWorkerSessionNotFound
		} else if err != nil {
			return fmt.Errorf("store: worker session CAS existence check: %w", err)
		}
		return fmt.Errorf("%w: worker session %q status/quarantine precondition failed", ErrStateConflict, pairID)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit WorkerSession CAS: %w", err)
	}
	return nil
}

// CreatePairProvisioningOperation persists provisioning intent or outcome metadata.
func (s *Store) CreatePairProvisioningOperation(ctx context.Context, operation domain.PairProvisioningOperation) error {
	_ = ctx
	_ = operation
	return fmt.Errorf("%w: use ReservePairProvisioning so Pair guard and audit commit atomically", ErrAtomicDispatchRequired)
}

// GetPairProvisioningOperation retrieves a provisioning operation by identity.
func (s *Store) GetPairProvisioningOperation(ctx context.Context, operationID string) (domain.PairProvisioningOperation, error) {
	var operation domain.PairProvisioningOperation
	var stage, requestedAt string
	var sessionID, completedAt, resolvedAt, resolvedBy, notes sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT operation_id, pair_id, stage, client_token, session_id, requested_at,
       completed_at, resolved_at, resolved_by, resolution_notes
FROM pair_provisioning_operations WHERE operation_id = ?
`, operationID).Scan(&operation.OperationID, &operation.PairID, &stage, &operation.ClientToken,
		&sessionID, &requestedAt, &completedAt, &resolvedAt, &resolvedBy, &notes)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PairProvisioningOperation{}, ErrOperationNotFound
	}
	if err != nil {
		return domain.PairProvisioningOperation{}, fmt.Errorf("store: query pair provisioning operation: %w", err)
	}
	operation.Stage = domain.PairProvisioningStage(stage)
	setNullableString(&operation.SessionID, sessionID)
	setNullableString(&operation.ResolvedBy, resolvedBy)
	setNullableString(&operation.ResolutionNotes, notes)
	operation.RequestedAt, err = parseTime(requestedAt)
	if err != nil {
		return domain.PairProvisioningOperation{}, err
	}
	operation.CompletedAt, err = parseNullableTime(completedAt)
	if err != nil {
		return domain.PairProvisioningOperation{}, err
	}
	operation.ResolvedAt, err = parseNullableTime(resolvedAt)
	return operation, err
}

func validProvisioningTransition(from, to domain.PairProvisioningStage) bool {
	return (from == domain.ProvisionRequested && (to == domain.ProvisionConfirmed || to == domain.ProvisionFailed)) ||
		(from == domain.ProvisionFailed && to == domain.ProvisionResolved)
}

// UpdatePairProvisioningOperationStage advances a provisioning operation with CAS.
func (s *Store) UpdatePairProvisioningOperationStage(ctx context.Context, operationID string, expected, next domain.PairProvisioningStage, update PairProvisioningStageUpdate) error {
	_ = ctx
	_ = operationID
	_ = expected
	_ = next
	_ = update
	return fmt.Errorf("%w: use ConfirmPairProvisioning, FailPairProvisioning, or an authorized resolution transaction", ErrAtomicDispatchRequired)
}

// CreateDispatchOperation is closed for P03. PrepareBoundDispatch owns the
// atomic TaskState, attempt, immutable snapshot, operation, and audit binding.
func (s *Store) CreateDispatchOperation(ctx context.Context, operation domain.DispatchOperation) error {
	_ = ctx
	_ = operation
	return fmt.Errorf("%w: use PrepareBoundDispatch so Pair guards and immutable binding commit atomically", ErrAtomicDispatchRequired)
}

// GetDispatchOperation retrieves a dispatch operation by identity.
func (s *Store) GetDispatchOperation(ctx context.Context, operationID string) (domain.DispatchOperation, error) {
	var operation domain.DispatchOperation
	var stage, requestedAt string
	var confirmedAt, resolutionState sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT operation_id, attempt_id, pair_id, task_id, session_id, terminal_generation,
       stage, requested_at, confirmed_at, resolution_state
FROM dispatch_operations WHERE operation_id = ?
`, operationID).Scan(&operation.OperationID, &operation.AttemptID, &operation.PairID, &operation.TaskID,
		&operation.SessionID, &operation.TerminalGeneration, &stage, &requestedAt, &confirmedAt, &resolutionState)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DispatchOperation{}, ErrOperationNotFound
	}
	if err != nil {
		return domain.DispatchOperation{}, fmt.Errorf("store: query dispatch operation: %w", err)
	}
	operation.Stage = domain.DispatchStage(stage)
	operation.RequestedAt, err = parseTime(requestedAt)
	if err != nil {
		return domain.DispatchOperation{}, err
	}
	operation.ConfirmedAt, err = parseNullableTime(confirmedAt)
	setNullableString(&operation.ResolutionState, resolutionState)
	return operation, err
}

func validDispatchTransition(from, to domain.DispatchStage) bool {
	return (from == domain.DispatchBound && to == domain.SendRequested) ||
		(from == domain.SendRequested && to == domain.SendConfirmed)
}

// UpdateDispatchOperationStage advances a dispatch operation with CAS.
func (s *Store) UpdateDispatchOperationStage(ctx context.Context, operationID string, expected, next domain.DispatchStage, update DispatchStageUpdate) error {
	_ = ctx
	_ = operationID
	_ = expected
	_ = next
	_ = update
	return fmt.Errorf("%w: use RecordSendRequested/RecordSendConfirmed/RecordUnknownDelivery", ErrAtomicDispatchRequired)
}

// CreateStopOperation persists a stop intent after validating optional lineage as one coherent tuple.
func (s *Store) CreateStopOperation(ctx context.Context, operation domain.StopOperation) error {
	// Keep the old API callable for existing clients, but give it the same
	// guarded/audited transaction as the 3C reservation path. It never returns
	// a transferable /kill permit; only the direct caller's new insert can do so.
	return s.ReserveStopOperation(ctx, operation)
}

func createStopOperationTx(ctx context.Context, tx *sql.Tx, operation domain.StopOperation, cleanupObservation ...string) error {
	var err error
	if operation.Purpose == domain.RunningAttemptStop || operation.Purpose == domain.QuarantineCleanup {
		if operation.AttemptID == nil {
			return fmt.Errorf("%w: %s requires attempt lineage", ErrAttemptLineageMismatch, operation.Purpose)
		}
	}
	if operation.Purpose == domain.PairMaintenance && operation.AttemptID != nil {
		return fmt.Errorf("%w: pair maintenance cannot bind an attempt", ErrAttemptLineageMismatch)
	}
	if operation.AttemptID != nil {
		if operation.TaskID == nil || operation.ContractID == nil {
			return fmt.Errorf("%w: attempt_id requires task_id and contract_id", ErrAttemptLineageMismatch)
		}
		var snapshotSessionID, snapshotGeneration sql.NullString
		err = tx.QueryRowContext(ctx, `
SELECT a.session_id, a.terminal_generation FROM task_attempts a JOIN tasks t ON t.task_id = a.task_id
WHERE a.attempt_id = ? AND a.task_id = ? AND a.contract_id = ? AND t.pair_id = ?
`, *operation.AttemptID, *operation.TaskID, *operation.ContractID, operation.PairID).Scan(&snapshotSessionID, &snapshotGeneration)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptLineageMismatch
		}
		if err != nil {
			return fmt.Errorf("store: stop operation lineage query: %w", err)
		}
		if !snapshotSessionID.Valid || !snapshotGeneration.Valid || snapshotSessionID.String != operation.SessionID || snapshotGeneration.String != operation.TerminalGeneration {
			return fmt.Errorf("%w: stop session/generation differs from attempt snapshot", ErrAttemptLineageMismatch)
		}
	} else if operation.ContractID != nil {
		if operation.TaskID == nil {
			return fmt.Errorf("%w: contract_id requires task_id", ErrAttemptLineageMismatch)
		}
		var exists int
		err = tx.QueryRowContext(ctx, `
SELECT 1 FROM task_contracts c JOIN tasks t ON t.task_id = c.task_id
WHERE c.contract_id = ? AND c.task_id = ? AND t.pair_id = ?
`, *operation.ContractID, *operation.TaskID, operation.PairID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptLineageMismatch
		}
		if err != nil {
			return fmt.Errorf("store: stop operation contract lineage query: %w", err)
		}
	} else if operation.TaskID != nil {
		var exists int
		err = tx.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE task_id = ? AND pair_id = ?`, *operation.TaskID, operation.PairID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAttemptLineageMismatch
		}
		if err != nil {
			return fmt.Errorf("store: stop operation task lineage query: %w", err)
		}
	}
	var restoreID, restoreStage, restoreResolution, restoreSession, restoreExpected, recoveryPrincipal string
	var restoreObserved sql.NullString
	restoreErr := tx.QueryRowContext(ctx, `SELECT operation_id,stage,resolution_state,session_id,expected_generation,observed_generation,recovery_principal FROM pair_restore_operations WHERE pair_id=? AND resolution_state<>'RESTORE_RESOLVED'`, operation.PairID).Scan(&restoreID, &restoreStage, &restoreResolution, &restoreSession, &restoreExpected, &restoreObserved, &recoveryPrincipal)
	if restoreErr != nil && !errors.Is(restoreErr, sql.ErrNoRows) {
		return restoreErr
	}
	if errors.Is(restoreErr, sql.ErrNoRows) {
		if operation.RestoreOperationID != nil {
			return fmt.Errorf("%w: restore operation is not unresolved for Pair", ErrAttemptLineageMismatch)
		}
		if err := rejectUnresolvedProvisioning(ctx, tx, operation.PairID); err != nil {
			return err
		}
	} else {
		observedRuntime := len(cleanupObservation) == 1 && cleanupObservation[0] != "" && operation.TerminalGeneration == cleanupObservation[0]
		validNewGenerationMaintenance := operation.Purpose == domain.PairMaintenance && operation.AttemptID == nil && operation.TaskID == nil && operation.ContractID == nil && restoreResolution == string(domain.RestoreCleanupClaimed) && operation.SessionID == restoreSession && observedRuntime && (!restoreObserved.Valid || operation.TerminalGeneration == restoreObserved.String)
		validOldGenerationCleanup := operation.Purpose == domain.QuarantineCleanup && operation.AttemptID != nil && restoreResolution == string(domain.RestoreCleanupClaimed) && operation.SessionID == restoreSession && operation.TerminalGeneration == restoreExpected
		if operation.RestoreOperationID == nil || *operation.RestoreOperationID != restoreID || operation.RestorePrincipal == nil || *operation.RestorePrincipal != recoveryPrincipal || recoveryPrincipal == "" || (!validNewGenerationMaintenance && !validOldGenerationCleanup) {
			return fmt.Errorf("%w: unresolved restore permits only linked exact-lineage cleanup or observed-generation PAIR_MAINTENANCE", ErrQuarantinedExecution)
		}
		if validOldGenerationCleanup {
			var closedQuarantine int
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM task_attempts WHERE attempt_id=? AND ended_at IS NOT NULL AND quarantine_state='QUARANTINED')`, *operation.AttemptID).Scan(&closedQuarantine); err != nil {
				return err
			}
			if closedQuarantine != 1 {
				return fmt.Errorf("%w: QUARANTINE_CLEANUP requires a closed quarantined attempt", ErrAttemptLineageMismatch)
			}
		}
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO stop_operations (
    operation_id, purpose, pair_id, task_id, contract_id, attempt_id, session_id,
    terminal_generation, stage, actor, requested_at, call_completed_at,
    confirmation_deadline_at, termination_confirmed_at, resolved_at, resolution_state, restore_operation_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, operation.OperationID, string(operation.Purpose), operation.PairID, nullableString(operation.TaskID),
		nullableString(operation.ContractID), nullableString(operation.AttemptID), operation.SessionID,
		operation.TerminalGeneration, string(operation.Stage), operation.Actor, formatTime(operation.RequestedAt),
		nullableTime(operation.CallCompletedAt), nullableTime(operation.ConfirmationDeadlineAt),
		nullableTime(operation.TerminationConfirmedAt), nullableTime(operation.ResolvedAt), string(operation.ResolutionState), nullableString(operation.RestoreOperationID))
	if err != nil {
		return mapLifecycleWriteError(err, "stop operation")
	}
	if operation.RestoreOperationID != nil {
		eventID, auditErr := newAuditEventID()
		if auditErr != nil {
			return auditErr
		}
		_, auditErr = appendAuditEventTx(ctx, tx, domain.AuditEvent{EventID: eventID, EventType: domain.AuditStopOperationRequested, Timestamp: operation.RequestedAt, PairID: operation.PairID, TaskID: valueOrEmpty(operation.TaskID), ContractID: valueOrEmpty(operation.ContractID), AttemptID: valueOrEmpty(operation.AttemptID), Actor: operation.Actor, Details: map[string]any{"stop_operation_id": operation.OperationID, "purpose": string(operation.Purpose), "restore_operation_id": *operation.RestoreOperationID, "principal": valueOrEmpty(operation.RestorePrincipal), "session_id": operation.SessionID, "target_generation": operation.TerminalGeneration}})
		if auditErr != nil {
			return auditErr
		}
	}
	return nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// GetStopOperation retrieves a stop operation by identity.
func (s *Store) GetStopOperation(ctx context.Context, operationID string) (domain.StopOperation, error) {
	var operation domain.StopOperation
	var purpose, stage, requestedAt, resolutionState string
	var taskID, contractID, attemptID sql.NullString
	var callCompletedAt, deadlineAt, terminationAt, resolvedAt sql.NullString
	var restoreOperationID sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT operation_id, purpose, pair_id, task_id, contract_id, attempt_id, session_id,
       terminal_generation, stage, actor, requested_at, call_completed_at,
       confirmation_deadline_at, termination_confirmed_at, resolved_at, resolution_state, restore_operation_id
FROM stop_operations WHERE operation_id = ?
`, operationID).Scan(&operation.OperationID, &purpose, &operation.PairID, &taskID, &contractID,
		&attemptID, &operation.SessionID, &operation.TerminalGeneration, &stage, &operation.Actor,
		&requestedAt, &callCompletedAt, &deadlineAt, &terminationAt, &resolvedAt, &resolutionState, &restoreOperationID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StopOperation{}, ErrOperationNotFound
	}
	if err != nil {
		return domain.StopOperation{}, fmt.Errorf("store: query stop operation: %w", err)
	}
	operation.Purpose = domain.StopPurpose(purpose)
	operation.Stage = domain.StopStage(stage)
	operation.ResolutionState = domain.StopResolutionState(resolutionState)
	setNullableString(&operation.TaskID, taskID)
	setNullableString(&operation.ContractID, contractID)
	setNullableString(&operation.AttemptID, attemptID)
	setNullableString(&operation.RestoreOperationID, restoreOperationID)
	operation.RequestedAt, err = parseTime(requestedAt)
	if err != nil {
		return domain.StopOperation{}, err
	}
	operation.CallCompletedAt, err = parseNullableTime(callCompletedAt)
	if err != nil {
		return domain.StopOperation{}, err
	}
	operation.ConfirmationDeadlineAt, err = parseNullableTime(deadlineAt)
	if err != nil {
		return domain.StopOperation{}, err
	}
	operation.TerminationConfirmedAt, err = parseNullableTime(terminationAt)
	if err != nil {
		return domain.StopOperation{}, err
	}
	operation.ResolvedAt, err = parseNullableTime(resolvedAt)
	return operation, err
}

func validStopTransition(from, to domain.StopStage) bool {
	if from == to {
		return true
	}
	return from == domain.StopRequested && (to == domain.StopCallSucceeded || to == domain.StopCallFailed || to == domain.StopTargetAbsent) ||
		from == domain.StopCallSucceeded && (to == domain.StopTerminationConfirmed || to == domain.StopTargetAbsent)
}

// UpdateStopOperationStage is a guarded compatibility entry point. Every
// effect/terminal transition delegates to the atomic D11/D12 transaction.
func (s *Store) UpdateStopOperationStage(ctx context.Context, operationID string, expected, next domain.StopStage, update StopStageUpdate) error {
	if !validStopTransition(expected, next) {
		return fmt.Errorf("%w: stop %q -> %q", ErrInvalidOperationTransition, expected, next)
	}
	if expected == domain.StopRequested && next == domain.StopCallSucceeded {
		if update.CallCompletedAt == nil || update.ConfirmationDeadlineAt == nil || update.ResolvedAt != nil || update.TerminationConfirmedAt != nil || update.ResolutionState != nil && *update.ResolutionState != domain.StopResolutionInFlight {
			return ErrStateConflict
		}
		return s.CommitStopCallAccepted(ctx, operationID, *update.CallCompletedAt, *update.ConfirmationDeadlineAt)
	}
	if expected == domain.StopCallSucceeded && next == domain.StopCallSucceeded && update.ResolutionState == nil {
		if update.CallCompletedAt == nil || update.ConfirmationDeadlineAt == nil || update.ResolvedAt != nil || update.TerminationConfirmedAt != nil {
			return ErrStateConflict
		}
		stop, err := s.GetStopOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if stop.Stage != domain.StopCallSucceeded || stop.ResolutionState != domain.StopResolutionInFlight || stop.CallCompletedAt == nil || stop.ConfirmationDeadlineAt == nil || !stop.CallCompletedAt.Equal(*update.CallCompletedAt) || !stop.ConfirmationDeadlineAt.Equal(*update.ConfirmationDeadlineAt) {
			return ErrStateConflict
		}
		return nil
	}
	if update.ResolutionState != nil && *update.ResolutionState == domain.StopResolutionInFlight {
		return ErrStateConflict
	}
	if update.ResolutionState == nil || update.ExpectedResolutionState != "" && update.ExpectedResolutionState != domain.StopResolutionInFlight || update.CallCompletedAt != nil || update.ConfirmationDeadlineAt != nil {
		return ErrAtomicDispatchRequired
	}
	at := timeNow()
	if update.ResolvedAt != nil {
		at = *update.ResolvedAt
	}
	if update.TerminationConfirmedAt != nil {
		at = *update.TerminationConfirmedAt
	}
	return s.CommitStopOutcome(ctx, operationID, StopTerminalOutcome{ExpectedStage: expected, Stage: next, Resolution: *update.ResolutionState, At: at, ObservedSessionID: update.ObservedSessionID, ObservedGeneration: update.ObservedGeneration, ObservedIsTerminated: update.ObservedIsTerminated})
}

func operationCASResult(result sql.Result, err error, operationID, kind string) error {
	if err != nil {
		return mapLifecycleWriteError(err, kind+" operation stage update")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: %s operation CAS rows affected: %w", kind, err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %s operation %q stage precondition failed", ErrStateConflict, kind, operationID)
	}
	return nil
}
