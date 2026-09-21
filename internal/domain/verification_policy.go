package domain

// VerificationProfilePolicy defines declarative profile validation metadata required by Phase P02.
// Per ADR-013, it exposes parameter schema, cwd constraints, and maximum timeout.
// It strictly does NOT expose executable paths, raw arguments, or shell options.
type VerificationProfilePolicy struct {
	ProfileID           string         `json:"profile_id"`
	ParameterSchema     map[string]any `json:"parameter_schema,omitempty"`
	CwdPolicy           string         `json:"cwd_policy,omitempty"`
	MaxTimeoutSeconds   int            `json:"max_timeout_seconds"`
	AllowedCapabilities []string       `json:"allowed_capabilities,omitempty"`
}

// VerificationPolicyCatalog is a pure domain interface for looking up registered verification profiles.
// Implementations in P02 are pure/in-memory, keeping validation decoupled from concrete P04 runners.
type VerificationPolicyCatalog interface {
	LookupProfile(profileID string) (VerificationProfilePolicy, bool)
}
