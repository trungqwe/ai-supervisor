package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	t.Logf("SQLITE_WAL = PASS")

	if ep.Synchronous != 2 {
		t.Errorf("expected synchronous 2, got %d", ep.Synchronous)
	}
	t.Logf("SQLITE_SYNCHRONOUS_FULL = PASS")

	if !ep.ForeignKeys {
		t.Errorf("expected foreign_keys true, got %v", ep.ForeignKeys)
	}
	t.Logf("SQLITE_FOREIGN_KEYS = PASS")

	if ep.BusyTimeout != 5000 {
		t.Errorf("expected busy_timeout 5000, got %d", ep.BusyTimeout)
	}
	if ep.UserVersion != 3 {
		t.Errorf("expected user_version 3, got %d", ep.UserVersion)
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
	if ep.UserVersion != 3 {
		t.Errorf("expected user_version 3, got %d", ep.UserVersion)
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

func TestStore_TaskCreationInitialInvariant(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p-init", Name: "P-Init", RootPath: "/init"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-init", ProjectID: "p-init", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	// Finding R2-002: Initial task state must be DRAFT and current_attempt must be 0
	// 1. Initial state = READY must fail
	taskReady := domain.Task{
		TaskID:         "task-init-ready",
		PhaseID:        "P02",
		PairID:         "pair-init",
		State:          domain.StateReady,
		CurrentAttempt: 0,
	}
	err := s.CreateTask(ctx, taskReady)
	if err == nil {
		t.Fatalf("expected non-DRAFT initial state to fail, got nil")
	}
	if !errors.Is(err, ErrInvalidInitialTaskState) {
		t.Errorf("expected ErrInvalidInitialTaskState, got %v", err)
	}

	// 2. Initial state = DISPATCHED must fail
	taskDisp := domain.Task{
		TaskID:         "task-init-disp",
		PhaseID:        "P02",
		PairID:         "pair-init",
		State:          domain.StateDispatched,
		CurrentAttempt: 0,
	}
	err = s.CreateTask(ctx, taskDisp)
	if err == nil || !errors.Is(err, ErrInvalidInitialTaskState) {
		t.Errorf("expected ErrInvalidInitialTaskState for DISPATCHED initial state, got %v", err)
	}

	// 3. Initial current_attempt > 0 must fail
	taskAttempt := domain.Task{
		TaskID:         "task-init-att",
		PhaseID:        "P02",
		PairID:         "pair-init",
		State:          domain.StateDraft,
		CurrentAttempt: 1,
	}
	err = s.CreateTask(ctx, taskAttempt)
	if err == nil || !errors.Is(err, ErrInvalidInitialTaskState) {
		t.Errorf("expected ErrInvalidInitialTaskState for non-zero initial attempt, got %v", err)
	}

	// 4. Valid DRAFT + attempt 0 succeeds
	taskDraft := domain.Task{
		TaskID:         "task-init-draft",
		PhaseID:        "P02",
		PairID:         "pair-init",
		State:          domain.StateDraft,
		CurrentAttempt: 0,
	}
	if err := s.CreateTask(ctx, taskDraft); err != nil {
		t.Fatalf("valid DRAFT task creation failed: %v", err)
	}
}

func TestStore_TaskTransition_GenericReadyToDispatchedBlocked(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p-tr", Name: "P-TR", RootPath: "/tr"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-tr", ProjectID: "p-tr", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	// Create in DRAFT
	task := domain.Task{TaskID: "task-tr-1", PhaseID: "P02", PairID: "pair-tr", State: domain.StateDraft, CurrentAttempt: 0}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Insert governing contract before transitioning DRAFT -> READY
	contract := domain.TaskContract{
		ContractID:     "contract-tr-1",
		TaskID:         "task-tr-1",
		RevisionNumber: 1,
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
	}
	if err := s.InsertTaskContract(ctx, contract); err != nil {
		t.Fatalf("InsertTaskContract failed: %v", err)
	}

	if err := s.TransitionTask(ctx, "task-tr-1", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask DRAFT -> READY failed: %v", err)
	}

	// Finding R2-001: Generic TransitionTask READY -> DISPATCHED MUST be blocked!
	err := s.TransitionTask(ctx, "task-tr-1", domain.StateReady, domain.StateDispatched)
	if err == nil {
		t.Fatalf("expected generic TransitionTask READY -> DISPATCHED to fail, got nil")
	}
	if !errors.Is(err, ErrAtomicDispatchRequired) {
		t.Errorf("expected ErrAtomicDispatchRequired, got %v", err)
	}
	t.Logf("GENERIC_READY_TO_DISPATCHED = REJECTED (%v)", err)

	// Verify task remains in READY state with current_attempt = 0
	tsk, err := s.GetTask(ctx, "task-tr-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateReady || tsk.CurrentAttempt != 0 {
		t.Errorf("task state was corrupted after rejected transition: %+v", tsk)
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
	task := domain.Task{TaskID: "task-c1", PhaseID: "P02", PairID: "pair-1", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, task)

	// Contract with large 53-bit+ integer in parameters inserted while in DRAFT (Finding R3-002)
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

	// Duplicate revision for same task must fail UNIQUE(task_id, revision_number) in DRAFT
	contractDup := contract
	contractDup.ContractID = "contract-002"
	err := s.InsertTaskContract(ctx, contractDup)
	if err == nil {
		t.Fatalf("expected duplicate revision to fail, got nil")
	}
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}

	// Transition DRAFT -> READY after contract insertion and duplicate test
	if err := s.TransitionTask(ctx, "task-c1", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask DRAFT -> READY failed: %v", err)
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
	val, ok := got.VerificationRequests[0].Parameters["max_count"]
	if !ok {
		t.Fatalf("expected max_count parameter")
	}
	num, ok := val.(json.Number)
	if !ok {
		t.Fatalf("expected json.Number, got %T (%v)", val, val)
	}
	if num.String() != "9007199254740993" {
		t.Errorf("loss of precision: got %s, want 9007199254740993", num.String())
	}
	t.Logf("JSON_NUMBER_STORE_ROUNDTRIP = PASS: %s", num.String())
}

func TestStore_ContractLineagePersistence(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p-lin", Name: "Lineage Proj", RootPath: "/lin"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-lin", ProjectID: "p-lin", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)
	task1 := domain.Task{TaskID: "task-lin-1", PhaseID: "P02", PairID: "pair-lin", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, task1)

	task2 := domain.Task{TaskID: "task-lin-2", PhaseID: "P02", PairID: "pair-lin", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, task2)

	baseSHA := "87fa16a001a825753bec9e3c5dd36d511e4c5318"

	// 1. Inserting an already-immutable contract must fail
	cImm := domain.TaskContract{
		ContractID:     "c-imm-fail",
		TaskID:         "task-lin-1",
		RevisionNumber: 1,
		BaseSHA:        baseSHA,
		IsImmutable:    true,
	}
	err := s.InsertTaskContract(ctx, cImm)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for immutable new contract, got %v", err)
	}

	// 2. Rev 1 specifying supersedes must fail
	supID := "other-id"
	cRev1Sup := domain.TaskContract{
		ContractID:           "c-rev1-sup",
		TaskID:               "task-lin-1",
		RevisionNumber:       1,
		SupersedesContractID: &supID,
		BaseSHA:              baseSHA,
	}
	err = s.InsertTaskContract(ctx, cRev1Sup)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for rev 1 with supersedes, got %v", err)
	}

	// 3. Valid Rev 1 insert succeeds in DRAFT
	cRev1 := domain.TaskContract{
		ContractID:     "c-lin-1",
		TaskID:         "task-lin-1",
		RevisionNumber: 1,
		BaseSHA:        baseSHA,
	}
	if err := s.InsertTaskContract(ctx, cRev1); err != nil {
		t.Fatalf("valid Rev 1 insert failed: %v", err)
	}

	// 4. Rev 2 without supersedes must fail
	cRev2NoSup := domain.TaskContract{
		ContractID:     "c-rev2-nosup",
		TaskID:         "task-lin-1",
		RevisionNumber: 2,
		BaseSHA:        baseSHA,
	}
	err = s.InsertTaskContract(ctx, cRev2NoSup)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for rev 2 without supersedes, got %v", err)
	}

	// 5. Rev 2 superseding non-existent contract must fail
	nonExist := "c-nonexist"
	cRev2Missing := domain.TaskContract{
		ContractID:           "c-rev2-missing",
		TaskID:               "task-lin-1",
		RevisionNumber:       2,
		SupersedesContractID: &nonExist,
		BaseSHA:              baseSHA,
	}
	err = s.InsertTaskContract(ctx, cRev2Missing)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for missing superseded contract, got %v", err)
	}

	// 6. Rev 2 following a non-immutable Rev 1 must fail
	rev1ID := "c-lin-1"
	cRev2Unfrozen := domain.TaskContract{
		ContractID:           "c-rev2-unfrozen",
		TaskID:               "task-lin-1",
		RevisionNumber:       2,
		SupersedesContractID: &rev1ID,
		BaseSHA:              baseSHA,
	}
	err = s.InsertTaskContract(ctx, cRev2Unfrozen)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for unfrozen previous contract, got %v", err)
	}

	// Transition task-lin-1 DRAFT -> READY
	if err := s.TransitionTask(ctx, "task-lin-1", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask DRAFT -> READY failed: %v", err)
	}

	// Freeze rev 1 via PrepareDispatch
	repPath, _ := CanonicalExpectedReportPath("task-lin-1", "att-lin-1")
	_, err = s.PrepareDispatch(ctx, "task-lin-1", "c-lin-1", "att-lin-1", repPath, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch on c-lin-1 failed: %v", err)
	}

	// Progress task-lin-1 canonically to REVISION_REQUIRED to permit rev 2 insertion
	s.TransitionTask(ctx, "task-lin-1", domain.StateDispatched, domain.StateRunning)
	s.TransitionTask(ctx, "task-lin-1", domain.StateRunning, domain.StateReportReady)
	s.TransitionTask(ctx, "task-lin-1", domain.StateReportReady, domain.StateEvidenceReady)
	s.TransitionTask(ctx, "task-lin-1", domain.StateEvidenceReady, domain.StateReviewing)
	s.TransitionTask(ctx, "task-lin-1", domain.StateReviewing, domain.StateRevisionRequired)

	// 7. Rev 2 superseding a contract of a DIFFERENT task must fail
	cRev2OtherTask := domain.TaskContract{
		ContractID:           "c-rev2-other",
		TaskID:               "task-lin-2", // Task 2 trying to supersede Task 1's contract!
		RevisionNumber:       2,
		SupersedesContractID: &rev1ID,
		BaseSHA:              baseSHA,
	}
	err = s.InsertTaskContract(ctx, cRev2OtherTask)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for cross-task supersedes, got %v", err)
	}
	t.Logf("CROSS_TASK_SUPERSEDES = REJECTED (%v)", err)

	// 8. Revision jump (Rev 3 instead of Rev 2) must fail
	cRevJump := domain.TaskContract{
		ContractID:           "c-rev3-jump",
		TaskID:               "task-lin-1",
		RevisionNumber:       3, // expected 2
		SupersedesContractID: &rev1ID,
		BaseSHA:              baseSHA,
	}
	err = s.InsertTaskContract(ctx, cRevJump)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for revision jump, got %v", err)
	}
	t.Logf("REVISION_JUMP = REJECTED (%v)", err)

	// 9. BaseSHA mutation across revisions must fail
	diffSHA := "ffffffffffffffffffffffffffffffffffffffff"
	cRevBaseMut := domain.TaskContract{
		ContractID:           "c-rev2-basemut",
		TaskID:               "task-lin-1",
		RevisionNumber:       2,
		SupersedesContractID: &rev1ID,
		BaseSHA:              diffSHA,
	}
	err = s.InsertTaskContract(ctx, cRevBaseMut)
	if err == nil || !errors.Is(err, ErrInvalidContractLineage) {
		t.Errorf("expected ErrInvalidContractLineage for base_sha change, got %v", err)
	}
	t.Logf("REVISION_BASE_SHA_CHANGE = REJECTED (%v)", err)

	// 10. Valid Rev 2 succeeds
	cRev2Valid := domain.TaskContract{
		ContractID:           "c-lin-2",
		TaskID:               "task-lin-1",
		RevisionNumber:       2,
		SupersedesContractID: &rev1ID,
		BaseSHA:              baseSHA,
	}
	if err := s.InsertTaskContract(ctx, cRev2Valid); err != nil {
		t.Fatalf("valid Rev 2 insert failed: %v", err)
	}
	t.Logf("VALID_REVISION_2_LINEAGE = PASS")
	t.Logf("CONTRACT_LINEAGE_PERSISTENCE = PASS")
}

