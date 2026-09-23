package domain

import "time"

// WorkerSessionStatus is the Supervisor-owned runtime lifecycle status.
type WorkerSessionStatus string

const (
	WorkerSessionActive     WorkerSessionStatus = "ACTIVE"
	WorkerSessionIdle       WorkerSessionStatus = "IDLE"
	WorkerSessionTerminated WorkerSessionStatus = "TERMINATED"
)

// QuarantineState is the independent dispatch and allocation safety gate.
type QuarantineState string

const (
	QuarantineClean       QuarantineState = "CLEAN"
	QuarantineQuarantined QuarantineState = "QUARANTINED"
)

// PairProvisioningStage records durable session provisioning progress.
type PairProvisioningStage string

const (
	ProvisionRequested PairProvisioningStage = "PROVISION_REQUESTED"
	ProvisionConfirmed PairProvisioningStage = "PROVISION_CONFIRMED"
	ProvisionFailed    PairProvisioningStage = "PROVISION_FAILED"
	ProvisionResolved  PairProvisioningStage = "PROVISION_RESOLVED"
)

// DispatchStage records durable task dispatch progress.
type DispatchStage string

const (
	DispatchBound DispatchStage = "DISPATCH_BOUND"
	SendRequested DispatchStage = "SEND_REQUESTED"
	SendConfirmed DispatchStage = "SEND_CONFIRMED"
)

// StopPurpose distinguishes live attempt stops from cleanup and maintenance.
type StopPurpose string

const (
	RunningAttemptStop StopPurpose = "RUNNING_ATTEMPT_STOP"
	QuarantineCleanup  StopPurpose = "QUARANTINE_CLEANUP"
	PairMaintenance    StopPurpose = "PAIR_MAINTENANCE"
)

// StopStage preserves the physical wire-effect fact for a stop operation.
type StopStage string

const (
	StopRequested            StopStage = "STOP_REQUESTED"
	StopCallSucceeded        StopStage = "STOP_CALL_SUCCEEDED"
	StopCallFailed           StopStage = "STOP_CALL_FAILED"
	StopTerminationConfirmed StopStage = "STOP_TERMINATION_CONFIRMED"
	StopTargetAbsent         StopStage = "STOP_TARGET_ABSENT"
)

// StopResolutionState records stop reconciliation independently from wire stage.
type StopResolutionState string

const (
	StopResolutionInFlight                        StopResolutionState = "IN_FLIGHT"
	StopResolutionTerminationConfirmed            StopResolutionState = "TERMINATION_CONFIRMED"
	StopResolutionTargetAbsent                    StopResolutionState = "STOP_TARGET_ABSENT"
	StopResolutionConfirmationTimeout             StopResolutionState = "STOP_CONFIRMATION_TIMEOUT"
	StopResolutionGenerationMismatch              StopResolutionState = "STOP_GENERATION_MISMATCH"
	StopResolutionEffectUnprovenAlreadyTerminated StopResolutionState = "STOP_EFFECT_UNPROVEN_TARGET_ALREADY_TERMINATED"
	StopResolutionReissueRequiresHuman            StopResolutionState = "STOP_REISSUE_REQUIRES_HUMAN"
	StopResolutionCallFailed                      StopResolutionState = "STOP_CALL_FAILED"
	StopResolutionCallOutcomeUnknown              StopResolutionState = "STOP_CALL_OUTCOME_UNKNOWN"
	StopResolutionAdministrativeRiskAccepted      StopResolutionState = "ADMINISTRATIVE_RISK_ACCEPTED"
)

