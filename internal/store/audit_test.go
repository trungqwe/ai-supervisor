package store

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestStore_V1ToV2Migration(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migration_v1_v2.db")
	cfg := Config{DBPath: dbPath, BusyTimeoutMs: 5000}

	// 1. Create a V1 database explicitly using v1Schema and PRAGMA user_version = 1
	rawDB, err := sql.Open("sqlite", cfg.DSN())
	if err != nil {
		t.Fatalf("failed to open raw sqlite: %v", err)
	}
	if _, err := rawDB.Exec(v1Schema); err != nil {
		t.Fatalf("failed to execute v1Schema: %v", err)
	}
	if _, err := rawDB.Exec("PRAGMA user_version = 1"); err != nil {
		t.Fatalf("failed to set user_version 1: %v", err)
	}

	var uv int
	if err := rawDB.QueryRow("PRAGMA user_version").Scan(&uv); err != nil || uv != 1 {
		t.Fatalf("expected user_version 1, got %d (err: %v)", uv, err)
	}
	rawDB.Close()

	// 2. Open via store.Open -> should upgrade 1 -> 2
	s, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to open store on existing V1 DB: %v", err)
	}
	ep, err := s.EffectivePragmas(ctx)
	if err != nil {
		t.Fatalf("EffectivePragmas failed: %v", err)
	}
	if ep.UserVersion != 2 {
		t.Fatalf("expected user_version 2 after migration, got %d", ep.UserVersion)
	}

	// Verify audit tables and singleton exist
	var headSeq int64
	var headHash string
	if err := s.db.QueryRowContext(ctx, "SELECT last_sequence, last_hash FROM audit_chain_state WHERE id = 1").Scan(&headSeq, &headHash); err != nil {
		t.Fatalf("failed to read audit_chain_state: %v", err)
	}
	if headSeq != 0 || headHash != GenesisAuditHash {
		t.Fatalf("expected initial chain state (0, %s), got (%d, %s)", GenesisAuditHash, headSeq, headHash)
	}
	s.Close()

	// 3. Reopen V2 DB -> idempotent
	sReopen, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to reopen V2 DB: %v", err)
	}
	epReopen, _ := sReopen.EffectivePragmas(ctx)
	if epReopen.UserVersion != 2 {
		t.Fatalf("expected user_version 2 on reopen, got %d", epReopen.UserVersion)
	}
	sReopen.Close()

	// 4. Test deliberate V2 migration failure rollback on another DB
	failDBPath := filepath.Join(tempDir, "migration_fail.db")
	failCfg := Config{DBPath: failDBPath, BusyTimeoutMs: 5000}
	rawFailDB, err := sql.Open("sqlite", failCfg.DSN())
	if err != nil {
		t.Fatalf("failed to open raw fail DB: %v", err)
	}
	if _, err := rawFailDB.Exec(v1Schema); err != nil {
		t.Fatalf("failed to execute v1Schema: %v", err)
	}
	if _, err := rawFailDB.Exec("PRAGMA user_version = 1"); err != nil {
		t.Fatalf("failed to set user_version 1: %v", err)
	}

	// Execute migration with broken V2 DDL
	err = migrateWithSchemas(ctx, rawFailDB, v1Schema, "INVALID SYNTAX STATEMENT;")
	if err == nil {
		t.Fatalf("expected error on broken V2 DDL, got nil")
	}

	// Verify user_version remains 1 and audit_events does not exist
	var failUV int
	if err := rawFailDB.QueryRow("PRAGMA user_version").Scan(&failUV); err != nil || failUV != 1 {
		t.Fatalf("expected user_version to remain 1 after failed migration, got %d", failUV)
	}
	var auditTableExists int
	_ = rawFailDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='audit_events'").Scan(&auditTableExists)
	if auditTableExists != 0 {
		t.Fatalf("expected audit_events table to not exist after rollback, but it was found")
	}
	rawFailDB.Close()

	t.Logf("SQLITE_SCHEMA_VERSION = 2")
	t.Logf("V1_TO_V2_MIGRATION = PASS")
}

