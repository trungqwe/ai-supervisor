package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

const (
	RedactedMarker       = "[REDACTED]"
	RedactedBinaryMarker = "[REDACTED_BINARY]"
)

var (
	ErrSanitizedKeyCollision = errors.New("audit: sanitized map key collision")
	ErrNonStringMapKey       = errors.New("audit: non-string map key is not supported")

	// sensitiveKeys defines map keys that must be unconditionally redacted.
	sensitiveKeys = map[string]struct{}{
		"password":            {},
		"passwd":              {},
		"secret":              {},
		"client_secret":       {},
		"api_key":             {},
		"apikey":              {},
		"token":               {},
		"access_token":        {},
		"refresh_token":       {},
		"id_token":            {},
		"session_token":       {},
		"authorization":       {},
		"proxy_authorization": {},
		"cookie":              {},
		"set_cookie":          {},
		"private_key":         {},
	}

	// Embedded credential patterns
	bearerRegex        = regexp.MustCompile("(?i)\\b(bearer\\s+)[A-Za-z0-9._~+/-]+=*")
	githubTokenRegex   = regexp.MustCompile("\\b(gh[opurs]_[A-Za-z0-9_]{16,}|github_pat_[A-Za-z0-9_]{16,})\\b")
	openAIRegex        = regexp.MustCompile("\\b(sk-(?:proj-)?[A-Za-z0-9_-]{16,})\\b")
	awsKeyRegex        = regexp.MustCompile("\\b(AKIA[0-9A-Z]{16})\\b")
	envAssignmentRegex = regexp.MustCompile("(?i)\\b(OPENAI_API_KEY|API_KEY|TOKEN|ACCESS_TOKEN|PASSWORD|CLIENT_SECRET)=([^\\s,;]+)")
)

func normalizeKey(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	k = strings.ReplaceAll(k, "-", "_")
	return k
}

// IsSensitiveKey returns true if the key is in the recognized set of sensitive structured keys.
func IsSensitiveKey(key string) bool {
	_, ok := sensitiveKeys[normalizeKey(key)]
	return ok
}

// ScrubString scrubs embedded credentials from an arbitrary string while preserving normal text.
func ScrubString(s string) string {
	if s == "" {
		return s
	}

	// 1. Bearer tokens: replace credential part with [REDACTED]
	s = bearerRegex.ReplaceAllString(s, "${1}"+RedactedMarker)

	// 2. Environment variable assignments: e.g. OPENAI_API_KEY=...
	s = envAssignmentRegex.ReplaceAllString(s, "${1}="+RedactedMarker)

	// 3. GitHub tokens
	s = githubTokenRegex.ReplaceAllString(s, RedactedMarker)

	// 4. OpenAI tokens
	s = openAIRegex.ReplaceAllString(s, RedactedMarker)

	// 5. AWS Access Key IDs
	s = awsKeyRegex.ReplaceAllString(s, RedactedMarker)

	return s
}

