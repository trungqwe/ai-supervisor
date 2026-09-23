package store

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func assertV4(t *testing.T, db *sql.DB) {
	t.Helper()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != CurrentSchemaVersion {
		t.Fatalf("user_version=%d err=%v, want current %d", version, err, CurrentSchemaVersion)
	}
	for _, table := range []string{"restore_authorizations", "pair_restore_operations"} {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("table %s count=%d err=%v", table, count, err)
		}
	}
	var columnCount int
	if err := db.QueryRow("SELECT count(*) FROM pragma_table_info('stop_operations') WHERE name='restore_operation_id'").Scan(&columnCount); err != nil || columnCount != 1 {
		t.Fatalf("restore_operation_id count=%d err=%v", columnCount, err)
	}
	for _, index := range []string{"idx_pair_restore_unresolved", "idx_restore_stage_resolution", "idx_stop_restore_operation"} {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&count); err != nil || count != 1 {
			t.Fatalf("index %s count=%d err=%v", index, count, err)
		}
	}
	for _, trigger := range []string{"trg_restore_auth_no_delete", "trg_restore_auth_immutable", "trg_restore_op_no_delete", "trg_restore_op_forward_only"} {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name=?", trigger).Scan(&count); err != nil || count != 1 {
			t.Fatalf("trigger %s count=%d err=%v", trigger, count, err)
		}
	}
	var authDDL, operationDDL string
	if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='restore_authorizations'").Scan(&authDDL); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='pair_restore_operations'").Scan(&operationDDL); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(authDDL, "consumed_by_operation_id IS NOT NULL") || !strings.Contains(operationDDL, "RESTORE_CLEANUP_CLAIMED") {
		t.Fatalf("v4 authorization or restore transition checks missing: auth=%s operation=%s", authDDL, operationDDL)
	}
}

func TestMigrationV4_FreshV1V2V3Repeat(t *testing.T) {
	ctx := context.Background()
	for _, from := range []int{0, 1, 2, 3, 4} {
		t.Run(string(rune('0'+from)), func(t *testing.T) {
			db, _ := openRawMigrationDB(t, "v4.db")
			installSchemaVersion(t, db, min(from, 3))
			if from == 4 {
				if _, err := db.Exec(v4Schema); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("PRAGMA user_version=4"); err != nil {
					t.Fatal(err)
				}
			}
			if err := migrate(ctx, db); err != nil {
				t.Fatalf("migrate from %d: %v", from, err)
			}
			assertV4(t, db)
			if err := migrate(ctx, db); err != nil {
				t.Fatalf("repeat v4 migration: %v", err)
			}
			assertV4(t, db)
		})
	}
}

func TestMigrationV4_PreservesV3RowsAndRollsBackFailure(t *testing.T) {
	ctx := context.Background()
	db, _ := openRawMigrationDB(t, "v4-preserve.db")
	installSchemaVersion(t, db, 3)
	_, err := db.Exec(`
INSERT INTO projects(project_id,name,root_path,registered_at) VALUES('p','p','/p','2026-01-01T00:00:00Z');
INSERT INTO pairs(pair_id,project_id,current_phase_id,state,created_at) VALUES('pair','p','P03','ACTIVE','2026-01-01T00:00:00Z');
INSERT INTO tasks(task_id,phase_id,pair_id,state,current_attempt,created_at,updated_at) VALUES('task','P03','pair','FAILED',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z');
INSERT INTO task_contracts(contract_id,task_id,revision_number,base_sha,payload_json,is_immutable) VALUES('contract','task',1,'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','{}',1);
INSERT INTO task_attempts(attempt_id,attempt_number,task_id,contract_id,expected_report_path,started_at,ended_at,session_id,terminal_generation,quarantine_state) VALUES('attempt',1,'task','contract','.supervisor/reports/task/attempt.json','2026-01-01T00:00:00Z','2026-01-01T00:01:00Z','old-session','g-old','QUARANTINED');
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var taskID, sessionID, generation, quarantine string
	if err := db.QueryRow("SELECT task_id,session_id,terminal_generation,quarantine_state FROM task_attempts WHERE attempt_id='attempt'").Scan(&taskID, &sessionID, &generation, &quarantine); err != nil {
		t.Fatal(err)
	}
	if taskID != "task" || sessionID != "old-session" || generation != "g-old" || quarantine != "QUARANTINED" {
		t.Fatalf("v3 lineage changed: %q %q %q %q", taskID, sessionID, generation, quarantine)
	}

	db2, _ := openRawMigrationDB(t, "v4-rollback.db")
	installSchemaVersion(t, db2, 3)
	broken := "CREATE TABLE v4_partial(id TEXT); INVALID SQL;"
	if err := migrateWithSchemas(ctx, db2, v1Schema, v2Schema, v3Schema, broken); err == nil {
		t.Fatal("broken v4 unexpectedly migrated")
	}
	var version, tables int
	if err := db2.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 3 {
		t.Fatalf("rollback version=%d err=%v, want 3", version, err)
	}
	if err := db2.QueryRow("SELECT count(*) FROM sqlite_master WHERE name IN ('v4_partial','restore_authorizations')").Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("partial v4 DDL remains: count=%d err=%v", tables, err)
	}
}
