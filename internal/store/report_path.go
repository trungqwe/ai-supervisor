package store

import (
	"fmt"
	"strings"
)

// validateIdentitySegment verifies that an identity can be safely used as a single path segment
// in Windows and POSIX environments (ADR-011, SEC-002, Finding R4-002).
func validateIdentitySegment(name, val string) error {
	if val == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	if strings.TrimSpace(val) != val {
		return fmt.Errorf("%s must not contain leading or trailing whitespace", name)
	}
	// Windows strips trailing spaces and periods during path resolution; reject them to prevent alias collisions
	if strings.HasSuffix(val, " ") || strings.HasSuffix(val, ".") {
		return fmt.Errorf("%s must not end with trailing space or period", name)
	}
	if val == "." || val == ".." {
		return fmt.Errorf("%s must not be '.' or '..'", name)
	}
	if strings.Contains(val, "..") {
		return fmt.Errorf("%s contains path traversal sequence", name)
	}

	// Validate bytes for ASCII control characters and Windows forbidden characters
	for i := 0; i < len(val); i++ {
		b := val[i]
		// Control characters: 0x00 through 0x1F, and 0x7F (DEL)
		if b <= 0x1F || b == 0x7F {
			return fmt.Errorf("%s contains control character (0x%02X)", name, b)
		}
		// Windows forbidden characters: < > : " /  | ? *
		switch b {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return fmt.Errorf("%s contains Windows reserved character %q", name, b)
		}
	}

	// Windows reserved DOS device names (case-insensitive, even with extensions)
	// Derive the base name before the first period
	base := val
	if idx := strings.IndexByte(val, '.'); idx != -1 {
		base = val[:idx]
	}
	baseUpper := strings.ToUpper(base)
	switch baseUpper {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return fmt.Errorf("%s uses Windows reserved device name %q", name, base)
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
