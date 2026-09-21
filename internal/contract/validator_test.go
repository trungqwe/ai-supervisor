package contract_test

import (
	"encoding/json"
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
				ProfileID:         "custom-runner",
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

func TestTaskContractValidator_StructuralSchemaCases(t *testing.T) {
	schemaBytes := loadCanonicalSchema(t)
	catalog := createValidCatalog()
	validator, err := contract.NewValidator(schemaBytes, catalog)
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

	validJSON, err := os.ReadFile("../../docs/schemas/examples/task-contract.valid.json")
	if err != nil {
		t.Fatalf("failed to read valid fixture: %v", err)
	}

	// 1. Unknown root property should fail (additionalProperties = false)
	t.Run("unknown root property fails", func(t *testing.T) {
		var obj map[string]any
		_ = json.Unmarshal(validJSON, &obj)
		obj["unknown_field"] = "malicious_payload"
		data, _ := json.Marshal(obj)

		_, err := validator.ValidateRaw(data, nil, ".")
		if err == nil {
			t.Errorf("expected error for unknown root property, got nil")
		}
	})

	// 2. Missing required root property should fail
	t.Run("missing required root property fails", func(t *testing.T) {
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
	t.Run("required array null fails", func(t *testing.T) {
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
	t.Run("trailing content fails", func(t *testing.T) {
		badJSON := append(validJSON, []byte(" trailing extra token")...)
		_, err := validator.ValidateRaw(badJSON, nil, ".")
		if err == nil {
			t.Errorf("expected error for trailing content, got nil")
		}
	})
}

func TestTaskContractValidator_RevisionLineage(t *testing.T) {
	schemaBytes := loadCanonicalSchema(t)
	catalog := createValidCatalog()
	validator, err := contract.NewValidator(schemaBytes, catalog)
	if err != nil {
		t.Fatalf("NewValidator failed: %v", err)
	}

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
	if err := validator.ValidateContract(rev1, nil, "."); err != nil {
		t.Fatalf("rev 1 expected valid, got: %v", err)
	}

	// Rev 1 with supersedes fails
	rev1WithSupersedes := *rev1
	rev1WithSupersedes.SupersedesContractID = &prevID
	if err := validator.ValidateContract(&rev1WithSupersedes, nil, "."); err == nil {
		t.Errorf("rev 1 with supersedes expected fail, got nil")
	}

	// Rev 2 valid
	rev2ID := "CONTRACT-TASK-P02-001-02"
	rev2 := *rev1
	rev2.ContractID = rev2ID
	rev2.RevisionNumber = 2
	rev2.SupersedesContractID = &prevID

	if err := validator.ValidateContract(&rev2, rev1, "."); err != nil {
		t.Fatalf("rev 2 expected valid, got: %v", err)
	}

	// Rev 2 missing previous fails
	if err := validator.ValidateContract(&rev2, nil, "."); err == nil {
		t.Errorf("rev 2 missing previous expected fail, got nil")
	}

	// Revision jump fails (rev 3 directly after rev 1)
	revJump := rev2
	revJump.RevisionNumber = 3
	if err := validator.ValidateContract(&revJump, rev1, "."); err == nil {
		t.Errorf("revision jump expected fail, got nil")
	}

	// Task ID mismatch fails
	revTaskMismatch := rev2
	revTaskMismatch.TaskID = "TASK-DIFFERENT"
	if err := validator.ValidateContract(&revTaskMismatch, rev1, "."); err == nil {
		t.Errorf("task_id mismatch expected fail, got nil")
	}

	// Supersedes mismatch fails
	wrongPrev := "CONTRACT-OTHER"
	revSupersedesMismatch := rev2
	revSupersedesMismatch.SupersedesContractID = &wrongPrev
	if err := validator.ValidateContract(&revSupersedesMismatch, rev1, "."); err == nil {
		t.Errorf("supersedes mismatch expected fail, got nil")
	}

	// base_sha change fails
	revBaseSHAMismatch := rev2
	revBaseSHAMismatch.BaseSHA = "different_sha_12345"
	if err := validator.ValidateContract(&revBaseSHAMismatch, rev1, "."); err == nil {
		t.Errorf("base_sha change expected fail, got nil")
	}

	// Same immutable contract_id mutation fails
	revMutated := *rev1
	revMutated.Objective = "Mutated Objective on same contract_id"
	if err := validator.ValidateContract(&revMutated, rev1, "."); err == nil {
		t.Errorf("same immutable contract_id mutation expected fail, got nil")
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

	linkPath := filepath.Join(tempDir, "link_to_outside")
	err := os.Symlink(outsideDir, linkPath)
	if err != nil {
		t.Logf("SYMLINK_ESCAPE_RUNTIME_FIXTURE = SKIPPED_ENVIRONMENT_CAPABILITY (symlink creation not supported: %v)", err)
		return
	}

	// When symlink points outside worktree root, validation must reject it
	err = contract.ValidateCwdContainment("link_to_outside", tempDir, "worktree_contained")
	if err == nil {
		t.Errorf("expected symlink escaping worktree root to fail, got nil")
	} else {
		t.Logf("Symlink escape successfully rejected: %v", err)
	}
}
