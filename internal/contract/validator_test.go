package contract_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/trungqwe/ai-supervisor/internal/contract"
	"github.com/trungqwe/ai-supervisor/internal/domain"
)

type testCatalog struct {
	profiles map[string]domain.VerificationProfilePolicy
}

func (c *testCatalog) LookupProfile(profileID string) (domain.VerificationProfilePolicy, bool) {
	p, ok := c.profiles[profileID]
	return p, ok
}

func loadCanonicalSchema(t *testing.T) []byte {
	t.Helper()
	schemaBytes, err := os.ReadFile("../../docs/schemas/task-contract.schema.json")
	if err != nil {
		t.Fatalf("failed to read canonical task-contract.schema.json: %v", err)
	}
	return schemaBytes
}

func createValidCatalog() *testCatalog {
	return &testCatalog{
		profiles: map[string]domain.VerificationProfilePolicy{
			"go-test": {
				ProfileID: "go-test",
				ParameterSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"package":     map[string]any{"type": "string"},
						"run_pattern": map[string]any{"type": "string"},
						"flags": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
					"required":             []any{"package"},
					"additionalProperties": false,
				},
				CwdPolicy:         "worktree_root",
				MaxTimeoutSeconds: 300,
			},
			"custom-runner": {
				ProfileID: "custom-runner",
				ParameterSchema: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
				},
				CwdPolicy:         "worktree_contained",
				MaxTimeoutSeconds: 600,
			},
		},
	}
}

func TestTaskContractValidator_CanonicalFixtures(t *testing.T) {
	schemaBytes := loadCanonicalSchema(t)
	catalog := createValidCatalog()
	validator, err := contract.NewValidator(schemaBytes, catalog)
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

	validJSON, err := os.ReadFile("../../docs/schemas/examples/task-contract.valid.json")
	if err != nil {
		t.Fatalf("failed to read task-contract.valid.json: %v", err)
	}

	parsed, err := validator.ValidateRaw(validJSON, nil, ".")
	if err != nil {
		t.Fatalf("expected valid fixture to pass, got error: %v", err)
	}
	if parsed.ContractID == "" {
		t.Errorf("expected parsed contract to have contract_id")
	}

	invalidJSON, err := os.ReadFile("../../docs/schemas/examples/task-contract.invalid.json")
	if err != nil {
		t.Fatalf("failed to read task-contract.invalid.json: %v", err)
	}

	_, err = validator.ValidateRaw(invalidJSON, nil, ".")
	if err == nil {
		t.Fatalf("expected invalid fixture to fail, got nil error")
	}
}

func TestTaskContractValidator_CanonicalSchemaEnforcement(t *testing.T) {
	schemaBytes := loadCanonicalSchema(t)
	catalog := createValidCatalog()

	// 1. NewValidator with nil schema FAIL
	t.Run("NewValidator nil schema fails", func(t *testing.T) {
		_, err := contract.NewValidator(nil, catalog)
		if err == nil {
			t.Errorf("expected error for nil schema bytes, got nil")
		}
	})

	// 2. NewValidator with empty schema FAIL
	t.Run("NewValidator empty schema fails", func(t *testing.T) {
		_, err := contract.NewValidator([]byte{}, catalog)
		if err == nil {
			t.Errorf("expected error for empty schema bytes, got nil")
		}
	})

	// 3. ParseAndValidateRaw with nil resolved schema FAIL
	t.Run("ParseAndValidateRaw nil schema fails", func(t *testing.T) {
		raw := []byte(`{"contract_id": "TEST"}`)
		_, err := contract.ParseAndValidateRaw(raw, nil)
		if err == nil {
			t.Errorf("expected error for nil resolved schema in ParseAndValidateRaw, got nil")
		}
	})

	// 4. NewValidator with valid canonical schema PASS
	t.Run("NewValidator valid canonical schema passes", func(t *testing.T) {
		v, err := contract.NewValidator(schemaBytes, catalog)
		if err != nil {
			t.Fatalf("expected NewValidator to pass with canonical schema, got: %v", err)
		}
		if v == nil {
			t.Fatal("expected non-nil validator")
		}
	})
}

func TestTaskContractValidator_StructuralSchemaCases(t *testing.T) {
	schemaBytes := loadCanonicalSchema(t)
	catalog := createValidCatalog()
	validator, err := contract.NewValidator(schemaBytes, catalog)
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

	validJSON, err := os.ReadFile("../../docs/schemas/examples/task-contract.valid.json")
	if err != nil {
		t.Fatalf("failed to read task-contract.valid.json: %v", err)
	}

	// 1. Unknown root property should fail
	t.Run("unknown_root_property_fails", func(t *testing.T) {
		var obj map[string]any
		_ = json.Unmarshal(validJSON, &obj)
		obj["malicious_extra_property"] = "exploit"
		data, _ := json.Marshal(obj)

		_, err := validator.ValidateRaw(data, nil, ".")
		if err == nil {
			t.Errorf("expected error for unknown root property, got nil")
		}
	})

	// 2. Missing required root property should fail
	t.Run("missing_required_root_property_fails", func(t *testing.T) {
		var obj map[string]any
		_ = json.Unmarshal(validJSON, &obj)
		delete(obj, "objective")
		data, _ := json.Marshal(obj)

		_, err := validator.ValidateRaw(data, nil, ".")
		if err == nil {
			t.Errorf("expected error for missing objective, got nil")
		}
	})

	// 3. Required array = null should fail
	t.Run("required_array_null_fails", func(t *testing.T) {
		var obj map[string]any
		_ = json.Unmarshal(validJSON, &obj)
		obj["allowed_scope"] = nil
		data, _ := json.Marshal(obj)

		_, err := validator.ValidateRaw(data, nil, ".")
		if err == nil {
			t.Errorf("expected error for null allowed_scope, got nil")
		}
	})

	// 4. Trailing content after JSON value should fail
	t.Run("trailing_content_fails", func(t *testing.T) {
		badJSON := append(validJSON, []byte(" trailing extra token")...)
		_, err := validator.ValidateRaw(badJSON, nil, ".")
		if err == nil {
			t.Errorf("expected error for trailing content, got nil")
		}
	})
}

