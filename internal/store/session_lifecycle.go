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

// CreateWorkerSession inserts the current WorkerSession binding for a Pair.
func (s *Store) CreateWorkerSession(ctx context.Context, session domain.WorkerSession) error {
	if session.QuarantineState == "" {
		session.QuarantineState = domain.QuarantineClean
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = timeNow()
	}
	if session.UpdatedAt.IsZero() {
		session.UpdatedAt = session.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO worker_sessions (
    pair_id, session_id, runtime_type, worktree_path, worker_agent_id,
    status, terminal_generation, quarantine_state, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, session.PairID, session.SessionID, session.RuntimeType, nullableString(session.WorktreePath),
		session.WorkerAgentID, string(session.Status), session.TerminalGeneration,
		string(session.QuarantineState), formatTime(session.CreatedAt), formatTime(session.UpdatedAt))
	if err != nil {
		return mapLifecycleWriteError(err, "worker session")
	}
	return nil
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
	updatedAt := update.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = timeNow()
	}
	result, err := s.db.ExecContext(ctx, `
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
		if err := s.db.QueryRowContext(ctx, "SELECT 1 FROM worker_sessions WHERE pair_id = ?", pairID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return ErrWorkerSessionNotFound
		} else if err != nil {
			return fmt.Errorf("store: worker session CAS existence check: %w", err)
		}
		return fmt.Errorf("%w: worker session %q status/quarantine precondition failed", ErrStateConflict, pairID)
	}
	return nil
}

// CreatePairProvisioningOperation persists provisioning intent or outcome metadata.
func (s *Store) CreatePairProvisioningOperation(ctx context.Context, operation domain.PairProvisioningOperation) error {
	if operation.RequestedAt.IsZero() {
		operation.RequestedAt = timeNow()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO pair_provisioning_operations (
    operation_id, pair_id, stage, client_token, session_id, requested_at,
    completed_at, resolved_at, resolved_by, resolution_notes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, operation.OperationID, operation.PairID, string(operation.Stage), operation.ClientToken,
		nullableString(operation.SessionID), formatTime(operation.RequestedAt), nullableTime(operation.CompletedAt),
		nullableTime(operation.ResolvedAt), nullableString(operation.ResolvedBy), nullableString(operation.ResolutionNotes))
	if err != nil {
		return mapLifecycleWriteError(err, "pair provisioning operation")
	}
	return nil
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
	if !validProvisioningTransition(expected, next) {
		return fmt.Errorf("%w: provisioning %q -> %q", ErrInvalidOperationTransition, expected, next)
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE pair_provisioning_operations
SET stage = ?, session_id = COALESCE(?, session_id), completed_at = COALESCE(?, completed_at),
    resolved_at = COALESCE(?, resolved_at), resolved_by = COALESCE(?, resolved_by),
    resolution_notes = COALESCE(?, resolution_notes)
WHERE operation_id = ? AND stage = ?
`, string(next), nullableString(update.SessionID), nullableTime(update.CompletedAt), nullableTime(update.ResolvedAt),
		nullableString(update.ResolvedBy), nullableString(update.ResolutionNotes), operationID, string(expected))
	return operationCASResult(result, err, operationID, "pair provisioning")
}

// CreateDispatchOperation persists a dispatch operation after verifying attempt lineage and snapshot identity.
func (s *Store) CreateDispatchOperation(ctx context.Context, operation domain.DispatchOperation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin dispatch operation transaction: %w", err)
	}
	defer tx.Rollback()
	var taskID, pairID string
	var sessionID, terminalGeneration sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT a.task_id, t.pair_id, a.session_id, a.terminal_generation
FROM task_attempts a JOIN tasks t ON t.task_id = a.task_id
WHERE a.attempt_id = ? AND a.attempt_number = t.current_attempt
`, operation.AttemptID).Scan(&taskID, &pairID, &sessionID, &terminalGeneration)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAttemptNotFound
	}
	if err != nil {
		return fmt.Errorf("store: dispatch operation lineage query: %w", err)
	}
	if taskID != operation.TaskID || pairID != operation.PairID || !sessionID.Valid || !terminalGeneration.Valid ||
		sessionID.String != operation.SessionID || terminalGeneration.String != operation.TerminalGeneration {
		return fmt.Errorf("%w: dispatch operation does not match attempt snapshot", ErrAttemptLineageMismatch)
	}
	if operation.RequestedAt.IsZero() {
		operation.RequestedAt = timeNow()
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO dispatch_operations (
    operation_id, attempt_id, pair_id, task_id, session_id, terminal_generation,
    stage, requested_at, confirmed_at, resolution_state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, operation.OperationID, operation.AttemptID, operation.PairID, operation.TaskID, operation.SessionID,
		operation.TerminalGeneration, string(operation.Stage), formatTime(operation.RequestedAt),
		nullableTime(operation.ConfirmedAt), nullableString(operation.ResolutionState))
	if err != nil {
		return mapLifecycleWriteError(err, "dispatch operation")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit dispatch operation: %w", err)
	}
	return nil
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
	if !validDispatchTransition(expected, next) {
		return fmt.Errorf("%w: dispatch %q -> %q", ErrInvalidOperationTransition, expected, next)
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE dispatch_operations
SET stage = ?, confirmed_at = COALESCE(?, confirmed_at), resolution_state = COALESCE(?, resolution_state)
WHERE operation_id = ? AND stage = ?
`, string(next), nullableTime(update.ConfirmedAt), nullableString(update.ResolutionState), operationID, string(expected))
	return operationCASResult(result, err, operationID, "dispatch")
}