func TestStore_AuditAppendOnly_Triggers(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// 1. Append valid event
	evt1 := domain.AuditEvent{
		EventID:   "evt-app-01",
		EventType: "system.started",
		Actor:     "supervisor-core",
		Details: map[string]any{
			"version": "1.0.0",
		},
	}
	rec1, err := s.AppendAuditEvent(ctx, evt1)
	if err != nil {
		t.Fatalf("AppendAuditEvent failed: %v", err)
	}
	if rec1.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", rec1.Sequence)
	}
	if rec1.PrevHash != GenesisAuditHash {
		t.Errorf("expected prevHash %s, got %s", GenesisAuditHash, rec1.PrevHash)
	}

	// 2. Append second event
	evt2 := domain.AuditEvent{
		EventID:   "evt-app-02",
		EventType: "task.created",
		Actor:     "test-worker",
		Details: map[string]any{
			"notes": "initial creation",
		},
	}
	rec2, err := s.AppendAuditEvent(ctx, evt2)
	if err != nil {
		t.Fatalf("Append second event failed: %v", err)
	}
	if rec2.Sequence != 2 {
		t.Errorf("expected sequence 2, got %d", rec2.Sequence)
	}
	if rec2.PrevHash != rec1.EventHash {
		t.Errorf("expected prevHash %s, got %s", rec1.EventHash, rec2.PrevHash)
	}

	// 3. Test GetAuditEvent
	gotRec, err := s.GetAuditEvent(ctx, "evt-app-01")
	if err != nil {
		t.Fatalf("GetAuditEvent failed: %v", err)
	}
	if gotRec.Event.EventID != "evt-app-01" || gotRec.Sequence != 1 {
		t.Errorf("unexpected GetAuditEvent result: %+v", gotRec)
	}

	// 4. Test ListAuditEvents
	list, err := s.ListAuditEvents(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ListAuditEvents failed: %v", err)
	}
	if len(list) != 2 || list[0].Sequence != 1 || list[1].Sequence != 2 {
		t.Fatalf("unexpected ListAuditEvents result: %+v", list)
	}

	// 5. Test raw SQL UPDATE trigger prevention
	_, err = s.db.ExecContext(ctx, "UPDATE audit_events SET actor = 'tampered' WHERE sequence = 1")
	if err == nil {
		t.Fatalf("expected UPDATE to fail via trigger, got nil")
	}
	if !strings.Contains(err.Error(), "cannot update audit events") {
		t.Errorf("unexpected UPDATE error: %v", err)
	}

	// 6. Test raw SQL DELETE trigger prevention
	_, err = s.db.ExecContext(ctx, "DELETE FROM audit_events WHERE sequence = 1")
	if err == nil {
		t.Fatalf("expected DELETE to fail via trigger, got nil")
	}
	if !strings.Contains(err.Error(), "cannot delete audit events") {
		t.Errorf("unexpected DELETE error: %v", err)
	}

	// 7. Test INSERT OR REPLACE prevention trigger
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO audit_events (
			sequence, event_id, event_type, timestamp,
			actor, details_json, prev_hash, event_hash
		) VALUES (1, 'evt-replaced', 'hacked', '2026-01-01T00:00:00Z', 'bad-actor', '{}', '000', '111')
	`)
	if err == nil {
		t.Fatalf("expected INSERT OR REPLACE by sequence to fail via trigger, got nil")
	}
	if !strings.Contains(err.Error(), "cannot replace existing audit event") {
		t.Errorf("unexpected replacement error: %v", err)
	}

	// Test INSERT OR REPLACE by event_id collision
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO audit_events (
			sequence, event_id, event_type, timestamp,
			actor, details_json, prev_hash, event_hash
		) VALUES (99, 'evt-app-01', 'hacked', '2026-01-01T00:00:00Z', 'bad-actor', '{}', '000', '111')
	`)
	if err == nil {
		t.Fatalf("expected INSERT OR REPLACE by event_id to fail via trigger, got nil")
	}
	if !strings.Contains(err.Error(), "cannot replace existing audit event") {
		t.Errorf("unexpected replacement error: %v", err)
	}

	// 8. Verify chain remains completely valid after all attempted mutations
	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("VerifyAuditChain failed after rejected mutations: %v", err)
	}

	t.Logf("AUDIT_APPEND_ONLY = PASS")
}

