package domain_test

import (
	"strings"
	"testing"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestDeriveHoldAndRejectionIDs(t *testing.T) {
	holdDesc := domain.HoldIdentityDescriptor{
		AttemptID:             "attempt-001",
		ContractID:            "contract-001",
		DiagnosticFingerprint: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		HoldReason:            string(domain.HoldReasonDirtyWorktreeDetected),
		OccurrenceNumber:      1,
		PairID:                "pair-001",
		TaskID:                "task-001",
	}

	holdID1, err := domain.DeriveHoldID(holdDesc)
	if err != nil {
		t.Fatalf("failed to derive hold_id: %v", err)
	}
	if !strings.HasPrefix(holdID1, "hold-") {
		t.Fatalf("hold_id must start with hold-, got %q", holdID1)
	}

	// Exact replay returns identical hold_id
	holdID2, err := domain.DeriveHoldID(holdDesc)
	if err != nil || holdID1 != holdID2 {
		t.Fatalf("expected exact replay to produce identical hold_id, got %q vs %q", holdID1, holdID2)
	}

	// Recurrence N=2 produces different hold_id
	holdDesc2 := holdDesc
	holdDesc2.OccurrenceNumber = 2
	holdID3, _ := domain.DeriveHoldID(holdDesc2)
	if holdID1 == holdID3 {
		t.Fatal("occurrence 2 must produce different hold_id")
	}

	// Derive rejection event ID
	rejDesc := domain.RejectionEventDescriptor{
		AttemptID:                 "attempt-001",
		ContractID:                "contract-001",
		DiagnosticFingerprint:     holdDesc.DiagnosticFingerprint,
		EventType:                 "EVIDENCE_COLLECTION_FAILED",
		HoldID:                    holdID1,
		OccurrenceNumber:          1,
		PairID:                    "pair-001",
		Reason:                    "DIRTY_WORKTREE_DETECTED",
		SanitizedInputFingerprint: "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		TaskID:                    "task-001",
	}

	eventID1, err := domain.DeriveRejectionEventID(rejDesc)
	if err != nil {
		t.Fatalf("failed to derive rejection_event_id: %v", err)
	}
	if len(eventID1) != 64 {
		t.Fatalf("expected 64 char hex event_id, got %d", len(eventID1))
	}

	// Variant B derivation (colliding_event_id absent)
	varBDesc := domain.RejectionEventDescriptorVariantB{
		AttemptID:                 "attempt-001",
		AttemptedReason:           "WORKSPACE_BINDING_MISSING",
		ContractID:                "contract-001",
		DiagnosticFingerprint:     holdDesc.DiagnosticFingerprint,
		DispatchOperationID:       "dispatch-op-001",
		HoldID:                    holdID1,
		OccurrenceNumber:          1,
		PairID:                    "pair-001",
		SanitizedInputFingerprint: rejDesc.SanitizedInputFingerprint,
		TaskID:                    "task-001",
	}
	varBID, err := domain.DeriveRejectionEventIDVariantB(varBDesc)
	if err != nil {
		t.Fatalf("failed to derive Variant B rejection event_id: %v", err)
	}
	if len(varBID) != 64 {
		t.Fatalf("expected 64 char hex varB event_id, got %d", len(varBID))
	}
}

func TestReviewIntegrityHold_Validate(t *testing.T) {
	fp := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hold := domain.ReviewIntegrityHold{
		HoldID:                "hold-1",
		TaskID:                "task-1",
		AttemptID:             "attempt-1",
		ContractID:            "contract-1",
		HoldReason:            domain.HoldReasonDirtyWorktreeDetected,
		HoldState:             domain.HoldStateActive,
		DiagnosticFingerprint: fp,
		OccurrenceNumber:      1,
		RejectionAuditEventID: "audit-rej-1",
		CreatedAtEpochMS:      1000,
	}

	if err := hold.Validate(); err != nil {
		t.Fatalf("expected valid hold, got: %v", err)
	}

	// Active hold with resolved fields
	badActive := hold
	now := int64(1500)
	badActive.ResolvedAtEpochMS = &now
	if err := badActive.Validate(); err == nil {
		t.Fatal("expected error on active hold with non-nil resolved_at")
	}

	// Resolved hold
	resHold := hold
	resHold.HoldState = domain.HoldStateResolved
	resAudit := "audit-res-1"
	principal := "operator-admin"
	resHold.ResolvedAtEpochMS = &now
	resHold.ResolutionAuditEventID = &resAudit
	resHold.ResolvedByPrincipal = &principal

	if err := resHold.Validate(); err != nil {
		t.Fatalf("expected valid resolved hold, got: %v", err)
	}
}
