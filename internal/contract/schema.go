package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/google/jsonschema-go/jsonschema"
)

// CompileSchema compiles raw JSON schema bytes into a resolved validator without network access.
func CompileSchema(schemaBytes []byte) (*jsonschema.Resolved, error) {
	if len(schemaBytes) == 0 {
		return nil, errors.New("schema bytes cannot be empty")
	}

	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON schema: %w", err)
	}

	// Resolve schema locally (nil options disables network loaders)
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve JSON schema: %w", err)
	}

	return resolved, nil
}

// CompileProfileSchema compiles parameter schema map into a resolved validator without network access.
func CompileProfileSchema(paramSchema map[string]any) (*jsonschema.Resolved, error) {
	if len(paramSchema) == 0 {
		return nil, nil
	}
	schemaBytes, err := json.Marshal(paramSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameter schema: %w", err)
	}
	return CompileSchema(schemaBytes)
}

// ParseAndValidateRaw parses raw JSON bytes, rejects malformed/trailing tokens,
// and validates the decoded value against the resolved JSON schema.
//
// Numbers are decoded as float64 (standard json.Unmarshal behavior) so that
// the jsonschema library can correctly classify integer-typed fields.
// The typed domain.TaskContract unmarshal happens separately in ValidateRaw.
func ParseAndValidateRaw(rawJSON []byte, resolved *jsonschema.Resolved) (any, error) {
	if len(rawJSON) == 0 {
		return nil, newSchemaError("empty JSON input", nil)
	}

	// First pass: detect and reject trailing content after the first JSON value.
	// UseNumber is NOT used here so that integer fields remain float64 (not json.Number strings)
	// and are correctly type-checked by the jsonschema library.
	decoder := json.NewDecoder(bytes.NewReader(rawJSON))
	var instance any
	if err := decoder.Decode(&instance); err != nil {
		return nil, newSchemaError(fmt.Sprintf("malformed JSON: %v", err), err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, newSchemaError("trailing content found after valid JSON value", nil)
	}

	// Validate decoded value against canonical Draft-07 schema
	if resolved != nil {
		if err := resolved.Validate(instance); err != nil {
			return nil, newSchemaError(fmt.Sprintf("schema validation failed: %v", err), err)
		}
	}

	return instance, nil
}