func TestStore_ContractImmutabilityTriggers(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p1", Name: "P1", RootPath: "/p1"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-1", ProjectID: "p1", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)
	task := domain.Task{TaskID: "task-t1", PhaseID: "P02", PairID: "pair-1", State: domain.StateDraft, CurrentAttempt: 0}
	s.CreateTask(ctx, task)

	c := domain.TaskContract{
		ContractID:     "c-imm",
		TaskID:         "task-t1",
		RevisionNumber: 1,
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
	}
	if err := s.InsertTaskContract(ctx, c); err != nil {
		t.Fatalf("InsertTaskContract failed: %v", err)
	}

	if err := s.TransitionTask(ctx, "task-t1", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask failed: %v", err)
	}

	// Freeze via PrepareDispatch
	repPath, _ := CanonicalExpectedReportPath("task-t1", "att-t1")
	_, err := s.PrepareDispatch(ctx, "task-t1", "c-imm", "att-t1", repPath, time.Now())
	if err != nil {
		t.Fatalf("PrepareDispatch failed: %v", err)
	}

	// Try to update immutable contract directly in SQLite -> trigger aborts
	_, err = s.db.Exec("UPDATE task_contracts SET base_sha = 'mutated' WHERE contract_id = 'c-imm'")
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

func TestStore_ReadyEntryContractPreconditions(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p-ready-test", Name: "Ready Precondition Proj", RootPath: "/rdy"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-ready-test", ProjectID: "p-ready-test", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	// 1. Create task in DRAFT
	task := domain.Task{
		TaskID:         "task-rdy-1",
		PhaseID:        "P02",
		PairID:         "pair-ready-test",
		State:          domain.StateDraft,
		CurrentAttempt: 0,
	}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 2. Transition DRAFT -> READY without contract must FAIL ErrReadyContractRequired
	err := s.TransitionTask(ctx, "task-rdy-1", domain.StateDraft, domain.StateReady)
	if err == nil || !errors.Is(err, ErrReadyContractRequired) {
		t.Fatalf("expected ErrReadyContractRequired for task without contract, got: %v", err)
	}
	t.Logf("DRAFT_READY_WITHOUT_CONTRACT = REJECTED (%v)", err)

	// Verify task remains in DRAFT
	tsk, err := s.GetTask(ctx, "task-rdy-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateDraft {
		t.Fatalf("task state corrupted: got %s, want DRAFT", tsk.State)
	}

	// 3. Insert rev 1 while in DRAFT
	c1 := domain.TaskContract{
		ContractID:     "c-rdy-1",
		TaskID:         "task-rdy-1",
		RevisionNumber: 1,
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
	}
	if err := s.InsertTaskContract(ctx, c1); err != nil {
		t.Fatalf("InsertTaskContract rev 1 failed: %v", err)
	}

	// 4. Transition DRAFT -> READY with contract must succeed
	if err := s.TransitionTask(ctx, "task-rdy-1", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("TransitionTask DRAFT -> READY failed after contract inserted: %v", err)
	}
	t.Logf("DRAFT_READY_WITH_CONTRACT = PASS")
	t.Logf("INITIAL_DRAFT_MUTABLE_REV1_READY = PASS")

	// 5. Attempt to insert another new contract while READY must FAIL ErrContractInsertState
	c2 := domain.TaskContract{
		ContractID:     "c-rdy-2-illegal",
		TaskID:         "task-rdy-1",
		RevisionNumber: 1,
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
	}
	err = s.InsertTaskContract(ctx, c2)
	if err == nil || !errors.Is(err, ErrContractInsertState) {
		t.Fatalf("expected ErrContractInsertState when inserting contract in READY state, got: %v", err)
	}
	t.Logf("CONTRACT_INSERT_WHILE_READY = REJECTED (%v)", err)
}