func TestStore_DuplicateEventID_Rollback(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Append first event
	_, err := s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:   "evt-dup-target",
		EventType: "test.event",
		Actor:     "actor-1",
	})
	if err != nil {
		t.Fatalf("first append failed: %v", err)
	}

	// Attempt duplicate append
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:   "evt-dup-target",
		EventType: "test.event",
		Actor:     "actor-2",
	})
	if err == nil {
		t.Fatalf("expected duplicate EventID to fail, got nil")
	}
	if !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("expected ErrDuplicateKey, got: %v", err)
	}

	// Verify chain state was NOT advanced
	var lastSeq int64
	var count int
	_ = s.db.QueryRowContext(ctx, "SELECT last_sequence FROM audit_chain_state WHERE id = 1").Scan(&lastSeq)
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events").Scan(&count)

	if lastSeq != 1 {
		t.Errorf("last_sequence advanced after duplicate rejection: %d", lastSeq)
	}
	if count != 1 {
		t.Errorf("audit_events count changed after duplicate rejection: %d", count)
	}

	// Verify chain is valid
	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("VerifyAuditChain failed: %v", err)
	}

	t.Logf("DUPLICATE_EVENT_ROLLBACK = PASS")
}

func TestStore_AuditHashChain_TamperEvidence(t *testing.T) {
	ctx := context.Background()
	var err error

	// 1. Empty chain
	sEmpty, _ := createTestStore(t)
	if err := sEmpty.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("empty chain verification failed: %v", err)
	}
	sEmpty.Close()

	// 2. Multi-event valid chain
	s, _ := createTestStore(t)
	defer s.Close()

	for i := 1; i <= 5; i++ {
		_, err := s.AppendAuditEvent(ctx, domain.AuditEvent{
			EventID:   fmt.Sprintf("evt-chain-%02d", i),
			EventType: "batch.event",
			Actor:     "batch-worker",
			Details: map[string]any{
				"idx": i,
			},
		})
		if err != nil {
			t.Fatalf("append %d failed: %v", i, err)
		}
	}

	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("multi-event chain verification failed: %v", err)
	}
	t.Logf("AUDIT_HASH_CHAIN = PASS")

	// 3. Tamper Scenario A: Interior event mutation
	// Drop update trigger temporarily in test to simulate SQLite corruption
	if _, err := s.db.ExecContext(ctx, "DROP TRIGGER trg_audit_events_prevent_update"); err != nil {
		t.Fatalf("failed to drop trigger: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE audit_events SET actor = 'malicious-tamper' WHERE sequence = 3"); err != nil {
		t.Fatalf("failed to update event 3: %v", err)
	}
	err = s.VerifyAuditChain(ctx)
	if err == nil || !errors.Is(err, ErrAuditChainInvalid) {
		t.Fatalf("expected ErrAuditChainInvalid on interior mutation, got: %v", err)
	}

	// Restore actor to restore validity
	if _, err := s.db.ExecContext(ctx, "UPDATE audit_events SET actor = 'batch-worker' WHERE sequence = 3"); err != nil {
		t.Fatalf("failed to restore actor: %v", err)
	}
	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("chain should be valid after restoring actor, got: %v", err)
	}

	// 4. Tamper Scenario B: Interior event deletion
	if _, err := s.db.ExecContext(ctx, "DROP TRIGGER trg_audit_events_prevent_delete"); err != nil {
		t.Fatalf("failed to drop delete trigger: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM audit_events WHERE sequence = 3"); err != nil {
		t.Fatalf("failed to delete event 3: %v", err)
	}
	err = s.VerifyAuditChain(ctx)
	if err == nil || !errors.Is(err, ErrAuditChainInvalid) {
		t.Fatalf("expected ErrAuditChainInvalid on interior deletion, got: %v", err)
	}

	// 5. Tamper Scenario C: Tail deletion with unchanged chain state
	// Re-create a clean store with 3 events
	sTail, _ := createTestStore(t)
	defer sTail.Close()
	for i := 1; i <= 3; i++ {
		_, _ = sTail.AppendAuditEvent(ctx, domain.AuditEvent{
			EventID:   fmt.Sprintf("evt-tail-%d", i),
			EventType: "tail.event",
			Actor:     "tail-worker",
		})
	}
	if _, err := sTail.db.ExecContext(ctx, "DROP TRIGGER trg_audit_events_prevent_delete"); err != nil {
		t.Fatalf("failed to drop delete trigger: %v", err)
	}
	// Delete sequence 3 (tail) without changing audit_chain_state
	if _, err := sTail.db.ExecContext(ctx, "DELETE FROM audit_events WHERE sequence = 3"); err != nil {
		t.Fatalf("failed to delete tail event: %v", err)
	}
	err = sTail.VerifyAuditChain(ctx)
	if err == nil || !errors.Is(err, ErrAuditChainInvalid) {
		t.Fatalf("expected ErrAuditChainInvalid on tail deletion, got: %v", err)
	}

	// 6. Tamper Scenario D: Audit chain-head mismatch
	sHead, _ := createTestStore(t)
	defer sHead.Close()
	_, _ = sHead.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:   "evt-head-1",
		EventType: "head.event",
		Actor:     "head-worker",
	})
	// Tamper last_hash in audit_chain_state
	if _, err := sHead.db.ExecContext(ctx, "UPDATE audit_chain_state SET last_hash = 'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff' WHERE id = 1"); err != nil {
		t.Fatalf("failed to tamper chain head: %v", err)
	}
	err = sHead.VerifyAuditChain(ctx)
	if err == nil || !errors.Is(err, ErrAuditChainInvalid) {
		t.Fatalf("expected ErrAuditChainInvalid on head mismatch, got: %v", err)
	}

	t.Logf("AUDIT_CHAIN_TAMPER_DETECTION = PASS")
}