// CreateStopOperation persists a stop intent after validating optional lineage as one coherent tuple.
func (s *Store) CreateStopOperation(ctx context.Context, operation domain.StopOperation) error {
	if operation.ResolutionState == "" {
		operation.ResolutionState = domain.StopResolutionInFlight
	}
	if operation.RequestedAt.IsZero() {
		operation.RequestedAt = timeNow()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin stop operation transaction: %w", err)
	}
	defer tx.Rollback()
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
	_, err = tx.ExecContext(ctx, `
INSERT INTO stop_operations (
    operation_id, purpose, pair_id, task_id, contract_id, attempt_id, session_id,
    terminal_generation, stage, actor, requested_at, call_completed_at,
    confirmation_deadline_at, termination_confirmed_at, resolved_at, resolution_state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, operation.OperationID, string(operation.Purpose), operation.PairID, nullableString(operation.TaskID),
		nullableString(operation.ContractID), nullableString(operation.AttemptID), operation.SessionID,
		operation.TerminalGeneration, string(operation.Stage), operation.Actor, formatTime(operation.RequestedAt),
		nullableTime(operation.CallCompletedAt), nullableTime(operation.ConfirmationDeadlineAt),
		nullableTime(operation.TerminationConfirmedAt), nullableTime(operation.ResolvedAt), string(operation.ResolutionState))
	if err != nil {
		return mapLifecycleWriteError(err, "stop operation")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit stop operation: %w", err)
	}
	return nil
}

// GetStopOperation retrieves a stop operation by identity.
func (s *Store) GetStopOperation(ctx context.Context, operationID string) (domain.StopOperation, error) {
	var operation domain.StopOperation
	var purpose, stage, requestedAt, resolutionState string
	var taskID, contractID, attemptID sql.NullString
	var callCompletedAt, deadlineAt, terminationAt, resolvedAt sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT operation_id, purpose, pair_id, task_id, contract_id, attempt_id, session_id,
       terminal_generation, stage, actor, requested_at, call_completed_at,
       confirmation_deadline_at, termination_confirmed_at, resolved_at, resolution_state
FROM stop_operations WHERE operation_id = ?
`, operationID).Scan(&operation.OperationID, &purpose, &operation.PairID, &taskID, &contractID,
		&attemptID, &operation.SessionID, &operation.TerminalGeneration, &stage, &operation.Actor,
		&requestedAt, &callCompletedAt, &deadlineAt, &terminationAt, &resolvedAt, &resolutionState)
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

// UpdateStopOperationStage advances wire stage or records a same-stage resolution with CAS.
func (s *Store) UpdateStopOperationStage(ctx context.Context, operationID string, expected, next domain.StopStage, update StopStageUpdate) error {
	if !validStopTransition(expected, next) {
		return fmt.Errorf("%w: stop %q -> %q", ErrInvalidOperationTransition, expected, next)
	}
	if update.ExpectedResolutionState == "" {
		update.ExpectedResolutionState = domain.StopResolutionInFlight
	}
	if update.ExpectedResolutionState != domain.StopResolutionInFlight ||
		(update.ResolutionState != nil && *update.ResolutionState == domain.StopResolutionAdministrativeRiskAccepted) {
		return fmt.Errorf("%w: administrative or resolved stop requires separate audited reconciliation", ErrInvalidOperationTransition)
	}
	if next == domain.StopTerminationConfirmed &&
		(update.ResolutionState == nil || *update.ResolutionState != domain.StopResolutionTerminationConfirmed) {
		return fmt.Errorf("%w: stop termination confirmation requires matching resolution", ErrInvalidOperationTransition)
	}
	var resolution any
	if update.ResolutionState != nil {
		resolution = string(*update.ResolutionState)
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE stop_operations
SET stage = ?, call_completed_at = COALESCE(?, call_completed_at),
    confirmation_deadline_at = COALESCE(?, confirmation_deadline_at),
    termination_confirmed_at = COALESCE(?, termination_confirmed_at),
    resolved_at = COALESCE(?, resolved_at), resolution_state = COALESCE(?, resolution_state)
WHERE operation_id = ? AND stage = ? AND resolution_state = ?
  AND (? IS NULL OR call_completed_at IS NULL OR call_completed_at = ?)
  AND (? IS NULL OR confirmation_deadline_at IS NULL OR confirmation_deadline_at = ?)
`, string(next), nullableTime(update.CallCompletedAt), nullableTime(update.ConfirmationDeadlineAt),
		nullableTime(update.TerminationConfirmedAt), nullableTime(update.ResolvedAt), resolution,
		operationID, string(expected), string(update.ExpectedResolutionState),
		nullableTime(update.CallCompletedAt), nullableTime(update.CallCompletedAt),
		nullableTime(update.ConfirmationDeadlineAt), nullableTime(update.ConfirmationDeadlineAt))
	return operationCASResult(result, err, operationID, "stop")
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
