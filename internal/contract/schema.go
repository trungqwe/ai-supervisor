package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

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

// ParseAndValidateRaw performs a two-view parse of raw JSON bytes:
//
//  1. Authoritative parse (decoder.UseNumber): preserves JSON numeric tokens
//     as json.Number; rejects malformed JSON and trailing content.
//
//  2. Validator-compatible projection: converts json.Number values to Go
//     integer or float64 types so the jsonschema library can classify them
//     correctly. This projection is used ONLY for structural schema validation
//     and is NOT returned as application data.
//
// Returns the authoritative decoded value (with json.Number intact).
func ParseAndValidateRaw(rawJSON []byte, resolved *jsonschema.Resolved) (any, error) {
	if len(rawJSON) == 0 {
		return nil, newSchemaError("empty JSON input", nil)
	}

	// ── View 1: Authoritative parse ─────────────────────────────────────────
	// UseNumber() preserves JSON integer tokens as json.Number (exact decimal).
	authDecoder := json.NewDecoder(bytes.NewReader(rawJSON))
	authDecoder.UseNumber()

	var authoritative any
	if err := authDecoder.Decode(&authoritative); err != nil {
		return nil, newSchemaError(fmt.Sprintf("malformed JSON: %v", err), err)
	}
	// Reject trailing tokens after the first JSON value
	var trailing any
	if err := authDecoder.Decode(&trailing); err != io.EOF {
		return nil, newSchemaError("trailing content found after valid JSON value", nil)
	}

	// ── View 2: Validator-compatible projection ──────────────────────────────
	// Convert json.Number to int64/uint64/float64 so jsonschema can classify
	// "integer" and "number" types correctly. The projection is not returned.
	if resolved != nil {
		projected, err := projectForValidator(authoritative)
		if err != nil {
			return nil, newSchemaError(fmt.Sprintf("numeric projection failed: %v", err), err)
		}
		if err := resolved.Validate(projected); err != nil {
			return nil, newSchemaError(fmt.Sprintf("schema validation failed: %v", err), err)
		}
	}

	return authoritative, nil
}

// projectForValidator recursively converts json.Number values in a decoded JSON
// tree to Go types compatible with the jsonschema library type classifier:
//
//   - Integral json.Number that fits int64   → int64
//   - Integral json.Number that fits uint64  → uint64
//   - Non-integral json.Number               → float64 (verified finite)
//   - Other scalar types                     → unchanged
//   - map[string]any                         → recursively projected
//   - []any                                  → recursively projected
func projectForValidator(v any) (any, error) {
	switch val := v.(type) {
	case json.Number:
		return projectNumber(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			p, err := projectForValidator(vv)
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", k, err)
			}
			out[k] = p
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			p, err := projectForValidator(vv)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			out[i] = p
		}
		return out, nil
	default:
		return v, nil
	}
}

// projectNumber converts a json.Number to int64, uint64, or float64 for
// schema validation purposes only. The original json.Number is preserved in
// the authoritative decoded tree.
func projectNumber(n json.Number) (any, error) {
	s := n.String()

	// Try integer representation first (no decimal point, no exponent notation
	// that would imply a fractional part in normal use).
	if isIntegerLexical(s) {
		// Try int64
		if i64, err := n.Int64(); err == nil {
			return i64, nil
		}
		// Try uint64 for non-negative integers that overflow int64
		if s[0] != '-' {
			var u64 uint64
			if _, err := fmt.Sscanf(s, "%d", &u64); err == nil {
				return u64, nil
			}
		}
		// Integer too large for int64 or uint64 — reject rather than corrupt
		return nil, fmt.Errorf("integer value %q cannot be safely projected for schema validation (out of int64/uint64 range)", s)
	}

	// Non-integral: convert to float64
	f64, err := n.Float64()
	if err != nil {
		return nil, fmt.Errorf("cannot convert JSON number %q to float64: %w", s, err)
	}
	if math.IsInf(f64, 0) || math.IsNaN(f64) {
		return nil, fmt.Errorf("JSON number %q projects to non-finite float64", s)
	}
	return f64, nil
}

// isIntegerLexical returns true when the JSON number string contains no
// decimal point or fractional exponent notation, treating it as an integer.
func isIntegerLexical(s string) bool {
	for _, c := range s {
		if c == '.' || c == 'e' || c == 'E' {
			return false
		}
	}
	return true
}