func TestTaskContractValidator_RevisionLineage(t *testing.T) {
	prevID := "CONTRACT-TASK-P02-001-01"
	rev1 := &domain.TaskContract{
		ContractID:           prevID,
		TaskID:               "TASK-P02-001",
		RevisionNumber:       1,
		SupersedesContractID: nil,
		PhaseID:              "P02",
		Objective:            "Rev 1 objective",
		Requirements:         []string{"FR-001"},
		ArchitectureRefs:     []string{},
		BaseSHA:              "ca3262eed4b1f72236e86457c865d07cef197094",
		AllowedScope:         []string{"internal/domain/**"},
		ForbiddenScope:       []string{"docs/**"},
		Constraints:          []string{},
		AcceptanceCriteria:   []string{"Pass"},
		VerificationRequests: []domain.VerificationRequest{},
		RequiredEvidence:     []string{"git_diff"},
		WorkerProfile:        "antigravity-standard",
		ReportContract:       "docs/schemas/worker-report.schema.json",
		StopConditions:       []string{},
		IsImmutable:          true,
	}

	// Rev 1 valid
	if err := contract.ValidateRevisionLineage(rev1, nil); err != nil {
		t.Fatalf("rev 1 expected valid, got: %v", err)
	}

	// Rev 1 with supersedes fails
	rev1WithSupersedes := *rev1
	rev1WithSupersedes.SupersedesContractID = &prevID
	if err := contract.ValidateRevisionLineage(&rev1WithSupersedes, nil); err == nil {
		t.Errorf("rev 1 with supersedes expected fail, got nil")
	}

	// Rev 2 valid
	rev2ID := "CONTRACT-TASK-P02-001-02"
	rev2 := *rev1
	rev2.ContractID = rev2ID
	rev2.RevisionNumber = 2
	rev2.SupersedesContractID = &prevID

	if err := contract.ValidateRevisionLineage(&rev2, rev1); err != nil {
		t.Fatalf("rev 2 expected valid, got: %v", err)
	}

	// Rev 2 missing previous fails
	if err := contract.ValidateRevisionLineage(&rev2, nil); err == nil {
		t.Errorf("rev 2 missing previous expected fail, got nil")
	}

	// Revision jump fails (rev 3 directly after rev 1)
	revJump := rev2
	revJump.RevisionNumber = 3
	if err := contract.ValidateRevisionLineage(&revJump, rev1); err == nil {
		t.Errorf("revision jump expected fail, got nil")
	}

	// Task ID mismatch fails
	taskMismatch := rev2
	taskMismatch.TaskID = "TASK-OTHER"
	if err := contract.ValidateRevisionLineage(&taskMismatch, rev1); err == nil {
		t.Errorf("task_id mismatch expected fail, got nil")
	}

	// Supersedes mismatch fails
	otherID := "CONTRACT-OTHER"
	supersedesMismatch := rev2
	supersedesMismatch.SupersedesContractID = &otherID
	if err := contract.ValidateRevisionLineage(&supersedesMismatch, rev1); err == nil {
		t.Errorf("supersedes mismatch expected fail, got nil")
	}

	// BaseSHA mutation fails
	baseMismatch := rev2
	baseMismatch.BaseSHA = "mutated_base_sha_fails"
	if err := contract.ValidateRevisionLineage(&baseMismatch, rev1); err == nil {
		t.Errorf("base_sha mutation across revisions expected fail, got nil")
	}
}