func TestStore_HumanReplanningFlow(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	proj := domain.Project{ProjectID: "p-replan", Name: "Replan Proj", RootPath: "/replan"}
	s.CreateProject(ctx, proj)
	pair := domain.Pair{PairID: "pair-replan", ProjectID: "p-replan", CurrentPhaseID: "P02", State: "ACTIVE"}
	s.CreatePair(ctx, pair)

	task := domain.Task{
		TaskID:         "task-replan-1",
		PhaseID:        "P02",
		PairID:         "pair-replan",
		State:          domain.StateDraft,
		CurrentAttempt: 0,
	}
	s.CreateTask(ctx, task)

	c1 := domain.TaskContract{
		ContractID:     "c-replan-1",
		TaskID:         "task-replan-1",
		RevisionNumber: 1,
		BaseSHA:        "87fa16a001a825753bec9e3c5dd36d511e4c5318",
	}
	s.InsertTaskContract(ctx, c1)
	s.TransitionTask(ctx, "task-replan-1", domain.StateDraft, domain.StateReady)

	// Dispatch rev 1 (freezes c1 to is_immutable = true)
	p1, _ := CanonicalExpectedReportPath("task-replan-1", "att-replan-1")
	_, err := s.PrepareDispatch(ctx, "task-replan-1", "c-replan-1", "att-replan-1", p1, time.Now())
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}

	// Canonical failure escalation: DISPATCHED -> FAILED -> HUMAN_REQUIRED
	s.TransitionTask(ctx, "task-replan-1", domain.StateDispatched, domain.StateFailed)
	s.TransitionTask(ctx, "task-replan-1", domain.StateFailed, domain.StateHumanRequired)

	// HUMAN_REQUIRED -> DRAFT (Human replanning initiated)
	if err := s.TransitionTask(ctx, "task-replan-1", domain.StateHumanRequired, domain.StateDraft); err != nil {
		t.Fatalf("transition HUMAN_REQUIRED -> DRAFT failed: %v", err)
	}

	// Finding R4-001: Attempting DRAFT -> READY without creating a new revision MUST FAIL
	// Reusing the old frozen contract is strictly rejected
	err = s.TransitionTask(ctx, "task-replan-1", domain.StateDraft, domain.StateReady)
	if err == nil || !errors.Is(err, ErrReadyContractRequired) {
		t.Fatalf("expected ErrReadyContractRequired when attempting DRAFT -> READY with old immutable contract, got: %v", err)
	}
	t.Logf("HUMAN_REPLAN_WITH_OLD_IMMUTABLE_CONTRACT = REJECTED (%v)", err)

	// Verify task remains in DRAFT
	tsk, err := s.GetTask(ctx, "task-replan-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if tsk.State != domain.StateDraft {
		t.Fatalf("task state corrupted: got %s, want DRAFT", tsk.State)
	}

	// Insert valid mutable Rev 2 while in DRAFT
	c1ID := "c-replan-1"
	c2 := domain.TaskContract{
		ContractID:           "c-replan-2",
		TaskID:               "task-replan-1",
		RevisionNumber:       2,
		SupersedesContractID: &c1ID,
		BaseSHA:              "87fa16a001a825753bec9e3c5dd36d511e4c5318",
	}
	if err := s.InsertTaskContract(ctx, c2); err != nil {
		t.Fatalf("insert rev 2 in DRAFT replanning state failed: %v", err)
	}

	// Now DRAFT -> READY succeeds because latest contract is mutable rev 2
	if err := s.TransitionTask(ctx, "task-replan-1", domain.StateDraft, domain.StateReady); err != nil {
		t.Fatalf("transition DRAFT -> READY failed after replanning: %v", err)
	}
	t.Logf("HUMAN_REPLAN_WITH_NEW_MUTABLE_REVISION = PASS")
	t.Logf("HUMAN_REPLAN_DRAFT_READY = PASS")

	// Dispatch rev 2 succeeds
	p2, _ := CanonicalExpectedReportPath("task-replan-1", "att-replan-2")
	att2, err := s.PrepareDispatch(ctx, "task-replan-1", "c-replan-2", "att-replan-2", p2, time.Now())
	if err != nil {
		t.Fatalf("dispatch rev 2 failed: %v", err)
	}
	if att2.ContractID != "c-replan-2" || att2.AttemptNumber != 2 {
		t.Errorf("unexpected attempt 2: %+v", att2)
	}
}

