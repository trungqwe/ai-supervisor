package store

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultBusyTimeoutMs = 5000
	MinBusyTimeoutMs     = 1
	MaxBusyTimeoutMs     = 60000
)

// Config holds local SQLite StateStore configuration.
type Config struct {
	DBPath        string
	BusyTimeoutMs int
}

// Validate checks that the configuration conforms to security and durability policies.
func (c *Config) Validate() error {
	if c.DBPath == "" {
		return errors.New("store: DBPath must not be empty")
	}

	absPath, err := filepath.Abs(filepath.Clean(c.DBPath))
	if err != nil {
		return fmt.Errorf("store: invalid DBPath: %w", err)
	}
	c.DBPath = absPath

	parent := filepath.Dir(absPath)
	info, err := os.Stat(parent)
	if err != nil {
		return fmt.Errorf("store: parent directory does not exist for DBPath: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("store: parent path %q is not a directory", parent)
	}

	if c.BusyTimeoutMs == 0 {
		c.BusyTimeoutMs = DefaultBusyTimeoutMs
	} else if c.BusyTimeoutMs < MinBusyTimeoutMs || c.BusyTimeoutMs > MaxBusyTimeoutMs {
		return fmt.Errorf("store: BusyTimeoutMs must be between %d and %d ms (got %d)",
			MinBusyTimeoutMs, MaxBusyTimeoutMs, c.BusyTimeoutMs)
	}

	return nil
}

// DSN builds the SQLite connection URI with mandatory durability and isolation PRAGMAs.
func (c *Config) DSN() string {
	q := url.Values{}
	q.Set("_journal_mode", "WAL")
	q.Set("_synchronous", "FULL")
	q.Set("_foreign_keys", "ON")
	q.Set("_busy_timeout", strconv.Itoa(c.BusyTimeoutMs))
	q.Set("_txlock", "immediate")

	cleanSlash := filepath.ToSlash(c.DBPath)
	return fmt.Sprintf("file:%s?%s", cleanSlash, q.Encode())
}