func TestStore_ConcurrentAuditAppend(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	const numAppenders = 24
	var wg sync.WaitGroup
	errCh := make(chan error, numAppenders)

	startGate := make(chan struct{})

	for i := 0; i < numAppenders; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startGate

			evt := domain.AuditEvent{
				EventID:   fmt.Sprintf("evt-concurrent-%03d", idx),
				EventType: "concurrent.test",
				Actor:     fmt.Sprintf("worker-%03d", idx),
				Details: map[string]any{
					"worker_index": idx,
				},
			}
			_, err := s.AppendAuditEvent(ctx, evt)
			if err != nil {
				errCh <- fmt.Errorf("appender %d failed: %w", idx, err)
			}
		}(i)
	}

	close(startGate)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent append error: %v", err)
	}

	// Verify contiguous sequence exactly 1..numAppenders
	list, err := s.ListAuditEvents(ctx, 0, 500)
	if err != nil {
		t.Fatalf("ListAuditEvents failed: %v", err)
	}
	if len(list) != numAppenders {
		t.Fatalf("expected %d events, got %d", numAppenders, len(list))
	}

	for i, rec := range list {
		expectedSeq := int64(i + 1)
		if rec.Sequence != expectedSeq {
			t.Errorf("at index %d: expected sequence %d, got %d", i, expectedSeq, rec.Sequence)
		}
	}

	// Verify chain integrity
	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("VerifyAuditChain failed: %v", err)
	}

	var headSeq int64
	_ = s.db.QueryRowContext(ctx, "SELECT last_sequence FROM audit_chain_state WHERE id = 1").Scan(&headSeq)
	if headSeq != numAppenders {
		t.Errorf("expected headSeq %d, got %d", numAppenders, headSeq)
	}

	t.Logf("CONCURRENT_AUDIT_APPEND = PASS")
}

