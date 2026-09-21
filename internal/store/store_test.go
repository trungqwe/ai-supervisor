package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func createTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "supervisor.db")
	cfg := Config{
		DBPath:        dbPath,
		BusyTimeoutMs: 5000,
	}
	s, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	return s, dbPath
}

func TestStore_OpenAndPragmas(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	ep, err := s.EffectivePragmas(ctx)
	if err != nil {
		t.Fatalf("EffectivePragmas failed: %v", err)
	}

	t.Logf("Effective Pragmas: journal_mode=%s, synchronous=%d, foreign_keys=%v, busy_timeout=%d, user_version=%d",
		ep.JournalMode, ep.Synchronous, ep.ForeignKeys, ep.BusyTimeout, ep.UserVersion)

	if ep.JournalMode != "wal" {
		t.Errorf("expected journal_mode wal, got %s", ep.JournalMode)
	}
	if ep.Synchronous != 2 {
		t.Errorf("expected synchronous 2, got %d", ep.Synchronous)
	}
	if !ep.ForeignKeys {
		t.Errorf("expected foreign_keys true, got %v", ep.ForeignKeys)
	}
	if ep.BusyTimeout != 5000 {
		t.Errorf("expected busy_timeout 5000, got %d", ep.BusyTimeout)
	}
	if ep.UserVersion != 1 {
		t.Errorf("expected user_version 1, got %d", ep.UserVersion)
	}
}

func TestStore_MigrationIdempotentAndFuture(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "mig_test.db")

	cfg := Config{DBPath: dbPath, BusyTimeoutMs: 5000}
	s1, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("First Open failed: %v", err)
	}
	s1.Close()

	// Reopen should be idempotent
	s2, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	ep, err := s2.EffectivePragmas(ctx)
	if err != nil {
		t.Fatalf("EffectivePragmas failed: %v", err)
	}
	if ep.UserVersion != 1 {
		t.Errorf("expected user_version 1, got %d", ep.UserVersion)
	}
	s2.Close()

	// Simulate future user_version
	rawDB, err := sql.Open("sqlite", cfg.DSN())
	if err != nil {
		t.Fatalf("raw Open failed: %v", err)
	}
	if _, err := rawDB.Exec("PRAGMA user_version = 99"); err != nil {
		t.Fatalf("failed to set future user_version: %v", err)
	}
	rawDB.Close()

	// Open should fail closed with ErrUnsupportedSchemaVersion
	_, err = Open(ctx, cfg)
	if err == nil {
		t.Fatalf("expected error on future user_version, got nil")
	}
	if !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Errorf("expected ErrUnsupportedSchemaVersion, got: %v", err)
	}
}

func TestStore_ProjectPersistence(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)
	p := domain.Project{
		ProjectID:    "proj-001",
		Name:         "AI Supervisor Core",
		RootPath:     "D:/TU_CODE/ai-supervisor",
		RepoURL:      "https://github.com/trungqwe/ai-supervisor",
		RegisteredAt: now,
	}

	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Retrieve
	got, err := s.GetProject(ctx, "proj-001")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if got.ProjectID != p.ProjectID || got.Name != p.Name || got.RootPath != p.RootPath || got.RepoURL != p.RepoURL {
		t.Errorf("project mismatch: got %+v, want %+v", got, p)
	}
	if !got.RegisteredAt.Equal(now) {
		t.Errorf("registered_at mismatch: got %v, want %v", got.RegisteredAt, now)
	}

	// Duplicate project ID fails
	err = s.CreateProject(ctx, p)
	if err == nil {
		t.Fatalf("expected duplicate project creation to fail, got nil")
	}
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}

	// Not found
	_, err = s.GetProject(ctx, "non-existent")
	if !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestStore_PairPersistence(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Pair without project must fail foreign key constraint
	pair := domain.Pair{
		PairID:         "pair-001",
		ProjectID:      "proj-missing",
		CurrentPhaseID: "P02",
		State:          "ACTIVE",
		CreatedAt:      time.Now().UTC(),
	}
	err := s.CreatePair(ctx, pair)
	if err == nil {
		t.Fatalf("expected FK violation for missing project, got nil")
	}
	if !errors.Is(err, ErrForeignKeyViolation) {
		t.Errorf("expected ErrForeignKeyViolation, got %v", err)
	}

	// Create project first
	proj := domain.Project{
		ProjectID: "proj-001",
		Name:      "Project 1",
		RootPath:  "/path/1",
	}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Now create pair
	pair.ProjectID = "proj-001"
	if err := s.CreatePair(ctx, pair); err != nil {
		t.Fatalf("CreatePair failed: %v", err)
	}

	got, err := s.GetPair(ctx, "pair-001")
	if err != nil {
		t.Fatalf("GetPair failed: %v", err)
	}
	if got.PairID != pair.PairID || got.ProjectID != pair.ProjectID {
		t.Errorf("pair mismatch: got %+v, want %+v", got, pair)
	}

	// 1:1 Project-Pair Invariant: Second pair for same project must fail
	pair2 := domain.Pair{
		PairID:         "pair-002",
		ProjectID:      "proj-001",
		CurrentPhaseID: "P02",
		State:          "ACTIVE",
	}
	err = s.CreatePair(ctx, pair2)
	if err == nil {
		t.Fatalf("expected 1:1 project-pair violation, got nil")
	}
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}
}