const (
	RecoveryAOBlockedEscalated          = "AO_BLOCKED_ESCALATED"
	AuditTaskDispatchBound              = "TASK_DISPATCH_BOUND"
	AuditTaskStateTransition            = "TASK_STATE_TRANSITION"
	AuditWorkerBlockedEscalated         = "WORKER_BLOCKED_ESCALATED"
	AuditPairRestoreRiskAccepted        = "PAIR_RESTORE_RISK_ACCEPTED"
	AuditPairRestoreRequested           = "PAIR_RESTORE_REQUESTED"
	AuditPairRestoreConfirmed           = "PAIR_RESTORE_CONFIRMED"
	AuditPairRestoreOutcomeUnknown      = "PAIR_RESTORE_OUTCOME_UNKNOWN"
	AuditPairRestoreRecoveryClaimed     = "PAIR_RESTORE_RECOVERY_CLAIMED"
	AuditPairRestoreRecoveryTransferred = "PAIR_RESTORE_RECOVERY_TRANSFERRED"
	AuditPairRestoreCleanupClaimed      = "PAIR_RESTORE_CLEANUP_CLAIMED"
	AuditPairRestoreResolved            = "PAIR_RESTORE_RESOLVED"
	AuditPreSendStatusRecovered         = "PRE_SEND_STATUS_RECOVERED"
	RecoveryDeliveryOutcomeUnknown      = "DELIVERY_OUTCOME_UNKNOWN"
	RecoveryUncertainDeliveryCrash      = "UNCERTAIN_DELIVERY_CRASH"
	AuditStopOperationRequested         = "STOP_OPERATION_REQUESTED"
)

type RestoreResolutionState string

const (
	RestoreInFlight        RestoreResolutionState = "IN_FLIGHT"
	RestoreOutcomeUnknown  RestoreResolutionState = "RESTORE_OUTCOME_UNKNOWN"
	RestoreRecoveryClaimed RestoreResolutionState = "RESTORE_RECOVERY_CLAIMED"
	RestoreCleanupClaimed  RestoreResolutionState = "RESTORE_CLEANUP_CLAIMED"
	RestoreResolved        RestoreResolutionState = "RESTORE_RESOLVED"
)

type RestoreResolutionBasis string

const (
	RestoreHTTP200Confirmed      RestoreResolutionBasis = "RESTORE_HTTP_200_CONFIRMED"
	PhysicalExecutionResolution  RestoreResolutionBasis = "PHYSICAL_EXECUTION_RESOLUTION"
	AdministrativeRiskResolution RestoreResolutionBasis = "ADMINISTRATIVE_RISK_RESOLUTION"
)

type RestoreAuthorization struct {
	AuthorizationID       string     `json:"authorization_id"`
	OperationID           string     `json:"operation_id"`
	PairID                string     `json:"pair_id"`
	SessionID             string     `json:"session_id"`
	ExpectedGeneration    string     `json:"expected_generation"`
	RiskScope             string     `json:"risk_scope"`
	AuthorizedPrincipal   string     `json:"authorized_principal"`
	AuthorizationEventID  string     `json:"authorization_event_id"`
	IssuedAt              time.Time  `json:"issued_at"`
	ConsumedAt            *time.Time `json:"consumed_at,omitempty"`
	ConsumedByOperationID *string    `json:"consumed_by_operation_id,omitempty"`
}

type PairRestoreOperation struct {
	OperationID        string                  `json:"operation_id"`
	AuthorizationID    string                  `json:"authorization_id"`
	PairID             string                  `json:"pair_id"`
	SessionID          string                  `json:"session_id"`
	ExpectedGeneration string                  `json:"expected_generation"`
	ObservedGeneration *string                 `json:"observed_generation,omitempty"`
	RestoreMode        *string                 `json:"restore_mode,omitempty"`
	Stage              string                  `json:"stage"`
	ResolutionState    RestoreResolutionState  `json:"resolution_state"`
	ResolutionBasis    *RestoreResolutionBasis `json:"resolution_basis,omitempty"`
	RecoveryPrincipal  *string                 `json:"recovery_principal,omitempty"`
	RecoveryClaimedAt  *time.Time              `json:"recovery_claimed_at,omitempty"`
	RequestedAt        time.Time               `json:"requested_at"`
	HTTP200At          *time.Time              `json:"http_200_at,omitempty"`
	ResolvedAt         *time.Time              `json:"resolved_at,omitempty"`
	ResolutionEventID  *string                 `json:"resolution_event_id,omitempty"`
	Version            int64                   `json:"version"`
}

