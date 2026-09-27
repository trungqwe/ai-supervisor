package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestMigrationV6FreshAndHistorical(t *testing.T) {
	for _, from := range []int{0, 1, 2, 3, 4, 5, 6} {
		t.Run(string(rune('0'+from)), func(t *testing.T) {
			db, _ := openRawMigrationDB(t, "v6.db")
			installSchemaVersion(t, db, min(from, 3))
			if from >= 4 {
				if _, err := db.Exec(v4Schema); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("PRAGMA user_version=4"); err != nil {
					t.Fatal(err)
				}
			}
			if from >= 5 {
				if _, err := db.Exec(v5Schema); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("PRAGMA user_version=5"); err != nil {
					t.Fatal(err)
				}
			}
			if from == 6 {
				if _, err := db.Exec(v6Schema); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec("PRAGMA user_version=6"); err != nil {
					t.Fatal(err)
				}
			}
			if err := migrate(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			assertV6(t, db)
			// Idempotency check
			if err := migrate(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			assertV6(t, db)
		})
	}
}

func assertV6(t *testing.T, db *sql.DB) {
	t.Helper()
	var version, count int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 6 {
		t.Fatalf("version=%d err=%v", version, err)
	}

	tables := []string{
		"attempt_workspace_bindings",
		"worker_claims",
		"review_integrity_holds",
	}
	for _, tbl := range tables {
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&count); err != nil || count != 1 {
			t.Fatalf("table %s count=%d err=%v", tbl, count, err)
		}
	}

	indexes := []string{
		"idx_review_integrity_holds_active_dedup",
	}
	for _, idx := range indexes {
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&count); err != nil || count != 1 {
			t.Fatalf("index %s count=%d err=%v", idx, count, err)
		}
	}

	triggers := []string{
		"trg_attempt_workspace_bindings_lineage_guard",
		"trg_attempt_workspace_bindings_cas_guard",
		"trg_attempt_workspace_bindings_no_delete",
		"trg_worker_claims_lineage_guard",
		"trg_worker_claims_no_update",
		"trg_worker_claims_no_delete",
		"trg_review_integrity_holds_lineage_guard",
		"trg_review_integrity_holds_cas_guard",
		"trg_review_integrity_holds_no_delete",
	}
	for _, trg := range triggers {
		if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name=?", trg).Scan(&count); err != nil || count != 1 {
			t.Fatalf("trigger %s count=%d err=%v", trg, count, err)
		}
	}
}

func TestMigrationV6RollbackOnBrokenDDL(t *testing.T) {
	db, _ := openRawMigrationDB(t, "v6-rollback.db")
	installSchemaVersion(t, db, 3)
	if _, err := db.Exec(v4Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version=4"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(v5Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version=5"); err != nil {
		t.Fatal(err)
	}

	brokenV6 := "CREATE TABLE v6_partial(id TEXT); INVALID SQL STATEMENT;"
	if err := migrateWithSchemas(context.Background(), db, v1Schema, v2Schema, v3Schema, v4Schema, v5Schema, brokenV6); err == nil {
		t.Fatal("broken v6 DDL succeeded when it should fail")
	}

	var version, count int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 5 {
		t.Fatalf("rollback version=%d err=%v (expected 5)", version, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='v6_partial'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial table count=%d err=%v (expected 0)", count, err)
	}
}