func TestStore_TaskPersistenceAndCanonicalTransitions(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p1", Name: "P1", RootPath: "/p1"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	pair := domain.Pair{PairID: "pair-1", ProjectID: "p1", CurrentPhaseID: "P02", State: "ACTIVE"}
	if err := s.CreatePair(ctx, pair); err != nil {
		t.Fatalf("CreatePair failed: %v", err)
	}

	// Test all 13 canonical states are admitted
	allStates := domain.AllTaskStates()
	if len(allStates) != 13 {
		t.Fatalf("expected 13 states, got %d", len(allStates))
	}
	for i, st := range allStates {
		tsk := domain.Task{
			TaskID:         fmt.Sprintf("task-%d", i),
			PhaseID:        "P02",
			PairID:         "pair-1",
			State:          st,
			CurrentAttempt: 0,
		}
		if err := s.CreateTask(ctx, tsk); err != nil {
			t.Fatalf("failed to insert task with canonical state %s: %v", st, err)
		}
	}

	// Invalid 14th state must fail CHECK constraint or domain validation
	badTask := domain.Task{
		TaskID:  "task-bad",
		PhaseID: "P02",
		PairID:  "pair-1",
		State:   domain.TaskState("UNKNOWN_STATE"),
	}
	if err := s.CreateTask(ctx, badTask); err == nil {
		t.Fatalf("expected invalid task state to fail, got nil")
	}

	// Test TransitionTask
	t1, err := s.GetTask(ctx, "task-0") // task-0 is DRAFT
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if t1.State != domain.StateDraft {
		t.Fatalf("expected DRAFT, got %s", t1.State)
	}

	// Canonical transition: DRAFT -> READY
	if err := s.TransitionTask(ctx, "task-0", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask DRAFT -> READY failed: %v", err)
	}

	t1After, err := s.GetTask(ctx, "task-0")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if t1After.State != domain.StateReady {
		t.Fatalf("expected READY, got %s", t1After.State)
	}

	// Non-canonical transition: READY -> APPROVED must fail
	err = s.TransitionTask(ctx, "task-0", domain.StateReady, domain.StateApproved)
	if err == nil {
		t.Fatalf("expected non-canonical transition READY -> APPROVED to fail, got nil")
	}

	// State conflict: expected DRAFT but task is now READY
	err = s.TransitionTask(ctx, "task-0", domain.StateDraft, domain.StateCancelled)
	if err == nil {
		t.Fatalf("expected state conflict, got nil")
	}
	if !errors.Is(err, ErrStateConflict) {
		t.Errorf("expected ErrStateConflict, got %v", err)
	}
}

func TestStore_TaskContractPersistenceAndNumericFidelity(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p1", Name: "P1", RootPath: "/p1"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-1", ProjectID: "p1", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)
	task := domain.Task{TaskID: "task-c1", PhaseID: "P02", PairID: "pair-1", State: domain.StateReady}
	s.CreateTask(ctx, task)

	// Contract with large 53-bit+ integer in parameters
	largeInt := json.Number("9007199254740993")
	contract := domain.TaskContract{
		ContractID:     "contract-001",
		TaskID:         "task-c1",
		RevisionNumber: 1,
		PhaseID:        "P02",
		Objective:      "Test contract storage",
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
		AllowedScope:   []string{"internal/store/**"},
		VerificationRequests: []domain.VerificationRequest{
			{
				ID:        "req-1",
				ProfileID: "go-test",
				Parameters: map[string]any{
					"max_count": largeInt,
				},
				Cwd:            ".",
				TimeoutSeconds: 60,
			},
		},
		WorkerProfile:  "antigravity-standard",
		ReportContract: "docs/schemas/worker-report.schema.json",
	}

	if err := s.InsertTaskContract(ctx, contract); err != nil {
		t.Fatalf("InsertTaskContract failed: %v", err)
	}

	// Retrieve contract and verify exact json.Number roundtrip
	got, err := s.GetTaskContract(ctx, "contract-001")
	if err != nil {
		t.Fatalf("GetTaskContract failed: %v", err)
	}

	if got.ContractID != contract.ContractID {
		t.Errorf("contract_id mismatch: got %s, want %s", got.ContractID, contract.ContractID)
	}
	if len(got.VerificationRequests) != 1 {
		t.Fatalf("expected 1 verification request, got %d", len(got.VerificationRequests))
	}
	rawParam, ok := got.VerificationRequests[0].Parameters["max_count"]
	if !ok {
		t.Fatalf("missing max_count parameter")
	}

	num, ok := rawParam.(json.Number)
	if !ok {
		t.Fatalf("parameter max_count is not json.Number (got %T: %v)", rawParam, rawParam)
	}
	if num.String() != "9007199254740993" {
		t.Fatalf("large integer corrupted: got %s, want 9007199254740993", num.String())
	}
	t.Logf("JSON_NUMBER_STORE_ROUNDTRIP = PASS: %s", num.String())

	// Duplicate revision for same task must fail UNIQUE(task_id, revision_number)
	contractDup := contract
	contractDup.ContractID = "contract-002" // different ID, same task & revision
	err = s.InsertTaskContract(ctx, contractDup)
	if err == nil {
		t.Fatalf("expected duplicate revision to fail, got nil")
	}
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}
}