func TestStore_AuditLineageValidation(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// Seed domain objects
	now := time.Now().UTC()
	_ = s.CreateProject(ctx, domain.Project{ProjectID: "proj-1", Name: "P1", RootPath: "/r", RegisteredAt: now})
	_ = s.CreatePair(ctx, domain.Pair{PairID: "pair-1", ProjectID: "proj-1", CurrentPhaseID: "P02", State: "ACTIVE", CreatedAt: now})
	_ = s.CreateTask(ctx, domain.Task{TaskID: "task-1", PhaseID: "P02", PairID: "pair-1", State: domain.StateDraft, CreatedAt: now, UpdatedAt: now})

	contract := domain.TaskContract{
		ContractID:     "contract-1",
		TaskID:         "task-1",
		RevisionNumber: 1,
		BaseSHA:        "0123456789012345678901234567890123456789",
	}
	_ = s.InsertTaskContract(ctx, contract)

	reportPath, _ := CanonicalExpectedReportPath("task-1", "att-1")
	_ = s.TransitionTask(ctx, "task-1", domain.StateDraft, domain.StateReady)
	_, _ = s.PrepareDispatch(ctx, "task-1", "contract-1", "att-1", reportPath, now)

	// Another task and contract
	_ = s.CreateProject(ctx, domain.Project{ProjectID: "proj-2", Name: "P2", RootPath: "/r2", RegisteredAt: now})
	_ = s.CreatePair(ctx, domain.Pair{PairID: "pair-2", ProjectID: "proj-2", CurrentPhaseID: "P02", State: "ACTIVE", CreatedAt: now})
	_ = s.CreateTask(ctx, domain.Task{TaskID: "task-2", PhaseID: "P02", PairID: "pair-2", State: domain.StateDraft, CreatedAt: now, UpdatedAt: now})
	_ = s.InsertTaskContract(ctx, domain.TaskContract{
		ContractID:     "contract-2",
		TaskID:         "task-2",
		RevisionNumber: 1,
		BaseSHA:        "0123456789012345678901234567890123456789",
	})

	// Case 1: Global event with no lineage -> PASS
	_, err := s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:   "evt-lin-01",
		EventType: "worker.started",
		Actor:     "daemon",
	})
	if err != nil {
		t.Fatalf("global event failed: %v", err)
	}

	// Case 2: Task-only event referencing existing task -> PASS
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:   "evt-lin-02",
		EventType: "task.noted",
		TaskID:    "task-1",
		Actor:     "worker",
	})
	if err != nil {
		t.Fatalf("task-only event failed: %v", err)
	}

	// Case 3: Matching Task + Contract -> PASS
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:    "evt-lin-03",
		EventType:  "contract.signed",
		TaskID:     "task-1",
		ContractID: "contract-1",
		Actor:      "worker",
	})
	if err != nil {
		t.Fatalf("matching task+contract failed: %v", err)
	}

	// Case 4: Contract belonging to different task -> FAIL
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:    "evt-lin-04",
		EventType:  "contract.mismatch",
		TaskID:     "task-1",
		ContractID: "contract-2", // belongs to task-2!
		Actor:      "worker",
	})
	if err == nil || !errors.Is(err, ErrInvalidAuditLineage) {
		t.Fatalf("expected ErrInvalidAuditLineage on contract mismatch, got: %v", err)
	}

	// Case 5: Attempt with matching task+contract -> PASS
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:    "evt-lin-05",
		EventType:  "attempt.started",
		TaskID:     "task-1",
		ContractID: "contract-1",
		AttemptID:  "att-1",
		Actor:      "worker",
	})
	if err != nil {
		t.Fatalf("matching attempt failed: %v", err)
	}

	// Case 6: Attempt with wrong task -> FAIL
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:    "evt-lin-06",
		EventType:  "attempt.wrong_task",
		TaskID:     "task-2",
		ContractID: "contract-1",
		AttemptID:  "att-1",
		Actor:      "worker",
	})
	if err == nil || !errors.Is(err, ErrInvalidAuditLineage) {
		t.Fatalf("expected ErrInvalidAuditLineage on wrong task, got: %v", err)
	}

	// Case 7: Attempt with wrong contract -> FAIL
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:    "evt-lin-07",
		EventType:  "attempt.wrong_contract",
		TaskID:     "task-1",
		ContractID: "contract-2",
		AttemptID:  "att-1",
		Actor:      "worker",
	})
	if err == nil || !errors.Is(err, ErrInvalidAuditLineage) {
		t.Fatalf("expected ErrInvalidAuditLineage on wrong contract, got: %v", err)
	}

	// Case 8: Task with wrong pair -> FAIL
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:   "evt-lin-08",
		EventType: "task.wrong_pair",
		PairID:    "pair-2", // task-1 belongs to pair-1!
		TaskID:    "task-1",
		Actor:     "worker",
	})
	if err == nil || !errors.Is(err, ErrInvalidAuditLineage) {
		t.Fatalf("expected ErrInvalidAuditLineage on wrong pair, got: %v", err)
	}

	// Case 9: Contract without TaskID -> FAIL
	_, err = s.AppendAuditEvent(ctx, domain.AuditEvent{
		EventID:    "evt-lin-09",
		EventType:  "contract.orphan",
		ContractID: "contract-1",
		Actor:      "worker",
	})
	if err == nil || !errors.Is(err, ErrInvalidAuditLineage) {
		t.Fatalf("expected ErrInvalidAuditLineage on orphan contract, got: %v", err)
	}

	// Verify chain is still valid after failed lineage checks
	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("VerifyAuditChain failed: %v", err)
	}

	t.Logf("AUDIT_LINEAGE_VALIDATION = PASS")
}

