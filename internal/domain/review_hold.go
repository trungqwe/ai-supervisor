package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// HoldReason represents enumerated root reasons for a review integrity hold.
type HoldReason string

const (
	HoldReasonDirtyWorktreeDetected   HoldReason = "DIRTY_WORKTREE_DETECTED"
	HoldReasonBundleHashConflict      HoldReason = "BUNDLE_HASH_CONFLICT"
	HoldReasonInvariantMismatch       HoldReason = "INVARIANT_MISMATCH"
	HoldReasonUnverifiedClaimDetected HoldReason = "UNVERIFIED_CLAIM_DETECTED"
	HoldReasonSecurityPolicyViolation HoldReason = "SECURITY_POLICY_VIOLATION"
)

// HoldState represents whether a hold is currently active or has been resolved.
type HoldState string

const (
	HoldStateActive   HoldState = "ACTIVE"
	HoldStateResolved HoldState = "RESOLVED"
)

var (
	hex64Regex = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ReviewIntegrityHold records a gate-blocking integrity hold.
type ReviewIntegrityHold struct {
	HoldID                 string     `json:"hold_id"`
	TaskID                 string     `json:"task_id"`
	AttemptID              string     `json:"attempt_id"`
	ContractID             string     `json:"contract_id"`
	HoldReason             HoldReason `json:"hold_reason"`
	HoldState              HoldState  `json:"hold_state"`
	DiagnosticFingerprint  string     `json:"diagnostic_fingerprint"`
	OccurrenceNumber       int        `json:"occurrence_number"`
	RejectionAuditEventID  string     `json:"rejection_audit_event_id"`
	ResolutionAuditEventID *string    `json:"resolution_audit_event_id,omitempty"`
	ResolvedByPrincipal    *string    `json:"resolved_by_principal,omitempty"`
	CreatedAtEpochMS       int64      `json:"created_at_epoch_ms"`
	ResolvedAtEpochMS      *int64     `json:"resolved_at_epoch_ms,omitempty"`
}

// Validate validates review integrity hold invariants.
func (h *ReviewIntegrityHold) Validate() error {
	if strings.TrimSpace(h.HoldID) == "" {
		return errors.New("hold_id must not be empty")
	}
	if strings.TrimSpace(h.TaskID) == "" {
		return errors.New("task_id must not be empty")
	}
	if strings.TrimSpace(h.AttemptID) == "" {
		return errors.New("attempt_id must not be empty")
	}
	if strings.TrimSpace(h.ContractID) == "" {
		return errors.New("contract_id must not be empty")
	}
	switch h.HoldReason {
	case HoldReasonDirtyWorktreeDetected, HoldReasonBundleHashConflict, HoldReasonInvariantMismatch,
		HoldReasonUnverifiedClaimDetected, HoldReasonSecurityPolicyViolation:
	default:
		return fmt.Errorf("invalid hold_reason %q", h.HoldReason)
	}
	if !hex64Regex.MatchString(h.DiagnosticFingerprint) {
		return fmt.Errorf("diagnostic_fingerprint must be 64 lowercase hex chars, got %q", h.DiagnosticFingerprint)
	}
	if h.OccurrenceNumber <= 0 {
		return errors.New("occurrence_number must be positive")
	}
	if strings.TrimSpace(h.RejectionAuditEventID) == "" {
		return errors.New("rejection_audit_event_id must not be empty")
	}
	if h.CreatedAtEpochMS <= 0 {
		return errors.New("created_at_epoch_ms must be positive")
	}
	switch h.HoldState {
	case HoldStateActive:
		if h.ResolvedAtEpochMS != nil || h.ResolutionAuditEventID != nil || h.ResolvedByPrincipal != nil {
			return errors.New("active hold must have nil resolution fields")
		}
	case HoldStateResolved:
		if h.ResolvedAtEpochMS == nil || h.ResolutionAuditEventID == nil || h.ResolvedByPrincipal == nil {
			return errors.New("resolved hold must have non-nil resolution fields")
		}
		if strings.TrimSpace(*h.ResolvedByPrincipal) == "" {
			return errors.New("resolved_by_principal must not be blank")
		}
		if *h.ResolvedAtEpochMS < h.CreatedAtEpochMS {
			return errors.New("resolved_at_epoch_ms must be >= created_at_epoch_ms")
		}
	default:
		return fmt.Errorf("invalid hold_state %q", h.HoldState)
	}
	return nil
}

// CanonicalJCS marshals data using RFC 8785 rules (sorted keys, no whitespace).
func CanonicalJCS(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

// ComputeFingerprint computes a 64-char lowercase hex SHA-256 of canonical JSON representation.
func ComputeFingerprint(v any) (string, error) {
	canonical, err := CanonicalJCS(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(canonical)
	return hex.EncodeToString(h[:]), nil
}

// HoldIdentityDescriptor defines Descriptor A for deriving hold_id.
type HoldIdentityDescriptor struct {
	AttemptID             string `json:"attempt_id"`
	ContractID            string `json:"contract_id"`
	DiagnosticFingerprint string `json:"diagnostic_fingerprint"`
	HoldReason            string `json:"hold_reason"`
	Kind                  string `json:"kind"`
	OccurrenceNumber      int    `json:"occurrence_number"`
	PairID                string `json:"pair_id"`
	TaskID                string `json:"task_id"`
	Version               int    `json:"version"`
}

// DeriveHoldID derives a deterministic hold_id from Descriptor A.
func DeriveHoldID(d HoldIdentityDescriptor) (string, error) {
	d.Kind = "review_integrity_hold"
	d.Version = 1
	fp, err := ComputeFingerprint(d)
	if err != nil {
		return "", fmt.Errorf("failed to derive hold_id: %w", err)
	}
	return "hold-" + fp, nil
}

// RejectionEventDescriptor defines Descriptor B for general rejection audit events.
type RejectionEventDescriptor struct {
	AttemptID                 string `json:"attempt_id"`
	ContractID                string `json:"contract_id"`
	DiagnosticFingerprint     string `json:"diagnostic_fingerprint"`
	EventType                 string `json:"event_type"`
	HoldID                    string `json:"hold_id"`
	OccurrenceNumber          int    `json:"occurrence_number"`
	PairID                    string `json:"pair_id"`
	Reason                    string `json:"reason"`
	SanitizedInputFingerprint string `json:"sanitized_input_fingerprint"`
	TaskID                    string `json:"task_id"`
	Version                   int    `json:"version"`
}

// DeriveRejectionEventID derives a deterministic rejection event_id from Descriptor B.
func DeriveRejectionEventID(d RejectionEventDescriptor) (string, error) {
	d.Version = 1
	fp, err := ComputeFingerprint(d)
	if err != nil {
		return "", fmt.Errorf("failed to derive rejection_event_id: %w", err)
	}
	return fp, nil
}

// RejectionEventDescriptorVariantB defines Descriptor B Variant B for WORKSPACE_BINDING_GUARD conflicts.
// Note: colliding_event_id is strictly absent.
type RejectionEventDescriptorVariantB struct {
	AttemptID                 string `json:"attempt_id"`
	AttemptedReason           string `json:"attempted_reason"`
	ConflictSource            string `json:"conflict_source"`
	ConflictType              string `json:"conflict_type"`
	ContractID                string `json:"contract_id"`
	DiagnosticFingerprint     string `json:"diagnostic_fingerprint"`
	DispatchOperationID       string `json:"dispatch_operation_id"`
	EventType                 string `json:"event_type"`
	HoldID                    string `json:"hold_id"`
	OccurrenceNumber          int    `json:"occurrence_number"`
	PairID                    string `json:"pair_id"`
	SanitizedInputFingerprint string `json:"sanitized_input_fingerprint"`
	TaskID                    string `json:"task_id"`
	Version                   int    `json:"version"`
}

// DeriveRejectionEventIDVariantB derives a deterministic event_id from Descriptor B Variant B.
func DeriveRejectionEventIDVariantB(d RejectionEventDescriptorVariantB) (string, error) {
	d.ConflictSource = "WORKSPACE_BINDING_GUARD"
	d.ConflictType = "LINEAGE_MISMATCH"
	d.EventType = "REVIEW_INTEGRITY_CONFLICT"
	d.Version = 2
	fp, err := ComputeFingerprint(d)
	if err != nil {
		return "", fmt.Errorf("failed to derive variant B rejection_event_id: %w", err)
	}
	return fp, nil
}

// ResolutionEventDescriptor defines Descriptor C for resolution audit events.
type ResolutionEventDescriptor struct {
	AttemptID                               string `json:"attempt_id"`
	ContractID                              string `json:"contract_id"`
	EventType                               string `json:"event_type"`
	HoldID                                  string `json:"hold_id"`
	Kind                                    string `json:"kind"`
	OccurrenceNumber                        int    `json:"occurrence_number"`
	PairID                                  string `json:"pair_id"`
	ResolvedByPrincipal                     string `json:"resolved_by_principal"`
	SanitizedResolutionRationaleFingerprint string `json:"sanitized_resolution_rationale_fingerprint"`
	TaskID                                  string `json:"task_id"`
	Version                                 int    `json:"version"`
}

// DeriveResolutionEventID derives a deterministic resolution event_id from Descriptor C.
func DeriveResolutionEventID(d ResolutionEventDescriptor) (string, error) {
	d.EventType = "REVIEW_INTEGRITY_HOLD_RESOLVED"
	d.Kind = "review_integrity_resolution_event"
	d.Version = 1
	fp, err := ComputeFingerprint(d)
	if err != nil {
		return "", fmt.Errorf("failed to derive resolution_event_id: %w", err)
	}
	return fp, nil
}