func TestStore_ContractImmutabilityTriggers(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p1", Name: "P1", RootPath: "/p1"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-1", ProjectID: "p1", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)
	task := domain.Task{TaskID: "task-t1", PhaseID: "P02", PairID: "pair-1", State: domain.StateReady}
	s.CreateTask(ctx, task)

	c := domain.TaskContract{
		ContractID:     "c-imm",
		TaskID:         "task-t1",
		RevisionNumber: 1,
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
		IsImmutable:    true, // already immutable
	}
	if err := s.InsertTaskContract(ctx, c); err != nil {
		t.Fatalf("InsertTaskContract failed: %v", err)
	}

	// Try to update immutable contract directly in SQLite -> trigger aborts
	_, err := s.db.Exec("UPDATE task_contracts SET base_sha = 'mutated' WHERE contract_id = 'c-imm'")
	if err == nil {
		t.Fatalf("expected UPDATE on immutable contract to fail, got nil")
	}
	t.Logf("IMMUTABLE_UPDATE_BLOCKED = PASS: %v", err)

	// Try to delete immutable contract directly in SQLite -> trigger aborts
	_, err = s.db.Exec("DELETE FROM task_contracts WHERE contract_id = 'c-imm'")
	if err == nil {
		t.Fatalf("expected DELETE on immutable contract to fail, got nil")
	}
	t.Logf("IMMUTABLE_DELETE_BLOCKED = PASS: %v", err)
}

func TestStore_ConfigValidation(t *testing.T) {
	// Empty DBPath fails
	c1 := Config{DBPath: ""}
	if err := c1.Validate(); err == nil {
		t.Errorf("expected error for empty DBPath, got nil")
	}

	// Non-existent parent directory fails
	c2 := Config{DBPath: filepath.Join(t.TempDir(), "nonexistent", "db.sqlite")}
	if err := c2.Validate(); err == nil {
		t.Errorf("expected error for non-existent parent directory, got nil")
	}

	// Busy timeout < 1 fails
	c3 := Config{DBPath: filepath.Join(t.TempDir(), "test.db"), BusyTimeoutMs: -5}
	if err := c3.Validate(); err == nil {
		t.Errorf("expected error for negative busy timeout, got nil")
	}

	// Busy timeout > 60000 fails
	c4 := Config{DBPath: filepath.Join(t.TempDir(), "test.db"), BusyTimeoutMs: 70000}
	if err := c4.Validate(); err == nil {
		t.Errorf("expected error for busy timeout > 60000, got nil")
	}

	// Zero busy timeout defaults to 5000
	c5 := Config{DBPath: filepath.Join(t.TempDir(), "test.db"), BusyTimeoutMs: 0}
	if err := c5.Validate(); err != nil {
		t.Errorf("unexpected error for zero busy timeout: %v", err)
	}
	if c5.BusyTimeoutMs != DefaultBusyTimeoutMs {
		t.Errorf("expected default busy timeout %d, got %d", DefaultBusyTimeoutMs, c5.BusyTimeoutMs)
	}
}

func TestStore_CloseReopenAndFileLockRelease(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "lock_test.db")
	cfg := Config{DBPath: dbPath, BusyTimeoutMs: 5000}

	s, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	proj := domain.Project{ProjectID: "p-lock", Name: "Lock Proj", RootPath: "/lock"}
	if err := s.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Close store
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// On Windows, if file handles are still held, renaming will fail.
	renamedPath := filepath.Join(tempDir, "renamed_test.db")
	if err := os.Rename(dbPath, renamedPath); err != nil {
		t.Fatalf("file lock held: failed to rename db after Close: %v", err)
	}
	t.Logf("FILE_LOCK_RELEASE_AFTER_CLOSE = PASS: file successfully renamed")

	// Open the renamed database file to prove clean reopen
	cfgRenamed := Config{DBPath: renamedPath, BusyTimeoutMs: 5000}
	sRenamed, err := Open(ctx, cfgRenamed)
	if err != nil {
		t.Fatalf("failed to reopen renamed db: %v", err)
	}
	defer sRenamed.Close()

	got, err := sRenamed.GetProject(ctx, "p-lock")
	if err != nil {
		t.Fatalf("failed to read from reopened db: %v", err)
	}
	if got.ProjectID != "p-lock" {
		t.Errorf("data mismatch after reopen: got %+v", got)
	}
	t.Logf("CLEAN_REOPEN_AFTER_CLOSE = PASS")
}
