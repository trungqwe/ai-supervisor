package store

import (
	"fmt"
	"strings"
)

// validateIdentitySegment verifies that an identity can be safely used as a single path segment.
func validateIdentitySegment(name, val string) error {
	if val == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	if strings.TrimSpace(val) != val {
		return fmt.Errorf("%s must not contain leading or trailing whitespace", name)
	}
	if val == "." || val == ".." {
		return fmt.Errorf("%s must not be '.' or '..'", name)
	}
	if strings.ContainsAny(val, "/\\:\x00\r\n\t") {
		return fmt.Errorf("%s contains invalid path separator or control characters", name)
	}
	if strings.Contains(val, "..") {
		return fmt.Errorf("%s contains path traversal sequence", name)
	}
	return nil
}

// CanonicalExpectedReportPath derives the authoritative report path per ADR-011:
// .supervisor/reports/<task_id>/<attempt_id>.json
// The returned path strictly uses forward slashes regardless of the host OS.
func CanonicalExpectedReportPath(taskID, attemptID string) (string, error) {
	if err := validateIdentitySegment("taskID", taskID); err != nil {
		return "", fmt.Errorf("%w: %v", ErrReportPathMismatch, err)
	}
	if err := validateIdentitySegment("attemptID", attemptID); err != nil {
		return "", fmt.Errorf("%w: %v", ErrReportPathMismatch, err)
	}
	return fmt.Sprintf(".supervisor/reports/%s/%s.json", taskID, attemptID), nil
}
