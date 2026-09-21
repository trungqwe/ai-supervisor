package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// TaskContractValidator performs structural schema validation, revision lineage validation,
// verification profile validation, and secure pre-dispatch path containment.
type TaskContractValidator struct {
	canonicalSchema *jsonschema.Resolved
	catalog         domain.VerificationPolicyCatalog
}

// NewValidator creates a TaskContractValidator with pre-compiled Draft-07 schema and policy catalog.
func NewValidator(canonicalSchemaBytes []byte, catalog domain.VerificationPolicyCatalog) (*TaskContractValidator, error) {
	var resolved *jsonschema.Resolved
	if len(canonicalSchemaBytes) > 0 {
		var err error
		resolved, err = CompileSchema(canonicalSchemaBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to compile canonical task contract schema: %w", err)
		}
	}
	return &TaskContractValidator{
		canonicalSchema: resolved,
		catalog:         catalog,
	}, nil
}

// ValidateRaw validates raw JSON bytes structurally against the schema,
// decodes into domain.TaskContract preserving json.Number in Parameters,
// and executes semantic validation.
func (v *TaskContractValidator) ValidateRaw(rawJSON []byte, previous *domain.TaskContract, worktreeRoot string) (*domain.TaskContract, error) {
	// Step 1: Two-view parse — authoritative UseNumber decode + validator projection.
	// ParseAndValidateRaw returns the authoritative decoded value (json.Number intact).
	// Schema validation is performed internally via the validator-compatible projection.
	if _, err := ParseAndValidateRaw(rawJSON, v.canonicalSchema); err != nil {
		return nil, err
	}

	// Step 2: Typed domain decode using UseNumber so that Parameters map[string]any
	// retains json.Number rather than silently coercing to float64.
	dec := json.NewDecoder(bytes.NewReader(rawJSON))
	dec.UseNumber()
	var contract domain.TaskContract
	if err := dec.Decode(&contract); err != nil {
		return nil, newSchemaError(fmt.Sprintf("failed to unmarshal into TaskContract: %v", err), err)
	}
	// Reject trailing content in the typed decode as well
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, newSchemaError("trailing content found after valid JSON value (typed decode)", nil)
	}

	// Step 3: Semantic validations
	if err := v.ValidateContract(&contract, previous, worktreeRoot); err != nil {
		return nil, err
	}

	return &contract, nil
}

// ValidateContract executes all semantic validations on an instantiated TaskContract.
func (v *TaskContractValidator) ValidateContract(contract *domain.TaskContract, previous *domain.TaskContract, worktreeRoot string) error {
	if contract == nil {
		return newSchemaError("contract cannot be nil", nil)
	}

	// 1. Revision Lineage
	if err := ValidateRevisionLineage(contract, previous); err != nil {
		return err
	}

	// 2. Immutability Check
	if previous != nil && previous.IsImmutable {
		if err := ValidateImmutability(previous, contract); err != nil {
			return err
		}
	}

	// 3. Scope pattern pre-dispatch validation
	if err := ValidateScopePatterns(contract.AllowedScope); err != nil {
		return err
	}
	if err := ValidateScopePatterns(contract.ForbiddenScope); err != nil {
		return err
	}

	// 4. Verification Requests validation against policy catalog & worktree
	if err := v.ValidateVerificationRequests(contract.VerificationRequests, worktreeRoot); err != nil {
		return err
	}

	return nil
}

// ValidateVerificationRequests validates each VerificationRequest against catalog policy,
// parameter schema, timeout, and cwd containment.
func (v *TaskContractValidator) ValidateVerificationRequests(requests []domain.VerificationRequest, worktreeRoot string) error {
	for i, req := range requests {
		if req.ID == "" {
			return newProfileError(fmt.Sprintf("verification_requests[%d].id", i), "request id cannot be empty")
		}
		if req.ProfileID == "" {
			return newProfileError(fmt.Sprintf("verification_requests[%d].profile_id", i), "profile_id cannot be empty")
		}

		if v.catalog == nil {
			return newProfileError(req.ProfileID, "policy catalog is not configured")
		}

		policy, found := v.catalog.LookupProfile(req.ProfileID)
		if !found {
			return newProfileError(req.ProfileID, fmt.Sprintf("profile %q not found in policy catalog", req.ProfileID))
		}

		// Timeout check
		if req.TimeoutSeconds < 0 {
			return newProfileError(fmt.Sprintf("verification_requests[%d].timeout_seconds", i), "timeout cannot be negative")
		}
		if req.TimeoutSeconds > 0 && policy.MaxTimeoutSeconds > 0 && req.TimeoutSeconds > policy.MaxTimeoutSeconds {
			return newProfileError(
				fmt.Sprintf("verification_requests[%d].timeout_seconds", i),
				fmt.Sprintf("requested timeout %d exceeds profile maximum %d", req.TimeoutSeconds, policy.MaxTimeoutSeconds),
			)
		}

		// Parameter schema validation
		// Parameters may contain json.Number; project them to validator-compatible types.
		if policy.ParameterSchema != nil {
			resolvedParamSchema, err := CompileProfileSchema(policy.ParameterSchema)
			if err != nil {
				return newProfileError(req.ProfileID, fmt.Sprintf("invalid parameter schema in profile %q: %v", req.ProfileID, err))
			}
			if resolvedParamSchema != nil {
				// Project parameters for schema validation (json.Number → int64/uint64/float64).
				// The authoritative req.Parameters (with json.Number) is NOT mutated.
				projectedParams, err := projectForValidator(req.Parameters)
				if err != nil {
					return newProfileError(
						fmt.Sprintf("verification_requests[%d].parameters", i),
						fmt.Sprintf("parameter numeric projection failed for profile %q: %v", req.ProfileID, err),
					)
				}
				if err := resolvedParamSchema.Validate(projectedParams); err != nil {
					return newProfileError(
						fmt.Sprintf("verification_requests[%d].parameters", i),
						fmt.Sprintf("parameter validation failed for profile %q: %v", req.ProfileID, err),
					)
				}
			}
		}

		// Cwd containment validation
		cwdPolicy := policy.CwdPolicy
		if cwdPolicy == "" {
			cwdPolicy = "worktree_root"
		}
		if err := ValidateCwdContainment(req.Cwd, worktreeRoot, cwdPolicy); err != nil {
			return err
		}
	}

	return nil
}