func TestStore_CanonicalExpectedReportPath_WindowsSafety(t *testing.T) {
	// 1. Valid safe identifiers
	validCases := []struct {
		taskID    string
		attemptID string
		want      string
	}{
		{"TASK-P02-003", "attempt-1", ".supervisor/reports/TASK-P02-003/attempt-1.json"},
		{"TASK-123", "ATT_123", ".supervisor/reports/TASK-123/ATT_123.json"},
		{"TASK-UUID", "550e8400-e29b-41d4-a716-446655440000", ".supervisor/reports/TASK-UUID/550e8400-e29b-41d4-a716-446655440000.json"},
	}
	for _, tc := range validCases {
		got, err := CanonicalExpectedReportPath(tc.taskID, tc.attemptID)
		if err != nil {
			t.Fatalf("unexpected error for valid (%s, %s): %v", tc.taskID, tc.attemptID, err)
		}
		if got != tc.want {
			t.Errorf("got %s, want %s", got, tc.want)
		}
	}
	t.Logf("CANONICAL_SAFE_ATTEMPT_ID = PASS")
	t.Logf("CANONICAL_REPORT_PATH = PASS")

	// 2. Windows reserved device base names (case-insensitive and with extensions)
	reservedDevices := []string{
		"CON", "con", "NUL", "nul", "PRN", "prn", "AUX", "aux",
		"COM1", "com1", "COM9", "com9", "LPT1", "lpt1", "LPT9", "lpt9",
		"CON.foo", "NUL.log", "aux.txt", "COM3.dat", "lpt4.out",
	}
	for _, dev := range reservedDevices {
		_, err := CanonicalExpectedReportPath("TASK-1", dev)
		if err == nil || !errors.Is(err, ErrReportPathMismatch) {
			t.Fatalf("expected ErrReportPathMismatch for reserved device attemptID %q, got: %v", dev, err)
		}
		_, err = CanonicalExpectedReportPath(dev, "att-1")
		if err == nil || !errors.Is(err, ErrReportPathMismatch) {
			t.Fatalf("expected ErrReportPathMismatch for reserved device taskID %q, got: %v", dev, err)
		}
	}
	t.Logf("WINDOWS_RESERVED_DEVICE_ATTEMPT_ID = REJECTED")
	t.Logf("WINDOWS_REPORT_SEGMENT_RESERVED_DEVICE = REJECTED")

	// 3. Windows forbidden characters: < > : " / \ | ? *
	forbiddenChars := []string{
		"bad*id", "bad?id", "bad|id", "bad\"id", "bad<id", "bad>id",
		"bad/id", "bad\\id", "bad:id",
	}
	for _, bad := range forbiddenChars {
		_, err := CanonicalExpectedReportPath("TASK-1", bad)
		if err == nil || !errors.Is(err, ErrReportPathMismatch) {
			t.Fatalf("expected ErrReportPathMismatch for forbidden char attemptID %q, got: %v", bad, err)
		}
	}
	t.Logf("WINDOWS_FORBIDDEN_CHARACTER_ATTEMPT_ID = REJECTED")
	t.Logf("WINDOWS_REPORT_SEGMENT_FORBIDDEN_CHARACTER = REJECTED")

	// 4. Trailing period and space normalization hazards
	trailingCases := []string{
		"name.", "name.txt.", " name", "name ", "name. ", "name .",
	}
	for _, tc := range trailingCases {
		_, err := CanonicalExpectedReportPath("TASK-1", tc)
		if err == nil || !errors.Is(err, ErrReportPathMismatch) {
			t.Fatalf("expected ErrReportPathMismatch for trailing dot/space attemptID %q, got: %v", tc, err)
		}
	}
	t.Logf("WINDOWS_TRAILING_PERIOD_ATTEMPT_ID = REJECTED")
	t.Logf("WINDOWS_REPORT_SEGMENT_TRAILING_PERIOD = REJECTED")

	// 5. Control characters: 0x01 through 0x1F, 0x00, 0x7F
	controlCases := []string{
		"att\x01id", "att\x1Fid", "att\x00id", "att\x07id", "att\x7Fid",
	}
	for _, cc := range controlCases {
		_, err := CanonicalExpectedReportPath("TASK-1", cc)
		if err == nil || !errors.Is(err, ErrReportPathMismatch) {
			t.Fatalf("expected ErrReportPathMismatch for control char attemptID %q, got: %v", cc, err)
		}
	}
	t.Logf("WINDOWS_CONTROL_CHARACTER_ATTEMPT_ID = REJECTED")
	t.Logf("WINDOWS_REPORT_SEGMENT_CONTROL_CHARACTER = REJECTED")
}
