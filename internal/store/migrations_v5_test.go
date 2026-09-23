package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestMigrationV5FreshAndHistorical(t *testing.T) {
	for _, from := range []int{0, 1, 2, 3, 4, 5} {
		t.Run(string(rune('0'+from)), func(t *testing.T) {
			db, _ := openRawMigrationDB(t, "v5.db")
			installSchemaVersion(t, db, min(from, 3))
			if from >= 4 {
				if _, err := db.Exec(v4Schema); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("PRAGMA user_version=4"); err != nil {
					t.Fatal(err)
				}
			}
			if from == 5 {
				if _, err := db.Exec(v5Schema); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("PRAGMA user_version=5"); err != nil {
					t.Fatal(err)
				}
			}
			if err := migrate(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			assertV5(t, db)
			if err := migrate(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			assertV5(t, db)
		})
	}
}

func assertV5(t *testing.T, db *sql.DB) {
	t.Helper()
	var version, count int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 5 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='attempt_execution_budgets'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("budget table=%d err=%v", count, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM pragma_table_info('stop_operations') WHERE name='initiating_failure_reason'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("stop cause=%d err=%v", count, err)
	}
	for _, name := range []string{"idx_one_live_stop_per_attempt", "trg_execution_budget_immutable_update", "trg_execution_budget_immutable_delete", "trg_dispatch_requires_execution_budget"} {
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name=?", name).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s=%d err=%v", name, count, err)
		}
	}
}

func TestMigrationV5RollbackOnBrokenDDL(t *testing.T) {
	db, _ := openRawMigrationDB(t, "v5-rollback.db")
	installSchemaVersion(t, db, 3)
	if _, err := db.Exec(v4Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version=4"); err != nil {
		t.Fatal(err)
	}
	if err := migrateWithSchemas(context.Background(), db, v1Schema, v2Schema, v3Schema, v4Schema, "CREATE TABLE v5_partial(id TEXT); INVALID SQL;"); err == nil {
		t.Fatal("broken DDL succeeded")
	}
	var version, count int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("rollback version=%d err=%v", version, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='v5_partial'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial table=%d err=%v", count, err)
	}
}