func TestTaskContractValidator_VerificationProfiles(t *testing.T) {
	schemaBytes := loadCanonicalSchema(t)
	catalog := createValidCatalog()
	validator, err := contract.NewValidator(schemaBytes, catalog)
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

	baseContract := &domain.TaskContract{
		ContractID:         "CONTRACT-TEST-01",
		TaskID:             "TASK-TEST",
		RevisionNumber:     1,
		PhaseID:            "P02",
		Objective:          "Profile test",
		Requirements:       []string{"FR-004"},
		ArchitectureRefs:   []string{},
		BaseSHA:            "ca3262eed4b1f72236e86457c865d07cef197094",
		AllowedScope:       []string{"internal/**"},
		ForbiddenScope:     []string{"docs/**"},
		Constraints:        []string{},
		AcceptanceCriteria: []string{"Pass"},
		RequiredEvidence:   []string{"git_diff"},
		WorkerProfile:      "antigravity-standard",
		ReportContract:     "docs/schemas/worker-report.schema.json",
		StopConditions:     []string{},
	}

	// 1. Profile not found fails
	t.Run("profile not found fails", func(t *testing.T) {
		c := *baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "nonexistent-profile", Parameters: map[string]any{}},
		}
		if err := validator.ValidateContract(&c, nil, "."); err == nil {
			t.Errorf("expected error for nonexistent profile, got nil")
		}
	})

	// 2. Profile parameters valid pass
	t.Run("profile parameters valid pass", func(t *testing.T) {
		c := *baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{
				ID:        "req-1",
				ProfileID: "go-test",
				Parameters: map[string]any{
					"package": "./...",
					"flags":   []any{"-count=1"},
				},
				TimeoutSeconds: 60,
				Cwd:            ".",
			},
		}
		if err := validator.ValidateContract(&c, nil, "."); err != nil {
			t.Fatalf("expected valid parameters to pass, got: %v", err)
		}
	})

	// 3. Profile parameters invalid (wrong type, missing required, unknown param)
	t.Run("profile parameters invalid fail", func(t *testing.T) {
		c := *baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{
				ID:        "req-1",
				ProfileID: "go-test",
				Parameters: map[string]any{
					"package":       12345, // Wrong type (should be string)
					"unknown_param": true,  // Forbidden by additionalProperties: false
				},
			},
		}
		if err := validator.ValidateContract(&c, nil, "."); err == nil {
			t.Errorf("expected error for invalid parameters, got nil")
		}
	})

	// 4. Timeout above profile maximum fails
	t.Run("timeout above profile maximum fails", func(t *testing.T) {
		c := *baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{
				ID:             "req-1",
				ProfileID:      "go-test",
				Parameters:     map[string]any{"package": "./..."},
				TimeoutSeconds: 9999, // Exceeds profile max 300
			},
		}
		if err := validator.ValidateContract(&c, nil, "."); err == nil {
			t.Errorf("expected error for timeout exceeding profile maximum, got nil")
		}
	})
}

func TestTaskContractValidator_ProfileConfigurationEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	schemaBytes := loadCanonicalSchema(t)

	baseContract := domain.TaskContract{
		ContractID:         "CONTRACT-EDGE-01",
		TaskID:             "TASK-EDGE",
		RevisionNumber:     1,
		PhaseID:            "P02",
		Objective:          "Edge test",
		Requirements:       []string{"FR-001"},
		ArchitectureRefs:   []string{},
		BaseSHA:            "ca3262eed4b1f72236e86457c865d07cef197094",
		AllowedScope:       []string{"internal/**"},
		ForbiddenScope:     []string{"docs/**"},
		Constraints:        []string{},
		AcceptanceCriteria: []string{"Pass"},
		RequiredEvidence:   []string{"git_diff"},
		WorkerProfile:      "antigravity-standard",
		ReportContract:     "docs/schemas/worker-report.schema.json",
		StopConditions:     []string{},
	}

	// 1. Catalog missing profile FAIL
	t.Run("catalog missing profile fails", func(t *testing.T) {
		cat := &testCatalog{profiles: map[string]domain.VerificationProfilePolicy{}}
		v, err := contract.NewValidator(schemaBytes, cat)
		if err != nil {
			t.Fatalf("NewValidator failed: %v", err)
		}
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "missing-profile", Parameters: map[string]any{}},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for missing profile, got nil")
		}
	})

	// 2. Returned policy ProfileID empty FAIL
	t.Run("returned policy ProfileID empty fails", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"empty-id-profile": {
					ProfileID:         "", // malformed
					MaxTimeoutSeconds: 60,
					ParameterSchema:   map[string]any{"type": "object", "additionalProperties": false},
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "empty-id-profile", Parameters: map[string]any{}},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for empty profile ID, got nil")
		}
	})

	// 3. Returned policy ProfileID mismatch FAIL
	t.Run("returned policy ProfileID mismatch fails", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"requested-id": {
					ProfileID:         "declared-different-id", // mismatch
					MaxTimeoutSeconds: 60,
					ParameterSchema:   map[string]any{"type": "object", "additionalProperties": false},
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "requested-id", Parameters: map[string]any{}},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for profile ID mismatch, got nil")
		}
	})

	// 4. MaxTimeoutSeconds = 0 FAIL
	t.Run("MaxTimeoutSeconds zero fails", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"zero-timeout": {
					ProfileID:         "zero-timeout",
					MaxTimeoutSeconds: 0, // malformed ceiling
					ParameterSchema:   map[string]any{"type": "object", "additionalProperties": false},
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "zero-timeout", Parameters: map[string]any{}},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for MaxTimeoutSeconds = 0, got nil")
		}
	})

	// 5. MaxTimeoutSeconds < 0 FAIL
	t.Run("MaxTimeoutSeconds negative fails", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"negative-timeout": {
					ProfileID:         "negative-timeout",
					MaxTimeoutSeconds: -10, // malformed ceiling
					ParameterSchema:   map[string]any{"type": "object", "additionalProperties": false},
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "negative-timeout", Parameters: map[string]any{}},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for MaxTimeoutSeconds < 0, got nil")
		}
	})

	// 6. ParameterSchema nil FAIL
	t.Run("ParameterSchema nil fails", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"nil-schema": {
					ProfileID:         "nil-schema",
					MaxTimeoutSeconds: 60,
					ParameterSchema:   nil, // malformed
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "nil-schema", Parameters: map[string]any{}},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for nil ParameterSchema, got nil")
		}
	})

	// 7. ParameterSchema empty map FAIL
	t.Run("ParameterSchema empty map fails", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"empty-schema": {
					ProfileID:         "empty-schema",
					MaxTimeoutSeconds: 60,
					ParameterSchema:   map[string]any{}, // malformed
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "empty-schema", Parameters: map[string]any{}},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for empty ParameterSchema, got nil")
		}
	})

	// 8. Direct semantic input with req.Parameters == nil FAIL
	t.Run("req.Parameters nil fails", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"no-param-profile": {
					ProfileID:         "no-param-profile",
					MaxTimeoutSeconds: 60,
					ParameterSchema: map[string]any{
						"type":                 "object",
						"additionalProperties": false,
					},
					CwdPolicy: "worktree_root",
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{ID: "req-1", ProfileID: "no-param-profile", Parameters: nil}, // nil parameters
		}
		if err := v.ValidateContract(&c, nil, tempDir); err == nil {
			t.Errorf("expected error for nil req.Parameters, got nil")
		}
	})

	// 9. Explicit no-parameter object schema + empty parameters PASS
	t.Run("explicit no-parameter schema with empty parameters passes", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"no-param-profile": {
					ProfileID:         "no-param-profile",
					MaxTimeoutSeconds: 60,
					ParameterSchema: map[string]any{
						"type":                 "object",
						"additionalProperties": false,
					},
					CwdPolicy: "worktree_root",
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{
				ID:             "req-1",
				ProfileID:      "no-param-profile",
				Parameters:     map[string]any{},
				Cwd:            ".",
				TimeoutSeconds: 30,
			},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err != nil {
			t.Fatalf("expected valid no-parameter profile to pass, got: %v", err)
		}
	})

	// 10. Valid policy PASS
	t.Run("valid policy passes", func(t *testing.T) {
		cat := &testCatalog{
			profiles: map[string]domain.VerificationProfilePolicy{
				"valid-profile": {
					ProfileID:         "valid-profile",
					MaxTimeoutSeconds: 120,
					ParameterSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"command": map[string]any{"type": "string"},
						},
						"required":             []any{"command"},
						"additionalProperties": false,
					},
					CwdPolicy: "worktree_root",
				},
			},
		}
		v, _ := contract.NewValidator(schemaBytes, cat)
		c := baseContract
		c.VerificationRequests = []domain.VerificationRequest{
			{
				ID:        "req-1",
				ProfileID: "valid-profile",
				Parameters: map[string]any{
					"command": "check",
				},
				Cwd:            ".",
				TimeoutSeconds: 60,
			},
		}
		if err := v.ValidateContract(&c, nil, tempDir); err != nil {
			t.Fatalf("expected valid policy to pass, got: %v", err)
		}
	})
}

