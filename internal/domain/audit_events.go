package domain

// Phase P04 Approved Audit Event Type Literals pursuant to accepted ADR-018.
const (
	AuditWorkspaceBindingCreated     = "WORKSPACE_BINDING_CREATED"
	AuditReviewIntegrityConflict     = "REVIEW_INTEGRITY_CONFLICT"
	AuditReviewIntegrityHoldResolved = "REVIEW_INTEGRITY_HOLD_RESOLVED"
	AuditEvidenceCollectionFailed    = "EVIDENCE_COLLECTION_FAILED"
)
