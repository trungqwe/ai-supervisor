package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"

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
//  2. Validator-compatible projection: converts json.Number values to exact Go
//     integer or exact float64 types so the jsonschema library can classify them
//     correctly. This projection is used ONLY for structural schema validation
//     and is NOT returned as application data.
//
// Returns the authoritative decoded value (with json.Number intact).
func ParseAndValidateRaw(rawJSON []byte, resolved *jsonschema.Resolved) (any, error) {
	if len(rawJSON) == 0 {
		return nil, newSchemaError("empty JSON input", nil)
	}

	// Finding R4-002: Canonical schema validator must never be optional
	if resolved == nil {
		return nil, newSchemaError("canonical schema validator is required and cannot be nil", nil)
	}

	// ── View 1: Authoritative parse ──────────────────────────────────────────
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
	projected, err := projectForValidator(authoritative)
	if err != nil {
		return nil, newSchemaError(fmt.Sprintf("numeric projection failed: %v", err), err)
	}
	if err := resolved.Validate(projected); err != nil {
		return nil, newSchemaError(fmt.Sprintf("schema validation failed: %v", err), err)
	}

	return authoritative, nil
}

// projectForValidator recursively converts json.Number values in a decoded JSON
// tree to Go types compatible with the jsonschema library type classifier:
//
//   - Mathematical integer that fits int64   → int64
//   - Mathematical integer that fits uint64  → uint64
//   - Mathematically exact float64           → float64
//   - Inexact decimal numbers                → deterministic precision error
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

// projectNumber converts a json.Number to int64, uint64, or mathematically exact float64.
// It parses the token into math/big.Rat as the authoritative mathematical representation.
// Inexact decimals fail closed to prevent silent rounding from bypassing schema constraints.
func projectNumber(n json.Number) (any, error) {
	s := n.String()

	rat := new(big.Rat)
	if _, ok := rat.SetString(s); !ok {
		return nil, fmt.Errorf("invalid numeric token %q", s)
	}

	// 19. Mathematical integer projection (denominator == 1)
	if rat.IsInt() {
		num := rat.Num()
		if num.IsInt64() {
			return num.Int64(), nil
		}
		if num.IsUint64() {
			return num.Uint64(), nil
		}
		return nil, fmt.Errorf("integer value %q cannot be safely projected for schema validation (out of int64/uint64 range)", s)
	}

	// 20. Non-integer float projection with exactness verification
	f64, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("cannot convert JSON number %q to float64: %w", s, err)
	}
	if math.IsInf(f64, 0) || math.IsNaN(f64) {
		return nil, fmt.Errorf("JSON number %q projects to non-finite float64", s)
	}

	f64Rat := new(big.Rat).SetFloat64(f64)
	if f64Rat == nil || rat.Cmp(f64Rat) != 0 {
		return nil, fmt.Errorf("numeric token %q cannot be represented exactly as float64 without precision loss", s)
	}

	return f64, nil
}
