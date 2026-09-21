package contract

import (
	"fmt"
	"reflect"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// ValidateRevisionLineage enforces monotonic contract revisions and baseline immutability.
func ValidateRevisionLineage(current, previous *domain.TaskContract) error {
	if current == nil {
		return newLineageError("current contract cannot be nil")
	}

	if current.RevisionNumber <= 0 {
		return newLineageError(fmt.Sprintf("invalid revision_number %d: must be positive", current.RevisionNumber))
	}

	if current.RevisionNumber == 1 {
		if current.SupersedesContractID != nil {
			return newLineageError(fmt.Sprintf("revision 1 must have null supersedes_contract_id, got %q", *current.SupersedesContractID))
		}
		if previous != nil {
			return newLineageError("revision 1 cannot supersede an existing contract")
		}
		return nil
	}

	// Revision N > 1
	if previous == nil {
		return newLineageError(fmt.Sprintf("revision %d requires a valid previous contract", current.RevisionNumber))
	}

	if current.TaskID != previous.TaskID {
		return newLineageError(fmt.Sprintf("task_id mismatch: current is %q, previous is %q", current.TaskID, previous.TaskID))
	}

	if current.ContractID == previous.ContractID {
		return newLineageError(fmt.Sprintf("contract_id must change across revisions, both are %q", current.ContractID))
	}

	if current.RevisionNumber != previous.RevisionNumber+1 {
		return newLineageError(fmt.Sprintf("revision jump rejected: current is %d, expected %d", current.RevisionNumber, previous.RevisionNumber+1))
	}

	if current.SupersedesContractID == nil {
		return newLineageError(fmt.Sprintf("revision %d must specify supersedes_contract_id matching previous contract %q", current.RevisionNumber, previous.ContractID))
	}

	if *current.SupersedesContractID != previous.ContractID {
		return newLineageError(fmt.Sprintf("supersedes_contract_id mismatch: got %q, expected previous %q", *current.SupersedesContractID, previous.ContractID))
	}

	if current.BaseSHA != previous.BaseSHA {
		return newLineageError(fmt.Sprintf("base_sha is immutable across revisions: current %q != previous %q", current.BaseSHA, previous.BaseSHA))
	}

	return nil
}

// ValidateImmutability checks that an immutable contract revision cannot have its specification altered.
func ValidateImmutability(original, candidate *domain.TaskContract) error {
	if original == nil || candidate == nil {
		return nil
	}

	if original.ContractID != candidate.ContractID {
		// New contract_id represents a different revision, not a mutation of original
		return nil
	}

	if !original.IsImmutable {
		return nil
	}

	// Compare SupersedesContractID
	var supersedesMatch bool
	if original.SupersedesContractID == nil && candidate.SupersedesContractID == nil {
		supersedesMatch = true
	} else if original.SupersedesContractID != nil && candidate.SupersedesContractID != nil {
		supersedesMatch = (*original.SupersedesContractID == *candidate.SupersedesContractID)
	} else {
		supersedesMatch = false
	}

	// Check all canonical serialized specification fields:
	// TaskID, RevisionNumber, SupersedesContractID, PhaseID, Objective, Requirements,
	// ArchitectureRefs, BaseSHA, AllowedScope, ForbiddenScope, Constraints,
	// AcceptanceCriteria, VerificationRequests, RequiredEvidence, WorkerProfile,
	// ReportContract, StopConditions.
	if !supersedesMatch ||
		original.TaskID != candidate.TaskID ||
		original.RevisionNumber != candidate.RevisionNumber ||
		original.PhaseID != candidate.PhaseID ||
		original.Objective != candidate.Objective ||
		original.BaseSHA != candidate.BaseSHA ||
		original.WorkerProfile != candidate.WorkerProfile ||
		original.ReportContract != candidate.ReportContract ||
		!reflect.DeepEqual(original.Requirements, candidate.Requirements) ||
		!reflect.DeepEqual(original.ArchitectureRefs, candidate.ArchitectureRefs) ||
		!reflect.DeepEqual(original.AllowedScope, candidate.AllowedScope) ||
		!reflect.DeepEqual(original.ForbiddenScope, candidate.ForbiddenScope) ||
		!reflect.DeepEqual(original.Constraints, candidate.Constraints) ||
		!reflect.DeepEqual(original.AcceptanceCriteria, candidate.AcceptanceCriteria) ||
		!reflect.DeepEqual(original.VerificationRequests, candidate.VerificationRequests) ||
		!reflect.DeepEqual(original.RequiredEvidence, candidate.RequiredEvidence) ||
		!reflect.DeepEqual(original.StopConditions, candidate.StopConditions) {
		return newImmutabilityError(fmt.Sprintf("contract %q is immutable; specification fields cannot be modified", original.ContractID))
	}

	return nil
}