func TestStore_AuditSecretSanitization_DiskScan(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "audit_disk_scan.db")

	cfg := Config{DBPath: dbPath, BusyTimeoutMs: 5000}
	s, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Synthetic secrets in values that must NEVER appear on disk
	syntheticBearer := "super-secret-synthetic-bearer-token-12345"
	syntheticOpenAI := "sk-proj-testsyntheticopenai1234567890abcdef"
	syntheticGH := "ghp_testsyntheticgithubtoken1234567890"
	syntheticSession := "synthetic-session-secret-token-xyz-999"
	syntheticPassword := "synthetic-master-password-value-456"

	// Synthetic secrets in keys that must NEVER appear on disk
	syntheticKeyBearer := "Authorization: Bearer secret-bearer-in-key-77777"
	syntheticKeyGH := "ghp_secretkeygithubtoken8888888888"
	syntheticKeyOpenAI := "sk-proj-secretkeyopenai9999999999"
	syntheticNestedKey := "github_pat_nestedsecretkey1111111111"

	// Event 1: Secrets in values
	evt1 := domain.AuditEvent{
		EventID:   "evt-disk-sec-01",
		EventType: "pair.bound",
		Actor:     fmt.Sprintf("worker Authorization: Bearer %s", syntheticBearer),
		Details: map[string]any{
			"session_token": syntheticSession,
			"password":      syntheticPassword,
			"gh_token":      syntheticGH,
			"model_key":     syntheticOpenAI,
			"safe_note":     "publicly safe message",
		},
	}

	rec1, err := s.AppendAuditEvent(ctx, evt1)
	if err != nil {
		t.Fatalf("AppendAuditEvent 1 failed: %v", err)
	}
	if strings.Contains(rec1.Event.Actor, syntheticBearer) {
		t.Errorf("returned record actor contains synthetic bearer")
	}
	if rec1.Event.Details["session_token"] != "[REDACTED]" {
		t.Errorf("returned record session_token not redacted: %v", rec1.Event.Details["session_token"])
	}

	// Event 2: Secrets in map keys (Finding R2-002 closeout)
	evt2 := domain.AuditEvent{
		EventID:   "evt-disk-sec-02",
		EventType: "system.config",
		Actor:     "config-loader",
		Details: map[string]any{
			syntheticKeyBearer: "value_for_bearer_key",
			syntheticKeyGH:     "value_for_gh_key",
			"nested": map[string]any{
				syntheticNestedKey: "nested_value",
			},
		},
	}

	rec2, err := s.AppendAuditEvent(ctx, evt2)
	if err != nil {
		t.Fatalf("AppendAuditEvent 2 failed: %v", err)
	}
	if _, exists := rec2.Event.Details[syntheticKeyGH]; exists {
		t.Errorf("returned record contains raw synthetic GH key")
	}

	// Event 3: OpenAI secret in key
	evt3 := domain.AuditEvent{
		EventID:   "evt-disk-sec-03",
		EventType: "model.registered",
		Actor:     "model-admin",
		Details: map[string]any{
			syntheticKeyOpenAI: "openai_val",
		},
	}
	_, err = s.AppendAuditEvent(ctx, evt3)
	if err != nil {
		t.Fatalf("AppendAuditEvent 3 failed: %v", err)
	}

	// Flush and close database to inspect disk files
	_, _ = s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE);")
	s.Close()

	// Read all database files on disk
	filesToCheck := []string{
		dbPath,
		dbPath + "-wal",
		dbPath + "-shm",
	}

	allSecrets := []string{
		syntheticBearer,
		syntheticOpenAI,
		syntheticGH,
		syntheticSession,
		syntheticPassword,
		syntheticKeyBearer,
		syntheticKeyGH,
		syntheticKeyOpenAI,
		syntheticNestedKey,
	}

	for _, fPath := range filesToCheck {
		data, err := os.ReadFile(fPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("failed to read file %s: %v", fPath, err)
		}

		for _, sec := range allSecrets {
			if bytes.Contains(data, []byte(sec)) {
				t.Fatalf("RAW SECRET DISK LEAK: file %s contains raw secret %q", fPath, sec)
			}
		}
	}

	t.Logf("RAW_SECRET_VALUE_DISK_SCAN = PASS")
	t.Logf("RAW_SECRET_KEY_DISK_SCAN = PASS")
	t.Logf("RAW_SECRET_DISK_SCAN = PASS")
}

