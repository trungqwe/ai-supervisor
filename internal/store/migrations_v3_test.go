package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func openRawMigrationDB(t *testing.T, name string) (*sql.DB, Config) {
	t.Helper()
	cfg := Config{DBPath: filepath.Join(t.TempDir(), name), BusyTimeoutMs: 5000}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate migration config: %v", err)
	}
	db, err := sql.Open("sqlite", cfg.DSN())
	if err != nil {
		t.Fatalf("open migration database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, cfg
}

func installSchemaVersion(t *testing.T, db *sql.DB, version int) {
	t.Helper()
	if version >= 1 {
		if _, err := db.Exec(v1Schema); err != nil {
			t.Fatalf("install v1 schema: %v", err)
		}
	}
	if version >= 2 {
		if _, err := db.Exec(v2Schema); err != nil {
			t.Fatalf("install v2 schema: %v", err)
		}
	}
	if version >= 3 {
		if _, err := db.Exec(v3Schema); err != nil {
			t.Fatalf("install v3 schema: %v", err)
		}
	}
	if _, err := db.Exec("PRAGMA user_version = " + strconv.Itoa(version)); err != nil {
		t.Fatalf("set user_version %d: %v", version, err)
	}
}

func assertV3Schema(t *testing.T, db *sql.DB) {
	t.Helper()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != 3 {
		t.Fatalf("user_version = %d, want 3", version)
	}
	for _, table := range []string{"worker_sessions", "pair_provisioning_operations", "dispatch_operations", "stop_operations"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("table %s missing: count=%d err=%v", table, count, err)
		}
	}
	var indexSQL string
	if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_pair_provisioning_unresolved'").Scan(&indexSQL); err != nil {
		t.Fatalf("partial unique index missing: %v", err)
	}
}

func TestMigrationV3_FreshV1V2AndRepeat(t *testing.T) {
	ctx := context.Background()
	for _, startVersion := range []int{0, 1, 2, 3} {
		t.Run(strconv.Itoa(startVersion)+"_to_v3", func(t *testing.T) {
			db, _ := openRawMigrationDB(t, "migration.db")
			installSchemaVersion(t, db, startVersion)
			if err := migrateWithSchemas(ctx, db, v1Schema, v2Schema, v3Schema); err != nil {
				t.Fatalf("migrate from v%d: %v", startVersion, err)
			}
			assertV3Schema(t, db)
			if err := migrateWithSchemas(ctx, db, v1Schema, v2Schema, v3Schema); err != nil {
				t.Fatalf("repeat migrate from v3: %v", err)
			}
			assertV3Schema(t, db)
		})
	}
}

func TestMigrationV3_PreservesV2DataAndDefaultsSnapshot(t *testing.T) {
	ctx := context.Background()
	db, _ := openRawMigrationDB(t, "preserve.db")
	installSchemaVersion(t, db, 2)
	_, err := db.Exec(`
INSERT INTO projects(project_id, name, root_path, registered_at) VALUES ('project-old', 'Old', '/old', '2026-09-23T00:00:00Z');
INSERT INTO pairs(pair_id, project_id, current_phase_id, state, created_at) VALUES ('pair-old', 'project-old', 'P02', 'ACTIVE', '2026-09-23T00:00:00Z');
INSERT INTO tasks(task_id, phase_id, pair_id, state, current_attempt, created_at, updated_at) VALUES ('task-old', 'P02', 'pair-old', 'DISPATCHED', 1, '2026-09-23T00:00:00Z', '2026-09-23T00:00:00Z');
INSERT INTO task_contracts(contract_id, task_id, revision_number, base_sha, payload_json, is_immutable) VALUES ('contract-old', 'task-old', 1, 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '{}', 1);
INSERT INTO task_attempts(attempt_id, attempt_number, task_id, contract_id, expected_report_path, started_at) VALUES ('attempt-old', 1, 'task-old', 'contract-old', '.supervisor/reports/task-old/attempt-old.json', '2026-09-23T00:00:00Z');
`)
	if err != nil {
		t.Fatalf("seed v2 data: %v", err)
	}
	if err := migrateWithSchemas(ctx, db, v1Schema, v2Schema, v3Schema); err != nil {
		t.Fatalf("migrate preserved v2 data: %v", err)
	}
	var taskID, quarantine string
	var sessionID, generation, disposition sql.NullString
	err = db.QueryRow(`
SELECT task_id, session_id, terminal_generation, recovery_disposition, quarantine_state
FROM task_attempts WHERE attempt_id = 'attempt-old'
`).Scan(&taskID, &sessionID, &generation, &disposition, &quarantine)
	if err != nil {
		t.Fatalf("read migrated attempt: %v", err)
	}
	if taskID != "task-old" || sessionID.Valid || generation.Valid || disposition.Valid || quarantine != "CLEAN" {
		t.Fatalf("migrated attempt changed unexpectedly: task=%q session=%v generation=%v disposition=%v quarantine=%q", taskID, sessionID, generation, disposition, quarantine)
	}
}