type RestoreReservation struct {
	AuthorizationID     string
	OperationID         string
	PairID              string
	SessionID           string
	ExpectedGeneration  string
	RiskScope           string
	AuthorizedPrincipal string
	Actor               string
	At                  time.Time
}

// WorkerSession is the current AO session binding for exactly one Pair.
type WorkerSession struct {
	PairID             string              `json:"pair_id"`
	SessionID          string              `json:"session_id"`
	RuntimeType        string              `json:"runtime_type"`
	WorktreePath       *string             `json:"worktree_path,omitempty"`
	WorkerAgentID      string              `json:"worker_agent_id"`
	Status             WorkerSessionStatus `json:"status"`
	TerminalGeneration string              `json:"terminal_generation"`
	QuarantineState    QuarantineState     `json:"quarantine_state"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

// PairProvisioningOperation records pre-effect provisioning intent and resolution.
type PairProvisioningOperation struct {
	OperationID     string                `json:"operation_id"`
	PairID          string                `json:"pair_id"`
	Stage           PairProvisioningStage `json:"stage"`
	ClientToken     string                `json:"client_token"`
	SessionID       *string               `json:"session_id,omitempty"`
	RequestedAt     time.Time             `json:"requested_at"`
	CompletedAt     *time.Time            `json:"completed_at,omitempty"`
	ResolvedAt      *time.Time            `json:"resolved_at,omitempty"`
	ResolvedBy      *string               `json:"resolved_by,omitempty"`
	ResolutionNotes *string               `json:"resolution_notes,omitempty"`
}

// DispatchOperation binds one durable dispatch saga to exactly one TaskAttempt.
type DispatchOperation struct {
	OperationID        string        `json:"operation_id"`
	AttemptID          string        `json:"attempt_id"`
	PairID             string        `json:"pair_id"`
	TaskID             string        `json:"task_id"`
	SessionID          string        `json:"session_id"`
	TerminalGeneration string        `json:"terminal_generation"`
	Stage              DispatchStage `json:"stage"`
	RequestedAt        time.Time     `json:"requested_at"`
	ConfirmedAt        *time.Time    `json:"confirmed_at,omitempty"`
	ResolutionState    *string       `json:"resolution_state,omitempty"`
}

// StopOperation records a purpose-aware stop wire effect and its reconciliation.
type StopOperation struct {
	OperationID            string              `json:"operation_id"`
	Purpose                StopPurpose         `json:"purpose"`
	PairID                 string              `json:"pair_id"`
	TaskID                 *string             `json:"task_id,omitempty"`
	ContractID             *string             `json:"contract_id,omitempty"`
	AttemptID              *string             `json:"attempt_id,omitempty"`
	SessionID              string              `json:"session_id"`
	TerminalGeneration     string              `json:"terminal_generation"`
	Stage                  StopStage           `json:"stage"`
	Actor                  string              `json:"actor"`
	RequestedAt            time.Time           `json:"requested_at"`
	CallCompletedAt        *time.Time          `json:"call_completed_at,omitempty"`
	ConfirmationDeadlineAt *time.Time          `json:"confirmation_deadline_at,omitempty"`
	TerminationConfirmedAt *time.Time          `json:"termination_confirmed_at,omitempty"`
	ResolvedAt             *time.Time          `json:"resolved_at,omitempty"`
	ResolutionState        StopResolutionState `json:"resolution_state"`
	RestoreOperationID     *string             `json:"restore_operation_id,omitempty"`
	RestorePrincipal       *string             `json:"restore_principal,omitempty"`
}