func TestStore_VerifyAuditChain_ConcurrentAppendSnapshot(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent_verify_snapshot.db")
	cfg := Config{DBPath: dbPath, BusyTimeoutMs: 5000}

	s, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	const numEvents = 50
	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	writerDone := make(chan struct{})

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(writerDone)
		for i := 1; i <= numEvents; i++ {
			evt := domain.AuditEvent{
				EventID:   fmt.Sprintf("evt-snap-writer-%03d", i),
				EventType: "stream.append",
				Actor:     "stream-writer",
				Details: map[string]any{
					"seq": i,
				},
			}
			if _, err := s.AppendAuditEvent(ctx, evt); err != nil {
				errCh <- fmt.Errorf("writer append %d failed: %w", i, err)
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	// Verifier goroutines
	const numVerifiers = 4
	for v := 0; v < numVerifiers; v++ {
		wg.Add(1)
		go func(vID int) {
			defer wg.Done()
			for {
				select {
				case <-writerDone:
					// Final verify after writer is done
					if err := s.VerifyAuditChain(ctx); err != nil {
						errCh <- fmt.Errorf("verifier %d final VerifyAuditChain failed: %w", vID, err)
					}
					return
				case <-ctx.Done():
					return
				default:
					if err := s.VerifyAuditChain(ctx); err != nil {
						errCh <- fmt.Errorf("verifier %d snapshot inconsistency: %w", vID, err)
						return
					}
					time.Sleep(3 * time.Millisecond)
				}
			}
		}(v)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent verify error: %v", err)
	}

	// Final verification on store
	if err := s.VerifyAuditChain(ctx); err != nil {
		t.Fatalf("final VerifyAuditChain failed: %v", err)
	}

	var headSeq int64
	if err := s.db.QueryRowContext(ctx, "SELECT last_sequence FROM audit_chain_state WHERE id = 1").Scan(&headSeq); err != nil {
		t.Fatalf("query headSeq failed: %v", err)
	}
	if headSeq != numEvents {
		t.Fatalf("expected headSeq %d, got %d", numEvents, headSeq)
	}

	t.Logf("AUDIT_VERIFY_CONCURRENT_APPEND = PASS")
	t.Logf("AUDIT_FINAL_CHAIN_VERIFY = PASS")
}

func TestStore_ListAuditEvents_Limits(t *testing.T) {
	ctx := context.Background()
	s, _ := createTestStore(t)
	defer s.Close()

	// limit <= 0 -> error
	_, err := s.ListAuditEvents(ctx, 0, 0)
	if err == nil || !errors.Is(err, ErrInvalidLimit) {
		t.Errorf("expected ErrInvalidLimit for limit=0, got: %v", err)
	}

	_, err = s.ListAuditEvents(ctx, 0, -5)
	if err == nil || !errors.Is(err, ErrInvalidLimit) {
		t.Errorf("expected ErrInvalidLimit for limit=-5, got: %v", err)
	}

	// limit > 500 -> error
	_, err = s.ListAuditEvents(ctx, 0, 501)
	if err == nil || !errors.Is(err, ErrInvalidLimit) {
		t.Errorf("expected ErrInvalidLimit for limit=501, got: %v", err)
	}

	// Non-existent GetAuditEvent
	_, err = s.GetAuditEvent(ctx, "non-existent-event-id")
	if err == nil || !errors.Is(err, ErrAuditEventNotFound) {
		t.Errorf("expected ErrAuditEventNotFound, got: %v", err)
	}
}
