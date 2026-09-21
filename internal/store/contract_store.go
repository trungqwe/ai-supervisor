package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"modernc.org/sqlite"
)

// InsertTaskContract persists a new TaskContract and its canonical serialized JSON payload.
func (s *Store) InsertTaskContract(ctx context.Context, c domain.TaskContract) error {
	if strings.TrimSpace(c.ContractID) == "" {
		return errors.New("store: contract_id must not be empty")
	}
	if strings.TrimSpace(c.TaskID) == "" {
		return errors.New("store: task_id must not be empty")
	}
	if c.RevisionNumber < 1 {
		return fmt.Errorf("store: revision_number must be >= 1 (got %d)", c.RevisionNumber)
	}
	if strings.TrimSpace(c.BaseSHA) == "" {
		return errors.New("store: base_sha must not be empty")
	}

	payloadBytes, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("store: failed to serialize task contract: %w", err)
	}

	isImmutableInt := 0
	if c.IsImmutable {
		isImmutableInt = 1
	}

	query := `
INSERT INTO task_contracts (
    contract_id, task_id, revision_number, supersedes_contract_id,
    base_sha, payload_json, is_immutable
) VALUES (?, ?, ?, ?, ?, ?, ?)
`

	_, err = s.db.ExecContext(ctx, query,
		c.ContractID,
		c.TaskID,
		c.RevisionNumber,
		c.SupersedesContractID,
		c.BaseSHA,
		string(payloadBytes),
		isImmutableInt,
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			if strings.Contains(sqliteErr.Error(), "FOREIGN KEY constraint failed") {
				return fmt.Errorf("%w: foreign key violation inserting contract", ErrForeignKeyViolation)
			}
			if strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("%w: duplicate contract or revision conflict", ErrDuplicateKey)
			}
		}
		return fmt.Errorf("store: failed to insert task contract: %w", err)
	}

	return nil
}

// GetTaskContract retrieves a TaskContract, unmarshaling the JSON payload with UseNumber() to preserve large integer precision.
func (s *Store) GetTaskContract(ctx context.Context, contractID string) (domain.TaskContract, error) {
	query := `
SELECT payload_json, is_immutable
FROM task_contracts
WHERE contract_id = ?
`

	var payloadJSON string
	var isImmutableInt int

	err := s.db.QueryRowContext(ctx, query, contractID).Scan(&payloadJSON, &isImmutableInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskContract{}, ErrContractNotFound
		}
		return domain.TaskContract{}, fmt.Errorf("store: failed to query task contract: %w", err)
	}

	var c domain.TaskContract
	dec := json.NewDecoder(bytes.NewReader([]byte(payloadJSON)))
	dec.UseNumber()
	if err := dec.Decode(&c); err != nil {
		return domain.TaskContract{}, fmt.Errorf("store: failed to decode contract payload: %w", err)
	}

	c.IsImmutable = (isImmutableInt == 1)
	return c, nil
}