func TestTaskContractValidator_PathContainment(t *testing.T) {
	tempDir := t.TempDir()
	containedDir := filepath.Join(tempDir, "sub", "contained")
	if err := os.MkdirAll(containedDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 1. cwd "." PASS
	if err := contract.ValidateCwdContainment(".", tempDir, "worktree_root"); err != nil {
		t.Errorf("expected cwd '.' to pass, got: %v", err)
	}

	// 2. contained cwd PASS under worktree_contained
	if err := contract.ValidateCwdContainment("sub/contained", tempDir, "worktree_contained"); err != nil {
		t.Errorf("expected contained cwd to pass, got: %v", err)
	}

	// 3. contained cwd fails under worktree_root
	if err := contract.ValidateCwdContainment("sub/contained", tempDir, "worktree_root"); err == nil {
		t.Errorf("expected non-root cwd to fail under 'worktree_root', got nil")
	}

	// 4. Traversal escapes fail
	badCwds := []string{
		"../outside",
		"../../outside",
		"sub/../../outside",
		"/absolute/path",
		"\\server\\share",
		`\\?\C:\device`,
		"C:\\absolute\\path",
		"C:relative",
	}

	for _, bad := range badCwds {
		if err := contract.ValidateCwdContainment(bad, tempDir, "worktree_contained"); err == nil {
			t.Errorf("expected escape cwd %q to fail, got nil", bad)
		}
	}

	// 5. Nonexistent contained suffix PASS
	if err := contract.ValidateCwdContainment("sub/contained/future/nonexistent", tempDir, "worktree_contained"); err != nil {
		t.Errorf("expected nonexistent contained suffix to pass, got: %v", err)
	}

	// 6. Nonexistent escaping suffix FAIL
	if err := contract.ValidateCwdContainment("sub/../../../outside_nonexistent", tempDir, "worktree_contained"); err == nil {
		t.Errorf("expected nonexistent escaping suffix to fail, got nil")
	}
}

func TestTaskContractValidator_RegularFileIntermediateEscape(t *testing.T) {
	// Finding R4-001 / Section 10: Privilege-free regression test
	// Create root/safe/blocker.txt where blocker.txt is a regular file.
	// Validate safe/blocker.txt/future under worktree_contained.
	// Expected: FAIL because intermediate component is not a directory.
	tempDir := t.TempDir()
	safeDir := filepath.Join(tempDir, "safe")
	if err := os.MkdirAll(safeDir, 0755); err != nil {
		t.Fatalf("failed to create safe dir: %v", err)
	}
	blockerFile := filepath.Join(safeDir, "blocker.txt")
	if err := os.WriteFile(blockerFile, []byte("I am a file, not a directory"), 0644); err != nil {
		t.Fatalf("failed to create blocker file: %v", err)
	}

	err := contract.ValidateCwdContainment("safe/blocker.txt/future", tempDir, "worktree_contained")
	if err == nil {
		t.Fatalf("expected error for path with regular file intermediate component, got nil")
	}
	t.Logf("REGULAR_FILE_INTERMEDIATE_ESCAPE = REJECTED: %v", err)
}

func TestTaskContractValidator_WhitespaceCanonicality(t *testing.T) {
	// Section 12: Leading/trailing whitespace must be rejected rather than silently trimmed
	tempDir := t.TempDir()

	// 1. cwd with leading/trailing whitespace FAIL
	if err := contract.ValidateCwdContainment(" .", tempDir, "worktree_root"); err == nil {
		t.Errorf("expected error for cwd with leading whitespace, got nil")
	}
	if err := contract.ValidateCwdContainment(". ", tempDir, "worktree_root"); err == nil {
		t.Errorf("expected error for cwd with trailing whitespace, got nil")
	}
	if err := contract.ValidateCwdContainment(" sub/contained ", tempDir, "worktree_contained"); err == nil {
		t.Errorf("expected error for contained cwd with whitespace, got nil")
	}

	// 2. scope patterns with leading/trailing whitespace FAIL
	if err := contract.ValidateScopePatterns([]string{" internal/** "}); err == nil {
		t.Errorf("expected error for scope pattern with whitespace, got nil")
	}
	if err := contract.ValidateScopePatterns([]string{"internal/** "}); err == nil {
		t.Errorf("expected error for scope pattern with trailing whitespace, got nil")
	}
	if err := contract.ValidateScopePatterns([]string{" internal/**"}); err == nil {
		t.Errorf("expected error for scope pattern with leading whitespace, got nil")
	}
}

func TestTaskContractValidator_RootFailureCases(t *testing.T) {
	tempDir := t.TempDir()
	regularFile := filepath.Join(tempDir, "regular_file.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}
	nonexistentPath := filepath.Join(tempDir, "nonexistent_worktree_root")

	// 1. empty worktreeRoot FAIL
	t.Run("empty worktreeRoot fails", func(t *testing.T) {
		if err := contract.ValidateCwdContainment(".", "", "worktree_root"); err == nil {
			t.Errorf("expected error for empty worktreeRoot, got nil")
		}
		if err := contract.ValidateCwdContainment("sub", "", "worktree_contained"); err == nil {
			t.Errorf("expected error for empty worktreeRoot with contained cwd, got nil")
		}
	})

	// 2. nonexistent worktreeRoot FAIL
	t.Run("nonexistent worktreeRoot fails", func(t *testing.T) {
		if err := contract.ValidateCwdContainment(".", nonexistentPath, "worktree_root"); err == nil {
			t.Errorf("expected error for nonexistent worktreeRoot, got nil")
		}
	})

	// 3. worktreeRoot is a regular file FAIL
	t.Run("worktreeRoot is regular file fails", func(t *testing.T) {
		if err := contract.ValidateCwdContainment(".", regularFile, "worktree_root"); err == nil {
			t.Errorf("expected error when worktreeRoot is a regular file, got nil")
		}
	})

	// 4. cwd "." with invalid root FAIL
	t.Run("cwd dot with invalid root fails", func(t *testing.T) {
		if err := contract.ValidateCwdContainment(".", nonexistentPath, "worktree_root"); err == nil {
			t.Errorf("expected error for cwd '.' with nonexistent root, got nil")
		}
		if err := contract.ValidateCwdContainment(".", regularFile, "worktree_root"); err == nil {
			t.Errorf("expected error for cwd '.' with regular file root, got nil")
		}
	})

	// 5. contained cwd with invalid root FAIL
	t.Run("contained cwd with invalid root fails", func(t *testing.T) {
		if err := contract.ValidateCwdContainment("sub/path", nonexistentPath, "worktree_contained"); err == nil {
			t.Errorf("expected error for contained cwd with nonexistent root, got nil")
		}
		if err := contract.ValidateCwdContainment("sub/path", regularFile, "worktree_contained"); err == nil {
			t.Errorf("expected error for contained cwd with regular file root, got nil")
		}
	})
}

func TestTaskContractValidator_ScopePatternValidation(t *testing.T) {
	// 1. Valid patterns
	validScopes := []string{
		"internal/**",
		"internal/domain/*.go",
		"tests/*",
		".",
		"cmd/supervisor",
	}
	if err := contract.ValidateScopePatterns(validScopes); err != nil {
		t.Errorf("expected valid scope patterns to pass, got: %v", err)
	}

	// 2. Invalid patterns
	badScopes := []struct {
		pattern string
		name    string
	}{
		{"", "empty pattern"},
		{"../**", "parent traversal"},
		{"internal/../../escaped", "embedded parent traversal"},
		{"/absolute/**", "leading slash absolute"},
		{`\\server\share\**`, "UNC path"},
		{`C:\**`, "Windows drive path"},
		{"C:relative", "Windows drive relative"},
		{`\\?\C:\**`, "device path"},
	}

	for _, tc := range badScopes {
		if err := contract.ValidateScopePatterns([]string{tc.pattern}); err == nil {
			t.Errorf("expected bad scope %s (%q) to fail, got nil", tc.name, tc.pattern)
		}
	}
}

func TestTaskContractValidator_SymlinkJunctionEscape(t *testing.T) {
	tempDir := t.TempDir()
	outsideDir := t.TempDir()

	// 1. Existing symlink to outside FAIL
	linkPath := filepath.Join(tempDir, "link_to_outside")
	err := os.Symlink(outsideDir, linkPath)
	if err != nil {
		t.Logf("SYMLINK_ESCAPE_RUNTIME_FIXTURE = SKIPPED_ENVIRONMENT_CAPABILITY (symlink creation not supported: %v)", err)
	} else {
		err = contract.ValidateCwdContainment("link_to_outside", tempDir, "worktree_contained")
		if err == nil {
			t.Errorf("expected symlink escaping worktree root to fail, got nil")
		} else {
			t.Logf("Symlink escape successfully rejected: %v", err)
		}

		// 2. Symlink prefix to outside FAIL
		err = contract.ValidateCwdContainment("link_to_outside/child_dir", tempDir, "worktree_contained")
		if err == nil {
			t.Errorf("expected symlink prefix escaping worktree root to fail, got nil")
		} else {
			t.Logf("Symlink prefix escape successfully rejected: %v", err)
		}

		// 3. Section 11: Symlink prefix with future suffix
		err = contract.ValidateCwdContainment("link_to_outside/future_not_created", tempDir, "worktree_contained")
		if err == nil {
			t.Errorf("expected symlink prefix with future suffix to fail, got nil")
		} else {
			t.Logf("SYMLINK_PREFIX_FUTURE_SUFFIX = REJECTED: %v", err)
		}
	}

	// 4. Dangling symlink FAIL
	danglingTarget := filepath.Join(tempDir, "nonexistent_target_dir")
	danglingLink := filepath.Join(tempDir, "dangling_link")
	err = os.Symlink(danglingTarget, danglingLink)
	if err != nil {
		t.Logf("DANGLING_SYMLINK_FIXTURE = SKIPPED_ENVIRONMENT_CAPABILITY (symlink creation not supported: %v)", err)
	} else {
		err = contract.ValidateCwdContainment("dangling_link", tempDir, "worktree_contained")
		if err == nil {
			t.Errorf("expected dangling symlink to fail closed, got nil")
		} else {
			t.Logf("Dangling symlink successfully rejected: %v", err)
		}
	}
}

func TestTaskContractValidator_DirectImmutability(t *testing.T) {
	supersedesID := "CONTRACT-ORIGINAL-00"
	original := &domain.TaskContract{
		ContractID:           "CONTRACT-IMMUTABLE-01",
		TaskID:               "TASK-IMMUTABLE",
		RevisionNumber:       1,
		SupersedesContractID: &supersedesID,
		PhaseID:              "P02",
		Objective:            "Original objective",
		Requirements:         []string{"REQ-1", "REQ-2"},
		ArchitectureRefs:     []string{"ADR-013"},
		BaseSHA:              "87fa16a001a825753bec9e3c5dd36d511e4c5318",
		AllowedScope:         []string{"internal/contract/**"},
		ForbiddenScope:       []string{"docs/**"},
		Constraints:          []string{"pure-go"},
		AcceptanceCriteria:   []string{"100% tests pass"},
		VerificationRequests: []domain.VerificationRequest{
			{
				ID:             "req-1",
				ProfileID:      "go-test",
				Parameters:     map[string]any{"package": "./..."},
				Cwd:            ".",
				TimeoutSeconds: 60,
			},
		},
		RequiredEvidence: []string{"git_diff", "test_logs"},
		WorkerProfile:    "antigravity-standard",
		ReportContract:   "docs/schemas/worker-report.schema.json",
		StopConditions:   []string{"zero_exit_code"},
		IsImmutable:      true,
	}

	// 1. Identical immutable contract = PASS
	t.Run("identical immutable contract passes", func(t *testing.T) {
		candidate := *original
		candidate.Requirements = append([]string{}, original.Requirements...)
		candidate.AllowedScope = append([]string{}, original.AllowedScope...)
		candidate.ForbiddenScope = append([]string{}, original.ForbiddenScope...)
		candidate.VerificationRequests = append([]domain.VerificationRequest{}, original.VerificationRequests...)

		if err := contract.ValidateImmutability(original, &candidate); err != nil {
			t.Fatalf("expected identical immutable contract to pass, got: %v", err)
		}
	})

	// 2. Objective mutation = IMMUTABILITY FAIL
	t.Run("objective mutation fails", func(t *testing.T) {
		candidate := *original
		candidate.Objective = "Mutated objective"
		err := contract.ValidateImmutability(original, &candidate)
		if err == nil {
			t.Fatalf("expected error for objective mutation, got nil")
		}
		var valErr *contract.ValidationError
		if errors.As(err, &valErr) {
			if valErr.Category != "IMMUTABILITY" {
				t.Errorf("expected Category IMMUTABILITY, got %q", valErr.Category)
			}
		} else {
			t.Errorf("expected ValidationError type, got %T", err)
		}
	})

	// 3. SupersedesContractID mutation = IMMUTABILITY FAIL
	t.Run("SupersedesContractID mutation fails", func(t *testing.T) {
		candidateNil := *original
		candidateNil.SupersedesContractID = nil
		if err := contract.ValidateImmutability(original, &candidateNil); err == nil {
			t.Errorf("expected error for SupersedesContractID mutated to nil, got nil")
		}

		diffID := "CONTRACT-MUTATED-ID"
		candidateDiff := *original
		candidateDiff.SupersedesContractID = &diffID
		if err := contract.ValidateImmutability(original, &candidateDiff); err == nil {
			t.Errorf("expected error for SupersedesContractID mutated to different ID, got nil")
		}
	})

	// 4. BaseSHA mutation = IMMUTABILITY FAIL
	t.Run("BaseSHA mutation fails", func(t *testing.T) {
		candidate := *original
		candidate.BaseSHA = "1111222233334444555566667777888899990000"
		err := contract.ValidateImmutability(original, &candidate)
		if err == nil {
			t.Fatalf("expected error for base_sha mutation, got nil")
		}
		var valErr *contract.ValidationError
		if errors.As(err, &valErr) && valErr.Category != "IMMUTABILITY" {
			t.Errorf("expected Category IMMUTABILITY, got %q", valErr.Category)
		}
	})

	// 5. Verification requests mutation = IMMUTABILITY FAIL
	t.Run("VerificationRequests mutation fails", func(t *testing.T) {
		candidate := *original
		candidate.VerificationRequests = []domain.VerificationRequest{}
		err := contract.ValidateImmutability(original, &candidate)
		if err == nil {
			t.Fatalf("expected error for verification_requests mutation, got nil")
		}
		var valErr *contract.ValidationError
		if errors.As(err, &valErr) && valErr.Category != "IMMUTABILITY" {
			t.Errorf("expected Category IMMUTABILITY, got %q", valErr.Category)
		}
	})

	// 6. Scope mutation = IMMUTABILITY FAIL
	t.Run("Scope mutation fails", func(t *testing.T) {
		candidate := *original
		candidate.AllowedScope = []string{"**"}
		err := contract.ValidateImmutability(original, &candidate)
		if err == nil {
			t.Fatalf("expected error for allowed_scope mutation, got nil")
		}
		var valErr *contract.ValidationError
		if errors.As(err, &valErr) && valErr.Category != "IMMUTABILITY" {
			t.Errorf("expected Category IMMUTABILITY, got %q", valErr.Category)
		}
	})

	// 7. New contract_id = not treated as mutation of original (returns nil)
	t.Run("new contract_id not treated as mutation", func(t *testing.T) {
		candidate := *original
		candidate.ContractID = "CONTRACT-NEW-REVISION-02"
		candidate.Objective = "New objective for new revision"
		if err := contract.ValidateImmutability(original, &candidate); err != nil {
			t.Errorf("expected new contract_id to not be treated as mutation of original, got error: %v", err)
		}
	})
}

func TestTaskContractValidator_NumericFidelityAndPrecision(t *testing.T) {
	schemaBytes := loadCanonicalSchema(t)

	catalog := &testCatalog{
		profiles: map[string]domain.VerificationProfilePolicy{
			"numeric-profile": {
				ProfileID: "numeric-profile",
				ParameterSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"large_id":    map[string]any{"type": "integer"},
						"exact_ratio": map[string]any{"type": "number"},
						"exp_int":     map[string]any{"type": "integer"},
					},
					"additionalProperties": false,
				},
				CwdPolicy:         "worktree_root",
				MaxTimeoutSeconds: 60,
			},
		},
	}

	validator, err := contract.NewValidator(schemaBytes, catalog)
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

	// 1. Integer fidelity test: 9007199254740993 preserved as json.Number exact
	t.Run("integer fidelity 9007199254740993 preserved", func(t *testing.T) {
		rawJSON := []byte(`{
  "contract_id": "CONTRACT-NUMERIC-01",
  "task_id": "TASK-NUMERIC",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P02",
  "objective": "Numeric fidelity test",
  "requirements": ["FR-001"],
  "architecture_refs": [],
  "base_sha": "a1b2c3d4e5f6789012345678901234567890abcd",
  "allowed_scope": ["internal/**"],
  "forbidden_scope": [],
  "constraints": [],
  "acceptance_criteria": ["Numeric token preserved"],
  "required_evidence": ["git_diff"],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [],
  "verification_requests": [
    {
      "id": "num-req-1",
      "profile_id": "numeric-profile",
      "parameters": {
        "large_id": 9007199254740993
      },
      "cwd": ".",
      "timeout_seconds": 10
    }
  ]
}`)
		parsed, err := validator.ValidateRaw(rawJSON, nil, ".")
		if err != nil {
			t.Fatalf("expected integer fidelity test to pass, got: %v", err)
		}
		largeID := parsed.VerificationRequests[0].Parameters["large_id"]
		numVal, ok := largeID.(json.Number)
		if !ok {
			t.Fatalf("expected large_id to be json.Number, got %T", largeID)
		}
		if numVal.String() != "9007199254740993" {
			t.Errorf("expected exact 9007199254740993, got %q", numVal.String())
		}
		t.Logf("INTEGER_NUMERIC_FIDELITY = PASS: %s", numVal.String())
	})

	// 2. Exact decimal projection: 0.5 passes
	t.Run("exact decimal 0.5 passes", func(t *testing.T) {
		rawJSON := []byte(`{
  "contract_id": "CONTRACT-NUMERIC-02",
  "task_id": "TASK-NUMERIC",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P02",
  "objective": "Exact decimal test",
  "requirements": ["FR-001"],
  "architecture_refs": [],
  "base_sha": "a1b2c3d4e5f6789012345678901234567890abcd",
  "allowed_scope": ["internal/**"],
  "forbidden_scope": [],
  "constraints": [],
  "acceptance_criteria": ["0.5 passes"],
  "required_evidence": ["git_diff"],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [],
  "verification_requests": [
    {
      "id": "num-req-2",
      "profile_id": "numeric-profile",
      "parameters": {
        "exact_ratio": 0.5
      },
      "cwd": ".",
      "timeout_seconds": 10
    }
  ]
}`)
		parsed, err := validator.ValidateRaw(rawJSON, nil, ".")
		if err != nil {
			t.Fatalf("expected exact decimal 0.5 to pass, got: %v", err)
		}
		ratio := parsed.VerificationRequests[0].Parameters["exact_ratio"]
		numVal, ok := ratio.(json.Number)
		if !ok {
			t.Fatalf("expected exact_ratio to be json.Number, got %T", ratio)
		}
		if numVal.String() != "0.5" {
			t.Errorf("expected exact 0.5, got %q", numVal.String())
		}
		t.Logf("EXACT_DECIMAL_PROJECTION = PASS: %s", numVal.String())
	})

	// 3. Exponent integer: 1e3 under integer schema passes and authoritative is json.Number
	t.Run("mathematical exponent integer 1e3 passes integer schema", func(t *testing.T) {
		rawJSON := []byte(`{
  "contract_id": "CONTRACT-NUMERIC-03",
  "task_id": "TASK-NUMERIC",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P02",
  "objective": "Exponent integer test",
  "requirements": ["FR-001"],
  "architecture_refs": [],
  "base_sha": "a1b2c3d4e5f6789012345678901234567890abcd",
  "allowed_scope": ["internal/**"],
  "forbidden_scope": [],
  "constraints": [],
  "acceptance_criteria": ["1e3 passes"],
  "required_evidence": ["git_diff"],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [],
  "verification_requests": [
    {
      "id": "num-req-3",
      "profile_id": "numeric-profile",
      "parameters": {
        "exp_int": 1e3
      },
      "cwd": ".",
      "timeout_seconds": 10
    }
  ]
}`)
		parsed, err := validator.ValidateRaw(rawJSON, nil, ".")
		if err != nil {
			t.Fatalf("expected exponent integer 1e3 to pass, got: %v", err)
		}
		expInt := parsed.VerificationRequests[0].Parameters["exp_int"]
		numVal, ok := expInt.(json.Number)
		if !ok {
			t.Fatalf("expected exp_int to be json.Number, got %T", expInt)
		}
		if numVal.String() != "1e3" {
			t.Errorf("expected exact 1e3, got %q", numVal.String())
		}
		t.Logf("EXPONENT_INTEGER_PROJECTION = PASS: %s", numVal.String())
	})

	// 4. Inexact decimal: 0.1000000000000000000000001 fails closed with precision error
	t.Run("inexact decimal fails closed", func(t *testing.T) {
		rawJSON := []byte(`{
  "contract_id": "CONTRACT-NUMERIC-04",
  "task_id": "TASK-NUMERIC",
  "revision_number": 1,
  "supersedes_contract_id": null,
  "phase_id": "P02",
  "objective": "Inexact decimal test",
  "requirements": ["FR-001"],
  "architecture_refs": [],
  "base_sha": "a1b2c3d4e5f6789012345678901234567890abcd",
  "allowed_scope": ["internal/**"],
  "forbidden_scope": [],
  "constraints": [],
  "acceptance_criteria": ["Inexact decimal fails"],
  "required_evidence": ["git_diff"],
  "worker_profile": "antigravity-standard",
  "report_contract": "docs/schemas/worker-report.schema.json",
  "stop_conditions": [],
  "verification_requests": [
    {
      "id": "num-req-4",
      "profile_id": "numeric-profile",
      "parameters": {
        "exact_ratio": 0.1000000000000000000000001
      },
      "cwd": ".",
      "timeout_seconds": 10
    }
  ]
}`)
		_, err := validator.ValidateRaw(rawJSON, nil, ".")
		if err == nil {
			t.Fatalf("expected inexact decimal to fail with numeric projection precision error, got nil")
		}
		t.Logf("INEXACT_DECIMAL_PROJECTION = REJECTED: %v", err)
	})
}
