package store

import (
	"errors"
)

var (
	ErrProjectNotFound            = errors.New("store: project not found")
	ErrPairNotFound               = errors.New("store: pair not found")
	ErrTaskNotFound               = errors.New("store: task not found")
	ErrContractNotFound           = errors.New("store: task contract not found")
	ErrAttemptNotFound            = errors.New("store: task attempt not found")
	ErrTaskNotReady               = errors.New("store: task is not in READY state")
	ErrContractNotOwned           = errors.New("store: contract does not belong to specified task")
	ErrContractImmutable          = errors.New("store: contract is immutable")
	ErrStateConflict              = errors.New("store: task state conflict")
	ErrInconsistentPersistedState = errors.New("store: inconsistent persisted state")
	ErrDuplicateKey               = errors.New("store: duplicate key constraint violation")
	ErrForeignKeyViolation        = errors.New("store: foreign key constraint violation")
	ErrUnsupportedSchemaVersion   = errors.New("store: unsupported database schema version")
	ErrWorkerSessionNotFound      = errors.New("store: worker session not found")
	ErrOperationNotFound          = errors.New("store: lifecycle operation not found")
	ErrInvalidOperationTransition = errors.New("store: invalid lifecycle operation transition")
	ErrAttemptLineageMismatch     = errors.New("store: task attempt lineage mismatch")

	// Revision 2 additions
	ErrAtomicDispatchRequired  = errors.New("store: READY -> DISPATCHED transition requires PrepareDispatch atomic allocation")
	ErrInvalidInitialTaskState = errors.New("store: initial task state must be DRAFT with current_attempt 0")
	ErrReportPathMismatch      = errors.New("store: expected_report_path does not match canonical path")
	ErrStaleContractRevision   = errors.New("store: contract revision is not the latest revision for the task")
	ErrInvalidContractLineage  = errors.New("store: invalid contract revision lineage")
	ErrPairBusy                = errors.New("store: pair already has an active task in DISPATCHED, RUNNING, or REVIEWING")

	// Revision 3 additions
	ErrReadyContractRequired = errors.New("store: ready state requires governing task contract")
	ErrContractInsertState   = errors.New("store: contract cannot be inserted in current task lifecycle state")

	// TASK-P02-004 Audit Core additions
	ErrAuditChainInvalid   = errors.New("store: audit chain is invalid")
	ErrInvalidAuditLineage = errors.New("store: invalid audit event lineage")
	ErrInvalidLimit        = errors.New("store: invalid limit")
	ErrAuditEventNotFound  = errors.New("store: audit event not found")
)