func TestMigrationV3_RollsBackBrokenDDL(t *testing.T) {
	ctx := context.Background()
	db, _ := openRawMigrationDB(t, "rollback.db")
	installSchemaVersion(t, db, 2)
	brokenV3 := `
ALTER TABLE task_attempts ADD COLUMN migration_probe TEXT;
CREATE TABLE migration_probe_table (id TEXT PRIMARY KEY);
INVALID SQL;
`
	if err := migrateWithSchemas(ctx, db, v1Schema, v2Schema, brokenV3); err == nil {
		t.Fatal("broken v3 migration unexpectedly succeeded")
	}
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 2 {
		t.Fatalf("user_version after rollback = %d, err=%v; want 2", version, err)
	}
	var tableCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name = 'migration_probe_table'").Scan(&tableCount); err != nil || tableCount != 0 {
		t.Fatalf("partial v3 table survived rollback: count=%d err=%v", tableCount, err)
	}
	rows, err := db.Query("PRAGMA table_info(task_attempts)")
	if err != nil {
		t.Fatalf("inspect task_attempts columns: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan task_attempts column: %v", err)
		}
		if name == "migration_probe" {
			t.Fatal("partial v3 column survived rollback")
		}
	}
}

func TestMigrationV3_ForeignKeysChecksAndDefaults(t *testing.T) {
	db, _ := openRawMigrationDB(t, "constraints.db")
	installSchemaVersion(t, db, 3)
	expected := map[string]map[string]string{
		"worker_sessions": {
			"pair_id": "pairs.pair_id",
		},
		"pair_provisioning_operations": {
			"pair_id": "pairs.pair_id",
		},
		"dispatch_operations": {
			"attempt_id": "task_attempts.attempt_id",
			"pair_id":    "pairs.pair_id",
			"task_id":    "tasks.task_id",
		},
		"stop_operations": {
			"pair_id":     "pairs.pair_id",
			"task_id":     "tasks.task_id",
			"contract_id": "task_contracts.contract_id",
			"attempt_id":  "task_attempts.attempt_id",
		},
	}
	for table, wanted := range expected {
		rows, err := db.Query("PRAGMA foreign_key_list(" + table + ")")
		if err != nil {
			t.Fatalf("foreign_key_list(%s): %v", table, err)
		}
		seen := make(map[string]string)
		for rows.Next() {
			var id, seq int
			var parent, from, to, onUpdate, onDelete, match string
			if err := rows.Scan(&id, &seq, &parent, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				rows.Close()
				t.Fatalf("scan foreign key for %s: %v", table, err)
			}
			if strings.ToUpper(onDelete) != "RESTRICT" {
				rows.Close()
				t.Fatalf("%s.%s on_delete = %q, want RESTRICT", table, from, onDelete)
			}
			seen[from] = parent + "." + to
		}
		rows.Close()
		for from, target := range wanted {
			if seen[from] != target {
				t.Fatalf("%s.%s foreign key = %q, want %q", table, from, seen[from], target)
			}
		}
	}

	var workerSQL, provisioningSQL, attemptSQL, stopSQL string
	for table, destination := range map[string]*string{
		"worker_sessions":              &workerSQL,
		"pair_provisioning_operations": &provisioningSQL,
		"task_attempts":                &attemptSQL,
		"stop_operations":              &stopSQL,
	} {
		if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(destination); err != nil {
			t.Fatalf("read schema for %s: %v", table, err)
		}
	}
	for _, check := range []struct {
		sql    string
		clause string
	}{
		{workerSQL, "quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))"},
		{provisioningSQL, "stage TEXT NOT NULL CHECK (stage IN ('PROVISION_REQUESTED', 'PROVISION_CONFIRMED', 'PROVISION_FAILED', 'PROVISION_RESOLVED'))"},
		{attemptSQL, "quarantine_state TEXT NOT NULL DEFAULT 'CLEAN' CHECK (quarantine_state IN ('CLEAN', 'QUARANTINED'))"},
		{stopSQL, "resolution_state TEXT NOT NULL DEFAULT 'IN_FLIGHT'"},
	} {
		if !strings.Contains(check.sql, check.clause) {
			t.Fatalf("schema missing canonical clause %q: %s", check.clause, check.sql)
		}
	}
}
