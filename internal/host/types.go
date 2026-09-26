package host

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrSharingViolation        = errors.New("host: database lock contention (ERROR_SHARING_VIOLATION)")
	ErrHardlinkUnsupported     = errors.New("host: hard links unsupported (nNumberOfLinks > 1)")
	ErrIdentityMismatch        = errors.New("host: database physical identity mismatch")
	ErrInvalidPath             = errors.New("host: invalid database path (local DOS volume required)")
	ErrUnauthorizedCaller      = errors.New("host: unauthorized caller SID on named pipe")
	ErrMissingPolicy           = errors.New("host: missing required operational policy (UNSET policies fail closed)")
	ErrAutomaticRestoreBlocked = errors.New("host: automatic restore disabled fail closed")
	ErrPortClosedBeforeStartup = errors.New("host: port closed before startup completion")
	ErrDrainTimeout            = errors.New("host: shutdown drain timeout exceeded")
)

// OwnerMetadata defines the content written to <canonical_db_path>.owner.json.
type OwnerMetadata struct {
	OwnerPID        int       `json:"owner_pid"`
	OwnerInstanceID string    `json:"owner_instance_id"`
	PipeName        string    `json:"pipe_name"`
	StartedAt       time.Time `json:"started_at"`
	CanonicalDBPath string    `json:"canonical_db_path"`
}

// Policies encapsulates the 8 mandatory operational policies.
// None of these may default to numeric values; all must be injected at startup.
type Policies struct {
	SupervisorHTTPTimeout          time.Duration
	SupervisorHealthProbeTimeout   time.Duration
	SupervisorSpawnTimeout         time.Duration
	SupervisorSendTimeout          time.Duration
	SupervisorActivityPollInterval time.Duration
	SupervisorExecutionDeadline    time.Duration
	SupervisorKillStopTimeout      time.Duration
	SupervisorWorkspaceReadTimeout time.Duration
}

// Validate checks that all 8 policies have been explicitly configured with positive durations.
func (p *Policies) Validate() error {
	if p.SupervisorHTTPTimeout <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_HTTP_TIMEOUT must be positive", ErrMissingPolicy)
	}
	if p.SupervisorHealthProbeTimeout <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_HEALTH_PROBE_TIMEOUT must be positive", ErrMissingPolicy)
	}
	if p.SupervisorSpawnTimeout <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_SPAWN_TIMEOUT must be positive", ErrMissingPolicy)
	}
	if p.SupervisorSendTimeout <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_SEND_TIMEOUT must be positive", ErrMissingPolicy)
	}
	if p.SupervisorActivityPollInterval <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_ACTIVITY_POLL_INTERVAL must be positive", ErrMissingPolicy)
	}
	if p.SupervisorExecutionDeadline <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_EXECUTION_DEADLINE must be positive", ErrMissingPolicy)
	}
	if p.SupervisorKillStopTimeout <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_KILL_STOP_TIMEOUT must be positive", ErrMissingPolicy)
	}
	if p.SupervisorWorkspaceReadTimeout <= 0 {
		return fmt.Errorf("%w: SUPERVISOR_WORKSPACE_READ_TIMEOUT must be positive", ErrMissingPolicy)
	}
	return nil
}