// ScrubValue recursively deep-copies and scrubs a value.
// It sanitizes arbitrary string map keys via ScrubString, redacts sensitive map keys,
// rejects post-scrub key collisions and non-string map keys, scrubs strings for token patterns,
// converts raw []byte to RedactedBinaryMarker, preserves numbers/booleans/null/json.Number,
// and ensures the caller's original data structure is never mutated.
func ScrubValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	switch val := v.(type) {
	case []byte:
		return RedactedBinaryMarker, nil
	case string:
		return ScrubString(val), nil
	case json.Number:
		return val, nil
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return val, nil
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			var outKey string
			if IsSensitiveKey(k) {
				outKey = k
			} else {
				outKey = ScrubString(k)
			}

			if _, exists := res[outKey]; exists {
				return nil, fmt.Errorf("%w: multiple input keys produced sanitized key %q", ErrSanitizedKeyCollision, outKey)
			}

			if IsSensitiveKey(k) {
				res[outKey] = RedactedMarker
			} else {
				scrubbed, err := ScrubValue(item)
				if err != nil {
					return nil, err
				}
				res[outKey] = scrubbed
			}
		}
		return res, nil
	case []any:
		res := make([]any, len(val))
		for i, item := range val {
			scrubbed, err := ScrubValue(item)
			if err != nil {
				return nil, err
			}
			res[i] = scrubbed
		}
		return res, nil
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			if rv.Type().Key().Kind() != reflect.String {
				return nil, fmt.Errorf("%w: map key type %v is not string", ErrNonStringMapKey, rv.Type().Key())
			}
			res := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				var outKey string
				if IsSensitiveKey(k) {
					outKey = k
				} else {
					outKey = ScrubString(k)
				}

				if _, exists := res[outKey]; exists {
					return nil, fmt.Errorf("%w: multiple input keys produced sanitized key %q", ErrSanitizedKeyCollision, outKey)
				}

				if IsSensitiveKey(k) {
					res[outKey] = RedactedMarker
				} else {
					scrubbed, err := ScrubValue(iter.Value().Interface())
					if err != nil {
						return nil, err
					}
					res[outKey] = scrubbed
				}
			}
			return res, nil
		case reflect.Slice, reflect.Array:
			res := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				scrubbed, err := ScrubValue(rv.Index(i).Interface())
				if err != nil {
					return nil, err
				}
				res[i] = scrubbed
			}
			return res, nil
		case reflect.Ptr:
			if rv.IsNil() {
				return nil, nil
			}
			return ScrubValue(rv.Elem().Interface())
		case reflect.Struct:
			// Marshal struct to json with UseNumber then scrub
			rawBytes, err := json.Marshal(val)
			if err != nil {
				return nil, fmt.Errorf("audit: failed to marshal struct: %w", err)
			}
			dec := json.NewDecoder(bytes.NewReader(rawBytes))
			dec.UseNumber()
			var decoded any
			if err := dec.Decode(&decoded); err != nil {
				return nil, fmt.Errorf("audit: failed to decode struct: %w", err)
			}
			return ScrubValue(decoded)
		default:
			return nil, fmt.Errorf("audit: unsupported detail value type: %T", v)
		}
	}
}

// SanitizeDetails scrubs a details map and serializes it to a deterministic canonical JSON string.
// If details is nil or empty, it returns an empty map and "{}" literal.
func SanitizeDetails(details map[string]any) (map[string]any, string, error) {
	if details == nil || len(details) == 0 {
		return make(map[string]any), "{}", nil
	}

	scrubbed, err := ScrubValue(details)
	if err != nil {
		return nil, "", err
	}

	scrubbedMap, ok := scrubbed.(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("audit: sanitized details root is not a map")
	}

	jsonBytes, err := json.Marshal(scrubbedMap)
	if err != nil {
		return nil, "", fmt.Errorf("audit: failed to marshal sanitized details: %w", err)
	}

	return scrubbedMap, string(jsonBytes), nil
}

// SanitizeAuditEvent returns a sanitized deep-copy of the input event and its canonical details_json.
// Identity fields are preserved without mutation. Actor, Details values, and Details keys are scrubbed.
// Timestamp is normalized to UTC. The original event is not modified.
func SanitizeAuditEvent(event domain.AuditEvent) (domain.AuditEvent, string, error) {
	sanitized := event

	// Normalize timestamp to UTC if set
	if !sanitized.Timestamp.IsZero() {
		sanitized.Timestamp = sanitized.Timestamp.UTC()
	}

	// Scrub actor
	sanitized.Actor = ScrubString(sanitized.Actor)

	// Sanitize details (both keys and values)
	scrubbedDetails, detailsJSON, err := SanitizeDetails(sanitized.Details)
	if err != nil {
		return domain.AuditEvent{}, "", err
	}
	sanitized.Details = scrubbedDetails

	return sanitized, detailsJSON, nil
}
