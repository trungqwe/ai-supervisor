package contract

import (
	"fmt"
)

// ValidationError represents a categorized validation failure.
type ValidationError struct {
	Category string
	Field    string
	Message  string
	Cause    error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s] field %q: %s", e.Category, e.Field, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Cause
}

func newSchemaError(msg string, cause error) error {
	return &ValidationError{Category: "SCHEMA", Message: msg, Cause: cause}
}

func newLineageError(msg string) error {
	return &ValidationError{Category: "LINEAGE", Message: msg}
}

func newImmutabilityError(msg string) error {
	return &ValidationError{Category: "IMMUTABILITY", Message: msg}
}

func newProfileError(field, msg string) error {
	return &ValidationError{Category: "VERIFICATION_PROFILE", Field: field, Message: msg}
}

func newPathError(field, msg string) error {
	return &ValidationError{Category: "PATH_CONTAINMENT", Field: field, Message: msg}
}

func newScopeError(field, msg string) error {
	return &ValidationError{Category: "SCOPE_PATTERN", Field: field, Message: msg}
}
