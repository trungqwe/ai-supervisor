package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"modernc.org/sqlite"
)

// EffectivePragmas holds the verified runtime pragma values from a live connection.
type EffectivePragmas struct {
	JournalMode string
	Synchronous int
	ForeignKeys bool
	BusyTimeout int
	UserVersion int
}

// Store provides thread-safe local persistence for the Supervisor Control Plane.
type Store struct {
	cfg Config
	db  *sql.DB
}

// Open initializes, connects, verifies pragmas, and migrates the SQLite database.
func Open(ctx context.Context, cfg Config) (*Store, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	connector, err := sqlite.NewConnector(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("store: failed to create sqlite connector: %w", err)
	}

	db := sql.OpenDB(connector)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: failed to ping database: %w", err)
	}

	// Verify effective PRAGMAs on live connection
	pragmas, err := queryEffectivePragmas(ctx, db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("store: failed to query effective pragmas: %w", err)
	}

	if strings.ToLower(pragmas.JournalMode) != "wal" {
		db.Close()
		return nil, fmt.Errorf("store: effective journal_mode is %q, expected %q", pragmas.JournalMode, "wal")
	}

	if pragmas.Synchronous != 2 { // FULL is 2
		db.Close()
		return nil, fmt.Errorf("store: effective synchronous is %d, expected 2 (FULL)", pragmas.Synchronous)
	}

	if !pragmas.ForeignKeys {
		db.Close()
		return nil, fmt.Errorf("store: foreign_keys pragma is OFF, expected ON")
	}

	if pragmas.BusyTimeout != cfg.BusyTimeoutMs {
		db.Close()
		return nil, fmt.Errorf("store: effective busy_timeout is %d, expected %d", pragmas.BusyTimeout, cfg.BusyTimeoutMs)
	}

	// Run migrations
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{
		cfg: cfg,
		db:  db,
	}, nil
}

// EffectivePragmas inspects the database PRAGMAs on a live connection.
func (s *Store) EffectivePragmas(ctx context.Context) (EffectivePragmas, error) {
	return queryEffectivePragmas(ctx, s.db)
}

func queryEffectivePragmas(ctx context.Context, db *sql.DB) (EffectivePragmas, error) {
	var ep EffectivePragmas

	// journal_mode
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&ep.JournalMode); err != nil {
		return ep, fmt.Errorf("failed to read journal_mode: %w", err)
	}

	// synchronous
	if err := db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&ep.Synchronous); err != nil {
		return ep, fmt.Errorf("failed to read synchronous: %w", err)
	}

	// foreign_keys
	var fk int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		return ep, fmt.Errorf("failed to read foreign_keys: %w", err)
	}
	ep.ForeignKeys = (fk == 1)

	// busy_timeout
	if err := db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&ep.BusyTimeout); err != nil {
		return ep, fmt.Errorf("failed to read busy_timeout: %w", err)
	}

	// user_version
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&ep.UserVersion); err != nil {
		return ep, fmt.Errorf("failed to read user_version: %w", err)
	}

	return ep, nil
}

// Close gracefully closes the SQLite database connection pool and releases file locks.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
